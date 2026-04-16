package models

import (
	"encoding/json"
	"time"
)

// ==================== 数据库模型 ====================

// User 用户模型，对应数据库 users 表
// v64(迭代3)新增：TeachingProfile 字段（JSONB，教学风格前测结果）
// v110新增：SchoolID 字段（所属学校，方案A）
type User struct {
	ID           string     `json:"id"`           // UUID 主键
	Username     string     `json:"username"`     // 用户名（唯一）
	DisplayName  string     `json:"display_name"` // 显示名称
	PasswordHash string     `json:"-"`            // 密码哈希（不输出到JSON）
	// Role 用户角色：admin / senior_operator / operator / viewer
	Role        string     `json:"role"`
	Status      string     `json:"status"`        // 状态：active / disabled
	LastLoginAt *time.Time `json:"last_login_at"` // 最后登录时间
	LoginCount  int        `json:"login_count"`   // 登录次数
	CreatedAt   *time.Time `json:"created_at"`    // 创建时间
	UpdatedAt   *time.Time `json:"updated_at"`    // 更新时间
	// v64新增：教学风格前测结果（JSONB，可能为NULL）
	TeachingProfileJSON *string `json:"-"` // 原始JSON字符串（不直接输出）
	// v110新增：所属学校ID（可为空，向后兼容）
	SchoolID *string `json:"school_id,omitempty"`
}

// UserInfo 返回给前端的用户信息（不含敏感字段）
type UserInfo struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	DisplayName        string     `json:"display_name"`
	Role               string     `json:"role"`
	Status             string     `json:"status"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	LoginCount         int        `json:"login_count"`
	CreatedAt          *time.Time `json:"created_at"`
	UpdatedAt          *time.Time `json:"updated_at"`
	HasTeachingProfile bool       `json:"has_teaching_profile"` // 是否已完成前测
	SchoolID           *string    `json:"school_id,omitempty"`  // v110新增
}

// ToUserInfo 将 User 转换为 UserInfo（过滤敏感信息）
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
		SchoolID:           u.SchoolID,
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

// ==================== 认证相关请求/响应 ====================

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
// v110新增：SchoolID 可选（学校管理员创建用户时写入本校ID）
type CreateUserRequest struct {
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Password    string  `json:"password"`
	Role        string  `json:"role"`
	SchoolID    *string `json:"school_id,omitempty"`
}

// UpdateUserRequest 编辑用户请求体
// v110新增：SchoolID 可选（系统管理员可用于纠偏；学校管理员接口不暴露修改能力）
type UpdateUserRequest struct {
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	SchoolID    *string `json:"school_id,omitempty"`
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
	RoleAdmin          = "admin"
	RoleSeniorOperator = "senior_operator" // 学校管理员
	RoleOperator       = "operator"        // 骨干教师
	RoleViewer         = "viewer"          // 普通教师
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

var ValidRoles = []string{RoleAdmin, RoleSeniorOperator, RoleOperator, RoleViewer}
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

// SchoolAdminCreatableRoles 学校管理员可创建的角色（仅比自己低级）
var SchoolAdminCreatableRoles = []string{
	RoleOperator,
	RoleViewer,
}

// IsSchoolAdminCreatableRole 校验学校管理员可创建角色
func IsSchoolAdminCreatableRole(role string) bool {
	for _, r := range SchoolAdminCreatableRoles {
		if r == role {
			return true
		}
	}
	return false
}
