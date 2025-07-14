// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v1

import (
	"context"
	"reflect"

	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance"
)

func BackfillRebalance(rootScope *envelope.Scope, matchID string, allies []models.MatchingAlly, activeAllianceRule models.AllianceRule, matchingRules []models.MatchingRule) []models.MatchingAlly {
	scope := rootScope.NewChildScope("BackfillRebalance")
	defer scope.Finish()

	// need to adjust allies based on max team number
	// to cover case in AR-5693
	allies = adjustAlliesBasedOnMaxTeamNumber(allies, activeAllianceRule)

	// extract allies into locked parties, unlocked parties,
	// and store current allies into best allies (map formatted)
	lockedParties, unlockedParties, bestAllies := rebalance.ExtractAllies(allies)

	// get attribute name from matchingRules
	attributeNames := rebalance.GetAttributeNameForRebalance(matchingRules)

	// no need to rebalance if no unlocked parties (newParties)
	if len(unlockedParties) == 0 {
		return allies
	}

	distance := countDistance(allies, attributeNames, matchingRules)

	// do rebalance with timeout duration
	ctx, cancel := context.WithTimeout(rootScope.Ctx, timeoutDuration)
	defer cancel()

	// create permutations from unlockedParties's indexes
	newIndexes := make([]int, len(unlockedParties))
	for i := range unlockedParties {
		newIndexes[i] = i
	}
	permutations := permutations(newIndexes)

outerLoop:
	for _, newIndexes := range permutations {
		select {
		case <-ctx.Done():
			scope.Log.Warnf("[backfill_rebalance] timeout matchid: %s", matchID)
			break outerLoop
		default:

			// reset session
			session := convert(lockedParties)

			// reorder new parties
			newParties := reorder(unlockedParties, newIndexes)

			for _, party := range newParties {
				memberCountDiff := make(map[int]int, len(session))
				for i, ally := range session {
					// record current length of matching parties to n
					n := len(ally.MatchingParties)

					// temporary add party to this ally
					ally.MatchingParties = append(ally.MatchingParties, party)
					session[i] = ally

					// count diff if party is in this ally
					memberCountDiff[i] = CountMemberDiff(session)

					// move party from this ally
					ally.MatchingParties = ally.MatchingParties[:n]
					session[i] = ally
				}

				// filter valid allies based on member count diff
				validAllies := make(map[int]struct{})
				smallestDiff := 0
				for i, diff := range memberCountDiff {
					switch {
					case len(validAllies) == 0 || diff == smallestDiff:
						validAllies[i] = struct{}{}
						smallestDiff = diff
					case diff < smallestDiff:
						validAllies = make(map[int]struct{})
						validAllies[i] = struct{}{}
						smallestDiff = diff
					}
				}

				distancePerAlly := make(map[int]float64)
				for i, ally := range session {
					if _, ok := validAllies[i]; !ok {
						// skip this ally if not valid based on member count
						continue
					}

					// record current length of matching parties to n
					n := len(ally.MatchingParties)

					// temporary add party to this ally
					ally.MatchingParties = append(ally.MatchingParties, party)
					session[i] = ally

					// validate ally when we add this party
					validationErr := activeAllianceRule.ValidateAlly(ally, i)

					// find best mmr distance
					distance := countDistance(session, attributeNames, matchingRules)

					// move party from this ally
					ally.MatchingParties = ally.MatchingParties[:n]
					session[i] = ally

					if validationErr != nil {
						// do not record the mmr distance if ally will not be valid
						continue
					}

					// record distance
					distancePerAlly[i] = distance
				}

				validAlly := make(map[int]struct{})
				smallestDistance := 0.0
				for i, distance := range distancePerAlly {
					if len(validAlly) == 0 || distance < smallestDistance {
						// we will record only 1 ally
						// if there are more than 1 ally with same smallest distance, we take the first ally
						validAlly = make(map[int]struct{})
						validAlly[i] = struct{}{}
						smallestDistance = distance
					}
				}

				if len(validAlly) == 0 {
					// try other combination, we need to make sure all new parties get the place
					continue outerLoop
				}

				var i int
				for ally := range validAlly {
					i = ally
					break
				}

				// append party to session
				session[i].MatchingParties = append(session[i].MatchingParties, party)
			}

			// compare with best
			// replace best if the mmr distance is better
			best := convert(bestAllies)

			currentDistance := countDistance(session, attributeNames, matchingRules)
			bestDistance := countDistance(best, attributeNames, matchingRules)

			currentMemberDiff := CountMemberDiff(session)
			bestMemberDiff := CountMemberDiff(best)

			if reflect.DeepEqual(bestAllies, allies) || (currentDistance <= bestDistance && currentMemberDiff <= bestMemberDiff) {
				bestAllies = make(map[int][]models.MatchingParty)
				for i, ally := range session {
					bestAllies[i] = append(bestAllies[i], ally.MatchingParties...)
				}
			}
		}
	}

	newAllies := convert(bestAllies)
	newDistance := countDistance(newAllies, attributeNames, matchingRules)

	scope.Log.Infof("[backfill_rebalance] done matchid: %s previous distance: %.2f new distance: %.2f", matchID, distance, newDistance)
	return newAllies
}

func adjustAlliesBasedOnMaxTeamNumber(allies []models.MatchingAlly, activeAllianceRule models.AllianceRule) []models.MatchingAlly {
	if len(allies) == activeAllianceRule.MaxNumber {
		return allies
	}
	for i := 0; i < activeAllianceRule.MaxNumber-len(allies); i++ {
		allies = append(allies, models.MatchingAlly{})
	}
	return allies
}

func convert(m map[int][]models.MatchingParty) []models.MatchingAlly {
	allies := make([]models.MatchingAlly, len(m))
	for k, v := range m {
		allies[k].MatchingParties = v
	}
	return allies
}

func reorder(parties []models.MatchingParty, newIndexes []int) []models.MatchingParty {
	reorderedParties := make([]models.MatchingParty, len(parties))
	for i, idx := range newIndexes {
		if idx < 0 || idx > (len(parties)-1) {
			continue
		}
		reorderedParties[i] = parties[idx]
	}
	return reorderedParties
}

func CountMemberDiff(allies []models.MatchingAlly) int {
	var _min, _max int
	for _, ally := range allies {
		if _min == 0 || ally.CountPlayer() < _min {
			_min = ally.CountPlayer()
		}
		if _max == 0 || ally.CountPlayer() > _max {
			_max = ally.CountPlayer()
		}
	}
	return _max - _min
}

func permutations(arr []int) [][]int {
	var helper func([]int, int)
	res := [][]int{}

	helper = func(arr []int, n int) {
		if n == 1 {
			tmp := make([]int, len(arr))
			copy(tmp, arr)
			res = append(res, tmp)
		} else {
			for i := 0; i < n; i++ {
				helper(arr, n-1)
				if n%2 == 1 {
					arr[i], arr[n-1] = arr[n-1], arr[i]
				} else {
					arr[0], arr[n-1] = arr[n-1], arr[0]
				}
			}
		}
	}
	helper(arr, len(arr))
	return res
}
