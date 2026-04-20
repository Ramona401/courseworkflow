package repository

// assistant_feedback_repo.go — AI 助手反馈数据访问层
//
// 职责:
//   - 写入:CreateAssistantFeedback(老师点 👍/👎 的主要入口)
//   - 查询:GetAssistantFeedbackStats(单个助手的统计)
//   - 列表:ListAssistantFeedback(admin 后台查看,支持多维过滤+分页)
//   - 删除:DeleteAssistantFeedback(老师可删自己的反馈)
//
// 事务与并发:
//   - 反馈写入是独立事件,不需要事务
//   - 并发写入由数据库行级锁自然处理
//   - 不对同一(user,assistant)做去重约束(同一老师可对同一助手多次反馈)

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrFeedbackNotFound = errors.New("反馈记录不存在")
)

// ==================== 创建 ====================

// CreateAssistantFeedback 写入一条反馈
//
// 调用方:handler 层(老师点 👍/👎 触发)
// 返回填充好 ID 和 CreatedAt 的实体
func CreateAssistantFeedback(ctx context.Context, f *models.AssistantFeedback) error {
	query := `
		INSERT INTO assistant_feedback (
			assistant_id, user_id, rating, comment,
			scene_code, lesson_plan_id, stage_code, ai_response_preview,
			trace_id
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9
		)
		RETURNING id, created_at
	`
	err := database.DB.QueryRow(ctx, query,
		f.AssistantID, f.UserID, f.Rating, f.Comment,
		f.SceneCode, f.LessonPlanID, f.StageCode, f.AIResponsePreview,
		f.TraceID,
	).Scan(&f.ID, &f.CreatedAt)
	if err != nil {
		return fmt.Errorf("创建反馈失败: %w", err)
	}
	return nil
}

// ==================== 查询单条(用于删除前校验所有权) ====================

// GetAssistantFeedbackByID 按 ID 获取反馈(不做权限校验)
//
// 调用方:service 层在删除前先查出来,校验 user_id 是否是当前用户
func GetAssistantFeedbackByID(ctx context.Context, id string) (*models.AssistantFeedback, error) {
	f := &models.AssistantFeedback{}
	query := `
		SELECT id, assistant_id, user_id, rating, comment,
		       scene_code, lesson_plan_id, stage_code, ai_response_preview,
		       trace_id, created_at
		FROM assistant_feedback
		WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.AssistantID, &f.UserID, &f.Rating, &f.Comment,
		&f.SceneCode, &f.LessonPlanID, &f.StageCode, &f.AIResponsePreview,
		&f.TraceID, &f.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFeedbackNotFound
		}
		return nil, fmt.Errorf("查询反馈失败: %w", err)
	}
	return f, nil
}

// ==================== 删除 ====================

// DeleteAssistantFeedback 硬删除反馈(不记审计日志,老师撤回反馈是常见操作)
//
// 调用方:service 层确认是本人操作后调用
func DeleteAssistantFeedback(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM assistant_feedback WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除反馈失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrFeedbackNotFound
	}
	return nil
}

// ==================== 单个助手统计 ====================

// GetAssistantFeedbackStats 获取某助手的反馈统计
//
// 用途:admin 后台展示助手的 👍/👎 比例,辅助 P2 数据飞轮决策
//
// SQL 优化:
//   - 利用 idx_feedback_assistant 索引(assistant_id, rating, created_at DESC)
//   - 一次查询用 CASE WHEN 聚合,避免两次 SELECT
func GetAssistantFeedbackStats(ctx context.Context, assistantID string) (*models.FeedbackStatsResponse, error) {
	stats := &models.FeedbackStatsResponse{
		AssistantID: assistantID,
	}

	query := `
		SELECT
			COUNT(*) FILTER (WHERE rating = 'up')   AS up_count,
			COUNT(*) FILTER (WHERE rating = 'down') AS down_count
		FROM assistant_feedback
		WHERE assistant_id = $1
	`
	if err := database.DB.QueryRow(ctx, query, assistantID).Scan(
		&stats.UpCount, &stats.DownCount,
	); err != nil {
		return nil, fmt.Errorf("查询助手反馈统计失败: %w", err)
	}

	stats.TotalCount = stats.UpCount + stats.DownCount
	if stats.TotalCount > 0 {
		stats.UpRatio = float64(stats.UpCount) / float64(stats.TotalCount)
	}
	return stats, nil
}

// ==================== 列表查询(admin 后台用) ====================

// ListAssistantFeedback 分页查询反馈
//
// 权限由 handler/service 层保证(一般只有 admin 能调用)
// 本方法内无权限校验,专注数据查询
//
// 索引命中策略:
//   - 按 assistant_id 过滤  → 命中 idx_feedback_assistant
//   - 按 user_id 过滤       → 命中 idx_feedback_user
//   - 按 scene_code 过滤    → 命中 idx_feedback_scene
//   - 其它场景            → 走 idx_feedback_created(时间倒序兜底)
func ListAssistantFeedback(ctx context.Context, params *models.ListFeedbackParams) (*models.FeedbackListResponse, error) {
	// 兜底分页参数
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	offset := (params.Page - 1) * params.PageSize

	// 动态构建 WHERE 子句
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if params.AssistantID != "" {
		where += fmt.Sprintf(" AND f.assistant_id = $%d", idx)
		args = append(args, params.AssistantID)
		idx++
	}
	if params.UserID != "" {
		where += fmt.Sprintf(" AND f.user_id = $%d", idx)
		args = append(args, params.UserID)
		idx++
	}
	if params.Rating != "" {
		where += fmt.Sprintf(" AND f.rating = $%d", idx)
		args = append(args, params.Rating)
		idx++
	}
	if params.SceneCode != "" {
		where += fmt.Sprintf(" AND f.scene_code = $%d", idx)
		args = append(args, params.SceneCode)
		idx++
	}
	if params.StartDate != "" {
		where += fmt.Sprintf(" AND f.created_at >= $%d::date", idx)
		args = append(args, params.StartDate)
		idx++
	}
	if params.EndDate != "" {
		where += fmt.Sprintf(" AND f.created_at < ($%d::date + INTERVAL '1 day')", idx)
		args = append(args, params.EndDate)
		idx++
	}

	// 查总数
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM assistant_feedback f
		%s`, where)

	var total int
	if err := database.DB.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("统计反馈数量失败: %w", err)
	}

	// 查数据(带分页+LEFT JOIN 补展示字段)
	dataArgs := append(args, params.PageSize, offset)
	dataSQL := fmt.Sprintf(`
		SELECT f.id, f.assistant_id,
		       COALESCE(a.name, '(已删除助手)') AS assistant_name,
		       f.user_id,
		       COALESCE(u.display_name, '(已删除用户)') AS user_name,
		       f.rating, f.comment,
		       f.scene_code, f.lesson_plan_id, f.stage_code, f.ai_response_preview,
		       f.created_at
		FROM assistant_feedback f
		LEFT JOIN ai_assistants a ON a.id = f.assistant_id
		LEFT JOIN users u         ON u.id = f.user_id
		%s
		ORDER BY f.created_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := database.DB.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("查询反馈列表失败: %w", err)
	}
	defer rows.Close()

	var list []*models.AssistantFeedbackListItem
	for rows.Next() {
		item := &models.AssistantFeedbackListItem{}
		if err := rows.Scan(
			&item.ID, &item.AssistantID, &item.AssistantName,
			&item.UserID, &item.UserName,
			&item.Rating, &item.Comment,
			&item.SceneCode, &item.LessonPlanID, &item.StageCode, &item.AIResponsePreview,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描反馈行失败: %w", err)
		}
		list = append(list, item)
	}

	if list == nil {
		list = []*models.AssistantFeedbackListItem{}
	}
	return &models.FeedbackListResponse{
		Feedbacks: list,
		Total:     total,
	}, nil
}
