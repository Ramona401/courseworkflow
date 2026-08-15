package repository

// courseware_review_impact_apply_item_relation_execute.go
//
// R-07 Atomic Apply第二部分业务执行器。
//
// 执行顺序固定为：
//   1. create_item
//   2. update_candidate_suggestion
//   3. create_relation
//   4. cancel_relation
//   5. dismiss_item
//
// 这样候选建议和关系在同一份教师确认方案中可以先形成审计事实，
// dismiss_item最后执行，不会因为本事务自己刚把item置dismissed而阻断其他已验证动作。
//
// 所有操作共享Impact Plan外层pgx.Tx。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
)

func applyCoursewareReviewImpactItemRelationOperationsTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operations []cwReviewImpactPreparedItemRelationOperation,
	actorID string,
) error {
	executionOrder := []string{
		models.CWReviewImpactOperationCreateItem,
		models.CWReviewImpactOperationUpdateCandidateSuggestion,
		models.CWReviewImpactOperationCreateRelation,
		models.CWReviewImpactOperationCancelRelation,
		models.CWReviewImpactOperationDismissItem,
	}

	for _, operationType := range executionOrder {
		for index := range operations {
			operation := &operations[index]

			if operation.OperationType != operationType {
				continue
			}

			switch operation.OperationType {
			case models.CWReviewImpactOperationCreateItem:
				if err := applyImpactCreateItemTx(
					ctx,
					tx,
					plan,
					operation,
					actorID,
				); err != nil {
					return err
				}

			case models.CWReviewImpactOperationUpdateCandidateSuggestion:
				if err := applyImpactUpdateCandidateSuggestionTx(
					ctx,
					tx,
					plan,
					operation,
				); err != nil {
					return err
				}

			case models.CWReviewImpactOperationCreateRelation:
				if err := applyImpactCreateRelationTx(
					ctx,
					tx,
					plan,
					operation,
					actorID,
				); err != nil {
					return err
				}

			case models.CWReviewImpactOperationCancelRelation:
				if err := applyImpactCancelRelationTx(
					ctx,
					tx,
					plan,
					operation,
					actorID,
				); err != nil {
					return err
				}

			case models.CWReviewImpactOperationDismissItem:
				if err := applyImpactDismissItemTx(
					ctx,
					tx,
					plan,
					operation,
					actorID,
				); err != nil {
					return err
				}

			default:
				return ErrCoursewareReviewImpactOperationUnsupported
			}
		}
	}

	return nil
}

func applyImpactCreateItemTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operation *cwReviewImpactPreparedItemRelationOperation,
	actorID string,
) error {
	value := operation.CreateItem
	if value == nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	sourceMessageID := plan.SourceMessageID
	sourceFindingID :=
		"impact_" +
			strings.ReplaceAll(
				operation.OperationID,
				"-",
				"",
			)

	evidence := map[string]interface{}{
		"origin_type":              models.CWReviewItemOriginGlobalDiscussionManual,
		"source_global_message_id": plan.SourceMessageID,
		"impact_plan_id":           plan.ID,
		"impact_operation_id":      operation.OperationID,
		"scope":                    value.Preconditions.Page.Scope,
	}

	item := &models.CoursewareReviewItem{
		CoursewareID:    plan.CoursewareID,
		SourceSessionID: plan.SourceSessionID,
		SourceFindingID: sourceFindingID,

		OriginType:            models.CWReviewItemOriginGlobalDiscussionManual,
		SourceGlobalMessageID: &sourceMessageID,

		SourceType:  value.SourceType,
		ReviewLevel: value.ReviewLevel,
		ReviewRound: 0,

		CreatedBy: actorID,
		OwnerID:   value.OwnerID,

		Severity:  value.Payload.Severity,
		Dimension: value.Payload.Dimension,

		Title:       value.Payload.Title,
		Description: value.Payload.Description,

		OriginalSuggestion: value.Payload.CandidateInstruction,
		Status:             models.CWReviewItemStatusDetected,
	}

	if value.Preconditions.Page.Scope == "page" {
		pageID := value.Preconditions.Page.PageID

		item.PageID = &pageID
		item.PageNumberSnapshot =
			value.Preconditions.Page.PageNumber
		item.PageTitleSnapshot =
			value.Preconditions.Page.Title
		item.PageHTMLHash =
			value.Preconditions.Page.HTMLHash
		item.PageUpdatedAtSnapshot =
			value.PageUpdatedAt

		evidence["page_id"] = pageID
		evidence["page_number_snapshot"] =
			value.Preconditions.Page.PageNumber
	}

	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		return fmt.Errorf(
			"序列化影响方案新增整改项证据失败: %w",
			err,
		)
	}
	item.EvidenceJSON = string(evidenceJSON)

	if err := CreateCoursewareReviewItemTx(
		ctx,
		tx,
		item,
	); err != nil {
		return mapCoursewareReviewImpactItemRelationError(
			err,
		)
	}

	auditJSON, err := json.Marshal(
		map[string]interface{}{
			"event":                    "impact_plan_item_created",
			"impact_plan_id":           plan.ID,
			"impact_operation_id":      operation.OperationID,
			"source_global_message_id": plan.SourceMessageID,
			"origin_type":              models.CWReviewItemOriginGlobalDiscussionManual,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"序列化影响方案新增整改项审计失败: %w",
			err,
		)
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO courseware_ai_review_messages (
			session_id,
			review_item_id,
			user_id,
			role,
			content,
			citations_json,
			tokens_used,
			model_used,
			created_at
		 )
		 VALUES (
			$1,
			$2,
			$3,
			'system',
			$4,
			$5::jsonb,
			0,
			'',
			NOW()
		 )`,
		plan.SourceSessionID,
		item.ID,
		strings.TrimSpace(actorID),
		"已按教师确认的全局讨论影响方案新增此整改项。"+
			"候选修改建议仍需独立确认。",
		string(auditJSON),
	)
	if err != nil {
		return fmt.Errorf(
			"记录影响方案新增整改项审计失败: %w",
			err,
		)
	}

	return nil
}

func applyImpactUpdateCandidateSuggestionTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operation *cwReviewImpactPreparedItemRelationOperation,
) error {
	value := operation.UpdateCandidate
	if value == nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	citations := []map[string]interface{}{
		{
			"type":                      "global_discussion",
			"global_discussion_message": plan.SourceMessageID,
			"impact_plan_id":            plan.ID,
			"impact_operation_id":       operation.OperationID,
		},
	}

	metaJSON, err := json.Marshal(
		map[string]interface{}{
			"summary":                "已按教师确认的全局讨论影响方案更新候选修改建议。",
			"ready_for_confirmation": true,
			"suggested_instruction":  value.Payload.CandidateInstruction,
			"citations":              citations,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"序列化影响方案候选修改建议失败: %w",
			err,
		)
	}

	result, err := tx.Exec(
		ctx,
		`INSERT INTO courseware_ai_review_messages (
			session_id,
			review_item_id,
			user_id,
			role,
			content,
			citations_json,
			tokens_used,
			model_used,
			created_at
		 )
		 SELECT
			$1,
			$2,
			NULL,
			'assistant',
			$3,
			$4::jsonb,
			0,
			source.model_used,
			NOW()
		 FROM courseware_ai_review_messages AS source
		 WHERE source.id = $5
		   AND source.session_id = $1
		   AND source.review_item_id IS NULL
		   AND source.role = 'assistant'`,
		plan.SourceSessionID,
		value.Payload.ItemID,
		"已按教师确认的全局讨论影响方案更新此候选修改建议。"+
			"该内容尚未独立确认，也不会自动修改页面或改变审核决定。",
		string(metaJSON),
		plan.SourceMessageID,
	)
	if err != nil {
		return fmt.Errorf(
			"保存影响方案候选修改建议失败: %w",
			err,
		)
	}

	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewImpactPlanConflict
	}

	// 此操作刻意不UPDATE courseware_review_items，
	// 因而不会修改status、confirmed_instruction或任何确认版本引用。
	return nil
}

func applyImpactCreateRelationTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operation *cwReviewImpactPreparedItemRelationOperation,
	actorID string,
) error {
	value := operation.CreateRelation
	if value == nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	sourceMessageID := plan.SourceMessageID

	relation := &models.CoursewareReviewItemRelation{
		CoursewareID:          plan.CoursewareID,
		SourceSessionID:       plan.SourceSessionID,
		SourceItemID:          value.Payload.SourceItemID,
		TargetItemID:          value.Payload.TargetItemID,
		RelationType:          value.Payload.RelationType,
		Explanation:           value.Payload.Explanation,
		SourceGlobalMessageID: &sourceMessageID,
		CreatedBy:             actorID,
	}

	normalizeCWReviewItemRelation(relation)

	if value.CurrentRelation == nil {
		current, err :=
			insertCoursewareReviewItemRelationTx(
				ctx,
				tx,
				relation,
			)
		if err != nil {
			return mapCoursewareReviewImpactItemRelationError(
				err,
			)
		}

		return insertCoursewareReviewItemRelationEventTx(
			ctx,
			tx,
			current,
			"confirmed",
			actorID,
			current.Explanation,
			&sourceMessageID,
		)
	}

	current, err :=
		reactivateCoursewareReviewItemRelationTx(
			ctx,
			tx,
			value.CurrentRelation,
			value.Payload.Explanation,
		)
	if err != nil {
		return mapCoursewareReviewImpactItemRelationError(
			err,
		)
	}

	return insertCoursewareReviewItemRelationEventTx(
		ctx,
		tx,
		current,
		"reactivated",
		actorID,
		value.Payload.Explanation,
		&sourceMessageID,
	)
}

func applyImpactCancelRelationTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operation *cwReviewImpactPreparedItemRelationOperation,
	actorID string,
) error {
	value := operation.CancelRelation
	if value == nil ||
		value.CurrentRelation == nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	current, err :=
		cancelCoursewareReviewItemRelationTx(
			ctx,
			tx,
			value.CurrentRelation,
			actorID,
		)
	if err != nil {
		return mapCoursewareReviewImpactItemRelationError(
			err,
		)
	}

	sourceMessageID := plan.SourceMessageID

	return insertCoursewareReviewItemRelationEventTx(
		ctx,
		tx,
		current,
		"cancelled",
		actorID,
		value.Payload.Reason,
		&sourceMessageID,
	)
}

func applyImpactDismissItemTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operation *cwReviewImpactPreparedItemRelationOperation,
	actorID string,
) error {
	value := operation.DismissItem
	if value == nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	previousStatus :=
		value.Preconditions.Item.Status

	result, err := tx.Exec(
		ctx,
		`UPDATE courseware_review_items
		 SET
			status = 'dismissed',
			updated_at = NOW()
		 WHERE id = $1
		   AND courseware_id = $2
		   AND source_session_id = $3
		   AND status = $4
		   AND (
				(
					source_type = 'formal'
					AND created_by = $5
				)
				OR
				(
					source_type = 'self'
					AND owner_id = $5
				)
		   )
		   AND courseware_review_id IS NULL
		   AND feedback_id IS NULL
		   AND delivered_instruction_version_id IS NULL
		   AND applied_instruction_version_id IS NULL
		   AND applied_at IS NULL`,
		value.Payload.ItemID,
		plan.CoursewareID,
		plan.SourceSessionID,
		previousStatus,
		strings.TrimSpace(actorID),
	)
	if err != nil {
		return fmt.Errorf(
			"应用影响方案暂不处理整改项失败: %w",
			err,
		)
	}

	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewImpactPlanConflict
	}

	metaJSON, err := json.Marshal(
		map[string]interface{}{
			"event":                    "dismissed",
			"previous_status":          previousStatus,
			"next_status":              "dismissed",
			"reason":                   value.Payload.Reason,
			"impact_plan_id":           plan.ID,
			"impact_operation_id":      operation.OperationID,
			"source_global_message_id": plan.SourceMessageID,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"序列化影响方案暂不处理审计失败: %w",
			err,
		)
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO courseware_ai_review_messages (
			session_id,
			review_item_id,
			user_id,
			role,
			content,
			citations_json,
			tokens_used,
			model_used,
			created_at
		 )
		 VALUES (
			$1,
			$2,
			$3,
			'system',
			$4,
			$5::jsonb,
			0,
			'',
			NOW()
		 )`,
		plan.SourceSessionID,
		value.Payload.ItemID,
		strings.TrimSpace(actorID),
		"已按教师确认的全局讨论影响方案将此整改项标记为“暂不处理”。原因："+value.Payload.Reason,
		string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf(
			"记录影响方案暂不处理审计失败: %w",
			err,
		)
	}

	return nil
}
