package handlers

// courseware_ai_review_impact_plan_views.go
//
// R-07结构化影响方案浏览器安全视图。
//
// 明确不返回：
//   - operations_json原始字符串；
//   - operation.preconditions；
//   - operations_hash；
//   - source_message_hash；
//   - source_message_id；
//   - created_by；
//   - applied_by；
//   - event.actor_id；
//   - 内部锁、页面哈希或身份字段。
//
// Payload是教师预览候选动作所需的业务参数，来自后端冻结方案，
// 前端只能展示，最终Apply不能把Payload回传并作为事实源。

import (
	"encoding/json"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/services"
)

type coursewareAIReviewImpactOperationView struct {
	OperationID   string                 `json:"operation_id"`
	OperationType string                 `json:"operation_type"`
	ActionLabel   string                 `json:"action_label"`
	Summary       string                 `json:"summary"`
	Payload       map[string]interface{} `json:"payload"`
}

type coursewareAIReviewImpactPlanEventView struct {
	PlanVersion int    `json:"plan_version"`
	EventType   string `json:"event_type"`

	SelectedOperationIDs []string `json:"selected_operation_ids"`

	CreatedAt *time.Time `json:"created_at"`
}

type coursewareAIReviewImpactPlanView struct {
	ID              string `json:"id"`
	CoursewareID    string `json:"courseware_id"`
	SourceSessionID string `json:"source_session_id"`

	Status  string `json:"status"`
	Version int    `json:"version"`

	OperationsSchemaVersion int `json:"operations_schema_version"`

	Operations []*coursewareAIReviewImpactOperationView `json:"operations"`

	AppliedOperationIDs []string `json:"applied_operation_ids"`

	CreatedAt *time.Time `json:"created_at"`
	AppliedAt *time.Time `json:"applied_at"`
	UpdatedAt *time.Time `json:"updated_at"`

	Events []*coursewareAIReviewImpactPlanEventView `json:"events"`
}

func buildCoursewareAIReviewImpactPlanView(
	record *services.CWAIReviewImpactPlanRecord,
) *coursewareAIReviewImpactPlanView {
	if record == nil || record.Plan == nil {
		return nil
	}

	plan := record.Plan

	return &coursewareAIReviewImpactPlanView{
		ID:                      plan.ID,
		CoursewareID:            plan.CoursewareID,
		SourceSessionID:         plan.SourceSessionID,
		Status:                  plan.Status,
		Version:                 plan.Version,
		OperationsSchemaVersion: plan.OperationsSchemaVersion,
		Operations: buildCoursewareAIReviewImpactOperationViews(
			record.Operations,
		),
		AppliedOperationIDs: parseCoursewareAIReviewImpactStringArray(
			plan.AppliedOperationIDsJSON,
		),
		CreatedAt: plan.CreatedAt,
		AppliedAt: plan.AppliedAt,
		UpdatedAt: plan.UpdatedAt,
		Events: buildCoursewareAIReviewImpactPlanEventViews(
			record.Events,
		),
	}
}

func buildCoursewareAIReviewImpactOperationViews(
	operations []models.CoursewareReviewImpactOperation,
) []*coursewareAIReviewImpactOperationView {
	result := make(
		[]*coursewareAIReviewImpactOperationView,
		0,
		len(operations),
	)

	for _, operation := range operations {
		payload := operation.Payload
		if payload == nil {
			payload = map[string]interface{}{}
		}

		result = append(
			result,
			&coursewareAIReviewImpactOperationView{
				OperationID:   operation.OperationID,
				OperationType: operation.OperationType,
				ActionLabel: coursewareAIReviewImpactActionLabel(
					operation.OperationType,
				),
				Summary: operation.Summary,
				Payload: payload,
			},
		)
	}

	return result
}

func buildCoursewareAIReviewImpactPlanEventViews(
	events []*models.CoursewareReviewImpactPlanEvent,
) []*coursewareAIReviewImpactPlanEventView {
	result := make(
		[]*coursewareAIReviewImpactPlanEventView,
		0,
		len(events),
	)

	for _, event := range events {
		if event == nil {
			continue
		}

		result = append(
			result,
			&coursewareAIReviewImpactPlanEventView{
				PlanVersion: event.PlanVersion,
				EventType:   event.EventType,
				SelectedOperationIDs: parseCoursewareAIReviewImpactStringArray(
					event.SelectedOperationIDsJSON,
				),
				CreatedAt: event.CreatedAt,
			},
		)
	}

	return result
}

func parseCoursewareAIReviewImpactStringArray(
	raw string,
) []string {
	result := make([]string, 0)

	if err := json.Unmarshal(
		[]byte(strings.TrimSpace(raw)),
		&result,
	); err != nil {
		return []string{}
	}

	if result == nil {
		return []string{}
	}

	return result
}

func coursewareAIReviewImpactActionLabel(
	operationType string,
) string {
	switch strings.TrimSpace(operationType) {
	case models.CWReviewImpactOperationCreateGroup,
		models.CWReviewImpactOperationCreateItem,
		models.CWReviewImpactOperationCreateRelation:
		return "将新增"

	case models.CWReviewImpactOperationMoveGroupMember:
		return "将调整分组"

	case models.CWReviewImpactOperationMergeGroups:
		return "将合并"

	case models.CWReviewImpactOperationSplitGroup:
		return "将拆分"

	case models.CWReviewImpactOperationCancelRelation:
		return "将取消关系"

	case models.CWReviewImpactOperationDismissItem:
		return "将暂不处理"

	case models.CWReviewImpactOperationUpdateCandidateSuggestion:
		return "将更新建议"

	default:
		return "将调整"
	}
}
