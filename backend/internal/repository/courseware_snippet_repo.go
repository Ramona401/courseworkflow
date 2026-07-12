package repository

// courseware_snippet_repo.go — 【代码收藏库·批次C新增】代码收藏数据访问层
//
// 配套表：courseware_code_snippets（老师打星收藏的课件页HTML快照，纯个人归属）。
// 镜像 courseware_page_version_repo.go 已验证的 database.DB.QueryRow/Exec 范式（非显式事务）。
//
// 提供 4 个函数：
//   - CreateCodeSnippet       收藏一条（含每用户 200 条上限保护，超限报错提示清理）
//   - ListCodeSnippetsByUser  按用户返回收藏列表（倒序，不含 html_content 全文，附字节数）
//   - GetCodeSnippet          取单条完整记录（含 HTML 全文，注入/预览时用）
//   - DeleteCodeSnippet       删除（WHERE id AND user_id 双条件，天然防越权删他人收藏）
//
// 设计要点：
//   - user_id/source_courseware_id 为 TEXT 且无外键：收藏是快照资产，
//     源课件/源页删除后收藏依然有效（溯源字段仅供展示）。
//   - 上限 200 条/用户：收藏是用户主动行为，不做自动裁剪（与版本快照的静默裁剪不同），
//     超限明确报错让老师自己清理，避免悄悄删掉老师的收藏。

import (
        "context"
        "fmt"

        "tedna/internal/database"
        "tedna/internal/models"
)

// cwSnippetMaxPerUser 每用户最多保留的收藏条数（防表膨胀；超限报错而非自动删除）
const cwSnippetMaxPerUser = 200

// CreateCodeSnippet 为指定用户新增一条代码收藏。
//
// 内部流程：
//  1. 统计该用户当前收藏数，达到上限（200）直接报错（提示清理，不自动删旧）。
//  2. INSERT 一行（返回 id / created_at）。
//
// 参数：
//
//	userID       —— 归属用户
//	title/note   —— 收藏名称与可选备注（note 空串存 NULL）
//	html         —— 页面完整 HTML 快照（调用方已校验非空）
//	srcCwID      —— 溯源课件ID（可为空串，存 NULL）
//	srcPageNum   —— 溯源页码
//
// 返回：写入后的完整实体（含 ID/CreatedAt）。
func CreateCodeSnippet(ctx context.Context, userID string, title string, note string, html string, srcCwID string, srcPageNum int) (*models.CoursewareCodeSnippet, error) {
        // 1. 上限保护
        var cnt int
        if err := database.DB.QueryRow(ctx,
                `SELECT COUNT(*) FROM courseware_code_snippets WHERE user_id = $1`,
                userID,
        ).Scan(&cnt); err != nil {
                return nil, fmt.Errorf("统计收藏数量失败: %w", err)
        }
        if cnt >= cwSnippetMaxPerUser {
                return nil, fmt.Errorf("代码收藏已达上限（%d条），请先删除一些不用的收藏", cwSnippetMaxPerUser)
        }

        // 2. 插入新收藏
        s := &models.CoursewareCodeSnippet{
                UserID:             userID,
                Title:              title,
                Note:               note,
                HTMLContent:        html,
                SourceCoursewareID: srcCwID,
                SourcePageNumber:   srcPageNum,
        }
        insertSQL := `INSERT INTO courseware_code_snippets
                (id, user_id, title, note, html_content, source_courseware_id, source_page_number)
                VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
                RETURNING id, created_at`
        if err := database.DB.QueryRow(ctx, insertSQL,
                userID, title, nullIfEmpty(note), html, nullIfEmpty(srcCwID), srcPageNum,
        ).Scan(&s.ID, &s.CreatedAt); err != nil {
                return nil, fmt.Errorf("写入代码收藏失败: %w", err)
        }
        return s, nil
}

// ListCodeSnippetsByUser 返回指定用户的全部收藏（按收藏时间倒序，最新在前）。
//
// 为省流量，列表不带 html_content 全文，只带 LENGTH(html_content) 字节数供前端展示体量；
// 前端"预览"或"注入微调"时再用 GetCodeSnippet 按 id 单独取全文。
func ListCodeSnippetsByUser(ctx context.Context, userID string) ([]*models.CoursewareCodeSnippetListItem, error) {
        sql := `SELECT id, title, COALESCE(note, ''), LENGTH(html_content),
                       COALESCE(source_courseware_id, ''), source_page_number, created_at
                FROM courseware_code_snippets
                WHERE user_id = $1
                ORDER BY created_at DESC`
        rows, err := database.DB.Query(ctx, sql, userID)
        if err != nil {
                return nil, fmt.Errorf("查询代码收藏列表失败: %w", err)
        }
        defer rows.Close()

        var items []*models.CoursewareCodeSnippetListItem
        for rows.Next() {
                it := &models.CoursewareCodeSnippetListItem{}
                if err := rows.Scan(&it.ID, &it.Title, &it.Note, &it.HTMLLen,
                        &it.SourceCoursewareID, &it.SourcePageNumber, &it.CreatedAt); err != nil {
                        return nil, fmt.Errorf("扫描代码收藏行失败: %w", err)
                }
                items = append(items, it)
        }
        return items, nil
}

// GetCodeSnippet 取单条收藏的完整记录（含 HTML 全文），供"预览"与"注入微调"使用。
// 归属校验（是否本人的收藏）由调用方比对返回的 UserID 完成。
func GetCodeSnippet(ctx context.Context, snippetID string) (*models.CoursewareCodeSnippet, error) {
        sql := `SELECT id, user_id, title, COALESCE(note, ''), html_content,
                       COALESCE(source_courseware_id, ''), source_page_number, created_at
                FROM courseware_code_snippets
                WHERE id = $1`
        s := &models.CoursewareCodeSnippet{}
        err := database.DB.QueryRow(ctx, sql, snippetID).Scan(
                &s.ID, &s.UserID, &s.Title, &s.Note, &s.HTMLContent,
                &s.SourceCoursewareID, &s.SourcePageNumber, &s.CreatedAt,
        )
        if err != nil {
                return nil, err
        }
        return s, nil
}

// DeleteCodeSnippet 删除指定收藏。WHERE 同时带 id 与 user_id 双条件——
// 即使拿到他人收藏的 id 也删不动（天然防越权），返回是否真的删掉了一行。
func DeleteCodeSnippet(ctx context.Context, snippetID string, userID string) (bool, error) {
        tag, err := database.DB.Exec(ctx,
                `DELETE FROM courseware_code_snippets WHERE id = $1 AND user_id = $2`,
                snippetID, userID,
        )
        if err != nil {
                return false, fmt.Errorf("删除代码收藏失败: %w", err)
        }
        return tag.RowsAffected() > 0, nil
}
