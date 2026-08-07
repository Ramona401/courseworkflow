package handlers

// speech_handler_bridge_io.go — ASR桥接消息读取与安全错误输出
//
// 浏览器和上游各自只有一个独占读取goroutine；
// Handler主循环仍是唯一浏览器写入方。

import (
	"context"

	"github.com/gorilla/websocket"

	"tedna/internal/ai"
)

// speechBrowserMessage 是浏览器读取goroutine交给主循环的消息。
type speechBrowserMessage struct {
	messageType int
	data        []byte
	err         error
}

// speechUpstreamMessage 是豆包读取goroutine交给主循环的消息。
type speechUpstreamMessage struct {
	result *ai.ASRRecognitionResult
	err    error
}

// writeUpstreamError 记录完整上游错误，并在浏览器仍可写时发送安全文案。
func (handler *SpeechHandler) writeUpstreamError(
	browserConn *websocket.Conn,
	upstreamSession *ai.ASRSession,
	userID string,
	err error,
	browserWritable bool,
) {
	code, message :=
		publicSpeechError(err)

	speechHandlerLog.Error(
		"ASR上游处理失败",
		"user_id", userID,
		"request_id", upstreamSession.RequestID(),
		"log_id", upstreamSession.LogID(),
		"error", err,
	)

	if !browserWritable {
		return
	}

	writeSpeechSocketError(
		browserConn,
		code,
		message,
		upstreamSession.RequestID(),
		upstreamSession.LogID(),
	)
}

// readSpeechBrowserMessages 独占浏览器WebSocket读取。
func readSpeechBrowserMessages(
	ctx context.Context,
	conn *websocket.Conn,
	output chan<- speechBrowserMessage,
) {
	for {
		messageType, data, err :=
			conn.ReadMessage()

		message := speechBrowserMessage{
			messageType: messageType,
			data:        data,
			err:         err,
		}

		select {
		case output <- message:
		case <-ctx.Done():
			return
		}

		if err != nil {
			return
		}
	}
}

// readSpeechUpstreamMessages 独占豆包WebSocket读取。
func readSpeechUpstreamMessages(
	ctx context.Context,
	session *ai.ASRSession,
	output chan<- speechUpstreamMessage,
) {
	for {
		result, err :=
			session.ReadResult(ctx)

		message := speechUpstreamMessage{
			result: result,
			err:    err,
		}

		select {
		case output <- message:
		case <-ctx.Done():
			return
		}

		if err != nil ||
			(result != nil &&
				result.Final) {
			return
		}
	}
}
