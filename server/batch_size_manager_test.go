package server

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBatchSizeManager_GetCurrent(t *testing.T) {
	r := require.New(t)
	mgr := newBatchSizeManager(100, "test", zap.NewNop())
	r.Equal(uint64(100), mgr.getCurrent())
}

func TestBatchSizeManager_OnSuccess(t *testing.T) {
	r := require.New(t)
	mgr := newBatchSizeManager(100, "test", zap.NewNop())

	// 连续成功9次，不应增加
	for i := 0; i < 9; i++ {
		mgr.onSuccess()
	}
	r.Equal(uint64(100), mgr.getCurrent())

	// 第10次成功，应该增加到120，但受限于maxSize保持100
	mgr.onSuccess()
	r.Equal(uint64(100), mgr.getCurrent())
}

func TestBatchSizeManager_OnSuccessCanIncrease(t *testing.T) {
	r := require.New(t)
	// 先失败减小批次，然后测试能否增加
	mgr := newBatchSizeManager(100, "test", zap.NewNop())
	mgr.onFailure(100) // 减小到50
	r.Equal(uint64(50), mgr.getCurrent())

	// 连续成功10次，应该从50增加到60
	for i := 0; i < 10; i++ {
		mgr.onSuccess()
	}
	r.Equal(uint64(60), mgr.getCurrent())
}

func TestBatchSizeManager_OnFailure(t *testing.T) {
	r := require.New(t)
	mgr := newBatchSizeManager(100, "test", zap.NewNop())

	// 失败时应该减半到50
	newSize := mgr.onFailure(100)
	r.Equal(uint64(50), newSize)
	r.Equal(uint64(50), mgr.getCurrent())

	// 再次失败应该减半到25
	newSize = mgr.onFailure(50)
	r.Equal(uint64(25), newSize)

	// 失败到小于minBatchSize时，应该保持minBatchSize
	newSize = mgr.onFailure(1)
	r.Equal(minBatchSize, newSize)
	r.Equal(minBatchSize, mgr.getCurrent())
}

func TestBatchSizeManager_SuccessAfterFailure(t *testing.T) {
	r := require.New(t)
	mgr := newBatchSizeManager(100, "test", zap.NewNop())

	// 先失败，减小到50
	mgr.onFailure(100)
	r.Equal(uint64(50), mgr.getCurrent())

	// 连续成功10次，应该增加到60
	for i := 0; i < 10; i++ {
		mgr.onSuccess()
	}
	r.Equal(uint64(60), mgr.getCurrent())

	// 再连续成功10次，应该增加到72
	for i := 0; i < 10; i++ {
		mgr.onSuccess()
	}
	r.Equal(uint64(72), mgr.getCurrent())
}

func TestBatchSizeManager_Concurrent(t *testing.T) {
	r := require.New(t)
	mgr := newBatchSizeManager(100, "test", zap.NewNop())

	// 并发调用不应panic
	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				mgr.onSuccess()
			}
			done <- true
		}()
		go func() {
			for j := 0; j < 100; j++ {
				mgr.onFailure(mgr.getCurrent())
			}
			done <- true
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	// 验证最终状态合理
	current := mgr.getCurrent()
	r.GreaterOrEqual(current, minBatchSize)
	r.LessOrEqual(current, uint64(100))
}
