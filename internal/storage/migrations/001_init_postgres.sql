-- PostgreSQL 初始化脚本
-- 创建所有必要的表和索引

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user' NOT NULL,
    status INTEGER DEFAULT 1 NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 数据源表
CREATE TABLE IF NOT EXISTS data_sources (
    id SERIAL PRIMARY KEY,
    collect_id VARCHAR(16) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    schema_config JSONB NOT NULL,
    status INTEGER DEFAULT 1 NOT NULL,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 数据 Token 表
CREATE TABLE IF NOT EXISTS data_tokens (
    id SERIAL PRIMARY KEY,
    source_id INTEGER NOT NULL REFERENCES data_sources(id),
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    status INTEGER DEFAULT 1 NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 数据记录表
CREATE TABLE IF NOT EXISTS data_records (
    id SERIAL PRIMARY KEY,
    source_id INTEGER NOT NULL REFERENCES data_sources(id),
    token_id INTEGER NOT NULL REFERENCES data_tokens(id),
    data JSONB NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 统计数据表
CREATE TABLE IF NOT EXISTS statistics (
    id SERIAL PRIMARY KEY,
    source_id INTEGER NOT NULL REFERENCES data_sources(id),
    stat_date DATE NOT NULL,
    count INTEGER DEFAULT 0 NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(source_id, stat_date)
);

-- 系统配置表
CREATE TABLE IF NOT EXISTS system_configs (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
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
