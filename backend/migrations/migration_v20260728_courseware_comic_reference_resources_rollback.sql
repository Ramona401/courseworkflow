-- ============================================================================
-- migration_v20260728_courseware_comic_reference_resources_rollback.sql
-- 知识点漫画参考资源迁移回滚
--
-- 警告：
--   1. 本文件会删除全部知识点漫画参考资源记录；
--   2. 不会删除courseware_assets中的图片物理文件或资产记录；
--   3. 执行前必须再次独立备份数据库；
--   4. PUBLIC是权限伪角色，不需要作为真实数据库角色处理。
-- ============================================================================

BEGIN;

-- 删除表会同时删除表级权限配置和依赖该表的业务索引。
-- 不使用CASCADE，避免意外删除未列入本迁移范围的其他对象。
DROP TABLE IF EXISTS
    public.courseware_comic_reference_resources;

-- 以下两个唯一索引仅用于本功能的复合外键边界。
-- 必须在参考资源表删除后才能安全删除。
DROP INDEX IF EXISTS
    public.ux_courseware_assets_id_courseware;

DROP INDEX IF EXISTS
    public.ux_courseware_comic_projects_id_courseware_owner;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '知识点漫画参考资源迁移已回滚';
END
$$;
