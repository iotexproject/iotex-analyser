package main

import (
	"context"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const VERSION = "1.0.1"

type priceWorkerPlugin struct {
	stop chan bool
	once *sync.Once
}

func (b priceWorkerPlugin) Name() string {
	return "price_worker"
}

func (b priceWorkerPlugin) Type() plugin.Type {
	return plugin.TypeWorker
}

func (b priceWorkerPlugin) Start(ctx context.Context) error {
	createSql := "CREATE TABLE IF NOT EXISTS `store` (" +
		"`key` varchar(64) NOT NULL," +
		"`value` json NOT NULL," +
		"`create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"`update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP," +
		"PRIMARY KEY (`key`) USING BTREE" +
		") ENGINE=InnoDB DEFAULT CHARSET=latin1;"
	if _, err := kernel.GetDB().Exec(createSql); err != nil {
		return errors.Wrap(err, "failed to start plugin")
	}

	go func() {
		ticker := time.NewTicker(time.Second * 120)
		defer ticker.Stop()
		for {
			select {
			case <-b.stop:
				return
			case <-ticker.C:
				price, err := priceFetcher()
				if err != nil {
					log.L().Error("failed to fetch latest IOTX price", zap.Error(err))
				} else {
					if _, err := kernel.GetDB().Exec("INSERT INTO store (`key`, `value`, `create_at`) VALUES (?, ?, CURRENT_TIMESTAMP) ON DUPLICATE KEY UPDATE `value` = ?", "iotx_latest_price", price, price); err != nil {
						log.L().Error("failed to exec query", zap.Error(err))
					}
				}
			}
		}
	}()
	return nil
}

func (b priceWorkerPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return nil
}

func (b priceWorkerPlugin) Stop(ctx context.Context) error {
	b.once.Do(func() {
		b.stop <- true
	})
	return nil
}

func (b priceWorkerPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = priceWorkerPlugin{
	once: new(sync.Once),
	stop: make(chan bool, 1),
}
