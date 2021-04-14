package main

import "github.com/iotexproject/iotex-analyser/kernel"

func createTables() error {
	var createSql string
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
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return err
	}

	createSql = "CREATE TABLE IF NOT EXISTS `" + receiptTransactionTableName + "` (" +
		"`id` bigint(20) unsigned NOT NULL AUTO_INCREMENT," +
		"`block_height` bigint(20) unsigned NOT NULL DEFAULT '0'," +
		"`action_hash` varchar(64) NOT NULL DEFAULT ''," +
		"`type` tinyint(3) unsigned NOT NULL DEFAULT '0'," +
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
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
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
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return err
	}
	return nil
}
