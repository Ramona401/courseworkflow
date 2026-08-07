-- ============================================================================
-- TE-DNA 2.0：R-02课件AI审核配置快照完整验证
-- 文件：20260805_05_courseware_ai_review_config_snapshot_verify.sql
-- ----------------------------------------------------------------------------
-- 使用范围：
--   - 临时恢复库；
--   - 完整迁移验证环境。
--
-- 本脚本检查：
--   1. 字段、约束、函数和触发器；
--   2. 存量配置规范化与哈希；
--   3. 非法、重复和未知维度拒绝；
--   4. 应用数据库角色函数权限；
--   5. 真实历史会话配置不可变行为。
--
-- 所有行为测试都位于事务中，结尾统一ROLLBACK，不保留测试写入。
-- ============================================================================

BEGIN;

DO $$
DECLARE
    required_column_count INTEGER;
    required_constraint_count INTEGER;
    invalid_row_count INTEGER;
    zero_hash_count INTEGER;

    test_session_id UUID;
    mutation_blocked BOOLEAN := FALSE;

    default_dimensions JSONB :=
        '[
            "teaching_logic",
            "technical_implementation",
            "interaction_experience",
            "lesson_alignment",
            "authenticity",
            "knowledge_accuracy",
            "page_readability",
            "operational_usability"
        ]'::jsonb;
BEGIN
    -- ========================================================================
    -- 一、基础结构
    -- ========================================================================

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

    IF to_regprocedure(
        'public.normalize_cw_ai_review_dimensions(jsonb)'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少审核维度规范化函数';
    END IF;

    IF to_regprocedure(
        'public.is_valid_cw_ai_review_dimensions(jsonb)'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少审核维度校验函数';
    END IF;

    IF to_regprocedure(
        'public.build_cw_ai_review_config_hash(smallint,jsonb,text,text)'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少审核配置哈希函数';
    END IF;

    IF to_regprocedure(
        'public.guard_cw_ai_review_session_config_snapshot()'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少审核会话配置不可变守卫函数';
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
            'R-02会话配置不可变触发器不存在或未启用';
    END IF;

    -- ========================================================================
    -- 二、生产数据完整性
    -- ========================================================================

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
       )
       OR review_config_hash <>
           public.build_cw_ai_review_config_hash(
               review_config_schema_version,
               review_dimensions_json,
               custom_dimension_description,
               lesson_reference_mode
           );

    IF invalid_row_count <> 0 THEN
        RAISE EXCEPTION
            '存在%条R-02审核配置快照不合法',
            invalid_row_count;
    END IF;

    SELECT COUNT(*)
    INTO zero_hash_count
    FROM courseware_ai_review_sessions
    WHERE review_config_hash =
        '0000000000000000000000000000000000000000000000000000000000000000';

    IF zero_hash_count <> 0 THEN
        RAISE EXCEPTION
            '存在%条会话仍保留迁移临时零哈希',
            zero_hash_count;
    END IF;

    -- ========================================================================
    -- 三、确定性函数行为
    -- ========================================================================

    IF NOT public.is_valid_cw_ai_review_dimensions(
        default_dimensions
    ) THEN
        RAISE EXCEPTION
            '现行兼容默认审核维度未通过数据库校验';
    END IF;

    IF public.is_valid_cw_ai_review_dimensions(
        '[
            "teaching_logic",
            "teaching_logic"
        ]'::jsonb
    ) THEN
        RAISE EXCEPTION
            '重复审核维度未被拒绝';
    END IF;

    IF public.is_valid_cw_ai_review_dimensions(
        '[
            "teaching_logic",
            "unknown_dimension"
        ]'::jsonb
    ) THEN
        RAISE EXCEPTION
            '未知审核维度未被拒绝';
    END IF;

    IF public.is_valid_cw_ai_review_dimensions(
        '[
            "technical_implementation",
            "teaching_logic"
        ]'::jsonb
    ) THEN
        RAISE EXCEPTION
            '非规范顺序审核维度未被拒绝';
    END IF;

    IF length(
        public.build_cw_ai_review_config_hash(
            1::SMALLINT,
            default_dimensions,
            ''::TEXT,
            'current_compatible'::TEXT
        )
    ) <> 64 THEN
        RAISE EXCEPTION
            '审核配置哈希长度不是64';
    END IF;

    IF public.build_cw_ai_review_config_hash(
        1::SMALLINT,
        default_dimensions,
        ''::TEXT,
        'current_compatible'::TEXT
    ) =
       public.build_cw_ai_review_config_hash(
        1::SMALLINT,
        default_dimensions,
        ''::TEXT,
        'no_lesson'::TEXT
    ) THEN
        RAISE EXCEPTION
            '不同教案参考模式生成了相同配置哈希';
    END IF;

    -- ========================================================================
    -- 四、应用角色最小权限
    -- ========================================================================

    IF NOT has_function_privilege(
        'tedna_user',
        'public.normalize_cw_ai_review_dimensions(jsonb)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少维度规范化函数执行权限';
    END IF;

    IF NOT has_function_privilege(
        'tedna_user',
        'public.is_valid_cw_ai_review_dimensions(jsonb)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少维度校验函数执行权限';
    END IF;

    IF NOT has_function_privilege(
        'tedna_user',
        'public.build_cw_ai_review_config_hash(smallint,jsonb,text,text)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少配置哈希函数执行权限';
    END IF;

    -- ========================================================================
    -- 五、真实历史行不可变行为
    -- ========================================================================

    SELECT id
    INTO test_session_id
    FROM courseware_ai_review_sessions
    ORDER BY created_at ASC
    LIMIT 1;

    IF test_session_id IS NULL THEN
        RAISE NOTICE
            '当前数据库没有审核会话，跳过真实行不可变行为测试';
    ELSE
        BEGIN
            UPDATE courseware_ai_review_sessions
            SET lesson_reference_mode =
                CASE
                    WHEN lesson_reference_mode =
                        'current_compatible'
                    THEN 'strict_alignment'
                    ELSE 'current_compatible'
                END
            WHERE id = test_session_id;

        EXCEPTION
            WHEN OTHERS THEN
                mutation_blocked := TRUE;
        END;

        IF NOT mutation_blocked THEN
            RAISE EXCEPTION
                '会话创建后的审核配置仍可被原地修改';
        END IF;
    END IF;

    RAISE NOTICE
        'R-02完整数据库验证通过：结构、数据、哈希、权限和不可变行为均正常';
END
$$;

ROLLBACK;
