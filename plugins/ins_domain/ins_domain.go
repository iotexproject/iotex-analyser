package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	VERSION = "2.0.0"
)

type insDomainPlugin struct {
}

func (b insDomainPlugin) Name() string {
	return "ins_domain"
}

func (b insDomainPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b insDomainPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}

	return nil
}

func (b insDomainPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, receipt := range blk.Receipts {
			if receipt.Status != uint64(1) {
				continue
			}
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			for _, log := range receipt.Logs() {
				if log.Address == "" || len(log.Topics) < 2 {
					continue
				}
				if !isInsDomainContract(log.Address) {
					//slog.L().Info("not ins domain contract", zap.String("address", log.Address))
					continue
				}
				switch log.Address {
				case InsDomainContractAddress["INSRegistry"]:
					if err := handleInsRegistry(ctx, tx, log, actionHash); err != nil {
						return errors.Wrap(err, "failed to handle ins registry")
					}
				case InsDomainContractAddress["Resolver"]:
					if err := handleResolver(ctx, tx, log, actionHash); err != nil {
						return errors.Wrap(err, "failed to handle resolver")
					}
				case InsDomainContractAddress["BaseRegistrar"]:
					if err := handleBaseRegistrar(ctx, tx, log, actionHash); err != nil {
						return errors.Wrap(err, "failed to handle iotx registrar controller")
					}
				case InsDomainContractAddress["NameWrapper"]:
					if err := handleNameWrapper(ctx, tx, log, actionHash); err != nil {
						return errors.Wrap(err, "failed to handle name wrapper")
					}
				}
			}
		}

		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b insDomainPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b insDomainPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = insDomainPlugin{}
