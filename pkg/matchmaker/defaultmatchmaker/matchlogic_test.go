package defaultmatchmaker

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker"
	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker/defaultmatchmaker/basic"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	player "github.com/AccelByte/extend-core-matchmaker/pkg/playerdata"
	"github.com/elliotchance/pie/v2"
)

var ExpectedFiveSinglePlayerTicketsMatchResults = []matchmaker.Match{
	{
		Tickets: []matchmaker.Ticket{basic.SampleFiveSinglePlayerTickets[0], basic.SampleFiveSinglePlayerTickets[1]},
		Teams: []matchmaker.Team{
			{
				UserIDs: pie.Map(basic.SampleFiveSinglePlayerTickets[0].Players, player.ToID),
				Parties: []matchmaker.Party{{
					PartyID: "",
					UserIDs: pie.Map(pie.Map(basic.SampleFiveSinglePlayerTickets[0].Players, player.ToID), func(id player.ID) string {
						return player.IDToString(id)
					}),
				}},
			},
			{
				UserIDs: pie.Map(basic.SampleFiveSinglePlayerTickets[1].Players, player.ToID),
				Parties: []matchmaker.Party{{
					PartyID: "",
					UserIDs: pie.Map(pie.Map(basic.SampleFiveSinglePlayerTickets[1].Players, player.ToID), func(id player.ID) string {
						return player.IDToString(id)
					}),
				}},
			},
		},
		RegionPreference: []string{""},
		MatchAttributes:  map[string]interface{}{},
	},
	{
		Tickets: []matchmaker.Ticket{basic.SampleFiveSinglePlayerTickets[2], basic.SampleFiveSinglePlayerTickets[3]},
		Teams: []matchmaker.Team{
			{
				UserIDs: pie.Map(basic.SampleFiveSinglePlayerTickets[2].Players, player.ToID),
				Parties: []matchmaker.Party{{
					PartyID: "",
					UserIDs: pie.Map(pie.Map(basic.SampleFiveSinglePlayerTickets[2].Players, player.ToID), func(id player.ID) string {
						return player.IDToString(id)
					}),
				}},
			},
			{
				UserIDs: pie.Map(basic.SampleFiveSinglePlayerTickets[3].Players, player.ToID),
				Parties: []matchmaker.Party{{
					PartyID: "",
					UserIDs: pie.Map(pie.Map(basic.SampleFiveSinglePlayerTickets[3].Players, player.ToID), func(id player.ID) string {
						return player.IDToString(id)
					}),
				}},
			},
		},
		RegionPreference: []string{""},
		MatchAttributes:  map[string]interface{}{},
	},
}

func get1v1Rules() models.RuleSet {
	return models.RuleSet{
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
}
