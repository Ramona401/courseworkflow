package handlers

// lesson_plan_word_handler.go — 原格式Word教案HTTP入口
//
// 当前提供两个端点：
//
//  1. POST /api/v1/lesson-plans/plans/import-word/preview
//     multipart/form-data：file=普通.docx文件
//     只创建24小时短时导入会话并返回浏览器安全预览。
//
//  2. GET /api/v1/lesson-plans/plans/{id}/word-document/download
//     只允许教案作者下载仍与当前正文同步的原格式Word文件。
//
// 两个端点均不暴露storage_key、文件哈希或服务器物理路径。

import (
	"errors"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"tedna/internal/services"
	"tedna/internal/utils"
)

const (
	// multipart边界、字段头、UTF-8文件名和其它表单元数据预留2MB。
	// 真正DOCX文件仍严格限制为30MB。
	lessonPlanWordMultipartOverhead = int64(2 * 1024 * 1024)

	// ParseMultipartForm参数是内存阈值，不是文件总大小上限。
	// 超过8MB的文件部分写入系统临时目录，并在请求结束时RemoveAll。
	lessonPlanWordMultipartMemory = int64(8 * 1024 * 1024)

	lessonPlanWordDownloadPathPrefix = "/api/v1/lesson-plans/plans/"
	lessonPlanWordDownloadPathSuffix = "/word-document/download"
)

// PreviewWordImport 上传并预解析一份保留原格式的DOCX。
func (h *LessonPlanGenHandler) PreviewWordImport(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	callerID := getCurrentUserID(r)
	if callerID == "" {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	// 在读取请求体前先验证媒体类型。
	//
	// application/json通常表示前端没有清除Axios全局JSON请求头；
	// multipart/form-data缺少boundary时也无法安全解析。
	mediaType, parameters, mediaTypeErr :=
		mime.ParseMediaType(
			r.Header.Get("Content-Type"),
		)

	if mediaTypeErr != nil ||
		mediaType != "multipart/form-data" ||
		parameters["boundary"] == "" {
		lpGenHandlerLog.Warn(
			"Word上传请求不是有效multipart表单",
			"caller_id",
			callerID,
			"content_type",
			r.Header.Get("Content-Type"),
			"content_length",
			r.ContentLength,
			"error",
			mediaTypeErr,
		)

		utils.BadRequest(
			w,
			"Word上传请求格式无效，请刷新页面后重新选择文件",
		)
		return
	}

	// MaxBytesReader限制整个HTTP请求体。
	//
	// 文件最大30MB，另为multipart元数据预留2MB；
	// 服务层仍会再次按实际落盘字节严格执行30MB文件上限。
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		services.MaxLessonPlanWordFileSize+
			lessonPlanWordMultipartOverhead,
	)

	if err := r.ParseMultipartForm(
		lessonPlanWordMultipartMemory,
	); err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(
			err,
			&maxBytesError,
		) {
			lpGenHandlerLog.Warn(
				"Word上传请求体超过限制",
				"caller_id",
				callerID,
				"content_length",
				r.ContentLength,
				"limit",
				maxBytesError.Limit,
			)

			utils.Fail(
				w,
				http.StatusRequestEntityTooLarge,
				services.
					ErrLessonPlanWordFileTooLarge.
					Error(),
			)
			return
		}

		lpGenHandlerLog.Warn(
			"解析Word上传multipart表单失败",
			"caller_id",
			callerID,
			"content_type",
			r.Header.Get("Content-Type"),
			"content_length",
			r.ContentLength,
			"error",
			err,
		)

		utils.BadRequest(
			w,
			"Word上传表单格式无效，请重新选择文件后重试",
		)
		return
	}

	if r.MultipartForm != nil {
		defer func() {
			_ = r.MultipartForm.RemoveAll()
		}()
	}

	file, header, err :=
		r.FormFile("file")
	if err != nil {
		utils.BadRequest(
			w,
			"请选择要上传的Word文档",
		)
		return
	}
	defer file.Close()

	response, err :=
		h.genService.
			PreviewLessonPlanWordImport(
				r.Context(),
				file,
				header,
				callerID,
			)
	if err != nil {
		handleLessonPlanWordImportError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		response,
	)
}

// DownloadWordDocument 下载作者本人的当前原格式Word文件。
func (h *LessonPlanGenHandler) DownloadWordDocument(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"原格式Word下载仅支持GET请求",
		)
		return
	}

	callerID := getCurrentUserID(r)
	if callerID == "" {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	lessonPlanID := extractLessonPlanWordDownloadPlanID(
		r.URL.Path,
	)
	if lessonPlanID == "" {
		utils.Fail(
			w,
			http.StatusNotFound,
			"原格式Word教案不存在",
		)
		return
	}

	download, err := services.OpenLessonPlanWordDownload(
		r.Context(),
		lessonPlanID,
		callerID,
	)
	if err != nil {
		handleLessonPlanWordDownloadError(
			w,
			err,
			lessonPlanID,
			callerID,
		)
		return
	}
	defer download.File.Close()

	// 使用RFC 5987 filename*发送UTF-8原文件名，同时提供ASCII兜底名。
	// 文件来自私有目录，不允许浏览器或中间代理缓存。
	w.Header().Set(
		"Content-Type",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	)
	w.Header().Set(
		"Content-Disposition",
		"attachment; filename=\"lesson-plan.docx\"; filename*=UTF-8''"+
			url.PathEscape(download.FileName),
	)
	w.Header().Set(
		"Cache-Control",
		"private, no-store, max-age=0",
	)
	w.Header().Set(
		"Pragma",
		"no-cache",
	)
	w.Header().Set(
		"X-Content-Type-Options",
		"nosniff",
	)

	http.ServeContent(
		w,
		r,
		download.FileName,
		download.ModTime,
		download.File,
	)
}

// extractLessonPlanWordDownloadPlanID 从精确下载路径中提取教案ID。
func extractLessonPlanWordDownloadPlanID(
	requestPath string,
) string {
	if !strings.HasPrefix(
		requestPath,
		lessonPlanWordDownloadPathPrefix,
	) ||
		!strings.HasSuffix(
			requestPath,
			lessonPlanWordDownloadPathSuffix,
		) {
		return ""
	}

	lessonPlanID := strings.TrimSuffix(
		strings.TrimPrefix(
			requestPath,
			lessonPlanWordDownloadPathPrefix,
		),
		lessonPlanWordDownloadPathSuffix,
	)
	lessonPlanID = strings.TrimSpace(lessonPlanID)

	if lessonPlanID == "" ||
		strings.Contains(lessonPlanID, "/") ||
		lessonPlanID == "." ||
		lessonPlanID == ".." {
		return ""
	}

	return lessonPlanID
}

func handleLessonPlanWordDownloadError(
	w http.ResponseWriter,
	err error,
	lessonPlanID string,
	callerID string,
) {
	switch {
	case errors.Is(
		err,
		services.ErrLessonPlanWordDownloadNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			services.ErrLessonPlanWordDownloadNotFound.Error(),
		)

	case errors.Is(
		err,
		services.ErrLessonPlanWordDownloadOutOfSync,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			services.ErrLessonPlanWordDownloadOutOfSync.Error(),
		)

	case errors.Is(
		err,
		services.ErrLessonPlanWordDownloadUnavailable,
	):
		lpGenHandlerLog.Error(
			"原格式Word文件安全校验未通过",
			"lesson_plan_id",
			lessonPlanID,
			"caller_id",
			callerID,
			"error",
			err,
		)
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			services.ErrLessonPlanWordDownloadUnavailable.Error(),
		)

	default:
		lpGenHandlerLog.Error(
			"下载原格式Word失败",
			"lesson_plan_id",
			lessonPlanID,
			"caller_id",
			callerID,
			"error",
			err,
		)
		utils.InternalError(
			w,
			"原格式Word下载失败，请稍后重试",
		)
	}
}

func handleLessonPlanWordImportError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrLessonPlanWordFileRequired,
	),
		errors.Is(
			err,
			services.ErrLessonPlanWordFileInvalid,
		):
		utils.BadRequest(
			w,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrLessonPlanWordFileTooLarge,
	):
		utils.Fail(
			w,
			http.StatusRequestEntityTooLarge,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrLessonPlanWordParseFailed,
	):
		utils.Fail(
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrLPGenServiceDraining,
	):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrLPGenUnauthorized,
	),
		errors.Is(
			err,
			services.
				ErrLPCreationEducationDomainRequired,
		),
		errors.Is(
			err,
			services.
				ErrLPCreationEducationDomainConflict,
		):
		utils.Forbidden(
			w,
			err.Error(),
		)

	case errors.Is(
		err,
		services.
			ErrLPCreationEducationDomainResolveFailed,
	):
		lpGenHandlerLog.Error(
			"Word预解析教育域解析失败",
			"error",
			err,
		)
		utils.InternalError(
			w,
			"当前教育域暂时无法确认，请稍后重试",
		)

	default:
		lpGenHandlerLog.Error(
			"Word保真预解析失败",
			"error",
			err,
		)
		utils.InternalError(
			w,
			"Word文档处理失败，请稍后重试",
		)
	}
}
