-- ============================================================================
-- TE-DNA 2.0：课件审核修改指令不可变版本体系 V2.1 验证
-- 文件：20260805_02_courseware_review_instruction_versions_verify.sql
-- ----------------------------------------------------------------------------
-- 本文件只读取数据库，不修改结构或业务数据。
-- 任一结构、权限、回填、版本归属、状态组合或双事实源异常都会抛出异常。
-- ============================================================================

-- ============================================================================
-- 一、共享函数、表和列
-- ============================================================================

DO $$
DECLARE
    required_column TEXT;
BEGIN
    IF to_regprocedure(
        'digest(bytea,text)'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少pgcrypto.digest(bytea,text)函数';
    END IF;

    IF to_regclass(
        'public.courseware_review_instruction_versions'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少courseware_review_instruction_versions表';
    END IF;

    FOREACH required_column IN ARRAY ARRAY[
        'current_instruction_version_id',
        'delivered_instruction_version_id',
        'applied_instruction_version_id'
    ]
    LOOP
        IF NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name =
                  'courseware_review_items'
              AND column_name =
                  required_column
        ) THEN
            RAISE EXCEPTION
                '缺少courseware_review_items.%字段',
                required_column;
        END IF;
    END LOOP;

    FOREACH required_column IN ARRAY ARRAY[
        'id',
        'item_id',
        'version_no',
        'content',
        'content_hash',
        'source_type',
        'created_by',
        'created_at',
        'confirmed_by',
        'confirmed_at',
        'page_snapshot_hash',
        'status'
    ]
    LOOP
        IF NOT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name =
                  'courseware_review_instruction_versions'
              AND column_name =
                  required_column
        ) THEN
            RAISE EXCEPTION
                '缺少指令版本表.%字段',
                required_column;
        END IF;
    END LOOP;
END
$$;

-- ============================================================================
-- 二、约束、索引和触发器
-- ============================================================================

DO $$
DECLARE
    required_constraint TEXT;
    required_index TEXT;
BEGIN
    FOREACH required_constraint IN ARRAY ARRAY[
        'uq_cw_review_instruction_version_no',
        'uq_cw_review_instruction_version_id_item',
        'chk_cw_review_instruction_version_hash',
        'chk_cw_review_instruction_version_confirmation',
        'fk_cw_review_item_current_instruction_version',
        'fk_cw_review_item_delivered_instruction_version',
        'fk_cw_review_item_applied_instruction_version',
        'chk_cw_review_item_instruction_compat',
        'chk_cw_review_item_delivered_instruction_version',
        'chk_cw_review_item_applied_instruction_version'
    ]
    LOOP
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname =
                  required_constraint
        ) THEN
            RAISE EXCEPTION
                '缺少约束%',
                required_constraint;
        END IF;
    END LOOP;

    FOREACH required_index IN ARRAY ARRAY[
        'uq_cw_review_instruction_current_confirmed',
        'idx_cw_review_instruction_version_item',
        'idx_cw_review_instruction_version_creator',
        'idx_cw_review_instruction_version_confirmer',
        'idx_cw_review_item_current_instruction_version',
        'idx_cw_review_item_delivered_instruction_version',
        'idx_cw_review_item_applied_instruction_version'
    ]
    LOOP
        IF NOT EXISTS (
            SELECT 1
            FROM pg_indexes
            WHERE schemaname = 'public'
              AND indexname =
                  required_index
        ) THEN
            RAISE EXCEPTION
                '缺少索引%',
                required_index;
        END IF;
    END LOOP;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname =
            'trg_cw_review_instruction_version_mutation_guard'
          AND tgrelid =
              'courseware_review_instruction_versions'::regclass
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION
            '缺少指令版本不可变和状态迁移守卫';
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname =
            'trg_cw_review_item_instruction_binding_guard'
          AND tgrelid =
              'courseware_review_items'::regclass
          AND NOT tgisinternal
    ) THEN
        RAISE EXCEPTION
            '缺少整改项版本引用和兼容快照守卫';
    END IF;

    -- 版本守卫只拦截UPDATE，不得拦截DELETE级联。
    IF EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname =
            'trg_cw_review_instruction_version_mutation_guard'
          AND (
              pg_get_triggerdef(oid) LIKE '% DELETE %'
              OR pg_get_triggerdef(oid) LIKE '% TRUNCATE %'
          )
    ) THEN
        RAISE EXCEPTION
            '版本守卫错误拦截DELETE或TRUNCATE，可能破坏父资源级联删除';
    END IF;
END
$$;

-- ============================================================================
-- 三、版本正文、编号、哈希和确认状态
-- ============================================================================

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_review_instruction_versions
        WHERE version_no <= 0
           OR length(BTRIM(content)) = 0
           OR content_hash <> encode(
               digest(
                   convert_to(
                       content,
                       'UTF8'
                   ),
                   'sha256'
               ),
               'hex'
           )
    ) THEN
        RAISE EXCEPTION
            '存在版本号、正文或内容哈希异常的指令版本';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM courseware_review_instruction_versions
        WHERE (
            status = 'draft'
            AND (
                confirmed_by IS NOT NULL
                OR confirmed_at IS NOT NULL
            )
        )
        OR (
            status IN (
                'confirmed',
                'superseded',
                'invalid_for_page'
            )
            AND (
                confirmed_by IS NULL
                OR confirmed_at IS NULL
            )
        )
    ) THEN
        RAISE EXCEPTION
            '存在版本状态与确认身份或时间不一致的数据';
    END IF;

    IF EXISTS (
        SELECT item_id
        FROM courseware_review_instruction_versions
        WHERE status = 'confirmed'
        GROUP BY item_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION
            '存在同一整改项多个confirmed版本';
    END IF;

    -- 每条整改项的版本必须从1开始且无缺号。
    IF EXISTS (
        SELECT item_id
        FROM courseware_review_instruction_versions
        GROUP BY item_id
        HAVING MIN(version_no) <> 1
            OR MAX(version_no) <> COUNT(*)
    ) THEN
        RAISE EXCEPTION
            '存在版本号未从1开始或中间缺号的整改项';
    END IF;
END
$$;

-- ============================================================================
-- 四、当前版本与兼容快照
-- ============================================================================

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE BTRIM(
                  COALESCE(
                      confirmed_instruction,
                      ''
                  )
              ) <> ''
          AND current_instruction_version_id IS NULL
    ) THEN
        RAISE EXCEPTION
            '存在非空confirmed_instruction但没有当前版本的整改项';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE BTRIM(
                  COALESCE(
                      confirmed_instruction,
                      ''
                  )
              ) = ''
          AND current_instruction_version_id IS NOT NULL
    ) THEN
        RAISE EXCEPTION
            '存在确认指令为空但仍绑定当前版本的整改项';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        LEFT JOIN courseware_review_instruction_versions AS version
          ON version.id =
                item.current_instruction_version_id
         AND version.item_id = item.id
        WHERE item.current_instruction_version_id IS NOT NULL
          AND (
              version.id IS NULL
              OR BTRIM(version.content) <>
                    BTRIM(item.confirmed_instruction)
              OR version.status NOT IN (
                  'confirmed',
                  'invalid_for_page'
              )
              OR version.confirmed_at IS DISTINCT FROM
                    item.confirmed_at
          )
    ) THEN
        RAISE EXCEPTION
            '存在当前版本归属、正文、状态或确认时间与整改项不一致的数据';
    END IF;

    -- 所有confirmed版本都必须正是整改项当前版本。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_instruction_versions AS version
        LEFT JOIN courseware_review_items AS item
          ON item.id = version.item_id
         AND item.current_instruction_version_id =
                version.id
        WHERE version.status = 'confirmed'
          AND item.id IS NULL
    ) THEN
        RAISE EXCEPTION
            '存在confirmed版本未被对应整改项设为当前版本';
    END IF;

    -- 有确认快照的整改项至少必须保留V1。
    IF EXISTS (
        SELECT 1
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
                AND version.version_no = 1
          )
    ) THEN
        RAISE EXCEPTION
            '存在非空确认指令但没有V1记录的整改项';
    END IF;
END
$$;

-- ============================================================================
-- 五、正式交付版本
-- ============================================================================

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE (
            feedback_id IS NULL
            AND delivered_instruction_version_id IS NOT NULL
        )
        OR (
            feedback_id IS NOT NULL
            AND delivered_instruction_version_id IS NULL
        )
    ) THEN
        RAISE EXCEPTION
            '存在正式反馈与交付版本引用不一致的数据';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE delivered_instruction_version_id IS NOT NULL
          AND (
              current_instruction_version_id IS NULL
              OR delivered_instruction_version_id <>
                    current_instruction_version_id
          )
    ) THEN
        RAISE EXCEPTION
            '存在正式交付版本与当前冻结版本不一致的数据';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        LEFT JOIN courseware_review_instruction_versions AS version
          ON version.id =
                item.delivered_instruction_version_id
         AND version.item_id = item.id
        WHERE item.delivered_instruction_version_id IS NOT NULL
          AND (
              version.id IS NULL
              OR version.status NOT IN (
                  'confirmed',
                  'invalid_for_page'
              )
          )
    ) THEN
        RAISE EXCEPTION
            '存在交付版本归属或状态异常的数据';
    END IF;
END
$$;

-- ============================================================================
-- 六、页面应用版本
-- ============================================================================

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'applying'
          AND (
              applied_instruction_version_id IS NULL
              OR applied_at IS NOT NULL
          )
    ) THEN
        RAISE EXCEPTION
            '存在正在应用但缺少应用版本或错误写入完成时间的整改项';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE applied_at IS NOT NULL
          AND applied_instruction_version_id IS NULL
    ) THEN
        RAISE EXCEPTION
            '存在页面修改事实但没有应用版本引用的整改项';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE applied_instruction_version_id IS NOT NULL
          AND status NOT IN (
              'applying',
              'applied',
              'resolved',
              'stale',
              'orphaned'
          )
    ) THEN
        RAISE EXCEPTION
            '存在不允许的整改项状态保留应用版本引用';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        LEFT JOIN courseware_review_instruction_versions AS version
          ON version.id =
                item.applied_instruction_version_id
         AND version.item_id = item.id
        WHERE item.applied_instruction_version_id IS NOT NULL
          AND (
              version.id IS NULL
              OR version.status NOT IN (
                  'confirmed',
                  'invalid_for_page'
              )
          )
    ) THEN
        RAISE EXCEPTION
            '存在应用版本归属或状态异常的数据';
    END IF;

    -- 正式问题的应用版本必须等于正式交付版本；
    -- 自审或尚未交付问题的应用版本必须等于当前版本。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE applied_instruction_version_id IS NOT NULL
          AND applied_instruction_version_id <>
                COALESCE(
                    delivered_instruction_version_id,
                    current_instruction_version_id
                )
    ) THEN
        RAISE EXCEPTION
            '存在应用版本与交付或当前版本不一致的数据';
    END IF;
END
$$;

-- ============================================================================
-- 七、页面失效状态与重新检查兼容
-- ============================================================================

DO $$
BEGIN
    -- stale或orphaned的当前版本不得继续保持confirmed执行资格。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        JOIN courseware_review_instruction_versions AS version
          ON version.id =
                item.current_instruction_version_id
         AND version.item_id = item.id
        WHERE item.status IN (
            'stale',
            'orphaned'
        )
          AND version.status <> 'invalid_for_page'
    ) THEN
        RAISE EXCEPTION
            '存在页面已变化或删除但当前版本仍可执行的数据';
    END IF;

    -- stale重新检查回到applied后，允许保留invalid_for_page历史版本。
    -- 该状态仅代表修改事实重新登记，不代表旧版本恢复执行资格。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        JOIN courseware_review_instruction_versions AS version
          ON version.id =
                item.current_instruction_version_id
         AND version.item_id = item.id
        WHERE item.status = 'applying'
          AND version.status <> 'confirmed'
    ) THEN
        RAISE EXCEPTION
            '存在实际页面执行使用失效指令版本的数据';
    END IF;
END
$$;

-- ============================================================================
-- 八、应用角色最小权限
-- ============================================================================

DO $$
BEGIN
    IF NOT has_table_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'SELECT'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少指令版本表SELECT权限';
    END IF;

    IF NOT has_table_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'INSERT'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少指令版本表INSERT权限';
    END IF;

    IF has_table_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'DELETE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user不应拥有指令版本表DELETE权限';
    END IF;

    IF NOT has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'status',
        'UPDATE'
    )
    OR NOT has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'confirmed_by',
        'UPDATE'
    )
    OR NOT has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'confirmed_at',
        'UPDATE'
    )
    OR NOT has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'page_snapshot_hash',
        'UPDATE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user缺少版本确认或状态迁移所需的列级UPDATE权限';
    END IF;

    IF has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'content',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'version_no',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'item_id',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'created_by',
        'UPDATE'
    )
    OR has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'source_type',
        'UPDATE'
    ) THEN
        RAISE EXCEPTION
            'tedna_user不应能修改版本正文、编号、归属、创建者或来源';
    END IF;
END
$$;

-- ============================================================================
-- 九、输出迁移证据
-- ============================================================================

SELECT
    item.source_type,
    item.status AS item_status,
    version.status AS current_version_status,
    COUNT(*) AS item_count
FROM courseware_review_items AS item
LEFT JOIN courseware_review_instruction_versions AS version
       ON version.id =
            item.current_instruction_version_id
      AND version.item_id = item.id
GROUP BY
    item.source_type,
    item.status,
    version.status
ORDER BY
    item.source_type,
    item.status,
    version.status;

SELECT
    version.source_type,
    version.status,
    COUNT(*) AS version_count,
    MIN(version.version_no) AS min_version_no,
    MAX(version.version_no) AS max_version_no
FROM courseware_review_instruction_versions AS version
GROUP BY
    version.source_type,
    version.status
ORDER BY
    version.source_type,
    version.status;

SELECT
    COUNT(*) FILTER (
        WHERE current_instruction_version_id IS NOT NULL
    ) AS current_version_items,
    COUNT(*) FILTER (
        WHERE delivered_instruction_version_id IS NOT NULL
    ) AS delivered_version_items,
    COUNT(*) FILTER (
        WHERE applied_instruction_version_id IS NOT NULL
    ) AS applied_version_items
FROM courseware_review_items;

SELECT
    tgrelid::regclass::text AS table_name,
    tgname,
    pg_get_triggerdef(oid) AS definition
FROM pg_trigger
WHERE tgname IN (
    'trg_cw_review_instruction_version_mutation_guard',
    'trg_cw_review_item_instruction_binding_guard'
)
ORDER BY table_name, tgname;

SELECT
    has_table_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'SELECT'
    ) AS version_select,
    has_table_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'INSERT'
    ) AS version_insert,
    has_table_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'DELETE'
    ) AS version_delete,
    has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'status',
        'UPDATE'
    ) AS version_status_update,
    has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'confirmed_by',
        'UPDATE'
    ) AS version_confirmer_update,
    has_column_privilege(
        'tedna_user',
        'courseware_review_instruction_versions',
        'content',
        'UPDATE'
    ) AS version_content_update;
