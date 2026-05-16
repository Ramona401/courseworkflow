package services

// verify_batch.go — 批量验收+夜间定时任务+索引保存+高分早停辅助
//
// 从 verify_service.go 拆分

import (
	"encoding/json"
	"fmt"
	"strings"
	"tedna/internal/models"
	"tedna/internal/repository"
	"time"
)

// ==================== 批量验收+夜间定时任务 ====================

// BatchVerifyResult 批量验收结果
type BatchVerifyResult struct {
	TotalFound   int      `json:"total_found"`
	StartedIDs   []string `json:"started_ids"`
	SkippedIDs   []string `json:"skipped_ids"`
	ErrorMessage string   `json:"error_message"`
}

// BatchVerify 批量触发验收
// 查询所有finalized状态的pipeline并逐个触发验收
// 包括高分早停的pipeline（它们也需要经过验收确认质量）
func (s *PipelineService) BatchVerify() (*BatchVerifyResult, error) {
	result := &BatchVerifyResult{}

	ids, err := repository.ListFinalizedPipelineIDs()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("查询finalized Pipeline失败: %s", err.Error())
		return result, err
	}
	result.TotalFound = len(ids)

	if len(ids) == 0 {
		return result, nil
	}

	for _, id := range ids {
		pipeline, pErr := repository.GetPipelineByID(id)
		if pErr != nil || pipeline.Status != models.PipelineStatusFinalized {
			result.SkippedIDs = append(result.SkippedIDs, id)
			continue
		}

		capturedID := id
		if s.engine != nil {
			task := &EngineTask{
				Type:       TaskTypeVerify,
				PipelineID: capturedID,
				ExecFunc: func() error {
					_, vErr := s.VerifyPipeline(capturedID)
					return vErr
				},
			}
			if s.engine.Submit(task) {
				result.StartedIDs = append(result.StartedIDs, capturedID)
			} else {
				verifyLog.Warn("批量验收任务提交失败：队列已满", "pipeline_id", capturedID)
				result.SkippedIDs = append(result.SkippedIDs, capturedID)
			}
		} else {
			go func(vid string) {
				_, _ = s.VerifyPipeline(vid)
			}(capturedID)
			result.StartedIDs = append(result.StartedIDs, capturedID)
		}
	}

	return result, nil
}

// StartNightlyVerifyScheduler 启动夜间批量验收定时任务
func (s *PipelineService) StartNightlyVerifyScheduler() {
	go func() {
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			verifyLog.Warn("加载Asia/Shanghai时区失败，降级为UTC+8", "error", err)
			loc = time.FixedZone("CST", 8*3600)
		}

		for {
			now := time.Now().In(loc)
			next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, loc)
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			waitDuration := next.Sub(now)

			verifyLog.Info("夜间验收调度器等待中",
				"next_run", next.Format("2006-01-02 15:04:05"),
				"wait_duration", waitDuration.String())

			timer := time.NewTimer(waitDuration)
			<-timer.C

			fmt.Printf("[夜间验收] %s 开始执行批量验收...\n",
				time.Now().In(loc).Format("2006-01-02 15:04:05"))
			batchResult, batchErr := s.BatchVerify()
			if batchErr != nil {
				verifyLog.Error("夜间验收执行失败", "error", batchErr)
			} else {
				verifyLog.Info("夜间验收执行完成",
					"total_found", batchResult.TotalFound,
					"started", len(batchResult.StartedIDs),
					"skipped", len(batchResult.SkippedIDs))
			}
		}
	}()
}

// ==================== savePipelineIndex 辅助方法 ====================

// savePipelineIndex 从verify步骤的step_data中提取生成的索引并写入pipeline_indexes表
func (s *PipelineService) savePipelineIndex(pipelineID string) error {
	verifyStep, err := repository.GetStepByName(pipelineID, models.StepVerify)
	if err != nil || verifyStep.StepData == "" || verifyStep.StepData == "null" {
		return fmt.Errorf("无法读取verify步骤数据")
	}

	var verifyData map[string]interface{}
	if err := json.Unmarshal([]byte(verifyStep.StepData), &verifyData); err != nil {
		return fmt.Errorf("解析verify步骤数据失败: %w", err)
	}

	generatedIndex, ok := verifyData["generated_index"].(string)
	if !ok || len(generatedIndex) < 100 {
		return fmt.Errorf("verify步骤中generated_index为空或过短")
	}

	reviewRound := 1
	if rr, ok := verifyData["review_round"].(float64); ok && rr > 0 {
		reviewRound = int(rr)
	}

	pageCount := strings.Count(generatedIndex, "═══")
	if pageCount == 0 {
		pageCount = 1
	}

	if err := repository.UpsertPipelineIndex(pipelineID, generatedIndex, reviewRound, pageCount); err != nil {
		return err
	}

	verifyLog.Info("pipeline定稿索引已写入",
		"pipeline_id", pipelineID,
		"review_round", reviewRound,
		"index_length", len(generatedIndex),
		"page_count", pageCount,
	)
	return nil
}

// ==================== 高分早停辅助方法（v68新增）====================

// isEarlyStopPipeline 判断pipeline是否为高分早停（所有页面都是keep操作）
// v68新增：用于验收时快速通过，避免对原始课件重复评估
// 检查逻辑：
//   1. 优先检查review步骤的step_data是否包含early_stop标记（由executePipelineAsync高分早停时写入）
//   2. 兜底检查所有页面是否都是keep操作
func (s *PipelineService) isEarlyStopPipeline(pipeline *models.Pipeline) bool {
	// 检查review步骤的step_data是否包含early_stop标记
	reviewStep, err := repository.GetStepByName(pipeline.ID, models.StepReview)
	if err != nil || reviewStep.StepData == "" {
		return false
	}
	var reviewData map[string]interface{}
	if err := json.Unmarshal([]byte(reviewStep.StepData), &reviewData); err != nil {
		return false
	}
	// 检查是否有early_stop标记（由executePipelineAsync高分早停时写入）
	if earlyStop, ok := reviewData["early_stop"].(bool); ok && earlyStop {
		return true
	}
	// 兜底：检查所有页面是否都是keep操作
	pages, err := repository.GetGeneratedPagesByPipelineID(pipeline.ID)
	if err != nil || len(pages) == 0 {
		return false
	}
	for _, p := range pages {
		if p.Operation != "keep" {
			return false
		}
	}
	return true
}
