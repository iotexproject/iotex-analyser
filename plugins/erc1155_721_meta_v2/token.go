package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.1.0"

type tokenPlugin struct {
}

func (b tokenPlugin) Name() string {
	return "erc1155_721_meta_" + VERSION
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&Erc1155721Meta{}); err != nil {
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
				ok, err := isErc721(log.Address)
				if err != nil {
					return err
				}
				if ok {
					ok, err := isHandled(log.Address)
					if err != nil {
						return errors.Wrap(err, "failed to check isHandled")
					}
					if ok {
						continue
					}
					isSBT, err := kernel.IsSBT(log.Address)
					if err != nil {
						return err
					}
					model := Erc1155721Meta{
						ContractAddress: log.Address,
						ErcType:         721,
						IsSBT:           isSBT,
					}
					if err := tx.Create(&model).Error; err != nil {
						return err
					}
					cachedContract[log.Address] = struct{}{}
				}
				ok, err = isErc1155(log.Address)
				if err != nil {
					return err
				}
				if ok {
					ok, err := isHandled(log.Address)
					if err != nil {
						return errors.Wrap(err, "failed to check isHandled")
					}
					if ok {
						continue
					}
					isSBT, err := kernel.IsSBT(log.Address)
					if err != nil {
						return err
					}
					model := Erc1155721Meta{
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
