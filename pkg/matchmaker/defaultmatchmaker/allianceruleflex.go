// Copyright (c) 2022-2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

func applyAllianceFlexingRulesForSession(session models.MatchmakingResult, ruleset models.RuleSet) (models.RuleSet, bool) {
	if len(ruleset.AllianceFlexingRule) == 0 {
		return ruleset, false
	}
	// determine if rule needs flexing based on the pivot time
	oldestTicketTimestamp := time.Unix(session.GetOldestTicketTimestamp(), 0)
	return applyAllianceFlexingRules(ruleset, oldestTicketTimestamp)
}

func applyAllianceFlexingRules(ruleset models.RuleSet, pivotTime time.Time) (models.RuleSet, bool) {
	isFlexed := false
	ruleset.AllianceRule, isFlexed = ApplyAllianceFlexingRule(ruleset.AllianceRule, ruleset.AllianceFlexingRule, pivotTime)

	return ruleset, isFlexed
}

func ApplyAllianceFlexingRule(allianceRule models.AllianceRule, allianceFlexingRule []models.AllianceFlexingRule, pivotTime time.Time) (models.AllianceRule, bool) {
	isFlexed := false
	var highestDuration int64
	for _, flexRule := range allianceFlexingRule {
		flexDuration := time.Duration(flexRule.Duration) * time.Second
		if isActiveFlexRule(pivotTime, flexDuration) {
			// choose active rule that has biggest time duration
			// this is to makesure that we don't implementing older flex rule.
			// e.g. 10s and 20s flex rule, when we already at 20s, we want to replace the 10s with 20s
			if highestDuration > flexRule.Duration {
				continue
			}
			highestDuration = flexRule.Duration

			allianceRule.MaxNumber = flexRule.MaxNumber
			allianceRule.MinNumber = flexRule.MinNumber
			allianceRule.PlayerMaxNumber = flexRule.PlayerMaxNumber
			allianceRule.PlayerMinNumber = flexRule.PlayerMinNumber
			isFlexed = true
		}
	}

	return allianceRule, isFlexed
}
