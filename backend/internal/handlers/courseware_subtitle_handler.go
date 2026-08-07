package handlers

// courseware_subtitle_handler.go — 课件字幕轨 HTTP 处理器
//
// v0.42.8 新增 5 个端点：
//   POST   /api/v1/coursewares/{id}/subtitles          — 创建/更新字幕轨
//   GET    /api/v1/coursewares/{id}/subtitles           — 查询字幕轨列表
//   DELETE /api/v1/coursewares/{id}/subtitles/{sub_id}  — 删除字幕轨
//   POST   /api/v1/coursewares/{id}/subtitles/{sub_id}/export-srt   — 导出 SRT
//   POST   /api/v1/coursewares/{id}/subtitles/{sub_id}/burn-in      — FFmpeg 硬字幕烧录

import (
        "encoding/json"
        "errors"
        "net/http"
        "strings"

        "github.com/google/uuid"

        "tedna/internal/ai"
        "tedna/internal/middleware"
        "tedna/internal/models"
        "tedna/internal/repository"
        "tedna/internal/services"
        "tedna/internal/utils"
)

// CoursewareSubtitleHandler 字幕轨处理器
type CoursewareSubtitleHandler struct {
	subtitleService *services.CoursewareSubtitleService
}

// NewCoursewareSubtitleHandler 创建字幕轨处理器
func NewCoursewareSubtitleHandler(svc *services.CoursewareSubtitleService) *CoursewareSubtitleHandler {
	return &CoursewareSubtitleHandler{subtitleService: svc}
}

// ==================== 创建/更新字幕轨 ====================

// UpsertSubtitle POST /api/v1/coursewares/{id}/subtitles
func (h *CoursewareSubtitleHandler) UpsertSubtitle(
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

	coursewareID :=
		extractSubtitleCoursewareID(
			r.URL.Path,
		)
	if coursewareID == "" {
		utils.BadRequest(
			w,
			"无效的课件ID",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	scopedActor, err :=
		authorizeCoursewareSubtitleRefine(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareSubtitleError(
			w,
			err,
		)
		return
	}

	var req models.UpsertSubtitleRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求体解析失败",
		)
		return
	}

	subtitle, err :=
		h.subtitleService.UpsertSubtitle(
			r.Context(),
			coursewareID,
			scopedActor,
			&req,
		)
	if err != nil {
		writeCoursewareSubtitleError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		subtitle,
	)
}

// ==================== 查询字幕轨列表 ====================

// ListSubtitles GET /api/v1/coursewares/{id}/subtitles?scope_type=x&scope_id=y
func (h *CoursewareSubtitleHandler) ListSubtitles(
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

	coursewareID :=
		extractSubtitleCoursewareID(
			r.URL.Path,
		)
	if coursewareID == "" {
		utils.BadRequest(
			w,
			"无效的课件ID",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	actor, err :=
		authorizeCoursewareSubtitleView(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareSubtitleError(
			w,
			err,
		)
		return
	}

	items, err :=
		h.subtitleService.ListSubtitles(
			r.Context(),
			coursewareID,
			actor,
			r.URL.Query().Get("scope_type"),
			r.URL.Query().Get("scope_id"),
		)
	if err != nil {
		writeCoursewareSubtitleError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		items,
	)
}

// ==================== 删除字幕轨 ====================

// DeleteSubtitle DELETE /api/v1/coursewares/{id}/subtitles/{sub_id}
func (h *CoursewareSubtitleHandler) DeleteSubtitle(
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

	coursewareID :=
		extractSubtitleCoursewareID(
			r.URL.Path,
		)
	subtitleID :=
		extractSubtitleID(
			r.URL.Path,
		)

	if coursewareID == "" ||
		subtitleID == "" {
		utils.BadRequest(
			w,
			"无效的路径参数",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	scopedActor, err :=
		authorizeCoursewareSubtitleRefine(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareSubtitleError(
			w,
			err,
		)
		return
	}

	if err :=
		h.subtitleService.DeleteSubtitle(
			r.Context(),
			coursewareID,
			subtitleID,
			scopedActor,
		); err != nil {
		writeCoursewareSubtitleError(
			w,
			err,
		)
		return
	}

	utils.Success(
		w,
		map[string]string{
			"message": "删除成功",
		},
	)
}

// ==================== 导出 SRT ====================

// ExportSRT POST /api/v1/coursewares/{id}/subtitles/{sub_id}/export-srt
func (h *CoursewareSubtitleHandler) ExportSRT(
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

	coursewareID :=
		extractSubtitleCoursewareID(
			r.URL.Path,
		)
	subtitleID :=
		extractSubtitleID(
			r.URL.Path,
		)

	if coursewareID == "" ||
		subtitleID == "" {
		utils.BadRequest(
			w,
			"无效的路径参数",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(w, "未认证")
		return
	}

	actor, err :=
		authorizeCoursewareSubtitleView(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareSubtitleError(
			w,
			err,
		)
		return
	}

	srtContent, err :=
		h.subtitleService.ExportSRT(
			r.Context(),
			coursewareID,
			subtitleID,
			actor,
		)
	if err != nil {
		writeCoursewareSubtitleError(
			w,
			err,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"text/plain; charset=utf-8",
	)
	w.Header().Set(
		"Content-Disposition",
		"attachment; filename=subtitle.srt",
	)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(
		[]byte(srtContent),
	)
}

// ==================== 硬字幕烧录 ====================

// BurnInSubtitle POST /api/v1/coursewares/{id}/subtitles/{sub_id}/burn-in
func (h *CoursewareSubtitleHandler) BurnInSubtitle(
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

	coursewareID :=
		extractSubtitleCoursewareID(
			r.URL.Path,
		)
	subtitleID :=
		extractSubtitleID(
			r.URL.Path,
		)

	if coursewareID == "" ||
		subtitleID == "" {
		utils.BadRequest(
			w,
			"无效的路径参数",
		)
		return
	}

	claims, ok :=
		middleware.GetClaims(
			r.Context(),
		)
	if !ok || claims == nil {
		utils.Unauthorized(
			w,
			"未认证",
		)
		return
	}

	scopedActor, err :=
		authorizeCoursewareSubtitleOwnerControl(
			r.Context(),
			coursewareID,
			claims.UserID,
			claims.Role,
		)
	if err != nil {
		writeCoursewareSubtitleError(
			w,
			err,
		)
		return
	}

	var req models.BurnInSubtitleRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		utils.BadRequest(
			w,
			"请求体解析失败",
		)
		return
	}

	if strings.TrimSpace(
		req.VideoAssetID,
	) == "" {
		utils.BadRequest(
			w,
			"video_asset_id 为必填",
		)
		return
	}

	result, err :=
		h.subtitleService.BurnInSubtitle(
			r.Context(),
			coursewareID,
			subtitleID,
			req.VideoAssetID,
			scopedActor,
		)
	if err != nil {
		writeCoursewareSubtitleError(
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

// ==================== v0.42.9 TTS 配音 ====================

// GenerateTTS POST /api/v1/coursewares/{id}/subtitles/{sub_id}/generate-tts
func (h *CoursewareSubtitleHandler) GenerateTTS(
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

        coursewareID :=
                extractSubtitleCoursewareID(
                        r.URL.Path,
                )

        subtitleID :=
                extractSubtitleID(
                        r.URL.Path,
                )

        if coursewareID == "" ||
                subtitleID == "" {
                utils.BadRequest(
                        w,
                        "无效的路径参数",
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
                        "未认证",
                )
                return
        }

        scopedActor, err :=
                authorizeCoursewareSubtitleOwnerControl(
                        r.Context(),
                        coursewareID,
                        claims.UserID,
                        claims.Role,
                )

        if err != nil {
                writeCoursewareSubtitleError(
                        w,
                        err,
                )
                return
        }

        var request struct {
                Voice       string   `json:"voice"`
                Speed       float64  `json:"speed,omitempty"`
                SegmentIDs  []string `json:"segment_ids,omitempty"`
                OperationID string   `json:"operation_id"`
        }

        if err :=
                json.NewDecoder(
                        r.Body,
                ).Decode(
                        &request,
                ); err != nil {
                utils.BadRequest(
                        w,
                        "请求体解析失败",
                )
                return
        }

        request.Voice =
                strings.TrimSpace(
                        request.Voice,
                )

        request.OperationID =
                strings.TrimSpace(
                        request.OperationID,
                )

        if request.Voice == "" {
                utils.BadRequest(
                        w,
                        "voice（音色代码）为必填",
                )
                return
        }

        if _, err :=
                uuid.Parse(
                        request.OperationID,
                ); err != nil {
                utils.BadRequest(
                        w,
                        "配音任务标识无效，请刷新窗口后重试",
                )
                return
        }

        result, err :=
                h.subtitleService.GenerateTTS(
                        r.Context(),
                        coursewareID,
                        subtitleID,
                        scopedActor,
                        request.Voice,
                        request.Speed,
                        request.SegmentIDs,
                        request.OperationID,
                )

        if err != nil {
                writeCoursewareSubtitleTTSError(
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

// writeCoursewareSubtitleTTSError 映射TTS积分和恢复错误。
//
// 普通用户不接收供应商、模型、内部幂等键、单价或成本数据。
func writeCoursewareSubtitleTTSError(
        w http.ResponseWriter,
        err error,
) {
        switch {
        case errors.Is(
                err,
                services.ErrMediaBillingPriceNotConfigured,
        ):
                utils.Fail(
                        w,
                        http.StatusServiceUnavailable,
                        "TTS积分计费尚未配置，请联系管理员",
                )

        case errors.Is(
                err,
                repository.ErrInsufficientBalance,
        ):
                utils.Fail(
                        w,
                        http.StatusPaymentRequired,
                        "积分余额不足，暂时无法生成配音",
                )

        case errors.Is(
                err,
                repository.ErrTokenAccountNotFound,
        ):
                utils.Fail(
                        w,
                        http.StatusConflict,
                        "尚未开通个人积分账户，暂时无法生成配音",
                )

        case errors.Is(
                err,
                repository.ErrAccountSuspended,
        ):
                utils.Fail(
                        w,
                        http.StatusForbidden,
                        "积分账户当前不可用，请联系管理员",
                )

        case errors.Is(
                err,
                services.ErrCoursewareTTSBillingInProgress,
        ),
                errors.Is(
                        err,
                        services.ErrCoursewareTTSBillingPending,
                ):
                utils.Fail(
                        w,
                        http.StatusConflict,
                        "同一字幕配音任务正在处理或等待恢复，请稍后重试",
                )

        case errors.Is(
                err,
                services.ErrCoursewareTTSBillingTerminal,
        ):
                utils.Fail(
                        w,
                        http.StatusConflict,
                        "该字幕配音任务已经结束，请重新发起生成",
                )

        case errors.Is(
                err,
                services.ErrCoursewareTTSBillingIdentityMismatch,
        ):
                utils.Fail(
                        w,
                        http.StatusConflict,
                        "字幕文本、音色或语速已经变化，请重新发起配音",
                )

        case errors.Is(
                err,
                services.ErrCoursewareTTSBillingOutputMissing,
        ):
                utils.Fail(
                        w,
                        http.StatusConflict,
                        "配音调用已经完成，但音频文件未正确形成，请联系管理员处理",
                )

        default:
                writeCoursewareSubtitleError(
                        w,
                        err,
                )
        }
}

// ListTTSVoices GET /api/v1/tts-voices?language=zh-CN
func (h *CoursewareSubtitleHandler) ListTTSVoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, "仅支持GET")
		return
	}

	language := r.URL.Query().Get("language")
	voices := ai.GetTTSVoicesByLanguage(language)

	utils.Success(w, map[string]interface{}{
		"voices": voices,
		"total":  len(voices),
	})
}

// ==================== 路径解析辅助函数 ====================

// extractSubtitleCoursewareID 从 /api/v1/coursewares/{id}/subtitles... 提取课件ID
func extractSubtitleCoursewareID(path string) string {
	// 路径格式: /api/v1/coursewares/{courseware_id}/subtitles[/...]
	const prefix = "/api/v1/coursewares/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := path[len(prefix):]
	// 找到下一个 /
	slashIdx := strings.Index(rest, "/")
	if slashIdx <= 0 {
		return ""
	}
	return rest[:slashIdx]
}

// extractSubtitleID 从 /api/v1/coursewares/{id}/subtitles/{sub_id}[/...] 提取字幕ID
func extractSubtitleID(path string) string {
	// 路径格式: /api/v1/coursewares/{cw_id}/subtitles/{sub_id}[/export-srt|/burn-in]
	idx := strings.Index(path, "/subtitles/")
	if idx < 0 {
		return ""
	}
	rest := path[idx+len("/subtitles/"):]
	// 去掉后缀 /export-srt 或 /burn-in
	slashIdx := strings.Index(rest, "/")
	if slashIdx > 0 {
		return rest[:slashIdx]
	}
	// 去除尾部斜杠
	return strings.TrimSuffix(rest, "/")
}
