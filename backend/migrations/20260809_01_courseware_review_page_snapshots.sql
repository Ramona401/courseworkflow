-- ============================================================================
-- TE-DNA 2.0：课件已审核记录只读历史页面快照 R-03
-- 文件：20260809_01_courseware_review_page_snapshots.sql
-- ----------------------------------------------------------------------------
-- 目标：
--   1. 每次正式 L1/L2 审核提交时，冻结审核当时的全部课件页面；
--   2. 页面以后修改时，历史详情继续展示审核当时 HTML；
--   3. 原 courseware_pages 行以后被删除时，历史页面仍然可以展示；
--   4. 页面快照与 courseware_review_id 形成不可变审核证据；
--   5. 不复用仅保留最近 20 版的 courseware_page_versions 编辑恢复链。
--
-- 重要边界：
--   - 本迁移只建立 R-03 新历史快照结构，不伪造已有审核记录的历史页面；
--   - 存量审核发生时没有保存完整历史 HTML，不能拿当前 HTML 冒充历史；
--   - page_id 故意不建立到 courseware_pages 的外键，使原页面删除后快照仍存在；
--   - courseware_review_id 和 courseware_id 必须真实对应同一审核记录；
--   - 快照建立后禁止 UPDATE；
--   - 应用角色仅允许 SELECT / INSERT，不允许 UPDATE / DELETE；
--   - DELETE 仅保留父审核记录或课件删除时的数据库级联能力。
-- ============================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- 一、迁移前结构与应用角色检查
-- ============================================================================

DO $$
BEGIN
    IF to_regclass('public.courseware_reviews') IS NULL THEN
        RAISE EXCEPTION
            '缺少 courseware_reviews 表，无法建立 R-03 页面审核历史快照';
    END IF;

    IF to_regclass('public.coursewares') IS NULL THEN
        RAISE EXCEPTION
            '缺少 coursewares 表，无法建立 R-03 页面审核历史快照';
    END IF;

    IF to_regclass('public.courseware_pages') IS NULL THEN
        RAISE EXCEPTION
            '缺少 courseware_pages 表，无法建立 R-03 页面审核历史快照';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_roles
        WHERE rolname = 'tedna_user'
    ) THEN
        RAISE EXCEPTION
            '缺少应用数据库角色 tedna_user';
    END IF;

    IF to_regclass(
        'public.courseware_review_page_snapshots'
    ) IS NOT NULL THEN
        RAISE EXCEPTION
            'courseware_review_page_snapshots 已存在，禁止重复执行迁移';
    END IF;
END
$$;

-- ============================================================================
-- 二、正式审核页面不可变快照
-- ============================================================================

CREATE TABLE public.courseware_review_page_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 一次正式人工审核记录。
    courseware_review_id UUID NOT NULL
        REFERENCES public.courseware_reviews(id)
        ON DELETE CASCADE,

    -- 冗余保存课件身份，方便历史查询和关系复核。
    courseware_id UUID NOT NULL
        REFERENCES public.coursewares(id)
        ON DELETE CASCADE,

    -- 审核时真实稳定页面 ID。
    --
    -- 故意不 REFERENCES courseware_pages：
    -- 原页面删除后，本历史证据仍必须能够正常读取。
    page_id UUID NOT NULL,

    -- 审核时页面展示身份。
    page_number_snapshot INTEGER NOT NULL
        CHECK (page_number_snapshot > 0),

    page_title_snapshot TEXT NOT NULL DEFAULT '',

    -- 审核提交瞬间的完整页面 HTML。
    html_content TEXT NOT NULL,

    -- HTML UTF-8 字节 SHA-256。
    --
    -- 既用于历史完整性校验，也用于后续历史页与当前页明确比较。
    html_hash VARCHAR(64) NOT NULL
        CHECK (html_hash ~ '^[0-9a-f]{64}$'),

    -- 审核提交时 courseware_pages.updated_at。
    page_updated_at_snapshot TIMESTAMPTZ NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cw_review_page_snapshot_page
        UNIQUE (
            courseware_review_id,
            page_id
        ),

    CONSTRAINT uq_cw_review_page_snapshot_number
        UNIQUE (
            courseware_review_id,
            page_number_snapshot
        ),

    -- 数据库最终保证存储哈希与完整 HTML 一致，
    -- 浏览器或 Service 均不能伪造历史 HTML 指纹。
    CONSTRAINT chk_cw_review_page_snapshot_hash_content
        CHECK (
            html_hash = encode(
                digest(
                    convert_to(
                        html_content,
                        'UTF8'
                    ),
                    'sha256'
                ),
                'hex'
            )
        )
);

CREATE INDEX idx_cw_review_page_snapshot_courseware_page
ON public.courseware_review_page_snapshots (
    courseware_id,
    page_id
);

COMMENT ON TABLE public.courseware_review_page_snapshots IS
    'R-03正式课件审核提交时的全量页面不可变HTML快照；独立于作者页面编辑版本链';

COMMENT ON COLUMN
    public.courseware_review_page_snapshots.courseware_review_id IS
    '形成该页面历史证据的正式人工审核记录ID';

COMMENT ON COLUMN
    public.courseware_review_page_snapshots.courseware_id IS
    '审核时课件身份；必须与courseware_review_id所属课件一致';

COMMENT ON COLUMN
    public.courseware_review_page_snapshots.page_id IS
    '审核时稳定页面ID；故意不外键到courseware_pages，以支持原页面删除后继续查看历史';

COMMENT ON COLUMN
    public.courseware_review_page_snapshots.page_number_snapshot IS
    '审核提交时页码快照，不跟随后续页面排序变化';

COMMENT ON COLUMN
    public.courseware_review_page_snapshots.page_title_snapshot IS
    '审核提交时页面标题快照';

COMMENT ON COLUMN
    public.courseware_review_page_snapshots.html_content IS
    '审核提交时完整页面HTML，不得以后续当前页面覆盖';

COMMENT ON COLUMN
    public.courseware_review_page_snapshots.html_hash IS
    '审核时完整HTML的UTF-8字节SHA-256';

COMMENT ON COLUMN
    public.courseware_review_page_snapshots.page_updated_at_snapshot IS
    '审核提交时页面updated_at快照';

-- ============================================================================
-- 三、关系真实性与不可变数据库守卫
-- ============================================================================

CREATE FUNCTION public.guard_cw_review_page_snapshot_write()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- 防止把 A 课件的页面快照挂到 B 课件的审核记录。
        IF NOT EXISTS (
            SELECT 1
            FROM public.courseware_reviews AS review
            WHERE review.id = NEW.courseware_review_id
              AND review.courseware_id = NEW.courseware_id
        ) THEN
            RAISE EXCEPTION
                '审核页面快照与审核记录课件归属不一致'
                USING ERRCODE = '23514';
        END IF;

        RETURN NEW;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        RAISE EXCEPTION
            '正式审核页面历史快照不可修改'
            USING ERRCODE = '55000';
    END IF;

    RETURN NEW;
END
$$;

-- Trigger Function 不作为普通业务函数暴露。
REVOKE ALL
ON FUNCTION public.guard_cw_review_page_snapshot_write()
FROM PUBLIC;

REVOKE ALL
ON FUNCTION public.guard_cw_review_page_snapshot_write()
FROM tedna_user;

CREATE TRIGGER trg_cw_review_page_snapshot_validate_insert
BEFORE INSERT
ON public.courseware_review_page_snapshots
FOR EACH ROW
EXECUTE FUNCTION public.guard_cw_review_page_snapshot_write();

CREATE TRIGGER trg_cw_review_page_snapshot_immutable
BEFORE UPDATE
ON public.courseware_review_page_snapshots
FOR EACH ROW
EXECUTE FUNCTION public.guard_cw_review_page_snapshot_write();

COMMENT ON FUNCTION public.guard_cw_review_page_snapshot_write() IS
    'R-03审核页面快照数据库守卫：INSERT校验review与courseware真实关系，UPDATE一律拒绝';

-- ============================================================================
-- 四、应用数据库角色最小权限
-- ============================================================================
--
-- 页面审核历史是证据表：
--   - Service 需要 INSERT，在正式审核提交事务中一次性落入快照；
--   - 历史详情需要 SELECT；
--   - 普通应用业务不允许 UPDATE / DELETE；
--   - 父审核记录或父课件的合法删除仍可由数据库 FK 级联完成。
-- ============================================================================

REVOKE ALL PRIVILEGES
ON TABLE public.courseware_review_page_snapshots
FROM PUBLIC;

REVOKE ALL PRIVILEGES
ON TABLE public.courseware_review_page_snapshots
FROM tedna_user;

GRANT
    SELECT,
    INSERT
ON TABLE public.courseware_review_page_snapshots
TO tedna_user;

-- ============================================================================
-- 五、迁移完成
-- ============================================================================

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        'R-03课件正式审核页面不可变快照结构已建立；未对旧审核记录伪造历史HTML';
END
$$;
