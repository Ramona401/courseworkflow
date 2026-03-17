package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"tedna/internal/services"
	"tedna/internal/utils"
)

// 上下文键类型（避免与其他包冲突）
type contextKey string

// 上下文键常量：存储当前用户的 JWT 声明
const ClaimsKey contextKey = "claims"

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
				log.Printf("权限拒绝: 用户 %s (角色: %s) 尝试访问需要 %v 角色的资源",
					claims.Username, userRole, allowedRoles)
				utils.Forbidden(w, "权限不足，需要以下角色之一: "+strings.Join(allowedRoles, ", "))
				return
			}

			// 3. 角色验证通过，继续处理
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
