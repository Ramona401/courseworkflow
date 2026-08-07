package handlers

// speech_handler_finalization.go — ASR最终成功结算与客户端结果输出
//
// 供应商返回Final即代表本次上游调用成功。
// 无论最终文字是否为空，都必须先按真实PCM秒数结算；
// 空文字只影响用户结果，不得被当作供应商失败而释放积分。

import (
	"github.com/gorilla/websocket"

	"tedna/internal/ai"
	"tedna/internal/models"
)

// finalizeSpeechRecognition 处理供应商最终成功结果。
func (handler *SpeechHandler) finalizeSpeechRecognition(
	browserConn *websocket.Conn,
	upstreamSession *ai.ASRSession,
	userID string,
	billingSession *speechBillingSession,
	result *ai.ASRRecognitionResult,
	finalText string,
	totalAudioBytes int64,
	browserWritable bool,
) {
	actualAudioSeconds :=
		float64(totalAudioBytes) /
			float64(speechPCMBytesPerSecond)

	settleErr :=
		billingSession.settle(
			actualAudioSeconds,
			map[string]interface{}{
				"audio_bytes":          totalAudioBytes,
				"provider_duration_ms": result.DurationMS,
				"provider_request_id":  result.RequestID,
				"provider_log_id":      result.LogID,
				"text_runes":           len([]rune(finalText)),
				"result_empty":         finalText == "",
			},
		)
	if settleErr != nil {
		// 供应商已经成功，计费记录保持reserved供补偿，
		// 不能走桥接defer释放已经发生的外部成本。
		speechHandlerLog.Error(
			"ASR供应商成功但积分结算失败，预留保持待补偿",
			"user_id", userID,
			"request_id", result.RequestID,
			"billing_idempotency_key", billingSession.idempotencyKey,
			"actual_audio_seconds", actualAudioSeconds,
			"error", settleErr,
		)
	}

	if finalText == "" {
		if browserWritable {
			writeSpeechSocketError(
				browserConn,
				"speech_result_empty",
				"没有识别到有效文字，请重新录音",
				upstreamSession.RequestID(),
				upstreamSession.LogID(),
			)
		}

		speechHandlerLog.Info(
			"语音识别完成但结果为空",
			"user_id", userID,
			"request_id", result.RequestID,
			"log_id", result.LogID,
			"audio_bytes", totalAudioBytes,
			"actual_audio_seconds", actualAudioSeconds,
			"duration_ms", result.DurationMS,
			"browser_writable", browserWritable,
			"billing_settled", settleErr == nil,
		)
		return
	}

	finalDelivered := false
	closedDelivered := false

	if browserWritable {
		finalEvent :=
			buildSpeechRecognitionEvent(
				models.SpeechEventFinal,
				finalText,
				true,
				result,
			)

		if err := writeSpeechEvent(
			browserConn,
			finalEvent,
		); err != nil {
			speechHandlerLog.Warn(
				"ASR最终结果已结算但浏览器写回失败",
				"user_id", userID,
				"request_id", result.RequestID,
				"error", err,
			)
		} else {
			finalDelivered = true

			if err := writeSpeechEvent(
				browserConn,
				models.SpeechRecognitionEvent{
					Event:     models.SpeechEventClosed,
					RequestID: result.RequestID,
					LogID:     result.LogID,
					Message:   "语音识别已完成",
				},
			); err != nil {
				speechHandlerLog.Warn(
					"ASR关闭事件写回失败",
					"user_id", userID,
					"request_id", result.RequestID,
					"error", err,
				)
			} else {
				closedDelivered = true
			}
		}
	}

	speechHandlerLog.Info(
		"语音识别完成",
		"user_id", userID,
		"request_id", result.RequestID,
		"log_id", result.LogID,
		"audio_bytes", totalAudioBytes,
		"actual_audio_seconds", actualAudioSeconds,
		"duration_ms", result.DurationMS,
		"text_runes", len(
			[]rune(finalText),
		),
		"browser_writable", browserWritable,
		"final_delivered", finalDelivered,
		"closed_delivered", closedDelivered,
		"billing_settled", settleErr == nil,
	)
}
