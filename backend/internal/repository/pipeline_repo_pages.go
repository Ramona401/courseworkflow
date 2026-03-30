package repository

// pipeline_repo_pages.go — 生成页面 + Pipeline分配数据访问层
//
// 职责：
//   - GeneratedPage CRUD（创建/查询/决策更新/HTML更新/删除）
//   - Pipeline分配（单个/批量/审核员列表）
//   - 相关类型定义（GeneratedPageRow/GeneratedPageFullRow）

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

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

// GeneratedPageFullRow 生成页面查询行（含完整HTML+修改理由）
type GeneratedPageFullRow struct {
	ID            string     `json:"id"`
	PipelineID    string     `json:"pipeline_id"`
	PageNumber    int        `json:"page_number"`
	PageTitle     string     `json:"page_title"`
	Operation     string     `json:"operation"`
	OriginalHTML  string     `json:"original_html"`
	GeneratedHTML string     `json:"generated_html"`
	FinalHTML     string     `json:"final_html"`
	Decision      string     `json:"decision"`
	LessonID      *int       `json:"lesson_id"`
	MergeSources  string     `json:"merge_sources"`
	ChangeReason  string     `json:"change_reason"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// ==================== GeneratedPage CRUD ====================

// CreateGeneratedPage 创建生成页面记录
func CreateGeneratedPage(pipelineID string, pageNumber int, pageTitle string,
	operation string, originalHTML string, generatedHTML string, finalHTML string,
	lessonID *int, mergeSources string, changeReason string) error {
	ctx := context.Background()

	var mergeParam interface{}
	if mergeSources != "" && mergeSources != "null" {
		mergeParam = mergeSources
	}

	_, err := database.DB.Exec(ctx,
		`INSERT INTO generated_pages (pipeline_id, page_number, page_title,
		        operation, original_html, generated_html, final_html,
		        decision, lesson_id, merge_sources, change_reason)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9::jsonb, $10)`,
		pipelineID, pageNumber, pageTitle,
		operation, originalHTML, generatedHTML, finalHTML,
		lessonID, mergeParam, changeReason)
	if err != nil {
		return fmt.Errorf("创建生成页面P%d失败: %w", pageNumber, err)
	}
	return nil
}

// GetGeneratedPagesByPipelineID 获取指定Pipeline的所有生成页面（不含完整HTML，只含长度）
func GetGeneratedPagesByPipelineID(pipelineID string) ([]*GeneratedPageRow, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, page_number, page_title, operation,
		        LENGTH(COALESCE(original_html,'')) as orig_len,
		        LENGTH(COALESCE(generated_html,'')) as gen_len,
		        LENGTH(COALESCE(final_html,'')) as final_len,
		        decision, lesson_id, merge_sources::text,
		        created_at, updated_at
		 FROM generated_pages
		 WHERE pipeline_id = $1
		 ORDER BY page_number ASC`, pipelineID)
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

// GetGeneratedPagesWithHTML 获取指定Pipeline的所有生成页面（含完整HTML+修改理由）
// 审核页面需要完整HTML用于预览和对比
func GetGeneratedPagesWithHTML(pipelineID string) ([]*GeneratedPageFullRow, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, pipeline_id, page_number, page_title, operation,
		        COALESCE(original_html, '') as original_html,
		        COALESCE(generated_html, '') as generated_html,
		        COALESCE(final_html, '') as final_html,
		        decision, lesson_id, merge_sources::text,
		        COALESCE(change_reason, '') as change_reason,
		        created_at, updated_at
		 FROM generated_pages
		 WHERE pipeline_id = $1
		 ORDER BY page_number ASC`, pipelineID)
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
// edit模式：同时更新final_html；approve/reject模式：只更新decision
func UpdatePageDecision(pipelineID string, pageNumber int, decision string, finalHTML *string) error {
	ctx := context.Background()

	if finalHTML != nil {
		_, err := database.DB.Exec(ctx,
			`UPDATE generated_pages
			 SET decision = $3, final_html = $4, updated_at = NOW()
			 WHERE pipeline_id = $1 AND page_number = $2`,
			pipelineID, pageNumber, decision, *finalHTML)
		if err != nil {
			return fmt.Errorf("更新页面P%d决策（含HTML）失败: %w", pageNumber, err)
		}
	} else {
		_, err := database.DB.Exec(ctx,
			`UPDATE generated_pages
			 SET decision = $3, updated_at = NOW()
			 WHERE pipeline_id = $1 AND page_number = $2`,
			pipelineID, pageNumber, decision)
		if err != nil {
			return fmt.Errorf("更新页面P%d决策失败: %w", pageNumber, err)
		}
	}
	return nil
}

// GetPageDecisionStats 获取Pipeline页面审核决策统计
// 排除被前端合并的虚拟页面（page_number >= 1000 且同位置有普通页面）
func GetPageDecisionStats(pipelineID string) (total int, decided int, err error) {
	ctx := context.Background()
	err = database.DB.QueryRow(ctx,
		`SELECT COUNT(*),
		        COUNT(*) FILTER (WHERE decision IN ('approve', 'reject', 'edit'))
		 FROM generated_pages gp
		 WHERE gp.pipeline_id = $1
		   AND NOT (
		         gp.page_number >= 1000
		         AND EXISTS (
		                 SELECT 1 FROM generated_pages gp2
		                 WHERE gp2.pipeline_id = gp.pipeline_id
		                   AND gp2.page_number = gp.page_number % 1000
		                   AND gp2.page_number < 1000
		         )
		   )`, pipelineID).Scan(&total, &decided)
	if err != nil {
		return 0, 0, fmt.Errorf("查询页面决策统计失败: %w", err)
	}
	return total, decided, nil
}

// DeleteGeneratedPagesByPipelineID 删除指定Pipeline的所有生成页面（重跑时清理）
func DeleteGeneratedPagesByPipelineID(pipelineID string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`DELETE FROM generated_pages WHERE pipeline_id = $1`, pipelineID)
	if err != nil {
		return fmt.Errorf("删除生成页面失败: %w", err)
	}
	return nil
}

// UpdateGeneratedPageHTML 更新指定页面的generated_html和final_html
// AI快修功能：审核员输入修改指令后AI重新生成HTML
func UpdateGeneratedPageHTML(pipelineID string, pageNumber int, generatedHTML string, finalHTML string) error {
	ctx := context.Background()
	_, err := database.DB.Exec(ctx,
		`UPDATE generated_pages
		 SET generated_html = $3, final_html = $4, updated_at = NOW()
		 WHERE pipeline_id = $1 AND page_number = $2`,
		pipelineID, pageNumber, generatedHTML, finalHTML)
	if err != nil {
		return fmt.Errorf("更新页面P%d的HTML失败: %w", pageNumber, err)
	}
	return nil
}

// ==================== Pipeline分配方法 ====================

// ListFinalizedPipelineIDs 获取所有finalized状态的Pipeline ID列表
// 夜间批量验收和手动批量验收使用，按创建时间正序（先创建的先验收）
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
// 修复E-04：单条SQL WHERE id = ANY($1)原子执行，消除逐条更新的事务缺失问题
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

// ListOperatorUsers 获取所有活跃的operator/admin用户列表（供分配审核员选择）
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
