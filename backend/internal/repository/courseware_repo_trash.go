package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"tedna/internal/database"
	"tedna/internal/models"
)

// SoftDeleteCourseware 软删除课件（移入回收站）
func SoftDeleteCourseware(id uuid.UUID, userID uuid.UUID) error {
	ctx := context.Background()
	tag, err := database.DB.Exec(ctx,
		`UPDATE coursewares SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		id, userID)
	if err != nil {
		return fmt.Errorf("软删除课件失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在或已被删除")
	}
	return nil
}

// RestoreCourseware 从回收站恢复课件
func RestoreCourseware(id uuid.UUID, userID uuid.UUID) error {
	ctx := context.Background()
	tag, err := database.DB.Exec(ctx,
		`UPDATE coursewares SET deleted_at = NULL, updated_at = NOW()
		 WHERE id = $1 AND user_id = $2 AND deleted_at IS NOT NULL`,
		id, userID)
	if err != nil {
		return fmt.Errorf("恢复课件失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在或未在回收站中")
	}
	return nil
}

// PermanentDeleteCourseware 永久删除课件（物理删除，仅限回收站中的）
func PermanentDeleteCourseware(id uuid.UUID, userID uuid.UUID) error {
	ctx := context.Background()
	tag, err := database.DB.Exec(ctx,
		`DELETE FROM coursewares WHERE id = $1 AND user_id = $2 AND deleted_at IS NOT NULL`,
		id, userID)
	if err != nil {
		return fmt.Errorf("永久删除课件失败: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("课件不存在或未在回收站中")
	}
	return nil
}

// ListCoursewareTrash 获取回收站中的课件列表
func ListCoursewareTrash(userID uuid.UUID) ([]models.TrashItem, error) {
	ctx := context.Background()
	rows, err := database.DB.Query(ctx,
		`SELECT id, title, COALESCE(subject, ''), COALESCE(grade, ''), deleted_at, created_at
		 FROM coursewares
		 WHERE user_id = $1 AND deleted_at IS NOT NULL
		 ORDER BY deleted_at DESC`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("获取课件回收站列表失败: %w", err)
	}
	defer rows.Close()

	var items []models.TrashItem
	for rows.Next() {
		var id uuid.UUID
		var title, subject, grade string
		var deletedAt, createdAt time.Time
		if err := rows.Scan(&id, &title, &subject, &grade, &deletedAt, &createdAt); err != nil {
			return nil, fmt.Errorf("扫描课件回收站行失败: %w", err)
		}
		daysLeft := 30 - int(time.Since(deletedAt).Hours()/24)
		if daysLeft < 0 {
			daysLeft = 0
		}
		items = append(items, models.TrashItem{
			ID:        id,
			Title:     title,
			Type:      "courseware",
			Subject:   subject,
			Grade:     grade,
			DeletedAt: deletedAt,
			CreatedAt: createdAt,
			DaysLeft:  daysLeft,
		})
	}
	return items, nil
}

// PurgeCoursewareExpiredTrash 清理过期的回收站课件（超过指定天数）
func PurgeCoursewareExpiredTrash(days int) (int64, error) {
	ctx := context.Background()
	interval := fmt.Sprintf("%d days", days)
	tag, err := database.DB.Exec(ctx,
		`DELETE FROM coursewares WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - $1::interval`,
		interval)
	if err != nil {
		return 0, fmt.Errorf("清理过期课件失败: %w", err)
	}
	return tag.RowsAffected(), nil
}

// SoftDeleteCoursewareByID 软删除课件（接收string参数，兼容service层调用风格）
func SoftDeleteCoursewareByID(ctx context.Context, id string, userID string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("课件ID无效: %w", err)
	}
	oid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("用户ID无效: %w", err)
	}
	return SoftDeleteCourseware(uid, oid)
}
