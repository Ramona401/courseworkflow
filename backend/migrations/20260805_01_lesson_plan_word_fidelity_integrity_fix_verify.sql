\set ON_ERROR_STOP on
\pset pager off

-- ============================================================================
-- 20260805_01_lesson_plan_word_fidelity_integrity_fix_verify.sql
-- ============================================================================

SELECT
    constraint_name,
    delete_rule
FROM information_schema.referential_constraints
WHERE constraint_schema = 'public'
  AND constraint_name IN (
      'lesson_plan_word_import_sessions_lesson_plan_id_fkey',
      'lesson_plan_word_documents_import_session_id_fkey',
      'lesson_plan_word_documents_created_by_fkey',
      'lesson_plan_word_documents_last_changed_by_fkey',
      'lesson_plan_word_document_versions_changed_by_fkey'
  )
ORDER BY constraint_name;

SELECT
    event_object_table,
    trigger_name,
    action_timing,
    event_manipulation
FROM information_schema.triggers
WHERE event_object_schema = 'public'
  AND trigger_name IN (
      'trg_lp_word_import_normalize_unlink',
      'trg_lp_word_import_touch',
      'trg_lp_word_document_validate',
      'trg_lp_word_version_immutable'
  )
ORDER BY
    event_object_table,
    trigger_name,
    event_manipulation;

SELECT
    proname,
    pg_get_functiondef(pg_proc.oid) AS function_definition
FROM pg_proc
JOIN pg_namespace
  ON pg_namespace.oid = pg_proc.pronamespace
WHERE pg_namespace.nspname = 'public'
  AND proname IN (
      'normalize_lesson_plan_word_import_session_unlink',
      'validate_lesson_plan_word_document_write',
      'reject_lesson_plan_word_version_update'
  )
ORDER BY proname;

SELECT
    status,
    COUNT(*) AS row_count
FROM public.lesson_plan_word_import_sessions
GROUP BY status
ORDER BY status;

SELECT
    COUNT(*) AS invalid_confirmed_sessions
FROM public.lesson_plan_word_import_sessions
WHERE status = 'confirmed'
  AND (
      lesson_plan_id IS NULL
      OR confirmed_at IS NULL
  );

SELECT
    COUNT(*) AS invalid_word_documents
FROM public.lesson_plan_word_documents word_document
LEFT JOIN public.lesson_plans lesson_plan
  ON lesson_plan.id = word_document.lesson_plan_id
WHERE lesson_plan.id IS NULL;

SELECT
    COUNT(*) AS orphan_word_versions
FROM public.lesson_plan_word_document_versions version_snapshot
LEFT JOIN public.lesson_plans lesson_plan
  ON lesson_plan.id = version_snapshot.lesson_plan_id
WHERE lesson_plan.id IS NULL;

DO $$
DECLARE
    unlink_trigger_count INTEGER;
    import_plan_delete_action "char";
    normalize_definition TEXT;
    validate_definition TEXT;
    immutable_definition TEXT;
    invalid_confirmed_count BIGINT;
    invalid_document_count BIGINT;
    orphan_version_count BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO unlink_trigger_count
    FROM pg_trigger
    WHERE tgrelid =
        'public.lesson_plan_word_import_sessions'::regclass
      AND tgname =
        'trg_lp_word_import_normalize_unlink'
      AND NOT tgisinternal;

    IF unlink_trigger_count <> 1 THEN
        RAISE EXCEPTION
            '验证失败：导入会话解除绑定触发器应为1个，实际为%',
            unlink_trigger_count;
    END IF;

    SELECT confdeltype
    INTO import_plan_delete_action
    FROM pg_constraint
    WHERE conrelid =
        'public.lesson_plan_word_import_sessions'::regclass
      AND conname =
        'lesson_plan_word_import_sessions_lesson_plan_id_fkey';

    IF import_plan_delete_action IS NULL THEN
        RAISE EXCEPTION
            '验证失败：导入会话lesson_plan_id外键不存在';
    END IF;

    IF import_plan_delete_action <> 'n' THEN
        RAISE EXCEPTION
            '验证失败：导入会话lesson_plan_id仍应使用ON DELETE SET NULL，实际动作代码为%',
            import_plan_delete_action;
    END IF;

    SELECT pg_get_functiondef(
        'public.normalize_lesson_plan_word_import_session_unlink()'::regprocedure
    )
    INTO normalize_definition;

    IF position(
        'NEW.status :='
        IN normalize_definition
    ) = 0
        OR position(
            'NEW.expires_at > NOW()'
            IN normalize_definition
        ) = 0
        OR position(
            'NEW.confirmed_at := NULL'
            IN normalize_definition
        ) = 0 THEN

        RAISE EXCEPTION
            '验证失败：导入会话解除绑定函数内容不完整';
    END IF;

    SELECT pg_get_functiondef(
        'public.validate_lesson_plan_word_document_write()'::regprocedure
    )
    INTO validate_definition;

    IF position(
        'OLD.import_session_id IS NOT NULL'
        IN validate_definition
    ) = 0
        OR position(
            'OLD.created_by IS NOT NULL'
            IN validate_definition
        ) = 0
        OR position(
            'OLD.last_changed_by IS NOT NULL'
            IN validate_definition
        ) = 0 THEN

        RAISE EXCEPTION
            '验证失败：当前Word文档外键置空白名单不完整';
    END IF;

    SELECT pg_get_functiondef(
        'public.reject_lesson_plan_word_version_update()'::regprocedure
    )
    INTO immutable_definition;

    IF position(
        'to_jsonb(NEW) - ''changed_by'''
        IN immutable_definition
    ) = 0
        OR position(
            'OLD.changed_by IS NOT NULL'
            IN immutable_definition
        ) = 0
        OR position(
            'NEW.changed_by IS NULL'
            IN immutable_definition
        ) = 0 THEN

        RAISE EXCEPTION
            '验证失败：不可变版本changed_by置空例外未建立';
    END IF;

    SELECT COUNT(*)
    INTO invalid_confirmed_count
    FROM public.lesson_plan_word_import_sessions
    WHERE status = 'confirmed'
      AND (
          lesson_plan_id IS NULL
          OR confirmed_at IS NULL
      );

    IF invalid_confirmed_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条无效confirmed导入会话',
            invalid_confirmed_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_document_count
    FROM public.lesson_plan_word_documents word_document
    LEFT JOIN public.lesson_plans lesson_plan
      ON lesson_plan.id = word_document.lesson_plan_id
    WHERE lesson_plan.id IS NULL;

    IF invalid_document_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条孤立Word当前文档',
            invalid_document_count;
    END IF;

    SELECT COUNT(*)
    INTO orphan_version_count
    FROM public.lesson_plan_word_document_versions version_snapshot
    LEFT JOIN public.lesson_plans lesson_plan
      ON lesson_plan.id = version_snapshot.lesson_plan_id
    WHERE lesson_plan.id IS NULL;

    IF orphan_version_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条孤立Word历史版本',
            orphan_version_count;
    END IF;

    RAISE NOTICE
        'Word保真父记录删除、用户删除和外键置空完整性修复验证通过';
END
$$;
