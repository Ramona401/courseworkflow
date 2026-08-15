package services

// courseware_ai_review_item_group_governance.go
//
// R-06 正式问题组基础治理公开业务服务。
//
// 本文件负责：
//   1. 问题组列表读取；
//   2. 创建问题组；
//   3. 重命名问题组；
//   4. 设置或清空主问题；
//   5. 加入或移除成员。
//
// 跨组移动、合并、拆分以及共享校验辅助位于：
// courseware_ai_review_item_group_structural_governance.go。
//
// 安全边界：
//   - 复用全局讨论既有会话授权，不扩大管理员权限；
//   - 分组不会修改整改项严重度、修改要求、页面或审核决定；
//   - 所有写操作都要求显式version，并在Repository事务内再次锁定。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	cwAIReviewGroupNameMaxRunes   = 200
	cwAIReviewGroupReasonMaxRunes = 500
	cwAIReviewGroupMaxMembers     = 100
)

var (
	ErrCWAIReviewGroupInvalid = errors.New(
		"课件审核问题组参数无效",
	)

	ErrCWAIReviewGroupNameInvalid = errors.New(
		"问题组名称不能为空或超过200字",
	)

	ErrCWAIReviewGroupReasonInvalid = errors.New(
		"问题组操作原因不能超过500字",
	)

	ErrCWAIReviewGroupSelectionInvalid = errors.New(
		"问题组成员选择无效",
	)

	ErrCWAIReviewGroupNotActionable = errors.New(
		"当前课件审核问题组不能继续治理",
	)
)

// CWAIReviewItemGroupRecord 是浏览器读取一条问题组时需要的完整治理记录。
type CWAIReviewItemGroupRecord struct {
	Group *models.CoursewareReviewItemGroup

	Members []*models.CoursewareReviewItemGroupMember
	Events  []*models.CoursewareReviewItemGroupEvent
}

// CWAIReviewItemGroupPairResult 用于移动、合并和拆分后的双组刷新。
type CWAIReviewItemGroupPairResult struct {
	Source *CWAIReviewItemGroupRecord
	Target *CWAIReviewItemGroupRecord
}

// CWAIReviewItemGroupCreateInput 创建问题组时必须明确初始成员。
type CWAIReviewItemGroupCreateInput struct {
	Name          string
	ItemIDs       []string
	PrimaryItemID string
}

// CWAIReviewItemGroupRenameInput 使用组version执行乐观并发重命名。
type CWAIReviewItemGroupRenameInput struct {
	ExpectedVersion int
	Name            string
}

// CWAIReviewItemGroupPrimaryInput 设置主问题；PrimaryItemID为空表示明确清空。
type CWAIReviewItemGroupPrimaryInput struct {
	ExpectedVersion int
	PrimaryItemID   string
}

// CWAIReviewItemGroupAddMemberInput 将未分组或历史已移除成员加入目标组。
type CWAIReviewItemGroupAddMemberInput struct {
	ExpectedGroupVersion int
	ItemID               string
}

// CWAIReviewItemGroupRemoveMemberInput 移除成员但保留稳定成员身份。
type CWAIReviewItemGroupRemoveMemberInput struct {
	ExpectedGroupVersion  int
	ExpectedMemberVersion int
	MemberID              string
	Reason                string
}

// CWAIReviewItemGroupMoveMemberInput 原子移动成员，来源组和目标组都必须提交当前version。
type CWAIReviewItemGroupMoveMemberInput struct {
	SourceGroupID string
	TargetGroupID string

	ExpectedSourceVersion int
	ExpectedTargetVersion int

	MemberID              string
	ExpectedMemberVersion int
	Reason                string
}

// CWAIReviewItemGroupMergeInput 将SourceGroup全部有效成员归入TargetGroup。
type CWAIReviewItemGroupMergeInput struct {
	SourceGroupID string
	TargetGroupID string

	ExpectedSourceVersion int
	ExpectedTargetVersion int
	Reason                string
}

// CWAIReviewItemGroupSplitInput 把来源组的明确成员子集移入一个新组。
type CWAIReviewItemGroupSplitInput struct {
	SourceGroupID         string
	ExpectedSourceVersion int

	Name          string
	ItemIDs       []string
	PrimaryItemID string
	Reason        string
}

// ListCWAIReviewItemGroups 读取当前会话创建者的全部问题组及追加式历史。
func (s *CoursewareAIReviewRunner) ListCWAIReviewItemGroups(
	ctx context.Context,
	sessionID string,
	actor *CoursewareActorContext,
) ([]*CWAIReviewItemGroupRecord, error) {
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

	groups, err :=
		repository.ListCoursewareReviewItemGroupsBySession(
			ctx,
			session.ID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	result := make(
		[]*CWAIReviewItemGroupRecord,
		0,
		len(groups),
	)

	for _, group := range groups {
		record, buildErr :=
			buildCWAIReviewItemGroupRecord(
				ctx,
				group,
				actor.UserID,
			)
		if buildErr != nil {
			return nil, buildErr
		}

		result = append(
			result,
			record,
		)
	}

	return result, nil
}

// CreateCWAIReviewItemGroup 创建教师可管理的问题组。
func (s *CoursewareAIReviewRunner) CreateCWAIReviewItemGroup(
	ctx context.Context,
	sessionID string,
	input *CWAIReviewItemGroupCreateInput,
	actor *CoursewareActorContext,
) (*CWAIReviewItemGroupRecord, error) {
	if input == nil {
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

	if _, err := loadCWAIReviewGlobalSelectedItems(
		ctx,
		session,
		itemIDs,
		actor,
	); err != nil {
		return nil, err
	}

	group, err :=
		repository.CreateCoursewareReviewItemGroup(
			ctx,
			session.CoursewareID,
			session.ID,
			name,
			itemIDs,
			primaryItemID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewItemGroupRecord(
		ctx,
		group,
		actor.UserID,
	)
}

// RenameCWAIReviewItemGroup 明确重命名一个仍可治理的问题组。
func (s *CoursewareAIReviewRunner) RenameCWAIReviewItemGroup(
	ctx context.Context,
	sessionID string,
	groupID string,
	input *CWAIReviewItemGroupRenameInput,
	actor *CoursewareActorContext,
) (*CWAIReviewItemGroupRecord, error) {
	if input == nil ||
		input.ExpectedVersion < 1 {
		return nil, ErrCWAIReviewGroupInvalid
	}

	name, err := normalizeCWAIReviewGroupName(
		input.Name,
	)
	if err != nil {
		return nil, err
	}

	session, record, err :=
		s.loadFreshCWAIReviewGroupForWrite(
			ctx,
			sessionID,
			groupID,
			input.ExpectedVersion,
			actor,
		)
	if err != nil {
		return nil, err
	}

	group, err :=
		repository.RenameCoursewareReviewItemGroup(
			ctx,
			session.ID,
			record.Group.ID,
			input.ExpectedVersion,
			name,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewItemGroupRecord(
		ctx,
		group,
		actor.UserID,
	)
}

// SetCWAIReviewItemGroupPrimary 设置或清空主问题。
func (s *CoursewareAIReviewRunner) SetCWAIReviewItemGroupPrimary(
	ctx context.Context,
	sessionID string,
	groupID string,
	input *CWAIReviewItemGroupPrimaryInput,
	actor *CoursewareActorContext,
) (*CWAIReviewItemGroupRecord, error) {
	if input == nil ||
		input.ExpectedVersion < 1 {
		return nil, ErrCWAIReviewGroupInvalid
	}

	session, record, err :=
		s.loadFreshCWAIReviewGroupForWrite(
			ctx,
			sessionID,
			groupID,
			input.ExpectedVersion,
			actor,
		)
	if err != nil {
		return nil, err
	}

	primaryItemID := strings.TrimSpace(
		input.PrimaryItemID,
	)

	var primary *string

	if primaryItemID != "" {
		if !recordHasActiveCWAIReviewGroupItem(
			record,
			primaryItemID,
		) {
			return nil,
				ErrCWAIReviewGroupSelectionInvalid
		}

		primary = &primaryItemID
	}

	group, err :=
		repository.SetCoursewareReviewItemGroupPrimary(
			ctx,
			session.ID,
			record.Group.ID,
			input.ExpectedVersion,
			primary,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewItemGroupRecord(
		ctx,
		group,
		actor.UserID,
	)
}

// AddCWAIReviewItemGroupMember 明确把一条可治理整改项加入问题组。
func (s *CoursewareAIReviewRunner) AddCWAIReviewItemGroupMember(
	ctx context.Context,
	sessionID string,
	groupID string,
	input *CWAIReviewItemGroupAddMemberInput,
	actor *CoursewareActorContext,
) (*CWAIReviewItemGroupRecord, error) {
	if input == nil ||
		input.ExpectedGroupVersion < 1 ||
		strings.TrimSpace(input.ItemID) == "" {
		return nil, ErrCWAIReviewGroupInvalid
	}

	session, record, err :=
		s.loadFreshCWAIReviewGroupForWrite(
			ctx,
			sessionID,
			groupID,
			input.ExpectedGroupVersion,
			actor,
		)
	if err != nil {
		return nil, err
	}

	itemID := strings.TrimSpace(
		input.ItemID,
	)

	if _, err := loadCWAIReviewGlobalSelectedItems(
		ctx,
		session,
		[]string{
			itemID,
		},
		actor,
	); err != nil {
		return nil, err
	}

	group, _, err :=
		repository.AddCoursewareReviewItemGroupMember(
			ctx,
			session.ID,
			record.Group.ID,
			input.ExpectedGroupVersion,
			itemID,
			actor.UserID,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewItemGroupRecord(
		ctx,
		group,
		actor.UserID,
	)
}

// RemoveCWAIReviewItemGroupMember 明确移除一个成员；若它是主问题则事务内先清空主问题。
func (s *CoursewareAIReviewRunner) RemoveCWAIReviewItemGroupMember(
	ctx context.Context,
	sessionID string,
	groupID string,
	input *CWAIReviewItemGroupRemoveMemberInput,
	actor *CoursewareActorContext,
) (*CWAIReviewItemGroupRecord, error) {
	if input == nil ||
		input.ExpectedGroupVersion < 1 ||
		input.ExpectedMemberVersion < 1 ||
		strings.TrimSpace(
			input.MemberID,
		) == "" {
		return nil, ErrCWAIReviewGroupInvalid
	}

	reason, err := normalizeCWAIReviewGroupReason(
		input.Reason,
		"人工移除问题组成员",
	)
	if err != nil {
		return nil, err
	}

	session, record, err :=
		s.loadFreshCWAIReviewGroupForWrite(
			ctx,
			sessionID,
			groupID,
			input.ExpectedGroupVersion,
			actor,
		)
	if err != nil {
		return nil, err
	}

	member := findCWAIReviewGroupMember(
		record,
		input.MemberID,
	)

	if member == nil ||
		member.Status != models.CWReviewItemGroupMemberStatusActive ||
		member.Version != input.ExpectedMemberVersion {
		return nil,
			repository.ErrCoursewareReviewItemGroupConflict
	}

	group, _, err :=
		repository.RemoveCoursewareReviewItemGroupMember(
			ctx,
			session.ID,
			record.Group.ID,
			input.ExpectedGroupVersion,
			member.ID,
			input.ExpectedMemberVersion,
			actor.UserID,
			reason,
		)
	if err != nil {
		return nil, err
	}

	return buildCWAIReviewItemGroupRecord(
		ctx,
		group,
		actor.UserID,
	)
}
