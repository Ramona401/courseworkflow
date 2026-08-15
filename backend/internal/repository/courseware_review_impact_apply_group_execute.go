package repository

// courseware_review_impact_apply_group_execute.go
//
// R-07 Atomic Apply问题组操作执行层第一部分：
//   - create_group
//   - move_group_member
//
// 所有precondition已经由validate文件在任何业务写入前全部验证。
// 本文件直接复用R-06私有Tx helper，不开启嵌套事务。

import (
	"context"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
)

func applyCoursewareReviewImpactGroupOperationsTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operations []cwReviewImpactPreparedGroupOperation,
	actorID string,
) error {
	for _, operation := range operations {
		switch operation.OperationType {
		case models.CWReviewImpactOperationCreateGroup:
			if err := applyImpactCreateGroupTx(
				ctx,
				tx,
				plan,
				operation.CreateGroup,
				actorID,
			); err != nil {
				return err
			}

		case models.CWReviewImpactOperationMoveGroupMember:
			if err := applyImpactMoveMemberTx(
				ctx,
				tx,
				plan,
				operation.MoveMember,
				actorID,
			); err != nil {
				return err
			}

		case models.CWReviewImpactOperationMergeGroups:
			if err := applyImpactMergeGroupsTx(
				ctx,
				tx,
				plan,
				operation.MergeGroups,
				actorID,
			); err != nil {
				return err
			}

		case models.CWReviewImpactOperationSplitGroup:
			if err := applyImpactSplitGroupTx(
				ctx,
				tx,
				plan,
				operation.SplitGroup,
				actorID,
			); err != nil {
				return err
			}

		default:
			return ErrCoursewareReviewImpactOperationUnsupported
		}
	}

	return nil
}

func applyImpactCreateGroupTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedCreateGroup,
	actorID string,
) error {
	if value == nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	group, err := insertCWReviewItemGroupTx(
		ctx,
		tx,
		plan.CoursewareID,
		plan.SourceSessionID,
		value.Payload.Name,
		actorID,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		group,
		models.CWReviewItemGroupEventCreated,
		actorID,
		value.Payload.Reason,
		nil,
		nil,
		nil,
		map[string]interface{}{
			"name":           group.Name,
			"impact_plan_id": plan.ID,
		},
	); err != nil {
		return err
	}

	for _, itemID := range value.Payload.ItemIDs {
		group, err = bumpCWReviewItemGroupVersionTx(
			ctx,
			tx,
			group,
		)
		if err != nil {
			return mapCoursewareReviewImpactGroupError(err)
		}

		member, err := attachCWReviewItemGroupMemberTx(
			ctx,
			tx,
			group,
			itemID,
			actorID,
		)
		if err != nil {
			return mapCoursewareReviewImpactGroupError(err)
		}

		memberID := member.ID
		memberVersion := member.Version

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			group,
			models.CWReviewItemGroupEventMemberAdded,
			actorID,
			value.Payload.Reason,
			&memberID,
			&memberVersion,
			nil,
			map[string]interface{}{
				"item_id":        member.ItemID,
				"impact_plan_id": plan.ID,
			},
		); err != nil {
			return err
		}
	}

	if value.Payload.PrimaryItemID != "" {
		group, err = setCWReviewItemGroupPrimaryTx(
			ctx,
			tx,
			group,
			&value.Payload.PrimaryItemID,
		)
		if err != nil {
			return mapCoursewareReviewImpactGroupError(err)
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			group,
			models.CWReviewItemGroupEventPrimaryChanged,
			actorID,
			value.Payload.Reason,
			nil,
			nil,
			nil,
			map[string]interface{}{
				"primary_item_id": value.Payload.PrimaryItemID,
				"impact_plan_id":  plan.ID,
			},
		); err != nil {
			return err
		}
	}

	return nil
}

func applyImpactMoveMemberTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedMoveMember,
	actorID string,
) error {
	if value == nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	pre := value.Preconditions

	groups, err := lockCWReviewItemGroupPairTx(
		ctx,
		tx,
		pre.SourceGroup.GroupID,
		pre.TargetGroup.GroupID,
		plan.SourceSessionID,
		actorID,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	source := groups[pre.SourceGroup.GroupID]
	target := groups[pre.TargetGroup.GroupID]

	if err := ensureCWReviewItemGroupExpectedVersion(
		source,
		pre.SourceGroup.Version,
	); err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		target,
		pre.TargetGroup.Version,
	); err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	member, err := lockCWReviewItemGroupMemberTx(
		ctx,
		tx,
		pre.Member.MemberID,
		plan.SourceSessionID,
		actorID,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	if member.GroupID != source.ID ||
		member.Version != pre.Member.Version ||
		member.Status !=
			models.CWReviewItemGroupMemberStatusActive {
		return ErrCoursewareReviewImpactPlanConflict
	}

	if source.PrimaryItemID != nil &&
		*source.PrimaryItemID == member.ItemID {
		source, err = setCWReviewItemGroupPrimaryTx(
			ctx,
			tx,
			source,
			nil,
		)
		if err != nil {
			return mapCoursewareReviewImpactGroupError(err)
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			source,
			models.CWReviewItemGroupEventPrimaryChanged,
			actorID,
			"影响方案移动主问题前清空来源组主问题",
			nil,
			nil,
			nil,
			map[string]interface{}{
				"old_primary_item_id": member.ItemID,
				"impact_plan_id":      plan.ID,
			},
		); err != nil {
			return err
		}
	}

	source, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		source,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	target, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		target,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	member, err = moveCWReviewItemGroupMemberTx(
		ctx,
		tx,
		member,
		target.ID,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	memberVersion := member.Version
	sourceRelated := target.ID
	targetRelated := source.ID

	metadata := map[string]interface{}{
		"item_id":        member.ItemID,
		"impact_plan_id": plan.ID,
	}

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		source,
		models.CWReviewItemGroupEventMemberMoved,
		actorID,
		value.Payload.Reason,
		&member.ID,
		&memberVersion,
		&sourceRelated,
		metadata,
	); err != nil {
		return err
	}

	return insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		target,
		models.CWReviewItemGroupEventMemberMoved,
		actorID,
		value.Payload.Reason,
		&member.ID,
		&memberVersion,
		&targetRelated,
		metadata,
	)
}
