-- ============================================================================
-- TE-DNA 2.0：课件原生教学智能体核心数据表
-- 文件：20260725_05_courseware_assistant_core.sql
-- ----------------------------------------------------------------------------
-- 本迁移属于课件教学智能体MVP第一批数据库结构，只建立：
--
--   1. courseware_assistant_slots
--      保存课件页面当前可编辑的教学智能体配置；
--
--   2. assistant_deployments
--      保存对外运行部署、状态、额度和允许来源；
--
--   3. assistant_deployment_versions
--      保存每次发布时的不可变完整快照。
--
-- 本迁移不建立匿名运行会话和调用流水；
-- assistant_runtime_sessions及assistant_runtime_usage由下一开发单元建立。
--
-- 核心边界：
--
--   * 每个课件页面最多一个编辑态插槽；
--   * 每个页面最多一个active或paused部署；
--   * revoked部署不可恢复，但同一页面可重新建立新部署；
--   * 课件或页面被物理删除时，对应插槽和部署失效并级联删除；
--   * 编辑态插槽中的助手删除后，assistant_id自动置空；
--   * 发布版本中的assistant_id是软关联快照，不建立外键、不被助手删除改写；
--   * 插槽被删除时，既有部署保留，slot_id置空；
--   * 部署版本禁止UPDATE，只允许追加新版本；
--   * 发布运行域只允许k12、vocational、adult具体教学域。
-- ============================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- 一、课件页面教学智能体编辑态插槽
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_assistant_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 插槽必须绑定真实课件及稳定页面ID。
    courseware_id UUID NOT NULL,
    page_id UUID NOT NULL,

    -- 可选择现有AI助手。
    -- 助手被删除后插槽仍保留，但必须重新选择助手或重新生成方案。
    assistant_id UUID NULL,

    -- 创建人仅用于审计；正式写权限仍由后端服务层按课件作者判断。
    created_by UUID NOT NULL,

    -- MVP只支持右下角悬浮助手。
    display_mode VARCHAR(24) NOT NULL DEFAULT 'floating',
    display_position VARCHAR(24) NOT NULL DEFAULT 'bottom_right',

    title VARCHAR(120) NOT NULL DEFAULT '',
    welcome_message TEXT NOT NULL DEFAULT '',
    teaching_role TEXT NOT NULL DEFAULT '',
    learning_objective TEXT NOT NULL DEFAULT '',

    -- 完整结构化教学方案：
    -- guiding_principles、question_chain、misconception_branches、
    -- forbidden_behaviors、completion_criteria等字段由后端模型约束。
    guidance_plan_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 当前编辑态允许读取的上下文范围设置。
    -- 这里只保存配置，不保存完整页面、教案或助手提示词正文。
    context_config_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    status VARCHAR(20) NOT NULL DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_courseware_assistant_slots_courseware
        FOREIGN KEY (courseware_id)
        REFERENCES coursewares(id)
        ON DELETE CASCADE,

    -- 复合外键保证page_id确实属于courseware_id。
    CONSTRAINT fk_courseware_assistant_slots_page
        FOREIGN KEY (page_id, courseware_id)
        REFERENCES courseware_pages(id, courseware_id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_assistant_slots_assistant
        FOREIGN KEY (assistant_id)
        REFERENCES ai_assistants(id)
        ON DELETE SET NULL,

    CONSTRAINT fk_courseware_assistant_slots_creator
        FOREIGN KEY (created_by)
        REFERENCES users(id)
        ON DELETE RESTRICT,

    CONSTRAINT uq_courseware_assistant_slots_courseware_page
        UNIQUE (courseware_id, page_id),

    -- 供部署表使用复合外键，保证部署引用的插槽、课件和页面完全一致。
    CONSTRAINT uq_courseware_assistant_slots_id_courseware_page
        UNIQUE (id, courseware_id, page_id),

    CONSTRAINT ck_courseware_assistant_slots_display_mode
        CHECK (display_mode = 'floating'),

    CONSTRAINT ck_courseware_assistant_slots_display_position
        CHECK (display_position = 'bottom_right'),

    CONSTRAINT ck_courseware_assistant_slots_status
        CHECK (status IN ('active', 'disabled')),

    CONSTRAINT ck_courseware_assistant_slots_guidance_plan
        CHECK (jsonb_typeof(guidance_plan_json) = 'object'),

    CONSTRAINT ck_courseware_assistant_slots_context_config
        CHECK (jsonb_typeof(context_config_json) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_courseware_assistant_slots_assistant
    ON courseware_assistant_slots(assistant_id)
    WHERE assistant_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_courseware_assistant_slots_creator
    ON courseware_assistant_slots(created_by, updated_at DESC);

COMMENT ON TABLE courseware_assistant_slots IS
    '课件页面教学智能体编辑态插槽；每个稳定页面最多一条配置';

COMMENT ON COLUMN courseware_assistant_slots.page_id IS
    '稳定页面ID；页面重排不改变插槽归属';

COMMENT ON COLUMN courseware_assistant_slots.guidance_plan_json IS
    '结构化教学方案，不保存模型密钥或完整系统提示词';

COMMENT ON COLUMN courseware_assistant_slots.context_config_json IS
    '教师选择的上下文范围配置，不是正式发布上下文快照';

-- ============================================================================
-- 二、对外运行部署
-- ============================================================================

CREATE TABLE IF NOT EXISTS assistant_deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 公开编号由服务端使用密码学安全随机源生成。
    -- 只允许URL安全字符，不使用自增编号或可推断序号。
    public_id VARCHAR(64) NOT NULL,

    -- 插槽删除后部署仍可依赖已发布版本继续运行。
    slot_id UUID NULL,

    courseware_id UUID NOT NULL,
    page_id UUID NOT NULL,

    -- MVP固定由部署创建者的个人积分账户付费。
    owner_user_id UUID NOT NULL,

    -- 发布时由服务端解析并固化，用于模型分流和追踪。
    school_id UUID NOT NULL,
    education_domain VARCHAR(20) NOT NULL,

    -- 0表示部署尚未完成首个版本；
    -- 正式运行必须由服务层要求current_version大于0。
    current_version INTEGER NOT NULL DEFAULT 0,

    -- MVP只支持按部署允许来源域名创建外部运行会话。
    access_mode VARCHAR(24) NOT NULL DEFAULT 'origin_allowlist',

    status VARCHAR(20) NOT NULL DEFAULT 'active',

    daily_call_limit INTEGER NOT NULL DEFAULT 100,
    per_session_turn_limit INTEGER NOT NULL DEFAULT 20,

    -- JSON字符串数组。空数组是安全的fail-closed状态：
    -- 没有任何外部来源可以创建会话。
    allowed_origins_json JSONB NOT NULL DEFAULT '[]'::jsonb,

    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_assistant_deployments_public_id
        UNIQUE (public_id),

    CONSTRAINT fk_assistant_deployments_courseware
        FOREIGN KEY (courseware_id)
        REFERENCES coursewares(id)
        ON DELETE CASCADE,

    -- 页面删除时部署立即失效并级联删除。
    CONSTRAINT fk_assistant_deployments_page
        FOREIGN KEY (page_id, courseware_id)
        REFERENCES courseware_pages(id, courseware_id)
        ON DELETE CASCADE,

    -- 删除编辑态插槽不会删除已发布部署。
    -- 只清空slot_id，courseware_id和page_id历史边界继续保留。
    CONSTRAINT fk_assistant_deployments_slot
        FOREIGN KEY (
            slot_id,
            courseware_id,
            page_id
        )
        REFERENCES courseware_assistant_slots(
            id,
            courseware_id,
            page_id
        )
        ON DELETE SET NULL (slot_id),

    CONSTRAINT fk_assistant_deployments_owner
        FOREIGN KEY (owner_user_id)
        REFERENCES users(id)
        ON DELETE RESTRICT,

    CONSTRAINT fk_assistant_deployments_school
        FOREIGN KEY (school_id)
        REFERENCES organizations(id)
        ON DELETE RESTRICT,

    CONSTRAINT ck_assistant_deployments_public_id
        CHECK (
            public_id ~ '^[A-Za-z0-9_-]{32,64}$'
        ),

    CONSTRAINT ck_assistant_deployments_domain
        CHECK (
            education_domain IN (
                'k12',
                'vocational',
                'adult'
            )
        ),

    CONSTRAINT ck_assistant_deployments_current_version
        CHECK (current_version >= 0),

    CONSTRAINT ck_assistant_deployments_access_mode
        CHECK (access_mode = 'origin_allowlist'),

    CONSTRAINT ck_assistant_deployments_status
        CHECK (
            status IN (
                'active',
                'paused',
                'revoked'
            )
        ),

    CONSTRAINT ck_assistant_deployments_daily_limit
        CHECK (
            daily_call_limit BETWEEN 1 AND 100000
        ),

    CONSTRAINT ck_assistant_deployments_session_turn_limit
        CHECK (
            per_session_turn_limit BETWEEN 1 AND 100
        ),

    CONSTRAINT ck_assistant_deployments_allowed_origins
        CHECK (
            jsonb_typeof(allowed_origins_json) = 'array'
        ),

    CONSTRAINT ck_assistant_deployments_valid_range
        CHECK (
            valid_until IS NULL
            OR valid_until > valid_from
        )
);

-- 同一页面同一时刻最多存在一个未撤销部署。
-- revoked部署保留历史，但允许页面重新建立新部署。
CREATE UNIQUE INDEX IF NOT EXISTS
    uq_assistant_deployments_page_live
ON assistant_deployments(courseware_id, page_id)
WHERE status IN ('active', 'paused');

CREATE INDEX IF NOT EXISTS idx_assistant_deployments_owner
    ON assistant_deployments(owner_user_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_assistant_deployments_school
    ON assistant_deployments(
        school_id,
        education_domain,
        status
    );

CREATE INDEX IF NOT EXISTS idx_assistant_deployments_status_validity
    ON assistant_deployments(
        status,
        valid_from,
        valid_until
    );

COMMENT ON TABLE assistant_deployments IS
    '课件教学智能体对外部署；保存公开编号、运行边界、额度和当前版本';

COMMENT ON COLUMN assistant_deployments.public_id IS
    '不可预测的公开部署编号；不是授权凭证，不包含内部主键';

COMMENT ON COLUMN assistant_deployments.owner_user_id IS
    '匿名学生调用的实际积分付费用户快照';

COMMENT ON COLUMN assistant_deployments.school_id IS
    '部署发布时固化的学校ID，用于模型分流和AI追踪';

COMMENT ON COLUMN assistant_deployments.allowed_origins_json IS
    '允许创建匿名运行会话的外部Origin字符串数组；空数组表示全部拒绝';

-- ============================================================================
-- 三、不可变发布版本
-- ============================================================================

CREATE TABLE IF NOT EXISTS assistant_deployment_versions (
    deployment_id UUID NOT NULL,
    version INTEGER NOT NULL,

    -- 软关联快照：
    -- 不建立外键，避免助手删除通过SET NULL修改不可变历史版本。
    -- 助手被删除后仍保留发布时的原始助手ID和完整提示词快照。
    assistant_id UUID NULL,

    assistant_prompt_snapshot TEXT NOT NULL,
    assistant_prompt_hash VARCHAR(64) NOT NULL,

    -- 欢迎语、教学角色、目标、问题链、错误分支和禁止行为等发布快照。
    teaching_plan_json JSONB NOT NULL,

    -- 页面、相邻页、互动证据和来源教案片段的确定性快照。
    context_snapshot_json JSONB NOT NULL,
    context_snapshot_hash VARCHAR(64) NOT NULL,

    page_html_hash VARCHAR(64) NOT NULL,

    -- 标题、学科、层级、页面ID、教育域和其它最小课件发布信息。
    courseware_snapshot_json JSONB NOT NULL,

    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_assistant_deployment_versions
        PRIMARY KEY (deployment_id, version),

    CONSTRAINT fk_assistant_deployment_versions_deployment
        FOREIGN KEY (deployment_id)
        REFERENCES assistant_deployments(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_assistant_deployment_versions_creator
        FOREIGN KEY (created_by)
        REFERENCES users(id)
        ON DELETE RESTRICT,

    CONSTRAINT ck_assistant_deployment_versions_version
        CHECK (version > 0),

    CONSTRAINT ck_assistant_deployment_versions_prompt
        CHECK (
            length(btrim(assistant_prompt_snapshot)) > 0
        ),

    CONSTRAINT ck_assistant_deployment_versions_prompt_hash
        CHECK (
            assistant_prompt_hash ~ '^[0-9a-f]{64}$'
        ),

    CONSTRAINT ck_assistant_deployment_versions_teaching_plan
        CHECK (
            jsonb_typeof(teaching_plan_json) = 'object'
        ),

    CONSTRAINT ck_assistant_deployment_versions_context
        CHECK (
            jsonb_typeof(context_snapshot_json) = 'object'
        ),

    CONSTRAINT ck_assistant_deployment_versions_context_hash
        CHECK (
            context_snapshot_hash ~ '^[0-9a-f]{64}$'
        ),

    CONSTRAINT ck_assistant_deployment_versions_page_hash
        CHECK (
            page_html_hash ~ '^[0-9a-f]{64}$'
        ),

    CONSTRAINT ck_assistant_deployment_versions_courseware_snapshot
        CHECK (
            jsonb_typeof(courseware_snapshot_json) = 'object'
        )
);

CREATE INDEX IF NOT EXISTS idx_assistant_deployment_versions_assistant
    ON assistant_deployment_versions(assistant_id)
    WHERE assistant_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_assistant_deployment_versions_creator
    ON assistant_deployment_versions(created_by, created_at DESC);

COMMENT ON TABLE assistant_deployment_versions IS
    '课件教学智能体不可变发布版本；只能追加，不允许覆盖历史版本';

COMMENT ON COLUMN assistant_deployment_versions.assistant_id IS
    '发布时助手ID软关联快照；不建立外键，助手删除后仍保留原始ID';

COMMENT ON COLUMN assistant_deployment_versions.assistant_prompt_snapshot IS
    '发布时完整助手提示词快照，仅供后端运行，禁止返回浏览器';

COMMENT ON COLUMN assistant_deployment_versions.context_snapshot_json IS
    '发布时确定性页面教学上下文快照，不随原课件后续修改自动变化';

COMMENT ON COLUMN assistant_deployment_versions.page_html_hash IS
    '发布时页面HTML的SHA-256十六进制哈希';

-- ============================================================================
-- 四、数据库级不可变保护
-- ============================================================================

CREATE OR REPLACE FUNCTION
    tedna_reject_assistant_deployment_version_update()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION
        'assistant_deployment_versions是不可变发布快照，禁止UPDATE；请追加新版本'
        USING ERRCODE = '55000';
END;
$$;

DROP TRIGGER IF EXISTS
    trg_reject_assistant_deployment_version_update
ON assistant_deployment_versions;

CREATE TRIGGER
    trg_reject_assistant_deployment_version_update
BEFORE UPDATE
ON assistant_deployment_versions
FOR EACH ROW
EXECUTE FUNCTION
    tedna_reject_assistant_deployment_version_update();

-- ============================================================================
-- 五、应用角色最小权限
-- ============================================================================

GRANT
    SELECT,
    INSERT,
    UPDATE,
    DELETE
ON TABLE
    courseware_assistant_slots,
    assistant_deployments
TO tedna_user;

-- 版本表只允许应用读取和追加。
-- 不授予UPDATE或DELETE，进一步保护历史发布快照。
GRANT
    SELECT,
    INSERT
ON TABLE
    assistant_deployment_versions
TO tedna_user;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件教学智能体核心表迁移完成：插槽、部署、不可变版本已创建';
END
$$;
