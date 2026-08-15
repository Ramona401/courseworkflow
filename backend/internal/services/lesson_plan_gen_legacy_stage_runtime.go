package services

// lesson_plan_gen_legacy_stage_runtime.go — 遗留完整教案阶段运行态恢复
//
// 历史旧Fork曾通过遗留创建链丢失forked_from，且正文虽完整却没有stage_config/current_stage。
// 因此恢复条件不能再依赖“是不是Fork”，而必须依赖数据库可验证的完整教案事实。
//
// 统一恢复语义复用“导入已有完整教案”的正式状态机：
//   - 保留现有正文、version、conversation_log、评审状态和所有挂载字段；
//   - 只读合并系统/配方阶段快照；
//   - review之前阶段记为skipped；
//   - review作为唯一in_progress当前阶段；
//   - 阶段配置、current_stage和阶段output由Repository单事务写入。
//
// 已有合法阶段运行态时只幂等确保当前阶段output存在，不改历史状态。
// 本函数由Chat在lesson_plan_ai单任务登记之后调用，因此同一教案不会并发执行两次恢复。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func (s *LessonPlanGenService) ensureLegacyCompleteLessonPlanStageRuntimeForChat(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
) (*models.LessonPlan, error) {
	if lessonPlan == nil {
		return nil, errors.New(
			"教案对象为空",
		)
	}

	stageConfig :=
		strings.TrimSpace(
			lessonPlan.StageConfig,
		)
	currentStage :=
		strings.TrimSpace(
			lessonPlan.CurrentStage,
		)

	if stageConfig != "" &&
		stageConfig != "[]" &&
		currentStage != "" {
		if err := s.stageService.EnsureStageOutput(
			ctx,
			lessonPlan.ID,
			currentStage,
		); err != nil {
			return nil, fmt.Errorf(
				"校验教案阶段运行态失败: %w",
				err,
			)
		}

		return lessonPlan, nil
	}

	if strings.TrimSpace(
		lessonPlan.ContentMarkdown,
	) == "" {
		return nil, fmt.Errorf(
			"%w: 历史教案正文为空，不能按完整教案恢复阶段运行态",
			ErrStageNotInitialized,
		)
	}

	recipeID := ""
	recipeStagesConfig := ""

	if lessonPlan.RecipeID != nil {
		recipeID =
			strings.TrimSpace(
				*lessonPlan.RecipeID,
			)
	}

	if recipeID != "" {
		recipe, recipeErr :=
			repository.GetRecipeByID(
				ctx,
				recipeID,
			)
		if recipeErr == nil &&
			recipe != nil {
			recipeStagesConfig =
				recipe.StagesConfig
		} else {
			lpGenLog.Warn(
				"遗留完整教案读取配方阶段配置失败，使用系统默认阶段恢复",
				"plan_id", lessonPlan.ID,
				"recipe_id", recipeID,
				"error", recipeErr,
			)
		}
	}

	snapshots, err :=
		s.stageService.MergeStages(
			ctx,
			recipeStagesConfig,
			recipeID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"合并遗留完整教案阶段失败: %w",
			err,
		)
	}

	stageOutputs,
		skippedStages,
		outputErr :=
		buildImportedLessonPlanStageOutputs(
			snapshots,
		)
	if outputErr != nil {
		return nil, fmt.Errorf(
			"构建遗留完整教案阶段恢复状态失败: %w",
			outputErr,
		)
	}
	if len(stageOutputs) == 0 {
		return nil, fmt.Errorf(
			"构建遗留完整教案阶段恢复状态失败: 阶段产出为空",
		)
	}

	stageConfigJSON, marshalErr :=
		json.Marshal(
			snapshots,
		)
	if marshalErr != nil {
		return nil, fmt.Errorf(
			"序列化遗留完整教案阶段快照失败: %w",
			marshalErr,
		)
	}

	currentStage =
		stageOutputs[len(stageOutputs)-1].
			StageCode

	for index := range stageOutputs {
		stageOutputs[index].LessonPlanID =
			lessonPlan.ID
	}

	repaired, repairErr :=
		repository.RepairLegacyCompleteLessonPlanStageRuntime(
			ctx,
			lessonPlan.ID,
			lessonPlan.AuthorID,
			string(stageConfigJSON),
			currentStage,
			stageOutputs,
		)
	if repairErr != nil {
		return nil, fmt.Errorf(
			"恢复遗留完整教案阶段运行态失败: %w",
			repairErr,
		)
	}

	refreshed, err :=
		repository.GetLessonPlanByID(
			ctx,
			lessonPlan.ID,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"恢复遗留完整教案阶段运行态后重新读取失败: %w",
			err,
		)
	}

	if err := s.stageService.EnsureStageOutput(
		ctx,
		refreshed.ID,
		refreshed.CurrentStage,
	); err != nil {
		return nil, fmt.Errorf(
			"恢复遗留完整教案阶段运行态后校验当前产出失败: %w",
			err,
		)
	}

	if repaired {
		forkedFrom := ""
		if lessonPlan.ForkedFrom != nil {
			forkedFrom =
				strings.TrimSpace(
					*lessonPlan.ForkedFrom,
				)
		}

		lpGenLog.Info(
			"遗留完整教案缺失阶段运行态，已在首次Chat前恢复为review-ready",
			"plan_id", lessonPlan.ID,
			"forked_from", forkedFrom,
			"current_stage", refreshed.CurrentStage,
			"skipped_stages", skippedStages,
			"stages_count", len(snapshots),
		)
	}

	return refreshed, nil
}
