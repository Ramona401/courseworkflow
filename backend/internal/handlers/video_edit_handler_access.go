package handlers

// video_edit_handler_access.go — FFmpeg入口的路径、Actor、正文和错误治理
//
// 本文件把七个Handler共用的安全流程集中起来：
//   - JWT可信Actor构造；
//   - 作者控制预检；
//   - 严格路径解析；
//   - 512KB正文限制、未知字段和多JSON值拒绝；
//   - 稳定HTTP错误映射。

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

// requireVideoEditActor 从认证上下文构造可信课件Actor。
func requireVideoEditActor(
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

// preflightVideoEditOwner 在解析正文前执行作者控制预检。
//
// 本预检只用于尽早拒绝无权请求；Service仍会重新加载正式课件，
// 并在FFmpeg执行前后及资产写库前再次授权。
func (h *VideoEditHandler) preflightVideoEditOwner(
	w http.ResponseWriter,
	r *http.Request,
	coursewareID string,
) (
	*services.CoursewareActorContext,
	bool,
) {
	actor, ok := requireVideoEditActor(w, r)
	if !ok {
		return nil, false
	}

	scopedActor, err :=
		h.editService.PreflightOwnerMutation(
			r.Context(),
			coursewareID,
			actor,
		)
	if err != nil {
		writeVideoEditError(w, err)
		return nil, false
	}

	return scopedActor, true
}

// decodeVideoEditJSON 严格解析一个JSON对象。
func decodeVideoEditJSON(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
) error {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		services.VideoEditMaxBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return err
	}

	var extra interface{}

	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("请求体包含多个JSON值")
	}

	return err
}

// prepareVideoEditRequest 校验POST方法并严格提取课件ID。
func (h *VideoEditHandler) prepareVideoEditRequest(
	w http.ResponseWriter,
	r *http.Request,
	suffix string,
) (
	string,
	bool,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			"仅支持POST请求",
		)
		return "", false
	}

	coursewareID :=
		extractVideoEditCoursewareID(
			r.URL.Path,
			suffix,
		)
	if coursewareID == "" {
		utils.BadRequest(w, "路径参数错误")
		return "", false
	}

	return coursewareID, true
}

// prepareSingleAssetVideoEdit 处理只有asset_id的公共入口形状。
func (h *VideoEditHandler) prepareSingleAssetVideoEdit(
	w http.ResponseWriter,
	r *http.Request,
	suffix string,
) (
	string,
	*services.CoursewareActorContext,
	string,
	bool,
) {
	coursewareID, ok :=
		h.prepareVideoEditRequest(
			w,
			r,
			suffix,
		)
	if !ok {
		return "", nil, "", false
	}

	actor, ok :=
		h.preflightVideoEditOwner(
			w,
			r,
			coursewareID,
		)
	if !ok {
		return "", nil, "", false
	}

	var body struct {
		AssetID string `json:"asset_id"`
	}

	if err := decodeVideoEditJSON(
		w,
		r,
		&body,
	); err != nil {
		writeVideoEditDecodeError(w, err)
		return "", nil, "", false
	}

	return coursewareID,
		actor,
		body.AssetID,
		true
}

// extractVideoEditCoursewareID 严格解析课件视频编辑路径。
func extractVideoEditCoursewareID(
	path string,
	suffix string,
) string {
	const prefix = "/api/v1/coursewares/"

	trimmed := strings.TrimRight(
		strings.TrimSpace(path),
		"/",
	)
	if !strings.HasPrefix(trimmed, prefix) ||
		!strings.HasSuffix(trimmed, suffix) {
		return ""
	}

	coursewareID := strings.TrimSuffix(
		strings.TrimPrefix(
			trimmed,
			prefix,
		),
		suffix,
	)

	if coursewareID == "" ||
		strings.Contains(coursewareID, "/") {
		return ""
	}

	return coursewareID
}

// writeVideoEditDecodeError 映射正文解析错误。
func writeVideoEditDecodeError(
	w http.ResponseWriter,
	err error,
) {
	var maxBytesError *http.MaxBytesError

	if errors.As(err, &maxBytesError) {
		utils.Fail(
			w,
			http.StatusRequestEntityTooLarge,
			"视频编辑请求体不能超过512KB",
		)
		return
	}

	utils.BadRequest(
		w,
		"视频编辑请求格式错误",
	)
}

// writeVideoEditError 映射稳定业务错误。
func writeVideoEditError(
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
			services.ErrVideoEditAssetNotFound,
		),
		errors.Is(
			err,
			services.ErrCoursewareSubtitleNotFound,
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
			services.ErrCoursewareOwnerRuntimeDenied,
		),
		errors.Is(
			err,
			services.ErrCoursewareEducationDomainMismatch,
		),
		errors.Is(
			err,
			services.ErrCoursewareSubtitleScopeTargetMismatch,
		):
		utils.Fail(
			w,
			http.StatusForbidden,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrVideoEditInputInvalid,
	):
		utils.BadRequest(
			w,
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrCoursewareControlMutationLocked,
	),
		errors.Is(
			err,
			services.ErrVideoEditBusy,
		),
		errors.Is(
			err,
			services.ErrVideoEditSourceChanged,
		),
		errors.Is(
			err,
			services.ErrCoursewareSubtitleMutationConflict,
		):
		utils.Fail(
			w,
			http.StatusConflict,
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
			"视频编辑服务异常，请稍后重试",
		)
	}
}
