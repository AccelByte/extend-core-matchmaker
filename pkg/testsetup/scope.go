// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package testsetup

import (
	"context"

	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/sirupsen/logrus"
)

// NewTestScope creates a new scope for test use
func NewTestScope() *envelope.Scope {
	return envelope.NewRootScope(context.Background(), "test", "")
}

// NewTestScopeWithLogger creates a new scope using the given logger for test use
func NewTestScopeWithLogger(logger *logrus.Logger) *envelope.Scope {
	scope := envelope.NewRootScope(context.Background(), "test", "")
	scope.SetLogger(logger)
	return scope
}
