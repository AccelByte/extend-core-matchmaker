// Copyright (c) 2024 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package partition

import "fmt"

type IntValue struct {
	value int
}

func (v IntValue) Value() float64 {
	return float64(v.value)
}

func (v IntValue) ID() int {
	return v.value
}

func (v IntValue) Count() int {
	return 1
}

func (v IntValue) String() string {
	return fmt.Sprintf("%v", v.value)
}

type IntSliceValue struct {
	items []int
	index int
}

func (v IntSliceValue) Value() float64 {
	value := float64(0)
	for _, item := range v.items {
		value += float64(item)
	}
	return value
}

func (v IntSliceValue) ID() int {
	return v.items[0]
}

func (v IntSliceValue) Count() int {
	return len(v.items)
}

func (v IntSliceValue) String() string {
	return fmt.Sprintf("%v", v.items)
}
