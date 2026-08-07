package models

import (
	"encoding/json"
	"time"
)

// ==================== 数据库模型 ====================

// User 用户模型，对应数据库 users 表
// v64(迭代3)新增：TeachingProfile 字段（JSONB，教学风格前测结果）
// 超管收口新增：IsSuper 字段（is_super 列，超级管理员标记位）
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	PasswordHash string `json:"-"`
	// Role：admin / region_admin / district_inspector / senior_operator / operator / viewer
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	LoginCount  int        `json:"login_count"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	// v64新增：教学风格前测结果（JSONB，可能为NULL）
	TeachingProfileJSON *string `json:"-"`
	// 超管收口新增：超级管理员标记位（is_super 列，默认 false）
	//   - true：真超管，可访问模型配置/积分/AI统计/审计日志等敏感入口
	//   - false：二线管理员（仍是 admin 角色），只能管基础数据/用户/组织架构
	IsSuper bool `json:"is_super"`
}

// ==================== 门户板块常量（v172新增：组织级板块可见性开关） ====================

const (
	PortalModuleLessonPlan = "lesson_plan"
	PortalModuleCourseware = "courseware"
	PortalModuleWorkflow   = "workflow"
)

var AllPortalModules = []string{
	PortalModuleLessonPlan,
	PortalModuleCourseware,
	PortalModuleWorkflow,
}

// DefaultPortalModules 返回全部板块开启的默认配置。
func DefaultPortalModules() map[string]bool {
	return map[string]bool{
		PortalModuleLessonPlan: true,
		PortalModuleCourseware: true,
		PortalModuleWorkflow:   true,
	}
}

// EducationDomainNotReadyMessage 是教育域异常时统一下发给前端的提示。
//
// 具体数据库错误、空值或跨域冲突只写服务端日志，不直接暴露给用户，
// 防止通过错误差异探测组织任命数据。
const EducationDomainNotReadyMessage = "教育域尚未正确配置，请联系管理员"

// UserInfo 返回给前端的用户信息（不含敏感字段）。
//
// 教育域字段：
//   - EducationDomain：已解析出的具体教育域；异常时为空；
//   - EducationOrgID：当前教学或任命组织；异常时为空；
//   - EducationDomainReady：是否已安全解析出可用于业务授权的教育域；
//   - EducationDomainError：异常时的统一用户提示；
//   - EducationProfile：当前教育域页面语义与能力开关。
type UserInfo struct {
	ID                   string           `json:"id"`
	Username             string           `json:"username"`
	DisplayName          string           `json:"display_name"`
	Role                 string           `json:"role"`
	Status               string           `json:"status"`
	LastLoginAt          *time.Time       `json:"last_login_at"`
	LoginCount           int              `json:"login_count"`
	CreatedAt            *time.Time       `json:"created_at"`
	UpdatedAt            *time.Time       `json:"updated_at"`
	HasTeachingProfile   bool             `json:"has_teaching_profile"`
	OrgLogoURL           string           `json:"org_logo_url"`
	OrgName              string           `json:"org_name"`
	PortalModules        map[string]bool  `json:"portal_modules"`
	EducationDomain      string           `json:"education_domain"`
	EducationOrgID       string           `json:"education_org_id"`
	EducationDomainReady bool             `json:"education_domain_ready"`
	EducationDomainError string           `json:"education_domain_error"`
	EducationProfile     EducationProfile `json:"education_profile"`
	IsSuper              bool             `json:"is_super"`
}

// ToUserInfo 将User转换为UserInfo。
//
// 初始值只用于数据库教育域解析完成前：
//   - admin、district_inspector固定mixed并标记ready；
//   - region_admin必须等待任命解析，初始即fail-closed；
//   - 其它存量教学角色保持K12兼容初值，随后由AuthService覆盖真实组织域。
func (u *User) ToUserInfo() *UserInfo {
	defaultDomain := EducationDomainK12
	profileDomain := EducationDomainK12
	domainReady := true
	domainError := ""

	switch u.Role {
	case RoleAdmin, RoleDistrictInspector:
		defaultDomain = EducationDomainMixed
		profileDomain = EducationDomainMixed

	case RoleRegionAdmin:
		// 区域管理员不能在数据库解析前假设为mixed或K12。
		defaultDomain = ""
		profileDomain = EducationDomainMixed
		domainReady = false
		domainError = EducationDomainNotReadyMessage
	}

	return &UserInfo{
		ID:                   u.ID,
		Username:             u.Username,
		DisplayName:          u.DisplayName,
		Role:                 u.Role,
		Status:               u.Status,
		LastLoginAt:          u.LastLoginAt,
		LoginCount:           u.LoginCount,
		CreatedAt:            u.CreatedAt,
		UpdatedAt:            u.UpdatedAt,
		HasTeachingProfile:   u.TeachingProfileJSON != nil,
		PortalModules:        DefaultPortalModules(),
		EducationDomain:      defaultDomain,
		EducationDomainReady: domainReady,
		EducationDomainError: domainError,
		EducationProfile:     EducationProfileForDomain(profileDomain),
		IsSuper:              u.IsSuper,
	}
}

// GetTeachingProfile 解析TeachingProfile JSON为结构体。
func (u *User) GetTeachingProfile() *TeachingProfile {
	if u.TeachingProfileJSON == nil {
		return nil
	}
	var profile TeachingProfile
	if err := json.Unmarshal([]byte(*u.TeachingProfileJSON), &profile); err != nil {
		return nil
	}
	return &profile
}

// ==================== 认证相关 ====================

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string    `json:"token"`
	User  *UserInfo `json:"user"`
}

// ==================== 用户管理请求/响应 ====================

type CreateUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        string `json:"role"`

	// SchoolID 是系统管理员创建教学账号时选择的所属学校。
	//
	// 该字段只作为“建号请求输入”，不会写入 users 表；正式学校归属仍以
	// school_members 为唯一事实源。学校管理员创建账号时，Handler 会使用
	// 其真实管理学校覆盖本字段，防止客户端伪造学校扩大权限。
	SchoolID string `json:"school_id"`
}

type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

type UpdateAssignmentsRequest struct {
	CourseCodes []string `json:"course_codes"`
}

type CourseAssignment struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	CourseCode string     `json:"course_code"`
	AssignedBy string     `json:"assigned_by"`
	AssignedAt *time.Time `json:"assigned_at"`
}

type UserListResponse struct {
	Users []*UserInfo `json:"users"`
	Total int         `json:"total"`
}

// ==================== 角色与状态常量 ====================

const (
	RoleAdmin             = "admin"
	RoleRegionAdmin       = "region_admin"
	RoleDistrictInspector = "district_inspector"
	RoleSeniorOperator    = "senior_operator"
	RoleOperator          = "operator"
	RoleViewer            = "viewer"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

var ValidRoles = []string{
	RoleAdmin,
	RoleRegionAdmin,
	RoleDistrictInspector,
	RoleSeniorOperator,
	RoleOperator,
	RoleViewer,
}

var ValidStatuses = []string{
	StatusActive,
	StatusDisabled,
}

func IsValidRole(role string) bool {
	for _, item := range ValidRoles {
		if item == role {
			return true
		}
	}
	return false
}

func IsValidStatus(status string) bool {
	for _, item := range ValidStatuses {
		if item == status {
			return true
		}
	}
	return false
}

// SchoolAdminCreatableRoles 学校管理员可创建的角色。
var SchoolAdminCreatableRoles = []string{
	RoleOperator,
	RoleViewer,
}

func IsSchoolAdminCreatableRole(role string) bool {
	for _, item := range SchoolAdminCreatableRoles {
		if item == role {
			return true
		}
	}
	return false
}

// ==================== 任命制身份 ====================

var AppointmentOnlyRoles = []string{
	RoleRegionAdmin,
	RoleSeniorOperator,
}

func IsAppointmentOnlyRole(role string) bool {
	for _, item := range AppointmentOnlyRoles {
		if item == role {
			return true
		}
	}
	return false
}
