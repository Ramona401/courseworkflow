-- ============================================================================
-- TE-DNA 2.0：重新讨论保留当前确认版本守卫
-- 文件：20260805_04_courseware_review_instruction_discussion_guard.sql
-- ----------------------------------------------------------------------------
-- 对应PRD：R-01 指令版本与重新确认。
--
-- 修复目标：
--   1. 重新进入讨论只改变整改项状态，不清空当前确认版本；
--   2. 已确认版本在新版明确确认成功前继续作为当前确认事实；
--   3. 当前草稿由讨论消息、AI候选和浏览器编辑态承载；
--   4. 正式交付或已经应用的整改项不得重新进入讨论；
--   5. 阻止旧后端通过“清空confirmed_instruction”破坏版本指针。
--
-- 本迁移新增一个更早执行的BEFORE UPDATE触发器。
-- 触发器名称以trg_00开头，确保先于既有
-- trg_cw_review_item_instruction_binding_guard执行。
--
-- 本文件不修改历史版本数据，也不改变已交付和已应用版本引用。
-- ============================================================================

BEGIN;

DO $$
BEGIN
    IF to_regclass(
        'public.courseware_review_items'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少courseware_review_items表';
    END IF;

    IF to_regclass(
        'public.courseware_review_instruction_versions'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少courseware_review_instruction_versions表';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'courseware_review_items'
          AND column_name = 'current_instruction_version_id'
    ) THEN
        RAISE EXCEPTION
            '缺少current_instruction_version_id字段';
    END IF;
END
$$;

CREATE OR REPLACE FUNCTION
    public.guard_cw_review_item_discussion_preserves_instruction()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    -- 只有既有可讨论状态进入或保持discussing时需要执行本守卫。
    IF NEW.status = 'discussing'
       AND OLD.status IN (
           'detected',
           'discussing',
           'confirmed'
       ) THEN
        -- 正式交付和页面应用形成后，确认版本已经冻结。
        -- 此时不允许借讨论入口改变整改项的执行阶段。
        IF OLD.courseware_review_id IS NOT NULL
           OR OLD.feedback_id IS NOT NULL
           OR OLD.delivered_instruction_version_id IS NOT NULL
           OR OLD.applied_instruction_version_id IS NOT NULL
           OR OLD.applied_at IS NOT NULL THEN
            RAISE EXCEPTION
                '已正式交付或已应用的整改项不能重新进入讨论';
        END IF;

        -- 重新讨论只能改变状态和更新时间。
        -- 当前确认版本、兼容正文和确认时间必须保持不变，
        -- 直到“保存为新版并确认”事务原子切换到新版本。
        IF NEW.current_instruction_version_id IS DISTINCT FROM
                OLD.current_instruction_version_id
           OR NEW.confirmed_instruction IS DISTINCT FROM
                OLD.confirmed_instruction
           OR NEW.confirmed_at IS DISTINCT FROM
                OLD.confirmed_at THEN
            RAISE EXCEPTION
                '重新讨论必须保留当前确认版本、确认正文和确认时间';
        END IF;
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION
    public.guard_cw_review_item_discussion_preserves_instruction()
FROM PUBLIC;

DROP TRIGGER IF EXISTS
    trg_00_cw_review_item_discussion_version_guard
ON courseware_review_items;

CREATE TRIGGER
    trg_00_cw_review_item_discussion_version_guard
BEFORE UPDATE
ON courseware_review_items
FOR EACH ROW
EXECUTE FUNCTION
    public.guard_cw_review_item_discussion_preserves_instruction();

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        'R-01重新讨论保留当前确认版本守卫迁移完成';
END
$$;
