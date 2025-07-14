// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"github.com/stretchr/testify/assert"
)

func Test_combo_findParty(t *testing.T) {
	type args struct {
		Tickets        []models.MatchmakingRequest
		PivotTicket    models.MatchmakingRequest
		MinPlayer      int
		MaxPlayer      int
		HasCombination bool
		Roles          []models.Role
		Current        []models.MatchmakingRequest
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
	tickets = generateRequestWithMemberRoles("", [][]string{{"dps", "support"}, {"dps"}, {"tank", "dps"}, {"support"}, {"support", "tank"}})
	tests = append(tests, testCase{
		name: "role | 1 combo | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      3,
			MaxPlayer:      6,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "dps",
					Min:  1,
					Max:  2,
				},
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  2,
				},
			},
			Current: nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[2], tickets[4]},
	})

	// case 2
	tickets = generateRequestWithMemberRoles("", [][]string{{"dps", "support"}, {"dps"}, {"tank", "dps"}, {"support"}, {"support", "tank"}})
	tests = append(tests, testCase{
		name: "role | min max 1 | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      3,
			MaxPlayer:      3,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "dps",
					Min:  1,
					Max:  1,
				},
				{
					Name: "tank",
					Min:  1,
					Max:  1,
				},
				{
					Name: "support",
					Min:  1,
					Max:  1,
				},
			},
			Current: nil,
		},
		want: nil, // because first ticket is pivot, it must be in the match, but no other ticket contains only tank
	})

	// case 3
	tickets = generateRequestWithMemberRoles("", [][]string{{"dps", "support"}, {"dps"}, {"tank", "support"}, {"support"}, {"support", "tank"}})
	tests = append(tests, testCase{
		name: "role | 1 combo | party of random | failed",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      6,
			MaxPlayer:      6,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "dps",
					Min:  2,
					Max:  2,
				},
				{
					Name: "tank",
					Min:  2,
					Max:  2,
				},
				{
					Name: "support",
					Min:  2,
					Max:  2,
				},
			},
			Current: nil,
		},
		want: nil,
	})

	// case 5
	tickets = generateRequestWithMemberRoles("", [][]string{{"hunter"}, {"monster"}, {"hunter", "hunter"}, {"monster"}, {"hunter", "hunter"}})
	tests = append(tests, testCase{
		name: "role | combo | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      2,
			MaxPlayer:      4,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "hunter",
					Min:  2,
					Max:  4,
				},
			},
			Current: nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[2]},
	})

	// case 6
	tickets = generateRequestWithMemberRoles("", [][]string{
		{"hunter"}, {"monster"}, {"hunter", "hunter"}, {"monster"}, {"hunter", "hunter"},
	})
	tests = append(tests, testCase{
		name: "role | combo | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      2,
			MaxPlayer:      4,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "hunter",
					Min:  2,
					Max:  4,
				},
			},
			Current: nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[2]},
	})

	// case 7
	tickets = generateRequestWithMemberRoles("", [][]string{
		{"hunter"}, {"monster"}, {"hunter", "hunter"}, {"monster"}, {"hunter", "hunter"},
	})
	tests = append(tests, testCase{
		name: "role | combo | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      2,
			MaxPlayer:      6,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "monster",
					Min:  1,
					Max:  2,
				},
				{
					Name: "hunter",
					Min:  1,
					Max:  4,
				},
			},
			Current: nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[1], tickets[2], tickets[3]},
	})

	// case 8
	tickets = generateRequestWithMemberRoles("", [][]string{{"tank"}, {"dps"}, {"tank", "dps"}, {"tank", "support"}, {"support"}})
	tests = append(tests, testCase{
		name: "role | combo | party of random | find as much allies",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      2,
			MaxPlayer:      6,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  4,
				},
			},
			Current: nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[3], tickets[4]},
	})

	// case 10
	tickets = generateRequestWithMemberRoles("", [][]string{
		{"tank"}, {"dps", "tank"}, {"dps"}, {"tank", "support"}, {"support"}, {"dps", "support"}, {"support"}, {"support"},
	})
	tests = append(tests, testCase{
		name: "role | combo | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      2,
			MaxPlayer:      6,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  4,
				},
			},
			Current: nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[3], tickets[4], tickets[6], tickets[7]},
	})

	// case 11
	tickets = generateRequestWithMemberRoles("", [][]string{
		{"fighter"}, {"marksman"}, {"support"}, {"fighter"}, {"support"}, {"marksman"}, {"support"}, {"tank"},
	})
	tests = append(tests, testCase{
		name: "role | combo | 1 member per party",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      5,
			MaxPlayer:      5,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "fighter",
					Min:  1,
					Max:  2,
				},
				{
					Name: "marksman",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  2,
				},
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
			},
			Current: nil,
		},
		want: []models.MatchmakingRequest{tickets[0], tickets[1], tickets[2], tickets[3], tickets[7]},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindPartyCombination(nil, tt.args.Tickets, tt.args.PivotTicket, tt.args.MinPlayer, tt.args.MaxPlayer, tt.args.HasCombination, tt.args.Roles, nil, "")
			if !assert.ElementsMatch(t, got, tt.want) {
				t.Errorf("combo.findParty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_combo_findParty_Any_And_Secondary(t *testing.T) {
	t.Parallel()

	type args struct {
		Tickets        []models.MatchmakingRequest
		PivotTicket    models.MatchmakingRequest
		MinPlayer      int
		MaxPlayer      int
		HasCombination bool
		Roles          []models.Role
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
	tickets = generateRequestWithMemberRoles("", [][]string{
		{"tank"}, {"support", "support"}, {"support"}, {"any", "tank"},
	})
	tests = append(tests, testCase{
		name: "role | combo | contains any",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      2,
			MaxPlayer:      6,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  4,
				},
			},
			Current: nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[1], tickets[2], tickets[3]},
		wantPartyRoles: [][]string{{"tank"}, {"support", "support"}, {"support"}, {"support", "tank"}},
	})

	// case 2
	tickets = generateRequestWithMemberRoles("", [][]string{{"any"}, {"monster"}, {"hunter", "any"}, {"monster"}, {"hunter", "hunter"}})
	tests = append(tests, testCase{
		name: "role | combo | with any | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      2,
			MaxPlayer:      4,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "hunter",
					Min:  2,
					Max:  4,
				},
			},
			Current: nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[2]},
		wantPartyRoles: [][]string{{"hunter"}, {"hunter", "hunter"}},
	})

	// case 3
	tickets = generateRequestWithMemberRoles("", [][]string{{"dps", "support"}, {"support", "support"}, {"dps"}, {"any", "any"}, {"support"}, {"support", "tank"}})
	tests = append(tests, testCase{
		name: "role | combo | with any | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      3,
			MaxPlayer:      6,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "dps",
					Min:  1,
					Max:  2,
				},
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  2,
				},
			},
			Current: nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[2], tickets[3], tickets[4]},
		wantPartyRoles: [][]string{{"dps", "support"}, {"dps"}, {"tank", "tank"}, {"support"}},
	})

	// case 4
	tickets = generateRequestWithMemberRoles("", [][]string{
		{`["fighter","tank"]`}, {`["tank","marksman"]`}, {`["support","any"]`}, {"fighter"}, {`["support","tank"]`}, {"marksman"}, {"support"}, {"tank"},
	})
	tests = append(tests, testCase{
		name: "role | combo | contains any and secondary",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      5,
			MaxPlayer:      5,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "fighter",
					Min:  1,
					Max:  2,
				},
				{
					Name: "marksman",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  2,
				},
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
			},
			Current: nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[1], tickets[2], tickets[3], tickets[4]},
		wantPartyRoles: [][]string{{"fighter"}, {"marksman"}, {"support"}, {"fighter"}, {"tank"}},
	})

	// case 5
	tickets = generateRequestWithMemberRoles("", [][]string{{"any"}, {"monster"}, {"hunter", "any"}, {"monster"}, {"hunter", "hunter"}})
	tests = append(tests, testCase{
		name: "role | combo | with any | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      2,
			MaxPlayer:      4,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "hunter",
					Min:  2,
					Max:  4,
				},
			},
			Current: nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[2]},
		wantPartyRoles: [][]string{{"hunter"}, {"hunter", "hunter"}},
	})

	// case 6
	tickets = generateRequestWithMemberRoles("", [][]string{{"dps", "support"}, {"support", "support"}, {"dps"}, {"any", "any"}, {"support"}, {"support", "tank"}})
	tests = append(tests, testCase{
		name: "role | combo | with any | party of random",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      3,
			MaxPlayer:      6,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "dps",
					Min:  1,
					Max:  2,
				},
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  2,
				},
			},
			Current: nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[2], tickets[3], tickets[4]},
		wantPartyRoles: [][]string{{"dps", "support"}, {"dps"}, {"tank", "tank"}, {"support"}},
	})

	// case 7
	tickets = generateRequestWithMemberRoles("", [][]string{
		{"fighter"}, {"marksman"}, {"support"}, {"fighter"}, {`["support","tank"]`}, {"marksman"}, {"support"}, {"tank"},
	})
	tests = append(tests, testCase{
		name: "role | combo | 1 member per party",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      5,
			MaxPlayer:      5,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "fighter",
					Min:  1,
					Max:  2,
				},
				{
					Name: "marksman",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  2,
				},
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
			},
			Current: nil,
		},
		want:           []models.MatchmakingRequest{tickets[0], tickets[1], tickets[2], tickets[3], tickets[4]},
		wantPartyRoles: [][]string{{"fighter"}, {"marksman"}, {"support"}, {"fighter"}, {"tank"}},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindPartyCombination(nil, tt.args.Tickets, tt.args.PivotTicket, tt.args.MinPlayer, tt.args.MaxPlayer, tt.args.HasCombination, tt.args.Roles, nil, "")

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

func Test_combo_findParty_Priority(t *testing.T) {
	type args struct {
		Tickets        []models.MatchmakingRequest
		PivotTicket    models.MatchmakingRequest
		MinPlayer      int
		MaxPlayer      int
		HasCombination bool
		Roles          []models.Role
		Current        []models.MatchmakingRequest
	}
	type testCase struct {
		name           string
		args           args
		wantPartyRoles [][]string
	}

	var (
		tests   = []testCase{}
		tickets []models.MatchmakingRequest
	)

	// case 1
	tickets = generateRequestWithMemberRoles("", [][]string{
		{"support", "support"}, // pivot ticket
		{"support"},
		{"support"},
	})
	tickets = append(tickets, generateRequestWithMemberRolesPriority("", [][]string{
		{"any", "tank"},
	}, 1)...)
	tests = append(tests, testCase{
		name: "role | combo | priority contains any",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      2,
			MaxPlayer:      6,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  4,
				},
			},
			Current: nil,
		},
		wantPartyRoles: [][]string{{"support", "support"}, {"tank", "tank"}, {"support"}, {"support"}},
	})

	// case 2
	tickets = generateRequestWithMemberRoles("", [][]string{
		{"tank"}, {"support", "support"}, {"support"}, {"any", "tank"},
	})
	tickets = append(tickets, generateRequestWithMemberRolesPriority("", [][]string{
		{"any", "tank"},
	}, 1)...)
	tests = append(tests, testCase{
		name: "role | combo | all tickets contains any",
		args: args{
			Tickets:        tickets,
			PivotTicket:    tickets[0],
			MinPlayer:      2,
			MaxPlayer:      6,
			HasCombination: true,
			Roles: []models.Role{
				{
					Name: "tank",
					Min:  1,
					Max:  2,
				},
				{
					Name: "support",
					Min:  1,
					Max:  4,
				},
			},
			Current: nil,
		},
		wantPartyRoles: [][]string{{"tank"}, {"support", "tank"}, {"support", "support"}, {"support"}},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindPartyCombination(nil, tt.args.Tickets, tt.args.PivotTicket, tt.args.MinPlayer, tt.args.MaxPlayer, tt.args.HasCombination, tt.args.Roles, nil, "")
			gotPartyRoles := make([][]string, 0)
			for _, request := range got {
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
			if !assert.ElementsMatch(t, gotPartyRoles, tt.wantPartyRoles) {
				t.Errorf("combo.findParty_Priority() = %v, want %v", gotPartyRoles, tt.wantPartyRoles)
			}
		})
	}
}

func Test_combo_getAvailableRole(t *testing.T) {
	roles := []models.Role{
		{Name: "fighter", Min: 1, Max: 3},
		{Name: "tank", Min: 1, Max: 3},
		{Name: "support", Min: 1, Max: 3},
	}
	type fields struct {
		roles     []models.Role
		countRole map[string]int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "every role reach max | no available role",
			fields: fields{
				roles: roles,
				countRole: map[string]int{
					"fighter": 3,
					"tank":    3,
					"support": 3,
				},
			},
			want: "",
		},
		{
			name: "tank as available role",
			fields: fields{
				roles: roles,
				countRole: map[string]int{
					"fighter": 3,
					"tank":    1,
					"support": 3,
				},
			},
			want: "tank",
		},
		{
			name: "smallest count as available role",
			fields: fields{
				roles: roles,
				countRole: map[string]int{
					"fighter": 2,
					"tank":    2,
					"support": 1,
				},
			},
			want: "support",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAvailableRole(tt.fields.roles, tt.fields.countRole); got != tt.want {
				t.Errorf("combo.getAvailableRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_combo_getAvailableRole_ConsiderMinFirst(t *testing.T) {
	type fields struct {
		roles     []models.Role
		countRole map[string]int
	}
	type testCase struct {
		name   string
		fields fields
		want   string
	}

	var tests []testCase

	// case 1
	tests = append(tests, testCase{
		name: "roles have different min | assign support first",
		fields: fields{
			roles: []models.Role{
				{Name: "midlaner", Min: 0, Max: 2},
				{Name: "support", Min: 2, Max: 2},
			},
			countRole: map[string]int{},
		},
		want: "support",
	})

	// case 2
	tests = append(tests, testCase{
		name: "roles have different min | support already have 1 | assign jungler",
		fields: fields{
			roles: []models.Role{
				{Name: "midlaner", Min: 0, Max: 2},
				{Name: "support", Min: 2, Max: 2},
				{Name: "jungler", Min: 1, Max: 2},
			},
			countRole: map[string]int{
				"support": 1,
			},
		},
		want: "jungler",
	})

	// case 3
	tests = append(tests, testCase{
		name: "roles have different min | assign to support again",
		fields: fields{
			roles: []models.Role{
				{Name: "midlaner", Min: 0, Max: 2},
				{Name: "support", Min: 2, Max: 2},
			},
			countRole: map[string]int{
				"support": 1,
			},
		},
		want: "support",
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getAvailableRole(tt.fields.roles, tt.fields.countRole); got != tt.want {
				t.Errorf("combo.getAvailableRole() = %v, want %v", got, tt.want)
			}
		})
	}
}
