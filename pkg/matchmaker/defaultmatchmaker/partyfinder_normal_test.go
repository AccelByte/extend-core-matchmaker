// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"github.com/stretchr/testify/assert"
)

func Test_normal_findParty(t *testing.T) {
	type args struct {
		Tickets     []models.MatchmakingRequest
		PivotTicket models.MatchmakingRequest
		MinPlayer   int
		MaxPlayer   int
		Current     []models.MatchmakingRequest
	}
	type testCase struct {
		name string
		args args
		want []models.MatchmakingRequest
	}

	var (
		tests   = []testCase{}
		tickets []models.MatchmakingRequest
	)

	// case 1
	tickets = generateRequest("", 4, 2)
	tests = append(tests, testCase{
		name: "normal | min 2 max 5 | party of 2",
		args: args{
			Tickets:     tickets,
			PivotTicket: tickets[0],
			MinPlayer:   2,
			MaxPlayer:   5,
			Current:     nil,
		},
		want: tickets[:2],
	})

	// case 2
	tickets = generateRequest("", 10, 1)
	tests = append(tests, testCase{
		name: "normal | min 2 max 5 | party of 1",
		args: args{
			Tickets:     tickets,
			PivotTicket: tickets[0],
			MinPlayer:   2,
			MaxPlayer:   5,
			Current:     nil,
		},
		want: tickets[:5],
	})

	// case 3
	tickets = generateRequest("", 5, 3)
	tests = append(tests, testCase{
		name: "normal | min 2 max 5 | party of 3",
		args: args{
			Tickets:     tickets,
			PivotTicket: tickets[0],
			MinPlayer:   2,
			MaxPlayer:   5,
			Current:     nil,
		},
		want: tickets[:1],
	})

	// case 4
	tickets = generateRequestWithMemberCount("", []int{3, 3, 4, 1, 2})
	tests = append(tests, testCase{
		name: "normal | min 2 max 5 | party of random",
		args: args{
			Tickets:     tickets,
			PivotTicket: tickets[0],
			MinPlayer:   2,
			MaxPlayer:   5,
			Current:     nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[3]},
	})

	// case 5
	tickets = generateRequestWithMemberCount("", []int{1, 2, 1, 5})
	tests = append(tests, testCase{
		name: "normal | min 2 max 5 | last party is 5",
		args: args{
			Tickets:     tickets,
			PivotTicket: tickets[0],
			MinPlayer:   2,
			MaxPlayer:   5,
			Current:     nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[1], tickets[2]},
	})

	// case 6
	tickets = generateRequestWithMemberCount("", []int{1, 2, 5, 2})
	tests = append(tests, testCase{
		name: "normal | min 2 max 5 | check first 3 party | case 1",
		args: args{
			Tickets:     tickets,
			PivotTicket: tickets[0],
			MinPlayer:   2,
			MaxPlayer:   5,
			Current:     nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[1], tickets[3]},
	})

	// case 7
	tickets = generateRequestWithMemberCount("", []int{1, 2, 2, 5})
	tests = append(tests, testCase{
		name: "normal | min 2 max 5 | check first 3 party | case 2",
		args: args{
			Tickets:     tickets,
			PivotTicket: tickets[0],
			MinPlayer:   2,
			MaxPlayer:   5,
			Current:     nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[1], tickets[2]},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindPartyCombination(nil, tt.args.Tickets, tt.args.PivotTicket, tt.args.MinPlayer, tt.args.MaxPlayer, false, nil, tt.args.Current, "")
			if !assert.ElementsMatch(t, got, tt.want) {
				t.Errorf("normal.findParty() = %v, want %v", got, tt.want)
			}
		})
	}
}
