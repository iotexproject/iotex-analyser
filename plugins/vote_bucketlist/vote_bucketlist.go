package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const VERSION = "2.0.1"

type voteBucketListPlugin struct {
}

func (b voteBucketListPlugin) Name() string {
	return "vote_bucketlist"
}

func (b voteBucketListPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b voteBucketListPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.VoteBucketList{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b voteBucketListPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()
	preEpochNum := epochNum - 1
	preEpochHeight := kernel.GetEpochHeight(preEpochNum)
	var voteBucketListData []byte
	var err error
	if blkHeight >= kernel.FairbankEffectiveHeight() && blkHeight == epochHeight {
		voteBucketList, err := GetAllStakingBuckets(chainClient, preEpochHeight)
		if err != nil {
			return errors.Wrapf(err, "failed to get staking bucketlist list from chain service in epoch %d", epochNum)
		}
		voteBucketListData, err = proto.Marshal(voteBucketList) //nolint:errcheck
		if err != nil {
			return errors.Wrapf(err, "failed to marshal vote bucket list in epoch %d", epochNum)
		}
	} else {
		return db.UpdateIndexHeight(b.Name(), blkHeight)
	}
	err = db.DB().Transaction(func(tx *gorm.DB) error {
		m := models.VoteBucketList{
			EpochNumber: preEpochNum,
			BucketList:  voteBucketListData,
		}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blkHeight)
	})

	return err
}

func (b voteBucketListPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b voteBucketListPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = voteBucketListPlugin{}
