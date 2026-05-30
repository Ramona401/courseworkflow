package models

import (
	"time"
)

// ==================== 课件组件库模型（对应 courseware_components 表） ====================

// CoursewareComponent 课件组件（代码级模板）
// 每个组件是一段完整的HTML/CSS/JS代码，代表一种页面布局、交互形式或视觉效果
// AI生成课件时参考这些组件，通过AOCI索引自动匹配
type CoursewareComponent struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`                  // 组件名称
	Description         string     `json:"description"`           // 组件描述
	ComponentType       string     `json:"component_type"`        // 大类: layout/interaction/3d/animation/data_viz/multimedia/style
	CodeContent         string     `json:"code_content"`          // 完整HTML/CSS/JS代码
	PreviewImageURL     string     `json:"preview_image_url"`     // 预览截图URL
	PreviewHTML         string     `json:"preview_html"`          // 精简版可内嵌预览HTML
	SubjectScope        string     `json:"subject_scope"`         // 适用学科，ALL=通用
	GradeScope          string     `json:"grade_scope"`           // 适用学段，ALL=通用
	ComponentIndex      string     `json:"component_index"`       // AOCI压缩索引字符串
	IdxInteractionLevel *int       `json:"idx_interaction_level"` // 冗余列：交互复杂度1-5
	IdxVisualFormat     string     `json:"idx_visual_format"`     // 冗余列：视觉形式代码
	IdxTechTag          string     `json:"idx_tech_tag"`          // 冗余列：技术标签代码
	TechDependencies    string     `json:"tech_dependencies"`     // 技术依赖JSONB
	Tags                string     `json:"tags"`                  // 自定义标签JSONB
	IsActive            bool       `json:"is_active"`
	ReviewStatus        string     `json:"review_status"` // draft/approved/archived
	CreatedAt           *time.Time `json:"created_at"`
	UpdatedAt           *time.Time `json:"updated_at"`
}

// ==================== 课件组件类型常量（6大类） ====================

const (
	CWCompTypeLayout      = "layout"      // 布局模板
	CWCompTypeInteraction = "interaction" // 交互功能
	CWCompType3D          = "3d"          // 3D/动画
	CWCompTypeAnimation   = "animation"   // 动画效果
	CWCompTypeDataViz     = "data_viz"    // 数据可视化
	CWCompTypeMultimedia  = "multimedia"  // 多媒体容器
	CWCompTypeStyle       = "style"       // 样式主题
)

// ValidCWComponentTypes 有效的课件组件类型列表
var ValidCWComponentTypes = []string{
	CWCompTypeLayout, CWCompTypeInteraction, CWCompType3D,
	CWCompTypeAnimation, CWCompTypeDataViz, CWCompTypeMultimedia,
	CWCompTypeStyle,
}

// CWComponentTypeNameMap 课件组件类型中文名映射
var CWComponentTypeNameMap = map[string]string{
	CWCompTypeLayout:      "布局模板",
	CWCompTypeInteraction: "交互功能",
	CWCompType3D:          "3D/动画",
	CWCompTypeAnimation:   "动画效果",
	CWCompTypeDataViz:     "数据可视化",
	CWCompTypeMultimedia:  "多媒体容器",
	CWCompTypeStyle:       "样式主题",
}

// IsValidCWComponentType 检查课件组件类型是否有效
func IsValidCWComponentType(ct string) bool {
	for _, v := range ValidCWComponentTypes {
		if v == ct {
			return true
		}
	}
	return false
}

// ==================== 课件组件审核状态常量 ====================

const (
	CWCompReviewDraft    = "draft"    // 草稿
	CWCompReviewApproved = "approved" // 已通过
	CWCompReviewArchived = "archived" // 已归档
)

// ==================== AOCI索引维度常量 ====================

// 交互复杂度（IL维度）
const (
	CWInteractionLevelDisplay   = 1 // 纯展示
	CWInteractionLevelClick     = 2 // 简单点击
	CWInteractionLevelDrag      = 3 // 拖拽/选择
	CWInteractionLevelMultiStep = 4 // 多步交互
	CWInteractionLevelComplex3D = 5 // 复杂模拟/3D
)

// 视觉形式（VF维度）
const (
	CWVisualFormatTimeline   = "TL" // 时间线
	CWVisualFormatCompare    = "CP" // 对比
	CWVisualFormatCard       = "CG" // 卡片
	CWVisualFormatTable      = "TB" // 表格
	CWVisualFormatChart      = "CH" // 图表
	CWVisualFormatFullscreen = "FS" // 全屏
	CWVisualFormatGrid       = "GD" // 网格
	CWVisualFormatFreeform   = "FR" // 自由版式
)

// 技术标签（TG维度）
const (
	CWTechTagCSS     = "CSS" // 纯样式
	CWTechTagJS      = "JS"  // 基础脚本
	CWTechTagSVG     = "SVG" // 矢量图形
	CWTechTagCanvas  = "CV"  // Canvas画布
	CWTechTagThreeJS = "TJ"  // Three.js三维
	CWTechTagAnim    = "AN"  // 动画
)

// ==================== 请求结构体 ====================

// CreateCWComponentRequest 创建课件组件请求
type CreateCWComponentRequest struct {
	Name             string `json:"name"`              // 组件名称（必填）
	Description      string `json:"description"`       // 组件描述
	ComponentType    string `json:"component_type"`    // 组件大类（必填）
	CodeContent      string `json:"code_content"`      // 完整HTML/CSS/JS代码（必填）
	PreviewImageURL  string `json:"preview_image_url"` // 预览截图URL
	PreviewHTML      string `json:"preview_html"`      // 精简版预览HTML
	SubjectScope     string `json:"subject_scope"`     // 适用学科
	GradeScope       string `json:"grade_scope"`       // 适用学段
	TechDependencies string `json:"tech_dependencies"` // 技术依赖JSON
	Tags             string `json:"tags"`              // 自定义标签JSON
}

// UpdateCWComponentRequest 更新课件组件请求
type UpdateCWComponentRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	ComponentType    string `json:"component_type"`
	CodeContent      string `json:"code_content"`
	PreviewImageURL  string `json:"preview_image_url"`
	PreviewHTML      string `json:"preview_html"`
	SubjectScope     string `json:"subject_scope"`
	GradeScope       string `json:"grade_scope"`
	TechDependencies string `json:"tech_dependencies"`
	Tags             string `json:"tags"`
	IsActive         *bool  `json:"is_active"`
	ReviewStatus     string `json:"review_status"`
}

// MatchCWComponentsRequest 课件组件匹配请求
// 根据交互类型+视觉形式+学科+学段+复杂度匹配最合适的组件
type MatchCWComponentsRequest struct {
	ComponentType    string `json:"component_type"`    // 组件大类（可选）
	SubjectScope     string `json:"subject_scope"`     // 学科
	GradeScope       string `json:"grade_scope"`       // 学段
	InteractionLevel int    `json:"interaction_level"` // 交互复杂度 1-5
	VisualFormat     string `json:"visual_format"`     // 视觉形式代码
	TechTag          string `json:"tech_tag"`          // 技术标签
	Limit            int    `json:"limit"`             // 返回数量上限（默认3）
}

// ==================== 响应结构体 ====================

// CWComponentListItem 课件组件列表单条
type CWComponentListItem struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	ComponentType       string     `json:"component_type"`
	ComponentTypeName   string     `json:"component_type_name"` // 中文名
	PreviewImageURL     string     `json:"preview_image_url"`
	SubjectScope        string     `json:"subject_scope"`
	GradeScope          string     `json:"grade_scope"`
	ComponentIndex      string     `json:"component_index"`
	IdxInteractionLevel *int       `json:"idx_interaction_level"`
	IsActive            bool       `json:"is_active"`
	ReviewStatus        string     `json:"review_status"`
	CreatedAt           *time.Time `json:"created_at"`
}

// CWComponentListResponse 课件组件列表响应
type CWComponentListResponse struct {
	Components []*CWComponentListItem `json:"components"`
	Total      int                    `json:"total"`
}

// MatchedCWComponent 匹配到的课件组件（返回代码供AI参考）
type MatchedCWComponent struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ComponentType    string `json:"component_type"`
	CodeContent      string `json:"code_content"` // 完整代码
	PreviewHTML      string `json:"preview_html"`
	ComponentIndex   string `json:"component_index"`
	InteractionLevel *int   `json:"interaction_level"`
}

// ==================== 风格模板模型（对应 courseware_templates 表） ====================
// v139 变更：
//   - 新增 ScopeTargetID 字段（学校ID或教研组ID,不加外键约束）
//   - 新增 IsDraft 字段（区分AI提取草稿和正式模板）
//   - 新增 RefineHistory 字段（AI微调历史快照JSONB数组）
//   - 新增 ExtractSourceMeta 字段（AI提取来源元信息JSONB）
//   - Scope 从 system/personal 扩展到 system/school/group/personal

// CoursewareTemplate 课件风格模板
// 预定义的视觉风格配置，包含配色、CSS变量、样例页面
// v137: user_id/scope/source_courseware_id 支持个人模板保存
// v139: 增加 scope_target_id/is_draft/refine_history/extract_source_meta 支持AI提取+微调+四级权限
type CoursewareTemplate struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	StyleCategory      string     `json:"style_category"`     // minimalist/playful/tech/academic/organic/immersive
	PreviewImageURL    string     `json:"preview_image_url"`
	ColorScheme        string     `json:"color_scheme"`        // 配色方案JSONB
	CSSVariables       string     `json:"css_variables"`       // CSS变量JSONB
	SamplePages        string     `json:"sample_pages"`        // 样例页面HTML数组JSONB
	PreviewURLs        string     `json:"preview_urls"`        // 在线预览链接数组JSONB
	IsActive           bool       `json:"is_active"`
	SortOrder          int        `json:"sort_order"`
	UserID             *string    `json:"user_id"`              // v137: 创建者ID（NULL=系统模板）
	Scope              string     `json:"scope"`                // v137: system/personal; v139扩: school/group
	SourceCoursewareID *string    `json:"source_courseware_id"` // v137: 来源课件ID（可空）
	// v139 新增 4 字段
	ScopeTargetID      *string `json:"scope_target_id"`       // v139: scope=school时为学校ID,scope=group时为教研组ID
	IsDraft            bool    `json:"is_draft"`              // v139: AI提取出来的草稿,默认列表过滤
	RefineHistory      string  `json:"refine_history"`        // v139: AI微调历史快照JSONB,最多20条
	ExtractSourceMeta  string  `json:"extract_source_meta"`   // v139: AI提取来源元信息JSONB
	CreatedAt          *time.Time `json:"created_at"`
	UpdatedAt          *time.Time `json:"updated_at"`
}

// ==================== 模板scope常量（v139扩展到 4 级） ====================

const (
	CWTemplateScopeSystem   = "system"   // 系统模板（admin管理）
	CWTemplateScopeSchool   = "school"   // v139: 学校模板（senior_operator管理本校）
	CWTemplateScopeGroup    = "group"    // v139: 教研组模板（lead/backbone管理本组）
	CWTemplateScopePersonal = "personal" // 个人模板（老师保存）
)

// ValidCWTemplateScopes 有效的模板 scope 列表
var ValidCWTemplateScopes = []string{
	CWTemplateScopeSystem,
	CWTemplateScopeSchool,
	CWTemplateScopeGroup,
	CWTemplateScopePersonal,
}

// IsValidCWTemplateScope 检查 scope 是否有效
func IsValidCWTemplateScope(scope string) bool {
	for _, v := range ValidCWTemplateScopes {
		if v == scope {
			return true
		}
	}
	return false
}

// CWTemplateScopeNameMap 模板 scope 中文名映射
var CWTemplateScopeNameMap = map[string]string{
	CWTemplateScopeSystem:   "系统模板",
	CWTemplateScopeSchool:   "本校模板",
	CWTemplateScopeGroup:    "本组模板",
	CWTemplateScopePersonal: "我的模板",
}

// ==================== 风格类别常量（Phase 4A: 新增immersive） ====================

const (
	CWStyleMinimalist = "minimalist" // 简约清新
	CWStylePlayful    = "playful"    // 活泼趣味
	CWStyleTech       = "tech"       // 科技感
	CWStyleAcademic   = "academic"   // 学术严谨
	CWStyleOrganic    = "organic"    // 自然有机
	CWStyleImmersive  = "immersive"  // 3D沉浸式
)

// CWStyleCategoryNameMap 风格类别中文名映射
var CWStyleCategoryNameMap = map[string]string{
	CWStyleMinimalist: "简约清新",
	CWStylePlayful:    "活泼趣味",
	CWStyleTech:       "科技感",
	CWStyleAcademic:   "学术严谨",
	CWStyleOrganic:    "自然有机",
	CWStyleImmersive:  "3D沉浸式",
}

// ValidCWStyleCategories 有效的风格类别列表
var ValidCWStyleCategories = []string{
	CWStyleMinimalist, CWStylePlayful, CWStyleTech,
	CWStyleAcademic, CWStyleOrganic, CWStyleImmersive,
}

// IsValidCWStyleCategory 检查风格类别是否有效
func IsValidCWStyleCategory(cat string) bool {
	for _, v := range ValidCWStyleCategories {
		if v == cat {
			return true
		}
	}
	return false
}

// ==================== 风格模板请求结构体 ====================

// CreateCWTemplateRequest 创建风格模板请求
type CreateCWTemplateRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	StyleCategory   string `json:"style_category"`
	PreviewImageURL string `json:"preview_image_url"`
	ColorScheme     string `json:"color_scheme"`
	CSSVariables    string `json:"css_variables"`
	SamplePages     string `json:"sample_pages"`
	PreviewURLs     string `json:"preview_urls"`
}

// UpdateCWTemplateRequest 更新风格模板请求
type UpdateCWTemplateRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	StyleCategory   string `json:"style_category"`
	PreviewImageURL string `json:"preview_image_url"`
	ColorScheme     string `json:"color_scheme"`
	CSSVariables    string `json:"css_variables"`
	SamplePages     string `json:"sample_pages"`
	PreviewURLs     string `json:"preview_urls"`
	IsActive        *bool  `json:"is_active"`
	SortOrder       *int   `json:"sort_order"`
}

// SaveAsMyTemplateRequest v137新增：保存为我的模板请求
// 从当前课件的导航栏+风格配置保存为个人可复用模板
type SaveAsMyTemplateRequest struct {
	Name          string `json:"name"`           // 模板名称（必填）
	Description   string `json:"description"`    // 描述（可选）
	StyleCategory string `json:"style_category"` // 风格类别（可选，默认从课件style_config提取）
}

// ==================== v139新增：AI 模板提取相关请求/响应 ====================

// ExtractTemplateRequest AI 模板提取请求
// 用户粘贴一段或多段 HTML 代码,AI 分析后生成草稿模板
type ExtractTemplateRequest struct {
	SamplePages []string `json:"sample_pages"` // 一个或多个 HTML 页面内容
	SourceType  string   `json:"source_type"`  // 来源类型: paste(粘贴) / upload(上传)
}

// ExtractTemplateResponse AI 模板提取响应
// 返回生成的草稿模板 ID,前端拿到 ID 后跳转到微调页面
type ExtractTemplateResponse struct {
	TemplateID       string `json:"template_id"`        // 草稿模板 ID
	SuggestedName    string `json:"suggested_name"`     // AI 建议的模板名称
	SuggestedDesc    string `json:"suggested_desc"`     // AI 建议的描述
	SuggestedCat     string `json:"suggested_category"` // AI 建议的风格分类
	ExtractionNotes  string `json:"extraction_notes"`   // AI 的提取观察备注
	Message          string `json:"message"`            // 提示消息
}

// AIExtractedTemplate AI 提取结果的 JSON 结构(从 AI 输出解析)
// 与 prompt_courseware_template_extract 提示词中定义的 JSON 格式完全对齐
type AIExtractedTemplate struct {
	SuggestedName        string                 `json:"suggested_name"`
	SuggestedDescription string                 `json:"suggested_description"`
	SuggestedCategory    string                 `json:"suggested_category"`
	ColorScheme          map[string]string      `json:"color_scheme"`
	CSSVariables         map[string]string      `json:"css_variables"`
	SamplePages          []string               `json:"sample_pages"`
	ExtractionNotes      string                 `json:"extraction_notes"`
	Error                string                 `json:"error,omitempty"` // AI 表示无法分析时设置
}

// ==================== v139新增：AI 模板微调相关请求/响应 ====================

// RefineTemplateRequest AI 模板微调请求(SSE 流式)
// 用户对当前模板提出修改意见,AI 生成更新后的样例页面+CSS变量
type RefineTemplateRequest struct {
	Instruction string `json:"instruction"` // 用户的修改指令(必填)
}

// AIRefinedTemplate AI 微调结果的 JSON 结构(从 AI 输出解析)
// 与 prompt_courseware_template_refine 提示词中定义的 JSON 格式完全对齐
type AIRefinedTemplate struct {
	ColorScheme       map[string]string `json:"color_scheme"`
	CSSVariables      map[string]string `json:"css_variables"`
	SamplePages       []string          `json:"sample_pages"`
	ChangeSummary     string            `json:"change_summary"`
	SuggestedCategory string            `json:"suggested_category"`
	Error             string            `json:"error,omitempty"`
}

// ==================== v139新增：草稿模板转正式模板请求 ====================

// PublishDraftRequest 草稿模板发布(转为正式模板)请求
// 用户在 AI 提取/微调过的草稿基础上,确认保存为某 scope 下的正式模板
type PublishDraftRequest struct {
	Name          string `json:"name"`            // 模板名称(必填)
	Description   string `json:"description"`     // 描述(可选)
	StyleCategory string `json:"style_category"`  // 风格类别(可选,默认沿用草稿)
	Scope         string `json:"scope"`           // 目标 scope: system/school/group/personal
	ScopeTargetID string `json:"scope_target_id"` // scope=school 时为学校ID, scope=group 时为教研组ID, 其他为空
}

// ==================== v139新增：微调历史相关 ====================

// RefineHistoryEntry 微调历史单条记录(存入 refine_history JSONB 数组)
// 每次成功微调前,把修改前的版本快照存入此结构
type RefineHistoryEntry struct {
	Timestamp           string   `json:"timestamp"`             // ISO 8601 时间戳
	UserInstruction     string   `json:"user_instruction"`      // 用户指令(便于回放)
	SamplePagesBefore   []string `json:"sample_pages_before"`   // 修改前的样例页面快照
	CSSVariablesBefore  string   `json:"css_variables_before"`  // 修改前的 CSS 变量 JSON 字符串
	ColorSchemeBefore   string   `json:"color_scheme_before"`   // 修改前的配色 JSON 字符串
	ChangeSummary       string   `json:"change_summary"`        // AI 给出的修改摘要
}

// RollbackToHistoryRequest 回退到历史版本请求
type RollbackToHistoryRequest struct {
	HistoryIndex int `json:"history_index"` // 历史快照在数组中的索引(0=最近一次微调前)
}

// ==================== v139新增：模板列表查询参数(扩展支持 scope/draft 过滤) ====================

// ListCWTemplatesParams 模板列表查询参数(repository 层使用)
type ListCWTemplatesParams struct {
	UserID         string   // 当前用户 ID(权限过滤用)
	SchoolID       string   // 当前用户所属学校 ID(可空)
	GroupIDs       []string // 当前用户所属教研组 ID 列表(可空)
	IncludeDrafts  bool     // 是否包含草稿(默认false,只看正式模板)
	OnlyMyDrafts   bool     // 仅看我的草稿(true时只返回 is_draft=true AND user_id=当前用户)
	ScopeFilter    string   // scope 过滤(空=全部可见,具体值=只看该scope)
	ActiveOnly     bool     // 是否只看激活的(默认true)
}

// ==================== v139.1 新增:发布目标查询响应 ====================
//
// 用于前端发布草稿弹窗 — 后端聚合返回当前用户可发布到的所有 scope 选项
// 前端据此动态渲染下拉选项,不再让用户手填 UUID

// PublishTargetSchool 学校发布目标(用户是该学校的管理员)
type PublishTargetSchool struct {
        Available bool   `json:"available"` // 是否可发布到学校(admin_user_id 命中即 true)
        SchoolID  string `json:"school_id"` // 学校 ID(用作 scope_target_id)
        Name      string `json:"name"`      // 学校名称(给用户看)
}

// PublishTargetGroup 教研组发布目标(用户是该组的 lead 或 backbone)
type PublishTargetGroup struct {
        ID         string `json:"id"`          // 教研组 ID(用作 scope_target_id)
        Name       string `json:"name"`        // 教研组名称
        SchoolName string `json:"school_name"` // 所属学校名称(辅助信息)
        Role       string `json:"role"`        // 当前用户在此组的角色: lead/backbone
}

// PublishTargetsResponse 发布目标聚合响应
//
// 前端根据各字段决定 PublishTab 哪些 scope 选项启用、哪些灰显
// personal: 任何登录用户都可发布
// system:   仅 role=admin 可发布
// school:   仅当 Available=true 显示该选项,scope_target_id 自动用 SchoolID
// groups:   只要数组非空就显示下拉选择
type PublishTargetsResponse struct {
        Personal struct {
                Available bool `json:"available"` // 任何登录用户都是 true
        } `json:"personal"`
        System struct {
                Available bool   `json:"available"` // role == admin 才为 true
                Reason    string `json:"reason"`    // false 时给前端展示的原因(可空)
        } `json:"system"`
        School PublishTargetSchool   `json:"school"`
        Groups []PublishTargetGroup  `json:"groups"`
}
