// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package reordertool

// Options for reorder tool.
type Options struct {
	/*
		SkipEmpty related with empty input (default: FALSE),

		SkipEmpty: FALSE will go inside HasNext() first,

		SkipEmpty: TRUE make HasNext() directly return false even in first loop,

		Example case:

		-	SkipEmpty FALSE is useful when tickets we have consist only priorityTickets,
		current flow will put priorityTickets in first elements and skip from reordering it,
		so we need to proceed empty otherTickets and not skipping it so we able to proceed the priorityTickets
	*/
	SkipEmpty bool

	/*
		MaxLoop limit the maximum number of combinations
		that will be returned by this function,

		HasNext() will return false when total loop reach this value
	*/
	MaxLoop int

	/*
		MaxSecond limit maximum process time taken in second,

		HasNext() will return false when process time reach this value
	*/
	MaxSecond int

	/*
		ElementsAlwaysFirst list all ids to be always become first elements
	*/
	ElementsAlwaysFirst []int
}
