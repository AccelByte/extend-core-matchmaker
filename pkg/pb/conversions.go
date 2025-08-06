// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package matchfunction

import (
	"encoding/json"

	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker"
	"github.com/AccelByte/extend-core-matchmaker/pkg/playerdata"

	pie_ "github.com/elliotchance/pie/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MatchfunctionTicketToProtoTicket will convert a matchmaker ticket to a proto ticket
func MatchfunctionTicketToProtoTicket(ticket matchmaker.Ticket) *Ticket {
	// convert ticket attributes to proto struct
	ticketAttrs, err := structpb.NewStruct(ticket.TicketAttributes)
	if err != nil {
		logrus.Errorf("error on structpb for ticket attributes")
	}

	return &Ticket{
		state:            protoimpl.MessageState{},
		sizeCache:        0,
		unknownFields:    nil,
		TicketId:         ticket.TicketID,
		MatchPool:        ticket.MatchPool,
		CreatedAt:        timestamppb.New(ticket.CreatedAt),
		Players:          pie_.Map(ticket.Players, toProtoPlayerData),
		TicketAttributes: ticketAttrs,
		Latencies:        ticket.Latencies,
		PartySessionId:   ticket.PartySessionID,
		Namespace:        ticket.Namespace,
		//ExcludedSessions: ticket.ExcludedSessions,
	}
}

// ProtoTicketToMatchfunctionTicket will convert a proto ticket to a matchmaker ticket
func ProtoTicketToMatchfunctionTicket(ticket *Ticket) matchmaker.Ticket {
	return matchmaker.Ticket{
		TicketID:  ticket.TicketId,
		MatchPool: ticket.MatchPool,
		CreatedAt: ticket.CreatedAt.AsTime(),
		Players: pie_.Map(ticket.Players, func(p *Ticket_PlayerData) playerdata.PlayerData {
			return playerdata.PlayerData{
				PlayerID:   playerdata.IDFromString(p.PlayerId),
				Attributes: p.Attributes.AsMap(),
			}
		}),
		TicketAttributes: ticket.TicketAttributes.AsMap(),
		Latencies:        ticket.Latencies,
		PartySessionID:   ticket.PartySessionId,
		Namespace:        ticket.Namespace,
		//ExcludedSessions: ticket.ExcludedSessions,
	}
}

// ProtoMatchToMatchfunctionMatch will convert a proto match to a matchmaker match
func ProtoMatchToMatchfunctionMatch(match *Match) matchmaker.Match {
	var serverPool matchmaker.ServerPoolSelectionParameter
	if match.ServerPool != nil {
		serverPool = matchmaker.ServerPoolSelectionParameter{
			Deployment:     match.ServerPool.Deployment,
			ServerProvider: match.ServerPool.ServerProvider,
			ClaimKeys:      match.ServerPool.ClaimKeys,
		}
	}

	return matchmaker.Match{
		Tickets: pie_.Map(match.Tickets, func(m *Ticket) matchmaker.Ticket {
			return matchmaker.Ticket{
				TicketID:  m.TicketId,
				MatchPool: m.MatchPool,
				CreatedAt: m.CreatedAt.AsTime(),
				Players: pie_.Map(m.Players, func(p *Ticket_PlayerData) playerdata.PlayerData {
					return playerdata.PlayerData{PlayerID: playerdata.IDFromString(p.PlayerId), Attributes: p.Attributes.AsMap()}
				}),
				TicketAttributes: m.TicketAttributes.AsMap(),
				Latencies:        m.Latencies,
				PartySessionID:   m.PartySessionId,
				Namespace:        m.Namespace,
			}
		}),
		Teams:                        protoMatchTeamToMatch(match.Teams),
		RegionPreference:             match.RegionPreferences,
		MatchAttributes:              match.MatchAttributes.AsMap(),
		Backfill:                     match.Backfill,
		ServerName:                   match.ServerName,
		ClientVersion:                match.ClientVersion,
		ServerPoolSelectionParameter: serverPool,
	}
}

func protoMatchTeamToMatch(protoTeams []*Match_Team) []matchmaker.Team {
	var teams []matchmaker.Team
	for _, protoTeam := range protoTeams {
		team := matchmaker.Team{
			TeamID: protoTeam.TeamId,
			UserIDs: pie_.Map(protoTeam.UserIds, func(s string) playerdata.ID {
				return playerdata.IDFromString(s)
			}),
			Parties: pie_.Map(protoTeam.Parties, func(party *Party) matchmaker.Party {
				return matchmaker.Party{
					UserIDs: party.UserIds,
					PartyID: party.PartyId,
				}
			}),
		}
		teams = append(teams, team)
	}

	return teams
}

// MatchfunctionMatchToProtoMatch will conert a matchmaker match to a proto match
func MatchfunctionMatchToProtoMatch(match matchmaker.Match) *Match {
	log := logrus.WithField("function", "MatchfunctionMatchToProtoMatch")

	var err error
	match.MatchAttributes, err = convertAttribute(match.MatchAttributes)
	if err != nil {
		log.Errorf("value match attributes marshal : %v error : %s", match.MatchAttributes, err.Error())
		log.Errorf("error on convertAttribute for match attributes")
	}
	matchAttrs, mErr := structpb.NewStruct(match.MatchAttributes)
	if mErr != nil {
		log.Errorf("value match attributes : %v error : %s", match.MatchAttributes, mErr.Error())
		log.Errorf("error on structpb for match attributes")
	}
	return &Match{
		Tickets: pie_.Map(match.Tickets, func(ticket matchmaker.Ticket) *Ticket {
			return MatchfunctionTicketToProtoTicket(ticket)
		}),
		Teams: pie_.Map(match.Teams, func(team matchmaker.Team) *Match_Team {
			return &Match_Team{
				TeamId: team.TeamID,
				UserIds: pie_.Map(team.UserIDs, func(x playerdata.ID) string {
					return playerdata.IDToString(x)
				}),
				Parties: pie_.Map(team.Parties, func(p matchmaker.Party) *Party {
					return &Party{
						PartyId: p.PartyID,
						UserIds: p.UserIDs,
					}
				}),
			}
		}),
		RegionPreferences: match.RegionPreference,
		MatchAttributes:   matchAttrs,
		Backfill:          match.Backfill,
		ServerName:        match.ServerName,
		ClientVersion:     match.ClientVersion,
		ServerPool: &ServerPool{
			ServerProvider: match.ServerPoolSelectionParameter.ServerProvider,
			Deployment:     match.ServerPoolSelectionParameter.Deployment,
			ClaimKeys:      match.ServerPoolSelectionParameter.ClaimKeys,
		},
	}
}

func ProtoBackfillProposalToMatchfunctionBackfillProposal(match *BackfillProposal) matchmaker.BackfillProposal {
	return matchmaker.BackfillProposal{
		BackfillTicketID: match.BackfillTicketId,
		CreatedAt:        match.CreatedAt.AsTime(),
		AddedTickets: pie_.Map(match.AddedTickets, func(m *Ticket) matchmaker.Ticket {
			return matchmaker.Ticket{
				TicketID:  m.TicketId,
				MatchPool: m.MatchPool,
				CreatedAt: m.CreatedAt.AsTime(),
				Players: pie_.Map(m.Players, func(p *Ticket_PlayerData) playerdata.PlayerData {
					return playerdata.PlayerData{PlayerID: playerdata.IDFromString(p.PlayerId), Attributes: p.Attributes.AsMap()}
				}),
				TicketAttributes: m.TicketAttributes.AsMap(),
				Latencies:        m.Latencies,
				PartySessionID:   m.PartySessionId,
				Namespace:        m.Namespace,
			}
		}),
		ProposedTeams: pie_.Map(match.ProposedTeams, func(team *BackfillProposal_Team) matchmaker.Team {
			return matchmaker.Team{
				TeamID: team.TeamId,
				UserIDs: pie_.Map(team.UserIds, func(p string) playerdata.ID {
					return playerdata.ID(p)
				}),
				Parties: pie_.Map(team.Parties, func(p *Party) matchmaker.Party {
					return matchmaker.Party{
						PartyID: p.PartyId,
						UserIDs: p.UserIds,
					}
				}),
			}
		}),
		MatchPool:      match.MatchPool,
		ProposalID:     match.ProposalId,
		MatchSessionID: match.MatchSessionId,
	}
}

func toProtoPlayerData(p playerdata.PlayerData) *Ticket_PlayerData {
	var err error
	p.Attributes, err = convertAttribute(p.Attributes)
	if err != nil {
		logrus.Errorf("value player attributes marshal : %v error : %s", p.Attributes, err.Error())
		logrus.Errorf("error on convertAttribute for player attributes")
	}
	playerAttrs, paErr := structpb.NewStruct(p.Attributes)
	if paErr != nil {
		logrus.Errorf("value player attributes : %v error : %s", p.Attributes, paErr.Error())
		logrus.Errorf("failed to create new proto struct for player attributes")
	}
	return &Ticket_PlayerData{
		state:      protoimpl.MessageState{},
		sizeCache:  0,
		PlayerId:   playerdata.IDToString(p.PlayerID),
		Attributes: playerAttrs,
	}
}

func newStrucPb(m map[string]interface{}) *structpb.Struct {
	attributes, err := convertAttribute(m)
	if err != nil {
		logrus.WithError(err).WithField("map", m).Error("failed to convertAttribute")
	}

	strucPb, err := structpb.NewStruct(attributes)
	if err != nil {
		logrus.WithError(err).WithField("map", m).Error("failed to create new proto struct")
	}
	return strucPb
}

func playerIDToString(id playerdata.ID) string {
	return string(id)
}

func toProtoParty(party matchmaker.Party) *Party {
	return &Party{
		PartyId: party.PartyID,
		UserIds: party.UserIDs,
	}
}

func toBackfillProposalTeam(team matchmaker.Team) *BackfillProposal_Team {
	return &BackfillProposal_Team{
		TeamId:  team.TeamID,
		UserIds: pie_.Map(team.UserIDs, playerIDToString),
		Parties: pie_.Map(team.Parties, toProtoParty),
	}
}

// ProtoBackfillProposalToMatchfunctionBackfillProposal will convert a proto backfill proposal to a matchmaker backfill proposal
func MatchfunctionBackfillProposalToProtoBackfillProposal(match matchmaker.BackfillProposal) *BackfillProposal {
	return &BackfillProposal{
		BackfillTicketId: match.BackfillTicketID,
		CreatedAt:        timestamppb.New(match.CreatedAt),
		AddedTickets: pie_.Map(match.AddedTickets, func(t matchmaker.Ticket) *Ticket {
			return &Ticket{
				TicketId:         t.TicketID,
				MatchPool:        t.MatchPool,
				CreatedAt:        timestamppb.New(t.CreatedAt),
				Players:          pie_.Map(t.Players, toProtoPlayerData),
				TicketAttributes: newStrucPb(t.TicketAttributes),
				Latencies:        t.Latencies,
				PartySessionId:   t.PartySessionID,
				Namespace:        t.Namespace,
			}
		}),
		ProposedTeams:  pie_.Map(match.ProposedTeams, toBackfillProposalTeam),
		ProposalId:     match.ProposalID,
		MatchPool:      match.MatchPool,
		MatchSessionId: match.MatchSessionID,
	}
}

// MatchfunctionBackfillTicketToProtoBackfillTicket will convert a matchmaker backfill ticket to a proto backfill ticket
func MatchfunctionBackfillTicketToProtoBackfillTicket(backfillTicket matchmaker.BackfillTicket) *BackfillTicket {
	log := logrus.WithField("function", "MatchfunctionBackfillTicketToProtoBackfillTicket")
	match := backfillTicket.PartialMatch
	var err error
	match.MatchAttributes, err = convertAttribute(match.MatchAttributes)
	if err != nil {
		log.Errorf("value match attributes marshal : %v error : %s", match.MatchAttributes, err.Error())
		log.Errorf("error on convertAttribute for match attributes")
	}
	// convert ticket attributes to proto struct
	ticketAttrs, err := structpb.NewStruct(match.MatchAttributes)
	if err != nil {
		log.Errorf("value match attributes : %v error : %s", match.MatchAttributes, err.Error())
		log.Errorf("error on structpb for ticket attributes")
	}
	var backfillTeams []*BackfillTicket_Team
	for _, team := range match.Teams {
		userIDs := pie_.Map(team.UserIDs, func(p playerdata.ID) string {
			return playerdata.IDToString(p)
		})
		if len(userIDs) > 0 {
			backfillTeams = append(backfillTeams, &BackfillTicket_Team{
				TeamId:  team.TeamID,
				UserIds: userIDs,
				Parties: pie_.Map(team.Parties, func(p matchmaker.Party) *Party {
					return &Party{
						PartyId: p.PartyID,
						UserIds: p.UserIDs,
					}
				}),
			})
		}
	}
	tickets := pie_.Map(match.Tickets, func(t matchmaker.Ticket) *Ticket {
		t.TicketAttributes, err = convertAttribute(t.TicketAttributes)
		if err != nil {
			log.Errorf("value ticket attributes marshal : %v error : %s", t.TicketAttributes, err.Error())
			log.Errorf("error on convert Attribute ticket attributes")
		}

		attributes, err := structpb.NewStruct(t.TicketAttributes)
		if err != nil {
			log.Errorf("value ticket attributes : %v error : %s", t.TicketAttributes, err.Error())
			log.Errorf("error on structpb for ticket attributes")
		}
		playerData := pie_.Map(t.Players, toProtoPlayerData)
		return &Ticket{
			state:            protoimpl.MessageState{},
			sizeCache:        0,
			TicketId:         t.TicketID,
			MatchPool:        t.MatchPool,
			CreatedAt:        timestamppb.New(t.CreatedAt),
			Players:          playerData,
			TicketAttributes: attributes,
			Latencies:        t.Latencies,
			PartySessionId:   t.PartySessionID,
			Namespace:        t.Namespace,
		}
	})
	return &BackfillTicket{
		state:     protoimpl.MessageState{},
		sizeCache: 0,
		TicketId:  backfillTicket.TicketID,
		MatchPool: backfillTicket.MatchPool,
		CreatedAt: timestamppb.New(backfillTicket.CreatedAt),
		PartialMatch: &BackfillTicket_PartialMatch{
			state:             protoimpl.MessageState{},
			sizeCache:         0,
			Tickets:           tickets,
			Backfill:          match.Backfill,
			ServerName:        match.ServerName,
			ClientVersion:     match.ClientVersion,
			Teams:             backfillTeams,
			MatchAttributes:   ticketAttrs,
			RegionPreferences: match.RegionPreference,
		},
		MatchSessionId: backfillTicket.MatchSessionID,
	}
}

func ProtoBackfillTicketToMatchfunctionBackfillTicket(ticket *BackfillTicket) matchmaker.BackfillTicket {
	return matchmaker.BackfillTicket{
		TicketID:       ticket.TicketId,
		MatchPool:      ticket.MatchPool,
		CreatedAt:      ticket.CreatedAt.AsTime(),
		PartialMatch:   ProtoPartialMatchToMatchfunctionMatch(ticket.PartialMatch),
		MatchSessionID: ticket.MatchSessionId,
		//MatchSessionVersion: ticket.MatchSessionVersion,
	}
}

func protoBackfillTicketTeamToMatch(protoTeams []*BackfillTicket_Team) []matchmaker.Team {
	var teams []matchmaker.Team
	for _, protoTeam := range protoTeams {
		team := matchmaker.Team{
			TeamID: protoTeam.TeamId,
			UserIDs: pie_.Map(protoTeam.UserIds, func(s string) playerdata.ID {
				return playerdata.IDFromString(s)
			}),
			Parties: pie_.Map(protoTeam.Parties, func(party *Party) matchmaker.Party {
				return matchmaker.Party{
					UserIDs: party.UserIds,
					PartyID: party.PartyId,
				}
			}),
		}
		teams = append(teams, team)
	}

	return teams
}

func ProtoPartialMatchToMatchfunctionMatch(match *BackfillTicket_PartialMatch) matchmaker.Match {
	return matchmaker.Match{
		Tickets: pie_.Map(match.Tickets, func(m *Ticket) matchmaker.Ticket {
			return matchmaker.Ticket{
				TicketID:  m.TicketId,
				MatchPool: m.MatchPool,
				CreatedAt: m.CreatedAt.AsTime(),
				Players: pie_.Map(m.Players, func(p *Ticket_PlayerData) playerdata.PlayerData {
					return playerdata.PlayerData{PlayerID: playerdata.IDFromString(p.PlayerId), Attributes: p.Attributes.AsMap()}
				}),
				TicketAttributes: m.TicketAttributes.AsMap(),
				Latencies:        m.Latencies,
				PartySessionID:   m.PartySessionId,
				Namespace:        m.Namespace,
			}
		}),
		Teams:            protoBackfillTicketTeamToMatch(match.Teams),
		RegionPreference: match.RegionPreferences,
		MatchAttributes:  match.MatchAttributes.AsMap(),
		Backfill:         match.Backfill,
		ServerName:       match.ServerName,
		ClientVersion:    match.ClientVersion,
	}
}

func PlayerDataToParties(players []playerdata.PlayerData) []matchmaker.Party {
	mapParty := make(map[string][]string)

	for _, player := range players {
		mapParty[player.PartyID] = append(mapParty[player.PartyID], string(player.PlayerID))
	}

	parties := make([]matchmaker.Party, 0, len(mapParty))
	for partyID, userIDs := range mapParty {
		parties = append(parties, matchmaker.Party{
			PartyID: partyID,
			UserIDs: userIDs,
		})
	}

	return parties
}

func convertAttribute(data map[string]interface{}) (map[string]interface{}, error) {
	marshal, err := json.Marshal(data)
	if err != nil {
		return data, err
	}

	result := make(map[string]interface{})
	err = json.Unmarshal(marshal, &result)
	if err != nil {
		return data, err
	}

	return result, err
}
