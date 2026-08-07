package models

// courseware_comic.go — 知识点漫画领域模型与稳定数据协议
//
// 第一版产品原则：
//   - AI自动规划4至8格漫画并完成气泡、旁白、知识卡和题目卡初始排版；
//   - 教师进入编辑器时看到可以直接使用的完整成品，而不是空白画布；
//   - 教师可以修改文字、题目、选项、答案、气泡样式、位置和尾巴指向；
//   - 自动重新排版只修改位置、尺寸、字号、层级和尾巴，不覆盖教师文字；
//   - 图片模型只生成无文字视觉画面；
//   - 中文文字、公式、题目和知识结论使用HTML与SVG覆盖层渲染；
//   - 数据库保存TE-DNA稳定文档协议，不保存第三方编辑器私有状态；
//   - 第三方宽松许可证组件仅作为编辑和渲染适配层，可被安全替换。
//
// 数据库JSONB字段在数据库实体中保留原始JSON字符串。
// 后续Handler必须显式构造浏览器安全响应，禁止直接返回内部数据库实体。

import (
	"strings"
	"time"
)

// SceneCoursewareComicPlan 是知识点漫画规划独立AI场景，
// 同时用于AI配置、积分追踪和漫画助手适用场景。
const SceneCoursewareComicPlan = "courseware_comic_plan"

// 漫画项目状态。
const (
	CWComicProjectStatusDraft      = "draft"
	CWComicProjectStatusPlanning   = "planning"
	CWComicProjectStatusPlanned    = "planned"
	CWComicProjectStatusGenerating = "generating"
	CWComicProjectStatusReady      = "ready"
	CWComicProjectStatusInserted   = "inserted"
	CWComicProjectStatusFailed     = "failed"
	CWComicProjectStatusArchived   = "archived"
)

// 漫画页面布局模式。
const (
	CWComicLayoutGrid      = "grid"
	CWComicLayoutSpotlight = "spotlight"
	CWComicLayoutCarousel  = "carousel"
)

// 漫画格生产状态。
const (
	CWComicPanelStatusPlanned    = "planned"
	CWComicPanelStatusGenerating = "generating"
	CWComicPanelStatusGenerated  = "generated"
	CWComicPanelStatusFailed     = "failed"
	CWComicPanelStatusStale      = "stale"
)

// 漫画格历史版本来源。
const (
	CWComicVersionSourceInitial    = "initial"
	CWComicVersionSourceRegenerate = "regenerate"
	CWComicVersionSourceRestore    = "restore"
	CWComicVersionSourceManualSave = "manual_save"
)

// 覆盖层元素类型。
const (
	CWComicElementSpeechBubble  = "speech_bubble"
	CWComicElementThoughtBubble = "thought_bubble"
	CWComicElementNarration     = "narration"
	CWComicElementKnowledgeCard = "knowledge_card"
	CWComicElementWarningCard   = "warning_card"
	CWComicElementQuestionCard  = "question_card"
	CWComicElementAnswerCard    = "answer_card"
	CWComicElementCaption       = "caption"
	CWComicElementEmphasis      = "emphasis"
)

// 自动布局区域。
const (
	CWComicRegionTopLeft      = "top_left"
	CWComicRegionTopCenter    = "top_center"
	CWComicRegionTopRight     = "top_right"
	CWComicRegionMiddleLeft   = "middle_left"
	CWComicRegionMiddleRight  = "middle_right"
	CWComicRegionBottomLeft   = "bottom_left"
	CWComicRegionBottomCenter = "bottom_center"
	CWComicRegionBottomRight  = "bottom_right"
)

// 人物与气泡尾巴可使用的画面锚点。
const (
	CWComicAnchorLeftTop      = "left_top"
	CWComicAnchorLeftCenter   = "left_center"
	CWComicAnchorLeftBottom   = "left_bottom"
	CWComicAnchorCenterTop    = "center_top"
	CWComicAnchorCenter       = "center"
	CWComicAnchorCenterBottom = "center_bottom"
	CWComicAnchorRightTop     = "right_top"
	CWComicAnchorRightCenter  = "right_center"
	CWComicAnchorRightBottom  = "right_bottom"
)

// 漫画人物主体类型。
const (
	CWComicCharacterSubjectPerson = "person"
	CWComicCharacterSubjectAnimal = "animal"
	CWComicCharacterSubjectObject = "object"
)

// 题目答案展示方式。
const (
	CWComicAnswerModeStatic      = "static"
	CWComicAnswerModeClickReveal = "click_reveal"
)

// 漫画文字颜色模式。
//
// auto 表示由渲染器根据气泡或卡片背景自动选择深色或白色文字；
// manual 表示严格使用教师选择的 Color。
// 历史文档没有该字段时按 auto 处理，避免深色背景与黑色文字重叠。
const (
	CWComicTextColorModeAuto   = "auto"
	CWComicTextColorModeManual = "manual"
)

// CoursewareComicProject 对应courseware_comic_projects。
type CoursewareComicProject struct {
	ID              string `json:"id"`
	CoursewareID    string `json:"courseware_id"`
	CreatedBy       string `json:"created_by"`
	EducationDomain string `json:"education_domain"`

	Title   string `json:"title"`
	Subject string `json:"subject"`
	Grade   string `json:"grade"`

	PublisherSnapshot        string  `json:"publisher_snapshot"`
	SemesterSnapshot         string  `json:"semester_snapshot"`
	TextbookUnitID           *string `json:"textbook_unit_id"`
	TextbookUnitSnapshotJSON string  `json:"textbook_unit_snapshot_json"`

	KnowledgePointsJSON      string `json:"knowledge_points_json"`
	KnowledgeContentSnapshot string `json:"knowledge_content_snapshot"`
	TeacherFocus             string `json:"teacher_focus"`

	AssistantID   *string `json:"assistant_id"`
	NarrativeMode string  `json:"narrative_mode"`
	VisualStyle   string  `json:"visual_style"`
	PanelCount    int     `json:"panel_count"`
	LayoutMode    string  `json:"layout_mode"`

	PageLayoutJSON        string `json:"page_layout_json"`
	InteractionConfigJSON string `json:"interaction_config_json"`

	StyleAOCIText         string  `json:"style_aoci_text"`
	CharacterBibleJSON    string  `json:"character_bible_json"`
	ContinuityLedgerJSON  string  `json:"continuity_ledger_json"`
	CharacterSheetAssetID *string `json:"character_sheet_asset_id"`

	Status                     string  `json:"status"`
	InsertedPageID             *string `json:"inserted_page_id"`
	InsertedPageNumberSnapshot int     `json:"inserted_page_number_snapshot"`

	Version   int    `json:"version"`
	LastError string `json:"last_error"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareComicPanel 对应courseware_comic_panels。
type CoursewareComicPanel struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	PanelNo   int    `json:"panel_no"`
	ImageKey  string `json:"image_key"`

	StoryPurpose          string `json:"story_purpose"`
	KnowledgeClaim        string `json:"knowledge_claim"`
	SceneText             string `json:"scene_text"`
	CharacterIDsJSON      string `json:"character_ids_json"`
	ActionText            string `json:"action_text"`
	CameraText            string `json:"camera_text"`
	NarrationText         string `json:"narration_text"`
	DialoguesJSON         string `json:"dialogues_json"`
	KnowledgePresentation string `json:"knowledge_presentation"`

	VisualPrompt   string `json:"visual_prompt"`
	NegativePrompt string `json:"negative_prompt"`
	AOCIText       string `json:"aoci_text"`
	RelationsJSON  string `json:"relations_json"`

	OverlayDocumentJSON string `json:"overlay_document_json"`
	OverlayVersion      int    `json:"overlay_version"`

	Status         string  `json:"status"`
	CurrentAssetID *string `json:"current_asset_id"`

	Version   int    `json:"version"`
	LastError string `json:"last_error"`

	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CoursewareComicPanelVersion 对应courseware_comic_panel_versions。
type CoursewareComicPanelVersion struct {
	ID        string `json:"id"`
	PanelID   string `json:"panel_id"`
	VersionNo int    `json:"version_no"`

	PromptSnapshot          string `json:"prompt_snapshot"`
	AOCISnapshot            string `json:"aoci_snapshot"`
	OverlayDocumentSnapshot string `json:"overlay_document_snapshot"`

	AssetID          *string    `json:"asset_id"`
	GenerationSource string     `json:"generation_source"`
	CreatedBy        string     `json:"created_by"`
	CreatedAt        *time.Time `json:"created_at"`
}

// PlanCoursewareComicRequest 是漫画AI规划请求。
//
// 身份、学校、教育域、学科、年级、教材和知识点均不从请求正文读取，
// 服务端只接受当前项目已经固化的可信快照。
//
// NarrativeMode为空时沿用项目当前叙事方式。
// 非空时必须是稳定枚举，并在领取planning状态时与版本CAS原子保存。
type PlanCoursewareComicRequest struct {
	ExpectedVersion    int    `json:"expected_version"`
	TeacherInstruction string `json:"teacher_instruction"`
	NarrativeMode      string `json:"narrative_mode"`
}

// CoursewareComicPlanResult 是后端规划服务内部结果。
//
// 后续HTTP层需要转换为浏览器安全视图，不能直接暴露项目内部长快照。
type CoursewareComicPlanResult struct {
	Project *CoursewareComicProject `json:"project"`
	Panels  []*CoursewareComicPanel `json:"panels"`
}

// CoursewareComicKnowledgePointSnapshot 是项目中固化的单个课标知识点。
type CoursewareComicKnowledgePointSnapshot struct {
	KPCode              string `json:"kp_code"`
	KPName              string `json:"kp_name"`
	ContentRequirement  string `json:"content_requirement"`
	AcademicRequirement string `json:"academic_requirement"`
	TeachingHint        string `json:"teaching_hint"`
	DepthLevel          int    `json:"depth_level"`
	SourceRef           string `json:"source_ref"`
}

// CoursewareComicTextbookUnitSnapshot 是项目创建时的教材单元快照。
type CoursewareComicTextbookUnitSnapshot struct {
	ID             string   `json:"id"`
	Publisher      string   `json:"publisher"`
	GradeNum       int      `json:"grade_num"`
	Semester       string   `json:"semester"`
	UnitNumber     int      `json:"unit_number"`
	UnitTitle      string   `json:"unit_title"`
	LessonTitle    string   `json:"lesson_title"`
	ContentSummary string   `json:"content_summary"`
	KPCodes        []string `json:"kp_codes"`
}

// CoursewareComicCharacterBible 保存整个项目的人物一致性规则。
type CoursewareComicCharacterBible struct {
	Version    int                        `json:"version"`
	Characters []CoursewareComicCharacter `json:"characters"`
}

// CoursewareComicCharacter 保存人物、动物或拟人知识对象的固定特征。
type CoursewareComicCharacter struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Role            string `json:"role"`
	SubjectType     string `json:"subject_type"`
	Appearance      string `json:"appearance"`
	DefaultPosition string `json:"default_position"`

	FixedFeatures    []string `json:"fixed_features"`
	ForbiddenChanges []string `json:"forbidden_changes"`

	ReferenceAssetID *string `json:"reference_asset_id,omitempty"`
}

// CoursewareComicDialogue 保存一条可编辑人物对白。
type CoursewareComicDialogue struct {
	ID          string `json:"id"`
	CharacterID string `json:"character_id"`
	Content     string `json:"content"`
	BubbleStyle string `json:"bubble_style"`
	Emotion     string `json:"emotion"`
}

// CoursewareComicPanelRelation 保存漫画格之间的连续性关系。
type CoursewareComicPanelRelation struct {
	TargetImageKey string `json:"target_image_key"`
	RelationCode   string `json:"relation_code"`
	InheritMask    string `json:"inherit_mask"`
	SemanticNote   string `json:"semantic_note"`
}

// CoursewareComicOverlayDocument 是可编辑文字与气泡的稳定文档协议。
//
// 坐标全部使用0至1的归一化比例，保证1920×1080课件画布、
// 编辑器缩放和离线导出使用同一份数据。
type CoursewareComicOverlayDocument struct {
	Version  int                             `json:"version"`
	Canvas   CoursewareComicOverlayCanvas    `json:"canvas"`
	Elements []CoursewareComicOverlayElement `json:"elements"`
}

// CoursewareComicOverlayCanvas 描述覆盖层设计坐标系。
type CoursewareComicOverlayCanvas struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// CoursewareComicOverlayElement 描述气泡、旁白、题目或教学卡片。
type CoursewareComicOverlayElement struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`

	// OriginalContent用于“恢复AI初稿”，自动排版不得修改。
	OriginalContent string `json:"original_content"`

	SpeakerID         string `json:"speaker_id,omitempty"`
	TargetCharacterID string `json:"target_character_id,omitempty"`
	TargetAnchor      string `json:"target_anchor,omitempty"`
	StyleID           string `json:"style_id"`

	// AutoLayoutRegion和Priority用于后续重新自动排版。
	AutoLayoutRegion string `json:"auto_layout_region"`
	Priority         int    `json:"priority"`

	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
	ZIndex   int     `json:"z_index"`

	Tail      *CoursewareComicBubbleTail `json:"tail,omitempty"`
	TextStyle CoursewareComicTextStyle   `json:"text_style"`

	Question         *CoursewareComicQuestionContent `json:"question,omitempty"`
	OriginalQuestion *CoursewareComicQuestionContent `json:"original_question,omitempty"`

	Locked       bool `json:"locked"`
	ContentDirty bool `json:"content_dirty"`
	LayoutDirty  bool `json:"layout_dirty"`
}

// CoursewareComicBubbleTail 保存气泡尾巴的连接点和人物指向位置。
//
// OriginX和OriginY使用气泡自身0至1本地坐标，表示尾巴与气泡边框的连接点。
// 指针类型用于兼容历史文档：旧数据没有连接点时保持nil，由编辑器和渲染器自动推导。
// TargetX和TargetY继续使用整格画布0至1坐标，表示尾巴最终指向位置。
type CoursewareComicBubbleTail struct {
	Type    string  `json:"type"`
	TargetX float64 `json:"target_x"`
	TargetY float64 `json:"target_y"`

	OriginX *float64 `json:"origin_x,omitempty"`
	OriginY *float64 `json:"origin_y,omitempty"`
}

// CoursewareComicTextStyle 保存可编辑文字排版。
//
// BackgroundOpacity 是对当前气泡或卡片预设背景透明度的乘数，合法范围0.2至1。
// 0表示历史文档未设置，所有渲染端必须按1处理，不能把背景错误变成全透明。
//
// OutlineWidth 是气泡整体外轮廓的CSS像素线宽，合法范围0.5至3。
// 0表示历史文档未设置，渲染端和保存规范化均按1处理。
// 说话气泡主体与尾巴必须共用该值，不能分别描边。
//
// ColorMode 为空或auto时由渲染器根据背景明暗自动选择高对比文字；
// manual时才使用Color，保证深色背景不会继续出现黑底黑字。
type CoursewareComicTextStyle struct {
	FontFamily        string  `json:"font_family"`
	FontSize          float64 `json:"font_size"`
	FontWeight        int     `json:"font_weight"`
	LineHeight        float64 `json:"line_height"`
	Align             string  `json:"align"`
	Color             string  `json:"color"`
	ColorMode         string  `json:"color_mode"`
	BackgroundOpacity float64 `json:"background_opacity"`
	OutlineWidth      float64 `json:"outline_width"`
}

// CoursewareComicQuestionContent 保存题目、选项、答案和解析。
type CoursewareComicQuestionContent struct {
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	AnswerIndex int      `json:"answer_index"`
	Explanation string   `json:"explanation"`
	AnswerMode  string   `json:"answer_mode"`
}

// IsValidCWComicProjectStatus 校验项目状态。
func IsValidCWComicProjectStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicProjectStatusDraft,
		CWComicProjectStatusPlanning,
		CWComicProjectStatusPlanned,
		CWComicProjectStatusGenerating,
		CWComicProjectStatusReady,
		CWComicProjectStatusInserted,
		CWComicProjectStatusFailed,
		CWComicProjectStatusArchived:
		return true

	default:
		return false
	}
}

// IsEditableCWComicProjectStatus 判断项目是否允许修改来源和重新规划。
func IsEditableCWComicProjectStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicProjectStatusDraft,
		CWComicProjectStatusPlanned,
		CWComicProjectStatusFailed:
		return true

	default:
		return false
	}
}

// IsValidCWComicLayoutMode 校验最终页面布局模式。
func IsValidCWComicLayoutMode(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicLayoutGrid,
		CWComicLayoutSpotlight,
		CWComicLayoutCarousel:
		return true

	default:
		return false
	}
}

// IsValidCWComicPanelStatus 校验漫画格状态。
func IsValidCWComicPanelStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicPanelStatusPlanned,
		CWComicPanelStatusGenerating,
		CWComicPanelStatusGenerated,
		CWComicPanelStatusFailed,
		CWComicPanelStatusStale:
		return true

	default:
		return false
	}
}

// IsValidCWComicVersionSource 校验历史版本来源。
func IsValidCWComicVersionSource(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicVersionSourceInitial,
		CWComicVersionSourceRegenerate,
		CWComicVersionSourceRestore,
		CWComicVersionSourceManualSave:
		return true

	default:
		return false
	}
}

// IsValidCWComicElementType 校验覆盖层元素类型。
func IsValidCWComicElementType(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicElementSpeechBubble,
		CWComicElementThoughtBubble,
		CWComicElementNarration,
		CWComicElementKnowledgeCard,
		CWComicElementWarningCard,
		CWComicElementQuestionCard,
		CWComicElementAnswerCard,
		CWComicElementCaption,
		CWComicElementEmphasis:
		return true

	default:
		return false
	}
}

// IsValidCWComicLayoutRegion 校验自动布局区域。
func IsValidCWComicLayoutRegion(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicRegionTopLeft,
		CWComicRegionTopCenter,
		CWComicRegionTopRight,
		CWComicRegionMiddleLeft,
		CWComicRegionMiddleRight,
		CWComicRegionBottomLeft,
		CWComicRegionBottomCenter,
		CWComicRegionBottomRight:
		return true

	default:
		return false
	}
}

// IsValidCWComicCharacterAnchor 校验人物或尾巴锚点。
func IsValidCWComicCharacterAnchor(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicAnchorLeftTop,
		CWComicAnchorLeftCenter,
		CWComicAnchorLeftBottom,
		CWComicAnchorCenterTop,
		CWComicAnchorCenter,
		CWComicAnchorCenterBottom,
		CWComicAnchorRightTop,
		CWComicAnchorRightCenter,
		CWComicAnchorRightBottom:
		return true

	default:
		return false
	}
}

// IsValidCWComicCharacterSubjectType 校验人物主体类型。
func IsValidCWComicCharacterSubjectType(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicCharacterSubjectPerson,
		CWComicCharacterSubjectAnimal,
		CWComicCharacterSubjectObject:
		return true

	default:
		return false
	}
}

// IsValidCWComicAnswerMode 校验答案展示方式。
func IsValidCWComicAnswerMode(value string) bool {
	switch strings.TrimSpace(value) {
	case CWComicAnswerModeStatic,
		CWComicAnswerModeClickReveal:
		return true

	default:
		return false
	}
}

// IsValidCWComicTextColorMode 校验文字颜色模式。
//
// 空值仅用于兼容历史文档，并与auto具有相同语义。
func IsValidCWComicTextColorMode(value string) bool {
	switch strings.TrimSpace(value) {
	case "", CWComicTextColorModeAuto, CWComicTextColorModeManual:
		return true

	default:
		return false
	}
}

// init把知识点漫画规划注册为AI配置场景和AI助手适用场景。
// 幂等检查避免滚动开发中重复注册。
func init() {
	registerScene := func(values *[]string, target string) {
		for _, value := range *values {
			if value == target {
				return
			}
		}

		*values = append(*values, target)
	}

	registerScene(
		&ValidSceneCodes,
		SceneCoursewareComicPlan,
	)
	SceneNameMap[SceneCoursewareComicPlan] =
		"知识点漫画规划"
	SceneGroupMap[SceneCoursewareComicPlan] =
		"courseware"

	registerScene(
		&ValidAssistantScenes,
		SceneCoursewareComicPlan,
	)
}
