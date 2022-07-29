package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.2"

type actionExecutionPlugin struct {
}

func (b actionExecutionPlugin) Name() string {
	return "action_execution"
}

func (b actionExecutionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b actionExecutionPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &ActionExecution{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func getDataFromAction(act action.Action) (string, []byte, error) {
	switch act := act.(type) {
	case *action.Execution:
		data := act.Data()
		if data == nil {
			data = []byte("")
		}
		return act.Contract(), data, nil
	default:
		return "", nil, errors.New("action is not execution")
	}
}

func (b actionExecutionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	actions := blk.Actions
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		for _, selp := range actions {
			actionHash, _ := selp.Hash()
			act := selp.Action()
			var contract string
			var data []byte
			var err error
			if contract, data, err = getDataFromAction(act); err != nil {
				continue
			}
			ae := &ActionExecution{
				BlockHeight: blk.Height(),
				ActionHash:  hex.EncodeToString(actionHash[:]),
				Contract:    contract,
				Data:        data,
			}

			for _, receipt := range blk.Receipts {
				if receipt.ActionHash == actionHash {
					ae.ReceiptContractAddress = receipt.ContractAddress
					break
				}
			}
			if err := tx.Create(ae).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b actionExecutionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b actionExecutionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = actionExecutionPlugin{}
