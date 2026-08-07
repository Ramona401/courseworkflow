-- ============================================================================
-- TE-DNA 2.0：课件AI审核配置不可变快照 V2.1
-- 文件：20260805_05_courseware_ai_review_config_snapshot.sql
-- ----------------------------------------------------------------------------
-- 对应PRD：R-02 审核维度与教案参考模式。
--
-- 本迁移负责：
--   1. 为每次课件AI审核会话保存不可变审核配置快照；
--   2. 定义审核维度和教案参考模式的数据库白名单；
--   3. 为存量会话确定性回填“现行兼容”配置；
--   4. 为旧后端创建会话提供安全兼容默认值；
--   5. 计算并校验规范化配置SHA-256；
--   6. 阻止会话创建后原地修改配置。
--
-- 本迁移不修改既有批次结果、最终报告、整改项或正式审核反馈。
-- 旧历史finding中的dimension保持原样，避免改写既有审核事实。
-- ============================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- 一、迁移前检查
-- ============================================================================

DO $$
BEGIN
    IF to_regclass(
        'public.courseware_ai_review_sessions'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少courseware_ai_review_sessions表';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_roles
        WHERE rolname = 'tedna_user'
    ) THEN
        RAISE EXCEPTION
            '缺少应用数据库角色tedna_user';
    END IF;
END
$$;

-- ============================================================================
-- 二、审核维度规范化与配置哈希函数
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.normalize_cw_ai_review_dimensions(
        input_dimensions JSONB
    )
RETURNS JSONB
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT CASE
        WHEN COALESCE(
            jsonb_typeof(input_dimensions),
            ''
        ) <> 'array' THEN
            '[]'::jsonb
        ELSE
            COALESCE(
                (
                    SELECT jsonb_agg(
                        to_jsonb(allowed.code)
                        ORDER BY allowed.position
                    )
                    FROM (
                        VALUES
                            (1, 'teaching_logic'::text),
                            (2, 'technical_implementation'::text),
                            (3, 'interaction_experience'::text),
                            (4, 'lesson_alignment'::text),
                            (5, 'authenticity'::text),
                            (6, 'knowledge_accuracy'::text),
                            (7, 'page_readability'::text),
                            (8, 'operational_usability'::text),
                            (9, 'custom'::text)
                    ) AS allowed(position, code)
                    WHERE input_dimensions ? allowed.code
                ),
                '[]'::jsonb
            )
    END
$$;

CREATE OR REPLACE FUNCTION
    public.is_valid_cw_ai_review_dimensions(
        input_dimensions JSONB
    )
RETURNS BOOLEAN
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT CASE
        WHEN COALESCE(
            jsonb_typeof(input_dimensions),
            ''
        ) <> 'array' THEN
            FALSE
        ELSE
            jsonb_array_length(input_dimensions) > 0
            AND input_dimensions =
                public.normalize_cw_ai_review_dimensions(
                    input_dimensions
                )
    END
$$;

CREATE OR REPLACE FUNCTION
    public.build_cw_ai_review_config_hash(
        schema_version SMALLINT,
        dimensions JSONB,
        custom_description TEXT,
        reference_mode TEXT
    )
RETURNS TEXT
LANGUAGE SQL
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT encode(
        digest(
            convert_to(
                jsonb_build_object(
                    'custom_dimension_description',
                    COALESCE(custom_description, ''),
                    'lesson_reference_mode',
                    COALESCE(reference_mode, ''),
                    'review_dimensions',
                    COALESCE(
                        dimensions,
                        '[]'::jsonb
                    ),
                    'schema_version',
                    schema_version
                )::text,
                'UTF8'
            ),
            'sha256'
        ),
        'hex'
    )
$$;

REVOKE ALL
ON FUNCTION
    public.normalize_cw_ai_review_dimensions(JSONB)
FROM PUBLIC;

REVOKE ALL
ON FUNCTION
    public.is_valid_cw_ai_review_dimensions(JSONB)
FROM PUBLIC;

REVOKE ALL
ON FUNCTION
    public.build_cw_ai_review_config_hash(
        SMALLINT,
        JSONB,
        TEXT,
        TEXT
    )
FROM PUBLIC;

GRANT EXECUTE
ON FUNCTION
    public.normalize_cw_ai_review_dimensions(JSONB)
TO tedna_user;

GRANT EXECUTE
ON FUNCTION
    public.is_valid_cw_ai_review_dimensions(JSONB)
TO tedna_user;

GRANT EXECUTE
ON FUNCTION
    public.build_cw_ai_review_config_hash(
        SMALLINT,
        JSONB,
        TEXT,
        TEXT
    )
TO tedna_user;

-- ============================================================================
-- 三、会话配置快照字段
-- ============================================================================

ALTER TABLE courseware_ai_review_sessions
    ADD COLUMN IF NOT EXISTS
        review_config_schema_version SMALLINT
        NOT NULL
        DEFAULT 1,

    ADD COLUMN IF NOT EXISTS
        review_dimensions_json JSONB
        NOT NULL
        DEFAULT '[
            "teaching_logic",
            "technical_implementation",
            "interaction_experience",
            "lesson_alignment",
            "authenticity",
            "knowledge_accuracy",
            "page_readability",
            "operational_usability"
        ]'::jsonb,

    ADD COLUMN IF NOT EXISTS
        custom_dimension_description TEXT
        NOT NULL
        DEFAULT '',

    ADD COLUMN IF NOT EXISTS
        lesson_reference_mode VARCHAR(32)
        NOT NULL
        DEFAULT 'current_compatible',

    ADD COLUMN IF NOT EXISTS
        review_config_hash VARCHAR(64)
        NOT NULL
        DEFAULT
            '0000000000000000000000000000000000000000000000000000000000000000';

ALTER TABLE courseware_ai_review_sessions
    ALTER COLUMN review_config_schema_version
        SET DEFAULT 1,
    ALTER COLUMN review_dimensions_json
        SET DEFAULT '[
            "teaching_logic",
            "technical_implementation",
            "interaction_experience",
            "lesson_alignment",
            "authenticity",
            "knowledge_accuracy",
            "page_readability",
            "operational_usability"
        ]'::jsonb,
    ALTER COLUMN custom_dimension_description
        SET DEFAULT '',
    ALTER COLUMN lesson_reference_mode
        SET DEFAULT 'current_compatible',
    ALTER COLUMN review_config_hash
        SET DEFAULT
            '0000000000000000000000000000000000000000000000000000000000000000';

-- ============================================================================
-- 四、存量会话确定性回填
-- ============================================================================

UPDATE courseware_ai_review_sessions
SET
    review_config_schema_version = 1,

    review_dimensions_json =
        CASE
            WHEN public.is_valid_cw_ai_review_dimensions(
                review_dimensions_json
            ) THEN
                public.normalize_cw_ai_review_dimensions(
                    review_dimensions_json
                )
            ELSE
                '[
                    "teaching_logic",
                    "technical_implementation",
                    "interaction_experience",
                    "lesson_alignment",
                    "authenticity",
                    "knowledge_accuracy",
                    "page_readability",
                    "operational_usability"
                ]'::jsonb
        END,

    lesson_reference_mode =
        CASE
            WHEN lesson_reference_mode IN (
                'current_compatible',
                'strict_alignment',
                'lesson_intent',
                'no_lesson'
            ) THEN
                lesson_reference_mode
            ELSE
                'current_compatible'
        END;

UPDATE courseware_ai_review_sessions
SET custom_dimension_description =
    CASE
        WHEN review_dimensions_json ? 'custom' THEN
            BTRIM(
                COALESCE(
                    custom_dimension_description,
                    ''
                )
            )
        ELSE
            ''
    END;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_ai_review_sessions
        WHERE review_dimensions_json ? 'custom'
          AND BTRIM(
              custom_dimension_description
          ) = ''
    ) THEN
        RAISE EXCEPTION
            '存在选择custom维度但说明为空的审核会话，禁止继续迁移';
    END IF;
END
$$;

UPDATE courseware_ai_review_sessions
SET review_config_hash =
    public.build_cw_ai_review_config_hash(
        review_config_schema_version,
        review_dimensions_json,
        custom_dimension_description,
        lesson_reference_mode
    );

-- ============================================================================
-- 五、数据库约束
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
            'courseware_ai_review_sessions'::regclass
          AND conname =
            'chk_cw_ai_review_config_schema_version'
    ) THEN
        ALTER TABLE courseware_ai_review_sessions
            ADD CONSTRAINT
                chk_cw_ai_review_config_schema_version
            CHECK (
                review_config_schema_version = 1
            );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
            'courseware_ai_review_sessions'::regclass
          AND conname =
            'chk_cw_ai_review_dimensions'
    ) THEN
        ALTER TABLE courseware_ai_review_sessions
            ADD CONSTRAINT
                chk_cw_ai_review_dimensions
            CHECK (
                public.is_valid_cw_ai_review_dimensions(
                    review_dimensions_json
                )
            );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
            'courseware_ai_review_sessions'::regclass
          AND conname =
            'chk_cw_ai_review_lesson_reference_mode'
    ) THEN
        ALTER TABLE courseware_ai_review_sessions
            ADD CONSTRAINT
                chk_cw_ai_review_lesson_reference_mode
            CHECK (
                lesson_reference_mode IN (
                    'current_compatible',
                    'strict_alignment',
                    'lesson_intent',
                    'no_lesson'
                )
            );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
            'courseware_ai_review_sessions'::regclass
          AND conname =
            'chk_cw_ai_review_custom_dimension'
    ) THEN
        ALTER TABLE courseware_ai_review_sessions
            ADD CONSTRAINT
                chk_cw_ai_review_custom_dimension
            CHECK (
                (
                    review_dimensions_json ? 'custom'
                    AND BTRIM(
                        custom_dimension_description
                    ) <> ''
                )
                OR
                (
                    NOT (
                        review_dimensions_json ? 'custom'
                    )
                    AND BTRIM(
                        custom_dimension_description
                    ) = ''
                )
            );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
            'courseware_ai_review_sessions'::regclass
          AND conname =
            'chk_cw_ai_review_config_hash'
    ) THEN
        ALTER TABLE courseware_ai_review_sessions
            ADD CONSTRAINT
                chk_cw_ai_review_config_hash
            CHECK (
                review_config_hash ~ '^[0-9a-f]{64}$'
                AND review_config_hash =
                    public.build_cw_ai_review_config_hash(
                        review_config_schema_version,
                        review_dimensions_json,
                        custom_dimension_description,
                        lesson_reference_mode
                    )
            );
    END IF;
END
$$;

-- ============================================================================
-- 六、不可变配置守卫
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.guard_cw_ai_review_session_config_snapshot()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.review_config_schema_version <> 1 THEN
            RAISE EXCEPTION
                '课件AI审核配置协议版本无效';
        END IF;

        IF NOT public.is_valid_cw_ai_review_dimensions(
            NEW.review_dimensions_json
        ) THEN
            RAISE EXCEPTION
                '课件AI审核维度为空、重复、顺序异常或包含非法值';
        END IF;

        IF NEW.lesson_reference_mode NOT IN (
            'current_compatible',
            'strict_alignment',
            'lesson_intent',
            'no_lesson'
        ) THEN
            RAISE EXCEPTION
                '课件AI审核教案参考模式无效';
        END IF;

        IF NEW.review_dimensions_json ? 'custom' THEN
            IF BTRIM(
                NEW.custom_dimension_description
            ) = '' THEN
                RAISE EXCEPTION
                    '选择自定义审核维度时必须填写说明';
            END IF;
        ELSIF BTRIM(
            NEW.custom_dimension_description
        ) <> '' THEN
            RAISE EXCEPTION
                '未选择自定义审核维度时不能保存自定义说明';
        END IF;

        NEW.review_config_hash :=
            public.build_cw_ai_review_config_hash(
                NEW.review_config_schema_version,
                NEW.review_dimensions_json,
                NEW.custom_dimension_description,
                NEW.lesson_reference_mode
            );

        RETURN NEW;
    END IF;

    IF NEW.review_config_schema_version IS DISTINCT FROM
            OLD.review_config_schema_version
       OR NEW.review_dimensions_json IS DISTINCT FROM
            OLD.review_dimensions_json
       OR NEW.custom_dimension_description IS DISTINCT FROM
            OLD.custom_dimension_description
       OR NEW.lesson_reference_mode IS DISTINCT FROM
            OLD.lesson_reference_mode
       OR NEW.review_config_hash IS DISTINCT FROM
            OLD.review_config_hash THEN
        RAISE EXCEPTION
            '课件AI审核配置在会话创建后不可原地修改，必须创建新会话';
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION
    public.guard_cw_ai_review_session_config_snapshot()
FROM PUBLIC;

DROP TRIGGER IF EXISTS
    trg_00_cw_ai_review_session_config_snapshot_guard
ON courseware_ai_review_sessions;

CREATE TRIGGER
    trg_00_cw_ai_review_session_config_snapshot_guard
BEFORE INSERT OR UPDATE
ON courseware_ai_review_sessions
FOR EACH ROW
EXECUTE FUNCTION
    public.guard_cw_ai_review_session_config_snapshot();

-- ============================================================================
-- 七、字段注释
-- ============================================================================

COMMENT ON COLUMN
    courseware_ai_review_sessions.review_config_schema_version IS
    '课件AI审核配置协议版本，R-02初版固定为1';

COMMENT ON COLUMN
    courseware_ai_review_sessions.review_dimensions_json IS
    '会话创建时固化的审核维度代码数组，按平台固定顺序保存且不可修改';

COMMENT ON COLUMN
    courseware_ai_review_sessions.custom_dimension_description IS
    '仅选择custom审核维度时保存的非空说明';

COMMENT ON COLUMN
    courseware_ai_review_sessions.lesson_reference_mode IS
    '教案参考模式：现行兼容、严格一致、参考教案意图或不使用教案';

COMMENT ON COLUMN
    courseware_ai_review_sessions.review_config_hash IS
    '规范化审核配置JSON的UTF-8字节SHA-256，不接受浏览器可信输入';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        'R-02课件AI审核配置不可变快照迁移完成';
END
$$;
