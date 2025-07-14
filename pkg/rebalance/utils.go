// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance

import (
	"strings"

	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

// GetAttributeNameForRebalance take any first attribute with "distance" criteria.
func GetAttributeNameForRebalance(matchingRules []models.MatchingRule) []string {
	// take any first attribute with "distance" criteria from matching rule
	attributes := []string{}
	firstAttribute := ""
	allFalse := true
	for _, matchingRule := range matchingRules {
		if strings.EqualFold(matchingRule.Criteria, constants.DistanceCriteria) {
			if matchingRule.IsForBalancing != nil {
				if *matchingRule.IsForBalancing {
					attributes = append(attributes, matchingRule.Attribute)
					allFalse = false
				}
			} else {
				allFalse = false
			}
			if firstAttribute == "" {
				firstAttribute = matchingRule.Attribute
			}
		}
	}
	if len(attributes) > 0 {
		return attributes
	}
	if firstAttribute == "" || allFalse {
		return nil
	}
	return []string{firstAttribute}
}

func CountDistance(allies []models.MatchingAlly, attributeNames []string, matchingRules []models.MatchingRule) float64 {
	var minAvg float64
	var maxAvg float64
	for i, a := range allies {
		avg := a.Avg(attributeNames, matchingRules)
		if i == 0 || avg < minAvg {
			minAvg = avg
		}
		if i == 0 || avg > maxAvg {
			maxAvg = avg
		}
	}
	return maxAvg - minAvg
}

func ExtractAllies(
	allies []models.MatchingAlly,
) (
	lockedParties map[int][]models.MatchingParty,
	unlockedParties []models.MatchingParty,
	bestAllies map[int][]models.MatchingParty,
) {
	// lockedParties are current session (party with locked=true)
	lockedParties = make(map[int][]models.MatchingParty)
	// unlockedParties are new parties (party with locked=false)
	unlockedParties = make([]models.MatchingParty, 0)
	// bestAllies store current allies
	bestAllies = make(map[int][]models.MatchingParty)

	for i, ally := range allies {
		// initiate lockedParties and bestAllies for each ally
		if _, ok := lockedParties[i]; !ok {
			lockedParties[i] = nil
		}
		if _, ok := bestAllies[i]; !ok {
			bestAllies[i] = nil
		}

		// assign party to lockedParties, unlockedParties and bestAllies
		for _, party := range ally.MatchingParties {
			if party.Locked {
				lockedParties[i] = append(lockedParties[i], party)
			} else {
				unlockedParties = append(unlockedParties, party)
			}
			bestAllies[i] = append(bestAllies[i], party)
		}
	}
	return lockedParties, unlockedParties, bestAllies
}
