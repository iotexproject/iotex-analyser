package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.1.0"

type tokenPlugin struct {
	tipHeight uint64
	metas     []*Erc1155721Meta
}

func (b *tokenPlugin) Name() string {
	return "erc1155_721_meta_" + VERSION
}

func (b *tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *tokenPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&Erc1155721Meta{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b *tokenPlugin) putBlock(ctx context.Context, blk *block.Block) error {
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
				b.metas = append(b.metas, &Erc1155721Meta{
					ContractAddress: log.Address,
					ErcType:         721,
					IsSBT:           isSBT,
				})
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
				b.metas = append(b.metas, &Erc1155721Meta{
					ContractAddress: log.Address,
					ErcType:         1155,
					IsSBT:           isSBT,
				})
				cachedContract[log.Address] = struct{}{}
			}
		}
	}
	return nil
}

func (b *tokenPlugin) commit() error {
	metas := b.metas
	b.metas = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(metas) > 0 {
			if err := tx.CreateInBatches(metas, 200).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *tokenPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *tokenPlugin) BatchSize() int {
	return 1000
}

func (b *tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *tokenPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = &tokenPlugin{}
