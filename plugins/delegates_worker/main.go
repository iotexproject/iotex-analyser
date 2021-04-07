package main

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const VERSION = "1.0.1"
const TableName = "node_delegates"

type delegatesWorkerPlugin struct {
	tableName string
	dao       blockdao.BlockDAO
	stop      chan bool
	once      *sync.Once
}

func (b delegatesWorkerPlugin) Name() string {
	return "delegates_worker"
}

func (b delegatesWorkerPlugin) Type() plugin.Type {
	return plugin.TypeWorker
}

func (b delegatesWorkerPlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `" + b.tableName + "` (" +
		"`id` int(11) unsigned NOT NULL AUTO_INCREMENT," +
		"`block_height` bigint(20) unsigned NOT NULL," +
		"`producer_address` varchar(42) NOT NULL DEFAULT ''," +
		"`active` tinyint(1) NOT NULL DEFAULT '0'," +
		"`producer_name` varchar(42) NOT NULL DEFAULT ''," +
		"`rank` tinyint(3) unsigned NOT NULL DEFAULT '0'," +
		"`blocks` int(11) unsigned NOT NULL DEFAULT '0'," +
		"`votes` decimal(42,0) unsigned NOT NULL DEFAULT '0'," +
		"`probated` tinyint(1) unsigned NOT NULL DEFAULT '0'," +
		"PRIMARY KEY (`id`)," +
		"KEY `block_height` (`block_height`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrap(err, "failed to start plugin")
	}

	dao, ok := kernel.GetBlockDAOCtx(ctx)
	if !ok {
		return errors.New("failed to get blockDAO in ctx")
	}
	b.dao = dao
	daoHeight, _ := b.dao.Height()
	if err := kernel.Transaction(func(tx *sql.Tx) error {
		return kernel.UpdateIndexHeight(tx, b.Name(), daoHeight)
	}); err != nil {
		return err
	}
	if err := delegates(); err != nil {
		log.L().Warn("failed to delegates", zap.Error(err))
	}
	go func() {
		ticker := time.NewTicker(time.Hour * 24)
		defer ticker.Stop()
		for {
			select {
			case <-b.stop:
				return
			case <-ticker.C:
				if err := delegates(); err != nil {
					log.L().Warn("failed to delegates", zap.Error(err))
				}
			}
		}
	}()
	return nil
}

func (b delegatesWorkerPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return nil
}

func (b delegatesWorkerPlugin) Stop(ctx context.Context) error {
	b.once.Do(func() {
		b.stop <- true
	})
	return nil
}

func (b delegatesWorkerPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = delegatesWorkerPlugin{
	once:      new(sync.Once),
	tableName: TableName,
	stop:      make(chan bool, 1),
}
