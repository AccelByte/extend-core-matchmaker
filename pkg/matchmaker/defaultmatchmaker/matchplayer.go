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

	// Sort the ticket before choosing a pivot, so the pivot ticket is always the oldest
	sortOldestFirst(matchmakingRequests)

	pivotRequest := matchmakingRequests[0]
	pivotTimeStampRequest := time.Unix(pivotRequest.CreatedAt, 0)

	// Determine if rule needs flexing based on pivot ticket age
	activeRulesetBefore, _ := applyRuleFlexing(ruleset, pivotTimeStampRequest)
	activeRuleset, _ := applyAllianceFlexingRules(activeRulesetBefore, pivotTimeStampRequest)

	scope.Log.WithField("ruleset", activeRuleset).Debug("ruleset applied")

	allianceComposition = DetermineAllianceComposition(activeRuleset)

	// Determine how many regions should this request be matchmaked based on
	// number of regions in the request and attempt count
	filteredRegion := filterRegionByStep(&pivotRequest, &channel)
	regionsToTry := len(filteredRegion)
	if regionsToTry == 0 {
		regionsToTry = 1
	}

regionloop:
	for regionIndex := 0; regionIndex < regionsToTry; regionIndex++ {
		// Make sure pivot request is usable
		if len(pivotRequest.PartyMembers) == 0 {
			break
		}

		// Search for matching tickets using manual search algorithm
		// [MANUALSEARCH]
		result := mm.SearchMatchTickets(&ruleset, &activeRuleset, &channel, regionIndex, &pivotRequest, matchmakingRequests, filteredRegion)

		var mmRequests []models.MatchmakingRequest
		playerCount = 0

		// Insert the pivot request as the first candidate
		req := getMatchmakingRequest(pivotRequest.PartyID, matchmakingRequests)
		if req == nil {
			continue
		}
		mmRequests = append(mmRequests, *req)
		playerCount += len(req.PartyMembers)

		// Add all matching candidates to the request list
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

		// Don't bother finding ally if number of tickets cannot form minimum teams
		if len(mmRequests) < allianceComposition.MinTeam {
			continue
		}

		// Don't bother finding ally if number of matched players is less than minimum needed
		if playerCount < allianceComposition.MinTotalPlayer() {
			continue
		}

		// Prioritize request with more players if configured
		if mm.cfg != nil && mm.cfg.PrioritizeLargerParties {
			sortDESC(mmRequests)
		}

		// Ally finding - attempt to create teams from the matching candidates
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

		// If we found enough allies to form a match
		if len(matchingAllies) >= allianceComposition.MinTeam {
			channelSlug := pivotRequest.Channel
			serverName, _ := pivotRequest.PartyAttributes[models.AttributeServerName].(string)
			clientVersion, _ := pivotRequest.PartyAttributes[models.AttributeClientVersion].(string)

			// Get the matched region (if any)
			region := ""
			if len(pivotRequest.SortedLatency) > 0 && regionIndex < len(pivotRequest.SortedLatency) {
				region = pivotRequest.SortedLatency[regionIndex].Region
			}

			// Filter based on optional match, skip if does not make sense
			optionValuesMap := make(map[string]map[interface{}]int)
			ruleOptions := make(map[string]models.MatchOption)
			isMultiOptions := make(map[string]bool)
			selectedOptions := make(map[string][]interface{})

			// Count the number of times the options and its values are found in each ticket
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
								// Handle single value
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

			// Process match options based on their type
			for name, option := range optionValuesMap {
				switch ruleOptions[name].Type {
				case models.MatchOptionTypeAll:
					// Fail if any party in the session does not have all options
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
						// Common value for all parties
						for val, count := range option {
							if partyCount == count {
								selectedOptions[name] = append(selectedOptions[name], val)
							}
						}
					} else {
						// Fail if cannot find common option
						for val, count := range option {
							// Single party(solo or coop) or multi party game with correct count
							if partyCount == 1 || count > 1 {
								selectedOptions[name] = append(selectedOptions[name], val)
							}
						}
					}

					if len(selectedOptions) == 0 {
						continue regionloop
					}
				case models.MatchOptionTypeUnique:
					// Fail if there's any common option
					for val, count := range option {
						if count > 1 {
							continue regionloop
						}
						selectedOptions[name] = append(selectedOptions[name], val)
					}
				}
			}

			// Combine party attributes into session attributes
			attributes := make(map[string]interface{})

			matchID := utils.GenerateUUID()

			// Process each ally and party to build session attributes
			for _, ally := range matchingAllies {
				for _, party := range ally.MatchingParties {
					// Get attributes from party
					for key, val := range party.PartyAttributes {
						// Only put shared options in the session attributes
						if values, ok := selectedOptions[key]; ok {
							// Make this always an array
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

						// Skip unselected option
						if _, ok := ruleOptions[key]; ok {
							continue
						}

						// Handle special attribute keys
						switch key {
						// Ignoring these keys
						case models.AttributeMatchAttempt:
						case models.AttributeLatencies:
						case models.AttributeServerName:
						case models.AttributeClientVersion:
						case models.AttributeMemberAttr:
						case models.AttributeSubGameMode:
						case models.AttributeBlockedPlayersDetail:
							// Handle blocked players
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
							// Store these keys as "must match this attribute"
							if _, ok := attributes[key]; !ok {
								attributes[key] = val
							}
						}
					}

					// Add party to satisfied tickets
					req := getMatchmakingRequest(party.PartyID, matchmakingRequests)
					if req == nil {
						continue
					}

					satisfiedTickets = append(satisfiedTickets, *req)

					matchmakingRequests = removeMatchmakingRequest(party.PartyID, matchmakingRequests)
				}
			}

			// Keep original attributes, set to single value if the original is not an array
			for key, value := range attributes {
				if isMulti, exists := isMultiOptions[key]; exists && !isMulti {
					if values, ok := value.([]interface{}); ok {
						if len(values) == 1 {
							attributes[key] = values[0]
						}
					}
				}
			}

			// Create the matchmaking result
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
		}

		// If we found matches, add them to batch results and continue with remaining tickets
		if len(mmResults) != 0 {
			batchResult = append(batchResult, mmResults...)
			if len(matchmakingRequests) > 0 && len(matchmakingRequests) >= allianceComposition.MinTeam {
				goto pivotMatching
			}

			// Exit region loop
			break
		}
	}

	// Handle timeout and cleanup of unmatchable tickets
	elapsed := time.Since(startTime)
	reqLen := len(matchmakingRequests)
	playerCount = 0
	for _, mmRequest := range matchmakingRequests {
		playerCount += len(mmRequest.PartyMembers)
	}
	if reqLen > 0 && reqLen >= allianceComposition.MinTeam && !(playerCount < allianceComposition.MinTotalPlayer() && !isUsingAllianceFlexing) && elapsed < timeLimit {
		// Remove the unmatchable ticket from the queue
		// Optimize selecting next pivot in case of unmatchable ticket found
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

	// Track remaining players for metrics
	remainingPlayers := make([]int, len(matchmakingRequests))
	for i, req := range matchmakingRequests {
		remainingPlayers[i] = req.CountPlayer()
	}

	return batchResult, satisfiedTickets, nil
}

// handleSinglePlayer handles matchmaking for single-player scenarios (1v1 or similar).
// This function creates individual matches for each single player request.
func (mm *MatchMaker) handleSinglePlayer(scope *envelope.Scope, namespace string, matchPool string, matchmakingRequests []models.MatchmakingRequest, channel models.Channel) ([]*models.MatchmakingResult, []models.MatchmakingRequest, error) {
	mmResults := make([]*models.MatchmakingResult, 0, len(matchmakingRequests))
	var satisfiedTickets []models.MatchmakingRequest

	for _, req := range matchmakingRequests {
		channelSlug := req.Channel

		// Create a single party for this player
		matchingParties := make([]models.MatchingParty, 0)
		matchingParties = append(matchingParties, createMatchingParty(&req))
		team := models.MatchingAlly{
			MatchingParties: matchingParties,
			PlayerCount:     1,
		}

		// Extract server name and client version
		serverName, _ := req.PartyAttributes[models.AttributeServerName].(string)
		clientVersion, _ := req.PartyAttributes[models.AttributeClientVersion].(string)

		// Get region preference
		region := ""
		if len(req.SortedLatency) > 0 {
			region = req.SortedLatency[0].Region
		}

		// Create matchmaking result for this single player
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

// findMatchingAlly attempts to find matching allies for a pivot ticket.
// This function uses a reordering algorithm to find optimal team combinations.
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

	// Get pivot index and set up reordering
	pivotIndex := getPivotTicketIndexFromTickets(sourceTickets, &pivotTicket)
	elementsAlwaysFirst := []int{pivotIndex}

	// Set up reorder tool with configuration
	maxLoop := 1
	if config != nil && config.FindAllyMaxLoop > 0 {
		maxLoop = config.FindAllyMaxLoop
	}
	r := reordertool.NewOnePointerByLength(len(sourceTickets))
	r.SetOptions(reordertool.Options{
		MaxLoop:             maxLoop,
		ElementsAlwaysFirst: elementsAlwaysFirst,
	})

	// Try different ticket orderings to find optimal matches
	for r.HasNext() {
		newIndexes := r.Get()
		tickets := reorderTickets(sourceTickets, newIndexes)
		// Use alliance rule's max for normal, all unique, & 1 combo rules
		maxAllyCount := allianceRule.MaxNumber
		minAllyCount := allianceRule.MinNumber

		var ticketsPerTeam [][]models.MatchmakingRequest

		// Calculate minimum member number across all tickets
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

		// Step 1: Create a match with min team & min players
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

			// Remove matched tickets from available pool
			for _, reqComb := range matchedTickets {
				tickets = removeMatchmakingRequest(reqComb.PartyID, tickets)
			}
		}

		// Step 2: Fill match up to max team & max players
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

			// Remove matched tickets from available pool
			for _, reqComb := range matchedTickets {
				tickets = removeMatchmakingRequest(reqComb.PartyID, tickets)
			}
		}

		// Step 3: Convert matching tickets to matching allies
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

			// Remove selected tickets
			for _, reqComb := range matchedTickets {
				tickets = removeMatchmakingRequest(reqComb.PartyID, tickets)
			}
		}

		// Check if these alliances can be used to fill a session
		if err := allianceRule.ValidateAllies(teams, blockedPlayerOption); err != nil {
			continue
		}
		return teams, tickets
	}

	return nil, nil
}

// FindPartyCombination finds the optimal combination of parties for a team.
// This function uses a party finder and reordering algorithm to find the best party combination.
func FindPartyCombination(
	config *config.Config,
	sourceTickets []models.MatchmakingRequest,
	pivotTicket models.MatchmakingRequest,
	minPlayer int,
	maxPlayer int,
	current []models.MatchmakingRequest,
	blockedPlayerOption models.BlockedPlayerOption,
) []models.MatchmakingRequest {
	// Define the partyFinder based on player requirements
	pf := GetPartyFinder(minPlayer, maxPlayer, current)

	// Get pivot index and priority indexes for reordering
	pivotIndex := getPivotTicketIndexFromTickets(sourceTickets, &pivotTicket)
	elementsAlwaysFirst := []int{pivotIndex}

	priorityIndexes := getPriorityTicketIndexesFromTickets(sourceTickets)
	for _, priorityIndex := range priorityIndexes {
		if utils.Contains(elementsAlwaysFirst, priorityIndex) {
			continue
		}
		elementsAlwaysFirst = append(elementsAlwaysFirst, priorityIndex)
	}

	// Set up reorder tool with configuration
	maxLoop := 1
	if config != nil && config.FindPartyMaxLoop > 0 {
		maxLoop = config.FindPartyMaxLoop
	}
	r := reordertool.NewTwoPointerByLength(len(sourceTickets))
	r.SetOptions(reordertool.Options{
		MaxLoop:             maxLoop,
		ElementsAlwaysFirst: elementsAlwaysFirst,
	})

	// Try different ticket orderings to find optimal combinations
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

// getMatchmakingRequest finds a matchmaking request by party ID.
// This function returns a pointer to the request if found, nil otherwise.
func getMatchmakingRequest(partyID string, requests []models.MatchmakingRequest) *models.MatchmakingRequest {
	for _, req := range requests {
		if req.PartyID == partyID {
			req := req // For pinning
			return &req
		}
	}
	return nil
}

// removeMatchmakingRequest removes a matchmaking request by party ID.
// This function returns a new slice without the specified request.
func removeMatchmakingRequest(partyID string, requests []models.MatchmakingRequest) []models.MatchmakingRequest {
	var cleanMMRequest []models.MatchmakingRequest
	for _, req := range requests {
		if req.PartyID != partyID {
			cleanMMRequest = append(cleanMMRequest, req)
		}
	}
	return cleanMMRequest
}

// resetTicket resets ticket data from source to destination.
// This function updates destination tickets with data from source tickets.
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

// getNamespace extracts namespace from channel string.
// Temporary solution to get namespace from channel - store channel interface instead in channel cache.
func getNamespace(channel string) (namespace string) {
	ch := strings.Split(channel, ":")
	if len(ch) != 2 {
		return ""
	}
	return ch[0]
}

// getGameMode extracts game mode from channel string.
// Temporary solution to get game mode from channel - store channel interface instead in channel cache.
func getGameMode(channel string) (gameMode string) {
	ch := strings.Split(channel, ":")
	if len(ch) != 2 {
		return ""
	}
	return ch[1]
}

// createMatchingParty creates a matching party from a matchmaking request.
// This function sanitizes the output so that internal fields are not returned.
func createMatchingParty(ticket *models.MatchmakingRequest) models.MatchingParty {
	// Sanitize output so that internal field not returned
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

// sortOldestFirst sorts matchmaking requests by priority (descending) and creation time (ascending).
// This function ensures that older and higher priority tickets are processed first.
func sortOldestFirst(requests []models.MatchmakingRequest) {
	sort.Slice(requests, func(i, j int) bool {
		// Consider priority first (DESC)
		if requests[i].Priority != requests[j].Priority {
			return requests[i].Priority > requests[j].Priority
		}
		// Then, createdAt (ASC)
		return requests[i].CreatedAt < requests[j].CreatedAt
	})
}
