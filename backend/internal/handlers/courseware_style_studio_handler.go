package handlers

// courseware_style_studio_handler.go — AI美术风格工作室HTTP处理器
//
// 本处理器保持独立，不继续扩张CoursewareAssetHandler。
//
// 处理流程：
//   1. 解析风格工作室路径；
//   2. 从认证中间件上下文读取JWT身份；
//   3. 构造可信课件Actor并执行作者权限预检；
//   4. 上传接口必须先授权，再解析multipart；
//   5. 预览和确认接口显式接收reference_mode；
//   6. 调用CoursewareStyleStudioService；
//   7. 将会话、模式和确认图片来源错误映射为明确HTTP状态。

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const (
	coursewareStyleStudioJSONMaxBytes = 1 << 20

	coursewareStyleStudioUploadMaxBytes = 9 << 20
)

// CoursewareStyleStudioHandler AI美术风格工作室处理器。
type CoursewareStyleStudioHandler struct {
	service *services.CoursewareStyleStudioService
}

// NewCoursewareStyleStudioHandler 创建风格工作室处理器。
func NewCoursewareStyleStudioHandler(
	service *services.CoursewareStyleStudioService,
) *CoursewareStyleStudioHandler {
	return &CoursewareStyleStudioHandler{
		service: service,
	}
}

// Handle 统一分发风格工作室全部HTTP操作。
func (h *CoursewareStyleStudioHandler) Handle(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil ||
		h.service == nil {
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"AI美术风格工作室服务未初始化",
		)
		return
	}

	parsedPath, err :=
		parseCoursewareStyleStudioPath(
			r.URL.Path,
		)
	if err != nil {
		utils.BadRequest(
			w,
			err.Error(),
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok ||
		claims == nil {
		utils.Unauthorized(
			w,
			"未登录",
		)
		return
	}

	// 参考图上传必须在ParseMultipartForm之前完成授权。
	// 其它操作也统一执行作者权限预检，服务层仍会进行二次校验。
	actor, allowed :=
		requireCoursewareAssetOwnerActor(
			w,
			r,
			parsedPath.CoursewareID,
			claims.UserID,
			claims.Role,
		)
	if !allowed {
		return
	}

	switch parsedPath.Action {
	case coursewareStyleStudioActionActive:
		h.handleStyleStudioActive(
			w,
			r,
			parsedPath,
			actor,
		)

	case coursewareStyleStudioActionSessions:
		h.handleStyleStudioCreateSession(
			w,
			r,
			parsedPath,
			actor,
		)

	case coursewareStyleStudioActionSession:
		h.handleStyleStudioSession(
			w,
			r,
			parsedPath,
			actor,
		)

	case coursewareStyleStudioActionMessages:
		h.handleStyleStudioMessage(
			w,
			r,
			parsedPath,
			actor,
		)

	case coursewareStyleStudioActionPreviews:
		h.handleStyleStudioPreviews(
			w,
			r,
			parsedPath,
			actor,
		)

	case coursewareStyleStudioActionConfirm:
		h.handleStyleStudioConfirm(
			w,
			r,
			parsedPath,
			actor,
		)

	case coursewareStyleStudioActionUploadReference:
		h.handleStyleStudioUploadReference(
			w,
			r,
			parsedPath,
			actor,
		)

	default:
		utils.Fail(
			w,
			http.StatusNotFound,
			"未找到风格工作室操作",
		)
	}
}

// handleStyleStudioActive 恢复当前活动会话。
func (h *CoursewareStyleStudioHandler) handleStyleStudioActive(
	w http.ResponseWriter,
	r *http.Request,
	path *coursewareStyleStudioPath,
	actor *services.CoursewareActorContext,
) {
	if r.Method != http.MethodGet {
		styleStudioMethodNotAllowed(
			w,
			"当前风格会话仅支持GET请求",
		)
		return
	}

	state, err :=
		h.service.GetActiveState(
			r.Context(),
			path.CoursewareID,
			actor,
		)
	if err != nil {
		handleCoursewareStyleStudioError(
			w,
			err,
		)
		return
	}

	utils.Success(w, state)
}

// handleStyleStudioCreateSession 创建新的风格共创会话。
func (h *CoursewareStyleStudioHandler) handleStyleStudioCreateSession(
	w http.ResponseWriter,
	r *http.Request,
	path *coursewareStyleStudioPath,
	actor *services.CoursewareActorContext,
) {
	if r.Method != http.MethodPost {
		styleStudioMethodNotAllowed(
			w,
			"创建风格会话仅支持POST请求",
		)
		return
	}

	request :=
		&models.CreateCoursewareStyleSessionRequest{}

	if !decodeCoursewareStyleStudioJSON(
		w,
		r,
		request,
		true,
	) {
		return
	}

	state, err :=
		h.service.CreateSession(
			r.Context(),
			path.CoursewareID,
			request,
			actor,
		)
	if err != nil {
		handleCoursewareStyleStudioError(
			w,
			err,
		)
		return
	}

	utils.Success(w, state)
}

// handleStyleStudioSession 查询或归档指定会话。
func (h *CoursewareStyleStudioHandler) handleStyleStudioSession(
	w http.ResponseWriter,
	r *http.Request,
	path *coursewareStyleStudioPath,
	actor *services.CoursewareActorContext,
) {
	switch r.Method {
	case http.MethodGet:
		state, err :=
			h.service.GetState(
				r.Context(),
				path.CoursewareID,
				path.SessionID,
				actor,
			)
		if err != nil {
			handleCoursewareStyleStudioError(
				w,
				err,
			)
			return
		}

		utils.Success(w, state)

	case http.MethodDelete:
		if err :=
			h.service.ArchiveSession(
				r.Context(),
				path.CoursewareID,
				path.SessionID,
				actor,
			); err != nil {
			handleCoursewareStyleStudioError(
				w,
				err,
			)
			return
		}

		utils.Success(
			w,
			map[string]string{
				"message": "风格共创会话已归档",
			},
		)

	default:
		styleStudioMethodNotAllowed(
			w,
			"风格会话仅支持GET或DELETE请求",
		)
	}
}

// handleStyleStudioMessage 发送一轮文字或参考图要求。
func (h *CoursewareStyleStudioHandler) handleStyleStudioMessage(
	w http.ResponseWriter,
	r *http.Request,
	path *coursewareStyleStudioPath,
	actor *services.CoursewareActorContext,
) {
	if r.Method != http.MethodPost {
		styleStudioMethodNotAllowed(
			w,
			"发送风格消息仅支持POST请求",
		)
		return
	}

	request :=
		&models.CoursewareStyleTurnRequest{}

	if !decodeCoursewareStyleStudioJSON(
		w,
		r,
		request,
		false,
	) {
		return
	}

	result, err :=
		h.service.SendTurn(
			r.Context(),
			path.CoursewareID,
			path.SessionID,
			request,
			actor,
		)
	if err != nil {
		handleCoursewareStyleStudioError(
			w,
			err,
		)
		return
	}

	utils.Success(w, result)
}

// handleStyleStudioPreviews 生成三类风格测试图。
//
// 请求体允许为空以兼容旧客户端。
// 新客户端提交：{"reference_mode":"style_only"}。
func (h *CoursewareStyleStudioHandler) handleStyleStudioPreviews(
	w http.ResponseWriter,
	r *http.Request,
	path *coursewareStyleStudioPath,
	actor *services.CoursewareActorContext,
) {
	if r.Method != http.MethodPost {
		styleStudioMethodNotAllowed(
			w,
			"生成风格预览仅支持POST请求",
		)
		return
	}

	request :=
		&models.GenerateCoursewareStylePreviewsRequest{}

	if !decodeCoursewareStyleStudioJSON(
		w,
		r,
		request,
		true,
	) {
		return
	}

	state, err :=
		h.service.GeneratePreviewsWithRequest(
			r.Context(),
			path.CoursewareID,
			path.SessionID,
			request,
			actor,
		)
	if err != nil {
		handleCoursewareStyleStudioError(
			w,
			err,
		)
		return
	}

	utils.Success(w, state)
}

// handleStyleStudioConfirm 确认正式课程美术风格。
func (h *CoursewareStyleStudioHandler) handleStyleStudioConfirm(
	w http.ResponseWriter,
	r *http.Request,
	path *coursewareStyleStudioPath,
	actor *services.CoursewareActorContext,
) {
	if r.Method != http.MethodPost {
		styleStudioMethodNotAllowed(
			w,
			"确认课程风格仅支持POST请求",
		)
		return
	}

	request :=
		&models.ConfirmCoursewareStyleSessionRequest{}

	if !decodeCoursewareStyleStudioJSON(
		w,
		r,
		request,
		false,
	) {
		return
	}

	state, err :=
		h.service.ConfirmSession(
			r.Context(),
			path.CoursewareID,
			path.SessionID,
			request,
			actor,
		)
	if err != nil {
		handleCoursewareStyleStudioError(
			w,
			err,
		)
		return
	}

	utils.Success(w, state)
}

// handleStyleStudioUploadReference 上传课程级参考图片。
func (h *CoursewareStyleStudioHandler) handleStyleStudioUploadReference(
	w http.ResponseWriter,
	r *http.Request,
	path *coursewareStyleStudioPath,
	actor *services.CoursewareActorContext,
) {
	if r.Method != http.MethodPost {
		styleStudioMethodNotAllowed(
			w,
			"上传风格参考图仅支持POST请求",
		)
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		coursewareStyleStudioUploadMaxBytes,
	)

	if err := r.ParseMultipartForm(
		4 << 20,
	); err != nil {
		utils.BadRequest(
			w,
			"参考图片解析失败: "+err.Error(),
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
			"缺少参考图片字段file",
		)
		return
	}
	defer file.Close()

	result, err :=
		h.service.UploadReferenceImage(
			r.Context(),
			path.CoursewareID,
			actor,
			file,
			header,
		)
	if err != nil {
		handleCoursewareStyleStudioError(
			w,
			err,
		)
		return
	}

	utils.Success(w, result)
}

// decodeCoursewareStyleStudioJSON 安全解析小体积JSON请求。
func decodeCoursewareStyleStudioJSON(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
	allowEmpty bool,
) bool {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		coursewareStyleStudioJSONMaxBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(target)

	if errors.Is(err, io.EOF) &&
		allowEmpty {
		return true
	}

	if err != nil {
		utils.BadRequest(
			w,
			"请求参数格式错误: "+err.Error(),
		)
		return false
	}

	var extra interface{}

	err = decoder.Decode(&extra)
	if !errors.Is(err, io.EOF) {
		utils.BadRequest(
			w,
			"请求体只能包含一个JSON对象",
		)
		return false
	}

	return true
}

// handleCoursewareStyleStudioError 映射风格工作室业务错误。
func handleCoursewareStyleStudioError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		repository.ErrCoursewareStyleSessionNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		repository.ErrCoursewareStyleSessionNotEditable,
	),
		errors.Is(
			err,
			repository.ErrCoursewareStylePreviewModeStale,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			err.Error(),
		)

	case errors.Is(
		err,
		repository.ErrCoursewareStyleAssetInvalid,
	),
		errors.Is(
			err,
			repository.ErrCoursewareStyleConfirmAssetInvalid,
		),
		errors.Is(
			err,
			repository.ErrCoursewareStyleMessageInvalid,
		):
		utils.BadRequest(
			w,
			err.Error(),
		)

	default:
		// 课件作者、教育域和Actor错误继续使用既有统一映射。
		// AI调用、图片生成、文件保存等错误保留真实错误正文。
		handleCoursewareAssetServiceError(
			w,
			err,
		)
	}
}

func styleStudioMethodNotAllowed(
	w http.ResponseWriter,
	message string,
) {
	utils.Fail(
		w,
		http.StatusMethodNotAllowed,
		message,
	)
}
