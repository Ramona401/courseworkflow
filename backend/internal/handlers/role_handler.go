package handlers

/*
 * role_handler.go — 角色权限管理 HTTP 处理器
 *
 * 路由前缀：/api/v1/admin/roles/
 * 权限：全部接口仅 admin 可访问（由路由层 adminOnly 中间件保护）
 *
 * 接口列表：
 *   GET    /api/v1/admin/roles                       — 角色列表（含权限数+用户数）
 *   POST   /api/v1/admin/roles                       — 新建自定义角色
 *   GET    /api/v1/admin/roles/{id}                  — 角色详情（含权限明细）
 *   PUT    /api/v1/admin/roles/{id}                  — 编辑角色（is_system 拒绝）
 *   PUT    /api/v1/admin/roles/{id}/status           — 启用/禁用（is_system 拒绝）
 *   DELETE /api/v1/admin/roles/{id}                  — 删除角色（is_system/有用户 拒绝）
 *   GET    /api/v1/admin/roles/{id}/permissions      — 获取权限列表
 *   PUT    /api/v1/admin/roles/{id}/permissions      — 全量更新权限（is_system 拒绝）
 */

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// RoleHandler 角色权限管理处理器
type RoleHandler struct {
	roleService *services.RoleService
}

// NewRoleHandler 构造函数
func NewRoleHandler(roleService *services.RoleService) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

// ==================== 角色列表 ====================

// ListRoles 获取所有角色（含权限数和用户数统计）
// GET /api/v1/admin/roles
func (h *RoleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	result, err := h.roleService.ListRoles(r.Context())
	if err != nil {
		utils.InternalError(w, "获取角色列表失败: "+err.Error())
		return
	}

	utils.Success(w, result)
}

// ==================== 新建角色 ====================

// CreateRole 新建自定义角色
// POST /api/v1/admin/roles
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 获取当前操作者信息
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未获取到用户信息")
		return
	}

	var req models.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	role, err := h.roleService.CreateRole(r.Context(), &req, claims.UserID)
	if err != nil {
		handleRoleError(w, err)
		return
	}

	utils.Success(w, role)
}

// ==================== 角色详情 ====================

// GetRole 获取角色详情（含权限明细）
// GET /api/v1/admin/roles/{id}
func (h *RoleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	roleID := extractRolePathID(r.URL.Path)
	if roleID == "" {
		utils.BadRequest(w, "缺少角色ID")
		return
	}

	detail, err := h.roleService.GetRole(r.Context(), roleID)
	if err != nil {
		handleRoleError(w, err)
		return
	}

	utils.Success(w, detail)
}

// ==================== 编辑角色 ====================

// UpdateRole 编辑角色显示名和描述
// PUT /api/v1/admin/roles/{id}
func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	roleID := extractRolePathID(r.URL.Path)
	if roleID == "" {
		utils.BadRequest(w, "缺少角色ID")
		return
	}

	var req models.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.roleService.UpdateRole(r.Context(), roleID, &req); err != nil {
		handleRoleError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "角色更新成功"})
}

// ==================== 状态管理 ====================

// UpdateRoleStatus 启用/禁用角色
// PUT /api/v1/admin/roles/{id}/status
func (h *RoleHandler) UpdateRoleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	roleID := extractRoleMiddleID(r.URL.Path, "/status")
	if roleID == "" {
		utils.BadRequest(w, "缺少角色ID")
		return
	}

	var req models.UpdateRoleStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.roleService.UpdateRoleStatus(r.Context(), roleID, req.Status); err != nil {
		handleRoleError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "角色状态更新成功"})
}

// ==================== 删除角色 ====================

// DeleteRole 删除角色（系统角色和有用户的角色均拒绝）
// DELETE /api/v1/admin/roles/{id}
func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}

	roleID := extractRolePathID(r.URL.Path)
	if roleID == "" {
		utils.BadRequest(w, "缺少角色ID")
		return
	}

	if err := h.roleService.DeleteRole(r.Context(), roleID); err != nil {
		handleRoleError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "角色删除成功"})
}

// ==================== 权限管理 ====================

// GetRolePermissions 获取角色权限列表
// GET /api/v1/admin/roles/{id}/permissions
func (h *RoleHandler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	roleID := extractRoleMiddleID(r.URL.Path, "/permissions")
	if roleID == "" {
		utils.BadRequest(w, "缺少角色ID")
		return
	}

	perms, err := h.roleService.GetRolePermissions(r.Context(), roleID)
	if err != nil {
		handleRoleError(w, err)
		return
	}

	utils.Success(w, perms)
}

// UpdateRolePermissions 全量更新角色权限
// PUT /api/v1/admin/roles/{id}/permissions
func (h *RoleHandler) UpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	roleID := extractRoleMiddleID(r.URL.Path, "/permissions")
	if roleID == "" {
		utils.BadRequest(w, "缺少角色ID")
		return
	}

	var req models.UpdateRolePermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.roleService.UpdateRolePermissions(r.Context(), roleID, &req); err != nil {
		handleRoleError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "权限更新成功"})
}

// ==================== 错误处理 ====================

// handleRoleError 统一错误码映射
func handleRoleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrRoleNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrSystemRoleReadonly):
		utils.Fail(w, http.StatusForbidden, err.Error())
	case errors.Is(err, services.ErrRoleCodeRequired),
		errors.Is(err, services.ErrRoleCodeInvalid),
		errors.Is(err, services.ErrRoleCodeExists),
		errors.Is(err, services.ErrRoleDisplayRequired),
		errors.Is(err, services.ErrInvalidBaseRole):
		utils.BadRequest(w, err.Error())
	default:
		// ErrRoleInUse 及包含用户数的动态错误也走 BadRequest（语义上是"操作前置条件未满足"）
		if strings.Contains(err.Error(), "还有") && strings.Contains(err.Error(), "个用户") {
			utils.BadRequest(w, err.Error())
			return
		}
		utils.InternalError(w, "操作失败: "+err.Error())
	}
}

// ==================== 路径提取工具 ====================

// extractRolePathID 提取末尾角色ID
// /api/v1/admin/roles/xxx-yyy → xxx-yyy
// /api/v1/admin/roles/xxx-yyy/status → xxx-yyy（当后面还有子路径时，取第一段）
func extractRolePathID(path string) string {
	prefix := "/api/v1/admin/roles/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimSuffix(rest, "/")
	// 若有子路径（如 /status /permissions），取 ID 段
	if idx := strings.Index(rest, "/"); idx > 0 {
		rest = rest[:idx]
	}
	return rest
}

// extractRoleMiddleID 提取中间角色ID（路径包含固定后缀时使用）
// /api/v1/admin/roles/xxx-yyy/status → xxx-yyy
// /api/v1/admin/roles/xxx-yyy/permissions → xxx-yyy
func extractRoleMiddleID(path, suffix string) string {
	prefix := "/api/v1/admin/roles/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(rest, "/"); idx > 0 {
		candidate := rest[:idx]
		tail := rest[idx:]
		if strings.HasPrefix(tail, suffix) {
			return candidate
		}
	}
	return ""
}
