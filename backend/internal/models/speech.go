package models

// speech.go — 全平台语音输入的浏览器通信协议模型
//
// 本文件只定义“浏览器与TE-DNA后端”之间的稳定协议，不包含：
//   1. 火山引擎私有二进制协议；
//   2. 数据库配置读取；
//   3. WebSocket连接管理；
//   4. 具体业务输入框状态。
//
// 语音输入的职责边界：
//   - 老师点击麦克风后，浏览器向后端发送一条start控制消息；
//   - 后续以WebSocket二进制消息发送16kHz、单声道、16bit PCM音频；
//   - 老师停止录音时发送stop控制消息；
//   - 后端把豆包返回的识别结果转换为下方统一事件；
//   - 前端只把识别文字写进当前输入框，绝不自动发送给AI。
//
// 隐私边界：
//   - 本协议不要求浏览器上传真实姓名、设备序列号或其它身份信息；
//   - 音频只在当前WebSocket连接中流转，不作为业务数据落库；
//   - 服务端日志不得记录完整音频和完整识别正文。

// ==================== 浏览器控制动作 ====================

const (
	// SpeechActionStart 表示浏览器准备开始发送音频。
	SpeechActionStart = "start"

	// SpeechActionStop 表示浏览器已经停止采集音频。
	// 后端收到后向豆包发送最后一个音频包，并等待最终识别结果。
	SpeechActionStop = "stop"

	// SpeechActionCancel 表示用户主动取消本次语音输入。
	// 取消后不要求等待最终识别结果。
	SpeechActionCancel = "cancel"
)

// ==================== 服务端事件类型 ====================

const (
	// SpeechEventReady 表示后端和豆包上游连接均已建立，
	// 浏览器可以开始发送PCM音频包。
	SpeechEventReady = "ready"

	// SpeechEventPartial 表示临时识别结果。
	// 后续结果可能覆盖或修正本次文字。
	SpeechEventPartial = "partial"

	// SpeechEventFinal 表示本次录音的最终识别结果。
	SpeechEventFinal = "final"

	// SpeechEventError 表示当前语音识别连接发生错误。
	SpeechEventError = "error"

	// SpeechEventClosed 表示连接已正常关闭。
	SpeechEventClosed = "closed"
)

// ==================== 浏览器请求模型 ====================

// SpeechStartRequest 浏览器开始录音时发送的JSON控制消息。
//
// 当前正式协议固定要求：
//   - 16000Hz；
//   - 16bit；
//   - 单声道；
//   - PCM little-endian。
//
// 字段仍由浏览器显式提交，便于服务端发现客户端实现错误，
// 但服务端不得因为客户端提交其它值而自动猜测或静默转换。
type SpeechStartRequest struct {
	Action        string `json:"action"`
	SampleRate    int    `json:"sample_rate"`
	BitsPerSample int    `json:"bits_per_sample"`
	Channels      int    `json:"channels"`
}

// SpeechControlRequest stop和cancel使用的轻量控制消息。
type SpeechControlRequest struct {
	Action string `json:"action"`
}

// ==================== 服务端响应模型 ====================

// SpeechUtterance 单个语音分句。
//
// Definite=true表示豆包已经通过VAD与二遍识别确认该分句，
// 后续通常不会再修改该分句内容。
type SpeechUtterance struct {
	Text      string `json:"text"`
	StartTime int    `json:"start_time"`
	EndTime   int    `json:"end_time"`
	Definite  bool   `json:"definite"`
}

// SpeechRecognitionEvent 后端发给浏览器的统一语音事件。
//
// RequestID由TE-DNA生成，用于串联一次识别请求；
// LogID由火山引擎返回，只用于服务端和控制台排错；
// 浏览器不得把二者当作授权凭证。
type SpeechRecognitionEvent struct {
	Event      string            `json:"event"`
	Text       string            `json:"text,omitempty"`
	Final      bool              `json:"final,omitempty"`
	Utterances []SpeechUtterance `json:"utterances,omitempty"`
	DurationMS int               `json:"duration_ms,omitempty"`
	RequestID  string            `json:"request_id,omitempty"`
	LogID      string            `json:"log_id,omitempty"`
	Code       string            `json:"code,omitempty"`
	Message    string            `json:"message,omitempty"`
}
