package integration

// assistant_runtime_http_stream_success_assertions.go
//
// 验证成功SSE完成后的真实数据库结果：
//   - 会话增加一个成功轮次并释放active_turn；
//   - 正式历史只包含学生消息和过滤后的助手正文；
//   - usage保存真实模型和输入/输出Token；
//   - 个人积分只扣一次并写一条消费流水。

import (
	"context"
	"encoding/json"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
)

// assertAssistantRuntimeSuccessfulStreamStored 验证成功流式结算。
func assertAssistantRuntimeSuccessfulStreamStored(
	t *testing.T,
	fixture *assistantRuntimeHTTPStreamFixture,
	turnID string,
	expectedVisible string,
) {
	t.Helper()

	var (
		turnCount            int
		activeTurnCleared    bool
		activeStartedCleared bool
		messageCount         int
		messagesJSON         string
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			turn_count,
			active_turn_id IS NULL,
			active_turn_started_at IS NULL,
			jsonb_array_length(messages_json),
			messages_json::text
		FROM assistant_runtime_sessions
		WHERE id = $1
		`,
		fixture.Session.SessionID,
	).Scan(
		&turnCount,
		&activeTurnCleared,
		&activeStartedCleared,
		&messageCount,
		&messagesJSON,
	); err != nil {
		t.Fatalf(
			"读取成功流式会话失败: %v",
			err,
		)
	}

	if turnCount != 1 ||
		!activeTurnCleared ||
		!activeStartedCleared ||
		messageCount != 2 {
		t.Fatalf(
			"成功流式会话状态错误: turn=%d active=%t started=%t messages=%d",
			turnCount,
			activeTurnCleared,
			activeStartedCleared,
			messageCount,
		)
	}

	var messages []models.AssistantRuntimeMessage

	if err := json.Unmarshal(
		[]byte(messagesJSON),
		&messages,
	); err != nil {
		t.Fatalf(
			"解析成功流式正式消息失败: %v raw=%s",
			err,
			messagesJSON,
		)
	}

	if len(messages) != 2 ||
		messages[0].Role !=
			models.AssistantRuntimeMessageRoleStudent ||
		messages[1].Role !=
			models.AssistantRuntimeMessageRoleAssistant ||
		messages[1].Content !=
			expectedVisible {
		t.Fatalf(
			"成功流式正式消息错误: %+v",
			messages,
		)
	}

	var (
		usageCount       int
		usageStatus      string
		usageModel       string
		inputTokens      int
		outputTokens     int
		creditsUsed      float64
		consumptionCount int
		balance          float64
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			(
				SELECT COUNT(*)::integer
				FROM assistant_runtime_usage
				WHERE turn_id = $1
			),
			(
				SELECT COALESCE(MAX(status), '')
				FROM assistant_runtime_usage
				WHERE turn_id = $1
			),
			(
				SELECT COALESCE(MAX(model_name), '')
				FROM assistant_runtime_usage
				WHERE turn_id = $1
			),
			(
				SELECT COALESCE(MAX(input_tokens), 0)
				FROM assistant_runtime_usage
				WHERE turn_id = $1
			),
			(
				SELECT COALESCE(MAX(output_tokens), 0)
				FROM assistant_runtime_usage
				WHERE turn_id = $1
			),
			(
				SELECT COALESCE(SUM(credits_used), 0)
				FROM assistant_runtime_usage
				WHERE turn_id = $1
			),
			(
				SELECT COUNT(*)::integer
				FROM token_consumption_logs
				WHERE memo = $2
			),
			ta.balance
		FROM token_accounts AS ta
		WHERE ta.id = $3
		`,
		turnID,
		"assistant_runtime_turn:"+
			turnID,
		fixture.Fixture.TokenAccountID,
	).Scan(
		&usageCount,
		&usageStatus,
		&usageModel,
		&inputTokens,
		&outputTokens,
		&creditsUsed,
		&consumptionCount,
		&balance,
	); err != nil {
		t.Fatalf(
			"读取成功流式结算失败: %v",
			err,
		)
	}

	if usageCount != 1 ||
		usageStatus != "succeeded" ||
		usageModel != "qwen-max" ||
		inputTokens != 20 ||
		outputTokens != 5 ||
		creditsUsed <= 0 ||
		consumptionCount != 1 ||
		balance >= 1000 {
		t.Fatalf(
			"成功流式结算错误: usage=%d status=%s model=%s input=%d output=%d credits=%.8f logs=%d balance=%.8f",
			usageCount,
			usageStatus,
			usageModel,
			inputTokens,
			outputTokens,
			creditsUsed,
			consumptionCount,
			balance,
		)
	}
}
