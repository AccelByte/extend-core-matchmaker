// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

type normal struct {
	minPlayer int
	maxPlayer int

	current []models.MatchmakingRequest // to reset
	best    []models.MatchmakingRequest // to store best min fulfilled
	result  []models.MatchmakingRequest
}

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

func (f *normal) Reset() {
	f.result = f.current
}

func (f *normal) GetCurrentResult() []models.MatchmakingRequest {
	return f.result
}

func (f *normal) GetBestResult() []models.MatchmakingRequest {
	return f.best
}

func (f *normal) AssignMembers(ticket models.MatchmakingRequest) (success bool) {
	currPlayerCount := countPlayers(f.result)
	addPlayerCount := currPlayerCount + len(ticket.PartyMembers)

	// // skip this ticket if player count will exceed max
	// if addPlayerCount > f.maxPlayer {
	// 	return false
	// }

	// return true

	return addPlayerCount <= f.maxPlayer
}

func (f *normal) AppendResult(ticket models.MatchmakingRequest) {
	f.result = append(f.result, ticket)
}

func (f *normal) IsFulfilled() bool {
	playerCount := countPlayers(f.result)

	// not fulfilled minimum player count
	if playerCount < f.minPlayer {
		return false
	}

	// if minimum requirement passed, store in best
	if playerCount <= f.maxPlayer && playerCount > countPlayers(f.best) {
		f.best = f.result
	}

	return playerCount == f.maxPlayer
}
