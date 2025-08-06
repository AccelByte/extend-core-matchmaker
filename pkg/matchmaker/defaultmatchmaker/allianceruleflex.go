// Copyright (c) 2022-2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package defaultmatchmaker provides the default implementation of the MatchLogic interface.
// This package contains the core matchmaking algorithms and logic for creating matches from tickets.
package defaultmatchmaker

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

// applyAllianceFlexingRulesForSession applies alliance flexing rules to a session.
// This function determines if alliance rules need to be adjusted based on the session's oldest ticket timestamp.
func applyAllianceFlexingRulesForSession(session models.MatchmakingResult, ruleset models.RuleSet) (models.RuleSet, bool) {
	if len(ruleset.AllianceFlexingRule) == 0 {
		return ruleset, false
	}
	// Determine if rule needs flexing based on the pivot time
	oldestTicketTimestamp := time.Unix(session.GetOldestTicketTimestamp(), 0)
	return applyAllianceFlexingRules(ruleset, oldestTicketTimestamp)
}

// applyAllianceFlexingRules applies alliance flexing rules based on a pivot time.
// This function modifies alliance rules to be more permissive as tickets age in the queue.
func applyAllianceFlexingRules(ruleset models.RuleSet, pivotTime time.Time) (models.RuleSet, bool) {
	isFlexed := false
	ruleset.AllianceRule, isFlexed = ApplyAllianceFlexingRule(ruleset.AllianceRule, ruleset.AllianceFlexingRule, pivotTime)

	return ruleset, isFlexed
}

// ApplyAllianceFlexingRule applies alliance flexing rules to an alliance rule based on pivot time.
// This function chooses the most appropriate flexing rule based on the time duration and applies it.
func ApplyAllianceFlexingRule(allianceRule models.AllianceRule, allianceFlexingRule []models.AllianceFlexingRule, pivotTime time.Time) (models.AllianceRule, bool) {
	isFlexed := false
	var highestDuration int64
	for _, flexRule := range allianceFlexingRule {
		flexDuration := time.Duration(flexRule.Duration) * time.Second
		if isActiveFlexRule(pivotTime, flexDuration) {
			// Choose active rule that has biggest time duration
			// This is to make sure that we don't implement older flex rule.
			// e.g. 10s and 20s flex rule, when we already at 20s, we want to replace the 10s with 20s
			if highestDuration > flexRule.Duration {
				continue
			}
			highestDuration = flexRule.Duration

			// Apply the flexing rule values to the alliance rule
			allianceRule.MaxNumber = flexRule.MaxNumber
			allianceRule.MinNumber = flexRule.MinNumber
			allianceRule.PlayerMaxNumber = flexRule.PlayerMaxNumber
			allianceRule.PlayerMinNumber = flexRule.PlayerMinNumber
			isFlexed = true
		}
	}

	return allianceRule, isFlexed
}
