package handlers

// recipe_available_handler.go — 对话模式可用配方列表处理器
//
// 提供 GET /api/v1/lesson-plans/recipes/available?subject=XX
// 返回当前老师在指定学科下可见的全部 active 配方，供对话模式首屏配方下拉消费。
//
// 解析逻辑：
//   1. 从 JWT 取当前用户ID
//   2. 调 GetSchoolIDByUserID 取用户所属学校
//   3. 调 GetUserTeachingGroups 取用户所属教研组ID列表
//   4. 调 ListAvailableRecipes 按可见性+学科查配方
//
// 失败容忍：学校/教研组解析失败只影响对应 scope 的配方不显示，不阻塞请求。

import (
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// HandleListAvailableRecipes 获取当前老师在指定学科下可见的配方列表
//
// 端点：GET /api/v1/lesson-plans/recipes/available?subject=人工智能
// 权限：登录即可（返回范围由可见性三路并集决定）
// 响应：{ code: 0, data: { recipes: [...], total: N } }
func HandleListAvailableRecipes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 从 JWT 取当前用户
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, "请先登录")
		return
	}
	userID := claims.UserID
	subject := r.URL.Query().Get("subject")

	// 解析用户所属学校（失败=空串，只影响学校共享配方不显示）
	schoolID, _ := repository.GetSchoolIDByUserID(r.Context(), userID)

	// 解析用户所属教研组ID列表（失败=空列表，只影响教研组共享配方不显示）
	var groupIDs []string
	groups, err := repository.GetUserTeachingGroups(r.Context(), userID)
	if err == nil {
		for _, g := range groups {
			groupIDs = append(groupIDs, g.ID)
		}
	}

	// 查询可见配方
	items, err := repository.ListAvailableRecipes(r.Context(), userID, groupIDs, schoolID, subject)
	if err != nil {
		utils.InternalError(w, "查询可用配方失败")
		return
	}

	utils.Success(w, map[string]interface{}{
		"recipes": items,
		"total":   len(items),
	})
}
