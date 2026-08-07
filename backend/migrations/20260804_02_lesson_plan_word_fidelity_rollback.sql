\set ON_ERROR_STOP on

-- ============================================================================
-- 20260804_02_lesson_plan_word_fidelity_rollback.sql
-- 回滚原格式Word教案数据层
--
-- 本回滚会删除：
--   - Word导入短时会话；
--   - 正式教案当前Word文档；
--   - Word不可变版本历史；
--   - 本迁移创建的触发器和函数。
--
-- 本回滚不会修改：
--   - lesson_plans现有正文和业务状态；
--   - lesson_plan_content_versions正文历史；
--   - 教案审核、批注、课件和AI索引数据。
--
-- 数据库回滚不会自动删除私有目录中的DOCX文件。
-- 正式执行前必须再次备份数据库，并由后续运维脚本按数据库存储键安全清理孤儿文件。
-- ============================================================================

BEGIN;

DROP TRIGGER IF EXISTS trg_lesson_plans_word_document_stale
ON public.lesson_plans;

DROP FUNCTION IF EXISTS
    public.mark_lesson_plan_word_document_stale();

DROP TRIGGER IF EXISTS trg_lp_word_version_immutable
ON public.lesson_plan_word_document_versions;

DROP FUNCTION IF EXISTS
    public.reject_lesson_plan_word_version_update();

DROP TRIGGER IF EXISTS trg_lp_word_document_snapshot
ON public.lesson_plan_word_documents;

DROP FUNCTION IF EXISTS
    public.snapshot_lesson_plan_word_document_version();

DROP TRIGGER IF EXISTS trg_lp_word_document_validate
ON public.lesson_plan_word_documents;

DROP FUNCTION IF EXISTS
    public.validate_lesson_plan_word_document_write();

DROP TRIGGER IF EXISTS trg_lp_word_import_touch
ON public.lesson_plan_word_import_sessions;

DROP FUNCTION IF EXISTS
    public.touch_lesson_plan_word_import_session();

DROP TABLE IF EXISTS
    public.lesson_plan_word_document_versions;

DROP TABLE IF EXISTS
    public.lesson_plan_word_documents;

DROP TABLE IF EXISTS
    public.lesson_plan_word_import_sessions;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '原格式Word教案导入会话、当前文档、版本历史和触发器已回滚';
END
$$;
