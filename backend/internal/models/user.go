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
	//   - false：二线管理员（仍是 admin 角色），只能管基础数据/用户/组织架构，
	//            碰不到上述敏感入口（前端隐藏 + 后端 superAdminOnly 中间件双重收口）
	// 注意：is_super 与 role 是正交的两个维度——is_super 只在 role=admin 时才有意义，
	//       它把"全能 admin"细分为"超管 admin(true)"与"二线 admin(false)"。
	IsSuper bool `json:"is_super"`
}

// ==================== 门户板块常量（v172新增：组织级板块可见性开关） ====================
//
// 门户(PortalPage)三大并行板块的 key，与前端 entries 的 key 一一对应。
// 通过组织 organizations.settings 里的 portal_modules 配置控制各组织能看到哪些板块。
// 例：{"portal_modules":{"lesson_plan":true,"courseware":true,"workflow":false}}
const (
	PortalModuleLessonPlan = "lesson_plan" // 备课工坊
	PortalModuleCourseware = "courseware"  // 课件工坊
	PortalModuleWorkflow   = "workflow"    // 课件审核（即 /workflow，Pipeline 审核系统）
)

// AllPortalModules 所有门户板块 key 列表
var AllPortalModules = []string{
	PortalModuleLessonPlan,
	PortalModuleCourseware,
	PortalModuleWorkflow,
}

// DefaultPortalModules 返回"全部板块开启"的默认配置
// 用途：没有任何配置 / 解析失败 / admin 时，一律全开（维持现状，不波及存量用户）
func DefaultPortalModules() map[string]bool {
	return map[string]bool{
		PortalModuleLessonPlan: true,
		PortalModuleCourseware: true,
		PortalModuleWorkflow:   true,
	}
}

// UserInfo 返回给前端的用户信息（不含敏感字段）
type UserInfo struct {
	ID                 string          `json:"id"`
	Username           string          `json:"username"`
	DisplayName        string          `json:"display_name"`
	Role               string          `json:"role"`
	Status             string          `json:"status"`
	LastLoginAt        *time.Time      `json:"last_login_at"`
	LoginCount         int             `json:"login_count"`
	CreatedAt          *time.Time      `json:"created_at"`
	UpdatedAt          *time.Time      `json:"updated_at"`
	HasTeachingProfile bool            `json:"has_teaching_profile"`
	OrgLogoURL         string          `json:"org_logo_url"`   // 用户所属组织的Logo URL（学校>区域>空）
	OrgName            string          `json:"org_name"`       // 用户所属组织名称
	PortalModules      map[string]bool `json:"portal_modules"` // v172新增：门户板块可见性（三个key各true/false）
	// 超管收口新增：超级管理员标记位，透传给前端用于敏感入口显隐
	// 前端据此隐藏"模型配置/积分/AI统计/审计日志"入口（二线 admin 拿到 false）
	IsSuper bool `json:"is_super"`
}

// ToUserInfo 将 User 转换为 UserInfo
func (u *User) ToUserInfo() *UserInfo {
	return &UserInfo{
		ID:                 u.ID,
		Username:           u.Username,
		DisplayName:        u.DisplayName,
		Role:               u.Role,
		Status:             u.Status,
		LastLoginAt:        u.LastLoginAt,
		LoginCount:         u.LoginCount,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
		HasTeachingProfile: u.TeachingProfileJSON != nil,
		// PortalModules 默认全开，由 auth_service 填充时根据组织配置/角色覆盖
		PortalModules: DefaultPortalModules(),
		// 超管标记位原样透传（数据库真值，前端据此收口敏感入口）
		IsSuper: u.IsSuper,
	}
}

// GetTeachingProfile 解析 TeachingProfile JSON 为结构体
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

// CreateUserRequest 创建用户请求体
type CreateUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

// UpdateUserRequest 编辑用户请求体
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
	RoleRegionAdmin       = "region_admin"       // 迭代一(P2-06)新增：区域管理员（管人+积分，管到学校管理员一级）
	RoleDistrictInspector = "district_inspector" // v127新增：区域教研员（管抽查），与区域管理员并存互不替代
	RoleSeniorOperator    = "senior_operator"    // 学校管理员
	RoleOperator          = "operator"           // 骨干教师
	RoleViewer            = "viewer"             // 普通教师
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

// ValidRoles 有效角色列表
// v127新增 district_inspector；迭代一新增 region_admin
// 顺序约定：admin > region_admin > district_inspector > senior_operator > operator > viewer
var ValidRoles = []string{
	RoleAdmin,
	RoleRegionAdmin,
	RoleDistrictInspector,
	RoleSeniorOperator,
	RoleOperator,
	RoleViewer,
}
var ValidStatuses = []string{StatusActive, StatusDisabled}

func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// SchoolAdminCreatableRoles 学校管理员可创建的角色（低于自身级别）
var SchoolAdminCreatableRoles = []string{RoleOperator, RoleViewer}

// IsSchoolAdminCreatableRole 校验学校管理员可创建的角色
func IsSchoolAdminCreatableRole(role string) bool {
	for _, r := range SchoolAdminCreatableRoles {
		if r == role {
			return true
		}
	}
	return false
}

// ==================== 任命制身份（归属治理批C）====================
//
// 不变式：users.role 出现下列身份 ⇔ organization_admins（或 organizations.admin_user_id）
// 存在该用户的对应任命。这两个身份是"任命的影子"，不允许独立存在：
//   - 获得：仅可经「组织架构→🛡️管理员」任命（B13 任命自动升级身份）；
//   - 失去：末个任命被移除时自动降级为骨干教师（批C，organization_admin_service）；
//   - 建号/编辑用户不得直接授予或改动（user_service 校验拒绝并给出任命引导）。

// AppointmentOnlyRoles 仅可经组织任命获得的管理身份
var AppointmentOnlyRoles = []string{RoleRegionAdmin, RoleSeniorOperator}

// IsAppointmentOnlyRole 判断某角色是否为任命制身份
func IsAppointmentOnlyRole(role string) bool {
	for _, r := range AppointmentOnlyRoles {
		if r == role {
			return true
		}
	}
	return false
}
