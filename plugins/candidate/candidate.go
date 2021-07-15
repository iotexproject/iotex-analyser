package main

import (
	"context"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.0.1"

type candidatePlugin struct {
}

func (b candidatePlugin) Name() string {
	return "candidate"
}

func (b candidatePlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b candidatePlugin) Start(ctx context.Context) error {
	if err := db.DB().AutoMigrate(&Candidate{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func (b candidatePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		actions := make(map[hash.Hash256]action.SealedEnvelope, len(blk.Actions))
		for _, selp := range blk.Actions {
			actions[selp.Hash()] = selp
		}

		for _, receipt := range blk.Receipts {
			if receipt.Status != uint64(iotextypes.ReceiptStatus_Success) {
				continue
			}
			selp, ok := actions[receipt.ActionHash]
			if !ok {
				continue
			}
			act := selp.Action()
			switch a := act.(type) {
			case *action.CandidateRegister:
				createData := Candidate{
					BlockHeight:     blk.Height(),
					Name:            a.Name(),
					OperatorAddress: a.OperatorAddress().String(),
					OwnerAddress:    a.OwnerAddress().String(),
					RewardAddress:   a.RewardAddress().String(),
					Amount:          decimal.NewFromBigInt(a.Amount(), 0),
					ActType:         "CandidateRegister",
					AutoStake:       a.AutoStake(),
					Duration:        a.Duration(),
					Payload:         a.Payload(),
				}
				if err := tx.Create(&createData).Error; err != nil {
					return err
				}
			case *action.CandidateUpdate:
				createData := Candidate{
					BlockHeight:     blk.Height(),
					Name:            a.Name(),
					OperatorAddress: a.OperatorAddress().String(),
					RewardAddress:   a.RewardAddress().String(),
					ActType:         "CandidateUpdate",
				}
				if err := tx.Create(&createData).Error; err != nil {
					return err
				}
			default:
				continue
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})
	return err
}

func (b candidatePlugin) Stop(ctx context.Context) error {
	return nil
}

func (b candidatePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = candidatePlugin{}
