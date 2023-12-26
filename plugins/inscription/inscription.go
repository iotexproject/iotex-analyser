package main

import (
	"context"
	"encoding/hex"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	slog "github.com/iotexproject/iotex-core/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
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
	startHeight := uint64(0)
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		cfg := &Config{}
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			slog.L().Error("failed to unmarshal plugin config", zap.Error(err), zap.String("config", string(cfgData)), zap.String("plugin", b.Name()))
		} else {
			startHeight = cfg.StartHeight
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	if height < startHeight && startHeight > 0 {
		return db.UpdateIndexHeight(b.Name(), startHeight-1)
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
		text, err := bytesToUTF8(data)
		if err != nil {
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
			RawData:     text,
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

func bytesToUTF8(data []byte) (string, error) {
	if !utf8.Valid(data) {
		return "", errors.New("not a utf8 string")
	}
	res := string(data)
	// remove null bytes
	res = strings.Replace(res, "\x00", "", -1)
	return res, nil
}
