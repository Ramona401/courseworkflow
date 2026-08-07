-- ============================================================================
-- migration_v20260728_courseware_comic_reference_resources.sql
-- 知识点漫画第一步：可选参考资源持久化
--
-- 目标：
--   1. 知识点仍是创建漫画项目的唯一必填内容；
--   2. 教材单元、已有课件、课程大纲、DOCX/PDF文字、参考图片和其他文字
--      均作为可选增强来源；
--   3. 文档只保存浏览器提取出的文字和可选摘要，不保存原始二进制文件；
--   4. 图片复用courseware_assets，并使用page_id=NULL的项目级图片资产；
--   5. 绑定正式资源时保存创建时文本快照，不依赖源资源永久存在；
--   6. 项目、课件、创建者和图片资产边界由复合外键约束；
--   7. AI上下文容量限制由服务层执行，本表只保存稳定事实源；
--   8. PUBLIC权限通过系统权限视图验证，不把PUBLIC当作数据库角色。
-- ============================================================================

BEGIN;

-- 供参考资源表复合外键校验项目、课件和创建者三重边界。
CREATE UNIQUE INDEX IF NOT EXISTS
    ux_courseware_comic_projects_id_courseware_owner
ON public.courseware_comic_projects (
    id,
    courseware_id,
    created_by
);

-- 供参考图片复合外键校验资产真实属于同一个课件。
CREATE UNIQUE INDEX IF NOT EXISTS
    ux_courseware_assets_id_courseware
ON public.courseware_assets (
    id,
    courseware_id
);

CREATE TABLE IF NOT EXISTS
    public.courseware_comic_reference_resources (
        id UUID PRIMARY KEY
            DEFAULT gen_random_uuid(),

        -- 漫画项目及可信资源边界。
        project_id UUID NOT NULL,
        courseware_id UUID NOT NULL,
        created_by UUID NOT NULL,

        -- 资源类型：
        --   textbook_unit     教材单元快照；
        --   courseware        已有课件内容摘要快照；
        --   course_outline    课程大纲正文快照；
        --   uploaded_document DOCX或文字型PDF提取文本；
        --   uploaded_image    课件级参考图片资产；
        --   other_text        教师粘贴的其他文字资料。
        resource_type VARCHAR(32) NOT NULL,

        -- 多态源资源ID。
        --
        -- textbook_unit、courseware和course_outline必须填写；
        -- uploaded_document、uploaded_image和other_text必须为空。
        -- 真实资源归属、教育域和可见性由服务层在创建时重新校验。
        source_id UUID,

        -- 仅uploaded_image使用。
        -- 复合外键保证图片资产属于同一个courseware_id。
        asset_id UUID,

        -- 创建时固化的浏览器展示标题。
        title VARCHAR(500) NOT NULL,

        -- 上传文档或图片的原始文件名。
        file_name VARCHAR(255) NOT NULL
            DEFAULT '',

        -- 上传文档或图片的MIME类型。
        mime_type VARCHAR(255) NOT NULL
            DEFAULT '',

        -- 创建时固化的正文快照。
        --
        -- uploaded_image不保存正文；
        -- 长资料可以同时保存正文和压缩摘要。
        content_text TEXT NOT NULL
            DEFAULT '',

        -- 长资料的结构化压缩摘要。
        -- AI装配时有摘要优先使用摘要，没有摘要时使用正文。
        summary_text TEXT NOT NULL
            DEFAULT '',

        original_length INTEGER NOT NULL
            DEFAULT 0,

        summary_length INTEGER NOT NULL
            DEFAULT 0,

        sort_order INTEGER NOT NULL
            DEFAULT 0,

        created_at TIMESTAMPTZ NOT NULL
            DEFAULT now(),

        updated_at TIMESTAMPTZ NOT NULL
            DEFAULT now(),

        -- 项目、课件和创建者必须与漫画项目记录完全一致。
        CONSTRAINT
            fk_cw_comic_reference_project_boundary
        FOREIGN KEY (
            project_id,
            courseware_id,
            created_by
        )
        REFERENCES
            public.courseware_comic_projects (
                id,
                courseware_id,
                created_by
            )
        ON DELETE CASCADE,

        -- 参考图片必须真实属于同一个课件。
        CONSTRAINT
            fk_cw_comic_reference_image_boundary
        FOREIGN KEY (
            asset_id,
            courseware_id
        )
        REFERENCES
            public.courseware_assets (
                id,
                courseware_id
            )
        ON DELETE CASCADE,

        CONSTRAINT
            ck_cw_comic_reference_type
        CHECK (
            resource_type IN (
                'textbook_unit',
                'courseware',
                'course_outline',
                'uploaded_document',
                'uploaded_image',
                'other_text'
            )
        ),

        CONSTRAINT
            ck_cw_comic_reference_title
        CHECK (
            char_length(
                btrim(title)
            ) BETWEEN 1 AND 500
        ),

        CONSTRAINT
            ck_cw_comic_reference_file_name
        CHECK (
            char_length(file_name) <= 255
        ),

        CONSTRAINT
            ck_cw_comic_reference_mime_type
        CHECK (
            char_length(mime_type) <= 255
        ),

        CONSTRAINT
            ck_cw_comic_reference_lengths
        CHECK (
            original_length >= 0
            AND summary_length >= 0
            AND char_length(content_text) <= 120000
            AND char_length(summary_text) <= 30000
        ),

        CONSTRAINT
            ck_cw_comic_reference_sort
        CHECK (
            sort_order >= 0
        ),

        -- 绑定正式资源时必须具有source_id；
        -- 浏览器上传或粘贴内容不得伪造source_id。
        CONSTRAINT
            ck_cw_comic_reference_source
        CHECK (
            (
                resource_type IN (
                    'textbook_unit',
                    'courseware',
                    'course_outline'
                )
                AND source_id IS NOT NULL
            )
            OR
            (
                resource_type IN (
                    'uploaded_document',
                    'uploaded_image',
                    'other_text'
                )
                AND source_id IS NULL
            )
        ),

        -- 只有参考图片允许绑定asset_id。
        CONSTRAINT
            ck_cw_comic_reference_asset
        CHECK (
            (
                resource_type = 'uploaded_image'
                AND asset_id IS NOT NULL
            )
            OR
            (
                resource_type <> 'uploaded_image'
                AND asset_id IS NULL
            )
        ),

        -- 文档和图片必须保留原始文件名与MIME类型。
        CONSTRAINT
            ck_cw_comic_reference_uploaded_file
        CHECK (
            resource_type NOT IN (
                'uploaded_document',
                'uploaded_image'
            )
            OR (
                char_length(
                    btrim(file_name)
                ) > 0
                AND char_length(
                    btrim(mime_type)
                ) > 0
            )
        ),

        -- 参考图片MIME必须属于图片类型。
        CONSTRAINT
            ck_cw_comic_reference_image_mime
        CHECK (
            resource_type <> 'uploaded_image'
            OR lower(
                btrim(mime_type)
            ) LIKE 'image/%'
        ),

        -- 除图片外，每条资料必须具有正文或摘要快照。
        CONSTRAINT
            ck_cw_comic_reference_content
        CHECK (
            resource_type = 'uploaded_image'
            OR char_length(
                btrim(content_text)
            ) > 0
            OR char_length(
                btrim(summary_text)
            ) > 0
        )
    );

-- 一个项目不能重复绑定同一个正式来源。
CREATE UNIQUE INDEX IF NOT EXISTS
    ux_cw_comic_reference_source
ON public.courseware_comic_reference_resources (
    project_id,
    resource_type,
    source_id
)
WHERE source_id IS NOT NULL;

-- 一个项目不能重复绑定同一张参考图片。
CREATE UNIQUE INDEX IF NOT EXISTS
    ux_cw_comic_reference_asset
ON public.courseware_comic_reference_resources (
    project_id,
    asset_id
)
WHERE asset_id IS NOT NULL;

-- 项目详情和AI上下文按教师顺序读取。
CREATE INDEX IF NOT EXISTS
    ix_cw_comic_reference_project_order
ON public.courseware_comic_reference_resources (
    project_id,
    sort_order,
    created_at,
    id
);

-- 课件删除、审计和项目级素材查询使用。
CREATE INDEX IF NOT EXISTS
    ix_cw_comic_reference_courseware
ON public.courseware_comic_reference_resources (
    courseware_id,
    created_at DESC
);

CREATE INDEX IF NOT EXISTS
    ix_cw_comic_reference_creator
ON public.courseware_comic_reference_resources (
    created_by,
    created_at DESC
);

CREATE INDEX IF NOT EXISTS
    ix_cw_comic_reference_type
ON public.courseware_comic_reference_resources (
    project_id,
    resource_type,
    sort_order
);

COMMENT ON TABLE
    public.courseware_comic_reference_resources IS
    '知识点漫画可选参考资源：保存可信资源快照、浏览器提取文档文字和同课件参考图片绑定';

COMMENT ON COLUMN
    public.courseware_comic_reference_resources.source_id IS
    '教材单元、已有课件或课程大纲的原始资源ID；正文事实使用创建时快照';

COMMENT ON COLUMN
    public.courseware_comic_reference_resources.asset_id IS
    'uploaded_image对应的同课件图片资产ID；删除资产时级联删除参考绑定';

COMMENT ON COLUMN
    public.courseware_comic_reference_resources.content_text IS
    '正式资源正文快照、浏览器提取的DOCX/PDF文字或教师粘贴文字';

COMMENT ON COLUMN
    public.courseware_comic_reference_resources.summary_text IS
    '长资料压缩摘要；AI上下文有摘要时优先使用摘要';

ALTER TABLE
    public.courseware_comic_reference_resources
OWNER TO postgres;

REVOKE ALL PRIVILEGES
ON TABLE
    public.courseware_comic_reference_resources
FROM PUBLIC;

REVOKE ALL PRIVILEGES
ON TABLE
    public.courseware_comic_reference_resources
FROM tedna_user;

GRANT
    SELECT,
    INSERT,
    UPDATE,
    DELETE
ON TABLE
    public.courseware_comic_reference_resources
TO tedna_user;

-- PUBLIC是PostgreSQL权限伪角色，不是pg_roles中的真实角色。
-- 因此不能调用has_table_privilege('PUBLIC', ...)。
-- 这里通过information_schema.table_privileges检查PUBLIC是否仍有显式授权。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.table_privileges privilege
        WHERE privilege.table_schema = 'public'
          AND privilege.table_name =
              'courseware_comic_reference_resources'
          AND privilege.grantee = 'PUBLIC'
    ) THEN
        RAISE EXCEPTION
            'PUBLIC仍拥有知识点漫画参考资源表权限';
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
            'tedna_user缺少知识点漫画参考资源表所需CRUD权限';
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
END
$$;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '知识点漫画参考资源表迁移完成';
END
$$;
