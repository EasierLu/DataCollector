package storage

import (
	"fmt"

	"github.com/datacollector/datacollector/internal/config"
	"github.com/datacollector/datacollector/internal/storage/postgres"
	"github.com/datacollector/datacollector/internal/storage/sqlite"
)

// NewDataStore 根据配置创建对应的数据存储实现
func NewDataStore(cfg *config.Config) (DataStore, error) {
	switch cfg.Database.Driver {
	case "sqlite":
		return sqlite.New(cfg)
	case "postgres":
		return postgres.New(cfg)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Database.Driver)
	}
}
