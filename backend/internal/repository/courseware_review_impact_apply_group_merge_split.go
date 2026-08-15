package repository

// courseware_review_impact_apply_group_merge_split.go
//
// R-07 Atomic Apply问题组操作执行层第二部分：
//   - merge_groups
//   - split_group
//
// 直接复用R-06成熟的group/member/event私有Tx helper。
// 所有写入共享Impact Plan外层pgx.Tx，因此任何失败都会整体回滚。

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
)

func applyImpactMergeGroupsTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedMergeGroups,
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

	sourceMembers, err := lockCWReviewItemGroupMembersTx(
		ctx,
		tx,
		source,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	if len(sourceMembers) != len(pre.SourceMembers) {
		return ErrCoursewareReviewImpactPlanConflict
	}

	frozenMemberByID := make(
		map[string]cwReviewImpactMemberPrecondition,
		len(pre.SourceMembers),
	)

	for _, member := range pre.SourceMembers {
		frozenMemberByID[member.MemberID] = member
	}

	for _, member := range sourceMembers {
		frozen, exists := frozenMemberByID[member.ID]
		if !exists ||
			member.Version != frozen.Version ||
			member.ItemID != frozen.ItemID ||
			member.Status != frozen.Status {
			return ErrCoursewareReviewImpactPlanConflict
		}
	}

	if source.PrimaryItemID != nil {
		oldPrimary := *source.PrimaryItemID

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
			"影响方案合并前清空来源组主问题",
			nil,
			nil,
			nil,
			map[string]interface{}{
				"old_primary_item_id": oldPrimary,
				"impact_plan_id":      plan.ID,
			},
		); err != nil {
			return err
		}
	}

	for _, member := range sourceMembers {
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
			"via":            "impact_plan_merge",
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

		if err := insertCWReviewItemGroupEventTx(
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
		); err != nil {
			return err
		}
	}

	source, err = mergeCWReviewItemGroupTx(
		ctx,
		tx,
		source,
		target.ID,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	targetID := target.ID

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		source,
		models.CWReviewItemGroupEventMerged,
		actorID,
		value.Payload.Reason,
		nil,
		nil,
		&targetID,
		map[string]interface{}{
			"target_group_id": target.ID,
			"impact_plan_id":  plan.ID,
		},
	); err != nil {
		return err
	}

	target, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		target,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	sourceID := source.ID

	return insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		target,
		models.CWReviewItemGroupEventMerged,
		actorID,
		value.Payload.Reason,
		nil,
		nil,
		&sourceID,
		map[string]interface{}{
			"source_group_id": source.ID,
			"impact_plan_id":  plan.ID,
		},
	)
}

func applyImpactSplitGroupTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedSplitGroup,
	actorID string,
) error {
	if value == nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	pre := value.Preconditions

	source, err := lockCWReviewItemGroupTx(
		ctx,
		tx,
		pre.SourceGroup.GroupID,
		plan.SourceSessionID,
		actorID,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		source,
		pre.SourceGroup.Version,
	); err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	sourceMembers, err := lockCWReviewItemGroupMembersTx(
		ctx,
		tx,
		source,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	if len(sourceMembers) < 2 ||
		len(value.Payload.ItemIDs) >= len(sourceMembers) {
		return ErrCoursewareReviewImpactPlanConflict
	}

	selected := make(
		map[string]struct{},
		len(value.Payload.ItemIDs),
	)

	for _, itemID := range value.Payload.ItemIDs {
		selected[itemID] = struct{}{}
	}

	memberByItem := make(
		map[string]*models.CoursewareReviewItemGroupMember,
		len(sourceMembers),
	)

	for _, member := range sourceMembers {
		memberByItem[member.ItemID] = member
	}

	for _, itemID := range value.Payload.ItemIDs {
		if memberByItem[itemID] == nil {
			return ErrCoursewareReviewImpactPlanConflict
		}
	}

	target, err := insertCWReviewItemGroupTx(
		ctx,
		tx,
		source.CoursewareID,
		source.SourceSessionID,
		value.Payload.Name,
		actorID,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		target,
		models.CWReviewItemGroupEventCreated,
		actorID,
		"影响方案拆分创建新问题组",
		nil,
		nil,
		nil,
		map[string]interface{}{
			"name":           target.Name,
			"impact_plan_id": plan.ID,
		},
	); err != nil {
		return err
	}

	if source.PrimaryItemID != nil {
		if _, movingPrimary :=
			selected[*source.PrimaryItemID]; movingPrimary {
			oldPrimary := *source.PrimaryItemID

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
				"影响方案拆分前清空来源组主问题",
				nil,
				nil,
				nil,
				map[string]interface{}{
					"old_primary_item_id": oldPrimary,
					"impact_plan_id":      plan.ID,
				},
			); err != nil {
				return err
			}
		}
	}

	for _, member := range sourceMembers {
		if _, move := selected[member.ItemID]; !move {
			continue
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
			"via":            "impact_plan_split",
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

		if err := insertCWReviewItemGroupEventTx(
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
		); err != nil {
			return err
		}
	}

	if value.Payload.PrimaryItemID != "" {
		target, err = setCWReviewItemGroupPrimaryTx(
			ctx,
			tx,
			target,
			&value.Payload.PrimaryItemID,
		)
		if err != nil {
			return mapCoursewareReviewImpactGroupError(err)
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			target,
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

	source, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		source,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	targetID := target.ID

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		source,
		models.CWReviewItemGroupEventSplit,
		actorID,
		value.Payload.Reason,
		nil,
		nil,
		&targetID,
		map[string]interface{}{
			"new_group_id":   target.ID,
			"impact_plan_id": plan.ID,
		},
	); err != nil {
		return err
	}

	target, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		target,
	)
	if err != nil {
		return mapCoursewareReviewImpactGroupError(err)
	}

	sourceID := source.ID

	return insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		target,
		models.CWReviewItemGroupEventSplit,
		actorID,
		value.Payload.Reason,
		nil,
		nil,
		&sourceID,
		map[string]interface{}{
			"source_group_id": source.ID,
			"impact_plan_id":  plan.ID,
		},
	)
}

func mapCoursewareReviewImpactGroupError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		ErrCoursewareReviewItemGroupConflict,
	),
		errors.Is(
			err,
			ErrCoursewareReviewItemGroupNotFound,
		),
		errors.Is(
			err,
			ErrCoursewareReviewItemGroupMemberNotFound,
		),
		errors.Is(
			err,
			ErrCoursewareReviewItemConflict,
		),
		errors.Is(
			err,
			ErrCoursewareReviewItemNotFound,
		):
		return ErrCoursewareReviewImpactPlanConflict

	default:
		return err
	}
}
