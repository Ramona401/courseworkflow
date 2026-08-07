-- 20260723_03_courseware_ai_self_review.sql
--
-- 课件AI分析会话统一使用 review_level 区分用途：
--   0 = 作者课件自审；
--   1 = L1教研组正式审核辅助；
--   2 = L2学校正式审核辅助。
--
-- 自审和正式审核继续共用：
--   - 全页面静态索引；
--   - HTML/CSS/JavaScript互动证据；
--   - 顺序分批；
--   - 跨批次连续性账本；
--   - 风险回看和最终报告。
--
-- 自审不会写入courseware_reviews，也不会修改课件publish_state或review_level。
--
-- 存量数据库可能没有review_level检查约束，也可能已有只允许1/2的约束。
-- 本迁移只处理courseware_ai_review_sessions表中明确引用review_level的检查约束，
-- 随后建立统一的0/1/2检查约束。

BEGIN;

DO $$
DECLARE
    item RECORD;
BEGIN
    FOR item IN
        SELECT conname
        FROM pg_constraint
        WHERE conrelid =
              'courseware_ai_review_sessions'::regclass
          AND contype = 'c'
          AND pg_get_constraintdef(oid)
              ILIKE '%review_level%'
    LOOP
        EXECUTE format(
            'ALTER TABLE courseware_ai_review_sessions DROP CONSTRAINT %I',
            item.conname
        );
    END LOOP;
END
$$;

ALTER TABLE courseware_ai_review_sessions
    ADD CONSTRAINT chk_courseware_ai_review_sessions_level
    CHECK (review_level IN (0, 1, 2));

COMMENT ON COLUMN
    courseware_ai_review_sessions.review_level
IS
    'AI分析用途：0=作者课件自审，1=L1正式审核辅助，2=L2正式审核辅助';

COMMIT;
