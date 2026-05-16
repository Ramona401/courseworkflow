package services

// pipeline_eval_meta.go — Evaluator + Meta 步骤执行
//
// 职责：
//   - executeEvaluator 评估打分步骤（多轮独立评估+均分计算+方差检测）
//   - executeMeta 元评估仲裁步骤（多次重试+评分提取+阈值判定）
//   - extractMetaScores Meta评分提取
//   - EvalRoundDetail 类型定义 + GetEvalRounds 查询方法
//   - buildFirstRoundContext 构建一审上下文摘要（v68新增）
//
// v68变更：二审注入一审上下文
//   - executeEvaluator: 当review_round>=2时，读取一审meta的step_data（修改方案摘要），
//     注入到evaluator的userPrompt中，让AI理解一审为什么做了这些修改
//   - executeMeta: 当review_round>=2时，读取一审translator的最终翻译方案和reviewer审核意见，
//     注入到meta的userPrompt中，让meta延续一审方向而非推倒重来
//   - 新增buildFirstRoundContext辅助函数，从一审step_data中提取关键信息构建上下文

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== Evaluator 步骤 ====================

// executeEvaluator 执行评估打分步骤
// 使用配置的轮次数独立调用AI评估，计算均分和方差
// 一轮读原始索引(course_indexes)，二审读定稿索引(pipeline_indexes)
// v68改进：二审时注入一审meta修改方案摘要，避免评估方向与一审相悖
func (s *PipelineService) executeEvaluator(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepEvaluator

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动evaluator失败: %w", err)
	}

	pCfg := models.ParsePipelineConfig(pipeline.Config)
	totalRounds := pCfg.EvalRounds

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

	abilityTable, _ := repository.GetCurrentPromptByKey("ability_table")

	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("evaluator: 课程 %s 不存在", pipeline.CourseCode)
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// 索引读取：一轮读原始索引(course_indexes)，二审读定稿索引(pipeline_indexes)
	var evalIndexContent string
	if pipeline.ReviewRound >= 2 {
		// 二审：读取上一轮verify生成的定稿索引
		pipelineIdx, pIdxErr := repository.GetPipelineIndex(pipeline.ID)
		if pIdxErr != nil {
			durationMs := time.Since(startTime).Milliseconds()
			errMsg := fmt.Sprintf("evaluator(2审): 定稿索引不存在，请确认一轮verify已完成: %s", pIdxErr.Error())
			_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
			return fmt.Errorf("%s", errMsg)
		}
		evalIndexContent = pipelineIdx.IndexContent
	} else {
		// 一轮：读取原始OSS索引
		tmpIdx, cIdxErr := repository.GetCourseIndex(course.ID)
		if cIdxErr != nil {
			durationMs := time.Since(startTime).Milliseconds()
			errMsg := "evaluator: 课程索引不存在"
			_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
			return fmt.Errorf("%s", errMsg)
		}
		evalIndexContent = tmpIdx.IndexContent
	}

	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrEvalScannerNotDone.Error())
		return ErrEvalScannerNotDone
	}
	scannerLocationJSON := extractScannerParsed(scannerStep.StepData)

	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey, "evaluator",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("evaluator: 获取AI配置失败: %s", err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// 构建用户提示词
	systemPrompt := promptB.Content
	userPromptParts := []string{
		"【课程定位】", scannerLocationJSON, "",
		"【待评估索引】", evalIndexContent, "",
		"【TE-DNA解压缩字典】", dict.Content,
	}
	if abilityTable != nil && len(abilityTable.Content) > 20 {
		userPromptParts = append(userPromptParts, "", "【能力定位表】", abilityTable.Content)
	}

	// v68新增：二审时注入一审修改背景上下文
	// 让evaluator理解一审为什么做了这些修改，避免给出与一审方向相悖的建议
	if pipeline.ReviewRound >= 2 {
		firstRoundCtx := buildFirstRoundContextForEval(pipeline.ID)
		if firstRoundCtx != "" {
			userPromptParts = append(userPromptParts, "", firstRoundCtx)
		}
	}

	userPromptParts = append(userPromptParts, "", "禁止输出<thinking>标签或任何思维过程标记。")
	userPrompt := strings.Join(userPromptParts, "\n")

	_ = repository.DeleteEvalRoundsByPipelineID(pipeline.ID)

	evalResult := &models.EvaluatorResult{TotalRounds: totalRounds}
	var roundScores []float64
	var totalTokens int
	var doneCount, failCount int
	var lastModelUsed string

	for i := 1; i <= totalRounds; i++ {
		roundRec, err := repository.CreateEvalRound(pipeline.ID, i)
		if err != nil {
			failCount++
			continue
		}
		_ = repository.UpdateEvalRoundRunning(roundRec.ID)

		callResult, callErr := s.callAIWithSemaphore(aiCfg, systemPrompt, userPrompt, pipeline.ID)
		if callErr != nil {
			_ = repository.FailEvalRound(roundRec.ID, "", callErr.Error())
			failCount++
			continue
		}

		output := callResult.Content
		lastModelUsed = callResult.ModelUsed
		totalTokens += callResult.TokensUsed

		scoreTotal, scoreE1, scoreE2, scoreE3, scoreE4, hardConstraint, grade, parseOk := extractEvalScores(output)
		if !parseOk || scoreTotal < 0 {
			_ = repository.FailEvalRound(roundRec.ID, truncate(output, 5000), "评分提取失败")
			failCount++
			continue
		}

		dimMap := map[string]interface{}{"hard_constraint": hardConstraint, "grade": grade}
		dimJSON, _ := json.Marshal(dimMap)

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
	if doneCount == 0 {
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrEvalAllRoundsFailed.Error())
		return ErrEvalAllRoundsFailed
	}

	// 计算各维度均分
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

	// 方差计算（≥2轮有效结果时）
	if doneCount >= 2 {
		var sumSqDiff float64
		for _, sc := range roundScores {
			diff := sc - evalResult.AvgTotal
			sumSqDiff += diff * diff
		}
		evalResult.Variance = math.Round(sumSqDiff/n*100) / 100
		evalResult.VarianceWarn = evalResult.Variance > pCfg.VarianceWarn
	}

	if err := repository.CompleteStep(
		pipeline.ID, stepName, durationMs,
		evalResult.ToJSON(), lastModelUsed, totalTokens,
	); err != nil {
		return fmt.Errorf("保存evaluator结果失败: %w", err)
	}
	return nil
}

// ==================== Meta 步骤 ====================

// executeMeta 执行元评估仲裁步骤
// 综合多轮Evaluator结果，产出修改方案+修改后索引
// 支持多次重试直到达到阈值或耗尽重试次数
// v68改进：二审时注入一审translator/reviewer的完整上下文，延续一审修改方向
func (s *PipelineService) executeMeta(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepMeta

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动meta失败: %w", err)
	}

	pCfg := models.ParsePipelineConfig(pipeline.Config)
	threshold := pCfg.Threshold
	maxRetry := pCfg.MaxMetaRetry

	promptE, err := repository.GetCurrentPromptByKey("prompt_e")
	if err != nil || promptE == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaPromptMissing.Error())
		return ErrMetaPromptMissing
	}

	dict, err := repository.GetCurrentPromptByKey("dict")
	if err != nil || dict == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaDictMissing.Error())
		return ErrMetaDictMissing
	}

	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("meta: 课程 %s 不存在", pipeline.CourseCode)
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// 索引读取：一轮读原始索引(course_indexes)，二审读定稿索引(pipeline_indexes)
	var metaIndexContent string
	if pipeline.ReviewRound >= 2 {
		// 二审：读取上一轮verify生成的定稿索引
		pipelineIdx, pIdxErr := repository.GetPipelineIndex(pipeline.ID)
		if pIdxErr != nil {
			durationMs := time.Since(startTime).Milliseconds()
			errMsg := fmt.Sprintf("meta(2审): 定稿索引不存在，请确认一轮verify已完成: %s", pIdxErr.Error())
			_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
			return fmt.Errorf("%s", errMsg)
		}
		metaIndexContent = pipelineIdx.IndexContent
	} else {
		// 一轮：读取原始OSS索引
		tmpIdx, cIdxErr := repository.GetCourseIndex(course.ID)
		if cIdxErr != nil {
			durationMs := time.Since(startTime).Milliseconds()
			errMsg := "meta: 课程索引不存在"
			_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
			return fmt.Errorf("%s", errMsg)
		}
		metaIndexContent = tmpIdx.IndexContent
	}

	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrMetaScannerNotDone.Error())
		return ErrMetaScannerNotDone
	}
	scannerLocationJSON := extractScannerParsed(scannerStep.StepData)

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
		return fmt.Errorf("%s", errMsg)
	}

	// 组装多轮评估报告文本
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
		return fmt.Errorf("%s", errMsg)
	}
	roundsText := strings.Join(roundsTextParts, "\n\n")

	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey, "meta",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("meta: 获取AI配置失败: %s", err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// 构建用户提示词
	systemPrompt := promptE.Content
	userPromptParts := []string{
		"【课程定位】", scannerLocationJSON, "",
		"【待评估索引（原始）】", metaIndexContent, "",
		"【多轮评估结果（共" + fmt.Sprintf("%d", doneCount) + "轮）】", roundsText, "",
		"【TE-DNA解压缩字典】", dict.Content, "",
	}

	// v68新增：二审时注入一审修改方案+翻译结果+审核意见上下文
	// 让meta在仲裁时延续一审方向，在一审基础上增量优化而非推倒重来
	if pipeline.ReviewRound >= 2 {
		firstRoundCtx := buildFirstRoundContextForMeta(pipeline.ID)
		if firstRoundCtx != "" {
			userPromptParts = append(userPromptParts, firstRoundCtx, "")
		}
	}

	userPromptParts = append(userPromptParts, "禁止输出<thinking>标签或任何思维过程标记。")
	userPrompt := strings.Join(userPromptParts, "\n")

	// Meta重试循环
	metaResult := &models.MetaResult{TotalRetries: maxRetry}
	var lastOutput string
	var lastModelUsed string
	var totalTokens int
	var totalLatencyMs int64

	for attempt := 1; attempt <= maxRetry; attempt++ {
		metaResult.Attempt = attempt

		callResult, callErr := s.callAIWithSemaphore(aiCfg, systemPrompt, userPrompt, pipeline.ID)
		if callErr != nil {
			totalLatencyMs += time.Since(startTime).Milliseconds() - totalLatencyMs
			if attempt == maxRetry {
				durationMs := time.Since(startTime).Milliseconds()
				errMsg := fmt.Sprintf("%s (第%d次): %s", ErrMetaAIFailed.Error(), attempt, callErr.Error())
				_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
				return fmt.Errorf("%s", errMsg)
			}
			continue
		}

		lastOutput = callResult.Content
		lastModelUsed = callResult.ModelUsed
		totalTokens += callResult.TokensUsed
		totalLatencyMs += callResult.LatencyMs

		scoreResult := extractMetaScores(lastOutput)
		if !scoreResult.parseOk {
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

		passed := metaResult.TotalFinal >= threshold
		metaResult.Passed = passed
		metaResult.RawOutput = truncate(lastOutput, 50000)
		metaResult.ModelUsed = lastModelUsed
		metaResult.TokensUsed = totalTokens
		metaResult.LatencyMs = totalLatencyMs

		if passed {
			durationMs := time.Since(startTime).Milliseconds()
			if err := repository.CompleteStep(
				pipeline.ID, stepName, durationMs,
				metaResult.ToJSON(), lastModelUsed, totalTokens,
			); err != nil {
				return fmt.Errorf("保存meta结果失败: %w", err)
			}
			return nil
		}

		if attempt == maxRetry {
			durationMs := time.Since(startTime).Milliseconds()
			s.saveStepData(pipeline.ID, stepName, metaResult.ToJSON())
			errMsg := fmt.Sprintf("%s (最终得分: %.1f, 阈值: %.1f, 共%d次尝试)",
				ErrMetaAllRetriesFailed.Error(), metaResult.TotalFinal, threshold, maxRetry)
			_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
			return fmt.Errorf("%s", errMsg)
		}
	}

	durationMs := time.Since(startTime).Milliseconds()
	_ = repository.FailStep(pipeline.ID, stepName, durationMs, "meta: 异常退出重试循环")
	return fmt.Errorf("meta: 异常退出重试循环")
}

// ==================== Meta 评分提取 ====================

// metaScoreResult Meta评分提取结果
type metaScoreResult struct {
	totalFinal                             float64
	e1Final, e2Final, e3Final, e4Final     float64
	hardConstraint                         string
	grade                                  string
	e1Rounds, e2Rounds, e3Rounds, e4Rounds []float64
	parseOk                                bool
}

// extractMetaScores 从Meta AI输出中提取META_SCORE评分块
func extractMetaScores(output string) *metaScoreResult {
	result := &metaScoreResult{}

	blockMatch := reMetaBlock.FindStringSubmatch(output)
	if len(blockMatch) < 2 {
		tfm := reTotalFallback.FindStringSubmatch(output)
		if tfm != nil {
			result.totalFinal = safeParseFloat(tfm[1])
			if result.totalFinal > 0 {
				result.parseOk = true
			}
		}
		return result
	}

	block := blockMatch[1]
	tm := reMetaTotal.FindStringSubmatch(block)
	if tm == nil {
		return result
	}
	result.totalFinal = safeParseFloat(tm[1])

	if m := reE1Final.FindStringSubmatch(block); m != nil {
		result.e1Final = safeParseFloat(m[1])
	}
	if m := reE2Final.FindStringSubmatch(block); m != nil {
		result.e2Final = safeParseFloat(m[1])
	}
	if m := reE3Final.FindStringSubmatch(block); m != nil {
		result.e3Final = safeParseFloat(m[1])
	}
	if m := reE4Final.FindStringSubmatch(block); m != nil {
		result.e4Final = safeParseFloat(m[1])
	}

	if hm := reMetaHard.FindStringSubmatch(block); hm != nil {
		result.hardConstraint = hm[1]
	}
	if gm := reMetaGrade.FindStringSubmatch(block); gm != nil {
		result.grade = gm[1]
	}

	// 提取每轮各维度分数
	allRoundMatches := reMetaRound.FindAllStringSubmatch(block, -1)
	roundMap := map[int]map[int]float64{1: {}, 2: {}, 3: {}, 4: {}}
	maxRound := 0
	for _, m := range allRoundMatches {
		dim, _ := fmt.Sscanf(m[1], "%d", new(int))
		rn, _ := fmt.Sscanf(m[2], "%d", new(int))
		_ = dim
		_ = rn
		dimVal := int(safeParseFloat(m[1]))
		rnVal := int(safeParseFloat(m[2]))
		score := safeParseFloat(m[3])
		if dimVal >= 1 && dimVal <= 4 && rnVal >= 1 {
			roundMap[dimVal][rnVal] = score
			if rnVal > maxRound {
				maxRound = rnVal
			}
		}
	}

	for rn := 1; rn <= maxRound; rn++ {
		result.e1Rounds = append(result.e1Rounds, roundMap[1][rn])
		result.e2Rounds = append(result.e2Rounds, roundMap[2][rn])
		result.e3Rounds = append(result.e3Rounds, roundMap[3][rn])
		result.e4Rounds = append(result.e4Rounds, roundMap[4][rn])
	}

	// 兜底：从全文提取综合评分
	if fsm := reFinalScore.FindStringSubmatch(output); fsm != nil {
		newScore := safeParseFloat(fsm[1])
		if newScore > 0 {
			result.totalFinal = newScore
		}
	}

	result.parseOk = true
	return result
}

// ==================== Eval Rounds 查询 ====================

// EvalRoundDetail 评估轮次详情（API返回用）
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

// GetEvalRounds 获取Pipeline的评估轮次列表
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
			ID: r.ID, RoundNumber: r.RoundNumber, Status: r.Status, Output: r.Output,
			ScoreTotal: r.ScoreTotal, ScoreE1: r.ScoreE1, ScoreE2: r.ScoreE2,
			ScoreE3: r.ScoreE3, ScoreE4: r.ScoreE4,
			ModelUsed: r.ModelUsed, TokensUsed: r.TokensUsed,
		}
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
