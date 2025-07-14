// Copyright (c) 2024 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package partition

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var functionVariants = []string{"recursive", "iterative"}

const functionVariantRecursive = "recursive"

func TestPartitionResult(t *testing.T) {
	numPartition := 3
	numbers := []int{9, 8, 7, 6, 5, 4, 3}
	items := make([]Item, len(numbers))
	for i, n := range numbers {
		items[i] = IntValue{value: n}
	}

	bestDistance := math.MaxInt
	numIteration := 0
	for _, variant := range functionVariants {
		recursive := variant == functionVariantRecursive
		t.Run(variant, func(t *testing.T) {
			data := cga(items, numPartition, recursive)

			if recursive {
				bestDistance = int(data.BestDistance)
				numIteration = data.NumIteration()
			} else {
				assert.Equal(t, bestDistance, int(data.BestDistance), "iterative value should same with recursive value")
				assert.Equal(t, numIteration, data.NumIteration(), "iterative value should same with recursive value")
			}

			require.Less(t, data.BestDistance, 2.4)
			require.NotNil(t, data.BestPartitions)
			require.Len(t, data.BestPartitions, numPartition)
		})
	}
}

func TestPartitionWithExistingItems(t *testing.T) {
	numPartition := 3
	numbers := []int{8, 7, 5, 4}
	items := make([]Item, len(numbers))
	for i, n := range numbers {
		items[i] = IntValue{value: n}
	}
	partitions := make([]*Partition, numPartition)
	for i := 0; i < numPartition; i++ {
		partitions[i] = &Partition{}
	}
	partitions[0].PushItem(IntValue{value: 9})
	partitions[1].PushItem(IntValue{value: 6})
	partitions[2].PushItem(IntValue{value: 3})

	bestDistance := math.MaxInt
	numIteration := 0

	for _, variant := range functionVariants {
		recursive := variant == functionVariantRecursive
		t.Run(variant, func(t *testing.T) {
			data := RunCga(items, partitions, numPartition, CgaOptions{UseRecursive: recursive})

			if recursive {
				bestDistance = int(data.BestDistance)
				numIteration = data.NumIteration()
			} else {
				assert.Equal(t, bestDistance, int(data.BestDistance), "iterative value should same with recursive value")
				assert.Equal(t, numIteration, data.NumIteration(), "iterative value should same with recursive value")
			}

			require.Less(t, data.BestDistance, 2.4)
			require.NotNil(t, data.BestPartitions)
			require.Len(t, data.BestPartitions, numPartition)
		})
	}
}

func TestPartition3v3v3(t *testing.T) {
	numPartition := 3
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	items := make([]Item, len(numbers))
	for i, n := range numbers {
		items[i] = IntValue{value: n}
	}

	bestDistance := math.MaxInt
	numIteration := 0
	for _, variant := range functionVariants {
		recursive := variant == functionVariantRecursive
		t.Run(variant, func(t *testing.T) {
			data := cga(items, numPartition, recursive)

			if recursive {
				bestDistance = int(data.BestDistance)
				numIteration = data.NumIteration()
			} else {
				assert.Equal(t, bestDistance, int(data.BestDistance), "iterative value should same with recursive value")
				assert.Equal(t, numIteration, data.NumIteration(), "iterative value should same with recursive value")
			}

			require.Zero(t, data.BestDistance)
			require.NotNil(t, data.BestPartitions)
			require.Len(t, data.BestPartitions, numPartition)
		})
	}
}

func BenchmarkPartition3v3v3_Recursive(b *testing.B) {
	numPartition := 3
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	items := make([]Item, len(numbers))
	for i, n := range numbers {
		items[i] = IntValue{value: n}
	}

	for i := 0; i < b.N; i++ {
		data := cga(items, numPartition, true)
		require.Zero(b, data.BestDistance)
	}
}

func BenchmarkPartition3v3v3_Iterative(b *testing.B) {
	numPartition := 3
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	items := make([]Item, len(numbers))
	for i, n := range numbers {
		items[i] = IntValue{value: n}
	}

	for i := 0; i < b.N; i++ {
		data := cga(items, numPartition, false)
		require.Zero(b, data.BestDistance)
	}
}

func TestPartitionMulti(t *testing.T) {
	numPartition := 2
	numbers := [][]int{{1, 2}, {3, 4}}
	items := make([]Item, len(numbers))
	for i, n := range numbers {
		items[i] = IntSliceValue{items: n, index: i}
	}

	bestDistance := math.MaxInt
	numIteration := 0
	for _, variant := range functionVariants {
		recursive := variant == functionVariantRecursive
		t.Run(variant, func(t *testing.T) {
			data := cga(items, numPartition, recursive)

			if recursive {
				bestDistance = int(data.BestDistance)
				numIteration = data.NumIteration()
			} else {
				assert.Equal(t, bestDistance, int(data.BestDistance), "iterative value should same with recursive value")
				assert.Equal(t, numIteration, data.NumIteration(), "iterative value should same with recursive value")
			}

			require.Equal(t, 2, int(data.BestDistance))
			require.NotNil(t, data.BestPartitions)
			require.Len(t, data.BestPartitions, numPartition)
			require.Equal(t, 1, data.NumIteration())
		})
	}
}

func TestPartitionMulti2(t *testing.T) {
	numPartition := 2
	numbers := [][]int{{1, 2}, {3, 4}, {5}, {6}}
	items := make([]Item, len(numbers))
	for i, n := range numbers {
		items[i] = IntSliceValue{items: n, index: i}
	}

	bestDistance := math.MaxInt
	numIteration := 0
	for _, variant := range functionVariants {
		recursive := variant == functionVariantRecursive
		t.Run(variant, func(t *testing.T) {
			data := cga(items, numPartition, recursive)

			if recursive {
				bestDistance = int(data.BestDistance)
				numIteration = data.NumIteration()
			} else {
				assert.Equal(t, bestDistance, int(data.BestDistance), "iterative value should same with recursive value")
				assert.Equal(t, numIteration, data.NumIteration(), "iterative value should same with recursive value")
			}

			require.Equal(t, 1, int(data.BestDistance))
			require.NotNil(t, data.BestPartitions)
			require.Len(t, data.BestPartitions, numPartition)
			require.Equal(t, 2, data.NumIteration()) // 2 combinations
		})
	}
}

func TestPartitionMulti_Backfill(t *testing.T) {
	numPartition := 2
	numbers := [][]int{{10}, {20, 20, 20}, {30, 30}}
	items := make([]Item, len(numbers))
	for i, n := range numbers {
		items[i] = IntSliceValue{items: n}
	}
	partitions := make([]*Partition, numPartition)
	for i := 0; i < numPartition; i++ {
		partitions[i] = &Partition{}
	}
	partitions[0].PushItem(IntSliceValue{items: []int{10, 10}})
	partitions[1].PushItem(IntSliceValue{items: []int{20, 20, 20}})

	bestDistance := math.MaxInt
	numIteration := 0

	for _, variant := range functionVariants {
		recursive := variant == functionVariantRecursive
		t.Run(variant, func(t *testing.T) {
			data := RunCga(items, partitions, numPartition, CgaOptions{UseRecursive: recursive, MaxCount: 10})

			if recursive {
				bestDistance = int(data.BestDistance)
				numIteration = data.NumIteration()
			} else {
				assert.Equal(t, bestDistance, int(data.BestDistance), "iterative value should same with recursive value")
				assert.Equal(t, numIteration, data.NumIteration(), "iterative value should same with recursive value")
			}

			require.Equal(t, 2, int(data.BestDistance))
			require.NotNil(t, data.BestPartitions)
			require.Len(t, data.BestPartitions, numPartition)
		})
	}
}

func TestPartitionMulti_Permutation(t *testing.T) {
	numPartition := 2
	numbers := [][]int{{1, 2}, {3, 4}, {5}, {6}}
	items := make([]Item, len(numbers))
	for i, n := range numbers {
		items[i] = IntSliceValue{items: n, index: i}
	}

	bestDistance := math.MaxInt
	numIteration := 0
	for _, variant := range functionVariants {
		recursive := variant == functionVariantRecursive
		t.Run(variant, func(t *testing.T) {
			data := RunCga(items, nil, numPartition, CgaOptions{
				UseRecursive:  recursive,
				IsOrderMatter: true,
			})

			if recursive {
				bestDistance = int(data.BestDistance)
				numIteration = data.NumIteration()
			} else {
				assert.Equal(t, bestDistance, int(data.BestDistance), "iterative value should same with recursive value")
				assert.Equal(t, numIteration, data.NumIteration(), "iterative value should same with recursive value")
			}

			require.NotNil(t, data.BestPartitions)
			require.Len(t, data.BestPartitions, numPartition)
		})
	}
}

func TestPartition_BigPartitions(t *testing.T) {
	numPartition := 8
	numbers := [][]int{
		{3280}, {1020}, {3010},
		{1090}, {1230}, {3240},
		{1570}, {1560}, {1550},
		{2870}, {2160}, {1350},
		{1240}, {1330}, {1720},
		{2240}, {3070}, {2910},
		{1810}, {2910}, {2740},
		{1810}, {2910}, {2740},
	}
	items := make([]Item, len(numbers))
	for i, n := range numbers {
		items[i] = IntSliceValue{items: n, index: i}
	}

	var bestPartitions [][]Item
	maxIteration := 1000
	bestDistance := math.MaxInt
	numIteration := 0
	for _, variant := range functionVariants {
		recursive := variant == functionVariantRecursive
		t.Run(variant, func(t *testing.T) {
			data := RunCga(items, nil, numPartition, CgaOptions{
				UseRecursive:  recursive,
				IsOrderMatter: true,
				MaxIteration:  maxIteration,
			})

			if recursive {
				bestDistance = int(data.BestDistance)
				numIteration = data.NumIteration()
				bestPartitions = data.BestPartitions
			} else {
				assert.Equal(t, bestDistance, int(data.BestDistance), "iterative value should same with recursive value")
				assert.Equal(t, numIteration, data.NumIteration(), "iterative value should same with recursive value")
				assert.Equal(t, bestPartitions, data.BestPartitions, "iterative value should same with recursive value")
			}

			assert.Equal(t, maxIteration, data.NumIteration())

			require.NotNil(t, data.BestPartitions)
			require.Len(t, data.BestPartitions, numPartition)
		})
	}
}
