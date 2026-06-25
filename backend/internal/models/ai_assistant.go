package models

// ai_assistant.go — AI 助手数据模型
//
// TE-DNA 3.0 P0 核心实体:统一存储系统/教研员/个人三种来源的 AI 助手
// 三层架构:老师只看助手 → 助手通过 AOCI 调用组件知识库 → 组件库退居幕后
//
// 对应数据库表:ai_assistants(v110 新增)
//
// ──────────────────────────────────────────────────────────────────────
// 里程碑一(教研组级分享打通)改动说明:
//   group 来源细分两档,靠 GroupID 是否为空区分:
//     - GroupID 非空 → 教研组级:仅该教研组成员可见,组长/骨干可发布
//     - GroupID 为空 → 全校级:  本校所有人可见,学校管理员(senior_operator)发布
//   配套字段新增:
//     - CreateAIAssistantRequest.GroupID  前端发布教研组助手时指定目标组
//     - ListAIAssistantsParams.CurrentGroupIDs  当前用户所属的全部教研组 ID(可见性用)
//     - AIAssistantListItem.GroupID / GroupName 列表展示助手归属的教研组
// ──────────────────────────────────────────────────────────────────────

import "time"

// ==================== 来源常量 ====================

const (
        AssistantSourceSystem   = "system"   // 系统预置(admin 管理)
        AssistantSourceGroup    = "group"    // 教研组/本校(senior_operator 发全校级 / 组长骨干发教研组级)
        AssistantSourcePersonal = "personal" // 个人私有
)

// ValidAssistantSources 有效的 source 值
var ValidAssistantSources = []string{
        AssistantSourceSystem,
        AssistantSourceGroup,
        AssistantSourcePersonal,
}

// IsValidAssistantSource 校验 source 是否有效
func IsValidAssistantSource(s string) bool {
        for _, v := range ValidAssistantSources {
                if v == s {
                        return true
                }
        }
        return false
}

// ==================== 场景常量 ====================
//
// scenes 字段是 JSONB 字符串数组,可取以下任意组合:
// - review_workbench:独立全屏评审工作台
// - workshop_analyze:备课工坊 — 教学分析阶段
// - workshop_design: 备课工坊 — 教学设计阶段
// - workshop_write:  备课工坊 — 教案撰写阶段
// - workshop_review: 备课工坊 — AI 评审阶段
// - workshop_revise: 备课工坊 — 修订定稿阶段

const (
        SceneReviewWorkbench = "review_workbench"
        SceneWorkshopAnalyze = "workshop_analyze"
        SceneWorkshopDesign  = "workshop_design"
        SceneWorkshopWrite   = "workshop_write"
        SceneWorkshopReview  = "workshop_review"
        SceneWorkshopRevise  = "workshop_revise"
)

// ValidAssistantScenes 有效的场景列表
var ValidAssistantScenes = []string{
        SceneReviewWorkbench,
        SceneWorkshopAnalyze, SceneWorkshopDesign, SceneWorkshopWrite,
        SceneWorkshopReview, SceneWorkshopRevise,
}

// IsValidAssistantScene 校验场景代码是否有效
func IsValidAssistantScene(s string) bool {
        for _, v := range ValidAssistantScenes {
                if v == s {
                        return true
                }
        }
        return false
}

// ==================== 数据库实体 ====================

// AIAssistant AI 助手主实体(对应 ai_assistants 表)
// 所有 id 字段统一使用 uuid 字符串,保持与系统其他实体一致
type AIAssistant struct {
        ID          string `json:"id"`
        Name        string `json:"name"`
        AvatarEmoji string `json:"avatar_emoji"`
        Description string `json:"description"`

        // 来源与归属
        Source         string  `json:"source"`          // system | group | personal
        CreatedBy      *string `json:"created_by"`      // 创建者用户 ID
        OrganizationID *string `json:"organization_id"` // source=group 时填写(学校 ID)
        // GroupID:教研组 ID。里程碑一启用——
        //   group 来源且非空 = 教研组级助手(仅该组成员可见)
        //   group 来源且为空 = 全校级助手(本校所有人可见,校管发布)
        GroupID *string `json:"group_id"`

        // 核心内容
        FullPrompt    string `json:"full_prompt"`
        KnowledgeRefs string `json:"knowledge_refs"` // JSONB 字符串,元素为组件/教案 ID

        // 匹配维度
        Subject    string `json:"subject"`
        GradeRange string `json:"grade_range"`
        Scenes     string `json:"scenes"` // JSONB 字符串数组

        // 创作轨迹(P0.5 用)
        CreationConversation *string `json:"creation_conversation"`
        ForkedFrom           *string `json:"forked_from"`

        // 数据飞轮(P0 预留,P2 启用)
        UseCount int      `json:"use_count"`
        AvgScore *float64 `json:"avg_score"`

        // 状态与排序
        SortOrder         int    `json:"sort_order"`
        IsDefaultForScene string `json:"is_default_for_scene"` // JSONB 字符串数组
        IsActive          bool   `json:"is_active"`

        // 审计
        CreatedAt *time.Time `json:"created_at"`
        UpdatedAt *time.Time `json:"updated_at"`
}

// ==================== 列表项(带展示辅助字段) ====================

// AIAssistantListItem 列表返回项,附带创建者显示名和学校名(便于前端展示)
type AIAssistantListItem struct {
        ID          string `json:"id"`
        Name        string `json:"name"`
        AvatarEmoji string `json:"avatar_emoji"`
        Description string `json:"description"`
        Source      string `json:"source"`
        SourceLabel string `json:"source_label"` // 中文展示:系统/本校/我的
        Subject     string `json:"subject"`
        GradeRange  string `json:"grade_range"`
        Scenes      []string `json:"scenes"` // 已解析为字符串数组
        UseCount    int      `json:"use_count"`
        AvgScore    *float64 `json:"avg_score"`
        IsActive    bool     `json:"is_active"`
        IsDefaultHere bool   `json:"is_default_here"` // 是否在当前场景被标为默认
        CanEdit       bool   `json:"can_edit"`        // 当前用户能否编辑
        CanDelete     bool   `json:"can_delete"`      // 当前用户能否删除
        CreatorName   string `json:"creator_name"`
        SchoolName    string `json:"school_name"`

        // 里程碑一新增:教研组归属展示
        //   GroupID 非空 = 教研组级助手, GroupName 为该教研组名称
        //   GroupID 为空 = 全校级或非 group 助手, GroupName 为空串
        GroupID   *string `json:"group_id"`
        GroupName string  `json:"group_name"`

        CreatedAt *time.Time `json:"created_at"`
        UpdatedAt *time.Time `json:"updated_at"`
}

// SourceLabelMap source → 中文展示名
// 注:group 来源细分两档的展示文案(教研组级/全校级)由前端按 group_id 有无区分,
//    此处保留 source 维度的兜底标签。
var SourceLabelMap = map[string]string{
        AssistantSourceSystem:   "系统",
        AssistantSourceGroup:    "本校",
        AssistantSourcePersonal: "我的",
}

// ==================== 请求/响应结构 ====================

// CreateAIAssistantRequest 创建助手请求
type CreateAIAssistantRequest struct {
        Name        string   `json:"name"`
        AvatarEmoji string   `json:"avatar_emoji"`
        Description string   `json:"description"`
        Source      string   `json:"source"` // 不可由前端随意设置,handler 内会根据用户角色校验
        FullPrompt  string   `json:"full_prompt"`
        Subject     string   `json:"subject"`
        GradeRange  string   `json:"grade_range"`
        Scenes      []string `json:"scenes"`
        ForkedFrom  *string  `json:"forked_from"`

        // 里程碑一新增:GroupID 发布教研组级助手时指定目标教研组
        //   - 非空:发布到该教研组(service 校验当前用户是否为该组 lead/backbone)
        //   - 为空且 source=group:发布全校级助手(仅 senior_operator/admin)
        //   - 为空且 source=personal:个人助手,本字段忽略
        GroupID *string `json:"group_id"`
}

// UpdateAIAssistantRequest 更新助手请求(只允许改内容和匹配维度,不允许改 source/归属)
type UpdateAIAssistantRequest struct {
        Name        string   `json:"name"`
        AvatarEmoji string   `json:"avatar_emoji"`
        Description string   `json:"description"`
        FullPrompt  string   `json:"full_prompt"`
        Subject     string   `json:"subject"`
        GradeRange  string   `json:"grade_range"`
        Scenes      []string `json:"scenes"`
        IsActive    *bool    `json:"is_active"`
}

// AIAssistantListResponse 列表响应
type AIAssistantListResponse struct {
        Assistants []*AIAssistantListItem `json:"assistants"`
        Total      int                    `json:"total"`
}

// ==================== 列表查询参数 ====================

// ListAIAssistantsParams 列表查询参数
type ListAIAssistantsParams struct {
        // 场景筛选(空=全部)
        Scene string

        // 学科筛选(空=全部)
        Subject string

        // 年级筛选(空=全部)
        GradeRange string

        // 可见性筛选
        CurrentUserID   string // 当前用户 ID(用于过滤 personal)
        CurrentUserRole string // 当前用户角色
        CurrentSchoolID string // 当前用户所属学校 ID(用于过滤 group 全校级,可为空)

        // 里程碑一新增:当前用户所属的全部教研组 ID 集合
        //   用于可见性过滤教研组级 group 助手(group 来源 + group_id IN 本集合)
        //   空集合表示当前用户不在任何教研组,看不到任何教研组级助手
        CurrentGroupIDs []string

        // 仅显示激活的
        OnlyActive bool
}
