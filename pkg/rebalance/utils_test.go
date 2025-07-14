// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance

import (
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"github.com/stretchr/testify/assert"
)

func Test_getAttributeNameForRebalance(t *testing.T) {
	type args struct {
		channel models.Channel
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get_mmr_attribute",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						MatchingRule: []models.MatchingRule{
							{Attribute: "attr1", Criteria: constants.DistanceCriteria},
							{Attribute: "attr1"},
						},
					},
				},
			},
			want: "attr1",
		}, {
			name: "get_first_attribute_with_distance_criteria",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						MatchingRule: []models.MatchingRule{
							{Attribute: "attr1"},
							{Attribute: "attr2", Criteria: constants.DistanceCriteria},
							{Attribute: "attr3", Criteria: constants.DistanceCriteria},
						},
					},
				},
			},
			want: "attr2",
		}, {
			name: "no_attribute_expected",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						MatchingRule: []models.MatchingRule{
							{Attribute: "attr1"},
							{Attribute: "attr2"},
						},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAttributeNameForRebalance(tt.args.channel.Ruleset.MatchingRule)
			if tt.want == "" {
				assert.Empty(t, got)
			} else {
				assert.EqualValues(t, 1, len(got))
				assert.Equal(t, tt.want, got[0])
			}
		})
	}
}

func Test_getMultiAttributeNameForRebalance(t *testing.T) {
	type args struct {
		channel models.Channel
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "default mmr",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						MatchingRule: []models.MatchingRule{
							{Attribute: "attr1", Criteria: constants.DistanceCriteria},
							{Attribute: "attr2"},
						},
					},
				},
			},
			want: []string{"attr1"},
		},
		{
			name: "mrr for balancing",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						MatchingRule: []models.MatchingRule{
							{Attribute: "attr1", Criteria: constants.DistanceCriteria},
							{Attribute: "attr2", Criteria: constants.DistanceCriteria, IsForBalancing: models.TRUE()},
						},
					},
				},
			},
			want: []string{"attr2"},
		},
		{
			name: "2 mmr for balancing",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						MatchingRule: []models.MatchingRule{
							{Attribute: "attr1", Criteria: constants.DistanceCriteria},
							{Attribute: "attr2", Criteria: constants.DistanceCriteria, IsForBalancing: models.TRUE()},
							{Attribute: "attr3", Criteria: constants.DistanceCriteria, IsForBalancing: models.TRUE()},
							{Attribute: "attr4", Criteria: constants.DistanceCriteria, IsForBalancing: models.FALSE()},
						},
					},
				},
			},
			want: []string{"attr2", "attr3"},
		},
		{
			name: "all mmr false",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						MatchingRule: []models.MatchingRule{
							{Attribute: "attr1", Criteria: constants.DistanceCriteria, IsForBalancing: models.FALSE()},
							{Attribute: "attr2", Criteria: constants.DistanceCriteria, IsForBalancing: models.FALSE()},
							{Attribute: "attr3", Criteria: constants.DistanceCriteria, IsForBalancing: models.FALSE()},
							{Attribute: "attr4", Criteria: constants.DistanceCriteria, IsForBalancing: models.FALSE()},
						},
					},
				},
			},
			want: nil,
		},
		{
			name: "null mmr",
			args: args{
				channel: models.Channel{
					Ruleset: models.RuleSet{
						MatchingRule: []models.MatchingRule{
							{Attribute: "attr1"},
							{Attribute: "attr2", Criteria: constants.DistanceCriteria},
							{Attribute: "attr3", Criteria: constants.DistanceCriteria},
							{Attribute: "attr4", Criteria: constants.DistanceCriteria},
						},
					},
				},
			},
			want: []string{"attr2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAttributeNameForRebalance(tt.args.channel.Ruleset.MatchingRule)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
