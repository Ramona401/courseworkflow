-- ============================================================================
-- TE-DNA 2.0：课件原生教学智能体核心表权限修复
-- 文件：20260725_06_courseware_assistant_core_privileges.sql
-- ----------------------------------------------------------------------------
-- 背景：
--
-- 数据库默认权限可能在CREATE TABLE时自动向tedna_user授予全部表权限。
-- 后续执行精简GRANT不会撤销已有的UPDATE、DELETE、TRUNCATE、
-- REFERENCES和TRIGGER权限。
--
-- 本迁移显式撤销PUBLIC和tedna_user在三张核心表上的全部权限，
-- 再按最小权限原则重新授权：
--
--   courseware_assistant_slots：
--       SELECT、INSERT、UPDATE、DELETE
--
--   assistant_deployments：
--       SELECT、INSERT、UPDATE、DELETE
--
--   assistant_deployment_versions：
--       SELECT、INSERT
--
-- 发布版本表不向应用角色开放UPDATE、DELETE或TRUNCATE。
-- 删除部署时，数据库外键ON DELETE CASCADE仍可内部清理版本记录，
-- 不要求调用用户拥有版本表DELETE权限。
--
-- 本迁移不修改表结构、不修改业务数据。
-- ============================================================================

BEGIN;

-- ============================================================================
-- 一、撤销PUBLIC继承的表权限
-- ============================================================================

REVOKE ALL PRIVILEGES
ON TABLE
    courseware_assistant_slots,
    assistant_deployments,
    assistant_deployment_versions
FROM PUBLIC;

-- ============================================================================
-- 二、撤销应用角色已有的全部显式表权限
-- ============================================================================

REVOKE ALL PRIVILEGES
ON TABLE
    courseware_assistant_slots,
    assistant_deployments,
    assistant_deployment_versions
FROM tedna_user;

-- ============================================================================
-- 三、按最小权限重新授权编辑态表
-- ============================================================================

GRANT
    SELECT,
    INSERT,
    UPDATE,
    DELETE
ON TABLE
    courseware_assistant_slots,
    assistant_deployments
TO tedna_user;

-- ============================================================================
-- 四、发布版本表只允许读取和追加
-- ============================================================================

GRANT
    SELECT,
    INSERT
ON TABLE
    assistant_deployment_versions
TO tedna_user;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件教学智能体核心表最小权限修复完成';
END
$$;
