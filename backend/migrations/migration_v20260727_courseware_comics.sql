-- ============================================================================
-- migration_v20260727_courseware_comics.sql
-- 知识点漫画第一阶段：项目、漫画格和漫画格历史版本
--
-- 核心原则：
--   1. 漫画图片和可编辑文字覆盖层分开保存；
--   2. 图片模型不得成为中文文字的事实源；
--   3. 气泡、旁白、题目、选项、答案和解析统一存入覆盖层JSON；
--   4. 覆盖层使用TE-DNA稳定协议，不直接保存任何第三方编辑器私有格式；
--   5. 每次单格生图保存不可变历史版本，支持后续恢复；
--   6. 教材、单元和知识点在项目创建时固化快照；
--   7. 资产、助手、教材单元和插入页均按软关联处理，运行时重新校验。
--
-- 第一版业务层只开放K12，但数据库预留具体教学域值，
-- 不允许mixed或common写入漫画项目资源快照。
-- ============================================================================

-- ============================================================================
-- 一、漫画项目
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_comic_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    courseware_id UUID NOT NULL
        REFERENCES coursewares(id)
        ON DELETE CASCADE,

    created_by UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    -- 创建时固化的资源教育域。
    education_domain VARCHAR(20) NOT NULL,

    -- 项目基本信息。
    title VARCHAR(200) NOT NULL,
    subject VARCHAR(100) NOT NULL,
    grade VARCHAR(100) NOT NULL,

    -- K12教材版本与单元快照。
    publisher_snapshot VARCHAR(200) NOT NULL DEFAULT '',
    semester_snapshot VARCHAR(32) NOT NULL DEFAULT '',
    textbook_unit_id UUID,
    textbook_unit_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 课标知识点快照。
    knowledge_points_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    knowledge_content_snapshot TEXT NOT NULL DEFAULT '',
    teacher_focus TEXT NOT NULL DEFAULT '',

    -- 漫画助手采用软关联，助手归档或停用后运行时重新校验。
    assistant_id UUID,

    -- 叙事方式和视觉风格使用可扩展业务代码，不在数据库写死枚举。
    narrative_mode VARCHAR(100) NOT NULL DEFAULT 'knowledge_story',
    visual_style VARCHAR(100) NOT NULL DEFAULT 'science_encyclopedia',

    panel_count SMALLINT NOT NULL DEFAULT 4,
    layout_mode VARCHAR(32) NOT NULL DEFAULT 'grid',

    -- 最终页面排版和互动策略。
    page_layout_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    interaction_config_json JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- 项目级人物与视觉连续性事实源。
    style_aoci_text TEXT NOT NULL DEFAULT '',
    character_bible_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    continuity_ledger_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    character_sheet_asset_id UUID,

    status VARCHAR(32) NOT NULL DEFAULT 'draft',

    -- 插入课件后的稳定页ID与历史页码快照。
    inserted_page_id UUID,
    inserted_page_number_snapshot INTEGER NOT NULL DEFAULT 0,

    version INTEGER NOT NULL DEFAULT 1,
    last_error TEXT NOT NULL DEFAULT '',

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT courseware_comic_projects_domain_chk
        CHECK (
            education_domain IN (
                'k12',
                'vocational',
                'adult'
            )
        ),

    CONSTRAINT courseware_comic_projects_panel_count_chk
        CHECK (panel_count BETWEEN 4 AND 8),

    CONSTRAINT courseware_comic_projects_layout_mode_chk
        CHECK (
            layout_mode IN (
                'grid',
                'spotlight',
                'carousel'
            )
        ),

    CONSTRAINT courseware_comic_projects_status_chk
        CHECK (
            status IN (
                'draft',
                'planning',
                'planned',
                'generating',
                'ready',
                'inserted',
                'failed',
                'archived'
            )
        ),

    CONSTRAINT courseware_comic_projects_unit_snapshot_chk
        CHECK (
            jsonb_typeof(textbook_unit_snapshot) = 'object'
        ),

    CONSTRAINT courseware_comic_projects_knowledge_points_chk
        CHECK (
            jsonb_typeof(knowledge_points_json) = 'array'
        ),

    CONSTRAINT courseware_comic_projects_page_layout_chk
        CHECK (
            jsonb_typeof(page_layout_json) = 'object'
        ),

    CONSTRAINT courseware_comic_projects_interaction_chk
        CHECK (
            jsonb_typeof(interaction_config_json) = 'object'
        ),

    CONSTRAINT courseware_comic_projects_character_bible_chk
        CHECK (
            jsonb_typeof(character_bible_json) = 'object'
        ),

    CONSTRAINT courseware_comic_projects_continuity_chk
        CHECK (
            jsonb_typeof(continuity_ledger_json) = 'object'
        ),

    CONSTRAINT courseware_comic_projects_page_snapshot_chk
        CHECK (inserted_page_number_snapshot >= 0),

    CONSTRAINT courseware_comic_projects_version_chk
        CHECK (version >= 1)
);

CREATE INDEX IF NOT EXISTS idx_courseware_comic_projects_courseware
    ON courseware_comic_projects (
        courseware_id,
        updated_at DESC
    );

CREATE INDEX IF NOT EXISTS idx_courseware_comic_projects_owner
    ON courseware_comic_projects (
        created_by,
        updated_at DESC
    );

CREATE INDEX IF NOT EXISTS idx_courseware_comic_projects_status
    ON courseware_comic_projects (
        courseware_id,
        status,
        updated_at DESC
    );

COMMENT ON TABLE courseware_comic_projects IS
    '知识点漫画项目：固化教材知识、漫画助手规划、人物设定和最终页面排版';

COMMENT ON COLUMN courseware_comic_projects.textbook_unit_snapshot IS
    '教材单元创建时快照，采用TE-DNA稳定JSON协议，不依赖第三方编辑器格式';

COMMENT ON COLUMN courseware_comic_projects.knowledge_points_json IS
    '课标知识点完整快照数组，知识库更新后不静默改变已有漫画';

COMMENT ON COLUMN courseware_comic_projects.page_layout_json IS
    '4至8格漫画最终课件页面布局设置';

COMMENT ON COLUMN courseware_comic_projects.interaction_config_json IS
    '题目、答案揭晓、逐格播放等HTML互动设置';

COMMENT ON COLUMN courseware_comic_projects.character_bible_json IS
    '人物固定特征、服装、颜色、角色关系和禁止漂移规则';

-- ============================================================================
-- 二、漫画格
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_comic_panels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    project_id UUID NOT NULL
        REFERENCES courseware_comic_projects(id)
        ON DELETE CASCADE,

    panel_no SMALLINT NOT NULL,
    image_key VARCHAR(15) NOT NULL,

    -- 本格故事和知识职责。
    story_purpose TEXT NOT NULL DEFAULT '',
    knowledge_claim TEXT NOT NULL DEFAULT '',
    scene_text TEXT NOT NULL DEFAULT '',
    character_ids_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    action_text TEXT NOT NULL DEFAULT '',
    camera_text TEXT NOT NULL DEFAULT '',
    narration_text TEXT NOT NULL DEFAULT '',
    dialogues_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    knowledge_presentation TEXT NOT NULL DEFAULT '',

    -- 单格图片生成事实源。
    visual_prompt TEXT NOT NULL DEFAULT '',
    negative_prompt TEXT NOT NULL DEFAULT '',
    aoci_text TEXT NOT NULL DEFAULT '',
    relations_json JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- 自动排好后供教师调整的文字与气泡覆盖文档。
    overlay_document_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    overlay_version INTEGER NOT NULL DEFAULT 1,

    status VARCHAR(32) NOT NULL DEFAULT 'planned',

    -- 资产使用软关联，图片删除后运行时重新校验。
    current_asset_id UUID,

    version INTEGER NOT NULL DEFAULT 1,
    last_error TEXT NOT NULL DEFAULT '',

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT courseware_comic_panels_panel_no_chk
        CHECK (panel_no BETWEEN 1 AND 8),

    CONSTRAINT courseware_comic_panels_image_key_chk
        CHECK (
            image_key ~ '^@I-[0-9A-F]{12}$'
        ),

    CONSTRAINT courseware_comic_panels_character_ids_chk
        CHECK (
            jsonb_typeof(character_ids_json) = 'array'
        ),

    CONSTRAINT courseware_comic_panels_dialogues_chk
        CHECK (
            jsonb_typeof(dialogues_json) = 'array'
        ),

    CONSTRAINT courseware_comic_panels_relations_chk
        CHECK (
            jsonb_typeof(relations_json) = 'array'
        ),

    CONSTRAINT courseware_comic_panels_overlay_document_chk
        CHECK (
            jsonb_typeof(overlay_document_json) = 'object'
        ),

    CONSTRAINT courseware_comic_panels_overlay_version_chk
        CHECK (overlay_version >= 1),

    CONSTRAINT courseware_comic_panels_status_chk
        CHECK (
            status IN (
                'planned',
                'generating',
                'generated',
                'failed',
                'stale'
            )
        ),

    CONSTRAINT courseware_comic_panels_version_chk
        CHECK (version >= 1),

    CONSTRAINT courseware_comic_panels_project_no_uq
        UNIQUE (
            project_id,
            panel_no
        ),

    CONSTRAINT courseware_comic_panels_project_key_uq
        UNIQUE (
            project_id,
            image_key
        )
);

CREATE INDEX IF NOT EXISTS idx_courseware_comic_panels_project
    ON courseware_comic_panels (
        project_id,
        panel_no
    );

CREATE INDEX IF NOT EXISTS idx_courseware_comic_panels_status
    ON courseware_comic_panels (
        project_id,
        status,
        panel_no
    );

CREATE INDEX IF NOT EXISTS idx_courseware_comic_panels_asset
    ON courseware_comic_panels (
        current_asset_id
    )
    WHERE current_asset_id IS NOT NULL;

COMMENT ON TABLE courseware_comic_panels IS
    '知识点漫画分格：保存图片IAOCI、连续叙事及可编辑HTML/SVG覆盖层';

COMMENT ON COLUMN courseware_comic_panels.overlay_document_json IS
    '自动排版后的气泡、旁白、知识卡、题目、选项、答案和解析文档';

COMMENT ON COLUMN courseware_comic_panels.overlay_version IS
    '教师每次确认保存覆盖层时递增，不随拖动中的临时草稿逐像素写库';

-- ============================================================================
-- 三、漫画格不可变历史版本
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_comic_panel_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    panel_id UUID NOT NULL
        REFERENCES courseware_comic_panels(id)
        ON DELETE CASCADE,

    version_no INTEGER NOT NULL,

    prompt_snapshot TEXT NOT NULL DEFAULT '',
    aoci_snapshot TEXT NOT NULL DEFAULT '',
    overlay_document_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,

    asset_id UUID,

    generation_source VARCHAR(32) NOT NULL DEFAULT 'initial',

    created_by UUID NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT courseware_comic_panel_versions_number_chk
        CHECK (version_no >= 1),

    CONSTRAINT courseware_comic_panel_versions_overlay_chk
        CHECK (
            jsonb_typeof(overlay_document_snapshot) = 'object'
        ),

    CONSTRAINT courseware_comic_panel_versions_source_chk
        CHECK (
            generation_source IN (
                'initial',
                'regenerate',
                'restore',
                'manual_save'
            )
        ),

    CONSTRAINT courseware_comic_panel_versions_panel_no_uq
        UNIQUE (
            panel_id,
            version_no
        )
);

CREATE INDEX IF NOT EXISTS idx_courseware_comic_panel_versions_panel
    ON courseware_comic_panel_versions (
        panel_id,
        version_no DESC
    );

CREATE INDEX IF NOT EXISTS idx_courseware_comic_panel_versions_asset
    ON courseware_comic_panel_versions (
        asset_id
    )
    WHERE asset_id IS NOT NULL;

COMMENT ON TABLE courseware_comic_panel_versions IS
    '漫画格不可变历史：保存每次生图或人工确认保存时的图片和覆盖层快照';

DO $$
BEGIN
    RAISE NOTICE
        'migration_v20260727_courseware_comics执行完成：漫画项目、分格和版本表已准备';
END $$;
