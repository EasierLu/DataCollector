-- 为已有的数据源表添加 collect_id 字段

-- 添加 collect_id 列（如果不存在）
ALTER TABLE data_sources ADD COLUMN IF NOT EXISTS collect_id VARCHAR(16) DEFAULT '' NOT NULL;

-- 为已存在的记录生成 collect_id
UPDATE data_sources SET collect_id = substr(md5(random()::text), 1, 8) WHERE collect_id = '';

-- 创建唯一索引（如果不存在）
CREATE UNIQUE INDEX IF NOT EXISTS idx_sources_collect_id ON data_sources(collect_id);
