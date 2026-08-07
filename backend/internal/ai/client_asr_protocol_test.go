package ai

// client_asr_protocol_test.go — ASR私有二进制协议的确定性单元测试
//
// 本测试不访问数据库、不连接火山、不读取真实密钥，也不处理真实音频。
// 只验证：
//   - 首包协议头与正序列号；
//   - 音频最后一包标志与负序列号；
//   - Gzip压缩和解压；
//   - 服务端序列号、事件号和结束标志；
//   - 服务端错误帧；
//   - 非法长度的fail-closed处理。

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

// TestASRBuildFullClientFrame 验证首包固定协议字段、序列号与JSON内容。
func TestASRBuildFullClientFrame(t *testing.T) {
	input := map[string]interface{}{
		"audio": map[string]interface{}{
			"format": "pcm",
			"rate":   float64(16000),
		},
	}

	frame, err := buildASRFullClientFrame(input, 1)
	if err != nil {
		t.Fatalf("构造ASR首包失败: %v", err)
	}

	if len(frame) < 12 {
		t.Fatalf("ASR首包长度异常: %d", len(frame))
	}

	if frame[0] != 0x11 {
		t.Fatalf("协议版本/Header size错误: 0x%X", frame[0])
	}

	if frame[1] != 0x11 {
		t.Fatalf("首包消息类型/标志错误: 0x%X", frame[1])
	}

	if frame[2] != 0x11 {
		t.Fatalf("首包序列化/压缩错误: 0x%X", frame[2])
	}

	sequence := int32(binary.BigEndian.Uint32(frame[4:8]))
	if sequence != 1 {
		t.Fatalf("首包序列号错误: %d", sequence)
	}

	payloadSize := int(binary.BigEndian.Uint32(frame[8:12]))
	if payloadSize != len(frame)-12 {
		t.Fatalf(
			"首包Payload长度不一致: 声明%d，实际%d",
			payloadSize,
			len(frame)-12,
		)
	}

	decoded, err := ungzipASRPayload(frame[12:])
	if err != nil {
		t.Fatalf("解压ASR首包失败: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(decoded, &output); err != nil {
		t.Fatalf("解析ASR首包JSON失败: %v", err)
	}

	audio, ok := output["audio"].(map[string]interface{})
	if !ok {
		t.Fatal("ASR首包缺少audio对象")
	}

	if audio["format"] != "pcm" {
		t.Fatalf("音频格式不正确: %v", audio["format"])
	}
}

// TestASRBuildFinalAudioFrame 验证最后一包标志、负序列号和PCM内容。
func TestASRBuildFinalAudioFrame(t *testing.T) {
	pcm := []byte{
		0x01, 0x02,
		0x03, 0x04,
	}

	frame, err := buildASRAudioFrame(pcm, 2, true)
	if err != nil {
		t.Fatalf("构造ASR音频包失败: %v", err)
	}

	if frame[1] != 0x23 {
		t.Fatalf("最后一包消息类型/标志错误: 0x%X", frame[1])
	}

	if frame[2] != 0x01 {
		t.Fatalf("音频包序列化/压缩错误: 0x%X", frame[2])
	}

	sequence := int32(binary.BigEndian.Uint32(frame[4:8]))
	if sequence != -2 {
		t.Fatalf("最后一包序列号不是负数: %d", sequence)
	}

	payloadSize := int(binary.BigEndian.Uint32(frame[8:12]))
	if payloadSize != len(frame)-12 {
		t.Fatalf(
			"音频包Payload长度不一致: 声明%d，实际%d",
			payloadSize,
			len(frame)-12,
		)
	}

	decoded, err := ungzipASRPayload(frame[12:])
	if err != nil {
		t.Fatalf("解压ASR音频包失败: %v", err)
	}

	if !bytes.Equal(decoded, pcm) {
		t.Fatalf("ASR音频内容不一致: %v", decoded)
	}
}

// TestASRParseServerResponse 验证带序列号和事件号的正常服务端响应。
func TestASRParseServerResponse(t *testing.T) {
	rawPayload := []byte(
		`{"result":{"text":"测试语音"},"audio_info":{"duration":800}}`,
	)

	compressed, err := gzipASRPayload(rawPayload)
	if err != nil {
		t.Fatalf("压缩测试Payload失败: %v", err)
	}

	frame := make([]byte, 16+len(compressed))
	copy(
		frame[:4],
		buildASRHeader(
			asrMessageTypeServerResponse,
			asrFlagSequence|asrFlagEvent,
			asrSerializationJSON,
			asrCompressionGzip,
		),
	)
	binary.BigEndian.PutUint32(frame[4:8], uint32(7))
	binary.BigEndian.PutUint32(frame[8:12], uint32(450))
	binary.BigEndian.PutUint32(frame[12:16], uint32(len(compressed)))
	copy(frame[16:], compressed)

	message, err := parseASRServerFrame(frame)
	if err != nil {
		t.Fatalf("解析ASR服务端响应失败: %v", err)
	}

	if message.Sequence == nil || *message.Sequence != 7 {
		t.Fatalf("ASR响应序列号错误: %v", message.Sequence)
	}

	if message.Event == nil || *message.Event != 450 {
		t.Fatalf("ASR响应事件号错误: %v", message.Event)
	}

	if !bytes.Equal(message.Payload, rawPayload) {
		t.Fatalf("ASR响应Payload不一致: %s", string(message.Payload))
	}
}

// TestASRParseLastServerResponse 验证结束响应的负序列号和结束标记。
func TestASRParseLastServerResponse(t *testing.T) {
	rawPayload := []byte(`{"result":{"text":"最终文本"}}`)
	compressed, err := gzipASRPayload(rawPayload)
	if err != nil {
		t.Fatalf("压缩结束Payload失败: %v", err)
	}

	frame := make([]byte, 12+len(compressed))
	copy(
		frame[:4],
		buildASRHeader(
			asrMessageTypeServerResponse,
			asrFlagSequenceLast,
			asrSerializationJSON,
			asrCompressionGzip,
		),
	)
	negativeSequence := int32(-9)
	binary.BigEndian.PutUint32(frame[4:8], uint32(negativeSequence))
	binary.BigEndian.PutUint32(frame[8:12], uint32(len(compressed)))
	copy(frame[12:], compressed)

	message, err := parseASRServerFrame(frame)
	if err != nil {
		t.Fatalf("解析ASR结束响应失败: %v", err)
	}

	if !message.IsLast {
		t.Fatal("ASR结束响应未标记为最后一包")
	}

	if message.Sequence == nil || *message.Sequence != -9 {
		t.Fatalf("ASR结束响应序列号错误: %v", message.Sequence)
	}
}

// TestASRParseServerError 验证服务端错误码与错误正文。
func TestASRParseServerError(t *testing.T) {
	rawPayload := []byte(`{"message":"音频格式不正确"}`)

	frame := make([]byte, 12+len(rawPayload))
	copy(
		frame[:4],
		buildASRHeader(
			asrMessageTypeServerError,
			asrFlagNone,
			asrSerializationJSON,
			asrCompressionNone,
		),
	)
	binary.BigEndian.PutUint32(frame[4:8], uint32(45000151))
	binary.BigEndian.PutUint32(frame[8:12], uint32(len(rawPayload)))
	copy(frame[12:], rawPayload)

	message, err := parseASRServerFrame(frame)
	if err != nil {
		t.Fatalf("解析ASR错误帧失败: %v", err)
	}

	if message.ErrorCode == nil || *message.ErrorCode != uint32(45000151) {
		t.Fatalf("ASR错误码不正确: %v", message.ErrorCode)
	}

	if !bytes.Equal(message.Payload, rawPayload) {
		t.Fatalf("ASR错误正文不一致: %s", string(message.Payload))
	}
}

// TestASRRejectsInvalidPayloadLength 验证声明长度大于真实长度时必须拒绝。
func TestASRRejectsInvalidPayloadLength(t *testing.T) {
	frame := make([]byte, 12)
	copy(
		frame[:4],
		buildASRHeader(
			asrMessageTypeServerResponse,
			asrFlagSequence,
			asrSerializationJSON,
			asrCompressionNone,
		),
	)
	binary.BigEndian.PutUint32(frame[4:8], uint32(1))
	binary.BigEndian.PutUint32(frame[8:12], uint32(100))

	if _, err := parseASRServerFrame(frame); err == nil {
		t.Fatal("非法Payload长度未被拒绝")
	}
}
