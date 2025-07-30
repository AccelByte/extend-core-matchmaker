// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

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

func countPlayers(mmReq []models.MatchmakingRequest) int {
	var sum int
	for _, req := range mmReq {
		sum += len(req.PartyMembers)
	}
	return sum
}

func getPivotTicketIndexFromTickets(tickets []models.MatchmakingRequest, pivotTicket *models.MatchmakingRequest) (pivotIndex int) {
	for i, v := range tickets {
		if pivotTicket.PartyID == v.PartyID {
			pivotIndex = i
			break
		}
	}
	return pivotIndex
}

func getPriorityTicketIndexesFromTickets(tickets []models.MatchmakingRequest) (priorityIndexes []int) {
	for i, ticket := range tickets {
		if ticket.IsPriority() {
			priorityIndexes = append(priorityIndexes, i)
		}
	}
	return priorityIndexes
}

// reorderTickets return the exact same element of tickets but in new order based on newIndexes
// if index lower than 0 or exceed the tickets length, this function will just skip those index
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

func getTicketRegionToTryCount(ticket *models.MatchmakingRequest, channel *models.Channel) int {
	regionsToTry := getTicketRegionExpansionStep(ticket, channel)
	regionsToTry = mathutil.Max(mathutil.Min(regionsToTry, len(ticket.SortedLatency)), 1) // don't let this be below 1
	return regionsToTry
}

func getTicketRegionExpansionStep(ticket *models.MatchmakingRequest, channel *models.Channel) int {
	step := 1
	expansionRateMs := channel.Ruleset.RegionExpansionRateMs
	if expansionRateMs > 0 {
		ticketAge := float64(Now().Sub(time.Unix(ticket.CreatedAt, 0)))
		regionExpansionRate := float64(expansionRateMs * int(time.Millisecond))
		step = int(math.Ceil(ticketAge / regionExpansionRate))
	} else {
		attemptCount := 0
		val, ok := ticket.PartyAttributes[models.AttributeMatchAttempt]
		if ok {
			attemptCount, _ = strconv.Atoi(fmt.Sprint(val))
		}
		step += attemptCount
	}
	return step
}

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
	latency := initialRange + (steps-1)*additionalStep // by default start from 100ms for every expansion add 50ms
	if channel.Ruleset.RegionLatencyMaxMs > 0 {
		latency = mathutil.Min(latency, channel.Ruleset.RegionLatencyMaxMs) // don't allow to go over the max
	}
	return latency
}

func sortDESC(matchmakingRequests []models.MatchmakingRequest) {
	sort.Slice(matchmakingRequests, func(i, j int) bool {
		return matchmakingRequests[i].CountPlayer() > matchmakingRequests[j].CountPlayer()
	})
}

func isContainBlockedPlayers(tickets []models.MatchmakingRequest, ticket *models.MatchmakingRequest) bool {
	memberIDs := make([]string, 0)
	blockedIDs := make(map[string]struct{}, 0)

	memberIDs = append(memberIDs, ticket.GetMemberUserIDs()...)
	for _, blockedID := range ticket.GetBlockedPlayerUserIDs() {
		blockedIDs[blockedID] = struct{}{}
	}

	for _, ticket := range tickets {
		memberIDs = append(memberIDs, ticket.GetMemberUserIDs()...)
		for _, blockedID := range ticket.GetBlockedPlayerUserIDs() {
			blockedIDs[blockedID] = struct{}{}
		}
	}

	for _, userID := range memberIDs {
		if _, exist := blockedIDs[userID]; exist {
			return true
		}
	}
	return false
}

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

func skipFilterCandidateRegion(ticket *models.MatchmakingRequest, channel *models.Channel) bool {
	if channel.Ruleset.DisableBidirectionalLatencyAfterMs <= 0 {
		return false
	}
	disableDuration := time.Duration(channel.Ruleset.DisableBidirectionalLatencyAfterMs) * time.Millisecond
	ticketAge := time.Since(time.Unix(ticket.CreatedAt, 0))
	return disableDuration < ticketAge
}

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
