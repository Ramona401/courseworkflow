package handlers

// assistant_runtime_handler.go
//
// 教学智能体公开运行HTTP处理器。本文件只负责编排请求方法、严格正文解析、
// 短时Bearer令牌、官方embed请求上下文、SSE握手和安全响应；部署状态、
// 外部父页面Origin、额度、主轮次、不可变快照和匿名计费全部由Service与
// Repository重新校验。
//
// 本单元只生成Handler，不注册路由。

import (
	"net/http"
	"strings"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// AssistantRuntimeHandler 是公开embed、会话和聊天处理器。
type AssistantRuntimeHandler struct {
	sessionService *services.AssistantRuntimeSessionService
	chatService    *services.AssistantRuntimeChatService
}

// 模块日志不记录Authorization、学生完整消息、助手完整回复或父页面完整URL。
var assistantRuntimeHandlerLog = logger.WithModule("assistant_runtime_handler")

// NewAssistantRuntimeHandler 创建公开运行处理器。
func NewAssistantRuntimeHandler(
	sessionService *services.AssistantRuntimeSessionService,
	chatService *services.AssistantRuntimeChatService,
) *AssistantRuntimeHandler {
	return &AssistantRuntimeHandler{
		sessionService: sessionService,
		chatService:    chatService,
	}
}

// Embed GET /embed/assistant/{public_id}
//
// 返回最小安全HTML壳和公开展示信息。
// 学生端固定模块由/assets/assistant-embed.js加载。
func (h *AssistantRuntimeHandler) Embed(w http.ResponseWriter, r *http.Request, publicID string) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	if h == nil || h.sessionService == nil {
		utils.Fail(w, http.StatusServiceUnavailable, "教学智能体服务未就绪")
		return
	}

	descriptor, err := h.sessionService.GetPublicDescriptor(r.Context(), strings.TrimSpace(publicID))
	if err != nil {
		writeAssistantRuntimeHTTPError(w, err)
		return
	}

	writeAssistantRuntimeEmbedHTML(w, descriptor)
}

// StartSession POST /api/v1/assistant-runtime/deployments/{public_id}/session
//
// 三方来源绑定：
//  1. HTTP Origin必须是当前TE-DNA运行站点；
//  2. Referer必须是当前public_id对应的官方embed页面；
//  3. 请求正文parent_origin必须精确命中部署allowed_origins。
//
// ParentOrigin只能由学生端从document.referrer解析；后端仍把它视为不可信字段，
// 必须在Service中规范化并与实时部署策略复核。
// AnonymousClientID是浏览器随机标识，Service会做用途隔离HMAC后才入库。
func (h *AssistantRuntimeHandler) StartSession(w http.ResponseWriter, r *http.Request, publicID string) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	if h == nil || h.sessionService == nil {
		utils.Fail(w, http.StatusServiceUnavailable, "教学智能体服务未就绪")
		return
	}

	publicID = strings.TrimSpace(publicID)

	var request models.AssistantRuntimeStartRequest
	if err := decodeAssistantRuntimeJSON(w, r, &request, assistantRuntimeStartBodyMaxBytes); err != nil {
		writeAssistantRuntimeDecodeError(w, err)
		return
	}

	if err := validateAssistantRuntimeStartRequestContext(r, publicID); err != nil {
		utils.Fail(w, http.StatusForbidden, "当前页面来源未获授权")
		return
	}

	clientIP, err := assistantRuntimeClientIP(r)
	if err != nil {
		utils.BadRequest(w, "无法识别客户端网络地址")
		return
	}

	response, err := h.sessionService.StartExternalSession(
		r.Context(),
		publicID,
		request.ParentOrigin,
		request.AnonymousClientID,
		clientIP,
	)
	if err != nil {
		writeAssistantRuntimeHTTPError(w, err)
		return
	}

	utils.Success(w, response)
}

// GetSession GET /api/v1/assistant-runtime/sessions/{session_id}
//
// completed和expired会话仍可读取正式历史；撤销、版本变化和伪造令牌拒绝。
func (h *AssistantRuntimeHandler) GetSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	if h == nil || h.sessionService == nil {
		utils.Fail(w, http.StatusServiceUnavailable, "教学智能体服务未就绪")
		return
	}

	tokenString, err := extractAssistantRuntimeBearerToken(r)
	if err != nil {
		utils.Unauthorized(w, "运行令牌无效")
		return
	}

	view, err := h.sessionService.GetRuntimeSessionView(
		r.Context(),
		tokenString,
		strings.TrimSpace(sessionID),
	)
	if err != nil {
		writeAssistantRuntimeHTTPError(w, err)
		return
	}

	utils.Success(w, view)
}

// Chat POST /api/v1/assistant-runtime/sessions/{session_id}/chat
//
// 响应协议：
//
//	event: connected -> {"phase":"ready"}
//	event: chunk     -> {"chunk":"学生可见增量"}
//	event: done      -> AssistantRuntimeChatResponse
//	event: error     -> {"error":"稳定公开错误文案"}
//
// SSE响应头只会在令牌、额度、账户、不可变快照和模型配置全部通过后写入。
func (h *AssistantRuntimeHandler) Chat(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodPost {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodPostOnly)
		return
	}

	if h == nil || h.chatService == nil {
		utils.Fail(w, http.StatusServiceUnavailable, "教学智能体服务未就绪")
		return
	}

	tokenString, err := extractAssistantRuntimeBearerToken(r)
	if err != nil {
		utils.Unauthorized(w, "运行令牌无效")
		return
	}

	var request models.AssistantRuntimeChatRequest
	if err := decodeAssistantRuntimeJSON(w, r, &request, assistantRuntimeChatBodyMaxBytes); err != nil {
		writeAssistantRuntimeDecodeError(w, err)
		return
	}

	if _, ok := w.(http.Flusher); !ok {
		utils.InternalError(w, "当前网关不支持流式响应")
		return
	}

	finishHandshake, accepted := beginSSEHandshake(w)
	if !accepted {
		return
	}

	streamStarted := false
	defer func() {
		if finishHandshake != nil {
			finishHandshake()
		}
	}()

	response, chatErr := h.chatService.ChatWithReady(
		r.Context(),
		tokenString,
		strings.TrimSpace(sessionID),
		request.Message,
		func() error {
			prepareAssistantRuntimeSSE(w)
			streamStarted = true

			if err := writeAssistantRuntimeSSEEvent(
				w,
				"connected",
				map[string]string{"phase": "ready"},
			); err != nil {
				return err
			}

			if finishHandshake != nil {
				finishHandshake()
				finishHandshake = nil
			}

			return nil
		},
		func(chunk string) error {
			return writeAssistantRuntimeSSEEvent(
				w,
				"chunk",
				map[string]string{"chunk": chunk},
			)
		},
	)

	if chatErr != nil {
		if !streamStarted {
			writeAssistantRuntimeHTTPError(w, chatErr)
			return
		}

		_ = writeAssistantRuntimeSSEEvent(
			w,
			"error",
			map[string]string{
				"error": assistantRuntimePublicErrorMessage(chatErr),
			},
		)

		assistantRuntimeHandlerLog.Warn(
			"教学智能体流式聊天失败",
			"session_id", strings.TrimSpace(sessionID),
			"error", chatErr,
		)
		return
	}

	if response == nil {
		_ = writeAssistantRuntimeSSEEvent(
			w,
			"error",
			map[string]string{
				"error": "教学智能体回复未完成，请重新尝试",
			},
		)
		return
	}

	if err := writeAssistantRuntimeSSEEvent(w, "done", response); err != nil {
		assistantRuntimeHandlerLog.Warn(
			"教学智能体完成事件写入失败",
			"session_id", strings.TrimSpace(sessionID),
			"error", err,
		)
	}
}
