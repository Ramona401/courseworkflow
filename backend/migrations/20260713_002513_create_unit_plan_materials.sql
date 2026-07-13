BEGIN;

-- unit_plan_materials
--
-- 大单元方案的持久化参考资料。
-- 第一阶段不保存原始PDF或Word文件，只保存浏览器提取出的文字和压缩摘要。
-- 可见范围继承unit_plans，不在本表重复存储scope字段。
CREATE TABLE IF NOT EXISTS public.unit_plan_materials (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,

    unit_plan_id uuid NOT NULL,
    material_type character varying(30) NOT NULL,
    file_name character varying(255) NOT NULL,

    content_text text DEFAULT ''::text NOT NULL,
    summary_text text DEFAULT ''::text NOT NULL,

    original_length integer DEFAULT 0 NOT NULL,
    summary_length integer DEFAULT 0 NOT NULL,

    uploaded_by uuid NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying NOT NULL,

    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,

    CONSTRAINT unit_plan_materials_unit_plan_fkey
        FOREIGN KEY (unit_plan_id)
        REFERENCES public.unit_plans(id)
        ON DELETE CASCADE,

    CONSTRAINT unit_plan_materials_uploaded_by_fkey
        FOREIGN KEY (uploaded_by)
        REFERENCES public.users(id),

    CONSTRAINT unit_plan_materials_type_check
        CHECK (
            material_type::text = ANY (
                ARRAY[
                    'textbook',
                    'teacher_guide',
                    'previous_unit_plan',
                    'teaching_requirement',
                    'excellent_case',
                    'other'
                ]::text[]
            )
        ),

    CONSTRAINT unit_plan_materials_status_check
        CHECK (
            status::text = ANY (
                ARRAY['active', 'archived']::text[]
            )
        ),

    CONSTRAINT unit_plan_materials_original_length_check
        CHECK (original_length >= 0),

    CONSTRAINT unit_plan_materials_summary_length_check
        CHECK (summary_length >= 0),

    CONSTRAINT unit_plan_materials_content_check
        CHECK (
            length(trim(content_text)) > 0
            OR length(trim(summary_text)) > 0
        )
);

COMMENT ON TABLE public.unit_plan_materials IS
    '大单元方案参考资料：保存浏览器提取文字与压缩摘要，可见范围继承unit_plans';

COMMENT ON COLUMN public.unit_plan_materials.material_type IS
    'textbook教材/teacher_guide教师用书/previous_unit_plan既有方案/teaching_requirement教研要求/excellent_case优秀课例/other其他';

COMMENT ON COLUMN public.unit_plan_materials.content_text IS
    '浏览器从docx或文字型PDF提取的原始文字；第一阶段不保存原始文件';

COMMENT ON COLUMN public.unit_plan_materials.summary_text IS
    '长资料经AI压缩后的结构化摘要；短资料可为空';

COMMENT ON COLUMN public.unit_plan_materials.status IS
    'active有效/archived软删除';

CREATE INDEX IF NOT EXISTS idx_upm_unit_status
    ON public.unit_plan_materials
    USING btree (unit_plan_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_upm_uploader
    ON public.unit_plan_materials
    USING btree (uploaded_by, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_upm_type
    ON public.unit_plan_materials
    USING btree (material_type, status);

ALTER TABLE public.unit_plan_materials OWNER TO postgres;

GRANT ALL ON TABLE public.unit_plan_materials TO tedna_user;

COMMIT;
