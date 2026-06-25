package handlers

// token_summary_handler.go — Token积分消费汇总报告HTTP处理器
//
// 积分消费汇总报告 batch 新增（本次）：
//   独立文件承载汇总端点，方法挂在 *TokenHandler 上，复用其 tokenService 与 resolveScope。
//
// 端点：
//   GET /api/v1/tokens/consumption-summary
//       ?dimension=school|user|model|scene|time
//       &from=YYYY-MM-DD（可选）&to=YYYY-MM-DD（可选）
//       &school_filter=<学校组织ID>（可选，下钻：看该校各老师）
//       &user_filter=<user_id>（可选，下钻：看该老师各模型/场景/时间）
//   登录即可访问，数据由 TokenScope 收窄。
//
// 下钻过滤的 scope 内二次校验（防越权，关键）：
//   - user_filter：必须 ∈ scope.UserIDs（admin 的 UserIDs=nil 放行任意）；否则忽略该过滤
//     并按空集处理（返回空），绝不让人传别人 user_id 越权看消费。
//   - school_filter：取该校成员 user_id 列表后，与 scope.UserIDs 求交集（admin 不收窄），
//     只把"既是该校成员、又在 scope 内"的 user_id 传给 repo，双重保证不越权。

import (
	"net/http"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// GetConsumptionSummary 消费汇总报告
// GET /api/v1/tokens/consumption-summary
func (h *TokenHandler) GetConsumptionSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, http.StatusMethodNotAllowed, -1, "仅支持GET请求", nil)
		return
	}
	q := r.URL.Query()
	dimension := q.Get("dimension")

	// 校验维度
	switch dimension {
	case models.SummaryDimRegion, models.SummaryDimSchool, models.SummaryDimUser, models.SummaryDimModel,
		models.SummaryDimScene, models.SummaryDimTime:
		// 合法
	default:
		utils.JSON(w, http.StatusBadRequest, -1, "无效的汇总维度(region/school/user/model/scene/time)", nil)
		return
	}

	// 解析时间范围（YYYY-MM-DD）。To 含全天：加一天，repo 用 < 严格小于。
	var fromT, toT time.Time
	if s := q.Get("from"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
			fromT = t
		}
	}
	if s := q.Get("to"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
			toT = t.AddDate(0, 0, 1) // 含当天:加一天,repo 用 created_at < toT
		}
	}

	scope := h.resolveScope(r)

	in := &services.ConsumptionSummaryInput{
		Dimension: dimension,
		From:      fromT,
		To:        toT,
	}

	// ---------- 下钻过滤的 scope 内二次校验 ----------
	// user_filter:必须在 scope.UserIDs 内(admin 的 UserIDs=nil 放行任意)
	if uf := q.Get("user_filter"); uf != "" {
		if userIDAllowedInScope(scope, uf) {
			in.UserFilter = uf
		} else {
			// 越权尝试:传了不在范围内的 user_id → 按空集返回(SchoolMember 设空切片触发 1=0)
			in.SchoolMember = []string{}
		}
	}

	// school_filter:取该校成员,与 scope.UserIDs 求交集后传入(admin 不收窄)
	if sf := q.Get("school_filter"); sf != "" && in.UserFilter == "" {
		memberIDs, err := repository.ListSchoolMemberIDs(r.Context(), sf)
		if err != nil {
			utils.JSON(w, http.StatusInternalServerError, -1, "查询学校成员失败", nil)
			return
		}
		in.SchoolMember = intersectWithScopeUsers(scope, memberIDs)
	}

	resp, err := h.tokenService.GetConsumptionSummary(r.Context(), in, scope)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, -1, "获取汇总报告失败", nil)
		return
	}
	utils.JSON(w, http.StatusOK, 0, "", resp)
}

// userIDAllowedInScope 判断某 user_id 是否在 scope 的用户白名单内
// admin(UserIDs==nil) 放行任意;空切片恒 false;否则成员判定
func userIDAllowedInScope(scope *services.TokenScope, userID string) bool {
	if scope == nil {
		return false
	}
	if scope.UserIDs == nil {
		return true // admin
	}
	for _, id := range scope.UserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// intersectWithScopeUsers 求 memberIDs 与 scope.UserIDs 的交集
// admin(UserIDs==nil) 不收窄,直接返回 memberIDs;
// scope 为空切片 → 返回空切片(触发 repo 的 1=0 空集);
// 否则返回既是学校成员、又在 scope 内的 user_id。
func intersectWithScopeUsers(scope *services.TokenScope, memberIDs []string) []string {
	if scope == nil {
		return []string{}
	}
	if scope.UserIDs == nil {
		return memberIDs // admin:学校成员全放行
	}
	scopeSet := make(map[string]struct{}, len(scope.UserIDs))
	for _, id := range scope.UserIDs {
		scopeSet[id] = struct{}{}
	}
	result := make([]string, 0, len(memberIDs))
	for _, id := range memberIDs {
		if _, ok := scopeSet[id]; ok {
			result = append(result, id)
		}
	}
	return result // 可能为空切片 → repo 按空集处理(看不到任何数据)
}
