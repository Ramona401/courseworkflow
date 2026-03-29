package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrOrgNotFound       = errors.New("组织不存在")
	ErrOrgNameExists     = errors.New("同类型下组织名称已存在")
	ErrGroupNotFound     = errors.New("教研组不存在")
	ErrGroupNameExists   = errors.New("该学校下教研组名称已存在")
	ErrMemberExists      = errors.New("该用户已是教研组成员")
	ErrMemberNotFound    = errors.New("教研组成员不存在")
)

// ==================== 组织 CRUD ====================

// CreateOrganization 创建组织（区域/学校）
func CreateOrganization(ctx context.Context, org *models.Organization) error {
	query := `
		INSERT INTO organizations (name, type, parent_id, admin_user_id, settings, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	settings := org.Settings
	if settings == "" {
		settings = "{}"
	}
	err := database.DB.QueryRow(ctx, query,
		org.Name, org.Type, org.ParentID, org.AdminUserID, settings, "active",
	).Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建组织失败: %w", err)
	}
	return nil
}

// GetOrganizationByID 根据ID查询组织
func GetOrganizationByID(ctx context.Context, id string) (*models.Organization, error) {
	org := &models.Organization{}
	query := `
		SELECT id, name, type, parent_id, admin_user_id, settings, status, created_at, updated_at
		FROM organizations WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Type, &org.ParentID, &org.AdminUserID,
		&org.Settings, &org.Status, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("查询组织失败: %w", err)
	}
	return org, nil
}

// ListOrganizations 获取组织列表（支持按类型和父级筛选）
func ListOrganizations(ctx context.Context, orgType string, parentID string) ([]*models.OrganizationListItem, error) {
	// 动态构建查询条件
	query := `
		SELECT o.id, o.name, o.type, o.parent_id, o.admin_user_id, o.status, o.created_at,
		       COALESCE(p.name, '') AS parent_name,
		       COALESCE(u.display_name, '') AS admin_user_name,
		       (SELECT COUNT(*) FROM teaching_groups tg WHERE tg.school_id = o.id AND tg.status = 'active') AS group_count,
		       (SELECT COUNT(DISTINCT tgm.user_id) FROM teaching_group_members tgm
		        JOIN teaching_groups tg2 ON tg2.id = tgm.group_id WHERE tg2.school_id = o.id) AS member_count
		FROM organizations o
		LEFT JOIN organizations p ON p.id = o.parent_id
		LEFT JOIN users u ON u.id = o.admin_user_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if orgType != "" {
		query += fmt.Sprintf(" AND o.type = $%d", argIdx)
		args = append(args, orgType)
		argIdx++
	}
	if parentID != "" {
		query += fmt.Sprintf(" AND o.parent_id = $%d", argIdx)
		args = append(args, parentID)
		argIdx++
	}
	query += " ORDER BY o.type ASC, o.name ASC"

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询组织列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.OrganizationListItem
	for rows.Next() {
		item := &models.OrganizationListItem{}
		err := rows.Scan(
			&item.ID, &item.Name, &item.Type, &item.ParentID, &item.AdminUserID,
			&item.Status, &item.CreatedAt,
			&item.ParentName, &item.AdminUserName,
			&item.GroupCount, &item.MemberCount,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描组织行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// UpdateOrganization 更新组织信息
func UpdateOrganization(ctx context.Context, id string, req *models.UpdateOrganizationRequest) error {
	query := `
		UPDATE organizations
		SET name = $1, admin_user_id = $2, settings = $3, status = $4, updated_at = $5
		WHERE id = $6
	`
	settings := req.Settings
	if settings == "" {
		settings = "{}"
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	now := time.Now()
	result, err := database.DB.Exec(ctx, query,
		req.Name, req.AdminUserID, settings, status, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新组织失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}
	return nil
}

// DeleteOrganization 删除组织（物理删除，需先确认无下属）
func DeleteOrganization(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除组织失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}
	return nil
}

// CheckOrgNameExists 检查同类型下组织名称是否已存在
func CheckOrgNameExists(ctx context.Context, name string, orgType string, excludeID string) (bool, error) {
	query := `SELECT COUNT(*) FROM organizations WHERE name = $1 AND type = $2`
	args := []interface{}{name, orgType}
	if excludeID != "" {
		query += " AND id != $3"
		args = append(args, excludeID)
	}
	var count int
	err := database.DB.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查组织名称失败: %w", err)
	}
	return count > 0, nil
}

// GetSchoolsByRegion 获取某区域下所有学校
func GetSchoolsByRegion(ctx context.Context, regionID string) ([]*models.Organization, error) {
	query := `
		SELECT id, name, type, parent_id, admin_user_id, settings, status, created_at, updated_at
		FROM organizations WHERE parent_id = $1 AND type = 'school' AND status = 'active'
		ORDER BY name ASC
	`
	rows, err := database.DB.Query(ctx, query, regionID)
	if err != nil {
		return nil, fmt.Errorf("查询学校列表失败: %w", err)
	}
	defer rows.Close()

	var orgs []*models.Organization
	for rows.Next() {
		org := &models.Organization{}
		err := rows.Scan(
			&org.ID, &org.Name, &org.Type, &org.ParentID, &org.AdminUserID,
			&org.Settings, &org.Status, &org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描学校行失败: %w", err)
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}

// ==================== 教研组 CRUD ====================

// CreateTeachingGroup 创建教研组
func CreateTeachingGroup(ctx context.Context, tg *models.TeachingGroup) error {
	query := `
		INSERT INTO teaching_groups (name, school_id, subject, grade_range, lead_user_id, description, settings, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	settings := tg.Settings
	if settings == "" {
		settings = "{}"
	}
	err := database.DB.QueryRow(ctx, query,
		tg.Name, tg.SchoolID, tg.Subject, tg.GradeRange,
		tg.LeadUserID, tg.Description, settings, "active",
	).Scan(&tg.ID, &tg.CreatedAt, &tg.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建教研组失败: %w", err)
	}
	return nil
}

// GetTeachingGroupByID 根据ID查询教研组
func GetTeachingGroupByID(ctx context.Context, id string) (*models.TeachingGroup, error) {
	tg := &models.TeachingGroup{}
	query := `
		SELECT id, name, school_id, subject, grade_range, lead_user_id,
		       description, settings, status, created_at, updated_at
		FROM teaching_groups WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&tg.ID, &tg.Name, &tg.SchoolID, &tg.Subject, &tg.GradeRange,
		&tg.LeadUserID, &tg.Description, &tg.Settings, &tg.Status,
		&tg.CreatedAt, &tg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("查询教研组失败: %w", err)
	}
	return tg, nil
}

// ListTeachingGroups 获取教研组列表（支持按学校筛选）
func ListTeachingGroups(ctx context.Context, schoolID string) ([]*models.TeachingGroupListItem, error) {
	query := `
		SELECT tg.id, tg.name, tg.school_id, tg.subject, tg.grade_range,
		       tg.lead_user_id, tg.status, tg.created_at,
		       COALESCE(o.name, '') AS school_name,
		       COALESCE(u.display_name, '') AS lead_user_name,
		       (SELECT COUNT(*) FROM teaching_group_members tgm WHERE tgm.group_id = tg.id) AS member_count
		FROM teaching_groups tg
		LEFT JOIN organizations o ON o.id = tg.school_id
		LEFT JOIN users u ON u.id = tg.lead_user_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if schoolID != "" {
		query += fmt.Sprintf(" AND tg.school_id = $%d", argIdx)
		args = append(args, schoolID)
		argIdx++
	}
	query += " ORDER BY tg.name ASC"

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询教研组列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.TeachingGroupListItem
	for rows.Next() {
		item := &models.TeachingGroupListItem{}
		err := rows.Scan(
			&item.ID, &item.Name, &item.SchoolID, &item.Subject, &item.GradeRange,
			&item.LeadUserID, &item.Status, &item.CreatedAt,
			&item.SchoolName, &item.LeadUserName, &item.MemberCount,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描教研组行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// UpdateTeachingGroup 更新教研组信息
func UpdateTeachingGroup(ctx context.Context, id string, req *models.UpdateTeachingGroupRequest) error {
	query := `
		UPDATE teaching_groups
		SET name = $1, subject = $2, grade_range = $3, lead_user_id = $4,
		    description = $5, settings = $6, status = $7, updated_at = $8
		WHERE id = $9
	`
	settings := req.Settings
	if settings == "" {
		settings = "{}"
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	now := time.Now()
	result, err := database.DB.Exec(ctx, query,
		req.Name, req.Subject, req.GradeRange, req.LeadUserID,
		req.Description, settings, status, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新教研组失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// DeleteTeachingGroup 删除教研组（级联删除成员）
func DeleteTeachingGroup(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx, `DELETE FROM teaching_groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除教研组失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// CheckGroupNameExists 检查同一学校下教研组名称是否已存在
func CheckGroupNameExists(ctx context.Context, schoolID string, name string, excludeID string) (bool, error) {
	query := `SELECT COUNT(*) FROM teaching_groups WHERE school_id = $1 AND name = $2`
	args := []interface{}{schoolID, name}
	if excludeID != "" {
		query += " AND id != $3"
		args = append(args, excludeID)
	}
	var count int
	err := database.DB.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查教研组名称失败: %w", err)
	}
	return count > 0, nil
}

// ==================== 教研组成员 CRUD ====================

// AddGroupMember 添加教研组成员
func AddGroupMember(ctx context.Context, member *models.TeachingGroupMember) error {
	query := `
		INSERT INTO teaching_group_members (group_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING id, joined_at
	`
	role := member.Role
	if role == "" {
		role = "member"
	}
	err := database.DB.QueryRow(ctx, query,
		member.GroupID, member.UserID, role,
	).Scan(&member.ID, &member.JoinedAt)
	if err != nil {
		return fmt.Errorf("添加教研组成员失败: %w", err)
	}
	return nil
}

// RemoveGroupMember 移除教研组成员
func RemoveGroupMember(ctx context.Context, groupID string, userID string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM teaching_group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("移除教研组成员失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// ListGroupMembers 获取教研组成员列表
func ListGroupMembers(ctx context.Context, groupID string) ([]*models.GroupMemberItem, error) {
	query := `
		SELECT tgm.id, tgm.user_id, u.username, u.display_name, tgm.role, tgm.joined_at
		FROM teaching_group_members tgm
		JOIN users u ON u.id = tgm.user_id
		WHERE tgm.group_id = $1
		ORDER BY tgm.joined_at ASC
	`
	rows, err := database.DB.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("查询教研组成员失败: %w", err)
	}
	defer rows.Close()

	var items []*models.GroupMemberItem
	for rows.Next() {
		item := &models.GroupMemberItem{}
		err := rows.Scan(
			&item.ID, &item.UserID, &item.Username, &item.DisplayName,
			&item.Role, &item.JoinedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描成员行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// UpdateGroupMemberRole 更新成员角色
func UpdateGroupMemberRole(ctx context.Context, groupID string, userID string, role string) error {
	result, err := database.DB.Exec(ctx,
		`UPDATE teaching_group_members SET role = $1 WHERE group_id = $2 AND user_id = $3`,
		role, groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("更新成员角色失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// CheckMemberExists 检查用户是否已是教研组成员
func CheckMemberExists(ctx context.Context, groupID string, userID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM teaching_group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查成员存在性失败: %w", err)
	}
	return count > 0, nil
}

// GetUserTeachingGroups 获取用户所属的所有教研组（用于判断教案权限）
func GetUserTeachingGroups(ctx context.Context, userID string) ([]*models.TeachingGroupListItem, error) {
	query := `
		SELECT tg.id, tg.name, tg.school_id, tg.subject, tg.grade_range,
		       tg.lead_user_id, tg.status, tg.created_at,
		       COALESCE(o.name, '') AS school_name,
		       COALESCE(u.display_name, '') AS lead_user_name,
		       (SELECT COUNT(*) FROM teaching_group_members tgm2 WHERE tgm2.group_id = tg.id) AS member_count
		FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		LEFT JOIN organizations o ON o.id = tg.school_id
		LEFT JOIN users u ON u.id = tg.lead_user_id
		WHERE tgm.user_id = $1 AND tg.status = 'active'
		ORDER BY tg.name ASC
	`
	rows, err := database.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户教研组失败: %w", err)
	}
	defer rows.Close()

	var items []*models.TeachingGroupListItem
	for rows.Next() {
		item := &models.TeachingGroupListItem{}
		err := rows.Scan(
			&item.ID, &item.Name, &item.SchoolID, &item.Subject, &item.GradeRange,
			&item.LeadUserID, &item.Status, &item.CreatedAt,
			&item.SchoolName, &item.LeadUserName, &item.MemberCount,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描教研组行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// IsGroupLead 检查用户是否是某教研组的组长
func IsGroupLead(ctx context.Context, groupID string, userID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM teaching_groups WHERE id = $1 AND lead_user_id = $2`,
		groupID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查组长权限失败: %w", err)
	}
	return count > 0, nil
}

// IsGroupLeadOrBackbone 检查用户是否是组长或骨干（有审核权限）
func IsGroupLeadOrBackbone(ctx context.Context, groupID string, userID string) (bool, error) {
	// 先检查是否是组长
	isLead, err := IsGroupLead(ctx, groupID, userID)
	if err != nil {
		return false, err
	}
	if isLead {
		return true, nil
	}
	// 再检查是否是骨干成员
	var count int
	err = database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM teaching_group_members WHERE group_id = $1 AND user_id = $2 AND role = 'backbone'`,
		groupID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查骨干权限失败: %w", err)
	}
	return count > 0, nil
}
