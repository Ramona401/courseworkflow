package repository

// courseware_review_item_group_mutation_repo.go
//
// R-06 正式问题组的组级基础写操作。
//
// 本文件负责：
//   1. 创建问题组并加入初始成员；
//   2. 重命名问题组；
//   3. 设置或清空主问题。
//
// 所有写操作都使用事务、组version CAS以及追加式事件。
// 成员单独加入/移除位于courseware_review_item_group_member_repo.go。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// CreateCoursewareReviewItemGroup 创建问题组，并把初始成员与主问题作为独立版本事件写入。
func CreateCoursewareReviewItemGroup(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	name string,
	itemIDs []string,
	primaryItemID string,
	actorID string,
) (*models.CoursewareReviewItemGroup, error) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始创建课件审核问题组事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	coursewareID = strings.TrimSpace(coursewareID)
	sessionID = strings.TrimSpace(sessionID)
	actorID = strings.TrimSpace(actorID)
	name = strings.TrimSpace(name)
	primaryItemID = strings.TrimSpace(primaryItemID)

	if err := lockCWReviewItemGroupItemIDsTx(
		ctx,
		tx,
		itemIDs,
		coursewareID,
		sessionID,
		actorID,
	); err != nil {
		return nil, err
	}

	group, err := insertCWReviewItemGroupTx(
		ctx,
		tx,
		coursewareID,
		sessionID,
		name,
		actorID,
	)
	if err != nil {
		return nil, err
	}

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		group,
		models.CWReviewItemGroupEventCreated,
		actorID,
		"创建问题组",
		nil,
		nil,
		nil,
		map[string]interface{}{
			"name": group.Name,
		},
	); err != nil {
		return nil, err
	}

	memberByItem := make(
		map[string]*models.CoursewareReviewItemGroupMember,
		len(itemIDs),
	)

	for _, itemID := range itemIDs {
		group, err = bumpCWReviewItemGroupVersionTx(
			ctx,
			tx,
			group,
		)
		if err != nil {
			return nil, err
		}

		member, attachErr := attachCWReviewItemGroupMemberTx(
			ctx,
			tx,
			group,
			itemID,
			actorID,
		)
		if attachErr != nil {
			return nil, attachErr
		}

		memberByItem[member.ItemID] = member

		memberID := member.ID
		memberVersion := member.Version

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			group,
			models.CWReviewItemGroupEventMemberAdded,
			actorID,
			"加入问题组",
			&memberID,
			&memberVersion,
			nil,
			map[string]interface{}{
				"item_id": member.ItemID,
			},
		); err != nil {
			return nil, err
		}
	}

	if primaryItemID != "" {
		if memberByItem[primaryItemID] == nil {
			return nil,
				ErrCoursewareReviewItemGroupConflict
		}

		group, err = setCWReviewItemGroupPrimaryTx(
			ctx,
			tx,
			group,
			&primaryItemID,
		)
		if err != nil {
			return nil, err
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			group,
			models.CWReviewItemGroupEventPrimaryChanged,
			actorID,
			"设置主问题",
			nil,
			nil,
			nil,
			map[string]interface{}{
				"primary_item_id": primaryItemID,
			},
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交创建课件审核问题组事务失败: %w",
			err,
		)
	}

	return group, nil
}

// RenameCoursewareReviewItemGroup 使用组version执行乐观并发重命名。
func RenameCoursewareReviewItemGroup(
	ctx context.Context,
	sessionID string,
	groupID string,
	expectedVersion int,
	name string,
	actorID string,
) (*models.CoursewareReviewItemGroup, error) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始重命名课件审核问题组事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	group, err := lockCWReviewItemGroupTx(
		ctx,
		tx,
		groupID,
		sessionID,
		actorID,
	)
	if err != nil {
		return nil, err
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		group,
		expectedVersion,
	); err != nil {
		return nil, err
	}

	if err := lockCWReviewItemGroupGovernableItemsTx(
		ctx,
		tx,
		group,
	); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)

	if group.Name == name {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf(
				"提交幂等问题组重命名事务失败: %w",
				err,
			)
		}

		return group, nil
	}

	oldName := group.Name

	group, err = renameCWReviewItemGroupTx(
		ctx,
		tx,
		group,
		name,
	)
	if err != nil {
		return nil, err
	}

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		group,
		models.CWReviewItemGroupEventRenamed,
		actorID,
		"重命名问题组",
		nil,
		nil,
		nil,
		map[string]interface{}{
			"old_name": oldName,
			"new_name": group.Name,
		},
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交重命名课件审核问题组事务失败: %w",
			err,
		)
	}

	return group, nil
}

// SetCoursewareReviewItemGroupPrimary 设置或清空主问题。
func SetCoursewareReviewItemGroupPrimary(
	ctx context.Context,
	sessionID string,
	groupID string,
	expectedVersion int,
	primaryItemID *string,
	actorID string,
) (*models.CoursewareReviewItemGroup, error) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始设置课件审核问题组主问题事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	group, err := lockCWReviewItemGroupTx(
		ctx,
		tx,
		groupID,
		sessionID,
		actorID,
	)
	if err != nil {
		return nil, err
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		group,
		expectedVersion,
	); err != nil {
		return nil, err
	}

	if err := lockCWReviewItemGroupGovernableItemsTx(
		ctx,
		tx,
		group,
	); err != nil {
		return nil, err
	}

	var normalizedPrimary *string

	if primaryItemID != nil {
		value := strings.TrimSpace(*primaryItemID)

		if value != "" {
			member, memberErr :=
				lockCWReviewItemGroupMemberByItemTx(
					ctx,
					tx,
					value,
					group.CoursewareID,
					group.SourceSessionID,
					group.CreatedBy,
				)
			if memberErr != nil {
				return nil, memberErr
			}

			if member == nil ||
				member.GroupID != group.ID ||
				member.Status !=
					models.CWReviewItemGroupMemberStatusActive {
				return nil,
					ErrCoursewareReviewItemGroupConflict
			}

			normalizedPrimary = &value
		}
	}

	if sameOptionalString(
		group.PrimaryItemID,
		normalizedPrimary,
	) {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf(
				"提交幂等主问题设置事务失败: %w",
				err,
			)
		}

		return group, nil
	}

	oldPrimary := optionalStringValue(
		group.PrimaryItemID,
	)

	group, err = setCWReviewItemGroupPrimaryTx(
		ctx,
		tx,
		group,
		normalizedPrimary,
	)
	if err != nil {
		return nil, err
	}

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		group,
		models.CWReviewItemGroupEventPrimaryChanged,
		actorID,
		"变更主问题",
		nil,
		nil,
		nil,
		map[string]interface{}{
			"old_primary_item_id": oldPrimary,
			"new_primary_item_id": optionalStringValue(
				normalizedPrimary,
			),
		},
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交设置课件审核问题组主问题事务失败: %w",
			err,
		)
	}

	return group, nil
}
