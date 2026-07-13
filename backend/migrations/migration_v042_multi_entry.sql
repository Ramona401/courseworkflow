-- v0.42 数据库迁移：课件多入口支持
-- 执行日期：2026-05-22
-- 说明：
--   1. lesson_plan_id 改为可空（支持从主题直接创建、PPT上传等非教案来源）
--   2. 新增 source_type 区分课件来源
--   3. 新增 source_file_path 存储上传文件路径
--   4. 预建 edu 平台关联字段（v0.43 发布桥使用）

ALTER TABLE coursewares ALTER COLUMN lesson_plan_id DROP NOT NULL;

ALTER TABLE coursewares ADD COLUMN IF NOT EXISTS source_type VARCHAR(30) NOT NULL DEFAULT 'lesson_plan'
  CHECK (source_type IN ('lesson_plan', 'ppt_upload', 'topic_direct', 'html_import'));

ALTER TABLE coursewares ADD COLUMN IF NOT EXISTS source_file_path VARCHAR(500);

ALTER TABLE coursewares ADD COLUMN IF NOT EXISTS edu_module_id VARCHAR(50);
ALTER TABLE coursewares ADD COLUMN IF NOT EXISTS published_version INT DEFAULT 0;
