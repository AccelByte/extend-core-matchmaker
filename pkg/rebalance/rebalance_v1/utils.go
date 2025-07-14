// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v1

import (
	"sort"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance"
)

var countDistance = rebalance.CountDistance

func Avg(members []models.PartyMember, attributeNames []string, matchingRules []models.MatchingRule) float64 {
	return models.PartyMemberAvg(members, attributeNames, matchingRules)
}

func SortPartiesInAllyASC(ally models.MatchingAlly, attributeNames []string, matchingRules []models.MatchingRule) {
	sort.Slice(ally.MatchingParties, func(i, j int) bool {
		return ally.MatchingParties[i].Avg(attributeNames, matchingRules) < ally.MatchingParties[j].Avg(attributeNames, matchingRules)
	})
}

func SortPartiesInAllyDESC(ally models.MatchingAlly, attributeNames []string, matchingRules []models.MatchingRule) {
	sort.Slice(ally.MatchingParties, func(i, j int) bool {
		return ally.MatchingParties[i].Avg(attributeNames, matchingRules) > ally.MatchingParties[j].Avg(attributeNames, matchingRules)
	})
}

func SortPartiesByMemberCountDESC(parties []models.MatchingParty) {
	sort.Slice(parties, func(i, j int) bool {
		return parties[i].CountPlayer() > parties[j].CountPlayer()
	})
}
