package repository

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 课件多媒体资源 CRUD ====================

// cwAssetSelectColumns 统一的SELECT列
//   - v0.42.11 新增 public_oss_url
//   - 视频锚点轮 新增 metadata（jsonb列，用::text转字符串后COALESCE兜底空串，对齐Go string字段）
const cwAssetSelectColumns = `id, courseware_id, page_id, COALESCE(placeholder_id,''),
        asset_type, COALESCE(generation_prompt,''), COALESCE(oss_url,''),
        COALESCE(public_oss_url,''), COALESCE(file_size,0), COALESCE(mime_type,''),
        COALESCE(metadata::text,''), status, created_at`

// scanCWAsset 统一的行扫描（列顺序与cwAssetSelectColumns严格对齐）。
func scanCWAsset(row interface {
	Scan(dest ...interface{}) error
}, a *models.CoursewareAsset) error {
	return row.Scan(
		&a.ID,
		&a.CoursewareID,
		&a.PageID,
		&a.PlaceholderID,
		&a.AssetType,
		&a.GenerationPrompt,
		&a.OssURL,
		&a.PublicOSSURL,
		&a.FileSize,
		&a.MimeType,
		&a.Metadata,
		&a.Status,
		&a.CreatedAt,
	)
}

// nullIfEmptyJSON 把Go string形态的JSON转为可安全写入jsonb列的参数。
func nullIfEmptyJSON(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}

	return s
}

// CreateCWAsset 创建课件多媒体资源记录。
//
// placeholder_id在统一仓储边界执行稳定短键转换，
// 防止漫画UUID语义键超过数据库varchar(50)限制。
func CreateCWAsset(
	ctx context.Context,
	asset *models.CoursewareAsset,
) error {
	if asset == nil {
		return fmt.Errorf(
			"课件资产对象为空",
		)
	}

	asset.PlaceholderID =
		normalizeCWAssetPlaceholderID(
			asset.PlaceholderID,
		)

	sql := `INSERT INTO courseware_assets (
id,
courseware_id,
page_id,
placeholder_id,
asset_type,
generation_prompt,
oss_url,
file_size,
mime_type,
metadata,
status
)
VALUES (
gen_random_uuid(),
$1, $2, $3, $4, $5,
$6, $7, $8, $9, $10
)
RETURNING id, created_at`

	return database.DB.QueryRow(
		ctx,
		sql,
		asset.CoursewareID,
		asset.PageID,
		asset.PlaceholderID,
		asset.AssetType,
		asset.GenerationPrompt,
		asset.OssURL,
		asset.FileSize,
		asset.MimeType,
		nullIfEmptyJSON(
			asset.Metadata,
		),
		asset.Status,
	).Scan(
		&asset.ID,
		&asset.CreatedAt,
	)
}

// GetCWAssetByID 根据ID获取多媒体资源。
func GetCWAssetByID(
	ctx context.Context,
	id string,
) (*models.CoursewareAsset, error) {
	sql :=
		`SELECT ` +
			cwAssetSelectColumns +
			` FROM courseware_assets WHERE id = $1`

	asset :=
		&models.CoursewareAsset{}

	if err :=
		scanCWAsset(
			database.DB.QueryRow(
				ctx,
				sql,
				strings.TrimSpace(id),
			),
			asset,
		); err != nil {
		return nil, err
	}

	return asset, nil
}

// ListCWAssetsByPage 获取指定页面的所有多媒体资源。
func ListCWAssetsByPage(
	ctx context.Context,
	pageID string,
) ([]*models.CoursewareAsset, error) {
	sql :=
		`SELECT ` +
			cwAssetSelectColumns +
			` FROM courseware_assets
WHERE page_id = $1
ORDER BY created_at ASC`

	rows, err :=
		database.DB.Query(
			ctx,
			sql,
			strings.TrimSpace(pageID),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"查询页面资源列表失败: %w",
				err,
			)
	}
	defer rows.Close()

	var assets []*models.CoursewareAsset

	for rows.Next() {
		asset :=
			&models.CoursewareAsset{}

		if err :=
			scanCWAsset(
				rows,
				asset,
			); err != nil {
			return nil,
				fmt.Errorf(
					"扫描资源行失败: %w",
					err,
				)
		}

		assets =
			append(
				assets,
				asset,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"遍历页面资源失败: %w",
				err,
			)
	}

	return assets, nil
}

// ListCWAssetsByCourseware 获取课件的所有多媒体资源。
func ListCWAssetsByCourseware(
	ctx context.Context,
	coursewareID string,
) ([]*models.CoursewareAsset, error) {
	sql :=
		`SELECT ` +
			cwAssetSelectColumns +
			` FROM courseware_assets
WHERE courseware_id = $1
ORDER BY created_at ASC`

	rows, err :=
		database.DB.Query(
			ctx,
			sql,
			strings.TrimSpace(
				coursewareID,
			),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"查询课件资源列表失败: %w",
				err,
			)
	}
	defer rows.Close()

	var assets []*models.CoursewareAsset

	for rows.Next() {
		asset :=
			&models.CoursewareAsset{}

		if err :=
			scanCWAsset(
				rows,
				asset,
			); err != nil {
			return nil,
				fmt.Errorf(
					"扫描资源行失败: %w",
					err,
				)
		}

		assets =
			append(
				assets,
				asset,
			)
	}

	if err := rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"遍历课件资源失败: %w",
				err,
			)
	}

	return assets, nil
}

// UpdateCWAssetStatus 更新资源状态。
func UpdateCWAssetStatus(
	ctx context.Context,
	id string,
	status string,
) error {
	sql :=
		`UPDATE courseware_assets
SET status = $1
WHERE id = $2`

	_, err :=
		database.DB.Exec(
			ctx,
			sql,
			status,
			strings.TrimSpace(id),
		)

	return err
}

// UpdateCWAssetOSSURL 上传成功后更新OSS链接。
func UpdateCWAssetOSSURL(
	ctx context.Context,
	id string,
	ossURL string,
	fileSize int64,
	mimeType string,
) error {
	sql :=
		`UPDATE courseware_assets
SET oss_url = $1,
    file_size = $2,
    mime_type = $3,
    status = $4
WHERE id = $5`

	_, err :=
		database.DB.Exec(
			ctx,
			sql,
			ossURL,
			fileSize,
			mimeType,
			models.CWAssetStatusUploaded,
			strings.TrimSpace(id),
		)

	return err
}

// UpdateCWAssetPublicURL 上传到阿里云OSS成功后回写公网URL。
func UpdateCWAssetPublicURL(
	ctx context.Context,
	id string,
	publicURL string,
) error {
	sql :=
		`UPDATE courseware_assets
SET public_oss_url = $1
WHERE id = $2`

	_, err :=
		database.DB.Exec(
			ctx,
			sql,
			publicURL,
			strings.TrimSpace(id),
		)

	return err
}

// UpdateCWAssetMetadata 更新资源metadata。
func UpdateCWAssetMetadata(
	ctx context.Context,
	id string,
	metadataJSON string,
) error {
	sql :=
		`UPDATE courseware_assets
SET metadata = $1
WHERE id = $2`

	_, err :=
		database.DB.Exec(
			ctx,
			sql,
			nullIfEmptyJSON(
				metadataJSON,
			),
			strings.TrimSpace(id),
		)

	return err
}

// DeleteCWAsset 删除多媒体资源。
func DeleteCWAsset(
	ctx context.Context,
	id string,
) error {
	sql :=
		`DELETE FROM courseware_assets
WHERE id = $1`

	_, err :=
		database.DB.Exec(
			ctx,
			sql,
			strings.TrimSpace(id),
		)

	return err
}

// GetCWAssetByPlaceholder 根据页面ID和占位符ID获取资源。
//
// 查询入口执行与创建入口相同的短键转换，
// 调用方可以继续传入原始长语义键。
func GetCWAssetByPlaceholder(
	ctx context.Context,
	pageID string,
	placeholderID string,
) (*models.CoursewareAsset, error) {
	sql :=
		`SELECT ` +
			cwAssetSelectColumns +
			` FROM courseware_assets
WHERE page_id = $1
  AND placeholder_id = $2
ORDER BY created_at DESC
LIMIT 1`

	asset :=
		&models.CoursewareAsset{}

	if err :=
		scanCWAsset(
			database.DB.QueryRow(
				ctx,
				sql,
				strings.TrimSpace(pageID),
				normalizeCWAssetPlaceholderID(
					placeholderID,
				),
			),
			asset,
		); err != nil {
		return nil, err
	}

	return asset, nil
}

// ==================== 课件组件萃取记录 ====================

// CreateCWComponentExtraction 创建组件萃取记录。
func CreateCWComponentExtraction(
	ctx context.Context,
	sourceType string,
	sourceID *string,
	componentID *string,
) (string, error) {
	var id string

	sql :=
		`INSERT INTO courseware_component_extractions (
id,
source_type,
source_id,
component_id,
status
)
VALUES (
gen_random_uuid(),
$1, $2, $3, 'pending'
)
RETURNING id`

	err :=
		database.DB.QueryRow(
			ctx,
			sql,
			sourceType,
			sourceID,
			componentID,
		).Scan(
			&id,
		)

	return id, err
}

// UpdateCWComponentExtractionStatus 更新萃取记录状态。
func UpdateCWComponentExtractionStatus(
	ctx context.Context,
	id string,
	status string,
) error {
	sql :=
		`UPDATE courseware_component_extractions
SET status = $1
WHERE id = $2`

	_, err :=
		database.DB.Exec(
			ctx,
			sql,
			status,
			strings.TrimSpace(id),
		)

	return err
}
