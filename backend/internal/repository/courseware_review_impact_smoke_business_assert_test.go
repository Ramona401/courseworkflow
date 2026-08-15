package repository

import (
	"context"
	"strings"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
)

func assertR07SmokeMixedBusinessResult(
	t *testing.T,
	ctx context.Context,
	fixture *r07SmokeFixture,
	planID string,
	itemA *models.CoursewareReviewItem,
	itemB *models.CoursewareReviewItem,
	groupName string,
	relationOperation models.CoursewareReviewImpactOperation,
	candidateOperation models.CoursewareReviewImpactOperation,
	createItemOperation models.CoursewareReviewImpactOperation,
) {
	t.Helper()

	var groupID string
	var primaryItem string

	err := database.DB.QueryRow(
		ctx,
		`SELECT
			 id::text,
			 COALESCE(primary_item_id::text, '')
		 FROM courseware_review_item_groups
		 WHERE source_session_id = $1
		   AND created_by = $2
		   AND name = $3
		   AND status = 'active'`,
		fixture.sessionID,
		fixture.actorID,
		groupName,
	).Scan(
		&groupID,
		&primaryItem,
	)
	if err != nil {
		t.Fatalf("成功plan未创建问题组: %v", err)
	}

	if primaryItem != itemA.ID {
		t.Fatalf(
			"问题组主问题=%s，期望=%s",
			primaryItem,
			itemA.ID,
		)
	}

	var groupMembers int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_review_item_group_members
		 WHERE group_id = $1
		   AND item_id = $2
		   AND status = 'active'`,
		groupID,
		itemA.ID,
	).Scan(&groupMembers); err != nil {
		t.Fatalf("检查问题组成员失败: %v", err)
	}

	if groupMembers != 1 {
		t.Fatalf(
			"问题组active成员数量=%d，期望1",
			groupMembers,
		)
	}

	var groupEvents int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_review_item_group_events
		 WHERE group_id = $1`,
		groupID,
	).Scan(&groupEvents); err != nil {
		t.Fatalf("检查问题组事件失败: %v", err)
	}

	if groupEvents < 3 {
		t.Fatalf(
			"问题组事件过少=%d，至少应有created/member_added/primary_changed",
			groupEvents,
		)
	}

	relationSourceID := r07SmokePayloadString(
		t,
		relationOperation,
		"source_item_id",
	)
	relationTargetID := r07SmokePayloadString(
		t,
		relationOperation,
		"target_item_id",
	)
	relationType := r07SmokePayloadString(
		t,
		relationOperation,
		"relation_type",
	)

	var relationID string

	err = database.DB.QueryRow(
		ctx,
		`SELECT id::text
		 FROM courseware_review_item_relations
		 WHERE source_session_id = $1
		   AND created_by = $2
		   AND source_item_id = $3
		   AND target_item_id = $4
		   AND relation_type = $5
		   AND status = 'active'`,
		fixture.sessionID,
		fixture.actorID,
		relationSourceID,
		relationTargetID,
		relationType,
	).Scan(&relationID)
	if err != nil {
		t.Fatalf("成功plan未创建relation: %v", err)
	}

	var relationEvents int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_review_item_relation_events
		 WHERE relation_id = $1`,
		relationID,
	).Scan(&relationEvents); err != nil {
		t.Fatalf("检查relation事件失败: %v", err)
	}

	if relationEvents < 1 {
		t.Fatal("relation缺少追加式事件")
	}

	var itemBStatus string
	var confirmedInstruction string

	if err := database.DB.QueryRow(
		ctx,
		`SELECT
			 status,
			 confirmed_instruction
		 FROM courseware_review_items
		 WHERE id = $1`,
		itemB.ID,
	).Scan(
		&itemBStatus,
		&confirmedInstruction,
	); err != nil {
		t.Fatalf("读取被dismiss整改项失败: %v", err)
	}

	if itemBStatus != "dismissed" {
		t.Fatalf(
			"dismiss结果status=%s，期望dismissed",
			itemBStatus,
		)
	}

	if confirmedInstruction != "" {
		t.Fatalf(
			"candidate update错误改写confirmed_instruction=%q",
			confirmedInstruction,
		)
	}

	candidateInstruction := r07SmokePayloadString(
		t,
		candidateOperation,
		"candidate_instruction",
	)

	var candidateMessages int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_ai_review_messages AS message
		 WHERE message.session_id = $1
		   AND message.review_item_id = $2
		   AND message.role = 'assistant'
		   AND message.citations_json->>'suggested_instruction' = $3
		   AND EXISTS (
			 SELECT 1
			 FROM jsonb_array_elements(
				 COALESCE(
					 message.citations_json->'citations',
					 '[]'::jsonb
				 )
			 ) AS citation
			 WHERE citation->>'impact_plan_id' = $4
			   AND citation->>'impact_operation_id' = $5
		   )`,
		fixture.sessionID,
		itemB.ID,
		candidateInstruction,
		planID,
		candidateOperation.OperationID,
	).Scan(&candidateMessages); err != nil {
		t.Fatalf("检查候选建议消息失败: %v", err)
	}

	if candidateMessages != 1 {
		t.Fatalf(
			"候选建议消息数量=%d，期望1",
			candidateMessages,
		)
	}

	var dismissEvents int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_ai_review_messages
		 WHERE session_id = $1
		   AND review_item_id = $2
		   AND role = 'system'
		   AND citations_json->>'impact_plan_id' = $3
		   AND citations_json->>'event' = 'dismissed'`,
		fixture.sessionID,
		itemB.ID,
		planID,
	).Scan(&dismissEvents); err != nil {
		t.Fatalf("检查dismiss审计消息失败: %v", err)
	}

	if dismissEvents != 1 {
		t.Fatalf(
			"dismiss审计消息数量=%d，期望1",
			dismissEvents,
		)
	}

	createdFindingID :=
		"impact_" +
			strings.ReplaceAll(
				createItemOperation.OperationID,
				"-",
				"",
			)

	var createdStatus string
	var createdInstruction string
	var createdPageID string

	err = database.DB.QueryRow(
		ctx,
		`SELECT
			 status,
			 confirmed_instruction,
			 COALESCE(page_id::text, '')
		 FROM courseware_review_items
		 WHERE source_session_id = $1
		   AND source_finding_id = $2`,
		fixture.sessionID,
		createdFindingID,
	).Scan(
		&createdStatus,
		&createdInstruction,
		&createdPageID,
	)
	if err != nil {
		t.Fatalf("读取create_item结果失败: %v", err)
	}

	if createdStatus != "detected" ||
		createdInstruction != "" ||
		createdPageID != "" {
		t.Fatalf(
			"create_item安全边界错误 status=%s instruction=%q page=%q",
			createdStatus,
			createdInstruction,
			createdPageID,
		)
	}
}

func assertR07SmokeMixedRollback(
	t *testing.T,
	ctx context.Context,
	fixture *r07SmokeFixture,
	planID string,
	itemA *models.CoursewareReviewItem,
	itemB *models.CoursewareReviewItem,
	groupName string,
	relationOperation models.CoursewareReviewImpactOperation,
	candidateOperation models.CoursewareReviewImpactOperation,
	createItemOperation models.CoursewareReviewImpactOperation,
) {
	t.Helper()

	assertR07SmokeGroupAbsent(
		t,
		ctx,
		fixture,
		groupName,
	)

	relationSourceID := r07SmokePayloadString(
		t,
		relationOperation,
		"source_item_id",
	)
	relationTargetID := r07SmokePayloadString(
		t,
		relationOperation,
		"target_item_id",
	)
	relationType := r07SmokePayloadString(
		t,
		relationOperation,
		"relation_type",
	)

	var relationCount int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_review_item_relations
		 WHERE source_session_id = $1
		   AND created_by = $2
		   AND source_item_id = $3
		   AND target_item_id = $4
		   AND relation_type = $5`,
		fixture.sessionID,
		fixture.actorID,
		relationSourceID,
		relationTargetID,
		relationType,
	).Scan(&relationCount); err != nil {
		t.Fatalf("检查回滚relation失败: %v", err)
	}

	if relationCount != 0 {
		t.Fatalf(
			"mixed stale错误写入relation count=%d",
			relationCount,
		)
	}

	var candidateCount int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_ai_review_messages AS message
		 WHERE message.session_id = $1
		   AND message.review_item_id = $2
		   AND message.role = 'assistant'
		   AND EXISTS (
			 SELECT 1
			 FROM jsonb_array_elements(
				 COALESCE(
					 message.citations_json->'citations',
					 '[]'::jsonb
				 )
			 ) AS citation
			 WHERE citation->>'impact_plan_id' = $3
			   AND citation->>'impact_operation_id' = $4
		   )`,
		fixture.sessionID,
		itemB.ID,
		planID,
		candidateOperation.OperationID,
	).Scan(&candidateCount); err != nil {
		t.Fatalf("检查回滚candidate失败: %v", err)
	}

	if candidateCount != 0 {
		t.Fatalf(
			"mixed stale错误写入candidate count=%d",
			candidateCount,
		)
	}

	createdFindingID :=
		"impact_" +
			strings.ReplaceAll(
				createItemOperation.OperationID,
				"-",
				"",
			)

	var createItemCount int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_review_items
		 WHERE source_session_id = $1
		   AND source_finding_id = $2`,
		fixture.sessionID,
		createdFindingID,
	).Scan(&createItemCount); err != nil {
		t.Fatalf("检查回滚create_item失败: %v", err)
	}

	if createItemCount != 0 {
		t.Fatalf(
			"mixed stale错误写入create_item count=%d",
			createItemCount,
		)
	}

	var groupedA int

	if err := database.DB.QueryRow(
		ctx,
		`SELECT COUNT(*)
		 FROM courseware_review_item_group_members AS member
		 INNER JOIN courseware_review_item_groups AS review_group
		    ON review_group.id = member.group_id
		 WHERE review_group.source_session_id = $1
		   AND review_group.created_by = $2
		   AND member.item_id = $3
		   AND member.status = 'active'`,
		fixture.sessionID,
		fixture.actorID,
		itemA.ID,
	).Scan(&groupedA); err != nil {
		t.Fatalf("检查回滚group member失败: %v", err)
	}

	if groupedA != 0 {
		t.Fatalf(
			"mixed stale错误写入group membership count=%d",
			groupedA,
		)
	}

	var itemAStatus string
	var itemBStatus string

	if err := database.DB.QueryRow(
		ctx,
		`SELECT status
		 FROM courseware_review_items
		 WHERE id = $1`,
		itemA.ID,
	).Scan(&itemAStatus); err != nil {
		t.Fatalf("读取stale回滚itemA失败: %v", err)
	}

	if err := database.DB.QueryRow(
		ctx,
		`SELECT status
		 FROM courseware_review_items
		 WHERE id = $1`,
		itemB.ID,
	).Scan(&itemBStatus); err != nil {
		t.Fatalf("读取stale回滚itemB失败: %v", err)
	}

	if itemAStatus != "detected" {
		t.Fatalf(
			"stale失败后itemA状态被意外修改为%s",
			itemAStatus,
		)
	}

	if itemBStatus != "discussing" {
		t.Fatalf(
			"stale失败后itemB状态=%s，期望保留测试前制造的discussing",
			itemBStatus,
		)
	}
}
