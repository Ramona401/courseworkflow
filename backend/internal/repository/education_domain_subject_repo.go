package repository

// education_domain_subject_repo.go — 分域课程目录查询
//
// 本文件从education_domain_repo.go拆出，仅负责按用户教育域和教学组织
// 返回有效课程目录，避免用户教育域解析主文件超过600行红线。
//
// 查询规则：
//   - k12、vocational、adult只返回当前域公共课程和当前学校私有课程；
//   - 学校私有目录优先覆盖同一课程的公共目录；
//   - mixed管理上下文返回全部启用课程；
//   - vocational和adult目录为空时绝不回退K12；
//   - k12目录异常时只回退is_system=true的内置学科。
//
// 本次仅做文件拆分，函数签名、SQL、返回值与原实现保持一致。

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// listActiveSystemSubjects 返回启用的K12内置学科。
//
// 仅用于K12课程目录迁移异常时兜底，不包含职业学校新增课程。
func listActiveSystemSubjects(
	ctx context.Context,
) ([]*models.Subject, error) {
	query := `SELECT ` + subjectSelectColumns + `
		FROM subjects
		WHERE is_active = true
		  AND is_system = true
		ORDER BY
			sort_order ASC,
			name ASC`

	rows, err := database.DB.Query(
		ctx,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询K12内置学科失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]*models.Subject, 0)

	for rows.Next() {
		item, err := scanSubject(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"扫描K12内置学科失败: %w",
				err,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历K12内置学科失败: %w",
			err,
		)
	}

	return items, nil
}

// ListActiveSubjectsForEducationContext 按教育域和教学组织返回有效课程目录。
func ListActiveSubjectsForEducationContext(
	ctx context.Context,
	educationDomain string,
	organizationID string,
) ([]*models.Subject, error) {
	domain := models.NormalizeEducationDomain(
		educationDomain,
	)

	// mixed是跨域管理上下文，管理者继续看到全部启用课程定义。
	if domain == models.EducationDomainMixed {
		return ListActiveSubjects(ctx)
	}

	query := `
		WITH candidates AS (
			SELECT
				s.id,
				COALESCE(
					NULLIF(
						sce.display_name,
						''
					),
					s.name
				) AS display_name,
				s.code,
				sce.sort_order AS catalog_sort_order,
				s.is_active,
				s.is_system,
				COALESCE(
					s.note,
					''
				) AS note,
				s.updated_by,
				s.created_at,
				s.updated_at,
				CASE
					WHEN sce.organization_id IS NULL
						THEN 1
					ELSE 0
				END AS source_rank
			FROM subject_catalog_entries sce
			JOIN subjects s
			  ON s.id = sce.subject_id
			WHERE sce.education_domain = $1
			  AND sce.is_active = true
			  AND s.is_active = true
			  AND (
					sce.organization_id IS NULL
					OR (
						NULLIF($2, '') IS NOT NULL
						AND sce.organization_id::text = $2
					)
			  )
		),
		ranked AS (
			SELECT
				*,
				ROW_NUMBER() OVER (
					PARTITION BY id
					ORDER BY
						source_rank ASC,
						catalog_sort_order ASC,
						display_name ASC
				) AS row_rank
			FROM candidates
		)
		SELECT
			id,
			display_name,
			code,
			catalog_sort_order,
			is_active,
			is_system,
			note,
			updated_by,
			created_at,
			updated_at
		FROM ranked
		WHERE row_rank = 1
		ORDER BY
			catalog_sort_order ASC,
			display_name ASC
	`

	rows, err := database.DB.Query(
		ctx,
		query,
		domain,
		organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询教育域课程目录失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]*models.Subject, 0)

	for rows.Next() {
		item := &models.Subject{}

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Code,
			&item.SortOrder,
			&item.IsActive,
			&item.IsSystem,
			&item.Note,
			&item.UpdatedBy,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描教育域课程目录失败: %w",
				err,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历教育域课程目录失败: %w",
			err,
		)
	}

	if len(items) > 0 {
		return items, nil
	}

	// K12在目录表异常或尚未建立时，只回退内置学科。
	// 职业教育和成人教育绝不回退K12。
	if domain == models.EducationDomainK12 {
		return listActiveSystemSubjects(ctx)
	}

	return []*models.Subject{}, nil
}
