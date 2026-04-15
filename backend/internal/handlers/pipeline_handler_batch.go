package handlers

// pipeline_handler_batch.go — Pipeline批量操作类接口

import (
	"encoding/json"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/utils"
)

// ==================== 批量创建 ====================

func (h *PipelineHandler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req struct {
		CourseCodes []string `json:"course_codes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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

func (h *PipelineHandler) BatchStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	var req struct {
		PipelineIDs []string `json:"pipeline_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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

func (h *PipelineHandler) AssignPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/assign")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingPipelineID)
		return
	}
	var req struct {
		AssignedTo string `json:"assigned_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	var req struct {
		PipelineIDs []string `json:"pipeline_ids"`
		AssignedTo  string   `json:"assigned_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
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

func (h *PipelineHandler) RestartFromStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	id := extractPipelineIDWithSuffix(r.URL.Path, "/restart-from")
	if id == "" {
		utils.BadRequest(w, utils.MsgMissingPipelineID)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req struct {
		StepName string `json:"step_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
		return
	}
	if req.StepName == "" {
		utils.BadRequest(w, "step_name 不能为空，请指定要从哪个步骤开始重跑")
		return
	}
	resp, err := h.pipelineService.RestartFromStep(id, req.StepName, claims.Role)
	if err != nil {
		handlePipelineError(w, err)
		return
	}
	utils.Success(w, resp)
}

func (h *PipelineHandler) BatchRestartFromStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		utils.Unauthorized(w, utils.MsgNotLoggedIn)
		return
	}
	var req struct {
		PipelineIDs []string `json:"pipeline_ids"`
		StepName    string   `json:"step_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, utils.MsgBadRequestBody)
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
