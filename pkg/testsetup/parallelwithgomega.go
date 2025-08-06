// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package testsetup

import (
	"testing"

	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/onsi/gomega"
)

type GomegaWithScope struct {
	TestScope *envelope.Scope
	*gomega.GomegaWithT
}

func ParallelWithGomega(t *testing.T) GomegaWithScope {
	t.Parallel()
	return GomegaWithScope{NewTestScope(), gomega.NewGomegaWithT(t)}
}

func WithGomega(t *testing.T) GomegaWithScope {
	return GomegaWithScope{NewTestScope(), gomega.NewGomegaWithT(t)}
}
