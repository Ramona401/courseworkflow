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

// ==================== Pipeline 执行引擎 ====================

// StartPipeline 启动Pipeline执行
// P4-1：执行dbCheck → 推进到scanner
// P4-2：autoMode时继续执行scanner → 推进到evaluator
// P4-3：autoMode时继续执行evaluator → 推进到meta
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

		// evaluator成功 → 推进到meta（P4-4实现时继续执行）
		if err := repository.UpdatePipelineStatus(id, models.StepMeta, models.PipelineStatusRunning); err != nil {
			return nil, fmt.Errorf("推进到Meta失败: %w", err)
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

// ==================== Evaluator 步骤（P4-3新增）====================

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

// extractScannerParsed 从scanner的step_data JSON中提取parsed字段
// 返回课程定位JSON字符串（供evaluator的用户消息使用）
func extractScannerParsed(stepData string) string {
	if stepData == "" || stepData == "null" {
		return "（无课程定位数据）"
	}
	// step_data格式：{"raw_output":"...","parsed":{...},"is_valid":true,...}
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stepData), &data); err != nil {
		return "（课程定位数据解析失败）"
	}
	parsed, ok := data["parsed"]
	if !ok || string(parsed) == "null" {
		return "（无课程定位数据）"
	}
	// 格式化输出（美化JSON）
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
		// 至少提取到3个维度分，计算综合分
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
