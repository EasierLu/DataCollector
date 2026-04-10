package webhook

import (
	"log/slog"
	"time"
)

// Dispatcher Webhook事件分发器
type Dispatcher struct {
	eventChan chan WebhookEvent
	client    *Client
	quit      chan struct{}
}

// NewDispatcher 创建分发器，chanSize为事件channel缓冲大小（建议1000）
func NewDispatcher(chanSize int) *Dispatcher {
	return &Dispatcher{
		eventChan: make(chan WebhookEvent, chanSize),
		client:    NewClient(),
		quit:      make(chan struct{}),
	}
}

// EventChan 返回事件channel，供processor写入
func (d *Dispatcher) EventChan() chan<- WebhookEvent {
	return d.eventChan
}

// Start 启动分发器，监听事件channel并处理
// 每个事件启动一个goroutine处理（含重试逻辑）
func (d *Dispatcher) Start() {
	go func() {
		for {
			select {
			case event, ok := <-d.eventChan:
				if !ok {
					return
				}
				go d.handleEvent(event)
			case <-d.quit:
				return
			}
		}
	}()
}

// Stop 优雅停止分发器
func (d *Dispatcher) Stop() {
	close(d.quit)
}

// handleEvent 处理单个事件（在goroutine中运行）
// 1. 构建payload（如有BodyTemplate则渲染模板，否则用默认Payload）
// 2. 调用client.Send
// 3. 如果失败，按Config.RetryCount和Config.RetryInterval重试
// 4. 每次重试之间等待 RetryInterval 秒（默认5秒）
// 5. 所有重试失败后记录slog.Error日志
func (d *Dispatcher) handleEvent(event WebhookEvent) {
	config := event.Config
	if config == nil {
		slog.Error("webhook config is nil",
			"source_id", event.SourceID,
			"record_id", event.RecordID,
		)
		return
	}

	// 构建请求体
	var body []byte
	var err error
	if config.BodyTemplate != "" {
		body, err = RenderTemplate(config.BodyTemplate, event)
	} else {
		body, err = BuildPayload(event)
	}
	if err != nil {
		slog.Error("failed to build webhook payload",
			"error", err,
			"source_id", event.SourceID,
			"record_id", event.RecordID,
		)
		return
	}

	// 计算重试间隔
	retryInterval := config.RetryInterval
	if retryInterval <= 0 {
		retryInterval = 5
	}

	// 首次发送 + 重试
	maxAttempts := 1 + config.RetryCount
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		statusCode, err := d.client.Send(config.URL, body, config)
		if err == nil && statusCode >= 200 && statusCode < 300 {
			slog.Debug("webhook sent successfully",
				"source_id", event.SourceID,
				"record_id", event.RecordID,
				"status_code", statusCode,
				"attempt", attempt,
			)
			return
		}

		// 记录失败信息
		if err != nil {
			slog.Warn("webhook send failed",
				"error", err,
				"source_id", event.SourceID,
				"record_id", event.RecordID,
				"attempt", attempt,
				"max_attempts", maxAttempts,
			)
		} else {
			slog.Warn("webhook returned non-2xx status",
				"status_code", statusCode,
				"source_id", event.SourceID,
				"record_id", event.RecordID,
				"attempt", attempt,
				"max_attempts", maxAttempts,
			)
		}

		// 如果还有重试机会，等待后重试
		if attempt < maxAttempts {
			time.Sleep(time.Duration(retryInterval) * time.Second)
		}
	}

	// 所有重试均失败
	slog.Error("webhook delivery failed after all retries",
		"source_id", event.SourceID,
		"record_id", event.RecordID,
		"url", config.URL,
		"max_attempts", maxAttempts,
	)
}
