package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const VERSION = "2.2.1"

type candidateListPlugin struct {
	tipHeight uint64
	entries   []models.CandidateList
}

func (b *candidateListPlugin) Name() string {
	return "candidatelist"
}

func (b *candidateListPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *candidateListPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.CandidateList{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b *candidateListPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()
	if blkHeight >= kernel.FairbankEffectiveHeight() && blkHeight == epochHeight {
		var count int64
		if err := db.DB().Model(&models.CandidateList{}).Where("epoch_number=?", epochNum).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		candidateList, err := GetAllStakingCandidates(chainClient, blkHeight)
		if err != nil {
			return errors.Wrapf(err, "failed to get staking bucketlist list from chain service in epoch %d", epochNum)
		}
		candidateListData, err := proto.Marshal(candidateList)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal vote bucket list in epoch %d", epochNum)
		}
		b.entries = append(b.entries, models.CandidateList{
			EpochNumber:   epochNum,
			CandidateList: candidateListData,
		})
	}
	return nil
}

func (b *candidateListPlugin) commit() error {
	entries := b.entries
	b.entries = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		for i := range entries {
			if err := tx.Create(&entries[i]).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *candidateListPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *candidateListPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *candidateListPlugin) BatchSize() int {
	return 1000
}

func (b *candidateListPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *candidateListPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = &candidateListPlugin{}
