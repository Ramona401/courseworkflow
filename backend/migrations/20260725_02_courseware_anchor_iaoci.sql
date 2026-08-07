-- ============================================================================
-- TE-DNA 2.0：课程锚点IAOCI同步与独立锚点提示词
-- 文件：20260725_02_courseware_anchor_iaoci.sql
-- ----------------------------------------------------------------------------
-- 修复说明：
--   - prompts表没有updated_at列，本版本不再写入该列；
--   - 迁移全程位于单事务内，任一步失败则全部回滚；
--   - 使用DO代码块执行历史锚点回填，不再输出大量void空行；
--   - 独立提示词键不会替换旧prompt_courseware_vaoci_extract。
-- ============================================================================

BEGIN;

-- 从IAOCI第一行读取指定机器编码。
CREATE OR REPLACE FUNCTION tedna_image_aoci_header_value(
    index_text text,
    field_code text
)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT btrim(split_part(segment, ':', 2))
    FROM unnest(
        string_to_array(
            split_part(COALESCE(index_text, ''), E'\n', 1),
            '|'
        )
    ) AS segment
    WHERE btrim(split_part(segment, ':', 1)) = field_code
    LIMIT 1
$$;

-- 从IAOCI中读取指定语义标签。
CREATE OR REPLACE FUNCTION tedna_image_aoci_tag_value(
    index_text text,
    tag_code text
)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT btrim(substring(line FROM 4))
    FROM unnest(
        string_to_array(
            replace(COALESCE(index_text, ''), E'\r', ''),
            E'\n'
        )
    ) AS line
    WHERE left(btrim(line), 3) = '[' || tag_code || ']'
    LIMIT 1
$$;

-- 将coursewares中的课程锚点同步到@ANCHOR索引。
CREATE OR REPLACE FUNCTION tedna_sync_courseware_anchor_iaoci(
    p_courseware_id uuid,
    p_asset_id uuid,
    p_aoci text
)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    validated_asset_id uuid;
    index_version_value integer;
    continuity_value integer;
    index_type_value text;
    usage_role_value text;
    subject_type_value text;
    aspect_ratio_value text;
    relation_count_value text;
    focus_value text;
    layout_value text;
    art_value text;
    character_value text;
    scene_value text;
    export_value text;
    negative_value text;
    generation_prompt_value text;
BEGIN
    -- 清除课程锚点时同步删除@ANCHOR索引。
    IF p_asset_id IS NULL
       OR btrim(COALESCE(p_aoci, '')) = '' THEN
        DELETE FROM courseware_image_indexes
        WHERE courseware_id = p_courseware_id
          AND index_type = 'A';

        RETURN;
    END IF;

    -- 兼容旧版单行VAOCI。
    -- 只有新版规范IAOCI才进入新图片索引表。
    IF left(ltrim(p_aoci), 3) <> 'IK:'
       OR position(
            E'\n[F]' IN replace(p_aoci, E'\r', '')
       ) = 0 THEN
        DELETE FROM courseware_image_indexes
        WHERE courseware_id = p_courseware_id
          AND index_type = 'A';

        RETURN;
    END IF;

    -- 锚点资产必须存在、属于同一课件且为图片。
    SELECT
        id,
        COALESCE(generation_prompt, '')
    INTO
        validated_asset_id,
        generation_prompt_value
    FROM courseware_assets
    WHERE id = p_asset_id
      AND courseware_id = p_courseware_id
      AND asset_type = 'image'
    LIMIT 1;

    IF validated_asset_id IS NULL THEN
        DELETE FROM courseware_image_indexes
        WHERE courseware_id = p_courseware_id
          AND index_type = 'A';

        RETURN;
    END IF;

    index_version_value :=
        tedna_image_aoci_header_value(
            p_aoci,
            'IV'
        )::integer;

    continuity_value :=
        tedna_image_aoci_header_value(
            p_aoci,
            'CT'
        )::integer;

    index_type_value :=
        tedna_image_aoci_header_value(
            p_aoci,
            'IT'
        );

    usage_role_value :=
        tedna_image_aoci_header_value(
            p_aoci,
            'UR'
        );

    subject_type_value :=
        tedna_image_aoci_header_value(
            p_aoci,
            'SB'
        );

    aspect_ratio_value :=
        tedna_image_aoci_header_value(
            p_aoci,
            'AR'
        );

    relation_count_value :=
        tedna_image_aoci_header_value(
            p_aoci,
            'RC'
        );

    focus_value :=
        tedna_image_aoci_tag_value(
            p_aoci,
            'F'
        );

    layout_value :=
        tedna_image_aoci_tag_value(
            p_aoci,
            'L'
        );

    art_value :=
        tedna_image_aoci_tag_value(
            p_aoci,
            'A'
        );

    character_value :=
        tedna_image_aoci_tag_value(
            p_aoci,
            'C'
        );

    scene_value :=
        tedna_image_aoci_tag_value(
            p_aoci,
            'S'
        );

    export_value :=
        tedna_image_aoci_tag_value(
            p_aoci,
            'E'
        );

    negative_value :=
        tedna_image_aoci_tag_value(
            p_aoci,
            'N'
        );

    -- 只允许课程锚点协议进入@ANCHOR槽。
    IF tedna_image_aoci_header_value(
            p_aoci,
            'IK'
       ) <> '@ANCHOR'
       OR index_type_value <> 'A'
       OR relation_count_value <> '0' THEN
        RAISE EXCEPTION
            '课程锚点IAOCI编码非法：IK必须为@ANCHOR、IT必须为A、RC必须为0';
    END IF;

    INSERT INTO courseware_image_indexes (
        courseware_id,
        page_id,
        placeholder_id,
        image_key,
        slot_order,
        index_version,
        index_type,
        usage_role,
        continuity_level,
        subject_type,
        aspect_ratio,
        relation_count,
        focus_text,
        layout_text,
        art_text,
        character_text,
        scene_text,
        export_text,
        negative_text,
        aoci_text,
        generation_prompt,
        asset_id,
        status,
        last_error,
        version
    )
    VALUES (
        p_courseware_id,
        NULL,
        'style-anchor',
        '@ANCHOR',
        0,
        index_version_value,
        index_type_value,
        usage_role_value,
        continuity_value,
        subject_type_value,
        aspect_ratio_value,
        relation_count_value,
        COALESCE(NULLIF(focus_value, ''), 'Ø'),
        COALESCE(NULLIF(layout_value, ''), 'Ø'),
        COALESCE(NULLIF(art_value, ''), 'Ø'),
        COALESCE(NULLIF(character_value, ''), 'Ø'),
        COALESCE(NULLIF(scene_value, ''), 'Ø'),
        COALESCE(NULLIF(export_value, ''), 'Ø'),
        COALESCE(NULLIF(negative_value, ''), 'Ø'),
        btrim(p_aoci),
        COALESCE(generation_prompt_value, ''),
        validated_asset_id,
        'generated',
        '',
        1
    )
    ON CONFLICT (courseware_id)
    WHERE index_type = 'A'
    DO UPDATE SET
        placeholder_id = EXCLUDED.placeholder_id,
        image_key = EXCLUDED.image_key,
        slot_order = EXCLUDED.slot_order,
        index_version = EXCLUDED.index_version,
        usage_role = EXCLUDED.usage_role,
        continuity_level = EXCLUDED.continuity_level,
        subject_type = EXCLUDED.subject_type,
        aspect_ratio = EXCLUDED.aspect_ratio,
        relation_count = EXCLUDED.relation_count,
        focus_text = EXCLUDED.focus_text,
        layout_text = EXCLUDED.layout_text,
        art_text = EXCLUDED.art_text,
        character_text = EXCLUDED.character_text,
        scene_text = EXCLUDED.scene_text,
        export_text = EXCLUDED.export_text,
        negative_text = EXCLUDED.negative_text,
        aoci_text = EXCLUDED.aoci_text,
        generation_prompt = EXCLUDED.generation_prompt,
        asset_id = EXCLUDED.asset_id,
        status = EXCLUDED.status,
        last_error = '',
        version = courseware_image_indexes.version + 1,
        updated_at = now();
END;
$$;

CREATE OR REPLACE FUNCTION tedna_courseware_anchor_sync_trigger()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM tedna_sync_courseware_anchor_iaoci(
        NEW.id,
        NEW.style_anchor_asset_id,
        NEW.style_anchor_vaoci
    );

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_courseware_anchor_iaoci_sync
ON coursewares;

CREATE TRIGGER trg_courseware_anchor_iaoci_sync
AFTER INSERT OR UPDATE OF
    style_anchor_asset_id,
    style_anchor_vaoci
ON coursewares
FOR EACH ROW
EXECUTE FUNCTION tedna_courseware_anchor_sync_trigger();

-- 安全回填现有规范IAOCI锚点。
-- 旧版单行VAOCI由同步函数自动跳过。
DO $$
DECLARE
    anchor_record record;
BEGIN
    FOR anchor_record IN
        SELECT
            id,
            style_anchor_asset_id,
            style_anchor_vaoci
        FROM coursewares
        WHERE style_anchor_asset_id IS NOT NULL
          AND COALESCE(style_anchor_vaoci, '') <> ''
    LOOP
        PERFORM tedna_sync_courseware_anchor_iaoci(
            anchor_record.id,
            anchor_record.style_anchor_asset_id,
            anchor_record.style_anchor_vaoci
        );
    END LOOP;
END;
$$;

-- 新增独立课程锚点IAOCI提示词。
INSERT INTO prompts (
    prompt_key,
    version,
    content,
    is_current,
    created_at
)
SELECT
    'prompt_courseware_image_anchor_iaoci',
    1,
    $PROMPT$
<!-- TEDNA_IAOCI_ANCHOR_V1 -->

你是 TE-DNA 课件图像 IAOCI 的课程锚点索引器。

任务：分析用户提供的一张锚点图片，只提取：

1. 可被整套课件继承的艺术风格；
2. 可被跨图复用的主要人物、动物或标志性主体身份。

严禁把锚点图片中的教室、课桌、黑板、讲台、家具、具体背景、具体构图、主体位置、景别、镜头和道具位置写成全课件固定条件。

只输出 IAOCI，不得输出 Markdown代码围栏、标题、说明、JSON或其它文字。

严格输出以下9行：

IK:@ANCHOR|IV:1|IT:A|UR:BG|CT:0|SB:<N/P/A/O/M>|AR:F|RC:0
[F]定义本课件统一艺术语言和可复用的主要角色或标志性主体
[L]Ø；课程锚点不锁定具体构图、镜头、景别、主体位置和留白
[A]<只写艺术媒介、造型语言、线条、材质、色彩、光影、渲染方式和精细度>
[C]<使用稳定实体编号描述可复用主体，例如C1人物、A1动物、O1物体；没有可复用主体时写Ø>
[S]Ø；锚点原图中的教室、课桌、黑板、家具、背景、镜头和道具位置均不继承
[E]统一渲染质量；具体尺寸和画幅由每张页面图片自己的槽位决定
[R]0
[N]禁止继承锚点环境、家具、构图、镜头、景别和道具位置；禁止把固定角色强行加入不需要人物的页面

字段说明：

- SB=N：没有固定主体；
- SB=P：主要固定主体是人物；
- SB=A：主要固定主体是动物；
- SB=O：主要固定主体是标志性物体；
- SB=M：包含多种固定主体；
- [A]中不得出现教室、课桌、黑板、讲台、家具、人物位置、前景、中景、后景、镜头或构图描述；
- [C]只描述稳定身份和外貌，不描述当前动作和所在场景；
- [L]、[S]、[R]必须使用上面给出的固定内容。
$PROMPT$,
    true,
    now()
WHERE NOT EXISTS (
    SELECT 1
    FROM prompts
    WHERE prompt_key =
        'prompt_courseware_image_anchor_iaoci'
);

COMMIT;
