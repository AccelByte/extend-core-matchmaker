// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/config"
	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	player "github.com/AccelByte/extend-core-matchmaker/pkg/playerdata"
	"github.com/AccelByte/extend-core-matchmaker/pkg/testsetup"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
	. "github.com/onsi/gomega"
)

var backfill1v1RUles = models.RuleSet{
	AutoBackfill: true,
	AllianceRule: models.AllianceRule{
		MinNumber:       1,
		MaxNumber:       2,
		PlayerMinNumber: 1,
		PlayerMaxNumber: 1,
	},
	MatchingRule: []models.MatchingRule{
		{
			Attribute: "mmr",
			Criteria:  "distance",
			Reference: float64(100),
		},
	},
}

func TestDefaultMatchMaker_Backfill_FinishhesWhenNoBackfillTicketsExist(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	ticket := matchmaker.Ticket{
		TicketID: utils.GenerateUUID(),
		Players: []player.PlayerData{{
			PlayerID: "playerB",
		}},
	}

	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets:         []matchmaker.Ticket{ticket},
		BackfillTickets: []matchmaker.BackfillTicket{},
	}

	proposals := mm.BackfillMatches(testsetup.NewTestScope(), ticketProvider, backfill1v1RUles)

	var results []matchmaker.BackfillProposal
	for proposal := range proposals {
		results = append(results, proposal)
	}

	g.Expect(len(results)).To(Equal(0))
}

func TestDefaultMatchMaker_BackfillTicket1v1_BackfillNewTeam(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	ticket := matchmaker.Ticket{
		TicketID: utils.GenerateUUID(),
		Players: []player.PlayerData{{
			PlayerID:   "playerB",
			Attributes: map[string]interface{}{"mmr": 10},
			PartyID:    "partyB",
		}},
	}

	ticketIDA := utils.GenerateUUID()
	teamID := utils.GenerateUUID()
	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{ticket},
		BackfillTickets: []matchmaker.BackfillTicket{
			{
				TicketID: utils.GenerateUUID(),
				PartialMatch: matchmaker.Match{
					MatchAttributes: map[string]interface{}{models.AttributeMemberAttr: map[string]interface{}{"mmr": 10}},
					Tickets: []matchmaker.Ticket{{
						TicketID: ticketIDA,
						Players: []player.PlayerData{{
							PlayerID: "playerA", Attributes: map[string]interface{}{"mmr": 10},
						}},
					}},
					Teams: []matchmaker.Team{
						{
							TeamID:  teamID,
							UserIDs: []player.ID{"playerA"},
							Parties: []matchmaker.Party{{
								PartyID: "",
								UserIDs: []string{"playerA"},
							}},
						},
					},
					Backfill: true,
				},
				MatchSessionID: utils.GenerateUUID(),
			},
		},
	}
	proposals := mm.BackfillMatches(testsetup.NewTestScope(), ticketProvider, backfill1v1RUles)

	var results []matchmaker.BackfillProposal
	for proposal := range proposals {
		results = append(results, proposal)
	}
	newTeamFound := false
	for _, team := range results[0].ProposedTeams {
		if team.TeamID != teamID {
			newTeamFound = true
			break
		}
	}
	g.Expect(newTeamFound).To(BeTrue())
	g.Expect(results[0].ProposedTeams).To(HaveLen(2))
	// reset the team id to satisfy the test
	for i := range results[0].ProposedTeams {
		results[0].ProposedTeams[i].TeamID = ""
	}

	g.Expect(len(results)).To(Equal(1))
	g.Expect(results[0].BackfillTicketID).To(Equal(ticketProvider.BackfillTickets[0].TicketID))
	g.Expect(results[0].AddedTickets).To(ConsistOf(ticket))
	g.Expect(results[0].ProposedTeams).To(ConsistOf(
		matchmaker.Team{UserIDs: []player.ID{"playerA"}, Parties: []matchmaker.Party{{
			PartyID: "",
			UserIDs: []string{"playerA"},
		}}},
		matchmaker.Team{UserIDs: []player.ID{"playerB"}, Parties: []matchmaker.Party{{
			PartyID: "",
			UserIDs: []string{"playerB"},
		}}},
	))
}

func TestDefaultMatchMaker_BackfillTicket3v3_BackfillPlayersOnTeam(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	ticket1 := matchmaker.Ticket{
		TicketID: utils.GenerateUUID(),
		Players: []player.PlayerData{{
			PlayerID: "playerC",
		}},
	}
	ticket2 := matchmaker.Ticket{
		TicketID: utils.GenerateUUID(),
		Players: []player.PlayerData{{
			PlayerID: "playerD",
		}},
	}
	ticket3 := matchmaker.Ticket{
		TicketID: utils.GenerateUUID(),
		Players: []player.PlayerData{{
			PlayerID: "playerE",
		}},
	}

	teamsIDs := []string{utils.GenerateUUID(), utils.GenerateUUID()}
	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{ticket1, ticket2, ticket3},
		BackfillTickets: []matchmaker.BackfillTicket{
			{
				TicketID: utils.GenerateUUID(),
				PartialMatch: matchmaker.Match{
					Tickets: []matchmaker.Ticket{
						{
							TicketID: utils.GenerateUUID(),
							Players: []player.PlayerData{
								{PlayerID: "playerA1"},
								{PlayerID: "playerA2"},
								{PlayerID: "playerA3"},
							},
						},
						{
							TicketID: utils.GenerateUUID(),
							Players: []player.PlayerData{
								{PlayerID: "playerB"},
							},
						},
					},
					Teams: []matchmaker.Team{
						{TeamID: teamsIDs[0], UserIDs: []player.ID{"playerA1", "playerA2", "playerA3"}},
						{TeamID: teamsIDs[1], UserIDs: []player.ID{"playerB"}},
					},
					Backfill: true,
				},
				MatchSessionID: utils.GenerateUUID(),
			},
		},
	}
	proposals := mm.BackfillMatches(testsetup.NewTestScope(), ticketProvider, models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 3,
		},
		RebalanceEnable: models.FALSE(),
	})

	var results []matchmaker.BackfillProposal
	for proposal := range proposals {
		results = append(results, proposal)
	}

	g.Expect(len(results)).To(Equal(1))
	g.Expect(results[0].AddedTickets).To(ConsistOf(ticket1, ticket2))
	g.Expect(results[0].ProposedTeams).To(ConsistOf(
		matchmaker.Team{
			TeamID:  teamsIDs[0],
			UserIDs: []player.ID{"playerA1", "playerA2", "playerA3"},
			Parties: []matchmaker.Party{
				{
					PartyID: "",
					UserIDs: []string{"playerA1", "playerA2", "playerA3"},
				},
			},
		},
		matchmaker.Team{
			TeamID:  teamsIDs[1],
			UserIDs: []player.ID{"playerB", "playerC", "playerD"},
			Parties: []matchmaker.Party{
				{
					PartyID: "",
					UserIDs: []string{"playerB"},
				},
				{
					PartyID: "",
					UserIDs: []string{"playerC"},
				},
				{
					PartyID: "",
					UserIDs: []string{"playerD"},
				},
			},
		},
	))
}

func TestDefaultMatchMaker_BackfillTicket3v3_BackfillAfterExternalPlayersJoinSession(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	ticket1 := matchmaker.Ticket{
		TicketID: "ticket1",
		Players: []player.PlayerData{{
			PlayerID: "playerC",
		}},
	}
	ticket2 := matchmaker.Ticket{
		TicketID: "ticket2",
		Players: []player.PlayerData{{
			PlayerID: "playerD",
		}},
	}

	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{ticket1, ticket2},
		BackfillTickets: []matchmaker.BackfillTicket{
			{
				TicketID: utils.GenerateUUID(),
				PartialMatch: matchmaker.Match{
					Tickets: []matchmaker.Ticket{
						{
							TicketID: "partyTicket",
							Players: []player.PlayerData{
								{PlayerID: "playerA1"},
								{PlayerID: "playerA2"},
								{PlayerID: "playerA3"},
							},
						},
						{
							TicketID: "playerBTicket",
							Players: []player.PlayerData{
								{PlayerID: "playerB"},
							},
						},
					},
					Teams: []matchmaker.Team{
						{UserIDs: []player.ID{"playerA1", "playerA2", "playerA3"}},
						{UserIDs: []player.ID{"playerB", "externalPlayer1", "externalPlayer2"}},
					},
					Backfill: true,
				},
				MatchSessionID: utils.GenerateUUID(),
			},
		},
	}
	proposals := mm.BackfillMatches(testsetup.NewTestScope(), ticketProvider, models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 4,
		},
	})

	var results []matchmaker.BackfillProposal
	for proposal := range proposals {
		results = append(results, proposal)
	}

	g.Expect(len(results)).To(Equal(1))
	resultingTeams := results[0].ProposedTeams
	g.Expect(len(resultingTeams)).To(Equal(2))
	// occasionally the ordering of player ids is not the same, se we need to assert on each individually using ConsistOf
	g.Expect(resultingTeams[0].UserIDs).To(ConsistOf([]player.ID{"playerA1", "playerA2", "playerA3", "playerC"}))
	g.Expect(resultingTeams[1].UserIDs).To(ConsistOf([]player.ID{"playerB", "externalPlayer1", "externalPlayer2", "playerD"}))
	g.Expect(results[0].AddedTickets).To(ConsistOf(ticket1, ticket2))
}

func TestDefaultMatchMaker_BackfillTicket3v3_BackfillWithExternalPlayersAnd(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	mm := newMatchLogic()

	ticket1 := matchmaker.Ticket{
		TicketID: "ticket1",
		Players: []player.PlayerData{{
			PlayerID: "playerC", Attributes: map[string]interface{}{"mmr": 100},
		}},
	}
	ticket2 := matchmaker.Ticket{
		TicketID: "ticket2",
		Players: []player.PlayerData{{
			PlayerID: "playerD", Attributes: map[string]interface{}{"mmr": 10},
		}},
	}

	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{ticket1, ticket2},
		BackfillTickets: []matchmaker.BackfillTicket{
			{
				TicketID: utils.GenerateUUID(),
				PartialMatch: matchmaker.Match{
					Tickets: []matchmaker.Ticket{
						{
							TicketID: "partyTicket",
							Players: []player.PlayerData{
								{PlayerID: "playerA1", Attributes: map[string]interface{}{"mmr": 10}},
								{PlayerID: "playerA2", Attributes: map[string]interface{}{"mmr": 10}},
								{PlayerID: "playerA3", Attributes: map[string]interface{}{"mmr": 10}},
							},
						},
						{
							TicketID: "playerBTicket",
							Players: []player.PlayerData{
								{PlayerID: "playerB", Attributes: map[string]interface{}{"mmr": 10}},
							},
						},
					},
					Teams: []matchmaker.Team{
						{UserIDs: []player.ID{"playerA1", "playerA2", "playerA3"}},
						{UserIDs: []player.ID{"playerB", "externalPlayer1", "externalPlayer2"}},
					},
					Backfill: true,
				},
				MatchSessionID: utils.GenerateUUID(),
			},
		},
	}
	proposals := mm.BackfillMatches(testsetup.NewTestScope(), ticketProvider, models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       2,
			MaxNumber:       2,
			PlayerMinNumber: 2,
			PlayerMaxNumber: 4,
		},
		MatchingRule: []models.MatchingRule{
			{
				Attribute: "mmr",
				Criteria:  "distance",
				Reference: float64(100),
			},
		},
		RebalanceEnable: models.FALSE(),
	})

	var results []matchmaker.BackfillProposal
	for proposal := range proposals {
		results = append(results, proposal)
	}

	g.Expect(len(results)).To(Equal(1))
	resultingTeams := results[0].ProposedTeams
	g.Expect(len(resultingTeams)).To(Equal(2))
	// occasionally the ordering of player ids is not the same, se we need to assert on each individually using ConsistOf
	g.Expect(resultingTeams[0].UserIDs).To(ConsistOf([]player.ID{"playerA1", "playerA2", "playerA3", "playerD"}))
	g.Expect(resultingTeams[1].UserIDs).To(ConsistOf([]player.ID{"playerB", "externalPlayer1", "externalPlayer2", "playerC"}))
	g.Expect(results[0].AddedTickets).To(ConsistOf(ticket1, ticket2))
}

func TestCompose_BackfillProposal_SinglePlayerJoin(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)

	userID1 := utils.GenerateUUID()
	userID2 := utils.GenerateUUID()
	userID3 := utils.GenerateUUID()

	ticketID1 := utils.GenerateUUID()
	ticketID2 := utils.GenerateUUID()

	teamID := utils.GenerateUUID()
	mmResult := &models.MatchmakingResult{
		MatchID:  utils.GenerateUUID(),
		Channel:  "fauzanparty",
		Joinable: true,
		MatchingAllies: []models.MatchingAlly{
			{
				TeamID: teamID,
				MatchingParties: []models.MatchingParty{
					{
						PartyID: ticketID1,
						PartyMembers: []models.PartyMember{
							{UserID: userID1},
							{UserID: userID2},
						},
					},
					{
						PartyID: ticketID2,
						PartyMembers: []models.PartyMember{
							{UserID: userID3},
						},
					},
				},
			},
		},
	}
	satisfiedRequest := []models.MatchmakingRequest{
		{
			Channel: "fauzanparty",
			PartyID: ticketID2,
			PartyMembers: []models.PartyMember{
				{UserID: userID3},
			},
		},
	}
	sourceTickets := []matchmaker.Ticket{
		{
			Namespace:      "accelbytetesting",
			PartySessionID: "",
			TicketID:       ticketID2,
			MatchPool:      "fauzanparty",
			Players: []player.PlayerData{
				{PlayerID: player.IDFromString(userID3)},
			},
		},
	}
	expectedResult := matchmaker.BackfillProposal{
		BackfillTicketID: mmResult.MatchID,
		AddedTickets:     sourceTickets,
		MatchPool:        mmResult.Channel,
		ProposedTeams: []matchmaker.Team{
			{
				TeamID: teamID,
				UserIDs: []player.ID{
					player.IDFromString(userID1),
					player.IDFromString(userID2),
					player.IDFromString(userID3),
				},
				Parties: []matchmaker.Party{
					{PartyID: "", UserIDs: []string{userID1, userID2}},
					{PartyID: "", UserIDs: []string{userID3}},
				},
			},
		},
	}

	result := fromMatchResultToBackfillProposal(mmResult, satisfiedRequest, sourceTickets)

	g.Expect(result).To(Equal(expectedResult))
}

func TestCompose_BackfillProposal_PartyJoin(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)

	userID1 := utils.GenerateUUID()
	userID2 := utils.GenerateUUID()
	userID3 := utils.GenerateUUID()

	ticketID1 := utils.GenerateUUID()
	ticketID2 := utils.GenerateUUID()
	partyID1 := utils.GenerateUUID()

	teamID := utils.GenerateUUID()
	mmResult := &models.MatchmakingResult{
		MatchID:  utils.GenerateUUID(),
		Channel:  "fauzanparty",
		Joinable: true,
		MatchingAllies: []models.MatchingAlly{
			{
				TeamID: teamID,
				MatchingParties: []models.MatchingParty{
					{
						PartyID: ticketID1,
						PartyMembers: []models.PartyMember{
							{UserID: userID1},
							{UserID: userID2},
						},
					},
					{
						PartyID: ticketID2,
						PartyMembers: []models.PartyMember{
							{UserID: userID3},
						},
					},
				},
			},
		},
	}
	satisfiedRequest := []models.MatchmakingRequest{
		{
			Channel: "fauzanparty",
			PartyID: ticketID1,
			PartyMembers: []models.PartyMember{
				{UserID: userID1},
				{UserID: userID2},
			},
		},
	}
	sourceTickets := []matchmaker.Ticket{
		{
			Namespace:      "accelbytetesting",
			PartySessionID: partyID1,
			TicketID:       ticketID1,
			MatchPool:      "fauzanparty",
			Players: []player.PlayerData{
				{PlayerID: player.IDFromString(userID1)},
				{PlayerID: player.IDFromString(userID2)},
			},
		},
	}
	expectedResult := matchmaker.BackfillProposal{
		BackfillTicketID: mmResult.MatchID,
		AddedTickets:     sourceTickets,
		MatchPool:        mmResult.Channel,
		ProposedTeams: []matchmaker.Team{
			{
				TeamID: teamID,
				UserIDs: []player.ID{
					player.IDFromString(userID1),
					player.IDFromString(userID2),
					player.IDFromString(userID3),
				},
				Parties: []matchmaker.Party{
					{PartyID: partyID1, UserIDs: []string{userID1, userID2}},
					{PartyID: "", UserIDs: []string{userID3}},
				},
			},
		},
	}

	result := fromMatchResultToBackfillProposal(mmResult, satisfiedRequest, sourceTickets)

	g.Expect(result).To(Equal(expectedResult))
}

func TestDefaultMatchMaker_BackfillTicket1v1_BackfillMultiple(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	c := config.Config{}
	c.TicketChunkSize = 4
	mm := New(&c)

	users := []string{utils.GenerateUUID(), utils.GenerateUUID(), utils.GenerateUUID()}
	ticket := matchmaker.Ticket{
		TicketID: utils.GenerateUUID(),
		Players: []player.PlayerData{{
			PlayerID:   player.IDFromString(users[1]),
			Attributes: map[string]interface{}{"mmr": 10},
			PartyID:    users[1],
		}},
	}

	teamID := utils.GenerateUUID()
	ticketIDA := utils.GenerateUUID()
	ticketProvider := testsetup.StubMatchTicketProvider{
		Tickets: []matchmaker.Ticket{ticket},
		BackfillTickets: []matchmaker.BackfillTicket{
			{
				TicketID: utils.GenerateUUID(),
				PartialMatch: matchmaker.Match{
					MatchAttributes: map[string]interface{}{models.AttributeMemberAttr: map[string]interface{}{"mmr": 20}},
					Tickets: []matchmaker.Ticket{{
						TicketID: ticketIDA,
						Players: []player.PlayerData{{
							PlayerID: player.IDFromString(users[0]), Attributes: map[string]interface{}{"mmr": 20},
						}},
					}},
					Teams: []matchmaker.Team{
						{
							TeamID:  teamID,
							UserIDs: []player.ID{player.IDFromString(users[0])},
							Parties: []matchmaker.Party{{
								PartyID: "",
								UserIDs: []string{users[0]},
							}},
						},
					},
					Backfill: true,
				},
				MatchSessionID: utils.GenerateUUID(),
			},
			{
				TicketID: utils.GenerateUUID(),
				PartialMatch: matchmaker.Match{
					MatchAttributes: map[string]interface{}{models.AttributeMemberAttr: map[string]interface{}{"mmr": 10}},
					Tickets: []matchmaker.Ticket{{
						TicketID: ticketIDA,
						Players: []player.PlayerData{{
							PlayerID: player.IDFromString(users[2]), Attributes: map[string]interface{}{"mmr": 10},
						}},
					}},
					Teams: []matchmaker.Team{
						{
							TeamID:  teamID,
							UserIDs: []player.ID{player.IDFromString(users[2])},
							Parties: []matchmaker.Party{{
								PartyID: "",
								UserIDs: []string{users[2]},
							}},
						},
					},
					Backfill: true,
				},
				MatchSessionID: utils.GenerateUUID(),
			},
		},
	}
	proposals := mm.BackfillMatches(testsetup.NewTestScope(), ticketProvider, models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	})

	var results []matchmaker.BackfillProposal
	for proposal := range proposals {
		results = append(results, proposal)
	}

	// 1 ticket and 2 backfill tickets should only produce 1 match
	g.Expect(len(results)).To(Equal(1))
	g.Expect(results[0].BackfillTicketID).To(Equal(ticketProvider.BackfillTickets[0].TicketID))
	g.Expect(results[0].AddedTickets).To(ConsistOf(ticket))
	newTeamID := ""
	for _, team := range results[0].ProposedTeams {
		if team.TeamID != teamID {
			newTeamID = team.TeamID
			break
		}
	}
	for i := range results[0].ProposedTeams {
		results[0].ProposedTeams[i].TeamID = ""
	}
	g.Expect(len(results[0].ProposedTeams)).To(Equal(2))
	g.Expect(newTeamID).ShouldNot(BeEmpty())
	g.Expect(results[0].ProposedTeams).To(ConsistOf(
		matchmaker.Team{UserIDs: []player.ID{player.IDFromString(users[0])}, Parties: []matchmaker.Party{{
			PartyID: "",
			UserIDs: []string{users[0]},
		}}},
		matchmaker.Team{UserIDs: []player.ID{player.IDFromString(users[1])}, Parties: []matchmaker.Party{{
			PartyID: "",
			UserIDs: []string{users[1]},
		}}},
	))
}

func TestDefaultMatchMaker_BackfillWithEmptyStats(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	c := config.Config{}
	c.TicketChunkSize = 4
	mm := New(&c)

	matchRule := models.RuleSet{
		MatchingRule: []models.MatchingRule{{
			Attribute: "mmr",
			Criteria:  constants.DistanceCriteria,
			Reference: 100,
		}},
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	}

	ticketIDA := utils.GenerateUUID()
	ticketIDs := []string{utils.GenerateUUID(), utils.GenerateUUID()}
	backfillTicket := matchmaker.BackfillTicket{
		TicketID: utils.GenerateUUID(),
		PartialMatch: matchmaker.Match{
			Tickets: []matchmaker.Ticket{{
				TicketID: ticketIDA,
				Players:  []player.PlayerData{},
			}},
			Teams: []matchmaker.Team{
				{
					TeamID:  ticketIDs[0],
					UserIDs: []player.ID{"playerA"},
					Parties: []matchmaker.Party{{
						PartyID: "",
						UserIDs: []string{"playerA"},
					}},
				},
				{TeamID: ticketIDs[1]},
			},
			Backfill: true,
		},
		MatchSessionID: utils.GenerateUUID(),
	}

	inRangeTicket := matchmaker.Ticket{
		TicketID: utils.GenerateUUID(),
		Players: []player.PlayerData{{
			PlayerID:   "playerB",
			Attributes: map[string]interface{}{"mmr": 10},
			PartyID:    "partyB",
		}},
	}

	outOfRangeTicket := matchmaker.Ticket{
		TicketID: utils.GenerateUUID(),
		Players: []player.PlayerData{{
			PlayerID:   "playerC",
			Attributes: map[string]interface{}{"mmr": 500},
			PartyID:    "partyC",
		}},
	}

	t.Run("inRange", func(t *testing.T) {
		ticketProvider := testsetup.StubMatchTicketProvider{
			Tickets:         []matchmaker.Ticket{inRangeTicket},
			BackfillTickets: []matchmaker.BackfillTicket{backfillTicket},
		}

		proposals := mm.BackfillMatches(testsetup.NewTestScope(), ticketProvider, matchRule)

		var results []matchmaker.BackfillProposal
		for proposal := range proposals {
			results = append(results, proposal)
		}

		// should be produce 1 match because mmr 10 is in range (0 +- 100)
		g.Expect(results).To(HaveLen(1))
		g.Expect(results[0].BackfillTicketID).To(Equal(ticketProvider.BackfillTickets[0].TicketID))
		g.Expect(results[0].AddedTickets).To(ConsistOf(inRangeTicket))
		g.Expect(results[0].ProposedTeams).To(ConsistOf(
			matchmaker.Team{TeamID: ticketIDs[0], UserIDs: []player.ID{"playerA"}, Parties: []matchmaker.Party{{
				PartyID: externalPartyID,
				UserIDs: []string{"playerA"},
			}}},
			matchmaker.Team{TeamID: ticketIDs[1], UserIDs: []player.ID{"playerB"}, Parties: []matchmaker.Party{{
				PartyID: "",
				UserIDs: []string{"playerB"},
			}}},
		))
	})

	t.Run("outOfRange", func(t *testing.T) {
		ticketProvider := testsetup.StubMatchTicketProvider{
			Tickets:         []matchmaker.Ticket{outOfRangeTicket},
			BackfillTickets: []matchmaker.BackfillTicket{backfillTicket},
		}

		proposals := mm.BackfillMatches(testsetup.NewTestScope(), ticketProvider, matchRule)

		var results []matchmaker.BackfillProposal
		for proposal := range proposals {
			results = append(results, proposal)
		}

		// should not be produce match because mmr 500 is out of range (0 +- 100)
		g.Expect(results).To(BeEmpty())
	})
}

func TestDefaultMatchMaker_BackfillWithExcludedSessions(t *testing.T) {
	g := testsetup.ParallelWithGomega(t)
	c := config.Config{}
	c.TicketChunkSize = 3
	mm := New(&c)

	matchRule := models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       2,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 1,
		},
	}

	ticketIDA := utils.GenerateUUID()
	backfillTicket := matchmaker.BackfillTicket{
		TicketID: utils.GenerateUUID(),
		PartialMatch: matchmaker.Match{
			Tickets: []matchmaker.Ticket{{
				TicketID: ticketIDA,
				Players:  []player.PlayerData{},
			}},
			Teams: []matchmaker.Team{
				{
					UserIDs: []player.ID{"playerA"},
					Parties: []matchmaker.Party{{
						PartyID: "",
						UserIDs: []string{"playerA"},
					}},
				},
				{},
			},
			Backfill: true,
		},
		MatchSessionID: utils.GenerateUUID(),
	}

	t.Run("providing excludedSessions", func(t *testing.T) {
		ticket := matchmaker.Ticket{
			TicketID: utils.GenerateUUID(),
			Players: []player.PlayerData{{
				PlayerID:   "playerB",
				Attributes: map[string]interface{}{"mmr": 10},
				PartyID:    "partyB",
			}},
			ExcludedSessions: []string{backfillTicket.MatchSessionID},
		}

		ticketProvider := testsetup.StubMatchTicketProvider{
			Tickets:         []matchmaker.Ticket{ticket},
			BackfillTickets: []matchmaker.BackfillTicket{backfillTicket},
		}

		proposals := mm.BackfillMatches(testsetup.NewTestScope(), ticketProvider, matchRule)

		var results []matchmaker.BackfillProposal
		for proposal := range proposals {
			results = append(results, proposal)
		}

		// should be produce 0 match
		g.Expect(results).To(HaveLen(0))
	})

	t.Run("empty excludedSessions", func(t *testing.T) {
		ticket := matchmaker.Ticket{
			TicketID: utils.GenerateUUID(),
			Players: []player.PlayerData{{
				PlayerID:   "playerB",
				Attributes: map[string]interface{}{"mmr": 10},
				PartyID:    "partyB",
			}},
			ExcludedSessions: []string{},
		}

		ticketProvider := testsetup.StubMatchTicketProvider{
			Tickets:         []matchmaker.Ticket{ticket},
			BackfillTickets: []matchmaker.BackfillTicket{backfillTicket},
		}

		proposals := mm.BackfillMatches(testsetup.NewTestScope(), ticketProvider, matchRule)

		var results []matchmaker.BackfillProposal
		for proposal := range proposals {
			results = append(results, proposal)
		}

		// should be produce 1 match
		g.Expect(results).To(HaveLen(1))
		g.Expect(results[0].BackfillTicketID).To(Equal(ticketProvider.BackfillTickets[0].TicketID))
		g.Expect(results[0].AddedTickets).To(ConsistOf(ticket))
		g.Expect(results[0].ProposedTeams).To(HaveLen(2))
		for i := range results[0].ProposedTeams {
			g.Expect(results[0].ProposedTeams[i].TeamID).NotTo(BeEmpty())
			results[0].ProposedTeams[i].TeamID = ""
		}
		g.Expect(results[0].ProposedTeams).To(ConsistOf(
			matchmaker.Team{UserIDs: []player.ID{"playerA"}, Parties: []matchmaker.Party{{
				PartyID: externalPartyID,
				UserIDs: []string{"playerA"},
			}}},
			matchmaker.Team{UserIDs: []player.ID{"playerB"}, Parties: []matchmaker.Party{{
				PartyID: "",
				UserIDs: []string{"playerB"},
			}}},
		))
	})
}
