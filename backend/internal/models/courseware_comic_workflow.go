package models

// courseware_comic_workflow.go — 知识点漫画五步教师工作流
//
// WorkflowStage描述教师当前需要完成的任务。
// CoursewareComicProject.Status继续描述后台生产事实。

import (
	"strings"
	"time"
)

// 教师视角工作流步骤。
const (
	CWComicWorkflowSource          = "source"
	CWComicWorkflowStoryboard      = "storyboard"
	CWComicWorkflowStylePreview    = "style_preview"
	CWComicWorkflowBatchGeneration = "batch_generation"
	CWComicWorkflowRefinement      = "refinement"
)

// 漫画格图片比例。
const (
	CWComicAspectRatioCourseware = "courseware"
	CWComicAspectRatio16x9       = "16:9"
	CWComicAspectRatio4x3        = "4:3"
	CWComicAspectRatio1x1        = "1:1"
	CWComicAspectRatio3x4        = "3:4"
	CWComicAspectRatio9x16       = "9:16"
)

// 漫画格图片清晰度。
const (
	CWComicImageQualityStandard = "standard"
	CWComicImageQualityHigh     = "high"
)

// 漫画画风来源严格二选一。
//
// courseware：只跟随课件整体风格锚点；
// selected：只使用老师选择的漫画画风。
// 生成服务不得同时应用两种来源。
const (
	CWComicVisualStyleSourceCourseware = "courseware"
	CWComicVisualStyleSourceSelected   = "selected"
)

// 完成漫画后的使用方式。
const (
	CWComicInsertionSinglePage      = "single_page"
	CWComicInsertionSmartPages      = "smart_pages"
	CWComicInsertionOnePanelPerPage = "one_panel_per_page"
	CWComicInsertionLibraryOnly     = "library_only"
)

// CoursewareComicWorkflowState 对应漫画项目表中的教师工作流字段。
//
// ProjectID只用于仓储定位，不从浏览器请求正文读取。
// StylePreviewPanelID采用软关联，使用前必须重新校验属于同一项目。
type CoursewareComicWorkflowState struct {
	ProjectID string `json:"project_id"`

	Stage string `json:"stage"`

	StoryboardConfirmedAt *time.Time `json:"storyboard_confirmed_at"`
	StyleConfirmedAt      *time.Time `json:"style_confirmed_at"`
	StylePreviewPanelID   *string    `json:"style_preview_panel_id"`

	VisualStyleSource string `json:"visual_style_source"`
	AspectRatio       string `json:"aspect_ratio"`
	ImageQuality      string `json:"image_quality"`
	InsertionMode     string `json:"insertion_mode"`
	StyleInstruction  string `json:"style_instruction"`
}

// NewDefaultCoursewareComicWorkflowState 创建新项目默认工作流。
func NewDefaultCoursewareComicWorkflowState(
	projectID string,
) *CoursewareComicWorkflowState {
	return &CoursewareComicWorkflowState{
		ProjectID:         strings.TrimSpace(projectID),
		Stage:             CWComicWorkflowSource,
		VisualStyleSource: CWComicVisualStyleSourceSelected,
		AspectRatio:       CWComicAspectRatioCourseware,
		ImageQuality:      CWComicImageQualityHigh,
		InsertionMode:     CWComicInsertionSinglePage,
	}
}

// NormalizeCoursewareComicWorkflowState 返回规范化副本。
//
// 空白枚举值使用产品默认值补齐。
// 返回false表示项目ID或枚举值不合法。
func NormalizeCoursewareComicWorkflowState(
	state *CoursewareComicWorkflowState,
) (*CoursewareComicWorkflowState, bool) {
	if state == nil {
		return nil, false
	}

	normalized := *state

	normalized.ProjectID =
		strings.TrimSpace(
			normalized.ProjectID,
		)

	normalized.Stage =
		strings.TrimSpace(
			normalized.Stage,
		)

	normalized.VisualStyleSource =
		strings.TrimSpace(
			normalized.VisualStyleSource,
		)

	normalized.AspectRatio =
		strings.TrimSpace(
			normalized.AspectRatio,
		)

	normalized.ImageQuality =
		strings.TrimSpace(
			normalized.ImageQuality,
		)

	normalized.InsertionMode =
		strings.TrimSpace(
			normalized.InsertionMode,
		)

	normalized.StyleInstruction =
		strings.TrimSpace(
			normalized.StyleInstruction,
		)

	if normalized.Stage == "" {
		normalized.Stage =
			CWComicWorkflowSource
	}

	if normalized.VisualStyleSource == "" {
		normalized.VisualStyleSource =
			CWComicVisualStyleSourceSelected
	}

	if normalized.AspectRatio == "" {
		normalized.AspectRatio =
			CWComicAspectRatioCourseware
	}

	if normalized.ImageQuality == "" {
		normalized.ImageQuality =
			CWComicImageQualityHigh
	}

	if normalized.InsertionMode == "" {
		normalized.InsertionMode =
			CWComicInsertionSinglePage
	}

	if normalized.StylePreviewPanelID != nil {
		panelID :=
			strings.TrimSpace(
				*normalized.StylePreviewPanelID,
			)

		if panelID == "" {
			normalized.StylePreviewPanelID = nil
		} else {
			normalized.StylePreviewPanelID =
				&panelID
		}
	}

	if normalized.ProjectID == "" ||
		!IsValidCWComicWorkflowStage(
			normalized.Stage,
		) ||
		!IsValidCWComicVisualStyleSource(
			normalized.VisualStyleSource,
		) ||
		!IsValidCWComicAspectRatio(
			normalized.AspectRatio,
		) ||
		!IsValidCWComicImageQuality(
			normalized.ImageQuality,
		) ||
		!IsValidCWComicInsertionMode(
			normalized.InsertionMode,
		) {
		return nil, false
	}

	return &normalized, true
}

// IsValidCWComicWorkflowStage 校验教师工作流步骤。
func IsValidCWComicWorkflowStage(
	value string,
) bool {
	switch strings.TrimSpace(value) {
	case CWComicWorkflowSource,
		CWComicWorkflowStoryboard,
		CWComicWorkflowStylePreview,
		CWComicWorkflowBatchGeneration,
		CWComicWorkflowRefinement:
		return true

	default:
		return false
	}
}

// IsValidCWComicVisualStyleSource 校验画风来源严格二选一。
func IsValidCWComicVisualStyleSource(
	value string,
) bool {
	switch strings.TrimSpace(
		value,
	) {
	case CWComicVisualStyleSourceCourseware,
		CWComicVisualStyleSourceSelected:
		return true

	default:
		return false
	}
}

// IsValidCWComicAspectRatio 校验图片比例。
func IsValidCWComicAspectRatio(
	value string,
) bool {
	switch strings.TrimSpace(value) {
	case CWComicAspectRatioCourseware,
		CWComicAspectRatio16x9,
		CWComicAspectRatio4x3,
		CWComicAspectRatio1x1,
		CWComicAspectRatio3x4,
		CWComicAspectRatio9x16:
		return true

	default:
		return false
	}
}

// IsValidCWComicImageQuality 校验图片清晰度。
func IsValidCWComicImageQuality(
	value string,
) bool {
	switch strings.TrimSpace(value) {
	case CWComicImageQualityStandard,
		CWComicImageQualityHigh:
		return true

	default:
		return false
	}
}

// IsValidCWComicInsertionMode 校验漫画使用方式。
func IsValidCWComicInsertionMode(
	value string,
) bool {
	switch strings.TrimSpace(value) {
	case CWComicInsertionSinglePage,
		CWComicInsertionSmartPages,
		CWComicInsertionOnePanelPerPage,
		CWComicInsertionLibraryOnly:
		return true

	default:
		return false
	}
}
