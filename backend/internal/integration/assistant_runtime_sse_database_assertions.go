package integration

// assistant_runtime_sse_database_assertions.go
//
// 提供流式失败场景的真实数据库等待与安全断言：
//   - 等待浏览器取消后的独立失败事务提交；
//   - 验证失败usage唯一且不扣积分；
//   - 验证部分输出没有写入正式消息；
//   - 验证active_turn两个字段同时释放。

import (
	"context"
	"testing"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// waitAssistantRuntimeFailureSettlement 等待独立失败结算事务。
func waitAssistantRuntimeFailureSettlement(
	t *testing.T,
	sessionID string,
	expectedErrorCode string,
) {
	t.Helper()

	deadline :=
		time.Now().Add(
			5 * time.Second,
		)

	for {
		var matchingCount int

		err := database.DB.QueryRow(
			context.Background(),
			`
			SELECT COUNT(*)::integer
			FROM assistant_runtime_usage
			WHERE runtime_session_id = $1
			  AND status = 'failed'
			  AND error_code = $2
			`,
			sessionID,
			expectedErrorCode,
		).Scan(
			&matchingCount,
		)
		if err != nil {
			t.Fatalf(
				"等待失败结算时查询usage失败: %v",
				err,
			)
		}

		if matchingCount == 1 {
			break
		}

		if time.Now().After(
			deadline,
		) {
			t.Fatalf(
				"等待失败结算超时: session=%s code=%s count=%d",
				sessionID,
				expectedErrorCode,
				matchingCount,
			)
		}

		time.Sleep(
			25 * time.Millisecond,
		)
	}

	assertAssistantRuntimeFailedStreamStored(
		t,
		sessionID,
		expectedErrorCode,
	)
}

// assertAssistantRuntimeFailedStreamStored 验证流式失败不污染正式历史。
func assertAssistantRuntimeFailedStreamStored(
	t *testing.T,
	sessionID string,
	expectedErrorCode string,
) {
	t.Helper()

	var (
		sessionStatus        string
		turnCount            int
		activeTurnCleared    bool
		activeStartedCleared bool
		messageCount         int
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			status,
			turn_count,
			active_turn_id IS NULL,
			active_turn_started_at IS NULL,
			jsonb_array_length(messages_json)
		FROM assistant_runtime_sessions
		WHERE id = $1
		`,
		sessionID,
	).Scan(
		&sessionStatus,
		&turnCount,
		&activeTurnCleared,
		&activeStartedCleared,
		&messageCount,
	); err != nil {
		t.Fatalf(
			"读取流式失败会话失败: %v",
			err,
		)
	}

	if sessionStatus !=
		models.AssistantRuntimeSessionStatusActive ||
		turnCount != 0 ||
		!activeTurnCleared ||
		!activeStartedCleared ||
		messageCount != 0 {
		t.Fatalf(
			"流式失败会话状态错误: status=%s turn=%d active=%t started=%t messages=%d",
			sessionStatus,
			turnCount,
			activeTurnCleared,
			activeStartedCleared,
			messageCount,
		)
	}

	var (
		usageCount   int
		failureCount int
		successCount int
		creditsUsed  float64
	)

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			COUNT(*)::integer,
			COUNT(*) FILTER (
				WHERE status = 'failed'
				  AND error_code = $2
			)::integer,
			COUNT(*) FILTER (
				WHERE status = 'succeeded'
			)::integer,
			COALESCE(SUM(credits_used), 0)
		FROM assistant_runtime_usage
		WHERE runtime_session_id = $1
		`,
		sessionID,
		expectedErrorCode,
	).Scan(
		&usageCount,
		&failureCount,
		&successCount,
		&creditsUsed,
	); err != nil {
		t.Fatalf(
			"读取流式失败usage失败: %v",
			err,
		)
	}

	if usageCount != 1 ||
		failureCount != 1 ||
		successCount != 0 ||
		creditsUsed != 0 {
		t.Fatalf(
			"流式失败usage错误: total=%d failed=%d succeeded=%d credits=%.4f",
			usageCount,
			failureCount,
			successCount,
			creditsUsed,
		)
	}

	var logCount int

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT COUNT(*)
		FROM token_consumption_logs
		`,
	).Scan(
		&logCount,
	); err != nil {
		t.Fatalf(
			"读取流式失败消费流水失败: %v",
			err,
		)
	}

	if logCount != 0 {
		t.Fatalf(
			"流式失败错误写入积分消费流水: %d",
			logCount,
		)
	}
}
