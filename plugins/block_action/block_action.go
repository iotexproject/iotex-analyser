package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert/yaml"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const VERSION = "2.3.0"

var (
	processTimeMetric = kernel.ProcessTimeMetric
)

type blockActionPlugin struct {
	batchSize int
}

func (b blockActionPlugin) Name() string {
	return "block_action"
}

func (b blockActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *blockActionPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(),
		&models.BlockAction{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		cfg := &Config{}
		if err := yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			b.batchSize = cfg.BatchSize
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	return nil
}

func getReceiptsFromBlock(blk *block.Block) map[hash.Hash256]*action.Receipt {
	receipts := make(map[hash.Hash256]*action.Receipt, len(blk.Receipts))
	for _, receipt := range blk.Receipts {
		receipts[receipt.ActionHash] = receipt
	}
	return receipts
}

// TODO: many repeated code, move to common package
func getPayloadAmount(act action.Action) (*big.Int, []byte) {
	amount := big.NewInt(0)

	var payload []byte
	switch a := act.(type) {
	case *action.Transfer:
		amount = a.Amount()
		payload = a.Payload()
	case *action.Execution:
		amount = a.Amount()
	case *action.DepositToRewardingFund:
		amount = a.Amount()
	case *action.ClaimFromRewardingFund:
		amount = a.ClaimAmount()
	case *action.CreateStake:
		amount = a.Amount()
		payload = a.Payload()
	case *action.DepositToStake:
		amount = a.Amount()
		payload = a.Payload()
	case *action.CandidateRegister:
		amount = a.Amount()
		payload = a.Payload()
	}
	return amount, payload
}

func getAccount(selp *action.SealedEnvelope, receipt *action.Receipt) (address.Address, string, error) {
	sender, err := address.FromBytes(selp.SrcPubkey().Hash())
	if err != nil {
		return nil, "", err
	}
	dst, _ := selp.Destination()
	return sender, dst, nil
}

func (b blockActionPlugin) BatchSize() int {
	return b.batchSize
}

func (b blockActionPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	var allActs []models.BlockAction
	var from, to uint64
	for i, blk := range blks {
		acts, err := b.handleBlock(ctx, blk)
		if err != nil {
			return err
		}
		allActs = append(allActs, acts...)
		if i == 0 {
			from = blk.Height()
		}
		to = blk.Height()
	}
	return b.commitActs(allActs, from, to)
}

func (b blockActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	acts, err := b.handleBlock(ctx, blk)
	if err != nil {
		return err
	}
	return b.commitActs(acts, blk.Height(), blk.Height())
}

func (b blockActionPlugin) handleBlock(ctx context.Context, blk *block.Block) ([]models.BlockAction, error) {
	t := time.Now()
	receipts := getReceiptsFromBlock(blk)
	processTimeMetric.WithLabelValues(b.Name(), "getReceipts").Observe(time.Since(t).Seconds())
	t = time.Now()
	var acts []models.BlockAction

	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		receipt, ok := receipts[actionHash]
		if !ok {
			continue
		}
		sender, dst, err := getAccount(selp, receipt)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get accounts from action %s", actionHash)
		}

		gasPrice := decimal.NewFromBigInt(selp.GasPrice(), 0)
		if selp.TxType() != action.LegacyTxType && selp.TxType() != action.AccessListTxType {
			gasPrice = decimal.NewFromBigInt(receipt.EffectiveGasPrice, 0)
		}
		gasLimit := selp.Gas()
		nonce := selp.Nonce()

		act := selp.Action()
		actionType := getActionTypeString(act)
		amount, payload := getPayloadAmount(act)

		amountDec := decimal.NewFromBigInt(amount, 0)
		acts = append(acts, models.BlockAction{
			ActionHash:         hex.EncodeToString(actionHash[:]),
			ActionType:         actionType,
			BlockHeight:        blk.Height(),
			Sender:             sender.String(),
			Recipient:          dst,
			GasPrice:           gasPrice,
			GasLimit:           gasLimit,
			Nonce:              nonce,
			Amount:             amountDec,
			GasConsumed:        receipt.GasConsumed,
			ChainID:            selp.ChainID(),
			Encoding:           selp.Encoding(),
			Version:            0, // TODO: how to get version
			ContractAddress:    receipt.ContractAddress,
			Status:             receipt.Status,
			Timestamp:          time.Unix(blk.Timestamp().Unix(), 0),
			ExecutionRevertMsg: strings.ReplaceAll(receipt.ExecutionRevertMsg(), string([]byte{0x00}), "0x00"),
			Payload:            payload,
		})
	}
	processTimeMetric.WithLabelValues(b.Name(), "makeData").Observe(time.Since(t).Seconds())
	return acts, nil
}

func (b blockActionPlugin) commitActs(acts []models.BlockAction, from, to uint64) error {
	t := time.Now()
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("block_height >= ? AND block_height <= ?", from, to).Delete(&models.BlockAction{}).Error; err != nil {
			return err
		}
		processTimeMetric.WithLabelValues(b.Name(), "deleteIfExisted").Observe(time.Since(t).Seconds())
		t = time.Now()
		if err := tx.Model(&models.BlockAction{}).CreateInBatches(acts, 200).Error; err != nil {
			return err
		}
		processTimeMetric.WithLabelValues(b.Name(), "insertData").Observe(time.Since(t).Seconds())
		t = time.Now()
		e := db.UpdateIndexHeightByTx(tx, b.Name(), to)
		processTimeMetric.WithLabelValues(b.Name(), "updateIndex").Observe(time.Since(t).Seconds())
		return e
	})
	return err
}

func (b blockActionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockActionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockActionPlugin{}
