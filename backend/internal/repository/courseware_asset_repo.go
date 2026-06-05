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
//   - 视频锚点轮 新增 metadata（jsonb 列，用 ::text 转字符串后 COALESCE 兜底空串，对齐 Go string 字段）
const cwAssetSelectColumns = `id, courseware_id, page_id, COALESCE(placeholder_id,''),
	asset_type, COALESCE(generation_prompt,''), COALESCE(oss_url,''),
	COALESCE(public_oss_url,''), COALESCE(file_size,0), COALESCE(mime_type,''),
	COALESCE(metadata::text,''), status, created_at`

// scanCWAsset 统一的行扫描（列顺序与 cwAssetSelectColumns 严格对齐）
// 注意：Metadata 紧跟 MimeType 之后、Status 之前，与 SELECT 列顺序一致
func scanCWAsset(row interface {
	Scan(dest ...interface{}) error
}, a *models.CoursewareAsset) error {
	return row.Scan(
		&a.ID, &a.CoursewareID, &a.PageID, &a.PlaceholderID,
		&a.AssetType, &a.GenerationPrompt, &a.OssURL,
		&a.PublicOSSURL, &a.FileSize, &a.MimeType,
		&a.Metadata, &a.Status, &a.CreatedAt,
	)
}

// nullIfEmptyJSON 把 Go string 形态的 JSON 转为可安全写入 jsonb 列的参数。
//   - 空白串 → 返回 nil（写入 SQL NULL，避免空串 "" 直接写 jsonb 报无效 JSON 错）
//   - 非空串 → 原样返回（pgx 把字符串绑定给 jsonb 列时由 Postgres 解析，调用方须保证是合法 JSON）
// 用于 CreateCWAsset 的 metadata 入参与 UpdateCWAssetMetadata。
func nullIfEmptyJSON(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// CreateCWAsset 创建课件多媒体资源记录
// 视频锚点轮：INSERT 接通 metadata 列（空串经 nullIfEmptyJSON 写 NULL，不破坏现有四个调用方——
// 它们均未设置 asset.Metadata，零值空串自动落 NULL，行为与之前完全一致）。
func CreateCWAsset(ctx context.Context, asset *models.CoursewareAsset) error {
	sql := `INSERT INTO courseware_assets (id, courseware_id, page_id, placeholder_id,
		asset_type, generation_prompt, oss_url, file_size, mime_type, metadata, status)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`
	return database.DB.QueryRow(ctx, sql,
		asset.CoursewareID, asset.PageID, asset.PlaceholderID,
		asset.AssetType, asset.GenerationPrompt, asset.OssURL,
		asset.FileSize, asset.MimeType, nullIfEmptyJSON(asset.Metadata), asset.Status,
	).Scan(&asset.ID, &asset.CreatedAt)
}

// GetCWAssetByID 根据ID获取多媒体资源
func GetCWAssetByID(ctx context.Context, id string) (*models.CoursewareAsset, error) {
	sql := `SELECT ` + cwAssetSelectColumns + ` FROM courseware_assets WHERE id = $1`
	a := &models.CoursewareAsset{}
	if err := scanCWAsset(database.DB.QueryRow(ctx, sql, id), a); err != nil {
		return nil, err
	}
	return a, nil
}

// ListCWAssetsByPage 获取指定页面的所有多媒体资源
func ListCWAssetsByPage(ctx context.Context, pageID string) ([]*models.CoursewareAsset, error) {
	sql := `SELECT ` + cwAssetSelectColumns + ` FROM courseware_assets WHERE page_id = $1
		ORDER BY created_at ASC`
	rows, err := database.DB.Query(ctx, sql, pageID)
	if err != nil {
		return nil, fmt.Errorf("查询页面资源列表失败: %w", err)
	}
	defer rows.Close()

	var assets []*models.CoursewareAsset
	for rows.Next() {
		a := &models.CoursewareAsset{}
		if err := scanCWAsset(rows, a); err != nil {
			return nil, fmt.Errorf("扫描资源行失败: %w", err)
		}
		assets = append(assets, a)
	}
	return assets, nil
}

// ListCWAssetsByCourseware 获取课件的所有多媒体资源
func ListCWAssetsByCourseware(ctx context.Context, coursewareID string) ([]*models.CoursewareAsset, error) {
	sql := `SELECT ` + cwAssetSelectColumns + ` FROM courseware_assets WHERE courseware_id = $1
		ORDER BY created_at ASC`
	rows, err := database.DB.Query(ctx, sql, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("查询课件资源列表失败: %w", err)
	}
	defer rows.Close()

	var assets []*models.CoursewareAsset
	for rows.Next() {
		a := &models.CoursewareAsset{}
		if err := scanCWAsset(rows, a); err != nil {
			return nil, fmt.Errorf("扫描资源行失败: %w", err)
		}
		assets = append(assets, a)
	}
	return assets, nil
}

// UpdateCWAssetStatus 更新资源状态
func UpdateCWAssetStatus(ctx context.Context, id string, status string) error {
	sql := `UPDATE courseware_assets SET status = $1 WHERE id = $2`
	_, err := database.DB.Exec(ctx, sql, status, id)
	return err
}

// UpdateCWAssetOSSURL 上传成功后更新OSS链接（此处的oss_url仍为本地路径语义，视频生成下载后回填本地URL）
func UpdateCWAssetOSSURL(ctx context.Context, id string, ossURL string, fileSize int64, mimeType string) error {
	sql := `UPDATE courseware_assets SET oss_url = $1, file_size = $2, mime_type = $3, status = $4
		WHERE id = $5`
	_, err := database.DB.Exec(ctx, sql, ossURL, fileSize, mimeType, models.CWAssetStatusUploaded, id)
	return err
}

// UpdateCWAssetPublicURL 上传到阿里云OSS成功后，回写公网URL到 public_oss_url 列
// oss_url（本地路径）保持不变，供删本地文件/插入HTML继续使用
func UpdateCWAssetPublicURL(ctx context.Context, id string, publicURL string) error {
	sql := `UPDATE courseware_assets SET public_oss_url = $1 WHERE id = $2`
	_, err := database.DB.Exec(ctx, sql, publicURL, id)
	return err
}

// UpdateCWAssetMetadata 更新资源的 metadata（jsonb 列）。
//
// 视频锚点轮新增。用途：
//   - 视频生成：写入 {"source_frame_asset_id":"<首帧图asset_id>"} 做首帧溯源；
//   - 视频上传：写入 ffprobe 提取的元数据（duration/width/height/codec/fps/bit_rate）。
//
// 入参 metadataJSON 须为合法 JSON 字符串；空串经 nullIfEmptyJSON 写 NULL（清空 metadata）。
// 调用方负责用 encoding/json 序列化，避免手拼字符串导致非法 JSON。
func UpdateCWAssetMetadata(ctx context.Context, id string, metadataJSON string) error {
	sql := `UPDATE courseware_assets SET metadata = $1 WHERE id = $2`
	_, err := database.DB.Exec(ctx, sql, nullIfEmptyJSON(metadataJSON), id)
	return err
}

// DeleteCWAsset 删除多媒体资源
func DeleteCWAsset(ctx context.Context, id string) error {
	sql := `DELETE FROM courseware_assets WHERE id = $1`
	_, err := database.DB.Exec(ctx, sql, id)
	return err
}

// GetCWAssetByPlaceholder 根据页面ID和占位符ID获取资源
func GetCWAssetByPlaceholder(ctx context.Context, pageID string, placeholderID string) (*models.CoursewareAsset, error) {
	sql := `SELECT ` + cwAssetSelectColumns + ` FROM courseware_assets
		WHERE page_id = $1 AND placeholder_id = $2
		ORDER BY created_at DESC LIMIT 1`
	a := &models.CoursewareAsset{}
	if err := scanCWAsset(database.DB.QueryRow(ctx, sql, pageID, placeholderID), a); err != nil {
		return nil, err
	}
	return a, nil
}

// ==================== 课件组件萃取记录 ====================

// CreateCWComponentExtraction 创建组件萃取记录
func CreateCWComponentExtraction(ctx context.Context, sourceType string, sourceID *string, componentID *string) (string, error) {
	var id string
	sql := `INSERT INTO courseware_component_extractions (id, source_type, source_id, component_id, status)
		VALUES (gen_random_uuid(), $1, $2, $3, 'pending')
		RETURNING id`
	err := database.DB.QueryRow(ctx, sql, sourceType, sourceID, componentID).Scan(&id)
	return id, err
}

// UpdateCWComponentExtractionStatus 更新萃取记录状态
func UpdateCWComponentExtractionStatus(ctx context.Context, id string, status string) error {
	sql := `UPDATE courseware_component_extractions SET status = $1 WHERE id = $2`
	_, err := database.DB.Exec(ctx, sql, status, id)
	return err
}
