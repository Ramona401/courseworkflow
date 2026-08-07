package services

// component_direct_selection_service.go — 用户直接提交组件ID安全包装。
//
// 本文件不改动已有超长核心Service，而是在外层提供严格验证入口：
//
// 阶段推进：
//   - 先读取教案并验证作者；
//   - 先解析目标阶段；
//   - 按目标阶段component_types整组验证ID；
//   - 全部通过后才调用原推进方法；
//   - 验证失败时不完成当前阶段、不创建新产出、不更新current_stage。
//
// 对话选择：
//   - 先读取可编辑教案；
//   - 按当前阶段component_types整组验证ID；
//   - 全部通过后才调用原Chat；
//   - 验证失败时不登记AI任务、不追加用户消息。
//
// 所有组件必须同时满足：存在、active、approved、同域或common、
// library_type属于本阶段。任一组件非法即整组失败。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// parseStageAllowedComponentTypes 解析阶段允许使用的组件类型。
//
// 空数组表示该阶段没有开放直接组件选择；
// 非法JSON、空类型和未知类型全部fail-closed。
func parseStageAllowedComponentTypes(
	raw string,
) ([]string, error) {
	raw = strings.TrimSpace(raw)

	if raw == "" || raw == "[]" {
		return []string{}, nil
	}

	var values []string

	if err := json.Unmarshal(
		[]byte(raw),
		&values,
	); err != nil {
		return nil,
			ErrComponentSelectionInvalid
	}

	result := make(
		[]string,
		0,
		len(values),
	)

	seen := make(
		map[string]bool,
		len(values),
	)

	for _, rawValue := range values {
		libraryType := strings.TrimSpace(
			rawValue,
		)

		if libraryType == "" ||
			!models.IsValidLibraryType(
				libraryType,
			) {
			return nil,
				ErrComponentSelectionInvalid
		}

		if seen[libraryType] {
			continue
		}

		seen[libraryType] = true

		result = append(
			result,
			libraryType,
		)
	}

	return result, nil
}

// loadAllowedComponentTypesForLessonStage 加载教案阶段允许的组件类型。
//
// 优先使用教案关联配方的同名或自定义阶段；
// 配方阶段不存在时回退系统同名阶段；
// 两者均不可用时fail-closed。
func loadAllowedComponentTypesForLessonStage(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
) ([]string, error) {
	if lessonPlan == nil {
		return nil,
			ErrComponentSelectionInvalid
	}

	stageCode = strings.TrimSpace(
		stageCode,
	)

	if stageCode == "" {
		return nil,
			ErrComponentSelectionInvalid
	}

	if !models.IsTeachingEducationDomain(
		strings.ToLower(
			strings.TrimSpace(
				lessonPlan.EducationDomain,
			),
		),
	) {
		return nil,
			ErrComponentEducationDomainInvalid
	}

	if lessonPlan.RecipeID != nil {
		recipeID := strings.TrimSpace(
			*lessonPlan.RecipeID,
		)

		if recipeID != "" {
			recipeStage, recipeErr :=
				repository.GetRecipeStageByCode(
					ctx,
					recipeID,
					stageCode,
				)

			if recipeErr == nil &&
				recipeStage != nil &&
				recipeStage.Status == "active" {
				return parseStageAllowedComponentTypes(
					recipeStage.ComponentTypes,
				)
			}
		}
	}

	systemStage, err :=
		repository.GetStageByCode(
			ctx,
			models.StageSourceSystem,
			stageCode,
		)

	if err != nil ||
		systemStage == nil ||
		systemStage.Status != "active" {
		return nil,
			ErrComponentSelectionInvalid
	}

	return parseStageAllowedComponentTypes(
		systemStage.ComponentTypes,
	)
}

// validateDirectComponentSelectionForStage 严格校验阶段直接ID。
func validateDirectComponentSelectionForStage(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	stageCode string,
	componentIDs []string,
) ([]string, error) {
	normalizedIDs :=
		NormalizeUniqueComponentIDs(
			componentIDs,
		)

	if len(normalizedIDs) == 0 {
		return []string{}, nil
	}

	allowedTypes, err :=
		loadAllowedComponentTypesForLessonStage(
			ctx,
			lessonPlan,
			stageCode,
		)

	if err != nil {
		return nil, err
	}

	// 阶段未配置任何组件类型时，不允许前端自行塞入任意组件。
	if len(allowedTypes) == 0 {
		return nil,
			ErrComponentSelectionInvalid
	}

	return ValidateLessonComponentIDsForUse(
		ctx,
		normalizedIDs,
		lessonPlan.EducationDomain,
		allowedTypes,
	)
}

// resolveAdvanceTargetForSelection 在无副作用阶段解析推进目标。
func (s *WorkshopStageService) resolveAdvanceTargetForSelection(
	lessonPlan *models.LessonPlan,
	targetStageCode string,
) (*models.StageConfigSnapshot, error) {
	snapshots, currentIndex, err :=
		s.resolveStages(lessonPlan)

	if err != nil {
		return nil, err
	}

	targetIndex := -1

	if strings.TrimSpace(
		targetStageCode,
	) != "" {
		targetIndex = findStageIndex(
			snapshots,
			targetStageCode,
		)

		if targetIndex == -1 {
			return nil,
				ErrStageInvalidTarget
		}
	} else {
		targetIndex = currentIndex + 1

		if targetIndex >= len(snapshots) {
			return nil,
				ErrStageAlreadyLast
		}
	}

	target := snapshots[targetIndex]

	return &target, nil
}

// validateStageAdvanceSelection 在原推进方法产生任何副作用前完成验证。
func (s *WorkshopStageService) validateStageAdvanceSelection(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
	componentIDs []string,
) ([]string, error) {
	lessonPlan, err :=
		repository.GetLessonPlanByID(
			ctx,
			lessonPlanID,
		)

	if err != nil {
		return nil, err
	}

	if lessonPlan.AuthorID != callerID {
		return nil,
			ErrLPGenUnauthorized
	}

	targetStage, err :=
		s.resolveAdvanceTargetForSelection(
			lessonPlan,
			targetStageCode,
		)

	if err != nil {
		return nil, err
	}

	return validateDirectComponentSelectionForStage(
		ctx,
		lessonPlan,
		targetStage.StageCode,
		componentIDs,
	)
}

// AdvanceStageValidated 严格验证组件后进入下一阶段。
func (s *WorkshopStageService) AdvanceStageValidated(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
	componentIDs []string,
) (*models.StageConfigSnapshot, error) {
	validatedIDs, err :=
		s.validateStageAdvanceSelection(
			ctx,
			lessonPlanID,
			targetStageCode,
			callerID,
			componentIDs,
		)

	if err != nil {
		return nil, err
	}

	if len(validatedIDs) == 0 {
		return s.AdvanceStage(
			ctx,
			lessonPlanID,
			targetStageCode,
			callerID,
		)
	}

	return s.AdvanceStageWithComponents(
		ctx,
		lessonPlanID,
		targetStageCode,
		callerID,
		validatedIDs,
	)
}

// AdvanceStageSilentValidated 是对话模式的严格验证推进入口。
func (s *WorkshopStageService) AdvanceStageSilentValidated(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
	componentIDs []string,
) (*models.StageConfigSnapshot, error) {
	validatedIDs, err :=
		s.validateStageAdvanceSelection(
			ctx,
			lessonPlanID,
			targetStageCode,
			callerID,
			componentIDs,
		)

	if err != nil {
		return nil, err
	}

	return s.AdvanceStageSilent(
		ctx,
		lessonPlanID,
		targetStageCode,
		callerID,
		validatedIDs,
	)
}

// ChatWithValidatedComponents 是HTTP对话入口使用的严格包装。
//
// 当没有selected_components时直接走原Chat。
// 有直接ID时必须在原Chat登记任务和追加消息之前完成验证。
func (s *LessonPlanGenService) ChatWithValidatedComponents(
	ctx context.Context,
	request *models.LessonPlanChatRequest,
	callerID string,
) error {
	if request == nil {
		return errors.New(
			"对话请求不能为空",
		)
	}

	if len(request.SelectedComponents) == 0 {
		return s.Chat(
			ctx,
			request,
			callerID,
		)
	}

	lessonPlan, err :=
		s.checkPlanEditable(
			ctx,
			request.PlanID,
			callerID,
		)

	if err != nil {
		return err
	}

	validatedIDs, err :=
		validateDirectComponentSelectionForStage(
			ctx,
			lessonPlan,
			lessonPlan.CurrentStage,
			request.SelectedComponents,
		)

	if err != nil {
		return err
	}

	validatedRequest := *request

	validatedRequest.SelectedComponents =
		validatedIDs

	return s.Chat(
		ctx,
		&validatedRequest,
		callerID,
	)
}
