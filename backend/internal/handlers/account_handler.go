package handlers

/*
 * AccountHandler — 通用用户中心处理器
 *
 * 提供跨系统共用的个人账户管理接口：
 *   - GET  /api/v1/account/profile      获取当前用户详情
 *   - PUT  /api/v1/account/profile      更新个人信息（display_name）
 *   - PUT  /api/v1/account/password     修改自己的密码（需验证旧密码）
 */

import (
	"encoding/json"
	"errors"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// AccountHandler 用户中心处理器
type AccountHandler struct{}

// NewAccountHandler 创建用户中心处理器实例
func NewAccountHandler() *AccountHandler {
	return &AccountHandler{}
}

// ==================== 请求/响应结构体 ====================

// UpdateProfileRequest 更新个人信息请求体
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
}

// ChangePasswordRequest 修改密码请求体
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ProfileResponse 个人信息响应体
type ProfileResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	Status      string  `json:"status"`
	LoginCount  int     `json:"login_count"`
	LastLoginAt *string `json:"last_login_at"`
	CreatedAt   string  `json:"created_at"`
}

// ==================== 获取个人信息 ====================

// GetProfile GET /api/v1/account/profile
func (h *AccountHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	user, err := repository.FindUserByID(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, "获取用户信息失败")
		return
	}
	if user == nil {
		utils.Fail(w, http.StatusNotFound, "用户不存在")
		return
	}
	var lastLoginStr *string
	if user.LastLoginAt != nil {
		s := user.LastLoginAt.Format("2006-01-02 15:04:05")
		lastLoginStr = &s
	}
	resp := ProfileResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Status:      user.Status,
		LoginCount:  user.LoginCount,
		LastLoginAt: lastLoginStr,
		CreatedAt:   user.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	utils.Success(w, resp)
}

// ==================== 更新个人信息 ====================

// UpdateProfile PUT /api/v1/account/profile
func (h *AccountHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if len([]rune(req.DisplayName)) == 0 {
		utils.BadRequest(w, "显示名称不能为空")
		return
	}
	if len([]rune(req.DisplayName)) > 50 {
		utils.BadRequest(w, "显示名称不能超过50个字符")
		return
	}
	if err := repository.UpdateUserDisplayName(r.Context(), claims.UserID, req.DisplayName); err != nil {
		utils.InternalError(w, "更新失败，请稍后重试")
		return
	}
	utils.Success(w, map[string]string{
		"message":      "个人信息更新成功",
		"display_name": req.DisplayName,
	})
}

// ==================== 修改密码 ====================

// ChangePassword PUT /api/v1/account/password
func (h *AccountHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPutOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgUnauthorized)
		return
	}
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if req.OldPassword == "" {
		utils.BadRequest(w, "请输入旧密码")
		return
	}
	if len(req.NewPassword) < 6 {
		utils.BadRequest(w, "新密码长度不能少于6位")
		return
	}
	if req.OldPassword == req.NewPassword {
		utils.BadRequest(w, "新密码不能与旧密码相同")
		return
	}
	if err := repository.ChangeUserPassword(r.Context(), claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, repository.ErrWrongPassword) {
			utils.BadRequest(w, "旧密码不正确")
			return
		}
		utils.InternalError(w, "密码修改失败，请稍后重试")
		return
	}
	utils.Success(w, map[string]string{"message": "密码修改成功，请重新登录"})
}
