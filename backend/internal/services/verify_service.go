package services

// ==================== P4.6-2 验收执行逻辑 + P4.6-3 2审自动流程 ====================
// Phase8修复V-02：夜间定时任务时区改为显式 Asia/Shanghai
//   原版：now.Location() 依赖服务器本地时区，若服务器时区变更则2:00执行时间漂移
//   修复后：time.LoadLocation("Asia/Shanghai") 显式指定，与业务时区强绑定

import (
	"fmt"
	"tedna/internal/logger"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 验收错误常量 ====================

var verifyLog = logger.WithModule("verify")

var (
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
	ErrRetrialResetFailed     = fmt.Errorf("2审重置步骤失败")
	ErrRetrialExecFailed      = fmt.Errorf("2审执行流程失败")
)

// ==================== VerifyPipeline 入口方法 ====================

// VerifyPipeline 手动触发验收流程
func (s *PipelineService) VerifyPipeline(pipelineID string) (*models.PipelineDetailResponse, error) {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	if pipeline.Status != models.PipelineStatusFinalized {
		return nil, ErrVerifyNotFinalized
	}

	if err := repository.UpdatePipelineStatus(pipelineID, models.StepVerify, models.PipelineStatusRunning); err != nil {
		return nil, fmt.Errorf("更新Pipeline状态失败: %w", err)
	}

	pipeline, err = repository.GetPipelineByID(pipelineID)
	if err != nil {
		return s.GetPipelineDetail(pipelineID)
	}

	verifyErr := s.executeVerify(pipeline)
	if verifyErr != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify, verifyErr.Error())
		return s.GetPipelineDetail(pipelineID)
	}

	verifyStep, stepErr := repository.GetStepByName(pipelineID, models.StepVerify)
	if stepErr != nil || verifyStep.Status != models.StepStatusDone {
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify, "验收步骤未正确完成")
		return s.GetPipelineDetail(pipelineID)
	}

	if verifyStep.StepData != "" && verifyStep.StepData != "null" {
		if strings.Contains(verifyStep.StepData, "\"passed\":true") {
			if err := repository.CompletePipeline(pipelineID, models.PipelineStatusVerified); err != nil {
				return nil, fmt.Errorf("标记验收通过失败: %w", err)
			}
		} else {
			if err := repository.CompletePipeline(pipelineID, models.PipelineStatusVerifyFailed); err != nil {
				return nil, fmt.Errorf("标记验收未通过失败: %w", err)
			}

			if pipeline.ReviewRound <= 1 {
				if s.engine != nil {
					task := &EngineTask{
						Type:       TaskTypeRetrial,
						PipelineID: pipelineID,
						ExecFunc:   func() { s.startRetrialPipeline(pipelineID) },
					}
					if !s.engine.Submit(task) {
						verifyLog.Warn("2审任务提交失败：队列已满", "pipeline_id", pipelineID)
					}
				} else {
					go s.startRetrialPipeline(pipelineID)
				}
			} else {
				_ = repository.UpdatePipelineStatus(pipelineID, models.StepVerify, models.PipelineStatusNeedsHuman)
			}
		}
	} else {
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify, "验收步骤输出数据为空")
		return s.GetPipelineDetail(pipelineID)
	}

	return s.GetPipelineDetail(pipelineID)
}

// ==================== P4.6-3 2审自动流程 ====================

func (s *PipelineService) startRetrialPipeline(pipelineID string) {
	if err := repository.UpdatePipelineReviewRound(pipelineID, 2); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify,
			fmt.Sprintf("2审启动失败: 更新review_round失败: %s", err.Error()))
		return
	}

	_ = repository.DeleteEvalRoundsByPipelineID(pipelineID)
	_ = repository.DeleteGeneratedPagesByPipelineID(pipelineID)

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

	if err := repository.UpdatePipelineStatus(pipelineID, models.StepEvaluator, models.PipelineStatusRunning); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepEvaluator,
			fmt.Sprintf("2审启动失败: 更新状态失败: %s", err.Error()))
		return
	}

	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepEvaluator,
			fmt.Sprintf("2审启动失败: 读取Pipeline失败: %s", err.Error()))
		return
	}

	pCfg := models.ParsePipelineConfig(pipeline.Config)
	originalEvalRounds := pCfg.EvalRounds
	pCfg.EvalRounds = 2
	pipeline.Config = pCfg.ToJSON()

	evalErr := s.executeEvaluator(pipeline)

	pCfg.EvalRounds = originalEvalRounds
	pipeline.Config = pCfg.ToJSON()

	if evalErr != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepEvaluator,
			fmt.Sprintf("2审Evaluator失败: %s", evalErr.Error()))
		return
	}

	if err := repository.UpdatePipelineStatus(pipelineID, models.StepMeta, models.PipelineStatusRunning); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepMeta,
			fmt.Sprintf("2审推进到Meta失败: %s", err.Error()))
		return
	}

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

	if err := repository.UpdatePipelineStatus(pipelineID, models.StepReview, models.PipelineStatusReviewQueue); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepReview,
			fmt.Sprintf("2审推进到Review失败: %s", err.Error()))
		return
	}
}

// ==================== executeVerify 核心执行方法 ====================

func (s *PipelineService) executeVerify(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepVerify

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动verify步骤失败: %w", err)
	}

	pCfg := models.ParsePipelineConfig(pipeline.Config)
	threshold := pCfg.Threshold

	promptG, err := repository.GetCurrentPromptByKey("prompt_g")
	if err != nil || promptG == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrVerifyPromptGMissing.Error())
		return ErrVerifyPromptGMissing
	}

	promptB, err := repository.GetCurrentPromptByKey("prompt_b")
	if err != nil || promptB == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrVerifyPromptBMissing.Error())
		return ErrVerifyPromptBMissing
	}

	dict, err := repository.GetCurrentPromptByKey("dict")
	if err != nil || dict == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrVerifyDictMissing.Error())
		return ErrVerifyDictMissing
	}

	abilityTable, _ := repository.GetCurrentPromptByKey("ability_table")

	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrVerifyScannerNotDone.Error())
		return ErrVerifyScannerNotDone
	}
	scannerLocationJSON := extractScannerParsed(scannerStep.StepData)

	finalHTMLContent, pageCount, err := s.collectFinalHTML(pipeline.ID)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, err.Error())
		return err
	}

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

	var totalTokens int
	var lastModelUsed string

	indexCallResult, indexCallErr := s.callAIWithSemaphore(aiCfg, promptG.Content, indexGenUserPrompt)
	if indexCallErr != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("%s: %s", ErrVerifyIndexGenFailed.Error(), indexCallErr.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	generatedIndex := indexCallResult.Content
	totalTokens += indexCallResult.TokensUsed
	lastModelUsed = indexCallResult.ModelUsed

	if len(generatedIndex) < 200 {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("%s (输出长度: %d字符)", ErrVerifyIndexTooShort.Error(), len(generatedIndex))
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

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
	if abilityTable != nil && len(abilityTable.Content) > 20 {
		evalUserParts = append(evalUserParts, "", "【能力定位表】", abilityTable.Content)
	}
	evalUserParts = append(evalUserParts, "", "禁止输出<thinking>标签或任何思维过程标记。")
	evalUserPrompt := strings.Join(evalUserParts, "\n")

	evalCallResult, evalCallErr := s.callAIWithSemaphore(aiCfg, promptB.Content, evalUserPrompt)
	if evalCallErr != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("%s: %s", ErrVerifyEvalFailed.Error(), evalCallErr.Error())
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

	scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4, _, _, parseOk := extractEvalScores(evalOutput)
	if !parseOk || scoreTotal < 0 {
		durationMs := time.Since(startTime).Milliseconds()
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

func (s *PipelineService) collectFinalHTML(pipelineID string) (string, int, error) {
	pages, err := repository.GetGeneratedPagesWithHTML(pipelineID)
	if err != nil {
		return "", 0, fmt.Errorf("获取生成页面失败: %w", err)
	}
	if len(pages) == 0 {
		return "", 0, ErrVerifyNoPages
	}

	var htmlParts []string
	validCount := 0

	for _, page := range pages {
		if page.Operation == models.PageOpDelete {
			continue
		}

		var html string
		switch page.Decision {
		case "approve":
			html = page.GeneratedHTML
			if html == "" {
				html = page.FinalHTML
			}
		case "reject":
			html = page.OriginalHTML
			if html == "" {
				html = page.FinalHTML
			}
		case "edit":
			html = page.FinalHTML
			if html == "" {
				html = page.GeneratedHTML
			}
		default:
			html = page.FinalHTML
			if html == "" {
				html = page.GeneratedHTML
			}
			if html == "" {
				html = page.OriginalHTML
			}
		}

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

// ==================== 批量验收+夜间定时任务 ====================

// BatchVerifyResult 批量验收结果
type BatchVerifyResult struct {
	TotalFound   int      `json:"total_found"`
	StartedIDs   []string `json:"started_ids"`
	SkippedIDs   []string `json:"skipped_ids"`
	ErrorMessage string   `json:"error_message"`
}

// BatchVerify 批量触发验收
func (s *PipelineService) BatchVerify() (*BatchVerifyResult, error) {
	result := &BatchVerifyResult{}

	ids, err := repository.ListFinalizedPipelineIDs()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("查询finalized Pipeline失败: %s", err.Error())
		return result, err
	}
	result.TotalFound = len(ids)

	if len(ids) == 0 {
		return result, nil
	}

	for _, id := range ids {
		pipeline, pErr := repository.GetPipelineByID(id)
		if pErr != nil || pipeline.Status != models.PipelineStatusFinalized {
			result.SkippedIDs = append(result.SkippedIDs, id)
			continue
		}

		capturedID := id
		if s.engine != nil {
			task := &EngineTask{
				Type:       TaskTypeVerify,
				PipelineID: capturedID,
				ExecFunc:   func() { s.VerifyPipeline(capturedID) },
			}
			if s.engine.Submit(task) {
				result.StartedIDs = append(result.StartedIDs, capturedID)
			} else {
				verifyLog.Warn("批量验收任务提交失败：队列已满", "pipeline_id", capturedID)
				result.SkippedIDs = append(result.SkippedIDs, capturedID)
			}
		} else {
			go s.VerifyPipeline(capturedID)
			result.StartedIDs = append(result.StartedIDs, capturedID)
		}
	}

	return result, nil
}

// StartNightlyVerifyScheduler 启动夜间批量验收定时任务
// 修复V-02：显式加载 Asia/Shanghai 时区，不再依赖服务器本地时区设置
// 原版：time.Date(..., now.Location()) 若服务器时区非Asia/Shanghai则2:00计算错误
// 修复后：强制使用 Asia/Shanghai，与业务时区（北京时间）强绑定
func (s *PipelineService) StartNightlyVerifyScheduler() {
	go func() {
		// 显式加载 Asia/Shanghai 时区（北京时间）
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			// 时区加载失败时降级使用UTC+8固定偏移，保证服务不中断
			verifyLog.Warn("加载Asia/Shanghai时区失败，降级为UTC+8", "error", err)
			loc = time.FixedZone("CST", 8*3600)
		}

		for {
			// 用 Asia/Shanghai 时区计算当前时间和下次2:00
			now := time.Now().In(loc)
			next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, loc)
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			waitDuration := next.Sub(now)

			verifyLog.Info("夜间验收调度器等待中", "next_run", next.Format("2006-01-02 15:04:05"), "wait_duration", waitDuration.String())

			timer := time.NewTimer(waitDuration)
			<-timer.C

			fmt.Printf("[夜间验收] %s 开始执行批量验收...\n",
				time.Now().In(loc).Format("2006-01-02 15:04:05"))
			batchResult, batchErr := s.BatchVerify()
			if batchErr != nil {
				verifyLog.Error("夜间验收执行失败", "error", batchErr)
			} else {
				verifyLog.Info("夜间验收执行完成", "total_found", batchResult.TotalFound, "started", len(batchResult.StartedIDs), "skipped", len(batchResult.SkippedIDs))
			}
		}
	}()
}
