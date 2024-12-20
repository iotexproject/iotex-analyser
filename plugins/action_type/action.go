package main

import (
	"context"
	"encoding/hex"
	"encoding/json"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

type actionTypePlugin struct {
}

func (b actionTypePlugin) Name() string {
	return "action_type"
}

func (b actionTypePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b actionTypePlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(),
		&models.ActionType{}); err != nil {
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
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			startHeight = cfg.StartHeight
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	if height < startHeight {
		return db.UpdateIndexHeight(b.Name(), startHeight-1)
	}
	return nil
}

func (b actionTypePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	ats := make([]*models.ActionType, 0, len(blk.Actions))
	getReceipt := func(h hash.Hash256) *action.Receipt {
		for _, r := range blk.Receipts {
			if r.ActionHash == h {
				return r
			}
		}
		return nil
	}
	for _, act := range blk.Actions {
		// skip legacy tx
		if act.Envelope.TxType() == action.LegacyTxType {
			continue
		}
		h, err := act.Hash()
		if err != nil {
			return errors.Wrap(err, "failed to get hash")
		}
		at := &models.ActionType{
			BlockHeight: blk.Height(),
			Hash:        hex.EncodeToString(h[:]),
			Type:        uint(act.Envelope.TxType()),
		}
		switch act.Envelope.TxType() {
		case action.BlobTxType:
			receipt := getReceipt(h)
			if receipt == nil {
				slog.L().Warn("failed to get receipt for blob tx", zap.String("hash", at.Hash))
				continue
			}
			at.BlobGas = act.BlobGas()
			at.BlobFeeCap = decimal.NewFromBigInt(act.BlobGasFeeCap(), 0)
			blobHashes := []string{}
			for _, h := range act.BlobHashes() {
				blobHashes = append(blobHashes, hex.EncodeToString(h[:]))
			}
			blobHashesJ, err := json.Marshal(blobHashes)
			if err != nil {
				return errors.Wrap(err, "failed to json marshal blob hashes")
			}
			at.BlobHashes = blobHashesJ
			at.BlobGasPrice = decimal.NewFromBigInt(receipt.BlobGasPrice, 0)
			fallthrough
		case action.DynamicFeeTxType:
			at.GasFeeCap = decimal.NewFromBigInt(act.GasFeeCap(), 0)
			at.GasTipCap = decimal.NewFromBigInt(act.GasTipCap(), 0)
			fallthrough
		case action.AccessListTxType:
			if acl := act.AccessList(); len(acl) > 0 {
				aclBytes, err := json.Marshal(acl)
				if err != nil {
					return errors.Wrap(err, "failed to marshal access list")
				}
				at.AccessList = aclBytes
			}
		default:
			return errors.Errorf("unknown tx type %d", act.Envelope.TxType())
		}
		ats = append(ats, at)
	}

	return db.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.CreateInBatches(ats, 200).Error; err != nil {
			return errors.Wrap(err, "failed to insert action types")
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
}

func (b actionTypePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b actionTypePlugin) Version() string {
	return "0.0.1"
}

// exported
var Plugin = actionTypePlugin{}
