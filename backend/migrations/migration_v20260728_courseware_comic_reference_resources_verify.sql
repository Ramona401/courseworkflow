-- ============================================================================
-- migration_v20260728_courseware_comic_reference_resources_verify.sql
-- 知识点漫画参考资源迁移只读验证
--
-- 用途：
--   正向迁移执行完成后，通过psql运行本文件。
--   本文件只读取系统目录和新业务表，不修改任何业务数据。
--
-- 权限验证说明：
--   PUBLIC是PostgreSQL权限伪角色，不存在于pg_roles；
--   因此不能调用has_table_privilege('PUBLIC', ...)。
--   PUBLIC显式授权改用information_schema.table_privileges检查。
-- ============================================================================

BEGIN TRANSACTION READ ONLY;

DO $$
DECLARE
    missing_columns INTEGER;
    missing_constraints INTEGER;
BEGIN
    IF to_regclass(
        'public.courseware_comic_reference_resources'
    ) IS NULL THEN
        RAISE EXCEPTION
            '参考资源表不存在';
    END IF;

    SELECT count(*)
    INTO missing_columns
    FROM (
        VALUES
            ('id'),
            ('project_id'),
            ('courseware_id'),
            ('created_by'),
            ('resource_type'),
            ('source_id'),
            ('asset_id'),
            ('title'),
            ('file_name'),
            ('mime_type'),
            ('content_text'),
            ('summary_text'),
            ('original_length'),
            ('summary_length'),
            ('sort_order'),
            ('created_at'),
            ('updated_at')
    ) AS expected(column_name)
    WHERE NOT EXISTS (
        SELECT 1
        FROM information_schema.columns actual
        WHERE actual.table_schema = 'public'
          AND actual.table_name =
              'courseware_comic_reference_resources'
          AND actual.column_name =
              expected.column_name
    );

    IF missing_columns <> 0 THEN
        RAISE EXCEPTION
            '参考资源表缺少%个预期字段',
            missing_columns;
    END IF;

    SELECT count(*)
    INTO missing_constraints
    FROM (
        VALUES
            ('fk_cw_comic_reference_project_boundary'),
            ('fk_cw_comic_reference_image_boundary'),
            ('ck_cw_comic_reference_type'),
            ('ck_cw_comic_reference_title'),
            ('ck_cw_comic_reference_file_name'),
            ('ck_cw_comic_reference_mime_type'),
            ('ck_cw_comic_reference_lengths'),
            ('ck_cw_comic_reference_sort'),
            ('ck_cw_comic_reference_source'),
            ('ck_cw_comic_reference_asset'),
            ('ck_cw_comic_reference_uploaded_file'),
            ('ck_cw_comic_reference_image_mime'),
            ('ck_cw_comic_reference_content')
    ) AS expected(constraint_name)
    WHERE NOT EXISTS (
        SELECT 1
        FROM pg_constraint actual
        WHERE actual.conname =
            expected.constraint_name
          AND actual.conrelid =
              'public.courseware_comic_reference_resources'::regclass
    );

    IF missing_constraints <> 0 THEN
        RAISE EXCEPTION
            '参考资源表缺少%个预期约束',
            missing_constraints;
    END IF;

    IF to_regclass(
        'public.ux_courseware_comic_projects_id_courseware_owner'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少漫画项目复合边界唯一索引';
    END IF;

    IF to_regclass(
        'public.ux_courseware_assets_id_courseware'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少课件资产复合边界唯一索引';
    END IF;

    IF to_regclass(
        'public.ux_cw_comic_reference_source'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少正式来源去重索引';
    END IF;

    IF to_regclass(
        'public.ux_cw_comic_reference_asset'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少参考图片去重索引';
    END IF;

    IF to_regclass(
        'public.ix_cw_comic_reference_project_order'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少项目资源顺序索引';
    END IF;

    IF to_regclass(
        'public.ix_cw_comic_reference_courseware'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少课件资源查询索引';
    END IF;

    IF to_regclass(
        'public.ix_cw_comic_reference_creator'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少创建者资源查询索引';
    END IF;

    IF to_regclass(
        'public.ix_cw_comic_reference_type'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少参考资源类型索引';
    END IF;

    IF NOT has_table_privilege(
        'tedna_user',
        'public.courseware_comic_reference_resources',
        'SELECT'
    ) OR NOT has_table_privilege(
        'tedna_user',
        'public.courseware_comic_reference_resources',
        'INSERT'
    ) OR NOT has_table_privilege(
        'tedna_user',
        'public.courseware_comic_reference_resources',
        'UPDATE'
    ) OR NOT has_table_privilege(
        'tedna_user',
        'public.courseware_comic_reference_resources',
        'DELETE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少参考资源表所需CRUD权限';
    END IF;

    IF has_table_privilege(
        'tedna_user',
        'public.courseware_comic_reference_resources',
        'TRUNCATE'
    ) OR has_table_privilege(
        'tedna_user',
        'public.courseware_comic_reference_resources',
        'REFERENCES'
    ) OR has_table_privilege(
        'tedna_user',
        'public.courseware_comic_reference_resources',
        'TRIGGER'
    ) THEN
        RAISE EXCEPTION
            'tedna_user拥有超出CRUD范围的参考资源表权限';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.table_privileges privilege
        WHERE privilege.table_schema = 'public'
          AND privilege.table_name =
              'courseware_comic_reference_resources'
          AND privilege.grantee = 'PUBLIC'
    ) THEN
        RAISE EXCEPTION
            'PUBLIC不应拥有参考资源表权限';
    END IF;

    RAISE NOTICE
        '知识点漫画参考资源表字段、约束、索引和权限验证通过';
END
$$;

SELECT
    resource_type,
    count(*) AS resource_count
FROM public.courseware_comic_reference_resources
GROUP BY resource_type
ORDER BY resource_type;

SELECT
    count(*) AS total_reference_resources
FROM public.courseware_comic_reference_resources;

ROLLBACK;
