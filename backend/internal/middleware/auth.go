package middleware

// JWT认证中间件 + RBAC权限中间件
// Phase8日志升级：权限拒绝事件输出结构化日志（级别WARN，含username/role/path等字段）
//
// 超管收口新增：SuperAdminOnly 中间件
//   在 RequireRole("admin") 之上再收一层——不仅要求是 admin 角色，还要求
//   claims.IsSuper == true。用于保护"模型配置/积分/AI统计/审计日志"等只有真超管
//   能碰的敏感路由。二线管理员(admin 但 is_super=false)会被本中间件拦下返回 403，
//   与前端入口隐藏形成双重收口（前端隐藏是体验，本中间件是真墙）。

import (
	"context"
	"net/http"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// 上下文键类型（避免与其他包冲突）
type contextKey string

// ClaimsKey 上下文键常量：存储当前用户的 JWT 声明
const ClaimsKey contextKey = "claims"

// 模块日志
var log = logger.WithModule("middleware")

// AuthMiddleware 认证中间件：验证 JWT token
// 用法：AuthMiddleware(authService)(nextHandler)
func AuthMiddleware(authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. 获取 Authorization 请求头
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.Unauthorized(w, "未提供认证令牌")
				return
			}

			// 2. 检查 Bearer 前缀并提取 token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				utils.Unauthorized(w, "认证令牌格式错误")
				return
			}

			tokenString := strings.TrimSpace(parts[1])
			if tokenString == "" {
				utils.Unauthorized(w, "认证令牌为空")
				return
			}

			// 3. 验证 token
			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				if err == services.ErrTokenExpired {
					utils.Unauthorized(w, "认证令牌已过期，请重新登录")
					return
				}
				utils.Unauthorized(w, "认证令牌无效")
				return
			}

			// 4. 将 claims 存入请求上下文，供后续 handler 使用
			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole RBAC 角色权限中间件：检查用户角色是否在允许列表中
// 用法：RequireRole("admin", "operator")(nextHandler)
// 必须在 AuthMiddleware 之后使用（依赖上下文中的 claims）
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. 从上下文获取 claims
			claims, ok := GetClaims(r.Context())
			if !ok {
				utils.Unauthorized(w, "未找到认证信息")
				return
			}

			// 2. 检查用户角色是否在允许列表中
			userRole := claims.Role
			allowed := false
			for _, role := range allowedRoles {
				if userRole == role {
					allowed = true
					break
				}
			}

			if !allowed {
				// WARN级别：权限拒绝属于安全事件，需要关注但不是系统错误
				log.Warn("权限拒绝",
					"username", claims.Username,
					"user_id", claims.UserID,
					"role", userRole,
					"required_roles", allowedRoles,
					"path", r.URL.Path,
					"method", r.Method,
					"remote_addr", r.RemoteAddr,
				)
				utils.Forbidden(w, "权限不足，需要以下角色之一: "+strings.Join(allowedRoles, ", "))
				return
			}

			// 3. 角色验证通过，继续处理
			next.ServeHTTP(w, r)
		})
	}
}

// SuperAdminOnly 超级管理员专属中间件：仅放行 is_super=true 的账号。
//
// 用法：SuperAdminOnly()(nextHandler)
// 必须在 AuthMiddleware 之后使用（依赖上下文中的 claims）。
//
// 判定：claims.IsSuper == true 才放行，否则返回 403。
//   - 真超管(admin 且 is_super=true)：放行；
//   - 二线管理员(admin 但 is_super=false)：拦下（有 admin 角色但非超管）；
//   - 其余角色：自然拦下（is_super 恒为 false）。
//
// 用于保护"模型配置/积分/AI统计/审计日志"等敏感路由，是这些入口的真实安全边界
// （前端入口隐藏仅是体验，绕过前端直敲 API 由本中间件兜底拦截）。
//
// 存量 token（未带 is_super 字段）解析后 IsSuper 默认 false，会被拦下——
// 属收紧方向（fail-safe），老超管重新登录换新 token 即恢复访问，绝不会误放行。
func SuperAdminOnly() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. 从上下文获取 claims
			claims, ok := GetClaims(r.Context())
			if !ok {
				utils.Unauthorized(w, "未找到认证信息")
				return
			}

			// 2. 仅超管放行
			if !claims.IsSuper {
				// WARN级别：非超管尝试访问超管专属入口，属安全事件需关注
				log.Warn("超管权限拒绝",
					"username", claims.Username,
					"user_id", claims.UserID,
					"role", claims.Role,
					"is_super", claims.IsSuper,
					"path", r.URL.Path,
					"method", r.Method,
					"remote_addr", r.RemoteAddr,
				)
				utils.Forbidden(w, "该功能仅超级管理员可访问")
				return
			}

			// 3. 超管验证通过，继续处理
			next.ServeHTTP(w, r)
		})
	}
}

// GetClaims 从上下文中获取当前用户的 JWT 声明
// 供 handler 层使用，获取当前登录用户信息
func GetClaims(ctx context.Context) (*services.JWTClaims, bool) {
	claims, ok := ctx.Value(ClaimsKey).(*services.JWTClaims)
	return claims, ok
}

// Chain 中间件链：按顺序执行多个中间件
// 用法：Chain(handler, AuthMiddleware(svc), RequireRole("admin"))
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	// 从右到左包裹，确保执行顺序是从左到右
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
