package repository

// courseware_comic_reference_repo.go — 知识点漫画参考资源仓储
//
// 本文件负责：
//   - 按项目、课件和创建者三重边界读取参考资源；
//   - 使用项目行锁串行化新增和删除操作；
//   - 在数据库事务内执行最多8项的数量限制；
//   - 只允许draft、planned和failed项目修改参考资源；
//   - 删除参考资源绑定时不删除独立图片资产；
//   - 正式来源去重和图片去重由数据库唯一索引作最终防线。
//
// 查询连接漫画项目表时，所有参考资源字段必须显式使用resource别名。
// 不能复用无表别名的RETURNING字段列表，否则id、courseware_id、
// created_by等同名字段会触发PostgreSQL SQLSTATE 42702。
//
// 校验与错误转换已拆到同包文件，保持本仓储文件低于600行。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// coursewareComicReferenceSelectColumns
// 用于单表INSERT RETURNING，不使用表别名。
const coursewareComicReferenceSelectColumns = `
id,
project_id,
courseware_id,
created_by,
resource_type,
source_id,
asset_id,
title,
file_name,
mime_type,
content_text,
summary_text,
original_length,
summary_length,
sort_order,
created_at,
updated_at`

// coursewareComicReferenceResourceSelectColumns
// 用于与漫画项目表连接的查询。
//
// 所有字段明确来自resource，避免与project.id、project.courseware_id、
// project.created_by、project.created_at和project.updated_at发生歧义。
const coursewareComicReferenceResourceSelectColumns = `
resource.id,
resource.project_id,
resource.courseware_id,
resource.created_by,
resource.resource_type,
resource.source_id,
resource.asset_id,
resource.title,
resource.file_name,
resource.mime_type,
resource.content_text,
resource.summary_text,
resource.original_length,
resource.summary_length,
resource.sort_order,
resource.created_at,
resource.updated_at`

func scanCoursewareComicReference(
	scanner interface {
		Scan(dest ...interface{}) error
	},
) (*models.CoursewareComicReferenceResource, error) {
	item :=
		&models.CoursewareComicReferenceResource{}

	err :=
		scanner.Scan(
			&item.ID,
			&item.ProjectID,
			&item.CoursewareID,
			&item.CreatedBy,
			&item.ResourceType,
			&item.SourceID,
			&item.AssetID,
			&item.Title,
			&item.FileName,
			&item.MimeType,
			&item.ContentText,
			&item.SummaryText,
			&item.OriginalLength,
			&item.SummaryLength,
			&item.SortOrder,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}

	return item, nil
}

// CreateCoursewareComicReferenceResource
// 在项目行锁保护下原子校验项目状态、数量上限并新增参考资源。
func CreateCoursewareComicReferenceResource(
	ctx context.Context,
	item *models.CoursewareComicReferenceResource,
) error {
	normalizeCoursewareComicReferenceRecord(
		item,
	)

	if err :=
		validateCoursewareComicReferenceRecord(
			item,
		); err != nil {
		return err
	}

	tx, err :=
		database.DB.Begin(
			ctx,
		)
	if err != nil {
		return fmt.Errorf(
			"开启漫画参考资源事务失败: %w",
			err,
		)
	}

	defer func() {
		_ =
			tx.Rollback(
				ctx,
			)
	}()

	var projectStatus string

	err =
		tx.QueryRow(
			ctx,
			`SELECT status
FROM courseware_comic_projects
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3
FOR UPDATE`,
			item.ProjectID,
			item.CoursewareID,
			item.CreatedBy,
		).Scan(
			&projectStatus,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrCoursewareComicProjectNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"锁定漫画参考资源所属项目失败: %w",
			err,
		)
	}

	if !models.IsEditableCWComicProjectStatus(
		projectStatus,
	) {
		return ErrCoursewareComicProjectNotEditable
	}

	var resourceCount int

	if err :=
		tx.QueryRow(
			ctx,
			`SELECT count(*)
FROM courseware_comic_reference_resources
WHERE project_id = $1`,
			item.ProjectID,
		).Scan(
			&resourceCount,
		); err != nil {
		return fmt.Errorf(
			"统计漫画参考资源数量失败: %w",
			err,
		)
	}

	if resourceCount >= 8 {
		return ErrCoursewareComicReferenceLimitReached
	}

	created, err :=
		scanCoursewareComicReference(
			tx.QueryRow(
				ctx,
				`INSERT INTO courseware_comic_reference_resources (
project_id,
courseware_id,
created_by,
resource_type,
source_id,
asset_id,
title,
file_name,
mime_type,
content_text,
summary_text,
original_length,
summary_length,
sort_order
)
VALUES (
$1, $2, $3, $4, $5,
$6, $7, $8, $9, $10,
$11, $12, $13, $14
)
RETURNING `+
					coursewareComicReferenceSelectColumns,
				item.ProjectID,
				item.CoursewareID,
				item.CreatedBy,
				item.ResourceType,
				coursewareComicReferenceNullableString(
					item.SourceID,
				),
				coursewareComicReferenceNullableString(
					item.AssetID,
				),
				item.Title,
				item.FileName,
				item.MimeType,
				item.ContentText,
				item.SummaryText,
				item.OriginalLength,
				item.SummaryLength,
				item.SortOrder,
			),
		)
	if err != nil {
		return wrapCoursewareComicReferenceWriteError(
			err,
		)
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return fmt.Errorf(
			"提交漫画参考资源事务失败: %w",
			err,
		)
	}

	*item =
		*created

	return nil
}

// ListCoursewareComicReferenceResourcesByProjectForUser
// 按项目、课件和创建者三重边界读取完整内部记录。
func ListCoursewareComicReferenceResourcesByProjectForUser(
	ctx context.Context,
	coursewareID string,
	projectID string,
	userID string,
) ([]*models.CoursewareComicReferenceResource, error) {
	rows, err :=
		database.DB.Query(
			ctx,
			`SELECT `+
				coursewareComicReferenceResourceSelectColumns+
				`
FROM courseware_comic_reference_resources resource
INNER JOIN courseware_comic_projects project
        ON project.id = resource.project_id
WHERE resource.project_id = $1
  AND resource.courseware_id = $2
  AND resource.created_by = $3
  AND project.courseware_id = $2
  AND project.created_by = $3
ORDER BY
    resource.sort_order ASC,
    resource.created_at ASC,
    resource.id ASC`,
			strings.TrimSpace(
				projectID,
			),
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
				"查询知识点漫画参考资源失败: %w",
				err,
			)
	}
	defer rows.Close()

	items :=
		make(
			[]*models.CoursewareComicReferenceResource,
			0,
		)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareComicReference(
				rows,
			)
		if scanErr != nil {
			return nil,
				fmt.Errorf(
					"扫描知识点漫画参考资源失败: %w",
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
				"遍历知识点漫画参考资源失败: %w",
				err,
			)
	}

	return items, nil
}

// DeleteCoursewareComicReferenceResourceForUser
// 删除参考资源绑定，但不删除独立的courseware_assets图片记录。
func DeleteCoursewareComicReferenceResourceForUser(
	ctx context.Context,
	coursewareID string,
	projectID string,
	referenceID string,
	userID string,
) error {
	tx, err :=
		database.DB.Begin(
			ctx,
		)
	if err != nil {
		return fmt.Errorf(
			"开启删除漫画参考资源事务失败: %w",
			err,
		)
	}

	defer func() {
		_ =
			tx.Rollback(
				ctx,
			)
	}()

	var projectStatus string

	err =
		tx.QueryRow(
			ctx,
			`SELECT status
FROM courseware_comic_projects
WHERE id = $1
  AND courseware_id = $2
  AND created_by = $3
FOR UPDATE`,
			strings.TrimSpace(
				projectID,
			),
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				userID,
			),
		).Scan(
			&projectStatus,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrCoursewareComicProjectNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"锁定漫画参考资源所属项目失败: %w",
			err,
		)
	}

	if !models.IsEditableCWComicProjectStatus(
		projectStatus,
	) {
		return ErrCoursewareComicProjectNotEditable
	}

	var deletedID string

	err =
		tx.QueryRow(
			ctx,
			`DELETE FROM courseware_comic_reference_resources
WHERE id = $1
  AND project_id = $2
  AND courseware_id = $3
  AND created_by = $4
RETURNING id`,
			strings.TrimSpace(
				referenceID,
			),
			strings.TrimSpace(
				projectID,
			),
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				userID,
			),
		).Scan(
			&deletedID,
		)

	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return ErrCoursewareComicReferenceNotFound
	}

	if err != nil {
		return fmt.Errorf(
			"删除知识点漫画参考资源失败: %w",
			err,
		)
	}

	if err :=
		tx.Commit(
			ctx,
		); err != nil {
		return fmt.Errorf(
			"提交删除漫画参考资源事务失败: %w",
			err,
		)
	}

	return nil
}
