package services

// ai_assistant_service.go — AI 助手业务逻辑层
//
// 职责:
//   - 权限判断(source 归属校验、教研组/学校归属校验)
//   - 参数校验(必填字段、枚举值、场景合法性)
//   - 组装可见性规则传给 Repository
//   - 维护数据一致性(如 personal/group 助手不可跨用户/跨组篡改)
//
// ──────────────────────────────────────────────────────────────────────
// 里程碑一(教研组级分享打通)权限矩阵:
//   admin            : 创建/编辑/删除 system 助手 + 任何 group/personal 助手
//   senior_operator  : 创建全校级 group 助手(group_id 空,本校可见) + 自己的 personal
//                      + 作为某教研组 lead/backbone 时也可发教研组级 group 助手
//   operator/viewer  : 自己的 personal 助手
//                      + 作为某教研组 lead/backbone 时可发教研组级 group 助手
//
//   group 来源细分两档(靠 ai_assistants.group_id 是否为空区分):
//     - group_id 非空 = 教研组级:仅该教研组成员可见;创建者本人 / 该组 lead / admin 可编辑
//     - group_id 为空 = 全校级:  本校所有人可见;创建者(校管) / admin 可编辑
//
//   可见(canView):
//     system → 所有人
//     group 教研组级 → 该组成员(group_id ∈ 我的教研组) + admin
//     group 全校级   → 同校 + admin
//     personal → 创建者本人 + admin
// ──────────────────────────────────────────────────────────────────────

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
        ErrAssistantNameRequired    = errors.New("助手名称不能为空")
        ErrAssistantPromptRequired  = errors.New("助手提示词不能为空")
        ErrAssistantScenesRequired  = errors.New("助手适用场景至少选择一项")
        ErrAssistantInvalidSource   = errors.New("助手来源无效")
        ErrAssistantInvalidScene    = errors.New("助手场景代码无效")
        ErrAssistantPermDenied      = errors.New("无权操作此助手")
        ErrAssistantPromptTooLong   = errors.New("助手提示词长度超过上限(128KB)")
        ErrSchoolBindingRequired    = errors.New("创建全校级助手前,当前账号需先绑定学校管理员身份")
        ErrAssistantGroupNotAllowed = errors.New("无权在该教研组发布助手(需为该组组长或骨干)")
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
//
// 里程碑一新增教研组三字段,由 BuildActorFromClaims 统一填充:
//   MyGroupIDs               我所属的全部教研组 ID(成员/骨干/组长都算)→ 定可见范围
//   MyLeadGroupIDs           我担任 lead(组长)的教研组 ID            → 定能否编辑该组助手
//   MyLeadOrBackboneGroupIDs 我担任 lead 或 backbone 的教研组 ID      → 定能往哪些组发布
type AssistantActorContext struct {
        UserID   string // 当前用户 ID
        Role     string // 角色:admin / senior_operator / operator / viewer
        SchoolID string // 当前用户所属学校 ID(senior_operator 经管理员身份反查;其他用户经教研组兜底反查)

        MyGroupIDs               []string // 我所属的全部教研组 ID
        MyLeadGroupIDs           []string // 我担任组长(lead)的教研组 ID
        MyLeadOrBackboneGroupIDs []string // 我担任组长或骨干的教研组 ID(可发布目标)
}

// containsStr 判断字符串切片是否包含目标值(本文件内部小工具)
func containsStr(list []string, target string) bool {
        for _, v := range list {
                if v == target {
                        return true
                }
        }
        return false
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
                CurrentGroupIDs: actor.MyGroupIDs, // 里程碑一:透传我的教研组集合供可见性 SQL
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
//
// 里程碑一:group 来源细分教研组级 / 全校级两档
func (s *AIAssistantService) canView(actor *AssistantActorContext, a *models.AIAssistant) bool {
        switch a.Source {
        case models.AssistantSourceSystem:
                return true
        case models.AssistantSourceGroup:
                if actor.Role == models.RoleAdmin {
                        return true
                }
                // 教研组级(group_id 非空):仅该组成员可见
                if a.GroupID != nil && *a.GroupID != "" {
                        return containsStr(actor.MyGroupIDs, *a.GroupID)
                }
                // 全校级(group_id 空):同校可见
                if a.OrganizationID == nil || actor.SchoolID == "" {
                        return false
                }
                return *a.OrganizationID == actor.SchoolID
        case models.AssistantSourcePersonal:
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
//
// 里程碑一 source 与归属判定:
//   - admin            → 默认 system;前端可显式选 group/personal
//   - senior_operator  → 默认 personal;可选全校级 group(不带 group_id)或教研组级 group(带 group_id)
//   - operator/viewer  → 默认 personal;若是某组 lead/backbone,可选教研组级 group(带 group_id)
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

        // 决定实际 source(不相信前端的 source,按角色+教研组身份校验并纠正)
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
                userID := actor.UserID
                a.CreatedBy = &userID

                reqGroupID := ""
                if req.GroupID != nil {
                        reqGroupID = strings.TrimSpace(*req.GroupID)
                }

                if reqGroupID != "" {
                        // ── 教研组级 ──:校验当前用户确实是该组 lead/backbone
                        if actor.Role != models.RoleAdmin &&
                                !containsStr(actor.MyLeadOrBackboneGroupIDs, reqGroupID) {
                                return nil, ErrAssistantGroupNotAllowed
                        }
                        // 取该教研组所属学校 ID 一并落库(organization_id 仍记学校,用于展示与全校兜底)
                        schoolID, err := s.resolveGroupSchoolID(ctx, reqGroupID)
                        if err != nil {
                                return nil, err
                        }
                        gid := reqGroupID
                        a.GroupID = &gid
                        if schoolID != "" {
                                a.OrganizationID = &schoolID
                        }
                } else {
                        // ── 全校级 ──:group_id 为空,仅 senior_operator/admin 可发,需绑定学校
                        if actor.Role != models.RoleSeniorOperator && actor.Role != models.RoleAdmin {
                                return nil, fmt.Errorf("%w: 仅学校管理员可发布全校级助手", ErrAssistantPermDenied)
                        }
                        if actor.SchoolID == "" {
                                return nil, ErrSchoolBindingRequired
                        }
                        schoolID := actor.SchoolID
                        a.OrganizationID = &schoolID
                        // a.GroupID 保持 nil = 全校级
                }

        case models.AssistantSourcePersonal:
                userID := actor.UserID
                a.CreatedBy = &userID
        }

        if err := repository.CreateAIAssistant(ctx, a); err != nil {
                return nil, err
        }
        return a, nil
}

// resolveSource 根据用户角色和教研组身份决定实际 source
// 安全原则:永远不信任前端传的 source,只根据角色/教研组身份允许什么就用什么
//
// 里程碑一:group 来源放行条件 = senior_operator / admin / 名下有可发布教研组的人
func (s *AIAssistantService) resolveSource(actor *AssistantActorContext, reqSource string) (string, error) {
        if reqSource == "" {
                // 不指定时按角色默认
                switch actor.Role {
                case models.RoleAdmin:
                        return models.AssistantSourceSystem, nil
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
                // 校管/admin 恒可(全校级);或名下有可发布教研组的人(教研组级)
                if actor.Role == models.RoleSeniorOperator ||
                        actor.Role == models.RoleAdmin ||
                        len(actor.MyLeadOrBackboneGroupIDs) > 0 {
                        return reqSource, nil
                }
                return "", fmt.Errorf("%w: 仅学校管理员或教研组组长/骨干可发布共享助手", ErrAssistantPermDenied)
        case models.AssistantSourcePersonal:
                return reqSource, nil
        }
        return "", ErrAssistantInvalidSource
}

// resolveGroupSchoolID 查某教研组所属学校 ID(供教研组级助手落 organization_id)
func (s *AIAssistantService) resolveGroupSchoolID(ctx context.Context, groupID string) (string, error) {
        tg, err := repository.GetTeachingGroupByID(ctx, groupID)
        if err != nil {
                return "", err
        }
        return tg.SchoolID, nil
}

// ==================== 4. 更新 ====================

// UpdateAssistant 更新助手
// 只有归属者(及 admin / 该组组长)可以编辑
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
//
// 里程碑一:
//   system     → 仅 admin
//   group 教研组级 → admin / 创建者本人 / 该组 lead(组员不能改,要改请 fork 成个人助手)
//   group 全校级   → admin / 创建者本人(校管)
//   personal   → admin / 创建者本人
func (s *AIAssistantService) canEdit(actor *AssistantActorContext, a *models.AIAssistant) bool {
        // admin 可编辑任何助手
        if actor.Role == models.RoleAdmin {
                return true
        }

        switch a.Source {
        case models.AssistantSourceSystem:
                return false
        case models.AssistantSourceGroup:
                // 创建者本人恒可编辑
                if a.CreatedBy != nil && *a.CreatedBy == actor.UserID {
                        return true
                }
                // 教研组级:该组 lead 可编辑(组员不可)
                if a.GroupID != nil && *a.GroupID != "" {
                        return containsStr(actor.MyLeadGroupIDs, *a.GroupID)
                }
                // 全校级:除创建者与 admin 外不开放(senior_operator 非创建者也不放,避免跨人篡改)
                return false
        case models.AssistantSourcePersonal:
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

// ForkAssistant 将系统/本校/本组助手复制一份到"我的"
// 复制后 source=personal,创建者为当前用户,full_prompt/scenes 原样复制
// 组员想改组助手 → fork 成自己的 personal 再改,原版不动
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
//
// 里程碑一:对所有用户都查教研组归属(不再只对 senior_operator),
//   - MyGroupIDs:所属全部教研组(GetUserTeachingGroups)→ 可见范围
//   - MyLeadOrBackboneGroupIDs / MyLeadGroupIDs:可发布/可编辑的组(ListMyLeadOrBackboneGroups)
//   - SchoolID:senior_operator 经管理员身份反查;其他用户经教研组所属学校兜底(取第一个组的学校)
func BuildActorFromClaims(ctx context.Context, userID, role string) *AssistantActorContext {
        actor := &AssistantActorContext{
                UserID: userID,
                Role:   role,
        }

        // (1) senior_operator 经管理员身份反查所管理的学校 ID(权威来源)
        if role == models.RoleSeniorOperator {
                school, err := repository.GetSchoolByAdminUserID(ctx, userID)
                if err == nil && school != nil {
                        actor.SchoolID = school.ID
                }
        }

        // (2) 所有用户都查所属教研组,填充 MyGroupIDs + 兜底 SchoolID
        groups, err := repository.GetUserTeachingGroups(ctx, userID)
        if err == nil {
                for _, g := range groups {
                        actor.MyGroupIDs = append(actor.MyGroupIDs, g.ID)
                        // 非 senior_operator 没有管理员学校,用教研组所属学校兜底(取首个)
                        if actor.SchoolID == "" && g.SchoolID != "" {
                                actor.SchoolID = g.SchoolID
                        }
                }
        }

        // (3) 查我担任 lead/backbone 的教研组(可发布目标),并拆出 lead 组(可编辑组助手)
        leadOrBB, err := repository.ListMyLeadOrBackboneGroups(ctx, userID)
        if err == nil {
                for _, g := range leadOrBB {
                        actor.MyLeadOrBackboneGroupIDs = append(actor.MyLeadOrBackboneGroupIDs, g.ID)
                        if g.Role == models.GroupMemberRoleLead {
                                actor.MyLeadGroupIDs = append(actor.MyLeadGroupIDs, g.ID)
                        }
                }
        }

        return actor
}
