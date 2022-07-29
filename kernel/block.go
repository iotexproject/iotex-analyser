package kernel

import (
	"context"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
)

func GetBlockByHeightFromBlockDAO(blkHeight uint64, dao blockdao.BlockDAO) (blk *block.Block, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown panic")
			}
		}
	}()
	blk, err = dao.GetBlockByHeight(blkHeight)
	if err != nil {
		return nil, err
	}
	blk.Receipts, err = dao.GetReceipts(blkHeight)
	if err != nil {
		return nil, err
	}
	tlogs, err := dao.TransactionLogs(blkHeight)
	if err != nil {
		return nil, err
	}
	return processTransactionLog(blk, tlogs)
}

func processTransactionLog(blk *block.Block, tlogs *iotextypes.TransactionLogs) (*block.Block, error) {
	for _, l := range tlogs.Logs {
		if len(l.Transactions) == 0 {
			continue
		}
		l := l
		logs := make([]*action.TransactionLog, len(l.Transactions))
		for i, txn := range l.Transactions {
			i := i
			txn := txn
			amount, ok := new(big.Int).SetString(txn.Amount, 10)
			if !ok {
				return nil, errors.New("failed to parse transaction amount")
			}
			logs[i] = &action.TransactionLog{
				Type:      txn.Type,
				Amount:    amount,
				Sender:    txn.Sender,
				Recipient: txn.Recipient,
			}
		}
		for k, j := range blk.Receipts {
			k := k
			if j.ActionHash == hash.BytesToHash256(l.ActionHash) {
				if len(j.TransactionLogs()) == 0 {
					blk.Receipts[k] = j.AddTransactionLogs(logs...)
				}
				if len(blk.Receipts[k].TransactionLogs()) != len(l.GetTransactions()) {
					return nil, errors.New("transaction log length does not match")
				}
			}
		}
	}
	return blk, nil
}

func GetBlockByHeightFromChain(height uint64) (*block.Block, error) {
	chainClient := ChainClient()
	count := uint64(1)
	startHeight := height
	rawRequest := &iotexapi.GetRawBlocksRequest{
		StartHeight:  startHeight,
		Count:        count,
		WithReceipts: true,
	}
	getRawBlocksRes, err := chainClient.GetRawBlocks(context.Background(), rawRequest)
	if err != nil {
		return nil, err
	}

	for _, blkInfo := range getRawBlocksRes.GetBlocks() {
		blk := &block.Block{}
		if err := blk.ConvertFromBlockPb(blkInfo.GetBlock()); err != nil {
			return nil, err
		}
		receipts := map[hash.Hash256]*action.Receipt{}
		for _, receiptPb := range blkInfo.GetReceipts() {
			receipt := &action.Receipt{}
			receipt.ConvertFromReceiptPb(receiptPb)
			receipts[receipt.ActionHash] = receipt
			blk.Receipts = append(blk.Receipts, receipt)
		}
		transactionLogs, err := chainClient.GetTransactionLogByBlockHeight(
			context.Background(),
			&iotexapi.GetTransactionLogByBlockHeightRequest{
				BlockHeight: blk.Header.Height(),
			},
		)
		if err != nil {
			return nil, err
		}
		for _, tlogs := range transactionLogs.TransactionLogs.Logs {
			logs := make([]*action.TransactionLog, len(tlogs.Transactions))
			for i, txn := range tlogs.Transactions {
				amount, ok := new(big.Int).SetString(txn.Amount, 10)
				if !ok {
					return nil, errors.Errorf("failed to parse %s", txn.Amount)
				}
				logs[i] = &action.TransactionLog{
					Type:      txn.Type,
					Amount:    amount,
					Sender:    txn.Sender,
					Recipient: txn.Recipient,
				}
			}
			actHash := hash.BytesToHash256(tlogs.ActionHash)
			receipts[actHash].AddTransactionLogs(logs...)
		}
		if blk.Height() == height {
			return blk, nil
		}
	}
	return nil, errors.New("failed to get block by height")
}
