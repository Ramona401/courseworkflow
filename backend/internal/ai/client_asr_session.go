package ai

// client_asr_session.go — 火山豆包流式语音识别2.0连接生命周期
//
// 本文件负责TE-DNA后端到火山引擎的上游WebSocket连接：
//   1. 使用旧版控制台的APP ID + Access Token完成鉴权；
//   2. 连接豆包双向流式优化版接口；
//   3. 发送带序列号的ASR首包和PCM音频包；
//   4. 接收并解析临时、确定分句和最终识别结果；
//   5. 保存X-Tt-Logid用于排错。
//
// 当前正式音频契约：
//   - PCM signed 16-bit little-endian；
//   - 16000Hz；
//   - 单声道；
//   - 浏览器建议每约200ms发送一次，即约6400字节。
//
// 结束包要求：
//   - 火山最后一包应携带真实PCM音频；
//   - 浏览器Handler必须暂存最后一块音频，在收到stop后再以final=true发送；
//   - 禁止发送空结束包，避免触发“空音频”错误。
//
// 安全要求：
//   - Access Token只存在于内存，不写日志；
//   - 单包大小、响应大小和各类超时均受限；
//   - 任意单次识别错误只关闭当前连接，不得退出tedna进程。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"tedna/internal/logger"
	"tedna/internal/utils"
)

// ==================== 连接与音频限制 ====================

const (
	asrDefaultDialTimeout  = 10 * time.Second
	asrDefaultReadTimeout  = 30 * time.Second
	asrDefaultWriteTimeout = 10 * time.Second

	// 200ms的16kHz单声道16bit PCM约6400字节。
	// 此处允许到256KB，兼容浏览器短暂调度抖动，但拒绝超大单帧。
	asrMaxAudioChunkBytes = 256 * 1024
)

// ==================== 会话 ====================

// ASRSession 表示一条TE-DNA后端到火山的上游WebSocket连接。
//
// gorilla/websocket允许一个并发Reader和一个并发Writer。
// 本结构通过writeMu和readMu进一步防止调用方误用。
type ASRSession struct {
	conn      *websocket.Conn
	cfg       ASRConfig
	requestID string
	connectID string
	logID     string

	writeMu      sync.Mutex
	readMu       sync.Mutex
	nextSequence int32
	finalSent    bool

	closeOnce sync.Once
	closed    chan struct{}
}

// 模块日志。
var asrLog = logger.WithModule("ai_asr")

// OpenASRSession 建立火山ASR连接、发送首包并等待服务端确认。
func OpenASRSession(
	ctx context.Context,
	cfg *ASRConfig,
	options ASRRequestOptions,
) (*ASRSession, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfgCopy := *cfg
	cfgCopy.AppID = strings.TrimSpace(cfgCopy.AppID)
	cfgCopy.AccessToken = strings.TrimSpace(cfgCopy.AccessToken)
	cfgCopy.WebSocketURL = strings.TrimSpace(cfgCopy.WebSocketURL)
	cfgCopy.ResourceID = strings.TrimSpace(cfgCopy.ResourceID)
	options = options.normalized()

	requestID := uuid.NewString()
	connectID := uuid.NewString()

	headers := http.Header{}
	headers.Set("X-Api-App-Key", cfgCopy.AppID)
	headers.Set("X-Api-Access-Key", cfgCopy.AccessToken)
	headers.Set("X-Api-Resource-Id", cfgCopy.ResourceID)
	headers.Set("X-Api-Request-Id", requestID)
	headers.Set("X-Api-Connect-Id", connectID)
	headers.Set("X-Api-Sequence", "-1")

	dialer := websocket.Dialer{
		HandshakeTimeout:  asrDefaultDialTimeout,
		EnableCompression: false,
	}

	conn, response, err := dialer.DialContext(ctx, cfgCopy.WebSocketURL, headers)
	if err != nil {
		detail := readASRHandshakeError(response)
		if detail != "" {
			return nil, fmt.Errorf(
				"建立ASR上游WebSocket失败: %w (%s)",
				err,
				detail,
			)
		}

		return nil, fmt.Errorf("建立ASR上游WebSocket失败: %w", err)
	}

	conn.SetReadLimit(asrProtocolMaxPayloadBytes + 1024)

	logID := ""
	if response != nil {
		logID = strings.TrimSpace(response.Header.Get("X-Tt-Logid"))
	}

	session := &ASRSession{
		conn:         conn,
		cfg:          cfgCopy,
		requestID:    requestID,
		connectID:    connectID,
		logID:        logID,
		nextSequence: 2,
		closed:       make(chan struct{}),
	}

	firstFrame, err := buildASRFullClientFrame(
		buildASRWireFullRequest(options),
		1,
	)
	if err != nil {
		_ = session.Close()
		return nil, err
	}

	if err := session.writeBinary(ctx, firstFrame); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("发送ASR首包失败: %w", err)
	}

	// 在向浏览器宣布ready之前先等待首包确认，
	// 使凭据、Resource ID或协议错误尽早暴露。
	if _, err := session.ReadResult(ctx); err != nil {
		_ = session.Close()
		return nil, fmt.Errorf("ASR首包确认失败: %w", err)
	}

	asrLog.Info(
		"ASR上游连接已建立",
		"request_id", requestID,
		"connect_id", connectID,
		"log_id", logID,
		"resource_id", cfgCopy.ResourceID,
	)

	// 调用上下文取消时主动关闭上游连接，
	// 避免浏览器断开后仍持续占用小时版服务。
	go func() {
		select {
		case <-ctx.Done():
			_ = session.Close()
		case <-session.closed:
		}
	}()

	return session, nil
}

// RequestID 返回TE-DNA生成的单次请求ID。
func (session *ASRSession) RequestID() string {
	if session == nil {
		return ""
	}
	return session.requestID
}

// LogID 返回火山握手响应中的X-Tt-Logid。
func (session *ASRSession) LogID() string {
	if session == nil {
		return ""
	}
	return session.logID
}

// MaxDurationSeconds 返回当前配置允许的最长录音时长。
func (session *ASRSession) MaxDurationSeconds() int {
	if session == nil {
		return 0
	}
	return session.cfg.MaxDurationSeconds
}

// SendAudioChunk 发送一个真实PCM音频包。
//
// final=false：普通音频包；
// final=true：最后一个真实音频包，协议层会写入负序列号。
func (session *ASRSession) SendAudioChunk(
	ctx context.Context,
	pcm []byte,
	final bool,
) error {
	if session == nil || session.isClosed() {
		return ErrASRSessionClosed
	}

	if len(pcm) == 0 {
		return ErrASREmptyAudioChunk
	}

	if len(pcm) > asrMaxAudioChunkBytes {
		return ErrASRAudioChunkTooLarge
	}

	// 16bit PCM每个采样点占2字节。
	if len(pcm)%2 != 0 {
		return ErrASRAudioChunkMisaligned
	}

	session.writeMu.Lock()
	defer session.writeMu.Unlock()

	if session.isClosed() {
		return ErrASRSessionClosed
	}
	if session.finalSent {
		return ErrASRFinalAlreadySent
	}

	sequence := session.nextSequence
	frame, err := buildASRAudioFrame(pcm, sequence, final)
	if err != nil {
		return err
	}

	if err := session.writeBinaryLocked(ctx, frame); err != nil {
		return fmt.Errorf("发送ASR音频包失败: %w", err)
	}

	session.nextSequence++
	if final {
		session.finalSent = true
	}

	return nil
}

// ReadResult 读取并解析一条火山响应。
//
// 首包确认响应可能没有文字，调用方应允许Text为空并继续读取。
func (session *ASRSession) ReadResult(
	ctx context.Context,
) (*ASRRecognitionResult, error) {
	if session == nil || session.isClosed() {
		return nil, ErrASRSessionClosed
	}

	session.readMu.Lock()
	defer session.readMu.Unlock()

	deadline := time.Now().Add(asrDefaultReadTimeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}

	if err := session.conn.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("设置ASR读取超时失败: %w", err)
	}

	messageType, data, err := session.conn.ReadMessage()
	if err != nil {
		if session.isClosed() {
			return nil, ErrASRSessionClosed
		}
		return nil, fmt.Errorf("读取ASR上游响应失败: %w", err)
	}

	if messageType != websocket.BinaryMessage {
		return nil, fmt.Errorf(
			"ASR上游返回了非二进制WebSocket消息: %d",
			messageType,
		)
	}

	protocolMessage, err := parseASRServerFrame(data)
	if err != nil {
		return nil, err
	}

	if protocolMessage.MessageType == asrMessageTypeServerError {
		code := uint32(0)
		if protocolMessage.ErrorCode != nil {
			code = *protocolMessage.ErrorCode
		}

		return nil, &ASRUpstreamError{
			Code:    code,
			Message: extractASRErrorMessage(protocolMessage.Payload),
			LogID:   session.logID,
		}
	}

	var envelope asrWireServerEnvelope
	if len(protocolMessage.Payload) > 0 {
		if err := json.Unmarshal(protocolMessage.Payload, &envelope); err != nil {
			return nil, fmt.Errorf("解析ASR识别结果JSON失败: %w", err)
		}
	}

	// 火山正常成功码可能为空，也可能为20000000。
	if envelope.Code != 0 && envelope.Code != 20000000 {
		return nil, &ASRUpstreamError{
			Code: uint32(envelope.Code),
			Message: firstNonEmptyASRString(
				envelope.Message,
				envelope.Error,
				"ASR服务返回业务错误",
			),
			LogID: session.logID,
		}
	}

	if strings.TrimSpace(envelope.Error) != "" {
		return nil, &ASRUpstreamError{
			Code:    0,
			Message: strings.TrimSpace(envelope.Error),
			LogID:   session.logID,
		}
	}

	utterances := make([]ASRUtterance, 0, len(envelope.Result.Utterances))
	for _, item := range envelope.Result.Utterances {
		utterances = append(utterances, ASRUtterance{
			Text:      item.Text,
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Definite:  item.Definite,
		})
	}

	return &ASRRecognitionResult{
		Text:       envelope.Result.Text,
		Utterances: utterances,
		DurationMS: envelope.AudioInfo.Duration,
		Final:      protocolMessage.IsLast,
		Sequence:   protocolMessage.Sequence,
		RequestID:  session.requestID,
		LogID:      session.logID,
	}, nil
}

// writeBinary 串行发送WebSocket二进制消息。
func (session *ASRSession) writeBinary(ctx context.Context, data []byte) error {
	session.writeMu.Lock()
	defer session.writeMu.Unlock()

	return session.writeBinaryLocked(ctx, data)
}

// writeBinaryLocked 在调用方已经持有writeMu时发送消息。
func (session *ASRSession) writeBinaryLocked(ctx context.Context, data []byte) error {
	if session.isClosed() {
		return ErrASRSessionClosed
	}

	deadline := time.Now().Add(asrDefaultWriteTimeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}

	if err := session.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}

	return session.conn.WriteMessage(websocket.BinaryMessage, data)
}

// Close 关闭当前上游连接。
//
// Close是幂等的，可由浏览器断开、超时、cancel或正常结束重复调用。
func (session *ASRSession) Close() error {
	if session == nil {
		return nil
	}

	var closeErr error

	session.closeOnce.Do(func() {
		close(session.closed)

		session.writeMu.Lock()
		defer session.writeMu.Unlock()

		deadline := time.Now().Add(2 * time.Second)
		_ = session.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			deadline,
		)

		closeErr = session.conn.Close()

		asrLog.Info(
			"ASR上游连接已关闭",
			"request_id", session.requestID,
			"log_id", session.logID,
		)
	})

	return closeErr
}

// isClosed 判断连接是否已关闭。
func (session *ASRSession) isClosed() bool {
	if session == nil {
		return true
	}

	select {
	case <-session.closed:
		return true
	default:
		return false
	}
}

// ==================== 辅助函数 ====================

// readASRHandshakeError 安全读取WebSocket握手失败响应。
func readASRHandshakeError(response *http.Response) string {
	if response == nil || response.Body == nil {
		return ""
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 4096))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(body))
}

// extractASRErrorMessage 从错误帧Payload提取人类可读说明。
func extractASRErrorMessage(payload []byte) string {
	if len(payload) == 0 {
		return "ASR服务返回空错误信息"
	}

	var object struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(payload, &object); err == nil {
		if value := firstNonEmptyASRString(object.Message, object.Error); value != "" {
			return value
		}
	}

	return utils.SafeTruncate(strings.TrimSpace(string(payload)), 500)
}

// firstNonEmptyASRString 返回第一个非空字符串。
func firstNonEmptyASRString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
