package model

import "time"

// Statistics 统计数据模型
type Statistics struct {
	ID        int64     `json:"id"`
	SourceID  int64     `json:"source_id"`
	StatDate  string    `json:"stat_date"` // YYYY-MM-DD
	Count     int64     `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
