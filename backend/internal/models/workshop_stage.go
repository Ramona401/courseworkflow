package models

// workshop_stage.go — 阶段化备课工坊数据模型
//
// Phase 7B 新增：
//   - WorkshopStage         主模型（对应 workshop_stages 表）
//   - WorkshopStageOutput   阶段产出物模型（对应 workshop_stage_outputs 表）
//   - StageConfigSnapshot   教案阶段配置快照（存入 lesson_plans.stage_config JSONB）
//   - 请求/响应结构体
//   - 常量定义
// 迭代1 修改：
//   - WorkshopStage 新增 PromptVariants 字段（对应 prompt_variants JSONB 列）
//   - UpdateStageRequest 新增 PromptVariants 字段
// 迭代2 修改：
//   - StageConfigSnapshot 新增 PromptModeOverride 字段（per_stage模式下的阶段级对话模式覆盖）
//   - 新增阶段属性常量表（哪些阶段可移除/可调序/有依赖）
// 迭代5 修改：
//   - 新增 CreateCustomStageRequest / UpdateCustomStageRequest 自定义阶段请求结构体
//   - 新增 CustomStageResponse 自定义阶段响应结构体
//   - StageConfigSnapshot 新增 IsCustom 字段标识自定义阶段

import "time"

// ==================== 阶段定义主模型 ====================

// WorkshopStage 阶段定义（对应 workshop_stages 表）
// source='system' 为系统预设5阶段，source='recipe' 为配方自定义阶段
// 迭代1新增：PromptVariants 存储引导版/高效版对话策略变体
type WorkshopStage struct {
	ID             string     `json:"id"`              // UUID主键
	StageCode      string     `json:"stage_code"`      // 阶段代码：analyze/design/write/review/revise/自定义
	StageName      string     `json:"stage_name"`      // 显示名称
	StageOrder     int        `json:"stage_order"`     // 排序序号（1,2,3...）
	Source         string     `json:"source"`          // 归属：system/recipe
	RecipeID       *string    `json:"recipe_id"`       // 配方ID（source=recipe时有值）
	AIRole         string     `json:"ai_role"`         // AI角色名称
	SystemPrompt   string     `json:"system_prompt"`   // 本阶段系统提示词（迭代1后仅存共享段：角色+工作目标）
	PromptVariants string     `json:"prompt_variants"` // 迭代1新增：对话策略变体（JSONB字符串 {"guided":"...","efficient":"..."}）
	OutputFormat   string     `json:"output_format"`   // 产出物结构定义（JSONB字符串）
	ComponentTypes string     `json:"component_types"` // 本阶段注入的library_type列表（JSONB字符串）
	GateMode       string     `json:"gate_mode"`       // 门控模式：suggest/force/auto
	Skippable      bool       `json:"skippable"`       // 是否可跳过
	Status         string     `json:"status"`          // active/disabled
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

// ==================== 阶段产出物模型 ====================

// WorkshopStageOutput 阶段产出物（对应 workshop_stage_outputs 表）
// 每次备课中每个阶段一条记录，存储结构化产出+自然语言补充+对话快照
type WorkshopStageOutput struct {
	ID                   string     `json:"id"`                    // UUID主键
	LessonPlanID         string     `json:"lesson_plan_id"`        // 关联教案ID
	StageCode            string     `json:"stage_code"`            // 阶段代码
	StageOrder           int        `json:"stage_order"`           // 执行时的阶段序号
	StructuredOutput     string     `json:"structured_output"`     // 结构化产出（JSONB字符串）
	NarrativeOutput      string     `json:"narrative_output"`      // 自然语言补充说明
	ConversationSnapshot string     `json:"conversation_snapshot"` // 本阶段对话快照（JSONB字符串）
	ModelUsed            string     `json:"model_used"`            // AI模型名
	TokensUsed           int        `json:"tokens_used"`           // token消耗
	Status               string     `json:"status"`                // in_progress/completed/skipped
	CompletedAt          *time.Time `json:"completed_at"`          // 完成时间
	CreatedAt            *time.Time `json:"created_at"`
	UpdatedAt            *time.Time `json:"updated_at"`
}

// ==================== 阶段来源常量 ====================

const (
	StageSourceSystem = "system" // 系统预设阶段
	StageSourceRecipe = "recipe" // 配方自定义阶段
)

// ==================== 门控模式常量 ====================

const (
	StageGateSuggest = "suggest" // 建议确认（AI建议进入下一阶段，老师可选择继续或跳过）
	StageGateForce   = "force"   // 强制确认（必须确认才能进入下一阶段）
	StageGateAuto    = "auto"    // 自动进入（无需确认，阶段完成后自动切换）
)

// ==================== 产出物状态常量 ====================

const (
	StageOutputInProgress = "in_progress" // 进行中
	StageOutputCompleted  = "completed"   // 已完成
	StageOutputSkipped    = "skipped"     // 已跳过
)

// ==================== 阶段属性规则表（迭代2新增）====================

// StageFlowRule 单个系统阶段的流程属性规则
type StageFlowRule struct {
	StageCode   string // 阶段代码
	StageName   string // 中文名
	Removable   bool   // 可移除（false=不可移除，即必须保留）
	Reorderable bool   // 可调序
	MustBeLast  bool   // 必须在最后（仅revise为true）
	MustAfter   string // 必须在某阶段之后（空=无依赖）；如review必须在write之后
}

// SystemStageFlowRules 系统5个默认阶段的流程属性规则表
// PRD §5.1 对照：
//   analyze: 可移除, 可调序, 无前置依赖
//   design:  可移除, 可调序, 建议在分析后（warning级）
//   write:   不可移除, 可调序, 建议在设计后（warning级）
//   review:  可移除, 可调序, 必须在write后（error级）
//   revise:  不可移除, 不可调序（固定最后）, 固定最后
var SystemStageFlowRules = map[string]StageFlowRule{
	"analyze": {StageCode: "analyze", StageName: "教学分析", Removable: true, Reorderable: true, MustBeLast: false, MustAfter: ""},
	"design":  {StageCode: "design", StageName: "教学设计", Removable: true, Reorderable: true, MustBeLast: false, MustAfter: ""},
	"write":   {StageCode: "write", StageName: "教案撰写", Removable: false, Reorderable: true, MustBeLast: false, MustAfter: ""},
	"review":  {StageCode: "review", StageName: "AI评审", Removable: true, Reorderable: true, MustBeLast: false, MustAfter: "write"},
	"revise":  {StageCode: "revise", StageName: "修订定稿", Removable: false, Reorderable: false, MustBeLast: true, MustAfter: ""},
}

// ==================== 教案阶段配置快照 ====================

// StageConfigSnapshot 教案启动时写入的阶段配置快照
// 存入 lesson_plans.stage_config JSONB 数组
// 从系统默认+配方覆盖合并后生成，记录本次备课实际使用的阶段列表
// 迭代2新增：PromptModeOverride（per_stage模式下的阶段级对话模式覆盖）
// 迭代5新增：IsCustom（标识自定义阶段，LoadStagePromptContext据此加载配方阶段提示词）
type StageConfigSnapshot struct {
	StageCode          string `json:"stage_code"`                     // 阶段代码
	StageName          string `json:"stage_name"`                     // 显示名称
	StageOrder         int    `json:"stage_order"`                    // 排序序号
	AIRole             string `json:"ai_role"`                        // AI角色
	GateMode           string `json:"gate_mode"`                      // 门控模式
	Skippable          bool   `json:"skippable"`                      // 是否可跳过
	PromptModeOverride string `json:"prompt_mode_override,omitempty"` // 迭代2新增：阶段级对话模式覆盖（per_stage用）
	IsCustom           bool   `json:"is_custom,omitempty"`            // 迭代5新增：是否为自定义阶段
}

// ==================== 迭代5新增：自定义阶段请求结构体 ====================

// CreateCustomStageRequest 创建配方自定义阶段的请求
// 老师在配方编辑器的流程搭建器中点击"添加自定义阶段"时提交
type CreateCustomStageRequest struct {
	StageCode      string `json:"stage_code"`      // 阶段代码（必填，英文标识符，配方内唯一）
	StageName      string `json:"stage_name"`      // 显示名称（必填，中文）
	AIRole         string `json:"ai_role"`         // AI角色名称（必填）
	SystemPrompt   string `json:"system_prompt"`   // 系统提示词（共享段）
	PromptVariants string `json:"prompt_variants"` // 对话策略变体 JSON {"guided":"...","efficient":"..."}
	OutputFormat   string `json:"output_format"`   // 产出物格式定义（JSONB字符串）
	ComponentTypes string `json:"component_types"` // 注入的组件类型列表（JSONB字符串）
	GateMode       string `json:"gate_mode"`       // 门控模式：suggest/force/auto（默认suggest）
	Skippable      bool   `json:"skippable"`       // 是否可跳过（默认true）
}

// UpdateCustomStageRequest 更新配方自定义阶段的请求
type UpdateCustomStageRequest struct {
	StageName      string `json:"stage_name"`      // 显示名称
	AIRole         string `json:"ai_role"`         // AI角色名称
	SystemPrompt   string `json:"system_prompt"`   // 系统提示词
	PromptVariants string `json:"prompt_variants"` // 对话策略变体 JSON
	OutputFormat   string `json:"output_format"`   // 产出物格式定义
	ComponentTypes string `json:"component_types"` // 注入的组件类型列表
	GateMode       string `json:"gate_mode"`       // 门控模式
	Skippable      bool   `json:"skippable"`       // 是否可跳过
}

// CustomStageResponse 自定义阶段响应（精简版，供前端流程搭建器使用）
type CustomStageResponse struct {
	StageCode  string `json:"stage_code"`  // 阶段代码
	StageName  string `json:"stage_name"`  // 显示名称
	AIRole     string `json:"ai_role"`     // AI角色名称
	GateMode   string `json:"gate_mode"`   // 门控模式
	Skippable  bool   `json:"skippable"`   // 是否可跳过
	HasPrompt  bool   `json:"has_prompt"`  // 是否已配置提示词
}

// ==================== 请求结构体 ====================

// AdvanceStageRequest 进入下一阶段请求
type AdvanceStageRequest struct {
	TargetStageCode string `json:"target_stage_code"` // 目标阶段代码（可选，空则自动进入下一个）
}

// SkipStageRequest 跳过当前阶段请求
type SkipStageRequest struct {
	TargetStageCode string `json:"target_stage_code"` // 跳到哪个阶段（可选，空则跳到下一个）
}

// ==================== 响应结构体 ====================

// StageStatusResponse 阶段状态响应（含当前阶段+全部阶段进度）
type StageStatusResponse struct {
	CurrentStage string               `json:"current_stage"` // 当前阶段代码
	TotalStages  int                  `json:"total_stages"`  // 阶段总数
	Stages       []*StageProgressItem `json:"stages"`        // 各阶段进度
}

// StageProgressItem 单个阶段的进度信息
type StageProgressItem struct {
	StageCode   string     `json:"stage_code"`             // 阶段代码
	StageName   string     `json:"stage_name"`             // 显示名称
	StageOrder  int        `json:"stage_order"`            // 排序序号
	AIRole      string     `json:"ai_role"`                // AI角色名称
	GateMode    string     `json:"gate_mode"`              // 门控模式
	Skippable   bool       `json:"skippable"`              // 是否可跳过
	Status      string     `json:"status"`                 // in_progress/completed/skipped/pending
	HasOutput   bool       `json:"has_output"`             // 是否已有产出物
	CompletedAt *time.Time `json:"completed_at,omitempty"` // 完成时间
	IsCustom    bool       `json:"is_custom,omitempty"`    // 迭代5新增：是否为自定义阶段
}

// StageOutputResponse 阶段产出物响应
type StageOutputResponse struct {
	StageCode        string `json:"stage_code"`        // 阶段代码
	StageName        string `json:"stage_name"`        // 阶段名称
	StructuredOutput string `json:"structured_output"` // 结构化产出（JSON字符串）
	NarrativeOutput  string `json:"narrative_output"`  // 自然语言补充
	Status           string `json:"status"`            // 状态
	ModelUsed        string `json:"model_used"`        // 使用的AI模型
	TokensUsed       int    `json:"tokens_used"`       // token消耗
}

// DefaultStagesResponse 系统默认阶段列表响应
type DefaultStagesResponse struct {
	Stages []*DefaultStageItem `json:"stages"` // 默认阶段列表
}

// DefaultStageItem 默认阶段单条
type DefaultStageItem struct {
	StageCode      string `json:"stage_code"`      // 阶段代码
	StageName      string `json:"stage_name"`      // 显示名称
	StageOrder     int    `json:"stage_order"`     // 排序序号
	AIRole         string `json:"ai_role"`         // AI角色
	GateMode       string `json:"gate_mode"`       // 门控模式
	Skippable      bool   `json:"skippable"`       // 是否可跳过
	ComponentTypes string `json:"component_types"` // 组件类型列表（JSON字符串）
}

// ==================== SSE事件类型扩展（Phase 7B新增）====================

const (
	LPSSEStageStarted  LPSSEEventType = "stage_started"  // 进入新阶段
	LPSSEStageComplete LPSSEEventType = "stage_complete"  // AI建议完成当前阶段
	LPSSEStageOutput   LPSSEEventType = "stage_output"    // 阶段产出物已生成
)

// StageEventData 阶段相关SSE事件的数据载荷
type StageEventData struct {
	StageCode   string `json:"stage_code"`           // 当前阶段代码
	StageName   string `json:"stage_name"`           // 当前阶段名称
	StageOrder  int    `json:"stage_order"`          // 当前阶段序号
	TotalStages int    `json:"total_stages"`         // 阶段总数
	NextStage   string `json:"next_stage,omitempty"` // 下一阶段代码（complete事件用）
	CanSkip     bool   `json:"can_skip,omitempty"`   // 是否可跳过（complete事件用）
}

// ==================== 阶段管理请求结构体（Admin管理页面）====================

// UpdateStageRequest Admin更新系统阶段的请求
// 仅用于 source='system' 的阶段，通过 stage_code 定位
// 迭代1新增：PromptVariants 字段
type UpdateStageRequest struct {
	StageName      string `json:"stage_name"`      // 阶段显示名称
	AIRole         string `json:"ai_role"`         // AI角色名称
	SystemPrompt   string `json:"system_prompt"`   // 系统提示词（迭代1后为共享段）
	PromptVariants string `json:"prompt_variants"` // 迭代1新增：对话策略变体JSON {"guided":"...","efficient":"..."}
	OutputFormat   string `json:"output_format"`   // 产出物格式定义（JSONB字符串）
	ComponentTypes string `json:"component_types"` // 注入的组件类型列表（JSONB字符串）
	GateMode       string `json:"gate_mode"`       // 门控模式：suggest/force/auto
	Skippable      bool   `json:"skippable"`       // 是否可跳过
	Status         string `json:"status"`          // active/disabled
}

// AdminStageListResponse 管理页面阶段列表响应（含全部字段）
type AdminStageListResponse struct {
	Stages []*WorkshopStage `json:"stages"` // 全部系统阶段（含disabled）
}

// ==================== 迭代12新增：阶段推荐组件响应 ====================

// StageRecommendedComponentsResponse 阶段推荐组件列表响应
// GET /api/v1/lesson-plans/plans/{id}/stages/{code}/recommended-components
type StageRecommendedComponentsResponse struct {
	StageCode  string                       `json:"stage_code"`  // 阶段代码
	StageName  string                       `json:"stage_name"`  // 阶段名称
	Components []*RecommendedComponentItem  `json:"components"`  // 推荐组件列表
}

// RecommendedComponentItem 推荐组件条目
type RecommendedComponentItem struct {
	ID           string  `json:"id"`            // 组件UUID
	LibraryType  string  `json:"library_type"`  // 组件库类型
	LibraryName  string  `json:"library_name"`  // 类型中文名
	DisplayLabel string  `json:"display_label"` // 展示标签
	DesignLogic    string  `json:"design_logic"`    // 设计逻辑简述
	FullGuide      string  `json:"full_guide"`      // 完整指引（v78新增）
	ExampleSnippet string  `json:"example_snippet"` // 示例片段（v78新增）
	QualityScore   float64 `json:"quality_score"`   // 质量分
	Source         string  `json:"source"`          // 来源：recipe=配方已选 / auto=自动匹配
}

// AdvanceStageWithComponentsRequest 进入下一阶段请求（带用户选中组件）
// 迭代12新增：前端阶段过渡弹窗选中组件后使用此请求
type AdvanceStageWithComponentsRequest struct {
	TargetStageCode      string   `json:"target_stage_code"`       // 目标阶段代码（可选）
	SelectedComponentIDs []string `json:"selected_component_ids"`  // 用户选中的组件ID列表
}

// ResetStageRequest 重启指定阶段请求（迭代12新增）
// 清空该阶段及之后阶段的产出物，将当前阶段设回目标阶段，重新触发AI开场白
type ResetStageRequest struct {
	TargetStageCode string `json:"target_stage_code"` // 要重启到的阶段代码（必填）
}

