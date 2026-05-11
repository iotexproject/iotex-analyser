package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
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
	tipHeight uint64
	ats       []*models.ActionType
	auths     []*models.Authorization
}

func (b *actionTypePlugin) Name() string {
	return "action_type"
}

func (b *actionTypePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *actionTypePlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(),
		&models.ActionType{},
		&models.Authorization{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	// Authorization was added after action_type was first deployed; on existing
	// installs AutoMigrate is a no-op (height > 0) so create it explicitly if
	// missing.
	if err := db.EnsureTables(&models.Authorization{}); err != nil {
		return errors.Wrapf(err, "failed to ensure authorization table for plugin %s", b.Name())
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

func (b *actionTypePlugin) putBlock(ctx context.Context, blk *block.Block) error {
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
			for _, h := range act.BlobHashes() {
				at.BlobHashes = append(at.BlobHashes, hex.EncodeToString(h[:]))
			}
			at.BlobGasPrice = decimal.NewFromBigInt(receipt.BlobGasPrice, 0)
			fallthrough
		case action.SetCodeTxType:
			if authList := act.Envelope.SetCodeAuthorizations(); len(authList) > 0 {
				authBytes, err := json.Marshal(authList)
				if err != nil {
					return errors.Wrap(err, "failed to marshal auth list")
				}
				at.AuthList = authBytes
				for i, auth := range authList {
					authRow, err := authorizationRow(ctx, at.Hash, blk.Height(), i, auth)
					if err != nil {
						return errors.Wrapf(err, "failed to build authorization row for %s[%d]", at.Hash, i)
					}
					b.auths = append(b.auths, authRow)
				}
			}
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
		b.ats = append(b.ats, at)
	}
	return nil
}

func (b *actionTypePlugin) commit() error {
	ats := b.ats
	b.ats = nil
	auths := b.auths
	b.auths = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.CreateInBatches(ats, 200).Error; err != nil {
			return errors.Wrap(err, "failed to insert action types")
		}
		if len(auths) > 0 {
			if err := tx.CreateInBatches(auths, 200).Error; err != nil {
				return errors.Wrap(err, "failed to insert authorizations")
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *actionTypePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *actionTypePlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *actionTypePlugin) BatchSize() int {
	return 1000
}

func (b *actionTypePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *actionTypePlugin) Version() string {
	return "0.0.1"
}

// authorizationRow builds an Authorization model from a SetCodeAuthorization
// and populates its Valid field via kernel.ComputeAuthorizationValidity, which
// queries an eth archive RPC endpoint.
func authorizationRow(ctx context.Context, actionHash string, blockHeight uint64, index int, auth ethtypes.SetCodeAuthorization) (*models.Authorization, error) {
	row := &models.Authorization{
		ActionHash:  actionHash,
		BlockHeight: blockHeight,
		Index:       index,
		ChainID:     auth.ChainID.Hex(),
		Address:     strings.ToLower(auth.Address.Hex()),
		Nonce:       fmt.Sprintf("0x%x", auth.Nonce),
		YParity:     fmt.Sprintf("0x%x", auth.V),
		R:           auth.R.Hex(),
		S:           auth.S.Hex(),
	}
	authority, recoverErr := auth.Authority()
	if recoverErr != nil {
		// Signature did not recover a valid authority → invalid auth.
		invalid := false
		row.Valid = &invalid
		return row, nil
	}
	row.Authority = strings.ToLower(authority.Hex())

	valid, err := kernel.ComputeAuthorizationValidity(ctx, authority, blockHeight, &auth.ChainID, auth.Nonce)
	if err != nil {
		return nil, err
	}
	row.Valid = &valid
	return row, nil
}

// exported
var Plugin = actionTypePlugin{}
