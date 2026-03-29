package handlers

/*
 * AccountHandler — 通用用户中心处理器
 *
 * 提供跨系统共用的个人账户管理接口：
 *   - GET  /api/v1/account/profile      获取当前用户详情（含登录记录）
 *   - PUT  /api/v1/account/profile      更新个人信息（display_name）
 *   - PUT  /api/v1/account/password     修改自己的密码（需验证旧密码）
 *
 * 与 UserHandler 的区别：
 *   UserHandler   → admin管理其他用户（CRUD、重置密码）
 *   AccountHandler → 任何已登录用户管理自己的账户
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
	DisplayName string `json:"display_name"` // 显示名称（必填）
}

// ChangePasswordRequest 修改密码请求体
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"` // 旧密码（必填，用于验证身份）
	NewPassword string `json:"new_password"` // 新密码（必填，最少6位）
}

// ProfileResponse 个人信息响应体
type ProfileResponse struct {
	ID           string  `json:"id"`            // 用户UUID
	Username     string  `json:"username"`      // 登录用户名
	DisplayName  string  `json:"display_name"`  // 显示名称
	Role         string  `json:"role"`          // 系统角色
	Status       string  `json:"status"`        // 账户状态
	LoginCount   int     `json:"login_count"`   // 累计登录次数
	LastLoginAt  *string `json:"last_login_at"` // 最近登录时间（ISO8601）
	CreatedAt    string  `json:"created_at"`    // 注册时间
}

// ==================== 获取个人信息 ====================

// GetProfile 获取当前登录用户的个人信息
// GET /api/v1/account/profile
// 响应：ProfileResponse
func (h *AccountHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 从JWT Claims获取当前用户ID
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未获取到用户信息")
		return
	}

	// 查询用户完整信息
	user, err := repository.FindUserByID(r.Context(), claims.UserID)
	if err != nil {
		utils.InternalError(w, "获取用户信息失败")
		return
	}
	if user == nil {
		utils.Fail(w, http.StatusNotFound, "用户不存在")
		return
	}

	// 格式化时间字段
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

// UpdateProfile 更新当前用户的显示名称
// PUT /api/v1/account/profile
// 请求体：{"display_name":"新名称"}
func (h *AccountHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未获取到用户信息")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	// 校验显示名称
	if len([]rune(req.DisplayName)) == 0 {
		utils.BadRequest(w, "显示名称不能为空")
		return
	}
	if len([]rune(req.DisplayName)) > 50 {
		utils.BadRequest(w, "显示名称不能超过50个字符")
		return
	}

	// 更新数据库
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

// ChangePassword 当前用户修改自己的密码
// PUT /api/v1/account/password
// 请求体：{"old_password":"旧密码","new_password":"新密码"}
func (h *AccountHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未获取到用户信息")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	// 参数校验
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

	// 调用Repository层验证旧密码+更新
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
