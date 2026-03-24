package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// AuthHandler 认证相关接口处理器
type AuthHandler struct {
	authService *services.AuthService // 认证服务
}

// 模块日志
var authHandlerLog = logger.WithModule("auth")

// NewAuthHandler 创建认证处理器实例
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login 登录接口
// POST /api/v1/auth/login
// 请求体：{"username": "admin", "password": "admin123"}
// 成功响应：{"code": 0, "message": "success", "data": {"token": "...", "user": {...}}}
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// 仅允许 POST 方法
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 解析请求体
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	// 校验必填字段
	if req.Username == "" || req.Password == "" {
		utils.BadRequest(w, "用户名和密码不能为空")
		return
	}

	// 调用认证服务执行登录
	resp, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		// 根据错误类型返回不同状态码
		if errors.Is(err, services.ErrInvalidCredentials) {
			utils.Unauthorized(w, "用户名或密码错误")
			return
		}
		if errors.Is(err, services.ErrUserDisabled) {
			utils.Forbidden(w, "账户已被禁用，请联系管理员")
			return
		}
		// 其他错误（数据库异常等）
		authHandlerLog.Error("登录接口系统错误", "error", err)
		utils.InternalError(w, "登录失败，请稍后重试")
		return
	}

	// 登录成功
	// 审计：用户登录成功
	repository.WriteAuditLog(resp.User.ID, repository.ActionLogin,
		map[string]interface{}{
			"username": resp.User.Username,
			"role":     resp.User.Role,
		},
		repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, resp)
}

// GetMe 获取当前登录用户信息
// GET /api/v1/auth/me
// 请求头：Authorization: Bearer <token>
// 成功响应：{"code": 0, "message": "success", "data": {"id": "...", "username": "...", ...}}
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// 仅允许 GET 方法
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 从请求头提取 token
	claims, err := h.extractClaims(r)
	if err != nil {
		utils.Unauthorized(w, err.Error())
		return
	}

	// 获取用户完整信息
	userInfo, err := h.authService.GetCurrentUser(r.Context(), claims)
	if err != nil {
		if errors.Is(err, services.ErrUserDisabled) {
			utils.Forbidden(w, "账户已被禁用")
			return
		}
		authHandlerLog.Error("获取用户信息失败", "error", err)
		utils.InternalError(w, "获取用户信息失败")
		return
	}

	utils.Success(w, userInfo)
}

// Logout 登出接口（前端清除 token 即可，后端记录日志）
// POST /api/v1/auth/logout
// 请求头：Authorization: Bearer <token>
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// 仅允许 POST 方法
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 验证 token（确保是合法用户发起的登出）
	claims, err := h.extractClaims(r)
	if err != nil {
		// 即使 token 无效也返回成功（前端清除即可）
		utils.Success(w, nil)
		return
	}

	authHandlerLog.Info("用户登出",
		"username", claims.Username,
		"user_id", claims.UserID,
		"role", claims.Role,
	)
	repository.WriteAuditLog(claims.UserID, repository.ActionLogout,
		map[string]interface{}{
			"username": claims.Username,
			"role":     claims.Role,
		},
		repository.GetClientIP(r.RemoteAddr))
	utils.Success(w, nil)
}

// extractClaims 从 Authorization 请求头中提取并验证 JWT token
// 格式：Authorization: Bearer <token>
func (h *AuthHandler) extractClaims(r *http.Request) (*services.JWTClaims, error) {
	// 获取 Authorization 头
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("未提供认证令牌")
	}

	// 检查 Bearer 前缀
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, errors.New("认证令牌格式错误")
	}

	tokenString := strings.TrimSpace(parts[1])
	if tokenString == "" {
		return nil, errors.New("认证令牌为空")
	}

	// 验证 token
	claims, err := h.authService.ValidateToken(tokenString)
	if err != nil {
		if errors.Is(err, services.ErrTokenExpired) {
			return nil, errors.New("认证令牌已过期，请重新登录")
		}
		return nil, errors.New("认证令牌无效")
	}

	return claims, nil
}
