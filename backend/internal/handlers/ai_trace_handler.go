package handlers

// ai_trace_handler.go — AI调用追踪仪表盘HTTP处理器
//
// 职责：
//   提供AI调用统计仪表盘API，仅admin可访问
//
// 路由：
//   GET /api/v1/admin/ai-traces/dashboard — 获取仪表盘数据
//
// 被引用：
//   routes/routes_admin.go — 路由注册

import (
	"net/http"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// AITraceHandler AI调用追踪处理器
type AITraceHandler struct{}

// NewAITraceHandler 创建AI追踪处理器实例
func NewAITraceHandler() *AITraceHandler {
	return &AITraceHandler{}
}

// GetDashboard 获取AI调用仪表盘数据
// GET /api/v1/admin/ai-traces/dashboard
//
// 查询参数（均可选）：
//   date_from  — 起始日期（YYYY-MM-DD）
//   date_to    — 结束日期（YYYY-MM-DD）
//   scene_code — 按场景筛选
//   model      — 按模型筛选
//   status     — 按状态筛选（success/error/timeout）
//
// 返回：
//   TraceDashboard — 包含概览数字、按场景/模型聚合、每日趋势、最近错误
func (h *AITraceHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 解析查询参数
	query := r.URL.Query()
	params := models.TraceQueryParams{
		DateFrom:  query.Get("date_from"),
		DateTo:    query.Get("date_to"),
		SceneCode: query.Get("scene_code"),
		ModelUsed: query.Get("model"),
		Status:    query.Get("status"),
	}

	// 查询仪表盘数据
	dashboard, err := repository.GetTraceDashboard(r.Context(), params)
	if err != nil {
		utils.InternalError(w, "获取AI调用统计失败: "+err.Error())
		return
	}

	utils.Success(w, dashboard)
}
