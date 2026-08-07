package services

// workshop_stage_service.go — 阶段化备课工坊基础服务
//
// 本文件只保留稳定、轻量的服务入口：
//   - 服务依赖及Setter；
//   - 系统阶段、阶段进度和阶段产出查询；
//   - 阶段产出保存；
//   - 阶段快照解析。
//
// 已拆分职责：
//   - 阶段前进、跳过、回退、切换和重启：workshop_stage_flow.go；
//   - 质量评估和Episodic摘要：workshop_stage_quality_memory.go；
//   - 资料与提示词装配：workshop_stage_context_loader.go；
//   - 自然阶段衔接：workshop_stage_transition.go；
//   - 阶段初始化与合并：workshop_stage_merge.go；
//   - 阶段组件和自定义阶段：workshop_stage_components.go。
//
// 阶段是后台工作状态，不是教师需要感知和维护的对话仪式。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrStageNotInitialized = errors.New("教案尚未初始化阶段配置")
	ErrStageAlreadyFirst   = errors.New("已经是第一个阶段，无法回退")
	ErrStageAlreadyLast    = errors.New("已经是最后一个阶段")
	ErrStageNotSkippable   = errors.New("当前阶段不可跳过")
	ErrStageInvalidTarget  = errors.New("目标阶段不存在")
	ErrCustomStageLimit    = errors.New("自定义阶段数量已达上限（最多10个）")
)

// lessonPlanStageChatService 是阶段推进后自然续接对话所需的最小接口。
type lessonPlanStageChatService interface {
	Chat(
		ctx context.Context,
		request *models.LessonPlanChatRequest,
		callerID string,
	) error
}

// WorkshopStageService 是阶段化备课工坊服务。
type WorkshopStageService struct {
	recipeService   *RecipeService
	genService      lessonPlanStageChatService
	aesKey          string
	textbookService *TextbookService
}

var wsLog = logger.WithModule("workshop_stage")

// NewWorkshopStageService 创建阶段服务实例。
func NewWorkshopStageService() *WorkshopStageService {
	return &WorkshopStageService{
		recipeService: NewRecipeService(),
	}
}

// SetGenService 注入生成服务，避免包内循环依赖。
func (s *WorkshopStageService) SetGenService(
	genService lessonPlanStageChatService,
) {
	s.genService = genService
}

// SetAESKey 注入阶段质量评估使用的密钥。
func (s *WorkshopStageService) SetAESKey(key string) {
	s.aesKey = key
}

// SetTextbookService 注入课本服务。
func (s *WorkshopStageService) SetTextbookService(
	textbookService *TextbookService,
) {
	s.textbookService = textbookService
}

// GetDefaultStages 获取系统默认阶段。
func (s *WorkshopStageService) GetDefaultStages(
	ctx context.Context,
) (*models.DefaultStagesResponse, error) {
	stages, err := repository.GetSystemDefaultStages(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取默认阶段失败: %w", err)
	}

	items := make([]*models.DefaultStageItem, 0, len(stages))
	for _, stage := range stages {
		if stage == nil {
			continue
		}

		items = append(items, &models.DefaultStageItem{
			StageCode:      stage.StageCode,
			StageName:      stage.StageName,
			StageOrder:     stage.StageOrder,
			AIRole:         stage.AIRole,
			GateMode:       stage.GateMode,
			Skippable:      stage.Skippable,
			ComponentTypes: stage.ComponentTypes,
		})
	}

	return &models.DefaultStagesResponse{
		Stages: items,
	}, nil
}

// GetStageStatus 获取教案阶段进度。
func (s *WorkshopStageService) GetStageStatus(
	ctx context.Context,
	lessonPlanID string,
	callerID string,
) (*models.StageStatusResponse, error) {
	lessonPlan, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lessonPlan.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	var snapshots []models.StageConfigSnapshot
	if lessonPlan.StageConfig != "" && lessonPlan.StageConfig != "[]" {
		_ = json.Unmarshal([]byte(lessonPlan.StageConfig), &snapshots)
	}
	if len(snapshots) == 0 {
		return nil, ErrStageNotInitialized
	}

	outputs, _ := repository.ListStageOutputs(ctx, lessonPlanID)
	outputByStage := make(
		map[string]*models.WorkshopStageOutput,
		len(outputs),
	)

	for _, output := range outputs {
		if output != nil {
			outputByStage[output.StageCode] = output
		}
	}

	items := make(
		[]*models.StageProgressItem,
		0,
		len(snapshots),
	)

	for _, snapshot := range snapshots {
		item := &models.StageProgressItem{
			StageCode:  snapshot.StageCode,
			StageName:  snapshot.StageName,
			StageOrder: snapshot.StageOrder,
			AIRole:     snapshot.AIRole,
			GateMode:   snapshot.GateMode,
			Skippable:  snapshot.Skippable,
			Status:     "pending",
			IsCustom:   snapshot.IsCustom,
		}

		if output := outputByStage[snapshot.StageCode]; output != nil {
			item.Status = output.Status
			item.HasOutput = output.StructuredOutput != "" &&
				output.StructuredOutput != "{}"
			item.CompletedAt = output.CompletedAt
		}

		items = append(items, item)
	}

	return &models.StageStatusResponse{
		CurrentStage: lessonPlan.CurrentStage,
		TotalStages:  len(snapshots),
		Stages:       items,
	}, nil
}

// GetStageOutput 获取单个阶段产出。
func (s *WorkshopStageService) GetStageOutput(
	ctx context.Context,
	lessonPlanID string,
	stageCode string,
	callerID string,
) (*models.StageOutputResponse, error) {
	lessonPlan, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}
	if lessonPlan.AuthorID != callerID {
		return nil, ErrLPGenUnauthorized
	}

	output, err := repository.GetStageOutput(
		ctx,
		lessonPlanID,
		stageCode,
	)
	if err != nil {
		return nil, err
	}

	return &models.StageOutputResponse{
		StageCode:       output.StageCode,
		StageName:       stageCodeToName(output.StageCode),
		StructuredOutput: output.StructuredOutput,
		NarrativeOutput: output.NarrativeOutput,
		Status:          output.Status,
		ModelUsed:       output.ModelUsed,
		TokensUsed:      output.TokensUsed,
	}, nil
}

// SaveStageOutput 保存阶段结构化产出和自然语言摘要。
func (s *WorkshopStageService) SaveStageOutput(
	ctx context.Context,
	lessonPlanID string,
	stageCode string,
	structuredJSON string,
	narrative string,
	modelUsed string,
	tokensUsed int,
) error {
	return repository.UpdateStageOutputContent(
		ctx,
		lessonPlanID,
		stageCode,
		structuredJSON,
		narrative,
		modelUsed,
		tokensUsed,
	)
}

// resolveStages 解析阶段快照并定位当前阶段。
func (s *WorkshopStageService) resolveStages(
	lessonPlan *models.LessonPlan,
) ([]models.StageConfigSnapshot, int, error) {
	var snapshots []models.StageConfigSnapshot

	if lessonPlan.StageConfig != "" && lessonPlan.StageConfig != "[]" {
		_ = json.Unmarshal(
			[]byte(lessonPlan.StageConfig),
			&snapshots,
		)
	}

	if len(snapshots) == 0 {
		return nil, -1, ErrStageNotInitialized
	}

	currentIndex := findStageIndex(
		snapshots,
		lessonPlan.CurrentStage,
	)
	if currentIndex == -1 {
		return nil, -1, fmt.Errorf(
			"当前阶段 %s 不在配置中",
			lessonPlan.CurrentStage,
		)
	}

	return snapshots, currentIndex, nil
}
