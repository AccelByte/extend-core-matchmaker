// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

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
	BackfillMatches(scope *envelope.Scope, ticketProvider TicketProvider, matchRules interface{}) <-chan BackfillProposal
	MakeMatches(scope *envelope.Scope, ticketProvider TicketProvider, matchRules interface{}) <-chan Match
	RulesFromJSON(scope *envelope.Scope, json string) (interface{}, error)
	GetStatCodes(scope *envelope.Scope, matchRules interface{}) []string
	ValidateTicket(scope *envelope.Scope, matchTicket Ticket, matchRules interface{}) (bool, error)
	EnrichTicket(scope *envelope.Scope, matchTicket Ticket, ruleSet interface{}) (ticket Ticket, err error)
}

// TicketProvider provides a mechanism for a match function to get tickets from the match pool it's trying to make matches for
type TicketProvider interface {
	GetTickets() chan Ticket // I think we'd like to be able to query this, but not yet sure what that looks like
	GetBackfillTickets() chan BackfillTicket
}

type Matchmaker interface {
	MatchPlayers(rootScope *envelope.Scope, namespace string, matchPool string, matchmakingRequests []models.MatchmakingRequest, channel models.Channel) (matchmakingResult []*models.MatchmakingResult, satisfiedTickets []models.MatchmakingRequest, err error)
	MatchSessions(rootScope *envelope.Scope, namespace string, matchPool string, tickets []models.MatchmakingRequest, sessions []*models.MatchmakingResult, channel models.Channel) ([]*models.MatchmakingResult, []*models.MatchmakingResult, []models.MatchmakingRequest, error)
}
