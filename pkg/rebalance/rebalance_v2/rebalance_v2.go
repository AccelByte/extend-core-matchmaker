// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v2

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance"

	"github.com/sirupsen/logrus"
)

func RebalanceV2(
	rootScope *envelope.Scope,
	matchID string,
	alliesInput []models.MatchingAlly,
	activeAllianceRule models.AllianceRule,
	matchingRules []models.MatchingRule,
	blockedPlayerOption models.BlockedPlayerOption,
) []models.MatchingAlly {
	scope := rootScope.NewChildScope("RebalanceV2")
	defer scope.Finish()

	logFields := logrus.Fields{
		"matchID":             matchID, // only filled when backfill or matchSessions
		"previousComposition": CopyAllies(alliesInput),
	}

	// get attribute name from matchingRules
	attributeNames := rebalance.GetAttributeNameForRebalance(matchingRules)
	logFields["attribute"] = attributeNames
	isAttributeExist := len(attributeNames) > 0

	// need to adjust allies based on max team number
	allies := adjustAlliesBasedOnMaxTeamNumber(alliesInput, activeAllianceRule)

	logFields["numTeam"] = len(allies)

	numParty, numPlayer := CountPartyAndPlayer(allies)
	logFields["numParty"] = numParty
	logFields["numPlayer"] = numPlayer

	// extract allies into locked allies (filled when backfill), parties,
	// and store current allies into best allies (map formatted)
	lockedAllies, parties, bestAllies := rebalance.ExtractAllies(allies)

	// no need to rebalance if no parties
	if len(parties) == 0 {
		logrus.WithFields(logFields).Debug("rebalance no parties to rebalance")
		return alliesInput
	}

	var diffOld float64
	if isAttributeExist {
		diffOld = countDistance(alliesInput, attributeNames, matchingRules)
	}

	newAllies := rebalanceWithCGAPartition(scope, logFields, lockedAllies, parties, bestAllies, activeAllianceRule, attributeNames, blockedPlayerOption, matchingRules)
	newAllies = RemoveEmptyMatchingParties(newAllies)
	diffNew := countDistance(newAllies, attributeNames, matchingRules)

	if isAttributeExist {
		logFields["diffOld"] = int(diffOld)
		logFields["diffNew"] = int(diffNew)
		logFields["equalDistance"] = int(diffOld) == int(diffNew)
		logFields["moreThan100"] = int(diffNew) > 100
	}

	scope.Log.WithFields(logFields).Info("rebalance done")
	return newAllies
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

func adjustAlliesBasedOnMaxTeamNumber(allies []models.MatchingAlly, activeAllianceRule models.AllianceRule) []models.MatchingAlly {
	if len(allies) == activeAllianceRule.MaxNumber {
		return allies
	}
	for i := 0; i < activeAllianceRule.MaxNumber-len(allies); i++ {
		allies = append(allies, models.MatchingAlly{})
	}
	return allies
}
