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

// ==================== Generator 错误常量（P4-6新增）====================

var (
	ErrGenPromptFMissing    = fmt.Errorf("Prompt F未配置，请先在提示词管理中设置prompt_f")
	ErrGenTranslatorNotDone = fmt.Errorf("Translator步骤未完成，无法执行Generator")
	ErrGenScannerNotDone    = fmt.Errorf("Scanner步骤未完成，无法执行Generator")
	ErrGenMetaNotDone       = fmt.Errorf("Meta步骤未完成，无法执行Generator")
	ErrGenNoModuleID        = fmt.Errorf("课程无external_module_id，无法从OSS读取课件")
	ErrGenPageMapFailed     = fmt.Errorf("无法建立页码→课时ID映射")
	ErrGenNoPages           = fmt.Errorf("未解析到任何页面操作")
)

// ==================== Generator 步骤执行（P4-6新增）====================

// executeGenerator 执行generator步骤：根据Translator输出逐页生成/修改HTML
// 流程：解析页面操作 → 建立页码→lesson_id映射 → 逐页执行5种操作 → 存入generated_pages表
// P4.5-E更新：每页存入change_reason（Translator的修改理由/指令），供审核页面展示
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

	// 2. 获取Translator步骤的最终输出（FinalTransOutput）
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

	// 3. 获取Scanner定位（用于构建prompt）
	scannerStep, err := repository.GetStepByName(pipeline.ID, models.StepScanner)
	if err != nil || scannerStep.Status != models.StepStatusDone {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrGenScannerNotDone.Error())
		return ErrGenScannerNotDone
	}

	// 4. 获取课程信息和module_id
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

	// 5. 建立页码→lesson_id映射（从OSS读取模块详情）
	ossService := NewOSSService(s.cfg)
	pageLessonMap, err := ossService.BuildPageLessonMap(moduleID)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("%s: %s", ErrGenPageMapFailed.Error(), err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	// 6. 解析Translator输出为页面操作列表
	pageOps := parsePageOps(transOutput, len(pageLessonMap))
	if len(pageOps) == 0 {
		durationMs := time.Since(startTime).Milliseconds()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, ErrGenNoPages.Error())
		return ErrGenNoPages
	}

	// 7. 获取AI配置（使用generator场景）
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey,
		"generator",
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		durationMs := time.Since(startTime).Milliseconds()
		errMsg := fmt.Sprintf("generator: 获取AI配置失败: %s", err.Error())
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, errMsg)
		return fmt.Errorf(errMsg)
	}

	// 8. 清理旧的生成页面（支持重跑）
	_ = repository.DeleteGeneratedPagesByPipelineID(pipeline.ID)

	// 9. 逐页执行
	result := &models.GeneratorResult{}
	var totalTokens int
	var lastModelUsed string

	for _, op := range pageOps {
		pageRec := &models.GeneratorPageRecord{
			PageNumber: op.PageNumber,
			PageTitle:  op.Title,
			Operation:  op.Operation,
		}

		// P4.5-E: 提取修改理由（Translator给出的逐页修改指令）
		// 对于keep/delete页面也保留Description，作为衔接说明
		changeReason := op.Description

		// 获取当前页的lesson_id
		lessonID, hasLesson := pageLessonMap[op.PageNumber]
		if hasLesson {
			pageRec.LessonID = lessonID
		}

		switch op.Operation {
		case models.PageOpKeep:
			// ===== keep: 从OSS读取原始HTML，直接保存 =====
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
					pageRec.Error = "OSS读取失败或内容过短"
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
			// ===== delete: 标记删除，不调AI =====
			result.DeletedPages++
			pageRec.Status = "done"
			_ = repository.CreateGeneratedPage(
				pipeline.ID, op.PageNumber, op.Title,
				"delete", "", "", "",
				nil, "", changeReason,
			)

		case models.PageOpModify:
			// ===== modify: 读取原始HTML + AI修改 =====
			result.ModifiedPages++
			if !hasLesson {
				pageRec.Status = "failed"
				pageRec.Error = "无lesson_id，无法读取原始HTML"
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
				pageRec.Error = fmt.Sprintf("OSS读取失败: %v", fetchErr)
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

			// 构建modify prompt并调用AI
			userPrompt := buildModifyUserPrompt(pipeline.CourseCode, op, origHTML)
			callStart := time.Now()
			callResult, callErr := ai.CallAI(aiCfg, promptF.Content, userPrompt)
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
			// ===== create: 用邻近页作为格式参考，AI生成新页面 =====
			result.CreatedPages++
			refHTML := findReferencePageHTML(ossService, pageLessonMap, op.PageNumber)

			userPrompt := buildCreateUserPrompt(pipeline.CourseCode, op, refHTML)
			callStart := time.Now()
			callResult, callErr := ai.CallAI(aiCfg, promptF.Content, userPrompt)
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
			// ===== merge: 读取多个源页面HTML，AI合并生成 =====
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
					callResult, callErr := ai.CallAI(aiCfg, promptF.Content, userPrompt)
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

			// 构建merge源信息
			mergeSourceNums := make([]int, len(sourceHTMLs))
			for i, sh := range sourceHTMLs {
				mergeSourceNums[i] = sh.pageNum
			}
			pageRec.MergeSources = mergeSourceNums

			userPrompt := buildMergeUserPrompt(pipeline.CourseCode, op, sourceHTMLs)
			callStart := time.Now()
			callResult, callErr := ai.CallAI(aiCfg, promptF.Content, userPrompt)
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

// ==================== 解析Translator输出为页面操作（P4-6新增）====================

// parsePageOps 解析Translator输出中的逐页修改指令
// Translator输出格式：每页用 "--\nPXX 标题 【修改】\n--" 分隔
// 每页内有 【操作】保留/修改/新增/删除/合并 等标记
func parsePageOps(transOutput string, totalOrigPages int) []*models.PageOperation {
	var ops []*models.PageOperation
	opsMap := make(map[int]*models.PageOperation)

	lines := strings.Split(transOutput, "\n")

	// 操作关键词映射
	opKeywords := map[string]string{
		"保留": models.PageOpKeep, "keep": models.PageOpKeep, "无变化": models.PageOpKeep,
		"修改": models.PageOpModify, "modify": models.PageOpModify,
		"新增": models.PageOpCreate, "新建": models.PageOpCreate, "create": models.PageOpCreate, "插入": models.PageOpCreate,
		"合并": models.PageOpMerge, "merge": models.PageOpMerge,
		"删除": models.PageOpDelete, "delete": models.PageOpDelete, "移除": models.PageOpDelete,
	}

	// 页码+标题正则：匹配 "P01 标题文字" 或 "--\nP01 标题" 格式
	pageHeaderRe := regexp.MustCompile(`(?m)^-*\s*P(\d{1,2})\s+(.+?)(?:\s*【|$)`)

	// 按页面分段
	var currentPage int
	var currentTitle string
	var currentDesc []string

	flushPage := func() {
		if currentPage <= 0 {
			return
		}
		desc := strings.Join(currentDesc, "\n")
		op := detectOperation(desc, opKeywords)

		// 检测merge源页面
		var mergeSources []int
		if op == models.PageOpMerge {
			mergeSources = extractMergeSources(desc, currentPage)
		}

		if existing, ok := opsMap[currentPage]; ok {
			// 如果已存在，追加描述，保留更高优先级的操作
			existing.Description += "\n" + desc
			if opPriority(op) > opPriority(existing.Operation) {
				existing.Operation = op
			}
			if len(mergeSources) > 0 {
				existing.MergeSources = mergeSources
			}
		} else {
			opsMap[currentPage] = &models.PageOperation{
				PageNumber:   currentPage,
				Operation:    op,
				Title:        currentTitle,
				Description:  desc,
				MergeSources: mergeSources,
			}
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 检测页面标题行
		if m := pageHeaderRe.FindStringSubmatch(trimmed); m != nil {
			flushPage()
			pn, _ := strconv.Atoi(m[1])
			currentPage = pn
			currentTitle = strings.TrimSpace(m[2])
			// 去掉标题中可能的修改标记
			currentTitle = regexp.MustCompile(`\s*【.*】.*$`).ReplaceAllString(currentTitle, "")
			currentTitle = strings.TrimSpace(currentTitle)
			if len([]rune(currentTitle)) > 60 {
				currentTitle = string([]rune(currentTitle)[:60])
			}
			currentDesc = []string{trimmed}
			continue
		}

		// 也支持 "P18" 这种纯页码格式（新增页面标题在下一行）
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

	// 补充未在Translator输出中提到的原始页面（默认keep）
	for pn := 1; pn <= totalOrigPages; pn++ {
		if _, exists := opsMap[pn]; !exists {
			opsMap[pn] = &models.PageOperation{
				PageNumber: pn,
				Operation:  models.PageOpKeep,
				Title:      fmt.Sprintf("Page %d", pn),
			}
		}
	}

	// 按页码排序
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
	// 优先检查【操作】标记行
	opLineRe := regexp.MustCompile(`【操作】\s*(.+)`)
	if m := opLineRe.FindStringSubmatch(desc); m != nil {
		opText := strings.ToLower(strings.TrimSpace(m[1]))
		for kw, op := range opKeywords {
			if strings.Contains(opText, kw) {
				return op
			}
		}
	}

	// 检查标题中的【修改】【保留】等标记
	headerOpRe := regexp.MustCompile(`【(修改|保留|新增|新建|删除|合并)】`)
	if m := headerOpRe.FindStringSubmatch(desc); m != nil {
		if op, ok := opKeywords[m[1]]; ok {
			return op
		}
	}

	// 检查 ✅无变化
	if strings.Contains(desc, "✅无变化") || strings.Contains(desc, "✅ 无变化") {
		return models.PageOpKeep
	}

	// 检查EX→IT转换等关键词
	if strings.Contains(desc, "→") && (strings.Contains(desc, "转换") || strings.Contains(desc, "升级")) {
		return models.PageOpModify
	}

	// 默认keep
	return models.PageOpKeep
}

// opPriority 操作优先级（用于同一页有多个操作标记时取最高优先级）
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
	// 匹配 "合并P3和P4" 或 "merge P3+P4" 等
	re := regexp.MustCompile(`(?i)(?:合并|merge)\s*(?:P|页面?)(\d+)\s*[+和与&,，]\s*(?:P|页面?)(\d+)`)
	if m := re.FindStringSubmatch(desc); m != nil {
		s1, _ := strconv.Atoi(m[1])
		s2, _ := strconv.Atoi(m[2])
		return []int{s1, s2}
	}
	return nil
}

// ==================== Prompt构建函数（P4-6新增）====================

// buildModifyUserPrompt 构建modify操作的用户消息
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

// buildCreateUserPrompt 构建create操作的用户消息
func buildCreateUserPrompt(courseCode string, op *models.PageOperation, refHTML string) string {
	var parts []string

	if refHTML != "" {
		parts = append(parts,
			fmt.Sprintf("【⚠️⚠️⚠️ 格式参考页面 — 新建页面必须完全模仿此格式】"),
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
		fmt.Sprintf("【页面】P%02d — %s", op.PageNumber, op.Title),
		"【操作类型】create（新建页面）",
		"",
		"【Translator创建指令（必须严格执行）】",
		op.Description,
		"",
		"【⚠️ 最终提醒】新建页面必须与参考页面的格式、布局、导航栏、CSS完全一致。只有内容区域不同。输出完整自包含HTML。",
	)
	return strings.Join(parts, "\n")
}

// sourcePageHTML merge源页面数据
type sourcePageHTML struct {
	pageNum  int
	html     string
	lessonID *int
}

// buildMergeUserPrompt 构建merge操作的用户消息
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

// ==================== 辅助工具函数（P4-6新增）====================

// extractTransFinalOutput 从translator步骤的step_data中提取final_trans_output
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

// extractGeneratedHTML 从AI输出中提取HTML（去除```html包裹和多余文字）
func extractGeneratedHTML(aiOutput string) string {
	// 尝试提取```html...```块
	codeBlockRe := regexp.MustCompile("(?s)```html\\s*\n?(.*?)```")
	if m := codeBlockRe.FindStringSubmatch(aiOutput); m != nil {
		return strings.TrimSpace(m[1])
	}
	// 尝试从<!DOCTYPE或<html开始截取
	doctypeRe := regexp.MustCompile(`(?si)(<(!DOCTYPE|html)[\s\S]*)`)
	if m := doctypeRe.FindStringSubmatch(aiOutput); m != nil {
		return strings.TrimSpace(m[1])
	}
	// 无法识别，返回原文
	return strings.TrimSpace(aiOutput)
}

// findReferencePageHTML 为create操作找一个邻近页面作为格式参考
// 优先找前一页，其次后一页，最后找任意有HTML的页面
func findReferencePageHTML(ossService *OSSService, pageLessonMap map[int]int, pageNum int) string {
	// 向前找
	for n := pageNum - 1; n >= 1; n-- {
		if lid, ok := pageLessonMap[n]; ok {
			html, err := ossService.FetchLessonHTML(lid)
			if err == nil && len(html) > 200 {
				return html
			}
		}
	}
	// 向后找
	for n := pageNum + 1; n <= pageNum+5; n++ {
		if lid, ok := pageLessonMap[n]; ok {
			html, err := ossService.FetchLessonHTML(lid)
			if err == nil && len(html) > 200 {
				return html
			}
		}
	}
	// 任意页
	for _, lid := range pageLessonMap {
		html, err := ossService.FetchLessonHTML(lid)
		if err == nil && len(html) > 200 {
			return html
		}
	}
	return ""
}

// collectMergeSourceHTMLs 收集merge操作所需的所有源页面HTML
func collectMergeSourceHTMLs(ossService *OSSService, pageLessonMap map[int]int, op *models.PageOperation) []sourcePageHTML {
	var sources []sourcePageHTML

	// 如果有明确的mergeSources
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

	// 没有明确来源或不足，用当前页+下一页
	sources = nil
	pn := op.PageNumber
	if lid, ok := pageLessonMap[pn]; ok {
		html, err := ossService.FetchLessonHTML(lid)
		if err == nil && len(html) > 100 {
			lidPtr := new(int)
			*lidPtr = lid
			sources = append(sources, sourcePageHTML{pageNum: pn, html: html, lessonID: lidPtr})
		}
	}
	nextPn := pn + 1
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
