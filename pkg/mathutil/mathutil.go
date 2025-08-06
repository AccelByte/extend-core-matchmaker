// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package mathutil

import "cmp"

// Max returns the larger of x and y.
func Max[T cmp.Ordered](x T, y T) T {
	return max(x, y)
}

// Min returns the smaller of x and y.
func Min[T cmp.Ordered](x T, y T) T {
	return min(x, y)
}
