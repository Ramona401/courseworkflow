-- 20260815_01_courseware_review_comment_candidates_rollback.sql
--
-- R-08审核意见候选结构回滚。
--
-- 注意：
--   - 本文件只回滚R-08新增结构；
--   - 不触碰courseware_reviews、courseware_review_feedback、
--     courseware_review_items、R-06或R-07已有数据；
--   - DROP TABLE会同时移除表级trigger，因此无需先逐条DELETE。

BEGIN;

DROP TABLE IF EXISTS courseware_review_comment_candidates;

DROP FUNCTION IF EXISTS
        guard_courseware_review_comment_candidate_immutable();

COMMIT;
