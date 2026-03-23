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
	// P4-4新增：Meta步骤错误常量
	ErrMetaPromptMissing      = errors.New("Prompt E未配置，请先在提示词管理中设置prompt_e")
	ErrMetaDictMissing        = errors.New("解压缩字典未配置（Meta需要dict）")
	ErrMetaEvalNotDone        = errors.New("Evaluator步骤未完成，无法执行Meta")
	ErrMetaScannerNotDone     = errors.New("Scanner步骤未完成，无法执行Meta")
	ErrMetaAllRetriesFailed   = errors.New("Meta所有重试均未达标")
	ErrMetaAIFailed           = errors.New("Meta AI调用失败")
	ErrMetaScoreExtractFailed = errors.New("Meta未能从AI输出中提取META_SCORE评分")
	// P4.5-C新增：审核决策错误常量
	ErrPipelineNotReviewable = errors.New("Pipeline不在审核状态，无法进行审核操作")
	ErrPageNotFound          = errors.New("页面不存在")
	ErrInvalidDecision       = errors.New("无效的决策值，必须是approve/reject/edit之一")
	ErrFinalizeIncomplete    = errors.New("尚有未决策的页面，无法定稿")
	// P4.5-D新增：快捷通过错误常量
	ErrMarkPassedNotAllowed = errors.New("Pipeline不在可快捷通过的状态")
	ErrMarkPassedNotMet     = errors.New("Pipeline评估未达标，无法快捷通过")
	// P4.5-E-2新增：AI快修错误常量
	ErrAIFixFailed = errors.New("AI快修失败")
	// P5-2新增：并发引擎错误常量
	ErrEngineQueueFull = errors.New("执行队列已满，请稍后再试")
)

// ==================== PipelineService ====================

// PipelineService Pipeline业务逻辑层
// P5-2更新：新增engine字段，支持并发执行和AI限流
type PipelineService struct {
	cfg    *config.Config
	engine *Engine // P5-2新增：并发执行引擎（由routes.Setup注入）
}

// NewPipelineService 创建Pipeline服务实例
func NewPipelineService(cfg *config.Config) *PipelineService {
	return &PipelineService{cfg: cfg}
}

// SetEngine 注入并发执行引擎（P5-2新增）
// 在routes.Setup中创建Engine后调用此方法注入
func (s *PipelineService) SetEngine(engine *Engine) {
	s.engine = engine
}

// GetEngine 获取并发执行引擎（P5-2新增）
// 供handler层查询引擎状态使用
func (s *PipelineService) GetEngine() *Engine {
	return s.engine
}

// ==================== Dashboard 统计（P4.5-D新增）====================

// GetDashboardStats 获取仪表盘统计数据
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
// P4.5-D新增：评估达标的Pipeline，跳过后续步骤直接标记为finalized
// 适用条件：Pipeline状态为review_queue/needs_human，或evaluator/meta已完成且达标
func (s *PipelineService) MarkPassed(pipelineID string) error {
	// 1. 获取Pipeline
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}

	// 2. 检查状态是否允许快捷通过
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue: true,
		models.PipelineStatusNeedsHuman:  true,
		models.PipelineStatusFailed:      true,
	}
	if !allowedStatuses[pipeline.Status] {
		return ErrMarkPassedNotAllowed
	}

	// 3. 检查meta步骤是否完成且达标
	metaStep, err := repository.GetStepByName(pipelineID, models.StepMeta)
	if err != nil || metaStep.Status != models.StepStatusDone {
		return ErrMarkPassedNotMet
	}

	// 从meta的step_data中提取total_final
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

	// 获取Pipeline配置中的阈值
	pCfg := models.ParsePipelineConfig(pipeline.Config)
	if totalFinal < pCfg.Threshold {
		return fmt.Errorf("%w (得分: %.1f, 阈值: %.1f)", ErrMarkPassedNotMet, totalFinal, pCfg.Threshold)
	}

	// 4. 标记review步骤完成（如果存在）
	reviewStep, err := repository.GetStepByName(pipelineID, models.StepReview)
	if err == nil && reviewStep.Status != models.StepStatusDone {
		_ = repository.StartStep(pipelineID, models.StepReview)
		statsJSON := fmt.Sprintf(`{"mark_passed":true,"meta_score":%.1f,"threshold":%.1f,"finalized_at":"%s"}`,
			totalFinal, pCfg.Threshold, time.Now().Format(time.RFC3339))
		_ = repository.CompleteStep(pipelineID, models.StepReview, 0, statsJSON, "", 0)
	}

	// 5. 更新Pipeline状态为finalized
	if err := repository.CompletePipeline(pipelineID, models.PipelineStatusFinalized); err != nil {
		return fmt.Errorf("快捷通过失败: %w", err)
	}

	return nil
}

// ==================== Pipeline 执行引擎 ====================

// StartPipeline 启动Pipeline执行
// P5-1改造：异步执行，立即返回running状态
// P5-2改造：通过Engine并发引擎提交任务，支持队列排队和AI限流
func (s *PipelineService) StartPipeline(id string) (*models.PipelineDetailResponse, error) {
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	if pipeline.Status != models.PipelineStatusPending {
		return nil, ErrPipelineNotPending
	}

	// 更新为running状态
	if err := repository.UpdatePipelineStatus(id, models.StepDbCheck, models.PipelineStatusRunning); err != nil {
		return nil, fmt.Errorf("更新Pipeline状态失败: %w", err)
	}

	// P5-2改造：通过Engine提交任务（如果Engine已注入）
	if s.engine != nil {
		task := &EngineTask{
			Type:       TaskTypePipeline,
			PipelineID: id,
			ExecFunc:   func() { s.executePipelineAsync(id) },
		}
		if !s.engine.Submit(task) {
			// 队列已满，回滚状态为pending
			_ = repository.UpdatePipelineStatus(id, models.StepDbCheck, models.PipelineStatusPending)
			return nil, ErrEngineQueueFull
		}
	} else {
		// 兼容模式：无Engine时直接goroutine执行（P5-1原行为）
		go s.executePipelineAsync(id)
	}

	// 返回当前状态（running）
	return s.GetPipelineDetail(id)
}

// executePipelineAsync 异步执行Pipeline全链路（goroutine/Engine Worker中运行）
// P5-1新增：将原StartPipeline中的同步执行逻辑抽取为独立方法
// P5-2更新：AI调用前后加信号量控制（AcquireAI/ReleaseAI）
// 注意：goroutine中的错误通过UpdatePipelineError记录到数据库，不会返回给HTTP调用者
func (s *PipelineService) executePipelineAsync(id string) {
	// 重新读取Pipeline（goroutine中需要独立读取，不依赖外部变量）
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		_ = repository.UpdatePipelineError(id, models.StepDbCheck, "异步执行: 读取Pipeline失败: "+err.Error())
		return
	}

	// P4-6修复：检测已完成的步骤，从第一个未完成的步骤开始执行
	existingSteps, stepsErr := repository.GetStepsByPipelineID(id)
	if stepsErr == nil {
		// 找到第一个非done的步骤
		var resumeStep string
		for _, st := range existingSteps {
			if st.Status != models.StepStatusDone {
				resumeStep = st.StepName
				break
			}
		}
		if resumeStep != "" && resumeStep != models.StepDbCheck {
			// 有已完成的步骤，从未完成的步骤恢复
			if err := repository.UpdatePipelineStatus(id, resumeStep, models.PipelineStatusRunning); err != nil {
				_ = repository.UpdatePipelineError(id, resumeStep, "恢复Pipeline失败: "+err.Error())
				return
			}
			pipeline, _ = repository.GetPipelineByID(id)

			switch resumeStep {
			case models.StepGenerator:
				genErr := s.executeGenerator(pipeline)
				if genErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepGenerator, genErr.Error())
					return
				}
				_ = repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue)
				return

			case models.StepReview:
				_ = repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue)
				return
			}
		}
	}

	// ===== 执行dbCheck =====
	dbCheckErr := s.executeDbCheck(pipeline)
	if dbCheckErr != nil {
		_ = repository.UpdatePipelineError(id, models.StepDbCheck, dbCheckErr.Error())
		s.broadcastStepUpdate(id, "pipeline_error", "dbCheck", "failed", "failed", dbCheckErr.Error())
		return
	}

	// 推进到scanner
	if err := repository.UpdatePipelineStatus(id, models.StepScanner, models.PipelineStatusRunning); err != nil {
		_ = repository.UpdatePipelineError(id, models.StepScanner, "推进到Scanner失败: "+err.Error())
		return
	}
	s.broadcastStepUpdate(id, "step_update", "scanner", "running", "running", "dbCheck完成，开始Scanner")

	// autoMode：自动执行后续步骤
	if pipeline.AutoMode {
		// ===== 执行scanner =====
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

		// scanner成功 -> 推进到evaluator
		if err := repository.UpdatePipelineStatus(id, models.StepEvaluator, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepEvaluator, "推进到Evaluator失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "evaluator", "running", "running", "Scanner完成，开始Evaluator")

		// ===== 执行evaluator =====
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

		// evaluator成功 -> 推进到meta
		if err := repository.UpdatePipelineStatus(id, models.StepMeta, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepMeta, "推进到Meta失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "meta", "running", "running", "Evaluator完成，开始Meta")

		// ===== 执行meta =====
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

		// meta成功 -> 推进到translator
		if err := repository.UpdatePipelineStatus(id, models.StepTranslator, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepTranslator, "推进到Translator失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "translator", "running", "running", "Meta完成，开始Translator")

		// ===== 执行translator =====
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

		// translator成功 -> 推进到generator
		if err := repository.UpdatePipelineStatus(id, models.StepGenerator, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepGenerator, "推进到Generator失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "generator", "running", "running", "Translator完成，开始Generator")

		// ===== 执行generator =====
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

		// generator成功 -> 推进到review（等待人工审核）
		if err := repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepReview, "推进到Review失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "pipeline_done", "review", "done", "review_queue", "Pipeline执行完成，等待审核")
	}
}

// ==================== AI调用包装方法（P5-2新增）====================

// callAIWithSemaphore 带信号量控制的AI调用包装方法
// P5-2新增：在AI调用前获取信号量，调用完毕后释放，防止并发超限
// 如果Engine未注入则直接调用（兼容模式）
func (s *PipelineService) callAIWithSemaphore(cfg *ai.EffectiveConfig, systemPrompt string, userPrompt string) (*ai.CallResult, error) {
	if s.engine != nil {
		s.engine.AcquireAI()
		defer s.engine.ReleaseAI()
	}
	return ai.CallAI(cfg, systemPrompt, userPrompt)
}

// ==================== P5-4 SSE事件广播辅助方法 ====================

// broadcastStepUpdate 广播步骤更新事件到SSE订阅者
// P5-4新增：每个步骤完成/失败/推进时调用，推送实时进度给前端
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


// ==================== dbCheck 步骤（P4-1）====================

// executeDbCheck 验证课程索引存在且有效
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

// doDbCheck 实际验证逻辑
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

// ==================== Scanner 步骤（P4-2）====================

// executeScanner 执行scanner步骤：调用Prompt A进行K12课程定位分析
// P5-2更新：AI调用改用callAIWithSemaphore（信号量控制）
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

// doScanner 调用AI执行课程定位分析
// P5-2更新：AI调用改用callAIWithSemaphore
func (s *PipelineService) doScanner(pipeline *models.Pipeline) (*models.ScannerResult, error) {
	result := &models.ScannerResult{}

	// 1. 加载 Prompt A
	promptA, err := repository.GetCurrentPromptByKey("prompt_a")
	if err != nil || promptA == nil {
		return nil, ErrScannerPromptMissing
	}

	// 2. 获取课程索引内容
	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		return nil, fmt.Errorf("scanner: 课程 %s 不存在", pipeline.CourseCode)
	}
	courseIndex, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		return nil, fmt.Errorf("scanner: 课程索引不存在，请先执行dbCheck")
	}

	// 3. 获取AI有效配置（三级回退：场景->全局->.env）
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

	// 4. 构造AI调用消息
	systemPrompt := promptA.Content
	userPrompt := fmt.Sprintf("请分析以下课程索引内容，按照要求输出JSON格式的定位信息：\n\n%s", courseIndex.IndexContent)

	// 5. 调用AI API（P5-2：带信号量控制）
	callResult, err := s.callAIWithSemaphore(aiCfg, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrScannerAIFailed, err.Error())
	}

	result.RawOutput = callResult.Content
	result.ModelUsed = callResult.ModelUsed
	result.TokensUsed = callResult.TokensUsed

	// 6. 从AI输出中提取JSON
	jsonStr, ok := ai.ExtractJSON(callResult.Content)
	if !ok {
		result.IsValid = false
		return result, fmt.Errorf("%w (AI输出前200字符: %s)",
			ErrScannerParseFailed, truncate(callResult.Content, 200))
	}

	// 验证JSON包含必要字段（target是必须的）
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

// ==================== Evaluator 步骤（P4-3）====================

// executeEvaluator 执行evaluator步骤：多轮独立评估（Prompt B * N轮）
// P5-2更新：每轮AI调用改用callAIWithSemaphore（信号量控制）
func (s *PipelineService) executeEvaluator(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepEvaluator

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动evaluator失败: %w", err)
	}

	// 解析Pipeline配置获取轮数
	pCfg := models.ParsePipelineConfig(pipeline.Config)
	totalRounds := pCfg.EvalRounds

	// 1. 加载所需提示词
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

	// ability_table可选（有则加入）
	abilityTable, _ := repository.GetCurrentPromptByKey("ability_table")

	// 2. 获取课程索引
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

	// 3. 获取Scanner步骤的parsed结果（课程定位JSON）
	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrEvalScannerNotDone.Error())
		return ErrEvalScannerNotDone
	}
	scannerLocationJSON := extractScannerParsed(scannerStep.StepData)

	// 4. 获取AI配置（使用evaluator场景）
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

	// 5. 构造系统提示词和用户消息
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

	// 6. 清理旧的评估轮次数据（支持重跑）
	_ = repository.DeleteEvalRoundsByPipelineID(pipeline.ID)

	// 7. 逐轮执行评估
	evalResult := &models.EvaluatorResult{
		TotalRounds: totalRounds,
	}
	var roundScores []float64
	var totalTokens int
	var doneCount, failCount int
	var lastModelUsed string

	for i := 1; i <= totalRounds; i++ {
		// 创建轮次记录
		roundRec, err := repository.CreateEvalRound(pipeline.ID, i)
		if err != nil {
			failCount++
			continue
		}

		// 标记为运行中
		_ = repository.UpdateEvalRoundRunning(roundRec.ID)

		// 调用AI（P5-2：带信号量控制）
		callResult, callErr := s.callAIWithSemaphore(aiCfg, systemPrompt, userPrompt)
		if callErr != nil {
			_ = repository.FailEvalRound(roundRec.ID, "", callErr.Error())
			failCount++
			continue
		}

		output := callResult.Content
		lastModelUsed = callResult.ModelUsed
		totalTokens += callResult.TokensUsed

		// 提取评分
		scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4, hardConstraint, grade, parseOk := extractEvalScores(output)

		if !parseOk || scoreTotal < 0 {
			_ = repository.FailEvalRound(roundRec.ID, truncate(output, 5000), "评分提取失败")
			failCount++
			continue
		}

		// 构造dimensions JSON
		dimMap := map[string]interface{}{
			"hard_constraint": hardConstraint,
			"grade":           grade,
		}
		dimJSON, _ := json.Marshal(dimMap)

		// 保存轮次结果
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

	// 8. 汇总结果
	if doneCount == 0 {
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrEvalAllRoundsFailed.Error())
		return ErrEvalAllRoundsFailed
	}

	// 计算均值
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

	// 计算方差
	if doneCount >= 2 {
		var sumSqDiff float64
		for _, sc := range roundScores {
			diff := sc - evalResult.AvgTotal
			sumSqDiff += diff * diff
		}
		evalResult.Variance = math.Round(sumSqDiff/n*100) / 100
		evalResult.VarianceWarn = evalResult.Variance > pCfg.VarianceWarn
	}

	// 9. 保存evaluator步骤汇总数据
	if err := repository.CompleteStep(
		pipeline.ID, stepName, durationMs,
		evalResult.ToJSON(), lastModelUsed, totalTokens,
	); err != nil {
		return fmt.Errorf("保存evaluator结果失败: %w", err)
	}

	return nil
}

// ==================== Meta 步骤（P4-4新增）====================

// executeMeta 执行meta步骤：元评估仲裁（Prompt E）
// P5-2更新：AI调用改用callAIWithSemaphore（信号量控制）
func (s *PipelineService) executeMeta(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepMeta

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动meta失败: %w", err)
	}

	// 解析Pipeline配置
	pCfg := models.ParsePipelineConfig(pipeline.Config)
	threshold := pCfg.Threshold
	maxRetry := pCfg.MaxMetaRetry

	// 1. 加载 Prompt E
	promptE, err := repository.GetCurrentPromptByKey("prompt_e")
	if err != nil || promptE == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaPromptMissing.Error())
		return ErrMetaPromptMissing
	}

	// 2. 加载解压缩字典
	dict, err := repository.GetCurrentPromptByKey("dict")
	if err != nil || dict == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaDictMissing.Error())
		return ErrMetaDictMissing
	}

	// 3. 获取课程索引
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

	// 4. 获取Scanner步骤的parsed结果
	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaScannerNotDone.Error())
		return ErrMetaScannerNotDone
	}
	scannerLocationJSON := extractScannerParsed(scannerStep.StepData)

	// 5. 获取Evaluator步骤的N轮评估报告
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

	// 6. 组装N轮评估报告文本
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

	// 7. 获取AI配置（使用meta场景）
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

	// 8. 构造系统提示词和用户消息
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

	// 9. 重试循环：最多maxRetry次
	metaResult := &models.MetaResult{
		TotalRetries: maxRetry,
	}
	var lastOutput string
	var lastModelUsed string
	var totalTokens int
	var totalLatencyMs int64

	for attempt := 1; attempt <= maxRetry; attempt++ {
		metaResult.Attempt = attempt

		// 调用AI（P5-2：带信号量控制）
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

		// 提取META_SCORE评分
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

		// 填充MetaResult评分字段
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

		// 判断是否通过阈值
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

		// 未通过，如果还有重试次数则继续
		if attempt == maxRetry {
			durationMs := time.Since(startTime).Milliseconds()
			s.saveStepData(pipeline.ID, stepName, metaResult.ToJSON())
			errMsg := fmt.Sprintf("%s (最终得分: %.1f, 阈值: %.1f, 共%d次尝试)",
				ErrMetaAllRetriesFailed.Error(), metaResult.TotalFinal, threshold, maxRetry)
			_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
			return fmt.Errorf(errMsg)
		}
	}

	// 理论上不会走到这里
	durationMs := time.Since(startTime).Milliseconds()
	_ = repository.FailStep(pipeline.ID, stepName, durationMs, "meta: 异常退出重试循环")
	return fmt.Errorf("meta: 异常退出重试循环")
}

// ==================== Meta 评分提取（P4-4新增）====================

// metaScoreResult Meta评分提取结果（内部使用）
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

// extractMetaScores 从AI Meta输出中提取<<<META_SCORE>>>块中的评分
func extractMetaScores(output string) *metaScoreResult {
	result := &metaScoreResult{}

	// 主解析：<<<META_SCORE>>>...<<<END_META_SCORE>>>
	metaBlockRe := regexp.MustCompile(`(?s)<<<META_SCORE>>>(.*?)<<<END_META_SCORE>>>`)
	blockMatch := metaBlockRe.FindStringSubmatch(output)

	if len(blockMatch) < 2 {
		totalFallbackRe := regexp.MustCompile(`(?i)TOTAL_FINAL:\s*([\d.]+)`)
		tfm := totalFallbackRe.FindStringSubmatch(output)
		if tfm != nil {
			result.totalFinal = safeParseFloat(tfm[1])
			if result.totalFinal > 0 {
				result.parseOk = true
			}
		}
		return result
	}

	block := blockMatch[1]

	// 提取 TOTAL_FINAL
	totalRe := regexp.MustCompile(`(?i)TOTAL_FINAL:\s*([\d.]+)`)
	tm := totalRe.FindStringSubmatch(block)
	if tm == nil {
		return result
	}
	result.totalFinal = safeParseFloat(tm[1])

	// 提取 E1_FINAL ~ E4_FINAL
	e1FinalRe := regexp.MustCompile(`(?i)E1_FINAL:\s*([\d.]+)`)
	e2FinalRe := regexp.MustCompile(`(?i)E2_FINAL:\s*([\d.]+)`)
	e3FinalRe := regexp.MustCompile(`(?i)E3_FINAL:\s*([\d.]+)`)
	e4FinalRe := regexp.MustCompile(`(?i)E4_FINAL:\s*([\d.]+)`)

	if m := e1FinalRe.FindStringSubmatch(block); m != nil {
		result.e1Final = safeParseFloat(m[1])
	}
	if m := e2FinalRe.FindStringSubmatch(block); m != nil {
		result.e2Final = safeParseFloat(m[1])
	}
	if m := e3FinalRe.FindStringSubmatch(block); m != nil {
		result.e3Final = safeParseFloat(m[1])
	}
	if m := e4FinalRe.FindStringSubmatch(block); m != nil {
		result.e4Final = safeParseFloat(m[1])
	}

	// 提取 HARD_CONSTRAINT
	hardRe := regexp.MustCompile(`(?i)HARD_CONSTRAINT:\s*(PASS|FAIL)`)
	if hm := hardRe.FindStringSubmatch(block); hm != nil {
		result.hardConstraint = hm[1]
	}

	// 提取 GRADE
	gradeRe := regexp.MustCompile(`(?i)GRADE:\s*([A-D])`)
	if gm := gradeRe.FindStringSubmatch(block); gm != nil {
		result.grade = gm[1]
	}

	// 提取各轮分数
	roundRe := regexp.MustCompile(`(?i)E([1-4])_R(\d+):\s*([\d.]+)`)
	allRoundMatches := roundRe.FindAllStringSubmatch(block, -1)

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

	// P4.5-D修复：从Meta完整输出末尾提取综合评分Y值（修改后预期分）
	finalScoreRe := regexp.MustCompile(`(?:综合评分|综合)[：:]\s*[\d.]+\s*→\s*([\d.]+)\s*/\s*10`)
	if fsm := finalScoreRe.FindStringSubmatch(output); fsm != nil {
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
func (s *PipelineService) UpdatePageDecision(pipelineID string, pageNumber int, decision string, finalHTML *string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusReviewQueue && pipeline.Status != models.PipelineStatusNeedsHuman {
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

// FinalizePipeline 定稿归档Pipeline
func (s *PipelineService) FinalizePipeline(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusReviewQueue && pipeline.Status != models.PipelineStatusNeedsHuman {
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

// extractScannerParsed 从scanner的step_data JSON中提取parsed字段
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

// extractEvalScores 从AI评估输出中提取SCORE_BLOCK评分
func extractEvalScores(output string) (float64, float64, float64, float64, float64, string, string, bool) {
	scoreBlockRe := regexp.MustCompile(`(?s)<<<SCORE_BLOCK>>>(.*?)<<<END_SCORE_BLOCK>>>`)
	sbMatch := scoreBlockRe.FindStringSubmatch(output)

	if len(sbMatch) >= 2 {
		block := sbMatch[1]

		totalRe := regexp.MustCompile(`(?i)TOTAL[:\s]+([\d.]+)`)
		e1Re := regexp.MustCompile(`(?i)E1[:\s]+([\d.]+)`)
		e2Re := regexp.MustCompile(`(?i)E2[:\s]+([\d.]+)`)
		e3Re := regexp.MustCompile(`(?i)E3[:\s]+([\d.]+)`)
		e4Re := regexp.MustCompile(`(?i)E4[:\s]+([\d.]+)`)
		hardRe := regexp.MustCompile(`(?i)HARD_CONSTRAINT[:\s]+(PASS|FAIL)`)
		gradeRe := regexp.MustCompile(`(?i)GRADE[:\s]+([A-D])`)

		tm := totalRe.FindStringSubmatch(block)
		e1m := e1Re.FindStringSubmatch(block)
		e2m := e2Re.FindStringSubmatch(block)
		e3m := e3Re.FindStringSubmatch(block)
		e4m := e4Re.FindStringSubmatch(block)
		hm := hardRe.FindStringSubmatch(block)
		gm := gradeRe.FindStringSubmatch(block)

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

	// 备用：尝试从正文中提取维度评分
	var scoreE1, scoreE2, scoreE3, scoreE4 float64
	dimCount := 0
	dimPatterns := []struct {
		re    *regexp.Regexp
		field *float64
	}{
		{regexp.MustCompile(`(?i)E1[^\n]{0,30}[：:]\s*([\d.]+)\s*/\s*10`), &scoreE1},
		{regexp.MustCompile(`(?i)E2[^\n]{0,30}[：:]\s*([\d.]+)\s*/\s*10`), &scoreE2},
		{regexp.MustCompile(`(?i)E3[^\n]{0,30}[：:]\s*([\d.]+)\s*/\s*10`), &scoreE3},
		{regexp.MustCompile(`(?i)E4[^\n]{0,30}[：:]\s*([\d.]+)\s*/\s*10`), &scoreE4},
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

// saveStepData 单独更新步骤的step_data字段
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

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// getMapKeys 获取map的key列表
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// safeParseFloat 安全解析浮点数，失败返回0
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

// getSubmatch 安全获取正则子匹配的第一个捕获组
func getSubmatch(m []string) string {
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// ==================== Eval Rounds 查询（P4.5-B新增）====================

// EvalRoundDetail 评估轮次详情（返回给前端）
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

// GetEvalRounds 获取Pipeline的评估轮次详情列表
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

// AIFixPage 审核员在全屏预览中发现生成HTML有问题时，输入修改指令让AI修复
// P5-2更新：AI调用改用callAIWithSemaphore（信号量控制）
func (s *PipelineService) AIFixPage(pipelineID string, pageNumber int, fixInstruction string) (string, error) {
	// 1. 验证Pipeline存在且状态允许审核操作
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return "", ErrPipelineNotFound
	}
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue: true,
		models.PipelineStatusNeedsHuman:  true,
		models.PipelineStatusFinalized:   true,
	}
	if !allowedStatuses[pipeline.Status] {
		return "", ErrPipelineNotReviewable
	}

	// 2. 获取当前页面的完整HTML
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

	// 3. 加载 Prompt F
	promptF, err := repository.GetCurrentPromptByKey("prompt_f")
	if err != nil || promptF == nil {
		return "", fmt.Errorf("Prompt F未配置，无法执行AI快修")
	}

	// 4. 获取AI配置
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

	// 5. 构建用户消息
	userPrompt := "【\u26a0\ufe0f\u26a0\ufe0f\u26a0\ufe0f 最重要 \u2014 当前页面HTML，你必须在此基础上修复，禁止重写】\n" +
		"以下是当前页面的完整HTML。你的输出必须以此为基础，只修改下方修复指令要求的部分，其余代码原封不动保留。\n\n" +
		currentHTML + "\n\n" +
		"\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\n" +
		"\u25b2 以上是当前HTML（必须作为修改基础） \u25bc 以下是修复指令\n" +
		"\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\n\n" +
		fmt.Sprintf("【页面】P%02d \u2014 %s\n", pageNumber, currentPage.PageTitle) +
		"【操作类型】AI快修（在当前HTML基础上修复指定问题）\n\n" +
		"【审核员修复指令（必须严格执行）】\n" +
		fixInstruction + "\n\n" +
		"【\u26a0\ufe0f 最终提醒】你的输出必须与上方HTML有90%以上代码重合。只改指令要求的部分。导航栏、视频、图片不允许任何改动。输出完整HTML。"

	// 6. 调用AI（P5-2：带信号量控制）
	callResult, callErr := s.callAIWithSemaphore(aiCfg, promptF.Content, userPrompt)
	if callErr != nil {
		return "", fmt.Errorf("%w: %s", ErrAIFixFailed, callErr.Error())
	}

	// 7. 提取生成的HTML
	newHTML := extractGeneratedHTML(callResult.Content)
	if len(newHTML) < 100 {
		return "", fmt.Errorf("%w: AI输出HTML过短(%d字符)", ErrAIFixFailed, len(newHTML))
	}

	// 8. 更新数据库
	if err := repository.UpdateGeneratedPageHTML(pipelineID, pageNumber, newHTML, newHTML); err != nil {
		return "", fmt.Errorf("保存修复后HTML失败: %w", err)
	}

	return newHTML, nil
}

// ==================== P5-3 批量创建+批量启动 ====================

// BatchCreateResult 批量创建结果
type BatchCreateResult struct {
	TotalRequested int      `json:"total_requested"` // 请求创建的课程数量
	CreatedIDs     []string `json:"created_ids"`     // 成功创建的Pipeline ID列表
	SkippedCodes   []string `json:"skipped_codes"`   // 跳过的课程编号（已有活跃Pipeline或课程不存在）
	SkippedReasons []string `json:"skipped_reasons"` // 跳过原因详情
	FailedCodes    []string `json:"failed_codes"`    // 创建失败的课程编号
	FailedReasons  []string `json:"failed_reasons"`  // 失败原因详情
}

// BatchCreatePipelines 批量创建Pipeline
// P5-3新增：从课程编号列表批量创建Pipeline，跳过已有活跃Pipeline的课程
func (s *PipelineService) BatchCreatePipelines(courseCodes []string, userID string) (*BatchCreateResult, error) {
	result := &BatchCreateResult{
		TotalRequested: len(courseCodes),
		CreatedIDs:     []string{},
		SkippedCodes:   []string{},
		SkippedReasons: []string{},
		FailedCodes:    []string{},
		FailedReasons:  []string{},
	}

	// 去重
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
			// 区分"跳过"和"失败"
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

// BatchStartResult 批量启动结果
type BatchStartResult struct {
	TotalRequested int      `json:"total_requested"` // 请求启动的Pipeline数量
	StartedIDs     []string `json:"started_ids"`     // 成功提交到引擎的Pipeline ID列表
	SkippedIDs     []string `json:"skipped_ids"`     // 跳过的Pipeline ID（非pending状态）
	SkippedReasons []string `json:"skipped_reasons"` // 跳过原因详情
	FailedIDs      []string `json:"failed_ids"`      // 启动失败的Pipeline ID
	FailedReasons  []string `json:"failed_reasons"`  // 失败原因详情
}

// BatchStartPipelines 批量启动Pipeline
// P5-3新增：批量启动多个pending状态的Pipeline，逐个通过Engine.Submit提交
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
