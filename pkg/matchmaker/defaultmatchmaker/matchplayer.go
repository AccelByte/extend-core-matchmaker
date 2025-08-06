// Copyright (c) 2019-2024 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package defaultmatchmaker provides the default implementation of the MatchLogic interface.
// This package contains the core matchmaking algorithms and logic for creating matches from tickets.
package defaultmatchmaker

import (
	"sort"
	"strings"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/config"
	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
	reordertool "github.com/AccelByte/extend-core-matchmaker/pkg/utils/reorder-tool"
)

// Constants for attribute keys used in matchmaking
const (
	partyAttributesKey  = "party_attributes"         // Key for party-level attributes
	memberAttributesKey = models.AttributeMemberAttr // Key for member-level attributes
	membersKey          = "party_members"            // Key for party members
	serverNameKey       = "server_name"              // Key for server name
	clientVersionKey    = "client_version"           // Key for client version
	userIDKey           = "user_id"                  // Key for user ID
	latencyMapKey       = "latency_map"              // Key for latency mapping
	distanceCriteria    = "distance"                 // Key for distance criteria
	// June 6th, 1983 00:00:00 - Date used to force flexing rules
	dateToForceFlexingRule = 423792000
)

// Ensure MatchMaker implements the matchmaker.Matchmaker interface
var _ matchmaker.Matchmaker = (*MatchMaker)(nil)

// MatchMaker is the main matchmaking engine that implements the Matchmaker interface.
// It handles player matching, session management, and various matchmaking strategies.
type MatchMaker struct {
	cfg              *config.Config // Configuration for the matchmaker
	isMatchAnyCommon bool           // Flag to enable matching any common attributes
}

// NewMatchMaker creates a new instance of the MatchMaker with the given configuration.
func NewMatchMaker(cfg *config.Config) *MatchMaker {
	return &MatchMaker{
		cfg:              cfg,
		isMatchAnyCommon: cfg.FlagAnyMatchOptionAllCommon,
	}
}

// MatchPlayers tries to match as many request as possible.
// This is the main entry point for player matchmaking operations.
//
//nolint:gocyclo
func (mm *MatchMaker) MatchPlayers(rootScope *envelope.Scope, namespace string, matchPool string, matchmakingRequests []models.MatchmakingRequest, channel models.Channel) ([]*models.MatchmakingResult, []models.MatchmakingRequest, error) {
	scope := rootScope.NewChildScope("MatchMaker.MatchPlayers")
	defer scope.Finish()

	var satisfiedTickets []models.MatchmakingRequest

	var (
		pivotMatchingCounter    int // Counter for pivot-based matching attempts
		findMatchingAllyCounter int // Counter for ally finding attempts
	)

	// Early return if no requests to process
	if len(matchmakingRequests) == 0 {
		return nil, nil, nil
	}

	ruleset := channel.Ruleset

	// Determine the alliance composition based on the ruleset
	allianceComposition := DetermineAllianceComposition(ruleset)

	// Check if alliance flexing is enabled
	isUsingAllianceFlexing := false
	if len(ruleset.AllianceFlexingRule) > 0 {
		isUsingAllianceFlexing = true
	}

	// Check if there are enough requests to form a match
	if len(matchmakingRequests) < allianceComposition.MinTeam && !isUsingAllianceFlexing {
		return nil, nil, nil
	}

	// Count total players across all requests
	playerCount := 0
	for _, mmRequest := range matchmakingRequests {
		playerCount += len(mmRequest.PartyMembers)
	}
	if playerCount < allianceComposition.MinTotalPlayer() && !isUsingAllianceFlexing {

		return nil, nil, nil
	}

	// Handle single player scenarios (1v1 or similar)
	isSinglePlayer := allianceComposition.MaxPlayer == 1 && allianceComposition.MinTeam == 1 && allianceComposition.MaxTeam == 1
	if isSinglePlayer {
		return mm.handleSinglePlayer(scope, namespace, matchPool, matchmakingRequests, channel)
	}

	// Set up timeout safeguard for pool lock
	startTime := time.Now()
	timeLimit := (constants.PoolLockTimeLimit * 2) / 5
	if mm.cfg != nil && mm.cfg.MatchTimeLimitSecond > 0 {
		timeLimit = time.Duration(mm.cfg.MatchTimeLimitSecond) * time.Second
	}

	batchResult := make([]*models.MatchmakingResult, 0)

pivotMatching:
	pivotMatchingCounter++
	scope.Log.Debugf("executing %d requests on local pool", len(matchmakingRequests))
	scope.Log.WithField("matchmakingRequests", matchmakingRequests).Debug("incoming requests")

	// sort the ticket before choosing a pivot, so the pivot ticket is always the oldest
	sortOldestFirst(matchmakingRequests)

	pivotRequest := matchmakingRequests[0]
	pivotTimeStampRequest := time.Unix(pivotRequest.CreatedAt, 0)

	// determine if rule needs flexing
	activeRulesetBefore, _ := applyRuleFlexing(ruleset, pivotTimeStampRequest)
	activeRuleset, _ := applyAllianceFlexingRules(activeRulesetBefore, pivotTimeStampRequest)

	scope.Log.WithField("ruleset", activeRuleset).Debug("ruleset applied")

	allianceComposition = DetermineAllianceComposition(activeRuleset)

	// determine how many regions should this request be matchmaked based on
	// number of regions in the request and attempt count
	filteredRegion := filterRegionByStep(&pivotRequest, &channel)
	regionsToTry := len(filteredRegion)
	if regionsToTry == 0 {
		regionsToTry = 1
	}

regionloop:
	for regionIndex := 0; regionIndex < regionsToTry; regionIndex++ {
		// make sure pivot request is usable
		if len(pivotRequest.PartyMembers) == 0 {
			break
		}

		// [MANUALSEARCH]
		result := mm.SearchMatchTickets(&ruleset, &activeRuleset, &channel, regionIndex, &pivotRequest, matchmakingRequests, filteredRegion)

		var mmRequests []models.MatchmakingRequest
		playerCount = 0

		// insert the pivot request
		req := getMatchmakingRequest(pivotRequest.PartyID, matchmakingRequests)
		if req == nil {
			continue
		}
		mmRequests = append(mmRequests, *req)
		playerCount += len(req.PartyMembers)

		for resultIndex := range result {
			// [MANUALSEARCH]
			req = &result[resultIndex]

			if len(req.PartyMembers) == 0 {
				continue
			}

			mmRequests = append(mmRequests, *req)
			playerCount += len(req.PartyMembers)
		}

		scope.Log.WithField("matching_candidates_number", len(mmRequests)).WithField("matching_candidates", mmRequests).Debug("matching candidates found")

		// don't bother finding ally if number of tickets cannot form minimum teams
		if len(mmRequests) < allianceComposition.MinTeam {
			continue
		}

		// don't bother finding ally if number of matched players is less than minimum needed
		if playerCount < allianceComposition.MinTotalPlayer() {
			continue
		}

		// prioritize request with more players
		if mm.cfg != nil && mm.cfg.PrioritizeLargerParties {
			sortDESC(mmRequests)
		}

		// ally finding
		var mmResults []*models.MatchmakingResult
		var matchingAllies []models.MatchingAlly

		{
			matchingAllies, _ = findMatchingAlly(
				scope,
				mm.cfg,
				mmRequests,
				pivotRequest,
				activeRuleset.AllianceRule,
				channel.Ruleset.MatchingRule,
				channel.Ruleset.BlockedPlayerOption,
			)

			findMatchingAllyCounter++
		}

		if len(matchingAllies) >= allianceComposition.MinTeam {
			channelSlug := pivotRequest.Channel
			serverName, _ := pivotRequest.PartyAttributes[models.AttributeServerName].(string)
			clientVersion, _ := pivotRequest.PartyAttributes[models.AttributeClientVersion].(string)

			// get the matched region (if any)
			region := ""
			if len(pivotRequest.SortedLatency) > 0 && regionIndex < len(pivotRequest.SortedLatency) {
				region = pivotRequest.SortedLatency[regionIndex].Region
			}

			// filter based on optional match, skip if does not make sense
			optionValuesMap := make(map[string]map[interface{}]int)
			ruleOptions := make(map[string]models.MatchOption)
			isMultiOptions := make(map[string]bool)
			selectedOptions := make(map[string][]interface{})

			// count the number of times the options and its values are found in each ticket
			for _, option := range activeRuleset.MatchOptions.Options {
				ruleOptions[option.Name] = option

				for _, ally := range matchingAllies {
					for _, party := range ally.MatchingParties {
						if v, o := party.PartyAttributes[option.Name]; o {
							if optionValuesMap[option.Name] == nil {
								optionValuesMap[option.Name] = make(map[interface{}]int)
							}

							multival, ok := v.([]interface{})
							isMultiOptions[option.Name] = ok
							if !ok {
								// handle single value
								optionValuesMap[option.Name][v]++
								continue
							}
							for _, val := range multival {
								optionValuesMap[option.Name][val]++
							}
						}
					}
				}
			}

			for name, option := range optionValuesMap {
				switch ruleOptions[name].Type {
				case models.MatchOptionTypeAll:
					// fail if any party in the session does not have all options
					partyCount := 0
					for _, ally := range matchingAllies {
						partyCount += len(ally.MatchingParties)
					}
					for val, count := range option {
						if partyCount != count {
							continue regionloop
						}
						selectedOptions[name] = append(selectedOptions[name], val)
					}
				case models.MatchOptionTypeAny:
					partyCount := 0
					for _, ally := range matchingAllies {
						partyCount += len(ally.MatchingParties)
					}

					if mm.isMatchAnyCommon {
						// common value for all parties
						for val, count := range option {
							if partyCount == count {
								selectedOptions[name] = append(selectedOptions[name], val)
							}
						}
					} else {
						// fail if cannot find common option
						for val, count := range option {
							// single party(solo or coop) or multi party game with correct count
							if partyCount == 1 || count > 1 {
								selectedOptions[name] = append(selectedOptions[name], val)
							}
						}
					}

					if len(selectedOptions) == 0 {
						continue regionloop
					}
				case models.MatchOptionTypeUnique:
					// fail if there's any common option
					for val, count := range option {
						if count > 1 {
							continue regionloop
						}
						selectedOptions[name] = append(selectedOptions[name], val)
					}
				}
			}

			attributes := make(map[string]interface{}) // store combined party attributes into session

			matchID := utils.GenerateUUID()

			for _, ally := range matchingAllies {
				for _, party := range ally.MatchingParties {
					// get attributes
					for key, val := range party.PartyAttributes {
						// only put shared options in the session attributes
						if values, ok := selectedOptions[key]; ok {
							// make this always an array
							if attr, k := attributes[key]; k {
								if arr, isArr := attr.([]interface{}); isArr {
								optionvaluesloop:
									for _, v := range values {
										for _, item := range arr {
											if v == item {
												continue optionvaluesloop
											}
										}
										arr = append(arr, v)
									}
									attributes[key] = arr
								} else {
									attributes[key] = values
								}
							} else {
								attributes[key] = values
							}
							continue
						}

						// skip unselected option
						if _, ok := ruleOptions[key]; ok {
							continue
						}

						switch key {
						// ignoring these keys
						case models.AttributeMatchAttempt:
						case models.AttributeLatencies:
						case models.AttributeServerName:
						case models.AttributeClientVersion:
						case models.AttributeMemberAttr:
						case models.AttributeSubGameMode:
						case models.AttributeBlockedPlayersDetail:
							// handle blocked players
						case models.AttributeBlocked:
							ids, ok := val.([]interface{})
							if !ok {
								continue
							}

							var list []interface{}

							v, ok := attributes[models.AttributeBlocked]
							if ok {
								if l, o := v.([]interface{}); o {
									list = l
								}
							}

							list = append(list, ids...)
							attributes[models.AttributeBlocked] = list

						default:
							// store these keys as "must match this attribute"
							if _, ok := attributes[key]; !ok {
								attributes[key] = val
							}
						}
					}

					req := getMatchmakingRequest(party.PartyID, matchmakingRequests)
					if req == nil {
						continue
					}

					satisfiedTickets = append(satisfiedTickets, *req)

					matchmakingRequests = removeMatchmakingRequest(party.PartyID, matchmakingRequests)
				}
			}

			// kept original attributes, set to single value if the original is not an array
			for key, value := range attributes {
				if isMulti, exists := isMultiOptions[key]; exists && !isMulti {
					if values, ok := value.([]interface{}); ok {
						if len(values) == 1 {
							attributes[key] = values[0]
						}
					}
				}
			}

			mmResults = append(mmResults, &models.MatchmakingResult{
				Status:          models.MatchmakingStatusDone,
				MatchID:         matchID,
				Channel:         channelSlug,
				Namespace:       getNamespace(channelSlug),
				GameMode:        getGameMode(channelSlug),
				ServerName:      serverName,
				ClientVersion:   clientVersion,
				Region:          region,
				MatchingAllies:  matchingAllies,
				PartyAttributes: attributes,
				UpdatedAt:       time.Now(),
				PivotID:         pivotRequest.PartyID,
			})

			// needRequestRotation = false
		}

		if len(mmResults) != 0 {
			batchResult = append(batchResult, mmResults...)
			if len(matchmakingRequests) > 0 && len(matchmakingRequests) >= allianceComposition.MinTeam {
				goto pivotMatching
			}

			// exit region loop
			break
		}
	}

	elapsed := time.Since(startTime)
	reqLen := len(matchmakingRequests)
	playerCount = 0
	for _, mmRequest := range matchmakingRequests {
		playerCount += len(mmRequest.PartyMembers)
	}
	if reqLen > 0 && reqLen >= allianceComposition.MinTeam && !(playerCount < allianceComposition.MinTotalPlayer() && !isUsingAllianceFlexing) && elapsed < timeLimit {
		// remove the unmatchable ticket from the queue
		// optimize selecting next pivot in case of unmatchable ticket found
		var removedID int
		for i, mmRequest := range matchmakingRequests {
			if mmRequest.PartyID == pivotRequest.PartyID {
				removedID = i
				break
			}
		}

		switch removedID {
		case 0:
			matchmakingRequests = matchmakingRequests[removedID+1:]
		case len(matchmakingRequests):
			matchmakingRequests = matchmakingRequests[:removedID-1]
		default:
			matchmakingRequests = append(matchmakingRequests[:removedID], matchmakingRequests[removedID+1:]...)
		}

		if len(matchmakingRequests) > 0 && reqLen >= allianceComposition.MinTeam {
			goto pivotMatching
		}
	}

	remainingPlayers := make([]int, len(matchmakingRequests))
	for i, req := range matchmakingRequests {
		remainingPlayers[i] = req.CountPlayer()
	}

	return batchResult, satisfiedTickets, nil
}

func (mm *MatchMaker) handleSinglePlayer(scope *envelope.Scope, namespace string, matchPool string, matchmakingRequests []models.MatchmakingRequest, channel models.Channel) ([]*models.MatchmakingResult, []models.MatchmakingRequest, error) {
	mmResults := make([]*models.MatchmakingResult, 0, len(matchmakingRequests))
	var satisfiedTickets []models.MatchmakingRequest

	for _, req := range matchmakingRequests {
		channelSlug := req.Channel

		matchingParties := make([]models.MatchingParty, 0)
		matchingParties = append(matchingParties, createMatchingParty(&req))
		team := models.MatchingAlly{
			MatchingParties: matchingParties,
			PlayerCount:     1,
		}

		serverName, _ := req.PartyAttributes[models.AttributeServerName].(string)
		clientVersion, _ := req.PartyAttributes[models.AttributeClientVersion].(string)

		region := ""
		if len(req.SortedLatency) > 0 {
			region = req.SortedLatency[0].Region
		}

		mmResults = append(mmResults, &models.MatchmakingResult{
			Status:          models.MatchmakingStatusDone,
			MatchID:         utils.GenerateUUID(),
			Channel:         channelSlug,
			Namespace:       getNamespace(channelSlug),
			GameMode:        getGameMode(channelSlug),
			PartyAttributes: req.PartyAttributes,
			MatchingAllies:  []models.MatchingAlly{team},
			ServerName:      serverName,
			ClientVersion:   clientVersion,
			Region:          region,
			UpdatedAt:       time.Now(),
			PivotID:         req.PartyID,
		})
	}

	return mmResults, satisfiedTickets, nil
}

func findMatchingAlly(
	rootScope *envelope.Scope,
	config *config.Config,
	sourceTickets []models.MatchmakingRequest,
	pivotTicket models.MatchmakingRequest,
	allianceRule models.AllianceRule,
	matchingRules []models.MatchingRule,
	blockedPlayerOption models.BlockedPlayerOption,
) ([]models.MatchingAlly, []models.MatchmakingRequest) {
	scope := rootScope.NewChildScope("findMatchingAlly")
	defer scope.Finish()

	// get pivot index
	pivotIndex := getPivotTicketIndexFromTickets(sourceTickets, &pivotTicket)
	elementsAlwaysFirst := []int{pivotIndex}

	// set up reorder tool
	maxLoop := 1
	if config != nil && config.FindAllyMaxLoop > 0 {
		maxLoop = config.FindAllyMaxLoop
	}
	r := reordertool.NewOnePointerByLength(len(sourceTickets))
	r.SetOptions(reordertool.Options{
		MaxLoop:             maxLoop,
		ElementsAlwaysFirst: elementsAlwaysFirst,
	})

	for r.HasNext() {
		newIndexes := r.Get()
		tickets := reorderTickets(sourceTickets, newIndexes)
		// use alliance rule's max for normal, all unique, & 1 combo rules
		maxAllyCount := allianceRule.MaxNumber
		minAllyCount := allianceRule.MinNumber

		var ticketsPerTeam [][]models.MatchmakingRequest

		minMemberNumber := allianceRule.PlayerMaxNumber
		for _, ticket := range tickets {
			memberCount := len(ticket.PartyMembers)
			if minMemberNumber > len(ticket.PartyMembers) {
				minMemberNumber = memberCount
			}
		}

		playerMaxNumber := allianceRule.PlayerMinNumber
		if playerMaxNumber < minMemberNumber {
			playerMaxNumber = minMemberNumber
		}

		// step 1: create a match with min team & min players
		for i := 0; i < minAllyCount; i++ {
			matchedTickets := FindPartyCombination(
				config,
				tickets,
				pivotTicket,
				allianceRule.PlayerMinNumber,
				playerMaxNumber,
				nil,
				blockedPlayerOption,
			)

			if len(matchedTickets) == 0 {
				break
			}

			ticketsPerTeam = append(ticketsPerTeam, matchedTickets)

			for _, reqComb := range matchedTickets {
				tickets = removeMatchmakingRequest(reqComb.PartyID, tickets)
			}
		}

		// step 2: fill match up to max team & max players
		for i := 0; i < maxAllyCount; i++ {
			var curTeamTickets []models.MatchmakingRequest
			if i < len(ticketsPerTeam) {
				curTeamTickets = ticketsPerTeam[i]
			}

			matchedTickets := FindPartyCombination(
				config,
				tickets,
				pivotTicket,
				allianceRule.PlayerMinNumber,
				allianceRule.PlayerMaxNumber,
				curTeamTickets,
				blockedPlayerOption,
			)

			if len(matchedTickets) == 0 {
				break
			}

			if i < len(ticketsPerTeam) {
				ticketsPerTeam[i] = matchedTickets
			} else {
				ticketsPerTeam = append(ticketsPerTeam, matchedTickets)
			}

			for _, reqComb := range matchedTickets {
				tickets = removeMatchmakingRequest(reqComb.PartyID, tickets)
			}
		}

		// step 3: convert matching tickets to matching allies
		var teams []models.MatchingAlly
		for _, matchedTickets := range ticketsPerTeam {
			matchingParties := make([]models.MatchingParty, 0)

			for _, req := range matchedTickets {
				matchingParties = append(matchingParties, createMatchingParty(&req))
			}

			team := models.MatchingAlly{
				MatchingParties: matchingParties,
				PlayerCount:     countPlayers(matchedTickets),
			}
			teams = append(teams, team)

			// remove selected tickets
			for _, reqComb := range matchedTickets {
				tickets = removeMatchmakingRequest(reqComb.PartyID, tickets)
			}
		}

		// check if these alliances can be used to fill a session
		if err := allianceRule.ValidateAllies(teams, blockedPlayerOption); err != nil {
			continue
		}
		return teams, tickets
	}

	return nil, nil
}

func FindPartyCombination(
	config *config.Config,
	sourceTickets []models.MatchmakingRequest,
	pivotTicket models.MatchmakingRequest,
	minPlayer int,
	maxPlayer int,
	current []models.MatchmakingRequest,
	blockedPlayerOption models.BlockedPlayerOption,
) []models.MatchmakingRequest {
	// define the partyFinder
	pf := GetPartyFinder(minPlayer, maxPlayer, current)

	// get pivot index and priority indexes
	pivotIndex := getPivotTicketIndexFromTickets(sourceTickets, &pivotTicket)
	elementsAlwaysFirst := []int{pivotIndex}

	priorityIndexes := getPriorityTicketIndexesFromTickets(sourceTickets)
	for _, priorityIndex := range priorityIndexes {
		if utils.Contains(elementsAlwaysFirst, priorityIndex) {
			continue
		}
		elementsAlwaysFirst = append(elementsAlwaysFirst, priorityIndex)
	}

	// set up reorder tool
	maxLoop := 1
	if config != nil && config.FindPartyMaxLoop > 0 {
		maxLoop = config.FindPartyMaxLoop
	}
	r := reordertool.NewTwoPointerByLength(len(sourceTickets))
	r.SetOptions(reordertool.Options{
		MaxLoop:             maxLoop,
		ElementsAlwaysFirst: elementsAlwaysFirst,
	})

	for r.HasNext() {
		newIndexes := r.Get()
		tickets := reorderTickets(sourceTickets, newIndexes)

		pf.Reset()
		for _, ticket := range tickets {
			/*
				[AR-7033] check blocked players for:
				- respect block only for the same team
			*/
			if blockedPlayerOption == models.BlockedPlayerCanMatchOnDifferentTeam &&
				isContainBlockedPlayers(pf.GetCurrentResult(), &ticket) {
				continue
			}
			success := pf.AssignMembers(ticket)
			if !success {
				continue
			}
			pf.AppendResult(ticket)
		}
		if pf.IsFulfilled() {
			return pf.GetCurrentResult()
		}
	}

	return pf.GetBestResult()
}

func getMatchmakingRequest(partyID string, requests []models.MatchmakingRequest) *models.MatchmakingRequest {
	for _, req := range requests {
		if req.PartyID == partyID {
			req := req // for pinning
			return &req
		}
	}
	return nil
}

func removeMatchmakingRequest(partyID string, requests []models.MatchmakingRequest) []models.MatchmakingRequest {
	var cleanMMRequest []models.MatchmakingRequest
	for _, req := range requests {
		if req.PartyID != partyID {
			cleanMMRequest = append(cleanMMRequest, req)
		}
	}
	return cleanMMRequest
}

func resetTicket(dest, source []models.MatchmakingRequest) {
	for i, rt := range dest {
		for _, t := range source {
			if rt.PartyID == t.PartyID {
				dest[i] = t
				break
			}
		}
	}
}

// temporary solution to get ns from channel
// store channel interface instead in channel cache
func getNamespace(channel string) (namespace string) {
	ch := strings.Split(channel, ":")
	if len(ch) != 2 {
		return ""
	}
	return ch[0]
}

// temporary solution to get game mode from channel
// store channel interface instead in channel cache
func getGameMode(channel string) (gameMode string) {
	ch := strings.Split(channel, ":")
	if len(ch) != 2 {
		return ""
	}
	return ch[1]
}

func createMatchingParty(ticket *models.MatchmakingRequest) models.MatchingParty {
	// sanitize output so that internal field not returned
	attr := ticket.PartyAttributes
	delete(attr, models.AttributeMatchAttempt)
	return models.MatchingParty{
		PartyID:         ticket.PartyID,
		PartyAttributes: attr,
		PartyMembers:    ticket.PartyMembers,
		MatchAttributes: models.MatchAttributes{
			FirstTicketCreatedAt: ticket.CreatedAt,
		},
	}
}

func sortOldestFirst(requests []models.MatchmakingRequest) {
	sort.Slice(requests, func(i, j int) bool {
		// consider priority first (DESC)
		if requests[i].Priority != requests[j].Priority {
			return requests[i].Priority > requests[j].Priority
		}
		// then, createdAt (ASC)
		return requests[i].CreatedAt < requests[j].CreatedAt
	})
}
