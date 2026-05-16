package services

// pipeline_review_batch.go — Pipeline批量操作+分配服务
//
// 从 pipeline_review.go 拆分,包含:
//   - BatchCreatePipelines/BatchStartPipelines
//   - AssignPipeline/BatchAssignPipelines

import (
	"fmt"
	"strings"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 批量创建+批量启动 ====================

// BatchCreateResult 批量创建结果
type BatchCreateResult struct {
	TotalRequested int      `json:"total_requested"`
	CreatedIDs     []string `json:"created_ids"`
	SkippedCodes   []string `json:"skipped_codes"`
	SkippedReasons []string `json:"skipped_reasons"`
	FailedCodes    []string `json:"failed_codes"`
	FailedReasons  []string `json:"failed_reasons"`
}

// BatchCreatePipelines 批量创建Pipeline
func (s *PipelineService) BatchCreatePipelines(courseCodes []string, userID string) (*BatchCreateResult, error) {
	result := &BatchCreateResult{
		TotalRequested: len(courseCodes), CreatedIDs: []string{},
		SkippedCodes: []string{}, SkippedReasons: []string{},
		FailedCodes: []string{}, FailedReasons: []string{},
	}

	seen := make(map[string]bool)
	var uniqueCodes []string
	for _, code := range courseCodes {
		code = strings.TrimSpace(code)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		uniqueCodes = append(uniqueCodes, code)
	}

	for _, code := range uniqueCodes {
		req := &models.CreatePipelineRequest{CourseCode: code}
		resp, err := s.CreatePipeline(req, userID)
		if err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "已有运行中的Pipeline") || strings.Contains(errMsg, "课程不存在") {
				result.SkippedCodes = append(result.SkippedCodes, code)
				result.SkippedReasons = append(result.SkippedReasons, code+": "+errMsg)
			} else {
				result.FailedCodes = append(result.FailedCodes, code)
				result.FailedReasons = append(result.FailedReasons, code+": "+errMsg)
			}
			continue
		}
		result.CreatedIDs = append(result.CreatedIDs, resp.ID)
	}
	return result, nil
}

// BatchStartResult 批量启动结果
type BatchStartResult struct {
	TotalRequested int      `json:"total_requested"`
	StartedIDs     []string `json:"started_ids"`
	SkippedIDs     []string `json:"skipped_ids"`
	SkippedReasons []string `json:"skipped_reasons"`
	FailedIDs      []string `json:"failed_ids"`
	FailedReasons  []string `json:"failed_reasons"`
}

// BatchStartPipelines 批量启动Pipeline
func (s *PipelineService) BatchStartPipelines(ids []string) (*BatchStartResult, error) {
	result := &BatchStartResult{
		TotalRequested: len(ids), StartedIDs: []string{},
		SkippedIDs: []string{}, SkippedReasons: []string{},
		FailedIDs: []string{}, FailedReasons: []string{},
	}

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		_, err := s.StartPipeline(id)
		if err != nil {
			errMsg := err.Error()
			if err == ErrPipelineNotPending || err == ErrPipelineNotFound {
				result.SkippedIDs = append(result.SkippedIDs, id)
				result.SkippedReasons = append(result.SkippedReasons, id+": "+errMsg)
			} else {
				result.FailedIDs = append(result.FailedIDs, id)
				result.FailedReasons = append(result.FailedReasons, id+": "+errMsg)
			}
			continue
		}
		result.StartedIDs = append(result.StartedIDs, id)
	}
	return result, nil
}

// ==================== Pipeline分配 ====================

// AssignPipelineResult 分配结果
type AssignPipelineResult struct {
	PipelineID   string `json:"pipeline_id"`
	AssignedTo   string `json:"assigned_to"`
	AssignedName string `json:"assigned_name"`
}

// BatchAssignResult 批量分配结果
type BatchAssignResult struct {
	TotalRequested int      `json:"total_requested"`
	SuccessCount   int      `json:"success_count"`
	AssignedTo     string   `json:"assigned_to"`
	AssignedName   string   `json:"assigned_name"`
	FailedIDs      []string `json:"failed_ids"`
}

// AssignPipeline 分配Pipeline给指定用户
func (s *PipelineService) AssignPipeline(pipelineID string, assignedToUserID string) (*AssignPipelineResult, error) {
	_, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}

	var assignPtr *string
	if assignedToUserID != "" {
		assignPtr = &assignedToUserID
	}
	if err := repository.AssignPipeline(pipelineID, assignPtr); err != nil {
		return nil, fmt.Errorf("分配失败: %w", err)
	}

	assignedName := repository.GetAssignedUserName(assignedToUserID)
	return &AssignPipelineResult{
		PipelineID: pipelineID, AssignedTo: assignedToUserID, AssignedName: assignedName,
	}, nil
}

// BatchAssignPipelines 批量分配Pipeline
func (s *PipelineService) BatchAssignPipelines(pipelineIDs []string, assignedToUserID string) (*BatchAssignResult, error) {
	var assignPtr *string
	if assignedToUserID != "" {
		assignPtr = &assignedToUserID
	}

	successCount, err := repository.BatchAssignPipelines(pipelineIDs, assignPtr)
	if err != nil {
		return nil, fmt.Errorf("批量分配失败: %w", err)
	}

	assignedName := repository.GetAssignedUserName(assignedToUserID)
	return &BatchAssignResult{
		TotalRequested: len(pipelineIDs), SuccessCount: successCount,
		AssignedTo: assignedToUserID, AssignedName: assignedName, FailedIDs: []string{},
	}, nil
}

