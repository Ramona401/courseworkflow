-- ============================================================================
-- migration_v20260728_courseware_comic_style_instruction.sql
-- 知识点漫画第三步：教师自然语言风格补充要求
--
-- visual_style保存系统稳定风格代码；
-- style_instruction保存教师可选的自然语言微调要求，例如：
--   - 颜色更明亮；
--   - 人物更像中学生；
--   - 减少写实感；
--   - 保持当前课件的简洁蓝紫色视觉语言。
--
-- 图片生成时由服务端将稳定风格代码、比例、清晰度和本字段共同转换为
-- 图片模型提示词。浏览器不得直接提交模型参数或第三方私有配置。
-- ============================================================================

BEGIN;

ALTER TABLE courseware_comic_projects
    ADD COLUMN IF NOT EXISTS style_instruction TEXT
        NOT NULL DEFAULT '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'courseware_comic_projects_style_instruction_length_chk'
    ) THEN
        ALTER TABLE courseware_comic_projects
            ADD CONSTRAINT
                courseware_comic_projects_style_instruction_length_chk
            CHECK (
                char_length(style_instruction) <= 2000
            );
    END IF;
END $$;

COMMENT ON COLUMN
    courseware_comic_projects.style_instruction IS
    '老师在第三步填写的可选自然语言画风补充要求，不是模型配置或授权字段';

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '知识点漫画style_instruction迁移完成';
END $$;
