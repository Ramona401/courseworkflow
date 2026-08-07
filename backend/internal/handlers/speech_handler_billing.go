package handlers

// speech_handler_billing.go — ASR媒体积分计费适配器
//
// 计费身份：
//   category/node/scene = asr / speech_asr_stream / speech_asr_stream
//   provider            = volcengine
//   model               = ASR配置中的Resource ID
//   variant             = streaming_2_0
//   unit                = audio_second
//
// 预留按配置允许的最大录音秒数；最终成功按真实PCM字节数换算秒数。
// 数据库操作使用独立短超时上下文，浏览器断开或请求上下文取消后仍可完成终态写入。
//
// 结果不确定边界：
//   - 最后一包真实音频已成功发送后，上游读取超时、网络断开或服务排空不能释放预留；
//   - 供应商最终成功后，先把真实秒数和供应商成功事实写入reserved metadata；
//   - 随后执行正式结算；结算失败时补偿程序仍可根据metadata精确补结算。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/services"
)

const (
	speechASRProvider        = "volcengine"
	speechASRVariant         = "streaming_2_0"
	speechASRBillingNodeCode = "speech_asr_stream"
	speechBillingDBTimeout   = 8 * time.Second
)

// speechBillingSession 保存单次ASR预留的进程内状态。
type speechBillingSession struct {
	service        *services.MediaBillingService
	idempotencyKey string
	startedAt      time.Time

	mu               sync.Mutex
	terminal         bool
	preserveReserved bool
}

// reserveSpeechASRBilling 在连接ASR供应商前预留最大可能积分。
func (handler *SpeechHandler) reserveSpeechASRBilling(
	ctx context.Context,
	userID string,
	config *ai.ASRConfig,
) (*speechBillingSession, error) {
	if handler == nil ||
		handler.mediaBillingService == nil ||
		config == nil {
		return nil,
			services.ErrMediaBillingInvalidRequest
	}

	userID =
		strings.TrimSpace(
			userID,
		)

	modelName :=
		strings.TrimSpace(
			config.ResourceID,
		)

	estimatedQuantity :=
		float64(
			config.MaxDurationSeconds,
		)

	if userID == "" ||
		modelName == "" ||
		estimatedQuantity <= 0 {
		return nil,
			fmt.Errorf(
				"%w: ASR计费身份不完整",
				services.ErrMediaBillingInvalidRequest,
			)
	}

	var schoolIDPointer *string

	schoolID, err :=
		repository.GetSchoolIDByUserID(
			ctx,
			userID,
		)

	if err == nil {
		schoolID =
			strings.TrimSpace(
				schoolID,
			)

		if schoolID != "" {
			schoolIDPointer =
				&schoolID
		}
	}

	idempotencyKey :=
		"speech-asr:" +
			uuid.NewString()

	billing, err :=
		handler.mediaBillingService.Reserve(
			ctx,
			&models.MediaBillingReserveRequest{
				UserID:
					userID,
				SchoolID:
					schoolIDPointer,
				BillingCategory:
					models.BillingCategoryASR,
				BillingNodeCode:
					speechASRBillingNodeCode,
				SceneCode:
					speechASRBillingNodeCode,
				MediaType:
					models.MediaTypeASR,
				Provider:
					speechASRProvider,
				ModelName:
					modelName,
				Variant:
					speechASRVariant,
				MediaUnit:
					models.MediaUnitAudioSecond,
				EstimatedQuantity:
					estimatedQuantity,
				IdempotencyKey:
					idempotencyKey,
				Metadata:
					map[string]interface{}{
						"sample_rate":
							16000,
						"bits_per_sample":
							16,
						"channels":
							1,
						"max_duration_seconds":
							config.MaxDurationSeconds,
						"reconciliation_required":
							false,
					},
			},
		)

	if err != nil {
		return nil, err
	}

	return &speechBillingSession{
		service:
			handler.mediaBillingService,
		idempotencyKey:
			billing.IdempotencyKey,
		startedAt:
			time.Now(),
	}, nil
}

// bindExternalTask 把火山request_id绑定到预留记录。
func (session *speechBillingSession) bindExternalTask(
	externalTaskID string,
	metadata map[string]interface{},
) error {
	if session == nil ||
		session.service == nil {
		return services.ErrMediaBillingInvalidRequest
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.terminal {
		return repository.ErrTokenMediaBillingTerminal
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			speechBillingDBTimeout,
		)
	defer cancel()

	_, err :=
		session.service.BindExternalTask(
			ctx,
			&models.MediaBillingBindTaskRequest{
				IdempotencyKey:
					session.idempotencyKey,
				ExternalTaskID:
					strings.TrimSpace(
						externalTaskID,
					),
				Metadata:
					metadata,
			},
		)

	return err
}

// preservePending 标记当前reserved记录必须等待人工或自动补偿。
func (session *speechBillingSession) preservePending(
	reason string,
	actualQuantity float64,
	metadata map[string]interface{},
) {
	if session == nil ||
		session.service == nil {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.terminal {
		return
	}

	session.preserveReserved = true

	payload :=
		cloneSpeechBillingMetadata(
			metadata,
		)

	payload["reconciliation_required"] =
		true
	payload["reconciliation_reason"] =
		truncateSpeechBillingText(
			reason,
		)
	payload["reconciliation_marked_at"] =
		time.Now().
			UTC().
			Format(
				time.RFC3339Nano,
			)

	if actualQuantity > 0 {
		payload["reconciliation_actual_quantity"] =
			actualQuantity
	}

	if err :=
		session.annotateReservedLocked(
			payload,
		); err != nil {
		speechHandlerLog.Error(
			"标记ASR预留待补偿失败",
			"billing_idempotency_key",
			session.idempotencyKey,
			"reason",
			reason,
			"actual_quantity",
			actualQuantity,
			"error",
			err,
		)
	}
}

// settle 按真实音频秒数完成最终结算。
//
// 供应商已经返回最终成功结果，因此先把补偿所需事实写入reserved metadata。
// 即使进程在正式结算前退出，后续补偿仍可精确恢复。
func (session *speechBillingSession) settle(
	actualQuantity float64,
	metadata map[string]interface{},
) error {
	if session == nil ||
		session.service == nil {
		return services.ErrMediaBillingInvalidRequest
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.terminal {
		return nil
	}

	session.preserveReserved = true

	if actualQuantity <= 0 {
		return fmt.Errorf(
			"%w: ASR实际音频时长为0",
			services.ErrMediaBillingInvalidRequest,
		)
	}

	recoveryMetadata :=
		cloneSpeechBillingMetadata(
			metadata,
		)

	recoveryMetadata["provider_succeeded"] =
		true
	recoveryMetadata["reconciliation_required"] =
		true
	recoveryMetadata["reconciliation_reason"] =
		"speech_provider_succeeded_pending_settlement"
	recoveryMetadata["reconciliation_actual_quantity"] =
		actualQuantity
	recoveryMetadata["reconciliation_marked_at"] =
		time.Now().
			UTC().
			Format(
				time.RFC3339Nano,
			)

	annotateErr :=
		session.annotateReservedLocked(
			recoveryMetadata,
		)

	latencyMS :=
		int(
			time.Since(
				session.startedAt,
			).Milliseconds(),
		)

	if latencyMS < 0 {
		latencyMS = 0
	}

	settleMetadata :=
		cloneSpeechBillingMetadata(
			metadata,
		)

	settleMetadata["provider_succeeded"] =
		true
	settleMetadata["reconciliation_required"] =
		false
	settleMetadata["reconciliation_actual_quantity"] =
		actualQuantity

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			speechBillingDBTimeout,
		)
	defer cancel()

	_, settleErr :=
		session.service.Settle(
			ctx,
			&models.MediaBillingSettleRequest{
				IdempotencyKey:
					session.idempotencyKey,
				ActualQuantity:
					actualQuantity,
				LatencyMs:
					latencyMS,
				Metadata:
					settleMetadata,
			},
		)

	if settleErr != nil {
		if annotateErr != nil {
			return fmt.Errorf(
				"ASR结算失败: %w；补偿事实写入也失败: %v",
				settleErr,
				annotateErr,
			)
		}

		return settleErr
	}

	session.terminal = true
	session.preserveReserved = false

	return nil
}

// annotateReservedLocked 合并更新reserved记录metadata。
// 调用方必须已经持有session.mu。
func (session *speechBillingSession) annotateReservedLocked(
	metadata map[string]interface{},
) error {
	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			speechBillingDBTimeout,
		)
	defer cancel()

	_, err :=
		session.service.AnnotateReserved(
			ctx,
			&models.MediaBillingAnnotateRequest{
				IdempotencyKey:
					session.idempotencyKey,
				Metadata:
					metadata,
			},
		)

	return err
}

// releasePending 释放明确失败或取消会话的预留积分。
func (session *speechBillingSession) releasePending(
	status string,
	reason string,
	metadata map[string]interface{},
) {
	if session == nil ||
		session.service == nil {
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.terminal ||
		session.preserveReserved {
		return
	}

	ctx, cancel :=
		context.WithTimeout(
			context.Background(),
			speechBillingDBTimeout,
		)
	defer cancel()

	_, err :=
		session.service.Release(
			ctx,
			&models.MediaBillingReleaseRequest{
				IdempotencyKey:
					session.idempotencyKey,
				Status:
					status,
				FailureReason:
					reason,
				Metadata:
					metadata,
			},
		)

	if err != nil {
		speechHandlerLog.Error(
			"释放ASR媒体积分预留失败",
			"billing_idempotency_key",
			session.idempotencyKey,
			"status",
			status,
			"reason",
			reason,
			"error",
			err,
		)
		return
	}

	session.terminal = true
}

func cloneSpeechBillingMetadata(
	source map[string]interface{},
) map[string]interface{} {
	result :=
		make(
			map[string]interface{},
			len(source)+4,
		)

	for key, value :=
		range source {
		result[key] = value
	}

	return result
}

func truncateSpeechBillingText(
	value string,
) string {
	value =
		strings.TrimSpace(
			value,
		)

	runes :=
		[]rune(value)

	if len(runes) <= 500 {
		return value
	}

	return string(
		runes[:500],
	)
}

// publicSpeechBillingError 把内部计费错误映射为安全的WebSocket错误。
func publicSpeechBillingError(
	err error,
) (
	string,
	string,
) {
	switch {
	case errors.Is(
		err,
		services.ErrMediaBillingPriceNotConfigured,
	):
		return "speech_billing_unavailable",
			"语音识别积分计费尚未配置，请联系管理员"

	case errors.Is(
		err,
		repository.ErrInsufficientBalance,
	):
		return "speech_credits_insufficient",
			"积分余额不足，暂时无法使用语音识别"

	case errors.Is(
		err,
		repository.ErrTokenAccountNotFound,
	):
		return "speech_credit_account_missing",
			"尚未开通个人积分账户，暂时无法使用语音识别"

	case errors.Is(
		err,
		repository.ErrAccountSuspended,
	):
		return "speech_credit_account_unavailable",
			"积分账户当前不可用，请联系管理员"

	default:
		return "speech_billing_failed",
			"语音识别积分校验失败，请稍后重试"
	}
}
