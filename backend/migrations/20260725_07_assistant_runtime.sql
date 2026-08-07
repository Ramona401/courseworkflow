-- ============================================================================
-- TE-DNA 2.0：课件教学智能体匿名运行会话与使用流水
-- 文件：20260725_07_assistant_runtime.sql
-- ----------------------------------------------------------------------------
-- 本迁移建立：
--
--   1. assistant_runtime_sessions
--      保存外部匿名学生或教师内部预览的短时运行会话、
--      令牌JTI哈希、来源快照、可见消息、轮数以及主轮次互斥状态；
--
--   2. assistant_runtime_usage
--      保存每一次成功或失败运行的模型、Token、积分、耗时和错误流水。
--
-- 安全原则：
--
--   * 不保存学生真实姓名；
--   * 不保存原始IP，只保存服务端加盐后的SHA-256哈希；
--   * 不保存运行令牌原文，只保存JTI的SHA-256哈希；
--   * 不保存模型API Key；
--   * 不保存隐藏推理；
--   * session严格绑定不可变deployment version；
--   * usage使用唯一turn_id实现幂等；
--   * usage为追加式流水，应用角色只能SELECT和INSERT。
--
-- 并发原则：
--
--   * active_turn_id为空时才能领取新主轮次；
--   * 后端仓储通过带条件的UPDATE原子领取；
--   * 成功或失败完成后必须清除active_turn_id；
--   * 成功时在同一事务内递增turn_count并写usage；
--   * turn_count不得超过max_turns。
-- ============================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- 一、短时运行会话
-- ============================================================================

CREATE TABLE IF NOT EXISTS assistant_runtime_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    deployment_id UUID NOT NULL,
    deployment_version INTEGER NOT NULL,

    -- 短时运行令牌jti经SHA-256后的十六进制小写哈希。
    -- 重新签发令牌时覆盖本字段，旧令牌立即失效。
    token_jti_hash VARCHAR(64) NOT NULL,

    -- 浏览器生成的匿名客户端标识经服务端加盐哈希。
    anonymous_client_hash VARCHAR(64) NOT NULL,

    -- 会话创建时的Origin快照。
    origin_snapshot VARCHAR(512) NOT NULL,

    -- 原始IP不落库，只保存服务端加盐SHA-256哈希。
    ip_hash VARCHAR(64) NOT NULL,

    -- external：外部学生iframe会话；
    -- teacher_preview：TE-DNA内部教师测试会话。
    session_kind VARCHAR(24) NOT NULL DEFAULT 'external',

    status VARCHAR(20) NOT NULL DEFAULT 'active',

    turn_count INTEGER NOT NULL DEFAULT 0,
    max_turns INTEGER NOT NULL,

    -- 当前正在执行的唯一主轮次。
    -- 后端通过UPDATE ... WHERE active_turn_id IS NULL原子领取。
    active_turn_id UUID NULL,
    active_turn_started_at TIMESTAMPTZ NULL,

    -- 只保存正式可见消息，不保存系统提示词或隐藏推理。
    -- 数据格式由后端模型限定：
    -- [{"role":"student|assistant","content":"...","created_at":"..."}]
    messages_json JSONB NOT NULL DEFAULT '[]'::jsonb,

    expires_at TIMESTAMPTZ NOT NULL,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_assistant_runtime_sessions_deployment
        FOREIGN KEY (deployment_id)
        REFERENCES assistant_deployments(id)
        ON DELETE CASCADE,

    -- 会话只能绑定已经真实发布的不可变版本。
    CONSTRAINT fk_assistant_runtime_sessions_version
        FOREIGN KEY (
            deployment_id,
            deployment_version
        )
        REFERENCES assistant_deployment_versions(
            deployment_id,
            version
        )
        ON DELETE CASCADE,

    CONSTRAINT uq_assistant_runtime_sessions_jti
        UNIQUE (token_jti_hash),

    CONSTRAINT ck_assistant_runtime_sessions_version
        CHECK (deployment_version > 0),

    CONSTRAINT ck_assistant_runtime_sessions_jti_hash
        CHECK (
            token_jti_hash ~ '^[0-9a-f]{64}$'
        ),

    CONSTRAINT ck_assistant_runtime_sessions_client_hash
        CHECK (
            anonymous_client_hash ~ '^[0-9a-f]{64}$'
        ),

    CONSTRAINT ck_assistant_runtime_sessions_ip_hash
        CHECK (
            ip_hash ~ '^[0-9a-f]{64}$'
        ),

    CONSTRAINT ck_assistant_runtime_sessions_origin
        CHECK (
            length(btrim(origin_snapshot)) BETWEEN 1 AND 512
        ),

    CONSTRAINT ck_assistant_runtime_sessions_kind
        CHECK (
            session_kind IN (
                'external',
                'teacher_preview'
            )
        ),

    CONSTRAINT ck_assistant_runtime_sessions_status
        CHECK (
            status IN (
                'active',
                'completed',
                'expired',
                'revoked'
            )
        ),

    CONSTRAINT ck_assistant_runtime_sessions_turn_count
        CHECK (
            turn_count >= 0
            AND turn_count <= max_turns
        ),

    CONSTRAINT ck_assistant_runtime_sessions_max_turns
        CHECK (
            max_turns BETWEEN 1 AND 100
        ),

    -- active_turn_id和开始时间必须同时为空或同时非空。
    CONSTRAINT ck_assistant_runtime_sessions_active_turn_pair
        CHECK (
            (
                active_turn_id IS NULL
                AND active_turn_started_at IS NULL
            )
            OR
            (
                active_turn_id IS NOT NULL
                AND active_turn_started_at IS NOT NULL
            )
        ),

    -- 非active终态不得继续持有运行中的轮次。
    CONSTRAINT ck_assistant_runtime_sessions_terminal_turn
        CHECK (
            status = 'active'
            OR active_turn_id IS NULL
        ),

    CONSTRAINT ck_assistant_runtime_sessions_messages
        CHECK (
            jsonb_typeof(messages_json) = 'array'
        ),

    CONSTRAINT ck_assistant_runtime_sessions_expiry
        CHECK (
            expires_at > created_at
        ),

    CONSTRAINT ck_assistant_runtime_sessions_activity_time
        CHECK (
            last_active_at >= created_at
        )
);

CREATE INDEX IF NOT EXISTS idx_assistant_runtime_sessions_deployment
    ON assistant_runtime_sessions(
        deployment_id,
        status,
        created_at DESC
    );

CREATE INDEX IF NOT EXISTS idx_assistant_runtime_sessions_expiry
    ON assistant_runtime_sessions(
        status,
        expires_at
    )
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_assistant_runtime_sessions_client
    ON assistant_runtime_sessions(
        deployment_id,
        anonymous_client_hash,
        created_at DESC
    );

CREATE INDEX IF NOT EXISTS idx_assistant_runtime_sessions_active_turn
    ON assistant_runtime_sessions(
        active_turn_started_at
    )
    WHERE active_turn_id IS NOT NULL;

COMMENT ON TABLE assistant_runtime_sessions IS
    '教学智能体匿名短时运行会话；绑定不可变部署版本并管理轮数和主轮次互斥';

COMMENT ON COLUMN assistant_runtime_sessions.token_jti_hash IS
    '短时运行令牌jti的SHA-256小写十六进制哈希，不保存令牌原文';

COMMENT ON COLUMN assistant_runtime_sessions.ip_hash IS
    '请求IP加服务端盐后的SHA-256哈希，不保存原始IP';

COMMENT ON COLUMN assistant_runtime_sessions.active_turn_id IS
    '当前唯一执行中的主轮次ID；条件更新用于防并发重复调用';

COMMENT ON COLUMN assistant_runtime_sessions.messages_json IS
    '学生和助手正式可见消息数组，不保存系统提示词或隐藏推理';

-- ============================================================================
-- 二、运行使用流水
-- ============================================================================

CREATE TABLE IF NOT EXISTS assistant_runtime_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 每个运行主轮次拥有全局唯一turn_id。
    -- 重复回调或重试写入将被唯一约束拒绝，防止重复结算。
    turn_id UUID NOT NULL,

    -- 部署或会话后续被删除时，流水继续保留。
    -- ID字段只被置空，其余快照字段仍保留。
    deployment_id UUID NULL,
    runtime_session_id UUID NULL,

    deployment_version INTEGER NOT NULL,

    -- 以下均为调用发生时的计费和资源快照ID。
    -- 不建立用户、学校、课件或页面外键，避免历史主体删除后丢失流水。
    owner_user_id UUID NOT NULL,
    school_id UUID NOT NULL,
    courseware_id UUID NOT NULL,
    page_id UUID NOT NULL,

    session_kind VARCHAR(24) NOT NULL DEFAULT 'external',

    input_chars INTEGER NOT NULL DEFAULT 0,
    output_chars INTEGER NOT NULL DEFAULT 0,

    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,

    credits_used NUMERIC(16,4) NOT NULL DEFAULT 0,

    model_name VARCHAR(128) NOT NULL DEFAULT '',
    provider VARCHAR(64) NOT NULL DEFAULT '',

    status VARCHAR(20) NOT NULL,
    error_code VARCHAR(64) NOT NULL DEFAULT '',
    latency_ms INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_assistant_runtime_usage_turn
        UNIQUE (turn_id),

    CONSTRAINT fk_assistant_runtime_usage_deployment
        FOREIGN KEY (deployment_id)
        REFERENCES assistant_deployments(id)
        ON DELETE SET NULL,

    CONSTRAINT fk_assistant_runtime_usage_session
        FOREIGN KEY (runtime_session_id)
        REFERENCES assistant_runtime_sessions(id)
        ON DELETE SET NULL,

    CONSTRAINT ck_assistant_runtime_usage_version
        CHECK (deployment_version > 0),

    CONSTRAINT ck_assistant_runtime_usage_kind
        CHECK (
            session_kind IN (
                'external',
                'teacher_preview'
            )
        ),

    CONSTRAINT ck_assistant_runtime_usage_input_chars
        CHECK (input_chars >= 0),

    CONSTRAINT ck_assistant_runtime_usage_output_chars
        CHECK (output_chars >= 0),

    CONSTRAINT ck_assistant_runtime_usage_input_tokens
        CHECK (input_tokens >= 0),

    CONSTRAINT ck_assistant_runtime_usage_output_tokens
        CHECK (output_tokens >= 0),

    CONSTRAINT ck_assistant_runtime_usage_credits
        CHECK (credits_used >= 0),

    CONSTRAINT ck_assistant_runtime_usage_status
        CHECK (
            status IN (
                'succeeded',
                'failed'
            )
        ),

    CONSTRAINT ck_assistant_runtime_usage_latency
        CHECK (latency_ms >= 0),

    -- 成功调用必须记录实际模型名称。
    CONSTRAINT ck_assistant_runtime_usage_success_model
        CHECK (
            status <> 'succeeded'
            OR length(btrim(model_name)) > 0
        ),

    -- 成功流水不得携带错误码。
    CONSTRAINT ck_assistant_runtime_usage_success_error
        CHECK (
            status <> 'succeeded'
            OR error_code = ''
        )
);

CREATE INDEX IF NOT EXISTS idx_assistant_runtime_usage_deployment
    ON assistant_runtime_usage(
        deployment_id,
        created_at DESC
    )
    WHERE deployment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_assistant_runtime_usage_session
    ON assistant_runtime_usage(
        runtime_session_id,
        created_at
    )
    WHERE runtime_session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_assistant_runtime_usage_owner
    ON assistant_runtime_usage(
        owner_user_id,
        created_at DESC
    );

-- 每日额度查询只统计成功调用。
CREATE INDEX IF NOT EXISTS idx_assistant_runtime_usage_daily_success
    ON assistant_runtime_usage(
        deployment_id,
        created_at
    )
    WHERE status = 'succeeded'
      AND deployment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_assistant_runtime_usage_errors
    ON assistant_runtime_usage(
        deployment_id,
        error_code,
        created_at DESC
    )
    WHERE status = 'failed'
      AND deployment_id IS NOT NULL;

COMMENT ON TABLE assistant_runtime_usage IS
    '教学智能体运行使用流水；记录成功和失败调用并保留Token、积分和资源快照';

COMMENT ON COLUMN assistant_runtime_usage.turn_id IS
    '服务端生成的唯一主轮次ID，用于防止重复流水和重复结算';

COMMENT ON COLUMN assistant_runtime_usage.credits_used IS
    '本次成功调用实际消费积分，口径与token_consumption_logs.credits_consumed一致';

COMMENT ON COLUMN assistant_runtime_usage.status IS
    'succeeded表示AI调用成功；failed表示配置、额度、模型或网络等失败';

-- ============================================================================
-- 三、运行流水不可原地修改
-- ============================================================================

CREATE OR REPLACE FUNCTION
    tedna_reject_assistant_runtime_usage_update()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION
        'assistant_runtime_usage是追加式使用流水，禁止UPDATE'
        USING ERRCODE = '55000';
END;
$$;

DROP TRIGGER IF EXISTS
    trg_reject_assistant_runtime_usage_update
ON assistant_runtime_usage;

CREATE TRIGGER
    trg_reject_assistant_runtime_usage_update
BEFORE UPDATE
ON assistant_runtime_usage
FOR EACH ROW
EXECUTE FUNCTION
    tedna_reject_assistant_runtime_usage_update();

-- ============================================================================
-- 四、显式收紧默认权限并按最小权限授权
-- ============================================================================

REVOKE ALL PRIVILEGES
ON TABLE
    assistant_runtime_sessions,
    assistant_runtime_usage
FROM PUBLIC;

REVOKE ALL PRIVILEGES
ON TABLE
    assistant_runtime_sessions,
    assistant_runtime_usage
FROM tedna_user;

-- 会话需要创建、读取、状态更新以及到期清理。
GRANT
    SELECT,
    INSERT,
    UPDATE,
    DELETE
ON TABLE
    assistant_runtime_sessions
TO tedna_user;

-- 流水只允许读取和追加。
GRANT
    SELECT,
    INSERT
ON TABLE
    assistant_runtime_usage
TO tedna_user;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '教学智能体运行会话和使用流水迁移完成';
END
$$;
