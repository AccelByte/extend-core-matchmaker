// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v2

import (
	"context"
	"crypto/rand"
	"math/big"
	_ "net/http/pprof"
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/constants"
	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"

	"github.com/sirupsen/logrus"
)

// For v1 benchmark see rebalance_v1_test.go

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

func BenchmarkRebalanceV2_3v3v3(b *testing.B) {
	logrus.SetLevel(logrus.ErrorLevel)
	scope := envelope.NewRootScope(context.Background(), "BenchRebalanceV2_3v3v3", "")
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
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
				distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)

				got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")

				newDistance := countDistance(got, attr, tt.args.matchingRules)
				gotBetterDistance := newDistance < distance
				if gotBetterDistance != tt.wantBetterDistance {
					b.Errorf("TestRebalanceV2_PositiveAndNegativeCase() distance = %v , newDistance %v", distance, newDistance)
				} else {
					b.Logf("got better distance: %f -> %f", distance, newDistance)
				}
			}
		})
	}
}

func BenchmarkRebalanceV2_5v5(b *testing.B) {
	logrus.SetLevel(logrus.ErrorLevel)
	scope := envelope.NewRootScope(context.Background(), "BenchmarkRebalanceV2_5v5", "")
	defer scope.Finish()

	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name               string
		args               args
		wantBetterDistance bool
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
						matchingParty([]models.PartyMember{playerWithMMR("b", 1)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 2)}),
						matchingParty([]models.PartyMember{playerWithMMR("d", 4)}),
						matchingParty([]models.PartyMember{playerWithMMR("e", 1)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("f", 2)}),
						matchingParty([]models.PartyMember{playerWithMMR("g", 3)}),
						matchingParty([]models.PartyMember{playerWithMMR("h", 3)}),
						matchingParty([]models.PartyMember{playerWithMMR("i", 4)}),
						matchingParty([]models.PartyMember{playerWithMMR("j", 4)}),
					}},
				},
			},
			wantBetterDistance: true,
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
				distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)

				got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")

				newDistance := countDistance(got, attr, tt.args.matchingRules)
				gotBetterDistance := newDistance < distance
				if gotBetterDistance != tt.wantBetterDistance {
					b.Errorf("TestRebalanceV2_PositiveAndNegativeCase() distance = %v , newDistance %v", distance, newDistance)
				} else {
					b.Logf("got better distance: %f -> %f", distance, newDistance)
				}
			}
		})
	}
}

func BenchmarkRebalanceV2_5v5_9parties(b *testing.B) {
	logrus.SetLevel(logrus.ErrorLevel)
	scope := envelope.NewRootScope(context.Background(), "BenchmarkRebalanceV2_5v5_9parties", "")
	defer scope.Finish()

	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name               string
		args               args
		wantBetterDistance bool
	}{
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
			wantBetterDistance: true,
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
				distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)

				got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")

				newDistance := countDistance(got, attr, tt.args.matchingRules)
				gotBetterDistance := newDistance < distance
				if gotBetterDistance != tt.wantBetterDistance {
					b.Errorf("TestRebalanceV2_PositiveAndNegativeCase() distance = %v , newDistance %v", distance, newDistance)
				} else {
					b.Logf("got better distance: %f -> %f", distance, newDistance)
				}
			}
		})
	}
}

func BenchmarkRebalanceV2_6v6_12partiesBackfill(b *testing.B) {
	logrus.SetLevel(logrus.ErrorLevel)
	scope := envelope.NewRootScope(context.Background(), "BenchmarkRebalanceV2_5v5_9partiesBackfill", "")
	defer scope.Finish()

	type args struct {
		activeAllianceRule models.AllianceRule
		matchingRules      []models.MatchingRule
		allies             []models.MatchingAlly
	}
	tests := []struct {
		name               string
		args               args
		wantBetterDistance bool
	}{
		{
			name: "rebalance done",
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
						matchingParty([]models.PartyMember{playerWithMMR("a", 1)}, true),
						matchingParty([]models.PartyMember{playerWithMMR("b", 2)}),
						matchingParty([]models.PartyMember{playerWithMMR("c", 3)}),
						matchingParty([]models.PartyMember{playerWithMMR("d", 4)}),
						matchingParty([]models.PartyMember{playerWithMMR("e", 5)}),
						matchingParty([]models.PartyMember{playerWithMMR("f", 6)}),
					}},
					{MatchingParties: []models.MatchingParty{
						matchingParty([]models.PartyMember{playerWithMMR("g", 7)}, true),
						matchingParty([]models.PartyMember{playerWithMMR("h", 8)}),
						matchingParty([]models.PartyMember{playerWithMMR("i", 9)}),
						matchingParty([]models.PartyMember{playerWithMMR("j", 10)}),
						matchingParty([]models.PartyMember{playerWithMMR("k", 11)}),
						matchingParty([]models.PartyMember{playerWithMMR("l", 12)}),
					}},
				},
			},
			wantBetterDistance: true,
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
				distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)

				got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")

				newDistance := countDistance(got, attr, tt.args.matchingRules)
				gotBetterDistance := newDistance < distance
				if gotBetterDistance != tt.wantBetterDistance {
					//b.Errorf("TestRebalanceV2_PositiveAndNegativeCase() distance = %v , newDistance %v", distance, newDistance)
				} else {
					b.Logf("got better distance: %f -> %f", distance, newDistance)
				}
			}
		})
	}
}

func BenchmarkRebalanceV2_8a3p(b *testing.B) {
	scope := envelope.NewRootScope(context.Background(), "BenchmarkRebalanceV2_8a3p", "")
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
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				attr := rebalance.GetAttributeNameForRebalance(tt.args.matchingRules)
				distance := countDistance(tt.args.allies, attr, tt.args.matchingRules)

				got := RebalanceV2(scope, "", tt.args.allies, tt.args.activeAllianceRule, tt.args.matchingRules, "")

				newDistance := countDistance(got, attr, tt.args.matchingRules)
				gotBetterDistance := newDistance < distance
				if gotBetterDistance != tt.wantBetterDistance {
					b.Errorf("TestRebalanceV2_PositiveAndNegativeCase() distance = %v , newDistance %v", distance, newDistance)
				} else {
					b.Logf("got better distance: %f -> %f", distance, newDistance)
				}
			}
		})
	}
}

func BenchmarkRebalanceV2(b *testing.B) {
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
		RebalanceV2(scope, "mid1", mockAllies, mockChannel.Ruleset.AllianceRule, mockChannel.Ruleset.MatchingRule, "")
	}
}

func BenchmarkRebalanceV2RandomWithMaxPlayer(b *testing.B) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalance", "")
	defer scope.Finish()

	// run the Rebalance function b.N times
	maxAlly := 5

	// we create several combinations of party with max player = 5
	partyCombinations := [][]int{
		{5},    // 1 party with 5 member
		{4, 1}, // 2 parties with first party have 4 member, second party have 1 member
		{3, 2},
		{2, 2, 1},
		{2, 1, 1, 1},
		{1, 1, 1, 1, 1},
	}
	activeAllianceRule := mockChannel.Ruleset.AllianceRule
	for n := 0; n < b.N; n++ {
		allyCount := generateRandomInt(1, maxAlly)
		activeAllianceRule.MaxNumber = allyCount
		allies := make([]models.MatchingAlly, 0, allyCount)
		for i := 0; i < allyCount; i++ {
			allies = append(allies, models.MatchingAlly{
				MatchingParties: generateRandomPartiesWithCombinations(partyCombinations),
			})
		}
		RebalanceV2(scope, "mid1", allies, activeAllianceRule, mockChannel.Ruleset.MatchingRule, "")
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

func BenchmarkRebalanceV2RandomAllies(b *testing.B) {
	scope := envelope.NewRootScope(context.Background(), "TestRebalance", "")
	defer scope.Finish()

	// run the Rebalance function b.N times
	maxCount := 5
	for n := 0; n < b.N; n++ {
		RebalanceV2(scope, "mid1", generateRandomAllies(maxCount), mockChannel.Ruleset.AllianceRule, mockChannel.Ruleset.MatchingRule, "")
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

func player(userID string, mmr float64) models.PartyMember {
	return models.PartyMember{UserID: userID, ExtraAttributes: map[string]interface{}{"anyDistanceAttribute": mmr}}
}
