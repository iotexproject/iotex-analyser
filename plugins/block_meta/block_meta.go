package main

import (
	"context"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

type blockMetaPlugin struct {
}

func (b blockMetaPlugin) Name() string {
	return "block_meta"
}

func (b blockMetaPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockMetaPlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&BlockMeta{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b blockMetaPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	var gasConsumed uint64
	grantRewardActs := make(map[hash.Hash256]bool)
	// log action index
	for _, selp := range blk.Actions {
		if _, ok := selp.Action().(*action.GrantReward); ok {
			grantRewardActs[selp.Hash()] = true
		}
	}
	totalReward := big.NewInt(0)
	// log receipt index
	for _, receipt := range blk.Receipts {
		gasConsumed += receipt.GasConsumed
		if _, ok := grantRewardActs[receipt.ActionHash]; ok {
			// Parse receipt of grant reward
			rewardInfoMap, err := getRewardInfoFromReceipt(receipt)
			if err != nil {
				return errors.Wrap(err, "failed to get reward info from receipt")
			}
			if len(rewardInfoMap) == 0 {
				continue
			}
			for _, rewards := range rewardInfoMap {
				totalReward.Add(totalReward, rewards.BlockReward)
			}
		}
	}
	bm := &BlockMeta{
		BlockHeight:     blk.Height(),
		GasConsumed:     gasConsumed,
		ProducerName:    "",
		ProducerAddress: "",
		BlockReward:     decimal.NewFromBigInt(totalReward, 0),
	}

	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(bm).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b blockMetaPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockMetaPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockMetaPlugin{}
