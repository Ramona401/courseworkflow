package services

// courseware_ai_review_global_discussion_access.go
//
// 全局讨论的会话授权、快照校验、整改项选择和结果恢复辅助模块。
//
// 设计边界：
//   1. 全局讨论不扩大管理员权限，只允许原会话创建者操作；
//   2. 自审和正式审核分别复用既有课件权限校验；
//   3. 写操作前重新比较课件与全部页面快照；
//   4. 选中项必须属于同一会话、同一课件和正确审核级别；
//   5. 已交付、终态、失效或页面已变化的整改项不能参与讨论；
//   6. 历史结果只从已保存的可信assistant消息恢复。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// authorizeCWAIReviewGlobalDiscussionSession
// 统一校验全局讨论会话所有权、课件访问、状态和可选快照。
func (s *CoursewareAIReviewRunner) authorizeCWAIReviewGlobalDiscussionSession(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
	requireCurrentSnapshot bool,
) (
	*models.CoursewareAIReviewSession,
	*models.Courseware,
	[]models.CWAIReviewPageDigest,
	error,
) {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return nil, nil, nil,
			ErrCWAIReviewActorRequired
	}
	if s == nil || s.cfg == nil {
		return nil, nil, nil,
			errors.New(
				"课件AI审核全局讨论服务未初始化",
			)
	}

	session, err :=
		repository.GetCoursewareAIReviewSessionByID(
			ctx,
			strings.TrimSpace(sessionID),
		)
	if err != nil {
		return nil, nil, nil, err
	}
	if session == nil {
		return nil, nil, nil,
			ErrCWAIReviewSessionNotFound
	}

	// 全局讨论不扩大管理员权限，只允许原会话创建者操作。
	if session.ReviewerID != actor.UserID {
		return nil, nil, nil,
			ErrCWAIReviewSessionOwnerMismatch
	}
	if session.Status !=
		models.CWAIReviewStatusDone {
		return nil, nil, nil,
			ErrCWAIReviewGlobalNotActionable
	}

	courseware, err :=
		repository.GetCoursewareByID(
			ctx,
			session.CoursewareID,
		)
	if err != nil || courseware == nil {
		return nil, nil, nil,
			ErrCWAIReviewCoursewareNotFound
	}

	if session.ReviewLevel ==
		models.CWAIReviewLevelSelf {
		if courseware.UserID != actor.UserID {
			return nil, nil, nil,
				ErrCWAIReviewNoPermission
		}
		if err :=
			ValidateCoursewareReviewEducationDomain(
				actor,
				courseware,
			); err != nil {
			return nil, nil, nil, err
		}
	} else {
		if s.reviewService == nil {
			return nil, nil, nil,
				errors.New(
					"课件AI审核权限服务未初始化",
				)
		}

		allowed, reviewErr :=
			s.reviewService.CanReviewLoadedCourseware(
				ctx,
				courseware,
				actor,
			)
		if reviewErr != nil {
			return nil, nil, nil,
				reviewErr
		}
		if !allowed {
			return nil, nil, nil,
				ErrCWAIReviewNoPermission
		}
	}

	if !requireCurrentSnapshot {
		return session, courseware, nil, nil
	}

	if err := validateCWAIReviewLevel(
		courseware,
		session.ReviewLevel,
	); err != nil {
		return nil, nil, nil,
			ErrCWAIReviewGlobalNotActionable
	}

	pageDigests, err :=
		loadCurrentCWAIReviewPageDigests(
			ctx,
			courseware.ID,
		)
	if err != nil {
		return nil, nil, nil, err
	}

	if err :=
		validateCWAIReviewGlobalSnapshot(
			session,
			courseware,
			pageDigests,
		); err != nil {
		return nil, nil, nil, err
	}

	return session,
		courseware,
		pageDigests,
		nil
}

// validateCWAIReviewGlobalSnapshot 阻止旧审核会话讨论已变化的课件。
func validateCWAIReviewGlobalSnapshot(
	session *models.CoursewareAIReviewSession,
	courseware *models.Courseware,
	pageDigests []models.CWAIReviewPageDigest,
) error {
	pageSnapshotJSON, err :=
		json.Marshal(pageDigests)
	if err != nil {
		return fmt.Errorf(
			"序列化全局讨论页面快照失败: %w",
			err,
		)
	}

	if cwAIReviewHash(
		string(pageSnapshotJSON),
	) != strings.TrimSpace(
		session.PagesSnapshotHash,
	) {
		return ErrCWAIReviewSnapshotExpired
	}

	coursewareSnapshotJSON, err :=
		json.Marshal(
			map[string]interface{}{
				"id":               courseware.ID,
				"title":            courseware.Title,
				"subject":          courseware.Subject,
				"grade":            courseware.Grade,
				"education_domain": courseware.EducationDomain,
				"source_type":      courseware.SourceType,
				"index_overview":   courseware.IndexOverview,
				"kp_codes":         courseware.KPCodes,
				"updated_at":       courseware.UpdatedAt,
			},
		)
	if err != nil {
		return fmt.Errorf(
			"序列化全局讨论课件快照失败: %w",
			err,
		)
	}

	if cwAIReviewHash(
		string(coursewareSnapshotJSON),
	) != strings.TrimSpace(
		session.CoursewareSnapshotHash,
	) {
		return ErrCWAIReviewSnapshotExpired
	}

	return nil
}

// normalizeCWAIReviewGlobalItemIDs 去空、去重并校验选择数量。
func normalizeCWAIReviewGlobalItemIDs(
	itemIDs []string,
) ([]string, error) {
	result := make(
		[]string,
		0,
		len(itemIDs),
	)
	seen := make(map[string]struct{})

	for _, raw := range itemIDs {
		itemID := strings.TrimSpace(raw)
		if itemID == "" {
			continue
		}
		if _, exists := seen[itemID]; exists {
			continue
		}

		seen[itemID] = struct{}{}
		result = append(result, itemID)
	}

	if len(result) <
		cwAIReviewGlobalMinItems ||
		len(result) >
			cwAIReviewGlobalMaxItems {
		return nil,
			ErrCWAIReviewGlobalSelectionInvalid
	}

	return result, nil
}

// loadCWAIReviewGlobalSelectedItems
// 按原始输入顺序加载并校验本轮选中的整改项。
func loadCWAIReviewGlobalSelectedItems(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
	itemIDs []string,
	actor *CoursewareActorContext,
) ([]*models.CoursewareReviewItem, error) {
	allItems, err :=
		repository.ListCoursewareReviewItemsBySessionForCreator(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	itemMap :=
		make(
			map[string]*models.CoursewareReviewItem,
		)
	for _, item := range allItems {
		if item == nil {
			continue
		}
		itemMap[item.ID] = item
	}

	expectedSource :=
		models.CWReviewItemSourceFormal
	if session.ReviewLevel ==
		models.CWAIReviewLevelSelf {
		expectedSource =
			models.CWReviewItemSourceSelf
	}

	selected := make(
		[]*models.CoursewareReviewItem,
		0,
		len(itemIDs),
	)

	for _, itemID := range itemIDs {
		item := itemMap[itemID]
		if item == nil ||
			item.CoursewareID !=
				session.CoursewareID ||
			item.SourceSessionID !=
				session.ID ||
			item.SourceType !=
				expectedSource ||
			item.ReviewLevel !=
				session.ReviewLevel {
			return nil,
				ErrCWAIReviewGlobalSelectionInvalid
		}

		if item.CoursewareReviewID != nil ||
			item.FeedbackID != nil {
			return nil,
				ErrCWAIReviewGlobalNotActionable
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

		selected = append(
			selected,
			item,
		)
	}

	return selected, nil
}

// buildCWAIReviewGlobalDiscussionResult
// 从最近一条可信助手消息恢复当前关系和候选指令。
func buildCWAIReviewGlobalDiscussionResult(
	messages []*models.CoursewareAIReviewMessage,
) *CWAIReviewGlobalDiscussionResult {
	result :=
		&CWAIReviewGlobalDiscussionResult{
			Messages:        messages,
			Relations:       []CWAIReviewGlobalRelation{},
			Proposals:       []CWAIReviewGlobalProposal{},
			SelectedItemIDs: []string{},
		}

	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message == nil ||
			message.Role != "assistant" {
			continue
		}

		var meta cwAIReviewGlobalDiscussionMeta
		if err := json.Unmarshal(
			[]byte(message.CitationsJSON),
			&meta,
		); err != nil ||
			meta.Kind !=
				cwAIReviewGlobalMetadataKind {
			continue
		}

		result.Summary =
			strings.TrimSpace(
				meta.Summary,
			)
		result.Relations =
			meta.Relations
		result.Proposals =
			meta.Proposals
		result.SelectedItemIDs =
			meta.SelectedItemIDs
		result.LatestMessageID =
			message.ID
		break
	}

	return result
}

func containsCWAIReviewGlobalItemID(
	itemIDs []string,
	target string,
) bool {
	target = strings.TrimSpace(target)

	for _, itemID := range itemIDs {
		if strings.TrimSpace(itemID) ==
			target {
			return true
		}
	}

	return false
}
