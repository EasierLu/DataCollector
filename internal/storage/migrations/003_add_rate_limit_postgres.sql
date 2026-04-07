-- 为数据源表添加限流配置字段

-- rate_limit: 每分钟请求数，0=使用全局默认
ALTER TABLE data_sources ADD COLUMN IF NOT EXISTS rate_limit INTEGER DEFAULT 0 NOT NULL;

-- rate_limit_burst: 突发量上限，0=使用全局默认
ALTER TABLE data_sources ADD COLUMN IF NOT EXISTS rate_limit_burst INTEGER DEFAULT 0 NOT NULL;
