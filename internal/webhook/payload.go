package webhook

import (
	"bytes"
	"encoding/json"
	"text/template"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// WebhookEvent 从processor发送到dispatcher的内部事件
type WebhookEvent struct {
	SourceID   int64
	SourceName string
	CollectID  string
	RecordID   int64
	Data       json.RawMessage
	Config     *model.WebhookConfig
}

// WebhookPayload 发送给外部的默认JSON载荷
type WebhookPayload struct {
	Event      string          `json:"event"`
	SourceID   int64           `json:"source_id"`
	SourceName string          `json:"source_name"`
	CollectID  string          `json:"collect_id"`
	RecordID   int64           `json:"record_id"`
	Data       json.RawMessage `json:"data"`
	Timestamp  string          `json:"timestamp"`
}

// BuildPayload 构建默认Payload的JSON字节
func BuildPayload(event WebhookEvent) ([]byte, error) {
	payload := WebhookPayload{
		Event:      "data.created",
		SourceID:   event.SourceID,
		SourceName: event.SourceName,
		CollectID:  event.CollectID,
		RecordID:   event.RecordID,
		Data:       event.Data,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	return json.Marshal(payload)
}

// templateData 模板渲染时可用的变量
type templateData struct {
	Event      string
	SourceID   int64
	SourceName string
	CollectID  string
	RecordID   int64
	Data       string
	Timestamp  string
}

// RenderTemplate 使用Go text/template渲染自定义请求体模板
// 可用变量: .Event, .SourceID, .SourceName, .CollectID, .RecordID, .Data, .Timestamp
func RenderTemplate(tmpl string, event WebhookEvent) ([]byte, error) {
	t, err := template.New("webhook").Parse(tmpl)
	if err != nil {
		return nil, err
	}

	data := templateData{
		Event:      "data.created",
		SourceID:   event.SourceID,
		SourceName: event.SourceName,
		CollectID:  event.CollectID,
		RecordID:   event.RecordID,
		Data:       string(event.Data),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
