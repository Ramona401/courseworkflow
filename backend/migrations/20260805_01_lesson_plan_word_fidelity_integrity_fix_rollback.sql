\set ON_ERROR_STOP on

-- ============================================================================
-- 20260805_01_lesson_plan_word_fidelity_integrity_fix_rollback.sql
--
-- 回滚20260805_01完整性修复，恢复20260804_02迁移最初的触发器行为。
--
-- 注意：
--   回滚后会重新出现父教案删除、用户删除与Word外键置空的冲突。
--   正式回滚前必须再次完整备份数据库。
-- ============================================================================

BEGIN;

DROP TRIGGER IF EXISTS
trg_lp_word_import_normalize_unlink
ON public.lesson_plan_word_import_sessions;

DROP FUNCTION IF EXISTS
public.normalize_lesson_plan_word_import_session_unlink();

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

    IF NEW.lesson_plan_id
            IS DISTINCT FROM OLD.lesson_plan_id
        OR NEW.import_session_id
            IS DISTINCT FROM OLD.import_session_id
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
        OR NEW.last_changed_by
            IS DISTINCT FROM OLD.last_changed_by
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
            '仅状态或错误变化时不得修改Word文档版本，当前版本%被改为%',
            OLD.version,
            NEW.version
            USING ERRCODE = '23514';
    END IF;

    NEW.updated_at := NOW();

    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION
public.reject_lesson_plan_word_version_update()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION
        'Word保真历史版本是不可变快照，禁止UPDATE'
        USING ERRCODE = '23514';
END;
$$;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        'Word保真删除与外键完整性修复已回滚';
END
$$;
