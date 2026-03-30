package handlers

// pipeline_handler_review.go — Pipeline审核定稿类接口
//
// 职责：
//   - 生成页面查询与决策
//   - 直接定稿（admin专用）
//   - 二级审批：提交定稿/确认定稿/退回重审
//   - 快捷通过（评估达标时跳过审核）
//   - AI快修页面
//   - 验收：单个验收/批量验收

import (
	"encoding/json"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 页面查询与决策 ====================

// GetGeneratedPages GET /api/v1/pipelines/{id}/pages
func (h *PipelineHandler) GetGeneratedPages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/pages")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}
	pages, err := h.pipelineService.GetGeneratedPages(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{
		"pipeline_id": id,
		"pages":       pages,
		"total":       len(pages),
	})
}

// UpdatePageDecision PUT /api/v1/pipelines/{id}/pages/{num}/decision
func (h *PipelineHandler) UpdatePageDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}
	pipelineID, pageNumber := extractPipelineIDAndPageNumber(r.URL.Path)
	if pipelineID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "缺少Pipeline ID或页码")
		return
	}
	var req struct {
		Decision  string  `json:"decision"`
		FinalHTML *string `json:"final_html"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if err := h.pipelineService.UpdatePageDecision(pipelineID, pageNumber, req.Decision, req.FinalHTML); err != nil {
		handlePipelineError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{
		"message":     "决策已更新",
		"page_number": pageNumber,
		"decision":    req.Decision,
	})
}

// ==================== 定稿操作 ====================

// FinalizePipeline POST /api/v1/pipelines/{id}/finalize
// 直接定稿（admin专用，跳过二级审批）
func (h *PipelineHandler) FinalizePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/finalize")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}
	if err := h.pipelineService.FinalizePipeline(id); err != nil {
		handlePipelineError(w, err)
		return
	}
	// 写入审计日志：直接定稿
	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, repository.ActionDirectFinalize,
			map[string]interface{}{"pipeline_id": id, "operator": claims.Username},
			repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, map[string]interface{}{"message": "Pipeline已定稿归档"})
}

// ==================== 二级审批接口 ====================

// SubmitFinalize POST /api/v1/pipelines/{id}/submit-finalize
// 操作员提交定稿申请，进入 pending_finalize 状态
func (h *PipelineHandler) SubmitFinalize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/submit-finalize")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}
	if err := h.pipelineService.SubmitFinalize(id); err != nil {
		handlePipelineError(w, err)
		return
	}
	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, repository.ActionSubmitFinalize,
			map[string]interface{}{"pipeline_id": id, "operator": claims.Username},
			repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, map[string]interface{}{"message": "已提交定稿申请，等待超级审核员确认"})
}

// ConfirmFinalize POST /api/v1/pipelines/{id}/confirm-finalize
// 高级操作员确认定稿，Pipeline进入 finalized 状态
func (h *PipelineHandler) ConfirmFinalize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/confirm-finalize")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}
	if err := h.pipelineService.ConfirmFinalize(id); err != nil {
		handlePipelineError(w, err)
		return
	}
	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, repository.ActionConfirmFinalize,
			map[string]interface{}{"pipeline_id": id, "operator": claims.Username},
			repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, map[string]interface{}{"message": "定稿已确认，Pipeline进入finalized状态"})
}

// RejectFinalize POST /api/v1/pipelines/{id}/reject-finalize
// 高级操作员退回定稿申请，Pipeline返回待审核状态
func (h *PipelineHandler) RejectFinalize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/reject-finalize")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if err := h.pipelineService.RejectFinalize(id, req.Reason); err != nil {
		handlePipelineError(w, err)
		return
	}
	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, repository.ActionRejectFinalize,
			map[string]interface{}{"pipeline_id": id, "operator": claims.Username, "reason": req.Reason},
			repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, map[string]interface{}{"message": "已退回重审，Pipeline返回待审核状态"})
}

// ==================== 快捷通过 ====================

// MarkPassed POST /api/v1/pipelines/{id}/mark-passed
// 评估均分达标（≥9.0）时快捷通过，跳过完整审核流程直接归档
func (h *PipelineHandler) MarkPassed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/mark-passed")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}
	if err := h.pipelineService.MarkPassed(id); err != nil {
		handlePipelineError(w, err)
		return
	}
	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, repository.ActionMarkPassed,
			map[string]interface{}{"pipeline_id": id, "operator": claims.Username},
			repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, map[string]interface{}{"message": "Pipeline已快捷通过并归档"})
}

// ==================== AI快修 ====================

// AIFixPage POST /api/v1/pipelines/{id}/pages/{num}/ai-fix
// 对指定页面执行AI快修，根据修复指令重新生成HTML
func (h *PipelineHandler) AIFixPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	pipelineID, pageNumber := extractPipelineIDAndPageNumberForAIFix(r.URL.Path)
	if pipelineID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "缺少Pipeline ID或页码")
		return
	}
	var req struct {
		FixInstruction string `json:"fix_instruction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.FixInstruction == "" {
		utils.BadRequest(w, "修复指令不能为空")
		return
	}
	newHTML, err := h.pipelineService.AIFixPage(pipelineID, pageNumber, req.FixInstruction)
	if err != nil {
		handlePipelineError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{
		"message":     "AI快修完成",
		"page_number": pageNumber,
		"new_html":    newHTML,
		"html_length": len(newHTML),
	})
}

// ==================== 验收接口 ====================

// VerifyPipeline POST /api/v1/pipelines/{id}/verify
// 对已定稿的Pipeline执行验收评估（索引压缩+Evaluator评估）
func (h *PipelineHandler) VerifyPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/verify")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}
	resp, err := h.pipelineService.VerifyPipeline(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}
	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, repository.ActionVerify,
			map[string]interface{}{"pipeline_id": id, "operator": claims.Username},
			repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, resp)
}

// BatchVerify POST /api/v1/pipelines/batch-verify
// 批量验收所有finalized状态的Pipeline（夜间调度也调用此逻辑）
func (h *PipelineHandler) BatchVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	result, err := h.pipelineService.BatchVerify()
	if err != nil {
		utils.InternalError(w, "批量验收失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}
