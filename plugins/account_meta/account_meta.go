package main

import (
	"context"
	"slices"
	"sync"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const VERSION = "2.1.2"

const (
	accountMetaBatchSize       = 256
	accountMetaQueryChunkSize  = 500
	accountMetaInsertBatchSize = 200
)

type accountMetaPlugin struct {
	cachedAccounts sync.Map
}

type accountHeight struct {
	Address string
}

func (b *accountMetaPlugin) Name() string {
	return "account_meta"
}

func (b *accountMetaPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *accountMetaPlugin) Start(ctx context.Context) error {
	b.cachedAccounts = sync.Map{}
	if err := db.AutoMigrate(b.Name(), &models.AccountMeta{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func getAccounts(blk *block.Block) ([]string, error) {
	seen := make(map[string]struct{})
	accounts := make([]string, 0)
	appendUnique := func(account string) {
		if account == "" {
			return
		}
		if _, ok := seen[account]; ok {
			return
		}
		seen[account] = struct{}{}
		accounts = append(accounts, account)
	}
	receipts := blk.Receipts
	for _, receipt := range receipts {
		appendUnique(receipt.ContractAddress)
		//transaction
		for _, transation := range receipt.TransactionLogs() {
			appendUnique(transation.Sender)
			recipient := transation.Recipient
			if len(recipient) > 0 {
				if addr, err := address.FromString(recipient); err != nil {
					//skip invalid address
					continue
				} else {
					recipient = addr.String()
				}
			}
			appendUnique(recipient)
		}
		for _, log := range receipt.Logs() {
			appendUnique(log.Address)
		}
	}
	return accounts, nil
}

func (b *accountMetaPlugin) BatchSize() int {
	return accountMetaBatchSize
}

func (b *accountMetaPlugin) loadExistingAccounts(accounts []string) (map[string]struct{}, error) {
	existing := make(map[string]struct{})
	for _, batch := range chunkStrings(accounts, accountMetaQueryChunkSize) {
		rows := make([]accountHeight, 0, len(batch))
		if err := db.DB().Model(&models.AccountMeta{}).Select("address").Where("address IN ?", batch).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			existing[row.Address] = struct{}{}
		}
	}
	return existing, nil
}

func (b *accountMetaPlugin) collectProcessAccounts(blks []*block.Block) (map[string]uint64, error) {
	processAccounts := make(map[string]uint64)
	for _, blk := range blks {
		accounts, err := getAccounts(blk)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get accounts from block %d", blk.Height())
		}
		for _, account := range accounts {
			if _, ok := b.cachedAccounts.Load(account); ok {
				continue
			}
			if _, ok := processAccounts[account]; ok {
				continue
			}
			processAccounts[account] = blk.Height()
		}
	}
	return processAccounts, nil
}

func (b *accountMetaPlugin) loadAccountMetas(accounts map[string]uint64) ([]*models.AccountMeta, []string) {
	if len(accounts) == 0 {
		return nil, nil
	}
	addresses := make([]string, 0, len(accounts))
	for account := range accounts {
		addresses = append(addresses, account)
	}
	slices.Sort(addresses)

	metas := make([]*models.AccountMeta, 0, len(addresses))
	loadedAccounts := make([]string, 0, len(addresses))
	for _, account := range addresses {
		meta, err := accountMeta(account)
		if err != nil {
			continue
		}
		metas = append(metas, &models.AccountMeta{
			BlockHeight:      accounts[account],
			Address:          account,
			IsContract:       meta.IsContract,
			ContractByteCode: meta.ContractByteCode,
		})
		loadedAccounts = append(loadedAccounts, account)
	}
	return metas, loadedAccounts
}

func (b *accountMetaPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return b.PutBlocks(ctx, []*block.Block{blk})
}

func (b *accountMetaPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	if len(blks) == 0 {
		return nil
	}

	processAccounts, err := b.collectProcessAccounts(blks)
	if err != nil {
		return err
	}

	if len(processAccounts) > 0 {
		addresses := make([]string, 0, len(processAccounts))
		for account := range processAccounts {
			addresses = append(addresses, account)
		}
		existing, err := b.loadExistingAccounts(addresses)
		if err != nil {
			return err
		}
		for account := range existing {
			b.cachedAccounts.Store(account, true)
			delete(processAccounts, account)
		}
	}

	metas, loadedAccounts := b.loadAccountMetas(processAccounts)
	tipHeight := blks[len(blks)-1].Height()
	err = db.DB().Transaction(func(tx *gorm.DB) error {
		if len(metas) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "address"}},
				DoNothing: true,
			}).CreateInBatches(metas, accountMetaInsertBatchSize).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
	if err != nil {
		return err
	}

	for _, account := range loadedAccounts {
		b.cachedAccounts.Store(account, true)
	}
	return nil
}

func (b *accountMetaPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *accountMetaPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = &accountMetaPlugin{}
