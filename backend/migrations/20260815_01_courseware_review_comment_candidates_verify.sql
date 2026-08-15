-- 20260815_01_courseware_review_comment_candidates_verify.sql
--
-- R-08审核意见候选数据库结构定向验证。
--
-- 本文件只读数据库元数据和候选表数量，不写业务数据。

\set ON_ERROR_STOP on

DO $$
BEGIN
        IF to_regclass(
                'public.courseware_review_comment_candidates'
        ) IS NULL THEN
                RAISE EXCEPTION
                        'R-08验证失败：courseware_review_comment_candidates不存在';
        END IF;

        IF NOT EXISTS (
                SELECT 1
                FROM pg_catalog.pg_trigger
                WHERE tgrelid =
                        'public.courseware_review_comment_candidates'::regclass
                  AND tgname =
                        'trg_cw_review_comment_candidate_immutable'
                  AND NOT tgisinternal
        ) THEN
                RAISE EXCEPTION
                        'R-08验证失败：不可变候选trigger不存在';
        END IF;

        IF to_regprocedure(
                'public.guard_courseware_review_comment_candidate_immutable()'
        ) IS NULL THEN
                RAISE EXCEPTION
                        'R-08验证失败：不可变候选guard函数不存在';
        END IF;
END;
$$;

SELECT
        column_name,
        data_type,
        is_nullable,
        column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'courseware_review_comment_candidates'
ORDER BY ordinal_position;

SELECT
        conname,
        contype,
        pg_get_constraintdef(oid) AS definition
FROM pg_catalog.pg_constraint
WHERE conrelid =
        'public.courseware_review_comment_candidates'::regclass
ORDER BY conname;

SELECT
        indexname,
        indexdef
FROM pg_catalog.pg_indexes
WHERE schemaname = 'public'
  AND tablename = 'courseware_review_comment_candidates'
ORDER BY indexname;

SELECT
        tgname,
        pg_get_triggerdef(oid) AS definition
FROM pg_catalog.pg_trigger
WHERE tgrelid =
        'public.courseware_review_comment_candidates'::regclass
  AND NOT tgisinternal
ORDER BY tgname;

SELECT
        COUNT(*) AS existing_candidate_rows
FROM courseware_review_comment_candidates;
