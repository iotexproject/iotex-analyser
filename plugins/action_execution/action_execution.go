package main

import (
	"context"
	"encoding/hex"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "3.0.0"

// methodSelectorLen is the Solidity ABI method-selector length. We store
// only the selector in action_execution.data; the full calldata, when
// needed (analyser-api ActionByHash with input_data), is fetched from the
// chain via gRPC. See iotex-analyser-api/common/chainrpc.
const methodSelectorLen = 4

type actionExecutionPlugin struct {
	tipHeight uint64
	actions   []*ActionExecution
}

func (b *actionExecutionPlugin) Name() string {
	return "action_execution"
}

func (b *actionExecutionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b *actionExecutionPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &ActionExecution{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	return nil
}

func getDataFromAction(act action.Action) (string, []byte, error) {
	switch act := act.(type) {
	case *action.Execution:
		// Copy into a fresh small buffer so we don't pin the original
		// calldata (often >1 KB) until the per-block batch commits, and
		// so the NOT NULL column always receives a non-nil slice.
		src := act.Data()
		n := len(src)
		if n > methodSelectorLen {
			n = methodSelectorLen
		}
		data := make([]byte, n)
		copy(data, src)
		return act.Contract(), data, nil
	default:
		return "", nil, errors.New("action is not execution")
	}
}

func (b *actionExecutionPlugin) putBlock(ctx context.Context, blk *block.Block) error {
	for _, selp := range blk.Actions {
		actionHash, _ := selp.Hash()
		act := selp.Action()
		contract, data, err := getDataFromAction(act)
		if err != nil {
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
		b.actions = append(b.actions, ae)
	}
	return nil
}

func (b *actionExecutionPlugin) commit() error {
	actions := b.actions
	b.actions = nil
	tipHeight := b.tipHeight
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if len(actions) > 0 {
			if err := tx.CreateInBatches(actions, 200).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func (b *actionExecutionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	if err := b.putBlock(ctx, blk); err != nil {
		return err
	}
	b.tipHeight = blk.Height()
	return b.commit()
}

func (b *actionExecutionPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	for _, blk := range blks {
		if err := b.putBlock(ctx, blk); err != nil {
			return err
		}
	}
	b.tipHeight = blks[len(blks)-1].Height()
	return b.commit()
}

func (b *actionExecutionPlugin) BatchSize() int {
	return 1000
}

func (b *actionExecutionPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b *actionExecutionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = actionExecutionPlugin{}
