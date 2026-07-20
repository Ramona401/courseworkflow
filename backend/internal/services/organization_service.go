package services

// organization_service.go
//
// 职责：
//   - 区域、学校CRUD；
//   - 组织数据范围过滤；
//   - 教研组CRUD；
//   - 教研组成员管理。
//
// 上下文7新增创建规则：
//   - 创建学校必须主动选择k12/vocational/adult；
//   - 不允许空值、mixed、common或非法值；
//   - 创建区域忽略客户端教育域并强制写mixed；
//   - 创建响应返回数据库最终写入的education_domain。
//
// 普通更新接口不包含education_domain，本上下文不关闭现有独立换域接口。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrOrgNameRequired               = errors.New("组织名称不能为空")
	ErrOrgTypeRequired               = errors.New("组织类型不能为空")
	ErrOrgTypeInvalid                = errors.New("无效的组织类型，可选值：region/school")
	ErrSchoolNeedsParent             = errors.New("学校必须指定所属区域")
	ErrSchoolEducationDomainRequired = errors.New("新建学校必须选择教育类型")
	ErrSchoolEducationDomainInvalid  = errors.New("学校教育类型必须为k12、vocational或adult")
	ErrOrgNameExists                 = errors.New("同类型下组织名称已存在")
	ErrOrgHasChildren                = errors.New("该组织下还有子组织，无法删除")
	ErrOrgHasGroups                  = errors.New("该学校下还有教研组，无法删除")
	ErrGroupNameRequired             = errors.New("教研组名称不能为空")
	ErrGroupSchoolRequired           = errors.New("教研组必须指定所属学校")
	ErrGroupSubjectRequired          = errors.New("教研组学科不能为空")
	ErrGroupNameExists               = errors.New("该学校下教研组名称已存在")
	ErrMemberUserRequired            = errors.New("成员用户ID不能为空")
	ErrMemberAlreadyExists           = errors.New("该用户已是教研组成员")
	ErrOrgNotFound                   = errors.New("组织不存在")
	ErrGroupNotFound                 = errors.New("教研组不存在")
	ErrMemberNotFound                = errors.New("教研组成员不存在")
	ErrNoReviewPermission            = errors.New("无评审权限，需要是教研组长或骨干教师")
)

// OrganizationService 组织与教研组业务服务。
type OrganizationService struct{}

var orgLog = logger.WithModule("organization")

// NewOrganizationService 创建组织服务。
func NewOrganizationService() *OrganizationService {
	return &OrganizationService{}
}

// normalizeCreateOrganizationEducationDomain 解析创建组织时的最终教育域。
//
// 区域始终返回mixed；学校必须显式提交具体教学域。
func normalizeCreateOrganizationEducationDomain(
	orgType string,
	requestedDomain string,
) (string, error) {
	if orgType == models.OrgTypeRegion {
		return models.EducationDomainMixed, nil
	}

	domain := strings.ToLower(
		strings.TrimSpace(requestedDomain),
	)
	if domain == "" {
		return "", ErrSchoolEducationDomainRequired
	}
	if !models.IsTeachingEducationDomain(domain) {
		return "", ErrSchoolEducationDomainInvalid
	}

	return domain, nil
}

// ==================== 组织 CRUD ====================

// CreateOrganization 创建区域或学校。
func (s *OrganizationService) CreateOrganization(
	ctx context.Context,
	req *models.CreateOrganizationRequest,
) (*models.Organization, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))

	if req.Name == "" {
		return nil, ErrOrgNameRequired
	}
	if req.Type == "" {
		return nil, ErrOrgTypeRequired
	}
	if !models.IsValidOrgType(req.Type) {
		return nil, ErrOrgTypeInvalid
	}

	if req.Type == models.OrgTypeSchool &&
		(req.ParentID == nil ||
			strings.TrimSpace(*req.ParentID) == "") {
		return nil, ErrSchoolNeedsParent
	}

	educationDomain, err :=
		normalizeCreateOrganizationEducationDomain(
			req.Type,
			req.EducationDomain,
		)
	if err != nil {
		return nil, err
	}

	exists, err := repository.CheckOrgNameExists(
		ctx,
		req.Name,
		req.Type,
		"",
	)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrOrgNameExists
	}

	if req.Type == models.OrgTypeSchool &&
		req.ParentID != nil {
		parent, getErr := repository.GetOrganizationByID(
			ctx,
			*req.ParentID,
		)
		if getErr != nil {
			return nil, ErrOrgNotFound
		}
		if parent.Type != models.OrgTypeRegion {
			return nil, errors.New(
				"父级组织必须是区域类型",
			)
		}
	}

	org := &models.Organization{
		Name:            req.Name,
		Type:            req.Type,
		EducationDomain: educationDomain,
		ParentID:        req.ParentID,
		AdminUserID:     req.AdminUserID,
	}

	if err := repository.CreateOrganization(
		ctx,
		org,
	); err != nil {
		orgLog.Error(
			"创建组织失败",
			"name",
			req.Name,
			"type",
			req.Type,
			"education_domain",
			educationDomain,
			"error",
			err,
		)
		return nil, err
	}

	orgLog.Info(
		"创建组织成功",
		"org_id",
		org.ID,
		"name",
		org.Name,
		"type",
		org.Type,
		"education_domain",
		org.EducationDomain,
	)

	return org, nil
}

// ListOrganizations 按数据范围查询组织。
func (s *OrganizationService) ListOrganizations(
	ctx context.Context,
	orgType string,
	parentID string,
	scope DataScope,
) (*models.OrganizationListResponse, error) {
	items, err := repository.ListOrganizations(
		ctx,
		orgType,
		parentID,
	)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.OrganizationListItem{}
	}

	if scope.IsAdmin {
		return &models.OrganizationListResponse{
			Organizations: items,
			Total:         len(items),
		}, nil
	}

	visible := make(map[string]struct{})
	for _, id := range scope.OrgIDs {
		if id != "" {
			visible[id] = struct{}{}
		}
	}

	if scope.Role == models.RoleSeniorOperator &&
		!scope.Blocked {
		for _, schoolID := range scope.OrgIDs {
			if schoolID == "" {
				continue
			}

			school, getErr :=
				repository.GetOrganizationByID(
					ctx,
					schoolID,
				)
			if getErr == nil &&
				school != nil &&
				school.ParentID != nil &&
				*school.ParentID != "" {
				visible[*school.ParentID] = struct{}{}
			}
		}
	}

	filtered := make(
		[]*models.OrganizationListItem,
		0,
		len(items),
	)
	for _, item := range items {
		if _, allowed := visible[item.ID]; allowed {
			filtered = append(filtered, item)
		}
	}

	return &models.OrganizationListResponse{
		Organizations: filtered,
		Total:         len(filtered),
	}, nil
}

// GetOrganization 查询单个组织。
func (s *OrganizationService) GetOrganization(
	ctx context.Context,
	id string,
) (*models.Organization, error) {
	org, err := repository.GetOrganizationByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}

	return org, nil
}

// UpdateOrganization 更新组织普通字段。
func (s *OrganizationService) UpdateOrganization(
	ctx context.Context,
	id string,
	req *models.UpdateOrganizationRequest,
) error {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return ErrOrgNameRequired
	}

	existing, err := repository.GetOrganizationByID(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return ErrOrgNotFound
		}
		return err
	}

	nameExists, err := repository.CheckOrgNameExists(
		ctx,
		req.Name,
		existing.Type,
		id,
	)
	if err != nil {
		return err
	}
	if nameExists {
		return ErrOrgNameExists
	}

	if strings.TrimSpace(req.Settings) == "" {
		req.Settings = existing.Settings
	}
	if strings.TrimSpace(req.Status) == "" {
		req.Status = existing.Status
	}

	if err := repository.UpdateOrganization(
		ctx,
		id,
		req,
	); err != nil {
		orgLog.Error(
			"更新组织失败",
			"org_id",
			id,
			"error",
			err,
		)
		return err
	}

	if req.ClearLogo {
		if err := repository.UpdateOrganizationLogo(
			ctx,
			id,
			"",
		); err != nil {
			orgLog.Error(
				"清除组织Logo失败",
				"org_id",
				id,
				"error",
				err,
			)
			return err
		}

		orgLog.Info(
			"清除组织Logo成功",
			"org_id",
			id,
		)
	}

	orgLog.Info(
		"更新组织成功",
		"org_id",
		id,
		"name",
		req.Name,
	)

	return nil
}

// DeleteOrganization 删除组织。
func (s *OrganizationService) DeleteOrganization(
	ctx context.Context,
	id string,
) error {
	children, err := repository.ListOrganizations(
		ctx,
		"",
		id,
	)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return ErrOrgHasChildren
	}

	groups, err := repository.ListTeachingGroups(
		ctx,
		id,
	)
	if err != nil {
		return err
	}
	if len(groups) > 0 {
		return ErrOrgHasGroups
	}

	if err := repository.DeleteOrganization(
		ctx,
		id,
	); err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return ErrOrgNotFound
		}

		orgLog.Error(
			"删除组织失败",
			"org_id",
			id,
			"error",
			err,
		)
		return err
	}

	orgLog.Info(
		"删除组织成功",
		"org_id",
		id,
	)

	return nil
}

// ==================== 教研组 CRUD ====================

// CreateTeachingGroup 创建教研组。
func (s *OrganizationService) CreateTeachingGroup(
	ctx context.Context,
	req *models.CreateTeachingGroupRequest,
) (*models.TeachingGroup, error) {
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

	school, err := repository.GetOrganizationByID(
		ctx,
		req.SchoolID,
	)
	if err != nil {
		return nil, ErrOrgNotFound
	}
	if school.Type != models.OrgTypeSchool {
		return nil, errors.New(
			"教研组只能属于学校类型的组织",
		)
	}

	exists, err := repository.CheckGroupNameExists(
		ctx,
		req.SchoolID,
		req.Name,
		"",
	)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrGroupNameExists
	}

	group := &models.TeachingGroup{
		Name:        req.Name,
		SchoolID:    req.SchoolID,
		Subject:     req.Subject,
		GradeRange:  req.GradeRange,
		Description: req.Description,
	}

	if err := repository.CreateTeachingGroup(
		ctx,
		group,
	); err != nil {
		orgLog.Error(
			"创建教研组失败",
			"name",
			req.Name,
			"school_id",
			req.SchoolID,
			"error",
			err,
		)
		return nil, err
	}

	orgLog.Info(
		"创建教研组成功",
		"group_id",
		group.ID,
		"name",
		group.Name,
		"school_id",
		req.SchoolID,
	)

	return group, nil
}

// ListTeachingGroups 按数据范围查询教研组。
func (s *OrganizationService) ListTeachingGroups(
	ctx context.Context,
	schoolID string,
	scope DataScope,
) (*models.TeachingGroupListResponse, error) {
	if scope.IsAdmin {
		items, err := repository.ListTeachingGroups(
			ctx,
			schoolID,
		)
		if err != nil {
			return nil, err
		}
		if items == nil {
			items = []*models.TeachingGroupListItem{}
		}

		return &models.TeachingGroupListResponse{
			Groups: items,
			Total:  len(items),
		}, nil
	}

	empty := &models.TeachingGroupListResponse{
		Groups: []*models.TeachingGroupListItem{},
		Total:  0,
	}

	if scope.Blocked || schoolID == "" {
		return empty, nil
	}

	allowed := false
	for _, visibleSchoolID := range scope.SchoolIDs {
		if visibleSchoolID == schoolID {
			allowed = true
			break
		}
	}

	if !allowed {
		orgLog.Warn(
			"教研组列表越权拦截：请求学校不在管辖范围",
			"role",
			scope.Role,
			"requested_school",
			schoolID,
		)
		return empty, nil
	}

	items, err := repository.ListTeachingGroups(
		ctx,
		schoolID,
	)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.TeachingGroupListItem{}
	}

	return &models.TeachingGroupListResponse{
		Groups: items,
		Total:  len(items),
	}, nil
}

// GetTeachingGroupDetail 查询教研组详情。
func (s *OrganizationService) GetTeachingGroupDetail(
	ctx context.Context,
	id string,
) (*models.TeachingGroupDetailResponse, error) {
	group, err := repository.GetTeachingGroupByID(
		ctx,
		id,
	)
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
	school, getSchoolErr :=
		repository.GetOrganizationByID(
			ctx,
			group.SchoolID,
		)
	if getSchoolErr == nil {
		schoolName = school.Name
	}

	leadUserName := ""
	if group.LeadUserID != nil {
		leadUser, getLeadErr :=
			repository.FindUserByID(
				ctx,
				*group.LeadUserID,
			)
		if getLeadErr == nil {
			leadUserName = leadUser.DisplayName
		}
	}

	leadUserNames, _ :=
		repository.GetGroupLeadNames(ctx, id)

	return &models.TeachingGroupDetailResponse{
		ID:            group.ID,
		Name:          group.Name,
		SchoolID:      group.SchoolID,
		SchoolName:    schoolName,
		Subject:       group.Subject,
		GradeRange:    group.GradeRange,
		LeadUserID:    group.LeadUserID,
		LeadUserName:  leadUserName,
		LeadUserNames: leadUserNames,
		Description:   group.Description,
		Settings:      group.Settings,
		Status:        group.Status,
		Members:       members,
		CreatedAt:     group.CreatedAt,
		UpdatedAt:     group.UpdatedAt,
	}, nil
}

// UpdateTeachingGroup 更新教研组。
func (s *OrganizationService) UpdateTeachingGroup(
	ctx context.Context,
	id string,
	req *models.UpdateTeachingGroupRequest,
) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Subject = strings.TrimSpace(req.Subject)

	if req.Name == "" {
		return ErrGroupNameRequired
	}
	if req.Subject == "" {
		return ErrGroupSubjectRequired
	}

	existing, err := repository.GetTeachingGroupByID(
		ctx,
		id,
	)
	if err != nil {
		if errors.Is(err, repository.ErrGroupNotFound) {
			return ErrGroupNotFound
		}
		return err
	}

	nameExists, err :=
		repository.CheckGroupNameExists(
			ctx,
			existing.SchoolID,
			req.Name,
			id,
		)
	if err != nil {
		return err
	}
	if nameExists {
		return ErrGroupNameExists
	}

	if strings.TrimSpace(req.Settings) == "" {
		req.Settings = existing.Settings
	}
	if strings.TrimSpace(req.Status) == "" {
		req.Status = existing.Status
	}

	if err := repository.UpdateTeachingGroup(
		ctx,
		id,
		req,
	); err != nil {
		orgLog.Error(
			"更新教研组失败",
			"group_id",
			id,
			"error",
			err,
		)
		return err
	}

	orgLog.Info(
		"更新教研组成功",
		"group_id",
		id,
		"name",
		req.Name,
	)

	return nil
}

// DeleteTeachingGroup 删除教研组。
func (s *OrganizationService) DeleteTeachingGroup(
	ctx context.Context,
	id string,
) error {
	if err := repository.DeleteTeachingGroup(
		ctx,
		id,
	); err != nil {
		if errors.Is(err, repository.ErrGroupNotFound) {
			return ErrGroupNotFound
		}

		orgLog.Error(
			"删除教研组失败",
			"group_id",
			id,
			"error",
			err,
		)
		return err
	}

	orgLog.Info(
		"删除教研组成功",
		"group_id",
		id,
	)

	return nil
}

// ==================== 教研组成员管理 ====================

// AddGroupMember 添加教研组成员。
//
// 成功后best-effort写入school_members，保持“加入本校教研组即本校成员”。
func (s *OrganizationService) AddGroupMember(
	ctx context.Context,
	groupID string,
	req *models.AddGroupMemberRequest,
) error {
	if req.UserID == "" {
		return ErrMemberUserRequired
	}

	group, err := repository.GetTeachingGroupByID(
		ctx,
		groupID,
	)
	if err != nil {
		return ErrGroupNotFound
	}

	exists, err := repository.CheckMemberExists(
		ctx,
		groupID,
		req.UserID,
	)
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
		return errors.New(
			"无效的成员角色，可选值：member/backbone/lead",
		)
	}

	member := &models.TeachingGroupMember{
		GroupID: groupID,
		UserID:  req.UserID,
		Role:    role,
	}

	if err := repository.AddGroupMember(
		ctx,
		member,
	); err != nil {
		orgLog.Error(
			"添加教研组成员失败",
			"group_id",
			groupID,
			"user_id",
			req.UserID,
			"error",
			err,
		)
		return err
	}

	if addErr := repository.AddSchoolMember(
		ctx,
		group.SchoolID,
		req.UserID,
		"group_member",
	); addErr != nil {
		orgLog.Warn(
			"添加教研组成员后写入学校成员失败",
			"user_id",
			req.UserID,
			"school_id",
			group.SchoolID,
			"error",
			addErr,
		)
	}

	orgLog.Info(
		"添加教研组成员成功",
		"group_id",
		groupID,
		"user_id",
		req.UserID,
		"role",
		role,
		"school_id",
		group.SchoolID,
	)

	return nil
}

// RemoveGroupMember 移除教研组成员。
//
// 本操作不自动删除school_members校籍。
func (s *OrganizationService) RemoveGroupMember(
	ctx context.Context,
	groupID string,
	userID string,
) error {
	if err := repository.RemoveGroupMember(
		ctx,
		groupID,
		userID,
	); err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			return ErrMemberNotFound
		}

		orgLog.Error(
			"移除教研组成员失败",
			"group_id",
			groupID,
			"user_id",
			userID,
			"error",
			err,
		)
		return err
	}

	orgLog.Info(
		"移除教研组成员成功",
		"group_id",
		groupID,
		"user_id",
		userID,
	)

	return nil
}

// UpdateGroupMemberRole 更新教研组成员角色。
func (s *OrganizationService) UpdateGroupMemberRole(
	ctx context.Context,
	groupID string,
	userID string,
	role string,
) error {
	if !models.IsValidGroupMemberRole(role) {
		return errors.New(
			"无效的成员角色，可选值：member/backbone/lead",
		)
	}

	if err := repository.UpdateGroupMemberRole(
		ctx,
		groupID,
		userID,
		role,
	); err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			return ErrMemberNotFound
		}
		return err
	}

	orgLog.Info(
		"更新成员角色成功",
		"group_id",
		groupID,
		"user_id",
		userID,
		"role",
		role,
	)

	return nil
}

// ==================== 权限辅助 ====================

// GetUserTeachingGroups 查询用户所属教研组。
func (s *OrganizationService) GetUserTeachingGroups(
	ctx context.Context,
	userID string,
) ([]*models.TeachingGroupListItem, error) {
	return repository.GetUserTeachingGroups(ctx, userID)
}

// CheckReviewPermission 校验用户是否为教研组长或骨干教师。
func (s *OrganizationService) CheckReviewPermission(
	ctx context.Context,
	groupID string,
	userID string,
) error {
	hasPermission, err :=
		repository.IsGroupLeadOrBackbone(
			ctx,
			groupID,
			userID,
		)
	if err != nil {
		return err
	}
	if !hasPermission {
		return ErrNoReviewPermission
	}

	return nil
}
