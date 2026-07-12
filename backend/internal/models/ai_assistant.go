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
//
// ──────────────────────────────────────────────────────────────────────
// share_policy(分享权限策略,本次新增)说明:
//   同事提出"助手应可分权限——有些只可用不可改/不可 fork"。
//   share_policy 是一个与 source/group_id 可见性机制【正交】的开关:
//   它不决定"谁能看到",只在"看得到之后"叠加一层"能不能带走、能不能被他人改"。
//
//   三档枚举(常量 SharePolicyOpen / UseOnly / Locked):
//     open      可用 + 可 fork:谁能看到就能复制一份带走并随意改(原 fork 行为)
//     use_only  可用,但不可 fork、不可被【非属主】编辑(防产权流失 + 防标准被改坏)
//     locked    仅属主/admin 可见可用(最严:挂在共享位但实际等于私有)
//
//   与既有 canEdit/canView/ForkAssistant 的配合(在 service 层实现):
//     - use_only/locked → 非属主非 admin 不可 fork(ForkAssistant 闸门)
//     - use_only        → 非属主非 admin 不可编辑(canEdit 收紧;属主/admin/组长仍可)
//     - locked          → 非属主非 admin 连可见性都收紧(canView 收紧)
//
//   存量与默认(Yuhan 拍板):存量助手与新建默认值均为 use_only(最保护产权)。
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

// ==================== 分享权限策略常量(share_policy,本次新增) ====================
//
// 三档枚举,详见文件头部说明。与数据库 CHECK 约束 chk_ai_assistants_share_policy 对齐。

const (
	SharePolicyOpen    = "open"     // 可用 + 可 fork(谁能看到就能复制带走并改)
	SharePolicyUseOnly = "use_only" // 可用,但不可 fork、不可被非属主编辑(默认值)
	SharePolicyLocked  = "locked"   // 仅属主/admin 可见可用(最严)
)

// ValidSharePolicies 有效的 share_policy 值
var ValidSharePolicies = []string{
	SharePolicyOpen,
	SharePolicyUseOnly,
	SharePolicyLocked,
}

// IsValidSharePolicy 校验 share_policy 是否有效
func IsValidSharePolicy(p string) bool {
	for _, v := range ValidSharePolicies {
		if v == p {
			return true
		}
	}
	return false
}

// SharePolicyLabelMap share_policy → 中文展示名(前端徽章/提示文案用)
var SharePolicyLabelMap = map[string]string{
	SharePolicyOpen:    "可复制",
	SharePolicyUseOnly: "仅可用",
	SharePolicyLocked:  "仅自己",
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

	// SharePolicy:分享权限策略(本次新增)——open / use_only / locked
	//   决定该助手能否被非属主 fork / 编辑,以及 locked 时的可见性收紧。
	//   与 source/group_id 的可见性机制正交,详见文件头部说明。
	SharePolicy string `json:"share_policy"`

	// PromptProtected:full_prompt 是否被保护性置空(本次新增,产权保护)
	//   当请求者无权查看原文(非 admin / 非属主 / 非本组组长,且助手为 use_only/locked)时,
	//   service 层会把 FullPrompt 置空并把本字段设为 true,前端据此提示"原文受保护"。
	//   注意:此字段不入库,仅在 API 响应时由 service 层动态设置。
	PromptProtected bool `json:"prompt_protected"`

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
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	AvatarEmoji string   `json:"avatar_emoji"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	SourceLabel string   `json:"source_label"` // 中文展示:系统/本校/我的
	Subject     string   `json:"subject"`
	GradeRange  string   `json:"grade_range"`
	Scenes      []string `json:"scenes"` // 已解析为字符串数组
	UseCount    int      `json:"use_count"`
	AvgScore    *float64 `json:"avg_score"`
	IsActive    bool     `json:"is_active"`
	IsDefaultHere bool   `json:"is_default_here"` // 是否在当前场景被标为默认
	CanEdit       bool   `json:"can_edit"`        // 当前用户能否编辑
	CanDelete     bool   `json:"can_delete"`      // 当前用户能否删除

	// CanFork:当前用户能否把该助手 fork 成自己的(本次新增)
	//   = 可见 && (share_policy=open || 自己是属主 || admin)
	//   前端据此显隐"复制到我的"按钮;最终拦截仍在 service.ForkAssistant
	CanFork bool `json:"can_fork"`

	// CanViewPrompt:当前用户能否查看该助手的 full_prompt 原文(本次新增,产权保护)
	//   = 与 canEdit 同款闸门(admin / 属主 / open可见者 / 本组组长对本组助手)
	//   前端据此显隐"丢给 AI 分析"按钮(分析=取原文注入对话);最终拦截在 service.GetAssistant
	CanViewPrompt bool `json:"can_view_prompt"`

	// SharePolicy:分享权限策略(本次新增,供前端展示徽章与按钮显隐)
	SharePolicy string `json:"share_policy"`

	CreatorName string `json:"creator_name"`
	SchoolName  string `json:"school_name"`

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

	// SharePolicy:分享权限策略(本次新增)——open / use_only / locked
	//   前端发布时选择;为空时 service 兜底为默认 use_only。
	//   personal 助手该字段意义不大(只有自己能看),但仍统一存储以保持一致。
	SharePolicy string `json:"share_policy"`
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

	// SharePolicy:分享权限策略(本次新增,可改)
	//   指针类型:nil = 本次不修改 share_policy(保持原值);非 nil = 改为该值。
	//   只有该助手的可编辑者(属主/组长/admin)能改,权限判定复用 canEdit。
	SharePolicy *string `json:"share_policy"`
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

	// CurrentLeadGroupIDs:当前用户担任组长(lead)的教研组 ID 集合(本次新增)
	//   用于列表层 canEditAssistant/canViewPromptAssistant 正确判定"组长可改/可看本组助手",
	//   与 service 层 canEdit 的组长逻辑对齐,修正此前列表层"组长看不到编辑按钮"的保守瑕疵。
	CurrentLeadGroupIDs []string

	// 仅显示激活的
	OnlyActive bool
}
