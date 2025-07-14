// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v2

import (
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/rebalance"

	"github.com/mitchellh/copystructure"
	"github.com/sirupsen/logrus"
)

var countDistance = rebalance.CountDistance

func CopyAllies(allies []models.MatchingAlly) interface{} {
	copied, err := copystructure.Copy(allies)
	if err != nil {
		logrus.Warn("failed copy allies:", err)
		return nil
	}
	return copied
}

func AllUnique(l1 [][]int) bool {
	if len(l1) == 0 || len(l1[0]) == 0 {
		return true
	}
	m := map[int]struct{}{}
	for _, l2 := range l1 {
		for _, x := range l2 {
			if _, found := m[x]; found {
				return false
			}
			m[x] = struct{}{}
		}
	}
	return true
}

func CountPartyAndPlayer(allies []models.MatchingAlly) (party, player int) {
	for _, ally := range allies {
		party += len(ally.MatchingParties)
		player += ally.CountPlayer()
	}
	return party, player
}

func IsLockedPartyExist(allies []models.MatchingAlly) bool {
	for _, ally := range allies {
		for _, party := range ally.MatchingParties {
			if party.Locked {
				return true
			}
		}
	}
	return false
}
