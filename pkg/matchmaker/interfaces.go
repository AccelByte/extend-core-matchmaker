// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

// Package matchmaker provides the core interfaces and data structures for implementing
// matchmaking logic in the AccelByte extend-core-matchmaker system.
package matchmaker

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

/*
MatchLogic is a thing that has logic to take Tickets and make Matches. It also can decode match rules from json
into a structure that it understands. When matchmaking for a particular pool is desired, the matchmaker core engine
will look up the match maker and ruleset (json) for that pool and ask the match logic to decode the ruleset
then will call MakeMatches, passing the decoded ruleset and a TicketProvider which will provide tickets from the
pool to match together.

MakeMatches returns a channel to which it will post matches as they are found, and should close the channel when
all matches are exhausted.  It should also watch for cancellation on the provided scope.Ctx, at which point it should
stop looking for matches and close the result channel.

ValidateTicket should return false AND api.ErrInvalidRequest when a ticket is not allowed to be queued
*/
type MatchLogic interface {
	// BackfillMatches attempts to find additional players for existing matches that need more participants.
	// Returns a channel of BackfillProposal objects as they are found.
	BackfillMatches(scope *envelope.Scope, ticketProvider TicketProvider, matchRules interface{}) <-chan BackfillProposal

	// MakeMatches performs the core matchmaking logic to create new matches from available tickets.
	// Returns a channel of Match objects as they are created.
	MakeMatches(scope *envelope.Scope, ticketProvider TicketProvider, matchRules interface{}) <-chan Match

	// RulesFromJSON decodes a JSON string into a structured ruleset that the matchmaker can understand.
	// This allows for dynamic rule configuration without code changes.
	RulesFromJSON(scope *envelope.Scope, json string) (interface{}, error)

	// GetStatCodes returns a list of statistic codes that this matchmaker tracks for monitoring purposes.
	GetStatCodes(scope *envelope.Scope, matchRules interface{}) []string

	// ValidateTicket checks if a matchmaking ticket meets all requirements to be queued.
	// Returns false and an error if the ticket is invalid.
	ValidateTicket(scope *envelope.Scope, matchTicket Ticket, matchRules interface{}) (bool, error)

	// EnrichTicket adds additional data or modifies the ticket before matchmaking begins.
	// This can include adding computed attributes or validating ticket data.
	EnrichTicket(scope *envelope.Scope, matchTicket Ticket, ruleSet interface{}) (ticket Ticket, err error)
}

// TicketProvider provides a mechanism for a match function to get tickets from the match pool it's trying to make matches for
type TicketProvider interface {
	// GetTickets returns a channel that provides regular matchmaking tickets from the pool.
	// I think we'd like to be able to query this, but not yet sure what that looks like
	GetTickets() chan Ticket

	// GetBackfillTickets returns a channel that provides tickets for backfill scenarios.
	// These are tickets that can be used to fill existing matches that need more players.
	GetBackfillTickets() chan BackfillTicket
}

// Matchmaker defines the high-level interface for matchmaking operations.
// This interface handles both player matching and session management.
type Matchmaker interface {
	// MatchPlayers attempts to match players into new game sessions based on the provided requests.
	// Returns matched results, satisfied tickets, and any error that occurred.
	MatchPlayers(rootScope *envelope.Scope, namespace string, matchPool string, matchmakingRequests []models.MatchmakingRequest, channel models.Channel) (matchmakingResult []*models.MatchmakingResult, satisfiedTickets []models.MatchmakingRequest, err error)

	// MatchSessions attempts to add additional players to existing game sessions.
	// Returns updated sessions, satisfied sessions, satisfied tickets, and any error.
	MatchSessions(rootScope *envelope.Scope, namespace string, matchPool string, tickets []models.MatchmakingRequest, sessions []*models.MatchmakingResult, channel models.Channel) ([]*models.MatchmakingResult, []*models.MatchmakingResult, []models.MatchmakingRequest, error)
}
