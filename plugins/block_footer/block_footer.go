package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

type blockFooterPlugin struct {
}

func (b blockFooterPlugin) Name() string {
	return "block_footer"
}

func (b blockFooterPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockFooterPlugin) Start(ctx context.Context) error {

	if err := db.AutoMigrate(b.Name(), &models.BlockFooter{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b blockFooterPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	var endorser []string
	for _, endor := range blk.Footer.Endorsements() {
		pubKey := endor.Endorser()
		endorser = append(endorser, pubKey.Address().String())
	}
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		bm := models.BlockFooter{
			BlockHeight: blkHeight,
			Endorser:    endorser,
		}

		if err := tx.Create(&bm).Error; err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b blockFooterPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockFooterPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockFooterPlugin{}
