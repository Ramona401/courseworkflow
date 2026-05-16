package models

import (
	"time"
)

// ==================== 课件工坊模型（对应 coursewares + courseware_pages + courseware_assets 表） ====================

// ==================== 课件主表模型 ====================

// Courseware 课件主记录（对应 coursewares 表）
// 状态机: draft → indexing → styling → generating → preview → confirmed → in_pipeline
type Courseware struct {
	ID            string     `json:"id"`
	LessonPlanID  string     `json:"lesson_plan_id"`   // 关联教案ID
	UserID        string     `json:"user_id"`          // 创建者用户ID
	Title         string     `json:"title"`            // 课件标题
	Subject       string     `json:"subject"`          // 学科
	Grade         string     `json:"grade"`            // 学段
	Status        string     `json:"status"`           // 状态
	StyleConfig   string     `json:"style_config"`     // 风格配置JSONB
	PageCount     int        `json:"page_count"`       // 页面总数
	IndexOverview string     `json:"index_overview"`   // Phase 3.5: 课件脉络概述
	PipelineID    *string    `json:"pipeline_id"`      // 提交审核后回填
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// ==================== 课件页面模型 ====================

// CoursewarePage 课件单页（对应 courseware_pages 表）
// Phase 3.5 两层架构：
//   - 层1 技术索引：page_index TEXT + 3个冗余列（admin可见）
//   - 层2 用户方案：title/purpose/content_summary等8个字段（翻译后展示）
type CoursewarePage struct {
	ID                  string     `json:"id"`
	CoursewareID        string     `json:"courseware_id"`
	PageNumber          int        `json:"page_number"`
	// ---- 层2：用户友好方案字段 ----
	Title               string     `json:"title"`
	Purpose             string     `json:"purpose"`
	ContentSummary      string     `json:"content_summary"`
	InteractionType     string     `json:"interaction_type"`
	VisualFormat        string     `json:"visual_format"`
	MediaRequirements   string     `json:"media_requirements"`
	EstimatedComplexity int        `json:"estimated_complexity"`
	// ---- 层1：AOCI技术索引 ----
	PageIndex           string     `json:"page_index"`
	IdxCognitiveLevel   int        `json:"idx_cognitive_level"`
	IdxInteractionLevel int        `json:"idx_interaction_level"`
	IdxVisualFormat     string     `json:"idx_visual_format"`
	// ---- 生成相关字段 ----
	HTMLContent         string     `json:"html_content"`
	PlaceholderMap      string     `json:"placeholder_map"`
	MatchedComponentIDs string     `json:"matched_component_ids"`
	Status              string     `json:"status"`
	CreatedAt           *time.Time `json:"created_at"`
	UpdatedAt           *time.Time `json:"updated_at"`
}

// ==================== 课件多媒体资源模型 ====================

// CoursewareAsset 课件多媒体资源（对应 courseware_assets 表）
type CoursewareAsset struct {
	ID              string     `json:"id"`
	CoursewareID    string     `json:"courseware_id"`
	PageID          *string    `json:"page_id"`
	PlaceholderID   string     `json:"placeholder_id"`
	AssetType       string     `json:"asset_type"`
	GenerationPrompt string    `json:"generation_prompt"`
	OssURL          string     `json:"oss_url"`
	FileSize        int64      `json:"file_size"`
	MimeType        string     `json:"mime_type"`
	Status          string     `json:"status"`
	CreatedAt       *time.Time `json:"created_at"`
}

// ==================== 课件状态常量 ====================

const (
	CoursewareStatusDraft      = "draft"
	CoursewareStatusIndexing   = "indexing"
	CoursewareStatusStyling    = "styling"
	CoursewareStatusGenerating = "generating"
	CoursewareStatusPreview    = "preview"
	CoursewareStatusConfirmed  = "confirmed"
	CoursewareStatusInPipeline = "in_pipeline"
)

// CoursewareStatusNameMap 课件状态中文名映射
var CoursewareStatusNameMap = map[string]string{
	CoursewareStatusDraft:      "草稿",
	CoursewareStatusIndexing:   "方案编辑中",
	CoursewareStatusStyling:    "风格选择中",
	CoursewareStatusGenerating: "课件生成中",
	CoursewareStatusPreview:    "预览确认中",
	CoursewareStatusConfirmed:  "已确认",
	CoursewareStatusInPipeline: "审核中",
}

// ==================== 课件页面状态常量 ====================

const (
	CWPageStatusPending      = "pending"
	CWPageStatusGenerated    = "generated"
	CWPageStatusMediaFilling = "media_filling"
	CWPageStatusConfirmed    = "confirmed"
)

// ==================== 课件资源状态/类型常量 ====================

const (
	CWAssetStatusPending    = "pending"
	CWAssetStatusGenerating = "generating"
	CWAssetStatusUploaded   = "uploaded"
	CWAssetStatusConfirmed  = "confirmed"
	CWAssetTypeImage        = "image"
	CWAssetTypeVideo        = "video"
)

// ==================== 请求结构体 ====================

// CreateCoursewareRequest 创建课件请求
type CreateCoursewareRequest struct {
	LessonPlanID string `json:"lesson_plan_id"`
	Title        string `json:"title"`
}

// UpdateCoursewareRequest 更新课件基本信息请求
type UpdateCoursewareRequest struct {
	Title string `json:"title"`
}

// UpdateCoursewareStyleRequest 保存风格选择请求
type UpdateCoursewareStyleRequest struct {
	StyleConfig string `json:"style_config"`
}

// UpdateCWPageIndexRequest 更新单页索引说明请求
type UpdateCWPageIndexRequest struct {
	Title              string `json:"title"`
	Purpose            string `json:"purpose"`
	ContentSummary     string `json:"content_summary"`
	InteractionType    string `json:"interaction_type"`
	VisualFormat       string `json:"visual_format"`
	MediaRequirements  string `json:"media_requirements"`
	EstimatedComplexity int   `json:"estimated_complexity"`
}

// AddCWPageRequest 手动添加课件页面请求
type AddCWPageRequest struct {
	Title              string `json:"title"`
	Purpose            string `json:"purpose"`
	ContentSummary     string `json:"content_summary"`
	InteractionType    string `json:"interaction_type"`
	VisualFormat       string `json:"visual_format"`
	MediaRequirements  string `json:"media_requirements"`
	EstimatedComplexity int   `json:"estimated_complexity"`
}

// ReorderCWPagesRequest 页面排序请求
type ReorderCWPagesRequest struct {
	PageIDs []string `json:"page_ids"`
}

// GenerateImageRequest 生成图片请求
type GenerateImageRequest struct {
	PlaceholderID    string `json:"placeholder_id"`
	GenerationPrompt string `json:"generation_prompt"`
}

// ApplyImageRequest 确认图片并替换占位符请求
type ApplyImageRequest struct {
	PlaceholderID string `json:"placeholder_id"`
	AssetID       string `json:"asset_id"`
}

// ==================== 响应结构体 ====================

// CoursewareListItem 课件列表单条
type CoursewareListItem struct {
	ID             string     `json:"id"`
	LessonPlanID   string     `json:"lesson_plan_id"`
	LessonPlanTitle string    `json:"lesson_plan_title"`
	Title          string     `json:"title"`
	Subject        string     `json:"subject"`
	Grade          string     `json:"grade"`
	Status         string     `json:"status"`
	StatusName     string     `json:"status_name"`
	PageCount      int        `json:"page_count"`
	PipelineID     *string    `json:"pipeline_id"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

// CoursewareDetailResponse 课件详情响应
type CoursewareDetailResponse struct {
	ID             string            `json:"id"`
	LessonPlanID   string            `json:"lesson_plan_id"`
	LessonPlanTitle string           `json:"lesson_plan_title"`
	UserID         string            `json:"user_id"`
	Title          string            `json:"title"`
	Subject        string            `json:"subject"`
	Grade          string            `json:"grade"`
	Status         string            `json:"status"`
	StatusName     string            `json:"status_name"`
	StyleConfig    string            `json:"style_config"`
	PageCount      int               `json:"page_count"`
	IndexOverview  string            `json:"index_overview"` // Phase 3.5: 课件脉络概述
	PipelineID     *string           `json:"pipeline_id"`
	Pages          []*CoursewarePage `json:"pages"`
	CreatedAt      *time.Time        `json:"created_at"`
	UpdatedAt      *time.Time        `json:"updated_at"`
}

// CoursewareListResponse 课件列表响应
type CoursewareListResponse struct {
	Coursewares []*CoursewareListItem `json:"coursewares"`
	Total       int                   `json:"total"`
}
