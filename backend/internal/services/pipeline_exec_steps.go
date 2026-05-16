package services

// pipeline_exec_steps.go — Pipeline dbCheck+Scanner 步骤执行
//
// 从 pipeline_exec.go 拆分

import (
	"encoding/json"
	"fmt"
	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
	"time"
)

// ==================== dbCheck 步骤 ====================

// executeDbCheck 执行数据检查步骤
func (s *PipelineService) executeDbCheck(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepDbCheck

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动dbCheck失败: %w", err)
	}

	result := &models.DbCheckResult{CourseCode: pipeline.CourseCode}
	checkErr := s.doDbCheck(pipeline, result)
	durationMs := time.Since(startTime).Milliseconds()

	if checkErr != nil {
		result.IsValid = false
		result.ErrorDetail = checkErr.Error()
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, checkErr.Error())
		s.saveStepData(pipeline.ID, stepName, result.ToJSON())
		return checkErr
	}

	result.IsValid = true
	if err := repository.CompleteStep(pipeline.ID, stepName, durationMs, result.ToJSON(), "", 0); err != nil {
		return fmt.Errorf("保存dbCheck结果失败: %w", err)
	}
	return nil
}

// doDbCheck 执行数据检查的具体逻辑
//
// v100修复Bug2：每次执行dbCheck时，自动从OSS重新拉取最新索引并更新数据库。
// 背景：用户在课件平台更新课件后，需要手动触发"拉取索引"才能在AI审核平台同步。
// 修复后，每次创建新Pipeline执行dbCheck时，系统自动重拉，无需手动操作。
// 失败时不阻断流程，仅打印警告，继续使用数据库中现有索引。
//
// v100功能优化：新增页数范围警告
// 中学课件（GradeNum >= 7）允许 25-35 页；小学课件（GradeNum < 7）允许 15-30 页。
// 超出范围时记录 PageCountWarn 警告，不阻断 Pipeline 执行。
func (s *PipelineService) doDbCheck(pipeline *models.Pipeline, result *models.DbCheckResult) error {
	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		return fmt.Errorf("课程 %s 不存在", pipeline.CourseCode)
	}
	result.CourseID = course.ID
	if course.ExternalModuleID != nil {
		result.ModuleID = *course.ExternalModuleID
	}

	// v100修复Bug2：自动重拉OSS索引，确保每次Pipeline使用最新索引
	// 仅在课程绑定了外部模块ID时执行，失败不阻断流程
	if course.ExternalModuleID != nil && *course.ExternalModuleID > 0 {
		courseService := NewCourseService(s.cfg)
		if _, fetchErr := courseService.FetchIndex(pipeline.CourseCode); fetchErr != nil {
			// 索引重拉失败，打印警告但不中断流程，继续用数据库现有索引
			fmt.Printf("[WARN] dbCheck自动重拉索引失败 course=%s err=%s，继续使用现有索引\n",
				pipeline.CourseCode, fetchErr.Error())
		} else {
			fmt.Printf("[INFO] dbCheck自动重拉索引成功 course=%s\n", pipeline.CourseCode)
		}
	}

	idx, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		result.HasIndex = false
		return ErrDbCheckIndexMissing
	}
	result.HasIndex = true
	result.IndexHash = idx.IndexHash
	result.PageCount = idx.PageCount
	result.TotalLength = idx.TotalLength

	if len(idx.IndexContent) < models.MinIndexLength {
		return fmt.Errorf("%w (实际长度: %d, 最小要求: %d)",
			ErrDbCheckIndexTooShort, len(idx.IndexContent), models.MinIndexLength)
	}

	actualHash := utils.SHA256Hash(idx.IndexContent)
	if actualHash != idx.IndexHash {
		return fmt.Errorf("%w (存储: %s, 实际: %s)",
			ErrDbCheckIndexHashMismatch, idx.IndexHash[:16]+"...", actualHash[:16]+"...")
	}

	// v90-4修复Bug4：用BuildPageLessonMap计算过滤禁用页面后的实际可用页面数
	// course_indexes.page_count来自索引条目数（可能包含禁用页面），与后续Pipeline步骤使用的实际页面数不一致
	// 这里用实际可用页面数覆盖，确保dbCheck展示的页面数与元评估仲裁等步骤一致
	if course.ExternalModuleID != nil && *course.ExternalModuleID > 0 {
		ossService := NewOSSService(s.cfg)
		if pageMap, mapErr := ossService.BuildPageLessonMap(*course.ExternalModuleID); mapErr == nil && len(pageMap) > 0 {
			result.PageCount = len(pageMap)
		}
	}

	// v100功能优化：页数范围校验
	// 中学课件（初中7年级及以上）允许 25-35 页
	// 小学课件（6年级及以下）允许 15-30 页
	// 超出范围记录警告信息，不阻断流程
	if result.PageCount > 0 {
		var minPages, maxPages int
		var stageLabel string
		if course.GradeNum != nil && *course.GradeNum >= 7 {
			// 中学阶段（初中+高中）
			minPages = 25
			maxPages = 35
			stageLabel = "中学"
		} else {
			// 小学阶段
			minPages = 15
			maxPages = 30
			stageLabel = "小学"
		}
		if result.PageCount < minPages || result.PageCount > maxPages {
			result.PageCountWarn = fmt.Sprintf(
				"⚠️ %s课件页数为%d页，建议范围%d-%d页，请确认课件结构是否合理",
				stageLabel, result.PageCount, minPages, maxPages,
			)
			fmt.Printf("[WARN] 页数范围警告 course=%s pages=%d range=%d-%d(%s)\n",
				pipeline.CourseCode, result.PageCount, minPages, maxPages, stageLabel)
		}
	}

	return nil
}

// ==================== Scanner 步骤 ====================

// executeScanner 执行Scanner步骤（课程定位分析）
func (s *PipelineService) executeScanner(pipeline *models.Pipeline) error {
	startTime := time.Now()
	stepName := models.StepScanner

	if err := repository.StartStep(pipeline.ID, stepName); err != nil {
		return fmt.Errorf("启动scanner失败: %w", err)
	}

	result, callErr := s.doScanner(pipeline)
	durationMs := time.Since(startTime).Milliseconds()

	if callErr != nil {
		_ = repository.FailStep(pipeline.ID, stepName, durationMs, callErr.Error())
		if result != nil {
			s.saveStepData(pipeline.ID, stepName, result.ToJSON())
		}
		return callErr
	}

	if err := repository.CompleteStep(
		pipeline.ID, stepName, durationMs,
		result.ToJSON(), result.ModelUsed, result.TokensUsed,
	); err != nil {
		return fmt.Errorf("保存scanner结果失败: %w", err)
	}

	return nil
}

// doScanner 执行Scanner的具体逻辑
//
// v100修复Bug3：兼容prompt_a新旧两种输出格式的顶层字段名
// 旧格式：顶层字段为 "target"（字符串）
// 新格式：顶层字段为 "ability_targets"（数组）
// 只要两者之一存在即通过校验，避免因prompt_a升级后Scanner步骤报错。
func (s *PipelineService) doScanner(pipeline *models.Pipeline) (*models.ScannerResult, error) {
	result := &models.ScannerResult{}

	promptA, err := repository.GetCurrentPromptByKey("prompt_a")
	if err != nil || promptA == nil {
		return nil, ErrScannerPromptMissing
	}

	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil {
		return nil, fmt.Errorf("scanner: 课程 %s 不存在", pipeline.CourseCode)
	}
	courseIndex, err := repository.GetCourseIndex(course.ID)
	if err != nil {
		return nil, fmt.Errorf("scanner: 课程索引不存在，请先执行dbCheck")
	}

	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey, "scanner",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf("scanner: 获取AI配置失败: %w", err)
	}

	systemPrompt := promptA.Content
	userPrompt := fmt.Sprintf("请分析以下课程索引内容，按照要求输出JSON格式的定位信息：\n\n%s", courseIndex.IndexContent)

	callResult, err := s.callAIWithSemaphore(aiCfg, systemPrompt, userPrompt, pipeline.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrScannerAIFailed, err.Error())
	}

	result.RawOutput = callResult.Content
	result.ModelUsed = callResult.ModelUsed
	result.TokensUsed = callResult.TokensUsed

	jsonStr, ok := ai.ExtractJSON(callResult.Content)
	if !ok {
		result.IsValid = false
		return result, fmt.Errorf("%w (AI输出前200字符: %s)",
			ErrScannerParseFailed, truncate(callResult.Content, 200))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		result.IsValid = false
		return result, fmt.Errorf("%w (JSON解析错误: %s)", ErrScannerParseFailed, err.Error())
	}

	// v100修复Bug3：兼容新旧两种顶层字段名
	// 旧格式 prompt_a 输出包含 "target" 字段
	// 新格式 prompt_a 输出包含 "ability_targets" 字段（数组形式）
	// 只要其中一个存在即视为有效，避免因提示词升级导致Scanner报字段缺失错误
	_, hasTarget := parsed["target"]
	_, hasAbilityTargets := parsed["ability_targets"]
	if !hasTarget && !hasAbilityTargets {
		result.IsValid = false
		return result, fmt.Errorf("%w (缺少必要字段 target 或 ability_targets，实际字段: %v)",
			ErrScannerParseFailed, getMapKeys(parsed))
	}

	result.Parsed = json.RawMessage(jsonStr)
	result.IsValid = true
	return result, nil
}
