-- ============================================================================
-- TE-DNA 2.0：R-03 课件审核历史页面快照结构验证
-- 文件：20260809_01_courseware_review_page_snapshots_verify.sql
-- ----------------------------------------------------------------------------
-- 本文件只读：
--   - 不创建结构；
--   - 不修改数据；
--   - 不修复异常；
--   - 任一关键约束或最小权限缺失直接 RAISE EXCEPTION。
--
-- 注意：
--   存量 courseware_reviews 在 R-03 上线前没有完整页面历史 HTML，
--   本验证不会要求旧审核记录必须存在快照，避免使用当前页面伪造历史。
-- ============================================================================

BEGIN;
SET TRANSACTION READ ONLY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_roles
        WHERE rolname = 'tedna_user'
    ) THEN
        RAISE EXCEPTION
            '缺少应用数据库角色 tedna_user';
    END IF;

    IF to_regclass(
        'public.courseware_review_page_snapshots'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少 courseware_review_page_snapshots 表';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint AS constraint_row
        WHERE constraint_row.conrelid =
              'public.courseware_review_page_snapshots'::regclass
          AND constraint_row.contype = 'f'
          AND constraint_row.confrelid =
              'public.courseware_reviews'::regclass
    ) THEN
        RAISE EXCEPTION
            '审核页面快照缺少到 courseware_reviews 的外键';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint AS constraint_row
        WHERE constraint_row.conrelid =
              'public.courseware_review_page_snapshots'::regclass
          AND constraint_row.contype = 'f'
          AND constraint_row.confrelid =
              'public.coursewares'::regclass
    ) THEN
        RAISE EXCEPTION
            '审核页面快照缺少到 coursewares 的外键';
    END IF;

    -- page_id 绝不能外键到当前页面：
    -- 否则删除原页时历史证据也会被删除或阻止页面删除。
    IF EXISTS (
        SELECT 1
        FROM pg_constraint AS constraint_row
        WHERE constraint_row.conrelid =
              'public.courseware_review_page_snapshots'::regclass
          AND constraint_row.contype = 'f'
          AND constraint_row.confrelid =
              'public.courseware_pages'::regclass
    ) THEN
        RAISE EXCEPTION
            '审核页面快照错误地外键到了 courseware_pages';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgrelid =
              'public.courseware_review_page_snapshots'::regclass
          AND tgname =
              'trg_cw_review_page_snapshot_validate_insert'
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION
            '审核页面快照缺少INSERT归属校验守卫';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgrelid =
              'public.courseware_review_page_snapshots'::regclass
          AND tgname =
              'trg_cw_review_page_snapshot_immutable'
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION
            '审核页面快照缺少不可变UPDATE守卫';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_proc AS procedure_row
        JOIN pg_namespace AS namespace_row
          ON namespace_row.oid =
             procedure_row.pronamespace
        WHERE namespace_row.nspname = 'public'
          AND procedure_row.proname =
              'guard_cw_review_page_snapshot_write'
          AND procedure_row.prosecdef
    ) THEN
        RAISE EXCEPTION
            '审核页面快照守卫不是SECURITY DEFINER';
    END IF;

    -- 应用角色只允许读取和创建不可变审核快照。
    IF NOT has_table_privilege(
        'tedna_user',
        'public.courseware_review_page_snapshots',
        'SELECT'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少审核页面快照SELECT权限';
    END IF;

    IF NOT has_table_privilege(
        'tedna_user',
        'public.courseware_review_page_snapshots',
        'INSERT'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少审核页面快照INSERT权限';
    END IF;

    IF has_table_privilege(
        'tedna_user',
        'public.courseware_review_page_snapshots',
        'UPDATE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user不应具有审核页面快照UPDATE权限';
    END IF;

    IF has_table_privilege(
        'tedna_user',
        'public.courseware_review_page_snapshots',
        'DELETE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user不应具有审核页面快照DELETE权限';
    END IF;

    IF has_function_privilege(
        'tedna_user',
        'public.guard_cw_review_page_snapshot_write()',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user不应直接执行审核页面快照数据库守卫函数';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.courseware_review_page_snapshots AS snapshot
        JOIN public.courseware_reviews AS review
          ON review.id = snapshot.courseware_review_id
        WHERE review.courseware_id <> snapshot.courseware_id
    ) THEN
        RAISE EXCEPTION
            '存在审核记录与快照课件归属不一致的数据';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.courseware_review_page_snapshots
        WHERE page_number_snapshot <= 0
    ) THEN
        RAISE EXCEPTION
            '存在非法审核历史页码';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.courseware_review_page_snapshots
        WHERE html_hash <> encode(
            digest(
                convert_to(
                    html_content,
                    'UTF8'
                ),
                'sha256'
            ),
            'hex'
        )
    ) THEN
        RAISE EXCEPTION
            '存在审核历史HTML与SHA-256不一致的数据';
    END IF;
END
$$;

SELECT
    COUNT(*) AS snapshot_page_count,
    COUNT(DISTINCT courseware_review_id)
        AS snapshot_review_count
FROM public.courseware_review_page_snapshots;

SELECT
    COUNT(*) AS total_review_count,
    COUNT(*) FILTER (
        WHERE EXISTS (
            SELECT 1
            FROM public.courseware_review_page_snapshots AS snapshot
            WHERE snapshot.courseware_review_id =
                  review.id
        )
    ) AS reviews_with_page_snapshot,
    COUNT(*) FILTER (
        WHERE NOT EXISTS (
            SELECT 1
            FROM public.courseware_review_page_snapshots AS snapshot
            WHERE snapshot.courseware_review_id =
                  review.id
        )
    ) AS reviews_without_page_snapshot
FROM public.courseware_reviews AS review;

SELECT
    'R-03 courseware_review_page_snapshots verify passed'
        AS verify_result;

COMMIT;
