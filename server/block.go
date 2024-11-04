package server

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
)

func getBlockString(blk *block.Block) string {
	res := bytes.Buffer{}
	res.Reset()
	res.WriteString(fmt.Sprintf("blockHeight: %d\n", blk.Height()))
	res.WriteString(fmt.Sprintf("===== header =====\n"))
	txtRoot := blk.Header.TxRoot()
	res.WriteString(fmt.Sprintf("txRoot : %s\n", hex.EncodeToString(txtRoot[:])))
	calTxtRoot, _ := blk.CalculateTxRoot()
	res.WriteString(fmt.Sprintf("CalculateTxRoot : %s\n", hex.EncodeToString(calTxtRoot[:])))

	for i, selp := range blk.Actions {
		res.WriteString(fmt.Sprintf("===== action: #%d =====\n", i))
		actionHash, _ := selp.Hash()
		res.WriteString(fmt.Sprintf("actionHash : %s\n", hex.EncodeToString(actionHash[:])))
		sender, _ := address.FromBytes(selp.SrcPubkey().Hash())
		res.WriteString(fmt.Sprintf("from : %s\n", sender))

		dst, _ := selp.Destination()
		res.WriteString(fmt.Sprintf("to : %s\n", dst))
		gasPrice := selp.GasPrice().String()
		res.WriteString(fmt.Sprintf("gasPrice : %s\n", gasPrice))
		gasLimit := selp.GasLimit()
		res.WriteString(fmt.Sprintf("gasLimit : %d\n", gasLimit))
		res.WriteString(fmt.Sprintf("gasPrice : %s\n", gasPrice))
		nonce := selp.Nonce()
		res.WriteString(fmt.Sprintf("nonce : %d\n", nonce))

		act := selp.Action()
		actionType := fmt.Sprintf("%T", act)
		res.WriteString(fmt.Sprintf("actionType : %s\n", actionType))
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
		res.WriteString(fmt.Sprintf("amount : %s\n", amount))
	}
	for j, receipt := range blk.Receipts {
		res.WriteString(fmt.Sprintf("===== receipt: #%d =====\n", j))
		res.WriteString(fmt.Sprintf("receipt.ActionHash : %s\n", hex.EncodeToString(receipt.ActionHash[:])))
		receiptHash := receipt.Hash()
		res.WriteString(fmt.Sprintf("receipt.Hash : %s\n", hex.EncodeToString(receiptHash[:])))
		res.WriteString(fmt.Sprintf("receipt.Status : %d\n", receipt.Status))
		res.WriteString(fmt.Sprintf("receipt.GasConsumed : %d\n", receipt.GasConsumed))
		res.WriteString(fmt.Sprintf("receipt.BlockHeight : %d\n", receipt.BlockHeight))
		res.WriteString(fmt.Sprintf("receipt.ContractAddress : %s\n", receipt.ContractAddress))
		res.WriteString(fmt.Sprintf("receipt.executionRevertMsg : %s\n", receipt.ExecutionRevertMsg()))
		for j1, transLog := range receipt.TransactionLogs() {
			res.WriteString(fmt.Sprintf("===== receipt: #%d transaction: #%d =====\n", j, j1))
			res.WriteString(fmt.Sprintf("transaction.Type : %s\n", transLog.Type))
			res.WriteString(fmt.Sprintf("transaction.Amount : %s\n", transLog.Amount.String()))
			res.WriteString(fmt.Sprintf("transaction.Sender : %s\n", transLog.Sender))
			res.WriteString(fmt.Sprintf("transaction.Recipient : %s\n", transLog.Recipient))
		}
		for j2, log := range receipt.Logs() {
			res.WriteString(fmt.Sprintf("===== receipt: #%d log: #%d =====\n", j, j2))
			if len(log.Topics) > 0 {
				bucketIndex := new(big.Int).SetBytes(log.Topics[1][:])

				res.WriteString(fmt.Sprintf("bucketID : %v \n", bucketIndex.String()))
			}
			res.WriteString(fmt.Sprintf("log.Address : %s\n", log.Address))
			res.WriteString(fmt.Sprintf("log.Topics : %x\n", log.Topics))
			res.WriteString(fmt.Sprintf("log.Data : %x\n", log.Data))
			res.WriteString(fmt.Sprintf("log.BlockHeight : %d\n", log.BlockHeight))
			res.WriteString(fmt.Sprintf("log.ActionHash : %s\n", hex.EncodeToString(log.ActionHash[:])))
			res.WriteString(fmt.Sprintf("log.Index : %d\n", log.Index))
			res.WriteString(fmt.Sprintf("log.NotFixTopicCopyBug : %v\n", log.NotFixTopicCopyBug))

		}

	}
	return res.String()
}
