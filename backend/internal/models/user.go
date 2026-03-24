package models

import (
	"time"
)

// ==================== 数据库模型 ====================

// User 用户模型，对应数据库 users 表
type User struct {
	ID           string     `json:"id"`            // UUID 主键
	Username     string     `json:"username"`       // 用户名（唯一）
	DisplayName  string     `json:"display_name"`   // 显示名称
	PasswordHash string     `json:"-"`              // 密码哈希（不输出到JSON）
	Role         string     `json:"role"`           // 角色：admin / operator / viewer
	Status       string     `json:"status"`         // 状态：active / disabled
	LastLoginAt  *time.Time `json:"last_login_at"`  // 最后登录时间
	LoginCount   int        `json:"login_count"`    // 登录次数
	CreatedAt    *time.Time `json:"created_at"`     // 创建时间
	UpdatedAt    *time.Time `json:"updated_at"`     // 更新时间
}

// UserInfo 返回给前端的用户信息（不含敏感字段）
type UserInfo struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	LoginCount  int        `json:"login_count"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// ToUserInfo 将 User 转换为 UserInfo（过滤敏感信息）
func (u *User) ToUserInfo() *UserInfo {
	return &UserInfo{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		Status:      u.Status,
		LastLoginAt: u.LastLoginAt,
		LoginCount:  u.LoginCount,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// ==================== 认证相关请求/响应 ====================

// LoginRequest 登录请求体
type LoginRequest struct {
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码（明文，服务端验证）
}

// LoginResponse 登录成功响应
type LoginResponse struct {
	Token string    `json:"token"` // JWT access token
	User  *UserInfo `json:"user"`  // 用户信息
}

// ==================== 用户管理请求/响应（P1-4新增） ====================

// CreateUserRequest 创建用户请求体（仅admin可调用）
type CreateUserRequest struct {
	Username    string `json:"username"`     // 用户名（必填，唯一）
	DisplayName string `json:"display_name"` // 显示名称（必填）
	Password    string `json:"password"`     // 初始密码（必填，>=6位）
	Role        string `json:"role"`         // 角色（必填：admin/operator/viewer）
}

// UpdateUserRequest 编辑用户请求体（仅admin可调用）
type UpdateUserRequest struct {
	DisplayName string `json:"display_name"` // 显示名称（必填）
	Role        string `json:"role"`         // 角色（必填：admin/operator/viewer）
}

// ResetPasswordRequest 重置密码请求体（仅admin可调用）
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"` // 新密码（必填，>=6位）
}

// UpdateStatusRequest 启用/禁用用户请求体（仅admin可调用）
type UpdateStatusRequest struct {
	Status string `json:"status"` // 状态（必填：active/disabled）
}

// UpdateAssignmentsRequest 更新课程分配请求体（仅admin可调用）
type UpdateAssignmentsRequest struct {
	CourseCodes []string `json:"course_codes"` // 分配的课程代码列表（全量替换）
}

// CourseAssignment 课程分配记录（返回给前端）
type CourseAssignment struct {
	ID         string     `json:"id"`          // UUID
	UserID     string     `json:"user_id"`     // 用户ID
	CourseCode string     `json:"course_code"` // 课程代码
	AssignedBy string     `json:"assigned_by"` // 分配人ID
	AssignedAt *time.Time `json:"assigned_at"` // 分配时间
}

// UserListResponse 用户列表响应（含分页信息）
type UserListResponse struct {
	Users []*UserInfo `json:"users"` // 用户列表
	Total int         `json:"total"` // 总数
}

// ==================== 角色与状态常量 ====================

// 角色常量
const (
	RoleAdmin    = "admin"    // 管理员：全部权限
	RoleOperator = "operator" // 操作员：课程处理 + 审核
	RoleViewer   = "viewer"   // 查看者：只读
)

// 用户状态常量
const (
	StatusActive   = "active"   // 正常
	StatusDisabled = "disabled" // 禁用
)

// ValidRoles 有效角色列表（用于校验）
var ValidRoles = []string{RoleAdmin, RoleSeniorOperator, RoleOperator, RoleViewer}

// ValidStatuses 有效状态列表（用于校验）
var ValidStatuses = []string{StatusActive, StatusDisabled}

// IsValidRole 检查角色是否有效
func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// IsValidStatus 检查状态是否有效
func IsValidStatus(status string) bool {
	for _, s := range ValidStatuses {
		if s == status {
			return true
		}
	}
	return false
}
