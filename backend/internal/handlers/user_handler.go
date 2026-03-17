package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// UserHandler 用户管理接口处理器（仅admin可访问）
type UserHandler struct {
	userService *services.UserService // 用户管理服务
}

// NewUserHandler 创建用户管理处理器实例
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// ==================== 用户列表 ====================

// List 获取所有用户列表
// GET /api/v1/users
// 响应：{"code":0,"data":{"users":[...],"total":N}}
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	result, err := h.userService.ListUsers(r.Context())
	if err != nil {
		log.Printf("获取用户列表失败: %v", err)
		utils.InternalError(w, "获取用户列表失败")
		return
	}

	utils.Success(w, result)
}

// ==================== 创建用户 ====================

// Create 创建新用户
// POST /api/v1/users
// 请求体：{"username":"...", "display_name":"...", "password":"...", "role":"operator"}
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	userInfo, err := h.userService.CreateUser(r.Context(), &req)
	if err != nil {
		h.handleUserError(w, err)
		return
	}

	utils.Success(w, userInfo)
}

// ==================== 编辑用户 ====================

// Update 更新用户基本信息
// PUT /api/v1/users/:id
// 请求体：{"display_name":"...", "role":"operator"}
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	// 从URL提取用户ID
	userID := extractUserID(r.URL.Path, "/api/v1/users/")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
		return
	}

	// 获取当前操作者ID（用于防止自改角色）
	currentUserID := getCurrentUserID(r)

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	userInfo, err := h.userService.UpdateUser(r.Context(), userID, currentUserID, &req)
	if err != nil {
		h.handleUserError(w, err)
		return
	}

	utils.Success(w, userInfo)
}

// ==================== 重置密码 ====================

// ResetPassword 重置用户密码
// PUT /api/v1/users/:id/password
// 请求体：{"new_password":"..."}
func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	// 从URL提取用户ID（路径格式：/api/v1/users/{id}/password）
	userID := extractMiddleID(r.URL.Path, "/api/v1/users/", "/password")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
		return
	}

	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.userService.ResetPassword(r.Context(), userID, &req); err != nil {
		h.handleUserError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "密码重置成功"})
}

// ==================== 启用/禁用用户 ====================

// UpdateStatus 更新用户状态
// PUT /api/v1/users/:id/status
// 请求体：{"status":"active"} 或 {"status":"disabled"}
func (h *UserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	// 从URL提取用户ID（路径格式：/api/v1/users/{id}/status）
	userID := extractMiddleID(r.URL.Path, "/api/v1/users/", "/status")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
		return
	}

	currentUserID := getCurrentUserID(r)

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	if err := h.userService.UpdateStatus(r.Context(), userID, currentUserID, &req); err != nil {
		h.handleUserError(w, err)
		return
	}

	utils.Success(w, map[string]string{"message": "用户状态更新成功"})
}

// ==================== 课程分配 ====================

// GetAssignments 获取用户的课程分配列表
// GET /api/v1/users/:id/assignments
func (h *UserHandler) GetAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	userID := extractMiddleID(r.URL.Path, "/api/v1/users/", "/assignments")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
		return
	}

	assignments, err := h.userService.GetAssignments(r.Context(), userID)
	if err != nil {
		h.handleUserError(w, err)
		return
	}

	// 如果为nil则返回空数组
	if assignments == nil {
		assignments = []*models.CourseAssignment{}
	}

	utils.Success(w, assignments)
}

// UpdateAssignments 更新用户的课程分配（全量替换）
// PUT /api/v1/users/:id/assignments
// 请求体：{"course_codes":["G7-03","G8-01"]}
func (h *UserHandler) UpdateAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	userID := extractMiddleID(r.URL.Path, "/api/v1/users/", "/assignments")
	if userID == "" {
		utils.BadRequest(w, "缺少用户ID")
		return
	}

	adminID := getCurrentUserID(r)

	var req models.UpdateAssignmentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	assignments, err := h.userService.UpdateAssignments(r.Context(), userID, adminID, &req)
	if err != nil {
		h.handleUserError(w, err)
		return
	}

	utils.Success(w, assignments)
}

// ==================== 内部工具方法 ====================

// handleUserError 统一处理用户管理业务错误
func (h *UserHandler) handleUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrUsernameRequired),
		errors.Is(err, services.ErrDisplayNameRequired),
		errors.Is(err, services.ErrPasswordTooShort),
		errors.Is(err, services.ErrInvalidRole),
		errors.Is(err, services.ErrInvalidStatus):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrUsernameExists):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrUserNotFound):
		utils.Fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrCannotDisableSelf):
		utils.BadRequest(w, err.Error())
	case errors.Is(err, services.ErrCannotChangeOwnRole):
		utils.BadRequest(w, err.Error())
	default:
		log.Printf("用户管理操作失败: %v", err)
		utils.InternalError(w, "操作失败，请稍后重试")
	}
}

// extractUserID 从URL路径中提取用户ID
// 示例："/api/v1/users/xxx-yyy" → prefix="/api/v1/users/" → "xxx-yyy"
func extractUserID(path string, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	// 去掉末尾可能的斜杠
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		return ""
	}
	return id
}

// extractMiddleID 从含子路径的URL中提取中间的ID
// 示例："/api/v1/users/xxx-yyy/password" → prefix="/api/v1/users/" suffix="/password" → "xxx-yyy"
func extractMiddleID(path string, prefix string, suffix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if !strings.HasSuffix(rest, suffix) {
		return ""
	}
	id := strings.TrimSuffix(rest, suffix)
	if id == "" {
		return ""
	}
	return id
}

// getCurrentUserID 从请求上下文中获取当前操作者的用户ID
func getCurrentUserID(r *http.Request) string {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		return ""
	}
	return claims.UserID
}
