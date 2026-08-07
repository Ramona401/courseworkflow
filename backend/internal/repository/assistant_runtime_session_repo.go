package repository

// assistant_runtime_session_repo.go
//
// 本文件实现教学智能体短时运行会话的创建、令牌绑定读取和终态标记。
//
// 支持两种创建边界：
//
//   external：
//     - 部署必须active；
//     - current_version必须一致；
//     - 部署处于有效期；
//     - 请求Origin必须仍包含在实时allowed_origins_json中；
//     - 会话最大轮数必须等于部署实时策略。
//
//   teacher_preview：
//     - 部署必须active；
//     - current_version必须一致；
//     - 部署处于有效期；
//     - owner_user_id必须等于已认证教师；
//     - 不读取或扩大外部Origin白名单；
//     - 会话最大轮数必须等于部署实时策略。
//
// 数据库只保存JTI哈希、匿名身份哈希和IP哈希，不保存运行令牌、
// 浏览器匿名标识、教师登录JWT或原始IP。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	// ErrAssistantRuntimeSessionNotFound 表示会话不存在。
	ErrAssistantRuntimeSessionNotFound = errors.New(
		"教学智能体运行会话不存在",
	)

	// ErrAssistantRuntimeSessionTokenMismatch 表示会话和JTI哈希不匹配。
	ErrAssistantRuntimeSessionTokenMismatch = errors.New(
		"教学智能体运行会话令牌不匹配",
	)

	// ErrAssistantRuntimeSessionDeploymentUnavailable 表示部署实时状态已不允许新建会话。
	ErrAssistantRuntimeSessionDeploymentUnavailable = errors.New(
		"教学智能体部署当前不可创建运行会话",
	)

	// ErrAssistantRuntimeSessionPolicyConflict 表示会话参数与部署实时策略不一致。
	ErrAssistantRuntimeSessionPolicyConflict = errors.New(
		"教学智能体运行会话策略已发生变化",
	)

	// ErrAssistantRuntimeSessionTokenConflict 表示随机会话ID或JTI哈希唯一冲突。
	ErrAssistantRuntimeSessionTokenConflict = errors.New(
		"教学智能体运行会话随机标识冲突",
	)

	// ErrAssistantRuntimeSessionInputInvalid 表示仓储输入不完整。
	ErrAssistantRuntimeSessionInputInvalid = errors.New(
		"教学智能体运行会话参数无效",
	)
)

// assistantRuntimeSessionSelectColumns 是完整内部会话扫描协议。
//
// 调用SQL必须使用assistant_runtime_sessions别名s。
const assistantRuntimeSessionSelectColumns = `
	s.id::text,
	s.deployment_id::text,
	s.deployment_version,
	s.token_jti_hash,
	s.anonymous_client_hash,
	s.origin_snapshot,
	s.ip_hash,
	s.session_kind,
	s.status,
	s.turn_count,
	s.max_turns,
	COALESCE(s.active_turn_id::text, ''),
	s.active_turn_started_at,
	COALESCE(s.messages_json::text, '[]'),
	s.expires_at,
	s.last_active_at,
	s.created_at,
	s.updated_at`

// scanAssistantRuntimeSession 扫描后端内部会话记录。
func scanAssistantRuntimeSession(
	row interface {
		Scan(dest ...interface{}) error
	},
) (
	*models.AssistantRuntimeSession,
	error,
) {
	session := &models.AssistantRuntimeSession{}

	activeTurnID := ""
	var activeTurnStartedAt *time.Time

	var expiresAt time.Time
	var lastActiveAt time.Time
	var createdAt time.Time
	var updatedAt time.Time

	err := row.Scan(
		&session.ID,
		&session.DeploymentID,
		&session.DeploymentVersion,
		&session.TokenJTIHash,
		&session.AnonymousClientHash,
		&session.OriginSnapshot,
		&session.IPHash,
		&session.SessionKind,
		&session.Status,
		&session.TurnCount,
		&session.MaxTurns,
		&activeTurnID,
		&activeTurnStartedAt,
		&session.MessagesJSON,
		&expiresAt,
		&lastActiveAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(activeTurnID) != "" {
		session.ActiveTurnID = &activeTurnID
	}
	session.ActiveTurnStartedAt = activeTurnStartedAt

	session.ExpiresAt = &expiresAt
	session.LastActiveAt = &lastActiveAt
	session.CreatedAt = &createdAt
	session.UpdatedAt = &updatedAt

	return session, nil
}

// validateAssistantRuntimeSessionCreateInput 校验写入参数。
func validateAssistantRuntimeSessionCreateInput(
	sessionID string,
	input *models.AssistantRuntimeSessionCreateInput,
) error {
	sessionID = strings.TrimSpace(sessionID)

	if sessionID == "" ||
		input == nil ||
		strings.TrimSpace(input.DeploymentID) == "" ||
		input.DeploymentVersion <= 0 ||
		len(strings.TrimSpace(input.TokenJTIHash)) != 64 ||
		len(strings.TrimSpace(input.AnonymousClientHash)) != 64 ||
		len(strings.TrimSpace(input.IPHash)) != 64 ||
		strings.TrimSpace(input.OriginSnapshot) == "" ||
		len([]rune(strings.TrimSpace(input.OriginSnapshot))) > 512 ||
		!models.IsValidAssistantRuntimeSessionKind(input.SessionKind) ||
		input.MaxTurns < 1 ||
		input.MaxTurns > 100 ||
		input.ExpiresAt.IsZero() {
		return ErrAssistantRuntimeSessionInputInvalid
	}

	return nil
}

// CreateAssistantRuntimeSession 原子校验外部部署Origin并创建短时会话。
func CreateAssistantRuntimeSession(
	ctx context.Context,
	sessionID string,
	input *models.AssistantRuntimeSessionCreateInput,
) (
	*models.AssistantRuntimeSession,
	error,
) {
	if err := validateAssistantRuntimeSessionCreateInput(
		sessionID,
		input,
	); err != nil {
		return nil, err
	}

	if strings.TrimSpace(input.SessionKind) !=
		models.AssistantRuntimeSessionKindExternal {
		return nil,
			ErrAssistantRuntimeSessionInputInvalid
	}

	sessionID = strings.TrimSpace(sessionID)
	originSnapshot := strings.TrimSpace(
		input.OriginSnapshot,
	)

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启教学智能体运行会话事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var deploymentMaxTurns int
	var deploymentValidUntil *time.Time
	var databaseNow time.Time

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			d.per_session_turn_limit,
			d.valid_until,
			NOW()
		FROM assistant_deployments d
		WHERE d.id = $1
		  AND d.current_version = $2
		  AND d.status = 'active'
		  AND d.access_mode = 'origin_allowlist'
		  AND d.valid_from <= NOW()
		  AND (
				d.valid_until IS NULL
				OR d.valid_until > NOW()
		  )
		  AND d.allowed_origins_json @>
				jsonb_build_array($3::text)
		FOR SHARE`,
		input.DeploymentID,
		input.DeploymentVersion,
		originSnapshot,
	).Scan(
		&deploymentMaxTurns,
		&deploymentValidUntil,
		&databaseNow,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrAssistantRuntimeSessionDeploymentUnavailable
		}

		return nil,
			fmt.Errorf(
				"锁定教学智能体部署会话策略失败: %w",
				err,
			)
	}

	if err :=
		validateAssistantRuntimeSessionRealtimePolicy(
			input,
			deploymentMaxTurns,
			deploymentValidUntil,
			databaseNow,
		); err != nil {
		return nil, err
	}

	session, err :=
		insertAssistantRuntimeSessionTx(
			ctx,
			tx,
			sessionID,
			input,
		)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf(
				"提交教学智能体运行会话事务失败: %w",
				err,
			)
	}

	return session, nil
}

// CreateAssistantRuntimeTeacherPreviewSession 为部署所有者创建教师预览会话。
//
// ownerUserID必须来自教师登录JWT。查询同时绑定部署ID与owner_user_id，
// 不允许管理员角色或客户端字段代替真实部署所有者。
//
// 教师预览只绕过外部Origin白名单，不绕过部署active状态、当前版本、
// 有效期、最大轮数、短时令牌、积分账户和正式结算链。
func CreateAssistantRuntimeTeacherPreviewSession(
	ctx context.Context,
	sessionID string,
	ownerUserID string,
	input *models.AssistantRuntimeSessionCreateInput,
) (
	*models.AssistantRuntimeSession,
	error,
) {
	if err := validateAssistantRuntimeSessionCreateInput(
		sessionID,
		input,
	); err != nil {
		return nil, err
	}

	ownerUserID = strings.TrimSpace(ownerUserID)

	if ownerUserID == "" ||
		strings.TrimSpace(input.SessionKind) !=
			models.AssistantRuntimeSessionKindTeacherPreview {
		return nil,
			ErrAssistantRuntimeSessionInputInvalid
	}

	sessionID = strings.TrimSpace(sessionID)

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开启教师预览运行会话事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var deploymentMaxTurns int
	var deploymentValidUntil *time.Time
	var databaseNow time.Time

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			d.per_session_turn_limit,
			d.valid_until,
			NOW()
		FROM assistant_deployments d
		WHERE d.id = $1
		  AND d.owner_user_id = $2
		  AND d.current_version = $3
		  AND d.status = 'active'
		  AND d.valid_from <= NOW()
		  AND (
				d.valid_until IS NULL
				OR d.valid_until > NOW()
		  )
		FOR SHARE`,
		input.DeploymentID,
		ownerUserID,
		input.DeploymentVersion,
	).Scan(
		&deploymentMaxTurns,
		&deploymentValidUntil,
		&databaseNow,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrAssistantRuntimeSessionDeploymentUnavailable
		}

		return nil,
			fmt.Errorf(
				"锁定教师预览部署策略失败: %w",
				err,
			)
	}

	if err :=
		validateAssistantRuntimeSessionRealtimePolicy(
			input,
			deploymentMaxTurns,
			deploymentValidUntil,
			databaseNow,
		); err != nil {
		return nil, err
	}

	session, err :=
		insertAssistantRuntimeSessionTx(
			ctx,
			tx,
			sessionID,
			input,
		)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil,
			fmt.Errorf(
				"提交教师预览运行会话事务失败: %w",
				err,
			)
	}

	return session, nil
}

// validateAssistantRuntimeSessionRealtimePolicy 复核会话输入与实时部署策略一致。
func validateAssistantRuntimeSessionRealtimePolicy(
	input *models.AssistantRuntimeSessionCreateInput,
	deploymentMaxTurns int,
	deploymentValidUntil *time.Time,
	databaseNow time.Time,
) error {
	if input == nil ||
		input.MaxTurns != deploymentMaxTurns ||
		!input.ExpiresAt.After(databaseNow) ||
		(deploymentValidUntil != nil &&
			input.ExpiresAt.After(*deploymentValidUntil)) {
		return ErrAssistantRuntimeSessionPolicyConflict
	}

	return nil
}

// insertAssistantRuntimeSessionTx 在调用方已完成部署锁定后插入会话。
func insertAssistantRuntimeSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	input *models.AssistantRuntimeSessionCreateInput,
) (
	*models.AssistantRuntimeSession,
	error,
) {
	session := &models.AssistantRuntimeSession{
		ID:                  strings.TrimSpace(sessionID),
		DeploymentID:        strings.TrimSpace(input.DeploymentID),
		DeploymentVersion:   input.DeploymentVersion,
		TokenJTIHash:        strings.TrimSpace(input.TokenJTIHash),
		AnonymousClientHash: strings.TrimSpace(input.AnonymousClientHash),
		OriginSnapshot:      strings.TrimSpace(input.OriginSnapshot),
		IPHash:              strings.TrimSpace(input.IPHash),
		SessionKind:         strings.TrimSpace(input.SessionKind),
		Status:              models.AssistantRuntimeSessionStatusActive,
		TurnCount:           0,
		MaxTurns:            input.MaxTurns,
		MessagesJSON:        "[]",
	}

	var expiresAt time.Time
	var lastActiveAt time.Time
	var createdAt time.Time
	var updatedAt time.Time

	err := tx.QueryRow(
		ctx,
		`
		INSERT INTO assistant_runtime_sessions (
			id,
			deployment_id,
			deployment_version,
			token_jti_hash,
			anonymous_client_hash,
			origin_snapshot,
			ip_hash,
			session_kind,
			status,
			turn_count,
			max_turns,
			active_turn_id,
			active_turn_started_at,
			messages_json,
			expires_at,
			last_active_at,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			'active',
			0,
			$9,
			NULL,
			NULL,
			'[]'::jsonb,
			$10,
			NOW(),
			NOW(),
			NOW()
		)
		RETURNING
			expires_at,
			last_active_at,
			created_at,
			updated_at`,
		session.ID,
		session.DeploymentID,
		session.DeploymentVersion,
		session.TokenJTIHash,
		session.AnonymousClientHash,
		session.OriginSnapshot,
		session.IPHash,
		session.SessionKind,
		session.MaxTurns,
		input.ExpiresAt,
	).Scan(
		&expiresAt,
		&lastActiveAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if assistantRuntimeSessionConstraintName(err) != "" {
			return nil,
				fmt.Errorf(
					"%w: %v",
					ErrAssistantRuntimeSessionTokenConflict,
					err,
				)
		}

		return nil,
			fmt.Errorf(
				"创建教学智能体运行会话失败: %w",
				err,
			)
	}

	session.ExpiresAt = &expiresAt
	session.LastActiveAt = &lastActiveAt
	session.CreatedAt = &createdAt
	session.UpdatedAt = &updatedAt

	return session, nil
}

// GetAssistantRuntimeSessionForToken 按会话ID和JTI哈希双重边界读取。
func GetAssistantRuntimeSessionForToken(
	ctx context.Context,
	sessionID string,
	tokenJTIHash string,
) (
	*models.AssistantRuntimeSession,
	error,
) {
	sessionID = strings.TrimSpace(sessionID)
	tokenJTIHash = strings.TrimSpace(tokenJTIHash)

	if sessionID == "" ||
		len(tokenJTIHash) != 64 {
		return nil,
			ErrAssistantRuntimeSessionTokenMismatch
	}

	session, err := scanAssistantRuntimeSession(
		database.DB.QueryRow(
			ctx,
			`SELECT `+
				assistantRuntimeSessionSelectColumns+
				`
			 FROM assistant_runtime_sessions s
			 WHERE s.id = $1
			   AND s.token_jti_hash = $2`,
			sessionID,
			tokenJTIHash,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrAssistantRuntimeSessionTokenMismatch
		}

		return nil,
			fmt.Errorf(
				"查询教学智能体运行会话失败: %w",
				err,
			)
	}

	return session, nil
}

// MarkAssistantRuntimeSessionExpired 把活动会话标记为expired。
func MarkAssistantRuntimeSessionExpired(
	ctx context.Context,
	sessionID string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`
		UPDATE assistant_runtime_sessions
		SET
			status = 'expired',
			active_turn_id = NULL,
			active_turn_started_at = NULL,
			updated_at = NOW()
		WHERE id = $1
		  AND status = 'active'`,
		strings.TrimSpace(sessionID),
	)
	if err != nil {
		return fmt.Errorf(
			"标记教学智能体运行会话过期失败: %w",
			err,
		)
	}

	return nil
}

// RevokeAssistantRuntimeSession 把活动会话永久标记为revoked。
func RevokeAssistantRuntimeSession(
	ctx context.Context,
	sessionID string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`
		UPDATE assistant_runtime_sessions
		SET
			status = 'revoked',
			active_turn_id = NULL,
			active_turn_started_at = NULL,
			updated_at = NOW()
		WHERE id = $1
		  AND status = 'active'`,
		strings.TrimSpace(sessionID),
	)
	if err != nil {
		return fmt.Errorf(
			"撤销教学智能体运行会话失败: %w",
			err,
		)
	}

	return nil
}

// assistantRuntimeSessionConstraintName 提取唯一约束名。
func assistantRuntimeSessionConstraintName(
	err error,
) string {
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) {
		return ""
	}

	switch strings.TrimSpace(pgError.ConstraintName) {
	case "assistant_runtime_sessions_pkey",
		"uq_assistant_runtime_sessions_jti":
		return strings.TrimSpace(pgError.ConstraintName)
	default:
		return ""
	}
}
