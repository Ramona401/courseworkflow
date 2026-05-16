package repository

/*
 * admin_repo.go — 统一用户管理中心数据访问层
 *
 * v122 方案B 主改动：
 *   - ListAdminUsers 的 school_id 筛选从 teaching_group_members 改为 school_members
 *     school_members 是 v122 新引入的"学校直接成员名单"权威来源
 *     配合 UNION 兜底查 teaching_group_members，确保历史数据不丢
 *
 * 提供跨表联合查询，用于统一用户管理中心：
 *   - ListAdminUsers    : 用户列表（含教研组/学校归属摘要）
 *   - GetAdminUserDetail: 用户详情（含课程分配+所有教研组）
 *   - GetAdminStats     : 统计摘要（用户数/组织数/教研组数/活跃数）
 *
 * 角色名称（与学校体系对齐）：
 *   admin           → 系统管理员
 *   senior_operator → 学校管理员
 *   operator        → 骨干教师
 *   viewer          → 普通教师
 */

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
)

// ==================== 参数结构 ====================

// AdminUserListParams 用户列表查询参数
type AdminUserListParams struct {
	Page     int
	PageSize int
	Role     string // 按课件审核角色筛选
	Status   string // 按状态筛选
	Keyword  string // 按用户名/显示名模糊搜索
	SchoolID string // 按学校筛选（v122：走 school_members ∪ teaching_group_members）
	GroupID  string // 按教研组筛选
}

// ==================== 响应结构 ====================

type AdminUserListItem struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	RoleName    string  `json:"role_name"`
	Status      string  `json:"status"`
	LoginCount  int     `json:"login_count"`
	LastLoginAt *string `json:"last_login_at"`
	CreatedAt   string  `json:"created_at"`
	SchoolName  string  `json:"school_name"`
	GroupName   string  `json:"group_name"`
	GroupRole   string  `json:"group_role"`
	GroupCount  int     `json:"group_count"`
}

type AdminUserListResult struct {
	Users    []AdminUserListItem `json:"users"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

type AdminUserDetailResult struct {
	AdminUserListItem
	CourseAssignments []AdminCourseAssignment `json:"course_assignments"`
	TeachingGroups    []AdminGroupMembership  `json:"teaching_groups"`
}

type AdminCourseAssignment struct {
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	AssignedAt string `json:"assigned_at"`
}

type AdminGroupMembership struct {
	GroupID    string `json:"group_id"`
	GroupName  string `json:"group_name"`
	SchoolName string `json:"school_name"`
	Role       string `json:"role"`
	RoleName   string `json:"role_name"`
	IsLead     bool   `json:"is_lead"`
	JoinedAt   string `json:"joined_at"`
}

type AdminStats struct {
	TotalUsers          int `json:"total_users"`
	ActiveUsers         int `json:"active_users"`
	DisabledUsers       int `json:"disabled_users"`
	TotalOrgs           int `json:"total_orgs"`
	TotalSchools        int `json:"total_schools"`
	TotalGroups         int `json:"total_groups"`
	TotalMembers        int `json:"total_members"`
	AdminCount          int `json:"admin_count"`
	SeniorOperatorCount int `json:"senior_operator_count"`
	OperatorCount       int `json:"operator_count"`
	ViewerCount         int `json:"viewer_count"`
}

// ==================== 角色中文名映射 ====================

var roleNameMap = map[string]string{
	"admin":           "系统管理员",
	"senior_operator": "学校管理员",
	"operator":        "骨干教师",
	"viewer":          "普通教师",
}

var memberRoleNameMap = map[string]string{
	"member":   "普通成员",
	"backbone": "骨干教师",
	"lead":     "教研组长",
}

// ==================== 用户列表联合查询 ====================

// ListAdminUsers 用户列表（含教研组/学校归属摘要，支持多条件筛选+分页）
// v122 方案B：school_id 筛选走 school_members ∪ teaching_group_members（并集兜底）
func ListAdminUsers(ctx context.Context, params AdminUserListParams) (*AdminUserListResult, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	idx := 1

	if params.Role != "" {
		where += fmt.Sprintf(" AND u.role = $%d", idx)
		args = append(args, params.Role)
		idx++
	}
	if params.Status != "" {
		where += fmt.Sprintf(" AND u.status = $%d", idx)
		args = append(args, params.Status)
		idx++
	}
	if params.Keyword != "" {
		where += fmt.Sprintf(" AND (u.username ILIKE $%d OR u.display_name ILIKE $%d)", idx, idx+1)
		kw := "%" + params.Keyword + "%"
		args = append(args, kw, kw)
		idx += 2
	}
	if params.SchoolID != "" {
		// v122 方案B：school_members 权威名单 ∪ teaching_group_members 兜底
		// 这样新建用户（只在 school_members）和历史用户（只在教研组）都能被查到
		where += fmt.Sprintf(` AND u.id IN (
			SELECT user_id FROM school_members WHERE school_id = $%d
			UNION
			SELECT tgm.user_id FROM teaching_group_members tgm
			JOIN teaching_groups tg ON tg.id = tgm.group_id
			WHERE tg.school_id = $%d
		)`, idx, idx)
		args = append(args, params.SchoolID)
		idx++
	}
	if params.GroupID != "" {
		where += fmt.Sprintf(` AND u.id IN (
			SELECT user_id FROM teaching_group_members WHERE group_id = $%d
		)`, idx)
		args = append(args, params.GroupID)
		idx++
	}

	// 查总数
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM users u %s`, where)
	var total int
	if err := database.DB.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("统计用户总数失败: %w", err)
	}

	// 查数据
	offset := (params.Page - 1) * params.PageSize
	dataArgs := append(args, params.PageSize, offset)

	dataSQL := fmt.Sprintf(`
		SELECT
			u.id, u.username, u.display_name, u.role, u.status,
			u.login_count, u.last_login_at, u.created_at,
			COALESCE(first_grp.school_name, '') AS school_name,
			COALESCE(first_grp.group_name, '') AS group_name,
			COALESCE(first_grp.member_role, '') AS group_role,
			COALESCE(grp_cnt.cnt, 0) AS group_count
		FROM users u
		LEFT JOIN LATERAL (
			SELECT o.name AS school_name, tg.name AS group_name, tgm.role AS member_role
			FROM teaching_group_members tgm
			JOIN teaching_groups tg ON tg.id = tgm.group_id
			JOIN organizations o ON o.id = tg.school_id
			WHERE tgm.user_id = u.id
			ORDER BY tgm.joined_at ASC LIMIT 1
		) first_grp ON true
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt FROM teaching_group_members tgm2 WHERE tgm2.user_id = u.id
		) grp_cnt ON true
		%s
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, idx, idx+1)

	rows, err := database.DB.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}
	defer rows.Close()

	var users []AdminUserListItem
	for rows.Next() {
		var item AdminUserListItem
		var lastLoginAt *time.Time
		var createdAt time.Time

		if err := rows.Scan(
			&item.ID, &item.Username, &item.DisplayName,
			&item.Role, &item.Status,
			&item.LoginCount, &lastLoginAt, &createdAt,
			&item.SchoolName, &item.GroupName, &item.GroupRole, &item.GroupCount,
		); err != nil {
			return nil, fmt.Errorf("扫描用户行失败: %w", err)
		}

		item.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		if lastLoginAt != nil {
			s := lastLoginAt.Format("2006-01-02 15:04:05")
			item.LastLoginAt = &s
		}
		if n, ok := roleNameMap[item.Role]; ok {
			item.RoleName = n
		} else {
			item.RoleName = item.Role
		}

		users = append(users, item)
	}

	if users == nil {
		users = []AdminUserListItem{}
	}

	return &AdminUserListResult{
		Users:    users,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

// ==================== 用户详情 ====================

func GetAdminUserDetail(ctx context.Context, userID string) (*AdminUserDetailResult, error) {
	var base AdminUserListItem
	var lastLoginAt *time.Time
	var createdAt time.Time

	err := database.DB.QueryRow(ctx, `
		SELECT
			u.id, u.username, u.display_name, u.role, u.status,
			u.login_count, u.last_login_at, u.created_at,
			COALESCE(first_grp.school_name, '') AS school_name,
			COALESCE(first_grp.group_name, '') AS group_name,
			COALESCE(first_grp.member_role, '') AS group_role,
			COALESCE(grp_cnt.cnt, 0) AS group_count
		FROM users u
		LEFT JOIN LATERAL (
			SELECT o.name AS school_name, tg.name AS group_name, tgm.role AS member_role
			FROM teaching_group_members tgm
			JOIN teaching_groups tg ON tg.id = tgm.group_id
			JOIN organizations o ON o.id = tg.school_id
			WHERE tgm.user_id = u.id
			ORDER BY tgm.joined_at ASC LIMIT 1
		) first_grp ON true
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt FROM teaching_group_members tgm2 WHERE tgm2.user_id = u.id
		) grp_cnt ON true
		WHERE u.id = $1
	`, userID).Scan(
		&base.ID, &base.Username, &base.DisplayName, &base.Role, &base.Status,
		&base.LoginCount, &lastLoginAt, &createdAt,
		&base.SchoolName, &base.GroupName, &base.GroupRole, &base.GroupCount,
	)
	if err != nil {
		return nil, fmt.Errorf("查询用户详情失败: %w", err)
	}

	base.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
	if lastLoginAt != nil {
		s := lastLoginAt.Format("2006-01-02 15:04:05")
		base.LastLoginAt = &s
	}
	if n, ok := roleNameMap[base.Role]; ok {
		base.RoleName = n
	}

	// 课程分配
	courseRows, err := database.DB.Query(ctx, `
		SELECT uca.course_code, COALESCE(c.course_name, uca.course_code) AS course_name, uca.assigned_at
		FROM user_course_assignments uca
		LEFT JOIN courses c ON c.course_code = uca.course_code
		WHERE uca.user_id = $1
		ORDER BY uca.assigned_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询课程分配失败: %w", err)
	}
	defer courseRows.Close()

	var courses []AdminCourseAssignment
	for courseRows.Next() {
		var a AdminCourseAssignment
		var assignedAt *time.Time
		if err := courseRows.Scan(&a.CourseCode, &a.CourseName, &assignedAt); err != nil {
			continue
		}
		if assignedAt != nil {
			a.AssignedAt = assignedAt.Format("2006-01-02 15:04:05")
		}
		courses = append(courses, a)
	}
	if courses == nil {
		courses = []AdminCourseAssignment{}
	}

	// 所有教研组归属
	groupRows, err := database.DB.Query(ctx, `
		SELECT
			tg.id AS group_id,
			tg.name AS group_name,
			COALESCE(o.name, '') AS school_name,
			tgm.role AS member_role,
			tgm.joined_at,
			(tg.lead_user_id = $1) AS is_lead
		FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		LEFT JOIN organizations o ON o.id = tg.school_id
		WHERE tgm.user_id = $1
		ORDER BY tgm.joined_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询教研组归属失败: %w", err)
	}
	defer groupRows.Close()

	var groups []AdminGroupMembership
	for groupRows.Next() {
		var g AdminGroupMembership
		var joinedAt time.Time
		if err := groupRows.Scan(
			&g.GroupID, &g.GroupName, &g.SchoolName,
			&g.Role, &joinedAt, &g.IsLead,
		); err != nil {
			continue
		}
		g.JoinedAt = joinedAt.Format("2006-01-02 15:04:05")
		if n, ok := memberRoleNameMap[g.Role]; ok {
			g.RoleName = n
		} else {
			g.RoleName = g.Role
		}
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []AdminGroupMembership{}
	}

	return &AdminUserDetailResult{
		AdminUserListItem: base,
		CourseAssignments: courses,
		TeachingGroups:    groups,
	}, nil
}

// ==================== 统计摘要 ====================

func GetAdminStats(ctx context.Context) (*AdminStats, error) {
	stats := &AdminStats{}

	err := database.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'active') AS active,
			COUNT(*) FILTER (WHERE status = 'disabled') AS disabled,
			COUNT(*) FILTER (WHERE role = 'admin') AS admin_cnt,
			COUNT(*) FILTER (WHERE role = 'senior_operator') AS senior_cnt,
			COUNT(*) FILTER (WHERE role = 'operator') AS operator_cnt,
			COUNT(*) FILTER (WHERE role = 'viewer') AS viewer_cnt
		FROM users
	`).Scan(
		&stats.TotalUsers, &stats.ActiveUsers, &stats.DisabledUsers,
		&stats.AdminCount, &stats.SeniorOperatorCount,
		&stats.OperatorCount, &stats.ViewerCount,
	)
	if err != nil {
		return nil, fmt.Errorf("统计用户失败: %w", err)
	}

	_ = database.DB.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_orgs,
			COUNT(*) FILTER (WHERE type = 'school') AS total_schools
		FROM organizations
	`).Scan(&stats.TotalOrgs, &stats.TotalSchools)

	_ = database.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM teaching_groups WHERE status = 'active'
	`).Scan(&stats.TotalGroups)

	_ = database.DB.QueryRow(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM teaching_group_members
	`).Scan(&stats.TotalMembers)

	return stats, nil
}
