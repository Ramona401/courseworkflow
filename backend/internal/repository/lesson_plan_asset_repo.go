package repository

// lesson_plan_asset_repo.go — 教案附属资产数据访问层
//
// v123 新增:为教案系统的图片资产提供 CRUD 能力
//
// 提供函数:
//   - CreateLessonPlanAsset      创建资产记录
//   - GetLessonPlanAssetByID     按 ID 查询单条资产
//   - ListLessonPlanAssets       按教案 ID 列出所有资产
//   - UpdateLessonPlanAssetAltText  更新 alt 文本
//   - DeleteLessonPlanAsset      删除资产(物理记录,文件由 service 层负责清理)
//   - DeleteLessonPlanAssetsByPlanID  级联删除某教案下所有资产记录
//
// 注意:
//   - 教案删除时,数据库 ON DELETE CASCADE 会自动清掉 lesson_plan_assets 行,
//     物理文件清理由 service 层在删除教案前显式调用 ListLessonPlanAssets
//     拿到所有 file_path 后再批量删除磁盘文件。

import (
        "context"
        "errors"
        "fmt"
        "time"

        "github.com/jackc/pgx/v5"
        "tedna/internal/database"
        "tedna/internal/models"
)

// ErrLessonPlanAssetNotFound 资产不存在错误
var ErrLessonPlanAssetNotFound = errors.New("教案资产不存在")

// ==================== 创建 ====================

// CreateLessonPlanAsset 创建资产记录
// 调用前 service 层应已完成文件物理保存,这里只写数据库
func CreateLessonPlanAsset(ctx context.Context, a *models.LessonPlanAsset) error {
        if a.AssetType == "" {
                a.AssetType = models.AssetTypeImage
        }
        query := `
                INSERT INTO lesson_plan_assets
                        (lesson_plan_id, uploader_id, asset_type, file_name, file_path,
                         file_size, mime_type, alt_text, width, height)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                RETURNING id, created_at, updated_at`
        return database.DB.QueryRow(ctx, query,
                a.LessonPlanID, a.UploaderID, a.AssetType, a.FileName, a.FilePath,
                a.FileSize, a.MimeType, a.AltText, a.Width, a.Height,
        ).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// ==================== 查询 ====================

// GetLessonPlanAssetByID 按 ID 查询单条资产
func GetLessonPlanAssetByID(ctx context.Context, id string) (*models.LessonPlanAsset, error) {
        query := `
                SELECT id, lesson_plan_id, uploader_id, asset_type, file_name, file_path,
                       file_size, mime_type, alt_text, width, height, created_at, updated_at
                FROM lesson_plan_assets WHERE id = $1`
        a := &models.LessonPlanAsset{}
        err := database.DB.QueryRow(ctx, query, id).Scan(
                &a.ID, &a.LessonPlanID, &a.UploaderID, &a.AssetType, &a.FileName, &a.FilePath,
                &a.FileSize, &a.MimeType, &a.AltText, &a.Width, &a.Height,
                &a.CreatedAt, &a.UpdatedAt,
        )
        if err != nil {
                if errors.Is(err, pgx.ErrNoRows) {
                        return nil, ErrLessonPlanAssetNotFound
                }
                return nil, fmt.Errorf("查询资产失败: %w", err)
        }
        return a, nil
}

// ListLessonPlanAssets 按教案 ID 列出所有资产
// LEFT JOIN users 填充上传者显示名
// 按创建时间降序(最新上传的在前)
func ListLessonPlanAssets(ctx context.Context, planID string) ([]*models.LessonPlanAssetListItem, int, error) {
        query := `
                SELECT a.id, a.lesson_plan_id, a.uploader_id, a.asset_type, a.file_name, a.file_path,
                       a.file_size, a.mime_type, a.alt_text, a.width, a.height,
                       a.created_at, a.updated_at,
                       COALESCE(u.display_name, '') AS uploader_name
                FROM lesson_plan_assets a
                LEFT JOIN users u ON u.id = a.uploader_id
                WHERE a.lesson_plan_id = $1
                ORDER BY a.created_at DESC`
        rows, err := database.DB.Query(ctx, query, planID)
        if err != nil {
                return nil, 0, fmt.Errorf("查询资产列表失败: %w", err)
        }
        defer rows.Close()

        var items []*models.LessonPlanAssetListItem
        for rows.Next() {
                item := &models.LessonPlanAssetListItem{}
                err := rows.Scan(
                        &item.ID, &item.LessonPlanID, &item.UploaderID, &item.AssetType,
                        &item.FileName, &item.FilePath, &item.FileSize, &item.MimeType,
                        &item.AltText, &item.Width, &item.Height,
                        &item.CreatedAt, &item.UpdatedAt,
                        &item.UploaderName,
                )
                if err != nil {
                        return nil, 0, fmt.Errorf("扫描资产行失败: %w", err)
                }
                items = append(items, item)
        }
        return items, len(items), nil
}

// ==================== 更新 ====================

// UpdateLessonPlanAssetAltText 仅更新 alt 文本(目前唯一可改字段)
func UpdateLessonPlanAssetAltText(ctx context.Context, id string, altText string) error {
        result, err := database.DB.Exec(ctx,
                `UPDATE lesson_plan_assets SET alt_text = $1, updated_at = $2 WHERE id = $3`,
                altText, time.Now(), id,
        )
        if err != nil {
                return fmt.Errorf("更新 alt 文本失败: %w", err)
        }
        if result.RowsAffected() == 0 {
                return ErrLessonPlanAssetNotFound
        }
        return nil
}

// ==================== 删除 ====================

// DeleteLessonPlanAsset 删除单条资产(只删数据库行,文件由 service 层删)
func DeleteLessonPlanAsset(ctx context.Context, id string) error {
        result, err := database.DB.Exec(ctx, `DELETE FROM lesson_plan_assets WHERE id = $1`, id)
        if err != nil {
                return fmt.Errorf("删除资产失败: %w", err)
        }
        if result.RowsAffected() == 0 {
                return ErrLessonPlanAssetNotFound
        }
        return nil
}

// DeleteLessonPlanAssetsByPlanID 级联删除某教案下所有资产记录
// 通常不会单独调用——因为表上已有 ON DELETE CASCADE,
// 但保留此函数供 service 层在删除教案前主动调用以拿到 file_path 列表清理磁盘
func DeleteLessonPlanAssetsByPlanID(ctx context.Context, planID string) error {
        _, err := database.DB.Exec(ctx,
                `DELETE FROM lesson_plan_assets WHERE lesson_plan_id = $1`, planID,
        )
        return err
}
