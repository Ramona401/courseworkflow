-- ============================================================================
-- TE-DNA 2.0：AI美术风格工作室数据库基础
-- 文件：20260725_03_courseware_style_studio.sql
-- ----------------------------------------------------------------------------
-- 目标：
--   1. 保存老师与AI共创课程美术风格的会话；
--   2. 保存文字消息和可选参考图片；
--   3. 保存人物、知识对象、教学图解三类预览图；
--   4. 最终风格事实源仍是规范IAOCI，不使用JSON保存图片风格索引；
--   5. 确认后继续复用coursewares.style_anchor_*和@ANCHOR同步机制；
--   6. 数据库约束确保会话、消息、预览和图片资产属于同一课件。
-- ============================================================================

BEGIN;

-- ============================================================================
-- 一、风格共创会话
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_style_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),

    courseware_id uuid NOT NULL,
    user_id uuid NOT NULL,

    -- draft：正在对话；
    -- previewing：已经形成IAOCI，正在生成或检查预览；
    -- confirmed：老师已确认并写入课程正式锚点；
    -- archived：老师放弃、重开或历史归档。
    status varchar(16) NOT NULL DEFAULT 'draft',

    -- style_only：只提取艺术风格；
    -- style_character：艺术风格及明确固定主体；
    -- inspiration：只取抽象灵感，进一步弱化原图复刻。
    reference_mode varchar(24) NOT NULL DEFAULT 'style_only',

    -- 本次会话最近使用的参考图片。
    reference_asset_id uuid,

    -- 老师确认时选择用于课程锚点展示的图片。
    -- 可以是上传参考图，也可以是三张预览中的一张。
    confirmed_asset_id uuid,

    -- 当前完整课程锚点IAOCI草稿。
    -- IAOCI是风格事实源；聊天消息只是形成该索引的过程记录。
    style_aoci_text text NOT NULL DEFAULT '',

    -- 给老师阅读的简短风格摘要，不参与图片索引解析。
    style_summary text NOT NULL DEFAULT '',

    version integer NOT NULL DEFAULT 1,
    confirmed_at timestamptz,

    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT fk_courseware_style_sessions_courseware
        FOREIGN KEY (courseware_id)
        REFERENCES coursewares(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_style_sessions_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_style_sessions_reference_asset
        FOREIGN KEY (reference_asset_id)
        REFERENCES courseware_assets(id)
        ON DELETE SET NULL,

    CONSTRAINT fk_courseware_style_sessions_confirmed_asset
        FOREIGN KEY (confirmed_asset_id)
        REFERENCES courseware_assets(id)
        ON DELETE SET NULL,

    -- 供消息和预览表使用复合外键，确保courseware_id边界一致。
    CONSTRAINT uq_courseware_style_sessions_id_courseware
        UNIQUE (id, courseware_id),

    CONSTRAINT ck_courseware_style_sessions_status
        CHECK (
            status IN (
                'draft',
                'previewing',
                'confirmed',
                'archived'
            )
        ),

    CONSTRAINT ck_courseware_style_sessions_reference_mode
        CHECK (
            reference_mode IN (
                'style_only',
                'style_character',
                'inspiration'
            )
        ),

    CONSTRAINT ck_courseware_style_sessions_version
        CHECK (version >= 1),

    -- 确认态必须已经形成IAOCI、有确认图片并记录确认时间。
    CONSTRAINT ck_courseware_style_sessions_confirmed
        CHECK (
            status <> 'confirmed'
            OR (
                confirmed_asset_id IS NOT NULL
                AND length(btrim(style_aoci_text)) > 0
                AND confirmed_at IS NOT NULL
            )
        )
);

-- 同一课件同一时刻最多只有一个正在编辑或预览的会话。
CREATE UNIQUE INDEX IF NOT EXISTS ux_courseware_style_sessions_active
ON courseware_style_sessions (courseware_id)
WHERE status IN ('draft', 'previewing');

CREATE INDEX IF NOT EXISTS ix_courseware_style_sessions_user_updated
ON courseware_style_sessions (user_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS ix_courseware_style_sessions_courseware_status
ON courseware_style_sessions (courseware_id, status, updated_at DESC);

-- ============================================================================
-- 二、风格共创消息
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_style_messages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),

    session_id uuid NOT NULL,
    courseware_id uuid NOT NULL,

    role varchar(16) NOT NULL,

    -- 用户可只上传图片而不输入文字，因此content允许空串。
    content text NOT NULL DEFAULT '',

    -- 当前消息随附的参考图片，可空。
    reference_asset_id uuid,

    -- AI回复后形成的完整IAOCI快照。
    -- 用户消息通常为空；助手消息可保存本轮新版本。
    style_aoci_text text NOT NULL DEFAULT '',

    -- 会话内严格递增序号，由仓储事务分配。
    sequence_no integer NOT NULL,

    created_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT fk_courseware_style_messages_session
        FOREIGN KEY (session_id, courseware_id)
        REFERENCES courseware_style_sessions(id, courseware_id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_style_messages_courseware
        FOREIGN KEY (courseware_id)
        REFERENCES coursewares(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_style_messages_reference_asset
        FOREIGN KEY (reference_asset_id)
        REFERENCES courseware_assets(id)
        ON DELETE SET NULL,

    CONSTRAINT ck_courseware_style_messages_role
        CHECK (role IN ('user', 'assistant')),

    CONSTRAINT ck_courseware_style_messages_sequence
        CHECK (sequence_no >= 1),

    -- 一条消息必须有文字或图片，禁止完全空消息。
    CONSTRAINT ck_courseware_style_messages_payload
        CHECK (
            length(btrim(content)) > 0
            OR reference_asset_id IS NOT NULL
        ),

    CONSTRAINT uq_courseware_style_messages_sequence
        UNIQUE (session_id, sequence_no)
);

CREATE INDEX IF NOT EXISTS ix_courseware_style_messages_session_created
ON courseware_style_messages (session_id, sequence_no);

CREATE INDEX IF NOT EXISTS ix_courseware_style_messages_courseware
ON courseware_style_messages (courseware_id, created_at DESC);

-- ============================================================================
-- 三、风格预览
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_style_previews (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),

    session_id uuid NOT NULL,
    courseware_id uuid NOT NULL,

    -- character：人物情境；
    -- object：知识对象；
    -- diagram：教学图解。
    preview_type varchar(16) NOT NULL,

    asset_id uuid,

    generation_prompt text NOT NULL DEFAULT '',

    status varchar(16) NOT NULL DEFAULT 'pending',
    last_error text NOT NULL DEFAULT '',
    version integer NOT NULL DEFAULT 1,

    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT fk_courseware_style_previews_session
        FOREIGN KEY (session_id, courseware_id)
        REFERENCES courseware_style_sessions(id, courseware_id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_style_previews_courseware
        FOREIGN KEY (courseware_id)
        REFERENCES coursewares(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_courseware_style_previews_asset
        FOREIGN KEY (asset_id)
        REFERENCES courseware_assets(id)
        ON DELETE SET NULL,

    CONSTRAINT ck_courseware_style_previews_type
        CHECK (
            preview_type IN (
                'character',
                'object',
                'diagram'
            )
        ),

    CONSTRAINT ck_courseware_style_previews_status
        CHECK (
            status IN (
                'pending',
                'generating',
                'generated',
                'failed',
                'stale'
            )
        ),

    CONSTRAINT ck_courseware_style_previews_version
        CHECK (version >= 1),

    CONSTRAINT ck_courseware_style_previews_generated_asset
        CHECK (
            status <> 'generated'
            OR asset_id IS NOT NULL
        ),

    CONSTRAINT uq_courseware_style_previews_type
        UNIQUE (session_id, preview_type)
);

CREATE INDEX IF NOT EXISTS ix_courseware_style_previews_courseware_status
ON courseware_style_previews (courseware_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS ix_courseware_style_previews_asset
ON courseware_style_previews (asset_id)
WHERE asset_id IS NOT NULL;

-- ============================================================================
-- 四、同课件图片资产边界触发器
-- ============================================================================

CREATE OR REPLACE FUNCTION tedna_validate_style_session_assets()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.reference_asset_id IS NOT NULL THEN
        PERFORM 1
        FROM courseware_assets
        WHERE id = NEW.reference_asset_id
          AND courseware_id = NEW.courseware_id
          AND asset_type = 'image';

        IF NOT FOUND THEN
            RAISE EXCEPTION
                '风格会话参考图片不存在、不是图片或不属于当前课件';
        END IF;
    END IF;

    IF NEW.confirmed_asset_id IS NOT NULL THEN
        PERFORM 1
        FROM courseware_assets
        WHERE id = NEW.confirmed_asset_id
          AND courseware_id = NEW.courseware_id
          AND asset_type = 'image';

        IF NOT FOUND THEN
            RAISE EXCEPTION
                '风格会话确认图片不存在、不是图片或不属于当前课件';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION tedna_validate_style_message_asset()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.reference_asset_id IS NOT NULL THEN
        PERFORM 1
        FROM courseware_assets
        WHERE id = NEW.reference_asset_id
          AND courseware_id = NEW.courseware_id
          AND asset_type = 'image';

        IF NOT FOUND THEN
            RAISE EXCEPTION
                '风格消息参考图片不存在、不是图片或不属于当前课件';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION tedna_validate_style_preview_asset()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.asset_id IS NOT NULL THEN
        PERFORM 1
        FROM courseware_assets
        WHERE id = NEW.asset_id
          AND courseware_id = NEW.courseware_id
          AND asset_type = 'image';

        IF NOT FOUND THEN
            RAISE EXCEPTION
                '风格预览图片不存在、不是图片或不属于当前课件';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_validate_style_session_assets
ON courseware_style_sessions;

CREATE TRIGGER trg_validate_style_session_assets
BEFORE INSERT OR UPDATE
ON courseware_style_sessions
FOR EACH ROW
EXECUTE FUNCTION tedna_validate_style_session_assets();

DROP TRIGGER IF EXISTS trg_validate_style_message_asset
ON courseware_style_messages;

CREATE TRIGGER trg_validate_style_message_asset
BEFORE INSERT OR UPDATE
ON courseware_style_messages
FOR EACH ROW
EXECUTE FUNCTION tedna_validate_style_message_asset();

DROP TRIGGER IF EXISTS trg_validate_style_preview_asset
ON courseware_style_previews;

CREATE TRIGGER trg_validate_style_preview_asset
BEFORE INSERT OR UPDATE
ON courseware_style_previews
FOR EACH ROW
EXECUTE FUNCTION tedna_validate_style_preview_asset();

-- ============================================================================
-- 五、说明
-- ============================================================================

COMMENT ON TABLE courseware_style_sessions IS
'AI美术风格工作室会话；保存当前课程锚点IAOCI草稿和确认状态';

COMMENT ON COLUMN courseware_style_sessions.style_aoci_text IS
'当前完整课程锚点IAOCI；这是最终风格事实源，不是聊天JSON';

COMMENT ON COLUMN courseware_style_sessions.reference_mode IS
'参考图使用方式：只提取风格、风格加固定主体、抽象灵感';

COMMENT ON TABLE courseware_style_messages IS
'老师与AI的风格共创消息；assistant消息可保存本轮IAOCI快照';

COMMENT ON TABLE courseware_style_previews IS
'风格确认前的三类测试图：人物、知识对象、教学图解';

COMMIT;
