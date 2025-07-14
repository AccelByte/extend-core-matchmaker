// Copyright (c) 2024 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package partition

import (
	"context"
	"fmt"
	"math"
	"sort"
)

// limit based on function iteration
const iterationLimit = 1_000_000

type Data struct {
	Partitions     []*Partition
	BestDistance   float64
	BestCountDiff  int
	BestPartitions [][]Item
	MaxCount       int
	IsOrderMatter  bool
	MaxIteration   int

	IsTimeout bool

	ctx           context.Context
	funcIteration int
	itemIteration int
	validateFunc  func([]*Partition) bool
}

type CgaOptions struct {
	Ctx          context.Context
	UseRecursive bool
	// IsOrderMatter mark whether partition order is matter. If true will behave like permutation, if false will behave like combination.
	// Only affect partitions order not the members.
	IsOrderMatter bool
	// MaxCount maximum count for each partition.
	MaxCount int
	// ValidateFunc validates generated partition.
	ValidateFunc func([]*Partition) bool
	// Limit how much item variant is generated
	MaxIteration int
}

// IsCanceled check if partitioning process is canceled
func (p *Data) IsCanceled() bool {
	if p.MaxIteration > 0 && p.itemIteration >= p.MaxIteration {
		return true
	}

	if p.ctx == nil {
		return false
	}
	select {
	case <-p.ctx.Done():
		p.IsTimeout = true
		return true
	default:
		// check the max iteration
		return false
	}
}

// IsValid validates current partitions
func (p *Data) IsValid() bool {
	if p.validateFunc == nil {
		return true
	}

	return p.validateFunc(p.Partitions)
}

// NumIteration return number of item iterations that currently generated
func (p *Data) NumIteration() int {
	return p.itemIteration
}

type Item interface {
	Value() float64
	Count() int
	// ID to identify the input and output items for external use.
	ID() int
}

type ItemSequence struct {
	Item
	// To prevent same partition with different order. [a][b] should be same as [b][a].
	sequence int
}

func (v ItemSequence) Sequence() int {
	return v.sequence
}

func (v ItemSequence) String() string {
	return fmt.Sprintf("%v", v.Item)
}

type Partition struct {
	index  int
	sum    float64
	count  int
	values []ItemSequence
}

func (p *Partition) Push(v ItemSequence) {
	p.sum += v.Value()
	p.count += v.Count()
	p.values = append(p.values, v)
}

func (p *Partition) PushItem(v Item) {
	p.Push(ItemSequence{
		Item: v,
	})
}

func (p *Partition) Pop() float64 {
	v := p.values[len(p.values)-1]
	p.values = p.values[0 : len(p.values)-1]
	p.sum -= v.Value()
	p.count -= v.Count()
	return v.Value()
}

func (p *Partition) Sum() float64 {
	return p.sum
}

func (p *Partition) Avg() float64 {
	return p.sum / float64(p.count)
}

func (p *Partition) FirstSequence() int {
	if len(p.values) == 0 {
		return -1
	}
	return p.values[0].Sequence()
}

func (p *Partition) IntValues() []int {
	values := make([]int, 0, len(p.values))
	for _, value := range p.values {
		values = append(values, int(value.Value()))
	}
	return values
}

func (p *Partition) StringValues() string {
	var values string
	for i, value := range p.values {
		values += fmt.Sprintf("%v", value)
		if i < len(p.values)-1 {
			values += ","
		}
	}
	return values
}

func (p *Partition) Values() []ItemSequence {
	return p.values
}

func distance(partition []*Partition) float64 {
	if len(partition) == 0 {
		return 0
	}

	minDist := math.MaxFloat64
	maxDist := float64(-1)

	for _, p := range partition {
		if minDist > p.Avg() {
			minDist = p.Avg()
		}
		if maxDist < p.Avg() {
			maxDist = p.Avg()
		}
	}

	return maxDist - minDist
}

func sumDistance(partition []*Partition) int {
	if len(partition) == 0 {
		return 0
	}

	minDist := math.MaxFloat64
	maxDist := float64(-1)

	for _, p := range partition {
		if minDist > p.Sum() {
			minDist = p.Sum()
		}
		if maxDist < p.Sum() {
			maxDist = p.Sum()
		}
	}

	return int(maxDist - minDist)
}

func countDiff(partition []*Partition) int {
	if len(partition) == 0 {
		return 0
	}

	minCount := math.MaxInt32
	maxCount := math.MinInt32

	for _, p := range partition {
		if minCount > p.count {
			minCount = p.count
		}
		if maxCount < p.count {
			maxCount = p.count
		}
	}

	return maxCount - minCount
}

func printPartition(partitions []*Partition, depth, iteration int) {
	vals := ""
	for i, partition := range partitions {
		vals += "(" + partition.StringValues()
		vals += ")"
		if i < len(partitions)-1 {
			vals += " "
		}
	}
	fmt.Printf("%2v|%2v| %v -> %2v | %4v | %.3v \n", iteration, depth, vals, countDiff(partitions), sumDistance(partitions), distance(partitions))
}

// cgaRecursive Complete Greedy Algorithm partition using recursive function.
// Please use the iterative function for performance.
// This function is slightly worse than the iterative function but more readable:
// BenchmarkPartition3v3v3_Recursive-16    	    9598	    111168 ns/op	   57480 B/op	    2131 allocs/op
// BenchmarkPartition3v3v3_Iterative-16    	    9853	    107154 ns/op	   42368 B/op	    1490 allocs/op
func cgaRecursive(data *Data, items *[]ItemSequence, partitions []*Partition, depth int) {
	if data.IsCanceled() {
		return
	}

	data.funcIteration++
	if data.funcIteration > iterationLimit {
		return
	}

	// all items are filled to partition
	if depth == len(*items) {
		data.itemIteration++
		if data.IsValid() {
			count := countDiff(partitions)
			// check the distance if lower than current minimum
			dist := distance(data.Partitions)
			if count < data.BestCountDiff || (count == data.BestCountDiff && dist < data.BestDistance) {
				data.BestCountDiff = count
				if data.BestPartitions == nil {
					data.BestPartitions = make([][]Item, len(data.Partitions))
				}
				setBest(data, dist, data.Partitions)
			}
		}
		// printPartition(data.Partitions, depth, data.itemIteration)
		return
	}

	// sort partitions from lowest to highest score to fill the item to the best heuristic value first
	sort.Slice(partitions, func(i, j int) bool {
		return partitions[i].Sum() < partitions[j].Sum()
	})

	// store partition indexes for current item to try
	partitionIndexes := make([]int, len(data.Partitions))
	for i, p := range partitions {
		partitionIndexes[i] = p.index
	}

	item := (*items)[depth]

	// try to fill the item to all possible partitions
	for _, i := range partitionIndexes {
		data.Partitions[i].Push(item)

		differentOrder := false
		if !data.IsOrderMatter {
			prevSequence := -1
			// to avoid combination generated with same partitions value regardless of the partition order, [ab],[cd] should be same as [cd],[ab]
			// the first element of partition should be in descending/ascending order with other partition
			// we only use ascending sequence here to avoid repetition
			// IsOrderMatter=true will behave like permutation
			for _, p := range data.Partitions {
				sequence := p.FirstSequence()
				if sequence == -1 {
					continue
				}
				if prevSequence > sequence {
					differentOrder = true
					break
				}
				prevSequence = sequence
			}
		}
		if differentOrder || data.Partitions[i].count > data.MaxCount {
			data.Partitions[i].Pop()
			continue
		}

		cgaRecursive(data, items, partitions, depth+1)
		data.Partitions[i].Pop()
	}
}

func setBest(data *Data, minDistance float64, partition []*Partition) {
	data.BestDistance = minDistance
	for i, p := range partition {
		if data.BestPartitions[i] == nil {
			data.BestPartitions[i] = make([]Item, 0, len(p.values))
		} else {
			// clear items
			data.BestPartitions[i] = data.BestPartitions[i][:0]
		}
		for _, v := range p.values {
			data.BestPartitions[i] = append(data.BestPartitions[i], v)
		}
	}
}

type cgaIterStack struct {
	i                int
	initialized      bool
	continueIter     bool
	partitionIndexes []int
}

func newCgaStack(numPartition int) *cgaIterStack {
	return &cgaIterStack{
		partitionIndexes: make([]int, 0, numPartition),
	}
}

func (s *cgaIterStack) init() {
	s.i = -1
	s.partitionIndexes = s.partitionIndexes[:0]
	s.initialized = true
}

func (s *cgaIterStack) currentIndex() int {
	return s.partitionIndexes[s.i]
}

func (s *cgaIterStack) nextIndex() (int, bool) {
	s.i++
	if s.i < len(s.partitionIndexes) {
		return s.partitionIndexes[s.i], true
	}
	return 0, false
}

func (s *cgaIterStack) hasNext() bool {
	return s.i < len(s.partitionIndexes)
}

// cgaIterative iterative version of cgaRecursive. Reduce function callstack and allocations.
// This function is slightly better than the recursive function:
// BenchmarkPartition3v3v3_Recursive-16    	    9598	    111168 ns/op	   57480 B/op	    2131 allocs/op.
// BenchmarkPartition3v3v3_Iterative-16    	    9853	    107154 ns/op	   42368 B/op	    1490 allocs/op.
func cgaIterative(data *Data, items *[]ItemSequence, partitions []*Partition) {
	depth := 0
	stacks := make([]*cgaIterStack, len(*items))
	for i := 0; i < len(*items); i++ {
		stacks[i] = newCgaStack(len(partitions))
	}

	for {
		if data.IsCanceled() {
			return
		}

		data.funcIteration++
		if data.funcIteration > iterationLimit {
			return
		}

		// all items are filled to partition
		if depth == len(*items) {
			data.itemIteration++
			if data.IsValid() {
				count := countDiff(partitions)
				// check the distance if lower than current minimum
				dist := distance(data.Partitions)
				if count < data.BestCountDiff || (count == data.BestCountDiff && dist < data.BestDistance) {
					data.BestCountDiff = count
					if data.BestPartitions == nil {
						data.BestPartitions = make([][]Item, len(data.Partitions))
					}
					setBest(data, dist, data.Partitions)
				}
			}
			//printPartition(data.Partitions, depth, data.itemIteration)

			// equivalent: unwind recursive call
			depth--
			if depth < 0 {
				return
			}

			continue
		}

		item := (*items)[depth]

		stack := stacks[depth]
		if !stack.initialized {
			// equivalent: operations before recursive call
			stack.init()

			// sort partitions from lowest to highest score to fill the item to the best heuristic value first
			sort.Slice(partitions, func(i, j int) bool {
				return partitions[i].Sum() < partitions[j].Sum()
			})

			// store partition indexes for current item to try
			for _, p := range partitions {
				stack.partitionIndexes = append(stack.partitionIndexes, p.index)
			}
		} else {
			// equivalent: operations after recursive call
			if !stack.continueIter {
				i := stack.currentIndex()
				data.Partitions[i].Pop()
			}
			stack.continueIter = false
		}

		// try to fill the item to all possible partitions
		if i, hasNext := stack.nextIndex(); hasNext {
			data.Partitions[i].Push(item)

			differentOrder := false
			if !data.IsOrderMatter {
				prevSequence := -1
				// to avoid combination generated with same partitions value regardless of the partition order, [ab],[cd] should be same as [cd],[ab]
				// the first element of partition should be in descending/ascending order with other partition
				// we only use ascending sequence here to avoid repetition
				// IsOrderMatter=true will behave like permutation
				for _, p := range data.Partitions {
					sequence := p.FirstSequence()
					if sequence == -1 {
						continue
					}
					if prevSequence > sequence {
						differentOrder = true
						break
					}
					prevSequence = sequence
				}
			}

			if differentOrder || data.Partitions[i].count > data.MaxCount {
				data.Partitions[i].Pop()
				stack.continueIter = true
				continue
			}

			// equivalent: recursive call with new parameter
			depth++
		} else {
			// equivalent: end of loop, we can reuse the stack data
			stack.initialized = false

			// equivalent: unwind recursive call
			depth--
			if depth < 0 {
				return
			}

			continue
		}
	}
}

func cga(itemsToAdd []Item, numPartition int, recursive bool) *Data {
	return RunCga(itemsToAdd, nil, numPartition, CgaOptions{UseRecursive: recursive})
}

// RunCga run complete greedy algorithm partitioning.
// This function is to fulfill findAllyRebalance
// Reference: https://en.wikipedia.org/wiki/Multiway_number_partitioning#Exact_algorithms
//
//  1. Sort items to add from the highest value to lowest
//  2. Using DFS try to fill the items to the partitions.
//     a. Get current item, try to fill to the all possible partitions from the lowest sum value to highest.
//     b. Repeat for next items until all items are filled.
//
// Example values [1,2,3,4] with 2 partitions, max 2 members for each partition
//
//	 i: iteration, d: depth, sd: sum distance
//		i│ d│ values       │sd  │distance
//		─┼──┼──────────────┼────┼──────────
//		1│ 0│ () ()        │  - │ -
//		1│ 1│ (4) ()       │  - │ -
//		1│ 2│ (4) (3)      │  - │ -
//		1│ 3│ (4) (3,2)    │  - │ -
//		1│ 4│ (4,1) (3,2)  │  0 │ 0 [best]
//		2│ 3│ (4,2) (3)    │  - │ -
//		2│ 4│ (4,2) (3,1)  │  2 │ 1
//		3│ 2│ (4,3) ()     │  - │ -
//		3│ 3│ (4,3) (2)    │  - │ -
//		3│ 4│ (4,3) (2,1)  │  4 │ 2
//		4│ 1│ () (4)       │  - │ -
//		4│ 2│ (3) (4)      │  - │ -
//		4│ 3│ (3,2) (4)    │  - │ -
//		4│ 4│ (3,2) (4,1)  │  0 │ 0 [best]
//		5│ 3│ (3) (4,2)    │  - │ -
//		5│ 4│ (3,1) (4,2)  │  2 │ 1
//		6│ 2│ () (4,3)     │  - │ -
//		6│ 3│ (2) (4,3)    │  - │ -
//		6│ 4│ (2,1) (4,3)  │  4 │ 2
//
//		Tree representation:
//	                   ┌─────────────────┴────────────────────┐
//	                 (4)()                                  ()(4)
//	           ┌───────┴─────────┐                   ┌────────┴───────┐
//	       (4)(3)            (4,3)()              (3)(4)           ()(4,3)
//	     ┌─────┴─────┐           │             ┌─────┴────┐           │
//	 (4)(3,2)    (4,2)(3)    (4,3)(2)      (3,2)(4)    (3)(4,2)    (2)(4,3)
//	     │           │           │             │          │           │
//	 (4,1)(3,2)  (4,2)(3,1)  (4,3)(2,1)    (3,2)(4,1)  (3,1)(4,2)  (2,1)(4,3)
//
// There are 2 best values with same members but different partition order.
// We can prune it by checking the first sequence for each partition.
// Ordered values for [1,2,3,4] are [4,3,2,1]. If the first partition value is 4 then next partition's first value can only be [3,2 or 1].
// From the example, by checking the partition's first sequence we can skip 3rd to 6th iteration.
func RunCga(itemsToAdd []Item, existingPartition []*Partition, numPartition int, option CgaOptions) *Data {
	// this partition will be sort from lowest to highest each funcIteration
	// the low score partition will be filled first to minimize difference across partition
	partitions := existingPartition
	// fill up the remaining with empty partition
	for i := 0; i < len(existingPartition); i++ {
		if partitions[i] == nil {
			partitions[i] = &Partition{}
		}
	}
	for i := len(existingPartition); i < numPartition; i++ {
		partitions = append(partitions, &Partition{})
	}
	for i := range partitions {
		partitions[i].index = i
	}
	// this partition order is unchanged, because we use partition index, we need to point to correct item
	originalPartitions := make([]*Partition, numPartition)
	copy(originalPartitions, partitions)

	// printPartition(originalPartitions, 0, 0)

	totalCount := 0
	seq := 0
	for i := range existingPartition {
		for j := range existingPartition[i].values {
			existingPartition[i].values[j].sequence = seq
			totalCount += existingPartition[i].values[j].Count()
			seq++
		}
	}

	items := make([]ItemSequence, len(itemsToAdd))
	for i, item := range itemsToAdd {
		items[i] = ItemSequence{
			Item: item,
		}
		totalCount += item.Count()
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Value() > items[j].Value()
	})

	// sequence should be set after sorting to minimize skipping on sequence order checking
	for i := range items {
		items[i].sequence = seq
		seq++
	}

	var maxCount int
	if option.MaxCount != 0 {
		maxCount = option.MaxCount
	} else {
		maxCount = int(math.Ceil(float64(totalCount) / float64(numPartition)))
	}
	data := &Data{
		Partitions:     originalPartitions,
		BestDistance:   math.MaxFloat64,
		BestCountDiff:  math.MaxInt32,
		BestPartitions: nil,
		MaxCount:       maxCount,
		IsOrderMatter:  option.IsOrderMatter,
		MaxIteration:   option.MaxIteration,
		ctx:            option.Ctx,
		validateFunc:   option.ValidateFunc,
	}

	if option.UseRecursive {
		cgaRecursive(data, &items, partitions, 0)
	} else {
		cgaIterative(data, &items, partitions)
	}

	return data
}
