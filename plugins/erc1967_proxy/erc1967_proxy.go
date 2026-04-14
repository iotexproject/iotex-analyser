package main

import (
	"context"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.2"

var (
	UPGRADED        hash.Hash256
	erc1967ProxyABI abi.ABI
	successStatus   = uint64(1)
)

type erc1967Proxy struct {
	tipHeight uint64
	proxies   []models.Erc1967Proxy
}

func (b *erc1967Proxy) Name() string {
	return "erc1967_proxy"
}

func (b *erc1967Proxy) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *erc1967Proxy) DependentPlugins() []string {
	return []string{"account_meta"}
}

func (b *erc1967Proxy) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(), &models.Erc1967Proxy{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b *erc1967Proxy) putBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	for _, receipt := range blk.Receipts {
		if receipt.Status != successStatus {
			continue
		}
		rows, err := handleLogs(receipt.Logs(), hex.EncodeToString(receipt.ActionHash[:]), blkHeight)
		if err != nil {
			return err
		}
		b.proxies = append(b.proxies, rows...)
	}
	return nil
}

func (b *erc1967Proxy) commit() error {
	proxies := b.proxies
	b.proxies = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(proxies) > 0 {
			if err := tx.CreateInBatches(proxies, 200).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *erc1967Proxy) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *erc1967Proxy) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *erc1967Proxy) BatchSize() int {
	return 1000
}

func (b *erc1967Proxy) Stop(ctx context.Context) error {
	return nil
}

func (b *erc1967Proxy) Version() string {
	return VERSION
}

// exported
var Plugin = &erc1967Proxy{}
