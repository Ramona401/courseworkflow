package models

// courseware_comic_workflow_api.go — 五步漫画工作流HTTP协议
//
// 请求正文只接受教师正在确认的业务选项和expected_version。
// 课件ID、项目ID、用户、学校和资产归属均由服务端可信路径读取。

// CoursewareComicWorkflowView 是浏览器安全的教师工作流视图。
type CoursewareComicWorkflowView struct {
	Stage string `json:"stage"`

	StoryboardConfirmedAt interface{} `json:"storyboard_confirmed_at"`
	StyleConfirmedAt      interface{} `json:"style_confirmed_at"`
	StylePreviewPanelID   *string     `json:"style_preview_panel_id"`

	VisualStyleSource string `json:"visual_style_source"`
	AspectRatio       string `json:"aspect_ratio"`
	ImageQuality      string `json:"image_quality"`
	InsertionMode     string `json:"insertion_mode"`
	StyleInstruction  string `json:"style_instruction"`
}

// ConfirmCoursewareComicStoryboardRequest 确认第二步分镜方案。
//
// NarrativeMode允许老师确认当前AI推荐。
// 叙事方式发生改变时，服务层必须拒绝直接确认并要求重新规划。
type ConfirmCoursewareComicStoryboardRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	NarrativeMode   string `json:"narrative_mode"`
}

// UpdateCoursewareComicStyleSettingsRequest 保存第三步视觉设置。
//
// VisualStyleSource是courseware或selected，禁止混合。
// VisualStyle是系统稳定美术风格代码；仅selected模式用于生成。
// StyleInstruction是教师可选的自然语言补充要求。
// 浏览器不得通过该字段提交模型配置、密钥或第三方授权信息。
type UpdateCoursewareComicStyleSettingsRequest struct {
	ExpectedVersion int `json:"expected_version"`

	VisualStyleSource string `json:"visual_style_source"`
	VisualStyle       string `json:"visual_style"`
	AspectRatio       string `json:"aspect_ratio"`
	ImageQuality      string `json:"image_quality"`
	StyleInstruction  string `json:"style_instruction"`
}

// GenerateCoursewareComicStylePreviewRequest 启动首格完整样张生成。
type GenerateCoursewareComicStylePreviewRequest struct {
	ExpectedVersion int `json:"expected_version"`
}

// ConfirmCoursewareComicStylePreviewRequest 确认首格样张。
//
// PanelID必须是当前项目第1格。
// 服务端必须重新校验分格归属和图片生成状态。
type ConfirmCoursewareComicStylePreviewRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	PanelID         string `json:"panel_id"`
}

// UpdateCoursewareComicInsertionModeRequest 保存第五步使用方式。
type UpdateCoursewareComicInsertionModeRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	InsertionMode   string `json:"insertion_mode"`
}
