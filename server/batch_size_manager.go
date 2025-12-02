package server

import (
	"sync"

	"go.uber.org/zap"
)

const (
	batchSizeIncreaseThreshold        = 10  // 连续成功10次后尝试增加批次大小
	batchSizeIncreaseStep             = 1.2 // 每次增加20%
	batchSizeDecreaseRatio            = 0.5 // 失败时减半
	minBatchSize               uint64 = 1   // 最小批次大小
)

// batchSizeManager 管理批次大小的自适应调整
type batchSizeManager struct {
	mu                   sync.Mutex
	effectiveSize        uint64 // 当前有效的批次大小
	maxSize              uint64 // 最大批次大小
	consecutiveSuccesses int    // 连续成功次数
	pluginName           string // 用于日志记录
	logger               *zap.Logger
}

// newBatchSizeManager 创建一个新的批次大小管理器
func newBatchSizeManager(initialSize uint64, pluginName string, logger *zap.Logger) *batchSizeManager {
	return &batchSizeManager{
		effectiveSize:        initialSize,
		maxSize:              initialSize,
		consecutiveSuccesses: 0,
		pluginName:           pluginName,
		logger:               logger,
	}
}

// getCurrent 获取当前应使用的批次大小
func (m *batchSizeManager) getCurrent() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.effectiveSize
}

// onSuccess 在成功获取批次后调用，尝试增加批次大小
func (m *batchSizeManager) onSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.consecutiveSuccesses++
	if m.consecutiveSuccesses >= batchSizeIncreaseThreshold {
		oldSize := m.effectiveSize
		// 尝试增加批次大小
		newSize := uint64(float64(m.effectiveSize) * batchSizeIncreaseStep)
		if newSize > m.maxSize {
			newSize = m.maxSize
		}
		if newSize > oldSize {
			m.effectiveSize = newSize
			m.consecutiveSuccesses = 0 // 重置计数器
			m.logger.Info("increasing batch size",
				zap.String("pluginName", m.pluginName),
				zap.Uint64("oldSize", oldSize),
				zap.Uint64("newSize", newSize),
			)
		}
	}
}

// onFailure 在获取批次失败后调用，减小批次大小并返回新的批次大小
func (m *batchSizeManager) onFailure(failedSize uint64) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.consecutiveSuccesses = 0 // 重置成功计数器
	newSize := uint64(float64(failedSize) * batchSizeDecreaseRatio)
	if newSize < minBatchSize {
		newSize = minBatchSize
	}
	m.effectiveSize = newSize
	m.logger.Info("decreasing batch size due to error",
		zap.String("pluginName", m.pluginName),
		zap.Uint64("oldSize", failedSize),
		zap.Uint64("newSize", newSize),
	)
	return newSize
}
