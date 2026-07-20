package services

// course_outline_service.go — 课程大纲业务逻辑
//
// 上下文16教育域收口：
//   1. 所有入口只接收userID，不再信任JWT中的role；
//   2. 每次请求实时读取users.role并严格解析唯一具体教学域；
//   3. 列表只返回操作者当前教育域内、且原有数据范围可见的资源；
//   4. mixed、异常域、无教学组织和跨域冲突在普通列表及出版社接口返回安全空数组；
//   5. admin保留K12课程大纲管理能力，但出版社选择接口仍返回空数组；
//   6. K12普通教学身份的出版社列表保持原行为；
//   7. vocational/adult出版社列表固定返回空数组；
//   8. 创建、更新、删除必须同时满足资源归属域与操作者实时域一致；
//   9. vocational/adult可以创建和编辑普通课程大纲，但publisher必须为空；
//  10. 通过直接API伪造人教版等具名出版社会被Service拒绝；
//  11. 详情执行同域和可见范围校验，跨域ID按不存在处理，防止资源探测；
//  12. Handler根据本服务返回的教育域决定是否输出publisher字段。

import (
	"context"
	"errors"
	"sort"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var courseOutlineLog = logger.WithModule(
	"services.course_outline",
)

var (
	ErrOutlineFieldRequired = errors.New(
		"学科、年级、册次、标题、正文均为必填",
	)
	ErrOutlineScopeInvalid = errors.New(
		"归属类型非法",
	)
	ErrOutlineNoPermission = errors.New(
		"您没有权限管理该归属的课程大纲",
	)
)

// CourseOutlineService 课程大纲服务。
type CourseOutlineService struct{}

// NewCourseOutlineService 创建服务。
func NewCourseOutlineService() *CourseOutlineService {
	return &CourseOutlineService{}
}

// ListOutlines 列出当前用户同教育域且可见的课程大纲。
//
// 普通mixed、异常域、无教学组织或跨域冲突返回成功空列表。
// admin使用受限K12管理兼容域，保留现有K12基础数据管理能力。
// 数据库和基础设施错误仍向上传递。
func (s *CourseOutlineService) ListOutlines(
	ctx context.Context,
	userID string,
) (
	[]*models.CourseOutlineListItem,
	string,
	error,
) {
	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		if isCourseOutlineSafeEmptyDomainError(
			err,
		) {
			return []*models.CourseOutlineListItem{},
				"",
				nil
		}

		return nil, "", err
	}

	groupIDs := s.resolveUserGroupIDs(
		ctx,
		actor.UserID,
	)
	schoolIDs := s.resolveUserSchoolIDs(
		ctx,
		actor.Role,
		actor.UserID,
	)

	items, err := repository.ListCourseOutlines(
		ctx,
		actor.Role == models.RoleAdmin,
		groupIDs,
		schoolIDs,
		actor.EducationDomain,
	)
	if err != nil {
		return nil, "", err
	}

	if items == nil {
		items = []*models.CourseOutlineListItem{}
	}

	return items,
		actor.EducationDomain,
		nil
}

// GetOutline 获取单条大纲并执行教育域与可见范围校验。
//
// 跨域、不可见或归属异常统一返回“课程大纲不存在”，
// 避免通过ID枚举探测其它教育域资源。
func (s *CourseOutlineService) GetOutline(
	ctx context.Context,
	userID string,
	id string,
) (
	*models.CourseOutline,
	string,
	error,
) {
	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		return nil, "", err
	}

	outline, err :=
		repository.GetCourseOutlineByID(
			ctx,
			strings.TrimSpace(id),
		)
	if err != nil {
		return nil, "", err
	}

	resourceDomain, err :=
		resolveCourseOutlineResourceDomain(
			ctx,
			outline.Scope,
			outline.ScopeTargetID,
		)
	if err != nil {
		courseOutlineLog.Warn(
			"课程大纲详情归属教育域解析失败",
			"outline_id", outline.ID,
			"scope", outline.Scope,
			"scope_target_id",
			outline.ScopeTargetID,
			"error", err,
		)

		return nil,
			"",
			repository.ErrCourseOutlineNotFound
	}

	if resourceDomain !=
		actor.EducationDomain {
		return nil,
			"",
			repository.ErrCourseOutlineNotFound
	}

	if !s.canViewScope(
		ctx,
		actor,
		outline.Scope,
		outline.ScopeTargetID,
	) {
		return nil,
			"",
			repository.ErrCourseOutlineNotFound
	}

	return outline,
		actor.EducationDomain,
		nil
}

// ListAvailablePublishers 查询K12学科和年级真实存在的课程大纲版本。
//
// 安全空列表：
//   - vocational；
//   - adult；
//   - admin等mixed管理身份；
//   - 无教学组织；
//   - 教育域异常；
//   - 跨域冲突。
//
// 只有普通K12教学身份继续查询数据库。
func (s *CourseOutlineService) ListAvailablePublishers(
	ctx context.Context,
	userID string,
	subject string,
	grade string,
) ([]string, error) {
	subject = strings.TrimSpace(subject)
	grade = strings.TrimSpace(grade)

	if subject == "" || grade == "" {
		return []string{}, nil
	}

	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		if isCourseOutlineSafeEmptyDomainError(
			err,
		) {
			return []string{}, nil
		}

		return nil, err
	}

	// admin虽然为K12课程大纲管理保留兼容域，
	// 但其本质仍是mixed管理身份，不应获得普通备课出版社选择结果。
	if actor.MixedManagement {
		return []string{}, nil
	}

	if actor.EducationDomain !=
		models.EducationDomainK12 {
		return []string{}, nil
	}

	candidates, err :=
		repository.
			ListActiveOutlinesBySubjectAndEducationDomain(
				ctx,
				subject,
				actor.EducationDomain,
			)
	if err != nil {
		return nil, err
	}

	hits := MatchOutlines(
		grade,
		candidates,
	)
	if len(hits) == 0 {
		return []string{}, nil
	}

	seen := make(
		map[string]struct{},
		len(hits),
	)
	publishers := make(
		[]string,
		0,
		len(hits),
	)

	for _, outline := range hits {
		publisher := strings.TrimSpace(
			outline.Publisher,
		)
		if _, exists := seen[publisher]; exists {
			continue
		}

		seen[publisher] = struct{}{}
		publishers = append(
			publishers,
			publisher,
		)
	}

	sort.SliceStable(
		publishers,
		func(i int, j int) bool {
			if publishers[i] == "" {
				return false
			}
			if publishers[j] == "" {
				return true
			}

			return publishers[i] <
				publishers[j]
		},
	)

	return publishers, nil
}

// CreateOutline 创建课程大纲。
func (s *CourseOutlineService) CreateOutline(
	ctx context.Context,
	userID string,
	req *models.CreateCourseOutlineRequest,
) (
	*models.CourseOutline,
	string,
	error,
) {
	if req == nil {
		return nil,
			"",
			ErrOutlineFieldRequired
	}

	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		return nil, "", err
	}

	req.Scope = strings.TrimSpace(
		req.Scope,
	)
	req.ScopeTargetID = strings.TrimSpace(
		req.ScopeTargetID,
	)

	if !models.IsValidCourseOutlineScope(
		req.Scope,
	) {
		return nil,
			"",
			ErrOutlineScopeInvalid
	}

	if req.Scope ==
		models.CourseOutlineScopeSystem {
		req.ScopeTargetID =
			models.CourseOutlineSystemTargetID
	}

	req.Subject = strings.TrimSpace(
		req.Subject,
	)
	req.Grade = strings.TrimSpace(
		req.Grade,
	)
	req.Volume = strings.TrimSpace(
		req.Volume,
	)
	req.Title = strings.TrimSpace(
		req.Title,
	)

	if req.Subject == "" ||
		req.Grade == "" ||
		req.Volume == "" ||
		req.Title == "" ||
		strings.TrimSpace(
			req.Content,
		) == "" ||
		req.ScopeTargetID == "" {
		return nil,
			"",
			ErrOutlineFieldRequired
	}

	resourceDomain, err :=
		resolveCourseOutlineResourceDomain(
			ctx,
			req.Scope,
			req.ScopeTargetID,
		)
	if err != nil {
		return nil, "", err
	}

	if resourceDomain !=
		actor.EducationDomain {
		return nil,
			"",
			ErrOutlineEducationDomainMismatch
	}

	if !s.canManageScope(
		ctx,
		actor,
		req.Scope,
		req.ScopeTargetID,
	) {
		return nil,
			"",
			ErrOutlineNoPermission
	}

	publisher, err :=
		normalizeCourseOutlinePublisherForDomain(
			actor.EducationDomain,
			req.Publisher,
		)
	if err != nil {
		return nil, "", err
	}

	outline := &models.CourseOutline{
		Scope:         req.Scope,
		ScopeTargetID: req.ScopeTargetID,
		Subject:       req.Subject,
		Grade:         req.Grade,
		Volume:        req.Volume,
		Publisher:     publisher,
		Title:         req.Title,
		Content:       req.Content,
		SourceType:
			models.CourseOutlineSourcePaste,
		CreatedBy: actor.UserID,
	}

	if err := repository.CreateCourseOutline(
		ctx,
		outline,
	); err != nil {
		return nil, "", err
	}

	return outline,
		actor.EducationDomain,
		nil
}

// UpdateOutline 更新课程大纲。
func (s *CourseOutlineService) UpdateOutline(
	ctx context.Context,
	userID string,
	id string,
	req *models.UpdateCourseOutlineRequest,
) error {
	if req == nil {
		return ErrOutlineFieldRequired
	}

	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		return err
	}

	existing, err :=
		repository.GetCourseOutlineByID(
			ctx,
			strings.TrimSpace(id),
		)
	if err != nil {
		return err
	}

	resourceDomain, err :=
		resolveCourseOutlineResourceDomain(
			ctx,
			existing.Scope,
			existing.ScopeTargetID,
		)
	if err != nil {
		return err
	}

	if resourceDomain !=
		actor.EducationDomain {
		return ErrOutlineEducationDomainMismatch
	}

	if !s.canManageScope(
		ctx,
		actor,
		existing.Scope,
		existing.ScopeTargetID,
	) {
		return ErrOutlineNoPermission
	}

	req.Subject = strings.TrimSpace(
		req.Subject,
	)
	req.Grade = strings.TrimSpace(
		req.Grade,
	)
	req.Volume = strings.TrimSpace(
		req.Volume,
	)
	req.Title = strings.TrimSpace(
		req.Title,
	)

	if req.Subject == "" ||
		req.Grade == "" ||
		req.Volume == "" ||
		req.Title == "" ||
		strings.TrimSpace(
			req.Content,
		) == "" {
		return ErrOutlineFieldRequired
	}

	publisher, err :=
		normalizeCourseOutlinePublisherForDomain(
			actor.EducationDomain,
			req.Publisher,
		)
	if err != nil {
		return err
	}
	req.Publisher = publisher

	return repository.UpdateCourseOutline(
		ctx,
		existing.ID,
		req,
	)
}

// DeleteOutline 软删除课程大纲。
func (s *CourseOutlineService) DeleteOutline(
	ctx context.Context,
	userID string,
	id string,
) error {
	actor, err := resolveCourseOutlineActor(
		ctx,
		userID,
	)
	if err != nil {
		return err
	}

	existing, err :=
		repository.GetCourseOutlineByID(
			ctx,
			strings.TrimSpace(id),
		)
	if err != nil {
		return err
	}

	resourceDomain, err :=
		resolveCourseOutlineResourceDomain(
			ctx,
			existing.Scope,
			existing.ScopeTargetID,
		)
	if err != nil {
		return err
	}

	if resourceDomain !=
		actor.EducationDomain {
		return ErrOutlineEducationDomainMismatch
	}

	if !s.canManageScope(
		ctx,
		actor,
		existing.Scope,
		existing.ScopeTargetID,
	) {
		return ErrOutlineNoPermission
	}

	return repository.DeleteCourseOutline(
		ctx,
		existing.ID,
	)
}

// canViewScope 判断用户是否拥有资源读取范围。
func (s *CourseOutlineService) canViewScope(
	ctx context.Context,
	actor *courseOutlineActor,
	scope string,
	targetID string,
) bool {
	if actor == nil {
		return false
	}

	if actor.Role == models.RoleAdmin {
		return true
	}

	switch scope {
	case models.CourseOutlineScopeSystem:
		return actor.EducationDomain ==
			models.EducationDomainK12

	case models.CourseOutlineScopeGroup:
		for _, groupID :=
			range s.resolveUserGroupIDs(
				ctx,
				actor.UserID,
			) {
			if groupID == targetID {
				return true
			}
		}

	case models.CourseOutlineScopeSchool:
		for _, schoolID :=
			range s.resolveUserSchoolIDs(
				ctx,
				actor.Role,
				actor.UserID,
			) {
			if schoolID == targetID {
				return true
			}
		}
	}

	return false
}

// canManageScope 判断用户是否拥有资源写权限。
func (s *CourseOutlineService) canManageScope(
	ctx context.Context,
	actor *courseOutlineActor,
	scope string,
	targetID string,
) bool {
	if actor == nil {
		return false
	}

	if actor.Role == models.RoleAdmin {
		return true
	}

	switch scope {
	case models.CourseOutlineScopeSystem:
		return false

	case models.CourseOutlineScopeSchool:
		if actor.Role !=
			models.RoleSeniorOperator {
			return false
		}

		school, err :=
			repository.GetSchoolByAdminUserID(
				ctx,
				actor.UserID,
			)
		if err != nil ||
			school == nil {
			return false
		}

		return school.ID == targetID

	case models.CourseOutlineScopeGroup:
		allowed, err :=
			repository.IsGroupLeadOrBackbone(
				ctx,
				targetID,
				actor.UserID,
			)
		if err != nil {
			courseOutlineLog.Warn(
				"校验课程大纲教研组管理权限失败",
				"group", targetID,
				"user", actor.UserID,
				"error", err,
			)
			return false
		}

		return allowed
	}

	return false
}

// resolveUserGroupIDs 解析用户所属教研组ID。
func (s *CourseOutlineService) resolveUserGroupIDs(
	ctx context.Context,
	userID string,
) []string {
	groups, err :=
		repository.GetUserTeachingGroups(
			ctx,
			userID,
		)
	if err != nil {
		courseOutlineLog.Warn(
			"查询用户课程大纲可见教研组失败",
			"user", userID,
			"error", err,
		)
		return []string{}
	}

	groupIDs := make(
		[]string,
		0,
		len(groups),
	)
	for _, group := range groups {
		if group != nil &&
			strings.TrimSpace(group.ID) != "" {
			groupIDs = append(
				groupIDs,
				group.ID,
			)
		}
	}

	return groupIDs
}

// resolveUserSchoolIDs 解析现有课程大纲列表允许读取的学校ID。
//
// 保持原有范围：只有senior_operator读取其管理学校的school级大纲。
func (s *CourseOutlineService) resolveUserSchoolIDs(
	ctx context.Context,
	role string,
	userID string,
) []string {
	if role !=
		models.RoleSeniorOperator {
		return []string{}
	}

	school, err :=
		repository.GetSchoolByAdminUserID(
			ctx,
			userID,
		)
	if err != nil ||
		school == nil ||
		strings.TrimSpace(
			school.ID,
		) == "" {
		return []string{}
	}

	return []string{
		school.ID,
	}
}
