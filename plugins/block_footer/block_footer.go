package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.1.0"

type blockFooterPlugin struct {
	tipHeight uint64
	footers   []*models.BlockFooter
}

func (b *blockFooterPlugin) Name() string {
	return "block_footer"
}

func (b *blockFooterPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *blockFooterPlugin) Start(ctx context.Context) error {

	if err := db.AutoMigrate(b.Name(), &models.BlockFooter{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b *blockFooterPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	for _, endor := range blk.Footer.Endorsements() {
		pubKey := endor.Endorser()
		endorser := pubKey.Address().String()
		b.footers = append(b.footers, &models.BlockFooter{
			BlockHeight: blkHeight,
			Endorser:    endorser,
		})
	}
	return nil
}

func (b *blockFooterPlugin) commit() error {
	footers := b.footers
	b.footers = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(footers) > 0 {
			if err := tx.Model(&models.BlockFooter{}).CreateInBatches(footers, 1000).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *blockFooterPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *blockFooterPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *blockFooterPlugin) BatchSize() int {
	return 1000
}

func (b *blockFooterPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *blockFooterPlugin) CatchUpSafe() bool { return true }

func (b *blockFooterPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockFooterPlugin{}
