package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== Generator 错误常量 ====================

var (
	ErrGenPromptFMissing    = fmt.Errorf("Prompt F未配置，请先在提示词管理中设置prompt_f")
	ErrGenTranslatorNotDone = fmt.Errorf("Translator步骤未完成，无法执行Generator")
	ErrGenScannerNotDone    = fmt.Errorf("Scanner步骤未完成，无法执行Generator")
	ErrGenNoModuleID        = fmt.Errorf("课程无external_module_id，无法从OSS读取课件")
	ErrGenPageMapFailed     = fmt.Errorf("无法建立页码→课时ID映射")
	ErrGenNoPages           = fmt.Errorf("未解析到任何页面操作")
)

// createPageOffset 新增页面虚拟页码偏移量
// 新增页面page_number = 新课件位置 + 1000
// 例：在P09之后新增P10 → page_number=1010，排序时插入P09之后P11之前
const createPageOffset = 1000

// ==================== 提取"逐页修改指令"区域 ====================
//
// Translator输出结构：
//   区域1：前言+总览+变化总览  ← 跳过（含页码提及，会被误读）
//   区域2：逐页修改指令        ← 只解析这里
//   区域3：修改后完整页面清单  ← 跳过（表格行会被误读为页面指令）
//   区域4：评估分布图等附录    ← 跳过
func extractPageOpsSection(transOutput string) string {
	lines := strings.Split(transOutput, "\n")
	startIdx := -1
	endIdx := len(lines)

	for i, line := range lines {
		if strings.Contains(strings.TrimSpace(line), "逐页修改指令") {
			startIdx = i + 1
			break
		}
	}
	if startIdx < 0 {
		return transOutput // 兼容旧格式
	}

	endKeywords := []string{
		"修改后完整页面清单", "评估分布图", "修改后时间估算",
		"互动质量总览", "能力目标达成总览", "修改后整体指标", "全版本变更历史",
	}
	for i := startIdx; i < len(lines); i++ {
		for _, kw := range endKeywords {
			if strings.Contains(strings.TrimSpace(lines[i]), kw) {
				endIdx = i
				goto foundEnd
			}
		}
	}
foundEnd:
	return strings.Join(lines[startIdx:endIdx], "\n")
}

// ==================== Generator 步骤执行 ====================
//
// 模型分层策略：
//   keep   → 不调AI，直接从OSS读取
//   delete → 不调AI，标记删除
//   modify → Sonnet（在原HTML基础上修改，模板化操作）
//   create → Opus（从零创建新页面，需要更强创意）
//   merge  → Opus（多页合并，逻辑复杂）
func (s *PipelineService) executeGenerator(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepGenerator

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动generator失败: %w", err)
	}

	// 1. 加载 Prompt F
	promptF, err := repository.GetCurrentPromptByKey("prompt_f")
	if err != nil || promptF == nil {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrGenPromptFMissing.Error())
		return ErrGenPromptFMissing
	}

	// 2. 获取Translator最终输出
	transStep, err := repository.GetStepByName(pipeline.ID, models.StepTranslator)
	if err != nil || transStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrGenTranslatorNotDone.Error())
		return ErrGenTranslatorNotDone
	}
	transOutput := extractTransFinalOutput(transStep.StepData)
	if transOutput == "" {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := "generator: Translator输出为空"
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	// 3. 获取Scanner定位
	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrGenScannerNotDone.Error())
		return ErrGenScannerNotDone
	}

	// 4. 获取课程module_id
	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("generator: 课程 %s 不存在", pipeline.CourseCode)
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}
	if course.ExternalModuleID == nil || *course.ExternalModuleID == 0 {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrGenNoModuleID.Error())
		return ErrGenNoModuleID
	}
	moduleID := *course.ExternalModuleID

	// 5. 从OSS实时建立页码→lesson_id映射（每次重跑都取最新内容）
	ossService := NewOSSService(s.cfg)
	pageLessonMap, err := ossService.BuildPageLessonMap(moduleID)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("%s: %s", ErrGenPageMapFailed.Error(), err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	// 6. 提取"逐页修改指令"区域，解析页面操作列表
	// Translator必须输出所有页面的完整指令，不存在需要自动补充的页面
	pageOpsSection := extractPageOpsSection(transOutput)
	pageOps := parsePageOps(pageOpsSection)
	if len(pageOps) == 0 {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrGenNoPages.Error())
		return ErrGenNoPages
	}

	// 7. 按操作类型获取AI配置（模型分层）
	// modify → generator场景（Sonnet）
	// create → generator_create场景（Opus）
	// merge  → generator_merge场景（Opus）
	aiCfgModify, err := ai.GetEffectiveConfig(
		s.cfg.AESKey, models.SceneGenerator,
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("generator: 获取modify AI配置失败: %s", err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	aiCfgCreate, err := ai.GetEffectiveConfig(
		s.cfg.AESKey, models.SceneGeneratorCreate,
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		// 获取失败时降级使用modify配置
		aiCfgCreate = aiCfgModify
	}

	aiCfgMerge, err := ai.GetEffectiveConfig(
		s.cfg.AESKey, models.SceneGeneratorMerge,
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		// 获取失败时降级使用modify配置
		aiCfgMerge = aiCfgModify
	}

	// 8. 清理旧数据（每次重跑从OSS取最新内容）
	_ = repository.DeleteGeneratedPagesByPipelineID(pipeline.ID)

	// 9. 逐页执行（pageOps已按最终页码顺序排列）
	result := &models.GeneratorResult{}
	var totalTokens int
	var lastModelUsed string

	for _, op := range pageOps {
		pageRec := &models.GeneratorPageRecord{
			PageNumber: op.PageNumber,
			PageTitle:  op.Title,
			Operation:  op.Operation,
		}
		changeReason := op.Description

		// 用【原始页码】直接查pageLessonMap，这是唯一权威来源
		// create页面OriginalPageNumber=0，用虚拟页码还原找参考页
		realPageNum := op.GetOrigPageNum()
		if op.Operation == models.PageOpCreate && op.PageNumber >= createPageOffset {
			realPageNum = op.PageNumber - createPageOffset
		}
		lessonID, hasLesson := pageLessonMap[realPageNum]
		if hasLesson {
			pageRec.LessonID = lessonID
		}

		switch op.Operation {

		case models.PageOpKeep:
			// keep：从OSS读取原始HTML，不调AI
			result.KeptPages++
			if hasLesson {
				origHTML, fetchErr := ossService.FetchLessonHTML(lessonID)
				if fetchErr == nil && len(origHTML) > 100 {
					pageRec.HasOrigHTML = true
					pageRec.OrigHTMLLen = len(origHTML)
					pageRec.Status = "done"
					lidPtr := &lessonID
					_ = repository.CreateGeneratedPage(
						pipeline.ID, op.PageNumber, op.Title,
						"keep", origHTML, "", origHTML,
						lidPtr, "", changeReason,
					)
				} else {
					pageRec.Status = "done"
					pageRec.Error = fmt.Sprintf("OSS读取失败(lesson=%d): %v", lessonID, fetchErr)
					_ = repository.CreateGeneratedPage(
						pipeline.ID, op.PageNumber, op.Title,
						"keep", "", "", "",
						nil, "", changeReason,
					)
				}
			} else {
				pageRec.Status = "done"
				_ = repository.CreateGeneratedPage(
					pipeline.ID, op.PageNumber, op.Title,
					"keep", "", "", "",
					nil, "", changeReason,
				)
			}

		case models.PageOpDelete:
			// delete：标记删除，不调AI
			result.DeletedPages++
			pageRec.Status = "done"
			_ = repository.CreateGeneratedPage(
				pipeline.ID, op.PageNumber, op.Title,
				"delete", "", "", "",
				nil, "", changeReason,
			)

		case models.PageOpModify:
			// modify：从OSS读取原始HTML + Sonnet修改
			result.ModifiedPages++
			if !hasLesson {
				pageRec.Status = "failed"
				pageRec.Error = fmt.Sprintf("modify页面未找到lesson_id，原始页码=%d", realPageNum)
				result.FailedPages++
				_ = repository.CreateGeneratedPage(
					pipeline.ID, op.PageNumber, op.Title,
					"modify", "", "", "",
					nil, "", changeReason,
				)
				result.Pages = append(result.Pages, pageRec)
				continue
			}
			origHTML, fetchErr := ossService.FetchLessonHTML(lessonID)
			if fetchErr != nil || len(origHTML) < 100 {
				pageRec.Status = "failed"
				pageRec.Error = fmt.Sprintf("OSS读取失败(lesson=%d): %v", lessonID, fetchErr)
				result.FailedPages++
				_ = repository.CreateGeneratedPage(
					pipeline.ID, op.PageNumber, op.Title,
					"modify", "", "", "",
					&lessonID, "", changeReason,
				)
				result.Pages = append(result.Pages, pageRec)
				continue
			}
			pageRec.HasOrigHTML = true
			pageRec.OrigHTMLLen = len(origHTML)

			// modify使用Sonnet（模板化修改，速度快）
			userPrompt := buildModifyUserPrompt(pipeline.CourseCode, op, origHTML)
			callStart := time.Now()
			callResult, callErr := s.callAIWithSemaphore(aiCfgModify, promptF.Content, userPrompt)
			pageRec.LatencyMs = time.Since(callStart).Milliseconds()

			if callErr != nil {
				pageRec.Status = "failed"
				pageRec.Error = callErr.Error()
				result.FailedPages++
				_ = repository.CreateGeneratedPage(
					pipeline.ID, op.PageNumber, op.Title,
					"modify", origHTML, "", origHTML,
					&lessonID, "", changeReason,
				)
			} else {
				genHTML := extractGeneratedHTML(callResult.Content)
				pageRec.GenHTMLLen = len(genHTML)
				pageRec.TokensUsed = callResult.TokensUsed
				pageRec.Status = "done"
				totalTokens += callResult.TokensUsed
				lastModelUsed = callResult.ModelUsed
				_ = repository.CreateGeneratedPage(
					pipeline.ID, op.PageNumber, op.Title,
					"modify", origHTML, genHTML, genHTML,
					&lessonID, "", changeReason,
				)
			}

		case models.PageOpCreate:
			// create：用邻近页作为格式参考，Opus生成新页面（从零创建需要更强创意）
			result.CreatedPages++
			refHTML := findReferencePageHTML(ossService, pageLessonMap, realPageNum)

			userPrompt := buildCreateUserPrompt(pipeline.CourseCode, op, refHTML)
			callStart := time.Now()
			callResult, callErr := s.callAIWithSemaphore(aiCfgCreate, promptF.Content, userPrompt)
			pageRec.LatencyMs = time.Since(callStart).Milliseconds()

			if callErr != nil {
				pageRec.Status = "failed"
				pageRec.Error = callErr.Error()
				result.FailedPages++
				_ = repository.CreateGeneratedPage(
					pipeline.ID, op.PageNumber, op.Title,
					"create", "", "", "",
					nil, "", changeReason,
				)
			} else {
				genHTML := extractGeneratedHTML(callResult.Content)
				pageRec.GenHTMLLen = len(genHTML)
				pageRec.TokensUsed = callResult.TokensUsed
				pageRec.Status = "done"
				totalTokens += callResult.TokensUsed
				lastModelUsed = callResult.ModelUsed
				_ = repository.CreateGeneratedPage(
					pipeline.ID, op.PageNumber, op.Title,
					"create", "", genHTML, genHTML,
					nil, "", changeReason,
				)
			}

		case models.PageOpMerge:
			// merge：读取多个源页面HTML，Opus合并生成（多页合并逻辑复杂）
			result.MergedPages++
			sourceHTMLs := collectMergeSourceHTMLs(ossService, pageLessonMap, op)
			if len(sourceHTMLs) < 2 {
				// 源页面不足，降级为modify
				pageRec.Operation = "modify"
				op.Operation = models.PageOpModify
				result.MergedPages--
				result.ModifiedPages++
				if len(sourceHTMLs) == 1 {
					userPrompt := buildModifyUserPrompt(pipeline.CourseCode, op, sourceHTMLs[0].html)
					callStart := time.Now()
					callResult, callErr := s.callAIWithSemaphore(aiCfgModify, promptF.Content, userPrompt)
					pageRec.LatencyMs = time.Since(callStart).Milliseconds()
					if callErr != nil {
						pageRec.Status = "failed"
						pageRec.Error = "merge降级modify后AI失败: " + callErr.Error()
						result.FailedPages++
						result.ModifiedPages--
					} else {
						genHTML := extractGeneratedHTML(callResult.Content)
						pageRec.GenHTMLLen = len(genHTML)
						pageRec.TokensUsed = callResult.TokensUsed
						pageRec.Status = "done"
						totalTokens += callResult.TokensUsed
						lastModelUsed = callResult.ModelUsed
						lidPtr := sourceHTMLs[0].lessonID
						_ = repository.CreateGeneratedPage(
							pipeline.ID, op.PageNumber, op.Title,
							"modify", sourceHTMLs[0].html, genHTML, genHTML,
							lidPtr, "", changeReason,
						)
					}
				} else {
					pageRec.Status = "failed"
					pageRec.Error = "merge源页面为0，无法生成"
					result.FailedPages++
					result.ModifiedPages--
				}
				result.Pages = append(result.Pages, pageRec)
				continue
			}

			mergeSourceNums := make([]int, len(sourceHTMLs))
			for i, sh := range sourceHTMLs {
				mergeSourceNums[i] = sh.pageNum
			}
			pageRec.MergeSources = mergeSourceNums

			// merge使用Opus（复杂合并需要更强推理）
			userPrompt := buildMergeUserPrompt(pipeline.CourseCode, op, sourceHTMLs)
			callStart := time.Now()
			callResult, callErr := s.callAIWithSemaphore(aiCfgMerge, promptF.Content, userPrompt)
			pageRec.LatencyMs = time.Since(callStart).Milliseconds()

			mergeJSON, _ := json.Marshal(mergeSourceNums)
			if callErr != nil {
				pageRec.Status = "failed"
				pageRec.Error = callErr.Error()
				result.FailedPages++
				_ = repository.CreateGeneratedPage(
					pipeline.ID, op.PageNumber, op.Title,
					"merge", sourceHTMLs[0].html, "", sourceHTMLs[0].html,
					nil, string(mergeJSON), changeReason,
				)
			} else {
				genHTML := extractGeneratedHTML(callResult.Content)
				pageRec.GenHTMLLen = len(genHTML)
				pageRec.TokensUsed = callResult.TokensUsed
				pageRec.Status = "done"
				totalTokens += callResult.TokensUsed
				lastModelUsed = callResult.ModelUsed
				_ = repository.CreateGeneratedPage(
					pipeline.ID, op.PageNumber, op.Title,
					"merge", sourceHTMLs[0].html, genHTML, genHTML,
					nil, string(mergeJSON), changeReason,
				)
			}
		}
		result.Pages = append(result.Pages, pageRec)
	}

	// 10. 汇总结果
	result.TotalPages = len(pageOps)
	result.TotalTokens = totalTokens
	result.TotalLatencyMs = time.Since(startTime).Milliseconds()
	result.ModelUsed = lastModelUsed

	durationMs := time.Since(startTime).Milliseconds()
	if err := repository.CompleteStep(
		pipeline.ID, stepName, durationMs,
		result.ToJSON(), lastModelUsed, totalTokens,
	); err != nil {
		return fmt.Errorf("保存generator结果失败: %w", err)
	}
	return nil
}

// ==================== 解析逐页修改指令 ====================
//
// 设计原则：
//   1. 只解析"逐页修改指令"区域（由extractPageOpsSection提取）
//   2. 每页的【原始页码】字段是取OSS内容的唯一依据
//   3. 完全不使用origCursor指针法，不自动补充任何页面
//   4. Translator必须输出所有页面的完整指令
//
// 虚拟页码规则：
//   新增页（create）page_number = 新课件位置 + 1000
//   例：在P09之后新增P10 → page_number=1010
//   排序时1010自然排在9之后11之前，前端按此顺序展示
func parsePageOps(transOutput string) []*models.PageOperation {
	var ops []*models.PageOperation
	opsMap := make(map[int]*models.PageOperation)
	createCounterMap := make(map[int]int)

	lines := strings.Split(transOutput, "\n")

	opKeywords := map[string]string{
		"保留": models.PageOpKeep, "keep": models.PageOpKeep, "无变化": models.PageOpKeep,
		"修改": models.PageOpModify, "modify": models.PageOpModify,
		"新增": models.PageOpCreate, "新建": models.PageOpCreate, "create": models.PageOpCreate, "插入": models.PageOpCreate,
		"合并": models.PageOpMerge, "merge": models.PageOpMerge,
		"删除": models.PageOpDelete, "delete": models.PageOpDelete, "移除": models.PageOpDelete,
	}

	pageHeaderRe := regexp.MustCompile(`(?m)^-*\s*P(\d{1,2})\s+(.+?)(?:\s*【|$)`)
	origPageFieldRe := regexp.MustCompile(`【原始页码】\s*(?:P|p)(\d{1,3})(?:\s*[+＋]\s*(?:P|p)(\d{1,3}))?`)
	origPageNoneRe := regexp.MustCompile(`【原始页码】\s*无`)

	var currentPage int
	var currentTitle string
	var currentDesc []string

	flushPage := func() {
		if currentPage <= 0 {
			return
		}
		desc := strings.Join(currentDesc, "\n")
		op := detectOperation(desc, opKeywords)

		var mergeSources []int
		if op == models.PageOpMerge {
			mergeSources = extractMergeSources(desc, currentPage)
		}

		// 从【原始页码】字段直接解析OriginalPageNumber
		// 这是取OSS内容的唯一依据，不使用任何指针法推算
		origPageNum := 0
		isCreateOp := false

		if origPageNoneRe.MatchString(desc) {
			// 【原始页码】无 → 纯新增页面
			isCreateOp = true
			origPageNum = 0
		} else if m := origPageFieldRe.FindStringSubmatch(desc); m != nil {
			// 【原始页码】P03 或 【原始页码】P03+P04
			if parsed, err := strconv.Atoi(m[1]); err == nil && parsed > 0 {
				origPageNum = parsed
			}
			if op == models.PageOpMerge && m[2] != "" {
				if parsed2, err := strconv.Atoi(m[2]); err == nil && parsed2 > 0 {
					mergeSources = []int{origPageNum, parsed2}
				}
			}
		} else {
			// 没有【原始页码】字段：兼容旧格式，用当前页码
			origPageNum = currentPage
		}

		if op == models.PageOpCreate || isCreateOp {
			// 新增页面使用虚拟页码
			createCounterMap[currentPage]++
			counter := createCounterMap[currentPage]
			virtualPageNum := currentPage + createPageOffset + (counter-1)*10

			newTitle := fmt.Sprintf("P%02d-new %s", currentPage, currentTitle)
			if len([]rune(newTitle)) > 60 {
				newTitle = string([]rune(newTitle)[:60])
			}

			opsMap[virtualPageNum] = &models.PageOperation{
				PageNumber:         virtualPageNum,
				OriginalPageNumber: 0,
				Operation:          models.PageOpCreate,
				Title:              newTitle,
				Description:        desc,
			}
		} else {
			if existing, ok := opsMap[currentPage]; ok {
				existing.Description += "\n" + desc
				if opPriority(op) > opPriority(existing.Operation) {
					existing.Operation = op
				}
				if len(mergeSources) > 0 {
					existing.MergeSources = mergeSources
				}
				if origPageNum > 0 {
					existing.OriginalPageNumber = origPageNum
				}
			} else {
				opsMap[currentPage] = &models.PageOperation{
					PageNumber:         currentPage,
					OriginalPageNumber: origPageNum,
					Operation:          op,
					Title:              currentTitle,
					Description:        desc,
					MergeSources:       mergeSources,
				}
			}
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 跳过表格行（含3个以上"|"）
		if strings.Count(trimmed, "|") >= 3 {
			continue
		}

		// 跳过纯分隔线
		if len(trimmed) >= 2 {
			allSep := true
			for _, ch := range trimmed {
				if ch != '-' && ch != '=' && ch != '＝' {
					allSep = false
					break
				}
			}
			if allSep {
				continue
			}
		}

		// 检测页面标题行
		if m := pageHeaderRe.FindStringSubmatch(trimmed); m != nil {
			flushPage()
			pn, _ := strconv.Atoi(m[1])
			currentPage = pn
			currentTitle = strings.TrimSpace(m[2])
			currentTitle = regexp.MustCompile(`\s*【.*】.*$`).ReplaceAllString(currentTitle, "")
			currentTitle = strings.TrimSpace(currentTitle)
			if len([]rune(currentTitle)) > 60 {
				currentTitle = string([]rune(currentTitle)[:60])
			}
			currentDesc = []string{trimmed}
			continue
		}

		// 支持纯页码格式 "P18"
		if m := regexp.MustCompile(`^P(\d{1,2})\b`).FindStringSubmatch(trimmed); m != nil {
			if currentPage > 0 {
				flushPage()
			}
			pn, _ := strconv.Atoi(m[1])
			currentPage = pn
			currentTitle = trimmed
			currentDesc = []string{trimmed}
			continue
		}

		if currentPage > 0 {
			currentDesc = append(currentDesc, trimmed)
		}
	}
	flushPage()

	// 按页码排序（虚拟页码1010排在10之后11之前，即最终课件顺序）
	for _, op := range opsMap {
		ops = append(ops, op)
	}
	sort.Slice(ops, func(i, j int) bool {
		return ops[i].PageNumber < ops[j].PageNumber
	})

	return ops
}

// detectOperation 从描述文本中检测操作类型
func detectOperation(desc string, opKeywords map[string]string) string {
	opLineRe := regexp.MustCompile(`【操作】\s*(.+)`)
	if m := opLineRe.FindStringSubmatch(desc); m != nil {
		opText := strings.ToLower(strings.TrimSpace(m[1]))
		for kw, op := range opKeywords {
			if strings.Contains(opText, kw) {
				return op
			}
		}
	}
	headerOpRe := regexp.MustCompile(`【(修改|保留|新增|新建|删除|合并)】`)
	if m := headerOpRe.FindStringSubmatch(desc); m != nil {
		if op, ok := opKeywords[m[1]]; ok {
			return op
		}
	}
	if strings.Contains(desc, "✅无变化") || strings.Contains(desc, "✅ 无变化") {
		return models.PageOpKeep
	}
	return models.PageOpKeep
}

// opPriority 操作优先级
func opPriority(op string) int {
	switch op {
	case models.PageOpDelete:
		return 5
	case models.PageOpMerge:
		return 4
	case models.PageOpCreate:
		return 3
	case models.PageOpModify:
		return 2
	case models.PageOpKeep:
		return 1
	default:
		return 0
	}
}

// extractMergeSources 从描述中提取merge源页码
func extractMergeSources(desc string, currentPage int) []int {
	re := regexp.MustCompile(`(?i)(?:合并|merge)\s*(?:P|页面?)(\d+)\s*[+和与&,，]\s*(?:P|页面?)(\d+)`)
	if m := re.FindStringSubmatch(desc); m != nil {
		s1, _ := strconv.Atoi(m[1])
		s2, _ := strconv.Atoi(m[2])
		return []int{s1, s2}
	}
	return nil
}

// ==================== Prompt构建函数 ====================

func buildModifyUserPrompt(courseCode string, op *models.PageOperation, origHTML string) string {
	var parts []string
	parts = append(parts,
		"【⚠️⚠️⚠️ 最重要 — 原始页面HTML，你必须在此基础上修改，禁止重写】",
		"以下是当前线上运行的完整HTML。你的输出必须以此为基础，只修改下方指令要求的部分，其余代码原封不动保留。",
		"",
		origHTML,
		"",
		"══════════════════════════════════════════════",
		"▲ 以上是原始HTML（必须作为修改基础） ▼ 以下是修改指令",
		"══════════════════════════════════════════════",
		"",
		fmt.Sprintf("【课程编号】%s", courseCode),
		fmt.Sprintf("【页面】P%02d — %s", op.PageNumber, op.Title),
		"【操作类型】modify（在原始HTML基础上修改）",
		"",
		"【Translator修改指令（必须严格执行）】",
		op.Description,
		"",
		"【⚠️ 最终提醒】你的输出必须与上方原始HTML有90%以上代码重合。只改指令要求的部分。导航栏、视频、图片不允许任何改动。输出完整HTML。",
	)
	return strings.Join(parts, "\n")
}

func buildCreateUserPrompt(courseCode string, op *models.PageOperation, refHTML string) string {
	var parts []string
	if refHTML != "" {
		parts = append(parts,
			"【⚠️⚠️⚠️ 格式参考页面 — 新建页面必须完全模仿此格式】",
			"以下是相邻页面的完整HTML。你必须：",
			"1. 完全复制其HTML结构、CSS样式、导航栏、配色方案、布局框架",
			"2. 只替换内容区域的文字和互动元素",
			"3. 导航栏100%原样复制",
			"",
			refHTML,
			"",
			"══════════════════════════════════════════════",
			"▲ 以上是格式参考页面 ▼ 以下是新页面内容要求",
			"══════════════════════════════════════════════",
			"",
		)
	} else {
		parts = append(parts,
			"【⚠️ 无可用参考页面，请生成完整独立HTML页面】",
			"要求：白色背景，100vh一屏，文字最小22px，自包含HTML。",
			"",
		)
	}
	parts = append(parts,
		fmt.Sprintf("【课程编号】%s", courseCode),
		fmt.Sprintf("【页面】%s（新增页面）", op.Title),
		"【操作类型】create（新建页面）",
		"",
		"【Translator创建指令（必须严格执行）】",
		op.Description,
		"",
		"【⚠️ 最终提醒】新建页面必须与参考页面的格式、布局、导航栏、CSS完全一致。只有内容区域不同。输出完整自包含HTML。",
	)
	return strings.Join(parts, "\n")
}

type sourcePageHTML struct {
	pageNum  int
	html     string
	lessonID *int
}

func buildMergeUserPrompt(courseCode string, op *models.PageOperation, sources []sourcePageHTML) string {
	var parts []string
	parts = append(parts,
		fmt.Sprintf("【⚠️⚠️⚠️ 合并任务 — 需要将以下%d个页面合并为1个页面】", len(sources)),
		"",
	)
	for i, src := range sources {
		parts = append(parts,
			fmt.Sprintf("═══ 源页面 %d/%d: P%02d ═══", i+1, len(sources), src.pageNum),
			src.html,
			"",
		)
	}
	parts = append(parts,
		"══════════════════════════════════════════════",
		"▲ 以上是需要合并的所有源页面 ▼ 以下是合并要求",
		"══════════════════════════════════════════════",
		"",
		"【合并规则】",
		fmt.Sprintf("1. 以第一个源页面(P%02d)的HTML结构、导航栏、CSS为基础", sources[0].pageNum),
		"2. 将其他源页面的核心内容整合进来",
		"3. 导航栏只保留一份",
		"4. 所有源页面的视频、图片等资产原位保留",
		"5. 合并后仍为一屏(100vh)，内容放不下用标签页切换或折叠面板",
		"",
		fmt.Sprintf("【课程编号】%s", courseCode),
		fmt.Sprintf("【目标页面】P%02d — %s", op.PageNumber, op.Title),
		"【操作类型】merge",
		"",
		"【Translator合并指令】",
		op.Description,
		"",
		"【⚠️ 最终提醒】合并后页面必须保留所有源页面的视频和图片资产。导航栏只保留一份。一屏展示。输出完整HTML。",
	)
	return strings.Join(parts, "\n")
}

// ==================== 辅助工具函数 ====================

func extractTransFinalOutput(stepData string) string {
	if stepData == "" || stepData == "null" {
		return ""
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stepData), &data); err != nil {
		return ""
	}
	raw, ok := data["final_trans_output"]
	if !ok || string(raw) == "null" {
		return ""
	}
	var output string
	if err := json.Unmarshal(raw, &output); err != nil {
		return strings.Trim(string(raw), `"`)
	}
	return output
}

func extractGeneratedHTML(aiOutput string) string {
	codeBlockRe := regexp.MustCompile("(?s)```html\\s*\n?(.*?)```")
	if m := codeBlockRe.FindStringSubmatch(aiOutput); m != nil {
		return strings.TrimSpace(m[1])
	}
	doctypeRe := regexp.MustCompile(`(?si)(<(!DOCTYPE|html)[\s\S]*)`)
	if m := doctypeRe.FindStringSubmatch(aiOutput); m != nil {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(aiOutput)
}

func findReferencePageHTML(ossService *OSSService, pageLessonMap map[int]int, pageNum int) string {
	for n := pageNum - 1; n >= 1; n-- {
		if lid, ok := pageLessonMap[n]; ok {
			html, err := ossService.FetchLessonHTML(lid)
			if err == nil && len(html) > 200 {
				return html
			}
		}
	}
	for n := pageNum + 1; n <= pageNum+5; n++ {
		if lid, ok := pageLessonMap[n]; ok {
			html, err := ossService.FetchLessonHTML(lid)
			if err == nil && len(html) > 200 {
				return html
			}
		}
	}
	for _, lid := range pageLessonMap {
		html, err := ossService.FetchLessonHTML(lid)
		if err == nil && len(html) > 200 {
			return html
		}
	}
	return ""
}

func collectMergeSourceHTMLs(ossService *OSSService, pageLessonMap map[int]int, op *models.PageOperation) []sourcePageHTML {
	var sources []sourcePageHTML
	if len(op.MergeSources) >= 2 {
		for _, pn := range op.MergeSources {
			if lid, ok := pageLessonMap[pn]; ok {
				html, err := ossService.FetchLessonHTML(lid)
				if err == nil && len(html) > 100 {
					lidPtr := new(int)
					*lidPtr = lid
					sources = append(sources, sourcePageHTML{pageNum: pn, html: html, lessonID: lidPtr})
				}
			}
		}
		if len(sources) >= 2 {
			return sources
		}
	}
	sources = nil
	origPn := op.GetOrigPageNum()
	if lid, ok := pageLessonMap[origPn]; ok {
		html, err := ossService.FetchLessonHTML(lid)
		if err == nil && len(html) > 100 {
			lidPtr := new(int)
			*lidPtr = lid
			sources = append(sources, sourcePageHTML{pageNum: origPn, html: html, lessonID: lidPtr})
		}
	}
	nextPn := origPn + 1
	if lid, ok := pageLessonMap[nextPn]; ok {
		html, err := ossService.FetchLessonHTML(lid)
		if err == nil && len(html) > 100 {
			lidPtr := new(int)
			*lidPtr = lid
			sources = append(sources, sourcePageHTML{pageNum: nextPn, html: html, lessonID: lidPtr})
		}
	}
	return sources
}
