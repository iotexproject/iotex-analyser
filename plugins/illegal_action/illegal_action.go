package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.2.0"

type illegalActionPlugin struct {
}

func (b illegalActionPlugin) Name() string {
	return "illegal_action"
}

func (b illegalActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b illegalActionPlugin) Start(ctx context.Context) error {
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

func (b illegalActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, receipt := range blk.Receipts {
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			//transaction
			for _, transation := range receipt.TransactionLogs() {
				if checkRecipient(transation.Recipient) {
					continue
				}

				m := models.IllegalAction{
					ActionHash:  actionHash,
					BlockHeight: blk.Height(),
					Sender:      transation.Sender,
					Recipient:   transation.Recipient,
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

func (b illegalActionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b illegalActionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = illegalActionPlugin{}
