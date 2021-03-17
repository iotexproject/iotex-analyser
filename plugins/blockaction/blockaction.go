package main

import (
	"context"
	"database/sql"
	"encoding/hex"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/ioctl/util"
	"github.com/pkg/errors"
)

type blockActionPlugin struct {
	tableName string
}

func (b blockActionPlugin) Name() string {
	return "blockaction"
}
func (b blockActionPlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `" + b.tableName + "` (" +
		"`action_hash` varchar(64) NOT NULL DEFAULT ''," +
		"`action_type` enum('transfer','execution','startSubChain','stopSubChain','putBlock','createDeposit','settleDeposit','createPlumChain','terminatePlumChain','plumPutBlock','plumCreateDeposit','plumStartExit','plumChallengeExit','plumResponseChallengeExit','plumFinalizeExit','plumSettleDeposit','plumTransfer','depositToRewardingFund','claimFromRewardingFund','grantReward','stakeCreate','stakeUnstake','stakeWithdraw','stakeAddDeposit','stakeRestake','stakeChangeCandidate','stakeTransferOwnership','candidateRegister','candidateUpdate','putPollResult') NOT NULL," +
		"`receipt_hash` varchar(64) NOT NULL DEFAULT ''," +
		"`block_height` bigint(20) UNSIGNED NOT NULL DEFAULT 0," +
		"`from` varchar(41) NOT NULL DEFAULT ''," +
		"`to` varchar(41) NOT NULL DEFAULT ''," +
		"`gas_price` bigint(20) UNSIGNED NOT NULL DEFAULT 0," +
		"`gas_limit` int(11) UNSIGNED NOT NULL DEFAULT 0," +
		"`gas_consumed` int(11) UNSIGNED NOT NULL," +
		"`nonce` bigint(20) UNSIGNED NOT NULL DEFAULT 0," +
		"`amount` bigint(20) UNSIGNED NOT NULL DEFAULT 0," +
		"`receipt_status` tinyint(3) unsigned NOT NULL DEFAULT 0," +
		"`contract_address` varchar(41) NOT NULL DEFAULT ''," +
		"PRIMARY KEY (`action_hash`)," +
		"KEY `from` (`from`)," +
		"KEY `to` (`to`)," +
		"KEY `action_type` (`action_type`)," +
		"KEY `block_height` (`block_height`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrap(err, "failed to start blockaction plugin")
	}

	return nil
}

func (b blockActionPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := kernel.Transaction(func(tx *sql.Tx) error {
		for _, selp := range blk.Actions {
			actionHash := selp.Hash()
			callerAddr, err := address.FromBytes(selp.SrcPubkey().Hash())
			if err != nil {
				return err
			}
			dst, _ := selp.Destination()
			gasPrice := selp.GasPrice().String()
			gasLimit := selp.GasLimit()
			nonce := selp.Nonce()

			act := selp.Action()
			actionType := getActionTypeString(act)
			amount := "0"

			switch a := act.(type) {
			case *action.Transfer:
				amount = util.RauToString(a.Amount(), util.IotxDecimalNum)
			case *action.Execution:
				amount = util.RauToString(a.Amount(), util.IotxDecimalNum)
			case *action.DepositToRewardingFund:
				amount = util.RauToString(a.Amount(), util.IotxDecimalNum)
			case *action.ClaimFromRewardingFund:
				amount = util.RauToString(a.Amount(), util.IotxDecimalNum)
			case *action.CreateStake:
				amount = util.RauToString(a.Amount(), util.IotxDecimalNum)
			case *action.DepositToStake:
				amount = util.RauToString(a.Amount(), util.IotxDecimalNum)
			case *action.CandidateRegister:
				amount = util.RauToString(a.Amount(), util.IotxDecimalNum)
			}
			insertData := map[string]interface{}{
				"block_height":     blk.Height(),
				"action_hash":      hex.EncodeToString(actionHash[:]),
				"action_type":      actionType,
				"receipt_hash":     "",
				"from":             callerAddr.String(),
				"to":               dst,
				"gas_price":        gasPrice,
				"gas_limit":        gasLimit,
				"gas_consumed":     "",
				"nonce":            nonce,
				"amount":           amount,
				"receipt_status":   "",
				"contract_address": "",
			}
			for _, receipt := range blk.Receipts {
				if receipt.ActionHash == actionHash {
					receiptHash := receipt.Hash()
					insertData["receipt_hash"] = hex.EncodeToString(receiptHash[:])
					insertData["gas_consumed"] = receipt.GasConsumed
					insertData["receipt_status"] = receipt.Status
					insertData["contract_address"] = receipt.ContractAddress
					break
				}
			}
			if err := kernel.InsertTableData(tx, b.tableName, insertData); err != nil {
				return err
			}
		}
		return kernel.UpdateIndexHeight(tx, b.Name(), blk.Height())
	})

	return err
}

func (b blockActionPlugin) Stop(ctx context.Context) error {
	return nil
}

// exported
var Plugin = blockActionPlugin{
	tableName: "block_action",
}
