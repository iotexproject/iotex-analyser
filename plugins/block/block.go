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
}

func (b blockPlugin) Name() string {
	return "block"
}

func (b blockPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.Block{}); err != nil {
		return errors.Wrap(err, "failed to start block plugin")
	}

	return nil
}

func (b blockPlugin) BatchSize() int {
	return 1000
}

func (b blockPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	ms := make([]*models.Block, 0, len(blks))
	for _, blk := range blks {
		m, err := b.convBlock(blk)
		if err != nil {
			return err
		}
		ms = append(ms, m)
	}
	if len(ms) == 0 {
		return nil
	}
	tipHeight := blks[len(blks)-1].Height()

	return db.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(ms).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b blockPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return db.DB().Transaction(func(tx *gorm.DB) error {
		m, err := b.convBlock(blk)
		if err != nil {
			return err
		}
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
}

func (b blockPlugin) convBlock(blk *block.Block) (*models.Block, error) {
	blkHash := blk.HashBlock()
	year, err := strconv.Atoi(blk.Timestamp().Format("2006"))
	if err != nil {
		return nil, err
	}
	month, err := strconv.Atoi(blk.Timestamp().Format("01"))
	if err != nil {
		return nil, err
	}
	day, err := strconv.Atoi(blk.Timestamp().Format("02"))
	if err != nil {
		return nil, err
	}
	return &models.Block{
		BlockHeight:     blk.Height(),
		BlockHash:       hex.EncodeToString(blkHash[:]),
		ProducerAddress: blk.ProducerAddress(),
		NumActions:      len(blk.Actions),
		Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
		Year:            year,
		Month:           month,
		Day:             day,
	}, nil
}

func (b blockPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockPlugin{}
