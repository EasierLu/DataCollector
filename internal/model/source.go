package model

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"time"
)

// WebhookConfig Webhook回调配置
type WebhookConfig struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	Secret        string            `json:"secret"`
	Timeout       int               `json:"timeout"`
	RetryCount    int               `json:"retry_count"`
	RetryInterval int               `json:"retry_interval"`
	BodyTemplate  string            `json:"body_template"`
}

// DataSource 数据源模型
type DataSource struct {
	ID             int64           `json:"id"`
	CollectID      string          `json:"collect_id"` // 采集短标识，用于 collect API 路由
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	SchemaConfig   json.RawMessage `json:"schema_config"` // JSON 格式字段定义
	Status         int             `json:"status"`        // 0:禁用, 1:启用
	CreatedBy      int64           `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	TokenCount     int             `json:"token_count,omitempty"` // 关联的 Token 数量（查询时填充）
	RateLimit      int             `json:"rate_limit"`            // 每分钟请求数，0=使用全局默认
	RateLimitBurst int             `json:"rate_limit_burst"`      // 突发量上限，0=使用全局默认
	WebhookEnabled bool            `json:"webhook_enabled"`
	WebhookConfig  *WebhookConfig  `json:"webhook_config,omitempty"`
}

// GenerateCollectID 生成短随机标识符，用于采集 API 路由
func GenerateCollectID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 8
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// SchemaField 字段定义
type SchemaField struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // string, number, email, url
	Required  bool   `json:"required"`
	MaxLength int    `json:"max_length,omitempty"`
	MinLength int    `json:"min_length,omitempty"`
	Pattern   string `json:"pattern,omitempty"` // 正则
}

// SchemaConfig 数据源配置结构
type SchemaConfig struct {
	Fields []SchemaField `json:"fields"`
}
