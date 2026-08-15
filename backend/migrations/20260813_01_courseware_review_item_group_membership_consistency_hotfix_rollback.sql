-- ============================================================================
-- TE-DNA 2.0：R-06 membership consistency Hotfix 回滚
-- 文件：20260813_01_courseware_review_item_group_membership_consistency_hotfix_rollback.sql
-- ----------------------------------------------------------------------------
-- 警告：
--   本文件只回退20260813_01 hotfix函数实现，不删除R-06/R-07表或业务数据。
--   回退后会恢复历史NEW快照语义，因此merge/split多版本事务可能再次产生假冲突。
--   执行前必须完整备份数据库。
-- ============================================================================

BEGIN;

CREATE OR REPLACE FUNCTION
    public.enforce_cw_review_item_group_membership_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    active_member_count BIGINT;
    primary_member_count BIGINT;
    merge_target_status VARCHAR(16);
BEGIN
    SELECT COUNT(*)
    INTO active_member_count
    FROM public.courseware_review_item_group_members AS member
    WHERE member.group_id = NEW.id
      AND member.courseware_id = NEW.courseware_id
      AND member.source_session_id = NEW.source_session_id
      AND member.status = 'active';

    IF NEW.status = 'merged' THEN
        IF active_member_count <> 0 THEN
            RAISE EXCEPTION
                '已合并问题组不能继续保留active成员';
        END IF;

        SELECT target_group.status
        INTO merge_target_status
        FROM public.courseware_review_item_groups AS target_group
        WHERE target_group.id = NEW.merged_into_group_id
          AND target_group.courseware_id = NEW.courseware_id
          AND target_group.source_session_id = NEW.source_session_id
          AND target_group.created_by = NEW.created_by;

        IF NOT FOUND OR merge_target_status <> 'active' THEN
            RAISE EXCEPTION
                '问题组合并目标不存在、越界或已经合并';
        END IF;

        RETURN NEW;
    END IF;

    IF NEW.primary_item_id IS NOT NULL THEN
        SELECT COUNT(*)
        INTO primary_member_count
        FROM public.courseware_review_item_group_members AS member
        WHERE member.group_id = NEW.id
          AND member.courseware_id = NEW.courseware_id
          AND member.source_session_id = NEW.source_session_id
          AND member.item_id = NEW.primary_item_id
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
        'R-06 membership consistency hotfix已回滚到原V1实现';
END
$$;

COMMIT;
