package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== PipelineHandler ====================

type PipelineHandler struct {
	pipelineService *services.PipelineService
}

func NewPipelineHandler(pipelineService *services.PipelineService) *PipelineHandler {
	return &PipelineHandler{pipelineService: pipelineService}
}

// ==================== Dashboard 统计接口 ====================

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

// ==================== 审核决策接口 ====================

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

// FinalizePipeline 直接定稿（admin专用，跳过二级审批）
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
	// 审计：直接定稿（admin专用）
	if claims, ok := middleware.GetClaims(r.Context()); ok {
		repository.WriteAuditLog(claims.UserID, repository.ActionDirectFinalize,
			map[string]interface{}{"pipeline_id": id, "operator": claims.Username},
			repository.GetClientIP(r.RemoteAddr))
	}
	utils.Success(w, map[string]interface{}{"message": "Pipeline已定稿归档"})
}

// ==================== P7新增：二级审批接口 ====================

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

// ==================== 快捷通过接口 ====================

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

// ==================== AI快修接口 ====================

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

// ==================== 断点续跑接口 ====================

// RestartFromStep POST /api/v1/pipelines/{id}/restart-from
// 从指定步骤重新开始执行Pipeline（断点续跑）
// 请求体：{"step_name": "scanner"} 指定从哪个步骤开始重跑
// 支持步骤：dbCheck / scanner / evaluator / meta / translator / generator
// v37改进：传入调用者角色，由服务层做细粒度权限校验
//   - failed/cancelled 状态：admin / senior_operator / operator 均可
//   - 其他已完成状态：仅 admin / senior_operator 可操作
func (h *PipelineHandler) RestartFromStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 从路径中提取 Pipeline ID（路径格式：/api/v1/pipelines/{id}/restart-from）
	id := extractPipelineIDWithSuffix(r.URL.Path, "/restart-from")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}

	// 获取调用者角色（用于服务层细粒度权限校验）
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 解析请求体，获取目标步骤名称
	var req struct {
		StepName string `json:"step_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if req.StepName == "" {
		utils.BadRequest(w, "step_name 不能为空，请指定要从哪个步骤开始重跑")
		return
	}

	// v37改进：传入调用者角色，服务层根据Pipeline状态+角色做权限判断
	resp, err := h.pipelineService.RestartFromStep(id, req.StepName, claims.Role)
	if err != nil {
		handlePipelineError(w, err)
		return
	}

	utils.Success(w, resp)
}

// ==================== v37新增：批量断点续跑接口 ====================

// BatchRestartFromStep POST /api/v1/pipelines/batch-restart
// 批量从指定步骤重新执行多个Pipeline
// 请求体：{"pipeline_ids": ["id1","id2",...], "step_name": "generator"}
// 权限：仅 admin / senior_operator（路由层已做权限控制）
// 用途：功能改造后，管理员需要对一批已完成的课程从某个步骤重跑
func (h *PipelineHandler) BatchRestartFromStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}

	// 获取调用者角色
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}

	// 解析请求体
	var req struct {
		PipelineIDs []string `json:"pipeline_ids"`
		StepName    string   `json:"step_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if len(req.PipelineIDs) == 0 {
		utils.BadRequest(w, "pipeline_ids 不能为空")
		return
	}
	if len(req.PipelineIDs) > 50 {
		utils.BadRequest(w, "单次批量重跑上限50个Pipeline")
		return
	}
	if req.StepName == "" {
		utils.BadRequest(w, "step_name 不能为空，请指定要从哪个步骤开始重跑")
		return
	}

	// 调用服务层批量执行
	result, err := h.pipelineService.BatchRestartFromStep(req.PipelineIDs, req.StepName, claims.Role)
	if err != nil {
		utils.InternalError(w, "批量重跑失败: "+err.Error())
		return
	}

	utils.Success(w, result)
}

// ==================== 批量操作接口 ====================

func (h *PipelineHandler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}
	var req struct {
		CourseCodes []string `json:"course_codes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if len(req.CourseCodes) == 0 {
		utils.BadRequest(w, "课程编号列表不能为空")
		return
	}
	if len(req.CourseCodes) > 100 {
		utils.BadRequest(w, "单次批量创建上限100个课程")
		return
	}
	result, err := h.pipelineService.BatchCreatePipelines(req.CourseCodes, claims.UserID)
	if err != nil {
		utils.InternalError(w, "批量创建失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

func (h *PipelineHandler) BatchStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	var req struct {
		PipelineIDs []string `json:"pipeline_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if len(req.PipelineIDs) == 0 {
		utils.BadRequest(w, "Pipeline ID列表不能为空")
		return
	}
	if len(req.PipelineIDs) > 50 {
		utils.BadRequest(w, "单次批量启动上限50个Pipeline")
		return
	}
	result, err := h.pipelineService.BatchStartPipelines(req.PipelineIDs)
	if err != nil {
		utils.InternalError(w, "批量启动失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

// ==================== 审核分配接口 ====================

func (h *PipelineHandler) AssignPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/assign")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}
	var req struct {
		AssignedTo string `json:"assigned_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	result, err := h.pipelineService.AssignPipeline(id, req.AssignedTo)
	if err != nil {
		handlePipelineError(w, err)
		return
	}
	utils.Success(w, result)
}

func (h *PipelineHandler) BatchAssign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	var req struct {
		PipelineIDs []string `json:"pipeline_ids"`
		AssignedTo  string   `json:"assigned_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数格式错误")
		return
	}
	if len(req.PipelineIDs) == 0 {
		utils.BadRequest(w, "Pipeline ID列表不能为空")
		return
	}
	result, err := h.pipelineService.BatchAssignPipelines(req.PipelineIDs, req.AssignedTo)
	if err != nil {
		utils.InternalError(w, "批量分配失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}

func (h *PipelineHandler) GetOperators(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET请求")
		return
	}
	operators, err := h.pipelineService.GetOperatorUsers()
	if err != nil {
		utils.InternalError(w, "获取审核员列表失败: "+err.Error())
		return
	}
	utils.Success(w, map[string]interface{}{
		"operators": operators,
		"total":     len(operators),
	})
}

// ==================== 路径解析辅助函数 ====================

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

func extractPipelineIDWithSuffix(path string, suffix string) string {
	idx := strings.LastIndex(path, suffix)
	if idx <= 0 {
		return ""
	}
	path = path[:idx]
	return extractPipelineID(path)
}

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
		// P7新增：二级审批状态冲突
		err == services.ErrSubmitFinalizeNotAllowed,
		err == services.ErrConfirmFinalizeNotAllowed,
		err == services.ErrRejectFinalizeNotAllowed,
		// 断点续跑：状态冲突
		err == services.ErrRestartPipelineBusy:
		utils.Fail(w, http.StatusConflict, errMsg)

	// 断点续跑：参数错误
	case err == services.ErrRestartInvalidStep,
		err == services.ErrRestartStepNotAllowed:
		utils.BadRequest(w, errMsg)

	// v37新增：已完成Pipeline重跑权限不足 → 403
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
