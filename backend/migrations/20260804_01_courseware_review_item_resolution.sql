-- 20260804_01_courseware_review_item_resolution.sql
--
-- V1.3 课件整改、重新提交与复审确认闭环。
--
-- 本迁移补齐两类事实：
--
--   一、作者已经完成页面修改，并在之后重新提交审核：
--       resubmitted_at
--       resubmitted_review_level
--       resubmitted_review_round
--
--   二、问题最终由谁、在哪次正式复审中确认解决：
--       resolved_by
--       resolved_review_id
--       resolved_review_level
--       resolved_review_round
--       resolution_note
--
-- 状态含义：
--
--   confirmed
--     已形成可执行的整改要求或修改方案。
--
--   applying
--     作者正在修改页面。
--
--   applied
--     页面修改已经完成，且当前页面仍与完成修改时一致。
--     正式审核问题等待审核员复审；
--     作者自审问题等待作者本人检查效果并明确确认。
--
--   stale
--     页面在完成该问题对应修改后又发生变化，
--     不能继续把旧的修改完成记录当作当前页面结论。
--
--   orphaned
--     原问题绑定的页面已经删除。
--
--   resolved
--     已由具备相应权限的人明确确认解决。
--     页面写入成功本身不再自动等于问题解决。
--
-- 历史数据纠正：
--
--   旧版本没有人工确认解决入口，历史resolved记录都缺少确认人。
--   本迁移使用与后端cwAIReviewHash完全一致的算法：
--
--       SHA-256（页面HTML原文UTF-8字节）
--
--   将历史记录恢复到最接近当前真实情况的阶段：
--
--     原页面已删除：
--       orphaned
--
--     有完整修改证据，但当前页面内容指纹与完成修改时不同：
--       stale
--
--     有完整修改证据，且当前页面仍保持不变：
--       applied
--
--     整课问题没有唯一页面，但有完整修改证据：
--       applied，等待整课人工检查
--
--     没有完整修改证据，但有确认整改要求：
--       confirmed
--
--     其余异常历史记录：
--       detected
--
-- 本迁移不删除审核反馈、不覆盖整改要求，也不改变审核历史快照。

BEGIN;

-- 历史内容指纹修正和后续数据库校验需要digest函数。
--
-- 当前生产库已经安装pgcrypto；IF NOT EXISTS保证其他环境重复执行安全。
-- 回滚V1.3结构时不会删除该共享扩展。
CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE courseware_review_items
    ADD COLUMN IF NOT EXISTS resubmitted_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS resubmitted_review_level SMALLINT
        NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS resubmitted_review_round INTEGER
        NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS resolved_by UUID NULL,
    ADD COLUMN IF NOT EXISTS resolved_review_id UUID NULL,
    ADD COLUMN IF NOT EXISTS resolved_review_level SMALLINT
        NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS resolved_review_round INTEGER
        NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS resolution_note TEXT
        NOT NULL DEFAULT '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
              'courseware_review_items'::regclass
          AND conname =
              'fk_cw_review_item_resolved_by'
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT fk_cw_review_item_resolved_by
            FOREIGN KEY (resolved_by)
            REFERENCES users(id)
            ON DELETE RESTRICT;
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
              'fk_cw_review_item_resolution_review'
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT fk_cw_review_item_resolution_review
            FOREIGN KEY (resolved_review_id)
            REFERENCES courseware_reviews(id)
            ON DELETE RESTRICT;
    END IF;
END
$$;

-- 先计算历史问题绑定页面的当前真实状态。
--
-- current_page_id为空有两种情况：
--
--   - item.page_id为空：整课问题；
--   - item.page_id非空：原页面已经删除。
--
-- current_page_hash与后端cwAIReviewHash完全一致：
--
--   sha256.Sum256([]byte(html_content))
--
-- PostgreSQL中对应：
--
--   digest(convert_to(html_content, 'UTF8'), 'sha256')
WITH historical_resolution AS (
    SELECT
        item.id,
        item.page_id,
        page.id AS current_page_id,
        CASE
            WHEN page.id IS NULL
                THEN ''
            ELSE encode(
                digest(
                    convert_to(
                        COALESCE(
                            page.html_content,
                            ''
                        ),
                        'UTF8'
                    ),
                    'sha256'
                ),
                'hex'
            )
        END AS current_page_hash
    FROM courseware_review_items AS item
    LEFT JOIN courseware_pages AS page
           ON page.id = item.page_id
          AND page.courseware_id =
              item.courseware_id
    WHERE item.status = 'resolved'
      AND item.resolved_by IS NULL
)
UPDATE courseware_review_items AS item
SET
    status = CASE
        -- 有稳定page_id但当前页面不存在，恢复为页面已删除。
        WHEN item.page_id IS NOT NULL
         AND history.current_page_id IS NULL
            THEN 'orphaned'

        -- 页面修改曾经完成，但当前HTML已经再次变化。
        WHEN item.page_id IS NOT NULL
         AND item.applied_at IS NOT NULL
         AND BTRIM(
             COALESCE(
                 item.applied_page_hash,
                 ''
             )
         ) <> ''
         AND history.current_page_hash <>
             BTRIM(
                 COALESCE(
                     item.applied_page_hash,
                     ''
                 )
             )
            THEN 'stale'

        -- 当前页面仍与完成修改时一致。
        --
        -- 整课问题没有page_id，只要有完整修改证据，
        -- 同样恢复为applied，等待作者进行整课人工检查。
        WHEN item.applied_at IS NOT NULL
         AND BTRIM(
             COALESCE(
                 item.applied_page_hash,
                 ''
             )
         ) <> ''
            THEN 'applied'

        -- 没有完整修改证据，但已经形成确认指令。
        WHEN BTRIM(
             COALESCE(
                 item.confirmed_instruction,
                 ''
             )
         ) <> ''
            THEN 'confirmed'

        ELSE 'detected'
    END,

    -- stale和orphaned保留历史修改证据，方便以后回看。
    --
    -- confirmed和detected没有完整修改证据，清理残缺字段。
    applied_at = CASE
        WHEN item.applied_at IS NOT NULL
         AND BTRIM(
             COALESCE(
                 item.applied_page_hash,
                 ''
             )
         ) <> ''
            THEN item.applied_at
        ELSE NULL
    END,

    applied_page_hash = CASE
        WHEN item.applied_at IS NOT NULL
         AND BTRIM(
             COALESCE(
                 item.applied_page_hash,
                 ''
             )
         ) <> ''
            THEN item.applied_page_hash
        ELSE ''
    END,

    resolved_at = NULL,
    resolved_by = NULL,
    resolved_review_id = NULL,
    resolved_review_level = 0,
    resolved_review_round = 0,
    resolution_note = '',
    updated_at = NOW()
FROM historical_resolution AS history
WHERE item.id = history.id;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid =
              'courseware_review_items'::regclass
          AND conname =
              'chk_cw_review_item_resubmission'
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT chk_cw_review_item_resubmission
            CHECK (
                (
                    resubmitted_at IS NULL
                    AND resubmitted_review_level = 0
                    AND resubmitted_review_round = 0
                )
                OR
                (
                    source_type = 'formal'
                    AND resubmitted_at IS NOT NULL
                    AND resubmitted_review_level IN (1, 2)
                    AND resubmitted_review_round > 0
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
              'chk_cw_review_item_resolution'
    ) THEN
        ALTER TABLE courseware_review_items
            ADD CONSTRAINT chk_cw_review_item_resolution
            CHECK (
                (
                    status <> 'resolved'
                    AND resolved_at IS NULL
                    AND resolved_by IS NULL
                    AND resolved_review_id IS NULL
                    AND resolved_review_level = 0
                    AND resolved_review_round = 0
                    AND BTRIM(resolution_note) = ''
                )
                OR
                (
                    status = 'resolved'
                    AND resolved_at IS NOT NULL
                    AND resolved_by IS NOT NULL
                    AND (
                        (
                            source_type = 'self'
                            AND resolved_review_id IS NULL
                            AND resolved_review_level = 0
                            AND resolved_review_round = 0
                        )
                        OR
                        (
                            source_type = 'formal'
                            AND resolved_review_id IS NOT NULL
                            AND resolved_review_level IN (1, 2)
                            AND resolved_review_round > 0
                        )
                    )
                )
            );
    END IF;
END
$$;

CREATE INDEX IF NOT EXISTS idx_cw_review_item_resubmitted
    ON courseware_review_items(
        courseware_id,
        resubmitted_review_level,
        resubmitted_review_round,
        status,
        updated_at DESC
    )
    WHERE resubmitted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_cw_review_item_resolution_review
    ON courseware_review_items(
        resolved_review_id,
        resolved_review_round
    )
    WHERE resolved_review_id IS NOT NULL;

COMMENT ON COLUMN courseware_review_items.resubmitted_at IS
    '作者完成正式整改后最近一次重新提交审核的时间';

COMMENT ON COLUMN courseware_review_items.resubmitted_review_level IS
    '最近一次重新提交后准备进入的审核级别；未重新提交为0';

COMMENT ON COLUMN courseware_review_items.resubmitted_review_round IS
    '最近一次重新提交预计进入的该级审核轮次；未重新提交为0';

COMMENT ON COLUMN courseware_review_items.resolved_by IS
    '明确确认问题已经解决的用户；页面修改成功不会自动填写';

COMMENT ON COLUMN courseware_review_items.resolved_review_id IS
    '正式问题确认解决时对应的courseware_reviews记录；自审问题为空';

COMMENT ON COLUMN courseware_review_items.resolved_review_level IS
    '正式复审确认解决时的审核级别；作者自审确认为0';

COMMENT ON COLUMN courseware_review_items.resolved_review_round IS
    '正式复审确认解决时的审核轮次；作者自审确认为0';

COMMENT ON COLUMN courseware_review_items.resolution_note IS
    '确认问题已经解决时保存的人工说明';

COMMIT;
