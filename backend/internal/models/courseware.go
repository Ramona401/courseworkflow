package models

import (
	"strings"
	"time"
)

// ==================== 课件工坊模型（对应 coursewares + courseware_pages + courseware_assets 表） ====================
// v0.42 变更：Courseware新增SourceType/SourceFilePath/EduModuleID/PublishedVersion字段
//            LessonPlanID改为*string(可空)；新增CreateCoursewareFromTopicRequest
//            CoursewareListItem/CoursewareDetailResponse同步扩展
//            来源类型常量+中文映射
// 风格锚点(VAOCI 课程级风格一致性)轮1变更：Courseware新增StyleAnchorAssetID/StyleAnchorVAOCI两字段
//            对应 coursewares 表新增列 style_anchor_asset_id(uuid,可空)/style_anchor_vaoci(text,可空)
//            NULL=未设锚点；设锚点后该课件所有页生成配图自动套用此风格DNA
// 视频锚点轮(本轮)变更：CoursewareAsset 新增 Metadata 字段，打通 courseware_assets.metadata(jsonb) 列读写
//            视频资产可在 metadata 里存 source_frame_asset_id（首帧图溯源）；
//            视频上传也用它接通 ffprobe 元数据（duration/width/height/codec/fps/bit_rate）。
//            Metadata 为原始 JSON 字符串，空串语义=NULL（仓储层 nullIfEmptyJSON 负责空串转 NULL，
//            避免空串直接写 jsonb 列报错）。
// 课程知识库轮（本轮）变更：
//            - CreateCoursewareFromTopicRequest 新增可选字段 KPCodes（选中的课标知识点编码数组）；
//              为空时完全兼容原有"纯主题规划"逻辑，非空时启用"难度自动适配"。
//            - 新增 CurriculumKP（课标骨架层）/ TextbookUnit（教材实例层）两个只读模型，
//              对应 curriculum_standards / textbook_units 两张表。
// 阶段1（课件审核与协作·发布与共享）变更：
//            - Courseware 新增 PublishState/ReviewLevel/ReviewSchoolID/CodeShareScope 四字段，
//              对应 coursewares 表新增列 publish_state/review_level/review_school_id/code_share_scope。
//              这是与既有 status 生产状态机【正交】的"发布/审核维度"——绝不改动 status，
//              一个课件可同时是"已生成(status=preview)"且"已共享(publish_state=published_shared)"。
//            - 新增发布态常量 CWPublish* + 中文映射；新增代码开放范围常量 CWCodeShare* + 中文映射。
//            - CoursewareListItem/CoursewareDetailResponse 同步加上述字段，供前端展示发布徽章/代码范围。
//            - 新增 SharedCoursewareListItem（共享课件库列表项，含作者名/学校名/能否复制源码）。
//            - 新增发布/撤回/设代码范围三个请求体。
// 阶段4（课件审核与协作·集体备课）变更：
//            - Courseware / CoursewareDetailResponse 新增 CollabState 字段，
//              对应 coursewares 表新增列 collab_state（idle/in_session），与 status/publish_state 三者正交。
//            - 新增集体备课态常量 CWCollab* + 中文映射 + 校验函数。
//            - 版本来源枚举新增 CWPageVersionSourceCollab（集体备课期间微调前的旧版）。
//            - 新增 CollabMember（参与者实体，对应 courseware_collab_members 表）、
//              CollabMemberView（带用户名的列表项）、CollabStatusResponse（状态+参与者列表）、
//              StartCollabRequest / AddCollabMemberRequest 请求体。
//            - 设计原则：最小增量。线下集体备课只做"标记态 + 共享微调权"，
//              议课走现有 courseware_annotations 批注，留痕走现有 courseware_page_versions 版本快照。

// ==================== 课件主表模型 ====================

// Courseware 课件主记录（对应 coursewares 表）
// 状态机: draft → indexing → styling → generating → preview → confirmed → in_pipeline
type Courseware struct {
	ID           string  `json:"id"`
	LessonPlanID *string `json:"lesson_plan_id"` // v0.42: 改为可空指针，支持非教案来源
	UserID       string  `json:"user_id"`
	Title        string  `json:"title"`
	Subject      string  `json:"subject"`
	Grade        string  `json:"grade"`
	// EducationDomain 课件创建时确定的资源教育域快照，创建后不随用户换校或组织变化。
	EducationDomain  string  `json:"education_domain"`
	Status           string  `json:"status"`
	StyleConfig      string  `json:"style_config"`
	PageCount        int     `json:"page_count"`
	IndexOverview    string  `json:"index_overview"`
	LogoURL          string  `json:"logo_url"`
	OrgName          string  `json:"org_name"`
	NavTemplateHTML  string  `json:"nav_template_html"`
	PipelineID       *string `json:"pipeline_id"`
	SourceType       string  `json:"source_type"`       // v0.42: 来源类型(lesson_plan/ppt_upload/topic_direct/html_import)
	SourceFilePath   string  `json:"source_file_path"`  // v0.42: PPT/文档上传时的文件路径
	EduModuleID      string  `json:"edu_module_id"`     // v0.43预留: edu平台模块ID
	PublishedVersion int     `json:"published_version"` // v0.43预留: 发布版本号
	// ---- 风格锚点字段（VAOCI 课程级风格一致性，轮1新增）----
	// StyleAnchorAssetID 风格锚点对应的图片资产ID(指向 courseware_assets.id)；nil=未设锚点
	StyleAnchorAssetID *string `json:"style_anchor_asset_id"`
	// StyleAnchorVAOCI 锚点图的 VAOCI 索引文本(多模态AI读图提取结果)；空=未设锚点
	StyleAnchorVAOCI string `json:"style_anchor_vaoci"`
	// KPCodes 课程知识库轮新增：从主题创建时勾选的课标知识点编码数组的JSON文本
	// 对应 coursewares.kp_codes(jsonb) 列；空串=未勾选。生成索引时读出注入难度适配约束。
	KPCodes string `json:"kp_codes"`
	// ---- 阶段1：发布/审核维度字段（与 status 生产状态机正交，绝不混用）----
	// PublishState 发布态：private/published_personal/submitted/approved/published_shared/revision
	//   对应 coursewares.publish_state 列，默认 private（私有）。详见 CWPublish* 常量。
	PublishState string `json:"publish_state"`
	// ReviewLevel 审核层级进度：0未提交 / 1=L1教研组通过 / 2=L2学校通过
	//   对应 coursewares.review_level 列，默认 0。
	ReviewLevel int `json:"review_level"`
	// ReviewSchoolID 提交审核时反查的作者所属学校ID（供 L2 学校审核按校过滤）
	//   对应 coursewares.review_school_id 列，可空指针；nil=未提交审核。
	ReviewSchoolID *string `json:"review_school_id"`
	// CodeShareScope 源代码开放范围（产权保护，独立于"课件可见范围"）：
	//   none/group/school/region/public，对应 coursewares.code_share_scope 列，默认 none。详见 CWCodeShare* 常量。
	CodeShareScope string `json:"code_share_scope"`
	// ---- 阶段4：集体备课维度字段（与 status/publish_state 正交）----
	// CollabState 集体备课态：idle（非集体备课，仅作者可微调）/ in_session（集体备课进行中，参与者也获共享微调权）
	//   对应 coursewares.collab_state 列，默认 idle。详见 CWCollab* 常量。
	CollabState string     `json:"collab_state"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// ==================== 课件页面模型 ====================

// CoursewarePage 课件单页（对应 courseware_pages 表）
// 两层架构：层1技术索引(admin可见) + 层2用户方案(翻译后展示)
type CoursewarePage struct {
	ID           string `json:"id"`
	CoursewareID string `json:"courseware_id"`
	PageNumber   int    `json:"page_number"`
	// ---- 层2：用户友好方案字段 ----
	Title               string `json:"title"`
	Purpose             string `json:"purpose"`
	ContentSummary      string `json:"content_summary"`
	InteractionType     string `json:"interaction_type"`
	VisualFormat        string `json:"visual_format"`
	MediaRequirements   string `json:"media_requirements"`
	EstimatedComplexity int    `json:"estimated_complexity"`
	// ---- 层1：AOCI技术索引 ----
	PageIndex           string `json:"page_index"`
	IdxCognitiveLevel   int    `json:"idx_cognitive_level"`
	IdxInteractionLevel int    `json:"idx_interaction_level"`
	IdxVisualFormat     string `json:"idx_visual_format"`
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
	ID               string  `json:"id"`
	CoursewareID     string  `json:"courseware_id"`
	PageID           *string `json:"page_id"`
	PlaceholderID    string  `json:"placeholder_id"`
	AssetType        string  `json:"asset_type"`
	GenerationPrompt string  `json:"generation_prompt"`
	OssURL           string  `json:"oss_url"`
	PublicOSSURL     string  `json:"public_oss_url"`
	FileSize         int64   `json:"file_size"`
	MimeType         string  `json:"mime_type"`
	// Metadata 资产元数据的原始 JSON 字符串（对应 courseware_assets.metadata jsonb 列）。
	// 视频锚点轮新增：
	//   - 视频资产：可存 {"source_frame_asset_id":"<首帧图asset_id>"} 做首帧溯源；
	//   - 视频上传：可存 ffprobe 元数据 {"duration","width","height","codec","fps","bit_rate"}。
	// 空串语义 = NULL（仓储层 nullIfEmptyJSON 负责空串↔NULL 转换，避免空串写 jsonb 报错）。
	Metadata  string     `json:"metadata"`
	Status    string     `json:"status"`
	CreatedAt *time.Time `json:"created_at"`
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

// ==================== 阶段1：课件发布态常量（publish_state，与 status 正交）====================
//
// 发布态描述"课件在共享/审核流程中的位置"，与生产态 status（draft→...→preview→confirmed）
// 完全正交：status 管"课件做没做完"，publish_state 管"做完后有没有发布/审核/共享"。
// 默认 private（私有），等于"只有作者自己能看，未进入任何共享或审核流程"。

const (
	CWPublishPrivate           = "private"            // 私有（默认）：仅作者可见
	CWPublishPublishedPersonal = "published_personal" // 个人发布：作者标记完成，暂未共享给他人
	CWPublishSubmitted         = "submitted"          // 已提交审核：进入教研组/学校审核流，提交后锁定编辑
	CWPublishApproved          = "approved"           // 审核通过：待发布共享
	CWPublishPublishedShared   = "published_shared"   // 已共享发布：同校/同组可见
	CWPublishRevision          = "revision"           // 审核退回：需修改后重新提交
)

// CWPublishStateNameMap 发布态中文名映射（前端发布徽章用）
var CWPublishStateNameMap = map[string]string{
	CWPublishPrivate:           "私有",
	CWPublishPublishedPersonal: "个人发布",
	CWPublishSubmitted:         "审核中",
	CWPublishApproved:          "审核通过",
	CWPublishPublishedShared:   "已共享",
	CWPublishRevision:          "已退回",
}

// IsValidPublishState 校验发布态取值是否合法
func IsValidPublishState(s string) bool {
	switch s {
	case CWPublishPrivate, CWPublishPublishedPersonal, CWPublishSubmitted,
		CWPublishApproved, CWPublishPublishedShared, CWPublishRevision:
		return true
	}
	return false
}

// ==================== 阶段1：源代码开放范围常量（code_share_scope，产权分级）====================
//
// 代码范围独立于"课件可见范围"：可见范围决定谁能【看渲染效果】，代码范围决定谁能【复制源码】。
// 举例：课件 published_shared 到全校，但 code_share_scope=none → 全校能看能放映能批注，
//       但没有"复制到我的"按钮，源码受保护。作者/教研组/学校可独立行使产权决定开放到哪一级。
// 默认 none（不开放源码）。

const (
	CWCodeShareNone   = "none"   // 不开放源码（默认）：别人只能看渲染效果，不能复制
	CWCodeShareGroup  = "group"  // 仅本教研组可复制源码
	CWCodeShareSchool = "school" // 仅本校可复制源码
	CWCodeShareRegion = "region" // 区域内可复制源码
	CWCodeSharePublic = "public" // 所有可见者均可复制源码
)

// CWCodeShareScopeNameMap 代码开放范围中文名映射（前端代码范围选择器用）
var CWCodeShareScopeNameMap = map[string]string{
	CWCodeShareNone:   "不开放源码",
	CWCodeShareGroup:  "本教研组可复制",
	CWCodeShareSchool: "本校可复制",
	CWCodeShareRegion: "本区域可复制",
	CWCodeSharePublic: "所有人可复制",
}

// IsValidCodeShareScope 校验代码开放范围取值是否合法
func IsValidCodeShareScope(s string) bool {
	switch s {
	case CWCodeShareNone, CWCodeShareGroup, CWCodeShareSchool,
		CWCodeShareRegion, CWCodeSharePublic:
		return true
	}
	return false
}

// ==================== 阶段4：集体备课态常量（collab_state，与 status/publish_state 正交）====================
//
// 集体备课是"线下一群老师聚在一起，对着同一个课件当场改、当场议"的场景。
// 系统只做最小增量三件事：① 标记态 collab_state ② 共享微调权（参与者也能微调）③ 留痕（走现有版本快照）。
// 不做实时多人在线协同（无 WebSocket/OT/CRDT），不做讨论楼层（议课直接走现有页级批注）。
//
// collab_state 与 status（生产态）、publish_state（发布态）三者正交：
//   一个课件可以同时是"已生成(status=preview)"+"已共享(publish_state=published_shared)"+"集体备课中(collab_state=in_session)"。
// 默认 idle（非集体备课，仅作者可微调）。

const (
	CWCollabIdle      = "idle"       // 非集体备课态（默认）：仅作者可微调该课件
	CWCollabInSession = "in_session" // 集体备课进行中：被拉进参与者名单的人也获得对该课件的共享微调权
)

// CWCollabStateNameMap 集体备课态中文名映射（前端徽章用）
var CWCollabStateNameMap = map[string]string{
	CWCollabIdle:      "未集体备课",
	CWCollabInSession: "集体备课中",
}

// IsValidCollabState 校验集体备课态取值是否合法
func IsValidCollabState(s string) bool {
	switch s {
	case CWCollabIdle, CWCollabInSession:
		return true
	}
	return false
}

// ==================== v0.42: 课件来源类型常量 ====================

const (
	CWSourceLessonPlan  = "lesson_plan"  // 从教案创建
	CWSourcePPTUpload   = "ppt_upload"   // 从PPT上传创建
	CWSourceTopicDirect = "topic_direct" // 从主题直接创建
	CWSourceDocUpload   = "doc_upload"   // 从Word文档上传创建
	CWSourceHTMLImport  = "html_import"  // HTML导入
	CWSource3DSingle    = "3d_single"    // 3D互动单页
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
	CWAssetTypeAudio        = "audio"
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
	{
		Key:         "custom",
		Name:        "自定义",
		Emoji:       "✏️",
		Description: "自己描述课件结构要求，AI按您的描述规划",
		GradeHint:   "通用",
		PageRange:   "自定义",
		PromptHint:  "", // 自定义预设的PromptHint由前端传入custom_prompt_hint字段，运行时动态注入
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

// ResolveSchemePromptHint 根据预设key和可选的自定义提示词，返回最终应注入AI的结构化指引文本。
// 逻辑：
//   - preset为空或"auto" → 返回空串（AI自行规划）
//   - preset为"custom"   → 返回customHint（老师自己写的描述），customHint为空则返回空串
//   - 其余系统预设        → 返回该预设的PromptHint（忽略customHint）
//
// 此函数是四条索引生成链路（教案/主题/PPT/Doc）注入预设提示词的统一出口，
// 替代原先各处直接调 GetSchemePresetByKey(preset).PromptHint 的分散逻辑。
func ResolveSchemePromptHint(preset string, customHint string) string {
	if preset == "" {
		return ""
	}
	if preset == "custom" {
		if strings.TrimSpace(customHint) == "" {
			return ""
		}
		// 把老师的自定义描述包装成与系统预设相同的格式，方便AI识别
		return "【方案结构约束——老师自定义】\n" + strings.TrimSpace(customHint)
	}
	obj := GetSchemePresetByKey(preset)
	if obj != nil {
		return obj.PromptHint
	}
	return ""
}

// ==================== 请求结构体 ====================

// CreateCoursewareRequest 创建课件请求（从教案出发）
type CreateCoursewareRequest struct {
	LessonPlanID string `json:"lesson_plan_id"`
	Title        string `json:"title"`
}

// CreateCoursewareFromTopicRequest v0.42新增：从主题直接创建课件请求
// 课程知识库轮新增 KPCodes：选中的课标知识点编码数组（可选）。
//   - 为空：完全兼容原有"纯主题规划"逻辑，AI 自行决定难度；
//   - 非空：服务层据此查 curriculum_standards 注入"难度自动适配约束段落"，
//     使生成的课件难度严格贴合课标对该年级该知识点的深度要求。
type CreateCoursewareFromTopicRequest struct {
	Subject    string   `json:"subject"`     // 学科（必填）
	Grade      string   `json:"grade"`       // 年级（必填）
	Topic      string   `json:"topic"`       // 主题名称（必填）
	PageRange  string   `json:"page_range"`  // 期望页数范围（可选，如"15-25"）
	ExtraNotes string   `json:"extra_notes"` // 额外说明（可选）
	KPCodes    []string `json:"kp_codes"`    // 课程知识库轮新增：选中的课标知识点编码数组（可选）
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
	Preset           string `json:"preset"`             // 可选: primary_fun/middle_standard/high_depth/auto/custom/空
	CustomPromptHint string `json:"custom_prompt_hint"` // 仅preset=custom时使用：老师自定义的课件结构描述文本
}

// RefineIndexRequest v136新增：AI修改方案请求
type RefineIndexRequest struct {
	Feedback string `json:"feedback"` // 用户对方案的修改意见
}

// ==================== 阶段1：发布/共享请求结构体 ====================

// PublishCoursewareRequest 发布课件请求（设置发布态）
// target 取值：published_personal（个人发布）/ published_shared（共享发布，需先审核通过或作者直接共享）
//
//	实际允许的目标态由 service 层按当前发布态 + 角色裁决，本结构只承载意图。
type PublishCoursewareRequest struct {
	Target string `json:"target"` // 目标发布态
}

// SetCodeShareScopeRequest 设置源代码开放范围请求（产权分级）
type SetCodeShareScopeRequest struct {
	CodeShareScope string `json:"code_share_scope"` // none/group/school/region/public
}

// ==================== 阶段4：集体备课请求结构体 ====================

// StartCollabRequest 发起集体备课请求（可选携带首批参与者）
// Members 为可选的参与者用户ID数组：发起集体备课时可一次性拉入多人，也可发起后再逐个加。
type StartCollabRequest struct {
	Members []string `json:"members"` // 可选：首批参与者用户ID数组（空则只发起，后续再加人）
}

// AddCollabMemberRequest 添加集体备课参与者请求
type AddCollabMemberRequest struct {
	UserID string `json:"user_id"` // 要拉入的参与者用户ID（必填）
}

// ==================== 响应结构体 ====================

// CoursewareListItem 课件列表单条
type CoursewareListItem struct {
	ID              string  `json:"id"`
	LessonPlanID    *string `json:"lesson_plan_id"` // v0.42: 改为可空指针
	LessonPlanTitle string  `json:"lesson_plan_title"`
	Title           string  `json:"title"`
	Subject         string  `json:"subject"`
	Grade           string  `json:"grade"`
	// EducationDomain 课件所属资源教育域，供列表展示和调用方确认隔离边界。
	EducationDomain string  `json:"education_domain"`
	Status          string  `json:"status"`
	StatusName      string  `json:"status_name"`
	PageCount       int     `json:"page_count"`
	PipelineID      *string `json:"pipeline_id"`
	SourceType      string  `json:"source_type"` // v0.42: 来源类型
	SourceName      string  `json:"source_name"` // v0.42: 来源类型中文名
	// ---- 阶段1：发布/共享维度（供"我的课件"列表展示发布徽章、代码范围）----
	PublishState     string     `json:"publish_state"`      // 发布态
	PublishStateName string     `json:"publish_state_name"` // 发布态中文名
	ReviewLevel      int        `json:"review_level"`       // 审核层级进度
	CodeShareScope   string     `json:"code_share_scope"`   // 源代码开放范围
	CreatedAt        *time.Time `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

// SharedCoursewareListItem 共享课件库列表单条（阶段1新增）
// 用于"共享课件库"页面：列出他人共享给我（同校/同组）的课件，带作者名、学校名、
// 以及"当前登录者能否复制此课件源码"的标记（前端据此显隐"复制到我的"按钮）。
type SharedCoursewareListItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Subject string `json:"subject"`
	Grade   string `json:"grade"`
	// EducationDomain 共享课件所属资源教育域，供共享市场执行跨域隔离。
	EducationDomain  string     `json:"education_domain"`
	PageCount        int        `json:"page_count"`
	SourceType       string     `json:"source_type"`
	SourceName       string     `json:"source_name"`
	AuthorID         string     `json:"author_id"`     // 作者用户ID
	AuthorName       string     `json:"author_name"`   // 作者显示名
	SchoolName       string     `json:"school_name"`   // 作者所属学校名
	PublishState     string     `json:"publish_state"` // 发布态（共享库里通常为 published_shared）
	PublishStateName string     `json:"publish_state_name"`
	CodeShareScope   string     `json:"code_share_scope"` // 源代码开放范围
	CanCopy          bool       `json:"can_copy"`         // 当前登录者能否复制源码（按 code_share_scope + 归属关系裁决）
	CreatedAt        *time.Time `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at"`
}

// SharedCoursewareListResponse 共享课件库列表响应
type SharedCoursewareListResponse struct {
	Coursewares []*SharedCoursewareListItem `json:"coursewares"`
	Total       int                         `json:"total"`
}

// CoursewareDetailResponse 课件详情响应
type CoursewareDetailResponse struct {
	ID              string  `json:"id"`
	LessonPlanID    *string `json:"lesson_plan_id"` // v0.42: 改为可空指针
	LessonPlanTitle string  `json:"lesson_plan_title"`
	UserID          string  `json:"user_id"`
	Title           string  `json:"title"`
	Subject         string  `json:"subject"`
	Grade           string  `json:"grade"`
	// EducationDomain 课件创建时的资源教育域快照，具体运行链必须以此字段为准。
	EducationDomain string  `json:"education_domain"`
	Status          string  `json:"status"`
	StatusName      string  `json:"status_name"`
	StyleConfig     string  `json:"style_config"`
	PageCount       int     `json:"page_count"`
	IndexOverview   string  `json:"index_overview"`
	LogoURL         string  `json:"logo_url"`
	OrgName         string  `json:"org_name"`
	NavTemplateHTML string  `json:"nav_template_html"`
	PipelineID      *string `json:"pipeline_id"`
	SourceType      string  `json:"source_type"` // v0.42: 来源类型
	SourceName      string  `json:"source_name"` // v0.42: 来源类型中文名
	// ---- 风格锚点字段（轮1新增，供前端读取当前锚点状态）----
	StyleAnchorAssetID *string `json:"style_anchor_asset_id"` // nil=未设锚点
	StyleAnchorVAOCI   string  `json:"style_anchor_vaoci"`    // 锚点VAOCI索引文本
	StyleAnchorURL     string  `json:"style_anchor_url"`      // 锚点图公网URL（轮3：供前端跨页显示缩略图，优先OSS地址）
	// KPCodes 课程知识库轮新增：本课件勾选的课标知识点编码数组JSON文本，供前端生成索引时回传
	KPCodes string `json:"kp_codes"`
	// ---- 阶段1：发布/审核维度（供详情页展示发布徽章、代码范围、审核进度）----
	PublishState     string  `json:"publish_state"`      // 发布态
	PublishStateName string  `json:"publish_state_name"` // 发布态中文名
	ReviewLevel      int     `json:"review_level"`       // 审核层级进度
	ReviewSchoolID   *string `json:"review_school_id"`   // 提交审核时的作者学校ID（可空）
	CodeShareScope   string  `json:"code_share_scope"`   // 源代码开放范围
	// ---- 阶段4：集体备课维度（供详情页展示集体备课徽章、判断是否可发起/结束）----
	CollabState string `json:"collab_state"` // 集体备课态：idle/in_session
	// CollabMemberCount 集体备课在场参与者数（阶段4治本补丁）：
	//   前端据此区分真集体备课(in_session且有人)与幽灵会话(in_session但0人)，
	//   由 GetCourseware 用 repository.CountCollabMembers 填充。
	CollabMemberCount int               `json:"collab_member_count"`
	Pages             []*CoursewarePage `json:"pages"`
	CreatedAt         *time.Time        `json:"created_at"`
	UpdatedAt         *time.Time        `json:"updated_at"`
}

// CoursewareListResponse 课件列表响应
type CoursewareListResponse struct {
	Coursewares []*CoursewareListItem `json:"coursewares"`
	Total       int                   `json:"total"`
}

// ==================== 阶段4：集体备课模型（对应 courseware_collab_members 表）====================

// CollabMember 集体备课参与者记录（对应 courseware_collab_members 表）
// 一行 = 某课件的一场集体备课里的一名参与者。线下聚集时由发起者手动拉入。
type CollabMember struct {
	ID           string     `json:"id"`
	CoursewareID string     `json:"courseware_id"` // 归属课件
	UserID       string     `json:"user_id"`       // 参与者用户ID
	InitiatorID  string     `json:"initiator_id"`  // 本场集体备课发起者用户ID（冗余存，便于审计）
	AddedAt      *time.Time `json:"added_at"`      // 加入时间
}

// CollabMemberView 集体备课参与者列表项（带用户名，供前端展示）
// 在 CollabMember 基础上 LEFT JOIN users 补 UserName，前端无需另查用户名。
type CollabMemberView struct {
	ID           string     `json:"id"`
	CoursewareID string     `json:"courseware_id"`
	UserID       string     `json:"user_id"`
	UserName     string     `json:"user_name"` // 参与者显示名（display_name 优先，回退 username）
	InitiatorID  string     `json:"initiator_id"`
	AddedAt      *time.Time `json:"added_at"`
}

// CollabStatusResponse 集体备课状态响应（查 GET /collab 返回）
// 一次返回：当前集体备课态、本课件作者ID（前端据此判断"我是不是作者，能否发起/加人"）、
// 当前请求者能否在本课件微调（canEdit，前端据此显隐微调入口）、全部参与者列表。
type CollabStatusResponse struct {
	CoursewareID string              `json:"courseware_id"`
	CollabState  string              `json:"collab_state"` // idle/in_session
	OwnerID      string              `json:"owner_id"`     // 课件作者ID（发起/加人/结束的权限主体）
	CanEdit      bool                `json:"can_edit"`     // 当前请求者此刻能否微调本课件（作者/admin/集体备课参与者，且非锁定态）
	Members      []*CollabMemberView `json:"members"`      // 参与者列表
	Total        int                 `json:"total"`        // 参与者数量
}

// ==================== 课程知识库模型（本轮新增，只读） ====================

// CurriculumKP 课标骨架层知识点（对应 curriculum_standards 表）
// 权威/稳定/版本无关，定义各学科各学段知识点与三档深度，是"难度自动适配"的依据。
type CurriculumKP struct {
	ID                  string `json:"id"`
	Subject             string `json:"subject"`              // 学科
	Stage               string `json:"stage"`                // 学段（小学低/小学中/小学高/初中）
	GradeNum            int    `json:"grade_num"`            // 具体年级1-9（0=学段级，无具体年级）
	Domain              string `json:"domain"`               // 领域（数与代数/图形与几何...）
	Theme               string `json:"theme"`                // 主题
	KPCode              string `json:"kp_code"`              // 知识点全局编码
	KPName              string `json:"kp_name"`              // 知识点名称
	ContentRequirement  string `json:"content_requirement"`  // 内容要求：学什么、范围
	AcademicRequirement string `json:"academic_requirement"` // 学业要求：学到什么程度（难度适配核心）
	TeachingHint        string `json:"teaching_hint"`        // 教学提示
	DepthLevel          int    `json:"depth_level"`          // 难度档 1体验感知/2理解应用/3分析迁移
	CoreCompetency      string `json:"core_competency"`      // 对应核心素养
	SourceRef           string `json:"source_ref"`           // 出处
	Confidence          int    `json:"confidence"`           // 置信度0-100
	SortOrder           int    `json:"sort_order"`           // 同年级内排序
}

// TextbookUnit 教材实例层单元（对应 textbook_units 表）
// 版本相关，各版本教材每年级每册每单元结构，通过 KPCodesJSON 软关联课标知识点。
type TextbookUnit struct {
	ID             string `json:"id"`
	Subject        string `json:"subject"`         // 学科
	Publisher      string `json:"publisher"`       // 教材版本（人教版/苏教版/北师大版...）
	GradeNum       int    `json:"grade_num"`       // 年级1-9
	Semester       string `json:"semester"`        // 册（上册/下册/全册）
	UnitNumber     int    `json:"unit_number"`     // 单元序号
	UnitTitle      string `json:"unit_title"`      // 单元标题
	LessonTitle    string `json:"lesson_title"`    // 课标题（可空）
	ContentSummary string `json:"content_summary"` // 单元内容概述
	KPCodesJSON    string `json:"kp_codes"`        // 引用的课标知识点编码数组（JSON文本，前端解析）
	IdxDepthLevel  int    `json:"idx_depth_level"` // 该单元综合深度档
	SourceType     string `json:"source_type"`     // 数据来源（web_search/pdf_upload/manual）
	Confidence     int    `json:"confidence"`      // 置信度0-100
	SortOrder      int    `json:"sort_order"`      // 排序
}

// ==================== 课件页面级版本与回退模型（页面级版本与回退·新增） ====================
// 对应 courseware_page_versions 表。每次"覆盖式修改 html_content"前存一份旧版本快照，
// 支持查看历次版本 + 一键回退到任意历史版本（回退本身可逆）。
// 本期挂载点：单页AI微调(refine) / 单页重生(regenerate)；背景/字体秒换本期不挂(留枚举位备用)。

// CoursewarePageVersion 课件页面 html_content 版本快照（完整记录，含 HTML）
// 用于 GetPageVersion（取单版完整内容预览/回退）与 CreatePageVersion 返回值。
type CoursewarePageVersion struct {
	ID           string `json:"id"`
	PageID       string `json:"page_id"`
	CoursewareID string `json:"courseware_id"`
	VersionNo    int    `json:"version_no"`

	// HTMLContent 保存该版本的完整页面HTML。
	HTMLContent string `json:"html_content"`

	// PlaceholderMap和MatchedComponentIDs保存创建版本时的页面JSON元数据。
	// 空字符串表示数据库中的NULL。
	PlaceholderMap      string `json:"placeholder_map"`
	MatchedComponentIDs string `json:"matched_component_ids"`

	// PageStatus保存创建该版本时的页面状态。
	// 历史仅HTML版本没有该值。
	PageStatus string `json:"page_status"`

	// MetadataSnapshotComplete区分：
	//   - false：迁移前的历史版本，仅保证HTML有效；
	//   - true：迁移后的完整版本，可恢复HTML及页面元数据。
	MetadataSnapshotComplete bool `json:"metadata_snapshot_complete"`

	Source    string     `json:"source"`
	Note      string     `json:"note"`
	CreatedAt *time.Time `json:"created_at"`
}

// CoursewarePageVersionListItem 版本列表轻量记录。
type CoursewarePageVersionListItem struct {
	ID                       string     `json:"id"`
	VersionNo                int        `json:"version_no"`
	Source                   string     `json:"source"`
	Note                     string     `json:"note"`
	MetadataSnapshotComplete bool       `json:"metadata_snapshot_complete"`
	CreatedAt                *time.Time `json:"created_at"`
}

// ==================== 课件页面版本来源常量 ====================
// source 字段枚举：标识这一版"是什么操作之前的旧版本"，前端据此显示友好标签。

const (
	CWPageVersionSourceRefine     = "refine"     // 单页AI微调前的旧版（本期挂载）
	CWPageVersionSourceRegenerate = "regenerate" // 单页重生前的旧版（本期挂载）
	CWPageVersionSourceBackground = "background" // 背景秒换前的旧版（本期不挂，留位备用）
	CWPageVersionSourceFont       = "font"       // 字体秒换前的旧版（本期不挂，留位备用）
	CWPageVersionSourceNavResync  = "nav_resync" // 导航栏分母刷新前的旧版（本期不挂，留位备用）
	CWPageVersionSourceRollback   = "rollback"   // 回退操作时存的"回退前当前版"（保证回退可逆）
	CWPageVersionSourceCollab     = "collab"     // 阶段4：集体备课期间微调前的旧版（留枚举位，实际标记走 Note 前缀）
	CWPageVersionSourceManual     = "manual"     // 预留：手动存档
)

// CWPageVersionSourceNameMap 版本来源中文标签映射（前端列表显示用）
var CWPageVersionSourceNameMap = map[string]string{
	CWPageVersionSourceRefine:     "🎨 微调前",
	CWPageVersionSourceRegenerate: "🔄 重生前",
	CWPageVersionSourceBackground: "🖼 换背景前",
	CWPageVersionSourceFont:       "🔤 换字体前",
	CWPageVersionSourceNavResync:  "📄 页码刷新前",
	CWPageVersionSourceRollback:   "↩️ 回退前",
	CWPageVersionSourceCollab:     "👥 集体备课前",
	CWPageVersionSourceManual:     "💾 手动存档",
}

// ==================== 阶段4：集体备课候选成员模型 ====================

// CollabCandidate 集体备课可选参与者（轻量，供"加参与者"下拉选人）。
// 来源：当前用户同校 + 同教研组的成员（复用 share 的 resolveSameOrgUserIDs），
//
//	已排除本人、已排除 admin/viewer（admin非教研参与者、viewer只读，均不作候选）。
type CollabCandidate struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// CollabCandidateListResponse 候选成员列表响应
type CollabCandidateListResponse struct {
	Candidates []*CollabCandidate `json:"candidates"`
	Total      int                `json:"total"`
}

// ==================== 阶段4：我参与的集体备课（参与者入口）====================

// JoinedCollabItem "我参与的集体备课"列表单条。
// 参与者（非作者）被拉入集体备课后，在自己界面通过此列表找到并进入这些课件。
// 只列当前仍处于 in_session 的课件（结束后参与者自动失权，不再出现）。
type JoinedCollabItem struct {
	ID          string     `json:"id"`           // 课件ID
	Title       string     `json:"title"`        // 课件标题
	Subject     string     `json:"subject"`      // 学科
	Grade       string     `json:"grade"`        // 年级
	Status      string     `json:"status"`       // 生产态
	StatusName  string     `json:"status_name"`  // 生产态中文名
	PageCount   int        `json:"page_count"`   // 页数
	OwnerID     string     `json:"owner_id"`     // 课件作者ID
	OwnerName   string     `json:"owner_name"`   // 课件作者显示名（这是谁的课件）
	CollabState string     `json:"collab_state"` // 集体备课态（恒为 in_session，冗余返回便于前端统一处理）
	JoinedAt    *time.Time `json:"joined_at"`    // 我被加入这场集体备课的时间
	UpdatedAt   *time.Time `json:"updated_at"`   // 课件更新时间
}

// JoinedCollabListResponse "我参与的集体备课"列表响应
type JoinedCollabListResponse struct {
	Coursewares []*JoinedCollabItem `json:"coursewares"`
	Total       int                 `json:"total"`
}
