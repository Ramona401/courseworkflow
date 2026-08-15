-- 20260815_01_courseware_review_comment_candidates.sql
--
-- R-08：审核意见重新汇总与差异预览。
--
-- 本迁移仅建立“不可变审核意见候选”持久化事实。
--
-- 设计原则：
--
--   1. AI生成的是候选，不直接修改courseware_reviews.comment；
--   2. 浏览器不能在后续替换/追加时回传候选正文作为可信事实；
--   3. 候选保存生成时的原意见快照、退回清单和服务端事实快照；
--   4. stale不保存为可漂移状态，而是在读取/采用时重新计算当前事实hash；
--   5. 正式审核意见仍只在最终人工提交事务中写入courseware_reviews；
--   6. 候选为追加型不可变记录，不提供UPDATE或DELETE业务语义；
--   7. 与R-06/R-07保持软关联模式，运行时重新校验会话、课件和审核员归属。
--
-- input_snapshot_json后续由Service按固定schema构建，至少包含：
--
--   - courseware_id / source_session_id / reviewer_id / review_level；
--   - 当前本次修改清单中的规范化item IDs；
--   - 每项current_instruction_version_id；
--   - 指令version_no / content_hash / page_snapshot_hash / status；
--   - 与当前清单有关的R-06 active group版本、名称、主问题；
--   - 对应active member稳定ID、item_id和version；
--   - original_comment_hash。
--
-- diff_json仅用于教师差异预览，不作为正式审核事实，也不参与最终提交事务。

BEGIN;

CREATE TABLE courseware_review_comment_candidates (
        id UUID PRIMARY KEY,

        -- 运行时作用域全部由Service重新读取和复核。
        courseware_id UUID NOT NULL,
        source_session_id UUID NOT NULL,
        created_by UUID NOT NULL,
        review_level INTEGER NOT NULL,

        -- 候选正文协议版本。
        candidate_schema_version INTEGER NOT NULL DEFAULT 1,

        -- AI生成后的服务器冻结候选正文。
        candidate_text TEXT NOT NULL,

        -- 生成候选时教师输入框中的原审核意见。
        -- 允许为空，因为教师可以在尚未手工填写意见时先执行重新整理。
        original_comment_snapshot TEXT NOT NULL DEFAULT '',
        original_comment_hash TEXT NOT NULL,

        -- 浏览器只提交本轮明确选择的item IDs作为教师意图。
        -- Service必须重新读取真实整改项与当前指令版本后再写入本表。
        selected_item_ids_json JSONB NOT NULL DEFAULT '[]'::jsonb,

        -- 服务端可信事实快照。
        input_snapshot_schema_version INTEGER NOT NULL DEFAULT 1,
        input_snapshot_json JSONB NOT NULL DEFAULT '{}'::jsonb,
        input_hash TEXT NOT NULL,

        -- 原意见与候选意见的教师化差异预览。
        -- 后续协议固定为object；具体added/removed/adjusted结构由Service校验。
        diff_schema_version INTEGER NOT NULL DEFAULT 1,
        diff_json JSONB NOT NULL DEFAULT
                '{"added":[],"removed":[],"adjusted":[]}'::jsonb,

        -- 仅用于内部AI调用审计与问题排查。
        -- HTTP安全视图后续不得依赖这些字段进行业务授权。
        model_used TEXT NOT NULL DEFAULT '',
        tokens_used INTEGER NOT NULL DEFAULT 0,

        -- 无updated_at：候选创建后即冻结。
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

        CONSTRAINT ck_cw_review_comment_candidate_level
                CHECK (review_level IN (1, 2)),

        CONSTRAINT ck_cw_review_comment_candidate_schema
                CHECK (
                        candidate_schema_version > 0
                        AND input_snapshot_schema_version > 0
                        AND diff_schema_version > 0
                ),

        CONSTRAINT ck_cw_review_comment_candidate_text
                CHECK (btrim(candidate_text) <> ''),

        CONSTRAINT ck_cw_review_comment_candidate_original_hash
                CHECK (
                        original_comment_hash ~ '^[0-9a-f]{64}$'
                ),

        CONSTRAINT ck_cw_review_comment_candidate_input_hash
                CHECK (
                        input_hash ~ '^[0-9a-f]{64}$'
                ),

        CONSTRAINT ck_cw_review_comment_candidate_selected_items
                CHECK (
                        jsonb_typeof(selected_item_ids_json) = 'array'
                ),

        CONSTRAINT ck_cw_review_comment_candidate_input_snapshot
                CHECK (
                        jsonb_typeof(input_snapshot_json) = 'object'
                ),

        CONSTRAINT ck_cw_review_comment_candidate_diff
                CHECK (
                        jsonb_typeof(diff_json) = 'object'
                ),

        CONSTRAINT ck_cw_review_comment_candidate_tokens
                CHECK (tokens_used >= 0)
);

-- 带完整作用域列的唯一锚点。
-- 后续Repository读取候选时应尽量同时携带session/courseware/created_by，
-- 避免只凭candidate_id形成跨作用域探测入口。
CREATE UNIQUE INDEX uq_cw_review_comment_candidate_scope
        ON courseware_review_comment_candidates (
                id,
                courseware_id,
                source_session_id,
                created_by
        );

-- 当前审核会话读取最近候选的主要查询路径。
CREATE INDEX idx_cw_review_comment_candidate_session
        ON courseware_review_comment_candidates (
                source_session_id,
                created_by,
                created_at DESC
        );

-- 按课件、审核员和审核级别排查候选历史。
CREATE INDEX idx_cw_review_comment_candidate_courseware
        ON courseware_review_comment_candidates (
                courseware_id,
                created_by,
                review_level,
                created_at DESC
        );

COMMENT ON TABLE courseware_review_comment_candidates IS
        'R-08正式课件审核意见不可变AI候选；正式审核意见仍由人工最终提交事务写入courseware_reviews';

COMMENT ON COLUMN courseware_review_comment_candidates.original_comment_snapshot IS
        '生成候选时教师审核意见原文快照，不代表正式审核记录';

COMMENT ON COLUMN courseware_review_comment_candidates.selected_item_ids_json IS
        '生成候选时教师明确选择的本次修改清单ID集合；真实整改事实由Service重新读取';

COMMENT ON COLUMN courseware_review_comment_candidates.input_snapshot_json IS
        '服务端冻结的R-08可信输入事实快照，用于重新计算stale';

COMMENT ON COLUMN courseware_review_comment_candidates.input_hash IS
        '规范化input_snapshot_json的SHA-256，用于fail-closed失效校验';

COMMENT ON COLUMN courseware_review_comment_candidates.diff_json IS
        '原意见与新候选的教师化新增/移除/调整预览，仅用于展示';

-- 候选必须真正不可变。
--
-- 不设置status、applied_at或cancelled_at：
-- “替换/追加/取消”只是正式提交前的教师编辑动作，
-- 不应把R-08候选再扩展成独立业务状态机。
CREATE OR REPLACE FUNCTION guard_courseware_review_comment_candidate_immutable()
RETURNS TRIGGER
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $$
BEGIN
        RAISE EXCEPTION
                'courseware_review_comment_candidates为不可变审核意见候选，禁止更新或删除'
                USING ERRCODE = 'P0001';
END;
$$;

CREATE TRIGGER trg_cw_review_comment_candidate_immutable
        BEFORE UPDATE OR DELETE
        ON courseware_review_comment_candidates
        FOR EACH ROW
        EXECUTE FUNCTION guard_courseware_review_comment_candidate_immutable();

COMMIT;
