package repository

// courseware_review_impact_apply_relation_validate.go
//
// R-07 Atomic Apply中pairwise relation操作的事务前置验证。
//
// create_relation保持既有正式锁序：
//   endpoint items -> advisory business key -> relation row。
//
// cancel_relation的relation行已经由总控在item锁之前预锁。
// 本文件不执行业务写入。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
)

func validatePreparedCoursewareReviewImpactItemRelationOperationTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operation *cwReviewImpactPreparedItemRelationOperation,
	actorID string,
	items map[string]*models.CoursewareReviewItem,
	prelockedCancelRelations map[string]*models.CoursewareReviewItemRelation,
) error {
	switch operation.OperationType {
	case models.CWReviewImpactOperationCreateRelation:
		return validatePreparedImpactCreateRelationTx(
			ctx,
			tx,
			plan,
			operation.CreateRelation,
			actorID,
			items,
		)

	case models.CWReviewImpactOperationCancelRelation:
		return validatePreparedImpactCancelRelationTx(
			ctx,
			tx,
			plan,
			operation.CancelRelation,
			actorID,
			items,
			prelockedCancelRelations,
		)

	case models.CWReviewImpactOperationCreateItem:
		return validatePreparedImpactCreateItemTx(
			ctx,
			tx,
			plan,
			operation.CreateItem,
			actorID,
		)

	case models.CWReviewImpactOperationDismissItem:
		if operation.DismissItem == nil {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		return validatePreparedImpactItemMutation(
			plan,
			operation.DismissItem.Payload.ItemID,
			operation.DismissItem.Preconditions.Item,
			actorID,
			items,
		)

	case models.CWReviewImpactOperationUpdateCandidateSuggestion:
		if operation.UpdateCandidate == nil {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		return validatePreparedImpactItemMutation(
			plan,
			operation.UpdateCandidate.Payload.ItemID,
			operation.UpdateCandidate.Preconditions.Item,
			actorID,
			items,
		)

	default:
		return ErrCoursewareReviewImpactOperationUnsupported
	}
}

func validatePreparedImpactCreateRelationTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedCreateRelation,
	actorID string,
	items map[string]*models.CoursewareReviewItem,
) error {
	if value == nil ||
		!models.IsCWReviewItemRelationType(value.Payload.RelationType) ||
		value.Payload.SourceItemID == "" ||
		value.Payload.TargetItemID == "" ||
		value.Payload.SourceItemID == value.Payload.TargetItemID ||
		value.Payload.Explanation == "" {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	if value.Payload.SourceItemID != value.Preconditions.SourceItem.ItemID ||
		value.Payload.TargetItemID != value.Preconditions.TargetItem.ItemID {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	sourceItem := items[value.Payload.SourceItemID]
	targetItem := items[value.Payload.TargetItemID]

	if err := validateCoursewareReviewImpactItemPrecondition(
		plan,
		sourceItem,
		value.Preconditions.SourceItem,
		actorID,
	); err != nil {
		return err
	}

	if err := validateCoursewareReviewImpactItemPrecondition(
		plan,
		targetItem,
		value.Preconditions.TargetItem,
		actorID,
	); err != nil {
		return err
	}

	if err := lockGovernableCWReviewItemPairTx(
		ctx,
		tx,
		value.Payload.SourceItemID,
		value.Payload.TargetItemID,
		plan.CoursewareID,
		plan.SourceSessionID,
		actorID,
		true,
	); err != nil {
		return mapCoursewareReviewImpactItemRelationError(err)
	}

	relation := &models.CoursewareReviewItemRelation{
		CoursewareID:    plan.CoursewareID,
		SourceSessionID: plan.SourceSessionID,
		SourceItemID:    value.Payload.SourceItemID,
		TargetItemID:    value.Payload.TargetItemID,
		RelationType:    value.Payload.RelationType,
		Explanation:     value.Payload.Explanation,
		CreatedBy:       actorID,
	}

	sourceMessageID := plan.SourceMessageID
	relation.SourceGlobalMessageID = &sourceMessageID

	normalizeCWReviewItemRelation(relation)

	if err := lockCoursewareReviewImpactRelationBusinessKeyTx(
		ctx,
		tx,
		relation,
	); err != nil {
		return err
	}

	current, currentErr := getCoursewareReviewItemRelationByKeyTx(
		ctx,
		tx,
		relation,
		actorID,
	)

	switch {
	case errors.Is(currentErr, pgx.ErrNoRows):
		if !value.Preconditions.Relation.ExpectedAbsent {
			return ErrCoursewareReviewImpactPlanConflict
		}

		value.CurrentRelation = nil

	case currentErr != nil:
		return currentErr

	default:
		if value.Preconditions.Relation.ExpectedAbsent ||
			current == nil ||
			current.ID != value.Preconditions.Relation.RelationID ||
			current.Status != value.Preconditions.Relation.Status ||
			current.Version != value.Preconditions.Relation.Version {
			return ErrCoursewareReviewImpactPlanConflict
		}

		if current.Status != "cancelled" {
			return ErrCoursewareReviewImpactPlanConflict
		}

		value.CurrentRelation = current
	}

	return nil
}

func validatePreparedImpactCancelRelationTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedCancelRelation,
	actorID string,
	items map[string]*models.CoursewareReviewItem,
	prelockedCancelRelations map[string]*models.CoursewareReviewItemRelation,
) error {
	if value == nil ||
		value.Payload.RelationID == "" ||
		value.Payload.Reason == "" ||
		value.Payload.RelationID != value.Preconditions.Relation.RelationID {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	// Go索引表达式保持在单行，避免换行触发自动分号插入。
	current := prelockedCancelRelations[value.Payload.RelationID]
	if current == nil ||
		current.CoursewareID != plan.CoursewareID ||
		current.SourceSessionID != plan.SourceSessionID ||
		current.Status != "active" ||
		current.Status != value.Preconditions.Relation.Status ||
		current.Version != value.Preconditions.Relation.Version {
		return ErrCoursewareReviewImpactPlanConflict
	}

	if items[current.SourceItemID] == nil ||
		items[current.TargetItemID] == nil {
		return ErrCoursewareReviewImpactPlanConflict
	}

	if err := lockGovernableCWReviewItemPairTx(
		ctx,
		tx,
		current.SourceItemID,
		current.TargetItemID,
		plan.CoursewareID,
		plan.SourceSessionID,
		actorID,
		false,
	); err != nil {
		return mapCoursewareReviewImpactItemRelationError(err)
	}

	value.CurrentRelation = current
	return nil
}

func lockCoursewareReviewImpactRelationBusinessKeyTx(
	ctx context.Context,
	tx pgx.Tx,
	relation *models.CoursewareReviewItemRelation,
) error {
	if relation == nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	key := strings.Join(
		[]string{
			relation.CoursewareID,
			relation.SourceSessionID,
			relation.RelationType,
			relation.SourceItemID,
			relation.TargetItemID,
		},
		"|",
	)

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(
			hashtextextended($1, 0)
		)`,
		key,
	); err != nil {
		return fmt.Errorf(
			"锁定影响方案整改项关系业务键失败: %w",
			err,
		)
	}

	return nil
}

func normalizePreparedImpactCreateRelation(
	value *cwReviewImpactPreparedCreateRelation,
) {
	value.Payload.RelationType =
		strings.TrimSpace(value.Payload.RelationType)
	value.Payload.SourceItemID =
		strings.TrimSpace(value.Payload.SourceItemID)
	value.Payload.TargetItemID =
		strings.TrimSpace(value.Payload.TargetItemID)
	value.Payload.Explanation =
		strings.TrimSpace(value.Payload.Explanation)

	if value.Payload.RelationType == models.CWReviewItemRelationConflict &&
		value.Payload.SourceItemID > value.Payload.TargetItemID {
		value.Payload.SourceItemID,
			value.Payload.TargetItemID =
			value.Payload.TargetItemID,
			value.Payload.SourceItemID
	}

	normalizePreparedImpactItemPrecondition(
		&value.Preconditions.SourceItem,
	)
	normalizePreparedImpactItemPrecondition(
		&value.Preconditions.TargetItem,
	)

	value.Preconditions.Relation.RelationID =
		strings.TrimSpace(value.Preconditions.Relation.RelationID)
	value.Preconditions.Relation.Status =
		strings.TrimSpace(value.Preconditions.Relation.Status)
}

func mapCoursewareReviewImpactItemRelationError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		ErrCoursewareReviewItemConflict,
	),
		errors.Is(
			err,
			ErrCoursewareReviewItemNotFound,
		),
		errors.Is(
			err,
			ErrCoursewareReviewItemRelationNotFound,
		):
		return ErrCoursewareReviewImpactPlanConflict

	default:
		return err
	}
}
