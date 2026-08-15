package services

// courseware_ai_review_impact_plan_freeze_item_relation.go
//
// R-07整改项与pairwise relation候选操作冻结器。
//
// 只负责：
//   - create_relation
//   - cancel_relation
//   - create_item
//   - dismiss_item
//   - update_candidate_suggestion
//
// create_relation / dismiss_item / update_candidate_suggestion都必须重新与
// 可信assistant消息中的relations/proposals逐字核对。
// candidate suggestion永远只是候选消息，不形成confirmed instruction。
//
// create_item页面前置条件不保存Service内部digest整体指纹，而保存Repository
// 可以在最终pgx.Tx内重新核对的稳定事实：page_id/page_number/title/html_hash。
//
// 本文件不写数据库。

import (
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

type cwAIReviewImpactCreateRelationPayload struct {
	RelationType string `json:"relation_type"`
	SourceItemID string `json:"source_item_id"`
	TargetItemID string `json:"target_item_id"`
	Explanation  string `json:"explanation"`
}

type cwAIReviewImpactCancelRelationPayload struct {
	RelationID string `json:"relation_id"`
	Reason     string `json:"reason"`
}

type cwAIReviewImpactCreateItemPayload struct {
	PageID               string `json:"page_id"`
	Severity             string `json:"severity"`
	Dimension            string `json:"dimension"`
	Title                string `json:"title"`
	Description          string `json:"description"`
	CandidateInstruction string `json:"candidate_instruction"`
}

type cwAIReviewImpactDismissItemPayload struct {
	ItemID string `json:"item_id"`
	Reason string `json:"reason"`
}

type cwAIReviewImpactUpdateCandidatePayload struct {
	ItemID               string `json:"item_id"`
	CandidateInstruction string `json:"candidate_instruction"`
}

func freezeCWAIReviewImpactCreateRelationOperation(
	aiOperation cwAIReviewImpactPlanAIOperation,
	itemMap map[string]*models.CoursewareReviewItem,
	selectedSet map[string]struct{},
	meta *cwAIReviewGlobalDiscussionMeta,
	relations []*repository.CoursewareReviewImpactRelationSnapshot,
) (map[string]interface{}, map[string]interface{}, error) {
	var value cwAIReviewImpactCreateRelationPayload

	if err := decodeCWAIReviewImpactPayload(
		aiOperation.Payload,
		&value,
	); err != nil {
		return nil, nil, err
	}

	normalizeCWAIReviewImpactRelationPayload(&value)

	if !models.IsCWReviewItemRelationType(
		value.RelationType,
	) ||
		value.SourceItemID == "" ||
		value.TargetItemID == "" ||
		value.SourceItemID == value.TargetItemID ||
		value.Explanation == "" {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	sourceItem, err := requireCWAIReviewImpactSelectedItem(
		itemMap,
		selectedSet,
		value.SourceItemID,
	)
	if err != nil {
		return nil, nil, err
	}

	targetItem, err := requireCWAIReviewImpactSelectedItem(
		itemMap,
		selectedSet,
		value.TargetItemID,
	)
	if err != nil {
		return nil, nil, err
	}

	if !trustedCWAIReviewImpactRelationExists(
		meta,
		value,
	) {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	existingRelation :=
		findCWAIReviewImpactRelationSnapshot(
			relations,
			value.RelationType,
			value.SourceItemID,
			value.TargetItemID,
		)

	relationPrecondition := map[string]interface{}{
		"expected_absent": true,
	}

	if existingRelation != nil {
		if existingRelation.Status == "active" {
			return nil, nil, ErrCWAIReviewImpactPlanConflict
		}

		relationPrecondition =
			cwAIReviewImpactRelationPrecondition(
				existingRelation,
			)
	}

	sourcePrecondition, err :=
		cwAIReviewImpactItemPrecondition(sourceItem)
	if err != nil {
		return nil, nil, err
	}

	targetPrecondition, err :=
		cwAIReviewImpactItemPrecondition(targetItem)
	if err != nil {
		return nil, nil, err
	}

	payload, err := cwAIReviewImpactObjectMap(value)
	if err != nil {
		return nil, nil, err
	}

	return payload,
		map[string]interface{}{
			"source_item": sourcePrecondition,
			"target_item": targetPrecondition,
			"relation":    relationPrecondition,
		},
		nil
}

func freezeCWAIReviewImpactCancelRelationOperation(
	aiOperation cwAIReviewImpactPlanAIOperation,
	selectedSet map[string]struct{},
	relationMap map[string]*repository.CoursewareReviewImpactRelationSnapshot,
) (map[string]interface{}, map[string]interface{}, error) {
	var value cwAIReviewImpactCancelRelationPayload

	if err := decodeCWAIReviewImpactPayload(
		aiOperation.Payload,
		&value,
	); err != nil {
		return nil, nil, err
	}

	value.RelationID = strings.TrimSpace(
		value.RelationID,
	)
	value.Reason = strings.TrimSpace(value.Reason)

	relation := relationMap[value.RelationID]
	if relation == nil ||
		relation.Status != "active" ||
		!validCWAIReviewImpactReason(value.Reason) {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	if _, selected :=
		selectedSet[relation.SourceItemID]; !selected {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	if _, selected :=
		selectedSet[relation.TargetItemID]; !selected {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	payload, err := cwAIReviewImpactObjectMap(value)
	if err != nil {
		return nil, nil, err
	}

	return payload,
		map[string]interface{}{
			"relation": cwAIReviewImpactRelationPrecondition(
				relation,
			),
		},
		nil
}

func freezeCWAIReviewImpactCreateItemOperation(
	aiOperation cwAIReviewImpactPlanAIOperation,
	pageMap map[string]models.CWAIReviewPageDigest,
) (map[string]interface{}, map[string]interface{}, error) {
	var value cwAIReviewImpactCreateItemPayload

	if err := decodeCWAIReviewImpactPayload(
		aiOperation.Payload,
		&value,
	); err != nil {
		return nil, nil, err
	}

	normalizeCWAIReviewImpactCreateItemPayload(&value)

	if value.Severity == "" {
		value.Severity = models.CWReviewSeverityMedium
	}

	if value.Dimension == "" {
		value.Dimension = "manual_review"
	}

	if !models.IsCWReviewSeverity(value.Severity) ||
		value.Title == "" ||
		value.Description == "" ||
		utf8.RuneCountInString(value.Title) >
			cwAIReviewImpactItemTitleMaxRunes ||
		utf8.RuneCountInString(value.Description) >
			cwAIReviewImpactItemDescriptionMaxRunes ||
		utf8.RuneCountInString(value.Dimension) >
			cwAIReviewImpactItemDimensionMaxRunes ||
		utf8.RuneCountInString(
			value.CandidateInstruction,
		) > cwReviewItemMaxInstructionRunes {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	pagePrecondition := map[string]interface{}{
		"scope": "global",
	}

	if value.PageID != "" {
		page, exists := pageMap[value.PageID]
		if !exists {
			return nil, nil, ErrCWAIReviewImpactPlanConflict
		}

		htmlHash := strings.TrimSpace(page.HTMLHash)
		if htmlHash == "" {
			return nil, nil, ErrCWAIReviewImpactPlanConflict
		}

		pagePrecondition = map[string]interface{}{
			"scope":       "page",
			"page_id":     value.PageID,
			"page_number": page.PageNumber,
			"title":       strings.TrimSpace(page.Title),
			"html_hash":   htmlHash,
		}
	}

	payload, err := cwAIReviewImpactObjectMap(value)
	if err != nil {
		return nil, nil, err
	}

	return payload,
		map[string]interface{}{
			"page": pagePrecondition,
		},
		nil
}

func freezeCWAIReviewImpactDismissItemOperation(
	aiOperation cwAIReviewImpactPlanAIOperation,
	itemMap map[string]*models.CoursewareReviewItem,
	selectedSet map[string]struct{},
	meta *cwAIReviewGlobalDiscussionMeta,
) (map[string]interface{}, map[string]interface{}, error) {
	var value cwAIReviewImpactDismissItemPayload

	if err := decodeCWAIReviewImpactPayload(
		aiOperation.Payload,
		&value,
	); err != nil {
		return nil, nil, err
	}

	value.ItemID = strings.TrimSpace(value.ItemID)
	value.Reason = strings.TrimSpace(value.Reason)

	item, err := requireCWAIReviewImpactSelectedItem(
		itemMap,
		selectedSet,
		value.ItemID,
	)
	if err != nil {
		return nil, nil, err
	}

	if !validCWAIReviewImpactReason(value.Reason) ||
		!trustedCWAIReviewImpactDismissalExists(
			meta,
			value.ItemID,
		) {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	itemPrecondition, err :=
		cwAIReviewImpactItemPrecondition(item)
	if err != nil {
		return nil, nil, err
	}

	payload, err := cwAIReviewImpactObjectMap(value)
	if err != nil {
		return nil, nil, err
	}

	return payload,
		map[string]interface{}{
			"item": itemPrecondition,
		},
		nil
}

func freezeCWAIReviewImpactUpdateCandidateOperation(
	aiOperation cwAIReviewImpactPlanAIOperation,
	itemMap map[string]*models.CoursewareReviewItem,
	selectedSet map[string]struct{},
	meta *cwAIReviewGlobalDiscussionMeta,
) (map[string]interface{}, map[string]interface{}, error) {
	var value cwAIReviewImpactUpdateCandidatePayload

	if err := decodeCWAIReviewImpactPayload(
		aiOperation.Payload,
		&value,
	); err != nil {
		return nil, nil, err
	}

	value.ItemID = strings.TrimSpace(value.ItemID)
	value.CandidateInstruction = strings.TrimSpace(
		value.CandidateInstruction,
	)

	item, err := requireCWAIReviewImpactSelectedItem(
		itemMap,
		selectedSet,
		value.ItemID,
	)
	if err != nil {
		return nil, nil, err
	}

	if value.CandidateInstruction == "" ||
		utf8.RuneCountInString(
			value.CandidateInstruction,
		) > cwReviewItemMaxInstructionRunes ||
		!trustedCWAIReviewImpactCandidateExists(
			meta,
			value.ItemID,
			value.CandidateInstruction,
		) {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	itemPrecondition, err :=
		cwAIReviewImpactItemPrecondition(item)
	if err != nil {
		return nil, nil, err
	}

	payload, err := cwAIReviewImpactObjectMap(value)
	if err != nil {
		return nil, nil, err
	}

	return payload,
		map[string]interface{}{
			"item": itemPrecondition,
		},
		nil
}

func normalizeCWAIReviewImpactCreateItemPayload(
	value *cwAIReviewImpactCreateItemPayload,
) {
	value.PageID = strings.TrimSpace(value.PageID)
	value.Severity = strings.TrimSpace(value.Severity)
	value.Dimension = strings.TrimSpace(value.Dimension)
	value.Title = strings.TrimSpace(value.Title)
	value.Description = strings.TrimSpace(value.Description)
	value.CandidateInstruction = strings.TrimSpace(
		value.CandidateInstruction,
	)
}

func normalizeCWAIReviewImpactRelationPayload(
	value *cwAIReviewImpactCreateRelationPayload,
) {
	value.RelationType = strings.TrimSpace(
		value.RelationType,
	)
	value.SourceItemID = strings.TrimSpace(
		value.SourceItemID,
	)
	value.TargetItemID = strings.TrimSpace(
		value.TargetItemID,
	)
	value.Explanation = strings.TrimSpace(
		value.Explanation,
	)

	if value.RelationType ==
		models.CWReviewItemRelationConflict &&
		value.SourceItemID > value.TargetItemID {
		value.SourceItemID,
			value.TargetItemID =
			value.TargetItemID,
			value.SourceItemID
	}
}

func trustedCWAIReviewImpactCandidateExists(
	meta *cwAIReviewGlobalDiscussionMeta,
	itemID string,
	instruction string,
) bool {
	for _, proposal := range meta.Proposals {
		if strings.TrimSpace(
			proposal.ItemID,
		) == itemID &&
			strings.TrimSpace(
				proposal.SuggestedInstruction,
			) == instruction {
			return true
		}
	}

	return false
}

func trustedCWAIReviewImpactDismissalExists(
	meta *cwAIReviewGlobalDiscussionMeta,
	itemID string,
) bool {
	for _, proposal := range meta.Proposals {
		if strings.TrimSpace(
			proposal.ItemID,
		) == itemID &&
			strings.TrimSpace(
				proposal.Recommendation,
			) == "consider_dismiss" {
			return true
		}
	}

	return false
}

func trustedCWAIReviewImpactRelationExists(
	meta *cwAIReviewGlobalDiscussionMeta,
	value cwAIReviewImpactCreateRelationPayload,
) bool {
	for _, relation := range meta.Relations {
		relationType := strings.TrimSpace(
			relation.Type,
		)
		sourceID := strings.TrimSpace(
			relation.SourceItemID,
		)
		targetID := strings.TrimSpace(
			relation.TargetItemID,
		)

		if relationType ==
			models.CWReviewItemRelationConflict &&
			sourceID > targetID {
			sourceID, targetID =
				targetID, sourceID
		}

		if relationType == value.RelationType &&
			sourceID == value.SourceItemID &&
			targetID == value.TargetItemID &&
			strings.TrimSpace(
				relation.Explanation,
			) == value.Explanation {
			return true
		}
	}

	return false
}

func findCWAIReviewImpactRelationSnapshot(
	relations []*repository.CoursewareReviewImpactRelationSnapshot,
	relationType string,
	sourceItemID string,
	targetItemID string,
) *repository.CoursewareReviewImpactRelationSnapshot {
	for _, relation := range relations {
		if relation == nil {
			continue
		}

		sourceID := relation.SourceItemID
		targetID := relation.TargetItemID

		if relation.RelationType ==
			models.CWReviewItemRelationConflict &&
			sourceID > targetID {
			sourceID, targetID =
				targetID, sourceID
		}

		if relation.RelationType == relationType &&
			sourceID == sourceItemID &&
			targetID == targetItemID {
			return relation
		}
	}

	return nil
}
