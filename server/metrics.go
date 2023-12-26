package server

import (
	"database/sql"
	"strconv"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TODO: move metric to a plugin
type metrics struct {
	Prefix        string
	Interval      int
	db            *gorm.DB
	gauges        map[string]prometheus.Gauge
	counters      map[string]prometheus.Counter
	blockGasPrice prometheus.Gauge
	lock          sync.RWMutex
}

func newMetrics(db *gorm.DB) *metrics {
	m := &metrics{
		Prefix:   "iotex_analyser_",
		Interval: 10,
		db:       db,
		gauges:   make(map[string]prometheus.Gauge),
		counters: make(map[string]prometheus.Counter),
		lock:     sync.RWMutex{},
	}

	return m
}

func (mt *metrics) Start() {
	funM := []func(*sync.WaitGroup){
		mt.updateBlockGasConsumed,
		mt.updateBlockGasPrice,
	}

	go func() {
		for range time.Tick(time.Duration(mt.Interval) * time.Second) {
			var wg sync.WaitGroup
			for _, f := range funM {
				wg.Add(1)
				go f(&wg)
			}
			wg.Wait()
		}
	}()
}

func (m *metrics) getGauge(identifier string) (prometheus.Gauge, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	g, ok := m.gauges[identifier]
	return g, ok
}

func (m *metrics) setGauge(identifier string, g prometheus.Gauge) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.gauges[identifier] = g
}

func (m *metrics) getCounter(identifier string) (prometheus.Counter, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	c, ok := m.counters[identifier]
	return c, ok
}

func (m *metrics) setCounter(identifier string, c prometheus.Counter) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.counters[identifier] = c
}

func (m *metrics) updateBlockGasConsumed(wg *sync.WaitGroup) {
	defer wg.Done()
	metric := "block_gas_consumed"
	blkHeight, err := db.GetIndexHeight("block_meta")
	if err != nil {
		return
	}
	var dbBlockMeta models.BlockMeta
	if err := m.db.Model(dbBlockMeta).Where("block_height = ?", blkHeight).First(&dbBlockMeta).Error; err != nil {
		log.L().Error("failed to get block meta", zap.Error(err))
		return
	}
	gauge, ok := m.getGauge(metric)
	if !ok {
		gauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: m.Prefix + metric,
			Help: "block gas consumed",
		})

		m.setGauge(metric, gauge)
		prometheus.Register(gauge)
	}
	log.L().Debug("metrics block gas consumed", zap.Uint64("height", blkHeight), zap.Uint64("gas_consumed", dbBlockMeta.GasConsumed))
	gauge.Set(float64(dbBlockMeta.GasConsumed))
}

func (m *metrics) updateBlockGasPrice(wg *sync.WaitGroup) {
	defer wg.Done()
	metric := "block_gas_price"
	blkHeight, err := db.GetIndexHeight("block_action")
	if err != nil {
		return
	}
	var gasprice sql.NullString
	if err := m.db.Model(&models.BlockAction{}).Select("avg(gas_price)").Where("block_height = ?", blkHeight).Scan(&gasprice).Error; err != nil {
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
			Name: m.Prefix + metric,
			Help: "block gas price",
		})

		m.setGauge(metric, gauge)
		prometheus.Register(gauge)
	}
	log.L().Debug("metrics block gas price", zap.Uint64("height", blkHeight), zap.Float64("gas_price", gasPrice))
	gauge.Set(gasPrice)
}
