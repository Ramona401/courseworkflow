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
// 绕过防护：
//   - Advance、Silent Advance、Skip、Switch和Reset均经过本文件；
//   - 精确挂载课程大纲且没有可用active知识脉络时，不能从analyze直接进入后续阶段；
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
//
// publisher-only历史关联不进入新知识脉络链；
// 只有正式course_outline_id非空时才启用同步生成和禁止绕过规则。
func lessonPlanHasExactCourseOutline(
        ctx context.Context,
        lessonPlanID string,
) (bool, error) {
        snapshot, err := repository.GetLessonPlanCourseOutlineSnapshot(
                ctx,
                strings.TrimSpace(lessonPlanID),
        )
        if err != nil {
                return false, err
        }

        return snapshot != nil &&
                snapshot.CourseOutlineID != nil &&
                strings.TrimSpace(*snapshot.CourseOutlineID) != "",
                nil
}

// ensureAnalyzeSummaryForKnowledgeLineage 严格保证分析摘要可读取。
//
// 原generateAndSaveEpisodicSummary是best-effort并吞掉错误，适合普通阶段过渡；
// 知识脉络属于强一致性链，不能在摘要读取或保存失败时继续生成。
func (s *WorkshopStageService) ensureAnalyzeSummaryForKnowledgeLineage(
        ctx context.Context,
        lessonPlanID string,
) error {
        stageOutput, err := repository.GetStageOutput(
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

        existingNarrative := strings.TrimSpace(
                stageOutput.NarrativeOutput,
        )
        if len([]rune(existingNarrative)) > 100 {
                return nil
        }

        currentMessages, err := repository.GetCurrentStageMessages(
                ctx,
                lessonPlanID,
        )
        if err != nil {
                return fmt.Errorf(
                        "读取教学分析阶段对话失败: %w",
                        err,
                )
        }

        summary := strings.TrimSpace(
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

        if err := repository.UpdateStageNarrativeOutput(
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
//
// 返回值prepared为true时，表示本次正在离开“精确挂载大纲的analyze阶段”。
// 调用方据此使用竞态安全的Silent推进，再在阶段切换后补跑质量评估。
func (s *WorkshopStageService) prepareConfirmedKnowledgeLineageBeforeAdvance(
        ctx context.Context,
        lessonPlanID string,
        callerID string,
) (prepared bool, err error) {
        lessonPlan, err := repository.GetLessonPlanByID(
                ctx,
                strings.TrimSpace(lessonPlanID),
        )
        if err != nil {
                return false, err
        }

        if lessonPlan.AuthorID != callerID {
                return false, ErrLPGenUnauthorized
        }

        if strings.TrimSpace(lessonPlan.CurrentStage) != "analyze" {
                return false, nil
        }

        hasExactOutline, err := lessonPlanHasExactCourseOutline(
                ctx,
                lessonPlan.ID,
        )
        if err != nil {
                return false, fmt.Errorf(
                        "读取教案课程大纲绑定失败: %w",
                        err,
                )
        }
        if !hasExactOutline {
                return false, nil
        }

        if err := s.ensureAnalyzeSummaryForKnowledgeLineage(
                ctx,
                lessonPlan.ID,
        ); err != nil {
                return true, err
        }

        // 上一次已经成功生成，且教师没有修改分析对话、分析产出或课程大纲时，
        // 直接复用active快照，避免阶段切换失败后的重试再次消耗隐藏AI调用。
        active, err := repository.GetActiveLessonPlanKnowledgeLineage(
                ctx,
                lessonPlan.ID,
        )
        if err != nil {
                return true, err
        }
        if active != nil && active.IsActiveUsable() {
                return true, nil
        }

        lineage, err := s.GenerateConfirmedLessonPlanKnowledgeLineage(
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
                return true, fmt.Errorf(
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
        validatedIDs, err := s.validateStageAdvanceSelection(
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
                                strings.TrimSpace(s.aesKey) != "" &&
                                s.leavingStageHasSubstantiveContent(
                                        ctx,
                                        lessonPlanID,
                                )

                // 先跳过原方法内部的提前异步评估，保证active快照不会在
                // current_stage仍为analyze时被质量教练消息误标为stale。
                targetStage, advanceErr := s.AdvanceStageSilent(
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
        lessonPlan, err := repository.GetLessonPlanByID(
                ctx,
                strings.TrimSpace(lessonPlanID),
        )
        if err != nil {
                return nil, err
        }
        if lessonPlan.AuthorID != callerID {
                return nil, ErrLPGenUnauthorized
        }

        if strings.TrimSpace(lessonPlan.CurrentStage) == "analyze" {
                hasExactOutline, outlineErr :=
                        lessonPlanHasExactCourseOutline(
                                ctx,
                                lessonPlan.ID,
                        )
                if outlineErr != nil {
                        return nil, fmt.Errorf(
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

// ensureAnalyzeCanBeLeftWithoutAdvance 防止Switch或Reset绕过确认推进。
//
// 如果分析阶段已有仍可用的active快照，说明此前已完成过确认且来源未变，
// 允许直接切换；否则必须使用正式Advance完成确认和生成。
func (s *WorkshopStageService) ensureAnalyzeCanBeLeftWithoutAdvance(
        ctx context.Context,
        lessonPlan *models.LessonPlan,
        targetStageCode string,
) error {
        if lessonPlan == nil ||
                strings.TrimSpace(lessonPlan.CurrentStage) != "analyze" ||
                strings.TrimSpace(targetStageCode) == "" ||
                strings.TrimSpace(targetStageCode) == "analyze" {
                return nil
        }

        hasExactOutline, err := lessonPlanHasExactCourseOutline(
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

        active, err := repository.GetActiveLessonPlanKnowledgeLineage(
                ctx,
                lessonPlan.ID,
        )
        if err != nil {
                return err
        }
        if active != nil && active.IsActiveUsable() {
                return nil
        }

        return ErrLessonPlanKnowledgeLineageAnalyzeRequired
}

// SwitchToStagePrepared 是HTTP阶段切换的防绕过入口。
func (s *WorkshopStageService) SwitchToStagePrepared(
        ctx context.Context,
        lessonPlanID string,
        targetStageCode string,
        callerID string,
) (*models.StageConfigSnapshot, error) {
        lessonPlan, err := repository.GetLessonPlanByID(
                ctx,
                strings.TrimSpace(lessonPlanID),
        )
        if err != nil {
                return nil, err
        }
        if lessonPlan.AuthorID != callerID {
                return nil, ErrLPGenUnauthorized
        }

        if err := s.ensureAnalyzeCanBeLeftWithoutAdvance(
                ctx,
                lessonPlan,
                targetStageCode,
        ); err != nil {
                return nil, err
        }

        return s.SwitchToStage(
                ctx,
                lessonPlanID,
                targetStageCode,
                callerID,
        )
}

// ResetStagePrepared 是HTTP阶段重启的防绕过入口。
func (s *WorkshopStageService) ResetStagePrepared(
        ctx context.Context,
        lessonPlanID string,
        targetStageCode string,
        callerID string,
) (*models.StageConfigSnapshot, error) {
        lessonPlan, err := repository.GetLessonPlanByID(
                ctx,
                strings.TrimSpace(lessonPlanID),
        )
        if err != nil {
                return nil, err
        }
        if lessonPlan.AuthorID != callerID {
                return nil, ErrLPGenUnauthorized
        }

        if err := s.ensureAnalyzeCanBeLeftWithoutAdvance(
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
