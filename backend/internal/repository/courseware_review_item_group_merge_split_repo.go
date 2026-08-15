package repository

// courseware_review_item_group_merge_split_repo.go
//
// R-06 问题组合并与拆分仓储。
//
// 设计边界：
//   1. 合并和拆分都在单一数据库事务内完成；
//   2. 所有涉及的问题组、成员和整改项都必须重新锁定；
//   3. 每个成员移动同时递增来源组、目标组和成员的治理版本；
//   4. 来源组和目标组分别写入追加式member_moved事件；
//   5. 合并后来源组冻结为merged，不物理删除历史；
//   6. 拆分创建新组，来源组必须至少保留一个有效成员。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// MergeCoursewareReviewItemGroups 将来源组全部有效成员移入目标组，并冻结来源组为merged。
func MergeCoursewareReviewItemGroups(
	ctx context.Context,
	sessionID string,
	sourceGroupID string,
	targetGroupID string,
	expectedSourceVersion int,
	expectedTargetVersion int,
	actorID string,
	reason string,
) (
	*models.CoursewareReviewItemGroup,
	*models.CoursewareReviewItemGroup,
	error,
) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"开始合并课件审核问题组事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	groups, err := lockCWReviewItemGroupPairTx(
		ctx,
		tx,
		sourceGroupID,
		targetGroupID,
		sessionID,
		actorID,
	)
	if err != nil {
		return nil, nil, err
	}

	source := groups[strings.TrimSpace(sourceGroupID)]
	target := groups[strings.TrimSpace(targetGroupID)]

	if source == nil ||
		target == nil ||
		source.CoursewareID != target.CoursewareID {
		return nil,
			nil,
			ErrCoursewareReviewItemGroupConflict
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		source,
		expectedSourceVersion,
	); err != nil {
		return nil, nil, err
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		target,
		expectedTargetVersion,
	); err != nil {
		return nil, nil, err
	}

	sourceMembers, err := lockCWReviewItemGroupMembersTx(
		ctx,
		tx,
		source,
	)
	if err != nil {
		return nil, nil, err
	}

	if err := lockCWReviewItemGroupGovernableItemsTx(
		ctx,
		tx,
		source,
	); err != nil {
		return nil, nil, err
	}

	if err := lockCWReviewItemGroupGovernableItemsTx(
		ctx,
		tx,
		target,
	); err != nil {
		return nil, nil, err
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
			return nil, nil, err
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			source,
			models.CWReviewItemGroupEventPrimaryChanged,
			actorID,
			"合并前清空来源组主问题",
			nil,
			nil,
			nil,
			map[string]interface{}{
				"old_primary_item_id": oldPrimary,
			},
		); err != nil {
			return nil, nil, err
		}
	}

	for _, member := range sourceMembers {
		source, err = bumpCWReviewItemGroupVersionTx(
			ctx,
			tx,
			source,
		)
		if err != nil {
			return nil, nil, err
		}

		target, err = bumpCWReviewItemGroupVersionTx(
			ctx,
			tx,
			target,
		)
		if err != nil {
			return nil, nil, err
		}

		member, err = moveCWReviewItemGroupMemberTx(
			ctx,
			tx,
			member,
			target.ID,
		)
		if err != nil {
			return nil, nil, err
		}

		memberVersion := member.Version
		sourceRelated := target.ID
		targetRelated := source.ID

		metadata := map[string]interface{}{
			"item_id": member.ItemID,
			"via":     "merge",
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			source,
			models.CWReviewItemGroupEventMemberMoved,
			actorID,
			strings.TrimSpace(reason),
			&member.ID,
			&memberVersion,
			&sourceRelated,
			metadata,
		); err != nil {
			return nil, nil, err
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			target,
			models.CWReviewItemGroupEventMemberMoved,
			actorID,
			strings.TrimSpace(reason),
			&member.ID,
			&memberVersion,
			&targetRelated,
			metadata,
		); err != nil {
			return nil, nil, err
		}
	}

	source, err = mergeCWReviewItemGroupTx(
		ctx,
		tx,
		source,
		target.ID,
	)
	if err != nil {
		return nil, nil, err
	}

	targetID := target.ID

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		source,
		models.CWReviewItemGroupEventMerged,
		actorID,
		strings.TrimSpace(reason),
		nil,
		nil,
		&targetID,
		map[string]interface{}{
			"target_group_id": target.ID,
		},
	); err != nil {
		return nil, nil, err
	}

	target, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		target,
	)
	if err != nil {
		return nil, nil, err
	}

	sourceID := source.ID

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		target,
		models.CWReviewItemGroupEventMerged,
		actorID,
		strings.TrimSpace(reason),
		nil,
		nil,
		&sourceID,
		map[string]interface{}{
			"source_group_id": source.ID,
		},
	); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"提交合并课件审核问题组事务失败: %w",
				err,
			)
	}

	return source, target, nil
}

// SplitCoursewareReviewItemGroup 从来源组移动指定成员创建新组，来源组至少保留一名成员。
func SplitCoursewareReviewItemGroup(
	ctx context.Context,
	sessionID string,
	sourceGroupID string,
	expectedSourceVersion int,
	name string,
	itemIDs []string,
	primaryItemID string,
	actorID string,
	reason string,
) (
	*models.CoursewareReviewItemGroup,
	*models.CoursewareReviewItemGroup,
	error,
) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"开始拆分课件审核问题组事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	source, err := lockCWReviewItemGroupTx(
		ctx,
		tx,
		sourceGroupID,
		sessionID,
		actorID,
	)
	if err != nil {
		return nil, nil, err
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		source,
		expectedSourceVersion,
	); err != nil {
		return nil, nil, err
	}

	sourceMembers, err := lockCWReviewItemGroupMembersTx(
		ctx,
		tx,
		source,
	)
	if err != nil {
		return nil, nil, err
	}

	if len(sourceMembers) < 2 {
		return nil,
			nil,
			ErrCoursewareReviewItemGroupConflict
	}

	if err := lockCWReviewItemGroupItemIDsTx(
		ctx,
		tx,
		memberItemIDs(sourceMembers),
		source.CoursewareID,
		source.SourceSessionID,
		source.CreatedBy,
	); err != nil {
		return nil, nil, err
	}

	selected := make(
		map[string]struct{},
		len(itemIDs),
	)

	for _, raw := range itemIDs {
		itemID := strings.TrimSpace(raw)
		if itemID == "" {
			return nil,
				nil,
				ErrCoursewareReviewItemGroupConflict
		}

		selected[itemID] = struct{}{}
	}

	if len(selected) == 0 ||
		len(selected) >= len(sourceMembers) {
		return nil,
			nil,
			ErrCoursewareReviewItemGroupConflict
	}

	memberByItem := make(
		map[string]*models.CoursewareReviewItemGroupMember,
		len(sourceMembers),
	)

	for _, member := range sourceMembers {
		memberByItem[member.ItemID] = member
	}

	for itemID := range selected {
		if memberByItem[itemID] == nil {
			return nil,
				nil,
				ErrCoursewareReviewItemGroupConflict
		}
	}

	primaryItemID = strings.TrimSpace(
		primaryItemID,
	)

	if primaryItemID != "" {
		if _, exists := selected[primaryItemID]; !exists {
			return nil,
				nil,
				ErrCoursewareReviewItemGroupConflict
		}
	}

	target, err := insertCWReviewItemGroupTx(
		ctx,
		tx,
		source.CoursewareID,
		source.SourceSessionID,
		strings.TrimSpace(name),
		actorID,
	)
	if err != nil {
		return nil, nil, err
	}

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		target,
		models.CWReviewItemGroupEventCreated,
		actorID,
		"拆分创建新问题组",
		nil,
		nil,
		nil,
		map[string]interface{}{
			"name": target.Name,
		},
	); err != nil {
		return nil, nil, err
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
				return nil, nil, err
			}

			if err := insertCWReviewItemGroupEventTx(
				ctx,
				tx,
				source,
				models.CWReviewItemGroupEventPrimaryChanged,
				actorID,
				"拆分前清空来源组主问题",
				nil,
				nil,
				nil,
				map[string]interface{}{
					"old_primary_item_id": oldPrimary,
				},
			); err != nil {
				return nil, nil, err
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
			return nil, nil, err
		}

		target, err = bumpCWReviewItemGroupVersionTx(
			ctx,
			tx,
			target,
		)
		if err != nil {
			return nil, nil, err
		}

		member, err = moveCWReviewItemGroupMemberTx(
			ctx,
			tx,
			member,
			target.ID,
		)
		if err != nil {
			return nil, nil, err
		}

		memberVersion := member.Version
		sourceRelated := target.ID
		targetRelated := source.ID

		metadata := map[string]interface{}{
			"item_id": member.ItemID,
			"via":     "split",
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			source,
			models.CWReviewItemGroupEventMemberMoved,
			actorID,
			strings.TrimSpace(reason),
			&member.ID,
			&memberVersion,
			&sourceRelated,
			metadata,
		); err != nil {
			return nil, nil, err
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			target,
			models.CWReviewItemGroupEventMemberMoved,
			actorID,
			strings.TrimSpace(reason),
			&member.ID,
			&memberVersion,
			&targetRelated,
			metadata,
		); err != nil {
			return nil, nil, err
		}
	}

	if primaryItemID != "" {
		target, err = setCWReviewItemGroupPrimaryTx(
			ctx,
			tx,
			target,
			&primaryItemID,
		)
		if err != nil {
			return nil, nil, err
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			target,
			models.CWReviewItemGroupEventPrimaryChanged,
			actorID,
			"设置拆分组主问题",
			nil,
			nil,
			nil,
			map[string]interface{}{
				"primary_item_id": primaryItemID,
			},
		); err != nil {
			return nil, nil, err
		}
	}

	source, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		source,
	)
	if err != nil {
		return nil, nil, err
	}

	targetID := target.ID

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		source,
		models.CWReviewItemGroupEventSplit,
		actorID,
		strings.TrimSpace(reason),
		nil,
		nil,
		&targetID,
		map[string]interface{}{
			"new_group_id": target.ID,
		},
	); err != nil {
		return nil, nil, err
	}

	target, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		target,
	)
	if err != nil {
		return nil, nil, err
	}

	sourceID := source.ID

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		target,
		models.CWReviewItemGroupEventSplit,
		actorID,
		strings.TrimSpace(reason),
		nil,
		nil,
		&sourceID,
		map[string]interface{}{
			"source_group_id": source.ID,
		},
	); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"提交拆分课件审核问题组事务失败: %w",
				err,
			)
	}

	return source, target, nil
}

func memberItemIDs(
	members []*models.CoursewareReviewItemGroupMember,
) []string {
	itemIDs := make(
		[]string,
		0,
		len(members),
	)

	for _, member := range members {
		if member == nil {
			continue
		}

		itemIDs = append(
			itemIDs,
			member.ItemID,
		)
	}

	return itemIDs
}
