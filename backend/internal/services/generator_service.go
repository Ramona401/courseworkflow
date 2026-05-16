package services

import (
	"encoding/json"
	"fmt"
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
		return fmt.Errorf("%s", errMsg)
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
		return fmt.Errorf("%s", errMsg)
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
		return fmt.Errorf("%s", errMsg)
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
		return fmt.Errorf("%s", errMsg)
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

	// 7.5 v65-bugfix: 2审时加载上一轮定稿HTML映射
	// 2审的generator应基于1审定稿后的版本进行修改，而非从OSS读取原始课件
	// prevRoundHTMLMap: map[pageNumber] → 上一轮定稿后的final_html
	var prevRoundHTMLMap map[int]string
	if pipeline.ReviewRound >= 2 {
		prevRoundHTMLMap = repository.GetPrevRoundFinalHTMLMap(pipeline.ID, pipeline.ReviewRound-1)
		if len(prevRoundHTMLMap) > 0 {
			fmt.Printf("[generator] 2审模式：已加载上一轮(%d)定稿HTML共%d页\n",
				pipeline.ReviewRound-1, len(prevRoundHTMLMap))
		}
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
			// keep：不调AI，直接使用原始HTML
			// v65-bugfix: 2审时优先从上一轮定稿读取，而非从OSS读原始课件
			result.KeptPages++
			if pipeline.ReviewRound >= 2 && prevRoundHTMLMap != nil {
				if prevHTML, hasPrev := prevRoundHTMLMap[realPageNum]; hasPrev && len(prevHTML) > 100 {
					pageRec.HasOrigHTML = true
					pageRec.OrigHTMLLen = len(prevHTML)
					pageRec.Status = "done"
					lidPtr := &lessonID
					_ = repository.CreateGeneratedPage(
						pipeline.ID, op.PageNumber, op.Title,
						"keep", prevHTML, "", prevHTML,
						lidPtr, "", changeReason,
					)
					result.Pages = append(result.Pages, pageRec)
					continue
				}
				// 上一轮没有该页面数据，降级从OSS读取
			}
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
			// modify：读取原始HTML + Sonnet修改
			// v65-bugfix: 2审时基于上一轮定稿版本修改，而非OSS原始课件
			result.ModifiedPages++
			if !hasLesson && (pipeline.ReviewRound < 2 || prevRoundHTMLMap == nil) {
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
			// v65-bugfix: 2审时优先从上一轮定稿读取原始HTML
			var origHTML string
			var fetchErr error
			if pipeline.ReviewRound >= 2 && prevRoundHTMLMap != nil {
				if prevHTML, hasPrev := prevRoundHTMLMap[realPageNum]; hasPrev && len(prevHTML) > 100 {
					origHTML = prevHTML
				}
			}
			// 没有上一轮定稿数据时，降级从OSS读取
			if origHTML == "" {
				if !hasLesson {
					pageRec.Status = "failed"
					pageRec.Error = fmt.Sprintf("modify页面无上一轮定稿且未找到lesson_id，原始页码=%d", realPageNum)
					result.FailedPages++
					_ = repository.CreateGeneratedPage(
						pipeline.ID, op.PageNumber, op.Title,
						"modify", "", "", "",
						nil, "", changeReason,
					)
					result.Pages = append(result.Pages, pageRec)
					continue
				}
				origHTML, fetchErr = ossService.FetchLessonHTML(lessonID)
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
			}
			pageRec.HasOrigHTML = true
			pageRec.OrigHTMLLen = len(origHTML)

			// modify使用Sonnet（模板化修改，速度快）
			userPrompt := buildModifyUserPrompt(pipeline.CourseCode, op, origHTML)
			callStart := time.Now()
			callResult, callErr := s.callAIWithSemaphore(aiCfgModify, promptF.Content, userPrompt, pipeline.ID)
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
				// v99修复Bug4：2审时如果1审已经创建过该页面，降级为modify
				// 原因：2审Translator可能对1审已新建的页面仍输出create指令，
				// 但该页面在1审定稿中已存在HTML，应基于1审版本修改而非重新创建
				if pipeline.ReviewRound >= 2 && prevRoundHTMLMap != nil {
					if prevHTML, hasPrev := prevRoundHTMLMap[realPageNum]; hasPrev && len(prevHTML) > 100 {
						// 1审已创建该页面，降级为modify
						fmt.Printf("[generator] 2审降级: P%d create→modify（1审已创建，基于定稿版本修改）\n", op.PageNumber)
						op.Operation = models.PageOpModify
						pageRec.Operation = "modify"
						result.CreatedPages--
						result.ModifiedPages++
						// 使用modify流程：基于1审定稿版本修改
						pageRec.HasOrigHTML = true
						pageRec.OrigHTMLLen = len(prevHTML)
						userPrompt := buildModifyUserPrompt(pipeline.CourseCode, op, prevHTML)
						callStart := time.Now()
						callResult, callErr := s.callAIWithSemaphore(aiCfgModify, promptF.Content, userPrompt, pipeline.ID)
						pageRec.LatencyMs = time.Since(callStart).Milliseconds()
						if callErr != nil {
							pageRec.Status = "failed"
							pageRec.Error = "2审降级modify失败: " + callErr.Error()
							result.FailedPages++
							result.ModifiedPages--
							_ = repository.CreateGeneratedPage(
								pipeline.ID, op.PageNumber, op.Title,
								"modify", prevHTML, "", prevHTML,
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
								"modify", prevHTML, genHTML, genHTML,
								nil, "", changeReason,
							)
						}
						result.Pages = append(result.Pages, pageRec)
						continue
					}
				}

			// create：用邻近页作为格式参考，Opus生成新页面（从零创建需要更强创意）
			result.CreatedPages++
			refHTML := findReferencePageHTML(ossService, pageLessonMap, realPageNum)

			userPrompt := buildCreateUserPrompt(pipeline.CourseCode, op, refHTML)
			callStart := time.Now()
			callResult, callErr := s.callAIWithSemaphore(aiCfgCreate, promptF.Content, userPrompt, pipeline.ID)
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
					callResult, callErr := s.callAIWithSemaphore(aiCfgModify, promptF.Content, userPrompt, pipeline.ID)
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
			callResult, callErr := s.callAIWithSemaphore(aiCfgMerge, promptF.Content, userPrompt, pipeline.ID)
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

