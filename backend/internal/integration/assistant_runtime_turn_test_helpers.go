package integration

// assistant_runtime_turn_test_helpers.go
//
// 为主轮次、每日额度和锁顺序测试提供确定性辅助：
//   - 生成合法且互不冲突的UUID；
//   - 为每个会话生成唯一64字符JTI哈希；
//   - 创建独立运行会话；
//   - 构造成功和失败结算输入。
//
// 本文件不直接运行测试，也不调用真实AI。
//
// 所有夹具方法统一接收正式的*testing.T，避免通过自定义接口
// 适配具体测试类型而造成编译期类型不匹配。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// assistantRuntimeIntegrationUUID 生成测试专用RFC 4122格式UUID。
func assistantRuntimeIntegrationUUID(
	namespace uint32,
	value uint64,
) string {
	return fmt.Sprintf(
		"%08x-0000-4000-8000-%012x",
		namespace,
		value,
	)
}

// assistantRuntimeIntegrationHash 生成唯一64字符测试哈希。
func assistantRuntimeIntegrationHash(
	value string,
) string {
	sum := sha256.Sum256(
		[]byte(value),
	)

	return hex.EncodeToString(
		sum[:],
	)
}

// createAssistantRuntimeIsolatedSession 创建唯一测试运行会话。
func createAssistantRuntimeIsolatedSession(
	t *testing.T,
	fixture *AssistantRuntimeFixture,
	deployment *models.AssistantDeployment,
	namespace uint32,
	value uint64,
) *models.AssistantRuntimeSession {
	t.Helper()

	if fixture == nil {
		t.Fatal(
			"创建独立运行会话失败：测试夹具为nil",
		)
	}

	if deployment == nil {
		t.Fatal(
			"创建独立运行会话失败：部署记录为nil",
		)
	}

	sessionID := assistantRuntimeIntegrationUUID(
		namespace,
		value,
	)

	jtiHash := assistantRuntimeIntegrationHash(
		"jti:" + sessionID,
	)

	// CreateRuntimeSession内部执行完整数据库错误检查。
	return fixture.CreateRuntimeSession(
		t,
		deployment,
		sessionID,
		jtiHash,
	)
}

// assistantRuntimeFailureSettlementInput 构造失败结算输入。
func assistantRuntimeFailureSettlementInput(
	fixture *AssistantRuntimeFixture,
	deployment *models.AssistantDeployment,
	session *models.AssistantRuntimeSession,
	turnID string,
) *repository.AssistantRuntimeFailureSettlementInput {
	return &repository.AssistantRuntimeFailureSettlementInput{
		TurnID:
			turnID,
		SessionID:
			session.ID,
		DeploymentID:
			deployment.ID,
		DeploymentVersion:
			deployment.CurrentVersion,
		OwnerUserID:
			SeedOperatorID,
		SchoolID:
			fixture.SchoolID,
		CoursewareID:
			fixture.CoursewareID,
		PageID:
			fixture.PageID,
		SceneCode:
			"courseware_assistant_runtime",
		SessionKind:
			models.AssistantRuntimeSessionKindExternal,
		InputChars:
			12,
		ErrorCode:
			"integration_failure",
		ModelName:
			"integration-test-model",
		Provider:
			"integration",
		LatencyMs:
			20,
	}
}

// assistantRuntimeSuccessSettlementInput 构造成功结算输入。
func assistantRuntimeSuccessSettlementInput(
	fixture *AssistantRuntimeFixture,
	deployment *models.AssistantDeployment,
	session *models.AssistantRuntimeSession,
	turnID string,
	credits float64,
) *repository.AssistantRuntimeSuccessSettlementInput {
	messageTime := time.Now().UTC()

	return &repository.AssistantRuntimeSuccessSettlementInput{
		TurnID:
			turnID,
		SessionID:
			session.ID,
		DeploymentID:
			deployment.ID,
		DeploymentVersion:
			deployment.CurrentVersion,
		OwnerUserID:
			SeedOperatorID,
		SchoolID:
			fixture.SchoolID,
		CoursewareID:
			fixture.CoursewareID,
		PageID:
			fixture.PageID,
		SceneCode:
			"courseware_assistant_runtime",
		StudentMessage:
			models.AssistantRuntimeMessage{
				Role:
					models.AssistantRuntimeMessageRoleStudent,
				Content:
					"我先尝试观察图形。",
				CreatedAt:
					&messageTime,
			},
		AssistantMessage:
			models.AssistantRuntimeMessage{
				Role:
					models.AssistantRuntimeMessageRoleAssistant,
				Content:
					"你观察到了哪些对应关系？",
				CreatedAt:
					&messageTime,
			},
		InputChars:
			12,
		OutputChars:
			14,
		InputTokens:
			20,
		OutputTokens:
			18,
		ModelName:
			"integration-test-model",
		Provider:
			"integration",
		LatencyMs:
			30,
		Calculation:
			&models.CreditCalculation{
				CostUSD:
					0,
				ExchangeRate:
					7,
				Multiplier:
					1,
				CreditsConsumed:
					credits,
			},
	}
}
