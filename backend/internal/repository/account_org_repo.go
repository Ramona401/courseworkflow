package repository

// account_org_repo.go — 个人中心"我的组织归属"聚合查询
//
// 背景（测试反馈 7-1 #2）：
//   平台用户在个人中心看不到自己的组织归属和职位角色，只知道系统身份（如"骨干教师"），
//   不知道自己属于哪个区域、哪个学校、在哪些教研组、在各组里是组长/骨干/普通成员。
//   测试者反映"对平台各板块的权限层级、审核流转比较茫然"。
//
// 本文件提供一个只读聚合查询 GetUserOrganizationProfile，把散落在三张表的归属信息
// 一次性组装成"区域 → 学校 → 教研组(含我的角色)"的层级结构，供个人中心"我的组织"Tab 展示。
//
// 数据来源与既有权威链路完全一致，不引入任何新口径：
//   1) 学校归属：school_members（直接成员，权威）∪ teaching_group_members 反查（教研组自动算本校）
//      —— 与 GetSchoolIDByUserID / GetUserPortalModules 同源。
//   2) 教研组归属与我的角色：teaching_group_members.role（member/backbone/lead）
//      —— 与 GetUserTeachingGroups 同源，额外带出 role。
//   3) 区域：organizations.parent_id（学校的父组织即区域），学校可能没挂区域故用 LEFT JOIN。
//
// 纯只读，不改任何状态；调用方为个人中心接口（登录即可查自己）。

import (
	"context"
	"fmt"

	"tedna/internal/database"
)

// ==================== 响应结构体 ====================

// UserOrgGroupItem 我所在的单个教研组（含我在该组的角色 + 该组所属学校/区域）
type UserOrgGroupItem struct {
	GroupID    string `json:"group_id"`    // 教研组ID
	GroupName  string `json:"group_name"`  // 教研组名称
	Subject    string `json:"subject"`     // 教研组学科
	GradeRange string `json:"grade_range"` // 教研组学段范围
	MyRole     string `json:"my_role"`     // 我在此组的角色：member/backbone/lead
	SchoolID   string `json:"school_id"`   // 该组所属学校ID
	SchoolName string `json:"school_name"` // 该组所属学校名称
	RegionID   string `json:"region_id"`   // 该学校所属区域ID（可空串=学校未挂区域）
	RegionName string `json:"region_name"` // 该学校所属区域名称（可空串）
}

// UserOrgSchoolItem 我所属的单个学校（来自 school_members 直接成员名单）
type UserOrgSchoolItem struct {
	SchoolID    string `json:"school_id"`    // 学校ID
	SchoolName  string `json:"school_name"`  // 学校名称
	RegionID    string `json:"region_id"`    // 所属区域ID（可空串）
	RegionName  string `json:"region_name"`  // 所属区域名称（可空串）
	Source      string `json:"source"`       // 入校来源（group_member/admin_create/...）
	IsSchoolAdmin bool `json:"is_school_admin"` // 我是否为该校的学校管理员
}

// UserOrganizationProfile 个人组织归属聚合结果
//
// schools = 我通过 school_members 直接归属的学校（可能多校）
// groups  = 我所在的全部教研组（含我在各组的角色）
// 前端据此渲染"区域 → 学校 → 教研组(我的角色)"的组织树。
type UserOrganizationProfile struct {
	Schools []UserOrgSchoolItem `json:"schools"` // 直接归属的学校列表
	Groups  []UserOrgGroupItem  `json:"groups"`  // 所在教研组列表（含我的角色）
}

// ==================== 聚合查询 ====================

// GetUserOrganizationProfile 查询用户完整组织归属（区域/学校/教研组+我的角色）
//
// userID 为空直接返回空结构（非错误）。任一子查询失败上抛 error 由 handler 处理。
// 返回的 Schools / Groups 恒为非 nil 切片（空归属时为空切片，保证前端 .map 不崩）。
func GetUserOrganizationProfile(ctx context.Context, userID string) (*UserOrganizationProfile, error) {
	profile := &UserOrganizationProfile{
		Schools: make([]UserOrgSchoolItem, 0),
		Groups:  make([]UserOrgGroupItem, 0),
	}
	if userID == "" {
		return profile, nil
	}

	// ---------- 1) 我直接归属的学校（school_members，带区域与是否本校管理员） ----------
	//
	// is_school_admin 双来源判定（与 admin_repo 口径一致）：
	//   organizations.admin_user_id 单字段 ∪ organization_admins(role_type='school_admin')
	schoolQuery := `
		SELECT
			sch.id::text                        AS school_id,
			sch.name                            AS school_name,
			COALESCE(reg.id::text, '')          AS region_id,
			COALESCE(reg.name, '')              AS region_name,
			COALESCE(sm.source, '')             AS source,
			(
				sch.admin_user_id = $1
				OR EXISTS (
					SELECT 1 FROM organization_admins oa
					WHERE oa.org_id = sch.id
					  AND oa.user_id = $1
					  AND oa.role_type = 'school_admin'
				)
			)                                    AS is_school_admin
		FROM school_members sm
		JOIN organizations sch ON sch.id = sm.school_id AND sch.type = 'school' AND sch.status = 'active'
		LEFT JOIN organizations reg ON reg.id = sch.parent_id
		WHERE sm.user_id = $1
		ORDER BY sch.name ASC
	`
	schoolRows, err := database.DB.Query(ctx, schoolQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户所属学校失败: %w", err)
	}
	defer schoolRows.Close()
	for schoolRows.Next() {
		var s UserOrgSchoolItem
		if err := schoolRows.Scan(
			&s.SchoolID, &s.SchoolName, &s.RegionID, &s.RegionName,
			&s.Source, &s.IsSchoolAdmin,
		); err != nil {
			return nil, fmt.Errorf("扫描学校归属行失败: %w", err)
		}
		profile.Schools = append(profile.Schools, s)
	}
	if err := schoolRows.Err(); err != nil {
		return nil, fmt.Errorf("遍历学校归属结果失败: %w", err)
	}

	// ---------- 2) 我所在的全部教研组（含我在各组的角色 + 组所属学校/区域） ----------
	groupQuery := `
		SELECT
			tg.id::text                         AS group_id,
			tg.name                             AS group_name,
			COALESCE(tg.subject, '')            AS subject,
			COALESCE(tg.grade_range, '')        AS grade_range,
			tgm.role                            AS my_role,
			sch.id::text                        AS school_id,
			COALESCE(sch.name, '')              AS school_name,
			COALESCE(reg.id::text, '')          AS region_id,
			COALESCE(reg.name, '')              AS region_name
		FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id AND tg.status = 'active'
		LEFT JOIN organizations sch ON sch.id = tg.school_id
		LEFT JOIN organizations reg ON reg.id = sch.parent_id
		WHERE tgm.user_id = $1
		ORDER BY sch.name ASC, tg.name ASC
	`
	groupRows, err := database.DB.Query(ctx, groupQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户所在教研组失败: %w", err)
	}
	defer groupRows.Close()
	for groupRows.Next() {
		var g UserOrgGroupItem
		if err := groupRows.Scan(
			&g.GroupID, &g.GroupName, &g.Subject, &g.GradeRange, &g.MyRole,
			&g.SchoolID, &g.SchoolName, &g.RegionID, &g.RegionName,
		); err != nil {
			return nil, fmt.Errorf("扫描教研组归属行失败: %w", err)
		}
		profile.Groups = append(profile.Groups, g)
	}
	if err := groupRows.Err(); err != nil {
		return nil, fmt.Errorf("遍历教研组归属结果失败: %w", err)
	}

	return profile, nil
}
