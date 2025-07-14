// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v1

import (
	"context"
	"math"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
)

const (
	timeoutDuration = 200 * time.Millisecond
)

// Rebalance try to find closest gap between allies as smallest as possible.
// It look for any first attribute with criteria "distance" from matchingRules,
// It will keep swapping member between allies until it found smallest gap or timeoutDuration = 200 ms.
// Param matchID is used for logging purpose only.
// DO NOT USE parameter channel here to get alliance rule, we should get the active alliance rule to consider flex rule and sub game mode
func Rebalance(
	rootScope *envelope.Scope,
	matchID string,
	allies []models.MatchingAlly,
	activeAllianceRule models.AllianceRule,
	matchingRules []models.MatchingRule,
) []models.MatchingAlly {
	scope := rootScope.NewChildScope("Rebalance")
	defer scope.Finish()

	ctx, cancel := context.WithTimeout(rootScope.Ctx, timeoutDuration)
	defer cancel()

	attributeNames := rebalance.GetAttributeNameForRebalance(matchingRules)
	if len(attributeNames) <= 0 {
		scope.Log.Infof("[rebalance] done matchid: %s only distribute member count because no attribute detected", matchID)
		return allies
	}

	distance := countDistance(allies, attributeNames, matchingRules)

loopAll:
	for i := range allies {
		// index out of bound
		if (i + 1) >= len(allies) {
			break
		}

		// do swap
		indexAlly1 := i
		ally1 := allies[indexAlly1]
		indexAlly2 := (i + 1)
		ally2 := allies[indexAlly2]
		s := NewSwapper(indexAlly1, indexAlly2, ally1, ally2, attributeNames, activeAllianceRule, matchingRules)
		for s.HasNext() {
			select {
			case <-ctx.Done():
				scope.Log.Warnf("[rebalance] timeout matchid: %s swap count: %d", matchID, s.swapCount)
				break loopAll
			default:
				s.Swap(scope)
				allies[i] = s.ally1
				allies[i+1] = s.ally2
			}
		}
	}

	newDistance := countDistance(allies, attributeNames, matchingRules)
	scope.Log.Infof("[rebalance] done matchid: %s previous distance: %.2f new distance: %.2f", matchID, distance, newDistance)
	return allies
}

type Swapper struct {
	indexAlly1         int
	indexAlly2         int
	ally1              models.MatchingAlly
	ally2              models.MatchingAlly
	attr               []string
	activeAllianceRule models.AllianceRule
	matchingRules      []models.MatchingRule

	startAvgAlly1 float64
	startAvgAlly2 float64

	swapDone  bool
	swapCount int
}

// NewSwapper initiate ally swapper, it receive:
// - attributeName to be compared for swap
// - activeAllianceRule to validate the new combination after swap still meet the rule
func NewSwapper(
	indexAlly1 int,
	indexAlly2 int,
	ally1 models.MatchingAlly,
	ally2 models.MatchingAlly,
	attributeNames []string,
	activeAllianceRule models.AllianceRule,
	matchingRules []models.MatchingRule,
) *Swapper {
	return &Swapper{
		indexAlly1:         indexAlly1,
		indexAlly2:         indexAlly2,
		ally1:              ally1,
		ally2:              ally2,
		attr:               attributeNames,
		activeAllianceRule: activeAllianceRule,
		matchingRules:      matchingRules,

		startAvgAlly1: ally1.Avg(attributeNames, matchingRules),
		startAvgAlly2: ally2.Avg(attributeNames, matchingRules),
	}
}

// HasNext check is it possible to do any further swap
func (s *Swapper) HasNext() bool {
	return !s.swapDone
}

// GetSwapCount return how many swap performed
func (s *Swapper) GetSwapCount() int {
	return s.swapCount
}

/*
Swap do 1 time swap only.
It swap parties between ally1 and ally2 based on average of initiated attribute.
Swap will loop parties and stop when one of these criteria are met:

	a) found parties to be swap
	b) avg attr of ally1 and ally2 already reversed
	c) all parties has been checked
*/
func (s *Swapper) Swap(scope *envelope.Scope) {
	// no need to swap if avg is equal
	if s.startAvgAlly1 == s.startAvgAlly2 {
		s.swapDone = true
		return
	}

	// 1) sort parties in each ally
	if s.startAvgAlly1 > s.startAvgAlly2 {
		SortPartiesInAllyASC(s.ally1, s.attr, s.matchingRules)
		SortPartiesInAllyDESC(s.ally2, s.attr, s.matchingRules)
	}
	if s.startAvgAlly1 < s.startAvgAlly2 {
		SortPartiesInAllyDESC(s.ally1, s.attr, s.matchingRules)
		SortPartiesInAllyASC(s.ally2, s.attr, s.matchingRules)
	}

	// 2) find parties from ally1 and ally2 to be swapped where party1 > party2 but has the smallest distance
	for i, party1 := range s.ally1.MatchingParties {
		if party1.Locked {
			continue
		}
		for j, party2 := range s.ally2.MatchingParties {
			if party2.Locked {
				continue
			}
			membersParty1 := party1.PartyMembers
			indexesParty1 := []int{i}
			membersParty2 := party2.PartyMembers
			indexesParty2 := []int{j}

			if party1.CountPlayer() > party2.CountPlayer() {
				membersParty2, indexesParty2 = getMoreMember(s.ally2, j, party1.CountPlayer())
			} else if party1.CountPlayer() < party2.CountPlayer() {
				membersParty1, indexesParty1 = getMoreMember(s.ally1, i, party2.CountPlayer())
			}

			// skip if we cannot found equal members
			if len(membersParty1) != len(membersParty2) {
				continue
			}

			if s.startAvgAlly1 > s.startAvgAlly2 {
				// skip because swapping will only make avg ticket1 getting bigger
				if Avg(membersParty2, s.attr, s.matchingRules) >= Avg(membersParty1, s.attr, s.matchingRules) {
					continue
				}
			}
			if s.startAvgAlly1 < s.startAvgAlly2 {
				// skip because swapping will only make avg ticket1 getting smaller
				if Avg(membersParty2, s.attr, s.matchingRules) <= Avg(membersParty1, s.attr, s.matchingRules) {
					continue
				}
			}

			// 3) swap member of ally1 and ally2 where avg MMR ally1 > avg MMR party2 but has the smallest distance
			var newAlly1, newAlly2 models.MatchingAlly
			newAlly1.MatchingParties, newAlly2.MatchingParties = swapParties(s.ally1.MatchingParties, s.ally2.MatchingParties, indexesParty1, indexesParty2)

			// skip if new allies not valid based on alliance rule
			if err := s.activeAllianceRule.ValidateAlly(newAlly1, s.indexAlly1); err != nil {
				scope.Log.
					WithField("ally", newAlly1).
					Debug("requests is invalid:", err)
				continue
			}
			if err := s.activeAllianceRule.ValidateAlly(newAlly2, s.indexAlly2); err != nil {
				scope.Log.
					WithField("ally", newAlly2).
					Debug("requests is invalid:", err)
				continue
			}

			// 4) validate new combination
			if s.isReversedAvg(newAlly1, newAlly2) {
				distance := math.Abs(s.ally1.Avg(s.attr, s.matchingRules) - s.ally2.Avg(s.attr, s.matchingRules))
				newDistance := math.Abs(newAlly1.Avg(s.attr, s.matchingRules) - newAlly2.Avg(s.attr, s.matchingRules))

				if newDistance < distance {
					// apply swap, the last changes has better distance
					s.ally1.MatchingParties = newAlly1.MatchingParties
					s.ally2.MatchingParties = newAlly2.MatchingParties
					s.swapCount++
				}

				// reaching limit where avg mmr of ally 1 and ally 2 already reversed from starting point
				s.swapDone = true
				return
			}

			// apply swap, not done yet, because it can be swap again
			s.ally1.MatchingParties = newAlly1.MatchingParties
			s.ally2.MatchingParties = newAlly2.MatchingParties
			s.swapCount++
			return
		}
	}

	// final stop
	// note: we not use defer since we have condition above where values returned but not done yet
	s.swapDone = true
}

func (s *Swapper) isReversedAvg(ally1, ally2 models.MatchingAlly) bool {
	if s.startAvgAlly1 > s.startAvgAlly2 {
		return ally1.Avg(s.attr, s.matchingRules) <= ally2.Avg(s.attr, s.matchingRules)
	}
	if s.startAvgAlly1 < s.startAvgAlly2 {
		return ally1.Avg(s.attr, s.matchingRules) >= ally2.Avg(s.attr, s.matchingRules)
	}
	return false
}

func getMoreMember(allies models.MatchingAlly, currentPartyIndex int, requiredCount int) (members []models.PartyMember, idxSwaps []int) {
	for i, p := range allies.MatchingParties {
		if i < currentPartyIndex {
			// skip lower index
			continue
		}
		if (len(members) + p.CountPlayer()) > requiredCount {
			// skip if more than required
			continue
		}
		members = append(members, p.PartyMembers...)
		idxSwaps = append(idxSwaps, i)
	}
	return
}

func swapParties(party1, party2 []models.MatchingParty, swapIDParty1, swapIDParty2 []int) (newParty1, newParty2 []models.MatchingParty) {
	for i, party := range party1 {
		if utils.Contains(swapIDParty1, i) {
			newParty2 = append(newParty2, party)
			continue
		}
		newParty1 = append(newParty1, party)
	}
	for i, party := range party2 {
		if utils.Contains(swapIDParty2, i) {
			newParty1 = append(newParty1, party)
			continue
		}
		newParty2 = append(newParty2, party)
	}
	return
}
