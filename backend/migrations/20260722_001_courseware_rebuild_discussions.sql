-- 课件全页重构讨论会话
--
-- 目标：
-- 1. AI讨论阶段只保存老师可见的正式交流内容，不保存隐藏推理过程。
-- 2. 只有老师显式确认后，后端才允许进入既有全页重构与页面写回链路。
-- 3. 使用页面updated_at快照识别讨论期间的页面变化，避免基于旧页面执行方案。
-- 4. 同一用户对同一页面同一时间只保留一个活动讨论，防止并发确认和重复执行。

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS courseware_rebuild_discussions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    courseware_id UUID NOT NULL
        REFERENCES coursewares(id) ON DELETE CASCADE,
    page_id UUID NOT NULL,
    page_number INTEGER NOT NULL
        CHECK (page_number > 0),
    created_by UUID NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    status VARCHAR(32) NOT NULL DEFAULT 'discussing'
        CHECK (
            status IN (
                'discussing',
                'awaiting_confirmation',
                'executing',
                'completed',
                'cancelled',
                'stale'
            )
        ),

    base_page_updated_at TIMESTAMPTZ NOT NULL,

    -- 老师选择的代码收藏、模板页及本课前页引用。
    -- 这些内容只作为讨论和最终执行的受控参考，
    -- 最终执行时仍由既有重构服务重新解析并完成权限校验。
    reference_context TEXT NOT NULL DEFAULT '',

    -- 仅保存可向老师展示的正式消息：
    -- [{"role":"teacher|assistant","content":"...","created_at":"..."}]
    messages JSONB NOT NULL DEFAULT '[]'::jsonb
        CHECK (jsonb_typeof(messages) = 'array'),

    -- AI在讨论成熟后生成的教师可确认执行说明，不包含HTML代码。
    final_instruction TEXT NOT NULL DEFAULT '',
    ai_summary TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',

    confirmed_at TIMESTAMPTZ,
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_courseware_rebuild_discussion_page
        FOREIGN KEY (page_id, courseware_id)
        REFERENCES courseware_pages(id, courseware_id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_courseware_rebuild_discussion_active
    ON courseware_rebuild_discussions (
        courseware_id,
        page_id,
        created_by
    )
    WHERE status IN (
        'discussing',
        'awaiting_confirmation',
        'executing'
    );

CREATE INDEX IF NOT EXISTS idx_courseware_rebuild_discussion_page
    ON courseware_rebuild_discussions (
        courseware_id,
        page_id,
        created_by,
        updated_at DESC
    );

CREATE INDEX IF NOT EXISTS idx_courseware_rebuild_discussion_status
    ON courseware_rebuild_discussions (
        status,
        updated_at DESC
    );

COMMENT ON TABLE courseware_rebuild_discussions IS
    '课件全页重构的教师可见讨论会话；只有显式确认后才能执行页面重构';

COMMENT ON COLUMN courseware_rebuild_discussions.messages IS
    '教师与AI的正式可见消息，不保存模型隐藏推理过程';

COMMENT ON COLUMN courseware_rebuild_discussions.base_page_updated_at IS
    '讨论建立时的页面更新时间快照，用于确认前识别页面是否已变化';

COMMIT;
