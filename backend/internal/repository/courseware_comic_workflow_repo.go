package repository

// courseware_comic_workflow_repo.go — 知识点漫画教师工作流仓储
//
// 本文件只读取和迁移教师视角工作流字段。
// 原有项目status继续由规划、生图和插页仓储维护。
//
// 权限边界：
//   - 所有查询同时限定courseware_id、project_id和created_by；
//   - 浏览器不能提交created_by或courseware_id改变资源归属；
//   - 首格样张ID采用软关联，后续确认入口必须校验属于同一项目。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

const coursewareComicWorkflowSelectColumns = `
id,
workflow_stage,
storyboard_confirmed_at,
style_confirmed_at,
style_preview_panel_id,
aspect_ratio,
image_quality,
insertion_mode,
style_instruction,
visual_style_source`

func scanCoursewareComicWorkflowState(
	scanner interface {
		Scan(dest ...interface{}) error
	},
) (*models.CoursewareComicWorkflowState, error) {
	state :=
		&models.CoursewareComicWorkflowState{}

	err :=
		scanner.Scan(
			&state.ProjectID,
			&state.Stage,
			&state.StoryboardConfirmedAt,
			&state.StyleConfirmedAt,
			&state.StylePreviewPanelID,
			&state.AspectRatio,
			&state.ImageQuality,
			&state.InsertionMode,
			&state.StyleInstruction,
			&state.VisualStyleSource,
		)
	if err != nil {
		return nil, err
	}

	normalized, valid :=
		models.NormalizeCoursewareComicWorkflowState(
			state,
		)
	if !valid {
		return nil,
			fmt.Errorf(
				"知识点漫画工作流数据无效",
			)
	}

	return normalized, nil
}

// GetCoursewareComicWorkflowState 返回作者自己的单个项目工作流。
func GetCoursewareComicWorkflowState(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
) (*models.CoursewareComicWorkflowState, error) {
	state, err :=
		scanCoursewareComicWorkflowState(
			database.DB.QueryRow(
				ctx,
				`SELECT `+
					coursewareComicWorkflowSelectColumns+
					` FROM courseware_comic_projects
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3`,
				strings.TrimSpace(
					projectID,
				),
				strings.TrimSpace(
					coursewareID,
				),
				strings.TrimSpace(
					userID,
				),
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return nil,
			ErrCoursewareComicProjectNotFound
	}
	if err != nil {
		return nil,
			fmt.Errorf(
				"读取知识点漫画工作流失败: %w",
				err,
			)
	}

	return state, nil
}

// ListCoursewareComicWorkflowStates 返回课件内作者全部项目的工作流。
func ListCoursewareComicWorkflowStates(
	ctx context.Context,
	coursewareID string,
	userID string,
) (map[string]*models.CoursewareComicWorkflowState, error) {
	rows, err :=
		database.DB.Query(
			ctx,
			`SELECT `+
				coursewareComicWorkflowSelectColumns+
				` FROM courseware_comic_projects
WHERE courseware_id = $1
  AND created_by = $2
ORDER BY updated_at DESC`,
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				userID,
			),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"查询知识点漫画工作流列表失败: %w",
				err,
			)
	}
	defer rows.Close()

	result :=
		make(
			map[string]*models.CoursewareComicWorkflowState,
		)

	for rows.Next() {
		state, scanErr :=
			scanCoursewareComicWorkflowState(
				rows,
			)
		if scanErr != nil {
			return nil,
				fmt.Errorf(
					"扫描知识点漫画工作流失败: %w",
					scanErr,
				)
		}

		result[state.ProjectID] =
			state
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"遍历知识点漫画工作流失败: %w",
				err,
			)
	}

	return result, nil
}

// AdvanceCoursewareComicWorkflowAfterPlanning 在AI分镜规划成功后进入第二步。
//
// 每次重新规划都会撤销旧的分镜确认、风格确认和样张定位。
// 已保存的视觉偏好和style_instruction继续保留，
// 老师可在第三步复用或重新修改。
//
// status必须已经由分格仓储收敛为planned，避免提前展示未落库分镜。
func AdvanceCoursewareComicWorkflowAfterPlanning(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
) (*models.CoursewareComicWorkflowState, error) {
	state, err :=
		scanCoursewareComicWorkflowState(
			database.DB.QueryRow(
				ctx,
				`UPDATE courseware_comic_projects
SET workflow_stage = $1,
    storyboard_confirmed_at = NULL,
    style_confirmed_at = NULL,
    style_preview_panel_id = NULL,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND created_by = $4
  AND status = $5
RETURNING `+
					coursewareComicWorkflowSelectColumns,
				models.CWComicWorkflowStoryboard,
				strings.TrimSpace(
					projectID,
				),
				strings.TrimSpace(
					coursewareID,
				),
				strings.TrimSpace(
					userID,
				),
				models.CWComicProjectStatusPlanned,
			),
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
				"推进知识点漫画分镜确认步骤失败: %w",
				err,
			)
	}

	return state, nil
}
