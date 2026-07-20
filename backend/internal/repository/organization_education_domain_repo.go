package repository

// organization_education_domain_repo.go — 组织教育域只读数据访问
//
// 职责：
//   1. 列出所有组织创建时确定的教育域；
//   2. 查询单个组织教育域；
//   3. 为管理界面和旧PUT接口的存在性检查提供只读数据。
//
// 本文件不再包含：
//   - 教学资源统计；
//   - organizations.education_domain更新；
//   - subjects或subject_catalog_entries写入；
//   - 换域后的课程目录建立、重建或重新启用。
//
// 学校教育域不可变由数据库触发器提供最终保护。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var ErrOrganizationEducationDomainNotFound = errors.New("组织不存在")

// ListOrganizationEducationDomains 列出全部组织教育域。
func ListOrganizationEducationDomains(
	ctx context.Context,
) ([]*models.OrganizationEducationDomainItem, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT
			o.id::text,
			o.name,
			o.type,
			o.parent_id::text,
			COALESCE(parent.name, ''),
			COALESCE(o.education_domain, 'k12'),
			o.status,
			(
				SELECT COUNT(*)
				FROM teaching_groups tg
				WHERE tg.school_id = o.id
				  AND tg.status = 'active'
			) AS group_count,
			(
				SELECT COUNT(DISTINCT sm.user_id)
				FROM school_members sm
				WHERE sm.school_id = o.id
			) AS member_count
		FROM organizations o
		LEFT JOIN organizations parent
		  ON parent.id = o.parent_id
		ORDER BY
			CASE o.type
				WHEN 'region' THEN 1
				WHEN 'school' THEN 2
				ELSE 3
			END,
			parent.name NULLS FIRST,
			o.name,
			o.id
	`)
	if err != nil {
		return nil, fmt.Errorf("查询组织教育域列表失败: %w", err)
	}
	defer rows.Close()

	items := make([]*models.OrganizationEducationDomainItem, 0)

	for rows.Next() {
		item := &models.OrganizationEducationDomainItem{}

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Type,
			&item.ParentID,
			&item.ParentName,
			&item.EducationDomain,
			&item.Status,
			&item.GroupCount,
			&item.MemberCount,
		); err != nil {
			return nil, fmt.Errorf("扫描组织教育域列表失败: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历组织教育域列表失败: %w", err)
	}

	return items, nil
}

// GetOrganizationEducationDomainByID 查询单个组织教育域。
//
// 本方法只读取，不锁行，也不提供后续更新能力。
// 旧PUT接口仅使用它确认组织是否真实存在。
func GetOrganizationEducationDomainByID(
	ctx context.Context,
	organizationID string,
) (*models.OrganizationEducationDomainItem, error) {
	item := &models.OrganizationEducationDomainItem{}

	err := database.DB.QueryRow(ctx, `
		SELECT
			o.id::text,
			o.name,
			o.type,
			o.parent_id::text,
			COALESCE(parent.name, ''),
			COALESCE(o.education_domain, 'k12'),
			o.status,
			(
				SELECT COUNT(*)
				FROM teaching_groups tg
				WHERE tg.school_id = o.id
				  AND tg.status = 'active'
			) AS group_count,
			(
				SELECT COUNT(DISTINCT sm.user_id)
				FROM school_members sm
				WHERE sm.school_id = o.id
			) AS member_count
		FROM organizations o
		LEFT JOIN organizations parent
		  ON parent.id = o.parent_id
		WHERE o.id = $1
	`, organizationID).Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&item.ParentID,
		&item.ParentName,
		&item.EducationDomain,
		&item.Status,
		&item.GroupCount,
		&item.MemberCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrganizationEducationDomainNotFound
		}
		return nil, fmt.Errorf("查询组织教育域失败: %w", err)
	}

	return item, nil
}
