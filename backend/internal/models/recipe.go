package models

// recipe.go — 备课配方（teaching_recipes）数据模型
//
// Phase 7A 新增：
//   - TeachingRecipe 主模型（对应 teaching_recipes 表）
//   - RecipeUsageLog 使用记录（对应 recipe_usage_log 表）
//   - 请求/响应结构体
//   - 常量定义
// Phase 7B-8 修改：
//   - TeachingRecipe 新增 StagesConfig 字段（对应 stages_config JSONB 列）
// 迭代1 修改：
//   - TeachingRecipe 新增 LessonStructure + PromptMode 字段
//   - CreateRecipeRequest / UpdateRecipeRequest 新增对应字段
// 迭代2 修改：
//   - 新增 StageFlowItem 结构体（stages_config 新格式，替代 StageOverride）
//   - 新增 FlowPreset 预设流程模板结构体
//   - 新增 FlowValidationResult / FlowValidationMessage 完整性校验结构体
//   - CreateRecipeRequest / UpdateRecipeRequest 新增 StagesConfig 字段
// 迭代5 修改：
//   - StageFlowItem 新增 IsCustom 字段（标识自定义阶段）
//   - StageFlowItem 新增 StageName 字段（自定义阶段显示名称）

import "time"

// ==================== 配方主模型 ====================

// TeachingRecipe 备课配方（对应 teaching_recipes 表）
// 一个配方 = 组件组合 + 个人知识 + 共享规则 + 教案结构 + 备课模式 + 流程配置
// 老师备课前选择/创建配方，AI从一开始就拥有全局视角
// v58新增：StagesConfig 阶段覆盖配置（JSONB数组）
// 迭代1新增：LessonStructure 教案结构偏好（JSONB数组）+ PromptMode 备课模式
// 迭代2升级：StagesConfig 从 StageOverride 格式升级为 StageFlowItem 格式（兼容旧格式）
type TeachingRecipe struct {
	ID                 string     `json:"id"`                  // UUID主键
	Name               string     `json:"name"`                // 配方名称，如"七年级AI课-张老师班"
	Description        string     `json:"description"`         // 配方说明
	Subject            string     `json:"subject"`             // 学科
	GradeRange         string     `json:"grade_range"`         // 适用年级
	ComponentIDs       string     `json:"component_ids"`       // 绑定的组件ID数组（JSONB）
	StudentProfile     string     `json:"student_profile"`     // 学情记录（自由文本，持续更新）
	TeachingStyle      string     `json:"teaching_style"`      // 教学风格偏好
	SchoolRequirements string     `json:"school_requirements"` // 学校特殊要求
	CustomNotes        string     `json:"custom_notes"`        // 备课心得/自定义笔记
	CustomPrompt       string     `json:"custom_prompt"`       // 自定义提示词（高级用户）
	Scope              string     `json:"scope"`               // 可见范围：personal/group/school
	ScopeRefID         *string    `json:"scope_ref_id"`        // 范围引用ID（教研组ID/学校ID）
	AuthorID           string     `json:"author_id"`           // 创建者用户ID
	ForkCount          int        `json:"fork_count"`          // 被Fork次数
	ForkedFrom         *string    `json:"forked_from"`         // Fork来源配方ID
	UseCount           int        `json:"use_count"`           // 使用次数
	Version            int        `json:"version"`             // 版本号
	Status             string     `json:"status"`              // active/archived
	StagesConfig       string     `json:"stages_config"`       // 流程配置（JSONB字符串，迭代2升级为StageFlowItem数组）
	LessonStructure    string     `json:"lesson_structure"`    // 迭代1：教案结构偏好（JSONB字符串，存储LessonStructureBlock数组）
	PromptMode         string     `json:"prompt_mode"`         // 迭代1：备课模式 guided/efficient/per_stage
	CreatedAt          *time.Time `json:"created_at"`
	UpdatedAt          *time.Time `json:"updated_at"`
}

// RecipeUsageLog 配方使用记录（对应 recipe_usage_log 表）
type RecipeUsageLog struct {
	ID            string     `json:"id"`
	RecipeID      string     `json:"recipe_id"`
	LessonPlanID  *string    `json:"lesson_plan_id"`
	UserID        string     `json:"user_id"`
	AIReviewScore *float64   `json:"ai_review_score"`
	CreatedAt     *time.Time `json:"created_at"`
}

// ==================== 配方可见范围常量 ====================

const (
	RecipeScopePersonal = "personal" // 个人可见
	RecipeScopeGroup    = "group"    // 教研组共享
	RecipeScopeSchool   = "school"   // 全校共享
)

// RecipeScopeNameMap 可见范围中文映射
var RecipeScopeNameMap = map[string]string{
	RecipeScopePersonal: "个人",
	RecipeScopeGroup:    "教研组",
	RecipeScopeSchool:   "全校",
}

// ==================== 备课模式常量（迭代1新增）====================

const (
	PromptModeGuided    = "guided"    // 引导版（新手/成长型，多轮对话）
	PromptModeEfficient = "efficient" // 高效版（成熟型，快速出稿）
	PromptModePerStage  = "per_stage" // 逐阶段选择（迭代2启用前端支持）
)

// PromptModeNameMap 备课模式中文映射
var PromptModeNameMap = map[string]string{
	PromptModeGuided:    "引导版",
	PromptModeEfficient: "高效版",
	PromptModePerStage:  "逐阶段",
}

// ==================== 教案结构定义（迭代1新增）====================

// LessonStructureBlock 教案结构板块（存入 teaching_recipes.lesson_structure JSONB 数组）
// 老师通过表格编辑器定义教案应包含哪些板块及要求
type LessonStructureBlock struct {
	Name        string                      `json:"name"`                   // 板块名称，如"教学目标"
	Required    bool                        `json:"required"`               // 是否必填
	Requirement string                      `json:"requirement"`            // 老师的要求（自然语言）
	Order       int                         `json:"order"`                  // 排序序号
	SubSections []LessonStructureSubSection `json:"sub_sections,omitempty"` // 子环节（仅教学过程有）
}

// LessonStructureSubSection 教学过程子环节（教学过程板块专属）
type LessonStructureSubSection struct {
	Name              string `json:"name"`              // 环节名称，如"导入"
	Duration          int    `json:"duration"`           // 时长（分钟）
	Goal              string `json:"goal"`               // 设计目标
	OutputRequirement string `json:"output_requirement"` // 输出要求
}

// ==================== 流程配置定义（迭代2新增，迭代5扩展）====================

// StageFlowItem 流程配置项（存入 teaching_recipes.stages_config JSONB 数组）
// 迭代2新格式：比 StageOverride 更简洁直观，老师在流程搭建器中拖拽配置
// 每一项对应系统默认阶段中的一个，通过 enabled 控制是否启用
// 迭代5扩展：新增 IsCustom + StageName 字段，支持自定义阶段与系统阶段混排
type StageFlowItem struct {
	StageCode  string `json:"stage_code"`            // 阶段代码（必填），系统阶段或自定义阶段代码
	Enabled    bool   `json:"enabled"`               // 是否启用此阶段
	Order      int    `json:"order"`                 // 排序序号（1,2,3...前端拖拽调整）
	PromptMode string `json:"prompt_mode,omitempty"` // 本阶段的对话模式覆盖（per_stage模式下有值）
	IsCustom   bool   `json:"is_custom,omitempty"`   // 迭代5：是否为自定义阶段（true=配方自定义，false/空=系统阶段）
	StageName  string `json:"stage_name,omitempty"`  // 迭代5：自定义阶段显示名称（系统阶段不需要，从默认定义获取）
}

// ==================== 流程预设模板（迭代2新增）====================

// FlowPreset 预设流程模板（后端硬编码，前端展示供老师快速选择）
type FlowPreset struct {
	Key         string          `json:"key"`          // 模板唯一标识
	Name        string          `json:"name"`         // 模板名称，如"完整引导"
	Description string          `json:"description"`  // 说明
	Duration    string          `json:"duration"`     // 预估时间，如"15-25分钟"
	Icon        string          `json:"icon"`         // 图标emoji
	Stages      []StageFlowItem `json:"stages"`       // 预设的阶段配置
	PromptMode  string          `json:"prompt_mode"`  // 推荐的备课模式
}

// ==================== 流程完整性校验（迭代2新增）====================

// FlowValidationResult 流程完整性校验结果
type FlowValidationResult struct {
	Valid    bool                    `json:"valid"`    // 是否通过校验（无阻断错误）
	Messages []FlowValidationMessage `json:"messages"` // 校验消息列表（可能同时有多条）
}

// FlowValidationMessage 单条校验消息
type FlowValidationMessage struct {
	Level   string `json:"level"`   // 级别：info / warning / error
	Code    string `json:"code"`    // 消息代码（前端国际化用）
	Message string `json:"message"` // 中文消息文本
}

// 校验消息级别常量
const (
	FlowMsgInfo    = "info"    // 信息提示（蓝色，仅展示）
	FlowMsgWarning = "warning" // 警告提示（黄色，需确认但不阻断）
	FlowMsgError   = "error"   // 阻断错误（红色，必须修正）
)

// ==================== 流程校验请求/响应（迭代2新增）====================

// ValidateFlowRequest 校验流程完整性请求
type ValidateFlowRequest struct {
	Stages []StageFlowItem `json:"stages"` // 要校验的阶段配置
}

// ==================== 请求结构体 ====================

// CreateRecipeRequest 创建配方请求
// 迭代1新增：LessonStructure + PromptMode
// 迭代2新增：StagesConfig
type CreateRecipeRequest struct {
	Name               string   `json:"name"`                // 配方名称（必填）
	Description        string   `json:"description"`         // 配方说明
	Subject            string   `json:"subject"`             // 学科（必填）
	GradeRange         string   `json:"grade_range"`         // 年级（必填）
	ComponentIDs       []string `json:"component_ids"`       // 绑定组件ID列表
	StudentProfile     string   `json:"student_profile"`     // 学情记录
	TeachingStyle      string   `json:"teaching_style"`      // 教学风格
	SchoolRequirements string   `json:"school_requirements"` // 学校要求
	CustomNotes        string   `json:"custom_notes"`        // 备课心得
	CustomPrompt       string   `json:"custom_prompt"`       // 自定义提示词
	LessonStructure    string   `json:"lesson_structure"`    // 迭代1：教案结构JSON字符串
	PromptMode         string   `json:"prompt_mode"`         // 迭代1：备课模式 guided/efficient
	StagesConfig       string   `json:"stages_config"`       // 迭代2：流程配置JSON字符串（StageFlowItem数组）
}

// UpdateRecipeRequest 更新配方请求
// 迭代1新增：LessonStructure + PromptMode
// 迭代2新增：StagesConfig
type UpdateRecipeRequest struct {
	Name               string   `json:"name"`                // 配方名称（必填）
	Description        string   `json:"description"`         // 配方说明
	ComponentIDs       []string `json:"component_ids"`       // 绑定组件ID列表
	StudentProfile     string   `json:"student_profile"`     // 学情记录
	TeachingStyle      string   `json:"teaching_style"`      // 教学风格
	SchoolRequirements string   `json:"school_requirements"` // 学校要求
	CustomNotes        string   `json:"custom_notes"`        // 备课心得
	CustomPrompt       string   `json:"custom_prompt"`       // 自定义提示词
	LessonStructure    string   `json:"lesson_structure"`    // 迭代1：教案结构JSON字符串
	PromptMode         string   `json:"prompt_mode"`         // 迭代1：备课模式 guided/efficient
	StagesConfig       string   `json:"stages_config"`       // 迭代2：流程配置JSON字符串（StageFlowItem数组）
}

// UpdateStudentProfileRequest 单独更新学情记录请求
type UpdateStudentProfileRequest struct {
	StudentProfile string `json:"student_profile"` // 学情记录内容
}

// ShareRecipeRequest 共享配方请求
type ShareRecipeRequest struct {
	Scope      string `json:"scope"`        // group / school
	ScopeRefID string `json:"scope_ref_id"` // 教研组ID / 学校ID
}

// RecipeRecommendRequest 智能推荐配方请求
type RecipeRecommendRequest struct {
	Subject    string `json:"subject"`     // 学科（必填）
	GradeRange string `json:"grade_range"` // 年级（必填）
}

// ==================== 响应结构体 ====================

// RecipeListResponse 配方列表响应
type RecipeListResponse struct {
	Recipes []*RecipeListItem `json:"recipes"` // 配方列表
	Total   int               `json:"total"`   // 总数
}

// RecipeListItem 配方列表单条（不含大文本字段）
type RecipeListItem struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Subject        string     `json:"subject"`
	GradeRange     string     `json:"grade_range"`
	ComponentCount int        `json:"component_count"` // 绑定组件数量
	Scope          string     `json:"scope"`
	ScopeName      string     `json:"scope_name"` // 中文名
	AuthorID       string     `json:"author_id"`
	AuthorName     string     `json:"author_name"` // 创建者显示名
	ForkCount      int        `json:"fork_count"`
	UseCount       int        `json:"use_count"`
	Version        int        `json:"version"`
	ForkedFrom     *string    `json:"forked_from"`
	Status         string     `json:"status"`
	PromptMode     string     `json:"prompt_mode"`    // 迭代1：备课模式
	StagesConfig   string     `json:"stages_config"`  // 迭代2：流程配置（供前端列表展示阶段数量）
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

// RecipeDetailResponse 配方详情响应（含全部字段+关联组件摘要）
type RecipeDetailResponse struct {
	TeachingRecipe                         // 嵌入全部字段（含迭代1的LessonStructure+PromptMode、迭代2的StagesConfig）
	ComponentCount int                     `json:"component_count"` // 绑定组件数量
	Components     []*RecipeComponentBrief `json:"components"`      // 组件摘要列表
	AuthorName     string                  `json:"author_name"`     // 创建者显示名
	ScopeName      string                  `json:"scope_name"`      // 范围中文名
}

// RecipeComponentBrief 配方中绑定的组件摘要
type RecipeComponentBrief struct {
	ID           string  `json:"id"`
	LibraryType  string  `json:"library_type"`
	LibraryName  string  `json:"library_name"` // 中文名
	DisplayLabel string  `json:"display_label"`
	QualityScore float64 `json:"quality_score"`
	Status       string  `json:"status"` // active/archived（用于检测失效组件）
}

// RecipeContextPreview 配方上下文预览（展示注入给AI的完整文本）
type RecipeContextPreview struct {
	RecipeID      string `json:"recipe_id"`
	RecipeName    string `json:"recipe_name"`
	ContextText   string `json:"context_text"`   // 完整的提示词上下文文本
	TokenEstimate int    `json:"token_estimate"` // 估算token数（按字符/2粗估）
}

// ==================== 旧格式兼容（Phase 7B原始格式，迭代2保留供MergeStages兼容）====================

// StageOverride 配方中的阶段覆盖配置（v58旧格式）
// 存入 teaching_recipes.stages_config JSONB 数组
// 迭代2已不推荐使用，MergeStages检测到旧格式会自动走旧逻辑
type StageOverride struct {
	StageCode      string   `json:"stage_code"`                // 阶段代码（必填）
	Action         string   `json:"action"`                    // 操作类型：override/insert_after/skip/replace_prompt
	InsertAfter    string   `json:"insert_after,omitempty"`    // insert_after时指定插入位置
	StageName      string   `json:"stage_name,omitempty"`      // 覆盖阶段名称
	AIRole         string   `json:"ai_role,omitempty"`         // 覆盖AI角色
	SystemPrompt   string   `json:"system_prompt,omitempty"`   // 覆盖系统提示词
	OutputFormat   string   `json:"output_format,omitempty"`   // 覆盖产出物格式
	ComponentTypes []string `json:"component_types,omitempty"` // 覆盖组件映射
	GateMode       string   `json:"gate_mode,omitempty"`       // 覆盖门控模式
	Skippable      *bool    `json:"skippable,omitempty"`       // 覆盖是否可跳过
}

// 配方阶段覆盖操作类型常量（旧格式）
const (
	StageActionOverride      = "override"       // 覆盖同code的默认阶段的部分字段
	StageActionInsertAfter   = "insert_after"   // 在指定阶段后插入新阶段
	StageActionSkip          = "skip"            // 跳过该默认阶段
	StageActionReplacePrompt = "replace_prompt"  // 仅替换提示词
)
