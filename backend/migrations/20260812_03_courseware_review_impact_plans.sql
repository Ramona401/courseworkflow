-- ============================================================================
-- TE-DNA 2.0：R-07 全局讨论结构化影响方案数据库底座 V1
-- 文件：20260812_03_courseware_review_impact_plans.sql
-- ----------------------------------------------------------------------------
-- 本文件必须紧接R-06结构和守卫迁移，在同一psql连接与同一事务中执行。
-- 目标：
--   1. 冻结可信全局assistant消息、结构化操作、前置条件与操作哈希；
--   2. 每个operation_id由服务端生成，浏览器只能回传选中ID集合；
--   3. draft/version=1只能一次性进入applied/version=2；
--   4. 应用前重新核对可信来源消息哈希；目标业务状态由后端事务重新读取；
--   5. 任一目标过期、变化、越权时，后端应用事务必须整体失败；
--   6. update_candidate_suggestion只更新候选建议，不自动确认当前修改要求；
--   7. draft创建和applied均写追加式不可变事件；
--   8. 本文件最后统一COMMIT R-06 + R-07整个数据库设计阶段。
-- ============================================================================

SAVEPOINT r06_group_guards_ready;

DO $$
BEGIN
    IF to_regclass('public.courseware_review_item_groups') IS NULL
       OR to_regclass('public.courseware_review_item_group_members') IS NULL
       OR to_regclass('public.courseware_review_item_group_events') IS NULL THEN
        RAISE EXCEPTION '必须先在同一事务执行R-06问题组结构与守卫迁移';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_item_group_member_event_consistency'
          AND tgrelid = 'courseware_review_item_group_members'::regclass
          AND tgdeferrable
          AND tginitdeferred
    ) THEN
        RAISE EXCEPTION 'R-06问题组成员事件一致性守卫尚未就绪';
    END IF;
END
$$;

-- ============================================================================
-- 一、R-07 影响方案不可变结构化快照函数
-- ============================================================================

CREATE OR REPLACE FUNCTION public.build_cw_review_impact_message_hash(
    message_content TEXT,
    message_citations JSONB
)
RETURNS TEXT
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT encode(
        digest(
            convert_to(
                jsonb_build_object(
                    'citations_json',
                    COALESCE(message_citations, 'null'::jsonb),
                    'content',
                    COALESCE(message_content, '')
                )::text,
                'UTF8'
            ),
            'sha256'
        ),
        'hex'
    )
$$;

CREATE OR REPLACE FUNCTION public.build_cw_review_impact_operations_hash(
    input_operations JSONB
)
RETURNS TEXT
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT encode(
        digest(
            convert_to(
                COALESCE(input_operations, 'null'::jsonb)::text,
                'UTF8'
            ),
            'sha256'
        ),
        'hex'
    )
$$;

CREATE OR REPLACE FUNCTION public.is_valid_cw_review_impact_operations(
    input_operations JSONB
)
RETURNS BOOLEAN
LANGUAGE plpgsql
IMMUTABLE
PARALLEL SAFE
AS $$
DECLARE
    total_count INTEGER;
    valid_count INTEGER;
    distinct_id_count INTEGER;
BEGIN
    IF COALESCE(jsonb_typeof(input_operations), '') <> 'array' THEN
        RETURN FALSE;
    END IF;

    total_count := jsonb_array_length(input_operations);
    IF total_count < 1 OR total_count > 100 THEN
        RETURN FALSE;
    END IF;

    SELECT
        COUNT(*),
        COUNT(DISTINCT operation ->> 'operation_id')
    INTO
        valid_count,
        distinct_id_count
    FROM jsonb_array_elements(input_operations) AS operation
    WHERE jsonb_typeof(operation) = 'object'
      AND jsonb_typeof(operation -> 'operation_id') = 'string'
      AND (operation ->> 'operation_id') ~
          '^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
      AND jsonb_typeof(operation -> 'operation_type') = 'string'
      AND (operation ->> 'operation_type') IN (
          'create_group',
          'move_group_member',
          'merge_groups',
          'split_group',
          'create_relation',
          'cancel_relation',
          'create_item',
          'dismiss_item',
          'update_candidate_suggestion'
      )
      AND jsonb_typeof(operation -> 'summary') = 'string'
      AND length(BTRIM(operation ->> 'summary')) > 0
      AND jsonb_typeof(operation -> 'payload') = 'object'
      AND jsonb_typeof(operation -> 'preconditions') = 'object';

    RETURN valid_count = total_count
       AND distinct_id_count = total_count;
END
$$;

CREATE OR REPLACE FUNCTION public.is_valid_cw_review_impact_selection(
    input_operations JSONB,
    selected_operation_ids JSONB
)
RETURNS BOOLEAN
LANGUAGE plpgsql
IMMUTABLE
PARALLEL SAFE
AS $$
DECLARE
    selected_count INTEGER;
    valid_selected_count INTEGER;
    distinct_selected_count INTEGER;
BEGIN
    IF NOT public.is_valid_cw_review_impact_operations(input_operations)
       OR COALESCE(jsonb_typeof(selected_operation_ids), '') <> 'array' THEN
        RETURN FALSE;
    END IF;

    selected_count := jsonb_array_length(selected_operation_ids);

    SELECT
        COUNT(*),
        COUNT(DISTINCT selected_id #>> '{}')
    INTO
        valid_selected_count,
        distinct_selected_count
    FROM jsonb_array_elements(selected_operation_ids) AS selected_id
    WHERE jsonb_typeof(selected_id) = 'string'
      AND EXISTS (
          SELECT 1
          FROM jsonb_array_elements(input_operations) AS operation
          WHERE operation ->> 'operation_id' = selected_id #>> '{}'
      );

    RETURN valid_selected_count = selected_count
       AND distinct_selected_count = selected_count;
END
$$;

CREATE OR REPLACE FUNCTION public.normalize_cw_review_impact_selection(
    input_operations JSONB,
    selected_operation_ids JSONB
)
RETURNS JSONB
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT COALESCE(
        jsonb_agg(
            to_jsonb(operation ->> 'operation_id')
            ORDER BY operation_ordinality
        ),
        '[]'::jsonb
    )
    FROM jsonb_array_elements(input_operations)
        WITH ORDINALITY AS source(operation, operation_ordinality)
    WHERE EXISTS (
        SELECT 1
        FROM jsonb_array_elements(selected_operation_ids) AS selected_id
        WHERE selected_id #>> '{}' = operation ->> 'operation_id'
    )
$$;

REVOKE ALL
ON FUNCTION public.build_cw_review_impact_message_hash(TEXT, JSONB)
FROM PUBLIC;

REVOKE ALL
ON FUNCTION public.build_cw_review_impact_operations_hash(JSONB)
FROM PUBLIC;

REVOKE ALL
ON FUNCTION public.is_valid_cw_review_impact_operations(JSONB)
FROM PUBLIC;

REVOKE ALL
ON FUNCTION public.is_valid_cw_review_impact_selection(JSONB, JSONB)
FROM PUBLIC;

REVOKE ALL
ON FUNCTION public.normalize_cw_review_impact_selection(JSONB, JSONB)
FROM PUBLIC;

GRANT EXECUTE
ON FUNCTION public.build_cw_review_impact_message_hash(TEXT, JSONB)
TO tedna_user;

GRANT EXECUTE
ON FUNCTION public.build_cw_review_impact_operations_hash(JSONB)
TO tedna_user;

GRANT EXECUTE
ON FUNCTION public.is_valid_cw_review_impact_operations(JSONB)
TO tedna_user;

GRANT EXECUTE
ON FUNCTION public.is_valid_cw_review_impact_selection(JSONB, JSONB)
TO tedna_user;

GRANT EXECUTE
ON FUNCTION public.normalize_cw_review_impact_selection(JSONB, JSONB)
TO tedna_user;

-- ============================================================================
-- 二、R-07 影响方案与追加式应用事件
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_review_impact_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    courseware_id UUID NOT NULL,
    source_session_id UUID NOT NULL,
    source_message_id UUID NOT NULL,

    status VARCHAR(16) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'applied')),

    -- draft固定version=1；一次成功应用后固定version=2。
    version INTEGER NOT NULL DEFAULT 1
        CHECK (version IN (1, 2)),

    operations_schema_version SMALLINT NOT NULL DEFAULT 1
        CHECK (operations_schema_version = 1),

    operations_json JSONB NOT NULL
        CHECK (
            public.is_valid_cw_review_impact_operations(
                operations_json
            )
        ),

    operations_hash VARCHAR(64) NOT NULL
        CHECK (
            operations_hash =
            public.build_cw_review_impact_operations_hash(
                operations_json
            )
        ),

    -- 来源全局assistant消息的正文+citations_json可信哈希。
    source_message_hash VARCHAR(64) NOT NULL
        CHECK (source_message_hash ~ '^[0-9a-f]{64}$'),

    created_by UUID NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 只在applied时保存教师最终选中的operation_id集合。
    applied_operation_ids_json JSONB NOT NULL DEFAULT '[]'::jsonb,

    applied_by UUID NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    applied_at TIMESTAMPTZ NULL,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cw_review_impact_plan_id_session
        UNIQUE (id, source_session_id),

    CONSTRAINT chk_cw_review_impact_plan_state
        CHECK (
            (
                status = 'draft'
                AND version = 1
                AND applied_by IS NULL
                AND applied_at IS NULL
                AND applied_operation_ids_json = '[]'::jsonb
            )
            OR
            (
                status = 'applied'
                AND version = 2
                AND applied_by IS NOT NULL
                AND applied_at IS NOT NULL
                AND public.is_valid_cw_review_impact_selection(
                    operations_json,
                    applied_operation_ids_json
                )
            )
        ),

    CONSTRAINT fk_cw_review_impact_plan_session_scope
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

    CONSTRAINT fk_cw_review_impact_plan_message_scope
        FOREIGN KEY (
            source_message_id,
            source_session_id
        )
        REFERENCES courseware_ai_review_messages(
            id,
            session_id
        )
        DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_cw_review_impact_plan_session
ON courseware_review_impact_plans(
    source_session_id,
    created_by,
    status,
    created_at DESC
);

CREATE INDEX IF NOT EXISTS idx_cw_review_impact_plan_courseware
ON courseware_review_impact_plans(
    courseware_id,
    created_by,
    status,
    created_at DESC
);

CREATE INDEX IF NOT EXISTS idx_cw_review_impact_plan_message
ON courseware_review_impact_plans(source_message_id);

CREATE TABLE IF NOT EXISTS courseware_review_impact_plan_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    plan_id UUID NOT NULL,
    source_session_id UUID NOT NULL,

    plan_version INTEGER NOT NULL
        CHECK (plan_version IN (1, 2)),

    event_type VARCHAR(20) NOT NULL
        CHECK (
            event_type IN (
                'draft_created',
                'applied'
            )
        ),

    actor_id UUID NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    selected_operation_ids_json JSONB NOT NULL DEFAULT '[]'::jsonb
        CHECK (jsonb_typeof(selected_operation_ids_json) = 'array'),

    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(metadata_json) = 'object'),

    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),

    CONSTRAINT uq_cw_review_impact_plan_event_version
        UNIQUE (plan_id, plan_version),

    CONSTRAINT fk_cw_review_impact_plan_event_plan
        FOREIGN KEY (
            plan_id,
            source_session_id
        )
        REFERENCES courseware_review_impact_plans(
            id,
            source_session_id
        )
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cw_review_impact_plan_event_plan
ON courseware_review_impact_plan_events(
    plan_id,
    plan_version DESC
);

CREATE INDEX IF NOT EXISTS idx_cw_review_impact_plan_event_session
ON courseware_review_impact_plan_events(
    source_session_id,
    actor_id,
    created_at DESC
);

-- ============================================================================
-- 三、R-07 数据库守卫
-- ============================================================================

CREATE OR REPLACE FUNCTION public.guard_cw_review_impact_plan_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    trusted_content TEXT;
    trusted_citations JSONB;
    trusted_hash TEXT;
BEGIN
    SELECT
        message.content,
        message.citations_json
    INTO
        trusted_content,
        trusted_citations
    FROM public.courseware_ai_review_messages AS message
    WHERE message.id = NEW.source_message_id
      AND message.session_id = NEW.source_session_id
      AND message.review_item_id IS NULL
      AND message.role = 'assistant'
    FOR SHARE;

    IF NOT FOUND THEN
        RAISE EXCEPTION
            '影响方案来源必须是同会话全局assistant消息';
    END IF;

    trusted_hash :=
        public.build_cw_review_impact_message_hash(
            trusted_content,
            trusted_citations
        );

    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'draft'
           OR NEW.version <> 1
           OR NEW.operations_schema_version <> 1
           OR NEW.applied_by IS NOT NULL
           OR NEW.applied_at IS NOT NULL
           OR NEW.applied_operation_ids_json <> '[]'::jsonb THEN
            RAISE EXCEPTION
                '新建影响方案必须是未应用的version=1草稿';
        END IF;

        NEW.operations_hash :=
            public.build_cw_review_impact_operations_hash(
                NEW.operations_json
            );
        NEW.source_message_hash := trusted_hash;

        RETURN NEW;
    END IF;

    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.courseware_id IS DISTINCT FROM OLD.courseware_id
       OR NEW.source_session_id IS DISTINCT FROM OLD.source_session_id
       OR NEW.source_message_id IS DISTINCT FROM OLD.source_message_id
       OR NEW.operations_schema_version IS DISTINCT FROM
            OLD.operations_schema_version
       OR NEW.operations_json IS DISTINCT FROM OLD.operations_json
       OR NEW.operations_hash IS DISTINCT FROM OLD.operations_hash
       OR NEW.source_message_hash IS DISTINCT FROM OLD.source_message_hash
       OR NEW.created_by IS DISTINCT FROM OLD.created_by
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION
            '影响方案来源、操作快照、哈希和创建身份不可修改';
    END IF;

    IF OLD.status <> 'draft'
       OR OLD.version <> 1
       OR NEW.status <> 'applied'
       OR NEW.version <> 2 THEN
        RAISE EXCEPTION
            '影响方案只允许从draft/version=1一次性进入applied/version=2';
    END IF;

    IF trusted_hash <> OLD.source_message_hash THEN
        RAISE EXCEPTION
            '影响方案来源AI消息已变化，禁止应用旧方案';
    END IF;

    IF NEW.applied_by IS NULL
       OR NEW.applied_by <> NEW.created_by
       OR NEW.applied_at IS NULL THEN
        RAISE EXCEPTION
            '影响方案必须由原会话审核者明确应用';
    END IF;

    IF NOT public.is_valid_cw_review_impact_selection(
        NEW.operations_json,
        NEW.applied_operation_ids_json
    ) THEN
        RAISE EXCEPTION
            '影响方案选中操作包含伪造、重复或不属于当前方案的operation_id';
    END IF;

    NEW.applied_operation_ids_json :=
        public.normalize_cw_review_impact_selection(
            NEW.operations_json,
            NEW.applied_operation_ids_json
        );

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION public.guard_cw_review_impact_plan_mutation()
FROM PUBLIC;

DROP TRIGGER IF EXISTS trg_cw_review_impact_plan_mutation_guard
ON courseware_review_impact_plans;

CREATE TRIGGER trg_cw_review_impact_plan_mutation_guard
BEFORE INSERT OR UPDATE
ON courseware_review_impact_plans
FOR EACH ROW
EXECUTE FUNCTION public.guard_cw_review_impact_plan_mutation();

CREATE OR REPLACE FUNCTION public.guard_cw_review_impact_plan_event_insert()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    current_status VARCHAR(16);
    current_version INTEGER;
    plan_creator UUID;
    applied_ids JSONB;
    expected_event VARCHAR(20);
BEGIN
    SELECT
        plan.status,
        plan.version,
        plan.created_by,
        plan.applied_operation_ids_json
    INTO
        current_status,
        current_version,
        plan_creator,
        applied_ids
    FROM public.courseware_review_impact_plans AS plan
    WHERE plan.id = NEW.plan_id
      AND plan.source_session_id = NEW.source_session_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION '影响方案事件对应的方案不存在或越界';
    END IF;

    IF NEW.actor_id <> plan_creator THEN
        RAISE EXCEPTION '影响方案事件操作者不是方案所属审核者';
    END IF;

    IF NEW.plan_version <> current_version THEN
        RAISE EXCEPTION '影响方案事件版本与方案当前版本不一致';
    END IF;

    IF current_version = 1
       AND current_status = 'draft' THEN
        expected_event := 'draft_created';

        IF NEW.selected_operation_ids_json <> '[]'::jsonb THEN
            RAISE EXCEPTION '草稿创建事件不能保存已选操作';
        END IF;
    ELSIF current_version = 2
          AND current_status = 'applied' THEN
        expected_event := 'applied';

        IF NEW.selected_operation_ids_json <> applied_ids THEN
            RAISE EXCEPTION
                '影响方案应用事件的选中操作必须与方案最终应用集合一致';
        END IF;
    ELSE
        RAISE EXCEPTION '影响方案当前状态与版本组合无效';
    END IF;

    IF NEW.event_type <> expected_event THEN
        RAISE EXCEPTION '影响方案事件类型与当前状态不一致';
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION public.guard_cw_review_impact_plan_event_insert()
FROM PUBLIC;

DROP TRIGGER IF EXISTS trg_cw_review_impact_plan_event_insert_guard
ON courseware_review_impact_plan_events;

CREATE TRIGGER trg_cw_review_impact_plan_event_insert_guard
BEFORE INSERT
ON courseware_review_impact_plan_events
FOR EACH ROW
EXECUTE FUNCTION public.guard_cw_review_impact_plan_event_insert();

CREATE OR REPLACE FUNCTION public.enforce_cw_review_impact_plan_event_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    expected_event VARCHAR(20);
    matching_event_count BIGINT;
BEGIN
    IF NEW.status = 'draft' AND NEW.version = 1 THEN
        expected_event := 'draft_created';
    ELSIF NEW.status = 'applied' AND NEW.version = 2 THEN
        expected_event := 'applied';
    ELSE
        RAISE EXCEPTION '影响方案状态与版本组合无效';
    END IF;

    SELECT COUNT(*)
    INTO matching_event_count
    FROM public.courseware_review_impact_plan_events AS event
    WHERE event.plan_id = NEW.id
      AND event.source_session_id = NEW.source_session_id
      AND event.plan_version = NEW.version
      AND event.event_type = expected_event;

    IF matching_event_count <> 1 THEN
        RAISE EXCEPTION
            '影响方案version=%缺少唯一匹配的%事件',
            NEW.version,
            expected_event;
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION public.enforce_cw_review_impact_plan_event_consistency()
FROM PUBLIC;

DROP TRIGGER IF EXISTS trg_cw_review_impact_plan_event_consistency
ON courseware_review_impact_plans;

CREATE CONSTRAINT TRIGGER trg_cw_review_impact_plan_event_consistency
AFTER INSERT OR UPDATE
ON courseware_review_impact_plans
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION public.enforce_cw_review_impact_plan_event_consistency();

-- ============================================================================
-- 四、R-07 应用角色最小权限
-- ============================================================================

REVOKE ALL PRIVILEGES ON TABLE courseware_review_impact_plans FROM PUBLIC;
REVOKE ALL PRIVILEGES ON TABLE courseware_review_impact_plans FROM tedna_user;
GRANT SELECT, INSERT ON TABLE courseware_review_impact_plans TO tedna_user;
GRANT UPDATE (
    status, version, applied_operation_ids_json, applied_by, applied_at, updated_at
) ON courseware_review_impact_plans TO tedna_user;

REVOKE ALL PRIVILEGES ON TABLE courseware_review_impact_plan_events FROM PUBLIC;
REVOKE ALL PRIVILEGES ON TABLE courseware_review_impact_plan_events FROM tedna_user;
GRANT SELECT, INSERT ON TABLE courseware_review_impact_plan_events TO tedna_user;

-- ============================================================================
-- 五、数据库注释
-- ============================================================================

COMMENT ON TABLE courseware_review_impact_plans IS
    'R-07结构化影响方案；冻结可信AI来源、操作与前置条件，教师明确选择后一次原子应用';

COMMENT ON COLUMN courseware_review_impact_plans.operations_json IS
    '不可变操作快照；每项含operation_id、operation_type、summary、payload和preconditions';

COMMENT ON COLUMN courseware_review_impact_plans.applied_operation_ids_json IS
    '教师最终明确选择并在同一事务应用的operation_id集合；由数据库按方案顺序规范化';

COMMENT ON TABLE courseware_review_impact_plan_events IS
    '影响方案追加式审计；version=1记录草稿创建，version=2记录教师明确应用';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE 'R-06正式问题组与R-07结构化影响方案数据库底座V1已原子提交';
END
$$;

