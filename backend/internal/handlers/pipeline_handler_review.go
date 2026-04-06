package handlers

// pipeline_handler_review.go — Pipeline审核定稿类接口
//
// v68变更：AIFixPage接口增加reference_pages参数+返回fix_summary字段

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/services"
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
	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, repository.ActionDirectFinalize,
			map[string]interface{}{"pipeline_id": id, "operator": claims.Username},
			repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, map[string]interface{}{"message": "Pipeline已定稿归档"})
}

// ==================== 二级审批接口 ====================

// SubmitFinalize POST /api/v1/pipelines/{id}/submit-finalize
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
// v68增强：接收reference_pages参数（参考页码数组）+ 返回fix_summary修改说明
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

	// v68增强：请求体增加reference_pages可选字段
	var req struct {
		FixInstruction string `json:"fix_instruction"`
		ReferencePages []int  `json:"reference_pages"` // v68新增：参考页码数组（可选）
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.FixInstruction == "" {
		utils.BadRequest(w, "修复指令不能为空")
		return
	}

	// 调用服务层（传入参考页码）
	result, err := h.pipelineService.AIFixPage(pipelineID, pageNumber, req.FixInstruction, req.ReferencePages)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"message":     "AI快修完成",
		"page_number": pageNumber,
		"new_html":    result.NewHTML,
		"html_length": result.HTMLLength,
		"fix_summary": result.FixSummary,
	})
}


// ==================== 回滚接口（v68新增）====================

// RollbackPageHTML POST /api/v1/pipelines/{id}/pages/{num}/rollback
// 将指定页面的HTML回滚到上一个历史版本（撤销最近一次编辑/AI快修）
func (h *PipelineHandler) RollbackPageHTML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	pipelineID, pageNumber := extractPipelineIDAndPageNumberForRollback(r.URL.Path)
	if pipelineID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "缺少Pipeline ID或页码")
		return
	}

	restoredHTML, remainingCount, err := repository.RollbackPageHTML(pipelineID, pageNumber)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"message":          "已回滚到上一版本",
		"page_number":      pageNumber,
		"restored_html":    restoredHTML,
		"html_length":      len(restoredHTML),
		"remaining_history": remainingCount,
	})
}

// ==================== 验收接口 ====================

// VerifyPipeline POST /api/v1/pipelines/{id}/verify
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

// PublishPipeline POST /api/v1/pipelines/{id}/publish
func (h *PipelineHandler) PublishPipeline(w http.ResponseWriter, r *http.Request) {
	id := extractPipelineIDWithSuffix(r.URL.Path, "/publish")
	if id == "" {
		utils.BadRequest(w, "无效的Pipeline ID")
		return
	}

	if err := h.pipelineService.PublishPipeline(id); err != nil {
		handlePipelineError(w, err)
		return
	}

	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, "pipeline.publish",
			map[string]interface{}{"pipeline_id": id}, r.RemoteAddr)
	}

	utils.Success(w, map[string]string{
		"message": "已确认发布至课程平台",
		"status":  "published",
	})
}

// ==================== 单页HTML按需加载（v69新增，编号8方案2）====================

// GetSinglePageHTML GET /api/v1/pipelines/{id}/pages/{num}/html
// v69新增：审核页HTML懒加载——前端选中页面时按需加载单页完整HTML
func (h *PipelineHandler) GetSinglePageHTML(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	pipelineID, pageNumber := extractPipelineIDAndPageNumberForHTML(r.URL.Path)
	if pipelineID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "缺少Pipeline ID或页码")
		return
	}
	page, err := h.pipelineService.GetSinglePageHTML(pipelineID, pageNumber)
	if err != nil {
		handlePipelineError(w, err)
		return
	}
	utils.Success(w, page)
}

// GetGeneratedPagesLightweight GET /api/v1/pipelines/{id}/pages-meta
// v69新增：审核页首次加载只获取轻量元数据列表（不含HTML），大幅减少传输数据量
func (h *PipelineHandler) GetGeneratedPagesLightweight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/pages-meta")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}
	pages, err := h.pipelineService.GetGeneratedPagesLightweight(id)
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

// ==================== AI快修流式SSE接口（v69新增，编号5）====================

// AIFixPageStream POST /api/v1/pipelines/{id}/pages/{num}/ai-fix-stream
// v69新增：AI快修流式返回——通过SSE逐token推送AI输出，前端实时展示
// SSE事件格式：
//   event: chunk   data: {"content":"..."}     — AI输出的增量token
//   event: done    data: {"new_html":"...","fix_summary":"...","html_length":N}  — 完成
//   event: error   data: {"message":"..."}     — 错误
func (h *PipelineHandler) AIFixPageStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	pipelineID, pageNumber := extractPipelineIDAndPageNumberForAIFixStream(r.URL.Path)
	if pipelineID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "缺少Pipeline ID或页码")
		return
	}

	var req struct {
		FixInstruction string `json:"fix_instruction"`
		ReferencePages []int  `json:"reference_pages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.FixInstruction == "" {
		utils.BadRequest(w, "修复指令不能为空")
		return
	}

	// 设置SSE响应头
	flusher, ok := w.(http.Flusher)
	if !ok {
		utils.InternalError(w, "服务器不支持流式响应")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // 禁止Nginx缓冲
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// 发送SSE事件的辅助函数
	sendSSE := func(event string, data string) {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		flusher.Flush()
	}

	// 发送连接成功事件
	sendSSE("connected", `{"status":"connected"}`)

	// 检测客户端断开
	ctx := r.Context()

	// 调用流式AI快修
	h.pipelineService.AIFixPageStream(
		pipelineID, pageNumber, req.FixInstruction, req.ReferencePages,
		// onChunk：逐token推送
		func(chunk string) {
			select {
			case <-ctx.Done():
				return // 客户端已断开
			default:
			}
			chunkJSON, _ := json.Marshal(map[string]string{"content": chunk})
			sendSSE("chunk", string(chunkJSON))
		},
		// onDone：完成，推送最终结果
		func(result *services.AIFixResult) {
			doneJSON, _ := json.Marshal(map[string]interface{}{
				"new_html":    result.NewHTML,
				"fix_summary": result.FixSummary,
				"html_length": result.HTMLLength,
			})
			sendSSE("done", string(doneJSON))
		},
		// onError：错误
		func(errMsg string) {
			errJSON, _ := json.Marshal(map[string]string{"message": errMsg})
			sendSSE("error", string(errJSON))
		},
	)
}

// extractPipelineIDAndPageNumberForAIFixStream 从流式AI快修路径提取Pipeline ID和页码（v69新增）
// 路径格式：/api/v1/pipelines/{id}/pages/{num}/ai-fix-stream
func extractPipelineIDAndPageNumberForAIFixStream(path string) (string, int) {
	pagesIdx := strings.Index(path, "/pages/")
	if pagesIdx < 0 {
		return "", 0
	}
	beforePages := path[:pagesIdx]
	pipelineID := extractPipelineID(beforePages)
	if pipelineID == "" {
		return "", 0
	}
	afterPages := path[pagesIdx+len("/pages/"):]
	suffixIdx := strings.Index(afterPages, "/ai-fix-stream")
	if suffixIdx < 0 {
		return pipelineID, 0
	}
	pageNumStr := afterPages[:suffixIdx]
	pageNum, err := strconv.Atoi(pageNumStr)
	if err != nil || pageNum <= 0 {
		return pipelineID, 0
	}
	return pipelineID, pageNum
}
