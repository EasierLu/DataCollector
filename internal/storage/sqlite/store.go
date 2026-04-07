package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"

	"github.com/datacollector/datacollector/internal/config"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/datacollector/datacollector/internal/storage/migrations"
)

// SQLiteStore SQLite 数据存储实现
type SQLiteStore struct {
	db *sql.DB
	mu sync.Mutex
}

// 编译时检查接口实现
var _ storage.DataStore = (*SQLiteStore)(nil)

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

	// 读取迁移文件
	sqlBytes, err := migrations.FS.ReadFile("001_init_sqlite.sql")
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
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// Ping 测试数据库连接
func (s *SQLiteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
