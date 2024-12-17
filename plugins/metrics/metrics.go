package main

import (
	"context"
	"database/sql"
	"strconv"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const VERSION = "0.0.1"

var lock sync.RWMutex

type metricsPlugin struct {
	prefix   string
	interval int
	gauges   map[string]prometheus.Gauge
	counters map[string]prometheus.Counter
}

func (b metricsPlugin) Name() string {
	return "metrics"
}

func (b metricsPlugin) Type() plugin.Type {
	return plugin.TypeWorker
}

func (b metricsPlugin) Start(ctx context.Context) error {
	funM := []func(*sync.WaitGroup){
		b.updateBlockMetrics,
		b.updateBlockGasPrice,
		b.updatePriorityFee,
	}

	go func() {
		for range time.Tick(time.Duration(b.interval) * time.Second) {
			var wg sync.WaitGroup
			for _, f := range funM {
				wg.Add(1)
				go f(&wg)
			}
			wg.Wait()
		}
	}()
	return nil
}

func (b metricsPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return nil
}

func (b metricsPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b metricsPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = metricsPlugin{
	prefix:   "iotex_analyser_",
	interval: 10,
	gauges:   make(map[string]prometheus.Gauge),
	counters: make(map[string]prometheus.Counter),
}

func (m metricsPlugin) getGauge(identifier string) (prometheus.Gauge, bool) {
	lock.Lock()
	defer lock.Unlock()
	g, ok := m.gauges[identifier]
	return g, ok
}

func (m metricsPlugin) setGauge(identifier string, g prometheus.Gauge) {
	lock.Lock()
	defer lock.Unlock()
	m.gauges[identifier] = g
}

func (m metricsPlugin) getCounter(identifier string) (prometheus.Counter, bool) {
	lock.Lock()
	defer lock.Unlock()
	c, ok := m.counters[identifier]
	return c, ok
}

func (m metricsPlugin) setCounter(identifier string, c prometheus.Counter) {
	lock.Lock()
	defer lock.Unlock()
	m.counters[identifier] = c
}

func (m metricsPlugin) updateBlockMetrics(wg *sync.WaitGroup) {
	defer wg.Done()
	metric := "block_gas_consumed"
	blkHeight, err := db.GetIndexHeight("block_meta")
	if err != nil {
		return
	}
	var dbBlockMeta models.BlockMeta
	if err := db.DB().Model(dbBlockMeta).Where("block_height = ?", blkHeight).First(&dbBlockMeta).Error; err != nil {
		log.L().Error("failed to get block meta", zap.Error(err))
		return
	}
	gauge, ok := m.getGauge(metric)
	if !ok {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: m.prefix + metric,
			Help: "block gas consumed",
		})

		m.setGauge(metric, gauge)
		prometheus.Register(gauge)
	}
	log.L().Debug("metrics block gas consumed", zap.Uint64("height", blkHeight), zap.Uint64("gas_consumed", dbBlockMeta.GasConsumed))
	gauge.Set(float64(dbBlockMeta.GasConsumed))

	// update block base fee
	metric = "block_base_fee"
	gauge, ok = m.getGauge(metric)
	if !ok {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: m.prefix + metric,
			Help: "block base fee",
		})

		m.setGauge(metric, gauge)
		prometheus.Register(gauge)
	}
	log.L().Debug("metrics block base fee", zap.Uint64("height", blkHeight), zap.Uint64("base_fee", dbBlockMeta.BaseFee.BigInt().Uint64()))
	gauge.Set(float64(dbBlockMeta.BaseFee.BigInt().Uint64()))

	// update block priority bonus
	metric = "block_priority_bonus"
	gauge, ok = m.getGauge(metric)
	if !ok {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: m.prefix + metric,
			Help: "block priority bonus",
		})

		m.setGauge(metric, gauge)
		prometheus.Register(gauge)
	}
	log.L().Debug("metrics block priority bonus", zap.Uint64("height", blkHeight), zap.Uint64("priority_bonus", dbBlockMeta.PriorityBonus.BigInt().Uint64()))
	gauge.Set(float64(dbBlockMeta.PriorityBonus.BigInt().Uint64()))
}

func (m metricsPlugin) updateBlockGasPrice(wg *sync.WaitGroup) {
	defer wg.Done()
	metric := "block_gas_price"
	blkHeight, err := db.GetIndexHeight("block_action")
	if err != nil {
		return
	}
	var gasprice sql.NullString
	if err := db.DB().Model(&models.BlockAction{}).Select("avg(gas_price)").Where("block_height = ?", blkHeight).Scan(&gasprice).Error; err != nil {
		log.L().Error("failed to get block action", zap.Error(err))
		return
	}
	if gasprice.String == "" {
		return
	}
	gasPrice, err := strconv.ParseFloat(gasprice.String, 64)
	if err != nil {
		log.L().Error("failed to parse gas price", zap.Error(err))
		return
	}

	gauge, ok := m.getGauge(metric)
	if !ok {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: m.prefix + metric,
			Help: "block gas price",
		})

		m.setGauge(metric, gauge)
		prometheus.Register(gauge)
	}
	log.L().Debug("metrics block gas price", zap.Uint64("height", blkHeight), zap.Float64("gas_price", gasPrice))
	gauge.Set(gasPrice)
}

func (m metricsPlugin) updatePriorityFee(wg *sync.WaitGroup) {
	defer wg.Done()
	metric := "priority_fee"
	blkHeight, err := db.GetIndexHeight("action_type")
	if err != nil {
		return
	}
	var gasTipCap sql.NullString
	if err := db.DB().Model(&models.ActionType{}).Select("avg(gas_tip_cap)").Where("block_height = ?", blkHeight).Scan(&gasTipCap).Error; err != nil {
		log.L().Error("failed to get action type", zap.Error(err))
		return
	}
	if gasTipCap.String == "" {
		return
	}
	gasTip, err := strconv.ParseFloat(gasTipCap.String, 64)
	if err != nil {
		log.L().Error("failed to parse gas tip cap", zap.Error(err))
		return
	}

	gauge, ok := m.getGauge(metric)
	if !ok {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: m.prefix + metric,
			Help: "gas tip cap",
		})

		m.setGauge(metric, gauge)
		prometheus.Register(gauge)
	}
	log.L().Debug("metrics gas tip price", zap.Uint64("height", blkHeight), zap.Float64("gas_tip_cap", gasTip))
	gauge.Set(gasTip)
}
