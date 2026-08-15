package services

// courseware_ai_review_item_state.go
//
// 课件审核整改项的忽略、恢复和作者自审解决确认业务服务。
//
// 设计边界：
//
//   1. 复用现有dismissed状态，不建设第二套排除模型；
//   2. 普通忽略适用于未交付自审项或正式审核草稿项；
//   3. 自审applied额外允许“暂时不处理”，并完整保留最近一次修改完成证据；
//   4. 恢复这类暂存问题时回到applied，而不是伪造回第一次修改前；
//   5. 正式整改项只能由创建该项的审核员在提交决定前操作；
//   6. 已绑定正式反馈的整改项属于不可变审核历史，禁止忽略和恢复；
//   7. 忽略原因通过整改项系统消息保存，不污染AI原始证据；
//   8. 忽略和恢复均不删除讨论记录、确认指令或整改项实体；
//   9. 自审applied相关动作以applied_page_hash重新检查修改完成后的页面；
//  10. 作者自审问题只有到达applied后，才允许作者本人确认解决；
//  11. 页面变化或删除时，问题改为stale或orphaned；
//  12. 正式审核问题不能通过作者自审接口关闭；
//  13. 全局讨论“确认忽略”复用普通未应用问题入口。

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	cwReviewItemMaxDismissReasonRunes = 500

	cwSelfReviewResolutionNote = "作者检查当前课件后确认问题已经解决"
)

var ErrCWReviewItemDismissReasonInvalid = errors.New(
	"忽略原因不能为空或内容过长",
)

// ResolveSelfCWReviewItem 由作者明确确认自己的自审问题已经解决。
func (s *CoursewareAIReviewService) ResolveSelfCWReviewItem(
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

	if item.SourceType !=
		models.CWReviewItemSourceSelf {
		return nil,
			ErrCWReviewItemNotActionable
	}

	if actor.UserID !=
		item.OwnerID {
		return nil,
			ErrCWAIReviewNoPermission
	}

	if item.CoursewareReviewID != nil ||
		item.FeedbackID != nil {
		return nil,
			ErrCWReviewItemNotActionable
	}

	// 重复确认时直接返回当前记录，保持接口幂等。
	if item.Status ==
		models.CWReviewItemStatusResolved {
		return buildCWReviewItemDiscussionResult(
			ctx,
			item,
		)
	}

	if item.Status !=
		models.CWReviewItemStatusApplied ||
		item.AppliedAt == nil ||
		strings.TrimSpace(
			item.AppliedPageHash,
		) == "" {
		return nil,
			ErrCWReviewItemNotActionable
	}

	err =
		repository.ResolveSelfCoursewareReviewItem(
			ctx,
			item.ID,
			actor.UserID,
			cwSelfReviewResolutionNote,
		)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrCoursewareReviewItemAppliedPageChanged,
		):
			return nil,
				ErrCWReviewItemStale

		case errors.Is(
			err,
			repository.ErrCoursewareReviewItemAppliedPageMissing,
		):
			return nil,
				ErrCWReviewItemOrphaned

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

// DismissCWReviewItem 将一个仍可处理且尚未交付的整改项标记为暂不处理。
//
// 自审applied走专用事务：以applied_page_hash检查最近一次修改完成后的页面，
// 并保留applied版本、时间和页面指纹，供后续恢复或继续调整。
func (s *CoursewareAIReviewService) DismissCWReviewItem(
	ctx context.Context,
	itemID string,
	reason string,
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

	if item.SourceType ==
		models.CWReviewItemSourceSelf &&
		item.Status ==
			models.CWReviewItemStatusApplied {
		normalizedReason, err :=
			normalizeCWReviewItemDismissReason(
				reason,
			)
		if err != nil {
			return nil, err
		}

		if err :=
			ensureCWReviewItemStateManageable(
				item,
				actor,
			); err != nil {
			return nil, err
		}

		err =
			repository.DismissAppliedSelfCoursewareReviewItem(
				ctx,
				item.ID,
				actor.UserID,
				normalizedReason,
			)
		if err != nil {
			return nil,
				mapCWReviewSelfPostApplyStateError(
					err,
				)
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

	return dismissCWReviewItem(
		ctx,
		item,
		reason,
		actor,
	)
}

// dismissCWReviewItem 是普通忽略接口和全局讨论“确认忽略”的共享入口。
//
// 调用方可以先完成额外的可信AI建议校验，但最终状态更新必须经过本函数，
// 不能直接调用仓储绕过权限、未交付和页面新鲜度边界。
func dismissCWReviewItem(
	ctx context.Context,
	item *models.CoursewareReviewItem,
	reason string,
	actor *CoursewareActorContext,
) (
	*CWReviewItemDiscussionResult,
	error,
) {
	normalizedReason, err :=
		normalizeCWReviewItemDismissReason(
			reason,
		)
	if err != nil {
		return nil, err
	}
	reason = normalizedReason

	if err :=
		ensureCWReviewItemStateManageable(
			item,
			actor,
		); err != nil {
		return nil, err
	}

	if err :=
		ensureCWReviewItemActionable(
			item,
		); err != nil {
		return nil, err
	}

	if _, err :=
		ensureCWReviewItemFresh(
			ctx,
			item,
			actor.UserID,
		); err != nil {
		return nil, err
	}

	if err :=
		repository.DismissCoursewareReviewItem(
			ctx,
			item.ID,
			actor.UserID,
			reason,
		); err != nil {
		return nil, err
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

// RestoreCWReviewItem 恢复一个未交付的已忽略整改项。
//
// 普通问题仍按既有规则恢复为confirmed/detected。
// 带applied事实的自审问题恢复为applied，继续等待人工检查或再次调整。
func (s *CoursewareAIReviewService) RestoreCWReviewItem(
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

	if err :=
		ensureCWReviewItemStateManageable(
			item,
			actor,
		); err != nil {
		return nil, err
	}

	if item.Status !=
		models.CWReviewItemStatusDismissed {
		return nil,
			ErrCWReviewItemNotActionable
	}

	appliedVersionID := ""
	if item.AppliedInstructionVersionID != nil {
		appliedVersionID =
			strings.TrimSpace(
				*item.AppliedInstructionVersionID,
			)
	}

	dismissedAfterApplied :=
		item.SourceType ==
			models.CWReviewItemSourceSelf &&
			item.AppliedAt != nil &&
			strings.TrimSpace(
				item.AppliedPageHash,
			) != "" &&
			appliedVersionID != ""

	if dismissedAfterApplied {
		if err :=
			repository.RestoreDismissedAppliedSelfCoursewareReviewItem(
				ctx,
				item.ID,
				actor.UserID,
			); err != nil {
			return nil,
				mapCWReviewSelfPostApplyStateError(
					err,
				)
		}
	} else {
		// 普通dismissed仍按问题最初稳定页面检查，
		// 并按既有规则恢复为confirmed或detected。
		if _, err :=
			ensureCWReviewItemFresh(
				ctx,
				item,
				actor.UserID,
			); err != nil {
			return nil, err
		}

		if err :=
			repository.RestoreCoursewareReviewItem(
				ctx,
				item.ID,
				actor.UserID,
			); err != nil {
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

func normalizeCWReviewItemDismissReason(
	reason string,
) (string, error) {
	normalized :=
		strings.TrimSpace(
			reason,
		)

	if normalized == "" ||
		utf8.RuneCountInString(
			normalized,
		) >
			cwReviewItemMaxDismissReasonRunes {
		return "",
			ErrCWReviewItemDismissReasonInvalid
	}

	return normalized, nil
}

func mapCWReviewSelfPostApplyStateError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		repository.ErrCoursewareReviewItemAppliedPageChanged,
	):
		return ErrCWReviewItemStale

	case errors.Is(
		err,
		repository.ErrCoursewareReviewItemAppliedPageMissing,
	):
		return ErrCWReviewItemOrphaned

	default:
		return err
	}
}

// ensureCWReviewItemStateManageable 校验忽略和恢复操作的不可变边界。
func ensureCWReviewItemStateManageable(
	item *models.CoursewareReviewItem,
	actor *CoursewareActorContext,
) error {
	if item == nil {
		return repository.
			ErrCoursewareReviewItemNotFound
	}

	if actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" {
		return ErrCWAIReviewActorRequired
	}

	// 正式提交后，整改项已经成为人工审核反馈的一部分，
	// 任何参与者都不能再通过忽略和恢复接口改写状态。
	if item.CoursewareReviewID != nil ||
		item.FeedbackID != nil {
		return ErrCWReviewItemNotActionable
	}

	switch item.SourceType {
	case models.CWReviewItemSourceFormal:
		// 未交付正式项只允许创建它的审核员排除或恢复。
		if actor.UserID !=
			item.CreatedBy {
			return ErrCWReviewItemNotActionable
		}

	case models.CWReviewItemSourceSelf:
		// 自审项只允许课件作者本人管理。
		if actor.UserID !=
			item.OwnerID {
			return ErrCWAIReviewNoPermission
		}

	default:
		return ErrCWReviewItemNotActionable
	}

	return nil
}
