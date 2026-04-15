package services

// organization_service.go — 组织与教研组管理业务逻辑层
//
// v109改动：
//   - 支持多组长：组长通过 teaching_group_members.role='lead' 管理
//   - CreateTeachingGroup / UpdateTeachingGroup 移除单一 LeadUserID 参数
//   - 角色校验新增 'lead' 选项
//   - GetTeachingGroupDetail 返回 lead_user_names（所有组长名称）

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrOrgNameRequired      = errors.New("组织名称不能为空")
	ErrOrgTypeRequired      = errors.New("组织类型不能为空")
	ErrOrgTypeInvalid       = errors.New("无效的组织类型，可选值：region/school")
	ErrSchoolNeedsParent    = errors.New("学校必须指定所属区域")
	ErrOrgNameExists        = errors.New("同类型下组织名称已存在")
	ErrOrgHasChildren       = errors.New("该组织下还有子组织，无法删除")
	ErrOrgHasGroups         = errors.New("该学校下还有教研组，无法删除")
	ErrGroupNameRequired    = errors.New("教研组名称不能为空")
	ErrGroupSchoolRequired  = errors.New("教研组必须指定所属学校")
	ErrGroupSubjectRequired = errors.New("教研组学科不能为空")
	ErrGroupNameExists      = errors.New("该学校下教研组名称已存在")
	ErrMemberUserRequired   = errors.New("成员用户ID不能为空")
	ErrMemberAlreadyExists  = errors.New("该用户已是教研组成员")
	ErrOrgNotFound          = errors.New("组织不存在")
	ErrGroupNotFound        = errors.New("教研组不存在")
	ErrMemberNotFound       = errors.New("教研组成员不存在")
	ErrNoReviewPermission   = errors.New("无评审权限，需要是教研组长或骨干教师")
)

// OrganizationService 组织与教研组管理服务
type OrganizationService struct{}

var orgLog = logger.WithModule("organization")

func NewOrganizationService() *OrganizationService {
	return &OrganizationService{}
}

// ==================== 组织 CRUD ====================

func (s *OrganizationService) CreateOrganization(ctx context.Context, req *models.CreateOrganizationRequest) (*models.Organization, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, ErrOrgNameRequired
	}
	if req.Type == "" {
		return nil, ErrOrgTypeRequired
	}
	if !models.IsValidOrgType(req.Type) {
		return nil, ErrOrgTypeInvalid
	}
	if req.Type == models.OrgTypeSchool && (req.ParentID == nil || *req.ParentID == "") {
		return nil, ErrSchoolNeedsParent
	}

	exists, err := repository.CheckOrgNameExists(ctx, req.Name, req.Type, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrOrgNameExists
	}

	if req.Type == models.OrgTypeSchool && req.ParentID != nil {
		parent, err := repository.GetOrganizationByID(ctx, *req.ParentID)
		if err != nil {
			return nil, ErrOrgNotFound
		}
		if parent.Type != models.OrgTypeRegion {
			return nil, errors.New("父级组织必须是区域类型")
		}
	}

	org := &models.Organization{
		Name:        req.Name,
		Type:        req.Type,
		ParentID:    req.ParentID,
		AdminUserID: req.AdminUserID,
	}
	if err := repository.CreateOrganization(ctx, org); err != nil {
		orgLog.Error("创建组织失败", "name", req.Name, "type", req.Type, "error", err)
		return nil, err
	}
	orgLog.Info("创建组织成功", "org_id", org.ID, "name", org.Name, "type", org.Type)
	return org, nil
}

func (s *OrganizationService) ListOrganizations(ctx context.Context, orgType string, parentID string) (*models.OrganizationListResponse, error) {
	items, err := repository.ListOrganizations(ctx, orgType, parentID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.OrganizationListItem{}
	}
	return &models.OrganizationListResponse{Organizations: items, Total: len(items)}, nil
}

func (s *OrganizationService) GetOrganization(ctx context.Context, id string) (*models.Organization, error) {
	org, err := repository.GetOrganizationByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return org, nil
}

func (s *OrganizationService) UpdateOrganization(ctx context.Context, id string, req *models.UpdateOrganizationRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return ErrOrgNameRequired
	}
	existing, err := repository.GetOrganizationByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return ErrOrgNotFound
		}
		return err
	}
	nameExists, err := repository.CheckOrgNameExists(ctx, req.Name, existing.Type, id)
	if err != nil {
		return err
	}
	if nameExists {
		return ErrOrgNameExists
	}
	if err := repository.UpdateOrganization(ctx, id, req); err != nil {
		orgLog.Error("更新组织失败", "org_id", id, "error", err)
		return err
	}
	orgLog.Info("更新组织成功", "org_id", id, "name", req.Name)
	return nil
}

func (s *OrganizationService) DeleteOrganization(ctx context.Context, id string) error {
	children, err := repository.ListOrganizations(ctx, "", id)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return ErrOrgHasChildren
	}
	groups, err := repository.ListTeachingGroups(ctx, id)
	if err != nil {
		return err
	}
	if len(groups) > 0 {
		return ErrOrgHasGroups
	}
	if err := repository.DeleteOrganization(ctx, id); err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return ErrOrgNotFound
		}
		orgLog.Error("删除组织失败", "org_id", id, "error", err)
		return err
	}
	orgLog.Info("删除组织成功", "org_id", id)
	return nil
}

// ==================== 教研组 CRUD ====================

// CreateTeachingGroup 创建教研组
// v109改动：移除单一 LeadUserID 参数，组长通过成员管理设置
func (s *OrganizationService) CreateTeachingGroup(ctx context.Context, req *models.CreateTeachingGroupRequest) (*models.TeachingGroup, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Subject = strings.TrimSpace(req.Subject)
	if req.Name == "" {
		return nil, ErrGroupNameRequired
	}
	if req.SchoolID == "" {
		return nil, ErrGroupSchoolRequired
	}
	if req.Subject == "" {
		return nil, ErrGroupSubjectRequired
	}

	school, err := repository.GetOrganizationByID(ctx, req.SchoolID)
	if err != nil {
		return nil, ErrOrgNotFound
	}
	if school.Type != models.OrgTypeSchool {
		return nil, errors.New("教研组只能属于学校类型的组织")
	}

	exists, err := repository.CheckGroupNameExists(ctx, req.SchoolID, req.Name, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrGroupNameExists
	}

	tg := &models.TeachingGroup{
		Name:        req.Name,
		SchoolID:    req.SchoolID,
		Subject:     req.Subject,
		GradeRange:  req.GradeRange,
		Description: req.Description,
	}
	if err := repository.CreateTeachingGroup(ctx, tg); err != nil {
		orgLog.Error("创建教研组失败", "name", req.Name, "school_id", req.SchoolID, "error", err)
		return nil, err
	}

	orgLog.Info("创建教研组成功", "group_id", tg.ID, "name", tg.Name, "school_id", req.SchoolID)
	return tg, nil
}

func (s *OrganizationService) ListTeachingGroups(ctx context.Context, schoolID string) (*models.TeachingGroupListResponse, error) {
	items, err := repository.ListTeachingGroups(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.TeachingGroupListItem{}
	}
	return &models.TeachingGroupListResponse{Groups: items, Total: len(items)}, nil
}

// GetTeachingGroupDetail 获取教研组详情（含成员列表）
// v109改动：LeadUserNames 从成员角色表聚合
func (s *OrganizationService) GetTeachingGroupDetail(ctx context.Context, id string) (*models.TeachingGroupDetailResponse, error) {
	tg, err := repository.GetTeachingGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrGroupNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	members, err := repository.ListGroupMembers(ctx, id)
	if err != nil {
		members = []*models.GroupMemberItem{}
	}

	schoolName := ""
	school, err := repository.GetOrganizationByID(ctx, tg.SchoolID)
	if err == nil {
		schoolName = school.Name
	}

	// 兼容保留：第一个组长名称
	leadUserName := ""
	if tg.LeadUserID != nil {
		leadUser, err := repository.FindUserByID(ctx, *tg.LeadUserID)
		if err == nil {
			leadUserName = leadUser.DisplayName
		}
	}

	// v109新增：所有组长名称（逗号分隔）
	leadUserNames, _ := repository.GetGroupLeadNames(ctx, id)

	return &models.TeachingGroupDetailResponse{
		ID:            tg.ID,
		Name:          tg.Name,
		SchoolID:      tg.SchoolID,
		SchoolName:    schoolName,
		Subject:       tg.Subject,
		GradeRange:    tg.GradeRange,
		LeadUserID:    tg.LeadUserID,
		LeadUserName:  leadUserName,
		LeadUserNames: leadUserNames,
		Description:   tg.Description,
		Settings:      tg.Settings,
		Status:        tg.Status,
		Members:       members,
		CreatedAt:     tg.CreatedAt,
		UpdatedAt:     tg.UpdatedAt,
	}, nil
}

// UpdateTeachingGroup 更新教研组
// v109改动：移除单一 LeadUserID 参数
func (s *OrganizationService) UpdateTeachingGroup(ctx context.Context, id string, req *models.UpdateTeachingGroupRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Subject = strings.TrimSpace(req.Subject)
	if req.Name == "" {
		return ErrGroupNameRequired
	}
	if req.Subject == "" {
		return ErrGroupSubjectRequired
	}

	existing, err := repository.GetTeachingGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrGroupNotFound) {
			return ErrGroupNotFound
		}
		return err
	}

	nameExists, err := repository.CheckGroupNameExists(ctx, existing.SchoolID, req.Name, id)
	if err != nil {
		return err
	}
	if nameExists {
		return ErrGroupNameExists
	}

	if err := repository.UpdateTeachingGroup(ctx, id, req); err != nil {
		orgLog.Error("更新教研组失败", "group_id", id, "error", err)
		return err
	}
	orgLog.Info("更新教研组成功", "group_id", id, "name", req.Name)
	return nil
}

func (s *OrganizationService) DeleteTeachingGroup(ctx context.Context, id string) error {
	if err := repository.DeleteTeachingGroup(ctx, id); err != nil {
		if errors.Is(err, repository.ErrGroupNotFound) {
			return ErrGroupNotFound
		}
		orgLog.Error("删除教研组失败", "group_id", id, "error", err)
		return err
	}
	orgLog.Info("删除教研组成功", "group_id", id)
	return nil
}

// ==================== 教研组成员管理 ====================

// AddGroupMember 添加教研组成员
// v109改动：角色校验允许 'lead'
func (s *OrganizationService) AddGroupMember(ctx context.Context, groupID string, req *models.AddGroupMemberRequest) error {
	if req.UserID == "" {
		return ErrMemberUserRequired
	}

	_, err := repository.GetTeachingGroupByID(ctx, groupID)
	if err != nil {
		return ErrGroupNotFound
	}

	exists, err := repository.CheckMemberExists(ctx, groupID, req.UserID)
	if err != nil {
		return err
	}
	if exists {
		return ErrMemberAlreadyExists
	}

	role := req.Role
	if role == "" {
		role = models.GroupMemberRoleMember
	}
	if !models.IsValidGroupMemberRole(role) {
		return errors.New("无效的成员角色，可选值：member/backbone/lead")
	}

	member := &models.TeachingGroupMember{
		GroupID: groupID,
		UserID:  req.UserID,
		Role:    role,
	}
	if err := repository.AddGroupMember(ctx, member); err != nil {
		orgLog.Error("添加教研组成员失败", "group_id", groupID, "user_id", req.UserID, "error", err)
		return err
	}
	orgLog.Info("添加教研组成员成功", "group_id", groupID, "user_id", req.UserID, "role", role)
	return nil
}

func (s *OrganizationService) RemoveGroupMember(ctx context.Context, groupID string, userID string) error {
	if err := repository.RemoveGroupMember(ctx, groupID, userID); err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			return ErrMemberNotFound
		}
		orgLog.Error("移除教研组成员失败", "group_id", groupID, "user_id", userID, "error", err)
		return err
	}
	orgLog.Info("移除教研组成员成功", "group_id", groupID, "user_id", userID)
	return nil
}

// UpdateGroupMemberRole 更新成员角色
// v109改动：允许设置为 'lead'
func (s *OrganizationService) UpdateGroupMemberRole(ctx context.Context, groupID string, userID string, role string) error {
	if !models.IsValidGroupMemberRole(role) {
		return errors.New("无效的成员角色，可选值：member/backbone/lead")
	}
	if err := repository.UpdateGroupMemberRole(ctx, groupID, userID, role); err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			return ErrMemberNotFound
		}
		return err
	}
	orgLog.Info("更新成员角色成功", "group_id", groupID, "user_id", userID, "role", role)
	return nil
}

// ==================== 权限判断辅助 ====================

func (s *OrganizationService) GetUserTeachingGroups(ctx context.Context, userID string) ([]*models.TeachingGroupListItem, error) {
	return repository.GetUserTeachingGroups(ctx, userID)
}

func (s *OrganizationService) CheckReviewPermission(ctx context.Context, groupID string, userID string) error {
	hasPermission, err := repository.IsGroupLeadOrBackbone(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !hasPermission {
		return ErrNoReviewPermission
	}
	return nil
}
