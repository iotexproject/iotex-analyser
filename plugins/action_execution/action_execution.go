package main

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
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
	if err := db.ChConn().Exec(ctx, ActionExecutionDDL); err != nil {
		return errors.Wrapf(err, "failed to create table %s", ActionExecution{}.TableName())
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
	execs := make([]*ActionExecution, 0, len(blk.Actions))
	for _, selp := range blk.Actions {
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
		// TODO: maybe find receipt by index instead of loop
		for _, receipt := range blk.Receipts {
			if receipt.ActionHash == actionHash {
				ae.ReceiptContractAddress = receipt.ContractAddress
				break
			}
		}
		execs = append(execs, ae)
	}
	// batch insert to clickhouse
	batch, err := db.ChConn().PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", ActionExecution{}.TableName()))
	if err != nil {
		return errors.Wrap(err, "failed to prepare batch")
	}
	for _, e := range execs {
		if err := batch.AppendStruct(e); err != nil {
			return errors.Wrap(err, "failed to append struct")
		}
	}
	if err := batch.Send(); err != nil {
		return errors.Wrap(err, "failed to send batch")
	}
	// update index height
	return db.UpdateIndexHeight(b.Name(), blk.Height())
}

func (b actionExecutionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b actionExecutionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = actionExecutionPlugin{}
