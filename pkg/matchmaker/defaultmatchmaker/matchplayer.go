// Copyright (c) 2019-2024 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/config"
	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker"
	"github.com/AccelByte/extend-core-matchmaker/pkg/metrics"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance/rebalance_v1"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance/rebalance_v2"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
	reordertool "github.com/AccelByte/extend-core-matchmaker/pkg/utils/reorder-tool"

	"github.com/sirupsen/logrus"
)

type elapsedTimer struct {
	startTime     time.Time
	endTime       time.Time
	totalDuration time.Duration
}

func (e *elapsedTimer) start() {
	e.startTime = time.Now().UTC()
}

func (e *elapsedTimer) end() {
	e.endTime = time.Now().UTC()
}

func (e *elapsedTimer) elapsed() time.Duration {
	if e.startTime.IsZero() || e.endTime.IsZero() {
		return 0
	}
	return e.endTime.Sub(e.startTime)
}

func (e *elapsedTimer) appendElapsed() {
	e.totalDuration += e.elapsed()
	e.startTime = time.Time{} // reset timer
}

func (e *elapsedTimer) totalElapsed() time.Duration {
	if e.totalDuration == 0 {
		return e.elapsed()
	}
	return e.totalDuration
}

const (
	partyAttributesKey  = "party_attributes"
	memberAttributesKey = models.AttributeMemberAttr
	membersKey          = "party_members"
	serverNameKey       = "server_name"
	clientVersionKey    = "client_version"
	userIDKey           = "user_id"
	latencyMapKey       = "latency_map"
	distanceCriteria    = "distance"
	// June 6th, 1983 00:00:00
	dateToForceFlexingRule = 423792000
)

var _ matchmaker.Matchmaker = (*MatchMaker)(nil)

type MatchMaker struct {
	cfg                *config.Config
	metrics            metrics.MatchmakingMetrics
	isMatchAnyCommon   bool
	useCurrentPlatform bool
}

func NewMatchMaker(cfg *config.Config, metrics metrics.MatchmakingMetrics) *MatchMaker {
	return &MatchMaker{
		cfg:                cfg,
		metrics:            metrics,
		isMatchAnyCommon:   cfg.FlagAnyMatchOptionAllCommon,
		useCurrentPlatform: !cfg.CrossPlatformNoCurrentPlatform,
	}
}

func (mm *MatchMaker) addUnmatchedReasonMetric(namespace string, channel string, reason string) {
	if mm.metrics != nil {
		mm.metrics.AddUnmatchedReason(namespace, channel, reason)
	}
}

// MatchPlayers tries to match as many request as possible
//
//nolint:gocyclo
func (mm *MatchMaker) MatchPlayers(rootScope *envelope.Scope, namespace string, matchPool string, matchmakingRequests []models.MatchmakingRequest, channel models.Channel) ([]*models.MatchmakingResult, []models.MatchmakingRequest, error) {
	scope := rootScope.NewChildScope("MatchMaker.MatchPlayers")
	defer scope.Finish()

	var satisfiedTickets []models.MatchmakingRequest

	var (
		matchPlayersTimer          elapsedTimer
		bleveIndexingTimer         elapsedTimer
		applyFlexingTimer          elapsedTimer
		regionLoopTimer            elapsedTimer
		runBleveQueryTimer         elapsedTimer
		findMatchingAllyTimer      elapsedTimer
		filterOptionsTimer         elapsedTimer
		prepareMatchingAlliesTimer elapsedTimer
		findAllyTimer              *timer
		pivotMatchingCounter       int
		regionsToTryCounter        int
		findMatchingAllyCounter    int
	)
	matchPlayersTimer.start()
	findAllyTimer = new(timer)

	defer func() {
		matchPlayersTimer.end()

		elapsedTimeMaps := map[string]time.Duration{
			"totalMatchPlayers":      matchPlayersTimer.elapsed(),
			"bleveIndexing":          bleveIndexingTimer.elapsed(),
			"applyFlexing":           applyFlexingTimer.totalElapsed(),
			"regionLoop":             regionLoopTimer.totalElapsed(),
			"runBleveQuery":          runBleveQueryTimer.totalElapsed(),
			"findMatchingAlly":       findMatchingAllyTimer.totalElapsed(),
			"filterOptions":          filterOptionsTimer.totalElapsed(),
			"prepareMatchingAllies":  prepareMatchingAlliesTimer.totalElapsed(),
			"findAllyReorderTickets": findAllyTimer.reorderTicketsTimer.totalElapsed(),
			"findAllyStep1":          findAllyTimer.step1Timer.totalElapsed(),
			"findAllyStep2":          findAllyTimer.step2Timer.totalElapsed(),
			"findAllyStep3":          findAllyTimer.step3Timer.totalElapsed(),
			"findAllyRebalance":      findAllyTimer.rebalanceTimer.totalElapsed(),
			"findAllyValidateAllies": findAllyTimer.validateAlliesTimer.totalElapsed(),
		}

		l := logrus.WithFields(convertToMapInterface(elapsedTimeMaps))
		l.WithFields(logrus.Fields{
			"matchPool":             matchPool,
			"findAllyLoopCount":     findAllyTimer.loopCounter,
			"pivotMatchingCount":    pivotMatchingCounter,
			"regionsToTryCount":     regionsToTryCounter,
			"findMatchingAllyCount": findMatchingAllyCounter,
		}).Debug("MatchMaker Match Players")

		// send elapsed time metric
		for k, v := range elapsedTimeMaps {
			mm.metrics.AddMatchPlayersElapsedTimeMs(namespace, matchPool, k, v)
		}
	}()

	if len(matchmakingRequests) == 0 {
		return nil, nil, nil
	}

	ruleset := channel.Ruleset

	allianceComposition := matchmaker.DetermineAllianceComposition(ruleset, ruleset.IsUseSubGamemode())

	isUsingAllianceFlexing := false
	if len(ruleset.AllianceFlexingRule) > 0 {
		isUsingAllianceFlexing = true
	}
	if ruleset.IsUseSubGamemode() {
		isUsingAllianceFlexing = false
		for _, subGameMode := range ruleset.SubGameModes {
			if len(subGameMode.AllianceFlexingRule) > 0 {
				isUsingAllianceFlexing = true
			}
		}
	}

	// not enough requests to be matched together
	if len(matchmakingRequests) < allianceComposition.MinTeam && !isUsingAllianceFlexing {
		mm.addUnmatchedReasonMetric(namespace, matchPool, constants.ReasonNotEnoughRequests)
		scope.Log.Debugf("%s to be matched, min: %d, found: %d", constants.ReasonNotEnoughRequests,
			allianceComposition.MinTeam, len(matchmakingRequests))

		return nil, nil, nil
	}

	playerCount := 0
	for _, mmRequest := range matchmakingRequests {
		playerCount += len(mmRequest.PartyMembers)
	}
	if playerCount < allianceComposition.MinTotalPlayer() && !isUsingAllianceFlexing {
		mm.addUnmatchedReasonMetric(namespace, matchPool, constants.ReasonNotEnoughPlayers)
		scope.Log.Debugf("%s to be matched, min: %d, found: %d", constants.ReasonNotEnoughPlayers,
			allianceComposition.MinTotalPlayer(), playerCount)

		return nil, nil, nil
	}

	isSinglePlayer := allianceComposition.MaxPlayer == 1 && allianceComposition.MinTeam == 1 && allianceComposition.MaxTeam == 1
	if isSinglePlayer {
		return mm.handleSinglePlayer(scope, namespace, matchPool, matchmakingRequests, channel)
	}

	// pool lock timeout safeguard
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

	// role-based flexing
	applyRoleBasedFlexing(matchmakingRequests, &channel)

	allianceComposition = matchmaker.DetermineAllianceComposition(activeRuleset, activeRuleset.IsUseSubGamemode())

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

		/* // [BLEVE]
		for _, matchResult := range result.Hits {
			req = getMatchmakingRequest(matchResult.ID, matchmakingRequests)
		*/

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
			mm.addUnmatchedReasonMetric(namespace, matchPool, "not_enough_teams")

			continue
		}

		// don't bother finding ally if number of matched players is less than minimum needed
		if playerCount < allianceComposition.MinTotalPlayer() {
			mm.addUnmatchedReasonMetric(namespace, matchPool, "not_enough_alliance_players")
			continue
		}

		// prioritize request with more players
		if mm.cfg != nil && mm.cfg.PrioritizeLargerParties {
			sortDESC(mmRequests)
		}

		// ally finding
		var mmResults []*models.MatchmakingResult
		var matchingAllies []models.MatchingAlly
		activeAllianceRule := activeRuleset.AllianceRule

		var selectedSubGamemodes []interface{}

		if activeRuleset.IsUseSubGamemode() {
			// use pivot ticket's sub gamemode
			subGameModeVal, ok := pivotRequest.PartyAttributes[models.AttributeSubGameMode]
			if ok {
				var subGameModeNames []interface{}
				subGameModeNames, ok := subGameModeVal.([]interface{})
				if !ok {
					// try using the value as a string rather than array of string
					subGameModeNames = append(subGameModeNames, subGameModeVal)
				}

				// shuffle sub gamemode array
				rand.Shuffle(len(subGameModeNames), func(i, j int) { subGameModeNames[i], subGameModeNames[j] = subGameModeNames[j], subGameModeNames[i] })

			subgamemodeloop:
				for _, v := range subGameModeNames {
					name, ok := v.(string)
					if !ok {
						continue
					}
					if subGameMode, ok := activeRuleset.SubGameModes[name]; ok {
						// filter out candidates that does not have the same subgamemode
						tempRequests := mmRequests
						for _, req := range tempRequests {
							val, ok := req.PartyAttributes[models.AttributeSubGameMode]
							if !ok {
								tempRequests = removeMatchmakingRequest(req.PartyID, tempRequests)
								continue
							}
							names, ok := val.([]interface{})
							if !ok {
								names = append(names, val)
							}

							found := false
							for _, v := range names {
								n, ok := v.(string)
								if !ok {
									tempRequests = removeMatchmakingRequest(req.PartyID, tempRequests)
									continue
								}
								if n == name {
									found = true
									break
								}
							}
							if !found {
								tempRequests = removeMatchmakingRequest(req.PartyID, tempRequests)
								continue
							}
						}

						tempMatchingAllies, _ := findMatchingAlly(
							scope,
							mm.cfg,
							tempRequests,
							pivotRequest,
							subGameMode.AllianceRule,
							channel.Ruleset.GetRebalanceMode(),
							channel.Ruleset.MatchingRule,
							findAllyTimer,
							channel.Ruleset.BlockedPlayerOption,
						)

						if len(tempMatchingAllies) >= subGameMode.AllianceRule.MinNumber {
							matchingAllies = tempMatchingAllies
							activeAllianceRule = subGameMode.AllianceRule
							allianceComposition.MinTeam = activeAllianceRule.MinNumber
							allianceComposition.MinPlayer = activeAllianceRule.PlayerMinNumber
							break subgamemodeloop
						}
					}
				}
			}
		} else {
			findMatchingAllyTimer.start()
			matchingAllies, _ = findMatchingAlly(
				scope,
				mm.cfg,
				mmRequests,
				pivotRequest,
				activeRuleset.AllianceRule,
				channel.Ruleset.GetRebalanceMode(),
				channel.Ruleset.MatchingRule,
				findAllyTimer,
				channel.Ruleset.BlockedPlayerOption,
			)
			findMatchingAllyTimer.end()
			findMatchingAllyTimer.appendElapsed()

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
			filterOptionsTimer.start()
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
							mm.addUnmatchedReasonMetric(namespace, matchPool, "unmatched_all_options")
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
							mm.addUnmatchedReasonMetric(namespace, matchPool, "unmatched_unique_options")
							continue regionloop
						}
						selectedOptions[name] = append(selectedOptions[name], val)
					}
				}
			}
			filterOptionsTimer.end()
			filterOptionsTimer.appendElapsed()

			// populate subgamemode in the party attribute
			if activeRuleset.IsUseSubGamemode() {
				// use pivot ticket's sub gamemodes
				subGameModeVal, ok := pivotRequest.PartyAttributes[models.AttributeSubGameMode]
				if ok {
					var subGameModeNames []interface{}
					subGameModeNames, ok := subGameModeVal.([]interface{})
					if !ok {
						// try using the value as a string rather than array of string
						subGameModeNames = append(subGameModeNames, subGameModeVal)
					}

					// count game mode occurrences in all tickets
					pivotGameModes := make(map[string]struct{})
					gameModeCounts := make(map[string]int)

					for _, v := range subGameModeNames {
						name, ok := v.(string)
						if !ok {
							continue
						}
						if _, ok := activeRuleset.SubGameModes[name]; ok {
							pivotGameModes[name] = struct{}{}
						}
					}

					partyCount := 0
					teamCount := len(matchingAllies)
					playerPerTeamCount := make([]int, teamCount)

					for teamIndex, ally := range matchingAllies {
						for _, party := range ally.MatchingParties {
							partyCount++
							playerPerTeamCount[teamIndex] += len(party.PartyMembers)

							if v, o := party.PartyAttributes[models.AttributeSubGameMode]; o {
								names, ok := v.([]interface{})
								if !ok {
									names = append(names, v)
								}
								for _, v := range names {
									n, ok := v.(string)
									if !ok {
										continue
									}
									if _, ok := pivotGameModes[n]; ok {
										gameModeCounts[n]++
									}
								}
							}
						}
					}

					// remove sub gamemode name that is not in all tickets
					for n, c := range gameModeCounts {
						if c == partyCount {
							selectedSubGamemodes = append(selectedSubGamemodes, n)
						}
					}

					// filter out sub gamemode that cannot be satisfied with current match
					filteredSubGameModes := make([]interface{}, 0)
					for _, nameVal := range selectedSubGamemodes {
						name, ok := nameVal.(string)
						if !ok {
							continue
						}
						subGameMode, ok := activeRuleset.SubGameModes[name]
						if !ok {
							continue
						}
						if teamCount < subGameMode.AllianceRule.MinNumber ||
							teamCount > subGameMode.AllianceRule.MaxNumber {
							continue
						}
						fit := true
						for i := 0; i < teamCount; i++ {
							if playerPerTeamCount[i] < subGameMode.AllianceRule.PlayerMinNumber ||
								playerPerTeamCount[i] > subGameMode.AllianceRule.PlayerMaxNumber {
								fit = false
								break
							}
						}
						if !fit {
							continue
						}
						filteredSubGameModes = append(filteredSubGameModes, name)
					}
					selectedSubGamemodes = filteredSubGameModes
				}
			}

			attributes := make(map[string]interface{}) // store combined party attributes into session

			prepareMatchingAlliesTimer.start()

			matchID := utils.GenerateUUID()

			currentPlatformMap := make(map[any]struct{})
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
							// merge current platform
							if key == models.AttributeCurrentPlatform && mm.useCurrentPlatform {
								switch v := val.(type) {
								case []any:
									for _, a := range v {
										currentPlatformMap[a] = struct{}{}
									}
								case any:
									currentPlatformMap[v] = struct{}{}
								case string:
									currentPlatformMap[v] = struct{}{}
								default:
									logrus.WithField("type", fmt.Sprintf("%T", v)).WithField("value", v).Error("unknown current_platform type")
								}
								continue
							}

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
			prepareMatchingAlliesTimer.end()
			prepareMatchingAlliesTimer.appendElapsed()

			if mm.useCurrentPlatform && len(currentPlatformMap) > 0 {
				var currentPlatform []any
				for platform := range currentPlatformMap {
					currentPlatform = append(currentPlatform, platform)
				}
				attributes[models.AttributeCurrentPlatform] = currentPlatform
			}

			// put selected subgamemodes in the session's attribute
			if len(selectedSubGamemodes) > 0 {
				attributes[models.AttributeSubGameMode] = selectedSubGamemodes
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
	regionLoopTimer.end()
	regionLoopTimer.appendElapsed()

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

		/* // [BLEVE] remove bleve index
		removed := matchmakingRequests[removedID]
		*/

		switch removedID {
		case 0:
			matchmakingRequests = matchmakingRequests[removedID+1:]
		case len(matchmakingRequests):
			matchmakingRequests = matchmakingRequests[:removedID-1]
		default:
			matchmakingRequests = append(matchmakingRequests[:removedID], matchmakingRequests[removedID+1:]...)
		}
		// if !needRequestRotation {

		/* // [BLEVE] remove bleve index
		err = index.Delete(removed.PartyID)
		if err != nil {
			scope.Log.Error("unable to remove request from index: ", err)
		}
		*/

		// } else {
		// 	// in case the pivot comes from the re-matchmaking ticket
		// 	// set first ticket created time to last created time to
		// 	// prevent the same request become pivot request again
		// 	removed.FirstTicketCreatedAt = removed.CreatedAt
		// 	matchmakingRequests = append(matchmakingRequests, removed)
		// }

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

type timer struct {
	reorderTicketsTimer elapsedTimer
	step1Timer          elapsedTimer
	step2Timer          elapsedTimer
	step3Timer          elapsedTimer
	rebalanceTimer      elapsedTimer
	validateAlliesTimer elapsedTimer
	loopCounter         int
}

func findMatchingAlly(
	rootScope *envelope.Scope,
	config *config.Config,
	sourceTickets []models.MatchmakingRequest,
	pivotTicket models.MatchmakingRequest,
	allianceRule models.AllianceRule,
	rebalanceVersion int,
	matchingRules []models.MatchingRule,
	timer *timer,
	blockedPlayerOption models.BlockedPlayerOption,
) ([]models.MatchingAlly, []models.MatchmakingRequest) {
	scope := rootScope.NewChildScope("findMatchingAlly")
	defer scope.Finish()

	// get pivot index
	pivotIndex := getPivotTicketIndexFromTickets(sourceTickets, &pivotTicket)
	elementsAlwaysFirst := []int{pivotIndex}

	// set up reorder tool
	maxLoop := 1
	if allianceRule.HasCombination {
		// role-based need more loop to test out combination for each roles
		// but can be overwritten by config FindAllyMaxLoop
		maxLoop = allianceRule.MaxNumber * allianceRule.PlayerMaxNumber
	}
	if config != nil && config.FindAllyMaxLoop > 0 {
		maxLoop = config.FindAllyMaxLoop
	}
	r := reordertool.NewOnePointerByLength(len(sourceTickets))
	r.SetOptions(reordertool.Options{
		MaxLoop:             maxLoop,
		ElementsAlwaysFirst: elementsAlwaysFirst,
	})

	for r.HasNext() {
		timer.loopCounter++

		timer.reorderTicketsTimer.start()
		newIndexes := r.Get()
		tickets := reorderTickets(sourceTickets, newIndexes)
		timer.reorderTicketsTimer.end()
		timer.reorderTicketsTimer.appendElapsed()

		// use alliance rule's max for normal, all unique, & 1 combo rules
		maxAllyCount := allianceRule.MaxNumber
		minAllyCount := allianceRule.MinNumber
		isMultiCombo := false

		if allianceRule.IsMultiComboRoleBased() {
			// use multi-combo's alliance rule
			maxAllyCount = len(allianceRule.Alliances)
			minAllyCount = maxAllyCount
			isMultiCombo = true
		}

		var ticketsPerTeam [][]models.MatchmakingRequest

		// step 1: create a match with min team & min players
		timer.step1Timer.start()
		for i := 0; i < minAllyCount; i++ {
			matchedTickets := FindPartyCombination(
				config,
				tickets,
				pivotTicket,
				allianceRule.PlayerMinNumber,
				allianceRule.PlayerMinNumber,
				allianceRule.HasCombination,
				allianceRule.GetRoles(i),
				nil,
				blockedPlayerOption,
			)

			if len(matchedTickets) == 0 {
				if isMultiCombo {
					// multi-combo might allow empty alliance
					totalMin := 0
					for _, role := range allianceRule.GetRoles(i) {
						totalMin += role.Min
					}
					if totalMin > 0 {
						break
					}
					ticketsPerTeam = append(ticketsPerTeam, []models.MatchmakingRequest{})
					continue
				}
				break
			}

			ticketsPerTeam = append(ticketsPerTeam, matchedTickets)

			for _, reqComb := range matchedTickets {
				tickets = removeMatchmakingRequest(reqComb.PartyID, tickets)
			}
		}
		timer.step1Timer.end()
		timer.step1Timer.appendElapsed()

		// step 2: fill match up to max team & max players
		timer.step2Timer.start()
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
				allianceRule.HasCombination,
				allianceRule.GetRoles(i),
				curTeamTickets,
				blockedPlayerOption,
			)

			if len(matchedTickets) == 0 {
				if isMultiCombo {
					// multi-combo might allow empty alliance
					totalMin := 0
					for _, role := range allianceRule.GetRoles(i) {
						totalMin += role.Min
					}
					if totalMin > 0 {
						break
					}
					continue
				}
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
		timer.step2Timer.end()
		timer.step2Timer.appendElapsed()

		// step 3: convert matching tickets to matching allies
		timer.step3Timer.start()
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
		timer.step3Timer.end()
		timer.step3Timer.appendElapsed()

		rebalanceEnable := rebalanceVersion != models.RebalanceDisabled
		if rebalanceEnable {
			timer.rebalanceTimer.start()

			if rebalanceVersion == models.RebalanceV1 {
				// distribute or rebalance member count, except for asymmetry
				teams = rebalance_v1.RebalanceMemberCount(scope, teams, allianceRule, blockedPlayerOption)

				// try to rebalance, swap between ally
				teams = rebalance_v1.Rebalance(scope, "", teams, allianceRule, matchingRules)
			} else {
				// default to v2
				teams = rebalance_v2.RebalanceV2(scope, "", teams, allianceRule, matchingRules, blockedPlayerOption)
			}

			timer.rebalanceTimer.end()
			timer.rebalanceTimer.appendElapsed()
		}

		// check if these alliances can be used to fill a session
		timer.validateAlliesTimer.start()
		if err := allianceRule.ValidateAllies(teams, blockedPlayerOption); err != nil {
			continue
		}
		timer.validateAlliesTimer.end()
		timer.validateAlliesTimer.appendElapsed()
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
	hasCombination bool,
	roles []models.Role,
	current []models.MatchmakingRequest,
	blockedPlayerOption models.BlockedPlayerOption,
) []models.MatchmakingRequest {
	// define the partyFinder
	pf := GetPartyFinder(hasCombination, roles, minPlayer, maxPlayer, current)

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
	if hasCombination {
		// role-based need more loop to test out combination for each roles
		// but can be overwritten by config FindPartyMaxLoop
		maxLoop = 1000
	}
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

		// start assigning role with new combination in each loop
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

func generateQueryString(ruleset models.RuleSet, channel *models.Channel, ticket *models.MatchmakingRequest, regionIndex int) (query string) {
	query += generateDistanceQuery(ruleset, ticket.PartyAttributes)
	query += generateSubGameModeQuery(ruleset, ticket.PartyAttributes)
	query += generateMatchOptionsAndPartyAttributesQuery(ruleset, ticket.PartyAttributes)

	// match all server_name when the attribute is empty
	if ticket.PartyAttributes[models.AttributeServerName] == nil || ticket.PartyAttributes[models.AttributeServerName] == "" {
		query += fmt.Sprintf(`-%s.server_name:* `, partyAttributesKey)
	}
	// match all client_version when the attribute is empty
	if ticket.PartyAttributes[models.AttributeClientVersion] == nil || ticket.PartyAttributes[models.AttributeClientVersion] == "" {
		query += fmt.Sprintf(`-%s.client_version:* `, partyAttributesKey)
	}

	// match based on region latency
	if len(ticket.SortedLatency) > 0 && regionIndex < len(ticket.SortedLatency) {
		latency := getTicketMaxLatency(ticket, channel)
		query += fmt.Sprintf("+%s.%s:<=%d ", latencyMapKey, ticket.SortedLatency[regionIndex].Region, latency)
	}

	// match based on additional criteria
	for criteria, val := range ticket.AdditionalCriterias {
		query += fmt.Sprintf(`additional_criteria.%s:"%v" `, criteria, val)
	}

	// avoid match with same party ID
	query += fmt.Sprintf("-party_id:%s ", ticket.PartyID)

	// avoid match with same user ID
	for _, member := range ticket.PartyMembers {
		query += fmt.Sprintf("-party_members.user_id:%s ", member.UserID)
	}

	return query
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

func generateDistanceQuery(ruleset models.RuleSet, partyAttributes map[string]interface{}) (query string) {
	memberAttributes, ok := partyAttributes[memberAttributesKey].(map[string]interface{})
	if !ok {
		return ""
	}
	for _, rule := range ruleset.MatchingRule {
		if rule.Criteria == distanceCriteria {
			value, ok := memberAttributes[rule.Attribute].(float64)
			if !ok {
				continue
			}
			query = fmt.Sprintf("+%s.%s.%s:>=%v ", partyAttributesKey, memberAttributesKey, rule.Attribute, value-rule.Reference)
			query += fmt.Sprintf("+%s.%s.%s:<=%v ", partyAttributesKey, memberAttributesKey, rule.Attribute, value+rule.Reference)
		}
	}
	return query
}

func generateSubGameModeQuery(ruleset models.RuleSet, partyAttributes map[string]interface{}) (query string) {
	gameModeMap := make(map[string]struct{})
	for _, subGameMode := range ruleset.SubGameModes {
		gameModeMap[subGameMode.Name] = struct{}{}
	}

	subGameModeVal, ok := partyAttributes[models.AttributeSubGameMode]
	if !ok {
		return ""
	}

	var subGameModeNames []interface{}
	subGameModeNames, ok = subGameModeVal.([]interface{})
	if !ok {
		subGameModeNames = append(subGameModeNames, subGameModeVal)
	}

	for _, v := range subGameModeNames {
		name, ok := v.(string)
		if !ok {
			continue
		}
		if _, ok = gameModeMap[name]; !ok {
			continue
		}
		query += fmt.Sprintf(`%s.%s:"%s" `, partyAttributesKey, models.AttributeSubGameMode, name)
	}
	return query
}

func generateMatchOptionsAndPartyAttributesQuery(ruleset models.RuleSet, partyAttributes map[string]interface{}) (query string) {
	// apply match options
	optionsMap := make(map[string]struct{})
	for _, option := range ruleset.MatchOptions.Options {
		optionsMap[option.Name] = struct{}{}

		optVal, ok := partyAttributes[option.Name]
		if !ok {
			continue
		}

		multiVal, ok := optVal.([]interface{})
		if !ok {
			multiVal = append(multiVal, optVal)
		}

		if option.Type != models.MatchOptionTypeDisable {
			query += fmt.Sprintf(`+%s.%s:* `, partyAttributesKey, option.Name)
		}

		if option.Type == models.MatchOptionTypeAny {
			var values string
			for i, v := range multiVal {
				if v == "" {
					continue
				}
				if i > 0 {
					values += "|"
				}
				values += fmt.Sprintf("%s", v)
			}
			query += fmt.Sprintf(`+%s.%s:/(%s)/ `, partyAttributesKey, option.Name, strings.ToLower(values))
		} else {
			for _, v := range multiVal {
				switch option.Type {
				case models.MatchOptionTypeAll:
					query += fmt.Sprintf(`+%s.%s:"%s" `, partyAttributesKey, option.Name, v)
				case models.MatchOptionTypeUnique:
					query += fmt.Sprintf(`-%s.%s:"%s" `, partyAttributesKey, option.Name, v)
				}
			}
		}
	}

	// matching party attributes
	for key, value := range partyAttributes {
		// skip options
		if _, ok := optionsMap[key]; ok {
			continue
		}

		switch key {
		// ignoring these keys
		case models.AttributeMatchAttempt:
		case models.AttributeLatencies:
		case models.AttributeMemberAttr:
		case models.AttributeSubGameMode:
		case models.AttributeBlockedPlayersDetail:
		case models.AttributeNewSessionOnly:

		// handle blocked players
		case models.AttributeBlocked:
			ids, ok := value.([]interface{})
			if !ok {
				continue
			}

			for _, id := range ids {
				if idStr, o := id.(string); o {
					query += fmt.Sprintf(`-%s.%s:%s `, membersKey, userIDKey, idStr)
				}
			}

		default:
			// store these keys as "must match this attribute"
			if value != "" {
				query += fmt.Sprintf(`+%s.%s:"%v" `, partyAttributesKey, key, value)
			}
		}
	}
	return query
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

func convertToMapInterface(maps map[string]time.Duration) map[string]interface{} {
	m := make(map[string]interface{}, len(maps))
	for k, v := range maps {
		m[k] = v
	}
	return m
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
