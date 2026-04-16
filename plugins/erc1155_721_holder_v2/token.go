package main

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const VERSION = "2.1.0"
const holderBatchSize = 256

type tokenPlugin struct {
}

func (b tokenPlugin) Name() string {
	return "erc1155_721_holder_" + VERSION
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&Erc1155721Holder{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b tokenPlugin) BatchSize() int {
	return holderBatchSize
}

func (b tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return b.PutBlocks(ctx, []*block.Block{blk})
}

func (b tokenPlugin) PutBlocks(ctx context.Context, blks []*block.Block) error {
	if len(blks) == 0 {
		return nil
	}
	ops, err := collectHolderOperationsFromBlocks(blks)
	if err != nil {
		return err
	}
	tipHeight := blks[len(blks)-1].Height()
	return db.DB().Transaction(func(tx *gorm.DB) error {
		if err := applyHolderOperations(tx, ops); err != nil {
			return err
		}
		return db.UpdateIndexHeightByTx(tx, b.Name(), tipHeight)
	})
}

func collectHolderOperationsFromBlocks(blks []*block.Block) ([]holderOperation, error) {
	ops := make([]holderOperation, 0)
	for _, blk := range blks {
		blockOps, err := collectHolderOperations(blk)
		if err != nil {
			return nil, err
		}
		ops = append(ops, blockOps...)
	}
	return ops, nil
}

func collectHolderOperations(blk *block.Block) ([]holderOperation, error) {
	ops := make([]holderOperation, 0)
	for _, receipt := range blk.Receipts {
		if receipt.Status != successStatus {
			continue
		}
		actionHash := hex.EncodeToString(receipt.ActionHash[:])
		for _, log := range receipt.Logs() {
			if log.Address == "" || len(log.Topics) < 2 {
				continue
			}
			ok, err := isErc721(log.Address)
			if err != nil {
				return nil, err
			}
			if ok {
				erc721Ops, err := collectERC721Operations(log, actionHash)
				if err != nil {
					return nil, err
				}
				ops = append(ops, erc721Ops...)
			}
			ok, err = isErc1155(log.Address)
			if err != nil {
				return nil, err
			}
			if ok {
				erc1155Ops, err := collectERC1155Operations(log)
				if err != nil {
					return nil, err
				}
				ops = append(ops, erc1155Ops...)
			}
		}
	}
	return ops, nil
}

func collectERC721Operations(log *action.Log, actionHash string) ([]holderOperation, error) {
	if log.Topics[0] != Transfer {
		return nil, nil
	}
	event := struct {
		From    common.Address
		To      common.Address
		TokenId *big.Int
	}{}
	if err := kernel.UnpackLog(erc721ABI, &event, "Transfer", log); err != nil {
		return nil, errors.WithMessagef(err, "failed to unpack Transfer on action: %s", actionHash)
	}
	tokenID := decimal.NewFromBigInt(event.TokenId, 0).String()
	fromAddr, _ := address.FromBytes(event.From.Bytes())
	toAddr, _ := address.FromBytes(event.To.Bytes())
	switch {
	case fromAddr.String() == address.ZeroAddress:
		return []holderOperation{{
			opType:   holderOpERC721Mint,
			contract: log.Address,
			to:       toAddr.String(),
			tokenID:  tokenID,
			ercType:  721,
			value:    decimal.NewFromInt(1),
		}}, nil
	case toAddr.String() == address.ZeroAddress:
		return []holderOperation{{
			opType:   holderOpERC721Burn,
			contract: log.Address,
			from:     fromAddr.String(),
			tokenID:  tokenID,
			ercType:  721,
			value:    decimal.NewFromInt(1),
		}}, nil
	default:
		return []holderOperation{{
			opType:   holderOpERC721Transfer,
			contract: log.Address,
			from:     fromAddr.String(),
			to:       toAddr.String(),
			tokenID:  tokenID,
			ercType:  721,
			value:    decimal.NewFromInt(1),
		}}, nil
	}
}

func collectERC1155Operations(log *action.Log) ([]holderOperation, error) {
	var tokenIDs, tokenVals []*big.Int
	var fromAddr, toAddr address.Address
	switch log.Topics[0] {
	case HashTransferBatch:
		event := struct {
			Operator common.Address
			From     common.Address
			To       common.Address
			Ids      []*big.Int
			Values   []*big.Int
		}{}
		if err := kernel.UnpackLog(erc1155ABI, &event, "TransferBatch", log); err != nil {
			return nil, err
		}
		fromAddr, _ = address.FromBytes(event.From.Bytes())
		toAddr, _ = address.FromBytes(event.To.Bytes())
		tokenIDs = event.Ids
		tokenVals = event.Values
	case HashTransferSingle:
		event := struct {
			Operator common.Address
			From     common.Address
			To       common.Address
			Id       *big.Int
			Value    *big.Int
		}{}
		if err := kernel.UnpackLog(erc1155ABI, &event, "TransferSingle", log); err != nil {
			return nil, err
		}
		fromAddr, _ = address.FromBytes(event.From.Bytes())
		toAddr, _ = address.FromBytes(event.To.Bytes())
		tokenIDs = []*big.Int{event.Id}
		tokenVals = []*big.Int{event.Value}
	default:
		return nil, nil
	}

	ops := make([]holderOperation, 0, len(tokenIDs))
	for idx, tokenID := range tokenIDs {
		tokenIDDec := decimal.NewFromBigInt(tokenID, 0).String()
		tokenValDec := decimal.NewFromBigInt(tokenVals[idx], 0)
		switch {
		case fromAddr.String() == address.ZeroAddress:
			ops = append(ops, holderOperation{
				opType:   holderOpERC1155Mint,
				contract: log.Address,
				to:       toAddr.String(),
				tokenID:  tokenIDDec,
				ercType:  1155,
				value:    tokenValDec,
			})
		case toAddr.String() == address.ZeroAddress:
			ops = append(ops, holderOperation{
				opType:   holderOpERC1155Burn,
				contract: log.Address,
				from:     fromAddr.String(),
				tokenID:  tokenIDDec,
				ercType:  1155,
				value:    tokenValDec,
			})
		default:
			ops = append(ops, holderOperation{
				opType:   holderOpERC1155Transfer,
				contract: log.Address,
				from:     fromAddr.String(),
				to:       toAddr.String(),
				tokenID:  tokenIDDec,
				ercType:  1155,
				value:    tokenValDec,
			})
		}
	}
	return ops, nil
}

func (b tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b tokenPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = tokenPlugin{}
