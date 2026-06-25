package repository

// courseware_page_version_repo.go — 课件页面级版本快照数据访问层（页面级版本与回退·新建）
//
// 配套表：courseware_page_versions（每次"覆盖式修改 html_content"前存一份旧版本快照）。
// 复用模板 history/rollback 已验证的 database.DB.QueryRow/Exec 范式（非显式事务）。
//
// 提供 4 个函数：
//   - CreatePageVersion  存一版（内部算 version_no = 当前最大+1；并裁剪超过20版的最老记录）
//   - ListPageVersions   按 version_no 倒序返回该页所有版本（列表不带 html_content，省流量）
//   - GetPageVersion     取单个版本的完整 html_content（点"预览"/"回退"时单独取）
//   - CountPageVersions  计数（裁剪用，亦供调试）
//
// 设计要点：
//   - version_no 每页独立从 1 递增（不是全局递增）。
//   - html_content 存完整快照（非 diff），单页几十 KB，20 版上限可控。
//   - 删页/删课件时版本由表上的 ON DELETE CASCADE 外键自动清理，无悬挂数据。

import (
        "context"
        "fmt"

        "tedna/internal/database"
        "tedna/internal/models"
)

// cwPageVersionMaxKeep 每页最多保留的版本数（与模板 history 的"保留近20版"策略一致，防表膨胀）
const cwPageVersionMaxKeep = 20

// CreatePageVersion 为指定页存一份 html_content 版本快照。
//
// 内部流程：
//  1. 算出该页当前最大 version_no，新版本号 = max+1（每页独立从1起）。
//  2. INSERT 一行（返回 id / version_no / created_at）。
//  3. 若该页版本数 > 20，删除最老的若干条（version_no 最小者），保持上限。
//
// 参数：
//
//	pageID/coursewareID —— 归属页与归属课件（courseware_id 冗余存，便于按课件批量查清）
//	html               —— 要快照的旧版 HTML（调用方在覆盖前传入 page.HTMLContent 旧值）
//	source             —— 版本来源枚举（refine/regenerate/rollback/... 见 models.CWPageVersionSource*）
//	note               —— 可选备注（如微调指令、重生说明、回退说明），空串存 NULL
//
// 返回：写入后的版本实体（含 ID/VersionNo/CreatedAt）。
func CreatePageVersion(ctx context.Context, pageID string, coursewareID string, html string, source string, note string) (*models.CoursewarePageVersion, error) {
        // 1. 计算新版本号 = 该页当前最大 version_no + 1（无记录时为 1）
        var nextNo int
        err := database.DB.QueryRow(ctx,
                `SELECT COALESCE(MAX(version_no), 0) + 1 FROM courseware_page_versions WHERE page_id = $1`,
                pageID,
        ).Scan(&nextNo)
        if err != nil {
                return nil, fmt.Errorf("计算页面版本号失败: %w", err)
        }

        // 2. 插入新版本快照
        v := &models.CoursewarePageVersion{
                PageID:       pageID,
                CoursewareID: coursewareID,
                VersionNo:    nextNo,
                HTMLContent:  html,
                Source:       source,
                Note:         note,
        }
        insertSQL := `INSERT INTO courseware_page_versions
                (id, page_id, courseware_id, version_no, html_content, source, note)
                VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
                RETURNING id, created_at`
        if err := database.DB.QueryRow(ctx, insertSQL,
                pageID, coursewareID, nextNo, html, source, nullIfEmpty(note),
        ).Scan(&v.ID, &v.CreatedAt); err != nil {
                return nil, fmt.Errorf("写入页面版本快照失败: %w", err)
        }

        // 3. 裁剪：超过上限时删除最老的若干条（version_no 最小者）
        //    用子查询选出"保留最近 N 条之外"的全部 id 删除，一条 SQL 完成，幂等安全。
        trimSQL := `DELETE FROM courseware_page_versions
                WHERE page_id = $1
                  AND id NOT IN (
                        SELECT id FROM courseware_page_versions
                        WHERE page_id = $1
                        ORDER BY version_no DESC
                        LIMIT $2
                  )`
        if _, err := database.DB.Exec(ctx, trimSQL, pageID, cwPageVersionMaxKeep); err != nil {
                // 裁剪失败不影响本次存版主流程（版本已写入），仅记日志由上层决定；这里返回成功的版本实体。
                // 不返回错误：保留版本比"严格控制条数"更重要，下次写入会再尝试裁剪。
                return v, nil
        }
        return v, nil
}

// ListPageVersions 返回某页的全部版本（按 version_no 倒序，最新在前）。
//
// 注意：为省流量，列表不带 html_content（该列在 SQL 中不查）。
// 前端点"预览"或"回退"时再用 GetPageVersion 单独取完整 HTML。
func ListPageVersions(ctx context.Context, pageID string) ([]*models.CoursewarePageVersionListItem, error) {
        sql := `SELECT id, version_no, source, COALESCE(note, ''), created_at
                FROM courseware_page_versions
                WHERE page_id = $1
                ORDER BY version_no DESC`
        rows, err := database.DB.Query(ctx, sql, pageID)
        if err != nil {
                return nil, fmt.Errorf("查询页面版本列表失败: %w", err)
        }
        defer rows.Close()

        var items []*models.CoursewarePageVersionListItem
        for rows.Next() {
                it := &models.CoursewarePageVersionListItem{}
                if err := rows.Scan(&it.ID, &it.VersionNo, &it.Source, &it.Note, &it.CreatedAt); err != nil {
                        return nil, fmt.Errorf("扫描页面版本行失败: %w", err)
                }
                items = append(items, it)
        }
        return items, nil
}

// GetPageVersion 取单个版本的完整记录（含 html_content），供"预览大图"与"回退"使用。
func GetPageVersion(ctx context.Context, versionID string) (*models.CoursewarePageVersion, error) {
        sql := `SELECT id, page_id, courseware_id, version_no, html_content, source, COALESCE(note, ''), created_at
                FROM courseware_page_versions
                WHERE id = $1`
        v := &models.CoursewarePageVersion{}
        err := database.DB.QueryRow(ctx, sql, versionID).Scan(
                &v.ID, &v.PageID, &v.CoursewareID, &v.VersionNo,
                &v.HTMLContent, &v.Source, &v.Note, &v.CreatedAt,
        )
        if err != nil {
                return nil, err
        }
        return v, nil
}

// CountPageVersions 统计某页的版本数（裁剪逻辑/调试用）。
func CountPageVersions(ctx context.Context, pageID string) (int, error) {
        var count int
        err := database.DB.QueryRow(ctx,
                `SELECT COUNT(*) FROM courseware_page_versions WHERE page_id = $1`,
                pageID,
        ).Scan(&count)
        return count, err
}
