-- 为已有的数据源表添加 collect_id 字段
-- 注意：SQLite 不支持 ADD COLUMN IF NOT EXISTS，需要通过判断处理

-- 添加 collect_id 列（如果不存在）
ALTER TABLE data_sources ADD COLUMN collect_id VARCHAR(16) DEFAULT '' NOT NULL;

-- 为已存在的记录生成 collect_id（使用 hex + id 的方式生成唯一值）
UPDATE data_sources SET collect_id = 'src_' || lower(hex(randomblob(4))) WHERE collect_id = '';

-- 创建唯一索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_sources_collect_id ON data_sources(collect_id);
