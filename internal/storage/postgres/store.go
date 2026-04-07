package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/datacollector/datacollector/internal/config"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/datacollector/datacollector/internal/storage/migrations"
)

// PostgresStore PostgreSQL 数据存储实现
type PostgresStore struct {
	db *sql.DB
}

// 编译时检查接口实现
var _ storage.DataStore = (*PostgresStore)(nil)

// New 创建 PostgreSQL 存储实例
func New(cfg *config.Config) (*PostgresStore, error) {
	dsn := cfg.Database.DSN()

	// 打开数据库连接
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return &PostgresStore{db: db}, nil
}

// Init 初始化数据库（执行迁移）
func (s *PostgresStore) Init(ctx context.Context) error {
	// 读取迁移文件
	sqlBytes, err := migrations.FS.ReadFile("001_init_postgres.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// 执行迁移
	if _, err := s.db.ExecContext(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

// Close 关闭数据库连接
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// Ping 测试数据库连接
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
