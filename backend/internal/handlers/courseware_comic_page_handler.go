package handlers

// courseware_comic_page_handler.go — 漫画课件页面HTTP入口
//
// 提供：
//   POST /comic-projects/{project_id}/insert-page
//   POST /comic-projects/{project_id}/panels/{panel_id}/sync-page
//
// insert-page为幂等接口：
//   - 未插入时创建新页；
//   - 已插入时刷新原页；
//   - 插页成功但项目状态尚未记录时，通过稳定项目标记恢复。
//
// sync-page只替换一个漫画格，适用于单格重新生成完成后的页面同步。

import (
	"net/http"

	"tedna/internal/services"
	"tedna/internal/utils"
)

type coursewareComicInsertPageRequest struct {
	ExpectedVersion int `json:"expected_version"`
	InsertAt        int `json:"insert_at"`
}

type coursewareComicSyncPanelPageRequest struct {
	ExpectedVersion int `json:"expected_version"`
}

// CoursewareComicPageHandler 漫画页面处理器。
type CoursewareComicPageHandler struct {
	service *services.CoursewareComicPageService
}

// NewCoursewareComicPageHandler 创建漫画页面处理器。
func NewCoursewareComicPageHandler(
	service *services.CoursewareComicPageService,
) *CoursewareComicPageHandler {
	return &CoursewareComicPageHandler{
		service: service,
	}
}

// InsertPage 首次插入或刷新完整漫画页面。
func (h *CoursewareComicPageHandler) InsertPage(
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

	var request coursewareComicInsertPageRequest

	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicPromptRequestMaxBytes,
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
		h.service.InsertOrRefreshPage(
			r.Context(),
			coursewareID,
			projectID,
			request.ExpectedVersion,
			request.InsertAt,
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

// SyncPanelPage 将最新单格图片同步到已插入漫画页。
func (h *CoursewareComicPageHandler) SyncPanelPage(
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

	var request coursewareComicSyncPanelPageRequest

	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicPromptRequestMaxBytes,
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
		h.service.SyncPanelToInsertedPage(
			r.Context(),
			coursewareID,
			projectID,
			panelID,
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
