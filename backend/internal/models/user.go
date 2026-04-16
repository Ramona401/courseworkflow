package models

import (
	"encoding/json"
	"time"
)

// ==================== 数据库模型 ====================

// User 用户模型，对应数据库 users 表
// v64(迭代3)新增：TeachingProfile 字段（JSONB，教学风格前测结果）
type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	DisplayName  string     `json:"display_name"`
	PasswordHash string     `json:"-"`
	// Role：admin / senior_operator / operator / viewer
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	LoginCount  int        `json:"login_count"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	// v64新增：教学风格前测结果（JSONB，可能为NULL）
	TeachingProfileJSON *string `json:"-"`
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
	HasTeachingProfile bool       `json:"has_teaching_profile"`
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
	RoleAdmin          = "admin"
	RoleSeniorOperator = "senior_operator" // 学校管理员
	RoleOperator       = "operator"        // 骨干教师
	RoleViewer         = "viewer"          // 普通教师
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

var ValidRoles    = []string{RoleAdmin, RoleSeniorOperator, RoleOperator, RoleViewer}
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
