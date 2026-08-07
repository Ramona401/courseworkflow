-- 20260724_01_courseware_review_items.sql
--
-- 课件AI审核与作者自审的整改闭环数据库底座。
--
-- 本迁移解决以下问题：
--   1. 正式审核退回时只能保存一段整体文字，无法保存整课AI评价快照；
--   2. AI问题只绑定页码，页面重新排序后无法稳定定位原页面；
--   3. 单条问题没有独立的讨论、确认、应用和解决状态；
--   4. 作者无法在退回后继续查看和处理审核员正式交付的页级问题；
--   5. 现有courseware_ai_review_messages没有关联具体问题。
--
-- 设计原则：
--   1. courseware_review_feedback保存一次正式人工审核的不可变整体反馈快照；
--   2. courseware_review_items保存可逐条讨论和整改的问题；
--   3. 一条跨多页AI发现应在服务层拆成多个页级整改项；
--   4. page_id是稳定定位依据，page_number_snapshot只负责展示审核时页码；
--   5. page_html_hash用于判断页面修改后原意见是否已经过期；
--   6. 自审和正式审核共用整改项表，但通过source_type和review_level严格区分；
--   7. AI分析结果不能直接改变人工审核决定或课件发布状态。
--
-- 本迁移不搬运存量AI审核报告，不自动生成整改项。
-- 存量会话仍可按原方式展示，新会话由后续服务层显式生成整改项。

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- 一、正式审核整体反馈快照
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_review_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- 一条正式人工审核记录只能对应一份反馈快照。
    courseware_review_id UUID NOT NULL
        REFERENCES courseware_reviews(id) ON DELETE CASCADE,

    courseware_id UUID NOT NULL
        REFERENCES coursewares(id) ON DELETE CASCADE,

    -- 原始AI审核会话可以在历史清理或账号治理后失去关联，
    -- 因此使用ON DELETE SET NULL；正式反馈内容仍由本表快照保留。
    ai_review_session_id UUID NULL
        REFERENCES courseware_ai_review_sessions(id) ON DELETE SET NULL,

    review_level SMALLINT NOT NULL
        CHECK (review_level IN (1, 2)),

    review_round INTEGER NOT NULL
        CHECK (review_round > 0),

    decision VARCHAR(20) NOT NULL
        CHECK (decision IN ('approved', 'revision')),

    overall_risk VARCHAR(20) NOT NULL DEFAULT 'info'
        CHECK (
            overall_risk IN (
                'critical',
                'high',
                'medium',
                'low',
                'info'
            )
        ),

    -- 以下字段是正式提交审核决定时的不可变快照。
    overall_summary TEXT NOT NULL DEFAULT '',
    strengths_json JSONB NOT NULL DEFAULT '[]'::jsonb
        CHECK (jsonb_typeof(strengths_json) = 'array'),
    obvious_problems_json JSONB NOT NULL DEFAULT '[]'::jsonb
        CHECK (jsonb_typeof(obvious_problems_json) = 'array'),
    review_comment_snapshot TEXT NOT NULL DEFAULT '',

    created_by UUID NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_courseware_review_feedback_review
        UNIQUE (courseware_review_id)
);

CREATE INDEX IF NOT EXISTS idx_cw_review_feedback_courseware
    ON courseware_review_feedback(
        courseware_id,
        created_at DESC
    );

CREATE INDEX IF NOT EXISTS idx_cw_review_feedback_session
    ON courseware_review_feedback(ai_review_session_id)
    WHERE ai_review_session_id IS NOT NULL;

-- ============================================================================
-- 二、页级整改项
-- ============================================================================

CREATE TABLE IF NOT EXISTS courseware_review_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    courseware_id UUID NOT NULL
        REFERENCES coursewares(id) ON DELETE CASCADE,

    -- 每条整改项必须追溯到产生它的AI审核会话和finding。
    source_session_id UUID NOT NULL
        REFERENCES courseware_ai_review_sessions(id) ON DELETE CASCADE,

    source_finding_id VARCHAR(128) NOT NULL,

    -- 正式审核提交前这两个字段为空；
    -- 审核员正式提交审核决定时，在同一事务内绑定审核记录和反馈快照。
    courseware_review_id UUID NULL
        REFERENCES courseware_reviews(id) ON DELETE CASCADE,

    feedback_id UUID NULL
        REFERENCES courseware_review_feedback(id) ON DELETE CASCADE,

    source_type VARCHAR(16) NOT NULL
        CHECK (source_type IN ('self', 'formal')),

    -- 0=作者自审；1=L1正式审核；2=L2正式审核。
    review_level SMALLINT NOT NULL,

    -- 自审没有正式审核轮次，固定为0；
    -- 正式审核绑定后写入大于0的实际轮次。
    review_round INTEGER NOT NULL DEFAULT 0
        CHECK (review_round >= 0),

    -- created_by是创建整改项的作者或审核员。
    created_by UUID NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    -- owner_id是最终负责修改课件的课件作者。
    owner_id UUID NOT NULL
        REFERENCES users(id) ON DELETE RESTRICT,

    -- page_id是稳定页面定位依据。
    -- 页面被删除后保留整改历史并将page_id置空。
    page_id UUID NULL
        REFERENCES courseware_pages(id) ON DELETE SET NULL,

    -- 审核时的页码、标题、HTML哈希和更新时间快照。
    -- 页码为0表示整课全局问题，没有单独页面。
    page_number_snapshot INTEGER NOT NULL DEFAULT 0
        CHECK (page_number_snapshot >= 0),

    page_title_snapshot TEXT NOT NULL DEFAULT '',
    page_html_hash VARCHAR(64) NOT NULL DEFAULT '',
    page_updated_at_snapshot TIMESTAMPTZ NULL,

    severity VARCHAR(20) NOT NULL DEFAULT 'medium'
        CHECK (
            severity IN (
                'critical',
                'high',
                'medium',
                'low',
                'info'
            )
        ),

    dimension VARCHAR(64) NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',

    -- 保存教案/大纲依据、页面证据、代码证据、连续性证据、
    -- 置信度以及人工操作复核标记。
    evidence_json JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(evidence_json) = 'object'),

    original_suggestion TEXT NOT NULL DEFAULT '',
    confirmed_instruction TEXT NOT NULL DEFAULT '',

    status VARCHAR(32) NOT NULL DEFAULT 'detected'
        CHECK (
            status IN (
                'detected',
                'discussing',
                'confirmed',
                'applying',
                'applied',
                'resolved',
                'dismissed',
                'stale',
                'orphaned'
            )
        ),

    -- 成功微调后页面的新HTML哈希，用于后续核验修改是否仍然存在。
    applied_page_hash VARCHAR(64) NOT NULL DEFAULT '',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ NULL,
    applied_at TIMESTAMPTZ NULL,
    resolved_at TIMESTAMPTZ NULL,

    CONSTRAINT chk_cw_review_item_source_level
        CHECK (
            (
                source_type = 'self'
                AND review_level = 0
                AND review_round = 0
                AND courseware_review_id IS NULL
                AND feedback_id IS NULL
            )
            OR
            (
                source_type = 'formal'
                AND review_level IN (1, 2)
            )
        ),

    CONSTRAINT chk_cw_review_item_feedback_pair
        CHECK (
            (
                courseware_review_id IS NULL
                AND feedback_id IS NULL
            )
            OR
            (
                courseware_review_id IS NOT NULL
                AND feedback_id IS NOT NULL
            )
        )
);

-- 同一AI会话、同一finding、同一页面只生成一条整改项。
-- 没有页面的整课问题统一使用零UUID参与唯一性判断。
CREATE UNIQUE INDEX IF NOT EXISTS uq_cw_review_item_source_page
    ON courseware_review_items(
        source_session_id,
        source_finding_id,
        COALESCE(
            page_id,
            '00000000-0000-0000-0000-000000000000'::uuid
        )
    );

CREATE INDEX IF NOT EXISTS idx_cw_review_item_owner
    ON courseware_review_items(
        owner_id,
        courseware_id,
        status,
        updated_at DESC
    );

CREATE INDEX IF NOT EXISTS idx_cw_review_item_creator_session
    ON courseware_review_items(
        created_by,
        source_session_id,
        created_at
    );

CREATE INDEX IF NOT EXISTS idx_cw_review_item_feedback
    ON courseware_review_items(
        feedback_id,
        page_number_snapshot,
        created_at
    )
    WHERE feedback_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_cw_review_item_page
    ON courseware_review_items(
        page_id,
        status
    )
    WHERE page_id IS NOT NULL;

-- ============================================================================
-- 三、现有AI审核消息关联具体整改项
-- ============================================================================

ALTER TABLE courseware_ai_review_messages
    ADD COLUMN IF NOT EXISTS review_item_id UUID NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
              'courseware_ai_review_messages'::regclass
          AND conname =
              'fk_cw_ai_review_message_item'
    ) THEN
        ALTER TABLE courseware_ai_review_messages
            ADD CONSTRAINT fk_cw_ai_review_message_item
            FOREIGN KEY (review_item_id)
            REFERENCES courseware_review_items(id)
            ON DELETE CASCADE;
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_cw_ai_review_message_item
    ON courseware_ai_review_messages(
        review_item_id,
        created_at
    )
    WHERE review_item_id IS NOT NULL;

-- ============================================================================
-- 四、数据库注释
-- ============================================================================

COMMENT ON TABLE courseware_review_feedback IS
    '正式课件审核整体反馈快照：保存提交决定时的整课评价、明显问题和人工意见';

COMMENT ON TABLE courseware_review_items IS
    '课件AI审核整改项：按稳定页面ID管理问题讨论、确认、应用和解决状态';

COMMENT ON COLUMN courseware_review_items.page_id IS
    '稳定页面定位；页面删除后置空，历史页码和标题快照继续保留';

COMMENT ON COLUMN courseware_review_items.page_number_snapshot IS
    '审核发生时的页码，仅用于历史展示，不作为后续修改的唯一定位依据';

COMMENT ON COLUMN courseware_review_items.page_html_hash IS
    '审核发生时的页面HTML哈希；注入微调前必须与当前页面重新比较';

COMMENT ON COLUMN courseware_review_items.confirmed_instruction IS
    '用户通过独立确认动作确定的最终修改指令，聊天中的自然语言确认不自动写入';

COMMENT ON COLUMN courseware_review_items.status IS
    'detected→discussing→confirmed→applying→applied→resolved；dismissed/stale/orphaned为终止状态';

COMMENT ON COLUMN courseware_ai_review_messages.review_item_id IS
    '单条整改项讨论线程；NULL表示兼容历史会话级消息';

COMMIT;
