package handlers

// lesson_plan_section_rewrite_handler.go — 教案目录段落AI修改HTTP入口。
//
// API：
//   POST /api/v1/lesson-plans/plans/{id}/section-rewrite
//     流式生成段落修改预览，不写数据库。
//
//   POST /api/v1/lesson-plans/plans/{id}/section-rewrite/apply
//     老师确认后原子应用修改，保存完整正文历史版本。
//
// 安全边界：
//   - 身份只从JWT Claims读取。
//   - 教案ID只从URL读取。
//   - 浏览器不能提交作者、学校、角色、教育域或可信段落正文。
//   - 预览接口使用全局SSE排空握手，不设置通配CORS。
//   - 应用接口由Repository再次执行作者、状态、版本和段落哈希复核。
//   - 所有请求严格限制大小、拒绝未知字段和尾随JSON对象。

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const (
	lessonPlanSectionRewritePreviewBodyMaxBytes int64 = 32 << 10
	lessonPlanSectionRewriteApplyBodyMaxBytes   int64 = 256 << 10
)

// GenerateSectionRewritePreview 流式生成某个教案段落的修改预览。
func (h *LessonPlanHandler) GenerateSectionRewritePreview(
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

	if h == nil ||
		h.sectionRewriteService == nil {
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"教案段落AI修改服务暂不可用",
		)
		return
	}

	planID := extractLPID(r.URL.Path)
	if planID == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims.UserID == "" {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	var req models.GenerateLessonPlanSectionRewriteRequest
	if !decodeLessonPlanSectionRewriteJSON(
		w,
		r,
		&req,
		lessonPlanSectionRewritePreviewBodyMaxBytes,
	) {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		utils.InternalError(
			w,
			"当前连接不支持流式响应",
		)
		return
	}

	finishHandshake, accepted :=
		beginSSEHandshake(w)
	if !accepted {
		return
	}

	handshakeFinished := false
	finish := func() {
		if handshakeFinished {
			return
		}
		handshakeFinished = true
		finishHandshake()
	}
	defer finish()

	w.Header().Set(
		"Content-Type",
		"text/event-stream; charset=utf-8",
	)
	w.Header().Set(
		"Cache-Control",
		"no-cache, no-store",
	)
	w.Header().Set(
		"Connection",
		"keep-alive",
	)
	w.Header().Set(
		"X-Accel-Buffering",
		"no",
	)
	w.Header().Set(
		"X-Content-Type-Options",
		"nosniff",
	)

	if err := writeLessonPlanSectionRewriteSSEEvent(
		w,
		flusher,
		"connected",
		map[string]interface{}{
			"plan_id": planID,
			"status":  "ready",
		},
	); err != nil {
		finish()
		return
	}

	// connected事件已经提交响应头，释放握手锁。
	finish()

	preview, err :=
		h.sectionRewriteService.GeneratePreview(
			r.Context(),
			planID,
			claims.UserID,
			&req,
			func(chunk string) error {
				return writeLessonPlanSectionRewriteSSEEvent(
					w,
					flusher,
					"chunk",
					map[string]string{
						"chunk": chunk,
					},
				)
			},
		)
	if err != nil {
		status, code, message :=
			lessonPlanSectionRewriteErrorInfo(
				err,
			)

		if status >= 500 {
			log.Printf(
				"[lesson-plan section rewrite] 生成预览失败 plan=%s user=%s err=%v",
				planID,
				claims.UserID,
				err,
			)
		}

		_ = writeLessonPlanSectionRewriteSSEEvent(
			w,
			flusher,
			"error",
			map[string]string{
				"code":    code,
				"message": message,
			},
		)
		return
	}

	if err := writeLessonPlanSectionRewriteSSEEvent(
		w,
		flusher,
		"done",
		map[string]interface{}{
			"preview": preview,
		},
	); err != nil {
		return
	}
}

// ApplySectionRewrite 原子应用老师确认的段落修改结果。
func (h *LessonPlanHandler) ApplySectionRewrite(
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

	if h == nil ||
		h.sectionRewriteService == nil {
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"教案段落AI修改服务暂不可用",
		)
		return
	}

	planID := extractLPID(r.URL.Path)
	if planID == "" {
		utils.BadRequest(
			w,
			utils.MsgMissingLessonPlanID,
		)
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok || claims.UserID == "" {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	var req models.ApplyLessonPlanSectionRewriteRequest
	if !decodeLessonPlanSectionRewriteJSON(
		w,
		r,
		&req,
		lessonPlanSectionRewriteApplyBodyMaxBytes,
	) {
		return
	}

	result, err :=
		h.sectionRewriteService.Apply(
			r.Context(),
			planID,
			claims.UserID,
			&req,
		)
	if err != nil {
		writeLessonPlanSectionRewriteHTTPError(
			w,
			planID,
			claims.UserID,
			err,
		)
		return
	}

	utils.Success(w, result)
}

// decodeLessonPlanSectionRewriteJSON 严格读取一份JSON对象。
func decodeLessonPlanSectionRewriteJSON(
	w http.ResponseWriter,
	r *http.Request,
	target interface{},
	maxBytes int64,
) bool {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(
			err,
			&maxBytesError,
		):
			utils.Fail(
				w,
				http.StatusRequestEntityTooLarge,
				"请求正文过大",
			)

		case errors.Is(
			err,
			io.EOF,
		):
			utils.BadRequest(
				w,
				"请求正文不能为空",
			)

		default:
			utils.BadRequest(
				w,
				"请求正文格式无效",
			)
		}
		return false
	}

	var trailing json.RawMessage
	if err := decoder.Decode(
		&trailing,
	); !errors.Is(err, io.EOF) {
		utils.BadRequest(
			w,
			"请求正文只能包含一个JSON对象",
		)
		return false
	}

	return true
}

// lessonPlanSectionRewriteErrorInfo 将业务错误映射为安全状态码和稳定公开文案。
func lessonPlanSectionRewriteErrorInfo(
	err error,
) (
	status int,
	code string,
	message string,
) {
	switch {
	case errors.Is(
		err,
		services.ErrLPSectionLocatorInvalid,
	),
		errors.Is(
			err,
			services.ErrLPSectionInstructionRequired,
		),
		errors.Is(
			err,
			services.ErrLPSectionInstructionTooLong,
		),
		errors.Is(
			err,
			services.ErrLPSectionReplacementRequired,
		),
		errors.Is(
			err,
			services.ErrLPSectionReplacementTooLong,
		):
		return http.StatusBadRequest,
			"invalid_request",
			err.Error()

	case errors.Is(
		err,
		services.ErrLPNotAuthor,
	),
		errors.Is(
			err,
			services.ErrLPCannotEdit,
		):
		return http.StatusForbidden,
			"forbidden",
			err.Error()

	case errors.Is(
		err,
		services.ErrLPNotFound,
	),
		errors.Is(
			err,
			services.ErrLPSectionNotFound,
		):
		return http.StatusNotFound,
			"not_found",
			err.Error()

	case errors.Is(
		err,
		services.ErrLPSectionVersionConflict,
	),
		errors.Is(
			err,
			services.ErrLPSectionHashConflict,
		):
		return http.StatusConflict,
			"content_conflict",
			err.Error()

	case errors.Is(
		err,
		services.ErrLPSectionInsufficientCredits,
	):
		return http.StatusPaymentRequired,
			"insufficient_credits",
			services.ErrLPSectionInsufficientCredits.Error()

	case errors.Is(
		err,
		services.ErrLPSectionAIConfigUnavailable,
	):
		return http.StatusServiceUnavailable,
			"ai_unavailable",
			"教案AI修改服务暂不可用，请稍后重试"

	case errors.Is(
		err,
		services.ErrLPSectionAIGenerationFailed,
	):
		return http.StatusBadGateway,
			"ai_generation_failed",
			"AI修改建议生成失败，请稍后重试"

	default:
		return http.StatusInternalServerError,
			"internal_error",
			"操作失败，请稍后重试"
	}
}

// writeLessonPlanSectionRewriteHTTPError 输出普通HTTP错误。
func writeLessonPlanSectionRewriteHTTPError(
	w http.ResponseWriter,
	planID string,
	userID string,
	err error,
) {
	status, _, message :=
		lessonPlanSectionRewriteErrorInfo(err)

	if status >= 500 {
		log.Printf(
			"[lesson-plan section rewrite] 应用修改失败 plan=%s user=%s err=%v",
			planID,
			userID,
			err,
		)
	}

	utils.Fail(
		w,
		status,
		message,
	)
}

// writeLessonPlanSectionRewriteSSEEvent 输出一条SSE事件。
func writeLessonPlanSectionRewriteSSEEvent(
	w http.ResponseWriter,
	flusher http.Flusher,
	eventType string,
	data interface{},
) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf(
			"序列化SSE事件失败: %w",
			err,
		)
	}

	if _, err := fmt.Fprintf(
		w,
		"event: %s\ndata: %s\n\n",
		eventType,
		string(jsonData),
	); err != nil {
		return fmt.Errorf(
			"写入SSE事件失败: %w",
			err,
		)
	}

	flusher.Flush()
	return nil
}
