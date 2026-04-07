package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/datacollector/datacollector/internal/collector"
	"github.com/datacollector/datacollector/internal/storage"
)

// StatEvent 统计事件（使用 collector 包的定义）
type StatEvent = collector.StatEvent

// Aggregator 统计聚合器
type Aggregator struct {
	store    storage.DataStore
	logger   *slog.Logger
	eventCh  chan StatEvent
	hub      *WebSocketHub // WebSocket 推送中心

	mu       sync.Mutex
	counters map[int64]int64 // sourceID -> count（内存中的增量）

	stopCh chan struct{}
}

// NewAggregator 创建新的统计聚合器
func NewAggregator(store storage.DataStore, hub *WebSocketHub, logger *slog.Logger) *Aggregator {
	return &Aggregator{
		store:    store,
		hub:      hub,
		logger:   logger,
		eventCh:  make(chan StatEvent, 1000),
		counters: make(map[int64]int64),
		stopCh:   make(chan struct{}),
	}
}

// EventChannel 返回事件 channel，供 collector.Processor 使用
func (a *Aggregator) EventChannel() chan<- StatEvent {
	return a.eventCh
}

// Start 启动聚合器后台 goroutine
func (a *Aggregator) Start(ctx context.Context) {
	go a.run(ctx)
}

// run 聚合器主循环
func (a *Aggregator) run(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-a.eventCh:
			a.handleEvent(ctx, event)

		case <-ticker.C:
			a.flush(ctx)

		case <-a.stopCh:
			a.flush(ctx)
			return

		case <-ctx.Done():
			a.flush(ctx)
			return
		}
	}
}

// handleEvent 处理单个统计事件
func (a *Aggregator) handleEvent(ctx context.Context, event StatEvent) {
	a.mu.Lock()
	a.counters[event.SourceID]++
	a.mu.Unlock()
	// 不在此处推送 WebSocket 消息，等待 flush 完成后再统一推送
}

// Stop 停止聚合器并执行最后一次持久化
func (a *Aggregator) Stop() {
	close(a.stopCh)
}

// flush 将内存中的计数器持久化到数据库
func (a *Aggregator) flush(ctx context.Context) {
	a.mu.Lock()
	if len(a.counters) == 0 {
		a.mu.Unlock()
		return
	}

	// 复制计数器数据并清空
	countersToFlush := make(map[int64]int64, len(a.counters))
	for sourceID, count := range a.counters {
		countersToFlush[sourceID] = count
	}
	// 清空计数器
	for sourceID := range a.counters {
		delete(a.counters, sourceID)
	}
	a.mu.Unlock()

	// 获取今天的日期
	today := time.Now().Format("2006-01-02")

	// 持久化到数据库
	for sourceID, count := range countersToFlush {
		// 调用多次 IncrementStatCount 来增加计数
		for i := int64(0); i < count; i++ {
			if err := a.store.IncrementStatCount(ctx, sourceID, today); err != nil {
				a.logger.Error("failed to increment stat count",
					"error", err,
					"source_id", sourceID,
					"date", today,
				)
				// 只记录日志，不中断运行
				break
			}
		}
	}

	a.logger.Debug("flushed counters to database", "count", len(countersToFlush), "date", today)

	// 数据已持久化到数据库，通知 WebSocket Hub 推送更新
	if a.hub != nil {
		a.hub.BroadcastStatsUpdate()
	}
}

// GetTodayCount 获取今天某个数据源的统计计数（内存中的计数）
func (a *Aggregator) GetTodayCount(sourceID int64) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.counters[sourceID]
}

// GetAllCounters 获取所有计数器的副本（用于调试）
func (a *Aggregator) GetAllCounters() map[int64]int64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	result := make(map[int64]int64, len(a.counters))
	for k, v := range a.counters {
		result[k] = v
	}
	return result
}

// ForceFlush 强制立即持久化（主要用于测试）
func (a *Aggregator) ForceFlush(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	a.mu.Lock()
	if len(a.counters) == 0 {
		a.mu.Unlock()
		return nil
	}

	countersToFlush := make(map[int64]int64, len(a.counters))
	for sourceID, count := range a.counters {
		countersToFlush[sourceID] = count
	}
	for sourceID := range a.counters {
		delete(a.counters, sourceID)
	}
	a.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	var lastErr error

	for sourceID, count := range countersToFlush {
		for i := int64(0); i < count; i++ {
			if err := a.store.IncrementStatCount(ctx, sourceID, today); err != nil {
				lastErr = err
				a.logger.Error("failed to increment stat count",
					"error", err,
					"source_id", sourceID,
					"date", today,
				)
				break
			}
		}
	}

	if lastErr != nil {
		return fmt.Errorf("flush failed: %w", lastErr)
	}
	return nil
}
