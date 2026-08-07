package repository

// courseware_comic_generation_failure_repo.go — 并发图片生产失败收敛仓储
//
// 四路并发生成时，成功事务和失败事务都必须遵守同一锁顺序：
// courseware_comic_projects → courseware_comic_panels。
//
// 旧串行流程中的失败函数仍保留给既有单格恢复逻辑使用；
// 整批并发 worker 只调用本文件中的项目优先版本，避免与成功结算事务
// 在 PostgreSQL 中形成反向锁等待。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// FailCoursewareComicPanelGenerationProjectFirst
// 按项目优先锁顺序标记一个整批生成分格失败。
//
// 多个并发 worker 失败时允许项目已经是 failed；每个分格仍必须从
// generating 原子进入 failed，避免重复请求覆盖已经完成或已恢复的结果。
func FailCoursewareComicPanelGenerationProjectFirst(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	failure error,
) error {
	coursewareID = strings.TrimSpace(coursewareID)
	projectID = strings.TrimSpace(projectID)
	panelID = strings.TrimSpace(panelID)
	userID = strings.TrimSpace(userID)

	message := "漫画格图片生成失败"
	if failure != nil && strings.TrimSpace(failure.Error()) != "" {
		message = strings.TrimSpace(failure.Error())
	}

	if coursewareID == "" || projectID == "" || panelID == "" || userID == "" {
		return fmt.Errorf("漫画格失败收敛参数无效")
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启漫画格并发失败事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var projectStatus string
	err = tx.QueryRow(
		ctx,
		`SELECT status
FROM courseware_comic_projects
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3
  AND status IN ($4, $5)
FOR UPDATE`,
		projectID,
		coursewareID,
		userID,
		models.CWComicProjectStatusGenerating,
		models.CWComicProjectStatusFailed,
	).Scan(&projectStatus)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCoursewareComicProjectConflict
	}
	if err != nil {
		return fmt.Errorf("锁定漫画项目失败: %w", err)
	}

	tag, err := tx.Exec(
		ctx,
		`UPDATE courseware_comic_panels
SET status = $1,
    last_error = $2,
    version = version + 1,
    updated_at = now()
WHERE id = $3
  AND project_id = $4
  AND status = $5`,
		models.CWComicPanelStatusFailed,
		message,
		panelID,
		projectID,
		models.CWComicPanelStatusGenerating,
	)
	if err != nil {
		return fmt.Errorf("标记并发漫画格生成失败异常: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareComicPanelConflict
	}

	tag, err = tx.Exec(
		ctx,
		`UPDATE courseware_comic_projects
SET status = $1,
    last_error = $2,
    version = version + 1,
    updated_at = now()
WHERE id = $3
  AND courseware_id = $4
  AND created_by = $5
  AND status IN ($6, $7)`,
		models.CWComicProjectStatusFailed,
		message,
		projectID,
		coursewareID,
		userID,
		models.CWComicProjectStatusGenerating,
		models.CWComicProjectStatusFailed,
	)
	if err != nil {
		return fmt.Errorf("标记并发漫画项目生成失败异常: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareComicProjectConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交漫画格并发失败事务失败: %w", err)
	}

	return nil
}
