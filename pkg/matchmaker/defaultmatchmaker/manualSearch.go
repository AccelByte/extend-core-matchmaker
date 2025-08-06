// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

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

type matchTicket struct {
	ticket models.MatchmakingRequest
	score  float64
}

func (mm *MatchMaker) SearchMatchTickets(originalRuleSet, activeRuleSet *models.RuleSet, channel *models.Channel, regionIndex int, pivot *models.MatchmakingRequest, tickets []models.MatchmakingRequest, filteredRegion []models.Region) []models.MatchmakingRequest {
	// define filter by pivot
	distances := getFilterByDistance(activeRuleSet, pivot.PartyAttributes)
	options := getFilterByMatchOption(activeRuleSet, pivot.PartyAttributes)
	anyCrossPlay := getFilterByCrossPlay(activeRuleSet, pivot.PartyAttributes)
	partyAttributes := getFilterByPartyAttribute(activeRuleSet, pivot.PartyAttributes)
	additionCriterias := getFilterByAdditionalCriteria(pivot)
	pivotUserID := pivot.GetMapUserIDs()

	if anyCrossPlay != nil {
		// remove cross_platform from options, anyCrossPlay have its own matching
		options = pie.Filter(options, func(o option) bool {
			return o.name != models.AttributeCrossPlatform
		})
	}

	userIDSet := pivot.GetMemberUserIDSet()
	blockSet := make(map[string]struct{})
	models.RangeBlockedPlayerUserIDs(pivot.PartyAttributes)(func(userID string) bool {
		blockSet[userID] = struct{}{}
		return true
	})

	var pivotRegion *models.Region
	if len(filteredRegion) > 0 && len(filteredRegion) > regionIndex {
		pivotRegion = &filteredRegion[regionIndex]
	}
	skipFilteredRegion := skipFilterCandidateRegion(pivot, channel)

	// filter tickets
	matchTickets := make([]matchTicket, 0)
ticketLoop:
	for ticketIndex := range tickets {
		ticket := &tickets[ticketIndex]
		var totalScore float64
		var finalizeFunctions []func()

		// avoid match with same party id
		if ticket.PartyID == pivot.PartyID {
			continue
		}

		// avoid match with same user id
		for _, member := range ticket.PartyMembers {
			if _, ok := pivotUserID[member.UserID]; ok {
				continue ticketLoop
			}
		}

		isMatch, score := matchByDistance(ticket, originalRuleSet, distances)
		if !isMatch {
			continue
		}
		totalScore += score

		if ok, fn := matchByAnyCrossPlay(ticket, anyCrossPlay, mm.isMatchAnyCommon); !ok {
			continue
		} else if fn != nil {
			finalizeFunctions = append(finalizeFunctions, fn)
		}

		if ok, fns := matchByMatchOption(ticket, options, mm.isMatchAnyCommon); !ok {
			continue
		} else {
			finalizeFunctions = append(finalizeFunctions, fns...)
		}

		if !matchByPartyAttribute(ticket, partyAttributes, activeRuleSet.MatchOptionsReferredForBackfill) {
			continue
		}

		if ok, fn := matchByBlockedPlayers(ticket, userIDSet, blockSet, channel.Ruleset.BlockedPlayerOption); !ok {
			continue
		} else if fn != nil {
			finalizeFunctions = append(finalizeFunctions, fn)
		}

		isMatch, score = matchByRegionLatencyNormalize(ticket, pivotRegion, channel, skipFilteredRegion)
		if !isMatch {
			continue
		}
		totalScore += score * channel.Ruleset.GetRegionLatencyRuleWeight()

		if !matchByAdditionalCriteria(ticket, additionCriterias) {
			continue
		}

		matchTickets = append(matchTickets, matchTicket{
			ticket: *ticket,
			score:  totalScore,
		})

		// call finalize function for matched ticket
		for _, fn := range finalizeFunctions {
			fn()
		}
	}

	// with rule (THE SMALLER THE BETTER)
	// - matchByDistance (ex: MMR)
	// - matchByRegionLatency
	// - firstCreatedAt
	// - matchByMatchOption (especially for ANY)
	// - etc

	sortMatchTickets(matchTickets, "")

	results := make([]models.MatchmakingRequest, len(matchTickets))
	for i, v := range matchTickets {
		results[i] = v.ticket
	}

	return results
}

func (mm *MatchMaker) SearchMatchTicketsBySession(rootScope *envelope.Scope, originalRuleSet, activeRuleSet *models.RuleSet, channel *models.Channel, session models.MatchmakingResult, tickets []models.MatchmakingRequest) []models.MatchmakingRequest {
	scope := rootScope.NewChildScope("ManualSearch.SearchMatchTicketsBySession")
	defer scope.Finish()

	// define filter by pivot
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

		// filter by session region
		if session.Region != "" {
			filteredRegions := filterRegionByStep(ticket, channel)
			regionIdx := slices.IndexFunc(filteredRegions, func(item models.Region) bool { return item.Region == session.Region })
			if regionIdx < 0 {
				continue
			}
		}

		isMatch, score := matchByDistance(ticket, originalRuleSet, distances)
		if !isMatch {
			continue
		}
		totalScore += score

		if ok, fn := matchByAnyCrossPlay(ticket, anyCrossPlay, mm.isMatchAnyCommon); !ok {
			continue
		} else if fn != nil {
			finalizeFunctions = append(finalizeFunctions, fn)
		}

		if ok, fns := matchByMatchOption(ticket, options, mm.isMatchAnyCommon); !ok {
			continue
		} else {
			for _, fn := range fns {
				finalizeFunctions = append(finalizeFunctions, fn)
			}
		}

		if !matchByPartyAttribute(ticket, partyAttributes, activeRuleSet.MatchOptionsReferredForBackfill) {
			continue
		}

		if ok, fn := matchByBlockedPlayers(ticket, userIDSet, blockSet, channel.Ruleset.BlockedPlayerOption); !ok {
			continue
		} else if fn != nil {
			finalizeFunctions = append(finalizeFunctions, fn)
		}

		if session.ServerName != "" {
			if session.ServerName != ticket.PartyAttributes[models.AttributeServerName] {
				continue
			}
		}

		if session.ClientVersion != "" {
			if session.ClientVersion != ticket.PartyAttributes[models.AttributeClientVersion] {
				continue
			}
		}

		matchTickets = append(matchTickets, matchTicket{
			ticket: *ticket,
			score:  totalScore,
		})

		// call finalize function for matched ticket
		for _, fn := range finalizeFunctions {
			fn()
		}
	}

	sortMatchTickets(matchTickets, session.Region)

	results := make([]models.MatchmakingRequest, len(matchTickets))
	for i, v := range matchTickets {
		results[i] = v.ticket
	}

	return results
}

func sortMatchTickets(matchTickets []matchTicket, sessionRegion string) {
	sort.Slice(matchTickets, func(i, j int) bool {
		// consider priority first (DESC)
		if matchTickets[i].ticket.Priority != matchTickets[j].ticket.Priority {
			return matchTickets[i].ticket.Priority > matchTickets[j].ticket.Priority
		}
		// then, score (ASC)
		if matchTickets[i].score != matchTickets[j].score {
			return matchTickets[i].score < matchTickets[j].score
		}
		// then, latency for matchSession (ASC)
		if sessionRegion != "" {
			iLatency := matchTickets[i].ticket.LatencyMap[sessionRegion]
			jLatency := matchTickets[j].ticket.LatencyMap[sessionRegion]
			return iLatency < jLatency
		}
		// then, createdAt (ASC)
		return matchTickets[i].ticket.CreatedAt < matchTickets[j].ticket.CreatedAt
	})
}

type distance struct {
	attribute         string
	value             float64
	min               float64
	max               float64
	attributeMaxValue float64
	weight            *float64
}

func (d distance) getWeight() float64 {
	if d.weight == nil {
		return models.DefaultWeightValue
	}
	return *d.weight
}

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

// matchByDistance with score, the smaller the better
func matchByDistance(ticket *models.MatchmakingRequest, originalRuleSet *models.RuleSet, distances []distance) (isMatch bool, score float64) {
	if len(distances) == 0 {
		return true, 0.0
	}

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

		candidateDistanceIndex := -1
		for i := range candidateDistances {
			if candidateDistances[i].attribute == distance.attribute {
				candidateDistanceIndex = i
				break
			}
		}

		if value < distance.min {
			return false, 0.0
		}
		if value > distance.max {
			return false, 0.0
		}
		if candidateDistanceIndex >= 0 {
			if distance.value < candidateDistances[candidateDistanceIndex].min {
				return false, 0.0
			}
			if distance.value > candidateDistances[candidateDistanceIndex].max {
				return false, 0.0
			}
		}
		if distance.attributeMaxValue > 0 {
			score += (math.Abs(value-distance.value) / distance.attributeMaxValue) * distance.getWeight()
		} else {
			score += math.Abs(value - distance.value)
		}
	}
	return true, score
}

type option struct {
	name   string
	types  string
	values []string
}

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

func matchByMatchOption(ticket *models.MatchmakingRequest, options []option, flagAnyMatchOptionAllCommon bool) (ok bool, fns []func()) {
	if len(options) == 0 {
		return true, fns
	}
	for i, option := range options {
		optVal, ok := ticket.PartyAttributes[option.name]
		if !ok {
			return false, fns
		}
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
						// update option with common value
						options[n].values = commonValue
					})
				}
			} else {
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

type crossPlayAttributes struct {
	wantPlatforms    map[string]struct{}
	currentPlatforms map[string]struct{}
}

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
			// this empty when CrossPlatformNoCurrentPlatform=true
			return nil
		}

		return &crossPlayAttributes{wantPlatforms: wantPlatforms, currentPlatforms: currentPlatforms}
	}
	return nil
}

func matchByAnyCrossPlay(ticket *models.MatchmakingRequest, anyCrossPlay *crossPlayAttributes, flagAnyMatchOptionAllCommon bool) (ok bool, finalizeFn func()) {
	if anyCrossPlay == nil {
		return true, finalizeFn
	}

	wantPlatforms := multiValueMapString(ticket.PartyAttributes, models.AttributeCrossPlatform)
	currentPlatforms := multiValueMapString(ticket.PartyAttributes, models.AttributeCurrentPlatform)

	if len(currentPlatforms) == 0 {
		return false, finalizeFn
	}

	// should satisfy both directions
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

		// should not happen, just in case
		if len(differentPlatforms) == len(anyCrossPlay.wantPlatforms) {
			return false, finalizeFn
		}

		// called when all other conditions are pass
		finalizeFn = func() {
			// keep only common values
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

func multiValueMapString(partyAttributes map[string]interface{}, name string) map[string]struct{} {
	optVal, ok := partyAttributes[name]
	if !ok {
		return nil
	}
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

type partyAttribute struct {
	key   string
	value interface{}
}

func getFilterByPartyAttribute(activeRuleSet *models.RuleSet, partyAttributes map[string]interface{}) []partyAttribute {
	result := make([]partyAttribute, 0)
	optionsMap := make(map[string]struct{})
	for _, option := range activeRuleSet.MatchOptions.Options {
		optionsMap[option.Name] = struct{}{}
	}
	for key, value := range partyAttributes {
		// ignore match options
		if _, ok := optionsMap[key]; ok {
			continue
		}
		switch key {
		// ignore these keys
		case models.AttributeMatchAttempt:
		case models.AttributeLatencies:
		case models.AttributeMemberAttr:
		case models.AttributeSubGameMode:
		case models.AttributeBlockedPlayersDetail:
		case models.AttributeNewSessionOnly:
		case models.ROLE:
		case models.AttributeCrossPlatform:
			// ignore if no matchoption in ruleset
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

func matchByPartyAttribute(ticket *models.MatchmakingRequest, partyAttributes []partyAttribute, matchOptionsReferredForBackfill bool) bool {
	if len(partyAttributes) == 0 {
		return true
	}
	for _, v := range partyAttributes {
		switch v.key {
		// handle blocked players
		case models.AttributeBlocked:
			// handled on different function

		// handle server name and client version
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
				// handle other attributes as "must match this attribute"
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

func matchByBlockedPlayers(ticket *models.MatchmakingRequest, userIDSet, blockSet map[string]struct{}, blockedPlayerOption models.BlockedPlayerOption) (ok bool, finalizeFn func()) {
	if blockedPlayerOption == models.BlockedPlayerCanMatchOnDifferentTeam ||
		blockedPlayerOption == models.BlockedPlayerCanMatch {
		return true, finalizeFn
	} else {
		blocked := false
		forBlockedUserID := models.RangeBlockedPlayerUserIDs(ticket.PartyAttributes)
		// for userID := range models.RangeBlockedPlayerUserIDs(ticket.PartyAttributes)
		forBlockedUserID(func(userID string) bool {
			// check if previous matched tickets already contains blocked player
			if _, ok = userIDSet[userID]; ok {
				blocked = true
				return false
			}
			return true
		})

		if blocked {
			return false, finalizeFn
		}

		// check if the new ticket's member contains blocked players
		for _, member := range ticket.PartyMembers {
			if _, ok = blockSet[member.UserID]; ok {
				return false, finalizeFn
			}
		}

		// called when all others condition are pass
		finalizeFn = func() {
			// update block data
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

	for i := range filteredRegion {
		if filteredRegion[i].Region == pivotRegion.Region {
			// score = abs( candidate region latency - candidate best latency )
			return true, float64(filteredRegion[i].Latency - filteredRegion[0].Latency)
		}
	}

	return false, 0
}

type additionalCriteria struct {
	name  string
	value interface{}
}

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
