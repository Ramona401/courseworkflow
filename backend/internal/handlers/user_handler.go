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

// ==================== 路径前缀常量 ====================

const usersPrefix = "/api/v1/users/"

// UserHandler 用户管理接口处理器（仅admin可访问）
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler 创建用户管理处理器实例
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// ==================== 用户列表 ====================

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
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

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	userID := extractUserID(r.URL.Path, usersPrefix)
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}
	currentUserID := getCurrentUserID(r)
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	userID := extractMiddleID(r.URL.Path, usersPrefix, "/password")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}
	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.userService.ResetPassword(r.Context(), userID, &req); err != nil {
		h.handleUserError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "密码重置成功"})
}

// ==================== 启用/禁用用户 ====================

func (h *UserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	userID := extractMiddleID(r.URL.Path, usersPrefix, "/status")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}
	currentUserID := getCurrentUserID(r)
	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if err := h.userService.UpdateStatus(r.Context(), userID, currentUserID, &req); err != nil {
		h.handleUserError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "用户状态更新成功"})
}

// ==================== 课程分配 ====================

func (h *UserHandler) GetAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	userID := extractMiddleID(r.URL.Path, usersPrefix, "/assignments")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}
	assignments, err := h.userService.GetAssignments(r.Context(), userID)
	if err != nil {
		h.handleUserError(w, err)
		return
	}
	if assignments == nil {
		assignments = []*models.CourseAssignment{}
	}
	utils.Success(w, assignments)
}

func (h *UserHandler) UpdateAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	userID := extractMiddleID(r.URL.Path, usersPrefix, "/assignments")
	if userID == "" {
		utils.BadRequest(w, utils.MsgMissingUserID)
		return
	}
	adminID := getCurrentUserID(r)
	var req models.UpdateAssignmentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	assignments, err := h.userService.UpdateAssignments(r.Context(), userID, adminID, &req)
	if err != nil {
		h.handleUserError(w, err)
		return
	}
	utils.Success(w, assignments)
}

// ==================== 错误处理 ====================

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

// ==================== 路径辅助函数 ====================

func extractUserID(path string, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		return ""
	}
	return id
}

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

func getCurrentUserID(r *http.Request) string {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		return ""
	}
	return claims.UserID
}
