-- ============================================================================
-- TE-DNA 2.0：课件审核修改指令版本数据库守卫 V2.1
-- 文件：20260805_03_courseware_review_instruction_version_guards.sql
-- ----------------------------------------------------------------------------
-- 本文件必须与以下结构迁移在同一psql连接中连续执行：
--
--   20260805_02_courseware_review_instruction_versions.sql
--
-- 本文件负责：
--   1. 保护版本正文、编号、来源和创建身份不可覆盖；
--   2. 允许draft在明确确认时补写确认人、确认时间和页面快照；
--   3. 兼容旧后端直接更新confirmed_instruction的短暂发布窗口；
--   4. 固化正式交付版本和页面应用版本；
--   5. 支持applying失败回退和既有stale重新检查链；
--   6. 页面变化或删除时将当前版本标记为invalid_for_page；
--   7. 设置应用数据库角色最小权限；
--   8. 最后统一提交两个迁移文件形成的同一事务。
-- ============================================================================

DO $$
BEGIN
    IF to_regclass(
        'public.courseware_review_instruction_versions'
    ) IS NULL THEN
        RAISE EXCEPTION
            '必须先在同一事务执行指令版本结构迁移';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'courseware_review_items'
          AND column_name = 'current_instruction_version_id'
    ) THEN
        RAISE EXCEPTION
            '结构迁移尚未完成，缺少当前指令版本字段';
    END IF;
END
$$;

-- ============================================================================
-- 一、指令版本不可变与状态迁移守卫
-- ============================================================================
-- 不建立DELETE触发器：
--
--   - 应用角色没有DELETE权限，不能单独删除历史版本；
--   - 删除课件或整改项时仍需允许ON DELETE CASCADE清理关联数据；
--   - 不能用版本不可变规则破坏父资源的既有永久删除链。

CREATE OR REPLACE FUNCTION
    public.guard_cw_review_instruction_version_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    -- 以下字段在任何UPDATE中都不可改变。
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.item_id IS DISTINCT FROM OLD.item_id
       OR NEW.version_no IS DISTINCT FROM OLD.version_no
       OR NEW.content IS DISTINCT FROM OLD.content
       OR NEW.content_hash IS DISTINCT FROM OLD.content_hash
       OR NEW.source_type IS DISTINCT FROM OLD.source_type
       OR NEW.created_by IS DISTINCT FROM OLD.created_by
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION
            '课件整改指令版本正文、编号、归属、来源和创建身份不可覆盖';
    END IF;

    -- 没有状态变化时，确认身份、确认时间和页面快照也不可修改。
    IF NEW.status = OLD.status THEN
        IF NEW.confirmed_by IS DISTINCT FROM OLD.confirmed_by
           OR NEW.confirmed_at IS DISTINCT FROM OLD.confirmed_at
           OR NEW.page_snapshot_hash IS DISTINCT FROM
                OLD.page_snapshot_hash THEN
            RAISE EXCEPTION
                '课件整改指令版本确认信息和页面快照不可原地修改';
        END IF;

        RETURN NEW;
    END IF;

    -- draft在用户明确确认时允许一次性补写确认事实和确认时页面快照。
    IF OLD.status = 'draft'
       AND NEW.status = 'confirmed' THEN
        IF OLD.confirmed_by IS NOT NULL
           OR OLD.confirmed_at IS NOT NULL
           OR NEW.confirmed_by IS NULL
           OR NEW.confirmed_at IS NULL THEN
            RAISE EXCEPTION
                '草稿确认必须一次性写入确认人和确认时间';
        END IF;

        RETURN NEW;
    END IF;

    -- 已确认版本只允许被新版替代或因页面变化失效。
    IF OLD.status = 'confirmed'
       AND NEW.status IN (
           'superseded',
           'invalid_for_page'
       ) THEN
        IF NEW.confirmed_by IS DISTINCT FROM OLD.confirmed_by
           OR NEW.confirmed_at IS DISTINCT FROM OLD.confirmed_at
           OR NEW.page_snapshot_hash IS DISTINCT FROM
                OLD.page_snapshot_hash THEN
            RAISE EXCEPTION
                '已确认版本状态变化时不得改写确认事实或页面快照';
        END IF;

        RETURN NEW;
    END IF;

    -- 已被替代的版本仍可因页面变化进一步标记为不适用。
    IF OLD.status = 'superseded'
       AND NEW.status = 'invalid_for_page' THEN
        IF NEW.confirmed_by IS DISTINCT FROM OLD.confirmed_by
           OR NEW.confirmed_at IS DISTINCT FROM OLD.confirmed_at
           OR NEW.page_snapshot_hash IS DISTINCT FROM
                OLD.page_snapshot_hash THEN
            RAISE EXCEPTION
                '历史版本失效时不得改写确认事实或页面快照';
        END IF;

        RETURN NEW;
    END IF;

    RAISE EXCEPTION
        '课件整改指令版本状态迁移无效：% -> %',
        OLD.status,
        NEW.status;
END
$$;

REVOKE ALL
ON FUNCTION
    public.guard_cw_review_instruction_version_mutation()
FROM PUBLIC;

DROP TRIGGER IF EXISTS
    trg_cw_review_instruction_version_mutation_guard
ON courseware_review_instruction_versions;

CREATE TRIGGER
    trg_cw_review_instruction_version_mutation_guard
BEFORE UPDATE
ON courseware_review_instruction_versions
FOR EACH ROW
EXECUTE FUNCTION
    public.guard_cw_review_instruction_version_mutation();

-- ============================================================================
-- 二、整改项当前、交付和应用版本守卫
-- ============================================================================

CREATE OR REPLACE FUNCTION
    public.guard_cw_review_item_instruction_bindings()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    referenced_content TEXT;
    referenced_status VARCHAR(24);
    referenced_confirmed_at TIMESTAMPTZ;

    next_version_no INTEGER;
    created_version_id UUID;
    effective_confirmed_at TIMESTAMPTZ;

    expected_application_version_id UUID;

    application_binding_allowed BOOLEAN := FALSE;
    application_clearing_allowed BOOLEAN := FALSE;
BEGIN
    -- ========================================================================
    -- 1. 正式交付后的指令冻结
    -- ========================================================================

    IF OLD.feedback_id IS NOT NULL
       AND (
           NEW.current_instruction_version_id IS DISTINCT FROM
                OLD.current_instruction_version_id
           OR NEW.confirmed_instruction IS DISTINCT FROM
                OLD.confirmed_instruction
           OR NEW.confirmed_at IS DISTINCT FROM
                OLD.confirmed_at
       ) THEN
        RAISE EXCEPTION
            '正式交付后的确认指令和当前版本不可修改';
    END IF;

    IF OLD.delivered_instruction_version_id IS NOT NULL
       AND NEW.delivered_instruction_version_id IS DISTINCT FROM
            OLD.delivered_instruction_version_id THEN
        RAISE EXCEPTION
            '正式审核交付版本不可修改';
    END IF;

    -- ========================================================================
    -- 2. 兼容快照和当前确认版本同步
    -- ========================================================================

    IF NEW.confirmed_instruction IS DISTINCT FROM
            OLD.confirmed_instruction THEN
        -- 清空确认指令表示重新进入讨论。
        IF BTRIM(
            COALESCE(
                NEW.confirmed_instruction,
                ''
            )
        ) = '' THEN
            IF NEW.status <> 'discussing' THEN
                RAISE EXCEPTION
                    '清空确认指令时整改项必须进入discussing状态';
            END IF;

            IF OLD.feedback_id IS NOT NULL
               OR OLD.delivered_instruction_version_id IS NOT NULL
               OR OLD.applied_instruction_version_id IS NOT NULL THEN
                RAISE EXCEPTION
                    '已交付或已应用的确认指令不可重新讨论或清空';
            END IF;

            IF OLD.current_instruction_version_id IS NOT NULL THEN
                UPDATE courseware_review_instruction_versions
                SET status = 'superseded'
                WHERE id = OLD.current_instruction_version_id
                  AND item_id = OLD.id
                  AND status = 'confirmed';
            END IF;

            NEW.current_instruction_version_id := NULL;
            NEW.confirmed_at := NULL;

        ELSE
            -- 保存并确认新版本时，整改项必须进入confirmed。
            IF NEW.status <> 'confirmed' THEN
                RAISE EXCEPTION
                    '确认非空指令时整改项必须进入confirmed状态';
            END IF;

            IF OLD.feedback_id IS NOT NULL
               OR OLD.delivered_instruction_version_id IS NOT NULL
               OR OLD.applied_instruction_version_id IS NOT NULL THEN
                RAISE EXCEPTION
                    '已交付或已应用的确认指令不可重新确认';
            END IF;

            -- 浏览器旧版本不会提交新version_id。
            --
            -- 如果当前引用为空，或UPDATE仍保留旧引用，则视为旧代码兼容路径：
            -- 在整改项行锁内先替代旧版本，再自动创建连续的新确认版本。
            IF NEW.current_instruction_version_id IS NULL
               OR NEW.current_instruction_version_id =
                    OLD.current_instruction_version_id THEN
                IF OLD.current_instruction_version_id IS NOT NULL THEN
                    UPDATE courseware_review_instruction_versions
                    SET status = 'superseded'
                    WHERE id = OLD.current_instruction_version_id
                      AND item_id = OLD.id
                      AND status = 'confirmed';
                END IF;

                SELECT
                    COALESCE(
                        MAX(version.version_no),
                        0
                    ) + 1
                INTO next_version_no
                FROM courseware_review_instruction_versions AS version
                WHERE version.item_id = NEW.id;

                effective_confirmed_at :=
                    COALESCE(
                        NEW.confirmed_at,
                        clock_timestamp()
                    );

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
                VALUES (
                    NEW.id,
                    next_version_no,
                    BTRIM(NEW.confirmed_instruction),
                    encode(
                        digest(
                            convert_to(
                                BTRIM(NEW.confirmed_instruction),
                                'UTF8'
                            ),
                            'sha256'
                        ),
                        'hex'
                    ),
                    'legacy_direct_update',
                    NEW.created_by,
                    effective_confirmed_at,
                    NEW.created_by,
                    effective_confirmed_at,
                    BTRIM(
                        COALESCE(
                            NEW.page_html_hash,
                            ''
                        )
                    ),
                    'confirmed'
                )
                RETURNING id
                INTO created_version_id;

                NEW.current_instruction_version_id :=
                    created_version_id;
                NEW.confirmed_at :=
                    effective_confirmed_at;

            ELSE
                -- 新后端已经在同一事务显式创建并确认版本。
                IF OLD.current_instruction_version_id IS NOT NULL THEN
                    UPDATE courseware_review_instruction_versions
                    SET status = 'superseded'
                    WHERE id = OLD.current_instruction_version_id
                      AND item_id = OLD.id
                      AND status = 'confirmed';
                END IF;

                SELECT
                    version.content,
                    version.status,
                    version.confirmed_at
                INTO
                    referenced_content,
                    referenced_status,
                    referenced_confirmed_at
                FROM courseware_review_instruction_versions AS version
                WHERE version.id =
                        NEW.current_instruction_version_id
                  AND version.item_id = NEW.id;

                IF NOT FOUND
                   OR referenced_status <> 'confirmed'
                   OR BTRIM(referenced_content) <>
                        BTRIM(NEW.confirmed_instruction) THEN
                    RAISE EXCEPTION
                        '显式当前版本不存在、归属错误、未确认或正文不一致';
                END IF;

                NEW.confirmed_at :=
                    referenced_confirmed_at;
            END IF;
        END IF;

    ELSIF NEW.current_instruction_version_id IS DISTINCT FROM
            OLD.current_instruction_version_id THEN
        -- 允许创建正文相同的新版本，但必须同时保持confirmed状态。
        IF NEW.current_instruction_version_id IS NULL THEN
            IF BTRIM(
                COALESCE(
                    NEW.confirmed_instruction,
                    ''
                )
            ) <> '' THEN
                RAISE EXCEPTION
                    '清空当前版本时必须同时清空兼容确认指令';
            END IF;

            IF NEW.status <> 'discussing' THEN
                RAISE EXCEPTION
                    '清空当前版本时整改项必须进入discussing状态';
            END IF;

            IF OLD.current_instruction_version_id IS NOT NULL THEN
                UPDATE courseware_review_instruction_versions
                SET status = 'superseded'
                WHERE id = OLD.current_instruction_version_id
                  AND item_id = OLD.id
                  AND status = 'confirmed';
            END IF;

            NEW.confirmed_at := NULL;

        ELSE
            IF NEW.status <> 'confirmed' THEN
                RAISE EXCEPTION
                    '切换当前确认版本时整改项必须进入confirmed状态';
            END IF;

            IF OLD.feedback_id IS NOT NULL
               OR OLD.delivered_instruction_version_id IS NOT NULL
               OR OLD.applied_instruction_version_id IS NOT NULL THEN
                RAISE EXCEPTION
                    '已交付或已应用的整改项不可切换当前版本';
            END IF;

            IF OLD.current_instruction_version_id IS NOT NULL THEN
                UPDATE courseware_review_instruction_versions
                SET status = 'superseded'
                WHERE id = OLD.current_instruction_version_id
                  AND item_id = OLD.id
                  AND status = 'confirmed';
            END IF;

            SELECT
                version.content,
                version.status,
                version.confirmed_at
            INTO
                referenced_content,
                referenced_status,
                referenced_confirmed_at
            FROM courseware_review_instruction_versions AS version
            WHERE version.id =
                    NEW.current_instruction_version_id
              AND version.item_id = NEW.id;

            IF NOT FOUND
               OR referenced_status <> 'confirmed'
               OR BTRIM(referenced_content) <>
                    BTRIM(NEW.confirmed_instruction) THEN
                RAISE EXCEPTION
                    '当前版本切换与兼容确认指令不一致';
            END IF;

            NEW.confirmed_at :=
                referenced_confirmed_at;
        END IF;
    END IF;

    -- ========================================================================
    -- 3. 正式审核交付版本
    -- ========================================================================

    IF OLD.delivered_instruction_version_id IS NULL
       AND NEW.delivered_instruction_version_id IS NOT NULL
       AND NOT (
           OLD.feedback_id IS NULL
           AND NEW.feedback_id IS NOT NULL
       ) THEN
        RAISE EXCEPTION
            '交付版本只能在正式审核提交时形成';
    END IF;

    IF NEW.feedback_id IS NOT NULL
       AND OLD.feedback_id IS NULL THEN
        IF NEW.current_instruction_version_id IS NULL THEN
            RAISE EXCEPTION
                '正式审核交付必须绑定当前确认版本';
        END IF;

        IF NEW.delivered_instruction_version_id IS NULL THEN
            NEW.delivered_instruction_version_id :=
                NEW.current_instruction_version_id;
        END IF;

        IF NEW.delivered_instruction_version_id <>
                NEW.current_instruction_version_id THEN
            RAISE EXCEPTION
                '正式审核交付版本必须等于提交时当前确认版本';
        END IF;

        SELECT version.status
        INTO referenced_status
        FROM courseware_review_instruction_versions AS version
        WHERE version.id =
                NEW.delivered_instruction_version_id
          AND version.item_id = NEW.id;

        IF NOT FOUND
           OR referenced_status <> 'confirmed' THEN
            RAISE EXCEPTION
                '正式审核交付版本不存在、归属错误或已经失效';
        END IF;
    END IF;

    -- ========================================================================
    -- 4. 页面应用版本
    -- ========================================================================

    expected_application_version_id :=
        COALESCE(
            NEW.delivered_instruction_version_id,
            NEW.current_instruction_version_id
        );

    -- confirmed -> applying：实际开始执行时必须绑定仍有效的确认版本。
    IF OLD.status = 'confirmed'
       AND NEW.status = 'applying' THEN
        application_binding_allowed := TRUE;

        IF expected_application_version_id IS NULL THEN
            RAISE EXCEPTION
                '开始页面应用时缺少可执行指令版本';
        END IF;

        IF NEW.applied_instruction_version_id IS NULL THEN
            NEW.applied_instruction_version_id :=
                expected_application_version_id;
        END IF;

        IF NEW.applied_instruction_version_id <>
                expected_application_version_id THEN
            RAISE EXCEPTION
                '页面应用版本必须等于当前交付或确认版本';
        END IF;

        SELECT version.status
        INTO referenced_status
        FROM courseware_review_instruction_versions AS version
        WHERE version.id =
                NEW.applied_instruction_version_id
          AND version.item_id = NEW.id;

        IF NOT FOUND
           OR referenced_status <> 'confirmed' THEN
            RAISE EXCEPTION
                '页面应用版本不存在、归属错误或已经失效';
        END IF;
    END IF;

    -- applying失败回退confirmed，或执行中页面已经变化/删除时，
    -- 尚未形成applied事实，可以清理本次临时应用版本绑定。
    IF OLD.status = 'applying'
       AND OLD.applied_at IS NULL
       AND NEW.applied_at IS NULL
       AND NEW.status IN (
           'confirmed',
           'stale',
           'orphaned'
       ) THEN
        application_clearing_allowed := TRUE;
        NEW.applied_instruction_version_id := NULL;
    END IF;

    -- applying/confirmed -> applied是正常完成；
    -- stale -> applied是既有重新检查链，不是重新执行旧版本。
    IF NEW.status = 'applied'
       AND OLD.status <> 'applied' THEN
        IF OLD.status NOT IN (
            'applying',
            'confirmed',
            'stale'
        ) THEN
            RAISE EXCEPTION
                '当前整改项状态不能直接进入applied';
        END IF;

        IF expected_application_version_id IS NULL THEN
            RAISE EXCEPTION
                '记录页面修改完成时缺少关联指令版本';
        END IF;

        IF NEW.applied_instruction_version_id IS NULL THEN
            NEW.applied_instruction_version_id :=
                expected_application_version_id;
        END IF;

        IF OLD.applied_instruction_version_id IS NULL THEN
            application_binding_allowed := TRUE;
        END IF;

        IF NEW.applied_instruction_version_id <>
                expected_application_version_id THEN
            RAISE EXCEPTION
                '页面修改完成版本必须等于当前交付或确认版本';
        END IF;

        SELECT version.status
        INTO referenced_status
        FROM courseware_review_instruction_versions AS version
        WHERE version.id =
                NEW.applied_instruction_version_id
          AND version.item_id = NEW.id;

        IF NOT FOUND THEN
            RAISE EXCEPTION
                '页面修改完成版本不存在或不属于对应整改项';
        END IF;

        IF OLD.status = 'stale' THEN
            -- 重新检查只确认当前页面仍满足原整改要求；
            -- 允许关联历史invalid_for_page版本，但不会恢复该版本执行资格。
            IF referenced_status NOT IN (
                'confirmed',
                'invalid_for_page'
            ) THEN
                RAISE EXCEPTION
                    '重新检查关联的指令版本状态无效';
            END IF;
        ELSIF referenced_status <> 'confirmed' THEN
            RAISE EXCEPTION
                '页面实际执行必须使用仍有效的confirmed版本';
        END IF;
    END IF;

    -- 新增应用版本引用只能发生在允许的开始或完成路径。
    IF OLD.applied_instruction_version_id IS NULL
       AND NEW.applied_instruction_version_id IS NOT NULL
       AND NOT application_binding_allowed THEN
        RAISE EXCEPTION
            '页面应用版本绑定时机无效';
    END IF;

    -- 已有应用版本不可替换。
    -- 只有尚未完成的applying失败或失效收敛可以清空临时引用。
    IF OLD.applied_instruction_version_id IS NOT NULL
       AND NEW.applied_instruction_version_id IS DISTINCT FROM
            OLD.applied_instruction_version_id
       AND NOT application_clearing_allowed THEN
        RAISE EXCEPTION
            '已经绑定的页面应用版本不可修改';
    END IF;

    -- ========================================================================
    -- 5. 页面变化或删除使当前版本失去执行资格
    -- ========================================================================

    IF NEW.status IN (
        'stale',
        'orphaned'
    )
       AND NEW.status IS DISTINCT FROM OLD.status
       AND NEW.current_instruction_version_id IS NOT NULL THEN
        UPDATE courseware_review_instruction_versions
        SET status = 'invalid_for_page'
        WHERE id = NEW.current_instruction_version_id
          AND item_id = NEW.id
          AND status IN (
              'confirmed',
              'superseded'
          );
    END IF;

    -- ========================================================================
    -- 6. 最终版本归属和双事实源复核
    -- ========================================================================

    IF NEW.current_instruction_version_id IS NULL THEN
        IF BTRIM(
            COALESCE(
                NEW.confirmed_instruction,
                ''
            )
        ) <> '' THEN
            RAISE EXCEPTION
                '确认指令非空时必须绑定当前版本';
        END IF;
    ELSE
        SELECT
            version.content,
            version.status,
            version.confirmed_at
        INTO
            referenced_content,
            referenced_status,
            referenced_confirmed_at
        FROM courseware_review_instruction_versions AS version
        WHERE version.id =
                NEW.current_instruction_version_id
          AND version.item_id = NEW.id;

        IF NOT FOUND
           OR BTRIM(referenced_content) <>
                BTRIM(NEW.confirmed_instruction)
           OR referenced_status NOT IN (
                'confirmed',
                'invalid_for_page'
           ) THEN
            RAISE EXCEPTION
                '当前版本归属、正文或状态与兼容快照不一致';
        END IF;

        NEW.confirmed_at :=
            referenced_confirmed_at;
    END IF;

    IF NEW.delivered_instruction_version_id IS NOT NULL THEN
        IF NEW.feedback_id IS NULL
           OR NEW.current_instruction_version_id IS NULL
           OR NEW.delivered_instruction_version_id <>
                NEW.current_instruction_version_id THEN
            RAISE EXCEPTION
                '正式反馈、当前版本和交付版本组合不一致';
        END IF;

        SELECT version.status
        INTO referenced_status
        FROM courseware_review_instruction_versions AS version
        WHERE version.id =
                NEW.delivered_instruction_version_id
          AND version.item_id = NEW.id;

        IF NOT FOUND
           OR referenced_status NOT IN (
                'confirmed',
                'invalid_for_page'
           ) THEN
            RAISE EXCEPTION
                '交付版本不存在、归属错误或状态异常';
        END IF;
    END IF;

    IF NEW.applied_instruction_version_id IS NOT NULL THEN
        IF expected_application_version_id IS NULL
           OR NEW.applied_instruction_version_id <>
                expected_application_version_id THEN
            RAISE EXCEPTION
                '应用版本与当前交付或确认版本不一致';
        END IF;

        SELECT version.status
        INTO referenced_status
        FROM courseware_review_instruction_versions AS version
        WHERE version.id =
                NEW.applied_instruction_version_id
          AND version.item_id = NEW.id;

        IF NOT FOUND
           OR referenced_status NOT IN (
                'confirmed',
                'invalid_for_page'
           ) THEN
            RAISE EXCEPTION
                '应用版本不存在、归属错误或状态异常';
        END IF;
    END IF;

    RETURN NEW;
END
$$;

REVOKE ALL
ON FUNCTION
    public.guard_cw_review_item_instruction_bindings()
FROM PUBLIC;

DROP TRIGGER IF EXISTS
    trg_cw_review_item_instruction_binding_guard
ON courseware_review_items;

CREATE TRIGGER
    trg_cw_review_item_instruction_binding_guard
BEFORE UPDATE
ON courseware_review_items
FOR EACH ROW
EXECUTE FUNCTION
    public.guard_cw_review_item_instruction_bindings();

-- ============================================================================
-- 三、应用角色最小权限
-- ============================================================================

REVOKE ALL PRIVILEGES
ON TABLE courseware_review_instruction_versions
FROM PUBLIC;

REVOKE ALL PRIVILEGES
ON TABLE courseware_review_instruction_versions
FROM tedna_user;

GRANT
    SELECT,
    INSERT
ON TABLE courseware_review_instruction_versions
TO tedna_user;

-- 新后端可先保存draft，再在明确确认时一次性写入以下确认字段。
-- 已确认版本后续只能通过同一UPDATE权限迁移为superseded或invalid_for_page；
-- 触发器负责限制合法状态边界。
GRANT UPDATE (
    status,
    confirmed_by,
    confirmed_at,
    page_snapshot_hash
)
ON courseware_review_instruction_versions
TO tedna_user;

COMMIT;

DO $$
BEGIN
    RAISE NOTICE
        '课件审核修改指令不可变版本体系V2.1数据库迁移已原子提交';
END
$$;
