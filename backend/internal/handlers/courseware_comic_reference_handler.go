package handlers

// courseware_comic_reference_handler.go — 知识点漫画参考资源HTTP处理器
//
// HTTP协议：
//   GET    /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/references
//   POST   /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/references
//   DELETE /api/v1/coursewares/{courseware_id}/comic-projects/{project_id}/references/{reference_id}
//
// 安全原则：
//   - 所有入口重新解析当前登录用户和课件作者运行边界；
//   - 正式教材、课件和课程大纲由服务端重新读取；
//   - 文档只接收浏览器提取的文字，不接收原始二进制；
//   - 图片只绑定当前课件中的正式图片资产；
//   - 返回浏览器的资源视图不包含正文和压缩摘要；
//   - 不开放IAOCI、图片提示词或任何内部生成协议。

import (
	"errors"
	"net/http"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const coursewareComicReferenceRequestMaxBytes int64 =
	1 << 20

type coursewareComicReferenceDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// CreateProjectReference 新增一项可选参考资源。
func (h *CoursewareComicHandler) CreateProjectReference(
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

	var request models.CreateCoursewareComicReferenceRequest

	if !decodeCoursewareComicJSON(
		w,
		r,
		&request,
		coursewareComicReferenceRequestMaxBytes,
	) {
		return
	}

	result, err :=
		h.resolveProjectService().
			CreateProjectReference(
				r.Context(),
				coursewareID,
				projectID,
				actor,
				&request,
			)
	if err != nil {
		writeCoursewareComicReferenceHandlerError(
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

// ListProjectReferences 返回当前项目的安全参考资源列表。
func (h *CoursewareComicHandler) ListProjectReferences(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
) {
	if r.Method != http.MethodGet {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持GET请求",
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

	result, err :=
		h.resolveProjectService().
			ListProjectReferencesForBrowser(
				r.Context(),
				coursewareID,
				projectID,
				actor,
			)
	if err != nil {
		writeCoursewareComicReferenceHandlerError(
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

// DeleteProjectReference 删除项目参考资源绑定。
//
// 图片资源只解除漫画项目绑定，不删除courseware_assets记录。
func (h *CoursewareComicHandler) DeleteProjectReference(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
	projectID string,
	referenceID string,
) {
	if r.Method != http.MethodDelete {
		coursewareComicMethodNotAllowed(
			w,
			"仅支持DELETE请求",
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

	if err :=
		h.resolveProjectService().
			DeleteProjectReference(
				r.Context(),
				coursewareID,
				projectID,
				referenceID,
				actor,
			); err != nil {
		writeCoursewareComicReferenceHandlerError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		&coursewareComicReferenceDeleteResponse{
			ID:      referenceID,
			Deleted: true,
		},
	)
}

// writeCoursewareComicReferenceHandlerError
// 为参考资源新增稳定、可预测的HTTP错误映射。
//
// 未在本模块识别的既有课件或漫画错误继续交给统一错误处理器，
// 避免复制作者运行边界、审核锁和课件状态的既有映射。
func writeCoursewareComicReferenceHandlerError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		return

	case errors.Is(
		err,
		repository.ErrCoursewareComicReferenceNotFound,
	),
		errors.Is(
			err,
			repository.ErrCoursewareComicProjectNotFound,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicReferenceSourceUnavailable,
		),
		errors.Is(
			err,
			services.ErrCoursewareComicReferenceImageUnavailable,
		):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		repository.ErrCoursewareComicReferenceConflict,
	),
		errors.Is(
			err,
			repository.ErrCoursewareComicProjectNotEditable,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			err.Error(),
		)

	case errors.Is(
		err,
		repository.ErrCoursewareComicReferenceLimitReached,
		):
		utils.Fail(
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewareComicReferenceContentTooLong,
	),
		errors.Is(
			err,
			services.ErrCoursewareComicReferenceSummaryTooLong,
		):
		utils.Fail(
			w,
			http.StatusRequestEntityTooLarge,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewareComicReferenceInvalidRequest,
	),
		errors.Is(
			err,
			services.ErrCoursewareComicReferenceDocumentTypeUnsupported,
		),
		errors.Is(
			err,
			repository.ErrCoursewareComicReferenceInvalid,
		):
		utils.Fail(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	default:
		writeCoursewareComicHandlerError(
			w,
			err,
		)
	}
}
