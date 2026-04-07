package model

import (
	"encoding/json"
	"time"
)

// DataRecord 数据记录模型
type DataRecord struct {
	ID        int64           `json:"id"`
	SourceID  int64           `json:"source_id"`
	TokenID   int64           `json:"token_id"`
	Data      json.RawMessage `json:"data"`
	IPAddress string          `json:"ip_address"`
	UserAgent string          `json:"user_agent"`
	CreatedAt time.Time       `json:"created_at"`
}

// RecordFilter 分页查询参数
type RecordFilter struct {
	SourceID  int64  `form:"source_id"`
	StartDate string `form:"start_date"` // 2024-01-01 格式
	EndDate   string `form:"end_date"`
	Page      int    `form:"page"`
	Size      int    `form:"size"`
}

// PageResult 分页结果
type PageResult struct {
	Total int64       `json:"total"`
	List  interface{} `json:"list"`
}
