// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package reordertool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewOnePointer(t *testing.T) {
	type test struct {
		name     string
		input    []int
		wantList [][]int
	}
	tc := test{
		name:  "three objects by value: 1,3,5",
		input: []int{1, 3, 5},
		wantList: [][]int{
			{1, 3, 5},
			{3, 1, 5},
			{5, 1, 3},
		},
	}

	var gotList [][]int
	r := NewOnePointer(tc.input)
	for r.HasNext() {
		gotList = append(gotList, r.Get())
	}

	// check length of output elements
	assert.Equal(t, len(gotList), len(tc.wantList))

	// check each combination of reorder elements
	for i, actual := range gotList {
		want := tc.wantList[i]
		assert.ElementsMatch(t, actual, want)
	}
}

func TestNewOnePointerByLength(t *testing.T) {
	type test struct {
		name     string
		length   int
		wantList [][]int
	}
	tc := test{
		name:   "three objects by length",
		length: 3,
		wantList: [][]int{
			{0, 1, 2},
			{1, 0, 2},
			{2, 0, 1},
		},
	}

	var gotList [][]int
	r := NewOnePointerByLength(tc.length)
	for r.HasNext() {
		gotList = append(gotList, r.Get())
	}

	// check length of output elements
	assert.Equal(t, len(gotList), len(tc.wantList))

	// check each combination of reorder elements
	for i, actual := range gotList {
		want := tc.wantList[i]
		assert.ElementsMatch(t, actual, want)
	}
}

func TestNewOnePointerOptionsSkipEmpty(t *testing.T) {
	type test struct {
		name        string
		options     Options
		wantHasNext bool
	}
	tests := []test{
		{
			name:        "empty elements with options SkipEmpty false",
			options:     Options{SkipEmpty: false},
			wantHasNext: true,
		}, {
			name:        "empty elements with options SkipEmpty true",
			options:     Options{SkipEmpty: true},
			wantHasNext: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := NewOnePointerByLength(0)
			r.SetOptions(tc.options)

			gotHasNext := false
			for r.HasNext() {
				gotHasNext = true
			}

			if gotHasNext != tc.wantHasNext {
				t.Errorf("TestNewOnePointerOptionsSkipEmpty() = %v , want %v", gotHasNext, tc.wantHasNext)
			}
		})
	}
}

func TestNewOnePointerOptionsMaxLoop(t *testing.T) {
	type test struct {
		name            string
		length          int
		options         Options
		wantOutputCount int
	}
	tests := []test{
		{
			name:            "three objects will results 3 combination, but maxLoop limit it",
			length:          3,
			options:         Options{MaxLoop: 1},
			wantOutputCount: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := NewOnePointerByLength(tc.length)
			r.SetOptions(tc.options)

			var gotOutputCount int
			for r.HasNext() {
				gotOutputCount++
			}

			if gotOutputCount != tc.wantOutputCount {
				t.Errorf("TestNewOnePointerOptionsMaxLoop() = %v , want %v", gotOutputCount, tc.wantOutputCount)
			}
		})
	}
}

func TestNewOnePointerOptionsMaxSecond(t *testing.T) {
	type test struct {
		name            string
		length          int
		options         Options
		wantOutputCount int
	}
	tests := []test{
		{
			name:            "test with sleep in 1 second, 3 elements should results 3 second, but maxSecond limit it",
			length:          3,
			options:         Options{MaxSecond: 1},
			wantOutputCount: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := NewOnePointerByLength(tc.length)
			r.SetOptions(tc.options)

			var gotOutputCount int
			for r.HasNext() {
				gotOutputCount++
				// WE PUT SLEEP 1 SEC HERE
				time.Sleep(time.Second)
			}

			if gotOutputCount != tc.wantOutputCount {
				t.Errorf("TestNewOnePointerOptionsMaxSecond() = %v , want %v", gotOutputCount, tc.wantOutputCount)
			}
		})
	}
}

func TestNewOnePointerOptionsElementsAlwaysFirst(t *testing.T) {
	type test struct {
		name     string
		input    []int
		options  Options
		wantList [][]int
	}
	tc := test{
		name:  "three objects by value: 1,3,5, with options value for ElementsAlwaysFirst",
		input: []int{1, 3, 5},
		options: Options{
			ElementsAlwaysFirst: []int{1},
		},
		wantList: [][]int{
			{1, 3, 5},
			{1, 5, 3},
		},
	}

	var gotList [][]int
	r := NewOnePointer(tc.input)
	r.SetOptions(tc.options)
	for r.HasNext() {
		gotList = append(gotList, r.Get())
	}

	// check length of output elements
	assert.Equal(t, len(gotList), len(tc.wantList))

	// check each combination of reorder elements
	for i, actual := range gotList {
		want := tc.wantList[i]
		assert.ElementsMatch(t, actual, want)
	}
}

func TestNewOnePointerOptionsElementsAlwaysFirstWrongValue(t *testing.T) {
	type test struct {
		name     string
		input    []int
		options  Options
		wantList [][]int
	}
	tc := test{
		name:  "three objects by value: 1,3,5, with wrong options value for ElementsAlwaysFirst",
		input: []int{1, 3, 5},
		options: Options{
			ElementsAlwaysFirst: []int{0},
		},
		wantList: [][]int{
			{1, 3, 5},
			{3, 1, 5},
			{5, 1, 3},
		},
	}

	var gotList [][]int
	r := NewOnePointer(tc.input)
	r.SetOptions(tc.options)
	for r.HasNext() {
		gotList = append(gotList, r.Get())
	}

	// check length of output elements
	assert.Equal(t, len(gotList), len(tc.wantList))

	// check each combination of reorder elements
	for i, actual := range gotList {
		want := tc.wantList[i]
		assert.ElementsMatch(t, actual, want)
	}
}
