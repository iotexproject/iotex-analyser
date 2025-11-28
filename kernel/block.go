package kernel

import (
	"context"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/blockchain/blockdao"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
)

type BatchBlockDao interface {
	blockdao.BlockDAO
	BatchGetBlocks(start, count uint64) (blks []*block.Block, err error)
}

type batchBlockDao struct {
	blockdao.BlockDAO
	cli iotexapi.APIServiceClient
}

func NewBatchBlockDao(dao blockdao.BlockDAO, cli iotexapi.APIServiceClient) BatchBlockDao {
	return &batchBlockDao{
		BlockDAO: dao,
		cli:      cli,
	}
}

func (b *batchBlockDao) BatchGetBlocks(start, count uint64) (blks []*block.Block, err error) {
	return GetBlocksFromChain(context.Background(), start, count, b.cli)
}

type localBatchBlockDao struct {
	blockdao.BlockDAO
}

func NewLocalBatchBlockDao(dao blockdao.BlockDAO) BatchBlockDao {
	return &localBatchBlockDao{
		BlockDAO: dao,
	}
}

func (l *localBatchBlockDao) BatchGetBlocks(start, count uint64) (blks []*block.Block, err error) {
	for i := uint64(0); i < count; i++ {
		blk, err := GetBlockByHeightFromBlockDAO(start+i, l)
		if err != nil {
			return nil, err
		}
		blks = append(blks, blk)
	}
	return blks, nil
}

func GetBlockByHeightFromBlockDAO(blkHeight uint64, dao BatchBlockDao) (blk *block.Block, err error) {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		switch x := r.(type) {
	// 		case string:
	// 			err = errors.New(x)
	// 		case error:
	// 			err = x
	// 		default:
	// 			err = errors.New("Unknown panic")
	// 		}
	// 	}
	// }()
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
	for _, l := range tlogs.GetLogs() {
		if len(l.GetTransactions()) == 0 {
			continue
		}
		logs := make([]*action.TransactionLog, len(l.GetTransactions()))
		for i, txn := range l.GetTransactions() {
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
		receipts := blk.Receipts
		for k, j := range receipts {
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

func GetBlockByHeightFromChain(ctx context.Context, height uint64) (*block.Block, error) {
	cli := ChainClient()
	count := uint64(1)
	startHeight := height
	blks, err := GetBlocksFromChain(ctx, startHeight, count, cli)
	if err != nil {
		return nil, err
	}
	if len(blks) == 0 {
		return nil, errors.Errorf("block not found at height %d", height)
	}
	return blks[0], nil
}

func GetBlocksFromChain(ctx context.Context, start, count uint64, cli iotexapi.APIServiceClient) ([]*block.Block, error) {
	startHeight := start
	rawRequest := &iotexapi.GetRawBlocksRequest{
		StartHeight:         startHeight,
		Count:               count,
		WithReceipts:        true,
		WithTransactionLogs: true,
	}
	response, err := cli.GetRawBlocks(ctx, rawRequest)
	if err != nil {
		return nil, err
	}
	deser := block.NewDeserializer(config.EVMNetworkID())
	var blocks []*block.Block
	for _, blkInfo := range response.GetBlocks() {
		blk, err := deser.FromBlockProto(blkInfo.GetBlock())
		if err != nil {
			return nil, err
		}
		receipts := make(map[hash.Hash256]*action.Receipt)
		for _, receiptPb := range blkInfo.GetReceipts() {
			receipt := &action.Receipt{}
			receipt.ConvertFromReceiptPb(receiptPb)
			receipts[receipt.ActionHash] = receipt
			blk.Receipts = append(blk.Receipts, receipt)
		}
		for _, tlogs := range blkInfo.GetTransactionLogs().GetLogs() {
			logs := make([]*action.TransactionLog, len(tlogs.GetTransactions()))
			for i, txn := range tlogs.GetTransactions() {
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
		blocks = append(blocks, blk)
	}
	return blocks, nil
}
