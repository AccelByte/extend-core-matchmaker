package testsetup

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/metrics"
)

type stubMetricsCollection struct{}

func (s stubMetricsCollection) PartiesRegionInMatchQueue(namespace string, matchPool string, region string, numPlayers int, numParties int) {
}

func (s stubMetricsCollection) AddMatchPlayersElapsedTimeMs(namespace, matchPool, function string, elapsedTime time.Duration) {
}

func (s stubMetricsCollection) AddMatchSessionsElapsedTimeMs(namespace, matchPool, function string, elapsedTime time.Duration) {
}

func (s stubMetricsCollection) AddUnmatchedReason(namespace string, matchPool string, reason string) {
}

func NewMetrics() metrics.MatchmakingMetrics {
	return stubMetricsCollection{}
}
