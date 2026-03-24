package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 包级正则变量（避免每次调用重新编译，提升性能）====================
// extractMetaScores 使用的正则
var (
	reMetaBlock     = regexp.MustCompile(`(?s)<<<META_SCORE>>>(.*?)<<<END_META_SCORE>>>`)
	reTotalFallback = regexp.MustCompile(`(?i)TOTAL_FINAL:\s*([\d.]+)`)
	reMetaTotal     = regexp.MustCompile(`(?i)TOTAL_FINAL:\s*([\d.]+)`)
	reE1Final       = regexp.MustCompile(`(?i)E1_FINAL:\s*([\d.]+)`)
	reE2Final       = regexp.MustCompile(`(?i)E2_FINAL:\s*([\d.]+)`)
	reE3Final       = regexp.MustCompile(`(?i)E3_FINAL:\s*([\d.]+)`)
	reE4Final       = regexp.MustCompile(`(?i)E4_FINAL:\s*([\d.]+)`)
	reMetaHard      = regexp.MustCompile(`(?i)HARD_CONSTRAINT:\s*(PASS|FAIL)`)
	reMetaGrade     = regexp.MustCompile(`(?i)GRADE:\s*([A-D])`)
	reMetaRound     = regexp.MustCompile(`(?i)E([1-4])_R(\d+):\s*([\d.]+)`)
	reFinalScore    = regexp.MustCompile(`(?:综合评分|综合)[：:]\s*[\d.]+\s*→\s*([\d.]+)\s*/\s*10`)
)

// extractEvalScores 使用的正则
var (
	reEvalBlock = regexp.MustCompile(`(?s)<<<SCORE_BLOCK>>>(.*?)<<<END_SCORE_BLOCK>>>`)
	reEvalTotal = regexp.MustCompile(`(?i)TOTAL[:\s]+([\d.]+)`)
	reEvalE1    = regexp.MustCompile(`(?i)E1[:\s]+([\d.]+)`)
	reEvalE2    = regexp.MustCompile(`(?i)E2[:\s]+([\d.]+)`)
	reEvalE3    = regexp.MustCompile(`(?i)E3[:\s]+([\d.]+)`)
	reEvalE4    = regexp.MustCompile(`(?i)E4[:\s]+([\d.]+)`)
	reEvalHard  = regexp.MustCompile(`(?i)HARD_CONSTRAINT[:\s]+(PASS|FAIL)`)
	reEvalGrade = regexp.MustCompile(`(?i)GRADE[:\s]+([A-D])`)
	reEvalDimE1 = regexp.MustCompile(`(?i)E1[^\n]{0,30}[：:]\s*([\d.]+)\s*/\s*10`)
	reEvalDimE2 = regexp.MustCompile(`(?i)E2[^\n]{0,30}[：:]\s*([\d.]+)\s*/\s*10`)
	reEvalDimE3 = regexp.MustCompile(`(?i)E3[^\n]{0,30}[：:]\s*([\d.]+)\s*/\s*10`)
	reEvalDimE4 = regexp.MustCompile(`(?i)E4[^\n]{0,30}[：:]\s*([\d.]+)\s*/\s*10`)
)

// ==================== Pipeline 错误常量 ====================

var (
	ErrPipelineCourseRequired   = errors.New("课程编号不能为空")
	ErrPipelineNotFound         = errors.New("Pipeline不存在")
	ErrPipelineAlreadyExists    = errors.New("该课程已有运行中的Pipeline")
	ErrPipelineCourseNotFound   = errors.New("课程不存在，请先注册课程")
	ErrPipelineNotPending       = errors.New("Pipeline不是待启动状态，无法启动")
	ErrPipelineNotCancellable   = errors.New("Pipeline不在可取消的状态")
	ErrPipelineNotDeletable     = errors.New("运行中的Pipeline不可删除")
	ErrDbCheckIndexMissing      = errors.New("课程索引不存在")
	ErrDbCheckIndexTooShort     = errors.New("课程索引内容过短，可能不完整")
	ErrDbCheckIndexHashMismatch = errors.New("课程索引校验码不一致，数据可能损坏")
	ErrScannerPromptMissing     = errors.New("Prompt A未配置，请先在提示词管理中设置prompt_a")
	ErrScannerAIFailed          = errors.New("Scanner AI调用失败")
	ErrScannerParseFailed       = errors.New("Scanner未能从AI输出中提取有效JSON")
	ErrEvalPromptMissing        = errors.New("Prompt B未配置，请先在提示词管理中设置prompt_b")
	ErrEvalDictMissing          = errors.New("解压缩字典未配置，请先在提示词管理中设置dict")
	ErrEvalScannerNotDone       = errors.New("Scanner步骤未完成，无法执行Evaluator")
	ErrEvalAllRoundsFailed      = errors.New("所有评估轮次均失败")
	ErrMetaPromptMissing        = errors.New("Prompt E未配置，请先在提示词管理中设置prompt_e")
	ErrMetaDictMissing          = errors.New("解压缩字典未配置（Meta需要dict）")
	ErrMetaEvalNotDone          = errors.New("Evaluator步骤未完成，无法执行Meta")
	ErrMetaScannerNotDone       = errors.New("Scanner步骤未完成，无法执行Meta")
	ErrMetaAllRetriesFailed     = errors.New("Meta所有重试均未达标")
	ErrMetaAIFailed             = errors.New("Meta AI调用失败")
	ErrMetaScoreExtractFailed   = errors.New("Meta未能从AI输出中提取META_SCORE评分")
	ErrPipelineNotReviewable    = errors.New("Pipeline不在审核状态，无法进行审核操作")
	ErrPageNotFound             = errors.New("页面不存在")
	ErrInvalidDecision          = errors.New("无效的决策值，必须是approve/reject/edit之一")
	ErrFinalizeIncomplete       = errors.New("尚有未决策的页面，无法定稿")
	ErrMarkPassedNotAllowed     = errors.New("Pipeline不在可快捷通过的状态")
	ErrMarkPassedNotMet         = errors.New("Pipeline评估未达标，无法快捷通过")
	ErrAIFixFailed              = errors.New("AI快修失败")
	ErrEngineQueueFull          = errors.New("执行队列已满，请稍后再试")
	// P7新增：二级审批错误常量
	ErrSubmitFinalizeNotAllowed  = errors.New("Pipeline不在可提交定稿的状态")
	ErrConfirmFinalizeNotAllowed = errors.New("Pipeline不在待确认定稿状态，无法确认")
	ErrRejectFinalizeNotAllowed  = errors.New("Pipeline不在待确认定稿状态，无法退回")
)

// ==================== PipelineService ====================

// PipelineService Pipeline业务逻辑层
type PipelineService struct {
	cfg    *config.Config
	engine *Engine
}

// NewPipelineService 创建Pipeline服务实例
func NewPipelineService(cfg *config.Config) *PipelineService {
	return &PipelineService{cfg: cfg}
}

// SetEngine 注入并发执行引擎
func (s *PipelineService) SetEngine(engine *Engine) {
	s.engine = engine
}

// GetEngine 获取并发执行引擎
func (s *PipelineService) GetEngine() *Engine {
	return s.engine
}

// ==================== Dashboard 统计 ====================

func (s *PipelineService) GetDashboardStats() (*repository.DashboardStats, error) {
	return repository.GetDashboardStats()
}

// ==================== Pipeline CRUD 方法 ====================

// CreatePipeline 创建新Pipeline
func (s *PipelineService) CreatePipeline(req *models.CreatePipelineRequest, userID string) (*models.PipelineDetailResponse, error) {
	courseCode := strings.TrimSpace(req.CourseCode)
	if courseCode == "" {
		return nil, ErrPipelineCourseRequired
	}

	course, err := repository.GetCourseByCode(courseCode)
	if err != nil {
		return nil, ErrPipelineCourseNotFound
	}

	exists, existingID, err := repository.CheckActivePipelineExists(courseCode)
	if err != nil {
		return nil, fmt.Errorf("检查Pipeline状态失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w (ID: %s)", ErrPipelineAlreadyExists, existingID)
	}

	pipelineCfg := models.DefaultPipelineConfig()
	if req.Config != nil {
		if req.Config.EvalRounds > 0 {
			pipelineCfg.EvalRounds = req.Config.EvalRounds
		}
		if req.Config.Threshold > 0 {
			pipelineCfg.Threshold = req.Config.Threshold
		}
		if req.Config.VarianceWarn > 0 {
			pipelineCfg.VarianceWarn = req.Config.VarianceWarn
		}
		if req.Config.MaxMetaRetry > 0 {
			pipelineCfg.MaxMetaRetry = req.Config.MaxMetaRetry
		}
		if req.Config.MaxTRLoop > 0 {
			pipelineCfg.MaxTRLoop = req.Config.MaxTRLoop
		}
	}

	autoMode := true
	if req.AutoMode != nil {
		autoMode = *req.AutoMode
	}

	pipeline := &models.Pipeline{
		CourseCode:       courseCode,
		CourseName:       course.CourseName,
		ExternalModuleID: course.ExternalModuleID,
		StartedBy:        &userID,
		CurrentStep:      models.StepDbCheck,
		Status:           models.PipelineStatusPending,
		AutoMode:         autoMode,
		Config:           pipelineCfg.ToJSON(),
		ReviewRound:      1,
	}

	if err := repository.CreatePipeline(pipeline); err != nil {
		return nil, fmt.Errorf("创建Pipeline失败: %w", err)
	}

	return s.GetPipelineDetail(pipeline.ID)
}

// ListPipelines 获取Pipeline列表（按角色过滤）
func (s *PipelineService) ListPipelines(userID string, role string) (*models.PipelineListResponse, error) {
	items, err := repository.ListPipelinesForUser(userID, role)
	if err != nil {
		return nil, fmt.Errorf("获取Pipeline列表失败: %w", err)
	}
	if items == nil {
		items = []*models.PipelineListItem{}
	}
	return &models.PipelineListResponse{
		Pipelines: items,
		Total:     len(items),
	}, nil
}

// GetPipelineDetail 获取Pipeline详情（含步骤列表）
// v33修复P-03：使用 GetAssignedUserName() 按主键直接查单条用户记录获取人名
// 原版：调用 ListOperatorUsers() 查全量活跃用户表，遍历数组匹配ID，查询代价 O(N)
// 修复后：直接 SELECT display_name FROM users WHERE id=$1，主键索引命中，查询代价 O(1)
func (s *PipelineService) GetPipelineDetail(id string) (*models.PipelineDetailResponse, error) {
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	steps, err := repository.GetStepsByPipelineID(id)
	if err != nil {
		return nil, fmt.Errorf("获取步骤列表失败: %w", err)
	}

	var stepItems []*models.StepListItem
	for _, step := range steps {
		stepItems = append(stepItems, &models.StepListItem{
			ID:           step.ID,
			StepName:     step.StepName,
			StepNameCN:   models.StepNameMap[step.StepName],
			StepOrder:    step.StepOrder,
			Status:       step.Status,
			StatusName:   models.StepStatusNameMap[step.Status],
			StartedAt:    step.StartedAt,
			CompletedAt:  step.CompletedAt,
			DurationMs:   step.DurationMs,
			Attempts:     step.Attempts,
			ModelUsed:    step.ModelUsed,
			TokensUsed:   step.TokensUsed,
			ErrorMessage: step.ErrorMessage,
			HasData:      step.StepData != "" && step.StepData != "null",
		})
	}

	// v33修复P-03：直接按主键查单条用户记录获取display_name
	// 替代原来的 ListOperatorUsers() 全量查询 + 数组遍历匹配
	var assignedName string
	if pipeline.AssignedTo != nil && *pipeline.AssignedTo != "" {
		assignedName = repository.GetAssignedUserName(*pipeline.AssignedTo)
	}

	return &models.PipelineDetailResponse{
		ID:               pipeline.ID,
		CourseCode:       pipeline.CourseCode,
		CourseName:       pipeline.CourseName,
		ExternalModuleID: pipeline.ExternalModuleID,
		CurrentStep:      pipeline.CurrentStep,
		CurrentStepName:  models.StepNameMap[pipeline.CurrentStep],
		Status:           pipeline.Status,
		StatusName:       models.PipelineStatusNameMap[pipeline.Status],
		AutoMode:         pipeline.AutoMode,
		Config:           models.ParsePipelineConfig(pipeline.Config),
		ErrorMessage:     pipeline.ErrorMessage,
		StartedBy:        pipeline.StartedBy,
		StartedAt:        pipeline.StartedAt,
		CompletedAt:      pipeline.CompletedAt,
		CreatedAt:        pipeline.CreatedAt,
		UpdatedAt:        pipeline.UpdatedAt,
		ReviewRound:      pipeline.ReviewRound,
		AssignedTo:       pipeline.AssignedTo,
		AssignedName:     assignedName,
		RejectReason:     pipeline.RejectReason,
		Steps:            stepItems,
	}, nil
}

// GetStepDetail 获取指定步骤的详细信息
func (s *PipelineService) GetStepDetail(pipelineID string, stepName string) (*models.StepDetailResponse, error) {
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	step, err := repository.GetStepByName(pipelineID, stepName)
	if err != nil {
		return nil, fmt.Errorf("步骤 %s 不存在", stepName)
	}

	var stepDataRaw []byte
	if step.StepData != "" && step.StepData != "null" {
		stepDataRaw = []byte(step.StepData)
	} else {
		stepDataRaw = []byte("null")
	}

	return &models.StepDetailResponse{
		ID:           step.ID,
		PipelineID:   step.PipelineID,
		StepName:     step.StepName,
		StepNameCN:   models.StepNameMap[step.StepName],
		StepOrder:    step.StepOrder,
		Status:       step.Status,
		StatusName:   models.StepStatusNameMap[step.Status],
		StartedAt:    step.StartedAt,
		CompletedAt:  step.CompletedAt,
		DurationMs:   step.DurationMs,
		Attempts:     step.Attempts,
		StepData:     stepDataRaw,
		ErrorMessage: step.ErrorMessage,
		ModelUsed:    step.ModelUsed,
		TokensUsed:   step.TokensUsed,
		CreatedAt:    step.CreatedAt,
		UpdatedAt:    step.UpdatedAt,
	}, nil
}

// CancelPipeline 取消Pipeline
func (s *PipelineService) CancelPipeline(id string) error {
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusPending && pipeline.Status != models.PipelineStatusRunning {
		return ErrPipelineNotCancellable
	}
	return repository.UpdatePipelineStatus(id, pipeline.CurrentStep, models.PipelineStatusCancelled)
}

// DeletePipeline 删除Pipeline（运行中不可删）
func (s *PipelineService) DeletePipeline(id string) error {
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status == models.PipelineStatusRunning {
		return ErrPipelineNotDeletable
	}
	return repository.DeletePipeline(id)
}

// ==================== P4.5-D 快捷通过（MarkPassed）====================

// MarkPassed 快捷通过Pipeline
func (s *PipelineService) MarkPassed(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}

	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue: true,
		models.PipelineStatusNeedsHuman:  true,
		models.PipelineStatusFailed:      true,
	}
	if !allowedStatuses[pipeline.Status] {
		return ErrMarkPassedNotAllowed
	}

	metaStep, err := repository.GetStepByName(pipelineID, models.StepMeta)
	if err != nil || metaStep.Status != models.StepStatusDone {
		return ErrMarkPassedNotMet
	}

	var metaData map[string]interface{}
	if metaStep.StepData != "" && metaStep.StepData != "null" {
		if jsonErr := json.Unmarshal([]byte(metaStep.StepData), &metaData); jsonErr != nil {
			return ErrMarkPassedNotMet
		}
	}
	totalFinal, ok := metaData["total_final"].(float64)
	if !ok || totalFinal <= 0 {
		return ErrMarkPassedNotMet
	}

	pCfg := models.ParsePipelineConfig(pipeline.Config)
	if totalFinal < pCfg.Threshold {
		return fmt.Errorf("%w (得分: %.1f, 阈值: %.1f)", ErrMarkPassedNotMet, totalFinal, pCfg.Threshold)
	}

	reviewStep, err := repository.GetStepByName(pipelineID, models.StepReview)
	if err == nil && reviewStep.Status != models.StepStatusDone {
		_ = repository.StartStep(pipelineID, models.StepReview)
		statsJSON := fmt.Sprintf(`{"mark_passed":true,"meta_score":%.1f,"threshold":%.1f,"finalized_at":"%s"}`,
			totalFinal, pCfg.Threshold, time.Now().Format(time.RFC3339))
		_ = repository.CompleteStep(pipelineID, models.StepReview, 0, statsJSON, "", 0)
	}

	if err := repository.CompletePipeline(pipelineID, models.PipelineStatusFinalized); err != nil {
		return fmt.Errorf("快捷通过失败: %w", err)
	}

	return nil
}

// ==================== Pipeline 执行引擎 ====================

// StartPipeline 启动Pipeline执行
func (s *PipelineService) StartPipeline(id string) (*models.PipelineDetailResponse, error) {
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	if pipeline.Status != models.PipelineStatusPending {
		return nil, ErrPipelineNotPending
	}

	if err := repository.UpdatePipelineStatus(id, models.StepDbCheck, models.PipelineStatusRunning); err != nil {
		return nil, fmt.Errorf("更新Pipeline状态失败: %w", err)
	}

	if s.engine != nil {
		task := &EngineTask{
			Type:       TaskTypePipeline,
			PipelineID: id,
			// v33改进：ExecFunc 返回 error，供 Engine 统计业务失败
		ExecFunc: func() error {
			s.executePipelineAsync(id)
			// 检查 Pipeline 最终状态判断执行是否成功
			p, pErr := repository.GetPipelineByID(id)
			if pErr != nil {
				return fmt.Errorf("Pipeline执行后读取状态失败: %w", pErr)
			}
			if p.Status == models.PipelineStatusFailed {
				return fmt.Errorf("Pipeline执行失败: %s", p.ErrorMessage)
			}
			return nil
		},
		}
		if !s.engine.Submit(task) {
			_ = repository.UpdatePipelineStatus(id, models.StepDbCheck, models.PipelineStatusPending)
			return nil, ErrEngineQueueFull
		}
	} else {
		go s.executePipelineAsync(id)
	}

	return s.GetPipelineDetail(id)
}

// executePipelineAsync 异步执行Pipeline全链路
// Phase8修复P-01：扩展断点续跑逻辑，覆盖全部8步
// 原版：只处理Generator和Review两步断点续跑
// 修复后：所有步骤（dbCheck/scanner/evaluator/meta/translator/generator/review/verify）均支持断点续跑
// 逻辑：找到第一个非done步骤 -> 从该步骤开始执行，跳过已完成的步骤
func (s *PipelineService) executePipelineAsync(id string) {
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		_ = repository.UpdatePipelineError(id, models.StepDbCheck, "异步执行: 读取Pipeline失败: "+err.Error())
		return
	}

	// ==================== 断点续跑逻辑（Phase8修复P-01，覆盖全部8步）====================
	// 找到第一个未完成的步骤，从该步骤开始执行
	// 原版只处理Generator/Review，导致Translator/Meta/Scanner/Evaluator失败后从dbCheck重跑，浪费大量AI调用
	existingSteps, stepsErr := repository.GetStepsByPipelineID(id)
	if stepsErr == nil && len(existingSteps) > 0 {
		// 找第一个非done的步骤
		var resumeStep string
		for _, st := range existingSteps {
			if st.Status != models.StepStatusDone {
				resumeStep = st.StepName
				break
			}
		}

		// 有需要恢复的步骤，且不是从头开始（dbCheck）
		if resumeStep != "" && resumeStep != models.StepDbCheck {
			if err := repository.UpdatePipelineStatus(id, resumeStep, models.PipelineStatusRunning); err != nil {
				_ = repository.UpdatePipelineError(id, resumeStep, "恢复Pipeline失败: "+err.Error())
				return
			}
			pipeline, _ = repository.GetPipelineByID(id)

			// 根据断点步骤分发执行（覆盖全部8步）
			switch resumeStep {

			case models.StepScanner:
				// 从Scanner断点续跑
				s.broadcastStepUpdate(id, "step_update", "scanner", "running", "running", "断点续跑: 从Scanner继续")
				if scanErr := s.executeScanner(pipeline); scanErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepScanner, scanErr.Error())
					s.broadcastStepUpdate(id, "pipeline_error", "scanner", "failed", "failed", scanErr.Error())
					return
				}
				// Scanner完成后继续后续步骤
				resumeStep = models.StepEvaluator
				fallthrough

			case models.StepEvaluator:
				// 从Evaluator断点续跑（或Scanner完成后顺序执行）
				if err := repository.UpdatePipelineStatus(id, models.StepEvaluator, models.PipelineStatusRunning); err != nil {
					_ = repository.UpdatePipelineError(id, models.StepEvaluator, "推进到Evaluator失败: "+err.Error())
					return
				}
				s.broadcastStepUpdate(id, "step_update", "evaluator", "running", "running", "断点续跑: 开始Evaluator")
				pipeline, _ = repository.GetPipelineByID(id)
				if evalErr := s.executeEvaluator(pipeline); evalErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepEvaluator, evalErr.Error())
					s.broadcastStepUpdate(id, "pipeline_error", "evaluator", "failed", "failed", evalErr.Error())
					return
				}
				resumeStep = models.StepMeta
				fallthrough

			case models.StepMeta:
				// 从Meta断点续跑（或Evaluator完成后顺序执行）
				if err := repository.UpdatePipelineStatus(id, models.StepMeta, models.PipelineStatusRunning); err != nil {
					_ = repository.UpdatePipelineError(id, models.StepMeta, "推进到Meta失败: "+err.Error())
					return
				}
				s.broadcastStepUpdate(id, "step_update", "meta", "running", "running", "断点续跑: 开始Meta")
				pipeline, _ = repository.GetPipelineByID(id)
				if metaErr := s.executeMeta(pipeline); metaErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepMeta, metaErr.Error())
					s.broadcastStepUpdate(id, "pipeline_error", "meta", "failed", "failed", metaErr.Error())
					return
				}
				resumeStep = models.StepTranslator
				fallthrough

			case models.StepTranslator:
				// 从Translator断点续跑（或Meta完成后顺序执行）
				if err := repository.UpdatePipelineStatus(id, models.StepTranslator, models.PipelineStatusRunning); err != nil {
					_ = repository.UpdatePipelineError(id, models.StepTranslator, "推进到Translator失败: "+err.Error())
					return
				}
				s.broadcastStepUpdate(id, "step_update", "translator", "running", "running", "断点续跑: 开始Translator")
				pipeline, _ = repository.GetPipelineByID(id)
				if transErr := s.executeTranslator(pipeline); transErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepTranslator, transErr.Error())
					s.broadcastStepUpdate(id, "pipeline_error", "translator", "failed", "failed", transErr.Error())
					return
				}
				resumeStep = models.StepGenerator
				fallthrough

			case models.StepGenerator:
				// 从Generator断点续跑（或Translator完成后顺序执行）
				if err := repository.UpdatePipelineStatus(id, models.StepGenerator, models.PipelineStatusRunning); err != nil {
					_ = repository.UpdatePipelineError(id, models.StepGenerator, "推进到Generator失败: "+err.Error())
					return
				}
				s.broadcastStepUpdate(id, "step_update", "generator", "running", "running", "断点续跑: 开始Generator")
				pipeline, _ = repository.GetPipelineByID(id)
				if genErr := s.executeGenerator(pipeline); genErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepGenerator, genErr.Error())
					s.broadcastStepUpdate(id, "pipeline_error", "generator", "failed", "failed", genErr.Error())
					return
				}
				// Generator完成后进入审核队列
				_ = repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue)
				s.broadcastStepUpdate(id, "pipeline_done", "review", "done", "review_queue", "Pipeline执行完成，等待审核")
				return

			case models.StepReview:
				// Review步骤断点续跑：直接恢复到审核队列，等待人工操作
				_ = repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue)
				s.broadcastStepUpdate(id, "pipeline_done", "review", "pending", "review_queue", "断点续跑: 恢复到审核队列")
				return

			case models.StepVerify:
				// Verify步骤断点续跑：不自动重跑验收，恢复到finalized等待手动触发
				// 验收流程有独立入口（/verify），断点不应自动触发
				_ = repository.UpdatePipelineStatus(id, models.StepVerify, models.PipelineStatusFinalized)
				return

			default:
				// 未知步骤：从dbCheck全量重跑（保险起见）
			}
		}
	}
	// ==================== 正常执行（全量/从dbCheck开始）====================

	dbCheckErr := s.executeDbCheck(pipeline)
	if dbCheckErr != nil {
		_ = repository.UpdatePipelineError(id, models.StepDbCheck, dbCheckErr.Error())
		s.broadcastStepUpdate(id, "pipeline_error", "dbCheck", "failed", "failed", dbCheckErr.Error())
		return
	}

	if err := repository.UpdatePipelineStatus(id, models.StepScanner, models.PipelineStatusRunning); err != nil {
		_ = repository.UpdatePipelineError(id, models.StepScanner, "推进到Scanner失败: "+err.Error())
		return
	}
	s.broadcastStepUpdate(id, "step_update", "scanner", "running", "running", "dbCheck完成，开始Scanner")

	if pipeline.AutoMode {
		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			_ = repository.UpdatePipelineError(id, models.StepScanner, "读取Pipeline失败: "+err.Error())
			return
		}

		scannerErr := s.executeScanner(pipeline)
		if scannerErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepScanner, scannerErr.Error())
			s.broadcastStepUpdate(id, "pipeline_error", "scanner", "failed", "failed", scannerErr.Error())
			return
		}

		if err := repository.UpdatePipelineStatus(id, models.StepEvaluator, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepEvaluator, "推进到Evaluator失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "evaluator", "running", "running", "Scanner完成，开始Evaluator")

		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			_ = repository.UpdatePipelineError(id, models.StepEvaluator, "读取Pipeline失败: "+err.Error())
			return
		}

		evalErr := s.executeEvaluator(pipeline)
		if evalErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepEvaluator, evalErr.Error())
			s.broadcastStepUpdate(id, "pipeline_error", "evaluator", "failed", "failed", evalErr.Error())
			return
		}

		if err := repository.UpdatePipelineStatus(id, models.StepMeta, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepMeta, "推进到Meta失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "meta", "running", "running", "Evaluator完成，开始Meta")

		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			_ = repository.UpdatePipelineError(id, models.StepMeta, "读取Pipeline失败: "+err.Error())
			return
		}

		metaErr := s.executeMeta(pipeline)
		if metaErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepMeta, metaErr.Error())
			s.broadcastStepUpdate(id, "pipeline_error", "meta", "failed", "failed", metaErr.Error())
			return
		}

		if err := repository.UpdatePipelineStatus(id, models.StepTranslator, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepTranslator, "推进到Translator失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "translator", "running", "running", "Meta完成，开始Translator")

		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			_ = repository.UpdatePipelineError(id, models.StepTranslator, "读取Pipeline失败: "+err.Error())
			return
		}

		transErr := s.executeTranslator(pipeline)
		if transErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepTranslator, transErr.Error())
			s.broadcastStepUpdate(id, "pipeline_error", "translator", "failed", "failed", transErr.Error())
			return
		}

		if err := repository.UpdatePipelineStatus(id, models.StepGenerator, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepGenerator, "推进到Generator失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "generator", "running", "running", "Translator完成，开始Generator")

		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			_ = repository.UpdatePipelineError(id, models.StepGenerator, "读取Pipeline失败: "+err.Error())
			return
		}

		genErr := s.executeGenerator(pipeline)
		if genErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepGenerator, genErr.Error())
			s.broadcastStepUpdate(id, "pipeline_error", "generator", "failed", "failed", genErr.Error())
			return
		}

		if err := repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepReview, "推进到Review失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "pipeline_done", "review", "done", "review_queue", "Pipeline执行完成，等待审核")
	}
}

// ==================== AI调用包装方法 ====================

func (s *PipelineService) callAIWithSemaphore(cfg *ai.EffectiveConfig, systemPrompt string, userPrompt string) (*ai.CallResult, error) {
	if s.engine != nil {
		s.engine.AcquireAI()
		defer s.engine.ReleaseAI()
	}
	return ai.CallAI(cfg, systemPrompt, userPrompt)
}

// ==================== P5-4 SSE事件广播 ====================

func (s *PipelineService) broadcastStepUpdate(pipelineID string, eventType string, currentStep string, stepStatus string, pipelineStatus string, message string) {
	event := SSEEvent{
		EventType:   eventType,
		PipelineID:  pipelineID,
		CurrentStep: currentStep,
		StepStatus:  stepStatus,
		Status:      pipelineStatus,
		Message:     message,
	}
	GlobalSSEHub.Broadcast(pipelineID, event)
}

// ==================== dbCheck 步骤 ====================

func (s *PipelineService) executeDbCheck(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepDbCheck

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动dbCheck失败: %w", err)
	}

	result := &models.DbCheckResult{CourseCode: pipeline.CourseCode}
	checkErr := s.doDbCheck(pipeline, result)
	durationMs := time.Since(startTime).Milliseconds()

	if checkErr != nil {
		result.IsValid = false
		result.ErrorDetail = checkErr.Error()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, checkErr.Error())
		s.saveStepData(pipeline.ID, stepName, result.ToJSON())
		return checkErr
	}

	result.IsValid = true
	if err := repository.CompleteStep(pipeline.ID, stepName, durationMs, result.ToJSON(), "", 0); err != nil {
		return fmt.Errorf("保存dbCheck结果失败: %w", err)
	}
	return nil
}

func (s *PipelineService) doDbCheck(pipeline *models.Pipeline, result *models.DbCheckResult) error {
	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		return fmt.Errorf("课程 %s 不存在", pipeline.CourseCode)
	}
	result.CourseID = course.ID
	if course.ExternalModuleID != nil {
		result.ModuleID = *course.ExternalModuleID
	}

	idx, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		result.HasIndex = false
		return ErrDbCheckIndexMissing
	}
	result.HasIndex = true
	result.IndexHash = idx.IndexHash
	result.PageCount = idx.PageCount
	result.TotalLength = idx.TotalLength

	if len(idx.IndexContent) < models.MinIndexLength {
		return fmt.Errorf("%w (实际长度: %d, 最小要求: %d)",
			ErrDbCheckIndexTooShort, len(idx.IndexContent), models.MinIndexLength)
	}

	actualHash := utils.SHA256Hash(idx.IndexContent)
	if actualHash != idx.IndexHash {
		return fmt.Errorf("%w (存储: %s, 实际: %s)",
			ErrDbCheckIndexHashMismatch, idx.IndexHash[:16]+"...", actualHash[:16]+"...")
	}

	return nil
}

// ==================== Scanner 步骤 ====================

func (s *PipelineService) executeScanner(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepScanner

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动scanner失败: %w", err)
	}

	result, callErr := s.doScanner(pipeline)
	durationMs := time.Since(startTime).Milliseconds()

	if callErr != nil {
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, callErr.Error())
		if result != nil {
			s.saveStepData(pipeline.ID, stepName, result.ToJSON())
		}
		return callErr
	}

	if err := repository.CompleteStep(
		pipeline.ID, stepName, durationMs,
		result.ToJSON(), result.ModelUsed, result.TokensUsed,
	); err != nil {
		return fmt.Errorf("保存scanner结果失败: %w", err)
	}

	return nil
}

func (s *PipelineService) doScanner(pipeline *models.Pipeline) (*models.ScannerResult, error) {
	result := &models.ScannerResult{}

	promptA, err := repository.GetCurrentPromptByKey("prompt_a")
	if err != nil || promptA == nil {
		return nil, ErrScannerPromptMissing
	}

	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		return nil, fmt.Errorf("scanner: 课程 %s 不存在", pipeline.CourseCode)
	}
	courseIndex, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		return nil, fmt.Errorf("scanner: 课程索引不存在，请先执行dbCheck")
	}

	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey,
		"scanner",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf("scanner: 获取AI配置失败: %w", err)
	}

	systemPrompt := promptA.Content
	userPrompt := fmt.Sprintf("请分析以下课程索引内容，按照要求输出JSON格式的定位信息：\n\n%s", courseIndex.IndexContent)

	callResult, err := s.callAIWithSemaphore(aiCfg, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrScannerAIFailed, err.Error())
	}

	result.RawOutput = callResult.Content
	result.ModelUsed = callResult.ModelUsed
	result.TokensUsed = callResult.TokensUsed

	jsonStr, ok := ai.ExtractJSON(callResult.Content)
	if !ok {
		result.IsValid = false
		return result, fmt.Errorf("%w (AI输出前200字符: %s)",
			ErrScannerParseFailed, truncate(callResult.Content, 200))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		result.IsValid = false
		return result, fmt.Errorf("%w (JSON解析错误: %s)", ErrScannerParseFailed, err.Error())
	}

	if _, hasTarget := parsed["target"]; !hasTarget {
		result.IsValid = false
		return result, fmt.Errorf("%w (缺少必要字段target，实际字段: %v)",
			ErrScannerParseFailed, getMapKeys(parsed))
	}

	result.Parsed = json.RawMessage(jsonStr)
	result.IsValid = true

	return result, nil
}

// ==================== Evaluator 步骤 ====================

func (s *PipelineService) executeEvaluator(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepEvaluator

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动evaluator失败: %w", err)
	}

	pCfg := models.ParsePipelineConfig(pipeline.Config)
	totalRounds := pCfg.EvalRounds

	promptB, err := repository.GetCurrentPromptByKey("prompt_b")
	if err != nil || promptB == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrEvalPromptMissing.Error())
		return ErrEvalPromptMissing
	}

	dict, err := repository.GetCurrentPromptByKey("dict")
	if err != nil || dict == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrEvalDictMissing.Error())
		return ErrEvalDictMissing
	}

	abilityTable, _ := repository.GetCurrentPromptByKey("ability_table")

	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("evaluator: 课程 %s 不存在", pipeline.CourseCode)
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}
	courseIndex, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := "evaluator: 课程索引不存在"
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrEvalScannerNotDone.Error())
		return ErrEvalScannerNotDone
	}
	scannerLocationJSON := extractScannerParsed(scannerStep.StepData)

	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey,
		"evaluator",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("evaluator: 获取AI配置失败: %s", err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	systemPrompt := promptB.Content

	userPromptParts := []string{
		"【课程定位】",
		scannerLocationJSON,
		"",
		"【待评估索引】",
		courseIndex.IndexContent,
		"",
		"【TE-DNA解压缩字典】",
		dict.Content,
	}
	if abilityTable != nil && len(abilityTable.Content) > 20 {
		userPromptParts = append(userPromptParts, "", "【能力定位表】", abilityTable.Content)
	}
	userPromptParts = append(userPromptParts, "", "禁止输出<thinking>标签或任何思维过程标记。")
	userPrompt := strings.Join(userPromptParts, "\n")

	_ = repository.DeleteEvalRoundsByPipelineID(pipeline.ID)

	evalResult := &models.EvaluatorResult{
		TotalRounds: totalRounds,
	}
	var roundScores []float64
	var totalTokens int
	var doneCount, failCount int
	var lastModelUsed string

	for i := 1; i <= totalRounds; i++ {
		roundRec, err := repository.CreateEvalRound(pipeline.ID, i)
		if err != nil {
			failCount++
			continue
		}

		_ = repository.UpdateEvalRoundRunning(roundRec.ID)

		callResult, callErr := s.callAIWithSemaphore(aiCfg, systemPrompt, userPrompt)
		if callErr != nil {
			_ = repository.FailEvalRound(roundRec.ID, "", callErr.Error())
			failCount++
			continue
		}

		output := callResult.Content
		lastModelUsed = callResult.ModelUsed
		totalTokens += callResult.TokensUsed

		scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4, hardConstraint, grade, parseOk := extractEvalScores(output)

		if !parseOk || scoreTotal < 0 {
			_ = repository.FailEvalRound(roundRec.ID, truncate(output, 5000), "评分提取失败")
			failCount++
			continue
		}

		dimMap := map[string]interface{}{
			"hard_constraint": hardConstraint,
			"grade":           grade,
		}
		dimJSON, _ := json.Marshal(dimMap)

		err = repository.CompleteEvalRound(
			roundRec.ID, truncate(output, 50000),
			scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4,
			string(dimJSON), callResult.ModelUsed, callResult.TokensUsed,
		)
		if err != nil {
			failCount++
			continue
		}

		doneCount++
		roundScores = append(roundScores, scoreTotal)
	}

	durationMs := time.Since(startTime).Milliseconds()

	if doneCount == 0 {
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrEvalAllRoundsFailed.Error())
		return ErrEvalAllRoundsFailed
	}

	var sumTotal, sumE1, sumE2, sumE3, sumE4 float64
	rounds, _ := repository.GetEvalRoundsByPipelineID(pipeline.ID)
	for _, r := range rounds {
		if r.Status == models.StepStatusDone && r.ScoreTotal != nil {
			sumTotal += *r.ScoreTotal
			if r.ScoreE1 != nil {
				sumE1 += *r.ScoreE1
			}
			if r.ScoreE2 != nil {
				sumE2 += *r.ScoreE2
			}
			if r.ScoreE3 != nil {
				sumE3 += *r.ScoreE3
			}
			if r.ScoreE4 != nil {
				sumE4 += *r.ScoreE4
			}
		}
	}
	n := float64(doneCount)
	evalResult.DoneRounds = doneCount
	evalResult.FailedRounds = failCount
	evalResult.AvgTotal = math.Round(sumTotal/n*10) / 10
	evalResult.AvgE1 = math.Round(sumE1/n*10) / 10
	evalResult.AvgE2 = math.Round(sumE2/n*10) / 10
	evalResult.AvgE3 = math.Round(sumE3/n*10) / 10
	evalResult.AvgE4 = math.Round(sumE4/n*10) / 10
	evalResult.RoundScores = roundScores
	evalResult.TotalTokens = totalTokens
	evalResult.TotalLatencyMs = durationMs
	evalResult.ModelUsed = lastModelUsed

	if doneCount >= 2 {
		var sumSqDiff float64
		for _, sc := range roundScores {
			diff := sc - evalResult.AvgTotal
			sumSqDiff += diff * diff
		}
		evalResult.Variance = math.Round(sumSqDiff/n*100) / 100
		evalResult.VarianceWarn = evalResult.Variance > pCfg.VarianceWarn
	}

	if err := repository.CompleteStep(
		pipeline.ID, stepName, durationMs,
		evalResult.ToJSON(), lastModelUsed, totalTokens,
	); err != nil {
		return fmt.Errorf("保存evaluator结果失败: %w", err)
	}

	return nil
}

// ==================== Meta 步骤 ====================

func (s *PipelineService) executeMeta(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepMeta

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动meta失败: %w", err)
	}

	pCfg := models.ParsePipelineConfig(pipeline.Config)
	threshold := pCfg.Threshold
	maxRetry := pCfg.MaxMetaRetry

	promptE, err := repository.GetCurrentPromptByKey("prompt_e")
	if err != nil || promptE == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaPromptMissing.Error())
		return ErrMetaPromptMissing
	}

	dict, err := repository.GetCurrentPromptByKey("dict")
	if err != nil || dict == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaDictMissing.Error())
		return ErrMetaDictMissing
	}

	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("meta: 课程 %s 不存在", pipeline.CourseCode)
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}
	courseIndex, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := "meta: 课程索引不存在"
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaScannerNotDone.Error())
		return ErrMetaScannerNotDone
	}
	scannerLocationJSON := extractScannerParsed(scannerStep.StepData)

	evalStep, err := repository.GetStepByName(pipeline.ID, models.StepEvaluator)
	if err != nil || evalStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaEvalNotDone.Error())
		return ErrMetaEvalNotDone
	}
	evalRounds, err := repository.GetEvalRoundsByPipelineID(pipeline.ID)
	if err != nil || len(evalRounds) == 0 {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := "meta: 无法获取评估轮次数据"
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	var roundsTextParts []string
	doneCount := 0
	for _, r := range evalRounds {
		if r.Status == models.StepStatusDone && r.Output != "" {
			doneCount++
			scoreStr := "?"
			if r.ScoreTotal != nil {
				scoreStr = fmt.Sprintf("%.1f", *r.ScoreTotal)
			}
			roundsTextParts = append(roundsTextParts,
				fmt.Sprintf("=== 【评估报告%d/%d】（综合: %s）===\n%s",
					r.RoundNumber, len(evalRounds), scoreStr, r.Output))
		}
	}
	if doneCount == 0 {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := "meta: 无有效的评估报告（所有轮次均失败）"
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}
	roundsText := strings.Join(roundsTextParts, "\n\n")

	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey,
		"meta",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("meta: 获取AI配置失败: %s", err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	systemPrompt := promptE.Content

	userPromptParts := []string{
		"【课程定位】",
		scannerLocationJSON,
		"",
		"【待评估索引（原始）】",
		courseIndex.IndexContent,
		"",
		"【多轮评估结果（共" + fmt.Sprintf("%d", doneCount) + "轮）】",
		roundsText,
		"",
		"【TE-DNA解压缩字典】",
		dict.Content,
		"",
		"禁止输出<thinking>标签或任何思维过程标记。",
	}
	userPrompt := strings.Join(userPromptParts, "\n")

	metaResult := &models.MetaResult{
		TotalRetries: maxRetry,
	}
	var lastOutput string
	var lastModelUsed string
	var totalTokens int
	var totalLatencyMs int64

	for attempt := 1; attempt <= maxRetry; attempt++ {
		metaResult.Attempt = attempt

		callResult, callErr := s.callAIWithSemaphore(aiCfg, systemPrompt, userPrompt)
		if callErr != nil {
			totalLatencyMs += time.Since(startTime).Milliseconds() - totalLatencyMs
			if attempt == maxRetry {
				durationMs := time.Since(startTime).Milliseconds()
				errMsg := fmt.Sprintf("%s (第%d次): %s", ErrMetaAIFailed.Error(), attempt, callErr.Error())
				_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
				return fmt.Errorf(errMsg)
			}
			continue
		}

		lastOutput = callResult.Content
		lastModelUsed = callResult.ModelUsed
		totalTokens += callResult.TokensUsed
		totalLatencyMs += callResult.LatencyMs

		scoreResult := extractMetaScores(lastOutput)

		if !scoreResult.parseOk {
			if attempt == maxRetry {
				durationMs := time.Since(startTime).Milliseconds()
				metaResult.RawOutput = truncate(lastOutput, 50000)
				metaResult.ModelUsed = lastModelUsed
				metaResult.TokensUsed = totalTokens
				metaResult.LatencyMs = totalLatencyMs
				s.saveStepData(pipeline.ID, stepName, metaResult.ToJSON())
				_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaScoreExtractFailed.Error())
				return ErrMetaScoreExtractFailed
			}
			continue
		}

		metaResult.TotalFinal = scoreResult.totalFinal
		metaResult.E1Final = scoreResult.e1Final
		metaResult.E2Final = scoreResult.e2Final
		metaResult.E3Final = scoreResult.e3Final
		metaResult.E4Final = scoreResult.e4Final
		metaResult.HardConstraint = scoreResult.hardConstraint
		metaResult.Grade = scoreResult.grade
		metaResult.E1Rounds = scoreResult.e1Rounds
		metaResult.E2Rounds = scoreResult.e2Rounds
		metaResult.E3Rounds = scoreResult.e3Rounds
		metaResult.E4Rounds = scoreResult.e4Rounds

		passed := metaResult.TotalFinal >= threshold
		metaResult.Passed = passed
		metaResult.RawOutput = truncate(lastOutput, 50000)
		metaResult.ModelUsed = lastModelUsed
		metaResult.TokensUsed = totalTokens
		metaResult.LatencyMs = totalLatencyMs

		if passed {
			durationMs := time.Since(startTime).Milliseconds()
			if err := repository.CompleteStep(
				pipeline.ID, stepName, durationMs,
				metaResult.ToJSON(), lastModelUsed, totalTokens,
			); err != nil {
				return fmt.Errorf("保存meta结果失败: %w", err)
			}
			return nil
		}

		if attempt == maxRetry {
			durationMs := time.Since(startTime).Milliseconds()
			s.saveStepData(pipeline.ID, stepName, metaResult.ToJSON())
			errMsg := fmt.Sprintf("%s (最终得分: %.1f, 阈值: %.1f, 共%d次尝试)",
				ErrMetaAllRetriesFailed.Error(), metaResult.TotalFinal, threshold, maxRetry)
			_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
			return fmt.Errorf(errMsg)
		}
	}

	durationMs := time.Since(startTime).Milliseconds()
	_ = repository.FailStep(pipeline.ID, stepName, durationMs, "meta: 异常退出重试循环")
	return fmt.Errorf("meta: 异常退出重试循环")
}

// ==================== Meta 评分提取 ====================

type metaScoreResult struct {
	totalFinal     float64
	e1Final        float64
	e2Final        float64
	e3Final        float64
	e4Final        float64
	hardConstraint string
	grade          string
	e1Rounds       []float64
	e2Rounds       []float64
	e3Rounds       []float64
	e4Rounds       []float64
	parseOk        bool
}

func extractMetaScores(output string) *metaScoreResult {
	result := &metaScoreResult{}

	blockMatch := reMetaBlock.FindStringSubmatch(output)

	if len(blockMatch) < 2 {
		tfm := reTotalFallback.FindStringSubmatch(output)
		if tfm != nil {
			result.totalFinal = safeParseFloat(tfm[1])
			if result.totalFinal > 0 {
				result.parseOk = true
			}
		}
		return result
	}

	block := blockMatch[1]

	tm := reMetaTotal.FindStringSubmatch(block)
	if tm == nil {
		return result
	}
	result.totalFinal = safeParseFloat(tm[1])

	if m := reE1Final.FindStringSubmatch(block); m != nil {
		result.e1Final = safeParseFloat(m[1])
	}
	if m := reE2Final.FindStringSubmatch(block); m != nil {
		result.e2Final = safeParseFloat(m[1])
	}
	if m := reE3Final.FindStringSubmatch(block); m != nil {
		result.e3Final = safeParseFloat(m[1])
	}
	if m := reE4Final.FindStringSubmatch(block); m != nil {
		result.e4Final = safeParseFloat(m[1])
	}

	if hm := reMetaHard.FindStringSubmatch(block); hm != nil {
		result.hardConstraint = hm[1]
	}

	if gm := reMetaGrade.FindStringSubmatch(block); gm != nil {
		result.grade = gm[1]
	}

	allRoundMatches := reMetaRound.FindAllStringSubmatch(block, -1)

	roundMap := map[int]map[int]float64{
		1: {}, 2: {}, 3: {}, 4: {},
	}
	maxRound := 0
	for _, m := range allRoundMatches {
		dim, _ := strconv.Atoi(m[1])
		rn, _ := strconv.Atoi(m[2])
		score := safeParseFloat(m[3])
		if dim >= 1 && dim <= 4 && rn >= 1 {
			roundMap[dim][rn] = score
			if rn > maxRound {
				maxRound = rn
			}
		}
	}

	for rn := 1; rn <= maxRound; rn++ {
		result.e1Rounds = append(result.e1Rounds, roundMap[1][rn])
		result.e2Rounds = append(result.e2Rounds, roundMap[2][rn])
		result.e3Rounds = append(result.e3Rounds, roundMap[3][rn])
		result.e4Rounds = append(result.e4Rounds, roundMap[4][rn])
	}

	if fsm := reFinalScore.FindStringSubmatch(output); fsm != nil {
		newScore := safeParseFloat(fsm[1])
		if newScore > 0 {
			result.totalFinal = newScore
		}
	}

	result.parseOk = true
	return result
}

// ==================== P4.5-C 审核决策方法 ====================

// GetGeneratedPages 获取Pipeline的生成页面列表（含完整HTML）
func (s *PipelineService) GetGeneratedPages(pipelineID string) ([]*repository.GeneratedPageFullRow, error) {
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	pages, err := repository.GetGeneratedPagesWithHTML(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("获取生成页面失败: %w", err)
	}

	if pages == nil {
		pages = []*repository.GeneratedPageFullRow{}
	}
	return pages, nil
}

// UpdatePageDecision 更新单页审核决策
// P7更新：新增 pending_finalize 状态也允许超级审核员修改决策
func (s *PipelineService) UpdatePageDecision(pipelineID string, pageNumber int, decision string, finalHTML *string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	// review_queue / needs_human / pending_finalize 三种状态均允许修改决策
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue:     true,
		models.PipelineStatusNeedsHuman:      true,
		models.PipelineStatusPendingFinalize: true,
	}
	if !allowedStatuses[pipeline.Status] {
		return ErrPipelineNotReviewable
	}

	validDecisions := map[string]bool{"approve": true, "reject": true, "edit": true}
	if !validDecisions[decision] {
		return ErrInvalidDecision
	}

	if decision == "edit" && (finalHTML == nil || *finalHTML == "") {
		return fmt.Errorf("edit决策必须提供修改后的HTML内容")
	}

	if err := repository.UpdatePageDecision(pipelineID, pageNumber, decision, finalHTML); err != nil {
		return err
	}

	return nil
}

// SubmitFinalize 提交定稿（审核员→待超级审核员确认）
// P7新增：审核员完成逐页决策后，点击"提交定稿"，状态从review_queue变为pending_finalize
func (s *PipelineService) SubmitFinalize(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	// 仅 review_queue / needs_human 状态允许提交定稿
	if pipeline.Status != models.PipelineStatusReviewQueue &&
		pipeline.Status != models.PipelineStatusNeedsHuman {
		return ErrSubmitFinalizeNotAllowed
	}

	// 检查所有页面已决策
	total, decided, err := repository.GetPageDecisionStats(pipelineID)
	if err != nil {
		return fmt.Errorf("检查页面决策状态失败: %w", err)
	}
	if total == 0 {
		return fmt.Errorf("该Pipeline没有生成页面，无法提交定稿")
	}
	if decided < total {
		return fmt.Errorf("%w (总页面: %d, 已决策: %d, 未决策: %d)",
			ErrFinalizeIncomplete, total, decided, total-decided)
	}

	// 状态变为 pending_finalize（待超级审核员确认）
	if err := repository.UpdatePipelineStatus(pipelineID, models.StepReview, models.PipelineStatusPendingFinalize); err != nil {
		return fmt.Errorf("提交定稿失败: %w", err)
	}

	return nil
}

// ConfirmFinalize 确认定稿（超级审核员→finalized）
// P7新增：超级审核员在审核中心确认定稿，状态从pending_finalize变为finalized
func (s *PipelineService) ConfirmFinalize(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusPendingFinalize {
		return ErrConfirmFinalizeNotAllowed
	}

	// 获取页面统计
	total, decided, err := repository.GetPageDecisionStats(pipelineID)
	if err != nil {
		return fmt.Errorf("检查页面决策状态失败: %w", err)
	}

	// 完成review步骤
	reviewStep, err := repository.GetStepByName(pipelineID, models.StepReview)
	if err == nil && reviewStep.Status != models.StepStatusDone {
		_ = repository.StartStep(pipelineID, models.StepReview)
		statsJSON := fmt.Sprintf(`{"total_pages":%d,"decided_pages":%d,"finalized_at":"%s"}`,
			total, decided, time.Now().Format(time.RFC3339))
		_ = repository.CompleteStep(pipelineID, models.StepReview, 0, statsJSON, "", 0)
	}

	// 状态变为 finalized
	if err := repository.CompletePipeline(pipelineID, models.PipelineStatusFinalized); err != nil {
		return fmt.Errorf("确认定稿失败: %w", err)
	}

	return nil
}

// RejectFinalize 退回重审（超级审核员→review_queue，退回给原assigned_to审核员）
// P7新增：超级审核员退回给原来的审核员重新审核
// Phase8修复P-02：退回原因现在会持久化到 pipelines.reject_reason 字段
//
//	原版：rejectReason参数接收后直接丢弃，无法溯源
//	修复后：调用 UpdatePipelineRejectReason 将原因写入数据库，审核员可在审核页面看到退回理由
func (s *PipelineService) RejectFinalize(pipelineID string, rejectReason string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusPendingFinalize {
		return ErrRejectFinalizeNotAllowed
	}

	// Phase8修复P-02：一条SQL同时更新状态+退回原因，保持 assigned_to 不变
	if err := repository.UpdatePipelineRejectReason(pipelineID, rejectReason); err != nil {
		return fmt.Errorf("退回重审失败: %w", err)
	}

	return nil
}

// FinalizePipeline 直接定稿（保持向后兼容，admin可直接定稿跳过二级审批）
func (s *PipelineService) FinalizePipeline(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	// admin直接定稿：review_queue / needs_human / pending_finalize 均允许
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue:     true,
		models.PipelineStatusNeedsHuman:      true,
		models.PipelineStatusPendingFinalize: true,
	}
	if !allowedStatuses[pipeline.Status] {
		return ErrPipelineNotReviewable
	}

	total, decided, err := repository.GetPageDecisionStats(pipelineID)
	if err != nil {
		return fmt.Errorf("检查页面决策状态失败: %w", err)
	}
	if total == 0 {
		return fmt.Errorf("该Pipeline没有生成页面，无法定稿")
	}
	if decided < total {
		return fmt.Errorf("%w (总页面: %d, 已决策: %d, 未决策: %d)",
			ErrFinalizeIncomplete, total, decided, total-decided)
	}

	reviewStep, err := repository.GetStepByName(pipelineID, models.StepReview)
	if err == nil && reviewStep.Status != models.StepStatusDone {
		_ = repository.StartStep(pipelineID, models.StepReview)
		statsJSON := fmt.Sprintf(`{"total_pages":%d,"decided_pages":%d,"finalized_at":"%s"}`,
			total, decided, time.Now().Format(time.RFC3339))
		_ = repository.CompleteStep(pipelineID, models.StepReview, 0, statsJSON, "", 0)
	}

	if err := repository.CompletePipeline(pipelineID, models.PipelineStatusFinalized); err != nil {
		return fmt.Errorf("定稿失败: %w", err)
	}

	return nil
}

// ==================== 公共工具方法 ====================

func extractScannerParsed(stepData string) string {
	if stepData == "" || stepData == "null" {
		return "（无课程定位数据）"
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stepData), &data); err != nil {
		return "（课程定位数据解析失败）"
	}
	parsed, ok := data["parsed"]
	if !ok || string(parsed) == "null" {
		return "（无课程定位数据）"
	}
	var obj interface{}
	if err := json.Unmarshal(parsed, &obj); err != nil {
		return string(parsed)
	}
	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return string(parsed)
	}
	return string(pretty)
}

func extractEvalScores(output string) (float64, float64, float64, float64, float64, string, string, bool) {
	sbMatch := reEvalBlock.FindStringSubmatch(output)

	if len(sbMatch) >= 2 {
		block := sbMatch[1]

		tm := reEvalTotal.FindStringSubmatch(block)
		e1m := reEvalE1.FindStringSubmatch(block)
		e2m := reEvalE2.FindStringSubmatch(block)
		e3m := reEvalE3.FindStringSubmatch(block)
		e4m := reEvalE4.FindStringSubmatch(block)
		hm := reEvalHard.FindStringSubmatch(block)
		gm := reEvalGrade.FindStringSubmatch(block)

		if tm != nil {
			scoreTotal := safeParseFloat(tm[1])
			scoreE1 := safeParseFloat(getSubmatch(e1m))
			scoreE2 := safeParseFloat(getSubmatch(e2m))
			scoreE3 := safeParseFloat(getSubmatch(e3m))
			scoreE4 := safeParseFloat(getSubmatch(e4m))
			hardConstraint := ""
			if hm != nil {
				hardConstraint = hm[1]
			}
			grade := ""
			if gm != nil {
				grade = gm[1]
			}
			return scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4, hardConstraint, grade, true
		}
	}

	var scoreE1, scoreE2, scoreE3, scoreE4 float64
	dimCount := 0
	dimPatterns := []struct {
		re    *regexp.Regexp
		field *float64
	}{
		{reEvalDimE1, &scoreE1},
		{reEvalDimE2, &scoreE2},
		{reEvalDimE3, &scoreE3},
		{reEvalDimE4, &scoreE4},
	}
	for _, dp := range dimPatterns {
		m := dp.re.FindStringSubmatch(output)
		if m != nil {
			*dp.field = safeParseFloat(m[1])
			dimCount++
		}
	}

	if dimCount >= 3 {
		vals := []float64{scoreE1, scoreE2, scoreE3, scoreE4}
		sum := 0.0
		cnt := 0
		for _, v := range vals {
			if v > 0 {
				sum += v
				cnt++
			}
		}
		if cnt > 0 {
			scoreTotal := math.Round(sum/float64(cnt)*10) / 10
			return scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4, "", "", true
		}
	}

	return -1, 0, 0, 0, 0, "", "", false
}

// ==================== 工具方法 ====================

func (s *PipelineService) saveStepData(pipelineID string, stepName string, data string) {
	if data == "" || data == "{}" {
		return
	}
	ctx := context.Background()
	_, _ = database.DB.Exec(ctx,
		`UPDATE pipeline_steps SET step_data = $3::jsonb, updated_at = NOW()
		 WHERE pipeline_id = $1 AND step_name = $2`,
		pipelineID, stepName, data)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func safeParseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return v
}

func getSubmatch(m []string) string {
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// ==================== Eval Rounds 查询 ====================

type EvalRoundDetail struct {
	ID             string   `json:"id"`
	RoundNumber    int      `json:"round_number"`
	Status         string   `json:"status"`
	Output         string   `json:"output"`
	ScoreTotal     *float64 `json:"score_total"`
	ScoreE1        *float64 `json:"score_e1"`
	ScoreE2        *float64 `json:"score_e2"`
	ScoreE3        *float64 `json:"score_e3"`
	ScoreE4        *float64 `json:"score_e4"`
	HardConstraint string   `json:"hard_constraint"`
	Grade          string   `json:"grade"`
	ModelUsed      string   `json:"model_used"`
	TokensUsed     int      `json:"tokens_used"`
}

func (s *PipelineService) GetEvalRounds(pipelineID string) ([]*EvalRoundDetail, error) {
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	rounds, err := repository.GetEvalRoundsByPipelineID(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("获取评估轮次失败: %w", err)
	}

	var details []*EvalRoundDetail
	for _, r := range rounds {
		detail := &EvalRoundDetail{
			ID:          r.ID,
			RoundNumber: r.RoundNumber,
			Status:      r.Status,
			Output:      r.Output,
			ScoreTotal:  r.ScoreTotal,
			ScoreE1:     r.ScoreE1,
			ScoreE2:     r.ScoreE2,
			ScoreE3:     r.ScoreE3,
			ScoreE4:     r.ScoreE4,
			ModelUsed:   r.ModelUsed,
			TokensUsed:  r.TokensUsed,
		}
		if r.Dimensions != "" && r.Dimensions != "null" {
			var dims map[string]interface{}
			if jsonErr := json.Unmarshal([]byte(r.Dimensions), &dims); jsonErr == nil {
				if hc, ok := dims["hard_constraint"].(string); ok {
					detail.HardConstraint = hc
				}
				if g, ok := dims["grade"].(string); ok {
					detail.Grade = g
				}
			}
		}
		details = append(details, detail)
	}

	if details == nil {
		details = []*EvalRoundDetail{}
	}
	return details, nil
}

// ==================== P4.5-E-2 AI快修方法 ====================

func (s *PipelineService) AIFixPage(pipelineID string, pageNumber int, fixInstruction string) (string, error) {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return "", ErrPipelineNotFound
	}
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue:     true,
		models.PipelineStatusNeedsHuman:      true,
		models.PipelineStatusFinalized:       true,
		models.PipelineStatusPendingFinalize: true,
	}
	if !allowedStatuses[pipeline.Status] {
		return "", ErrPipelineNotReviewable
	}

	pages, err := repository.GetGeneratedPagesWithHTML(pipelineID)
	if err != nil {
		return "", fmt.Errorf("获取页面数据失败: %w", err)
	}
	var currentPage *repository.GeneratedPageFullRow
	for _, p := range pages {
		if p.PageNumber == pageNumber {
			currentPage = p
			break
		}
	}
	if currentPage == nil {
		return "", ErrPageNotFound
	}

	currentHTML := currentPage.FinalHTML
	if currentHTML == "" {
		currentHTML = currentPage.GeneratedHTML
	}
	if currentHTML == "" {
		currentHTML = currentPage.OriginalHTML
	}
	if currentHTML == "" {
		return "", fmt.Errorf("页面P%d无可用HTML内容", pageNumber)
	}

	promptF, err := repository.GetCurrentPromptByKey("prompt_f")
	if err != nil || promptF == nil {
		return "", fmt.Errorf("Prompt F未配置，无法执行AI快修")
	}

	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey,
		"generator",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", fmt.Errorf("获取AI配置失败: %w", err)
	}

	userPrompt := "【⚠️⚠️⚠️ 最重要 — 当前页面HTML，你必须在此基础上修复，禁止重写】\n" +
		"以下是当前页面的完整HTML。你的输出必须以此为基础，只修改下方修复指令要求的部分，其余代码原封不动保留。\n\n" +
		currentHTML + "\n\n" +
		"══════════════════════════════════════════════\n" +
		"▲ 以上是当前HTML（必须作为修改基础） ▼ 以下是修复指令\n" +
		"══════════════════════════════════════════════\n\n" +
		fmt.Sprintf("【页面】P%02d — %s\n", pageNumber, currentPage.PageTitle) +
		"【操作类型】AI快修（在当前HTML基础上修复指定问题）\n\n" +
		"【审核员修复指令（必须严格执行）】\n" +
		fixInstruction + "\n\n" +
		"【⚠️ 最终提醒】你的输出必须与上方HTML有90%以上代码重合。只改指令要求的部分。导航栏、视频、图片不允许任何改动。输出完整HTML。"

	callResult, callErr := s.callAIWithSemaphore(aiCfg, promptF.Content, userPrompt)
	if callErr != nil {
		return "", fmt.Errorf("%w: %s", ErrAIFixFailed, callErr.Error())
	}

	newHTML := extractGeneratedHTML(callResult.Content)
	if len(newHTML) < 100 {
		return "", fmt.Errorf("%w: AI输出HTML过短(%d字符)", ErrAIFixFailed, len(newHTML))
	}

	if err := repository.UpdateGeneratedPageHTML(pipelineID, pageNumber, newHTML, newHTML); err != nil {
		return "", fmt.Errorf("保存修复后HTML失败: %w", err)
	}

	return newHTML, nil
}

// ==================== P5-3 批量创建+批量启动 ====================

type BatchCreateResult struct {
	TotalRequested int      `json:"total_requested"`
	CreatedIDs     []string `json:"created_ids"`
	SkippedCodes   []string `json:"skipped_codes"`
	SkippedReasons []string `json:"skipped_reasons"`
	FailedCodes    []string `json:"failed_codes"`
	FailedReasons  []string `json:"failed_reasons"`
}

func (s *PipelineService) BatchCreatePipelines(courseCodes []string, userID string) (*BatchCreateResult, error) {
	result := &BatchCreateResult{
		TotalRequested: len(courseCodes),
		CreatedIDs:     []string{},
		SkippedCodes:   []string{},
		SkippedReasons: []string{},
		FailedCodes:    []string{},
		FailedReasons:  []string{},
	}

	seen := make(map[string]bool)
	var uniqueCodes []string
	for _, code := range courseCodes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if seen[code] {
			continue
		}
		seen[code] = true
		uniqueCodes = append(uniqueCodes, code)
	}

	for _, code := range uniqueCodes {
		req := &models.CreatePipelineRequest{
			CourseCode: code,
		}
		resp, err := s.CreatePipeline(req, userID)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "已有运行中的Pipeline") ||
				strings.Contains(errMsg, "课程不存在") {
				result.SkippedCodes = append(result.SkippedCodes, code)
				result.SkippedReasons = append(result.SkippedReasons, code+": "+errMsg)
			} else {
				result.FailedCodes = append(result.FailedCodes, code)
				result.FailedReasons = append(result.FailedReasons, code+": "+errMsg)
			}
			continue
		}
		result.CreatedIDs = append(result.CreatedIDs, resp.ID)
	}

	return result, nil
}

type BatchStartResult struct {
	TotalRequested int      `json:"total_requested"`
	StartedIDs     []string `json:"started_ids"`
	SkippedIDs     []string `json:"skipped_ids"`
	SkippedReasons []string `json:"skipped_reasons"`
	FailedIDs      []string `json:"failed_ids"`
	FailedReasons  []string `json:"failed_reasons"`
}

func (s *PipelineService) BatchStartPipelines(ids []string) (*BatchStartResult, error) {
	result := &BatchStartResult{
		TotalRequested: len(ids),
		StartedIDs:     []string{},
		SkippedIDs:     []string{},
		SkippedReasons: []string{},
		FailedIDs:      []string{},
		FailedReasons:  []string{},
	}

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		_, err := s.StartPipeline(id)
		if err != nil {
			errMsg := err.Error()
			if err == ErrPipelineNotPending || err == ErrPipelineNotFound {
				result.SkippedIDs = append(result.SkippedIDs, id)
				result.SkippedReasons = append(result.SkippedReasons, id+": "+errMsg)
			} else {
				result.FailedIDs = append(result.FailedIDs, id)
				result.FailedReasons = append(result.FailedReasons, id+": "+errMsg)
			}
			continue
		}
		result.StartedIDs = append(result.StartedIDs, id)
	}

	return result, nil
}

// ==================== P6-2 Pipeline分配服务方法 ====================

type AssignPipelineResult struct {
	PipelineID   string `json:"pipeline_id"`
	AssignedTo   string `json:"assigned_to"`
	AssignedName string `json:"assigned_name"`
}

type BatchAssignResult struct {
	TotalRequested int      `json:"total_requested"`
	SuccessCount   int      `json:"success_count"`
	AssignedTo     string   `json:"assigned_to"`
	AssignedName   string   `json:"assigned_name"`
	FailedIDs      []string `json:"failed_ids"`
}

// AssignPipeline 分配Pipeline给指定审核员
// v33修复P-03：使用 GetAssignedUserName() 替代 ListOperatorUsers() 获取人名
func (s *PipelineService) AssignPipeline(pipelineID string, assignedToUserID string) (*AssignPipelineResult, error) {
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	var assignPtr *string
	if assignedToUserID != "" {
		assignPtr = &assignedToUserID
	}
	if err := repository.AssignPipeline(pipelineID, assignPtr); err != nil {
		return nil, fmt.Errorf("分配失败: %w", err)
	}

	// v33修复P-03：直接按主键查单条用户记录获取display_name
	assignedName := repository.GetAssignedUserName(assignedToUserID)

	return &AssignPipelineResult{
		PipelineID:   pipelineID,
		AssignedTo:   assignedToUserID,
		AssignedName: assignedName,
	}, nil
}

// BatchAssignPipelines 批量分配Pipeline给指定审核员
// v33修复P-03：使用 GetAssignedUserName() 替代 ListOperatorUsers() 获取人名
func (s *PipelineService) BatchAssignPipelines(pipelineIDs []string, assignedToUserID string) (*BatchAssignResult, error) {
	var assignPtr *string
	if assignedToUserID != "" {
		assignPtr = &assignedToUserID
	}

	successCount, err := repository.BatchAssignPipelines(pipelineIDs, assignPtr)
	if err != nil {
		return nil, fmt.Errorf("批量分配失败: %w", err)
	}

	// v33修复P-03：直接按主键查单条用户记录获取display_name
	assignedName := repository.GetAssignedUserName(assignedToUserID)

	return &BatchAssignResult{
		TotalRequested: len(pipelineIDs),
		SuccessCount:   successCount,
		AssignedTo:     assignedToUserID,
		AssignedName:   assignedName,
		FailedIDs:      []string{},
	}, nil
}

func (s *PipelineService) GetOperatorUsers() ([]map[string]string, error) {
	return repository.ListOperatorUsers()
}
