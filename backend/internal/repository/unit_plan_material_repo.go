package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrUnitPlanMaterialNotFound 大单元参考资料不存在。
var ErrUnitPlanMaterialNotFound = errors.New("大单元参考资料不存在")

const unitPlanMaterialSelectColumns = `
    id, unit_plan_id, material_type, file_name,
    content_text, summary_text,
    original_length, summary_length,
    uploaded_by, status, created_at, updated_at
`

func scanUnitPlanMaterial(row pgx.Row) (*models.UnitPlanMaterial, error) {
	material := &models.UnitPlanMaterial{}

	err := row.Scan(
		&material.ID,
		&material.UnitPlanID,
		&material.MaterialType,
		&material.FileName,
		&material.ContentText,
		&material.SummaryText,
		&material.OriginalLength,
		&material.SummaryLength,
		&material.UploadedBy,
		&material.Status,
		&material.CreatedAt,
		&material.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnitPlanMaterialNotFound
		}
		return nil, fmt.Errorf("扫描大单元参考资料失败: %w", err)
	}

	return material, nil
}

// CreateUnitPlanMaterial 新增一份参考资料。
func CreateUnitPlanMaterial(
	ctx context.Context,
	material *models.UnitPlanMaterial,
) error {
	err := database.DB.QueryRow(ctx, `
        INSERT INTO unit_plan_materials (
            unit_plan_id,
            material_type,
            file_name,
            content_text,
            summary_text,
            original_length,
            summary_length,
            uploaded_by,
            status
        )
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'active')
        RETURNING id, status, created_at, updated_at
    `,
		material.UnitPlanID,
		material.MaterialType,
		material.FileName,
		material.ContentText,
		material.SummaryText,
		material.OriginalLength,
		material.SummaryLength,
		material.UploadedBy,
	).Scan(
		&material.ID,
		&material.Status,
		&material.CreatedAt,
		&material.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建大单元参考资料失败: %w", err)
	}

	return nil
}

// GetUnitPlanMaterialByID 查询单条资料。
func GetUnitPlanMaterialByID(
	ctx context.Context,
	id string,
) (*models.UnitPlanMaterial, error) {
	sql := `
        SELECT ` + unitPlanMaterialSelectColumns + `
        FROM unit_plan_materials
        WHERE id = $1
    `

	return scanUnitPlanMaterial(database.DB.QueryRow(ctx, sql, id))
}

// ListUnitPlanMaterials 返回资料轻量列表，不返回正文。
func ListUnitPlanMaterials(
	ctx context.Context,
	unitPlanID string,
) ([]*models.UnitPlanMaterialListItem, error) {
	rows, err := database.DB.Query(ctx, `
        SELECT
            id,
            unit_plan_id,
            material_type,
            file_name,
            original_length,
            summary_length,
            length(trim(summary_text)) > 0 AS has_summary,
            uploaded_by,
            status,
            created_at,
            updated_at
        FROM unit_plan_materials
        WHERE unit_plan_id = $1
          AND status = 'active'
        ORDER BY created_at DESC
    `, unitPlanID)
	if err != nil {
		return nil, fmt.Errorf("查询大单元参考资料列表失败: %w", err)
	}
	defer rows.Close()

	items := make([]*models.UnitPlanMaterialListItem, 0)

	for rows.Next() {
		item := &models.UnitPlanMaterialListItem{}

		if err := rows.Scan(
			&item.ID,
			&item.UnitPlanID,
			&item.MaterialType,
			&item.FileName,
			&item.OriginalLength,
			&item.SummaryLength,
			&item.HasSummary,
			&item.UploadedBy,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描大单元参考资料列表失败: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历大单元参考资料列表失败: %w", err)
	}

	return items, nil
}

// ListActiveUnitPlanMaterialsForContext 返回AI上下文装配所需的有效资料全文。
func ListActiveUnitPlanMaterialsForContext(
	ctx context.Context,
	unitPlanID string,
) ([]*models.UnitPlanMaterial, error) {
	rows, err := database.DB.Query(ctx, `
        SELECT `+unitPlanMaterialSelectColumns+`
        FROM unit_plan_materials
        WHERE unit_plan_id = $1
          AND status = 'active'
        ORDER BY created_at ASC
    `, unitPlanID)
	if err != nil {
		return nil, fmt.Errorf("查询大单元参考资料正文失败: %w", err)
	}
	defer rows.Close()

	materials := make([]*models.UnitPlanMaterial, 0)

	for rows.Next() {
		material := &models.UnitPlanMaterial{}

		if err := rows.Scan(
			&material.ID,
			&material.UnitPlanID,
			&material.MaterialType,
			&material.FileName,
			&material.ContentText,
			&material.SummaryText,
			&material.OriginalLength,
			&material.SummaryLength,
			&material.UploadedBy,
			&material.Status,
			&material.CreatedAt,
			&material.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描大单元参考资料正文失败: %w", err)
		}

		materials = append(materials, material)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历大单元参考资料正文失败: %w", err)
	}

	return materials, nil
}

// ArchiveUnitPlanMaterial 软删除资料。
func ArchiveUnitPlanMaterial(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx, `
        UPDATE unit_plan_materials
        SET status = 'archived',
            updated_at = now()
        WHERE id = $1
          AND status = 'active'
    `, id)
	if err != nil {
		return fmt.Errorf("删除大单元参考资料失败: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUnitPlanMaterialNotFound
	}

	return nil
}
