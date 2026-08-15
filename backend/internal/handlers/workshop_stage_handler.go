package handlers

// workshop_stage_handler.go — 阶段化备课工坊核心HTTP处理器
//
// 本文件只保留阶段进度、推进、跳过、回退、切换和重启等核心入口。
// 自定义阶段CRUD与管理端系统阶段接口拆分到
// workshop_stage_handler_management.go，避免单文件继续超过900行。
//
// 知识脉络安全规则：
//   - 所有HTTP推进统一经过Prepared入口；
//   - 组件整组校验完成后，才允许生成知识脉络；
//   - 精确挂载课程大纲时，离开analyze前必须存在active知识脉络；
//   - Skip、Switch和Reset不能绕过教学分析确认。

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

// WorkshopStageHandler 阶段化备课工坊处理器。
type WorkshopStageHandler struct {
	stageService *services.WorkshopStageService
}

// NewWorkshopStageHandler 创建阶段处理器。
func NewWorkshopStageHandler(
	service *services.WorkshopStageService,
) *WorkshopStageHandler {
	return &WorkshopStageHandler{
		stageService: service,
	}
}

// GetDefaultStages 获取系统默认阶段。
func (h *WorkshopStageHandler) GetDefaultStages(
	w http.ResponseWriter,
	r *http.Request,
) {
	response, err := h.stageService.GetDefaultStages(
		r.Context(),
	)
	if err != nil {
		utils.InternalError(
			w,
			"获取默认阶段失败",
		)
		return
	}

	utils.Success(w, response)
}

// GetStageStatus 获取教案阶段进度。
func (h *WorkshopStageHandler) GetStageStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	planID := extractPlanIDBeforeStages(
		r.URL.Path,
	)
	if planID == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidPlanID,
		)
		return
	}

	response, err := h.stageService.GetStageStatus(
		r.Context(),
		planID,
		claims.UserID,
	)
	if err != nil {
		handleStageError(w, err)
		return
	}

	utils.Success(w, response)
}

// GetStageOutput 获取阶段产出物。
func (h *WorkshopStageHandler) GetStageOutput(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	planID, stageCode := extractPlanIDAndStageCode(
		r.URL.Path,
	)
	if planID == "" || stageCode == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidPlanOrStage,
		)
		return
	}

	response, err := h.stageService.GetStageOutput(
		r.Context(),
		planID,
		stageCode,
		claims.UserID,
	)
	if err != nil {
		handleStageError(w, err)
		return
	}

	utils.Success(w, response)
}

// ResetStage 重启指定阶段。
//
// 必须调用ResetStagePrepared，防止当前仍在analyze且没有active知识脉络时，
// 通过重启后续阶段绕过正式确认入口。
func (h *WorkshopStageHandler) ResetStage(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	planID := extractPlanIDBeforeStages(
		r.URL.Path,
	)
	if planID == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidPlanID,
		)
		return
	}

	var request models.ResetStageRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil ||
		strings.TrimSpace(
			request.TargetStageCode,
		) == "" {
		utils.BadRequest(
			w,
			"请指定要重启的阶段代码",
		)
		return
	}

	stage, err := h.stageService.ResetStagePrepared(
		r.Context(),
		planID,
		request.TargetStageCode,
		claims.UserID,
	)
	if err != nil {
		handleStageError(w, err)
		return
	}

	utils.Success(w, stage)
}

// GetStageCompleteness 获取阶段完成度。
func (h *WorkshopStageHandler) GetStageCompleteness(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	planID, stageCode := extractPlanIDAndStageCode(
		r.URL.Path,
	)
	if planID == "" || stageCode == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidPlanOrStage,
		)
		return
	}

	lessonPlan, err := services.GetLessonPlanForCheck(
		r.Context(),
		planID,
	)
	if err != nil {
		utils.Fail(
			w,
			http.StatusNotFound,
			"教案不存在",
		)
		return
	}

	if lessonPlan.AuthorID != claims.UserID {
		utils.Fail(
			w,
			http.StatusForbidden,
			"无权操作此教案",
		)
		return
	}

	response, err := services.CheckStageCompleteness(
		r.Context(),
		planID,
		stageCode,
	)
	if err != nil {
		handleStageError(w, err)
		return
	}

	utils.Success(w, response)
}

// GetStageRecommendedComponents 获取阶段推荐组件。
func (h *WorkshopStageHandler) GetStageRecommendedComponents(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	planID, stageCode := extractPlanIDAndStageCode(
		r.URL.Path,
	)
	if planID == "" || stageCode == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidPlanOrStage,
		)
		return
	}

	response, err := h.stageService.GetRecommendedComponents(
		r.Context(),
		planID,
		stageCode,
		claims.UserID,
	)
	if err != nil {
		handleStageError(w, err)
		return
	}

	utils.Success(w, response)
}

// AdvanceStage 进入下一阶段。
//
// 无组件和有组件均经过同一Prepared入口：
//   - 先解析目标阶段；
//   - 再整组校验组件；
//   - 离开analyze时生成或复用active知识脉络；
//   - 最后才执行原阶段推进副作用。
func (h *WorkshopStageHandler) AdvanceStage(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	planID := extractPlanIDBeforeStages(
		r.URL.Path,
	)
	if planID == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidPlanID,
		)
		return
	}

	var request models.AdvanceStageWithComponentsRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		request = models.AdvanceStageWithComponentsRequest{}
	}

	var (
		stage interface{}
		err   error
	)

	if r.URL.Query().Get(
		"silent_eval",
	) == "1" {
		stage, err = h.stageService.AdvanceStageSilentPrepared(
			r.Context(),
			planID,
			request.TargetStageCode,
			claims.UserID,
			request.SelectedComponentIDs,
		)
	} else {
		stage, err = h.stageService.AdvanceStagePrepared(
			r.Context(),
			planID,
			request.TargetStageCode,
			claims.UserID,
			request.SelectedComponentIDs,
		)
	}

	if err != nil {
		handleStageError(w, err)
		return
	}

	utils.Success(w, stage)
}

// SkipStage 跳过当前阶段。
func (h *WorkshopStageHandler) SkipStage(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	planID := extractPlanIDBeforeStages(
		r.URL.Path,
	)
	if planID == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidPlanID,
		)
		return
	}

	var request models.SkipStageRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		request = models.SkipStageRequest{}
	}

	stage, err := h.stageService.SkipStagePrepared(
		r.Context(),
		planID,
		request.TargetStageCode,
		claims.UserID,
	)
	if err != nil {
		handleStageError(w, err)
		return
	}

	utils.Success(w, stage)
}

// BackStage 回退到上一阶段。
//
// 回到analyze本身允许；教师随后修改分析对话或阶段产出时，
// 数据库失效触发器会立即把旧知识脉络标记为stale。
func (h *WorkshopStageHandler) BackStage(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	planID := extractPlanIDBeforeStages(
		r.URL.Path,
	)
	if planID == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidPlanID,
		)
		return
	}

	stage, err := h.stageService.BackStage(
		r.Context(),
		planID,
		claims.UserID,
	)
	if err != nil {
		handleStageError(w, err)
		return
	}

	utils.Success(w, stage)
}

// SwitchToStage 切换到指定阶段。
//
// 必须调用SwitchToStagePrepared，防止从未确认的analyze直接切入后续阶段。
func (h *WorkshopStageHandler) SwitchToStage(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	planID := extractPlanIDBeforeStages(
		r.URL.Path,
	)
	if planID == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidPlanID,
		)
		return
	}

	var request models.ResetStageRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil ||
		strings.TrimSpace(
			request.TargetStageCode,
		) == "" {
		utils.BadRequest(
			w,
			"请指定要切换的阶段代码",
		)
		return
	}

	stage, err := h.stageService.SwitchToStagePrepared(
		r.Context(),
		planID,
		request.TargetStageCode,
		claims.UserID,
	)
	if err != nil {
		handleStageError(w, err)
		return
	}

	utils.Success(w, stage)
}

// extractPlanIDBeforeStages 从阶段操作路径提取教案ID。
func extractPlanIDBeforeStages(
	path string,
) string {
	parts := strings.Split(
		strings.TrimSuffix(path, "/"),
		"/",
	)

	for index, part := range parts {
		if part == "plans" &&
			index+1 < len(parts) {
			id := parts[index+1]
			if len(id) >= 10 {
				return id
			}
		}
	}

	return ""
}

// extractPlanIDAndStageCode 从阶段详情路径提取教案ID和阶段代码。
func extractPlanIDAndStageCode(
	path string,
) (string, string) {
	parts := strings.Split(
		strings.TrimSuffix(path, "/"),
		"/",
	)

	planID := ""
	stageCode := ""

	for index, part := range parts {
		if part == "plans" &&
			index+1 < len(parts) {
			planID = parts[index+1]
		}

		if part == "stages" &&
			index+1 < len(parts) {
			candidate := parts[index+1]

			switch candidate {
			case "advance",
				"skip",
				"back",
				"defaults",
				"reset",
				"switch":
				// 子操作不是阶段代码。
			default:
				stageCode = candidate
			}
		}
	}

	return planID, stageCode
}

// handleStageError 统一映射阶段操作错误。
func handleStageError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrLPGenTaskRunning,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrLPGenServiceDraining,
	):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrStageNotInitialized,
	):
		utils.Fail(
			w,
			http.StatusBadRequest,
			"教案尚未初始化阶段配置",
		)

	case errors.Is(
		err,
		services.ErrStageAlreadyFirst,
	):
		utils.Fail(
			w,
			http.StatusBadRequest,
			"已经是第一个阶段，无法回退",
		)

	case errors.Is(
		err,
		services.ErrStageAlreadyLast,
	):
		utils.Fail(
			w,
			http.StatusBadRequest,
			"已经是最后一个阶段",
		)

	case errors.Is(
		err,
		services.ErrStageNotSkippable,
	):
		utils.Fail(
			w,
			http.StatusBadRequest,
			"当前阶段不可跳过",
		)

	case errors.Is(
		err,
		services.ErrStageInvalidTarget,
	):
		utils.Fail(
			w,
			http.StatusBadRequest,
			"目标阶段不存在",
		)

	case errors.Is(
		err,
		services.ErrComponentSelectionInvalid,
	),
		errors.Is(
			err,
			services.ErrComponentEducationDomainInvalid,
		):
		utils.Fail(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrLessonPlanKnowledgeLineageAnalyzeRequired,
	),
		errors.Is(
			err,
			services.ErrLessonPlanKnowledgeAnchorsIncomplete,
		):
		utils.Fail(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	case errors.Is(
		err,
		repository.ErrLessonPlanKnowledgeLineageSourceChanged,
	),
		errors.Is(
			err,
			repository.ErrLessonPlanKnowledgeLineageSourceUnavailable,
		),
		errors.Is(
			err,
			repository.ErrLessonPlanKnowledgeLineageConfirmedStageUnavailable,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrLessonPlanKnowledgeAnchorExtractionUnavailable,
	),
		errors.Is(
			err,
			services.ErrLessonPlanKnowledgeLineageExtractionFailed,
		):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrLPGenPlanNotFound,
	),
		errors.Is(
			err,
			repository.ErrLessonPlanNotFound,
		):
		utils.Fail(
			w,
			http.StatusNotFound,
			"教案不存在",
		)

	case errors.Is(
		err,
		services.ErrLPGenUnauthorized,
	):
		utils.Fail(
			w,
			http.StatusForbidden,
			"无权操作此教案",
		)

	case strings.Contains(
		err.Error(),
		"必须先完成教学分析",
	):
		// 数据库硬闸在极窄竞态下可能返回普通pg错误包装。
		// 保留明确业务提示，不把可恢复问题伪装成服务器故障。
		utils.Fail(
			w,
			http.StatusConflict,
			"已关联课程大纲的教案必须先完成教学分析并生成知识脉络",
		)

	default:
		utils.InternalError(
			w,
			"操作失败: "+err.Error(),
		)
	}
}
