package models

import (
	"time"
)

// ==================== 组织模型（对应 organizations 表） ====================

// Organization 组织模型（区域/学校）
// 支持区域→学校两级组织结构
// 区域(type=region)下辖多个学校(type=school)
type Organization struct {
	ID          string     `json:"id"`             // UUID主键
	Name        string     `json:"name"`           // 组织名称
	Type        string     `json:"type"`           // 组织类型：region=区域 / school=学校
	ParentID    *string    `json:"parent_id"`      // 父级组织ID（学校→所属区域）
	AdminUserID *string    `json:"admin_user_id"`  // 组织管理员用户ID
	Settings    string     `json:"settings"`       // 组织级设置JSON
	Status      string     `json:"status"`         // 状态：active / disabled
	CreatedAt   *time.Time `json:"created_at"`     // 创建时间
	UpdatedAt   *time.Time `json:"updated_at"`     // 更新时间
}

// ==================== 教研组模型（对应 teaching_groups 表） ====================

// TeachingGroup 教研组模型
// 教研组属于某个学校，按学科+学段划分
// 每个教研组有一个组长(lead_user_id)
type TeachingGroup struct {
	ID          string     `json:"id"`             // UUID主键
	Name        string     `json:"name"`           // 教研组名称
	SchoolID    string     `json:"school_id"`      // 所属学校ID → organizations.id
	Subject     string     `json:"subject"`        // 学科（如：AI课程、数学、语文）
	GradeRange  string     `json:"grade_range"`    // 学段范围（如：K1-K3、G7-G9）
	LeadUserID  *string    `json:"lead_user_id"`   // 教研组长用户ID
	Description string     `json:"description"`    // 教研组描述
	Settings    string     `json:"settings"`       // 组级设置JSON
	Status      string     `json:"status"`         // 状态：active / disabled
	CreatedAt   *time.Time `json:"created_at"`     // 创建时间
	UpdatedAt   *time.Time `json:"updated_at"`     // 更新时间
}

// ==================== 教研组成员模型（对应 teaching_group_members 表） ====================

// TeachingGroupMember 教研组成员关联
// 一个用户可以属于多个教研组
type TeachingGroupMember struct {
	ID       string     `json:"id"`         // UUID主键
	GroupID  string     `json:"group_id"`   // 教研组ID → teaching_groups.id
	UserID   string     `json:"user_id"`    // 用户ID → users.id
	Role     string     `json:"role"`       // 成员角色：member=普通成员 / backbone=骨干教师
	JoinedAt *time.Time `json:"joined_at"`  // 加入时间
}

// ==================== 组织类型常量 ====================

const (
	OrgTypeRegion = "region" // 区域
	OrgTypeSchool = "school" // 学校
)

// ValidOrgTypes 有效的组织类型列表
var ValidOrgTypes = []string{OrgTypeRegion, OrgTypeSchool}

// IsValidOrgType 检查组织类型是否有效
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
)

// ValidGroupMemberRoles 有效的教研组成员角色列表
var ValidGroupMemberRoles = []string{GroupMemberRoleMember, GroupMemberRoleBackbone}

// IsValidGroupMemberRole 检查教研组成员角色是否有效
func IsValidGroupMemberRole(role string) bool {
	for _, v := range ValidGroupMemberRoles {
		if v == role {
			return true
		}
	}
	return false
}

// ==================== 请求结构体 ====================

// CreateOrganizationRequest 创建组织请求
type CreateOrganizationRequest struct {
	Name        string  `json:"name"`           // 组织名称（必填）
	Type        string  `json:"type"`           // 组织类型（必填：region/school）
	ParentID    *string `json:"parent_id"`      // 父级组织ID（学校必填，指向区域）
	AdminUserID *string `json:"admin_user_id"`  // 管理员用户ID（可选）
}

// UpdateOrganizationRequest 更新组织请求
type UpdateOrganizationRequest struct {
	Name        string  `json:"name"`           // 组织名称（必填）
	AdminUserID *string `json:"admin_user_id"`  // 管理员用户ID（可选）
	Settings    string  `json:"settings"`       // 组织级设置JSON（可选）
	Status      string  `json:"status"`         // 状态（可选：active/disabled）
}

// CreateTeachingGroupRequest 创建教研组请求
type CreateTeachingGroupRequest struct {
	Name        string  `json:"name"`           // 教研组名称（必填）
	SchoolID    string  `json:"school_id"`      // 所属学校ID（必填）
	Subject     string  `json:"subject"`        // 学科（必填）
	GradeRange  string  `json:"grade_range"`    // 学段范围（可选）
	LeadUserID  *string `json:"lead_user_id"`   // 教研组长ID（可选）
	Description string  `json:"description"`    // 描述（可选）
}

// UpdateTeachingGroupRequest 更新教研组请求
type UpdateTeachingGroupRequest struct {
	Name        string  `json:"name"`           // 教研组名称（必填）
	Subject     string  `json:"subject"`        // 学科（必填）
	GradeRange  string  `json:"grade_range"`    // 学段范围（可选）
	LeadUserID  *string `json:"lead_user_id"`   // 教研组长ID（可选）
	Description string  `json:"description"`    // 描述（可选）
	Settings    string  `json:"settings"`       // 组级设置JSON（可选）
	Status      string  `json:"status"`         // 状态（可选：active/disabled）
}

// AddGroupMemberRequest 添加教研组成员请求
type AddGroupMemberRequest struct {
	UserID string `json:"user_id"` // 用户ID（必填）
	Role   string `json:"role"`    // 成员角色（可选，默认member）
}

// ==================== 响应结构体 ====================

// OrganizationListResponse 组织列表响应
type OrganizationListResponse struct {
	Organizations []*OrganizationListItem `json:"organizations"` // 组织列表
	Total         int                     `json:"total"`         // 总数
}

// OrganizationListItem 组织列表单条
type OrganizationListItem struct {
	ID            string     `json:"id"`              // UUID
	Name          string     `json:"name"`            // 组织名称
	Type          string     `json:"type"`            // 组织类型
	ParentID      *string    `json:"parent_id"`       // 父级ID
	ParentName    string     `json:"parent_name"`     // 父级名称（前端展示用）
	AdminUserID   *string    `json:"admin_user_id"`   // 管理员ID
	AdminUserName string     `json:"admin_user_name"` // 管理员名称（前端展示用）
	Status        string     `json:"status"`          // 状态
	GroupCount    int        `json:"group_count"`     // 下属教研组数量
	MemberCount   int        `json:"member_count"`    // 下属成员总数
	CreatedAt     *time.Time `json:"created_at"`      // 创建时间
}

// TeachingGroupListResponse 教研组列表响应
type TeachingGroupListResponse struct {
	Groups []*TeachingGroupListItem `json:"groups"` // 教研组列表
	Total  int                      `json:"total"`  // 总数
}

// TeachingGroupListItem 教研组列表单条
type TeachingGroupListItem struct {
	ID           string     `json:"id"`              // UUID
	Name         string     `json:"name"`            // 教研组名称
	SchoolID     string     `json:"school_id"`       // 所属学校ID
	SchoolName   string     `json:"school_name"`     // 所属学校名称
	Subject      string     `json:"subject"`         // 学科
	GradeRange   string     `json:"grade_range"`     // 学段范围
	LeadUserID   *string    `json:"lead_user_id"`    // 组长ID
	LeadUserName string     `json:"lead_user_name"`  // 组长名称
	MemberCount  int        `json:"member_count"`    // 成员数量
	Status       string     `json:"status"`          // 状态
	CreatedAt    *time.Time `json:"created_at"`      // 创建时间
}

// TeachingGroupDetailResponse 教研组详情响应（含成员列表）
type TeachingGroupDetailResponse struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	SchoolID    string                `json:"school_id"`
	SchoolName  string                `json:"school_name"`
	Subject     string                `json:"subject"`
	GradeRange  string                `json:"grade_range"`
	LeadUserID  *string               `json:"lead_user_id"`
	LeadUserName string               `json:"lead_user_name"`
	Description string                `json:"description"`
	Settings    string                `json:"settings"`
	Status      string                `json:"status"`
	Members     []*GroupMemberItem    `json:"members"`     // 成员列表
	CreatedAt   *time.Time            `json:"created_at"`
	UpdatedAt   *time.Time            `json:"updated_at"`
}

// GroupMemberItem 教研组成员列表单条
type GroupMemberItem struct {
	ID          string     `json:"id"`           // 关联记录ID
	UserID      string     `json:"user_id"`      // 用户ID
	Username    string     `json:"username"`     // 用户名
	DisplayName string     `json:"display_name"` // 显示名称
	Role        string     `json:"role"`         // 成员角色：member/backbone
	JoinedAt    *time.Time `json:"joined_at"`    // 加入时间
}
