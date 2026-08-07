package services

// assistant_runtime_billing.go
//
// 本文件连接短时运行授权、数据库每日额度、AI追踪身份、精确积分计算
// 和成功或失败的最终原子结算。
//
// 调用顺序：
//   1. AuthorizeAndClaimTurn；
//   2. 使用返回的TraceContext执行AI调用；
//   3. 必须调用CompleteSuccess或CompleteFailure之一。
//
// TraceContext中的UserID固定为部署创建者，SchoolID固定为发布学校。
// 公开运行场景会跳过现有全局异步积分钩子，避免重复扣费。
//
// 本文件不接受匿名请求提交的owner_user_id、school_id、courseware_id或page_id。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrAssistantRuntimeBillingContextInvalid = errors.New(
		"教学智能体匿名计费上下文无效",
	)

	ErrAssistantRuntimeBillingResultInvalid = errors.New(
		"教学智能体匿名计费结果无效",
	)
)

// AssistantRuntimeBillingContext 是单轮AI调用的后端内部上下文。
//
// 本结构包含部署和版本敏感快照，不得直接序列化返回浏览器。
type AssistantRuntimeBillingContext struct {
	Authorization *AssistantRuntimeAuthorization
	Claim         *models.AssistantRuntimeTurnClaim
	TraceContext  *ai.TraceContext
}

// AssistantRuntimeBillingService 是匿名运行计费桥。
type AssistantRuntimeBillingService struct {
	sessionService      *AssistantRuntimeSessionService
	creditPolicyService *CreditPolicyService
}

// NewAssistantRuntimeBillingService 创建匿名计费桥。
func NewAssistantRuntimeBillingService(
	sessionService *AssistantRuntimeSessionService,
	creditPolicyService *CreditPolicyService,
) *AssistantRuntimeBillingService {
	return &AssistantRuntimeBillingService{
		sessionService:      sessionService,
		creditPolicyService: creditPolicyService,
	}
}

// AuthorizeAndClaimTurn 验证运行令牌并原子领取每日额度和唯一主轮次。
func (s *AssistantRuntimeBillingService) AuthorizeAndClaimTurn(
	ctx context.Context,
	tokenString string,
	sessionID string,
) (
	*AssistantRuntimeBillingContext,
	error,
) {
	if s == nil ||
		s.sessionService == nil {
		return nil,
			ErrAssistantRuntimeBillingContextInvalid
	}

	authorization, err :=
		s.sessionService.ValidateRuntimeToken(
			ctx,
			tokenString,
			sessionID,
		)
	if err != nil {
		return nil, err
	}

	if err :=
		validateAssistantRuntimeBillingAuthorization(
			authorization,
		); err != nil {
		return nil, err
	}

	// TraceContext先于主轮次领取完成构建。
	// 这样即使内部身份装配异常，也不会留下未释放的active_turn_id。
	traceContext, err :=
		buildAssistantRuntimeBillingTrace(
			authorization,
		)
	if err != nil {
		return nil, err
	}

	turnID, err :=
		generateAssistantRuntimeSessionID()
	if err != nil {
		return nil, err
	}

	claim, err :=
		repository.ClaimAssistantRuntimeTurn(
			ctx,
			authorization.Session.ID,
			authorization.Deployment.ID,
			authorization.Session.DeploymentVersion,
			turnID,
		)
	if err != nil {
		return nil, err
	}

	return &AssistantRuntimeBillingContext{
		Authorization: authorization,
		Claim:         claim,
		TraceContext:  traceContext,
	}, nil
}

// CompleteSuccess 计算实际积分并执行成功原子结算。
func (s *AssistantRuntimeBillingService) CompleteSuccess(
	ctx context.Context,
	billingContext *AssistantRuntimeBillingContext,
	completion *models.AssistantRuntimeTurnCompletion,
) (
	*models.AssistantRuntimeSession,
	error,
) {
	if err :=
		validateAssistantRuntimeBillingContext(
			billingContext,
		); err != nil {
		return nil, err
	}

	normalized, err :=
		normalizeAssistantRuntimeSuccessCompletion(
			billingContext,
			completion,
		)
	if err != nil {
		return nil, err
	}

	schoolID :=
		strings.TrimSpace(
			billingContext.Authorization.
				Deployment.SchoolID,
		)

	totalTokens :=
		normalized.InputTokens +
			normalized.OutputTokens

	calculation :=
		s.resolveCreditPolicyService().
			CalculateCredits(
				ctx,
				normalized.ModelName,
				normalized.InputTokens,
				normalized.OutputTokens,
				totalTokens,
				&schoolID,
				int64(normalized.LatencyMs),
			)
	if calculation == nil ||
		calculation.CreditsConsumed < 0 {
		return nil,
			ErrAssistantRuntimeBillingResultInvalid
	}

	normalized.CreditsUsed =
		calculation.CreditsConsumed
	normalized.Provider =
		strings.TrimSpace(
			calculation.Provider,
		)

	authorization :=
		billingContext.Authorization

	settlement, err :=
		repository.CompleteAssistantRuntimeTurnSuccess(
			ctx,
			&repository.AssistantRuntimeSuccessSettlementInput{
				TurnID:       normalized.TurnID,
				SessionID:    normalized.SessionID,
				DeploymentID: authorization.Deployment.ID,
				DeploymentVersion: authorization.Session.
					DeploymentVersion,
				OwnerUserID: authorization.Deployment.
					OwnerUserID,
				SchoolID: authorization.Deployment.SchoolID,
				CoursewareID: authorization.Deployment.
					CoursewareID,
				PageID:           authorization.Deployment.PageID,
				SceneCode:        ai.SceneCoursewareAssistantRuntime,
				StudentMessage:   normalized.StudentMessage,
				AssistantMessage: normalized.AssistantMessage,
				InputChars:       normalized.InputChars,
				OutputChars:      normalized.OutputChars,
				InputTokens:      normalized.InputTokens,
				OutputTokens:     normalized.OutputTokens,
				ModelName:        normalized.ModelName,
				Provider:         normalized.Provider,
				LatencyMs:        normalized.LatencyMs,
				Calculation:      calculation,
			},
		)
	if err != nil {
		return nil, err
	}

	if calculation.CreditsConsumed > 0 &&
		settlement != nil &&
		settlement.Account != nil {
		// 自动补足是结算提交后的best-effort旁路。
		// 自动补失败不会回滚已发生的AI调用和精确消费流水。
		TryAutoRefill(
			ctx,
			settlement.Account,
			settlement.BalanceAfter,
		)
	}

	if settlement == nil ||
		settlement.Session == nil {
		return nil,
			ErrAssistantRuntimeBillingResultInvalid
	}

	return settlement.Session, nil
}

// CompleteFailure 写失败流水并释放主轮次。
func (s *AssistantRuntimeBillingService) CompleteFailure(
	ctx context.Context,
	billingContext *AssistantRuntimeBillingContext,
	failure *models.AssistantRuntimeTurnFailure,
) (
	*models.AssistantRuntimeSession,
	error,
) {
	if err :=
		validateAssistantRuntimeBillingContext(
			billingContext,
		); err != nil {
		return nil, err
	}

	normalized, err :=
		normalizeAssistantRuntimeFailure(
			billingContext,
			failure,
		)
	if err != nil {
		return nil, err
	}

	authorization :=
		billingContext.Authorization

	return repository.CompleteAssistantRuntimeTurnFailure(
		ctx,
		&repository.AssistantRuntimeFailureSettlementInput{
			TurnID:       normalized.TurnID,
			SessionID:    normalized.SessionID,
			DeploymentID: authorization.Deployment.ID,
			DeploymentVersion: authorization.Session.
				DeploymentVersion,
			OwnerUserID: authorization.Deployment.
				OwnerUserID,
			SchoolID: authorization.Deployment.SchoolID,
			CoursewareID: authorization.Deployment.
				CoursewareID,
			PageID:      authorization.Deployment.PageID,
			SceneCode:   ai.SceneCoursewareAssistantRuntime,
			SessionKind: authorization.Session.SessionKind,
			InputChars:  normalized.InputChars,
			ErrorCode:   normalized.ErrorCode,
			ModelName:   normalized.ModelName,
			Provider:    normalized.Provider,
			LatencyMs:   normalized.LatencyMs,
		},
	)
}
