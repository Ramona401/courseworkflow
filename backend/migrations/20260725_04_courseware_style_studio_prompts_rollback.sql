-- ============================================================================
-- TE-DNA 2.0：AI美术风格工作室提示词回滚
-- 文件：20260725_04_courseware_style_studio_prompts_rollback.sql
-- ============================================================================

BEGIN;

DELETE FROM prompts
WHERE prompt_key IN (
    'prompt_courseware_style_studio_chat',
    'prompt_courseware_style_studio_image'
);

COMMIT;
