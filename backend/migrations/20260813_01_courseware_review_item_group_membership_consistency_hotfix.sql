-- ============================================================================
-- TE-DNA 2.0：R-06 正式问题组最终成员一致性延迟守卫 Hotfix V1
-- 文件：20260813_01_courseware_review_item_group_membership_consistency_hotfix.sql
-- ----------------------------------------------------------------------------
-- 缺陷背景：
--   原 enforce_cw_review_item_group_membership_consistency() 是
--   DEFERRABLE INITIALLY DEFERRED constraint trigger，但函数在事务末尾执行时
--   仍直接使用产生该trigger event时保存的 NEW.status / NEW.primary_item_id。
--
--   对于同一事务内存在多个group version的操作，例如：
--     1. 先设置主问题；
--     2. 后续清空主问题；
--     3. 再把原主问题成员移动到其他组；
--   较早排队的deferred trigger会用“历史group行快照”检查“最终成员状态”，
--   从而产生假冲突。
--
-- 修复原则：
--   1. deferred trigger只使用NEW.id作为稳定定位键；
--   2. 执行时重新读取courseware_review_item_groups当前最终行；
--   3. 所有成员、主问题、merged目标检查均针对事务当前最终状态；
--   4. 不改变group/member/event版本协议和追加式事件语义；
--   5. 不修改任何现有业务数据。
-- ============================================================================

BEGIN;

DO $$
BEGIN
    IF to_regclass('public.courseware_review_item_groups') IS NULL
       OR to_regclass('public.courseware_review_item_group_members') IS NULL
       OR to_regclass('public.courseware_review_item_group_events') IS NULL THEN
        RAISE EXCEPTION
            'R-06问题组核心表不完整，禁止执行membership consistency hotfix';
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
    ) THEN
        RAISE EXCEPTION
            'R-06 membership consistency延迟触发器不存在或属性异常';
    END IF;
END
$$;

CREATE OR REPLACE FUNCTION
    public.enforce_cw_review_item_group_membership_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    current_courseware_id UUID;
    current_source_session_id UUID;
    current_created_by UUID;

    current_status VARCHAR(16);
    current_primary_item_id UUID;
    current_merged_into_group_id UUID;

    active_member_count BIGINT;
    primary_member_count BIGINT;
    merge_target_status VARCHAR(16);
BEGIN
    -- ========================================================================
    -- 关键修复：
    -- deferred constraint trigger可能在同一group经历多个version后才执行。
    -- 因此除稳定group ID外，禁止使用历史NEW行作为最终业务事实。
    -- 必须重新读取事务当前可见的最终group行。
    -- ========================================================================
    SELECT
        current_group.courseware_id,
        current_group.source_session_id,
        current_group.created_by,
        current_group.status,
        current_group.primary_item_id,
        current_group.merged_into_group_id
    INTO
        current_courseware_id,
        current_source_session_id,
        current_created_by,
        current_status,
        current_primary_item_id,
        current_merged_into_group_id
    FROM public.courseware_review_item_groups AS current_group
    WHERE current_group.id = NEW.id;

    IF NOT FOUND THEN
        RAISE EXCEPTION
            '问题组延迟成员一致性校验时当前问题组不存在';
    END IF;

    SELECT COUNT(*)
    INTO active_member_count
    FROM public.courseware_review_item_group_members AS member
    WHERE member.group_id = NEW.id
      AND member.courseware_id = current_courseware_id
      AND member.source_session_id = current_source_session_id
      AND member.status = 'active';

    -- merged组的最终状态必须没有active成员，且合并目标仍为同范围active组。
    IF current_status = 'merged' THEN
        IF active_member_count <> 0 THEN
            RAISE EXCEPTION
                '已合并问题组不能继续保留active成员';
        END IF;

        SELECT target_group.status
        INTO merge_target_status
        FROM public.courseware_review_item_groups AS target_group
        WHERE target_group.id = current_merged_into_group_id
          AND target_group.courseware_id = current_courseware_id
          AND target_group.source_session_id = current_source_session_id
          AND target_group.created_by = current_created_by;

        IF NOT FOUND OR merge_target_status <> 'active' THEN
            RAISE EXCEPTION
                '问题组合并目标不存在、越界或已经合并';
        END IF;

        RETURN NEW;
    END IF;

    -- active组如果设置主问题，最终主问题必须恰好命中本组一个active成员。
    IF current_primary_item_id IS NOT NULL THEN
        SELECT COUNT(*)
        INTO primary_member_count
        FROM public.courseware_review_item_group_members AS member
        WHERE member.group_id = NEW.id
          AND member.courseware_id = current_courseware_id
          AND member.source_session_id = current_source_session_id
          AND member.item_id = current_primary_item_id
          AND member.status = 'active';

        IF primary_member_count <> 1 THEN
            RAISE EXCEPTION
                '问题组主问题必须是该组唯一有效成员';
        END IF;
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION
    public.enforce_cw_review_item_group_membership_consistency()
FROM PUBLIC;

DO $$
BEGIN
    RAISE NOTICE
        'R-06 membership consistency deferred final-state hotfix已安装';
END
$$;

COMMIT;
