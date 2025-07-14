// Copyright (c) 2023 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package group_generator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	groupType  [][]int
	groupsType []groupType
)

func convertToGroups(slice [][]*Slice[int, string]) groupsType {
	result := make([]groupType, len(slice))
	for i, groups := range slice {
		result[i] = make(groupType, len(groups))
		for j, g := range groups {
			result[i][j] = g.Value()
		}
	}
	return result
}

func sliceToSimpleString(slice [][]*Slice[int, string]) string {
	result := "{"
	for i, groups := range slice {
		result += "{"
		for j, g := range groups {
			result += "{"
			for k, v := range g.Value() {
				result += fmt.Sprintf("%v", v)
				if k < len(g.Value())-1 {
					result += ", "
				}
			}
			result += "}"
			if j < len(groups)-1 {
				result += ", "
			}
		}
		result += "}"
		if i < len(slice)-1 {
			result += ", "
		}
	}
	result += "}"
	return result
}

func TestCombinationGenerator_Next(t *testing.T) {
	type test struct {
		name string
		arr  groupType // group
		r    int
		want []groupsType // combination of groups
	}

	tests := []test{
		{
			name: "3v3",
			arr:  groupType{{1, 2}, {3}, {4}, {5, 6}},
			r:    3,
			want: []groupsType{
				{{{1, 2}, {3}}, {{4}, {5, 6}}},
				{{{1, 2}, {4}}, {{3}, {5, 6}}},
			},
		},
		{
			name: "3v3_2",
			arr:  groupType{{1}, {2, 3}, {4}, {5}, {6}},
			r:    3,
			want: []groupsType{
				{{{1}, {2, 3}}, {{4}, {5}, {6}}},
				{{{1}, {4}, {5}}, {{2, 3}, {6}}},
				{{{1}, {4}, {6}}, {{2, 3}, {5}}},
				{{{1}, {5}, {6}}, {{2, 3}, {4}}},
			},
		},
		{
			name: "2v2v2",
			arr:  groupType{{1}, {2, 3}, {4}, {5}, {6}},
			r:    2,
			want: []groupsType{
				{{{1}, {4}}, {{2, 3}}, {{5}, {6}}},
				{{{1}, {5}}, {{2, 3}}, {{4}, {6}}},
				{{{1}, {6}}, {{2, 3}}, {{4}, {5}}},
			},
		},
		{
			name: "4v4",
			arr:  groupType{{1, 2}, {3}, {4, 5, 6}, {7}, {8}},
			r:    4,
			want: []groupsType{
				{{{1, 2}, {3}, {7}}, {{4, 5, 6}, {8}}},
				{{{1, 2}, {3}, {8}}, {{4, 5, 6}, {7}}},
				{{{1, 2}, {7}, {8}}, {{3}, {4, 5, 6}}},
			},
		},
		{
			name: "4v4_2",
			arr:  groupType{{1, 2}, {3}, {4, 5}, {6}, {7, 8}},
			r:    4,
			want: []groupsType{
				{{{1, 2}, {3}, {6}}, {{4, 5}, {7, 8}}},
				{{{1, 2}, {4, 5}}, {{3}, {6}, {7, 8}}},
				{{{1, 2}, {7, 8}}, {{3}, {4, 5}, {6}}},
			},
		},
		{
			name: "3v3v3",
			arr:  groupType{{1, 2}, {3}, {4}, {5, 6}, {7}, {8, 9}},
			r:    3,
			want: []groupsType{
				{{{1, 2}, {3}}, {{4}, {5, 6}}, {{7}, {8, 9}}},
				{{{1, 2}, {3}}, {{4}, {8, 9}}, {{5, 6}, {7}}},
				{{{1, 2}, {4}}, {{3}, {5, 6}}, {{7}, {8, 9}}},
				{{{1, 2}, {4}}, {{3}, {8, 9}}, {{5, 6}, {7}}},
				{{{1, 2}, {7}}, {{3}, {5, 6}}, {{4}, {8, 9}}},
				{{{1, 2}, {7}}, {{3}, {8, 9}}, {{4}, {5, 6}}},
			},
		},
		{
			name: "3v3v3_2",
			arr:  groupType{{1, 2}, {3, 4, 5}, {6}, {7}, {8, 9}},
			r:    3,
			want: []groupsType{
				{{{1, 2}, {6}}, {{3, 4, 5}}, {{7}, {8, 9}}},
				{{{1, 2}, {7}}, {{3, 4, 5}}, {{6}, {8, 9}}},
			},
		},
		{
			name: "3v3v3_invalid",
			arr:  groupType{{1, 2}, {3, 4}, {5, 6}, {7}, {8, 9}},
			r:    3,
			want: []groupsType{},
		},
		{
			name: "5v5",
			arr:  groupType{{1, 2}, {3, 4, 5, 6}, {7}, {8, 9, 10}},
			r:    5,
			want: []groupsType{
				{{{1, 2}, {8, 9, 10}}, {{3, 4, 5, 6}, {7}}},
			},
		},
		{
			name: "5v5_2",
			arr:  groupType{{1, 2}, {3, 4, 5, 6}, {7}, {8}, {9, 10}},
			r:    5,
			want: []groupsType{
				{{{1, 2}, {7}, {9, 10}}, {{3, 4, 5, 6}, {8}}},
				{{{1, 2}, {8}, {9, 10}}, {{3, 4, 5, 6}, {7}}},
			},
		},
		{
			name: "5v5_3",
			arr:  groupType{{1, 2}, {3, 4, 5}, {6}, {7}, {8}, {9, 10}},
			r:    5,
			want: []groupsType{
				{{{1, 2}, {3, 4, 5}}, {{6}, {7}, {8}, {9, 10}}},
				{{{1, 2}, {6}, {7}, {8}}, {{3, 4, 5}, {9, 10}}},
				{{{1, 2}, {6}, {9, 10}}, {{3, 4, 5}, {7}, {8}}},
				{{{1, 2}, {7}, {9, 10}}, {{3, 4, 5}, {6}, {8}}},
				{{{1, 2}, {8}, {9, 10}}, {{3, 4, 5}, {6}, {7}}},
			},
		},
	}

	for _, ts := range tests {
		t.Run(ts.name, func(t *testing.T) {
			slice := NewSliceOfIntSlice(ts.arr)
			gen := NewCombinationGenerator[int, string](slice, ts.r)
			i := 0
			for {
				comb := gen.Next()
				if comb == nil {
					break
				}

				fmt.Printf("%v,\n", sliceToSimpleString(comb))

				require.Less(t, i, len(ts.want))
				assert.Equalf(t, ts.want[i], convertToGroups(comb), "at index %v", i)

				i++
			}

			assert.Equal(t, len(ts.want), i)
		})
	}
}
