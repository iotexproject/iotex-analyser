package main

import (
	"context"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.2"

type blockPlugin struct {
	tipHeight uint64
	blocks    []*models.Block
}

func (b *blockPlugin) Name() string {
	return "block"
}

func (b *blockPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *blockPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.Block{}); err != nil {
		return errors.Wrap(err, "failed to start block plugin")
	}

	return nil
}

func (b *blockPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	blkHash := blk.HashBlock()

	year, err := strconv.Atoi(blk.Timestamp().Format("2006"))
	if err != nil {
		return err
	}
	month, err := strconv.Atoi(blk.Timestamp().Format("01"))
	if err != nil {
		return err
	}
	day, err := strconv.Atoi(blk.Timestamp().Format("02"))
	if err != nil {
		return err
	}
	b.blocks = append(b.blocks, &models.Block{
		BlockHeight:     blk.Height(),
		BlockHash:       hex.EncodeToString(blkHash[:]),
		ProducerAddress: blk.ProducerAddress(),
		NumActions:      len(blk.Actions),
		Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
		Year:            year,
		Month:           month,
		Day:             day,
	})
	return nil
}

func (b *blockPlugin) commit() error {
	blocks := b.blocks
	b.blocks = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(blocks) > 0 {
			if err := tx.Model(&models.Block{}).CreateInBatches(blocks, 1000).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *blockPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *blockPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *blockPlugin) BatchSize() int {
	return 1000
}

func (b *blockPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *blockPlugin) CatchUpSafe() bool { return true }

func (b *blockPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockPlugin{}
