package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.1.0"

type probationPlugin struct {
	tipHeight  uint64
	probations []models.Probation
}

func (b *probationPlugin) Name() string {
	return "probation"
}

func (b *probationPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *probationPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.Probation{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b *probationPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	if blkHeight == epochHeight {
		chainClient := kernel.ChainClient()
		probationList, err := fetchProbationList(chainClient, epochNum)
		if err != nil {
			return errors.Wrapf(err, "failed to get probation list from chain service in epoch %d", epochNum)
		}
		for _, k := range probationList.ProbationList {
			b.probations = append(b.probations, models.Probation{
				BlockHeight:   blkHeight,
				EpochNumber:   epochNum,
				Address:       k.GetAddress(),
				IntensityRate: probationList.IntensityRate,
				Count:         uint64(k.GetCount()),
			})
		}
	}
	return nil
}

func (b *probationPlugin) commit() error {
	probations := b.probations
	b.probations = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(probations) > 0 {
			if err := tx.CreateInBatches(probations, 200).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *probationPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *probationPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *probationPlugin) BatchSize() int {
	return 1000
}

func (b *probationPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *probationPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = &probationPlugin{}
