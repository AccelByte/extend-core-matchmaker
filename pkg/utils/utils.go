// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package utils

import (
	"strings"

	"github.com/google/uuid"
)

// GetMapValueAs get and cast to a type
func GetMapValueAs[T any](m map[string]interface{}, key string) (t T, ok bool) {
	var v interface{}
	if m == nil {
		return t, false
	}
	if v, ok = m[key]; !ok {
		return t, false
	}
	switch val := v.(type) {
	case T:
		return val, true
	default:
		return t, false
	}
}

// Contains return true if val exist in list, else return false.
func Contains[T comparable](list []T, val T) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}

// GenerateUUID generates uuid without hyphens.
func GenerateUUID() string {
	id, _ := uuid.NewRandom()
	return strings.ReplaceAll(id.String(), "-", "")
}

func HasSameElement(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	m1 := make(map[string]bool, len(s1))
	for _, v := range s1 {
		m1[v] = true
	}
	for _, v := range s2 {
		if !m1[v] {
			return false
		}
	}
	return true
}

func IntersectionOfStringLists(stringLists ...[]string) []string {
	neededCount := len(stringLists)
	countMap := make(map[string]int)
	intersection := make([]string, 0)
	for _, stringList := range stringLists {
		seenStringAlreadyInThisList := make(map[string]struct{})
		for _, str := range stringList {
			if _, yes := seenStringAlreadyInThisList[str]; yes {
				continue
			}
			seenStringAlreadyInThisList[str] = struct{}{}
			countMap[str] += 1
			if countMap[str] == neededCount {
				intersection = append(intersection, str)
			}
		}
	}
	return intersection
}
