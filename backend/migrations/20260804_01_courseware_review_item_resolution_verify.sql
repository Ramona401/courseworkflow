-- 20260804_01_courseware_review_item_resolution_verify.sql
--
-- V1.3整改复审闭环迁移后的数据库结构、状态和内容指纹校验。
--
-- 任意检查不通过都会抛出异常。

DO $$
BEGIN
    IF to_regprocedure(
        'digest(bytea,text)'
    ) IS NULL THEN
        RAISE EXCEPTION
            '缺少pgcrypto.digest(bytea,text)函数';
    END IF;
END
$$;

DO $$
DECLARE
    required_column TEXT;
BEGIN
    FOREACH required_column IN ARRAY ARRAY[
        'resubmitted_at',
        'resubmitted_review_level',
        'resubmitted_review_round',
        'resolved_by',
        'resolved_review_id',
        'resolved_review_level',
        'resolved_review_round',
        'resolution_note'
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
END
$$;

DO $$
DECLARE
    required_constraint TEXT;
BEGIN
    FOREACH required_constraint IN ARRAY ARRAY[
        'chk_cw_review_item_resubmission',
        'chk_cw_review_item_resolution',
        'fk_cw_review_item_resolved_by',
        'fk_cw_review_item_resolution_review'
    ]
    LOOP
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conrelid =
                  'courseware_review_items'::regclass
              AND conname =
                  required_constraint
        ) THEN
            RAISE EXCEPTION
                '缺少约束%',
                required_constraint;
        END IF;
    END LOOP;
END
$$;

DO $$
DECLARE
    required_index TEXT;
BEGIN
    FOREACH required_index IN ARRAY ARRAY[
        'idx_cw_review_item_resubmitted',
        'idx_cw_review_item_resolution_review'
    ]
    LOOP
        IF NOT EXISTS (
            SELECT 1
            FROM pg_indexes
            WHERE schemaname = 'public'
              AND tablename =
                  'courseware_review_items'
              AND indexname =
                  required_index
        ) THEN
            RAISE EXCEPTION
                '缺少索引%',
                required_index;
        END IF;
    END LOOP;
END
$$;

DO $$
BEGIN
    -- 已显示解决的问题必须存在明确确认人。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'resolved'
          AND resolved_by IS NULL
    ) THEN
        RAISE EXCEPTION
            '存在已显示解决但没有人工确认人的问题';
    END IF;

    -- 尚未解决的问题不得保留任何解决确认字段。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status <> 'resolved'
          AND (
              resolved_at IS NOT NULL
              OR resolved_by IS NOT NULL
              OR resolved_review_id IS NOT NULL
              OR resolved_review_level <> 0
              OR resolved_review_round <> 0
              OR BTRIM(resolution_note) <> ''
          )
    ) THEN
        RAISE EXCEPTION
            '存在尚未解决但保留解决确认信息的问题';
    END IF;

    -- 正式问题必须绑定一次真实正式复审。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'resolved'
          AND source_type = 'formal'
          AND (
              resolved_review_id IS NULL
              OR resolved_review_level NOT IN (1, 2)
              OR resolved_review_round <= 0
          )
    ) THEN
        RAISE EXCEPTION
            '存在正式问题已解决但缺少复审核心信息';
    END IF;

    -- 作者自审解决不应伪造正式审核记录。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'resolved'
          AND source_type = 'self'
          AND (
              resolved_review_id IS NOT NULL
              OR resolved_review_level <> 0
              OR resolved_review_round <> 0
          )
    ) THEN
        RAISE EXCEPTION
            '存在作者自审问题错误绑定正式复审信息';
    END IF;
END
$$;

DO $$
BEGIN
    -- applied必须具有完整的修改完成证据。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'applied'
          AND (
              applied_at IS NULL
              OR BTRIM(
                  COALESCE(
                      applied_page_hash,
                      ''
                  )
              ) = ''
          )
    ) THEN
        RAISE EXCEPTION
            '存在已完成修改但缺少完成时间或页面指纹的问题';
    END IF;

    -- 有稳定页面的applied记录，原页面必须仍然存在。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        LEFT JOIN courseware_pages AS page
               ON page.id = item.page_id
              AND page.courseware_id =
                  item.courseware_id
        WHERE item.status = 'applied'
          AND item.page_id IS NOT NULL
          AND page.id IS NULL
    ) THEN
        RAISE EXCEPTION
            '存在已完成修改但原页面已经删除的问题';
    END IF;

    -- 有稳定页面的applied记录，当前HTML必须仍与完成修改时一致。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items AS item
        JOIN courseware_pages AS page
          ON page.id = item.page_id
         AND page.courseware_id =
             item.courseware_id
        WHERE item.status = 'applied'
          AND encode(
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
          ) <> BTRIM(
              COALESCE(
                  item.applied_page_hash,
                  ''
              )
          )
    ) THEN
        RAISE EXCEPTION
            '存在已完成修改但当前页面内容已经变化的问题';
    END IF;
END
$$;

DO $$
BEGIN
    -- stale应保留曾经完成修改的证据，供人工回看。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'stale'
          AND applied_at IS NOT NULL
          AND BTRIM(
              COALESCE(
                  applied_page_hash,
                  ''
              )
          ) = ''
    ) THEN
        RAISE EXCEPTION
            '存在页面已变化但修改证据不完整的问题';
    END IF;

    -- 历史自动关闭记录已经全部恢复，不应再存在无确认人的resolved。
    IF EXISTS (
        SELECT 1
        FROM courseware_review_items
        WHERE status = 'resolved'
          AND (
              resolved_at IS NULL
              OR resolved_by IS NULL
          )
    ) THEN
        RAISE EXCEPTION
            '仍存在旧流程自动关闭的问题';
    END IF;
END
$$;

-- 输出迁移后的状态分布，便于部署记录保存。
SELECT
    source_type,
    status,
    COUNT(*) AS item_count
FROM courseware_review_items
GROUP BY
    source_type,
    status
ORDER BY
    source_type,
    status;

SELECT
    status,
    COUNT(*) AS historical_recovered_count
FROM courseware_review_items
WHERE applied_at IS NOT NULL
  AND BTRIM(
      COALESCE(
          applied_page_hash,
          ''
      )
  ) <> ''
  AND status IN (
      'applied',
      'stale',
      'orphaned'
  )
GROUP BY status
ORDER BY status;
