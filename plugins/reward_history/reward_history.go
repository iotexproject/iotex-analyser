package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// VERSION 2.2.0 — Zanzibar: resolve delegate names by owner address as well as
// reward address, and stamp which of the two a payout landed on.
const VERSION = "2.2.0"

const (
	payoutKindReward = "reward"
	payoutKindOwner  = "owner"
)

var (
	RewardAddrToName map[string][]string
	// PayoutAddrKind maps a payout address to the candidate role it plays.
	PayoutAddrKind map[string]string
)

func addName(addr, name string) {
	if addr == "" {
		return
	}
	for _, existing := range RewardAddrToName[addr] {
		if existing == name {
			return
		}
	}
	RewardAddrToName[addr] = append(RewardAddrToName[addr], name)
}

type rewardHistoryPlugin struct {
	tipHeight uint64
	histories []models.RewardHistory
}

func (b *rewardHistoryPlugin) Name() string {
	return "reward_history"
}

func (b *rewardHistoryPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *rewardHistoryPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.RewardHistory{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b *rewardHistoryPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNumber := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNumber)
	chainClient := kernel.ChainClient()
	if blkHeight == epochHeight && blkHeight >= kernel.FairbankEffectiveHeight() {
		preEpochHeight := kernel.GetEpochHeight(epochNumber - 1)
		candidateList, err := GetAllStakingCandidates(chainClient, preEpochHeight)
		if err != nil {
			return errors.Wrap(err, "get candidate error")
		}
		RewardAddrToName = make(map[string][]string)
		PayoutAddrKind = make(map[string]string)
		for _, c := range candidateList.Candidates {
			// Index the owner address as well as the reward address. Since
			// Zanzibar, a delegate that opted in to IIP-59 is paid its
			// commission at its *owner* address, so a reward-address-only map
			// stops resolving exactly the delegates the fork affects — and a
			// blank candidate_name silently drops them out of every by-name
			// reward aggregation downstream.
			addName(c.RewardAddress, c.Name)
			addName(c.OwnerAddress, c.Name)
			// Record which index a payout address came from, so a consumer can
			// tell a full legacy payout from an IIP-59 commission-only payout
			// without a state read. When the two addresses coincide the reward
			// role wins and the distinction has to come from
			// delegate_reward_config instead.
			if c.OwnerAddress != "" && c.OwnerAddress != c.RewardAddress {
				PayoutAddrKind[c.OwnerAddress] = payoutKindOwner
			}
			PayoutAddrKind[c.RewardAddress] = payoutKindReward
		}
	}

	grantRewardActs := make(map[hash.Hash256]bool)
	for _, selp := range blk.Actions {
		if _, ok := selp.Action().(*action.GrantReward); ok {
			actHash, _ := selp.Hash()
			grantRewardActs[actHash] = true
		}
	}
	for _, receipt := range blk.Receipts {
		if _, ok := grantRewardActs[receipt.ActionHash]; !ok {
			continue
		}
		rewardInfoMap, err := kernel.RewardInfoFromReceipt(receipt)
		if err != nil {
			return err
		}
		if len(rewardInfoMap) == 0 {
			continue
		}
		for rewardAddress, reward := range rewardInfoMap {
			var candidateName string
			if len(RewardAddrToName[rewardAddress]) > 0 {
				candidateName = RewardAddrToName[rewardAddress][0]
			}
			b.histories = append(b.histories, models.RewardHistory{
				BlockHeight:     blkHeight,
				EpochNumber:     epochNumber,
				RewardAddress:   rewardAddress,
				PayoutAddrKind:  PayoutAddrKind[rewardAddress],
				ActionHash:      hex.EncodeToString(receipt.ActionHash[:]),
				CandidateName:   candidateName,
				BlockReward:     decimal.NewFromBigInt(reward.BlockReward, 0),
				EpochReward:     decimal.NewFromBigInt(reward.EpochReward, 0),
				FoundationBonus: decimal.NewFromBigInt(reward.FoundationBonus, 0),
			})
		}
	}
	return nil
}

func (b *rewardHistoryPlugin) commit() error {
	histories := b.histories
	b.histories = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(histories) > 0 {
			if err := tx.Model(&models.RewardHistory{}).CreateInBatches(histories, 200).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *rewardHistoryPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *rewardHistoryPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *rewardHistoryPlugin) BatchSize() int {
	return 1000
}

func (b *rewardHistoryPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *rewardHistoryPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = rewardHistoryPlugin{}
