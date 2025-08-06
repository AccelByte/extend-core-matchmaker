// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package defaultmatchmaker provides the default implementation of the MatchLogic interface.
// This package contains the core matchmaking algorithms and logic for creating matches from tickets.
package defaultmatchmaker

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

// normal implements the PartyFinder interface for standard (non-role-based) matchmaking.
// This party finder focuses on player count and basic party management without role considerations.
type normal struct {
	minPlayer int // Minimum number of players required for a match
	maxPlayer int // Maximum number of players allowed in a match

	current []models.MatchmakingRequest // Current party combination being evaluated
	best    []models.MatchmakingRequest // Best party combination found so far
	result  []models.MatchmakingRequest // Current result set
}

// newNormal creates a new normal party finder instance.
// This is the constructor for standard party finding without role-based logic.
func newNormal(
	minPlayer int,
	maxPlayer int,
	current []models.MatchmakingRequest,
) PartyFinder {
	return &normal{
		minPlayer: minPlayer,
		maxPlayer: maxPlayer,

		current: current,
		best:    current,
		result:  current,
	}
}

// Reset resets the party finder to its initial state.
// This allows the finder to start a new search iteration.
func (f *normal) Reset() {
	f.result = f.current
}

// GetCurrentResult returns the current party combination being evaluated.
func (f *normal) GetCurrentResult() []models.MatchmakingRequest {
	return f.result
}

// GetBestResult returns the best party combination found so far.
// This is the combination that best meets the matchmaking criteria.
func (f *normal) GetBestResult() []models.MatchmakingRequest {
	return f.best
}

// AssignMembers attempts to assign a ticket to the current party combination.
// Returns true if the assignment would not exceed the maximum player count.
func (f *normal) AssignMembers(ticket models.MatchmakingRequest) (success bool) {
	currPlayerCount := countPlayers(f.result)
	addPlayerCount := currPlayerCount + len(ticket.PartyMembers)

	// Skip this ticket if player count will exceed max
	// if addPlayerCount > f.maxPlayer {
	// 	return false
	// }

	// return true

	return addPlayerCount <= f.maxPlayer
}

// AppendResult adds a ticket to the current result set.
// This is called when a ticket is successfully assigned.
func (f *normal) AppendResult(ticket models.MatchmakingRequest) {
	f.result = append(f.result, ticket)
}

// IsFulfilled checks if the current party combination meets the minimum requirements.
// Returns true if the combination has enough players and updates the best result if needed.
func (f *normal) IsFulfilled() bool {
	playerCount := countPlayers(f.result)

	// Not fulfilled if minimum player count is not met
	if playerCount < f.minPlayer {
		return false
	}

	// If minimum requirement is passed, store in best if it's better than current best
	if playerCount <= f.maxPlayer && playerCount > countPlayers(f.best) {
		f.best = f.result
	}

	return playerCount == f.maxPlayer
}
