package main

import (
	"context"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.1.3"

type blockRewardPlugin struct {
}

type RewardInfo struct {
	BlockReward     *big.Int
	EpochReward     *big.Int
	FoundationBonus *big.Int
}

func (b blockRewardPlugin) Name() string {
	return "block_reward"
}

func (b blockRewardPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockRewardPlugin) DependentPlugins() []string {
	return []string{"candidate"}
}

func (b blockRewardPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.BlockReward{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b blockRewardPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)

	grantRewardActs := make(map[hash.Hash256]bool)
	// log action index
	for _, selp := range blk.Actions {
		if _, ok := selp.Action().(*action.GrantReward); ok {
			actHash, _ := selp.Hash()
			grantRewardActs[actHash] = true
		}
	}
	err := db.DB().Transaction(func(tx *gorm.DB) error {

		// log receipt index
		for _, receipt := range blk.Receipts {
			receipt := receipt
			if _, ok := grantRewardActs[receipt.ActionHash]; !ok {
				continue
			}
			// Parse receipt of grant reward
			rewardInfoMap, err := getRewardInfoFromReceipt(receipt)
			if err != nil {
				return errors.Wrap(err, "failed to get reward info from receipt")
			}
			if err = handleRewardInfoMap(tx, blkHeight, epochNum, receipt, rewardInfoMap); err != nil {
				return errors.Wrap(err, "failed to handle reward info map")
			}

		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blkHeight)
	})
	return err

}

func (b blockRewardPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockRewardPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockRewardPlugin{}
