// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"github.com/stretchr/testify/assert"
)

func Test_unique_findParty(t *testing.T) {
	type args struct {
		Tickets        []models.MatchmakingRequest
		PivotTicket    models.MatchmakingRequest
		MinPlayer      int
		MaxPlayer      int
		HasCombination bool
		Current        []models.MatchmakingRequest
	}
	type testCase struct {
		name           string
		args           args
		want           []models.MatchmakingRequest
		wantPartyRoles [][]string
	}

	var (
		tests   = []testCase{}
		tickets []models.MatchmakingRequest
	)

	// case 1
	tickets = generateRequestWithMemberRoles("", [][]string{{"a", "b"}, {"b"}, {"a", "c"}, {"c"}})
	tests = append(tests, testCase{
		name: "role | all unique | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      3,
			MaxPlayer:      5,
			HasCombination: true,
			Current:        nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[3]},
		wantPartyRoles: [][]string{{"a", "b"}, {"c"}},
	})

	// case 2
	tickets = generateRequestWithMemberRoles("", [][]string{{"a", "b"}, {"b"}, {"a", "c"}, {"c"}})
	tests = append(tests, testCase{
		name: "role | all unique | party of random | failed",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      4,
			MaxPlayer:      5,
			HasCombination: true,
			Current:        nil,
		},
		want:           nil,
		wantPartyRoles: nil,
	})

	// case 3
	tickets = generateRequestWithMemberRoles("", [][]string{{"a", "b"}, {"b"}, {"a", "c"}, {"c"}, {"c", "d"}})
	tests = append(tests, testCase{
		name: "role | all unique | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      3,
			MaxPlayer:      5,
			HasCombination: true,
			Current:        nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[4]},
		wantPartyRoles: [][]string{{"a", "b"}, {"c", "d"}},
	})

	// case 4
	tickets = generateRequestWithMemberRoles("", [][]string{{`["a","b"]`, "b"}, {`["b","e"]`}, {"a", "c"}, {"c"}, {"c", "d"}})
	tests = append(tests, testCase{
		name: "role | all unique | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      3,
			MaxPlayer:      5,
			HasCombination: true,
			Current:        nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[1], tickets[4]},
		wantPartyRoles: [][]string{{"a", "b"}, {"e"}, {"c", "d"}},
	})

	// case 5
	tickets = generateRequestWithMemberRoles("", [][]string{{`["a","b"]`, "b"}, {`["c","e"]`}, {"a", "c"}, {"c"}, {"c", "d"}})
	tests = append(tests, testCase{
		name: "role | all unique | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      3,
			MaxPlayer:      5,
			HasCombination: true,
			Current:        nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[1], tickets[4]},
		wantPartyRoles: [][]string{{"a", "b"}, {"e"}, {"c", "d"}},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindPartyCombination(nil, tt.args.Tickets, tt.args.PivotTicket, tt.args.MinPlayer, tt.args.MaxPlayer, tt.args.HasCombination, nil, nil, "")

			gotPartyIDs := make([]string, 0)
			gotPartyRoles := make([][]string, 0)
			for _, request := range got {
				gotPartyIDs = append(gotPartyIDs, request.PartyID)
				roles := make([]string, 0)
				for _, member := range request.PartyMembers {
					var role string
					// output should only have 1 role value for each member
					if len(member.GetRole()) == 1 {
						role = member.GetRole()[0]
					}
					roles = append(roles, role)
				}
				gotPartyRoles = append(gotPartyRoles, roles)
			}

			wantPartyIDs := make([]string, 0)
			for _, request := range tt.want {
				wantPartyIDs = append(wantPartyIDs, request.PartyID)
			}
			wantPartyRoles := tt.wantPartyRoles

			if !assert.ElementsMatch(t, gotPartyIDs, wantPartyIDs) {
				t.Errorf("combo.findParty_Any() party ids = %v, want %v", gotPartyIDs, wantPartyIDs)
			}
			if !assert.ElementsMatch(t, gotPartyRoles, wantPartyRoles) {
				t.Errorf("combo.findParty_Any() party roles = %v, want %v", gotPartyRoles, wantPartyRoles)
			}
		})
	}
}
