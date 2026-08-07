-- ============================================================================
-- TE-DNA 2.0：课件图片 IAOCI 索引与图片关系
-- 文件：20260725_01_courseware_image_aoci.sql
-- ----------------------------------------------------------------------------
-- 目标：
--   1. 每一个真实图片占位拥有独立、稳定的图片索引记录；
--   2. IAOCI 文本作为图片语义索引本体，不再以自由 JSON 作为事实源；
--   3. 图片之间通过独立关系表保存 R 关系；
--   4. 页面重排只改变 page_number，不改变 page_id、image_key 和图片关系；
--   5. 课程锚点、人物连续性、环境连续性和构图连续性分别控制；
--   6. 为后续多图独立生成、失败隔离和按关系选参考图提供数据库基础。
--
-- 本迁移不会删除或修改：
--   - coursewares.style_anchor_vaoci
--   - courseware_pages.image_suggestions
--   - courseware_pages.video_storyboards
--
-- 旧字段进入兼容期，后续业务链切换完成后再独立清理。
-- ============================================================================

BEGIN;

-- page_id 本身已经唯一；增加复合唯一索引，用于保证图片索引中的
-- (page_id, courseware_id) 确实指向同一课件下的页面。
CREATE UNIQUE INDEX IF NOT EXISTS ux_courseware_pages_id_courseware
ON courseware_pages (id, courseware_id);

CREATE TABLE IF NOT EXISTS courseware_image_indexes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 资源边界
    courseware_id uuid NOT NULL,
    page_id uuid,
    placeholder_id text NOT NULL,
    image_key varchar(32) NOT NULL,
    slot_order integer NOT NULL DEFAULT 1,

    -- IAOCI机器编码第一行
    index_version smallint NOT NULL DEFAULT 1,
    index_type varchar(1) NOT NULL,
    usage_role varchar(2) NOT NULL,
    continuity_level smallint NOT NULL DEFAULT 0,
    subject_type varchar(1) NOT NULL,
    aspect_ratio varchar(1) NOT NULL,
    relation_count varchar(1) NOT NULL DEFAULT '0',

    -- IAOCI可读语义标签
    focus_text text NOT NULL,
    layout_text text NOT NULL DEFAULT 'Ø',
    art_text text NOT NULL DEFAULT 'Ø',
    character_text text NOT NULL DEFAULT 'Ø',
    scene_text text NOT NULL DEFAULT 'Ø',
    export_text text NOT NULL DEFAULT 'Ø',
    negative_text text NOT NULL DEFAULT 'Ø',

    -- 完整规范化索引和生成结果
    aoci_text text NOT NULL,
    generation_prompt text NOT NULL DEFAULT '',
    asset_id uuid,
    status varchar(16) NOT NULL DEFAULT 'planned',
    last_error text NOT NULL DEFAULT '',
    version integer NOT NULL DEFAULT 1,

    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT fk_courseware_image_indexes_courseware
        FOREIGN KEY (courseware_id)
        REFERENCES coursewares(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_image_indexes_page
        FOREIGN KEY (page_id, courseware_id)
        REFERENCES courseware_pages(id, courseware_id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_image_indexes_asset
        FOREIGN KEY (asset_id)
        REFERENCES courseware_assets(id)
        ON DELETE SET NULL,

    -- 供关系表用复合外键验证同一课件边界
    CONSTRAINT uq_courseware_image_indexes_id_courseware
        UNIQUE (id, courseware_id),

    CONSTRAINT ck_courseware_image_indexes_placeholder
        CHECK (length(btrim(placeholder_id)) > 0),

    CONSTRAINT ck_courseware_image_indexes_key
        CHECK (image_key ~ '^(@ANCHOR|@I-[0-9A-F]{12})$'),

    CONSTRAINT ck_courseware_image_indexes_type
        CHECK (index_type IN ('A', 'I', 'V')),

    CONSTRAINT ck_courseware_image_indexes_usage
        CHECK (usage_role IN ('CV', 'KN', 'ST', 'EX', 'DG', 'BG')),

    CONSTRAINT ck_courseware_image_indexes_continuity
        CHECK (continuity_level BETWEEN 0 AND 3),

    CONSTRAINT ck_courseware_image_indexes_subject
        CHECK (subject_type IN ('N', 'P', 'A', 'O', 'M')),

    CONSTRAINT ck_courseware_image_indexes_aspect
        CHECK (aspect_ratio IN ('H', 'V', 'Q', 'F')),

    CONSTRAINT ck_courseware_image_indexes_relation_count
        CHECK (relation_count IN ('0', '1', 'M')),

    CONSTRAINT ck_courseware_image_indexes_status
        CHECK (
            status IN (
                'planned',
                'generating',
                'generated',
                'failed',
                'stale'
            )
        ),

    CONSTRAINT ck_courseware_image_indexes_version
        CHECK (index_version >= 1 AND version >= 1),

    CONSTRAINT ck_courseware_image_indexes_focus
        CHECK (length(btrim(focus_text)) > 0),

    CONSTRAINT ck_courseware_image_indexes_aoci
        CHECK (length(btrim(aoci_text)) > 0),

    -- 课程锚点使用固定 @ANCHOR，不绑定页面；
    -- 页面图片和视频首帧必须绑定稳定 page_id。
    CONSTRAINT ck_courseware_image_indexes_scope
        CHECK (
            (
                index_type = 'A'
                AND page_id IS NULL
                AND image_key = '@ANCHOR'
                AND slot_order = 0
            )
            OR
            (
                index_type IN ('I', 'V')
                AND page_id IS NOT NULL
                AND image_key LIKE '@I-%'
                AND slot_order >= 1
            )
        )
);

-- 同一课件下的稳定图片键不可重复。
CREATE UNIQUE INDEX IF NOT EXISTS ux_courseware_image_indexes_courseware_key
ON courseware_image_indexes (courseware_id, image_key);

-- 同一页面、同一图片类型、同一占位只能有一条当前索引。
CREATE UNIQUE INDEX IF NOT EXISTS ux_courseware_image_indexes_page_placeholder_type
ON courseware_image_indexes (page_id, placeholder_id, index_type)
WHERE page_id IS NOT NULL;

-- 每个课件最多一条课程级锚点索引。
CREATE UNIQUE INDEX IF NOT EXISTS ux_courseware_image_indexes_anchor
ON courseware_image_indexes (courseware_id)
WHERE index_type = 'A';

CREATE INDEX IF NOT EXISTS ix_courseware_image_indexes_page_order
ON courseware_image_indexes (page_id, slot_order);

CREATE INDEX IF NOT EXISTS ix_courseware_image_indexes_courseware_status
ON courseware_image_indexes (courseware_id, status);

CREATE INDEX IF NOT EXISTS ix_courseware_image_indexes_asset
ON courseware_image_indexes (asset_id)
WHERE asset_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS courseware_image_relations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    courseware_id uuid NOT NULL,
    source_image_index_id uuid NOT NULL,
    target_image_index_id uuid NOT NULL,

    -- 关系类型：
    -- >  时间、动作、故事或实验过程继续
    -- =  同一状态的另一视角
    -- ~  同主题平行图
    -- <> 正误、前后或条件对照
    -- ^  局部放大或派生细节
    relation_code varchar(2) NOT NULL,

    -- 可继承维度，按固定顺序组合：
    -- A 艺术风格
    -- C 人物、动物或固定主体
    -- S 环境场景
    -- O 物体及其状态
    -- L 构图与镜头
    inherit_mask varchar(5) NOT NULL,

    semantic_note text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT fk_courseware_image_relations_courseware
        FOREIGN KEY (courseware_id)
        REFERENCES coursewares(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_image_relations_source
        FOREIGN KEY (source_image_index_id, courseware_id)
        REFERENCES courseware_image_indexes(id, courseware_id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_image_relations_target
        FOREIGN KEY (target_image_index_id, courseware_id)
        REFERENCES courseware_image_indexes(id, courseware_id)
        ON DELETE CASCADE,

    CONSTRAINT ck_courseware_image_relations_not_self
        CHECK (source_image_index_id <> target_image_index_id),

    CONSTRAINT ck_courseware_image_relations_code
        CHECK (relation_code IN ('>', '=', '~', '<>', '^')),

    CONSTRAINT ck_courseware_image_relations_mask
        CHECK (
            length(inherit_mask) BETWEEN 1 AND 5
            AND inherit_mask ~ '^[ACSOL]+$'
        ),

    CONSTRAINT uq_courseware_image_relations_edge
        UNIQUE (
            source_image_index_id,
            target_image_index_id,
            relation_code
        )
);

CREATE INDEX IF NOT EXISTS ix_courseware_image_relations_source
ON courseware_image_relations (source_image_index_id);

CREATE INDEX IF NOT EXISTS ix_courseware_image_relations_target
ON courseware_image_relations (target_image_index_id);

COMMENT ON TABLE courseware_image_indexes IS
'课件图片IAOCI索引：每个真实图片占位一条独立记录，AOCI文本为语义事实源';

COMMENT ON COLUMN courseware_image_indexes.image_key IS
'稳定图片键：页面图片为@I-加12位确定性哈希，课程锚点固定@ANCHOR';

COMMENT ON COLUMN courseware_image_indexes.aoci_text IS
'规范化IAOCI全文：第一行机器编码，后续为[F][L][A][C][S][E][R][N]语义标签';

COMMENT ON TABLE courseware_image_relations IS
'图片间R关系：显式记录关系类型、继承范围和语义说明';

COMMIT;
