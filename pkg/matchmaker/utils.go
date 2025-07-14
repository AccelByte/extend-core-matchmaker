// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package matchmaker

import (
	"math"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

func DetermineAllianceComposition(ruleSet models.RuleSet, isUseSubGameMode bool) models.AllianceComposition {
	// on rules with sub-game modes, ignore alliance rule
	minTeam := ruleSet.AllianceRule.MinNumber
	maxTeam := ruleSet.AllianceRule.MaxNumber
	maxPlayer := ruleSet.AllianceRule.PlayerMaxNumber
	minPlayer := ruleSet.AllianceRule.PlayerMinNumber
	if isUseSubGameMode {
		minTeam = math.MaxInt16
		maxPlayer = 0
		minPlayer = math.MaxInt16
		for _, SubGameMode := range ruleSet.SubGameModes {
			if SubGameMode.AllianceRule.MinNumber < minTeam {
				minTeam = SubGameMode.AllianceRule.MinNumber
			}
			if SubGameMode.AllianceRule.MaxNumber < maxTeam {
				maxTeam = SubGameMode.AllianceRule.MaxNumber
			}
			if SubGameMode.AllianceRule.PlayerMaxNumber > maxPlayer {
				maxPlayer = SubGameMode.AllianceRule.PlayerMaxNumber
			}
			if SubGameMode.AllianceRule.PlayerMinNumber < minPlayer {
				minPlayer = SubGameMode.AllianceRule.PlayerMinNumber
			}
		}
	}

	return models.AllianceComposition{
		MinTeam:   minTeam,
		MaxTeam:   maxTeam,
		MaxPlayer: maxPlayer,
		MinPlayer: minPlayer,
	}
}
