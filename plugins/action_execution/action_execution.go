package main

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

const VERSION = "2.0.2"

type actionExecutionPlugin struct {
	batchSize int
}

func (b actionExecutionPlugin) Name() string {
	return "action_execution"
}

func (b actionExecutionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b actionExecutionPlugin) BatchSize() int {
	return b.batchSize
}

func (b *actionExecutionPlugin) Start(ctx context.Context) error {
	var err error
	cfg := &Config{
		BatchSize: 200,
	}
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	b.batchSize = cfg.BatchSize

	if err := db.ChConn().Exec(ctx, models.ActionExecutionDDL); err != nil {
		return errors.Wrapf(err, "failed to create table %s", models.ActionExecution{}.TableName())
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

func (b actionExecutionPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	execs := make([]*models.ActionExecution, 0)
	for _, blk := range blks {
		execs = append(execs, b.handleBlock(ctx, blk)...)
	}
	return b.commit(ctx, execs, blks[len(blks)-1].Height())
}

func (b actionExecutionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return b.commit(ctx, b.handleBlock(ctx, blk), blk.Height())
}

func (b actionExecutionPlugin) handleBlock(ctx context.Context, blk *block.Block) []*models.ActionExecution {
	execs := make([]*models.ActionExecution, 0, len(blk.Actions))
	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		act := selp.Action()
		var contract string
		var data []byte
		var err error
		if contract, data, err = getDataFromAction(act); err != nil {
			continue
		}
		ae := &models.ActionExecution{
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
	return execs
}

func (b actionExecutionPlugin) commit(ctx context.Context, execs []*models.ActionExecution, height uint64) error {
	// batch insert to clickhouse
	batch, err := db.ChConn().PrepareBatch(ctx, fmt.Sprintf("INSERT INTO %s", models.ActionExecution{}.TableName()))
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
	return db.UpdateIndexHeight(b.Name(), height)
}

func (b actionExecutionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b actionExecutionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = actionExecutionPlugin{}
