package handlers

// courseware_comic_style_preview_handler.go
//
// 第三步首格样张HTTP入口：
//   POST /comic-projects/{project_id}/generate-style-preview
//   POST /comic-projects/{project_id}/confirm-style-preview
//
// 生成接口只返回后台任务登记结果。
// 确认接口返回完整项目详情，其中包含新项目version和最新workflow。

import (
	"net/http"
	"strings"

	"tedna/internal/services"
	"tedna/internal/utils"
)

type coursewareComicGenerateStylePreviewRequest struct {
	ExpectedVersion int `json:"expected_version"`
}

type coursewareComicConfirmStylePreviewRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	PreviewPanelID  string `json:"preview_panel_id"`
}

// GenerateStylePreview 启动第1格完整样张后台生成。
func (h *CoursewareComicGenerationHandler) GenerateStylePreview(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
) {
	if r.Method != http.MethodPost {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持POST请求",
		)
		return
	}

	actor, ok :=
		authorizeCoursewareComicActor(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var request coursewareComicGenerateStylePreviewRequest

	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicPlanRequestMaxBytes,
	) {
		return
	}

	if h == nil ||
		h.service == nil {
		writeCoursewareComicHandlerError(
			w,
			services.ErrCoursewareComicProjectServiceUnavailable,
		)
		return
	}

	result, err :=
		h.service.StartStylePreviewGeneration(
			r.Context(),
			coursewareID,
			projectID,
			request.ExpectedVersion,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		result,
	)
}

// ConfirmStylePreview 确认首格完整样张并进入整批生成步骤。
func (h *CoursewareComicHandler) ConfirmStylePreview(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
) {
	if r.Method != http.MethodPost {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持POST请求",
		)
		return
	}

	actor, ok :=
		authorizeCoursewareComicActor(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return
	}

	var request coursewareComicConfirmStylePreviewRequest

	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicPlanRequestMaxBytes,
	) {
		return
	}

	request.PreviewPanelID =
		strings.TrimSpace(
			request.PreviewPanelID,
		)

	projectService :=
		h.resolveProjectService()

	_, err :=
		projectService.ConfirmProjectStylePreview(
			r.Context(),
			coursewareID,
			projectID,
			request.PreviewPanelID,
			request.ExpectedVersion,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	result, err :=
		projectService.GetProjectDetailForBrowser(
			r.Context(),
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	workflow, err :=
		projectService.GetProjectWorkflowForBrowser(
			r.Context(),
			coursewareID,
			projectID,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	if err :=
		attachCoursewareComicWorkflowToDetail(
			result,
			workflow,
		); err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		result,
	)
}
