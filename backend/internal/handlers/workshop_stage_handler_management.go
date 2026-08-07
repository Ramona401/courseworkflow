package handlers

// workshop_stage_handler_management.go
// 阶段化备课工坊的自定义阶段CRUD与管理端系统阶段接口。
//
// 从workshop_stage_handler.go拆分的原因：
//   - 原处理器已经超过1200行；
//   - 本文件与普通教师阶段推进没有共享事务或知识脉络状态；
//   - 拆分后两个文件职责清晰，且都保持在900行以内。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ListCustomStages 列出当前Actor有权查看的配方自定义阶段。
func (h *WorkshopStageHandler) ListCustomStages(
	w http.ResponseWriter,
	r *http.Request,
) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	recipeID := extractRecipeIDFromCustomStagePath(
		r.URL.Path,
	)
	if recipeID == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidRecipeID,
		)
		return
	}

	stages, err := h.stageService.ListCustomStagesForActor(
		r.Context(),
		actor,
		recipeID,
	)
	if err != nil {
		handleCustomStageError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"stages": stages,
		},
	)
}

// CreateCustomStage 为作者本人配方创建自定义阶段。
func (h *WorkshopStageHandler) CreateCustomStage(
	w http.ResponseWriter,
	r *http.Request,
) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	recipeID := extractRecipeIDFromCustomStagePath(
		r.URL.Path,
	)
	if recipeID == "" {
		utils.BadRequest(
			w,
			utils.MsgInvalidRecipeID,
		)
		return
	}

	var request models.CreateCustomStageRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestArgs,
		)
		return
	}

	response, err := h.stageService.CreateCustomStageForActor(
		r.Context(),
		actor,
		recipeID,
		&request,
	)
	if err != nil {
		handleCustomStageError(w, err)
		return
	}

	utils.Success(w, response)
}

// UpdateCustomStage 更新作者本人配方的自定义阶段。
func (h *WorkshopStageHandler) UpdateCustomStage(
	w http.ResponseWriter,
	r *http.Request,
) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	recipeID, stageCode :=
		extractRecipeIDAndStageCodeFromCustomStagePath(
			r.URL.Path,
		)
	if recipeID == "" || stageCode == "" {
		utils.BadRequest(
			w,
			"配方ID或阶段代码无效",
		)
		return
	}

	var request models.UpdateCustomStageRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestArgs,
		)
		return
	}

	if err := h.stageService.UpdateCustomStageForActor(
		r.Context(),
		actor,
		recipeID,
		stageCode,
		&request,
	); err != nil {
		handleCustomStageError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "更新成功",
		},
	)
}

// DeleteCustomStage 删除作者本人配方的自定义阶段。
func (h *WorkshopStageHandler) DeleteCustomStage(
	w http.ResponseWriter,
	r *http.Request,
) {
	actor, ok := buildRecipeActor(w, r)
	if !ok {
		return
	}

	recipeID, stageCode :=
		extractRecipeIDAndStageCodeFromCustomStagePath(
			r.URL.Path,
		)
	if recipeID == "" || stageCode == "" {
		utils.BadRequest(
			w,
			"配方ID或阶段代码无效",
		)
		return
	}

	if err := h.stageService.DeleteCustomStageForActor(
		r.Context(),
		actor,
		recipeID,
		stageCode,
	); err != nil {
		handleCustomStageError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "删除成功",
		},
	)
}

// extractRecipeIDFromCustomStagePath 从自定义阶段集合路径提取配方ID。
func extractRecipeIDFromCustomStagePath(
	path string,
) string {
	parts := strings.Split(
		strings.TrimSuffix(path, "/"),
		"/",
	)

	for index, part := range parts {
		if part == "recipes" &&
			index+1 < len(parts) {
			id := parts[index+1]
			if len(id) >= 10 {
				return id
			}
		}
	}

	return ""
}

// extractRecipeIDAndStageCodeFromCustomStagePath
// 从自定义阶段详情路径提取配方ID和阶段代码。
func extractRecipeIDAndStageCodeFromCustomStagePath(
	path string,
) (string, string) {
	parts := strings.Split(
		strings.TrimSuffix(path, "/"),
		"/",
	)

	recipeID := ""
	stageCode := ""

	for index, part := range parts {
		if part == "recipes" &&
			index+1 < len(parts) {
			id := parts[index+1]
			if len(id) >= 10 {
				recipeID = id
			}
		}

		if part == "custom-stages" &&
			index+1 < len(parts) {
			stageCode = parts[index+1]
		}
	}

	return recipeID, stageCode
}

// handleCustomStageError 统一映射自定义阶段错误。
func handleCustomStageError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrRecipeNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"配方不存在",
		)

	case errors.Is(
		err,
		services.ErrRecipeUnauthorized,
	):
		utils.Fail(
			w,
			http.StatusForbidden,
			"无权操作此配方",
		)

	case errors.Is(
		err,
		services.ErrCustomStageLimit,
	):
		utils.Fail(
			w,
			http.StatusBadRequest,
			"自定义阶段数量已达上限（最多10个）",
		)

	case errors.Is(
		err,
		repository.ErrStageNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"自定义阶段不存在",
		)

	case errors.Is(
		err,
		repository.ErrStageCodeConflict,
	):
		utils.Fail(
			w,
			http.StatusBadRequest,
			"阶段代码已存在",
		)

	default:
		errorMessage := err.Error()

		if strings.Contains(
			errorMessage,
			"不能为空",
		) ||
			strings.Contains(
				errorMessage,
				"仅允许",
			) ||
			strings.Contains(
				errorMessage,
				"冲突",
			) {
			utils.BadRequest(
				w,
				errorMessage,
			)
			return
		}

		utils.InternalError(
			w,
			"操作失败: "+errorMessage,
		)
	}
}

// ListAllSystemStages 管理端列出全部系统阶段。
func (h *WorkshopStageHandler) ListAllSystemStages(
	w http.ResponseWriter,
	r *http.Request,
) {
	stages, err := repository.GetAllSystemStages(
		r.Context(),
	)
	if err != nil {
		utils.InternalError(
			w,
			"获取系统阶段失败",
		)
		return
	}

	utils.Success(
		w,
		&models.AdminStageListResponse{
			Stages: stages,
		},
	)
}

// UpdateSystemStage 管理端更新系统阶段。
func (h *WorkshopStageHandler) UpdateSystemStage(
	w http.ResponseWriter,
	r *http.Request,
) {
	parts := strings.Split(
		strings.TrimSuffix(
			r.URL.Path,
			"/",
		),
		"/",
	)

	stageCode := ""
	for index, part := range parts {
		if part == "workshop-stages" &&
			index+1 < len(parts) {
			stageCode = parts[index+1]
			break
		}
	}

	if stageCode == "" {
		utils.BadRequest(
			w,
			"阶段代码无效",
		)
		return
	}

	var request models.UpdateStageRequest
	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		utils.BadRequest(
			w,
			utils.MsgBadRequestBody,
		)
		return
	}

	if strings.TrimSpace(
		request.StageName,
	) == "" {
		utils.BadRequest(
			w,
			"阶段名称不能为空",
		)
		return
	}

	if strings.TrimSpace(
		request.AIRole,
	) == "" {
		utils.BadRequest(
			w,
			"AI角色不能为空",
		)
		return
	}

	if request.GateMode != "suggest" &&
		request.GateMode != "force" &&
		request.GateMode != "auto" {
		utils.BadRequest(
			w,
			"门控模式无效，可选值：suggest/force/auto",
		)
		return
	}

	if request.Status != "active" &&
		request.Status != "disabled" {
		utils.BadRequest(
			w,
			"状态无效，可选值：active/disabled",
		)
		return
	}

	if err := repository.UpdateSystemStage(
		r.Context(),
		stageCode,
		&request,
	); err != nil {
		if errors.Is(
			err,
			repository.ErrStageNotFound,
		) {
			utils.Fail(
				w,
				http.StatusNotFound,
				"阶段不存在",
			)
			return
		}

		utils.InternalError(
			w,
			"更新阶段失败: "+err.Error(),
		)
		return
	}

	updated, err := repository.GetStageByCode(
		r.Context(),
		models.StageSourceSystem,
		stageCode,
	)
	if err != nil {
		utils.Success(
			w,
			map[string]string{
				"message":    "更新成功",
				"stage_code": stageCode,
			},
		)
		return
	}

	utils.Success(w, updated)
}

