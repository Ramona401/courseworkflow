package repository

// pipeline_repo_pages.go — 生成页面 + Pipeline分配数据访问层
//
// v68变更：
//   - UpdateGeneratedPageHTML：更新前将当前HTML追加到html_history快照数组
//   - 新增RollbackPageHTML：从html_history回滚到指定版本
//   - 新增GetPageHTMLHistoryCount：获取指定页面的历史版本数量
//   - GeneratedPageFullRow新增HtmlHistoryCount字段
//
// v90-2修复：
//   - GetPrevRoundFinalHTMLMap：增加decision字段读取，reject页面返回original_html
//     原因：1审拒绝的页面，2审generator仍基于修改后版本（final_html），导致拒绝决策未生效

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// getEffectiveReviewRound 获取有效的review_round用于查询generated_pages
// 逻辑：先按当前pipeline的review_round查，如果没有数据则回退到上一轮
func getEffectiveReviewRound(ctx context.Context, pipelineID string) int {
	var reviewRound int
	_ = database.DB.QueryRow(ctx,
		`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&reviewRound)
	if reviewRound == 0 {
		return 1
	}

	var count int
	_ = database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM generated_pages WHERE pipeline_id = $1 AND review_round = $2`,
		pipelineID, reviewRound,
	).Scan(&count)

	if count > 0 {
		return reviewRound
	}

	if reviewRound > 1 {
		return reviewRound - 1
	}
	return reviewRound
}

// ==================== 生成页面类型定义 ====================

// GeneratedPageRow 生成页面查询行（不含完整HTML，只含长度）
type GeneratedPageRow struct {
	ID           string     `json:"id"`
	PipelineID   string     `json:"pipeline_id"`
	PageNumber   int        `json:"page_number"`
	PageTitle    string     `json:"page_title"`
	Operation    string     `json:"operation"`
	OrigLen      int        `json:"orig_len"`
	GenLen       int        `json:"gen_len"`
	FinalLen     int        `json:"final_len"`
	Decision     string     `json:"decision"`
	LessonID     *int       `json:"lesson_id"`
	MergeSources string     `json:"merge_sources"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// GeneratedPageFullRow 生成页面查询行（含完整HTML+修改理由+历史版本数）
// v68新增：HtmlHistoryCount字段，前端据此展示回滚按钮
type GeneratedPageFullRow struct {
	ID               string     `json:"id"`
	PipelineID       string     `json:"pipeline_id"`
	PageNumber       int        `json:"page_number"`
	PageTitle        string     `json:"page_title"`
	Operation        string     `json:"operation"`
	OriginalHTML     string     `json:"original_html"`
	GeneratedHTML    string     `json:"generated_html"`
	FinalHTML        string     `json:"final_html"`
	Decision         string     `json:"decision"`
	LessonID         *int       `json:"lesson_id"`
	MergeSources     string     `json:"merge_sources"`
	ChangeReason     string     `json:"change_reason"`
	HtmlHistoryCount int        `json:"html_history_count"`
	CreatedAt        *time.Time `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

// ==================== HTML历史快照类型 ====================

// HtmlHistoryEntry 单条HTML编辑历史快照
type HtmlHistoryEntry struct {
	Html      string `json:"html"`      // 快照时的HTML内容
	Source    string `json:"source"`    // 来源：ai_fix / manual_edit / rollback
	Timestamp string `json:"timestamp"` // ISO8601时间戳
}

// ==================== GeneratedPage CRUD ====================

// CreateGeneratedPage 创建生成页面记录
func CreateGeneratedPage(pipelineID string, pageNumber int, pageTitle string,
	operation string, originalHTML string, generatedHTML string, finalHTML string,
	lessonID *int, mergeSources string, changeReason string) error {
	ctx := context.Background()

	var reviewRound int
	_ = database.DB.QueryRow(ctx,
		`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&reviewRound)
	if reviewRound == 0 {
		reviewRound = 1
	}

	var mergeParam interface{}
	if mergeSources != "" && mergeSources != "null" {
		mergeParam = mergeSources
	}

	_, err := database.DB.Exec(ctx,
		`INSERT INTO generated_pages (pipeline_id, page_number, page_title,
			operation, original_html, generated_html, final_html,
			decision, lesson_id, merge_sources, change_reason, review_round)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9::jsonb, $10, $11)`,
		pipelineID, pageNumber, pageTitle,
		operation, originalHTML, generatedHTML, finalHTML,
		lessonID, mergeParam, changeReason, reviewRound)
	if err != nil {
		return fmt.Errorf("创建生成页面P%d失败: %w", pageNumber, err)
	}
	return nil
}

// GetGeneratedPagesByPipelineID 获取指定Pipeline的所有生成页面（不含完整HTML）
func GetGeneratedPagesByPipelineID(pipelineID string) ([]*GeneratedPageRow, error) {
	ctx := context.Background()

	reviewRound := getEffectiveReviewRound(ctx, pipelineID)

	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, page_number, page_title, operation,
			LENGTH(COALESCE(original_html,'')) as orig_len,
			LENGTH(COALESCE(generated_html,'')) as gen_len,
			LENGTH(COALESCE(final_html,'')) as final_len,
			decision, lesson_id, merge_sources::text,
			created_at, updated_at
		 FROM generated_pages
		 WHERE pipeline_id = $1 AND review_round = $2
		 ORDER BY page_number ASC`, pipelineID, reviewRound)
	if err != nil {
		return nil, fmt.Errorf("查询生成页面失败: %w", err)
	}
	defer rows.Close()

	var pages []*GeneratedPageRow
	for rows.Next() {
		p := &GeneratedPageRow{}
		var pageTitle, decision, mergeSources *string
		var lessonID *int
		err := rows.Scan(
			&p.ID, &p.PipelineID, &p.PageNumber, &pageTitle, &p.Operation,
			&p.OrigLen, &p.GenLen, &p.FinalLen,
			&decision, &lessonID, &mergeSources,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描生成页面行失败: %w", err)
		}
		if pageTitle != nil {
			p.PageTitle = *pageTitle
		}
		if decision != nil {
			p.Decision = *decision
		}
		if lessonID != nil {
			p.LessonID = lessonID
		}
		if mergeSources != nil {
			p.MergeSources = *mergeSources
		}
		pages = append(pages, p)
	}
	return pages, nil
}

// GetGeneratedPagesWithHTML 获取指定Pipeline的所有生成页面（含完整HTML+修改理由+历史版本数）
// v68增强：新增jsonb_array_length(COALESCE(html_history,'[]'))查询历史版本数
func GetGeneratedPagesWithHTML(pipelineID string) ([]*GeneratedPageFullRow, error) {
	ctx := context.Background()

	reviewRound := getEffectiveReviewRound(ctx, pipelineID)

	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, page_number, page_title, operation,
			COALESCE(original_html, '') as original_html,
			COALESCE(generated_html, '') as generated_html,
			COALESCE(final_html, '') as final_html,
			decision, lesson_id, merge_sources::text,
			COALESCE(change_reason, '') as change_reason,
			jsonb_array_length(COALESCE(html_history, '[]'::jsonb)) as html_history_count,
			created_at, updated_at
		 FROM generated_pages
		 WHERE pipeline_id = $1 AND review_round = $2
		 ORDER BY page_number ASC`, pipelineID, reviewRound)
	if err != nil {
		return nil, fmt.Errorf("查询生成页面（含HTML）失败: %w", err)
	}
	defer rows.Close()

	var pages []*GeneratedPageFullRow
	for rows.Next() {
		p := &GeneratedPageFullRow{}
		var pageTitle, decision, mergeSources *string
		var lessonID *int
		err := rows.Scan(
			&p.ID, &p.PipelineID, &p.PageNumber, &pageTitle, &p.Operation,
			&p.OriginalHTML, &p.GeneratedHTML, &p.FinalHTML,
			&decision, &lessonID, &mergeSources,
			&p.ChangeReason,
			&p.HtmlHistoryCount,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描生成页面行（含HTML）失败: %w", err)
		}
		if pageTitle != nil {
			p.PageTitle = *pageTitle
		}
		if decision != nil {
			p.Decision = *decision
		}
		if lessonID != nil {
			p.LessonID = lessonID
		}
		if mergeSources != nil {
			p.MergeSources = *mergeSources
		}
		pages = append(pages, p)
	}
	return pages, nil
}

// UpdatePageDecision 更新单页审核决策
// v68增强：edit模式下在更新前将当前HTML追加到html_history
func UpdatePageDecision(pipelineID string, pageNumber int, decision string, finalHTML *string) error {
	ctx := context.Background()
	reviewRound := getEffectiveReviewRound(ctx, pipelineID)

	if finalHTML != nil {
		// v68新增：edit保存前，将当前final_html快照到html_history
		appendHTMLHistory(ctx, pipelineID, pageNumber, reviewRound, "manual_edit")

		_, err := database.DB.Exec(ctx,
			`UPDATE generated_pages
			 SET decision = $3, final_html = $4, updated_at = NOW()
			 WHERE pipeline_id = $1 AND page_number = $2 AND review_round = $5`,
			pipelineID, pageNumber, decision, *finalHTML, reviewRound)
		if err != nil {
			return fmt.Errorf("更新页面P%d决策（含HTML）失败: %w", pageNumber, err)
		}
	} else {
		_, err := database.DB.Exec(ctx,
			`UPDATE generated_pages
			 SET decision = $3, updated_at = NOW()
			 WHERE pipeline_id = $1 AND page_number = $2 AND review_round = $4`,
			pipelineID, pageNumber, decision, reviewRound)
		if err != nil {
			return fmt.Errorf("更新页面P%d决策失败: %w", pageNumber, err)
		}
	}
	return nil
}

// GetPageDecisionStats 获取Pipeline页面审核决策统计
func GetPageDecisionStats(pipelineID string) (total int, decided int, err error) {
	ctx := context.Background()

	reviewRound := getEffectiveReviewRound(ctx, pipelineID)

	err = database.DB.QueryRow(ctx,
		`SELECT COUNT(*),
			COUNT(*) FILTER (WHERE decision IN ('approve', 'reject', 'edit'))
		 FROM generated_pages gp
		 WHERE gp.pipeline_id = $1
		   AND gp.review_round = $2
		   AND NOT (
			 gp.page_number >= 1000
			 AND EXISTS (
				 SELECT 1 FROM generated_pages gp2
				 WHERE gp2.pipeline_id = gp.pipeline_id
				   AND gp2.page_number = gp.page_number % 1000
				   AND gp2.page_number < 1000
				   AND gp2.review_round = gp.review_round
			 )
		   )`, pipelineID, reviewRound).Scan(&total, &decided)
	if err != nil {
		return 0, 0, fmt.Errorf("查询页面决策统计失败: %w", err)
	}
	return total, decided, nil
}

// DeleteGeneratedPagesByPipelineID 删除指定Pipeline当前review_round的所有生成页面
func DeleteGeneratedPagesByPipelineID(pipelineID string) error {
	ctx := context.Background()
	var reviewRound int
	_ = database.DB.QueryRow(ctx,
		`SELECT review_round FROM pipelines WHERE id = $1`, pipelineID,
	).Scan(&reviewRound)
	if reviewRound == 0 {
		reviewRound = 1
	}
	_, err := database.DB.Exec(ctx,
		`DELETE FROM generated_pages WHERE pipeline_id = $1 AND review_round = $2`,
		pipelineID, reviewRound)
	if err != nil {
		return fmt.Errorf("删除生成页面失败: %w", err)
	}
	return nil
}

// UpdateGeneratedPageHTML 更新指定页面的generated_html和final_html
// v68增强：更新前将当前HTML追加到html_history快照数组，支持后续回滚
func UpdateGeneratedPageHTML(pipelineID string, pageNumber int, generatedHTML string, finalHTML string) error {
	ctx := context.Background()
	reviewRound := getEffectiveReviewRound(ctx, pipelineID)

	// v68新增：AI快修前，将当前final_html快照到html_history
	appendHTMLHistory(ctx, pipelineID, pageNumber, reviewRound, "ai_fix")

	_, err := database.DB.Exec(ctx,
		`UPDATE generated_pages
		 SET generated_html = $3, final_html = $4, updated_at = NOW()
		 WHERE pipeline_id = $1 AND page_number = $2 AND review_round = $5`,
		pipelineID, pageNumber, generatedHTML, finalHTML, reviewRound)
	if err != nil {
		return fmt.Errorf("更新页面P%d的HTML失败: %w", pageNumber, err)
	}
	return nil
}

// ==================== HTML历史快照方法（v68新增）====================

// appendHTMLHistory 将当前页面的final_html追加到html_history数组
// 如果当前没有final_html（为空），则尝试保存generated_html
// source参数标识来源：ai_fix / manual_edit / rollback
func appendHTMLHistory(ctx context.Context, pipelineID string, pageNumber int, reviewRound int, source string) {
	// 读取当前的final_html和generated_html
	var finalHTML, genHTML *string
	_ = database.DB.QueryRow(ctx,
		`SELECT final_html, generated_html FROM generated_pages
		 WHERE pipeline_id = $1 AND page_number = $2 AND review_round = $3`,
		pipelineID, pageNumber, reviewRound,
	).Scan(&finalHTML, &genHTML)

	// 确定要保存的HTML（优先final_html，没有则用generated_html）
	var htmlToSave string
	if finalHTML != nil && *finalHTML != "" {
		htmlToSave = *finalHTML
	} else if genHTML != nil && *genHTML != "" {
		htmlToSave = *genHTML
	}

	// 如果没有可保存的HTML，跳过
	if htmlToSave == "" {
		return
	}

	// 构建快照条目
	entry := HtmlHistoryEntry{
		Html:      htmlToSave,
		Source:    source,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return
	}

	// 追加到html_history数组（使用jsonb拼接，最多保留20个版本）
	// 先追加，再截取最新的20个
	_, _ = database.DB.Exec(ctx,
		`UPDATE generated_pages
		 SET html_history = (
		     SELECT CASE
			 WHEN jsonb_array_length(COALESCE(html_history, '[]'::jsonb) || $4::jsonb) > 20
			 THEN (COALESCE(html_history, '[]'::jsonb) || $4::jsonb) - 0
			 ELSE COALESCE(html_history, '[]'::jsonb) || $4::jsonb
		     END
		 ),
		 updated_at = NOW()
		 WHERE pipeline_id = $1 AND page_number = $2 AND review_round = $3`,
		pipelineID, pageNumber, reviewRound, string(entryJSON))
}

// RollbackPageHTML 回滚页面HTML到上一个历史版本
// 将html_history数组最后一条取出，恢复为当前的final_html和generated_html
// 同时从数组中移除该条目
// 返回回滚后的HTML内容和剩余历史版本数
func RollbackPageHTML(pipelineID string, pageNumber int) (string, int, error) {
	ctx := context.Background()
	reviewRound := getEffectiveReviewRound(ctx, pipelineID)

	// 读取当前html_history
	var historyJSON *string
	err := database.DB.QueryRow(ctx,
		`SELECT html_history::text FROM generated_pages
		 WHERE pipeline_id = $1 AND page_number = $2 AND review_round = $3`,
		pipelineID, pageNumber, reviewRound,
	).Scan(&historyJSON)
	if err != nil {
		return "", 0, fmt.Errorf("读取页面P%d历史失败: %w", pageNumber, err)
	}

	if historyJSON == nil || *historyJSON == "" || *historyJSON == "[]" || *historyJSON == "null" {
		return "", 0, fmt.Errorf("页面P%d没有可回滚的历史版本", pageNumber)
	}

	// 解析历史数组
	var history []HtmlHistoryEntry
	if err := json.Unmarshal([]byte(*historyJSON), &history); err != nil {
		return "", 0, fmt.Errorf("解析页面P%d历史数据失败: %w", pageNumber, err)
	}

	if len(history) == 0 {
		return "", 0, fmt.Errorf("页面P%d没有可回滚的历史版本", pageNumber)
	}

	// 取出最后一条（最近的快照）
	lastEntry := history[len(history)-1]
	remaining := history[:len(history)-1]

	// 序列化剩余历史
	remainingJSON, err := json.Marshal(remaining)
	if err != nil {
		return "", 0, fmt.Errorf("序列化历史数据失败: %w", err)
	}

	// 更新：将最后一条快照恢复为final_html和generated_html，同时更新html_history
	_, err = database.DB.Exec(ctx,
		`UPDATE generated_pages
		 SET generated_html = $4, final_html = $4,
		     html_history = $5::jsonb,
		     updated_at = NOW()
		 WHERE pipeline_id = $1 AND page_number = $2 AND review_round = $3`,
		pipelineID, pageNumber, reviewRound, lastEntry.Html, string(remainingJSON))
	if err != nil {
		return "", 0, fmt.Errorf("回滚页面P%d失败: %w", pageNumber, err)
	}

	return lastEntry.Html, len(remaining), nil
}

// ==================== Pipeline分配方法 ====================

// ListFinalizedPipelineIDs 获取所有finalized状态的Pipeline ID列表
func ListFinalizedPipelineIDs() ([]string, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id FROM pipelines WHERE status = $1 ORDER BY created_at ASC`,
		models.PipelineStatusFinalized)
	if err != nil {
		return nil, fmt.Errorf("查询finalized Pipeline列表失败: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("扫描Pipeline ID失败: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// AssignPipeline 分配Pipeline给指定审核员
func AssignPipeline(pipelineID string, assignedTo *string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE pipelines SET assigned_to = $2, updated_at = NOW()
		 WHERE id = $1`, pipelineID, assignedTo)
	if err != nil {
		return fmt.Errorf("分配Pipeline失败: %w", err)
	}
	return nil
}

// BatchAssignPipelines 批量分配Pipeline给指定审核员
func BatchAssignPipelines(pipelineIDs []string, assignedTo *string) (int, error) {
	if len(pipelineIDs) == 0 {
		return 0, nil
	}
	ctx := context.Background()
	result, err := database.DB.Exec(ctx,
		`UPDATE pipelines
		 SET assigned_to = $2, updated_at = NOW()
		 WHERE id = ANY($1)`,
		pipelineIDs, assignedTo)
	if err != nil {
		return 0, fmt.Errorf("批量分配Pipeline失败: %w", err)
	}
	return int(result.RowsAffected()), nil
}

// ListOperatorUsers 获取所有活跃的operator/admin用户列表
func ListOperatorUsers() ([]map[string]string, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, username, display_name, role
		 FROM users
		 WHERE status = 'active' AND role IN ('admin', 'operator', 'senior_operator')
		 ORDER BY role ASC, display_name ASC`)
	if err != nil {
		return nil, fmt.Errorf("查询审核员列表失败: %w", err)
	}
	defer rows.Close()

	var result []map[string]string
	for rows.Next() {
		var id, username, displayName, role string
		if err := rows.Scan(&id, &username, &displayName, &role); err != nil {
			continue
		}
		result = append(result, map[string]string{
			"id": id, "username": username,
			"display_name": displayName, "role": role,
		})
	}
	if result == nil {
		result = []map[string]string{}
	}
	return result, nil
}

// ==================== 2审辅助：上一轮定稿HTML映射 ====================

// GetPrevRoundFinalHTMLMap 获取上一轮所有页面的定稿HTML映射
//
// v90-2修复：增加decision字段读取，根据审核决策决定返回哪个HTML版本
//   - decision='reject' → 返回original_html（审核员拒绝了AI修改，应回退到原始版本）
//   - decision='approve' → 返回generated_html优先，降级final_html
//   - decision='edit' → 返回final_html（审核员手动编辑的版本）
//   - 其他（pending等）→ 保持原有逻辑：final_html > generated_html > original_html
//
// 原始问题：该函数不读取decision字段，对所有页面统一按final_html>generated_html>original_html
// 取HTML，导致1审拒绝的页面在2审时仍然基于AI修改后的版本进行处理
func GetPrevRoundFinalHTMLMap(pipelineID string, prevRound int) map[int]string {
	ctx := context.Background()
	result := make(map[int]string)

	rows, err := database.DB.Query(ctx,
		`SELECT page_number,
			COALESCE(final_html, '') as final_html,
			COALESCE(generated_html, '') as generated_html,
			COALESCE(original_html, '') as original_html,
			COALESCE(decision, 'pending') as decision
		 FROM generated_pages
		 WHERE pipeline_id = $1 AND review_round = $2
		 ORDER BY page_number ASC`, pipelineID, prevRound)
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var pageNum int
		var finalHTML, genHTML, origHTML, decision string
		if err := rows.Scan(&pageNum, &finalHTML, &genHTML, &origHTML, &decision); err != nil {
			continue
		}

		// v90-2修复：根据审核决策决定返回哪个HTML版本
		var html string
		switch decision {
		case "reject":
			// 审核员拒绝了AI修改，应回退到原始版本
			// 这样2审的generator会基于原始课件重新修改，而不是基于被拒绝的AI修改版本
			html = origHTML
			if html == "" {
				html = finalHTML
			}
		case "approve":
			// v99修复Bug5：审核员批准了AI修改，优先使用finalHTML
			// 原因：打回重审后operator可能编辑了finalHTML，此时应使用编辑后版本
			html = finalHTML
			if html == "" {
				html = genHTML
			}
			if html == "" {
				html = origHTML
			}
		case "edit":
			// 审核员手动编辑，使用编辑后版本
			html = finalHTML
			if html == "" {
				html = genHTML
			}
			if html == "" {
				html = origHTML
			}
		default:
			// pending或其他状态，保持原有逻辑
			html = finalHTML
			if html == "" {
				html = genHTML
			}
			if html == "" {
				html = origHTML
			}
		}

		if len(html) > 100 {
			result[pageNum] = html
		}
	}
	return result
}

// ==================== 单页HTML按需加载（v69新增，编号8方案2）====================

// GetSinglePageHTML 获取指定Pipeline指定页码的单页完整HTML数据
// v69新增：审核页HTML懒加载——前端不再一次加载所有页面的完整HTML，
// 而是先加载轻量元数据列表，选中页面时再按需加载单页HTML
// 返回：单页的完整HTML数据（含original_html/generated_html/final_html/change_reason/html_history_count）
func GetSinglePageHTML(pipelineID string, pageNumber int) (*GeneratedPageFullRow, error) {
	ctx := context.Background()
	reviewRound := getEffectiveReviewRound(ctx, pipelineID)

	p := &GeneratedPageFullRow{}
	var pageTitle, decision, mergeSources *string
	var lessonID *int

	err := database.DB.QueryRow(ctx,
		`SELECT id, pipeline_id, page_number, page_title, operation,
			COALESCE(original_html, '') as original_html,
			COALESCE(generated_html, '') as generated_html,
			COALESCE(final_html, '') as final_html,
			decision, lesson_id, merge_sources::text,
			COALESCE(change_reason, '') as change_reason,
			jsonb_array_length(COALESCE(html_history, '[]'::jsonb)) as html_history_count,
			created_at, updated_at
		 FROM generated_pages
		 WHERE pipeline_id = $1 AND page_number = $2 AND review_round = $3
		 LIMIT 1`,
		pipelineID, pageNumber, reviewRound,
	).Scan(
		&p.ID, &p.PipelineID, &p.PageNumber, &pageTitle, &p.Operation,
		&p.OriginalHTML, &p.GeneratedHTML, &p.FinalHTML,
		&decision, &lessonID, &mergeSources,
		&p.ChangeReason,
		&p.HtmlHistoryCount,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("页面P%d不存在: %w", pageNumber, err)
	}

	if pageTitle != nil {
		p.PageTitle = *pageTitle
	}
	if decision != nil {
		p.Decision = *decision
	}
	if lessonID != nil {
		p.LessonID = lessonID
	}
	if mergeSources != nil {
		p.MergeSources = *mergeSources
	}
	return p, nil
}

// GetGeneratedPagesLightweight 获取指定Pipeline的所有页面轻量元数据（不含完整HTML）
// v69新增：审核页首次加载时只获取元数据，减少传输数据量
// 返回HTML内容长度而非完整HTML，前端可根据此判断页面是否有内容
func GetGeneratedPagesLightweight(pipelineID string) ([]*GeneratedPageFullRow, error) {
	ctx := context.Background()
	reviewRound := getEffectiveReviewRound(ctx, pipelineID)

	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, page_number, page_title, operation,
			'' as original_html,
			'' as generated_html,
			'' as final_html,
			decision, lesson_id, merge_sources::text,
			COALESCE(change_reason, '') as change_reason,
			jsonb_array_length(COALESCE(html_history, '[]'::jsonb)) as html_history_count,
			created_at, updated_at
		 FROM generated_pages
		 WHERE pipeline_id = $1 AND review_round = $2
		 ORDER BY page_number ASC`, pipelineID, reviewRound)
	if err != nil {
		return nil, fmt.Errorf("查询页面元数据失败: %w", err)
	}
	defer rows.Close()

	var pages []*GeneratedPageFullRow
	for rows.Next() {
		p := &GeneratedPageFullRow{}
		var pageTitle, decision, mergeSources *string
		var lessonID *int
		err := rows.Scan(
			&p.ID, &p.PipelineID, &p.PageNumber, &pageTitle, &p.Operation,
			&p.OriginalHTML, &p.GeneratedHTML, &p.FinalHTML,
			&decision, &lessonID, &mergeSources,
			&p.ChangeReason,
			&p.HtmlHistoryCount,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描页面元数据行失败: %w", err)
		}
		if pageTitle != nil {
			p.PageTitle = *pageTitle
		}
		if decision != nil {
			p.Decision = *decision
		}
		if lessonID != nil {
			p.LessonID = lessonID
		}
		if mergeSources != nil {
			p.MergeSources = *mergeSources
		}
		pages = append(pages, p)
	}
	return pages, nil
}
