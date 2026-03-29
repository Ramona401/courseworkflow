package models

import (
	"time"
)

// ==================== 教案模型（对应 lesson_plans 表） ====================

// LessonPlan 教案模型
// 教案是独立资产，可独立使用、分享、fork
// 双路径状态流转：个人使用（无需审核）/ 共享沉淀（需要评审）
type LessonPlan struct {
	ID                string     `json:"id"`                  // UUID主键
	Title             string     `json:"title"`               // 教案标题
	Subject           string     `json:"subject"`             // 学科
	Grade             string     `json:"grade"`               // 年级
	Topic             string     `json:"topic"`               // 课题
	DurationMinutes   int        `json:"duration_minutes"`    // 课时时长（分钟）
	ContentMarkdown   string     `json:"content_markdown"`    // Markdown格式教案正文
	ContentStructured string     `json:"content_structured"`  // 结构化内容JSON
	GenerationConfig  string     `json:"generation_config"`   // 生成配置JSON
	MatchedComponents string     `json:"matched_components"`  // 匹配注入的组件ID列表JSON
	ConversationLog   string     `json:"conversation_log"`    // AI对话记录JSON
	AIReviewScore     *float64   `json:"ai_review_score"`     // AI评审总分
	AIReviewResult    string     `json:"ai_review_result"`    // AI评审详细结果JSON
	AIReviewHistory   string     `json:"ai_review_history"`   // 评审历史JSON
	Status            string     `json:"status"`              // 教案状态
	Visibility        string     `json:"visibility"`          // 可见范围
	AuthorID          string     `json:"author_id"`           // 作者用户ID
	GroupID           *string    `json:"group_id"`            // 所属教研组ID
	SchoolID          *string    `json:"school_id"`           // 所属学校ID
	ForkedFrom        *string    `json:"forked_from"`         // fork来源教案ID
	ForkCount         int        `json:"fork_count"`          // 被fork次数
	TemplateID        *string    `json:"template_id"`         // 使用的提示词模板ID
	ViewCount         int        `json:"view_count"`          // 浏览次数
	UseCount          int        `json:"use_count"`           // 使用次数
	Version           int        `json:"version"`             // 版本号
	CreatedAt         *time.Time `json:"created_at"`          // 创建时间
	UpdatedAt         *time.Time `json:"updated_at"`          // 更新时间
}

// ==================== 教案状态常量 ====================

const (
	LPStatusDraft             = "draft"              // 草稿
	LPStatusPublishedPersonal = "published_personal"  // 个人发布（即刻可用）
	LPStatusSubmitted         = "submitted"           // 已提交评审
	LPStatusRevision          = "revision"            // 退回修改
	LPStatusApproved          = "approved"            // 评审通过
	LPStatusPublishedShared   = "published_shared"    // 共享发布
	LPStatusDeveloping        = "developing"          // 课件开发中（教案锁定）
	LPStatusCompleted         = "completed"           // 已完成
)

// LPStatusNameMap 教案状态中文名映射
var LPStatusNameMap = map[string]string{
	LPStatusDraft:             "草稿",
	LPStatusPublishedPersonal: "个人发布",
	LPStatusSubmitted:         "已提交评审",
	LPStatusRevision:          "退回修改",
	LPStatusApproved:          "评审通过",
	LPStatusPublishedShared:   "共享发布",
	LPStatusDeveloping:        "课件开发中",
	LPStatusCompleted:         "已完成",
}

// ==================== 教案可见范围常量 ====================

const (
	LPVisibilityPersonal = "personal" // 个人可见
	LPVisibilityGroup    = "group"    // 教研组可见
	LPVisibilitySchool   = "school"   // 学校可见
	LPVisibilityRegion   = "region"   // 区域可见
	LPVisibilityPublic   = "public"   // 公开可见
)

// ==================== 教案评审模型（对应 lesson_plan_reviews 表） ====================

// LessonPlanReview 教案评审记录
// 记录教研组长/骨干对教案的人工评审
type LessonPlanReview struct {
	ID           string     `json:"id"`             // UUID主键
	LessonPlanID string     `json:"lesson_plan_id"` // 教案ID
	ReviewerID   string     `json:"reviewer_id"`    // 评审人用户ID
	Decision     string     `json:"decision"`       // 评审决策：approved/revision/rejected
	Score        *float64   `json:"score"`          // 评审总分
	Dimensions   string     `json:"dimensions"`     // 各维度评分JSON
	Comments     string     `json:"comments"`       // 评审意见
	Suggestions  string     `json:"suggestions"`    // 具体建议列表JSON
	Round        int        `json:"round"`          // 评审轮次
	CreatedAt    *time.Time `json:"created_at"`     // 创建时间
}

// ==================== 组件萃取模型（对应 component_extractions 表） ====================

// ComponentExtraction 组件萃取记录
// 记录从教案/对话中萃取组件的过程
type ComponentExtraction struct {
	ID                   string     `json:"id"`                      // UUID主键
	SourceType           string     `json:"source_type"`             // 来源类型：conversation/lesson_plan/manual
	SourceLessonPlanID   *string    `json:"source_lesson_plan_id"`   // 来源教案ID
	SourceContent        string     `json:"source_content"`          // 来源内容片段
	ExtractedComponentID *string    `json:"extracted_component_id"`  // 萃取生成的组件ID
	ExtractionType       string     `json:"extraction_type"`         // 萃取类型
	Status               string     `json:"status"`                  // 状态：pending/confirmed/rejected
	ConfirmedBy          *string    `json:"confirmed_by"`            // 确认人
	ConfirmedAt          *time.Time `json:"confirmed_at"`            // 确认时间
	CreatedBy            *string    `json:"created_by"`              // 创建者
	CreatedAt            *time.Time `json:"created_at"`              // 创建时间
}

// ==================== 提示词模板模型（对应 prompt_templates 表） ====================

// PromptTemplate 提示词模板
// 支持区域→学校→教研组→个人四级继承
// 子级通过 ParentTemplateID 指向父级，解析时子级覆盖父级
type PromptTemplate struct {
	ID                string     `json:"id"`                  // UUID主键
	Name              string     `json:"name"`                // 模板名称
	Description       *string    `json:"description"`         // 模板描述（可NULL）
	Level             string     `json:"level"`               // 归属层级：region/school/group/personal
	OwnerID           string     `json:"owner_id"`            // 归属ID
	ParentTemplateID  *string    `json:"parent_template_id"`  // 父级模板ID
	SystemPrompt      *string    `json:"system_prompt"`       // 系统提示词（可NULL）
	ContextRules      *string    `json:"context_rules"`       // 上下文规则JSON（可NULL）
	GenerationRules   *string    `json:"generation_rules"`    // 生成规则JSON（可NULL）
	ReviewRules       *string    `json:"review_rules"`        // 评审规则JSON（可NULL）
	OutputFormat      *string    `json:"output_format"`       // 输出格式JSON（可NULL）
	CustomInstructions *string   `json:"custom_instructions"` // 自定义指令（可NULL）
	Subject           string     `json:"subject"`             // 适用学科
	GradeRange        string     `json:"grade_range"`         // 适用学段
	IsDefault         bool       `json:"is_default"`          // 是否默认模板
	Version           int        `json:"version"`             // 版本号
	Status            string     `json:"status"`              // 状态：active/disabled
	CreatedBy         *string    `json:"created_by"`          // 创建者
	CreatedAt         *time.Time `json:"created_at"`          // 创建时间
	UpdatedAt         *time.Time `json:"updated_at"`          // 更新时间
}

// ==================== 提示词模板层级常量 ====================

const (
	TemplateLevelRegion   = "region"   // 区域级
	TemplateLevelSchool   = "school"   // 学校级
	TemplateLevelGroup    = "group"    // 教研组级
	TemplateLevelPersonal = "personal" // 个人级
)

// ValidTemplateLevels 有效的模板层级列表
var ValidTemplateLevels = []string{
	TemplateLevelRegion, TemplateLevelSchool,
	TemplateLevelGroup, TemplateLevelPersonal,
}

// IsValidTemplateLevel 检查模板层级是否有效
func IsValidTemplateLevel(level string) bool {
	for _, v := range ValidTemplateLevels {
		if v == level {
			return true
		}
	}
	return false
}

// ==================== 教案请求结构体 ====================

// CreateLessonPlanRequest 创建教案请求（从备课工坊发起）
type CreateLessonPlanRequest struct {
	Title           string `json:"title"`            // 教案标题（必填）
	Subject         string `json:"subject"`          // 学科（必填）
	Grade           string `json:"grade"`            // 年级（必填）
	Topic           string `json:"topic"`            // 课题（必填）
	DurationMinutes int    `json:"duration_minutes"` // 课时时长（可选，默认45）
}

// UpdateLessonPlanRequest 更新教案请求
type UpdateLessonPlanRequest struct {
	Title           string `json:"title"`            // 教案标题
	ContentMarkdown string `json:"content_markdown"` // Markdown内容
	DurationMinutes int    `json:"duration_minutes"` // 课时时长
}

// SubmitLessonPlanReviewRequest 提交教案评审请求
type SubmitLessonPlanReviewRequest struct {
	GroupID string `json:"group_id"` // 提交到哪个教研组（必填）
}

// CreateLessonPlanReviewRequest 创建教案评审请求（评审人操作）
type CreateLessonPlanReviewRequest struct {
	Decision    string   `json:"decision"`    // 评审决策：approved/revision/rejected（必填）
	Score       *float64 `json:"score"`       // 评审总分（可选）
	Dimensions  string   `json:"dimensions"`  // 各维度评分JSON（可选）
	Comments    string   `json:"comments"`    // 评审意见（必填）
	Suggestions string   `json:"suggestions"` // 具体建议列表JSON（可选）
}

// ForkLessonPlanRequest fork教案请求
type ForkLessonPlanRequest struct {
	// 暂无额外字段，fork时自动复制教案内容
}

// ==================== 教案响应结构体 ====================

// LessonPlanListResponse 教案列表响应
type LessonPlanListResponse struct {
	LessonPlans []*LessonPlanListItem `json:"lesson_plans"` // 教案列表
	Total       int                   `json:"total"`        // 总数
}

// LessonPlanListItem 教案列表单条
type LessonPlanListItem struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Subject         string     `json:"subject"`
	Grade           string     `json:"grade"`
	Topic           string     `json:"topic"`
	DurationMinutes int        `json:"duration_minutes"`
	Status          string     `json:"status"`
	StatusName      string     `json:"status_name"`     // 状态中文名
	Visibility      string     `json:"visibility"`
	AuthorID        string     `json:"author_id"`
	AuthorName      string     `json:"author_name"`     // 作者名称
	AIReviewScore   *float64   `json:"ai_review_score"` // AI评审分
	ForkCount       int        `json:"fork_count"`
	ViewCount       int        `json:"view_count"`
	ForkedFrom      *string    `json:"forked_from"`
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
}

// LessonPlanDetailResponse 教案详情响应
type LessonPlanDetailResponse struct {
	ID                string     `json:"id"`
	Title             string     `json:"title"`
	Subject           string     `json:"subject"`
	Grade             string     `json:"grade"`
	Topic             string     `json:"topic"`
	DurationMinutes   int        `json:"duration_minutes"`
	ContentMarkdown   string     `json:"content_markdown"`
	ContentStructured string     `json:"content_structured"`
	GenerationConfig  string     `json:"generation_config"`
	MatchedComponents string     `json:"matched_components"`
	AIReviewScore     *float64   `json:"ai_review_score"`
	AIReviewResult    string     `json:"ai_review_result"`
	AIReviewHistory   string     `json:"ai_review_history"`
	Status            string     `json:"status"`
	StatusName        string     `json:"status_name"`
	Visibility        string     `json:"visibility"`
	AuthorID          string     `json:"author_id"`
	AuthorName        string     `json:"author_name"`
	GroupID           *string    `json:"group_id"`
	GroupName         string     `json:"group_name"`
	SchoolID          *string    `json:"school_id"`
	ForkedFrom        *string    `json:"forked_from"`
	ForkCount         int        `json:"fork_count"`
	ViewCount         int        `json:"view_count"`
	UseCount          int        `json:"use_count"`
	Version           int        `json:"version"`
	Reviews           []*LessonPlanReviewItem `json:"reviews"`
	LinkedPipelineID  *string                `json:"linked_pipeline_id,omitempty"` // Phase6：关联Pipeline ID // 评审记录
	CreatedAt         *time.Time `json:"created_at"`
	UpdatedAt         *time.Time `json:"updated_at"`
}

// LessonPlanReviewItem 评审记录展示项
type LessonPlanReviewItem struct {
	ID           string     `json:"id"`
	ReviewerID   string     `json:"reviewer_id"`
	ReviewerName string     `json:"reviewer_name"`
	Decision     string     `json:"decision"`
	Score        *float64   `json:"score"`
	Comments     string     `json:"comments"`
	Round        int        `json:"round"`
	CreatedAt    *time.Time `json:"created_at"`
}

// ==================== 提示词模板请求结构体 ====================

// CreatePromptTemplateRequest 创建提示词模板请求
type CreatePromptTemplateRequest struct {
	Name               string  `json:"name"`                // 模板名称（必填）
	Description        string  `json:"description"`         // 描述（可选）
	Level              string  `json:"level"`               // 归属层级（必填）
	OwnerID            string  `json:"owner_id"`            // 归属ID（必填）
	ParentTemplateID   *string `json:"parent_template_id"`  // 父级模板ID（可选）
	SystemPrompt       string  `json:"system_prompt"`       // 系统提示词（可选）
	ContextRules       string  `json:"context_rules"`       // 上下文规则JSON（可选）
	GenerationRules    string  `json:"generation_rules"`    // 生成规则JSON（可选）
	ReviewRules        string  `json:"review_rules"`        // 评审规则JSON（可选）
	OutputFormat       string  `json:"output_format"`       // 输出格式JSON（可选）
	CustomInstructions string  `json:"custom_instructions"` // 自定义指令（可选）
	Subject            string  `json:"subject"`             // 适用学科（可选）
	GradeRange         string  `json:"grade_range"`         // 适用学段（可选）
	IsDefault          bool    `json:"is_default"`          // 是否默认（可选）
}

// UpdatePromptTemplateRequest 更新提示词模板请求
type UpdatePromptTemplateRequest struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	SystemPrompt       string `json:"system_prompt"`
	ContextRules       string `json:"context_rules"`
	GenerationRules    string `json:"generation_rules"`
	ReviewRules        string `json:"review_rules"`
	OutputFormat       string `json:"output_format"`
	CustomInstructions string `json:"custom_instructions"`
	Subject            string `json:"subject"`
	GradeRange         string `json:"grade_range"`
	IsDefault          bool   `json:"is_default"`
	Status             string `json:"status"`
}

// ==================== 提示词模板响应结构体 ====================

// PromptTemplateListResponse 模板列表响应
type PromptTemplateListResponse struct {
	Templates []*PromptTemplateListItem `json:"templates"` // 模板列表
	Total     int                       `json:"total"`     // 总数
}

// PromptTemplateListItem 模板列表单条
type PromptTemplateListItem struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Level            string     `json:"level"`
	OwnerID          string     `json:"owner_id"`
	OwnerName        string     `json:"owner_name"`         // 归属名称
	ParentTemplateID *string    `json:"parent_template_id"`
	Subject          string     `json:"subject"`
	GradeRange       string     `json:"grade_range"`
	IsDefault        bool       `json:"is_default"`
	Version          int        `json:"version"`
	Status           string     `json:"status"`
	CreatedAt        *time.Time `json:"created_at"`
}

// ResolvedPromptTemplate 解析后的完整模板（继承链合并结果）
// 从个人→教研组→学校→区域逐级向上合并，子级覆盖父级
type ResolvedPromptTemplate struct {
	SystemPrompt       string   `json:"system_prompt"`
	ContextRules       string   `json:"context_rules"`
	GenerationRules    string   `json:"generation_rules"`
	ReviewRules        string   `json:"review_rules"`
	OutputFormat       string   `json:"output_format"`
	CustomInstructions string   `json:"custom_instructions"`
	InheritanceChain   []string `json:"inheritance_chain"` // 继承链（从根到叶的模板ID列表）
}
