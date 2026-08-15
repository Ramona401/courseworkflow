-- ============================================================================
-- TE-DNA 2.0：课件AI审核Hash函数灾备恢复可移植性只读验证
-- 文件：20260814_01_cw_review_hash_function_search_path_verify.sql
-- ----------------------------------------------------------------------------
-- 本文件只读，不修改结构或业务数据。
-- ============================================================================

DO $$
DECLARE
    invalid_function_count INTEGER;
    invalid_digest_count INTEGER;
    config_hash TEXT;
    message_hash TEXT;
    operations_hash TEXT;
BEGIN
    SELECT COUNT(*)
    INTO invalid_digest_count
    FROM pg_proc AS procedure
    INNER JOIN pg_namespace AS namespace
        ON namespace.oid = procedure.pronamespace
    WHERE namespace.nspname = 'public'
      AND procedure.proname = 'digest'
      AND pg_get_function_identity_arguments(procedure.oid)
          IN (
              'bytea, text',
              'text, text'
          );

    IF invalid_digest_count <> 2 THEN
        RAISE EXCEPTION
            'public.digest函数签名不完整，实际数量=%',
            invalid_digest_count;
    END IF;

    SELECT COUNT(*)
    INTO invalid_function_count
    FROM pg_proc AS procedure
    INNER JOIN pg_namespace AS namespace
        ON namespace.oid = procedure.pronamespace
    WHERE namespace.nspname = 'public'
      AND procedure.proname IN (
          'build_cw_ai_review_config_hash',
          'build_cw_review_impact_message_hash',
          'build_cw_review_impact_operations_hash'
      )
      AND COALESCE(
              procedure.proconfig,
              ARRAY[]::TEXT[]
          ) @> ARRAY[
              'search_path=public, pg_temp'
          ]::TEXT[];

    IF invalid_function_count <> 3 THEN
        RAISE EXCEPTION
            'Hash函数search_path固定数量错误，实际=%，期望=3',
            invalid_function_count;
    END IF;

    SELECT public.build_cw_ai_review_config_hash(
        1::SMALLINT,
        '["teaching_logic"]'::JSONB,
        '',
        'no_lesson'
    )
    INTO config_hash;

    SELECT public.build_cw_review_impact_message_hash(
        'restore-portability-verify',
        '{}'::JSONB
    )
    INTO message_hash;

    SELECT public.build_cw_review_impact_operations_hash(
        '[]'::JSONB
    )
    INTO operations_hash;

    IF config_hash !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION
            '配置Hash函数结果异常：%',
            config_hash;
    END IF;

    IF message_hash !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION
            '消息Hash函数结果异常：%',
            message_hash;
    END IF;

    IF operations_hash !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION
            '操作Hash函数结果异常：%',
            operations_hash;
    END IF;
END
$$;

SELECT
    namespace.nspname AS function_schema,
    procedure.proname,
    pg_get_function_identity_arguments(
        procedure.oid
    ) AS arguments,
    procedure.proconfig
FROM pg_proc AS procedure
INNER JOIN pg_namespace AS namespace
    ON namespace.oid = procedure.pronamespace
WHERE namespace.nspname = 'public'
  AND procedure.proname IN (
      'build_cw_ai_review_config_hash',
      'build_cw_review_impact_message_hash',
      'build_cw_review_impact_operations_hash'
  )
ORDER BY procedure.proname;

SELECT
    public.build_cw_ai_review_config_hash(
        1::SMALLINT,
        '["teaching_logic"]'::JSONB,
        '',
        'no_lesson'
    ) AS config_hash,
    public.build_cw_review_impact_message_hash(
        'restore-portability-verify',
        '{}'::JSONB
    ) AS message_hash,
    public.build_cw_review_impact_operations_hash(
        '[]'::JSONB
    ) AS operations_hash;
