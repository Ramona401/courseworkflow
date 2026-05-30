package models

import (
"time"
)

// ==================== 课件工坊模型（对应 coursewares + courseware_pages + courseware_assets 表） ====================
// v0.42 变更：Courseware新增SourceType/SourceFilePath/EduModuleID/PublishedVersion字段
//            LessonPlanID改为*string(可空)；新增CreateCoursewareFromTopicRequest
//            CoursewareListItem/CoursewareDetailResponse同步扩展
//            来源类型常量+中文映射

// ==================== 课件主表模型 ====================

// Courseware 课件主记录（对应 coursewares 表）
// 状态机: draft → indexing → styling → generating → preview → confirmed → in_pipeline
type Courseware struct {
ID              string     `json:"id"`
LessonPlanID    *string    `json:"lesson_plan_id"`    // v0.42: 改为可空指针，支持非教案来源
UserID          string     `json:"user_id"`
Title           string     `json:"title"`
Subject         string     `json:"subject"`
Grade           string     `json:"grade"`
Status          string     `json:"status"`
StyleConfig     string     `json:"style_config"`
PageCount       int        `json:"page_count"`
IndexOverview   string     `json:"index_overview"`
LogoURL         string     `json:"logo_url"`
OrgName         string     `json:"org_name"`
NavTemplateHTML string     `json:"nav_template_html"`
PipelineID      *string    `json:"pipeline_id"`
SourceType      string     `json:"source_type"`       // v0.42: 来源类型(lesson_plan/ppt_upload/topic_direct/html_import)
SourceFilePath  string     `json:"source_file_path"`  // v0.42: PPT/文档上传时的文件路径
EduModuleID     string     `json:"edu_module_id"`     // v0.43预留: edu平台模块ID
PublishedVersion int       `json:"published_version"` // v0.43预留: 发布版本号
CreatedAt       *time.Time `json:"created_at"`
UpdatedAt       *time.Time `json:"updated_at"`
}

// ==================== 课件页面模型 ====================

// CoursewarePage 课件单页（对应 courseware_pages 表）
// 两层架构：层1技术索引(admin可见) + 层2用户方案(翻译后展示)
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
ID               string     `json:"id"`
CoursewareID     string     `json:"courseware_id"`
PageID           *string    `json:"page_id"`
PlaceholderID    string     `json:"placeholder_id"`
AssetType        string     `json:"asset_type"`
GenerationPrompt string     `json:"generation_prompt"`
OssURL           string     `json:"oss_url"`
FileSize         int64      `json:"file_size"`
MimeType         string     `json:"mime_type"`
Status           string     `json:"status"`
CreatedAt        *time.Time `json:"created_at"`
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

// CoursewareStatusOrder 状态顺序映射（用于回退校验：只能回退到序号更小的状态）
var CoursewareStatusOrder = map[string]int{
CoursewareStatusDraft:      0,
CoursewareStatusIndexing:   1,
CoursewareStatusStyling:    2,
CoursewareStatusGenerating: 3,
CoursewareStatusPreview:    4,
CoursewareStatusConfirmed:  5,
CoursewareStatusInPipeline: 6,
}

// ==================== v0.42: 课件来源类型常量 ====================

const (
CWSourceLessonPlan = "lesson_plan"  // 从教案创建
CWSourcePPTUpload  = "ppt_upload"   // 从PPT上传创建
CWSourceTopicDirect = "topic_direct" // 从主题直接创建
CWSourceDocUpload  = "doc_upload"   // 从Word文档上传创建
	CWSourceHTMLImport = "html_import"  // HTML导入
	CWSource3DSingle   = "3d_single"    // 3D互动单页
)

// CWSourceNameMap 课件来源类型中文名映射
var CWSourceNameMap = map[string]string{
CWSourceLessonPlan:  "教案生成",
CWSourcePPTUpload:   "PPT上传",
CWSourceTopicDirect: "主题创建",
CWSourceDocUpload:   "文档上传",
	CWSourceHTMLImport:  "HTML导入",
	CWSource3DSingle:    "3D互动单页",
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

// ==================== 方案结构预设常量 ====================

// SchemePreset 方案结构预设（v136新增）
type SchemePreset struct {
Key         string `json:"key"`         // 预设标识
Name        string `json:"name"`        // 显示名称
Emoji       string `json:"emoji"`       // 图标
Description string `json:"description"` // 描述
GradeHint   string `json:"grade_hint"`  // 适用学段提示
PageRange   string `json:"page_range"`  // 建议页数范围
PromptHint  string `json:"-"`           // 注入AI提示词的结构化指引（不暴露给前端）
}

// CoursewareSchemePresets 四种方案结构预设
var CoursewareSchemePresets = []SchemePreset{
{
Key:         "primary_fun",
Name:        "小学趣味版",
Emoji:       "🎈",
Description: "多互动、少文字、趣味导入，适合小学1-6年级",
GradeHint:   "小学",
PageRange:   "12-18页",
PromptHint: `【方案结构约束——小学趣味版】
- 总页数控制在12-18页
- 不要设计"学习目标"这种纯文字罗列页（小学生看不了长段文字）
- 封面页→趣味导入(1页,用故事/问题/游戏引入)→核心知识(2-3页,图文为主,文字精简)→互动练习(2-3页,拖拽/选择/游戏)→创意活动(1页)→趣味总结(1页,轻松回顾)
- 每页文字量尽量少,多用图片占位、动画、互动元素
- 交互类型偏向game/drag/quiz,避免大段static
- 复杂度主要在1-3之间,避免4-5`,
},
{
Key:         "middle_standard",
Name:        "初中标准版",
Emoji:       "📘",
Description: "知识讲解+例题+练习，适合初中7-9年级",
GradeHint:   "初中",
PageRange:   "18-25页",
PromptHint: `【方案结构约束——初中标准版】
- 总页数控制在18-25页
- 封面页→学习目标(1页,简洁列出3-5条)→知识讲解(3-5页,图文并茂)→例题解析(2-3页)→练习巩固(2-3页,选择/填空/拖拽)→知识小结(1页)→课后作业(1页)
- 学习目标页要有但不要太长,3-5条即可
- 例题和练习穿插,不要集中在最后
- 复杂度分布在2-4之间`,
},
{
Key:         "high_depth",
Name:        "高中深度版",
Emoji:       "🎓",
Description: "知识体系+重难点+拓展思考，适合高中10-12年级",
GradeHint:   "高中",
PageRange:   "22-30页",
PromptHint: `【方案结构约束——高中深度版】
- 总页数控制在22-30页
- 封面页→学习目标(1页)→知识体系梳理(4-6页,结构化呈现)→重难点突破(2-3页)→例题精讲(3-4页,含步骤分解)→综合练习(2-3页)→拓展思考(1页,深层问题)→总结归纳(1页)
- 知识点可以更密集,支持较长文本
- 图表和数据可视化占比高
- 复杂度分布在2-5之间,允许高复杂度页面`,
},
{
Key:         "auto",
Name:        "AI自动规划",
Emoji:       "🤖",
Description: "AI根据教案内容和学段自动决定最佳结构",
GradeHint:   "通用",
PageRange:   "AI自动",
PromptHint:  "", // 空字符串表示不注入额外约束，由AI自行规划
},
}

// GetSchemePresetByKey 根据key获取预设
func GetSchemePresetByKey(key string) *SchemePreset {
for i := range CoursewareSchemePresets {
if CoursewareSchemePresets[i].Key == key {
return &CoursewareSchemePresets[i]
}
}
return nil
}

// ==================== 请求结构体 ====================

// CreateCoursewareRequest 创建课件请求（从教案出发）
type CreateCoursewareRequest struct {
LessonPlanID string `json:"lesson_plan_id"`
Title        string `json:"title"`
}

// CreateCoursewareFromTopicRequest v0.42新增：从主题直接创建课件请求
type CreateCoursewareFromTopicRequest struct {
Subject    string `json:"subject"`     // 学科（必填）
Grade      string `json:"grade"`       // 年级（必填）
Topic      string `json:"topic"`       // 主题名称（必填）
PageRange  string `json:"page_range"`  // 期望页数范围（可选，如"15-25"）
ExtraNotes string `json:"extra_notes"` // 额外说明（可选）
}

// UpdateCoursewareRequest 更新课件基本信息请求
type UpdateCoursewareRequest struct {
Title string `json:"title"`
}

// UpdateCoursewareStyleRequest 保存风格选择请求
type UpdateCoursewareStyleRequest struct {
StyleConfig string `json:"style_config"`
}

// SaveStyleFullRequest 完整风格保存请求
type SaveStyleFullRequest struct {
TemplateID         string `json:"template_id"`
LogoURL            string `json:"logo_url"`
OrgName            string `json:"org_name"`
CustomPrimaryColor string `json:"custom_primary_color"`
}

// SaveNavTemplateRequest 保存导航栏HTML模板请求
type SaveNavTemplateRequest struct {
NavTemplateHTML string `json:"nav_template_html"`
}

// UploadLogoResponse Logo上传后的响应
type UploadLogoResponse struct {
URL string `json:"url"`
}

// UpdateCWPageIndexRequest 更新单页索引说明请求
type UpdateCWPageIndexRequest struct {
Title               string `json:"title"`
Purpose             string `json:"purpose"`
ContentSummary      string `json:"content_summary"`
InteractionType     string `json:"interaction_type"`
VisualFormat        string `json:"visual_format"`
MediaRequirements   string `json:"media_requirements"`
EstimatedComplexity int    `json:"estimated_complexity"`
}

// AddCWPageRequest 手动添加课件页面请求
type AddCWPageRequest struct {
Title               string `json:"title"`
Purpose             string `json:"purpose"`
ContentSummary      string `json:"content_summary"`
InteractionType     string `json:"interaction_type"`
VisualFormat        string `json:"visual_format"`
MediaRequirements   string `json:"media_requirements"`
EstimatedComplexity int    `json:"estimated_complexity"`
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

// RollbackStatusRequest v136新增：步骤回退请求
type RollbackStatusRequest struct {
TargetStatus string `json:"target_status"` // 目标状态: draft/indexing/styling/generating/preview
}

// GenerateIndexRequest v136新增：生成索引请求（含可选预设）
type GenerateIndexRequest struct {
Preset string `json:"preset"` // 可选: primary_fun/middle_standard/high_depth/auto/空
}

// RefineIndexRequest v136新增：AI修改方案请求
type RefineIndexRequest struct {
Feedback string `json:"feedback"` // 用户对方案的修改意见
}

// ==================== 响应结构体 ====================

// CoursewareListItem 课件列表单条
type CoursewareListItem struct {
ID              string     `json:"id"`
LessonPlanID    *string    `json:"lesson_plan_id"`     // v0.42: 改为可空指针
LessonPlanTitle string     `json:"lesson_plan_title"`
Title           string     `json:"title"`
Subject         string     `json:"subject"`
Grade           string     `json:"grade"`
Status          string     `json:"status"`
StatusName      string     `json:"status_name"`
PageCount       int        `json:"page_count"`
PipelineID      *string    `json:"pipeline_id"`
SourceType      string     `json:"source_type"`        // v0.42: 来源类型
SourceName      string     `json:"source_name"`        // v0.42: 来源类型中文名
CreatedAt       *time.Time `json:"created_at"`
UpdatedAt       *time.Time `json:"updated_at"`
}

// CoursewareDetailResponse 课件详情响应
type CoursewareDetailResponse struct {
ID              string            `json:"id"`
LessonPlanID    *string           `json:"lesson_plan_id"`    // v0.42: 改为可空指针
LessonPlanTitle string            `json:"lesson_plan_title"`
UserID          string            `json:"user_id"`
Title           string            `json:"title"`
Subject         string            `json:"subject"`
Grade           string            `json:"grade"`
Status          string            `json:"status"`
StatusName      string            `json:"status_name"`
StyleConfig     string            `json:"style_config"`
PageCount       int               `json:"page_count"`
IndexOverview   string            `json:"index_overview"`
LogoURL         string            `json:"logo_url"`
OrgName         string            `json:"org_name"`
NavTemplateHTML string            `json:"nav_template_html"`
PipelineID      *string           `json:"pipeline_id"`
SourceType      string            `json:"source_type"`       // v0.42: 来源类型
SourceName      string            `json:"source_name"`       // v0.42: 来源类型中文名
Pages           []*CoursewarePage `json:"pages"`
CreatedAt       *time.Time        `json:"created_at"`
UpdatedAt       *time.Time        `json:"updated_at"`
}

// CoursewareListResponse 课件列表响应
type CoursewareListResponse struct {
Coursewares []*CoursewareListItem `json:"coursewares"`
Total       int                   `json:"total"`
}
