// Copyright (c) 2019-2024 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

//nolint:gosec,goconst
package defaultmatchmaker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/config"
	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
	"github.com/caarlos0/env"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-openapi/swag"
	ulid "github.com/oklog/ulid/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	timeNow     = time.Now()
	entropy     = ulid.Monotonic(rand.New(rand.NewSource(timeNow.UnixNano())), 0)
	timeNowUlid = ulid.Timestamp(timeNow)

	ulidMutex = sync.RWMutex{}
)

//nolint:gochecknoinits
func init() {
	testing.Init()
	logrus.SetOutput(os.Stdout)
	configuration := &config.Config{}
	err := env.Parse(configuration)
	if err != nil {
		logrus.Fatal("unable to parse environment variables: ", err)
	}
	logrus.SetLevel(logrus.ErrorLevel)
}

func generateUlid(t time.Time) string {
	ulidMutex.Lock()
	defer ulidMutex.Unlock()
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

func generateRequest(channelName string, requestCount, memberPerAllyCount int) []models.MatchmakingRequest {
	return generateRequestWithMMR(channelName, requestCount, memberPerAllyCount, rand.Intn(1000)) //nolint:gosec
}

func generateRequestWithMemberCount(channelName string, partyMemberCounts []int) []models.MatchmakingRequest {
	out := make([]models.MatchmakingRequest, 0)
	for _, i := range partyMemberCounts {
		out = append(out, generateRequestWithMMR(channelName, 1, i, rand.Intn(1000))...)
	}
	return out
}

func generateRequestWithMMR(channelName string, requestCount, memberPerAllyCount, mmr int) []models.MatchmakingRequest {
	var mmRequests []models.MatchmakingRequest
	t := time.Now()
	for i := 0; i < requestCount; i++ {
		var partyMembers []models.PartyMember
		var totalPing float64
		for j := 0; j < memberPerAllyCount; j++ {
			ping := float64(rand.Intn(300)) //nolint:gosec
			totalPing += ping
			partyMember := models.PartyMember{
				UserID: utils.GenerateUUID(),
				ExtraAttributes: map[string]interface{}{
					"mmr":  float64(mmr),
					"ping": ping,
				},
			}
			partyMembers = append(partyMembers, partyMember)
		}
		meanPing := totalPing / float64(memberPerAllyCount)
		request := models.MatchmakingRequest{
			// note:
			// using ULID to ensure party IDs are sorted according to the creation order,
			// since it will sort party IDs that has the same score
			PartyID:      generateUlid(t),
			Channel:      channelName,
			CreatedAt:    time.Now().Add(-time.Duration(rand.Intn(100000)) * time.Millisecond).UTC().Unix(), //nolint:gosec
			PartyMembers: partyMembers,
			PartyAttributes: map[string]interface{}{
				models.AttributeMemberAttr: map[string]interface{}{
					"mmr":  float64(mmr),
					"ping": meanPing,
				},
			},
			LatencyMap:          make(map[string]int),
			AdditionalCriterias: map[string]interface{}{},
		}
		mmRequests = append(mmRequests, request)
	}

	return mmRequests
}

func generateRequestNoMMR(channelName string, requestCount, memberPerAllyCount int) []models.MatchmakingRequest {
	return generateRequestWithoutMMR(channelName, requestCount, memberPerAllyCount) //nolint:gosec
}

func generateRequestWithoutMMR(channelName string, requestCount, memberPerAllyCount int) []models.MatchmakingRequest {
	var mmRequests []models.MatchmakingRequest
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	for i := 0; i < requestCount; i++ {
		var partyMembers []models.PartyMember
		var totalPing float64
		for j := 0; j < memberPerAllyCount; j++ {
			ping := float64(rand.Intn(300)) //nolint:gosec
			totalPing += ping
			partyMember := models.PartyMember{
				UserID:          utils.GenerateUUID(),
				ExtraAttributes: map[string]interface{}{},
			}
			partyMembers = append(partyMembers, partyMember)
		}
		request := models.MatchmakingRequest{
			// note:
			// using ULID to ensure party IDs are sorted according to the creation order,
			// since it will sort party IDs that has the same score
			PartyID:      ulid.MustNew(ulid.Timestamp(t), entropy).String(),
			Channel:      channelName,
			CreatedAt:    time.Now().Add(-time.Duration(rand.Intn(100000)) * time.Millisecond).UTC().Unix(), //nolint:gosec
			PartyMembers: partyMembers,
			PartyAttributes: map[string]interface{}{
				models.AttributeMemberAttr: map[string]interface{}{},
			},
			LatencyMap:          make(map[string]int),
			AdditionalCriterias: map[string]interface{}{},
			Namespace:           "test",
		}
		mmRequests = append(mmRequests, request)
	}

	return mmRequests
}

func generateRequestWithMMRAndRole(channelName string, memberCount, mmr int, roles []string, priority int) models.MatchmakingRequest {
	var partyMembers []models.PartyMember
	var totalPing float64
	for j := 0; j < memberCount; j++ {
		ping := float64(rand.Intn(300)) //nolint:gosec
		totalPing += ping
		partyMember := models.PartyMember{
			UserID: utils.GenerateUUID(),
			ExtraAttributes: map[string]interface{}{
				"mmr":  float64(mmr),
				"ping": ping,
				"role": roles[j],
			},
		}
		partyMembers = append(partyMembers, partyMember)
	}
	meanPing := totalPing / float64(memberCount)
	request := models.MatchmakingRequest{
		// note:
		// using ULID to ensure party IDs are sorted according to the creation order,
		// since it will sort party IDs that has the same score
		Priority:     priority,
		PartyID:      generateUlid(timeNow),
		Channel:      channelName,
		CreatedAt:    time.Now().Add(-time.Duration(rand.Intn(100000)) * time.Millisecond).UTC().Unix(), //nolint:gosec
		PartyMembers: partyMembers,
		PartyAttributes: map[string]interface{}{
			models.AttributeMemberAttr: map[string]interface{}{
				"mmr":   float64(mmr),
				"ping":  meanPing,
				"roles": roles,
			},
		},
		LatencyMap:          make(map[string]int),
		AdditionalCriterias: map[string]interface{}{},
	}
	return request
}

func createMatchingAlly(tickets ...models.MatchmakingRequest) models.MatchingAlly {
	ally := models.MatchingAlly{}
	for _, ticket := range tickets {
		ally.MatchingParties = append(ally.MatchingParties, models.MatchingParty{
			PartyID:         ticket.PartyID,
			PartyMembers:    ticket.PartyMembers,
			PartyAttributes: ticket.PartyAttributes,
		})
	}
	return ally
}

func appendMatchingAlly(ally models.MatchingAlly, tickets ...models.MatchmakingRequest) models.MatchingAlly {
	for _, ticket := range tickets {
		ally.MatchingParties = append(ally.MatchingParties, models.MatchingParty{
			PartyID:         ticket.PartyID,
			PartyMembers:    ticket.PartyMembers,
			PartyAttributes: ticket.PartyAttributes,
		})
	}
	return ally
}

func NewMatchmaker() *MatchMaker {
	return NewMatchmakerWithConfigOverride(nil)
}

func NewMatchmakerWithConfigOverride(overrideFunc func(cfg *config.Config)) *MatchMaker {
	configuration := &config.Config{}
	err := env.Parse(configuration)
	if err != nil {
		logrus.Fatal("unable to parse environment variables: ", err)
	}
	if overrideFunc != nil {
		overrideFunc(configuration)
	}
	return NewMatchMaker(configuration)
}

func TestMatchmaker1Team2PartyShouldNotRemovePartyAttribute(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1Success", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel" //nolint:goconst
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 1)
	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       1,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}
	channel := models.Channel{
		Ruleset: *ruleset,
	}

	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

	for _, ally := range results[0].MatchingAllies {
		for _, party := range ally.MatchingParties {
			assert.NotNil(t, party.PartyAttributes)
		}
	}
}

func TestMatchmaker_MatchOptions_SingleValuePartyAttributes_ShouldPresentOnTheResult(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1Team2PartyRespectPartyAttributes", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "pvheist" //nolint:goconst
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 1)

	mmRequests[0].PartyAttributes = map[string]interface{}{
		"cross_platform": []interface{}{"STEAM"},
		"PartyCode":      "12345",
	}
	mmRequests[1].PartyAttributes = map[string]interface{}{
		"cross_platform": []interface{}{"STEAM"},
		"PartyCode":      "54321",
	}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       1,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 4,
		},
		MatchOptions: models.MatchOptionRule{
			Options: []models.MatchOption{
				{Name: "cross_platform", Type: "all"},
				{Name: "PartyCode", Type: "all"},
			},
		},
	}
	channel := models.Channel{
		Ruleset: *ruleset,
	}

	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Len(t, results, 2, "should not match as PartyCode are different")

	// partyID to partyAttributes map
	expectedPartyAttributes := make(map[string]map[string]interface{})
	actualPartyAttributes := make(map[string]map[string]interface{})

	for _, request := range mmRequests {
		attributes := request.PartyAttributes
		expectedPartyAttributes[request.PartyID] = attributes
	}

	for _, result := range results {
		for _, ally := range result.MatchingAllies {
			for _, party := range ally.MatchingParties {
				actualPartyAttributes[party.PartyID] = result.PartyAttributes
			}
		}
	}

	assert.Equal(t, expectedPartyAttributes, actualPartyAttributes)
}

func TestMatchmaker1v1Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1Success", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel" //nolint:goconst
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 1)
	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchmaker1v5Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v5Success", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "battleroyale:solo"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 5, 1)
	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       5,
			MaxNumber:       5,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchmaker2v4Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker2v4Success", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "battleroyale:duo"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 3, 2)
	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       3,
			MaxNumber:       3,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 2,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchmaker6v6Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker6v6Success", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "6v6"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 16, 1)
	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 6,
			PlayerMaxNumber: 6,
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchmaker200Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker200Success", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "100royale"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 200, 1)
	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       100,
			MaxNumber:       100,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 2, "unexpected matchmaking result count. expected: %d, actual: %d", 2, len(results))
}

func TestMatchmaker5v10PlayerFulfilledSuccess(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker5v10PlayerFulfilledSuccess", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "battleroyale:squads"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 3, 5)
	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       3,
			MaxNumber:       3,
			PlayerMinNumber: 5,
			PlayerMaxNumber: 5,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchmaker5v10PlayerNotFulfilledSuccess(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker5v10PlayerNotFulfilledSuccess", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "battleroyale:squads"
	matchmaker := NewMatchmaker()
	mmRequest := generateRequest(channelName, 3, 3)
	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       3,
			MaxNumber:       3,
			PlayerMinNumber: 3,
			PlayerMaxNumber: 5,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequest, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

// // Comment below test, because the pivot will be the oldest,
// // and currently if the pivot has party attribute it should match with tickets who also have party attributes
// func TestMatchMakerPivotWithEmptyAttribute(t *testing.T) {
// 	t.Parallel()
// 	scope := envelope.NewRootScope(context.Background(), "TestMatchMakerPivotWithEmptyAttribute", "")
// 	t.Cleanup(func() { scope.Finish() })

// 	channelName := "fight:solo"
// 	matchmaker := NewMatchmaker()
// 	mmRequests := generateRequest(channelName, 2, 1)

// 	ruleset := &models.RuleSet{
// 		AllianceRule: models.AllianceRule{
// 			MinNumber:       1,
// 			MaxNumber:       1,
// 			PlayerMinNumber: 2,
// 			PlayerMaxNumber: 2,
// 		},
// 	}

// 	channel := models.Channel{
// 		Ruleset: *ruleset,
// 	}

// 	// t.Run("first request contains party attribute", func(t *testing.T) {
// 	// 	t.Parallel()
// 	// 	mmRequests[0].PartyAttributes = map[string]interface{}{
// 	// 		"map": "world",
// 	// 	}
// 	// 	mmRequests[1].PartyAttributes = map[string]interface{}{}
// 	//
// 	// 	results, _, err := matchmaker.MatchPlayers(scope, mmRequests, channel)
// 	// 	assert.NoError(t, err, "unable to execute matchmaking request")
// 	// 	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
// 	// })

// 	t.Run("first request contains empty party attribute", func(t *testing.T) {
// 		t.Parallel()
// 		mmRequests[0].PartyAttributes = map[string]interface{}{}
// 		mmRequests[1].PartyAttributes = map[string]interface{}{
// 			"map": "world",
// 		}

// 		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel, mmkr.NewWorkerInfo())
// 		assert.NoError(t, err, "unable to execute matchmaking request")
// 		assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
// 	})
// }

func TestMatchmakerPivotUnmatchableSuccess(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmakerPivotUnmatchableSuccess", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "fight:solo"
	matchmaker := NewMatchmaker()
	mmRequest := generateRequest(channelName, 3, 1)

	var mmr float64
	for i, req := range mmRequest {
		if i == 0 {
			mmr = 0
		} else {
			mmr = 10
		}
		for _, member := range req.PartyMembers {
			member.ExtraAttributes["mmr"] = mmr
		}
	}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(0),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequest, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchmakerAllyFindingSuccess(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmakerAllyFindingSuccess", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "moba:3v3v3"
	matchmaker := NewMatchmaker()
	mmRequest1 := generateRequest(channelName, 1, 2)
	mmRequest2 := generateRequest(channelName, 1, 3)
	mmRequest3 := generateRequest(channelName, 1, 1)
	mmRequest4 := generateRequest(channelName, 1, 1)
	mmRequest5 := generateRequest(channelName, 1, 1)
	mmRequest6 := generateRequest(channelName, 1, 1)

	mmRequest := mmRequest1
	mmRequest = append(mmRequest, mmRequest2...)
	mmRequest = append(mmRequest, mmRequest3...)
	mmRequest = append(mmRequest, mmRequest4...)
	mmRequest = append(mmRequest, mmRequest5...)
	mmRequest = append(mmRequest, mmRequest6...)

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       3,
			MaxNumber:       3,
			PlayerMinNumber: 3,
			PlayerMaxNumber: 3,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequest, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

	for _, result := range results {
		assert.Equal(t, 3, len(result.MatchingAllies), "unexpected matching allies")
		for _, ally := range result.MatchingAllies {
			playerCount := 0
			for _, party := range ally.MatchingParties {
				playerCount += len(party.PartyMembers)
			}
			assert.Equal(t, 3, playerCount, "unexpected matching party")
		}
	}
}

func TestMatchmakerAllyFindingFail(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmakerAllyFindingFail", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "moba:3v3"
	matchmaker := NewMatchmaker()
	mmRequest1 := generateRequest(channelName, 1, 2)
	mmRequest2 := generateRequest(channelName, 1, 3)

	mmRequest := mmRequest1
	mmRequest = append(mmRequest, mmRequest2...)

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 3,
			PlayerMaxNumber: 3,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequest, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 0, "unexpected matchmaking result count. expected: %d, actual: %d", 0, len(results))
}

func TestMatchmakerAllyFindingFail_NotEnoughMatchedPlayers(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmakerAllyFindingFail_NotEnoughMatchedPlayers", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "moba:3v3"
	matchmaker := NewMatchmaker()
	mmRequest1 := generateRequestWithMMR(channelName, 1, 2, 10)
	mmRequest2 := generateRequestWithMMR(channelName, 1, 3, 15)
	mmRequest3 := generateRequestWithMMR(channelName, 1, 1, 100)

	mmRequest := mmRequest1
	mmRequest = append(mmRequest, mmRequest2...)
	mmRequest = append(mmRequest, mmRequest3...)

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 3,
			PlayerMaxNumber: 3,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(50),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequest, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 0, "unexpected matchmaking result count. expected: %d, actual: %d", 0, len(results))
}

// nolint: dupl
func TestMatchmakerAllyFillingSuccess(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmakerAllyFillingSuccess", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "flexbattle:max10v10"
	matchmaker := NewMatchmaker()
	mmRequest1 := generateRequest(channelName, 1, 2)
	mmRequest2 := generateRequest(channelName, 1, 3)
	mmRequest3 := generateRequest(channelName, 1, 1)
	mmRequest4 := generateRequest(channelName, 1, 4)
	mmRequest5 := generateRequest(channelName, 1, 1)
	mmRequest6 := generateRequest(channelName, 1, 1)
	mmRequest7 := generateRequest(channelName, 1, 5)
	mmRequest8 := generateRequest(channelName, 1, 2)

	mmRequest := mmRequest1
	mmRequest = append(mmRequest, mmRequest2...)
	mmRequest = append(mmRequest, mmRequest3...)
	mmRequest = append(mmRequest, mmRequest4...)
	mmRequest = append(mmRequest, mmRequest5...)
	mmRequest = append(mmRequest, mmRequest6...)
	mmRequest = append(mmRequest, mmRequest7...)
	mmRequest = append(mmRequest, mmRequest8...)

	ensureSortedByAge(mmRequest)

	for i := range mmRequest {
		for j := range mmRequest[i].PartyMembers {
			// prevent random matching allies
			mmRequest[i].PartyMembers[j].ExtraAttributes["mmr"] = 500
		}
	}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 3,
			PlayerMaxNumber: 10,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequest, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

	for _, result := range results {
		assert.Equal(t, 2, len(result.MatchingAllies), "unexpected matching allies")
		for _, ally := range result.MatchingAllies {
			playerCount := 0
			for _, party := range ally.MatchingParties {
				playerCount += len(party.PartyMembers)
			}

			assert.GreaterOrEqual(t, len(ally.MatchingParties), 3, "unexpected matching party count")
			assert.Greater(t, playerCount, 3, "unexpected player count")
			assert.LessOrEqual(t, playerCount, 10, "unexpected player count")
		}
	}
}

// nolint: dupl
func TestMatchmakerAddingAllySuccess(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmakerAddingAllySuccess", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "flexbattle:max5v5v5v5"
	matchmaker := NewMatchmaker()
	mmRequest1 := generateRequest(channelName, 1, 2)
	mmRequest2 := generateRequest(channelName, 1, 3)
	mmRequest3 := generateRequest(channelName, 1, 1)
	mmRequest4 := generateRequest(channelName, 1, 4)
	mmRequest5 := generateRequest(channelName, 1, 1)
	mmRequest6 := generateRequest(channelName, 1, 1)
	mmRequest7 := generateRequest(channelName, 1, 5)
	mmRequest8 := generateRequest(channelName, 1, 2)

	mmRequest := mmRequest1
	mmRequest = append(mmRequest, mmRequest2...)
	mmRequest = append(mmRequest, mmRequest3...)
	mmRequest = append(mmRequest, mmRequest4...)
	mmRequest = append(mmRequest, mmRequest5...)
	mmRequest = append(mmRequest, mmRequest6...)
	mmRequest = append(mmRequest, mmRequest7...)
	mmRequest = append(mmRequest, mmRequest8...)

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       4,
			PlayerMinNumber: 3,
			PlayerMaxNumber: 5,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequest, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

	for _, result := range results {
		assert.Equal(t, 4, len(result.MatchingAllies), "unexpected matching allies")

		for _, ally := range result.MatchingAllies {
			playerCount := 0
			for _, party := range ally.MatchingParties {
				playerCount += len(party.PartyMembers)
			}

			assert.GreaterOrEqual(t, len(ally.MatchingParties), 1, "unexpected matching party count")
			assert.GreaterOrEqual(t, playerCount, 3, "unexpected player count")
			assert.LessOrEqual(t, playerCount, 5, "unexpected player count")
		}
	}
}

func TestMatchmaker1v1WithClientVersionSuccess(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1WithClientVersionSuccess", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 1)
	// add client version
	for i := 0; i < len(mmRequests); i++ {
		mmRequests[i].PartyAttributes = map[string]interface{}{models.AttributeClientVersion: "123"}
	}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

	for _, v := range results {
		assert.NotEmpty(t, v.ClientVersion, "client version should not empty")
	}
}

func TestMatchmaker_WithClientVersion_Blank_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_WithClientVersion_Blank_Failed", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 1)
	// add client version
	mmRequests[0].PartyAttributes = map[string]interface{}{models.AttributeClientVersion: ""}
	mmRequests[1].PartyAttributes = map[string]interface{}{models.AttributeClientVersion: "123"}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 2,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Equal(t, 0, len(results), "unexpected matchmaking result count")
}

func TestMatchmaker_WithServerName_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_WithServerName_Success", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel"
	serverName := "v123"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 1)
	// add client version
	for i := range mmRequests {
		mmRequests[i].PartyAttributes = map[string]interface{}{models.AttributeServerName: serverName}
	}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 2,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Equal(t, 1, len(results), "unexpected matchmaking result count")

	for _, v := range results {
		assert.Equal(t, serverName, v.ServerName, "server name should not be empty")
	}
}

func TestMatchmaker_WithServerName_Blank_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_WithServerName_Blank_Failed", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 1)
	// add client version
	mmRequests[0].PartyAttributes = map[string]interface{}{models.AttributeServerName: ""}
	mmRequests[1].PartyAttributes = map[string]interface{}{models.AttributeServerName: "v123"}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 2,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Equal(t, 0, len(results), "unexpected matchmaking result count")
}

func TestMatchmaker1v1WithLatencySuccess(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1WithLatencySuccess", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 4, 1)

	// add latencies
	mmRequests[0].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "us": 100 }`}
	mmRequests[0].LatencyMap = map[string]int{"us": 100}
	mmRequests[0].SortedLatency = []models.Region{{Region: "us", Latency: 100}}
	mmRequests[1].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "eu": 100 }`}
	mmRequests[1].LatencyMap = map[string]int{"eu": 100}
	mmRequests[1].SortedLatency = []models.Region{{Region: "eu", Latency: 100}}
	mmRequests[2].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "asia": 100 }`}
	mmRequests[2].LatencyMap = map[string]int{"asia": 100}
	mmRequests[2].SortedLatency = []models.Region{{Region: "asia", Latency: 100}}
	mmRequests[3].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "us": 100 }`}
	mmRequests[3].LatencyMap = map[string]int{"us": 100}
	mmRequests[3].SortedLatency = []models.Region{{Region: "us", Latency: 100}}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
	assert.Equal(t, "us", results[0].Region, "region should be US")
	assert.Contains(t, results[0].MatchingAllies[0].MatchingParties[0].PartyAttributes[models.AttributeLatencies], "us", "region should be US")
	assert.Contains(t, results[0].MatchingAllies[1].MatchingParties[0].PartyAttributes[models.AttributeLatencies], "us", "region should be US")
}

func TestMatchmaker1v1WithLatencySuccess_SecondTry(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1WithLatencySuccess_SecondTry", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 4, 1)

	// add latencies
	mmRequests[0].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "us": 100, "asia": 100 }`, models.AttributeMatchAttempt: float64(1)}
	mmRequests[0].LatencyMap = map[string]int{"us": 100, "asia": 100}
	mmRequests[0].SortedLatency = []models.Region{{Region: "us", Latency: 100}, {Region: "asia", Latency: 100}}
	mmRequests[1].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "eu": 100 }`}
	mmRequests[1].LatencyMap = map[string]int{"eu": 100}
	mmRequests[1].SortedLatency = []models.Region{{Region: "eu", Latency: 100}}
	mmRequests[2].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "asia": 100 }`}
	mmRequests[2].LatencyMap = map[string]int{"asia": 100}
	mmRequests[2].SortedLatency = []models.Region{{Region: "asia", Latency: 100}}
	mmRequests[3].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "australia": 100 }`}
	mmRequests[3].LatencyMap = map[string]int{"australia": 100}
	mmRequests[3].SortedLatency = []models.Region{{Region: "australia", Latency: 100}}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
	assert.Equal(t, "asia", results[0].Region, "region should be Asia")
	assert.Contains(t, results[0].MatchingAllies[0].MatchingParties[0].PartyAttributes[models.AttributeLatencies], "asia", "region should be Asia")
	assert.Contains(t, results[0].MatchingAllies[1].MatchingParties[0].PartyAttributes[models.AttributeLatencies], "asia", "region should be Asia")
}

func TestMatchmaker1v1ShouldNotMatchWithUnselectedDifferentRegion(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1ShouldNotMatchWithUnselectedDifferentRegion", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 4, 1)

	// add latencies
	mmRequests[0].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "us": 100 }`, models.AttributeMatchAttempt: float64(1)}
	mmRequests[0].LatencyMap = map[string]int{"us": 100}
	mmRequests[0].SortedLatency = []models.Region{{Region: "us", Latency: 100}}
	mmRequests[1].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "eu": 100 }`}
	mmRequests[1].LatencyMap = map[string]int{"eu": 100}
	mmRequests[1].SortedLatency = []models.Region{{Region: "eu", Latency: 100}}
	mmRequests[2].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "asia": 100 }`}
	mmRequests[2].LatencyMap = map[string]int{"asia": 100}
	mmRequests[2].SortedLatency = []models.Region{{Region: "asia", Latency: 100}}
	mmRequests[3].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "pacific": 100 }`}
	mmRequests[3].LatencyMap = map[string]int{"pacific": 100}
	mmRequests[3].SortedLatency = []models.Region{{Region: "pacific", Latency: 100}}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1000),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	require.NoError(t, err, "unable to execute matchmaking request")
	require.Len(t, results, 0)
}

// func TestMatchmaker1v1ShouldMatchWithHighPing(t *testing.T) {
// 	t.Parallel()
// 	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1ShouldMatchWithHighPing")
// 	t.Cleanup(func() { scope.Finish() })
//
// 	channelName := "chess:duel"
// 	matchmaker := NewMatchmaker()
// 	mmRequests := generateRequest(channelName, 2, 1)
//
// 	// add latencies
// 	mmRequests[0].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "eu": 100 }`, models.AttributeMatchAttempt: float64(3)}
// 	mmRequests[0].LatencyMap = map[string]int{"eu": 100}
// 	mmRequests[0].SortedLatency = []models.Region{{Region: "eu", Latency: 100}}
// 	// latency upper bound should be: latency + (35 * (attempt + 1))
// 	// 100 + (35 * (3 + 1)) = 240
// 	mmRequests[1].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "eu": 240 }`}
// 	mmRequests[1].LatencyMap = map[string]int{"eu": 240}
// 	mmRequests[1].SortedLatency = []models.Region{{Region: "eu", Latency: 240}}
//
// 	ruleset := &models.RuleSet{
// 		AllianceRule: models.AllianceRule{
// 			MinNumber:       2,
// 			MaxNumber:       2,
// 			PlayerMinNumber: 1,
// 			PlayerMaxNumber: 1,
// 		},
// 		MatchingRule: []models.MatchingRule{
// 			{
// 				Attribute: "mmr",
// 				Criteria:  "distance",
// 				Reference: float64(1000),
// 			},
// 		},
// 	}
//
// 	channel := models.Channel{
// 		Ruleset: *ruleset,
// 	}
// 	results, _, err := matchmaker.MatchPlayers(scope, mmRequests, channel)
// 	require.NoError(t, err, "unable to execute matchmaking request")
// 	require.Len(t, results, 1)
// 	require.Contains(t, results[0].MatchingAllies[0].MatchingParties[0].PartyAttributes[models.AttributeLatencies], "eu")
// 	require.Contains(t, results[0].MatchingAllies[1].MatchingParties[0].PartyAttributes[models.AttributeLatencies], "eu")
// }

// func TestMatchmaker1v1ShouldMatch220PingOn2ndAttempt(t *testing.T) {
// 	t.Parallel()
// 	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1ShouldMatch220PingOn2ndAttempt")
// 	t.Cleanup(func() { scope.Finish() })
//
// 	channelName := "chess:duel"
// 	matchmaker := NewMatchmaker()
// 	mmRequests := generateRequest(channelName, 2, 1)
//
// 	var results []*models.MatchmakingResult
// 	var err error
// 	for matchAttempt := 0; matchAttempt < 2; matchAttempt++ {
// 		mmRequests[0].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "eu": 100 }`, models.AttributeMatchAttempt: float64(matchAttempt)}
// 		mmRequests[0].LatencyMap = map[string]int{"eu": 100}
// 		mmRequests[0].SortedLatency = []models.Region{{Region: "eu", Latency: 100}}
// 		// latency upper bound should be: latency + (35 * (attempt + 1))
// 		// 100 + (35 * (1 + 1)) = 170
// 		mmRequests[1].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "eu": 170 }`}
// 		mmRequests[1].LatencyMap = map[string]int{"eu": 170}
// 		mmRequests[1].SortedLatency = []models.Region{{Region: "eu", Latency: 170}}
//
// 		ruleset := &models.RuleSet{
// 			AllianceRule: models.AllianceRule{
// 				MinNumber:       2,
// 				MaxNumber:       2,
// 				PlayerMinNumber: 1,
// 				PlayerMaxNumber: 1,
// 			},
// 			MatchingRule: []models.MatchingRule{
// 				{
// 					Attribute: "mmr",
// 					Criteria:  "distance",
// 					Reference: float64(1000),
// 				},
// 			},
// 		}
//
// 		channel := models.Channel{
// 			Ruleset: *ruleset,
// 		}
// 		results, _, err = matchmaker.MatchPlayers(scope, mmRequests, channel)
// 	}
//
// 	require.NoError(t, err, "unable to execute matchmaking request")
// 	require.Len(t, results, 1)
// 	require.Contains(t, results[0].MatchingAllies[0].MatchingParties[0].PartyAttributes[models.AttributeLatencies], "eu")
// 	require.Contains(t, results[0].MatchingAllies[1].MatchingParties[0].PartyAttributes[models.AttributeLatencies], "eu")
// }

func TestMatchmaker_WithMMR_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_WithMMR_Success", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "2v2"
	matchmaker := NewMatchmaker()
	mmRequest1 := generateRequestWithMMR(channelName, 1, 2, 78)
	mmRequest2 := generateRequestWithMMR(channelName, 1, 2, 80)

	mmRequests := mmRequest1
	mmRequests = append(mmRequests, mmRequest2...)

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 2,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(10),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchmaker_WithMMR_Distance0(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_WithMMR_Distance0", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "1v1"
	matchmaker := NewMatchmaker()
	mmRequest1 := generateRequestWithMMR(channelName, 1, 1, 79)
	mmRequest2 := generateRequestWithMMR(channelName, 1, 1, 80)

	mmRequests := mmRequest1
	mmRequests = append(mmRequests, mmRequest2...)

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(0),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 0, "unexpected matchmaking result count. expected: %d, actual: %d", 0, len(results))
}

func TestMatchmaker_WithMMR_Distance1(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_WithMMR_Distance1", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "1v1"
	matchmaker := NewMatchmaker()
	mmRequest1 := generateRequestWithMMR(channelName, 1, 1, 79)
	mmRequest2 := generateRequestWithMMR(channelName, 1, 1, 80)

	mmRequests := mmRequest1
	mmRequests = append(mmRequests, mmRequest2...)

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(1),
			},
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchmaker1v1_Blocked(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1_Blocked", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel" //nolint:goconst
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 1)
	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	}

	mmRequests[0].PartyAttributes[models.AttributeBlocked] = []interface{}{mmRequests[1].PartyMembers[0].UserID}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 0, "unexpected matchmaking result count. expected: %d, actual: %d", 0, len(results))
}

func TestMatchmaker_MatchOptions(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_MatchOptions", "")
	t.Cleanup(func() { scope.Finish() })

	type args struct {
		tickets []models.MatchmakingRequest
		channel models.Channel
	}
	type testItem struct {
		name                  string
		args                  args
		wantMatchmakingResult []*models.MatchmakingResult
		// wantSatisfiedTickets  []*models.MatchmakingRequest
		wantErr bool
	}
	tests := []testItem{}

	contains := func(tt *testing.T, got, expected *models.MatchmakingResult) {
		tt.Helper()
		expectedCount := 0
		for _, expectedAlly := range expected.MatchingAllies {
			expectedCount += len(expectedAlly.MatchingParties)
		}
		gotCount := 0
	gotloop:
		for _, gotAlly := range got.MatchingAllies {
			for _, gotParty := range gotAlly.MatchingParties {
				for _, expectedAlly := range expected.MatchingAllies {
					for _, expectedParty := range expectedAlly.MatchingParties {
						if expectedParty.PartyID == gotParty.PartyID {
							gotCount++
							continue gotloop
						}
					}
				}
			}
		}
		if !assert.Equal(tt, expectedCount, gotCount, "unexpected party found in match result") {
			fmt.Println("got:")
			spew.Dump(got)
			fmt.Println("expected:")
			spew.Dump(expected)
		}
	}

	// case 1
	{
		tickets := generateRequest("", 3, 1)
		tickets[1].PartyAttributes["language"] = []interface{}{"en"}
		tickets[2].PartyAttributes["language"] = []interface{}{"en"}
		tests = append(tests, testItem{
			name: "any | should prefer ticket with existing attribute",
			args: args{
				tickets: tickets,
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 1,
						},
						MatchOptions: models.MatchOptionRule{
							Options: []models.MatchOption{
								{
									Name: "language",
									Type: models.MatchOptionTypeAny,
								},
							},
						},
					},
				},
			},
			wantMatchmakingResult: []*models.MatchmakingResult{
				{
					MatchingAllies: []models.MatchingAlly{
						createMatchingAlly(tickets[1]),
						createMatchingAlly(tickets[2]),
					},
				},
			},
		})
	}

	// case 2
	{
		tickets := generateRequest("", 3, 1)

		tickets[0].PartyAttributes["maps"] = []interface{}{"a", "b"}

		tickets[1].PartyAttributes["maps"] = []interface{}{"b"}

		tickets[2].PartyAttributes["maps"] = []interface{}{"c", "d"}

		tests = append(tests, testItem{
			name: "any | should find same map item in multivalue",
			args: args{
				tickets: tickets,
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 1,
						},
						MatchOptions: models.MatchOptionRule{
							Options: []models.MatchOption{
								{
									Name: "maps",
									Type: models.MatchOptionTypeAny,
								},
							},
						},
					},
				},
			},
			wantMatchmakingResult: []*models.MatchmakingResult{
				{
					MatchingAllies: []models.MatchingAlly{
						createMatchingAlly(tickets[0]),
						createMatchingAlly(tickets[1]),
					},
				},
			},
		})
	}

	// case 3
	{
		tickets := generateRequest("", 3, 1)

		tickets[0].PartyAttributes["maps"] = []interface{}{"c"}

		tickets[1].PartyAttributes["maps"] = []interface{}{"b"}

		tickets[2].PartyAttributes["maps"] = []interface{}{"a"}

		tests = append(tests, testItem{
			name: "any | should not find match",
			args: args{
				tickets: tickets,
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 1,
						},
						MatchOptions: models.MatchOptionRule{
							Options: []models.MatchOption{
								{
									Name: "maps",
									Type: models.MatchOptionTypeAny,
								},
							},
						},
					},
				},
			},
			wantMatchmakingResult: []*models.MatchmakingResult{},
		})
	}

	// case 4
	{
		tickets := generateRequest("", 3, 1)

		createdAt := tickets[0].CreatedAt

		tickets[0].PartyAttributes["maps"] = []interface{}{"b", "c"}
		tickets[0].PartyAttributes["language"] = []interface{}{"en", "fr"}

		tickets[1].PartyAttributes["maps"] = []interface{}{"b"}
		tickets[1].PartyAttributes["language"] = []interface{}{"en", "de"}

		tickets[2].PartyAttributes["maps"] = []interface{}{"c"}
		tickets[2].PartyAttributes["language"] = []interface{}{"en", "fr"}

		for i := range tickets {
			tickets[i].CreatedAt = createdAt + int64(i*10)
		}

		tests = append(tests, testItem{
			name: "combination | should respect match all",
			args: args{
				tickets: tickets,
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 1,
						},
						MatchOptions: models.MatchOptionRule{
							Options: []models.MatchOption{
								{
									Name: "maps",
									Type: models.MatchOptionTypeAny,
								},
								{
									Name: "language",
									Type: models.MatchOptionTypeAll,
								},
							},
						},
					},
				},
			},
			wantMatchmakingResult: []*models.MatchmakingResult{
				{
					MatchingAllies: []models.MatchingAlly{
						createMatchingAlly(tickets[0]),
						createMatchingAlly(tickets[2]),
					},
				},
			},
		})
	}

	// case 5
	{
		tickets := generateRequest("", 3, 1)

		tickets[0].PartyAttributes["lane"] = []interface{}{"top"}

		tickets[1].PartyAttributes["lane"] = []interface{}{"top"}

		// ensure first ticket is the oldest
		ensureSortedByAge(tickets)

		tests = append(tests, testItem{
			name: "unique | should not get match",
			args: args{
				tickets: tickets,
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 1,
						},
						MatchOptions: models.MatchOptionRule{
							Options: []models.MatchOption{
								{
									Name: "lane",
									Type: models.MatchOptionTypeUnique,
								},
							},
						},
					},
				},
			},
			wantMatchmakingResult: []*models.MatchmakingResult{},
		})
	}

	// case 6
	{
		tickets := generateRequest("", 1, 1)
		tickets[0].PartyAttributes["lane"] = []interface{}{"top"}

		tests = append(tests, testItem{
			name: "any | should able to get match for single party / solo",
			args: args{
				tickets: tickets,
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							MinNumber:       1,
							MaxNumber:       1,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 1,
						},
						MatchOptions: models.MatchOptionRule{
							Options: []models.MatchOption{
								{
									Name: "lane",
									Type: models.MatchOptionTypeAny,
								},
							},
						},
					},
				},
			},
			wantMatchmakingResult: []*models.MatchmakingResult{
				{
					MatchingAllies: []models.MatchingAlly{
						createMatchingAlly(tickets[0]),
					},
				},
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mm := NewMatchmaker()
			results, _, err := mm.MatchPlayers(scope, "", "", tt.args.tickets, tt.args.channel)
			if (err != nil) != tt.wantErr {
				t.Errorf("Matchmaker.MatchPlayers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, len(tt.wantMatchmakingResult), len(results), "unexpected matchmaking result count. testname: %s, expected: %d, actual: %d", tt.name, len(tt.wantMatchmakingResult), len(results))

			if len(tt.wantMatchmakingResult) == 0 {
				return
			}

			for i, result := range results {
				contains(t, result, tt.wantMatchmakingResult[i])
			}
		})
	}
}

func TestMatchmaker_MatchOptionsResultAttributes(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_MatchOptions", "")
	t.Cleanup(func() { scope.Finish() })

	type args struct {
		tickets []models.MatchmakingRequest
		channel models.Channel
	}
	type testItem struct {
		name                  string
		args                  args
		wantMatchmakingResult []*models.MatchmakingResult
		wantAttributes        map[string]interface{}
		wantErr               bool
	}
	tests := []testItem{}

	contains := func(tt *testing.T, got, expected *models.MatchmakingResult) {
		tt.Helper()
		expectedCount := 0
		for _, expectedAlly := range expected.MatchingAllies {
			expectedCount += len(expectedAlly.MatchingParties)
		}
		gotCount := 0
	gotloop:
		for _, gotAlly := range got.MatchingAllies {
			for _, gotParty := range gotAlly.MatchingParties {
				for _, expectedAlly := range expected.MatchingAllies {
					for _, expectedParty := range expectedAlly.MatchingParties {
						if expectedParty.PartyID == gotParty.PartyID {
							gotCount++
							continue gotloop
						}
					}
				}
			}
		}
		if !assert.Equal(tt, expectedCount, gotCount, "unexpected party found in match result") {
			fmt.Println("got:")
			spew.Dump(got)
			fmt.Println("expected:")
			spew.Dump(expected)
		}
	}

	// case 1
	{
		partyAttribute := make(map[string]interface{})
		err := json.Unmarshal([]byte(`{"mm_configuration":1}`), &partyAttribute)
		require.NoError(t, err)

		tickets := generateRequest("", 3, 1)
		tickets[1].PartyAttributes = partyAttribute
		tickets[2].PartyAttributes = partyAttribute
		tests = append(tests, testItem{
			name: "any | preserved single value",
			args: args{
				tickets: tickets,
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 1,
						},
						MatchOptions: models.MatchOptionRule{
							Options: []models.MatchOption{
								{
									Name: "mm_configuration",
									Type: models.MatchOptionTypeAny,
								},
							},
						},
					},
				},
			},
			wantMatchmakingResult: []*models.MatchmakingResult{
				{
					MatchingAllies: []models.MatchingAlly{
						createMatchingAlly(tickets[1]),
						createMatchingAlly(tickets[2]),
					},
				},
			},
			wantAttributes: partyAttribute,
		})
	}

	// case 2
	{
		partyAttribute := make(map[string]interface{})
		err := json.Unmarshal([]byte(`{"mm_configuration":["any"]}`), &partyAttribute)
		require.NoError(t, err)

		tickets := generateRequest("", 3, 1)
		tickets[1].PartyAttributes = partyAttribute
		tickets[2].PartyAttributes = partyAttribute
		tests = append(tests, testItem{
			name: "any | preserved multi values",
			args: args{
				tickets: tickets,
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 1,
						},
						MatchOptions: models.MatchOptionRule{
							Options: []models.MatchOption{
								{
									Name: "mm_configuration",
									Type: models.MatchOptionTypeAny,
								},
							},
						},
					},
				},
			},
			wantMatchmakingResult: []*models.MatchmakingResult{
				{
					MatchingAllies: []models.MatchingAlly{
						createMatchingAlly(tickets[1]),
						createMatchingAlly(tickets[2]),
					},
				},
			},
			wantAttributes: partyAttribute,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mm := NewMatchmaker()
			results, _, err := mm.MatchPlayers(scope, "", "", tt.args.tickets, tt.args.channel)
			if (err != nil) != tt.wantErr {
				t.Errorf("Matchmaker.MatchPlayers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, len(tt.wantMatchmakingResult), len(results), "unexpected matchmaking result count. testname: %s, expected: %d, actual: %d", tt.name, len(tt.wantMatchmakingResult), len(results))

			if len(tt.wantMatchmakingResult) == 0 {
				return
			}

			for i, result := range results {
				contains(t, result, tt.wantMatchmakingResult[i])
				assert.Equal(t, tt.wantAttributes, result.PartyAttributes)
			}
		})
	}
}

func TestMatchmaker1v1_AvoidMatchWithSelf(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1_AvoidMatchWithSelf", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel" //nolint:goconst
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 1)
	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	}

	// party index 1 has same user ID
	mmRequests[1].PartyMembers[0].UserID = mmRequests[0].PartyMembers[0].UserID

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 0, "unexpected matchmaking result count. expected: %d, actual: %d", 0, len(results))
}

func TestMatchmakerGameModeAllianceFlexingRuleActiveShouldMatch(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestMatchmakerAllianceFlexingRuleActiveShouldMatch", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "flex:2v2"
	matchmaker := NewMatchmaker()
	mmRequests := []models.MatchmakingRequest{
		{
			PartyID:   "partyA",
			Channel:   channelName,
			CreatedAt: time.Date(2021, 12, 10, 10, 0, 0, 0, time.UTC).Unix(),
			PartyMembers: []models.PartyMember{
				{UserID: "userA"},
			},
			PartyAttributes: map[string]interface{}{
				models.AttributeMemberAttr: map[string]interface{}{
					"mmr":  float64(100),
					"ping": 50,
				},
			},
		},
		{
			PartyID:   "partyB",
			Channel:   channelName,
			CreatedAt: time.Date(2021, 12, 10, 10, 0, 0, 0, time.UTC).Unix(),
			PartyMembers: []models.PartyMember{
				{UserID: "userB"},
			},
			PartyAttributes: map[string]interface{}{
				models.AttributeMemberAttr: map[string]interface{}{
					"mmr":  float64(100),
					"ping": 50,
				},
			},
		},
	}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 2,
		},
		AllianceFlexingRule: []models.AllianceFlexingRule{
			{
				Duration: 20,
				AllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 1,
				},
			},
		},
	}
	channel := models.Channel{
		Ruleset: *ruleset,
	}

	Now = func() time.Time { return time.Date(2021, 12, 10, 10, 0, 30, 0, time.UTC) }

	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchmakerGameModeAllianceFlexingRuleInactiveShouldNotMatch(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestMatchmakerGameModeAllianceFlexingRuleInactiveShouldNotMatch", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "flex:2v2"
	matchmaker := NewMatchmaker()
	mmRequests := []models.MatchmakingRequest{
		{
			PartyID:   "partyA",
			Channel:   channelName,
			CreatedAt: time.Date(2021, 12, 10, 10, 0, 0, 0, time.UTC).Unix(),
			PartyMembers: []models.PartyMember{
				{UserID: "userA"},
			},
			PartyAttributes: map[string]interface{}{
				models.AttributeMemberAttr: map[string]interface{}{
					"mmr":  float64(100),
					"ping": 50,
				},
			},
		},
		{
			PartyID:   "partyB",
			Channel:   channelName,
			CreatedAt: time.Date(2021, 12, 10, 10, 0, 0, 0, time.UTC).Unix(),
			PartyMembers: []models.PartyMember{
				{UserID: "userB"},
			},
			PartyAttributes: map[string]interface{}{
				models.AttributeMemberAttr: map[string]interface{}{
					"mmr":  float64(100),
					"ping": 50,
				},
			},
		},
	}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 2,
		},
		AllianceFlexingRule: []models.AllianceFlexingRule{
			{
				Duration: 40,
				AllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 1,
				},
			},
		},
	}
	channel := models.Channel{
		Ruleset: *ruleset,
	}

	Now = func() time.Time { return time.Date(2021, 12, 10, 10, 0, 30, 0, time.UTC) }

	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 0, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchmakerWithNewSessionOnlyAttr(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker1v1Success", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel" //nolint:goconst
	matchmaker := NewMatchmaker()
	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}
	channel := models.Channel{
		Ruleset: ruleset,
	}

	// adding parameter new_session_only
	requests := generateRequest(channelName, 2, 1)
	requests[0].PartyAttributes = map[string]interface{}{
		models.AttributeNewSessionOnly: "true",
	}
	requests[1].PartyAttributes = map[string]interface{}{
		models.AttributeNewSessionOnly: "false",
	}
	results, satisfiedTickets, err := matchmaker.MatchPlayers(scope, "", "", requests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Equal(t, requests, satisfiedTickets, "satisfied tickets count should be %d", len(satisfiedTickets))
	expectedCount := 1
	assert.Truef(t, len(results) == expectedCount, "unexpected matchmaking result count. expected: %d, actual: %d", expectedCount, len(results))
}

func Test_resetTicket(t *testing.T) {
	type args struct {
		dest   []models.MatchmakingRequest
		source []models.MatchmakingRequest
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test 1",
			args: args{
				dest: []models.MatchmakingRequest{
					{
						PartyID: "party1",
						PartyMembers: []models.PartyMember{
							{
								UserID: "user a",
								ExtraAttributes: map[string]interface{}{
									models.ROLE: "fighter",
								},
							},
						},
					},
				},
				source: []models.MatchmakingRequest{
					{
						PartyID: "party1",
						PartyMembers: []models.PartyMember{
							{
								UserID: "user a",
								ExtraAttributes: map[string]interface{}{
									models.ROLE: `["fighter","tank"]`,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, "fighter", tt.args.dest[0].PartyMembers[0].ExtraAttributes[models.ROLE], "before reset, dest role should be 1")

			resetTicket(tt.args.dest, tt.args.source)

			assert.Equal(t, `["fighter","tank"]`, tt.args.dest[0].PartyMembers[0].ExtraAttributes[models.ROLE], "after reset, dest role should be 2")
		})
	}
}

func TestMatchPlayer_RegionRate_Success(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestMatchPlayer_RegionRate_Success", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 1)
	Now = func() time.Time { return time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC) }
	defer func() { Now = time.Now }()

	// add latencies
	mmRequests[0].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "us": 100 }`, models.AttributeMatchAttempt: float64(1)}
	mmRequests[0].LatencyMap = map[string]int{"us": 100, "eu": 100}
	mmRequests[0].SortedLatency = []models.Region{{Region: "us", Latency: 100}, {Region: "eu", Latency: 100}}
	mmRequests[0].CreatedAt = Now().Add(-time.Second * 2).Unix()
	mmRequests[1].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "eu": 100 }`}
	mmRequests[1].LatencyMap = map[string]int{"eu": 100}
	mmRequests[1].SortedLatency = []models.Region{{Region: "eu", Latency: 100}}
	mmRequests[1].CreatedAt = Now().Unix()

	ruleset := &models.RuleSet{
		RegionExpansionRateMs: 1000,
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	require.NoError(t, err, "unable to execute matchmaking request")
	require.Len(t, results, 1)
}

func TestMatchPlayer_RegionRate_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchPlayer_RegionRate_Failed", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 3, 1)

	// add latencies
	mmRequests[0].LatencyMap = map[string]int{"us": 100, "eu": 110}
	mmRequests[0].SortedLatency = []models.Region{{Region: "us", Latency: 100}, {Region: "eu", Latency: 110}}
	mmRequests[0].CreatedAt = time.Now().Unix()
	mmRequests[1].LatencyMap = map[string]int{"eu": 100}
	mmRequests[1].SortedLatency = []models.Region{{Region: "eu", Latency: 100}}
	mmRequests[1].CreatedAt = time.Now().Unix()
	mmRequests[2].LatencyMap = map[string]int{"us": 90, "ap": 80}
	mmRequests[2].SortedLatency = []models.Region{{Region: "ap", Latency: 80}, {Region: "us", Latency: 90}}
	mmRequests[2].CreatedAt = time.Now().Unix()

	ruleset := &models.RuleSet{
		RegionExpansionRateMs: 10000,
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	require.NoError(t, err, "unable to execute matchmaking request")
	require.Len(t, results, 0)
}

func TestMatchmaker_WithServerName_Issue_Team_Balance_With_Empty_Alliance(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_WithServerName_Issue_Team_Balance_With_Empty_Alliance", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "chess:duel"
	matchmaker := NewMatchmaker()
	mmRequests := generateRequest(channelName, 2, 2)

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 4,
		},
		MatchingRule: []models.MatchingRule{},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)
	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Equal(t, 1, len(results), "unexpected matchmaking result count")
}

func TestMatchmakerAllianceFlexingRuleActiveShouldMatch(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestMatchmakerAllianceFlexingRuleActiveShouldMatch", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "flex:4players"
	matchmaker := NewMatchmaker()
	mmRequests := []models.MatchmakingRequest{
		{
			PartyID:   "partyA",
			Channel:   channelName,
			CreatedAt: time.Date(2021, 12, 10, 10, 0, 0, 0, time.UTC).Unix(),
			PartyMembers: []models.PartyMember{
				{UserID: "userA"},
			},
			PartyAttributes: map[string]interface{}{},
		},
		{
			PartyID:   "partyB",
			Channel:   channelName,
			CreatedAt: time.Date(2021, 12, 10, 10, 0, 0, 0, time.UTC).Unix(),
			PartyMembers: []models.PartyMember{
				{UserID: "userB"},
			},
			PartyAttributes: map[string]interface{}{},
		},
	}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       1,
			PlayerMinNumber: 4,
			PlayerMaxNumber: 4,
		},
		AllianceFlexingRule: []models.AllianceFlexingRule{
			{
				Duration: 10,
				AllianceRule: models.AllianceRule{
					MinNumber:       1,
					MaxNumber:       1,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 4,
				},
			},
		},
	}
	channel := models.Channel{
		Ruleset: *ruleset,
	}

	Now = func() time.Time { return time.Date(2021, 12, 10, 10, 0, 20, 0, time.UTC) }

	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 0, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

	mmRequests = append(mmRequests, models.MatchmakingRequest{
		PartyID:   "partyC",
		Channel:   channelName,
		CreatedAt: time.Date(2021, 12, 10, 10, 0, 0, 0, time.UTC).Unix(),
		PartyMembers: []models.PartyMember{
			{UserID: "userC"},
		},
		PartyAttributes: map[string]interface{}{},
	})

	results, _, err = matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	assert.NoError(t, err, "unable to execute matchmaking request")
	assert.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
}

func TestMatchPlayers_SinglePlayer(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchPlayers_SinglePlayer", "")
	t.Cleanup(func() { scope.Finish() })

	t.Run("successful single player match", func(t *testing.T) {
		matchRequestCount := 5
		matchResultCount := 5

		channelName := "chess:solo"
		matchmaker := NewMatchmaker()
		mmRequests := generateRequest(channelName, matchRequestCount, 1)
		ruleset := &models.RuleSet{
			AllianceRule: models.AllianceRule{
				MinNumber:       1,
				MaxNumber:       1,
				PlayerMinNumber: 1,
				PlayerMaxNumber: 1,
			},
		}

		channel := models.Channel{
			Ruleset: *ruleset,
		}
		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

		assert.Nil(t, err)
		assert.Equal(t, len(results), matchResultCount)
		assert.Equal(t, len(results[0].MatchingAllies), 1)
	})

	t.Run("not using single player handler when max player is more than one", func(t *testing.T) {
		matchRequestCount := 5
		matchResultCount := 5

		channelName := "chess:solo"
		matchmaker := NewMatchmaker()
		mmRequests := generateRequest(channelName, matchRequestCount, 1)
		ruleset := &models.RuleSet{
			AllianceRule: models.AllianceRule{
				MinNumber:       1,
				MaxNumber:       1,
				PlayerMinNumber: 1,
				PlayerMaxNumber: 2,
			},
		}

		channel := models.Channel{
			Ruleset: *ruleset,
		}
		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

		assert.Nil(t, err)
		assert.NotEqual(t, len(results), matchResultCount)
	})
}

// TestMatchmaker_BlockedPlayerCannotMatch generate match requests with 2 players, 1 of them blocked the other, they should be in separated match result
func TestMatchmaker_BlockedPlayerCannotMatch(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_BlockedPlayerCannotMatch", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "test:" + utils.GenerateUUID()
	matchmaker := NewMatchmaker()
	reqs := generateRequest(channelName, 2, 1)

	// set blocked
	reqs[0].PartyAttributes[models.AttributeBlocked] = []interface{}{"", reqs[1].PartyMembers[0].UserID}

	ensureSortedByAge(reqs)

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       1,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", reqs, channel)
	require.NoError(t, err)
	require.Len(t, results, 2)

	for _, r := range results {
		require.Len(t, r.GetMemberUserIDs(), 1)
	}
}

// TestMatchmaker_BlockedPlayerCanMatchOnDifferentTeam generate match requests with 2 players, 1 of them blocked the other, both player should able to be match but in different team
func TestMatchmaker_BlockedPlayerCanMatchOnDifferentTeam(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_BlockedPlayerCanMatchOnDifferentTeam", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "test:" + utils.GenerateUUID()
	matchmaker := NewMatchmaker()
	reqs := generateRequest(channelName, 2, 1)

	// set blocked
	reqs[0].PartyAttributes[models.AttributeBlocked] = []interface{}{"", reqs[1].PartyMembers[0].UserID}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
		BlockedPlayerOption: models.BlockedPlayerCanMatchOnDifferentTeam,
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}
	results, _, err := matchmaker.MatchPlayers(scope, "", "", reqs, channel)
	require.NoError(t, err)
	require.Len(t, results, 1)

	for _, r := range results {
		require.Len(t, r.GetMemberUserIDs(), 2)
		for _, ally := range r.MatchingAllies {
			require.Len(t, ally.GetMemberUserIDs(), 1)
		}
	}
}

// TestMatchmaker_BlockedPlayerCanMatchOnDifferentTeam_Case2 generate match requests with 3 players, they block each other, all of them should able to be match and respect block
func TestMatchmaker_BlockedPlayerCanMatchOnDifferentTeam_Case2(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_BlockedPlayerCanMatchOnDifferentTeam_Case2", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "test:" + utils.GenerateUUID()
	matchmaker := NewMatchmaker()
	reqs := generateRequest(channelName, 3, 1)

	// set blocked - player[0] blocked player[1]
	reqs[0].PartyAttributes[models.AttributeBlocked] = []interface{}{"", reqs[1].PartyMembers[0].UserID}

	// set blocked - player[1] blocked player[2]
	reqs[1].PartyAttributes[models.AttributeBlocked] = []interface{}{"", reqs[2].PartyMembers[0].UserID}

	// get the ticket here before it's being ordered
	ticket0 := reqs[0]
	ticket1 := reqs[1]
	ticket2 := reqs[2]

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
		BlockedPlayerOption: models.BlockedPlayerCanMatchOnDifferentTeam,
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}

	results, _, err := matchmaker.MatchPlayers(scope, "", "", reqs, channel)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// team/ally combinations must match one of these:
	expectedTeamCombinations := [][]string{
		// player[1] should be alone and always alone
		{ticket1.PartyMembers[0].UserID},

		// OR player[0] and player[2] together
		{ticket0.PartyMembers[0].UserID, ticket2.PartyMembers[0].UserID},
		// OR player[0] alone
		{ticket0.PartyMembers[0].UserID},
		// OR player[2] alone
		{ticket2.PartyMembers[0].UserID},
	}

	for _, r := range results {
		for _, ally := range r.MatchingAllies {
			var match bool
			for _, expectedTeamCombination := range expectedTeamCombinations {
				if utils.HasSameElement(ally.GetMemberUserIDs(), expectedTeamCombination) {
					match = true
					break
				}
			}
			if !match {
				require.Equal(t, expectedTeamCombinations, ally.GetMemberUserIDs(), "team combinations should match one of the expectedTeamCombinations")
			}
		}
	}
}

// TestMatchmaker_BlockedPlayerCanMatch generate match requests with 2 players, 1 of them blocked the other, both player should able to be match in a same team
func TestMatchmaker_BlockedPlayerCanMatch(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_BlockedPlayerCanMatch", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "test:" + utils.GenerateUUID()
	matchmaker := NewMatchmaker()
	reqs := generateRequest(channelName, 2, 1)

	// set blocked
	reqs[0].PartyAttributes[models.AttributeBlocked] = []interface{}{"", reqs[1].PartyMembers[0].UserID}

	// ensure reqs[0] is oldest
	for i := range reqs {
		reqs[i].CreatedAt = time.Now().Add(time.Duration(i) * time.Millisecond).Unix()
	}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       1,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
		BlockedPlayerOption: models.BlockedPlayerCanMatch,
	}

	channel := models.Channel{
		Ruleset: *ruleset,
	}

	results, _, err := matchmaker.MatchPlayers(scope, "", "", reqs, channel)
	require.NoError(t, err)
	require.Len(t, results, 1)

	for _, r := range results {
		require.Len(t, r.GetMemberUserIDs(), 2)
		require.Len(t, r.MatchingAllies, 1)
		require.Len(t, r.MatchingAllies[0].GetMemberUserIDs(), 2)
	}
}

func TestMatchPlayers_FindMatchingAlly(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchPlayers_SinglePlayer", "")
	t.Cleanup(func() { scope.Finish() })

	cfg := &config.Config{}

	t.Run("matchingMinimalAllianceWithZeroMMRDiff", func(t *testing.T) {
		channel := "test"
		sourceTickets := generateRequestWithMMR(channel, 1, 1, 0)
		allianceRule := models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		}
		matchingRules := []models.MatchingRule{{
			Attribute: "mmr",
			Criteria:  distanceCriteria,
			Reference: 100,
		}}

		allies, _ := findMatchingAlly(scope, cfg, sourceTickets, sourceTickets[0], allianceRule, matchingRules, models.BlockedPlayerCannotMatch)
		// alliance MinNumber=1 PlayerMinNumber=1 supplied with 1 ticket should produce 1 alliance
		require.Lenf(t, allies, 1, " should produce 1 allies")
	})
}

func setRole(m *models.PartyMember, role ...string) {
	extraAttributes := m.ExtraAttributes
	if extraAttributes == nil {
		extraAttributes = make(map[string]interface{})
	}
	extraAttributes[models.ROLE] = role
	m.ExtraAttributes = extraAttributes
}

func TestMatchmaker_DebugOldestTicket(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_DebugOldestTicket", "")
	t.Cleanup(func() { scope.Finish() })

	repeat := 100

	for i := 0; i < repeat; i++ {
		channelName := "test:" + utils.GenerateUUID()
		ruleset := &models.RuleSet{
			AllianceRule: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 5,
				PlayerMaxNumber: 5,
			},
		}
		channel := models.Channel{
			Ruleset: *ruleset,
		}
		matchmaker := NewMatchmaker()

		reqs := make([]models.MatchmakingRequest, 0)
		numRequest := 71
		for i := 0; i < numRequest; i++ {
			t := time.Now()

			// exception for the first request only
			if i == 0 {
				t = time.Now().Add(-time.Duration(4) * time.Minute).UTC()
			}

			request := models.MatchmakingRequest{
				PartyID:      generateUlid(t),
				Channel:      channelName,
				CreatedAt:    t.Unix(),
				PartyMembers: []models.PartyMember{{UserID: utils.GenerateUUID()}},
			}
			reqs = append(reqs, request)
		}
		oldestTicket := reqs[0]
		lastTicket := reqs[len(reqs)-1]

		results, matchedRequests, err := matchmaker.MatchPlayers(scope, "", "", reqs, channel)

		require.NoError(t, err)
		require.Len(t, results, 7)
		require.Len(t, matchedRequests, 70)
		require.Contains(t, matchedRequests, oldestTicket)
		require.NotContains(t, matchedRequests, lastTicket)
	}
}

func TestMatchmaker_DebugOldestTicket_2(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_DebugOldestTicket_2", "")
	t.Cleanup(func() { scope.Finish() })

	repeat := 100

	for i := 0; i < repeat; i++ {
		channelName := "test:" + utils.GenerateUUID()
		ruleset := &models.RuleSet{
			AllianceRule: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 5,
				PlayerMaxNumber: 5,
			},
		}
		channel := models.Channel{
			Ruleset: *ruleset,
		}
		matchmaker := NewMatchmaker()

		reqs := make([]models.MatchmakingRequest, 0)
		numRequest := 71
		for i := 0; i < numRequest; i++ {
			priority := 0

			// exception for the last request only
			if i == numRequest-1 {
				priority = 1
			}

			t := time.Now()
			request := models.MatchmakingRequest{
				Priority:     priority,
				PartyID:      generateUlid(t),
				Channel:      channelName,
				CreatedAt:    t.Unix(),
				PartyMembers: []models.PartyMember{{UserID: utils.GenerateUUID()}},
			}
			reqs = append(reqs, request)
		}
		oldestTicket := reqs[numRequest-1]

		results, matchedRequests, err := matchmaker.MatchPlayers(scope, "", "", reqs, channel)

		require.NoError(t, err)
		require.Len(t, results, 7)
		require.Len(t, matchedRequests, 70)
		require.Contains(t, matchedRequests, oldestTicket)
	}
}

func TestMatchmaker_MatchOptionAnyAllCommon(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchmaker_MatchOptionAnyAllCommon", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "pveheist" //nolint:goconst
	matchmaker := NewMatchmakerWithConfigOverride(func(cfg *config.Config) {
		cfg.FlagAnyMatchOptionAllCommon = true
	})
	mmRequests := generateRequest(channelName, 5, 1)

	createdAt := time.Now().Add(-time.Duration(rand.Intn(100000)) * time.Millisecond).UTC().Unix()

	mmRequests[0].PartyAttributes = map[string]interface{}{
		"MapAssetNameTest": []interface{}{"BranchBank", "ArmoredTransport", "JewelryStore", "NightClub", "ArtGallery", "FirstPlayable", "CargoDock", "Penthouse", "Station", "Villa"},
	}
	mmRequests[1].PartyAttributes = map[string]interface{}{
		"MapAssetNameTest": []interface{}{"ArmoredTransport"},
	}
	mmRequests[2].PartyAttributes = map[string]interface{}{
		"MapAssetNameTest": []interface{}{"CargoDock"},
	}
	mmRequests[3].PartyAttributes = map[string]interface{}{
		"MapAssetNameTest": []interface{}{"BranchBank", "ArmoredTransport", "JewelryStore", "NightClub"},
	}
	mmRequests[4].PartyAttributes = map[string]interface{}{
		"MapAssetNameTest": []interface{}{"ArmoredTransport", "NightClub"},
	}

	for i := range mmRequests {
		mmRequests[i].PartyID = fmt.Sprintf("party%d", i)
		// make sure request-0 become pivot
		mmRequests[i].CreatedAt = createdAt + int64(i)
	}

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       1,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 4,
		},
		MatchOptions: models.MatchOptionRule{
			Options: []models.MatchOption{
				{Name: "MapAssetNameTest", Type: "any"},
			},
		},
	}
	channel := models.Channel{
		Ruleset: *ruleset,
	}

	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	require.NoError(t, err, "unable to execute matchmaking request")
	require.Len(t, results, 1)

	matchedPartyIDs := make([]string, 0)
	for _, result := range results {
		for _, ally := range result.MatchingAllies {
			for _, party := range ally.MatchingParties {
				matchedPartyIDs = append(matchedPartyIDs, party.PartyID)
			}
		}
	}
	require.Contains(t, matchedPartyIDs, mmRequests[0].PartyID)
	require.Contains(t, matchedPartyIDs, mmRequests[1].PartyID)
	require.Contains(t, matchedPartyIDs, mmRequests[3].PartyID)
	require.Contains(t, matchedPartyIDs, mmRequests[4].PartyID)
	require.NotContains(t, matchedPartyIDs, mmRequests[2].PartyID, "party should not matched because not shared common value with previous parties")
	require.Equal(t, []interface{}{"ArmoredTransport"}, results[0].PartyAttributes["MapAssetNameTest"], "should only store common value for all parties")
}

func Test_sortOldestFirst(t *testing.T) {
	type args struct {
		requests []models.MatchmakingRequest
	}
	tests := []struct {
		name string
		args args
		want []models.MatchmakingRequest
	}{
		{
			name: "test 1 - check priority DESC, then createdAt ASC",
			args: args{
				requests: []models.MatchmakingRequest{
					{PartyID: "A", Priority: 0, CreatedAt: timeNow.Unix()},
					{PartyID: "B", Priority: 0, CreatedAt: timeNow.Add(-1 * time.Second).Unix()},
					{PartyID: "C", Priority: 1, CreatedAt: timeNow.Add(-1 * time.Second).Unix()},
				},
			},
			want: []models.MatchmakingRequest{
				{PartyID: "C", Priority: 1, CreatedAt: timeNow.Add(-1 * time.Second).Unix()},
				{PartyID: "B", Priority: 0, CreatedAt: timeNow.Add(-1 * time.Second).Unix()},
				{PartyID: "A", Priority: 0, CreatedAt: timeNow.Unix()},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortOldestFirst(tt.args.requests)
			if got := tt.args.requests; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortMatchTickets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ensureSortedByAge(tickets []models.MatchmakingRequest) {
	now := time.Now().Unix()
	for i := range tickets {
		tickets[i].PartyID = fmt.Sprintf("party-%c", 'a'+i)
		tickets[i].CreatedAt = now + int64(i)
	}
}

func checkSessionBlockedPlayers(t *testing.T, matchingAllies []models.MatchingAlly) {
	t.Helper()
	blockMap := make(map[string]string)
	playerMap := make(map[string]string)
	for _, ally := range matchingAllies {
		for _, party := range ally.MatchingParties {
			for _, member := range party.PartyMembers {
				partyID, exists := blockMap[member.UserID]
				assert.Falsef(t, exists, "same session blocked member %v, blocked by party %v", member.UserID, partyID)
				playerMap[member.UserID] = party.PartyID
			}

			if blockList, ok := utils.GetMapValueAs[[]any](party.PartyAttributes, models.AttributeBlocked); ok {
				for _, v := range blockList {
					if userID, ok := v.(string); ok {
						_, exists := playerMap[userID]
						assert.Falsef(t, exists, "same session blocked member %v, blocked by party %v", userID, party.PartyID)
						blockMap[userID] = party.PartyID
					}
				}
			}
		}
	}
}

func TestMatchPlayers_BlockedPlayers(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchPlayers_BlockedPlayers", "")
	t.Cleanup(func() { scope.Finish() })

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 5,
			PlayerMaxNumber: 5,
		},
		AutoBackfill:                true,
		RegionExpansionRateMs:       5000,
		RegionExpansionRangeMs:      50,
		RegionLatencyInitialRangeMs: 100,
		RegionLatencyMaxMs:          300,
	}
	channel := models.Channel{
		Ruleset: *ruleset,
	}

	channelName := "test" //nolint:goconst
	matchmaker := NewMatchmaker()

	t.Run("ticketExcludedBecauseBlockedByPreviousTicket", func(t *testing.T) {
		mmRequests := generateRequest(channelName, 12, 1)
		createdAt := mmRequests[0].CreatedAt
		for i := range mmRequests {
			n := 'A' + i
			mmRequests[i].PartyID = fmt.Sprintf("party%c", n)
			mmRequests[i].PartyMembers[0].UserID = fmt.Sprintf("player%c", n)
			mmRequests[i].CreatedAt = createdAt + int64(i)
		}

		// random pivot excluding for A,B,C
		randPivot := rand.Intn(9) + 3
		mmRequests[randPivot].CreatedAt = createdAt - 100

		// player A block B and C
		mmRequests[0].PartyAttributes = map[string]interface{}{
			models.AttributeBlocked: []interface{}{
				mmRequests[1].PartyMembers[0].UserID,
				mmRequests[2].PartyMembers[0].UserID,
			},
		}
		// as player A checked before player B and C, player B and C will be excluded because player A block player B and C
		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

		assert.NoError(t, err, "unable to execute matchmaking request")
		require.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

		checkSessionBlockedPlayers(t, results[0].MatchingAllies)
	})

	t.Run("ticketExcludedBecauseBlockedByPreviousTicketPivot", func(t *testing.T) {
		mmRequests := generateRequest(channelName, 12, 1)
		createdAt := mmRequests[0].CreatedAt
		for i := range mmRequests {
			n := 'A' + i
			mmRequests[i].PartyID = fmt.Sprintf("party%c", n)
			mmRequests[i].PartyMembers[0].UserID = fmt.Sprintf("player%c", n)
			mmRequests[i].CreatedAt = createdAt + int64(i)
		}

		// A as pivot
		mmRequests[0].CreatedAt = createdAt - 100

		// player A block B and C
		mmRequests[0].PartyAttributes = map[string]interface{}{
			models.AttributeBlocked: []interface{}{
				mmRequests[1].PartyMembers[0].UserID,
				mmRequests[2].PartyMembers[0].UserID,
			},
		}
		// as player A checked before player B and C, player B and C will be excluded because player A block player B and C
		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

		assert.NoError(t, err, "unable to execute matchmaking request")
		require.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

		checkSessionBlockedPlayers(t, results[0].MatchingAllies)
	})

	t.Run("ticketExcludedBecauseBlockThePreviousTicket", func(t *testing.T) {
		mmRequests := generateRequest(channelName, 12, 1)
		createdAt := mmRequests[0].CreatedAt
		for i := range mmRequests {
			n := 'A' + i
			mmRequests[i].PartyID = fmt.Sprintf("party%c", n)
			mmRequests[i].PartyMembers[0].UserID = fmt.Sprintf("player%c", n)
			mmRequests[i].CreatedAt = createdAt + int64(i)
		}
		// swap order A and B
		mmRequests[0].CreatedAt, mmRequests[1].CreatedAt = mmRequests[1].CreatedAt, mmRequests[0].CreatedAt
		excludedPartyID := mmRequests[0].PartyID

		// random pivot excluding for A,B,C
		randPivot := rand.Intn(9) + 3
		mmRequests[randPivot].CreatedAt = createdAt - 100

		// player A block B and C
		mmRequests[0].PartyAttributes = map[string]interface{}{
			models.AttributeBlocked: []interface{}{
				mmRequests[1].PartyMembers[0].UserID,
				mmRequests[2].PartyMembers[0].UserID,
			},
		}
		// as player B checked before player A, player A will be excluded because block player B
		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

		assert.NoError(t, err, "unable to execute matchmaking request")
		require.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
		for _, allies := range results[0].MatchingAllies {
			for _, party := range allies.MatchingParties {
				assert.NotEqual(t, excludedPartyID, party.PartyID, "party A should be excluded")
			}
		}

		checkSessionBlockedPlayers(t, results[0].MatchingAllies)
	})

	t.Run("ticketExcludedBecauseBlockThePreviousTicketPivot", func(t *testing.T) {
		mmRequests := generateRequest(channelName, 12, 1)
		createdAt := mmRequests[0].CreatedAt
		for i := range mmRequests {
			n := 'A' + i
			mmRequests[i].PartyID = fmt.Sprintf("party%c", n)
			mmRequests[i].PartyMembers[0].UserID = fmt.Sprintf("player%c", n)
			mmRequests[i].CreatedAt = createdAt + int64(i)
		}
		// swap order A and B
		mmRequests[0].CreatedAt, mmRequests[1].CreatedAt = mmRequests[1].CreatedAt, mmRequests[0].CreatedAt
		excludedPartyID := mmRequests[0].PartyID

		// set B as pivot
		mmRequests[1].CreatedAt = createdAt - 100

		// player A block B and C
		mmRequests[0].PartyAttributes = map[string]interface{}{
			models.AttributeBlocked: []interface{}{
				mmRequests[1].PartyMembers[0].UserID,
				mmRequests[2].PartyMembers[0].UserID,
			},
		}
		// as player B checked before player A, player A will be excluded because block player B
		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

		assert.NoError(t, err, "unable to execute matchmaking request")
		require.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
		for _, allies := range results[0].MatchingAllies {
			for _, party := range allies.MatchingParties {
				assert.NotEqual(t, excludedPartyID, party.PartyID, "party A should be excluded")
			}
		}

		checkSessionBlockedPlayers(t, results[0].MatchingAllies)
	})

	t.Run("onlyCheckBlockForMatchedTickets", func(t *testing.T) {
		mmRequests := generateRequest(channelName, 12, 1)
		createdAt := mmRequests[0].CreatedAt
		for i := range mmRequests {
			n := 'A' + i
			mmRequests[i].PartyID = fmt.Sprintf("party%c", n)
			mmRequests[i].PartyMembers[0].UserID = fmt.Sprintf("player%c", n)
			mmRequests[i].CreatedAt = createdAt + int64(i)
			mmRequests[i].LatencyMap = map[string]int{"us-east-2": 100}
			mmRequests[i].SortedLatency = []models.Region{{Region: "us-east-2", Latency: 100}}
		}
		// swap order A and B
		mmRequests[0].CreatedAt, mmRequests[1].CreatedAt = mmRequests[1].CreatedAt, mmRequests[0].CreatedAt
		excludedPartyIDs := []string{mmRequests[1].PartyID, mmRequests[2].PartyID}

		// make sure B is not match with high latency
		mmRequests[1].LatencyMap = map[string]int{"us-east-1": 50, "us-east-2": 999}
		mmRequests[1].SortedLatency = []models.Region{{Region: "us-east-1", Latency: 50}, {Region: "us-east-2", Latency: 999}}

		// random pivot excluding for A,B,C
		randPivot := rand.Intn(9) + 3
		mmRequests[randPivot].CreatedAt = createdAt - 100

		// player A block B and C
		mmRequests[0].PartyAttributes = map[string]interface{}{
			models.AttributeBlocked: []interface{}{
				mmRequests[1].PartyMembers[0].UserID,
				mmRequests[2].PartyMembers[0].UserID,
			},
		}
		// player B will check block before player A, because B is not match A can match
		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

		assert.NoError(t, err, "unable to execute matchmaking request")
		require.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
		for _, allies := range results[0].MatchingAllies {
			for _, party := range allies.MatchingParties {
				assert.NotContains(t, excludedPartyIDs, party.PartyID, "%s should be excluded", party.PartyID)
			}
		}

		checkSessionBlockedPlayers(t, results[0].MatchingAllies)
	})
}

func TestAR_8138_RespectTeamComposition(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestAR_8138_RespectTeamComposition", "")
	t.Cleanup(func() { scope.Finish() })

	channelName := "spyduo" //nolint:goconst
	matchmaker := NewMatchmaker()
	ruleset := &models.RuleSet{}
	rulesetJson := `{"alliance":{"min_number":6,"max_number":6,"player_min_number":2,"player_max_number":2},"alliance_flexing_rule":[{"duration":30,"min_number":1,"max_number":6,"player_min_number":2,"player_max_number":2}],"matching_rule":[{"attribute":"crossplay-code","criteria":"distance","reference":4}],"max_delay_ms":100,"auto_backfill":false}`
	err := json.Unmarshal([]byte(rulesetJson), ruleset)
	require.NoError(t, err)
	channel := models.Channel{
		Ruleset: *ruleset,
	}

	now := time.Now().Add(-32 * time.Second)

	mmRequests := generateRequest(channelName, 4, 1)

	mmRequests[0].CreatedAt = now.Unix()
	mmRequests[0].PartyID = "1384ddeeb5124543a286dbeea59a4dd7"
	mmRequests[0].PartyAttributes = map[string]interface{}{
		"SUBREGION":         "worldwide",
		"client_version":    "1.10.01",
		"member_attributes": map[string]interface{}{},
	}
	mmRequests[0].PartyMembers = []models.PartyMember{
		{UserID: "0b06d63b29fb4923b75041e3d87b7391", ExtraAttributes: map[string]interface{}{"crossplay-code": 10}},
	}
	mmRequests[0].LatencyMap = map[string]int{"us-east-2": 40}
	mmRequests[0].SortedLatency = []models.Region{{Region: "us-east-2", Latency: 40}}

	mmRequests[1].CreatedAt = now.Unix() + 1
	mmRequests[1].PartyID = "3ebd4f36b8404212a91be55f8a793fda"
	mmRequests[1].PartyAttributes = map[string]interface{}{
		"SUBREGION":         "worldwide",
		"client_version":    "1.10.01",
		"member_attributes": map[string]interface{}{},
	}
	mmRequests[1].PartyMembers = []models.PartyMember{
		{UserID: "d36085428c9c435eb81006244780b8d8", ExtraAttributes: map[string]interface{}{"crossplay-code": 10}},
	}
	mmRequests[1].LatencyMap = map[string]int{"us-east-2": 39}
	mmRequests[1].SortedLatency = []models.Region{{Region: "us-east-2", Latency: 39}}

	mmRequests[2].CreatedAt = now.Unix() + 1
	mmRequests[2].PartyID = "d2aca65e32ff4f819441ee893cde0175"
	mmRequests[2].PartyAttributes = map[string]interface{}{
		"SUBREGION":         "worldwide",
		"client_version":    "1.10.01",
		"member_attributes": map[string]interface{}{},
	}
	mmRequests[2].PartyMembers = []models.PartyMember{
		{UserID: "a1cfae3565bf4220b582185695f5b622", ExtraAttributes: map[string]interface{}{"crossplay-code": 10}},
	}
	mmRequests[2].LatencyMap = map[string]int{"us-east-2": 37}
	mmRequests[2].SortedLatency = []models.Region{{Region: "us-east-2", Latency: 37}}

	mmRequests[3].CreatedAt = now.Unix()
	mmRequests[3].PartyID = "04c9d112a8c840a29d325ade7f1d42d3"
	mmRequests[3].PartyAttributes = map[string]interface{}{
		"SUBREGION":         "worldwide",
		"client_version":    "1.10.01",
		"member_attributes": map[string]interface{}{},
	}
	mmRequests[3].PartyMembers = []models.PartyMember{
		{UserID: "57a5c9d0b30e440291ff9909762f1b39", ExtraAttributes: map[string]interface{}{"crossplay-code": 10}},
	}
	mmRequests[3].LatencyMap = map[string]int{"us-east-2": 50}
	mmRequests[3].SortedLatency = []models.Region{{Region: "us-east-2", Latency: 50}}

	results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

	assert.NoError(t, err, "unable to execute matchmaking request")
	require.Truef(t, len(results) == 1, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))
	// flexed to minAllies=1 minPlayer=2, resulting 2v2
	require.Len(t, results[0].MatchingAllies, 2)
	require.Len(t, results[0].MatchingAllies[0].MatchingParties, 2)
	require.Len(t, results[0].MatchingAllies[1].MatchingParties, 2)
}

func getSortedLatencies(latencies map[string]int) []models.Region {
	sortedLatency := make([]models.Region, 0, len(latencies))
	for region, latency := range latencies {
		sortedLatency = append(sortedLatency, models.Region{Region: region, Latency: int(latency)})
	}
	sort.SliceStable(sortedLatency, func(i, j int) bool {
		return sortedLatency[i].Latency < sortedLatency[j].Latency
	})
	return sortedLatency
}

func TestMatchPlayers_WeightAttribute(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchPlayers_WeightAttribute", "")
	t.Cleanup(func() { scope.Finish() })

	ruleset := &models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 2,
		},
		MatchingRule: []models.MatchingRule{
			{Attribute: "mmr", Criteria: constants.DistanceCriteria, Reference: 100, Weight: swag.Float64(1.0)},
		},
		FlexingRule: []models.FlexingRule{
			{
				Duration:     10,
				MatchingRule: models.MatchingRule{Attribute: "mmr", Criteria: constants.DistanceCriteria, Reference: 200},
			},
			{
				Duration:     20,
				MatchingRule: models.MatchingRule{Attribute: "mmr", Criteria: constants.DistanceCriteria, Reference: 300},
			},
			{
				Duration:     30,
				MatchingRule: models.MatchingRule{Attribute: "mmr", Criteria: constants.DistanceCriteria, Reference: 400},
			},
			{
				Duration:     40,
				MatchingRule: models.MatchingRule{Attribute: "mmr", Criteria: constants.DistanceCriteria, Reference: 500},
			},
		},
		AutoBackfill:                true,
		RegionExpansionRateMs:       5000,
		RegionExpansionRangeMs:      50,
		RegionLatencyInitialRangeMs: 100,
		RegionLatencyMaxMs:          200,
	}
	ruleset.SetDefaultValues()
	channel := models.Channel{
		Ruleset: *ruleset,
	}

	type weightAttributes struct {
		mmr       float64
		latencies map[string]int
	}

	requestAttributes := []weightAttributes{
		// latencyDelta = abs(ticketLatency - pivotLatency)
		// normalizedLatencyScore = latencyDelta / maxLatency
		// to simplify the calculation we 0 latency for the pivot
		// us-west-2 region will be chosen since it the smallest latency on the pivot
		{mmr: 1000, latencies: map[string]int{"us-east-1": 10, "us-west-2": 0}},  // partyA -> pivot
		{mmr: 1500, latencies: map[string]int{"us-east-1": 5, "us-west-2": 200}}, // partyB
		{mmr: 1500, latencies: map[string]int{"us-east-1": 5, "us-west-2": 120}}, // partyC
		{mmr: 1500, latencies: map[string]int{"us-east-1": 5, "us-west-2": 40}},  // partyD
		{mmr: 1500, latencies: map[string]int{"us-east-1": 5, "us-west-2": 100}}, // partyE
		{mmr: 1300, latencies: map[string]int{"us-east-1": 5, "us-west-2": 200}}, // partyF
		{mmr: 1100, latencies: map[string]int{"us-east-1": 5, "us-west-2": 200}}, // partyG
		{mmr: 1250, latencies: map[string]int{"us-east-1": 5, "us-west-2": 200}}, // partyH
	}

	applyAttributes := func(mmRequests []models.MatchmakingRequest) {
		createdAt := time.Now().Unix() - 100
		for i := range mmRequests {
			n := 'A' + i
			mmRequests[i].PartyID = fmt.Sprintf("party%c", n)
			mmRequests[i].PartyMembers[0].UserID = fmt.Sprintf("player%c", n)
			mmRequests[i].CreatedAt = createdAt + int64(i)
			mmRequests[i].LatencyMap = requestAttributes[i].latencies
			mmRequests[i].SortedLatency = getSortedLatencies(requestAttributes[i].latencies)
			mmRequests[i].PartyMembers[0].ExtraAttributes = map[string]interface{}{"mmr": requestAttributes[i].mmr}
			mmRequests[i].PartyAttributes = map[string]interface{}{
				models.AttributeMemberAttr: map[string]interface{}{
					"mmr": requestAttributes[i].mmr,
				}}
		}
	}

	channelName := "test" //nolint:goconst
	matchmaker := NewMatchmaker()

	t.Run("weightOne", func(t *testing.T) {
		/*
			+---------+----------+-----------+----------+-------+--------+
			| mmrDist | latDelta | distScore | latScore | total | party  |
			|     500 |      200 |         1 |        1 |     2 | partyB |
			|     500 |      120 |         1 |      0.6 |   1.6 | partyC |
			|     500 |       40 |         1 |      0.2 |   1.2 | partyD |
			|     500 |      100 |         1 |      0.5 |   1.5 | partyE |
			|     300 |      200 |       0.6 |        1 |   1.6 | partyF |
			|     100 |      200 |       0.2 |        1 |   1.2 | partyG |
			|     250 |      200 |       0.5 |        1 |   1.5 | partyH |
			+---------+----------+-----------+----------+-------+--------+
		*/

		mmRequests := generateRequest(channelName, 8, 1)
		applyAttributes(mmRequests)
		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

		assert.NoError(t, err, "unable to execute matchmaking request")
		require.Truef(t, len(results) == 2, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

		expectedParties := []string{"partyA", "partyD", "partyE", "partyG"}
		var actualParties []string
		for _, allies := range results[0].MatchingAllies {
			for _, party := range allies.MatchingParties {
				actualParties = append(actualParties, party.PartyID)
			}
		}
		require.ElementsMatch(t, expectedParties, actualParties)
	})

	t.Run("prioritizeLatency", func(t *testing.T) {
		/*
			+---------+----------+-----------+----------+-------+--------+
			| mmrDist | latDelta | distScore | latScore | total | party  |
			+---------+----------+-----------+----------+-------+--------+
			|     500 |      200 |         1 |      0.5 |   1.5 | partyB |
			|     500 |      120 |         1 |      0.3 |   1.3 | partyC |
			|     500 |       40 |         1 |      0.1 |   1.1 | partyD |
			|     500 |      100 |         1 |     0.25 |  1.25 | partyE |
			|     300 |      200 |       0.6 |      0.5 |   1.1 | partyF |
			|     100 |      200 |       0.2 |      0.5 |   0.7 | partyG |
			|     250 |      200 |       0.5 |      0.5 |     1 | partyH |
			+---------+----------+-----------+----------+-------+--------+
		*/

		channel := models.Channel{
			Ruleset: *ruleset,
		}
		channel.Ruleset.RegionLatencyRuleWeight = swag.Float64(0.5)

		mmRequests := generateRequest(channelName, 8, 1)
		applyAttributes(mmRequests)
		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

		assert.NoError(t, err, "unable to execute matchmaking request")
		require.Truef(t, len(results) == 2, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

		expectedParties := []string{"partyA", "partyD", "partyG", "partyH"}
		var actualParties []string
		for _, allies := range results[0].MatchingAllies {
			for _, party := range allies.MatchingParties {
				actualParties = append(actualParties, party.PartyID)
			}
		}
		require.ElementsMatch(t, expectedParties, actualParties)
	})

	t.Run("prioritizeMMR", func(t *testing.T) {
		/*
			+---------+----------+-----------+----------+-------+--------+
			| mmrDist | latDelta | distScore | latScore | total | party  |
			+---------+----------+-----------+----------+-------+--------+
			|     500 |      200 |       0.5 |        1 |   1.5 | partyB |
			|     500 |      120 |       0.5 |      0.6 |   1.1 | partyC |
			|     500 |       40 |       0.5 |      0.2 |   0.7 | partyD |
			|     500 |      100 |       0.5 |      0.5 |     1 | partyE |
			|     300 |      200 |       0.3 |        1 |   1.3 | partyF |
			|     100 |      200 |       0.1 |        1 |   1.1 | partyG |
			|     250 |      200 |      0.25 |        1 |  1.25 | partyH |
			+---------+----------+-----------+----------+-------+--------+
		*/
		channel := models.Channel{
			Ruleset: *ruleset,
		}
		channel.Ruleset.MatchingRule[0].Weight = swag.Float64(0.5)

		mmRequests := generateRequest(channelName, 8, 1)
		applyAttributes(mmRequests)
		results, _, err := matchmaker.MatchPlayers(scope, "", "", mmRequests, channel)

		assert.NoError(t, err, "unable to execute matchmaking request")
		require.Truef(t, len(results) == 2, "unexpected matchmaking result count. expected: %d, actual: %d", 1, len(results))

		expectedParties := []string{"partyA", "partyD", "partyE", "partyC"}
		var actualParties []string
		for _, allies := range results[0].MatchingAllies {
			for _, party := range allies.MatchingParties {
				actualParties = append(actualParties, party.PartyID)
			}
		}
		require.ElementsMatch(t, expectedParties, actualParties)
	})
}
