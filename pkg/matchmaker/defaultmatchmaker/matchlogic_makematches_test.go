// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"fmt"
	_ "net/http/pprof"
	"strconv"
	"testing"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/config"
	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker"
	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker/defaultmatchmaker/basic"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	player "github.com/AccelByte/extend-core-matchmaker/pkg/playerdata"
	"github.com/AccelByte/extend-core-matchmaker/pkg/testsetup"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
	"github.com/elliotchance/pie/v2"
	. "github.com/onsi/gomega"
)

func TestDefaultMatchMaker_ClosesChannelWhenWrongTypeOfRulesPassed(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	matches := mm.MakeMatches(testsetup.NewTestScope(), testsetup.StubMatchTicketProvider{}, struct{}{})

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	g.Expect(len(results)).To(Equal(0))
}

func TestDefaultMatchMaker_ClosesChannelWhenNoMoreMatchesFound(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	matches := mm.MakeMatches(testsetup.NewTestScope(), testsetup.StubMatchTicketProvider{}, get1v1Rules())

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	g.Expect(len(results)).To(Equal(0))
}

func TestDefaultMatchMaker_MatchesTicketsIn1v1(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: basic.SampleFiveSinglePlayerTickets,
	}
	matches := mm.MakeMatches(testsetup.NewTestScope(), ticketProvider, get1v1Rules())

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	g.Expect(len(results)).To(Equal(2))
	g.Expect(results[0].Tickets).To(Equal(ExpectedFiveSinglePlayerTicketsMatchResults[0].Tickets))
	g.Expect(results[1].Tickets).To(Equal(ExpectedFiveSinglePlayerTicketsMatchResults[1].Tickets))
	for _, result := range results {
		g.Expect(result.Backfill).To(BeFalse())
	}
}

func TestDefaultMatchMake_Load_1v1(t *testing.T) {
	t.Skip("skip race condition")
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	tickets := generateInMemoryTickets(1000)

	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: tickets,
	}
	startTime := time.Now()
	matches := mm.MakeMatches(testsetup.NewTestScope(), ticketProvider, get1v1Rules())

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	fmt.Println("results: ", len(results))
	g.Expect(len(results)).To(Equal(len(tickets) / 2))
	for _, result := range results {
		g.Expect(result.Backfill).To(BeFalse())
	}
	fmt.Println("run time ", time.Since(startTime))
}

func TestDefaultMatchMaker_CanSetServerNameAndOnMatch(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	DSName := "local-ds-name"
	attributesWithServerName := map[string]interface{}{"server_name": DSName}
	ticket1 := matchmaker.Ticket{TicketID: "ticket1", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user1"}}, TicketAttributes: attributesWithServerName}
	ticket2 := matchmaker.Ticket{TicketID: "ticket2", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user2"}}}
	ticket3 := matchmaker.Ticket{TicketID: "ticket3", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user3"}}, TicketAttributes: attributesWithServerName}

	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{ticket1, ticket2, ticket3},
	}
	matches := mm.MakeMatches(testsetup.NewTestScope(), ticketProvider, get1v1Rules())

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	g.Expect(len(results)).To(Equal(1))
	g.Expect(results[0].Tickets).To(ConsistOf(ticket1, ticket3))
	g.Expect(results[0].ServerName).To(Equal(DSName))
}

func TestDefaultMatchMaker_CanSetClientVersionOnMatch(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	ClientVersion := "ds-version"
	attributesWithClientVersion := map[string]interface{}{"client_version": ClientVersion}
	ticket1 := matchmaker.Ticket{CreatedAt: time.Now().Add(-5 * time.Second), TicketID: "ticket1", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user1"}}, TicketAttributes: map[string]interface{}{"client_version": "failed"}}
	ticket2 := matchmaker.Ticket{CreatedAt: time.Now().Add(-4 * time.Second), TicketID: "ticket2", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user2"}}, TicketAttributes: attributesWithClientVersion}
	ticket3 := matchmaker.Ticket{CreatedAt: time.Now().Add(-3 * time.Second), TicketID: "ticket3", MatchPool: "test-pool", Players: []player.PlayerData{{PlayerID: "user3"}}, TicketAttributes: attributesWithClientVersion}

	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{ticket1, ticket2, ticket3},
	}
	matches := mm.MakeMatches(testsetup.NewTestScope(), ticketProvider, get1v1Rules())

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	g.Expect(len(results)).To(Equal(1))
	g.Expect(results[0].Tickets).To(ConsistOf(ticket2, ticket3))
	g.Expect(results[0].ClientVersion).To(Equal(ClientVersion))
}

func TestDefaultMatchMaker_WithAttributes(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()
	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{
			{
				TicketID:         "first",
				MatchPool:        "test-pool",
				CreatedAt:        time.Time{},
				Players:          []player.PlayerData{{PlayerID: "user1"}},
				TicketAttributes: map[string]interface{}{"maps": []interface{}{"foo", "bar", "baz"}},
			},
			{
				TicketID:         "second",
				MatchPool:        "test-pool",
				CreatedAt:        time.Time{},
				Players:          []player.PlayerData{{PlayerID: "user2"}},
				TicketAttributes: map[string]interface{}{"maps": []interface{}{"alpha", "beta", "gamma"}},
			},
			{
				TicketID:         "third",
				MatchPool:        "test-pool",
				CreatedAt:        time.Time{},
				Players:          []player.PlayerData{{PlayerID: "user3"}},
				TicketAttributes: map[string]interface{}{"maps": []interface{}{"baz", "bar", "Magrathea"}},
			},
		},
	}

	matches := mm.MakeMatches(testsetup.NewTestScope(), ticketProvider, models.RuleSet{
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
	})

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	g.Expect(len(results)).To(Equal(1))
	ticketIDs := pie.Map(results[0].Tickets, func(ticket matchmaker.Ticket) string {
		return ticket.TicketID
	})
	g.Expect(ticketIDs).To(ConsistOf("first", "third"))
	g.Expect(pie.Keys(results[0].MatchAttributes)).To(ConsistOf("maps"))
	maps := pie.Map(results[0].MatchAttributes["maps"].([]interface{}), func(val interface{}) string {
		return val.(string)
	})
	g.Expect(maps).To(ConsistOf("bar", "baz")) // maps should be the overlap between the map options in the two tickets
}

func TestDefaultMatchMaker_WithFlexing(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()
	fiveSecondsAgo := time.Now().Add(-5 * time.Second)
	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{
			{
				TicketID:  "first",
				MatchPool: "test-pool",
				CreatedAt: fiveSecondsAgo,
				Players:   []player.PlayerData{{PlayerID: "user1", Attributes: map[string]interface{}{"mmr": 10}}},
			},
			{
				TicketID:  "second",
				MatchPool: "test-pool",
				CreatedAt: fiveSecondsAgo,
				Players:   []player.PlayerData{{PlayerID: "user2", Attributes: map[string]interface{}{"mmr": 150}}},
			},
		},
	}

	rules := models.RuleSet{
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
				Reference: float64(100),
			},
		},
		FlexingRule: []models.FlexingRule{
			{
				Duration: int64(30), // seconds before flexing rule is applied
				MatchingRule: models.MatchingRule{
					Attribute: "mmr",
					Criteria:  "distance",
					Reference: float64(200),
				},
			},
		},
	}

	matches := mm.MakeMatches(testsetup.NewTestScope(), ticketProvider, rules)

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	// the players should not match because the mmr difference is 100 and rules specify within 50
	g.Expect(len(results)).To(Equal(0))

	// but if the ticket was submitted a minute ago, it should match because the flex difference is
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	ticketProvider.Tickets[0].CreatedAt = oneMinuteAgo
	ticketProvider.Tickets[1].CreatedAt = oneMinuteAgo
	matches = mm.MakeMatches(testsetup.NewTestScope(), ticketProvider, rules)

	var flexResults []matchmaker.Match
	for match := range matches {
		flexResults = append(flexResults, match)
	}

	// the players should not match because the mmr difference is 100 and rules specify within 50
	g.Expect(len(flexResults)).To(Equal(1))
}

func TestDefaultMatchMaker_WithLatency_250(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()
	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{
			{
				TicketID:  "first",
				MatchPool: "test-pool",
				CreatedAt: time.Now().Add(-5 * time.Second),
				Players:   []player.PlayerData{{PlayerID: "user1"}},
				Latencies: map[string]int64{"us-west": 50, "eu-central": 100, "ap": 300},
			},
			{
				TicketID:  "second",
				MatchPool: "test-pool",
				CreatedAt: time.Now().Add(-4 * time.Second),
				Players:   []player.PlayerData{{PlayerID: "user2"}},
				Latencies: map[string]int64{"us-west": 100, "eu-central": 50, "ap": 100},
			},
			{
				TicketID:  "third",
				MatchPool: "test-pool",
				CreatedAt: time.Now().Add(-3 * time.Second),
				Players:   []player.PlayerData{{PlayerID: "user3"}},
				Latencies: map[string]int64{"us-west": 50, "eu-central": 100, "ap": 300},
			},
		},
	}

	matches := mm.MakeMatches(testsetup.NewTestScope(), ticketProvider, models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	})

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	g.Expect(len(results)).To(Equal(1))
	ticketIDs := pie.Map(results[0].Tickets, func(ticket matchmaker.Ticket) string {
		return ticket.TicketID
	})
	g.Expect(ticketIDs).To(ConsistOf("first", "third"))
	g.Expect(results[0].RegionPreference).To(ConsistOf("us-west"))
}

func TestDefaultMatchMaker_WithLatency(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()
	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{
			{
				TicketID:  "first",
				MatchPool: "test-pool",
				CreatedAt: time.Time{},
				Players:   []player.PlayerData{{PlayerID: "user1"}},
				Latencies: map[string]int64{"us-west": 50, "eu-central": 100, "ap": 300},
			},
			{
				TicketID:  "second",
				MatchPool: "test-pool",
				CreatedAt: time.Time{},
				Players:   []player.PlayerData{{PlayerID: "user2"}},
				Latencies: map[string]int64{"us-west": 500, "eu-central": 50, "ap": 100},
			},
			{
				TicketID:  "third",
				MatchPool: "test-pool",
				CreatedAt: time.Time{},
				Players:   []player.PlayerData{{PlayerID: "user3"}},
				Latencies: map[string]int64{"us-west": 50, "eu-central": 100, "ap": 300},
			},
		},
	}

	matches := mm.MakeMatches(testsetup.NewTestScope(), ticketProvider, models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	})

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	g.Expect(len(results)).To(Equal(1))
	ticketIDs := pie.Map(results[0].Tickets, func(ticket matchmaker.Ticket) string {
		return ticket.TicketID
	})
	g.Expect(ticketIDs).To(ConsistOf("first", "third"))
	g.Expect(results[0].RegionPreference).To(ConsistOf("us-west"))
}

func TestDefaultMatchMaker_PartialMatchResul_IndicatesNeedsBackfill(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: basic.SampleFiveSinglePlayerTickets,
	}

	partialMatchAllowedRules := models.RuleSet{
		AutoBackfill: true,
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	}
	matches := mm.MakeMatches(testsetup.NewTestScope(), ticketProvider, partialMatchAllowedRules)

	var results []matchmaker.Match
	for match := range matches {
		results = append(results, match)
	}

	g.Expect(len(results)).To(Equal(3))
	for _, result := range results[:2] {
		g.Expect(result.Backfill).To(BeFalse())
	}
	partialResult := results[2]
	g.Expect(len(partialResult.Tickets)).To(Equal(1))
	g.Expect(len(partialResult.Teams)).To(Equal(1))
	g.Expect(partialResult.Backfill).To(BeTrue())
}

// generating numbers that would represent tickets
func generateInMemoryTickets(amountOfTickets int) []matchmaker.Ticket {
	var tickets []matchmaker.Ticket
	for i := 0; i < amountOfTickets; i++ {
		var userName string
		userName = fmt.Sprintf("user" + strconv.Itoa(i))
		ticket := matchmaker.Ticket{
			TicketID:         strconv.Itoa(i),
			MatchPool:        "some-test-pool",
			Players:          []player.PlayerData{{PlayerID: player.IDFromString(userName), Attributes: map[string]interface{}{"mmr": float64(25)}}},
			TicketAttributes: nil,
			Latencies:        nil,
		}
		tickets = append(tickets, ticket)
	}
	return tickets
}

func newMatchLogic() matchmaker.MatchLogic {
	c := config.Config{}
	c.TicketChunkSize = 10
	metric := testsetup.NewMetrics()

	return New(&c, metric)
}

func TestToTeamsConvertion(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	now := time.Now().UTC()
	player1 := utils.GenerateUUID()
	player2 := utils.GenerateUUID()
	player3 := utils.GenerateUUID()

	sessionID := utils.GenerateUUID()
	teamIDs := []string{utils.GenerateUUID(), utils.GenerateUUID(), utils.GenerateUUID()}

	tickets := []matchmaker.Ticket{
		{
			Namespace:      "namespace",
			PartySessionID: sessionID,
			TicketID:       sessionID,
			MatchPool:      "matchpool",
			CreatedAt:      now,
			Players: []player.PlayerData{
				{
					PlayerID: player.IDFromString(player1),
				},
			},
			TicketAttributes: map[string]interface{}{
				"mmr": float64(100),
			},
			Latencies: map[string]int64{
				"us-west-2": 100,
			},
		},
		{
			Namespace:      "namespace",
			PartySessionID: "",
			TicketID:       utils.GenerateUUID(),
			MatchPool:      "matchpool",
			CreatedAt:      now,
			Players: []player.PlayerData{
				{
					PlayerID: player.IDFromString(player2),
				},
			},
			TicketAttributes: map[string]interface{}{
				"mmr": float64(100),
			},
			Latencies: map[string]int64{
				"us-west-2": 100,
			},
		},
	}
	matchingAllies := []models.MatchingAlly{
		{
			TeamID: teamIDs[0],
			MatchingParties: []models.MatchingParty{
				{
					PartyID: sessionID,
					PartyMembers: []models.PartyMember{
						{
							UserID: player1,
						},
					},
				},
			},
			PlayerCount: 1,
		},
		{
			TeamID: teamIDs[1],
			MatchingParties: []models.MatchingParty{
				{
					PartyID: "",
					PartyMembers: []models.PartyMember{
						{
							UserID: player2,
						},
					},
				},
			},
			PlayerCount: 1,
		},
		{
			TeamID: teamIDs[2],
			MatchingParties: []models.MatchingParty{
				{
					PartyID: "external",
					PartyMembers: []models.PartyMember{
						{
							UserID: player3,
						},
					},
				},
			},
			PlayerCount: 1,
		},
	}

	expectedResult := []matchmaker.Team{
		{
			TeamID:  teamIDs[0],
			UserIDs: []player.ID{player.IDFromString(player1)},
			Parties: []matchmaker.Party{
				{
					PartyID: sessionID,
					UserIDs: []string{player1},
				},
			},
		},
		{
			TeamID:  teamIDs[1],
			UserIDs: []player.ID{player.IDFromString(player2)},
			Parties: []matchmaker.Party{
				{
					PartyID: "",
					UserIDs: []string{player2},
				},
			},
		},
		{
			TeamID:  teamIDs[2],
			UserIDs: []player.ID{player.IDFromString(player3)},
			Parties: []matchmaker.Party{
				{
					PartyID: "external",
					UserIDs: []string{player3},
				},
			},
		},
	}

	result := toTeams(tickets, matchingAllies)

	g.Expect(result).To(Equal(expectedResult))
}
