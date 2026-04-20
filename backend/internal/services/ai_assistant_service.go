package services

// ai_assistant_service.go — AI 助手业务逻辑层
//
// 职责:
//   - 权限判断(source 归属校验、学校归属校验)
//   - 参数校验(必填字段、枚举值、场景合法性)
//   - 组装可见性规则传给 Repository
//   - 维护数据一致性(如 personal/group 助手不可跨用户/跨校篡改)
//
// 权限矩阵:
//   admin           : 可创建/编辑/删除 system 助手 + 自己的 personal 助手
//   senior_operator : 可创建/编辑/删除 本校 group 助手 + 自己的 personal 助手
//   operator/viewer : 仅可创建/编辑/删除 自己的 personal 助手
//   所有人均可查看:system + 本校 group + 自己的 personal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrAssistantNameRequired   = errors.New("助手名称不能为空")
	ErrAssistantPromptRequired = errors.New("助手提示词不能为空")
	ErrAssistantScenesRequired = errors.New("助手适用场景至少选择一项")
	ErrAssistantInvalidSource  = errors.New("助手来源无效")
	ErrAssistantInvalidScene   = errors.New("助手场景代码无效")
	ErrAssistantPermDenied     = errors.New("无权操作此助手")
	ErrAssistantPromptTooLong  = errors.New("助手提示词长度超过上限(128KB)")
	ErrSchoolBindingRequired   = errors.New("创建本校助手前,当前账号需先绑定学校管理员身份")
)

// 提示词最大长度(128KB,足够放下 v3.0 / v1.0 这种 60K/30K 级别的长 prompt)
const maxAssistantPromptLen = 128 * 1024

// ==================== 服务结构体 ====================

// AIAssistantService AI 助手服务
type AIAssistantService struct{}

// NewAIAssistantService 创建服务实例
func NewAIAssistantService() *AIAssistantService {
	return &AIAssistantService{}
}

// ==================== 上下文结构 ====================

// AssistantActorContext 操作者上下文(调用方从 JWT claims 解析后传入)
type AssistantActorContext struct {
	UserID   string // 当前用户 ID
	Role     string // 角色:admin / senior_operator / operator / viewer
	SchoolID string // 当前用户所属学校 ID(senior_operator 专属,其他为空)
}

// ==================== 1. 列表 ====================

// ListAssistants 根据场景和用户角色返回可见助手
// Scene/Subject/GradeRange 来自前端 query,其余字段由调用方根据 JWT 填充
func (s *AIAssistantService) ListAssistants(
	ctx context.Context,
	actor *AssistantActorContext,
	scene, subject, gradeRange string,
	onlyActive bool,
) (*models.AIAssistantListResponse, error) {
	params := &models.ListAIAssistantsParams{
		Scene:           scene,
		Subject:         subject,
		GradeRange:      gradeRange,
		CurrentUserID:   actor.UserID,
		CurrentUserRole: actor.Role,
		CurrentSchoolID: actor.SchoolID,
		OnlyActive:      onlyActive,
	}
	items, total, err := repository.ListAIAssistants(ctx, params)
	if err != nil {
		return nil, err
	}
	return &models.AIAssistantListResponse{
		Assistants: items,
		Total:      total,
	}, nil
}

// ==================== 2. 获取详情 ====================

// GetAssistant 获取助手详情并校验可见性
// 可见:system(所有人) / group(本校) / personal(自己)
func (s *AIAssistantService) GetAssistant(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
) (*models.AIAssistant, error) {
	a, err := repository.GetAIAssistantByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !s.canView(actor, a) {
		return nil, ErrAssistantPermDenied
	}
	return a, nil
}

// canView 判断当前用户是否能查看该助手
func (s *AIAssistantService) canView(actor *AssistantActorContext, a *models.AIAssistant) bool {
	switch a.Source {
	case models.AssistantSourceSystem:
		return true
	case models.AssistantSourceGroup:
		// 必须属于同一所学校(admin 可查看全部)
		if actor.Role == models.RoleAdmin {
			return true
		}
		if a.OrganizationID == nil || actor.SchoolID == "" {
			return false
		}
		return *a.OrganizationID == actor.SchoolID
	case models.AssistantSourcePersonal:
		// 创建者本人 + admin
		if actor.Role == models.RoleAdmin {
			return true
		}
		if a.CreatedBy == nil {
			return false
		}
		return *a.CreatedBy == actor.UserID
	}
	return false
}

// ==================== 3. 创建 ====================

// CreateAssistant 创建助手
// source 根据用户角色自动判定(不信任前端传入的 source):
//   - admin         → system(如果前端要 personal,可改为 personal)
//   - senior_operator → group 或 personal(前端选)
//   - operator/viewer → 仅 personal
func (s *AIAssistantService) CreateAssistant(
	ctx context.Context,
	actor *AssistantActorContext,
	req *models.CreateAIAssistantRequest,
) (*models.AIAssistant, error) {
	// 校验必填字段
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrAssistantNameRequired
	}
	if strings.TrimSpace(req.FullPrompt) == "" {
		return nil, ErrAssistantPromptRequired
	}
	if len(req.FullPrompt) > maxAssistantPromptLen {
		return nil, ErrAssistantPromptTooLong
	}
	if len(req.Scenes) == 0 {
		return nil, ErrAssistantScenesRequired
	}
	for _, sc := range req.Scenes {
		if !models.IsValidAssistantScene(sc) {
			return nil, fmt.Errorf("%w: %s", ErrAssistantInvalidScene, sc)
		}
	}

	// 决定实际 source(不相信前端的 source,按角色校验并纠正)
	actualSource, err := s.resolveSource(actor, req.Source)
	if err != nil {
		return nil, err
	}

	// 场景序列化
	scenesJSON, _ := json.Marshal(req.Scenes)

	// 构建实体
	a := &models.AIAssistant{
		Name:              strings.TrimSpace(req.Name),
		AvatarEmoji:       strings.TrimSpace(req.AvatarEmoji),
		Description:       strings.TrimSpace(req.Description),
		Source:            actualSource,
		FullPrompt:        req.FullPrompt,
		KnowledgeRefs:     "[]",
		Subject:           strings.TrimSpace(req.Subject),
		GradeRange:        strings.TrimSpace(req.GradeRange),
		Scenes:            string(scenesJSON),
		ForkedFrom:        req.ForkedFrom,
		SortOrder:         0,
		IsDefaultForScene: "[]",
		IsActive:          true,
	}

	// 按 source 设置归属
	switch actualSource {
	case models.AssistantSourceSystem:
		// 系统助手无归属
	case models.AssistantSourceGroup:
		if actor.SchoolID == "" {
			return nil, ErrSchoolBindingRequired
		}
		schoolID := actor.SchoolID
		userID := actor.UserID
		a.OrganizationID = &schoolID
		a.CreatedBy = &userID
	case models.AssistantSourcePersonal:
		userID := actor.UserID
		a.CreatedBy = &userID
	}

	if err := repository.CreateAIAssistant(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// resolveSource 根据用户角色和前端请求决定实际 source
// 安全原则:永远不信任前端传的 source,只根据角色允许什么就用什么
func (s *AIAssistantService) resolveSource(actor *AssistantActorContext, reqSource string) (string, error) {
	if reqSource == "" {
		// 不指定时按角色默认
		switch actor.Role {
		case models.RoleAdmin:
			return models.AssistantSourceSystem, nil
		case models.RoleSeniorOperator:
			return models.AssistantSourcePersonal, nil
		default:
			return models.AssistantSourcePersonal, nil
		}
	}

	if !models.IsValidAssistantSource(reqSource) {
		return "", ErrAssistantInvalidSource
	}

	switch reqSource {
	case models.AssistantSourceSystem:
		if actor.Role != models.RoleAdmin {
			return "", fmt.Errorf("%w: 仅系统管理员可创建 system 助手", ErrAssistantPermDenied)
		}
		return reqSource, nil
	case models.AssistantSourceGroup:
		if actor.Role != models.RoleSeniorOperator && actor.Role != models.RoleAdmin {
			return "", fmt.Errorf("%w: 仅学校管理员可创建本校助手", ErrAssistantPermDenied)
		}
		return reqSource, nil
	case models.AssistantSourcePersonal:
		return reqSource, nil
	}
	return "", ErrAssistantInvalidSource
}

// ==================== 4. 更新 ====================

// UpdateAssistant 更新助手
// 只有归属者(及 admin)可以编辑
func (s *AIAssistantService) UpdateAssistant(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
	req *models.UpdateAIAssistantRequest,
) error {
	a, err := repository.GetAIAssistantByID(ctx, id)
	if err != nil {
		return err
	}
	if !s.canEdit(actor, a) {
		return ErrAssistantPermDenied
	}

	// 校验
	if strings.TrimSpace(req.Name) == "" {
		return ErrAssistantNameRequired
	}
	if strings.TrimSpace(req.FullPrompt) == "" {
		return ErrAssistantPromptRequired
	}
	if len(req.FullPrompt) > maxAssistantPromptLen {
		return ErrAssistantPromptTooLong
	}
	if len(req.Scenes) == 0 {
		return ErrAssistantScenesRequired
	}
	for _, sc := range req.Scenes {
		if !models.IsValidAssistantScene(sc) {
			return fmt.Errorf("%w: %s", ErrAssistantInvalidScene, sc)
		}
	}

	return repository.UpdateAIAssistant(ctx, id, req)
}

// canEdit 判断当前用户是否能编辑该助手
func (s *AIAssistantService) canEdit(actor *AssistantActorContext, a *models.AIAssistant) bool {
	// admin 可编辑任何助手
	if actor.Role == models.RoleAdmin {
		return true
	}

	switch a.Source {
	case models.AssistantSourceSystem:
		// system 仅 admin 可编辑
		return false
	case models.AssistantSourceGroup:
		// group 仅本校 senior_operator 可编辑
		if actor.Role != models.RoleSeniorOperator {
			return false
		}
		if a.OrganizationID == nil || actor.SchoolID == "" {
			return false
		}
		return *a.OrganizationID == actor.SchoolID
	case models.AssistantSourcePersonal:
		// personal 仅创建者可编辑
		if a.CreatedBy == nil {
			return false
		}
		return *a.CreatedBy == actor.UserID
	}
	return false
}

// ==================== 5. 删除 ====================

// DeleteAssistant 删除助手(硬删除)
// system 助手不允许删除(仅 admin 可改 is_active=false 停用)
func (s *AIAssistantService) DeleteAssistant(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
) error {
	a, err := repository.GetAIAssistantByID(ctx, id)
	if err != nil {
		return err
	}

	// system 助手禁止硬删除
	if a.Source == models.AssistantSourceSystem {
		return fmt.Errorf("%w: 系统助手不可删除,如需停用请修改 is_active", ErrAssistantPermDenied)
	}

	if !s.canEdit(actor, a) {
		return ErrAssistantPermDenied
	}
	return repository.DeleteAIAssistant(ctx, id)
}

// ==================== 6. Fork(复制到我的) ====================

// ForkAssistant 将系统/本校助手复制一份到"我的"
// 复制后 source=personal,创建者为当前用户,full_prompt/scenes 原样复制
func (s *AIAssistantService) ForkAssistant(
	ctx context.Context,
	actor *AssistantActorContext,
	sourceID string,
) (*models.AIAssistant, error) {
	// 首先校验能看到原助手
	origin, err := s.GetAssistant(ctx, actor, sourceID)
	if err != nil {
		return nil, err
	}

	// 构造 personal 副本
	userID := actor.UserID
	newAssistant := &models.AIAssistant{
		Name:              origin.Name + " (我的副本)",
		AvatarEmoji:       origin.AvatarEmoji,
		Description:       origin.Description,
		Source:            models.AssistantSourcePersonal,
		CreatedBy:         &userID,
		FullPrompt:        origin.FullPrompt,
		KnowledgeRefs:     origin.KnowledgeRefs,
		Subject:           origin.Subject,
		GradeRange:        origin.GradeRange,
		Scenes:            origin.Scenes,
		ForkedFrom:        &origin.ID,
		SortOrder:         0,
		IsDefaultForScene: "[]",
		IsActive:          true,
	}

	if err := repository.CreateAIAssistant(ctx, newAssistant); err != nil {
		return nil, err
	}
	return newAssistant, nil
}

// ==================== 7. 运行时使用(供对话入口调用) ====================

// LoadActiveAssistantForUse 加载一个助手用于对话(含可见性校验 + is_active 校验 + 使用量埋点)
// 评审工作台 / 工坊各阶段调用此方法取得助手内容
func (s *AIAssistantService) LoadActiveAssistantForUse(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
) (*models.AIAssistant, error) {
	a, err := repository.GetAIAssistantByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !a.IsActive {
		return nil, repository.ErrAIAssistantInactive
	}
	if !s.canView(actor, a) {
		return nil, ErrAssistantPermDenied
	}

	// 异步埋点(P0 预留,失败不影响主流程)
	go func(aid string) {
		_ = repository.IncrementAIAssistantUseCount(context.Background(), aid)
	}(a.ID)

	return a, nil
}

// BuildActorFromClaims 辅助工具:从 JWT claims 和仓储反查构建 ActorContext
// 供 handler / 其他 service 复用
func BuildActorFromClaims(ctx context.Context, userID, role string) *AssistantActorContext {
	actor := &AssistantActorContext{
		UserID: userID,
		Role:   role,
	}

	// senior_operator 额外查出管理的学校 ID
	if role == models.RoleSeniorOperator {
		school, err := repository.GetSchoolByAdminUserID(ctx, userID)
		if err == nil && school != nil {
			actor.SchoolID = school.ID
		}
	}

	return actor
}
