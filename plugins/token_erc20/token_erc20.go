package main

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.1.1"

type tokenPlugin struct {
}

func (b tokenPlugin) Name() string {
	return "token_erc20"
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.TokenErc20{}, &models.TokenErc20Holder{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {

		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			for _, log := range receipt.Logs() {
				data := hex.EncodeToString(log.Data)
				var topics string
				for _, t := range log.Topics {
					topics += hex.EncodeToString(t[:])
				}
				if !isErc20(log.Address, topics, data) {
					continue
				}

				from, to, amount, err := ParseContractData(topics, data)
				if err != nil {
					return errors.Wrap(err, "failed to parse contract data")
				}
				amountDec := decimal.NewFromBigInt(amount, 0)
				m := &models.TokenErc20{
					BlockHeight:     blk.Height(),
					ActionHash:      actionHash,
					ContractAddress: log.Address,
					Amount:          amountDec,
					Sender:          from,
					Recipient:       to,
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				}
				if err := tx.Create(m).Error; err != nil {
					return errors.Wrap(err, "failed to insert table data")
				}
				for _, addr := range []string{from, to} {
					if addr == "" {
						continue
					}
					h := &models.TokenErc20Holder{
						ContractAddress: log.Address,
						Holder:          addr,
					}
					if err := tx.Where("contract_address = ? and holder= ?", log.Address, addr).First(h).Error; err != nil {
						if err != gorm.ErrRecordNotFound {
							return err
						}
						if err := tx.Create(h).Error; err != nil {
							return err
						}
					}
				}

			}
		}

		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b tokenPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = tokenPlugin{}
