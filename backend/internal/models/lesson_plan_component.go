package models

import (
	"time"
)

// ==================== 组件库模型（对应 lesson_plan_components 表） ====================

// LessonPlanComponent 教案组件（13类统一存储）。
//
// 所有13类组件存储在同一张表，通过LibraryType区分。
// 支持三种注入模式：silent（静默）/recommend（推荐确认）/on_demand（按需）。
// 四层展开：DisplayLabel → DesignLogic → ExampleSnippet → FullGuide。
//
// EducationDomain是组件资源创建时固化的教育域快照：
//   - k12 / vocational / adult：具体教学域资源；
//   - common：三个具体教学域都可以使用的公共资源；
//   - mixed绝不能写入组件资源。
//
// 具体教案运行时必须使用lesson_plans.education_domain快照进行匹配，
// 不能使用当前登录者后来变化的学校或组织教育域。
type LessonPlanComponent struct {
	ID              string `json:"id"`               // UUID主键
	EducationDomain string `json:"education_domain"` // 资源教育域：k12/vocational/adult/common

	LibraryType    string `json:"library_type"`    // 组件库类型（13种）
	Subject        string `json:"subject"`         // 课程/学科（general=通用）
	GradeRange     string `json:"grade_range"`     // 适用年级、学段或学习层级
	Tags           string `json:"tags"`            // 标签数组JSON
	InjectionMode  string `json:"injection_mode"`  // silent/recommend/on_demand
	DisplayLabel   string `json:"display_label"`   // 第一层：展示标签
	DesignLogic    string `json:"design_logic"`    // 第二层：设计逻辑
	ExampleSnippet string `json:"example_snippet"` // 第三层：参考案例片段
	FullGuide      string `json:"full_guide"`      // 第四层：完整指引
	Content        string `json:"content"`         // 结构化内容JSON

	Source    string `json:"source"`     // manual/ai_extracted/user_contributed
	SourceRef string `json:"source_ref"` // 来源引用

	QualityScore float64 `json:"quality_score"`
	UsageCount   int     `json:"usage_count"`
	SelectCount  int     `json:"select_count"`
	LikeCount    int     `json:"like_count"`
	DislikeCount int     `json:"dislike_count"`

	Scope      string  `json:"scope"`        // global/region/school/group/personal
	ScopeRefID *string `json:"scope_ref_id"` // 范围引用ID
	CreatedBy  *string `json:"created_by"`   // 创建者用户ID

	ReviewStatus string     `json:"review_status"`
	ReviewedBy   *string    `json:"reviewed_by"`
	ReviewedAt   *time.Time `json:"reviewed_at"`

	Status    string     `json:"status"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// ==================== 组件库类型常量（13种） ====================

const (
	LibCurriculumStandard  = "curriculum_standard"
	LibKnowledgeGraph      = "knowledge_graph"
	LibStudentProfile      = "student_profile"
	LibPedagogy            = "pedagogy"
	LibAssessmentStrategy  = "assessment_strategy"
	LibActivityDesign      = "activity_design"
	LibQuestioningStrategy = "questioning_strategy"
	LibCrossSubject        = "cross_subject"
	LibTeachingTool        = "teaching_tool"
	LibScenarioMaterial    = "scenario_material"
	LibQualityRubric       = "quality_rubric"
	LibDesignDefect        = "design_defect"
	LibReviewRubric        = "review_rubric"
)

// ValidLibraryTypes 有效的组件库类型列表。
var ValidLibraryTypes = []string{
	LibCurriculumStandard,
	LibKnowledgeGraph,
	LibStudentProfile,
	LibPedagogy,
	LibAssessmentStrategy,
	LibActivityDesign,
	LibQuestioningStrategy,
	LibCrossSubject,
	LibTeachingTool,
	LibScenarioMaterial,
	LibQualityRubric,
	LibDesignDefect,
	LibReviewRubric,
}

// LibraryTypeNameMap 组件库类型中文名。
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

// IsValidLibraryType 检查组件库类型是否有效。
func IsValidLibraryType(libraryType string) bool {
	for _, item := range ValidLibraryTypes {
		if item == libraryType {
			return true
		}
	}

	return false
}

// ==================== 注入模式常量 ====================

const (
	InjectionSilent    = "silent"
	InjectionRecommend = "recommend"
	InjectionOnDemand  = "on_demand"
)

// ValidInjectionModes 有效注入模式。
var ValidInjectionModes = []string{
	InjectionSilent,
	InjectionRecommend,
	InjectionOnDemand,
}

// IsValidInjectionMode 检查注入模式是否有效。
func IsValidInjectionMode(mode string) bool {
	for _, item := range ValidInjectionModes {
		if item == mode {
			return true
		}
	}

	return false
}

// ==================== 组件可见范围常量 ====================

const (
	ScopeGlobal   = "global"
	ScopeRegion   = "region"
	ScopeSchool   = "school"
	ScopeGroup    = "group"
	ScopePersonal = "personal"
)

// ValidScopes 有效可见范围。
var ValidScopes = []string{
	ScopeGlobal,
	ScopeRegion,
	ScopeSchool,
	ScopeGroup,
	ScopePersonal,
}

// IsValidScope 检查可见范围是否有效。
func IsValidScope(scope string) bool {
	for _, item := range ValidScopes {
		if item == scope {
			return true
		}
	}

	return false
}

// ==================== 组件审核状态常量 ====================

const (
	ComponentReviewDraft    = "draft"
	ComponentReviewCaptured = "captured"
	ComponentReviewPending  = "pending"
	ComponentReviewApproved = "approved"
	ComponentReviewRejected = "rejected"
)

// ==================== 请求结构体 ====================

// CreateComponentRequest 创建组件请求。
//
// EducationDomain只对mixed系统管理上下文生效：
//   - 普通教学Actor由服务端强制使用其可信教学域，忽略前端伪造值；
//   - mixed系统管理员必须显式选择资源域；
//   - common只允许系统管理员创建。
type CreateComponentRequest struct {
	EducationDomain string `json:"education_domain,omitempty"`

	LibraryType    string  `json:"library_type"`
	Subject        string  `json:"subject"`
	GradeRange     string  `json:"grade_range"`
	Tags           string  `json:"tags"`
	InjectionMode  string  `json:"injection_mode"`
	DisplayLabel   string  `json:"display_label"`
	DesignLogic    string  `json:"design_logic"`
	ExampleSnippet string  `json:"example_snippet"`
	FullGuide      string  `json:"full_guide"`
	Content        string  `json:"content"`
	Scope          string  `json:"scope"`
	ScopeRefID     *string `json:"scope_ref_id"`
}

// UpdateComponentRequest 更新组件请求。
//
// 教育域不在普通更新协议中，避免把资源从一个教育域原地搬到另一个教育域。
// 跨域迁移属于独立的数据治理操作，不应混入组件内容编辑。
type UpdateComponentRequest struct {
	Subject        string  `json:"subject"`
	GradeRange     string  `json:"grade_range"`
	Tags           string  `json:"tags"`
	InjectionMode  string  `json:"injection_mode"`
	DisplayLabel   string  `json:"display_label"`
	DesignLogic    string  `json:"design_logic"`
	ExampleSnippet string  `json:"example_snippet"`
	FullGuide      string  `json:"full_guide"`
	Content        string  `json:"content"`
	Scope          string  `json:"scope"`
	ScopeRefID     *string `json:"scope_ref_id"`
	Status         string  `json:"status"`
}

// MatchComponentsRequest 组件匹配请求。
//
// EducationDomain由服务端根据可信Actor或具体教案快照覆盖。
// 前端字段只用于mixed管理页面显式指定目标资源域，不能作为普通用户授权依据。
type MatchComponentsRequest struct {
	EducationDomain string `json:"education_domain,omitempty"`

	Subject       string   `json:"subject"`
	GradeRange    string   `json:"grade_range"`
	LibraryTypes  []string `json:"library_types"`
	InjectionMode string   `json:"injection_mode"`
	Tags          []string `json:"tags"`
	Limit         int      `json:"limit"`

	CognitiveLevel    []int `json:"cognitive_level"`
	StageTiming       []int `json:"stage_timing"`
	PedagogyIntensity []int `json:"pedagogy_intensity"`
}

// ReviewComponentRequest 审核组件请求。
type ReviewComponentRequest struct {
	Decision string `json:"decision"`
	Comment  string `json:"comment"`
}

// ==================== 响应结构体 ====================

// ComponentListResponse 组件列表响应。
type ComponentListResponse struct {
	Components []*ComponentListItem `json:"components"`
	Total      int                  `json:"total"`
}

// ComponentListItem 组件列表单条。
type ComponentListItem struct {
	ID              string `json:"id"`
	EducationDomain string `json:"education_domain"`

	LibraryType   string `json:"library_type"`
	LibraryName   string `json:"library_name"`
	Subject       string `json:"subject"`
	GradeRange    string `json:"grade_range"`
	InjectionMode string `json:"injection_mode"`
	DisplayLabel  string `json:"display_label"`

	QualityScore float64 `json:"quality_score"`
	UsageCount   int     `json:"usage_count"`
	SelectCount  int     `json:"select_count"`

	Source       string     `json:"source"`
	ReviewStatus string     `json:"review_status"`
	Scope        string     `json:"scope"`
	Status       string     `json:"status"`
	CreatedAt    *time.Time `json:"created_at"`
}

// MatchedComponentGroup 匹配结果分组。
type MatchedComponentGroup struct {
	LibraryType string              `json:"library_type"`
	LibraryName string              `json:"library_name"`
	Components  []*MatchedComponent `json:"components"`
}

// MatchedComponent 匹配到的单个组件。
type MatchedComponent struct {
	ID              string `json:"id"`
	EducationDomain string `json:"education_domain"`

	DisplayLabel   string `json:"display_label"`
	DesignLogic    string `json:"design_logic"`
	ExampleSnippet string `json:"example_snippet"`
	FullGuide      string `json:"full_guide"`

	QualityScore float64 `json:"quality_score"`
	UsageCount   int     `json:"usage_count"`
	SelectCount  int     `json:"select_count"`

	Tags           string `json:"tags"`
	ComponentIndex string `json:"component_index"`
}

// MatchComponentsResponse 组件匹配响应。
type MatchComponentsResponse struct {
	Groups []*MatchedComponentGroup `json:"groups"`
}
