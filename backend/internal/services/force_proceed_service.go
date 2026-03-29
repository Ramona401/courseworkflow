package services

// ==================== v38新增：Translator FAIL后强制推进到Generator ====================
//
// 场景说明：
//   Translator-Reviewer循环有时会因为以下原因FAIL：
//   1. 总分差0.1分不达标（如8.9 vs 阈值9.0）
//   2. Reviewer发现"必须修复项"但实际是文档格式问题（如估时加总计算错误）
//   3. Translator反复修不好某个细节导致3轮用完
//
//   此时Translator的方案质量已经足够好（A级评分），操作员查看后可判断：
//   "这个方案可以用，不需要再花30分钟重跑"
//
//   本文件提供 ForceProceedToGenerator 方法：
//   将Translator步骤标记为done → 重置Generator → 提交执行
//
// 权限：admin / senior_operator / operator（与restart-from一致）
// 路由：POST /api/v1/pipelines/{id}/force-proceed

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ForceProceedToGenerator 当Translator步骤FAIL但方案质量可接受时，
// 操作员可确认使用当前Translator输出，强制将Translator标记为done并启动Generator。
//
// 前提条件：
//   - Pipeline状态为failed（Translator失败导致整个Pipeline失败）
//   - Translator步骤状态为failed
//   - Translator步骤有final_trans_output（说明至少完成了一轮有效输出）
//
// 执行流程：
//   1. 校验Pipeline状态和Translator输出
//   2. 将Translator步骤状态从failed改为done
//   3. 重置Generator及后续步骤（generator/review/verify）为pending
//   4. 清空旧的generated_pages数据
//   5. 更新Pipeline状态为running，current_step设为generator
//   6. 通过Engine提交异步执行任务
func (s *PipelineService) ForceProceedToGenerator(pipelineID string, callerRole string) (*models.PipelineDetailResponse, error) {
	// ===== 1. 校验Pipeline存在 =====
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	// ===== 2. 校验Pipeline状态：必须是failed =====
	// Translator循环FAIL会导致整个Pipeline变为failed状态
	// 只有failed状态才需要强制推进，其他状态应该走正常流程
	if pipeline.Status != models.PipelineStatusFailed {
		return nil, fmt.Errorf("仅Pipeline状态为failed时可强制推进，当前状态: %s", pipeline.Status)
	}

	// ===== 3. 校验Translator步骤状态：必须是failed =====
	transStep, err := repository.GetStepByName(pipelineID, models.StepTranslator)
	if err != nil {
		return nil, fmt.Errorf("获取Translator步骤失败: %w", err)
	}
	if transStep.Status != models.StepStatusFailed {
		return nil, fmt.Errorf("Translator步骤未失败（状态: %s），无需强制推进", transStep.Status)
	}

	// ===== 4. 校验Translator有有效输出 =====
	// extractTransFinalOutput 定义在 generator_service.go 中，从step_data提取final_trans_output
	transOutput := extractTransFinalOutput(transStep.StepData)
	if transOutput == "" {
		return nil, fmt.Errorf("Translator无有效输出，无法强制推进。请重跑Translator步骤")
	}

	// ===== 5. 将Translator步骤标记为done =====
	// 保留原始的step_data（包含各轮评分记录），只修改status和清除error_message
	ctx := context.Background()
	_, err = database.DB.Exec(ctx, `
		UPDATE pipeline_steps
		SET status        = 'done',
		    error_message = NULL,
		    updated_at    = NOW()
		WHERE pipeline_id = $1
		  AND step_name   = 'translator'
	`, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("更新Translator步骤状态失败: %w", err)
	}

	// ===== 6. 重置Generator及后续所有步骤为pending =====
	// generator(order=6) / review(order=7) / verify(order=8) 全部重置
	generatorOrder := getStepOrder(models.StepGenerator)
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
	`, pipelineID, generatorOrder)
	if err != nil {
		return nil, fmt.Errorf("重置Generator及后续步骤失败: %w", err)
	}

	// ===== 7. 清空旧的generated_pages =====
	_, err = database.DB.Exec(ctx,
		`DELETE FROM generated_pages WHERE pipeline_id = $1`, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("清空旧页面数据失败: %w", err)
	}

	// ===== 8. 更新Pipeline状态为running，从generator开始执行 =====
	_, err = database.DB.Exec(ctx, `
		UPDATE pipelines
		SET status        = 'running',
		    current_step  = 'generator',
		    error_message = NULL,
		    reject_reason = NULL,
		    completed_at  = NULL,
		    updated_at    = NOW()
		WHERE id = $1
	`, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("更新Pipeline状态失败: %w", err)
	}

	// ===== 9. 通过Engine提交异步执行任务 =====
	if s.engine != nil {
		task := &EngineTask{
			Type:       TaskTypePipeline,
			PipelineID: pipelineID,
			ExecFunc: func() error {
				// 异步执行Pipeline（从generator步骤开始，因为前面的步骤都已经是done状态）
				s.executePipelineAsync(pipelineID)
				// 执行完成后检查最终状态
				p, pErr := repository.GetPipelineByID(pipelineID)
				if pErr != nil {
					return fmt.Errorf("强制推进后读取Pipeline状态失败: %w", pErr)
				}
				if p.Status == models.PipelineStatusFailed {
					return fmt.Errorf("强制推进Pipeline执行失败: %s", p.ErrorMessage)
				}
				return nil
			},
		}
		if !s.engine.Submit(task) {
			// 队列满，回退Pipeline状态
			_ = repository.UpdatePipelineStatus(pipelineID, "generator", models.PipelineStatusFailed)
			return nil, ErrEngineQueueFull
		}
	} else {
		// 无Engine时直接启动goroutine（开发/测试模式）
		go s.executePipelineAsync(pipelineID)
	}

	// 返回更新后的Pipeline详情
	return s.GetPipelineDetail(pipelineID)
}
