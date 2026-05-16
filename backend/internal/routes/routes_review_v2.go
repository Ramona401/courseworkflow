package routes

// routes_review_v2.go — 多级审核+抽查路由注册
//
// v127.2 新增：GET /reviews/reviewed 已审核记录列表
//
// 路由使用 middleware.Chain 模式，与 routes.go 一致

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// registerReviewV2Routes 注册多级审核和区域抽查相关路由
//
// 参数签名与 routes.go 调用一致：
//   authMW, adminOnly, adminOrInspector, adminOrSchoolAdmin 都是 func(http.Handler) http.Handler
func registerReviewV2Routes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	adminOrInspector func(http.Handler) http.Handler,
	adminOrSchoolAdmin func(http.Handler) http.Handler,
	rv2 *handlers.ReviewV2Handler,
	insp *handlers.InspectionHandler,
) {
	// ==================== 审核路由（登录即可） ====================

	// 通配器：/api/v1/reviews/ 下的所有请求
	mux.Handle("/api/v1/reviews/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		rest := strings.TrimPrefix(path, "/api/v1/reviews/")

		// 固定路径优先匹配
		switch {
		case rest == "pending" && r.Method == http.MethodGet:
			rv2.GetPendingReviews(w, r)
			return
		case rest == "reviewed" && r.Method == http.MethodGet:
			rv2.GetReviewedRecords(w, r)
			return
		case rest == "stats" && r.Method == http.MethodGet:
			rv2.GetReviewStats(w, r)
			return
		}

		// 动态路径：{plan_id}/l1, {plan_id}/l2, {plan_id}/history
		if strings.HasSuffix(path, "/l1") {
			rv2.ReviewL1(w, r)
			return
		}
		if strings.HasSuffix(path, "/l2") {
			rv2.ReviewL2(w, r)
			return
		}
		if strings.HasSuffix(path, "/history") {
			rv2.GetReviewHistory(w, r)
			return
		}

		http.NotFound(w, r)
	}), authMW))

	// ==================== 审核配置（学校管理员+admin） ====================

	mux.Handle("/api/v1/review-config", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			rv2.GetReviewConfig(w, r)
		case http.MethodPut:
			rv2.UpdateReviewConfig(w, r)
		default:
			http.NotFound(w, r)
		}
	}), authMW, adminOrSchoolAdmin))

	// ==================== 抽查路由（admin + district_inspector） ====================

	mux.Handle("/api/v1/inspections/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		rest := strings.TrimPrefix(path, "/api/v1/inspections/")

		// 固定路径优先
		switch {
		case rest == "batch-sample" && r.Method == http.MethodPost:
			insp.BatchSample(w, r)
			return
		case rest == "stats" && r.Method == http.MethodGet:
			insp.GetInspectionStats(w, r)
			return
		}

		// 动态路径
		if strings.HasSuffix(path, "/review") {
			insp.ReviewInspection(w, r)
			return
		}
		if strings.HasSuffix(path, "/assign") {
			insp.AssignInspector(w, r)
			return
		}

		// GET /inspections/{id}
		if rest != "" && !strings.Contains(rest, "/") {
			insp.GetInspection(w, r)
			return
		}

		http.NotFound(w, r)
	}), authMW, adminOrInspector))

	// GET /api/v1/inspections（无尾斜杠）
	mux.Handle("/api/v1/inspections", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			insp.ListInspections(w, r)
			return
		}
		http.NotFound(w, r)
	}), authMW, adminOrInspector))

	// ==================== 教研员管理（admin only） ====================

	mux.Handle("/api/v1/district-inspectors/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			insp.DeleteDistrictInspector(w, r)
			return
		}
		http.NotFound(w, r)
	}), authMW, adminOnly))

	mux.Handle("/api/v1/district-inspectors", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			insp.ListDistrictInspectors(w, r)
		case http.MethodPost:
			insp.CreateDistrictInspector(w, r)
		default:
			http.NotFound(w, r)
		}
	}), authMW, adminOnly))
}
