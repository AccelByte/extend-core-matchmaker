// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package reordertool

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
)

/**

TwoPointer reorder elements using 2 pointers

for example:

if we have elements > a, b, c
pointer1=0, pointer2=1 > a,b,c
pointer1=0, pointer2=2 > a,c,b
pointer1=1, pointer2=0 > b,a,c
pointer1=1, pointer2=2 > b,c,a
pointer1=2, pointer2=0 > c,a,b
pointer1=2, pointer2=1 > c,b,a

It is currently used in FindPartyCombination()
to reorder the list of tickets and try to find party again,
this is useful especially to find party for role based

**/

type TwoPointer struct {
	input    []int
	output   []int
	pointer1 int
	pointer2 int

	opt           Options
	firstElements []int

	countLoop int
	startTime time.Time

	done bool
}

// NewTwoPointer receive list of integer (input),
// it become input to the main function HasNext(),
// finally Get() will return reordered list of integer.
func NewTwoPointer(input []int) *TwoPointer {
	return &TwoPointer{
		input:     input,
		startTime: time.Now(),
	}
}

// NewTwoPointerByLength receive length of object (length),
// it will convert n into list of index (from 0 to n-1),
// then put as an input to the main function HasNext(),
// finally Get() will return list of index in reorder from 0 to n-1.
func NewTwoPointerByLength(length int) *TwoPointer {
	input := make([]int, 0, length)
	for i := 0; i < length; i++ {
		input = append(input, i)
	}
	return &TwoPointer{
		input:     input,
		startTime: time.Now(),
	}
}

func (p *TwoPointer) SetOptions(opt Options) {
	p.opt = opt

	if len(p.opt.ElementsAlwaysFirst) > 0 {
		// sanity ElementsAlwaysFirst value
		firstElements := make([]int, 0)
		for _, v := range p.opt.ElementsAlwaysFirst {
			// skip value if not contains in input
			if !utils.Contains(p.input, v) {
				continue
			}
			firstElements = append(firstElements, v)
		}

		// put into firtElements
		p.firstElements = firstElements

		// update input, remove first elements
		newInput := make([]int, 0)
		for _, v := range p.input {
			// skip from input if it included in firstElements
			if utils.Contains(p.firstElements, v) {
				continue
			}
			newInput = append(newInput, v)
		}
		p.input = newInput
	}
}

func (p *TwoPointer) Get() []int {
	return append(p.firstElements, p.output...)
}

func (p *TwoPointer) HasNext() bool {
	if p.done {
		return false
	}

	if len(p.input) == 0 {
		p.done = true
		if p.opt.SkipEmpty {
			return false
		}
		return true
	}

	if p.opt.MaxLoop > 0 && p.countLoop >= p.opt.MaxLoop {
		p.done = true
		return false
	}
	if p.opt.MaxSecond > 0 && time.Since(p.startTime) >= time.Duration(p.opt.MaxSecond)*time.Second {
		p.done = true
		return false
	}

	if p.pointer1 == 0 && p.pointer2 == 0 {
		p.pointer2 = 1
	}

	for {
		incr := func() {
			if p.pointer2 < (len(p.input) - 1) {
				p.pointer2++
			} else {
				p.pointer1++
				p.pointer2 = 0
			}
		}
		if p.pointer2 == p.pointer1 {
			incr()
			continue
		}
		p.output = make([]int, 0)
		if p.pointer1 < len(p.input) {
			p.output = append(p.output, p.input[p.pointer1])
			if p.pointer2 < len(p.input) {
				p.output = append(p.output, p.input[p.pointer2])
				skipIndex := []int{p.pointer1, p.pointer2}
				for i, v := range p.input {
					if utils.Contains(skipIndex, i) {
						continue
					}
					p.output = append(p.output, v)
				}
			}
		}
		incr()
		if p.pointer1 >= (len(p.input)-1) && p.pointer2 >= (len(p.input)-1) {
			p.done = true
		}
		break
	}

	if len(p.output) == 0 && p.opt.SkipEmpty {
		p.done = true
		return false
	}

	p.countLoop++
	return true
}
