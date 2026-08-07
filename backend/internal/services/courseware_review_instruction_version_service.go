package services

// courseware_review_instruction_version_service.go
//
// 课件审核整改指令版本的读取与“保存为新版并确认”业务服务。
//
// 安全边界：
//   1. 读取复用整改项参与者、课件归属和教育域授权；
//   2. 正式项只能由创建它的审核员在交付前确认，自审项只能由作者确认；
//   3. 浏览器只提交正文和预期当前版本ID，其他可信字段全部由后端生成；
//   4. 候选来源只从已落库assistant消息元数据推导，手工改写归类为manual；
//   5. 服务层先检查页面，仓储事务再锁行复核，防止检查与写入之间发生变化；
//   6. 成功后继续返回原讨论视图，兼容现有问题面板刷新链。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// CWReviewInstructionVersionListResult 是版本历史及当前引用。
type CWReviewInstructionVersionListResult struct {
	Versions         []*models.CoursewareReviewInstructionVersion
	CurrentVersionID string
}

// ListCWReviewItemInstructionVersions 读取一条整改项的全部指令版本。
func (s *CoursewareAIReviewRunner) ListCWReviewItemInstructionVersions(
	ctx context.Context,
	itemID string,
	actor *CoursewareActorContext,
) (*CWReviewInstructionVersionListResult, error) {
	item, _, err := loadAuthorizedCWReviewItem(
		ctx,
		itemID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	versions, currentVersionID, err :=
		repository.ListCoursewareReviewInstructionVersions(
			ctx,
			item.ID,
			actor.UserID,
		)
	if err != nil {
		return nil,
			mapCWReviewInstructionVersionError(err)
	}

	return &CWReviewInstructionVersionListResult{
		Versions:         versions,
		CurrentVersionID: currentVersionID,
	}, nil
}

// GetCurrentCWReviewItemInstructionVersion 读取当前确认或已失效的历史版本。
func (s *CoursewareAIReviewRunner) GetCurrentCWReviewItemInstructionVersion(
	ctx context.Context,
	itemID string,
	actor *CoursewareActorContext,
) (*models.CoursewareReviewInstructionVersion, error) {
	item, _, err := loadAuthorizedCWReviewItem(
		ctx,
		itemID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	version, err :=
		repository.GetCurrentCoursewareReviewInstructionVersion(
			ctx,
			item.ID,
			actor.UserID,
		)
	if err != nil {
		return nil,
			mapCWReviewInstructionVersionError(err)
	}

	return version, nil
}

// GetCWReviewItemInstructionVersion 读取一条指定历史版本。
func (s *CoursewareAIReviewRunner) GetCWReviewItemInstructionVersion(
	ctx context.Context,
	itemID string,
	versionID string,
	actor *CoursewareActorContext,
) (*models.CoursewareReviewInstructionVersion, error) {
	item, _, err := loadAuthorizedCWReviewItem(
		ctx,
		itemID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	version, err :=
		repository.GetCoursewareReviewInstructionVersion(
			ctx,
			item.ID,
			strings.TrimSpace(versionID),
			actor.UserID,
		)
	if err != nil {
		return nil,
			mapCWReviewInstructionVersionError(err)
	}

	return version, nil
}

// ConfirmCWReviewItemInstructionVersion 创建并确认一个连续的新版本。
//
// expectedCurrentVersionID必须来自最近一次读取；首次确认传空字符串。
// 两个窗口使用同一旧值并发确认时，整改项行锁保证只有一个请求成功。
func (s *CoursewareAIReviewRunner) ConfirmCWReviewItemInstructionVersion(
	ctx context.Context,
	itemID string,
	instruction string,
	expectedCurrentVersionID string,
	actor *CoursewareActorContext,
) (*CWReviewItemDiscussionResult, error) {
	instruction = strings.TrimSpace(instruction)
	if instruction == "" ||
		utf8.RuneCountInString(instruction) >
			cwReviewItemMaxInstructionRunes {
		return nil,
			ErrCWReviewItemInstructionInvalid
	}

	item, _, err := loadAuthorizedCWReviewItem(
		ctx,
		itemID,
		actor,
	)
	if err != nil {
		return nil, err
	}

	if err :=
		ensureCWReviewItemInstructionConfirmationAccess(
			item,
			actor,
		); err != nil {
		return nil, err
	}

	if err := ensureCWReviewItemActionable(item); err != nil {
		return nil, err
	}

	if _, err := ensureCWReviewItemFresh(
		ctx,
		item,
		actor.UserID,
	); err != nil {
		return nil, err
	}

	messages, err :=
		repository.ListCoursewareReviewItemMessages(
			ctx,
			item.ID,
		)
	if err != nil {
		return nil, err
	}

	sourceType :=
		resolveCWReviewInstructionVersionSource(
			messages,
			instruction,
		)

	_, err =
		repository.ConfirmCoursewareReviewInstructionVersion(
			ctx,
			&repository.ConfirmCoursewareReviewInstructionVersionInput{
				ItemID:      item.ID,
				ActorID:     actor.UserID,
				Instruction: instruction,
				ExpectedCurrentVersionID: strings.TrimSpace(
					expectedCurrentVersionID,
				),
				SourceType: sourceType,
			},
		)
	if err != nil {
		return nil,
			mapCWReviewInstructionVersionError(err)
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

func ensureCWReviewItemInstructionConfirmationAccess(
	item *models.CoursewareReviewItem,
	actor *CoursewareActorContext,
) error {
	if item == nil {
		return repository.ErrCoursewareReviewItemNotFound
	}
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return ErrCWAIReviewActorRequired
	}
	if item.CoursewareReviewID != nil ||
		item.FeedbackID != nil ||
		item.AppliedAt != nil {
		return ErrCWReviewItemNotActionable
	}

	switch item.SourceType {
	case models.CWReviewItemSourceFormal:
		if actor.UserID != item.CreatedBy {
			return ErrCWAIReviewNoPermission
		}

	case models.CWReviewItemSourceSelf:
		if actor.UserID != item.OwnerID {
			return ErrCWAIReviewNoPermission
		}

	default:
		return ErrCWReviewItemNotActionable
	}

	return nil
}

// resolveCWReviewInstructionVersionSource 只信后端已经持久化的assistant候选。
//
// 用户只要对候选正文做了任何实质改写，就不再冒充AI原文，按manual记录。
func resolveCWReviewInstructionVersionSource(
	messages []*models.CoursewareReviewItemMessage,
	instruction string,
) string {
	normalizedInstruction :=
		strings.TrimSpace(instruction)

	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message == nil ||
			message.Role != "assistant" {
			continue
		}

		var meta cwReviewItemMessageMeta
		if err := json.Unmarshal(
			[]byte(message.CitationsJSON),
			&meta,
		); err != nil {
			continue
		}

		if strings.TrimSpace(
			meta.SuggestedInstruction,
		) != normalizedInstruction {
			continue
		}

		for _, citation := range meta.Citations {
			citationType, _ :=
				citation["type"].(string)

			if strings.TrimSpace(citationType) ==
				"global_discussion" {
				return models.
					CWReviewInstructionVersionSourceGlobalDiscussion
			}
		}

		return models.
			CWReviewInstructionVersionSourceAICandidate
	}

	return models.
		CWReviewInstructionVersionSourceManual
}

func mapCWReviewInstructionVersionError(
	err error,
) error {
	switch {
	case errors.Is(
		err,
		repository.ErrCoursewareReviewInstructionVersionNotFound,
	):
		return repository.ErrCoursewareReviewItemNotFound

	case errors.Is(
		err,
		repository.ErrCoursewareReviewInstructionVersionConflict,
	):
		return repository.ErrCoursewareReviewItemConflict

	case errors.Is(
		err,
		repository.ErrCoursewareReviewInstructionVersionNotConfirmable,
	):
		return ErrCWReviewItemNotActionable

	case errors.Is(
		err,
		repository.ErrCoursewareReviewInstructionVersionPageStale,
	):
		return ErrCWReviewItemStale

	case errors.Is(
		err,
		repository.ErrCoursewareReviewInstructionVersionPageOrphaned,
	):
		return ErrCWReviewItemOrphaned

	default:
		return err
	}
}
