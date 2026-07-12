package services

// course_outline_service.go — 课程大纲业务逻辑（大单元备课能力·批次一 + 教材版本增强）
//
// 职责：
//   1. 列表：按角色解析可见范围（admin 全量 / 其余按所属教研组 + 本校过滤；全局 system 人人可见）
//   2. 写操作归属校验（canManageScope）：
//        admin            → 任意大纲可改
//        senior_operator  → 仅本校 school 范围大纲可改
//        lead/backbone    → 仅自己担任组长/骨干的教研组的 group 范围大纲可改
//        system(全局)      → 仅 admin
//   3. 创建时校验目标归属合法（不能往不属于自己的组/校建大纲）；system 由后端填占位归属ID
//   4. 教材版本(publisher)：CRUD 透传，创建/更新写入；新增「按学科+年级查可用版本列表」，
//      供备课首屏的教材版本选择器使用（只列出该学科年级真实存在大纲的版本）。
//
// 复用现有：repository.GetUserTeachingGroups / IsGroupLeadOrBackbone /
//          GetSchoolByAdminUserID（与 data_scope.go 同口径）。

import (
	"context"
	"errors"
	"sort"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var courseOutlineLog = logger.WithModule("services.course_outline")

// 业务错误（供 handler 映射 HTTP 码）
var (
	ErrOutlineFieldRequired = errors.New("学科、年级、册次、标题、正文均为必填")
	ErrOutlineScopeInvalid  = errors.New("归属类型非法")
	ErrOutlineNoPermission  = errors.New("您没有权限管理该归属的课程大纲")
)

// CourseOutlineService 课程大纲服务
type CourseOutlineService struct{}

// NewCourseOutlineService 创建服务
func NewCourseOutlineService() *CourseOutlineService {
	return &CourseOutlineService{}
}

// ListOutlines 列出当前用户可见的课程大纲
//
//	admin → 全量；其余 → 全局(system) + 自己所属教研组 + 本校
func (s *CourseOutlineService) ListOutlines(ctx context.Context, role, userID string) ([]*models.CourseOutlineListItem, error) {
	if role == models.RoleAdmin {
		return repository.ListCourseOutlines(ctx, true, nil, nil)
	}

	// 收集该用户所属的全部教研组ID（备课要看本组大纲，所有成员都可读）
	groups, gErr := repository.GetUserTeachingGroups(ctx, userID)
	if gErr != nil {
		courseOutlineLog.Warn("查询用户教研组失败", "user", userID, "error", gErr)
	}
	groupIDs := make([]string, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}

	// 本校ID（用于看本校 school 范围大纲）
	schoolIDs := s.resolveUserSchoolIDs(ctx, role, userID)

	return repository.ListCourseOutlines(ctx, false, groupIDs, schoolIDs)
}

// ListAvailablePublishers 查某学科+年级下「真实存在大纲」的可选教材版本列表
//
// 供备课首屏的教材版本选择器使用（Yuhan 决策：首屏选版本，没大纲就不关联）：
//   - 只返回该学科、且大纲年级与教案年级「学段相交」的大纲所拥有的版本；
//   - 版本严格去重；空串版本（通用/不限版本）若存在则作为一个独立可选项一并返回；
//   - 一份相交大纲都没有 → 返回空切片（前端据此不显示版本选择、不关联大纲）。
//
// 注意：不做任何跨版本兜底——这里只如实汇报"该学科该年级到底有哪些版本的大纲可用"，
// 老师选哪个版本，注入层就严格只注入哪个版本（见 course_outline_match.go 的版本过滤）。
//
// 返回的字符串切片里，空串("")代表"通用/不限版本"，前端负责把空串显示成"通用/不限版本"。
func (s *CourseOutlineService) ListAvailablePublishers(ctx context.Context, subject, grade string) ([]string, error) {
	subject = strings.TrimSpace(subject)
	grade = strings.TrimSpace(grade)
	if subject == "" || grade == "" {
		return []string{}, nil
	}

	// 按学科粗筛全部 active 大纲，再用与注入同口径的「学段相交」过滤出真正适用本年级的大纲
	candidates, err := repository.ListActiveOutlinesBySubject(ctx, subject)
	if err != nil {
		return nil, err
	}
	hits := MatchOutlines(grade, candidates)
	if len(hits) == 0 {
		return []string{}, nil
	}

	// 去重收集版本（含空串=通用）
	seen := make(map[string]struct{}, len(hits))
	var publishers []string
	for _, o := range hits {
		p := o.Publisher
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		publishers = append(publishers, p)
	}

	// 稳定排序：空串(通用)永远排最后，其余按字典序，保证前端下拉顺序稳定
	sort.SliceStable(publishers, func(i, j int) bool {
		if publishers[i] == "" {
			return false
		}
		if publishers[j] == "" {
			return true
		}
		return publishers[i] < publishers[j]
	})
	return publishers, nil
}

// CreateOutline 创建课程大纲（含字段校验 + 归属合法性校验）
func (s *CourseOutlineService) CreateOutline(ctx context.Context, role, userID string, req *models.CreateCourseOutlineRequest) (*models.CourseOutline, error) {
	if !models.IsValidCourseOutlineScope(req.Scope) {
		return nil, ErrOutlineScopeInvalid
	}

	// 全局大纲无具体归属，后端统一填占位ID（满足 scope_target_id NOT NULL 与唯一索引去重）
	if req.Scope == models.CourseOutlineScopeSystem {
		req.ScopeTargetID = models.CourseOutlineSystemTargetID
	}

	if strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Grade) == "" ||
		strings.TrimSpace(req.Volume) == "" || strings.TrimSpace(req.Title) == "" ||
		strings.TrimSpace(req.Content) == "" || strings.TrimSpace(req.ScopeTargetID) == "" {
		return nil, ErrOutlineFieldRequired
	}

	// 归属校验：不能往不属于自己的组/校建大纲；system 仅 admin
	if !s.canManageScope(ctx, role, userID, req.Scope, req.ScopeTargetID) {
		return nil, ErrOutlineNoPermission
	}

	o := &models.CourseOutline{
		Scope:         req.Scope,
		ScopeTargetID: req.ScopeTargetID,
		Subject:       strings.TrimSpace(req.Subject),
		Grade:         strings.TrimSpace(req.Grade),
		Volume:        strings.TrimSpace(req.Volume),
		Publisher:     strings.TrimSpace(req.Publisher), // 教材版本（空=通用/不限版本）
		Title:         strings.TrimSpace(req.Title),
		Content:       req.Content,
		SourceType:    models.CourseOutlineSourcePaste,
		CreatedBy:     userID,
	}
	if err := repository.CreateCourseOutline(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

// UpdateOutline 更新大纲（先查出归属再校验写权限）
func (s *CourseOutlineService) UpdateOutline(ctx context.Context, role, userID, id string, req *models.UpdateCourseOutlineRequest) error {
	existing, err := repository.GetCourseOutlineByID(ctx, id)
	if err != nil {
		return err
	}
	if !s.canManageScope(ctx, role, userID, existing.Scope, existing.ScopeTargetID) {
		return ErrOutlineNoPermission
	}
	if strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Grade) == "" ||
		strings.TrimSpace(req.Volume) == "" || strings.TrimSpace(req.Title) == "" ||
		strings.TrimSpace(req.Content) == "" {
		return ErrOutlineFieldRequired
	}
	// 版本规范化：去空白后写回（空=通用/不限版本，允许；不强校验是否在预置清单内）
	req.Publisher = strings.TrimSpace(req.Publisher)
	return repository.UpdateCourseOutline(ctx, id, req)
}

// DeleteOutline 软删除大纲（先查归属再校验写权限）
func (s *CourseOutlineService) DeleteOutline(ctx context.Context, role, userID, id string) error {
	existing, err := repository.GetCourseOutlineByID(ctx, id)
	if err != nil {
		return err
	}
	if !s.canManageScope(ctx, role, userID, existing.Scope, existing.ScopeTargetID) {
		return ErrOutlineNoPermission
	}
	return repository.DeleteCourseOutline(ctx, id)
}

// canManageScope 写权限归属校验（增删改统一入口）
//
//	admin            → 任意
//	senior_operator  → 仅 school 范围且 target 是自己学校
//	lead/backbone    → 仅 group 范围且自己是该组 lead/backbone
//	system(全局)      → 仅 admin（普通角色一律拒绝）
func (s *CourseOutlineService) canManageScope(ctx context.Context, role, userID, scope, targetID string) bool {
	if role == models.RoleAdmin {
		return true
	}

	switch scope {
	case models.CourseOutlineScopeSystem:
		// 全局大纲仅 admin 可管；admin 已在函数开头 return true，故此处非 admin 一律拒绝
		return false

	case models.CourseOutlineScopeSchool:
		// 仅校管可管学校级，且必须是自己绑定的学校
		if role != models.RoleSeniorOperator {
			return false
		}
		school, err := repository.GetSchoolByAdminUserID(ctx, userID)
		if err != nil || school == nil {
			return false
		}
		return school.ID == targetID

	case models.CourseOutlineScopeGroup:
		// 组长/骨干可管自己组的 group 级大纲
		isLeadOrBackbone, err := repository.IsGroupLeadOrBackbone(ctx, targetID, userID)
		if err != nil {
			courseOutlineLog.Warn("校验组长/骨干权限失败", "group", targetID, "user", userID, "error", err)
			return false
		}
		return isLeadOrBackbone
	}
	return false
}

// resolveUserSchoolIDs 解析用户可见的学校ID（用于列表过滤本校 school 大纲）
//
//	senior_operator → 其绑定的学校；其余角色 → 暂返空（普通老师本步不依赖看 school 级，留待注入阶段细化）
func (s *CourseOutlineService) resolveUserSchoolIDs(ctx context.Context, role, userID string) []string {
	if role == models.RoleSeniorOperator {
		school, err := repository.GetSchoolByAdminUserID(ctx, userID)
		if err == nil && school != nil && school.ID != "" {
			return []string{school.ID}
		}
	}
	return []string{}
}
