package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const VERSION = "2.0.2"

type blockPlugin struct {
	batchSize int
}

func (b blockPlugin) Name() string {
	return "block"
}

func (b blockPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockPlugin) BatchSize() int {
	return b.batchSize
}

func (b *blockPlugin) Start(ctx context.Context) error {
	var err error
	cfg := &Config{}
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	if err := b.migrateTable(ctx); err != nil {
		return errors.Wrap(err, "failed to migrate table")
	}

	b.batchSize = cfg.BatchSize
	return nil
}

func (b blockPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	total := []*Block{}
	for _, blk := range blks {
		res, err := b.putBlock(ctx, blk)
		if err != nil {
			return err
		}
		total = append(total, res)
	}
	return b.commit(blks[0].Height()+uint64(len(blks))-1, total)
}

func (b blockPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	res, err := b.putBlock(ctx, blk)
	if err != nil {
		return err
	}
	return b.commit(blk.Height(), []*Block{res})
}

func (b blockPlugin) putBlock(ctx context.Context, blk *block.Block) (*Block, error) {
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
	return &Block{
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

func (b blockPlugin) commit(height uint64, bs []*Block) error {
	if len(bs) > 0 {
		batch, err := db.ChConn().PrepareBatch(context.Background(), fmt.Sprintf("INSERT INTO %s", Block{}.TableName()))
		if err != nil {
			return errors.Wrap(err, "failed to prepare batch")
		}
		for _, e := range bs {
			if err := batch.AppendStruct(e); err != nil {
				return errors.Wrap(err, "failed to append struct")
			}
		}
		if err := batch.Send(); err != nil {
			return errors.Wrap(err, "failed to insert table")
		}
	}
	err := db.UpdateIndexHeight(b.Name(), height)
	return errors.Wrap(err, "failed to update index height")
}

func (b blockPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockPlugin{}
