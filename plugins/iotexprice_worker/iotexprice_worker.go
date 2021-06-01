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
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const VERSION = "2.0.0"

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
	if err := db.DB().AutoMigrate(&db.Store{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}

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
	goPrice1d := func() {
		price1d, err := price1dFetcher()
		if err != nil {
			log.L().Error("failed to fetch latest IOTX 1d price", zap.Error(err))
		} else {
			raw, _ := json.Marshal(price1d)
			store := &db.Store{
				Key:   "iotx_latest_price_1d",
				Value: string(raw),
			}
			if err := store.Save(); err != nil {
				log.L().Error("failed to exec query", zap.Error(err))
			}
		}
	}

	go func() {
		goPrice()
		goPrice1d()

		ticker := time.NewTicker(time.Minute * 5)
		ticker1d := time.NewTicker(time.Hour * 1)
		defer func() {
			ticker.Stop()
			ticker1d.Stop()
		}()
		for {
			select {
			case <-b.stop:
				return
			case <-ticker1d.C:
				goPrice1d()
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
