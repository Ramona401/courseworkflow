package repository

// courseware_review_item_group_structural_repo.go
//
// R-06 问题组成员跨组移动仓储。
//
// 设计边界：
//   1. 来源组、目标组和成员都使用显式version做乐观并发；
//   2. 两个问题组按稳定顺序加锁，避免并发移动形成死锁；
//   3. 成员移动只改变group_id和member version，不改变整改项本身；
//   4. 来源组和目标组必须各追加一条member_moved事件；
//   5. 如果被移动成员是来源组主问题，先独立清空主问题并写事件；
//   6. 问题组合并和拆分位于courseware_review_item_group_merge_split_repo.go。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// MoveCoursewareReviewItemGroupMember 原子移动一个成员，并在来源组和目标组各追加事件。
func MoveCoursewareReviewItemGroupMember(
	ctx context.Context,
	sessionID string,
	sourceGroupID string,
	targetGroupID string,
	expectedSourceVersion int,
	expectedTargetVersion int,
	memberID string,
	expectedMemberVersion int,
	actorID string,
	reason string,
) (
	*models.CoursewareReviewItemGroup,
	*models.CoursewareReviewItemGroup,
	*models.CoursewareReviewItemGroupMember,
	error,
) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			nil,
			nil,
			fmt.Errorf(
				"开始移动课件审核问题组成员事务失败: %w",
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
		return nil, nil, nil, err
	}

	source := groups[strings.TrimSpace(sourceGroupID)]
	target := groups[strings.TrimSpace(targetGroupID)]

	if source == nil ||
		target == nil ||
		source.CoursewareID != target.CoursewareID {
		return nil,
			nil,
			nil,
			ErrCoursewareReviewItemGroupConflict
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		source,
		expectedSourceVersion,
	); err != nil {
		return nil, nil, nil, err
	}

	if err := ensureCWReviewItemGroupExpectedVersion(
		target,
		expectedTargetVersion,
	); err != nil {
		return nil, nil, nil, err
	}

	if err := lockCWReviewItemGroupGovernableItemsTx(
		ctx,
		tx,
		source,
	); err != nil {
		return nil, nil, nil, err
	}

	if err := lockCWReviewItemGroupGovernableItemsTx(
		ctx,
		tx,
		target,
	); err != nil {
		return nil, nil, nil, err
	}

	member, err := lockCWReviewItemGroupMemberTx(
		ctx,
		tx,
		memberID,
		sessionID,
		actorID,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	if member.GroupID != source.ID ||
		member.Status != models.CWReviewItemGroupMemberStatusActive ||
		member.Version != expectedMemberVersion {
		return nil,
			nil,
			nil,
			ErrCoursewareReviewItemGroupConflict
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
			return nil, nil, nil, err
		}

		if err := insertCWReviewItemGroupEventTx(
			ctx,
			tx,
			source,
			models.CWReviewItemGroupEventPrimaryChanged,
			actorID,
			"移动主问题前清空来源组主问题",
			nil,
			nil,
			nil,
			map[string]interface{}{
				"old_primary_item_id": member.ItemID,
			},
		); err != nil {
			return nil, nil, nil, err
		}
	}

	source, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		source,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	target, err = bumpCWReviewItemGroupVersionTx(
		ctx,
		tx,
		target,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	member, err = moveCWReviewItemGroupMemberTx(
		ctx,
		tx,
		member,
		target.ID,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	memberVersion := member.Version
	sourceRelated := target.ID
	targetRelated := source.ID

	metadata := map[string]interface{}{
		"item_id": member.ItemID,
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
		return nil, nil, nil, err
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
		return nil, nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			nil,
			nil,
			fmt.Errorf(
				"提交移动课件审核问题组成员事务失败: %w",
				err,
			)
	}

	return source, target, member, nil
}
