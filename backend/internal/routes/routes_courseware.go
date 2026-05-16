package routes

import (
	"net/http"
	"strings"

	"tedna/internal/handlers"
	"tedna/internal/middleware"
)

// ==================== 课件工坊路由注册 ====================
// PRD v1.0: 课件CRUD + 页面操作 + 状态流转 + 风格模板 + 组件库
// Phase 2: 种子数据填充 + admin模板管理
// Phase 3: 课件索引AI生成 + SSE流式推送 + 删除单页
// v2.1修复: courseware-components 子路由404问题（需注册带/的路径）

// registerCoursewareRoutes 注册课件工坊全部路由
func registerCoursewareRoutes(
	mux *http.ServeMux,
	authMW func(http.Handler) http.Handler,
	adminOnly func(http.Handler) http.Handler,
	cwHandler *handlers.CoursewareHandler,
	cwCompHandler *handlers.CWComponentHandler,
	cwSeedHandler *handlers.CWSeedHandler,
	cwIndexHandler *handlers.CoursewareIndexHandler,
) {
	// ==================== 课件SSE（内部Token验证，不走authMW） ====================

	// GET /api/v1/coursewares/{id}/index-stream?token=xxx — SSE订阅索引生成进度
	// 注意：EventSource不支持Header，token通过URL参数传递，handler内部验证
	mux.HandleFunc("/api/v1/sse/courseware/", cwIndexHandler.IndexStream)

	// ==================== 课件CRUD（登录即可） ====================

	// /api/v1/coursewares 精确匹配（列表+创建）
	// /api/v1/coursewares/ 通配子路由（详情/更新/删除/页面操作/状态流转/索引生成/SSE）
	cwMux := middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/v1/coursewares" || path == "/api/v1/coursewares/" {
			switch r.Method {
			case http.MethodGet:
				cwHandler.ListCoursewares(w, r)
			case http.MethodPost:
				cwHandler.CreateCourseware(w, r)
			default:
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		dispatchCoursewareSubRoutes(w, r, cwHandler, cwIndexHandler)
	}), authMW)
	mux.Handle("/api/v1/coursewares", cwMux)
	mux.Handle("/api/v1/coursewares/", cwMux)

	// ==================== 风格模板查询（登录即可） ====================

	// GET /api/v1/courseware-templates — 模板列表
	mux.Handle("/api/v1/courseware-templates", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwHandler.ListTemplates(w, r)
	}), authMW))

	// GET /api/v1/courseware-templates/{id}/preview — 模板预览
	mux.Handle("/api/v1/courseware-templates/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/preview") {
			cwHandler.GetTemplatePreview(w, r)
			return
		}
		http.Error(w, `{"code":-1,"message":"未找到路由"}`, http.StatusNotFound)
	}), authMW))

	// ==================== 课件组件库（admin管理，登录可查询） ====================

	// /api/v1/courseware-components 精确匹配（列表+创建）
	// /api/v1/courseware-components/ 通配子路由（详情/更新/删除/索引压缩）
	compMux := middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/v1/courseware-components" || path == "/api/v1/courseware-components/" {
			switch r.Method {
			case http.MethodGet:
				cwCompHandler.ListComponents(w, r)
			case http.MethodPost:
				cwCompHandler.CreateComponent(w, r)
			default:
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
		dispatchCWComponentSubRoutes(w, r, cwCompHandler)
	}), authMW)
	mux.Handle("/api/v1/courseware-components", compMux)
	mux.Handle("/api/v1/courseware-components/", compMux)

	// POST /api/v1/courseware-components/match — 组件匹配测试（登录即可）
	mux.Handle("/api/v1/courseware-components/match", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwCompHandler.MatchComponents(w, r)
	}), authMW))

	// ==================== 种子数据填充（admin） ====================

	// POST /api/v1/admin/courseware-seed — 一键填充种子数据
	mux.Handle("/api/v1/admin/courseware-seed", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cwSeedHandler.SeedAll(w, r)
	}), authMW, adminOnly))

	// ==================== Admin模板管理 ====================

	// POST /api/v1/admin/courseware-templates — 创建模板
	mux.Handle("/api/v1/admin/courseware-templates", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/v1/admin/courseware-templates" || path == "/api/v1/admin/courseware-templates/" {
			switch r.Method {
			case http.MethodPost:
				cwSeedHandler.CreateTemplate(w, r)
			default:
				http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
			}
			return
		}
	}), authMW, adminOnly))

	// PUT/DELETE /api/v1/admin/courseware-templates/{id}
	mux.Handle("/api/v1/admin/courseware-templates/", middleware.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			cwSeedHandler.UpdateTemplate(w, r)
		case http.MethodDelete:
			cwSeedHandler.DeleteTemplate(w, r)
		default:
			http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}), authMW, adminOnly))
}

// dispatchCoursewareSubRoutes 课件子路由分发
// Phase 3 新增: generate-index / index-stream / DELETE pages/{num}
func dispatchCoursewareSubRoutes(w http.ResponseWriter, r *http.Request, h *handlers.CoursewareHandler, idxH *handlers.CoursewareIndexHandler) {
	path := r.URL.Path

	// Phase 3: POST /api/v1/coursewares/{id}/generate-index — 触发AI生成索引
	if strings.HasSuffix(path, "/generate-index") {
		idxH.GenerateIndex(w, r)
		return
	}
	// Phase 3: GET /api/v1/coursewares/{id}/index-stream — SSE订阅索引进度
	if strings.HasSuffix(path, "/index-stream") {
		idxH.IndexStream(w, r)
		return
	}
	// POST /api/v1/coursewares/{id}/confirm-index
	if strings.HasSuffix(path, "/confirm-index") {
		h.ConfirmIndex(w, r)
		return
	}
	// PUT /api/v1/coursewares/{id}/style
	if strings.HasSuffix(path, "/style") {
		h.SaveStyle(w, r)
		return
	}
	// POST /api/v1/coursewares/{id}/confirm
	if strings.HasSuffix(path, "/confirm") && !strings.HasSuffix(path, "/confirm-index") {
		h.ConfirmCourseware(w, r)
		return
	}
	// PUT /api/v1/coursewares/{id}/pages/reorder
	if strings.HasSuffix(path, "/pages/reorder") {
		h.ReorderPages(w, r)
		return
	}
	// GET/POST /api/v1/coursewares/{id}/pages
	if strings.HasSuffix(path, "/pages") || strings.HasSuffix(path, "/pages/") {
		switch r.Method {
		case http.MethodGet:
			h.GetCoursewarePages(w, r)
		case http.MethodPost:
			h.AddPage(w, r)
		default:
			http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}
	// PUT/DELETE /api/v1/coursewares/{id}/pages/{num} — 更新或删除单页索引
	if strings.Contains(path, "/pages/") {
		switch r.Method {
		case http.MethodPut:
			h.UpdatePageIndex(w, r)
		case http.MethodDelete:
			// Phase 3 新增：删除单页
			idxH.DeletePage(w, r)
		default:
			http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
		}
		return
	}
	// GET/PUT/DELETE /api/v1/coursewares/{id} — 课件详情/更新/删除
	switch r.Method {
	case http.MethodGet:
		h.GetCourseware(w, r)
	case http.MethodPut:
		h.UpdateCourseware(w, r)
	case http.MethodDelete:
		h.DeleteCourseware(w, r)
	default:
		http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// dispatchCWComponentSubRoutes 课件组件子路由分发
func dispatchCWComponentSubRoutes(w http.ResponseWriter, r *http.Request, h *handlers.CWComponentHandler) {
	path := r.URL.Path

	// POST /api/v1/courseware-components/{id}/index — 触发索引压缩
	if strings.HasSuffix(path, "/index") {
		h.CompressIndex(w, r)
		return
	}
	// GET/PUT/DELETE /api/v1/courseware-components/{id}
	switch r.Method {
	case http.MethodGet:
		h.GetComponent(w, r)
	case http.MethodPut:
		h.UpdateComponent(w, r)
	case http.MethodDelete:
		h.DeleteComponent(w, r)
	default:
		http.Error(w, `{"code":-1,"message":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
