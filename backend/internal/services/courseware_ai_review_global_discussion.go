package services

// courseware_ai_review_global_discussion.go
//
// 已完成课件AI审核会话的跨页面、跨问题全局讨论核心业务服务。
//
// 本文件负责：
//   1. 全局讨论历史读取和消息编排；
//   2. 将用户问题与AI结果写入会话级消息；
//   3. 从可信全局助手消息采用逐项候选指令；
//   4. 将候选指令追加到原单条整改项讨论历史；
//   5. 定义全局讨论消息及关系协议。
//
// 会话授权、快照校验、整改项选择校验和结果恢复位于：
// courseware_ai_review_global_discussion_access.go。
//
// AI上下文构造、提示词、模型调用和结构化结果解析位于：
// courseware_ai_review_global_discussion_ai.go。
//
// 安全与业务边界：
//   1. 只允许会话创建者操作自己的已完成会话；
//   2. 每轮必须明确选择2至12条同会话整改项；
//   3. 选中项必须未交付、仍可处理且页面快照有效；
//   4. AI不得自动确认指令、忽略问题、修改页面或提交审核决定；
//   5. 采用动作只追加候选消息，不改变整改项状态或确认指令；
//   6. 候选正文从后端可信消息元数据读取，不接受浏览器伪造；
//   7. 最终指令仍须通过原有单条整改项confirm接口独立确认；
//   8. 有方向关系必须同时保存可信source_item_id和target_item_id；
//   9. 旧版无方向元数据不得用于确认有方向关系。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	cwAIReviewGlobalMinItems        = 2
	cwAIReviewGlobalMaxItems        = 12
	cwAIReviewGlobalMaxContentRunes = 8000
	cwAIReviewGlobalMaxMessages     = 40
	cwAIReviewGlobalMetadataKind    = "courseware_review_global_discussion"

	// 第二版关系协议明确保存source和target。
	// 旧版消息没有该版本号，只能展示，不能确认有方向关系。
	cwAIReviewGlobalRelationSchemaVersion = 2
)

var (
	ErrCWAIReviewGlobalContentInvalid = errors.New(
		"全局讨论内容不能为空或内容过长",
	)
	ErrCWAIReviewGlobalSelectionInvalid = errors.New(
		"全局讨论必须选择2至12条同一会话中的有效整改项",
	)
	ErrCWAIReviewGlobalNotActionable = errors.New(
		"当前课件AI审核会话不能继续全局讨论",
	)
	ErrCWAIReviewGlobalMessageLimit = errors.New(
		"全局讨论轮次过多，请重新开始课件AI审核",
	)
	ErrCWAIReviewGlobalProposalNotFound = errors.New(
		"全局讨论候选指令不存在或已经失效",
	)
)

// CWAIReviewGlobalRelation 描述两条整改项之间的一条明确关系。
//
// 方向语义：
//   - duplicate：source重复target，target为保留主问题；
//   - merge：source合并进入target；
//   - dependency：source依赖target先完成；
//   - possibly_resolved：source可能被target的修改连带解决；
//   - conflict：无方向，source和target按UUID文本升序保存。
//
// ItemIDs固定为[source_item_id, target_item_id]，用于兼容原有前端展示。
// 真正关系确认必须校验SourceItemID和TargetItemID，不能只检查集合成员。
type CWAIReviewGlobalRelation struct {
	Type string `json:"type"`

	SourceItemID string `json:"source_item_id"`
	TargetItemID string `json:"target_item_id"`

	ItemIDs []string `json:"item_ids"`

	Explanation string `json:"explanation"`
}

// CWAIReviewGlobalProposal 是一条整改项的全局分析建议。
//
// SuggestedInstruction只是候选，采用后仍需在单条整改项中独立确认。
type CWAIReviewGlobalProposal struct {
	ItemID               string `json:"item_id"`
	Recommendation       string `json:"recommendation"`
	Reason               string `json:"reason"`
	SuggestedInstruction string `json:"suggested_instruction"`
}

// CWAIReviewGlobalDiscussionResult 是浏览器可见的全局讨论结果。
type CWAIReviewGlobalDiscussionResult struct {
	Messages []*models.CoursewareAIReviewMessage

	Summary         string
	Relations       []CWAIReviewGlobalRelation
	Proposals       []CWAIReviewGlobalProposal
	SelectedItemIDs []string
	LatestMessageID string
}

// cwAIReviewGlobalDiscussionMeta 保存一轮可信AI综合结果。
//
// 后续采用候选指令、确认关系或确认忽略时，
// 必须按消息ID重新读取并解析本结构。
type cwAIReviewGlobalDiscussionMeta struct {
	Kind string `json:"kind"`

	RelationSchemaVersion int `json:"relation_schema_version"`

	Summary         string                     `json:"summary"`
	SelectedItemIDs []string                   `json:"selected_item_ids"`
	Relations       []CWAIReviewGlobalRelation `json:"relations"`
	Proposals       []CWAIReviewGlobalProposal `json:"proposals"`
}

// cwAIReviewGlobalSelectionMeta 保存用户本轮明确选择的整改项。
type cwAIReviewGlobalSelectionMeta struct {
	Kind            string   `json:"kind"`
	Event           string   `json:"event"`
	SelectedItemIDs []string `json:"selected_item_ids"`
}

// GetCWAIReviewGlobalDiscussion 读取会话级全局讨论历史。
func (s *CoursewareAIReviewRunner) GetCWAIReviewGlobalDiscussion(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
) (*CWAIReviewGlobalDiscussionResult, error) {
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

	messages, err :=
		repository.ListCoursewareAIReviewSessionMessages(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewGlobalDiscussionResult(
		messages,
	), nil
}

// MessageCWAIReviewGlobalDiscussion 提交一次跨问题综合讨论。
func (s *CoursewareAIReviewRunner) MessageCWAIReviewGlobalDiscussion(
	ctx context.Context,
	sessionID string,
	content string,
	itemIDs []string,
	actor *CoursewareActorContext,
) (*CWAIReviewGlobalDiscussionResult, error) {
	content = strings.TrimSpace(content)
	if content == "" ||
		utf8.RuneCountInString(content) >
			cwAIReviewGlobalMaxContentRunes {
		return nil,
			ErrCWAIReviewGlobalContentInvalid
	}

	normalizedItemIDs, err :=
		normalizeCWAIReviewGlobalItemIDs(
			itemIDs,
		)
	if err != nil {
		return nil, err
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

	items, err :=
		loadCWAIReviewGlobalSelectedItems(
			ctx,
			session,
			normalizedItemIDs,
			actor,
		)
	if err != nil {
		return nil, err
	}

	messages, err :=
		repository.ListCoursewareAIReviewSessionMessages(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	// 一轮新增用户和助手两条消息，因此最多允许已有38条。
	if len(messages) >
		cwAIReviewGlobalMaxMessages-2 {
		return nil,
			ErrCWAIReviewGlobalMessageLimit
	}

	selectionMetaJSON, err := json.Marshal(
		cwAIReviewGlobalSelectionMeta{
			Kind:            cwAIReviewGlobalMetadataKind,
			Event:           "selection",
			SelectedItemIDs: normalizedItemIDs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化全局讨论选择信息失败: %w",
			err,
		)
	}

	userID := actor.UserID
	userMessage := &models.CoursewareAIReviewMessage{
		SessionID:     session.ID,
		UserID:        &userID,
		Role:          "user",
		Content:       content,
		CitationsJSON: string(selectionMetaJSON),
	}
	if err :=
		repository.AppendCoursewareAIReviewSessionMessage(
			ctx,
			userMessage,
			actor.UserID,
		); err != nil {
		return nil, err
	}

	messages, err =
		repository.ListCoursewareAIReviewSessionMessages(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	aiResponse, callResult, err :=
		s.generateCWAIReviewGlobalDiscussion(
			ctx,
			session,
			courseware,
			pageDigests,
			items,
			normalizedItemIDs,
			messages,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	metaJSON, err := json.Marshal(
		cwAIReviewGlobalDiscussionMeta{
			Kind:                  cwAIReviewGlobalMetadataKind,
			RelationSchemaVersion: cwAIReviewGlobalRelationSchemaVersion,
			Summary:               aiResponse.Summary,
			SelectedItemIDs:       normalizedItemIDs,
			Relations:             aiResponse.Relations,
			Proposals:             aiResponse.Proposals,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化全局讨论AI结果失败: %w",
			err,
		)
	}

	assistantMessage :=
		&models.CoursewareAIReviewMessage{
			SessionID:     session.ID,
			Role:          "assistant",
			Content:       aiResponse.Reply,
			CitationsJSON: string(metaJSON),
			TokensUsed:    callResult.TokensUsed,
			ModelUsed: strings.TrimSpace(
				callResult.ModelUsed,
			),
		}
	if err :=
		repository.AppendCoursewareAIReviewSessionMessage(
			ctx,
			assistantMessage,
			actor.UserID,
		); err != nil {
		return nil, err
	}

	messages, err =
		repository.ListCoursewareAIReviewSessionMessages(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewGlobalDiscussionResult(
		messages,
	), nil
}

// AdoptCWAIReviewGlobalProposal 明确采用一条全局讨论候选指令。
//
// 浏览器只提交全局助手消息ID和整改项ID。
// 指令正文、理由和引用全部从后端可信消息元数据重新读取。
func (s *CoursewareAIReviewRunner) AdoptCWAIReviewGlobalProposal(
	ctx context.Context,
	sessionID string,
	messageID string,
	itemID string,
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

	globalMessage, err :=
		repository.GetCoursewareAIReviewSessionMessageByID(
			ctx,
			session.ID,
			strings.TrimSpace(messageID),
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}
	if globalMessage == nil ||
		globalMessage.Role != "assistant" {
		return nil,
			ErrCWAIReviewGlobalProposalNotFound
	}

	var meta cwAIReviewGlobalDiscussionMeta
	if err := json.Unmarshal(
		[]byte(globalMessage.CitationsJSON),
		&meta,
	); err != nil ||
		meta.Kind !=
			cwAIReviewGlobalMetadataKind {
		return nil,
			ErrCWAIReviewGlobalProposalNotFound
	}

	itemID = strings.TrimSpace(itemID)
	var proposal *CWAIReviewGlobalProposal

	for index := range meta.Proposals {
		current := &meta.Proposals[index]
		if strings.TrimSpace(
			current.ItemID,
		) == itemID {
			proposal = current
			break
		}
	}

	if proposal == nil ||
		!containsCWAIReviewGlobalItemID(
			meta.SelectedItemIDs,
			itemID,
		) {
		return nil,
			ErrCWAIReviewGlobalProposalNotFound
	}

	instruction :=
		strings.TrimSpace(
			proposal.SuggestedInstruction,
		)
	if instruction == "" ||
		utf8.RuneCountInString(instruction) >
			cwReviewItemMaxInstructionRunes {
		return nil,
			ErrCWAIReviewGlobalProposalNotFound
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
			ErrCWAIReviewGlobalProposalNotFound
	}
	if err :=
		ensureCWReviewItemInstructionGenerationAccess(
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

	page, err :=
		ensureCWReviewItemFresh(
			ctx,
			item,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	citations :=
		buildCWReviewItemCitations(
			item,
			page,
		)
	citations = append(
		citations,
		map[string]interface{}{
			"type":                      "global_discussion",
			"global_discussion_message": globalMessage.ID,
			"recommendation":            proposal.Recommendation,
		},
	)

	summary :=
		strings.TrimSpace(
			proposal.Reason,
		)
	if summary == "" {
		summary =
			"已从跨页面、跨问题全局讨论中采用候选修改指令。"
	}

	itemMetaJSON, err := json.Marshal(
		cwReviewItemMessageMeta{
			Summary:              summary,
			ReadyForConfirmation: true,
			SuggestedInstruction: instruction,
			Citations:            citations,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"序列化全局讨论采用结果失败: %w",
			err,
		)
	}

	itemMessage :=
		&models.CoursewareReviewItemMessage{
			SessionID:    session.ID,
			ReviewItemID: item.ID,
			Role:         "assistant",
			Content: "已从全局讨论采用此候选修改指令。" +
				"该内容尚未独立确认，也不会自动修改页面或改变审核决定。",
			CitationsJSON: string(itemMetaJSON),
			ModelUsed:     globalMessage.ModelUsed,
		}

	if err :=
		repository.AppendCoursewareReviewItemCandidateFromGlobalDiscussion(
			ctx,
			itemMessage,
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

	result, err :=
		buildCWReviewItemDiscussionResult(
			ctx,
			updatedItem,
		)
	if err != nil {
		return nil, err
	}

	result.Summary = summary
	result.ReadyForConfirmation = true
	result.SuggestedInstruction = instruction

	return result, nil
}
