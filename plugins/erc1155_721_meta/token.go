package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

type tokenPlugin struct {
}

func (b tokenPlugin) Name() string {
	return "erc1155_721_meta"
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&models.Erc1155721Meta{}); err != nil {
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
			for _, log := range receipt.Logs() {
				if log.Address == "" || len(log.Topics) < 2 {
					continue
				}
				data := hex.EncodeToString(log.Data)
				var topics string
				for _, t := range log.Topics {
					topics += hex.EncodeToString(t[:])
				}
				if isErc721(log.Address, topics, data) {
					if isHandled(log.Address) {
						continue
					}
					isSBT, err := isSBT(log.Address, erc721ABI)
					if err != nil {
						return err
					}
					model := models.Erc1155721Meta{
						ContractAddress: log.Address,
						ErcType:         721,
						IsSBT:           isSBT,
					}
					if err := tx.Create(&model).Error; err != nil {
						return err
					}
					cachedContract[log.Address] = struct{}{}
				}
				if isErc1155(log.Address, topics, data) {
					if isHandled(log.Address) {
						continue
					}
					isSBT, err := isSBT(log.Address, erc1155ABI)
					if err != nil {
						return err
					}
					model := models.Erc1155721Meta{
						ContractAddress: log.Address,
						ErcType:         1155,
						IsSBT:           isSBT,
					}
					if err := tx.Create(&model).Error; err != nil {
						return err
					}
					cachedContract[log.Address] = struct{}{}
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
