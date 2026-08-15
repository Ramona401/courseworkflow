package services

// courseware_ai_review_impact_plan_freeze_group.go
//
// R-07问题组类候选操作冻结器。
//
// 只负责：
//   - create_group
//   - move_group_member
//   - merge_groups
//   - split_group
//
// 每个操作都冻结当前group/member CAS版本。
// 所有实际参与操作的整改项同时冻结status与完整服务端指纹。
// 本文件不写数据库。

import (
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

type cwAIReviewImpactCreateGroupPayload struct {
	Name          string   `json:"name"`
	ItemIDs       []string `json:"item_ids"`
	PrimaryItemID string   `json:"primary_item_id"`
	Reason        string   `json:"reason"`
}

type cwAIReviewImpactMoveMemberPayload struct {
	MemberID      string `json:"member_id"`
	TargetGroupID string `json:"target_group_id"`
	Reason        string `json:"reason"`
}

type cwAIReviewImpactMergeGroupsPayload struct {
	SourceGroupID string `json:"source_group_id"`
	TargetGroupID string `json:"target_group_id"`
	Reason        string `json:"reason"`
}

type cwAIReviewImpactSplitGroupPayload struct {
	SourceGroupID string   `json:"source_group_id"`
	Name          string   `json:"name"`
	ItemIDs       []string `json:"item_ids"`
	PrimaryItemID string   `json:"primary_item_id"`
	Reason        string   `json:"reason"`
}

func freezeCWAIReviewImpactCreateGroupOperation(
	aiOperation cwAIReviewImpactPlanAIOperation,
	itemMap map[string]*models.CoursewareReviewItem,
	selectedSet map[string]struct{},
	groups []*repository.CoursewareReviewImpactGroupSnapshot,
) (map[string]interface{}, map[string]interface{}, error) {
	var value cwAIReviewImpactCreateGroupPayload

	if err := decodeCWAIReviewImpactPayload(
		aiOperation.Payload,
		&value,
	); err != nil {
		return nil, nil, err
	}

	value.Name = strings.TrimSpace(value.Name)
	value.PrimaryItemID = strings.TrimSpace(
		value.PrimaryItemID,
	)
	value.Reason = strings.TrimSpace(value.Reason)

	if value.Name == "" ||
		utf8.RuneCountInString(value.Name) >
			cwAIReviewImpactGroupNameMaxRunes ||
		!validCWAIReviewImpactReason(value.Reason) {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	itemIDs, err := normalizeCWAIReviewImpactItemIDs(
		value.ItemIDs,
	)
	if err != nil {
		return nil, nil, err
	}
	value.ItemIDs = itemIDs

	if value.PrimaryItemID != "" &&
		!containsCWAIReviewImpactString(
			value.ItemIDs,
			value.PrimaryItemID,
		) {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	itemPreconditions := make(
		[]map[string]interface{},
		0,
		len(value.ItemIDs),
	)

	for _, itemID := range value.ItemIDs {
		item, err := requireCWAIReviewImpactSelectedItem(
			itemMap,
			selectedSet,
			itemID,
		)
		if err != nil {
			return nil, nil, err
		}

		if activeMemberForCWAIReviewImpactItem(
			groups,
			itemID,
		) != nil {
			return nil, nil, ErrCWAIReviewImpactPlanConflict
		}

		precondition, err :=
			cwAIReviewImpactItemPrecondition(item)
		if err != nil {
			return nil, nil, err
		}

		itemPreconditions = append(
			itemPreconditions,
			precondition,
		)
	}

	payload, err := cwAIReviewImpactObjectMap(value)
	if err != nil {
		return nil, nil, err
	}

	return payload,
		map[string]interface{}{
			"items": itemPreconditions,
		},
		nil
}

func freezeCWAIReviewImpactMoveMemberOperation(
	aiOperation cwAIReviewImpactPlanAIOperation,
	itemMap map[string]*models.CoursewareReviewItem,
	selectedSet map[string]struct{},
	groupMap map[string]*repository.CoursewareReviewImpactGroupSnapshot,
	memberMap map[string]repository.CoursewareReviewImpactGroupMemberSnapshot,
) (map[string]interface{}, map[string]interface{}, error) {
	var value cwAIReviewImpactMoveMemberPayload

	if err := decodeCWAIReviewImpactPayload(
		aiOperation.Payload,
		&value,
	); err != nil {
		return nil, nil, err
	}

	value.MemberID = strings.TrimSpace(value.MemberID)
	value.TargetGroupID = strings.TrimSpace(
		value.TargetGroupID,
	)
	value.Reason = strings.TrimSpace(value.Reason)

	member, exists := memberMap[value.MemberID]
	if !exists ||
		member.Status != "active" ||
		!validCWAIReviewImpactReason(value.Reason) {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	item, err := requireCWAIReviewImpactSelectedItem(
		itemMap,
		selectedSet,
		member.ItemID,
	)
	if err != nil {
		return nil, nil, err
	}

	sourceGroup := groupMap[member.GroupID]
	targetGroup := groupMap[value.TargetGroupID]

	if sourceGroup == nil ||
		targetGroup == nil ||
		sourceGroup.ID == targetGroup.ID ||
		sourceGroup.Status != "active" ||
		targetGroup.Status != "active" {
		return nil, nil, ErrCWAIReviewImpactPlanConflict
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
			"member": cwAIReviewImpactMemberPrecondition(
				member,
			),
			"source_group": cwAIReviewImpactGroupPrecondition(
				sourceGroup,
			),
			"target_group": cwAIReviewImpactGroupPrecondition(
				targetGroup,
			),
			"item": itemPrecondition,
		},
		nil
}

func freezeCWAIReviewImpactMergeGroupsOperation(
	aiOperation cwAIReviewImpactPlanAIOperation,
	itemMap map[string]*models.CoursewareReviewItem,
	selectedSet map[string]struct{},
	groupMap map[string]*repository.CoursewareReviewImpactGroupSnapshot,
) (map[string]interface{}, map[string]interface{}, error) {
	var value cwAIReviewImpactMergeGroupsPayload

	if err := decodeCWAIReviewImpactPayload(
		aiOperation.Payload,
		&value,
	); err != nil {
		return nil, nil, err
	}

	value.SourceGroupID = strings.TrimSpace(
		value.SourceGroupID,
	)
	value.TargetGroupID = strings.TrimSpace(
		value.TargetGroupID,
	)
	value.Reason = strings.TrimSpace(value.Reason)

	sourceGroup := groupMap[value.SourceGroupID]
	targetGroup := groupMap[value.TargetGroupID]

	if sourceGroup == nil ||
		targetGroup == nil ||
		sourceGroup.ID == targetGroup.ID ||
		sourceGroup.Status != "active" ||
		targetGroup.Status != "active" ||
		!validCWAIReviewImpactReason(value.Reason) {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	sourceMembers := make(
		[]map[string]interface{},
		0,
	)
	sourceItems := make(
		[]map[string]interface{},
		0,
	)

	for _, member := range sourceGroup.Members {
		if member.Status != "active" {
			continue
		}

		item, err := requireCWAIReviewImpactSelectedItem(
			itemMap,
			selectedSet,
			member.ItemID,
		)
		if err != nil {
			return nil, nil, err
		}

		itemPrecondition, err :=
			cwAIReviewImpactItemPrecondition(item)
		if err != nil {
			return nil, nil, err
		}

		sourceMembers = append(
			sourceMembers,
			cwAIReviewImpactMemberPrecondition(member),
		)
		sourceItems = append(
			sourceItems,
			itemPrecondition,
		)
	}

	if len(sourceMembers) == 0 {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	payload, err := cwAIReviewImpactObjectMap(value)
	if err != nil {
		return nil, nil, err
	}

	return payload,
		map[string]interface{}{
			"source_group": cwAIReviewImpactGroupPrecondition(
				sourceGroup,
			),
			"target_group": cwAIReviewImpactGroupPrecondition(
				targetGroup,
			),
			"source_members": sourceMembers,
			"source_items":   sourceItems,
		},
		nil
}

func freezeCWAIReviewImpactSplitGroupOperation(
	aiOperation cwAIReviewImpactPlanAIOperation,
	itemMap map[string]*models.CoursewareReviewItem,
	selectedSet map[string]struct{},
	groupMap map[string]*repository.CoursewareReviewImpactGroupSnapshot,
) (map[string]interface{}, map[string]interface{}, error) {
	var value cwAIReviewImpactSplitGroupPayload

	if err := decodeCWAIReviewImpactPayload(
		aiOperation.Payload,
		&value,
	); err != nil {
		return nil, nil, err
	}

	value.SourceGroupID = strings.TrimSpace(
		value.SourceGroupID,
	)
	value.Name = strings.TrimSpace(value.Name)
	value.PrimaryItemID = strings.TrimSpace(
		value.PrimaryItemID,
	)
	value.Reason = strings.TrimSpace(value.Reason)

	if value.Name == "" ||
		utf8.RuneCountInString(value.Name) >
			cwAIReviewImpactGroupNameMaxRunes ||
		!validCWAIReviewImpactReason(value.Reason) {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	sourceGroup := groupMap[value.SourceGroupID]
	if sourceGroup == nil ||
		sourceGroup.Status != "active" {
		return nil, nil, ErrCWAIReviewImpactPlanConflict
	}

	itemIDs, err := normalizeCWAIReviewImpactItemIDs(
		value.ItemIDs,
	)
	if err != nil {
		return nil, nil, err
	}
	value.ItemIDs = itemIDs

	if value.PrimaryItemID != "" &&
		!containsCWAIReviewImpactString(
			value.ItemIDs,
			value.PrimaryItemID,
		) {
		return nil, nil, ErrCWAIReviewImpactPlanInvalid
	}

	memberPreconditions := make(
		[]map[string]interface{},
		0,
		len(value.ItemIDs),
	)
	itemPreconditions := make(
		[]map[string]interface{},
		0,
		len(value.ItemIDs),
	)

	for _, itemID := range value.ItemIDs {
		item, err := requireCWAIReviewImpactSelectedItem(
			itemMap,
			selectedSet,
			itemID,
		)
		if err != nil {
			return nil, nil, err
		}

		member := activeMemberForCWAIReviewImpactItem(
			[]*repository.CoursewareReviewImpactGroupSnapshot{
				sourceGroup,
			},
			itemID,
		)
		if member == nil {
			return nil, nil, ErrCWAIReviewImpactPlanConflict
		}

		itemPrecondition, err :=
			cwAIReviewImpactItemPrecondition(item)
		if err != nil {
			return nil, nil, err
		}

		memberPreconditions = append(
			memberPreconditions,
			cwAIReviewImpactMemberPrecondition(*member),
		)
		itemPreconditions = append(
			itemPreconditions,
			itemPrecondition,
		)
	}

	payload, err := cwAIReviewImpactObjectMap(value)
	if err != nil {
		return nil, nil, err
	}

	return payload,
		map[string]interface{}{
			"source_group": cwAIReviewImpactGroupPrecondition(
				sourceGroup,
			),
			"members": memberPreconditions,
			"items":   itemPreconditions,
		},
		nil
}
