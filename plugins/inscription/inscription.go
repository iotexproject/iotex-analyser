package main

import (
	"context"
	"encoding/hex"
	"time"
	"unicode/utf8"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	slog "github.com/iotexproject/iotex-core/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	batchSize = 200
)

type tokenPlugin struct {
}

func (b tokenPlugin) Name() string {
	return "inscription"
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(),
		&models.InscriptionRaw{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	height, err := db.GetIndexHeight(b.Name())
	if err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	if height == 0 {
		return db.UpdateIndexHeight(b.Name(), config.Default.Iotex.InscriptionStartHeight)
	}
	return nil
}

func (b tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return db.DB().Transaction(func(gormTx *gorm.DB) error {
		return b.putBlock(ctx, gormTx, blk)
	})
}

func (b tokenPlugin) putBlock(ctx context.Context, gormTx *gorm.DB, blk *block.Block) error {
	getActReceiptFun := func(blk *block.Block, actHash hash.Hash256) *action.Receipt {
		for _, receipt := range blk.Receipts {
			if receipt.ActionHash == actHash {
				return receipt
			}
		}
		return nil
	}
	inscripts := make([]*models.InscriptionRaw, 0)
	for _, act := range blk.Actions {
		actHash, err := act.Hash()
		if err != nil {
			continue
		}
		slog.L().Debug("handle action", zap.Any("hash", hex.EncodeToString(actHash[:])))
		receipt := getActReceiptFun(blk, actHash)
		if receipt == nil {
			slog.L().Debug("skip action: no receipt", zap.Any("hash", hex.EncodeToString(actHash[:])))
			continue
		}
		if receipt.Status != uint64(iotextypes.ReceiptStatus_Success) {
			slog.L().Debug("skip action: action failed", zap.Any("hash", hex.EncodeToString(actHash[:])))
			continue
		}
		tx, ok := act.Action().(action.EthCompatibleAction)
		if !ok {
			slog.L().Debug("skip action: not eth-compatible", zap.Any("hash", hex.EncodeToString(actHash[:])))
			continue
		}
		ethTx, err := tx.ToEthTx()
		if err != nil {
			slog.L().Debug("skip action: to eth error", zap.Any("hash", hex.EncodeToString(actHash[:])), zap.Error(err))
			continue
		}
		data := ethTx.Data()
		if len(data) == 0 {
			slog.L().Debug("skip action: empty data", zap.Any("hash", hex.EncodeToString(actHash[:])))
			continue
		}
		if !utf8.Valid(data) {
			slog.L().Debug("skip action: not a utf8 string", zap.Any("hash", hex.EncodeToString(actHash[:])))
			continue
		}
		fromAddr, _ := address.FromBytes(act.SenderAddress().Bytes())
		toAddr, _ := address.FromBytes(ethTx.To().Bytes())
		inscripts = append(inscripts, &models.InscriptionRaw{
			BlockHeight: blk.Height(),
			ActionHash:  hex.EncodeToString(actHash[:]),
			Sender:      fromAddr.String(),
			Recipient:   toAddr.String(),
			Timestamp:   time.Unix(blk.Timestamp().Unix(), 0),
			RawData:     string(data),
		})
	}
	if err := gormTx.CreateInBatches(inscripts, batchSize).Error; err != nil {
		return errors.Wrapf(err, "failed to put block %d", blk.Height())
	}
	return nil
}

func (b tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b tokenPlugin) Version() string {
	return "0.0.1"
}

// exported
var Plugin = tokenPlugin{}
