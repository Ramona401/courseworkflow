package services

// assistant_runtime_chat_service.go
//
// 编排教学智能体单轮运行聊天：先领取额度和唯一主轮次，再使用不可变版本、
// 最近正式消息和学生输入调用AI。成功必须原子结算；任何配置错误、快照错误、
// 上游断流、客户端取消、流建立失败或输出回调错误都必须失败结算并释放
// active_turn_id。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	// 结算使用与浏览器请求取消解耦的短时上下文。
	assistantRuntimeSettlementTimeout = 20 * time.Second
)

var (
	// ErrAssistantRuntimeChatInvalidRequest 表示学生消息或运行参数非法。
	ErrAssistantRuntimeChatInvalidRequest = errors.New(
		"教学智能体运行聊天请求无效",
	)

	// ErrAssistantRuntimeChatSnapshotInvalid 表示不可变版本无法安全运行。
	ErrAssistantRuntimeChatSnapshotInvalid = errors.New(
		"教学智能体不可变运行快照无效",
	)

	// ErrAssistantRuntimeChatUnavailable 表示聊天服务依赖或结果不可用。
	ErrAssistantRuntimeChatUnavailable = errors.New(
		"教学智能体运行聊天服务不可用",
	)
)

var assistantRuntimeChatLog = logger.WithModule(
	"services.assistant_runtime_chat",
)

// AssistantRuntimeChatService 是公开运行和教师内部预览共用的聊天服务。
type AssistantRuntimeChatService struct {
	cfg            *config.Config
	billingService *AssistantRuntimeBillingService
}

// NewAssistantRuntimeChatService 创建运行聊天服务。
func NewAssistantRuntimeChatService(
	cfg *config.Config,
	billingService *AssistantRuntimeBillingService,
) *AssistantRuntimeChatService {
	return &AssistantRuntimeChatService{
		cfg:            cfg,
		billingService: billingService,
	}
}

// Chat 保留直接调用入口。
//
// onChunk只接收学生可见正文。若其返回错误，AI流立即中止并按失败轮次结算。
func (s *AssistantRuntimeChatService) Chat(
	ctx context.Context,
	tokenString string,
	sessionID string,
	studentMessage string,
	onChunk func(string) error,
) (
	*models.AssistantRuntimeChatResponse,
	error,
) {
	return s.ChatWithReady(
		ctx,
		tokenString,
		sessionID,
		studentMessage,
		nil,
		onChunk,
	)
}

// ChatWithReady 在完成令牌、额度、账户、快照和模型配置检查后，调用onReady。
//
// HTTP Handler应在onReady中写入SSE响应头和connected事件。这样可以保证：
//   - 未通过授权或额度检查时仍能返回普通HTTP JSON错误；
//   - SSE响应建立前已经领取唯一主轮次；
//   - onReady失败也会执行失败结算释放主轮次。
func (s *AssistantRuntimeChatService) ChatWithReady(
	ctx context.Context,
	tokenString string,
	sessionID string,
	studentMessage string,
	onReady func() error,
	onChunk func(string) error,
) (
	response *models.AssistantRuntimeChatResponse,
	err error,
) {
	if ctx == nil {
		ctx = context.Background()
	}

	if s == nil ||
		s.cfg == nil ||
		s.billingService == nil {
		return nil, ErrAssistantRuntimeChatUnavailable
	}

	tokenString = strings.TrimSpace(tokenString)
	sessionID = strings.TrimSpace(sessionID)
	studentMessage = strings.TrimSpace(studentMessage)

	inputChars := utf8.RuneCountInString(
		studentMessage,
	)

	if tokenString == "" ||
		sessionID == "" ||
		studentMessage == "" ||
		inputChars > assistantRuntimeChatMaxMessageRunes {
		return nil, ErrAssistantRuntimeChatInvalidRequest
	}

	billingContext, err := s.billingService.AuthorizeAndClaimTurn(
		ctx,
		tokenString,
		sessionID,
	)
	if err != nil {
		return nil, err
	}

	failure := &models.AssistantRuntimeTurnFailure{
		InputChars: inputChars,
		ErrorCode:  "runtime_failed",
	}

	finalized := false

	// 无论后续在哪个步骤返回，主轮次都必须被成功或失败结算。
	defer func() {
		if finalized {
			return
		}

		settlementErr := s.completeFailureDetached(
			ctx,
			billingContext,
			failure,
		)

		if settlementErr == nil ||
			assistantRuntimeTurnAlreadyClosed(
				settlementErr,
			) {
			return
		}

		assistantRuntimeChatLog.Error(
			"运行聊天失败结算未完成",
			"session_id",
			sessionID,
			"turn_id",
			billingContext.Claim.TurnID,
			"error",
			settlementErr,
		)

		if err == nil {
			err = settlementErr
			return
		}

		err = errors.Join(
			err,
			settlementErr,
		)
	}()

	messages, buildErr := buildAssistantRuntimeChatMessages(
		billingContext.Authorization,
		billingContext.Claim,
		studentMessage,
	)
	if buildErr != nil {
		failure.ErrorCode = "snapshot_invalid"
		return nil, buildErr
	}

	aiConfig, configErr := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		ai.SceneCoursewareAssistantRuntime,
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if configErr != nil {
		failure.ErrorCode = "ai_config_unavailable"
		return nil, fmt.Errorf(
			"加载教学智能体运行模型配置失败: %w",
			configErr,
		)
	}

	if onReady != nil {
		if readyErr := onReady(); readyErr != nil {
			failure.ErrorCode = "stream_open_failed"
			return nil, readyErr
		}
	}

	studentAt := time.Now().UTC()

	streamResult, streamErr := ai.CallAIStreamMessagesContext(
		ctx,
		aiConfig,
		messages,
		func(chunk string) error {
			if onChunk == nil {
				return nil
			}

			return onChunk(
				chunk,
			)
		},
		billingContext.TraceContext,
	)

	if streamResult != nil {
		failure.ModelName = streamResult.ModelUsed
		failure.LatencyMs = assistantRuntimeLatencyToInt(
			streamResult.LatencyMs,
		)
	}

	if streamErr != nil {
		failure.ErrorCode = assistantRuntimeChatFailureCode(
			streamErr,
		)
		return nil, streamErr
	}

	if streamResult == nil ||
		strings.TrimSpace(
			streamResult.Content,
		) == "" {
		failure.ErrorCode = "ai_empty_response"
		return nil, ErrAssistantRuntimeChatUnavailable
	}

	assistantAt := time.Now().UTC()

	completion := &models.AssistantRuntimeTurnCompletion{
		StudentMessage: models.AssistantRuntimeMessage{
			Role:      models.AssistantRuntimeMessageRoleStudent,
			Content:   studentMessage,
			CreatedAt: &studentAt,
		},
		AssistantMessage: models.AssistantRuntimeMessage{
			Role:      models.AssistantRuntimeMessageRoleAssistant,
			Content:   streamResult.Content,
			CreatedAt: &assistantAt,
		},
		InputChars: inputChars,
		OutputChars: utf8.RuneCountInString(
			streamResult.Content,
		),
		InputTokens:  streamResult.PromptTokens,
		OutputTokens: streamResult.CompletionTokens,
		ModelName:    streamResult.ModelUsed,
		LatencyMs: assistantRuntimeLatencyToInt(
			streamResult.LatencyMs,
		),
	}

	settlementContext, cancel := assistantRuntimeDetachedContext(
		ctx,
	)

	session, settlementErr := s.billingService.CompleteSuccess(
		settlementContext,
		billingContext,
		completion,
	)

	cancel()

	if settlementErr != nil {
		failure.ErrorCode = "success_settlement_failed"
		failure.ModelName = streamResult.ModelUsed
		failure.LatencyMs = completion.LatencyMs

		return nil, fmt.Errorf(
			"完成教学智能体成功结算失败: %w",
			settlementErr,
		)
	}

	if session == nil {
		failure.ErrorCode = "success_settlement_invalid"
		return nil, ErrAssistantRuntimeChatUnavailable
	}

	finalized = true

	remainingTurns := session.MaxTurns - session.TurnCount
	if remainingTurns < 0 {
		remainingTurns = 0
	}

	return &models.AssistantRuntimeChatResponse{
		TurnID:         billingContext.Claim.TurnID,
		Message:        completion.AssistantMessage,
		TurnCount:      session.TurnCount,
		RemainingTurns: remainingTurns,
		SessionStatus:  session.Status,
	}, nil
}

// completeFailureDetached 使用独立短时上下文释放失败轮次。
func (s *AssistantRuntimeChatService) completeFailureDetached(
	ctx context.Context,
	billingContext *AssistantRuntimeBillingContext,
	failure *models.AssistantRuntimeTurnFailure,
) error {
	settlementContext, cancel := assistantRuntimeDetachedContext(
		ctx,
	)
	defer cancel()

	_, err := s.billingService.CompleteFailure(
		settlementContext,
		billingContext,
		failure,
	)

	return err
}

// assistantRuntimeDetachedContext 在浏览器断开后仍允许完成数据库结算。
func assistantRuntimeDetachedContext(
	ctx context.Context,
) (
	context.Context,
	context.CancelFunc,
) {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithTimeout(
		context.WithoutCancel(
			ctx,
		),
		assistantRuntimeSettlementTimeout,
	)
}

// assistantRuntimeTurnAlreadyClosed 判断主轮次是否已经由成功事务关闭。
func assistantRuntimeTurnAlreadyClosed(
	err error,
) bool {
	return errors.Is(
		err,
		repository.ErrAssistantRuntimeTurnAlreadyFinalized,
	) ||
		errors.Is(
			err,
			repository.ErrAssistantRuntimeTurnNotClaimed,
		)
}

// assistantRuntimeChatFailureCode 将内部错误收敛为稳定流水错误码。
func assistantRuntimeChatFailureCode(
	err error,
) string {
	switch {
	case errors.Is(
		err,
		context.Canceled,
	):
		return "client_cancelled"

	case errors.Is(
		err,
		context.DeadlineExceeded,
	):
		return "runtime_timeout"

	default:
		return "ai_stream_failed"
	}
}

// assistantRuntimeLatencyToInt 安全转换AI毫秒耗时。
func assistantRuntimeLatencyToInt(
	value int64,
) int {
	if value <= 0 {
		return 0
	}

	maximum := int64(
		^uint(0) >> 1,
	)

	if value > maximum {
		return int(
			maximum,
		)
	}

	return int(value)
}
