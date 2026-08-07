package ai

// client_asr.go — 火山豆包流式语音识别2.0公共数据协议
//
// 本文件集中定义：
//   1. 单次识别的固定请求选项；
//   2. 火山首包JSON结构；
//   3. 识别结果与上游错误类型；
//   4. PCM音频包的公共业务错误。
//
// WebSocket连接生命周期位于client_asr_session.go；
// 配置加载位于client_asr_config.go；
// 私有二进制协议位于client_asr_protocol.go。

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ==================== 公共错误 ====================

var (
	ErrASRSessionClosed = errors.New("ASR连接已关闭")

	ErrASREmptyAudioChunk = errors.New("ASR音频包为空")

	ErrASRAudioChunkTooLarge = errors.New("ASR音频包超过大小限制")

	ErrASRAudioChunkMisaligned = errors.New("ASR PCM音频长度不是16bit采样点的整数倍")

	ErrASRFinalAlreadySent = errors.New("ASR最后一包已经发送")
)

// ==================== 请求参数 ====================

// ASRRequestOptions 单次上游识别请求参数。
//
// 正式浏览器Handler应使用DefaultASRRequestOptions构造，
// 不接受前端直接指定这些模型参数。
type ASRRequestOptions struct {
	UID                  string
	EnableNonstream      bool
	EnableITN            bool
	EnablePunctuation    bool
	EnableDDC            bool
	ShowUtterances       bool
	ResultType           string
	EndWindowSizeMS      int
	ForceToSpeechTimeMS  int
	EnableAccelerateText bool
	AccelerateScore      int
}

// DefaultASRRequestOptions 返回平台统一的识别参数。
//
// enable_nonstream=true：
//
//	实时返回逐字结果，并在VAD判停后执行二遍识别提高准确率。
func DefaultASRRequestOptions(uid string) ASRRequestOptions {
	return ASRRequestOptions{
		UID:                  strings.TrimSpace(uid),
		EnableNonstream:      true,
		EnableITN:            true,
		EnablePunctuation:    true,
		EnableDDC:            false,
		ShowUtterances:       true,
		ResultType:           "full",
		EndWindowSizeMS:      800,
		ForceToSpeechTimeMS:  1000,
		EnableAccelerateText: false,
		AccelerateScore:      0,
	}
}

// normalized 对非权限类参数做安全归一化。
func (options ASRRequestOptions) normalized() ASRRequestOptions {
	options.UID = strings.TrimSpace(options.UID)
	if options.UID == "" {
		options.UID = uuid.NewString()
	}

	switch options.ResultType {
	case "full", "single":
	default:
		options.ResultType = "full"
	}

	if options.EndWindowSizeMS < 200 || options.EndWindowSizeMS > 5000 {
		options.EndWindowSizeMS = 800
	}

	if options.ForceToSpeechTimeMS < 0 || options.ForceToSpeechTimeMS > 60000 {
		options.ForceToSpeechTimeMS = 1000
	}

	if options.AccelerateScore < 0 || options.AccelerateScore > 20 {
		options.AccelerateScore = 0
	}

	if !options.EnableAccelerateText {
		options.AccelerateScore = 0
	}

	return options
}

// ==================== 火山首包JSON结构 ====================

type asrWireFullRequest struct {
	User    asrWireUser    `json:"user"`
	Audio   asrWireAudio   `json:"audio"`
	Request asrWireRequest `json:"request"`
}

type asrWireUser struct {
	UID      string `json:"uid"`
	Platform string `json:"platform"`
}

type asrWireAudio struct {
	Format  string `json:"format"`
	Codec   string `json:"codec"`
	Rate    int    `json:"rate"`
	Bits    int    `json:"bits"`
	Channel int    `json:"channel"`
}

type asrWireRequest struct {
	ModelName            string `json:"model_name"`
	EnableNonstream      bool   `json:"enable_nonstream"`
	EnableITN            bool   `json:"enable_itn"`
	EnablePunctuation    bool   `json:"enable_punc"`
	EnableDDC            bool   `json:"enable_ddc"`
	ShowUtterances       bool   `json:"show_utterances"`
	ResultType           string `json:"result_type"`
	EndWindowSizeMS      int    `json:"end_window_size"`
	ForceToSpeechTimeMS  int    `json:"force_to_speech_time,omitempty"`
	EnableAccelerateText bool   `json:"enable_accelerate_text"`
	AccelerateScore      int    `json:"accelerate_score,omitempty"`
}

// buildASRWireFullRequest 构建固定为PCM 16kHz单声道的上游首包。
func buildASRWireFullRequest(options ASRRequestOptions) asrWireFullRequest {
	options = options.normalized()

	return asrWireFullRequest{
		User: asrWireUser{
			UID:      options.UID,
			Platform: "web",
		},
		Audio: asrWireAudio{
			Format:  "pcm",
			Codec:   "raw",
			Rate:    16000,
			Bits:    16,
			Channel: 1,
		},
		Request: asrWireRequest{
			ModelName:            "bigmodel",
			EnableNonstream:      options.EnableNonstream,
			EnableITN:            options.EnableITN,
			EnablePunctuation:    options.EnablePunctuation,
			EnableDDC:            options.EnableDDC,
			ShowUtterances:       options.ShowUtterances,
			ResultType:           options.ResultType,
			EndWindowSizeMS:      options.EndWindowSizeMS,
			ForceToSpeechTimeMS:  options.ForceToSpeechTimeMS,
			EnableAccelerateText: options.EnableAccelerateText,
			AccelerateScore:      options.AccelerateScore,
		},
	}
}

// ==================== 识别结果 ====================

// ASRUtterance 火山返回的单个分句。
type ASRUtterance struct {
	Text      string
	StartTime int
	EndTime   int
	Definite  bool
}

// ASRRecognitionResult 单次上游响应转换后的结果。
type ASRRecognitionResult struct {
	Text       string
	Utterances []ASRUtterance
	DurationMS int
	Final      bool
	Sequence   *int32
	RequestID  string
	LogID      string
}

// ASRUpstreamError 火山服务端错误帧或业务错误。
type ASRUpstreamError struct {
	Code    uint32
	Message string
	LogID   string
}

// Error 实现error接口。
func (err *ASRUpstreamError) Error() string {
	if err == nil {
		return "ASR上游错误"
	}

	if err.Code == 0 {
		return fmt.Sprintf("ASR上游错误: %s", err.Message)
	}

	return fmt.Sprintf("ASR上游错误(%d): %s", err.Code, err.Message)
}

type asrWireServerEnvelope struct {
	Code      int                 `json:"code"`
	Message   string              `json:"message"`
	Error     string              `json:"error"`
	Result    asrWireServerResult `json:"result"`
	AudioInfo asrWireAudioInfo    `json:"audio_info"`
}

type asrWireServerResult struct {
	Text       string                   `json:"text"`
	Utterances []asrWireServerUtterance `json:"utterances"`
}

type asrWireServerUtterance struct {
	Text      string `json:"text"`
	StartTime int    `json:"start_time"`
	EndTime   int    `json:"end_time"`
	Definite  bool   `json:"definite"`
}

type asrWireAudioInfo struct {
	Duration int `json:"duration"`
}
