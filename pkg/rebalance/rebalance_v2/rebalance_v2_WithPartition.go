// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package rebalance_v2

import (
	"context"
	"fmt"
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/envelope"
	"github.com/AccelByte/extend-core-matchmaker/pkg/models"
	"github.com/AccelByte/extend-core-matchmaker/pkg/utils/partition"

	"github.com/sirupsen/logrus"
)

const (
	rebalanceTimeout = 400 * time.Millisecond
)

type PartitionItem struct {
	value float64
	count int
	index int
}

func (v PartitionItem) Value() float64 {
	return v.value
}

func (v PartitionItem) ID() int {
	return v.index
}

func (v PartitionItem) Count() int {
	return v.count
}

func (v PartitionItem) String() string {
	return fmt.Sprintf("%v_%v_%v", v.index, v.count, v.value)
}

func rebalanceWithCGAPartition(
	rootScope *envelope.Scope,
	logFields logrus.Fields,
	lockedAllies map[int][]models.MatchingParty,
	partiesToAdd []models.MatchingParty,
	bestAllies map[int][]models.MatchingParty,
	activeAllianceRule models.AllianceRule,
	attributeNames []string,
	blockedPlayerOption models.BlockedPlayerOption,
	matchingRules []models.MatchingRule,
) []models.MatchingAlly {
	scope := rootScope.NewChildScope("rebalanceWithCGAPartition")
	defer scope.Finish()
	logFields["method"] = "rebalanceWithCGAPartition"

	// do rebalance with timeout duration
	ctx, cancel := context.WithTimeout(scope.Ctx, rebalanceTimeout)
	defer cancel()

	allParties := make([]models.MatchingParty, 0, 16)

	numItem := 0
	existingPartitions := make([]*partition.Partition, activeAllianceRule.MaxNumber)
	for i, matchingParties := range lockedAllies {
		if i < 0 || i >= activeAllianceRule.MaxNumber {
			logrus.WithField("index", i).Error("lockedAllies index out of bound")
			continue
		}
		p := &partition.Partition{}
		for _, party := range matchingParties {
			index := len(allParties)
			allParties = append(allParties, party)
			item := PartitionItem{
				value: party.Total(attributeNames, matchingRules),
				count: len(party.PartyMembers),
				index: index,
			}
			p.PushItem(item)
			numItem++
		}
		existingPartitions[i] = p
	}

	itemsToAdd := make([]partition.Item, 0, len(partiesToAdd))
	for _, party := range partiesToAdd {
		index := len(allParties)
		allParties = append(allParties, party)
		item := PartitionItem{
			value: party.Total(attributeNames, matchingRules),
			count: len(party.PartyMembers),
			index: index,
		}
		itemsToAdd = append(itemsToAdd, item)
	}
	numItem += len(itemsToAdd)

	isOrderMatter := activeAllianceRule.IsRoleBased() && !activeAllianceRule.IsSingleComboRoleBased()
	numPartitions := activeAllianceRule.MaxNumber

	allies := make([]models.MatchingAlly, 0, numPartitions)

	validateFunc := func(partitions []*partition.Partition) bool {
		allies = allies[:0]
		for _, p := range partitions {
			var ally models.MatchingAlly
			for _, item := range p.Values() {
				ally.MatchingParties = append(ally.MatchingParties, allParties[item.ID()])
			}
			if len(p.Values()) > 0 {
				allies = append(allies, ally)
			}
		}
		err := activeAllianceRule.ValidateAllies(allies, blockedPlayerOption)
		return err == nil
	}

	maxIteration := 0
	if numItem > 12 {
		// reduce iteration for high variance allies that most likely timeout before finding the optimal allies
		// using heuristic, the optimal or close to optimal allies are on the early iterations, it is very unlikely to find optimal allies on later iterations
		maxIteration = 1000
		// don't check partition order because too much checking happens rather than generating allies
		// on high variance, allies with same members but different allies order rarely happen at early iterations
		isOrderMatter = true
	}

	// PlayerMaxNumber pruning and validation is already integrated to cga
	// TBD: use ValidateAllyMaxOnly on cga for roleBased ruleset to optimize the cga function (prune more nodes)

	result := partition.RunCga(itemsToAdd, existingPartitions, numPartitions, partition.CgaOptions{
		Ctx:           ctx,
		UseRecursive:  false,
		IsOrderMatter: isOrderMatter,
		MaxCount:      activeAllianceRule.PlayerMaxNumber,
		MaxIteration:  maxIteration,
		ValidateFunc:  validateFunc,
	})

	logFields["iteration"] = result.NumIteration()

	if result.IsTimeout {
		logrus.WithFields(logFields).Warn("rebalance timeout")
	}

	best := Convert(bestAllies)
	if result.BestPartitions == nil {
		return best
	}

	bestMemberDiff := CountMemberDiff(best)
	bestDistance := countDistance(best, attributeNames, matchingRules)

	session := make([]models.MatchingAlly, 0, numPartitions)
	for _, p := range result.BestPartitions {
		var ally models.MatchingAlly
		for _, item := range p {
			ally.MatchingParties = append(ally.MatchingParties, allParties[item.ID()])
		}
		session = append(session, ally)
	}

	currentDistance := countDistance(session, attributeNames, matchingRules)
	currentMemberDiff := CountMemberDiff(session)

	if currentMemberDiff < bestMemberDiff || (currentMemberDiff <= bestMemberDiff && currentDistance <= bestDistance) {
		return session
	}

	return best
}
