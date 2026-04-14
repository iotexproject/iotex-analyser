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
	tipHeight uint64
	supplies  []*models.BlockSupply
}

func (b *blockSupplyPlugin) Name() string {
	return "block_supply"
}

func (b *blockSupplyPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *blockSupplyPlugin) DependentPlugins() []string {
	return []string{"account_income"}
}

func (b *blockSupplyPlugin) Start(ctx context.Context) error {

	if err := db.AutoMigrate(b.Name(), &models.BlockSupply{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b *blockSupplyPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()

	totalSupply, err := getTotalSupply(blkHeight)
	if err != nil {
		return err
	}
	totalCirculatingSupply, err := getTotalCirculatingSupply(blkHeight, totalSupply)
	if err != nil {
		return err
	}
	b.supplies = append(b.supplies, &models.BlockSupply{
		BlockHeight:            blkHeight,
		TotalSupply:            totalSupply,
		TotalCirculatingSupply: totalCirculatingSupply,
	})
	return nil
}

func (b *blockSupplyPlugin) commit() error {
	supplies := b.supplies
	b.supplies = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(supplies) > 0 {
			if err := tx.Model(&models.BlockSupply{}).CreateInBatches(supplies, 1000).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *blockSupplyPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *blockSupplyPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *blockSupplyPlugin) BatchSize() int {
	return 1000
}

func (b *blockSupplyPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *blockSupplyPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = &blockSupplyPlugin{}
