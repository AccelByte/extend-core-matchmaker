// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package defaultmatchmaker provides the default implementation of the MatchLogic interface.
// This package contains the core matchmaking algorithms and logic for creating matches from tickets.
package defaultmatchmaker

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

// PartyFinder defines the interface for finding and managing party combinations during matchmaking.
// This interface is used to determine optimal party groupings for different matchmaking scenarios.
type PartyFinder interface {
	// AssignMembers attempts to assign a ticket to the current party combination.
	// Returns true if the assignment was successful, false otherwise.
	AssignMembers(ticket models.MatchmakingRequest) (success bool)

	// AppendResult adds a ticket to the current result set.
	// This is called when a ticket is successfully assigned.
	AppendResult(ticket models.MatchmakingRequest)

	// IsFulfilled checks if the current party combination meets the minimum requirements.
	// Returns true if the combination is ready for matchmaking.
	IsFulfilled() bool

	// Reset resets the party finder to its initial state.
	// This is used to start a new search iteration.
	Reset()

	// GetCurrentResult returns the current party combination being evaluated.
	GetCurrentResult() []models.MatchmakingRequest

	// GetBestResult returns the best party combination found so far.
	// This is the combination that best meets the matchmaking criteria.
	GetBestResult() []models.MatchmakingRequest
}

// GetPartyFinder returns party finder implementations. We have 3 party finder types:
// 1) newRoleBasedCombo() to find party for role-based with role combination (combo role-based), [NOTE: only available in matchmaking service build-in function]
// 2) newRoleBasedUnique() to find party for role-based without role combination (unique role-based), [NOTE: only available in matchmaking service build-in function]
// 3) newNormal() to find party for non role-based
func GetPartyFinder(playerMinNumber, playerMaxNumber int, current []models.MatchmakingRequest) (pf PartyFinder) {
	return newNormal(playerMinNumber, playerMaxNumber, current)
}
