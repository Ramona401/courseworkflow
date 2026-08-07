package services

// courseware_ai_review_global_discussion_governance.go
//
// 全局讨论结论落地与问题列表治理V1.1公开业务服务。
//
// 本文件只处理明确的人工作用：
//   1. 从可信全局讨论助手消息人工新增整改项；
//   2. 人工确认或重新启用结构化整改项关系；
//   3. 人工确认AI的consider_dismiss建议并复用原有忽略服务；
//   4. 人工取消已经确认的关系；
//   5. 读取关系及其追加式操作历史。
//
// 人工问题构建、可信关系匹配和关系历史组装位于：
// courseware_ai_review_global_discussion_governance_helpers.go。
//
// 不允许的行为：
//   - AI自动创建整改项；
//   - 自动确认候选修改指令；
//   - 自动忽略问题；
//   - 自动修改页面；
//   - 自动提交人工审核决定；
//   - 修改已经随正式反馈交付的历史整改项；
//   - 使用旧版无方向关系元数据确认有方向关系；
//   - 由浏览器反转或重新指定AI关系方向。

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	cwAIReviewGlobalManualTitleMaxRunes       = 300
	cwAIReviewGlobalManualDescriptionMaxRunes = 6000
	cwAIReviewGlobalManualDimensionMaxRunes   = 64
	cwAIReviewGlobalManualMaxPages            = 100
	cwAIReviewGlobalRelationReasonMaxRunes    = 500
)

var (
	ErrCWAIReviewGlobalManualItemInvalid = errors.New(
		"全局讨论人工新增整改项参数无效",
	)

	ErrCWAIReviewGlobalRelationInvalid = errors.New(
		"全局讨论整改项关系参数无效",
	)

	ErrCWAIReviewGlobalRelationNotSuggested = errors.New(
		"指定关系不在可信全局讨论结果中",
	)

	ErrCWAIReviewGlobalDismissNotSuggested = errors.New(
		"指定整改项没有可信的忽略建议",
	)

	ErrCWAIReviewGlobalRelationReasonInvalid = errors.New(
		"取消整改项关系必须填写不超过500字的原因",
	)
)

// CWAIReviewGlobalManualItemInput 是人工新增一个问题的明确输入。
//
// PageIDs为空表示整课问题；多个页面会拆成多条共享同一人工来源ID的页级整改项。
// CandidateInstruction只是候选，创建后仍需走单条整改项独立确认。
type CWAIReviewGlobalManualItemInput struct {
	Title                string
	Description          string
	CandidateInstruction string
	Severity             string
	Dimension            string
	PageIDs              []string
}

// CWAIReviewGlobalRelationConfirmInput 是人工确认关系的方向选择。
//
// 浏览器只提交AI已经给出的关系类型和双端ID。
// 后端必须要求其与可信全局消息中的source和target完全一致。
// Explanation始终从可信全局assistant消息的结构化元数据中读取。
type CWAIReviewGlobalRelationConfirmInput struct {
	RelationType string
	SourceItemID string
	TargetItemID string
}

// CWAIReviewGlobalRelationRecord 是关系和不可变事件历史的组合。
type CWAIReviewGlobalRelationRecord struct {
	Relation *models.CoursewareReviewItemRelation
	Events   []*models.CoursewareReviewItemRelationEvent
}

// CreateCWAIReviewGlobalManualItems 从可信全局讨论消息人工新增整改项。
func (s *CoursewareAIReviewRunner) CreateCWAIReviewGlobalManualItems(
	ctx context.Context,
	sessionID string,
	messageID string,
	input *CWAIReviewGlobalManualItemInput,
	actor *CoursewareActorContext,
) ([]*models.CoursewareReviewItem, error) {
	if input == nil {
		return nil,
			ErrCWAIReviewGlobalManualItemInvalid
	}

	session, courseware, pageDigests, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			sessionID,
			actor,
			true,
		)
	if err != nil {
		return nil, err
	}

	globalMessage, _, err :=
		loadTrustedCWAIReviewGlobalMessage(
			ctx,
			session,
			messageID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(input.Title)
	description :=
		strings.TrimSpace(input.Description)
	instruction :=
		strings.TrimSpace(
			input.CandidateInstruction,
		)
	severity := strings.TrimSpace(input.Severity)
	dimension :=
		strings.TrimSpace(input.Dimension)

	if severity == "" {
		severity =
			models.CWReviewSeverityMedium
	}
	if dimension == "" {
		dimension = "manual_review"
	}

	if title == "" ||
		utf8.RuneCountInString(title) >
			cwAIReviewGlobalManualTitleMaxRunes ||
		description == "" ||
		utf8.RuneCountInString(description) >
			cwAIReviewGlobalManualDescriptionMaxRunes ||
		instruction == "" ||
		utf8.RuneCountInString(instruction) >
			cwReviewItemMaxInstructionRunes ||
		!models.IsCWReviewSeverity(severity) ||
		utf8.RuneCountInString(dimension) >
			cwAIReviewGlobalManualDimensionMaxRunes {
		return nil,
			ErrCWAIReviewGlobalManualItemInvalid
	}

	pageIDs, err :=
		normalizeCWAIReviewGlobalManualPageIDs(
			input.PageIDs,
		)
	if err != nil {
		return nil, err
	}

	sourceFindingID, err :=
		newCWAIReviewGlobalManualFindingID()
	if err != nil {
		return nil, err
	}

	sourceType :=
		models.CWReviewItemSourceFormal
	if session.ReviewLevel ==
		models.CWAIReviewLevelSelf {
		sourceType =
			models.CWReviewItemSourceSelf
	}

	sourceMessageID := globalMessage.ID
	items := make(
		[]*models.CoursewareReviewItem,
		0,
	)

	if len(pageIDs) == 0 {
		item, buildErr :=
			buildCWAIReviewGlobalManualItem(
				session,
				courseware,
				sourceFindingID,
				sourceType,
				&sourceMessageID,
				title,
				description,
				instruction,
				severity,
				dimension,
				nil,
			)
		if buildErr != nil {
			return nil, buildErr
		}

		items = append(items, item)
	} else {
		digestMap := make(
			map[string]models.CWAIReviewPageDigest,
		)
		for _, digest := range pageDigests {
			pageID :=
				strings.TrimSpace(digest.PageID)
			if pageID == "" {
				continue
			}
			digestMap[pageID] = digest
		}

		for _, pageID := range pageIDs {
			digest, exists :=
				digestMap[pageID]
			if !exists {
				return nil,
					ErrCWAIReviewGlobalManualItemInvalid
			}

			pageSnapshot, pageErr :=
				repository.
					GetCoursewareReviewPageSnapshotByID(
						ctx,
						pageID,
						courseware.ID,
					)
			if pageErr != nil {
				return nil, pageErr
			}

			// 授权阶段和人工提交阶段之间页面若发生任何变化，
			// 不使用旧摘要创建整改项，要求重新打开全局讨论。
			if pageSnapshot.PageNumber !=
				digest.PageNumber ||
				strings.TrimSpace(
					pageSnapshot.Title,
				) != strings.TrimSpace(
					digest.Title,
				) ||
				cwAIReviewHash(
					pageSnapshot.HTMLContent,
				) != strings.TrimSpace(
					digest.HTMLHash,
				) {
				return nil,
					ErrCWAIReviewSnapshotExpired
			}

			item, buildErr :=
				buildCWAIReviewGlobalManualItem(
					session,
					courseware,
					sourceFindingID,
					sourceType,
					&sourceMessageID,
					title,
					description,
					instruction,
					severity,
					dimension,
					pageSnapshot,
				)
			if buildErr != nil {
				return nil, buildErr
			}

			items = append(items, item)
		}
	}

	if err := repository.
		CreateManualCoursewareReviewItemsFromGlobalDiscussion(
			ctx,
			items,
			actor.UserID,
		); err != nil {
		return nil, err
	}

	return items, nil
}

// ConfirmCWAIReviewGlobalRelation 明确确认或重新启用一条AI建议关系。
func (s *CoursewareAIReviewRunner) ConfirmCWAIReviewGlobalRelation(
	ctx context.Context,
	sessionID string,
	messageID string,
	input *CWAIReviewGlobalRelationConfirmInput,
	actor *CoursewareActorContext,
) (*CWAIReviewGlobalRelationRecord, error) {
	if input == nil {
		return nil,
			ErrCWAIReviewGlobalRelationInvalid
	}

	session, _, _, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			sessionID,
			actor,
			true,
		)
	if err != nil {
		return nil, err
	}

	globalMessage, meta, err :=
		loadTrustedCWAIReviewGlobalMessage(
			ctx,
			session,
			messageID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	// 旧版全局消息只有item_ids集合，没有可信方向。
	// 为避免浏览器反转关系，旧消息只能展示，不能确认关系。
	if meta.RelationSchemaVersion !=
		cwAIReviewGlobalRelationSchemaVersion {
		return nil,
			ErrCWAIReviewGlobalRelationNotSuggested
	}

	relationType :=
		strings.TrimSpace(input.RelationType)
	sourceItemID :=
		strings.TrimSpace(input.SourceItemID)
	targetItemID :=
		strings.TrimSpace(input.TargetItemID)

	if !models.IsCWReviewItemRelationType(
		relationType,
	) ||
		sourceItemID == "" ||
		targetItemID == "" ||
		sourceItemID == targetItemID {
		return nil,
			ErrCWAIReviewGlobalRelationInvalid
	}

	// conflict无方向，输入先规范为数据库要求的UUID文本升序。
	if relationType ==
		models.CWReviewItemRelationConflict &&
		sourceItemID > targetItemID {
		sourceItemID, targetItemID =
			targetItemID, sourceItemID
	}

	suggestedRelation :=
		findCWAIReviewGlobalSuggestedRelation(
			meta.Relations,
			relationType,
			sourceItemID,
			targetItemID,
		)
	if suggestedRelation == nil {
		return nil,
			ErrCWAIReviewGlobalRelationNotSuggested
	}

	// 再次加载两端整改项，复核归属、未交付、可处理和页面快照有效性。
	if _, err :=
		loadCWAIReviewGlobalSelectedItems(
			ctx,
			session,
			[]string{
				sourceItemID,
				targetItemID,
			},
			actor,
		); err != nil {
		return nil, err
	}

	sourceMessageID := globalMessage.ID
	relation, err :=
		repository.
			ConfirmCoursewareReviewItemRelation(
				ctx,
				&models.
					CoursewareReviewItemRelation{
					CoursewareID:    session.CoursewareID,
					SourceSessionID: session.ID,
					SourceItemID:    sourceItemID,
					TargetItemID:    targetItemID,
					RelationType:    relationType,
					Explanation: strings.TrimSpace(
						suggestedRelation.
							Explanation,
					),
					SourceGlobalMessageID: &sourceMessageID,
					CreatedBy:             actor.UserID,
				},
				actor.UserID,
			)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewGlobalRelationRecord(
		ctx,
		relation,
		actor.UserID,
	)
}

// ConfirmCWAIReviewGlobalDismissal 明确采用可信AI的consider_dismiss建议。
//
// 本方法只负责验证建议来源，最终状态迁移继续复用原有整改项忽略服务。
// 因此未交付限制、操作者权限、页面新鲜度和系统消息审计不会被绕过。
func (s *CoursewareAIReviewRunner) ConfirmCWAIReviewGlobalDismissal(
	ctx context.Context,
	sessionID string,
	messageID string,
	itemID string,
	reason string,
	actor *CoursewareActorContext,
) (*CWReviewItemDiscussionResult, error) {
	session, _, _, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			sessionID,
			actor,
			true,
		)
	if err != nil {
		return nil, err
	}

	_, meta, err :=
		loadTrustedCWAIReviewGlobalMessage(
			ctx,
			session,
			messageID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	itemID = strings.TrimSpace(itemID)
	proposal :=
		findCWAIReviewGlobalProposal(
			meta.Proposals,
			itemID,
		)

	if proposal == nil ||
		strings.TrimSpace(
			proposal.Recommendation,
		) != "consider_dismiss" ||
		!containsCWAIReviewGlobalItemID(
			meta.SelectedItemIDs,
			itemID,
		) {
		return nil,
			ErrCWAIReviewGlobalDismissNotSuggested
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
	if item.SourceSessionID != session.ID {
		return nil,
			ErrCWAIReviewGlobalDismissNotSuggested
	}

	return dismissCWReviewItem(
		ctx,
		item,
		reason,
		actor,
	)
}

// CancelCWAIReviewGlobalRelation 明确取消一条关系，不改变任何整改项状态。
func (s *CoursewareAIReviewRunner) CancelCWAIReviewGlobalRelation(
	ctx context.Context,
	sessionID string,
	relationID string,
	reason string,
	actor *CoursewareActorContext,
) (*CWAIReviewGlobalRelationRecord, error) {
	reason = strings.TrimSpace(reason)

	if reason == "" ||
		utf8.RuneCountInString(reason) >
			cwAIReviewGlobalRelationReasonMaxRunes {
		return nil,
			ErrCWAIReviewGlobalRelationReasonInvalid
	}

	session, _, _, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			sessionID,
			actor,
			false,
		)
	if err != nil {
		return nil, err
	}

	relation, err :=
		repository.
			CancelCoursewareReviewItemRelation(
				ctx,
				session.ID,
				relationID,
				actor.UserID,
				reason,
			)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewGlobalRelationRecord(
		ctx,
		relation,
		actor.UserID,
	)
}

// ListCWAIReviewGlobalRelations 读取当前会话已经人工确认的全部关系及历史。
func (s *CoursewareAIReviewRunner) ListCWAIReviewGlobalRelations(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
) ([]*CWAIReviewGlobalRelationRecord, error) {
	session, _, _, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			sessionID,
			actor,
			false,
		)
	if err != nil {
		return nil, err
	}

	relations, err :=
		repository.
			ListCoursewareReviewItemRelationsBySession(
				ctx,
				session.ID,
				actor.UserID,
			)
	if err != nil {
		return nil, err
	}

	result := make(
		[]*CWAIReviewGlobalRelationRecord,
		0,
		len(relations),
	)
	for _, relation := range relations {
		record, buildErr :=
			buildCWAIReviewGlobalRelationRecord(
				ctx,
				relation,
				actor.UserID,
			)
		if buildErr != nil {
			return nil, buildErr
		}

		result = append(result, record)
	}

	return result, nil
}
