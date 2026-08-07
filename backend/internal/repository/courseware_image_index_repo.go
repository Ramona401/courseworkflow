package repository

// courseware_image_index_repo.go — 图片IAOCI索引基础仓储
//
// 本文件只负责：
//   - 索引创建、覆盖和读取；
//   - 资产绑定与状态更新；
//   - 基础写入校验。
//
// 图片R关系位于courseware_image_relation_repo.go。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrCoursewareImageIndexNotFound = errors.New("课件图片索引不存在")

	ErrCoursewareImageRelationCycle = errors.New("课件图片关系形成循环")
)

const cwImageIndexSelectColumns = `
id,
courseware_id,
page_id,
placeholder_id,
image_key,
slot_order,
index_version,
index_type,
usage_role,
continuity_level,
subject_type,
aspect_ratio,
relation_count,
focus_text,
layout_text,
art_text,
character_text,
scene_text,
export_text,
negative_text,
aoci_text,
generation_prompt,
asset_id,
status,
last_error,
version,
created_at,
updated_at`

func scanCoursewareImageIndex(
	scanner interface {
		Scan(dest ...interface{}) error
	},
) (*models.CoursewareImageIndex, error) {
	item := &models.CoursewareImageIndex{}

	err := scanner.Scan(
		&item.ID,
		&item.CoursewareID,
		&item.PageID,
		&item.PlaceholderID,
		&item.ImageKey,
		&item.SlotOrder,
		&item.IndexVersion,
		&item.IndexType,
		&item.UsageRole,
		&item.ContinuityLevel,
		&item.SubjectType,
		&item.AspectRatio,
		&item.RelationCount,
		&item.FocusText,
		&item.LayoutText,
		&item.ArtText,
		&item.CharacterText,
		&item.SceneText,
		&item.ExportText,
		&item.NegativeText,
		&item.AOCIText,
		&item.GenerationPrompt,
		&item.AssetID,
		&item.Status,
		&item.LastError,
		&item.Version,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}

// CreateCoursewareImageIndex 创建索引。
func CreateCoursewareImageIndex(
	ctx context.Context,
	item *models.CoursewareImageIndex,
) error {
	if err := validateCoursewareImageIndexWrite(item); err != nil {
		return err
	}
	if err := validateCoursewareImageAssetBinding(
		ctx,
		item.CoursewareID,
		item.AssetID,
	); err != nil {
		return err
	}

	sql := `INSERT INTO courseware_image_indexes (
courseware_id,
page_id,
placeholder_id,
image_key,
slot_order,
index_version,
index_type,
usage_role,
continuity_level,
subject_type,
aspect_ratio,
relation_count,
focus_text,
layout_text,
art_text,
character_text,
scene_text,
export_text,
negative_text,
aoci_text,
generation_prompt,
asset_id,
status,
last_error,
version
)
VALUES (
$1, $2, $3, $4, $5,
$6, $7, $8, $9, $10,
$11, $12, $13, $14, $15,
$16, $17, $18, $19, $20,
$21, $22, $23, $24, $25
)
RETURNING id, created_at, updated_at`

	return database.DB.QueryRow(
		ctx,
		sql,
		item.CoursewareID,
		nullableImageIndexString(item.PageID),
		item.PlaceholderID,
		item.ImageKey,
		item.SlotOrder,
		item.IndexVersion,
		item.IndexType,
		item.UsageRole,
		item.ContinuityLevel,
		item.SubjectType,
		item.AspectRatio,
		item.RelationCount,
		item.FocusText,
		item.LayoutText,
		item.ArtText,
		item.CharacterText,
		item.SceneText,
		item.ExportText,
		item.NegativeText,
		item.AOCIText,
		item.GenerationPrompt,
		nullableImageIndexString(item.AssetID),
		item.Status,
		item.LastError,
		item.Version,
	).Scan(
		&item.ID,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
}

// UpsertCoursewarePageImageIndex 创建或覆盖页面图片索引。
func UpsertCoursewarePageImageIndex(
	ctx context.Context,
	item *models.CoursewareImageIndex,
) error {
	if err := validateCoursewareImageIndexWrite(item); err != nil {
		return err
	}
	if item.IndexType ==
		models.CWImageIndexTypeAnchor ||
		item.PageID == nil ||
		strings.TrimSpace(*item.PageID) == "" {
		return fmt.Errorf(
			"页面图片索引必须绑定page_id",
		)
	}
	if err := validateCoursewareImageAssetBinding(
		ctx,
		item.CoursewareID,
		item.AssetID,
	); err != nil {
		return err
	}

	sql := `INSERT INTO courseware_image_indexes (
courseware_id,
page_id,
placeholder_id,
image_key,
slot_order,
index_version,
index_type,
usage_role,
continuity_level,
subject_type,
aspect_ratio,
relation_count,
focus_text,
layout_text,
art_text,
character_text,
scene_text,
export_text,
negative_text,
aoci_text,
generation_prompt,
asset_id,
status,
last_error,
version
)
VALUES (
$1, $2, $3, $4, $5,
$6, $7, $8, $9, $10,
$11, $12, $13, $14, $15,
$16, $17, $18, $19, $20,
$21, $22, $23, $24, $25
)
ON CONFLICT (page_id, placeholder_id, index_type)
WHERE page_id IS NOT NULL
DO UPDATE SET
courseware_id = EXCLUDED.courseware_id,
image_key = EXCLUDED.image_key,
slot_order = EXCLUDED.slot_order,
index_version = EXCLUDED.index_version,
usage_role = EXCLUDED.usage_role,
continuity_level = EXCLUDED.continuity_level,
subject_type = EXCLUDED.subject_type,
aspect_ratio = EXCLUDED.aspect_ratio,
relation_count = EXCLUDED.relation_count,
focus_text = EXCLUDED.focus_text,
layout_text = EXCLUDED.layout_text,
art_text = EXCLUDED.art_text,
character_text = EXCLUDED.character_text,
scene_text = EXCLUDED.scene_text,
export_text = EXCLUDED.export_text,
negative_text = EXCLUDED.negative_text,
aoci_text = EXCLUDED.aoci_text,
generation_prompt = EXCLUDED.generation_prompt,
asset_id = EXCLUDED.asset_id,
status = EXCLUDED.status,
last_error = EXCLUDED.last_error,
version = courseware_image_indexes.version + 1,
updated_at = now()
RETURNING id, version, created_at, updated_at`

	return database.DB.QueryRow(
		ctx,
		sql,
		item.CoursewareID,
		*item.PageID,
		item.PlaceholderID,
		item.ImageKey,
		item.SlotOrder,
		item.IndexVersion,
		item.IndexType,
		item.UsageRole,
		item.ContinuityLevel,
		item.SubjectType,
		item.AspectRatio,
		item.RelationCount,
		item.FocusText,
		item.LayoutText,
		item.ArtText,
		item.CharacterText,
		item.SceneText,
		item.ExportText,
		item.NegativeText,
		item.AOCIText,
		item.GenerationPrompt,
		nullableImageIndexString(item.AssetID),
		item.Status,
		item.LastError,
		item.Version,
	).Scan(
		&item.ID,
		&item.Version,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
}

// GetCoursewareImageIndexByID 按ID读取。
func GetCoursewareImageIndexByID(
	ctx context.Context,
	id string,
) (*models.CoursewareImageIndex, error) {
	sql := `SELECT ` +
		cwImageIndexSelectColumns +
		` FROM courseware_image_indexes
WHERE id = $1`

	item, err := scanCoursewareImageIndex(
		database.DB.QueryRow(ctx, sql, id),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoursewareImageIndexNotFound
	}
	if err != nil {
		return nil, fmt.Errorf(
			"读取课件图片索引失败: %w",
			err,
		)
	}

	return item, nil
}

// GetCoursewareImageIndexByKey 按稳定键读取。
func GetCoursewareImageIndexByKey(
	ctx context.Context,
	coursewareID string,
	imageKey string,
) (*models.CoursewareImageIndex, error) {
	sql := `SELECT ` +
		cwImageIndexSelectColumns +
		` FROM courseware_image_indexes
WHERE courseware_id = $1
  AND image_key = $2`

	item, err := scanCoursewareImageIndex(
		database.DB.QueryRow(
			ctx,
			sql,
			coursewareID,
			imageKey,
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoursewareImageIndexNotFound
	}
	if err != nil {
		return nil, fmt.Errorf(
			"读取课件图片索引失败: %w",
			err,
		)
	}

	return item, nil
}

// ListCoursewareImageIndexesByPage 返回一页索引。
func ListCoursewareImageIndexesByPage(
	ctx context.Context,
	pageID string,
) ([]*models.CoursewareImageIndex, error) {
	sql := `SELECT ` +
		cwImageIndexSelectColumns +
		` FROM courseware_image_indexes
WHERE page_id = $1
ORDER BY slot_order ASC, created_at ASC`

	rows, err := database.DB.Query(ctx, sql, pageID)
	if err != nil {
		return nil, fmt.Errorf(
			"查询页面图片索引失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]*models.CoursewareImageIndex, 0)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareImageIndex(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描页面图片索引失败: %w",
				scanErr,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历页面图片索引失败: %w",
			err,
		)
	}

	return items, nil
}

// ListCoursewareImageIndexesByCourseware 返回课件全部索引。
func ListCoursewareImageIndexesByCourseware(
	ctx context.Context,
	coursewareID string,
) ([]*models.CoursewareImageIndex, error) {
	sql := `SELECT
idx.id,
idx.courseware_id,
idx.page_id,
idx.placeholder_id,
idx.image_key,
idx.slot_order,
idx.index_version,
idx.index_type,
idx.usage_role,
idx.continuity_level,
idx.subject_type,
idx.aspect_ratio,
idx.relation_count,
idx.focus_text,
idx.layout_text,
idx.art_text,
idx.character_text,
idx.scene_text,
idx.export_text,
idx.negative_text,
idx.aoci_text,
idx.generation_prompt,
idx.asset_id,
idx.status,
idx.last_error,
idx.version,
idx.created_at,
idx.updated_at
FROM courseware_image_indexes idx
LEFT JOIN courseware_pages page
	ON page.id = idx.page_id
WHERE idx.courseware_id = $1
ORDER BY
	CASE WHEN idx.index_type = 'A' THEN 0 ELSE 1 END,
	COALESCE(page.page_number, 0),
	idx.slot_order,
	idx.created_at`

	rows, err := database.DB.Query(
		ctx,
		sql,
		coursewareID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件图片索引失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]*models.CoursewareImageIndex, 0)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareImageIndex(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描课件图片索引失败: %w",
				scanErr,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件图片索引失败: %w",
			err,
		)
	}

	return items, nil
}

// UpdateCoursewareImageIndexAssetStatus 绑定资产并更新状态。
func UpdateCoursewareImageIndexAssetStatus(
	ctx context.Context,
	id string,
	assetID *string,
	status string,
	lastError string,
) error {
	if !models.IsValidCWImageIndexStatus(status) {
		return fmt.Errorf(
			"图片索引状态不合法: %s",
			status,
		)
	}

	if assetID != nil {
		index, err :=
			GetCoursewareImageIndexByID(ctx, id)
		if err != nil {
			return err
		}

		if err := validateCoursewareImageAssetBinding(
			ctx,
			index.CoursewareID,
			assetID,
		); err != nil {
			return err
		}
	}

	tag, err := database.DB.Exec(
		ctx,
		`UPDATE courseware_image_indexes
SET asset_id = $1,
	status = $2,
	last_error = $3,
	updated_at = now()
WHERE id = $4`,
		nullableImageIndexString(assetID),
		status,
		strings.TrimSpace(lastError),
		id,
	)
	if err != nil {
		return fmt.Errorf(
			"更新图片索引状态失败: %w",
			err,
		)
	}
	if tag.RowsAffected() != 1 {
		return ErrCoursewareImageIndexNotFound
	}

	return nil
}

func validateCoursewareImageAssetBinding(
	ctx context.Context,
	coursewareID string,
	assetID *string,
) error {
	if assetID == nil ||
		strings.TrimSpace(*assetID) == "" {
		return nil
	}

	var valid bool

	err := database.DB.QueryRow(
		ctx,
		`SELECT EXISTS(
	SELECT 1
	FROM courseware_assets
	WHERE id = $1
	  AND courseware_id = $2
	  AND asset_type = 'image'
)`,
		strings.TrimSpace(*assetID),
		coursewareID,
	).Scan(&valid)
	if err != nil {
		return fmt.Errorf(
			"校验图片资产归属失败: %w",
			err,
		)
	}
	if !valid {
		return fmt.Errorf(
			"资产不存在、不是图片或不属于索引所在课件",
		)
	}

	return nil
}

func validateCoursewareImageIndexWrite(
	item *models.CoursewareImageIndex,
) error {
	if item == nil {
		return fmt.Errorf("图片索引对象为空")
	}
	if strings.TrimSpace(item.CoursewareID) == "" {
		return fmt.Errorf(
			"图片索引courseware_id不能为空",
		)
	}
	if strings.TrimSpace(item.PlaceholderID) == "" {
		return fmt.Errorf(
			"图片索引placeholder_id不能为空",
		)
	}
	if !isRepositoryImageKey(item.ImageKey) {
		return fmt.Errorf(
			"图片索引image_key不合法",
		)
	}
	if item.IndexVersion < 1 {
		return fmt.Errorf(
			"index_version必须大于等于1",
		)
	}
	if !models.IsValidCWImageIndexType(
		item.IndexType,
	) {
		return fmt.Errorf("index_type不合法")
	}
	if !models.IsValidCWImageUsageRole(
		item.UsageRole,
	) {
		return fmt.Errorf("usage_role不合法")
	}
	if item.ContinuityLevel < 0 ||
		item.ContinuityLevel > 3 {
		return fmt.Errorf(
			"continuity_level必须为0至3",
		)
	}
	if !models.IsValidCWImageSubjectType(
		item.SubjectType,
	) {
		return fmt.Errorf("subject_type不合法")
	}
	if !models.IsValidCWImageAspectRatio(
		item.AspectRatio,
	) {
		return fmt.Errorf("aspect_ratio不合法")
	}
	if item.RelationCount != "0" &&
		item.RelationCount != "1" &&
		item.RelationCount != "M" {
		return fmt.Errorf("relation_count不合法")
	}
	if strings.TrimSpace(item.FocusText) == "" {
		return fmt.Errorf("focus_text不能为空")
	}
	if strings.TrimSpace(item.AOCIText) == "" {
		return fmt.Errorf("aoci_text不能为空")
	}
	if !models.IsValidCWImageIndexStatus(
		item.Status,
	) {
		return fmt.Errorf("status不合法")
	}

	if item.Version < 1 {
		item.Version = 1
	}

	if item.IndexType ==
		models.CWImageIndexTypeAnchor {
		if item.PageID != nil {
			return fmt.Errorf(
				"课程锚点不能绑定page_id",
			)
		}
		if item.ImageKey != "@ANCHOR" ||
			item.SlotOrder != 0 {
			return fmt.Errorf(
				"课程锚点必须使用@ANCHOR且slot_order为0",
			)
		}
	} else {
		if item.PageID == nil ||
			strings.TrimSpace(*item.PageID) == "" {
			return fmt.Errorf(
				"页面图片必须绑定page_id",
			)
		}
		if item.ImageKey == "@ANCHOR" {
			return fmt.Errorf(
				"页面图片不能使用@ANCHOR",
			)
		}
		if item.SlotOrder < 1 {
			return fmt.Errorf(
				"页面图片slot_order必须大于等于1",
			)
		}
	}

	return nil
}

func isRepositoryImageKey(value string) bool {
	value = strings.TrimSpace(value)

	if value == "@ANCHOR" {
		return true
	}
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

func nullableImageIndexString(
	value *string,
) interface{} {
	if value == nil ||
		strings.TrimSpace(*value) == "" {
		return nil
	}

	return strings.TrimSpace(*value)
}
