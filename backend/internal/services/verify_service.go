package services

// ==================== P4.6-2 验收执行逻辑 + P4.6-3 2审自动流程 ====================
// P4.6-2：executeVerify方法：收集最终HTML → 调索引生成器(prompt_g) → 调Evaluator(prompt_b) 1轮 → 判定
// P4.6-2：VerifyPipeline入口方法：手动触发验收（POST /pipelines/{id}/verify）
// P4.6-3：startRetrialPipeline方法：验收失败后自动启动2审（evaluator 2轮→meta→translator→generator→审核队列）

import (
	"fmt"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 验收错误常量 ====================

var (
	// P4.6-2新增：验收流程错误常量
	ErrVerifyNotFinalized     = fmt.Errorf("Pipeline不是finalized状态，无法启动验收")
	ErrVerifyPromptGMissing   = fmt.Errorf("Prompt G（索引生成器）未配置，请先在提示词管理中设置prompt_g")
	ErrVerifyPromptBMissing   = fmt.Errorf("Prompt B（评估器）未配置，请先在提示词管理中设置prompt_b")
	ErrVerifyDictMissing      = fmt.Errorf("解压缩字典未配置（验收需要dict）")
	ErrVerifyScannerNotDone   = fmt.Errorf("Scanner步骤未完成，无法执行验收")
	ErrVerifyNoPages          = fmt.Errorf("没有生成页面，无法执行验收")
	ErrVerifyNoValidHTML      = fmt.Errorf("没有有效的最终HTML页面，无法执行验收")
	ErrVerifyIndexGenFailed   = fmt.Errorf("索引生成器AI调用失败")
	ErrVerifyIndexTooShort    = fmt.Errorf("索引生成器输出过短，可能生成失败")
	ErrVerifyEvalFailed       = fmt.Errorf("验收评估AI调用失败")
	ErrVerifyScoreExtractFail = fmt.Errorf("验收评估未能提取有效评分")
	// P4.6-3新增：2审流程错误常量
	ErrRetrialResetFailed = fmt.Errorf("2审重置步骤失败")
	ErrRetrialExecFailed  = fmt.Errorf("2审执行流程失败")
)

// ==================== VerifyPipeline 入口方法 ====================

// VerifyPipeline 手动触发验收流程
// P4.6-2新增：POST /pipelines/{id}/verify 的业务逻辑入口
// P4.6-3增强：验收失败后根据review_round自动决定后续动作
//   - review_round=1 且 verify_failed → 自动启动2审流程（goroutine异步执行）
//   - review_round≥2 且 verify_failed → 标记needs_human（严重质量问题，不再自动重试）
// 前置条件：Pipeline状态必须是 finalized
// 流程：更新状态为running → 执行verify步骤 → 根据评分设置verified/verify_failed → 判断是否触发2审
func (s *PipelineService) VerifyPipeline(pipelineID string) (*models.PipelineDetailResponse, error) {
	// 1. 获取Pipeline并检查状态
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	// 只有finalized状态才允许启动验收
	if pipeline.Status != models.PipelineStatusFinalized {
		return nil, ErrVerifyNotFinalized
	}

	// 2. 更新Pipeline状态为running，当前步骤为verify
	if err := repository.UpdatePipelineStatus(pipelineID, models.StepVerify, models.PipelineStatusRunning); err != nil {
		return nil, fmt.Errorf("更新Pipeline状态失败: %w", err)
	}

	// 3. 重新读取最新Pipeline
	pipeline, err = repository.GetPipelineByID(pipelineID)
	if err != nil {
		return s.GetPipelineDetail(pipelineID)
	}

	// 4. 执行验收步骤
	verifyErr := s.executeVerify(pipeline)
	if verifyErr != nil {
		// 验收执行失败（AI调用失败等技术错误），标记Pipeline为failed
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify, verifyErr.Error())
		return s.GetPipelineDetail(pipelineID)
	}

	// 5. 验收步骤执行成功后，根据verify步骤的step_data判断通过/失败
	verifyStep, stepErr := repository.GetStepByName(pipelineID, models.StepVerify)
	if stepErr != nil || verifyStep.Status != models.StepStatusDone {
		// 异常情况：步骤未正确完成
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify, "验收步骤未正确完成")
		return s.GetPipelineDetail(pipelineID)
	}

	// 从verify步骤的step_data提取passed字段
	if verifyStep.StepData != "" && verifyStep.StepData != "null" {
		// 使用简单的字符串匹配方式（避免重复导入encoding/json）
		if strings.Contains(verifyStep.StepData, `"passed":true`) {
			// ===== 验收通过 → verified =====
			if err := repository.CompletePipeline(pipelineID, models.PipelineStatusVerified); err != nil {
				return nil, fmt.Errorf("标记验收通过失败: %w", err)
			}
		} else {
			// ===== 验收未通过 → verify_failed =====
			if err := repository.CompletePipeline(pipelineID, models.PipelineStatusVerifyFailed); err != nil {
				return nil, fmt.Errorf("标记验收未通过失败: %w", err)
			}

			// P4.6-3：根据review_round决定后续动作
			if pipeline.ReviewRound <= 1 {
				// 初审验收失败 → 自动启动2审流程（goroutine异步执行，不阻塞当前请求）
				go s.startRetrialPipeline(pipelineID)
			} else {
				// 2审验收仍然失败 → 标记"严重质量问题"，需要人工介入
				_ = repository.UpdatePipelineStatus(pipelineID, models.StepVerify, models.PipelineStatusNeedsHuman)
			}
		}
	} else {
		// step_data为空，异常情况
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify, "验收步骤输出数据为空")
		return s.GetPipelineDetail(pipelineID)
	}

	return s.GetPipelineDetail(pipelineID)
}

// ==================== P4.6-3 2审自动流程 ====================

// startRetrialPipeline 验收失败后自动启动2审流程
// P4.6-3新增：异步执行（在goroutine中调用），不阻塞验收接口返回
// 流程：
//  1. 设置review_round=2
//  2. 清理旧的eval_rounds和generated_pages
//  3. 重置evaluator/meta/translator/generator/review/verify步骤为pending
//  4. 更新Pipeline状态为running
//  5. 依次执行：evaluator(2轮) → meta → translator → generator
//  6. 成功后放入审核队列（review_queue），标记为2审待审核
//  7. 失败则标记Pipeline为failed并记录错误
func (s *PipelineService) startRetrialPipeline(pipelineID string) {
	// 1. 更新review_round为2
	if err := repository.UpdatePipelineReviewRound(pipelineID, 2); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify,
			fmt.Sprintf("2审启动失败: 更新review_round失败: %s", err.Error()))
		return
	}

	// 2. 清理旧的评估轮次数据和生成页面数据
	_ = repository.DeleteEvalRoundsByPipelineID(pipelineID)
	_ = repository.DeleteGeneratedPagesByPipelineID(pipelineID)

	// 3. 重置需要重跑的步骤为pending状态（evaluator/meta/translator/generator/review/verify）
	// 注意：dbCheck和scanner不需要重跑（课程索引和定位信息不变）
	resetSteps := []string{
		models.StepEvaluator,
		models.StepMeta,
		models.StepTranslator,
		models.StepGenerator,
		models.StepReview,
		models.StepVerify,
	}
	if err := repository.ResetStepsForRetrial(pipelineID, resetSteps); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepEvaluator,
			fmt.Sprintf("2审启动失败: 重置步骤失败: %s", err.Error()))
		return
	}

	// 4. 更新Pipeline状态为running，当前步骤为evaluator
	if err := repository.UpdatePipelineStatus(pipelineID, models.StepEvaluator, models.PipelineStatusRunning); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepEvaluator,
			fmt.Sprintf("2审启动失败: 更新状态失败: %s", err.Error()))
		return
	}

	// 5. 重新读取Pipeline（review_round已更新为2）
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepEvaluator,
			fmt.Sprintf("2审启动失败: 读取Pipeline失败: %s", err.Error()))
		return
	}

	// ===== 阶段A：执行Evaluator（2轮评估，比初审的3轮少1轮以节省时间） =====
	// 临时修改Pipeline config中的eval_rounds为2
	pCfg := models.ParsePipelineConfig(pipeline.Config)
	originalEvalRounds := pCfg.EvalRounds
	pCfg.EvalRounds = 2
	pipeline.Config = pCfg.ToJSON()
	// 注意：这里只修改内存中的config，不写回数据库，因为executeEvaluator从pipeline.Config读取

	evalErr := s.executeEvaluator(pipeline)

	// 恢复原始eval_rounds（避免影响后续步骤读取config）
	pCfg.EvalRounds = originalEvalRounds
	pipeline.Config = pCfg.ToJSON()

	if evalErr != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepEvaluator,
			fmt.Sprintf("2审Evaluator失败: %s", evalErr.Error()))
		return
	}

	// ===== 阶段B：推进到Meta =====
	if err := repository.UpdatePipelineStatus(pipelineID, models.StepMeta, models.PipelineStatusRunning); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepMeta,
			fmt.Sprintf("2审推进到Meta失败: %s", err.Error()))
		return
	}

	// 重新读取Pipeline（current_step已更新）
	pipeline, err = repository.GetPipelineByID(pipelineID)
	if err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepMeta, "2审读取Pipeline失败")
		return
	}

	metaErr := s.executeMeta(pipeline)
	if metaErr != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepMeta,
			fmt.Sprintf("2审Meta失败: %s", metaErr.Error()))
		return
	}

	// ===== 阶段C：推进到Translator =====
	if err := repository.UpdatePipelineStatus(pipelineID, models.StepTranslator, models.PipelineStatusRunning); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepTranslator,
			fmt.Sprintf("2审推进到Translator失败: %s", err.Error()))
		return
	}

	pipeline, err = repository.GetPipelineByID(pipelineID)
	if err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepTranslator, "2审读取Pipeline失败")
		return
	}

	transErr := s.executeTranslator(pipeline)
	if transErr != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepTranslator,
			fmt.Sprintf("2审Translator失败: %s", transErr.Error()))
		return
	}

	// ===== 阶段D：推进到Generator =====
	if err := repository.UpdatePipelineStatus(pipelineID, models.StepGenerator, models.PipelineStatusRunning); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepGenerator,
			fmt.Sprintf("2审推进到Generator失败: %s", err.Error()))
		return
	}

	pipeline, err = repository.GetPipelineByID(pipelineID)
	if err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepGenerator, "2审读取Pipeline失败")
		return
	}

	genErr := s.executeGenerator(pipeline)
	if genErr != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepGenerator,
			fmt.Sprintf("2审Generator失败: %s", genErr.Error()))
		return
	}

	// ===== 阶段E：2审完成，放入审核队列 =====
	if err := repository.UpdatePipelineStatus(pipelineID, models.StepReview, models.PipelineStatusReviewQueue); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepReview,
			fmt.Sprintf("2审推进到Review失败: %s", err.Error()))
		return
	}

	// 2审流程成功完成，Pipeline现在处于review_queue状态，等待人工审核
	// 人工审核后定稿 → 再次触发验收 → 通过则verified，不通过则needs_human
}

// ==================== executeVerify 核心执行方法 ====================

// executeVerify 执行验收步骤：收集最终HTML → 索引生成器(prompt_g) → Evaluator(prompt_b) 1轮 → 判定
// P4.6-2新增：验收步骤的核心执行逻辑
func (s *PipelineService) executeVerify(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepVerify

	// 启动verify步骤
	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动verify步骤失败: %w", err)
	}

	// 解析Pipeline配置获取阈值
	pCfg := models.ParsePipelineConfig(pipeline.Config)
	threshold := pCfg.Threshold // 默认9.0

	// ===== 阶段1：加载所需提示词 =====

	// 1a. 加载 Prompt G（索引生成器 v4.8）
	promptG, err := repository.GetCurrentPromptByKey("prompt_g")
	if err != nil || promptG == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrVerifyPromptGMissing.Error())
		return ErrVerifyPromptGMissing
	}

	// 1b. 加载 Prompt B（评估器）
	promptB, err := repository.GetCurrentPromptByKey("prompt_b")
	if err != nil || promptB == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrVerifyPromptBMissing.Error())
		return ErrVerifyPromptBMissing
	}

	// 1c. 加载解压缩字典
	dict, err := repository.GetCurrentPromptByKey("dict")
	if err != nil || dict == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrVerifyDictMissing.Error())
		return ErrVerifyDictMissing
	}

	// 1d. ability_table可选
	abilityTable, _ := repository.GetCurrentPromptByKey("ability_table")

	// ===== 阶段2：获取Scanner课程定位 =====
	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrVerifyScannerNotDone.Error())
		return ErrVerifyScannerNotDone
	}
	scannerLocationJSON := extractScannerParsed(scannerStep.StepData)

	// ===== 阶段3：收集最终HTML =====
	finalHTMLContent, pageCount, err := s.collectFinalHTML(pipeline.ID)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, err.Error())
		return err
	}

	// ===== 阶段4：调用索引生成器(prompt_g)压缩索引 =====

	// 4a. 获取AI配置（使用evaluator场景，因为索引生成器也是评估类任务）
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey,
		"evaluator",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("verify: 获取AI配置失败: %s", err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	// 4b. 构造索引生成器的用户消息：课程定位 + 最终HTML
	indexGenUserParts := []string{
		"【课程定位】",
		scannerLocationJSON,
		"",
		fmt.Sprintf("【最终课件HTML（共%d页）】", pageCount),
		"以下是经过人工审核定稿后的最终课件HTML内容，请按照要求压缩为TE-DNA课程索引+模块索引。",
		"",
		finalHTMLContent,
		"",
		"禁止输出<thinking>标签或任何思维过程标记。",
	}
	indexGenUserPrompt := strings.Join(indexGenUserParts, "\n")

	// 4c. 调用AI索引生成器
	var totalTokens int
	var lastModelUsed string

	indexCallResult, indexCallErr := ai.CallAI(aiCfg, promptG.Content, indexGenUserPrompt)
	if indexCallErr != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("%s: %s", ErrVerifyIndexGenFailed.Error(), indexCallErr.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	generatedIndex := indexCallResult.Content
	totalTokens += indexCallResult.TokensUsed
	lastModelUsed = indexCallResult.ModelUsed

	// 4d. 验证索引生成结果（至少应有一定长度）
	if len(generatedIndex) < 200 {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("%s (输出长度: %d字符)", ErrVerifyIndexTooShort.Error(), len(generatedIndex))
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	// ===== 阶段5：调用Evaluator(prompt_b) 1轮评估 =====

	// 5a. 构造Evaluator的用户消息：课程定位 + 新索引 + 解压缩字典
	evalUserParts := []string{
		"【课程定位】",
		scannerLocationJSON,
		"",
		"【待评估索引】",
		generatedIndex,
		"",
		"【TE-DNA解压缩字典】",
		dict.Content,
	}
	// 如果有能力定位表，追加
	if abilityTable != nil && len(abilityTable.Content) > 20 {
		evalUserParts = append(evalUserParts, "", "【能力定位表】", abilityTable.Content)
	}
	evalUserParts = append(evalUserParts, "", "禁止输出<thinking>标签或任何思维过程标记。")
	evalUserPrompt := strings.Join(evalUserParts, "\n")

	// 5b. 调用AI评估
	evalCallResult, evalCallErr := ai.CallAI(aiCfg, promptB.Content, evalUserPrompt)
	if evalCallErr != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("%s: %s", ErrVerifyEvalFailed.Error(), evalCallErr.Error())
		// 即使评估调用失败，也保存已生成的索引到step_data
		verifyResult := &models.VerifyResult{
			GeneratedIndex: truncate(generatedIndex, 50000),
			ReviewRound:    pipeline.ReviewRound,
			ModelUsed:      lastModelUsed,
			TokensUsed:     totalTokens,
			LatencyMs:      time.Since(startTime).Milliseconds(),
		}
		s.saveStepData(pipeline.ID, stepName, verifyResult.ToJSON())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	evalOutput := evalCallResult.Content
	totalTokens += evalCallResult.TokensUsed
	lastModelUsed = evalCallResult.ModelUsed

	// 5c. 提取评估评分（复用extractEvalScores方法，从SCORE_BLOCK中提取）
	scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4, _, _, parseOk := extractEvalScores(evalOutput)
	if !parseOk || scoreTotal < 0 {
		durationMs := time.Since(startTime).Milliseconds()
		// 评分提取失败，保存诊断数据
		verifyResult := &models.VerifyResult{
			GeneratedIndex: truncate(generatedIndex, 50000),
			EvalOutput:     truncate(evalOutput, 50000),
			ReviewRound:    pipeline.ReviewRound,
			ModelUsed:      lastModelUsed,
			TokensUsed:     totalTokens,
			LatencyMs:      time.Since(startTime).Milliseconds(),
		}
		s.saveStepData(pipeline.ID, stepName, verifyResult.ToJSON())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrVerifyScoreExtractFail.Error())
		return ErrVerifyScoreExtractFail
	}

	// ===== 阶段6：组装结果并判定 =====
	passed := scoreTotal >= threshold

	verifyResult := &models.VerifyResult{
		GeneratedIndex: truncate(generatedIndex, 50000),
		EvalScore:      scoreTotal,
		EvalOutput:     truncate(evalOutput, 50000),
		EvalE1:         scoreE1,
		EvalE2:         scoreE2,
		EvalE3:         scoreE3,
		EvalE4:         scoreE4,
		Passed:         passed,
		ReviewRound:    pipeline.ReviewRound,
		ModelUsed:      lastModelUsed,
		TokensUsed:     totalTokens,
		LatencyMs:      time.Since(startTime).Milliseconds(),
	}

	// 保存verify步骤完成
	durationMs := time.Since(startTime).Milliseconds()
	if err := repository.CompleteStep(
		pipeline.ID, stepName, durationMs,
		verifyResult.ToJSON(), lastModelUsed, totalTokens,
	); err != nil {
		return fmt.Errorf("保存verify结果失败: %w", err)
	}

	return nil
}

// ==================== 收集最终HTML辅助方法 ====================

// collectFinalHTML 收集Pipeline的所有最终HTML页面内容
// P4.6-2新增：遍历generated_pages，根据decision选择最终HTML版本
// 返回：拼接后的HTML内容字符串、有效页面数、错误
func (s *PipelineService) collectFinalHTML(pipelineID string) (string, int, error) {
	// 从数据库获取所有生成页面（含完整HTML）
	pages, err := repository.GetGeneratedPagesWithHTML(pipelineID)
	if err != nil {
		return "", 0, fmt.Errorf("获取生成页面失败: %w", err)
	}
	if len(pages) == 0 {
		return "", 0, ErrVerifyNoPages
	}

	// 遍历页面，根据decision选择最终HTML版本
	var htmlParts []string
	validCount := 0

	for _, page := range pages {
		// 跳过删除的页面
		if page.Operation == models.PageOpDelete {
			continue
		}

		// 根据decision选择HTML版本
		var html string
		switch page.Decision {
		case "approve":
			// 采用AI生成版本：优先generated_html，回退到final_html
			html = page.GeneratedHTML
			if html == "" {
				html = page.FinalHTML
			}
		case "reject":
			// 保留原版：优先original_html，回退到final_html
			html = page.OriginalHTML
			if html == "" {
				html = page.FinalHTML
			}
		case "edit":
			// 使用编辑后版本：优先final_html
			html = page.FinalHTML
			if html == "" {
				html = page.GeneratedHTML
			}
		default:
			// pending或其他状态：取final_html → generated_html → original_html
			html = page.FinalHTML
			if html == "" {
				html = page.GeneratedHTML
			}
			if html == "" {
				html = page.OriginalHTML
			}
		}

		// 只收集有内容的页面
		if len(html) > 100 {
			htmlParts = append(htmlParts,
				fmt.Sprintf("═══ 【P%02d %s】 ═══", page.PageNumber, page.PageTitle),
				html,
			)
			validCount++
		}
	}

	if validCount == 0 {
		return "", 0, ErrVerifyNoValidHTML
	}

	finalContent := strings.Join(htmlParts, "\n\n")
	return finalContent, validCount, nil
}

// ==================== P4.6-4 批量验收+夜间定时任务 ====================

// BatchVerifyResult 批量验收结果（返回给前端/日志）
type BatchVerifyResult struct {
	TotalFound   int      `json:"total_found"`   // 找到的finalized Pipeline数量
	StartedIDs   []string `json:"started_ids"`   // 已启动验收的Pipeline ID列表
	SkippedIDs   []string `json:"skipped_ids"`   // 跳过的Pipeline ID列表（状态已变化等）
	ErrorMessage string   `json:"error_message"` // 整体错误信息（如有）
}

// BatchVerify 批量触发验收：扫描所有finalized状态的Pipeline，逐个触发验收
// P4.6-4新增：手动批量验收接口(POST /pipelines/batch-verify)和夜间定时任务共用
// 每个Pipeline的验收在独立goroutine中执行，不阻塞批量接口返回
func (s *PipelineService) BatchVerify() (*BatchVerifyResult, error) {
	result := &BatchVerifyResult{}

	// 1. 查询所有finalized状态的Pipeline
	ids, err := repository.ListFinalizedPipelineIDs()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("查询finalized Pipeline失败: %s", err.Error())
		return result, err
	}
	result.TotalFound = len(ids)

	if len(ids) == 0 {
		return result, nil
	}

	// 2. 逐个检查并启动验收（每个在独立goroutine中异步执行）
	for _, id := range ids {
		// 再次确认状态仍为finalized（防止并发修改）
		pipeline, pErr := repository.GetPipelineByID(id)
		if pErr != nil || pipeline.Status != models.PipelineStatusFinalized {
			result.SkippedIDs = append(result.SkippedIDs, id)
			continue
		}

		result.StartedIDs = append(result.StartedIDs, id)

		// 异步执行验收（复用VerifyPipeline方法，内部会处理verify_failed→2审等逻辑）
		go s.VerifyPipeline(id)
	}

	return result, nil
}

// StartNightlyVerifyScheduler 启动夜间批量验收定时任务
// P4.6-4新增：在main.go中调用，每晚凌晨2:00（UTC+8）扫描finalized Pipeline批量验收
// 使用goroutine + time.Timer实现，无需外部cron依赖
func (s *PipelineService) StartNightlyVerifyScheduler() {
	go func() {
		for {
			// 计算距离下一个凌晨2:00的时间间隔
			now := time.Now()
			// 构造今天凌晨2:00（东八区，服务器本地时区）
			next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
			// 如果当前时间已过凌晨2:00，则设为明天凌晨2:00
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			waitDuration := next.Sub(now)

			fmt.Printf("[夜间验收] 下次执行时间: %s（等待 %s）\n", next.Format("2006-01-02 15:04:05"), waitDuration)

			// 等待到目标时间
			timer := time.NewTimer(waitDuration)
			<-timer.C

			// 执行批量验收
			fmt.Printf("[夜间验收] %s 开始执行批量验收...\n", time.Now().Format("2006-01-02 15:04:05"))
			result, err := s.BatchVerify()
			if err != nil {
				fmt.Printf("[夜间验收] 执行失败: %s\n", err.Error())
			} else {
				fmt.Printf("[夜间验收] 执行完成: 找到%d个finalized Pipeline，已启动%d个验收，跳过%d个\n",
					result.TotalFound, len(result.StartedIDs), len(result.SkippedIDs))
			}
		}
	}()
}
