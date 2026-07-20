package handlers

// video_draft_handler.go — 视频编辑器草稿HTTP处理器
//
// 路由：
//   POST   /api/v1/coursewares/{id}/video-drafts
//   GET    /api/v1/coursewares/{id}/video-drafts
//   DELETE /api/v1/coursewares/{id}/video-drafts/{draft_id}
//
// Handler职责：
//   - 严格解析路径与HTTP方法；
//   - 从JWT读取可信操作者身份；
//   - 保存入口在解析正文前完成课件微调权限预检；
//   - 限制请求体大小并解析JSON；
//   - 调用VideoDraftService执行正式权限与业务治理；
//   - 映射稳定HTTP错误，不向浏览器泄露数据库内部错误。

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"tedna/internal/middleware"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// VideoDraftHandler 视频编辑器草稿处理器。
type VideoDraftHandler struct {
	draftService *services.VideoDraftService
}

// NewVideoDraftHandler 创建视频草稿处理器。
//
// 保持无参数构造函数，避免改变现有路由接线。
func NewVideoDraftHandler() *VideoDraftHandler {
	return &VideoDraftHandler{
		draftService: services.NewVideoDraftService(),
	}
}

// videoDraftRouteKind 表示视频草稿路径类型。
type videoDraftRouteKind int

const (
	videoDraftRouteInvalid videoDraftRouteKind = iota
	videoDraftRouteCollection
	videoDraftRouteItem
)

// parseVideoDraftPath 严格解析正式视频草稿路径。
//
// 只接受：
//   - /api/v1/coursewares/{courseware_id}/video-drafts
//   - /api/v1/coursewares/{courseware_id}/video-drafts/{draft_id}
func parseVideoDraftPath(
	path string,
) (
	string,
	string,
	videoDraftRouteKind,
) {
	const prefix = "/api/v1/coursewares/"

	trimmed := strings.TrimRight(path, "/")
	if !strings.HasPrefix(trimmed, prefix) {
		return "", "", videoDraftRouteInvalid
	}

	rest := strings.TrimPrefix(trimmed, prefix)
	parts := strings.Split(rest, "/")

	if len(parts) == 2 &&
		parts[0] != "" &&
		parts[1] == "video-drafts" {
		return parts[0], "", videoDraftRouteCollection
	}

	if len(parts) == 3 &&
		parts[0] != "" &&
		parts[1] == "video-drafts" &&
		parts[2] != "" {
		return parts[0], parts[2], videoDraftRouteItem
	}

	return "", "", videoDraftRouteInvalid
}

// requireVideoDraftActor 从请求上下文构造可信课件操作者。
func requireVideoDraftActor(
	w http.ResponseWriter,
	r *http.Request,
) (
	*services.CoursewareActorContext,
	bool,
) {
	claims, ok := middleware.GetClaims(r.Context())
	if !ok || claims == nil {
		utils.Unauthorized(w, "未登录")
		return nil, false
	}

	return services.BuildCoursewareActorFromClaims(
		r.Context(),
		claims.UserID,
		claims.Role,
	), true
}

// HandleDrafts 统一执行严格路由分发。
func (h *VideoDraftHandler) HandleDrafts(
	w http.ResponseWriter,
	r *http.Request,
) {
	_, _, routeKind := parseVideoDraftPath(r.URL.Path)

	switch routeKind {
	case videoDraftRouteCollection:
		switch r.Method {
		case http.MethodGet:
			h.ListDrafts(w, r)

		case http.MethodPost:
			h.SaveDraft(w, r)

		default:
			utils.Fail(
				w,
				http.StatusMethodNotAllowed,
				"仅支持GET或POST",
			)
		}

	case videoDraftRouteItem:
		if r.Method != http.MethodDelete {
			utils.Fail(
				w,
				http.StatusMethodNotAllowed,
				"草稿详情仅支持DELETE",
			)
			return
		}

		h.DeleteDraft(w, r)

	default:
		utils.Fail(
			w,
			http.StatusNotFound,
			"未找到视频草稿路由",
		)
	}
}

// SaveDraft 保存视频编辑器草稿。
func (h *VideoDraftHandler) SaveDraft(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST",
		)
		return
	}

	coursewareID, _, routeKind :=
		parseVideoDraftPath(r.URL.Path)

	if routeKind != videoDraftRouteCollection ||
		coursewareID == "" {
		utils.BadRequest(
			w,
			"无效的课件草稿路径",
		)
		return
	}

	actor, ok := requireVideoDraftActor(w, r)
	if !ok {
		return
	}

	// 在读取最多2MB正文前先完成课件微调权限预检。
	//
	// 该预检只用于尽早拒绝无权请求，不替代Service正式写库前
	// 重新加载课件并再次授权。
	scopedActor, err := h.draftService.PreflightSaveDraft(
		r.Context(),
		coursewareID,
		actor,
	)
	if err != nil {
		writeVideoDraftError(w, err)
		return
	}

	input, err := decodeVideoDraftSaveInput(w, r)
	if err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(err, &maxBytesError) {
			utils.Fail(
				w,
				http.StatusRequestEntityTooLarge,
				"视频草稿请求体不能超过2MB",
			)
			return
		}

		utils.BadRequest(
			w,
			"视频草稿请求格式错误",
		)
		return
	}

	draft, err := h.draftService.SaveDraft(
		r.Context(),
		coursewareID,
		scopedActor,
		input,
	)
	if err != nil {
		writeVideoDraftError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"id":         draft.ID,
			"created_at": draft.CreatedAt,
			"message":    "草稿保存成功",
		},
	)
}

// decodeVideoDraftSaveInput 限制请求体并确保只包含一个JSON对象。
func decodeVideoDraftSaveInput(
	w http.ResponseWriter,
	r *http.Request,
) (
	*services.VideoDraftSaveInput,
	error,
) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		services.VideoDraftMaxBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	input := &services.VideoDraftSaveInput{}
	if err := decoder.Decode(input); err != nil {
		return nil, err
	}

	var extra interface{}

	err := decoder.Decode(&extra)
	if !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New(
				"请求体包含多个JSON值",
			)
		}

		return nil, err
	}

	return input, nil
}

// ListDrafts 列出当前用户在指定课件中的草稿。
func (h *VideoDraftHandler) ListDrafts(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持GET",
		)
		return
	}

	coursewareID, _, routeKind :=
		parseVideoDraftPath(r.URL.Path)

	if routeKind != videoDraftRouteCollection ||
		coursewareID == "" {
		utils.BadRequest(
			w,
			"无效的课件草稿路径",
		)
		return
	}

	actor, ok := requireVideoDraftActor(w, r)
	if !ok {
		return
	}

	drafts, err := h.draftService.ListDrafts(
		r.Context(),
		coursewareID,
		actor,
	)
	if err != nil {
		writeVideoDraftError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]interface{}{
			"drafts": drafts,
			"total":  len(drafts),
		},
	)
}

// DeleteDraft 删除当前路径课件中的本人草稿。
func (h *VideoDraftHandler) DeleteDraft(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodDelete {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持DELETE",
		)
		return
	}

	coursewareID, draftID, routeKind :=
		parseVideoDraftPath(r.URL.Path)

	if routeKind != videoDraftRouteItem ||
		coursewareID == "" ||
		draftID == "" {
		utils.BadRequest(
			w,
			"无效的视频草稿路径",
		)
		return
	}

	actor, ok := requireVideoDraftActor(w, r)
	if !ok {
		return
	}

	if err := h.draftService.DeleteDraft(
		r.Context(),
		coursewareID,
		draftID,
		actor,
	); err != nil {
		writeVideoDraftError(w, err)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "草稿已删除",
		},
	)
}

// writeVideoDraftError 映射视频草稿稳定业务错误。
func writeVideoDraftError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrCoursewareAccessNotFound,
	),
		errors.Is(
			err,
			services.ErrVideoDraftNotFound,
		):
		utils.Fail(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewareActorRequired,
	),
		errors.Is(
			err,
			services.ErrCoursewareViewDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEditDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainMismatch,
		):
		utils.Fail(
			w,
			http.StatusForbidden,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrVideoDraftInputInvalid,
	):
		utils.BadRequest(
			w,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewareEducationDomainInvalid,
	),
		errors.Is(
			err,
			services.ErrCoursewareRuntimeDomainRequired,
		):
		utils.InternalError(
			w,
			"课件运行环境异常，请联系管理员",
		)

	default:
		utils.InternalError(
			w,
			"视频草稿服务异常，请稍后重试",
		)
	}
}
