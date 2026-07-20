package repository

// organization_crud_repo.go
//
// 本文件承载组织CRUD和组织树查询：
//   - 创建、详情、列表、更新、删除；
//   - 按管理员反查学校；
//   - 区域下学校查询；
//   - 有效学校批量校验；
//   - 区域树递归学校ID查询。
//
// 教育域规则：
//   - 创建时必须由Service确定最终教育域；
//   - INSERT显式写入education_domain；
//   - 创建响应以数据库RETURNING值为准；
//   - 普通UpdateOrganization不修改education_domain。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// CreateOrganization 创建组织并返回数据库最终写入的教育域。
func CreateOrganization(
	ctx context.Context,
	org *models.Organization,
) error {
	query := `
		INSERT INTO organizations (
			name,
			type,
			parent_id,
			admin_user_id,
			settings,
			status,
			education_domain
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING
			id::text,
			education_domain,
			created_at,
			updated_at
	`

	settings := org.Settings
	if settings == "" {
		settings = "{}"
	}

	err := database.DB.QueryRow(
		ctx,
		query,
		org.Name,
		org.Type,
		org.ParentID,
		org.AdminUserID,
		settings,
		"active",
		org.EducationDomain,
	).Scan(
		&org.ID,
		&org.EducationDomain,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("创建组织失败: %w", err)
	}

	org.Settings = settings
	org.Status = "active"

	return nil
}

// GetOrganizationByID 查询单个组织。
func GetOrganizationByID(
	ctx context.Context,
	id string,
) (*models.Organization, error) {
	org := &models.Organization{}

	err := database.DB.QueryRow(ctx, `
		SELECT
			id::text,
			name,
			type,
			COALESCE(education_domain, 'k12'),
			parent_id::text,
			admin_user_id::text,
			settings,
			COALESCE(logo_url, ''),
			status,
			created_at,
			updated_at
		FROM organizations
		WHERE id = $1
	`, id).Scan(
		&org.ID,
		&org.Name,
		&org.Type,
		&org.EducationDomain,
		&org.ParentID,
		&org.AdminUserID,
		&org.Settings,
		&org.LogoURL,
		&org.Status,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("查询组织失败: %w", err)
	}

	return org, nil
}

// GetSchoolByAdminUserID 根据学校管理员用户ID获取其管理的学校。
//
// 查找顺序：
//  1. organizations.admin_user_id主管理员字段；
//  2. organization_admins多管理员任命；
//  3. 两处均无时返回ErrOrgNotFound。
func GetSchoolByAdminUserID(
	ctx context.Context,
	adminUserID string,
) (*models.Organization, error) {
	if adminUserID == "" {
		return nil, ErrOrgNotFound
	}

	org := &models.Organization{}

	err := database.DB.QueryRow(ctx, `
		SELECT
			id::text,
			name,
			type,
			COALESCE(education_domain, 'k12'),
			parent_id::text,
			admin_user_id::text,
			settings,
			COALESCE(logo_url, ''),
			status,
			created_at,
			updated_at
		FROM organizations
		WHERE admin_user_id = $1
		  AND type = 'school'
		ORDER BY created_at ASC, id ASC
		LIMIT 1
	`, adminUserID).Scan(
		&org.ID,
		&org.Name,
		&org.Type,
		&org.EducationDomain,
		&org.ParentID,
		&org.AdminUserID,
		&org.Settings,
		&org.LogoURL,
		&org.Status,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err == nil {
		return org, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf(
			"查询学校管理员所属学校失败: %w",
			err,
		)
	}

	fallback := &models.Organization{}

	err = database.DB.QueryRow(ctx, `
		SELECT
			o.id::text,
			o.name,
			o.type,
			COALESCE(o.education_domain, 'k12'),
			o.parent_id::text,
			o.admin_user_id::text,
			o.settings,
			COALESCE(o.logo_url, ''),
			o.status,
			o.created_at,
			o.updated_at
		FROM organization_admins oa
		JOIN organizations o
		  ON o.id = oa.org_id
		WHERE oa.user_id = $1
		  AND oa.role_type = 'school_admin'
		  AND o.type = 'school'
		ORDER BY oa.created_at ASC, o.id ASC
		LIMIT 1
	`, adminUserID).Scan(
		&fallback.ID,
		&fallback.Name,
		&fallback.Type,
		&fallback.EducationDomain,
		&fallback.ParentID,
		&fallback.AdminUserID,
		&fallback.Settings,
		&fallback.LogoURL,
		&fallback.Status,
		&fallback.CreatedAt,
		&fallback.UpdatedAt,
	)
	if err == nil {
		return fallback, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrgNotFound
	}

	return nil, fmt.Errorf(
		"查询学校管理员所属学校(多管理员兜底)失败: %w",
		err,
	)
}

// ListOrganizations 查询组织列表。
func ListOrganizations(
	ctx context.Context,
	orgType string,
	parentID string,
) ([]*models.OrganizationListItem, error) {
	query := `
		SELECT
			o.id::text,
			o.name,
			o.type,
			COALESCE(o.education_domain, 'k12'),
			o.parent_id::text,
			o.admin_user_id::text,
			COALESCE(o.logo_url, ''),
			o.status,
			o.created_at,
			COALESCE(parent.name, '') AS parent_name,
			COALESCE(admin_user.display_name, '') AS admin_user_name,
			(
				SELECT COUNT(*)
				FROM teaching_groups tg
				WHERE tg.school_id = o.id
				  AND tg.status = 'active'
			) AS group_count,
			(
				SELECT COUNT(DISTINCT tgm.user_id)
				FROM teaching_group_members tgm
				JOIN teaching_groups tg2
				  ON tg2.id = tgm.group_id
				WHERE tg2.school_id = o.id
			) AS member_count
		FROM organizations o
		LEFT JOIN organizations parent
		  ON parent.id = o.parent_id
		LEFT JOIN users admin_user
		  ON admin_user.id = o.admin_user_id
		WHERE 1 = 1
	`

	args := make([]interface{}, 0, 2)
	argIndex := 1

	if orgType != "" {
		query += fmt.Sprintf(
			" AND o.type = $%d",
			argIndex,
		)
		args = append(args, orgType)
		argIndex++
	}

	if parentID != "" {
		query += fmt.Sprintf(
			" AND o.parent_id = $%d",
			argIndex,
		)
		args = append(args, parentID)
	}

	query += " ORDER BY o.type ASC, o.name ASC, o.id ASC"

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询组织列表失败: %w", err)
	}
	defer rows.Close()

	items := make([]*models.OrganizationListItem, 0)

	for rows.Next() {
		item := &models.OrganizationListItem{}

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Type,
			&item.EducationDomain,
			&item.ParentID,
			&item.AdminUserID,
			&item.LogoURL,
			&item.Status,
			&item.CreatedAt,
			&item.ParentName,
			&item.AdminUserName,
			&item.GroupCount,
			&item.MemberCount,
		); err != nil {
			return nil, fmt.Errorf("扫描组织行失败: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历组织列表失败: %w", err)
	}

	return items, nil
}

// UpdateOrganization 更新组织的普通可编辑字段。
//
// 本方法刻意不更新education_domain。
func UpdateOrganization(
	ctx context.Context,
	id string,
	req *models.UpdateOrganizationRequest,
) error {
	settings := req.Settings
	if settings == "" {
		settings = "{}"
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	result, err := database.DB.Exec(ctx, `
		UPDATE organizations
		SET
			name = $1,
			admin_user_id = $2,
			settings = $3,
			status = $4,
			updated_at = $5
		WHERE id = $6
	`,
		req.Name,
		req.AdminUserID,
		settings,
		status,
		time.Now(),
		id,
	)
	if err != nil {
		return fmt.Errorf("更新组织失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}

	return nil
}

// DeleteOrganization 物理删除组织。
func DeleteOrganization(
	ctx context.Context,
	id string,
) error {
	result, err := database.DB.Exec(
		ctx,
		`DELETE FROM organizations WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("删除组织失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}

	return nil
}

// CheckOrgNameExists 检查同类型组织名称是否已经存在。
func CheckOrgNameExists(
	ctx context.Context,
	name string,
	orgType string,
	excludeID string,
) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM organizations
		WHERE name = $1
		  AND type = $2
	`
	args := []interface{}{
		name,
		orgType,
	}

	if excludeID != "" {
		query += " AND id != $3"
		args = append(args, excludeID)
	}

	var count int
	if err := database.DB.QueryRow(
		ctx,
		query,
		args...,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("检查组织名称失败: %w", err)
	}

	return count > 0, nil
}

// GetSchoolsByRegion 查询区域直接下属的启用学校。
func GetSchoolsByRegion(
	ctx context.Context,
	regionID string,
) ([]*models.Organization, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT
			id::text,
			name,
			type,
			COALESCE(education_domain, 'k12'),
			parent_id::text,
			admin_user_id::text,
			settings,
			COALESCE(logo_url, ''),
			status,
			created_at,
			updated_at
		FROM organizations
		WHERE parent_id = $1
		  AND type = 'school'
		  AND status = 'active'
		ORDER BY name ASC, id ASC
	`, regionID)
	if err != nil {
		return nil, fmt.Errorf("查询学校列表失败: %w", err)
	}
	defer rows.Close()

	organizations := make([]*models.Organization, 0)

	for rows.Next() {
		org := &models.Organization{}

		if err := rows.Scan(
			&org.ID,
			&org.Name,
			&org.Type,
			&org.EducationDomain,
			&org.ParentID,
			&org.AdminUserID,
			&org.Settings,
			&org.LogoURL,
			&org.Status,
			&org.CreatedAt,
			&org.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描学校行失败: %w", err)
		}

		organizations = append(organizations, org)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历学校列表失败: %w", err)
	}

	return organizations, nil
}

// ListExistingActiveSchoolIDs 批量校验学校ID。
//
// 只返回真实存在、类型为school且状态为active的学校。
func ListExistingActiveSchoolIDs(
	ctx context.Context,
	schoolIDs []string,
) (map[string]bool, error) {
	result := make(map[string]bool)

	if len(schoolIDs) == 0 {
		return result, nil
	}

	rows, err := database.DB.Query(ctx, `
		SELECT id::text
		FROM organizations
		WHERE id::text = ANY($1)
		  AND type = 'school'
		  AND status = 'active'
	`, schoolIDs)
	if err != nil {
		return nil, fmt.Errorf("批量校验学校ID失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string

		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("扫描有效学校ID失败: %w", err)
		}

		result[id] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历有效学校ID结果失败: %w", err)
	}

	return result, nil
}

// ListDescendantSchoolIDs 递归查询区域树下全部启用学校ID。
//
// regionID为空时返回非nil空切片。
func ListDescendantSchoolIDs(
	ctx context.Context,
	regionID string,
) ([]string, error) {
	if regionID == "" {
		return []string{}, nil
	}

	rows, err := database.DB.Query(ctx, `
		WITH RECURSIVE org_tree AS (
			SELECT
				id,
				type,
				parent_id,
				status,
				0 AS depth
			FROM organizations
			WHERE id = $1

			UNION ALL

			SELECT
				child.id,
				child.type,
				child.parent_id,
				child.status,
				parent.depth + 1
			FROM organizations child
			JOIN org_tree parent
			  ON child.parent_id = parent.id
		)
		SELECT id::text
		FROM org_tree
		WHERE type = 'school'
		  AND status = 'active'
		  AND depth > 0
		ORDER BY id
	`, regionID)
	if err != nil {
		return nil, fmt.Errorf(
			"递归查询区域树下学校失败: %w",
			err,
		)
	}
	defer rows.Close()

	ids := make([]string, 0)

	for rows.Next() {
		var id string

		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf(
				"扫描区域树学校ID失败: %w",
				err,
			)
		}

		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历区域树学校结果失败: %w",
			err,
		)
	}

	return ids, nil
}
