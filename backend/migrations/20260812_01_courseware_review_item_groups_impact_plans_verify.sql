-- ============================================================================
-- TE-DNA 2.0：R-06正式问题组 + R-07影响方案数据库只读验证
-- 文件：20260812_01_courseware_review_item_groups_impact_plans_verify.sql
-- ----------------------------------------------------------------------------
-- 本文件只读，不修改结构或业务数据。
-- 任一关键结构、权限或现有数据一致性不符合预期时RAISE EXCEPTION。
-- ============================================================================

DO $$
DECLARE
    invalid_group_scope_count BIGINT;
    invalid_group_merge_count BIGINT;
    invalid_group_primary_count BIGINT;
    invalid_member_scope_count BIGINT;
    invalid_member_group_count BIGINT;
    invalid_group_event_count BIGINT;
    invalid_group_event_sequence_count BIGINT;

    invalid_plan_source_count BIGINT;
    invalid_plan_hash_count BIGINT;
    invalid_plan_state_count BIGINT;
    invalid_plan_selection_count BIGINT;
    invalid_plan_event_sequence_count BIGINT;
BEGIN
    -- ========================================================================
    -- 一、核心表、约束与触发器
    -- ========================================================================

    IF to_regclass('public.courseware_review_item_groups') IS NULL
       OR to_regclass('public.courseware_review_item_group_members') IS NULL
       OR to_regclass('public.courseware_review_item_group_events') IS NULL
       OR to_regclass('public.courseware_review_impact_plans') IS NULL
       OR to_regclass('public.courseware_review_impact_plan_events') IS NULL THEN
        RAISE EXCEPTION 'R-06/R-07核心表不完整';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'uq_cw_ai_review_session_id_courseware_reviewer'
          AND conrelid = 'courseware_ai_review_sessions'::regclass
    ) THEN
        RAISE EXCEPTION '缺少审核会话ID/课件/审核者复合唯一约束';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_cw_review_item_group_session_scope'
          AND conrelid = 'courseware_review_item_groups'::regclass
    ) OR NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_cw_review_item_group_member_item_scope'
          AND conrelid = 'courseware_review_item_group_members'::regclass
    ) OR NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'uq_cw_review_item_group_event_version'
          AND conrelid = 'courseware_review_item_group_events'::regclass
    ) THEN
        RAISE EXCEPTION 'R-06复合归属或组事件版本约束不完整';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_cw_review_impact_plan_session_scope'
          AND conrelid = 'courseware_review_impact_plans'::regclass
    ) OR NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_cw_review_impact_plan_message_scope'
          AND conrelid = 'courseware_review_impact_plans'::regclass
    ) OR NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'uq_cw_review_impact_plan_event_version'
          AND conrelid = 'courseware_review_impact_plan_events'::regclass
    ) THEN
        RAISE EXCEPTION 'R-07可信来源或方案事件版本约束不完整';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_item_group_mutation_guard'
          AND tgrelid = 'courseware_review_item_groups'::regclass
          AND NOT tgisinternal
    ) OR NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_item_group_member_mutation_guard'
          AND tgrelid = 'courseware_review_item_group_members'::regclass
          AND NOT tgisinternal
    ) OR NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_item_group_event_insert_guard'
          AND tgrelid = 'courseware_review_item_group_events'::regclass
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION 'R-06写入守卫不完整';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_item_group_event_consistency'
          AND tgrelid = 'courseware_review_item_groups'::regclass
          AND tgdeferrable
          AND tginitdeferred
    ) OR NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_item_group_member_event_consistency'
          AND tgrelid = 'courseware_review_item_group_members'::regclass
          AND tgdeferrable
          AND tginitdeferred
    ) OR NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_item_group_membership_consistency'
          AND tgrelid = 'courseware_review_item_groups'::regclass
          AND tgdeferrable
          AND tginitdeferred
    ) THEN
        RAISE EXCEPTION 'R-06延迟一致性守卫不完整';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_impact_plan_mutation_guard'
          AND tgrelid = 'courseware_review_impact_plans'::regclass
          AND NOT tgisinternal
    ) OR NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_impact_plan_event_insert_guard'
          AND tgrelid = 'courseware_review_impact_plan_events'::regclass
          AND NOT tgisinternal
    ) OR NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_cw_review_impact_plan_event_consistency'
          AND tgrelid = 'courseware_review_impact_plans'::regclass
          AND tgdeferrable
          AND tginitdeferred
    ) THEN
        RAISE EXCEPTION 'R-07方案状态与事件守卫不完整';
    END IF;

    -- ========================================================================
    -- 二、R-06现有数据一致性
    -- ========================================================================

    SELECT COUNT(*)
    INTO invalid_group_scope_count
    FROM courseware_review_item_groups AS review_group
    LEFT JOIN courseware_ai_review_sessions AS session
      ON session.id = review_group.source_session_id
     AND session.courseware_id = review_group.courseware_id
     AND session.reviewer_id = review_group.created_by
    WHERE session.id IS NULL;

    IF invalid_group_scope_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条跨课件、跨会话或创建者越界的问题组',
            invalid_group_scope_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_group_merge_count
    FROM courseware_review_item_groups AS review_group
    LEFT JOIN courseware_review_item_groups AS target_group
      ON target_group.id = review_group.merged_into_group_id
     AND target_group.courseware_id = review_group.courseware_id
     AND target_group.source_session_id = review_group.source_session_id
     AND target_group.created_by = review_group.created_by
    WHERE (
        review_group.status = 'active'
        AND review_group.merged_into_group_id IS NOT NULL
    )
    OR (
        review_group.status = 'merged'
        AND (
            review_group.merged_into_group_id IS NULL
            OR review_group.primary_item_id IS NOT NULL
            OR target_group.id IS NULL
            OR target_group.status <> 'active'
            OR EXISTS (
                SELECT 1
                FROM courseware_review_item_group_members AS active_member
                WHERE active_member.group_id = review_group.id
                  AND active_member.status = 'active'
            )
        )
    );

    IF invalid_group_merge_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条合并状态、目标组或残留成员不一致的问题组',
            invalid_group_merge_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_group_primary_count
    FROM courseware_review_item_groups AS review_group
    LEFT JOIN courseware_review_item_group_members AS primary_member
      ON primary_member.group_id = review_group.id
     AND primary_member.courseware_id = review_group.courseware_id
     AND primary_member.source_session_id = review_group.source_session_id
     AND primary_member.item_id = review_group.primary_item_id
     AND primary_member.status = 'active'
    WHERE review_group.status = 'active'
      AND review_group.primary_item_id IS NOT NULL
      AND primary_member.id IS NULL;

    IF invalid_group_primary_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条主问题不是本组有效成员的问题组',
            invalid_group_primary_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_member_scope_count
    FROM courseware_review_item_group_members AS member
    LEFT JOIN courseware_review_item_groups AS review_group
      ON review_group.id = member.group_id
     AND review_group.courseware_id = member.courseware_id
     AND review_group.source_session_id = member.source_session_id
     AND review_group.created_by = member.created_by
    LEFT JOIN courseware_review_items AS item
      ON item.id = member.item_id
     AND item.courseware_id = member.courseware_id
     AND item.source_session_id = member.source_session_id
    WHERE review_group.id IS NULL
       OR item.id IS NULL
       OR (
           item.source_type = 'formal'
           AND item.created_by <> member.created_by
       )
       OR (
           item.source_type = 'self'
           AND item.owner_id <> member.created_by
       )
       OR item.source_type NOT IN ('formal', 'self');

    IF invalid_member_scope_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条问题组成员范围或治理身份不一致',
            invalid_member_scope_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_member_group_count
    FROM courseware_review_item_group_members AS member
    INNER JOIN courseware_review_item_groups AS review_group
      ON review_group.id = member.group_id
    WHERE member.status = 'active'
      AND review_group.status <> 'active';

    IF invalid_member_group_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条active成员仍位于已合并问题组',
            invalid_member_group_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_group_event_count
    FROM courseware_review_item_group_events AS event
    LEFT JOIN courseware_review_item_groups AS review_group
      ON review_group.id = event.group_id
     AND review_group.courseware_id = event.courseware_id
     AND review_group.source_session_id = event.source_session_id
     AND review_group.created_by = event.actor_id
    LEFT JOIN courseware_review_item_group_members AS member
      ON member.id = event.member_id
    WHERE review_group.id IS NULL
       OR (
           event.event_type IN (
               'member_added',
               'member_removed',
               'member_moved'
           )
           AND (
               member.id IS NULL
               OR member.courseware_id <> event.courseware_id
               OR member.source_session_id <> event.source_session_id
               OR member.created_by <> event.actor_id
               OR event.member_version > member.version
           )
       );

    IF invalid_group_event_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条问题组事件范围、成员或成员版本不一致',
            invalid_group_event_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_group_event_sequence_count
    FROM courseware_review_item_groups AS review_group
    LEFT JOIN LATERAL (
        SELECT
            COUNT(*) AS event_count,
            MIN(event.group_version) AS min_version,
            MAX(event.group_version) AS max_version
        FROM courseware_review_item_group_events AS event
        WHERE event.group_id = review_group.id
          AND event.source_session_id = review_group.source_session_id
    ) AS event_stats ON TRUE
    LEFT JOIN courseware_review_item_group_events AS latest_event
      ON latest_event.group_id = review_group.id
     AND latest_event.source_session_id = review_group.source_session_id
     AND latest_event.group_version = review_group.version
    WHERE event_stats.event_count <> review_group.version
       OR event_stats.min_version IS DISTINCT FROM 1
       OR event_stats.max_version IS DISTINCT FROM review_group.version
       OR latest_event.id IS NULL
       OR (
           review_group.version = 1
           AND latest_event.event_type <> 'created'
       )
       OR (
           review_group.status = 'merged'
           AND latest_event.event_type <> 'merged'
       );

    IF invalid_group_event_sequence_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条问题组事件链缺失、断号或与当前状态不一致',
            invalid_group_event_sequence_count;
    END IF;

    -- ========================================================================
    -- 三、R-07现有数据一致性
    -- ========================================================================

    SELECT COUNT(*)
    INTO invalid_plan_source_count
    FROM courseware_review_impact_plans AS plan
    LEFT JOIN courseware_ai_review_messages AS message
      ON message.id = plan.source_message_id
     AND message.session_id = plan.source_session_id
    WHERE message.id IS NULL
       OR message.role <> 'assistant'
       OR message.review_item_id IS NOT NULL;

    IF invalid_plan_source_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条影响方案来源不是同会话全局assistant消息',
            invalid_plan_source_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_plan_hash_count
    FROM courseware_review_impact_plans AS plan
    INNER JOIN courseware_ai_review_messages AS message
      ON message.id = plan.source_message_id
     AND message.session_id = plan.source_session_id
    WHERE plan.operations_hash <>
          public.build_cw_review_impact_operations_hash(
              plan.operations_json
          )
       OR plan.source_message_hash <>
          public.build_cw_review_impact_message_hash(
              message.content,
              message.citations_json
          )
       OR NOT public.is_valid_cw_review_impact_operations(
              plan.operations_json
          );

    IF invalid_plan_hash_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条影响方案操作快照、可信来源哈希或操作协议不一致',
            invalid_plan_hash_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_plan_state_count
    FROM courseware_review_impact_plans
    WHERE (
        status = 'draft'
        AND (
            version <> 1
            OR applied_by IS NOT NULL
            OR applied_at IS NOT NULL
            OR applied_operation_ids_json <> '[]'::jsonb
        )
    )
    OR (
        status = 'applied'
        AND (
            version <> 2
            OR applied_by IS NULL
            OR applied_by <> created_by
            OR applied_at IS NULL
        )
    );

    IF invalid_plan_state_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条影响方案生命周期或应用身份不一致',
            invalid_plan_state_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_plan_selection_count
    FROM courseware_review_impact_plans AS plan
    WHERE plan.status = 'applied'
      AND (
          NOT public.is_valid_cw_review_impact_selection(
              plan.operations_json,
              plan.applied_operation_ids_json
          )
          OR plan.applied_operation_ids_json <>
             public.normalize_cw_review_impact_selection(
                 plan.operations_json,
                 plan.applied_operation_ids_json
             )
      );

    IF invalid_plan_selection_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条影响方案选中操作包含伪造、重复或顺序未规范化',
            invalid_plan_selection_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_plan_event_sequence_count
    FROM courseware_review_impact_plans AS plan
    LEFT JOIN LATERAL (
        SELECT
            COUNT(*) AS event_count,
            MIN(event.plan_version) AS min_version,
            MAX(event.plan_version) AS max_version
        FROM courseware_review_impact_plan_events AS event
        WHERE event.plan_id = plan.id
          AND event.source_session_id = plan.source_session_id
    ) AS event_stats ON TRUE
    LEFT JOIN courseware_review_impact_plan_events AS latest_event
      ON latest_event.plan_id = plan.id
     AND latest_event.source_session_id = plan.source_session_id
     AND latest_event.plan_version = plan.version
    WHERE event_stats.event_count <> plan.version
       OR event_stats.min_version IS DISTINCT FROM 1
       OR event_stats.max_version IS DISTINCT FROM plan.version
       OR latest_event.id IS NULL
       OR (
           plan.status = 'draft'
           AND latest_event.event_type <> 'draft_created'
       )
       OR (
           plan.status = 'applied'
           AND (
               latest_event.event_type <> 'applied'
               OR latest_event.selected_operation_ids_json <>
                  plan.applied_operation_ids_json
           )
       );

    IF invalid_plan_event_sequence_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条影响方案事件链缺失、断号或与当前状态不一致',
            invalid_plan_event_sequence_count;
    END IF;

    -- ========================================================================
    -- 四、应用角色最小权限
    -- ========================================================================

    IF NOT has_table_privilege(
        'tedna_user',
        'courseware_review_item_groups',
        'SELECT'
    )
    OR NOT has_table_privilege(
        'tedna_user',
        'courseware_review_item_groups',
        'INSERT'
    )
    OR has_table_privilege(
        'tedna_user',
        'courseware_review_item_groups',
        'DELETE'
    ) THEN
        RAISE EXCEPTION 'tedna_user问题组表权限不符合最小权限要求';
    END IF;

    IF NOT has_column_privilege(
        'tedna_user',
        'courseware_review_item_groups',
        'version',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_item_groups',
        'courseware_id',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_item_groups',
        'created_by',
        'UPDATE'
    ) THEN
        RAISE EXCEPTION 'tedna_user问题组可变字段权限边界异常';
    END IF;

    IF NOT has_table_privilege(
        'tedna_user',
        'courseware_review_item_group_members',
        'INSERT'
    )
    OR has_table_privilege(
        'tedna_user',
        'courseware_review_item_group_members',
        'DELETE'
    )
    OR NOT has_table_privilege(
        'tedna_user',
        'courseware_review_item_group_events',
        'INSERT'
    )
    OR has_table_privilege(
        'tedna_user',
        'courseware_review_item_group_events',
        'UPDATE'
    )
    OR has_table_privilege(
        'tedna_user',
        'courseware_review_item_group_events',
        'DELETE'
    ) THEN
        RAISE EXCEPTION 'tedna_user问题组成员或事件权限异常';
    END IF;

    IF NOT has_table_privilege(
        'tedna_user',
        'courseware_review_impact_plans',
        'SELECT'
    )
    OR NOT has_table_privilege(
        'tedna_user',
        'courseware_review_impact_plans',
        'INSERT'
    )
    OR has_table_privilege(
        'tedna_user',
        'courseware_review_impact_plans',
        'DELETE'
    ) THEN
        RAISE EXCEPTION 'tedna_user影响方案表权限不符合最小权限要求';
    END IF;

    IF NOT has_column_privilege(
        'tedna_user',
        'courseware_review_impact_plans',
        'status',
        'UPDATE'
    )
    OR NOT has_column_privilege(
        'tedna_user',
        'courseware_review_impact_plans',
        'version',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_impact_plans',
        'operations_json',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_impact_plans',
        'source_message_id',
        'UPDATE'
    ) THEN
        RAISE EXCEPTION 'tedna_user影响方案可变字段权限边界异常';
    END IF;

    IF NOT has_table_privilege(
        'tedna_user',
        'courseware_review_impact_plan_events',
        'INSERT'
    )
    OR has_table_privilege(
        'tedna_user',
        'courseware_review_impact_plan_events',
        'UPDATE'
    )
    OR has_table_privilege(
        'tedna_user',
        'courseware_review_impact_plan_events',
        'DELETE'
    ) THEN
        RAISE EXCEPTION 'tedna_user影响方案事件权限异常';
    END IF;
END
$$;

SELECT
    'courseware_review_item_groups' AS table_name,
    COUNT(*) AS row_count
FROM courseware_review_item_groups
UNION ALL
SELECT
    'courseware_review_item_group_members',
    COUNT(*)
FROM courseware_review_item_group_members
UNION ALL
SELECT
    'courseware_review_item_group_events',
    COUNT(*)
FROM courseware_review_item_group_events
UNION ALL
SELECT
    'courseware_review_impact_plans',
    COUNT(*)
FROM courseware_review_impact_plans
UNION ALL
SELECT
    'courseware_review_impact_plan_events',
    COUNT(*)
FROM courseware_review_impact_plan_events
ORDER BY table_name;

SELECT
    has_table_privilege(
        'tedna_user',
        'courseware_review_item_groups',
        'SELECT'
    ) AS group_select,
    has_table_privilege(
        'tedna_user',
        'courseware_review_item_groups',
        'DELETE'
    ) AS group_delete,
    has_table_privilege(
        'tedna_user',
        'courseware_review_item_group_events',
        'INSERT'
    ) AS group_event_insert,
    has_table_privilege(
        'tedna_user',
        'courseware_review_item_group_events',
        'UPDATE'
    ) AS group_event_update,
    has_table_privilege(
        'tedna_user',
        'courseware_review_impact_plans',
        'INSERT'
    ) AS plan_insert,
    has_column_privilege(
        'tedna_user',
        'courseware_review_impact_plans',
        'status',
        'UPDATE'
    ) AS plan_status_update,
    has_column_privilege(
        'tedna_user',
        'courseware_review_impact_plans',
        'operations_json',
        'UPDATE'
    ) AS plan_operations_update,
    has_table_privilege(
        'tedna_user',
        'courseware_review_impact_plan_events',
        'UPDATE'
    ) AS plan_event_update;

