// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type MatchmakingMetrics interface {
	PartiesRegionInMatchQueue(namespace string, matchPool string, region string, numPlayers int, numParties int)
	AddMatchPlayersElapsedTimeMs(namespace, matchPool, function string, elapsedTime time.Duration)
	AddMatchSessionsElapsedTimeMs(namespace, matchPool, function string, elapsedTime time.Duration)
	AddUnmatchedReason(namespace string, matchPool string, reason string)
}

func NewMetrics(registry *prometheus.Registry) MatchmakingMetrics {
	return setupPrometheusMetrics(registry)
}
