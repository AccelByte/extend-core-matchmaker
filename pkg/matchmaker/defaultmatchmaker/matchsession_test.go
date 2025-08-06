// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/config"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateSession(channelName string, allianceCount int, playerCounts []int) *models.MatchmakingResult {
	alliances := make([]models.MatchingAlly, 0, allianceCount)
	t := time.Now()
	for i := 0; i < allianceCount; i++ {
		players := playerCounts[i]
		members := make([]models.PartyMember, 0, players)
		for j := 0; j < players; j++ {
			members = append(members, models.PartyMember{
				UserID:          utils.GenerateUUID(),
				ExtraAttributes: nil,
			})
		}

		alliances = append(alliances, models.MatchingAlly{
			MatchingParties: []models.MatchingParty{
				{
					PartyID:         generateUlid(t),
					PartyAttributes: nil,
					PartyMembers:    members,
				},
			},
		})
	}

	return &models.MatchmakingResult{
		MatchID:         utils.GenerateUUID(),
		MatchSessionID:  utils.GenerateUUID(),
		Channel:         channelName,
		Namespace:       "test",
		GameMode:        "test",
		Joinable:        true,
		MatchingAllies:  alliances,
		PartyAttributes: map[string]interface{}{},
	}
}

func containsTicket(session *models.MatchmakingResult, ticket *models.MatchmakingRequest) bool {
	for _, ally := range session.MatchingAllies {
		for _, party := range ally.MatchingParties {
			if party.PartyID != ticket.PartyID {
				continue
			}

			memberFound := 0
			for _, member := range party.PartyMembers {
				found := false
				for _, partyMember := range ticket.PartyMembers {
					if member.UserID == partyMember.UserID {
						found = true
						break
					}
				}

				if found {
					memberFound++
				}
			}

			if memberFound != len(ticket.PartyMembers) {
				continue
			}

			return true
		}
	}

	return false
}

func TestMatchSession_AddToAlly_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_AddToAlly_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	tickets := generateRequest("2v2", 1, 1)

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_AddToAlly_TooManyPlayers_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_AddToAlly_TooManyPlayers_Failed", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	tickets := generateRequest("2v2", 1, 2)

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	updatedSessions, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, updatedSessions, session, "updated session should not contain the session")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain the ticket")
}

func TestMatchSession_CreateNewAlly_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_CreateNewAlly_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("royale", 2, []int{1, 1})
	tickets := generateRequest("royale", 1, 1)

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       3,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
	assert.Equal(t, 3, len(session.MatchingAllies), "session should have added alliance")
}

func getPartyIDs(ally models.MatchingAlly) (partyIDs []string) {
	for _, party := range ally.MatchingParties {
		partyIDs = append(partyIDs, party.PartyID)
	}
	return partyIDs
}

func TestMatchSession_MultiTickets_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithClientVersion_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("4v4", 2, []int{2, 2})
	tickets := generateRequest("4v4", 2, 2)

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 4,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.Contains(t, matchedTickets, tickets[1], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[1]), "matched session should contain the ticket")
}

func TestMatchSession_WithClientVersion_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithClientVersion_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.ClientVersion = "test-version.1.0" //nolint:goconst

	tickets := generateRequest("2v2", 1, 1)
	tickets[0].PartyAttributes[models.AttributeClientVersion] = "test-version.1.0"

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_WithClientVersion_Mismatch_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithClientVersion_Mismatch_Failed", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.ClientVersion = "test-version.1.0"

	tickets := generateRequest("2v2", 1, 1)
	tickets[0].PartyAttributes[models.AttributeClientVersion] = "test-version.2.0"

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain the ticket")
}

func TestMatchSession_WithClientVersion_Blank_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithClientVersion_Blank_Failed", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.ClientVersion = "test-version.1.0"

	tickets := generateRequest("2v2", 1, 1)
	tickets[0].PartyAttributes[models.AttributeClientVersion] = ""

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain the ticket")
}

func TestMatchSession_WithServerName_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithClientVersion_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.ServerName = "test-version.1.0"

	tickets := generateRequest("2v2", 1, 1)
	tickets[0].PartyAttributes[models.AttributeServerName] = "test-version.1.0"

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_WithServerName_Mismatch_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithClientVersion_Mismatch_Failed", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.ServerName = "test-version.1.0"

	tickets := generateRequest("2v2", 1, 1)
	tickets[0].PartyAttributes[models.AttributeServerName] = "test-version.2.0"

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain the ticket")
}

func TestMatchSession_WithServerName_Blank_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithServerName_Blank_Failed", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 1, []int{2})
	session.ServerName = "v1.500"

	tickets := generateRequestWithMMR("2v2", 1, 1, 70)
	tickets[0].PartyAttributes[models.AttributeServerName] = ""

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	updatedSessions, satisfiedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, updatedSessions, session, "updated session should not contain the session")
	assert.NotContains(t, satisfiedSessions, session, "satisfied session should not contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain the ticket")
}

func TestMatchSession_WithRegion_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithRegion_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.Region = "us-west"

	tickets := generateRequest("2v2", 1, 1)
	tickets[0].PartyAttributes[models.AttributeLatencies] = `{ "us-west": 50 }`
	tickets[0].LatencyMap["us-west"] = 50
	tickets[0].SortedLatency = []models.Region{{Region: "us-west", Latency: 50}}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")

	// just to make sure that ticket source data is not changed
	assert.Equal(t, tickets[0].LatencyMap["us-west"], 50, "ticket LatencyMap should not change")
	assert.Equal(t, tickets[0].SortedLatency[0].Region, "us-west", "ticket SortedLatency map should not change")
}

func TestMatchSession_WithRegion_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithRegion_Failed", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.Region = "eu"

	tickets := generateRequest("2v2", 1, 1)
	tickets[0].PartyAttributes[models.AttributeLatencies] = `{ "us-west": 100 }`
	tickets[0].LatencyMap["us-west"] = 100
	tickets[0].SortedLatency = []models.Region{{Region: "us-west", Latency: 100}}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain the ticket")
}

func TestMatchSession_WithMultiRegion_2ndTry_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithMultiRegion_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.Region = "us-west"

	tickets := generateRequest("2v2", 1, 1)
	tickets[0].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "eu": 100, "us-west": 100 }`, models.AttributeMatchAttempt: float64(1)}
	tickets[0].LatencyMap = map[string]int{"eu": 100, "us-west": 100}
	tickets[0].SortedLatency = []models.Region{{Region: "eu", Latency: 100}, {Region: "us-west", Latency: 100}}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_WithAnyRegion_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithAnyRegion_Failed", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.Region = "ap-southeast-1"

	tickets := generateRequest("2v2", 1, 1)
	tickets[0].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "us-east-1": 100 }`, models.AttributeMatchAttempt: float64(1)}
	tickets[0].LatencyMap = map[string]int{"us-east-1": 100}
	tickets[0].SortedLatency = []models.Region{{Region: "us-east-1", Latency: 100}}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_WithMMR_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithMMR_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.Region = "eu"
	session.PartyAttributes[models.AttributeMemberAttr] = map[string]interface{}{"mmr": float64(80)}

	tickets := generateRequestWithMMR("2v2", 1, 1, 70)
	tickets[0].PartyAttributes[models.AttributeLatencies] = `{ "eu": 50 }` //nolint:goconst
	tickets[0].LatencyMap = map[string]int{"eu": 50}
	tickets[0].SortedLatency = []models.Region{
		{
			Region:  "eu",
			Latency: 50,
		},
	}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
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
				Reference: float64(10),
			},
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_WithFlexMMR_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithFlexMMR_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.Region = "eu"
	session.PartyAttributes[models.AttributeMemberAttr] = map[string]interface{}{"mmr": float64(100)}

	tickets := generateRequestWithMMR("2v2", 1, 1, 70)
	tickets[0].PartyAttributes[models.AttributeLatencies] = `{ "eu": 50 }`
	tickets[0].LatencyMap = map[string]int{"eu": 50}
	tickets[0].SortedLatency = []models.Region{{Region: "eu", Latency: 50}}
	tickets[0].CreatedAt = time.Now().UTC().Unix() - int64(5*time.Second)

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
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
				Reference: float64(10),
			},
		},
		FlexingRule: []models.FlexingRule{
			{
				Duration: 5,
				MatchingRule: models.MatchingRule{
					Attribute: "mmr",
					Criteria:  "distance",
					Reference: float64(30),
				},
			},
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_WithFlexMMR_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithFlexMMR_Failed", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.Region = "eu"
	session.PartyAttributes[models.AttributeMemberAttr] = map[string]interface{}{"mmr": float64(100)}

	tickets := generateRequestWithMMR("2v2", 1, 1, 70)
	tickets[0].PartyAttributes[models.AttributeLatencies] = `{ "eu": 50 }`
	tickets[0].LatencyMap = map[string]int{"eu": 50}
	tickets[0].SortedLatency = []models.Region{{Region: "eu", Latency: 50}}
	tickets[0].CreatedAt = time.Now().UTC().Add(-6 * time.Second).Unix()

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
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
				Reference: float64(10),
			},
		},
		FlexingRule: []models.FlexingRule{
			{
				Duration: int64(5),
				MatchingRule: models.MatchingRule{
					Attribute: "mmr",
					Criteria:  "distance",
					Reference: float64(20),
				},
			},
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain the ticket")
}

func TestMatchSession_WithCustomAttribute_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithCustomAttribute_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.PartyAttributes["map"] = "area.1-10" //nolint:goconst

	tickets := generateRequestWithMMR("2v2", 1, 1, 70)
	tickets[0].PartyAttributes["map"] = "area.1-10"

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_WithCustomAttribute_Mismatch_Failed(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithCustomAttribute_Mismatch_Failed", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.PartyAttributes["map"] = "area.2-10"

	tickets := generateRequestWithMMR("2v2", 1, 1, 70)
	tickets[0].PartyAttributes["map"] = "area.1-10"

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain the ticket")
}

func TestMatchSession_UpdatedSessions_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_UpdatedSessions_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("4v4", 2, []int{2, 2})
	tickets := generateRequest("4v4", 1, 2)

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 4,
		},
	}

	updatedSessions, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, updatedSessions, session, "updated session should contain the session")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_UpdatedSessions2_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_UpdatedSessions2_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("3v3v3", 1, []int{1})
	tickets := generateRequest("3v3v3", 1, 1)

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       3,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 3,
		},
	}

	updatedSessions, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, updatedSessions, session, "updated session should contain the session")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_Blocked_ByTicket(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_Blocked_ByTicket", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	tickets := generateRequest("2v2", 1, 1)

	tickets[0].PartyAttributes[models.AttributeBlocked] = []interface{}{session.MatchingAllies[0].MatchingParties[0].PartyMembers[0].UserID}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_Blocked_BySession(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_Blocked_BySession", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	tickets := generateRequest("2v2", 1, 1)

	session.PartyAttributes[models.AttributeBlocked] = []interface{}{tickets[0].PartyMembers[0].UserID}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_MatchOptions(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_MatchOptions", "")
	t.Cleanup(func() { scope.Finish() })

	type args struct {
		tickets  []models.MatchmakingRequest
		sessions []*models.MatchmakingResult
		channel  models.Channel
	}
	type testItem struct {
		name                  string
		args                  args
		wantUpdatedSessions   []*models.MatchmakingResult
		wantSatisfiedSessions []*models.MatchmakingResult
		// wantSatisfiedTickets  []*models.MatchmakingRequest
		wantErr bool
	}
	tests := []testItem{}

	// case 1
	{
		tickets := generateRequest("", 1, 1)
		tickets[0].PartyAttributes["platform"] = []interface{}{"xbox", "pc"}

		session := generateSession("", 1, []int{1})
		session.PartyAttributes["platform"] = []interface{}{"xbox"}

		session2 := generateSession("", 1, []int{1})
		session2.PartyAttributes["platform"] = []interface{}{"ps4"}

		tests = append(tests, testItem{
			name: "should find xbox",
			args: args{
				tickets:  tickets,
				sessions: []*models.MatchmakingResult{session2, session},
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							MinNumber:       1,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 1,
						},
						MatchOptions: models.MatchOptionRule{
							Options: []models.MatchOption{
								{
									Name: "platform",
									Type: models.MatchOptionTypeAny,
								},
							},
						},
					},
				},
			},
			wantSatisfiedSessions: []*models.MatchmakingResult{
				session,
			},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mm := NewMatchmaker()
			updated, satisfied, _, err := mm.MatchSessions(scope, "", "", tt.args.tickets, tt.args.sessions, tt.args.channel)
			if (err != nil) != tt.wantErr {
				t.Errorf("Matchmaker.MatchSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, len(tt.wantUpdatedSessions), len(updated), "unexpected updated session count. testname: %s, expected: %d, actual: %d", tt.name, len(tt.wantUpdatedSessions), len(updated))
			assert.Equal(t, len(tt.wantSatisfiedSessions), len(satisfied), "unexpected satisfied session count. testname: %s, expected: %d, actual: %d", tt.name, len(tt.wantSatisfiedSessions), len(satisfied))

			for _, v := range tt.wantUpdatedSessions {
				assert.Contains(t, updated, v)
			}

			for _, v := range tt.wantSatisfiedSessions {
				assert.Contains(t, satisfied, v)
			}
		})
	}
}

func TestMatchSession_AvoidMatchWithSelf(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_AvoidMatchWithSelf", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})

	tickets := generateRequest("2v2", 1, 1)
	tickets[0].PartyMembers[0].UserID = session.MatchingAllies[0].MatchingParties[0].PartyMembers[0].UserID

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain the ticket")
}

func TestMatchSession_MatchBasedOnRegion(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_MatchBasedOnRegion", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.Region = "us-west-1"

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	// not match based on region
	tickets := generateRequest("2v2", 1, 1)
	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.NotContains(t, matchedSessions, session, "matched session should not contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain the ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain the ticket")

	// match based on region
	tickets = generateRequest("2v2", 2, 1)
	tickets[0].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "us-west-1": 100 }`}
	tickets[0].LatencyMap = map[string]int{"us-west-1": 100}
	tickets[0].SortedLatency = []models.Region{{Region: "us-west-1", Latency: 100}}
	tickets[1].PartyAttributes = map[string]interface{}{models.AttributeLatencies: `{ "us-west-1": 10 }`}
	tickets[1].LatencyMap = map[string]int{"us-west-1": 10}
	tickets[1].SortedLatency = []models.Region{{Region: "us-west-1", Latency: 10}}
	_, matchedSessions, matchedTickets, err = matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain first ticket because the latency is higher than second ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain first ticket because the latency is higher than second ticket")
	assert.Contains(t, matchedTickets, tickets[1], "matched tickets should contain the second ticket")
	assert.True(t, containsTicket(session, &tickets[1]), "matched session should contain the second ticket")
	assert.Equal(t, matchedTickets[0].LatencyMap["us-west-1"], 10, "matched ticket should contain the second ticket with latency 10")
}

func TestMatchSession_MatchBasedOnRegion_RegionRate(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_MatchBasedOnRegion_RegionRate", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.Region = "us-west-1"

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		RegionExpansionRateMs: 1000,
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	channel := models.Channel{Ruleset: ruleset}

	Now = func() time.Time { return time.Date(2023, 1, 1, 1, 0, 0, 0, time.UTC) }
	defer func() { Now = time.Now }()

	// match based on region
	tickets := generateRequest("2v2", 2, 1)
	tickets[0].CreatedAt = Now().Unix()
	tickets[0].LatencyMap = map[string]int{"us-west-1": 90, "us-east-1": 80}
	tickets[0].SortedLatency = []models.Region{{Region: "us-east-1", Latency: 80}, {Region: "us-west-1", Latency: 90}}
	tickets[1].CreatedAt = Now().Add(-time.Second * 2).Unix() // should match with this one
	tickets[1].LatencyMap = map[string]int{"us-west-1": 99, "us-east-1": 100}
	tickets[1].SortedLatency = []models.Region{{Region: "us-east-1", Latency: 90}, {Region: "us-west-1", Latency: 99}}
	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, channel)
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.NotContains(t, matchedTickets, tickets[0], "matched tickets should not contain first ticket because first ticket is newer")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should not contain first ticket because first ticket is newer")
	assert.Contains(t, matchedTickets, tickets[1], "matched tickets should contain the second ticket")
	assert.True(t, containsTicket(session, &tickets[1]), "matched session should contain the second ticket")
	assert.Equal(t, matchedTickets[0].LatencyMap["us-west-1"], 99, "matched ticket should contain the second ticket with latency 99")
}

func TestMatchSession_JoinTicket_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_JoinTicket_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("test-pool", 1, []int{1})
	tickets := generateRequestNoMMR("test-pool", 1, 1)

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       4,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 4,
		},
	}

	updatedSessions, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, updatedSessions, session, "matched session should contain the session")
	assert.NotContains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[0]), "matched session should contain the ticket")
}

func TestMatchSession_JoinTicket_SuccessWithUniqueRuleset(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_JoinTicket_SuccessWithUniqueRuleset", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	uniqueRuleSet := utils.GenerateUUID() // "pokedex"
	uniqueRuleSet1 := utils.GenerateUUID()
	uniqueRuleSet2 := utils.GenerateUUID()
	session := generateSession("test-pool", 1, []int{1})
	session.PartyAttributes = map[string]interface{}{
		uniqueRuleSet: uniqueRuleSet1, // "pikachu",
	}

	tickets := generateRequestNoMMR("test-pool", 2, 1)
	tickets[0].PartyAttributes = map[string]interface{}{
		uniqueRuleSet: uniqueRuleSet2, //"charizard",
	}
	tickets[1].PartyAttributes = map[string]interface{}{
		uniqueRuleSet: uniqueRuleSet2, //"charizard",
	}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	enableRebalance := false

	ruleset := models.RuleSet{
		AutoBackfill:    true,
		RebalanceEnable: &enableRebalance,
		MatchOptions: models.MatchOptionRule{
			Options: []models.MatchOption{
				{
					Name: uniqueRuleSet,
					Type: "unique",
				},
			},
		},

		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       4,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 4,
		},
	}

	updatedSessions, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	for _, data := range updatedSessions {
		assert.ElementsMatch(t, []interface{}{uniqueRuleSet1, uniqueRuleSet2}, data.PartyAttributes[uniqueRuleSet])
	}

	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, updatedSessions, session, "matched session should contain the session")
	assert.NotContains(t, matchedSessions, session, "matched session should contain the session")
	ticketIndex := -1
	for i := range matchedTickets {
		for j := range tickets {
			if tickets[j].PartyID == matchedTickets[i].PartyID {
				ticketIndex = j
				break
			}
		}
	}
	assert.Contains(t, matchedTickets, tickets[ticketIndex], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[ticketIndex]), "matched session should contain the ticket")
}

func TestMatchSession_JoinTicket_SuccessWithUniqueRulesetWith3Input(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_JoinTicket_SuccessWithUniqueRuleset", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	uniqueRuleSet := utils.GenerateUUID() //"pokedex"
	uniqueRuleSet1 := "uniqueRuleSet1"
	uniqueRuleSet2 := "uniqueRuleSet2"
	uniqueRuleSet3 := "uniqueRuleSet3"
	session := generateSession("test-pool", 1, []int{1})
	session.PartyAttributes = map[string]interface{}{
		uniqueRuleSet: uniqueRuleSet1, // "pikachu",
	}

	tickets := generateRequestNoMMR("test-pool", 3, 1)
	createdAt := tickets[0].CreatedAt
	tickets[0].PartyAttributes = map[string]interface{}{
		uniqueRuleSet: uniqueRuleSet2, //"charizard",
	}
	tickets[1].PartyAttributes = map[string]interface{}{
		uniqueRuleSet: uniqueRuleSet2, // "charizard",
	}
	tickets[1].CreatedAt = createdAt + 1
	tickets[2].PartyAttributes = map[string]interface{}{
		uniqueRuleSet: uniqueRuleSet3, //"eevee",
	}
	tickets[2].CreatedAt = createdAt + 2

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	enablerRebalance := false

	ruleset := models.RuleSet{
		AutoBackfill:    true,
		RebalanceEnable: &enablerRebalance,
		MatchOptions: models.MatchOptionRule{
			Options: []models.MatchOption{
				{
					Name: uniqueRuleSet,
					Type: "unique",
				},
			},
		},

		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       4,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 4,
		},
	}

	updatedSessions, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	for _, data := range updatedSessions {
		assert.ElementsMatch(t, []interface{}{uniqueRuleSet1, uniqueRuleSet2, uniqueRuleSet3}, data.PartyAttributes[uniqueRuleSet])
	}

	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, updatedSessions, session, "matched session should contain the session")
	assert.NotContains(t, matchedSessions, session, "matched session should contain the session")
	ticketIndex := -1
	for i := range matchedTickets {
		for j := range tickets {
			if tickets[j].PartyID == matchedTickets[i].PartyID {
				ticketIndex = j
				break
			}
		}
	}
	assert.Contains(t, matchedTickets, tickets[ticketIndex], "matched tickets should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[ticketIndex]), "matched session should contain the ticket")
	assert.True(t, containsTicket(session, &tickets[2]), "matched session should contain the ticket")
}

// TestMatchSession_BlockedPlayerCannotMatch has a session of 1 player (a), generate match requests with 1 player (b), b block a, b cannot join the session
func TestMatchSession_BlockedPlayerCannotMatch(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_BlockedPlayerCannotMatch", "")
	defer scope.Finish()

	channelName := "test:" + utils.GenerateUUID()
	matchmaker := NewMatchmaker()
	session := generateSession(channelName, 1, []int{1})
	tickets := generateRequest(channelName, 1, 1)

	// set blocked
	tickets[0].PartyAttributes[models.AttributeBlocked] = []interface{}{"", session.MatchingAllies[0].MatchingParties[0].PartyMembers[0].UserID}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       1,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	updatedSessions, fullSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	require.NoError(t, err)
	require.NotContains(t, updatedSessions, session)
	require.NotContains(t, fullSessions, session)
	require.NotContains(t, matchedTickets, tickets[0])
	require.False(t, containsTicket(session, &tickets[0]))

	require.Len(t, updatedSessions, 0)
	require.Len(t, session.GetMemberUserIDs(), 1)
}

// TestMatchSession_BlockedPlayerCanMatchOnDifferentTeam has a session of 1 player (a), generate match requests with 1 player (b), b block a, both player should able to be match but in different team
func TestMatchSession_BlockedPlayerCanMatchOnDifferentTeam(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_BlockedPlayerCanMatchOnDifferentTeam", "")
	defer scope.Finish()

	channelName := "test:" + utils.GenerateUUID()
	matchmaker := NewMatchmaker()
	session := generateSession(channelName, 1, []int{1})
	tickets := generateRequest(channelName, 1, 1)

	// set blocked
	tickets[0].PartyAttributes[models.AttributeBlocked] = []interface{}{"", session.MatchingAllies[0].MatchingParties[0].PartyMembers[0].UserID}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},

		BlockedPlayerOption: models.BlockedPlayerCanMatchOnDifferentTeam,
	}

	updatedSessions, fullSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	require.NoError(t, err)
	require.Contains(t, updatedSessions, session)
	require.NotContains(t, fullSessions, session)
	require.Contains(t, matchedTickets, tickets[0])
	require.True(t, containsTicket(session, &tickets[0]))

	require.Len(t, updatedSessions, 1)
	require.Len(t, updatedSessions[0].MatchingAllies, 2)
	require.Len(t, updatedSessions[0].MatchingAllies[0].GetMemberUserIDs(), 1)
	require.Len(t, updatedSessions[0].MatchingAllies[1].GetMemberUserIDs(), 1)
}

// TestMatchSession_BlockedPlayerCanMatchOnDifferentTeam_Case2 has a session of 1 player (a), generate match requests with 2 players (b,c), b block c, c block a. They can match but respect block
func TestMatchSession_BlockedPlayerCanMatchOnDifferentTeam_Case2(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_BlockedPlayerCanMatchOnDifferentTeam_Case2", "")
	defer scope.Finish()

	channelName := "test:" + utils.GenerateUUID()
	matchmaker := NewMatchmaker()
	session := generateSession(channelName, 1, []int{1})
	tickets := generateRequest(channelName, 2, 1)

	// set blocked
	tickets[0].PartyAttributes[models.AttributeBlocked] = []interface{}{"", tickets[1].PartyMembers[0].UserID}
	tickets[1].PartyAttributes[models.AttributeBlocked] = []interface{}{"", session.MatchingAllies[0].MatchingParties[0].PartyMembers[0].UserID}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},

		BlockedPlayerOption: models.BlockedPlayerCanMatchOnDifferentTeam,
	}

	updatedSessions, fullSessions, _, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	require.NoError(t, err)
	require.Contains(t, updatedSessions, session)
	require.NotContains(t, fullSessions, session)

	// team/ally combinations must match one of these:
	expectedTeamCombinations := [][]string{
		// player[1] should be alone and always alone
		{tickets[1].PartyMembers[0].UserID},

		// OR player[0] and session's player together
		{tickets[0].PartyMembers[0].UserID, session.MatchingAllies[0].MatchingParties[0].PartyMembers[0].UserID},
		// OR player[0] alone
		{tickets[0].PartyMembers[0].UserID},
		// OR session's player alone
		{session.MatchingAllies[0].MatchingParties[0].PartyMembers[0].UserID},
	}

	for _, updatedSession := range updatedSessions {
		for _, ally := range updatedSession.MatchingAllies {
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

// TestMatchSession_BlockedPlayerCanMatchOnDifferentTeam_Case3 has a session of 1 player (a), generate match requests with 1 player (b), b block a, b cannot join session because ruleset only have 1 ally
func TestMatchSession_BlockedPlayerCanMatchOnDifferentTeam_Case3(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_BlockedPlayerCanMatchOnDifferentTeam_Case3", "")
	defer scope.Finish()

	channelName := "test:" + utils.GenerateUUID()
	matchmaker := NewMatchmaker()
	session := generateSession(channelName, 1, []int{1})
	tickets := generateRequest(channelName, 1, 1)

	// set blocked
	tickets[0].PartyAttributes[models.AttributeBlocked] = []interface{}{"", session.MatchingAllies[0].MatchingParties[0].PartyMembers[0].UserID}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       1,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},

		BlockedPlayerOption: models.BlockedPlayerCanMatchOnDifferentTeam,
	}

	updatedSessions, fullSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	require.NoError(t, err)
	require.NotContains(t, updatedSessions, session)
	require.NotContains(t, fullSessions, session)
	require.NotContains(t, matchedTickets, tickets[0])
	require.False(t, containsTicket(session, &tickets[0]))

	require.Len(t, updatedSessions, 0)
	require.Len(t, session.GetMemberUserIDs(), 1)
}

// TestMatchSession_BlockedPlayerCanMatch has a session of 1 player (a), generate match requests with 1 player (b), b block a, both player should able to be match in the same team
func TestMatchSession_BlockedPlayerCanMatch(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_BlockedPlayerCanMatch", "")
	defer scope.Finish()

	channelName := "test:" + utils.GenerateUUID()
	matchmaker := NewMatchmaker()
	session := generateSession(channelName, 1, []int{1})
	tickets := generateRequest(channelName, 1, 1)

	// set blocked
	tickets[0].PartyAttributes[models.AttributeBlocked] = []interface{}{"", session.MatchingAllies[0].MatchingParties[0].PartyMembers[0].UserID}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       1,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 2,
		},

		BlockedPlayerOption: models.BlockedPlayerCanMatch,
	}

	updatedSessions, fullSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	require.NoError(t, err)
	require.NotContains(t, updatedSessions, session)
	require.Contains(t, fullSessions, session)
	require.Contains(t, matchedTickets, tickets[0])
	require.True(t, containsTicket(session, &tickets[0]))

	require.Len(t, fullSessions, 1)
	require.Len(t, fullSessions[0].GetMemberUserIDs(), 2)
	require.Len(t, fullSessions[0].MatchingAllies[0].GetMemberUserIDs(), 2)
}

func TestMatchSession_DuplicateUserID(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_DuplicateUserID", "")
	defer scope.Finish()

	userIDA := "userA"

	channelName := "test:" + utils.GenerateUUID()
	matchmaker := NewMatchmaker()
	session := generateSession(channelName, 2, []int{3, 1})
	tickets := generateRequest(channelName, 2, 1)
	tickets[0].PartyMembers[0].UserID = userIDA
	tickets[1].PartyMembers[0].UserID = userIDA

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 5,
		},

		AutoBackfill:                    true,
		RebalanceEnable:                 models.FALSE(),
		RegionLatencyInitialRangeMs:     25,
		RegionExpansionRateMs:           10000,
		RegionExpansionRangeMs:          50,
		RegionLatencyMaxMs:              500,
		MatchOptionsReferredForBackfill: false,
	}

	updatedSessions, _, _, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	require.NoError(t, err)
	require.Contains(t, updatedSessions, session)

	require.Len(t, updatedSessions, 1)
	require.Len(t, updatedSessions[0].GetMemberUserIDs(), 5)
	require.Len(t, updatedSessions[0].MatchingAllies, 2)
	require.Len(t, updatedSessions[0].MatchingAllies[0].GetMemberUserIDs(), 3)
	require.Len(t, updatedSessions[0].MatchingAllies[1].GetMemberUserIDs(), 2)

	var countUserA int
	for _, ally := range updatedSessions[0].MatchingAllies {
		for _, party := range ally.MatchingParties {
			for _, member := range party.PartyMembers {
				if member.UserID == "userA" {
					countUserA++
				}
			}
		}
	}
	require.Equal(t, countUserA, 1)
}

func TestMatchSession_MatchOptionAnyAllCommon(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_AddToAlly_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmakerWithConfigOverride(func(cfg *config.Config) {
		cfg.FlagAnyMatchOptionAllCommon = true
	})

	session := generateSession("pveheist", 1, []int{2})
	tickets := generateRequest("pveheist", 3, 1)

	session.PartyAttributes = map[string]interface{}{
		"MapAssetNameTest": []interface{}{"ArmoredTransport", "JewelryStore"},
	}

	tickets[0].PartyAttributes = map[string]interface{}{
		"MapAssetNameTest": []interface{}{"BranchBank", "ArmoredTransport", "JewelryStore", "CargoDock"},
	}
	tickets[1].PartyAttributes = map[string]interface{}{
		"MapAssetNameTest": []interface{}{"CargoDock"},
	}
	tickets[2].PartyAttributes = map[string]interface{}{
		"MapAssetNameTest": []interface{}{"ArmoredTransport"},
	}

	for i := range tickets {
		tickets[i].PartyID = fmt.Sprintf("party%d", i)
	}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
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

	_, satisfiedSession, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, matchedTickets, tickets[0], "matched tickets should contain the ticket")
	assert.Contains(t, matchedTickets, tickets[2], "matched tickets should contain the ticket")
	assert.NotContains(t, matchedTickets, tickets[1], "matched tickets should contain the ticket")
	require.Contains(t, satisfiedSession, session, "matched session should contain the session")
	assert.Equal(t, []interface{}{"ArmoredTransport"}, satisfiedSession[0].PartyAttributes["MapAssetNameTest"])
}

func TestMatchSession_WithExcludedSessions_Success(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithExcludedSessions_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.MatchSessionID = utils.GenerateUUID()
	tickets := generateRequest("2v2", 2, 1)
	assert.Len(t, tickets, 2, "should generate 2 tickets")

	// first ticket excludes the only available session
	excludedSessionID := session.MatchSessionID
	tickets[0].ExcludedSessions = []string{excludedSessionID}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, matchedTickets, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, tickets[0].ExcludedSessions, session.MatchSessionID, "ticket should contain the session")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
	assert.Contains(t, matchedTickets, tickets[1], "matched tickets should contain the 2nd ticket")
	assert.False(t, containsTicket(session, &tickets[0]), "matched session should contain the 1st ticket")
}

func TestMatchSession_WithEmptyExcludedSessions_Success(t *testing.T) {
	t.Parallel()
	scope := envelope.NewRootScope(context.Background(), "TestMatchSession_WithExcludedSessions_Success", "")
	defer scope.Finish()

	matchmaker := NewMatchmaker()

	session := generateSession("2v2", 2, []int{2, 1})
	session.MatchSessionID = ""
	tickets := generateRequest("2v2", 2, 1)
	assert.Len(t, tickets, 2, "should generate 2 tickets")

	// first ticket excludes the only available session
	excludedSessionID := session.MatchSessionID
	tickets[0].ExcludedSessions = []string{excludedSessionID}

	var sessions []*models.MatchmakingResult
	sessions = append(sessions, session)

	ruleset := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 2,
		},
	}

	_, matchedSessions, _, err := matchmaker.MatchSessions(scope, "", "", tickets, sessions, models.Channel{Ruleset: ruleset})
	assert.NoError(t, err, "finding session should have no error")
	assert.Contains(t, tickets[0].ExcludedSessions, session.MatchSessionID, "ticket should contain the session")
	assert.Contains(t, matchedSessions, session, "matched session should contain the session")
}
