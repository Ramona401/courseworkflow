package services

// courseware_ai_review_item_group_structural_governance.go
//
// R-06 正式问题组的跨组治理与共享业务辅助。
//
// 本文件负责：
//   1. 成员跨组移动；
//   2. 问题组合并；
//   3. 问题组拆分；
//   4. 写操作前重新加载问题组、成员和整改项；
//   5. 组名、原因和成员选择的统一规范化；
//   6. 组记录和双组结果的统一组装。
//
// 所有写动作都要求明确的版本参数，Repository仍会在事务内再次执行锁定和CAS。

import (
	"context"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// MoveCWAIReviewItemGroupMember 原子移动成员，并要求来源组、目标组和成员version全部匹配。
func (s *CoursewareAIReviewRunner) MoveCWAIReviewItemGroupMember(
	ctx context.Context,
	sessionID string,
	input *CWAIReviewItemGroupMoveMemberInput,
	actor *CoursewareActorContext,
) (*CWAIReviewItemGroupPairResult, error) {
	if input == nil ||
		input.ExpectedSourceVersion < 1 ||
		input.ExpectedTargetVersion < 1 ||
		input.ExpectedMemberVersion < 1 ||
		strings.TrimSpace(input.MemberID) == "" ||
		strings.TrimSpace(input.SourceGroupID) == "" ||
		strings.TrimSpace(input.TargetGroupID) == "" ||
		strings.TrimSpace(input.SourceGroupID) ==
			strings.TrimSpace(input.TargetGroupID) {
		return nil, ErrCWAIReviewGroupInvalid
	}

	reason, err := normalizeCWAIReviewGroupReason(
		input.Reason,
		"人工移动问题组成员",
	)
	if err != nil {
		return nil, err
	}

	session, sourceRecord, err :=
		s.loadFreshCWAIReviewGroupForWrite(
			ctx,
			sessionID,
			input.SourceGroupID,
			input.ExpectedSourceVersion,
			actor,
		)
	if err != nil {
		return nil, err
	}

	_, targetRecord, err :=
		s.loadFreshCWAIReviewGroupForWrite(
			ctx,
			sessionID,
			input.TargetGroupID,
			input.ExpectedTargetVersion,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if sourceRecord.Group.CoursewareID !=
		targetRecord.Group.CoursewareID {
		return nil,
			ErrCWAIReviewGroupSelectionInvalid
	}

	member := findCWAIReviewGroupMember(
		sourceRecord,
		input.MemberID,
	)

	if member == nil ||
		member.Status != models.CWReviewItemGroupMemberStatusActive ||
		member.Version != input.ExpectedMemberVersion {
		return nil,
			repository.ErrCoursewareReviewItemGroupConflict
	}

	source, target, _, err :=
		repository.MoveCoursewareReviewItemGroupMember(
			ctx,
			session.ID,
			sourceRecord.Group.ID,
			targetRecord.Group.ID,
			input.ExpectedSourceVersion,
			input.ExpectedTargetVersion,
			member.ID,
			input.ExpectedMemberVersion,
			actor.UserID,
			reason,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewItemGroupPairResult(
		ctx,
		source,
		target,
		actor.UserID,
	)
}

// MergeCWAIReviewItemGroups 将来源组全部成员原子移动到目标组，再冻结来源组。
func (s *CoursewareAIReviewRunner) MergeCWAIReviewItemGroups(
	ctx context.Context,
	sessionID string,
	input *CWAIReviewItemGroupMergeInput,
	actor *CoursewareActorContext,
) (*CWAIReviewItemGroupPairResult, error) {
	if input == nil ||
		input.ExpectedSourceVersion < 1 ||
		input.ExpectedTargetVersion < 1 ||
		strings.TrimSpace(input.SourceGroupID) == "" ||
		strings.TrimSpace(input.TargetGroupID) == "" ||
		strings.TrimSpace(input.SourceGroupID) ==
			strings.TrimSpace(input.TargetGroupID) {
		return nil, ErrCWAIReviewGroupInvalid
	}

	reason, err := normalizeCWAIReviewGroupReason(
		input.Reason,
		"人工合并问题组",
	)
	if err != nil {
		return nil, err
	}

	session, sourceRecord, err :=
		s.loadFreshCWAIReviewGroupForWrite(
			ctx,
			sessionID,
			input.SourceGroupID,
			input.ExpectedSourceVersion,
			actor,
		)
	if err != nil {
		return nil, err
	}

	_, targetRecord, err :=
		s.loadFreshCWAIReviewGroupForWrite(
			ctx,
			sessionID,
			input.TargetGroupID,
			input.ExpectedTargetVersion,
			actor,
		)
	if err != nil {
		return nil, err
	}

	if sourceRecord.Group.CoursewareID !=
		targetRecord.Group.CoursewareID {
		return nil,
			ErrCWAIReviewGroupSelectionInvalid
	}

	source, target, err :=
		repository.MergeCoursewareReviewItemGroups(
			ctx,
			session.ID,
			sourceRecord.Group.ID,
			targetRecord.Group.ID,
			input.ExpectedSourceVersion,
			input.ExpectedTargetVersion,
			actor.UserID,
			reason,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewItemGroupPairResult(
		ctx,
		source,
		target,
		actor.UserID,
	)
}

// SplitCWAIReviewItemGroup 把明确成员子集移动到一个新的教师问题组。
func (s *CoursewareAIReviewRunner) SplitCWAIReviewItemGroup(
	ctx context.Context,
	sessionID string,
	input *CWAIReviewItemGroupSplitInput,
	actor *CoursewareActorContext,
) (*CWAIReviewItemGroupPairResult, error) {
	if input == nil ||
		input.ExpectedSourceVersion < 1 ||
		strings.TrimSpace(input.SourceGroupID) == "" {
		return nil, ErrCWAIReviewGroupInvalid
	}

	name, err := normalizeCWAIReviewGroupName(
		input.Name,
	)
	if err != nil {
		return nil, err
	}

	itemIDs, err := normalizeCWAIReviewGroupItemIDs(
		input.ItemIDs,
	)
	if err != nil {
		return nil, err
	}

	reason, err := normalizeCWAIReviewGroupReason(
		input.Reason,
		"人工拆分问题组",
	)
	if err != nil {
		return nil, err
	}

	session, sourceRecord, err :=
		s.loadFreshCWAIReviewGroupForWrite(
			ctx,
			sessionID,
			input.SourceGroupID,
			input.ExpectedSourceVersion,
			actor,
		)
	if err != nil {
		return nil, err
	}

	activeCount := countActiveCWAIReviewGroupMembers(
		sourceRecord,
	)

	if len(itemIDs) >= activeCount {
		return nil,
			ErrCWAIReviewGroupSelectionInvalid
	}

	for _, itemID := range itemIDs {
		if !recordHasActiveCWAIReviewGroupItem(
			sourceRecord,
			itemID,
		) {
			return nil,
				ErrCWAIReviewGroupSelectionInvalid
		}
	}

	primaryItemID := strings.TrimSpace(
		input.PrimaryItemID,
	)

	if primaryItemID != "" &&
		!containsCWAIReviewGroupItemID(
			itemIDs,
			primaryItemID,
		) {
		return nil,
			ErrCWAIReviewGroupSelectionInvalid
	}

	source, target, err :=
		repository.SplitCoursewareReviewItemGroup(
			ctx,
			session.ID,
			sourceRecord.Group.ID,
			input.ExpectedSourceVersion,
			name,
			itemIDs,
			primaryItemID,
			actor.UserID,
			reason,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewItemGroupPairResult(
		ctx,
		source,
		target,
		actor.UserID,
	)
}

func (s *CoursewareAIReviewRunner) loadFreshCWAIReviewGroupForWrite(
	ctx context.Context,
	sessionID string,
	groupID string,
	expectedVersion int,
	actor *CoursewareActorContext,
) (
	*models.CoursewareAIReviewSession,
	*CWAIReviewItemGroupRecord,
	error,
) {
	session, _, _, err :=
		s.authorizeCWAIReviewGlobalDiscussionSession(
			ctx,
			sessionID,
			actor,
			false,
		)
	if err != nil {
		return nil, nil, err
	}

	group, err :=
		repository.GetCoursewareReviewItemGroupByID(
			ctx,
			groupID,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, nil, err
	}

	if group.CoursewareID != session.CoursewareID ||
		group.Status != models.CWReviewItemGroupStatusActive {
		return nil,
			nil,
			ErrCWAIReviewGroupNotActionable
	}

	if expectedVersion > 0 &&
		group.Version != expectedVersion {
		return nil,
			nil,
			repository.ErrCoursewareReviewItemGroupConflict
	}

	record, err := buildCWAIReviewItemGroupRecord(
		ctx,
		group,
		actor.UserID,
	)
	if err != nil {
		return nil, nil, err
	}

	itemIDs := activeCWAIReviewGroupItemIDs(
		record,
	)

	if len(itemIDs) > 0 {
		if _, err := loadCWAIReviewGlobalSelectedItems(
			ctx,
			session,
			itemIDs,
			actor,
		); err != nil {
			return nil, nil, err
		}
	}

	return session, record, nil
}

func buildCWAIReviewItemGroupRecord(
	ctx context.Context,
	group *models.CoursewareReviewItemGroup,
	actorID string,
) (*CWAIReviewItemGroupRecord, error) {
	if group == nil {
		return nil,
			repository.ErrCoursewareReviewItemGroupNotFound
	}

	members, err :=
		repository.ListCoursewareReviewItemGroupMembers(
			ctx,
			group.ID,
			group.SourceSessionID,
			actorID,
		)
	if err != nil {
		return nil, err
	}

	events, err :=
		repository.ListCoursewareReviewItemGroupEvents(
			ctx,
			group.ID,
			group.SourceSessionID,
			actorID,
		)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil,
			repository.ErrCoursewareReviewItemGroupNotFound
	}

	return &CWAIReviewItemGroupRecord{
		Group:   group,
		Members: members,
		Events:  events,
	}, nil
}

func buildCWAIReviewItemGroupPairResult(
	ctx context.Context,
	source *models.CoursewareReviewItemGroup,
	target *models.CoursewareReviewItemGroup,
	actorID string,
) (*CWAIReviewItemGroupPairResult, error) {
	sourceRecord, err := buildCWAIReviewItemGroupRecord(
		ctx,
		source,
		actorID,
	)
	if err != nil {
		return nil, err
	}

	targetRecord, err := buildCWAIReviewItemGroupRecord(
		ctx,
		target,
		actorID,
	)
	if err != nil {
		return nil, err
	}

	return &CWAIReviewItemGroupPairResult{
		Source: sourceRecord,
		Target: targetRecord,
	}, nil
}

func normalizeCWAIReviewGroupName(
	input string,
) (string, error) {
	name := strings.TrimSpace(input)

	if name == "" ||
		utf8.RuneCountInString(name) >
			cwAIReviewGroupNameMaxRunes {
		return "",
			ErrCWAIReviewGroupNameInvalid
	}

	return name, nil
}

func normalizeCWAIReviewGroupReason(
	input string,
	fallback string,
) (string, error) {
	reason := strings.TrimSpace(input)

	if reason == "" {
		reason = strings.TrimSpace(fallback)
	}

	if utf8.RuneCountInString(reason) >
		cwAIReviewGroupReasonMaxRunes {
		return "",
			ErrCWAIReviewGroupReasonInvalid
	}

	return reason, nil
}

func normalizeCWAIReviewGroupItemIDs(
	input []string,
) ([]string, error) {
	result := make(
		[]string,
		0,
		len(input),
	)
	seen := make(
		map[string]struct{},
		len(input),
	)

	for _, raw := range input {
		itemID := strings.TrimSpace(raw)
		if itemID == "" {
			continue
		}

		if _, exists := seen[itemID]; exists {
			continue
		}

		seen[itemID] = struct{}{}
		result = append(
			result,
			itemID,
		)
	}

	if len(result) == 0 ||
		len(result) > cwAIReviewGroupMaxMembers {
		return nil,
			ErrCWAIReviewGroupSelectionInvalid
	}

	return result, nil
}

func containsCWAIReviewGroupItemID(
	itemIDs []string,
	target string,
) bool {
	target = strings.TrimSpace(target)

	for _, itemID := range itemIDs {
		if strings.TrimSpace(itemID) == target {
			return true
		}
	}

	return false
}

func findCWAIReviewGroupMember(
	record *CWAIReviewItemGroupRecord,
	memberID string,
) *models.CoursewareReviewItemGroupMember {
	if record == nil {
		return nil
	}

	memberID = strings.TrimSpace(memberID)

	for _, member := range record.Members {
		if member != nil &&
			member.ID == memberID {
			return member
		}
	}

	return nil
}

func recordHasActiveCWAIReviewGroupItem(
	record *CWAIReviewItemGroupRecord,
	itemID string,
) bool {
	if record == nil {
		return false
	}

	itemID = strings.TrimSpace(itemID)

	for _, member := range record.Members {
		if member != nil &&
			member.Status ==
				models.CWReviewItemGroupMemberStatusActive &&
			member.ItemID == itemID {
			return true
		}
	}

	return false
}

func countActiveCWAIReviewGroupMembers(
	record *CWAIReviewItemGroupRecord,
) int {
	count := 0

	if record == nil {
		return count
	}

	for _, member := range record.Members {
		if member != nil &&
			member.Status ==
				models.CWReviewItemGroupMemberStatusActive {
			count++
		}
	}

	return count
}

func activeCWAIReviewGroupItemIDs(
	record *CWAIReviewItemGroupRecord,
) []string {
	if record == nil {
		return []string{}
	}

	itemIDs := make(
		[]string,
		0,
		len(record.Members),
	)

	for _, member := range record.Members {
		if member == nil ||
			member.Status !=
				models.CWReviewItemGroupMemberStatusActive {
			continue
		}

		itemIDs = append(
			itemIDs,
			member.ItemID,
		)
	}

	return itemIDs
}
