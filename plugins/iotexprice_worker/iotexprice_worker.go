package main

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/pkg/log"
	"go.uber.org/zap"
)

const VERSION = "2.2.0"

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
	goPrice := func() {
		price, err := priceFetcher()
		if err != nil {
			log.L().Error("failed to fetch latest IOTX price", zap.Error(err))
		} else {
			raw, _ := json.Marshal(price)

			store := &db.Store{
				Key:   "iotx_latest_price",
				Value: string(raw),
			}
			if err := store.Save(); err != nil {
				log.L().Error("failed to exec query", zap.Error(err))
			}
		}
	}

	go func() {
		goPrice()

		ticker := time.NewTicker(time.Minute * 5)
		defer func() {
			ticker.Stop()
		}()
		for {
			select {
			case <-b.stop:
				return
			case <-ticker.C:
				goPrice()
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
