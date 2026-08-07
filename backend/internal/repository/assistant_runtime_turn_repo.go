package repository

// assistant_runtime_turn_repo.go
//
// 本文件在AI调用建立前原子领取一个运行主轮次。
//
// 单一事务中的锁定顺序：
//   1. 锁定assistant_deployments行；
//   2. 锁定目标assistant_runtime_sessions行；
//   3. 校验会话状态、版本、有效期、剩余轮数和并发主轮次；
//   4. external会话检查外部学生每日额度；
//   5. teacher_preview会话跳过外部每日额度，但仍执行积分账户检查；
//   6. 锁定并校验部署创建者个人积分账户；
//   7. 写入active_turn_id和active_turn_started_at。
//
// 计费原则：
//
//   - external和teacher_preview均由deployment.owner_user_id个人账户付费；
//   - 教师预览是真实AI调用，因此仍扣教师积分；
//   - daily_call_limit只控制学生和外部渠道，不被教师调试占用；
//   - 会话自己的max_turns对两种会话都生效。
//
// 外部每日额度使用：今日external成功流水数 + 最近20分钟external在途主轮次。
// AI客户端总超时为15分钟，20分钟窗口覆盖正常调用，同时避免异常进程
// 留下的陈旧占位永久占满部署外部额度。
//
// 本文件不调用AI、不扣积分，也不相信请求提供的教师或学校ID。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrAssistantRuntimeDailyQuotaExceeded = errors.New(
		"教学智能体部署今日调用额度已用尽",
	)
	ErrAssistantRuntimeTurnLimitReached = errors.New(
		"教学智能体运行会话轮数已用尽",
	)
	ErrAssistantRuntimeTurnInProgress = errors.New(
		"教学智能体运行会话已有正在执行的主轮次",
	)
	ErrAssistantRuntimeBillingAccountUnavailable = errors.New(
		"教学智能体部署创建者积分账户不可用或余额不足",
	)
	ErrAssistantRuntimeTurnClaimInvalid = errors.New(
		"教学智能体运行主轮次领取参数无效",
	)
	ErrAssistantRuntimeTurnSessionUnavailable = errors.New(
		"教学智能体运行会话当前不可领取主轮次",
	)
	ErrAssistantRuntimeMessagesInvalid = errors.New(
		"教学智能体运行会话消息记录无效",
	)
)

// assistantRuntimeDailyQuotaAllows 是外部每日额度纯判断。
func assistantRuntimeDailyQuotaAllows(
	limit int,
	succeeded int,
	active int,
) bool {
	return limit > 0 &&
		succeeded >= 0 &&
		active >= 0 &&
		succeeded+active < limit
}

// assistantRuntimeBillingAccountAllows 是个人账户前置纯判断。
func assistantRuntimeBillingAccountAllows(
	status string,
	balance float64,
	frozenAmount float64,
	expiresAt *time.Time,
	now time.Time,
) bool {
	if strings.TrimSpace(status) !=
		models.AccountStatusActive {
		return false
	}

	if expiresAt != nil &&
		!expiresAt.After(now) {
		return false
	}

	return balance-frozenAmount > 0
}

// decodeAssistantRuntimeClaimMessages 解码正式可见消息。
func decodeAssistantRuntimeClaimMessages(
	raw string,
) (
	[]models.AssistantRuntimeMessage,
	error,
) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "[]"
	}

	var messages []models.AssistantRuntimeMessage
	if err := json.Unmarshal(
		[]byte(raw),
		&messages,
	); err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrAssistantRuntimeMessagesInvalid,
				err,
			)
	}

	if messages == nil {
		messages =
			[]models.AssistantRuntimeMessage{}
	}

	for _, message := range messages {
		if !models.IsValidAssistantRuntimeMessageRole(
			message.Role,
		) ||
			strings.TrimSpace(
				message.Content,
			) == "" {
			return nil,
				ErrAssistantRuntimeMessagesInvalid
		}
	}

	return messages, nil
}

// ClaimAssistantRuntimeTurn 原子领取会话唯一主轮次。
//
// external会话同时领取部署外部每日额度；
// teacher_preview会话不占外部每日额度，但仍检查个人积分和会话轮数。
func ClaimAssistantRuntimeTurn(
	ctx context.Context,
	sessionID string,
	deploymentID string,
	deploymentVersion int,
	turnID string,
) (
	*models.AssistantRuntimeTurnClaim,
	error,
) {
	sessionID = strings.TrimSpace(sessionID)
	deploymentID = strings.TrimSpace(deploymentID)
	turnID = strings.TrimSpace(turnID)

	if sessionID == "" ||
		deploymentID == "" ||
		deploymentVersion <= 0 ||
		turnID == "" {
		return nil,
			ErrAssistantRuntimeTurnClaimInvalid
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启教学智能体主轮次领取事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		ownerUserID     string
		schoolID        string
		coursewareID    string
		pageID          string
		currentVersion  int
		dailyCallLimit  int
		deploymentState string
		validFrom       time.Time
		validUntil      *time.Time
		databaseNow     time.Time
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			d.owner_user_id::text,
			d.school_id::text,
			d.courseware_id::text,
			d.page_id::text,
			d.current_version,
			d.daily_call_limit,
			d.status,
			d.valid_from,
			d.valid_until,
			NOW()
		FROM assistant_deployments d
		WHERE d.id = $1
		FOR UPDATE`,
		deploymentID,
	).Scan(
		&ownerUserID,
		&schoolID,
		&coursewareID,
		&pageID,
		&currentVersion,
		&dailyCallLimit,
		&deploymentState,
		&validFrom,
		&validUntil,
		&databaseNow,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrAssistantRuntimeTurnSessionUnavailable
		}

		return nil,
			fmt.Errorf(
				"锁定教学智能体部署额度失败: %w",
				err,
			)
	}

	if strings.TrimSpace(deploymentState) !=
		models.AssistantDeploymentStatusActive ||
		currentVersion != deploymentVersion ||
		databaseNow.Before(validFrom) ||
		(validUntil != nil &&
			!validUntil.After(databaseNow)) ||
		strings.TrimSpace(ownerUserID) == "" ||
		strings.TrimSpace(schoolID) == "" ||
		strings.TrimSpace(coursewareID) == "" ||
		strings.TrimSpace(pageID) == "" {
		return nil,
			ErrAssistantRuntimeTurnSessionUnavailable
	}

	var (
		sessionStatus    string
		sessionKind      string
		turnCount        int
		maxTurns         int
		activeTurnID     string
		messagesJSON     string
		sessionExpiresAt time.Time
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			s.status,
			s.session_kind,
			s.turn_count,
			s.max_turns,
			COALESCE(s.active_turn_id::text, ''),
			COALESCE(s.messages_json::text, '[]'),
			s.expires_at
		FROM assistant_runtime_sessions s
		WHERE s.id = $1
		  AND s.deployment_id = $2
		  AND s.deployment_version = $3
		FOR UPDATE`,
		sessionID,
		deploymentID,
		deploymentVersion,
	).Scan(
		&sessionStatus,
		&sessionKind,
		&turnCount,
		&maxTurns,
		&activeTurnID,
		&messagesJSON,
		&sessionExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrAssistantRuntimeTurnSessionUnavailable
		}

		return nil,
			fmt.Errorf(
				"锁定教学智能体运行会话失败: %w",
				err,
			)
	}

	sessionKind = strings.TrimSpace(sessionKind)

	if strings.TrimSpace(sessionStatus) !=
		models.AssistantRuntimeSessionStatusActive ||
		!models.IsValidAssistantRuntimeSessionKind(
			sessionKind,
		) ||
		!sessionExpiresAt.After(databaseNow) {
		return nil,
			ErrAssistantRuntimeTurnSessionUnavailable
	}

	if strings.TrimSpace(activeTurnID) != "" {
		return nil,
			ErrAssistantRuntimeTurnInProgress
	}

	if turnCount >= maxTurns {
		return nil,
			ErrAssistantRuntimeTurnLimitReached
	}

	messages, err :=
		decodeAssistantRuntimeClaimMessages(
			messagesJSON,
		)
	if err != nil {
		return nil, err
	}

	if sessionKind ==
		models.AssistantRuntimeSessionKindExternal {
		var (
			succeededToday int
			activeClaims   int
		)

		err = tx.QueryRow(
			ctx,
			`
			SELECT
				(
					SELECT COUNT(*)::integer
					FROM assistant_runtime_usage u
					WHERE u.deployment_id = $1
					  AND u.session_kind = 'external'
					  AND u.status = 'succeeded'
					  AND u.created_at >= CURRENT_DATE
				),
				(
					SELECT COUNT(*)::integer
					FROM assistant_runtime_sessions s
					WHERE s.deployment_id = $1
					  AND s.session_kind = 'external'
					  AND s.status = 'active'
					  AND s.active_turn_id IS NOT NULL
					  AND s.active_turn_started_at >=
							NOW() - INTERVAL '20 minutes'
				)`,
			deploymentID,
		).Scan(
			&succeededToday,
			&activeClaims,
		)
		if err != nil {
			return nil,
				fmt.Errorf(
					"统计教学智能体外部每日额度失败: %w",
					err,
				)
		}

		if !assistantRuntimeDailyQuotaAllows(
			dailyCallLimit,
			succeededToday,
			activeClaims,
		) {
			return nil,
				ErrAssistantRuntimeDailyQuotaExceeded
		}
	}

	var (
		accountStatus    string
		accountBalance   float64
		accountFrozen    float64
		accountExpiresAt *time.Time
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			ta.status,
			ta.balance,
			ta.frozen_amount,
			ta.expires_at
		FROM token_accounts ta
		WHERE ta.account_type = 'personal'
		  AND ta.owner_id = $1
		FOR SHARE`,
		ownerUserID,
	).Scan(
		&accountStatus,
		&accountBalance,
		&accountFrozen,
		&accountExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrAssistantRuntimeBillingAccountUnavailable
		}

		return nil,
			fmt.Errorf(
				"检查教学智能体付费账户失败: %w",
				err,
			)
	}

	if !assistantRuntimeBillingAccountAllows(
		accountStatus,
		accountBalance,
		accountFrozen,
		accountExpiresAt,
		databaseNow,
	) {
		return nil,
			ErrAssistantRuntimeBillingAccountUnavailable
	}

	var claimedAt time.Time
	err = tx.QueryRow(
		ctx,
		`
		UPDATE assistant_runtime_sessions
		SET
			active_turn_id = $2,
			active_turn_started_at = NOW(),
			last_active_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
		  AND active_turn_id IS NULL
		RETURNING active_turn_started_at`,
		sessionID,
		turnID,
	).Scan(&claimedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrAssistantRuntimeTurnInProgress
		}

		return nil,
			fmt.Errorf(
				"领取教学智能体运行主轮次失败: %w",
				err,
			)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf(
				"提交教学智能体主轮次领取事务失败: %w",
				err,
			)
	}

	return &models.AssistantRuntimeTurnClaim{
		TurnID:            turnID,
		SessionID:         sessionID,
		DeploymentID:      deploymentID,
		DeploymentVersion: deploymentVersion,
		TurnCount:         turnCount,
		MaxTurns:          maxTurns,
		Messages:          messages,
		ClaimedAt:         claimedAt,
	}, nil
}
