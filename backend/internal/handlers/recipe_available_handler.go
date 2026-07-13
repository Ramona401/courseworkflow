package handlers

// recipe_available_handler.go — 备课首屏可用配方列表处理器
//
// GET /api/v1/lesson-plans/recipes/available?subject=XX&grade=高三
//
// 返回当前老师在指定学科、具体年级下可见的 active 配方。
// 无同年级配方时返回空列表，不降级到学段、空年级或其他年级。

import (
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// HandleListAvailableRecipes 获取首屏可用配方列表。
func HandleListAvailableRecipes(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET请求",
		)
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims.UserID == "" {
		utils.Unauthorized(w, "请先登录")
		return
	}

	subject := strings.TrimSpace(
		r.URL.Query().Get("subject"),
	)
	grade := strings.TrimSpace(
		r.URL.Query().Get("grade"),
	)

	if subject == "" {
		utils.BadRequest(w, "缺少 subject 参数")
		return
	}
	if grade == "" {
		utils.BadRequest(w, "缺少 grade 参数")
		return
	}

	schoolID, _ := repository.GetSchoolIDByUserID(
		r.Context(),
		claims.UserID,
	)

	var groupIDs []string
	groups, err := repository.GetUserTeachingGroups(
		r.Context(),
		claims.UserID,
	)
	if err == nil {
		for _, group := range groups {
			groupIDs = append(groupIDs, group.ID)
		}
	}

	items, err := repository.ListAvailableRecipes(
		r.Context(),
		claims.UserID,
		groupIDs,
		schoolID,
		subject,
		grade,
	)
	if err != nil {
		utils.InternalError(w, "查询可用配方失败")
		return
	}

	utils.Success(w, map[string]interface{}{
		"recipes": items,
		"total":   len(items),
	})
}
