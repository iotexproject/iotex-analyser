package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.2.0"

type illegalActionPlugin struct {
	tipHeight uint64
	actions   []*models.IllegalAction
}

func (b *illegalActionPlugin) Name() string {
	return "illegal_action"
}

func (b *illegalActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *illegalActionPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.IllegalAction{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func checkRecipient(recipient string) bool {
	if len(recipient) == 0 {
		return true
	}
	if _, err := address.FromString(recipient); err != nil {
		return false
	}
	return true
}

func (b *illegalActionPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	for _, receipt := range blk.Receipts {
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for _, transation := range receipt.TransactionLogs() {
			if checkRecipient(transation.Recipient) {
				continue
			}
			b.actions = append(b.actions, &models.IllegalAction{
				ActionHash:  actionHash,
				BlockHeight: blk.Height(),
				Sender:      transation.Sender,
				Recipient:   transation.Recipient,
			})
		}
	}
	return nil
}

func (b *illegalActionPlugin) commit() error {
	actions := b.actions
	b.actions = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(actions) > 0 {
			if err := tx.CreateInBatches(actions, 200).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *illegalActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *illegalActionPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *illegalActionPlugin) BatchSize() int {
	return 1000
}

func (b *illegalActionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *illegalActionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = &illegalActionPlugin{}
