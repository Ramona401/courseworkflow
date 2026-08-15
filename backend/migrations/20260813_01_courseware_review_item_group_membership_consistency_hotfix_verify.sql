-- ============================================================================
-- TE-DNA 2.0：R-06 membership consistency Hotfix 只读验证
-- 文件：20260813_01_courseware_review_item_group_membership_consistency_hotfix_verify.sql
-- ----------------------------------------------------------------------------
-- 本文件完全只读：
--   1. 检查修复函数仍为SECURITY DEFINER；
--   2. 检查延迟constraint trigger属性；
--   3. 检查函数实现已经改为重新读取当前group最终行；
--   4. 拒绝重新出现NEW.primary_item_id / NEW.status作为最终状态事实；
--   5. 检查现有R-06最终业务数据仍满足主问题、merged目标和成员一致性。
-- ============================================================================

DO $$
DECLARE
    function_definition TEXT;
    function_security_definer BOOLEAN;

    invalid_primary_count BIGINT;
    invalid_merged_member_count BIGINT;
    invalid_merge_target_count BIGINT;
BEGIN
    SELECT
        pg_get_functiondef(proc.oid),
        proc.prosecdef
    INTO
        function_definition,
        function_security_definer
    FROM pg_proc AS proc
    WHERE proc.pronamespace = 'public'::regnamespace
      AND proc.proname =
          'enforce_cw_review_item_group_membership_consistency'
      AND pg_get_function_identity_arguments(proc.oid) = '';

    IF function_definition IS NULL THEN
        RAISE EXCEPTION
            '缺少R-06 membership consistency函数';
    END IF;

    IF NOT function_security_definer THEN
        RAISE EXCEPTION
            'R-06 membership consistency函数必须保持SECURITY DEFINER';
    END IF;

    IF STRPOS(
        function_definition,
        'FROM public.courseware_review_item_groups AS current_group'
    ) = 0
    OR STRPOS(
        function_definition,
        'current_group.id = NEW.id'
    ) = 0
    OR STRPOS(
        function_definition,
        'current_primary_item_id'
    ) = 0
    OR STRPOS(
        function_definition,
        'current_merged_into_group_id'
    ) = 0 THEN
        RAISE EXCEPTION
            'R-06 membership hotfix未使用当前group最终行作为一致性事实';
    END IF;

    IF STRPOS(
        function_definition,
        'member.item_id = NEW.primary_item_id'
    ) > 0
    OR STRPOS(
        function_definition,
        'IF NEW.status = ''merged'''
    ) > 0 THEN
        RAISE EXCEPTION
            'R-06 membership函数仍残留历史NEW状态作为最终一致性事实';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname =
            'trg_cw_review_item_group_membership_consistency'
          AND tgrelid =
            'courseware_review_item_groups'::regclass
          AND tgdeferrable
          AND tginitdeferred
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION
            'R-06 membership consistency constraint trigger属性异常';
    END IF;

    SELECT COUNT(*)
    INTO invalid_primary_count
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

    IF invalid_primary_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条主问题不是当前组active成员的问题组',
            invalid_primary_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_merged_member_count
    FROM courseware_review_item_groups AS review_group
    WHERE review_group.status = 'merged'
      AND EXISTS (
          SELECT 1
          FROM courseware_review_item_group_members AS member
          WHERE member.group_id = review_group.id
            AND member.courseware_id = review_group.courseware_id
            AND member.source_session_id = review_group.source_session_id
            AND member.status = 'active'
      );

    IF invalid_merged_member_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条merged问题组仍保留active成员',
            invalid_merged_member_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_merge_target_count
    FROM courseware_review_item_groups AS review_group
    LEFT JOIN courseware_review_item_groups AS target_group
      ON target_group.id = review_group.merged_into_group_id
     AND target_group.courseware_id = review_group.courseware_id
     AND target_group.source_session_id = review_group.source_session_id
     AND target_group.created_by = review_group.created_by
    WHERE review_group.status = 'merged'
      AND (
          target_group.id IS NULL
          OR target_group.status <> 'active'
      );

    IF invalid_merge_target_count <> 0 THEN
        RAISE EXCEPTION
            '发现%条merged问题组的最终合并目标不存在、越界或非active',
            invalid_merge_target_count;
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
ORDER BY table_name;

SELECT
    tgname AS trigger_name,
    tgdeferrable,
    tginitdeferred,
    pg_get_triggerdef(oid, true) AS trigger_definition
FROM pg_trigger
WHERE tgname =
    'trg_cw_review_item_group_membership_consistency'
  AND tgrelid =
    'courseware_review_item_groups'::regclass
  AND NOT tgisinternal;
