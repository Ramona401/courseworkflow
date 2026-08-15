package services

// courseware_review_item_execution_note.go
//
// 作者正式整改“本次执行补充”服务。
//
// 业务边界：
//   1. 只允许课件作者本人对已经正式交付的formal整改项追加；
//   2. 补充只保存为用户可见消息，不调用AI；
//   3. 不修改确认整改要求、指令版本、页面内容或整改状态；
//   4. stale/orphaned仍允许作者说明实际处理情况，但resolved后历史冻结；
//   5. 仓储写入再次带作者、交付状态和整改状态条件，抵御校验后的并发变化。

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const cwReviewItemMaxExecutionNoteRunes = 2000

var (
	ErrCWReviewItemExecutionNoteInvalid     = errors.New("本次执行补充内容无效")
	ErrCWReviewItemExecutionNoteForbidden   = errors.New("只有课件作者可以补充本次执行说明")
	ErrCWReviewItemExecutionNoteUnavailable = errors.New("当前整改项不能添加本次执行补充")
)

func normalizeCWReviewItemExecutionNote(
	content string,
) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" ||
		utf8.RuneCountInString(content) >
			cwReviewItemMaxExecutionNoteRunes {
		return "",
			ErrCWReviewItemExecutionNoteInvalid
	}

	return content, nil
}

func validateCWReviewItemExecutionNote(
	item *models.CoursewareReviewItem,
	actorID string,
) error {
	actorID = strings.TrimSpace(actorID)
	if item == nil || actorID == "" {
		return ErrCWReviewItemExecutionNoteForbidden
	}

	if actorID != strings.TrimSpace(item.OwnerID) {
		return ErrCWReviewItemExecutionNoteForbidden
	}

	if item.SourceType !=
		models.CWReviewItemSourceFormal ||
		item.CoursewareReviewID == nil ||
		item.FeedbackID == nil ||
		item.DeliveredInstructionVersionID == nil {
		return ErrCWReviewItemExecutionNoteUnavailable
	}

	switch item.Status {
	case models.CWReviewItemStatusConfirmed,
		models.CWReviewItemStatusApplying,
		models.CWReviewItemStatusApplied,
		models.CWReviewItemStatusStale,
		models.CWReviewItemStatusOrphaned:
		return nil
	default:
		return ErrCWReviewItemExecutionNoteUnavailable
	}
}

// AddCWReviewItemExecutionNote 为作者已收到的正式整改项追加“本次执行补充”。
//
// 本方法刻意不调用ensureCWReviewItemFresh：执行补充是人工过程记录，页面已经变化或删除时
// 仍应允许作者说明实际处理情况，但该文字绝不能据此改变stale/orphaned或复审结论。
func (s *CoursewareAIReviewService) AddCWReviewItemExecutionNote(
	ctx context.Context,
	itemID string,
	content string,
	actor *CoursewareActorContext,
) (*CWReviewItemDiscussionResult, error) {
	if s == nil {
		return nil,
			errors.New("课件AI审核服务未初始化")
	}

	content, err :=
		normalizeCWReviewItemExecutionNote(
			content,
		)
	if err != nil {
		return nil, err
	}

	item, _, err :=
		loadAuthorizedCWReviewItem(
			ctx,
			itemID,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if err := validateCWReviewItemExecutionNote(
		item,
		actor.UserID,
	); err != nil {
		return nil, err
	}

	userID := actor.UserID
	message := &models.CoursewareReviewItemMessage{
		SessionID:     item.SourceSessionID,
		ReviewItemID:  item.ID,
		UserID:        &userID,
		Role:          "user",
		Content:       content,
		CitationsJSON: `{"event":"owner_execution_note","label":"本次执行补充"}`,
		TokensUsed:    0,
		ModelUsed:     "",
	}

	if err := repository.AppendCoursewareReviewItemExecutionNote(
		ctx,
		message,
		actor.UserID,
	); err != nil {
		if errors.Is(
			err,
			repository.ErrCoursewareReviewItemConflict,
		) {
			return nil,
				ErrCWReviewItemExecutionNoteUnavailable
		}
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
