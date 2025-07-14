// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v1

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

// RebalanceMemberCount will rebalance allies based on the member count only, and validate the output based on activeAllianceRule
// DO NOT USE parameter channel here to get alliance rule, we should get the active alliance rule to consider flex rule and sub game mode
func RebalanceMemberCount(rootScope *envelope.Scope, allies []models.MatchingAlly, activeAllianceRule models.AllianceRule, blockedPlayerOption models.BlockedPlayerOption) []models.MatchingAlly {
	scope := rootScope.NewChildScope("RebalanceMemberCount")
	defer scope.Finish()

	// skip rebalance member count for asymmetry rule set
	if activeAllianceRule.IsAsymmetry() {
		return allies
	}

	// check min, max player count from all given allies to count the diff
	var _min, _max int
	var partyCount int

	// we need to add empty alliance to make rebalance
	if len(allies) < activeAllianceRule.MinNumber {
		for i := 0; i < activeAllianceRule.MinNumber-len(allies); i++ {
			allies = append(allies, models.MatchingAlly{})
		}
	}

	for _, ally := range allies {
		if _min == 0 || ally.CountPlayer() < _min {
			_min = ally.CountPlayer()
		}
		if _max == 0 || ally.CountPlayer() > _max {
			_max = ally.CountPlayer()
		}
		partyCount += len(ally.MatchingParties)
	}

	// no need to rebalance if diff less than or equal 1
	if _max-_min <= 1 {
		return allies
	}

	/*
		here is the logic:
		- prepare new allies variable with the same length as previous one
		- put locked parties inside new allies
		- separate unlocked parties from the locked parties
		- load unlocked parties
		- idx will store id of the ally to be filled in
		- load the new allies inside each party,
			mark the ally id which has lowest player count into idx,
			but consider the ally rule set
		- append the party into ally[idx]
	*/
	newAllies := make([]models.MatchingAlly, len(allies))

	unlockedParties := make([]models.MatchingParty, 0, partyCount)
	for i, ally := range allies {
		for _, party := range ally.MatchingParties {
			if party.Locked {
				newAllies[i].MatchingParties = append(newAllies[i].MatchingParties, party)
			} else {
				unlockedParties = append(unlockedParties, party)
			}
		}
	}

	// sort parties by member count in DESC order
	SortPartiesByMemberCountDESC(unlockedParties)

	for _, party := range unlockedParties {
		idx := 0
		assigned := false
		for i, newAlly := range newAllies {
			tempAlly := newAlly
			tempAlly.MatchingParties = append(tempAlly.MatchingParties, party)

			// validate combination cannot exceed max
			if tempAlly.CountPlayer() > activeAllianceRule.PlayerMaxNumber {
				continue
			}
			if activeAllianceRule.HasCombination {
				var invalid bool
				mapRoleCount := make(map[string]int)
				for _, party := range tempAlly.MatchingParties {
					for _, member := range party.PartyMembers {
						if len(member.GetRole()) != 1 {
							scope.Log.Warnf("[rebalanceMemberCount] failed because userID %s has more or less than 1 assigned role", member.UserID)
							return allies
						}
						assignedRole := member.GetRole()[0]
						mapRoleCount[assignedRole]++
					}
				}
				for _, role := range activeAllianceRule.GetRoles(i) {
					if mapRoleCount[role.Name] > role.Max {
						invalid = true
						break
					}
				}
				if invalid {
					continue
				}
			}

			if idx == i || newAlly.CountPlayer() < newAllies[idx].CountPlayer() {
				idx = i
				assigned = true
			}
		}

		// failed rebalance member count, cannot find valid combination
		if !assigned {
			scope.Log.Warnf("[rebalanceMemberCount] failed because cannot find valid combination")
			return allies
		}

		newAllies[idx].MatchingParties = append(newAllies[idx].MatchingParties, party)
	}
	// validate all allies
	err := activeAllianceRule.ValidateAllies(newAllies, blockedPlayerOption)
	if err != nil {
		scope.Log.Warnf("[rebalanceMemberCount] failed because allies invalid: %v", err)
		return allies
	}
	return newAllies
}
