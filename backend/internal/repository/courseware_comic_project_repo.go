package repository

// courseware_comic_project_repo.go — 知识点漫画项目仓储
//
// 本文件负责：
//   - 创建漫画项目并固化课程、教材和教育域快照；
//   - 按课件和创建者双边界读取项目；
//   - 保存尚未正式生图的项目草稿；
//   - 使用version字段领取AI规划状态，防止旧标签页覆盖新草稿；
//   - 保存项目级风格IAOCI、人物设定和连续性账本；
//   - 执行通用项目状态CAS迁移；
//   - 在最终HTML插页后记录稳定页面ID。
//
// 权限说明：
//   - 仓储使用courseware_id、created_by和课件作者三重边界；
//   - HTTP层仍必须先构建可信Actor并经作者运行通道授权；
//   - 本仓储不接受前端提交的学校、角色或教育域作为授权依据；
//   - 第一版只允许K12项目，教材和知识点均须存在创建快照。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrCoursewareComicProjectNotFound = errors.New(
		"知识点漫画项目不存在",
	)

	ErrCoursewareComicProjectConflict = errors.New(
		"知识点漫画项目已发生变化，请刷新后重试",
	)

	ErrCoursewareComicProjectNotEditable = errors.New(
		"知识点漫画项目当前不可编辑",
	)

	ErrCoursewareComicEducationDomainUnsupported = errors.New(
		"知识点漫画第一版仅支持K12教育域",
	)

	ErrCoursewareComicAssetInvalid = errors.New(
		"漫画图片资产不存在、不是图片或不属于当前课件",
	)
)

const coursewareComicProjectSelectColumns = `
id,
courseware_id,
created_by,
education_domain,
title,
subject,
grade,
publisher_snapshot,
semester_snapshot,
textbook_unit_id,
textbook_unit_snapshot::text,
knowledge_points_json::text,
knowledge_content_snapshot,
teacher_focus,
assistant_id,
narrative_mode,
visual_style,
panel_count,
layout_mode,
page_layout_json::text,
interaction_config_json::text,
style_aoci_text,
character_bible_json::text,
continuity_ledger_json::text,
character_sheet_asset_id,
status,
inserted_page_id,
inserted_page_number_snapshot,
version,
last_error,
created_at,
updated_at`

func scanCoursewareComicProject(
	scanner interface {
		Scan(dest ...interface{}) error
	},
) (*models.CoursewareComicProject, error) {
	item := &models.CoursewareComicProject{}

	err := scanner.Scan(
		&item.ID,
		&item.CoursewareID,
		&item.CreatedBy,
		&item.EducationDomain,
		&item.Title,
		&item.Subject,
		&item.Grade,
		&item.PublisherSnapshot,
		&item.SemesterSnapshot,
		&item.TextbookUnitID,
		&item.TextbookUnitSnapshotJSON,
		&item.KnowledgePointsJSON,
		&item.KnowledgeContentSnapshot,
		&item.TeacherFocus,
		&item.AssistantID,
		&item.NarrativeMode,
		&item.VisualStyle,
		&item.PanelCount,
		&item.LayoutMode,
		&item.PageLayoutJSON,
		&item.InteractionConfigJSON,
		&item.StyleAOCIText,
		&item.CharacterBibleJSON,
		&item.ContinuityLedgerJSON,
		&item.CharacterSheetAssetID,
		&item.Status,
		&item.InsertedPageID,
		&item.InsertedPageNumberSnapshot,
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

// CreateCoursewareComicProject 创建漫画项目。
//
// 创建过程使用课件行锁串行化，并重新读取数据库中的课件教育域。
// 只有课件作者本人且课件仍存在时才能落库。
func CreateCoursewareComicProject(
	ctx context.Context,
	item *models.CoursewareComicProject,
) error {
	if err := normalizeCoursewareComicProjectInput(
		item,
	); err != nil {
		return err
	}

	if err := validateCoursewareComicProjectCreate(
		item,
	); err != nil {
		return err
	}

	if item.Status !=
		models.CWComicProjectStatusDraft {
		return fmt.Errorf(
			"新漫画项目必须从draft状态开始",
		)
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启漫画项目事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var storedEducationDomain string

	err = tx.QueryRow(
		ctx,
		`SELECT LOWER(TRIM(education_domain))
FROM coursewares
WHERE id = $1
  AND user_id = $2
  AND deleted_at IS NULL
FOR UPDATE`,
		item.CoursewareID,
		item.CreatedBy,
	).Scan(&storedEducationDomain)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCoursewareComicProjectNotFound
	}
	if err != nil {
		return fmt.Errorf(
			"锁定漫画项目所属课件失败: %w",
			err,
		)
	}

	if storedEducationDomain !=
		item.EducationDomain {
		return fmt.Errorf(
			"%w：课件教育域快照与项目不一致",
			ErrCoursewareComicEducationDomainUnsupported,
		)
	}

	created, err := scanCoursewareComicProject(
		tx.QueryRow(
			ctx,
			`INSERT INTO courseware_comic_projects (
courseware_id,
created_by,
education_domain,
title,
subject,
grade,
publisher_snapshot,
semester_snapshot,
textbook_unit_id,
textbook_unit_snapshot,
knowledge_points_json,
knowledge_content_snapshot,
teacher_focus,
assistant_id,
narrative_mode,
visual_style,
panel_count,
layout_mode,
page_layout_json,
interaction_config_json,
style_aoci_text,
character_bible_json,
continuity_ledger_json,
character_sheet_asset_id,
status,
inserted_page_id,
inserted_page_number_snapshot,
version,
last_error
)
VALUES (
$1, $2, $3, $4, $5,
$6, $7, $8, $9, $10::jsonb,
$11::jsonb, $12, $13, $14, $15,
$16, $17, $18, $19::jsonb, $20::jsonb,
$21, $22::jsonb, $23::jsonb, $24, $25,
$26, $27, $28, $29
)
RETURNING `+coursewareComicProjectSelectColumns,
			item.CoursewareID,
			item.CreatedBy,
			item.EducationDomain,
			item.Title,
			item.Subject,
			item.Grade,
			item.PublisherSnapshot,
			item.SemesterSnapshot,
			cwComicNullableString(
				item.TextbookUnitID,
			),
			item.TextbookUnitSnapshotJSON,
			item.KnowledgePointsJSON,
			item.KnowledgeContentSnapshot,
			item.TeacherFocus,
			cwComicNullableString(
				item.AssistantID,
			),
			item.NarrativeMode,
			item.VisualStyle,
			item.PanelCount,
			item.LayoutMode,
			item.PageLayoutJSON,
			item.InteractionConfigJSON,
			item.StyleAOCIText,
			item.CharacterBibleJSON,
			item.ContinuityLedgerJSON,
			cwComicNullableString(
				item.CharacterSheetAssetID,
			),
			item.Status,
			cwComicNullableString(
				item.InsertedPageID,
			),
			item.InsertedPageNumberSnapshot,
			item.Version,
			item.LastError,
		),
	)
	if err != nil {
		return fmt.Errorf(
			"创建知识点漫画项目失败: %w",
			err,
		)
	}

	*item = *created

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交漫画项目事务失败: %w",
			err,
		)
	}

	return nil
}

// GetCoursewareComicProjectByIDForUser 按课件、项目和创建者读取。
func GetCoursewareComicProjectByIDForUser(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
) (*models.CoursewareComicProject, error) {
	item, err := scanCoursewareComicProject(
		database.DB.QueryRow(
			ctx,
			`SELECT `+
				coursewareComicProjectSelectColumns+
				` FROM courseware_comic_projects
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3`,
			strings.TrimSpace(projectID),
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(userID),
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf(
			"读取知识点漫画项目失败: %w",
			err,
		)
	}

	return item, nil
}

// ListCoursewareComicProjectsByCourseware 返回作者在当前课件中的漫画项目。
func ListCoursewareComicProjectsByCourseware(
	ctx context.Context,
	coursewareID string,
	userID string,
) ([]*models.CoursewareComicProject, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT `+
			coursewareComicProjectSelectColumns+
			` FROM courseware_comic_projects
WHERE courseware_id = $1
  AND created_by = $2
ORDER BY updated_at DESC, created_at DESC`,
		strings.TrimSpace(coursewareID),
		strings.TrimSpace(userID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询知识点漫画项目失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CoursewareComicProject,
		0,
	)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareComicProject(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描知识点漫画项目失败: %w",
				scanErr,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历知识点漫画项目失败: %w",
			err,
		)
	}

	return items, nil
}

// UpdateCoursewareComicProjectDraft 保存尚未正式生图的项目设置。
//
// expectedVersion用于防止多个浏览器标签页互相覆盖。
// 已存在generating或generated漫画格时拒绝回写来源快照。
func UpdateCoursewareComicProjectDraft(
	ctx context.Context,
	item *models.CoursewareComicProject,
	expectedVersion int,
) (*models.CoursewareComicProject, error) {
	if item == nil {
		return nil, fmt.Errorf(
			"漫画项目对象为空",
		)
	}

	if expectedVersion < 1 {
		return nil, fmt.Errorf(
			"漫画项目版本号不合法",
		)
	}

	if err := normalizeCoursewareComicProjectInput(
		item,
	); err != nil {
		return nil, err
	}

	if err := validateCoursewareComicProjectCreate(
		item,
	); err != nil {
		return nil, err
	}

	updated, err := scanCoursewareComicProject(
		database.DB.QueryRow(
			ctx,
			`UPDATE courseware_comic_projects project
SET title = $1,
    publisher_snapshot = $2,
    semester_snapshot = $3,
    textbook_unit_id = $4,
    textbook_unit_snapshot = $5::jsonb,
    knowledge_points_json = $6::jsonb,
    knowledge_content_snapshot = $7,
    teacher_focus = $8,
    assistant_id = $9,
    narrative_mode = $10,
    visual_style = $11,
    panel_count = $12,
    layout_mode = $13,
    page_layout_json = $14::jsonb,
    interaction_config_json = $15::jsonb,
    style_aoci_text = '',
    character_bible_json = '{}'::jsonb,
    continuity_ledger_json = '{}'::jsonb,
    character_sheet_asset_id = NULL,
    status = $16,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $17
  AND courseware_id = $18
  AND created_by = $19
  AND version = $20
  AND status IN ($21, $22)
  AND NOT EXISTS (
      SELECT 1
      FROM courseware_comic_panels panel
      WHERE panel.project_id = project.id
        AND panel.status IN ($23, $24)
  )
RETURNING `+coursewareComicProjectSelectColumns,
			item.Title,
			item.PublisherSnapshot,
			item.SemesterSnapshot,
			cwComicNullableString(
				item.TextbookUnitID,
			),
			item.TextbookUnitSnapshotJSON,
			item.KnowledgePointsJSON,
			item.KnowledgeContentSnapshot,
			item.TeacherFocus,
			cwComicNullableString(
				item.AssistantID,
			),
			item.NarrativeMode,
			item.VisualStyle,
			item.PanelCount,
			item.LayoutMode,
			item.PageLayoutJSON,
			item.InteractionConfigJSON,
			models.CWComicProjectStatusDraft,
			item.ID,
			item.CoursewareID,
			item.CreatedBy,
			expectedVersion,
			models.CWComicProjectStatusDraft,
			models.CWComicProjectStatusFailed,
			models.CWComicPanelStatusGenerating,
			models.CWComicPanelStatusGenerated,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicProjectConflict
	}
	if err != nil {
		return nil, fmt.Errorf(
			"保存知识点漫画项目草稿失败: %w",
			err,
		)
	}

	return updated, nil
}

// BeginCoursewareComicProjectPlanning 使用版本CAS领取AI规划。
//
// 该入口必须在任何产生外部费用的AI调用之前执行。
// expectedVersion不匹配时拒绝，防止旧浏览器标签页覆盖新草稿。
func BeginCoursewareComicProjectPlanning(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	expectedVersion int,
) (*models.CoursewareComicProject, error) {
	if expectedVersion < 1 {
		return nil, fmt.Errorf(
			"漫画项目版本号不合法",
		)
	}

	updated, err := scanCoursewareComicProject(
		database.DB.QueryRow(
			ctx,
			`UPDATE courseware_comic_projects
SET status = $1,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $2
  AND courseware_id = $3
  AND created_by = $4
  AND version = $5
  AND status IN ($6, $7, $8)
RETURNING `+coursewareComicProjectSelectColumns,
			models.CWComicProjectStatusPlanning,
			strings.TrimSpace(projectID),
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(userID),
			expectedVersion,
			models.CWComicProjectStatusDraft,
			models.CWComicProjectStatusPlanned,
			models.CWComicProjectStatusFailed,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicProjectConflict
	}
	if err != nil {
		return nil, fmt.Errorf(
			"领取知识点漫画AI规划失败: %w",
			err,
		)
	}

	return updated, nil
}

// SaveCoursewareComicProjectPlanningResult 保存AI规划的项目级事实源。
//
// 只允许planning状态且版本必须匹配。
// 本入口不改变planning状态，随后由分格仓储在同批分镜落库后推进planned。
func SaveCoursewareComicProjectPlanningResult(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	expectedVersion int,
	styleAOCIText string,
	characterBibleJSON string,
	continuityLedgerJSON string,
) (*models.CoursewareComicProject, error) {
	if expectedVersion < 1 {
		return nil, fmt.Errorf(
			"漫画项目版本号不合法",
		)
	}

	styleAOCIText =
		strings.TrimSpace(styleAOCIText)
	if styleAOCIText == "" {
		return nil, fmt.Errorf(
			"漫画项目风格IAOCI不能为空",
		)
	}

	var err error

	characterBibleJSON, err =
		cwComicNormalizeJSON(
			characterBibleJSON,
			"{}",
			"object",
		)
	if err != nil {
		return nil, fmt.Errorf(
			"漫画人物设定无效: %w",
			err,
		)
	}

	continuityLedgerJSON, err =
		cwComicNormalizeJSON(
			continuityLedgerJSON,
			"{}",
			"object",
		)
	if err != nil {
		return nil, fmt.Errorf(
			"漫画连续性账本无效: %w",
			err,
		)
	}

	updated, err := scanCoursewareComicProject(
		database.DB.QueryRow(
			ctx,
			`UPDATE courseware_comic_projects
SET style_aoci_text = $1,
    character_bible_json = $2::jsonb,
    continuity_ledger_json = $3::jsonb,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $4
  AND courseware_id = $5
  AND created_by = $6
  AND version = $7
  AND status = $8
RETURNING `+coursewareComicProjectSelectColumns,
			styleAOCIText,
			characterBibleJSON,
			continuityLedgerJSON,
			strings.TrimSpace(projectID),
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(userID),
			expectedVersion,
			models.CWComicProjectStatusPlanning,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicProjectConflict
	}
	if err != nil {
		return nil, fmt.Errorf(
			"保存知识点漫画AI规划结果失败: %w",
			err,
		)
	}

	return updated, nil
}

// TransitionCoursewareComicProjectStatus 使用状态CAS迁移项目状态。
func TransitionCoursewareComicProjectStatus(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	expectedStatuses []string,
	targetStatus string,
	lastError string,
) (*models.CoursewareComicProject, error) {
	if len(expectedStatuses) == 0 {
		return nil, fmt.Errorf(
			"漫画项目前置状态不能为空",
		)
	}

	for _, status := range expectedStatuses {
		if !models.IsValidCWComicProjectStatus(
			status,
		) {
			return nil, fmt.Errorf(
				"漫画项目前置状态不合法: %s",
				status,
			)
		}
	}

	if !models.IsValidCWComicProjectStatus(
		targetStatus,
	) {
		return nil, fmt.Errorf(
			"漫画项目目标状态不合法: %s",
			targetStatus,
		)
	}

	updated, err := scanCoursewareComicProject(
		database.DB.QueryRow(
			ctx,
			`UPDATE courseware_comic_projects
SET status = $1,
    last_error = $2,
    version = version + 1,
    updated_at = now()
WHERE id = $3
  AND courseware_id = $4
  AND created_by = $5
  AND status = ANY($6::text[])
RETURNING `+coursewareComicProjectSelectColumns,
			targetStatus,
			strings.TrimSpace(lastError),
			strings.TrimSpace(projectID),
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(userID),
			expectedStatuses,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicProjectConflict
	}
	if err != nil {
		return nil, fmt.Errorf(
			"迁移知识点漫画项目状态失败: %w",
			err,
		)
	}

	return updated, nil
}

// MarkCoursewareComicProjectInserted 记录最终漫画HTML所在页面。
func MarkCoursewareComicProjectInserted(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
	pageID string,
	pageNumber int,
	expectedVersion int,
) (*models.CoursewareComicProject, error) {
	pageID = strings.TrimSpace(pageID)

	if pageID == "" ||
		pageNumber < 1 ||
		expectedVersion < 1 {
		return nil, fmt.Errorf(
			"漫画插入页面定位或版本号不合法",
		)
	}

	updated, err := scanCoursewareComicProject(
		database.DB.QueryRow(
			ctx,
			`UPDATE courseware_comic_projects
SET status = $1,
    inserted_page_id = $2,
    inserted_page_number_snapshot = $3,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $4
  AND courseware_id = $5
  AND created_by = $6
  AND version = $7
  AND status = $8
RETURNING `+coursewareComicProjectSelectColumns,
			models.CWComicProjectStatusInserted,
			pageID,
			pageNumber,
			strings.TrimSpace(projectID),
			strings.TrimSpace(coursewareID),
			strings.TrimSpace(userID),
			expectedVersion,
			models.CWComicProjectStatusReady,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil,
			ErrCoursewareComicProjectConflict
	}
	if err != nil {
		return nil, fmt.Errorf(
			"记录漫画插入页面失败: %w",
			err,
		)
	}

	return updated, nil
}

func normalizeCoursewareComicProjectInput(
	item *models.CoursewareComicProject,
) error {
	if item == nil {
		return fmt.Errorf(
			"漫画项目对象为空",
		)
	}

	item.CoursewareID =
		strings.TrimSpace(item.CoursewareID)
	item.CreatedBy =
		strings.TrimSpace(item.CreatedBy)
	item.EducationDomain =
		strings.ToLower(
			strings.TrimSpace(
				item.EducationDomain,
			),
		)

	item.Title =
		strings.TrimSpace(item.Title)
	item.Subject =
		strings.TrimSpace(item.Subject)
	item.Grade =
		strings.TrimSpace(item.Grade)
	item.PublisherSnapshot =
		strings.TrimSpace(
			item.PublisherSnapshot,
		)
	item.SemesterSnapshot =
		strings.TrimSpace(
			item.SemesterSnapshot,
		)
	item.KnowledgeContentSnapshot =
		strings.TrimSpace(
			item.KnowledgeContentSnapshot,
		)
	item.TeacherFocus =
		strings.TrimSpace(item.TeacherFocus)
	item.NarrativeMode =
		strings.TrimSpace(item.NarrativeMode)
	item.VisualStyle =
		strings.TrimSpace(item.VisualStyle)
	item.LayoutMode =
		strings.TrimSpace(item.LayoutMode)
	item.StyleAOCIText =
		strings.TrimSpace(item.StyleAOCIText)
	item.Status =
		strings.TrimSpace(item.Status)
	item.LastError =
		strings.TrimSpace(item.LastError)

	if item.NarrativeMode == "" {
		item.NarrativeMode =
			"knowledge_story"
	}
	if item.VisualStyle == "" {
		item.VisualStyle =
			"science_encyclopedia"
	}
	if item.PanelCount == 0 {
		item.PanelCount = 4
	}
	if item.LayoutMode == "" {
		item.LayoutMode =
			models.CWComicLayoutGrid
	}
	if item.Status == "" {
		item.Status =
			models.CWComicProjectStatusDraft
	}
	if item.Version < 1 {
		item.Version = 1
	}

	var err error

	item.TextbookUnitSnapshotJSON, err =
		cwComicNormalizeJSON(
			item.TextbookUnitSnapshotJSON,
			"{}",
			"object",
		)
	if err != nil {
		return fmt.Errorf(
			"教材单元快照无效: %w",
			err,
		)
	}

	item.KnowledgePointsJSON, err =
		cwComicNormalizeJSON(
			item.KnowledgePointsJSON,
			"[]",
			"array",
		)
	if err != nil {
		return fmt.Errorf(
			"知识点快照无效: %w",
			err,
		)
	}

	item.PageLayoutJSON, err =
		cwComicNormalizeJSON(
			item.PageLayoutJSON,
			"{}",
			"object",
		)
	if err != nil {
		return fmt.Errorf(
			"漫画页面布局无效: %w",
			err,
		)
	}

	item.InteractionConfigJSON, err =
		cwComicNormalizeJSON(
			item.InteractionConfigJSON,
			"{}",
			"object",
		)
	if err != nil {
		return fmt.Errorf(
			"漫画互动配置无效: %w",
			err,
		)
	}

	item.CharacterBibleJSON, err =
		cwComicNormalizeJSON(
			item.CharacterBibleJSON,
			"{}",
			"object",
		)
	if err != nil {
		return fmt.Errorf(
			"人物设定无效: %w",
			err,
		)
	}

	item.ContinuityLedgerJSON, err =
		cwComicNormalizeJSON(
			item.ContinuityLedgerJSON,
			"{}",
			"object",
		)
	if err != nil {
		return fmt.Errorf(
			"连续性账本无效: %w",
			err,
		)
	}

	return nil
}

func validateCoursewareComicProjectCreate(
	item *models.CoursewareComicProject,
) error {
	if item.CoursewareID == "" ||
		item.CreatedBy == "" {
		return fmt.Errorf(
			"漫画项目课件ID和创建者不能为空",
		)
	}

	if item.EducationDomain !=
		models.EducationDomainK12 {
		return ErrCoursewareComicEducationDomainUnsupported
	}

	if item.Title == "" ||
		item.Subject == "" ||
		item.Grade == "" {
		return fmt.Errorf(
			"漫画标题、学科和年级不能为空",
		)
	}

	if item.PublisherSnapshot == "" {
		return fmt.Errorf(
			"知识点漫画必须明确教材版本",
		)
	}

	if item.TextbookUnitID == nil ||
		strings.TrimSpace(
			*item.TextbookUnitID,
		) == "" {
		return fmt.Errorf(
			"知识点漫画必须明确教材单元",
		)
	}

	if item.KnowledgeContentSnapshot == "" {
		return fmt.Errorf(
			"知识点漫画必须固化知识内容",
		)
	}

	var knowledgePoints []json.RawMessage

	if err := json.Unmarshal(
		[]byte(item.KnowledgePointsJSON),
		&knowledgePoints,
	); err != nil ||
		len(knowledgePoints) == 0 {
		return fmt.Errorf(
			"知识点漫画至少需要一个知识点快照",
		)
	}

	if item.PanelCount < 4 ||
		item.PanelCount > 8 {
		return fmt.Errorf(
			"漫画格数必须为4至8",
		)
	}

	if !models.IsValidCWComicLayoutMode(
		item.LayoutMode,
	) {
		return fmt.Errorf(
			"漫画页面布局模式不合法",
		)
	}

	if !models.IsValidCWComicProjectStatus(
		item.Status,
	) {
		return fmt.Errorf(
			"漫画项目状态不合法",
		)
	}

	return nil
}

func cwComicNormalizeJSON(
	value string,
	fallback string,
	expectedType string,
) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}

	var decoded interface{}

	if err := json.Unmarshal(
		[]byte(value),
		&decoded,
	); err != nil {
		return "", err
	}

	switch expectedType {
	case "array":
		if _, ok :=
			decoded.([]interface{}); !ok {
			return "", fmt.Errorf(
				"JSON必须是数组",
			)
		}

	case "object":
		if _, ok :=
			decoded.(map[string]interface{}); !ok {
			return "", fmt.Errorf(
				"JSON必须是对象",
			)
		}

	default:
		return "", fmt.Errorf(
			"未知JSON类型约束",
		)
	}

	normalized, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}

	return string(normalized), nil
}

func cwComicNullableString(
	value *string,
) interface{} {
	if value == nil ||
		strings.TrimSpace(*value) == "" {
		return nil
	}

	return strings.TrimSpace(*value)
}
