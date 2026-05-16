package services

// ==================== verify_service.go — 验收执行 + 2审自动流程 ====================
//
// v63改造：
//   1. executeVerify 从单轮评估改为多轮评估（复用PipelineConfig.EvalRounds）
//      多轮取均值判定，减少AI评分波动误判
//   2. verify无论通过与否都写入定稿索引（savePipelineIndex）
//   3. verify未通过时，把每轮评分写入eval_rounds表（review_round=2）
//      供二审meta直接读取，executeMeta零改动
//   4. startRetrialPipeline 跳过evaluator，直接从meta开始
//      因为verify已经做了评估（用的是同一个定稿索引），重跑evaluator纯浪费
//
// v65-bugfix（2处关键修复）：
//   修复1: VerifyPipeline 的passed判定从字符串匹配改为JSON解析
//          原因: strings.Contains("passed\":true")缺少空格，Go json.Marshal输出"passed": true有空格
//          导致所有verify结果都被误判为未通过，包括实际通过的也触发了2审
//   修复2: writeVerifyScoresToEvalRounds 移到 startRetrialPipeline 中 UpdatePipelineReviewRound(2) 之后调用
//          原因: 原来在review_round=1时就写入eval_rounds，2审meta查review_round=2找不到数据
//
// v68变更：
//   1. 高分早停pipeline快速验收（isEarlyStopPipeline检测→跳过索引生成和评估→直接passed）
//   2. v68-bugfix：修复executeVerify中高分早停代码块被重复粘贴两次的问题
//   3. v68-bugfix：修复文件末尾isEarlyStopPipeline方法重复定义（缺少函数体）的编译错误
//
// 流程说明：
//   无论是evaluator高分早停还是正常流程，finalized后都需要经过验收(verify)
//   verify通过 → verified（等待人工确认publish归档）
//   verify未通过 + round=1 → 自动触发2审
//   verify未通过 + round=2 → needs_human（人工干预）

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/logger"
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
	ErrVerifyAllRoundsFailed  = fmt.Errorf("验收评估所有轮次均失败")
	ErrRetrialResetFailed     = fmt.Errorf("2审重置步骤失败")
	ErrRetrialExecFailed      = fmt.Errorf("2审执行流程失败")
)

// ==================== VerifyPipeline 入口方法 ====================

// VerifyPipeline 手动触发验收流程
// 所有finalized的pipeline都走验收，包括高分早停的pipeline
// verify通过 → verified（等待人工confirm publish归档）
// verify未通过 + round=1 → 自动触发2审
// verify未通过 + round=2 → needs_human
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

	if verifyStep.StepData == "" || verifyStep.StepData == "null" {
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify, "验收步骤输出数据为空")
		return s.GetPipelineDetail(pipelineID)
	}

	// ---- v63：verify无论通过与否，都先写入定稿索引 ----
	if idxErr := s.savePipelineIndex(pipelineID); idxErr != nil {
		verifyLog.Warn("写入pipeline定稿索引失败，不影响verify结果",
			"pipeline_id", pipelineID, "error", idxErr)
	}

	// ---- v65-bugfix：从字符串匹配改为JSON解析判定passed ----
	// 原代码: strings.Contains(verifyStep.StepData, "\"passed\":true")
	// 问题: Go的json.Marshal输出 "passed": true（有空格），字符串匹配"passed\":true"（无空格）永远不匹配
	// 修复: 用json.Unmarshal解析后读取passed字段
	verifyPassed := s.parseVerifyPassed(verifyStep.StepData)

	if verifyPassed {
		// ---- verify通过 → verified，等待人工confirm publish归档 ----
		if err := repository.CompletePipeline(pipelineID, models.PipelineStatusVerified); err != nil {
			return nil, fmt.Errorf("标记验收通过失败: %w", err)
		}
		verifyLog.Info("验收通过，等待人工归档",
			"pipeline_id", pipelineID,
			"status", "verified")
	} else {
		// ---- verify未通过 ----
		if err := repository.CompletePipeline(pipelineID, models.PipelineStatusVerifyFailed); err != nil {
			return nil, fmt.Errorf("标记验收未通过失败: %w", err)
		}

		// v65-bugfix：eval_rounds写入移到startRetrialPipeline中
		// 原来在这里调用writeVerifyScoresToEvalRounds，此时review_round还是1
		// 导致eval_rounds写入review_round=1，2审meta查review_round=2找不到数据
		// 现在改为在startRetrialPipeline中UpdatePipelineReviewRound(2)之后再写入

		if pipeline.ReviewRound <= 1 {
			// 1审verify未通过 → 自动触发2审
			if s.engine != nil {
				capturedStepData := verifyStep.StepData
				task := &EngineTask{
					Type:       TaskTypeRetrial,
					PipelineID: pipelineID,
					ExecFunc: func() error {
						s.startRetrialPipeline(pipelineID, capturedStepData)
						p, pErr := repository.GetPipelineByID(pipelineID)
						if pErr != nil {
							return fmt.Errorf("2审执行后读取Pipeline状态失败: %w", pErr)
						}
						if p.Status == models.PipelineStatusFailed {
							return fmt.Errorf("2审执行失败: %s", p.ErrorMessage)
						}
						return nil
					},
				}
				if !s.engine.Submit(task) {
					verifyLog.Warn("2审任务提交失败：队列已满", "pipeline_id", pipelineID)
				}
			} else {
				capturedStepData := verifyStep.StepData
				go s.startRetrialPipeline(pipelineID, capturedStepData)
			}
		} else {
			// 2审verify未通过 → 需要人工干预
			_ = repository.UpdatePipelineStatus(pipelineID, models.StepVerify, models.PipelineStatusNeedsHuman)
			verifyLog.Info("2审验收未通过，需要人工干预",
				"pipeline_id", pipelineID,
				"status", "needs_human")
		}
	}

	return s.GetPipelineDetail(pipelineID)
}

// parseVerifyPassed 从verify步骤的step_data JSON中解析passed字段
// v65-bugfix新增：替代原来的strings.Contains字符串匹配
func (s *PipelineService) parseVerifyPassed(stepData string) bool {
	if stepData == "" || stepData == "null" {
		return false
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(stepData), &data); err != nil {
		verifyLog.Warn("解析verify步骤数据失败，降级为未通过", "error", err)
		return false
	}
	passed, ok := data["passed"].(bool)
	return ok && passed
}

// ==================== 2审自动流程 ====================

// startRetrialPipeline 触发二审流程
// v63改造：跳过evaluator，直接从meta开始
// 原因：verify已经做了多轮评估（用的是定稿索引），评分已写入eval_rounds(review_round=2)
//       重跑evaluator完全是浪费（相同索引+相同提示词=几乎相同的分数）
//       meta读取eval_rounds(review_round=2)即可得到verify的评估结果
//
// v65-bugfix：新增verifyStepData参数，在UpdatePipelineReviewRound(2)之后再写入eval_rounds
//       原来writeVerifyScoresToEvalRounds在VerifyPipeline中调用，此时review_round还是1
//       导致eval_rounds写入review_round=1，2审meta查review_round=2找不到数据
func (s *PipelineService) startRetrialPipeline(pipelineID string, verifyStepData string) {
	if err := repository.UpdatePipelineReviewRound(pipelineID, 2); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepVerify,
			fmt.Sprintf("2审启动失败: 更新review_round失败: %s", err.Error()))
		return
	}

	// v65-bugfix：在review_round=2之后再写入eval_rounds
	// 这样CreateEvalRound读取pipelines.review_round=2，写入的eval_rounds.review_round=2
	// 2审的executeMeta通过GetEvalRoundsByPipelineID查review_round=2就能读到数据
	if verifyStepData != "" {
		s.writeVerifyScoresToEvalRounds(pipelineID, verifyStepData)
	}

	// v63改造：不再重置evaluator（跳过），只重置meta及后续步骤
	resetSteps := []string{
		models.StepMeta,
		models.StepTranslator,
		models.StepGenerator,
		models.StepReview,
		models.StepVerify,
	}
	if err := repository.ResetStepsForRetrial(pipelineID, resetSteps); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepMeta,
			fmt.Sprintf("2审启动失败: 重置步骤失败: %s", err.Error()))
		return
	}

	// v63改造：二审也需要evaluator步骤记录存在（即使跳过执行）
	// 标记evaluator为done，step_data写入verify的评估结果摘要
	s.markEvaluatorDoneForRetrial(pipelineID)

	// 直接从meta开始
	if err := repository.UpdatePipelineStatus(pipelineID, models.StepMeta, models.PipelineStatusRunning); err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepMeta,
			fmt.Sprintf("2审启动失败: 更新状态失败: %s", err.Error()))
		return
	}

	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		_ = repository.UpdatePipelineError(pipelineID, models.StepMeta,
			fmt.Sprintf("2审启动失败: 读取Pipeline失败: %s", err.Error()))
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

// ==================== executeVerify 核心执行方法（v63多轮评估）====================

func (s *PipelineService) executeVerify(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepVerify

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动verify步骤失败: %w", err)
	}

	pCfg := models.ParsePipelineConfig(pipeline.Config)
	threshold := pCfg.Threshold
	evalRounds := pCfg.EvalRounds // v63：复用Pipeline配置的评估轮次数（默认3）

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
		s.cfg.AESKey, "evaluator",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("verify: 获取AI配置失败: %s", err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// ==================== v68：高分早停pipeline快速验收 ====================
	// 如果evaluator已经高分早停（所有页面为keep），验收时跳过重新评估直接通过
	// 原理：高分早停意味着原始课件质量已达标，页面全部保留无修改，
	//       重新跑一遍索引生成+评估只会浪费时间和token，结果几乎相同
	if s.isEarlyStopPipeline(pipeline) {
		verifyLog.Info("高分早停pipeline快速验收通过",
			"pipeline_id", pipeline.ID, "review_round", pipeline.ReviewRound)
		avgScore := s.getEvalAvgScore(pipeline.ID)
		quickResult := &models.VerifyResult{
			GeneratedIndex:  "(高分早停快速验收：使用原始课程索引)",
			TotalEvalRounds: 0,
			DoneEvalRounds:  0,
			EvalScore:       avgScore,
			Passed:          true,
			ReviewRound:     pipeline.ReviewRound,
			ModelUsed:       "quick_pass",
			TokensUsed:      0,
			LatencyMs:       time.Since(startTime).Milliseconds(),
		}
		durationMs := time.Since(startTime).Milliseconds()
		if err := repository.CompleteStep(
			pipeline.ID, stepName, durationMs,
			quickResult.ToJSON(), "quick_pass", 0,
		); err != nil {
			return fmt.Errorf("保存verify快速通过结果失败: %w", err)
		}
		return nil
	}

	// ==================== 第一步：用定稿HTML生成新索引 ====================
	indexGenUserParts := []string{
		"【课程定位】", scannerLocationJSON, "",
		fmt.Sprintf("【最终课件HTML（共%d页）】", pageCount),
		"以下是经过人工审核定稿后的最终课件HTML内容，请按照要求压缩为TE-DNA课程索引+模块索引。",
		"", finalHTMLContent, "",
		"禁止输出<thinking>标签或任何思维过程标记。",
	}
	indexGenUserPrompt := strings.Join(indexGenUserParts, "\n")

	var totalTokens int
	var lastModelUsed string

	indexCallResult, indexCallErr := s.callAIWithSemaphore(aiCfg, promptG.Content, indexGenUserPrompt, pipeline.ID)
	if indexCallErr != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("%s: %s", ErrVerifyIndexGenFailed.Error(), indexCallErr.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	generatedIndex := indexCallResult.Content
	totalTokens += indexCallResult.TokensUsed
	lastModelUsed = indexCallResult.ModelUsed

	if len(generatedIndex) < 200 {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("%s (输出长度: %d字符)", ErrVerifyIndexTooShort.Error(), len(generatedIndex))
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// ==================== 第二步：多轮评估（v63核心改造） ====================
	evalUserParts := []string{
		"【课程定位】", scannerLocationJSON, "",
		"【待评估索引】", generatedIndex, "",
		"【TE-DNA解压缩字典】", dict.Content,
	}
	if abilityTable != nil && len(abilityTable.Content) > 20 {
		evalUserParts = append(evalUserParts, "", "【能力定位表】", abilityTable.Content)
	}

	// v90机制优化2：验收评估注入一审修改方案上下文
	// 原因：验收评估完全独立评分，不知道一审做了什么修改，
	// 导致验收评分与元评估仲裁差异大（元评估高分→验收低分→2审大幅修改→反复不通过）
	// 修复：注入一审meta修改方案摘要，让验收评估在相同上下文下评分，提高一致性
	if pipeline.ReviewRound >= 1 {
		firstRoundCtx := buildFirstRoundContextForEval(pipeline.ID)
		if firstRoundCtx != "" {
			evalUserParts = append(evalUserParts, "", firstRoundCtx)
		}
	}

	evalUserParts = append(evalUserParts, "", "禁止输出<thinking>标签或任何思维过程标记。")
	evalUserPrompt := strings.Join(evalUserParts, "\n")

	var roundResults []models.VerifyEvalRound
	var doneCount int

	for i := 1; i <= evalRounds; i++ {
		evalCallResult, evalCallErr := s.callAIWithSemaphore(aiCfg, promptB.Content, evalUserPrompt, pipeline.ID)
		if evalCallErr != nil {
			// 单轮失败不终止，继续下一轮
			verifyLog.Warn("verify评估轮次失败",
				"pipeline_id", pipeline.ID, "round", i, "error", evalCallErr)
			continue
		}

		evalOutput := evalCallResult.Content
		totalTokens += evalCallResult.TokensUsed
		lastModelUsed = evalCallResult.ModelUsed

		scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4, hardConstraint, grade, parseOk := extractEvalScores(evalOutput)
		if !parseOk || scoreTotal < 0 {
			verifyLog.Warn("verify评估轮次评分提取失败",
				"pipeline_id", pipeline.ID, "round", i)
			continue
		}

		roundResults = append(roundResults, models.VerifyEvalRound{
			RoundNumber: i,
			ScoreTotal:  scoreTotal,
			ScoreE1:     scoreE1,
			ScoreE2:     scoreE2,
			ScoreE3:     scoreE3,
			ScoreE4:     scoreE4,
			Hard:        hardConstraint,
			Grade:       grade,
			Output:      truncate(evalOutput, 50000),
			ModelUsed:   evalCallResult.ModelUsed,
			TokensUsed:  evalCallResult.TokensUsed,
		})
		doneCount++
	}

	// 所有轮次都失败
	if doneCount == 0 {
		durationMs := time.Since(startTime).Milliseconds()
		verifyResult := &models.VerifyResult{
			GeneratedIndex:  truncate(generatedIndex, 50000),
			TotalEvalRounds: evalRounds,
			DoneEvalRounds:  0,
			ReviewRound:     pipeline.ReviewRound,
			ModelUsed:       lastModelUsed,
			TokensUsed:      totalTokens,
			LatencyMs:       time.Since(startTime).Milliseconds(),
		}
		s.saveStepData(pipeline.ID, stepName, verifyResult.ToJSON())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrVerifyAllRoundsFailed.Error())
		return ErrVerifyAllRoundsFailed
	}

	// ==================== 第三步：计算均分并判定 ====================
	var sumTotal, sumE1, sumE2, sumE3, sumE4 float64
	for _, r := range roundResults {
		sumTotal += r.ScoreTotal
		sumE1 += r.ScoreE1
		sumE2 += r.ScoreE2
		sumE3 += r.ScoreE3
		sumE4 += r.ScoreE4
	}
	n := float64(doneCount)
	avgTotal := math.Round(sumTotal/n*10) / 10
	avgE1 := math.Round(sumE1/n*10) / 10
	avgE2 := math.Round(sumE2/n*10) / 10
	avgE3 := math.Round(sumE3/n*10) / 10
	avgE4 := math.Round(sumE4/n*10) / 10

	passed := avgTotal >= threshold

	// 最后一轮的输出作为eval_output（向后兼容）
	lastRoundOutput := roundResults[len(roundResults)-1].Output

	verifyResult := &models.VerifyResult{
		GeneratedIndex:  truncate(generatedIndex, 50000),
		EvalRoundScores: roundResults,
		TotalEvalRounds: evalRounds,
		DoneEvalRounds:  doneCount,
		EvalScore:       avgTotal,
		EvalOutput:      lastRoundOutput,
		EvalE1:          avgE1,
		EvalE2:          avgE2,
		EvalE3:          avgE3,
		EvalE4:          avgE4,
		Passed:          passed,
		ReviewRound:     pipeline.ReviewRound,
		ModelUsed:       lastModelUsed,
		TokensUsed:      totalTokens,
		LatencyMs:       time.Since(startTime).Milliseconds(),
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

// ==================== verify评分写入eval_rounds ====================

// writeVerifyScoresToEvalRounds 将verify的多轮评分写入eval_rounds表
// v65-bugfix：现在在startRetrialPipeline中UpdatePipelineReviewRound(2)之后调用
// 这样CreateEvalRound从pipelines.review_round读到2，写入的eval_rounds.review_round=2
// 2审的executeMeta通过GetEvalRoundsByPipelineID查review_round=2就能读到这些记录
func (s *PipelineService) writeVerifyScoresToEvalRounds(pipelineID string, stepData string) {
	var verifyData models.VerifyResult
	if err := json.Unmarshal([]byte(stepData), &verifyData); err != nil {
		verifyLog.Warn("解析verify结果写入eval_rounds失败", "pipeline_id", pipelineID, "error", err)
		return
	}

	// 如果有多轮评分数据（v63新格式），逐轮写入
	if len(verifyData.EvalRoundScores) > 0 {
		for _, round := range verifyData.EvalRoundScores {
			roundRec, err := repository.CreateEvalRound(pipelineID, round.RoundNumber)
			if err != nil {
				verifyLog.Warn("创建verify→eval_round记录失败",
					"pipeline_id", pipelineID, "round", round.RoundNumber, "error", err)
				continue
			}
			dimMap := map[string]interface{}{"hard_constraint": round.Hard, "grade": round.Grade}
			dimJSON, _ := json.Marshal(dimMap)
			_ = repository.CompleteEvalRound(
				roundRec.ID, truncate(round.Output, 50000),
				round.ScoreTotal, round.ScoreE1, round.ScoreE2, round.ScoreE3, round.ScoreE4,
				string(dimJSON), round.ModelUsed, round.TokensUsed,
			)
		}
		verifyLog.Info("verify多轮评分已写入eval_rounds",
			"pipeline_id", pipelineID, "rounds", len(verifyData.EvalRoundScores))
		return
	}

	// 兜底：旧格式（单轮），写入一条eval_round记录
	roundRec, err := repository.CreateEvalRound(pipelineID, 1)
	if err != nil {
		verifyLog.Warn("创建verify→eval_round记录失败(兼容)", "pipeline_id", pipelineID, "error", err)
		return
	}
	_ = repository.CompleteEvalRound(
		roundRec.ID, truncate(verifyData.EvalOutput, 50000),
		verifyData.EvalScore, verifyData.EvalE1, verifyData.EvalE2, verifyData.EvalE3, verifyData.EvalE4,
		"{}", verifyData.ModelUsed, verifyData.TokensUsed,
	)
	verifyLog.Info("verify单轮评分已写入eval_rounds(兼容模式)", "pipeline_id", pipelineID)
}

// markEvaluatorDoneForRetrial 二审时标记evaluator步骤为done（跳过执行）
// 目的：让Pipeline的步骤列表显示evaluator已完成，前端不会显示异常
// step_data写入verify评估的摘要信息
func (s *PipelineService) markEvaluatorDoneForRetrial(pipelineID string) {
	// 读取verify步骤的评分摘要
	// 使用GetStepByName按review_round DESC取最新的
	// 此时verify的round=2记录刚被ResetStepsForRetrial重置为pending
	// 所以GetStepByName会取到round=1的done记录，这是正确的
	verifyStep, err := repository.GetStepByName(pipelineID, models.StepVerify)
	if err != nil {
		return
	}

	var verifyData models.VerifyResult
	if err := json.Unmarshal([]byte(verifyStep.StepData), &verifyData); err != nil {
		return
	}

	// 构造evaluator的step_data（与正常evaluator输出格式兼容）
	evalResult := &models.EvaluatorResult{
		TotalRounds:  verifyData.DoneEvalRounds,
		DoneRounds:   verifyData.DoneEvalRounds,
		FailedRounds: verifyData.TotalEvalRounds - verifyData.DoneEvalRounds,
		AvgTotal:     verifyData.EvalScore,
		AvgE1:        verifyData.EvalE1,
		AvgE2:        verifyData.EvalE2,
		AvgE3:        verifyData.EvalE3,
		AvgE4:        verifyData.EvalE4,
		ModelUsed:    verifyData.ModelUsed,
	}

	// 启动并立即完成evaluator步骤
	_ = repository.StartStep(pipelineID, models.StepEvaluator)
	_ = repository.CompleteStep(pipelineID, models.StepEvaluator, 0,
		evalResult.ToJSON(), verifyData.ModelUsed, 0)
}

