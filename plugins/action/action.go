package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.1.1"

type actionPlugin struct {
}

func (b actionPlugin) Name() string {
	return "action"
}

func (b actionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b actionPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.Action{}); err != nil {
		return errors.Wrap(err, "failed to start plugin : "+b.Name())
	}

	return nil
}

func (b actionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		receipts := make(map[hash.Hash256]*action.Receipt, len(blk.Receipts))
		for _, receipt := range blk.Receipts {
			receipts[receipt.ActionHash] = receipt
		}
		for _, selp := range blk.Actions {
			actionHash, _ := selp.Hash()
			receipt, ok := receipts[actionHash]
			if !ok {
				continue
			}
			sender, _ := address.FromBytes(selp.SrcPubkey().Hash())

			dst, _ := selp.Destination()
			if len(dst) > 0 {
				if addr, err := address.FromString(dst); err != nil {
					dst = ""
				} else {
					dst = addr.String()
				}
			}
			gasPrice := decimal.NewFromBigInt(selp.GasPrice(), 0)
			gasLimit := selp.GasLimit()
			nonce := selp.Nonce()

			act := selp.Action()
			actionType := getActionTypeString(act)
			amount := big.NewInt(0)

			switch a := act.(type) {
			case *action.Transfer:
				amount = a.Amount()
			case *action.Execution:
				amount = a.Amount()
			case *action.DepositToRewardingFund:
				amount = a.Amount()
			case *action.ClaimFromRewardingFund:
				amount = a.Amount()
			case *action.CreateStake:
				amount = a.Amount()
			case *action.DepositToStake:
				amount = a.Amount()
			case *action.CandidateRegister:
				amount = a.Amount()
			}

			amountDec := decimal.NewFromBigInt(amount, 0)
			m := &models.Action{
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
				ContractAddress:    receipt.ContractAddress,
				Status:             receipt.Status,
				Timestamp:          time.Unix(blk.Timestamp().Unix(), 0),
				ExecutionRevertMsg: receipt.ExecutionRevertMsg(),
			}
			if err := tx.Create(m).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b actionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b actionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = actionPlugin{}
