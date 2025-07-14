// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"

	"github.com/elliotchance/pie/v2"
)

// applyRoleBasedFlexing make role based more lenient, for x seconds, x player's role changed to "any",
// when the channel is using sub game mode and has more than 1 sub game mode, we choose the combination based on higher number of changes to "any",
// for example: we have a channel enable sub game mode with 2 sub game modes
// - sub game mode A with role flexing 60sec 1 player
// - sub game mode B with role flexing 30sec 1 player
// sub game mode B is chosen, because in 60sec it will change 2 players, where A only change 1 player
func applyRoleBasedFlexing(matchmakingRequests []models.MatchmakingRequest, channel *models.Channel) {
	var combination models.Combination

	// get the role combination from available alliance rules in the channel
	allianceRules := channel.GetAllianceRules()
	if len(allianceRules) == 1 {
		combination = allianceRules[0].Combination
	} else {
		// this must be using sub game mode and it has more than 1 sub game mode
		// let's choose 1 combination with higher number of changes
		var highestChangesPerMinute float64
		for _, allianceRule := range allianceRules {
			if allianceRule.RoleFlexingSecond == 0 {
				continue
			}

			changesPerMinute := (time.Minute.Seconds() / float64(allianceRule.RoleFlexingSecond)) * float64(allianceRule.RoleFlexingPlayer)

			if changesPerMinute > highestChangesPerMinute {
				highestChangesPerMinute = changesPerMinute
				combination = allianceRule.Combination
			}
		}
	}

	// skip for non role based
	if !combination.HasCombination {
		return
	}

	// skip if not enable
	if !combination.RoleFlexingEnable {
		return
	}

	var (
		now       = time.Now().UTC()
		xDuration = time.Duration(combination.RoleFlexingSecond) * time.Second
		xPlayer   = combination.RoleFlexingPlayer
	)

	var oldestRequest models.MatchmakingRequest
	var isNeedFlex bool
	for _, request := range matchmakingRequests {
		firstCreatedAt := time.Unix(request.CreatedAt, 0).UTC()
		diffDuration := now.Sub(firstCreatedAt)

		// skip this request if not pass xSecond
		if diffDuration < xDuration {
			continue
		}

		if oldestRequest.CreatedAt == 0 {
			oldestRequest = request
		} else if request.CreatedAt < oldestRequest.CreatedAt {
			oldestRequest = request
		}
		isNeedFlex = true
	}

	// no request need flexing
	if !isNeedFlex {
		return
	}

	// count how many player should change to any
	firstCreatedAt := time.Unix(oldestRequest.CreatedAt, 0).UTC()
	diffDuration := now.Sub(firstCreatedAt)
	countPlayerChangeToAny := int(diffDuration.Seconds()/xDuration.Seconds()) * xPlayer

	// no changes yet
	if countPlayerChangeToAny == 0 {
		return
	}

	var currentTotalAny int
	for _, request := range matchmakingRequests {
		currentTotalAny += request.CountPlayerByRole(models.AnyRole)
	}
	diffCountAny := countPlayerChangeToAny - currentTotalAny

	// no more role to change
	if diffCountAny <= 0 {
		return
	}

	// [AR-7192] choose which player should we change to any

	numPlayersForRole := getNumPlayersForRole(matchmakingRequests)
	numMinimumForRole := getNumMinimumForRole(combination)

	// count the number of deficiency of each role
	numDeficitForRole := make(map[string]int, 0)
	for role, min := range numMinimumForRole {
		numDeficitForRole[role] = min - numPlayersForRole[role]
	}

	// sort roles based on the number of deficiency in ASC order
	orderedRolesToBeConvertedToAny := pie.SortUsing(extractRoles(combination), func(role1, role2 string) bool {
		return numDeficitForRole[role1] < numDeficitForRole[role2]
	})

	// sort requests by newest age
	requestsOrderByNewest := pie.SortUsing(matchmakingRequests, func(request1, request2 models.MatchmakingRequest) bool {
		return request1.CreatedAt > request2.CreatedAt
	})

	// assign "any" role to players with newest age and whose role is high demand first
	memberAssignedWithAnyRole := make(map[string]struct{})
outerLoop:
	for _, role := range orderedRolesToBeConvertedToAny {
		for _, request := range requestsOrderByNewest {
			for _, member := range request.PartyMembers {
				// skip this member because has no required role to be changed in this iteration
				if !utils.Contains(member.GetRole(), role) {
					continue
				}
				// skip this member because the role already changed to "any"
				if utils.Contains(member.GetRole(), models.AnyRole) {
					continue
				}
				// assign role "any"
				memberAssignedWithAnyRole[member.UserID] = struct{}{}
				diffCountAny--
				if diffCountAny <= 0 {
					break outerLoop
				}
			}
		}
	}

	// assign any role to members
	for i, request := range matchmakingRequests {
		for j, member := range request.PartyMembers {
			if _, ok := memberAssignedWithAnyRole[member.UserID]; !ok {
				continue
			}
			matchmakingRequests[i].PartyMembers[j].SetRole(models.AnyRole)
		}
	}
}

func getNumPlayersForRole(requests []models.MatchmakingRequest) map[string]int {
	numPlayerInRole := make(map[string]int, 0)
	for _, request := range requests {
		for _, member := range request.PartyMembers {
			for _, role := range member.GetRole() {
				numPlayerInRole[role]++
			}
		}
	}
	return numPlayerInRole
}

func getNumMinimumForRole(combination models.Combination) map[string]int {
	numMinimumForRole := make(map[string]int, 0)
	for _, roles := range combination.Alliances {
		for _, role := range roles {
			numMinimumForRole[role.Name] += role.Min
		}
	}
	return numMinimumForRole
}

func extractRoles(combination models.Combination) []string {
	allRoles := make([]string, 0)
	for _, roles := range combination.Alliances {
		for _, role := range roles {
			allRoles = append(allRoles, role.Name)
		}
	}
	return pie.Unique(allRoles)
}
