package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

type blockSupplyPlugin struct {
}

func (b blockSupplyPlugin) Name() string {
	return "block_supply"
}

func (b blockSupplyPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockSupplyPlugin) DependentPlugins() []string {
	return []string{"account_income"}
}

func (b blockSupplyPlugin) Start(ctx context.Context) error {

	if err := db.AutoMigrate(b.Name(), &models.BlockSupply{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b blockSupplyPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()

	err := db.DB().Transaction(func(tx *gorm.DB) error {
		totalSupply, err := getTotalSupply(blkHeight)
		if err != nil {
			return err
		}
		totalCirculatingSupply, err := getTotalCirculatingSupply(blkHeight, totalSupply)
		if err != nil {
			return err
		}
		bm := models.BlockSupply{
			BlockHeight:            blkHeight,
			TotalSupply:            totalSupply,
			TotalCirculatingSupply: totalCirculatingSupply,
		}

		if err := tx.Create(&bm).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blkHeight)
	})
	return err
}

func (b blockSupplyPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockSupplyPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockSupplyPlugin{}
