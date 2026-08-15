-- ============================================================================
-- TE-DNA 2.0：作者自审修改完成后三项人工决策数据库守卫
-- 文件：20260809_01_courseware_review_self_post_apply.sql
-- ----------------------------------------------------------------------------
-- R-01.1 自审在页面修改完成后必须由教师人工决定：
--
--   1. 确认已经解决：applied -> resolved，由既有专用事务处理；
--   2. 继续调整：applied -> applying，以applied_page_hash为新执行起点；
--   3. 暂时不处理：applied -> dismissed，保留最近一次applied事实。
--
-- 本迁移只补充数据库层对“暂时不处理 / 恢复”的支持。
--
-- 设计边界：
--
--   - dismissed只有作者自审项才允许保留applied版本与完成时间；
--   - dismissed -> applied只能恢复原有完成事实，不能创建或替换版本；
--   - 恢复时当前页面必须仍等于applied_page_hash；
--   - page_id因ON DELETE SET NULL变空时，page_number_snapshot > 0
--     明确表示原页级页面已经删除，禁止恢复；
--   - 正式审核项不能利用本路径；
--   - 原有大型指令版本守卫函数保持不变；
--   - 原守卫触发器只对这一条经过专用守卫的恢复路径让行；
--   - 其他全部UPDATE继续由原守卫处理。
--
-- 本文件可重复执行。
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

    IF to_regprocedure(
        'public.guard_cw_review_item_instruction_bindings()'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少既有课件整改指令绑定守卫函数';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgrelid =
              'public.courseware_review_items'::regclass
          AND tgname =
              'trg_cw_review_item_instruction_binding_guard'
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION
            '缺少既有课件整改指令绑定守卫触发器';
    END IF;
END
$$;

-- 在收紧正式约束前先拒绝任何无法解释的历史组合。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'dismissed'
          AND applied_instruction_version_id IS NOT NULL
          AND source_type <> 'self'
    ) THEN
        RAISE EXCEPTION
            '存在正式整改项以dismissed保留applied版本，拒绝迁移';
    END IF;
END
$$;

-- --------------------------------------------------------------------------
-- 一、扩展应用版本事实约束
-- --------------------------------------------------------------------------
--
-- 普通dismissed仍保持：
--
--   applied_instruction_version_id IS NULL
--   applied_at IS NULL
--
-- 只有self dismissed可以保留上一轮页面修改完成事实。
ALTER TABLE courseware_review_items
    DROP CONSTRAINT IF EXISTS
        chk_cw_review_item_applied_instruction_version;

ALTER TABLE courseware_review_items
    ADD CONSTRAINT
        chk_cw_review_item_applied_instruction_version
    CHECK (
        (
            applied_instruction_version_id IS NULL
            AND applied_at IS NULL
            AND status <> 'applying'
        )
        OR
        (
            applied_instruction_version_id IS NOT NULL
            AND status = 'applying'
            AND applied_at IS NULL
        )
        OR
        (
            applied_instruction_version_id IS NOT NULL
            AND status IN (
                'applied',
                'resolved',
                'stale',
                'orphaned',
                'dismissed'
            )
            AND applied_at IS NOT NULL
            AND (
                status <> 'dismissed'
                OR source_type = 'self'
            )
        )
    );

-- --------------------------------------------------------------------------
-- 二、专门保护 self dismissed -> applied 恢复
-- --------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION
    public.guard_cw_review_item_self_applied_restore()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    referenced_version_status VARCHAR(24);
    current_page_hash TEXT;
BEGIN
    -- 本函数只应该被专用WHEN触发器调用。
    IF OLD.status <> 'dismissed'
       OR NEW.status <> 'applied' THEN
        RETURN NEW;
    END IF;

    -- 这个专用路径只允许改变status与updated_at。
    --
    -- 原大型绑定守卫会对此路径让行，因此必须在这里完整阻止
    -- 浏览器或数据库调用方借恢复动作同时覆盖其他业务事实。
    IF (
        to_jsonb(NEW) - ARRAY['status', 'updated_at']
    ) IS DISTINCT FROM (
        to_jsonb(OLD) - ARRAY['status', 'updated_at']
    ) THEN
        RAISE EXCEPTION
            '恢复作者自审问题时只能改变处理状态';
    END IF;

    -- 只有尚未进入正式反馈历史的作者自审问题可以恢复。
    IF OLD.source_type <> 'self'
       OR NEW.source_type <> 'self'
       OR OLD.courseware_review_id IS NOT NULL
       OR NEW.courseware_review_id IS NOT NULL
       OR OLD.feedback_id IS NOT NULL
       OR NEW.feedback_id IS NOT NULL THEN
        RAISE EXCEPTION
            '只有未进入正式反馈历史的作者自审问题可以恢复修改完成状态';
    END IF;

    -- dismissed必须确实来自一条完整的applied事实。
    IF OLD.applied_instruction_version_id IS NULL
       OR OLD.current_instruction_version_id IS NULL
       OR OLD.applied_instruction_version_id <>
            OLD.current_instruction_version_id
       OR OLD.applied_at IS NULL
       OR BTRIM(
            COALESCE(
                OLD.applied_page_hash,
                ''
            )
          ) = '' THEN
        RAISE EXCEPTION
            '暂时不处理的问题缺少可恢复的修改完成事实';
    END IF;

    SELECT version.status
    INTO referenced_version_status
    FROM courseware_review_instruction_versions AS version
    WHERE version.id =
            OLD.applied_instruction_version_id
      AND version.item_id =
            OLD.id;

    IF NOT FOUND
       OR referenced_version_status <> 'confirmed' THEN
        RAISE EXCEPTION
            '恢复作者自审问题时关联的修改方案版本不存在或已经失效';
    END IF;

    -- page_id使用ON DELETE SET NULL。
    --
    -- 因此page_id为空不能直接解释成“整课问题”：
    -- page_number_snapshot > 0表示这条记录最初明确属于某一页，
    -- 此时空page_id只能视为原页面已经删除。
    IF OLD.page_id IS NULL
       AND OLD.page_number_snapshot > 0 THEN
        RAISE EXCEPTION
            '恢复作者自审问题失败：原页面已不存在';
    END IF;

    -- 真正的页级问题必须再次以数据库当前HTML验证修改完成快照。
    IF OLD.page_id IS NOT NULL THEN
        SELECT encode(
            digest(
                convert_to(
                    COALESCE(
                        page.html_content,
                        ''
                    ),
                    'UTF8'
                ),
                'sha256'
            ),
            'hex'
        )
        INTO current_page_hash
        FROM courseware_pages AS page
        WHERE page.id =
                OLD.page_id
          AND page.courseware_id =
                OLD.courseware_id;

        IF NOT FOUND THEN
            RAISE EXCEPTION
                '恢复作者自审问题失败：原页面已不存在';
        END IF;

        IF current_page_hash <>
                BTRIM(
                    COALESCE(
                        OLD.applied_page_hash,
                        ''
                    )
                ) THEN
            RAISE EXCEPTION
                '恢复作者自审问题失败：页面内容已变化';
        END IF;
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION
    public.guard_cw_review_item_self_applied_restore()
FROM PUBLIC;

-- --------------------------------------------------------------------------
-- 三、让既有大型绑定守卫只对这一个安全恢复路径让行
-- --------------------------------------------------------------------------
--
-- 其他所有UPDATE仍调用原函数。
DROP TRIGGER IF EXISTS
    trg_cw_review_item_instruction_binding_guard
ON courseware_review_items;

CREATE TRIGGER
    trg_cw_review_item_instruction_binding_guard
BEFORE UPDATE
ON courseware_review_items
FOR EACH ROW
WHEN (
    NOT (
        OLD.status = 'dismissed'
        AND NEW.status = 'applied'
    )
)
EXECUTE FUNCTION
    public.guard_cw_review_item_instruction_bindings();

-- 专用守卫只命中dismissed -> applied。
DROP TRIGGER IF EXISTS
    trg_01_cw_review_item_self_applied_restore_guard
ON courseware_review_items;

CREATE TRIGGER
    trg_01_cw_review_item_self_applied_restore_guard
BEFORE UPDATE
ON courseware_review_items
FOR EACH ROW
WHEN (
    OLD.status = 'dismissed'
    AND NEW.status = 'applied'
)
EXECUTE FUNCTION
    public.guard_cw_review_item_self_applied_restore();

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        'R-01.1作者自审修改完成后三项人工决策数据库守卫已安装';
END
$$;
