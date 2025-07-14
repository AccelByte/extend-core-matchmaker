// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

var Now = time.Now

func isActiveFlexRule(ticketTime time.Time, flexDuration time.Duration) bool {
	return ticketTime.Add(flexDuration).Before(Now())
}

func applyRuleFlexingForSession(session models.MatchmakingResult, ruleSet models.RuleSet) (models.RuleSet, bool) {
	if len(ruleSet.FlexingRule) == 0 {
		return ruleSet, false
	}

	// determine if rule needs flexing based on the pivot time
	oldestTicketTimestamp := time.Unix(session.GetOldestTicketTimestamp(), 0)
	return applyRuleFlexing(ruleSet, oldestTicketTimestamp)
}

// applyRuleFlexing return new RuleSet object,
// and update the []MatchingRule values with the active flex rule based on the pivotTime
func applyRuleFlexing(sourceRuleSet models.RuleSet, pivotTime time.Time) (models.RuleSet, bool) {
	isFlexed := false
	if len(sourceRuleSet.FlexingRule) == 0 {
		return sourceRuleSet, isFlexed
	}

	// we need to deep copy the ruleset,
	//
	// in MatchPlayers() function we have pivotMatching process,
	// when the first iteration of pivotMatching get a pivot ticket with flex rule,
	// if the following next iteration has no flex rule,
	// then the []MatchingRule is being replaced with the flex rule from the previous iteration,
	//
	// so we deep copy sourceRuleSet because we want to update MatchingRule value in ruleset but keep sourceRuleSet as it is
	ruleset := sourceRuleSet.Copy()

	maxDuration := make(map[string]int64)
	for _, flexRule := range ruleset.FlexingRule {
		// determine active flex rule
		flexDuration := time.Duration(flexRule.Duration) * time.Second
		if isActiveFlexRule(pivotTime, flexDuration) {
			// choose active rule that has biggest time duration
			if flexRule.Duration < maxDuration[flexRule.Attribute] {
				continue
			}
			maxDuration[flexRule.Attribute] = flexRule.Duration

			for i := range ruleset.MatchingRule {
				if ruleset.MatchingRule[i].Attribute == flexRule.Attribute {
					ruleset.MatchingRule[i].Reference = flexRule.Reference
					ruleset.MatchingRule[i].Criteria = flexRule.Criteria
					isFlexed = true
					break
				}
			}
		}
	}
	return ruleset, isFlexed
}
