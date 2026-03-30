package handlers

// pipeline_handler_batch.go — Pipeline批量操作类接口
//
// 职责：
//   - 批量创建Pipeline
//   - 批量启动Pipeline
//   - 批量分配审核员（含单个分配）
//   - 断点续跑（单个/批量）
//   - 获取审核员列表

import (
	"encoding/json"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/utils"
)

// ==================== 批量创建 ====================

// BatchCreate POST /api/v1/pipelines/batch-create
// 批量创建Pipeline，上限100个课程，已存在的自动跳过
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

// ==================== 批量启动 ====================

// BatchStart POST /api/v1/pipelines/batch-start
// 批量启动pending状态的Pipeline，上限50个，非pending自动跳过
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

// ==================== 分配审核员 ====================

// AssignPipeline POST /api/v1/pipelines/{id}/assign
// 将单个Pipeline分配给指定审核员
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

// BatchAssign POST /api/v1/pipelines/batch-assign
// 批量将多个Pipeline分配给同一个审核员
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

// GetOperators GET /api/v1/pipelines/operators
// 获取可分配的审核员列表（admin/senior_operator/operator角色）
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

// ==================== 断点续跑 ====================

// RestartFromStep POST /api/v1/pipelines/{id}/restart-from
// 从指定步骤重新执行Pipeline（断点续跑）
//
// 请求体：{"step_name": "scanner"}
// 支持步骤：dbCheck / scanner / evaluator / meta / translator / generator
//
// 权限规则（v37改进）：
//   - failed/cancelled 状态：admin/senior_operator/operator 均可
//   - 其他已完成状态：仅 admin/senior_operator 可操作
func (h *PipelineHandler) RestartFromStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持POST请求")
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/restart-from")
	if id == "" {
		utils.BadRequest(w, "缺少Pipeline ID")
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, "未登录")
		return
	}
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
	// 传入调用者角色，服务层根据Pipeline状态+角色做细粒度权限判断
	resp, err := h.pipelineService.RestartFromStep(id, req.StepName, claims.Role)
	if err != nil {
		handlePipelineError(w, err)
		return
	}
	utils.Success(w, resp)
}

// BatchRestartFromStep POST /api/v1/pipelines/batch-restart
// 批量从指定步骤重新执行多个Pipeline，上限50个
// 权限：仅 admin/senior_operator（路由层已做权限控制）
func (h *PipelineHandler) BatchRestartFromStep(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.pipelineService.BatchRestartFromStep(req.PipelineIDs, req.StepName, claims.Role)
	if err != nil {
		utils.InternalError(w, "批量重跑失败: "+err.Error())
		return
	}
	utils.Success(w, result)
}
