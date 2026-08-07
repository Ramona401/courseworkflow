package handlers

// courseware_comic_generation_handler.go — 漫画图片生产HTTP入口
//
// 提供：
//   - POST /comic-projects/{project_id}/generate
//   - POST /comic-projects/{project_id}/panels/{panel_id}/regenerate
//
// 两个端点只负责严格读取版本号、执行作者预检和启动后台任务。
// 实际图片调用不使用请求Context，避免HTTP响应结束后任务被取消。

import (
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

type coursewareComicGenerateProjectRequest struct {
	ExpectedVersion int `json:"expected_version"`
}

type coursewareComicRegeneratePanelRequest struct {
	ExpectedVersion         int    `json:"expected_version"`
	RegenerationInstruction string `json:"regeneration_instruction"`
}

// CoursewareComicGenerationHandler 漫画图片生产处理器。
type CoursewareComicGenerationHandler struct {
	service *services.CoursewareComicGenerationService
}

// NewCoursewareComicGenerationHandler 创建图片生产处理器。
func NewCoursewareComicGenerationHandler(
	service *services.CoursewareComicGenerationService,
) *CoursewareComicGenerationHandler {
	return &CoursewareComicGenerationHandler{
		service: service,
	}
}

// GenerateProject 启动人物设定图和全部漫画格顺序生成。
func (h *CoursewareComicGenerationHandler) GenerateProject(
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

	var request coursewareComicGenerateProjectRequest

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
		h.service.StartProjectGeneration(
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

	utils.Success(w, result)
}

// RegeneratePanel 启动单格重新生成。
func (h *CoursewareComicGenerationHandler) RegeneratePanel(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
	panelID string,
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

	var request coursewareComicRegeneratePanelRequest

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
		h.service.StartPanelRegeneration(
			r.Context(),
			coursewareID,
			projectID,
			panelID,
			request.ExpectedVersion,
			request.RegenerationInstruction,
			actor,
		)
	if err != nil {
		writeCoursewareComicHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(w, result)
}
