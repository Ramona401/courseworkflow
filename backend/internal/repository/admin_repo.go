package repository

/*
 * admin_repo.go — 统一用户管理中心数据访问层
 *
 * 归属治理批A(2026-07-04)——归属显示同源化：
 *   - 列表/详情的 school_name 改为【school_members 优先】、教研组反推兜底：
 *       COALESCE(sm_first.school_name, first_grp.school_name, '')
 *     根治"前端显示没学校、后端可见性说有学校"的显示源错位(lichao01 事件)。
 *   - 列表/详情新增 school_count(school_members 计数)——前端据此区分
 *     "零归属"与"有校籍但未加入教研组"两种状态。
 *   - 详情新增 schools 数组(AdminSchoolMembership)：逐校列出 校名/入校来源/
 *     入校时间/该校组数，供 UserDetailModal"所属学校"区块与「移出本校」按钮消费。
 *
 * 本批(区域管理员只读视图)：
 *   - AdminUserListParams 新增 SchoolIDs []string(学校白名单,region_admin 辖区口径):
 *       与单校 SchoolID 互斥(handler 保证);非 nil 即生效,空切片=匹配空集(fail-closed)。
 *       SQL 用 school_id::text = ANY($n) —— school_id 列是 uuid,入参是 []string(text[]),
 *       不加 ::text 会报 operator does not exist: uuid = text。
 *   - 新增 IsUserInSchools: 判目标用户是否属于学校白名单中任一学校,
 *     口径与列表筛选完全一致(school_members ∪ teaching_group_members),
 *     供用户详情/课程分配的 region 范围校验(列表可见的人详情必须点得开)。
 *
 * v122 方案B 主改动：
 *   - ListAdminUsers 的 school_id 筛选从 teaching_group_members 改为 school_members
 *     school_members 是 v122 新引入的"学校直接成员名单"权威来源
 *     配合 UNION 兜底查 teaching_group_members，确保历史数据不丢
 *
 * 提供跨表联合查询，用于统一用户管理中心：
 *   - ListAdminUsers    : 用户列表（含教研组/学校归属摘要）
 *   - GetAdminUserDetail: 用户详情（含课程分配+所有教研组+所有校籍）
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
	// SchoolIDs 学校白名单筛选（region_admin 辖区口径,本批新增）：
	//   与 SchoolID 互斥（handler 保证不同时传）；非 nil 即生效——
	//   非空=仅名单内学校成员，空切片=匹配空集（辖区无学校时天然看不到任何人）。
	SchoolIDs []string
	GroupID   string // 按教研组筛选
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
	// SchoolCount 校籍数(school_members 计数,批A新增)：
	//   0=零归属；>0 且 GroupCount=0 = 有校籍但未加入任何教研组
	SchoolCount int `json:"school_count"`
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
	// Schools 校籍清单(批A新增)：来自 school_members(用户归属的唯一事实源),
	// 供前端"所属学校"区块与「移出本校」按钮消费
	Schools []AdminSchoolMembership `json:"schools"`
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

// AdminSchoolMembership 用户的单条校籍记录(批A新增,数据源 school_members)
type AdminSchoolMembership struct {
	SchoolID   string `json:"school_id"`
	SchoolName string `json:"school_name"`
	Source     string `json:"source"`      // 入校来源原始值(group_member/admin_create/...)
	SourceName string `json:"source_name"` // 入校来源中文名
	JoinedAt   string `json:"joined_at"`
	GroupCount int    `json:"group_count"` // 该用户在此学校的教研组数(0=有校籍但无组)
	// IsSchoolAdmin 批C2：目标是否为该校管理员(organization_admins ∪ admin_user_id 双来源)，
	// 前端据此在"移出本校"弹窗提供"同时移除任命"勾选
	IsSchoolAdmin bool `json:"is_school_admin"`
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

// ==================== 角色/来源中文名映射 ====================

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

// schoolSourceNameMap 入校来源中文名(批A新增;未知来源回退显示原始值)
var schoolSourceNameMap = map[string]string{
	"group_member":                    "随教研组自动加入",
	"admin_create":                    "管理员建号入校",
	"admin_batch_create":              "批量导入入校",
	"school_admin_batch_create":       "批量导入入校",
	"admin_multi_school_batch_create": "跨校批量导入入校",
	"manual":                          "手动添加",
}

// ==================== 用户列表联合查询 ====================

// ListAdminUsers 用户列表（含教研组/学校归属摘要，支持多条件筛选+分页）
// v122 方案B：school_id 筛选走 school_members ∪ teaching_group_members（并集兜底）
// 区域批：新增 SchoolIDs 学校白名单筛选（region_admin 辖区），并集口径与单校完全一致
// 批A：school_name 改为 school_members 优先(sm_first)、教研组反推兜底；新增 school_count
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
	} else if params.SchoolIDs != nil {
		// 区域批新增：学校白名单（region_admin 辖区口径）——并集口径与单校完全一致。
		// school_id 列是 uuid 而入参 []string 编码为 text[]，须 ::text 后再 ANY，
		// 否则报 operator does not exist: uuid = text；空切片 ANY('{}') 天然匹配空集（fail-closed）。
		where += fmt.Sprintf(` AND u.id IN (
			SELECT user_id FROM school_members WHERE school_id::text = ANY($%d)
			UNION
			SELECT tgm.user_id FROM teaching_group_members tgm
			JOIN teaching_groups tg ON tg.id = tgm.group_id
			WHERE tg.school_id::text = ANY($%d)
		)`, idx, idx)
		args = append(args, params.SchoolIDs)
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

	// 查数据（批A：sm_first=校籍首选学校名 / sch_cnt=校籍数；school_name 校籍优先）
	offset := (params.Page - 1) * params.PageSize
	dataArgs := append(args, params.PageSize, offset)

	dataSQL := fmt.Sprintf(`
		SELECT
			u.id, u.username, u.display_name, u.role, u.status,
			u.login_count, u.last_login_at, u.created_at,
			COALESCE(sm_first.school_name, first_grp.school_name, '') AS school_name,
			COALESCE(first_grp.group_name, '') AS group_name,
			COALESCE(first_grp.member_role, '') AS group_role,
			COALESCE(grp_cnt.cnt, 0) AS group_count,
			COALESCE(sch_cnt.cnt, 0) AS school_count
		FROM users u
		LEFT JOIN LATERAL (
			SELECT o2.name AS school_name
			FROM school_members sm
			JOIN organizations o2 ON o2.id = sm.school_id
			WHERE sm.user_id = u.id
			ORDER BY sm.joined_at ASC LIMIT 1
		) sm_first ON true
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
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt FROM school_members sm2 WHERE sm2.user_id = u.id
		) sch_cnt ON true
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
			&item.SchoolCount,
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

// ==================== 学校白名单归属判定(区域批新增) ====================

// IsUserInSchools 判目标用户是否属于学校白名单中【任一】学校
//
// 口径与 ListAdminUsers 的学校筛选完全一致（school_members ∪ teaching_group_members），
// 保证"用户列表里看得到的人，详情/课程分配一定点得开"。
// 供 region_admin 的用户详情范围校验（ensureUserInScope 的 region 分支）消费。
//
// 入参防御：userID 空或名单为空 → 直接返回 false（fail-closed，不发 SQL）。
// school_id 列是 uuid，入参 []string 编码为 text[]，须 ::text 后再 ANY。
func IsUserInSchools(ctx context.Context, userID string, schoolIDs []string) (bool, error) {
	if userID == "" || len(schoolIDs) == 0 {
		return false, nil
	}
	var in bool
	err := database.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM school_members
			WHERE user_id = $1 AND school_id::text = ANY($2)
		) OR EXISTS(
			SELECT 1 FROM teaching_group_members tgm
			JOIN teaching_groups tg ON tg.id = tgm.group_id
			WHERE tgm.user_id = $1 AND tg.school_id::text = ANY($2)
		)
	`, userID, schoolIDs).Scan(&in)
	if err != nil {
		return false, fmt.Errorf("校验用户学校归属失败: %w", err)
	}
	return in, nil
}

// ==================== 用户详情 ====================

// GetAdminUserDetail 用户详情（批A：school_name 校籍优先 + 新增 schools 校籍清单）
func GetAdminUserDetail(ctx context.Context, userID string) (*AdminUserDetailResult, error) {
	var base AdminUserListItem
	var lastLoginAt *time.Time
	var createdAt time.Time

	err := database.DB.QueryRow(ctx, `
		SELECT
			u.id, u.username, u.display_name, u.role, u.status,
			u.login_count, u.last_login_at, u.created_at,
			COALESCE(sm_first.school_name, first_grp.school_name, '') AS school_name,
			COALESCE(first_grp.group_name, '') AS group_name,
			COALESCE(first_grp.member_role, '') AS group_role,
			COALESCE(grp_cnt.cnt, 0) AS group_count,
			COALESCE(sch_cnt.cnt, 0) AS school_count
		FROM users u
		LEFT JOIN LATERAL (
			SELECT o2.name AS school_name
			FROM school_members sm
			JOIN organizations o2 ON o2.id = sm.school_id
			WHERE sm.user_id = u.id
			ORDER BY sm.joined_at ASC LIMIT 1
		) sm_first ON true
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
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt FROM school_members sm2 WHERE sm2.user_id = u.id
		) sch_cnt ON true
		WHERE u.id = $1
	`, userID).Scan(
		&base.ID, &base.Username, &base.DisplayName, &base.Role, &base.Status,
		&base.LoginCount, &lastLoginAt, &createdAt,
		&base.SchoolName, &base.GroupName, &base.GroupRole, &base.GroupCount,
		&base.SchoolCount,
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

	// 校籍清单(批A新增)：school_members 逐校列出，含入校来源与该校组数
	schoolRows, err := database.DB.Query(ctx, `
		SELECT
			sm.school_id,
			COALESCE(o.name, '') AS school_name,
			COALESCE(sm.source, '') AS source,
			sm.joined_at,
			(
				SELECT COUNT(*) FROM teaching_group_members tgm
				JOIN teaching_groups tg ON tg.id = tgm.group_id
				WHERE tgm.user_id = sm.user_id AND tg.school_id = sm.school_id
			) AS group_cnt,
			(
				EXISTS(SELECT 1 FROM organization_admins oa
					WHERE oa.org_id = sm.school_id AND oa.user_id = sm.user_id AND oa.role_type = 'school_admin')
				OR EXISTS(SELECT 1 FROM organizations o2
					WHERE o2.id = sm.school_id AND o2.admin_user_id = sm.user_id)
			) AS is_school_admin
		FROM school_members sm
		JOIN organizations o ON o.id = sm.school_id
		WHERE sm.user_id = $1
		ORDER BY sm.joined_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("查询校籍清单失败: %w", err)
	}
	defer schoolRows.Close()

	var schools []AdminSchoolMembership
	for schoolRows.Next() {
		var s AdminSchoolMembership
		var joinedAt *time.Time
		if err := schoolRows.Scan(&s.SchoolID, &s.SchoolName, &s.Source, &joinedAt, &s.GroupCount, &s.IsSchoolAdmin); err != nil {
			continue
		}
		if joinedAt != nil {
			s.JoinedAt = joinedAt.Format("2006-01-02 15:04:05")
		}
		if n, ok := schoolSourceNameMap[s.Source]; ok {
			s.SourceName = n
		} else {
			s.SourceName = s.Source
		}
		schools = append(schools, s)
	}
	if schools == nil {
		schools = []AdminSchoolMembership{}
	}

	return &AdminUserDetailResult{
		AdminUserListItem: base,
		CourseAssignments: courses,
		TeachingGroups:    groups,
		Schools:           schools,
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
