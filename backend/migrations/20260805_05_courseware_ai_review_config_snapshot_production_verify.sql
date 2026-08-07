-- ============================================================================
-- TE-DNA 2.0：R-02课件AI审核配置生产库只读验证
-- 文件：20260805_05_courseware_ai_review_config_snapshot_production_verify.sql
-- ----------------------------------------------------------------------------
-- 本脚本只执行元数据和数据查询，不尝试更新任何生产记录。
-- 不创建、修改或删除业务数据。
-- ============================================================================

DO $$
DECLARE
    required_column_count INTEGER;
    required_constraint_count INTEGER;
    invalid_row_count INTEGER;
    total_session_count INTEGER;
    valid_hash_count INTEGER;
    zero_hash_count INTEGER;
BEGIN
    IF to_regclass(
        'public.courseware_ai_review_sessions'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少courseware_ai_review_sessions表';
    END IF;

    SELECT COUNT(*)
    INTO required_column_count
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'courseware_ai_review_sessions'
      AND column_name IN (
          'review_config_schema_version',
          'review_dimensions_json',
          'custom_dimension_description',
          'lesson_reference_mode',
          'review_config_hash'
      );

    IF required_column_count <> 5 THEN
        RAISE EXCEPTION
            'R-02配置字段不完整，实际数量=%',
            required_column_count;
    END IF;

    SELECT COUNT(*)
    INTO required_constraint_count
    FROM pg_constraint
    WHERE conrelid =
        'courseware_ai_review_sessions'::regclass
      AND conname IN (
          'chk_cw_ai_review_config_schema_version',
          'chk_cw_ai_review_dimensions',
          'chk_cw_ai_review_lesson_reference_mode',
          'chk_cw_ai_review_custom_dimension',
          'chk_cw_ai_review_config_hash'
      );

    IF required_constraint_count <> 5 THEN
        RAISE EXCEPTION
            'R-02配置约束不完整，实际数量=%',
            required_constraint_count;
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgrelid =
            'courseware_ai_review_sessions'::regclass
          AND tgname =
            'trg_00_cw_ai_review_session_config_snapshot_guard'
          AND NOT tgisinternal
          AND tgenabled <> 'D'
    ) THEN
        RAISE EXCEPTION
            'R-02配置不可变触发器不存在或未启用';
    END IF;

    SELECT COUNT(*)
    INTO total_session_count
    FROM courseware_ai_review_sessions;

    SELECT COUNT(*)
    INTO valid_hash_count
    FROM courseware_ai_review_sessions
    WHERE review_config_hash =
        public.build_cw_ai_review_config_hash(
            review_config_schema_version,
            review_dimensions_json,
            custom_dimension_description,
            lesson_reference_mode
        );

    SELECT COUNT(*)
    INTO invalid_row_count
    FROM courseware_ai_review_sessions
    WHERE review_config_schema_version <> 1
       OR NOT public.is_valid_cw_ai_review_dimensions(
           review_dimensions_json
       )
       OR lesson_reference_mode NOT IN (
           'current_compatible',
           'strict_alignment',
           'lesson_intent',
           'no_lesson'
       )
       OR (
           review_dimensions_json ? 'custom'
           AND BTRIM(
               custom_dimension_description
           ) = ''
       )
       OR (
           NOT (
               review_dimensions_json ? 'custom'
           )
           AND BTRIM(
               custom_dimension_description
           ) <> ''
       );

    SELECT COUNT(*)
    INTO zero_hash_count
    FROM courseware_ai_review_sessions
    WHERE review_config_hash =
        '0000000000000000000000000000000000000000000000000000000000000000';

    IF invalid_row_count <> 0 THEN
        RAISE EXCEPTION
            '生产库存在%条非法R-02审核配置',
            invalid_row_count;
    END IF;

    IF valid_hash_count <> total_session_count THEN
        RAISE EXCEPTION
            '生产库配置哈希对账失败：总数=%，有效哈希=%',
            total_session_count,
            valid_hash_count;
    END IF;

    IF zero_hash_count <> 0 THEN
        RAISE EXCEPTION
            '生产库存在%条临时零哈希',
            zero_hash_count;
    END IF;

    RAISE NOTICE
        'R-02生产库只读验证通过：会话总数=%，有效哈希=%',
        total_session_count,
        valid_hash_count;
END
$$;

SELECT
    COUNT(*) AS total_sessions,

    COUNT(*) FILTER (
        WHERE lesson_reference_mode =
            'current_compatible'
    ) AS compatible_sessions,

    COUNT(*) FILTER (
        WHERE lesson_reference_mode =
            'strict_alignment'
    ) AS strict_alignment_sessions,

    COUNT(*) FILTER (
        WHERE lesson_reference_mode =
            'lesson_intent'
    ) AS lesson_intent_sessions,

    COUNT(*) FILTER (
        WHERE lesson_reference_mode =
            'no_lesson'
    ) AS no_lesson_sessions,

    COUNT(*) FILTER (
        WHERE review_dimensions_json ? 'custom'
    ) AS custom_sessions,

    COUNT(*) FILTER (
        WHERE review_config_hash =
            public.build_cw_ai_review_config_hash(
                review_config_schema_version,
                review_dimensions_json,
                custom_dimension_description,
                lesson_reference_mode
            )
    ) AS valid_hash_sessions
FROM courseware_ai_review_sessions;
