// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package models

import (
	"errors"
)

var (
	ValidationErrorTotalCombination = errors.New("total combination alliance should equal max ally")
	ValidationErrorZeroTotalMaxRole = errors.New("total max role cannot be 0")
	ValidationErrorTotalMinRole     = errors.New("total min role for each ally should not exceed max player number")
	ValidationErrorTotalMaxRole     = errors.New("total max role for each ally should not less than min player number")
	ValidationErrorMaxRole          = errors.New("max role should not exceed max player number, its just unnecessary")
)

var validationErrorCodeMap = map[error]int{
	ValidationErrorTotalCombination: 510115,
	ValidationErrorZeroTotalMaxRole: 510116,
	ValidationErrorTotalMinRole:     510117,
	ValidationErrorTotalMaxRole:     510118,
	ValidationErrorMaxRole:          510119,
}

// ValidationErrorCode returns a code for the error.
// It returns log.EIDValidationErrorV1 (20002) if the error is not registered in the map.
func ValidationErrorCode(err error) int {
	code, ok := validationErrorCodeMap[err]
	if !ok {
		return 20002
	}
	return code
}
