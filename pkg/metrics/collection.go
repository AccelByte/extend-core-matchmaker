// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type prometheusMetrics struct {
	partyRegionInMatchQueue  prometheus.GaugeVec
	matchPlayersElapsedTime  prometheus.HistogramVec
	matchSessionsElapsedTime prometheus.HistogramVec
	unmatchedReasons         prometheus.CounterVec
}

func setupPrometheusMetrics(registry *prometheus.Registry) prometheusMetrics {
	factory := promauto.With(registry)
	partyLabelDimensions := []string{"game_namespace", "matchpool", "numPlayers"}

	partyRegionInMatchQueue := factory.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ab_mmv2_party_region_in_match_queue",
			Help: "A histogram of numbers of parties with num players per region in the match queue",
		}, append(partyLabelDimensions, "region"))

	//nolint:promlinter
	matchPlayersElapsedTime := factory.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ab_mmv2_match_players_elapsed_time_ms",
			Help:    "A histogram of match players functions elapsed time in milliseconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		}, []string{"game_namespace", "matchpool", "function"})
	//nolint:promlinter
	matchSessionsElapsedTime := factory.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ab_mmv2_match_sessions_elapsed_time_ms",
			Help:    "A histogram of match sessions functions elapsed time in milliseconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		}, []string{"game_namespace", "matchpool", "function"})
	//nolint:promlinter
	unmatchedReasons := factory.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ab_mmv2_unmatched_reasons",
			Help: "A histogram for unmatched matchmaking request reasons",
		}, []string{"game_namespace", "matchpool", "reason"})

	return prometheusMetrics{
		partyRegionInMatchQueue:  *partyRegionInMatchQueue,
		matchPlayersElapsedTime:  *matchPlayersElapsedTime,
		matchSessionsElapsedTime: *matchSessionsElapsedTime,
		unmatchedReasons:         *unmatchedReasons,
	}
}

func (metrics prometheusMetrics) PartiesRegionInMatchQueue(namespace string, matchPool string, region string, numPlayers int, numParties int) {
	metrics.partyRegionInMatchQueue.With(prometheus.Labels{"game_namespace": namespace, "matchpool": matchPool, "region": region, "numPlayers": strconv.Itoa(numPlayers)}).Set(float64(numParties))
}

func (metrics prometheusMetrics) AddMatchPlayersElapsedTimeMs(namespace, matchPool, function string, elapsedTime time.Duration) {
	metrics.matchPlayersElapsedTime.With(prometheus.Labels{"game_namespace": namespace, "matchpool": matchPool, "function": function}).Observe(float64(elapsedTime.Milliseconds()))
}

func (metrics prometheusMetrics) AddMatchSessionsElapsedTimeMs(namespace, matchPool, function string, elapsedTime time.Duration) {
	metrics.matchSessionsElapsedTime.With(prometheus.Labels{"game_namespace": namespace, "matchpool": matchPool, "function": function}).Observe(float64(elapsedTime.Milliseconds()))
}

func (metrics prometheusMetrics) AddUnmatchedReason(namespace string, matchPool string, reason string) {
	metrics.unmatchedReasons.With(prometheus.Labels{"game_namespace": namespace, "matchpool": matchPool, "reason": reason}).Add(float64(1))
}
