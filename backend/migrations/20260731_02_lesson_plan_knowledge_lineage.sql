\set ON_ERROR_STOP on

-- ============================================================================
-- 20260731_02_lesson_plan_knowledge_lineage.sql
-- 教案知识脉络快照
--
-- 本迁移只建立存储与失效保护：
--   - 不自动调用AI；
--   - 不在创建教案时生成；
--   - 不在绑定课程大纲时生成；
--   - 不根据lesson_plans.topic直接猜测；
--   - 不把课程大纲全文保存到知识脉络表。
--
-- 正式时序：
--   1. 教师先确定具体课文、章节、主题或课时范围；
--   2. 教师确认教学目标、核心知识点、学习深度和排除内容；
--   3. 教师完成教学分析阶段；
--   4. 后端从已经确认的分析对话提取课程锚点；
--   5. 每条锚点证据必须能在分析对话原文中找到；
--   6. 锚点严格合格后才读取唯一课程大纲；
--   7. 当前知识节点必须与确认知识点逐项对应；
--   8. 大纲证据必须能在课程大纲原文中找到；
--   9. 大纲没有明确证据时只能记录证据缺口；
--  10. 后续阶段只注入短版知识脉络，不再重复注入大纲全文。
--
-- 数据库硬边界：
--   - 精确挂载课程大纲的教案，从analyze离开前必须已有匹配的active知识脉络；
--   - 任何绕过HTTP Service直接更新current_stage的路径也会被数据库拒绝。
--
-- 失效来源：
--   - 教案学科、年级、课题或课程大纲绑定变化；
--   - 教师在教学分析阶段继续修改对话；
--   - 教学分析结构化产出或摘要变化；
--   - 课程大纲正文、教材属性或状态变化。
-- ============================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS public.lesson_plan_knowledge_lineages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_plan_id UUID NOT NULL,
    course_outline_id UUID NOT NULL,

    -- generating：正在生成；
    -- active：可供后续阶段使用；
    -- stale：来源已经改变；
    -- failed：本次提取失败。
    status VARCHAR(20) NOT NULL DEFAULT 'generating',

    -- 教师确认的课文、教学范围、目标、知识点和学习深度。
    anchor_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 围绕确认锚点，从课程大纲定向提取的知识脉络。
    lineage_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 后续阶段注入的短版统一上下文，不含课程大纲全文。
    context_text TEXT NOT NULL DEFAULT '',

    -- 规范化锚点JSON与课程大纲全文的SHA-256。
    anchor_hash VARCHAR(64) NOT NULL DEFAULT '',
    outline_hash VARCHAR(64) NOT NULL DEFAULT '',

    confirmed_stage_code VARCHAR(100) NOT NULL DEFAULT 'analyze',

    -- 只用于记录提取时对应的分析阶段产出版本时间。
    -- 并发正确性由服务层内容哈希复核，不依赖updated_at本身。
    confirmed_stage_output_updated_at TIMESTAMPTZ,

    model_used VARCHAR(255) NOT NULL DEFAULT '',
    tokens_used INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',

    generated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lesson_plan_knowledge_lineages_plan_unique
        UNIQUE (lesson_plan_id),

    CONSTRAINT lesson_plan_knowledge_lineages_plan_fkey
        FOREIGN KEY (lesson_plan_id)
        REFERENCES public.lesson_plans(id)
        ON DELETE CASCADE,

    CONSTRAINT lesson_plan_knowledge_lineages_outline_fkey
        FOREIGN KEY (course_outline_id)
        REFERENCES public.course_outlines(id)
        ON DELETE RESTRICT,

    CONSTRAINT lesson_plan_knowledge_lineages_status_check
        CHECK (
            status IN (
                'generating',
                'active',
                'stale',
                'failed'
            )
        ),

    CONSTRAINT lesson_plan_knowledge_lineages_anchor_object_check
        CHECK (
            jsonb_typeof(anchor_snapshot) = 'object'
        ),

    CONSTRAINT lesson_plan_knowledge_lineages_lineage_object_check
        CHECK (
            jsonb_typeof(lineage_snapshot) = 'object'
        ),

    CONSTRAINT lesson_plan_knowledge_lineages_tokens_check
        CHECK (
            tokens_used >= 0
        ),

    CONSTRAINT lesson_plan_knowledge_lineages_hash_check
        CHECK (
            (anchor_hash = '' OR anchor_hash ~ '^[0-9a-f]{64}$')
            AND
            (outline_hash = '' OR outline_hash ~ '^[0-9a-f]{64}$')
        ),

    CONSTRAINT lesson_plan_knowledge_lineages_active_content_check
        CHECK (
            status <> 'active'
            OR (
                anchor_snapshot <> '{}'::jsonb
                AND lineage_snapshot <> '{}'::jsonb
                AND btrim(context_text) <> ''
                AND anchor_hash <> ''
                AND outline_hash <> ''
                AND confirmed_stage_output_updated_at IS NOT NULL
                AND generated_at IS NOT NULL
            )
        )
);

CREATE INDEX IF NOT EXISTS
    idx_lesson_plan_knowledge_lineages_status
ON public.lesson_plan_knowledge_lineages (
    status,
    updated_at DESC
);

CREATE INDEX IF NOT EXISTS
    idx_lesson_plan_knowledge_lineages_outline
ON public.lesson_plan_knowledge_lineages (
    course_outline_id,
    status
);

COMMENT ON TABLE public.lesson_plan_knowledge_lineages
IS
    '教师确认本课范围、目标和知识点后，围绕唯一课程大纲定向提取的知识脉络快照';

COMMENT ON COLUMN public.lesson_plan_knowledge_lineages.anchor_snapshot
IS
    '教师完成教学分析后确认的课程锚点JSON；证据必须来自分析对话原文';

COMMENT ON COLUMN public.lesson_plan_knowledge_lineages.lineage_snapshot
IS
    '与确认知识点逐项对应的本课知识位置、前置、后续、误区和评价证据JSON';

COMMENT ON COLUMN public.lesson_plan_knowledge_lineages.context_text
IS
    '课程设计、教案撰写、审核和修订统一注入的短版知识脉络，不含大纲全文';

-- ============================================================================
-- 零、精确挂载课程大纲的教案离开analyze前必须已有active知识脉络
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.enforce_lesson_plan_knowledge_lineage_before_leaving_analyze()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.deleted_at IS NULL
       AND btrim(COALESCE(OLD.current_stage, '')) = 'analyze'
       AND btrim(COALESCE(NEW.current_stage, '')) <> 'analyze'
       AND NEW.course_outline_id IS NOT NULL
       AND NOT EXISTS (
           SELECT 1
           FROM public.lesson_plan_knowledge_lineages lineage
           WHERE lineage.lesson_plan_id = NEW.id
             AND lineage.course_outline_id = NEW.course_outline_id
             AND lineage.status = 'active'
             AND lineage.confirmed_stage_code = 'analyze'
             AND lineage.anchor_snapshot <> '{}'::jsonb
             AND lineage.lineage_snapshot <> '{}'::jsonb
             AND btrim(lineage.context_text) <> ''
             AND lineage.anchor_hash <> ''
             AND lineage.outline_hash <> ''
             AND lineage.generated_at IS NOT NULL
       ) THEN
        RAISE EXCEPTION
            '已关联课程大纲的教案必须先完成教学分析并生成active知识脉络'
            USING
                ERRCODE = '23514',
                HINT =
                    '请在教学分析阶段确认课文、教学范围、教学目标和知识点后，通过正式阶段推进入口生成知识脉络';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
    trg_lesson_plans_require_knowledge_lineage
ON public.lesson_plans;

CREATE TRIGGER
    trg_lesson_plans_require_knowledge_lineage
BEFORE UPDATE OF current_stage
ON public.lesson_plans
FOR EACH ROW
EXECUTE FUNCTION
    public.enforce_lesson_plan_knowledge_lineage_before_leaving_analyze();

COMMENT ON FUNCTION
    public.enforce_lesson_plan_knowledge_lineage_before_leaving_analyze()
IS
    '精确挂载课程大纲的教案离开教学分析阶段前，强制要求存在匹配的active知识脉络';

-- ============================================================================
-- 一、教案定位、大纲绑定或分析阶段对话发生变化时，使快照失效
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.mark_lesson_plan_knowledge_lineage_stale_from_plan()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    source_changed BOOLEAN;
BEGIN
    source_changed :=
        NEW.course_outline_id IS DISTINCT FROM OLD.course_outline_id
        OR btrim(COALESCE(NEW.subject, ''))
           IS DISTINCT FROM btrim(COALESCE(OLD.subject, ''))
        OR btrim(COALESCE(NEW.grade, ''))
           IS DISTINCT FROM btrim(COALESCE(OLD.grade, ''))
        OR btrim(COALESCE(NEW.topic, ''))
           IS DISTINCT FROM btrim(COALESCE(OLD.topic, ''))
        OR (
            NEW.conversation_log IS DISTINCT FROM OLD.conversation_log
            AND (
                btrim(COALESCE(OLD.current_stage, '')) = 'analyze'
                OR btrim(COALESCE(NEW.current_stage, '')) = 'analyze'
            )
        );

    IF source_changed THEN
        UPDATE public.lesson_plan_knowledge_lineages
        SET
            status = 'stale',
            error_message =
                '教案定位、课程大纲绑定或教学分析对话已经变化，需要重新确认后生成',
            updated_at = NOW()
        WHERE lesson_plan_id = NEW.id
          AND status <> 'stale';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
    trg_lesson_plans_knowledge_lineage_stale
ON public.lesson_plans;

CREATE TRIGGER
    trg_lesson_plans_knowledge_lineage_stale
AFTER UPDATE OF
    course_outline_id,
    subject,
    grade,
    topic,
    conversation_log
ON public.lesson_plans
FOR EACH ROW
EXECUTE FUNCTION
    public.mark_lesson_plan_knowledge_lineage_stale_from_plan();

COMMENT ON FUNCTION
    public.mark_lesson_plan_knowledge_lineage_stale_from_plan()
IS
    '教案定位、大纲绑定或教学分析阶段对话变化后，使知识脉络失效';

-- ============================================================================
-- 二、课程大纲正文、教材属性或状态变化时，使依赖快照失效
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.mark_lesson_plan_knowledge_lineage_stale_from_outline()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF btrim(COALESCE(NEW.subject, ''))
          IS DISTINCT FROM btrim(COALESCE(OLD.subject, ''))
       OR btrim(COALESCE(NEW.grade, ''))
          IS DISTINCT FROM btrim(COALESCE(OLD.grade, ''))
       OR btrim(COALESCE(NEW.title, ''))
          IS DISTINCT FROM btrim(COALESCE(OLD.title, ''))
       OR btrim(COALESCE(NEW.content, ''))
          IS DISTINCT FROM btrim(COALESCE(OLD.content, ''))
       OR btrim(COALESCE(NEW.publisher, ''))
          IS DISTINCT FROM btrim(COALESCE(OLD.publisher, ''))
       OR btrim(COALESCE(NEW.volume, ''))
          IS DISTINCT FROM btrim(COALESCE(OLD.volume, ''))
       OR btrim(COALESCE(NEW.school_system, ''))
          IS DISTINCT FROM btrim(COALESCE(OLD.school_system, ''))
       OR btrim(COALESCE(NEW.status, ''))
          IS DISTINCT FROM btrim(COALESCE(OLD.status, '')) THEN

        UPDATE public.lesson_plan_knowledge_lineages
        SET
            status = 'stale',
            error_message =
                '所依据的课程大纲已经变化，需要使用确认课程锚点重新生成',
            updated_at = NOW()
        WHERE course_outline_id = NEW.id
          AND status <> 'stale';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
    trg_course_outlines_knowledge_lineage_stale
ON public.course_outlines;

CREATE TRIGGER
    trg_course_outlines_knowledge_lineage_stale
AFTER UPDATE OF
    subject,
    grade,
    title,
    content,
    publisher,
    volume,
    school_system,
    status
ON public.course_outlines
FOR EACH ROW
EXECUTE FUNCTION
    public.mark_lesson_plan_knowledge_lineage_stale_from_outline();

COMMENT ON FUNCTION
    public.mark_lesson_plan_knowledge_lineage_stale_from_outline()
IS
    '课程大纲正文、教材属性或状态变化后，使依赖知识脉络失效';

-- ============================================================================
-- 三、教学分析结构化结果或摘要变化时，使旧快照失效
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.mark_lesson_plan_knowledge_lineage_stale_from_stage_output()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    affected_plan_id UUID;
    affected_stage_code VARCHAR;
    content_changed BOOLEAN;
BEGIN
    IF TG_OP = 'DELETE' THEN
        affected_plan_id := OLD.lesson_plan_id;
        affected_stage_code := OLD.stage_code;
        content_changed := TRUE;
    ELSE
        affected_plan_id := NEW.lesson_plan_id;
        affected_stage_code := NEW.stage_code;
        content_changed :=
            NEW.structured_output IS DISTINCT FROM OLD.structured_output
            OR NEW.narrative_output IS DISTINCT FROM OLD.narrative_output;
    END IF;

    IF affected_stage_code = 'analyze'
       AND content_changed THEN
        UPDATE public.lesson_plan_knowledge_lineages
        SET
            status = 'stale',
            error_message =
                '教学分析结论已经变化，需要重新确认课程目标和知识点后生成',
            updated_at = NOW()
        WHERE lesson_plan_id = affected_plan_id
          AND status <> 'stale';
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS
    trg_stage_outputs_knowledge_lineage_stale
ON public.workshop_stage_outputs;

CREATE TRIGGER
    trg_stage_outputs_knowledge_lineage_stale
AFTER UPDATE OF
    structured_output,
    narrative_output
OR DELETE
ON public.workshop_stage_outputs
FOR EACH ROW
EXECUTE FUNCTION
    public.mark_lesson_plan_knowledge_lineage_stale_from_stage_output();

COMMENT ON FUNCTION
    public.mark_lesson_plan_knowledge_lineage_stale_from_stage_output()
IS
    '教学分析结构化结果或摘要变化后，使已有知识脉络失效；单纯完成状态变化不触发';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '教案知识脉络存储、离开analyze硬闸与来源失效保护已建立；本迁移不会自动提取知识脉络';
END
$$;
