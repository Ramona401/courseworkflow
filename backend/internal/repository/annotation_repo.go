package repository

// annotation_repo.go — 教案段落批注数据访问层
// 提供批注的增删改查和状态更新操作
//
// v104改动：
//   - CreateAnnotation / ListAnnotationsByPlanID / GetAnnotationByID 支持 review_round 字段
//   - 新增 ArchiveAnnotationsByPlanID — 提交新一轮评审时将所有 pending 批注归档
//   - 新增 GetCurrentAnnotationRound — 获取教案当前最大评审轮次
// v121改动（AI辅助修改 Bug 修复 · 方案C）：
//   - 新增 RestoreArchivedAnnotationsForLatestRound — 教研员退回教案时,把
//     最新一轮已归档(archived)批注恢复为 pending,让作者可以继续用 AI 辅助修改
//     处理上一轮未彻底解决的批注(符合"退回即继续工作"的业务直觉)

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrAnnotationNotFound 批注不存在错误
var ErrAnnotationNotFound = errors.New("批注不存在")

// ==================== 创建批注 ====================

// CreateAnnotation 创建段落批注
// review_round 由调用方传入（通常从教案的评审历史轮次推断）
func CreateAnnotation(ctx context.Context, a *models.LessonPlanAnnotation) error {
	// review_round 默认至少为1
	round := a.ReviewRound
	if round <= 0 {
		round = 1
	}
	query := `
		INSERT INTO lesson_plan_annotations
			(lesson_plan_id, reviewer_id, reviewer_name, paragraph_index, paragraph_preview, content, status, review_round)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7)
		RETURNING id, created_at, updated_at`
	return database.DB.QueryRow(ctx, query,
		a.LessonPlanID,
		a.ReviewerID,
		a.ReviewerName,
		a.ParagraphIndex,
		a.ParagraphPreview,
		a.Content,
		round,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// ==================== 查询批注 ====================

// ListAnnotationsByPlanID 查询教案的全部批注，按轮次→段落序号→时间排序
// 前端收到后可按 review_round 分组展示，区分历史轮次和当前轮次
func ListAnnotationsByPlanID(ctx context.Context, planID string) ([]*models.LessonPlanAnnotation, error) {
	query := `
		SELECT id, lesson_plan_id, reviewer_id, reviewer_name,
		       paragraph_index, paragraph_preview, content, status,
		       COALESCE(review_round, 1) AS review_round,
		       created_at, updated_at
		FROM lesson_plan_annotations
		WHERE lesson_plan_id = $1
		ORDER BY review_round ASC, paragraph_index ASC, created_at ASC`
	rows, err := database.DB.Query(ctx, query, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.LessonPlanAnnotation
	for rows.Next() {
		a := &models.LessonPlanAnnotation{}
		if err := rows.Scan(
			&a.ID, &a.LessonPlanID, &a.ReviewerID, &a.ReviewerName,
			&a.ParagraphIndex, &a.ParagraphPreview, &a.Content, &a.Status,
			&a.ReviewRound,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// GetAnnotationByID 按ID查询单条批注
func GetAnnotationByID(ctx context.Context, id string) (*models.LessonPlanAnnotation, error) {
	query := `
		SELECT id, lesson_plan_id, reviewer_id, reviewer_name,
		       paragraph_index, paragraph_preview, content, status,
		       COALESCE(review_round, 1) AS review_round,
		       created_at, updated_at
		FROM lesson_plan_annotations
		WHERE id = $1`
	a := &models.LessonPlanAnnotation{}
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.LessonPlanID, &a.ReviewerID, &a.ReviewerName,
		&a.ParagraphIndex, &a.ParagraphPreview, &a.Content, &a.Status,
		&a.ReviewRound,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAnnotationNotFound
		}
		return nil, err
	}
	return a, nil
}

// GetCurrentAnnotationRound 获取教案当前最大评审轮次（用于新建批注时填入轮次号）
// 如果没有任何批注，返回1
func GetCurrentAnnotationRound(ctx context.Context, planID string) (int, error) {
	var maxRound int
	err := database.DB.QueryRow(ctx,
		`SELECT COALESCE(MAX(review_round), 0) FROM lesson_plan_annotations WHERE lesson_plan_id = $1`,
		planID,
	).Scan(&maxRound)
	if err != nil {
		return 1, err
	}
	if maxRound <= 0 {
		return 1, nil
	}
	return maxRound, nil
}

// ==================== 更新批注 ====================

// UpdateAnnotationContent 更新批注文字内容（仅评审员本人可操作）
func UpdateAnnotationContent(ctx context.Context, id string, content string) error {
	query := `
		UPDATE lesson_plan_annotations
		SET content = $1, updated_at = $2
		WHERE id = $3`
	tag, err := database.DB.Exec(ctx, query, content, time.Now(), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAnnotationNotFound
	}
	return nil
}

// UpdateAnnotationStatus 更新批注处理状态（pending/resolved）
func UpdateAnnotationStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE lesson_plan_annotations
		SET status = $1, updated_at = $2
		WHERE id = $3`
	tag, err := database.DB.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAnnotationNotFound
	}
	return nil
}

// ArchiveAnnotationsByPlanID 将教案所有 pending 批注归档
// 在老师重新提交评审时调用，防止新旧轮次批注混显
// 只归档 pending 状态，已 resolved 的保持不变
func ArchiveAnnotationsByPlanID(ctx context.Context, planID string) error {
	_, err := database.DB.Exec(ctx,
		`UPDATE lesson_plan_annotations
		 SET status = $1, updated_at = $2
		 WHERE lesson_plan_id = $3 AND status = $4`,
		models.AnnotationStatusArchived, time.Now(), planID, models.AnnotationStatusPending,
	)
	return err
}

// RestoreArchivedAnnotationsForLatestRound 把教案最新一轮的 archived 批注恢复为 pending
// v121新增：教研员退回教案时调用，配合"退回=继续工作"的业务场景
// 业务含义：
//   - 教研员退回意味着作者需要继续修改
//   - 上一轮提交时被归档的批注(但作者实际还没处理)应该重新激活
//   - 这样作者能继续用"AI辅助修改"按钮处理这些批注
// 实现要点：
//   - 只影响"最新一轮"的 archived 批注(保留更早轮次的归档历史)
//   - 已 resolved 的批注不动(它们是真的处理完了的历史记录)
func RestoreArchivedAnnotationsForLatestRound(ctx context.Context, planID string) error {
	_, err := database.DB.Exec(ctx, `
		UPDATE lesson_plan_annotations
		SET status = $1, updated_at = $2
		WHERE lesson_plan_id = $3
		  AND status = $4
		  AND review_round = (
		    SELECT COALESCE(MAX(review_round), 1)
		    FROM lesson_plan_annotations
		    WHERE lesson_plan_id = $3
		  )
	`, models.AnnotationStatusPending, time.Now(), planID, models.AnnotationStatusArchived)
	return err
}

// ==================== 删除批注 ====================

// DeleteAnnotation 删除批注（仅评审员本人或管理员可操作）
func DeleteAnnotation(ctx context.Context, id string) error {
	query := `DELETE FROM lesson_plan_annotations WHERE id = $1`
	tag, err := database.DB.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAnnotationNotFound
	}
	return nil
}

// DeleteAnnotationsByPlanID 删除教案的全部批注（教案删除时级联调用）
func DeleteAnnotationsByPlanID(ctx context.Context, planID string) error {
	_, err := database.DB.Exec(ctx,
		`DELETE FROM lesson_plan_annotations WHERE lesson_plan_id = $1`, planID)
	return err
}
