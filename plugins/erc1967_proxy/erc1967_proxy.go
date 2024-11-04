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
}

func (b erc1967Proxy) Name() string {
	return "erc1967_proxy"
}

func (b erc1967Proxy) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b erc1967Proxy) DependentPlugins() []string {
	return []string{"account_meta"}
}

func (b erc1967Proxy) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(), &models.Erc1967Proxy{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b erc1967Proxy) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		blkHeight := blk.Height()
		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			if err := handleLogs(receipt.Logs(), hex.EncodeToString(receipt.ActionHash[:]), blkHeight, tx); err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b erc1967Proxy) Stop(ctx context.Context) error {
	return nil
}

func (b erc1967Proxy) Version() string {
	return VERSION
}

// exported
var Plugin = erc1967Proxy{}
