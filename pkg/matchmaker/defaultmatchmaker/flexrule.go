// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package defaultmatchmaker provides the default implementation of the MatchLogic interface.
// This package contains the core matchmaking algorithms and logic for creating matches from tickets.
package defaultmatchmaker

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

// Now is a variable that holds the current time function.
// This can be overridden for testing purposes.
var Now = time.Now

// isActiveFlexRule checks if a flex rule is active based on the ticket time and flex duration.
// A flex rule is active if the ticket has been in the queue longer than the flex duration.
func isActiveFlexRule(ticketTime time.Time, flexDuration time.Duration) bool {
	return ticketTime.Add(flexDuration).Before(Now())
}

// applyRuleFlexingForSession applies rule flexing to a session based on its oldest ticket.
// This function determines if matching rules need to be adjusted based on how long tickets have been waiting.
func applyRuleFlexingForSession(session models.MatchmakingResult, ruleSet models.RuleSet) (models.RuleSet, bool) {
	if len(ruleSet.FlexingRule) == 0 {
		return ruleSet, false
	}

	// Determine if rule needs flexing based on the pivot time
	oldestTicketTimestamp := time.Unix(session.GetOldestTicketTimestamp(), 0)
	return applyRuleFlexing(ruleSet, oldestTicketTimestamp)
}

// applyRuleFlexing returns a new RuleSet object with updated MatchingRule values.
// The function applies the active flex rule based on the pivotTime to make matching more permissive over time.
func applyRuleFlexing(sourceRuleSet models.RuleSet, pivotTime time.Time) (models.RuleSet, bool) {
	isFlexed := false
	if len(sourceRuleSet.FlexingRule) == 0 {
		return sourceRuleSet, isFlexed
	}

	// We need to deep copy the ruleset,
	//
	// In MatchPlayers() function we have pivotMatching process,
	// when the first iteration of pivotMatching get a pivot ticket with flex rule,
	// if the following next iteration has no flex rule,
	// then the []MatchingRule is being replaced with the flex rule from the previous iteration,
	//
	// so we deep copy sourceRuleSet because we want to update MatchingRule value in ruleset but keep sourceRuleSet as it is
	ruleset := sourceRuleSet.Copy()

	maxDuration := make(map[string]int64)
	for _, flexRule := range ruleset.FlexingRule {
		// Determine active flex rule
		flexDuration := time.Duration(flexRule.Duration) * time.Second
		if isActiveFlexRule(pivotTime, flexDuration) {
			// Choose active rule that has biggest time duration
			if flexRule.Duration < maxDuration[flexRule.Attribute] {
				continue
			}
			maxDuration[flexRule.Attribute] = flexRule.Duration

			// Apply the flexing rule to the matching rule with the same attribute
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
