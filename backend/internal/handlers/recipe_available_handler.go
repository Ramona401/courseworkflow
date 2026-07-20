package handlers

// recipe_available_handler.go — 备课首屏手动可选配方列表处理器
//
// GET /api/v1/lesson-plans/recipes/available?subject=XX&grade=高三
//
// 本接口服务于老师“指定配方”的手动选择，不参与平台自动挂载。
//
// 返回当前老师有权使用并且与当前教学教育域兼容的全部active配方：
//  1. 老师个人创建的配方；
//  2. 老师所属教研组共享的配方；
//  3. 老师所属学校共享的配方。
//
// 教育域规则：
//   - K12教学上下文只能看到k12或common配方；
//   - 职教教学上下文只能看到vocational或common配方；
//   - 成教教学上下文只能看到adult或common配方；
//   - mixed只用于跨域管理，不能直接作为具体教学运行域；
//   - 空值、common和非法当前域严格返回空候选。
//
// subject和grade只用于相关性排序：
//   - 同学科、同具体年级优先；
//   - 同学科、其它年级或学段其次；
//   - 其它学科最后。
//
// 平台自动选择仍由recipe_resolver.go负责，继续执行同学科、
// 同具体年级和教育域严格匹配。

import (
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// HandleListAvailableRecipes 获取老师手动选择时可用的配方列表。
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

	// 当前课程的学科和年级仅用于候选排序，
	// 但仍要求前端传入，避免手动列表失去相关性顺序。
	if subject == "" {
		utils.BadRequest(w, "缺少 subject 参数")
		return
	}
	if grade == "" {
		utils.BadRequest(w, "缺少 grade 参数")
		return
	}

	// 使用统一Actor解析用户的学校、教研组和确定性教育域。
	//
	// 普通教师会解析为k12、vocational或adult具体教学域；
	// admin、region_admin和district_inspector在管理上下文中为mixed，
	// 因为本候选接口尚未绑定具体教案，仓储会对mixed严格返回空候选。
	actor := services.BuildActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	)
	if actor == nil {
		utils.InternalError(w, "解析当前用户教学范围失败")
		return
	}

	items, err := repository.ListAvailableRecipesForDomain(
		r.Context(),
		actor.UserID,
		actor.MyGroupIDs,
		actor.SchoolID,
		actor.EducationDomain,
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
