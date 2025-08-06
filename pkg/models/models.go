// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"

	//"accelbyte.net/justice-matchmaking/pkg/constants"
	//"accelbyte.net/justice-matchmaking/pkg/utils"
	validator "github.com/AccelByte/justice-input-validation-go"
	"github.com/elliotchance/pie/v2"
	"github.com/mitchellh/copystructure"
	"github.com/sirupsen/logrus"
)

const (
	// NamespacePathParameter is the path before namespace parameter.
	NamespacePathParameter = "namespace"

	// OffsetQueryParameter is the query parameter for response paging.
	OffsetQueryParameter = "offset"

	// ChannelPathParameter  is the placeholder for channel parameter.
	ChannelPathParameter = "channel"

	// LimitQueryParameter is the query parameter for response paging.
	LimitQueryParameter = "limit"

	// ChannelNamePathParameter  is the placeholder for channel parameter.
	ChannelNamePathParameter = "channelName"

	// ImportChannelsParameterName is form name to import channels from file.
	ImportChannelsParameterName = "file"

	// ImportStrategyParameterName is field name to decide import strategy.
	ImportStrategyParameterName = "strategy"

	// MatchIDPathParameter is the placeholder for matchID parameter.
	MatchIDPathParameter = "matchID"

	// MatchIDsPathParameter is the placeholder for multiple matchID parameter.
	MatchIDsPathParameter = "matchIDs"

	// UserIDPathParameter is the placeholder for userID parameter.
	UserIDPathParameter = "userID"

	// PartyIDPathParameter is the placeholder for partyID parameter.
	PartyIDPathParameter = "partyID"

	// IsDeletedPathParameter is the placeholder for isDeleted parameter.
	IsDeletedPathParameter = "deleted"

	// LeaveOutStrategy define value to use ignore strategy when importing channels (existing channel will be used).
	LeaveOutStrategy = "leaveOut"

	// ReplaceStrategy define value to use replace strategy when importing channels (existing channel will be replaced).
	ReplaceStrategy = "replace"

	// MatchmakingStatusDone  is the status when matchmaking successfully done.
	MatchmakingStatusDone = "matched"
	// MatchmakingStatusCancelled  is the status when the matchmaking request was cancelled.
	MatchmakingStatusCancelled = "cancel"
	// MatchmakingStatusTimeout  is the status when the matchmaking request was timed out.
	MatchmakingStatusTimeout = "timeout"
	// MatchmakingStatusSessionInQueue  is the status when the joinable session is in queue.
	MatchmakingStatusSessionInQueue = "sessionInQueue"
	// MatchmakingStatusSessionFull  is the status when the joinable session is full and removed from the queue.
	MatchmakingStatusSessionFull = "sessionFull"
	// MatchmakingStatusSessionQueueTimeout  is the status when the joinable session queue was timed out.
	MatchmakingStatusSessionQueueTimeout = "sessionTimeout"
)

// MatchOptions types.
const (
	MatchOptionTypeAll     = "all"
	MatchOptionTypeAny     = "any"
	MatchOptionTypeUnique  = "unique"
	MatchOptionTypeDisable = "disable"
)

const DefaultWeightValue = float64(1.0)

// ErrChannelConflict error types when storing same channel name.
var ErrChannelConflict = errors.New("channel already exist")

// pool reusable object to reduce garbage collection that can affect performance
var pool = NewPool()

// MatchingParty contains information about matching party.
type MatchingParty struct {
	PartyID         string                 `json:"party_id"         x-nullable:"false"`
	PartyAttributes map[string]interface{} `json:"party_attributes"`
	PartyMembers    []PartyMember          `json:"party_members"`
	MatchAttributes

	// internal use only
	Locked bool `json:"-"`
}

func (mp *MatchingParty) GetPartyUserIDs() []string {
	userIDs := make([]string, 0)
	for _, m := range mp.PartyMembers {
		userIDs = append(userIDs, m.UserID)
	}

	return userIDs
}

// MatchAttributes contains information about matched results.
type MatchAttributes struct {
	FirstTicketCreatedAt int64 `json:"first_ticket_created_at"`
}

// CountPlayer count party members.
func (mp *MatchingParty) CountPlayer() (count int) {
	return len(mp.PartyMembers)
}

// Total get total float64 values of given attribute name.
func (mp *MatchingParty) Total(attributeNames []string, matchingRules []MatchingRule) float64 {
	if len(attributeNames) == 0 {
		return 0.0
	}
	var total float64
	for _, attributeName := range attributeNames {
		maxValue := getMaxValue(matchingRules, attributeName)
		for _, m := range mp.PartyMembers {
			if maxValue > 0 {
				total += (m.GetAttrFloat64(attributeName) / maxValue)
			} else {
				total += m.GetAttrFloat64(attributeName)
			}
		}
	}
	return total
}

func getMaxValue(matchingRules []MatchingRule, attributeName string) float64 {
	for _, rule := range matchingRules {
		if rule.Attribute == attributeName {
			return rule.NormalizationMax
		}
	}
	return 0
}

// Avg get average float64 values of given attribute name.
func (mp *MatchingParty) Avg(attributeNames []string, matchingRules []MatchingRule) float64 {
	if len(attributeNames) == 0 {
		return 0.0
	}
	var total float64
	for _, attributeName := range attributeNames {
		maxValue := getMaxValue(matchingRules, attributeName)
		var attributeTotal float64
		for _, m := range mp.PartyMembers {
			if maxValue > 0 {
				attributeTotal += m.GetAttrFloat64(attributeName) / maxValue
			} else {
				attributeTotal += m.GetAttrFloat64(attributeName)
			}
		}
		total += (attributeTotal / float64(mp.CountPlayer()))
	}
	return total / float64(len(attributeNames))
}

func PartyMemberAvg(members []PartyMember, attributeNames []string, matchingRules []MatchingRule) float64 {
	if len(attributeNames) == 0 {
		return 0.0
	}
	var total float64
	for _, attributeName := range attributeNames {
		maxValue := getMaxValue(matchingRules, attributeName)
		var attributeTotal float64
		for _, member := range members {
			if maxValue > 0 {
				attributeTotal += (member.GetAttrFloat64(attributeName) / maxValue)
			} else {
				attributeTotal += member.GetAttrFloat64(attributeName)
			}
		}
		total += (attributeTotal / float64(len(members)))
	}
	return total / float64(len(attributeNames))
}

// UpdateBlockedPlayersDetail update blocked players metadata for a party.
func (mp *MatchingParty) UpdateBlockedPlayersDetail(blocker string, blockedPlayers []string) {
	if blocker == "" || len(blockedPlayers) == 0 || blockedPlayers == nil {
		return
	}

	if mp.PartyAttributes == nil {
		mp.PartyAttributes = make(map[string]interface{})
	}

	// update blocked players metadata
	// append if existing metadata found
	newBlockedDetail := []interface{}{
		map[string]interface{}{
			AttributeBlocker: blocker,
			AttributeBlocked: blockedPlayers,
		},
	}
	existingBlockedDetail, blockedDetailExist := mp.PartyAttributes[AttributeBlockedPlayersDetail]
	if blockedDetailExist {
		if val, okVal := existingBlockedDetail.([]interface{}); okVal {
			newBlockedDetail = append(newBlockedDetail, val...)
		}
	}
	mp.PartyAttributes[AttributeBlockedPlayersDetail] = newBlockedDetail

	// update blocked players list
	// append if existing list found
	newBlockedPlayers := make([]string, 0)
	newBlockedPlayers = append(newBlockedPlayers, blockedPlayers...)
	if existingBlockedPlayers, blockedPlayersExist := mp.PartyAttributes[AttributeBlocked]; blockedPlayersExist {
		if val, okVal := existingBlockedPlayers.([]interface{}); okVal {
			for _, v := range val {
				if blockedPlayer, okBlockedPlayer := v.(string); okBlockedPlayer {
					newBlockedPlayers = append(newBlockedPlayers, blockedPlayer)
				}
			}
		}
	}
	mp.PartyAttributes[AttributeBlocked] = newBlockedPlayers
}

// GetBlockedPlayersMap get blocked players per user in party.
func (mp *MatchingParty) GetBlockedPlayersMap() (map[string][]interface{}, error) {
	val, ok := mp.PartyAttributes[AttributeBlockedPlayersDetail]
	if !ok {
		return nil, nil
	}

	mapValues, okMapValues := val.([]interface{})
	if !okMapValues {
		return nil, fmt.Errorf("invalid type of %s, expecting []interface{}", AttributeBlockedPlayersDetail)
	}

	blockedPlayersDetail := make(map[string][]interface{})
	for _, blockerMapInterface := range mapValues {
		blockerStr, blockedPlayers := extractBlockedPlayersDetail(blockerMapInterface)
		if blockerStr != "" {
			blockedPlayersDetail[blockerStr] = blockedPlayers
		}
	}
	return blockedPlayersDetail, nil
}

func (mp *MatchingParty) RemoveBlockedPlayersDetail(blocker string) {
	val, ok := mp.PartyAttributes[AttributeBlockedPlayersDetail]
	if !ok {
		return
	}

	mapValues, okMapValues := val.([]interface{})
	if !okMapValues {
		return
	}

	newBlockedPlayers := make([]interface{}, 0)
	newBlockedPlayersDetail := make([]interface{}, 0)
	for _, blockerMapInterface := range mapValues {
		blockerStr, blockedPlayers := extractBlockedPlayersDetail(blockerMapInterface)
		if blockerStr == blocker {
			continue
		}
		newBlockedPlayersDetail = append(newBlockedPlayersDetail, map[string]interface{}{
			AttributeBlocker: blockerStr,
			AttributeBlocked: blockedPlayers,
		})
		newBlockedPlayers = append(newBlockedPlayers, blockedPlayers...)
	}
	mp.PartyAttributes[AttributeBlockedPlayersDetail] = newBlockedPlayersDetail
	mp.PartyAttributes[AttributeBlocked] = newBlockedPlayers
}

func (mp MatchingParty) GetMemberUserIDs() []string {
	userIDs := make([]string, 0)
	for _, member := range mp.PartyMembers {
		userIDs = append(userIDs, member.UserID)
	}
	return userIDs
}

func (r *MatchmakingResult) LockParties() {
	for j, ally := range r.MatchingAllies {
		for k, party := range ally.MatchingParties {
			party.Locked = true
			r.MatchingAllies[j].MatchingParties[k] = party
		}
	}
}

func extractBlockedPlayersDetail(blockedPlayersDetail interface{}) (string, []interface{}) {
	data, ok := blockedPlayersDetail.(map[string]interface{})
	if !ok {
		return "", nil
	}
	blockerInterface, okBlockerInterface := data[AttributeBlocker]
	if !okBlockerInterface {
		return "", nil
	}

	blockerStr, okBlockerStr := blockerInterface.(string)
	if !okBlockerStr {
		return "", nil
	}

	blockedPlayersInterface, okBlockedPlayersInterface := data[AttributeBlocked]
	if !okBlockedPlayersInterface {
		return "", nil
	}

	blockedPlayers, okBlockedPlayers := blockedPlayersInterface.([]interface{})
	if !okBlockedPlayers {
		return "", nil
	}
	return blockerStr, blockedPlayers
}

// UpdateMemberAttributesValue update member attributes in session partyAttributes.
func (r *MatchmakingResult) UpdateMemberAttributesValue() {
	accumulatedMemberMap := map[string]float64{}
	accumulatedPlayerCount := map[string]float64{}

	for _, ally := range r.MatchingAllies {
		for _, party := range ally.MatchingParties {
			memberCount := float64(len(party.PartyMembers))
			val, ok := party.PartyAttributes[AttributeMemberAttr]
			if !ok {
				continue
			}

			// member attributes should only contain numerical
			// attribute that needs to be averaged
			attr, ok := val.(map[string]interface{})
			if !ok {
				continue
			}

			for k, v := range attr {
				var ref float64
				var err error

				switch c := v.(type) {
				case string:
					ref, err = strconv.ParseFloat(c, 64)
					if err != nil {
						continue
					}
				case float64:
					ref = c
				case int:
					ref = float64(c)
				case json.Number:
					ref, err = c.Float64()
					if err != nil {
						continue
					}
				default:
					continue
				}
				accumulatedMemberMap[k] += ref * memberCount
				accumulatedPlayerCount[k] += memberCount
			}
		}
	}
	accumulatedMemberAttr := map[string]interface{}{}
	for k := range accumulatedMemberMap {
		accumulatedMemberAttr[k] = accumulatedMemberMap[k] / accumulatedPlayerCount[k]
	}

	if r.PartyAttributes == nil {
		r.PartyAttributes = map[string]interface{}{}
	}
	r.PartyAttributes[AttributeMemberAttr] = accumulatedMemberAttr
}

// MatchingPartyV1 contains information about matching party.
type MatchingPartyV1 struct {
	PartyID      string          `json:"partyId"      x-nullable:"false"`
	PartyMembers []PartyMemberV1 `json:"partyMembers"`
}

// PartyAttributes consts.
const (
	AttributeMatchAttempt         = "match_attempt"
	AttributeLatencies            = "latencies"
	AttributeMemberAttr           = "member_attributes"
	AttributeBlocked              = "blocked_players"
	AttributeServerName           = "server_name"
	AttributeClientVersion        = "client_version"
	AttributeSubGameMode          = "sub_game_mode"
	AttributeNewSessionOnly       = "new_session_only"
	AttributeBlockedPlayersDetail = "blocked_players_detail"
	AttributeBlocker              = "blocker"
	AttributeCrossPlatform        = "cross_platform"
	AttributeCurrentPlatform      = "current_platform"
	AttributeRole                 = "role"
)

func GetBlockedPlayerUserIDs(partyAttributes map[string]interface{}) []string {
	var blockedPlayers []string
	if v, ok := partyAttributes[AttributeBlocked]; ok {
		if list, o := v.([]interface{}); o {
			for _, id := range list {
				if idStr, o := id.(string); o {
					blockedPlayers = append(blockedPlayers, idStr)
				}
			}
		}
	}
	return blockedPlayers
}

func RangeBlockedPlayerUserIDs(partyAttributes map[string]interface{}) func(func(userID string) bool) {
	return func(yield func(string) bool) {
		if list, ok := utils.GetMapValueAs[[]interface{}](partyAttributes, AttributeBlocked); ok {
			for _, id := range list {
				if userID, ok2 := id.(string); ok2 {
					if !yield(userID) {
						return
					}
				}
			}
		}
	}
}

// ExtraAttributes consts.
const (
	ROLE = "role" // unsupported
)

// role-based value.
const (
	AnyRole = "any" // unsupported
)

// MatchmakingRequest is the request for a party to get matched
// PartyAttributes can contain any of:
//   - server_name: string of preferred server name (for local DS)
//   - client_version: string of preferred client version (for matching with DS version)
//   - latencies: map of string: int containing pairs of region name and latency in ms
//   - match_attempt: (internal use only) int of number of retries to match this request
//   - member_attributes: (internal use only) map of attribute name (string) and value (interface{})
//     containing mean value of member attributes
type MatchmakingRequest struct {
	Priority            int                    `json:"priority"` // internal use only
	CreatedAt           int64                  `json:"created_at"`
	Channel             string                 `json:"channel"`
	Namespace           string                 `json:"namespace"`
	PartyID             string                 `json:"party_id"`
	PartyLeaderID       string                 `json:"party_leader_id"`
	PartyAttributes     map[string]interface{} `json:"party_attributes"`
	PartyMembers        []PartyMember          `json:"party_members"`
	AdditionalCriterias map[string]interface{} `json:"additional_criteria"`
	ExcludedSessions    []string               `json:"excluded_sessions"`

	// for internal use
	LatencyMap    map[string]int `json:"latency_map"`
	SortedLatency []Region       `json:"sorted_latency"`
}

func (r MatchmakingRequest) Copy() MatchmakingRequest {
	copied, err := copystructure.Copy(r)
	if err != nil {
		logrus.Warn("failed copy matchmakingRequest:", err)
	}
	copyRequest, _ := copied.(MatchmakingRequest)
	return copyRequest
}

func (r MatchmakingRequest) IsPriority() bool {
	return r.Priority > 0
}

func (r MatchmakingRequest) IsNewSessionOnly() bool {
	if attributeVal, ok := r.PartyAttributes[AttributeNewSessionOnly]; ok {
		isNewSessionOnly := strings.ToLower(fmt.Sprint(attributeVal))
		if isNewSessionOnly == "true" {
			return true
		}
	}
	return false
}

func (r MatchmakingRequest) CountPlayer() int {
	return len(r.PartyMembers)
}

// Avg get average float64 values of given attribute name.
func (r MatchmakingRequest) Avg(attributeName string) float64 {
	var total float64
	for _, m := range r.PartyMembers {
		total += m.GetAttrFloat64(attributeName)
	}
	return total / float64(r.CountPlayer())
}

func (r MatchmakingRequest) GetMapUserIDs() map[string]struct{} {
	mapUserIDs := make(map[string]struct{}, len(r.PartyMembers))
	for _, member := range r.PartyMembers {
		mapUserIDs[member.UserID] = struct{}{}
	}
	return mapUserIDs
}

func (r MatchmakingRequest) GetMemberUserIDs() []string {
	userIDs := make([]string, 0, len(r.PartyMembers))
	for _, member := range r.PartyMembers {
		userIDs = append(userIDs, member.UserID)
	}
	return userIDs
}

func (r MatchmakingRequest) GetMemberUserIDSet() map[string]struct{} {
	userIDSet := make(map[string]struct{})
	for _, member := range r.PartyMembers {
		userIDSet[member.UserID] = struct{}{}
	}
	return userIDSet
}

func (r MatchmakingRequest) GetBlockedPlayerUserIDs() []string {
	return GetBlockedPlayerUserIDs(r.PartyAttributes)
}

func (r MatchmakingRequest) GetMemberAttributes() map[string]interface{} {
	memberAttributes, ok := r.PartyAttributes[AttributeMemberAttr].(map[string]interface{})
	if !ok {
		return nil
	}
	return memberAttributes
}

// Region represents region latency data.
type Region struct {
	Region  string `json:"region"`
	Latency int    `json:"latency"`
}

// CancelRequest is the request for matchmaking cancellation.
type CancelRequest struct {
	Channel string `json:"channel,omitempty"`
	PartyID string `json:"party_id"`
}

// PartyMember is the member of the party and its predefined attribute.
type PartyMember struct {
	UserID          string                 `json:"user_id"          x-nullable:"false"`
	ExtraAttributes map[string]interface{} `json:"extra_attributes"`
}

func (m PartyMember) GetRole() []string {
	var values []string
	switch v := m.ExtraAttributes[ROLE].(type) {
	case []interface{}:
		for _, r := range v {
			values = append(values, r.(string))
		}
	case []string:
		values = v
	case string:
		// redis value is string in format of json array
		// ex: "[\"fighter\",\"tank\"]"
		err := json.Unmarshal([]byte(v), &values)
		if err != nil {
			// or the value can be in format string only
			// ex: "fighter"
			values = []string{v}
		}
	}
	roles := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		roles = append(roles, v)
	}
	return pie.Unique(roles)
}

// SetRole should only set 1 role, which is the final role
func (m *PartyMember) SetRole(role string) {
	extraAttributes := m.ExtraAttributes
	if extraAttributes == nil {
		extraAttributes = make(map[string]interface{})
	}
	extraAttributes[ROLE] = role
	m.ExtraAttributes = extraAttributes
}

func (p PartyMember) GetAttrFloat64(attributeName string) float64 {
	attributeValue, ok := p.ExtraAttributes[attributeName]
	if !ok {
		return 0.0
	}
	if float64Val, isFloat := attributeValue.(float64); isFloat {
		return float64Val
	}
	// the conversion is slow operation should avoid this
	val, _ := strconv.ParseFloat(fmt.Sprint(attributeValue), 64)
	return val
}

// PartyMemberV1 is the member of the party and its predefined attribute.
type PartyMemberV1 struct {
	UserID          string                 `json:"userId"          x-nullable:"false"`
	ExtraAttributes map[string]interface{} `json:"extraAttributes"`
}

// PlayerAttribute is (any) key-value attributes of a user calculated when doing matchmaking
// do extend this model if modifications are needed.
type PlayerAttribute struct {
	UserID     string                 `json:"user_id"    x-nullable:"false"`
	Attributes map[string]interface{} `json:"attributes"`
}

// MatchmakingResult is the result of matchmaking.
type MatchmakingResult struct {
	Status          string                 `json:"status"             x-nullable:"false"`
	PartyID         string                 `json:"party_id,omitempty" x-nullable:"true"` // exists only on status cancel
	MatchID         string                 `json:"match_id"           x-nullable:"false"`
	Channel         string                 `json:"channel"            x-nullable:"false"`
	Namespace       string                 `json:"namespace"          x-nullable:"false"`
	GameMode        string                 `json:"game_mode"          x-nullable:"false"`
	ServerName      string                 `json:"server_name"        x-nullable:"false"`
	ClientVersion   string                 `json:"client_version"     x-nullable:"false"`
	Region          string                 `json:"region"             x-nullable:"false"`
	Joinable        bool                   `json:"joinable"           optional:"true"    x-nullable:"true"`
	MatchingAllies  []MatchingAlly         `json:"matching_allies"`
	Deployment      string                 `json:"deployment"         x-nullable:"false"`
	UpdatedAt       time.Time              `json:"updated_at"`
	QueuedAt        int64                  `json:"queued_at"          x-nullable:"false"`
	MatchSessionID  string                 `json:"match_session_id"` // need match session id for sending it to session history
	PartyAttributes map[string]interface{} `json:"party_attributes"` // this will be populated when receiving joinable session queue request
	PivotID         string                 `json:"pivot_id"`
}

func (r MatchmakingResult) Validate() error {
	if r.Channel == "" {
		return errors.New("channel cannot be empty")
	}
	if r.GameMode == "" {
		return errors.New("game mode cannot be empty")
	}
	if r.Namespace == "" {
		return errors.New("namespace cannot be empty")
	}
	return nil
}

func (r *MatchmakingResult) UpdateBlockedPlayers(blockedPlayers []string) {
	if len(blockedPlayers) == 0 {
		return
	}

	if r.PartyAttributes == nil {
		r.PartyAttributes = make(map[string]interface{})
	}

	existingBlockedPlayers, blockedPlayersExist := r.PartyAttributes[AttributeBlocked]
	if blockedPlayersExist {
		if val, okVal := existingBlockedPlayers.([]interface{}); okVal {
			for _, v := range val {
				if blockedPlayer, okBlockedPlayer := v.(string); okBlockedPlayer {
					blockedPlayers = append(blockedPlayers, blockedPlayer)
				}
			}
		}
	}
	r.PartyAttributes[AttributeBlocked] = blockedPlayers
}

func (r MatchmakingResult) GetOldestTicketTimestamp() int64 {
	oldestTicketTimestamp := int64(math.MaxInt64)
	for _, ally := range r.MatchingAllies {
		for _, party := range ally.MatchingParties {
			if party.FirstTicketCreatedAt < oldestTicketTimestamp {
				oldestTicketTimestamp = party.FirstTicketCreatedAt
			}
		}
	}
	if oldestTicketTimestamp == int64(math.MaxInt64) || oldestTicketTimestamp == 0 {
		// just as a last resort fallback
		oldestTicketTimestamp = r.QueuedAt
	}
	return oldestTicketTimestamp
}

func (r MatchmakingResult) GetMapPartyIDs() map[string]struct{} {
	mapPartyIDs := make(map[string]struct{})
	for _, ally := range r.MatchingAllies {
		for _, party := range ally.MatchingParties {
			mapPartyIDs[party.PartyID] = struct{}{}
		}
	}
	return mapPartyIDs
}

func (r MatchmakingResult) GetMapUserIDs() map[string]struct{} {
	mapUserIDs := make(map[string]struct{})
	for _, ally := range r.MatchingAllies {
		for _, party := range ally.MatchingParties {
			for _, member := range party.PartyMembers {
				mapUserIDs[member.UserID] = struct{}{}
			}
		}
	}
	return mapUserIDs
}

func (r MatchmakingResult) GetMemberUserIDs() []string {
	var userIDs []string
	for _, ally := range r.MatchingAllies {
		for _, party := range ally.MatchingParties {
			for _, member := range party.PartyMembers {
				userIDs = append(userIDs, member.UserID)
			}
		}
	}
	return userIDs
}

func (r MatchmakingResult) GetMemberUserIDSet() map[string]struct{} {
	userIDSet := make(map[string]struct{})
	for _, ally := range r.MatchingAllies {
		for _, party := range ally.MatchingParties {
			for _, member := range party.PartyMembers {
				userIDSet[member.UserID] = struct{}{}
			}
		}
	}
	return userIDSet
}

func (r MatchmakingResult) GetBlockedPlayerUserIDs() []string {
	return GetBlockedPlayerUserIDs(r.PartyAttributes)
}

// MatchingAlly is the model of a side.
type MatchingAlly struct {
	TeamID          string          `json:"team_id"`
	MatchingParties []MatchingParty `json:"matching_parties"`
	PlayerCount     int             `json:"-"`
}

func (a MatchingAlly) CountPlayer() (count int) {
	for _, p := range a.MatchingParties {
		count += len(p.PartyMembers)
	}
	return
}

func (a MatchingAlly) Total(attributeNames []string, matchingRules []MatchingRule) float64 {
	if len(a.MatchingParties) == 0 || len(attributeNames) == 0 {
		return 0.0
	}
	var total float64
	for _, attribute := range attributeNames {
		maxValue := getMaxValue(matchingRules, attribute)
		var attributeTotal float64
		for _, p := range a.MatchingParties {
			for _, m := range p.PartyMembers {
				if maxValue > 0 {
					attributeTotal += (m.GetAttrFloat64(attribute) / maxValue)
				} else {
					attributeTotal += m.GetAttrFloat64(attribute)
				}
			}
		}
		total += attributeTotal
	}
	return total
}

func (a MatchingAlly) Avg(attributeNames []string, matchingRules []MatchingRule) float64 {
	if len(a.MatchingParties) == 0 || len(attributeNames) == 0 {
		return 0.0
	}
	var total float64
	for _, attribute := range attributeNames {
		maxValue := getMaxValue(matchingRules, attribute)
		var attributeTotal float64
		for _, p := range a.MatchingParties {
			for _, m := range p.PartyMembers {
				if maxValue > 0 {
					attributeTotal += (m.GetAttrFloat64(attribute) / maxValue)
				} else {
					attributeTotal += m.GetAttrFloat64(attribute)
				}
			}
		}
		total += attributeTotal / float64(a.CountPlayer())
	}
	return total / float64(len(attributeNames))
}

func (a MatchingAlly) GetMembers() []PartyMember {
	partyMembers := make([]PartyMember, 0, len(a.MatchingParties))
	for _, party := range a.MatchingParties {
		partyMembers = append(partyMembers, party.PartyMembers...)
	}
	return partyMembers
}

func (a MatchingAlly) GetMemberUserIDs() []string {
	userIDs := make([]string, 0)
	for _, party := range a.MatchingParties {
		userIDs = append(userIDs, party.GetMemberUserIDs()...)
	}
	return userIDs
}

func (a MatchingAlly) GetBlockedPlayerUserIDs() []string {
	blockedIDs := make([]string, 0)
	for _, p := range a.MatchingParties {
		blockedIDs = append(blockedIDs, GetBlockedPlayerUserIDs(p.PartyAttributes)...)
	}
	return blockedIDs
}

type BlockedPlayerOption string

const (
	// BlockedPlayerCannotMatch respect block (default value if empty)
	BlockedPlayerCannotMatch BlockedPlayerOption = "blockedPlayerCannotMatch" // default

	// BlockedPlayerCanMatchOnDifferentTeam respect block only for the same team
	BlockedPlayerCanMatchOnDifferentTeam BlockedPlayerOption = "blockedPlayerCanMatchOnDifferentTeam"

	// BlockedPlayerCanMatch don't respect block
	BlockedPlayerCanMatch BlockedPlayerOption = "blockedPlayerCanMatch"
)

var AvailableBlockedOptions = []BlockedPlayerOption{BlockedPlayerCanMatch, BlockedPlayerCanMatchOnDifferentTeam, BlockedPlayerCannotMatch}

func (b BlockedPlayerOption) Validate() error {
	if b != "" && !slices.Contains(AvailableBlockedOptions, b) {
		return fmt.Errorf("blocked_player_option should be one of %v", AvailableBlockedOptions)
	}
	return nil
}

// RuleSet is a rule set.
type RuleSet struct {
	AutoBackfill                       bool                  `bson:"auto_backfill"                          json:"auto_backfill"`
	RegionExpansionRateMs              int                   `bson:"region_expansion_rate_ms"               json:"region_expansion_rate_ms"               valid:"range(0|2147483647)"` // how old a ticket is before expanding the latency and region search by 1 step
	RegionExpansionRangeMs             int                   `bson:"region_expansion_range_ms"              json:"region_expansion_range_ms"              valid:"range(0|2147483647)"` // how big 1 step of latency expansion is
	RegionLatencyInitialRangeMs        int                   `bson:"region_latency_initial_range_ms"        json:"region_latency_initial_range_ms"        valid:"range(0|2147483647)"` // minimum latency to allow matching in a region
	RegionLatencyMaxMs                 int                   `bson:"region_latency_max_ms"                  json:"region_latency_max_ms"                  valid:"range(0|2147483647)"` // maximum latency search can expand to in 1 region
	AllianceRule                       AllianceRule          `bson:"allianceRule"                           json:"alliance"`
	MatchingRule                       []MatchingRule        `bson:"matchingRule"                           json:"matching_rule"`
	FlexingRule                        []FlexingRule         `bson:"flexingRule"                            json:"flexing_rule"`
	AllianceFlexingRule                []AllianceFlexingRule `bson:"alliance_flexing_rule"                  json:"alliance_flexing_rule"`
	MatchOptions                       MatchOptionRule       `bson:"match_options"                          json:"match_options"`
	RebalanceEnable                    *bool                 `json:"rebalance_enable,omitempty"`
	RebalanceVersion                   int                   `json:"rebalance_version,omitempty"` // can be 1 or 2. Any other value will default to latest.
	TicketObservabilityEnable          bool                  `bson:"ticket_observability_enable"            json:"ticket_observability_enable"            optional:"true"`
	MatchOptionsReferredForBackfill    bool                  `bson:"match_options_referred_for_backfill"    json:"match_options_referred_for_backfill"    optional:"true"`
	BlockedPlayerOption                BlockedPlayerOption   `bson:"blocked_player_option"                  json:"blocked_player_option,omitempty"        optional:"true"`
	MaxDelayMs                         int                   `bson:"max_delay_ms"                           json:"max_delay_ms,omitempty"                 optional:"true"             valid:"range(0|2147483647)"`
	DisableBidirectionalLatencyAfterMs int                   `bson:"disable_bidirectional_latency_after_ms" json:"disable_bidirectional_latency_after_ms" optional:"true"             valid:"range(0|2147483647)"`
	RegionLatencyRuleWeight            *float64              `bson:"region_latency_rule_weight"             json:"region_latency_rule_weight,omitempty"   optional:"true"             valid:"range(0|1000)"`

	ExtraAttributes ExtraAttributes `bson:"-" json:"extra_attributes,omitempty" optional:"true"`

	// internal use
	isDefaultSet bool
}

func (ruleSet *RuleSet) Validate() error {
	if _, err := validator.ValidateStruct(ruleSet); err != nil {
		return err
	}

	if err := ruleSet.AllianceRule.Validate(); err != nil {
		return err
	}

	for _, flexingRule := range ruleSet.AllianceFlexingRule {
		err := flexingRule.Validate()
		if err != nil {
			return err
		}
	}

	for _, matchingRule := range ruleSet.MatchingRule {
		err := matchingRule.Validate()
		if err != nil {
			return err
		}
	}

	for _, flexingRule := range ruleSet.FlexingRule {
		err := flexingRule.Validate()
		if err != nil {
			return err
		}
	}

	for _, matchOption := range ruleSet.MatchOptions.Options {
		err := matchOption.Validate()
		if err != nil {
			return err
		}
	}

	if err := ruleSet.BlockedPlayerOption.Validate(); err != nil {
		return err
	}

	if ruleSet.RegionExpansionRangeMs < 0 {
		return errors.New("region expansion range ms cannot lower than 0")
	}

	if ruleSet.RegionExpansionRateMs < 0 {
		return errors.New("region expansion rate ms cannot lower than 0")
	}

	if ruleSet.RegionLatencyInitialRangeMs < 0 {
		return errors.New("region latency initial range ms cannot lower than 0")
	}

	if ruleSet.RegionLatencyMaxMs < 0 {
		return errors.New("region latency max ms cannot lower than 0")
	}

	if ruleSet.RegionLatencyInitialRangeMs > ruleSet.RegionLatencyMaxMs {
		return errors.New("max region latency must equal or more than region latency initial")
	}

	for _, rule := range ruleSet.MatchingRule {
		if rule.Criteria == constants.DistanceCriteria {
			maxDistance := rule.Reference
			for _, flexingRule := range ruleSet.FlexingRule {
				if rule.Attribute == flexingRule.Attribute {
					if maxDistance < flexingRule.Reference {
						maxDistance = flexingRule.Reference
					}
					if flexingRule.Weight != nil {
						return fmt.Errorf("invalid flexing rule for attribute '%s', weight can only be set in the main rule", flexingRule.Attribute)
					}
					if flexingRule.NormalizationMax > 0 {
						return fmt.Errorf("invalid flexing rule for attribute '%s', normalizationMax can only be set in the main rule", flexingRule.Attribute)
					}
				}
			}
			if rule.NormalizationMax > 0 && rule.NormalizationMax < maxDistance {
				return fmt.Errorf("invalid matching rule for attribute '%s', max must be greater or equal than flexing reference value", rule.Attribute)
			}
		}
	}

	return nil
}

func (ruleSet *RuleSet) SetDefaultValues() {
	if ruleSet.isDefaultSet {
		return
	}
	ruleSet.isDefaultSet = true
	for i, rule := range ruleSet.MatchingRule {
		if rule.Criteria == constants.DistanceCriteria && rule.NormalizationMax == 0 {
			// max is required when using weight, set default from the reference when matching rule max is not defined
			maxRef := rule.Reference
			for _, flexingRule := range ruleSet.FlexingRule {
				if flexingRule.Criteria == constants.DistanceCriteria {
					if maxRef < flexingRule.Reference {
						maxRef = flexingRule.Reference
					}
				}
			}
			ruleSet.MatchingRule[i].NormalizationMax = maxRef
		}
	}
}

func TRUE() *bool {
	b := true
	return &b
}

func FALSE() *bool {
	b := false
	return &b
}

func (r RuleSet) Copy() RuleSet {
	copied, err := copystructure.Copy(r)
	if err != nil {
		logrus.Warn("Failed to copy RuleSet struct:", err)
		return r
	}
	ruleset, _ := copied.(RuleSet)
	return ruleset
}

func (r RuleSet) BlockedPlayerAllowedToMatch() bool {
	return r.BlockedPlayerOption == BlockedPlayerCanMatch
}

func (r RuleSet) IsSinglePlay() bool {
	return r.AllianceRule.MinNumber == 1 && r.AllianceRule.MaxNumber == 1 && r.AllianceRule.PlayerMinNumber == 1 && r.AllianceRule.PlayerMaxNumber == 1
}

func (r RuleSet) GetRegionLatencyRuleWeight() float64 {
	if r.RegionLatencyRuleWeight == nil {
		return DefaultWeightValue
	}
	return *r.RegionLatencyRuleWeight
}

type AllianceRule struct {
	MinNumber       int `json:"min_number"        valid:"range(0|2147483647)"`
	MaxNumber       int `json:"max_number"        valid:"range(0|2147483647)"`
	PlayerMinNumber int `json:"player_min_number" valid:"range(0|2147483647)"`
	PlayerMaxNumber int `json:"player_max_number" valid:"range(0|2147483647)"`
}

func (reqData *AllianceRule) Validate() error {
	_, err := validator.ValidateStruct(reqData)
	if err != nil {
		return err
	}

	if reqData.MinNumber > reqData.MaxNumber {
		return errors.New("maximum alliance number must be greater than or equal with minimum alliance number")
	}

	if reqData.MinNumber == 0 || reqData.MaxNumber == 0 {
		return errors.New("rule should have minimum 1 alliance")
	}

	if reqData.PlayerMinNumber > reqData.PlayerMaxNumber {
		return errors.New("maximum player number must be greater than or equal with minimum player number")
	}

	if reqData.PlayerMinNumber == 0 || reqData.PlayerMaxNumber == 0 {
		return errors.New("rule should have minimum 1 player in alliance")
	}

	return nil
}

// ValidateAllyMaxOnly validate an ally based on alliance rule maximum capability only
// please specify allyIndex to get role for multi-combo role-based
// (especially when it is assymetry, where each ally has different set of roles),
// otherwise we can set it to 0
func (rule AllianceRule) ValidateAllyMaxOnly(ally MatchingAlly, allyIndex int) error {
	members := pool.PartyMembers.Get()
	defer pool.PartyMembers.Put(members)

	for _, party := range ally.MatchingParties {
		members = append(members, party.PartyMembers...)
	}
	playerCount := ally.CountPlayer()
	if playerCount > rule.PlayerMaxNumber {
		return fmt.Errorf("player count %d more than max %d", playerCount, rule.PlayerMaxNumber)
	}
	return nil
}

// ValidateAlly validate an ally based on alliance rule
// please specify allyIndex to get role for multi-combo role-based
// (especially when it is assymetry, where each ally has different set of roles),
// otherwise we can set it to 0
func (rule AllianceRule) ValidateAlly(ally MatchingAlly, allyIndex int) error {
	members := pool.PartyMembers.Get()
	defer pool.PartyMembers.Put(members)

	for _, party := range ally.MatchingParties {
		members = append(members, party.PartyMembers...)
	}
	playerCount := ally.CountPlayer()
	if playerCount < rule.PlayerMinNumber {
		return fmt.Errorf("player count %d less than min %d", playerCount, rule.PlayerMinNumber)
	}
	if playerCount > rule.PlayerMaxNumber {
		return fmt.Errorf("player count %d more than max %d", playerCount, rule.PlayerMaxNumber)
	}
	return nil
}

// ValidateAllies validate allies based on alliance rule
func (rule AllianceRule) ValidateAllies(allies []MatchingAlly, blockedPlayerOption BlockedPlayerOption) error {
	// validate max ally count
	if len(allies) > rule.MaxNumber {
		return fmt.Errorf("ally count %d more than max %d", len(allies), rule.MaxNumber)
	}
	// validate min ally count
	if len(allies) < rule.MinNumber {
		return fmt.Errorf("ally count %d less than min %d", len(allies), rule.MinNumber)
	}
	minAlly := rule.MinNumber
	var countAllyWithMinMember int
	// validate each ally
	for _, ally := range allies {
		playerCount := ally.CountPlayer()
		if playerCount == 0 {
			continue
		}
		if playerCount < rule.PlayerMinNumber {
			return fmt.Errorf("player count %d less than min %d", playerCount, rule.PlayerMinNumber)
		}
		countAllyWithMinMember++
		if playerCount > rule.PlayerMaxNumber {
			return fmt.Errorf("player count %d more than max %d", playerCount, rule.PlayerMaxNumber)
		}
	}
	if countAllyWithMinMember < minAlly {
		return fmt.Errorf("player count less than min, should have min %d ally with %d player", minAlly, rule.PlayerMinNumber)
	}
	/*
		[AR-7033] check blocked players for:
		- respect block only for the same team
	*/
	if blockedPlayerOption == BlockedPlayerCanMatchOnDifferentTeam {
		for _, ally := range allies {
			// check blocked players for each team
			memberIDs := ally.GetMemberUserIDs()

			blockedIDs := make(map[string]struct{}, 0)
			for _, v := range ally.GetBlockedPlayerUserIDs() {
				blockedIDs[v] = struct{}{}
			}

			for _, userID := range memberIDs {
				if _, exist := blockedIDs[userID]; exist {
					return fmt.Errorf("there is blocked player as a team, player %s was blocked by other player in the team", userID)
				}
			}
		}
	}
	return nil
}

type AllianceFlexingRule struct {
	Duration int64 `bson:"duration" json:"duration" valid:"range(0|2147483647)"`
	AllianceRule
}

func (a AllianceFlexingRule) Validate() error {
	if _, err := validator.ValidateStruct(a); err != nil {
		return err
	}

	if a.Duration < 0 {
		return errors.New("duration cannot be minus")
	}

	if err := a.AllianceRule.Validate(); err != nil {
		return err
	}
	return nil
}

// Channel contains channel information.
type Channel struct {
	Ruleset RuleSet `bson:"ruleset" json:"ruleset"`
}

// GetAllianceRules return alliance rule whether it is from game mode or sub game mode.
func (c Channel) GetAllianceRules() []AllianceRule {
	var allianceRules []AllianceRule
	{
		// game mode alliance rule
		allianceRules = []AllianceRule{c.Ruleset.AllianceRule}
	}
	return allianceRules
}

// MatchingRule defines a matching rule
// attribute is the target attribute name
// criteria is property condition need to be met
// reference is value to test against the criteria
// example :
// rule : match players with mmr(attribute) distance(criteria) 500(reference)
// attribute="mmr"
// criteria="distance"
// reference="500"
// max="1500"
// isForBalancing=false.
type MatchingRule struct {
	Attribute        string  `bson:"attribute"        json:"attribute"        valid:"stringlength(1|64),lowercase"         x-nullable:"false"`
	Criteria         string  `bson:"criteria"         json:"criteria"         valid:"in(distance|average|smaller|greater)" x-nullable:"false"`
	Reference        float64 `bson:"reference"        json:"reference"        valid:"range(0|2147483647)"                  x-nullable:"false"`
	NormalizationMax float64 `bson:"normalizationMax" json:"normalizationMax" valid:"range(0|2147483647)"                  x-nullable:"false"`
	/*
		IsForBalancing is nullable because we need to keep for backward compatible with below behaviour:
		- if all distance rule for isForBalancing is null, then use the first rule as the balancing rule (backward compatibility)
		- if all distance rule for isForBalancing is false, then use nothing for balance the team
		- if there is one or more isForBalancing is true, then use it as balancing rule
	*/
	IsForBalancing *bool    `bson:"isForBalancing" json:"isForBalancing"   x-nullable:"true"`
	Weight         *float64 `bson:"weight"         json:"weight,omitempty" valid:"range(0|1000)" x-nullable:"true"`
}

func (m MatchingRule) Validate() error {
	if m.Attribute == "" {
		return errors.New("matching rule attribute name cannot be empty")
	}

	if m.Criteria == "" {
		return errors.New("matching rule Criteria name cannot be empty")
	}

	if m.Reference < 0 {
		return errors.New("matching rule reference cannot be minus")
	}

	if _, err := validator.ValidateStruct(m); err != nil {
		return err
	}

	return nil
}

func (m MatchingRule) GetWeight() float64 {
	if m.Weight == nil {
		return DefaultWeightValue
	}
	return *m.Weight
}

// FlexingRule defines a matching rule.
type FlexingRule struct {
	Duration int64 `bson:"duration" json:"duration" valid:"range(0|2147483647)"`
	MatchingRule
}

func (f FlexingRule) Validate() error {
	if _, err := validator.ValidateStruct(f); err != nil {
		return err
	}

	if f.Duration < 0 {
		return errors.New("duration cannot be minus")
	}

	if err := f.MatchingRule.Validate(); err != nil {
		return err
	}

	return nil
}

type MatchOptionRule struct {
	Options []MatchOption `bson:"options" json:"options"`
}

type MatchOption struct {
	Name string `bson:"name" json:"name" valid:"stringlength(1|64)"         x-nullable:"false"`
	Type string `bson:"type" json:"type" valid:"in(all|any|unique|disable)" x-nullable:"false"`
}

func (m MatchOption) Validate() error {
	if m.Name == "" {
		return errors.New("match options name cannot be empty")
	}

	if m.Type == "" {
		return errors.New("match options type cannot be empty")
	}

	if _, err := validator.ValidateStruct(m); err != nil {
		return err
	}
	return nil
}

type Role struct {
	Name string `bson:"name" json:"name" valid:"stringlength(1|64),lowercase" x-nullable:"false"`
	Min  int    `bson:"min"  json:"min"  valid:"range(0|2147483647)"          x-nullable:"false"`
	Max  int    `bson:"max"  json:"max"  valid:"range(0|2147483647)"          x-nullable:"false"`
}

type AllianceComposition struct {
	MinTeam   int
	MaxTeam   int
	MaxPlayer int
	MinPlayer int
}

func (a AllianceComposition) MinTotalPlayer() int {
	return a.MinTeam * a.MinPlayer
}

type UserStatsResp struct {
	ProfileID string  `json:"profileId"`
	StatCode  string  `json:"statCode"`
	Value     float64 `json:"value"`
}

type MatchingParties []MatchingParty

type AllPartiesResponse map[string]MatchingParties

type CrossPlayAttribute struct {
	CrossplayEnabled bool
	CrossPlatforms   []string
}

type Action string

const (
	MatchFound     Action = "matchFound"
	MatchNotFound  Action = "matchNotFound"
	Flexed         Action = "flexed"
	ReturnedToPool Action = "returnedToPool"
)

type EventTicketObservability struct {
	Timestamp                 time.Time      `json:"timestamp"`
	Action                    Action         `json:"action"`
	PartyID                   string         `json:"partyID"`
	MatchID                   string         `json:"matchID,omitempty"` // match ID only when match found
	Namespace                 string         `json:"namespace"`         // filled in by default in event
	GameMode                  string         `json:"gameMode"`          // filled in by default in event
	ActiveAllianceRule        *AllianceRule  `json:"activeAllianceRule,omitempty"`
	ActiveMatchingRule        []MatchingRule `json:"activeMatchingRule,omitempty"`
	Function                  string         `json:"function,omitempty"`                  // matchPlayers or matchSessions
	Iteration                 int            `json:"iteration,omitempty"`                 // pivotMatchingCounter as iteration
	TimeToMatchSec            float64        `json:"timeToMatchSec,omitempty"`            // time to match
	UnmatchReason             string         `json:"unmatchReason,omitempty"`             // when unable to find match
	RemainingTickets          int            `json:"remainingTickets,omitempty"`          // when unable to find match
	RemainingPlayersPerTicket []int          `json:"remainingPlayersPerTicket,omitempty"` // when unable to find match
	MatchedRegion             string         `json:"matchedRegion,omitempty"`
	SessionTickID             string         `json:"sessionTickID"`
	PodName                   string         `json:"podName"`                    // filled in by default in event
	UnbackfillReason          string         `json:"unbackfillReason,omitempty"` // when unable to backfill
	IsBackfillMatch           bool           `json:"isBackfillMatch"`            // flag to distinguish between new match and backfill match
	IsRuleSetFlexed           bool           `json:"isRuleSetFlexed"`            // flag is ruleset is getting flexed
	TickID                    int64          `json:"tickID"`                     // tick id for the matchmaking tick
	IsPivot                   bool           `json:"isPivot"`
	TotalPlayers              int            `json:"totalPlayers"`

	ElapsedTime      float64                `json:"elapsedTime"`
	MemberAttributes map[string]interface{} `json:"memberAttributes"`
}

type EventTicketObservabilitySession struct {
	SessionTickID string             `json:"sessionTickID"`
	Timestamp     time.Time          `json:"timestamp"`
	Session       *MatchmakingResult `json:"session"`
}

type ActionMatchHistory string

const (
	ActionMatchHistoryCreated         ActionMatchHistory = "matchCreated"
	ActionMatchHistoryUpdated         ActionMatchHistory = "matchUpdated"
	ActionMatchHistoryRemoved         ActionMatchHistory = "matchRemoved"
	ActionMatchHistoryExpired         ActionMatchHistory = "matchExpired"
	ActionMatchHistoryAddedToBackfill ActionMatchHistory = "addedToBackfill"
)

type EventMatchHistory struct {
	Timestamp time.Time          `json:"timestamp"`
	MatchID   string             `json:"matchID"`
	Namespace string             `json:"namespace"`
	Matchpool string             `json:"matchpool"`
	Action    ActionMatchHistory `json:"action"`
	PodName   string             `json:"podName"`
	RuleSet   string             `json:"ruleSet,omitempty"`
	TickID    int64              `json:"tickID,omitempty"`
	Match     any                `json:"match,omitempty"` // this should be type of *matchmaker.Match. Uses the "any" to avoid cycle import in the future
}

type DefaultParam struct {
	Namespace string
	Matchpool string
	RuleSet   string
}

type ExtraAttributes struct {
	DefaultParam    DefaultParam
	TicketChunkSize int
}
