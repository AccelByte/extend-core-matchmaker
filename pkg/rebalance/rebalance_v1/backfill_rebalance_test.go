// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v1

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"

	"github.com/stretchr/testify/assert"
)

var (
	mockActiveAllianceRule = models.AllianceRule{
		MinNumber:       2,
		MaxNumber:       2,
		PlayerMinNumber: 1,
		PlayerMaxNumber: 10,
	}

	mockMatchingRules = []models.MatchingRule{
		{
			Attribute: "mmr",
			Criteria:  constants.DistanceCriteria,
			Reference: 100,
		},
	}
)

func generateMatchingParty(countMember int, avgMmr float64, locked bool) models.MatchingParty {
	partyMembers := make([]models.PartyMember, countMember)
	for i := 0; i < countMember; i++ {
		partyMembers[i] = models.PartyMember{
			UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{"mmr": avgMmr},
		}
	}
	return models.MatchingParty{
		PartyID:      utils.GenerateUUID(),
		PartyMembers: partyMembers,
		Locked:       locked,
	}
}

func Test_adjustAlliesBasedOnMaxTeamNumber(t *testing.T) {
	type args struct {
		allies             []models.MatchingAlly
		activeAllianceRule models.AllianceRule
		wantAllies         []models.MatchingAlly
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "number of allies EQUAL max team number",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{{PartyID: "party1"}}},
					{MatchingParties: []models.MatchingParty{{PartyID: "party2"}}},
				},
				activeAllianceRule: models.AllianceRule{
					MinNumber: 1,
					MaxNumber: 2,
				},
				wantAllies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{{PartyID: "party1"}}},
					{MatchingParties: []models.MatchingParty{{PartyID: "party2"}}},
				},
			},
		},
		{
			name: "number of allies LESS THAN max team number",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{{PartyID: "party1"}}},
				},
				activeAllianceRule: models.AllianceRule{
					MinNumber: 1,
					MaxNumber: 2,
				},
				wantAllies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{{PartyID: "party1"}}},
					{MatchingParties: nil},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAllies := adjustAlliesBasedOnMaxTeamNumber(tt.args.allies, tt.args.activeAllianceRule)
			assert.Equal(t, tt.args.wantAllies, gotAllies, "allies")
		})
	}
}

func Test_extractAllies(t *testing.T) {
	type args struct {
		allies []models.MatchingAlly
	}
	tests := []struct {
		name                string
		args                args
		wantLockedParties   map[int][]models.MatchingParty
		wantUnlockedParties []models.MatchingParty
		wantBestAllies      map[int][]models.MatchingParty
	}{
		{
			name: "full parties in allies",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyID: "party1", Locked: true},
						{PartyID: "party2", Locked: true},
						{PartyID: "party3", Locked: false},
					}},
					{MatchingParties: []models.MatchingParty{
						{PartyID: "party4", Locked: true},
						{PartyID: "party5", Locked: false},
					}},
				},
			},
			wantLockedParties: map[int][]models.MatchingParty{
				0: {
					{PartyID: "party1", Locked: true},
					{PartyID: "party2", Locked: true},
				},
				1: {{PartyID: "party4", Locked: true}},
			},
			wantUnlockedParties: []models.MatchingParty{
				{PartyID: "party3", Locked: false},
				{PartyID: "party5", Locked: false},
			},
			wantBestAllies: map[int][]models.MatchingParty{
				0: {
					{PartyID: "party1", Locked: true},
					{PartyID: "party2", Locked: true},
					{PartyID: "party3", Locked: false},
				},
				1: {
					{PartyID: "party4", Locked: true},
					{PartyID: "party5", Locked: false},
				},
			},
		},
		{
			name: "empty parties in second ally",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						{PartyID: "party1", Locked: true},
						{PartyID: "party2", Locked: false},
					}},
					{MatchingParties: nil},
				},
			},
			wantLockedParties: map[int][]models.MatchingParty{
				0: {{PartyID: "party1", Locked: true}},
				1: nil,
			},
			wantUnlockedParties: []models.MatchingParty{
				{PartyID: "party2", Locked: false},
			},
			wantBestAllies: map[int][]models.MatchingParty{
				0: {
					{PartyID: "party1", Locked: true},
					{PartyID: "party2", Locked: false},
				},
				1: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLockedParties, gotUnlockedParties, gotBestAllies := rebalance.ExtractAllies(tt.args.allies)
			if !reflect.DeepEqual(gotLockedParties, tt.wantLockedParties) {
				t.Errorf("extractAllies() gotLockedParties = %v, want %v", gotLockedParties, tt.wantLockedParties)
			}
			if !reflect.DeepEqual(gotUnlockedParties, tt.wantUnlockedParties) {
				t.Errorf("extractAllies() gotUnlockedParties = %v, want %v", gotUnlockedParties, tt.wantUnlockedParties)
			}
			if !reflect.DeepEqual(gotBestAllies, tt.wantBestAllies) {
				t.Errorf("extractAllies() gotBestAllies = %v, want %v", gotBestAllies, tt.wantBestAllies)
			}
		})
	}
}

func Test_BackfillRebalance_1(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "Test_BackfillRebalance_1", "")
	defer scope.Finish()

	party1 := models.MatchingParty{
		PartyID: utils.GenerateUUID(),
		PartyMembers: []models.PartyMember{
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{"mmr": 20.0}},
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{"mmr": 30.0}},
		},
		Locked: true,
	}
	party2 := generateMatchingParty(2, 10, true)
	party3 := generateMatchingParty(1, 20, false)

	allies := []models.MatchingAlly{
		{
			MatchingParties: []models.MatchingParty{
				party1,
				party3,
			},
		},
		{
			MatchingParties: []models.MatchingParty{
				party2,
			},
		},
	}

	wantAllies := []models.MatchingAlly{
		{
			MatchingParties: []models.MatchingParty{
				party1,
			},
		},
		{
			MatchingParties: []models.MatchingParty{
				party2,
				party3,
			},
		},
	}

	got := BackfillRebalance(scope, "", allies, mockActiveAllianceRule, mockMatchingRules)
	assert.Equal(t, wantAllies, got, "")
}

func Test_BackfillRebalance_2(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "Test_BackfillRebalance_2", "")
	defer scope.Finish()

	// assign locked parties (current session)
	party1 := generateMatchingParty(2, 10.0, true)
	ally1 := models.MatchingAlly{MatchingParties: []models.MatchingParty{party1}}

	party2 := generateMatchingParty(3, 20.0, true)
	ally2 := models.MatchingAlly{MatchingParties: []models.MatchingParty{party2}}

	// assign unlocked parties (new parties)
	party3 := generateMatchingParty(1, 10.0, false)
	party4 := generateMatchingParty(3, 20.0, false)
	party5 := generateMatchingParty(2, 30.0, false)
	newParties := []models.MatchingParty{party3, party4, party5}
	ally2.MatchingParties = append(ally2.MatchingParties, newParties...)

	allies := []models.MatchingAlly{
		ally1,
		ally2,
	}

	wantAllies := []models.MatchingAlly{
		// ally1
		{
			MatchingParties: []models.MatchingParty{
				party1,
				party5,
				party3,
			},
		},

		// ally2
		{
			MatchingParties: []models.MatchingParty{
				party2,
				party4,
			},
		},
	}

	got := BackfillRebalance(scope, "", allies, mockActiveAllianceRule, mockMatchingRules)
	assert.ElementsMatch(t, getPartyIDs(wantAllies[0]), getPartyIDs(got[0]), "ally 1 should equal")
	assert.ElementsMatch(t, getPartyIDs(wantAllies[1]), getPartyIDs(got[1]), "ally 2 should equal")
}

func Test_BackfillRebalance_NoNewParties(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "Test_BackfillRebalance_NoNewParties", "")
	defer scope.Finish()

	party1 := models.MatchingParty{
		PartyID: utils.GenerateUUID(),
		PartyMembers: []models.PartyMember{
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{"mmr": 20.0}},
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{"mmr": 30.0}},
		},
		Locked: true,
	}
	party2 := generateMatchingParty(2, 10, true)
	party3 := generateMatchingParty(1, 20, true)

	allies := []models.MatchingAlly{
		{
			MatchingParties: []models.MatchingParty{
				party1,
				party3,
			},
		},
		{
			MatchingParties: []models.MatchingParty{
				party2,
			},
		},
	}

	got := BackfillRebalance(scope, "", allies, mockActiveAllianceRule, mockMatchingRules)
	assert.Equal(t, math.Round(countDistance(allies, []string{"mmr"}, nil)), math.Round(countDistance(got, []string{"mmr"}, nil)), "distance should be 0")
	assert.Equal(t, 13.0, math.Round(countDistance(got, []string{"mmr"}, nil)), "distance should be 0")
	assert.Equal(t, 1, CountMemberDiff(got), "member diff should be 1")
}

func Test_BackfillRebalance_NoSession(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "Test_BackfillRebalance_NoSession", "")
	defer scope.Finish()

	party1 := models.MatchingParty{
		PartyID: utils.GenerateUUID(),
		PartyMembers: []models.PartyMember{
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{"mmr": 20.0}},
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{"mmr": 30.0}},
		},
		Locked: false,
	}
	party2 := generateMatchingParty(2, 10, false)
	party3 := generateMatchingParty(1, 20, false)

	allies := []models.MatchingAlly{
		{
			MatchingParties: []models.MatchingParty{
				party1,
				party3,
			},
		},
		{
			MatchingParties: []models.MatchingParty{
				party2,
			},
		},
	}

	got := BackfillRebalance(scope, "", allies, mockActiveAllianceRule, mockMatchingRules)
	assert.Greater(t, math.Round(countDistance(allies, []string{"mmr"}, nil)), math.Round(countDistance(got, []string{"mmr"}, nil)), "distance should be 0")
	assert.Equal(t, 12.0, math.Round(countDistance(got, []string{"mmr"}, nil)), "distance should be 0")
	assert.Equal(t, 1, CountMemberDiff(got), "member diff should be 1")
}

func getPartyIDs(ally models.MatchingAlly) []string {
	partyIDs := make([]string, 0)
	for _, party := range ally.MatchingParties {
		partyIDs = append(partyIDs, party.PartyID)
	}
	return partyIDs
}

func Test_permutations(t *testing.T) {
	type args struct {
		arr []int
	}
	tests := []struct {
		name string
		args args
		want [][]int
	}{
		{
			name: "test_1",
			args: args{
				arr: []int{
					0, 1, 2,
				},
			},
			want: [][]int{
				{0, 1, 2},
				{1, 0, 2},
				{2, 1, 0},
				{1, 2, 0},
				{2, 0, 1},
				{0, 2, 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := permutations(tt.args.arr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("permutations() = %v, want %v", got, tt.want)
			}
		})
	}
}
