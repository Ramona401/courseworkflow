-- ============================================================================
-- TE-DNA 2.0：全局讨论结论落地与问题列表治理 V1.1 验证
-- 文件：20260803_01_courseware_review_item_governance_verify.sql
-- ----------------------------------------------------------------------------
-- 本文件只读取数据库，不修改结构或业务数据。
-- 任一关键结构、权限或数据一致性不符合预期时RAISE EXCEPTION。
-- ============================================================================

DO $$
DECLARE
    invalid_origin_count BIGINT;
    invalid_item_message_scope_count BIGINT;
    invalid_relation_scope_count BIGINT;
    invalid_relation_message_scope_count BIGINT;
    invalid_relation_state_count BIGINT;
    invalid_conflict_order_count BIGINT;
    invalid_event_scope_count BIGINT;
    invalid_event_message_scope_count BIGINT;
    invalid_event_sequence_count BIGINT;
BEGIN
    -- ========================================================================
    -- 一、整改项来源字段和消息归属
    -- ========================================================================

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'uq_cw_ai_review_messages_id_session'
          AND conrelid =
            'courseware_ai_review_messages'::regclass
    ) THEN
        RAISE EXCEPTION
            '缺少审核消息ID与会话ID复合唯一约束';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'courseware_review_items'
          AND column_name = 'origin_type'
          AND is_nullable = 'NO'
    ) THEN
        RAISE EXCEPTION
            'courseware_review_items.origin_type不存在或允许NULL';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'courseware_review_items'
          AND column_name = 'source_global_message_id'
    ) THEN
        RAISE EXCEPTION
            'courseware_review_items.source_global_message_id不存在';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'chk_cw_review_item_origin_source'
          AND conrelid =
            'courseware_review_items'::regclass
    ) THEN
        RAISE EXCEPTION
            '缺少整改项来源类型与来源消息一致性约束';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_item_origin_guard'
          AND tgrelid = 'courseware_review_items'::regclass
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION
            '缺少整改项来源不可变及可信消息守卫';
    END IF;

    SELECT COUNT(*)
    INTO invalid_origin_count
    FROM courseware_review_items
    WHERE (
        origin_type = 'ai_finding'
        AND source_global_message_id IS NOT NULL
    )
    OR (
        origin_type = 'global_discussion_manual'
        AND source_global_message_id IS NULL
    )
    OR origin_type NOT IN (
        'ai_finding',
        'global_discussion_manual'
    );

    IF invalid_origin_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条来源字段不一致的整改项',
            invalid_origin_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_item_message_scope_count
    FROM courseware_review_items AS item
    LEFT JOIN courseware_ai_review_messages AS message
      ON message.id = item.source_global_message_id
    WHERE item.source_global_message_id IS NOT NULL
      AND (
          message.id IS NULL
          OR message.session_id <> item.source_session_id
          OR message.review_item_id IS NOT NULL
          OR message.role <> 'assistant'
      );

    IF invalid_item_message_scope_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条来源消息不是同会话全局assistant消息的整改项',
            invalid_item_message_scope_count;
    END IF;

    -- ========================================================================
    -- 二、关系和事件结构
    -- ========================================================================

    IF to_regclass(
        'public.courseware_review_item_relations'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少courseware_review_item_relations表';
    END IF;

    IF to_regclass(
        'public.courseware_review_item_relation_events'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少courseware_review_item_relation_events表';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'fk_cw_review_item_relation_source'
          AND conrelid =
              'courseware_review_item_relations'::regclass
    ) THEN
        RAISE EXCEPTION
            '缺少关系源整改项复合归属外键';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'fk_cw_review_item_relation_target'
          AND conrelid =
              'courseware_review_item_relations'::regclass
    ) THEN
        RAISE EXCEPTION
            '缺少关系目标整改项复合归属外键';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'chk_cw_review_item_relation_status_version'
          AND conrelid =
              'courseware_review_item_relations'::regclass
    ) THEN
        RAISE EXCEPTION
            '缺少关系状态和版本奇偶一致性约束';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'chk_cw_review_item_relation_event_version_type'
          AND conrelid =
              'courseware_review_item_relation_events'::regclass
    ) THEN
        RAISE EXCEPTION
            '缺少事件版本和类型一致性约束';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'uq_cw_review_item_relation_event_version'
          AND conrelid =
              'courseware_review_item_relation_events'::regclass
    ) THEN
        RAISE EXCEPTION
            '缺少关系事件版本唯一约束';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname =
            'trg_cw_review_item_relation_mutation_guard'
          AND tgrelid =
              'courseware_review_item_relations'::regclass
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION
            '缺少关系身份不可变和状态迁移守卫';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname =
            'trg_cw_review_item_relation_event_insert_guard'
          AND tgrelid =
              'courseware_review_item_relation_events'::regclass
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION
            '缺少关系事件写入守卫';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname =
            'trg_cw_review_item_relation_event_consistency'
          AND tgrelid =
              'courseware_review_item_relations'::regclass
          AND tgdeferrable
          AND tginitdeferred
    ) THEN
        RAISE EXCEPTION
            '缺少延迟的关系与事件原子一致性约束触发器';
    END IF;

    -- ========================================================================
    -- 三、关系和事件现有数据一致性
    -- ========================================================================

    SELECT COUNT(*)
    INTO invalid_relation_scope_count
    FROM courseware_review_item_relations AS relation
    LEFT JOIN courseware_review_items AS source_item
      ON source_item.id = relation.source_item_id
    LEFT JOIN courseware_review_items AS target_item
      ON target_item.id = relation.target_item_id
    WHERE source_item.id IS NULL
       OR target_item.id IS NULL
       OR source_item.courseware_id <> relation.courseware_id
       OR target_item.courseware_id <> relation.courseware_id
       OR source_item.source_session_id <>
            relation.source_session_id
       OR target_item.source_session_id <>
            relation.source_session_id;

    IF invalid_relation_scope_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条跨课件、跨会话或悬空整改项关系',
            invalid_relation_scope_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_relation_message_scope_count
    FROM courseware_review_item_relations AS relation
    LEFT JOIN courseware_ai_review_messages AS message
      ON message.id = relation.source_global_message_id
    WHERE relation.source_global_message_id IS NOT NULL
      AND (
          message.id IS NULL
          OR message.session_id <> relation.source_session_id
          OR message.review_item_id IS NOT NULL
          OR message.role <> 'assistant'
      );

    IF invalid_relation_message_scope_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条来源消息不是同会话全局assistant消息的关系',
            invalid_relation_message_scope_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_relation_state_count
    FROM courseware_review_item_relations
    WHERE (
        status = 'active'
        AND (
            cancelled_by IS NOT NULL
            OR cancelled_at IS NOT NULL
            OR MOD(version, 2) <> 1
        )
    )
    OR (
        status = 'cancelled'
        AND (
            cancelled_by IS NULL
            OR cancelled_at IS NULL
            OR MOD(version, 2) <> 0
        )
    )
    OR version < 1;

    IF invalid_relation_state_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条关系状态、取消信息或版本不一致数据',
            invalid_relation_state_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_conflict_order_count
    FROM courseware_review_item_relations
    WHERE relation_type = 'conflict'
      AND source_item_id::text >= target_item_id::text;

    IF invalid_conflict_order_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条未按规范顺序保存的冲突关系',
            invalid_conflict_order_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_event_scope_count
    FROM courseware_review_item_relation_events AS event
    LEFT JOIN courseware_review_item_relations AS relation
      ON relation.id = event.relation_id
    WHERE relation.id IS NULL
       OR relation.source_session_id <>
            event.source_session_id;

    IF invalid_event_scope_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条悬空或跨会话关系事件',
            invalid_event_scope_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_event_message_scope_count
    FROM courseware_review_item_relation_events AS event
    LEFT JOIN courseware_ai_review_messages AS message
      ON message.id = event.source_global_message_id
    WHERE event.source_global_message_id IS NOT NULL
      AND (
          message.id IS NULL
          OR message.session_id <> event.source_session_id
          OR message.review_item_id IS NOT NULL
          OR message.role <> 'assistant'
      );

    IF invalid_event_message_scope_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条来源消息不是同会话全局assistant消息的关系事件',
            invalid_event_message_scope_count;
    END IF;

    -- 每条关系必须具有从1开始、无缺号、无重复且到当前version结束的事件链。
    -- 当前version对应的事件类型还必须与关系状态一致。
    SELECT COUNT(*)
    INTO invalid_event_sequence_count
    FROM courseware_review_item_relations AS relation
    LEFT JOIN LATERAL (
        SELECT
            COUNT(*) AS event_count,
            MIN(event.relation_version) AS min_version,
            MAX(event.relation_version) AS max_version
        FROM courseware_review_item_relation_events AS event
        WHERE event.relation_id = relation.id
          AND event.source_session_id =
                relation.source_session_id
    ) AS event_stats ON TRUE
    LEFT JOIN courseware_review_item_relation_events AS latest_event
      ON latest_event.relation_id = relation.id
     AND latest_event.source_session_id =
            relation.source_session_id
     AND latest_event.relation_version = relation.version
    WHERE event_stats.event_count <> relation.version
       OR event_stats.min_version IS DISTINCT FROM 1
       OR event_stats.max_version IS DISTINCT FROM
            relation.version
       OR latest_event.id IS NULL
       OR (
           relation.status = 'active'
           AND relation.version = 1
           AND latest_event.event_type <> 'confirmed'
       )
       OR (
           relation.status = 'active'
           AND relation.version > 1
           AND latest_event.event_type <> 'reactivated'
       )
       OR (
           relation.status = 'cancelled'
           AND latest_event.event_type <> 'cancelled'
       );

    IF invalid_event_sequence_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条事件链缺失、断号或与关系状态不一致的关系',
            invalid_event_sequence_count;
    END IF;

    -- ========================================================================
    -- 四、应用角色最小权限
    -- ========================================================================

    IF NOT has_table_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'SELECT'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少关系表SELECT权限';
    END IF;

    IF NOT has_table_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'INSERT'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少关系表INSERT权限';
    END IF;

    IF has_table_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'DELETE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user不应拥有关系表DELETE权限';
    END IF;

    IF NOT has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'status',
        'UPDATE'
    )
    OR NOT has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'version',
        'UPDATE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少关系状态或版本UPDATE权限';
    END IF;

    IF has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'source_item_id',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'target_item_id',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'relation_type',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'source_global_message_id',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'confirmed_at',
        'UPDATE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user不应能修改关系身份、端点、类型或首次来源';
    END IF;

    IF NOT has_table_privilege(
        'tedna_user',
        'courseware_review_item_relation_events',
        'SELECT'
    )
    OR NOT has_table_privilege(
        'tedna_user',
        'courseware_review_item_relation_events',
        'INSERT'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少关系事件SELECT或INSERT权限';
    END IF;

    IF has_table_privilege(
        'tedna_user',
        'courseware_review_item_relation_events',
        'UPDATE'
    )
    OR has_table_privilege(
        'tedna_user',
        'courseware_review_item_relation_events',
        'DELETE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user不应能更新或删除关系事件';
    END IF;
END
$$;

-- ============================================================================
-- 五、输出结构、约束、权限和当前数据概况
-- ============================================================================

SELECT
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'courseware_review_items'
  AND column_name IN (
      'origin_type',
      'source_global_message_id'
  )
ORDER BY ordinal_position;

SELECT
    conrelid::regclass::text AS table_name,
    conname,
    condeferrable,
    condeferred,
    pg_get_constraintdef(oid) AS definition
FROM pg_constraint
WHERE conname IN (
    'uq_cw_ai_review_messages_id_session',
    'chk_cw_review_item_origin_type',
    'chk_cw_review_item_origin_source',
    'fk_cw_review_item_source_global_message_session',
    'uq_cw_review_items_id_courseware_session',
    'fk_cw_review_item_relation_source',
    'fk_cw_review_item_relation_target',
    'fk_cw_review_item_relation_global_message_session',
    'uq_cw_review_item_relation_id_session',
    'chk_cw_review_item_relation_not_self',
    'chk_cw_review_item_conflict_canonical',
    'chk_cw_review_item_relation_cancel_state',
    'chk_cw_review_item_relation_status_version',
    'uq_cw_review_item_relation',
    'fk_cw_review_item_relation_event_relation',
    'fk_cw_review_item_relation_event_global_message_session',
    'chk_cw_review_item_relation_event_version_type',
    'uq_cw_review_item_relation_event_version'
)
ORDER BY table_name, conname;

SELECT
    tgrelid::regclass::text AS table_name,
    tgname,
    tgdeferrable,
    tginitdeferred,
    pg_get_triggerdef(oid) AS definition
FROM pg_trigger
WHERE tgname IN (
    'trg_cw_review_item_origin_guard',
    'trg_cw_review_item_relation_mutation_guard',
    'trg_cw_review_item_relation_event_insert_guard',
    'trg_cw_review_item_relation_event_consistency'
)
ORDER BY table_name, tgname;

SELECT
    origin_type,
    source_type,
    review_level,
    status,
    COUNT(*) AS item_count
FROM courseware_review_items
GROUP BY
    origin_type,
    source_type,
    review_level,
    status
ORDER BY
    origin_type,
    source_type,
    review_level,
    status;

SELECT
    relation_type,
    status,
    version,
    COUNT(*) AS relation_count
FROM courseware_review_item_relations
GROUP BY relation_type, status, version
ORDER BY relation_type, status, version;

SELECT
    event_type,
    relation_version,
    COUNT(*) AS event_count
FROM courseware_review_item_relation_events
GROUP BY event_type, relation_version
ORDER BY relation_version, event_type;

SELECT
    has_table_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'SELECT'
    ) AS relation_select,
    has_table_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'INSERT'
    ) AS relation_insert,
    has_table_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'DELETE'
    ) AS relation_delete,
    has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'status',
        'UPDATE'
    ) AS relation_status_update,
    has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'version',
        'UPDATE'
    ) AS relation_version_update,
    has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'source_item_id',
        'UPDATE'
    ) AS relation_source_update,
    has_column_privilege(
        'tedna_user',
        'courseware_review_item_relations',
        'source_global_message_id',
        'UPDATE'
    ) AS relation_message_update,
    has_table_privilege(
        'tedna_user',
        'courseware_review_item_relation_events',
        'INSERT'
    ) AS event_insert,
    has_table_privilege(
        'tedna_user',
        'courseware_review_item_relation_events',
        'UPDATE'
    ) AS event_update,
    has_table_privilege(
        'tedna_user',
        'courseware_review_item_relation_events',
        'DELETE'
    ) AS event_delete;
