package models

import (
	"time"
)

// ==================== 组件库模型（对应 lesson_plan_components 表） ====================

// LessonPlanComponent 教案组件（13类统一存储）
// 所有13类组件存储在同一张表，通过 LibraryType 区分
// 支持三种注入模式：silent(静默) / recommend(推荐+确认) / on_demand(按需)
// 四层展开：DisplayLabel → DesignLogic → ExampleSnippet → FullGuide
type LessonPlanComponent struct {
	ID            string     `json:"id"`              // UUID主键
	LibraryType   string     `json:"library_type"`    // 组件库类型（13种）
	Subject       string     `json:"subject"`         // 学科（general=通用）
	GradeRange    string      `json:"grade_range"`     // 适用学段
	Tags          string     `json:"tags"`            // 标签数组JSON，匹配引擎使用
	InjectionMode string     `json:"injection_mode"`  // 注入模式：silent/recommend/on_demand
	DisplayLabel  string     `json:"display_label"`   // 第一层：展示标签（emoji+大白话）
	DesignLogic   string      `json:"design_logic"`    // 第二层：设计逻辑
	ExampleSnippet string    `json:"example_snippet"` // 第三层：参考案例片段
	FullGuide     string      `json:"full_guide"`      // 第四层：完整指引（仅AI使用）
	Content       string     `json:"content"`         // 结构化内容JSON
	Source        string     `json:"source"`          // 来源：manual/ai_extracted/user_contributed
	SourceRef     string      `json:"source_ref"`      // 来源引用
	QualityScore  float64    `json:"quality_score"`   // 综合质量分
	UsageCount    int        `json:"usage_count"`     // 被使用次数
	SelectCount   int        `json:"select_count"`    // 被选中次数
	LikeCount     int        `json:"like_count"`      // 点赞数
	DislikeCount  int        `json:"dislike_count"`   // 点踩数
	Scope         string     `json:"scope"`           // 可见范围：global/region/school/group/personal
	ScopeRefID    *string    `json:"scope_ref_id"`    // 范围引用ID
	CreatedBy     *string    `json:"created_by"`      // 创建者用户ID
	ReviewStatus  string     `json:"review_status"`   // 审核状态：draft/captured/pending/approved/rejected
	ReviewedBy    *string    `json:"reviewed_by"`     // 审核人用户ID
	ReviewedAt    *time.Time `json:"reviewed_at"`     // 审核时间
	Status        string     `json:"status"`          // 状态：active/disabled/archived
	CreatedAt     *time.Time `json:"created_at"`      // 创建时间
	UpdatedAt     *time.Time `json:"updated_at"`      // 更新时间
}

// ==================== 组件库类型常量（13种） ====================

const (
	LibCurriculumStandard  = "curriculum_standard"   // 课标与能力框架库
	LibKnowledgeGraph      = "knowledge_graph"       // 知识图谱库
	LibStudentProfile      = "student_profile"       // 学情特征库
	LibPedagogy            = "pedagogy"              // 教学法库
	LibAssessmentStrategy  = "assessment_strategy"   // 评估策略库
	LibActivityDesign      = "activity_design"       // 活动设计方案库
	LibQuestioningStrategy = "questioning_strategy"  // 提问引导策略库
	LibCrossSubject        = "cross_subject"         // 跨学科连接库
	LibTeachingTool        = "teaching_tool"         // 教学工具库
	LibScenarioMaterial    = "scenario_material"     // 素材情境库
	LibQualityRubric       = "quality_rubric"        // 质量评估标准库
	LibDesignDefect        = "design_defect"         // 常见设计缺陷库
	LibReviewRubric        = "review_rubric"         // 教案评审规则库
)

// ValidLibraryTypes 有效的组件库类型列表
var ValidLibraryTypes = []string{
	LibCurriculumStandard, LibKnowledgeGraph, LibStudentProfile,
	LibPedagogy, LibAssessmentStrategy, LibActivityDesign,
	LibQuestioningStrategy, LibCrossSubject, LibTeachingTool,
	LibScenarioMaterial, LibQualityRubric, LibDesignDefect,
	LibReviewRubric,
}

// LibraryTypeNameMap 组件库类型中文名映射
var LibraryTypeNameMap = map[string]string{
	LibCurriculumStandard:  "课标与能力框架库",
	LibKnowledgeGraph:      "知识图谱库",
	LibStudentProfile:      "学情特征库",
	LibPedagogy:            "教学法库",
	LibAssessmentStrategy:  "评估策略库",
	LibActivityDesign:      "活动设计方案库",
	LibQuestioningStrategy: "提问引导策略库",
	LibCrossSubject:        "跨学科连接库",
	LibTeachingTool:        "教学工具库",
	LibScenarioMaterial:    "素材情境库",
	LibQualityRubric:       "质量评估标准库",
	LibDesignDefect:        "常见设计缺陷库",
	LibReviewRubric:        "教案评审规则库",
}

// IsValidLibraryType 检查组件库类型是否有效
func IsValidLibraryType(lt string) bool {
	for _, v := range ValidLibraryTypes {
		if v == lt {
			return true
		}
	}
	return false
}

// ==================== 注入模式常量 ====================

const (
	InjectionSilent    = "silent"    // 静默注入：老师无感，系统自动注入
	InjectionRecommend = "recommend" // 推荐+确认：展示推荐方案，老师选择
	InjectionOnDemand  = "on_demand" // 按需调用：AI按需匹配或老师主动要求
)

// ValidInjectionModes 有效的注入模式列表
var ValidInjectionModes = []string{InjectionSilent, InjectionRecommend, InjectionOnDemand}

// IsValidInjectionMode 检查注入模式是否有效
func IsValidInjectionMode(mode string) bool {
	for _, v := range ValidInjectionModes {
		if v == mode {
			return true
		}
	}
	return false
}

// ==================== 组件可见范围常量 ====================

const (
	ScopeGlobal   = "global"   // 全局可见
	ScopeRegion   = "region"   // 区域可见
	ScopeSchool   = "school"   // 学校可见
	ScopeGroup    = "group"    // 教研组可见
	ScopePersonal = "personal" // 个人可见
)

// ValidScopes 有效的可见范围列表
var ValidScopes = []string{ScopeGlobal, ScopeRegion, ScopeSchool, ScopeGroup, ScopePersonal}

// IsValidScope 检查可见范围是否有效
func IsValidScope(scope string) bool {
	for _, v := range ValidScopes {
		if v == scope {
			return true
		}
	}
	return false
}

// ==================== 组件审核状态常量 ====================

const (
	ComponentReviewDraft    = "draft"    // 草稿
	ComponentReviewCaptured = "captured" // 对话捕获待审
	ComponentReviewPending  = "pending"  // 待审核
	ComponentReviewApproved = "approved" // 已通过
	ComponentReviewRejected = "rejected" // 已拒绝
)

// ==================== 请求结构体 ====================

// CreateComponentRequest 创建组件请求
type CreateComponentRequest struct {
	LibraryType    string  `json:"library_type"`    // 组件库类型（必填）
	Subject        string  `json:"subject"`         // 学科（可选，默认general）
	GradeRange     string  `json:"grade_range"`     // 适用学段（可选）
	Tags           string  `json:"tags"`            // 标签数组JSON（可选）
	InjectionMode  string  `json:"injection_mode"`  // 注入模式（可选，默认on_demand）
	DisplayLabel   string  `json:"display_label"`   // 展示标签（必填）
	DesignLogic    string  `json:"design_logic"`    // 设计逻辑（可选）
	ExampleSnippet string  `json:"example_snippet"` // 参考案例（可选）
	FullGuide      string  `json:"full_guide"`      // 完整指引（可选）
	Content        string  `json:"content"`         // 结构化内容JSON（可选）
	Scope          string  `json:"scope"`           // 可见范围（可选，默认global）
	ScopeRefID     *string `json:"scope_ref_id"`    // 范围引用ID（可选）
}

// UpdateComponentRequest 更新组件请求
type UpdateComponentRequest struct {
	Subject        string  `json:"subject"`         // 学科
	GradeRange     string  `json:"grade_range"`     // 适用学段
	Tags           string  `json:"tags"`            // 标签数组JSON
	InjectionMode  string  `json:"injection_mode"`  // 注入模式
	DisplayLabel   string  `json:"display_label"`   // 展示标签（必填）
	DesignLogic    string  `json:"design_logic"`    // 设计逻辑
	ExampleSnippet string  `json:"example_snippet"` // 参考案例
	FullGuide      string  `json:"full_guide"`      // 完整指引
	Content        string  `json:"content"`         // 结构化内容JSON
	Scope          string  `json:"scope"`           // 可见范围
	ScopeRefID     *string `json:"scope_ref_id"`    // 范围引用ID
	Status         string  `json:"status"`          // 状态
}

// MatchComponentsRequest 组件匹配请求
// 匹配引擎根据学科+学段+标签+注入模式查找最合适的组件
type MatchComponentsRequest struct {
	Subject       string   `json:"subject"`        // 学科（必填）
	GradeRange    string   `json:"grade_range"`    // 学段（必填）
	LibraryTypes  []string `json:"library_types"`  // 要匹配的组件库类型列表（可选，空=全部）
	InjectionMode string   `json:"injection_mode"` // 筛选注入模式（可选，空=全部）
	Tags          []string `json:"tags"`           // 标签筛选（可选）
	Limit         int      `json:"limit"`          // 每种类型返回数量上限（默认5）
}

// ReviewComponentRequest 审核组件请求（教研组长/骨干）
type ReviewComponentRequest struct {
	Decision string `json:"decision"` // 审核决策：approved/rejected
	Comment  string `json:"comment"`  // 审核意见（可选）
}

// ==================== 响应结构体 ====================

// ComponentListResponse 组件列表响应
type ComponentListResponse struct {
	Components []*ComponentListItem `json:"components"` // 组件列表
	Total      int                  `json:"total"`      // 总数
}

// ComponentListItem 组件列表单条
type ComponentListItem struct {
	ID            string     `json:"id"`
	LibraryType   string     `json:"library_type"`
	LibraryName   string     `json:"library_name"`    // 组件库类型中文名
	Subject       string     `json:"subject"`
	GradeRange    string     `json:"grade_range"`
	InjectionMode string     `json:"injection_mode"`
	DisplayLabel  string     `json:"display_label"`
	QualityScore  float64    `json:"quality_score"`
	UsageCount    int        `json:"usage_count"`
	SelectCount   int        `json:"select_count"`
	Source        string     `json:"source"`
	ReviewStatus  string     `json:"review_status"`
	Scope         string     `json:"scope"`
	Status        string     `json:"status"`
	CreatedAt     *time.Time `json:"created_at"`
}

// MatchedComponentGroup 匹配结果分组（按library_type分组）
type MatchedComponentGroup struct {
	LibraryType string                `json:"library_type"`  // 组件库类型
	LibraryName string                `json:"library_name"`  // 中文名
	Components  []*MatchedComponent   `json:"components"`    // 匹配到的组件列表
}

// MatchedComponent 匹配到的单个组件（返回给前端/AI）
type MatchedComponent struct {
	ID             string  `json:"id"`
	DisplayLabel   string  `json:"display_label"`    // 第一层：展示标签
	DesignLogic    string  `json:"design_logic"`     // 第二层：设计逻辑
	ExampleSnippet string  `json:"example_snippet"`  // 第三层：参考案例
	FullGuide      string  `json:"full_guide"`       // 第四层：完整指引（仅AI使用时返回）
	QualityScore   float64 `json:"quality_score"`    // 质量分
	UsageCount     int     `json:"usage_count"`      // 使用次数
	SelectCount    int     `json:"select_count"`     // 选中次数
	Tags           string  `json:"tags"`             // 标签
	ComponentIndex string  `json:"component_index"`  // v83: AOCI压缩索引
}

// MatchComponentsResponse 组件匹配响应
type MatchComponentsResponse struct {
	Groups []*MatchedComponentGroup `json:"groups"` // 按类型分组的匹配结果
}
