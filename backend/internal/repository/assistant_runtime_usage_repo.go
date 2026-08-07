package repository

// assistant_runtime_usage_repo.go
//
// 本文件定义运行结算公共协议、幂等检查、会话锁定、部署身份复核、
// 教师个人账户锁定、正式消息装配和输入校验。
//
// 成功和失败事务的实际写入分别集中在：
//   assistant_runtime_usage_settlement_repo.go
//
// 安全边界：
//   - turn_id是唯一结算幂等键；
//   - active_turn_id必须与当前turn_id完全一致；
//   - owner_user_id、school_id、courseware_id和page_id必须重新匹配部署主表；
//   - 个人积分账户只能按部署创建者读取；
//   - 正式消息只允许student和assistant角色；
//   - 不保存系统提示词或隐藏推理。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tedna/internal/models"
)

const (
	assistantRuntimeVisibleMessageMaxCount = 200
	assistantRuntimeVisibleMessageMaxRunes = 128 * 1024
)

var (
	ErrAssistantRuntimeTurnAlreadyFinalized = errors.New(
		"教学智能体运行主轮次已经结算",
	)

	ErrAssistantRuntimeTurnNotClaimed = errors.New(
		"教学智能体运行主轮次未被当前请求领取",
	)

	ErrAssistantRuntimeUsageInputInvalid = errors.New(
		"教学智能体运行结算参数无效",
	)

	ErrAssistantRuntimeSettlementIdentityMismatch = errors.New(
		"教学智能体运行结算身份与部署快照不匹配",
	)
)

// AssistantRuntimeSuccessSettlementInput 是成功结算输入。
type AssistantRuntimeSuccessSettlementInput struct {
	TurnID            string
	SessionID         string
	DeploymentID      string
	DeploymentVersion int

	OwnerUserID  string
	SchoolID     string
	CoursewareID string
	PageID       string
	SceneCode    string

	StudentMessage   models.AssistantRuntimeMessage
	AssistantMessage models.AssistantRuntimeMessage

	InputChars   int
	OutputChars  int
	InputTokens  int
	OutputTokens int

	ModelName string
	Provider  string
	LatencyMs int

	Calculation *models.CreditCalculation
}

// AssistantRuntimeFailureSettlementInput 是失败结算输入。
type AssistantRuntimeFailureSettlementInput struct {
	TurnID            string
	SessionID         string
	DeploymentID      string
	DeploymentVersion int

	OwnerUserID  string
	SchoolID     string
	CoursewareID string
	PageID       string
	SceneCode    string
	SessionKind  string

	InputChars int
	ErrorCode  string
	ModelName  string
	Provider   string
	LatencyMs  int
}

// AssistantRuntimeBillingSettlement 返回成功后的会话和账户快照。
type AssistantRuntimeBillingSettlement struct {
	Session      *models.AssistantRuntimeSession
	Account      *models.TokenAccount
	BalanceAfter float64
}

// assistantRuntimeLockedSettlementSession 是事务内锁定的会话快照。
type assistantRuntimeLockedSettlementSession struct {
	Session      *models.AssistantRuntimeSession
	ActiveTurnID string
	ExpiresAt    time.Time
}

// assistantRuntimeUsageExistsTx 检查turn_id是否已经写入运行流水。
func assistantRuntimeUsageExistsTx(
	ctx context.Context,
	tx pgx.Tx,
	turnID string,
) (
	bool,
	error,
) {
	var exists bool

	err := tx.QueryRow(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1
			FROM assistant_runtime_usage
			WHERE turn_id = $1
		)`,
		strings.TrimSpace(turnID),
	).Scan(&exists)

	return exists, err
}

// lockAssistantRuntimeSettlementSession 锁定并验证待结算会话。
//
// 会话在AI调用期间可以自然到期，但已经发生的模型成本仍须结算。
// 因此这里只要求会话仍持有当前active_turn_id，不因expires_at经过而拒绝结算。
func lockAssistantRuntimeSettlementSession(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	turnID string,
	deploymentID string,
	deploymentVersion int,
) (
	*assistantRuntimeLockedSettlementSession,
	error,
) {
	session := &models.AssistantRuntimeSession{}
	activeTurnID := ""
	expiresAt := time.Time{}

	err := tx.QueryRow(
		ctx,
		`
		SELECT
			s.id::text,
			s.deployment_id::text,
			s.deployment_version,
			s.session_kind,
			s.status,
			s.turn_count,
			s.max_turns,
			COALESCE(s.active_turn_id::text, ''),
			COALESCE(s.messages_json::text, '[]'),
			s.expires_at
		FROM assistant_runtime_sessions s
		WHERE s.id = $1
		FOR UPDATE`,
		strings.TrimSpace(sessionID),
	).Scan(
		&session.ID,
		&session.DeploymentID,
		&session.DeploymentVersion,
		&session.SessionKind,
		&session.Status,
		&session.TurnCount,
		&session.MaxTurns,
		&activeTurnID,
		&session.MessagesJSON,
		&expiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrAssistantRuntimeTurnNotClaimed
		}

		return nil,
			fmt.Errorf(
				"锁定教学智能体结算会话失败: %w",
				err,
			)
	}

	if session.Status !=
		models.AssistantRuntimeSessionStatusActive ||
		strings.TrimSpace(session.DeploymentID) !=
			strings.TrimSpace(deploymentID) ||
		session.DeploymentVersion != deploymentVersion ||
		strings.TrimSpace(activeTurnID) !=
			strings.TrimSpace(turnID) {
		return nil,
			ErrAssistantRuntimeTurnNotClaimed
	}

	if session.TurnCount < 0 ||
		session.MaxTurns < 1 ||
		session.TurnCount >= session.MaxTurns {
		return nil,
			ErrAssistantRuntimeUsageInputInvalid
	}

	session.ExpiresAt = &expiresAt

	return &assistantRuntimeLockedSettlementSession{
		Session:      session,
		ActiveTurnID: activeTurnID,
		ExpiresAt:    expiresAt,
	}, nil
}

// validateAssistantRuntimeSettlementDeploymentTx 重新读取部署计费身份。
//
// 部署在AI调用期间可以暂停、撤销或发布新版本；已经发生的调用仍需按领取
// 主轮次时绑定的创建者、学校、课件和页面完成结算，因此不要求当前状态或版本不变。
func validateAssistantRuntimeSettlementDeploymentTx(
	ctx context.Context,
	tx pgx.Tx,
	deploymentID string,
	ownerUserID string,
	schoolID string,
	coursewareID string,
	pageID string,
) error {
	var (
		storedOwnerID      string
		storedSchoolID     string
		storedCoursewareID string
		storedPageID       string
	)

	err := tx.QueryRow(
		ctx,
		`
		SELECT
			d.owner_user_id::text,
			d.school_id::text,
			d.courseware_id::text,
			d.page_id::text
		FROM assistant_deployments d
		WHERE d.id = $1
		FOR SHARE`,
		strings.TrimSpace(deploymentID),
	).Scan(
		&storedOwnerID,
		&storedSchoolID,
		&storedCoursewareID,
		&storedPageID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAssistantRuntimeSettlementIdentityMismatch
		}

		return fmt.Errorf(
			"复核运行结算部署身份失败: %w",
			err,
		)
	}

	if strings.TrimSpace(storedOwnerID) !=
		strings.TrimSpace(ownerUserID) ||
		strings.TrimSpace(storedSchoolID) !=
			strings.TrimSpace(schoolID) ||
		strings.TrimSpace(storedCoursewareID) !=
			strings.TrimSpace(coursewareID) ||
		strings.TrimSpace(storedPageID) !=
			strings.TrimSpace(pageID) {
		return ErrAssistantRuntimeSettlementIdentityMismatch
	}

	return nil
}

// loadAssistantRuntimeBillingAccountTx 锁定部署创建者个人账户。
//
// 调用开始前已完成严格余额检查。即使账户在AI执行期间被暂停，已经发生的
// 模型成本仍须落账，因此最终结算不再用status条件过滤账户。
func loadAssistantRuntimeBillingAccountTx(
	ctx context.Context,
	tx pgx.Tx,
	ownerUserID string,
) (
	*models.TokenAccount,
	error,
) {
	account := &models.TokenAccount{}

	parentAccountID := ""
	var expiresAt *time.Time
	createdAt := time.Time{}
	updatedAt := time.Time{}

	err := tx.QueryRow(
		ctx,
		`
		SELECT
			ta.id::text,
			ta.account_type,
			ta.owner_id::text,
			COALESCE(ta.parent_account_id::text, ''),
			ta.display_name,
			ta.balance,
			ta.frozen_amount,
			ta.total_consumed,
			ta.total_quota,
			ta.monthly_quota,
			ta.expires_at,
			ta.status,
			ta.created_at,
			ta.updated_at
		FROM token_accounts ta
		WHERE ta.account_type = 'personal'
		  AND ta.owner_id = $1
		FOR UPDATE`,
		strings.TrimSpace(ownerUserID),
	).Scan(
		&account.ID,
		&account.AccountType,
		&account.OwnerID,
		&parentAccountID,
		&account.DisplayName,
		&account.Balance,
		&account.FrozenAmount,
		&account.TotalConsumed,
		&account.TotalQuota,
		&account.MonthlyQuota,
		&expiresAt,
		&account.Status,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrAssistantRuntimeBillingAccountUnavailable
		}

		return nil,
			fmt.Errorf(
				"锁定教学智能体付费账户失败: %w",
				err,
			)
	}

	if strings.TrimSpace(parentAccountID) != "" {
		account.ParentAccountID = &parentAccountID
	}
	account.ExpiresAt = expiresAt
	account.CreatedAt = &createdAt
	account.UpdatedAt = &updatedAt

	return account, nil
}

// appendAssistantRuntimeVisibleMessages 追加正式可见消息。
func appendAssistantRuntimeVisibleMessages(
	raw string,
	student models.AssistantRuntimeMessage,
	assistant models.AssistantRuntimeMessage,
) (
	string,
	error,
) {
	messages, err :=
		decodeAssistantRuntimeClaimMessages(raw)
	if err != nil {
		return "",
			err
	}

	if student.Role !=
		models.AssistantRuntimeMessageRoleStudent ||
		assistant.Role !=
			models.AssistantRuntimeMessageRoleAssistant ||
		strings.TrimSpace(student.Content) == "" ||
		strings.TrimSpace(assistant.Content) == "" ||
		len([]rune(student.Content)) >
			assistantRuntimeVisibleMessageMaxRunes ||
		len([]rune(assistant.Content)) >
			assistantRuntimeVisibleMessageMaxRunes ||
		len(messages)+2 >
			assistantRuntimeVisibleMessageMaxCount {
		return "",
			ErrAssistantRuntimeMessagesInvalid
	}

	messages = append(
		messages,
		student,
		assistant,
	)

	encoded, err := json.Marshal(messages)
	if err != nil {
		return "",
			fmt.Errorf(
				"%w: %v",
				ErrAssistantRuntimeMessagesInvalid,
				err,
			)
	}

	return string(encoded), nil
}

// assistantRuntimeNonNegativeFinite 校验金额和成本数值。
func assistantRuntimeNonNegativeFinite(
	value float64,
) bool {
	return !math.IsNaN(value) &&
		!math.IsInf(value, 0) &&
		value >= 0
}

// validateAssistantRuntimeSuccessSettlementInput 校验成功结算输入。
func validateAssistantRuntimeSuccessSettlementInput(
	input *AssistantRuntimeSuccessSettlementInput,
) error {
	if input == nil ||
		strings.TrimSpace(input.TurnID) == "" ||
		strings.TrimSpace(input.SessionID) == "" ||
		strings.TrimSpace(input.DeploymentID) == "" ||
		input.DeploymentVersion <= 0 ||
		strings.TrimSpace(input.OwnerUserID) == "" ||
		strings.TrimSpace(input.SchoolID) == "" ||
		strings.TrimSpace(input.CoursewareID) == "" ||
		strings.TrimSpace(input.PageID) == "" ||
		strings.TrimSpace(input.SceneCode) == "" ||
		input.StudentMessage.Role !=
			models.AssistantRuntimeMessageRoleStudent ||
		input.AssistantMessage.Role !=
			models.AssistantRuntimeMessageRoleAssistant ||
		strings.TrimSpace(input.StudentMessage.Content) == "" ||
		strings.TrimSpace(input.AssistantMessage.Content) == "" ||
		input.InputChars < 0 ||
		input.OutputChars < 0 ||
		input.InputTokens < 0 ||
		input.OutputTokens < 0 ||
		input.LatencyMs < 0 ||
		strings.TrimSpace(input.ModelName) == "" ||
		input.Calculation == nil ||
		!assistantRuntimeNonNegativeFinite(
			input.Calculation.CreditsConsumed,
		) ||
		!assistantRuntimeNonNegativeFinite(
			input.Calculation.CostUSD,
		) ||
		!assistantRuntimeNonNegativeFinite(
			input.Calculation.ExchangeRate,
		) ||
		!assistantRuntimeNonNegativeFinite(
			input.Calculation.Multiplier,
		) {
		return ErrAssistantRuntimeUsageInputInvalid
	}

	return nil
}

// validateAssistantRuntimeFailureSettlementInput 校验失败结算输入。
func validateAssistantRuntimeFailureSettlementInput(
	input *AssistantRuntimeFailureSettlementInput,
) error {
	if input == nil ||
		strings.TrimSpace(input.TurnID) == "" ||
		strings.TrimSpace(input.SessionID) == "" ||
		strings.TrimSpace(input.DeploymentID) == "" ||
		input.DeploymentVersion <= 0 ||
		strings.TrimSpace(input.OwnerUserID) == "" ||
		strings.TrimSpace(input.SchoolID) == "" ||
		strings.TrimSpace(input.CoursewareID) == "" ||
		strings.TrimSpace(input.PageID) == "" ||
		strings.TrimSpace(input.SceneCode) == "" ||
		!models.IsValidAssistantRuntimeSessionKind(
			input.SessionKind,
		) ||
		input.InputChars < 0 ||
		input.LatencyMs < 0 ||
		strings.TrimSpace(input.ErrorCode) == "" ||
		len([]rune(input.ErrorCode)) > 64 {
		return ErrAssistantRuntimeUsageInputInvalid
	}

	return nil
}

// assistantRuntimeUsageConstraintName 提取运行流水唯一约束名。
func assistantRuntimeUsageConstraintName(
	err error,
) string {
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) {
		return ""
	}

	if strings.TrimSpace(pgError.ConstraintName) ==
		"uq_assistant_runtime_usage_turn" {
		return pgError.ConstraintName
	}

	return ""
}
