package repository

// courseware_rebuild_discussion_repo.go — 课件全页重构讨论会话数据访问。
// 消息只保存老师可见的正式交流，并通过JSONB原子追加避免并发覆盖。
// 活动会话受部分唯一索引约束；确认执行使用状态条件更新防止重复生成。
// 所有读取和修改均绑定created_by，避免跨用户读取讨论内容。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
)

const (
	CWRebuildDiscussionStatusDiscussing           = "discussing"
	CWRebuildDiscussionStatusAwaitingConfirmation = "awaiting_confirmation"
	CWRebuildDiscussionStatusExecuting            = "executing"
	CWRebuildDiscussionStatusCompleted            = "completed"
	CWRebuildDiscussionStatusCancelled            = "cancelled"
	CWRebuildDiscussionStatusStale                = "stale"
)

var (
	ErrCWRebuildDiscussionNotFound = errors.New("课件重构讨论不存在")
	ErrCWRebuildDiscussionConflict = errors.New("课件重构讨论状态已变化")
)

// CoursewareRebuildDiscussionMessage 是老师可见的正式讨论消息。
// Role只允许teacher或assistant；CreatedAt使用RFC3339字符串。
type CoursewareRebuildDiscussionMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// CoursewareRebuildDiscussion 表示一场页面全页重构讨论。
type CoursewareRebuildDiscussion struct {
	ID                string                               `json:"id"`
	CoursewareID      string                               `json:"courseware_id"`
	PageID            string                               `json:"page_id"`
	PageNumber        int                                  `json:"page_number"`
	CreatedBy         string                               `json:"created_by"`
	Status            string                               `json:"status"`
	BasePageUpdatedAt time.Time                            `json:"base_page_updated_at"`
	ReferenceContext  string                               `json:"reference_context"`
	Messages          []CoursewareRebuildDiscussionMessage `json:"messages"`
	FinalInstruction  string                               `json:"final_instruction"`
	AISummary         string                               `json:"ai_summary"`
	ErrorMessage      string                               `json:"error_message"`
	ConfirmedAt       *time.Time                           `json:"confirmed_at"`
	ExecutedAt        *time.Time                           `json:"executed_at"`
	CreatedAt         time.Time                            `json:"created_at"`
	UpdatedAt         time.Time                            `json:"updated_at"`
}

type cwRebuildDiscussionScanner interface {
	Scan(dest ...any) error
}

const cwRebuildDiscussionSelectColumns = `
	id,
	courseware_id,
	page_id,
	page_number,
	created_by,
	status,
	base_page_updated_at,
	reference_context,
	messages::text,
	final_instruction,
	ai_summary,
	error_message,
	confirmed_at,
	executed_at,
	created_at,
	updated_at
`

func scanCWRebuildDiscussion(
	row cwRebuildDiscussionScanner,
) (*CoursewareRebuildDiscussion, error) {
	var item CoursewareRebuildDiscussion
	var messagesJSON string

	if err := row.Scan(
		&item.ID,
		&item.CoursewareID,
		&item.PageID,
		&item.PageNumber,
		&item.CreatedBy,
		&item.Status,
		&item.BasePageUpdatedAt,
		&item.ReferenceContext,
		&messagesJSON,
		&item.FinalInstruction,
		&item.AISummary,
		&item.ErrorMessage,
		&item.ConfirmedAt,
		&item.ExecutedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCWRebuildDiscussionNotFound
		}
		return nil, err
	}

	item.Messages = make(
		[]CoursewareRebuildDiscussionMessage,
		0,
	)

	if messagesJSON != "" {
		if err := json.Unmarshal(
			[]byte(messagesJSON),
			&item.Messages,
		); err != nil {
			return nil, fmt.Errorf(
				"解析课件重构讨论消息失败: %w",
				err,
			)
		}
	}

	if item.Messages == nil {
		item.Messages = make(
			[]CoursewareRebuildDiscussionMessage,
			0,
		)
	}

	return &item, nil
}

// GetOrCreateActiveCWRebuildDiscussion 返回当前用户在指定页面上的活动讨论。
//
// 部分唯一索引保证并发创建时仍只会存在一条活动记录。
// 若已经存在活动记录，本SQL保留原记录和原参考上下文，不覆盖讨论内容。
func GetOrCreateActiveCWRebuildDiscussion(
	ctx context.Context,
	coursewareID string,
	pageID string,
	pageNumber int,
	userID string,
	basePageUpdatedAt time.Time,
	referenceContext string,
) (*CoursewareRebuildDiscussion, error) {
	query := `
		INSERT INTO courseware_rebuild_discussions (
			courseware_id,
			page_id,
			page_number,
			created_by,
			status,
			base_page_updated_at,
			reference_context,
			messages
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			'discussing',
			$5,
			$6,
			'[]'::jsonb
		)
		ON CONFLICT (
			courseware_id,
			page_id,
			created_by
		)
		WHERE status IN (
			'discussing',
			'awaiting_confirmation',
			'executing'
		)
		DO UPDATE SET
			updated_at =
				courseware_rebuild_discussions.updated_at
		RETURNING
	` + cwRebuildDiscussionSelectColumns

	return scanCWRebuildDiscussion(
		database.DB.QueryRow(
			ctx,
			query,
			coursewareID,
			pageID,
			pageNumber,
			userID,
			basePageUpdatedAt,
			referenceContext,
		),
	)
}

// GetLatestActiveCWRebuildDiscussion 查询当前用户在指定页面上的活动讨论。
func GetLatestActiveCWRebuildDiscussion(
	ctx context.Context,
	coursewareID string,
	pageID string,
	userID string,
) (*CoursewareRebuildDiscussion, error) {
	query := `
		SELECT
	` + cwRebuildDiscussionSelectColumns + `
		FROM courseware_rebuild_discussions
		WHERE courseware_id = $1
		  AND page_id = $2
		  AND created_by = $3
		  AND status IN (
				'discussing',
				'awaiting_confirmation',
				'executing'
		  )
		ORDER BY updated_at DESC
		LIMIT 1
	`

	return scanCWRebuildDiscussion(
		database.DB.QueryRow(
			ctx,
			query,
			coursewareID,
			pageID,
			userID,
		),
	)
}

// GetCWRebuildDiscussionByIDForUser 按会话ID和创建者读取讨论。
func GetCWRebuildDiscussionByIDForUser(
	ctx context.Context,
	discussionID string,
	userID string,
) (*CoursewareRebuildDiscussion, error) {
	query := `
		SELECT
	` + cwRebuildDiscussionSelectColumns + `
		FROM courseware_rebuild_discussions
		WHERE id = $1
		  AND created_by = $2
	`

	return scanCWRebuildDiscussion(
		database.DB.QueryRow(
			ctx,
			query,
			discussionID,
			userID,
		),
	)
}

func encodeCWRebuildDiscussionMessage(
	message CoursewareRebuildDiscussionMessage,
) (string, error) {
	payload, err := json.Marshal(
		[]CoursewareRebuildDiscussionMessage{
			message,
		},
	)
	if err != nil {
		return "", fmt.Errorf(
			"编码课件重构讨论消息失败: %w",
			err,
		)
	}

	return string(payload), nil
}

// AppendTeacherCWRebuildDiscussionMessage 追加老师消息。
//
// 老师继续补充后，先前的“待确认”结论自动失效，状态回到discussing，
// 并清空旧final_instruction，确保最终执行始终对应最新一轮讨论。
func AppendTeacherCWRebuildDiscussionMessage(
	ctx context.Context,
	discussionID string,
	userID string,
	message CoursewareRebuildDiscussionMessage,
) (*CoursewareRebuildDiscussion, error) {
	messageJSON, err :=
		encodeCWRebuildDiscussionMessage(
			message,
		)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE courseware_rebuild_discussions
		SET messages =
				messages || $3::jsonb,
			status = 'discussing',
			final_instruction = '',
			ai_summary = '',
			error_message = '',
			updated_at = NOW()
		WHERE id = $1
		  AND created_by = $2
		  AND status IN (
				'discussing',
				'awaiting_confirmation'
		  )
		RETURNING
	` + cwRebuildDiscussionSelectColumns

	item, scanErr := scanCWRebuildDiscussion(
		database.DB.QueryRow(
			ctx,
			query,
			discussionID,
			userID,
			messageJSON,
		),
	)
	if errors.Is(
		scanErr,
		ErrCWRebuildDiscussionNotFound,
	) {
		return nil, ErrCWRebuildDiscussionConflict
	}

	return item, scanErr
}

// AppendAssistantCWRebuildDiscussionMessage 追加AI正式回复，并更新确认状态。
func AppendAssistantCWRebuildDiscussionMessage(
	ctx context.Context,
	discussionID string,
	userID string,
	message CoursewareRebuildDiscussionMessage,
	status string,
	finalInstruction string,
	aiSummary string,
	errorMessage string,
) (*CoursewareRebuildDiscussion, error) {
	messageJSON, err :=
		encodeCWRebuildDiscussionMessage(
			message,
		)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE courseware_rebuild_discussions
		SET messages =
				messages || $3::jsonb,
			status = $4,
			final_instruction = $5,
			ai_summary = $6,
			error_message = $7,
			updated_at = NOW()
		WHERE id = $1
		  AND created_by = $2
		  AND status = 'discussing'
		RETURNING
	` + cwRebuildDiscussionSelectColumns

	item, scanErr := scanCWRebuildDiscussion(
		database.DB.QueryRow(
			ctx,
			query,
			discussionID,
			userID,
			messageJSON,
			status,
			finalInstruction,
			aiSummary,
			errorMessage,
		),
	)
	if errors.Is(
		scanErr,
		ErrCWRebuildDiscussionNotFound,
	) {
		return nil, ErrCWRebuildDiscussionConflict
	}

	return item, scanErr
}

// SetCWRebuildDiscussionError 保存AI讨论调用错误，不删除老师已经提交的消息。
func SetCWRebuildDiscussionError(
	ctx context.Context,
	discussionID string,
	userID string,
	errorMessage string,
) error {
	commandTag, err := database.DB.Exec(
		ctx,
		`
			UPDATE courseware_rebuild_discussions
			SET error_message = $3,
				updated_at = NOW()
			WHERE id = $1
			  AND created_by = $2
			  AND status = 'discussing'
		`,
		discussionID,
		userID,
		errorMessage,
	)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrCWRebuildDiscussionConflict
	}

	return nil
}

// MarkCWRebuildDiscussionStale 将基于旧页面的讨论标记为过期。
func MarkCWRebuildDiscussionStale(
	ctx context.Context,
	discussionID string,
	userID string,
	message string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`
			UPDATE courseware_rebuild_discussions
			SET status = 'stale',
				error_message = $3,
				updated_at = NOW()
			WHERE id = $1
			  AND created_by = $2
			  AND status IN (
					'discussing',
					'awaiting_confirmation'
			  )
		`,
		discussionID,
		userID,
		message,
	)

	return err
}

// MarkCWRebuildDiscussionExecuting 原子取得执行权。
//
// 只有awaiting_confirmation且final_instruction非空时才能进入executing。
// 重复点击确认时，只有第一个请求能更新成功。
func MarkCWRebuildDiscussionExecuting(
	ctx context.Context,
	discussionID string,
	userID string,
) (*CoursewareRebuildDiscussion, error) {
	query := `
		UPDATE courseware_rebuild_discussions
		SET status = 'executing',
			confirmed_at = NOW(),
			error_message = '',
			updated_at = NOW()
		WHERE id = $1
		  AND created_by = $2
		  AND status = 'awaiting_confirmation'
		  AND BTRIM(final_instruction) <> ''
		RETURNING
	` + cwRebuildDiscussionSelectColumns

	item, err := scanCWRebuildDiscussion(
		database.DB.QueryRow(
			ctx,
			query,
			discussionID,
			userID,
		),
	)
	if errors.Is(
		err,
		ErrCWRebuildDiscussionNotFound,
	) {
		return nil, ErrCWRebuildDiscussionConflict
	}

	return item, err
}

// MarkCWRebuildDiscussionCompleted 标记重构执行成功。
func MarkCWRebuildDiscussionCompleted(
	ctx context.Context,
	discussionID string,
	userID string,
) (*CoursewareRebuildDiscussion, error) {
	query := `
		UPDATE courseware_rebuild_discussions
		SET status = 'completed',
			executed_at = NOW(),
			error_message = '',
			updated_at = NOW()
		WHERE id = $1
		  AND created_by = $2
		  AND status = 'executing'
		RETURNING
	` + cwRebuildDiscussionSelectColumns

	item, err := scanCWRebuildDiscussion(
		database.DB.QueryRow(
			ctx,
			query,
			discussionID,
			userID,
		),
	)
	if errors.Is(
		err,
		ErrCWRebuildDiscussionNotFound,
	) {
		return nil, ErrCWRebuildDiscussionConflict
	}

	return item, err
}

// RestoreCWRebuildDiscussionAfterExecutionFailure 允许老师在生成失败后再次确认重试。
func RestoreCWRebuildDiscussionAfterExecutionFailure(
	ctx context.Context,
	discussionID string,
	userID string,
	errorMessage string,
) error {
	commandTag, err := database.DB.Exec(
		ctx,
		`
			UPDATE courseware_rebuild_discussions
			SET status = 'awaiting_confirmation',
				error_message = $3,
				updated_at = NOW()
			WHERE id = $1
			  AND created_by = $2
			  AND status = 'executing'
		`,
		discussionID,
		userID,
		errorMessage,
	)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrCWRebuildDiscussionConflict
	}

	return nil
}

// CancelCWRebuildDiscussion 取消尚未执行的讨论。
func CancelCWRebuildDiscussion(
	ctx context.Context,
	discussionID string,
	userID string,
) (*CoursewareRebuildDiscussion, error) {
	query := `
		UPDATE courseware_rebuild_discussions
		SET status = 'cancelled',
			error_message = '',
			updated_at = NOW()
		WHERE id = $1
		  AND created_by = $2
		  AND status IN (
				'discussing',
				'awaiting_confirmation'
		  )
		RETURNING
	` + cwRebuildDiscussionSelectColumns

	item, err := scanCWRebuildDiscussion(
		database.DB.QueryRow(
			ctx,
			query,
			discussionID,
			userID,
		),
	)
	if errors.Is(
		err,
		ErrCWRebuildDiscussionNotFound,
	) {
		return nil, ErrCWRebuildDiscussionConflict
	}

	return item, err
}
