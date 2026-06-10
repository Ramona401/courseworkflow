package middleware

// kb_guard.go — 知识库压缩子系统访问白名单守卫
//
// 守卫语义（不绑角色）：
//   1. admin 角色恒通过（admin 是平台超管，永远可进）
//   2. 否则查 kb_authorized_users 白名单，user_id 在名单内则通过
//   3. 其余一律 403
//
// 必须挂在 AuthMiddleware 之后（依赖上下文 claims）。
// 与 RequireRole 的区别：RequireRole 只认角色；本守卫认「具体成员名单」，
// 被授权人在系统里是什么角色都不影响（PKU AI 学校录入人员按项目增删）。

import (
	"net/http"

	"tedna/internal/repository"
	"tedna/internal/utils"
)

const kbRoleAdmin = "admin"

// RequireKBAuthorized 知识库白名单守卫中间件
func RequireKBAuthorized() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaims(r.Context())
			if !ok || claims == nil {
				utils.Unauthorized(w, "未找到认证信息")
				return
			}

			// admin 恒通过
			if claims.Role == kbRoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// 否则查白名单
			authorized, err := repository.IsKBAuthorized(r.Context(), claims.UserID)
			if err != nil {
				utils.InternalError(w, "校验访问权限失败")
				return
			}
			if !authorized {
				log.Warn("知识库白名单拒绝",
					"username", claims.Username,
					"user_id", claims.UserID,
					"role", claims.Role,
					"path", r.URL.Path,
				)
				utils.Forbidden(w, "无权访问知识库压缩系统")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
