package repository

// courseware_review_impact_apply_group_validate.go
//
// R-07 Atomic Apply四类问题组操作的事务前置验证。
//
// 绝对规则：
//   1. 所有选中operation先完成解析和资源冲突声明；
//   2. 所有现有group按稳定ID顺序加锁；
//   3. group治理成员和整改项随后锁定；
//   4. item/member也按稳定ID顺序补充锁定；
//   5. 全部冻结preconditions通过后，外层才允许开始任何业务写入。

import (
	"context"

	"github.com/jackc/pgx/v5"

	"tedna/internal/models"
)

func prevalidateCoursewareReviewImpactGroupOperationsTx(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operations []models.CoursewareReviewImpactOperation,
	actorID string,
) ([]cwReviewImpactPreparedGroupOperation, error) {
	prepared := make(
		[]cwReviewImpactPreparedGroupOperation,
		0,
		len(operations),
	)

	groupClaims := make(map[string]string)
	itemClaims := make(map[string]string)

	groupIDs := make(map[string]struct{})
	itemIDs := make(map[string]struct{})
	memberIDs := make(map[string]struct{})

	for _, operation := range operations {
		preparedOperation, err :=
			prepareCoursewareReviewImpactGroupOperation(
				operation,
			)
		if err != nil {
			return nil, err
		}

		if err := claimCoursewareReviewImpactGroupOperation(
			preparedOperation,
			groupClaims,
			itemClaims,
			groupIDs,
			itemIDs,
			memberIDs,
		); err != nil {
			return nil, err
		}

		prepared = append(
			prepared,
			preparedOperation,
		)
	}

	lockedGroups := make(
		map[string]*models.CoursewareReviewItemGroup,
		len(groupIDs),
	)

	for _, groupID := range sortedCoursewareReviewImpactKeys(
		groupIDs,
	) {
		group, err := lockCWReviewItemGroupTx(
			ctx,
			tx,
			groupID,
			plan.SourceSessionID,
			actorID,
		)
		if err != nil {
			return nil, mapCoursewareReviewImpactGroupError(err)
		}

		if group.CoursewareID != plan.CoursewareID {
			return nil,
				ErrCoursewareReviewImpactPlanConflict
		}

		lockedGroups[groupID] = group
	}

	// 先稳定现有group的成员集合和全部治理整改项。
	for _, groupID := range sortedCoursewareReviewImpactKeys(
		groupIDs,
	) {
		if err := lockCWReviewItemGroupGovernableItemsTx(
			ctx,
			tx,
			lockedGroups[groupID],
		); err != nil {
			return nil, mapCoursewareReviewImpactGroupError(err)
		}
	}

	lockedItems := make(
		map[string]*models.CoursewareReviewItem,
		len(itemIDs),
	)

	for _, itemID := range sortedCoursewareReviewImpactKeys(
		itemIDs,
	) {
		item, err := lockCoursewareReviewImpactItemTx(
			ctx,
			tx,
			itemID,
		)
		if err != nil {
			return nil, err
		}

		lockedItems[itemID] = item
	}

	lockedMembers := make(
		map[string]*models.CoursewareReviewItemGroupMember,
		len(memberIDs),
	)

	for _, memberID := range sortedCoursewareReviewImpactKeys(
		memberIDs,
	) {
		member, err := lockCWReviewItemGroupMemberTx(
			ctx,
			tx,
			memberID,
			plan.SourceSessionID,
			actorID,
		)
		if err != nil {
			return nil, mapCoursewareReviewImpactGroupError(err)
		}

		lockedMembers[memberID] = member
	}

	for _, operation := range prepared {
		if err := validatePreparedCoursewareReviewImpactGroupOperation(
			ctx,
			tx,
			plan,
			operation,
			actorID,
			lockedGroups,
			lockedItems,
			lockedMembers,
		); err != nil {
			return nil, err
		}
	}

	return prepared, nil
}

func validatePreparedCoursewareReviewImpactGroupOperation(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	operation cwReviewImpactPreparedGroupOperation,
	actorID string,
	groups map[string]*models.CoursewareReviewItemGroup,
	items map[string]*models.CoursewareReviewItem,
	members map[string]*models.CoursewareReviewItemGroupMember,
) error {
	switch operation.OperationType {
	case models.CWReviewImpactOperationCreateGroup:
		return validatePreparedImpactCreateGroup(
			ctx,
			tx,
			plan,
			operation.CreateGroup,
			actorID,
			items,
		)

	case models.CWReviewImpactOperationMoveGroupMember:
		return validatePreparedImpactMoveMember(
			plan,
			operation.MoveMember,
			actorID,
			groups,
			items,
			members,
		)

	case models.CWReviewImpactOperationMergeGroups:
		return validatePreparedImpactMergeGroups(
			plan,
			operation.MergeGroups,
			actorID,
			groups,
			items,
			members,
		)

	case models.CWReviewImpactOperationSplitGroup:
		return validatePreparedImpactSplitGroup(
			plan,
			operation.SplitGroup,
			actorID,
			groups,
			items,
			members,
		)

	default:
		return ErrCoursewareReviewImpactOperationUnsupported
	}
}

func validatePreparedImpactCreateGroup(
	ctx context.Context,
	tx pgx.Tx,
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedCreateGroup,
	actorID string,
	items map[string]*models.CoursewareReviewItem,
) error {
	if value == nil ||
		value.Payload.Name == "" ||
		len(value.Payload.ItemIDs) == 0 ||
		len(value.Preconditions.Items) !=
			len(value.Payload.ItemIDs) {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	preconditionByItem := make(
		map[string]cwReviewImpactItemPrecondition,
		len(value.Preconditions.Items),
	)

	for _, precondition := range value.Preconditions.Items {
		if precondition.ItemID == "" {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if _, exists :=
			preconditionByItem[precondition.ItemID]; exists {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		preconditionByItem[precondition.ItemID] =
			precondition
	}

	for _, itemID := range value.Payload.ItemIDs {
		precondition, exists :=
			preconditionByItem[itemID]
		if !exists {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if err := validateCoursewareReviewImpactItemPrecondition(
			plan,
			items[itemID],
			precondition,
			actorID,
		); err != nil {
			return err
		}

		member, err := lockCWReviewItemGroupMemberByItemTx(
			ctx,
			tx,
			itemID,
			plan.CoursewareID,
			plan.SourceSessionID,
			actorID,
		)
		if err != nil {
			return mapCoursewareReviewImpactGroupError(err)
		}

		if member != nil &&
			member.Status ==
				models.CWReviewItemGroupMemberStatusActive {
			return ErrCoursewareReviewImpactPlanConflict
		}
	}

	if value.Payload.PrimaryItemID != "" &&
		!containsString(
			value.Payload.ItemIDs,
			value.Payload.PrimaryItemID,
		) {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	return nil
}

func validatePreparedImpactMoveMember(
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedMoveMember,
	actorID string,
	groups map[string]*models.CoursewareReviewItemGroup,
	items map[string]*models.CoursewareReviewItem,
	members map[string]*models.CoursewareReviewItemGroupMember,
) error {
	if value == nil {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	preconditions := value.Preconditions

	if value.Payload.MemberID !=
		preconditions.Member.MemberID ||
		value.Payload.TargetGroupID !=
			preconditions.TargetGroup.GroupID ||
		preconditions.Member.GroupID !=
			preconditions.SourceGroup.GroupID ||
		preconditions.Member.ItemID !=
			preconditions.Item.ItemID {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	if err := validateImpactGroupPrecondition(
		groups[preconditions.SourceGroup.GroupID],
		preconditions.SourceGroup,
	); err != nil {
		return err
	}

	if err := validateImpactGroupPrecondition(
		groups[preconditions.TargetGroup.GroupID],
		preconditions.TargetGroup,
	); err != nil {
		return err
	}

	if err := validateImpactMemberPrecondition(
		members[preconditions.Member.MemberID],
		preconditions.Member,
	); err != nil {
		return err
	}

	return validateCoursewareReviewImpactItemPrecondition(
		plan,
		items[preconditions.Item.ItemID],
		preconditions.Item,
		actorID,
	)
}

func validatePreparedImpactMergeGroups(
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedMergeGroups,
	actorID string,
	groups map[string]*models.CoursewareReviewItemGroup,
	items map[string]*models.CoursewareReviewItem,
	members map[string]*models.CoursewareReviewItemGroupMember,
) error {
	if value == nil ||
		len(value.Preconditions.SourceMembers) == 0 ||
		len(value.Preconditions.SourceMembers) !=
			len(value.Preconditions.SourceItems) {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	if value.Payload.SourceGroupID !=
		value.Preconditions.SourceGroup.GroupID ||
		value.Payload.TargetGroupID !=
			value.Preconditions.TargetGroup.GroupID {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	if err := validateImpactGroupPrecondition(
		groups[value.Preconditions.SourceGroup.GroupID],
		value.Preconditions.SourceGroup,
	); err != nil {
		return err
	}

	if err := validateImpactGroupPrecondition(
		groups[value.Preconditions.TargetGroup.GroupID],
		value.Preconditions.TargetGroup,
	); err != nil {
		return err
	}

	itemPreconditionByID := make(
		map[string]cwReviewImpactItemPrecondition,
		len(value.Preconditions.SourceItems),
	)

	for _, item := range value.Preconditions.SourceItems {
		if item.ItemID == "" {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if _, exists :=
			itemPreconditionByID[item.ItemID]; exists {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		itemPreconditionByID[item.ItemID] = item
	}

	seenMembers := make(
		map[string]struct{},
		len(value.Preconditions.SourceMembers),
	)

	for _, memberPrecondition := range value.Preconditions.SourceMembers {
		if _, exists :=
			seenMembers[memberPrecondition.MemberID]; exists {
			return ErrCoursewareReviewImpactSelectionInvalid
		}
		seenMembers[memberPrecondition.MemberID] = struct{}{}

		member := members[memberPrecondition.MemberID]

		if err := validateImpactMemberPrecondition(
			member,
			memberPrecondition,
		); err != nil {
			return err
		}

		itemPrecondition, exists :=
			itemPreconditionByID[memberPrecondition.ItemID]
		if !exists {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if err := validateCoursewareReviewImpactItemPrecondition(
			plan,
			items[itemPrecondition.ItemID],
			itemPrecondition,
			actorID,
		); err != nil {
			return err
		}
	}

	return nil
}

func validatePreparedImpactSplitGroup(
	plan *models.CoursewareReviewImpactPlan,
	value *cwReviewImpactPreparedSplitGroup,
	actorID string,
	groups map[string]*models.CoursewareReviewItemGroup,
	items map[string]*models.CoursewareReviewItem,
	members map[string]*models.CoursewareReviewItemGroupMember,
) error {
	if value == nil ||
		len(value.Payload.ItemIDs) == 0 ||
		len(value.Preconditions.Members) !=
			len(value.Payload.ItemIDs) ||
		len(value.Preconditions.Items) !=
			len(value.Payload.ItemIDs) {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	if value.Payload.SourceGroupID !=
		value.Preconditions.SourceGroup.GroupID {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	if err := validateImpactGroupPrecondition(
		groups[value.Preconditions.SourceGroup.GroupID],
		value.Preconditions.SourceGroup,
	); err != nil {
		return err
	}

	memberByItem := make(
		map[string]cwReviewImpactMemberPrecondition,
		len(value.Preconditions.Members),
	)
	itemByID := make(
		map[string]cwReviewImpactItemPrecondition,
		len(value.Preconditions.Items),
	)

	for _, member := range value.Preconditions.Members {
		if member.ItemID == "" {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if _, exists := memberByItem[member.ItemID]; exists {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		memberByItem[member.ItemID] = member
	}

	for _, item := range value.Preconditions.Items {
		if item.ItemID == "" {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if _, exists := itemByID[item.ItemID]; exists {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		itemByID[item.ItemID] = item
	}

	for _, itemID := range value.Payload.ItemIDs {
		memberPrecondition, memberExists :=
			memberByItem[itemID]
		itemPrecondition, itemExists :=
			itemByID[itemID]

		if !memberExists || !itemExists {
			return ErrCoursewareReviewImpactSelectionInvalid
		}

		if err := validateImpactMemberPrecondition(
			members[memberPrecondition.MemberID],
			memberPrecondition,
		); err != nil {
			return err
		}

		if err := validateCoursewareReviewImpactItemPrecondition(
			plan,
			items[itemID],
			itemPrecondition,
			actorID,
		); err != nil {
			return err
		}
	}

	if value.Payload.PrimaryItemID != "" &&
		!containsString(
			value.Payload.ItemIDs,
			value.Payload.PrimaryItemID,
		) {
		return ErrCoursewareReviewImpactSelectionInvalid
	}

	return nil
}

func validateImpactGroupPrecondition(
	group *models.CoursewareReviewItemGroup,
	precondition cwReviewImpactGroupPrecondition,
) error {
	if group == nil ||
		group.ID != precondition.GroupID ||
		group.Status != precondition.Status ||
		group.Status != models.CWReviewItemGroupStatusActive ||
		group.Version != precondition.Version {
		return ErrCoursewareReviewImpactPlanConflict
	}

	return nil
}

func validateImpactMemberPrecondition(
	member *models.CoursewareReviewItemGroupMember,
	precondition cwReviewImpactMemberPrecondition,
) error {
	if member == nil ||
		member.ID != precondition.MemberID ||
		member.GroupID != precondition.GroupID ||
		member.ItemID != precondition.ItemID ||
		member.Status != precondition.Status ||
		member.Status !=
			models.CWReviewItemGroupMemberStatusActive ||
		member.Version != precondition.Version {
		return ErrCoursewareReviewImpactPlanConflict
	}

	return nil
}
