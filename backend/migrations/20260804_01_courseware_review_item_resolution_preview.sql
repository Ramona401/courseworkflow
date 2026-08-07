-- 20260804_01_courseware_review_item_resolution_preview.sql
--
-- V1.3迁移执行前的只读预览。
--
-- 本文件不修改数据库，只按照正式迁移完全相同的规则，
-- 计算旧流程自动关闭问题将恢复到哪个状态。

\pset pager off

WITH historical_resolution AS (
    SELECT
        item.id,
        item.source_type,
        item.page_id,
        item.applied_at,
        item.applied_page_hash,
        item.confirmed_instruction,
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
),
classification AS (
    SELECT
        id,
        source_type,
        CASE
            WHEN page_id IS NOT NULL
             AND current_page_id IS NULL
                THEN 'orphaned'

            WHEN page_id IS NOT NULL
             AND applied_at IS NOT NULL
             AND BTRIM(
                 COALESCE(
                     applied_page_hash,
                     ''
                 )
             ) <> ''
             AND current_page_hash <>
                 BTRIM(
                     COALESCE(
                         applied_page_hash,
                         ''
                     )
                 )
                THEN 'stale'

            WHEN applied_at IS NOT NULL
             AND BTRIM(
                 COALESCE(
                     applied_page_hash,
                     ''
                 )
             ) <> ''
                THEN 'applied'

            WHEN BTRIM(
                 COALESCE(
                     confirmed_instruction,
                     ''
                 )
             ) <> ''
                THEN 'confirmed'

            ELSE 'detected'
        END AS target_status
    FROM historical_resolution
)
SELECT
    source_type,
    target_status,
    COUNT(*) AS item_count
FROM classification
GROUP BY
    source_type,
    target_status
ORDER BY
    source_type,
    target_status;

WITH historical_resolution AS (
    SELECT
        item.id,
        item.page_id,
        item.applied_at,
        item.applied_page_hash,
        item.confirmed_instruction,
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
),
classification AS (
    SELECT
        CASE
            WHEN page_id IS NOT NULL
             AND current_page_id IS NULL
                THEN 'orphaned'

            WHEN page_id IS NOT NULL
             AND applied_at IS NOT NULL
             AND BTRIM(
                 COALESCE(
                     applied_page_hash,
                     ''
                 )
             ) <> ''
             AND current_page_hash <>
                 BTRIM(
                     COALESCE(
                         applied_page_hash,
                         ''
                     )
                 )
                THEN 'stale'

            WHEN applied_at IS NOT NULL
             AND BTRIM(
                 COALESCE(
                     applied_page_hash,
                     ''
                 )
             ) <> ''
                THEN 'applied'

            WHEN BTRIM(
                 COALESCE(
                     confirmed_instruction,
                     ''
                 )
             ) <> ''
                THEN 'confirmed'

            ELSE 'detected'
        END AS target_status
    FROM historical_resolution
)
SELECT
    target_status,
    COUNT(*) AS total_count
FROM classification
GROUP BY target_status
ORDER BY target_status;
