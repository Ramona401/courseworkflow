package services

// lesson_plan_knowledge_lineage_transition.go
// 教学分析阶段离开前的知识脉络同步安全闸。
//
// 本文件放在现有阶段推进逻辑外层，避免改动超长核心Service，
// 同时保留原推进方法的阶段完成、产出创建、SSE开场白和质量评估行为。
//
// 正式顺序：
//   1. 校验教案作者、目标阶段和老师提交的组件ID；
//   2. 若当前阶段是analyze且精确挂载课程大纲，可靠保存分析摘要；
//   3. 从确认后的教学分析对话提取课程锚点；
//   4. 基于课程锚点和课程大纲生成active知识脉络；
//   5. 只有知识脉络成功后，才完成阶段并切换。
//
// 绕过防护与直接出稿例外：
//   - Advance、Silent Advance、Skip和Reset继续执行精确大纲知识脉络安全闸；
//   - Switch默认也执行安全闸，但明确切到write/revise代表用户选择非线性正式出稿，可直接进入目标阶段；
//   - 直接出稿不会伪造active知识脉络，后续提示词装配只在知识脉络真实可用时注入，不可用时让位给课本等事实源；
//   - Back回到analyze允许；一旦分析对话或产出改变，数据库会把旧快照标记为stale。
//
// 竞态防护：
//   - 普通专家模式离开analyze时，先使用Silent原子推进，避免异步质量评估
//     在current_stage仍为analyze时追加消息并误使刚生成的快照stale；
//   - 阶段切换成功后再异步评估analyze，保持原有质量教练能力。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ErrLessonPlanKnowledgeLineageAnalyzeRequired 表示精确挂载课程大纲后，
// 教师必须完成教学分析，不能直接绕到后续阶段。
var ErrLessonPlanKnowledgeLineageAnalyzeRequired = errors.New(
	"已关联课程大纲的教案必须先完成教学分析，确认课文、教学目标和知识点后才能进入下一阶段",
)

// lessonPlanHasExactCourseOutline 判断教案是否存在唯一精确课程大纲ID。
func lessonPlanHasExactCourseOutline(
	ctx context.Context,
	lessonPlanID string,
) (bool, error) {
	snapshot, err :=
		repository.GetLessonPlanCourseOutlineSnapshot(
			ctx,
			strings.TrimSpace(
				lessonPlanID,
			),
		)
	if err != nil {
		return false, err
	}

	return snapshot != nil &&
			snapshot.CourseOutlineID != nil &&
			strings.TrimSpace(
				*snapshot.CourseOutlineID,
			) != "",
		nil
}

// ensureAnalyzeSummaryForKnowledgeLineage 严格保证分析摘要可读取。
func (s *WorkshopStageService) ensureAnalyzeSummaryForKnowledgeLineage(
	ctx context.Context,
	lessonPlanID string,
) error {
	stageOutput, err :=
		repository.GetStageOutput(
			ctx,
			lessonPlanID,
			"analyze",
		)
	if err != nil {
		return fmt.Errorf(
			"读取教学分析阶段产出失败: %w",
			err,
		)
	}

	existingNarrative :=
		strings.TrimSpace(
			stageOutput.NarrativeOutput,
		)

	if len(
		[]rune(existingNarrative),
	) > 100 {
		return nil
	}

	currentMessages, err :=
		repository.GetCurrentStageMessages(
			ctx,
			lessonPlanID,
		)
	if err != nil {
		return fmt.Errorf(
			"读取教学分析阶段对话失败: %w",
			err,
		)
	}

	summary :=
		strings.TrimSpace(
			GenerateStageSummary(
				"analyze",
				currentMessages,
				stageOutput.StructuredOutput,
			),
		)

	if summary == "" {
		return fmt.Errorf(
			"%w: 教学分析尚未形成可保存的确认摘要",
			ErrLessonPlanKnowledgeAnchorsIncomplete,
		)
	}

	if summary == existingNarrative {
		return nil
	}

	if err :=
		repository.UpdateStageNarrativeOutput(
			ctx,
			lessonPlanID,
			"analyze",
			summary,
		); err != nil {
		return fmt.Errorf(
			"保存教学分析确认摘要失败: %w",
			err,
		)
	}

	return nil
}

// prepareConfirmedKnowledgeLineageBeforeAdvance 在原推进产生副作用前，
// 完成教学分析摘要和知识脉络生成。
func (s *WorkshopStageService) prepareConfirmedKnowledgeLineageBeforeAdvance(
	ctx context.Context,
	lessonPlanID string,
	callerID string,
) (prepared bool, err error) {
	lessonPlan, err :=
		repository.GetLessonPlanByID(
			ctx,
			strings.TrimSpace(
				lessonPlanID,
			),
		)
	if err != nil {
		return false, err
	}

	if lessonPlan.AuthorID != callerID {
		return false,
			ErrLPGenUnauthorized
	}

	if strings.TrimSpace(
		lessonPlan.CurrentStage,
	) != "analyze" {
		return false, nil
	}

	hasExactOutline, err :=
		lessonPlanHasExactCourseOutline(
			ctx,
			lessonPlan.ID,
		)
	if err != nil {
		return false,
			fmt.Errorf(
				"读取教案课程大纲绑定失败: %w",
				err,
			)
	}

	if !hasExactOutline {
		return false, nil
	}

	if err :=
		s.ensureAnalyzeSummaryForKnowledgeLineage(
			ctx,
			lessonPlan.ID,
		); err != nil {
		return true, err
	}

	active, err :=
		repository.GetActiveLessonPlanKnowledgeLineage(
			ctx,
			lessonPlan.ID,
		)
	if err != nil {
		return true, err
	}

	if active != nil &&
		active.IsActiveUsable() {
		return true, nil
	}

	lineage, err :=
		s.GenerateConfirmedLessonPlanKnowledgeLineage(
			ctx,
			lessonPlan.ID,
		)
	if err != nil {
		return true, err
	}

	if lineage == nil {
		return true,
			repository.ErrLessonPlanKnowledgeLineageSourceChanged
	}

	if !lineage.IsActiveUsable() {
		return true,
			fmt.Errorf(
				"%w: 生成结果没有形成可供后续阶段使用的active知识脉络",
				ErrLessonPlanKnowledgeLineageExtractionFailed,
			)
	}

	return true, nil
}

// advanceStagePrepared 执行统一的安全推进。
func (s *WorkshopStageService) advanceStagePrepared(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
	componentIDs []string,
	silentEvaluation bool,
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

	preparedAnalyze, err :=
		s.prepareConfirmedKnowledgeLineageBeforeAdvance(
			ctx,
			lessonPlanID,
			callerID,
		)
	if err != nil {
		return nil, err
	}

	if preparedAnalyze {
		shouldEvaluateAfterAdvance :=
			!silentEvaluation &&
				strings.TrimSpace(
					s.aesKey,
				) != "" &&
				s.leavingStageHasSubstantiveContent(
					ctx,
					lessonPlanID,
				)

		targetStage, advanceErr :=
			s.AdvanceStageSilent(
				ctx,
				lessonPlanID,
				targetStageCode,
				callerID,
				validatedIDs,
			)
		if advanceErr != nil {
			return nil, advanceErr
		}

		if shouldEvaluateAfterAdvance {
			go s.asyncLLMEvaluateAndBroadcast(
				context.Background(),
				lessonPlanID,
				"analyze",
			)
		}

		return targetStage, nil
	}

	if silentEvaluation {
		return s.AdvanceStageSilent(
			ctx,
			lessonPlanID,
			targetStageCode,
			callerID,
			validatedIDs,
		)
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

// AdvanceStagePrepared 是HTTP专家模式的统一安全推进入口。
func (s *WorkshopStageService) AdvanceStagePrepared(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
	componentIDs []string,
) (*models.StageConfigSnapshot, error) {
	return s.advanceStagePrepared(
		ctx,
		lessonPlanID,
		targetStageCode,
		callerID,
		componentIDs,
		false,
	)
}

// AdvanceStageSilentPrepared 是HTTP对话模式的统一安全推进入口。
func (s *WorkshopStageService) AdvanceStageSilentPrepared(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
	componentIDs []string,
) (*models.StageConfigSnapshot, error) {
	return s.advanceStagePrepared(
		ctx,
		lessonPlanID,
		targetStageCode,
		callerID,
		componentIDs,
		true,
	)
}

// SkipStagePrepared 是HTTP跳过阶段使用的安全入口。
func (s *WorkshopStageService) SkipStagePrepared(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
) (*models.StageConfigSnapshot, error) {
	lessonPlan, err :=
		repository.GetLessonPlanByID(
			ctx,
			strings.TrimSpace(
				lessonPlanID,
			),
		)
	if err != nil {
		return nil, err
	}

	if lessonPlan.AuthorID != callerID {
		return nil,
			ErrLPGenUnauthorized
	}

	if strings.TrimSpace(
		lessonPlan.CurrentStage,
	) == "analyze" {
		hasExactOutline, outlineErr :=
			lessonPlanHasExactCourseOutline(
				ctx,
				lessonPlan.ID,
			)

		if outlineErr != nil {
			return nil,
				fmt.Errorf(
					"读取教案课程大纲绑定失败: %w",
					outlineErr,
				)
		}

		if hasExactOutline {
			return nil,
				ErrLessonPlanKnowledgeLineageAnalyzeRequired
		}
	}

	return s.SkipStage(
		ctx,
		lessonPlanID,
		targetStageCode,
		callerID,
	)
}

// ensureAnalyzeCanBeLeftWithoutAdvance 防止普通Switch或Reset绕过确认推进。
func (s *WorkshopStageService) ensureAnalyzeCanBeLeftWithoutAdvance(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	targetStageCode string,
) error {
	if lessonPlan == nil ||
		strings.TrimSpace(
			lessonPlan.CurrentStage,
		) != "analyze" ||
		strings.TrimSpace(
			targetStageCode,
		) == "" ||
		strings.TrimSpace(
			targetStageCode,
		) == "analyze" {
		return nil
	}

	hasExactOutline, err :=
		lessonPlanHasExactCourseOutline(
			ctx,
			lessonPlan.ID,
		)
	if err != nil {
		return fmt.Errorf(
			"读取教案课程大纲绑定失败: %w",
			err,
		)
	}

	if !hasExactOutline {
		return nil
	}

	active, err :=
		repository.GetActiveLessonPlanKnowledgeLineage(
			ctx,
			lessonPlan.ID,
		)
	if err != nil {
		return err
	}

	if active != nil &&
		active.IsActiveUsable() {
		return nil
	}

	return ErrLessonPlanKnowledgeLineageAnalyzeRequired
}

func isLessonPlanDirectArtifactSwitchTarget(
	targetStageCode string,
) bool {
	switch strings.ToLower(
		strings.TrimSpace(
			targetStageCode,
		),
	) {
	case "write", "revise":
		return true

	default:
		return false
	}
}

// ensureLessonPlanSwitchTargetOutput 确保非线性切换到write/revise后目标阶段产出记录真实存在。
func ensureLessonPlanSwitchTargetOutput(
	ctx context.Context,
	lessonPlanID string,
	targetStage *models.StageConfigSnapshot,
) error {
	if targetStage == nil {
		return errors.New(
			"直接出稿目标阶段为空",
		)
	}

	outputs, err :=
		repository.ListStageOutputs(
			ctx,
			lessonPlanID,
		)
	if err != nil {
		return fmt.Errorf(
			"检查直接出稿目标阶段产出失败: %w",
			err,
		)
	}

	for _, output := range outputs {
		if output != nil &&
			strings.TrimSpace(
				output.StageCode,
			) ==
				strings.TrimSpace(
					targetStage.StageCode,
				) {
			return nil
		}
	}

	output :=
		&models.WorkshopStageOutput{
			LessonPlanID:         lessonPlanID,
			StageCode:            targetStage.StageCode,
			StageOrder:           targetStage.StageOrder,
			StructuredOutput:     "{}",
			NarrativeOutput:      "",
			ConversationSnapshot: "[]",
			Status:               models.StageOutputInProgress,
		}

	createErr :=
		repository.CreateStageOutput(
			ctx,
			output,
		)
	if createErr == nil {
		return nil
	}

	outputs, readErr :=
		repository.ListStageOutputs(
			ctx,
			lessonPlanID,
		)
	if readErr == nil {
		for _, current := range outputs {
			if current != nil &&
				strings.TrimSpace(
					current.StageCode,
				) ==
					strings.TrimSpace(
						targetStage.StageCode,
					) {
				return nil
			}
		}
	}

	return fmt.Errorf(
		"创建直接出稿目标阶段产出失败: %w",
		createErr,
	)
}

// SwitchToStagePrepared 是HTTP阶段切换的统一安全入口。
//
// 普通阶段切换继续受精确大纲知识脉络闸约束；write/revise属于明确的非线性正式出稿目标，
// 允许直接切入，但必须补齐目标阶段产出与阶段分隔符。知识脉络只在真实可用时注入。
func (s *WorkshopStageService) SwitchToStagePrepared(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
) (*models.StageConfigSnapshot, error) {
	lessonPlan, err :=
		repository.GetLessonPlanByID(
			ctx,
			strings.TrimSpace(
				lessonPlanID,
			),
		)
	if err != nil {
		return nil, err
	}

	if lessonPlan.AuthorID != callerID {
		return nil,
			ErrLPGenUnauthorized
	}

	previousStage :=
		strings.TrimSpace(
			lessonPlan.CurrentStage,
		)

	directArtifactTarget :=
		isLessonPlanDirectArtifactSwitchTarget(
			targetStageCode,
		)

	if !directArtifactTarget {
		if err :=
			s.ensureAnalyzeCanBeLeftWithoutAdvance(
				ctx,
				lessonPlan,
				targetStageCode,
			); err != nil {
			return nil, err
		}
	}

	targetStage, err :=
		s.SwitchToStage(
			ctx,
			lessonPlanID,
			targetStageCode,
			callerID,
		)
	if err != nil {
		return nil, err
	}

	if !directArtifactTarget {
		return targetStage, nil
	}

	if err :=
		ensureLessonPlanSwitchTargetOutput(
			ctx,
			lessonPlanID,
			targetStage,
		); err != nil {
		if previousStage !=
			strings.TrimSpace(
				targetStage.StageCode,
			) {
			rollbackErr :=
				repository.UpdateLessonPlanCurrentStage(
					ctx,
					lessonPlanID,
					previousStage,
				)

			if rollbackErr != nil {
				return nil,
					fmt.Errorf(
						"准备直接出稿阶段失败: %v；阶段回滚失败: %w",
						err,
						rollbackErr,
					)
			}
		}

		return nil, err
	}

	if previousStage !=
		strings.TrimSpace(
			targetStage.StageCode,
		) {
		s.appendStageSeparator(
			ctx,
			lessonPlanID,
			*targetStage,
		)
	}

	wsLog.Info(
		"非线性正式出稿阶段已就绪",
		"plan_id", lessonPlanID,
		"from", previousStage,
		"to", targetStage.StageCode,
	)

	return targetStage, nil
}

// ResetStagePrepared 是HTTP阶段重启的防绕过入口。
func (s *WorkshopStageService) ResetStagePrepared(
	ctx context.Context,
	lessonPlanID string,
	targetStageCode string,
	callerID string,
) (*models.StageConfigSnapshot, error) {
	lessonPlan, err :=
		repository.GetLessonPlanByID(
			ctx,
			strings.TrimSpace(
				lessonPlanID,
			),
		)
	if err != nil {
		return nil, err
	}

	if lessonPlan.AuthorID != callerID {
		return nil,
			ErrLPGenUnauthorized
	}

	if err :=
		s.ensureAnalyzeCanBeLeftWithoutAdvance(
			ctx,
			lessonPlan,
			targetStageCode,
		); err != nil {
		return nil, err
	}

	return s.ResetStage(
		ctx,
		lessonPlanID,
		targetStageCode,
		callerID,
	)
}
