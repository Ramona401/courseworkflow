package services

// assistant_runtime_billing_validation.go
//
// 本文件负责匿名计费上下文校验、教师和学校TraceContext装配、
// 成功结果规范化、失败码收敛以及积分策略依赖解析。
//
// 所有身份字段均来自已通过短时令牌和数据库实时校验的部署记录，
// 不读取匿名请求提交的owner_id、school_id、courseware_id或page_id。

import (
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
)

// buildAssistantRuntimeBillingTrace 固定教师和学校计费身份。
func buildAssistantRuntimeBillingTrace(
	authorization *AssistantRuntimeAuthorization,
) (
	*ai.TraceContext,
	error,
) {
	if err :=
		validateAssistantRuntimeBillingAuthorization(
			authorization,
		); err != nil {
		return nil, err
	}

	userID :=
		strings.TrimSpace(
			authorization.Deployment.OwnerUserID,
		)
	schoolID :=
		strings.TrimSpace(
			authorization.Deployment.SchoolID,
		)

	return &ai.TraceContext{
		SceneCode: ai.SceneCoursewareAssistantRuntime,
		UserID:    &userID,
		SchoolID:  &schoolID,
	}, nil
}

// validateAssistantRuntimeBillingAuthorization 校验部署计费快照。
func validateAssistantRuntimeBillingAuthorization(
	authorization *AssistantRuntimeAuthorization,
) error {
	if authorization == nil ||
		authorization.Session == nil ||
		authorization.Deployment == nil ||
		authorization.Version == nil ||
		strings.TrimSpace(
			authorization.Session.ID,
		) == "" ||
		authorization.Session.Status !=
			models.AssistantRuntimeSessionStatusActive ||
		strings.TrimSpace(
			authorization.Deployment.ID,
		) == "" ||
		strings.TrimSpace(
			authorization.Deployment.OwnerUserID,
		) == "" ||
		strings.TrimSpace(
			authorization.Deployment.SchoolID,
		) == "" ||
		strings.TrimSpace(
			authorization.Deployment.CoursewareID,
		) == "" ||
		strings.TrimSpace(
			authorization.Deployment.PageID,
		) == "" ||
		authorization.Session.DeploymentID !=
			authorization.Deployment.ID ||
		authorization.Session.DeploymentVersion <= 0 ||
		authorization.Session.DeploymentVersion !=
			authorization.Deployment.CurrentVersion ||
		authorization.Session.DeploymentVersion !=
			authorization.Version.Version ||
		strings.TrimSpace(
			authorization.Version.DeploymentID,
		) !=
			strings.TrimSpace(
				authorization.Deployment.ID,
			) {
		return ErrAssistantRuntimeBillingContextInvalid
	}

	return nil
}

// validateAssistantRuntimeBillingContext 校验领取结果未串会话或部署。
func validateAssistantRuntimeBillingContext(
	billingContext *AssistantRuntimeBillingContext,
) error {
	if billingContext == nil ||
		billingContext.Claim == nil ||
		billingContext.TraceContext == nil {
		return ErrAssistantRuntimeBillingContextInvalid
	}

	if err :=
		validateAssistantRuntimeBillingAuthorization(
			billingContext.Authorization,
		); err != nil {
		return err
	}

	if billingContext.Claim.SessionID !=
		billingContext.Authorization.Session.ID ||
		billingContext.Claim.DeploymentID !=
			billingContext.Authorization.Deployment.ID ||
		billingContext.Claim.DeploymentVersion !=
			billingContext.Authorization.Session.
				DeploymentVersion ||
		strings.TrimSpace(
			billingContext.Claim.TurnID,
		) == "" ||
		billingContext.TraceContext.UserID == nil ||
		billingContext.TraceContext.SchoolID == nil ||
		strings.TrimSpace(
			*billingContext.TraceContext.UserID,
		) !=
			strings.TrimSpace(
				billingContext.Authorization.
					Deployment.OwnerUserID,
			) ||
		strings.TrimSpace(
			*billingContext.TraceContext.SchoolID,
		) !=
			strings.TrimSpace(
				billingContext.Authorization.
					Deployment.SchoolID,
			) ||
		!ai.IsExternallyBilledTrace(
			billingContext.TraceContext,
		) {
		return ErrAssistantRuntimeBillingContextInvalid
	}

	return nil
}

// normalizeAssistantRuntimeSuccessCompletion 规范成功消息和计量字段。
//
// 调用方提交的turn_id、session_id和消息角色均被服务端覆盖。
func normalizeAssistantRuntimeSuccessCompletion(
	billingContext *AssistantRuntimeBillingContext,
	completion *models.AssistantRuntimeTurnCompletion,
) (
	*models.AssistantRuntimeTurnCompletion,
	error,
) {
	if completion == nil ||
		strings.TrimSpace(
			completion.StudentMessage.Content,
		) == "" ||
		strings.TrimSpace(
			completion.AssistantMessage.Content,
		) == "" ||
		completion.InputChars < 0 ||
		completion.OutputChars < 0 ||
		completion.InputTokens < 0 ||
		completion.OutputTokens < 0 ||
		completion.LatencyMs < 0 ||
		strings.TrimSpace(
			completion.ModelName,
		) == "" {
		return nil,
			ErrAssistantRuntimeBillingResultInvalid
	}

	normalized := *completion
	now := time.Now().UTC()

	normalized.TurnID =
		billingContext.Claim.TurnID
	normalized.SessionID =
		billingContext.Claim.SessionID
	normalized.ModelName =
		strings.TrimSpace(
			normalized.ModelName,
		)
	normalized.Provider =
		strings.TrimSpace(
			normalized.Provider,
		)

	normalized.StudentMessage.Role =
		models.AssistantRuntimeMessageRoleStudent
	normalized.AssistantMessage.Role =
		models.AssistantRuntimeMessageRoleAssistant

	if normalized.StudentMessage.CreatedAt == nil {
		studentAt := now
		normalized.StudentMessage.CreatedAt =
			&studentAt
	}
	if normalized.AssistantMessage.CreatedAt == nil {
		assistantAt := now
		normalized.AssistantMessage.CreatedAt =
			&assistantAt
	}

	return &normalized, nil
}

// normalizeAssistantRuntimeFailure 规范失败码并绑定当前turn_id。
func normalizeAssistantRuntimeFailure(
	billingContext *AssistantRuntimeBillingContext,
	failure *models.AssistantRuntimeTurnFailure,
) (
	*models.AssistantRuntimeTurnFailure,
	error,
) {
	if failure == nil ||
		failure.InputChars < 0 ||
		failure.LatencyMs < 0 {
		return nil,
			ErrAssistantRuntimeBillingResultInvalid
	}

	normalized := *failure
	normalized.TurnID =
		billingContext.Claim.TurnID
	normalized.SessionID =
		billingContext.Claim.SessionID
	normalized.ErrorCode =
		normalizeAssistantRuntimeErrorCode(
			normalized.ErrorCode,
		)
	normalized.ModelName =
		strings.TrimSpace(
			normalized.ModelName,
		)
	normalized.Provider =
		strings.TrimSpace(
			normalized.Provider,
		)

	return &normalized, nil
}

// normalizeAssistantRuntimeErrorCode 生成最长64字符的稳定安全错误码。
func normalizeAssistantRuntimeErrorCode(
	raw string,
) string {
	raw = strings.ToLower(
		strings.TrimSpace(raw),
	)
	if raw == "" {
		return "runtime_failed"
	}

	builder := strings.Builder{}

	for _, value := range raw {
		if builder.Len() >= 64 {
			break
		}

		switch {
		case value >= 'a' && value <= 'z',
			value >= '0' && value <= '9',
			value == '_',
			value == '-',
			value == ':':
			builder.WriteRune(value)

		default:
			builder.WriteByte('_')
		}
	}

	result :=
		strings.Trim(
			builder.String(),
			"_",
		)
	if result == "" {
		return "runtime_failed"
	}

	if len(result) > 64 {
		result = result[:64]
	}

	return result
}

// resolveCreditPolicyService 返回精确积分计算服务。
func (s *AssistantRuntimeBillingService) resolveCreditPolicyService() *CreditPolicyService {
	if s != nil &&
		s.creditPolicyService != nil {
		return s.creditPolicyService
	}

	return NewCreditPolicyService()
}
