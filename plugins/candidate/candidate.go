package main

import (
	"context"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

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
		for _, selp := range blk.Actions {
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
					Nonce:           a.Nonce(),
					GasLimit:        a.GasLimit(),
					GasPrice:        decimal.NewFromBigInt(a.GasPrice(), 0),
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
					Nonce:           a.Nonce(),
					GasLimit:        a.GasLimit(),
					GasPrice:        decimal.NewFromBigInt(a.GasPrice(), 0),
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
