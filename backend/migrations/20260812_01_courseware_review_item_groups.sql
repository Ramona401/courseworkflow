-- ============================================================================
-- TE-DNA 2.0：R-06 正式问题组数据库底座 V1
-- 文件：20260812_01_courseware_review_item_groups.sql
-- ----------------------------------------------------------------------------
-- 目标：
--   1. 建立正式问题组、稳定成员关系、主问题、组版本和追加式事件；
--   2. 支持重命名、成员加入/移除/移动、主问题变更、合并和拆分的审计底座；
--   3. 问题组与既有 pairwise relation 并存，不替代或重解释既有关系；
--   4. 组和成员使用version/CAS语义，为并发移动提供明确冲突基础；
--   5. 已正式交付问题不允许继续改变组成员关系；
--   6. 所有跨表归属由复合外键、触发器和延迟一致性约束兜底；
--   7. 本文件只建立R-06结构，不修改页面、审核决定或既有relation/event；
--   8. 本文件故意不提交事务，必须紧接R-06守卫与R-07迁移并由R-07统一COMMIT。
-- ============================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_roles
        WHERE rolname = 'tedna_user'
    ) THEN
        RAISE EXCEPTION '缺少应用数据库角色tedna_user';
    END IF;

    IF to_regclass('public.courseware_ai_review_sessions') IS NULL
       OR to_regclass('public.courseware_ai_review_messages') IS NULL
       OR to_regclass('public.courseware_review_items') IS NULL THEN
        RAISE EXCEPTION '缺少R-06/R-07所需的课件审核基础表';
    END IF;
END
$$;

-- ============================================================================
-- 一、审核会话的复合归属键
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'uq_cw_ai_review_session_id_courseware_reviewer'
          AND conrelid = 'courseware_ai_review_sessions'::regclass
    ) THEN
        ALTER TABLE courseware_ai_review_sessions
            ADD CONSTRAINT uq_cw_ai_review_session_id_courseware_reviewer
            UNIQUE (id, courseware_id, reviewer_id);
    END IF;
END
$$;

-- ============================================================================
-- 二、R-06 正式问题组
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_review_item_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    courseware_id UUID NOT NULL,
    source_session_id UUID NOT NULL,

    -- 组名必须是教师可理解的教学主题或改进目标。
    -- 数据库只负责非空与长度边界，具体教学语义由Service校验。
    name TEXT NOT NULL
        CHECK (
            length(BTRIM(name)) > 0
            AND length(BTRIM(name)) <= 200
        ),

    primary_item_id UUID NULL,

    status VARCHAR(16) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'merged')),

    -- 创建为1；每次明确的组治理动作精确加1。
    version INTEGER NOT NULL DEFAULT 1
        CHECK (version >= 1),

    -- 仅status=merged时保存目标组；合并不可逆。
    merged_into_group_id UUID NULL,

    created_by UUID NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cw_review_item_group_scope
        UNIQUE (id, courseware_id, source_session_id, created_by),

    CONSTRAINT chk_cw_review_item_group_merge_state
        CHECK (
            (
                status = 'active'
                AND merged_into_group_id IS NULL
            )
            OR
            (
                status = 'merged'
                AND merged_into_group_id IS NOT NULL
                AND primary_item_id IS NULL
            )
        ),

    CONSTRAINT chk_cw_review_item_group_not_self_merge
        CHECK (
            merged_into_group_id IS NULL
            OR merged_into_group_id <> id
        ),

    CONSTRAINT fk_cw_review_item_group_session_scope
        FOREIGN KEY (
            source_session_id,
            courseware_id,
            created_by
        )
        REFERENCES courseware_ai_review_sessions(
            id,
            courseware_id,
            reviewer_id
        )
        ON DELETE CASCADE,

    CONSTRAINT fk_cw_review_item_group_primary_scope
        FOREIGN KEY (
            primary_item_id,
            courseware_id,
            source_session_id
        )
        REFERENCES courseware_review_items(
            id,
            courseware_id,
            source_session_id
        )
        DEFERRABLE INITIALLY DEFERRED,

    CONSTRAINT fk_cw_review_item_group_merge_target_scope
        FOREIGN KEY (
            merged_into_group_id,
            courseware_id,
            source_session_id,
            created_by
        )
        REFERENCES courseware_review_item_groups(
            id,
            courseware_id,
            source_session_id,
            created_by
        )
        DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_cw_review_item_group_session
ON courseware_review_item_groups(
    source_session_id,
    created_by,
    status,
    updated_at DESC
);

CREATE INDEX IF NOT EXISTS idx_cw_review_item_group_courseware
ON courseware_review_item_groups(
    courseware_id,
    created_by,
    status,
    updated_at DESC
);

CREATE INDEX IF NOT EXISTS idx_cw_review_item_group_primary
ON courseware_review_item_groups(primary_item_id)
WHERE primary_item_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS courseware_review_item_group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    group_id UUID NOT NULL,
    courseware_id UUID NOT NULL,
    source_session_id UUID NOT NULL,
    item_id UUID NOT NULL,

    status VARCHAR(16) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'removed')),

    -- 成员身份稳定；移动、移除或恢复时精确加1。
    version INTEGER NOT NULL DEFAULT 1
        CHECK (version >= 1),

    created_by UUID NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    removed_by UUID NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    removed_at TIMESTAMPTZ NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cw_review_item_group_member_item
        UNIQUE (courseware_id, source_session_id, item_id),

    CONSTRAINT uq_cw_review_item_group_member_scope
        UNIQUE (id, courseware_id, source_session_id),

    CONSTRAINT chk_cw_review_item_group_member_remove_state
        CHECK (
            (
                status = 'active'
                AND removed_by IS NULL
                AND removed_at IS NULL
            )
            OR
            (
                status = 'removed'
                AND removed_by IS NOT NULL
                AND removed_at IS NOT NULL
            )
        ),

    CONSTRAINT fk_cw_review_item_group_member_group_scope
        FOREIGN KEY (
            group_id,
            courseware_id,
            source_session_id,
            created_by
        )
        REFERENCES courseware_review_item_groups(
            id,
            courseware_id,
            source_session_id,
            created_by
        )
        DEFERRABLE INITIALLY DEFERRED,

    CONSTRAINT fk_cw_review_item_group_member_item_scope
        FOREIGN KEY (
            item_id,
            courseware_id,
            source_session_id
        )
        REFERENCES courseware_review_items(
            id,
            courseware_id,
            source_session_id
        )
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cw_review_item_group_member_group
ON courseware_review_item_group_members(
    group_id,
    status,
    updated_at DESC
);

CREATE INDEX IF NOT EXISTS idx_cw_review_item_group_member_item
ON courseware_review_item_group_members(
    item_id,
    status
);

CREATE TABLE IF NOT EXISTS courseware_review_item_group_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    group_id UUID NOT NULL,
    courseware_id UUID NOT NULL,
    source_session_id UUID NOT NULL,

    group_version INTEGER NOT NULL
        CHECK (group_version >= 1),

    event_type VARCHAR(32) NOT NULL
        CHECK (
            event_type IN (
                'created',
                'renamed',
                'primary_changed',
                'member_added',
                'member_removed',
                'member_moved',
                'merged',
                'split'
            )
        ),

    actor_id UUID NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    -- 成员类事件绑定稳定成员身份与该次成员CAS版本。
    member_id UUID NULL,
    member_version INTEGER NULL
        CHECK (member_version IS NULL OR member_version >= 1),

    -- move/merge/split可记录另一组，仍须属于同课件、同会话、同治理人。
    related_group_id UUID NULL,

    reason TEXT NOT NULL DEFAULT '',

    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(metadata_json) = 'object'),

    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),

    CONSTRAINT uq_cw_review_item_group_event_version
        UNIQUE (group_id, group_version),

    CONSTRAINT chk_cw_review_item_group_event_member
        CHECK (
            (
                event_type IN (
                    'member_added',
                    'member_removed',
                    'member_moved'
                )
                AND member_id IS NOT NULL
                AND member_version IS NOT NULL
            )
            OR
            (
                event_type NOT IN (
                    'member_added',
                    'member_removed',
                    'member_moved'
                )
                AND member_id IS NULL
                AND member_version IS NULL
            )
        ),

    CONSTRAINT chk_cw_review_item_group_event_related_group
        CHECK (
            event_type NOT IN (
                'member_moved',
                'merged',
                'split'
            )
            OR related_group_id IS NOT NULL
        ),

    CONSTRAINT chk_cw_review_item_group_event_not_self_related
        CHECK (
            related_group_id IS NULL
            OR related_group_id <> group_id
        ),

    CONSTRAINT fk_cw_review_item_group_event_group_scope
        FOREIGN KEY (
            group_id,
            courseware_id,
            source_session_id,
            actor_id
        )
        REFERENCES courseware_review_item_groups(
            id,
            courseware_id,
            source_session_id,
            created_by
        )
        ON DELETE CASCADE,

    CONSTRAINT fk_cw_review_item_group_event_member
        FOREIGN KEY (member_id)
        REFERENCES courseware_review_item_group_members(id)
        DEFERRABLE INITIALLY DEFERRED,

    CONSTRAINT fk_cw_review_item_group_event_related_scope
        FOREIGN KEY (
            related_group_id,
            courseware_id,
            source_session_id,
            actor_id
        )
        REFERENCES courseware_review_item_groups(
            id,
            courseware_id,
            source_session_id,
            created_by
        )
        DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_cw_review_item_group_event_group
ON courseware_review_item_group_events(
    group_id,
    group_version DESC
);

CREATE INDEX IF NOT EXISTS idx_cw_review_item_group_event_session
ON courseware_review_item_group_events(
    source_session_id,
    actor_id,
    created_at DESC
);

CREATE INDEX IF NOT EXISTS idx_cw_review_item_group_event_member
ON courseware_review_item_group_events(member_id)
WHERE member_id IS NOT NULL;

-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'R-06问题组结构完成，等待同事务执行R-06守卫迁移';
END
$$;

-- 本文件故意不执行COMMIT。

