package repository

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 课件多媒体资源 CRUD ====================

// CreateCWAsset 创建课件多媒体资源记录
func CreateCWAsset(ctx context.Context, asset *models.CoursewareAsset) error {
	sql := `INSERT INTO courseware_assets (id, courseware_id, page_id, placeholder_id,
		asset_type, generation_prompt, oss_url, file_size, mime_type, status)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return database.DB.QueryRow(ctx, sql,
		asset.CoursewareID, asset.PageID, asset.PlaceholderID,
		asset.AssetType, asset.GenerationPrompt, asset.OssURL,
		asset.FileSize, asset.MimeType, asset.Status,
	).Scan(&asset.ID, &asset.CreatedAt)
}

// GetCWAssetByID 根据ID获取多媒体资源
func GetCWAssetByID(ctx context.Context, id string) (*models.CoursewareAsset, error) {
	sql := `SELECT id, courseware_id, page_id, COALESCE(placeholder_id,''),
		asset_type, COALESCE(generation_prompt,''), COALESCE(oss_url,''),
		COALESCE(file_size,0), COALESCE(mime_type,''), status, created_at
		FROM courseware_assets WHERE id = $1`
	a := &models.CoursewareAsset{}
	err := database.DB.QueryRow(ctx, sql, id).Scan(
		&a.ID, &a.CoursewareID, &a.PageID, &a.PlaceholderID,
		&a.AssetType, &a.GenerationPrompt, &a.OssURL,
		&a.FileSize, &a.MimeType, &a.Status, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// ListCWAssetsByPage 获取指定页面的所有多媒体资源
func ListCWAssetsByPage(ctx context.Context, pageID string) ([]*models.CoursewareAsset, error) {
	sql := `SELECT id, courseware_id, page_id, COALESCE(placeholder_id,''),
		asset_type, COALESCE(generation_prompt,''), COALESCE(oss_url,''),
		COALESCE(file_size,0), COALESCE(mime_type,''), status, created_at
		FROM courseware_assets WHERE page_id = $1
		ORDER BY created_at ASC`
	rows, err := database.DB.Query(ctx, sql, pageID)
	if err != nil {
		return nil, fmt.Errorf("查询页面资源列表失败: %w", err)
	}
	defer rows.Close()

	var assets []*models.CoursewareAsset
	for rows.Next() {
		a := &models.CoursewareAsset{}
		if err := rows.Scan(
			&a.ID, &a.CoursewareID, &a.PageID, &a.PlaceholderID,
			&a.AssetType, &a.GenerationPrompt, &a.OssURL,
			&a.FileSize, &a.MimeType, &a.Status, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描资源行失败: %w", err)
		}
		assets = append(assets, a)
	}
	return assets, nil
}

// ListCWAssetsByCourseware 获取课件的所有多媒体资源
func ListCWAssetsByCourseware(ctx context.Context, coursewareID string) ([]*models.CoursewareAsset, error) {
	sql := `SELECT id, courseware_id, page_id, COALESCE(placeholder_id,''),
		asset_type, COALESCE(generation_prompt,''), COALESCE(oss_url,''),
		COALESCE(file_size,0), COALESCE(mime_type,''), status, created_at
		FROM courseware_assets WHERE courseware_id = $1
		ORDER BY created_at ASC`
	rows, err := database.DB.Query(ctx, sql, coursewareID)
	if err != nil {
		return nil, fmt.Errorf("查询课件资源列表失败: %w", err)
	}
	defer rows.Close()

	var assets []*models.CoursewareAsset
	for rows.Next() {
		a := &models.CoursewareAsset{}
		if err := rows.Scan(
			&a.ID, &a.CoursewareID, &a.PageID, &a.PlaceholderID,
			&a.AssetType, &a.GenerationPrompt, &a.OssURL,
			&a.FileSize, &a.MimeType, &a.Status, &a.CreatedAt,
		); err != nil {
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

// UpdateCWAssetOSSURL 上传成功后更新OSS链接
func UpdateCWAssetOSSURL(ctx context.Context, id string, ossURL string, fileSize int64, mimeType string) error {
	sql := `UPDATE courseware_assets SET oss_url = $1, file_size = $2, mime_type = $3, status = $4
		WHERE id = $5`
	_, err := database.DB.Exec(ctx, sql, ossURL, fileSize, mimeType, models.CWAssetStatusUploaded, id)
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
	sql := `SELECT id, courseware_id, page_id, COALESCE(placeholder_id,''),
		asset_type, COALESCE(generation_prompt,''), COALESCE(oss_url,''),
		COALESCE(file_size,0), COALESCE(mime_type,''), status, created_at
		FROM courseware_assets WHERE page_id = $1 AND placeholder_id = $2
		ORDER BY created_at DESC LIMIT 1`
	a := &models.CoursewareAsset{}
	err := database.DB.QueryRow(ctx, sql, pageID, placeholderID).Scan(
		&a.ID, &a.CoursewareID, &a.PageID, &a.PlaceholderID,
		&a.AssetType, &a.GenerationPrompt, &a.OssURL,
		&a.FileSize, &a.MimeType, &a.Status, &a.CreatedAt,
	)
	if err != nil {
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

