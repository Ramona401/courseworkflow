\set ON_ERROR_STOP on
\pset pager off

-- ============================================================================
-- 20260804_02_lesson_plan_word_fidelity_verify.sql
-- 原格式Word教案数据层结构和数据完整性验证
-- ============================================================================

SELECT
    table_name,
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name IN (
      'lesson_plan_word_import_sessions',
      'lesson_plan_word_documents',
      'lesson_plan_word_document_versions'
  )
ORDER BY table_name, ordinal_position;

SELECT
    conrelid::regclass::text AS table_name,
    conname AS constraint_name,
    pg_get_constraintdef(oid) AS constraint_definition
FROM pg_constraint
WHERE conrelid IN (
    'public.lesson_plan_word_import_sessions'::regclass,
    'public.lesson_plan_word_documents'::regclass,
    'public.lesson_plan_word_document_versions'::regclass
)
ORDER BY table_name, constraint_name;

SELECT
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename IN (
      'lesson_plan_word_import_sessions',
      'lesson_plan_word_documents',
      'lesson_plan_word_document_versions'
  )
ORDER BY tablename, indexname;

SELECT
    event_object_table,
    trigger_name,
    action_timing,
    event_manipulation
FROM information_schema.triggers
WHERE event_object_schema = 'public'
  AND trigger_name IN (
      'trg_lp_word_import_touch',
      'trg_lp_word_document_validate',
      'trg_lp_word_document_snapshot',
      'trg_lp_word_version_immutable',
      'trg_lesson_plans_word_document_stale'
  )
ORDER BY event_object_table, trigger_name, event_manipulation;

SELECT
    status,
    COUNT(*) AS row_count
FROM public.lesson_plan_word_import_sessions
GROUP BY status
ORDER BY status;

SELECT
    status,
    COUNT(*) AS row_count
FROM public.lesson_plan_word_documents
GROUP BY status
ORDER BY status;

SELECT
    COUNT(*) AS invalid_confirmed_import_sessions
FROM public.lesson_plan_word_import_sessions import_session
LEFT JOIN public.lesson_plans lesson_plan
  ON lesson_plan.id = import_session.lesson_plan_id
WHERE import_session.status = 'confirmed'
  AND (
      import_session.lesson_plan_id IS NULL
      OR import_session.confirmed_at IS NULL
      OR lesson_plan.id IS NULL
  );

SELECT
    COUNT(*) AS active_word_documents_without_matching_version
FROM public.lesson_plan_word_documents word_document
LEFT JOIN public.lesson_plan_word_document_versions version_snapshot
  ON version_snapshot.lesson_plan_id = word_document.lesson_plan_id
 AND version_snapshot.version = word_document.version
WHERE word_document.status = 'active'
  AND version_snapshot.id IS NULL;

SELECT
    COUNT(*) AS orphan_word_document_versions
FROM public.lesson_plan_word_document_versions version_snapshot
LEFT JOIN public.lesson_plans lesson_plan
  ON lesson_plan.id = version_snapshot.lesson_plan_id
WHERE lesson_plan.id IS NULL;

SELECT
    COUNT(*) AS word_document_domain_mismatches
FROM public.lesson_plan_word_documents word_document
JOIN public.lesson_plans lesson_plan
  ON lesson_plan.id = word_document.lesson_plan_id
WHERE word_document.education_domain
      IS DISTINCT FROM lower(btrim(COALESCE(lesson_plan.education_domain, '')));

SELECT
    COUNT(*) AS invalid_active_word_documents
FROM public.lesson_plan_word_documents word_document
JOIN public.lesson_plans lesson_plan
  ON lesson_plan.id = word_document.lesson_plan_id
WHERE word_document.status = 'active'
  AND (
      lesson_plan.deleted_at IS NOT NULL
      OR btrim(word_document.current_storage_key) = ''
      OR word_document.current_file_sha256 !~ '^[0-9a-f]{64}$'
      OR word_document.structure_json = '{}'::jsonb
      OR btrim(word_document.semantic_markdown) = ''
      OR word_document.semantic_markdown_hash !~ '^[0-9a-f]{64}$'
      OR word_document.structure_hash !~ '^[0-9a-f]{64}$'
      OR word_document.semantic_markdown
         IS DISTINCT FROM COALESCE(lesson_plan.content_markdown, '')
  );

DO $$
DECLARE
    required_trigger_count INTEGER;
    invalid_confirmed_count BIGINT;
    missing_version_count BIGINT;
    orphan_version_count BIGINT;
    domain_mismatch_count BIGINT;
    invalid_active_count BIGINT;
BEGIN
    IF to_regclass(
        'public.lesson_plan_word_import_sessions'
    ) IS NULL THEN
        RAISE EXCEPTION
            '验证失败：Word导入会话表不存在';
    END IF;

    IF to_regclass(
        'public.lesson_plan_word_documents'
    ) IS NULL THEN
        RAISE EXCEPTION
            '验证失败：当前Word文档表不存在';
    END IF;

    IF to_regclass(
        'public.lesson_plan_word_document_versions'
    ) IS NULL THEN
        RAISE EXCEPTION
            '验证失败：Word文档版本表不存在';
    END IF;

    SELECT COUNT(DISTINCT trigger_name)
    INTO required_trigger_count
    FROM information_schema.triggers
    WHERE event_object_schema = 'public'
      AND trigger_name IN (
          'trg_lp_word_import_touch',
          'trg_lp_word_document_validate',
          'trg_lp_word_document_snapshot',
          'trg_lp_word_version_immutable',
          'trg_lesson_plans_word_document_stale'
      );

    IF required_trigger_count <> 5 THEN
        RAISE EXCEPTION
            '验证失败：Word保真必需触发器应为5个，实际为%',
            required_trigger_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_confirmed_count
    FROM public.lesson_plan_word_import_sessions import_session
    LEFT JOIN public.lesson_plans lesson_plan
      ON lesson_plan.id = import_session.lesson_plan_id
    WHERE import_session.status = 'confirmed'
      AND (
          import_session.lesson_plan_id IS NULL
          OR import_session.confirmed_at IS NULL
          OR lesson_plan.id IS NULL
      );

    IF invalid_confirmed_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条无效confirmed Word导入会话',
            invalid_confirmed_count;
    END IF;

    SELECT COUNT(*)
    INTO missing_version_count
    FROM public.lesson_plan_word_documents word_document
    LEFT JOIN public.lesson_plan_word_document_versions version_snapshot
      ON version_snapshot.lesson_plan_id = word_document.lesson_plan_id
     AND version_snapshot.version = word_document.version
    WHERE word_document.status = 'active'
      AND version_snapshot.id IS NULL;

    IF missing_version_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条active Word文档没有对应不可变版本',
            missing_version_count;
    END IF;

    SELECT COUNT(*)
    INTO orphan_version_count
    FROM public.lesson_plan_word_document_versions version_snapshot
    LEFT JOIN public.lesson_plans lesson_plan
      ON lesson_plan.id = version_snapshot.lesson_plan_id
    WHERE lesson_plan.id IS NULL;

    IF orphan_version_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条孤立Word文档版本',
            orphan_version_count;
    END IF;

    SELECT COUNT(*)
    INTO domain_mismatch_count
    FROM public.lesson_plan_word_documents word_document
    JOIN public.lesson_plans lesson_plan
      ON lesson_plan.id = word_document.lesson_plan_id
    WHERE word_document.education_domain
          IS DISTINCT FROM lower(btrim(COALESCE(lesson_plan.education_domain, '')));

    IF domain_mismatch_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条Word文档教育域与教案快照不一致',
            domain_mismatch_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_active_count
    FROM public.lesson_plan_word_documents word_document
    JOIN public.lesson_plans lesson_plan
      ON lesson_plan.id = word_document.lesson_plan_id
    WHERE word_document.status = 'active'
      AND (
          lesson_plan.deleted_at IS NOT NULL
          OR btrim(word_document.current_storage_key) = ''
          OR word_document.current_file_sha256 !~ '^[0-9a-f]{64}$'
          OR word_document.structure_json = '{}'::jsonb
          OR btrim(word_document.semantic_markdown) = ''
          OR word_document.semantic_markdown_hash !~ '^[0-9a-f]{64}$'
          OR word_document.structure_hash !~ '^[0-9a-f]{64}$'
          OR word_document.semantic_markdown
             IS DISTINCT FROM COALESCE(lesson_plan.content_markdown, '')
      );

    IF invalid_active_count <> 0 THEN
        RAISE EXCEPTION
            '验证失败：存在%条无效active Word保真文档',
            invalid_active_count;
    END IF;

    RAISE NOTICE
        '原格式Word教案导入会话、当前文档、不可变版本和语义同步约束验证通过';
END
$$;
