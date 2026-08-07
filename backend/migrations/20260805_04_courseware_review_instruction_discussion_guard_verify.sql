-- ============================================================================
-- TE-DNA 2.0：重新讨论保留当前确认版本验证
-- 文件：20260805_04_courseware_review_instruction_discussion_guard_verify.sql
-- ----------------------------------------------------------------------------
-- 本文件只读验证数据库结构和现有数据，不修改业务数据。
-- ============================================================================

DO $$
DECLARE
    trigger_enabled CHAR;
BEGIN
    SELECT trigger.tgenabled
    INTO trigger_enabled
    FROM pg_trigger AS trigger
    WHERE trigger.tgrelid =
            'public.courseware_review_items'::regclass
      AND trigger.tgname =
            'trg_00_cw_review_item_discussion_version_guard'
      AND NOT trigger.tgisinternal;

    IF trigger_enabled IS NULL THEN
        RAISE EXCEPTION
            '缺少重新讨论版本保持触发器';
    END IF;

    IF trigger_enabled NOT IN (
        'O',
        'A'
    ) THEN
        RAISE EXCEPTION
            '重新讨论版本保持触发器未启用，状态=%',
            trigger_enabled;
    END IF;
END
$$;

DO $$
BEGIN
    IF to_regprocedure(
        'public.guard_cw_review_item_discussion_preserves_instruction()'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少重新讨论版本保持触发函数';
    END IF;
END
$$;

-- 兼容快照和当前版本必须成对存在。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        WHERE (
                BTRIM(
                    COALESCE(
                        item.confirmed_instruction,
                        ''
                    )
                ) = ''
                AND item.current_instruction_version_id IS NOT NULL
              )
           OR (
                BTRIM(
                    COALESCE(
                        item.confirmed_instruction,
                        ''
                    )
                ) <> ''
                AND item.current_instruction_version_id IS NULL
              )
    ) THEN
        RAISE EXCEPTION
            '存在确认正文和当前版本引用不成对的整改项';
    END IF;
END
$$;

-- 当前版本正文必须等于兼容快照。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        JOIN courseware_review_instruction_versions AS version
          ON version.id =
                item.current_instruction_version_id
         AND version.item_id =
                item.id
        WHERE BTRIM(version.content) <>
                BTRIM(item.confirmed_instruction)
    ) THEN
        RAISE EXCEPTION
            '存在当前版本正文与兼容快照不一致的整改项';
    END IF;
END
$$;

-- 讨论中的当前版本仍应保持有效确认状态。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        JOIN courseware_review_instruction_versions AS version
          ON version.id =
                item.current_instruction_version_id
         AND version.item_id =
                item.id
        WHERE item.status = 'discussing'
          AND version.status <> 'confirmed'
    ) THEN
        RAISE EXCEPTION
            '存在讨论中但当前版本已失效的整改项';
    END IF;
END
$$;

-- 已交付或已应用记录不得处于讨论状态。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        WHERE item.status = 'discussing'
          AND (
                item.courseware_review_id IS NOT NULL
                OR item.feedback_id IS NOT NULL
                OR item.delivered_instruction_version_id IS NOT NULL
                OR item.applied_instruction_version_id IS NOT NULL
                OR item.applied_at IS NOT NULL
          )
    ) THEN
        RAISE EXCEPTION
            '存在已交付或已应用但仍处于讨论状态的整改项';
    END IF;
END
$$;

SELECT
    COUNT(*) FILTER (
        WHERE item.status = 'discussing'
    ) AS discussing_item_count,
    COUNT(*) FILTER (
        WHERE item.status = 'discussing'
          AND item.current_instruction_version_id IS NOT NULL
    ) AS discussing_with_confirmed_version_count,
    COUNT(*) FILTER (
        WHERE item.delivered_instruction_version_id IS NOT NULL
    ) AS delivered_version_count,
    COUNT(*) FILTER (
        WHERE item.applied_instruction_version_id IS NOT NULL
    ) AS applied_version_count
FROM courseware_review_items AS item;

DO $$
BEGIN
    RAISE NOTICE
        'R-01重新讨论保留当前确认版本验证通过';
END
$$;
