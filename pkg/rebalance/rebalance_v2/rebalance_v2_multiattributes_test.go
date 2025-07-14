// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v2

import (
	"context"
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"

	"github.com/stretchr/testify/assert"
)

func TestRebalanceMultiAttributes(t *testing.T) {
	mmr2 := "mmr2"
	testCases := []struct {
		name               string
		alliesInput        []models.MatchingAlly
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		expected           []models.MatchingAlly
	}{
		{name: "single attributes",
			activeAllianceRule: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 2,
				PlayerMaxNumber: 2,
			},
			matchingRules: []models.MatchingRule{
				{Attribute: mmr, Criteria: constants.DistanceCriteria, IsForBalancing: models.TRUE(), NormalizationMax: 5000},
			},
			alliesInput: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("d", 1090)}),
					matchingParty([]models.PartyMember{playerWithMMR("e", 1230)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("a", 3280)}),
					matchingParty([]models.PartyMember{playerWithMMR("c", 3010)}),
				}},
			},
			expected: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("a", 3280)}),
					matchingParty([]models.PartyMember{playerWithMMR("d", 1090)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("c", 3010)}),
					matchingParty([]models.PartyMember{playerWithMMR("e", 1230)}),
				}},
			},
		},
		{name: "no mmr to balance",
			activeAllianceRule: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 2,
				PlayerMaxNumber: 2,
			},
			matchingRules: []models.MatchingRule{
				{Attribute: mmr, Criteria: constants.DistanceCriteria, IsForBalancing: models.FALSE()},
				{Attribute: mmr2, Criteria: constants.DistanceCriteria, IsForBalancing: models.FALSE()},
			},
			alliesInput: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("d", mmr, mmr2, 1090, 109)}),
					matchingParty([]models.PartyMember{playerWith2MMR("e", mmr, mmr2, 1230, 123)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("a", mmr, mmr2, 3280, 328)}),
					matchingParty([]models.PartyMember{playerWith2MMR("c", mmr, mmr2, 3010, 301)}),
				}},
			},
			expected: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("d", mmr, mmr2, 1090, 109)}),
					matchingParty([]models.PartyMember{playerWith2MMR("e", mmr, mmr2, 1230, 123)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("a", mmr, mmr2, 3280, 328)}),
					matchingParty([]models.PartyMember{playerWith2MMR("c", mmr, mmr2, 3010, 301)}),
				}},
			},
		},
		{name: "mmr2 to balance",
			activeAllianceRule: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 2,
				PlayerMaxNumber: 2,
			},
			matchingRules: []models.MatchingRule{
				{Attribute: mmr, Criteria: constants.DistanceCriteria, IsForBalancing: models.FALSE(), NormalizationMax: 5000},
				{Attribute: mmr2, Criteria: constants.DistanceCriteria, IsForBalancing: models.TRUE(), NormalizationMax: 5000},
			},
			alliesInput: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("d", mmr, mmr2, 1090, 3280)}),
					matchingParty([]models.PartyMember{playerWith2MMR("e", mmr, mmr2, 1230, 3010)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("a", mmr, mmr2, 3280, 1090)}),
					matchingParty([]models.PartyMember{playerWith2MMR("c", mmr, mmr2, 3010, 1230)}),
				}},
			},
			expected: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("d", mmr, mmr2, 1090, 3280)}),
					matchingParty([]models.PartyMember{playerWith2MMR("a", mmr, mmr2, 3280, 1090)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("e", mmr, mmr2, 1230, 3010)}),
					matchingParty([]models.PartyMember{playerWith2MMR("c", mmr, mmr2, 3010, 1230)}),
				}},
			},
		},
		{name: "both mmr to balance",
			activeAllianceRule: models.AllianceRule{
				MinNumber:       2,
				MaxNumber:       2,
				PlayerMinNumber: 2,
				PlayerMaxNumber: 2,
			},
			matchingRules: []models.MatchingRule{
				{Attribute: mmr, Criteria: constants.DistanceCriteria, IsForBalancing: models.TRUE(), NormalizationMax: 5000},
				{Attribute: mmr2, Criteria: constants.DistanceCriteria, IsForBalancing: models.TRUE(), NormalizationMax: 5000},
			},
			alliesInput: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("d", mmr, mmr2, 1090, 3280)}),
					matchingParty([]models.PartyMember{playerWith2MMR("e", mmr, mmr2, 1230, 3010)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("a", mmr, mmr2, 3280, 1090)}),
					matchingParty([]models.PartyMember{playerWith2MMR("c", mmr, mmr2, 3010, 1230)}),
				}},
			},
			expected: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("d", mmr, mmr2, 1090, 3280)}),
					matchingParty([]models.PartyMember{playerWith2MMR("c", mmr, mmr2, 3010, 1230)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWith2MMR("a", mmr, mmr2, 3280, 1090)}),
					matchingParty([]models.PartyMember{playerWith2MMR("e", mmr, mmr2, 1230, 3010)}),
				}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scope := envelope.NewRootScope(context.Background(), utils.GenerateUUID(), utils.GenerateUUID())
			defer scope.Finish()
			got := RebalanceV2(scope, "", tc.alliesInput, tc.activeAllianceRule, tc.matchingRules, "")
			assert.Equal(t, tc.expected, got)
		})
	}
}
