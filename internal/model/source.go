package model

import (
	"encoding/json"
	"time"
)

// DataSource 数据源模型
type DataSource struct {
	ID           int64           `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	SchemaConfig json.RawMessage `json:"schema_config"` // JSON 格式字段定义
	Status       int             `json:"status"`        // 0:禁用, 1:启用
	CreatedBy    int64           `json:"created_by"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	TokenCount   int             `json:"token_count,omitempty"` // 关联的 Token 数量（查询时填充）
}

// SchemaField 字段定义
type SchemaField struct {
	Name      string `json:"name"`
	Type      string `json:"type"`       // string, number, email, url
	Required  bool   `json:"required"`
	MaxLength int    `json:"max_length,omitempty"`
	MinLength int    `json:"min_length,omitempty"`
	Pattern   string `json:"pattern,omitempty"` // 正则
}

// SchemaConfig 数据源配置结构
type SchemaConfig struct {
	Fields []SchemaField `json:"fields"`
}
