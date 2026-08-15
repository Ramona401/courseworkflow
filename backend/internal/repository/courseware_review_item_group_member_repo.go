package repository

// courseware_review_item_group_member_repo.go
//
// R-06 正式问题组的成员加入与移除公开仓储入口。
//
// 设计边界：
//   1. 一个整改项在同一会话只保留一个稳定成员身份；
//   2. 移除不删除成员记录，重新加入时复用原ID并递增member version；
//   3. 组version和member version同时作为乐观并发事实；
//   4. 如果移除的是当前主问题，事务内先独立清空主问题并追加事件；
//   5. 跨组移动由courseware_review_item_group_structural_repo.go负责。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// AddCoursewareReviewItemGroupMember 将一个未分组或历史已移除的整改项加入问题组。
func AddCoursewareReviewItemGroupMember(
	ctx context.Context,
	sessionID string,
	groupID string,
	expectedGroupVersion int,
	itemID string,
	actorID string,
) (
	*models.CoursewareReviewItemGroup,
	*models.CoursewareReviewItemGroupMember,
	error,
) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"开始加入课件审核问题组成员事务失败: %w",
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
		return nil, nil, err
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		group,
		expectedGroupVersion,
	); err != nil {
		return nil, nil, err
	}

	if err := lockCWReviewItemGroupGovernableItemsTx(
		ctx,
		tx,
		group,
	); err != nil {
		return nil, nil, err
	}

	itemID = strings.TrimSpace(itemID)

	if err := lockGovernableCWReviewItemTx(
		ctx,
		tx,
		itemID,
		group.CoursewareID,
		group.SourceSessionID,
		actorID,
		true,
	); err != nil {
		return nil, nil, err
	}

	existing, err :=
		lockCWReviewItemGroupMemberByItemTx(
			ctx,
			tx,
			itemID,
			group.CoursewareID,
			group.SourceSessionID,
			actorID,
		)
	if err != nil {
		return nil, nil, err
	}

	if existing != nil &&
		existing.Status ==
			models.CWReviewItemGroupMemberStatusActive {
		if existing.GroupID != group.ID {
			return nil,
				nil,
				ErrCoursewareReviewItemGroupConflict
		}

		if err := tx.Commit(ctx); err != nil {
			return nil,
				nil,
				fmt.Errorf(
					"提交幂等加入问题组事务失败: %w",
					err,
				)
		}

		return group, existing, nil
	}

	group, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		group,
	)
	if err != nil {
		return nil, nil, err
	}

	member, err := attachCWReviewItemGroupMemberTx(
		ctx,
		tx,
		group,
		itemID,
		actorID,
	)
	if err != nil {
		return nil, nil, err
	}

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
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"提交加入课件审核问题组成员事务失败: %w",
				err,
			)
	}

	return group, member, nil
}

// RemoveCoursewareReviewItemGroupMember 从问题组移除成员但保留稳定成员历史。
func RemoveCoursewareReviewItemGroupMember(
	ctx context.Context,
	sessionID string,
	groupID string,
	expectedGroupVersion int,
	memberID string,
	expectedMemberVersion int,
	actorID string,
	reason string,
) (
	*models.CoursewareReviewItemGroup,
	*models.CoursewareReviewItemGroupMember,
	error,
) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"开始移除课件审核问题组成员事务失败: %w",
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
		return nil, nil, err
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		group,
		expectedGroupVersion,
	); err != nil {
		return nil, nil, err
	}

	member, err := lockCWReviewItemGroupMemberTx(
		ctx,
		tx,
		memberID,
		sessionID,
		actorID,
	)
	if err != nil {
		return nil, nil, err
	}

	if member.GroupID != group.ID ||
		member.Status !=
			models.CWReviewItemGroupMemberStatusActive ||
		member.Version != expectedMemberVersion {
		return nil,
			nil,
			ErrCoursewareReviewItemGroupConflict
	}

	if err := lockGovernableCWReviewItemTx(
		ctx,
		tx,
		member.ItemID,
		group.CoursewareID,
		group.SourceSessionID,
		actorID,
		true,
	); err != nil {
		return nil, nil, err
	}

	if group.PrimaryItemID != nil &&
		*group.PrimaryItemID == member.ItemID {
		group, err = setCWReviewItemGroupPrimaryTx(
			ctx,
			tx,
			group,
			nil,
		)
		if err != nil {
			return nil, nil, err
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			group,
			models.CWReviewItemGroupEventPrimaryChanged,
			actorID,
			"移除成员前清空主问题",
			nil,
			nil,
			nil,
			map[string]interface{}{
				"old_primary_item_id": member.ItemID,
			},
		); err != nil {
			return nil, nil, err
		}
	}

	group, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		group,
	)
	if err != nil {
		return nil, nil, err
	}

	member, err = removeCWReviewItemGroupMemberTx(
		ctx,
		tx,
		member,
		actorID,
	)
	if err != nil {
		return nil, nil, err
	}

	memberVersion := member.Version

	if err := insertCWReviewItemGroupEventTx(
		ctx,
		tx,
		group,
		models.CWReviewItemGroupEventMemberRemoved,
		actorID,
		strings.TrimSpace(reason),
		&member.ID,
		&memberVersion,
		nil,
		map[string]interface{}{
			"item_id": member.ItemID,
		},
	); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			nil,
			fmt.Errorf(
				"提交移除课件审核问题组成员事务失败: %w",
				err,
			)
	}

	return group, member, nil
}
