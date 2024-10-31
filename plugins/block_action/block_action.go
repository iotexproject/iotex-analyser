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
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.3.0"

var (
	processTimeMetric = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "iotex_analyser_plugin_inner_processing_seconds_per_block",
			Help:       "iotex analyser plugin inner processing seconds per block",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"name", "step"},
	)
)

func init() {
	prometheus.MustRegister(processTimeMetric)
}

type blockActionPlugin struct {
}

func (b blockActionPlugin) Name() string {
	return "block_action"
}

func (b blockActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockActionPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(),
		&models.BlockAction{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
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
		amount = a.Amount()
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

func (b blockActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
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
			return errors.Wrapf(err, "failed to get accounts from action %s", actionHash)
		}

		gasPrice := decimal.NewFromBigInt(selp.GasPrice(), 0)
		gasLimit := selp.GasLimit()
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
			Version:            selp.Version(),
			ContractAddress:    receipt.ContractAddress,
			Status:             receipt.Status,
			Timestamp:          time.Unix(blk.Timestamp().Unix(), 0),
			ExecutionRevertMsg: strings.ReplaceAll(receipt.ExecutionRevertMsg(), string([]byte{0x00}), "0x00"),
			Payload:            payload,
		})
	}
	processTimeMetric.WithLabelValues(b.Name(), "makeData").Observe(time.Since(t).Seconds())
	t = time.Now()
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("block_height = ?", blk.Height()).Delete(&models.BlockAction{}).Error; err != nil {
			return err
		}
		processTimeMetric.WithLabelValues(b.Name(), "deleteIfExisted").Observe(time.Since(t).Seconds())
		t = time.Now()
		if err := tx.Model(&models.BlockAction{}).CreateInBatches(acts, 200).Error; err != nil {
			return err
		}
		processTimeMetric.WithLabelValues(b.Name(), "insertData").Observe(time.Since(t).Seconds())
		t = time.Now()
		e := db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
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
