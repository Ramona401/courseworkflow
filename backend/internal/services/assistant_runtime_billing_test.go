package services

// assistant_runtime_billing_test.go
//
// 本测试验证匿名计费身份绑定、全局积分钩子隔离、成功消息规范化、
// 失败码收敛和防串会话校验。
//
// 不连接数据库、不调用AI，也不实际扣积分。

import (
	"errors"
	"strings"
	"testing"

	"tedna/internal/ai"
	"tedna/internal/models"
)

// TestAssistantRuntimeBillingBuildsTeacherTrace 验证教师和学校固化身份。
func TestAssistantRuntimeBillingBuildsTeacherTrace(
	t *testing.T,
) {
	billingContext :=
		buildAssistantRuntimeBillingTestContext()

	traceContext, err :=
		buildAssistantRuntimeBillingTrace(
			billingContext.Authorization,
		)
	if err != nil {
		t.Fatalf(
			"构造匿名计费TraceContext失败: %v",
			err,
		)
	}

	if traceContext.UserID == nil ||
		*traceContext.UserID !=
			billingContext.Authorization.
				Deployment.OwnerUserID {
		t.Fatal(
			"匿名调用UserID没有绑定部署创建者",
		)
	}

	if traceContext.SchoolID == nil ||
		*traceContext.SchoolID !=
			billingContext.Authorization.
				Deployment.SchoolID {
		t.Fatal(
			"匿名调用SchoolID没有绑定发布学校",
		)
	}

	if !ai.IsExternallyBilledTrace(
		traceContext,
	) {
		t.Fatal(
			"运行时Trace必须跳过普通全局积分钩子",
		)
	}
}

// TestAssistantRuntimeBillingRejectsEmptyIdentity 验证空教师或学校被拒绝。
func TestAssistantRuntimeBillingRejectsEmptyIdentity(
	t *testing.T,
) {
	contextValue :=
		buildAssistantRuntimeBillingTestContext()

	contextValue.Authorization.
		Deployment.OwnerUserID = ""

	if !errors.Is(
		validateAssistantRuntimeBillingAuthorization(
			contextValue.Authorization,
		),
		ErrAssistantRuntimeBillingContextInvalid,
	) {
		t.Fatal(
			"空部署创建者必须被匿名计费桥拒绝",
		)
	}

	contextValue =
		buildAssistantRuntimeBillingTestContext()

	contextValue.Authorization.
		Deployment.SchoolID = ""

	if !errors.Is(
		validateAssistantRuntimeBillingAuthorization(
			contextValue.Authorization,
		),
		ErrAssistantRuntimeBillingContextInvalid,
	) {
		t.Fatal(
			"空发布学校必须被匿名计费桥拒绝",
		)
	}
}

// TestAssistantRuntimeBillingRejectsMismatchedClaim 验证不能串会话。
func TestAssistantRuntimeBillingRejectsMismatchedClaim(
	t *testing.T,
) {
	contextValue :=
		buildAssistantRuntimeBillingTestContext()

	contextValue.Claim.SessionID =
		"99999999-9999-4999-8999-999999999999"

	if !errors.Is(
		validateAssistantRuntimeBillingContext(
			contextValue,
		),
		ErrAssistantRuntimeBillingContextInvalid,
	) {
		t.Fatal(
			"主轮次和令牌会话不一致时必须拒绝",
		)
	}
}

// TestAssistantRuntimeBillingRejectsVersionMismatch 验证部署版本变化被拒绝。
func TestAssistantRuntimeBillingRejectsVersionMismatch(
	t *testing.T,
) {
	contextValue :=
		buildAssistantRuntimeBillingTestContext()

	contextValue.Authorization.
		Deployment.CurrentVersion = 4

	if !errors.Is(
		validateAssistantRuntimeBillingAuthorization(
			contextValue.Authorization,
		),
		ErrAssistantRuntimeBillingContextInvalid,
	) {
		t.Fatal(
			"部署当前版本与会话版本不一致时必须拒绝",
		)
	}
}

// TestAssistantRuntimeBillingNormalizesCompletion 验证成功消息角色和ID不可伪造。
func TestAssistantRuntimeBillingNormalizesCompletion(
	t *testing.T,
) {
	contextValue :=
		buildAssistantRuntimeBillingTestContext()

	normalized, err :=
		normalizeAssistantRuntimeSuccessCompletion(
			contextValue,
			&models.AssistantRuntimeTurnCompletion{
				TurnID:    "伪造turn",
				SessionID: "伪造session",
				StudentMessage: models.AssistantRuntimeMessage{
					Role:    "system",
					Content: "我先尝试把两个三角形拼起来。",
				},
				AssistantMessage: models.AssistantRuntimeMessage{
					Role:    "tool",
					Content: "你拼成了什么熟悉的图形？",
				},
				InputChars:   15,
				OutputChars:  14,
				InputTokens:  20,
				OutputTokens: 18,
				ModelName:    "test-model",
				LatencyMs:    120,
			},
		)
	if err != nil {
		t.Fatalf(
			"合法成功结果规范化失败: %v",
			err,
		)
	}

	if normalized.TurnID !=
		contextValue.Claim.TurnID ||
		normalized.SessionID !=
			contextValue.Claim.SessionID {
		t.Fatal(
			"成功结算必须覆盖调用方伪造的会话和轮次ID",
		)
	}

	if normalized.StudentMessage.Role !=
		models.AssistantRuntimeMessageRoleStudent ||
		normalized.AssistantMessage.Role !=
			models.AssistantRuntimeMessageRoleAssistant {
		t.Fatal(
			"成功结算必须固定正式可见消息角色",
		)
	}

	if normalized.StudentMessage.CreatedAt == nil ||
		normalized.AssistantMessage.CreatedAt == nil {
		t.Fatal(
			"正式可见消息必须补齐时间戳",
		)
	}
}

// TestAssistantRuntimeBillingNormalizesFailureCode 验证失败码安全收敛。
func TestAssistantRuntimeBillingNormalizesFailureCode(
	t *testing.T,
) {
	actual :=
		normalizeAssistantRuntimeErrorCode(
			"  Upstream Timeout / HTTP 504 !!!  ",
		)

	if actual !=
		"upstream_timeout___http_504" {
		t.Fatalf(
			"失败码规范化结果错误: %s",
			actual,
		)
	}

	longCode :=
		normalizeAssistantRuntimeErrorCode(
			strings.Repeat(
				"a",
				100,
			),
		)
	if len(longCode) != 64 {
		t.Fatalf(
			"失败码没有收敛到64字符: %d",
			len(longCode),
		)
	}

	if normalizeAssistantRuntimeErrorCode(
		"",
	) != "runtime_failed" {
		t.Fatal(
			"空失败码必须使用稳定默认值",
		)
	}
}

// buildAssistantRuntimeBillingTestContext 构造纯内存合法上下文。
func buildAssistantRuntimeBillingTestContext() *AssistantRuntimeBillingContext {
	deploymentID :=
		"11111111-1111-4111-8111-111111111111"
	sessionID :=
		"22222222-2222-4222-8222-222222222222"
	ownerID :=
		"33333333-3333-4333-8333-333333333333"
	schoolID :=
		"44444444-4444-4444-8444-444444444444"

	authorization :=
		&AssistantRuntimeAuthorization{
			Session: &models.AssistantRuntimeSession{
				ID:                sessionID,
				DeploymentID:      deploymentID,
				DeploymentVersion: 3,
				SessionKind:       models.AssistantRuntimeSessionKindExternal,
				Status:            models.AssistantRuntimeSessionStatusActive,
			},
			Deployment: &models.AssistantDeployment{
				ID:             deploymentID,
				OwnerUserID:    ownerID,
				SchoolID:       schoolID,
				CoursewareID:   "55555555-5555-4555-8555-555555555555",
				PageID:         "66666666-6666-4666-8666-666666666666",
				CurrentVersion: 3,
				Status:         models.AssistantDeploymentStatusActive,
			},
			Version: &models.AssistantDeploymentVersion{
				DeploymentID: deploymentID,
				Version:      3,
			},
		}

	traceContext, _ :=
		buildAssistantRuntimeBillingTrace(
			authorization,
		)

	return &AssistantRuntimeBillingContext{
		Authorization: authorization,
		Claim: &models.AssistantRuntimeTurnClaim{
			TurnID:            "77777777-7777-4777-8777-777777777777",
			SessionID:         sessionID,
			DeploymentID:      deploymentID,
			DeploymentVersion: 3,
			TurnCount:         0,
			MaxTurns:          10,
			Messages:          []models.AssistantRuntimeMessage{},
		},
		TraceContext: traceContext,
	}
}
