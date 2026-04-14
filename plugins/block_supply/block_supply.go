package main

import (
	"context"
	"math/big"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.1.0"

type blockSupplyPlugin struct {
	tipHeight uint64
	supplies  []*models.BlockSupply
	// runningBalance tracks cumulative net balance (in_flow - out_flow) for each
	// tracked address in memory, so putBlock only needs a cheap per-block point query.
	runningBalance map[string]*big.Int
}

func (b *blockSupplyPlugin) Name() string {
	return "block_supply"
}

func (b *blockSupplyPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *blockSupplyPlugin) DependentPlugins() []string {
	return []string{"account_income"}
}

func (b *blockSupplyPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.BlockSupply{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	// Load cumulative balances once at current tip height to seed runningBalance.
	// putBlock will then only issue cheap single-block delta queries per block.
	tipHeight, err := db.GetIndexHeight(b.Name())
	if err != nil {
		return errors.Wrap(err, "failed to get index height for block_supply")
	}
	trackedAddrs := []string{address.ZeroAddress, lockAddresses}
	b.runningBalance = make(map[string]*big.Int, len(trackedAddrs))
	for _, addr := range trackedAddrs {
		bal, err := accountBalanceByHeight(tipHeight, addr)
		if err != nil {
			return errors.Wrapf(err, "failed to init running balance for %s", addr)
		}
		b.runningBalance[addr] = bal
	}
	return nil
}

func (b *blockSupplyPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()

	// Apply per-block delta for each tracked address (O(1) point query).
	for _, addr := range []string{address.ZeroAddress, lockAddresses} {
		delta, err := accountBalanceDeltaAtHeight(blkHeight, addr)
		if err != nil {
			return errors.Wrapf(err, "failed to get balance delta at height %d for %s", blkHeight, addr)
		}
		b.runningBalance[addr].Add(b.runningBalance[addr], delta)
	}

	totalSupply := computeTotalSupply(b.runningBalance[address.ZeroAddress])
	totalCirculatingSupply, err := computeTotalCirculatingSupply(totalSupply, b.runningBalance[lockAddresses])
	if err != nil {
		return err
	}
	b.supplies = append(b.supplies, &models.BlockSupply{
		BlockHeight:            blkHeight,
		TotalSupply:            totalSupply,
		TotalCirculatingSupply: totalCirculatingSupply,
	})
	return nil
}

func (b *blockSupplyPlugin) commit() error {
	supplies := b.supplies
	b.supplies = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(supplies) > 0 {
			if err := tx.Model(&models.BlockSupply{}).CreateInBatches(supplies, 1000).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *blockSupplyPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *blockSupplyPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *blockSupplyPlugin) BatchSize() int {
	return 1000
}

func (b *blockSupplyPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *blockSupplyPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockSupplyPlugin{}
