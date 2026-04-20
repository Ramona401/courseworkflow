package models

// ai_assistant.go — AI 助手数据模型
//
// TE-DNA 3.0 P0 核心实体:统一存储系统/教研员/个人三种来源的 AI 助手
// 三层架构:老师只看助手 → 助手通过 AOCI 调用组件知识库 → 组件库退居幕后
//
// 对应数据库表:ai_assistants(v110 新增)

import "time"

// ==================== 来源常量 ====================

const (
	AssistantSourceSystem   = "system"   // 系统预置(admin 管理)
	AssistantSourceGroup    = "group"    // 教研员本校(senior_operator 管理)
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
	GroupID        *string `json:"group_id"`        // 预留:未来按教研组可见

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
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	AvatarEmoji   string     `json:"avatar_emoji"`
	Description   string     `json:"description"`
	Source        string     `json:"source"`
	SourceLabel   string     `json:"source_label"` // 中文展示:系统/教研员/个人
	Subject       string     `json:"subject"`
	GradeRange    string     `json:"grade_range"`
	Scenes        []string   `json:"scenes"` // 已解析为字符串数组
	UseCount      int        `json:"use_count"`
	AvgScore      *float64   `json:"avg_score"`
	IsActive      bool       `json:"is_active"`
	IsDefaultHere bool       `json:"is_default_here"` // 是否在当前场景被标为默认
	CanEdit       bool       `json:"can_edit"`        // 当前用户能否编辑
	CanDelete     bool       `json:"can_delete"`      // 当前用户能否删除
	CreatorName   string     `json:"creator_name"`
	SchoolName    string     `json:"school_name"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// SourceLabelMap source → 中文展示名
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
	CurrentSchoolID string // 当前用户所属学校 ID(用于过滤 group,可为空)

	// 仅显示激活的
	OnlyActive bool
}
