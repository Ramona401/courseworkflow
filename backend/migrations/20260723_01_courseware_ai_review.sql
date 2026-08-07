-- 课件 AI 审核助手数据库骨架
--
-- 设计目标：
--   1. 一次课件 AI 审核对应一条 session，会话可恢复、可追溯；
--   2. 长课件按教学逻辑拆成多个 batch 顺序审核；
--   3. 前一批产生的连续性账本持续传给后一批；
--   4. 页面、教案、大纲和提示词均记录快照哈希，防止旧结果冒充新审核；
--   5. AI 对话单独持久化，但 AI 不直接改变人工审核决定；
--   6. 所有 JSONB 字段均保存结构化结果，禁止仅保存不可解析的自然语言。
--
-- 本迁移采用 IF NOT EXISTS，重复执行不会重复建表。
-- 本批只建立持久化骨架，不创建后台任务，不调用模型。

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS courseware_ai_review_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    courseware_id UUID NOT NULL
        REFERENCES coursewares(id) ON DELETE CASCADE,

    reviewer_id UUID NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    assistant_id UUID NULL
        REFERENCES ai_assistants(id) ON DELETE SET NULL,

    lesson_plan_id UUID NULL
        REFERENCES lesson_plans(id) ON DELETE SET NULL,

    review_level SMALLINT NOT NULL DEFAULT 1
        CHECK (review_level IN (1, 2)),

    education_domain VARCHAR(32) NOT NULL,
    subject VARCHAR(128) NOT NULL DEFAULT '',
    grade VARCHAR(128) NOT NULL DEFAULT '',

    status VARCHAR(32) NOT NULL DEFAULT 'pending'
        CHECK (
            status IN (
                'pending',
                'preparing',
                'reviewing',
                'aggregating',
                'done',
                'failed',
                'cancelled'
            )
        ),

    current_stage VARCHAR(32) NOT NULL DEFAULT 'baseline'
        CHECK (
            current_stage IN (
                'baseline',
                'indexing',
                'batch_review',
                'risk_recheck',
                'finalize',
                'done'
            )
        ),

    current_batch_no INTEGER NOT NULL DEFAULT 0,
    total_batches INTEGER NOT NULL DEFAULT 0,

    -- 审核对象快照。任一哈希变化时，旧结果必须标记过期并重新审核。
    courseware_snapshot_hash VARCHAR(64) NOT NULL DEFAULT '',
    pages_snapshot_hash VARCHAR(64) NOT NULL DEFAULT '',
    lesson_plan_snapshot_hash VARCHAR(64) NOT NULL DEFAULT '',
    course_outline_snapshot_hash VARCHAR(64) NOT NULL DEFAULT '',

    -- 提示词留痕。系统硬规则与可配置助手提示词分别保存。
    system_prompt_key VARCHAR(128) NOT NULL DEFAULT '',
    system_prompt_version INTEGER NOT NULL DEFAULT 0,
    system_prompt_snapshot TEXT NOT NULL DEFAULT '',
    assistant_prompt_snapshot TEXT NOT NULL DEFAULT '',

    -- 结构化上下文与审核成果。
    context_manifest_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    baseline_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    page_index_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    continuity_ledger_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    final_report_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    model_used VARCHAR(128) NOT NULL DEFAULT '',
    tokens_used INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS courseware_ai_review_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    session_id UUID NOT NULL
        REFERENCES courseware_ai_review_sessions(id) ON DELETE CASCADE,

    batch_no INTEGER NOT NULL CHECK (batch_no > 0),

    -- 使用 JSONB 而非整数数组，便于同时保存页码、页面 ID、重叠页和风险页原因。
    page_scope_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    status VARCHAR(32) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'done', 'failed')),

    input_hash VARCHAR(64) NOT NULL DEFAULT '',

    -- 本批执行前的连续性账本快照。
    continuity_before_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 本批实际送入模型的结构化上下文清单，不保存密钥。
    input_manifest_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 本批发现、页码证据、互动代码证据、年级适配风险等。
    result_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 本批完成后供下一批继续使用的连续性账本。
    continuity_after_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    risk_pages_json JSONB NOT NULL DEFAULT '[]'::jsonb,

    model_used VARCHAR(128) NOT NULL DEFAULT '',
    tokens_used INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',

    started_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_courseware_ai_review_batch
        UNIQUE (session_id, batch_no)
);

CREATE TABLE IF NOT EXISTS courseware_ai_review_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    session_id UUID NOT NULL
        REFERENCES courseware_ai_review_sessions(id) ON DELETE CASCADE,

    user_id UUID NULL
        REFERENCES users(id) ON DELETE SET NULL,

    role VARCHAR(16) NOT NULL
        CHECK (role IN ('system', 'user', 'assistant')),

    content TEXT NOT NULL DEFAULT '',

    -- AI 回答引用的页码、批次、问题 ID 和证据位置。
    citations_json JSONB NOT NULL DEFAULT '[]'::jsonb,

    tokens_used INTEGER NOT NULL DEFAULT 0,
    model_used VARCHAR(128) NOT NULL DEFAULT '',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cw_ai_review_session_courseware
    ON courseware_ai_review_sessions(courseware_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_cw_ai_review_session_reviewer
    ON courseware_ai_review_sessions(reviewer_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_cw_ai_review_session_status
    ON courseware_ai_review_sessions(status, updated_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_cw_ai_review_active_session
    ON courseware_ai_review_sessions(
        courseware_id,
        reviewer_id,
        review_level
    )
    WHERE status IN (
        'pending',
        'preparing',
        'reviewing',
        'aggregating'
    );

CREATE INDEX IF NOT EXISTS idx_cw_ai_review_batch_session
    ON courseware_ai_review_batches(session_id, batch_no);

CREATE INDEX IF NOT EXISTS idx_cw_ai_review_message_session
    ON courseware_ai_review_messages(session_id, created_at);

COMMENT ON TABLE courseware_ai_review_sessions IS
    '课件AI审核会话：保存审核对象快照、课程基准、连续性账本和最终报告';

COMMENT ON TABLE courseware_ai_review_batches IS
    '课件AI审核分批结果：后一批必须继承前一批连续性账本';

COMMENT ON TABLE courseware_ai_review_messages IS
    '课件AI审核助手对话记录，AI回答不直接改变人工审核决定';

COMMENT ON COLUMN courseware_ai_review_sessions.continuity_ledger_json IS
    '跨批次连续性账本：案例、人物、数字、符号、已形成结论、待揭示问题和互动状态';

COMMENT ON COLUMN courseware_ai_review_batches.result_json IS
    '结构化本批审核结果，必须带页码、证据、严重程度、建议和人工复核标记';
