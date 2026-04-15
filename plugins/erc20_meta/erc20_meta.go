package main

import (
	"context"
	"sync"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "1.0.0"

const (
	successStatus = uint64(1)
)

var (
	processed = sync.Map{}
)

type erc20MetaPlugin struct {
}

func (b erc20MetaPlugin) Name() string {
	return "erc20_meta"
}

func (b erc20MetaPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b erc20MetaPlugin) DependentPlugins() []string {
	return []string{"erc20"}
}

func (b erc20MetaPlugin) Start(ctx context.Context) error {
	if err := initErc20(); err != nil {
		return errors.Wrap(err, "cannot init erc20")
	}
	if err := db.AutoMigrate(b.Name(),
		&models.Erc20Meta{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	var metas []*models.Erc20Meta
	if err := db.DB().Model(&models.Erc20Meta{}).Select("contract_address").Find(&metas).Error; err != nil {
		return errors.Wrap(err, "failed to get erc20 metas")
	}
	for _, meta := range metas {
		processed.Store(meta.ContractAddress, struct{}{})
	}
	return nil
}

func (b erc20MetaPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	cli := kernel.ChainClient()
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			for _, log := range receipt.Logs() {
				if log.Address == "" || len(log.Topics) < 2 {
					continue
				}
				if _, ok := processed.Load(log.Address); ok {
					continue
				}
				var count int64
				err := tx.Table("erc20_holders_v2").Where("contract_address = ?", log.Address).Count(&count).Error
				if err != nil {
					return errors.New("failed to count erc20 holder")
				}
				if count == 0 {
					processed.Store(log.Address, struct{}{})
					continue
				}
				name, _ := ReadERC20Name(cli, log.Address)
				symbol, _ := ReadERC20Symbol(cli, log.Address)
				decimals, _ := ReadERC20Decimals(cli, log.Address)
				meta := &models.Erc20Meta{
					ContractAddress: log.Address,
					Name:            name,
					Symbol:          symbol,
					Decimals:        decimals,
				}
				if err := tx.Create(meta).Error; err != nil {
					return errors.Wrap(err, "failed to create erc20 meta")
				}
				processed.Store(log.Address, struct{}{})
			}
		}

		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b erc20MetaPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b erc20MetaPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = erc20MetaPlugin{}
