package main

import (
	"context"
	"slices"
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
	successStatus            = uint64(1)
	erc20MetaBatchSize       = 256
	erc20MetaQueryChunkSize  = 500
	erc20MetaInsertBatchSize = 200
)

var (
	processed = sync.Map{}
)

type erc20MetaPlugin struct {
}

type contractAddressRow struct {
	ContractAddress string
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

func (b erc20MetaPlugin) BatchSize() int {
	return erc20MetaBatchSize
}

func (b erc20MetaPlugin) collectContracts(blks []*block.Block) []string {
	contracts := make(map[string]struct{})
	for _, blk := range blks {
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
				contracts[log.Address] = struct{}{}
			}
		}
	}
	addresses := make([]string, 0, len(contracts))
	for contract := range contracts {
		addresses = append(addresses, contract)
	}
	slices.Sort(addresses)
	return addresses
}

func (b erc20MetaPlugin) loadExistingContracts(table string, contracts []string) (map[string]struct{}, error) {
	existing := make(map[string]struct{})
	for _, batch := range chunkContracts(contracts, erc20MetaQueryChunkSize) {
		rows := make([]contractAddressRow, 0, len(batch))
		if err := db.DB().Table(table).Select("contract_address").Where("contract_address IN ?", batch).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			existing[row.ContractAddress] = struct{}{}
		}
	}
	return existing, nil
}

func (b erc20MetaPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return b.PutBlocks(ctx, []*block.Block{blk})
}

func (b erc20MetaPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	if len(blks) == 0 {
		return nil
	}

	contracts := b.collectContracts(blks)
	if len(contracts) == 0 {
		return db.UpdateIndexHeight(b.Name(), blks[len(blks)-1].Height())
	}

	holderContracts, err := b.loadExistingContracts("erc20_holders_v2", contracts)
	if err != nil {
		return errors.New("failed to query erc20 holders")
	}
	existingMetas, err := b.loadExistingContracts("erc20_metas", contracts)
	if err != nil {
		return errors.Wrap(err, "failed to query erc20 metas")
	}

	pendingContracts := make([]string, 0, len(contracts))
	for _, contract := range contracts {
		if _, ok := existingMetas[contract]; ok {
			processed.Store(contract, struct{}{})
			continue
		}
		if _, ok := holderContracts[contract]; !ok {
			processed.Store(contract, struct{}{})
			continue
		}
		pendingContracts = append(pendingContracts, contract)
	}

	cli := kernel.ChainClient()
	metas := make([]*models.Erc20Meta, 0, len(pendingContracts))
	for _, contract := range pendingContracts {
		name, _ := ReadERC20Name(cli, contract)
		symbol, _ := ReadERC20Symbol(cli, contract)
		decimals, _ := ReadERC20Decimals(cli, contract)
		metas = append(metas, &models.Erc20Meta{
			ContractAddress: contract,
			Name:            name,
			Symbol:          symbol,
			Decimals:        decimals,
		})
	}

	tipHeight := blks[len(blks)-1].Height()
	err = db.DB().Transaction(func(tx *gorm.DB) error {
		if len(metas) > 0 {
			if err := tx.CreateInBatches(metas, erc20MetaInsertBatchSize).Error; err != nil {
				return errors.Wrap(err, "failed to create erc20 metas")
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
	if err != nil {
		return err
	}

	for _, contract := range pendingContracts {
		processed.Store(contract, struct{}{})
	}
	return nil
}

func (b erc20MetaPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b erc20MetaPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = erc20MetaPlugin{}

func chunkContracts(items []string, chunkSize int) [][]string {
	if chunkSize <= 0 || len(items) == 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(items)+chunkSize-1)/chunkSize)
	for start := 0; start < len(items); start += chunkSize {
		end := start + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
}
