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
	LogoURL     string     `json:"logo_url"` // 组织Logo URL
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
	LeadUserID  *string    `json:"lead_user_id"` // 兼容保留，实际组长通过成员角色管理
	Description string     `json:"description"`
	Settings    string     `json:"settings"`
	LogoURL     string     `json:"logo_url"` // 组织Logo URL
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
	Role     string     `json:"role"` // member=普通成员 / backbone=骨干教师 / lead=教研组长
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

// UpdateOrganizationRequest 更新组织请求
//
// 账户与权限修复批（Logo 移除链路①）新增 ClearLogo 字段：
//   - 背景：编辑弹窗"移除Logo"此前只清前端本地 state，请求体没有任何 logo 字段，
//     后端无从得知"用户想删掉 Logo"，移除从不落库（假移除）。
//   - 语义：clear_logo=true → organization_service.UpdateOrganization 在常规字段
//     更新成功后调用 repository.UpdateOrganizationLogo(id, "") 清空 logo_url；
//     缺省 false → 完全不动 logo_url。Logo 上传仍走独立上传接口，与本字段互不干扰。
//   - handler 为整体 JSON 绑定后透传 service，无需任何改动（向后兼容）。
//
// B10 部分更新语义（见 organization_service.UpdateOrganization）：
//   - Settings 为空串 → 本次不修改，service 回填库中现值（真要清空须显式传 "{}"）
//   - Status   为空串 → 本次不修改，service 回填库中现值
type UpdateOrganizationRequest struct {
	Name        string  `json:"name"`
	AdminUserID *string `json:"admin_user_id"`
	Settings    string  `json:"settings"`
	Status      string  `json:"status"`
	ClearLogo   bool    `json:"clear_logo"` // true=清空组织Logo（假移除根治）
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
	LogoURL       string     `json:"logo_url"`
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

// ==================== 迭代一 新增：组织管理员模型（对应 organization_admins 表，P2-05） ====================
//
// organization_admins 是"组织全部管理员"的权威来源（迭代一新增）。
// 一个组织可有多个管理员；与 organizations.admin_user_id 单字段并存
// （旧字段保留作"主管理员"兼容，本表作"全部管理员"权威来源）。

// 组织管理员类型常量
const (
	OrgAdminRoleRegion = "region_admin" // 区域管理员（org 须为 region 类型）
	OrgAdminRoleSchool = "school_admin" // 学校管理员（org 须为 school 类型）
)

// ValidOrgAdminRoleTypes 有效的组织管理员类型列表
var ValidOrgAdminRoleTypes = []string{OrgAdminRoleRegion, OrgAdminRoleSchool}

// IsValidOrgAdminRoleType 校验组织管理员类型是否合法
func IsValidOrgAdminRoleType(t string) bool {
	for _, v := range ValidOrgAdminRoleTypes {
		if v == t {
			return true
		}
	}
	return false
}

// OrganizationAdmin 组织管理员关联实体（对应 organization_admins 表）
type OrganizationAdmin struct {
	OrgID     string     `json:"org_id"`
	UserID    string     `json:"user_id"`
	RoleType  string     `json:"role_type"`  // region_admin / school_admin
	CreatedBy *string    `json:"created_by"` // 任命人（可空）
	CreatedAt *time.Time `json:"created_at"`
}

// OrganizationAdminItem 组织管理员列表单条（含用户名、显示名，供前端展示）
// 对应 repository.ListOrgAdmins 的返回结构：
//   CreatedBy 用 COALESCE(::text,'') 转为 string（空串表示无任命人/迁移数据）
type OrganizationAdminItem struct {
	OrgID       string     `json:"org_id"`
	UserID      string     `json:"user_id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	RoleType    string     `json:"role_type"`  // region_admin / school_admin
	CreatedBy   string     `json:"created_by"` // 任命人ID（空串表示无）
	CreatedAt   *time.Time `json:"created_at"`
}

// ==================== 发布目标组（供模板发布等场景复用） ====================
//
// 说明：PublishTargetGroup 历史上定义在 courseware_component.go（课件模板发布用）。
// organization_repo.go 的 ListMyLeadOrBackboneGroups 也返回此类型。
// 此处不重复定义，避免重复声明编译错误（仅作注释标注其归属）。
