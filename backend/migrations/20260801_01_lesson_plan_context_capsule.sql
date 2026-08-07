\set ON_ERROR_STOP on

-- ============================================================================
-- 20260801_01_lesson_plan_context_capsule.sql
-- 备课核心共识胶囊、版本快照与原文证据路由
--
-- 产品目标：
--   1. 教师只通过自然语言形成、修正、否定和恢复备课共识；
--   2. 教师端不再维护机械的附件挂载清单或胶囊确认表单；
--   3. 课程大纲知识脉络和课本核心事实作为稳定权威来源贯穿备课；
--   4. 后续阶段继承前序阶段已经确认的有效结论；
--   5. 已被教师纠正、否定或替代的旧结论保留历史，但不得重新注入；
--   6. 普通对话只读取active短版胶囊，不重复加载课本或课程大纲全文；
--   7. 只有事实不足、发生冲突或教师要求准确依据时，才沿证据路由回溯原文；
--   8. 胶囊更新在AI主回复链之外旁路完成，不能阻塞流式输出。
--
-- 本迁移只建立数据库事实源和安全约束：
--   - 不自动调用AI；
--   - 不自动为存量教案生成胶囊；
--   - 不修改现有知识脉络表；
--   - 不改变现有教案阶段状态；
--   - 不把课本全文、课程大纲全文或完整提示词复制进胶囊表。
--
-- 数据模型：
--   lesson_plan_context_capsules
--       每个教案当前唯一的胶囊快照，供运行时快速读取和教师端轻量展示。
--
--   lesson_plan_context_capsule_versions
--       每次有效内容变化产生一份不可变版本，用于追溯、差异展示和安全回退。
--
--   lesson_plan_context_capsule_evidence
--       胶囊原子条目到课本页、大纲、教师消息、阶段产出等原文来源的召回路由。
--
-- 胶囊JSON建议结构由Go模型层正式约束，数据库只强制其为JSON对象。典型结构包含：
--   - course_core：课程大纲和课本共同支持的知识核心；
--   - teaching_consensus：教师已经明确确认的教学目标、主线和活动决定；
--   - constraints：禁止偏离、教学边界和已经订正的错误；
--   - open_questions：当前仍需讨论但尚未进入强约束的问题；
--   - deferred_items：暂时搁置、当前不应主动提起的内容；
--   - superseded_items：已经被替代或否定、后续不得重新注入的历史内容；
--   - stage_focus：当前阶段只负责注意力收窄，不切断全课共同记忆。
-- ============================================================================

BEGIN;

-- ============================================================================
-- 一、当前唯一胶囊
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.lesson_plan_context_capsules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_plan_id UUID NOT NULL,

    -- active：可进入后续AI提示词和教师端“本课共识”展示；
    -- stale：权威来源发生变化，旧胶囊不得继续用于正式事实约束；
    -- failed：最近一次更新失败，保留错误说明供补偿任务处理。
    status VARCHAR(20) NOT NULL DEFAULT 'stale',

    -- 每次胶囊有效内容变化必须严格递增1。
    -- 仅status/error_message变化不增加版本。
    version INTEGER NOT NULL DEFAULT 1,

    -- 领域JSON协议版本，便于后续兼容升级。
    schema_version INTEGER NOT NULL DEFAULT 1,

    -- 只表示当前工作的注意力阶段，不表示记忆被阶段切断。
    current_stage_code VARCHAR(100) NOT NULL DEFAULT '',

    -- 后端正式使用的完整核心胶囊，不含课本全文、大纲全文或助手完整提示词。
    capsule_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 教师端安全展示视图，只呈现当前核心理解、重要变化和待推敲事项。
    -- 不显示文件挂载清单、Token、提示词或内部处理状态。
    display_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 每轮AI运行时直接注入的稳定短版文本。
    context_text TEXT NOT NULL DEFAULT '',

    -- 本版本依赖的来源清单和来源版本摘要，不保存来源全文。
    source_manifest JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- capsule_json、context_text和source_manifest规范化后的统一SHA-256。
    source_hash VARCHAR(64) NOT NULL DEFAULT '',

    -- 产生本版本的教师主轮次，用于幂等更新和前端差异关联。
    last_turn_id VARCHAR(255) NOT NULL DEFAULT '',

    -- 新增、细化、修正、替代、否定、暂停、恢复或证据增强等变化摘要。
    last_update_reason VARCHAR(500) NOT NULL DEFAULT '',

    error_message TEXT NOT NULL DEFAULT '',

    generated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lesson_plan_context_capsules_plan_unique
        UNIQUE (lesson_plan_id),

    CONSTRAINT lesson_plan_context_capsules_plan_fkey
        FOREIGN KEY (lesson_plan_id)
        REFERENCES public.lesson_plans(id)
        ON DELETE CASCADE,

    CONSTRAINT lesson_plan_context_capsules_status_check
        CHECK (
            status IN (
                'active',
                'stale',
                'failed'
            )
        ),

    CONSTRAINT lesson_plan_context_capsules_version_check
        CHECK (
            version >= 1
            AND schema_version >= 1
        ),

    CONSTRAINT lesson_plan_context_capsules_json_check
        CHECK (
            jsonb_typeof(capsule_json) = 'object'
            AND jsonb_typeof(display_json) = 'object'
            AND jsonb_typeof(source_manifest) = 'object'
        ),

    CONSTRAINT lesson_plan_context_capsules_hash_check
        CHECK (
            source_hash = ''
            OR source_hash ~ '^[0-9a-f]{64}$'
        ),

    CONSTRAINT lesson_plan_context_capsules_reason_length_check
        CHECK (
            char_length(last_update_reason) <= 500
        ),

    CONSTRAINT lesson_plan_context_capsules_error_length_check
        CHECK (
            char_length(error_message) <= 4000
        ),

    CONSTRAINT lesson_plan_context_capsules_active_content_check
        CHECK (
            status <> 'active'
            OR (
                capsule_json <> '{}'::jsonb
                AND source_manifest <> '{}'::jsonb
                AND btrim(context_text) <> ''
                AND source_hash <> ''
                AND generated_at IS NOT NULL
            )
        )
);

CREATE INDEX IF NOT EXISTS idx_lesson_plan_context_capsules_status
ON public.lesson_plan_context_capsules (
    status,
    updated_at DESC
);

CREATE INDEX IF NOT EXISTS idx_lesson_plan_context_capsules_stage
ON public.lesson_plan_context_capsules (
    current_stage_code,
    status
);

COMMENT ON TABLE public.lesson_plan_context_capsules
IS
    '每个教案当前唯一的备课核心共识胶囊；教师通过自然语言形成共识，运行时只注入active短版核心';

COMMENT ON COLUMN public.lesson_plan_context_capsules.capsule_json
IS
    '课程核心、教师共识、边界、待推敲、搁置及已替代内容的结构化快照，不保存来源全文';

COMMENT ON COLUMN public.lesson_plan_context_capsules.display_json
IS
    '教师端本课共识安全展示视图；禁止退化为附件文件名、挂载状态和Token统计清单';

COMMENT ON COLUMN public.lesson_plan_context_capsules.context_text
IS
    'AI运行时快速注入的短版核心记忆；旁路更新失败不得阻塞正在进行的流式回复';

COMMENT ON COLUMN public.lesson_plan_context_capsules.source_manifest
IS
    '胶囊依赖的课本页、大纲知识脉络、阶段产出和教师共识版本摘要，不保存完整原文';

-- ============================================================================
-- 二、不可变版本快照
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.lesson_plan_context_capsule_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_plan_id UUID NOT NULL,
    version INTEGER NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1,
    current_stage_code VARCHAR(100) NOT NULL DEFAULT '',
    capsule_json JSONB NOT NULL,
    display_json JSONB NOT NULL,
    context_text TEXT NOT NULL,
    source_manifest JSONB NOT NULL,
    source_hash VARCHAR(64) NOT NULL DEFAULT '',
    last_turn_id VARCHAR(255) NOT NULL DEFAULT '',
    update_reason VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lesson_plan_context_capsule_versions_unique
        UNIQUE (
            lesson_plan_id,
            version
        ),

    CONSTRAINT lesson_plan_context_capsule_versions_plan_fkey
        FOREIGN KEY (lesson_plan_id)
        REFERENCES public.lesson_plans(id)
        ON DELETE CASCADE,

    CONSTRAINT lesson_plan_context_capsule_versions_version_check
        CHECK (
            version >= 1
            AND schema_version >= 1
        ),

    CONSTRAINT lesson_plan_context_capsule_versions_json_check
        CHECK (
            jsonb_typeof(capsule_json) = 'object'
            AND jsonb_typeof(display_json) = 'object'
            AND jsonb_typeof(source_manifest) = 'object'
        ),

    CONSTRAINT lesson_plan_context_capsule_versions_hash_check
        CHECK (
            source_hash = ''
            OR source_hash ~ '^[0-9a-f]{64}$'
        ),

    CONSTRAINT lesson_plan_context_capsule_versions_reason_length_check
        CHECK (
            char_length(update_reason) <= 500
        )
);

CREATE INDEX IF NOT EXISTS idx_lesson_plan_context_capsule_versions_plan
ON public.lesson_plan_context_capsule_versions (
    lesson_plan_id,
    version DESC
);

CREATE INDEX IF NOT EXISTS idx_lesson_plan_context_capsule_versions_turn
ON public.lesson_plan_context_capsule_versions (
    lesson_plan_id,
    last_turn_id
);

COMMENT ON TABLE public.lesson_plan_context_capsule_versions
IS
    '备课核心共识胶囊的不可变历史版本，用于变化追溯、差异展示和安全恢复';

-- ============================================================================
-- 三、胶囊条目到原文的精准召回路由
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.lesson_plan_context_capsule_evidence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_plan_id UUID NOT NULL,
    capsule_version INTEGER NOT NULL,

    -- 胶囊内稳定原子条目键，例如：
    -- course_core.buoyancy_rule
    -- teaching_consensus.inquiry_before_formula
    -- constraints.no_weight_proportionality
    item_key VARCHAR(160) NOT NULL,

    source_type VARCHAR(40) NOT NULL,
    source_id VARCHAR(255) NOT NULL DEFAULT '',
    source_title VARCHAR(500) NOT NULL DEFAULT '',

    -- 页码、章节、段落范围、消息ID、阶段代码、片段位置和召回关键词等。
    locator_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 来源当前版本哈希和本条证据片段哈希。
    source_hash VARCHAR(64) NOT NULL DEFAULT '',
    excerpt_hash VARCHAR(64) NOT NULL DEFAULT '',

    -- 只允许保存支持该核心逻辑所需的短证据，不保存整份附件或大纲全文。
    evidence_excerpt TEXT NOT NULL DEFAULT '',

    -- teacher_explicit：教师明确表达；
    -- source_verified：权威原文直接支持；
    -- teacher_source_confirmed：教师表达与权威来源共同支持；
    -- ai_inferred：AI暂时推断，只能进入待推敲区，不能作为强约束。
    authority VARCHAR(40) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lesson_plan_context_capsule_evidence_version_fkey
        FOREIGN KEY (
            lesson_plan_id,
            capsule_version
        )
        REFERENCES public.lesson_plan_context_capsule_versions (
            lesson_plan_id,
            version
        )
        ON DELETE CASCADE,

    CONSTRAINT lesson_plan_context_capsule_evidence_unique
        UNIQUE (
            lesson_plan_id,
            capsule_version,
            item_key,
            source_type,
            source_id,
            excerpt_hash
        ),

    CONSTRAINT lesson_plan_context_capsule_evidence_item_key_check
        CHECK (
            item_key ~ '^[a-z0-9][a-z0-9._:-]{0,159}$'
        ),

    CONSTRAINT lesson_plan_context_capsule_evidence_source_type_check
        CHECK (
            source_type IN (
                'textbook_page',
                'course_outline',
                'teacher_message',
                'stage_output',
                'unit_plan',
                'class_profile',
                'reference_material',
                'system'
            )
        ),

    CONSTRAINT lesson_plan_context_capsule_evidence_authority_check
        CHECK (
            authority IN (
                'teacher_explicit',
                'source_verified',
                'teacher_source_confirmed',
                'ai_inferred'
            )
        ),

    CONSTRAINT lesson_plan_context_capsule_evidence_locator_check
        CHECK (
            jsonb_typeof(locator_json) = 'object'
        ),

    CONSTRAINT lesson_plan_context_capsule_evidence_hash_check
        CHECK (
            (source_hash = '' OR source_hash ~ '^[0-9a-f]{64}$')
            AND
            (excerpt_hash = '' OR excerpt_hash ~ '^[0-9a-f]{64}$')
        ),

    CONSTRAINT lesson_plan_context_capsule_evidence_excerpt_length_check
        CHECK (
            char_length(evidence_excerpt) <= 2000
        )
);

CREATE INDEX IF NOT EXISTS idx_lesson_plan_context_capsule_evidence_item
ON public.lesson_plan_context_capsule_evidence (
    lesson_plan_id,
    capsule_version,
    item_key
);

CREATE INDEX IF NOT EXISTS idx_lesson_plan_context_capsule_evidence_source
ON public.lesson_plan_context_capsule_evidence (
    source_type,
    source_id
);

COMMENT ON TABLE public.lesson_plan_context_capsule_evidence
IS
    '胶囊原子条目到课本、大纲、教师消息和阶段产出的精准原文召回路由';

COMMENT ON COLUMN public.lesson_plan_context_capsule_evidence.evidence_excerpt
IS
    '支持当前核心逻辑所需的短证据，最长2000字符；禁止复制整份课本、大纲或附件全文';

-- ============================================================================
-- 四、当前胶囊版本递增与写入完整性保护
-- ============================================================================

CREATE OR REPLACE FUNCTION public.validate_lesson_plan_context_capsule_write()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    content_changed BOOLEAN;
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.version <> 1 THEN
            RAISE EXCEPTION
                '备课核心胶囊首个版本必须为1，当前为%',
                NEW.version
                USING ERRCODE = '23514';
        END IF;

        NEW.created_at := COALESCE(NEW.created_at, NOW());
        NEW.updated_at := NOW();
        RETURN NEW;
    END IF;

    content_changed :=
        NEW.schema_version IS DISTINCT FROM OLD.schema_version
        OR NEW.current_stage_code IS DISTINCT FROM OLD.current_stage_code
        OR NEW.capsule_json IS DISTINCT FROM OLD.capsule_json
        OR NEW.display_json IS DISTINCT FROM OLD.display_json
        OR NEW.context_text IS DISTINCT FROM OLD.context_text
        OR NEW.source_manifest IS DISTINCT FROM OLD.source_manifest
        OR NEW.source_hash IS DISTINCT FROM OLD.source_hash
        OR NEW.last_turn_id IS DISTINCT FROM OLD.last_turn_id
        OR NEW.last_update_reason IS DISTINCT FROM OLD.last_update_reason
        OR NEW.generated_at IS DISTINCT FROM OLD.generated_at;

    IF content_changed AND NEW.version <> OLD.version + 1 THEN
        RAISE EXCEPTION
            '备课核心胶囊内容变化时版本必须从%递增到%，当前提交为%',
            OLD.version,
            OLD.version + 1,
            NEW.version
            USING
                ERRCODE = '23514',
                HINT = '请在同一事务中读取当前版本并以current_version+1写入完整新快照';
    END IF;

    IF NOT content_changed AND NEW.version <> OLD.version THEN
        RAISE EXCEPTION
            '仅状态或错误信息变化时不得修改胶囊版本，当前版本%被改为%',
            OLD.version,
            NEW.version
            USING ERRCODE = '23514';
    END IF;

    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_lesson_plan_context_capsule_validate
ON public.lesson_plan_context_capsules;

CREATE TRIGGER trg_lesson_plan_context_capsule_validate
BEFORE INSERT OR UPDATE
ON public.lesson_plan_context_capsules
FOR EACH ROW
EXECUTE FUNCTION public.validate_lesson_plan_context_capsule_write();

COMMENT ON FUNCTION public.validate_lesson_plan_context_capsule_write()
IS
    '保证胶囊首版为1、有效内容变化严格递增1，状态失效操作不伪造新内容版本';

-- ============================================================================
-- 五、每次内容版本变化自动保存不可变快照
-- ============================================================================

CREATE OR REPLACE FUNCTION public.snapshot_lesson_plan_context_capsule_version()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.version = OLD.version THEN
        RETURN NEW;
    END IF;

    INSERT INTO public.lesson_plan_context_capsule_versions (
        id,
        lesson_plan_id,
        version,
        schema_version,
        current_stage_code,
        capsule_json,
        display_json,
        context_text,
        source_manifest,
        source_hash,
        last_turn_id,
        update_reason,
        created_at
    )
    VALUES (
        gen_random_uuid(),
        NEW.lesson_plan_id,
        NEW.version,
        NEW.schema_version,
        NEW.current_stage_code,
        NEW.capsule_json,
        NEW.display_json,
        NEW.context_text,
        NEW.source_manifest,
        NEW.source_hash,
        NEW.last_turn_id,
        NEW.last_update_reason,
        NOW()
    );

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_lesson_plan_context_capsule_snapshot
ON public.lesson_plan_context_capsules;

CREATE TRIGGER trg_lesson_plan_context_capsule_snapshot
AFTER INSERT OR UPDATE OF version
ON public.lesson_plan_context_capsules
FOR EACH ROW
EXECUTE FUNCTION public.snapshot_lesson_plan_context_capsule_version();

COMMENT ON FUNCTION public.snapshot_lesson_plan_context_capsule_version()
IS
    '当前胶囊首次创建或版本递增时，自动保存一份不可变完整快照';

-- ============================================================================
-- 六、教案权威来源挂载或课程定位变化时使胶囊失效
--
-- 不监听conversation_log：
--   教师每轮自然语言消息由旁路胶囊更新器产生下一版本；
--   若每条消息都先把当前胶囊标stale，会破坏“上一版胶囊+当前消息立即流式回答”。
-- ============================================================================

CREATE OR REPLACE FUNCTION public.mark_lesson_plan_context_capsule_stale_from_plan()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    source_changed BOOLEAN;
BEGIN
    source_changed :=
        NEW.course_outline_id IS DISTINCT FROM OLD.course_outline_id
        OR NEW.textbook_page_ids IS DISTINCT FROM OLD.textbook_page_ids
        OR NEW.unit_plan_id IS DISTINCT FROM OLD.unit_plan_id
        OR NEW.class_profile_id IS DISTINCT FROM OLD.class_profile_id
        OR btrim(COALESCE(NEW.subject, ''))
           IS DISTINCT FROM btrim(COALESCE(OLD.subject, ''))
        OR btrim(COALESCE(NEW.grade, ''))
           IS DISTINCT FROM btrim(COALESCE(OLD.grade, ''))
        OR btrim(COALESCE(NEW.topic, ''))
           IS DISTINCT FROM btrim(COALESCE(OLD.topic, ''));

    IF source_changed THEN
        UPDATE public.lesson_plan_context_capsules
        SET
            status = 'stale',
            error_message =
                '教案课程定位或权威资料挂载已经变化，需要旁路重建核心共识胶囊',
            updated_at = NOW()
        WHERE lesson_plan_id = NEW.id
          AND status <> 'stale';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_lesson_plans_context_capsule_stale
ON public.lesson_plans;

CREATE TRIGGER trg_lesson_plans_context_capsule_stale
AFTER UPDATE OF
    course_outline_id,
    textbook_page_ids,
    unit_plan_id,
    class_profile_id,
    subject,
    grade,
    topic
ON public.lesson_plans
FOR EACH ROW
EXECUTE FUNCTION public.mark_lesson_plan_context_capsule_stale_from_plan();

COMMENT ON FUNCTION public.mark_lesson_plan_context_capsule_stale_from_plan()
IS
    '教案课程定位或权威资料挂载变化后，使旧核心共识胶囊失效；普通对话消息不触发失效';

-- ============================================================================
-- 七、课程大纲正文、范围或版本属性变化时使依赖胶囊失效
-- ============================================================================

CREATE OR REPLACE FUNCTION public.mark_lesson_plan_context_capsule_stale_from_outline()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    old_row JSONB;
    new_row JSONB;
    source_changed BOOLEAN;
BEGIN
    old_row := to_jsonb(OLD);
    new_row := to_jsonb(NEW);

    source_changed :=
        new_row ->> 'subject' IS DISTINCT FROM old_row ->> 'subject'
        OR new_row ->> 'grade' IS DISTINCT FROM old_row ->> 'grade'
        OR new_row ->> 'title' IS DISTINCT FROM old_row ->> 'title'
        OR new_row ->> 'content' IS DISTINCT FROM old_row ->> 'content'
        OR new_row ->> 'publisher' IS DISTINCT FROM old_row ->> 'publisher'
        OR new_row ->> 'volume' IS DISTINCT FROM old_row ->> 'volume'
        OR new_row ->> 'school_system' IS DISTINCT FROM old_row ->> 'school_system'
        OR new_row ->> 'status' IS DISTINCT FROM old_row ->> 'status';

    IF source_changed THEN
        UPDATE public.lesson_plan_context_capsules capsule
        SET
            status = 'stale',
            error_message =
                '胶囊所依据的课程大纲已经变化，需要使用最新active知识脉络重建',
            updated_at = NOW()
        FROM public.lesson_plans lesson_plan
        WHERE capsule.lesson_plan_id = lesson_plan.id
          AND lesson_plan.course_outline_id = NEW.id
          AND capsule.status <> 'stale';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_course_outlines_context_capsule_stale
ON public.course_outlines;

CREATE TRIGGER trg_course_outlines_context_capsule_stale
AFTER UPDATE
ON public.course_outlines
FOR EACH ROW
EXECUTE FUNCTION public.mark_lesson_plan_context_capsule_stale_from_outline();

COMMENT ON FUNCTION public.mark_lesson_plan_context_capsule_stale_from_outline()
IS
    '课程大纲正文、范围、教材属性或状态变化后，使依赖核心共识胶囊失效';

-- ============================================================================
-- 八、已挂载课本页面OCR、章节、属性或状态变化时使胶囊失效
--
-- 使用to_jsonb读取字段，避免数据库存量列的可空差异影响触发器创建。
-- 课本使用次数等纯统计字段变化不会触发胶囊失效。
-- ============================================================================

CREATE OR REPLACE FUNCTION public.mark_lesson_plan_context_capsule_stale_from_textbook()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    affected_page_id UUID;
    old_row JSONB;
    new_row JSONB;
    source_changed BOOLEAN;
BEGIN
    IF TG_OP = 'DELETE' THEN
        affected_page_id := OLD.id;
        source_changed := TRUE;
    ELSE
        affected_page_id := NEW.id;
        old_row := to_jsonb(OLD);
        new_row := to_jsonb(NEW);

        source_changed :=
            new_row ->> 'ocr_text' IS DISTINCT FROM old_row ->> 'ocr_text'
            OR new_row ->> 'status' IS DISTINCT FROM old_row ->> 'status'
            OR new_row ->> 'textbook_name' IS DISTINCT FROM old_row ->> 'textbook_name'
            OR new_row ->> 'chapter' IS DISTINCT FROM old_row ->> 'chapter'
            OR new_row ->> 'subject' IS DISTINCT FROM old_row ->> 'subject'
            OR new_row ->> 'grade_range' IS DISTINCT FROM old_row ->> 'grade_range';
    END IF;

    IF source_changed THEN
        UPDATE public.lesson_plan_context_capsules capsule
        SET
            status = 'stale',
            error_message =
                '胶囊所依据的课本页面已经变化，需要重新提取课本核心逻辑',
            updated_at = NOW()
        FROM public.lesson_plans lesson_plan
        WHERE capsule.lesson_plan_id = lesson_plan.id
          AND capsule.status <> 'stale'
          AND CASE
              WHEN btrim(COALESCE(lesson_plan.textbook_page_ids::text, ''))
                   IN ('', '[]', 'null') THEN FALSE
              ELSE lesson_plan.textbook_page_ids::jsonb
                   @> jsonb_build_array(affected_page_id::text)
          END;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_textbook_pages_context_capsule_stale
ON public.textbook_pages;

CREATE TRIGGER trg_textbook_pages_context_capsule_stale
AFTER UPDATE OR DELETE
ON public.textbook_pages
FOR EACH ROW
EXECUTE FUNCTION public.mark_lesson_plan_context_capsule_stale_from_textbook();

COMMENT ON FUNCTION public.mark_lesson_plan_context_capsule_stale_from_textbook()
IS
    '已挂载课本页面的OCR、章节、课程属性、状态变化或删除后，使依赖核心共识胶囊失效';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '备课核心共识胶囊、不可变版本、原文证据路由和来源失效保护已建立';
END
$$;
