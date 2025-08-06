// Copyright (c) 2022 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package basic provides sample test data for basic matchmaker functionality.
// This package contains predefined test scenarios and expected results for testing the matchmaker.
package basic

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker"
	player "github.com/AccelByte/extend-core-matchmaker/pkg/playerdata"
	"github.com/elliotchance/pie/v2"
)

// SampleFiveSinglePlayerTickets contains five sample tickets for single-player matchmaking tests.
// Each ticket represents a single player with MMR attribute set to 25.
var SampleFiveSinglePlayerTickets = []matchmaker.Ticket{
	{CreatedAt: time.Now().Add(-5 * time.Millisecond), TicketID: "first", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user1", Attributes: map[string]interface{}{"mmr": float64(25)}}}},
	{CreatedAt: time.Now().Add(-4 * time.Millisecond), TicketID: "second", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user2", Attributes: map[string]interface{}{"mmr": float64(25)}}}},
	{CreatedAt: time.Now().Add(-3 * time.Millisecond), TicketID: "third", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user3", Attributes: map[string]interface{}{"mmr": float64(25)}}}},
	{CreatedAt: time.Now().Add(-2 * time.Millisecond), TicketID: "fourth", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user4", Attributes: map[string]interface{}{"mmr": float64(25)}}}},
	{CreatedAt: time.Now().Add(-1 * time.Millisecond), TicketID: "fifth", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user5", Attributes: map[string]interface{}{"mmr": float64(25)}}}},
}

// PlayerDataToUserID is a helper function that extracts the player ID from PlayerData.
// This is used for converting player data to user IDs in test scenarios.
func PlayerDataToUserID(p player.PlayerData) player.ID {
	return p.PlayerID
}

// BasicExpectedFiveSinglePlayerTicketsMatchResults contains the expected match results
// when processing the five single-player tickets. It shows how the matchmaker should
// group players into teams of two, with the fifth player remaining unmatched.
var BasicExpectedFiveSinglePlayerTicketsMatchResults = []matchmaker.Match{
	{
		Tickets:          []matchmaker.Ticket{SampleFiveSinglePlayerTickets[0], SampleFiveSinglePlayerTickets[1]},
		Teams:            []matchmaker.Team{{UserIDs: pie.Map(append(SampleFiveSinglePlayerTickets[0].Players, SampleFiveSinglePlayerTickets[1].Players...), PlayerDataToUserID)}},
		RegionPreference: []string{"any"},
	},
	{
		Tickets:          []matchmaker.Ticket{SampleFiveSinglePlayerTickets[2], SampleFiveSinglePlayerTickets[3]},
		Teams:            []matchmaker.Team{{UserIDs: pie.Map(append(SampleFiveSinglePlayerTickets[2].Players, SampleFiveSinglePlayerTickets[3].Players...), PlayerDataToUserID)}},
		RegionPreference: []string{"any"},
	},
}
