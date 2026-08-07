package models

// courseware_comic_storyboard_api.go — 教师端分镜内容编辑协议
//
// 浏览器只提交教师可以理解和检查的教学分镜字段。
// 图片提示词、负面提示词、IAOCI、内部图片键和跨格关系仍由后端维护，
// 禁止通过本协议直接写入或返回。

// UpdateCoursewareComicStoryboardPanelRequest 保存第二步中的单格分镜修改。
//
// ExpectedVersion 使用 panel.version，避免多个标签页静默覆盖同一格。
// 对白、旁白和题目继续由覆盖层编辑器维护，避免分镜步骤与覆盖层文档
// 同时修改同一份课堂文字而产生双事实源。
type UpdateCoursewareComicStoryboardPanelRequest struct {
	ExpectedVersion int `json:"expected_version"`

	StoryPurpose          string `json:"story_purpose"`
	KnowledgeClaim        string `json:"knowledge_claim"`
	SceneText             string `json:"scene_text"`
	ActionText            string `json:"action_text"`
	CameraText            string `json:"camera_text"`
	KnowledgePresentation string `json:"knowledge_presentation"`
}
