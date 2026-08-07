package services

// courseware_ai_review_global_discussion_governance_helpers.go
//
// 全局讨论问题治理的内部辅助模块。
//
// 本文件负责：
//   1. 重新读取和解析可信全局assistant消息；
//   2. 规范人工新增问题的页面ID；
//   3. 生成人工问题来源ID；
//   4. 构建整课或页级人工整改项快照；
//   5. 按可信双端和方向匹配AI关系；
//   6. 读取逐项建议；
//   7. 组装关系及追加式审计历史。
//
// 所有公开人工动作仍位于：
// courseware_ai_review_global_discussion_governance.go。

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func loadTrustedCWAIReviewGlobalMessage(
	ctx context.Context,
	session *models.CoursewareAIReviewSession,
	messageID string,
	actorID string,
) (
	*models.CoursewareAIReviewMessage,
	*cwAIReviewGlobalDiscussionMeta,
	error,
) {
	if session == nil {
		return nil,
			nil,
			ErrCWAIReviewGlobalProposalNotFound
	}

	message, err :=
		repository.
			GetCoursewareAIReviewSessionMessageByID(
				ctx,
				session.ID,
				strings.TrimSpace(messageID),
				strings.TrimSpace(actorID),
			)
	if err != nil {
		return nil, nil, err
	}
	if message == nil ||
		message.Role != "assistant" {
		return nil,
			nil,
			ErrCWAIReviewGlobalProposalNotFound
	}

	meta := &cwAIReviewGlobalDiscussionMeta{}
	if err := json.Unmarshal(
		[]byte(message.CitationsJSON),
		meta,
	); err != nil ||
		meta.Kind !=
			cwAIReviewGlobalMetadataKind {
		return nil,
			nil,
			ErrCWAIReviewGlobalProposalNotFound
	}

	return message, meta, nil
}

func normalizeCWAIReviewGlobalManualPageIDs(
	input []string,
) ([]string, error) {
	result := make(
		[]string,
		0,
		len(input),
	)
	seen := make(map[string]struct{})

	for _, raw := range input {
		pageID := strings.TrimSpace(raw)
		if pageID == "" {
			continue
		}
		if _, exists := seen[pageID]; exists {
			continue
		}

		seen[pageID] = struct{}{}
		result = append(result, pageID)
	}

	if len(result) >
		cwAIReviewGlobalManualMaxPages {
		return nil,
			ErrCWAIReviewGlobalManualItemInvalid
	}

	return result, nil
}

func newCWAIReviewGlobalManualFindingID() (
	string,
	error,
) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "",
			fmt.Errorf(
				"生成人工整改项来源ID失败: %w",
				err,
			)
	}

	return "manual_" +
			hex.EncodeToString(randomBytes),
		nil
}

func buildCWAIReviewGlobalManualItem(
	session *models.CoursewareAIReviewSession,
	courseware *models.Courseware,
	sourceFindingID string,
	sourceType string,
	sourceMessageID *string,
	title string,
	description string,
	instruction string,
	severity string,
	dimension string,
	page *repository.CoursewareReviewPageSnapshot,
) (*models.CoursewareReviewItem, error) {
	evidence := map[string]interface{}{
		"origin_type": models.
			CWReviewItemOriginGlobalDiscussionManual,
		"source_global_message_id": "",
		"scope":                    "courseware",
	}
	if sourceMessageID != nil {
		evidence["source_global_message_id"] = strings.TrimSpace(
			*sourceMessageID,
		)
	}

	item := &models.CoursewareReviewItem{
		CoursewareID:    courseware.ID,
		SourceSessionID: session.ID,
		SourceFindingID: sourceFindingID,

		OriginType: models.
			CWReviewItemOriginGlobalDiscussionManual,
		SourceGlobalMessageID: sourceMessageID,

		SourceType:  sourceType,
		ReviewLevel: session.ReviewLevel,
		ReviewRound: 0,

		CreatedBy: session.ReviewerID,
		OwnerID:   courseware.UserID,

		Severity:  severity,
		Dimension: dimension,

		Title:       title,
		Description: description,

		OriginalSuggestion: instruction,
		Status:             models.CWReviewItemStatusDetected,
	}

	if page != nil {
		pageID :=
			strings.TrimSpace(page.ID)

		item.PageID = &pageID
		item.PageNumberSnapshot =
			page.PageNumber
		item.PageTitleSnapshot =
			strings.TrimSpace(page.Title)
		item.PageHTMLHash =
			cwAIReviewHash(page.HTMLContent)
		item.PageUpdatedAtSnapshot =
			page.UpdatedAt

		evidence["scope"] = "page"
		evidence["page_id"] = pageID
		evidence["page_number_snapshot"] =
			page.PageNumber
	}

	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		return nil,
			fmt.Errorf(
				"序列化人工整改项证据失败: %w",
				err,
			)
	}

	item.EvidenceJSON = string(evidenceJSON)
	return item, nil
}

// findCWAIReviewGlobalSuggestedRelation 只接受可信消息中完全相同的关系方向。
//
// directed关系不得只通过“两个ID都在集合中”的方式匹配。
// conflict已在解析和调用入口中统一规范为UUID文本升序。
func findCWAIReviewGlobalSuggestedRelation(
	relations []CWAIReviewGlobalRelation,
	relationType string,
	sourceItemID string,
	targetItemID string,
) *CWAIReviewGlobalRelation {
	relationType =
		strings.TrimSpace(relationType)
	sourceItemID =
		strings.TrimSpace(sourceItemID)
	targetItemID =
		strings.TrimSpace(targetItemID)

	for index := range relations {
		relation := &relations[index]

		if strings.TrimSpace(
			relation.Type,
		) != relationType {
			continue
		}

		if strings.TrimSpace(
			relation.SourceItemID,
		) != sourceItemID ||
			strings.TrimSpace(
				relation.TargetItemID,
			) != targetItemID {
			continue
		}

		if len(relation.ItemIDs) != 2 ||
			strings.TrimSpace(
				relation.ItemIDs[0],
			) != sourceItemID ||
			strings.TrimSpace(
				relation.ItemIDs[1],
			) != targetItemID {
			continue
		}

		if strings.TrimSpace(
			relation.Explanation,
		) == "" {
			continue
		}

		return relation
	}

	return nil
}

func findCWAIReviewGlobalProposal(
	proposals []CWAIReviewGlobalProposal,
	itemID string,
) *CWAIReviewGlobalProposal {
	itemID = strings.TrimSpace(itemID)

	for index := range proposals {
		proposal := &proposals[index]
		if strings.TrimSpace(
			proposal.ItemID,
		) == itemID {
			return proposal
		}
	}

	return nil
}

func buildCWAIReviewGlobalRelationRecord(
	ctx context.Context,
	relation *models.CoursewareReviewItemRelation,
	actorID string,
) (*CWAIReviewGlobalRelationRecord, error) {
	if relation == nil {
		return nil,
			repository.
				ErrCoursewareReviewItemRelationNotFound
	}

	events, err :=
		repository.
			ListCoursewareReviewItemRelationEvents(
				ctx,
				relation.ID,
				relation.SourceSessionID,
				actorID,
			)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil,
			repository.
				ErrCoursewareReviewItemRelationNotFound
	}

	// 仓储已经按relation_version升序返回。
	// 这里额外排序，防止未来仓储实现变化破坏浏览器时间线。
	sort.SliceStable(
		events,
		func(left int, right int) bool {
			return events[left].
				RelationVersion <
				events[right].
					RelationVersion
		},
	)

	return &CWAIReviewGlobalRelationRecord{
		Relation: relation,
		Events:   events,
	}, nil
}
