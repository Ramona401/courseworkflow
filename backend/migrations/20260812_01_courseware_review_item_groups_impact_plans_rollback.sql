-- ============================================================================
-- TE-DNA 2.0：R-06正式问题组 + R-07影响方案数据库回滚
-- 文件：20260812_01_courseware_review_item_groups_impact_plans_rollback.sql
-- ----------------------------------------------------------------------------
-- 警告：
--   1. 本文件会删除全部R-06问题组、成员、组事件、R-07影响方案与方案事件；
--   2. 只允许在确认需要回退本次功能时执行；
--   3. 已产生正式业务数据后，优先从完整数据库备份恢复；
--   4. 执行本文件前必须再次完整备份数据库。
-- ============================================================================

BEGIN;

DROP TRIGGER IF EXISTS trg_cw_review_impact_plan_event_consistency
ON courseware_review_impact_plans;

DROP TRIGGER IF EXISTS trg_cw_review_impact_plan_event_insert_guard
ON courseware_review_impact_plan_events;

DROP TRIGGER IF EXISTS trg_cw_review_impact_plan_mutation_guard
ON courseware_review_impact_plans;

DROP TABLE IF EXISTS courseware_review_impact_plan_events;
DROP TABLE IF EXISTS courseware_review_impact_plans;

DROP FUNCTION IF EXISTS
    public.enforce_cw_review_impact_plan_event_consistency();

DROP FUNCTION IF EXISTS
    public.guard_cw_review_impact_plan_event_insert();

DROP FUNCTION IF EXISTS
    public.guard_cw_review_impact_plan_mutation();

DROP FUNCTION IF EXISTS
    public.normalize_cw_review_impact_selection(JSONB, JSONB);

DROP FUNCTION IF EXISTS
    public.is_valid_cw_review_impact_selection(JSONB, JSONB);

DROP FUNCTION IF EXISTS
    public.is_valid_cw_review_impact_operations(JSONB);

DROP FUNCTION IF EXISTS
    public.build_cw_review_impact_operations_hash(JSONB);

DROP FUNCTION IF EXISTS
    public.build_cw_review_impact_message_hash(TEXT, JSONB);

DROP TRIGGER IF EXISTS trg_cw_review_item_group_membership_consistency
ON courseware_review_item_groups;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_member_event_consistency
ON courseware_review_item_group_members;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_event_consistency
ON courseware_review_item_groups;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_event_insert_guard
ON courseware_review_item_group_events;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_member_mutation_guard
ON courseware_review_item_group_members;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_mutation_guard
ON courseware_review_item_groups;

DROP TABLE IF EXISTS courseware_review_item_group_events;
DROP TABLE IF EXISTS courseware_review_item_group_members;
DROP TABLE IF EXISTS courseware_review_item_groups;

DROP FUNCTION IF EXISTS
    public.enforce_cw_review_item_group_membership_consistency();

DROP FUNCTION IF EXISTS
    public.enforce_cw_review_item_group_member_event_consistency();

DROP FUNCTION IF EXISTS
    public.enforce_cw_review_item_group_event_consistency();

DROP FUNCTION IF EXISTS
    public.guard_cw_review_item_group_event_insert();

DROP FUNCTION IF EXISTS
    public.guard_cw_review_item_group_member_mutation();

DROP FUNCTION IF EXISTS
    public.guard_cw_review_item_group_mutation();

ALTER TABLE courseware_ai_review_sessions
    DROP CONSTRAINT IF EXISTS
        uq_cw_ai_review_session_id_courseware_reviewer;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE 'R-06问题组与R-07影响方案数据库结构已回滚';
END
$$;

