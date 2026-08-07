package services

// courseware_ai_review_item_state.go
//
// 课件审核整改项的忽略、恢复和作者自审解决确认业务服务。
//
// 设计边界：
//
//   1. 复用现有dismissed状态，不建设第二套排除模型；
//   2. 忽略只适用于未交付的自审项或正式审核草稿项；
//   3. 正式整改项只能由创建该项的审核员在提交决定前操作；
//   4. 已绑定正式反馈的整改项属于不可变审核历史，禁止忽略和恢复；
//   5. 忽略原因通过整改项系统消息保存，不污染AI原始证据；
//   6. 忽略和恢复均不删除讨论记录、确认指令或整改项实体；
//   7. 恢复前重新检查稳定页面及HTML哈希；
//   8. 作者自审问题只有到达applied后，才允许作者本人确认解决；
//   9. 作者确认解决前，仓储会原子重新检查当前页面内容指纹；
//  10. 页面变化或删除时，问题改为stale或orphaned；
//  11. 正式审核问题不能通过作者自审接口关闭；
//  12. 全局讨论“确认忽略”复用同一内部入口。

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

// DismissCWReviewItem 将一个仍可处理且尚未交付的整改项标记为无需修改。
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
	reason =
		strings.TrimSpace(
			reason,
		)

	if reason == "" ||
		utf8.RuneCountInString(
			reason,
		) >
			cwReviewItemMaxDismissReasonRunes {
		return nil,
			ErrCWReviewItemDismissReasonInvalid
	}

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
// 有确认指令时仓储恢复为confirmed，没有确认指令时恢复为detected。
// 恢复不会自动勾选正式退回清单；前端控制器根据返回状态执行既有选择规则。
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

	// dismissed状态不会被ensureCWReviewItemFresh自动迁移，
	// 但仍会返回页面变化或删除错误，从而阻止恢复失效问题。
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
