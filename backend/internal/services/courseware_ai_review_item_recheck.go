package services

// courseware_ai_review_item_recheck.go
//
// 页面变化问题由作者重新检查后的业务入口。
//
// stale表示页面在问题形成或修改完成后又发生了变化。
// 它不能直接恢复，也不能自动关闭。
//
// 作者必须明确执行“重新检查当前页面”：
//
//   - 系统重新读取当前页面；
//   - 作者确认当前页面仍满足既有修改方案或正式整改要求；
//   - 当前页面指纹重新登记为修改完成结果；
//   - 状态回到applied；
//   - 自审问题等待作者最终确认；
//   - 正式问题等待审核员复审。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// RecheckCWReviewItem 由课件作者重新检查一个页面变化问题。
func (s *CoursewareAIReviewService) RecheckCWReviewItem(
	ctx context.Context,
	itemID string,
	actor *CoursewareActorContext,
) (
	*CWReviewItemDiscussionResult,
	error,
) {
	item, _, err :=
		loadAuthorizedCWReviewItem(
			ctx,
			itemID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" {
		return nil,
			ErrCWAIReviewActorRequired
	}

	if actor.UserID !=
		item.OwnerID {
		return nil,
			ErrCWAIReviewNoPermission
	}

	switch item.SourceType {
	case models.CWReviewItemSourceSelf:
		if item.CoursewareReviewID != nil ||
			item.FeedbackID != nil {
			return nil,
				ErrCWReviewItemNotActionable
		}

	case models.CWReviewItemSourceFormal:
		if item.CoursewareReviewID == nil ||
			item.FeedbackID == nil {
			return nil,
				ErrCWReviewItemNotDelivered
		}

	default:
		return nil,
			ErrCWReviewItemNotActionable
	}

	if item.Status !=
		models.CWReviewItemStatusStale ||
		item.PageID == nil ||
		strings.TrimSpace(
			*item.PageID,
		) == "" ||
		strings.TrimSpace(
			item.ConfirmedInstruction,
		) == "" {
		return nil,
			ErrCWReviewItemNotActionable
	}

	err =
		repository.RecheckCoursewareReviewItem(
			ctx,
			item.ID,
			actor.UserID,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrCoursewareReviewItemAppliedPageMissing,
		):
			return nil,
				ErrCWReviewItemOrphaned

		case errors.Is(
			err,
			repository.ErrCoursewareReviewItemRecheckInvalid,
		):
			return nil,
				ErrCWReviewItemNotActionable

		default:
			return nil, err
		}
	}

	updatedItem, err :=
		repository.GetCoursewareReviewItemForParticipant(
			ctx,
			item.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return buildCWReviewItemDiscussionResult(
		ctx,
		updatedItem,
	)
}
