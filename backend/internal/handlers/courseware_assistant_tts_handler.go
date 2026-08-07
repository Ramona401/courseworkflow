package handlers

// courseware_assistant_tts_handler.go
//
// 老师端课件教学智能体回答豆包朗读HTTP入口：
//
//   POST /api/v1/assistant-deployments/{deployment_id}/tts
//
// 请求正文：
//   {
//     "text": "教学智能体完整回答",
//     "operation_id": "浏览器生成的UUID"
//   }
//
// 安全边界：
//   - 路由必须经过教师JWT认证；
//   - 教师身份只取claims.UserID；
//   - 请求正文不能提交owner_user_id、school_id、courseware_id、page_id、音色或付费账户；
//   - Service按deployment_id和claims.UserID重新校验部署所有者；
//   - 音色由服务端按文本语言自动选择：中文vivi 2.0，英文Tim；
//   - 成功直接返回audio/mpeg，不返回服务器私有文件路径；
//   - 失败响应沿用统一JSON信封，不泄露供应商密钥、内部路径、计费价格或数据库错误。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"tedna/internal/middleware"
	"tedna/internal/repository"
	"tedna/internal/services"
	"tedna/internal/utils"
)

const coursewareAssistantTTSRequestMaxBytes = 64 * 1024

// CoursewareAssistantTTSHandler 是老师端教学智能体豆包朗读处理器。
type CoursewareAssistantTTSHandler struct {
	service *services.CoursewareAssistantTTSService
}

// NewCoursewareAssistantTTSHandler 创建老师端豆包朗读处理器。
func NewCoursewareAssistantTTSHandler(
	service *services.CoursewareAssistantTTSService,
) *CoursewareAssistantTTSHandler {
	return &CoursewareAssistantTTSHandler{
		service: service,
	}
}

// Synthesize POST /api/v1/assistant-deployments/{deployment_id}/tts。
func (h *CoursewareAssistantTTSHandler) Synthesize(
	w http.ResponseWriter,
	r *http.Request,
	deploymentID string,
) {
	if r.Method != http.MethodPost {
		utils.Fail(
			w,
			http.StatusMethodNotAllowed,
			utils.MsgMethodPostOnly,
		)
		return
	}

	claims, ok := middleware.GetClaims(
		r.Context(),
	)
	if !ok ||
		claims == nil ||
		strings.TrimSpace(claims.UserID) == "" {
		utils.Unauthorized(
			w,
			utils.MsgNotLoggedIn,
		)
		return
	}

	deploymentID = strings.TrimSpace(
		deploymentID,
	)
	if deploymentID == "" {
		utils.BadRequest(
			w,
			"教学智能体部署ID无效",
		)
		return
	}

	if h == nil ||
		h.service == nil {
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"教学智能体朗读服务未就绪",
		)
		return
	}

	var request struct {
		Text        string `json:"text"`
		OperationID string `json:"operation_id"`
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		coursewareAssistantTTSRequestMaxBytes,
	)

	decoder := json.NewDecoder(
		r.Body,
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		&request,
	); err != nil {
		utils.BadRequest(
			w,
			"朗读请求体无效",
		)
		return
	}

	request.Text = strings.TrimSpace(
		request.Text,
	)
	request.OperationID = strings.TrimSpace(
		request.OperationID,
	)

	if request.Text == "" {
		utils.BadRequest(
			w,
			"没有可朗读的教学智能体回答",
		)
		return
	}

	if _, err := uuid.Parse(
		request.OperationID,
	); err != nil {
		utils.BadRequest(
			w,
			"朗读任务标识无效，请重新发起",
		)
		return
	}

	result, err := h.service.Synthesize(
		r.Context(),
		deploymentID,
		claims.UserID,
		request.Text,
		request.OperationID,
	)
	if err != nil {
		writeCoursewareAssistantTTSError(
			w,
			err,
		)
		return
	}

	if result == nil ||
		len(result.Audio) < 100 {
		utils.InternalError(
			w,
			"教学智能体朗读音频未正确形成",
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"audio/mpeg",
	)
	w.Header().Set(
		"Content-Length",
		strconv.Itoa(
			len(result.Audio),
		),
	)
	w.Header().Set(
		"Content-Disposition",
		`inline; filename="assistant-reply.mp3"`,
	)
	w.Header().Set(
		"Cache-Control",
		"no-store, no-cache, must-revalidate",
	)
	w.Header().Set(
		"X-TEDNA-TTS-Voice",
		result.Voice,
	)
	w.Header().Set(
		"X-TEDNA-TTS-Language",
		result.Language,
	)
	w.Header().Set(
		"X-TEDNA-TTS-Cache",
		strconv.FormatBool(
			result.CacheHit,
		),
	)

	w.WriteHeader(
		http.StatusOK,
	)
	_, _ = w.Write(
		result.Audio,
	)
}

// writeCoursewareAssistantTTSError 返回稳定且不泄露内部数据的朗读错误。
func writeCoursewareAssistantTTSError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrCoursewareAssistantTTSInvalidRequest,
	):
		utils.BadRequest(
			w,
			"教学智能体回答为空、过长或朗读任务标识无效",
		)

	case errors.Is(
		err,
		repository.ErrAssistantDeploymentNotFound,
	):
		utils.Fail(
			w,
			http.StatusNotFound,
			"教学智能体部署不存在或当前教师无权使用",
		)

	case errors.Is(
		err,
		services.ErrAssistantRuntimeDeploymentUnavailable,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"教学智能体部署当前不可朗读，请确认部署处于启用状态",
		)

	case errors.Is(
		err,
		services.ErrMediaBillingPriceNotConfigured,
	):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"教学智能体豆包朗读积分计费尚未配置，请联系管理员",
		)

	case errors.Is(
		err,
		repository.ErrInsufficientBalance,
	):
		utils.Fail(
			w,
			http.StatusPaymentRequired,
			"积分余额不足，暂时无法生成豆包朗读",
		)

	case errors.Is(
		err,
		repository.ErrTokenAccountNotFound,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"尚未开通个人积分账户，暂时无法生成豆包朗读",
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
		services.ErrCoursewareAssistantTTSIdentityMismatch,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"朗读文本或教学智能体部署已经变化，请重新发起朗读",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantTTSInProgress,
	),
		errors.Is(
			err,
			services.ErrCoursewareAssistantTTSPending,
		):
		utils.Fail(
			w,
			http.StatusConflict,
			"同一朗读任务正在处理或等待恢复，请稍后重试",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantTTSTerminal,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"该朗读任务已经结束，请重新发起朗读",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantTTSOutputMissing,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"朗读已经结算，但短时音频缓存已过期，请重新朗读",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantTTSUnavailable,
	):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"豆包朗读服务暂不可用，请稍后重试",
		)

	case errors.Is(
		err,
		services.ErrCoursewareAssistantTTSSynthesisFailed,
	):
		utils.Fail(
			w,
			http.StatusBadGateway,
			"豆包朗读合成失败，请稍后重试",
		)

	default:
		utils.InternalError(
			w,
			"教学智能体朗读请求处理失败",
		)
	}
}
