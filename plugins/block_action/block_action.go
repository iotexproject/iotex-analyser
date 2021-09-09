package main

import (
	"context"
	"encoding/hex"
	"math/big"

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

const VERSION = "2.0.2"

type blockActionPlugin struct {
}

func (b blockActionPlugin) Name() string {
	return "block_action"
}

func (b blockActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockActionPlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&models.BlockAction{}); err != nil {
		return errors.Wrap(err, "failed to start block plugin")
	}

	return nil
}

func (b blockActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, selp := range blk.Actions {
			actionHash, err := selp.Hash()
			if err != nil {
				return err
			}
			sender, _ := address.FromBytes(selp.SrcPubkey().Hash())

			dst, _ := selp.Destination()
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
			m := &models.BlockAction{
				ActionHash:  hex.EncodeToString(actionHash[:]),
				ActionType:  actionType,
				BlockHeight: blk.Height(),
				From:        sender.String(),
				To:          dst,
				GasPrice:    gasPrice,
				GasLimit:    gasLimit,
				Nonce:       nonce,
				Amount:      amountDec,
			}
			if err := tx.Create(m).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
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
