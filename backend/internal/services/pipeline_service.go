package services

// pipeline_service.go — Pipeline核心服务主文件
//
// 职责：
//   - PipelineService 结构体定义与初始化
//   - 错误常量定义
//   - Pipeline CRUD（Create/List/GetDetail/GetStepDetail/Cancel/Delete）
//   - Dashboard 统计
//   - 公共工具方法（extractScannerParsed/extractEvalScores/saveStepData/truncate等）
//   - sanitizeVerifyStepData verify步骤数据脱敏（v68新增）
//
// v68变更：
//   - GetStepDetail增加callerRole参数，verify步骤对非admin用户脱敏
//   - 新增sanitizeVerifyStepData函数：隐藏索引原文，清洗eval_output中的索引行
//   - 新增reIndexLine正则：匹配TE-DNA索引结构行

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"context"

	"tedna/internal/config"
	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 包级正则变量（避免每次调用重新编译，提升性能）====================

// v68新增：reIndexLine 匹配TE-DNA索引结构行，用于verify数据脱敏
// 格式如：P01:PT:LC|IM:5|DF:3|AI:0|EV:0 [S]...[K]...
var reIndexLine = regexp.MustCompile(`(?m)^P\d{1,3}:PT:[A-Z]{2}\|.*$`)

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
	// 断点续跑错误常量
	ErrRestartInvalidStep      = errors.New("无效的步骤名称")
	ErrRestartPipelineBusy     = errors.New("Pipeline正在运行中，无法重启")
	ErrRestartStepNotAllowed   = errors.New("该步骤不支持从此处重启（verify步骤请使用验收入口）")
	ErrRestartPermissionDenied = errors.New("已完成的Pipeline重跑仅限管理员和高级操作员操作")
)

// ==================== PipelineService 结构体 ====================

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
// v68改动：增加callerRole参数，verify步骤对非admin用户脱敏（隐藏索引结构）
func (s *PipelineService) GetStepDetail(pipelineID string, stepName string, callerRole string) (*models.StepDetailResponse, error) {
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

	// v68新增：verify步骤数据对非admin用户脱敏
	// 隐藏generated_index（含TE-DNA索引结构），清洗eval_output中的索引行
	if stepName == models.StepVerify && callerRole != models.RoleAdmin && step.StepData != "" && step.StepData != "null" {
		stepDataRaw = sanitizeVerifyStepData(step.StepData)
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
// v65-bugfix: 验收过程中取消回退到finalized
func (s *PipelineService) CancelPipeline(id string) error {
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusPending && pipeline.Status != models.PipelineStatusRunning {
		return ErrPipelineNotCancellable
	}

	if pipeline.CurrentStep == models.StepVerify && pipeline.Status == models.PipelineStatusRunning {
		return repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusFinalized)
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

// ==================== 公共工具方法 ====================

// extractScannerParsed 从Scanner步骤的step_data中提取parsed JSON
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

// extractEvalScores 从Evaluator AI输出中提取E1-E4四维度评分
// 返回: scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4, hardConstraint, grade, parseOk
func extractEvalScores(output string) (float64, float64, float64, float64, float64, string, string, bool) {
	// 优先从 SCORE_BLOCK 标签提取
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

	// 兜底：从全文按维度模式提取
	var scoreE1, scoreE2, scoreE3, scoreE4 float64
	dimCount := 0
	dimPatterns := []struct {
		re    *regexp.Regexp
		field *float64
	}{
		{reEvalDimE1, &scoreE1}, {reEvalDimE2, &scoreE2},
		{reEvalDimE3, &scoreE3}, {reEvalDimE4, &scoreE4},
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

// saveStepData 保存步骤的step_data到数据库（不返回error，失败静默忽略）
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

// truncate 截断字符串到指定最大长度
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// getMapKeys 获取map的所有key
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// safeParseFloat 安全解析浮点数（解析失败返回0）
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

// getSubmatch 安全获取正则子匹配结果
func getSubmatch(m []string) string {
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// getStepOrder 获取步骤的执行顺序编号（辅助函数）
func getStepOrder(stepName string) int {
	for _, def := range models.StepDefinitions {
		if def.Name == stepName {
			return def.Order
		}
	}
	return 999
}

// ==================== verify步骤数据脱敏（v68新增）====================

// sanitizeVerifyStepData 对verify步骤数据进行脱敏处理
// 非admin用户看到的verify数据：隐藏索引原文，eval_output中清洗索引结构行
// 保留评分结论、问题诊断等审核员需要的信息
func sanitizeVerifyStepData(stepData string) []byte {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stepData), &data); err != nil {
		return []byte("null")
	}

	// 1. 隐藏generated_index（替换为摘要信息）
	if idx, ok := data["generated_index"].(string); ok && len(idx) > 0 {
		data["generated_index"] = fmt.Sprintf(
			"（索引内容已隐藏，共%d字符。仅管理员可查看完整索引。）", len(idx))
	}

	// 2. eval_output中清洗索引结构行（P01:PT:LC|IM:5|DF:3... 格式）
	if output, ok := data["eval_output"].(string); ok && len(output) > 0 {
		data["eval_output"] = reIndexLine.ReplaceAllString(output, "[索引行已隐藏]")
	}

	// 3. eval_round_scores中每轮的output也做清洗
	if rounds, ok := data["eval_round_scores"].([]interface{}); ok {
		for i, r := range rounds {
			if roundMap, ok := r.(map[string]interface{}); ok {
				if rOutput, ok := roundMap["output"].(string); ok && len(rOutput) > 0 {
					roundMap["output"] = reIndexLine.ReplaceAllString(rOutput, "[索引行已隐藏]")
					rounds[i] = roundMap
				}
			}
		}
		data["eval_round_scores"] = rounds
	}

	result, err := json.Marshal(data)
	if err != nil {
		return []byte("null")
	}
	return result
}
