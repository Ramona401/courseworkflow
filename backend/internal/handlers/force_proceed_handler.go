package handlers

// ==================== v38新增：Translator FAIL后强制推进到Generator ====================
//
// HTTP接口：POST /api/v1/pipelines/{id}/force-proceed
//
// 使用场景：
//   Translator-Reviewer循环FAIL（如评分8.9差0.1不达标），但方案质量已足够好。
//   操作员在前端查看FAIL原因和Translator输出后，判断方案可用，
//   点击"确认使用当前方案"按钮调用此接口，跳过Translator重跑，直接启动Generator。
//
// 请求：无请求体（POST确认操作）
// 响应：更新后的Pipeline详情（status变为running）
// 权限：admin / senior_operator / operator

import (
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/utils"
)

// ForceProceed POST /api/v1/pipelines/{id}/force-proceed
// 当Translator步骤FAIL但方案质量可接受时，操作员确认使用当前方案，直接启动Generator
//
// 前端调用时机：
//   在Translator步骤面板中，当步骤状态为failed且有有效输出时，
//   显示FAIL原因分析和"确认使用当前方案→启动Generator"按钮。
//   操作员点击按钮后弹出确认对话框，确认后调用此接口。
func (h *PipelineHandler) ForceProceed(w http.ResponseWriter, r *http.Request) {
	// 仅接受POST请求
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 从路径中提取Pipeline ID（路径格式：/api/v1/pipelines/{id}/force-proceed）
	id := extractPipelineIDWithSuffix(r.URL.Path, "/force-proceed")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	// 获取调用者角色（用于权限校验）
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 调用服务层执行强制推进
	resp, err := h.pipelineService.ForceProceedToGenerator(id, claims.Role)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}
