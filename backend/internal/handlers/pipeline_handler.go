package handlers

// pipeline_handler.go — Pipeline处理器主文件
//
// 职责：
//   - PipelineHandler struct定义与构造
//   - Dashboard统计
//   - Pipeline基础操作：创建/列表/详情/启动/取消/删除
//   - 步骤查询：步骤列表/步骤详情/评估轮
//   - 路径解析辅助函数（extractPipelineID系列）
//   - 统一错误处理（handlePipelineError）
//
// 审核定稿类接口 → pipeline_handler_review.go
// 批量操作类接口 → pipeline_handler_batch.go

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

// ==================== Handler结构体 ====================

// PipelineHandler 处理所有Pipeline相关HTTP请求
type PipelineHandler struct {
	pipelineService *services.PipelineService
}

// NewPipelineHandler 构造函数
func NewPipelineHandler(pipelineService *services.PipelineService) *PipelineHandler {
	return &PipelineHandler{pipelineService: pipelineService}
}

// ==================== Dashboard统计 ====================

// GetDashboardStats GET /api/v1/dashboard/stats
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

// ==================== Pipeline基础操作 ====================

// CreatePipeline POST /api/v1/pipelines
func (h *PipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}
	var req models.CreatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	resp, err := h.pipelineService.CreatePipeline(&req, claims.UserID)
	if err != nil {
		handlePipelineError(w, err)
		return
	}
	utils.Success(w, resp)
}

// ListPipelines GET /api/v1/pipelines
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
	utils.Success(w, map[string]interface{}{"message": "Pipeline已取消"})
}

// DeletePipeline DELETE /api/v1/pipelines/{id}
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
	utils.Success(w, map[string]interface{}{"message": "Pipeline已删除"})
}

// ==================== 步骤查询 ====================

// GetSteps GET /api/v1/pipelines/{id}/steps
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
	resp, err := h.pipelineService.GetPipelineDetail(id)
	if err != nil {
		handlePipelineError(w, err)
		return
	}
	utils.Success(w, map[string]interface{}{
		"pipeline_id": resp.ID,
		"course_code": resp.CourseCode,
		"steps":       resp.Steps,
		"total":       len(resp.Steps),
	})
}

// GetStepDetail GET /api/v1/pipelines/{id}/steps/{name}
func (h *PipelineHandler) GetStepDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
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

// ==================== 路径解析辅助函数 ====================

// extractPipelineID 从路径末尾提取Pipeline ID
// 例：/api/v1/pipelines/abc123 → "abc123"
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

// extractPipelineIDWithSuffix 提取带固定后缀路径中的Pipeline ID
// 例：/api/v1/pipelines/abc123/start → "abc123"（suffix="/start"）
func extractPipelineIDWithSuffix(path string, suffix string) string {
	idx := strings.LastIndex(path, suffix)
	if idx <= 0 {
		return ""
	}
	path = path[:idx]
	return extractPipelineID(path)
}

// extractPipelineIDAndStepName 从步骤路径提取Pipeline ID和步骤名
// 例：/api/v1/pipelines/abc123/steps/scanner → "abc123", "scanner"
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

// extractPipelineIDAndPageNumber 从页面决策路径提取Pipeline ID和页码
// 例：/api/v1/pipelines/abc123/pages/3/decision → "abc123", 3
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

// extractPipelineIDAndPageNumberForAIFix 从AI快修路径提取Pipeline ID和页码
// 例：/api/v1/pipelines/abc123/pages/3/ai-fix → "abc123", 3
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

// ==================== 统一错误处理 ====================

// handlePipelineError 将服务层错误映射到HTTP状态码
func handlePipelineError(w http.ResponseWriter, err error) {
	errMsg := err.Error()

	switch {
	case err == services.ErrPipelineCourseRequired,
		err == services.ErrInvalidDecision:
		utils.BadRequest(w, errMsg)

	case err == services.ErrPipelineNotFound,
		err == services.ErrPipelineCourseNotFound,
		err == services.ErrPageNotFound:
		utils.Fail(w, http.StatusNotFound, errMsg)

	case strings.Contains(errMsg, "已有运行中的Pipeline"),
		err == services.ErrPipelineNotPending,
		err == services.ErrPipelineNotCancellable,
		err == services.ErrPipelineNotDeletable,
		err == services.ErrPipelineNotReviewable,
		err == services.ErrFinalizeIncomplete,
		err == services.ErrMarkPassedNotAllowed,
		err == services.ErrVerifyNotFinalized,
		err == services.ErrEngineQueueFull,
		// 二级审批状态冲突
		err == services.ErrSubmitFinalizeNotAllowed,
		err == services.ErrConfirmFinalizeNotAllowed,
		err == services.ErrRejectFinalizeNotAllowed,
		// 断点续跑状态冲突
		err == services.ErrRestartPipelineBusy:
		utils.Fail(w, http.StatusConflict, errMsg)

	// 断点续跑参数错误
	case err == services.ErrRestartInvalidStep,
		err == services.ErrRestartStepNotAllowed:
		utils.BadRequest(w, errMsg)

	// 已完成Pipeline重跑权限不足 → 403
	case err == services.ErrRestartPermissionDenied:
		utils.Forbidden(w, errMsg)

	case err == services.ErrAIFixFailed:
		utils.Fail(w, http.StatusBadGateway, errMsg)

	case err == services.ErrVerifyIndexGenFailed,
		err == services.ErrVerifyEvalFailed:
		utils.Fail(w, http.StatusBadGateway, errMsg)

	case err == services.ErrMarkPassedNotMet:
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	case err == services.ErrVerifyNoPages,
		err == services.ErrVerifyNoValidHTML,
		err == services.ErrVerifyIndexTooShort,
		err == services.ErrVerifyScoreExtractFail,
		err == services.ErrVerifyScannerNotDone:
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	case err == services.ErrVerifyPromptGMissing,
		err == services.ErrVerifyPromptBMissing,
		err == services.ErrVerifyDictMissing:
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	case err == services.ErrDbCheckIndexMissing,
		err == services.ErrDbCheckIndexTooShort,
		err == services.ErrDbCheckIndexHashMismatch:
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	case strings.Contains(errMsg, "尚有未决策"):
		utils.Fail(w, http.StatusConflict, errMsg)

	case strings.Contains(errMsg, "edit决策必须提供"):
		utils.BadRequest(w, errMsg)

	case strings.Contains(errMsg, "没有生成页面"):
		utils.BadRequest(w, errMsg)

	case strings.Contains(errMsg, "AI快修失败"):
		utils.Fail(w, http.StatusBadGateway, errMsg)

	case strings.Contains(errMsg, "评估未达标"):
		utils.Fail(w, http.StatusUnprocessableEntity, errMsg)

	case strings.Contains(errMsg, "索引生成器AI调用失败"),
		strings.Contains(errMsg, "验收评估AI调用失败"):
		utils.Fail(w, http.StatusBadGateway, errMsg)

	case strings.Contains(errMsg, "不是finalized状态"):
		utils.Fail(w, http.StatusConflict, errMsg)

	default:
		utils.InternalError(w, errMsg)
	}
}
