-- ============================================================================
-- migration_v210_courseware_publish.sql
-- 阶段1：课件发布与共享 + 产权分级共享
--
-- 目的：给 coursewares 主表新增"发布/审核维度"四列，与既有 status 生产状态机正交，
--       绝不改动 status（draft→...→preview→confirmed→in_pipeline 那套生产流程）。
--
-- 四列语义：
--   publish_state    课件发布态（独立于 status 生产态）
--                      private            私有（默认，等于现状，谁也看不到）
--                      published_personal 个人发布（作者自己标记完成，暂未共享）
--                      submitted          已提交审核（进入教研组/学校审核流，锁定编辑）
--                      approved           审核通过（待发布共享）
--                      published_shared   已共享发布（同校/同组可见）
--                      revision           审核退回（需修改后重新提交）
--   review_level     审核层级进度  0=未提交 / 1=L1教研组通过 / 2=L2学校通过
--   review_school_id 提交审核时反查到的作者所属学校ID（供 L2 学校审核按校过滤）
--   code_share_scope 源代码开放范围（产权保护，独立于"课件可见范围"）
--                      none   不开放源码（默认，别人只能看渲染效果不能复制）
--                      group  仅本教研组可复制源码
--                      school 仅本校可复制源码
--                      region 区域内可复制源码
--                      public 所有可见者均可复制源码
--
-- 对存量数据影响：四列均带 DEFAULT，存量 coursewares 自动填默认值
--   （private / 0 / NULL / none），语义=私有未审核代码不开放，完全等于改动前现状。
-- 幂等：使用 ADD COLUMN IF NOT EXISTS，可重复执行不报错。
-- ============================================================================

-- 1) publish_state：发布态（默认 private）
ALTER TABLE coursewares
    ADD COLUMN IF NOT EXISTS publish_state VARCHAR(30) NOT NULL DEFAULT 'private';

-- 2) review_level：审核层级进度（默认 0=未提交）
ALTER TABLE coursewares
    ADD COLUMN IF NOT EXISTS review_level SMALLINT NOT NULL DEFAULT 0;

-- 3) review_school_id：提交审核时反查的作者学校ID（可空，未提交时为 NULL）
ALTER TABLE coursewares
    ADD COLUMN IF NOT EXISTS review_school_id UUID DEFAULT NULL;

-- 4) code_share_scope：源代码开放范围（默认 none=不开放）
ALTER TABLE coursewares
    ADD COLUMN IF NOT EXISTS code_share_scope VARCHAR(20) NOT NULL DEFAULT 'none';

-- ---- 查询索引（共享课件库列表 + 审核列表按 publish_state / review 过滤）----

-- 共享课件库：按发布态 + 时间倒序列出已共享课件
CREATE INDEX IF NOT EXISTS idx_coursewares_publish_state
    ON coursewares (publish_state, updated_at DESC);

-- L2 学校审核：按 review_school_id + review_level 找待本校审核课件
CREATE INDEX IF NOT EXISTS idx_coursewares_review
    ON coursewares (review_school_id, review_level)
    WHERE publish_state = 'submitted';

-- 完成提示
DO $$
BEGIN
    RAISE NOTICE 'migration_v210_courseware_publish 执行完成：coursewares 已新增 4 列 + 2 索引';
END $$;
