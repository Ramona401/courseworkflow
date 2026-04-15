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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
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
