-- ============================================================================
-- TE-DNA 2.0：R-06 正式问题组数据库守卫 V1
-- 文件：20260812_02_courseware_review_item_group_guards.sql
-- ----------------------------------------------------------------------------
-- 本文件必须与20260812_01_courseware_review_item_groups.sql在同一psql连接、
-- 同一事务中连续执行；SAVEPOINT用于阻止脱离事务的误执行。
-- 本文件完成组、成员、事件的不可变守卫、延迟一致性检查和最小权限。
-- 本文件仍不COMMIT，必须继续执行R-07影响方案迁移统一提交。
-- ============================================================================

SAVEPOINT r06_group_structure_ready;

-- 一、R-06 数据库守卫
-- ============================================================================

CREATE OR REPLACE FUNCTION public.guard_cw_review_item_group_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'active'
           OR NEW.version <> 1
           OR NEW.merged_into_group_id IS NOT NULL THEN
            RAISE EXCEPTION
                '新建问题组必须是active、version=1且不能预设合并目标';
        END IF;

        RETURN NEW;
    END IF;

    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.courseware_id IS DISTINCT FROM OLD.courseware_id
       OR NEW.source_session_id IS DISTINCT FROM OLD.source_session_id
       OR NEW.created_by IS DISTINCT FROM OLD.created_by
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION
            '问题组身份、课件、会话、创建者和创建时间不可修改';
    END IF;

    IF OLD.status = 'merged' THEN
        RAISE EXCEPTION '已合并问题组不可继续修改';
    END IF;

    IF NEW.version <> OLD.version + 1 THEN
        RAISE EXCEPTION '问题组更新必须将version精确加1';
    END IF;

    IF OLD.status = 'active'
       AND NEW.status NOT IN ('active', 'merged') THEN
        RAISE EXCEPTION '问题组状态迁移无效';
    END IF;

    IF NEW.status = 'merged'
       AND NEW.name IS DISTINCT FROM OLD.name THEN
        RAISE EXCEPTION '合并问题组时不能同时重命名';
    END IF;

    IF NEW.status = 'active'
       AND NEW.name IS DISTINCT FROM OLD.name
       AND NEW.primary_item_id IS DISTINCT FROM OLD.primary_item_id THEN
        RAISE EXCEPTION
            '一次问题组版本只能执行重命名或主问题变更中的一种';
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION public.guard_cw_review_item_group_mutation()
FROM PUBLIC;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_mutation_guard
ON courseware_review_item_groups;

CREATE TRIGGER trg_cw_review_item_group_mutation_guard
BEFORE INSERT OR UPDATE
ON courseware_review_item_groups
FOR EACH ROW
EXECUTE FUNCTION public.guard_cw_review_item_group_mutation();

CREATE OR REPLACE FUNCTION public.guard_cw_review_item_group_member_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    group_status VARCHAR(16);
    group_creator UUID;

    item_source_type VARCHAR(16);
    item_created_by UUID;
    item_owner_id UUID;
    item_status VARCHAR(32);
    item_delivered BOOLEAN;
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'active'
           OR NEW.version <> 1
           OR NEW.removed_by IS NOT NULL
           OR NEW.removed_at IS NOT NULL THEN
            RAISE EXCEPTION
                '新建问题组成员必须是active且version=1';
        END IF;
    ELSE
        IF NEW.id IS DISTINCT FROM OLD.id
           OR NEW.courseware_id IS DISTINCT FROM OLD.courseware_id
           OR NEW.source_session_id IS DISTINCT FROM OLD.source_session_id
           OR NEW.item_id IS DISTINCT FROM OLD.item_id
           OR NEW.created_by IS DISTINCT FROM OLD.created_by
           OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
            RAISE EXCEPTION
                '问题组成员身份、整改项、范围和创建身份不可修改';
        END IF;

        IF NEW.version <> OLD.version + 1 THEN
            RAISE EXCEPTION
                '问题组成员更新必须将version精确加1';
        END IF;

        IF OLD.status = 'active' THEN
            PERFORM 1
            FROM public.courseware_review_item_groups AS source_group
            WHERE source_group.id = OLD.group_id
              AND source_group.courseware_id = OLD.courseware_id
              AND source_group.source_session_id = OLD.source_session_id
              AND source_group.created_by = OLD.created_by
              AND source_group.status = 'active'
            FOR UPDATE;

            IF NOT FOUND THEN
                RAISE EXCEPTION
                    '成员来源问题组不存在、越界或已经合并';
            END IF;
        END IF;

        IF OLD.status = 'active'
           AND NEW.status = 'active'
           AND NEW.group_id = OLD.group_id THEN
            RAISE EXCEPTION
                'active成员更新必须实际移动到其他问题组';
        END IF;

        IF OLD.status = 'active'
           AND NEW.status = 'removed'
           AND NEW.group_id <> OLD.group_id THEN
            RAISE EXCEPTION
                '移除成员时不能同时改变问题组';
        END IF;

        IF OLD.status = 'removed'
           AND NEW.status <> 'active' THEN
            RAISE EXCEPTION
                '已移除成员只能通过明确恢复重新加入问题组';
        END IF;
    END IF;

    SELECT
        target_group.status,
        target_group.created_by
    INTO
        group_status,
        group_creator
    FROM public.courseware_review_item_groups AS target_group
    WHERE target_group.id = NEW.group_id
      AND target_group.courseware_id = NEW.courseware_id
      AND target_group.source_session_id = NEW.source_session_id
      AND target_group.created_by = NEW.created_by
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION '目标问题组不存在或超出当前治理范围';
    END IF;

    IF NEW.status = 'active'
       AND group_status <> 'active' THEN
        RAISE EXCEPTION '不能把有效成员放入已合并问题组';
    END IF;

    SELECT
        item.source_type,
        item.created_by,
        item.owner_id,
        item.status,
        (
            item.courseware_review_id IS NOT NULL
            OR item.feedback_id IS NOT NULL
        )
    INTO
        item_source_type,
        item_created_by,
        item_owner_id,
        item_status,
        item_delivered
    FROM public.courseware_review_items AS item
    WHERE item.id = NEW.item_id
      AND item.courseware_id = NEW.courseware_id
      AND item.source_session_id = NEW.source_session_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION '问题组成员整改项不存在或范围不一致';
    END IF;

    IF item_delivered THEN
        RAISE EXCEPTION '已正式交付的整改项不能继续变更问题组成员关系';
    END IF;

    IF item_status NOT IN ('detected', 'discussing', 'confirmed') THEN
        RAISE EXCEPTION '当前整改项状态不允许变更问题组成员关系';
    END IF;

    IF (
        item_source_type = 'formal'
        AND item_created_by <> group_creator
    )
    OR (
        item_source_type = 'self'
        AND item_owner_id <> group_creator
    )
    OR item_source_type NOT IN ('formal', 'self') THEN
        RAISE EXCEPTION '问题组成员操作者与整改项治理身份不一致';
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION public.guard_cw_review_item_group_member_mutation()
FROM PUBLIC;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_member_mutation_guard
ON courseware_review_item_group_members;

CREATE TRIGGER trg_cw_review_item_group_member_mutation_guard
BEFORE INSERT OR UPDATE
ON courseware_review_item_group_members
FOR EACH ROW
EXECUTE FUNCTION public.guard_cw_review_item_group_member_mutation();

CREATE OR REPLACE FUNCTION public.guard_cw_review_item_group_event_insert()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    current_status VARCHAR(16);
    current_version INTEGER;
BEGIN
    SELECT
        review_group.status,
        review_group.version
    INTO
        current_status,
        current_version
    FROM public.courseware_review_item_groups AS review_group
    WHERE review_group.id = NEW.group_id
      AND review_group.courseware_id = NEW.courseware_id
      AND review_group.source_session_id = NEW.source_session_id
      AND review_group.created_by = NEW.actor_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION '问题组事件对应的问题组不存在或越界';
    END IF;

    IF NEW.group_version <> current_version THEN
        RAISE EXCEPTION '问题组事件版本与问题组当前版本不一致';
    END IF;

    IF current_version = 1
       AND NEW.event_type <> 'created' THEN
        RAISE EXCEPTION '问题组version=1必须写created事件';
    END IF;

    IF current_version > 1
       AND NEW.event_type = 'created' THEN
        RAISE EXCEPTION 'created事件只能用于问题组version=1';
    END IF;

    IF current_status = 'merged'
       AND NEW.event_type <> 'merged' THEN
        RAISE EXCEPTION '问题组进入merged状态时必须写merged事件';
    END IF;

    IF NEW.member_id IS NOT NULL THEN
        PERFORM 1
        FROM public.courseware_review_item_group_members AS member
        WHERE member.id = NEW.member_id
          AND member.courseware_id = NEW.courseware_id
          AND member.source_session_id = NEW.source_session_id
          AND member.created_by = NEW.actor_id
          AND member.version = NEW.member_version
          AND (
              NEW.event_type <> 'member_added'
              OR (
                  member.status = 'active'
                  AND member.group_id = NEW.group_id
              )
          )
          AND (
              NEW.event_type <> 'member_removed'
              OR (
                  member.status = 'removed'
                  AND member.group_id = NEW.group_id
              )
          )
          AND (
              NEW.event_type <> 'member_moved'
              OR (
                  member.status = 'active'
                  AND (
                      member.group_id = NEW.group_id
                      OR member.group_id = NEW.related_group_id
                  )
              )
          );

        IF NOT FOUND THEN
            RAISE EXCEPTION
                '问题组成员事件与成员当前范围、状态或版本不一致';
        END IF;
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION public.guard_cw_review_item_group_event_insert()
FROM PUBLIC;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_event_insert_guard
ON courseware_review_item_group_events;

CREATE TRIGGER trg_cw_review_item_group_event_insert_guard
BEFORE INSERT
ON courseware_review_item_group_events
FOR EACH ROW
EXECUTE FUNCTION public.guard_cw_review_item_group_event_insert();

CREATE OR REPLACE FUNCTION public.enforce_cw_review_item_group_event_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    matching_event_count BIGINT;
    matching_event_type VARCHAR(32);
BEGIN
    SELECT
        COUNT(*),
        MIN(event.event_type)
    INTO
        matching_event_count,
        matching_event_type
    FROM public.courseware_review_item_group_events AS event
    WHERE event.group_id = NEW.id
      AND event.source_session_id = NEW.source_session_id
      AND event.group_version = NEW.version;

    IF matching_event_count <> 1 THEN
        RAISE EXCEPTION
            '问题组version=%缺少唯一匹配事件',
            NEW.version;
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF matching_event_type <> 'created' THEN
            RAISE EXCEPTION '新建问题组必须对应created事件';
        END IF;

        RETURN NEW;
    END IF;

    IF OLD.status = 'active'
       AND NEW.status = 'merged' THEN
        IF matching_event_type <> 'merged' THEN
            RAISE EXCEPTION '问题组合并必须对应merged事件';
        END IF;
    ELSIF NEW.name IS DISTINCT FROM OLD.name THEN
        IF matching_event_type <> 'renamed' THEN
            RAISE EXCEPTION '问题组重命名必须对应renamed事件';
        END IF;
    ELSIF NEW.primary_item_id IS DISTINCT FROM OLD.primary_item_id THEN
        IF matching_event_type <> 'primary_changed' THEN
            RAISE EXCEPTION '问题组主问题变更必须对应primary_changed事件';
        END IF;
    ELSIF matching_event_type NOT IN (
        'member_added',
        'member_removed',
        'member_moved',
        'merged',
        'split'
    ) THEN
        RAISE EXCEPTION
            '问题组无字段变化版本只能记录成员、合并目标或拆分事件';
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION public.enforce_cw_review_item_group_event_consistency()
FROM PUBLIC;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_event_consistency
ON courseware_review_item_groups;

CREATE CONSTRAINT TRIGGER trg_cw_review_item_group_event_consistency
AFTER INSERT OR UPDATE
ON courseware_review_item_groups
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION public.enforce_cw_review_item_group_event_consistency();

CREATE OR REPLACE FUNCTION public.enforce_cw_review_item_group_member_event_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    matching_event_count BIGINT;
BEGIN
    IF TG_OP = 'INSERT' THEN
        SELECT COUNT(*)
        INTO matching_event_count
        FROM public.courseware_review_item_group_events AS event
        WHERE event.group_id = NEW.group_id
          AND event.member_id = NEW.id
          AND event.member_version = NEW.version
          AND event.event_type = 'member_added';

        IF matching_event_count <> 1 THEN
            RAISE EXCEPTION
                '新建问题组成员缺少唯一member_added事件';
        END IF;

        RETURN NEW;
    END IF;

    IF OLD.status = 'active'
       AND NEW.status = 'removed' THEN
        SELECT COUNT(*)
        INTO matching_event_count
        FROM public.courseware_review_item_group_events AS event
        WHERE event.group_id = OLD.group_id
          AND event.member_id = NEW.id
          AND event.member_version = NEW.version
          AND event.event_type = 'member_removed';

        IF matching_event_count <> 1 THEN
            RAISE EXCEPTION
                '移除问题组成员缺少唯一member_removed事件';
        END IF;

        RETURN NEW;
    END IF;

    IF OLD.status = 'removed'
       AND NEW.status = 'active' THEN
        SELECT COUNT(*)
        INTO matching_event_count
        FROM public.courseware_review_item_group_events AS event
        WHERE event.group_id = NEW.group_id
          AND event.member_id = NEW.id
          AND event.member_version = NEW.version
          AND event.event_type = 'member_added';

        IF matching_event_count <> 1 THEN
            RAISE EXCEPTION
                '恢复问题组成员缺少唯一member_added事件';
        END IF;

        RETURN NEW;
    END IF;

    IF OLD.status = 'active'
       AND NEW.status = 'active'
       AND OLD.group_id <> NEW.group_id THEN
        SELECT COUNT(*)
        INTO matching_event_count
        FROM public.courseware_review_item_group_events AS event
        WHERE event.member_id = NEW.id
          AND event.member_version = NEW.version
          AND event.event_type = 'member_moved'
          AND (
              (
                  event.group_id = OLD.group_id
                  AND event.related_group_id = NEW.group_id
              )
              OR
              (
                  event.group_id = NEW.group_id
                  AND event.related_group_id = OLD.group_id
              )
          );

        IF matching_event_count <> 2 THEN
            RAISE EXCEPTION
                '移动问题组成员必须在来源组和目标组各写一条member_moved事件';
        END IF;

        RETURN NEW;
    END IF;

    RAISE EXCEPTION '问题组成员状态迁移未形成可审计事件';
END
$$;

REVOKE ALL
ON FUNCTION public.enforce_cw_review_item_group_member_event_consistency()
FROM PUBLIC;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_member_event_consistency
ON courseware_review_item_group_members;

CREATE CONSTRAINT TRIGGER trg_cw_review_item_group_member_event_consistency
AFTER INSERT OR UPDATE
ON courseware_review_item_group_members
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION public.enforce_cw_review_item_group_member_event_consistency();

CREATE OR REPLACE FUNCTION public.enforce_cw_review_item_group_membership_consistency()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    active_member_count BIGINT;
    primary_member_count BIGINT;
    merge_target_status VARCHAR(16);
BEGIN
    SELECT COUNT(*)
    INTO active_member_count
    FROM public.courseware_review_item_group_members AS member
    WHERE member.group_id = NEW.id
      AND member.courseware_id = NEW.courseware_id
      AND member.source_session_id = NEW.source_session_id
      AND member.status = 'active';

    IF NEW.status = 'merged' THEN
        IF active_member_count <> 0 THEN
            RAISE EXCEPTION
                '已合并问题组不能继续保留active成员';
        END IF;

        SELECT target_group.status
        INTO merge_target_status
        FROM public.courseware_review_item_groups AS target_group
        WHERE target_group.id = NEW.merged_into_group_id
          AND target_group.courseware_id = NEW.courseware_id
          AND target_group.source_session_id = NEW.source_session_id
          AND target_group.created_by = NEW.created_by;

        IF NOT FOUND OR merge_target_status <> 'active' THEN
            RAISE EXCEPTION
                '问题组合并目标不存在、越界或已经合并';
        END IF;

        RETURN NEW;
    END IF;

    IF NEW.primary_item_id IS NOT NULL THEN
        SELECT COUNT(*)
        INTO primary_member_count
        FROM public.courseware_review_item_group_members AS member
        WHERE member.group_id = NEW.id
          AND member.courseware_id = NEW.courseware_id
          AND member.source_session_id = NEW.source_session_id
          AND member.item_id = NEW.primary_item_id
          AND member.status = 'active';

        IF primary_member_count <> 1 THEN
            RAISE EXCEPTION
                '问题组主问题必须是该组唯一有效成员';
        END IF;
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION public.enforce_cw_review_item_group_membership_consistency()
FROM PUBLIC;

DROP TRIGGER IF EXISTS trg_cw_review_item_group_membership_consistency
ON courseware_review_item_groups;

CREATE CONSTRAINT TRIGGER trg_cw_review_item_group_membership_consistency
AFTER INSERT OR UPDATE
ON courseware_review_item_groups
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION public.enforce_cw_review_item_group_membership_consistency();

-- ============================================================================
-- 二、R-06 应用角色最小权限
-- ============================================================================

REVOKE ALL PRIVILEGES ON TABLE courseware_review_item_groups FROM PUBLIC;
REVOKE ALL PRIVILEGES ON TABLE courseware_review_item_groups FROM tedna_user;
GRANT SELECT, INSERT ON TABLE courseware_review_item_groups TO tedna_user;
GRANT UPDATE (
    name, primary_item_id, status, version, merged_into_group_id, updated_at
) ON courseware_review_item_groups TO tedna_user;

REVOKE ALL PRIVILEGES ON TABLE courseware_review_item_group_members FROM PUBLIC;
REVOKE ALL PRIVILEGES ON TABLE courseware_review_item_group_members FROM tedna_user;
GRANT SELECT, INSERT ON TABLE courseware_review_item_group_members TO tedna_user;
GRANT UPDATE (
    group_id, status, version, removed_by, removed_at, updated_at
) ON courseware_review_item_group_members TO tedna_user;

REVOKE ALL PRIVILEGES ON TABLE courseware_review_item_group_events FROM PUBLIC;
REVOKE ALL PRIVILEGES ON TABLE courseware_review_item_group_events FROM tedna_user;
GRANT SELECT, INSERT ON TABLE courseware_review_item_group_events TO tedna_user;

-- ============================================================================
-- 三、数据库注释
-- ============================================================================

COMMENT ON TABLE courseware_review_item_groups IS
    'R-06正式问题组；问题组独立于pairwise relation，按version和追加式事件治理';

COMMENT ON COLUMN courseware_review_item_groups.name IS
    '教师可管理的问题组名称；业务层要求使用教学主题或改进目标，不使用技术关系名';

COMMENT ON COLUMN courseware_review_item_groups.primary_item_id IS
    '教师明确设置的主问题；必须是本组active成员';

COMMENT ON TABLE courseware_review_item_group_members IS
    '问题组稳定成员身份；移动、移除和恢复使用version乐观并发，不删除历史身份';

COMMENT ON TABLE courseware_review_item_group_events IS
    '问题组追加式治理事件；每个group_version必须有且仅有一条事件';

DO $$
BEGIN
    RAISE NOTICE 'R-06正式问题组结构完成，等待同事务执行R-07影响方案迁移';
END
$$;

-- 本文件故意不执行COMMIT。
