// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

type PartyFinder interface {
	AssignMembers(ticket models.MatchmakingRequest) (success bool)
	AppendResult(ticket models.MatchmakingRequest)
	IsFulfilled() bool
	Reset()
	GetCurrentResult() []models.MatchmakingRequest
	GetBestResult() []models.MatchmakingRequest
}

// GetPartyFinder return party finder implementations, we have 3 party finder:
// 1) newRoleBasedCombo() to find party for role-based with role combination (combo role-based),
// 2) newRoleBasedUnique() to find party for role-based without role combination (unique role-based),
// 3) newNormal() to find party for non role-based
func GetPartyFinder(hasCombination bool, roles []models.Role, playerMinNumber, playerMaxNumber int, current []models.MatchmakingRequest) (pf PartyFinder) {
	if hasCombination {
		if len(roles) > 0 {
			pf = newRoleBasedCombo(playerMinNumber, playerMaxNumber, roles, current)
		} else {
			pf = newRoleBasedUnique(playerMinNumber, playerMaxNumber, current)
		}
	} else {
		pf = newNormal(playerMinNumber, playerMaxNumber, current)
	}
	return pf
}
