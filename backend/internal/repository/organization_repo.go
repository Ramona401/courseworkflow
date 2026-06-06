package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 错误常量 ====================

var (
	ErrOrgNotFound     = errors.New("组织不存在")
	ErrOrgNameExists   = errors.New("同类型下组织名称已存在")
	ErrGroupNotFound   = errors.New("教研组不存在")
	ErrGroupNameExists = errors.New("该学校下教研组名称已存在")
	ErrMemberExists    = errors.New("该用户已是教研组成员")
	ErrMemberNotFound  = errors.New("教研组成员不存在")
)

// ==================== 组织 CRUD ====================

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

func GetOrganizationByID(ctx context.Context, id string) (*models.Organization, error) {
	org := &models.Organization{}
	query := `
		SELECT id, name, type, parent_id, admin_user_id, settings, COALESCE(logo_url,''), status, created_at, updated_at
		FROM organizations WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&org.ID, &org.Name, &org.Type, &org.ParentID, &org.AdminUserID,
		&org.Settings, &org.LogoURL, &org.Status, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("查询组织失败: %w", err)
	}
	return org, nil
}

// GetSchoolByAdminUserID 根据学校管理员用户ID获取其管理的学校
// 规则：仅返回 type='school' 的组织；若无则返回 ErrOrgNotFound
func GetSchoolByAdminUserID(ctx context.Context, adminUserID string) (*models.Organization, error) {
	org := &models.Organization{}
	query := `
		SELECT id, name, type, parent_id, admin_user_id, settings, COALESCE(logo_url,''), status, created_at, updated_at
		FROM organizations
		WHERE admin_user_id = $1 AND type = 'school'
		LIMIT 1
	`
	err := database.DB.QueryRow(ctx, query, adminUserID).Scan(
		&org.ID, &org.Name, &org.Type, &org.ParentID, &org.AdminUserID,
		&org.Settings, &org.LogoURL, &org.Status, &org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("查询学校管理员所属学校失败: %w", err)
	}
	return org, nil
}

func ListOrganizations(ctx context.Context, orgType string, parentID string) ([]*models.OrganizationListItem, error) {
	query := `
		SELECT o.id, o.name, o.type, o.parent_id, o.admin_user_id, COALESCE(o.logo_url,''), o.status, o.created_at,
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
			&item.LogoURL, &item.Status, &item.CreatedAt,
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

func GetSchoolsByRegion(ctx context.Context, regionID string) ([]*models.Organization, error) {
	query := `
		SELECT id, name, type, parent_id, admin_user_id, settings, COALESCE(logo_url,''), status, created_at, updated_at
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
			&org.Settings, &org.LogoURL, &org.Status, &org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描学校行失败: %w", err)
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}

// ==================== 迭代一 新增：组织树递归查询 ====================

// ListDescendantSchoolIDs 递归查询某区域树下的所有学校ID（WITH RECURSIVE）
//
// 用途：
//   - region_admin 数据范围解析（ResolveDataScope 的 region_admin 分支，Phase 2 接入）
//   - 迭代二积分三级分配（区域→旗下所有学校）
//
// 语义：
//   - 从 regionID 出发，沿 parent_id 链向下递归（支持多级区域嵌套，当前数据仅两级，递归退化为一层）
//   - 只返回 type='school' 且 status='active' 的组织ID
//   - 中间层 region（多级嵌套时）参与递归遍历但不计入返回结果（只要学校）
//   - regionID 本身不是 school，不会出现在结果里
//
// fail-safe：regionID 为空 → 返回空切片（非 nil），由调用方按"空集"语义处理
func ListDescendantSchoolIDs(ctx context.Context, regionID string) ([]string, error) {
	if regionID == "" {
		return []string{}, nil
	}
	// 递归 CTE：org_tree 收集 regionID 及其所有后代组织（不限类型）
	// 最外层 SELECT 再过滤出其中的 school
	query := `
		WITH RECURSIVE org_tree AS (
			-- 基准：起始区域本身，depth=0（仅用于启动递归，不计入结果）
			SELECT id, type, parent_id, status, 0 AS depth
			FROM organizations
			WHERE id = $1
			UNION ALL
			-- 递归：所有 parent_id 指向已在树内节点的下级组织，depth 逐层+1
			SELECT o.id, o.type, o.parent_id, o.status, t.depth + 1
			FROM organizations o
			JOIN org_tree t ON o.parent_id = t.id
		)
		-- depth > 0 排除起点本身：
		--   起点若误传为 school，不会把它自己当成"下级学校"返回（堵住误用缺陷）；
		--   起点若为 region（正常用法），region 自身非 school 本就不会被选，depth 过滤无副作用。
		--   只返回通过 parent_id 链真正向下到达的下级学校。
		SELECT id FROM org_tree
		WHERE type = 'school' AND status = 'active' AND depth > 0
		ORDER BY id
	`
	rows, err := database.DB.Query(ctx, query, regionID)
	if err != nil {
		return nil, fmt.Errorf("递归查询区域树下学校失败: %w", err)
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历区域树学校结果失败: %w", err)
	}
	return ids, nil
}

// ==================== v172 新增：门户板块可见性查询 ====================

// parsePortalModulesFromSettings 从组织 settings(JSONB字符串) 解析 portal_modules
//
// 规则（容错优先，保证不波及存量）：
//   - settings 为空 / 非法 JSON / 无 portal_modules 键 → 返回全开
//   - portal_modules 中缺失的板块 key → 该板块按 true 处理（缺省即开启）
//   - 只有显式写成 false 的板块才会被关闭
func parsePortalModulesFromSettings(settings string) map[string]bool {
	result := models.DefaultPortalModules() // 先全开

	settings = strings.TrimSpace(settings)
	if settings == "" || settings == "{}" {
		return result
	}

	// settings 形如 {"portal_modules":{"lesson_plan":true,"workflow":false}, ...}
	var raw struct {
		PortalModules map[string]bool `json:"portal_modules"`
	}
	if err := json.Unmarshal([]byte(settings), &raw); err != nil {
		// 解析失败 → 保持全开
		return result
	}
	if raw.PortalModules == nil {
		// 没有 portal_modules 键 → 保持全开
		return result
	}

	// 用显式配置覆盖默认值（仅覆盖出现的 key，缺失的 key 保持 true）
	for _, k := range models.AllPortalModules {
		if v, ok := raw.PortalModules[k]; ok {
			result[k] = v
		}
	}
	return result
}

// GetUserPortalModules 获取用户所属组织的门户板块可见性配置
//
// 查找链路与 GetUserOrgLogo 一致：
//   school_members → 学校 → 学校 settings；school_members 查不到则教研组兜底反查学校
// 解析学校 settings 里的 portal_modules；未绑定任何组织 / 查不到 → 返回全开
//
// 注意：admin 的"全开"由 auth_service 层兜底，不在此函数处理（此函数只管按组织配置返回）
func GetUserPortalModules(ctx context.Context, userID string) map[string]bool {
	var settings string

	// 1. 通过 school_members 查用户所属学校的 settings
	err := database.DB.QueryRow(ctx, `
		SELECT COALESCE(o.settings, '{}')
		FROM school_members sm
		JOIN organizations o ON o.id = sm.school_id
		WHERE sm.user_id = $1 AND o.status = 'active'
		LIMIT 1
	`, userID).Scan(&settings)
	if err != nil {
		// 2. 兜底：通过教研组反查学校 settings
		err = database.DB.QueryRow(ctx, `
			SELECT COALESCE(o.settings, '{}')
			FROM teaching_group_members tgm
			JOIN teaching_groups tg ON tg.id = tgm.group_id
			JOIN organizations o ON o.id = tg.school_id
			WHERE tgm.user_id = $1 AND o.status = 'active'
			LIMIT 1
		`, userID).Scan(&settings)
		if err != nil {
			// 未绑定任何学校 → 全开
			return models.DefaultPortalModules()
		}
	}

	return parsePortalModulesFromSettings(settings)
}

// ==================== 教研组 CRUD ====================

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

// ListTeachingGroups 获取教研组列表
// v109改动：lead_user_names 从成员角色表聚合所有 role='lead' 的成员名称（逗号分隔）
func ListTeachingGroups(ctx context.Context, schoolID string) ([]*models.TeachingGroupListItem, error) {
	query := `
		SELECT tg.id, tg.name, tg.school_id, tg.subject, tg.grade_range,
		       tg.lead_user_id, tg.status, tg.created_at,
		       COALESCE(o.name, '') AS school_name,
		       COALESCE(u.display_name, '') AS lead_user_name,
		       (SELECT COUNT(*) FROM teaching_group_members tgm WHERE tgm.group_id = tg.id) AS member_count,
		       COALESCE(
		         (SELECT string_agg(u2.display_name, '、' ORDER BY tgm2.joined_at)
		          FROM teaching_group_members tgm2
		          JOIN users u2 ON u2.id = tgm2.user_id
		          WHERE tgm2.group_id = tg.id AND tgm2.role = 'lead'),
		         ''
		       ) AS lead_user_names
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
			&item.LeadUserNames,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描教研组行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func UpdateTeachingGroup(ctx context.Context, id string, req *models.UpdateTeachingGroupRequest) error {
	query := `
		UPDATE teaching_groups
		SET name = $1, subject = $2, grade_range = $3,
		    description = $4, settings = $5, status = $6, updated_at = $7
		WHERE id = $8
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
		req.Name, req.Subject, req.GradeRange,
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

func ListGroupMembers(ctx context.Context, groupID string) ([]*models.GroupMemberItem, error) {
	query := `
		SELECT tgm.id, tgm.user_id, u.username, u.display_name, tgm.role, tgm.joined_at
		FROM teaching_group_members tgm
		JOIN users u ON u.id = tgm.user_id
		WHERE tgm.group_id = $1
		ORDER BY
		  CASE tgm.role WHEN 'lead' THEN 0 WHEN 'backbone' THEN 1 ELSE 2 END,
		  tgm.joined_at ASC
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

func GetUserTeachingGroups(ctx context.Context, userID string) ([]*models.TeachingGroupListItem, error) {
	query := `
		SELECT tg.id, tg.name, tg.school_id, tg.subject, tg.grade_range,
		       tg.lead_user_id, tg.status, tg.created_at,
		       COALESCE(o.name, '') AS school_name,
		       COALESCE(u.display_name, '') AS lead_user_name,
		       (SELECT COUNT(*) FROM teaching_group_members tgm2 WHERE tgm2.group_id = tg.id) AS member_count,
		       COALESCE(
		         (SELECT string_agg(u2.display_name, '、' ORDER BY tgm3.joined_at)
		          FROM teaching_group_members tgm3
		          JOIN users u2 ON u2.id = tgm3.user_id
		          WHERE tgm3.group_id = tg.id AND tgm3.role = 'lead'),
		         ''
		       ) AS lead_user_names
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
			&item.LeadUserNames,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描教研组行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// IsGroupLead 检查用户是否是某教研组的组长
// v109改动：从成员角色表查 role='lead'（支持多组长），不再只查 lead_user_id 字段
func IsGroupLead(ctx context.Context, groupID string, userID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM teaching_group_members
		 WHERE group_id = $1 AND user_id = $2 AND role = 'lead'`,
		groupID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查组长权限失败: %w", err)
	}
	// 兼容旧数据：同时检查 teaching_groups.lead_user_id
	if count == 0 {
		err = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM teaching_groups WHERE id = $1 AND lead_user_id = $2`,
			groupID, userID,
		).Scan(&count)
		if err != nil {
			return false, fmt.Errorf("检查组长权限(兼容)失败: %w", err)
		}
	}
	return count > 0, nil
}

// IsGroupLeadOrBackbone 检查用户是否有评审权限（组长或骨干）
func IsGroupLeadOrBackbone(ctx context.Context, groupID string, userID string) (bool, error) {
	isLead, err := IsGroupLead(ctx, groupID, userID)
	if err != nil {
		return false, err
	}
	if isLead {
		return true, nil
	}
	var count int
	err = database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM teaching_group_members
		 WHERE group_id = $1 AND user_id = $2 AND role = 'backbone'`,
		groupID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查骨干权限失败: %w", err)
	}
	return count > 0, nil
}

// GetGroupLeadNames 获取教研组所有组长的名称列表（逗号分隔）
func GetGroupLeadNames(ctx context.Context, groupID string) (string, error) {
	var names []string
	rows, err := database.DB.Query(ctx,
		`SELECT u.display_name FROM teaching_group_members tgm
		 JOIN users u ON u.id = tgm.user_id
		 WHERE tgm.group_id = $1 AND tgm.role = 'lead'
		 ORDER BY tgm.joined_at`,
		groupID,
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			names = append(names, name)
		}
	}
	return strings.Join(names, "、"), nil
}

// ==================== v122 方案B 新增：school_members 直接归属 ====================
//
// school_members 是"学校直接成员名单"的权威来源（v122 新增）。
// 与 teaching_group_members 正交：教研组成员自动算本校成员，但本校成员不一定要在教研组。
// 新建用户入校、加入本校教研组 都会自动写入 school_members。

// AddSchoolMember 将用户加入学校的直接成员名单
// - 幂等：ON CONFLICT 不报错
// - source 记录来源('school_admin_create'/'admin_create'/'group_member'/'migration'/'manual')
func AddSchoolMember(ctx context.Context, schoolID string, userID string, source string) error {
	if schoolID == "" || userID == "" {
		return fmt.Errorf("schoolID 或 userID 为空")
	}
	if source == "" {
		source = "manual"
	}
	_, err := database.DB.Exec(ctx, `
		INSERT INTO school_members (school_id, user_id, joined_at, source)
		VALUES ($1, $2, now(), $3)
		ON CONFLICT (school_id, user_id) DO NOTHING
	`, schoolID, userID, source)
	if err != nil {
		return fmt.Errorf("加入学校成员失败: %w", err)
	}
	return nil
}

// RemoveSchoolMember 从学校直接成员名单移除用户
// - 仅当学校管理员显式移除用户时调用
// - 禁用用户不调此函数（禁用只改 users.status）
func RemoveSchoolMember(ctx context.Context, schoolID string, userID string) error {
	_, err := database.DB.Exec(ctx,
		`DELETE FROM school_members WHERE school_id = $1 AND user_id = $2`,
		schoolID, userID,
	)
	if err != nil {
		return fmt.Errorf("移除学校成员失败: %w", err)
	}
	return nil
}

// IsUserInSchool 检查用户是否属于指定学校（v122 方案B 权威判定）
// 同时兜底查 teaching_group_members，防止回填遗漏或新加入教研组但 school_members 漏写
//
// 注意（迭代一说明）：
//   本函数保留教研组兜底，服务于"学校管理员校验本校成员、放行管理操作"等 6 处既有调用点
//   （宁可多放行本校的人，也别漏掉只在教研组的人）。行为与历史一致，不改。
//   而"数据隔离/防跨校越权"场景（教案/审核隔离）应改用下方 IsUserInSchoolStrict（无兜底），
//   两者命名即表达语义差异，杜绝再次误用。
func IsUserInSchool(ctx context.Context, userID string, schoolID string) (bool, error) {
	var count int
	// 主判：school_members
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM school_members WHERE user_id = $1 AND school_id = $2`,
		userID, schoolID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查学校直接成员失败: %w", err)
	}
	if count > 0 {
		return true, nil
	}
	// 兜底：通过教研组反查（历史兼容 + 防漏）
	err = database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		WHERE tgm.user_id = $1 AND tg.school_id = $2
	`, userID, schoolID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查用户学校归属(教研组兜底)失败: %w", err)
	}
	return count > 0, nil
}

// IsUserInSchoolStrict 严格判定用户是否属于指定学校（迭代一新增：只认 school_members，无教研组兜底）
//
// 与 IsUserInSchool 的区别：
//   - IsUserInSchool      ：school_members 主判 + 教研组兜底（宽松，给"放行管理操作"用）
//   - IsUserInSchoolStrict：只认 school_members（严格，给"数据隔离/防跨校越权"用）
//
// 设计动机（P0-02 根治）：
//   P0-02 漏洞的根源是"数据隔离误用了带兜底的归属判断"——只要加个教研组就能看跨校教案。
//   严格版只认 school_members 这一唯一权威归属来源，加教研组也无法越权看跨校数据。
//
// 用途：Phase 4 教案/审核数据隔离收口时使用；以及任何需要严格归属判断的新场景。
func IsUserInSchoolStrict(ctx context.Context, userID string, schoolID string) (bool, error) {
	if userID == "" || schoolID == "" {
		return false, nil
	}
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM school_members WHERE user_id = $1 AND school_id = $2`,
		userID, schoolID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("严格检查学校直接成员失败: %w", err)
	}
	return count > 0, nil
}

// ListSchoolMemberIDs 返回某学校所有成员的 user_id
// 用于 ListAdminUsers 按学校筛选的 IN 子查询构建
func ListSchoolMemberIDs(ctx context.Context, schoolID string) ([]string, error) {
	rows, err := database.DB.Query(ctx,
		`SELECT user_id FROM school_members WHERE school_id = $1`,
		schoolID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询学校成员ID列表失败: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// IsUserInSchoolByGroup 检查用户是否通过教研组归属于某学校（v110 老接口，保留向后兼容）
// Deprecated: v122 改用 IsUserInSchool（school_members 主判 + 教研组兜底）
// 保留此函数避免未发现的调用点编译失败
func IsUserInSchoolByGroup(ctx context.Context, userID string, schoolID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		WHERE tgm.user_id = $1 AND tg.school_id = $2
	`, userID, schoolID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查用户学校归属失败: %w", err)
	}
	return count > 0, nil
}

// ==================== 迭代一 新增：组织多管理员 CRUD（organization_admins 表，P2-05）====================
//
// organization_admins 是"组织全部管理员"的权威来源（迭代一新增）。
// 与 organizations.admin_user_id 单字段并存：旧字段保留作"主管理员"兼容，本表作"全部管理员"权威。
// role_type：region_admin（org 须为 region）/ school_admin（org 须为 school），由 service/handler 层保证类型匹配。

// AddOrgAdmin 任命某用户为某组织的管理员
// - 幂等：ON CONFLICT (org_id, user_id) 不报错（同组织同用户只一条）
// - createdBy 任命人（审计用，可空）
func AddOrgAdmin(ctx context.Context, orgID string, userID string, roleType string, createdBy string) error {
	if orgID == "" || userID == "" {
		return fmt.Errorf("orgID 或 userID 为空")
	}
	if roleType != "region_admin" && roleType != "school_admin" {
		return fmt.Errorf("无效的管理员类型: %s", roleType)
	}
	var createdByArg interface{}
	if createdBy == "" {
		createdByArg = nil
	} else {
		createdByArg = createdBy
	}
	_, err := database.DB.Exec(ctx, `
		INSERT INTO organization_admins (org_id, user_id, role_type, created_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (org_id, user_id) DO NOTHING
	`, orgID, userID, roleType, createdByArg)
	if err != nil {
		return fmt.Errorf("任命组织管理员失败: %w", err)
	}
	return nil
}

// RemoveOrgAdmin 移除某组织的某管理员
// - 找不到记录返回 ErrMemberNotFound（供上层判断）
func RemoveOrgAdmin(ctx context.Context, orgID string, userID string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM organization_admins WHERE org_id = $1 AND user_id = $2`,
		orgID, userID,
	)
	if err != nil {
		return fmt.Errorf("移除组织管理员失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// ListOrgAdmins 列出某组织的全部管理员（含用户名、显示名、类型、任命时间）
func ListOrgAdmins(ctx context.Context, orgID string) ([]*models.OrganizationAdminItem, error) {
	query := `
		SELECT oa.org_id, oa.user_id, u.username, u.display_name, oa.role_type,
		       COALESCE(oa.created_by::text, ''), oa.created_at
		FROM organization_admins oa
		JOIN users u ON u.id = oa.user_id
		WHERE oa.org_id = $1
		ORDER BY oa.role_type, oa.created_at
	`
	rows, err := database.DB.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("查询组织管理员列表失败: %w", err)
	}
	defer rows.Close()

	items := []*models.OrganizationAdminItem{}
	for rows.Next() {
		item := &models.OrganizationAdminItem{}
		if err := rows.Scan(
			&item.OrgID, &item.UserID, &item.Username, &item.DisplayName,
			&item.RoleType, &item.CreatedBy, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描组织管理员行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// CountOrgAdminsByUser 统计某用户当前还担任多少个组织的管理员
// 用途：移除某组织管理员后，判断该用户是否还是任何组织的管理员（若否，service 层可据此决定是否降级其角色）
func CountOrgAdminsByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM organization_admins WHERE user_id = $1`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计用户组织管理员身份失败: %w", err)
	}
	return count, nil
}

// ListRegionIDsByAdmin 查询某用户担任 region_admin 的所有区域组织ID
// 用途：ResolveDataScope 的 region_admin 分支——确定该用户管辖哪些区域（Phase 2 接入）
// 返回空切片（非 nil）表示该用户不是任何区域的管理员
func ListRegionIDsByAdmin(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return []string{}, nil
	}
	rows, err := database.DB.Query(ctx,
		`SELECT org_id FROM organization_admins WHERE user_id = $1 AND role_type = 'region_admin'`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询用户管辖区域失败: %w", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// UpdateOrganizationLogo 更新组织Logo URL
func UpdateOrganizationLogo(ctx context.Context, id string, logoURL string) error {
	sql := `UPDATE organizations SET logo_url = $1, updated_at = $2 WHERE id = $3`
	result, err := database.DB.Exec(ctx, sql, logoURL, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新组织Logo失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}
	return nil
}

// GetUserOrgLogo 获取用户所属组织的Logo和名称
// 查找链路：school_members → 学校 → 学校Logo → 如果没有则取区域Logo
// 返回 (logoURL, orgName)，全部为空表示用户未绑定任何组织
func GetUserOrgLogo(ctx context.Context, userID string) (string, string) {
	// 1. 通过school_members查用户所属学校
	var schoolID, schoolName, schoolLogoURL string
	var parentID *string
	err := database.DB.QueryRow(ctx, `
		SELECT o.id, o.name, COALESCE(o.logo_url, ''), o.parent_id
		FROM school_members sm
		JOIN organizations o ON o.id = sm.school_id
		WHERE sm.user_id = $1 AND o.status = 'active'
		LIMIT 1
	`, userID).Scan(&schoolID, &schoolName, &schoolLogoURL, &parentID)
	if err != nil {
		// 兜底：通过教研组反查学校
		err = database.DB.QueryRow(ctx, `
			SELECT o.id, o.name, COALESCE(o.logo_url, ''), o.parent_id
			FROM teaching_group_members tgm
			JOIN teaching_groups tg ON tg.id = tgm.group_id
			JOIN organizations o ON o.id = tg.school_id
			WHERE tgm.user_id = $1 AND o.status = 'active'
			LIMIT 1
		`, userID).Scan(&schoolID, &schoolName, &schoolLogoURL, &parentID)
		if err != nil {
			return "", ""
		}
	}

	// 2. 如果学校有Logo，直接返回
	if schoolLogoURL != "" {
		return schoolLogoURL, schoolName
	}

	// 3. 学校没有Logo，尝试从所属区域获取
	if parentID != nil && *parentID != "" {
		var regionLogoURL, regionName string
		err = database.DB.QueryRow(ctx, `
			SELECT COALESCE(logo_url, ''), name
			FROM organizations WHERE id = $1 AND status = 'active'
		`, *parentID).Scan(&regionLogoURL, &regionName)
		if err == nil && regionLogoURL != "" {
			return regionLogoURL, schoolName
		}
	}

	// 4. 都没有Logo，返回学校名称但空Logo
	return "", schoolName
}

// ==================== v139.1 新增:查询用户管理的教研组 ====================

// ListMyLeadOrBackboneGroups 查询当前用户在其中担任 lead 或 backbone 的所有教研组
//
// 用途:发布模板时,让用户从"自己管理的教研组"下拉中选择,而不是手填 UUID
// 返回:每个教研组的 ID/名称/所属学校名/当前用户在此组的角色
// 排序:lead 排前,然后按 joined_at 升序
//
// 复用 organizations + teaching_groups + teaching_group_members 三表 JOIN
func ListMyLeadOrBackboneGroups(ctx context.Context, userID string) ([]models.PublishTargetGroup, error) {
	query := `
		SELECT tg.id, tg.name, COALESCE(o.name, ''), tgm.role
		FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		LEFT JOIN organizations o ON o.id = tg.school_id
		WHERE tgm.user_id = $1
		  AND tgm.role IN ('lead', 'backbone')
		  AND tg.status = 'active'
		ORDER BY
		  CASE tgm.role WHEN 'lead' THEN 0 ELSE 1 END,
		  tgm.joined_at ASC
	`
	rows, err := database.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户管理的教研组失败: %w", err)
	}
	defer rows.Close()

	var groups []models.PublishTargetGroup
	for rows.Next() {
		g := models.PublishTargetGroup{}
		if err := rows.Scan(&g.ID, &g.Name, &g.SchoolName, &g.Role); err != nil {
			return nil, fmt.Errorf("扫描教研组行失败: %w", err)
		}
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []models.PublishTargetGroup{}
	}
	return groups, nil
}
