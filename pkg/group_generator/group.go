// Copyright (c) 2023-2024 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package group_generator

import (
	"fmt"
)

// Slice holds slice of T with id type U
// store id for faster identification.
type Slice[T any, U comparable] struct {
	value []T
	id    U
}

func NewSlice[T any, U string](value []T) *Slice[T, U] {
	str := fmt.Sprintf("%v", value)
	return &Slice[T, U]{value: value, id: U(str)}
}

func NewSliceWithID[T any, U comparable](value []T, id U) *Slice[T, U] {
	return &Slice[T, U]{value: value, id: id}
}

func NewIntSlice(value []int) *Slice[int, string] {
	str := fmt.Sprintf("%v", value)
	return &Slice[int, string]{value: value, id: str}
}

func NewIntSliceWithID(value []int, id string) *Slice[int, string] {
	return &Slice[int, string]{value: value, id: id}
}

func (s Slice[T, U]) Value() []T {
	return s.value
}

func (s Slice[T, U]) ID() U {
	return s.id
}

func (s Slice[T, U]) String() string {
	return fmt.Sprintf("%v", s.id)
}

func NewSliceOfIntSlice(value [][]int) []*Slice[int, string] {
	slice := make([]*Slice[int, string], len(value))
	for i, ints := range value {
		slice[i] = NewIntSlice(ints)
	}
	return slice
}

func RemoveValues[T any, U comparable](slice []*Slice[T, U], valuesToRemove []*Slice[T, U]) []*Slice[T, U] {
	valuesMap := make(map[U]struct{})
	for _, value := range valuesToRemove {
		valuesMap[value.ID()] = struct{}{}
	}

	result := make([]*Slice[T, U], 0)

	for _, item := range slice {
		if _, ok := valuesMap[item.ID()]; !ok {
			result = append(result, item)
		}
	}

	return result
}

// Validator validates group.
type Validator[T any, U comparable] interface {
	// ValidateStart initiates validation
	ValidateStart()
	// Validate newValue to be added to current group
	Validate(currentGroup []*Slice[T, U], newValue *Slice[T, U]) bool
	// IsComplete is to stop a newValue added to current group
	IsComplete() bool
}

type Iterator[T any, U comparable] interface {
	Next() []*Slice[T, U]
	Values() []*Slice[T, U]
}

type DefaultValidator[T any, U comparable] struct {
	currentMembers int
	maxMembers     int
}

func NewDefaultValidator[T any, U comparable](maxMembers int) *DefaultValidator[T, U] {
	return &DefaultValidator[T, U]{currentMembers: 0, maxMembers: maxMembers}
}

func (d *DefaultValidator[T, U]) ValidateStart() {
	d.currentMembers = 0
}

func (d *DefaultValidator[T, U]) Validate(currentGroup []*Slice[T, U], newValue *Slice[T, U]) bool {
	if d.currentMembers+len(newValue.Value()) <= d.maxMembers {
		d.currentMembers += len(newValue.value)
		return true
	}
	return false
}

func (d *DefaultValidator[T, U]) IsComplete() bool {
	return d.currentMembers >= d.maxMembers
}

// CombinationIterator is to generate combinations one at a time.
// The difference with normal combination is
// the first value won't change to prevent same combination value on other group.
type CombinationIterator[T any, U comparable] struct {
	arr     []*Slice[T, U]
	r       int
	indices []int
	done    bool

	validator Validator[T, U]
}

func NewCombinationIteratorDefault[T any, U comparable](slice []*Slice[T, U], r int) *CombinationIterator[T, U] {
	indices := make([]int, r)
	for i := 0; i < len(slice); i++ {
		indices[i] = i
	}
	return &CombinationIterator[T, U]{
		arr:       slice,
		r:         r,
		indices:   indices,
		validator: NewDefaultValidator[T, U](r),
	}
}

func NewCombinationIterator[T any, U comparable](slice []*Slice[T, U], r int, validator Validator[T, U]) *CombinationIterator[T, U] {
	indices := make([]int, len(slice))
	for i := 0; i < len(slice); i++ {
		indices[i] = i
	}
	return &CombinationIterator[T, U]{
		arr:       slice,
		r:         r,
		indices:   indices,
		validator: validator,
	}
}

func (c *CombinationIterator[T, U]) Values() []*Slice[T, U] {
	return c.arr
}

// Next returns the next group combination. It returns nil when there are no more combinations.
// First member won't change to avoid duplication on other groups.
func (c *CombinationIterator[T, U]) Next() []*Slice[T, U] {
retry:
	if c.done {
		return nil
	}

	combination := make([]*Slice[T, U], 0, c.r)
	newIndices := make([]int, 0, c.r)
	needUpdateIndex := false

	c.validator.ValidateStart()
	complete := false
	for i := 0; i < len(c.indices); i++ {
		if c.indices[i] >= len(c.arr) {
			break
		}

		v := c.arr[c.indices[i]]
		// check whether fit into group
		if c.validator.Validate(combination, v) {
			combination = append(combination, v)
			newIndices = append(newIndices, c.indices[i])

			if complete = c.validator.IsComplete(); complete {
				break
			}
		} else {
			// indices changed because current index is not fit into group
			needUpdateIndex = true
		}
	}

	if needUpdateIndex {
		l := len(newIndices)
		c.indices = newIndices
		// fill the remaining values with increment values
		for j := l; j < c.r; j++ {
			c.indices = append(c.indices, c.indices[j-1]+1)
		}
	}

	i := len(combination) - 1
	// find index that can be increased, check descending value from tail that has not maxed
	// [0,1,2,3,4] with r=3 -> [0,3,4] -> max is 4 -> [3,4] is descending values that already maxed -> 0 can be increased
	for ; i >= 0 && c.indices[i] == len(c.arr)-len(combination)+i; i-- {
	}

	// don't increment fist value to avoid same group arrangement
	// because the first value will move to another group and produce same combination with this group
	if i <= 0 {
		c.done = true
		if !complete {
			return nil
		}
	} else {
		// increase index value, value after it should be increment values
		// [0,3,4] -> 0 increased to 1 -> increment value from 1 -> [1,2,3]
		c.indices[i]++
		for j := i + 1; j < c.r; j++ {
			c.indices[j] = c.indices[j-1] + 1
		}
		if !complete {
			goto retry
		}
	}

	return combination
}

type Group[T any, U comparable] struct {
	iterator    Iterator[T, U]
	combination []*Slice[T, U]
}

type CombinationGenerator[T any, U comparable] struct {
	groups     []*Group[T, U]
	numGroup   int
	groupIndex int
	r          int

	validator Validator[T, U]
}

func NewCombinationGenerator[T any, U comparable](slice []*Slice[T, U], r int) *CombinationGenerator[T, U] {
	return NewCombinationGeneratorWithValidator(slice, r, Validator[T, U](NewDefaultValidator[T, U](r)))
}

func NewCombinationGeneratorWithValidator[T any, U comparable](slice []*Slice[T, U], r int, validator Validator[T, U]) *CombinationGenerator[T, U] {
	numElements := 0
	for _, s := range slice {
		numElements += len(s.Value())
	}

	numGroup := numElements / r

	groups := make([]*Group[T, U], numGroup)
	for i := 0; i < numGroup; i++ {
		groups[i] = &Group[T, U]{
			iterator:    NewCombinationIterator(slice, r, validator),
			combination: nil,
		}
	}

	return &CombinationGenerator[T, U]{
		groups:     groups,
		numGroup:   numGroup,
		groupIndex: 0,
		r:          r,
		validator:  validator,
	}
}

// Next generate combination for groups.
// Will generate combination for one group first then the next group will be generated with remaining values.
// Returns combined combination for all groups.
func (g *CombinationGenerator[T, U]) Next() [][]*Slice[T, U] {
	if g.groupIndex < 0 {
		return nil
	}

	for {
		group := g.groups[g.groupIndex]
		group.combination = group.iterator.Next()
		// no more combination of current group, move to previous group
		if group.combination == nil {
			g.groupIndex--
			if g.groupIndex < 0 {
				return nil
			}
			continue
		}

		// don't need to find combination for last group, the last group combination should be remaining value
		if g.groupIndex == g.numGroup-2 {
			remaining := RemoveValues(group.iterator.Values(), group.combination)

			groups := make([][]*Slice[T, U], g.numGroup)
			for i := 0; i < g.numGroup-1; i++ {
				groups[i] = g.groups[i].combination
			}
			groups[g.numGroup-1] = remaining

			return groups
		} else if g.groupIndex < g.numGroup-1 {
			// move to next group
			g.groupIndex++
			remaining := RemoveValues(group.iterator.Values(), group.combination)
			// create iterator with remaining values for the next group
			g.groups[g.groupIndex].iterator = NewCombinationIterator(remaining, g.r, g.validator)
		} else if g.numGroup == 1 {
			return [][]*Slice[T, U]{g.groups[0].combination}
		}
	}
}
