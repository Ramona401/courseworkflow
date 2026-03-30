package models

/*
 * role.go — 角色权限数据模型
 *
 * 对应数据库表：
 *   roles            — 自定义角色表（含4个系统内置角色，is_system=true）
 *   role_permissions — 角色权限明细表
 *
 * 业务规则（在 service 层强制执行）：
 *   1. is_system=true 的角色只读，不可编辑/删除/改状态
 *   2. 新建角色时若指定 base_role，自动复制该内置角色的所有权限
 *   3. 删除角色前检查 users.role = role_code，有用户使用则拒绝
 */

// ==================== 角色主表模型 ====================

// Role 对应 roles 表
type Role struct {
	ID          string  `json:"id"`
	RoleCode    string  `json:"role_code"`    // 角色唯一码，如 custom_teacher
	DisplayName string  `json:"display_name"` // 显示名称
	Description string  `json:"description"`  // 描述
	BaseRole    string  `json:"base_role"`    // 基础角色（admin/senior_operator/operator/viewer）
	IsSystem    bool    `json:"is_system"`    // 是否系统内置（true则只读）
	Status      string  `json:"status"`       // active / disabled
	CreatedBy   *string `json:"created_by"`   // 创建者ID（系统内置为null）
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// ==================== 角色权限表模型 ====================

// RolePermission 对应 role_permissions 表
type RolePermission struct {
	ID             string `json:"id"`
	RoleID         string `json:"role_id"`
	PermissionCode string `json:"permission_code"` // 格式：{resource}.{action}
	Resource       string `json:"resource"`        // pipeline / user / course / ai_config / prompt / system / lesson_plan
	Action         string `json:"action"`          // create / read / update / delete / start / cancel / ...
	CreatedAt      string `json:"created_at"`
}

// ==================== 列表响应 ====================

// RoleListItem 角色列表项（含权限数量和使用人数统计）
type RoleListItem struct {
	ID              string `json:"id"`
	RoleCode        string `json:"role_code"`
	DisplayName     string `json:"display_name"`
	Description     string `json:"description"`
	BaseRole        string `json:"base_role"`
	IsSystem        bool   `json:"is_system"`
	Status          string `json:"status"`
	PermissionCount int    `json:"permission_count"` // 权限条数
	UserCount       int    `json:"user_count"`       // 使用该角色的用户数
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// RoleListResponse 角色列表响应
type RoleListResponse struct {
	Roles []*RoleListItem `json:"roles"`
	Total int             `json:"total"`
}

// ==================== 详情响应 ====================

// RoleDetailResponse 角色详情（含权限列表）
type RoleDetailResponse struct {
	ID          string            `json:"id"`
	RoleCode    string            `json:"role_code"`
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	BaseRole    string            `json:"base_role"`
	IsSystem    bool              `json:"is_system"`
	Status      string            `json:"status"`
	CreatedBy   *string           `json:"created_by"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	Permissions []*RolePermission `json:"permissions"` // 权限明细
	UserCount   int               `json:"user_count"`  // 使用人数
}

// ==================== 请求结构体 ====================

// CreateRoleRequest 新建角色请求
type CreateRoleRequest struct {
	RoleCode    string `json:"role_code"`    // 必填，唯一，字母数字下划线
	DisplayName string `json:"display_name"` // 必填，显示名称
	Description string `json:"description"`  // 选填，描述
	BaseRole    string `json:"base_role"`    // 选填，指定则自动复制该内置角色的权限
}

// UpdateRoleRequest 编辑角色请求（仅允许改显示名和描述）
type UpdateRoleRequest struct {
	DisplayName string `json:"display_name"` // 必填
	Description string `json:"description"`  // 选填
}

// UpdateRoleStatusRequest 启用/禁用角色请求
type UpdateRoleStatusRequest struct {
	Status string `json:"status"` // active / disabled
}

// UpdateRolePermissionsRequest 批量更新角色权限请求（全量替换）
type UpdateRolePermissionsRequest struct {
	Permissions []PermissionItem `json:"permissions"` // 权限列表（全量替换）
}

// PermissionItem 单条权限
type PermissionItem struct {
	PermissionCode string `json:"permission_code"` // {resource}.{action}
	Resource       string `json:"resource"`
	Action         string `json:"action"`
}

// ==================== 枚举常量 ====================

// 系统内置角色码（对应 roles.role_code，与 users.role 枚举保持一致）
const (
	RoleCodeAdmin          = "admin"
	RoleCodeSeniorOperator = "senior_operator"
	RoleCodeOperator       = "operator"
	RoleCodeViewer         = "viewer"
)

// 角色状态
const (
	RoleStatusActive   = "active"
	RoleStatusDisabled = "disabled"
)

// 权限资源枚举
var ValidResources = map[string]bool{
	"pipeline":    true,
	"user":        true,
	"course":      true,
	"ai_config":   true,
	"prompt":      true,
	"system":      true,
	"lesson_plan": true,
}

// 权限动作枚举
var ValidActions = map[string]bool{
	"create":        true,
	"read":          true,
	"update":        true,
	"delete":        true,
	"start":         true,
	"cancel":        true,
	"review":        true,
	"finalize":      true,
	"verify":        true,
	"batch":         true,
	"force_proceed": true,
	"restart":       true,
	"manage":        true,
	"config":        true,
	"write":         true,
}

// ValidBaseRoles 允许作为 base_role 的内置角色码
var ValidBaseRoles = map[string]bool{
	RoleCodeAdmin:          true,
	RoleCodeSeniorOperator: true,
	RoleCodeOperator:       true,
	RoleCodeViewer:         true,
}
