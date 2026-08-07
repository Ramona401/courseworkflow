\set ON_ERROR_STOP on

-- ============================================================================
-- 20260805_01_lesson_plan_word_fidelity_integrity_fix.sql
--
-- 修复原格式Word数据层在父教案、上传会话或用户删除时的外键动作冲突。
--
-- 修复内容：
--   1. confirmed导入会话因父教案删除而解除绑定时，自动恢复为parsed；
--      已过期会话自动进入expired，避免confirmed状态留下空lesson_plan_id；
--   2. 当前Word文档允许外键驱动的以下单向置空：
--        import_session_id：非空 → NULL；
--        created_by：非空 → NULL；
--        last_changed_by：非空 → NULL；
--      其它重新绑定或身份篡改仍然拒绝；
--   3. last_changed_by仅因用户删除而置空时，不要求Word版本递增；
--   4. 不可变版本表只允许外键驱动的changed_by非空→NULL，
--      文件、结构、正文、版本号和其它快照字段仍禁止修改；
--   5. 保持lesson_plan_id外键ON DELETE SET NULL，
--      使导入失败补偿后短时会话可以重新确认。
-- ============================================================================

BEGIN;

DO $$
BEGIN
    IF to_regclass(
        'public.lesson_plan_word_import_sessions'
    ) IS NULL THEN
        RAISE EXCEPTION
            'Word导入会话表不存在，不能应用完整性修复';
    END IF;

    IF to_regclass(
        'public.lesson_plan_word_documents'
    ) IS NULL THEN
        RAISE EXCEPTION
            'Word当前文档表不存在，不能应用完整性修复';
    END IF;

    IF to_regclass(
        'public.lesson_plan_word_document_versions'
    ) IS NULL THEN
        RAISE EXCEPTION
            'Word版本表不存在，不能应用完整性修复';
    END IF;
END
$$;

-- ============================================================================
-- 一、父教案删除或补偿删除时自动解除confirmed状态
-- ============================================================================

CREATE OR REPLACE FUNCTION
public.normalize_lesson_plan_word_import_session_unlink()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.status = 'confirmed'
        AND OLD.lesson_plan_id IS NOT NULL
        AND NEW.lesson_plan_id IS NULL THEN

        NEW.status :=
            CASE
                WHEN NEW.expires_at > NOW() THEN
                    'parsed'
                ELSE
                    'expired'
            END;

        NEW.confirmed_at := NULL;
        NEW.error_message := '';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
trg_lp_word_import_normalize_unlink
ON public.lesson_plan_word_import_sessions;

CREATE TRIGGER trg_lp_word_import_normalize_unlink
BEFORE UPDATE OF lesson_plan_id
ON public.lesson_plan_word_import_sessions
FOR EACH ROW
EXECUTE FUNCTION
public.normalize_lesson_plan_word_import_session_unlink();

COMMENT ON FUNCTION
public.normalize_lesson_plan_word_import_session_unlink()
IS
    '父教案删除使lesson_plan_id置空时，把confirmed Word导入会话安全恢复为parsed或expired';

-- ============================================================================
-- 二、修正当前Word文档写入边界
-- ============================================================================

CREATE OR REPLACE FUNCTION
public.validate_lesson_plan_word_document_write()
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

        NEW.created_at := COALESCE(
            NEW.created_at,
            NOW()
        );
        NEW.updated_at := NOW();

        RETURN NEW;
    END IF;

    -- lesson_plan_id、教育域、来源格式和原始母版始终不可修改。
    --
    -- import_session_id和created_by只允许外键删除动作导致的
    -- “原非空值 → NULL”，禁止改绑成其它会话或其它用户。
    IF NEW.lesson_plan_id
            IS DISTINCT FROM OLD.lesson_plan_id

        OR (
            NEW.import_session_id
                IS DISTINCT FROM OLD.import_session_id
            AND NOT (
                OLD.import_session_id IS NOT NULL
                AND NEW.import_session_id IS NULL
            )
        )

        OR (
            NEW.created_by
                IS DISTINCT FROM OLD.created_by
            AND NOT (
                OLD.created_by IS NOT NULL
                AND NEW.created_by IS NULL
            )
        )

        OR NEW.education_domain
            IS DISTINCT FROM OLD.education_domain

        OR NEW.source_format
            IS DISTINCT FROM OLD.source_format

        OR NEW.original_file_name
            IS DISTINCT FROM OLD.original_file_name

        OR NEW.original_storage_key
            IS DISTINCT FROM OLD.original_storage_key

        OR NEW.original_file_sha256
            IS DISTINCT FROM OLD.original_file_sha256 THEN

        RAISE EXCEPTION
            'Word保真文档身份、原始母版和教育域字段不可修改'
            USING ERRCODE = '23514';
    END IF;

    content_changed :=
        NEW.current_storage_key
            IS DISTINCT FROM OLD.current_storage_key

        OR NEW.current_file_sha256
            IS DISTINCT FROM OLD.current_file_sha256

        OR NEW.parser_version
            IS DISTINCT FROM OLD.parser_version

        OR NEW.structure_schema_version
            IS DISTINCT FROM OLD.structure_schema_version

        OR NEW.structure_json
            IS DISTINCT FROM OLD.structure_json

        OR NEW.semantic_markdown
            IS DISTINCT FROM OLD.semantic_markdown

        OR NEW.semantic_markdown_hash
            IS DISTINCT FROM OLD.semantic_markdown_hash

        OR NEW.structure_hash
            IS DISTINCT FROM OLD.structure_hash

        OR NEW.metrics_json
            IS DISTINCT FROM OLD.metrics_json

        OR NEW.warnings_json
            IS DISTINCT FROM OLD.warnings_json

        OR NEW.last_change_source
            IS DISTINCT FROM OLD.last_change_source

        -- last_changed_by正常改成其它用户仍属于版本元数据变化；
        -- 只有外键删除产生的非空→NULL不要求递增版本。
        OR (
            NEW.last_changed_by
                IS DISTINCT FROM OLD.last_changed_by
            AND NOT (
                OLD.last_changed_by IS NOT NULL
                AND NEW.last_changed_by IS NULL
            )
        )

        OR NEW.last_change_summary
            IS DISTINCT FROM OLD.last_change_summary

        OR NEW.generated_at
            IS DISTINCT FROM OLD.generated_at;

    IF content_changed
        AND NEW.version <> OLD.version + 1 THEN

        RAISE EXCEPTION
            'Word内容变化时版本必须从%递增到%，当前提交为%',
            OLD.version,
            OLD.version + 1,
            NEW.version
            USING
                ERRCODE = '23514',
                HINT =
                    '请在同一事务中读取当前版本并以current_version+1写入完整新快照';
    END IF;

    IF NOT content_changed
        AND NEW.version <> OLD.version THEN

        RAISE EXCEPTION
            '仅状态、错误或外键删除置空时不得修改Word文档版本，当前版本%被改为%',
            OLD.version,
            NEW.version
            USING ERRCODE = '23514';
    END IF;

    NEW.updated_at := NOW();

    RETURN NEW;
END;
$$;

-- ============================================================================
-- 三、不可变版本仅允许changed_by因用户删除而置空
-- ============================================================================

CREATE OR REPLACE FUNCTION
public.reject_lesson_plan_word_version_update()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.changed_by IS NOT NULL
        AND NEW.changed_by IS NULL
        AND (
            to_jsonb(NEW) - 'changed_by'
        ) IS NOT DISTINCT FROM (
            to_jsonb(OLD) - 'changed_by'
        ) THEN

        RETURN NEW;
    END IF;

    RAISE EXCEPTION
        'Word保真历史版本是不可变快照，禁止UPDATE'
        USING ERRCODE = '23514';
END;
$$;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        'Word保真父记录删除、用户删除和不可变版本外键置空冲突已修复';
END
$$;
