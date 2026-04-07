package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"

	"github.com/datacollector/datacollector/internal/config"
	"github.com/datacollector/datacollector/internal/storage/migrations"
)

// SQLiteStore SQLite 数据存储实现
type SQLiteStore struct {
	db *sql.DB
	mu sync.Mutex
}

// New 创建 SQLite 存储实例
func New(cfg *config.Config) (*SQLiteStore, error) {
	dbPath := cfg.Database.SQLite.Path

	// 确保数据库目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(1) // SQLite 只支持单写
	db.SetMaxIdleConns(1)

	// 启用 WAL 模式
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	// 设置 busy timeout
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// Init 初始化数据库（执行迁移）
func (s *SQLiteStore) Init(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 读取并执行初始化迁移
	sqlBytes, err := migrations.FS.ReadFile("001_init_sqlite.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// 执行增量迁移：添加 collect_id 字段
	// 先检查列是否已存在
	var colCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('data_sources') WHERE name='collect_id'").Scan(&colCount)
	if err != nil {
		return fmt.Errorf("failed to check collect_id column: %w", err)
	}
	if colCount == 0 {
		sqlBytes2, err := migrations.FS.ReadFile("002_add_collect_id_sqlite.sql")
		if err != nil {
			return fmt.Errorf("failed to read migration file 002: %w", err)
		}
		statements := splitSQL(string(sqlBytes2))
		for _, stmt := range statements {
			if stmt != "" {
				if _, err := s.db.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("failed to execute migration 002: %w", err)
				}
			}
		}
	}

	return nil
}

// splitSQL 将 SQL 文件按分号拆分为多条语句
func splitSQL(sql string) []string {
	var result []string
	for _, s := range strings.Split(sql, ";") {
		s = strings.TrimSpace(s)
		if s != "" && !strings.HasPrefix(s, "--") {
			result = append(result, s)
		}
	}
	return result
}

// ResetAllData 清除所有业务数据（用于重新初始化）
func (s *SQLiteStore) ResetAllData(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 按外键依赖顺序删除，子表先删
	tables := []string{
		"data_records",
		"statistics",
		"data_tokens",
		"data_sources",
		"users",
		"system_configs",
	}

	for _, table := range tables {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
	}

	return nil
}

// Close 关闭数据库连接
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Ping 测试数据库连接
func (s *SQLiteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
