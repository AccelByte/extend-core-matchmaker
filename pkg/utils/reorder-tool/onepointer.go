// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package reordertool

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/utils"
)

/**

OnePointer reorder elements using 1 pointer

for example:

if we have elements > a, b, c
pointer=0 > a,b,c
pointer=1 > b,c,a
pointer=2 > c,a,b

It is currently used in findMatchingAlly()
to reorder the list of tickets and try to find ally again,
this is useful especially for party with diverse number of player

**/

type OnePointer struct {
	input   []int
	output  []int
	pointer int

	opt           Options
	firstElements []int

	countLoop int
	startTime time.Time

	done bool
}

func NewOnePointer(input []int) *OnePointer {
	return &OnePointer{
		input:     input,
		startTime: time.Now(),
	}
}

func NewOnePointerByLength(length int) *OnePointer {
	input := make([]int, 0, length)
	for i := 0; i < length; i++ {
		input = append(input, i)
	}
	return &OnePointer{
		input:     input,
		startTime: time.Now(),
	}
}

func (p *OnePointer) SetOptions(opt Options) {
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

func (p *OnePointer) Get() []int {
	return append(p.firstElements, p.output...)
}

func (p *OnePointer) HasNext() bool {
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

	p.output = make([]int, 0)
	if p.pointer < len(p.input) {
		p.output = append(p.output, p.input[p.pointer])
		if p.pointer < (len(p.input) - 1) {
			p.output = append(p.output, p.input[p.pointer+1:]...)
		}
		if p.pointer > 0 {
			p.output = append(p.output, p.input[:p.pointer]...)
		}
	}

	p.pointer++
	if p.pointer >= (len(p.input)) {
		p.done = true
	}

	if len(p.output) == 0 && p.opt.SkipEmpty {
		p.done = true
		return false
	}

	p.countLoop++
	return true
}
