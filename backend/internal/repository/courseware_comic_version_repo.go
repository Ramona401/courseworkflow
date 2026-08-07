package repository

// courseware_comic_version_repo.go — 漫画格生成完成与不可变历史仓储
//
// 本文件负责：
//   - 校验生成资产属于当前课件且为图片；
//   - 将漫画格generating状态原子结算为generated；
//   - 每次成功生图创建一条不可变版本快照；
//   - 根据全项目漫画格结果更新项目ready、generating或failed状态；
//   - 按课件、项目、分格和创建者边界读取历史版本。
//
// 图片模型已经产生费用后，即使后续课件HTML替换失败，
// 图片资产和本历史版本仍必须保留，不得回滚或物理删除。

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
	ErrCoursewareComicPanelVersionNotFound = errors.New(
		"知识点漫画分格版本不存在",
	)
)

const coursewareComicPanelVersionSelectColumns = `
id,
panel_id,
version_no,
prompt_snapshot,
aoci_snapshot,
overlay_document_snapshot::text,
asset_id,
generation_source,
created_by,
created_at`

// projectStatus使用$1并由status字段确定类型。
// preserveLastError使用独立boolean参数$2，避免把同一个参数
// 同时推断成status字段类型和text比较类型，从而触发SQLSTATE 42P08。
const coursewareComicAggregateProjectUpdateSQL = `
UPDATE courseware_comic_projects
SET status = $1,
    last_error = CASE
        WHEN $2::boolean THEN last_error
        ELSE ''
    END,
    version = version + 1,
    updated_at = now()
WHERE id = $3
  AND courseware_id = $4
  AND created_by = $5`

func scanCoursewareComicPanelVersion(
	scanner interface {
		Scan(dest ...interface{}) error
	},
) (*models.CoursewareComicPanelVersion, error) {
	item :=
		&models.CoursewareComicPanelVersion{}

	err :=
		scanner.Scan(
			&item.ID,
			&item.PanelID,
			&item.VersionNo,
			&item.PromptSnapshot,
			&item.AOCISnapshot,
			&item.OverlayDocumentSnapshot,
			&item.AssetID,
			&item.GenerationSource,
			&item.CreatedBy,
			&item.CreatedAt,
		)

	if err != nil {
		return nil, err
	}

	return item, nil
}

// resolveCoursewareComicAggregateStatus
// 根据全部漫画格状态收敛项目状态。
func resolveCoursewareComicAggregateStatus(
	totalCount int,
	generatedCount int,
	failedCount int,
) string {
	if totalCount > 0 &&
		generatedCount == totalCount {
		return models.CWComicProjectStatusReady
	}

	if failedCount > 0 {
		return models.CWComicProjectStatusFailed
	}

	return models.CWComicProjectStatusGenerating
}

// CompleteCoursewareComicPanelGeneration 完成一个漫画格的图片生成。
//
// 事务顺序：
//  1. 锁项目；
//  2. 锁漫画格；
//  3. 校验资产归属；
//  4. 计算下一不可变版本号；
//  5. 写入版本快照；
//  6. 更新漫画格当前资产；
//  7. 聚合项目状态。
func CompleteCoursewareComicPanelGeneration(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
	assetID string,
	aociText string,
	generationSource string,
) (*models.CoursewareComicPanel, error) {
	assetID =
		strings.TrimSpace(
			assetID,
		)

	aociText =
		strings.TrimSpace(
			aociText,
		)

	generationSource =
		strings.TrimSpace(
			generationSource,
		)

	if assetID == "" {
		return nil,
			fmt.Errorf(
				"漫画格生成结果缺少图片资产",
			)
	}

	if !models.IsValidCWComicVersionSource(
		generationSource,
	) {
		return nil,
			fmt.Errorf(
				"漫画格版本来源不合法",
			)
	}

	tx, err :=
		database.DB.Begin(
			ctx,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"开启漫画格完成事务失败: %w",
				err,
			)
	}

	defer func() {
		_ =
			tx.Rollback(
				ctx,
			)
	}()

	var lockedProjectID string

	err =
		tx.QueryRow(
			ctx,
			`SELECT id
FROM courseware_comic_projects
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3
  AND status IN ($4, $5)
FOR UPDATE`,
			projectID,
			coursewareID,
			userID,
			models.CWComicProjectStatusGenerating,
			models.CWComicProjectStatusFailed,
		).Scan(
			&lockedProjectID,
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
				"锁定漫画项目失败: %w",
				err,
			)
	}

	panel, err :=
		scanCoursewareComicPanel(
			tx.QueryRow(
				ctx,
				`SELECT `+
					coursewareComicPanelSelectColumns+
					` FROM courseware_comic_panels
WHERE id = $1
  AND project_id = $2
FOR UPDATE`,
				panelID,
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
				"锁定漫画格失败: %w",
				err,
			)
	}

	if panel.Status !=
		models.CWComicPanelStatusGenerating {
		return nil,
			ErrCoursewareComicPanelConflict
	}

	var validAsset bool

	err =
		tx.QueryRow(
			ctx,
			`SELECT EXISTS (
    SELECT 1
    FROM courseware_assets
    WHERE id = $1
      AND courseware_id = $2
      AND asset_type = 'image'
)`,
			assetID,
			coursewareID,
		).Scan(
			&validAsset,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"校验漫画图片资产失败: %w",
				err,
			)
	}

	if !validAsset {
		return nil,
			ErrCoursewareComicAssetInvalid
	}

	if aociText == "" {
		aociText =
			strings.TrimSpace(
				panel.AOCIText,
			)
	}

	if aociText == "" {
		return nil,
			fmt.Errorf(
				"漫画格生成结果缺少IAOCI",
			)
	}

	var nextVersionNo int

	err =
		tx.QueryRow(
			ctx,
			`SELECT COALESCE(MAX(version_no), 0) + 1
FROM courseware_comic_panel_versions
WHERE panel_id = $1`,
			panelID,
		).Scan(
			&nextVersionNo,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"计算漫画格版本号失败: %w",
				err,
			)
	}

	if _, err :=
		tx.Exec(
			ctx,
			`INSERT INTO courseware_comic_panel_versions (
panel_id,
version_no,
prompt_snapshot,
aoci_snapshot,
overlay_document_snapshot,
asset_id,
generation_source,
created_by
)
VALUES (
$1, $2, $3, $4,
$5::jsonb, $6, $7, $8
)`,
			panelID,
			nextVersionNo,
			panel.VisualPrompt,
			aociText,
			panel.OverlayDocumentJSON,
			assetID,
			generationSource,
			userID,
		); err != nil {
		return nil,
			fmt.Errorf(
				"保存漫画格历史版本失败: %w",
				err,
			)
	}

	updatedPanel, err :=
		scanCoursewareComicPanel(
			tx.QueryRow(
				ctx,
				`UPDATE courseware_comic_panels
SET current_asset_id = $1,
    aoci_text = $2,
    status = $3,
    version = version + 1,
    last_error = '',
    updated_at = now()
WHERE id = $4
  AND project_id = $5
  AND status = $6
RETURNING `+
					coursewareComicPanelSelectColumns,
				assetID,
				aociText,
				models.CWComicPanelStatusGenerated,
				panelID,
				projectID,
				models.CWComicPanelStatusGenerating,
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
				"绑定漫画格图片资产失败: %w",
				err,
			)
	}

	var totalCount int
	var generatedCount int
	var failedCount int

	err =
		tx.QueryRow(
			ctx,
			`SELECT
    COUNT(*),
    COUNT(*) FILTER (
        WHERE status = $2
    ),
    COUNT(*) FILTER (
        WHERE status = $3
    )
FROM courseware_comic_panels
WHERE project_id = $1`,
			projectID,
			models.CWComicPanelStatusGenerated,
			models.CWComicPanelStatusFailed,
		).Scan(
			&totalCount,
			&generatedCount,
			&failedCount,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"统计漫画格生成结果失败: %w",
				err,
			)
	}

	projectStatus :=
		resolveCoursewareComicAggregateStatus(
			totalCount,
			generatedCount,
			failedCount,
		)

	preserveLastError :=
		projectStatus ==
			models.CWComicProjectStatusFailed

	tag, err :=
		tx.Exec(
			ctx,
			coursewareComicAggregateProjectUpdateSQL,
			projectStatus,
			preserveLastError,
			projectID,
			coursewareID,
			userID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"更新漫画项目聚合状态失败: %w",
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
				"提交漫画格完成事务失败: %w",
				err,
			)
	}

	return updatedPanel, nil
}

// ListCoursewareComicPanelVersions 返回一个漫画格的全部历史版本。
func ListCoursewareComicPanelVersions(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	userID string,
) ([]*models.CoursewareComicPanelVersion, error) {
	rows, err :=
		database.DB.Query(
			ctx,
			`SELECT `+
				coursewareComicPanelVersionSelectColumns+
				` FROM courseware_comic_panel_versions version
WHERE version.panel_id = $1
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_panels panel
      JOIN courseware_comic_projects project
        ON project.id = panel.project_id
      WHERE panel.id = version.panel_id
        AND panel.project_id = $2
        AND project.courseware_id = $3
        AND project.created_by = $4
  )
ORDER BY version.version_no DESC`,
			panelID,
			projectID,
			coursewareID,
			userID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"查询漫画格历史版本失败: %w",
				err,
			)
	}

	defer rows.Close()

	items :=
		make(
			[]*models.CoursewareComicPanelVersion,
			0,
		)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareComicPanelVersion(
				rows,
			)

		if scanErr != nil {
			return nil,
				fmt.Errorf(
					"扫描漫画格历史版本失败: %w",
					scanErr,
				)
		}

		items =
			append(
				items,
				item,
			)
	}

	if err :=
		rows.Err(); err != nil {
		return nil,
			fmt.Errorf(
				"遍历漫画格历史版本失败: %w",
				err,
			)
	}

	return items, nil
}

// GetCoursewareComicPanelVersion 读取指定历史版本。
func GetCoursewareComicPanelVersion(
	ctx context.Context,
	coursewareID string,
	projectID string,
	panelID string,
	versionID string,
	userID string,
) (*models.CoursewareComicPanelVersion, error) {
	item, err :=
		scanCoursewareComicPanelVersion(
			database.DB.QueryRow(
				ctx,
				`SELECT `+
					coursewareComicPanelVersionSelectColumns+
					` FROM courseware_comic_panel_versions version
WHERE version.id = $1
  AND version.panel_id = $2
  AND EXISTS (
      SELECT 1
      FROM courseware_comic_panels panel
      JOIN courseware_comic_projects project
        ON project.id = panel.project_id
      WHERE panel.id = version.panel_id
        AND panel.project_id = $3
        AND project.courseware_id = $4
        AND project.created_by = $5
  )`,
				versionID,
				panelID,
				projectID,
				coursewareID,
				userID,
			),
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return nil,
			ErrCoursewareComicPanelVersionNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"读取漫画格历史版本失败: %w",
				err,
			)
	}

	return item, nil
}
