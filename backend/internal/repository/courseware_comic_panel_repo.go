package repository

// courseware_comic_panel_repo.go — 知识点漫画分格仓储
//
// 本文件负责：
//   - 原子替换AI规划的4至8格漫画方案；
//   - 查询单格和项目全部漫画格；
//   - 保存教师修改后的气泡文字覆盖文档；
//   - 保存教师调整后的单格提示词和IAOCI；
//   - 领取、失败和重试单格图片生成状态；
//   - 使用version字段避免多个标签页互相覆盖。
//
// 本文件不负责图片模型调用和资产下载。
// 图片生成成功与历史版本落库由courseware_comic_version_repo.go完成。

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrCoursewareComicPanelNotFound = errors.New(
		"知识点漫画分格不存在",
	)

	ErrCoursewareComicPanelConflict = errors.New(
		"知识点漫画分格已发生变化，请刷新后重试",
	)

	ErrCoursewareComicPanelNotGeneratable = errors.New(
		"知识点漫画分格当前不能生成图片",
	)
)

const coursewareComicPanelSelectColumns = `
id,
project_id,
panel_no,
image_key,
story_purpose,
knowledge_claim,
scene_text,
character_ids_json::text,
action_text,
camera_text,
narration_text,
dialogues_json::text,
knowledge_presentation,
visual_prompt,
negative_prompt,
aoci_text,
relations_json::text,
overlay_document_json::text,
overlay_version,
status,
current_asset_id,
version,
last_error,
created_at,
updated_at`

func scanCoursewareComicPanel(
	scanner interface {
		Scan(dest ...interface{}) error
	},
) (*models.CoursewareComicPanel, error) {
	item := &models.CoursewareComicPanel{}

	err := scanner.Scan(
		&item.ID,
		&item.ProjectID,
		&item.PanelNo,
		&item.ImageKey,
		&item.StoryPurpose,
		&item.KnowledgeClaim,
		&item.SceneText,
		&item.CharacterIDsJSON,
		&item.ActionText,
		&item.CameraText,
		&item.NarrationText,
		&item.DialoguesJSON,
		&item.KnowledgePresentation,
		&item.VisualPrompt,
		&item.NegativePrompt,
		&item.AOCIText,
		&item.RelationsJSON,
		&item.OverlayDocumentJSON,
		&item.OverlayVersion,
		&item.Status,
		&item.CurrentAssetID,
		&item.Version,
		&item.LastError,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}

// ReplaceCoursewareComicPanels 原子替换项目分镜。
//
// 只有尚未开始正式生图的项目可以重新规划。
// 已有generating或generated分格时拒绝删除，防止产生不可追溯资产。
func ReplaceCoursewareComicPanels(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	expectedProjectVersion int,
	panels []*models.CoursewareComicPanel,
) ([]*models.CoursewareComicPanel, error) {
	if len(panels) < 4 || len(panels) > 8 {
		return nil, fmt.Errorf(
			"漫画规划必须包含4至8格",
		)
	}

	sortedPanels := append(
		[]*models.CoursewareComicPanel{},
		panels...,
	)

	sort.Slice(
		sortedPanels,
		func(left int, right int) bool {
			return sortedPanels[left].PanelNo <
				sortedPanels[right].PanelNo
		},
	)

	seenImageKeys := make(map[string]bool)

	for index, panel := range sortedPanels {
		if panel == nil {
			return nil, fmt.Errorf(
				"漫画第%d格对象为空",
				index+1,
			)
		}

		panel.ProjectID =
			strings.TrimSpace(projectID)

		if panel.PanelNo != index+1 {
			return nil, fmt.Errorf(
				"漫画格必须从1开始连续编号",
			)
		}

		if err := normalizeCoursewareComicPanel(
			panel,
		); err != nil {
			return nil, fmt.Errorf(
				"漫画第%d格无效: %w",
				panel.PanelNo,
				err,
			)
		}

		if seenImageKeys[panel.ImageKey] {
			return nil, fmt.Errorf(
				"漫画图片键重复: %s",
				panel.ImageKey,
			)
		}

		seenImageKeys[panel.ImageKey] = true
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开启漫画分镜事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var projectStatus string
	var projectVersion int

	err = tx.QueryRow(
		ctx,
		`SELECT status, version
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
		&projectVersion,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf(
			"锁定漫画项目失败: %w",
			err,
		)
	}

	if projectVersion != expectedProjectVersion {
		return nil,
			ErrCoursewareComicProjectConflict
	}

	switch projectStatus {
	case models.CWComicProjectStatusDraft,
		models.CWComicProjectStatusPlanning,
		models.CWComicProjectStatusPlanned,
		models.CWComicProjectStatusFailed:
	default:
		return nil,
			ErrCoursewareComicProjectNotEditable
	}

	var hasStartedPanels bool

	err = tx.QueryRow(
		ctx,
		`SELECT EXISTS (
    SELECT 1
    FROM courseware_comic_panels
    WHERE project_id = $1
      AND (
          current_asset_id IS NOT NULL
          OR status IN ($2, $3)
      )
)`,
		projectID,
		models.CWComicPanelStatusGenerating,
		models.CWComicPanelStatusGenerated,
	).Scan(&hasStartedPanels)
	if err != nil {
		return nil, fmt.Errorf(
			"检查漫画分格生成状态失败: %w",
			err,
		)
	}

	if hasStartedPanels {
		return nil,
			ErrCoursewareComicProjectNotEditable
	}

	if _, err := tx.Exec(
		ctx,
		`DELETE FROM courseware_comic_panels
WHERE project_id = $1`,
		projectID,
	); err != nil {
		return nil, fmt.Errorf(
			"清理旧漫画分镜失败: %w",
			err,
		)
	}

	createdPanels := make(
		[]*models.CoursewareComicPanel,
		0,
		len(sortedPanels),
	)

	for _, panel := range sortedPanels {
		created, insertErr :=
			scanCoursewareComicPanel(
				tx.QueryRow(
					ctx,
					`INSERT INTO courseware_comic_panels (
project_id,
panel_no,
image_key,
story_purpose,
knowledge_claim,
scene_text,
character_ids_json,
action_text,
camera_text,
narration_text,
dialogues_json,
knowledge_presentation,
visual_prompt,
negative_prompt,
aoci_text,
relations_json,
overlay_document_json,
overlay_version,
status,
current_asset_id,
version,
last_error
)
VALUES (
$1, $2, $3, $4, $5,
$6, $7::jsonb, $8, $9, $10,
$11::jsonb, $12, $13, $14, $15,
$16::jsonb, $17::jsonb, $18, $19, $20,
$21, $22
)
RETURNING `+coursewareComicPanelSelectColumns,
					panel.ProjectID,
					panel.PanelNo,
					panel.ImageKey,
					panel.StoryPurpose,
					panel.KnowledgeClaim,
					panel.SceneText,
					panel.CharacterIDsJSON,
					panel.ActionText,
					panel.CameraText,
					panel.NarrationText,
					panel.DialoguesJSON,
					panel.KnowledgePresentation,
					panel.VisualPrompt,
					panel.NegativePrompt,
					panel.AOCIText,
					panel.RelationsJSON,
					panel.OverlayDocumentJSON,
					panel.OverlayVersion,
					panel.Status,
					cwComicNullableString(
						panel.CurrentAssetID,
					),
					panel.Version,
					panel.LastError,
				),
			)

		if insertErr != nil {
			return nil, fmt.Errorf(
				"保存漫画第%d格失败: %w",
				panel.PanelNo,
				insertErr,
			)
		}

		createdPanels = append(
			createdPanels,
			created,
		)
	}

	tag, err := tx.Exec(
		ctx,
		`UPDATE courseware_comic_projects
SET status = $1,
    panel_count = $2,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $3
  AND courseware_id = $4
  AND created_by = $5
  AND version = $6`,
		models.CWComicProjectStatusPlanned,
		len(createdPanels),
		projectID,
		coursewareID,
		userID,
		expectedProjectVersion,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"更新漫画项目规划状态失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return nil,
			ErrCoursewareComicProjectConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交漫画分镜事务失败: %w",
			err,
		)
	}

	return createdPanels, nil
}

// ListCoursewareComicPanels 返回项目全部分格。
func ListCoursewareComicPanels(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
) ([]*models.CoursewareComicPanel, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT `+
			coursewareComicPanelSelectColumns+
			` FROM courseware_comic_panels panel
WHERE panel.project_id = $1
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_projects project
      WHERE project.id = panel.project_id
        AND project.courseware_id = $2
        AND project.created_by = $3
  )
ORDER BY panel.panel_no ASC`,
		projectID,
		coursewareID,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询漫画分格失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CoursewareComicPanel,
		0,
	)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareComicPanel(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描漫画分格失败: %w",
				scanErr,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历漫画分格失败: %w",
			err,
		)
	}

	return items, nil
}

// GetCoursewareComicPanelByIDForProject 按项目和分格ID读取。
func GetCoursewareComicPanelByIDForProject(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
) (*models.CoursewareComicPanel, error) {
	item, err := scanCoursewareComicPanel(
		database.DB.QueryRow(
			ctx,
			`SELECT `+
				coursewareComicPanelSelectColumns+
				` FROM courseware_comic_panels panel
WHERE panel.id = $1
  AND panel.project_id = $2
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_projects project
      WHERE project.id = panel.project_id
        AND project.courseware_id = $3
        AND project.created_by = $4
  )`,
			panelID,
			projectID,
			coursewareID,
			userID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicPanelNotFound
	}
	if err != nil {
		return nil, fmt.Errorf(
			"读取漫画分格失败: %w",
			err,
		)
	}

	return item, nil
}

// UpdateCoursewareComicPanelOverlayIfUnchanged 保存自动排版或教师调整后的覆盖层。
//
// 该操作不重新生成图片，也不会改写visual_prompt或aoci_text。
// 自动重新排版必须保留教师已经修改的文字内容。
func UpdateCoursewareComicPanelOverlayIfUnchanged(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	expectedVersion int,
	narrationText string,
	dialoguesJSON string,
	overlayDocumentJSON string,
) (*models.CoursewareComicPanel, error) {
	var err error

	dialoguesJSON, err =
		cwComicNormalizeJSON(
			dialoguesJSON,
			"[]",
			"array",
		)
	if err != nil {
		return nil, fmt.Errorf(
			"漫画对白无效: %w",
			err,
		)
	}

	overlayDocumentJSON, err =
		cwComicNormalizeJSON(
			overlayDocumentJSON,
			"{}",
			"object",
		)
	if err != nil {
		return nil, fmt.Errorf(
			"漫画覆盖层无效: %w",
			err,
		)
	}

	updated, err := scanCoursewareComicPanel(
		database.DB.QueryRow(
			ctx,
			`UPDATE courseware_comic_panels panel
SET narration_text = $1,
    dialogues_json = $2::jsonb,
    overlay_document_json = $3::jsonb,
    overlay_version = overlay_version + 1,
    version = version + 1,
    updated_at = now()
WHERE panel.id = $4
  AND panel.project_id = $5
  AND panel.version = $6
  AND panel.status IN ($7, $8, $9, $10)
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_projects project
      WHERE project.id = panel.project_id
        AND project.courseware_id = $11
        AND project.created_by = $12
        AND project.status NOT IN ($13, $14)
  )
RETURNING `+coursewareComicPanelSelectColumns,
			strings.TrimSpace(narrationText),
			dialoguesJSON,
			overlayDocumentJSON,
			panelID,
			projectID,
			expectedVersion,
			models.CWComicPanelStatusPlanned,
			models.CWComicPanelStatusGenerated,
			models.CWComicPanelStatusFailed,
			models.CWComicPanelStatusStale,
			coursewareID,
			userID,
			models.CWComicProjectStatusInserted,
			models.CWComicProjectStatusArchived,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicPanelConflict
	}
	if err != nil {
		return nil, fmt.Errorf(
			"保存漫画文字和气泡排版失败: %w",
			err,
		)
	}

	return updated, nil
}

// UpdateCoursewareComicPanelPromptIfUnchanged 保存单格提示词和IAOCI。
//
// 已生成图片的分格修改提示词后转为stale，明确提示需要重新生成；
// 仅修改文字覆盖层不会触发stale。
func UpdateCoursewareComicPanelPromptIfUnchanged(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	expectedVersion int,
	visualPrompt string,
	negativePrompt string,
	aociText string,
	relationsJSON string,
) (*models.CoursewareComicPanel, error) {
	relationsJSON, err :=
		cwComicNormalizeJSON(
			relationsJSON,
			"[]",
			"array",
		)
	if err != nil {
		return nil, fmt.Errorf(
			"漫画格关系无效: %w",
			err,
		)
	}

	visualPrompt =
		strings.TrimSpace(visualPrompt)
	aociText =
		strings.TrimSpace(aociText)

	if visualPrompt == "" ||
		aociText == "" {
		return nil, fmt.Errorf(
			"漫画格提示词和IAOCI不能为空",
		)
	}

	updated, err := scanCoursewareComicPanel(
		database.DB.QueryRow(
			ctx,
			`UPDATE courseware_comic_panels panel
SET visual_prompt = $1,
    negative_prompt = $2,
    aoci_text = $3,
    relations_json = $4::jsonb,
    status = CASE
        WHEN panel.status = $5 THEN $6
        ELSE panel.status
    END,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE panel.id = $7
  AND panel.project_id = $8
  AND panel.version = $9
  AND panel.status IN ($10, $11, $12, $13)
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_projects project
      WHERE project.id = panel.project_id
        AND project.courseware_id = $14
        AND project.created_by = $15
        AND project.status NOT IN ($16, $17)
  )
RETURNING `+coursewareComicPanelSelectColumns,
			visualPrompt,
			strings.TrimSpace(negativePrompt),
			aociText,
			relationsJSON,
			models.CWComicPanelStatusGenerated,
			models.CWComicPanelStatusStale,
			panelID,
			projectID,
			expectedVersion,
			models.CWComicPanelStatusPlanned,
			models.CWComicPanelStatusGenerated,
			models.CWComicPanelStatusFailed,
			models.CWComicPanelStatusStale,
			coursewareID,
			userID,
			models.CWComicProjectStatusInserted,
			models.CWComicProjectStatusArchived,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicPanelConflict
	}
	if err != nil {
		return nil, fmt.Errorf(
			"保存漫画格提示词失败: %w",
			err,
		)
	}

	return updated, nil
}

// ClaimCoursewareComicPanelGeneration 领取一个单格生成任务。
func ClaimCoursewareComicPanelGeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	expectedVersion int,
) (*models.CoursewareComicPanel, error) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开启漫画格领取事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	updated, err := scanCoursewareComicPanel(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_comic_panels panel
SET status = $1,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE panel.id = $2
  AND panel.project_id = $3
  AND panel.version = $4
  AND panel.status IN ($5, $6, $7)
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_projects project
      WHERE project.id = panel.project_id
        AND project.courseware_id = $8
        AND project.created_by = $9
        AND project.status IN ($10, $11, $12)
  )
RETURNING `+coursewareComicPanelSelectColumns,
			models.CWComicPanelStatusGenerating,
			panelID,
			projectID,
			expectedVersion,
			models.CWComicPanelStatusPlanned,
			models.CWComicPanelStatusFailed,
			models.CWComicPanelStatusStale,
			coursewareID,
			userID,
			models.CWComicProjectStatusPlanned,
			models.CWComicProjectStatusGenerating,
			models.CWComicProjectStatusFailed,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicPanelNotGeneratable
	}
	if err != nil {
		return nil, fmt.Errorf(
			"领取漫画格生成任务失败: %w",
			err,
		)
	}

	tag, err := tx.Exec(
		ctx,
		`UPDATE courseware_comic_projects
SET status = $1,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND created_by = $4
  AND status IN ($5, $6, $7)`,
		models.CWComicProjectStatusGenerating,
		projectID,
		coursewareID,
		userID,
		models.CWComicProjectStatusPlanned,
		models.CWComicProjectStatusGenerating,
		models.CWComicProjectStatusFailed,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"更新漫画项目生成状态失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return nil,
			ErrCoursewareComicProjectConflict
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交漫画格领取事务失败: %w",
			err,
		)
	}

	return updated, nil
}

// FailCoursewareComicPanelGeneration 标记一个单格生成失败。
func FailCoursewareComicPanelGeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	failure error,
) error {
	message := "漫画格图片生成失败"
	if failure != nil {
		message = strings.TrimSpace(
			failure.Error(),
		)
	}
	if message == "" {
		message = "漫画格图片生成失败"
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启漫画格失败事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(
		ctx,
		`UPDATE courseware_comic_panels panel
SET status = $1,
    last_error = $2,
    version = version + 1,
    updated_at = now()
WHERE panel.id = $3
  AND panel.project_id = $4
  AND panel.status = $5
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_projects project
      WHERE project.id = panel.project_id
        AND project.courseware_id = $6
        AND project.created_by = $7
  )`,
		models.CWComicPanelStatusFailed,
		message,
		panelID,
		projectID,
		models.CWComicPanelStatusGenerating,
		coursewareID,
		userID,
	)
	if err != nil {
		return fmt.Errorf(
			"标记漫画格生成失败异常: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareComicPanelConflict
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE courseware_comic_projects
SET status = $1,
    last_error = $2,
    version = version + 1,
    updated_at = now()
WHERE id = $3
  AND courseware_id = $4
  AND created_by = $5
  AND status = $6`,
		models.CWComicProjectStatusFailed,
		message,
		projectID,
		coursewareID,
		userID,
		models.CWComicProjectStatusGenerating,
	); err != nil {
		return fmt.Errorf(
			"标记漫画项目生成失败异常: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交漫画格失败事务失败: %w",
			err,
		)
	}

	return nil
}

func normalizeCoursewareComicPanel(
	item *models.CoursewareComicPanel,
) error {
	item.ProjectID =
		strings.TrimSpace(item.ProjectID)
	item.ImageKey =
		strings.TrimSpace(item.ImageKey)
	item.StoryPurpose =
		strings.TrimSpace(item.StoryPurpose)
	item.KnowledgeClaim =
		strings.TrimSpace(item.KnowledgeClaim)
	item.SceneText =
		strings.TrimSpace(item.SceneText)
	item.ActionText =
		strings.TrimSpace(item.ActionText)
	item.CameraText =
		strings.TrimSpace(item.CameraText)
	item.NarrationText =
		strings.TrimSpace(item.NarrationText)
	item.KnowledgePresentation =
		strings.TrimSpace(
			item.KnowledgePresentation,
		)
	item.VisualPrompt =
		strings.TrimSpace(item.VisualPrompt)
	item.NegativePrompt =
		strings.TrimSpace(item.NegativePrompt)
	item.AOCIText =
		strings.TrimSpace(item.AOCIText)
	item.Status =
		strings.TrimSpace(item.Status)
	item.LastError =
		strings.TrimSpace(item.LastError)

	if item.Status == "" {
		item.Status =
			models.CWComicPanelStatusPlanned
	}
	if item.OverlayVersion < 1 {
		item.OverlayVersion = 1
	}
	if item.Version < 1 {
		item.Version = 1
	}

	if item.ProjectID == "" {
		return fmt.Errorf(
			"project_id不能为空",
		)
	}
	if item.PanelNo < 1 ||
		item.PanelNo > 8 {
		return fmt.Errorf(
			"panel_no必须为1至8",
		)
	}
	if !cwComicIsImageKey(item.ImageKey) {
		return fmt.Errorf(
			"image_key不合法",
		)
	}
	if item.StoryPurpose == "" ||
		item.KnowledgeClaim == "" {
		return fmt.Errorf(
			"故事职责和知识结论不能为空",
		)
	}
	if item.VisualPrompt == "" ||
		item.AOCIText == "" {
		return fmt.Errorf(
			"图片提示词和IAOCI不能为空",
		)
	}
	if !models.IsValidCWComicPanelStatus(
		item.Status,
	) {
		return fmt.Errorf(
			"漫画格状态不合法",
		)
	}

	var err error

	item.CharacterIDsJSON, err =
		cwComicNormalizeJSON(
			item.CharacterIDsJSON,
			"[]",
			"array",
		)
	if err != nil {
		return fmt.Errorf(
			"人物ID列表无效: %w",
			err,
		)
	}

	item.DialoguesJSON, err =
		cwComicNormalizeJSON(
			item.DialoguesJSON,
			"[]",
			"array",
		)
	if err != nil {
		return fmt.Errorf(
			"对白列表无效: %w",
			err,
		)
	}

	item.RelationsJSON, err =
		cwComicNormalizeJSON(
			item.RelationsJSON,
			"[]",
			"array",
		)
	if err != nil {
		return fmt.Errorf(
			"漫画关系无效: %w",
			err,
		)
	}

	item.OverlayDocumentJSON, err =
		cwComicNormalizeJSON(
			item.OverlayDocumentJSON,
			"{}",
			"object",
		)
	if err != nil {
		return fmt.Errorf(
			"覆盖层文档无效: %w",
			err,
		)
	}

	return nil
}

func cwComicIsImageKey(
	value string,
) bool {
	value = strings.TrimSpace(value)

	if len(value) != 15 ||
		!strings.HasPrefix(value, "@I-") {
		return false
	}

	for _, code := range value[3:] {
		if !strings.ContainsRune(
			"0123456789ABCDEF",
			code,
		) {
			return false
		}
	}

	return true
}
