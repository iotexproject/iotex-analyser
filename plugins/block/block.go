package main

import (
	"context"
	"encoding/hex"
	"strconv"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
)

const VERSION = "2.0.0"

type blockPlugin struct {
}

func (b blockPlugin) Name() string {
	return "block"
}

func (b blockPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockPlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&Block{}); err != nil {
		return errors.Wrap(err, "failed to start block plugin")
	}

	return nil
}

func (b blockPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
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
	blkModel := &Block{
		BlockHeight:     blk.Height(),
		BlockHash:       hex.EncodeToString(blkHash[:]),
		ProducerAddress: blk.ProducerAddress(),
		NumActions:      len(blk.Actions),
		Timestamp:       blk.Timestamp().Unix(),
		Year:            year,
		Month:           month,
		Day:             day,
		BaseModel: db.BaseModel{
			IndexName:   b.Name(),
			IndexHeight: blk.Height(),
		},
	}

	return blkModel.Save()
}

func (b blockPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockPlugin{}
