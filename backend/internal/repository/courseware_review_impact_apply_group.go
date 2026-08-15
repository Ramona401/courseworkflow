package repository

// courseware_review_impact_apply_group.go
//
// R-07 Atomic Apply问题组操作的类型、准备与资源声明层。
//
// 负责：
//   1. 四类group operation的payload/preconditions类型；
//   2. 严格解析数据库冻结的payload/preconditions；
//   3. 声明每个operation会使用的现有group/item/member；
//   4. 在事务写入前拒绝同一份V1方案内互相覆盖的group/item操作；
//   5. 提供规范化和简单集合辅助。
//
// 事务锁定与最终precondition复核位于:
//   courseware_review_impact_apply_group_validate.go
//
// 真正写入位于:
//   courseware_review_impact_apply_group_execute.go
//   courseware_review_impact_apply_group_merge_split.go

import (
	"strings"

	"tedna/internal/models"
)

type cwReviewImpactCreateGroupPayload struct {
	Name          string   `json:"name"`
	ItemIDs       []string `json:"item_ids"`
	PrimaryItemID string   `json:"primary_item_id"`
	Reason        string   `json:"reason"`
}

type cwReviewImpactMoveMemberPayload struct {
	MemberID      string `json:"member_id"`
	TargetGroupID string `json:"target_group_id"`
	Reason        string `json:"reason"`
}

type cwReviewImpactMergeGroupsPayload struct {
	SourceGroupID string `json:"source_group_id"`
	TargetGroupID string `json:"target_group_id"`
	Reason        string `json:"reason"`
}

type cwReviewImpactSplitGroupPayload struct {
	SourceGroupID string   `json:"source_group_id"`
	Name          string   `json:"name"`
	ItemIDs       []string `json:"item_ids"`
	PrimaryItemID string   `json:"primary_item_id"`
	Reason        string   `json:"reason"`
}

type cwReviewImpactCreateGroupPreconditions struct {
	Items []cwReviewImpactItemPrecondition `json:"items"`
}

type cwReviewImpactMoveMemberPreconditions struct {
	Member      cwReviewImpactMemberPrecondition `json:"member"`
	SourceGroup cwReviewImpactGroupPrecondition  `json:"source_group"`
	TargetGroup cwReviewImpactGroupPrecondition  `json:"target_group"`
	Item        cwReviewImpactItemPrecondition   `json:"item"`
}

type cwReviewImpactMergeGroupsPreconditions struct {
	SourceGroup   cwReviewImpactGroupPrecondition    `json:"source_group"`
	TargetGroup   cwReviewImpactGroupPrecondition    `json:"target_group"`
	SourceMembers []cwReviewImpactMemberPrecondition `json:"source_members"`
	SourceItems   []cwReviewImpactItemPrecondition   `json:"source_items"`
}

type cwReviewImpactSplitGroupPreconditions struct {
	SourceGroup cwReviewImpactGroupPrecondition    `json:"source_group"`
	Members     []cwReviewImpactMemberPrecondition `json:"members"`
	Items       []cwReviewImpactItemPrecondition   `json:"items"`
}

type cwReviewImpactPreparedGroupOperation struct {
	OperationID   string
	OperationType string

	CreateGroup *cwReviewImpactPreparedCreateGroup
	MoveMember  *cwReviewImpactPreparedMoveMember
	MergeGroups *cwReviewImpactPreparedMergeGroups
	SplitGroup  *cwReviewImpactPreparedSplitGroup
}

type cwReviewImpactPreparedCreateGroup struct {
	Payload       cwReviewImpactCreateGroupPayload
	Preconditions cwReviewImpactCreateGroupPreconditions
}

type cwReviewImpactPreparedMoveMember struct {
	Payload       cwReviewImpactMoveMemberPayload
	Preconditions cwReviewImpactMoveMemberPreconditions
}

type cwReviewImpactPreparedMergeGroups struct {
	Payload       cwReviewImpactMergeGroupsPayload
	Preconditions cwReviewImpactMergeGroupsPreconditions
}

type cwReviewImpactPreparedSplitGroup struct {
	Payload       cwReviewImpactSplitGroupPayload
	Preconditions cwReviewImpactSplitGroupPreconditions
}

func isCoursewareReviewImpactGroupOperation(
	operationType string,
) bool {
	switch strings.TrimSpace(operationType) {
	case models.CWReviewImpactOperationCreateGroup,
		models.CWReviewImpactOperationMoveGroupMember,
		models.CWReviewImpactOperationMergeGroups,
		models.CWReviewImpactOperationSplitGroup:
		return true

	default:
		return false
	}
}

func prepareCoursewareReviewImpactGroupOperation(
	operation models.CoursewareReviewImpactOperation,
) (cwReviewImpactPreparedGroupOperation, error) {
	prepared := cwReviewImpactPreparedGroupOperation{
		OperationID:   operation.OperationID,
		OperationType: operation.OperationType,
	}

	switch operation.OperationType {
	case models.CWReviewImpactOperationCreateGroup:
		value := &cwReviewImpactPreparedCreateGroup{}

		if err := decodeCoursewareReviewImpactMap(
			operation.Payload,
			&value.Payload,
		); err != nil {
			return prepared, err
		}

		if err := decodeCoursewareReviewImpactMap(
			operation.Preconditions,
			&value.Preconditions,
		); err != nil {
			return prepared, err
		}

		normalizePreparedCreateGroup(value)
		prepared.CreateGroup = value

	case models.CWReviewImpactOperationMoveGroupMember:
		value := &cwReviewImpactPreparedMoveMember{}

		if err := decodeCoursewareReviewImpactMap(
			operation.Payload,
			&value.Payload,
		); err != nil {
			return prepared, err
		}

		if err := decodeCoursewareReviewImpactMap(
			operation.Preconditions,
			&value.Preconditions,
		); err != nil {
			return prepared, err
		}

		normalizePreparedMoveMember(value)
		prepared.MoveMember = value

	case models.CWReviewImpactOperationMergeGroups:
		value := &cwReviewImpactPreparedMergeGroups{}

		if err := decodeCoursewareReviewImpactMap(
			operation.Payload,
			&value.Payload,
		); err != nil {
			return prepared, err
		}

		if err := decodeCoursewareReviewImpactMap(
			operation.Preconditions,
			&value.Preconditions,
		); err != nil {
			return prepared, err
		}

		normalizePreparedMergeGroups(value)
		prepared.MergeGroups = value

	case models.CWReviewImpactOperationSplitGroup:
		value := &cwReviewImpactPreparedSplitGroup{}

		if err := decodeCoursewareReviewImpactMap(
			operation.Payload,
			&value.Payload,
		); err != nil {
			return prepared, err
		}

		if err := decodeCoursewareReviewImpactMap(
			operation.Preconditions,
			&value.Preconditions,
		); err != nil {
			return prepared, err
		}

		normalizePreparedSplitGroup(value)
		prepared.SplitGroup = value

	default:
		return prepared,
			ErrCoursewareReviewImpactOperationUnsupported
	}

	return prepared, nil
}

func claimCoursewareReviewImpactGroupOperation(
	operation cwReviewImpactPreparedGroupOperation,
	groupClaims map[string]string,
	itemClaims map[string]string,
	groupIDs map[string]struct{},
	itemIDs map[string]struct{},
	memberIDs map[string]struct{},
) error {
	claimGroup := func(groupID string) error {
		groupID = strings.TrimSpace(groupID)
		if groupID == "" {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if owner, exists := groupClaims[groupID]; exists &&
			owner != operation.OperationID {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		groupClaims[groupID] = operation.OperationID
		groupIDs[groupID] = struct{}{}

		return nil
	}

	claimItem := func(itemID string) error {
		itemID = strings.TrimSpace(itemID)
		if itemID == "" {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if owner, exists := itemClaims[itemID]; exists &&
			owner != operation.OperationID {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		itemClaims[itemID] = operation.OperationID
		itemIDs[itemID] = struct{}{}

		return nil
	}

	switch operation.OperationType {
	case models.CWReviewImpactOperationCreateGroup:
		if operation.CreateGroup == nil {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		for _, item := range operation.CreateGroup.Preconditions.Items {
			if err := claimItem(item.ItemID); err != nil {
				return err
			}
		}

	case models.CWReviewImpactOperationMoveGroupMember:
		if operation.MoveMember == nil {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		value := operation.MoveMember.Preconditions

		if err := claimGroup(
			value.SourceGroup.GroupID,
		); err != nil {
			return err
		}

		if err := claimGroup(
			value.TargetGroup.GroupID,
		); err != nil {
			return err
		}

		if err := claimItem(value.Item.ItemID); err != nil {
			return err
		}

		memberIDs[value.Member.MemberID] = struct{}{}

	case models.CWReviewImpactOperationMergeGroups:
		if operation.MergeGroups == nil {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		value := operation.MergeGroups.Preconditions

		if err := claimGroup(
			value.SourceGroup.GroupID,
		); err != nil {
			return err
		}

		if err := claimGroup(
			value.TargetGroup.GroupID,
		); err != nil {
			return err
		}

		for _, item := range value.SourceItems {
			if err := claimItem(item.ItemID); err != nil {
				return err
			}
		}

		for _, member := range value.SourceMembers {
			memberIDs[member.MemberID] = struct{}{}
		}

	case models.CWReviewImpactOperationSplitGroup:
		if operation.SplitGroup == nil {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		value := operation.SplitGroup.Preconditions

		if err := claimGroup(
			value.SourceGroup.GroupID,
		); err != nil {
			return err
		}

		for _, item := range value.Items {
			if err := claimItem(item.ItemID); err != nil {
				return err
			}
		}

		for _, member := range value.Members {
			memberIDs[member.MemberID] = struct{}{}
		}
	}

	return nil
}

func normalizePreparedCreateGroup(
	value *cwReviewImpactPreparedCreateGroup,
) {
	value.Payload.Name = strings.TrimSpace(
		value.Payload.Name,
	)
	value.Payload.PrimaryItemID = strings.TrimSpace(
		value.Payload.PrimaryItemID,
	)
	value.Payload.Reason = strings.TrimSpace(
		value.Payload.Reason,
	)

	for index := range value.Payload.ItemIDs {
		value.Payload.ItemIDs[index] =
			strings.TrimSpace(
				value.Payload.ItemIDs[index],
			)
	}
}

func normalizePreparedMoveMember(
	value *cwReviewImpactPreparedMoveMember,
) {
	value.Payload.MemberID = strings.TrimSpace(
		value.Payload.MemberID,
	)
	value.Payload.TargetGroupID = strings.TrimSpace(
		value.Payload.TargetGroupID,
	)
	value.Payload.Reason = strings.TrimSpace(
		value.Payload.Reason,
	)
}

func normalizePreparedMergeGroups(
	value *cwReviewImpactPreparedMergeGroups,
) {
	value.Payload.SourceGroupID = strings.TrimSpace(
		value.Payload.SourceGroupID,
	)
	value.Payload.TargetGroupID = strings.TrimSpace(
		value.Payload.TargetGroupID,
	)
	value.Payload.Reason = strings.TrimSpace(
		value.Payload.Reason,
	)
}

func normalizePreparedSplitGroup(
	value *cwReviewImpactPreparedSplitGroup,
) {
	value.Payload.SourceGroupID = strings.TrimSpace(
		value.Payload.SourceGroupID,
	)
	value.Payload.Name = strings.TrimSpace(
		value.Payload.Name,
	)
	value.Payload.PrimaryItemID = strings.TrimSpace(
		value.Payload.PrimaryItemID,
	)
	value.Payload.Reason = strings.TrimSpace(
		value.Payload.Reason,
	)

	for index := range value.Payload.ItemIDs {
		value.Payload.ItemIDs[index] =
			strings.TrimSpace(
				value.Payload.ItemIDs[index],
			)
	}
}

func containsString(
	values []string,
	target string,
) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
