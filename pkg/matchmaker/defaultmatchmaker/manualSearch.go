// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package defaultmatchmaker provides the default implementation of the MatchLogic interface.
// This package contains the core matchmaking algorithms and logic for creating matches from tickets.
package defaultmatchmaker

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"github.com/elliotchance/pie/v2"
	"gopkg.in/typ.v4/slices"
)

// matchTicket represents a ticket with an associated score for ranking during search
type matchTicket struct {
	ticket models.MatchmakingRequest // The actual matchmaking request
	score  float64                   // Score used for ranking and prioritization
}

// SearchMatchTickets searches for tickets that match the given pivot ticket based on various criteria.
// This is the main search function for finding compatible tickets for matchmaking.
func (mm *MatchMaker) SearchMatchTickets(originalRuleSet, activeRuleSet *models.RuleSet, channel *models.Channel, regionIndex int, pivot *models.MatchmakingRequest, tickets []models.MatchmakingRequest, filteredRegion []models.Region) []models.MatchmakingRequest {
	// Define filters based on the pivot ticket
	distances := getFilterByDistance(activeRuleSet, pivot.PartyAttributes)
	options := getFilterByMatchOption(activeRuleSet, pivot.PartyAttributes)
	anyCrossPlay := getFilterByCrossPlay(activeRuleSet, pivot.PartyAttributes)
	partyAttributes := getFilterByPartyAttribute(activeRuleSet, pivot.PartyAttributes)
	additionCriterias := getFilterByAdditionalCriteria(pivot)
	pivotUserID := pivot.GetMapUserIDs()

	// Remove cross_platform from options if anyCrossPlay has its own matching
	if anyCrossPlay != nil {
		options = pie.Filter(options, func(o option) bool {
			return o.name != models.AttributeCrossPlatform
		})
	}

	// Build sets for user ID and blocked player checking
	userIDSet := pivot.GetMemberUserIDSet()
	blockSet := make(map[string]struct{})
	models.RangeBlockedPlayerUserIDs(pivot.PartyAttributes)(func(userID string) bool {
		blockSet[userID] = struct{}{}
		return true
	})

	// Get the pivot region for latency matching
	var pivotRegion *models.Region
	if len(filteredRegion) > 0 && len(filteredRegion) > regionIndex {
		pivotRegion = &filteredRegion[regionIndex]
	}
	skipFilteredRegion := skipFilterCandidateRegion(pivot, channel)

	// Filter tickets based on various criteria
	matchTickets := make([]matchTicket, 0)
ticketLoop:
	for ticketIndex := range tickets {
		ticket := &tickets[ticketIndex]
		var totalScore float64
		var finalizeFunctions []func()

		// Avoid matching with same party ID
		if ticket.PartyID == pivot.PartyID {
			continue
		}

		// Avoid matching with same user ID
		for _, member := range ticket.PartyMembers {
			if _, ok := pivotUserID[member.UserID]; ok {
				continue ticketLoop
			}
		}

		// Check distance-based matching
		isMatch, score := matchByDistance(ticket, originalRuleSet, distances)
		if !isMatch {
			continue
		}
		totalScore += score

		// Check cross-play compatibility
		if ok, fn := matchByAnyCrossPlay(ticket, anyCrossPlay, mm.isMatchAnyCommon); !ok {
			continue
		} else if fn != nil {
			finalizeFunctions = append(finalizeFunctions, fn)
		}

		// Check match option compatibility
		if ok, fns := matchByMatchOption(ticket, options, mm.isMatchAnyCommon); !ok {
			continue
		} else {
			finalizeFunctions = append(finalizeFunctions, fns...)
		}

		// Check party attribute compatibility
		if !matchByPartyAttribute(ticket, partyAttributes, activeRuleSet.MatchOptionsReferredForBackfill) {
			continue
		}

		// Check blocked players
		if ok, fn := matchByBlockedPlayers(ticket, userIDSet, blockSet, channel.Ruleset.BlockedPlayerOption); !ok {
			continue
		} else if fn != nil {
			finalizeFunctions = append(finalizeFunctions, fn)
		}

		// Check region latency and normalize
		isMatch, score = matchByRegionLatencyNormalize(ticket, pivotRegion, channel, skipFilteredRegion)
		if !isMatch {
			continue
		}
		totalScore += score * channel.Ruleset.GetRegionLatencyRuleWeight()

		// Check additional criteria
		if !matchByAdditionalCriteria(ticket, additionCriterias) {
			continue
		}

		// Add ticket to results if all criteria are met
		matchTickets = append(matchTickets, matchTicket{
			ticket: *ticket,
			score:  totalScore,
		})

		// Call finalize functions for matched ticket
		for _, fn := range finalizeFunctions {
			fn()
		}
	}

	// Sort tickets based on priority, score, and latency
	sortMatchTickets(matchTickets, "")

	// Convert matchTickets to models.MatchmakingRequest slice
	results := make([]models.MatchmakingRequest, len(matchTickets))
	for i, v := range matchTickets {
		results[i] = v.ticket
	}

	return results
}

func (mm *MatchMaker) SearchMatchTicketsBySession(rootScope *envelope.Scope, originalRuleSet, activeRuleSet *models.RuleSet, channel *models.Channel, session models.MatchmakingResult, tickets []models.MatchmakingRequest) []models.MatchmakingRequest {
	scope := rootScope.NewChildScope("ManualSearch.SearchMatchTicketsBySession")
	defer scope.Finish()

	// Define filter by pivot
	distances := getFilterByDistance(activeRuleSet, session.PartyAttributes)
	options := getFilterByMatchOption(activeRuleSet, session.PartyAttributes)
	anyCrossPlay := getFilterByCrossPlay(activeRuleSet, session.PartyAttributes)
	partyAttributes := getFilterByPartyAttribute(activeRuleSet, session.PartyAttributes)
	sessionPartyIDs := session.GetMapPartyIDs()
	sessionUserID := session.GetMapUserIDs()

	if anyCrossPlay != nil {
		// remove cross_platform from options, anyCrossPlay have its own matching
		options = pie.Filter(options, func(o option) bool {
			return o.name != models.AttributeCrossPlatform
		})
	}

	userIDSet := session.GetMemberUserIDSet()
	blockSet := make(map[string]struct{})
	models.RangeBlockedPlayerUserIDs(session.PartyAttributes)(func(userID string) bool {
		blockSet[userID] = struct{}{}
		return true
	})

	// filter tickets
	matchTickets := make([]matchTicket, 0)
ticketLoop:
	for ticketIndex := range tickets {
		ticket := &tickets[ticketIndex]
		var totalScore float64
		var finalizeFunctions []func()

		// avoid match with same party id
		if _, ok := sessionPartyIDs[ticket.PartyID]; ok {
			continue
		}

		// avoid match with excluded sessions
		allowMatch := true
		for _, t := range ticket.ExcludedSessions {
			if t == session.MatchSessionID || t == session.PartyID {
				allowMatch = false
				break
			}
		}
		if !allowMatch {
			continue
		}

		// avoid match with same user id
		for _, member := range ticket.PartyMembers {
			if _, ok := sessionUserID[member.UserID]; ok {
				continue ticketLoop
			}
		}

		// Filter by session region
		if session.Region != "" {
			filteredRegions := filterRegionByStep(ticket, channel)
			regionIdx := slices.IndexFunc(filteredRegions, func(item models.Region) bool { return item.Region == session.Region })
			if regionIdx < 0 {
				continue
			}
		}

		// Check distance-based matching
		isMatch, score := matchByDistance(ticket, originalRuleSet, distances)
		if !isMatch {
			continue
		}
		totalScore += score

		// Check cross-play compatibility
		if ok, fn := matchByAnyCrossPlay(ticket, anyCrossPlay, mm.isMatchAnyCommon); !ok {
			continue
		} else if fn != nil {
			finalizeFunctions = append(finalizeFunctions, fn)
		}

		// Check match option compatibility
		if ok, fns := matchByMatchOption(ticket, options, mm.isMatchAnyCommon); !ok {
			continue
		} else {
			for _, fn := range fns {
				finalizeFunctions = append(finalizeFunctions, fn)
			}
		}

		// Check party attribute compatibility
		if !matchByPartyAttribute(ticket, partyAttributes, activeRuleSet.MatchOptionsReferredForBackfill) {
			continue
		}

		// Check blocked players
		if ok, fn := matchByBlockedPlayers(ticket, userIDSet, blockSet, channel.Ruleset.BlockedPlayerOption); !ok {
			continue
		} else if fn != nil {
			finalizeFunctions = append(finalizeFunctions, fn)
		}

		// Check server name compatibility
		if session.ServerName != "" {
			if session.ServerName != ticket.PartyAttributes[models.AttributeServerName] {
				continue
			}
		}

		// Check client version compatibility
		if session.ClientVersion != "" {
			if session.ClientVersion != ticket.PartyAttributes[models.AttributeClientVersion] {
				continue
			}
		}

		// Add ticket to results if all criteria are met
		matchTickets = append(matchTickets, matchTicket{
			ticket: *ticket,
			score:  totalScore,
		})

		// Call finalize function for matched ticket
		for _, fn := range finalizeFunctions {
			fn()
		}
	}

	// Sort tickets based on priority, score, and latency
	sortMatchTickets(matchTickets, session.Region)

	// Convert matchTickets to models.MatchmakingRequest slice
	results := make([]models.MatchmakingRequest, len(matchTickets))
	for i, v := range matchTickets {
		results[i] = v.ticket
	}

	return results
}

// sortMatchTickets sorts match tickets based on multiple criteria.
// This function prioritizes tickets by priority (descending), score (ascending), latency (ascending), and creation time (ascending).
func sortMatchTickets(matchTickets []matchTicket, sessionRegion string) {
	sort.Slice(matchTickets, func(i, j int) bool {
		// Consider priority first (DESC)
		if matchTickets[i].ticket.Priority != matchTickets[j].ticket.Priority {
			return matchTickets[i].ticket.Priority > matchTickets[j].ticket.Priority
		}
		// Then, score (ASC)
		if matchTickets[i].score != matchTickets[j].score {
			return matchTickets[i].score < matchTickets[j].score
		}
		// Then, latency for matchSession (ASC)
		if sessionRegion != "" {
			iLatency := matchTickets[i].ticket.LatencyMap[sessionRegion]
			jLatency := matchTickets[j].ticket.LatencyMap[sessionRegion]
			return iLatency < jLatency
		}
		// Then, createdAt (ASC)
		return matchTickets[i].ticket.CreatedAt < matchTickets[j].ticket.CreatedAt
	})
}

// distance represents a distance-based matching criterion.
// This structure contains the attribute name, value range, and weight for distance-based matching.
type distance struct {
	attribute         string   // The attribute name to match on
	value             float64  // The pivot value
	min               float64  // Minimum acceptable value
	max               float64  // Maximum acceptable value
	attributeMaxValue float64  // Maximum value for normalization
	weight            *float64 // Weight for scoring
}

// getWeight returns the weight value for this distance criterion.
// If no weight is specified, returns the default weight value.
func (d distance) getWeight() float64 {
	if d.weight == nil {
		return models.DefaultWeightValue
	}
	return *d.weight
}

// getFilterByDistance extracts distance-based matching criteria from party attributes.
// This function creates distance structures for each matching rule that uses distance criteria.
func getFilterByDistance(activeRuleSet *models.RuleSet, partyAttributes map[string]interface{}) []distance {
	memberAttributes, ok := partyAttributes[memberAttributesKey].(map[string]interface{})
	if !ok {
		return nil
	}
	distances := make([]distance, 0)
	for _, rule := range activeRuleSet.MatchingRule {
		if rule.Criteria == distanceCriteria {
			value, ok := memberAttributes[rule.Attribute].(float64)
			if !ok {
				continue
			}
			distance := distance{
				attribute: rule.Attribute,
				value:     value,
				min:       value - rule.Reference,
				max:       value + rule.Reference,
				weight:    rule.Weight,
			}
			if rule.NormalizationMax > 0 {
				distance.attributeMaxValue = rule.NormalizationMax
			}
			distances = append(distances, distance)
		}
	}
	return distances
}

// matchByDistance checks if a ticket matches distance-based criteria and returns a score.
// The smaller the score, the better the match. This function considers rule flexing for aging tickets.
func matchByDistance(ticket *models.MatchmakingRequest, originalRuleSet *models.RuleSet, distances []distance) (isMatch bool, score float64) {
	if len(distances) == 0 {
		return true, 0.0
	}

	// Apply rule flexing for the candidate ticket based on its age
	candidateTimeStampRequest := time.Unix(ticket.CreatedAt, 0)
	candidateRuleSet, _ := applyRuleFlexing(*originalRuleSet, candidateTimeStampRequest)
	candidateDistances := getFilterByDistance(&candidateRuleSet, ticket.PartyAttributes)

	memberAttributes, ok := ticket.PartyAttributes[memberAttributesKey].(map[string]interface{})
	if !ok {
		return false, 0.0
	}
	for _, distance := range distances {
		value, ok := memberAttributes[distance.attribute].(float64)
		if !ok {
			return false, 0.0
		}

		// Find corresponding candidate distance for bidirectional checking
		candidateDistanceIndex := -1
		for i := range candidateDistances {
			if candidateDistances[i].attribute == distance.attribute {
				candidateDistanceIndex = i
				break
			}
		}

		// Check if value is within acceptable range
		if value < distance.min {
			return false, 0.0
		}
		if value > distance.max {
			return false, 0.0
		}
		// Bidirectional check if candidate distance exists
		if candidateDistanceIndex >= 0 {
			if distance.value < candidateDistances[candidateDistanceIndex].min {
				return false, 0.0
			}
			if distance.value > candidateDistances[candidateDistanceIndex].max {
				return false, 0.0
			}
		}
		// Calculate score based on distance difference
		if distance.attributeMaxValue > 0 {
			score += (math.Abs(value-distance.value) / distance.attributeMaxValue) * distance.getWeight()
		} else {
			score += math.Abs(value - distance.value)
		}
	}
	return true, score
}

// option represents a match option criterion.
// This structure contains the option name, type, and values for option-based matching.
type option struct {
	name   string   // The option name
	types  string   // The option type (Any, All, Unique, etc.)
	values []string // The acceptable values for this option
}

// getFilterByMatchOption extracts match option criteria from party attributes.
// This function creates option structures for each match option in the ruleset.
func getFilterByMatchOption(activeRuleSet *models.RuleSet, partyAttributes map[string]interface{}) []option {
	options := make([]option, 0)
	for _, v := range activeRuleSet.MatchOptions.Options {
		if v.Type == models.MatchOptionTypeDisable {
			continue
		}
		optVal, ok := partyAttributes[v.Name]
		if !ok {
			continue
		}
		// Handle both single values and arrays
		multiVal, ok := optVal.([]interface{})
		if !ok {
			multiVal = append(multiVal, optVal)
		}
		values := make([]string, len(multiVal))
		for i, val := range multiVal {
			valStr, ok := val.(string)
			if !ok {
				valStr = fmt.Sprint(val)
			}
			values[i] = valStr
		}
		options = append(options, option{
			name:   v.Name,
			types:  v.Type,
			values: values,
		})
	}
	return options
}

// matchByMatchOption checks if a ticket matches option-based criteria.
// This function handles different option types (Any, All, Unique) and returns finalize functions for updates.
func matchByMatchOption(ticket *models.MatchmakingRequest, options []option, flagAnyMatchOptionAllCommon bool) (ok bool, fns []func()) {
	if len(options) == 0 {
		return true, fns
	}
	for i, option := range options {
		optVal, ok := ticket.PartyAttributes[option.name]
		if !ok {
			return false, fns
		}
		// Handle both single values and arrays
		multiVal, ok := optVal.([]interface{})
		if !ok {
			multiVal = append(multiVal, optVal)
		}
		mapVal := make(map[string]struct{})
		for _, val := range multiVal {
			valStr, ok := val.(string)
			if !ok {
				valStr = fmt.Sprint(val)
			}
			mapVal[valStr] = struct{}{}
		}
		switch option.types {
		case models.MatchOptionTypeAny:
			var exist bool
			if flagAnyMatchOptionAllCommon {
				// Find common values between pivot and candidate
				commonValueSet := make(map[string]struct{})
				for _, v := range option.values {
					if _, ok := mapVal[v]; ok {
						exist = true
						commonValueSet[v] = struct{}{}
					}
				}
				if exist {
					commonValue := make([]string, 0, len(commonValueSet))
					for v := range commonValueSet {
						commonValue = append(commonValue, v)
					}
					n := i
					fns = append(fns, func() {
						// Update option with common value
						options[n].values = commonValue
					})
				}
			} else {
				// Check if any value matches
				for _, v := range option.values {
					if _, ok := mapVal[v]; ok {
						exist = true
						break
					}
				}
			}

			if !exist {
				return false, fns
			}
		case models.MatchOptionTypeAll:
			// Check if all values are present
			var notExist bool
			for _, v := range option.values {
				if _, ok := mapVal[v]; !ok {
					notExist = true
					break
				}
			}
			if notExist {
				return false, fns
			}
		case models.MatchOptionTypeUnique:
			// Check that no values are common (all must be unique)
			var exist bool
			for _, v := range option.values {
				if _, ok := mapVal[v]; ok {
					exist = true
					break
				}
			}
			if exist {
				return false, fns
			}
		}
	}
	return true, fns
}

// crossPlayAttributes represents cross-play matching criteria.
// This structure contains the platforms that a party wants to play with and their current platform.
type crossPlayAttributes struct {
	wantPlatforms    map[string]struct{} // Platforms the party wants to play with
	currentPlatforms map[string]struct{} // Current platform of the party
}

// getFilterByCrossPlay extracts cross-play matching criteria from party attributes.
// This function checks if cross-platform matching is enabled and extracts platform preferences.
func getFilterByCrossPlay(activeRuleSet *models.RuleSet, partyAttributes map[string]interface{}) *crossPlayAttributes {

	anyCrossPlatform := false
	for _, v := range activeRuleSet.MatchOptions.Options {
		if v.Type == models.MatchOptionTypeDisable {
			continue
		}
		if v.Name == models.AttributeCrossPlatform {
			anyCrossPlatform = v.Type == models.MatchOptionTypeAny
			break
		}
	}

	if anyCrossPlatform {
		wantPlatforms := multiValueMapString(partyAttributes, models.AttributeCrossPlatform)
		currentPlatforms := multiValueMapString(partyAttributes, models.AttributeCurrentPlatform)
		if len(currentPlatforms) == 0 {
			// This empty when CrossPlatformNoCurrentPlatform=true
			return nil
		}

		return &crossPlayAttributes{wantPlatforms: wantPlatforms, currentPlatforms: currentPlatforms}
	}
	return nil
}

// matchByAnyCrossPlay checks if a ticket matches cross-play criteria.
// This function ensures bidirectional compatibility between platforms.
func matchByAnyCrossPlay(ticket *models.MatchmakingRequest, anyCrossPlay *crossPlayAttributes, flagAnyMatchOptionAllCommon bool) (ok bool, finalizeFn func()) {
	if anyCrossPlay == nil {
		return true, finalizeFn
	}

	wantPlatforms := multiValueMapString(ticket.PartyAttributes, models.AttributeCrossPlatform)
	currentPlatforms := multiValueMapString(ticket.PartyAttributes, models.AttributeCurrentPlatform)

	if len(currentPlatforms) == 0 {
		return false, finalizeFn
	}

	// Should satisfy both directions
	for platform := range anyCrossPlay.currentPlatforms {
		if _, ok := wantPlatforms[platform]; !ok {
			return false, finalizeFn
		}
	}
	for platform := range currentPlatforms {
		if _, ok := anyCrossPlay.wantPlatforms[platform]; !ok {
			return false, finalizeFn
		}
	}

	if flagAnyMatchOptionAllCommon {
		differentPlatforms := make([]string, 0)
		for platform := range anyCrossPlay.wantPlatforms {
			if _, ok := wantPlatforms[platform]; !ok {
				differentPlatforms = append(differentPlatforms, platform)
			}
		}

		// Should not happen, just in case
		if len(differentPlatforms) == len(anyCrossPlay.wantPlatforms) {
			return false, finalizeFn
		}

		// Called when all other conditions are pass
		finalizeFn = func() {
			// Keep only common values
			for _, platform := range differentPlatforms {
				delete(anyCrossPlay.wantPlatforms, platform)
			}

			for platform := range currentPlatforms {
				anyCrossPlay.currentPlatforms[platform] = struct{}{}
			}
		}
	}

	return true, finalizeFn
}

// multiValueMapString converts a party attribute to a set of strings.
// This function handles both single values and arrays, converting them to a map for efficient lookup.
func multiValueMapString(partyAttributes map[string]interface{}, name string) map[string]struct{} {
	optVal, ok := partyAttributes[name]
	if !ok {
		return nil
	}
	// Handle both single values and arrays
	multiVal, ok := optVal.([]interface{})
	if !ok {
		multiVal = append(multiVal, optVal)
	}

	mapVal := make(map[string]struct{})
	for _, val := range multiVal {
		valStr, ok := val.(string)
		if !ok {
			valStr = fmt.Sprint(val)
		}
		mapVal[valStr] = struct{}{}
	}
	return mapVal
}

// partyAttribute represents a party attribute matching criterion.
// This structure contains the attribute key and value for party attribute matching.
type partyAttribute struct {
	key   string      // The attribute key
	value interface{} // The expected value
}

// getFilterByPartyAttribute extracts party attribute matching criteria.
// This function filters out match options and special attributes, keeping only relevant party attributes.
func getFilterByPartyAttribute(activeRuleSet *models.RuleSet, partyAttributes map[string]interface{}) []partyAttribute {
	result := make([]partyAttribute, 0)
	optionsMap := make(map[string]struct{})
	for _, option := range activeRuleSet.MatchOptions.Options {
		optionsMap[option.Name] = struct{}{}
	}
	for key, value := range partyAttributes {
		// Ignore match options
		if _, ok := optionsMap[key]; ok {
			continue
		}
		switch key {
		// Ignore these keys
		case models.AttributeMatchAttempt:
		case models.AttributeLatencies:
		case models.AttributeMemberAttr:
		case models.AttributeSubGameMode:
		case models.AttributeBlockedPlayersDetail:
		case models.AttributeNewSessionOnly:
		case models.ROLE:
		case models.AttributeCrossPlatform:
			// Ignore if no matchoption in ruleset
			if activeRuleSet.MatchOptions.Options != nil {
				result = append(result, partyAttribute{
					key:   key,
					value: value,
				})
			}
		default:
			result = append(result, partyAttribute{
				key:   key,
				value: value,
			})
		}
	}
	return result
}

// matchByPartyAttribute checks if a ticket matches party attribute criteria.
// This function handles different types of party attributes including blocked players, server name, and client version.
func matchByPartyAttribute(ticket *models.MatchmakingRequest, partyAttributes []partyAttribute, matchOptionsReferredForBackfill bool) bool {
	if len(partyAttributes) == 0 {
		return true
	}
	for _, v := range partyAttributes {
		switch v.key {
		// Handle blocked players
		case models.AttributeBlocked:
			// Handled on different function

		// Handle server name and client version
		case models.AttributeServerName, models.AttributeClientVersion:
			if v.value == nil || v.value == "" {
				if ticket.PartyAttributes[v.key] == nil || ticket.PartyAttributes[v.key] == "" {
					continue
				}
			}
			if !reflect.DeepEqual(ticket.PartyAttributes[v.key], v.value) {
				return false
			}

		default:
			if !matchOptionsReferredForBackfill {
				// Handle other attributes as "must match this attribute"
				if v.value != "" {
					if !reflect.DeepEqual(ticket.PartyAttributes[v.key], v.value) {
						return false
					}
				}
			}
		}
	}
	return true
}

// matchByBlockedPlayers checks if a ticket can be matched based on blocked player criteria.
// This function handles different blocked player options and returns finalize functions for updating blocked player sets.
func matchByBlockedPlayers(ticket *models.MatchmakingRequest, userIDSet, blockSet map[string]struct{}, blockedPlayerOption models.BlockedPlayerOption) (ok bool, finalizeFn func()) {
	if blockedPlayerOption == models.BlockedPlayerCanMatchOnDifferentTeam ||
		blockedPlayerOption == models.BlockedPlayerCanMatch {
		return true, finalizeFn
	} else {
		blocked := false
		forBlockedUserID := models.RangeBlockedPlayerUserIDs(ticket.PartyAttributes)
		// Check if previous matched tickets already contains blocked player
		forBlockedUserID(func(userID string) bool {
			if _, ok = userIDSet[userID]; ok {
				blocked = true
				return false
			}
			return true
		})

		if blocked {
			return false, finalizeFn
		}

		// Check if the new ticket's member contains blocked players
		for _, member := range ticket.PartyMembers {
			if _, ok = blockSet[member.UserID]; ok {
				return false, finalizeFn
			}
		}

		// Called when all others condition are pass
		finalizeFn = func() {
			// Update block data
			forBlockedUserID(func(userID string) bool {
				blockSet[userID] = struct{}{}
				return true
			})
			for _, member := range ticket.PartyMembers {
				userIDSet[member.UserID] = struct{}{}
			}
		}
	}

	return true, finalizeFn
}

// matchByRegionLatencyNormalize checks region latency with normalized scoring.
// This function normalizes the latency score by dividing by the maximum allowed latency.
func matchByRegionLatencyNormalize(ticket *models.MatchmakingRequest, pivotRegion *models.Region, channel *models.Channel, skipRegionFilter bool) (bool, float64) {
	match, score := matchByRegionLatency(ticket, pivotRegion, channel, skipRegionFilter)
	var maxLatency int = 0
	if channel.Ruleset.RegionLatencyMaxMs > 0 {
		maxLatency = channel.Ruleset.RegionLatencyMaxMs
	}
	if maxLatency == 0 {
		return match, 0
	}
	return match, score / float64(maxLatency)
}

// matchByRegionLatency checks if a ticket matches region latency criteria.
// This function calculates a score based on the difference between the region's latency and the best available latency.
func matchByRegionLatency(ticket *models.MatchmakingRequest, pivotRegion *models.Region, channel *models.Channel, skipRegionFilter bool) (bool, float64) {
	if pivotRegion == nil {
		return true, 0
	}
	var filteredRegion []models.Region
	if skipRegionFilter {
		filteredRegion = ticket.SortedLatency
	} else {
		filteredRegion = filterRegionByStep(ticket, channel)
	}

	if len(filteredRegion) == 0 {
		return true, 0
	}

	// Find the pivot region in the filtered regions
	for i := range filteredRegion {
		if filteredRegion[i].Region == pivotRegion.Region {
			// Score = abs(candidate region latency - candidate best latency)
			return true, float64(filteredRegion[i].Latency - filteredRegion[0].Latency)
		}
	}

	return false, 0
}

// additionalCriteria represents additional matching criteria.
// This structure contains the criteria name and value for additional matching.
type additionalCriteria struct {
	name  string      // The criteria name
	value interface{} // The expected value
}

// getFilterByAdditionalCriteria extracts additional criteria from a pivot ticket.
// This function creates additional criteria structures from the pivot's additional criteria map.
func getFilterByAdditionalCriteria(pivot *models.MatchmakingRequest) []additionalCriteria {
	additionalCriterias := make([]additionalCriteria, 0)
	for k, v := range pivot.AdditionalCriterias {
		additionalCriterias = append(additionalCriterias, additionalCriteria{
			name:  k,
			value: v,
		})
	}
	return additionalCriterias
}

// matchByAdditionalCriteria checks if a ticket matches additional criteria.
// This function ensures that all additional criteria values match exactly.
func matchByAdditionalCriteria(ticket *models.MatchmakingRequest, additionalCriterias []additionalCriteria) bool {
	for _, v := range additionalCriterias {
		value, ok := ticket.AdditionalCriterias[v.name]
		if !ok {
			return false
		}
		if value != v.value {
			return false
		}
	}
	return true
}
