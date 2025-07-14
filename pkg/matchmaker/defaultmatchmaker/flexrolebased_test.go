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

func Test_applyRoleBasedFlexing(t *testing.T) {
	now := time.Now()
	type args struct {
		matchmakingRequests []models.MatchmakingRequest
		channel             models.Channel
	}
	tests := []struct {
		name string
		args args
		want []models.MatchmakingRequest
	}{
		{
			name: "1 minute 1 player role change to any | firstCreatedAt = 0 minute before | changed: 0",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							Combination: models.Combination{
								Alliances: [][]models.Role{
									{
										{Name: "fighter", Min: 1, Max: 2},
										{Name: "tank", Min: 1, Max: 2},
									},
								},
								HasCombination:    true,
								RoleFlexingEnable: true,
								RoleFlexingSecond: 60,
								RoleFlexingPlayer: 1,
							},
						},
					},
				},
				matchmakingRequests: []models.MatchmakingRequest{
					{
						PartyID: "party1",
						PartyMembers: []models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Unix(),
					},
					{
						PartyID: "party2",
						PartyMembers: []models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Unix(),
					},
				},
			},
			want: []models.MatchmakingRequest{
				{
					PartyID: "party1",
					PartyMembers: []models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
					},
					CreatedAt: now.Unix(),
				},
				{
					PartyID: "party2",
					PartyMembers: []models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
					},
					CreatedAt: now.Unix(),
				},
			},
		},
		{
			name: "1 minute 1 player role change to any | firstCreatedAt = 1 minute before | changed: 1",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							Combination: models.Combination{
								Alliances: [][]models.Role{
									{
										{Name: "fighter", Min: 1, Max: 2},
										{Name: "tank", Min: 1, Max: 2},
									},
								},
								HasCombination:    true,
								RoleFlexingEnable: true,
								RoleFlexingSecond: 60,
								RoleFlexingPlayer: 1,
							},
						},
					},
				},
				matchmakingRequests: []models.MatchmakingRequest{
					{
						PartyID: "party1",
						PartyMembers: []models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Add(time.Minute * -1).Unix(),
					},
					{
						PartyID: "party2",
						PartyMembers: []models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Add(time.Minute * -1).Unix(),
					},
				},
			},
			want: []models.MatchmakingRequest{
				{
					PartyID: "party1",
					PartyMembers: []models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "any"}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
					},
					CreatedAt: now.Add(time.Minute * -1).Unix(),
				},
				{
					PartyID: "party2",
					PartyMembers: []models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
					},
					CreatedAt: now.Add(time.Minute * -1).Unix(),
				},
			},
		},
		{
			name: "1 minute 1 player role change to any | firstCreatedAt = 2 minute before | changed: 2",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							Combination: models.Combination{
								Alliances: [][]models.Role{
									{
										{Name: "fighter", Min: 1, Max: 2},
										{Name: "tank", Min: 1, Max: 2},
									},
								},
								HasCombination:    true,
								RoleFlexingEnable: true,
								RoleFlexingSecond: 60,
								RoleFlexingPlayer: 1,
							},
						},
					},
				},
				matchmakingRequests: []models.MatchmakingRequest{
					{
						PartyMembers: []models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Add(time.Minute * -2).Unix(),
					},
					{
						PartyMembers: []models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Add(time.Minute * -2).Unix(),
					},
				},
			},
			want: []models.MatchmakingRequest{
				{
					PartyMembers: []models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "any"}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "any"}},
					},
					CreatedAt: now.Add(time.Minute * -2).Unix(),
				},
				{
					PartyMembers: []models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
					},
					CreatedAt: now.Add(time.Minute * -2).Unix(),
				},
			},
		},
		{
			name: "1 minute 1 player role change to any | party1.firstCreatedAt = 2 minute before | party2.firstCreatedAt = 1 minute before | changed: 2",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							Combination: models.Combination{
								Alliances: [][]models.Role{
									{
										{Name: "fighter", Min: 1, Max: 2},
										{Name: "tank", Min: 1, Max: 2},
									},
								},
								HasCombination:    true,
								RoleFlexingEnable: true,
								RoleFlexingSecond: 60,
								RoleFlexingPlayer: 1,
							},
						},
					},
				},
				matchmakingRequests: []models.MatchmakingRequest{
					{
						PartyID: "party1",
						PartyMembers: []models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Add(time.Minute * -2).Unix(),
					},
					{
						PartyID: "party2",
						PartyMembers: []models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Add(time.Minute * -1).Unix(),
					},
				},
			},
			want: []models.MatchmakingRequest{
				{
					PartyID: "party1",
					PartyMembers: []models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
					},
					CreatedAt: now.Add(time.Minute * -2).Unix(),
				},
				{
					PartyID: "party2",
					PartyMembers: []models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "any"}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "any"}},
					},
					CreatedAt: now.Add(time.Minute * -1).Unix(),
				},
			},
		},
		{
			name: "1 minute 1 player role change to any | party1.firstCreatedAt = 2 minute before | party2.firstCreatedAt = now | changed: 2 in party1 only",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							Combination: models.Combination{
								Alliances: [][]models.Role{
									{
										{Name: "fighter", Min: 1, Max: 2},
										{Name: "tank", Min: 1, Max: 2},
									},
								},
								HasCombination:    true,
								RoleFlexingEnable: true,
								RoleFlexingSecond: 60,
								RoleFlexingPlayer: 1,
							},
						},
					},
				},
				matchmakingRequests: []models.MatchmakingRequest{
					{
						PartyID: "party1",
						PartyMembers: []models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Add(time.Minute * -2).Unix(),
					},
					{
						PartyID: "party2",
						PartyMembers: []models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Unix(),
					},
				},
			},
			want: []models.MatchmakingRequest{
				{
					PartyID: "party1",
					PartyMembers: []models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
					},
					CreatedAt: now.Add(time.Minute * -2).Unix(),
				},
				{
					PartyID: "party2",
					PartyMembers: []models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "any"}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "any"}},
					},
					CreatedAt: now.Unix(),
				},
			},
		},
		{
			name: "using sub game mode | changed: 2",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						SubGameModes: map[string]models.SubGameMode{
							"sgm": {
								AllianceRule: models.AllianceRule{
									Combination: models.Combination{
										Alliances: [][]models.Role{
											{
												{Name: "fighter", Min: 1, Max: 2},
												{Name: "tank", Min: 1, Max: 2},
											},
										},
										HasCombination:    true,
										RoleFlexingEnable: true,
										RoleFlexingSecond: 60,
										RoleFlexingPlayer: 1,
									},
								},
							},
						},
					},
				},
				matchmakingRequests: []models.MatchmakingRequest{
					{
						PartyID: "party1",
						PartyMembers: []models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Add(time.Minute * -2).Unix(),
						PartyAttributes: map[string]interface{}{
							models.AttributeSubGameMode: "sgm",
						},
					},
					{
						PartyID: "party2",
						PartyMembers: []models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						},
						CreatedAt: now.Add(time.Minute * -1).Unix(),
						PartyAttributes: map[string]interface{}{
							models.AttributeSubGameMode: "sgm",
						},
					},
				},
			},
			want: []models.MatchmakingRequest{
				{
					PartyID: "party1",
					PartyMembers: []models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "fighter"}},
					},
					CreatedAt: now.Add(time.Minute * -2).Unix(),
					PartyAttributes: map[string]interface{}{
						models.AttributeSubGameMode: "sgm",
					},
				},
				{
					PartyID: "party2",
					PartyMembers: []models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "any"}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "any"}},
					},
					CreatedAt: now.Add(time.Minute * -1).Unix(),
					PartyAttributes: map[string]interface{}{
						models.AttributeSubGameMode: "sgm",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyRoleBasedFlexing(tt.args.matchmakingRequests, &tt.args.channel)
			assert.ElementsMatch(t, tt.args.matchmakingRequests, tt.want, "matchmaking requests not match with expected")
		})
	}
}

func Test_getNumPlayersForRole(t *testing.T) {
	type args struct {
		requests []models.MatchmakingRequest
	}
	tests := []struct {
		name string
		args args
		want map[string]int
	}{
		{
			name: "test",
			args: args{
				requests: []models.MatchmakingRequest{
					{PartyMembers: []models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: []string{"carry", "support"}}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "carry"}},
					}},
					{PartyMembers: []models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: []string{"carry", "support"}}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: models.AnyRole}},
					}},
				},
			},
			want: map[string]int{
				"carry":   3,
				"support": 2,
				"any":     1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNumPlayersForRole(tt.args.requests); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getNumPlayersForRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getNumMinimumForRole(t *testing.T) {
	type args struct {
		combination models.Combination
	}
	tests := []struct {
		name string
		args args
		want map[string]int
	}{
		{
			name: "test",
			args: args{
				combination: models.Combination{
					Alliances: [][]models.Role{
						{
							{Name: "monster", Min: 1, Max: 2},
						},
						{
							{Name: "monster", Min: 1, Max: 2},
							{Name: "hunter", Min: 1, Max: 2},
						},
					},
				},
			},
			want: map[string]int{
				"monster": 2,
				"hunter":  1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNumMinimumForRole(tt.args.combination); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getNumMinimumForRole() = %v, want %v", got, tt.want)
			}
		})
	}
}
