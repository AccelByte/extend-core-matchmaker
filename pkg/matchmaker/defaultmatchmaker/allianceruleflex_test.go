// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package defaultmatchmaker

import (
	"testing"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestApplyAllianceFlexingRules(t *testing.T) {
	tests := []struct {
		Name        string
		CurrentTime func() time.Time
		TicketTime  time.Time
		RuleSet     models.RuleSet
		Want        models.RuleSet
	}{
		{
			Name:        "apply the lattest alliance rule flexing",
			CurrentTime: func() time.Time { return time.Date(2021, 10, 14, 10, 0, 25, 0, time.UTC) },
			TicketTime:  time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			RuleSet: models.RuleSet{
				AllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 4,
					PlayerMaxNumber: 4,
				},
				AllianceFlexingRule: []models.AllianceFlexingRule{
					{
						Duration: 10,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 2,
							PlayerMaxNumber: 4,
						},
					},
					{
						Duration: 20,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 4,
						},
					},
				},
			},
			Want: models.RuleSet{
				AllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 4,
				},
				AllianceFlexingRule: []models.AllianceFlexingRule{
					{
						Duration: 10,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 2,
							PlayerMaxNumber: 4,
						},
					},
					{
						Duration: 20,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 4,
						},
					},
				},
			},
		},
		{
			Name:        "alliance rule flexing that haven't active should not applied",
			CurrentTime: func() time.Time { return time.Date(2021, 10, 14, 10, 0, 25, 0, time.UTC) },
			TicketTime:  time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			RuleSet: models.RuleSet{
				AllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 4,
					PlayerMaxNumber: 4,
				},
				AllianceFlexingRule: []models.AllianceFlexingRule{
					{
						Duration: 10,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 2,
							PlayerMaxNumber: 4,
						},
					},
					{
						Duration: 40,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 4,
						},
					},
				},
			},
			Want: models.RuleSet{
				AllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 2,
					PlayerMaxNumber: 4,
				},
				AllianceFlexingRule: []models.AllianceFlexingRule{
					{
						Duration: 10,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 2,
							PlayerMaxNumber: 4,
						},
					},
					{
						Duration: 40,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 4,
						},
					},
				},
			},
		},
		{
			Name:        "apply the lattest alliance rule flexing even the duration is not sorted",
			CurrentTime: func() time.Time { return time.Date(2021, 10, 14, 10, 0, 25, 0, time.UTC) },
			TicketTime:  time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			RuleSet: models.RuleSet{
				AllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 4,
					PlayerMaxNumber: 4,
				},
				AllianceFlexingRule: []models.AllianceFlexingRule{
					{
						Duration: 20,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 4,
						},
					},
					{
						Duration: 10,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 2,
							PlayerMaxNumber: 4,
						},
					},
				},
			},
			Want: models.RuleSet{
				AllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 4,
				},
				AllianceFlexingRule: []models.AllianceFlexingRule{
					{
						Duration: 20,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 1,
							PlayerMaxNumber: 4,
						},
					},
					{
						Duration: 10,
						AllianceRule: models.AllianceRule{
							MinNumber:       2,
							MaxNumber:       2,
							PlayerMinNumber: 2,
							PlayerMaxNumber: 4,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			Now = tt.CurrentTime

			got, _ := applyAllianceFlexingRules(tt.RuleSet, tt.TicketTime)

			require.Equal(t, tt.Want, got)
		})
	}
}

func TestApplyAllianceFlexingRule(t *testing.T) {
	tests := []struct {
		Name                 string
		CurrentTime          func() time.Time
		TicketTime           time.Time
		AllianceRule         models.AllianceRule
		AllianceFlexingRules []models.AllianceFlexingRule
		Want                 models.AllianceRule
	}{
		{
			Name:        "apply the lattest alliance rule flexing",
			CurrentTime: func() time.Time { return time.Date(2021, 10, 14, 10, 0, 25, 0, time.UTC) },
			TicketTime:  time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			AllianceRule: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 4,
				PlayerMaxNumber: 4,
			},
			AllianceFlexingRules: []models.AllianceFlexingRule{
				{
					Duration: 10,
					AllianceRule: models.AllianceRule{
						MinNumber:       2,
						MaxNumber:       2,
						PlayerMinNumber: 2,
						PlayerMaxNumber: 4,
					},
				},
				{
					Duration: 20,
					AllianceRule: models.AllianceRule{
						MinNumber:       2,
						MaxNumber:       2,
						PlayerMinNumber: 1,
						PlayerMaxNumber: 4,
					},
				},
			},
			Want: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 1,
				PlayerMaxNumber: 4,
			},
		},
		{
			Name:        "alliance rule flexing that haven't active should not applied",
			CurrentTime: func() time.Time { return time.Date(2021, 10, 14, 10, 0, 25, 0, time.UTC) },
			TicketTime:  time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			AllianceRule: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 4,
				PlayerMaxNumber: 4,
			},
			AllianceFlexingRules: []models.AllianceFlexingRule{
				{
					Duration: 10,
					AllianceRule: models.AllianceRule{
						MinNumber:       2,
						MaxNumber:       2,
						PlayerMinNumber: 2,
						PlayerMaxNumber: 4,
					},
				},
				{
					Duration: 40,
					AllianceRule: models.AllianceRule{
						MinNumber:       2,
						MaxNumber:       2,
						PlayerMinNumber: 1,
						PlayerMaxNumber: 4,
					},
				},
			},
			Want: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 2,
				PlayerMaxNumber: 4,
			},
		},
		{
			Name:        "apply the lattest alliance rule flexing even the duration is not sorted",
			CurrentTime: func() time.Time { return time.Date(2021, 10, 14, 10, 0, 25, 0, time.UTC) },
			TicketTime:  time.Date(2021, 10, 14, 10, 0, 0, 0, time.UTC),
			AllianceRule: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 4,
				PlayerMaxNumber: 4,
			},
			AllianceFlexingRules: []models.AllianceFlexingRule{
				{
					Duration: 20,
					AllianceRule: models.AllianceRule{
						MinNumber:       2,
						MaxNumber:       2,
						PlayerMinNumber: 1,
						PlayerMaxNumber: 4,
					},
				},
				{
					Duration: 10,
					AllianceRule: models.AllianceRule{
						MinNumber:       2,
						MaxNumber:       2,
						PlayerMinNumber: 2,
						PlayerMaxNumber: 4,
					},
				},
			},
			Want: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 1,
				PlayerMaxNumber: 4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			Now = tt.CurrentTime

			got, _ := ApplyAllianceFlexingRule(tt.AllianceRule, tt.AllianceFlexingRules, tt.TicketTime)

			require.Equal(t, tt.Want, got)
		})
	}
}
