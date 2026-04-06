package services

// pipeline_review.go — 审核决策 + 定稿 + 批量操作 + 发布
//
// v68变更：
//   - AIFixPage增强：使用ai_fix独立场景(Sonnet+64000) + 专用系统提示词 + 修改说明提取 + 参考页面支持
//   - 新增AIFixResult/extractFixSummary/extractFixedHTML/aiFixSystemPrompt

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 快捷通过 ====================

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

// ==================== 审核决策方法 ====================

// GetGeneratedPages 获取Pipeline的所有生成页面
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

	return repository.UpdatePageDecision(pipelineID, pageNumber, decision, finalHTML)
}

// ==================== 定稿流程 ====================

// SubmitFinalize 提交定稿申请
func (s *PipelineService) SubmitFinalize(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusReviewQueue &&
		pipeline.Status != models.PipelineStatusNeedsHuman {
		return ErrSubmitFinalizeNotAllowed
	}

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

	return repository.UpdatePipelineStatus(pipelineID, models.StepReview, models.PipelineStatusPendingFinalize)
}

// ConfirmFinalize 确认定稿
func (s *PipelineService) ConfirmFinalize(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusPendingFinalize {
		return ErrConfirmFinalizeNotAllowed
	}

	total, decided, err := repository.GetPageDecisionStats(pipelineID)
	if err != nil {
		return fmt.Errorf("检查页面决策状态失败: %w", err)
	}

	reviewStep, err := repository.GetStepByName(pipelineID, models.StepReview)
	if err == nil && reviewStep.Status != models.StepStatusDone {
		_ = repository.StartStep(pipelineID, models.StepReview)
		statsJSON := fmt.Sprintf(`{"total_pages":%d,"decided_pages":%d,"finalized_at":"%s"}`,
			total, decided, time.Now().Format(time.RFC3339))
		_ = repository.CompleteStep(pipelineID, models.StepReview, 0, statsJSON, "", 0)
	}

	return repository.CompletePipeline(pipelineID, models.PipelineStatusFinalized)
}

// RejectFinalize 退回定稿
func (s *PipelineService) RejectFinalize(pipelineID string, rejectReason string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusPendingFinalize {
		return ErrRejectFinalizeNotAllowed
	}
	return repository.UpdatePipelineRejectReason(pipelineID, rejectReason)
}

// FinalizePipeline 直接定稿（兼容旧API）
func (s *PipelineService) FinalizePipeline(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
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

	return repository.CompletePipeline(pipelineID, models.PipelineStatusFinalized)
}

// ==================== AI快修 ====================

// aiFixSystemPrompt AI快修专用系统提示词（v68新增）
const aiFixSystemPrompt = `你是一个精确的HTML课件修复专家。你的唯一任务是根据审核员的修复指令，在现有HTML代码基础上做最小化修改。

## 铁律（违反任何一条即为失败）

1. **只改指令要求的部分**：审核员没提到的地方，一个字符都不能动
2. **保持原有格式和布局**：CSS样式、class名称、id属性、HTML结构层级必须原封不动保留
3. **禁止重写页面**：你不是在"重新生成"页面，你是在"修补"页面
4. **导航栏/视频/图片/音频等媒体元素**：绝对禁止任何改动
5. **保留所有注释和空行**：原代码中的注释和格式化空行必须保留
6. **输出完整HTML**：输出修改后的完整HTML，不要输出代码片段

## 输出格式要求

你必须先输出修改说明，再输出完整HTML。格式如下：

<<<FIX_SUMMARY>>>
- 修改点1：简要说明改了什么（例如：将第3题的选项A从"光合作用"改为"呼吸作用"）
- 修改点2：简要说明改了什么
<<<END_FIX_SUMMARY>>>

<<<FIXED_HTML>>>
这里是修改后的完整HTML代码
<<<END_FIXED_HTML>>>

## 修改说明的规范
- 每个修改点用"- "开头，一行一个
- 说明要具体：改了哪里、从什么改成什么
- 如果修改涉及多处，逐一列出
- 总共不超过10个修改点

## 参考页面的使用
- 如果审核员提供了参考页面，仅作为风格/格式/内容的参考依据
- 参考页面的代码不能直接复制到当前页面
- 你只能参考其设计思路、交互方式、排版风格来改进当前页面`

// reFixSummary 提取修改说明的正则（v68新增）
var reFixSummary = regexp.MustCompile(`(?s)<<<FIX_SUMMARY>>>(.*?)<<<END_FIX_SUMMARY>>>`)

// reFixedHTML 提取修复后HTML的正则（v68新增）
var reFixedHTML = regexp.MustCompile(`(?s)<<<FIXED_HTML>>>(.*?)<<<END_FIXED_HTML>>>`)

// AIFixResult AI快修返回结果（v68增强）
type AIFixResult struct {
	NewHTML    string `json:"new_html"`
	FixSummary string `json:"fix_summary"`
	HTMLLength int    `json:"html_length"`
}

// AIFixPage AI快修单个页面
// v68增强：使用ai_fix独立场景(Sonnet) + 专用系统提示词 + 修改说明提取 + 参考页面支持
// 参数 referencePageNums：审核员选择的参考页码列表（可为空）
func (s *PipelineService) AIFixPage(pipelineID string, pageNumber int, fixInstruction string, referencePageNums []int) (*AIFixResult, error) {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue:     true,
		models.PipelineStatusNeedsHuman:      true,
		models.PipelineStatusFinalized:       true,
		models.PipelineStatusPendingFinalize: true,
	}
	if !allowedStatuses[pipeline.Status] {
		return nil, ErrPipelineNotReviewable
	}

	// 获取所有页面数据（供当前页面+参考页面使用）
	pages, err := repository.GetGeneratedPagesWithHTML(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("获取页面数据失败: %w", err)
	}

	// 查找当前页面
	var currentPage *repository.GeneratedPageFullRow
	// 构建页码到页面的映射（用于参考页面查找）
	pageMap := make(map[int]*repository.GeneratedPageFullRow)
	for _, p := range pages {
		pageMap[p.PageNumber] = p
		if p.PageNumber == pageNumber {
			currentPage = p
		}
	}
	if currentPage == nil {
		return nil, ErrPageNotFound
	}

	// 获取当前最新HTML（优先级：final_html > generated_html > original_html）
	currentHTML := currentPage.FinalHTML
	if currentHTML == "" {
		currentHTML = currentPage.GeneratedHTML
	}
	if currentHTML == "" {
		currentHTML = currentPage.OriginalHTML
	}
	if currentHTML == "" {
		return nil, fmt.Errorf("页面P%d无可用HTML内容", pageNumber)
	}

	// v68改动：使用ai_fix独立场景（Sonnet + 64000 tokens）替代generator场景（Opus + 200000）
	// AI快修只修改单页局部内容，不需要最强模型和最大token
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey, "ai_fix",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf("获取AI配置失败: %w", err)
	}

	// 使用专用系统提示词
	systemPrompt := aiFixSystemPrompt

	// 构建用户提示词
	var userPromptParts []string

	userPromptParts = append(userPromptParts,
		fmt.Sprintf("【当前页面】P%02d — %s", pageNumber, currentPage.PageTitle),
		"",
		"【当前页面完整HTML — 你必须在此基础上修复，禁止重写】",
		currentHTML,
		"",
	)

	// v68新增：拼入参考页面HTML（审核员选择的其他页面作为参考）
	if len(referencePageNums) > 0 {
		userPromptParts = append(userPromptParts,
			"══════════════════════════════════════════════",
			"【参考页面（仅供参考风格/格式/内容，禁止直接复制代码）】",
			"",
		)
		refCount := 0
		for _, refPN := range referencePageNums {
			if refPN == pageNumber {
				continue // 跳过当前页面自身
			}
			refPage, exists := pageMap[refPN]
			if !exists {
				continue
			}
			// 获取参考页面的最新HTML
			refHTML := refPage.FinalHTML
			if refHTML == "" {
				refHTML = refPage.GeneratedHTML
			}
			if refHTML == "" {
				refHTML = refPage.OriginalHTML
			}
			if refHTML == "" {
				continue
			}
			refCount++
			userPromptParts = append(userPromptParts,
				fmt.Sprintf("--- 参考页面 P%02d: %s ---", refPN, refPage.PageTitle),
				refHTML,
				"",
			)
			// 最多5个参考页面，避免token爆炸
			if refCount >= 5 {
				userPromptParts = append(userPromptParts, "（已达参考页面上限，后续省略）")
				break
			}
		}
		userPromptParts = append(userPromptParts, "")
	}

	userPromptParts = append(userPromptParts,
		"══════════════════════════════════════════════",
		"【审核员修复指令（必须严格执行，只改这里要求的部分）】",
		fixInstruction,
	)

	userPrompt := strings.Join(userPromptParts, "\n")

	// 调用AI
	callResult, callErr := s.callAIWithSemaphore(aiCfg, systemPrompt, userPrompt)
	if callErr != nil {
		return nil, fmt.Errorf("%w: %s", ErrAIFixFailed, callErr.Error())
	}

	aiOutput := callResult.Content

	// 提取修改说明
	fixSummary := extractFixSummary(aiOutput)

	// 优先从<<<FIXED_HTML>>>标签提取HTML
	newHTML := extractFixedHTML(aiOutput)
	if newHTML == "" {
		// 降级：使用原有的extractGeneratedHTML（定义在generator_service.go）
		newHTML = extractGeneratedHTML(aiOutput)
	}

	if len(newHTML) < 100 {
		return nil, fmt.Errorf("%w: AI输出HTML过短(%d字符)", ErrAIFixFailed, len(newHTML))
	}

	// 保存修复后的HTML
	if err := repository.UpdateGeneratedPageHTML(pipelineID, pageNumber, newHTML, newHTML); err != nil {
		return nil, fmt.Errorf("保存修复后HTML失败: %w", err)
	}

	return &AIFixResult{
		NewHTML:    newHTML,
		FixSummary: fixSummary,
		HTMLLength: len(newHTML),
	}, nil
}

// extractFixSummary 从AI输出中提取修改说明
func extractFixSummary(output string) string {
	match := reFixSummary.FindStringSubmatch(output)
	if len(match) >= 2 {
		summary := strings.TrimSpace(match[1])
		if summary != "" {
			return summary
		}
	}

	// 降级：查找第一个HTML标签之前的文本
	htmlStartIdx := strings.Index(output, "<!DOCTYPE")
	if htmlStartIdx < 0 {
		htmlStartIdx = strings.Index(output, "<html")
	}
	if htmlStartIdx < 0 {
		htmlStartIdx = strings.Index(output, "<div")
	}
	if htmlStartIdx > 20 {
		preText := strings.TrimSpace(output[:htmlStartIdx])
		if len(preText) > 10 && len(preText) < 2000 {
			return preText
		}
	}

	return ""
}

// extractFixedHTML 从AI输出中提取<<<FIXED_HTML>>>块中的HTML
func extractFixedHTML(output string) string {
	match := reFixedHTML.FindStringSubmatch(output)
	if len(match) >= 2 {
		html := strings.TrimSpace(match[1])
		if len(html) >= 100 {
			return html
		}
	}
	return ""
}

// ==================== 高分早停辅助方法 ====================

// fillEarlyStopPages 高分早停时将所有原始页面以 keep 操作写入 generated_pages
func (s *PipelineService) fillEarlyStopPages(pipeline *models.Pipeline) {
	if pipeline.ReviewRound >= 2 {
		return
	}

	_ = repository.DeleteGeneratedPagesByPipelineID(pipeline.ID)

	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil || course.ExternalModuleID == nil || *course.ExternalModuleID == 0 {
		return
	}
	moduleID := *course.ExternalModuleID

	ossService := NewOSSService(s.cfg)
	pageLessonMap, err := ossService.BuildPageLessonMap(moduleID)
	if err != nil || len(pageLessonMap) == 0 {
		return
	}

	pageNums := make([]int, 0, len(pageLessonMap))
	for pn := range pageLessonMap {
		pageNums = append(pageNums, pn)
	}
	for i := 0; i < len(pageNums); i++ {
		for j := i + 1; j < len(pageNums); j++ {
			if pageNums[i] > pageNums[j] {
				pageNums[i], pageNums[j] = pageNums[j], pageNums[i]
			}
		}
	}

	for _, pn := range pageNums {
		lessonID := pageLessonMap[pn]
		origHTML, fetchErr := ossService.FetchLessonHTML(lessonID)
		if fetchErr != nil || len(origHTML) < 100 {
			origHTML = ""
		}
		lidPtr := new(int)
		*lidPtr = lessonID
		pageTitle := fmt.Sprintf("P%02d", pn)
		_ = repository.CreateGeneratedPage(
			pipeline.ID, pn, pageTitle,
			"keep", origHTML, "", origHTML,
			lidPtr, "", "高分早停：原始课件质量已达标",
		)
	}
}

// shouldEarlyStop 检查是否满足高分早停条件
func (s *PipelineService) shouldEarlyStop(pipelineID string, pipeline *models.Pipeline) bool {
	if pipeline == nil {
		return false
	}
	evalStep, err := repository.GetStepByName(pipelineID, models.StepEvaluator)
	if err != nil || evalStep.Status != models.StepStatusDone {
		return false
	}
	if evalStep.StepData == "" || evalStep.StepData == "null" {
		return false
	}
	var evalData map[string]interface{}
	if err := json.Unmarshal([]byte(evalStep.StepData), &evalData); err != nil {
		return false
	}
	avgTotal, ok := evalData["avg_total"].(float64)
	if !ok || avgTotal <= 0 {
		return false
	}
	pCfg := models.ParsePipelineConfig(pipeline.Config)
	return avgTotal >= pCfg.Threshold
}

// getEvalAvgScore 获取Evaluator的均分
func (s *PipelineService) getEvalAvgScore(pipelineID string) float64 {
	evalStep, err := repository.GetStepByName(pipelineID, models.StepEvaluator)
	if err != nil || evalStep.StepData == "" || evalStep.StepData == "null" {
		return 0.0
	}
	var evalData map[string]interface{}
	if err := json.Unmarshal([]byte(evalStep.StepData), &evalData); err != nil {
		return 0.0
	}
	if avg, ok := evalData["avg_total"].(float64); ok {
		return avg
	}
	return 0.0
}

// GetOperatorUsers 获取所有操作员用户列表
func (s *PipelineService) GetOperatorUsers() ([]map[string]string, error) {
	return repository.ListOperatorUsers()
}

// ==================== 批量创建+批量启动 ====================

// BatchCreateResult 批量创建结果
type BatchCreateResult struct {
	TotalRequested int      `json:"total_requested"`
	CreatedIDs     []string `json:"created_ids"`
	SkippedCodes   []string `json:"skipped_codes"`
	SkippedReasons []string `json:"skipped_reasons"`
	FailedCodes    []string `json:"failed_codes"`
	FailedReasons  []string `json:"failed_reasons"`
}

// BatchCreatePipelines 批量创建Pipeline
func (s *PipelineService) BatchCreatePipelines(courseCodes []string, userID string) (*BatchCreateResult, error) {
	result := &BatchCreateResult{
		TotalRequested: len(courseCodes), CreatedIDs: []string{},
		SkippedCodes: []string{}, SkippedReasons: []string{},
		FailedCodes: []string{}, FailedReasons: []string{},
	}

	seen := make(map[string]bool)
	var uniqueCodes []string
	for _, code := range courseCodes {
		code = strings.TrimSpace(code)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		uniqueCodes = append(uniqueCodes, code)
	}

	for _, code := range uniqueCodes {
		req := &models.CreatePipelineRequest{CourseCode: code}
		resp, err := s.CreatePipeline(req, userID)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "已有运行中的Pipeline") || strings.Contains(errMsg, "课程不存在") {
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
	TotalRequested int      `json:"total_requested"`
	StartedIDs     []string `json:"started_ids"`
	SkippedIDs     []string `json:"skipped_ids"`
	SkippedReasons []string `json:"skipped_reasons"`
	FailedIDs      []string `json:"failed_ids"`
	FailedReasons  []string `json:"failed_reasons"`
}

// BatchStartPipelines 批量启动Pipeline
func (s *PipelineService) BatchStartPipelines(ids []string) (*BatchStartResult, error) {
	result := &BatchStartResult{
		TotalRequested: len(ids), StartedIDs: []string{},
		SkippedIDs: []string{}, SkippedReasons: []string{},
		FailedIDs: []string{}, FailedReasons: []string{},
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

// ==================== Pipeline分配 ====================

// AssignPipelineResult 分配结果
type AssignPipelineResult struct {
	PipelineID   string `json:"pipeline_id"`
	AssignedTo   string `json:"assigned_to"`
	AssignedName string `json:"assigned_name"`
}

// BatchAssignResult 批量分配结果
type BatchAssignResult struct {
	TotalRequested int      `json:"total_requested"`
	SuccessCount   int      `json:"success_count"`
	AssignedTo     string   `json:"assigned_to"`
	AssignedName   string   `json:"assigned_name"`
	FailedIDs      []string `json:"failed_ids"`
}

// AssignPipeline 分配Pipeline给指定用户
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

	assignedName := repository.GetAssignedUserName(assignedToUserID)
	return &AssignPipelineResult{
		PipelineID: pipelineID, AssignedTo: assignedToUserID, AssignedName: assignedName,
	}, nil
}

// BatchAssignPipelines 批量分配Pipeline
func (s *PipelineService) BatchAssignPipelines(pipelineIDs []string, assignedToUserID string) (*BatchAssignResult, error) {
	var assignPtr *string
	if assignedToUserID != "" {
		assignPtr = &assignedToUserID
	}

	successCount, err := repository.BatchAssignPipelines(pipelineIDs, assignPtr)
	if err != nil {
		return nil, fmt.Errorf("批量分配失败: %w", err)
	}

	assignedName := repository.GetAssignedUserName(assignedToUserID)
	return &BatchAssignResult{
		TotalRequested: len(pipelineIDs), SuccessCount: successCount,
		AssignedTo: assignedToUserID, AssignedName: assignedName, FailedIDs: []string{},
	}, nil
}

// ==================== 发布至课程平台 ====================

// ErrPublishNotVerified 只有验收通过的Pipeline才能发布
var ErrPublishNotVerified = fmt.Errorf("只有验收通过(verified)的Pipeline才能发布至课程平台")

// ErrPublishAlreadyDone 已发布过的Pipeline不能重复发布
var ErrPublishAlreadyDone = fmt.Errorf("该Pipeline已发布至课程平台，不可重复操作")

// PublishPipeline 发布Pipeline至课程平台（单向不可逆）
func (s *PipelineService) PublishPipeline(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}

	if pipeline.Status != models.PipelineStatusVerified {
		return ErrPublishNotVerified
	}

	if err := repository.UpdatePipelineStatus(
		pipelineID,
		models.StepVerify,
		models.PipelineStatusPublished,
	); err != nil {
		return fmt.Errorf("更新Pipeline发布状态失败: %w", err)
	}

	return nil
}

// ==================== 单页HTML按需加载（v69新增，编号8方案2）====================

// GetSinglePageHTML 获取单页完整HTML数据
// v69新增：供审核页前端按需加载，选中页面时才请求完整HTML
func (s *PipelineService) GetSinglePageHTML(pipelineID string, pageNumber int) (*repository.GeneratedPageFullRow, error) {
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}
	page, err := repository.GetSinglePageHTML(pipelineID, pageNumber)
	if err != nil {
		return nil, fmt.Errorf("获取页面P%d HTML失败: %w", pageNumber, err)
	}
	return page, nil
}

// GetGeneratedPagesLightweight 获取所有页面轻量元数据（不含HTML内容）
// v69新增：审核页首次加载只获取元数据列表，大幅减少传输数据量
func (s *PipelineService) GetGeneratedPagesLightweight(pipelineID string) ([]*repository.GeneratedPageFullRow, error) {
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}
	pages, err := repository.GetGeneratedPagesLightweight(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("获取页面元数据失败: %w", err)
	}
	if pages == nil {
		pages = []*repository.GeneratedPageFullRow{}
	}
	return pages, nil
}

// ==================== AI快修流式调用（v69新增，编号5）====================

// AIFixPageStream AI快修流式版本 — 通过onChunk回调逐token推送
// 与AIFixPage逻辑一致，但使用CallAIStream流式调用
// 完成后自动保存HTML、提取修改说明
// 参数：
//   - onChunk: 每收到一个AI输出token时的回调（前端SSE推送用）
//   - onDone: AI输出完成后回调，传入最终结果（含new_html/fix_summary）
//   - onError: 出错时回调
func (s *PipelineService) AIFixPageStream(
	pipelineID string, pageNumber int, fixInstruction string, referencePageNums []int,
	onChunk func(chunk string),
	onDone func(result *AIFixResult),
	onError func(errMsg string),
) {
	// ===== 前置校验（与AIFixPage一致）=====
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		onError("Pipeline不存在")
		return
	}
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue:     true,
		models.PipelineStatusNeedsHuman:      true,
		models.PipelineStatusFinalized:       true,
		models.PipelineStatusPendingFinalize: true,
	}
	if !allowedStatuses[pipeline.Status] {
		onError("Pipeline不在审核状态，无法进行AI快修")
		return
	}

	// 获取所有页面数据
	pages, err := repository.GetGeneratedPagesWithHTML(pipelineID)
	if err != nil {
		onError("获取页面数据失败: " + err.Error())
		return
	}

	var currentPage *repository.GeneratedPageFullRow
	pageMap := make(map[int]*repository.GeneratedPageFullRow)
	for _, p := range pages {
		pageMap[p.PageNumber] = p
		if p.PageNumber == pageNumber {
			currentPage = p
		}
	}
	if currentPage == nil {
		onError(fmt.Sprintf("页面P%d不存在", pageNumber))
		return
	}

	// 获取当前最新HTML
	currentHTML := currentPage.FinalHTML
	if currentHTML == "" {
		currentHTML = currentPage.GeneratedHTML
	}
	if currentHTML == "" {
		currentHTML = currentPage.OriginalHTML
	}
	if currentHTML == "" {
		onError(fmt.Sprintf("页面P%d无可用HTML内容", pageNumber))
		return
	}

	// 获取AI配置
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey, "ai_fix",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		onError("获取AI配置失败: " + err.Error())
		return
	}

	// 构建提示词（与AIFixPage完全一致）
	systemPrompt := aiFixSystemPrompt
	var userPromptParts []string
	userPromptParts = append(userPromptParts,
		fmt.Sprintf("【当前页面】P%02d — %s", pageNumber, currentPage.PageTitle),
		"",
		"【当前页面完整HTML — 你必须在此基础上修复，禁止重写】",
		currentHTML,
		"",
	)
	if len(referencePageNums) > 0 {
		userPromptParts = append(userPromptParts,
			"══════════════════════════════════════════════",
			"【参考页面（仅供参考风格/格式/内容，禁止直接复制代码）】", "",
		)
		refCount := 0
		for _, refPN := range referencePageNums {
			if refPN == pageNumber {
				continue
			}
			refPage, exists := pageMap[refPN]
			if !exists {
				continue
			}
			refHTML := refPage.FinalHTML
			if refHTML == "" {
				refHTML = refPage.GeneratedHTML
			}
			if refHTML == "" {
				refHTML = refPage.OriginalHTML
			}
			if refHTML == "" {
				continue
			}
			refCount++
			userPromptParts = append(userPromptParts,
				fmt.Sprintf("--- 参考页面 P%02d: %s ---", refPN, refPage.PageTitle),
				refHTML, "",
			)
			if refCount >= 5 {
				break
			}
		}
		userPromptParts = append(userPromptParts, "")
	}
	userPromptParts = append(userPromptParts,
		"══════════════════════════════════════════════",
		"【审核员修复指令（必须严格执行，只改这里要求的部分）】",
		fixInstruction,
	)
	userPrompt := strings.Join(userPromptParts, "\n")

	// ===== 流式AI调用（带信号量控制）=====
	if s.engine != nil {
		s.engine.AcquireAI()
		defer s.engine.ReleaseAI()
	}

	callResult, callErr := ai.CallAIStream(aiCfg, systemPrompt, userPrompt, func(chunk string) error {
		onChunk(chunk)
		return nil
	})

	if callErr != nil {
		onError("AI快修调用失败: " + callErr.Error())
		return
	}

	aiOutput := callResult.Content

	// 提取修改说明和HTML（与AIFixPage一致）
	fixSummary := extractFixSummary(aiOutput)
	newHTML := extractFixedHTML(aiOutput)
	if newHTML == "" {
		newHTML = extractGeneratedHTML(aiOutput)
	}
	if len(newHTML) < 100 {
		onError(fmt.Sprintf("AI输出HTML过短(%d字符)，修复可能失败", len(newHTML)))
		return
	}

	// 保存修复后的HTML
	if err := repository.UpdateGeneratedPageHTML(pipelineID, pageNumber, newHTML, newHTML); err != nil {
		onError("保存修复后HTML失败: " + err.Error())
		return
	}

	onDone(&AIFixResult{
		NewHTML:    newHTML,
		FixSummary: fixSummary,
		HTMLLength: len(newHTML),
	})
}
