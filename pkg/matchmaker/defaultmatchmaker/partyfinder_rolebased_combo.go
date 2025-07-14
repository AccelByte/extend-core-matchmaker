// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"strings"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
)

type combo struct {
	minPlayer int
	maxPlayer int
	roles     []models.Role

	mapAssignedRole map[string]string // map[userid]role
	mapResult       map[string]int    // map[role]count

	current []models.MatchmakingRequest // to reset
	best    []models.MatchmakingRequest // to store best min fulfilled
	result  []models.MatchmakingRequest
}

func newRoleBasedCombo(
	minPlayer int,
	maxPlayer int,
	roles []models.Role,
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
	return &combo{
		minPlayer: minPlayer,
		maxPlayer: maxPlayer,
		roles:     roles,

		mapAssignedRole: mapAssignedRole,
		mapResult:       mapResult,

		current: current,
		best:    current,
		result:  current,
	}
}

func (f *combo) Reset() {
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

func (f *combo) GetCurrentResult() []models.MatchmakingRequest {
	return f.result
}

func (f *combo) GetBestResult() []models.MatchmakingRequest {
	return f.best
}

func (f *combo) AssignMembers(ticket models.MatchmakingRequest) (success bool) {
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

	// skip this ticket if role count will exceed max (this is to handle if there's "any" role)
	if len(ticket.PartyMembers) > countAvailableRole(f.roles, f.mapResult) {
		return false
	}

	// initiate map roles max and min
	mapRolesMax := make(map[string]int)
	mapRolesMin := make(map[string]int)
	for _, role := range f.roles {
		mapRolesMax[role.Name] = role.Max
		mapRolesMin[role.Name] = role.Min
	}

	// before assign the role, let put member with "any" role in last of the list
	orderedMembers := make([]models.PartyMember, 0, len(ticket.PartyMembers))
	membersWithAny := make([]models.PartyMember, 0)
	for _, member := range ticket.PartyMembers {
		if utils.Contains(member.GetRole(), models.AnyRole) {
			membersWithAny = append(membersWithAny, member)
			continue
		}
		orderedMembers = append(orderedMembers, member)
	}
	if len(membersWithAny) > 0 {
		orderedMembers = append(orderedMembers, membersWithAny...)
	}

	// initiate count role with current result (cannot be direct)
	countRole := make(map[string]int)
	for _, role := range f.roles {
		count := f.mapResult[role.Name]
		countRole[role.Name] = count
	}
	countPlayer := countPlayers(f.result)

	// loop member to assign the role
	for _, member := range orderedMembers {
		// skip if member already have assigned role
		if f.mapAssignedRole[member.UserID] != "" {
			continue
		}

		requestedRoles := member.GetRole()
		if len(requestedRoles) == 0 {
			continue
		}

		suggestedRole := getAvailableRole(f.roles, countRole)

		var assignedRole string
		switch {
		case len(requestedRoles) == 1 && requestedRoles[0] == models.AnyRole:
			assignedRole = suggestedRole
		case utils.Contains(requestedRoles, suggestedRole):
			assignedRole = suggestedRole
		default:
			for _, role := range requestedRoles {
				// update role when "any"
				if role == models.AnyRole {
					role = suggestedRole
				}

				// skip if role count already reach max
				if countRole[role] == mapRolesMax[role] {
					continue
				}

				assignedRole = role
				break
			}
		}

		// skip if player count will reach max but there's role not fulfilled and is not the selected role
		if (countPlayer + 1) == f.maxPlayer {
			mapLessRole := make(map[string]bool)
			for _, r := range f.roles {
				if countRole[r.Name] < mapRolesMin[r.Name] {
					mapLessRole[r.Name] = true
				}
			}
			if len(mapLessRole) > 0 && !mapLessRole[assignedRole] {
				continue
			}
		}

		// assign role to member
		f.mapAssignedRole[member.UserID] = assignedRole
		countRole[assignedRole]++
		countPlayer++
	}

	// skip ticket if there's a member doesn't have assigned role
	for _, member := range ticket.PartyMembers {
		if f.mapAssignedRole[member.UserID] == "" {
			return false
		}
	}

	// skip this ticket if player count will reach max player but there's role not fulfilled
	if addPlayerCount == f.maxPlayer {
		for _, role := range f.roles {
			if countRole[role.Name] < role.Min {
				return false
			}
		}
	}

	return true
}

func (f *combo) AppendResult(ticket models.MatchmakingRequest) {
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

func (f *combo) IsFulfilled() bool {
	for _, role := range f.roles {
		// not fulfilled minimum role
		if f.mapResult[role.Name] < role.Min {
			return false
		}
	}

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

// getAvailableRole get role consecutively by considering min and max requirement
func getAvailableRole(roles []models.Role, countRole map[string]int) (availableRole string) {
	// first, we need to consider min role
	for _, role := range roles {
		// skip this role if already reach max
		if countRole[role.Name] >= role.Max {
			continue
		}

		// skip this role if already fulfill min
		if countRole[role.Name] >= role.Min {
			continue
		}

		// assigned first read role
		if availableRole == "" {
			availableRole = role.Name
			continue
		}

		// assigned role with less count first
		if countRole[role.Name] < countRole[availableRole] {
			availableRole = role.Name
		}
	}
	if availableRole != "" {
		return availableRole
	}

	// second, just assign consecutively
	for _, role := range roles {
		// skip this role if already reach max
		if countRole[role.Name] >= role.Max {
			continue
		}

		// assigned first read role
		if availableRole == "" {
			availableRole = role.Name
			continue
		}

		// assigned role with less count first
		if countRole[role.Name] < countRole[availableRole] {
			availableRole = role.Name
		}
	}
	return availableRole
}

func countAvailableRole(roles []models.Role, countRole map[string]int) int {
	var count int
	for _, role := range roles {
		if countRole[role.Name] >= role.Max {
			continue
		}
		diff := role.Max - countRole[role.Name]
		count += diff
	}
	return count
}
