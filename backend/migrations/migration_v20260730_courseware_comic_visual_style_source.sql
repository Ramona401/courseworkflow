-- ============================================================================
-- migration_v20260730_courseware_comic_visual_style_source.sql
-- 知识点漫画第三步：画风来源严格二选一
--
-- visual_style_source只允许：
--   courseware  跟随课件整体风格锚点；
--   selected    只使用老师选择的漫画画风。
--
-- 迁移策略：
--   1. 历史项目回填为courseware，保持原有生成行为；
--   2. 迁移完成后的新项目默认selected，真正尊重第三步画风选择；
--   3. 数据库约束禁止空值和第三种“混合”模式；
--   4. 本迁移不删除旧图片、人物设定图或样张资产。
-- ============================================================================

BEGIN;

ALTER TABLE courseware_comic_projects
    ADD COLUMN IF NOT EXISTS visual_style_source VARCHAR(16);

-- 只有历史空值或异常值会被回填。
-- 已经明确保存为courseware或selected的项目不会在重复执行时被覆盖。
UPDATE courseware_comic_projects
SET visual_style_source = 'courseware'
WHERE visual_style_source IS NULL
   OR btrim(visual_style_source) = ''
   OR visual_style_source NOT IN (
       'courseware',
       'selected'
   );

-- 新建项目从迁移完成后默认使用老师选择的漫画画风。
ALTER TABLE courseware_comic_projects
    ALTER COLUMN visual_style_source
        SET DEFAULT 'selected';

ALTER TABLE courseware_comic_projects
    ALTER COLUMN visual_style_source
        SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'courseware_comic_projects_visual_style_source_chk'
    ) THEN
        ALTER TABLE courseware_comic_projects
            ADD CONSTRAINT
                courseware_comic_projects_visual_style_source_chk
            CHECK (
                visual_style_source IN (
                    'courseware',
                    'selected'
                )
            );
    END IF;
END $$;

COMMENT ON COLUMN
    courseware_comic_projects.visual_style_source IS
    '漫画画风来源严格二选一：courseware仅跟随课件风格锚点；selected仅使用老师选择的漫画画风';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '知识点漫画visual_style_source严格二选一迁移完成';
END $$;
