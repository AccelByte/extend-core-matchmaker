// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v2

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/group_generator"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"

	"gonum.org/v1/gonum/stat/combin"
)

/*
GenerateCombination generate combinations of parties based on activeAllianceRule.
Return combinations of parties.

Step 1: Generate combinations of players from the parties.
It generates nCr combinations where n = number of players from parties and r = player number from the ruleset while respecting players in the same party.
See the comment in the generateCombinationsOfPlayers() function for a more detailed example.

Step 2: Generate combinations of parties from the combinations of players.
It basically converts the index of players into the index of parties.
See the comment in the generateCombinationsOfParties() function for a more detailed example.

Step 3: Generate sequences of parties from the combinations of parties.
It merges some index of parties into a sequence of parties while ensuring no intersection between them.
See the comment in the generatePartiesSequence() function for a more detailed example.
*/
func GenerateCombination(parties []models.MatchingParty, activeAllianceRule models.AllianceRule) [][]int {
	// step 1: generate combinations of players
	combinationsOfPlayers := generateCombinationsOfPlayers(parties, activeAllianceRule)

	// step 2: generate combinations of parties from the combinations of players
	combinationsOfParties := generateCombinationsOfParties(parties, combinationsOfPlayers)

	// step 3: generate sequences of parties from the combinations of parties
	partiesSequence := generatePartiesSequence(parties, combinationsOfParties)

	return partiesSequence
}

func CreateCombinationGenerator(parties []models.MatchingParty, maxNumber int) *group_generator.CombinationGenerator[*models.PartyMember, *models.MatchingParty] {
	// user pointers for value and id to save up memory, only use 64-bit memory per value
	group := make([]*group_generator.Slice[*models.PartyMember, *models.MatchingParty], len(parties))

	for i := 0; i < len(parties); i++ {
		party := parties[i]
		members := make([]*models.PartyMember, len(party.PartyMembers))
		for j := 0; j < len(party.PartyMembers); j++ {
			member := party.PartyMembers[j]
			members[j] = &member
		}
		group[i] = group_generator.NewSliceWithID[*models.PartyMember, *models.MatchingParty](members, &party)
	}

	return group_generator.NewCombinationGenerator[*models.PartyMember, *models.MatchingParty](group, maxNumber)
}

type playerIndex int

/*
The generateCombinationsOfPlayers generate combinations of players with all possible number of players from max player to min player.
It respect players in the same party, which means a combination that separates players in parties will be skipped from the result.
Return combinations of player index.

For example: we have 5 parties with 6 players, and we want to find combination of 3 players, with userIDs as below:

[a b], [c], [d], [e], [f]

[0 1], [2], [3], [4], [5] --> player index

Result:

[0]: [0 1 2],
[1]: [0 1 3],
[2]: [0 1 4],
[3]: [0 1 5],
[4]: [2 3 4],
[5]: [2 3 5],
[6]: [2 4 5],
[7]: [3 4 5]

Result in userID:

[0]: [a b c],
[1]: [a b d],
[2]: [a b e],
[3]: [a b f],
[4]: [c d e],
[5]: [c d f],
[6]: [c e f],
[7]: [d e f]
*/
func generateCombinationsOfPlayers(parties []models.MatchingParty, activeAllianceRule models.AllianceRule) (combinationsOfPlayers [][]playerIndex) {
	numPlayer := countPlayer(parties)

	playerInParty, partyHasPlayers := getMapPartyPlayer(parties)

	combinationsOfPlayers = make([][]playerIndex, 0)
	for r := activeAllianceRule.PlayerMaxNumber; r >= activeAllianceRule.PlayerMinNumber; r-- {
		if numPlayer < r {
			continue
		}
	cLoop:
		for _, indexes := range combin.Combinations(numPlayer, r) {
			// check whether this combination separate player in one party
			parties := make(map[int]struct{})
			players := make(map[int]struct{}, len(indexes))
			playerIndexes := make([]playerIndex, 0, len(indexes))
			for _, iPlayer := range indexes {
				iParty := playerInParty[iPlayer]
				parties[iParty] = struct{}{}
				players[iPlayer] = struct{}{}
				playerIndexes = append(playerIndexes, playerIndex(iPlayer))
			}
			for iParty := range parties {
				for _, iPlayer := range partyHasPlayers[iParty] {
					if _, exist := players[iPlayer]; !exist {
						// skip this combination because it separate players in a party, check next combination
						continue cLoop
					}
				}
			}
			combinationsOfPlayers = append(combinationsOfPlayers, playerIndexes)
		}
	}
	return combinationsOfPlayers
}

/*
The generateCombinationsOfParties generate combinations of parties index from combinations of players.
Return combination of party index.

For example: we have 5 parties with 6 players, and combinations of players as below:

parties:

[a b], [c], [d], [e], [f]

[0 1], [2], [3], [4], [5] --> player index

[ 0 ], [1], [2], [3], [4] --> party index

combination of players:

[0]: [0 1 2],
[1]: [0 1 3],
[2]: [0 1 4],
[3]: [0 1 5],
[4]: [2 3 4],
[5]: [2 3 5],
[6]: [2 4 5],
[7]: [3 4 5]

Result:

[0]: [0 1],
[1]: [0 2],
[2]: [0 3],
[3]: [0 4],
[4]: [1 2 3],
[5]: [1 2 4],
[6]: [1 3 4],
[7]: [2 3 4]
*/
func generateCombinationsOfParties(parties []models.MatchingParty, combinationsOfPlayers [][]playerIndex) (combinationsOfParties [][]int) {
	playerInParty, _ := getMapPartyPlayer(parties)

	combinationsOfParties = make([][]int, 0)
	for _, indexes := range combinationsOfPlayers {
		parties := make([]int, 0)
		partiesCheck := make(map[int]struct{})
		for _, iPlayer := range indexes {
			iParty := playerInParty[int(iPlayer)]
			if _, exist := partiesCheck[iParty]; exist {
				continue
			}
			parties = append(parties, iParty)
			partiesCheck[iParty] = struct{}{}
		}

		combinationsOfParties = append(combinationsOfParties, parties)
	}
	return combinationsOfParties
}

/*
The generatePartiesSequence generate sequences of parties by selecting from combinations of parties,
and make sure the next team doesn't contain combination from previous team.
Return combination of party index.

For example: we have 5 parties with 6 players, and combinations of parties as below:

parties:

[a b], [c], [d], [e], [f]

[0 1], [2], [3], [4], [5] --> player index

[ 0 ], [1], [2], [3], [4] --> party index

combination of parties:

[0]: [0 1],
[1]: [0 2],
[2]: [0 3],
[3]: [0 4],
[4]: [1 2 3],
[5]: [1 2 4],
[6]: [1 3 4],
[7]: [2 3 4]

Result:

[0]: [[0 1] [2 3 4]]
[1]: [[0 2] [1 3 4]]
[2]: [[0 3] [1 2 4]]
[3]: [[0 4] [1 2 3]]
[4]: [[1 2 3] [0 4]]
[5]: [[1 2 4] [0 3]]
[6]: [[1 3 4] [0 2]]
[7]: [[2 3 4] [0 1]]
*/
func generatePartiesSequence(parties []models.MatchingParty, combinationsOfParties [][]int) [][]int {
	numParties := len(parties)

	memo := make(map[int][][][]int) // result cache used to speed up computation
	partySequences := generatePartiesSequenceRec(numParties, combinationsOfParties, memo)

	// convert from [[[1], [2,3]],[[2,3],[4]],...] to [[1,2,3],[2,3,4],...]
	result := make([][]int, 0, len(partySequences))
	for _, partySequence := range partySequences {
		seq := make([]int, 0, numParties)
		for _, party := range partySequence {
			seq = append(seq, party...)
		}

		result = append(result, seq)
	}
	return result
}

func generatePartiesSequenceRec(maxLen int, combinationsOfParties [][]int, resultMemo map[int][][][]int) [][][]int {
	if maxLen == 0 {
		return [][][]int{{}}
	}

	// The combinations for a given maxLen will always be the same. Example:
	// for maxLen=1 we returns a list of 1-party teams: [[a] [b] [c] ... ]
	// for maxLen=2 we returns a list of 2-party teams: [[ab] [ac] [ad] [bc] [bd] ... ]
	// Thus we can store the result in memoResult for later invocations.
	if memoResult, ok := resultMemo[maxLen]; ok {
		return memoResult
	}

	// If we don't have the result stored in memoResult, compute the actual result, then store it in memoResult.
	var results [][][]int
	for _, c1 := range combinationsOfParties {
		remaining := maxLen - len(c1)
		if remaining >= 0 {
			recursiveResult := generatePartiesSequenceRec(remaining, combinationsOfParties, resultMemo)
			for _, listOfPartyCombinations := range recursiveResult {
				fullSeq := make([][]int, 0, len(listOfPartyCombinations))
				fullSeq = append(fullSeq, c1)
				fullSeq = append(fullSeq, listOfPartyCombinations...)
				if AllUnique(fullSeq) {
					results = append(results, fullSeq)
				}
			}
		}
	}

	resultMemo[maxLen] = results
	return resultMemo[maxLen]
}

func countPlayer(parties []models.MatchingParty) int {
	var numPlayer int
	for _, party := range parties {
		numPlayer += party.CountPlayer()
	}
	return numPlayer
}

func getMapPartyPlayer(parties []models.MatchingParty) (playerInParty map[int]int, partyHasPlayers map[int][]int) {
	numPlayer := countPlayer(parties)

	playerInParty = make(map[int]int, numPlayer)
	partyHasPlayers = make(map[int][]int, len(parties))

	iPlayer := 0
	for iParty, party := range parties {
		players := make([]int, 0, len(party.PartyMembers))
		for range party.PartyMembers {
			playerInParty[iPlayer] = iParty
			players = append(players, iPlayer)
			iPlayer++
		}
		partyHasPlayers[iParty] = players
	}

	return playerInParty, partyHasPlayers
}
