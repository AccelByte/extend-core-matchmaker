// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v1

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
)

var mockChannel = models.Channel{
	Ruleset: models.RuleSet{
		AllianceRule: models.AllianceRule{
			MinNumber:       1,
			MaxNumber:       3,
			PlayerMinNumber: 1,
			PlayerMaxNumber: 5,
		},
		MatchingRule: []models.MatchingRule{
			{Attribute: "anyDistanceAttribute", Criteria: constants.DistanceCriteria},
		},
	},
}

func player(userID string, mmr float64) models.PartyMember {
	return models.PartyMember{UserID: userID, ExtraAttributes: map[string]interface{}{"anyDistanceAttribute": mmr}}
}

func party(partyID string, locked bool, memberCount int, mmrs []float64) models.MatchingParty {
	var partyMembers []models.PartyMember
	for i := 0; i < memberCount; i++ {
		partyMember := models.PartyMember{UserID: fmt.Sprintf("user_%d", (i + 1))}
		if i < len(mmrs) {
			partyMember.ExtraAttributes = map[string]interface{}{"anyDistanceAttribute": mmrs[i]}
		}
		partyMembers = append(partyMembers, partyMember)
	}
	return models.MatchingParty{
		PartyID:      partyID,
		PartyMembers: partyMembers,
		Locked:       locked,
	}
}

func TestRebalance(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalance", "")
	defer scope.Finish()

	type args struct {
		allies []models.MatchingAlly
	}
	tests := []struct {
		name               string
		args               args
		want               []models.MatchingAlly
		wantBetterDistance bool
	}{
		{
			name: "2_allies",
			args: args{allies: []models.MatchingAlly{
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
			}},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("d", 7)}},
					{PartyMembers: []models.PartyMember{player("c", 9)}},
					{PartyMembers: []models.PartyMember{player("a", 10)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("f", 9)}},
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("e", 8)}},
				}},
			},
			wantBetterDistance: true,
		},
		{
			name: "2_allies_1st_ally_has_same_party",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("a", 10), player("c", 9)}},
						{PartyMembers: []models.PartyMember{player("b", 8)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("d", 7)}},
						{PartyMembers: []models.PartyMember{player("e", 8)}},
						{PartyMembers: []models.PartyMember{player("f", 9)}},
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10), player("c", 9)}},
					{PartyMembers: []models.PartyMember{player("d", 7)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("e", 8)}},
					{PartyMembers: []models.PartyMember{player("f", 9)}},
				}},
			},
			wantBetterDistance: true,
		},
		{
			name: "2_allies_2nd_ally_has_same_party",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("a", 10)}},
						{PartyMembers: []models.PartyMember{player("b", 8)}},
						{PartyMembers: []models.PartyMember{player("c", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("d", 7), player("e", 8)}},
						{PartyMembers: []models.PartyMember{player("f", 9)}},
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("d", 7), player("e", 8)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("c", 9)}},
					{PartyMembers: []models.PartyMember{player("f", 9)}},
				}},
			},
			wantBetterDistance: true,
		},
		{
			name: "3_allies",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("a", 10)}},
						{PartyMembers: []models.PartyMember{player("b", 8)}},
						{PartyMembers: []models.PartyMember{player("c", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("d", 7), player("e", 8)}},
						{PartyMembers: []models.PartyMember{player("f", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("i", 2)}},
						{PartyMembers: []models.PartyMember{player("j", 2), player("k", 2)}},
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("d", 7), player("e", 8)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("f", 9)}},
					{PartyMembers: []models.PartyMember{player("j", 2), player("k", 2)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("c", 9)}},
					{PartyMembers: []models.PartyMember{player("i", 2)}},
				}},
			},
			wantBetterDistance: true,
		},
		{
			name: "3_allies_unbalanced_member_has_3_members_in_same_party",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("a", 10), player("b", 8), player("c", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("d", 9)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("e", 2)}},
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10), player("b", 8), player("c", 9)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("d", 9)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("e", 2)}},
				}},
			},
			wantBetterDistance: false, // not better because we cannot separate a,b,and c
		},
		{
			name: "3_allies_9_members_all_in_different_parties",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("a", 3280)}},
						{PartyMembers: []models.PartyMember{player("b", 1020)}},
						{PartyMembers: []models.PartyMember{player("c", 3010)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("d", 1090)}},
						{PartyMembers: []models.PartyMember{player("e", 1230)}},
						{PartyMembers: []models.PartyMember{player("f", 3240)}},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{player("g", 1570)}},
						{PartyMembers: []models.PartyMember{player("h", 1560)}},
						{PartyMembers: []models.PartyMember{player("i", 1550)}},
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 3280)}},
					{PartyMembers: []models.PartyMember{player("b", 1020)}},
					{PartyMembers: []models.PartyMember{player("c", 3010)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("d", 1090)}},
					{PartyMembers: []models.PartyMember{player("e", 1230)}},
					{PartyMembers: []models.PartyMember{player("f", 3240)}},
				}},
				{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("g", 1570)}},
					{PartyMembers: []models.PartyMember{player("h", 1560)}},
					{PartyMembers: []models.PartyMember{player("i", 1550)}},
				}},
			},
			wantBetterDistance: false, // this case gets better distance in rebalance V2
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := countDistance(tt.args.allies, []string{"anyDistanceAttribute"}, nil)

			got := Rebalance(scope, "", tt.args.allies, mockChannel.Ruleset.AllianceRule, mockChannel.Ruleset.MatchingRule)
			for i, gotAlly := range got {
				gotMember := make(map[string]map[string]interface{})
				for _, gotParty := range gotAlly.MatchingParties {
					for _, m := range gotParty.PartyMembers {
						gotMember[m.UserID] = m.ExtraAttributes
					}
				}

				wantMember := make(map[string]map[string]interface{})
				for _, wantParty := range tt.want[i].MatchingParties {
					for _, m := range wantParty.PartyMembers {
						wantMember[m.UserID] = m.ExtraAttributes
					}
				}

				if !reflect.DeepEqual(gotMember, wantMember) {
					t.Errorf("TestRebalance() got = %v, want %v", gotMember, wantMember)
				}
			}

			newDistance := countDistance(got, []string{"anyDistanceAttribute"}, nil)
			gotBetterDistance := newDistance < distance
			if gotBetterDistance != tt.wantBetterDistance {
				t.Errorf("TestRebalance() distance = %v , newDistance %v", distance, newDistance)
			} else {
				t.Logf("got better distance: %f -> %f", distance, newDistance)
			}
		})
	}
}

func TestRebalanceNew(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceNew", "")
	defer scope.Finish()

	type args struct {
		allies []models.MatchingAlly
	}
	tests := []struct {
		name               string
		args               args
		want               []models.MatchingAlly
		wantBetterDistance bool
	}{
		{
			name: "all parties locked | no changes",
			args: args{allies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					party("a", true, 1, []float64{10}),
					party("b", true, 1, []float64{8}),
					party("c", true, 1, []float64{9}),
				}},
				{MatchingParties: []models.MatchingParty{
					party("d", true, 1, []float64{7}),
					party("e", true, 1, []float64{8}),
					party("f", true, 1, []float64{9}),
				}},
			}},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					party("a", true, 1, []float64{10}),
					party("b", true, 1, []float64{8}),
					party("c", true, 1, []float64{9}),
				}},
				{MatchingParties: []models.MatchingParty{
					party("d", true, 1, []float64{7}),
					party("e", true, 1, []float64{8}),
					party("f", true, 1, []float64{9}),
				}},
			},
			wantBetterDistance: false,
		},
		{
			name: "all parties not locked | rebalanced",
			args: args{allies: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					party("a", false, 1, []float64{10}),
					party("b", false, 1, []float64{8}),
					party("c", false, 1, []float64{9}),
				}},
				{MatchingParties: []models.MatchingParty{
					party("d", false, 1, []float64{7}),
					party("e", false, 1, []float64{8}),
					party("f", false, 1, []float64{9}),
				}},
			}},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					party("d", false, 1, []float64{7}),
					party("c", false, 1, []float64{9}),
					party("a", false, 1, []float64{10}),
				}},
				{MatchingParties: []models.MatchingParty{
					party("f", false, 1, []float64{9}),
					party("b", false, 1, []float64{8}),
					party("e", false, 1, []float64{8}),
				}},
			},
			wantBetterDistance: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := countDistance(tt.args.allies, []string{"anyDistanceAttribute"}, nil)

			got := Rebalance(scope, "", tt.args.allies, mockChannel.Ruleset.AllianceRule, mockChannel.Ruleset.MatchingRule)
			for i, ally := range got {
				gotParties := make(map[string]struct{})
				for _, party := range ally.MatchingParties {
					gotParties[party.PartyID] = struct{}{}
				}

				wantParties := make(map[string]struct{})
				for _, party := range tt.want[i].MatchingParties {
					wantParties[party.PartyID] = struct{}{}
				}

				if !reflect.DeepEqual(gotParties, wantParties) {
					t.Errorf("TestRebalance() ally[%d] gotParties = %v, wantParties %v", i, gotParties, wantParties)
				}
			}

			newDistance := countDistance(got, []string{"anyDistanceAttribute"}, nil)
			gotBetterDistance := newDistance < distance
			if gotBetterDistance != tt.wantBetterDistance {
				t.Errorf("TestRebalance() distance = %v , newDistance %v", distance, newDistance)
			}
		})
	}
}

func playerWithRole(userID string, mmr float64, role string) models.PartyMember {
	extraAttributes := map[string]interface{}{
		"anyDistanceAttribute": mmr,
		models.ROLE:            role,
	}
	return models.PartyMember{UserID: userID, ExtraAttributes: extraAttributes}
}

func getMapOfMember(ally models.MatchingAlly) map[string]interface{} {
	maps := make(map[string]interface{})
	for _, party := range ally.MatchingParties {
		for _, member := range party.PartyMembers {
			maps[member.UserID] = member.ExtraAttributes
		}
	}
	return maps
}

func TestSwapper_Swap(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestSwapper_Swap", "")
	defer scope.Finish()

	type fields struct {
		attr    string
		channel models.Channel
	}
	type args struct {
		ally1 models.MatchingAlly
		ally2 models.MatchingAlly
	}
	type want struct {
		newAlly1 models.MatchingAlly
		newAlly2 models.MatchingAlly
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "swap by mmr, the role is same for expected swap member",
			fields: fields{
				attr: "anyDistanceAttribute",
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							PlayerMinNumber: 3,
							PlayerMaxNumber: 3,
						},
					},
				},
			},
			args: args{
				ally1: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("a", 10, "support"),
							playerWithRole("b", 10, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("c", 9, "tank"),
						}},
					},
				},
				ally2: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("d", 9, "support"),
							playerWithRole("e", 9, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("f", 7, "tank"),
						}},
					},
				},
			},
			want: want{
				newAlly1: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("a", 10, "support"),
							playerWithRole("b", 10, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("f", 7, "tank"),
						}},
					},
				},
				newAlly2: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("d", 9, "support"),
							playerWithRole("e", 9, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("c", 9, "tank"),
						}},
					},
				},
			},
		},
		{
			name: "swap by mmr, avg of ally 1 < ally 2",
			fields: fields{
				attr: "anyDistanceAttribute",
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							PlayerMinNumber: 3,
							PlayerMaxNumber: 3,
						},
					},
				},
			},
			args: args{
				ally1: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("a", 7, "support"),
							playerWithRole("b", 7, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("c", 8, "tank"),
						}},
					},
				},
				ally2: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("d", 9, "support"),
							playerWithRole("e", 9, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("f", 10, "tank"),
						}},
					},
				},
			},
			want: want{
				newAlly1: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("a", 7, "support"),
							playerWithRole("b", 7, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("f", 10, "tank"),
						}},
					},
				},
				newAlly2: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("d", 9, "support"),
							playerWithRole("e", 9, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("c", 8, "tank"),
						}},
					},
				},
			},
		},
		{
			name: "no swap, since avg of ally 1 = ally 2",
			fields: fields{
				attr: "anyDistanceAttribute",
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							PlayerMinNumber: 3,
							PlayerMaxNumber: 3,
						},
					},
				},
			},
			args: args{
				ally1: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("a", 7, "support"),
							playerWithRole("b", 7, "fighter"),
						}},
					},
				},
				ally2: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("d", 7, "support"),
							playerWithRole("e", 7, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("f", 7, "tank"),
						}},
					},
				},
			},
			want: want{
				newAlly1: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("a", 7, "support"),
							playerWithRole("b", 7, "fighter"),
						}},
					},
				},
				newAlly2: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("d", 7, "support"),
							playerWithRole("e", 7, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("f", 7, "tank"),
						}},
					},
				},
			},
		},
		{
			name: "no swap since the role is different",
			fields: fields{
				attr: "anyDistanceAttribute",
				channel: models.Channel{
					Ruleset: models.RuleSet{
						AllianceRule: models.AllianceRule{
							PlayerMinNumber: 3,
							PlayerMaxNumber: 3,
							Combination: models.Combination{
								HasCombination: true,
							},
						},
					},
				},
			},
			args: args{
				ally1: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("a", 10, "support"),
							playerWithRole("b", 10, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("c", 9, "tank"),
						}},
					},
				},
				ally2: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("d", 9, "support"),
							playerWithRole("e", 9, "tank"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("f", 7, "fighter"),
						}},
					},
				},
			},
			want: want{
				newAlly1: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("a", 10, "support"),
							playerWithRole("b", 10, "fighter"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("c", 9, "tank"),
						}},
					},
				},
				newAlly2: models.MatchingAlly{
					MatchingParties: []models.MatchingParty{
						{PartyMembers: []models.PartyMember{
							playerWithRole("d", 9, "support"),
							playerWithRole("e", 9, "tank"),
						}},
						{PartyMembers: []models.PartyMember{
							playerWithRole("f", 7, "fighter"),
						}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSwapper(0, 1, tt.args.ally1, tt.args.ally2, []string{tt.fields.attr}, tt.fields.channel.Ruleset.AllianceRule, tt.fields.channel.Ruleset.MatchingRule)
			s.Swap(scope)
			gotAlly1 := s.ally1
			gotAlly2 := s.ally2
			if !reflect.DeepEqual(getMapOfMember(gotAlly1), getMapOfMember(tt.want.newAlly1)) {
				t.Errorf("Swap() got = %v, want %v", gotAlly1, tt.want.newAlly1)
			}
			if !reflect.DeepEqual(getMapOfMember(gotAlly2), getMapOfMember(tt.want.newAlly2)) {
				t.Errorf("Swap() got = %v, want %v", gotAlly2, tt.want.newAlly2)
			}
		})
	}
}

func Test_getMoreMember(t *testing.T) {
	type args struct {
		allies           models.MatchingAlly
		allyCurrentIndex int
		requiredCount    int
	}
	tests := []struct {
		name         string
		args         args
		wantMembers  []models.PartyMember
		wantIdxSwaps []int
	}{
		{
			name: "no member in same party",
			args: args{
				allies: models.MatchingAlly{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("c", 9)}},
				}},
				allyCurrentIndex: 0,
				requiredCount:    2,
			},
			wantMembers: []models.PartyMember{
				player("a", 10),
				player("b", 8),
			},
			wantIdxSwaps: []int{0, 1},
		},
		{
			name: "member in same party in the last - expected although not eligible, this condition is handled in the next function",
			args: args{
				allies: models.MatchingAlly{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("b", 8), player("c", 9)}},
				}},
				allyCurrentIndex: 0,
				requiredCount:    2,
			},
			wantMembers: []models.PartyMember{
				player("a", 10),
			},
			wantIdxSwaps: []int{0},
		},
		{
			name: "member in same party in the middle",
			args: args{
				allies: models.MatchingAlly{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("b", 8), player("c", 9)}},
					{PartyMembers: []models.PartyMember{player("d", 7)}},
				}},
				allyCurrentIndex: 0,
				requiredCount:    2,
			},
			wantMembers: []models.PartyMember{
				player("a", 10),
				player("d", 7),
			},
			wantIdxSwaps: []int{0, 2},
		},
		{
			name: "no member in same party - get next member",
			args: args{
				allies: models.MatchingAlly{MatchingParties: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("c", 9)}},
				}},
				allyCurrentIndex: 1,
				requiredCount:    2,
			},
			wantMembers: []models.PartyMember{
				player("b", 8),
				player("c", 9),
			},
			wantIdxSwaps: []int{1, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMembers, gotIdxSwaps := getMoreMember(tt.args.allies, tt.args.allyCurrentIndex, tt.args.requiredCount)
			if !reflect.DeepEqual(gotMembers, tt.wantMembers) {
				t.Errorf("getMoreMember() gotMembers = %v, want %v", gotMembers, tt.wantMembers)
			}
			if !reflect.DeepEqual(gotIdxSwaps, tt.wantIdxSwaps) {
				t.Errorf("getMoreMember() gotIdxSwaps = %v, want %v", gotIdxSwaps, tt.wantIdxSwaps)
			}
		})
	}
}

func Test_swapParties(t *testing.T) {
	type args struct {
		party1       []models.MatchingParty
		party2       []models.MatchingParty
		swapIDParty1 []int
		swapIDParty2 []int
	}
	tests := []struct {
		name          string
		args          args
		wantnewParty1 []models.MatchingParty
		wantnewParty2 []models.MatchingParty
	}{
		{
			name: "swap party1[1] with party2[0]",
			args: args{
				party1: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("c", 9)}},
				},
				party2: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("d", 7)}},
					{PartyMembers: []models.PartyMember{player("e", 8)}},
					{PartyMembers: []models.PartyMember{player("f", 9)}},
				},
				swapIDParty1: []int{1},
				swapIDParty2: []int{0},
			},
			wantnewParty1: []models.MatchingParty{
				{PartyMembers: []models.PartyMember{player("a", 10)}},
				{PartyMembers: []models.PartyMember{player("c", 9)}},
				{PartyMembers: []models.PartyMember{player("d", 7)}},
			},
			wantnewParty2: []models.MatchingParty{
				{PartyMembers: []models.PartyMember{player("b", 8)}},
				{PartyMembers: []models.PartyMember{player("e", 8)}},
				{PartyMembers: []models.PartyMember{player("f", 9)}},
			},
		},
		{
			name: "swap party1[1,2] with party2[0]",
			args: args{
				party1: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("a", 10)}},
					{PartyMembers: []models.PartyMember{player("b", 8)}},
					{PartyMembers: []models.PartyMember{player("c", 9)}},
				},
				party2: []models.MatchingParty{
					{PartyMembers: []models.PartyMember{player("d", 7), player("e", 8)}},
					{PartyMembers: []models.PartyMember{player("f", 9)}},
				},
				swapIDParty1: []int{1, 2},
				swapIDParty2: []int{0},
			},
			wantnewParty1: []models.MatchingParty{
				{PartyMembers: []models.PartyMember{player("a", 10)}},
				{PartyMembers: []models.PartyMember{player("d", 7), player("e", 8)}},
			},
			wantnewParty2: []models.MatchingParty{
				{PartyMembers: []models.PartyMember{player("b", 8)}},
				{PartyMembers: []models.PartyMember{player("c", 9)}},
				{PartyMembers: []models.PartyMember{player("f", 9)}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNewParty1, gotNewParty2 := swapParties(tt.args.party1, tt.args.party2, tt.args.swapIDParty1, tt.args.swapIDParty2)
			if !reflect.DeepEqual(gotNewParty1, tt.wantnewParty1) {
				t.Errorf("swapParties() gotNewParty1 = %v, want %v", gotNewParty1, tt.wantnewParty1)
			}
			if !reflect.DeepEqual(gotNewParty2, tt.wantnewParty2) {
				t.Errorf("swapParties() gotNewParty2 = %v, want %v", gotNewParty2, tt.wantnewParty2)
			}
		})
	}
}

// profiling with pprof of a benchmark in go
//
// go test -bench=. -benchmem -memprofile memprofile.out -cpuprofile profile.out -run=^#
// note: -run=^# avoid executing any tests functions in the test files
//
// go tool pprof profile.out
//
// go tool pprof memprofile.out
func BenchmarkRebalance(b *testing.B) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalance", "")
	defer scope.Finish()

	// mock allies
	mockAllies := []models.MatchingAlly{
		{MatchingParties: []models.MatchingParty{
			{PartyMembers: []models.PartyMember{player("a", 10)}},
			{PartyMembers: []models.PartyMember{player("b", 8), player("c", 9)}},
		}},
		{MatchingParties: []models.MatchingParty{
			{PartyMembers: []models.PartyMember{player("d", 9)}},
		}},
		{MatchingParties: []models.MatchingParty{
			{PartyMembers: []models.PartyMember{player("e", 2)}},
		}},
	}
	// run the Rebalance function b.N times
	for n := 0; n < b.N; n++ {
		Rebalance(scope, "", mockAllies, mockChannel.Ruleset.AllianceRule, mockChannel.Ruleset.MatchingRule)
	}
}

func BenchmarkRebalanceRandomWithMaxPlayer(b *testing.B) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalance", "")
	defer scope.Finish()

	// run the Rebalance function b.N times
	maxAlly := 10

	// we create several combinations of party with max player = 5
	partyCombinations := [][]int{
		{5},    // 1 party with 5 member
		{4, 1}, // 2 parties with first party have 4 member, second party have 1 member
		{3, 2},
		{2, 2, 1},
		{2, 1, 1, 1},
		{1, 1, 1, 1, 1},
	}
	for n := 0; n < b.N; n++ {
		allyCount := generateRandomInt(1, maxAlly)
		allies := make([]models.MatchingAlly, 0, allyCount)
		for i := 0; i < allyCount; i++ {
			allies = append(allies, models.MatchingAlly{
				MatchingParties: generateRandomPartiesWithCombinations(partyCombinations),
			})
		}
		Rebalance(scope, "", allies, mockChannel.Ruleset.AllianceRule, mockChannel.Ruleset.MatchingRule)
	}
}

func generateRandomPartiesWithCombinations(partyCombinations [][]int) []models.MatchingParty {
	randomIndex := generateRandomInt(0, len(partyCombinations))
	combination := partyCombinations[randomIndex]

	parties := make([]models.MatchingParty, 0, len(combination))
	for _, memberCount := range combination {
		members := make([]models.PartyMember, 0, memberCount)
		for i := 0; i < memberCount; i++ {
			members = append(members, generateRandomPlayer())
		}
		parties = append(parties, models.MatchingParty{PartyMembers: members})
	}
	return parties
}

func BenchmarkRebalanceRandomAllies(b *testing.B) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalance", "")
	defer scope.Finish()

	// run the Rebalance function b.N times
	maxCount := 10
	for n := 0; n < b.N; n++ {
		Rebalance(scope, "", generateRandomAllies(maxCount), mockChannel.Ruleset.AllianceRule, mockChannel.Ruleset.MatchingRule)
	}
}

func generateRandomAllies(maxCount int) []models.MatchingAlly {
	allyCount := generateRandomInt(1, maxCount)
	allies := make([]models.MatchingAlly, 0, allyCount)
	for i := 0; i < allyCount; i++ {
		allies = append(allies, models.MatchingAlly{
			MatchingParties: generateRandomParties(maxCount),
		})
	}
	return allies
}

func generateRandomParties(maxCount int) []models.MatchingParty {
	partyCount := generateRandomInt(1, maxCount)
	parties := make([]models.MatchingParty, 0, partyCount)
	for i := 0; i < partyCount; i++ {
		parties = append(parties, models.MatchingParty{
			PartyMembers: generateRandomMembers(maxCount),
		})
	}
	return parties
}

func generateRandomMembers(maxCount int) []models.PartyMember {
	memberCount := generateRandomInt(1, maxCount)
	members := make([]models.PartyMember, 0, memberCount)
	for i := 0; i < memberCount; i++ {
		members = append(members, generateRandomPlayer())
	}
	return members
}

func generateRandomPlayer() models.PartyMember {
	userID := utils.GenerateUUID()
	mmr := generateRandomFloat64(0, 100)
	return player(userID, mmr)
}

func generateRandomFloat64(_min, _max int) float64 {
	nBig, err := rand.Int(rand.Reader, big.NewInt(27))
	if err != nil {
		panic(err)
	}
	return float64(_min) + float64(nBig.Int64())*float64(_max-_min)
}

func generateRandomInt(_min, _max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(_max)))
	if err != nil {
		panic(err)
	}
	return int(n.Int64()) + _min
}

func Test_avg(t *testing.T) {
	type args struct {
		members       []models.PartyMember
		attributeName string
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "average",
			args: args{
				members: []models.PartyMember{
					{UserID: "a", ExtraAttributes: map[string]interface{}{"mmr": 10}},
					{UserID: "b", ExtraAttributes: map[string]interface{}{"mmr": 8}},
				},
				attributeName: "mmr",
			},
			want: 9.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Avg(tt.args.members, []string{tt.args.attributeName}, mockChannel.Ruleset.MatchingRule); got != tt.want {
				t.Errorf("avg() = %v, want %v", got, tt.want)
			}
		})
	}
}
