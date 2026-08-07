-- ============================================================================
-- migration_v20260727_courseware_comic_workflow.sql
-- 知识点漫画五步任务式工作流
--
-- 五个教师视角步骤：
--   1. source            输入知识点与可选教学来源
--   2. storyboard        确认叙事方式、分镜、对白和知识呈现
--   3. style_preview     选择美术风格、画面比例、清晰度并确认首格样张
--   4. batch_generation  自动生成其余漫画格
--   5. refinement        微调画面、文字、气泡并选择使用方式
--
-- workflow_stage只描述教师当前任务，不替代原有status生产状态。
-- status继续描述AI规划、生图、完成、插页和失败等后台事实。
-- ============================================================================

BEGIN;

ALTER TABLE courseware_comic_projects
    ADD COLUMN IF NOT EXISTS workflow_stage VARCHAR(32)
        NOT NULL DEFAULT 'source';

ALTER TABLE courseware_comic_projects
    ADD COLUMN IF NOT EXISTS storyboard_confirmed_at TIMESTAMPTZ;

ALTER TABLE courseware_comic_projects
    ADD COLUMN IF NOT EXISTS style_confirmed_at TIMESTAMPTZ;

ALTER TABLE courseware_comic_projects
    ADD COLUMN IF NOT EXISTS style_preview_panel_id UUID;

ALTER TABLE courseware_comic_projects
    ADD COLUMN IF NOT EXISTS aspect_ratio VARCHAR(16)
        NOT NULL DEFAULT 'courseware';

ALTER TABLE courseware_comic_projects
    ADD COLUMN IF NOT EXISTS image_quality VARCHAR(16)
        NOT NULL DEFAULT 'high';

ALTER TABLE courseware_comic_projects
    ADD COLUMN IF NOT EXISTS insertion_mode VARCHAR(32)
        NOT NULL DEFAULT 'single_page';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'courseware_comic_projects_workflow_stage_chk'
    ) THEN
        ALTER TABLE courseware_comic_projects
            ADD CONSTRAINT
                courseware_comic_projects_workflow_stage_chk
            CHECK (
                workflow_stage IN (
                    'source',
                    'storyboard',
                    'style_preview',
                    'batch_generation',
                    'refinement'
                )
            );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'courseware_comic_projects_aspect_ratio_chk'
    ) THEN
        ALTER TABLE courseware_comic_projects
            ADD CONSTRAINT
                courseware_comic_projects_aspect_ratio_chk
            CHECK (
                aspect_ratio IN (
                    'courseware',
                    '16:9',
                    '4:3',
                    '1:1',
                    '3:4',
                    '9:16'
                )
            );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'courseware_comic_projects_image_quality_chk'
    ) THEN
        ALTER TABLE courseware_comic_projects
            ADD CONSTRAINT
                courseware_comic_projects_image_quality_chk
            CHECK (
                image_quality IN (
                    'standard',
                    'high'
                )
            );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'courseware_comic_projects_insertion_mode_chk'
    ) THEN
        ALTER TABLE courseware_comic_projects
            ADD CONSTRAINT
                courseware_comic_projects_insertion_mode_chk
            CHECK (
                insertion_mode IN (
                    'single_page',
                    'smart_pages',
                    'one_panel_per_page',
                    'library_only'
                )
            );
    END IF;
END $$;

-- 旧项目按现有生产事实映射到合理教师步骤。
UPDATE courseware_comic_projects project
SET workflow_stage =
    CASE
        WHEN project.status IN (
            'ready',
            'inserted'
        ) THEN
            'refinement'

        WHEN project.status =
            'generating' THEN
            'batch_generation'

        WHEN project.status =
            'planned' THEN
            'storyboard'

        WHEN project.status =
            'failed'
            AND EXISTS (
                SELECT 1
                FROM courseware_comic_panels panel
                WHERE panel.project_id =
                    project.id
                  AND (
                      panel.current_asset_id IS NOT NULL
                      OR panel.status IN (
                          'generating',
                          'generated',
                          'failed',
                          'stale'
                      )
                  )
            ) THEN
            'batch_generation'

        WHEN project.status =
            'failed'
            AND EXISTS (
                SELECT 1
                FROM courseware_comic_panels panel
                WHERE panel.project_id =
                    project.id
            ) THEN
            'storyboard'

        ELSE
            'source'
    END;

-- 已经进入过图片生产的旧项目视为已确认分镜和风格，
-- 避免升级后要求老师重新完成历史步骤。
UPDATE courseware_comic_projects project
SET storyboard_confirmed_at =
        COALESCE(
            storyboard_confirmed_at,
            updated_at,
            created_at
        ),
    style_confirmed_at =
        COALESCE(
            style_confirmed_at,
            updated_at,
            created_at
        )
WHERE project.status IN (
        'generating',
        'ready',
        'inserted'
    )
   OR (
        project.status = 'failed'
        AND EXISTS (
            SELECT 1
            FROM courseware_comic_panels panel
            WHERE panel.project_id =
                project.id
              AND (
                  panel.current_asset_id IS NOT NULL
                  OR panel.status IN (
                      'generating',
                      'generated',
                      'failed',
                      'stale'
                  )
              )
        )
   );

-- 旧项目已有第一格图片时，将其作为可恢复的样张定位。
UPDATE courseware_comic_projects project
SET style_preview_panel_id =
    (
        SELECT panel.id
        FROM courseware_comic_panels panel
        WHERE panel.project_id =
            project.id
          AND panel.panel_no = 1
          AND panel.current_asset_id IS NOT NULL
        LIMIT 1
    )
WHERE style_preview_panel_id IS NULL
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_panels panel
      WHERE panel.project_id =
          project.id
        AND panel.panel_no = 1
        AND panel.current_asset_id IS NOT NULL
  );

CREATE INDEX IF NOT EXISTS
    idx_courseware_comic_projects_workflow
ON courseware_comic_projects (
    courseware_id,
    workflow_stage,
    updated_at DESC
);

CREATE INDEX IF NOT EXISTS
    idx_courseware_comic_projects_style_preview
ON courseware_comic_projects (
    style_preview_panel_id
)
WHERE style_preview_panel_id IS NOT NULL;

COMMENT ON COLUMN
    courseware_comic_projects.workflow_stage IS
    '教师视角五步任务状态，独立于后台status生产状态';

COMMENT ON COLUMN
    courseware_comic_projects.storyboard_confirmed_at IS
    '老师最后一次确认叙事方式、分镜、对白和知识呈现的时间';

COMMENT ON COLUMN
    courseware_comic_projects.style_confirmed_at IS
    '老师最后一次确认首格完整画风样张的时间';

COMMENT ON COLUMN
    courseware_comic_projects.style_preview_panel_id IS
    '当前被老师确认或等待确认的首格样张漫画格ID，采用软关联';

COMMENT ON COLUMN
    courseware_comic_projects.aspect_ratio IS
    '图片比例：跟随课件或显式16:9、4:3、1:1、3:4、9:16';

COMMENT ON COLUMN
    courseware_comic_projects.image_quality IS
    '图片清晰度档位：standard或high';

COMMENT ON COLUMN
    courseware_comic_projects.insertion_mode IS
    '使用方式：单页完整漫画、智能分页、每格一页或仅保存素材库';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '知识点漫画五步工作流迁移完成';
END $$;
