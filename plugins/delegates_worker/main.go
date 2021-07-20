package main

import (
	"context"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const VERSION = "2.0.2"
const TableName = "node_delegates"

type delegatesWorkerPlugin struct {
	tableName string
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
	if err := db.DB().AutoMigrate(&NodeDelegates{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

	if err := delegates(); err != nil {
		log.L().Warn("failed to delegates", zap.Error(err))
	}
	go func() {
		ticker := time.NewTicker(time.Minute * 1)
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
