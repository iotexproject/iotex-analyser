package main

import (
	"context"
	"encoding/hex"
	"strconv"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.1"

type blockPlugin struct {
}

func (b blockPlugin) Name() string {
	return "block"
}

func (b blockPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockPlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&models.Block{}); err != nil {
		return errors.Wrap(err, "failed to start block plugin")
	}

	return nil
}

func (b blockPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
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
		m := &models.Block{
			BlockHeight:     blk.Height(),
			BlockHash:       hex.EncodeToString(blkHash[:]),
			ProducerAddress: blk.ProducerAddress(),
			NumActions:      len(blk.Actions),
			Timestamp:       blk.Timestamp().Unix(),
			Year:            year,
			Month:           month,
			Day:             day,
		}
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b blockPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockPlugin{}
