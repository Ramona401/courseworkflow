package ai

// client_asr_protocol.go — 火山豆包流式语音识别二进制协议
//
// 火山ASR WebSocket不是普通JSON WebSocket。
// 每一个WebSocket二进制消息内部仍包含一层私有协议：
//
//   4字节协议头
//   + 可选序列号或事件号
//   + 4字节大端Payload长度
//   + Payload
//
// 正常请求和响应的Payload通常使用Gzip压缩。
// 本文件只负责编解码，不负责：
//   - WebSocket建连；
//   - APP ID与Access Token读取；
//   - 业务鉴权；
//   - 浏览器音频接收；
//   - 识别结果的业务转换。
//
// 安全要求：
//   - 所有长度在切片前必须做边界检查；
//   - 解压后大小必须受限，防止压缩炸弹；
//   - 未知消息类型、压缩方式和标志位必须返回错误；
//   - 任何异常帧都不得触发数组越界或panic。

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// ==================== 协议常量 ====================

const (
	// 当前火山协议版本固定为1。
	asrProtocolVersion byte = 0x01

	// Header size字段的单位为4字节。
	// 当前基础头为1个单位，即4字节。
	asrProtocolHeaderWords byte = 0x01

	// 消息类型。
	asrMessageTypeFullClientRequest byte = 0x01
	asrMessageTypeAudioRequest      byte = 0x02
	asrMessageTypeServerResponse    byte = 0x09
	asrMessageTypeServerError       byte = 0x0F

	// 消息标志位采用位掩码：
	// bit0=携带序列号；bit1=最后一包；bit2=携带事件号。
	asrFlagNone         byte = 0x00
	asrFlagSequence     byte = 0x01
	asrFlagLast         byte = 0x02
	asrFlagSequenceLast byte = 0x03
	asrFlagEvent        byte = 0x04
	asrFlagKnownMask    byte = 0x07

	// 序列化方式。
	asrSerializationNone byte = 0x00
	asrSerializationJSON byte = 0x01

	// 压缩方式。
	asrCompressionNone byte = 0x00
	asrCompressionGzip byte = 0x01

	// 单个解压后Payload最大8MB。
	// 识别结果正常远小于该值，本限制用于防御异常响应。
	asrProtocolMaxPayloadBytes = 8 * 1024 * 1024
)

// asrProtocolMessage 是完成协议解码后的内部消息。
type asrProtocolMessage struct {
	MessageType   byte
	Flags         byte
	Serialization byte
	Compression   byte
	Sequence      *int32
	Event         *uint32
	ErrorCode     *uint32
	IsLast        bool
	Payload       []byte
}

// ==================== 客户端帧构造 ====================

// buildASRFullClientFrame 构造建立WebSocket连接后的第一条完整请求。
//
// 请求参数先序列化为JSON，再使用Gzip压缩。
// 客户端序列号必须为正数，正式会话从1开始。
func buildASRFullClientFrame(payload interface{}, sequence int32) ([]byte, error) {
	if sequence <= 0 {
		return nil, fmt.Errorf("ASR首包序列号必须为正数: %d", sequence)
	}

	rawJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化ASR首包失败: %w", err)
	}

	compressed, err := gzipASRPayload(rawJSON)
	if err != nil {
		return nil, fmt.Errorf("压缩ASR首包失败: %w", err)
	}

	return buildASRSequencedPayloadFrame(
		asrMessageTypeFullClientRequest,
		asrFlagSequence,
		asrSerializationJSON,
		asrCompressionGzip,
		sequence,
		compressed,
	)
}

// buildASRAudioFrame 构造音频数据包。
//
// sequence必须传入正数；final=true时协议层自动把序列号改为负数，
// 并设置“携带序列号+最后一包”标志。
func buildASRAudioFrame(pcm []byte, sequence int32, final bool) ([]byte, error) {
	if sequence <= 0 {
		return nil, fmt.Errorf("ASR音频包序列号必须为正数: %d", sequence)
	}

	compressed, err := gzipASRPayload(pcm)
	if err != nil {
		return nil, fmt.Errorf("压缩ASR音频包失败: %w", err)
	}

	flags := asrFlagSequence
	wireSequence := sequence
	if final {
		flags = asrFlagSequenceLast
		wireSequence = -sequence
	}

	return buildASRSequencedPayloadFrame(
		asrMessageTypeAudioRequest,
		flags,
		asrSerializationNone,
		asrCompressionGzip,
		wireSequence,
		compressed,
	)
}

// buildASRSequencedPayloadFrame 构造带序列号的客户端Payload帧。
func buildASRSequencedPayloadFrame(
	messageType byte,
	flags byte,
	serialization byte,
	compression byte,
	sequence int32,
	payload []byte,
) ([]byte, error) {
	if uint64(len(payload)) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("ASR Payload过大: %d字节", len(payload))
	}

	frame := make([]byte, 12+len(payload))
	copy(
		frame[:4],
		buildASRHeader(
			messageType,
			flags,
			serialization,
			compression,
		),
	)

	binary.BigEndian.PutUint32(frame[4:8], uint32(sequence))
	binary.BigEndian.PutUint32(frame[8:12], uint32(len(payload)))
	copy(frame[12:], payload)

	return frame, nil
}

// buildASRHeader 构造4字节基础协议头。
func buildASRHeader(
	messageType byte,
	flags byte,
	serialization byte,
	compression byte,
) []byte {
	return []byte{
		(asrProtocolVersion << 4) |
			asrProtocolHeaderWords,
		(messageType << 4) |
			(flags & 0x0F),
		(serialization << 4) |
			(compression & 0x0F),
		0x00,
	}
}

// ==================== 服务端帧解析 ====================

// parseASRServerFrame 解析火山服务端二进制帧。
func parseASRServerFrame(data []byte) (*asrProtocolMessage, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf(
			"ASR响应长度不足，至少需要4字节，实际%d字节",
			len(data),
		)
	}

	version := data[0] >> 4
	if version != asrProtocolVersion {
		return nil, fmt.Errorf("ASR协议版本不支持: %d", version)
	}

	headerWords := data[0] & 0x0F
	if headerWords == 0 {
		return nil, fmt.Errorf("ASR响应Header size为0")
	}

	headerBytes := int(headerWords) * 4
	if headerBytes < 4 || len(data) < headerBytes {
		return nil, fmt.Errorf(
			"ASR响应Header不完整: header=%d, total=%d",
			headerBytes,
			len(data),
		)
	}

	message := &asrProtocolMessage{
		MessageType:   data[1] >> 4,
		Flags:         data[1] & 0x0F,
		Serialization: data[2] >> 4,
		Compression:   data[2] & 0x0F,
	}

	if message.Flags&^asrFlagKnownMask != 0 {
		return nil, fmt.Errorf("ASR响应标志位不支持: %d", message.Flags)
	}

	message.IsLast = message.Flags&asrFlagLast != 0
	offset := headerBytes

	switch message.MessageType {
	case asrMessageTypeServerResponse:
		if message.Flags&asrFlagSequence != 0 {
			if len(data) < offset+4 {
				return nil, fmt.Errorf("ASR响应缺少4字节序列号")
			}

			sequence := int32(binary.BigEndian.Uint32(data[offset : offset+4]))
			message.Sequence = &sequence
			offset += 4
		}

		if message.Flags&asrFlagEvent != 0 {
			if len(data) < offset+4 {
				return nil, fmt.Errorf("ASR响应缺少4字节事件号")
			}

			event := binary.BigEndian.Uint32(data[offset : offset+4])
			message.Event = &event
			offset += 4
		}

		if message.Serialization != asrSerializationNone &&
			message.Serialization != asrSerializationJSON {
			return nil, fmt.Errorf(
				"ASR响应序列化方式不支持: %d",
				message.Serialization,
			)
		}

		payload, err := readAndDecodeASRPayload(data, offset, message.Compression)
		if err != nil {
			return nil, err
		}
		message.Payload = payload
		return message, nil

	case asrMessageTypeServerError:
		if len(data) < offset+8 {
			return nil, fmt.Errorf("ASR错误响应缺少错误码或Payload长度")
		}

		errorCode := binary.BigEndian.Uint32(data[offset : offset+4])
		message.ErrorCode = &errorCode
		offset += 4

		payload, err := readAndDecodeASRPayload(data, offset, message.Compression)
		if err != nil {
			return nil, err
		}
		message.Payload = payload
		return message, nil

	default:
		return nil, fmt.Errorf(
			"ASR服务端消息类型不支持: 0x%X",
			message.MessageType,
		)
	}
}

// readAndDecodeASRPayload 从指定位置读取长度字段和Payload。
func readAndDecodeASRPayload(data []byte, offset int, compression byte) ([]byte, error) {
	if offset < 0 || len(data) < offset+4 {
		return nil, fmt.Errorf("ASR响应缺少4字节Payload长度")
	}

	payloadSize := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	remaining := len(data) - offset
	if uint64(payloadSize) > uint64(remaining) {
		return nil, fmt.Errorf(
			"ASR Payload长度越界: 声明%d字节，实际仅%d字节",
			payloadSize,
			remaining,
		)
	}

	if uint64(payloadSize) < uint64(remaining) {
		return nil, fmt.Errorf(
			"ASR响应存在未声明尾部数据: 声明%d字节，剩余%d字节",
			payloadSize,
			remaining,
		)
	}

	payload := data[offset : offset+int(payloadSize)]
	return decodeASRPayload(payload, compression)
}

// ==================== 压缩辅助 ====================

// gzipASRPayload 使用标准Gzip压缩Payload。
func gzipASRPayload(raw []byte) ([]byte, error) {
	var buffer bytes.Buffer

	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// decodeASRPayload 按协议压缩方式解码Payload。
func decodeASRPayload(payload []byte, compression byte) ([]byte, error) {
	switch compression {
	case asrCompressionNone:
		if len(payload) > asrProtocolMaxPayloadBytes {
			return nil, fmt.Errorf(
				"ASR未压缩Payload超过限制: %d字节",
				len(payload),
			)
		}

		decoded := make([]byte, len(payload))
		copy(decoded, payload)
		return decoded, nil

	case asrCompressionGzip:
		return ungzipASRPayload(payload)

	default:
		return nil, fmt.Errorf("ASR压缩方式不支持: %d", compression)
	}
}

// ungzipASRPayload 安全解压Gzip Payload。
func ungzipASRPayload(compressed []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("创建ASR Gzip读取器失败: %w", err)
	}
	defer reader.Close()

	limited := io.LimitReader(reader, asrProtocolMaxPayloadBytes+1)
	decoded, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("读取ASR Gzip Payload失败: %w", err)
	}

	if len(decoded) > asrProtocolMaxPayloadBytes {
		return nil, fmt.Errorf(
			"ASR解压后Payload超过%d字节限制",
			asrProtocolMaxPayloadBytes,
		)
	}

	return decoded, nil
}
