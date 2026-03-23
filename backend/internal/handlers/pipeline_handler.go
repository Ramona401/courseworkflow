package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== PipelineHandler ====================

// PipelineHandler Pipeline管理API处理器
type PipelineHandler struct {
	pipelineService *services.PipelineService
}

// NewPipelineHandler 创建Pipeline处理器实例
func NewPipelineHandler(pipelineService *services.PipelineService) *PipelineHandler {
	return &PipelineHandler{pipelineService: pipelineService}
}

// ==================== Dashboard 统计接口（P4.5-D新增）====================

// GetDashboardStats GET /api/v1/dashboard/stats
// 获取仪表盘统计数据（课程数/Pipeline各状态/达标数/AI消耗）
func (h *PipelineHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	stats, err := h.pipelineService.GetDashboardStats()
	if err != nil {
		utils.InternalError(w, "获取统计数据失败: "+err.Error())
		return
	}

	utils.Success(w, stats)
}

// ==================== Pipeline 接口 ====================

// CreatePipeline POST /api/v1/pipelines
// 创建新Pipeline（admin/operator可操作）
func (h *PipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 获取当前用户信息
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 解析请求体
	var req models.CreatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}

	// 创建Pipeline
	resp, err := h.pipelineService.CreatePipeline(&req, claims.UserID)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}

// ListPipelines GET /api/v1/pipelines
// 获取Pipeline列表（按角色过滤可见范围）
func (h *PipelineHandler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	resp, err := h.pipelineService.ListPipelines(claims.UserID, claims.Role)
	if err != nil {
		utils.InternalError(w, "获取Pipeline列表失败: "+err.Error())
		return
	}

	utils.Success(w, resp)
}

// GetPipelineDetail GET /api/v1/pipelines/{id}
// 获取Pipeline详情（含步骤列表）
func (h *PipelineHandler) GetPipelineDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	id := extractPipelineID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	resp, err := h.pipelineService.GetPipelineDetail(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}

// StartPipeline POST /api/v1/pipelines/{id}/start
// 启动Pipeline执行（P5-1改造：异步执行，立即返回running状态）
// 改造前：同步执行全链路45-55分钟，HTTP连接阻塞
// 改造后：立即返回{status:"running"}，goroutine后台执行，前端轮询刷新
func (h *PipelineHandler) StartPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	id := extractPipelineIDWithSuffix(r.URL.Path, "/start")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	resp, err := h.pipelineService.StartPipeline(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}

// CancelPipeline POST /api/v1/pipelines/{id}/cancel
// 取消Pipeline（仅pending或running状态可取消）
func (h *PipelineHandler) CancelPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	id := extractPipelineIDWithSuffix(r.URL.Path, "/cancel")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	if err := h.pipelineService.CancelPipeline(id); err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"message": "Pipeline已取消",
	})
}

// DeletePipeline DELETE /api/v1/pipelines/{id}
// 删除Pipeline（运行中的不可删除）
func (h *PipelineHandler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持DELETE请求")
		return
	}

	id := extractPipelineID(r.URL.Path)
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	if err := h.pipelineService.DeletePipeline(id); err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"message": "Pipeline已删除",
	})
}

// GetSteps GET /api/v1/pipelines/{id}/steps
// 获取Pipeline的步骤列表
func (h *PipelineHandler) GetSteps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	id := extractPipelineIDWithSuffix(r.URL.Path, "/steps")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	// 获取完整详情（含步骤列表）
	resp, err := h.pipelineService.GetPipelineDetail(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	// 只返回步骤列表部分
	utils.Success(w, map[string]interface{}{
		"pipeline_id": resp.ID,
		"course_code": resp.CourseCode,
		"steps":       resp.Steps,
		"total":       len(resp.Steps),
	})
}

// GetStepDetail GET /api/v1/pipelines/{id}/steps/{name}
// 获取指定步骤的详细信息（含完整step_data）
func (h *PipelineHandler) GetStepDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	// 从路径中提取pipeline_id和step_name
	pipelineID, stepName := extractPipelineIDAndStepName(r.URL.Path)
	if pipelineID == "" || stepName == "" {
		utils.BadRequest(w, "缺少Pipeline ID或步骤名称")
		return
	}

	resp, err := h.pipelineService.GetStepDetail(pipelineID, stepName)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}

// GetEvalRounds GET /api/v1/pipelines/{id}/eval-rounds
// 获取Pipeline的评估轮次详情列表（含每轮AI原始输出）
func (h *PipelineHandler) GetEvalRounds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}

	id := extractPipelineIDWithSuffix(r.URL.Path, "/eval-rounds")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	rounds, err := h.pipelineService.GetEvalRounds(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, map[string]interface{}{
		"pipeline_id": id,
		"rounds":      rounds,
		"total":       len(rounds),
	})
}

// ==================== P4.5-C 审核决策接口 ====================

// GetGeneratedPages GET /api/v1/pipelines/{id}/pages
// 获取Pipeline的生成页面列表（含完整HTML内容，用于审核预览）
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

// UpdatePageDecision PUT /api/v1/pipelines/{id}/pages/{pageNumber}/decision
// 更新单页审核决策（approve/reject/edit）
func (h *PipelineHandler) UpdatePageDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持PUT请求")
		return
	}

	// 从路径提取pipeline_id和pageNumber
	pipelineID, pageNumber := extractPipelineIDAndPageNumber(r.URL.Path)
	if pipelineID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "缺少Pipeline ID或页码")
		return
	}

	// 解析请求体
	var req struct {
		Decision  string  `json:"decision"`  // approve / reject / edit
		FinalHTML *string `json:"final_html"` // edit时提供修改后的HTML
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

// FinalizePipeline POST /api/v1/pipelines/{id}/finalize
// 定稿归档Pipeline（所有页面必须已决策）
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

	utils.Success(w, map[string]interface{}{
		"message": "Pipeline已定稿归档",
	})
}

// ==================== P4.5-D 快捷通过接口 ====================

// MarkPassed POST /api/v1/pipelines/{id}/mark-passed
// 快捷通过Pipeline（评估达标直接标记为finalized，跳过后续步骤）
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

	utils.Success(w, map[string]interface{}{
		"message": "Pipeline已快捷通过并归档",
	})
}

// ==================== P4.5-E-2 AI快修接口 ====================

// AIFixPage POST /api/v1/pipelines/{id}/pages/{n}/ai-fix
// 审核员在全屏预览中输入修改指令，AI基于当前HTML修复并返回新HTML
func (h *PipelineHandler) AIFixPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 从路径提取pipeline_id和pageNumber
	pipelineID, pageNumber := extractPipelineIDAndPageNumberForAIFix(r.URL.Path)
	if pipelineID == "" || pageNumber <= 0 {
		utils.BadRequest(w, "缺少Pipeline ID或页码")
		return
	}

	// 解析请求体
	var req struct {
		FixInstruction string `json:"fix_instruction"` // 修复指令文本
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.FixInstruction == "" {
		utils.BadRequest(w, "修复指令不能为空")
		return
	}

	// 调用服务层执行AI快修
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

// ==================== P4.6-2 验收接口 ====================

// VerifyPipeline POST /api/v1/pipelines/{id}/verify
// 手动触发验收评估（finalized状态的Pipeline，收集HTML→索引生成→评估→判定通过/失败）
// P4.6-2新增：验收流程入口接口
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

	// 调用服务层执行验收（可能耗时较长：索引生成+评估两次AI调用）
	resp, err := h.pipelineService.VerifyPipeline(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}

// ==================== P4.6-4 批量验收接口 ====================

// BatchVerify POST /api/v1/pipelines/batch-verify
// 手动触发批量验收：扫描所有finalized状态的Pipeline，逐个异步触发验收
// P4.6-4新增：批量验收入口接口（仅admin可操作）
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

// ==================== 路径解析辅助函数 ====================

// extractPipelineIDAndPageNumberForAIFix 从ai-fix路径提取Pipeline ID和页码
// P4.5-E-2新增：路径格式 /api/v1/pipelines/{id}/pages/{n}/ai-fix
func extractPipelineIDAndPageNumberForAIFix(path string) (string, int) {
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
	aiFixIdx := strings.Index(afterPages, "/ai-fix")
	if aiFixIdx < 0 {
		return pipelineID, 0
	}
	pageNumStr := afterPages[:aiFixIdx]
	pageNum, err := strconv.Atoi(pageNumStr)
	if err != nil || pageNum <= 0 {
		return pipelineID, 0
	}
	return pipelineID, pageNum
}

// extractPipelineID 从路径提取Pipeline ID
func extractPipelineID(path string) string {
	path = strings.TrimSuffix(path, "/")
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 0 {
		return ""
	}
	id := path[lastSlash+1:]
	if id == "" || id == "pipelines" {
		return ""
	}
	return id
}

// extractPipelineIDWithSuffix 从带后缀的路径提取Pipeline ID
func extractPipelineIDWithSuffix(path string, suffix string) string {
	idx := strings.LastIndex(path, suffix)
	if idx <= 0 {
		return ""
	}
	path = path[:idx]
	return extractPipelineID(path)
}

// extractPipelineIDAndStepName 从路径提取Pipeline ID和步骤名称
func extractPipelineIDAndStepName(path string) (string, string) {
	stepsIdx := strings.Index(path, "/steps/")
	if stepsIdx < 0 {
		return "", ""
	}
	stepName := strings.TrimSuffix(path[stepsIdx+len("/steps/"):], "/")
	if stepName == "" {
		return "", ""
	}
	beforeSteps := path[:stepsIdx]
	pipelineID := extractPipelineID(beforeSteps)
	return pipelineID, stepName
}

// extractPipelineIDAndPageNumber 从路径提取Pipeline ID和页码
// P4.5-C新增
func extractPipelineIDAndPageNumber(path string) (string, int) {
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
	decisionIdx := strings.Index(afterPages, "/decision")
	if decisionIdx < 0 {
		return pipelineID, 0
	}
	pageNumStr := afterPages[:decisionIdx]
	pageNum, err := strconv.Atoi(pageNumStr)
	if err != nil || pageNum <= 0 {
		return pipelineID, 0
	}
	return pipelineID, pageNum
}

// ==================== 错误处理 ====================

// handlePipelineError Pipeline统一错误处理
// P4.6-2更新：新增验收相关错误处理
func handlePipelineError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	// 参数错误 (400)
	case err == services.ErrPipelineCourseRequired,
		err == services.ErrInvalidDecision:
		utils.BadRequest(w, errMsg)

	// 未找到 (404)
	case err == services.ErrPipelineNotFound,
		err == services.ErrPipelineCourseNotFound,
		err == services.ErrPageNotFound:
		utils.Fail(w, http.StatusNotFound, errMsg)

	// 状态冲突 (409)
	case strings.Contains(errMsg, "已有运行中的Pipeline"),
		err == services.ErrPipelineNotPending,
		err == services.ErrPipelineNotCancellable,
		err == services.ErrPipelineNotDeletable,
		err == services.ErrPipelineNotReviewable,
		err == services.ErrFinalizeIncomplete,
		err == services.ErrMarkPassedNotAllowed,
		err == services.ErrVerifyNotFinalized,
		err == services.ErrEngineQueueFull:
		utils.Fail(w, http.StatusConflict, errMsg)

	// P4.5-E-2: AI快修失败 (502)
	case err == services.ErrAIFixFailed:
		utils.Fail(w, http.StatusBadGateway, errMsg)

	// P4.6-2: 验收AI调用失败 (502)
	case err == services.ErrVerifyIndexGenFailed,
		err == services.ErrVerifyEvalFailed:
		utils.Fail(w, http.StatusBadGateway, errMsg)

	// P4.5-D: 快捷通过未达标 (422)
	case err == services.ErrMarkPassedNotMet:
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	// P4.6-2: 验收前置条件不满足 (422)
	case err == services.ErrVerifyNoPages,
		err == services.ErrVerifyNoValidHTML,
		err == services.ErrVerifyIndexTooShort,
		err == services.ErrVerifyScoreExtractFail,
		err == services.ErrVerifyScannerNotDone:
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	// P4.6-2: 验收提示词缺失 (422)
	case err == services.ErrVerifyPromptGMissing,
		err == services.ErrVerifyPromptBMissing,
		err == services.ErrVerifyDictMissing:
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	// dbCheck验证失败 -> 返回422
	case err == services.ErrDbCheckIndexMissing,
		err == services.ErrDbCheckIndexTooShort,
		err == services.ErrDbCheckIndexHashMismatch:
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	// 包含"尚有未决策"的文字（ErrFinalizeIncomplete带详情）
	case strings.Contains(errMsg, "尚有未决策"):
		utils.Fail(w, http.StatusConflict, errMsg)

	// 包含"edit决策必须提供"的文字
	case strings.Contains(errMsg, "edit决策必须提供"):
		utils.BadRequest(w, errMsg)

	// 包含"没有生成页面"的文字
	case strings.Contains(errMsg, "没有生成页面"):
		utils.BadRequest(w, errMsg)

	// 包含"AI快修失败"的文字（AIFixPage带详情）
	case strings.Contains(errMsg, "AI快修失败"):
		utils.Fail(w, http.StatusBadGateway, errMsg)

	// 包含"评估未达标"的文字（MarkPassed带详情）
	case strings.Contains(errMsg, "评估未达标"):
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	// P4.6-2: 包含验收相关错误文字的模糊匹配
	case strings.Contains(errMsg, "索引生成器AI调用失败"),
		strings.Contains(errMsg, "验收评估AI调用失败"):
		utils.Fail(w, http.StatusBadGateway, errMsg)

	case strings.Contains(errMsg, "不是finalized状态"):
		utils.Fail(w, http.StatusConflict, errMsg)

	// 其他服务器错误 (500)
	default:
		utils.InternalError(w, errMsg)
	}
}
