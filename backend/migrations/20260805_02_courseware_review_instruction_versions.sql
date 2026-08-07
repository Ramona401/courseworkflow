-- ============================================================================
-- TE-DNA 2.0：课件审核修改指令不可变版本体系 V2.1
-- 文件：20260805_02_courseware_review_instruction_versions.sql
-- ----------------------------------------------------------------------------
-- 对应PRD：R-01 指令版本与重新确认。
--
-- 本文件负责：
--   1. 创建不可变指令版本表；
--   2. 为整改项增加当前、正式交付和页面应用版本引用；
--   3. 将存量非空confirmed_instruction确定性回填为V1；
--   4. 建立版本归属、兼容快照、交付和应用引用约束；
--   5. 建立必要索引与数据库注释。
--
-- 本文件只开启事务，不提交事务。
-- 必须在同一个psql连接中紧接着执行：
--
--   20260805_03_courseware_review_instruction_version_guards.sql
--
-- 第二个文件完成数据库守卫、最小权限并统一COMMIT。
-- 任一文件失败时必须整体回滚，禁止只提交本文件。
-- ============================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- 一、迁移前数据检查
-- ============================================================================

DO $$
BEGIN
    IF to_regclass(
        'public.courseware_review_items'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少courseware_review_items表';
    END IF;

    -- 已正式交付的问题必须具有可迁移的确认指令。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE feedback_id IS NOT NULL
          AND BTRIM(
              COALESCE(
                  confirmed_instruction,
                  ''
              )
          ) = ''
    ) THEN
        RAISE EXCEPTION
            '存在已正式交付但确认指令为空的整改项，禁止自动迁移';
    END IF;

    -- 已产生页面修改事实的问题必须能够追溯到确认指令。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE applied_at IS NOT NULL
          AND BTRIM(
              COALESCE(
                  confirmed_instruction,
                  ''
              )
          ) = ''
    ) THEN
        RAISE EXCEPTION
            '存在已记录页面修改但确认指令为空的整改项，禁止自动迁移';
    END IF;

    -- 正在应用的记录也必须具有确认指令，迁移后会自动绑定应用版本。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'applying'
          AND BTRIM(
              COALESCE(
                  confirmed_instruction,
                  ''
              )
          ) = ''
    ) THEN
        RAISE EXCEPTION
            '存在正在应用但确认指令为空的整改项，禁止自动迁移';
    END IF;
END
$$;

-- ============================================================================
-- 二、不可变指令版本表
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_review_instruction_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    item_id UUID NOT NULL
        REFERENCES courseware_review_items(id)
        ON DELETE CASCADE,

    version_no INTEGER NOT NULL
        CHECK (version_no > 0),

    -- 指令正文创建后不可原地修改。
    content TEXT NOT NULL
        CHECK (length(BTRIM(content)) > 0),

    -- 使用与Go端一致的UTF-8字节SHA-256。
    content_hash VARCHAR(64) NOT NULL
        CHECK (content_hash ~ '^[0-9a-f]{64}$'),

    source_type VARCHAR(32) NOT NULL
        CHECK (
            source_type IN (
                'legacy_backfill',
                'legacy_direct_update',
                'manual',
                'ai_candidate',
                'global_discussion'
            )
        ),

    created_by UUID NOT NULL
        REFERENCES users(id)
        ON DELETE RESTRICT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    confirmed_by UUID NULL
        REFERENCES users(id)
        ON DELETE RESTRICT,

    confirmed_at TIMESTAMPTZ NULL,

    -- 整课问题允许为空字符串；
    -- 非空时必须是小写SHA-256十六进制文本。
    page_snapshot_hash VARCHAR(64) NOT NULL DEFAULT ''
        CHECK (
            page_snapshot_hash = ''
            OR page_snapshot_hash ~ '^[0-9a-f]{64}$'
        ),

    status VARCHAR(24) NOT NULL
        CHECK (
            status IN (
                'draft',
                'confirmed',
                'superseded',
                'invalid_for_page'
            )
        ),

    CONSTRAINT uq_cw_review_instruction_version_no
        UNIQUE (item_id, version_no),

    -- 整改项引用版本时使用该复合唯一键保证版本归属。
    CONSTRAINT uq_cw_review_instruction_version_id_item
        UNIQUE (id, item_id),

    CONSTRAINT chk_cw_review_instruction_version_hash
        CHECK (
            content_hash = encode(
                digest(
                    convert_to(
                        content,
                        'UTF8'
                    ),
                    'sha256'
                ),
                'hex'
            )
        ),

    CONSTRAINT chk_cw_review_instruction_version_confirmation
        CHECK (
            (
                status = 'draft'
                AND confirmed_by IS NULL
                AND confirmed_at IS NULL
            )
            OR
            (
                status IN (
                    'confirmed',
                    'superseded',
                    'invalid_for_page'
                )
                AND confirmed_by IS NOT NULL
                AND confirmed_at IS NOT NULL
            )
        )
);

-- 同一整改项同时最多只有一个仍具执行资格的confirmed版本。
CREATE UNIQUE INDEX IF NOT EXISTS
    uq_cw_review_instruction_current_confirmed
ON courseware_review_instruction_versions(item_id)
WHERE status = 'confirmed';

CREATE INDEX IF NOT EXISTS
    idx_cw_review_instruction_version_item
ON courseware_review_instruction_versions(
    item_id,
    version_no DESC
);

CREATE INDEX IF NOT EXISTS
    idx_cw_review_instruction_version_creator
ON courseware_review_instruction_versions(
    created_by,
    created_at DESC
);

CREATE INDEX IF NOT EXISTS
    idx_cw_review_instruction_version_confirmer
ON courseware_review_instruction_versions(
    confirmed_by,
    confirmed_at DESC
)
WHERE confirmed_by IS NOT NULL;

COMMENT ON TABLE courseware_review_instruction_versions IS
    '课件审核整改项不可变修改指令版本；正文、编号和身份不允许原地覆盖';

COMMENT ON COLUMN courseware_review_instruction_versions.item_id IS
    '所属整改项；版本号只在同一整改项内递增';

COMMENT ON COLUMN courseware_review_instruction_versions.version_no IS
    '整改项内从1开始连续递增的版本号';

COMMENT ON COLUMN courseware_review_instruction_versions.content IS
    '不可变指令正文；错误版本只能由新版本替代';

COMMENT ON COLUMN courseware_review_instruction_versions.content_hash IS
    '指令正文UTF-8字节的SHA-256';

COMMENT ON COLUMN courseware_review_instruction_versions.source_type IS
    '版本来源：存量回填、旧接口兼容、人工、AI候选或全局讨论';

COMMENT ON COLUMN courseware_review_instruction_versions.page_snapshot_hash IS
    '确认时页面HTML哈希；整课问题或历史缺失时允许为空';

COMMENT ON COLUMN courseware_review_instruction_versions.status IS
    'draft、confirmed、superseded或invalid_for_page';

-- ============================================================================
-- 三、整改项版本引用
-- ============================================================================

ALTER TABLE courseware_review_items
    ADD COLUMN IF NOT EXISTS
        current_instruction_version_id UUID NULL,
    ADD COLUMN IF NOT EXISTS
        delivered_instruction_version_id UUID NULL,
    ADD COLUMN IF NOT EXISTS
        applied_instruction_version_id UUID NULL;

COMMENT ON COLUMN
    courseware_review_items.current_instruction_version_id IS
    '当前指令版本；页面失效后仍可指向invalid_for_page历史版本';

COMMENT ON COLUMN
    courseware_review_items.delivered_instruction_version_id IS
    '正式审核提交时实际交付作者的确认版本；交付后不可替换';

COMMENT ON COLUMN
    courseware_review_items.applied_instruction_version_id IS
    '页面微调开始和完成时实际使用的版本；完成后不可替换';

-- ============================================================================
-- 四、存量非空确认指令回填为V1
-- ============================================================================

INSERT INTO courseware_review_instruction_versions (
    item_id,
    version_no,
    content,
    content_hash,
    source_type,
    created_by,
    created_at,
    confirmed_by,
    confirmed_at,
    page_snapshot_hash,
    status
)
SELECT
    item.id,
    1,
    BTRIM(item.confirmed_instruction),
    encode(
        digest(
            convert_to(
                BTRIM(item.confirmed_instruction),
                'UTF8'
            ),
            'sha256'
        ),
        'hex'
    ),
    'legacy_backfill',
    item.created_by,
    COALESCE(
        item.confirmed_at,
        item.updated_at,
        item.created_at,
        NOW()
    ),
    item.created_by,
    COALESCE(
        item.confirmed_at,
        item.updated_at,
        item.created_at,
        NOW()
    ),
    BTRIM(
        COALESCE(
            item.page_html_hash,
            ''
        )
    ),
    CASE
        WHEN item.status IN (
            'stale',
            'orphaned'
        )
            THEN 'invalid_for_page'
        ELSE 'confirmed'
    END
FROM courseware_review_items AS item
WHERE BTRIM(
          COALESCE(
              item.confirmed_instruction,
              ''
          )
      ) <> ''
  AND NOT EXISTS (
      SELECT 1
      FROM courseware_review_instruction_versions AS version
      WHERE version.item_id = item.id
  );

UPDATE courseware_review_items AS item
SET current_instruction_version_id = version.id
FROM courseware_review_instruction_versions AS version
WHERE version.item_id = item.id
  AND version.version_no = 1
  AND item.current_instruction_version_id IS NULL
  AND BTRIM(
      COALESCE(
          item.confirmed_instruction,
          ''
      )
  ) <> '';

-- 正式交付问题固化当时唯一存在的V1。
UPDATE courseware_review_items AS item
SET delivered_instruction_version_id =
        item.current_instruction_version_id
WHERE item.feedback_id IS NOT NULL
  AND item.delivered_instruction_version_id IS NULL;

-- 已完成修改或正在修改的存量问题绑定当时唯一存在的V1。
UPDATE courseware_review_items AS item
SET applied_instruction_version_id =
        COALESCE(
            item.delivered_instruction_version_id,
            item.current_instruction_version_id
        )
WHERE (
        item.applied_at IS NOT NULL
        OR item.status = 'applying'
      )
  AND item.applied_instruction_version_id IS NULL;

-- ============================================================================
-- 五、版本归属外键
-- ============================================================================
-- 三个外键延迟到事务提交时检查。
--
-- 这样删除整改项时，item→version和version→item形成的循环引用
-- 可以与ON DELETE CASCADE在同一事务中安全收敛。

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
              'courseware_review_items'::regclass
          AND conname =
              'fk_cw_review_item_current_instruction_version'
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT
                fk_cw_review_item_current_instruction_version
            FOREIGN KEY (
                current_instruction_version_id,
                id
            )
            REFERENCES courseware_review_instruction_versions(
                id,
                item_id
            )
            DEFERRABLE INITIALLY DEFERRED;
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
              'courseware_review_items'::regclass
          AND conname =
              'fk_cw_review_item_delivered_instruction_version'
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT
                fk_cw_review_item_delivered_instruction_version
            FOREIGN KEY (
                delivered_instruction_version_id,
                id
            )
            REFERENCES courseware_review_instruction_versions(
                id,
                item_id
            )
            DEFERRABLE INITIALLY DEFERRED;
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
              'courseware_review_items'::regclass
          AND conname =
              'fk_cw_review_item_applied_instruction_version'
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT
                fk_cw_review_item_applied_instruction_version
            FOREIGN KEY (
                applied_instruction_version_id,
                id
            )
            REFERENCES courseware_review_instruction_versions(
                id,
                item_id
            )
            DEFERRABLE INITIALLY DEFERRED;
    END IF;
END
$$;

-- ============================================================================
-- 六、整改项版本组合约束
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
              'courseware_review_items'::regclass
          AND conname =
              'chk_cw_review_item_instruction_compat'
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT
                chk_cw_review_item_instruction_compat
            CHECK (
                (
                    BTRIM(
                        COALESCE(
                            confirmed_instruction,
                            ''
                        )
                    ) = ''
                    AND current_instruction_version_id IS NULL
                )
                OR
                (
                    BTRIM(
                        COALESCE(
                            confirmed_instruction,
                            ''
                        )
                    ) <> ''
                    AND current_instruction_version_id IS NOT NULL
                )
            );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
              'courseware_review_items'::regclass
          AND conname =
              'chk_cw_review_item_delivered_instruction_version'
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT
                chk_cw_review_item_delivered_instruction_version
            CHECK (
                (
                    feedback_id IS NULL
                    AND delivered_instruction_version_id IS NULL
                )
                OR
                (
                    feedback_id IS NOT NULL
                    AND delivered_instruction_version_id IS NOT NULL
                )
            );
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
              'courseware_review_items'::regclass
          AND conname =
              'chk_cw_review_item_applied_instruction_version'
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT
                chk_cw_review_item_applied_instruction_version
            CHECK (
                (
                    applied_instruction_version_id IS NULL
                    AND applied_at IS NULL
                    AND status <> 'applying'
                )
                OR
                (
                    applied_instruction_version_id IS NOT NULL
                    AND status = 'applying'
                    AND applied_at IS NULL
                )
                OR
                (
                    applied_instruction_version_id IS NOT NULL
                    AND status IN (
                        'applied',
                        'resolved',
                        'stale',
                        'orphaned'
                    )
                    AND applied_at IS NOT NULL
                )
            );
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_current_instruction_version
ON courseware_review_items(current_instruction_version_id)
WHERE current_instruction_version_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_delivered_instruction_version
ON courseware_review_items(delivered_instruction_version_id)
WHERE delivered_instruction_version_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS
    idx_cw_review_item_applied_instruction_version
ON courseware_review_items(applied_instruction_version_id)
WHERE applied_instruction_version_id IS NOT NULL;

DO $$
BEGIN
    RAISE NOTICE
        'R-01指令版本结构与存量V1回填完成，等待同事务执行守卫迁移';
END
$$;

-- 本文件故意不执行COMMIT。
