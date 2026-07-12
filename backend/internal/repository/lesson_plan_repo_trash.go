package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"tedna/internal/database"
	"tedna/internal/models"
)

// SoftDeleteLessonPlan 软删除教案（移入回收站）
func SoftDeleteLessonPlan(id uuid.UUID, authorID uuid.UUID) error {
	ctx := context.Background()
	tag, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND author_id = $2 AND deleted_at IS NULL`,
		id, authorID)
	if err != nil {
		return fmt.Errorf("软删除教案失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("教案不存在或已被删除")
	}
	return nil
}

// RestoreLessonPlan 从回收站恢复教案
func RestoreLessonPlan(id uuid.UUID, authorID uuid.UUID) error {
	ctx := context.Background()
	tag, err := database.DB.Exec(ctx,
		`UPDATE lesson_plans SET deleted_at = NULL, updated_at = NOW()
		 WHERE id = $1 AND author_id = $2 AND deleted_at IS NOT NULL`,
		id, authorID)
	if err != nil {
		return fmt.Errorf("恢复教案失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("教案不存在或未在回收站中")
	}
	return nil
}

// PermanentDeleteLessonPlan 永久删除教案（物理删除，仅限回收站中的）
func PermanentDeleteLessonPlan(id uuid.UUID, authorID uuid.UUID) error {
	ctx := context.Background()
	tag, err := database.DB.Exec(ctx,
		`DELETE FROM lesson_plans WHERE id = $1 AND author_id = $2 AND deleted_at IS NOT NULL`,
		id, authorID)
	if err != nil {
		return fmt.Errorf("永久删除教案失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("教案不存在或未在回收站中")
	}
	return nil
}

// ListLessonPlanTrash 获取回收站中的教案列表
func ListLessonPlanTrash(authorID uuid.UUID) ([]models.TrashItem, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, title, subject, grade, deleted_at, created_at
		 FROM lesson_plans
		 WHERE author_id = $1 AND deleted_at IS NOT NULL
		 ORDER BY deleted_at DESC`,
		authorID)
	if err != nil {
		return nil, fmt.Errorf("获取教案回收站列表失败: %w", err)
	}
	defer rows.Close()

	var items []models.TrashItem
	for rows.Next() {
		var id uuid.UUID
		var title, subject, grade string
		var deletedAt, createdAt time.Time
		if err := rows.Scan(&id, &title, &subject, &grade, &deletedAt, &createdAt); err != nil {
			return nil, fmt.Errorf("扫描教案回收站行失败: %w", err)
		}
		daysLeft := 30 - int(time.Since(deletedAt).Hours()/24)
		if daysLeft < 0 {
			daysLeft = 0
		}
		items = append(items, models.TrashItem{
			ID:        id,
			Title:     title,
			Type:      "lesson_plan",
			Subject:   subject,
			Grade:     grade,
			DeletedAt: deletedAt,
			CreatedAt: createdAt,
			DaysLeft:  daysLeft,
		})
	}
	return items, nil
}

// PurgeLessonPlanExpiredTrash 清理过期的回收站教案（超过指定天数）
func PurgeLessonPlanExpiredTrash(days int) (int64, error) {
	ctx := context.Background()
	interval := fmt.Sprintf("%d days", days)
	tag, err := database.DB.Exec(ctx,
		`DELETE FROM lesson_plans WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - $1::interval`,
		interval)
	if err != nil {
		return 0, fmt.Errorf("清理过期教案失败: %w", err)
	}
	return tag.RowsAffected(), nil
}

// SoftDeleteLessonPlanByID 软删除教案（接收string参数，兼容service层调用风格）
func SoftDeleteLessonPlanByID(ctx context.Context, id string, authorID string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("教案ID无效: %w", err)
	}
	aid, err := uuid.Parse(authorID)
	if err != nil {
		return fmt.Errorf("用户ID无效: %w", err)
	}
	return SoftDeleteLessonPlan(uid, aid)
}
