package main

import (
	"context"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const VERSION = "2.3.1"

type delegatePlugin struct {
	stop chan bool
	once *sync.Once
}

func (b delegatePlugin) Name() string {
	return "delegate"
}

func (b delegatePlugin) Type() plugin.Type {
	return plugin.TypeWorker
}

func (b delegatePlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(), &models.Delegate{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	if err := delegate(); err != nil {
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
				if err := delegate(); err != nil {
					log.L().Warn("failed to delegate", zap.Error(err))
				}
			}
		}
	}()
	return nil
}

func (b delegatePlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return nil
}

func (b delegatePlugin) Stop(ctx context.Context) error {
	b.once.Do(func() {
		b.stop <- true
	})
	return nil
}

func (b delegatePlugin) Version() string {
	return VERSION
}

// exported
var Plugin = delegatePlugin{
	once: new(sync.Once),
	stop: make(chan bool, 1),
}
