package models

import (
	"time"
)

// ==================== 组织模型（对应 organizations 表） ====================

// Organization 组织模型（区域/学校）。
//
// EducationDomain 是组织创建时确定的教育域：
//   - region：只能是 mixed；
//   - school：只能是 k12 / vocational / adult。
//
// 创建学校时必须由调用者显式选择具体教学域；创建区域时由后端强制写 mixed。
// 组织创建成功后，普通业务不能修改 EducationDomain。
type Organization struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	EducationDomain string     `json:"education_domain"`
	ParentID        *string    `json:"parent_id"`
	AdminUserID     *string    `json:"admin_user_id"`
	Settings        string     `json:"settings"`
	LogoURL         string     `json:"logo_url"` // 组织Logo URL
	Status          string     `json:"status"`
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
}

// ==================== 教研组模型（对应 teaching_groups 表） ====================

// TeachingGroup 教研组模型。
// v109改动：多组长支持——组长通过 teaching_group_members.role='lead' 管理。
// lead_user_id 字段保留兼容性，不再作为主要组长标识。
type TeachingGroup struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	SchoolID    string     `json:"school_id"`
	Subject     string     `json:"subject"`
	GradeRange  string     `json:"grade_range"`
	LeadUserID  *string    `json:"lead_user_id"` // 兼容保留，实际组长通过成员角色管理
	Description string     `json:"description"`
	Settings    string     `json:"settings"`
	LogoURL     string     `json:"logo_url"` // 组织Logo URL
	Status      string     `json:"status"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// ==================== 教研组成员模型 ====================

// TeachingGroupMember 教研组成员关联。
type TeachingGroupMember struct {
	ID       string     `json:"id"`
	GroupID  string     `json:"group_id"`
	UserID   string     `json:"user_id"`
	Role     string     `json:"role"` // member=普通成员 / backbone=骨干教师 / lead=教研组长
	JoinedAt *time.Time `json:"joined_at"`
}

// ==================== 组织类型常量 ====================

const (
	OrgTypeRegion = "region"
	OrgTypeSchool = "school"
)

var ValidOrgTypes = []string{
	OrgTypeRegion,
	OrgTypeSchool,
}

// IsValidOrgType 判断组织类型是否合法。
func IsValidOrgType(t string) bool {
	for _, value := range ValidOrgTypes {
		if value == t {
			return true
		}
	}
	return false
}

// ==================== 教研组成员角色常量 ====================

const (
	GroupMemberRoleMember   = "member"
	GroupMemberRoleBackbone = "backbone"
	GroupMemberRoleLead     = "lead"
)

var ValidGroupMemberRoles = []string{
	GroupMemberRoleMember,
	GroupMemberRoleBackbone,
	GroupMemberRoleLead,
}

// IsValidGroupMemberRole 判断教研组成员角色是否合法。
func IsValidGroupMemberRole(role string) bool {
	for _, value := range ValidGroupMemberRoles {
		if value == role {
			return true
		}
	}
	return false
}

// ==================== 请求结构体 ====================

// CreateOrganizationRequest 创建区域或学校请求。
//
// EducationDomain：
//   - 创建学校时必填，只允许 k12 / vocational / adult；
//   - 创建区域时客户端值会被忽略，后端始终写 mixed；
//   - 创建成功后不能通过普通业务接口修改。
type CreateOrganizationRequest struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	EducationDomain string  `json:"education_domain"`
	ParentID        *string `json:"parent_id"`
	AdminUserID     *string `json:"admin_user_id"`
}

// UpdateOrganizationRequest 更新组织请求。
//
// ClearLogo：
//   - true：清空组织Logo；
//   - false：不修改组织Logo。
//
// 部分更新语义：
//   - Settings 为空串：本次不修改；
//   - Status 为空串：本次不修改。
//
// 本请求刻意不包含 EducationDomain。
// 即使客户端伪造 education_domain 字段，普通组织更新也不会读取或写入该字段。
type UpdateOrganizationRequest struct {
	Name        string  `json:"name"`
	AdminUserID *string `json:"admin_user_id"`
	Settings    string  `json:"settings"`
	Status      string  `json:"status"`
	ClearLogo   bool    `json:"clear_logo"`
}

// CreateTeachingGroupRequest 创建教研组请求。
// v109改动：移除 LeadUserID，组长通过成员管理设置。
type CreateTeachingGroupRequest struct {
	Name        string `json:"name"`
	SchoolID    string `json:"school_id"`
	Subject     string `json:"subject"`
	GradeRange  string `json:"grade_range"`
	Description string `json:"description"`
}

// UpdateTeachingGroupRequest 更新教研组请求。
type UpdateTeachingGroupRequest struct {
	Name        string `json:"name"`
	Subject     string `json:"subject"`
	GradeRange  string `json:"grade_range"`
	Description string `json:"description"`
	Settings    string `json:"settings"`
	Status      string `json:"status"`
}

// AddGroupMemberRequest 添加教研组成员请求。
type AddGroupMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// ==================== 响应结构体 ====================

// OrganizationListResponse 组织列表响应。
type OrganizationListResponse struct {
	Organizations []*OrganizationListItem `json:"organizations"`
	Total         int                     `json:"total"`
}

// OrganizationListItem 组织列表单条。
type OrganizationListItem struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	EducationDomain string     `json:"education_domain"`
	ParentID        *string    `json:"parent_id"`
	ParentName      string     `json:"parent_name"`
	AdminUserID     *string    `json:"admin_user_id"`
	AdminUserName   string     `json:"admin_user_name"`
	Status          string     `json:"status"`
	LogoURL         string     `json:"logo_url"`
	GroupCount      int        `json:"group_count"`
	MemberCount     int        `json:"member_count"`
	CreatedAt       *time.Time `json:"created_at"`
}

// TeachingGroupListResponse 教研组列表响应。
type TeachingGroupListResponse struct {
	Groups []*TeachingGroupListItem `json:"groups"`
	Total  int                      `json:"total"`
}

// TeachingGroupListItem 教研组列表单条。
type TeachingGroupListItem struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	SchoolID      string     `json:"school_id"`
	SchoolName    string     `json:"school_name"`
	Subject       string     `json:"subject"`
	GradeRange    string     `json:"grade_range"`
	LeadUserID    *string    `json:"lead_user_id"`
	LeadUserName  string     `json:"lead_user_name"`
	LeadUserNames string     `json:"lead_user_names"`
	MemberCount   int        `json:"member_count"`
	Status        string     `json:"status"`
	CreatedAt     *time.Time `json:"created_at"`
}

// TeachingGroupDetailResponse 教研组详情响应。
type TeachingGroupDetailResponse struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	SchoolID      string             `json:"school_id"`
	SchoolName    string             `json:"school_name"`
	Subject       string             `json:"subject"`
	GradeRange    string             `json:"grade_range"`
	LeadUserID    *string            `json:"lead_user_id"`
	LeadUserName  string             `json:"lead_user_name"`
	LeadUserNames string             `json:"lead_user_names"`
	Description   string             `json:"description"`
	Settings      string             `json:"settings"`
	Status        string             `json:"status"`
	Members       []*GroupMemberItem `json:"members"`
	CreatedAt     *time.Time         `json:"created_at"`
	UpdatedAt     *time.Time         `json:"updated_at"`
}

// GroupMemberItem 教研组成员列表单条。
type GroupMemberItem struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	JoinedAt    *time.Time `json:"joined_at"`
}

// ==================== 组织管理员模型 ====================

const (
	OrgAdminRoleRegion = "region_admin"
	OrgAdminRoleSchool = "school_admin"
)

var ValidOrgAdminRoleTypes = []string{
	OrgAdminRoleRegion,
	OrgAdminRoleSchool,
}

// IsValidOrgAdminRoleType 判断组织管理员类型是否合法。
func IsValidOrgAdminRoleType(roleType string) bool {
	for _, value := range ValidOrgAdminRoleTypes {
		if value == roleType {
			return true
		}
	}
	return false
}

// OrganizationAdmin 组织管理员关联实体。
type OrganizationAdmin struct {
	OrgID           string     `json:"org_id"`
	UserID          string     `json:"user_id"`
	RoleType        string     `json:"role_type"`
	EducationDomain string     `json:"education_domain"`
	CreatedBy       *string    `json:"created_by"`
	CreatedAt       *time.Time `json:"created_at"`
}

// OrganizationAdminItem 组织管理员列表单条。
type OrganizationAdminItem struct {
	OrgID           string     `json:"org_id"`
	UserID          string     `json:"user_id"`
	Username        string     `json:"username"`
	DisplayName     string     `json:"display_name"`
	RoleType        string     `json:"role_type"`
	EducationDomain string     `json:"education_domain"`
	CreatedBy       string     `json:"created_by"`
	CreatedAt       *time.Time `json:"created_at"`
}
