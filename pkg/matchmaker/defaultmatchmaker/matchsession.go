// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"slices"
	"strings"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance/rebalance_v2"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
	"github.com/elliotchance/pie/v2"
	"github.com/sirupsen/logrus"
)

// TBD: rebalance for match session use different logic,
// while adding ticket to a session, we should check the mmr balance

//nolint:gocyclo
func (mm *MatchMaker) MatchSessions(rootScope *envelope.Scope, namespace string, matchPool string, tickets []models.MatchmakingRequest, sessions []*models.MatchmakingResult, channel models.Channel) (updatedSessions []*models.MatchmakingResult, satisfiedSessions []*models.MatchmakingResult, satisfiedTickets []models.MatchmakingRequest, err error) {
	scope := rootScope.NewChildScope("Matchmaker.MatchSessions")
	defer scope.Finish()

	if len(tickets) == 0 || len(sessions) == 0 {
		return nil, nil, nil, nil
	}

	// [REBALANCE_BACKFILL]
	// lock parties in current sessions
	rebalanceEnable := channel.Ruleset.GetRebalanceMode() != models.RebalanceDisabled
	if rebalanceEnable {
		for _, session := range sessions {
			session.LockParties()
		}
	}

	var (
		matchSessionsTimer         elapsedTimer
		applyFlexingTimer          elapsedTimer
		filterOptionsTimer         elapsedTimer
		findMatchingAllyTimer      elapsedTimer
		updatePartyAttributesTimer elapsedTimer
		updateSessionsTimer        elapsedTimer
	)
	matchSessionsTimer.start()

	defer func() {
		matchSessionsTimer.end()

		elapsedTimeMaps := map[string]time.Duration{
			"totalMatchSessions":    matchSessionsTimer.elapsed(),
			"applyFlexing":          applyFlexingTimer.totalElapsed(),
			"filterOptions":         filterOptionsTimer.totalElapsed(),
			"findMatchingAlly":      findMatchingAllyTimer.totalElapsed(),
			"updatePartyAttributes": updatePartyAttributesTimer.totalElapsed(),
			"updateSessions":        updateSessionsTimer.totalElapsed(),
		}

		l := logrus.WithFields(convertToMapInterface(elapsedTimeMaps))
		l.WithFields(logrus.Fields{
			"matchPool": matchPool,
		}).Info("Matchmaker Match Sessions")

		// send elapsed time metric
		for k, v := range elapsedTimeMaps {
			if mm.metrics != nil {
				mm.metrics.AddMatchSessionsElapsedTimeMs(namespace, matchPool, k, v)
			}
		}
	}()

	// pool lock timeout safeguard
	startTime := time.Now()
	timeLimit := (constants.PoolLockTimeLimit * 2) / 5
	if mm.cfg != nil && mm.cfg.MatchTimeLimitSecond > 0 {
		timeLimit = time.Duration(mm.cfg.MatchTimeLimitSecond) * time.Second
	}
	satisfiedTickets = make([]models.MatchmakingRequest, 0)
	updatedSessions = make([]*models.MatchmakingResult, 0)
	satisfiedSessions = make([]*models.MatchmakingResult, 0)

	// prioritize request with more players
	if mm.cfg != nil && mm.cfg.PrioritizeLargerParties {
		sortDESC(tickets)
	}

	// for each session
allsession:
	for _, session := range sessions {
		// determine if rule needs flexing
		applyFlexingTimer.start()
		activeRuleset, _ := applyRuleFlexingForSession(*session, channel.Ruleset)
		activeRuleset, _ = applyAllianceFlexingRulesForSession(*session, activeRuleset)
		scope.Log.WithField("ruleset", activeRuleset).Debug("ruleset applied")

		// role-based flexing
		applyRoleBasedFlexing(tickets, &channel)
		applyFlexingTimer.end()
		applyFlexingTimer.appendElapsed()

		// [MANUALSEARCH]
		result, unBackfilledReasons := mm.SearchMatchTicketsBySession(scope, &channel.Ruleset, &activeRuleset, &channel, *session, tickets)
		if len(unBackfilledReasons) > 0 {
			// send data to log
			scope.Log.WithField("unbackfilled_reasons", unBackfilledReasons).
				WithField("match_id", session.MatchID).
				Debugf("unable to backfill to sessions %v", session.MatchID)

			for _, candidateTicket := range tickets {
				if reason, ok := unBackfilledReasons[candidateTicket.PartyID]; ok {
					// add metrics for unbackfilled
					mm.addUnmatchedReasonMetric(namespace, matchPool, reason)
				}
			}
		}

	tickethitloop:

		// [MANUALSEARCH]
		for resultIndex := range result {
			candidateTicket := &result[resultIndex]

			// check if still have time to try
			elapsed := time.Since(startTime)
			if elapsed >= timeLimit {
				break allsession
			}

			// prevent duplicate userid match into same session
			sessionUserIDs := session.GetMapUserIDs()
			for _, member := range candidateTicket.PartyMembers {
				if _, exist := sessionUserIDs[member.UserID]; exist {
					continue tickethitloop
				}
			}

			// validate region latency in 3 steps:
			sessionRegion := strings.TrimSpace(session.Region)
			if sessionRegion != "" {
				// just to re-ensure candidate ticket's region is same with session region
				filteredRegions := filterRegionByStep(candidateTicket, &channel)
				var isRegionMatch bool
				for _, region := range filteredRegions {
					if region.Region == session.Region {
						isRegionMatch = true
						break
					}
				}
				// skip ticket if somehow its not match
				if !isRegionMatch {
					scope.Log.WithField("match_id", session.MatchID).
						WithField("channel", session.Channel).
						WithField("candidate_party_id", candidateTicket.PartyID).
						Warn("region is not match")

					continue
				}
			} else {
				// if session region is empty just log warn
				scope.Log.WithField("match_id", session.MatchID).
					WithField("channel", session.Channel).
					Warn("session region is empty")
			}

			// update list of blocked player in session
			toBeUpdated := make([]*models.MatchmakingResult, 0, len(updatedSessions)+len(satisfiedSessions))
			toBeUpdated = append(toBeUpdated, satisfiedSessions...)
			toBeUpdated = append(toBeUpdated, updatedSessions...)
			for _, tbu := range toBeUpdated {
				updateBlockedPlayerInSession(tbu)
			}

			/*
				[AR-7033] skip checking blocked players for:
				- respect block only for the same team
				- don't respect block
			*/
			if channel.Ruleset.BlockedPlayerOption == "" ||
				channel.Ruleset.BlockedPlayerOption == models.BlockedPlayerCannotMatch {
				// check if any players in session is blocked by anyone in the ticket (use ticket's PartyAttribute root level)
				for _, blockedUserID := range candidateTicket.GetBlockedPlayerUserIDs() {
					for _, userID := range session.GetMemberUserIDs() {
						if userID == blockedUserID {
							continue tickethitloop
						}
					}
				}

				// check if any players in ticket is blocked by anyone in the session (use session's PartyAttribute root level)
				for _, blockedUserID := range session.GetBlockedPlayerUserIDs() {
					for _, userID := range candidateTicket.GetMemberUserIDs() {
						if userID == blockedUserID {
							continue tickethitloop
						}
					}
				}
			}

			// filter based on optional match, skip if does not make sense
			filterOptionsTimer.start()
			optionValuesMap := make(map[string]map[interface{}]int)
			ruleOptions := make(map[string]models.MatchOption)
			isMultiOptions := make(map[string]bool)
			selectedOptions := make(map[string][]interface{})
			replaceOptions := make(map[string]bool)

			// count the number of times the options and its values are found in session's combined party attributes
			for _, option := range activeRuleset.MatchOptions.Options {
				ruleOptions[option.Name] = option

				// include the session's attribute in options count
				if v, o := session.PartyAttributes[option.Name]; o {
					if optionValuesMap[option.Name] == nil {
						optionValuesMap[option.Name] = make(map[interface{}]int)
					}

					multival, ok := v.([]interface{})
					isMultiOptions[option.Name] = ok
					if !ok {
						// handle single value
						optionValuesMap[option.Name][v]++
					} else {
						for _, val := range multival {
							optionValuesMap[option.Name][val]++
						}
					}
				}

				// read candidate ticket's attribute
				if v, o := candidateTicket.PartyAttributes[option.Name]; o {
					if optionValuesMap[option.Name] == nil {
						optionValuesMap[option.Name] = make(map[interface{}]int)
					}

					multival, ok := v.([]interface{})
					if !ok {
						// handle single value
						optionValuesMap[option.Name][v]++
					} else {
						for _, val := range multival {
							optionValuesMap[option.Name][val]++
						}
					}
				}
			}

			anyCrossPlay := false
			for name, option := range optionValuesMap {
				switch ruleOptions[name].Type {
				case models.MatchOptionTypeAll:
					// fail if any party in the session does not have all options
					for val, count := range option {
						if count < 2 { // must match the ticket and the session = 2
							continue tickethitloop
						}
						selectedOptions[name] = append(selectedOptions[name], val)
					}
				case models.MatchOptionTypeAny:
					// fail if cannot find common option
					for val, count := range option {
						if count > 1 {
							selectedOptions[name] = append(selectedOptions[name], val)
						}
					}

					if mm.isMatchAnyCommon {
						// replace with all parties common value
						replaceOptions[name] = true
					}

					if name == models.AttributeCrossPlatform {
						anyCrossPlay = true
					}

					if len(selectedOptions) == 0 {
						continue tickethitloop
					}
				case models.MatchOptionTypeUnique:
					// fail if there's any common option
					for val, count := range option {
						if count > 1 {
							continue tickethitloop
						}
						selectedOptions[name] = append(selectedOptions[name], val)
					}
				}
			}
			filterOptionsTimer.end()
			filterOptionsTimer.appendElapsed()

			// filter based on session's subgamemode
			var selectedSubGamemodeNames []string
			if activeRuleset.IsUseSubGamemode() {
				// get intersection between session's sub game mode and ticket's sub game mode
				intersectedSubGameModeNames := utils.IntersectionOfStringLists(
					session.GetSubGameModeNames(),
					candidateTicket.GetSubGameModeNames(),
				)

				// loop the intersected sub game mode, append to selected if it exist in active rule set
				for _, subGameModeName := range intersectedSubGameModeNames {
					if _, ok := activeRuleset.SubGameModes[subGameModeName]; ok {
						selectedSubGamemodeNames = append(selectedSubGamemodeNames, subGameModeName)
					}
				}

				// skip ticket if it does not have any same subgamemode
				if len(selectedSubGamemodeNames) == 0 {
					continue tickethitloop
				}
			}

			// find proper alliance for ticket
			findMatchingAllyTimer.start()
			teamCount := len(session.MatchingAllies)
			playerPerTeamCount := make([]int, teamCount)
			originalSessionPlayerCount := 0
			ticketPlayerCount := len(candidateTicket.PartyMembers)
			{
				// try for all possible subgamemode
				var allianceRules []models.AllianceRule

				if activeRuleset.IsUseSubGamemode() {
					for _, name := range selectedSubGamemodeNames {
						r, ok := activeRuleset.SubGameModes[name]
						if !ok {
							continue
						}
						allianceRules = append(allianceRules, r.AllianceRule)
					}
				} else {
					allianceRules = append(allianceRules, activeRuleset.AllianceRule)
				}

				found := false

				for _, allianceRule := range allianceRules {
					for allyIndex, ally := range session.MatchingAllies {
						playerPerTeamCount[allyIndex] = ally.CountPlayer()
						originalSessionPlayerCount += ally.CountPlayer()
					}

				findMatchingAlly:
					for allyIndex, ally := range session.MatchingAllies {
						// prepare PartyFinder params
						hasCombination := allianceRule.HasCombination
						roles := allianceRule.GetRoles(allyIndex)
						minPlayer := allianceRule.PlayerMinNumber
						maxPlayer := allianceRule.PlayerMaxNumber
						current := []models.MatchmakingRequest{
							// PartyFinder only need the party members to find a party
							{PartyMembers: ally.GetMembers()},
						}

						// use PartyFinder to assign members
						pf := GetPartyFinder(hasCombination, roles, minPlayer, maxPlayer, current)
						/*
							[AR-7033] check blocked players for:
							- respect block only for the same team
						*/
						if channel.Ruleset.BlockedPlayerOption == models.BlockedPlayerCanMatchOnDifferentTeam &&
							isContainBlockedPlayers(pf.GetCurrentResult(), candidateTicket) {
							continue
						}
						success := pf.AssignMembers(*candidateTicket)
						if !success {
							continue
						}
						pf.AppendResult(*candidateTicket)
						for _, res := range pf.GetCurrentResult() {
							if res.PartyID == candidateTicket.PartyID {
								// to copy all member's extra attributes
								candidateTicket.PartyMembers = res.PartyMembers
								break
							}
						}

						found = true
						session.MatchingAllies[allyIndex].MatchingParties = append(session.MatchingAllies[allyIndex].MatchingParties, createMatchingParty(candidateTicket))
						playerPerTeamCount[allyIndex] += ticketPlayerCount
						break
					}

					if found {
						break
					}

					// try creating a new alliance
					if len(session.MatchingAllies) < allianceRule.MaxNumber {
						session.MatchingAllies = append(session.MatchingAllies, models.MatchingAlly{
							MatchingParties: []models.MatchingParty{},
						})
						teamCount++
						playerPerTeamCount = append(playerPerTeamCount, 0)
						goto findMatchingAlly
					}

					if found {
						break
					}
				}

				session.MatchingAllies = rebalance_v2.RemoveEmptyMatchingParties(session.MatchingAllies)

				if !found {
					continue
				}

				// filter out subgamemodes that does not fulfill the session composition
				if activeRuleset.IsUseSubGamemode() {
					filteredSubGameModeNames := make([]string, 0)
					for _, name := range selectedSubGamemodeNames {
						subGameMode, ok := activeRuleset.SubGameModes[name]
						if !ok {
							continue
						}
						if teamCount < subGameMode.AllianceRule.MinNumber ||
							teamCount > subGameMode.AllianceRule.MaxNumber {
							continue
						}
						fit := true
						for t := 0; t < teamCount; t++ {
							if playerPerTeamCount[t] < subGameMode.AllianceRule.PlayerMinNumber ||
								playerPerTeamCount[t] > subGameMode.AllianceRule.PlayerMaxNumber {
								fit = false
								break
							}
						}
						if !fit {
							continue
						}
						filteredSubGameModeNames = append(filteredSubGameModeNames, name)
					}
					selectedSubGamemodeNames = filteredSubGameModeNames
				}
			} // find ally end
			findMatchingAllyTimer.end()
			findMatchingAllyTimer.appendElapsed()

			// update combined party attributes
			updatePartyAttributesTimer.start()
			{
				// update match options, insert new if any
				for key, values := range selectedOptions {
					if _, ok := session.PartyAttributes[key]; !ok {
						session.PartyAttributes[key] = values
					} else {
						if replaceOptions[key] {
							session.PartyAttributes[key] = values
						} else if attr, k := session.PartyAttributes[key]; k {
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
								session.PartyAttributes[key] = arr
							} else {
								session.PartyAttributes[key] = values
							}
						} else {
							session.PartyAttributes[key] = values
						}
					}
				}

				// kept original attributes, set to single value if the original is not an array
				for key, value := range session.PartyAttributes {
					if isMulti, exists := isMultiOptions[key]; exists && !isMulti {
						if values, ok := value.([]interface{}); ok {
							if len(values) == 1 {
								session.PartyAttributes[key] = values[0]
							}
						}
					}
				}

				// update subgamemodes list
				// should be in format of []interface{}
				if len(selectedSubGamemodeNames) > 0 {
					var values []interface{}
					for _, v := range selectedSubGamemodeNames {
						values = append(values, v)
					}
					session.PartyAttributes[models.AttributeSubGameMode] = values
				}

				// update member attributes
				sessionMemberAttributes, ok := session.PartyAttributes[memberAttributesKey].(map[string]interface{})
				if !ok {
					sessionMemberAttributes = make(map[string]interface{})
				}
				ticketMemberAttributes, ok := candidateTicket.PartyAttributes[memberAttributesKey].(map[string]interface{})
				if !ok {
					ticketMemberAttributes = make(map[string]interface{})
				}
				for _, rule := range activeRuleset.MatchingRule {
					if rule.Criteria == distanceCriteria {
						currentAvg, ok := sessionMemberAttributes[rule.Attribute].(float64)
						if !ok {
							currentAvg = 0
						}
						ticketAvg, ok := ticketMemberAttributes[rule.Attribute].(float64)
						if !ok {
							ticketAvg = 0
						}
						newAvg := (float64(originalSessionPlayerCount)*currentAvg + float64(ticketPlayerCount)*ticketAvg) / (float64(originalSessionPlayerCount) + float64(ticketPlayerCount))
						sessionMemberAttributes[rule.Attribute] = newAvg
					}
				}
				session.PartyAttributes[memberAttributesKey] = sessionMemberAttributes

				if anyCrossPlay && mm.useCurrentPlatform {
					// merge current platform
					if currentPlatforms, ok := utils.GetMapValueAs[[]any](session.PartyAttributes, models.AttributeCurrentPlatform); ok {
						if ticketCurrentPlatforms, ok := utils.GetMapValueAs[[]any](candidateTicket.PartyAttributes, models.AttributeCurrentPlatform); ok {
							for _, platform := range ticketCurrentPlatforms {
								if !slices.Contains(currentPlatforms, platform) {
									currentPlatforms = append(currentPlatforms, platform)
								}
							}
						}
						session.PartyAttributes[models.AttributeCurrentPlatform] = currentPlatforms
					}
				}
			}
			updatePartyAttributesTimer.end()
			updatePartyAttributesTimer.appendElapsed()

			// check if session full, remove from session list to avoid adding more players
			updateSessionsTimer.start()
			{
				var allianceRules []models.AllianceRule
				if activeRuleset.IsUseSubGamemode() {
					for _, name := range selectedSubGamemodeNames {
						subGameMode, ok := activeRuleset.SubGameModes[name]
						if !ok {
							continue
						}
						allianceRules = append(allianceRules, subGameMode.AllianceRule)
					}
				} else {
					allianceRules = append(allianceRules, activeRuleset.AllianceRule)
				}

				// append ticket to satisfiedTickets
				satisfiedTickets = append(satisfiedTickets, *candidateTicket)

				tickets = removeMatchmakingRequest(candidateTicket.PartyID, tickets)

				full := false
				for _, allianceRule := range allianceRules {
					if teamCount == allianceRule.MaxNumber {
						full = true
						for teamIndex := 0; teamIndex < teamCount; teamIndex++ {
							if playerPerTeamCount[teamIndex] < allianceRule.PlayerMaxNumber {
								full = false
								break
							}
						}
					}

					if !full {
						break
					}
				}

				if full {
					id := -1
					for j, s := range sessions {
						if s.MatchID == session.MatchID {
							id = j
							break
						}
					}

					if id > -1 {
						sessions = append(sessions[0:id], sessions[id+1:]...)
					}

					// put session in list of satisfied sessions
					found := false
					for _, v := range satisfiedSessions {
						if v.MatchID == session.MatchID {
							found = true
							break
						}
					}

					if !found {
						satisfiedSessions = append(satisfiedSessions, session)
					}

					// remove from updated session if any
					id = -1
					for k, v := range updatedSessions {
						if v.MatchID == session.MatchID {
							id = k
							break
						}
					}

					if id > -1 {
						updatedSessions = append(updatedSessions[0:id], updatedSessions[id+1:]...)
					}

					continue allsession
				} else {
					// put session in list of updated sessions
					found := false
					for _, v := range updatedSessions {
						if v.MatchID == session.MatchID {
							found = true
							break
						}
					}

					if !found {
						updatedSessions = append(updatedSessions, session)
					}
					updateBlockedPlayerInSession(session)
				}
			} // remove full session end
			updateSessionsTimer.end()
			updateSessionsTimer.appendElapsed()
		} // query hits loop end
		// } // session's regions loop end
	} // sessions loop end

	// [REBALANCE_BACKFILL]
	// DO REBALANCE FOR EACH SESSION with its first combination
	if rebalanceEnable {
		for i, v := range updatedSessions {
			activeRuleset, _ := applyRuleFlexingForSession(*v, channel.Ruleset)
			activeRuleset, _ = applyAllianceFlexingRulesForSession(*v, activeRuleset)
			// ignore minimal requirement
			activeRuleset.AllianceRule.MinNumber = 0
			activeRuleset.AllianceRule.PlayerMinNumber = 0

			teams := v.MatchingAllies
			oldTeamIDs := pie.Map(teams, func(ally models.MatchingAlly) string { return ally.TeamID })

			// do rebalance
			teams = rebalance_v2.RebalanceV2(scope, v.MatchID, teams, activeRuleset.AllianceRule, channel.Ruleset.MatchingRule, channel.Ruleset.BlockedPlayerOption)

			//put back the team ID based on order
			for i := range oldTeamIDs {
				if len(teams) <= i {
					break
				}
				teams[i].TeamID = oldTeamIDs[i]
			}

			for j, s := range sessions {
				if s.MatchID != v.MatchID {
					continue
				}
				sessions[j].MatchingAllies = teams
				break
			}

			updatedSessions[i].MatchingAllies = teams
		}
		for i, v := range satisfiedSessions {
			activeRuleset, _ := applyRuleFlexingForSession(*v, channel.Ruleset)
			activeRuleset, _ = applyAllianceFlexingRulesForSession(*v, activeRuleset)

			teams := v.MatchingAllies
			oldTeamIDs := pie.Map(teams, func(ally models.MatchingAlly) string { return ally.TeamID })

			// do rebalance
			teams = rebalance_v2.RebalanceV2(scope, v.MatchID, teams, activeRuleset.AllianceRule, channel.Ruleset.MatchingRule, channel.Ruleset.BlockedPlayerOption)

			//put back the team ID based on order
			for i := range oldTeamIDs {
				if len(teams) <= i {
					break
				}
				teams[i].TeamID = oldTeamIDs[i]
			}

			for j, s := range sessions {
				if s.MatchID != v.MatchID {
					continue
				}
				sessions[j].MatchingAllies = teams
				break
			}

			satisfiedSessions[i].MatchingAllies = teams
		}
	}

	return updatedSessions, satisfiedSessions, satisfiedTickets, nil
}

func updateBlockedPlayerInSession(session *models.MatchmakingResult) {
	blockedPlayers := make([]interface{}, 0)
	for _, ally := range session.MatchingAllies {
		for _, party := range ally.MatchingParties {
			blockedPlayersInterface, ok := party.PartyAttributes[models.AttributeBlocked]
			if !ok {
				continue
			}
			blockedPlayersArr, okArr := blockedPlayersInterface.([]interface{})
			if !okArr {
				continue
			}
			blockedPlayers = append(blockedPlayers, blockedPlayersArr...)
		}
	}
	if session.PartyAttributes == nil {
		session.PartyAttributes = make(map[string]interface{})
	}
	session.PartyAttributes[models.AttributeBlocked] = blockedPlayers
}
