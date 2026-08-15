-- ============================================================================
-- TE-DNA 2.0：作者自审修改完成后三项人工决策数据库守卫验证
-- 文件：20260809_01_courseware_review_self_post_apply_verify.sql
-- ----------------------------------------------------------------------------
-- 本文件只读验证结构与现有数据，不修改业务数据。
-- 任意核心检查失败都会抛出异常。
-- ============================================================================

\pset pager off

DO $$
DECLARE
    constraint_definition TEXT;
BEGIN
    SELECT pg_get_constraintdef(con.oid)
    INTO constraint_definition
    FROM pg_constraint AS con
    WHERE con.conrelid =
            'public.courseware_review_items'::regclass
      AND con.conname =
            'chk_cw_review_item_applied_instruction_version';

    IF constraint_definition IS NULL THEN
        RAISE EXCEPTION
            '缺少应用版本事实约束';
    END IF;

    IF POSITION(
        'dismissed'
        IN constraint_definition
    ) = 0 THEN
        RAISE EXCEPTION
            '应用版本事实约束尚未允许self dismissed保留完成事实';
    END IF;

    IF POSITION(
        'source_type'
        IN constraint_definition
    ) = 0 THEN
        RAISE EXCEPTION
            '应用版本事实约束没有限制dismissed来源类型';
    END IF;
END
$$;

DO $$
DECLARE
    trigger_enabled CHAR;
    trigger_when TEXT;
BEGIN
    SELECT
        trg.tgenabled,
        trg.tgqual::text
    INTO
        trigger_enabled,
        trigger_when
    FROM pg_trigger AS trg
    WHERE trg.tgrelid =
            'public.courseware_review_items'::regclass
      AND trg.tgname =
            'trg_cw_review_item_instruction_binding_guard'
      AND NOT trg.tgisinternal;

    IF trigger_enabled IS NULL THEN
        RAISE EXCEPTION
            '缺少既有课件整改指令绑定守卫触发器';
    END IF;

    IF trigger_enabled NOT IN ('O', 'A') THEN
        RAISE EXCEPTION
            '既有绑定守卫触发器未启用，状态=%',
            trigger_enabled;
    END IF;

    IF trigger_when IS NULL
       OR BTRIM(trigger_when) = '' THEN
        RAISE EXCEPTION
            '既有绑定守卫没有排除专用dismissed恢复路径';
    END IF;
END
$$;

DO $$
DECLARE
    trigger_enabled CHAR;
    trigger_when TEXT;
BEGIN
    SELECT
        trg.tgenabled,
        trg.tgqual::text
    INTO
        trigger_enabled,
        trigger_when
    FROM pg_trigger AS trg
    WHERE trg.tgrelid =
            'public.courseware_review_items'::regclass
      AND trg.tgname =
            'trg_01_cw_review_item_self_applied_restore_guard'
      AND NOT trg.tgisinternal;

    IF trigger_enabled IS NULL THEN
        RAISE EXCEPTION
            '缺少作者自审修改完成恢复专用触发器';
    END IF;

    IF trigger_enabled NOT IN ('O', 'A') THEN
        RAISE EXCEPTION
            '作者自审恢复专用触发器未启用，状态=%',
            trigger_enabled;
    END IF;

    IF trigger_when IS NULL
       OR BTRIM(trigger_when) = '' THEN
        RAISE EXCEPTION
            '作者自审恢复专用触发器缺少WHEN边界';
    END IF;
END
$$;

DO $$
DECLARE
    security_definer BOOLEAN;
    guard_definition TEXT;
BEGIN
    SELECT
        proc.prosecdef,
        pg_get_functiondef(proc.oid)
    INTO
        security_definer,
        guard_definition
    FROM pg_proc AS proc
    JOIN pg_namespace AS ns
      ON ns.oid = proc.pronamespace
    WHERE ns.nspname = 'public'
      AND proc.proname =
            'guard_cw_review_item_self_applied_restore'
      AND pg_get_function_identity_arguments(
            proc.oid
          ) = '';

    IF security_definer IS NULL THEN
        RAISE EXCEPTION
            '缺少作者自审修改完成恢复守卫函数';
    END IF;

    IF NOT security_definer THEN
        RAISE EXCEPTION
            '作者自审修改完成恢复守卫必须使用SECURITY DEFINER';
    END IF;

    IF POSITION(
        'page_number_snapshot'
        IN guard_definition
    ) = 0 THEN
        RAISE EXCEPTION
            '作者自审恢复守卫没有识别删除页面的历史页码事实';
    END IF;

    IF POSITION(
        'to_jsonb'
        IN guard_definition
    ) = 0 THEN
        RAISE EXCEPTION
            '作者自审恢复守卫没有限制恢复动作只能改变状态';
    END IF;
END
$$;

DO $$
BEGIN
    -- applying必须已经绑定版本且尚未形成新的完成时间。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'applying'
          AND (
                applied_instruction_version_id IS NULL
                OR applied_at IS NOT NULL
          )
    ) THEN
        RAISE EXCEPTION
            '存在非法applying应用事实组合';
    END IF;

    -- 带applied事实的dismissed只能是作者自审。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'dismissed'
          AND applied_instruction_version_id IS NOT NULL
          AND (
                source_type <> 'self'
                OR applied_at IS NULL
          )
    ) THEN
        RAISE EXCEPTION
            '存在非法dismissed修改完成事实';
    END IF;

    -- 普通dismissed不应出现只有一半的applied事实。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'dismissed'
          AND (
                (
                    applied_instruction_version_id IS NULL
                    AND applied_at IS NOT NULL
                )
                OR
                (
                    applied_instruction_version_id IS NOT NULL
                    AND applied_at IS NULL
                )
          )
    ) THEN
        RAISE EXCEPTION
            '存在不完整的dismissed应用事实';
    END IF;

    -- 带applied事实的self dismissed必须仍绑定当前确认版本。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'dismissed'
          AND applied_instruction_version_id IS NOT NULL
          AND (
                current_instruction_version_id IS NULL
                OR applied_instruction_version_id <>
                    current_instruction_version_id
                OR BTRIM(
                    COALESCE(
                        applied_page_hash,
                        ''
                    )
                ) = ''
          )
    ) THEN
        RAISE EXCEPTION
            '存在无法安全恢复的self dismissed记录';
    END IF;

    -- 页级dismissed若page_id已经因删除而变空，不允许被解释为整课问题。
    --
    -- 该状态允许存在，因为后端会在下一次恢复/继续动作时原子收敛为orphaned；
    -- 数据库专用恢复守卫本身也必须拒绝dismissed -> applied。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'dismissed'
          AND applied_instruction_version_id IS NOT NULL
          AND page_id IS NULL
          AND page_number_snapshot > 0
          AND source_type <> 'self'
    ) THEN
        RAISE EXCEPTION
            '存在非自审的删除页面dismissed记录';
    END IF;
END
$$;

SELECT
    source_type,
    status,
    COUNT(*) AS item_count,
    COUNT(*) FILTER (
        WHERE applied_instruction_version_id IS NOT NULL
    ) AS with_applied_version,
    COUNT(*) FILTER (
        WHERE applied_at IS NOT NULL
    ) AS with_applied_time
FROM courseware_review_items
GROUP BY
    source_type,
    status
ORDER BY
    source_type,
    status;

SELECT
    COUNT(*) AS dismissed_total,
    COUNT(*) FILTER (
        WHERE source_type = 'self'
          AND applied_instruction_version_id IS NOT NULL
          AND applied_at IS NOT NULL
    ) AS self_dismissed_with_applied_fact,
    COUNT(*) FILTER (
        WHERE source_type = 'self'
          AND applied_instruction_version_id IS NOT NULL
          AND page_id IS NULL
          AND page_number_snapshot > 0
    ) AS self_dismissed_with_deleted_page
FROM courseware_review_items
WHERE status = 'dismissed';

DO $$
BEGIN
    RAISE NOTICE
        'R-01.1作者自审修改完成后三项人工决策数据库验证通过';
END
$$;
