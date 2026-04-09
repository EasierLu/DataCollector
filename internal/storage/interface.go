package storage

import (
	"context"

	"github.com/datacollector/datacollector/internal/model"
)

// DataStore 定义数据存储接口
type DataStore interface {
	// 初始化和迁移
	Init(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error

	// 用户管理
	CreateUser(ctx context.Context, user *model.User) (int64, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByID(ctx context.Context, id int64) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error

	// 数据源管理
	CreateSource(ctx context.Context, source *model.DataSource) (int64, error)
	GetSourceByID(ctx context.Context, id int64) (*model.DataSource, error)
	GetSourceByCollectID(ctx context.Context, collectID string) (*model.DataSource, error)
	ListSources(ctx context.Context, page, size int) (*model.PageResult, error)
	UpdateSource(ctx context.Context, source *model.DataSource) error
	DeleteSource(ctx context.Context, id int64) error // 软删除

	// Token 管理
	CreateToken(ctx context.Context, token *model.DataToken) (int64, error)
	GetTokenByHash(ctx context.Context, hash string) (*model.DataToken, error)
	ListTokensBySourceID(ctx context.Context, sourceID int64) ([]*model.DataToken, error)
	UpdateTokenStatus(ctx context.Context, id int64, status int) error
	UpdateTokenLastUsed(ctx context.Context, id int64) error
	DeleteToken(ctx context.Context, id int64) error

	// API Key 管理（独立于 Data Token，用于数据查询等操作）
	CreateApiKey(ctx context.Context, apiKey *model.ApiKey) (int64, error)
	GetApiKeyByHash(ctx context.Context, hash string) (*model.ApiKey, error)
	ListApiKeys(ctx context.Context) ([]*model.ApiKey, error)
	UpdateApiKeyPermissions(ctx context.Context, id int64, permissions string) error
	UpdateApiKeyLastUsed(ctx context.Context, id int64) error
	DeleteApiKey(ctx context.Context, id int64) error

	// 数据记录
	CreateRecord(ctx context.Context, record *model.DataRecord) (int64, error)
	GetRecordByID(ctx context.Context, id int64) (*model.DataRecord, error)
	GetLastRecordBySourceID(ctx context.Context, sourceID int64) (*model.DataRecord, error)
	GetRecordsByIDRange(ctx context.Context, sourceID int64, startID, endID int64) ([]*model.DataRecord, error)
	QueryRecords(ctx context.Context, filter model.RecordFilter) (*model.PageResult, error)
	DeleteRecord(ctx context.Context, id int64) error
	DeleteRecordsByIDs(ctx context.Context, ids []int64) (int64, error)
	ExportRecords(ctx context.Context, filter model.RecordFilter) ([]*model.DataRecord, error)

	// 统计
	IncrementStatCount(ctx context.Context, sourceID int64, date string) error
	IncrementStatCountBy(ctx context.Context, sourceID int64, date string, count int64) error
	GetStatsBySourceAndDateRange(ctx context.Context, sourceID int64, startDate, endDate string) ([]*model.Statistics, error)
	GetTotalCountByDateRange(ctx context.Context, startDate, endDate string) (int64, error)
	GetCountBySourceID(ctx context.Context, sourceID int64) (int64, error)
	GetDailyTrend(ctx context.Context, startDate, endDate string, sourceID, tokenID int64) ([]*model.TrendPoint, error)

	// 系统配置
	GetConfig(ctx context.Context, key string) (string, error)
	SetConfig(ctx context.Context, key, value string) error
	GetAllConfigs(ctx context.Context) ([]*model.SystemConfig, error)

	// 系统重置
	ResetAllData(ctx context.Context) error
}
