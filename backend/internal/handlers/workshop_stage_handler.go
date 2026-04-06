package handlers

// workshop_stage_handler.go — 阶段化备课工坊HTTP处理器
//
// Phase 7B 新增：6个REST接口（阶段操作）
// 迭代5 新增：4个REST接口（自定义阶段CRUD）
//   GET    /api/v1/lesson-plans/recipes/{id}/custom-stages           — 获取配方自定义阶段列表
//   POST   /api/v1/lesson-plans/recipes/{id}/custom-stages           — 创建自定义阶段
//   PUT    /api/v1/lesson-plans/recipes/{id}/custom-stages/{code}    — 更新自定义阶段
//   DELETE /api/v1/lesson-plans/recipes/{id}/custom-stages/{code}    — 删除自定义阶段

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// WorkshopStageHandler 阶段化备课工坊处理器
type WorkshopStageHandler struct {
	stageService *services.WorkshopStageService
}

// NewWorkshopStageHandler 创建阶段处理器
func NewWorkshopStageHandler(ss *services.WorkshopStageService) *WorkshopStageHandler {
	return &WorkshopStageHandler{stageService: ss}
}

// ==================== 获取系统默认阶段 ====================

// GetDefaultStages GET /api/v1/lesson-plans/workshop/stages/defaults
func (h *WorkshopStageHandler) GetDefaultStages(w http.ResponseWriter, r *http.Request) {
	resp, err := h.stageService.GetDefaultStages(r.Context())
	if err != nil {
		utils.InternalError(w, "获取默认阶段失败")
		return
	}
	utils.Success(w, resp)
}

// ==================== 获取教案阶段进度 ====================

// GetStageStatus GET /api/v1/lesson-plans/plans/{id}/stages
func (h *WorkshopStageHandler) GetStageStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	planID := extractPlanIDBeforeStages(r.URL.Path)
	if planID == "" {
		utils.BadRequest(w, "教案ID无效")
		return
	}

	resp, err := h.stageService.GetStageStatus(r.Context(), planID, claims.UserID)
	if err != nil {
		handleStageError(w, err)
		return
	}
	utils.Success(w, resp)
}

// ==================== 获取阶段产出物 ====================

// GetStageOutput GET /api/v1/lesson-plans/plans/{id}/stages/{code}/output
func (h *WorkshopStageHandler) GetStageOutput(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	planID, stageCode := extractPlanIDAndStageCode(r.URL.Path)
	if planID == "" || stageCode == "" {
		utils.BadRequest(w, "教案ID或阶段代码无效")
		return
	}

	resp, err := h.stageService.GetStageOutput(r.Context(), planID, stageCode, claims.UserID)
	if err != nil {
		handleStageError(w, err)
		return
	}
	utils.Success(w, resp)
}

// ==================== 进入下一阶段 ====================

// AdvanceStage POST /api/v1/lesson-plans/plans/{id}/stages/advance
func (h *WorkshopStageHandler) AdvanceStage(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	planID := extractPlanIDBeforeStages(r.URL.Path)
	if planID == "" {
		utils.BadRequest(w, "教案ID无效")
		return
	}

	var req models.AdvanceStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// body可以为空，默认进入下一个
		req = models.AdvanceStageRequest{}
	}

	stage, err := h.stageService.AdvanceStage(r.Context(), planID, req.TargetStageCode, claims.UserID)
	if err != nil {
		handleStageError(w, err)
		return
	}
	utils.Success(w, stage)
}

// ==================== 跳过当前阶段 ====================

// SkipStage POST /api/v1/lesson-plans/plans/{id}/stages/skip
func (h *WorkshopStageHandler) SkipStage(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	planID := extractPlanIDBeforeStages(r.URL.Path)
	if planID == "" {
		utils.BadRequest(w, "教案ID无效")
		return
	}

	var req models.SkipStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = models.SkipStageRequest{}
	}

	stage, err := h.stageService.SkipStage(r.Context(), planID, req.TargetStageCode, claims.UserID)
	if err != nil {
		handleStageError(w, err)
		return
	}
	utils.Success(w, stage)
}

// ==================== 回退到上一阶段 ====================

// BackStage POST /api/v1/lesson-plans/plans/{id}/stages/back
func (h *WorkshopStageHandler) BackStage(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	planID := extractPlanIDBeforeStages(r.URL.Path)
	if planID == "" {
		utils.BadRequest(w, "教案ID无效")
		return
	}

	stage, err := h.stageService.BackStage(r.Context(), planID, claims.UserID)
	if err != nil {
		handleStageError(w, err)
		return
	}
	utils.Success(w, stage)
}

// ==================== 迭代5新增：自定义阶段 CRUD ====================

// ListCustomStages GET /api/v1/lesson-plans/recipes/{id}/custom-stages
// 获取配方的自定义阶段列表
func (h *WorkshopStageHandler) ListCustomStages(w http.ResponseWriter, r *http.Request) {
	recipeID := extractRecipeIDFromCustomStagePath(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, "配方ID无效")
		return
	}

	stages, err := h.stageService.ListCustomStages(r.Context(), recipeID)
	if err != nil {
		utils.InternalError(w, "获取自定义阶段失败")
		return
	}
	utils.Success(w, map[string]interface{}{"stages": stages})
}

// CreateCustomStage POST /api/v1/lesson-plans/recipes/{id}/custom-stages
// 创建配方自定义阶段
func (h *WorkshopStageHandler) CreateCustomStage(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	recipeID := extractRecipeIDFromCustomStagePath(r.URL.Path)
	if recipeID == "" {
		utils.BadRequest(w, "配方ID无效")
		return
	}

	var req models.CreateCustomStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数无效")
		return
	}

	resp, err := h.stageService.CreateCustomStage(r.Context(), recipeID, &req, claims.UserID)
	if err != nil {
		handleCustomStageError(w, err)
		return
	}
	utils.Success(w, resp)
}

// UpdateCustomStage PUT /api/v1/lesson-plans/recipes/{id}/custom-stages/{code}
// 更新配方自定义阶段
func (h *WorkshopStageHandler) UpdateCustomStage(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	recipeID, stageCode := extractRecipeIDAndStageCodeFromCustomStagePath(r.URL.Path)
	if recipeID == "" || stageCode == "" {
		utils.BadRequest(w, "配方ID或阶段代码无效")
		return
	}

	var req models.UpdateCustomStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求参数无效")
		return
	}

	if err := h.stageService.UpdateCustomStage(r.Context(), recipeID, stageCode, &req, claims.UserID); err != nil {
		handleCustomStageError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "更新成功"})
}

// DeleteCustomStage DELETE /api/v1/lesson-plans/recipes/{id}/custom-stages/{code}
// 删除配方自定义阶段
func (h *WorkshopStageHandler) DeleteCustomStage(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return
	}

	recipeID, stageCode := extractRecipeIDAndStageCodeFromCustomStagePath(r.URL.Path)
	if recipeID == "" || stageCode == "" {
		utils.BadRequest(w, "配方ID或阶段代码无效")
		return
	}

	if err := h.stageService.DeleteCustomStage(r.Context(), recipeID, stageCode, claims.UserID); err != nil {
		handleCustomStageError(w, err)
		return
	}
	utils.Success(w, map[string]string{"message": "删除成功"})
}

// ==================== 辅助函数 ====================

// extractPlanIDBeforeStages 从路径 .../plans/{id}/stages/... 中提取教案ID
// 查找 "plans" 后面的那个段
func extractPlanIDBeforeStages(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, p := range parts {
		if p == "plans" && i+1 < len(parts) {
			id := parts[i+1]
			if len(id) >= 10 {
				return id
			}
		}
	}
	return ""
}

// extractPlanIDAndStageCode 从路径 .../plans/{id}/stages/{code}/output 中提取教案ID和阶段代码
func extractPlanIDAndStageCode(path string) (string, string) {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	planID := ""
	stageCode := ""
	for i, p := range parts {
		if p == "plans" && i+1 < len(parts) {
			planID = parts[i+1]
		}
		if p == "stages" && i+1 < len(parts) {
			candidate := parts[i+1]
			// 排除操作路径（advance/skip/back）
			if candidate != "advance" && candidate != "skip" && candidate != "back" && candidate != "defaults" {
				stageCode = candidate
			}
		}
	}
	return planID, stageCode
}

// extractRecipeIDFromCustomStagePath 从路径 .../recipes/{id}/custom-stages 中提取配方ID
// 迭代5新增
func extractRecipeIDFromCustomStagePath(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, p := range parts {
		if p == "recipes" && i+1 < len(parts) {
			id := parts[i+1]
			if len(id) >= 10 {
				return id
			}
		}
	}
	return ""
}

// extractRecipeIDAndStageCodeFromCustomStagePath 从路径 .../recipes/{id}/custom-stages/{code} 中提取配方ID和阶段代码
// 迭代5新增
func extractRecipeIDAndStageCodeFromCustomStagePath(path string) (string, string) {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	recipeID := ""
	stageCode := ""
	for i, p := range parts {
		if p == "recipes" && i+1 < len(parts) {
			id := parts[i+1]
			if len(id) >= 10 {
				recipeID = id
			}
		}
		if p == "custom-stages" && i+1 < len(parts) {
			stageCode = parts[i+1]
		}
	}
	return recipeID, stageCode
}

// handleStageError 统一阶段错误处理
func handleStageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrStageNotInitialized):
		utils.Fail(w, 400, "教案尚未初始化阶段配置")
	case errors.Is(err, services.ErrStageAlreadyFirst):
		utils.Fail(w, 400, "已经是第一个阶段，无法回退")
	case errors.Is(err, services.ErrStageAlreadyLast):
		utils.Fail(w, 400, "已经是最后一个阶段")
	case errors.Is(err, services.ErrStageNotSkippable):
		utils.Fail(w, 400, "当前阶段不可跳过")
	case errors.Is(err, services.ErrStageInvalidTarget):
		utils.Fail(w, 400, "目标阶段不存在")
	case errors.Is(err, services.ErrLPGenPlanNotFound):
		utils.Fail(w, 404, "教案不存在")
	case errors.Is(err, services.ErrLPGenUnauthorized):
		utils.Fail(w, 403, "无权操作此教案")
	default:
		utils.InternalError(w, "操作失败: "+err.Error())
	}
}

// handleCustomStageError 迭代5新增：自定义阶段错误处理
func handleCustomStageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrRecipeNotFound):
		utils.Fail(w, 404, "配方不存在")
	case errors.Is(err, services.ErrRecipeUnauthorized):
		utils.Fail(w, 403, "无权操作此配方")
	case errors.Is(err, services.ErrCustomStageLimit):
		utils.Fail(w, 400, "自定义阶段数量已达上限（最多10个）")
	case errors.Is(err, repository.ErrStageNotFound):
		utils.Fail(w, 404, "自定义阶段不存在")
	case errors.Is(err, repository.ErrStageCodeConflict):
		utils.Fail(w, 400, "阶段代码已存在")
	default:
		errMsg := err.Error()
		if strings.Contains(errMsg, "不能为空") || strings.Contains(errMsg, "仅允许") || strings.Contains(errMsg, "冲突") {
			utils.BadRequest(w, errMsg)
		} else {
			utils.InternalError(w, "操作失败: "+errMsg)
		}
	}
}

// ==================== Admin管理：获取全部系统阶段（含disabled）====================

// ListAllSystemStages GET /api/v1/admin/workshop-stages
// Admin专用：返回全部系统阶段（含disabled），包含完整字段
func (h *WorkshopStageHandler) ListAllSystemStages(w http.ResponseWriter, r *http.Request) {
	stages, err := repository.GetAllSystemStages(r.Context())
	if err != nil {
		utils.InternalError(w, "获取系统阶段失败")
		return
	}
	utils.Success(w, &models.AdminStageListResponse{Stages: stages})
}

// ==================== Admin管理：更新系统阶段 ====================

// UpdateSystemStage PUT /api/v1/admin/workshop-stages/{code}
// Admin专用：更新系统阶段的可编辑字段
func (h *WorkshopStageHandler) UpdateSystemStage(w http.ResponseWriter, r *http.Request) {
	// 从路径中提取 stage_code
	path := r.URL.Path
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	stageCode := ""
	for i, p := range parts {
		if p == "workshop-stages" && i+1 < len(parts) {
			stageCode = parts[i+1]
			break
		}
	}
	if stageCode == "" {
		utils.BadRequest(w, "阶段代码无效")
		return
	}

	var req models.UpdateStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "请求体解析失败")
		return
	}

	// 基本校验
	if strings.TrimSpace(req.StageName) == "" {
		utils.BadRequest(w, "阶段名称不能为空")
		return
	}
	if strings.TrimSpace(req.AIRole) == "" {
		utils.BadRequest(w, "AI角色不能为空")
		return
	}
	if req.GateMode != "suggest" && req.GateMode != "force" && req.GateMode != "auto" {
		utils.BadRequest(w, "门控模式无效，可选值：suggest/force/auto")
		return
	}
	if req.Status != "active" && req.Status != "disabled" {
		utils.BadRequest(w, "状态无效，可选值：active/disabled")
		return
	}

	if err := repository.UpdateSystemStage(r.Context(), stageCode, &req); err != nil {
		if errors.Is(err, repository.ErrStageNotFound) {
			utils.Fail(w, 404, "阶段不存在")
			return
		}
		utils.InternalError(w, "更新阶段失败: "+err.Error())
		return
	}

	// 返回更新后的阶段数据
	updated, err := repository.GetStageByCode(r.Context(), models.StageSourceSystem, stageCode)
	if err != nil {
		// 如果刚设为disabled，GetStageByCode(active)会找不到，直接返回成功
		utils.Success(w, map[string]string{"message": "更新成功", "stage_code": stageCode})
		return
	}
	utils.Success(w, updated)
}
