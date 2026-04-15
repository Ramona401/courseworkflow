package models

import (
	"time"
)

// ==================== 组织模型（对应 organizations 表） ====================

// Organization 组织模型（区域/学校）
type Organization struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	ParentID    *string    `json:"parent_id"`
	AdminUserID *string    `json:"admin_user_id"`
	Settings    string     `json:"settings"`
	Status      string     `json:"status"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// ==================== 教研组模型（对应 teaching_groups 表） ====================

// TeachingGroup 教研组模型
// v109改动：多组长支持——组长通过 teaching_group_members.role='lead' 管理
// lead_user_id 字段保留兼容性，不再作为主要组长标识
type TeachingGroup struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	SchoolID    string     `json:"school_id"`
	Subject     string     `json:"subject"`
	GradeRange  string     `json:"grade_range"`
	LeadUserID  *string    `json:"lead_user_id"`  // 兼容保留，实际组长通过成员角色管理
	Description string     `json:"description"`
	Settings    string     `json:"settings"`
	Status      string     `json:"status"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// ==================== 教研组成员模型（对应 teaching_group_members 表） ====================

// TeachingGroupMember 教研组成员关联
type TeachingGroupMember struct {
	ID       string     `json:"id"`
	GroupID  string     `json:"group_id"`
	UserID   string     `json:"user_id"`
	Role     string     `json:"role"`     // member=普通成员 / backbone=骨干教师 / lead=教研组长
	JoinedAt *time.Time `json:"joined_at"`
}

// ==================== 组织类型常量 ====================

const (
	OrgTypeRegion = "region"
	OrgTypeSchool = "school"
)

var ValidOrgTypes = []string{OrgTypeRegion, OrgTypeSchool}

func IsValidOrgType(t string) bool {
	for _, v := range ValidOrgTypes {
		if v == t {
			return true
		}
	}
	return false
}

// ==================== 教研组成员角色常量 ====================

const (
	GroupMemberRoleMember   = "member"   // 普通成员
	GroupMemberRoleBackbone = "backbone" // 骨干教师
	GroupMemberRoleLead     = "lead"     // 教研组长（v109新增，支持多组长）
)

// ValidGroupMemberRoles 有效的教研组成员角色列表
var ValidGroupMemberRoles = []string{
	GroupMemberRoleMember,
	GroupMemberRoleBackbone,
	GroupMemberRoleLead,
}

func IsValidGroupMemberRole(role string) bool {
	for _, v := range ValidGroupMemberRoles {
		if v == role {
			return true
		}
	}
	return false
}

// ==================== 请求结构体 ====================

type CreateOrganizationRequest struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	ParentID    *string `json:"parent_id"`
	AdminUserID *string `json:"admin_user_id"`
}

type UpdateOrganizationRequest struct {
	Name        string  `json:"name"`
	AdminUserID *string `json:"admin_user_id"`
	Settings    string  `json:"settings"`
	Status      string  `json:"status"`
}

// CreateTeachingGroupRequest 创建教研组请求
// v109改动：移除 LeadUserID，组长通过成员管理设置
type CreateTeachingGroupRequest struct {
	Name        string `json:"name"`
	SchoolID    string `json:"school_id"`
	Subject     string `json:"subject"`
	GradeRange  string `json:"grade_range"`
	Description string `json:"description"`
}

// UpdateTeachingGroupRequest 更新教研组请求
// v109改动：移除 LeadUserID，组长通过成员管理设置
type UpdateTeachingGroupRequest struct {
	Name        string `json:"name"`
	Subject     string `json:"subject"`
	GradeRange  string `json:"grade_range"`
	Description string `json:"description"`
	Settings    string `json:"settings"`
	Status      string `json:"status"`
}

type AddGroupMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// ==================== 响应结构体 ====================

type OrganizationListResponse struct {
	Organizations []*OrganizationListItem `json:"organizations"`
	Total         int                     `json:"total"`
}

type OrganizationListItem struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	ParentID      *string    `json:"parent_id"`
	ParentName    string     `json:"parent_name"`
	AdminUserID   *string    `json:"admin_user_id"`
	AdminUserName string     `json:"admin_user_name"`
	Status        string     `json:"status"`
	GroupCount    int        `json:"group_count"`
	MemberCount   int        `json:"member_count"`
	CreatedAt     *time.Time `json:"created_at"`
}

type TeachingGroupListResponse struct {
	Groups []*TeachingGroupListItem `json:"groups"`
	Total  int                      `json:"total"`
}

// TeachingGroupListItem 教研组列表单条
// v109改动：LeadUserName → LeadUserNames（支持多组长，逗号分隔）
type TeachingGroupListItem struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	SchoolID      string     `json:"school_id"`
	SchoolName    string     `json:"school_name"`
	Subject       string     `json:"subject"`
	GradeRange    string     `json:"grade_range"`
	LeadUserID    *string    `json:"lead_user_id"`    // 兼容保留
	LeadUserName  string     `json:"lead_user_name"`  // 兼容保留（第一个组长名称）
	LeadUserNames string     `json:"lead_user_names"` // v109新增：所有组长名称，逗号分隔
	MemberCount   int        `json:"member_count"`
	Status        string     `json:"status"`
	CreatedAt     *time.Time `json:"created_at"`
}

// TeachingGroupDetailResponse 教研组详情响应（含成员列表）
// v109改动：LeadUserName → LeadUserNames
type TeachingGroupDetailResponse struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	SchoolID      string             `json:"school_id"`
	SchoolName    string             `json:"school_name"`
	Subject       string             `json:"subject"`
	GradeRange    string             `json:"grade_range"`
	LeadUserID    *string            `json:"lead_user_id"`    // 兼容保留
	LeadUserName  string             `json:"lead_user_name"`  // 兼容保留
	LeadUserNames string             `json:"lead_user_names"` // v109新增
	Description   string             `json:"description"`
	Settings      string             `json:"settings"`
	Status        string             `json:"status"`
	Members       []*GroupMemberItem `json:"members"`
	CreatedAt     *time.Time         `json:"created_at"`
	UpdatedAt     *time.Time         `json:"updated_at"`
}

// GroupMemberItem 教研组成员列表单条
// role 现在可以是 member / backbone / lead
type GroupMemberItem struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"` // member / backbone / lead
	JoinedAt    *time.Time `json:"joined_at"`
}
