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
	ReviewStatus        string     `json:"review_status"`         // draft/approved/archived
	CreatedAt           *time.Time `json:"created_at"`
	UpdatedAt           *time.Time `json:"updated_at"`
}

// ==================== 课件组件类型常量（6大类） ====================

const (
	CWCompTypeLayout      = "layout"      // 布局模板
	CWCompTypeInteraction = "interaction"  // 交互功能
	CWCompType3D          = "3d"           // 3D/动画
	CWCompTypeAnimation   = "animation"    // 动画效果
	CWCompTypeDataViz     = "data_viz"     // 数据可视化
	CWCompTypeMultimedia  = "multimedia"   // 多媒体容器
	CWCompTypeStyle       = "style"        // 样式主题
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
	CWInteractionLevelDisplay    = 1 // 纯展示
	CWInteractionLevelClick      = 2 // 简单点击
	CWInteractionLevelDrag       = 3 // 拖拽/选择
	CWInteractionLevelMultiStep  = 4 // 多步交互
	CWInteractionLevelComplex3D  = 5 // 复杂模拟/3D
)

// 视觉形式（VF维度）
const (
	CWVisualFormatTimeline  = "TL" // 时间线
	CWVisualFormatCompare   = "CP" // 对比
	CWVisualFormatCard      = "CG" // 卡片
	CWVisualFormatTable     = "TB" // 表格
	CWVisualFormatChart     = "CH" // 图表
	CWVisualFormatFullscreen = "FS" // 全屏
	CWVisualFormatGrid      = "GD" // 网格
	CWVisualFormatFreeform  = "FR" // 自由版式
)

// 技术标签（TG维度）
const (
	CWTechTagCSS    = "CSS" // 纯样式
	CWTechTagJS     = "JS"  // 基础脚本
	CWTechTagSVG    = "SVG" // 矢量图形
	CWTechTagCanvas = "CV"  // Canvas画布
	CWTechTagThreeJS = "TJ" // Three.js三维
	CWTechTagAnim   = "AN"  // 动画
)

// ==================== 请求结构体 ====================

// CreateCWComponentRequest 创建课件组件请求
type CreateCWComponentRequest struct {
	Name              string `json:"name"`               // 组件名称（必填）
	Description       string `json:"description"`        // 组件描述
	ComponentType     string `json:"component_type"`     // 组件大类（必填）
	CodeContent       string `json:"code_content"`       // 完整HTML/CSS/JS代码（必填）
	PreviewImageURL   string `json:"preview_image_url"`  // 预览截图URL
	PreviewHTML       string `json:"preview_html"`       // 精简版预览HTML
	SubjectScope      string `json:"subject_scope"`      // 适用学科
	GradeScope        string `json:"grade_scope"`        // 适用学段
	TechDependencies  string `json:"tech_dependencies"`  // 技术依赖JSON
	Tags              string `json:"tags"`               // 自定义标签JSON
}

// UpdateCWComponentRequest 更新课件组件请求
type UpdateCWComponentRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	ComponentType     string `json:"component_type"`
	CodeContent       string `json:"code_content"`
	PreviewImageURL   string `json:"preview_image_url"`
	PreviewHTML       string `json:"preview_html"`
	SubjectScope      string `json:"subject_scope"`
	GradeScope        string `json:"grade_scope"`
	TechDependencies  string `json:"tech_dependencies"`
	Tags              string `json:"tags"`
	IsActive          *bool  `json:"is_active"`
	ReviewStatus      string `json:"review_status"`
}

// MatchCWComponentsRequest 课件组件匹配请求
// 根据交互类型+视觉形式+学科+学段+复杂度匹配最合适的组件
type MatchCWComponentsRequest struct {
	ComponentType    string `json:"component_type"`     // 组件大类（可选）
	SubjectScope     string `json:"subject_scope"`      // 学科
	GradeScope       string `json:"grade_scope"`        // 学段
	InteractionLevel int    `json:"interaction_level"`  // 交互复杂度 1-5
	VisualFormat     string `json:"visual_format"`      // 视觉形式代码
	TechTag          string `json:"tech_tag"`           // 技术标签
	Limit            int    `json:"limit"`              // 返回数量上限（默认3）
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
	ID              string `json:"id"`
	Name            string `json:"name"`
	ComponentType   string `json:"component_type"`
	CodeContent     string `json:"code_content"`     // 完整代码
	PreviewHTML     string `json:"preview_html"`
	ComponentIndex  string `json:"component_index"`
	InteractionLevel *int  `json:"interaction_level"`
}

// ==================== 风格模板模型（对应 courseware_templates 表） ====================

// CoursewareTemplate 课件风格模板
// 预定义的视觉风格配置，包含配色、CSS变量、样例页面
type CoursewareTemplate struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	StyleCategory   string     `json:"style_category"`    // minimalist/playful/tech/academic/organic
	PreviewImageURL string     `json:"preview_image_url"`
	ColorScheme     string     `json:"color_scheme"`      // 配色方案JSONB
	CSSVariables    string     `json:"css_variables"`     // CSS变量JSONB
	SamplePages     string     `json:"sample_pages"`      // 样例页面HTML数组JSONB
	IsActive        bool       `json:"is_active"`
	SortOrder       int        `json:"sort_order"`
	CreatedAt       *time.Time `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
}

// ==================== 风格类别常量 ====================

const (
	CWStyleMinimalist = "minimalist" // 简约清新
	CWStylePlayful    = "playful"    // 活泼趣味
	CWStyleTech       = "tech"       // 科技感
	CWStyleAcademic   = "academic"   // 学术严谨
	CWStyleOrganic    = "organic"    // 自然有机
)

// CWStyleCategoryNameMap 风格类别中文名映射
var CWStyleCategoryNameMap = map[string]string{
	CWStyleMinimalist: "简约清新",
	CWStylePlayful:    "活泼趣味",
	CWStyleTech:       "科技感",
	CWStyleAcademic:   "学术严谨",
	CWStyleOrganic:    "自然有机",
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
	IsActive        *bool  `json:"is_active"`
	SortOrder       *int   `json:"sort_order"`
}
