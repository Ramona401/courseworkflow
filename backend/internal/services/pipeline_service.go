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
)

// ==================== PipelineService ====================

// PipelineService Pipeline业务逻辑层
type PipelineService struct {
	cfg *config.Config
}

// NewPipelineService 创建Pipeline服务实例
func NewPipelineService(cfg *config.Config) *PipelineService {
	return &PipelineService{cfg: cfg}
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
	// 允许的状态：review_queue / needs_human / running（但meta已完成）/ failed（但meta已完成）
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
// P4-1：执行dbCheck → 推进到scanner
// P4-2：autoMode时继续执行scanner → 推进到evaluator
// P4-3：autoMode时继续执行evaluator → 推进到meta
// P4-4：autoMode时继续执行meta → 通过则推进到translator
// P4-5：autoMode时继续执行translator（Prompt C+D循环） → 通过则推进到generator
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

	// P4-6修复：检测已完成的步骤，从第一个未完成的步骤开始执行
	// 这样重跑Pipeline时不会重复执行已done的步骤
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
				return nil, fmt.Errorf("恢复Pipeline到%s失败: %w", resumeStep, err)
			}
			pipeline, _ = repository.GetPipelineByID(id)

			switch resumeStep {
			case models.StepGenerator:
				genErr := s.executeGenerator(pipeline)
				if genErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepGenerator, genErr.Error())
					return s.GetPipelineDetail(id)
				}
				_ = repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue)
				return s.GetPipelineDetail(id)

			case models.StepReview:
				// review是人工步骤，标记为等待审核
				_ = repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue)
				return s.GetPipelineDetail(id)
			}
			// 其他步骤（scanner/evaluator/meta/translator）走正常全链路
		}
	}

	// 执行dbCheck
	dbCheckErr := s.executeDbCheck(pipeline)
	if dbCheckErr != nil {
		_ = repository.UpdatePipelineError(id, models.StepDbCheck, dbCheckErr.Error())
		return s.GetPipelineDetail(id)
	}

	// 推进到scanner
	if err := repository.UpdatePipelineStatus(id, models.StepScanner, models.PipelineStatusRunning); err != nil {
		return nil, fmt.Errorf("推进到Scanner失败: %w", err)
	}

	// autoMode：自动执行scanner
	if pipeline.AutoMode {
		// 重新读取最新Pipeline（current_step已更新）
		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			return s.GetPipelineDetail(id)
		}

		scannerErr := s.executeScanner(pipeline)
		if scannerErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepScanner, scannerErr.Error())
			return s.GetPipelineDetail(id)
		}

		// scanner成功 → 推进到evaluator
		if err := repository.UpdatePipelineStatus(id, models.StepEvaluator, models.PipelineStatusRunning); err != nil {
			return nil, fmt.Errorf("推进到Evaluator失败: %w", err)
		}

		// P4-3：autoMode时继续执行evaluator
		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			return s.GetPipelineDetail(id)
		}

		evalErr := s.executeEvaluator(pipeline)
		if evalErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepEvaluator, evalErr.Error())
			return s.GetPipelineDetail(id)
		}

		// evaluator成功 → 推进到meta
		if err := repository.UpdatePipelineStatus(id, models.StepMeta, models.PipelineStatusRunning); err != nil {
			return nil, fmt.Errorf("推进到Meta失败: %w", err)
		}

		// P4-4：autoMode时继续执行meta
		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			return s.GetPipelineDetail(id)
		}

		metaErr := s.executeMeta(pipeline)
		if metaErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepMeta, metaErr.Error())
			return s.GetPipelineDetail(id)
		}

		// meta成功（通过阈值） → 推进到translator
		if err := repository.UpdatePipelineStatus(id, models.StepTranslator, models.PipelineStatusRunning); err != nil {
			return nil, fmt.Errorf("推进到Translator失败: %w", err)
		}

		// P4-5：autoMode时继续执行translator（Prompt C + D循环）
		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			return s.GetPipelineDetail(id)
		}

		transErr := s.executeTranslator(pipeline)
		if transErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepTranslator, transErr.Error())
			return s.GetPipelineDetail(id)
		}

		// translator成功 → 推进到generator（P4-6实现时继续执行）
		if err := repository.UpdatePipelineStatus(id, models.StepGenerator, models.PipelineStatusRunning); err != nil {
			return nil, fmt.Errorf("推进到Generator失败: %w", err)
		}

		// P4-6：autoMode时继续执行generator（Prompt F × 每页）
		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			return s.GetPipelineDetail(id)
		}

		genErr := s.executeGenerator(pipeline)
		if genErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepGenerator, genErr.Error())
			return s.GetPipelineDetail(id)
		}

		// generator成功 → 推进到review（等待人工审核）
		if err := repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue); err != nil {
			return nil, fmt.Errorf("推进到Review失败: %w", err)
		}
	}

	return s.GetPipelineDetail(id)
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
func (s *PipelineService) doScanner(pipeline *models.Pipeline) (*models.ScannerResult, error) {
	result := &models.ScannerResult{}

	// 1. 加载 Prompt A（使用正确的函数名 GetCurrentPromptByKey）
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

	// 3. 获取AI有效配置（三级回退：场景→全局→.env）
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

	// 5. 调用AI API
	callResult, err := ai.CallAI(aiCfg, systemPrompt, userPrompt)
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

// executeEvaluator 执行evaluator步骤：多轮独立评估（Prompt B × N轮）
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
	// 从scanner的step_data中提取parsed字段作为课程定位
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

	// 用户消息：课程定位 + 待评估索引 + 解压缩字典
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
	// 如果有能力定位表，追加到用户消息
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

		// 调用AI
		callResult, callErr := ai.CallAI(aiCfg, systemPrompt, userPrompt)
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
			// 评分提取失败，标记轮次失败但保存输出
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
// 综合N轮评估报告，输出仲裁分数+修改方案+修改后完整索引
// 评分≥threshold(9.0)通过 | <threshold重试(最多max_meta_retry次)
func (s *PipelineService) executeMeta(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepMeta

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动meta失败: %w", err)
	}

	// 解析Pipeline配置
	pCfg := models.ParsePipelineConfig(pipeline.Config)
	threshold := pCfg.Threshold   // 默认9.0
	maxRetry := pCfg.MaxMetaRetry // 默认3

	// 1. 加载 Prompt E
	promptE, err := repository.GetCurrentPromptByKey("prompt_e")
	if err != nil || promptE == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaPromptMissing.Error())
		return ErrMetaPromptMissing
	}

	// 2. 加载解压缩字典（Meta也需要dict）
	dict, err := repository.GetCurrentPromptByKey("dict")
	if err != nil || dict == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaDictMissing.Error())
		return ErrMetaDictMissing
	}

	// 3. 获取课程索引（原始索引，作为Meta的输入）
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

	// 4. 获取Scanner步骤的parsed结果（课程定位JSON）
	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaScannerNotDone.Error())
		return ErrMetaScannerNotDone
	}
	scannerLocationJSON := extractScannerParsed(scannerStep.StepData)

	// 5. 获取Evaluator步骤的N轮评估报告（eval_rounds表中的output字段）
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

	// 9. 重试循环：最多maxRetry次，每次调用AI并提取评分，达标即通过
	metaResult := &models.MetaResult{
		TotalRetries: maxRetry,
	}
	var lastOutput string
	var lastModelUsed string
	var totalTokens int
	var totalLatencyMs int64

	for attempt := 1; attempt <= maxRetry; attempt++ {
		metaResult.Attempt = attempt

		// 调用AI
		callResult, callErr := ai.CallAI(aiCfg, systemPrompt, userPrompt)
		if callErr != nil {
			// AI调用失败，继续重试
			totalLatencyMs += time.Since(startTime).Milliseconds() - totalLatencyMs
			if attempt == maxRetry {
				// 最后一次也失败
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
			// 评分提取失败，继续重试
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
			// 通过！保存结果并退出重试循环
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
			// 所有重试用完，仍未达标
			durationMs := time.Since(startTime).Milliseconds()
			s.saveStepData(pipeline.ID, stepName, metaResult.ToJSON())
			errMsg := fmt.Sprintf("%s (最终得分: %.1f, 阈值: %.1f, 共%d次尝试)",
				ErrMetaAllRetriesFailed.Error(), metaResult.TotalFinal, threshold, maxRetry)
			_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
			return fmt.Errorf(errMsg)
		}
	}

	// 理论上不会走到这里，但保险起见
	durationMs := time.Since(startTime).Milliseconds()
	_ = repository.FailStep(pipeline.ID, stepName, durationMs, "meta: 异常退出重试循环")
	return fmt.Errorf("meta: 异常退出重试循环")
}

// ==================== Meta 评分提取（P4-4新增）====================

// metaScoreResult Meta评分提取结果（内部使用）
type metaScoreResult struct {
	totalFinal     float64   // TOTAL_FINAL
	e1Final        float64   // E1_FINAL
	e2Final        float64   // E2_FINAL
	e3Final        float64   // E3_FINAL
	e4Final        float64   // E4_FINAL
	hardConstraint string    // HARD_CONSTRAINT: PASS/FAIL
	grade          string    // GRADE: A/B/C/D
	e1Rounds       []float64 // 各轮E1分数 [R1,R2,...RN]
	e2Rounds       []float64 // 各轮E2分数
	e3Rounds       []float64 // 各轮E3分数
	e4Rounds       []float64 // 各轮E4分数
	parseOk        bool      // 是否成功提取
}

// extractMetaScores 从AI Meta输出中提取<<<META_SCORE>>>块中的评分
func extractMetaScores(output string) *metaScoreResult {
	result := &metaScoreResult{}

	// 主解析：<<<META_SCORE>>>...<<<END_META_SCORE>>>
	metaBlockRe := regexp.MustCompile(`(?s)<<<META_SCORE>>>(.*?)<<<END_META_SCORE>>>`)
	blockMatch := metaBlockRe.FindStringSubmatch(output)

	if len(blockMatch) < 2 {
		// 未找到META_SCORE块，尝试备用提取TOTAL_FINAL
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
		return result // 没有TOTAL_FINAL视为解析失败
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

	// 提取各轮分数：E1_R1:{值} E1_R2:{值} ...
	roundRe := regexp.MustCompile(`(?i)E([1-4])_R(\d+):\s*([\d.]+)`)
	allRoundMatches := roundRe.FindAllStringSubmatch(block, -1)

	// 用map暂存：dim -> roundNum -> score
	roundMap := map[int]map[int]float64{
		1: {}, 2: {}, 3: {}, 4: {},
	}
	maxRound := 0
	for _, m := range allRoundMatches {
		dim, _ := strconv.Atoi(m[1]) // 1~4
		rn, _ := strconv.Atoi(m[2])  // 轮次序号
		score := safeParseFloat(m[3])
		if dim >= 1 && dim <= 4 && rn >= 1 {
			roundMap[dim][rn] = score
			if rn > maxRound {
				maxRound = rn
			}
		}
	}

	// 转换为有序数组
	for rn := 1; rn <= maxRound; rn++ {
		result.e1Rounds = append(result.e1Rounds, roundMap[1][rn])
		result.e2Rounds = append(result.e2Rounds, roundMap[2][rn])
		result.e3Rounds = append(result.e3Rounds, roundMap[3][rn])
		result.e4Rounds = append(result.e4Rounds, roundMap[4][rn])
	}

	// P4.5-D修复：从Meta完整输出末尾提取「综合评分：X→Y/10」中的Y值（修改后预期分）
	// META_SCORE块中的TOTAL_FINAL是原始索引的仲裁分，末尾的综合评分才是修改后的预期分
	// 格式示例：综合评分：8.3→9.9/10
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
// P4.5-C新增：审核页面需要完整HTML用于预览和对比
func (s *PipelineService) GetGeneratedPages(pipelineID string) ([]*repository.GeneratedPageFullRow, error) {
	// 验证Pipeline存在
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
// P4.5-C新增：支持approve（采用AI生成）/reject（保留原版）/edit（使用编辑后的HTML）
func (s *PipelineService) UpdatePageDecision(pipelineID string, pageNumber int, decision string, finalHTML *string) error {
	// 验证Pipeline存在且状态允许审核
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	// 允许在review_queue或needs_human状态下进行审核操作
	if pipeline.Status != models.PipelineStatusReviewQueue && pipeline.Status != models.PipelineStatusNeedsHuman {
		return ErrPipelineNotReviewable
	}

	// 验证决策值有效性
	validDecisions := map[string]bool{"approve": true, "reject": true, "edit": true}
	if !validDecisions[decision] {
		return ErrInvalidDecision
	}

	// edit决策必须提供finalHTML
	if decision == "edit" && (finalHTML == nil || *finalHTML == "") {
		return fmt.Errorf("edit决策必须提供修改后的HTML内容")
	}

	// 执行数据库更新
	if err := repository.UpdatePageDecision(pipelineID, pageNumber, decision, finalHTML); err != nil {
		return err
	}

	return nil
}

// FinalizePipeline 定稿归档Pipeline
// P4.5-C新增：检查所有页面都已决策后，将Pipeline标记为finalized
func (s *PipelineService) FinalizePipeline(pipelineID string) error {
	// 验证Pipeline存在且状态允许定稿
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusReviewQueue && pipeline.Status != models.PipelineStatusNeedsHuman {
		return ErrPipelineNotReviewable
	}

	// 检查是否所有页面都已决策
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

	// 标记review步骤完成
	reviewStep, err := repository.GetStepByName(pipelineID, models.StepReview)
	if err == nil && reviewStep.Status != models.StepStatusDone {
		_ = repository.StartStep(pipelineID, models.StepReview)
		// review步骤的step_data记录定稿统计
		statsJSON := fmt.Sprintf(`{"total_pages":%d,"decided_pages":%d,"finalized_at":"%s"}`,
			total, decided, time.Now().Format(time.RFC3339))
		_ = repository.CompleteStep(pipelineID, models.StepReview, 0, statsJSON, "", 0)
	}

	// 更新Pipeline状态为finalized
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
// 返回：scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4, hardConstraint, grade, parseOk
func extractEvalScores(output string) (float64, float64, float64, float64, float64, string, string, bool) {
	// 主要格式：<<<SCORE_BLOCK>>>...<<<END_SCORE_BLOCK>>>
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

	// 提取失败
	return -1, 0, 0, 0, 0, "", "", false
}

// ==================== 工具方法 ====================

// saveStepData 单独更新步骤的step_data字段（用于失败后保存诊断数据）
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

// getMapKeys 获取map的key列表（供错误诊断）
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
		// 从dimensions JSON提取hard_constraint和grade
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
// 流程：读取当前页generated_html + prompt_f + fix_instruction → 调AI → 返回新HTML → 更新数据库
func (s *PipelineService) AIFixPage(pipelineID string, pageNumber int, fixInstruction string) (string, error) {
	// 1. 验证Pipeline存在且状态允许审核操作
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return "", ErrPipelineNotFound
	}
	// 允许在 review_queue / needs_human / finalized 状态下使用AI快修
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

	// 取当前最佳HTML：优先final_html，其次generated_html，最后original_html
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

	// 3. 加载 Prompt F（作为系统提示词）
	promptF, err := repository.GetCurrentPromptByKey("prompt_f")
	if err != nil || promptF == nil {
		return "", fmt.Errorf("Prompt F未配置，无法执行AI快修")
	}

	// 4. 获取AI配置（使用generator场景）
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

	// 5. 构建用户消息：原始HTML + 修复指令
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

	// 6. 调用AI
	callResult, callErr := ai.CallAI(aiCfg, promptF.Content, userPrompt)
	if callErr != nil {
		return "", fmt.Errorf("%w: %s", ErrAIFixFailed, callErr.Error())
	}

	// 7. 提取生成的HTML
	newHTML := extractGeneratedHTML(callResult.Content)
	if len(newHTML) < 100 {
		return "", fmt.Errorf("%w: AI输出HTML过短(%d字符)", ErrAIFixFailed, len(newHTML))
	}

	// 8. 更新数据库：同时更新generated_html和final_html
	if err := repository.UpdateGeneratedPageHTML(pipelineID, pageNumber, newHTML, newHTML); err != nil {
		return "", fmt.Errorf("保存修复后HTML失败: %w", err)
	}

	return newHTML, nil
}
