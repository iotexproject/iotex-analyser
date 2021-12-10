package main

import (
	"context"
	"sync"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.1"

type accountMetaPlugin struct {
	cachedAccounts sync.Map
}

func (b accountMetaPlugin) Name() string {
	return "account_meta"
}

func (b accountMetaPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b accountMetaPlugin) Start(ctx context.Context) error {
	b.cachedAccounts = sync.Map{}
	if err := db.AutoMigrate(b.Name(), &AccountMeta{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b accountMetaPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	accounts := []string{}
	receipts := blk.Receipts
	for _, receipt := range receipts {
		if receipt.ContractAddress != "" {
			accounts = appendIfMissing(accounts, receipt.ContractAddress)
		}
		//transaction
		for _, transation := range receipt.TransactionLogs() {
			if transation.Sender != "" {
				accounts = appendIfMissing(accounts, transation.Sender)
			}
			if transation.Recipient != "" {
				accounts = appendIfMissing(accounts, transation.Recipient)
			}
		}
		for _, log := range receipt.Logs() {
			if log.Address != "" {
				accounts = appendIfMissing(accounts, log.Address)
			}
		}
	}

	processAccounts := []string{}
	for _, account := range accounts {
		_, ok := b.cachedAccounts.Load(account)
		if ok {
			continue
		}
		var count int64
		if err := db.DB().Model(&AccountMeta{}).Where("address=?", account).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			b.cachedAccounts.Store(account, true)
			continue
		}
		processAccounts = appendIfMissing(processAccounts, account)
	}

	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, account := range processAccounts {
			meta, err := accountMeta(account)
			if err == nil {
				am := &AccountMeta{
					BlockHeight:      blk.Height(),
					Address:          account,
					IsContract:       meta.IsContract,
					ContractByteCode: meta.ContractByteCode,
				}

				if err := tx.Create(am).Error; err != nil {
					return err
				}
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b accountMetaPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b accountMetaPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = accountMetaPlugin{}
