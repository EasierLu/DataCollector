package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/datacollector/datacollector/internal/config"
	"github.com/datacollector/datacollector/internal/storage/migrations"
)

// PostgresStore PostgreSQL 数据存储实现
type PostgresStore struct {
	db *sql.DB
}

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
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	return &PostgresStore{db: db}, nil
}

// Init 初始化数据库（执行迁移）
func (s *PostgresStore) Init(ctx context.Context) error {
	// 读取并执行初始化迁移
	sqlBytes, err := migrations.FS.ReadFile("001_init_postgres.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

// ResetAllData 清除所有业务数据（用于重新初始化）
func (s *PostgresStore) ResetAllData(ctx context.Context) error {
	// 使用 TRUNCATE CASCADE 一次性清空所有表
	_, err := s.db.ExecContext(ctx, `TRUNCATE TABLE data_records, statistics, data_tokens, api_keys, data_sources, users, system_configs RESTART IDENTITY CASCADE`)
	if err != nil {
		return fmt.Errorf("failed to truncate tables: %w", err)
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
