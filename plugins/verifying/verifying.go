package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
)

const VERSION = "2.0.0"

type verifyingPlugin struct {
}

func (b verifyingPlugin) Name() string {
	return "verifying"
}

func (b verifyingPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b verifyingPlugin) DependentPlugins() []string {
	return []string{"block", "block_action", "block_receipts"}
}

func (b verifyingPlugin) Start(ctx context.Context) error {
	return nil
}

func (b verifyingPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	db2 := db.DB()
	if err := verifyAction(blk, db2, blkHeight); err != nil {
		return err
	}
	if err := verifyReceipt(blk, db2, blkHeight); err != nil {
		return err
	}
	if err := verifyTransactions(blk, db2, blkHeight); err != nil {
		return err
	}
	return db.UpdateIndexHeight(b.Name(), blkHeight)
}

func (b verifyingPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b verifyingPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = verifyingPlugin{}
