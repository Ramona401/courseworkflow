package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
)

func newR07SmokeItemPrecondition(
	t *testing.T,
	item *models.CoursewareReviewItem,
) map[string]interface{} {
	t.Helper()

	fingerprint, err := coursewareReviewImpactItemFingerprint(item)
	if err != nil {
		t.Fatalf("生成smoke整改项指纹失败: %v", err)
	}

	return map[string]interface{}{
		"item_id":     item.ID,
		"status":      item.Status,
		"fingerprint": fingerprint,
	}
}

func newR07SmokeCreateGroupOperation(
	t *testing.T,
	item *models.CoursewareReviewItem,
	name string,
	reason string,
) models.CoursewareReviewImpactOperation {
	t.Helper()

	return models.CoursewareReviewImpactOperation{
		OperationID:   newR07SmokeOperationUUID(t),
		OperationType: models.CWReviewImpactOperationCreateGroup,
		Summary:       "smoke：建立正式问题组",
		Payload: map[string]interface{}{
			"name":            name,
			"item_ids":        []string{item.ID},
			"primary_item_id": item.ID,
			"reason":          reason,
		},
		Preconditions: map[string]interface{}{
			"items": []map[string]interface{}{
				newR07SmokeItemPrecondition(t, item),
			},
		},
	}
}

func newR07SmokeCreateRelationOperation(
	t *testing.T,
	source *models.CoursewareReviewItem,
	target *models.CoursewareReviewItem,
	relationType string,
	explanation string,
) models.CoursewareReviewImpactOperation {
	t.Helper()

	return models.CoursewareReviewImpactOperation{
		OperationID:   newR07SmokeOperationUUID(t),
		OperationType: models.CWReviewImpactOperationCreateRelation,
		Summary:       "smoke：建立问题关系",
		Payload: map[string]interface{}{
			"relation_type":  relationType,
			"source_item_id": source.ID,
			"target_item_id": target.ID,
			"explanation":    explanation,
		},
		Preconditions: map[string]interface{}{
			"source_item": newR07SmokeItemPrecondition(t, source),
			"target_item": newR07SmokeItemPrecondition(t, target),
			"relation": map[string]interface{}{
				"expected_absent": true,
			},
		},
	}
}

func newR07SmokeUpdateCandidateOperation(
	t *testing.T,
	item *models.CoursewareReviewItem,
	instruction string,
) models.CoursewareReviewImpactOperation {
	t.Helper()

	return models.CoursewareReviewImpactOperation{
		OperationID:   newR07SmokeOperationUUID(t),
		OperationType: models.CWReviewImpactOperationUpdateCandidateSuggestion,
		Summary:       "smoke：更新候选建议",
		Payload: map[string]interface{}{
			"item_id":               item.ID,
			"candidate_instruction": instruction,
		},
		Preconditions: map[string]interface{}{
			"item": newR07SmokeItemPrecondition(t, item),
		},
	}
}

func newR07SmokeCreateItemOperation(
	t *testing.T,
	title string,
) models.CoursewareReviewImpactOperation {
	t.Helper()

	return models.CoursewareReviewImpactOperation{
		OperationID:   newR07SmokeOperationUUID(t),
		OperationType: models.CWReviewImpactOperationCreateItem,
		Summary:       "smoke：新增独立整改项",
		Payload: map[string]interface{}{
			"page_id":               "",
			"severity":              "medium",
			"dimension":             "manual_review",
			"title":                 title,
			"description":           "R-07 Atomic Apply隔离smoke新增问题",
			"candidate_instruction": "smoke候选建议，仍需独立确认",
		},
		Preconditions: map[string]interface{}{
			"page": map[string]interface{}{
				"scope": "global",
			},
		},
	}
}

func newR07SmokeDismissOperation(
	t *testing.T,
	item *models.CoursewareReviewItem,
	reason string,
) models.CoursewareReviewImpactOperation {
	t.Helper()

	return models.CoursewareReviewImpactOperation{
		OperationID:   newR07SmokeOperationUUID(t),
		OperationType: models.CWReviewImpactOperationDismissItem,
		Summary:       "smoke：暂不处理问题",
		Payload: map[string]interface{}{
			"item_id": item.ID,
			"reason":  reason,
		},
		Preconditions: map[string]interface{}{
			"item": newR07SmokeItemPrecondition(t, item),
		},
	}
}

func newR07SmokeOperationUUID(
	t *testing.T,
) string {
	t.Helper()

	var value string

	if err := database.DB.QueryRow(
		context.Background(),
		`SELECT gen_random_uuid()::text`,
	).Scan(&value); err != nil {
		t.Fatalf("生成smoke operation UUID失败: %v", err)
	}

	return value
}

func r07SmokeOperationIDs(
	operations []models.CoursewareReviewImpactOperation,
) []string {
	result := make([]string, 0, len(operations))

	for _, operation := range operations {
		result = append(result, operation.OperationID)
	}

	return result
}

func r07SmokePayloadString(
	t *testing.T,
	operation models.CoursewareReviewImpactOperation,
	key string,
) string {
	t.Helper()

	raw, exists := operation.Payload[key]
	if !exists {
		t.Fatalf(
			"operation=%s缺少payload字段%s",
			operation.OperationID,
			key,
		)
	}

	value, ok := raw.(string)
	if !ok {
		t.Fatalf(
			"operation=%s payload字段%s不是string",
			operation.OperationID,
			key,
		)
	}

	return value
}

func assertR07SmokePlanApplied(
	t *testing.T,
	ctx context.Context,
	fixture *r07SmokeFixture,
	plan *models.CoursewareReviewImpactPlan,
	expectedIDs []string,
) {
	t.Helper()

	if plan == nil {
		t.Fatal("applied plan为空")
	}

	if plan.Status != models.CWReviewImpactPlanStatusApplied ||
		plan.Version != 2 {
		t.Fatalf(
			"plan终态错误 status=%s version=%d",
			plan.Status,
			plan.Version,
		)
	}

	var actualIDs []string

	if err := json.Unmarshal(
		[]byte(plan.AppliedOperationIDsJSON),
		&actualIDs,
	); err != nil {
		t.Fatalf("解析applied operation IDs失败: %v", err)
	}

	if strings.Join(actualIDs, "|") != strings.Join(expectedIDs, "|") {
		t.Fatalf(
			"applied IDs不匹配 actual=%v expected=%v",
			actualIDs,
			expectedIDs,
		)
	}

	events, err := ListCoursewareReviewImpactPlanEvents(
		ctx,
		plan.ID,
		fixture.sessionID,
		fixture.actorID,
	)
	if err != nil {
		t.Fatalf("读取plan事件失败: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("plan事件数量=%d，期望2", len(events))
	}

	if events[0].PlanVersion != 1 ||
		events[0].EventType != models.CWReviewImpactPlanEventDraftCreated ||
		events[1].PlanVersion != 2 ||
		events[1].EventType != models.CWReviewImpactPlanEventApplied {
		t.Fatalf("plan事件链不正确: %+v", events)
	}
}

func assertR07SmokePlanStillDraft(
	t *testing.T,
	ctx context.Context,
	fixture *r07SmokeFixture,
	planID string,
) {
	t.Helper()

	plan, err := GetCoursewareReviewImpactPlanByID(
		ctx,
		planID,
		fixture.sessionID,
		fixture.actorID,
	)
	if err != nil {
		t.Fatalf("重新读取draft plan失败: %v", err)
	}

	if plan.Status != models.CWReviewImpactPlanStatusDraft ||
		plan.Version != 1 {
		t.Fatalf(
			"失败后plan被错误推进 status=%s version=%d",
			plan.Status,
			plan.Version,
		)
	}

	events, err := ListCoursewareReviewImpactPlanEvents(
		ctx,
		planID,
		fixture.sessionID,
		fixture.actorID,
	)
	if err != nil {
		t.Fatalf("读取失败plan事件失败: %v", err)
	}

	if len(events) != 1 ||
		events[0].PlanVersion != 1 ||
		events[0].EventType != models.CWReviewImpactPlanEventDraftCreated {
		t.Fatalf("失败plan事件链被污染: %+v", events)
	}
}

func assertR07SmokeGroupAbsent(
	t *testing.T,
	ctx context.Context,
	fixture *r07SmokeFixture,
	groupName string,
) {
	t.Helper()

	var count int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_review_item_groups
		 WHERE source_session_id = $1
		   AND created_by = $2
		   AND name = $3`,
		fixture.sessionID,
		fixture.actorID,
		groupName,
	).Scan(&count); err != nil {
		t.Fatalf("检查问题组不存在失败: %v", err)
	}

	if count != 0 {
		t.Fatalf(
			"失败事务错误写入问题组 count=%d name=%s",
			count,
			groupName,
		)
	}
}
