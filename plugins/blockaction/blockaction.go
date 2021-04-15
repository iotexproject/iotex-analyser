package main

import (
	"context"
	"database/sql"
	"encoding/hex"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
)

const VERSION = "1.1.0"

type blockActionPlugin struct {
	tableName string
}

func (b blockActionPlugin) Name() string {
	return "blockaction"
}

func (b blockActionPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b blockActionPlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `" + b.tableName + "` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`action_hash` varchar(64) NOT NULL DEFAULT ''," +
		"`action_type` enum('transfer','execution','startSubChain','stopSubChain','putBlock','createDeposit','settleDeposit','createPlumChain','terminatePlumChain','plumPutBlock','plumCreateDeposit','plumStartExit','plumChallengeExit','plumResponseChallengeExit','plumFinalizeExit','plumSettleDeposit','plumTransfer','depositToRewardingFund','claimFromRewardingFund','grantReward','createStake','withdrawStake','restake','changeCandidate','transferStake','candidateRegister','candidateUpdate','putPollResult','depositToStake','unstake') NOT NULL," +
		"`block_height` bigint(20) UNSIGNED NOT NULL DEFAULT 0," +
		"`from` varchar(41) NOT NULL DEFAULT ''," +
		"`to` varchar(41) NOT NULL DEFAULT ''," +
		"`gas_price` DECIMAL(42, 0) UNSIGNED NOT NULL DEFAULT 0," +
		"`gas_limit` int(11) UNSIGNED NOT NULL DEFAULT 0," +
		"`nonce` bigint(20) UNSIGNED NOT NULL DEFAULT 0," +
		"`amount` DECIMAL(42, 0) UNSIGNED NOT NULL DEFAULT 0," +
		"PRIMARY KEY (`id`)," +
		"KEY `from` (`from`)," +
		"KEY `to` (`to`)," +
		"KEY `action_type` (`action_type`)," +
		"KEY `block_height` (`block_height`)," +
		"KEY `action_hash` (`action_hash`(9))" +
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
			sender, _ := address.FromBytes(selp.SrcPubkey().Hash())

			dst, _ := selp.Destination()
			gasPrice := selp.GasPrice().String()
			gasLimit := selp.GasLimit()
			nonce := selp.Nonce()

			act := selp.Action()
			actionType := getActionTypeString(act)
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
			insertData := map[string]interface{}{
				"block_height": blk.Height(),
				"action_hash":  hex.EncodeToString(actionHash[:]),
				"action_type":  actionType,
				"from":         sender.String(),
				"to":           dst,
				"gas_price":    gasPrice,
				"gas_limit":    gasLimit,
				"nonce":        nonce,
				"amount":       amount,
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

func (b blockActionPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockActionPlugin{
	tableName: "block_action",
}
