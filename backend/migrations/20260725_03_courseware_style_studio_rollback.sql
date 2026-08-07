-- ============================================================================
-- TE-DNA 2.0：AI美术风格工作室数据库回滚
-- 文件：20260725_03_courseware_style_studio_rollback.sql
-- ============================================================================

BEGIN;

DROP TRIGGER IF EXISTS trg_validate_style_preview_asset
ON courseware_style_previews;

DROP TRIGGER IF EXISTS trg_validate_style_message_asset
ON courseware_style_messages;

DROP TRIGGER IF EXISTS trg_validate_style_session_assets
ON courseware_style_sessions;

DROP FUNCTION IF EXISTS tedna_validate_style_preview_asset();
DROP FUNCTION IF EXISTS tedna_validate_style_message_asset();
DROP FUNCTION IF EXISTS tedna_validate_style_session_assets();

DROP TABLE IF EXISTS courseware_style_previews;
DROP TABLE IF EXISTS courseware_style_messages;
DROP TABLE IF EXISTS courseware_style_sessions;

COMMIT;
