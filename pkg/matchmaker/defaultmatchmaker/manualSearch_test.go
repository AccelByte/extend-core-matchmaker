// Copyright (c) 2023-2024 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"reflect"
	"testing"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"github.com/stretchr/testify/assert"
)

func Test_matchByPartyAttribute(t *testing.T) {
	type args struct {
		ticket                          models.MatchmakingRequest
		partyAttributes                 []partyAttribute
		matchOptionsReferredForBackfill bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			// previously this test will panic
			name: "test compare with slice of interface",
			args: args{
				ticket: models.MatchmakingRequest{
					PartyAttributes: map[string]interface{}{
						"map": []interface{}{"dessert"},
					},
				},
				partyAttributes: []partyAttribute{
					{
						key:   "map",
						value: []interface{}{"dessert"},
					},
				},
			},
			want: true,
		},
		{
			// previously this test will panic
			name: "test compare with slice of string",
			args: args{
				ticket: models.MatchmakingRequest{
					PartyAttributes: map[string]interface{}{
						"map": []string{"dessert"},
					},
				},
				partyAttributes: []partyAttribute{
					{
						key:   "map",
						value: []string{"dessert"},
					},
				},
			},
			want: true,
		},
		{
			name: "test compare string match",
			args: args{
				ticket: models.MatchmakingRequest{
					PartyAttributes: map[string]interface{}{
						"map": "dessert",
					},
				},
				partyAttributes: []partyAttribute{
					{
						key:   "map",
						value: "dessert",
					},
				},
			},
			want: true,
		},
		{
			name: "test compare string not match",
			args: args{
				ticket: models.MatchmakingRequest{
					PartyAttributes: map[string]interface{}{
						"map": "dessert",
					},
				},
				partyAttributes: []partyAttribute{
					{
						key:   "map",
						value: "not-dessert",
					},
				},
			},
			want: false,
		},
		{
			name: "test compare string not match with matchOptionsReferredForBackfill is true",
			args: args{
				ticket: models.MatchmakingRequest{
					PartyAttributes: map[string]interface{}{
						"map": "dessert",
					},
				},
				partyAttributes: []partyAttribute{
					{
						key:   "map",
						value: "not-dessert",
					},
				},
				matchOptionsReferredForBackfill: true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchByPartyAttribute(&tt.args.ticket, tt.args.partyAttributes, tt.args.matchOptionsReferredForBackfill); got != tt.want {
				t.Errorf("matchByPartyAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sortMatchTickets(t *testing.T) {
	timeNow := time.Now()

	type args struct {
		matchTickets  []matchTicket
		sessionRegion string
	}
	tests := []struct {
		name string
		args args
		want []matchTicket
	}{
		{
			name: "test 1 - order priority DESC then score ASC",
			args: args{
				matchTickets: []matchTicket{
					{ticket: models.MatchmakingRequest{Priority: 0}, score: 0},
					{ticket: models.MatchmakingRequest{Priority: 0}, score: 10},
					{ticket: models.MatchmakingRequest{Priority: 0}, score: 0},
					{ticket: models.MatchmakingRequest{Priority: 2}, score: 0},
					{ticket: models.MatchmakingRequest{Priority: 2}, score: 10},
				},
			},
			want: []matchTicket{
				{ticket: models.MatchmakingRequest{Priority: 2}, score: 0},
				{ticket: models.MatchmakingRequest{Priority: 2}, score: 10},
				{ticket: models.MatchmakingRequest{Priority: 0}, score: 0},
				{ticket: models.MatchmakingRequest{Priority: 0}, score: 0},
				{ticket: models.MatchmakingRequest{Priority: 0}, score: 10},
			},
		},
		{
			name: "test 2 - cek createdAt ASC",
			args: args{
				matchTickets: []matchTicket{
					{ticket: models.MatchmakingRequest{PartyID: "a", Priority: 2, CreatedAt: timeNow.Unix()}, score: 10},
					{ticket: models.MatchmakingRequest{PartyID: "b", Priority: 2, CreatedAt: timeNow.Add(-1 * time.Second).Unix()}, score: 10},
					{ticket: models.MatchmakingRequest{PartyID: "c", Priority: 2, CreatedAt: timeNow.Add(-5 * time.Second).Unix()}, score: 10},
				},
			},
			want: []matchTicket{
				{ticket: models.MatchmakingRequest{PartyID: "c", Priority: 2, CreatedAt: timeNow.Add(-5 * time.Second).Unix()}, score: 10},
				{ticket: models.MatchmakingRequest{PartyID: "b", Priority: 2, CreatedAt: timeNow.Add(-1 * time.Second).Unix()}, score: 10},
				{ticket: models.MatchmakingRequest{PartyID: "a", Priority: 2, CreatedAt: timeNow.Unix()}, score: 10},
			},
		},
		{
			name: "test 3 - check priority, then score, then createdAt",
			args: args{
				matchTickets: []matchTicket{
					{ticket: models.MatchmakingRequest{PartyID: "A", Priority: 2, CreatedAt: timeNow.Unix()}, score: 20},
					{ticket: models.MatchmakingRequest{PartyID: "B", Priority: 0, CreatedAt: timeNow.Unix()}, score: 10},
					{ticket: models.MatchmakingRequest{PartyID: "C", Priority: 0, CreatedAt: timeNow.Unix()}, score: 0},
					{ticket: models.MatchmakingRequest{PartyID: "D", Priority: 2, CreatedAt: timeNow.Add(-1 * time.Second).Unix()}, score: 10},
					{ticket: models.MatchmakingRequest{PartyID: "E", Priority: 2, CreatedAt: timeNow.Add(-5 * time.Second).Unix()}, score: 20},
				},
			},
			want: []matchTicket{
				{ticket: models.MatchmakingRequest{PartyID: "D", Priority: 2, CreatedAt: timeNow.Add(-1 * time.Second).Unix()}, score: 10},
				{ticket: models.MatchmakingRequest{PartyID: "E", Priority: 2, CreatedAt: timeNow.Add(-5 * time.Second).Unix()}, score: 20},
				{ticket: models.MatchmakingRequest{PartyID: "A", Priority: 2, CreatedAt: timeNow.Unix()}, score: 20},
				{ticket: models.MatchmakingRequest{PartyID: "C", Priority: 0, CreatedAt: timeNow.Unix()}, score: 0},
				{ticket: models.MatchmakingRequest{PartyID: "B", Priority: 0, CreatedAt: timeNow.Unix()}, score: 10},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortMatchTickets(tt.args.matchTickets, tt.args.sessionRegion)
			if got := tt.args.matchTickets; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortMatchTickets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchByRegionLatency(t *testing.T) {
	t.Parallel()
	channel := &models.Channel{
		Ruleset: models.RuleSet{RegionExpansionRateMs: 1000, RegionExpansionRangeMs: 50},
	}
	testCases := []struct {
		Name             string
		PivotCreatedAt   time.Time
		PivotRegion      *models.Region
		CandidateRegions []models.Region
		CandidateStep    int
		DisableBirection int
		ExpectedMatched  bool
		ExpectedScore    float64
	}{
		{
			Name: "1", PivotCreatedAt: time.Now(),
			PivotRegion: &models.Region{Region: "us-east-1", Latency: 20},
			CandidateRegions: []models.Region{
				{Region: "us-east-1", Latency: 100},
				{Region: "us-east-2", Latency: 120},
			}, CandidateStep: 0, ExpectedMatched: true, ExpectedScore: 0,
		},
		{
			Name: "2", PivotCreatedAt: time.Now(),
			PivotRegion: &models.Region{Region: "us-east-2", Latency: 20},
			CandidateRegions: []models.Region{
				{Region: "us-east-1", Latency: 100},
				{Region: "us-east-2", Latency: 120},
			}, CandidateStep: 0, ExpectedMatched: false, ExpectedScore: 0,
		},
		{
			Name: "3", PivotCreatedAt: time.Now(),
			PivotRegion: &models.Region{Region: "us-east-2", Latency: 20},
			CandidateRegions: []models.Region{
				{Region: "us-east-1", Latency: 100},
				{Region: "us-east-2", Latency: 170},
			}, CandidateStep: 1, ExpectedMatched: false, ExpectedScore: 0,
		},
		{
			Name: "4", PivotCreatedAt: time.Now(),
			PivotRegion: &models.Region{Region: "us-east-2", Latency: 20},
			CandidateRegions: []models.Region{
				{Region: "us-east-1", Latency: 100},
				{Region: "us-east-2", Latency: 170},
			}, CandidateStep: 2, ExpectedMatched: true, ExpectedScore: 70,
		},
		// matched because disable filter candidate
		{
			Name: "5", PivotCreatedAt: time.Now().Add(-2 * time.Second),
			PivotRegion: &models.Region{Region: "us-east-2", Latency: 20},
			CandidateRegions: []models.Region{
				{Region: "us-east-1", Latency: 100},
				{Region: "us-east-2", Latency: 120},
			}, CandidateStep: 0, ExpectedMatched: true, ExpectedScore: 20, DisableBirection: 1000,
		},
		{
			Name: "6", PivotCreatedAt: time.Now().Add(-2 * time.Second),
			PivotRegion: &models.Region{Region: "us-east-2", Latency: 20},
			CandidateRegions: []models.Region{
				{Region: "us-east-1", Latency: 100},
				{Region: "us-east-2", Latency: 120},
			}, CandidateStep: 0, ExpectedMatched: false, ExpectedScore: 20, DisableBirection: 5000,
		},
	}
	for _, tc := range testCases {
		expansionDuration := time.Duration(channel.Ruleset.RegionExpansionRateMs) * time.Millisecond
		ticket := &models.MatchmakingRequest{CreatedAt: time.Now().Add(-time.Duration(tc.CandidateStep) * expansionDuration).Unix(),
			SortedLatency: tc.CandidateRegions}
		channel.Ruleset.DisableBidirectionalLatencyAfterMs = tc.DisableBirection
		skipFilteredCandidate := skipFilterCandidateRegion(&models.MatchmakingRequest{CreatedAt: tc.PivotCreatedAt.Unix()}, channel)
		matched, score := matchByRegionLatency(ticket, tc.PivotRegion, channel, skipFilteredCandidate)
		assert.Equal(t, tc.ExpectedMatched, matched, tc.Name)
		if matched {
			assert.Equal(t, tc.ExpectedScore, score, tc.Name)
		}
	}
}

func TestMatchByDistance(t *testing.T) {
	var reference float64 = 100
	testCases := []struct {
		TicketMmr       float64
		Distances       []distance
		ExpectedMatched bool
		ExpectedScore   float64
	}{
		{
			Distances: []distance{{attribute: "mmr", value: 20, min: 20 - reference, max: 20 + reference}},
			TicketMmr: 50, ExpectedMatched: true, ExpectedScore: 30,
		},
		{
			Distances: []distance{{attribute: "mmr", value: 20, min: 20 - reference, max: 20 + reference}},
			TicketMmr: 125, ExpectedMatched: false, ExpectedScore: 0,
		},
		{
			Distances: []distance{{attribute: "mmr", value: 20, min: 20 - reference, max: 20 + reference, attributeMaxValue: 500}},
			TicketMmr: 50, ExpectedMatched: true, ExpectedScore: float64(30) / float64(500),
		},
	}
	for _, tc := range testCases {
		ticket := &models.MatchmakingRequest{
			PartyAttributes: map[string]interface{}{memberAttributesKey: map[string]any{"mmr": tc.TicketMmr}},
		}
		matched, score := matchByDistance(ticket, &models.RuleSet{}, tc.Distances)
		assert.Equal(t, tc.ExpectedMatched, matched)
		if matched {
			assert.Equal(t, tc.ExpectedScore, score)
		}
	}
}

func TestMatchByDistanceWithBiDirectionFlexing(t *testing.T) {
	t.Parallel()
	ruleSet := models.RuleSet{
		MatchingRule: []models.MatchingRule{
			{Attribute: "mmr", Criteria: "distance", Reference: 100},
		},
		FlexingRule: []models.FlexingRule{
			{Duration: 10, MatchingRule: models.MatchingRule{
				Attribute: "mmr", Criteria: "distance", Reference: 200,
			}},
		},
	}
	testCases := []struct {
		Name               string
		PivotMmr           float64
		PivotCreatedAt     time.Time
		CandidateMmr       float64
		CandidateCreatedAt time.Time
		ExpectedMatched    bool
	}{
		{
			Name:     "Simple matched",
			PivotMmr: 50, PivotCreatedAt: time.Now(), CandidateMmr: 60, CandidateCreatedAt: time.Now(), ExpectedMatched: true,
		},
		{
			Name:     "Simple not matched",
			PivotMmr: 50, PivotCreatedAt: time.Now(), CandidateMmr: 200, CandidateCreatedAt: time.Now(), ExpectedMatched: false,
		},
		{
			Name:     "Flexed not matched because candidate not flexed",
			PivotMmr: 50, PivotCreatedAt: time.Now().Add(-20 * time.Second), CandidateMmr: 200, CandidateCreatedAt: time.Now(), ExpectedMatched: false,
		},
		{
			Name:     "Both pivot and cadidate flexed so it got matched",
			PivotMmr: 50, PivotCreatedAt: time.Now().Add(-20 * time.Second), CandidateMmr: 200, CandidateCreatedAt: time.Now().Add(-20 * time.Second), ExpectedMatched: true,
		},
	}
	for _, tc := range testCases {
		pivot := &models.MatchmakingRequest{
			PartyAttributes: map[string]interface{}{memberAttributesKey: map[string]any{"mmr": tc.PivotMmr}},
		}
		candidate := &models.MatchmakingRequest{
			CreatedAt:       tc.CandidateCreatedAt.Unix(),
			PartyAttributes: map[string]interface{}{memberAttributesKey: map[string]any{"mmr": tc.CandidateMmr}},
		}
		activeRuleSet, _ := applyRuleFlexing(ruleSet, tc.PivotCreatedAt)
		distances := getFilterByDistance(&activeRuleSet, pivot.PartyAttributes)
		matched, _ := matchByDistance(candidate, &ruleSet, distances)
		assert.Equal(t, tc.ExpectedMatched, matched)
	}
}
