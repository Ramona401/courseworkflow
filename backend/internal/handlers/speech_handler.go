package handlers

// speech_handler.go — 全平台语音输入WebSocket入口
//
// 浏览器连接协议：
//   GET /api/v1/speech/stream?token=<JWT>
//
// 建连后：
//   1. 浏览器先发送start JSON控制消息；
//   2. 后端验证固定音频格式；
//   3. 在连接ASR供应商之前按最大录音时长预留积分；
//   4. 后端连接豆包ASR并返回ready事件；
//   5. 浏览器连续发送16kHz、单声道、16bit PCM二进制消息；
//   6. 浏览器发送stop JSON控制消息；
//   7. 后端把最后一块真实PCM作为火山最后一包发送；
//   8. 最终识别成功后按实际PCM秒数原子结算积分。
//
// 权限和隐私：
//   - 浏览器WebSocket不能可靠设置Authorization请求头，因此沿用现有SSE的query token；
//   - JWT必须在WebSocket升级前验证；
//   - Origin必须严格等于生产站点；
//   - APP ID与Access Token永远只存在于后端；
//   - 音频与完整识别正文不落库、不写日志；
//   - 识别结果只返回输入框，绝不自动发送给AI。
//
// 计费安全：
//   - 单价未启用时在供应商连接前拒绝，不产生外部成本；
//   - 失败、取消和超时释放冻结积分；
//   - 最后一包真实音频发送后，即使浏览器断开，也继续等待供应商最终结果并完成结算；
//   - 供应商最终成功但结算数据库异常时保留reserved状态，禁止误释放已发生的成本；
//   - ASR计费细节集中在speech_handler_billing.go。

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// ==================== 连接与音频限制 ====================

const (
	// speechAllowedOrigin 是生产站点唯一允许的浏览器Origin。
	speechAllowedOrigin = "https://workflow.pkuailab.com"

	// speechStartMessageTimeout 是升级后等待start控制消息的最长时间。
	speechStartMessageTimeout = 15 * time.Second

	// speechFinishGracePeriod 是录音上限之外等待最终结果的额外时间。
	speechFinishGracePeriod = 45 * time.Second

	// speechDetachedSessionBuffer 是独立会话上下文相对桥接计时器的额外硬截止缓冲。
	speechDetachedSessionBuffer = 30 * time.Second

	// speechBrowserWriteTimeout 是向浏览器写单个事件的最长时间。
	speechBrowserWriteTimeout = 10 * time.Second

	// speechBrowserMessageMaxBytes 允许的单条浏览器消息上限。
	speechBrowserMessageMaxBytes = 256*1024 + 4096

	// speechPCMBytesPerSecond 是16kHz、16bit、单声道PCM每秒字节数。
	speechPCMBytesPerSecond = 16000 * 2

	// speechBrowserAudioChunkMaxBytes 与上游客户端单包防线保持一致。
	speechBrowserAudioChunkMaxBytes = 256 * 1024
)

// ==================== Handler ====================

// SpeechHandler 管理浏览器语音WebSocket入口。
type SpeechHandler struct {
	cfg                 *config.Config
	authService         *services.AuthService
	registry            *services.SpeechConnectionRegistry
	mediaBillingService *services.MediaBillingService
	upgrader            websocket.Upgrader
}

// 模块日志。
var speechHandlerLog = logger.WithModule("speech_handler")

// NewSpeechHandler 创建语音输入处理器。
func NewSpeechHandler(
	cfg *config.Config,
	authService *services.AuthService,
	registry *services.SpeechConnectionRegistry,
) *SpeechHandler {
	return &SpeechHandler{
		cfg:                 cfg,
		authService:         authService,
		registry:            registry,
		mediaBillingService: services.NewMediaBillingService(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:    32 * 1024,
			WriteBufferSize:   16 * 1024,
			EnableCompression: false,
			CheckOrigin: func(r *http.Request) bool {
				return isAllowedSpeechOrigin(r)
			},
		},
	}
}

// Stream GET /api/v1/speech/stream?token=xxx。
//
// 本入口不能套普通AuthMiddleware，因为浏览器WebSocket构造器无法设置Bearer请求头。
// 鉴权、Origin、ASR配置、积分预留和连接配额全部在本入口完成。
func (handler *SpeechHandler) Stream(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.Fail(w, http.StatusMethodNotAllowed, utils.MsgMethodGetOnly)
		return
	}

	if handler == nil ||
		handler.cfg == nil ||
		handler.authService == nil ||
		handler.registry == nil ||
		handler.mediaBillingService == nil {
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"语音识别服务未就绪",
		)
		return
	}

	if !isAllowedSpeechOrigin(r) {
		speechHandlerLog.Warn(
			"语音WebSocket Origin被拒绝",
			"origin", r.Header.Get("Origin"),
			"remote_addr", r.RemoteAddr,
		)
		utils.Forbidden(w, "语音连接来源不受信任")
		return
	}

	token := strings.TrimSpace(
		r.URL.Query().Get("token"),
	)
	if token == "" {
		utils.Unauthorized(w, "缺少token参数")
		return
	}

	claims, err := handler.authService.ValidateToken(token)
	if err != nil {
		if errors.Is(err, services.ErrTokenExpired) {
			utils.Unauthorized(w, "认证令牌已过期，请重新登录")
			return
		}
		utils.Unauthorized(w, "认证令牌无效")
		return
	}

	asrConfig, err := ai.GetASRConfig(
		handler.cfg.GetAESKey(),
	)
	if err != nil {
		speechHandlerLog.Error(
			"加载ASR配置失败",
			"user_id", claims.UserID,
			"error", err,
		)
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"语音识别服务尚未配置或暂不可用",
		)
		return
	}

	lease, err := handler.registry.Acquire(
		claims.UserID,
	)
	if err != nil {
		handleSpeechConnectionLimitError(
			w,
			err,
		)
		return
	}
	defer lease.Release()

	browserConn, err := handler.upgrader.Upgrade(
		w,
		r,
		nil,
	)
	if err != nil {
		speechHandlerLog.Warn(
			"升级语音WebSocket失败",
			"user_id", claims.UserID,
			"origin", r.Header.Get("Origin"),
			"remote_addr", r.RemoteAddr,
			"error", err,
		)
		return
	}
	defer browserConn.Close()

	// WebSocket升级后使用独立且有硬截止的会话上下文：
	//   - 浏览器断开由browserReader显式处理；
	//   - 最后一包音频发送后，浏览器即使断开也不能取消上游最终结果和积分结算；
	//   - 服务排空仍通过lease.SetCloser主动取消并关闭连接；
	//   - 硬截止防止网络异常使分离后的会话无限存活。
	sessionTimeout :=
		time.Duration(
			asrConfig.MaxDurationSeconds,
		)*time.Second +
			speechFinishGracePeriod +
			speechDetachedSessionBuffer

	sessionContext, cancelSession :=
		context.WithTimeout(
			context.Background(),
			sessionTimeout,
		)
	defer cancelSession()

	// Hijacked WebSocket不会由http.Server.Shutdown主动关闭。
	// 注册表进入排空时可通过本关闭函数取消上下文并关闭浏览器连接。
	lease.SetCloser(func() {
		cancelSession()
		_ = browserConn.Close()
	})

	browserConn.SetReadLimit(
		speechBrowserMessageMaxBytes,
	)

	startRequest, err :=
		readSpeechStartRequest(
			browserConn,
		)
	if err != nil {
		writeSpeechSocketError(
			browserConn,
			"speech_start_invalid",
			err.Error(),
			"",
			"",
		)
		return
	}

	// 必须先预留积分，再连接供应商。
	billingSession, err :=
		handler.reserveSpeechASRBilling(
			sessionContext,
			claims.UserID,
			asrConfig,
		)
	if err != nil {
		code, message :=
			publicSpeechBillingError(err)

		speechHandlerLog.Warn(
			"ASR积分预留失败",
			"user_id", claims.UserID,
			"resource_id", asrConfig.ResourceID,
			"error", err,
		)

		writeSpeechSocketError(
			browserConn,
			code,
			message,
			"",
			"",
		)
		return
	}

	bridgeOwnsBilling := false
	billingReleaseStage := "open_upstream"

	// runSpeechBridge开始前的任意失败都必须释放预留。
	defer func() {
		if bridgeOwnsBilling {
			return
		}

		billingSession.releasePending(
			models.MediaBillingStatusFailed,
			"speech_"+billingReleaseStage+"_failed",
			map[string]interface{}{
				"stage": billingReleaseStage,
			},
		)
	}()

	upstreamSession, err := ai.OpenASRSession(
		sessionContext,
		asrConfig,
		ai.DefaultASRRequestOptions(""),
	)
	if err != nil {
		code, message :=
			publicSpeechError(err)

		speechHandlerLog.Error(
			"建立ASR上游连接失败",
			"user_id", claims.UserID,
			"error", err,
		)

		writeSpeechSocketError(
			browserConn,
			code,
			message,
			"",
			"",
		)
		return
	}
	defer upstreamSession.Close()

	billingReleaseStage = "bind_external_task"

	if err := billingSession.bindExternalTask(
		upstreamSession.RequestID(),
		map[string]interface{}{
			"provider_request_id": upstreamSession.RequestID(),
			"provider_log_id":     upstreamSession.LogID(),
		},
	); err != nil {
		speechHandlerLog.Error(
			"绑定ASR供应商请求ID失败",
			"user_id", claims.UserID,
			"request_id", upstreamSession.RequestID(),
			"error", err,
		)

		writeSpeechSocketError(
			browserConn,
			"speech_billing_bind_failed",
			"语音识别积分任务初始化失败，请稍后重试",
			upstreamSession.RequestID(),
			upstreamSession.LogID(),
		)
		return
	}

	billingReleaseStage = "write_ready"

	if err := writeSpeechEvent(
		browserConn,
		models.SpeechRecognitionEvent{
			Event:     models.SpeechEventReady,
			RequestID: upstreamSession.RequestID(),
			LogID:     upstreamSession.LogID(),
			Message:   "语音识别已就绪",
		},
	); err != nil {
		return
	}

	speechHandlerLog.Info(
		"语音识别会话已就绪",
		"user_id", claims.UserID,
		"request_id", upstreamSession.RequestID(),
		"log_id", upstreamSession.LogID(),
		"sample_rate", startRequest.SampleRate,
		"bits_per_sample", startRequest.BitsPerSample,
		"channels", startRequest.Channels,
		"max_duration_seconds", upstreamSession.MaxDurationSeconds(),
		"billing_idempotency_key", billingSession.idempotencyKey,
	)

	bridgeOwnsBilling = true

	handler.runSpeechBridge(
		sessionContext,
		browserConn,
		upstreamSession,
		claims.UserID,
		billingSession,
	)
}
