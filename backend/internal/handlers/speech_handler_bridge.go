package handlers

// speech_handler_bridge.go — 浏览器音频、ASR结果与积分结算主循环
//
// 关键边界：
//   - stop前没有提交最终音频时，明确失败或取消释放预留；
//   - 最后一包真实音频成功发送后，只等待供应商最终结果；
//   - 最终结果成功时按真实PCM秒数结算；
//   - 最终音频提交后发生网络中断、读取超时、服务排空或桥接总超时，
//     结果无法确定，保留reserved并写入精确补偿metadata；
//   - 供应商明确返回业务错误时仍按失败释放。

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"tedna/internal/ai"
	"tedna/internal/models"
)

// runSpeechBridge 双向转发浏览器音频和豆包识别结果。
func (handler *SpeechHandler) runSpeechBridge(
	ctx context.Context,
	browserConn *websocket.Conn,
	upstreamSession *ai.ASRSession,
	userID string,
	billingSession *speechBillingSession,
) {
	browserMessages :=
		make(
			chan speechBrowserMessage,
			4,
		)

	upstreamMessages :=
		make(
			chan speechUpstreamMessage,
			8,
		)

	go readSpeechBrowserMessages(
		ctx,
		browserConn,
		browserMessages,
	)

	go readSpeechUpstreamMessages(
		ctx,
		upstreamSession,
		upstreamMessages,
	)

	var browserInput <-chan speechBrowserMessage =
		browserMessages

	browserWritable := true

	maxSessionDuration :=
		time.Duration(
			upstreamSession.MaxDurationSeconds(),
		)*time.Second +
			speechFinishGracePeriod

	sessionTimer :=
		time.NewTimer(
			maxSessionDuration,
		)
	defer sessionTimer.Stop()

	maxPCMBytes :=
		int64(
			upstreamSession.MaxDurationSeconds(),
		) *
			speechPCMBytesPerSecond

	var pendingAudio []byte
	var totalAudioBytes int64
	var stopping bool
	var latestText string
	var lastSentText string

	releaseStatus :=
		models.MediaBillingStatusFailed

	releaseReason :=
		"speech_bridge_ended_without_final_result"

	preserveFinalUncertainty :=
		func(
			reason string,
			cause error,
		) {
			actualAudioSeconds :=
				float64(
					totalAudioBytes,
				) /
					float64(
						speechPCMBytesPerSecond,
					)

			metadata :=
				map[string]interface{}{
					"audio_bytes":
						totalAudioBytes,
					"actual_audio_seconds":
						actualAudioSeconds,
					"provider_request_id":
						upstreamSession.RequestID(),
					"provider_log_id":
						upstreamSession.LogID(),
					"final_audio_submitted":
						true,
				}

			if cause != nil {
				metadata["uncertain_error"] =
					truncateSpeechBillingText(
						cause.Error(),
					)
			}

			billingSession.preservePending(
				reason,
				actualAudioSeconds,
				metadata,
			)
		}

	defer func() {
		if billingSession == nil {
			return
		}

		billingSession.releasePending(
			releaseStatus,
			releaseReason,
			map[string]interface{}{
				"audio_bytes":
					totalAudioBytes,
				"provider_request_id":
					upstreamSession.RequestID(),
				"provider_log_id":
					upstreamSession.LogID(),
				"final_audio_submitted":
					stopping,
			},
		)
	}()

	for {
		select {
		case <-ctx.Done():
			if stopping {
				preserveFinalUncertainty(
					"speech_context_cancelled_after_final",
					ctx.Err(),
				)
				return
			}

			releaseStatus =
				models.MediaBillingStatusCancelled
			releaseReason =
				"speech_context_cancelled"
			return

		case <-sessionTimer.C:
			if stopping {
				preserveFinalUncertainty(
					"speech_final_result_timeout",
					context.DeadlineExceeded,
				)
			} else {
				releaseReason =
					"speech_session_timeout"
			}

			if browserWritable {
				writeSpeechSocketError(
					browserConn,
					"speech_session_timeout",
					"本次语音输入时间已到，请重新开始",
					upstreamSession.RequestID(),
					upstreamSession.LogID(),
				)
			}

			return

		case browserMessage :=
			<-browserInput:
			if browserMessage.err != nil {
				releaseStatus =
					models.MediaBillingStatusCancelled
				releaseReason =
					"speech_browser_disconnected"

				if websocket.IsUnexpectedCloseError(
					browserMessage.err,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseNoStatusReceived,
				) {
					releaseStatus =
						models.MediaBillingStatusFailed
					releaseReason =
						"speech_browser_connection_failed"

					speechHandlerLog.Warn(
						"语音浏览器连接异常断开",
						"user_id",
						userID,
						"request_id",
						upstreamSession.RequestID(),
						"error",
						browserMessage.err,
					)
				}

				return
			}

			switch browserMessage.messageType {
			case websocket.BinaryMessage:
				if stopping {
					continue
				}

				if err :=
					validateSpeechAudioChunk(
						browserMessage.data,
					); err != nil {
					releaseReason =
						"speech_audio_invalid"

					writeSpeechSocketError(
						browserConn,
						"speech_audio_invalid",
						err.Error(),
						upstreamSession.RequestID(),
						upstreamSession.LogID(),
					)
					return
				}

				totalAudioBytes +=
					int64(
						len(
							browserMessage.data,
						),
					)

				if totalAudioBytes >
					maxPCMBytes {
					releaseReason =
						"speech_audio_too_long"

					writeSpeechSocketError(
						browserConn,
						"speech_audio_too_long",
						"录音时长超过平台限制",
						upstreamSession.RequestID(),
						upstreamSession.LogID(),
					)
					return
				}

				if len(
					pendingAudio,
				) > 0 {
					if err :=
						upstreamSession.SendAudioChunk(
							ctx,
							pendingAudio,
							false,
						); err != nil {
						releaseReason =
							"speech_upstream_audio_send_failed"

						handler.writeUpstreamError(
							browserConn,
							upstreamSession,
							userID,
							err,
							browserWritable,
						)
						return
					}
				}

				pendingAudio =
					append(
						pendingAudio[:0],
						browserMessage.data...,
					)

			case websocket.TextMessage:
				control, err :=
					parseSpeechControlRequest(
						browserMessage.data,
					)

				if err != nil {
					releaseReason =
						"speech_control_invalid"

					writeSpeechSocketError(
						browserConn,
						"speech_control_invalid",
						err.Error(),
						upstreamSession.RequestID(),
						upstreamSession.LogID(),
					)
					return
				}

				switch control.Action {
				case models.SpeechActionStop:
					if stopping {
						continue
					}

					if len(
						pendingAudio,
					) == 0 {
						releaseReason =
							"speech_audio_empty"

						writeSpeechSocketError(
							browserConn,
							"speech_audio_empty",
							"没有收到可识别的语音",
							upstreamSession.RequestID(),
							upstreamSession.LogID(),
						)
						return
					}

					if err :=
						upstreamSession.SendAudioChunk(
							ctx,
							pendingAudio,
							true,
						); err != nil {
						releaseReason =
							"speech_upstream_final_audio_send_failed"

						handler.writeUpstreamError(
							browserConn,
							upstreamSession,
							userID,
							err,
							browserWritable,
						)
						return
					}

					pendingAudio = nil
					stopping = true

					// 最后一包已经被上游写入成功。
					// 此后浏览器输入不再影响供应商调用和计费终态。
					browserInput = nil

				case models.SpeechActionCancel:
					releaseStatus =
						models.MediaBillingStatusCancelled
					releaseReason =
						"speech_user_cancelled"

					_ =
						writeSpeechEvent(
							browserConn,
							models.SpeechRecognitionEvent{
								Event:
									models.SpeechEventClosed,
								RequestID:
									upstreamSession.RequestID(),
								LogID:
									upstreamSession.LogID(),
								Message:
									"本次语音输入已取消",
							},
						)
					return

				case models.SpeechActionStart:
					releaseReason =
						"speech_start_duplicate"

					writeSpeechSocketError(
						browserConn,
						"speech_start_duplicate",
						"语音识别已经开始",
						upstreamSession.RequestID(),
						upstreamSession.LogID(),
					)
					return

				default:
					releaseReason =
						"speech_control_unknown"

					writeSpeechSocketError(
						browserConn,
						"speech_control_unknown",
						"未知的语音控制动作",
						upstreamSession.RequestID(),
						upstreamSession.LogID(),
					)
					return
				}

			default:
				releaseReason =
					"speech_message_type_invalid"

				writeSpeechSocketError(
					browserConn,
					"speech_message_type_invalid",
					"语音连接只接受JSON控制消息和PCM二进制音频",
					upstreamSession.RequestID(),
					upstreamSession.LogID(),
				)
				return
			}

		case upstreamMessage :=
			<-upstreamMessages:
			if upstreamMessage.err != nil {
				if ctx.Err() != nil {
					if stopping {
						preserveFinalUncertainty(
							"speech_context_cancelled_after_final",
							ctx.Err(),
						)
						return
					}

					releaseStatus =
						models.MediaBillingStatusCancelled
					releaseReason =
						"speech_context_cancelled"
					return
				}

				var explicitProviderError *ai.ASRUpstreamError

				if stopping &&
					!errors.As(
						upstreamMessage.err,
						&explicitProviderError,
					) {
					preserveFinalUncertainty(
						"speech_upstream_result_uncertain_after_final",
						upstreamMessage.err,
					)

					handler.writeUpstreamError(
						browserConn,
						upstreamSession,
						userID,
						upstreamMessage.err,
						browserWritable,
					)
					return
				}

				releaseReason =
					"speech_upstream_result_failed"

				handler.writeUpstreamError(
					browserConn,
					upstreamSession,
					userID,
					upstreamMessage.err,
					browserWritable,
				)
				return
			}

			result :=
				upstreamMessage.result

			if result == nil {
				continue
			}

			text :=
				strings.TrimSpace(
					result.Text,
				)

			if text != "" {
				latestText = text
			}

			if result.Final {
				finalText := text

				if finalText == "" {
					finalText =
						latestText
				}

				handler.finalizeSpeechRecognition(
					browserConn,
					upstreamSession,
					userID,
					billingSession,
					result,
					finalText,
					totalAudioBytes,
					browserWritable,
				)
				return
			}

			if text == "" ||
				text == lastSentText {
				continue
			}

			lastSentText = text

			if !browserWritable {
				continue
			}

			if err :=
				writeSpeechEvent(
					browserConn,
					buildSpeechRecognitionEvent(
						models.SpeechEventPartial,
						text,
						false,
						result,
					),
				); err != nil {
				if stopping {
					browserWritable = false
					continue
				}

				releaseStatus =
					models.MediaBillingStatusCancelled
				releaseReason =
					"speech_partial_write_failed"
				return
			}
		}
	}
}
