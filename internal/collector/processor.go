package collector

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/datacollector/datacollector/internal/webhook"
)

// StatEvent 统计事件
type StatEvent struct {
	SourceID int64
}

// Processor 数据处理器
type Processor struct {
	store       storage.DataStore
	statChan    chan<- StatEvent            // 通知监控模块的 channel
	webhookChan chan<- webhook.WebhookEvent // 通知 webhook 分发器的 channel
}

// NewProcessor 创建新的数据处理器
func NewProcessor(store storage.DataStore, statChan chan<- StatEvent, webhookChan chan<- webhook.WebhookEvent) *Processor {
	return &Processor{
		store:       store,
		statChan:    statChan,
		webhookChan: webhookChan,
	}
}

// ProcessRecord 处理单条数据记录
// 1. 写入 data_records 表
// 2. 发送统计事件到 channel
// 3. 如果数据源启用了 webhook，发送 webhook 事件
// 返回记录 ID
func (p *Processor) ProcessRecord(ctx context.Context, source *model.DataSource, record *model.DataRecord) (int64, error) {
	// 写入数据记录
	recordID, err := p.store.CreateRecord(ctx, record)
	if err != nil {
		return 0, fmt.Errorf("failed to create record: %w", err)
	}

	// 发送统计事件（如果 channel 不为 nil）
	if p.statChan != nil {
		select {
		case p.statChan <- StatEvent{SourceID: record.SourceID}:
			// 成功发送
		default:
			// channel 已满或阻塞，跳过发送（避免阻塞主流程）
		}
	}

	// 发送 webhook 事件（非阻塞）
	if p.webhookChan != nil && source != nil && source.WebhookEnabled && source.WebhookConfig != nil {
		select {
		case p.webhookChan <- webhook.WebhookEvent{
			SourceID:   source.ID,
			SourceName: source.Name,
			CollectID:  source.CollectID,
			RecordID:   recordID,
			Data:       record.Data,
			Config:     source.WebhookConfig,
		}:
		default:
			slog.Warn("webhook channel full, dropping event", "source_id", source.ID, "record_id", recordID)
		}
	}

	return recordID, nil
}

// ProcessBatch 处理批量数据记录
// 逐条写入，统计成功/失败数
// 返回 succeeded, failed, record_ids, error
func (p *Processor) ProcessBatch(ctx context.Context, source *model.DataSource, records []*model.DataRecord) (int, int, []int64, error) {
	succeeded := 0
	failed := 0
	recordIDs := make([]int64, 0, len(records))

	var lastErr error

	for _, record := range records {
		recordID, err := p.ProcessRecord(ctx, source, record)
		if err != nil {
			failed++
			lastErr = err
			recordIDs = append(recordIDs, 0) // 占位，表示失败
		} else {
			succeeded++
			recordIDs = append(recordIDs, recordID)
		}
	}

	// 如果全部失败，返回错误
	if succeeded == 0 && failed > 0 {
		return 0, failed, recordIDs, lastErr
	}

	// 部分成功或全部成功
	return succeeded, failed, recordIDs, nil
}
