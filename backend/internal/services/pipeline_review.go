package services

// pipeline_review.go — 审核决策 + 定稿 + 批量操作 + 发布
//
// v68变更：
//   - AIFixPage增强：使用ai_fix独立场景(Sonnet+64000) + 专用系统提示词 + 修改说明提取 + 参考页面支持
//   - 新增AIFixResult/extractFixSummary/extractFixedHTML/aiFixSystemPrompt

import (
	"encoding/json"
	"fmt"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 快捷通过 ====================

// MarkPassed 快捷通过Pipeline
func (s *PipelineService) MarkPassed(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}

	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue: true,
		models.PipelineStatusNeedsHuman:  true,
		models.PipelineStatusFailed:      true,
	}
	if !allowedStatuses[pipeline.Status] {
		return ErrMarkPassedNotAllowed
	}

	metaStep, err := repository.GetStepByName(pipelineID, models.StepMeta)
	if err != nil || metaStep.Status != models.StepStatusDone {
		return ErrMarkPassedNotMet
	}

	var metaData map[string]interface{}
	if metaStep.StepData != "" && metaStep.StepData != "null" {
		if jsonErr := json.Unmarshal([]byte(metaStep.StepData), &metaData); jsonErr != nil {
			return ErrMarkPassedNotMet
		}
	}
	totalFinal, ok := metaData["total_final"].(float64)
	if !ok || totalFinal <= 0 {
		return ErrMarkPassedNotMet
	}

	pCfg := models.ParsePipelineConfig(pipeline.Config)
	if totalFinal < pCfg.Threshold {
		return fmt.Errorf("%w (得分: %.1f, 阈值: %.1f)", ErrMarkPassedNotMet, totalFinal, pCfg.Threshold)
	}

	reviewStep, err := repository.GetStepByName(pipelineID, models.StepReview)
	if err == nil && reviewStep.Status != models.StepStatusDone {
		_ = repository.StartStep(pipelineID, models.StepReview)
		statsJSON := fmt.Sprintf(`{"mark_passed":true,"meta_score":%.1f,"threshold":%.1f,"finalized_at":"%s"}`,
			totalFinal, pCfg.Threshold, time.Now().Format(time.RFC3339))
		_ = repository.CompleteStep(pipelineID, models.StepReview, 0, statsJSON, "", 0)
	}

	if err := repository.CompletePipeline(pipelineID, models.PipelineStatusFinalized); err != nil {
		return fmt.Errorf("快捷通过失败: %w", err)
	}

	return nil
}

// ==================== 审核决策方法 ====================

// GetGeneratedPages 获取Pipeline的所有生成页面
func (s *PipelineService) GetGeneratedPages(pipelineID string) ([]*repository.GeneratedPageFullRow, error) {
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	pages, err := repository.GetGeneratedPagesWithHTML(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("获取生成页面失败: %w", err)
	}
	if pages == nil {
		pages = []*repository.GeneratedPageFullRow{}
	}
	return pages, nil
}

// UpdatePageDecision 更新单页审核决策
func (s *PipelineService) UpdatePageDecision(pipelineID string, pageNumber int, decision string, finalHTML *string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue:     true,
		models.PipelineStatusNeedsHuman:      true,
		models.PipelineStatusPendingFinalize: true,
	}
	if !allowedStatuses[pipeline.Status] {
		return ErrPipelineNotReviewable
	}

	validDecisions := map[string]bool{"approve": true, "reject": true, "edit": true}
	if !validDecisions[decision] {
		return ErrInvalidDecision
	}

	if decision == "edit" && (finalHTML == nil || *finalHTML == "") {
		return fmt.Errorf("edit决策必须提供修改后的HTML内容")
	}

	return repository.UpdatePageDecision(pipelineID, pageNumber, decision, finalHTML)
}

// ==================== 定稿流程 ====================

// SubmitFinalize 提交定稿申请
func (s *PipelineService) SubmitFinalize(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusReviewQueue &&
		pipeline.Status != models.PipelineStatusNeedsHuman {
		return ErrSubmitFinalizeNotAllowed
	}

	total, decided, err := repository.GetPageDecisionStats(pipelineID)
	if err != nil {
		return fmt.Errorf("检查页面决策状态失败: %w", err)
	}
	if total == 0 {
		return fmt.Errorf("该Pipeline没有生成页面，无法提交定稿")
	}
	if decided < total {
		return fmt.Errorf("%w (总页面: %d, 已决策: %d, 未决策: %d)",
			ErrFinalizeIncomplete, total, decided, total-decided)
	}

	return repository.UpdatePipelineStatus(pipelineID, models.StepReview, models.PipelineStatusPendingFinalize)
}

// ConfirmFinalize 确认定稿
func (s *PipelineService) ConfirmFinalize(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusPendingFinalize {
		return ErrConfirmFinalizeNotAllowed
	}

	total, decided, err := repository.GetPageDecisionStats(pipelineID)
	if err != nil {
		return fmt.Errorf("检查页面决策状态失败: %w", err)
	}

	reviewStep, err := repository.GetStepByName(pipelineID, models.StepReview)
	if err == nil && reviewStep.Status != models.StepStatusDone {
		_ = repository.StartStep(pipelineID, models.StepReview)
		statsJSON := fmt.Sprintf(`{"total_pages":%d,"decided_pages":%d,"finalized_at":"%s"}`,
			total, decided, time.Now().Format(time.RFC3339))
		_ = repository.CompleteStep(pipelineID, models.StepReview, 0, statsJSON, "", 0)
	}

	return repository.CompletePipeline(pipelineID, models.PipelineStatusFinalized)
}

// RejectFinalize 退回定稿
func (s *PipelineService) RejectFinalize(pipelineID string, rejectReason string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	if pipeline.Status != models.PipelineStatusPendingFinalize {
		return ErrRejectFinalizeNotAllowed
	}
	return repository.UpdatePipelineRejectReason(pipelineID, rejectReason)
}

// FinalizePipeline 直接定稿（兼容旧API）
func (s *PipelineService) FinalizePipeline(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue:     true,
		models.PipelineStatusNeedsHuman:      true,
		models.PipelineStatusPendingFinalize: true,
	}
	if !allowedStatuses[pipeline.Status] {
		return ErrPipelineNotReviewable
	}

	total, decided, err := repository.GetPageDecisionStats(pipelineID)
	if err != nil {
		return fmt.Errorf("检查页面决策状态失败: %w", err)
	}
	if total == 0 {
		return fmt.Errorf("该Pipeline没有生成页面，无法定稿")
	}
	if decided < total {
		return fmt.Errorf("%w (总页面: %d, 已决策: %d, 未决策: %d)",
			ErrFinalizeIncomplete, total, decided, total-decided)
	}

	reviewStep, err := repository.GetStepByName(pipelineID, models.StepReview)
	if err == nil && reviewStep.Status != models.StepStatusDone {
		_ = repository.StartStep(pipelineID, models.StepReview)
		statsJSON := fmt.Sprintf(`{"total_pages":%d,"decided_pages":%d,"finalized_at":"%s"}`,
			total, decided, time.Now().Format(time.RFC3339))
		_ = repository.CompleteStep(pipelineID, models.StepReview, 0, statsJSON, "", 0)
	}

	return repository.CompletePipeline(pipelineID, models.PipelineStatusFinalized)
}

// ==================== 高分早停辅助方法 ====================

// fillEarlyStopPages 高分早停时将所有原始页面以 keep 操作写入 generated_pages
func (s *PipelineService) fillEarlyStopPages(pipeline *models.Pipeline) {
	if pipeline.ReviewRound >= 2 {
		return
	}

	_ = repository.DeleteGeneratedPagesByPipelineID(pipeline.ID)

	course, err := repository.GetCourseByCode(pipeline.CourseCode)
	if err != nil || course.ExternalModuleID == nil || *course.ExternalModuleID == 0 {
		return
	}
	moduleID := *course.ExternalModuleID

	ossService := NewOSSService(s.cfg)
	pageLessonMap, err := ossService.BuildPageLessonMap(moduleID)
	if err != nil || len(pageLessonMap) == 0 {
		return
	}

	pageNums := make([]int, 0, len(pageLessonMap))
	for pn := range pageLessonMap {
		pageNums = append(pageNums, pn)
	}
	for i := 0; i < len(pageNums); i++ {
		for j := i + 1; j < len(pageNums); j++ {
			if pageNums[i] > pageNums[j] {
				pageNums[i], pageNums[j] = pageNums[j], pageNums[i]
			}
		}
	}

	for _, pn := range pageNums {
		lessonID := pageLessonMap[pn]
		origHTML, fetchErr := ossService.FetchLessonHTML(lessonID)
		if fetchErr != nil || len(origHTML) < 100 {
			origHTML = ""
		}
		lidPtr := new(int)
		*lidPtr = lessonID
		pageTitle := fmt.Sprintf("P%02d", pn)
		_ = repository.CreateGeneratedPage(
			pipeline.ID, pn, pageTitle,
			"keep", origHTML, "", origHTML,
			lidPtr, "", "高分早停：原始课件质量已达标",
		)
	}
}

// shouldEarlyStop 检查是否满足高分早停条件
func (s *PipelineService) shouldEarlyStop(pipelineID string, pipeline *models.Pipeline) bool {
	if pipeline == nil {
		return false
	}
	evalStep, err := repository.GetStepByName(pipelineID, models.StepEvaluator)
	if err != nil || evalStep.Status != models.StepStatusDone {
		return false
	}
	if evalStep.StepData == "" || evalStep.StepData == "null" {
		return false
	}
	var evalData map[string]interface{}
	if err := json.Unmarshal([]byte(evalStep.StepData), &evalData); err != nil {
		return false
	}
	avgTotal, ok := evalData["avg_total"].(float64)
	if !ok || avgTotal <= 0 {
		return false
	}
	pCfg := models.ParsePipelineConfig(pipeline.Config)
	return avgTotal >= pCfg.Threshold
}

// getEvalAvgScore 获取Evaluator的均分
func (s *PipelineService) getEvalAvgScore(pipelineID string) float64 {
	evalStep, err := repository.GetStepByName(pipelineID, models.StepEvaluator)
	if err != nil || evalStep.StepData == "" || evalStep.StepData == "null" {
		return 0.0
	}
	var evalData map[string]interface{}
	if err := json.Unmarshal([]byte(evalStep.StepData), &evalData); err != nil {
		return 0.0
	}
	if avg, ok := evalData["avg_total"].(float64); ok {
		return avg
	}
	return 0.0
}

// GetOperatorUsers 获取所有操作员用户列表
func (s *PipelineService) GetOperatorUsers() ([]map[string]string, error) {
	return repository.ListOperatorUsers()
}

// ==================== 发布至课程平台 ====================

// ErrPublishNotVerified 只有验收通过的Pipeline才能发布
var ErrPublishNotVerified = fmt.Errorf("只有验收通过(verified)的Pipeline才能发布至课程平台")

// ErrPublishAlreadyDone 已发布过的Pipeline不能重复发布
var ErrPublishAlreadyDone = fmt.Errorf("该Pipeline已发布至课程平台，不可重复操作")

// PublishPipeline 发布Pipeline至课程平台（单向不可逆）
func (s *PipelineService) PublishPipeline(pipelineID string) error {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return ErrPipelineNotFound
	}

	if pipeline.Status != models.PipelineStatusVerified {
		return ErrPublishNotVerified
	}

	if err := repository.UpdatePipelineStatus(
		pipelineID,
		models.StepVerify,
		models.PipelineStatusPublished,
	); err != nil {
		return fmt.Errorf("更新Pipeline发布状态失败: %w", err)
	}

	return nil
}

// ==================== 单页HTML按需加载（v69新增，编号8方案2）====================

// GetSinglePageHTML 获取单页完整HTML数据
// v69新增：供审核页前端按需加载，选中页面时才请求完整HTML
func (s *PipelineService) GetSinglePageHTML(pipelineID string, pageNumber int) (*repository.GeneratedPageFullRow, error) {
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}
	page, err := repository.GetSinglePageHTML(pipelineID, pageNumber)
	if err != nil {
		return nil, fmt.Errorf("获取页面P%d HTML失败: %w", pageNumber, err)
	}
	return page, nil
}

// GetGeneratedPagesLightweight 获取所有页面轻量元数据（不含HTML内容）
// v69新增：审核页首次加载只获取元数据列表，大幅减少传输数据量
func (s *PipelineService) GetGeneratedPagesLightweight(pipelineID string) ([]*repository.GeneratedPageFullRow, error) {
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}
	pages, err := repository.GetGeneratedPagesLightweight(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("获取页面元数据失败: %w", err)
	}
	if pages == nil {
		pages = []*repository.GeneratedPageFullRow{}
	}
	return pages, nil
}

