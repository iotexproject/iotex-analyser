package tools

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/action/protocol"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/urfave/cli/v2"
)

var TraceBlock = &cli.Command{
	Name:        "traceblock",
	Usage:       "traceblock --block <height>",
	Description: `trace block in blockDAO`,
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "block",
			Usage: "block height",
			Value: 1,
		},
	},
	Action: traceBlock,
}

func debugBlock(blk *block.Block) {
	fmt.Println("blockHeight: ", blk.Height())
	fmt.Printf("===== header =====\n")
	txtRoot := blk.Header.TxRoot()
	fmt.Printf("txRoot : %s\n", hex.EncodeToString(txtRoot[:]))
	calTxtRoot := blk.CalculateTxRoot()
	fmt.Printf("CalculateTxRoot : %s\n", hex.EncodeToString(calTxtRoot[:]))

	for i, selp := range blk.Actions {
		fmt.Printf("===== action: #%d =====\n", i)
		actionHash := selp.Hash()
		fmt.Printf("actionHash : %s\n", hex.EncodeToString(actionHash[:]))
		sender, _ := address.FromBytes(selp.SrcPubkey().Hash())
		fmt.Printf("from : %s\n", sender)

		dst, _ := selp.Destination()
		fmt.Printf("to : %s\n", dst)
		gasPrice := selp.GasPrice().String()
		fmt.Printf("gasPrice : %s\n", gasPrice)
		gasLimit := selp.GasLimit()
		fmt.Printf("gasLimit : %d\n", gasLimit)
		fmt.Printf("gasPrice : %s\n", gasPrice)
		nonce := selp.Nonce()
		fmt.Printf("nonce : %d\n", nonce)

		act := selp.Action()
		actionType := fmt.Sprintf("%T", act)
		fmt.Printf("actionType : %s\n", actionType)
		amount := "0"

		switch a := act.(type) {
		case *action.Transfer:
			amount = a.Amount().String()
		case *action.Execution:
			amount = a.Amount().String()
		case *action.DepositToRewardingFund:
			amount = a.Amount().String()
		case *action.ClaimFromRewardingFund:
			amount = a.Amount().String()
		case *action.CreateStake:
			amount = a.Amount().String()
		case *action.DepositToStake:
			amount = a.Amount().String()
		case *action.CandidateRegister:
			amount = a.Amount().String()
		}
		fmt.Printf("amount : %s\n", amount)
	}
	for j, receipt := range blk.Receipts {
		fmt.Printf("===== receipt: #%d =====\n", j)
		fmt.Printf("receipt.ActionHash : %s\n", hex.EncodeToString(receipt.ActionHash[:]))
		receiptHash := receipt.Hash()
		fmt.Printf("receipt.Hash : %s\n", hex.EncodeToString(receiptHash[:]))
		fmt.Printf("receipt.Status : %d\n", receipt.Status)
		fmt.Printf("receipt.GasConsumed : %d\n", receipt.GasConsumed)
		fmt.Printf("receipt.BlockHeight : %d\n", receipt.BlockHeight)
		fmt.Printf("receipt.ContractAddress : %s\n", receipt.ContractAddress)
		fmt.Printf("receipt.executionRevertMsg : %s\n", receipt.ExecutionRevertMsg())
		for j1, transLog := range receipt.TransactionLogs() {
			fmt.Printf("===== receipt: #%d transaction: #%d =====\n", j, j1)
			fmt.Printf("transaction.Type : %s\n", transLog.Type)
			fmt.Printf("transaction.Amount : %s\n", transLog.Amount.String())
			fmt.Printf("transaction.Sender : %s\n", transLog.Sender)
			fmt.Printf("transaction.Recipient : %s\n", transLog.Recipient)
		}
		for j2, log := range receipt.Logs() {
			fmt.Printf("===== receipt: #%d log: #%d =====\n", j, j2)
			if len(log.Topics) > 0 {
				bucketIndex := new(big.Int).SetBytes(log.Topics[1][:])

				fmt.Printf("bucketID : %v \n", bucketIndex.String())
			}
			fmt.Printf("log.Address : %s\n", log.Address)
			fmt.Printf("log.Topics : %x\n", log.Topics)
			fmt.Printf("log.Data : %x\n", log.Data)
			fmt.Printf("log.BlockHeight : %d\n", log.BlockHeight)
			fmt.Printf("log.ActionHash : %s\n", hex.EncodeToString(log.ActionHash[:]))
			fmt.Printf("log.Index : %d\n", log.Index)
			fmt.Printf("log.NotFixTopicCopyBug : %v\n", log.NotFixTopicCopyBug)

		}

	}
}

func traceBlock(c *cli.Context) error {
	var tip protocol.TipInfo
	ctxDao := protocol.WithBlockchainCtx(
		context.Background(),
		protocol.BlockchainCtx{
			Genesis: genesis.Default,
			Tip:     tip,
		},
	)
	var indexers []blockdao.BlockIndexer
	var dao blockdao.BlockDAO
	dao = blockdao.NewBlockDAO(indexers, config.Default.BlockDB)
	if err := dao.Start(ctxDao); err != nil {
		return err
	}
	block, err := dao.GetBlockByHeight(c.Uint64("block"))
	if err != nil {
		return err
	}
	debugBlock(block)
	return dao.Stop(ctxDao)
}
