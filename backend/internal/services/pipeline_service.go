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
// 负责Pipeline的创建、启动、步骤执行调度
type PipelineService struct {
	cfg *config.Config // 全局配置
}

// NewPipelineService 创建Pipeline服务实例
func NewPipelineService(cfg *config.Config) *PipelineService {
	return &PipelineService{cfg: cfg}
}

// ==================== Pipeline CRUD 方法 ====================

// CreatePipeline 创建新Pipeline
// 验证课程存在且无运行中Pipeline后，创建Pipeline+7个步骤记录
func (s *PipelineService) CreatePipeline(req *models.CreatePipelineRequest, userID string) (*models.PipelineDetailResponse, error) {
	// 1. 参数校验
	courseCode := strings.TrimSpace(req.CourseCode)
	if courseCode == "" {
		return nil, ErrPipelineCourseRequired
	}

	// 2. 验证课程存在
	course, err := repository.GetCourseByCode(courseCode)
	if err != nil {
		return nil, ErrPipelineCourseNotFound
	}

	// 3. 检查是否已有运行中的Pipeline（同一课程不允许并行）
	exists, existingID, err := repository.CheckActivePipelineExists(courseCode)
	if err != nil {
		return nil, fmt.Errorf("检查Pipeline状态失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%w (ID: %s)", ErrPipelineAlreadyExists, existingID)
	}

	// 4. 构造Pipeline配置
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

	// 5. 确定是否自动模式（默认true）
	autoMode := true
	if req.AutoMode != nil {
		autoMode = *req.AutoMode
	}

	// 6. 创建Pipeline主记录
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

	// 7. 查询完整详情返回
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
	// 1. 获取Pipeline主记录
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	// 2. 获取所有步骤
	steps, err := repository.GetStepsByPipelineID(id)
	if err != nil {
		return nil, fmt.Errorf("获取步骤列表失败: %w", err)
	}

	// 3. 转换步骤为列表项
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

	// 4. 组装响应
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

// GetStepDetail 获取指定步骤的详细信息（含完整step_data）
func (s *PipelineService) GetStepDetail(pipelineID string, stepName string) (*models.StepDetailResponse, error) {
	// 1. 验证Pipeline存在
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	// 2. 获取步骤
	step, err := repository.GetStepByName(pipelineID, stepName)
	if err != nil {
		return nil, fmt.Errorf("步骤 %s 不存在", stepName)
	}

	// 3. 处理step_data为json.RawMessage
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

// CancelPipeline 取消Pipeline（仅pending或running状态可取消）
func (s *PipelineService) CancelPipeline(id string) error {
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return ErrPipelineNotFound
	}

	// 只有pending或running状态可以取消
	if pipeline.Status != models.PipelineStatusPending && pipeline.Status != models.PipelineStatusRunning {
		return ErrPipelineNotCancellable
	}

	return repository.UpdatePipelineStatus(id, pipeline.CurrentStep, models.PipelineStatusCancelled)
}

// DeletePipeline 删除Pipeline（运行中的不可删除）
func (s *PipelineService) DeletePipeline(id string) error {
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return ErrPipelineNotFound
	}

	// 运行中的不可删除
	if pipeline.Status == models.PipelineStatusRunning {
		return ErrPipelineNotDeletable
	}

	return repository.DeletePipeline(id)
}

// ==================== Pipeline 执行引擎 ====================

// StartPipeline 启动Pipeline执行（从dbCheck开始，autoMode下自动推进到scanner）
// P4-1：同步执行dbCheck
// P4-2：autoMode时同步执行scanner
// P5：改为异步goroutine
func (s *PipelineService) StartPipeline(id string) (*models.PipelineDetailResponse, error) {
	// 1. 获取Pipeline
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	// 2. 校验状态：只有pending状态可以启动
	if pipeline.Status != models.PipelineStatusPending {
		return nil, ErrPipelineNotPending
	}

	// 3. 更新Pipeline为running状态
	if err := repository.UpdatePipelineStatus(id, models.StepDbCheck, models.PipelineStatusRunning); err != nil {
		return nil, fmt.Errorf("更新Pipeline状态失败: %w", err)
	}

	// 4. 执行dbCheck步骤
	dbCheckErr := s.executeDbCheck(pipeline)

	if dbCheckErr != nil {
		// dbCheck失败 -> Pipeline标记为failed，返回详情（含失败信息）
		_ = repository.UpdatePipelineError(id, models.StepDbCheck, dbCheckErr.Error())
		return s.GetPipelineDetail(id)
	}

	// 5. dbCheck成功 -> 推进到scanner步骤
	if err := repository.UpdatePipelineStatus(id, models.StepScanner, models.PipelineStatusRunning); err != nil {
		return nil, fmt.Errorf("推进到Scanner失败: %w", err)
	}

	// 6. autoMode时自动执行scanner步骤（P4-2新增）
	if pipeline.AutoMode {
		// 重新读取最新Pipeline状态（含更新后的current_step）
		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			return s.GetPipelineDetail(id)
		}

		scannerErr := s.executeScanner(pipeline)
		if scannerErr != nil {
			// scanner失败 -> Pipeline标记为failed
			_ = repository.UpdatePipelineError(id, models.StepScanner, scannerErr.Error())
			return s.GetPipelineDetail(id)
		}

		// scanner成功 -> 推进到evaluator（P4-3实现时继续）
		// 目前P4-2只实现到scanner，停在evaluator步骤等待
		if err := repository.UpdatePipelineStatus(id, models.StepEvaluator, models.PipelineStatusRunning); err != nil {
			return nil, fmt.Errorf("推进到Evaluator失败: %w", err)
		}
	}

	// 7. 返回最新状态
	return s.GetPipelineDetail(id)
}

// ==================== dbCheck 步骤实现（P4-1）====================

// executeDbCheck 执行dbCheck步骤：验证课程索引存在且有效
// 验证规则：
//  1. 课程必须存在于courses表
//  2. course_indexes表必须有该课程的索引记录
//  3. 索引内容长度必须 > MinIndexLength (50字符)
//  4. 索引SHA-256校验码必须与实际内容一致
func (s *PipelineService) executeDbCheck(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepDbCheck

	// 标记步骤开始
	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动dbCheck失败: %w", err)
	}

	// 构造dbCheck结果对象
	result := &models.DbCheckResult{
		CourseCode: pipeline.CourseCode,
	}

	// 执行验证逻辑
	checkErr := s.doDbCheck(pipeline, result)

	// 计算耗时
	durationMs := time.Since(startTime).Milliseconds()

	if checkErr != nil {
		// 验证失败 -> 记录失败信息
		result.IsValid = false
		result.ErrorDetail = checkErr.Error()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, checkErr.Error())
		s.saveStepData(pipeline.ID, stepName, result.ToJSON())
		return checkErr
	}

	// 验证成功 -> 标记步骤完成
	result.IsValid = true
	if err := repository.CompleteStep(pipeline.ID, stepName, durationMs, result.ToJSON(), "", 0); err != nil {
		return fmt.Errorf("保存dbCheck结果失败: %w", err)
	}

	return nil
}

// doDbCheck 执行实际的索引验证逻辑
func (s *PipelineService) doDbCheck(pipeline *models.Pipeline, result *models.DbCheckResult) error {
	// 1. 查询课程记录
	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		return fmt.Errorf("课程 %s 不存在", pipeline.CourseCode)
	}
	result.CourseID = course.ID
	if course.ExternalModuleID != nil {
		result.ModuleID = *course.ExternalModuleID
	}

	// 2. 查询课程索引
	idx, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		result.HasIndex = false
		return ErrDbCheckIndexMissing
	}
	result.HasIndex = true
	result.IndexHash = idx.IndexHash
	result.PageCount = idx.PageCount
	result.TotalLength = idx.TotalLength

	// 3. 验证索引内容长度
	if len(idx.IndexContent) < models.MinIndexLength {
		return fmt.Errorf("%w (实际长度: %d, 最小要求: %d)",
			ErrDbCheckIndexTooShort, len(idx.IndexContent), models.MinIndexLength)
	}

	// 4. 验证SHA-256校验码
	actualHash := utils.SHA256Hash(idx.IndexContent)
	if actualHash != idx.IndexHash {
		return fmt.Errorf("%w (存储: %s, 实际: %s)",
			ErrDbCheckIndexHashMismatch, idx.IndexHash[:16]+"...", actualHash[:16]+"...")
	}

	return nil
}

// ==================== Scanner 步骤实现（P4-2新增）====================

// executeScanner 执行scanner步骤：调用Prompt A对课程进行K12定位分析
// 输入：课程索引内容
// 输出：{target, ability_targets, grade_standard, course_standard} JSON
// 流程：
//  1. 从数据库加载Prompt A（当前激活版本）
//  2. 获取课程索引内容
//  3. 构造用户消息（Prompt A + 课程索引）
//  4. 调用AI API（使用scanner场景配置）
//  5. 从AI输出中提取JSON
//  6. 保存结果到step_data
func (s *PipelineService) executeScanner(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepScanner

	// 标记步骤开始
	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动scanner失败: %w", err)
	}

	// 执行实际scanner逻辑
	result, callErr := s.doScanner(pipeline)

	// 计算耗时
	durationMs := time.Since(startTime).Milliseconds()

	if callErr != nil {
		// scanner失败
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, callErr.Error())
		// 如果result有部分数据（如raw_output），也保存下来供诊断
		if result != nil {
			s.saveStepData(pipeline.ID, stepName, result.ToJSON())
		}
		return callErr
	}

	// scanner成功 -> 保存结果
	if err := repository.CompleteStep(
		pipeline.ID, stepName, durationMs,
		result.ToJSON(), result.ModelUsed, result.TokensUsed,
	); err != nil {
		return fmt.Errorf("保存scanner结果失败: %w", err)
	}

	return nil
}

// doScanner 执行实际的scanner逻辑（调用AI + 解析JSON）
func (s *PipelineService) doScanner(pipeline *models.Pipeline) (*models.ScannerResult, error) {
	result := &models.ScannerResult{}

	// -------- 1. 加载 Prompt A --------
	promptA, err := repository.GetCurrentPrompt("prompt_a")
	if err != nil || promptA == nil {
		return nil, ErrScannerPromptMissing
	}

	// -------- 2. 获取课程索引内容 --------
	// 先获取课程ID
	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		return nil, fmt.Errorf("scanner: 课程 %s 不存在", pipeline.CourseCode)
	}
	courseIndex, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		return nil, fmt.Errorf("scanner: 课程索引不存在，请先执行dbCheck")
	}

	// -------- 3. 获取AI有效配置（三级回退）--------
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey,
		"scanner",           // 场景代码：使用scanner场景配置
		s.cfg.AIAPIBaseURL,  // .env兜底：API地址
		s.cfg.AIAPIKey,      // .env兜底：API Key
		s.cfg.AIDefaultModel, // .env兜底：模型名
	)
	if err != nil {
		return nil, fmt.Errorf("scanner: 获取AI配置失败: %w", err)
	}

	// -------- 4. 构造AI调用消息 --------
	// Prompt A作为系统提示词（定义AI角色和输出格式）
	// 课程索引作为用户消息（待分析的课程内容）
	systemPrompt := promptA.Content
	userPrompt := fmt.Sprintf("请分析以下课程索引内容，按照要求输出JSON格式的定位信息：\n\n%s", courseIndex.IndexContent)

	// -------- 5. 调用AI --------
	callResult, err := ai.CallAI(aiCfg, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrScannerAIFailed, err.Error())
	}

	// 保存原始输出
	result.RawOutput = callResult.Content
	result.ModelUsed = callResult.ModelUsed
	result.TokensUsed = callResult.TokensUsed

	// -------- 6. 从AI输出中提取JSON --------
	jsonStr, ok := ai.ExtractJSON(callResult.Content)
	if !ok {
		result.IsValid = false
		// 原始输出已保存，返回错误（但result有诊断数据）
		return result, fmt.Errorf("%w (AI输出前200字符: %s)",
			ErrScannerParseFailed, truncate(callResult.Content, 200))
	}

	// 验证提取的JSON包含必要字段
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		result.IsValid = false
		return result, fmt.Errorf("%w (JSON解析错误: %s)", ErrScannerParseFailed, err.Error())
	}

	// 检查必要字段是否存在（target是必须的）
	if _, hasTarget := parsed["target"]; !hasTarget {
		result.IsValid = false
		return result, fmt.Errorf("%w (缺少必要字段target，实际字段: %v)",
			ErrScannerParseFailed, getMapKeys(parsed))
	}

	// 保存解析后的JSON
	result.Parsed = json.RawMessage(jsonStr)
	result.IsValid = true

	return result, nil
}

// ==================== 工具方法 ====================

// saveStepData 单独更新步骤的step_data字段
// 用于在FailStep之后补充保存诊断数据
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

// truncate 截断字符串到指定长度，超出部分加"..."
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// getMapKeys 获取map的所有key列表（用于错误诊断）
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
