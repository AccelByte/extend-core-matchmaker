// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v2

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance/rebalance_v1"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"

	"github.com/stretchr/testify/assert"
)

var (
	mmr = "mmr"

	mockActiveAllianceRule = models.AllianceRule{
		MinNumber:       2,
		MaxNumber:       2,
		PlayerMinNumber: 1,
		PlayerMaxNumber: 10,
	}

	mockMatchingRules = []models.MatchingRule{
		{
			Attribute: mmr,
			Criteria:  constants.DistanceCriteria,
			Reference: 100,
		},
	}
)

func matchingParty(members []models.PartyMember, locked ...bool) models.MatchingParty {
	var matchingParty models.MatchingParty
	matchingParty.PartyMembers = members
	if locked != nil {
		matchingParty.Locked = locked[0]
	}
	matchingParty.PartyAttributes = partyAttributes(matchingParty)
	return matchingParty
}

func partyAttributes(party models.MatchingParty) map[string]interface{} {
	partyAttributes := party.PartyAttributes
	if partyAttributes == nil {
		partyAttributes = make(map[string]interface{})
	}
	memberAttributes, ok := partyAttributes[models.AttributeMemberAttr].(map[string]interface{})
	if !ok || memberAttributes == nil {
		memberAttributes = make(map[string]interface{})
	}
	extraAttributes := make(map[string]struct{})
	for _, member := range party.PartyMembers {
		for k := range member.ExtraAttributes {
			extraAttributes[k] = struct{}{}
		}
	}
	for attrName := range extraAttributes {
		var total float64
		for _, member := range party.PartyMembers {
			v, ok := member.ExtraAttributes[attrName]
			if ok {
				vstr, ok := v.(string)
				if !ok {
					vstr = fmt.Sprint(v)
				}
				vfloat, _ := strconv.ParseFloat(vstr, 64)
				total += vfloat
			}
		}
		memberAttributes[attrName] = total / float64(party.CountPlayer())
	}
	partyAttributes[models.AttributeMemberAttr] = memberAttributes
	return partyAttributes
}

func generateMatchingParty(countMember int, avgMmr float64, locked bool) models.MatchingParty {
	return generateMatchingPartyWithID(utils.GenerateUUID(), countMember, avgMmr, locked)
}

func generateMatchingPartyWithID(partyID string, countMember int, avgMmr float64, locked bool) models.MatchingParty {
	partyMembers := make([]models.PartyMember, countMember)
	for i := 0; i < countMember; i++ {
		partyMembers[i] = models.PartyMember{
			UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{mmr: avgMmr},
		}
	}
	party := models.MatchingParty{
		PartyID:      partyID,
		PartyMembers: partyMembers,
		Locked:       locked,
	}
	party.PartyAttributes = partyAttributes(party)
	return party
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

func Test_RebalanceV2_1(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "Test_RebalanceV2_1", "")
	defer scope.Finish()

	party1 := models.MatchingParty{
		PartyID: "party1",
		PartyMembers: []models.PartyMember{
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{mmr: 20.0}},
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{mmr: 30.0}},
		},
		Locked: true,
	}
	party1.PartyAttributes = partyAttributes(party1)
	party2 := generateMatchingPartyWithID("party2", 2, 10, true)
	party3 := generateMatchingPartyWithID("party3", 1, 20, false)

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

	got := RebalanceV2(scope, "", allies, mockActiveAllianceRule, mockMatchingRules, "")
	assert.Equal(t, wantAllies, got, "")
}

func Test_RebalanceV2_2(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "Test_RebalanceV2_2", "")
	defer scope.Finish()

	// assign locked parties (current session)
	party1 := generateMatchingPartyWithID("party1", 2, 10.0, true)
	ally1 := models.MatchingAlly{MatchingParties: []models.MatchingParty{party1}}

	party2 := generateMatchingPartyWithID("party2", 3, 20.0, true)
	ally2 := models.MatchingAlly{MatchingParties: []models.MatchingParty{party2}}

	// assign unlocked parties (new parties)
	party3 := generateMatchingPartyWithID("party3", 1, 10.0, false)
	party4 := generateMatchingPartyWithID("party4", 3, 20.0, false)
	party5 := generateMatchingPartyWithID("party5", 2, 30.0, false)
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

	got := RebalanceV2(scope, "", allies, mockActiveAllianceRule, mockMatchingRules, "")
	assert.ElementsMatch(t, getPartyIDs(wantAllies[0]), getPartyIDs(got[0]), "ally 1 should equal")
	assert.ElementsMatch(t, getPartyIDs(wantAllies[1]), getPartyIDs(got[1]), "ally 2 should equal")
}

func Test_RebalanceV2_Locked(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "Test_RebalanceV2_Locked", "")
	defer scope.Finish()

	// assign locked parties (current session)
	party1 := generateMatchingParty(2, 10.0, true)
	party1.PartyID = "party1"
	ally1 := models.MatchingAlly{MatchingParties: []models.MatchingParty{party1}}

	party2 := generateMatchingParty(3, 20.0, true)
	party2.PartyID = "party2"
	ally2 := models.MatchingAlly{MatchingParties: []models.MatchingParty{party2}}

	// assign unlocked parties (new parties)
	party3 := generateMatchingParty(2, 10.0, false)
	party3.PartyID = "party3"
	party4 := generateMatchingParty(3, 5.0, false)
	party4.PartyID = "party4"
	party5 := generateMatchingParty(2, 30.0, false)
	party5.PartyID = "party5"
	newParties := []models.MatchingParty{party3, party4, party5}
	ally2.MatchingParties = append(ally2.MatchingParties, newParties...)

	allies := []models.MatchingAlly{
		ally1,
		ally2,
	}

	got := RebalanceV2(scope, "", allies, mockActiveAllianceRule, mockMatchingRules, "")
	assert.Contains(t, getPartyIDs(got[0]), party1.PartyID, "party1 should be locked")
	assert.Contains(t, getPartyIDs(got[1]), party2.PartyID, "party2 should be locked")

	oldDistance := countDistance(allies, []string{mmr}, nil)
	newDistance := countDistance(allies, []string{mmr}, nil)
	assert.LessOrEqual(t, newDistance, oldDistance)
}

func Test_RebalanceV2_NoNewParties(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "Test_RebalanceV2_NoNewParties", "")
	defer scope.Finish()

	party1 := models.MatchingParty{
		PartyID: utils.GenerateUUID(),
		PartyMembers: []models.PartyMember{
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{mmr: 20.0}},
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{mmr: 30.0}},
		},
		Locked: true,
	}
	party1.PartyAttributes = partyAttributes(party1)
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

	got := RebalanceV2(scope, "", allies, mockActiveAllianceRule, mockMatchingRules, "")
	assert.Equal(t, math.Round(countDistance(allies, []string{mmr}, nil)), math.Round(countDistance(got, []string{mmr}, nil)), "distance should be 0")
	assert.Equal(t, 13.0, math.Round(countDistance(got, []string{mmr}, nil)), "distance should be 0")
	assert.Equal(t, 1, rebalance_v1.CountMemberDiff(got), "member diff should be 1")
}

func Test_RebalanceV2_NoSession(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "Test_RebalanceV2_NoSession", "")
	defer scope.Finish()

	party1 := models.MatchingParty{
		PartyID: utils.GenerateUUID(),
		PartyMembers: []models.PartyMember{
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{mmr: 20.0}},
			{UserID: utils.GenerateUUID(), ExtraAttributes: map[string]interface{}{mmr: 30.0}},
		},
		Locked: false,
	}
	party1.PartyAttributes = partyAttributes(party1)
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

	got := RebalanceV2(scope, "", allies, mockActiveAllianceRule, mockMatchingRules, "")
	assert.Greater(t, math.Round(countDistance(allies, []string{mmr}, nil)), math.Round(countDistance(got, []string{mmr}, nil)), "distance should be greater")
	assert.Equal(t, 12.0, math.Round(countDistance(got, []string{mmr}, nil)), "distance should be 12")
	assert.Equal(t, 1, rebalance_v1.CountMemberDiff(got), "member diff should be 1")
}

func getPartyIDs(ally models.MatchingAlly) []string {
	partyIDs := make([]string, 0)
	for _, party := range ally.MatchingParties {
		partyIDs = append(partyIDs, party.PartyID)
	}
	return partyIDs
}

func TestRebalanceV2_PositiveAndNegativeCase(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceV2_PositiveAndNegativeCase", "")
	defer scope.Finish()

	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name               string
		args               args
		want               []models.MatchingAlly
		wantBetterDistance bool
	}{
		{
			name: "rebalance done - different min max player",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 9}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 7}},
						}),
					}},
					{MatchingParties: []models.MatchingParty{}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
					}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 9}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 7}},
					}),
				}},
			},
			wantBetterDistance: true,
		},
		{
			name: "rebalance done - equal min max player",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 2,
					PlayerMaxNumber: 2,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 9}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 7}},
						}),
					}},
					{MatchingParties: []models.MatchingParty{}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 7}},
					}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 9}},
					}),
				}},
			},
			wantBetterDistance: true,
		},
		{
			name: "rebalance done - 10 parties",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 5,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1577}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 2105}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 2113}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 3723}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 3768}},
						}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 4511}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "g", ExtraAttributes: map[string]interface{}{mmr: 4773}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "h", ExtraAttributes: map[string]interface{}{mmr: 5231}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "i", ExtraAttributes: map[string]interface{}{mmr: 5791}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "j", ExtraAttributes: map[string]interface{}{mmr: 6548}},
						}),
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 2105}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 2113}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 4511}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "g", ExtraAttributes: map[string]interface{}{mmr: 4773}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "j", ExtraAttributes: map[string]interface{}{mmr: 6548}},
					}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1577}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 3723}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 3768}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "h", ExtraAttributes: map[string]interface{}{mmr: 5231}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "i", ExtraAttributes: map[string]interface{}{mmr: 5791}},
					}),
				}},
			},
			wantBetterDistance: true,
		},
		{
			name: "rebalance done - without attribute",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 9}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 7}},
						}),
					}},
					{MatchingParties: []models.MatchingParty{}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
					}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 9}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 7}},
					}),
				}},
			},
			wantBetterDistance: false,
		},
		{
			name: "rebalance no team to rebalance",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       1,
					MaxNumber:       1,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 9}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 7}},
						}),
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 9}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 7}},
					}),
				}},
			},
			wantBetterDistance: false,
		},
		{
			name: "rebalance no parties to rebalance",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 2,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
						}, true),
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 9}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 7}},
						}, true),
					}},
					{MatchingParties: []models.MatchingParty{}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 9}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 7}},
					}),
				}},
				{MatchingParties: []models.MatchingParty{}},
			},
			wantBetterDistance: false,
		},
		{
			name: "rebalance difference already 0",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 2,
					PlayerMaxNumber: 2,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
						}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 10}},
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 8}},
						}),
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 10}},
						{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 8}},
					}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 10}},
						{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 8}},
					}),
				}},
			},
			wantBetterDistance: false,
		},
		{
			name: "rebalance done - 9 parties in 5v5",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 5,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1838}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 1841}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 1762}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 1936}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 1884}},
						}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 1655}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "g", ExtraAttributes: map[string]interface{}{mmr: 1678}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "h", ExtraAttributes: map[string]interface{}{mmr: 1661}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "i", ExtraAttributes: map[string]interface{}{mmr: 1597}},
							{UserID: "j", ExtraAttributes: map[string]interface{}{mmr: 1609}},
						}),
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 1841}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 1884}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 1655}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "g", ExtraAttributes: map[string]interface{}{mmr: 1678}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "h", ExtraAttributes: map[string]interface{}{mmr: 1661}},
					}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1838}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 1762}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 1936}},
					}),
					matchingParty([]models.PartyMember{
						{UserID: "i", ExtraAttributes: map[string]interface{}{mmr: 1597}},
						{UserID: "j", ExtraAttributes: map[string]interface{}{mmr: 1609}},
					}),
				}},
			},
			wantBetterDistance: true,
		},
		{
			name: "rebalance done - 9 parties in 3v3v3",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       3,
					MaxNumber:       3,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 3,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("a", 3280)}),
						matchingParty([]models.PartyMember{playerWithMMR("b", 1020)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3010)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("d", 1090)}),
						matchingParty([]models.PartyMember{playerWithMMR("e", 1230)}),
						matchingParty([]models.PartyMember{playerWithMMR("f", 3240)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("g", 1570)}),
						matchingParty([]models.PartyMember{playerWithMMR("h", 1560)}),
						matchingParty([]models.PartyMember{playerWithMMR("i", 1550)}),
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("d", 1090)}),
					matchingParty([]models.PartyMember{playerWithMMR("f", 3240)}),
					matchingParty([]models.PartyMember{playerWithMMR("i", 1550)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("c", 3010)}),
					matchingParty([]models.PartyMember{playerWithMMR("e", 1230)}),
					matchingParty([]models.PartyMember{playerWithMMR("g", 1570)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("a", 3280)}),
					matchingParty([]models.PartyMember{playerWithMMR("b", 1020)}),
					matchingParty([]models.PartyMember{playerWithMMR("h", 1560)}),
				}},
			},
			wantBetterDistance: true,
		},
		{
			name: "rebalance done - 9 parties in 3v3v3 - reorder2",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       3,
					MaxNumber:       3,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 3,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("d", 1090)}),
						matchingParty([]models.PartyMember{playerWithMMR("f", 3240)}),
						matchingParty([]models.PartyMember{playerWithMMR("e", 1230)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("a", 3280)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3010)}),
						matchingParty([]models.PartyMember{playerWithMMR("g", 1570)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("i", 1550)}),
						matchingParty([]models.PartyMember{playerWithMMR("b", 1020)}),
						matchingParty([]models.PartyMember{playerWithMMR("h", 1560)}),
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("d", 1090)}),
					matchingParty([]models.PartyMember{playerWithMMR("f", 3240)}),
					matchingParty([]models.PartyMember{playerWithMMR("i", 1550)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("c", 3010)}),
					matchingParty([]models.PartyMember{playerWithMMR("e", 1230)}),
					matchingParty([]models.PartyMember{playerWithMMR("g", 1570)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("a", 3280)}),
					matchingParty([]models.PartyMember{playerWithMMR("b", 1020)}),
					matchingParty([]models.PartyMember{playerWithMMR("h", 1560)}),
				}},
			},
			wantBetterDistance: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
			distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)

			got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")
			for _, gotAlly := range got {
				gotMember := make(map[string]map[string]interface{})
				for _, gotParty := range gotAlly.MatchingParties {
					for _, m := range gotParty.PartyMembers {
						gotMember[m.UserID] = m.ExtraAttributes
					}
				}

				var exist bool
				for _, wantAlly := range tt.want {
					wantMember := make(map[string]map[string]interface{})
					for _, wantParty := range wantAlly.MatchingParties {
						for _, m := range wantParty.PartyMembers {
							wantMember[m.UserID] = m.ExtraAttributes
						}
					}
					if reflect.DeepEqual(gotMember, wantMember) {
						exist = true
						break
					}
				}

				if !exist {
					t.Errorf("TestRebalanceV2_PositiveAndNegativeCase() got = %v", gotMember)
				}
			}

			newDistance := countDistance(got, attr, tt.args.matchingRules)
			gotBetterDistance := newDistance < distance
			if gotBetterDistance != tt.wantBetterDistance {
				t.Errorf("TestRebalanceV2_PositiveAndNegativeCase() distance = %v , newDistance %v", distance, newDistance)
			} else {
				t.Logf("got better distance: %f -> %f", distance, newDistance)
			}
		})
	}
}

func TestRebalanceV2_5v5(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceV2_PositiveAndNegativeCase", "")
	defer scope.Finish()

	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name               string
		args               args
		want               []models.MatchingAlly
		wantBetterDistance bool
		wantDistance       int
	}{
		{
			name: "rebalance done - 10 parties in 5v5",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("a", 1)}),
						matchingParty([]models.PartyMember{playerWithMMR("b", 2)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3)}),
						matchingParty([]models.PartyMember{playerWithMMR("d", 4)}),
						matchingParty([]models.PartyMember{playerWithMMR("e", 5)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("f", 6)}),
						matchingParty([]models.PartyMember{playerWithMMR("g", 7)}),
						matchingParty([]models.PartyMember{playerWithMMR("h", 8)}),
						matchingParty([]models.PartyMember{playerWithMMR("i", 9)}),
						matchingParty([]models.PartyMember{playerWithMMR("j", 10)}),
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("a", 1)}),
					matchingParty([]models.PartyMember{playerWithMMR("d", 4)}),
					matchingParty([]models.PartyMember{playerWithMMR("e", 5)}),
					matchingParty([]models.PartyMember{playerWithMMR("h", 8)}),
					matchingParty([]models.PartyMember{playerWithMMR("i", 9)}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{playerWithMMR("b", 2)}),
					matchingParty([]models.PartyMember{playerWithMMR("c", 3)}),
					matchingParty([]models.PartyMember{playerWithMMR("f", 6)}),
					matchingParty([]models.PartyMember{playerWithMMR("g", 7)}),
					matchingParty([]models.PartyMember{playerWithMMR("j", 10)}),
				}},
			},
			wantDistance:       0,
			wantBetterDistance: true,
		},
		{
			name: "rebalance done - 9 parties in 5v5",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("a", 1)}),
						matchingParty([]models.PartyMember{playerWithMMR("b", 2)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3)}),
						matchingParty([]models.PartyMember{playerWithMMR("d", 4)}),
						matchingParty([]models.PartyMember{playerWithMMR("e", 5)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("f", 6)}),
						matchingParty([]models.PartyMember{playerWithMMR("g", 7)}),
						matchingParty([]models.PartyMember{playerWithMMR("h", 8)}),
						matchingParty([]models.PartyMember{playerWithMMR("i", 9)}),
					}},
				},
			},
			wantDistance:       0,
			wantBetterDistance: true,
		},
		{
			name: "rebalance done - 8 parties in 5v5",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("a", 1)}),
						matchingParty([]models.PartyMember{playerWithMMR("b", 2)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3)}),
						matchingParty([]models.PartyMember{playerWithMMR("d", 4)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("e", 5)}),
						matchingParty([]models.PartyMember{playerWithMMR("f", 6)}),
						matchingParty([]models.PartyMember{playerWithMMR("g", 7)}),
						matchingParty([]models.PartyMember{playerWithMMR("h", 8)}),
					}},
				},
			},
			wantDistance:       0,
			wantBetterDistance: true,
		},
		{
			name: "rebalance done - 7 parties in 5v5",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("a", 2)}),
						matchingParty([]models.PartyMember{playerWithMMR("b", 2)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3)}),
						matchingParty([]models.PartyMember{playerWithMMR("d", 5)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("e", 5)}),
						matchingParty([]models.PartyMember{playerWithMMR("f", 6)}),
						matchingParty([]models.PartyMember{playerWithMMR("g", 7)}),
					}},
				},
			},
			wantDistance:       1,
			wantBetterDistance: true,
		},
		{
			name: "rebalance done - 6 parties in 5v5",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("a", 1)}),
						matchingParty([]models.PartyMember{playerWithMMR("b", 2)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("e", 5)}),
						matchingParty([]models.PartyMember{playerWithMMR("f", 6)}),
						matchingParty([]models.PartyMember{playerWithMMR("g", 7)}),
					}},
				},
			},
			wantDistance:       0,
			wantBetterDistance: true,
		},
		{
			name: "not rebalanced - 5 parties in 5v5 - insufficient num player",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("a", 1)}),
						matchingParty([]models.PartyMember{playerWithMMR("b", 2)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("e", 5)}),
						matchingParty([]models.PartyMember{playerWithMMR("f", 6)}),
					}},
				},
			},
			wantBetterDistance: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
			distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)

			got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")
			if tt.want != nil {
				for _, gotAlly := range got {
					gotMember := make(map[string]map[string]interface{})
					for _, gotParty := range gotAlly.MatchingParties {
						for _, m := range gotParty.PartyMembers {
							gotMember[m.UserID] = m.ExtraAttributes
						}
					}

					var exist bool
					for _, wantAlly := range tt.want {
						wantMember := make(map[string]map[string]interface{})
						for _, wantParty := range wantAlly.MatchingParties {
							for _, m := range wantParty.PartyMembers {
								wantMember[m.UserID] = m.ExtraAttributes
							}
						}
						if reflect.DeepEqual(gotMember, wantMember) {
							exist = true
							break
						}
					}

					if !exist {
						t.Errorf("TestRebalanceV2_PositiveAndNegativeCase() got = %v", gotMember)
					}
				}
			}

			newDistance := countDistance(got, attr, tt.args.matchingRules)
			gotBetterDistance := newDistance < distance
			stats := fmt.Sprintf("distance=%v newDistance=%v sum=[%v vs %v] newSum=[%v vs %v]",
				distance, newDistance,
				tt.args.allies[0].Total(attr, tt.args.matchingRules), tt.args.allies[1].Total(attr, tt.args.matchingRules),
				got[0].Total(attr, tt.args.matchingRules), got[1].Total(attr, tt.args.matchingRules))
			if gotBetterDistance != tt.wantBetterDistance {
				t.Errorf("TestRebalanceV2_PositiveAndNegativeCase() %v", stats)
			} else {
				t.Logf("got better distance: %f -> %f", distance, newDistance)
			}

			if tt.wantBetterDistance {
				assert.LessOrEqual(t, int(newDistance), tt.wantDistance, stats)
			}
		})
	}
}

func playerWithMMR(userID string, mmrVal float64) models.PartyMember {
	return models.PartyMember{UserID: userID, ExtraAttributes: map[string]interface{}{mmr: mmrVal}}
}

func playerWith2MMR(userID, mmr1Name, mmr2Name string, mmr1Val, mmr2Val float64) models.PartyMember {
	return models.PartyMember{UserID: userID, ExtraAttributes: map[string]interface{}{mmr1Name: mmr1Val, mmr2Name: mmr2Val}}
}

func TestRebalanceV2_CompareDifference(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceV2_CompareDifference", "")
	defer scope.Finish()

	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name                       string
		args                       args
		want                       []models.MatchingAlly
		wantBetterDistance         bool
		wantDiffByPartyAttributes  int
		wantDiffByMemberAttributes int
	}{
		{
			name: "AvgByMemberAttributes",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 5,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 2028}},
								{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 2078}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 2057}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 1857}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 1793}},
							}),
						},
					},
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 1926}},
								{UserID: "g", ExtraAttributes: map[string]interface{}{mmr: 1921}},
								{UserID: "h", ExtraAttributes: map[string]interface{}{mmr: 1938}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "i", ExtraAttributes: map[string]interface{}{mmr: 1905}},
								{UserID: "j", ExtraAttributes: map[string]interface{}{mmr: 1852}},
							}),
						},
					},
				},
			},
			want: []models.MatchingAlly{
				{
					MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 2028}},
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 2078}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 1793}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "i", ExtraAttributes: map[string]interface{}{mmr: 1905}},
							{UserID: "j", ExtraAttributes: map[string]interface{}{mmr: 1852}},
						}),
					},
				},
				{
					MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 1857}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 2057}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 1926}},
							{UserID: "g", ExtraAttributes: map[string]interface{}{mmr: 1921}},
							{UserID: "h", ExtraAttributes: map[string]interface{}{mmr: 1938}},
						}),
					},
				},
			},
			wantBetterDistance:         true,
			wantDiffByPartyAttributes:  8,
			wantDiffByMemberAttributes: 8,
		},
		{
			name: "AvgByPartyAttributes",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 5,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{
						MatchingParties: []models.MatchingParty{
							{
								PartyMembers: []models.PartyMember{
									// a has 0 to trigger count mmr difference using party attributes
									{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 0}}, // 2028
									{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 2078}},
								},
								PartyAttributes: map[string]interface{}{
									models.AttributeMemberAttr: map[string]interface{}{mmr: 2053},
								},
							},
							matchingParty([]models.PartyMember{
								{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 2057}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 1857}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 1793}},
							}),
						},
					},
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 1926}},
								{UserID: "g", ExtraAttributes: map[string]interface{}{mmr: 1921}},
								{UserID: "h", ExtraAttributes: map[string]interface{}{mmr: 1938}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "i", ExtraAttributes: map[string]interface{}{mmr: 1905}},
								{UserID: "j", ExtraAttributes: map[string]interface{}{mmr: 1852}},
							}),
						},
					},
				},
			},
			want: []models.MatchingAlly{
				{
					MatchingParties: []models.MatchingParty{
						{
							PartyMembers: []models.PartyMember{
								// a has 0 to trigger count mmr difference using party attributes
								{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 0}}, // 2028
								{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 2078}},
							},
							PartyAttributes: map[string]interface{}{
								models.AttributeMemberAttr: map[string]interface{}{mmr: 2053},
							},
						},
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 2057}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "i", ExtraAttributes: map[string]interface{}{mmr: 1905}},
							{UserID: "j", ExtraAttributes: map[string]interface{}{mmr: 1852}},
						}),
					},
				},
				{
					MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 1857}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 1793}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 1926}},
							{UserID: "g", ExtraAttributes: map[string]interface{}{mmr: 1921}},
							{UserID: "h", ExtraAttributes: map[string]interface{}{mmr: 1938}},
						}),
					},
				},
			},
			wantBetterDistance:         true,
			wantDiffByPartyAttributes:  308,
			wantDiffByMemberAttributes: 351,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
			distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)

			got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")
			for _, gotAlly := range got {
				gotMember := make(map[string]map[string]interface{})
				for _, gotParty := range gotAlly.MatchingParties {
					for _, m := range gotParty.PartyMembers {
						gotMember[m.UserID] = m.ExtraAttributes
					}
				}

				var exist bool
				for _, wantAlly := range tt.want {
					wantMember := make(map[string]map[string]interface{})
					for _, wantParty := range wantAlly.MatchingParties {
						for _, m := range wantParty.PartyMembers {
							wantMember[m.UserID] = m.ExtraAttributes
						}
					}
					if reflect.DeepEqual(gotMember, wantMember) {
						exist = true
						break
					}
				}

				if !exist {
					t.Errorf("TestRebalanceV2_CompareDifference() got = %v", gotMember)
				}
			}

			newDistance := countDistance(got, attr, tt.args.matchingRules)
			gotBetterDistance := newDistance < distance
			if gotBetterDistance != tt.wantBetterDistance {
				t.Errorf("TestRebalanceV2_CompareDifference() distance = %v , newDistance %v", distance, newDistance)
			}

			diffByPartyAttributes := int(math.Abs(got[0].Avg([]string{mmr}, tt.args.matchingRules) - got[1].Avg([]string{mmr}, tt.args.matchingRules)))
			if diffByPartyAttributes != tt.wantDiffByPartyAttributes {
				t.Errorf("TestRebalanceV2_CompareDifference() got = %v , wantDiffByPartyAttributes %v", diffByPartyAttributes, tt.wantDiffByPartyAttributes)
			}
		})
	}
}

func Test_removeEmptyMatchingParties(t *testing.T) {
	type args struct {
		allies []models.MatchingAlly
	}
	tests := []struct {
		name string
		args args
		want []models.MatchingAlly
	}{
		{
			name: "Empty first matching parties",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: nil},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1841}},
						}),
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1841}},
					}),
				}},
			},
		}, {
			name: "Empty last matching parties",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1841}},
						}),
					}},
					{MatchingParties: nil},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1841}},
					}),
				}},
			},
		}, {
			name: "Empty middle matching parties",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1841}},
						}),
					}},
					{MatchingParties: nil},
					{MatchingParties: nil},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 1841}},
						}),
					}},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1841}},
					}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 1841}},
					}),
				}},
			},
		}, {
			name: "Empty first, middle and last matching parties",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: nil},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1841}},
						}),
					}},
					{MatchingParties: nil},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 1841}},
						}),
					}},
					{MatchingParties: nil},
				},
			},
			want: []models.MatchingAlly{
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 1841}},
					}),
				}},
				{MatchingParties: []models.MatchingParty{
					matchingParty([]models.PartyMember{
						{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 1841}},
					}),
				}},
			},
		}, {
			name: "Empty all matching parties",
			args: args{
				allies: []models.MatchingAlly{
					{MatchingParties: nil},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{}),
					}},
					{MatchingParties: nil},
				},
			},
			want: []models.MatchingAlly{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveEmptyMatchingParties(tt.args.allies)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_permutation(t *testing.T) {
	type args struct {
		input  [][]string
		result [][]string
		s      []string
	}
	tests := []struct {
		name string
		args args
		want [][]string
	}{
		{
			name: "test 1",
			args: args{
				input: [][]string{
					{"support", "tank"},
					{"fighter"},
					{"marksman", "support"},
					{"tank", "fighter"},
				},
			},
			want: [][]string{
				{"support", "fighter", "marksman", "tank"},
				{"support", "fighter", "marksman", "fighter"},
				{"support", "fighter", "support", "tank"},
				{"support", "fighter", "support", "fighter"},
				{"tank", "fighter", "marksman", "tank"},
				{"tank", "fighter", "marksman", "fighter"},
				{"tank", "fighter", "support", "tank"},
				{"tank", "fighter", "support", "fighter"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := permutation(tt.args.input, tt.args.result, tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("permutation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRebalanceV2_PermutationExamples(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceV2_PermutationExamples", "")
	defer scope.Finish()
	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name                       string
		args                       args
		want                       []models.MatchingAlly
		wantBetterDistance         bool
		wantDiffByPartyAttributes  int
		wantDiffByMemberAttributes int
	}{
		{
			name: "PermutationExample",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       1,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 3,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 80}},
							}, true),
							matchingParty([]models.PartyMember{
								{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 50}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 10}},
							}),
						},
					},
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 40}},
								{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 40}},
							}),
						},
					},
				},
			},
			want: []models.MatchingAlly{
				{
					MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 80}},
						}, true),
						matchingParty([]models.PartyMember{
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 10}},
						}),
					},
				},
				{
					MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 50}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 40}},
							{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 40}},
						}),
					},
				},
			},
			wantBetterDistance:         true,
			wantDiffByPartyAttributes:  1,
			wantDiffByMemberAttributes: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
			distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)
			got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")
			for _, gotAlly := range got {
				gotMember := make(map[string]map[string]interface{})
				for _, gotParty := range gotAlly.MatchingParties {
					for _, m := range gotParty.PartyMembers {
						gotMember[m.UserID] = m.ExtraAttributes
					}
				}
				var exist bool
				for _, wantAlly := range tt.want {
					wantMember := make(map[string]map[string]interface{})
					for _, wantParty := range wantAlly.MatchingParties {
						for _, m := range wantParty.PartyMembers {
							wantMember[m.UserID] = m.ExtraAttributes
						}
					}
					if reflect.DeepEqual(gotMember, wantMember) {
						exist = true
						break
					}
				}
				if !exist {
					t.Errorf("TestRebalanceV2_CompareDifference() got = %v", gotMember)
				}
			}
			newDistance := countDistance(got, attr, tt.args.matchingRules)
			gotBetterDistance := newDistance < distance
			if gotBetterDistance != tt.wantBetterDistance {
				t.Errorf("TestRebalanceV2_CompareDifference() distance = %v , newDistance %v", distance, newDistance)
			}
			diffByPartyAttributes := int(math.Abs(got[0].Avg([]string{mmr}, tt.args.matchingRules) - got[1].Avg([]string{mmr}, tt.args.matchingRules)))
			if diffByPartyAttributes != tt.wantDiffByPartyAttributes {
				t.Errorf("TestRebalanceV2_CompareDifference() got = %v , wantDiffByPartyAttributes %v", diffByPartyAttributes, tt.wantDiffByPartyAttributes)
			}
		})
	}
}

func TestRebalanceV2_CombinationExamples(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceV2_CombinationExamples", "")
	defer scope.Finish()
	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name                       string
		args                       args
		want                       []models.MatchingAlly
		wantBetterDistance         bool
		wantDiffByPartyAttributes  int
		wantDiffByMemberAttributes int
	}{
		{
			name: "CombinationExample",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 3,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 70}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 60}},
								{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 60}},
							}),
						},
					},
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 40}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 40}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 10}},
							}),
						},
					},
				},
			},
			want: []models.MatchingAlly{
				{
					MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 60}},
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 60}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 10}},
						}),
					},
				},
				{
					MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 70}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 40}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 40}},
						}),
					},
				},
			},
			wantBetterDistance:         true,
			wantDiffByPartyAttributes:  6,
			wantDiffByMemberAttributes: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
			distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)
			got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")
			for _, gotAlly := range got {
				gotMember := make(map[string]map[string]interface{})
				for _, gotParty := range gotAlly.MatchingParties {
					for _, m := range gotParty.PartyMembers {
						gotMember[m.UserID] = m.ExtraAttributes
					}
				}
				var exist bool
				for _, wantAlly := range tt.want {
					wantMember := make(map[string]map[string]interface{})
					for _, wantParty := range wantAlly.MatchingParties {
						for _, m := range wantParty.PartyMembers {
							wantMember[m.UserID] = m.ExtraAttributes
						}
					}
					if reflect.DeepEqual(gotMember, wantMember) {
						exist = true
						break
					}
				}
				if !exist {
					t.Errorf("TestRebalanceV2_CompareDifference() got = %v", gotMember)
				}
			}
			newDistance := countDistance(got, attr, tt.args.matchingRules)
			gotBetterDistance := newDistance < distance
			if gotBetterDistance != tt.wantBetterDistance {
				t.Errorf("TestRebalanceV2_CompareDifference() distance = %v , newDistance %v", distance, newDistance)
			}
			diffByPartyAttributes := int(math.Abs(got[0].Avg([]string{mmr}, tt.args.matchingRules) - got[1].Avg([]string{mmr}, tt.args.matchingRules)))
			if diffByPartyAttributes != tt.wantDiffByPartyAttributes {
				t.Errorf("TestRebalanceV2_CompareDifference() got = %v , wantDiffByPartyAttributes %v", diffByPartyAttributes, tt.wantDiffByPartyAttributes)
			}
		})
	}
}

func TestRebalanceV2_PartitioningExamples(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceV2_PartitioningExamples", "")
	defer scope.Finish()
	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name                       string
		args                       args
		want                       []models.MatchingAlly
		wantBetterDistance         bool
		wantDiffByPartyAttributes  int
		wantDiffByMemberAttributes int
	}{
		{
			name: "PartitioningExample",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       2,
					MaxNumber:       2,
					PlayerMinNumber: 7,
					PlayerMaxNumber: 7,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 80}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 40}},
								{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 40}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 40}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 60}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "g", ExtraAttributes: map[string]interface{}{mmr: 70}},
								{UserID: "h", ExtraAttributes: map[string]interface{}{mmr: 70}},
							}),
						},
					},
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 50}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "i", ExtraAttributes: map[string]interface{}{mmr: 60}},
								{UserID: "j", ExtraAttributes: map[string]interface{}{mmr: 60}},
								{UserID: "k", ExtraAttributes: map[string]interface{}{mmr: 60}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "l", ExtraAttributes: map[string]interface{}{mmr: 50}},
								{UserID: "m", ExtraAttributes: map[string]interface{}{mmr: 50}},
								{UserID: "n", ExtraAttributes: map[string]interface{}{mmr: 50}},
							}),
						},
					},
				},
			},
			want: []models.MatchingAlly{
				{
					MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 80}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 40}},
							{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 40}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "f", ExtraAttributes: map[string]interface{}{mmr: 50}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "i", ExtraAttributes: map[string]interface{}{mmr: 60}},
							{UserID: "j", ExtraAttributes: map[string]interface{}{mmr: 60}},
							{UserID: "k", ExtraAttributes: map[string]interface{}{mmr: 60}},
						}),
					},
				},
				{
					MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{
							{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 40}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 60}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "g", ExtraAttributes: map[string]interface{}{mmr: 70}},
							{UserID: "h", ExtraAttributes: map[string]interface{}{mmr: 70}},
						}),
						matchingParty([]models.PartyMember{
							{UserID: "l", ExtraAttributes: map[string]interface{}{mmr: 50}},
							{UserID: "m", ExtraAttributes: map[string]interface{}{mmr: 50}},
							{UserID: "n", ExtraAttributes: map[string]interface{}{mmr: 50}},
						}),
					},
				},
			},
			wantBetterDistance:         true,
			wantDiffByPartyAttributes:  0,
			wantDiffByMemberAttributes: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
			distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)
			got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")
			for _, gotAlly := range got {
				gotMember := make(map[string]map[string]interface{})
				for _, gotParty := range gotAlly.MatchingParties {
					for _, m := range gotParty.PartyMembers {
						gotMember[m.UserID] = m.ExtraAttributes
					}
				}
				var exist bool
				for _, wantAlly := range tt.want {
					wantMember := make(map[string]map[string]interface{})
					for _, wantParty := range wantAlly.MatchingParties {
						for _, m := range wantParty.PartyMembers {
							wantMember[m.UserID] = m.ExtraAttributes
						}
					}
					if reflect.DeepEqual(gotMember, wantMember) {
						exist = true
						break
					}
				}
				if !exist {
					t.Errorf("TestRebalanceV2_CompareDifference() got = %v", gotMember)
				}
			}
			newDistance := countDistance(got, attr, tt.args.matchingRules)
			gotBetterDistance := newDistance < distance
			if gotBetterDistance != tt.wantBetterDistance {
				t.Errorf("TestRebalanceV2_CompareDifference() distance = %v , newDistance %v", distance, newDistance)
			}
			diffByPartyAttributes := int(math.Abs(got[0].Avg([]string{mmr}, tt.args.matchingRules) - got[1].Avg([]string{mmr}, tt.args.matchingRules)))
			if diffByPartyAttributes != tt.wantDiffByPartyAttributes {
				t.Errorf("TestRebalanceV2_CompareDifference() got = %v , wantDiffByPartyAttributes %v", diffByPartyAttributes, tt.wantDiffByPartyAttributes)
			}
		})
	}
}

func TestRebalanceV2_8a3p(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceV2_8a3p", "")
	defer scope.Finish()

	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name               string
		args               args
		want               []models.MatchingAlly
		wantBetterDistance bool
	}{
		{
			name: "rebalance done - 24 parties in 3v3v3v3v3v3v3v3",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       8,
					MaxNumber:       8,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 3,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("a", 3280)}),
						matchingParty([]models.PartyMember{playerWithMMR("b", 1020)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3010)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("d", 1090)}),
						matchingParty([]models.PartyMember{playerWithMMR("e", 1230)}),
						matchingParty([]models.PartyMember{playerWithMMR("f", 3240)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("g", 1570)}),
						matchingParty([]models.PartyMember{playerWithMMR("h", 1560)}),
						matchingParty([]models.PartyMember{playerWithMMR("i", 1550)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("k", 2870)}),
						matchingParty([]models.PartyMember{playerWithMMR("l", 2160)}),
						matchingParty([]models.PartyMember{playerWithMMR("m", 1350)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("n", 1240)}),
						matchingParty([]models.PartyMember{playerWithMMR("o", 1330)}),
						matchingParty([]models.PartyMember{playerWithMMR("p", 1720)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("n", 2240)}),
						matchingParty([]models.PartyMember{playerWithMMR("o", 3070)}),
						matchingParty([]models.PartyMember{playerWithMMR("p", 2910)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("p", 1810)}),
						matchingParty([]models.PartyMember{playerWithMMR("q", 2910)}),
						matchingParty([]models.PartyMember{playerWithMMR("r", 2740)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("s", 1810)}),
						matchingParty([]models.PartyMember{playerWithMMR("t", 2910)}),
						matchingParty([]models.PartyMember{playerWithMMR("u", 2740)}),
					}},
				},
			},
			wantBetterDistance: true,
		},
		{
			name: "rebalance done - 12 parties in 3v3v3v3",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       4,
					MaxNumber:       4,
					PlayerMinNumber: 3,
					PlayerMaxNumber: 3,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("a", 3280)}),
						matchingParty([]models.PartyMember{playerWithMMR("b", 1020)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3010)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("d", 1090)}),
						matchingParty([]models.PartyMember{playerWithMMR("e", 1230)}),
						matchingParty([]models.PartyMember{playerWithMMR("f", 3240)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("g", 1570)}),
						matchingParty([]models.PartyMember{playerWithMMR("h", 1560)}),
						matchingParty([]models.PartyMember{playerWithMMR("i", 1550)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("k", 2870)}),
						matchingParty([]models.PartyMember{playerWithMMR("l", 2160)}),
						matchingParty([]models.PartyMember{playerWithMMR("m", 1350)}),
					}},
				},
			},
			wantBetterDistance: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
			distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)

			got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")

			newDistance := countDistance(got, attr, tt.args.matchingRules)
			gotBetterDistance := newDistance < distance
			if gotBetterDistance != tt.wantBetterDistance {
				t.Errorf("TestRebalanceV2_PositiveAndNegativeCase() distance = %v , newDistance %v", distance, newDistance)
			} else {
				t.Logf("got better distance: %f -> %f", distance, newDistance)
			}
		})
	}
}

func TestRebalanceV2_RebalanceWithMinimalAlliance(t *testing.T) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalanceV2_RebalanceWithMinimalAlliance", "")
	defer scope.Finish()
	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name                      string
		args                      args
		want                      []models.MatchingAlly
		wantBetterDistance        bool
		wantDiffByPartyAttributes int
		wantDiffByMember          int
	}{
		{
			name: "3v3v3_empty1",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       1,
					MaxNumber:       3,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 3,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 80}},
							}),
						},
					},
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 40}},
							}),
						},
					},
				},
			},
			wantBetterDistance:        false,
			wantDiffByPartyAttributes: 40,
			wantDiffByMember:          0,
		},
		{
			name: "3v3v3_partialFill",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       1,
					MaxNumber:       3,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 3,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 80}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 80}},
							}),
						},
					},
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 40}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 40}},
							}),
						},
					},
				},
			},
			wantBetterDistance:        false,
			wantDiffByPartyAttributes: 40,
			wantDiffByMember:          1,
		},
		{
			name: "3v3v3_partialFill2",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       1,
					MaxNumber:       3,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 3,
				},
				matchingRules: []models.MatchingRule{
					{Attribute: mmr, Criteria: constants.DistanceCriteria},
				},
				allies: []models.MatchingAlly{
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "a", ExtraAttributes: map[string]interface{}{mmr: 80}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "b", ExtraAttributes: map[string]interface{}{mmr: 80}},
							}),
						},
					},
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "c", ExtraAttributes: map[string]interface{}{mmr: 40}},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "d", ExtraAttributes: map[string]interface{}{mmr: 40}},
							}),
						},
					},
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "e", ExtraAttributes: map[string]interface{}{mmr: 10}},
							}),
						},
					},
				},
			},
			wantBetterDistance:        true,
			wantDiffByPartyAttributes: 20,
			wantDiffByMember:          1,
		},
		{
			name: "5v5_without_mmr",
			args: args{
				activeAllianceRule: models.AllianceRule{
					MinNumber:       1,
					MaxNumber:       2,
					PlayerMinNumber: 1,
					PlayerMaxNumber: 5,
				},
				matchingRules: []models.MatchingRule{},
				allies: []models.MatchingAlly{
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "a"},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "b"},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "c"},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "d"},
							}),
							matchingParty([]models.PartyMember{
								{UserID: "e"},
							}),
						},
					},
					{
						MatchingParties: []models.MatchingParty{
							matchingParty([]models.PartyMember{
								{UserID: "f"},
							}),
						},
					},
				},
			},
			wantBetterDistance:        false,
			wantDiffByPartyAttributes: 0,
			wantDiffByMember:          0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
			distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)
			got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")
			newDistance := countDistance(got, attr, tt.args.matchingRules)
			gotBetterDistance := newDistance < distance
			if gotBetterDistance != tt.wantBetterDistance {
				t.Errorf("TestRebalanceV2_CompareDifference() distance = %v , newDistance %v", distance, newDistance)
			}
			diffByPartyAttributes := int(countDistance(got, attr, tt.args.matchingRules))
			assert.Equal(t, tt.wantDiffByPartyAttributes, diffByPartyAttributes)
			if diffByPartyAttributes != tt.wantDiffByPartyAttributes {
				t.Errorf("TestRebalanceV2_CompareDifference() got = %v , wantDiffByPartyAttributes %v", diffByPartyAttributes, tt.wantDiffByPartyAttributes)
			}

			memberDiff := CountMemberDiff(got)
			if memberDiff != tt.wantDiffByMember {
				t.Errorf("TestRebalanceV2_CompareDifference() got = %v , wantDiffByMember %v", memberDiff, tt.wantDiffByMember)
			}

			if tt.want != nil {
				for _, gotAlly := range got {
					gotMember := make(map[string]map[string]interface{})
					for _, gotParty := range gotAlly.MatchingParties {
						for _, m := range gotParty.PartyMembers {
							gotMember[m.UserID] = m.ExtraAttributes
						}
					}

					var exist bool
					for _, wantAlly := range tt.want {
						wantMember := make(map[string]map[string]interface{})
						for _, wantParty := range wantAlly.MatchingParties {
							for _, m := range wantParty.PartyMembers {
								wantMember[m.UserID] = m.ExtraAttributes
							}
						}
						if reflect.DeepEqual(gotMember, wantMember) {
							exist = true
							break
						}
					}

					if !exist {
						t.Errorf("TestRebalanceV2_PositiveAndNegativeCase() got = %v", gotMember)
					}
				}
			}
		})
	}
}
