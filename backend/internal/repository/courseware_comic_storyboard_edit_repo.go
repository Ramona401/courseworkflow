package repository

// courseware_comic_storyboard_edit_repo.go — 教师端单格分镜安全编辑仓储
//
// 本文件只允许在第二步、分镜尚未确认且尚未开始任何图片生产时，
// 使用 panel.version CAS 修改教师可见的教学分镜字段。
//
// 锁顺序固定为：courseware_comic_projects → courseware_comic_panels。
// 该顺序与图片生成完成事务保持一致，避免后续并发生产引入反向锁死锁。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// UpdateCoursewareComicStoryboardPanelIfUnchanged 保存第二步的单格教学分镜。
//
// visualPrompt 由服务端根据安全业务字段确定性重建，浏览器不能直接提交。
// 已有图片资产、正在生成或已经生成的项目不能通过本入口修改分镜。
func UpdateCoursewareComicStoryboardPanelIfUnchanged(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	expectedVersion int,
	storyPurpose string,
	knowledgeClaim string,
	sceneText string,
	actionText string,
	cameraText string,
	knowledgePresentation string,
	visualPrompt string,
) (*models.CoursewareComicPanel, error) {
	coursewareID = strings.TrimSpace(coursewareID)
	projectID = strings.TrimSpace(projectID)
	panelID = strings.TrimSpace(panelID)
	userID = strings.TrimSpace(userID)

	if coursewareID == "" || projectID == "" || panelID == "" ||
		userID == "" || expectedVersion < 1 {
		return nil, fmt.Errorf("知识点漫画分镜编辑参数无效")
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启漫画分镜编辑事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var projectStatus string
	var workflowStage string
	var storyboardConfirmed bool

	err = tx.QueryRow(
		ctx,
		`SELECT status,
       workflow_stage,
       storyboard_confirmed_at IS NOT NULL
FROM courseware_comic_projects
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3
FOR UPDATE`,
		projectID,
		coursewareID,
		userID,
	).Scan(
		&projectStatus,
		&workflowStage,
		&storyboardConfirmed,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoursewareComicProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("锁定漫画项目失败: %w", err)
	}

	if projectStatus != models.CWComicProjectStatusPlanned ||
		workflowStage != models.CWComicWorkflowStoryboard ||
		storyboardConfirmed {
		return nil, ErrCoursewareComicProjectNotEditable
	}

	updated, err := scanCoursewareComicPanel(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_comic_panels panel
SET story_purpose = $1,
    knowledge_claim = $2,
    scene_text = $3,
    action_text = $4,
    camera_text = $5,
    knowledge_presentation = $6,
    visual_prompt = $7,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE panel.id = $8
  AND panel.project_id = $9
  AND panel.version = $10
  AND panel.status IN ($11, $12, $13)
  AND panel.current_asset_id IS NULL
  AND NOT EXISTS (
      SELECT 1
      FROM courseware_comic_panels started
      WHERE started.project_id = panel.project_id
        AND (
            started.current_asset_id IS NOT NULL
            OR started.status IN ($14, $15)
        )
  )
RETURNING `+coursewareComicPanelSelectColumns,
			strings.TrimSpace(storyPurpose),
			strings.TrimSpace(knowledgeClaim),
			strings.TrimSpace(sceneText),
			strings.TrimSpace(actionText),
			strings.TrimSpace(cameraText),
			strings.TrimSpace(knowledgePresentation),
			strings.TrimSpace(visualPrompt),
			panelID,
			projectID,
			expectedVersion,
			models.CWComicPanelStatusPlanned,
			models.CWComicPanelStatusFailed,
			models.CWComicPanelStatusStale,
			models.CWComicPanelStatusGenerating,
			models.CWComicPanelStatusGenerated,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoursewareComicPanelConflict
	}
	if err != nil {
		return nil, fmt.Errorf("保存漫画单格分镜失败: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("提交漫画分镜编辑事务失败: %w", err)
	}

	return updated, nil
}
