package services

// courseware_ai_review_impact_plan_freeze.go
//
// R-07影响方案候选操作的可信冻结总调度与共享辅助。
//
// AI只能提出operation_type / summary / payload。
// 本文件负责：
//   1. 服务端生成稳定operation_id；
//   2. 调度九类候选操作的严格冻结器；
//   3. 提供统一JSON严格解析、指纹与precondition构建辅助；
//   4. 最终返回可写入不可变operations_json的可信操作。
//
// 问题组类操作位于：
//   courseware_ai_review_impact_plan_freeze_group.go
//
// 整改项与关系类操作位于：
//   courseware_ai_review_impact_plan_freeze_item_relation.go
//
// 本层不执行任何业务修改。

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	cwAIReviewImpactGroupNameMaxRunes       = 200
	cwAIReviewImpactReasonMaxRunes          = 500
	cwAIReviewImpactItemTitleMaxRunes       = 300
	cwAIReviewImpactItemDescriptionMaxRunes = 6000
	cwAIReviewImpactItemDimensionMaxRunes   = 64
)

func freezeCWAIReviewImpactOperations(
	aiOperations []cwAIReviewImpactPlanAIOperation,
	items []*models.CoursewareReviewItem,
	pageDigests []models.CWAIReviewPageDigest,
	meta *cwAIReviewGlobalDiscussionMeta,
	groups []*repository.CoursewareReviewImpactGroupSnapshot,
	relations []*repository.CoursewareReviewImpactRelationSnapshot,
) ([]models.CoursewareReviewImpactOperation, error) {
	if meta == nil ||
		len(aiOperations) == 0 ||
		len(aiOperations) > cwAIReviewImpactPlanMaxOperations {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	itemMap := make(
		map[string]*models.CoursewareReviewItem,
		len(items),
	)
	selectedSet := make(
		map[string]struct{},
		len(meta.SelectedItemIDs),
	)

	for _, itemID := range meta.SelectedItemIDs {
		itemID = strings.TrimSpace(itemID)
		if itemID != "" {
			selectedSet[itemID] = struct{}{}
		}
	}

	for _, item := range items {
		if item != nil {
			itemMap[item.ID] = item
		}
	}

	groupMap := make(
		map[string]*repository.CoursewareReviewImpactGroupSnapshot,
		len(groups),
	)
	memberMap := make(
		map[string]repository.CoursewareReviewImpactGroupMemberSnapshot,
	)

	for _, group := range groups {
		if group == nil {
			continue
		}

		groupMap[group.ID] = group

		for _, member := range group.Members {
			memberMap[member.ID] = member
		}
	}

	relationMap := make(
		map[string]*repository.CoursewareReviewImpactRelationSnapshot,
		len(relations),
	)

	for _, relation := range relations {
		if relation != nil {
			relationMap[relation.ID] = relation
		}
	}

	pageMap := make(
		map[string]models.CWAIReviewPageDigest,
		len(pageDigests),
	)

	for _, page := range pageDigests {
		pageMap[page.PageID] = page
	}

	result := make(
		[]models.CoursewareReviewImpactOperation,
		0,
		len(aiOperations),
	)

	for _, aiOperation := range aiOperations {
		operationID, err := newCWAIReviewImpactOperationID()
		if err != nil {
			return nil, err
		}

		operation := models.CoursewareReviewImpactOperation{
			OperationID: operationID,
			OperationType: strings.TrimSpace(
				aiOperation.OperationType,
			),
			Summary: strings.TrimSpace(
				aiOperation.Summary,
			),
		}

		var payload map[string]interface{}
		var preconditions map[string]interface{}

		switch operation.OperationType {
		case models.CWReviewImpactOperationCreateGroup:
			payload, preconditions, err =
				freezeCWAIReviewImpactCreateGroupOperation(
					aiOperation,
					itemMap,
					selectedSet,
					groups,
				)

		case models.CWReviewImpactOperationMoveGroupMember:
			payload, preconditions, err =
				freezeCWAIReviewImpactMoveMemberOperation(
					aiOperation,
					itemMap,
					selectedSet,
					groupMap,
					memberMap,
				)

		case models.CWReviewImpactOperationMergeGroups:
			payload, preconditions, err =
				freezeCWAIReviewImpactMergeGroupsOperation(
					aiOperation,
					itemMap,
					selectedSet,
					groupMap,
				)

		case models.CWReviewImpactOperationSplitGroup:
			payload, preconditions, err =
				freezeCWAIReviewImpactSplitGroupOperation(
					aiOperation,
					itemMap,
					selectedSet,
					groupMap,
				)

		case models.CWReviewImpactOperationCreateRelation:
			payload, preconditions, err =
				freezeCWAIReviewImpactCreateRelationOperation(
					aiOperation,
					itemMap,
					selectedSet,
					meta,
					relations,
				)

		case models.CWReviewImpactOperationCancelRelation:
			payload, preconditions, err =
				freezeCWAIReviewImpactCancelRelationOperation(
					aiOperation,
					selectedSet,
					relationMap,
				)

		case models.CWReviewImpactOperationCreateItem:
			payload, preconditions, err =
				freezeCWAIReviewImpactCreateItemOperation(
					aiOperation,
					pageMap,
				)

		case models.CWReviewImpactOperationDismissItem:
			payload, preconditions, err =
				freezeCWAIReviewImpactDismissItemOperation(
					aiOperation,
					itemMap,
					selectedSet,
					meta,
				)

		case models.CWReviewImpactOperationUpdateCandidateSuggestion:
			payload, preconditions, err =
				freezeCWAIReviewImpactUpdateCandidateOperation(
					aiOperation,
					itemMap,
					selectedSet,
					meta,
				)

		default:
			return nil, ErrCWAIReviewImpactPlanInvalid
		}

		if err != nil {
			return nil, err
		}

		operation.Payload = payload
		operation.Preconditions = preconditions

		result = append(result, operation)
	}

	if err := validateCWAIReviewImpactOperations(result); err != nil {
		return nil, err
	}

	return result, nil
}

func decodeCWAIReviewImpactPayload(
	raw json.RawMessage,
	dest interface{},
) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dest); err != nil {
		return ErrCWAIReviewImpactPlanInvalid
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ErrCWAIReviewImpactPlanInvalid
	}

	return nil
}

func cwAIReviewImpactObjectMap(
	value interface{},
) (map[string]interface{}, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	return result, nil
}

func cwAIReviewImpactFingerprint(
	value interface{},
) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf(
			"生成影响方案业务快照指纹失败: %w",
			err,
		)
	}

	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func cwAIReviewImpactItemPrecondition(
	item *models.CoursewareReviewItem,
) (map[string]interface{}, error) {
	if item == nil {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	fingerprint, err := cwAIReviewImpactFingerprint(item)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"item_id":     item.ID,
		"status":      item.Status,
		"fingerprint": fingerprint,
	}, nil
}

func cwAIReviewImpactGroupPrecondition(
	group *repository.CoursewareReviewImpactGroupSnapshot,
) map[string]interface{} {
	return map[string]interface{}{
		"group_id": group.ID,
		"status":   group.Status,
		"version":  group.Version,
	}
}

func cwAIReviewImpactMemberPrecondition(
	member repository.CoursewareReviewImpactGroupMemberSnapshot,
) map[string]interface{} {
	return map[string]interface{}{
		"member_id": member.ID,
		"group_id":  member.GroupID,
		"item_id":   member.ItemID,
		"status":    member.Status,
		"version":   member.Version,
	}
}

func cwAIReviewImpactRelationPrecondition(
	relation *repository.CoursewareReviewImpactRelationSnapshot,
) map[string]interface{} {
	return map[string]interface{}{
		"relation_id": relation.ID,
		"status":      relation.Status,
		"version":     relation.Version,
	}
}

func requireCWAIReviewImpactSelectedItem(
	itemMap map[string]*models.CoursewareReviewItem,
	selectedSet map[string]struct{},
	itemID string,
) (*models.CoursewareReviewItem, error) {
	itemID = strings.TrimSpace(itemID)

	if _, selected := selectedSet[itemID]; !selected {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	item := itemMap[itemID]
	if item == nil {
		return nil, ErrCWAIReviewImpactPlanConflict
	}

	return item, nil
}

func normalizeCWAIReviewImpactItemIDs(
	values []string,
) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{})

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if _, exists := seen[value]; exists {
			return nil, ErrCWAIReviewImpactPlanInvalid
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	if len(result) == 0 {
		return nil, ErrCWAIReviewImpactPlanInvalid
	}

	return result, nil
}

func containsCWAIReviewImpactString(
	values []string,
	target string,
) bool {
	target = strings.TrimSpace(target)

	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}

func activeMemberForCWAIReviewImpactItem(
	groups []*repository.CoursewareReviewImpactGroupSnapshot,
	itemID string,
) *repository.CoursewareReviewImpactGroupMemberSnapshot {
	for _, group := range groups {
		if group == nil {
			continue
		}

		for index := range group.Members {
			member := &group.Members[index]

			if member.ItemID == itemID &&
				member.Status == "active" {
				return member
			}
		}
	}

	return nil
}

func validCWAIReviewImpactReason(value string) bool {
	value = strings.TrimSpace(value)

	return value != "" &&
		utf8.RuneCountInString(value) <=
			cwAIReviewImpactReasonMaxRunes
}

func newCWAIReviewImpactOperationID() (string, error) {
	var raw [16]byte

	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf(
			"生成影响方案operation_id失败: %w",
			err,
		)
	}

	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(raw[0:4]),
		hex.EncodeToString(raw[4:6]),
		hex.EncodeToString(raw[6:8]),
		hex.EncodeToString(raw[8:10]),
		hex.EncodeToString(raw[10:16]),
	), nil
}
