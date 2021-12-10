package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.2"

type probationPlugin struct {
}

func (b probationPlugin) Name() string {
	return "probation"
}

func (b probationPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b probationPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.Probation{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b probationPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	epochNum := kernel.GetEpochNum(blkHeight)
	epochHeight := kernel.GetEpochHeight(epochNum)
	chainClient := kernel.ChainClient()
	err := db.DB().Transaction(func(tx *gorm.DB) error {

		if blkHeight == epochHeight {
			probationList, err := fetchProbationList(chainClient, epochNum)
			if err != nil {
				return errors.Wrapf(err, "failed to get probation list from chain service in epoch %d", epochNum)
			}
			for _, k := range probationList.ProbationList {
				m := models.Probation{
					BlockHeight:   blkHeight,
					EpochNumber:   epochNum,
					Address:       k.GetAddress(),
					IntensityRate: probationList.IntensityRate,
					Count:         uint64(k.GetCount()),
				}
				if err := tx.Create(&m).Error; err != nil {
					return err
				}
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b probationPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b probationPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = probationPlugin{}
