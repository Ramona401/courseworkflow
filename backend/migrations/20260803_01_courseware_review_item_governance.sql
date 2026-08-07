-- ============================================================================
-- TE-DNA 2.0：全局讨论结论落地与问题列表治理 V1.1
-- 文件：20260803_01_courseware_review_item_governance.sql
-- ----------------------------------------------------------------------------
-- 目标：
--   1. 保留source_type=self/formal的审核用途与权限语义；
--   2. 独立标记整改项来自AI finding或全局讨论人工新增；
--   3. 持久化重复、冲突、合并、依赖和可能连带解决关系；
--   4. 所有关系必须由用户明确确认，AI不能自行落库；
--   5. 关系确认、取消和重新启用使用追加式不可变事件；
--   6. 关系两端必须属于同一课件和同一AI审核会话；
--   7. 来源消息必须是同一会话中的全局assistant消息；
--   8. 关系状态和事件版本由数据库延迟约束保证原子一致；
--   9. 不修改页面，不提交审核决定，不改写正式历史反馈。
--
-- 关系方向：
--   duplicate：
--       source_item_id重复target_item_id，target为保留主问题；
--   conflict：
--       无方向关系，两个ID按UUID文本升序保存；
--   merge：
--       source_item_id合并进入target_item_id；
--   dependency：
--       source_item_id依赖target_item_id先完成；
--   possibly_resolved：
--       source_item_id可能被target_item_id的修改连带解决。
--
-- 生命周期：
--   version=1：active，必须写confirmed事件；
--   version=2：cancelled，必须写cancelled事件；
--   version=3：active，必须写reactivated事件；
--   后续继续按奇数active、偶数cancelled交替。
--
-- 每次状态迁移必须：
--   1. 锁定关系；
--   2. 按旧version执行CAS更新；
--   3. version精确加1；
--   4. 在同一事务写入相同relation_version的事件；
--   5. 否则延迟约束在COMMIT时拒绝事务。
-- ============================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- 一、审核消息复合会话引用键
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'uq_cw_ai_review_messages_id_session'
          AND conrelid =
            'courseware_ai_review_messages'::regclass
    ) THEN
        ALTER TABLE courseware_ai_review_messages
            ADD CONSTRAINT
                uq_cw_ai_review_messages_id_session
            UNIQUE (id, session_id);
    END IF;
END
$$;

-- ============================================================================
-- 二、整改项来源追踪
-- ============================================================================

ALTER TABLE courseware_review_items
    ADD COLUMN IF NOT EXISTS origin_type VARCHAR(32)
        NOT NULL DEFAULT 'ai_finding';

ALTER TABLE courseware_review_items
    ADD COLUMN IF NOT EXISTS source_global_message_id UUID NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_cw_review_item_origin_type'
          AND conrelid = 'courseware_review_items'::regclass
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT chk_cw_review_item_origin_type
            CHECK (
                origin_type IN (
                    'ai_finding',
                    'global_discussion_manual'
                )
            );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'chk_cw_review_item_origin_source'
          AND conrelid = 'courseware_review_items'::regclass
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT chk_cw_review_item_origin_source
            CHECK (
                (
                    origin_type = 'ai_finding'
                    AND source_global_message_id IS NULL
                )
                OR
                (
                    origin_type = 'global_discussion_manual'
                    AND source_global_message_id IS NOT NULL
                )
            );
    END IF;
END
$$;

ALTER TABLE courseware_review_items
    DROP CONSTRAINT IF EXISTS
        fk_cw_review_item_source_global_message;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'fk_cw_review_item_source_global_message_session'
          AND conrelid = 'courseware_review_items'::regclass
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT
                fk_cw_review_item_source_global_message_session
            FOREIGN KEY (
                source_global_message_id,
                source_session_id
            )
            REFERENCES courseware_ai_review_messages(
                id,
                session_id
            )
            DEFERRABLE INITIALLY DEFERRED;
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_source_global_message
ON courseware_review_items(source_global_message_id)
WHERE source_global_message_id IS NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'uq_cw_review_items_id_courseware_session'
          AND conrelid = 'courseware_review_items'::regclass
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT
                uq_cw_review_items_id_courseware_session
            UNIQUE (
                id,
                courseware_id,
                source_session_id
            );
    END IF;
END
$$;

-- 人工新增整改项只能在INSERT时确定来源。
-- 创建后禁止改变origin_type、来源消息和来源会话。
-- 同时复核来源必须是本会话的全局assistant消息。
CREATE OR REPLACE FUNCTION
    public.guard_cw_review_item_origin()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF NEW.origin_type IS DISTINCT FROM OLD.origin_type
           OR NEW.source_global_message_id IS DISTINCT FROM
                OLD.source_global_message_id
           OR NEW.source_session_id IS DISTINCT FROM
                OLD.source_session_id THEN
            RAISE EXCEPTION
                '整改项来源、来源消息和来源会话创建后不可修改';
        END IF;
    END IF;

    IF NEW.origin_type = 'ai_finding' THEN
        IF NEW.source_global_message_id IS NOT NULL THEN
            RAISE EXCEPTION
                'AI finding整改项不能绑定全局讨论来源消息';
        END IF;

        RETURN NEW;
    END IF;

    IF NEW.origin_type <> 'global_discussion_manual'
       OR NEW.source_global_message_id IS NULL THEN
        RAISE EXCEPTION
            '全局讨论人工新增整改项必须绑定来源消息';
    END IF;

    PERFORM 1
    FROM public.courseware_ai_review_messages AS message
    WHERE message.id = NEW.source_global_message_id
      AND message.session_id = NEW.source_session_id
      AND message.review_item_id IS NULL
      AND message.role = 'assistant';

    IF NOT FOUND THEN
        RAISE EXCEPTION
            '整改项来源消息不是同会话的全局assistant消息';
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION public.guard_cw_review_item_origin()
FROM PUBLIC;

DROP TRIGGER IF EXISTS
    trg_cw_review_item_origin_guard
ON courseware_review_items;

CREATE TRIGGER trg_cw_review_item_origin_guard
BEFORE INSERT
    OR UPDATE OF
        origin_type,
        source_global_message_id,
        source_session_id
ON courseware_review_items
FOR EACH ROW
EXECUTE FUNCTION public.guard_cw_review_item_origin();

COMMENT ON COLUMN courseware_review_items.origin_type IS
    '整改项产生方式：ai_finding=AI审核发现；global_discussion_manual=全局讨论中人工确认新增';

COMMENT ON COLUMN
    courseware_review_items.source_global_message_id IS
    '人工新增整改项所依据的同会话全局assistant消息；创建后不可修改';

-- ============================================================================
-- 三、整改项结构化关系
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_review_item_relations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    courseware_id UUID NOT NULL,
    source_session_id UUID NOT NULL,

    source_item_id UUID NOT NULL,
    target_item_id UUID NOT NULL,

    relation_type VARCHAR(32) NOT NULL
        CHECK (
            relation_type IN (
                'duplicate',
                'conflict',
                'merge',
                'dependency',
                'possibly_resolved'
            )
        ),

    status VARCHAR(16) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'cancelled')),

    -- 每次状态迁移精确加1。
    version INTEGER NOT NULL DEFAULT 1
        CHECK (version >= 1),

    explanation TEXT NOT NULL DEFAULT ''
        CHECK (length(btrim(explanation)) > 0),

    -- 允许为空：用户也可以在具体页面问题列表中直接建立关系。
    source_global_message_id UUID NULL,

    created_by UUID NOT NULL
        REFERENCES users(id)
        ON DELETE RESTRICT,

    confirmed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    cancelled_by UUID NULL
        REFERENCES users(id)
        ON DELETE RESTRICT,

    cancelled_at TIMESTAMPTZ NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cw_review_item_relation_courseware
        FOREIGN KEY (courseware_id)
        REFERENCES coursewares(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_cw_review_item_relation_session
        FOREIGN KEY (source_session_id)
        REFERENCES courseware_ai_review_sessions(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_cw_review_item_relation_source
        FOREIGN KEY (
            source_item_id,
            courseware_id,
            source_session_id
        )
        REFERENCES courseware_review_items(
            id,
            courseware_id,
            source_session_id
        )
        ON DELETE CASCADE,

    CONSTRAINT fk_cw_review_item_relation_target
        FOREIGN KEY (
            target_item_id,
            courseware_id,
            source_session_id
        )
        REFERENCES courseware_review_items(
            id,
            courseware_id,
            source_session_id
        )
        ON DELETE CASCADE,

    CONSTRAINT
        fk_cw_review_item_relation_global_message_session
        FOREIGN KEY (
            source_global_message_id,
            source_session_id
        )
        REFERENCES courseware_ai_review_messages(
            id,
            session_id
        )
        DEFERRABLE INITIALLY DEFERRED,

    CONSTRAINT uq_cw_review_item_relation_id_session
        UNIQUE (id, source_session_id),

    CONSTRAINT chk_cw_review_item_relation_not_self
        CHECK (source_item_id <> target_item_id),

    CONSTRAINT chk_cw_review_item_conflict_canonical
        CHECK (
            relation_type <> 'conflict'
            OR source_item_id::text < target_item_id::text
        ),

    CONSTRAINT chk_cw_review_item_relation_cancel_state
        CHECK (
            (
                status = 'active'
                AND cancelled_by IS NULL
                AND cancelled_at IS NULL
            )
            OR
            (
                status = 'cancelled'
                AND cancelled_by IS NOT NULL
                AND cancelled_at IS NOT NULL
            )
        ),

    -- 初始active为奇数版本，cancelled为偶数版本。
    CONSTRAINT chk_cw_review_item_relation_status_version
        CHECK (
            (
                status = 'active'
                AND MOD(version, 2) = 1
            )
            OR
            (
                status = 'cancelled'
                AND MOD(version, 2) = 0
            )
        ),

    CONSTRAINT uq_cw_review_item_relation
        UNIQUE (
            courseware_id,
            source_session_id,
            relation_type,
            source_item_id,
            target_item_id
        )
);

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_relation_session
ON courseware_review_item_relations(
    source_session_id,
    status,
    updated_at DESC
);

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_relation_courseware
ON courseware_review_item_relations(
    courseware_id,
    status,
    updated_at DESC
);

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_relation_source
ON courseware_review_item_relations(
    source_item_id,
    status,
    relation_type
);

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_relation_target
ON courseware_review_item_relations(
    target_item_id,
    status,
    relation_type
);

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_relation_global_message
ON courseware_review_item_relations(source_global_message_id)
WHERE source_global_message_id IS NOT NULL;

COMMENT ON TABLE courseware_review_item_relations IS
    '经过人工确认的整改项结构化关系；状态使用version和追加式事件原子治理';

COMMENT ON COLUMN
    courseware_review_item_relations.source_item_id IS
    '关系源整改项；duplicate、merge、dependency和possibly_resolved均按方向解释';

COMMENT ON COLUMN
    courseware_review_item_relations.target_item_id IS
    '关系目标整改项；duplicate和merge通常以target作为保留主问题';

COMMENT ON COLUMN
    courseware_review_item_relations.version IS
    '关系治理版本；创建为1，每次取消或重新启用精确加1';

-- ============================================================================
-- 四、关系治理追加式事件
-- ============================================================================

CREATE TABLE IF NOT EXISTS
    courseware_review_item_relation_events (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

        relation_id UUID NOT NULL,
        source_session_id UUID NOT NULL,

        -- 与关系当前version完全一致。
        relation_version INTEGER NOT NULL
            CHECK (relation_version >= 1),

        event_type VARCHAR(20) NOT NULL
            CHECK (
                event_type IN (
                    'confirmed',
                    'cancelled',
                    'reactivated'
                )
            ),

        actor_id UUID NOT NULL
            REFERENCES users(id)
            ON DELETE RESTRICT,

        reason TEXT NOT NULL DEFAULT '',

        source_global_message_id UUID NULL,

        metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb
            CHECK (jsonb_typeof(metadata_json) = 'object'),

        created_at TIMESTAMPTZ NOT NULL DEFAULT
            clock_timestamp(),

        CONSTRAINT fk_cw_review_item_relation_event_relation
            FOREIGN KEY (
                relation_id,
                source_session_id
            )
            REFERENCES courseware_review_item_relations(
                id,
                source_session_id
            )
            ON DELETE CASCADE,

        CONSTRAINT
            fk_cw_review_item_relation_event_global_message_session
            FOREIGN KEY (
                source_global_message_id,
                source_session_id
            )
            REFERENCES courseware_ai_review_messages(
                id,
                session_id
            )
            DEFERRABLE INITIALLY DEFERRED,

        CONSTRAINT chk_cw_review_item_relation_event_version_type
            CHECK (
                (
                    relation_version = 1
                    AND event_type = 'confirmed'
                )
                OR
                (
                    relation_version > 1
                    AND MOD(relation_version, 2) = 0
                    AND event_type = 'cancelled'
                )
                OR
                (
                    relation_version > 1
                    AND MOD(relation_version, 2) = 1
                    AND event_type = 'reactivated'
                )
            ),

        CONSTRAINT uq_cw_review_item_relation_event_version
            UNIQUE (
                relation_id,
                relation_version
            )
    );

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_relation_event_relation
ON courseware_review_item_relation_events(
    relation_id,
    relation_version DESC
);

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_relation_event_session
ON courseware_review_item_relation_events(
    source_session_id,
    created_at DESC
);

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_relation_event_actor
ON courseware_review_item_relation_events(
    actor_id,
    created_at DESC
);

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_relation_event_global_message
ON courseware_review_item_relation_events(
    source_global_message_id
)
WHERE source_global_message_id IS NOT NULL;

COMMENT ON TABLE courseware_review_item_relation_events IS
    '整改项关系追加式治理历史；relation_version确定事件顺序，不依赖时间戳或UUID排序';

-- ============================================================================
-- 五、关系与事件数据库状态机
-- ============================================================================

-- 关系首次写入必须为active/version=1。
-- 后续更新只能在active和cancelled之间切换，并精确增加一个版本。
-- 关系身份、方向、类型、创建者和首次来源均不可修改。
CREATE OR REPLACE FUNCTION
    public.guard_cw_review_item_relation_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'active'
           OR NEW.version <> 1
           OR NEW.cancelled_by IS NOT NULL
           OR NEW.cancelled_at IS NOT NULL THEN
            RAISE EXCEPTION
                '新建整改项关系必须是active且version=1';
        END IF;
    ELSE
        IF NEW.id IS DISTINCT FROM OLD.id
           OR NEW.courseware_id IS DISTINCT FROM OLD.courseware_id
           OR NEW.source_session_id IS DISTINCT FROM
                OLD.source_session_id
           OR NEW.source_item_id IS DISTINCT FROM OLD.source_item_id
           OR NEW.target_item_id IS DISTINCT FROM OLD.target_item_id
           OR NEW.relation_type IS DISTINCT FROM OLD.relation_type
           OR NEW.source_global_message_id IS DISTINCT FROM
                OLD.source_global_message_id
           OR NEW.created_by IS DISTINCT FROM OLD.created_by
           OR NEW.confirmed_at IS DISTINCT FROM OLD.confirmed_at
           OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
            RAISE EXCEPTION
                '整改项关系身份、方向、类型和首次来源创建后不可修改';
        END IF;

        IF NEW.version <> OLD.version + 1 THEN
            RAISE EXCEPTION
                '整改项关系更新必须将version精确加1';
        END IF;

        IF NOT (
            (
                OLD.status = 'active'
                AND NEW.status = 'cancelled'
            )
            OR
            (
                OLD.status = 'cancelled'
                AND NEW.status = 'active'
            )
        ) THEN
            RAISE EXCEPTION
                '整改项关系只能在active和cancelled之间切换';
        END IF;
    END IF;

    IF NEW.source_global_message_id IS NOT NULL THEN
        PERFORM 1
        FROM public.courseware_ai_review_messages AS message
        WHERE message.id = NEW.source_global_message_id
          AND message.session_id = NEW.source_session_id
          AND message.review_item_id IS NULL
          AND message.role = 'assistant';

        IF NOT FOUND THEN
            RAISE EXCEPTION
                '整改项关系来源消息不是同会话的全局assistant消息';
        END IF;
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION public.guard_cw_review_item_relation_mutation()
FROM PUBLIC;

DROP TRIGGER IF EXISTS
    trg_cw_review_item_relation_mutation_guard
ON courseware_review_item_relations;

CREATE TRIGGER
    trg_cw_review_item_relation_mutation_guard
BEFORE INSERT OR UPDATE
ON courseware_review_item_relations
FOR EACH ROW
EXECUTE FUNCTION
    public.guard_cw_review_item_relation_mutation();

-- 事件写入时锁定关系，要求事件版本等于关系当前版本，
-- 并复核事件类型和关系当前状态一致。
CREATE OR REPLACE FUNCTION
    public.guard_cw_review_item_relation_event_insert()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    current_status VARCHAR(16);
    current_version INTEGER;
    expected_event VARCHAR(20);
BEGIN
    SELECT
        relation.status,
        relation.version
    INTO
        current_status,
        current_version
    FROM public.courseware_review_item_relations AS relation
    WHERE relation.id = NEW.relation_id
      AND relation.source_session_id = NEW.source_session_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION
            '整改项关系不存在或不属于指定审核会话';
    END IF;

    IF NEW.relation_version <> current_version THEN
        RAISE EXCEPTION
            '关系事件版本与关系当前版本不一致';
    END IF;

    IF current_version = 1 THEN
        expected_event := 'confirmed';
    ELSIF current_status = 'cancelled' THEN
        expected_event := 'cancelled';
    ELSE
        expected_event := 'reactivated';
    END IF;

    IF NEW.event_type <> expected_event THEN
        RAISE EXCEPTION
            '关系事件类型与关系当前状态不一致';
    END IF;

    IF NEW.event_type = 'cancelled'
       AND length(btrim(NEW.reason)) = 0 THEN
        RAISE EXCEPTION
            '取消整改项关系必须填写原因';
    END IF;

    IF NEW.source_global_message_id IS NOT NULL THEN
        PERFORM 1
        FROM public.courseware_ai_review_messages AS message
        WHERE message.id = NEW.source_global_message_id
          AND message.session_id = NEW.source_session_id
          AND message.review_item_id IS NULL
          AND message.role = 'assistant';

        IF NOT FOUND THEN
            RAISE EXCEPTION
                '关系事件来源消息不是同会话的全局assistant消息';
        END IF;
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION
    public.guard_cw_review_item_relation_event_insert()
FROM PUBLIC;

DROP TRIGGER IF EXISTS
    trg_cw_review_item_relation_event_insert_guard
ON courseware_review_item_relation_events;

CREATE TRIGGER
    trg_cw_review_item_relation_event_insert_guard
BEFORE INSERT
ON courseware_review_item_relation_events
FOR EACH ROW
EXECUTE FUNCTION
    public.guard_cw_review_item_relation_event_insert();

-- 延迟到事务提交时检查：
-- 每条关系当前version必须存在且只能存在一条对应事件。
CREATE OR REPLACE FUNCTION
    public.enforce_cw_review_item_relation_event_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    expected_event VARCHAR(20);
    matching_event_count BIGINT;
BEGIN
    IF NEW.version = 1 THEN
        expected_event := 'confirmed';
    ELSIF NEW.status = 'cancelled' THEN
        expected_event := 'cancelled';
    ELSE
        expected_event := 'reactivated';
    END IF;

    SELECT COUNT(*)
    INTO matching_event_count
    FROM public.courseware_review_item_relation_events AS event
    WHERE event.relation_id = NEW.id
      AND event.source_session_id = NEW.source_session_id
      AND event.relation_version = NEW.version
      AND event.event_type = expected_event;

    IF matching_event_count <> 1 THEN
        RAISE EXCEPTION
            '整改项关系version=%缺少唯一匹配的%事件',
            NEW.version,
            expected_event;
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION
    public.enforce_cw_review_item_relation_event_consistency()
FROM PUBLIC;

DROP TRIGGER IF EXISTS
    trg_cw_review_item_relation_event_consistency
ON courseware_review_item_relations;

CREATE CONSTRAINT TRIGGER
    trg_cw_review_item_relation_event_consistency
AFTER INSERT OR UPDATE
ON courseware_review_item_relations
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION
    public.enforce_cw_review_item_relation_event_consistency();

-- ============================================================================
-- 六、应用角色最小权限
-- ============================================================================

REVOKE ALL PRIVILEGES
ON TABLE courseware_review_item_relations
FROM PUBLIC;

REVOKE ALL PRIVILEGES
ON TABLE courseware_review_item_relations
FROM tedna_user;

GRANT
    SELECT,
    INSERT
ON TABLE courseware_review_item_relations
TO tedna_user;

GRANT UPDATE (
    status,
    version,
    explanation,
    cancelled_by,
    cancelled_at,
    updated_at
)
ON courseware_review_item_relations
TO tedna_user;

REVOKE ALL PRIVILEGES
ON TABLE courseware_review_item_relation_events
FROM PUBLIC;

REVOKE ALL PRIVILEGES
ON TABLE courseware_review_item_relation_events
FROM tedna_user;

GRANT
    SELECT,
    INSERT
ON TABLE courseware_review_item_relation_events
TO tedna_user;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件审核问题列表治理V1.1数据库底座迁移完成';
END
$$;
