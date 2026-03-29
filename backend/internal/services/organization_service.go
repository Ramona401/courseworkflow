package services

// 组织与教研组管理业务逻辑层
// 负责组织（区域/学校）和教研组的CRUD、成员管理、权限判断

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

// 模块日志
var orgLog = logger.WithModule("organization")

// NewOrganizationService 创建组织管理服务实例
func NewOrganizationService() *OrganizationService {
	return &OrganizationService{}
}

// ==================== 组织 CRUD ====================

// CreateOrganization 创建组织
func (s *OrganizationService) CreateOrganization(ctx context.Context, req *models.CreateOrganizationRequest) (*models.Organization, error) {
	// 1. 参数校验
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
	// 学校必须有父级（区域）
	if req.Type == models.OrgTypeSchool && (req.ParentID == nil || *req.ParentID == "") {
		return nil, ErrSchoolNeedsParent
	}

	// 2. 检查名称唯一性
	exists, err := repository.CheckOrgNameExists(ctx, req.Name, req.Type, "")
	if err != nil {
		orgLog.Error("检查组织名称失败", "name", req.Name, "error", err)
		return nil, err
	}
	if exists {
		return nil, ErrOrgNameExists
	}

	// 3. 如果是学校，验证父级区域存在
	if req.Type == models.OrgTypeSchool && req.ParentID != nil {
		parent, err := repository.GetOrganizationByID(ctx, *req.ParentID)
		if err != nil {
			return nil, ErrOrgNotFound
		}
		if parent.Type != models.OrgTypeRegion {
			return nil, errors.New("父级组织必须是区域类型")
		}
	}

	// 4. 创建
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

// ListOrganizations 获取组织列表
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

// GetOrganization 获取组织详情
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

// UpdateOrganization 更新组织
func (s *OrganizationService) UpdateOrganization(ctx context.Context, id string, req *models.UpdateOrganizationRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return ErrOrgNameRequired
	}

	// 获取现有组织（确认存在+获取type）
	existing, err := repository.GetOrganizationByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return ErrOrgNotFound
		}
		return err
	}

	// 检查名称唯一性（排除自身）
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

// DeleteOrganization 删除组织
func (s *OrganizationService) DeleteOrganization(ctx context.Context, id string) error {
	// 检查是否有子组织
	children, err := repository.ListOrganizations(ctx, "", id)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return ErrOrgHasChildren
	}

	// 检查是否有教研组
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

	// 验证学校存在
	school, err := repository.GetOrganizationByID(ctx, req.SchoolID)
	if err != nil {
		return nil, ErrOrgNotFound
	}
	if school.Type != models.OrgTypeSchool {
		return nil, errors.New("教研组只能属于学校类型的组织")
	}

	// 检查名称唯一性
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
		LeadUserID:  req.LeadUserID,
		Description: req.Description,
	}
	if err := repository.CreateTeachingGroup(ctx, tg); err != nil {
		orgLog.Error("创建教研组失败", "name", req.Name, "school_id", req.SchoolID, "error", err)
		return nil, err
	}

	// 如果指定了组长，自动添加为成员
	if req.LeadUserID != nil && *req.LeadUserID != "" {
		member := &models.TeachingGroupMember{
			GroupID: tg.ID,
			UserID:  *req.LeadUserID,
			Role:    models.GroupMemberRoleBackbone,
		}
		_ = repository.AddGroupMember(ctx, member) // 忽略错误（可能已存在）
	}

	orgLog.Info("创建教研组成功", "group_id", tg.ID, "name", tg.Name, "school_id", req.SchoolID)
	return tg, nil
}

// ListTeachingGroups 获取教研组列表
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
func (s *OrganizationService) GetTeachingGroupDetail(ctx context.Context, id string) (*models.TeachingGroupDetailResponse, error) {
	tg, err := repository.GetTeachingGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrGroupNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}

	// 获取成员列表
	members, err := repository.ListGroupMembers(ctx, id)
	if err != nil {
		members = []*models.GroupMemberItem{}
	}

	// 获取学校名称
	schoolName := ""
	school, err := repository.GetOrganizationByID(ctx, tg.SchoolID)
	if err == nil {
		schoolName = school.Name
	}

	// 获取组长名称
	leadUserName := ""
	if tg.LeadUserID != nil {
		leadUser, err := repository.FindUserByID(ctx, *tg.LeadUserID)
		if err == nil {
			leadUserName = leadUser.DisplayName
		}
	}

	return &models.TeachingGroupDetailResponse{
		ID:           tg.ID,
		Name:         tg.Name,
		SchoolID:     tg.SchoolID,
		SchoolName:   schoolName,
		Subject:      tg.Subject,
		GradeRange:   tg.GradeRange,
		LeadUserID:   tg.LeadUserID,
		LeadUserName: leadUserName,
		Description:  tg.Description,
		Settings:     tg.Settings,
		Status:       tg.Status,
		Members:      members,
		CreatedAt:    tg.CreatedAt,
		UpdatedAt:    tg.UpdatedAt,
	}, nil
}

// UpdateTeachingGroup 更新教研组
func (s *OrganizationService) UpdateTeachingGroup(ctx context.Context, id string, req *models.UpdateTeachingGroupRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Subject = strings.TrimSpace(req.Subject)
	if req.Name == "" {
		return ErrGroupNameRequired
	}
	if req.Subject == "" {
		return ErrGroupSubjectRequired
	}

	// 获取现有教研组
	existing, err := repository.GetTeachingGroupByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrGroupNotFound) {
			return ErrGroupNotFound
		}
		return err
	}

	// 检查名称唯一性（排除自身）
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

// DeleteTeachingGroup 删除教研组
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
func (s *OrganizationService) AddGroupMember(ctx context.Context, groupID string, req *models.AddGroupMemberRequest) error {
	if req.UserID == "" {
		return ErrMemberUserRequired
	}

	// 验证教研组存在
	_, err := repository.GetTeachingGroupByID(ctx, groupID)
	if err != nil {
		return ErrGroupNotFound
	}

	// 检查是否已是成员
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
		return errors.New("无效的成员角色，可选值：member/backbone")
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

// RemoveGroupMember 移除教研组成员
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
func (s *OrganizationService) UpdateGroupMemberRole(ctx context.Context, groupID string, userID string, role string) error {
	if !models.IsValidGroupMemberRole(role) {
		return errors.New("无效的成员角色，可选值：member/backbone")
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

// GetUserTeachingGroups 获取用户所属教研组列表
func (s *OrganizationService) GetUserTeachingGroups(ctx context.Context, userID string) ([]*models.TeachingGroupListItem, error) {
	return repository.GetUserTeachingGroups(ctx, userID)
}

// CheckReviewPermission 检查用户是否有评审权限（组长或骨干）
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
