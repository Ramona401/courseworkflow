package services

// ==================== P4.6-2 验收执行逻辑 ====================
// 实现 executeVerify 方法：收集最终HTML → 调索引生成器(prompt_g) → 调Evaluator(prompt_b) 1轮 → 判定
// 新增 VerifyPipeline 入口方法：手动触发验收（POST /pipelines/{id}/verify）

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
)

// ==================== VerifyPipeline 入口方法 ====================

// VerifyPipeline 手动触发验收流程
// P4.6-2新增：POST /pipelines/{id}/verify 的业务逻辑入口
// 前置条件：Pipeline状态必须是 finalized
// 流程：更新状态为running → 执行verify步骤 → 根据评分设置verified/verify_failed
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
	var verifyResult models.VerifyResult
	if verifyStep.StepData != "" && verifyStep.StepData != "null" {
		// 手动提取passed字段（避免引入encoding/json，该包在pipeline_service.go中已导入）
		// 使用简单的字符串匹配方式
		if strings.Contains(verifyStep.StepData, `"passed":true`) {
			// 验收通过 → verified
			if err := repository.CompletePipeline(pipelineID, models.PipelineStatusVerified); err != nil {
				return nil, fmt.Errorf("标记验收通过失败: %w", err)
			}
		} else {
			// 验收未通过 → verify_failed
			if err := repository.CompletePipeline(pipelineID, models.PipelineStatusVerifyFailed); err != nil {
				return nil, fmt.Errorf("标记验收未通过失败: %w", err)
			}
		}
	} else {
		// step_data为空，异常情况
		_ = verifyResult // 消除unused警告
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify, "验收步骤输出数据为空")
		return s.GetPipelineDetail(pipelineID)
	}

	return s.GetPipelineDetail(pipelineID)
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
