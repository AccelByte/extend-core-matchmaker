// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v2

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
)

//nolint:gochecknoglobals
var PermutationCount = map[int]int{
	1:  1,
	2:  2,
	3:  6,
	4:  24,
	5:  120,
	6:  720,
	7:  5_040,
	8:  40_320,
	9:  362_880,
	10: 3_628_800,
	11: 39_916_800,
	12: 479_001_600,
}

func NextPerm(p []int) {
	for i := len(p) - 1; i >= 0; i-- {
		if i == 0 || p[i] < len(p)-i-1 {
			p[i]++
			return
		}
		p[i] = 0
	}
}

func GetPerm(orig, p []int) []int {
	result := append([]int{}, orig...)
	for i, v := range p {
		result[i], result[i+v] = result[i+v], result[i]
	}
	return result
}

func Convert(m map[int][]models.MatchingParty) []models.MatchingAlly {
	allies := make([]models.MatchingAlly, len(m))
	for k, v := range m {
		allies[k].MatchingParties = v
	}
	return allies
}

func Reorder(parties []models.MatchingParty, newIndexes []int, reorderedParties *[]models.MatchingParty) {
	*reorderedParties = (*reorderedParties)[:0]
	for _, idx := range newIndexes {
		if idx < 0 || idx > (len(parties)-1) {
			continue
		}
		*reorderedParties = append(*reorderedParties, parties[idx])
	}
}

func CountMemberDiff(allies []models.MatchingAlly) int {
	var _min, _max int
	for _, ally := range allies {
		count := ally.CountPlayer()
		if _min == 0 || count < _min {
			_min = count
		}
		if _max == 0 || count > _max {
			_max = count
		}
	}
	return _max - _min
}

func permutation(input [][]string, result [][]string, s []string) [][]string {
	for _, a := range input[0] {
		src := s
		if len(input) > 1 {
			src = append(src, a)
			dst := make([]string, len(src))
			copy(dst, src)
			result = permutation(input[1:], result, dst)
			continue
		}
		src = append(src, a)
		dst := make([]string, len(src))
		copy(dst, src)
		result = append(result, dst)
	}
	return result
}
