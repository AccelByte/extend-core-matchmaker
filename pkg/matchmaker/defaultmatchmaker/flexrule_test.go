// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"testing"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"github.com/go-openapi/swag"
	"github.com/stretchr/testify/require"
)

func TestIsActiveFlexRule(t *testing.T) {
	tests := []struct {
		Name         string
		CurrentTime  func() time.Time
		TicketTime   time.Time
		FlexDuration time.Duration
		Want         bool
	}{
		{
			Name:         "should active flex given current time bigger than ticket time + flex duration",
			CurrentTime:  func() time.Time { return time.Date(2021, 10, 14, 10, 0, 25, 0, time.UTC) },
			TicketTime:   time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			FlexDuration: 20 * time.Second,
			Want:         true,
		},
		{
			Name:         "should not active flex given current time smaller than ticket time + flex duration",
			CurrentTime:  func() time.Time { return time.Date(2021, 10, 14, 10, 0, 10, 0, time.UTC) },
			TicketTime:   time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			FlexDuration: 20 * time.Second,
			Want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			Now = tt.CurrentTime

			got := isActiveFlexRule(tt.TicketTime, tt.FlexDuration)

			require.Equal(t, tt.Want, got)
		})
	}
}

func TestApplyRuleFlexing(t *testing.T) {
	tests := []struct {
		Name        string
		CurrentTime func() time.Time
		TicketTime  time.Time
		RuleSet     models.RuleSet
		Want        models.RuleSet
	}{
		{
			Name:        "apply the lattest alliance rule flexing",
			CurrentTime: func() time.Time { return time.Date(2021, 10, 14, 10, 0, 15, 0, time.UTC) },
			TicketTime:  time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			RuleSet: models.RuleSet{
				MatchingRule: []models.MatchingRule{
					{
						Attribute: "mmr",
						Criteria:  "distance",
						Reference: 10,
					},
				},
				FlexingRule: []models.FlexingRule{
					{
						Duration: 10,
						MatchingRule: models.MatchingRule{
							Attribute: "mmr",
							Criteria:  "distance",
							Reference: 20,
						},
					},
				},
			},
			Want: models.RuleSet{
				MatchingRule: []models.MatchingRule{
					{
						Attribute: "mmr",
						Criteria:  "distance",
						Reference: 20,
					},
				},
				FlexingRule: []models.FlexingRule{
					{
						Duration: 10,
						MatchingRule: models.MatchingRule{
							Attribute: "mmr",
							Criteria:  "distance",
							Reference: 20,
						},
					},
				},
			},
		},
		{
			Name:        "apply the lattest alliance rule flexing for multiple flex rule that not sorted",
			CurrentTime: func() time.Time { return time.Date(2021, 10, 14, 10, 0, 25, 0, time.UTC) },
			TicketTime:  time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			RuleSet: models.RuleSet{
				MatchingRule: []models.MatchingRule{
					{
						Attribute: "mmr",
						Criteria:  "distance",
						Reference: 10,
					},
				},
				FlexingRule: []models.FlexingRule{
					{
						Duration: 20,
						MatchingRule: models.MatchingRule{
							Attribute: "mmr",
							Criteria:  "distance",
							Reference: 30,
						},
					},
					{
						Duration: 10,
						MatchingRule: models.MatchingRule{
							Attribute: "mmr",
							Criteria:  "distance",
							Reference: 20,
						},
					},
				},
			},
			Want: models.RuleSet{
				MatchingRule: []models.MatchingRule{
					{
						Attribute: "mmr",
						Criteria:  "distance",
						Reference: 30,
					},
				},
				FlexingRule: []models.FlexingRule{
					{
						Duration: 20,
						MatchingRule: models.MatchingRule{
							Attribute: "mmr",
							Criteria:  "distance",
							Reference: 30,
						},
					},
					{
						Duration: 10,
						MatchingRule: models.MatchingRule{
							Attribute: "mmr",
							Criteria:  "distance",
							Reference: 20,
						},
					},
				},
			},
		},
		{
			Name:        "apply multiple active flexing rule for multiple attribute",
			CurrentTime: func() time.Time { return time.Date(2021, 10, 14, 10, 0, 25, 0, time.UTC) },
			TicketTime:  time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			RuleSet: models.RuleSet{
				MatchingRule: []models.MatchingRule{
					{
						Attribute: "mmr",
						Criteria:  "distance",
						Reference: 10,
					},
					{
						Attribute: "conduct_summary",
						Criteria:  "distance",
						Reference: 10,
					},
				},
				FlexingRule: []models.FlexingRule{
					{
						Duration: 20,
						MatchingRule: models.MatchingRule{
							Attribute: "mmr",
							Criteria:  "distance",
							Reference: 20,
						},
					},
					{
						Duration: 20,
						MatchingRule: models.MatchingRule{
							Attribute: "conduct_summary",
							Criteria:  "distance",
							Reference: 20,
						},
					},
				},
			},
			Want: models.RuleSet{
				MatchingRule: []models.MatchingRule{
					{
						Attribute: "mmr",
						Criteria:  "distance",
						Reference: 20,
					},
					{
						Attribute: "conduct_summary",
						Criteria:  "distance",
						Reference: 20,
					},
				},
				FlexingRule: []models.FlexingRule{
					{
						Duration: 20,
						MatchingRule: models.MatchingRule{
							Attribute: "mmr",
							Criteria:  "distance",
							Reference: 20,
						},
					},
					{
						Duration: 20,
						MatchingRule: models.MatchingRule{
							Attribute: "conduct_summary",
							Criteria:  "distance",
							Reference: 20,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			Now = tt.CurrentTime

			got, _ := applyRuleFlexing(tt.RuleSet, tt.TicketTime)

			require.Equal(t, tt.Want, got)
		})
	}
}

func TestApplyRuleFlexingSetDefault(t *testing.T) {
	tests := []struct {
		Name        string
		CurrentTime func() time.Time
		TicketTime  time.Time
		RuleSet     models.RuleSet
		Want        models.RuleSet
	}{
		{
			Name:        "apply multiple active flexing rule for multiple attribute with weight max is set should not flexed",
			CurrentTime: func() time.Time { return time.Date(2021, 10, 14, 10, 0, 25, 0, time.UTC) },
			TicketTime:  time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			RuleSet: models.RuleSet{
				MatchingRule: []models.MatchingRule{
					{
						Attribute: "mmr",
						Criteria:  "distance",
						Reference: 10,
						Weight:    swag.Float64(1.0),
					},
					{
						Attribute: "conduct_summary",
						Criteria:  "distance",
						Reference: 10,
						Weight:    swag.Float64(1.0),
					},
				},
				FlexingRule: []models.FlexingRule{
					{
						Duration: 20,
						MatchingRule: models.MatchingRule{
							Attribute: "mmr",
							Criteria:  "distance",
							Reference: 20,
						},
					},
					{
						Duration: 20,
						MatchingRule: models.MatchingRule{
							Attribute: "conduct_summary",
							Criteria:  "distance",
							Reference: 20,
						},
					},
				},
			},
			Want: models.RuleSet{
				MatchingRule: []models.MatchingRule{
					{
						Attribute:        "mmr",
						Criteria:         "distance",
						Reference:        20,
						NormalizationMax: 20,
						Weight:           swag.Float64(1.0),
					},
					{
						Attribute:        "conduct_summary",
						Criteria:         "distance",
						Reference:        20,
						NormalizationMax: 20,
						Weight:           swag.Float64(1.0),
					},
				},
				FlexingRule: []models.FlexingRule{
					{
						Duration: 20,
						MatchingRule: models.MatchingRule{
							Attribute:        "mmr",
							Criteria:         "distance",
							Reference:        20,
							NormalizationMax: 0,
						},
					},
					{
						Duration: 20,
						MatchingRule: models.MatchingRule{
							Attribute:        "conduct_summary",
							Criteria:         "distance",
							Reference:        20,
							NormalizationMax: 0,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			Now = tt.CurrentTime

			tt.RuleSet.SetDefaultValues()
			got, _ := applyRuleFlexing(tt.RuleSet, tt.TicketTime)

			require.Equal(t, tt.Want, got)
		})
	}
}
