package models

// courseware_comic_api.go — 知识点漫画HTTP协议
//
// 安全原则：
//   - 创建请求可以只提交教师自由输入的knowledge_text；
//   - 教材、单元、课标知识点和项目参考资源均为可选增强来源；
//   - 学科、年级、教育域和作者身份仍由服务端读取正式课件；
//   - 浏览器响应使用显式安全结构；
//   - 不返回助手完整提示词、模型密钥和积分内部信息；
//   - 不向浏览器序列化IAOCI、图片生成提示词、图片关系索引和内部ImageKey；
//   - 参考资源不返回content_text和summary_text；
//   - 图片URL必须来自服务端校验过的同课件图片资产。
//
// 数据库实体和后端内部视图仍保留IAOCI等字段，供图片规划、
// 人物连续性、重新生成和历史审计使用。保密字段统一使用json:"-"，
// 即使后续服务层误把字段赋值到浏览器视图，也不会被JSON序列化。

import "encoding/json"

// CreateCoursewareComicProjectRequest 创建知识点漫画项目。
//
// 推荐的一键模式只需要KnowledgeText。
// 其余字段保留用于旧前端、内部调用和后续高级选项。
type CreateCoursewareComicProjectRequest struct {
	KnowledgeText string `json:"knowledge_text"`

	Title          string   `json:"title"`
	Publisher      string   `json:"publisher"`
	Semester       string   `json:"semester"`
	TextbookUnitID string   `json:"textbook_unit_id"`
	KPCodes        []string `json:"kp_codes"`

	AssistantID *string `json:"assistant_id"`

	NarrativeMode string `json:"narrative_mode"`
	VisualStyle   string `json:"visual_style"`
	PanelCount    int    `json:"panel_count"`
	LayoutMode    string `json:"layout_mode"`
	TeacherFocus  string `json:"teacher_focus"`
}

// UpdateCoursewareComicPanelOverlayRequest 保存教师调整后的文字与气泡。
type UpdateCoursewareComicPanelOverlayRequest struct {
	ExpectedVersion int `json:"expected_version"`

	NarrationText   string                          `json:"narration_text"`
	OverlayDocument CoursewareComicOverlayDocument `json:"overlay_document"`
}

// UpdateCoursewareComicPanelPromptRequest 是后端内部兼容请求结构。
//
// 教师端公共HTTP路由已经关闭图片提示词和IAOCI直接编辑能力。
// 该结构暂时保留，避免影响既有内部服务、历史测试和后续将自然语言
// 画面修改要求转换为内部IAOCI的服务端实现。
//
// 禁止重新把该结构直接挂载到教师端浏览器路由。
type UpdateCoursewareComicPanelPromptRequest struct {
	ExpectedVersion int `json:"expected_version"`

	VisualPrompt   string `json:"visual_prompt"`
	NegativePrompt string `json:"negative_prompt"`
	AOCIText       string `json:"aoci_text"`
}

// CoursewareComicProjectView 是浏览器安全的漫画项目视图。
type CoursewareComicProjectView struct {
	ID              string `json:"id"`
	CoursewareID    string `json:"courseware_id"`
	EducationDomain string `json:"education_domain"`

	Title   string `json:"title"`
	Subject string `json:"subject"`
	Grade   string `json:"grade"`

	Publisher string `json:"publisher"`
	Semester  string `json:"semester"`

	TextbookUnit     CoursewareComicTextbookUnitSnapshot     `json:"textbook_unit"`
	KnowledgePoints  []CoursewareComicKnowledgePointSnapshot `json:"knowledge_points"`
	KnowledgeContent string                                  `json:"knowledge_content"`
	TeacherFocus     string                                  `json:"teacher_focus"`

	AssistantID *string `json:"assistant_id"`

	NarrativeMode string `json:"narrative_mode"`
	VisualStyle   string `json:"visual_style"`
	PanelCount    int    `json:"panel_count"`
	LayoutMode    string `json:"layout_mode"`

	PageLayout        json.RawMessage `json:"page_layout"`
	InteractionConfig json.RawMessage `json:"interaction_config"`

	// StyleAOCIText和ContinuityLedger仅供后端内部图片生成、
	// 连续性校验及历史审计使用，禁止进入浏览器响应。
	StyleAOCIText    string          `json:"-"`
	ContinuityLedger json.RawMessage `json:"-"`

	CharacterBible CoursewareComicCharacterBible `json:"character_bible"`

	CharacterSheetAssetID *string `json:"character_sheet_asset_id"`
	CharacterSheetURL     string  `json:"character_sheet_url"`

	// Workflow描述老师当前应该完成的五步任务，
	// Status继续描述后台规划、生图、完成和失败等生产事实。
	Workflow *CoursewareComicWorkflowView `json:"workflow"`

	Status                     string  `json:"status"`
	InsertedPageID             *string `json:"inserted_page_id"`
	InsertedPageNumberSnapshot int     `json:"inserted_page_number_snapshot"`

	Version   int    `json:"version"`
	LastError string `json:"last_error"`

	CreatedAt interface{} `json:"created_at"`
	UpdatedAt interface{} `json:"updated_at"`
}

// CoursewareComicPanelView 是浏览器安全的漫画格视图。
type CoursewareComicPanelView struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	PanelNo   int    `json:"panel_no"`

	// ImageKey属于图片IAOCI稳定索引键，只供后端关系校验和生成链使用。
	ImageKey string `json:"-"`

	StoryPurpose          string                    `json:"story_purpose"`
	KnowledgeClaim        string                    `json:"knowledge_claim"`
	SceneText             string                    `json:"scene_text"`
	CharacterIDs          []string                  `json:"character_ids"`
	ActionText            string                    `json:"action_text"`
	CameraText            string                    `json:"camera_text"`
	NarrationText         string                    `json:"narration_text"`
	Dialogues             []CoursewareComicDialogue `json:"dialogues"`
	KnowledgePresentation string                    `json:"knowledge_presentation"`

	// 以下四项是内部图片生产协议。
	// 服务层可以继续赋值，但JSON编码时必须彻底剥离。
	VisualPrompt   string                         `json:"-"`
	NegativePrompt string                         `json:"-"`
	AOCIText       string                         `json:"-"`
	Relations      []CoursewareComicPanelRelation `json:"-"`

	OverlayDocument CoursewareComicOverlayDocument `json:"overlay_document"`
	OverlayVersion  int                            `json:"overlay_version"`

	Status          string  `json:"status"`
	CurrentAssetID  *string `json:"current_asset_id"`
	CurrentAssetURL string  `json:"current_asset_url"`

	Version   int    `json:"version"`
	LastError string `json:"last_error"`

	CreatedAt interface{} `json:"created_at"`
	UpdatedAt interface{} `json:"updated_at"`
}

// CoursewareComicProjectDetailView
// 返回一个项目、全部漫画格和安全参考资源元数据。
type CoursewareComicProjectDetailView struct {
	Project    *CoursewareComicProjectView             `json:"project"`
	Panels     []*CoursewareComicPanelView             `json:"panels"`
	References []*CoursewareComicReferenceResourceView `json:"references"`
}

// CoursewareComicProjectListView 返回课件中的漫画项目。
type CoursewareComicProjectListView struct {
	Projects []*CoursewareComicProjectView `json:"projects"`
	Total    int                            `json:"total"`
}
