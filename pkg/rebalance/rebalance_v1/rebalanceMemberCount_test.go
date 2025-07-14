// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v1

import (
	"context"
	"reflect"
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

func TestRebalanceMemberCount(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceMemberCount", "")
	defer scope.Finish()

	type args struct {
		allies             []models.MatchingAlly
		activeAllianceRule models.AllianceRule
	}
	tests := []struct {
		name          string
		args          args
		wantNewAllies []models.MatchingAlly
	}{
		{
			name: "normal | balance already | no changes",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("a", 10)}},
						{PartyMembers: []models.PartyMember{player("b", 8)}},
						{PartyMembers: []models.PartyMember{player("c", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("d", 7)}},
						{PartyMembers: []models.PartyMember{player("e", 8)}},
						{PartyMembers: []models.PartyMember{player("f", 9)}},
					}},
				},
				activeAllianceRule: mockChannel.Ruleset.AllianceRule,
			},
			wantNewAllies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("c", 9)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("d", 7)}},
					{PartyMembers: []models.PartyMember{player("e", 8)}},
					{PartyMembers: []models.PartyMember{player("f", 9)}},
				}},
			},
		}, {
			name: "normal | difference only 1 | no changes",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("a", 10)}},
						{PartyMembers: []models.PartyMember{player("b", 8)}},
						{PartyMembers: []models.PartyMember{player("c", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("d", 7)}},
						{PartyMembers: []models.PartyMember{player("e", 8)}},
						{PartyMembers: []models.PartyMember{player("f", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("g", 7)}},
						{PartyMembers: []models.PartyMember{player("h", 8)}},
					}},
				},
				activeAllianceRule: mockChannel.Ruleset.AllianceRule,
			},
			wantNewAllies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("c", 9)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("d", 7)}},
					{PartyMembers: []models.PartyMember{player("e", 8)}},
					{PartyMembers: []models.PartyMember{player("f", 9)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("g", 7)}},
					{PartyMembers: []models.PartyMember{player("h", 8)}},
				}},
			},
		}, {
			name: "normal | party member more than 1 | rebalanced",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							player("a", 0),
							player("b", 0),
							player("c", 0),
						}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							player("d", 0),
							player("e", 0),
						}},
						{PartyMembers: []models.PartyMember{
							player("f", 0),
						}},
					}},
					{MatchingParties: []models.MatchingParty{}},
				},
				activeAllianceRule: models.AllianceRule{
					MinNumber:       3,
					MaxNumber:       3,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 3,
				},
			},
			wantNewAllies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{
						player("a", 0),
						player("b", 0),
						player("c", 0),
					}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{
						player("d", 0),
						player("e", 0),
					}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{
						player("f", 0),
					}},
				}},
			},
		}, {
			name: "normal | 3 x 3 x 1 | rebalance into 3 x 2 x 2",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("a", 10)}},
						{PartyMembers: []models.PartyMember{player("b", 8)}},
						{PartyMembers: []models.PartyMember{player("c", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("d", 7)}},
						{PartyMembers: []models.PartyMember{player("e", 8)}},
						{PartyMembers: []models.PartyMember{player("f", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("g", 7)}},
					}},
				},
				activeAllianceRule: mockChannel.Ruleset.AllianceRule,
			},
			wantNewAllies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("d", 7)}},
					{PartyMembers: []models.PartyMember{player("g", 7)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("e", 8)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("c", 9)}},
					{PartyMembers: []models.PartyMember{player("f", 9)}},
				}},
			},
		}, {
			name: "normal | partyof2 x 3 x 1 | rebalance into partyof2 x 2 x 2",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("a", 10), player("b", 8), player("c", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("d", 7)}},
						{PartyMembers: []models.PartyMember{player("e", 8)}},
						{PartyMembers: []models.PartyMember{player("f", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("g", 7)}},
					}},
				},
				activeAllianceRule: mockChannel.Ruleset.AllianceRule,
			},
			wantNewAllies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10), player("b", 8), player("c", 9)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("d", 7)}},
					{PartyMembers: []models.PartyMember{player("f", 9)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("e", 8)}},
					{PartyMembers: []models.PartyMember{player("g", 7)}},
				}},
			},
		}, {
			name: "role based asymmetry | not rebalanced | no changes",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("a", 0)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("b", 0), player("c", 0), player("d", 0)}},
						{PartyMembers: []models.PartyMember{player("e", 0)}},
					}},
				},
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 4,
					Combination: models.Combination{
						HasCombination: true,
						Alliances: [][]models.Role{
							{
								{Name: "monster", Min: 1, Max: 1},
							},
							{
								{Name: "hunter", Min: 1, Max: 2},
								{Name: "villager", Min: 1, Max: 3},
							},
						},
					},
				},
			},
			wantNewAllies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 0)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("b", 0), player("c", 0), player("d", 0)}},
					{PartyMembers: []models.PartyMember{player("e", 0)}},
				}},
			},
		}, {
			name: "role based single combo | need to consider the role | rebalanced",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "carry"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "carry"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "e", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{{UserID: "f", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "g", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "h", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					}},
				},
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 5,
					Combination: models.Combination{
						HasCombination: true,
						Alliances: [][]models.Role{
							{
								{Name: "carry", Min: 1, Max: 3},
								{Name: "support", Min: 1, Max: 3},
								{Name: "mage", Min: 0, Max: 2},
							},
						},
					},
				},
			},
			wantNewAllies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "carry"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "e", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "g", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "carry"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "f", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "h", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
				}},
			},
		}, {
			// this case is complex to be resolved in this function,
			// we need to implement horizontal filling instead,
			// which will be happened in upper level (findMatchingAlly) to distribute member bettween each ally
			name: "role based single combo | need to consider the role | not rebalanced because of order",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "carry"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "carry"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "e", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{{UserID: "f", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "g", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
						{PartyMembers: []models.PartyMember{{UserID: "h", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					}},
				},
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 5,
					Combination: models.Combination{
						HasCombination: true,
						Alliances: [][]models.Role{
							{
								{Name: "carry", Min: 1, Max: 3},
								{Name: "support", Min: 1, Max: 3},
								{Name: "mage", Min: 0, Max: 2},
							},
						},
					},
				},
			},
			wantNewAllies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{{UserID: "a", ExtraAttributes: map[string]interface{}{models.ROLE: "carry"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "b", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "c", ExtraAttributes: map[string]interface{}{models.ROLE: "carry"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "d", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "e", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{{UserID: "f", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "g", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
					{PartyMembers: []models.PartyMember{{UserID: "h", ExtraAttributes: map[string]interface{}{models.ROLE: "support"}}}},
				}},
			},
		}, {
			name: "role based multi combo symmetry | ",
			args: args{
				allies:             []models.MatchingAlly{},
				activeAllianceRule: models.AllianceRule{},
			},
			wantNewAllies: []models.MatchingAlly{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotNewAllies := RebalanceMemberCount(scope, tt.args.allies, tt.args.activeAllianceRule, ""); !reflect.DeepEqual(gotNewAllies, tt.wantNewAllies) {
				t.Errorf("RebalanceMemberCount() = %v, want %v", gotNewAllies, tt.wantNewAllies)
			}
		})
	}
}

func TestRebalanceMemberCountNew(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceMemberCountNew", "")
	defer scope.Finish()

	type args struct {
		allies             []models.MatchingAlly
		activeAllianceRule models.AllianceRule
	}
	tests := []struct {
		name          string
		args          args
		wantNewAllies []models.MatchingAlly
	}{
		{
			name: "all parties locked | no changes",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						party("a", true, 1, nil),
						party("b", true, 1, nil),
						party("c", true, 1, nil),
					}},
					{MatchingParties: []models.MatchingParty{
						party("d", true, 1, nil),
					}},
				},
				activeAllianceRule: mockChannel.Ruleset.AllianceRule,
			},
			wantNewAllies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					party("a", true, 1, nil),
					party("b", true, 1, nil),
					party("c", true, 1, nil),
				}},
				{MatchingParties: []models.MatchingParty{
					party("d", true, 1, nil),
				}},
			},
		}, {
			name: "all parties not locked | rebalanced",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						party("a", false, 1, nil),
						party("b", false, 1, nil),
						party("c", false, 1, nil),
					}},
					{MatchingParties: []models.MatchingParty{
						party("d", false, 1, nil),
					}},
				},
				activeAllianceRule: mockChannel.Ruleset.AllianceRule,
			},
			wantNewAllies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					party("a", false, 1, nil),
					party("c", false, 1, nil),
				}},
				{MatchingParties: []models.MatchingParty{
					party("b", false, 1, nil),
					party("d", false, 1, nil),
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotNewAllies := RebalanceMemberCount(scope, tt.args.allies, tt.args.activeAllianceRule, ""); !reflect.DeepEqual(gotNewAllies, tt.wantNewAllies) {
				t.Errorf("RebalanceMemberCount() = %v, want %v", gotNewAllies, tt.wantNewAllies)
			}
		})
	}
}
