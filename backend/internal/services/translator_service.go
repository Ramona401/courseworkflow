package services

// Translator + Reviewer 循环执行服务
// v46修复：所有AI调用改用 callAIWithSemaphore，通过引擎信号量控制并发
//          修复之前直接调用 ai.CallAI() 绕过信号量导致AI API并发过高卡死的问题

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

// ==================== Translator+Reviewer 错误常量（P4-5新增）====================

var (
	ErrTransPromptCMissing   = fmt.Errorf("Prompt C未配置，请先在提示词管理中设置prompt_c")
	ErrTransPromptDMissing   = fmt.Errorf("Prompt D未配置，请先在提示词管理中设置prompt_d")
	ErrTransMetaNotDone      = fmt.Errorf("Meta步骤未完成，无法执行Translator")
	ErrTransScannerNotDone   = fmt.Errorf("Scanner步骤未完成，无法执行Translator")
	ErrTransAllLoopsFailed   = fmt.Errorf("Translator-Reviewer所有循环均未通过")
	ErrTransTranslatorFailed = fmt.Errorf("Translator AI调用失败")
	ErrTransReviewerFailed   = fmt.Errorf("Reviewer AI调用失败")
	ErrTransScoreExtract     = fmt.Errorf("Reviewer未能从AI输出中提取REVIEW_SCORE评分")
)

// ==================== Translator+Reviewer 步骤执行（P4-5新增）====================

// executeTranslator 执行translator步骤：Translator(Prompt C) + Reviewer(Prompt D) 循环
// 循环逻辑：Reviewer评分≥threshold且QUALITY_GATE=PASS → 通过
//           否则提取反馈重跑Translator（最多max_tr_loop次）
// v46修复：所有AI调用改用 s.callAIWithSemaphore() 以遵守引擎并发限制
func (s *PipelineService) executeTranslator(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepTranslator

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动translator失败: %w", err)
	}

	// 解析Pipeline配置
	pCfg := models.ParsePipelineConfig(pipeline.Config)
	threshold := pCfg.Threshold // 默认9.0
	maxLoop := pCfg.MaxTRLoop   // 默认3

	// 1. 加载 Prompt C（Translator提示词）
	promptC, err := repository.GetCurrentPromptByKey("prompt_c")
	if err != nil || promptC == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrTransPromptCMissing.Error())
		return ErrTransPromptCMissing
	}

	// 2. 加载 Prompt D（Reviewer提示词）
	promptD, err := repository.GetCurrentPromptByKey("prompt_d")
	if err != nil || promptD == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrTransPromptDMissing.Error())
		return ErrTransPromptDMissing
	}

	// 3. 获取课程索引（原始索引）
	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("translator: 课程 %s 不存在", pipeline.CourseCode)
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}
	courseIndex, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := "translator: 课程索引不存在"
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	// 4. 获取Scanner步骤的parsed结果（课程定位JSON）
	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrTransScannerNotDone.Error())
		return ErrTransScannerNotDone
	}
	scannerLocationJSON := extractScannerParsed(scannerStep.StepData)

	// 5. 获取Meta步骤的raw_output（包含修改方案+修改后索引）
	metaStep, err := repository.GetStepByName(pipeline.ID, models.StepMeta)
	if err != nil || metaStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrTransMetaNotDone.Error())
		return ErrTransMetaNotDone
	}
	metaRawOutput := extractMetaRawOutput(metaStep.StepData)

	// 6. 获取AI配置（使用translator场景）
	aiCfgTrans, err := ai.GetEffectiveConfig(
		s.cfg.AESKey,
		"translator",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("translator: 获取Translator AI配置失败: %s", err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	// 7. 获取AI配置（使用reviewer场景）
	aiCfgReview, err := ai.GetEffectiveConfig(
		s.cfg.AESKey,
		"reviewer",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("translator: 获取Reviewer AI配置失败: %s", err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	// 8. 循环执行 Translator → Reviewer
	result := &models.TranslatorResult{
		MaxLoops:  maxLoop,
		Threshold: threshold,
	}
	var totalTokens int
	var lastModelUsed string
	var feedback string // Reviewer反馈（用于下一轮Translator输入）

	for loop := 1; loop <= maxLoop; loop++ {
		roundRecord := &models.TranslatorRoundRecord{
			Round: loop,
		}

		// ---- 8a. Translator调用（Prompt C）----
		// v46修复：使用 callAIWithSemaphore 替代直接 ai.CallAI，遵守引擎并发限制
		transUserPrompt := buildTranslatorUserPrompt(
			scannerLocationJSON,
			courseIndex.IndexContent,
			metaRawOutput,
			feedback,
			loop,
		)

		transResult, transErr := s.callAIWithSemaphore(aiCfgTrans, promptC.Content, transUserPrompt, pipeline.ID)
		if transErr != nil {
			roundRecord.TransError = transErr.Error()
			result.Rounds = append(result.Rounds, roundRecord)
			totalTokens += 0
			// Translator失败，继续下一轮重试
			if loop == maxLoop {
				durationMs := time.Since(startTime).Milliseconds()
				result.TotalTokens = totalTokens
				result.TotalLatencyMs = durationMs
				result.ModelUsed = lastModelUsed
				s.saveStepData(pipeline.ID, stepName, result.ToJSON())
				errMsg := fmt.Sprintf("%s (第%d轮): %s", ErrTransTranslatorFailed.Error(), loop, transErr.Error())
				_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
				return fmt.Errorf(errMsg)
			}
			continue
		}

		transOutput := transResult.Content
		lastModelUsed = transResult.ModelUsed
		totalTokens += transResult.TokensUsed
		roundRecord.TransOutput = truncate(transOutput, 50000)
		roundRecord.TransTokens = transResult.TokensUsed
		roundRecord.TransLatencyMs = transResult.LatencyMs

		// ---- 8b. Reviewer调用（Prompt D）----
		// v46修复：使用 callAIWithSemaphore 替代直接 ai.CallAI，遵守引擎并发限制
		reviewUserPrompt := buildReviewerUserPrompt(
			scannerLocationJSON,
			courseIndex.IndexContent,
			metaRawOutput,
			transOutput,
		)

		reviewResult, reviewErr := s.callAIWithSemaphore(aiCfgReview, promptD.Content, reviewUserPrompt, pipeline.ID)
		if reviewErr != nil {
			roundRecord.ReviewError = reviewErr.Error()
			result.Rounds = append(result.Rounds, roundRecord)
			totalTokens += 0
			// Reviewer失败，继续下一轮重试
			if loop == maxLoop {
				durationMs := time.Since(startTime).Milliseconds()
				result.TotalTokens = totalTokens
				result.TotalLatencyMs = durationMs
				result.ModelUsed = lastModelUsed
				s.saveStepData(pipeline.ID, stepName, result.ToJSON())
				errMsg := fmt.Sprintf("%s (第%d轮): %s", ErrTransReviewerFailed.Error(), loop, reviewErr.Error())
				_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
				return fmt.Errorf(errMsg)
			}
			continue
		}

		reviewOutput := reviewResult.Content
		totalTokens += reviewResult.TokensUsed
		roundRecord.ReviewOutput = truncate(reviewOutput, 50000)
		roundRecord.ReviewTokens = reviewResult.TokensUsed
		roundRecord.ReviewLatencyMs = reviewResult.LatencyMs

		// ---- 8c. 提取Reviewer评分 ----
		scoreInfo := extractReviewScores(reviewOutput)
		roundRecord.Score = scoreInfo.score
		roundRecord.QualityGate = scoreInfo.qualityGate
		roundRecord.HardCheck = scoreInfo.hardCheck
		roundRecord.Grade = scoreInfo.grade
		roundRecord.E1 = scoreInfo.e1
		roundRecord.E2 = scoreInfo.e2
		roundRecord.E3 = scoreInfo.e3
		roundRecord.E4 = scoreInfo.e4

		// ---- 8d. 判断是否通过 ----
		// 通过规则：QUALITY_GATE=PASS直接通过；QUALITY_GATE=FAIL直接不通过（有必须修复项）；
		// QUALITY_GATE为空时（未提取到）用分数≥threshold判定
		var passed bool
		if scoreInfo.qualityGate == "PASS" {
			passed = true
		} else if scoreInfo.qualityGate == "FAIL" {
			passed = false
		} else {
			// 未提取到QUALITY_GATE，回退到分数判定
			passed = scoreInfo.score > 0 && scoreInfo.score >= threshold
		}
		roundRecord.Passed = passed

		result.Rounds = append(result.Rounds, roundRecord)

		if passed {
			// 通过！保存结果
			result.Passed = true
			result.FinalScore = scoreInfo.score
			result.FinalQualityGate = scoreInfo.qualityGate
			result.FinalGrade = scoreInfo.grade
			result.FinalRound = loop
			result.FinalTransOutput = truncate(transOutput, 50000)
			result.FinalReviewOutput = truncate(reviewOutput, 50000)
			result.TotalTokens = totalTokens
			result.TotalLatencyMs = time.Since(startTime).Milliseconds()
			result.ModelUsed = lastModelUsed

			durationMs := time.Since(startTime).Milliseconds()
			if err := repository.CompleteStep(
				pipeline.ID, stepName, durationMs,
				result.ToJSON(), lastModelUsed, totalTokens,
			); err != nil {
				return fmt.Errorf("保存translator结果失败: %w", err)
			}
			return nil
		}

		// 未通过，提取反馈供下一轮使用
		feedback = extractReviewFeedback(reviewOutput)
		if feedback == "" {
			// 如果提取不到结构化反馈，使用Reviewer输出的后半部分
			feedback = truncate(reviewOutput, 3000)
		}
	}

	// 所有循环用完，仍未通过
	durationMs := time.Since(startTime).Milliseconds()
	lastRound := result.Rounds[len(result.Rounds)-1]
	result.Passed = false
	result.FinalScore = lastRound.Score
	result.FinalQualityGate = lastRound.QualityGate
	result.FinalGrade = lastRound.Grade
	result.FinalRound = maxLoop
	result.FinalTransOutput = lastRound.TransOutput
	result.FinalReviewOutput = lastRound.ReviewOutput
	result.TotalTokens = totalTokens
	result.TotalLatencyMs = durationMs
	result.ModelUsed = lastModelUsed

	s.saveStepData(pipeline.ID, stepName, result.ToJSON())
	errMsg := fmt.Sprintf("%s (最终得分: %.1f, 阈值: %.1f, 共%d轮)",
		ErrTransAllLoopsFailed.Error(), lastRound.Score, threshold, maxLoop)
	_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
	return fmt.Errorf(errMsg)
}

// ==================== Translator/Reviewer 输入构造（P4-5新增）====================

// buildTranslatorUserPrompt 构造Translator的用户消息
// 输入：课程定位JSON + 原始索引 + Meta完整输出（含修改方案+修改后索引） + Reviewer反馈（非首轮）
func buildTranslatorUserPrompt(
	scannerLocationJSON string,
	originalIndex string,
	metaRawOutput string,
	feedback string,
	round int,
) string {
	parts := []string{
		"【课程定位】",
		scannerLocationJSON,
		"",
		"【原始索引】",
		originalIndex,
		"",
		"【Meta元评估完整输出（含修改方案+修改后索引）】",
		metaRawOutput,
	}

	// 非首轮：追加Reviewer反馈
	if round > 1 && feedback != "" {
		parts = append(parts,
			"",
			fmt.Sprintf("【Reviewer审核反馈（第%d轮，请针对性修复扣分项）】", round-1),
			feedback,
			"",
			fmt.Sprintf("⚠️ 这是第%d次翻译。请仔细阅读上方审核反馈，只修复被扣分的部分，保留上一轮中通过的内容。", round),
		)
	}

	parts = append(parts, "", "禁止输出<thinking>标签或任何思维过程标记。")
	return strings.Join(parts, "\n")
}

// buildReviewerUserPrompt 构造Reviewer的用户消息
// 输入：课程定位JSON + 原始索引（作为通过索引对照） + Meta输出 + Translator输出
func buildReviewerUserPrompt(
	scannerLocationJSON string,
	originalIndex string,
	metaRawOutput string,
	transOutput string,
) string {
	parts := []string{
		"【工作模式】初审",
		"",
		"【课程定位】",
		scannerLocationJSON,
		"",
		"【通过索引】",
		// 从Meta输出中提取修改后索引作为通过索引
		// Meta输出包含完整修改后索引，Reviewer需要它作为对照基准
		extractMetaNewIndex(metaRawOutput),
		"",
		"【开发者方案】",
		transOutput,
	}

	parts = append(parts, "", "禁止输出<thinking>标签或任何思维过程标记。")
	return strings.Join(parts, "\n")
}

// ==================== Reviewer 评分提取（P4-5新增）====================

// reviewScoreResult Reviewer评分提取结果（内部使用）
type reviewScoreResult struct {
	score       float64 // TOTAL评分
	qualityGate string  // QUALITY_GATE: PASS/FAIL
	hardCheck   string  // HARD_CHECK: PASS/FAIL
	grade       string  // GRADE: A/B/C/D
	e1          float64 // E1评分
	e2          float64 // E2评分
	e3          float64 // E3评分
	e4          float64 // E4评分
	parseOk     bool    // 是否成功提取
}

// extractReviewScores 从Reviewer输出中提取<<<REVIEW_SCORE>>>块中的评分
// 格式：
//
//	<<<REVIEW_SCORE>>>
//	HARD_CHECK:PASS或FAIL
//	E1:{X.X}
//	E2:{X.X}
//	E3:{X.X}
//	E4:{X.X}
//	TOTAL:{X.X}
//	GRADE:{A/B/C/D}
//	QUALITY_GATE:{PASS/FAIL}
//	<<<END_REVIEW_SCORE>>>
func extractReviewScores(output string) *reviewScoreResult {
	result := &reviewScoreResult{}

	// 主解析：<<<REVIEW_SCORE>>>...<<<END_REVIEW_SCORE>>>
	blockRe := regexp.MustCompile(`(?s)<<<REVIEW_SCORE>>>(.*?)<<<END_REVIEW_SCORE>>>`)
	blockMatch := blockRe.FindStringSubmatch(output)

	if len(blockMatch) < 2 {
		// 未找到REVIEW_SCORE块，尝试备用提取
		totalFallbackRe := regexp.MustCompile(`(?i)(?:TOTAL|综合评分)[：:\s]+([\d.]+)`)
		tfm := totalFallbackRe.FindStringSubmatch(output)
		if tfm != nil {
			result.score = safeParseFloat(tfm[1])
			if result.score > 0 {
				result.parseOk = true
			}
		}
		// 尝试提取QUALITY_GATE
		qgRe := regexp.MustCompile(`(?i)QUALITY_GATE[：:\s]+(PASS|FAIL)`)
		if qgm := qgRe.FindStringSubmatch(output); qgm != nil {
			result.qualityGate = strings.ToUpper(qgm[1])
		}
		return result
	}

	block := blockMatch[1]

	// 提取 TOTAL
	totalRe := regexp.MustCompile(`(?i)TOTAL[：:\s]+([\d.]+)`)
	if tm := totalRe.FindStringSubmatch(block); tm != nil {
		result.score = safeParseFloat(tm[1])
	}

	// 提取 E1~E4
	e1Re := regexp.MustCompile(`(?i)E1[：:\s]+([\d.]+)`)
	e2Re := regexp.MustCompile(`(?i)E2[：:\s]+([\d.]+)`)
	e3Re := regexp.MustCompile(`(?i)E3[：:\s]+([\d.]+)`)
	e4Re := regexp.MustCompile(`(?i)E4[：:\s]+([\d.]+)`)
	if m := e1Re.FindStringSubmatch(block); m != nil {
		result.e1 = safeParseFloat(m[1])
	}
	if m := e2Re.FindStringSubmatch(block); m != nil {
		result.e2 = safeParseFloat(m[1])
	}
	if m := e3Re.FindStringSubmatch(block); m != nil {
		result.e3 = safeParseFloat(m[1])
	}
	if m := e4Re.FindStringSubmatch(block); m != nil {
		result.e4 = safeParseFloat(m[1])
	}

	// 提取 QUALITY_GATE
	qgRe := regexp.MustCompile(`(?i)QUALITY_GATE[：:\s]+(PASS|FAIL)`)
	if qgm := qgRe.FindStringSubmatch(block); qgm != nil {
		result.qualityGate = strings.ToUpper(qgm[1])
	}

	// 提取 HARD_CHECK
	hcRe := regexp.MustCompile(`(?i)HARD_CHECK[：:\s]+(PASS|FAIL)`)
	if hcm := hcRe.FindStringSubmatch(block); hcm != nil {
		result.hardCheck = strings.ToUpper(hcm[1])
	}

	// 提取 GRADE
	gradeRe := regexp.MustCompile(`(?i)GRADE[：:\s]+([A-D])`)
	if gm := gradeRe.FindStringSubmatch(block); gm != nil {
		result.grade = strings.ToUpper(gm[1])
	}

	if result.score > 0 {
		result.parseOk = true
	}
	return result
}

// extractReviewFeedback 从Reviewer输出中提取审核反馈信息
// 提取【问题汇总】【必须修复】等关键反馈段落，供下一轮Translator参考
func extractReviewFeedback(output string) string {
	var parts []string

	// 提取REVIEW_SCORE块（含评分明细）
	scoreBlockRe := regexp.MustCompile(`(?s)<<<REVIEW_SCORE>>>(.*?)<<<END_REVIEW_SCORE>>>`)
	if sbm := scoreBlockRe.FindStringSubmatch(output); len(sbm) >= 2 {
		parts = append(parts, "【评分明细】\n"+strings.TrimSpace(sbm[1]))
	}

	// 提取包含关键词的段落
	feedbackPatterns := []string{
		"必须修复", "建议改进", "扣分", "问题汇总",
		"E1诊断", "E2诊断", "E3诊断", "E4诊断",
		"结论",
	}
	lines := strings.Split(output, "\n")
	capturing := false
	var captureLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isHeader := false
		for _, kw := range feedbackPatterns {
			if strings.Contains(trimmed, kw) {
				isHeader = true
				break
			}
		}
		if isHeader {
			if capturing && len(captureLines) > 0 {
				parts = append(parts, strings.Join(captureLines, "\n"))
			}
			captureLines = []string{trimmed}
			capturing = true
		} else if capturing {
			captureLines = append(captureLines, trimmed)
			// 限制每段最多30行
			if len(captureLines) > 30 {
				capturing = false
				parts = append(parts, strings.Join(captureLines, "\n"))
				captureLines = nil
			}
		}
	}
	if capturing && len(captureLines) > 0 {
		parts = append(parts, strings.Join(captureLines, "\n"))
	}

	result := strings.Join(parts, "\n\n")
	// 限制总长度
	if len([]rune(result)) > 5000 {
		result = string([]rune(result)[:5000]) + "..."
	}
	return result
}

// ==================== Meta 数据提取工具（P4-5新增）====================

// extractMetaRawOutput 从meta步骤的step_data中提取raw_output字段
func extractMetaRawOutput(stepData string) string {
	if stepData == "" || stepData == "null" {
		return ""
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stepData), &data); err != nil {
		return ""
	}
	rawOutput, ok := data["raw_output"]
	if !ok || string(rawOutput) == "null" {
		return ""
	}
	// raw_output是JSON字符串，需要反序列化
	var output string
	if err := json.Unmarshal(rawOutput, &output); err != nil {
		// 可能已经是纯文本
		return strings.Trim(string(rawOutput), `"`)
	}
	return output
}

// extractMetaNewIndex 从Meta原始输出中提取修改后索引部分
// Meta输出格式中，修改后索引通常在【修改方案】之后，以 P01: PT:... 格式逐页列出
// 如果提取失败，返回Meta完整输出（让AI自行从中提取）
func extractMetaNewIndex(metaRawOutput string) string {
	if metaRawOutput == "" {
		return "（无Meta输出）"
	}

	// 尝试提取从第一个 "P01:" 或 "P1:" 开始到 "===模块索引===" 或 "---" 分隔线之间的内容
	// Meta输出中修改后索引通常以页面编号开头
	indexStartRe := regexp.MustCompile(`(?m)^P0?1[:.]`)
	loc := indexStartRe.FindStringIndex(metaRawOutput)
	if loc == nil {
		// 找不到索引起始标记，返回完整Meta输出
		return metaRawOutput
	}

	indexPart := metaRawOutput[loc[0]:]

	// 尝试找到索引结束标记
	endMarkers := []string{"===模块索引===", "---\n\n## 瓶颈", "---\n\n## 【", "\n---\n\n>"}
	for _, marker := range endMarkers {
		endIdx := strings.Index(indexPart, marker)
		if endIdx > 0 {
			// 包含模块索引摘要行
			if marker == "===模块索引===" {
				// 找到模块索引后的结束位置
				afterModule := indexPart[endIdx:]
				moduleEndIdx := strings.Index(afterModule, "\n---")
				if moduleEndIdx > 0 {
					return indexPart[:endIdx+moduleEndIdx]
				}
				// 没找到结束，取到下一个空行
				return indexPart[:endIdx+len(marker)+200]
			}
			return indexPart[:endIdx]
		}
	}

	// 没找到结束标记，返回截取的索引部分（限制长度）
	if len([]rune(indexPart)) > 15000 {
		return string([]rune(indexPart)[:15000])
	}
	return indexPart
}
