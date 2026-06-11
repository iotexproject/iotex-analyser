package main

import (
	"context"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.1.3"

type blockRewardPlugin struct {
	tipHeight uint64
	rewards   []models.BlockReward
}

func (b *blockRewardPlugin) Name() string {
	return "block_reward"
}

func (b *blockRewardPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *blockRewardPlugin) DependentPlugins() []string {
	return []string{"candidate"}
}

func (b *blockRewardPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.BlockReward{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b *blockRewardPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)

	grantRewardActs := make(map[hash.Hash256]bool)
	for _, selp := range blk.Actions {
		if _, ok := selp.Action().(*action.GrantReward); ok {
			actHash, _ := selp.Hash()
			grantRewardActs[actHash] = true
		}
	}

	for _, receipt := range blk.Receipts {
		receipt := receipt
		if _, ok := grantRewardActs[receipt.ActionHash]; !ok {
			continue
		}
		rewardInfoMap, err := kernel.RewardInfoFromReceipt(receipt)
		if err != nil {
			return errors.Wrap(err, "failed to get reward info from receipt")
		}
		rows, err := collectRewardRows(blkHeight, epochNum, receipt, rewardInfoMap)
		if err != nil {
			return errors.Wrap(err, "failed to collect reward rows")
		}
		b.rewards = append(b.rewards, rows...)
	}
	return nil
}

func (b *blockRewardPlugin) commit() error {
	rewards := b.rewards
	b.rewards = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(rewards) > 0 {
			if err := tx.Model(&models.BlockReward{}).CreateInBatches(rewards, 200).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *blockRewardPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *blockRewardPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *blockRewardPlugin) BatchSize() int {
	return 1000
}

func (b *blockRewardPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *blockRewardPlugin) CatchUpSafe() bool { return true }

func (b *blockRewardPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockRewardPlugin{}
