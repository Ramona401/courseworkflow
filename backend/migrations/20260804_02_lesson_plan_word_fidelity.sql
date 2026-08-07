\set ON_ERROR_STOP on

-- ============================================================================
-- 20260804_02_lesson_plan_word_fidelity.sql
-- 原格式Word教案导入会话、当前文档与不可变版本历史
--
-- 产品目标：
--   1. 在原有“导入已有教案”内增加“保留原Word格式”模式；
--   2. 原始DOCX、表格结构、单元格、图片引用、上下标和公式对象不再扁平化丢失；
--   3. 同时生成content_markdown，继续复用现有AI评审、审核、索引和课件生成链；
--   4. 老师修改Word内容块时只替换目标内容块，不重建学校原有模板；
--   5. 每次有效修改保存不可变完整版本，可按原格式导出和安全恢复；
--   6. 所有DOCX文件存放在服务端私有目录，数据库只保存受控相对存储键和哈希；
--   7. 本迁移不改变现有教案状态机、审核记录、正文版本表或课件生成行为。
--
-- 数据模型：
--
--   lesson_plan_word_import_sessions
--       DOCX上传后的短时导入会话。解析完成并由老师确认后绑定正式教案。
--
--   lesson_plan_word_documents
--       每份正式教案当前唯一的Word保真文档快照。
--
--   lesson_plan_word_document_versions
--       Word文档每次内容变化产生的不可变完整版本。
--
-- 安全边界：
--   - 不在数据库中保存DOCX二进制；
--   - storage_key只能是私有存储根目录下的受控相对键；
--   - 文件哈希、结构哈希和语义正文哈希统一使用小写SHA-256；
--   - 具体教学域只能是k12、vocational或adult；
--   - 当前Word正文与lesson_plans.content_markdown发生漂移时自动标记stale；
--   - 版本表禁止UPDATE，父教案物理删除时仍允许级联清理历史。
-- ============================================================================

BEGIN;

-- ============================================================================
-- 一、短时Word导入会话
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.lesson_plan_word_import_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    created_by UUID NOT NULL
        REFERENCES public.users(id)
        ON DELETE CASCADE,

    education_domain VARCHAR(20) NOT NULL,

    -- uploaded：原文件已安全落盘，尚未完成结构解析；
    -- parsed：结构、语义正文和质量告警均已生成，可供老师预览确认；
    -- confirmed：已绑定正式教案并创建当前Word文档；
    -- failed：解析失败，可展示明确原因；
    -- expired：短时会话已过期，不得继续确认。
    status VARCHAR(20) NOT NULL DEFAULT 'uploaded',

    original_file_name VARCHAR(255) NOT NULL,

    -- 相对于私有Word存储根目录的受控相对键，不得是绝对路径。
    storage_key VARCHAR(1024) NOT NULL,

    file_size BIGINT NOT NULL,

    mime_type VARCHAR(100) NOT NULL DEFAULT
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',

    file_sha256 VARCHAR(64) NOT NULL,

    parser_version VARCHAR(50) NOT NULL DEFAULT '',

    structure_schema_version INTEGER NOT NULL DEFAULT 1,

    -- 统一结构对象，第一版至少包含：
    -- document、tables、blocks、media、formulas、relationships和source_order。
    structure_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 供AI评审、索引和课件生成使用的语义正文。
    semantic_markdown TEXT NOT NULL DEFAULT '',

    semantic_markdown_hash VARCHAR(64) NOT NULL DEFAULT '',

    -- 表格数、内容块数、图片数、公式数和可编辑块数等确定性指标。
    metrics_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 解析告警数组，例如未支持的OLE对象、WMF预览、复杂浮动对象或公式降级。
    warnings_json JSONB NOT NULL DEFAULT '[]'::jsonb,

    error_message TEXT NOT NULL DEFAULT '',

    lesson_plan_id UUID
        REFERENCES public.lesson_plans(id)
        ON DELETE SET NULL,

    expires_at TIMESTAMPTZ NOT NULL DEFAULT
        (NOW() + INTERVAL '24 hours'),

    parsed_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lp_word_import_domain_check
        CHECK (
            education_domain IN (
                'k12',
                'vocational',
                'adult'
            )
        ),

    CONSTRAINT lp_word_import_status_check
        CHECK (
            status IN (
                'uploaded',
                'parsed',
                'confirmed',
                'failed',
                'expired'
            )
        ),

    CONSTRAINT lp_word_import_file_name_check
        CHECK (
            btrim(original_file_name) <> ''
            AND char_length(original_file_name) <= 255
        ),

    CONSTRAINT lp_word_import_storage_key_check
        CHECK (
            btrim(storage_key) <> ''
            AND storage_key NOT LIKE '/%'
            AND position('..' IN storage_key) = 0
            AND position(chr(92) IN storage_key) = 0
            AND position('//' IN storage_key) = 0
        ),

    CONSTRAINT lp_word_import_file_size_check
        CHECK (
            file_size > 0
            AND file_size <= 31457280
        ),

    CONSTRAINT lp_word_import_mime_check
        CHECK (
            mime_type =
                'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
        ),

    CONSTRAINT lp_word_import_hash_check
        CHECK (
            file_sha256 ~ '^[0-9a-f]{64}$'
            AND (
                semantic_markdown_hash = ''
                OR semantic_markdown_hash ~ '^[0-9a-f]{64}$'
            )
        ),

    CONSTRAINT lp_word_import_schema_check
        CHECK (
            structure_schema_version >= 1
        ),

    CONSTRAINT lp_word_import_json_check
        CHECK (
            jsonb_typeof(structure_json) = 'object'
            AND jsonb_typeof(metrics_json) = 'object'
            AND jsonb_typeof(warnings_json) = 'array'
        ),

    CONSTRAINT lp_word_import_error_length_check
        CHECK (
            char_length(error_message) <= 4000
        ),

    CONSTRAINT lp_word_import_expiry_check
        CHECK (
            expires_at > created_at
        ),

    CONSTRAINT lp_word_import_parsed_content_check
        CHECK (
            status NOT IN ('parsed', 'confirmed')
            OR (
                btrim(parser_version) <> ''
                AND structure_json <> '{}'::jsonb
                AND btrim(semantic_markdown) <> ''
                AND semantic_markdown_hash <> ''
                AND parsed_at IS NOT NULL
            )
        ),

    CONSTRAINT lp_word_import_confirmed_check
        CHECK (
            status <> 'confirmed'
            OR (
                lesson_plan_id IS NOT NULL
                AND confirmed_at IS NOT NULL
            )
        ),

    CONSTRAINT lp_word_import_failed_check
        CHECK (
            status <> 'failed'
            OR btrim(error_message) <> ''
        )
);

CREATE INDEX IF NOT EXISTS idx_lp_word_import_owner_status
ON public.lesson_plan_word_import_sessions (
    created_by,
    status,
    created_at DESC
);

CREATE INDEX IF NOT EXISTS idx_lp_word_import_expiry
ON public.lesson_plan_word_import_sessions (
    status,
    expires_at
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_lp_word_import_lesson_plan
ON public.lesson_plan_word_import_sessions (
    lesson_plan_id
)
WHERE lesson_plan_id IS NOT NULL;

COMMENT ON TABLE public.lesson_plan_word_import_sessions
IS
    '保留原Word格式导入的短时会话；解析完成后由老师确认并绑定正式教案';

COMMENT ON COLUMN public.lesson_plan_word_import_sessions.storage_key
IS
    '私有Word存储根目录下的受控相对键，禁止返回浏览器或拼成Nginx公开URL';

COMMENT ON COLUMN public.lesson_plan_word_import_sessions.structure_json
IS
    'DOCX OOXML解析后的统一结构对象，包含稳定block_id并保留表格、媒体和公式关系';

COMMENT ON COLUMN public.lesson_plan_word_import_sessions.semantic_markdown
IS
    '从Word结构确定性生成的AI语义正文，后续同步写入lesson_plans.content_markdown';

-- 导入会话任何更新都刷新updated_at。
CREATE OR REPLACE FUNCTION public.touch_lesson_plan_word_import_session()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_lp_word_import_touch
ON public.lesson_plan_word_import_sessions;

CREATE TRIGGER trg_lp_word_import_touch
BEFORE UPDATE
ON public.lesson_plan_word_import_sessions
FOR EACH ROW
EXECUTE FUNCTION public.touch_lesson_plan_word_import_session();

-- ============================================================================
-- 二、正式教案当前唯一Word文档
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.lesson_plan_word_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    lesson_plan_id UUID NOT NULL,

    import_session_id UUID
        REFERENCES public.lesson_plan_word_import_sessions(id)
        ON DELETE SET NULL,

    created_by UUID
        REFERENCES public.users(id)
        ON DELETE SET NULL,

    education_domain VARCHAR(20) NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'active',

    version INTEGER NOT NULL DEFAULT 1,

    source_format VARCHAR(10) NOT NULL DEFAULT 'docx',

    original_file_name VARCHAR(255) NOT NULL,

    -- 原始母版文件永远不被覆盖。
    original_storage_key VARCHAR(1024) NOT NULL,

    original_file_sha256 VARCHAR(64) NOT NULL,

    -- 当前版本指向一个不可变版本文件，不使用固定current.docx覆盖历史。
    current_storage_key VARCHAR(1024) NOT NULL,

    current_file_sha256 VARCHAR(64) NOT NULL,

    parser_version VARCHAR(50) NOT NULL,

    structure_schema_version INTEGER NOT NULL DEFAULT 1,

    structure_json JSONB NOT NULL,

    semantic_markdown TEXT NOT NULL,

    semantic_markdown_hash VARCHAR(64) NOT NULL,

    structure_hash VARCHAR(64) NOT NULL,

    metrics_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    warnings_json JSONB NOT NULL DEFAULT '[]'::jsonb,

    last_change_source VARCHAR(30) NOT NULL DEFAULT 'import',

    last_changed_by UUID
        REFERENCES public.users(id)
        ON DELETE SET NULL,

    last_change_summary VARCHAR(500) NOT NULL DEFAULT '',

    error_message TEXT NOT NULL DEFAULT '',

    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lp_word_documents_plan_unique
        UNIQUE (lesson_plan_id),

    CONSTRAINT lp_word_documents_plan_fkey
        FOREIGN KEY (lesson_plan_id)
        REFERENCES public.lesson_plans(id)
        ON DELETE CASCADE,

    CONSTRAINT lp_word_documents_domain_check
        CHECK (
            education_domain IN (
                'k12',
                'vocational',
                'adult'
            )
        ),

    CONSTRAINT lp_word_documents_status_check
        CHECK (
            status IN (
                'active',
                'stale',
                'failed'
            )
        ),

    CONSTRAINT lp_word_documents_version_check
        CHECK (
            version >= 1
            AND structure_schema_version >= 1
        ),

    CONSTRAINT lp_word_documents_format_check
        CHECK (
            source_format = 'docx'
        ),

    CONSTRAINT lp_word_documents_file_name_check
        CHECK (
            btrim(original_file_name) <> ''
            AND char_length(original_file_name) <= 255
        ),

    CONSTRAINT lp_word_documents_storage_key_check
        CHECK (
            btrim(original_storage_key) <> ''
            AND btrim(current_storage_key) <> ''
            AND original_storage_key NOT LIKE '/%'
            AND current_storage_key NOT LIKE '/%'
            AND position('..' IN original_storage_key) = 0
            AND position('..' IN current_storage_key) = 0
            AND position(chr(92) IN original_storage_key) = 0
            AND position(chr(92) IN current_storage_key) = 0
            AND position('//' IN original_storage_key) = 0
            AND position('//' IN current_storage_key) = 0
        ),

    CONSTRAINT lp_word_documents_hash_check
        CHECK (
            original_file_sha256 ~ '^[0-9a-f]{64}$'
            AND current_file_sha256 ~ '^[0-9a-f]{64}$'
            AND semantic_markdown_hash ~ '^[0-9a-f]{64}$'
            AND structure_hash ~ '^[0-9a-f]{64}$'
        ),

    CONSTRAINT lp_word_documents_json_check
        CHECK (
            jsonb_typeof(structure_json) = 'object'
            AND jsonb_typeof(metrics_json) = 'object'
            AND jsonb_typeof(warnings_json) = 'array'
        ),

    CONSTRAINT lp_word_documents_source_check
        CHECK (
            last_change_source IN (
                'import',
                'manual',
                'ai',
                'restore',
                'system'
            )
        ),

    CONSTRAINT lp_word_documents_summary_check
        CHECK (
            char_length(last_change_summary) <= 500
            AND char_length(error_message) <= 4000
        ),

    CONSTRAINT lp_word_documents_active_check
        CHECK (
            status <> 'active'
            OR (
                btrim(parser_version) <> ''
                AND structure_json <> '{}'::jsonb
                AND btrim(semantic_markdown) <> ''
                AND semantic_markdown_hash <> ''
                AND structure_hash <> ''
                AND btrim(current_storage_key) <> ''
                AND current_file_sha256 <> ''
                AND generated_at IS NOT NULL
            )
        )
);

CREATE INDEX IF NOT EXISTS idx_lp_word_documents_status
ON public.lesson_plan_word_documents (
    status,
    updated_at DESC
);

CREATE INDEX IF NOT EXISTS idx_lp_word_documents_domain
ON public.lesson_plan_word_documents (
    education_domain,
    status
);

COMMENT ON TABLE public.lesson_plan_word_documents
IS
    '每份教案当前唯一的原格式Word文档快照；语义正文与结构化Word共同维护';

COMMENT ON COLUMN public.lesson_plan_word_documents.structure_json
IS
    '当前Word统一结构对象，包含稳定block_id、表格合并关系、媒体关系和公式保护节点';

COMMENT ON COLUMN public.lesson_plan_word_documents.semantic_markdown
IS
    '与当前Word结构同步的AI语义正文，必须与lesson_plans.content_markdown保持一致';

-- ============================================================================
-- 三、Word文档不可变版本
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.lesson_plan_word_document_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    lesson_plan_id UUID NOT NULL,

    version INTEGER NOT NULL,

    storage_key VARCHAR(1024) NOT NULL,

    file_sha256 VARCHAR(64) NOT NULL,

    parser_version VARCHAR(50) NOT NULL,

    structure_schema_version INTEGER NOT NULL DEFAULT 1,

    structure_json JSONB NOT NULL,

    semantic_markdown TEXT NOT NULL,

    semantic_markdown_hash VARCHAR(64) NOT NULL,

    structure_hash VARCHAR(64) NOT NULL,

    metrics_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    warnings_json JSONB NOT NULL DEFAULT '[]'::jsonb,

    change_source VARCHAR(30) NOT NULL,

    changed_by UUID
        REFERENCES public.users(id)
        ON DELETE SET NULL,

    change_summary VARCHAR(500) NOT NULL DEFAULT '',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lp_word_doc_versions_unique
        UNIQUE (
            lesson_plan_id,
            version
        ),

    CONSTRAINT lp_word_doc_versions_plan_fkey
        FOREIGN KEY (lesson_plan_id)
        REFERENCES public.lesson_plans(id)
        ON DELETE CASCADE,

    CONSTRAINT lp_word_doc_versions_version_check
        CHECK (
            version >= 1
            AND structure_schema_version >= 1
        ),

    CONSTRAINT lp_word_doc_versions_storage_key_check
        CHECK (
            btrim(storage_key) <> ''
            AND storage_key NOT LIKE '/%'
            AND position('..' IN storage_key) = 0
            AND position(chr(92) IN storage_key) = 0
            AND position('//' IN storage_key) = 0
        ),

    CONSTRAINT lp_word_doc_versions_hash_check
        CHECK (
            file_sha256 ~ '^[0-9a-f]{64}$'
            AND semantic_markdown_hash ~ '^[0-9a-f]{64}$'
            AND structure_hash ~ '^[0-9a-f]{64}$'
        ),

    CONSTRAINT lp_word_doc_versions_json_check
        CHECK (
            jsonb_typeof(structure_json) = 'object'
            AND jsonb_typeof(metrics_json) = 'object'
            AND jsonb_typeof(warnings_json) = 'array'
        ),

    CONSTRAINT lp_word_doc_versions_source_check
        CHECK (
            change_source IN (
                'import',
                'manual',
                'ai',
                'restore',
                'system'
            )
        ),

    CONSTRAINT lp_word_doc_versions_summary_check
        CHECK (
            char_length(change_summary) <= 500
        )
);

CREATE INDEX IF NOT EXISTS idx_lp_word_doc_versions_plan
ON public.lesson_plan_word_document_versions (
    lesson_plan_id,
    version DESC
);

CREATE INDEX IF NOT EXISTS idx_lp_word_doc_versions_created
ON public.lesson_plan_word_document_versions (
    lesson_plan_id,
    created_at DESC
);

COMMENT ON TABLE public.lesson_plan_word_document_versions
IS
    '原格式Word教案的不可变完整版本；文件、结构和语义正文保持同一版本边界';

-- ============================================================================
-- 四、当前Word文档写入边界与版本单调递增
-- ============================================================================

CREATE OR REPLACE FUNCTION public.validate_lesson_plan_word_document_write()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    plan_domain VARCHAR(20);
    plan_deleted_at TIMESTAMPTZ;
    content_changed BOOLEAN;
BEGIN
    SELECT
        lower(btrim(COALESCE(education_domain, ''))),
        deleted_at
    INTO
        plan_domain,
        plan_deleted_at
    FROM public.lesson_plans
    WHERE id = NEW.lesson_plan_id;

    IF NOT FOUND OR plan_deleted_at IS NOT NULL THEN
        RAISE EXCEPTION
            'Word保真文档必须绑定存在且未删除的教案'
            USING ERRCODE = '23503';
    END IF;

    IF plan_domain <> NEW.education_domain THEN
        RAISE EXCEPTION
            'Word保真文档教育域%与教案教育域%不一致',
            NEW.education_domain,
            plan_domain
            USING ERRCODE = '23514';
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF NEW.version <> 1 THEN
            RAISE EXCEPTION
                'Word保真文档首个版本必须为1，当前为%',
                NEW.version
                USING ERRCODE = '23514';
        END IF;

        NEW.created_at := COALESCE(NEW.created_at, NOW());
        NEW.updated_at := NOW();
        RETURN NEW;
    END IF;

    IF NEW.lesson_plan_id IS DISTINCT FROM OLD.lesson_plan_id
        OR NEW.import_session_id IS DISTINCT FROM OLD.import_session_id
        OR NEW.education_domain IS DISTINCT FROM OLD.education_domain
        OR NEW.source_format IS DISTINCT FROM OLD.source_format
        OR NEW.original_file_name IS DISTINCT FROM OLD.original_file_name
        OR NEW.original_storage_key IS DISTINCT FROM OLD.original_storage_key
        OR NEW.original_file_sha256 IS DISTINCT FROM OLD.original_file_sha256 THEN
        RAISE EXCEPTION
            'Word保真文档身份、原始母版和教育域字段不可修改'
            USING ERRCODE = '23514';
    END IF;

    content_changed :=
        NEW.current_storage_key IS DISTINCT FROM OLD.current_storage_key
        OR NEW.current_file_sha256 IS DISTINCT FROM OLD.current_file_sha256
        OR NEW.parser_version IS DISTINCT FROM OLD.parser_version
        OR NEW.structure_schema_version IS DISTINCT FROM OLD.structure_schema_version
        OR NEW.structure_json IS DISTINCT FROM OLD.structure_json
        OR NEW.semantic_markdown IS DISTINCT FROM OLD.semantic_markdown
        OR NEW.semantic_markdown_hash IS DISTINCT FROM OLD.semantic_markdown_hash
        OR NEW.structure_hash IS DISTINCT FROM OLD.structure_hash
        OR NEW.metrics_json IS DISTINCT FROM OLD.metrics_json
        OR NEW.warnings_json IS DISTINCT FROM OLD.warnings_json
        OR NEW.last_change_source IS DISTINCT FROM OLD.last_change_source
        OR NEW.last_changed_by IS DISTINCT FROM OLD.last_changed_by
        OR NEW.last_change_summary IS DISTINCT FROM OLD.last_change_summary
        OR NEW.generated_at IS DISTINCT FROM OLD.generated_at;

    IF content_changed AND NEW.version <> OLD.version + 1 THEN
        RAISE EXCEPTION
            'Word内容变化时版本必须从%递增到%，当前提交为%',
            OLD.version,
            OLD.version + 1,
            NEW.version
            USING
                ERRCODE = '23514',
                HINT = '请在同一事务中读取当前版本并以current_version+1写入完整新快照';
    END IF;

    IF NOT content_changed AND NEW.version <> OLD.version THEN
        RAISE EXCEPTION
            '仅状态或错误变化时不得修改Word文档版本，当前版本%被改为%',
            OLD.version,
            NEW.version
            USING ERRCODE = '23514';
    END IF;

    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_lp_word_document_validate
ON public.lesson_plan_word_documents;

CREATE TRIGGER trg_lp_word_document_validate
BEFORE INSERT OR UPDATE
ON public.lesson_plan_word_documents
FOR EACH ROW
EXECUTE FUNCTION public.validate_lesson_plan_word_document_write();

-- ============================================================================
-- 五、当前Word文档自动保存不可变版本
-- ============================================================================

CREATE OR REPLACE FUNCTION public.snapshot_lesson_plan_word_document_version()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.version = OLD.version THEN
        RETURN NEW;
    END IF;

    INSERT INTO public.lesson_plan_word_document_versions (
        id,
        lesson_plan_id,
        version,
        storage_key,
        file_sha256,
        parser_version,
        structure_schema_version,
        structure_json,
        semantic_markdown,
        semantic_markdown_hash,
        structure_hash,
        metrics_json,
        warnings_json,
        change_source,
        changed_by,
        change_summary,
        created_at
    )
    VALUES (
        gen_random_uuid(),
        NEW.lesson_plan_id,
        NEW.version,
        NEW.current_storage_key,
        NEW.current_file_sha256,
        NEW.parser_version,
        NEW.structure_schema_version,
        NEW.structure_json,
        NEW.semantic_markdown,
        NEW.semantic_markdown_hash,
        NEW.structure_hash,
        NEW.metrics_json,
        NEW.warnings_json,
        NEW.last_change_source,
        NEW.last_changed_by,
        NEW.last_change_summary,
        NOW()
    );

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_lp_word_document_snapshot
ON public.lesson_plan_word_documents;

CREATE TRIGGER trg_lp_word_document_snapshot
AFTER INSERT OR UPDATE OF version
ON public.lesson_plan_word_documents
FOR EACH ROW
EXECUTE FUNCTION public.snapshot_lesson_plan_word_document_version();

-- ============================================================================
-- 六、Word版本禁止原地修改
-- ============================================================================

CREATE OR REPLACE FUNCTION public.reject_lesson_plan_word_version_update()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION
        'Word保真历史版本是不可变快照，禁止UPDATE'
        USING ERRCODE = '23514';
END;
$$;

DROP TRIGGER IF EXISTS trg_lp_word_version_immutable
ON public.lesson_plan_word_document_versions;

CREATE TRIGGER trg_lp_word_version_immutable
BEFORE UPDATE
ON public.lesson_plan_word_document_versions
FOR EACH ROW
EXECUTE FUNCTION public.reject_lesson_plan_word_version_update();

-- ============================================================================
-- 七、平台语义正文或课程元信息被其它链路修改时标记Word文档失步
--
-- Word内容块正式写回链会在同一事务中同步更新当前Word文档。
-- 如果最终semantic_markdown与lesson_plans.content_markdown完全一致，
-- 本触发器不会误标stale。
-- ============================================================================

CREATE OR REPLACE FUNCTION public.mark_lesson_plan_word_document_stale()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    metadata_changed BOOLEAN;
BEGIN
    metadata_changed :=
        NEW.title IS DISTINCT FROM OLD.title
        OR NEW.subject IS DISTINCT FROM OLD.subject
        OR NEW.grade IS DISTINCT FROM OLD.grade
        OR NEW.topic IS DISTINCT FROM OLD.topic
        OR NEW.duration_minutes IS DISTINCT FROM OLD.duration_minutes;

    IF metadata_changed
        OR NEW.content_markdown IS DISTINCT FROM OLD.content_markdown THEN
        UPDATE public.lesson_plan_word_documents word_document
        SET
            status = 'stale',
            error_message =
                CASE
                    WHEN metadata_changed THEN
                        '教案标题、课程定位或课时时长已由其它链路修改，需要重新同步原格式Word文档'
                    ELSE
                        '平台语义正文已由其它链路修改，需要重新同步原格式Word文档'
                END,
            updated_at = NOW()
        WHERE word_document.lesson_plan_id = NEW.id
          AND word_document.status = 'active'
          AND (
              metadata_changed
              OR word_document.semantic_markdown
                 IS DISTINCT FROM COALESCE(NEW.content_markdown, '')
          );
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_lesson_plans_word_document_stale
ON public.lesson_plans;

CREATE TRIGGER trg_lesson_plans_word_document_stale
AFTER UPDATE OF
    title,
    subject,
    grade,
    topic,
    duration_minutes,
    content_markdown
ON public.lesson_plans
FOR EACH ROW
EXECUTE FUNCTION public.mark_lesson_plan_word_document_stale();

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '原格式Word教案导入会话、当前文档、不可变版本和语义失步保护已建立';
END
$$;
