// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package models

import (
	"gopkg.in/typ.v4/sync2"
)

// Pool reusable objects to reduce garbage collector
type Pool struct {
	PartyMembers *sync2.Pool[[]PartyMember]
}

func NewPool() *Pool {
	return &Pool{
		PartyMembers: &sync2.Pool[[]PartyMember]{
			New: func() []PartyMember {
				return make([]PartyMember, 0, 12)
			},
		},
	}
}
