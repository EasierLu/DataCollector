-- SQLite 初始化脚本
-- 创建所有必要的表和索引

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user' NOT NULL,
    status INTEGER DEFAULT 1 NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 数据源表
CREATE TABLE IF NOT EXISTS data_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    schema_config TEXT NOT NULL,
    status INTEGER DEFAULT 1 NOT NULL,
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- 数据 Token 表
CREATE TABLE IF NOT EXISTS data_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    status INTEGER DEFAULT 1 NOT NULL,
    expires_at DATETIME,
    last_used_at DATETIME,
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_id) REFERENCES data_sources(id),
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- 数据记录表
CREATE TABLE IF NOT EXISTS data_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL,
    token_id INTEGER NOT NULL,
    data TEXT NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_id) REFERENCES data_sources(id),
    FOREIGN KEY (token_id) REFERENCES data_tokens(id)
);

-- 统计数据表
CREATE TABLE IF NOT EXISTS statistics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL,
    stat_date DATE NOT NULL,
    count INTEGER DEFAULT 0 NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_id, stat_date),
    FOREIGN KEY (source_id) REFERENCES data_sources(id)
);

-- 系统配置表
CREATE TABLE IF NOT EXISTS system_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

CREATE INDEX IF NOT EXISTS idx_sources_status ON data_sources(status);
CREATE INDEX IF NOT EXISTS idx_sources_created_by ON data_sources(created_by);

CREATE INDEX IF NOT EXISTS idx_tokens_source_id ON data_tokens(source_id);
CREATE INDEX IF NOT EXISTS idx_tokens_hash ON data_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_tokens_status ON data_tokens(status);

CREATE INDEX IF NOT EXISTS idx_records_source_id ON data_records(source_id);
CREATE INDEX IF NOT EXISTS idx_records_token_id ON data_records(token_id);
CREATE INDEX IF NOT EXISTS idx_records_created_at ON data_records(created_at);

CREATE INDEX IF NOT EXISTS idx_statistics_source_id ON statistics(source_id);
CREATE INDEX IF NOT EXISTS idx_statistics_stat_date ON statistics(stat_date);
CREATE INDEX IF NOT EXISTS idx_statistics_source_date ON statistics(source_id, stat_date);

CREATE INDEX IF NOT EXISTS idx_configs_key ON system_configs(config_key);
