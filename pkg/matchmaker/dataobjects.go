// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package matchmaker

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/playerdata"
)

// Ticket represents a matchmaking request in a particular match pool for one or more players.
type Ticket struct {
	Namespace        string
	PartySessionID   string
	TicketID         string
	MatchPool        string
	CreatedAt        time.Time
	Players          []playerdata.PlayerData
	TicketAttributes map[string]interface{}
	Latencies        map[string]int64
	ExcludedSessions []string
}

// BackfillTicket represents a match result that needs additional players.
type BackfillTicket struct {
	TicketID       string
	MatchPool      string
	CreatedAt      time.Time
	PartialMatch   Match
	MatchSessionID string
}

// BackfillProposal represents a proposal to update a match with additional players.
type BackfillProposal struct {
	BackfillTicketID string
	CreatedAt        time.Time
	AddedTickets     []Ticket
	ProposedTeams    []Team
	ProposalID       string
	MatchPool        string
	MatchSessionID   string
	Attribute        map[string]interface{}
}

// Team is a set of players that have been matched onto the same team.
type Team struct {
	TeamID  string `json:",omitempty"`
	UserIDs []playerdata.ID
	Parties []Party
}

type Party struct {
	//nolint:tagliatelle
	PartyID string   `json:"partyID"`
	UserIDs []string `json:"userIDs"`
}

// Match represents a matchmaking result with players placed on teams and tracking which tickets were included in the match.
type Match struct {
	Tickets                      []Ticket
	Teams                        []Team
	RegionPreference             []string // ordered list of
	MatchAttributes              map[string]interface{}
	Backfill                     bool   // false for complete matches, true if more players are desired.
	ServerName                   string // fill this with local DS name from ticket, used for directing match session to local DS
	ClientVersion                string // fill this with specific game version from ticket, for overriding DS version
	ServerPoolSelectionParameter ServerPoolSelectionParameter

	//NOTE: below is additional field, please note it haven't updated in proto yet
	PivotID   string    `json:",omitempty"`
	Timestamp time.Time `json:",omitempty"`
}

// ServerPoolSelectionParameter server selection parameter.
type ServerPoolSelectionParameter struct {
	ServerProvider string   // "AMS" or empty for DS Armada
	Deployment     string   // used by DS Armada if ServerProdiver is empty
	ClaimKeys      []string // used by AMS if ServerProvider is AMS
}
