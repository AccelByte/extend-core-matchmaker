// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package constants

import "time"

const (
	PoolLockTimeLimit = 10 * time.Second
)

const (
	AttrMMR          = "mmr"
	DistanceCriteria = "distance"
)

const (
	MatchSessionFunction = "matchSessions"
	MatchPlayersFunction = "matchPlayers"

	// Unbackfill reason constants.
	UnbackfillReasonAutoBackfillIsFalse              = "unbackfill_auto_backfill_is_false"
	UnbackfillReasonAutoBackfillIsFalseOrMatchIsFull = "unbackfill_auto_backfill_is_false_or_match_is_full"
	UnbackfillReasonRegionIsNotMatch                 = "unbackfill_region_is_not_match"
	UnbackfillReasonLatencyIsNotInRange              = "unbackfill_latency_is_not_in_range"
	UnbackfillReasonBlockedPlayerExist               = "unbackfill_blocked_player_exist"
	UnbackfillReasonMatchOptionNotSatisfied          = "unbackfill_match_option_not_satisfied"
	UnbackfillReasonSubGameModeNotSatisfied          = "unbackfill_sub_game_mode_not_satisfied"
	UnbackfillReasonSessionIsFull                    = "unbackfill_session_is_full"
	UnbackfillReasonBackfillProposalRejected         = "unbackfill_backfill_proposal_rejected"
	UnbackfillReasonBackfillProposalRejectedAndStop  = "unbackfill_backfill_proposal_rejected_and_should_stop"
	UnbackfillReasonBackfillProposalExpired          = "unbackfill_backfill_proposal_expired"
	UnbackfillReasonBackfillTicketExpired            = "unbackfill_backfill_ticket_expired"
	UnbackfillReasonBackfillProposalCanceled         = "unbackfill_backfill_proposal_canceled"
	UnbackfillReasonAttributeDistanceNotSatisfied    = "unbackfill_attribute_distance_not_satisfied"
	UnbackfillReasonServerNameNotSatisfied           = "unbackfill_server_name_not_satisfied"
	UnbackfillReasonClientVersionNotSatisfied        = "unbackfill_client_version_not_satisfied"
	UnbackfillReasonSessionExcluded                  = "unbackfill_session_excluded"

	// not matched reason constants.
	ReasonNotEnoughRequests  = "not_enough_requests"
	ReasonNotEnoughPlayers   = "not_enough_players"
	ReasonNoMatchableTickets = "no_matchable_tickets"
)
