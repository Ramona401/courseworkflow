package services

// pipeline_exec.go — Pipeline执行引擎
//
// 职责：
//   - StartPipeline 启动Pipeline
//   - executePipelineAsync 异步全链路执行（含断点续跑）
//   - RestartFromStep 从指定步骤重新开始
//   - BatchRestartFromStep 批量断点续跑
//   - executeDbCheck / doDbCheck 数据检查步骤
//   - executeScanner / doScanner Scanner步骤
//   - callAIWithSemaphore AI调用包装
//   - broadcastStepUpdate SSE事件广播
//
// v100修复：
//   Bug2 — doDbCheck 自动重拉索引：每次创建Pipeline执行dbCheck时，
//           自动从OSS重新拉取最新索引，避免隔天数据不同步问题。
//   Bug3 — doScanner 字段校验：兼容prompt_a新格式（ability_targets）
//           和旧格式（target），任一存在即通过校验。
//   优化  — doDbCheck 新增页数范围校验：中学课件（初中/高中）允许25-35页，
//           小学课件允许15-30页，超出范围给出警告而不阻断流程。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== Pipeline 启动 ====================

// StartPipeline 启动Pipeline执行
func (s *PipelineService) StartPipeline(id string) (*models.PipelineDetailResponse, error) {
	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	if pipeline.Status != models.PipelineStatusPending {
		return nil, ErrPipelineNotPending
	}

	if err := repository.UpdatePipelineStatus(id, models.StepDbCheck, models.PipelineStatusRunning); err != nil {
		return nil, fmt.Errorf("更新Pipeline状态失败: %w", err)
	}

	if s.engine != nil {
		task := &EngineTask{
			Type:       TaskTypePipeline,
			PipelineID: id,
			ExecFunc: func() error {
				s.executePipelineAsync(id)
				p, pErr := repository.GetPipelineByID(id)
				if pErr != nil {
					return fmt.Errorf("Pipeline执行后读取状态失败: %w", pErr)
				}
				if p.Status == models.PipelineStatusFailed {
					return fmt.Errorf("Pipeline执行失败: %s", p.ErrorMessage)
				}
				return nil
			},
		}
		if !s.engine.Submit(task) {
			_ = repository.UpdatePipelineStatus(id, models.StepDbCheck, models.PipelineStatusPending)
			return nil, ErrEngineQueueFull
		}
	} else {
		go s.executePipelineAsync(id)
	}

	return s.GetPipelineDetail(id)
}

// ==================== 断点续跑：RestartFromStep ====================

// RestartFromStep 从指定步骤重新开始执行Pipeline
// v37改进：增加 callerRole 参数实现细粒度权限控制
func (s *PipelineService) RestartFromStep(pipelineID string, stepName string, callerRole string) (*models.PipelineDetailResponse, error) {
	// ===== 1. 校验 Pipeline 是否存在 =====
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	// ===== 2. 不允许在 running 状态重启（防止并发执行） =====
	if pipeline.Status == models.PipelineStatusRunning {
		return nil, ErrRestartPipelineBusy
	}

	// ===== 3. 非 failed/cancelled 状态的重跑需要高级权限 =====
	if pipeline.Status != models.PipelineStatusFailed && pipeline.Status != models.PipelineStatusCancelled {
		if callerRole != models.RoleAdmin && callerRole != models.RoleSeniorOperator {
			return nil, ErrRestartPermissionDenied
		}
	}

	// ===== 4. 校验步骤名称合法性 =====
	allowedRestartSteps := map[string]bool{
		models.StepDbCheck:    true,
		models.StepScanner:    true,
		models.StepEvaluator:  true,
		models.StepMeta:       true,
		models.StepTranslator: true,
		models.StepGenerator:  true,
	}
	if _, ok := allowedRestartSteps[stepName]; !ok {
		if stepName == models.StepReview || stepName == models.StepVerify {
			return nil, ErrRestartStepNotAllowed
		}
		return nil, ErrRestartInvalidStep
	}

	// ===== 5. 获取目标步骤的 order =====
	targetOrder := -1
	for _, def := range models.StepDefinitions {
		if def.Name == stepName {
			targetOrder = def.Order
			break
		}
	}
	if targetOrder < 0 {
		return nil, ErrRestartInvalidStep
	}

	// ===== 6. 重置目标步骤及后续所有自动步骤 =====
	ctx := context.Background()
	_, err = database.DB.Exec(ctx, `
		UPDATE pipeline_steps
		SET status        = 'pending',
		    started_at    = NULL,
		    completed_at  = NULL,
		    duration_ms   = 0,
		    attempts      = 0,
		    step_data     = NULL,
		    error_message = NULL,
		    model_used    = NULL,
		    tokens_used   = 0,
		    updated_at    = NOW()
		WHERE pipeline_id = $1
		  AND step_order  >= $2
		  AND review_round = $3
		  AND step_name NOT IN ('review', 'verify')
	`, pipelineID, targetOrder, pipeline.ReviewRound)
	if err != nil {
		return nil, fmt.Errorf("重置步骤状态失败: %w", err)
	}

	// ===== 7. 同时重置 review 和 verify 步骤 =====
	_, err = database.DB.Exec(ctx, `
		UPDATE pipeline_steps
		SET status        = 'pending',
		    started_at    = NULL,
		    completed_at  = NULL,
		    duration_ms   = 0,
		    attempts      = 0,
		    step_data     = NULL,
		    error_message = NULL,
		    model_used    = NULL,
		    tokens_used   = 0,
		    updated_at    = NOW()
		WHERE pipeline_id = $1
		  AND review_round = $2
		  AND step_name IN ('review', 'verify')
	`, pipelineID, pipeline.ReviewRound)
	if err != nil {
		return nil, fmt.Errorf("重置review/verify步骤失败: %w", err)
	}

	// ===== 8. 清空 generated_pages（如需要） =====
	if targetOrder <= getStepOrder(models.StepGenerator) {
		_, err = database.DB.Exec(ctx,
			`DELETE FROM generated_pages WHERE pipeline_id = $1 AND review_round = $2`,
			pipelineID, pipeline.ReviewRound)
		if err != nil {
			return nil, fmt.Errorf("清空旧页面数据失败: %w", err)
		}
	}

	// ===== 9. 重置 Pipeline 状态 =====
	_, err = database.DB.Exec(ctx, `
		UPDATE pipelines
		SET status        = 'pending',
		    current_step  = $2,
		    error_message = NULL,
		    reject_reason = NULL,
		    completed_at  = NULL,
		    updated_at    = NOW()
		WHERE id = $1
	`, pipelineID, stepName)
	if err != nil {
		return nil, fmt.Errorf("重置Pipeline状态失败: %w", err)
	}

	// ===== 10. 通过引擎提交执行任务 =====
	if err := repository.UpdatePipelineStatus(pipelineID, stepName, models.PipelineStatusRunning); err != nil {
		return nil, fmt.Errorf("更新Pipeline为running状态失败: %w", err)
	}

	if s.engine != nil {
		task := &EngineTask{
			Type:       TaskTypePipeline,
			PipelineID: pipelineID,
			ExecFunc: func() error {
				s.executePipelineAsync(pipelineID)
				p, pErr := repository.GetPipelineByID(pipelineID)
				if pErr != nil {
					return fmt.Errorf("重跑后读取Pipeline状态失败: %w", pErr)
				}
				if p.Status == models.PipelineStatusFailed {
					return fmt.Errorf("重跑Pipeline执行失败: %s", p.ErrorMessage)
				}
				return nil
			},
		}
		if !s.engine.Submit(task) {
			_ = repository.UpdatePipelineStatus(pipelineID, stepName, models.PipelineStatusFailed)
			return nil, ErrEngineQueueFull
		}
	} else {
		go s.executePipelineAsync(pipelineID)
	}

	return s.GetPipelineDetail(pipelineID)
}

// ==================== 批量断点续跑 ====================

// BatchRestartResult 批量断点续跑结果
type BatchRestartResult struct {
	TotalRequested int      `json:"total_requested"`
	SuccessCount   int      `json:"success_count"`
	SkippedIDs     []string `json:"skipped_ids"`
	SkippedReasons []string `json:"skipped_reasons"`
	FailedIDs      []string `json:"failed_ids"`
	FailedReasons  []string `json:"failed_reasons"`
}

// BatchRestartFromStep 批量从指定步骤重新执行多个Pipeline
func (s *PipelineService) BatchRestartFromStep(pipelineIDs []string, stepName string, callerRole string) (*BatchRestartResult, error) {
	result := &BatchRestartResult{
		TotalRequested: len(pipelineIDs),
		SkippedIDs:     []string{},
		SkippedReasons: []string{},
		FailedIDs:      []string{},
		FailedReasons:  []string{},
	}

	for _, id := range pipelineIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		_, err := s.RestartFromStep(id, stepName, callerRole)
		if err != nil {
			errMsg := err.Error()
			if err == ErrRestartPipelineBusy || err == ErrPipelineNotFound {
				result.SkippedIDs = append(result.SkippedIDs, id)
				result.SkippedReasons = append(result.SkippedReasons, id+": "+errMsg)
			} else {
				result.FailedIDs = append(result.FailedIDs, id)
				result.FailedReasons = append(result.FailedReasons, id+": "+errMsg)
			}
			continue
		}
		result.SuccessCount++
	}

	return result, nil
}

// ==================== 异步全链路执行 ====================

// executePipelineAsync 异步执行Pipeline全链路
// Phase8修复P-01：扩展断点续跑逻辑，覆盖全部8步
func (s *PipelineService) executePipelineAsync(id string) {
	// v90功能优化2：Pipeline执行总超时监控（60分钟）
	// 启动一个goroutine定期检查执行时间，超时后自动标记失败并广播通知
	pipelineStartTime := time.Now()
	// v99优化4：Pipeline执行总超时从60分钟提升到120分钟，适应页面生成步骤较长的场景
	const pipelineTimeout = 120 * time.Minute
	timeoutCtx, timeoutCancel := context.WithCancel(context.Background())
	defer timeoutCancel()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-timeoutCtx.Done():
				return
			case <-ticker.C:
				elapsed := time.Since(pipelineStartTime)
				if elapsed > pipelineTimeout {
					// 总超时：标记Pipeline失败
					p, pErr := repository.GetPipelineByID(id)
					if pErr == nil && p.Status == models.PipelineStatusRunning {
						timeoutMsg := fmt.Sprintf("Pipeline执行超时（已运行%d分钟，上限%d分钟），已自动标记失败。请检查AI服务状态后重试。",
							int(elapsed.Minutes()), int(pipelineTimeout.Minutes()))
						_ = repository.UpdatePipelineError(id, p.CurrentStep, timeoutMsg)
						s.broadcastStepUpdate(id, "pipeline_error", p.CurrentStep, "failed", "failed", timeoutMsg)
					}
					return
				}
			}
		}
	}()

	pipeline, err := repository.GetPipelineByID(id)
	if err != nil {
		_ = repository.UpdatePipelineError(id, models.StepDbCheck, "异步执行: 读取Pipeline失败: "+err.Error())
		return
	}

	// ==================== 断点续跑逻辑 ====================
	existingSteps, stepsErr := repository.GetStepsByPipelineID(id)
	if stepsErr == nil && len(existingSteps) > 0 {
		var resumeStep string
		for _, st := range existingSteps {
			if st.Status != models.StepStatusDone {
				resumeStep = st.StepName
				break
			}
		}

		if resumeStep != "" && resumeStep != models.StepDbCheck {
			if err := repository.UpdatePipelineStatus(id, resumeStep, models.PipelineStatusRunning); err != nil {
				_ = repository.UpdatePipelineError(id, resumeStep, "恢复Pipeline失败: "+err.Error())
				return
			}
			pipeline, _ = repository.GetPipelineByID(id)

			switch resumeStep {
			case models.StepScanner:
				s.broadcastStepUpdate(id, "step_update", "scanner", "running", "running", "断点续跑: 从Scanner继续")
				if scanErr := s.executeScanner(pipeline); scanErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepScanner, scanErr.Error())
					s.broadcastStepUpdate(id, "pipeline_error", "scanner", "failed", "failed", scanErr.Error())
					return
				}
				fallthrough

			case models.StepEvaluator:
				if err := repository.UpdatePipelineStatus(id, models.StepEvaluator, models.PipelineStatusRunning); err != nil {
					_ = repository.UpdatePipelineError(id, models.StepEvaluator, "推进到Evaluator失败: "+err.Error())
					return
				}
				s.broadcastStepUpdate(id, "step_update", "evaluator", "running", "running", "断点续跑: 开始Evaluator")
				pipeline, _ = repository.GetPipelineByID(id)
				if evalErr := s.executeEvaluator(pipeline); evalErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepEvaluator, evalErr.Error())
					s.broadcastStepUpdate(id, "pipeline_error", "evaluator", "failed", "failed", evalErr.Error())
					return
				}
				// 高分早停检查
				pipeline, _ = repository.GetPipelineByID(id)
				if s.shouldEarlyStop(id, pipeline) {
					pCfgStop := models.ParsePipelineConfig(pipeline.Config)
					avgStop := s.getEvalAvgScore(id)
					earlyJSON := fmt.Sprintf(`{"early_stop":true,"reason":"Evaluator均分已达标，跳过Meta/Translator/Generator","eval_avg_score":%.2f,"threshold":%.2f}`, avgStop, pCfgStop.Threshold)
					pipeline, _ = repository.GetPipelineByID(id)
					s.fillEarlyStopPages(pipeline)
					_ = repository.StartStep(id, models.StepReview)
					_ = repository.CompleteStep(id, models.StepReview, 0, earlyJSON, "", 0)
					_ = repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue)
					s.broadcastStepUpdate(id, "pipeline_done", "review", "done", "review_queue",
						fmt.Sprintf("高分早停：Evaluator均分%.1f达标（阈值%.1f），直接进入审核队列", avgStop, pCfgStop.Threshold))
					return
				}
				fallthrough

			case models.StepMeta:
				if err := repository.UpdatePipelineStatus(id, models.StepMeta, models.PipelineStatusRunning); err != nil {
					_ = repository.UpdatePipelineError(id, models.StepMeta, "推进到Meta失败: "+err.Error())
					return
				}
				s.broadcastStepUpdate(id, "step_update", "meta", "running", "running", "断点续跑: 开始Meta")
				pipeline, _ = repository.GetPipelineByID(id)
				if metaErr := s.executeMeta(pipeline); metaErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepMeta, metaErr.Error())
					s.broadcastStepUpdate(id, "pipeline_error", "meta", "failed", "failed", metaErr.Error())
					return
				}
				fallthrough

			case models.StepTranslator:
				if err := repository.UpdatePipelineStatus(id, models.StepTranslator, models.PipelineStatusRunning); err != nil {
					_ = repository.UpdatePipelineError(id, models.StepTranslator, "推进到Translator失败: "+err.Error())
					return
				}
				s.broadcastStepUpdate(id, "step_update", "translator", "running", "running", "断点续跑: 开始Translator")
				pipeline, _ = repository.GetPipelineByID(id)
				if transErr := s.executeTranslator(pipeline); transErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepTranslator, transErr.Error())
					s.broadcastStepUpdate(id, "pipeline_error", "translator", "failed", "failed", transErr.Error())
					return
				}
				fallthrough

			case models.StepGenerator:
				if err := repository.UpdatePipelineStatus(id, models.StepGenerator, models.PipelineStatusRunning); err != nil {
					_ = repository.UpdatePipelineError(id, models.StepGenerator, "推进到Generator失败: "+err.Error())
					return
				}
				s.broadcastStepUpdate(id, "step_update", "generator", "running", "running", "断点续跑: 开始Generator")
				pipeline, _ = repository.GetPipelineByID(id)
				if genErr := s.executeGenerator(pipeline); genErr != nil {
					_ = repository.UpdatePipelineError(id, models.StepGenerator, genErr.Error())
					s.broadcastStepUpdate(id, "pipeline_error", "generator", "failed", "failed", genErr.Error())
					return
				}
				_ = repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue)
				s.broadcastStepUpdate(id, "pipeline_done", "review", "done", "review_queue", "Pipeline执行完成，等待审核")
				return

			case models.StepReview:
				_ = repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue)
				s.broadcastStepUpdate(id, "pipeline_done", "review", "pending", "review_queue", "断点续跑: 恢复到审核队列")
				return

			case models.StepVerify:
				_ = repository.UpdatePipelineStatus(id, models.StepVerify, models.PipelineStatusFinalized)
				return

			default:
				// 未知步骤：从dbCheck全量重跑
			}
		}
	}

	// ==================== 正常执行（全量/从dbCheck开始）====================

	dbCheckErr := s.executeDbCheck(pipeline)
	if dbCheckErr != nil {
		_ = repository.UpdatePipelineError(id, models.StepDbCheck, dbCheckErr.Error())
		s.broadcastStepUpdate(id, "pipeline_error", "dbCheck", "failed", "failed", dbCheckErr.Error())
		return
	}

	if err := repository.UpdatePipelineStatus(id, models.StepScanner, models.PipelineStatusRunning); err != nil {
		_ = repository.UpdatePipelineError(id, models.StepScanner, "推进到Scanner失败: "+err.Error())
		return
	}
	s.broadcastStepUpdate(id, "step_update", "scanner", "running", "running", "dbCheck完成，开始Scanner")

	if pipeline.AutoMode {
		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			_ = repository.UpdatePipelineError(id, models.StepScanner, "读取Pipeline失败: "+err.Error())
			return
		}

		scannerErr := s.executeScanner(pipeline)
		if scannerErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepScanner, scannerErr.Error())
			s.broadcastStepUpdate(id, "pipeline_error", "scanner", "failed", "failed", scannerErr.Error())
			return
		}

		if err := repository.UpdatePipelineStatus(id, models.StepEvaluator, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepEvaluator, "推进到Evaluator失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "evaluator", "running", "running", "Scanner完成，开始Evaluator")

		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			_ = repository.UpdatePipelineError(id, models.StepEvaluator, "读取Pipeline失败: "+err.Error())
			return
		}

		evalErr := s.executeEvaluator(pipeline)
		if evalErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepEvaluator, evalErr.Error())
			s.broadcastStepUpdate(id, "pipeline_error", "evaluator", "failed", "failed", evalErr.Error())
			return
		}

		// ===== 高分早停检查 =====
		pipeline, err = repository.GetPipelineByID(id)
		if err == nil && s.shouldEarlyStop(id, pipeline) {
			pCfgStop := models.ParsePipelineConfig(pipeline.Config)
			avgStop := s.getEvalAvgScore(id)
			earlyJSON := fmt.Sprintf(`{"early_stop":true,"reason":"Evaluator均分已达标，跳过Meta/Translator/Generator","eval_avg_score":%.2f,"threshold":%.2f}`, avgStop, pCfgStop.Threshold)
			s.fillEarlyStopPages(pipeline)
			_ = repository.StartStep(id, models.StepReview)
			_ = repository.CompleteStep(id, models.StepReview, 0, earlyJSON, "", 0)
			_ = repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue)
			s.broadcastStepUpdate(id, "pipeline_done", "review", "done", "review_queue",
				fmt.Sprintf("高分早停：Evaluator均分%.1f达标（阈值%.1f），直接进入审核队列", avgStop, pCfgStop.Threshold))
			return
		}

		if err := repository.UpdatePipelineStatus(id, models.StepMeta, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepMeta, "推进到Meta失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "meta", "running", "running", "Evaluator完成，开始Meta")

		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			_ = repository.UpdatePipelineError(id, models.StepMeta, "读取Pipeline失败: "+err.Error())
			return
		}

		metaErr := s.executeMeta(pipeline)
		if metaErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepMeta, metaErr.Error())
			s.broadcastStepUpdate(id, "pipeline_error", "meta", "failed", "failed", metaErr.Error())
			return
		}

		if err := repository.UpdatePipelineStatus(id, models.StepTranslator, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepTranslator, "推进到Translator失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "translator", "running", "running", "Meta完成，开始Translator")

		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			_ = repository.UpdatePipelineError(id, models.StepMeta, "读取Pipeline失败: "+err.Error())
			return
		}

		transErr := s.executeTranslator(pipeline)
		if transErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepTranslator, transErr.Error())
			s.broadcastStepUpdate(id, "pipeline_error", "translator", "failed", "failed", transErr.Error())
			return
		}

		if err := repository.UpdatePipelineStatus(id, models.StepGenerator, models.PipelineStatusRunning); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepGenerator, "推进到Generator失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "step_update", "generator", "running", "running", "Translator完成，开始Generator")

		pipeline, err = repository.GetPipelineByID(id)
		if err != nil {
			_ = repository.UpdatePipelineError(id, models.StepGenerator, "读取Pipeline失败: "+err.Error())
			return
		}

		genErr := s.executeGenerator(pipeline)
		if genErr != nil {
			_ = repository.UpdatePipelineError(id, models.StepGenerator, genErr.Error())
			s.broadcastStepUpdate(id, "pipeline_error", "generator", "failed", "failed", genErr.Error())
			return
		}

		if err := repository.UpdatePipelineStatus(id, models.StepReview, models.PipelineStatusReviewQueue); err != nil {
			_ = repository.UpdatePipelineError(id, models.StepReview, "推进到Review失败: "+err.Error())
			return
		}
		s.broadcastStepUpdate(id, "pipeline_done", "review", "done", "review_queue", "Pipeline执行完成，等待审核")
	}
}

// ==================== AI调用包装方法 ====================

// callAIWithSemaphore 通过引擎信号量控制AI并发调用
// v89-2变更：增加pipelineID参数，传入真实TraceContext用于成本追踪
// v99优化5：增加单步AI调用超时监控（15分钟），超时后记录警告日志
func (s *PipelineService) callAIWithSemaphore(cfg *ai.EffectiveConfig, systemPrompt string, userPrompt string, pipelineID string) (*ai.CallResult, error) {
	if s.engine != nil {
		s.engine.AcquireAI()
		defer s.engine.ReleaseAI()
	}

	// v89-2：构建真实TraceContext，关联Pipeline ID
	var traceCtx *ai.TraceContext
	if pipelineID != "" {
		pid := pipelineID
		traceCtx = &ai.TraceContext{
			SceneCode:  "scanner", // Pipeline步骤默认使用scanner场景（实际场景由cfg决定）
			PipelineID: &pid,
		}
	}

	// v99优化5：单步AI调用超时监控
	// 启动一个goroutine在15分钟后打印警告日志（AI调用本身通过HTTP超时控制）
	// 这里不强制cancel，因为HTTP client有自己的超时，这里只是监控告警
	done := make(chan struct{})
	go func() {
		select {
		case <-done:
			return
		case <-time.After(15 * time.Minute):
			fmt.Printf("[WARN] AI调用超过15分钟未返回 pipeline=%s model=%s\n", pipelineID, cfg.Model)
		}
	}()
	result, err := ai.CallAI(cfg, systemPrompt, userPrompt, traceCtx)
	close(done)
	return result, err
}

// ==================== SSE事件广播 ====================

// broadcastStepUpdate 广播Pipeline步骤更新事件
func (s *PipelineService) broadcastStepUpdate(pipelineID string, eventType string, currentStep string, stepStatus string, pipelineStatus string, message string) {
	event := SSEEvent{
		EventType:   eventType,
		PipelineID:  pipelineID,
		CurrentStep: currentStep,
		StepStatus:  stepStatus,
		Status:      pipelineStatus,
		Message:     message,
	}
	GlobalSSEHub.Broadcast(pipelineID, event)
}

