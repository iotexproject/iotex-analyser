package main

import (
	"context"
	"database/sql"
	"math/big"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
)

const VERSION = "1.0.1"

type blockMetaRewardPlugin struct {
	tableName string
}

func (b blockMetaRewardPlugin) Name() string {
	return "blockmeta_reward"
}

/*
DROP PROCEDURE IF EXISTS `?`;
DELIMITER //
CREATE PROCEDURE `?`()
BEGIN
  DECLARE CONTINUE HANDLER FOR SQLEXCEPTION BEGIN END;
  ALTER TABLE `block_meta` ADD COLUMN `block_reward` bigint(20) NOT NULL default '0';
END //
DELIMITER ;
CALL `?`();
DROP PROCEDURE `?`;
*/
func (b blockMetaRewardPlugin) Start(ctx context.Context) error {
	createSql := "ALTER TABLE `block_meta` ADD COLUMN `block_reward` bigint(20) NOT NULL default '0';"
	//skipping
	if _, err := kernel.GetDB().Exec(createSql); err != nil && kernel.MySQLErrorCode(err) != 1060 {
		return errors.Wrapf(err, "failed to start plugin: %s", b.Name())
	}

	return nil
}

func (b blockMetaRewardPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	grantRewardActs := make(map[hash.Hash256]bool)
	// log action index
	for _, selp := range blk.Actions {
		if _, ok := selp.Action().(*action.GrantReward); ok {
			grantRewardActs[selp.Hash()] = true
		}
	}
	totalReward := big.NewInt(0)
	// log receipt index
	for _, receipt := range blk.Receipts {
		if _, ok := grantRewardActs[receipt.ActionHash]; ok {
			// Parse receipt of grant reward
			rewardInfoMap, err := getRewardInfoFromReceipt(receipt)
			if err != nil {
				return errors.Wrap(err, "failed to get reward info from receipt")
			}
			if len(rewardInfoMap) == 0 {
				continue
			}
			for _, rewards := range rewardInfoMap {
				totalReward.Add(totalReward, rewards.BlockReward)
			}
		}
	}

	updateMap := map[string]interface{}{
		"block_reward": totalReward.String(),
	}
	whereMap := map[string]interface{}{
		"block_height": blk.Height(),
	}
	err := kernel.Transaction(func(tx *sql.Tx) error {
		var count int
		row := tx.QueryRow("SELECT count(1) FROM `"+b.tableName+"` WHERE block_height=?", blk.Height())
		if err := row.Scan(&count); err != nil {
			return err
		}
		if err := kernel.UpdateTableData(tx, b.tableName, updateMap, whereMap); err != nil {
			return err
		}
		return kernel.UpdateIndexHeight(tx, b.Name(), blk.Height())
	})
	return err
}

func (b blockMetaRewardPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b blockMetaRewardPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = blockMetaRewardPlugin{
	tableName: "block_meta",
}
