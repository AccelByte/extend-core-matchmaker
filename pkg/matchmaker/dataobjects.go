// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package matchmaker provides the core interfaces and data structures for implementing
// matchmaking logic in the AccelByte extend-core-matchmaker system.
package matchmaker

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/playerdata"
)

// Ticket represents a matchmaking request in a particular match pool for one or more players.
// This is the primary data structure that contains all information needed for matchmaking.
type Ticket struct {
	Namespace        string                  // The namespace/tenant this ticket belongs to
	PartySessionID   string                  // Unique identifier for the party session
	TicketID         string                  // Unique identifier for this matchmaking ticket
	MatchPool        string                  // The match pool this ticket is queued in
	CreatedAt        time.Time               // When this ticket was created (used for queue ordering)
	Players          []playerdata.PlayerData // List of players in this ticket
	TicketAttributes map[string]interface{}  // Custom attributes for matchmaking rules
	Latencies        map[string]int64        // Network latency to different regions (region -> latency in ms)
	ExcludedSessions []string                // List of session IDs this ticket should not be matched into
}

// BackfillTicket represents a match result that needs additional players.
// This is used when an existing match session needs more players to start or continue.
type BackfillTicket struct {
	TicketID       string    // Unique identifier for this backfill request
	MatchPool      string    // The match pool this backfill is for
	CreatedAt      time.Time // When this backfill request was created
	PartialMatch   Match     // The existing match that needs more players
	MatchSessionID string    // The session ID of the match being backfilled
}

// BackfillProposal represents a proposal to update a match with additional players.
// This is the result of a successful backfill operation.
type BackfillProposal struct {
	BackfillTicketID string                 // ID of the original backfill ticket
	CreatedAt        time.Time              // When this proposal was created
	AddedTickets     []Ticket               // New tickets being added to the match
	ProposedTeams    []Team                 // Updated team composition
	ProposalID       string                 // Unique identifier for this proposal
	MatchPool        string                 // The match pool this proposal is for
	MatchSessionID   string                 // The session ID of the match being updated
	Attribute        map[string]interface{} // Additional attributes for the proposal
}

// Team is a set of players that have been matched onto the same team.
// Teams are used to group players within a match for team-based games.
type Team struct {
	TeamID  string          `json:",omitempty"` // Optional team identifier
	UserIDs []playerdata.ID // List of player IDs in this team
	Parties []Party         // List of parties in this team
}

// Party represents a group of players that joined matchmaking together.
// Parties are preserved as units during matchmaking to keep friends together.
type Party struct {
	//nolint:tagliatelle
	PartyID string   `json:"partyID"` // Unique identifier for the party
	UserIDs []string `json:"userIDs"` // List of user IDs in this party
}

// Match represents a matchmaking result with players placed on teams and tracking which tickets were included in the match.
// This is the final output of the matchmaking process.
type Match struct {
	Tickets                      []Ticket                     // All tickets that were matched together
	Teams                        []Team                       // Team assignments for the matched players
	RegionPreference             []string                     // Ordered list of preferred regions for this match
	MatchAttributes              map[string]interface{}       // Custom attributes for the match
	Backfill                     bool                         // False for complete matches, true if more players are desired
	ServerName                   string                       // Local DS name from ticket, used for directing match session to local DS
	ClientVersion                string                       // Specific game version from ticket, for overriding DS version
	ServerPoolSelectionParameter ServerPoolSelectionParameter // Parameters for server selection

	//NOTE: below is additional field, please note it haven't updated in proto yet
	PivotID   string    `json:",omitempty"` // ID of the pivot ticket used for this match
	Timestamp time.Time `json:",omitempty"` // When this match was created
}

// ServerPoolSelectionParameter server selection parameter.
// This structure contains parameters used to select the appropriate game server for the match.
type ServerPoolSelectionParameter struct {
	ServerProvider string   // "AMS" or empty for DS Armada
	Deployment     string   // Used by DS Armada if ServerProdiver is empty
	ClaimKeys      []string // Used by AMS if ServerProvider is AMS
}
