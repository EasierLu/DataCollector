package model

import "time"

// DataToken 数据Token模型
type DataToken struct {
	ID         int64      `json:"id"`
	SourceID   int64      `json:"source_id"`
	TokenHash  string     `json:"-"`          // 哈希值，不暴露
	Name       string     `json:"name"`
	Status     int        `json:"status"`     // 0:禁用, 1:启用
	ExpiresAt  *time.Time `json:"expires_at"` // nil = 永不过期
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedBy  int64      `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
}
