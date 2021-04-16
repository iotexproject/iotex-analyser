package main

import (
	"database/sql"
	"encoding/hex"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
)

var specialActionHash = hash.ZeroHash256

func createTables() error {
	var createSql string
	db := kernel.GetDB()
	createSql = "CREATE TABLE IF NOT EXISTS `" + receiptTableName + "` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`block_height` bigint(20) unsigned NOT NULL DEFAULT '0'," +
		"`action_hash` varchar(64) NOT NULL DEFAULT ''," +
		"`gas_consumed` int(11) unsigned NOT NULL," +
		"`contract_address` varchar(41) NOT NULL DEFAULT ''," +
		"`status` tinyint(3) unsigned NOT NULL DEFAULT '0'," +
		"`execution_revert_msg` varchar(255) NOT NULL DEFAULT ''," +
		"PRIMARY KEY (`id`)," +
		"KEY `block_height` (`block_height`)," +
		"KEY `action_hash` (`action_hash`(9))" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := db.Exec(createSql); err != nil {
		return err
	}

	createSql = "CREATE TABLE IF NOT EXISTS `" + receiptTransactionTableName + "` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`block_height` bigint(20) unsigned NOT NULL DEFAULT '0'," +
		"`action_hash` varchar(64) NOT NULL DEFAULT ''," +
		"`type` enum('transfer','execution','depositToRewardingFund','claimFromRewardingFund','stakeCreate','stakeWithdraw','stakeAddDeposit','candidateRegisterFee','candidateRegisterSelfStake','gasFee','genesis') NOT NULL," +
		"`amount` DECIMAL(42, 0) UNSIGNED NOT NULL DEFAULT 0," +
		"`sender` varchar(41) NOT NULL DEFAULT ''," +
		"`recipient` varchar(41) NOT NULL DEFAULT ''," +
		"PRIMARY KEY (`id`)," +
		"KEY `block_height` (`block_height`)," +
		"KEY `sender` (`sender`)," +
		"KEY `type` (`type`)," +
		"KEY `recipient` (`recipient`)," +
		"KEY `action_hash` (`action_hash`(9))" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := db.Exec(createSql); err != nil {
		return err
	}

	createSql = "CREATE TABLE IF NOT EXISTS `" + receiptLogTableName + "` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`block_height` bigint(20) unsigned NOT NULL DEFAULT '0'," +
		"`action_hash` varchar(64) NOT NULL DEFAULT ''," +
		"`address` varchar(41) NOT NULL DEFAULT ''," +
		"`topics` blob," +
		"`data` blob," +
		"`index` tinyint(3) unsigned NOT NULL DEFAULT '0'," +
		"`not_fix_topic_copy_bug` tinyint(1) NOT NULL DEFAULT 0," +
		"PRIMARY KEY (`id`)," +
		"KEY `block_height` (`block_height`)," +
		"KEY `action_hash` (`action_hash`(9))," +
		"KEY `address` (`address`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := db.Exec(createSql); err != nil {
		return err
	}

	//check wetcher genesis exist
	hashStr := hex.EncodeToString(specialActionHash[:])
	query := "SELECT count(1) FROM `" + receiptTransactionTableName + "` WHERE action_hash=?"
	var count int
	if err := db.QueryRow(query, hashStr).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	err := kernel.Transaction(func(tx *sql.Tx) error {
		for addr, amount := range Default.Genesis.Account.InitBalanceMap {
			insertData := map[string]interface{}{
				"block_height": uint64(0),
				"action_hash":  hashStr,
				"type":         "genesis",
				"amount":       amount,
				"sender":       "",
				"recipient":    addr,
			}
			if err := kernel.InsertTableData(tx, receiptTransactionTableName, insertData); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func getActionType(t iotextypes.TransactionLogType) string {
	switch {
	case t == iotextypes.TransactionLogType_IN_CONTRACT_TRANSFER:
		return execution
	case t == iotextypes.TransactionLogType_WITHDRAW_BUCKET:
		return stakeWithdraw
	case t == iotextypes.TransactionLogType_CREATE_BUCKET:
		return stakeCreate
	case t == iotextypes.TransactionLogType_DEPOSIT_TO_BUCKET:
		return stakeAddDeposit
	case t == iotextypes.TransactionLogType_CLAIM_FROM_REWARDING_FUND:
		return claimFromRewardingFund
	case t == iotextypes.TransactionLogType_DEPOSIT_TO_REWARDING_FUND:
		return depositToRewardingFund
	case t == iotextypes.TransactionLogType_CANDIDATE_REGISTRATION_FEE:
		return candidateRegisterFee
	case t == iotextypes.TransactionLogType_CANDIDATE_SELF_STAKE:
		return candidateRegisterSelfStake
	case t == iotextypes.TransactionLogType_GAS_FEE:
		return gasFee
	case t == iotextypes.TransactionLogType_NATIVE_TRANSFER:
		return transfer
	}
	return ""
}
