package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

		// scanner成功 → 推进到evaluator（P4-3实现时继续执行）
		if err := repository.UpdatePipelineStatus(id, models.StepEvaluator, models.PipelineStatusRunning); err != nil {
			return nil, fmt.Errorf("推进到Evaluator失败: %w", err)
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
	// Prompt A作为系统提示词（定义AI角色和JSON输出格式要求）
	// 课程索引作为用户消息（待分析内容）
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
