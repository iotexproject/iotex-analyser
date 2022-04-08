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

const VERSION = "2.0.0"

type candidateListPlugin struct {
}

func (b candidateListPlugin) Name() string {
	return "candidatelist"
}

func (b candidateListPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b candidateListPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.CandidateList{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b candidateListPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()
	var candidateListData []byte
	var err error
	if blkHeight == epochHeight {
		candidateList, err := GetAllStakingCandidates(chainClient, epochHeight)
		if err != nil {
			return errors.Wrapf(err, "failed to get staking bucketlist list from chain service in epoch %d", epochNum)
		}
		candidateListData, err = proto.Marshal(candidateList)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal vote bucket list in epoch %d", epochNum)
		}
	} else {
		return db.UpdateIndexHeight(b.Name(), blkHeight)
	}
	err = db.DB().Transaction(func(tx *gorm.DB) error {
		m := models.CandidateList{
			EpochNumber:   epochNum,
			CandidateList: candidateListData,
		}
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blkHeight)
	})

	return err
}

func (b candidateListPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b candidateListPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = candidateListPlugin{}
