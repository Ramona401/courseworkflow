package repository

// courseware_comic_style_preview_claim_repo.go
//
// 本文件负责第三步首格样张的任务领取和失败收敛。
//
// 与整批生图不同：
//   - 项目status始终保持planned；
//   - 工作流始终保持style_preview；
//   - 只允许领取panel_no=1；
//   - 只把第1格临时改为generating；
//   - 失败后只把第1格改为failed；
//   - 不聚合整个项目为ready或failed。
//
// 项目version仍会递增，防止旧标签页重复启动样张任务。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ClaimCoursewareComicStylePreview 领取第1格样张生成任务。
func ClaimCoursewareComicStylePreview(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	expectedProjectVersion int,
) (*models.CoursewareComicPanel, error) {
	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

	userID =
		strings.TrimSpace(
			userID,
		)

	if coursewareID == "" ||
		projectID == "" ||
		userID == "" ||
		expectedProjectVersion < 1 {
		return nil,
			fmt.Errorf(
				"首格样张领取参数无效",
			)
	}

	tx, err :=
		database.DB.Begin(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启首格样张领取事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	var (
		projectStatus       string
		workflowStage       string
		storyboardConfirmed bool
	)

	err =
		tx.QueryRow(
			ctx,
			`SELECT
    status,
    workflow_stage,
    storyboard_confirmed_at IS NOT NULL
FROM courseware_comic_projects
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3
  AND version = $4
FOR UPDATE`,
			projectID,
			coursewareID,
			userID,
			expectedProjectVersion,
		).Scan(
			&projectStatus,
			&workflowStage,
			&storyboardConfirmed,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return nil,
			ErrCoursewareComicProjectConflict
	}
	if err != nil {
		return nil,
			fmt.Errorf(
				"锁定首格样张项目失败: %w",
				err,
			)
	}

	if projectStatus !=
		models.CWComicProjectStatusPlanned ||
		workflowStage !=
			models.CWComicWorkflowStylePreview ||
		!storyboardConfirmed {
		return nil,
			ErrCoursewareComicProjectNotEditable
	}

	panel, err :=
		scanCoursewareComicPanel(
			tx.QueryRow(
				ctx,
				`SELECT `+
					coursewareComicPanelSelectColumns+
					` FROM courseware_comic_panels
WHERE project_id = $1
  AND panel_no = 1
FOR UPDATE`,
				projectID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return nil,
			ErrCoursewareComicPanelNotFound
	}
	if err != nil {
		return nil,
			fmt.Errorf(
				"锁定首格样张分格失败: %w",
				err,
			)
	}

	switch panel.Status {
	case models.CWComicPanelStatusPlanned,
		models.CWComicPanelStatusGenerated,
		models.CWComicPanelStatusFailed,
		models.CWComicPanelStatusStale:
	default:
		return nil,
			ErrCoursewareComicPanelNotGeneratable
	}

	claimedPanel, err :=
		scanCoursewareComicPanel(
			tx.QueryRow(
				ctx,
				`UPDATE courseware_comic_panels
SET status = $1,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $2
  AND project_id = $3
  AND panel_no = 1
  AND version = $4
  AND status IN ($5, $6, $7, $8)
RETURNING `+
					coursewareComicPanelSelectColumns,
				models.CWComicPanelStatusGenerating,
				panel.ID,
				projectID,
				panel.Version,
				models.CWComicPanelStatusPlanned,
				models.CWComicPanelStatusGenerated,
				models.CWComicPanelStatusFailed,
				models.CWComicPanelStatusStale,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return nil,
			ErrCoursewareComicPanelConflict
	}
	if err != nil {
		return nil,
			fmt.Errorf(
				"领取首格样张分格失败: %w",
				err,
			)
	}

	tag, err :=
		tx.Exec(
			ctx,
			`UPDATE courseware_comic_projects
SET style_preview_panel_id = NULL,
    style_confirmed_at = NULL,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3
  AND version = $4
  AND status = $5
  AND workflow_stage = $6
  AND storyboard_confirmed_at IS NOT NULL`,
			projectID,
			coursewareID,
			userID,
			expectedProjectVersion,
			models.CWComicProjectStatusPlanned,
			models.CWComicWorkflowStylePreview,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"更新首格样张项目版本失败: %w",
				err,
			)
	}

	if tag.RowsAffected() != 1 {
		return nil,
			ErrCoursewareComicProjectConflict
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return nil,
			fmt.Errorf(
				"提交首格样张领取事务失败: %w",
				err,
			)
	}

	return claimedPanel, nil
}

// FailCoursewareComicStylePreview 收敛首格样张生成失败。
//
// 已存在的旧图片资产和历史版本不会被删除。
// style_preview_panel_id保持为空，老师可以修改设置后再次生成。
func FailCoursewareComicStylePreview(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	failure error,
) error {
	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	projectID =
		strings.TrimSpace(
			projectID,
		)

	panelID =
		strings.TrimSpace(
			panelID,
		)

	userID =
		strings.TrimSpace(
			userID,
		)

	if coursewareID == "" ||
		projectID == "" ||
		panelID == "" ||
		userID == "" {
		return fmt.Errorf(
			"首格样张失败收敛参数无效",
		)
	}

	message :=
		"首格样张生成失败"

	if failure != nil &&
		strings.TrimSpace(
			failure.Error(),
		) != "" {
		message =
			strings.TrimSpace(
				failure.Error(),
			)
	}

	tx, err :=
		database.DB.Begin(
			ctx,
		)
	if err != nil {
		return fmt.Errorf(
			"开启首格样张失败事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(
			ctx,
		)
	}()

	panelTag, err :=
		tx.Exec(
			ctx,
			`UPDATE courseware_comic_panels panel
SET status = $1,
    last_error = $2,
    version = version + 1,
    updated_at = now()
WHERE panel.id = $3
  AND panel.project_id = $4
  AND panel.panel_no = 1
  AND panel.status = $5
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_projects project
      WHERE project.id = panel.project_id
        AND project.courseware_id = $6
        AND project.created_by = $7
        AND project.status = $8
        AND project.workflow_stage = $9
  )`,
			models.CWComicPanelStatusFailed,
			message,
			panelID,
			projectID,
			models.CWComicPanelStatusGenerating,
			coursewareID,
			userID,
			models.CWComicProjectStatusPlanned,
			models.CWComicWorkflowStylePreview,
		)
	if err != nil {
		return fmt.Errorf(
			"标记首格样张失败异常: %w",
			err,
		)
	}

	if panelTag.RowsAffected() != 1 {
		return ErrCoursewareComicPanelConflict
	}

	projectTag, err :=
		tx.Exec(
			ctx,
			`UPDATE courseware_comic_projects
SET style_preview_panel_id = NULL,
    style_confirmed_at = NULL,
    last_error = $1,
    version = version + 1,
    updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND created_by = $4
  AND status = $5
  AND workflow_stage = $6`,
			message,
			projectID,
			coursewareID,
			userID,
			models.CWComicProjectStatusPlanned,
			models.CWComicWorkflowStylePreview,
		)
	if err != nil {
		return fmt.Errorf(
			"记录首格样张项目错误失败: %w",
			err,
		)
	}

	if projectTag.RowsAffected() != 1 {
		return ErrCoursewareComicProjectConflict
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return fmt.Errorf(
			"提交首格样张失败事务失败: %w",
			err,
		)
	}

	return nil
}
