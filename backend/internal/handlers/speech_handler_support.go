package handlers

// speech_handler_support.go — 语音输入请求解析、响应构建和错误映射
//
// 本文件不建立WebSocket，也不读取音频流。
// 它提供：
//   - start、stop、cancel控制消息的严格JSON解析；
//   - PCM音频块的确定性校验；
//   - 浏览器Origin校验；
//   - 豆包识别结果到统一前端事件的转换；
//   - 上游错误到稳定用户文案的收敛；
//   - WebSocket升级前连接治理错误的HTTP映射。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/services"
	"tedna/internal/utils"
)

// readSpeechStartRequest 读取并严格解析第一条start控制消息。
func readSpeechStartRequest(
	conn *websocket.Conn,
) (*models.SpeechStartRequest, error) {
	if err := conn.SetReadDeadline(
		time.Now().Add(
			speechStartMessageTimeout,
		),
	); err != nil {
		return nil, errors.New(
			"无法设置语音启动超时",
		)
	}
	defer conn.SetReadDeadline(
		time.Time{},
	)

	messageType, data, err :=
		conn.ReadMessage()
	if err != nil {
		return nil, errors.New(
			"等待语音启动消息失败",
		)
	}
	if messageType !=
		websocket.TextMessage {
		return nil, errors.New(
			"第一条消息必须是start JSON控制消息",
		)
	}

	var request models.SpeechStartRequest
	if err := decodeStrictSpeechJSON(
		data,
		&request,
	); err != nil {
		return nil, errors.New(
			"start消息格式错误",
		)
	}

	if request.Action !=
		models.SpeechActionStart {
		return nil, errors.New(
			"第一条控制动作必须为start",
		)
	}
	if request.SampleRate != 16000 ||
		request.BitsPerSample != 16 ||
		request.Channels != 1 {
		return nil, errors.New(
			"仅支持16000Hz、16bit、单声道PCM音频",
		)
	}

	return &request, nil
}

// parseSpeechControlRequest 解析stop、cancel等控制消息。
func parseSpeechControlRequest(
	data []byte,
) (*models.SpeechControlRequest, error) {
	var request models.SpeechControlRequest
	if err := decodeStrictSpeechJSON(
		data,
		&request,
	); err != nil {
		return nil, errors.New(
			"语音控制消息格式错误",
		)
	}

	request.Action = strings.TrimSpace(
		request.Action,
	)
	if request.Action == "" {
		return nil, errors.New(
			"语音控制动作不能为空",
		)
	}

	return &request, nil
}

// decodeStrictSpeechJSON 严格解析单个JSON对象并拒绝未知字段和尾随数据。
func decodeStrictSpeechJSON(
	data []byte,
	target interface{},
) error {
	decoder := json.NewDecoder(
		bytes.NewReader(data),
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		target,
	); err != nil {
		return err
	}

	var trailing interface{}
	if err := decoder.Decode(
		&trailing,
	); !errors.Is(err, io.EOF) {
		return errors.New(
			"JSON包含多余内容",
		)
	}

	return nil
}

// validateSpeechAudioChunk 校验浏览器单块PCM音频。
func validateSpeechAudioChunk(
	audio []byte,
) error {
	if len(audio) == 0 {
		return errors.New(
			"音频包不能为空",
		)
	}
	if len(audio) >
		speechBrowserAudioChunkMaxBytes {
		return errors.New(
			"单个音频包过大",
		)
	}
	if len(audio)%2 != 0 {
		return errors.New(
			"PCM音频长度必须是2字节采样点的整数倍",
		)
	}

	return nil
}

// isAllowedSpeechOrigin 严格校验浏览器Origin。
func isAllowedSpeechOrigin(
	r *http.Request,
) bool {
	if r == nil {
		return false
	}

	return strings.TrimSpace(
		r.Header.Get("Origin"),
	) == speechAllowedOrigin
}

// buildSpeechRecognitionEvent 转换统一浏览器识别事件。
func buildSpeechRecognitionEvent(
	eventType string,
	text string,
	final bool,
	result *ai.ASRRecognitionResult,
) models.SpeechRecognitionEvent {
	utterances := make(
		[]models.SpeechUtterance,
		0,
		len(result.Utterances),
	)

	for _, item := range result.Utterances {
		utterances = append(
			utterances,
			models.SpeechUtterance{
				Text:      item.Text,
				StartTime: item.StartTime,
				EndTime:   item.EndTime,
				Definite:  item.Definite,
			},
		)
	}

	return models.SpeechRecognitionEvent{
		Event:      eventType,
		Text:       text,
		Final:      final,
		Utterances: utterances,
		DurationMS: result.DurationMS,
		RequestID:  result.RequestID,
		LogID:      result.LogID,
	}
}

// writeSpeechEvent 串行向浏览器写一个JSON事件。
func writeSpeechEvent(
	conn *websocket.Conn,
	event models.SpeechRecognitionEvent,
) error {
	if conn == nil {
		return errors.New(
			"浏览器语音连接不存在",
		)
	}

	if err := conn.SetWriteDeadline(
		time.Now().Add(
			speechBrowserWriteTimeout,
		),
	); err != nil {
		return err
	}

	return conn.WriteJSON(event)
}

// writeSpeechSocketError 写error事件后再写closed事件。
func writeSpeechSocketError(
	conn *websocket.Conn,
	code string,
	message string,
	requestID string,
	logID string,
) {
	_ = writeSpeechEvent(
		conn,
		models.SpeechRecognitionEvent{
			Event:     models.SpeechEventError,
			Code:      code,
			Message:   message,
			RequestID: requestID,
			LogID:     logID,
		},
	)

	_ = writeSpeechEvent(
		conn,
		models.SpeechRecognitionEvent{
			Event:     models.SpeechEventClosed,
			RequestID: requestID,
			LogID:     logID,
			Message:   "语音连接已关闭",
		},
	)
}

// publicSpeechError 把内部和上游错误映射为稳定的浏览器文案。
func publicSpeechError(
	err error,
) (string, string) {
	if err == nil {
		return "speech_unknown_error",
			"语音识别失败，请稍后重试"
	}

	if errors.Is(
		err,
		context.DeadlineExceeded,
	) {
		return "speech_upstream_timeout",
			"语音识别服务响应超时，请重新尝试"
	}

	switch {
	case errors.Is(
		err,
		ai.ErrASREmptyAudioChunk,
	):
		return "speech_audio_empty",
			"没有收到可识别的语音"

	case errors.Is(
		err,
		ai.ErrASRAudioChunkTooLarge,
	):
		return "speech_audio_chunk_too_large",
			"单个音频包过大"

	case errors.Is(
		err,
		ai.ErrASRAudioChunkMisaligned,
	):
		return "speech_audio_format_invalid",
			"浏览器音频格式不正确"

	case errors.Is(
		err,
		ai.ErrASRFinalAlreadySent,
	):
		return "speech_stop_duplicate",
			"录音已经停止"

	case errors.Is(
		err,
		ai.ErrASRSessionClosed,
	):
		return "speech_connection_closed",
			"语音识别连接已关闭"
	}

	var upstreamError *ai.ASRUpstreamError
	if errors.As(
		err,
		&upstreamError,
	) {
		switch upstreamError.Code {
		case 45000002:
			return "speech_audio_empty",
				"没有检测到有效语音"

		case 45000081:
			return "speech_upstream_wait_timeout",
				"语音数据等待超时，请重新录音"

		case 45000151:
			return "speech_audio_format_invalid",
				"浏览器音频格式不正确"

		case 55000031:
			return "speech_upstream_busy",
				"语音识别服务繁忙，请稍后重试"
		}

		return "speech_upstream_error",
			"语音识别服务暂时不可用，请稍后重试"
	}

	return "speech_internal_error",
		"语音识别失败，请稍后重试"
}

// handleSpeechConnectionLimitError 处理升级前的连接治理错误。
func handleSpeechConnectionLimitError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrSpeechConnectionUserLimit,
	):
		utils.Fail(
			w,
			http.StatusConflict,
			"当前账号已有语音输入连接，请先结束原连接",
		)

	case errors.Is(
		err,
		services.ErrSpeechConnectionsDraining,
	):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"服务正在更新，请稍后重新开始语音输入",
		)

	case errors.Is(
		err,
		services.ErrSpeechConnectionGlobalLimit,
	):
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"当前语音使用人数较多，请稍后重试",
		)

	default:
		utils.Fail(
			w,
			http.StatusServiceUnavailable,
			"暂时无法建立语音连接",
		)
	}
}
