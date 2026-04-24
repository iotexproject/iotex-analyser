package main

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
)

func getCandidateAddressByName(name string, height uint64) (string, error) {
	candidate := &models.Candidate{}
	if err := candidate.FetchByNameWithHeight(name, height); err != nil {
		return "", err
	}
	return candidate.OwnerAddress, nil
}

type bucketStateRow struct {
	BucketID         uint64 `gorm:"column:bucket_id"`
	OwnerAddress     string `gorm:"column:owner_address"`
	Candidate        string `gorm:"column:candidate"`
	AutoStake        bool   `gorm:"column:auto_stake"`
	Duration         uint32 `gorm:"column:duration"`
	TotalAmount      string `gorm:"column:total_amount"`
	NonUnstakeAmount string `gorm:"column:non_unstake_amount"`
	UnstakeCount     int64  `gorm:"column:unstake_count"`
	PositiveRowCount int64  `gorm:"column:positive_row_count"`
}

type BucketInfo struct {
	OwnerAddress string
	Candidate    string
	AutoStake    bool
	Duration     uint32
}

func (b *stakingActionChPlugin) ensureBucketStateCache() {
	if b.bucketStateCache == nil {
		b.bucketStateCache = newBucketStateCache(defaultBucketStateCacheSize)
	}
	if b.pendingBucketInfo == nil {
		b.pendingBucketInfo = make(map[uint64]*pendingBucketState)
	}
}

func collectBucketIDs(blks []*block.Block) []uint64 {
	seen := make(map[uint64]struct{})
	bucketIDs := make([]uint64, 0)
	for _, blk := range blks {
		for _, selp := range blk.Actions {
			var bucketID uint64
			switch a := selp.Action().(type) {
			case *action.TransferStake:
				bucketID = a.BucketIndex()
			case *action.Restake:
				bucketID = a.BucketIndex()
			case *action.ChangeCandidate:
				bucketID = a.BucketIndex()
			case *action.DepositToStake:
				bucketID = a.BucketIndex()
			case *action.Unstake:
				bucketID = a.BucketIndex()
			default:
				continue
			}
			if _, ok := seen[bucketID]; ok {
				continue
			}
			seen[bucketID] = struct{}{}
			bucketIDs = append(bucketIDs, bucketID)
		}
	}
	return bucketIDs
}

func (b *stakingActionChPlugin) preloadBucketStates(bucketIDs []uint64) error {
	if len(bucketIDs) == 0 {
		return nil
	}
	b.ensureBucketStateCache()

	missing := make([]uint64, 0, len(bucketIDs))
	seen := make(map[uint64]struct{}, len(bucketIDs))
	for _, bucketID := range bucketIDs {
		if _, ok := seen[bucketID]; ok {
			continue
		}
		seen[bucketID] = struct{}{}
		if _, ok := b.bucketStateCache.Get(bucketID); ok {
			continue
		}
		missing = append(missing, bucketID)
	}
	if len(missing) == 0 {
		return nil
	}

	var rows []bucketStateRow
	if err := chDB.Raw(`
		SELECT
			bucket_id,
			argMaxIf(owner_address, tuple(block_height, "index"), toDecimal256(amount, 10) > 0) AS owner_address,
			argMaxIf(candidate, tuple(block_height, "index"), toDecimal256(amount, 10) > 0) AS candidate,
			argMaxIf(auto_stake, tuple(block_height, "index"), toDecimal256(amount, 10) > 0) AS auto_stake,
			argMaxIf(duration, tuple(block_height, "index"), toDecimal256(amount, 10) > 0) AS duration,
			toString(sum(toDecimal256(amount, 10))) AS total_amount,
			toString(sumIf(toDecimal256(amount, 10), act_type <> 'Unstake')) AS non_unstake_amount,
			countIf(act_type = 'Unstake') AS unstake_count,
			countIf(toDecimal256(amount, 10) > 0) AS positive_row_count
		FROM staking_actions
		WHERE bucket_id IN ?
		GROUP BY bucket_id
	`, missing).Scan(&rows).Error; err != nil {
		return err
	}

	rowMap := make(map[uint64]bucketStateRow, len(rows))
	for _, row := range rows {
		rowMap[row.BucketID] = row
	}
	for _, bucketID := range missing {
		row, ok := rowMap[bucketID]
		if !ok {
			b.bucketStateCache.Set(bucketID, cachedBucketState{})
			continue
		}
		totalAmount, err := parseDecimal(row.TotalAmount)
		if err != nil {
			return fmt.Errorf("parse total amount for bucket %d: %w", bucketID, err)
		}
		nonUnstakeAmount, err := parseDecimal(row.NonUnstakeAmount)
		if err != nil {
			return fmt.Errorf("parse non-unstake amount for bucket %d: %w", bucketID, err)
		}
		b.bucketStateCache.Set(bucketID, cachedBucketState{
			info: BucketInfo{
				OwnerAddress: row.OwnerAddress,
				Candidate:    row.Candidate,
				AutoStake:    row.AutoStake,
				Duration:     row.Duration,
			},
			hasInfo:          row.PositiveRowCount > 0,
			totalAmount:      totalAmount,
			nonUnstakeAmount: nonUnstakeAmount,
			unstakeCount:     row.UnstakeCount,
		})
	}
	return nil
}

func (b *stakingActionChPlugin) getBucketSumAmountFromCacheByBucketID(bucketID uint64) (decimal.Decimal, error) {
	if err := b.preloadBucketStates([]uint64{bucketID}); err != nil {
		return decimal.Zero, err
	}
	state, _ := b.bucketStateCache.Get(bucketID)
	total := state.totalAmount
	if pending, ok := b.pendingBucketInfo[bucketID]; ok {
		total = total.Add(pending.totalDelta)
	}
	return total, nil
}

func (b *stakingActionChPlugin) getFixBucketSumAmountFromCacheByBucketID(bucketID uint64) (decimal.Decimal, error) {
	if err := b.preloadBucketStates([]uint64{bucketID}); err != nil {
		return decimal.Zero, err
	}
	state, _ := b.bucketStateCache.Get(bucketID)
	unstakeCount := state.unstakeCount
	nonUnstakeAmount := state.nonUnstakeAmount
	if pending, ok := b.pendingBucketInfo[bucketID]; ok {
		unstakeCount += pending.unstakeCountDelta
		nonUnstakeAmount = nonUnstakeAmount.Add(pending.nonUnstakeDelta)
	}
	if unstakeCount == 0 {
		return decimal.Zero, nil
	}
	return nonUnstakeAmount, nil
}

func (b *stakingActionChPlugin) getBucketInfoAddressFromCacheByBucketID(bucketID uint64) (*BucketInfo, error) {
	if pending, ok := b.pendingBucketInfo[bucketID]; ok && pending.latestInfo != nil {
		info := *pending.latestInfo
		return &info, nil
	}
	if err := b.preloadBucketStates([]uint64{bucketID}); err != nil {
		return nil, err
	}
	state, _ := b.bucketStateCache.Get(bucketID)
	info := state.info
	return &info, nil
}

func (b *stakingActionChPlugin) recordPendingStakingAction(actions ...*StakingActions) {
	b.ensureBucketStateCache()
	for _, stakingAction := range actions {
		pending, ok := b.pendingBucketInfo[stakingAction.BucketID]
		if !ok {
			pending = &pendingBucketState{}
			b.pendingBucketInfo[stakingAction.BucketID] = pending
		}
		info := &BucketInfo{
			OwnerAddress: stakingAction.OwnerAddress,
			Candidate:    stakingAction.Candidate,
			AutoStake:    stakingAction.AutoStake,
			Duration:     stakingAction.Duration,
		}
		pending.latestInfo = info
		pending.totalDelta = pending.totalDelta.Add(stakingAction.Amount)
		if stakingAction.ActType == "Unstake" {
			pending.unstakeCountDelta++
		} else {
			pending.nonUnstakeDelta = pending.nonUnstakeDelta.Add(stakingAction.Amount)
		}
		if stakingAction.Amount.GreaterThan(decimal.Zero) {
			pending.latestPositiveInfo = info
		}
	}
}

func (b *stakingActionChPlugin) applyPendingBucketStates() {
	b.ensureBucketStateCache()
	for bucketID, pending := range b.pendingBucketInfo {
		state, _ := b.bucketStateCache.Get(bucketID)
		state.totalAmount = state.totalAmount.Add(pending.totalDelta)
		state.nonUnstakeAmount = state.nonUnstakeAmount.Add(pending.nonUnstakeDelta)
		state.unstakeCount += pending.unstakeCountDelta
		if pending.latestPositiveInfo != nil {
			state.info = *pending.latestPositiveInfo
			state.hasInfo = true
		}
		b.bucketStateCache.Set(bucketID, state)
	}
}

func parseDecimal(value string) (decimal.Decimal, error) {
	if value == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(value)
}
