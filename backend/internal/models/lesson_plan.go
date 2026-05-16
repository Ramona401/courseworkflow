package models

import (
	"time"
)

// ==================== 教案模型（对应 lesson_plans 表） ====================

// LessonPlan 教案模型
// 教案是独立资产，可独立使用、分享、fork
// 双路径状态流转：个人使用（无需审核）/ 共享沉淀（需要评审）
// v56新增：RecipeID 关联备课配方
// v58新增：CurrentStage + StageConfig 阶段化备课工坊
// v127新增：ReviewLevel + ReviewSchoolID 多级审核支持
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
	RecipeID          *string    `json:"recipe_id"`           // Phase 7A：使用的备课配方ID
	ViewCount         int        `json:"view_count"`          // 浏览次数
	UseCount          int        `json:"use_count"`           // 使用次数
	Version           int        `json:"version"`             // 版本号
	CurrentStage      string     `json:"current_stage"`       // Phase 7B：当前所在阶段代码（空=未开始/旧模式）
	StageConfig       string     `json:"stage_config"`        // Phase 7B：阶段配置快照JSON
	TextbookPageIDs   string     `json:"textbook_page_ids"`   // 迭代7B：关联的课本图片ID数组JSON
	LessonIndex          string     `json:"lesson_index"`           // v86新增：AOCI索引文本（编码行+语义标签行）
	IdxCognitiveLevel    int        `json:"idx_cognitive_level"`    // v86新增：认知层级冗余列（1-6）
	IdxPedagogyIntensity int        `json:"idx_pedagogy_intensity"` // v86新增：教法强度冗余列（1-3）
	IdxStructureType     int        `json:"idx_structure_type"`     // v86新增：结构类型冗余列（1-5）
	IdxQualityLevel      int        `json:"idx_quality_level"`      // v86新增：质量等级冗余列（1-5）
	ReviewLevel       int        `json:"review_level"`        // v127新增：当前审核级别（0=未提交, 1=L1, 2=L2, 3=L3）
	ReviewSchoolID    *string    `json:"review_school_id"`    // v127新增：审核关联的学校ID
	CreatedAt         *time.Time `json:"created_at"`          // 创建时间
	UpdatedAt         *time.Time `json:"updated_at"`          // 更新时间
}

// ==================== 教案状态常量 ====================

const (
	LPStatusDraft             = "draft"              // 草稿
	LPStatusPublishedPersonal = "published_personal"  // 个人发布（即刻可用）
	LPStatusSubmitted         = "submitted"           // 已提交评审（等待L1）
	LPStatusRevision          = "revision"            // 退回修改（来自任意级别）
	LPStatusApproved          = "approved"            // 最终审核通过
	LPStatusPublishedShared   = "published_shared"    // 共享发布
	LPStatusDeveloping        = "developing"          // 课件开发中（教案锁定）
	LPStatusCompleted         = "completed"           // 已完成
)

// LPStatusNameMap 教案状态中文名映射（v127新增三种状态）
var LPStatusNameMap = map[string]string{
	LPStatusDraft:             "草稿",
	LPStatusPublishedPersonal: "个人发布",
	LPStatusSubmitted:         "已提交评审",
	LPStatusRevision:          "退回修改",
	LPStatusApproved:          "审核通过",
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

// ==================== 教案评审模型（对应 lesson_plan_reviews 表，保留向后兼容） ====================

// LessonPlanReview 教案评审记录（旧版单级审核，保留兼容）
type LessonPlanReview struct {
	ID           string     `json:"id"`
	LessonPlanID string     `json:"lesson_plan_id"`
	ReviewerID   string     `json:"reviewer_id"`
	Decision     string     `json:"decision"`
	Score        *float64   `json:"score"`
	Dimensions   string     `json:"dimensions"`
	Comments     string     `json:"comments"`
	Suggestions  string     `json:"suggestions"`
	Round        int        `json:"round"`
	CreatedAt    *time.Time `json:"created_at"`
}

// ==================== 组件萃取模型（对应 component_extractions 表） ====================

// ComponentExtraction 组件萃取记录
type ComponentExtraction struct {
	ID                   string     `json:"id"`
	SourceType           string     `json:"source_type"`
	SourceLessonPlanID   *string    `json:"source_lesson_plan_id"`
	SourceContent        string     `json:"source_content"`
	ExtractedComponentID *string    `json:"extracted_component_id"`
	ExtractionType       string     `json:"extraction_type"`
	Status               string     `json:"status"`
	ConfirmedBy          *string    `json:"confirmed_by"`
	ConfirmedAt          *time.Time `json:"confirmed_at"`
	CreatedBy            *string    `json:"created_by"`
	CreatedAt            *time.Time `json:"created_at"`
}

// ==================== 提示词模板模型（对应 prompt_templates 表） ====================

// PromptTemplate 提示词模板
type PromptTemplate struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Description       *string    `json:"description"`
	Level             string     `json:"level"`
	OwnerID           string     `json:"owner_id"`
	ParentTemplateID  *string    `json:"parent_template_id"`
	SystemPrompt      *string    `json:"system_prompt"`
	ContextRules      *string    `json:"context_rules"`
	GenerationRules   *string    `json:"generation_rules"`
	ReviewRules       *string    `json:"review_rules"`
	OutputFormat      *string    `json:"output_format"`
	CustomInstructions *string   `json:"custom_instructions"`
	Subject           string     `json:"subject"`
	GradeRange        string     `json:"grade_range"`
	IsDefault         bool       `json:"is_default"`
	Version           int        `json:"version"`
	Status            string     `json:"status"`
	CreatedBy         *string    `json:"created_by"`
	CreatedAt         *time.Time `json:"created_at"`
	UpdatedAt         *time.Time `json:"updated_at"`
}

// ==================== 提示词模板层级常量 ====================

const (
	TemplateLevelRegion   = "region"
	TemplateLevelSchool   = "school"
	TemplateLevelGroup    = "group"
	TemplateLevelPersonal = "personal"
)

var ValidTemplateLevels = []string{
	TemplateLevelRegion, TemplateLevelSchool,
	TemplateLevelGroup, TemplateLevelPersonal,
}

func IsValidTemplateLevel(level string) bool {
	for _, v := range ValidTemplateLevels {
		if v == level {
			return true
		}
	}
	return false
}

// ==================== 教案请求结构体 ====================

type CreateLessonPlanRequest struct {
	Title           string `json:"title"`
	Subject         string `json:"subject"`
	Grade           string `json:"grade"`
	Topic           string `json:"topic"`
	DurationMinutes int    `json:"duration_minutes"`
}

type UpdateLessonPlanRequest struct {
	Title           string `json:"title"`
	ContentMarkdown string `json:"content_markdown"`
	DurationMinutes int    `json:"duration_minutes"`
}

type SubmitLessonPlanReviewRequest struct {
	GroupID string `json:"group_id"`
}

type CreateLessonPlanReviewRequest struct {
	Decision    string   `json:"decision"`
	Score       *float64 `json:"score"`
	Dimensions  string   `json:"dimensions"`
	Comments    string   `json:"comments"`
	Suggestions string   `json:"suggestions"`
}

type ForkLessonPlanRequest struct{}

// ==================== 教案响应结构体 ====================

type LessonPlanListResponse struct {
	LessonPlans []*LessonPlanListItem `json:"lesson_plans"`
	Total       int                   `json:"total"`
}

type LessonPlanListItem struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Subject         string     `json:"subject"`
	Grade           string     `json:"grade"`
	Topic           string     `json:"topic"`
	DurationMinutes int        `json:"duration_minutes"`
	Status          string     `json:"status"`
	StatusName      string     `json:"status_name"`
	Visibility      string     `json:"visibility"`
	AuthorID        string     `json:"author_id"`
	AuthorName      string     `json:"author_name"`
	AIReviewScore   *float64   `json:"ai_review_score"`
	ForkCount       int        `json:"fork_count"`
	ViewCount       int        `json:"view_count"`
	ForkedFrom      *string    `json:"forked_from"`
	RecipeID        *string    `json:"recipe_id,omitempty"`        // Phase 7A：关联配方ID
	RecipeName      string     `json:"recipe_name,omitempty"`     // Phase 7A：关联配方名称
	LessonIndex     string     `json:"lesson_index,omitempty"`    // v86新增：AOCI索引文本
	IdxQualityLevel int        `json:"idx_quality_level"`         // v86新增：质量等级
	ReviewLevel     int        `json:"review_level"`              // v127新增：当前审核级别
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
	// v125新增：互动计数（由service层填充）
	LikeCount     int  `json:"like_count"`      // 点赞总数
	FavoriteCount int  `json:"favorite_count"`   // 收藏总数
	IsLiked       bool `json:"is_liked"`         // 当前用户是否已点赞
	IsFavorited   bool `json:"is_favorited"`     // 当前用户是否已收藏
}

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
	RecipeID          *string    `json:"recipe_id,omitempty"`           // Phase 7A：关联配方ID
	RecipeName        string     `json:"recipe_name,omitempty"`         // Phase 7A：关联配方名称
	LessonIndex       string     `json:"lesson_index,omitempty"`        // v86新增：AOCI索引
	IdxCognitiveLevel int        `json:"idx_cognitive_level"`           // v86新增：认知层级
	IdxPedagogyIntensity int     `json:"idx_pedagogy_intensity"`        // v86新增：教法强度
	IdxStructureType  int        `json:"idx_structure_type"`            // v86新增：结构类型
	IdxQualityLevel   int        `json:"idx_quality_level"`             // v86新增：质量等级
	ReviewLevel       int        `json:"review_level"`                  // v127新增：当前审核级别
	ReviewSchoolID    *string    `json:"review_school_id,omitempty"`    // v127新增：审核关联学校
	CurrentStage      string     `json:"current_stage,omitempty"`       // Phase 7B：当前阶段
	StageConfig       string     `json:"stage_config,omitempty"`        // Phase 7B：阶段配置
	// v125新增：互动计数
	LikeCount     int  `json:"like_count"`      // 点赞总数
	FavoriteCount int  `json:"favorite_count"`   // 收藏总数
	IsLiked       bool `json:"is_liked"`         // 当前用户是否已点赞
	IsFavorited   bool `json:"is_favorited"`     // 当前用户是否已收藏
	Reviews           []*LessonPlanReviewItem `json:"reviews"`
	LinkedPipelineID  *string                `json:"linked_pipeline_id,omitempty"`
	CreatedAt         *time.Time `json:"created_at"`
	UpdatedAt         *time.Time `json:"updated_at"`
}

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

type CreatePromptTemplateRequest struct {
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	Level              string  `json:"level"`
	OwnerID            string  `json:"owner_id"`
	ParentTemplateID   *string `json:"parent_template_id"`
	SystemPrompt       string  `json:"system_prompt"`
	ContextRules       string  `json:"context_rules"`
	GenerationRules    string  `json:"generation_rules"`
	ReviewRules        string  `json:"review_rules"`
	OutputFormat       string  `json:"output_format"`
	CustomInstructions string  `json:"custom_instructions"`
	Subject            string  `json:"subject"`
	GradeRange         string  `json:"grade_range"`
	IsDefault          bool    `json:"is_default"`
}

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

type PromptTemplateListResponse struct {
	Templates []*PromptTemplateListItem `json:"templates"`
	Total     int                       `json:"total"`
}

type PromptTemplateListItem struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Level            string     `json:"level"`
	OwnerID          string     `json:"owner_id"`
	OwnerName        string     `json:"owner_name"`
	ParentTemplateID *string    `json:"parent_template_id"`
	Subject          string     `json:"subject"`
	GradeRange       string     `json:"grade_range"`
	IsDefault        bool       `json:"is_default"`
	Version          int        `json:"version"`
	Status           string     `json:"status"`
	CreatedAt        *time.Time `json:"created_at"`
}

type ResolvedPromptTemplate struct {
	SystemPrompt       string   `json:"system_prompt"`
	ContextRules       string   `json:"context_rules"`
	GenerationRules    string   `json:"generation_rules"`
	ReviewRules        string   `json:"review_rules"`
	OutputFormat       string   `json:"output_format"`
	CustomInstructions string   `json:"custom_instructions"`
	InheritanceChain   []string `json:"inheritance_chain"`
}
