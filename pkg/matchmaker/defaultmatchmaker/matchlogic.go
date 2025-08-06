// Copyright (c) 2022-2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package defaultmatchmaker provides the default implementation of the MatchLogic interface.
// This package contains the core matchmaking algorithms and logic for creating matches from tickets.
package defaultmatchmaker

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/common"
	"github.com/AccelByte/extend-core-matchmaker/pkg/config"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker"
	player "github.com/AccelByte/extend-core-matchmaker/pkg/playerdata"

	pie "github.com/elliotchance/pie/v2"
)

// externalPartyID is used to identify parties that are not part of the internal system
const externalPartyID = "external"

// defaultMatchMaker implements the MatchLogic interface with the default matchmaking algorithm.
// It handles ticket validation, match creation, and backfill operations.
type defaultMatchMaker struct {
	unmatchedTickets    []matchmaker.Ticket   // Tickets that haven't been matched yet
	mm                  matchmaker.Matchmaker // The underlying matchmaker implementation
	indexedTicketLength int                   // Size of ticket chunks for processing
}

// New returns a defaultMatchMaker of the MatchLogic interface.
// This is the main constructor for creating a new default matchmaker instance.
func New(cfg *config.Config) matchmaker.MatchLogic {
	return defaultMatchMaker{
		indexedTicketLength: cfg.TicketChunkSize,
		mm:                  NewMatchMaker(cfg),
	}
}

// ValidateTicket returns a bool if the match ticket is valid.
// This method checks if a ticket meets all requirements to be queued for matchmaking.
func (b defaultMatchMaker) ValidateTicket(scope *envelope.Scope, matchTicket matchmaker.Ticket, matchRules interface{}) (bool, error) {
	scope.Log.Info("MATCHMAKER: validate ticket")
	scope.Log.Info("Ticket Validation successful")

	// Type assertion to get the ruleset
	ruleSet, ok := matchRules.(models.RuleSet)
	if !ok {
		scope.Log.WithField("RuleSet", fmt.Sprintf("%T", matchRules)).Error("invalid RuleSet type")
		return false, errors.New("invalid ruleset")
	}

	// Check if the ticket has valid latency to at least one region
	hasValidLatency := false
	for _, latency := range matchTicket.Latencies {
		if latency <= int64(ruleSet.RegionLatencyMaxMs) {
			hasValidLatency = true
			break
		}
	}

	if !hasValidLatency {
		return false, errors.New("no region latency below max")
	}

	return true, nil
}

// EnrichTicket is responsible for adding logic to the match ticket before match making.
// This method can modify or add data to tickets before they enter the matchmaking process.
func (b defaultMatchMaker) EnrichTicket(scope *envelope.Scope, matchTicket matchmaker.Ticket, ruleSet interface{}) (ticket matchmaker.Ticket, err error) {
	scope.Log.Info("MATCHMAKER: enrich ticket")

	return matchTicket, nil
}

// GetStatCodes returns the string slice of the stat codes in matchrules.
// This method provides statistics codes for monitoring and metrics collection.
func (b defaultMatchMaker) GetStatCodes(scope *envelope.Scope, matchRules interface{}) []string {
	scope.Log.Infof("MATCHMAKER: stat codes: %s", []string{})

	return []string{}
}

// RulesFromJSON returns the ruleset from the Game rules.
// This method parses JSON configuration into a structured ruleset that the matchmaker can use.
func (b defaultMatchMaker) RulesFromJSON(rootScope *envelope.Scope, jsonRules string) (interface{}, error) {
	scope := rootScope.NewChildScope("defaultMatchMaker.RulesFromJson")
	defer scope.Finish()

	var ruleSet models.RuleSet
	err := json.Unmarshal([]byte(jsonRules), &ruleSet)
	if err != nil {
		return nil, err
	}

	err = ruleSet.Validate()
	if err != nil {
		return nil, err
	}

	ruleSet.SetDefaultValues()

	// since we cannot store match attempt set default region expansion rate
	if ruleSet.RegionExpansionRateMs == 0 {
		ruleSet.RegionExpansionRateMs = 5000
	}

	return ruleSet, nil
}

func (b defaultMatchMaker) MakeMatches(rootScope *envelope.Scope, ticketProvider matchmaker.TicketProvider, matchRules interface{}) <-chan matchmaker.Match {
	scope := rootScope.NewChildScope("defaultMatchMaker.BackfillMatches")
	defer scope.Finish()

	results := make(chan matchmaker.Match)
	ruleset, ok := matchRules.(models.RuleSet)
	if !ok {
		scope.Log.Errorf("invalid type for matchRules")
		close(results)
		return results
	}

	go func() {
		var wg sync.WaitGroup
		channel := models.Channel{
			Ruleset: ruleset,
		}

		ticketChannel := ticketProvider.GetTickets()
		for requests, tickets := getNextNRequests(ticketChannel, b.indexedTicketLength, ruleset); len(tickets) > 0; requests, tickets = getNextNRequests(ticketChannel, b.indexedTicketLength, ruleset) {
			wg.Add(1)

			requestValues := requests
			sourceTickets := tickets

			b.addToPartiesRegionInMatchQueueMetrics(scope, requestValues, sourceTickets, channel)

			go b.runMatchMaking(scope, requestValues, results, &wg, channel, channel.Ruleset, sourceTickets)
		}

		wg.Wait()
		close(results)
	}()

	return results
}

func (b defaultMatchMaker) BackfillMatches(rootScope *envelope.Scope, ticketProvider matchmaker.TicketProvider, matchRules interface{}) <-chan matchmaker.BackfillProposal {
	scope := rootScope.NewChildScope("defaultMatchMaker.BackfillMatches")
	defer scope.Finish()

	results := make(chan matchmaker.BackfillProposal)
	ruleset, ok := matchRules.(models.RuleSet)
	if !ok {
		scope.Log.Errorf("invalid type for matchRules")
		close(results)
		return results
	}

	go func() {
		var wg sync.WaitGroup
		channel := models.Channel{
			Ruleset: ruleset,
		}

		backfillTicketChannel := ticketProvider.GetBackfillTickets()
		ticketChannel := ticketProvider.GetTickets()
		for requests, sessions, tickets := getNextNBackfillRequests(scope, backfillTicketChannel, ticketChannel, b.indexedTicketLength, ruleset); len(sessions) > 0; requests, sessions, tickets = getNextNBackfillRequests(scope, backfillTicketChannel, ticketChannel, b.indexedTicketLength, ruleset) {
			wg.Add(1)

			requestValues := requests
			sessionValues := sessions
			go b.runBackfilling(scope, tickets, requestValues, sessionValues, channel, results, &wg)
		}
		wg.Wait()
		close(results)
	}()
	return results
}

func (b defaultMatchMaker) runBackfilling(
	rootScope *envelope.Scope, tickets []matchmaker.Ticket, requests []models.MatchmakingRequest,
	sessions []*models.MatchmakingResult, channel models.Channel, results chan matchmaker.BackfillProposal,
	wg *sync.WaitGroup,
) {
	scope := rootScope.NewChildScope("runBackfilling")
	defer scope.Finish()
	defer wg.Done()
	namespace, matchPool := getNamespaceMatchPool(tickets)

	updatedSessions, satisfiedSessions, satisfiedTickets, err := b.mm.MatchSessions(scope, namespace, matchPool, requests, sessions, channel)
	if err != nil {
		scope.Log.Errorf("error backfilling matches: %s", err)
	}

	for _, result := range updatedSessions {
		results <- fromMatchResultToBackfillProposal(result, satisfiedTickets, tickets)
	}
	for _, result := range satisfiedSessions {
		results <- fromMatchResultToBackfillProposal(result, satisfiedTickets, tickets)
	}
}

func getNextNRequests(ticketChannel chan matchmaker.Ticket, maxTicketCount int, ruleset models.RuleSet) ([]models.MatchmakingRequest, []matchmaker.Ticket) {
	var indexedTickets []matchmaker.Ticket
	var requests []models.MatchmakingRequest

	for i := 0; i < maxTicketCount; i++ {
		ticket, ok := <-ticketChannel
		if !ok {
			break
		}
		indexedTickets = append(indexedTickets, ticket)
	}
	requests = pie.Map(indexedTickets, toMatchRequest(ruleset))

	return requests, indexedTickets
}

func getNextNBackfillRequests(rootScope *envelope.Scope, backfillTicketChannel chan matchmaker.BackfillTicket,
	ticketChannel chan matchmaker.Ticket, maxTicketCount int, ruleSet models.RuleSet) ([]models.MatchmakingRequest, []*models.MatchmakingResult, []matchmaker.Ticket) {

	scope := rootScope.NewChildScope("getNextNBackfillRequests")
	defer scope.Finish()

	var indexedTickets []matchmaker.Ticket
	var requests []models.MatchmakingRequest
	var indexedBackfillTickets []matchmaker.BackfillTicket
	var sessions []*models.MatchmakingResult
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		for i := 0; i < maxTicketCount; i++ {
			ticket, ok := <-ticketChannel
			if !ok {
				break
			}

			request := toMatchRequest(ruleSet)(ticket)
			if request.IsNewSessionOnly() {
				continue
			}

			indexedTickets = append(indexedTickets, ticket)
			requests = append(requests, request)
		}
		wg.Done()
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		for i := 0; i < maxTicketCount; i++ {
			backfillTicket, ok := <-backfillTicketChannel
			if !ok {
				break
			}
			indexedBackfillTickets = append(indexedBackfillTickets, backfillTicket)
		}
		sessions = pie.Map(indexedBackfillTickets, fromBackfillTicketsToMatchResult(scope, ruleSet))
		wg.Done()
	}(&wg)
	wg.Wait()

	return requests, sessions, indexedTickets
}

func toMatchRequest(ruleset models.RuleSet) func(ticket matchmaker.Ticket) models.MatchmakingRequest {
	return func(ticket matchmaker.Ticket) models.MatchmakingRequest {
		if ticket.TicketAttributes == nil {
			ticket.TicketAttributes = make(map[string]interface{})
		}

		// fit player's stat data into v1 ticket attribute format
		memberAttributes := avergaeMatchingRuleAttributes(ticket.Players, ruleset)

		ticket.TicketAttributes[models.AttributeMemberAttr] = memberAttributes

		var sortedLatency []models.Region
		for region, latency := range ticket.Latencies {
			sortedLatency = append(sortedLatency, models.Region{Region: region, Latency: int(latency)})
		}
		sort.SliceStable(sortedLatency, func(i, j int) bool {
			return sortedLatency[i].Latency < sortedLatency[j].Latency
		})

		// convert latency to int because matchmaker.Ticket is int64
		latencyMap := map[string]int{}
		for k, v := range ticket.Latencies {
			latencyMap[k] = int(v)
		}

		createdAt := ticket.CreatedAt.UTC().Unix()
		request := models.MatchmakingRequest{
			Namespace:           ticket.Namespace,
			PartyID:             ticket.TicketID,
			Channel:             ticket.MatchPool,
			CreatedAt:           createdAt,
			PartyMembers:        pie.Map(ticket.Players, playerDataToPartyMember),
			PartyAttributes:     ticket.TicketAttributes,
			LatencyMap:          latencyMap,
			SortedLatency:       sortedLatency,
			AdditionalCriterias: map[string]interface{}{},
			ExcludedSessions:    ticket.ExcludedSessions,
		}

		return request
	}
}

func avergaeMatchingRuleAttributes(players []player.PlayerData, ruleset models.RuleSet) map[string]interface{} {
	memberAttributes := make(map[string]interface{})

	for _, rule := range ruleset.MatchingRule {
		var totalAttr float64
		for _, playerData := range players {
			if playerData.Attributes == nil {
				continue
			}
			if attr, ok := playerData.Attributes[rule.Attribute]; ok {
				switch attr.(type) {
				case float64:
					totalAttr += attr.(float64)
				case int32:
					totalAttr += float64(attr.(int32))
				case int64:
					totalAttr += float64(attr.(int64))
				case int:
					totalAttr += float64(attr.(int))
				}
			}
		}
		if len(players) > 0 {
			memberAttributes[rule.Attribute] = totalAttr / float64(len(players))
		} else {
			memberAttributes[rule.Attribute] = float64(0)
		}
	}
	return memberAttributes
}

func fromMatchResult(result *models.MatchmakingResult, sourceTickets []matchmaker.Ticket, ruleset models.RuleSet) matchmaker.Match {
	// generate ticket array from teams
	var matchingTickets []matchmaker.Ticket
	for _, ally := range result.MatchingAllies {
		for _, party := range ally.MatchingParties {
			ticketIndex := pie.FindFirstUsing(sourceTickets, func(t matchmaker.Ticket) bool { return t.TicketID == party.PartyID })
			if ticketIndex != -1 {
				matchingTickets = append(matchingTickets, sourceTickets[ticketIndex])
			}
		}
	}

	// handle backfill flag
	backfill := ruleset.AutoBackfill
	if backfill {
		backfill = !isMatchFull(matchingTickets, result, ruleset)
	}

	// fill v1-specific fields back into match attributes
	if result.ServerName != "" {
		result.PartyAttributes[models.AttributeServerName] = result.ServerName
	}
	if result.ClientVersion != "" {
		result.PartyAttributes[models.AttributeClientVersion] = result.ClientVersion
	}

	return matchmaker.Match{
		Tickets:          matchingTickets,
		Teams:            toTeams(sourceTickets, result.MatchingAllies),
		RegionPreference: []string{result.Region},
		MatchAttributes:  result.PartyAttributes,
		Backfill:         backfill,
		ServerName:       result.ServerName,
		ClientVersion:    result.ClientVersion,
		Timestamp:        result.UpdatedAt,
		PivotID:          result.PivotID,
	}
}

func isMatchFull(tickets []matchmaker.Ticket, result *models.MatchmakingResult, ruleset models.RuleSet) bool {
	// find the oldest ticket age to figure out if there's any flexing
	oldestTicket := pie.SortUsing(tickets, func(t1, t2 matchmaker.Ticket) bool {
		return t1.CreatedAt.Before(t2.CreatedAt)
	})[0]

	var maxPlayerCount int
	{
		currentRule, _ := applyAllianceFlexingRules(ruleset, oldestTicket.CreatedAt)
		maxPlayerCount = currentRule.AllianceRule.MaxNumber * currentRule.AllianceRule.PlayerMaxNumber
	}

	// check if match is full
	var matchPlayerCount int
	pie.Each(tickets, func(t matchmaker.Ticket) {
		matchPlayerCount += len(t.Players)
	})

	return maxPlayerCount <= matchPlayerCount
}

func toTeams(tickets []matchmaker.Ticket, alliances []models.MatchingAlly) []matchmaker.Team {
	teams := make([]matchmaker.Team, 0, len(alliances))
	ticketsMap := make(map[string]int)
	for i, ticket := range tickets {
		if _, ok := ticketsMap[ticket.TicketID]; !ok {
			ticketsMap[ticket.TicketID] = i
		}
	}
	for _, alliance := range alliances {
		parties := alliance.MatchingParties
		if len(parties) == 0 {
			continue
		}
		var userIDs []player.ID
		var partyMembers []matchmaker.Party
		for _, party := range parties {
			userIDs = append(userIDs, pie.Map(party.PartyMembers, partyMemberToUserID)...)

			// get party ID from tickets
			var partyID string
			if party.PartyID == externalPartyID {
				partyID = externalPartyID
			}

			if partyID == "" {
				if ticketIndex, ok := ticketsMap[party.PartyID]; ok {
					partyID = tickets[ticketIndex].PartySessionID
				}
			}

			partyMembers = append(partyMembers, matchmaker.Party{
				PartyID: partyID,
				UserIDs: party.GetPartyUserIDs(),
			})
		}
		teamID := alliance.TeamID
		if teamID == "" {
			teamID = common.GenerateUUID()
		}
		teams = append(teams, matchmaker.Team{
			TeamID:  teamID,
			UserIDs: userIDs,
			Parties: partyMembers,
		})
	}

	return teams
}

func partyMemberToUserID(m models.PartyMember) player.ID {
	return player.ID(m.UserID)
}

func playerDataToPartyMember(playerData player.PlayerData) models.PartyMember {
	partyMember := models.PartyMember{
		UserID:          string(playerData.PlayerID),
		ExtraAttributes: playerData.Attributes,
	}
	return partyMember
}

func fromBackfillTicketsToMatchResult(rootScope *envelope.Scope, rules models.RuleSet) func(backfillTicket matchmaker.BackfillTicket) *models.MatchmakingResult {
	return func(backfillTicket matchmaker.BackfillTicket) *models.MatchmakingResult {
		scope := rootScope.NewChildScope("defaultmatchfunction.fromBackfillTicketsToMatchResult")
		defer scope.Finish()

		var teamMembers []string
		for _, team := range backfillTicket.PartialMatch.Teams {
			teamMembers = append(teamMembers, pie.Map(team.UserIDs, player.IDToString)...)
		}
		scope.SetAttributes(envelope.TeamMembersTag, teamMembers)

		var region string
		if len(backfillTicket.PartialMatch.RegionPreference) > 0 {
			region = backfillTicket.PartialMatch.RegionPreference[0]
		}
		partyAttributes := make(map[string]interface{})
		if backfillTicket.PartialMatch.MatchAttributes != nil {
			partyAttributes = backfillTicket.PartialMatch.MatchAttributes
		}
		var serverName string
		if partyAttributes[models.AttributeServerName] != nil {
			scope.SetAttributes(envelope.ServerNameTag, serverName)
			serverName = partyAttributes[models.AttributeServerName].(string)
		}
		var clientVersion string
		if partyAttributes[models.AttributeClientVersion] != nil {
			clientVersion = partyAttributes[models.AttributeClientVersion].(string)
		}

		var playerData []player.PlayerData

		for _, ticket := range backfillTicket.PartialMatch.Tickets {
			playerData = append(playerData, ticket.Players...)
		}

		partyAttributes[models.AttributeMemberAttr] = avergaeMatchingRuleAttributes(playerData, rules)

		return &models.MatchmakingResult{
			MatchID:         backfillTicket.TicketID,
			MatchSessionID:  backfillTicket.MatchSessionID,
			Channel:         backfillTicket.MatchPool,
			Namespace:       "",
			GameMode:        "",
			ServerName:      serverName,
			ClientVersion:   clientVersion,
			Region:          region,
			Joinable:        true,
			MatchingAllies:  pie.Map(backfillTicket.PartialMatch.Teams, fromTeamToMatchingAllies(backfillTicket.PartialMatch.Tickets)),
			Deployment:      "",
			UpdatedAt:       time.Time{},
			QueuedAt:        backfillTicket.CreatedAt.Unix(),
			PartyAttributes: partyAttributes,
		}
	}
}

func fromTeamToMatchingAllies(sourceTickets []matchmaker.Ticket) func(team matchmaker.Team) models.MatchingAlly {
	return func(team matchmaker.Team) models.MatchingAlly {
		foundParties := teamToMatchingParties(team, sourceTickets)

		return models.MatchingAlly{
			TeamID:          team.TeamID,
			MatchingParties: pie.Values(foundParties),
			PlayerCount:     0,
		}
	}
}

func teamToMatchingParties(team matchmaker.Team, sourceTickets []matchmaker.Ticket) map[string]models.MatchingParty {
	foundParties := make(map[string]models.MatchingParty)
	for _, id := range team.UserIDs {
		ticketIndex := pie.FindFirstUsing(sourceTickets,
			func(t matchmaker.Ticket) bool {
				index := pie.FindFirstUsing(t.Players, func(p player.PlayerData) bool { return p.PlayerID == id })
				return index >= 0
			})
		if ticketIndex < 0 {
			externalParty, found := foundParties[externalPartyID]
			if !found {
				externalParty = models.MatchingParty{
					PartyID:         externalPartyID,
					PartyMembers:    []models.PartyMember{},
					MatchAttributes: models.MatchAttributes{},
				}
			}
			partyMember := models.PartyMember{
				UserID: player.IDToString(id),
			}
			externalParty.PartyMembers = append(externalParty.PartyMembers, partyMember)
			foundParties[externalPartyID] = externalParty
			continue
		}

		ticket := sourceTickets[ticketIndex]
		if _, found := foundParties[ticket.TicketID]; found {
			continue
		}

		partyID := ticket.TicketID
		// this from manual backfill
		if strings.Contains(partyID, "-") {
			partyID = externalPartyID
		}

		party := models.MatchingParty{
			PartyID:         partyID,
			PartyAttributes: ticket.TicketAttributes,
			PartyMembers:    pie.Map(ticket.Players, playerDataToPartyMember),
			MatchAttributes: models.MatchAttributes{},
		}
		foundParties[ticket.TicketID] = party
	}
	return foundParties
}

func fromMatchResultToBackfillProposal(result *models.MatchmakingResult, satisfiedRequests []models.MatchmakingRequest, sourceTickets []matchmaker.Ticket) matchmaker.BackfillProposal {
	var addedTickets []matchmaker.Ticket
	for _, ally := range result.MatchingAllies {
		for _, party := range ally.MatchingParties {
			// check if party's ticket is in satisfied requests (tickets)
			index := pie.FindFirstUsing(satisfiedRequests, func(r models.MatchmakingRequest) bool { return r.PartyID == party.PartyID })
			if index < 0 {
				continue
			}
			// store the ticket to added tickets
			index = pie.FindFirstUsing(sourceTickets, func(t matchmaker.Ticket) bool { return t.TicketID == party.PartyID })
			if index < 0 {
				continue
			}
			addedTickets = append(addedTickets, sourceTickets[index])
		}
	}

	return matchmaker.BackfillProposal{
		BackfillTicketID: result.MatchID,
		AddedTickets:     addedTickets,
		ProposedTeams:    toTeams(sourceTickets, result.MatchingAllies),
		MatchPool:        result.Channel,
		Attribute:        result.PartyAttributes,
		MatchSessionID:   result.MatchSessionID,
	}
}

func (b defaultMatchMaker) addToPartiesRegionInMatchQueueMetrics(rootScope *envelope.Scope, requests []models.MatchmakingRequest,
	sourceTickets []matchmaker.Ticket, channel models.Channel,
) {
	scope := rootScope.NewChildScope("defaultMatchMaker.addToPartiesRegionInMatchQueueMetrics")
	defer scope.Finish()

	remainingRegionStats := make(map[string]map[int]int)
	for _, request := range requests {
		if len(request.SortedLatency) > 0 {
			for _, latency := range request.SortedLatency {
				stats := remainingRegionStats[latency.Region]
				if stats == nil {
					stats = make(map[int]int)
				}
				stats[request.CountPlayer()]++
				remainingRegionStats[latency.Region] = stats
			}
		} else {
			stats := remainingRegionStats["empty-region"]
			if stats == nil {
				stats = make(map[int]int)
			}
			stats[request.CountPlayer()]++
			remainingRegionStats["empty-region"] = stats
		}
	}
}

func getNamespaceMatchPool(tickets []matchmaker.Ticket) (string, string) {
	if len(tickets) == 0 {
		return "", ""
	}
	return tickets[0].Namespace, tickets[0].MatchPool
}

func (b defaultMatchMaker) runMatchMaking(rootScope *envelope.Scope, requests []models.MatchmakingRequest,
	resultChan chan matchmaker.Match, wg *sync.WaitGroup, modelChannel models.Channel, ruleSet models.RuleSet,
	sourceTickets []matchmaker.Ticket,
) {
	scope := rootScope.NewChildScope("runMatchMaking")
	defer scope.Finish()
	defer wg.Done()

	namespace, matchPool := getNamespaceMatchPool(sourceTickets)

	matchResults, _, err := b.mm.MatchPlayers(scope, namespace, matchPool, requests, modelChannel)
	if err != nil {
		scope.Log.Errorf("error making matches: %s", err)
	}

	matchedTicketCount := 0
	for _, result := range matchResults {
		for _, allies := range result.MatchingAllies {
			matchedTicketCount += len(allies.MatchingParties)
		}
		resultChan <- fromMatchResult(result, sourceTickets, ruleSet)
	}
}
