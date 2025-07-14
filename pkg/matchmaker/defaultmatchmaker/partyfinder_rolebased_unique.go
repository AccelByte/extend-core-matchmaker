// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"strings"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

type unique struct {
	minPlayer int
	maxPlayer int

	mapAssignedRole map[string]string // map[userid]role
	mapResult       map[string]int    // map[role]count

	current []models.MatchmakingRequest // to reset
	best    []models.MatchmakingRequest // to store best min fulfilled
	result  []models.MatchmakingRequest
}

func newRoleBasedUnique(
	minPlayer int,
	maxPlayer int,
	current []models.MatchmakingRequest,
) PartyFinder {
	mapAssignedRole := make(map[string]string, 0)
	mapResult := make(map[string]int)
	for _, ticket := range current {
		for _, member := range ticket.PartyMembers {
			if len(member.GetRole()) == 1 {
				assignedRole := member.GetRole()[0]
				mapAssignedRole[member.UserID] = assignedRole
				mapResult[assignedRole]++
			}
		}
	}
	return &unique{
		minPlayer: minPlayer,
		maxPlayer: maxPlayer,

		mapAssignedRole: mapAssignedRole,
		mapResult:       mapResult,

		current: current,
		best:    current,
		result:  current,
	}
}

func (f *unique) Reset() {
	mapAssignedRole := make(map[string]string, 0)
	mapResult := make(map[string]int)
	for _, ticket := range f.current {
		for _, member := range ticket.PartyMembers {
			if len(member.GetRole()) == 1 {
				assignedRole := member.GetRole()[0]
				mapAssignedRole[member.UserID] = assignedRole
				mapResult[assignedRole]++
			}
		}
	}
	f.mapAssignedRole = mapAssignedRole
	f.mapResult = mapResult
	f.result = f.current
}

func (f *unique) GetCurrentResult() []models.MatchmakingRequest {
	return f.result
}

func (f *unique) GetBestResult() []models.MatchmakingRequest {
	return f.best
}

func (f *unique) AssignMembers(ticket models.MatchmakingRequest) (success bool) {
	defer func() {
		if !success {
			// revert the assigned role
			for _, member := range ticket.PartyMembers {
				f.mapAssignedRole[member.UserID] = ""
			}
		}
	}()

	currPlayerCount := countPlayers(f.result)
	addPlayerCount := currPlayerCount + len(ticket.PartyMembers)

	// skip this ticket if player count will exceed max
	if addPlayerCount > f.maxPlayer {
		return false
	}

	// initiate count role with current result (cannot be direct)
	countRole := make(map[string]int)
	for role, count := range f.mapResult {
		countRole[role] = count
	}

	// assign the role
	for _, member := range ticket.PartyMembers {
		requestedRoles := member.GetRole()
		for _, role := range requestedRoles {
			// skip if member already have assigned role
			if f.mapAssignedRole[member.UserID] != "" {
				continue
			}

			// skip if this role already exist in result
			if countRole[role] > 0 {
				continue
			}

			// assign role to member
			f.mapAssignedRole[member.UserID] = role
			countRole[role]++
		}
	}

	// skip ticket if there's a member doesn't have assigned role
	for _, member := range ticket.PartyMembers {
		if f.mapAssignedRole[member.UserID] == "" {
			return false
		}
	}

	return true
}

func (f *unique) AppendResult(ticket models.MatchmakingRequest) {
	// we need to deep copy ticket because we wanna change role value in resultTicket but keep ticket as it is
	resultTicket := ticket.Copy()
	for memberID, member := range resultTicket.PartyMembers {
		assignedRole := f.mapAssignedRole[member.UserID]
		if strings.TrimSpace(assignedRole) == "" {
			continue
		}
		resultTicket.PartyMembers[memberID].ExtraAttributes[models.ROLE] = assignedRole
		f.mapResult[assignedRole]++
	}
	f.result = append(f.result, resultTicket)
}

func (f *unique) IsFulfilled() bool {
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
