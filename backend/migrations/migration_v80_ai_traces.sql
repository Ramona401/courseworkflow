-- ============================================================
-- TE-DNA 2.0  v80 迁移: AI调用追踪表
-- 执行方式: sudo -u postgres psql -d tedna -f migration_v80_ai_traces.sql
-- ============================================================

-- 1. 创建AI调用追踪表
CREATE TABLE IF NOT EXISTS ai_call_traces (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scene_code        VARCHAR(50)     NOT NULL,          -- 场景代码(scanner/evaluator/lesson_plan等)
    model_used        VARCHAR(100)    NOT NULL,          -- 实际使用的模型(含fallback后的)
    prompt_tokens     INT             DEFAULT 0,         -- 输入token数
    completion_tokens INT             DEFAULT 0,         -- 输出token数
    total_tokens      INT             DEFAULT 0,         -- 总token数
    latency_ms        INT             DEFAULT 0,         -- 调用延迟(毫秒)
    status            VARCHAR(20)     DEFAULT 'success', -- success/error/timeout/fallback
    error_message     TEXT            DEFAULT '',         -- 失败时的错误信息
    pipeline_id       UUID,                              -- 关联Pipeline(可为空)
    lesson_plan_id    UUID,                              -- 关联教案(可为空)
    user_id           UUID,                              -- 关联用户(可为空)
    estimated_cost_usd NUMERIC(10,6)  DEFAULT 0,         -- 基于模型定价的成本估算(美元)
    output_length     INT             DEFAULT 0,         -- AI输出内容长度(字符数)
    is_stream         BOOLEAN         DEFAULT false,     -- 是否流式调用
    created_at        TIMESTAMPTZ     DEFAULT now()      -- 调用时间
);

-- 2. 索引：按场景+时间查询（最常用）
CREATE INDEX idx_act_scene       ON ai_call_traces(scene_code, created_at DESC);
-- 按模型+时间查询
CREATE INDEX idx_act_model       ON ai_call_traces(model_used, created_at DESC);
-- 只索引非成功状态（错误追踪）
CREATE INDEX idx_act_status      ON ai_call_traces(status) WHERE status != 'success';
-- 按用户查询（仅有值时）
CREATE INDEX idx_act_user        ON ai_call_traces(user_id) WHERE user_id IS NOT NULL;
-- 按Pipeline查询（仅有值时）
CREATE INDEX idx_act_pipeline    ON ai_call_traces(pipeline_id) WHERE pipeline_id IS NOT NULL;
-- 按教案查询（仅有值时）
CREATE INDEX idx_act_lesson_plan ON ai_call_traces(lesson_plan_id) WHERE lesson_plan_id IS NOT NULL;
-- 按时间倒序查询（仪表盘趋势图）
CREATE INDEX idx_act_created     ON ai_call_traces(created_at DESC);

-- 3. 验证
DO $$
BEGIN
    RAISE NOTICE 'v80 migration complete: ai_call_traces table created with 7 indexes';
END $$;
