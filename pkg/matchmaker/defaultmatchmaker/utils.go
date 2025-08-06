// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package defaultmatchmaker provides the default implementation of the MatchLogic interface.
// This package contains the core matchmaking algorithms and logic for creating matches from tickets.
package defaultmatchmaker

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/mathutil"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"gopkg.in/typ.v4/slices"
)

// countPlayers counts the total number of players across all matchmaking requests.
// This is used to determine if there are enough players to form a match.
func countPlayers(mmReq []models.MatchmakingRequest) int {
	var sum int
	for _, req := range mmReq {
		sum += len(req.PartyMembers)
	}
	return sum
}

// getPivotTicketIndexFromTickets finds the index of a pivot ticket within a slice of tickets.
// The pivot ticket is used as the reference point for matchmaking algorithms.
func getPivotTicketIndexFromTickets(tickets []models.MatchmakingRequest, pivotTicket *models.MatchmakingRequest) (pivotIndex int) {
	for i, v := range tickets {
		if pivotTicket.PartyID == v.PartyID {
			pivotIndex = i
			break
		}
	}
	return pivotIndex
}

// getPriorityTicketIndexesFromTickets finds all tickets that have priority status.
// Priority tickets are processed first in the matchmaking queue.
func getPriorityTicketIndexesFromTickets(tickets []models.MatchmakingRequest) (priorityIndexes []int) {
	for i, ticket := range tickets {
		if ticket.IsPriority() {
			priorityIndexes = append(priorityIndexes, i)
		}
	}
	return priorityIndexes
}

// reorderTickets returns the exact same elements of tickets but in a new order based on newIndexes.
// If an index is lower than 0 or exceeds the tickets length, this function will just skip those indexes.
func reorderTickets(tickets []models.MatchmakingRequest, newIndexes []int) (reorderedTickets []models.MatchmakingRequest) {
	reorderedTickets = make([]models.MatchmakingRequest, 0)
	for _, newIndex := range newIndexes {
		if newIndex < 0 || newIndex > (len(tickets)-1) {
			continue
		}
		reorderedTickets = append(reorderedTickets, tickets[newIndex])
	}
	return reorderedTickets
}

// getTicketRegionToTryCount determines how many regions to try for a given ticket.
// This is based on the ticket's age and the channel's expansion rate configuration.
func getTicketRegionToTryCount(ticket *models.MatchmakingRequest, channel *models.Channel) int {
	regionsToTry := getTicketRegionExpansionStep(ticket, channel)
	regionsToTry = mathutil.Max(mathutil.Min(regionsToTry, len(ticket.SortedLatency)), 1) // Don't let this be below 1
	return regionsToTry
}

// getTicketRegionExpansionStep calculates the region expansion step for a ticket.
// This determines how many regions to consider based on the ticket's age and expansion rate.
func getTicketRegionExpansionStep(ticket *models.MatchmakingRequest, channel *models.Channel) int {
	step := 1
	expansionRateMs := channel.Ruleset.RegionExpansionRateMs
	if expansionRateMs > 0 {
		// Calculate step based on ticket age and expansion rate
		ticketAge := float64(Now().Sub(time.Unix(ticket.CreatedAt, 0)))
		regionExpansionRate := float64(expansionRateMs * int(time.Millisecond))
		step = int(math.Ceil(ticketAge / regionExpansionRate))
	} else {
		// Use attempt count as fallback
		attemptCount := 0
		val, ok := ticket.PartyAttributes[models.AttributeMatchAttempt]
		if ok {
			attemptCount, _ = strconv.Atoi(fmt.Sprint(val))
		}
		step += attemptCount
	}
	return step
}

// getTicketMaxLatency calculates the maximum acceptable latency for a ticket.
// This is based on the initial range, expansion steps, and maximum allowed latency.
func getTicketMaxLatency(ticket *models.MatchmakingRequest, channel *models.Channel) int {
	initialRange := 200
	if channel.Ruleset.RegionLatencyInitialRangeMs > 0 {
		initialRange = channel.Ruleset.RegionLatencyInitialRangeMs
	}
	additionalStep := 50
	if channel.Ruleset.RegionExpansionRangeMs > 0 {
		additionalStep = channel.Ruleset.RegionExpansionRangeMs
	}
	steps := getTicketRegionExpansionStep(ticket, channel)
	steps = mathutil.Max(steps, 1)
	latency := initialRange + (steps-1)*additionalStep // By default start from 100ms for every expansion add 50ms
	if channel.Ruleset.RegionLatencyMaxMs > 0 {
		latency = mathutil.Min(latency, channel.Ruleset.RegionLatencyMaxMs) // Don't allow to go over the max
	}
	return latency
}

// sortDESC sorts matchmaking requests by player count in descending order.
// This function prioritizes larger parties for matchmaking.
func sortDESC(matchmakingRequests []models.MatchmakingRequest) {
	sort.Slice(matchmakingRequests, func(i, j int) bool {
		return matchmakingRequests[i].CountPlayer() > matchmakingRequests[j].CountPlayer()
	})
}

// isContainBlockedPlayers checks if any players in the given tickets are blocked.
// This function collects all member IDs and blocked IDs, then checks for conflicts.
func isContainBlockedPlayers(tickets []models.MatchmakingRequest, ticket *models.MatchmakingRequest) bool {
	memberIDs := make([]string, 0)
	blockedIDs := make(map[string]struct{}, 0)

	// Collect member IDs and blocked IDs from the current ticket
	memberIDs = append(memberIDs, ticket.GetMemberUserIDs()...)
	for _, blockedID := range ticket.GetBlockedPlayerUserIDs() {
		blockedIDs[blockedID] = struct{}{}
	}

	// Collect member IDs and blocked IDs from all other tickets
	for _, ticket := range tickets {
		memberIDs = append(memberIDs, ticket.GetMemberUserIDs()...)
		for _, blockedID := range ticket.GetBlockedPlayerUserIDs() {
			blockedIDs[blockedID] = struct{}{}
		}
	}

	// Check if any member ID is in the blocked IDs list
	for _, userID := range memberIDs {
		if _, exist := blockedIDs[userID]; exist {
			return true
		}
	}
	return false
}

// filterRegionByStep filters regions based on the ticket's expansion step and latency requirements.
// This function determines which regions are acceptable for matchmaking based on latency constraints.
func filterRegionByStep(ticket *models.MatchmakingRequest, channel *models.Channel) []models.Region {
	if len(ticket.SortedLatency) == 0 {
		return nil
	}
	expansionStep := mathutil.Max(getTicketRegionExpansionStep(ticket, channel)-1, 0)
	additionalLatency := 50
	if channel.Ruleset.RegionExpansionRangeMs > 0 {
		additionalLatency = channel.Ruleset.RegionExpansionRangeMs
	}
	maxLatency := ticket.SortedLatency[0].Latency + channel.Ruleset.RegionLatencyInitialRangeMs + (expansionStep * additionalLatency)
	hardMaxLatency := channel.Ruleset.RegionLatencyMaxMs
	return slices.Filter(ticket.SortedLatency, func(item models.Region) bool {
		return item.Latency <= maxLatency && (hardMaxLatency <= 0 || (hardMaxLatency > 0 && item.Latency <= hardMaxLatency))
	})
}

// skipFilterCandidateRegion determines if region filtering should be skipped based on ticket age.
// This function implements a timeout mechanism to disable bidirectional latency filtering after a certain duration.
func skipFilterCandidateRegion(ticket *models.MatchmakingRequest, channel *models.Channel) bool {
	if channel.Ruleset.DisableBidirectionalLatencyAfterMs <= 0 {
		return false
	}
	disableDuration := time.Duration(channel.Ruleset.DisableBidirectionalLatencyAfterMs) * time.Millisecond
	ticketAge := time.Since(time.Unix(ticket.CreatedAt, 0))
	return disableDuration < ticketAge
}

// RemoveEmptyMatchingParties removes empty matching parties from the allies list.
// This function cleans up the allies list by removing parties with no members.
func RemoveEmptyMatchingParties(allies []models.MatchingAlly) []models.MatchingAlly {
	for i := 0; i < len(allies); i++ {
		ally := allies[i]
		if len(ally.MatchingParties) == 0 || (len(ally.MatchingParties) == 1 && len(ally.MatchingParties[0].PartyMembers) == 0) {
			allies[i] = allies[len(allies)-1]
			allies = allies[:len(allies)-1]
			i--
		}
	}
	return allies
}

// DetermineAllianceComposition extracts alliance composition from a ruleset.
// This function creates an AllianceComposition structure from the alliance rule configuration.
func DetermineAllianceComposition(ruleSet models.RuleSet) models.AllianceComposition {
	minTeam := ruleSet.AllianceRule.MinNumber
	maxTeam := ruleSet.AllianceRule.MaxNumber
	maxPlayer := ruleSet.AllianceRule.PlayerMaxNumber
	minPlayer := ruleSet.AllianceRule.PlayerMinNumber

	return models.AllianceComposition{
		MinTeam:   minTeam,
		MaxTeam:   maxTeam,
		MaxPlayer: maxPlayer,
		MinPlayer: minPlayer,
	}
}
