// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
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

func Test_countPlayers(t *testing.T) {
	type args struct {
		mmReq []models.MatchmakingRequest
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "count players",
			args: args{mmReq: []models.MatchmakingRequest{
				{PartyMembers: []models.PartyMember{
					{UserID: "A"},
					{UserID: "B"},
				}},
				{PartyMembers: []models.PartyMember{
					{UserID: "C"},
				}},
			}},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countPlayers(tt.args.mmReq); got != tt.want {
				t.Errorf("countPlayers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getPivotTicketIndexFromTickets(t *testing.T) {
	type args struct {
		tickets     []models.MatchmakingRequest
		pivotTicket models.MatchmakingRequest
	}
	type test struct {
		name string
		args args
		want int
	}

	a := generateRequest("", 1, 1)[0]
	b := generateRequest("", 1, 1)[0]
	c := generateRequest("", 1, 1)[0]
	d := generateRequest("", 1, 1)[0]

	var tests []test

	/* Test Cases */

	tests = append(tests, test{
		name: "pivot ticket is a",
		args: args{
			tickets:     []models.MatchmakingRequest{a, b, c, d},
			pivotTicket: a,
		},
		want: 0,
	})

	tests = append(tests, test{
		name: "pivot ticket is c",
		args: args{
			tickets:     []models.MatchmakingRequest{a, b, c, d},
			pivotTicket: c,
		},
		want: 2,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPivotTicketIndexFromTickets(tt.args.tickets, &tt.args.pivotTicket); got != tt.want {
				t.Errorf("getPivotTicketIndexFromTickets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_reorderTickets(t *testing.T) {
	type test struct {
		name       string
		tickets    []models.MatchmakingRequest
		newIndexes []int
		want       []models.MatchmakingRequest
	}
	var tests []test
	a := generateRequest("", 1, 1)[0]
	b := generateRequest("", 1, 1)[0]
	c := generateRequest("", 1, 1)[0]
	d := generateRequest("", 1, 1)[0]
	tests = append(tests, test{
		name:       "test 1",
		tickets:    []models.MatchmakingRequest{a, b, c, d},
		newIndexes: []int{3, 0, 2, 1, -1, 10}, // -1 and 10 is exceed the ticket size, it just not being proceed
		want:       []models.MatchmakingRequest{d, a, c, b},
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotReorderedTickets := reorderTickets(tt.tickets, tt.newIndexes); !reflect.DeepEqual(gotReorderedTickets, tt.want) {
				t.Errorf("reorderTickets() = %v, want %v", gotReorderedTickets, tt.want)
			}
		})
	}
}

func Test_sortDESC(t *testing.T) {
	type args struct {
		matchmakingRequests []models.MatchmakingRequest
	}
	tests := []struct {
		name string
		args args
		want []models.MatchmakingRequest
	}{
		{
			name: "same player number",
			args: args{
				matchmakingRequests: []models.MatchmakingRequest{
					{
						PartyID:      "partyA",
						PartyMembers: []models.PartyMember{{UserID: "userA"}},
					},
					{
						PartyID:      "partyB",
						PartyMembers: []models.PartyMember{{UserID: "userB"}},
					},
					{
						PartyID:      "partyC",
						PartyMembers: []models.PartyMember{{UserID: "userC"}},
					},
				},
			},
			want: []models.MatchmakingRequest{
				{
					PartyID:      "partyA",
					PartyMembers: []models.PartyMember{{UserID: "userA"}},
				},
				{
					PartyID:      "partyB",
					PartyMembers: []models.PartyMember{{UserID: "userB"}},
				},
				{
					PartyID:      "partyC",
					PartyMembers: []models.PartyMember{{UserID: "userC"}},
				},
			},
		},
		{
			name: "different player number",
			args: args{
				matchmakingRequests: []models.MatchmakingRequest{
					{
						PartyID:      "partyA",
						PartyMembers: []models.PartyMember{{UserID: "userA"}},
					},
					{
						PartyID: "partyB",
						PartyMembers: []models.PartyMember{
							{UserID: "userB1"},
							{UserID: "userB2"},
						},
					},
					{
						PartyID: "partyC",
						PartyMembers: []models.PartyMember{
							{UserID: "userC1"},
							{UserID: "userC2"},
							{UserID: "userC3"},
						},
					},
				},
			},
			want: []models.MatchmakingRequest{
				{
					PartyID: "partyC",
					PartyMembers: []models.PartyMember{
						{UserID: "userC1"},
						{UserID: "userC2"},
						{UserID: "userC3"},
					},
				},
				{
					PartyID: "partyB",
					PartyMembers: []models.PartyMember{
						{UserID: "userB1"},
						{UserID: "userB2"},
					},
				},
				{
					PartyID:      "partyA",
					PartyMembers: []models.PartyMember{{UserID: "userA"}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortDESC(tt.args.matchmakingRequests)
			got := tt.args.matchmakingRequests
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortDESC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getTicketRegionExpansionStep(t *testing.T) {
	type args struct {
		ticket  models.MatchmakingRequest
		channel *models.Channel
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "region expansion",
			args: args{
				ticket: models.MatchmakingRequest{
					CreatedAt: time.Now().Add(-time.Duration(6) * time.Second).UTC().Unix(),
				},
				channel: &models.Channel{
					Ruleset: models.RuleSet{
						RegionExpansionRateMs: 5000,
					},
				},
			},
			want: 2,
		},
		{
			name: "non region expansion",
			args: args{
				ticket: models.MatchmakingRequest{
					PartyAttributes: map[string]interface{}{
						models.AttributeMatchAttempt: 1,
					},
				},
				channel: &models.Channel{},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTicketRegionExpansionStep(&tt.args.ticket, tt.args.channel); got != tt.want {
				t.Errorf("getTicketRegionExpansionStep() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getTicketMaxLatency(t *testing.T) {
	type args struct {
		ticket  models.MatchmakingRequest
		channel *models.Channel
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "test region expansion",
			args: args{
				ticket: models.MatchmakingRequest{
					CreatedAt: time.Now().Add(-time.Duration(6) * time.Second).UTC().Unix(),
				},
				channel: &models.Channel{
					Ruleset: models.RuleSet{
						RegionExpansionRateMs:       5000,
						RegionLatencyInitialRangeMs: 50,
						RegionExpansionRangeMs:      50,
						RegionLatencyMaxMs:          200,
					},
				},
			},
			want: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTicketMaxLatency(&tt.args.ticket, tt.args.channel); got != tt.want {
				t.Errorf("getTicketMaxLatency() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterRegionStep(t *testing.T) {
	testCases := []struct {
		Name           string
		Channel        *models.Channel
		Step           int
		Region         []models.Region
		ExpectedRegion []models.Region
	}{
		{Name: "1",
			Channel: &models.Channel{
				Ruleset: models.RuleSet{RegionExpansionRateMs: 1000, RegionExpansionRangeMs: 50},
			}, Step: 0, Region: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
			}, ExpectedRegion: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
			},
		},
		{Name: "2",
			Channel: &models.Channel{
				Ruleset: models.RuleSet{RegionExpansionRateMs: 1000, RegionExpansionRangeMs: 50},
			}, Step: 1, Region: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
				{Region: "ap-east-2", Latency: 150},
			}, ExpectedRegion: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
			},
		},
		{Name: "3",
			Channel: &models.Channel{
				Ruleset: models.RuleSet{RegionExpansionRateMs: 1000, RegionExpansionRangeMs: 50},
			}, Step: 2, Region: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
				{Region: "ap-east-2", Latency: 170},
			}, ExpectedRegion: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
			},
		},
		{Name: "4",
			Channel: &models.Channel{
				Ruleset: models.RuleSet{RegionExpansionRateMs: 1000, RegionExpansionRangeMs: 50},
			}, Step: 3, Region: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
				{Region: "ap-east-2", Latency: 170},
			}, ExpectedRegion: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
				{Region: "ap-east-2", Latency: 170},
			},
		},
		{Name: "5",
			Channel: &models.Channel{
				Ruleset: models.RuleSet{RegionExpansionRateMs: 1000, RegionExpansionRangeMs: 50, RegionLatencyMaxMs: 100},
			}, Step: 3, Region: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
				{Region: "ap-east-2", Latency: 170},
			}, ExpectedRegion: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
			},
		},
		{Name: "6",
			Channel: &models.Channel{
				Ruleset: models.RuleSet{RegionExpansionRateMs: 1000, RegionExpansionRangeMs: 1000, RegionLatencyMaxMs: 0},
			}, Step: 3, Region: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
				{Region: "ap-east-2", Latency: 1900},
			}, ExpectedRegion: []models.Region{
				{Region: "us-east-1", Latency: 50},
				{Region: "us-east-2", Latency: 50},
				{Region: "us-west-2", Latency: 100},
				{Region: "ap-east-2", Latency: 1900},
			},
		},
	}
	for _, tc := range testCases {
		expansionDuration := time.Duration(tc.Channel.Ruleset.RegionExpansionRateMs) * time.Millisecond
		ticket := &models.MatchmakingRequest{CreatedAt: time.Now().Add(-time.Duration(tc.Step) * expansionDuration).Unix(),
			SortedLatency: tc.Region}
		result := filterRegionByStep(ticket, tc.Channel)
		assert.Equal(t, tc.ExpectedRegion, result, tc.Name)
	}
}
