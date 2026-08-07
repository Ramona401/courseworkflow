package repository

// courseware_style_message_repo.go — AI美术风格工作室消息仓储
//
// 本文件负责：
//   - 在会话行锁保护下分配严格递增的消息序号；
//   - 单独追加消息；
//   - 原子保存一轮“老师消息 + AI回复 + 新IAOCI草稿”；
//   - 新一轮风格修改后把旧预览标记为stale；
//   - 已确认或归档的会话禁止追加消息。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var ErrCoursewareStyleMessageInvalid = errors.New("课件风格消息无效")

const coursewareStyleMessageSelectColumns = `
id,
session_id,
courseware_id,
role,
content,
reference_asset_id,
style_aoci_text,
sequence_no,
created_at`

func scanCoursewareStyleMessage(
	scanner interface {
		Scan(dest ...interface{}) error
	},
) (*models.CoursewareStyleMessage, error) {
	item := &models.CoursewareStyleMessage{}

	err := scanner.Scan(
		&item.ID,
		&item.SessionID,
		&item.CoursewareID,
		&item.Role,
		&item.Content,
		&item.ReferenceAssetID,
		&item.StyleAOCIText,
		&item.SequenceNo,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}

// AppendCoursewareStyleMessage 追加一条消息。
func AppendCoursewareStyleMessage(
	ctx context.Context,
	userID string,
	item *models.CoursewareStyleMessage,
) error {
	if err := validateCoursewareStyleMessage(
		item,
	); err != nil {
		return err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启风格消息事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockEditableCoursewareStyleSessionTx(
		ctx,
		tx,
		item.CoursewareID,
		item.SessionID,
		userID,
	); err != nil {
		return err
	}

	if err := validateCoursewareStyleAssetTx(
		ctx,
		tx,
		item.CoursewareID,
		item.ReferenceAssetID,
	); err != nil {
		return err
	}

	sequenceNo, err :=
		nextCoursewareStyleMessageSequenceTx(
			ctx,
			tx,
			item.SessionID,
		)
	if err != nil {
		return err
	}

	item.SequenceNo = sequenceNo

	err = tx.QueryRow(
		ctx,
		`INSERT INTO courseware_style_messages (
session_id,
courseware_id,
role,
content,
reference_asset_id,
style_aoci_text,
sequence_no
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at`,
		item.SessionID,
		item.CoursewareID,
		item.Role,
		strings.TrimSpace(item.Content),
		nullableCoursewareStyleString(
			item.ReferenceAssetID,
		),
		strings.TrimSpace(item.StyleAOCIText),
		item.SequenceNo,
	).Scan(
		&item.ID,
		&item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"追加风格消息失败: %w",
			err,
		)
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE courseware_style_sessions
SET updated_at = now()
WHERE id = $1
  AND courseware_id = $2`,
		item.SessionID,
		item.CoursewareID,
	); err != nil {
		return fmt.Errorf(
			"更新风格会话时间失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交风格消息事务失败: %w",
			err,
		)
	}

	return nil
}

// AppendCoursewareStyleTurn 原子保存一轮完整对话。
//
// 同一事务内：
//  1. 保存老师消息；
//  2. 保存AI回复及完整IAOCI快照；
//  3. 更新会话当前IAOCI和摘要；
//  4. 将此前三类预览全部标记为stale。
func AppendCoursewareStyleTurn(
	ctx context.Context,
	userID string,
	userMessage *models.CoursewareStyleMessage,
	assistantMessage *models.CoursewareStyleMessage,
	referenceMode string,
	referenceAssetID *string,
	styleAOCIText string,
	styleSummary string,
) (*models.CoursewareStyleSession, error) {
	if err := validateCoursewareStyleTurn(
		userMessage,
		assistantMessage,
		referenceMode,
		styleAOCIText,
	); err != nil {
		return nil, err
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开启风格对话事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := lockEditableCoursewareStyleSessionTx(
		ctx,
		tx,
		userMessage.CoursewareID,
		userMessage.SessionID,
		userID,
	); err != nil {
		return nil, err
	}

	if err := validateCoursewareStyleAssetTx(
		ctx,
		tx,
		userMessage.CoursewareID,
		referenceAssetID,
	); err != nil {
		return nil, err
	}

	if err := validateCoursewareStyleAssetTx(
		ctx,
		tx,
		userMessage.CoursewareID,
		userMessage.ReferenceAssetID,
	); err != nil {
		return nil, err
	}

	firstSequence, err :=
		nextCoursewareStyleMessageSequenceTx(
			ctx,
			tx,
			userMessage.SessionID,
		)
	if err != nil {
		return nil, err
	}

	userMessage.SequenceNo = firstSequence
	assistantMessage.SequenceNo = firstSequence + 1
	assistantMessage.StyleAOCIText =
		strings.TrimSpace(styleAOCIText)

	if err := insertCoursewareStyleMessageTx(
		ctx,
		tx,
		userMessage,
	); err != nil {
		return nil, err
	}

	if err := insertCoursewareStyleMessageTx(
		ctx,
		tx,
		assistantMessage,
	); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE courseware_style_previews
SET status = $1,
	last_error = '',
	version = version + 1,
	updated_at = now()
WHERE session_id = $2
  AND courseware_id = $3
  AND status <> $1`,
		models.CWStylePreviewStatusStale,
		userMessage.SessionID,
		userMessage.CoursewareID,
	); err != nil {
		return nil, fmt.Errorf(
			"标记旧风格预览过期失败: %w",
			err,
		)
	}

	session, err := scanCoursewareStyleSession(
		tx.QueryRow(
			ctx,
			`UPDATE courseware_style_sessions
SET status = $1,
	reference_mode = $2,
	reference_asset_id = COALESCE($3, reference_asset_id),
	style_aoci_text = $4,
	style_summary = $5,
	version = version + 1,
	updated_at = now()
WHERE id = $6
  AND courseware_id = $7
  AND user_id = $8
RETURNING `+coursewareStyleSessionSelectColumns,
			models.CWStyleSessionStatusDraft,
			referenceMode,
			nullableCoursewareStyleString(
				referenceAssetID,
			),
			strings.TrimSpace(styleAOCIText),
			strings.TrimSpace(styleSummary),
			userMessage.SessionID,
			userMessage.CoursewareID,
			userID,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"更新风格会话草稿失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交风格对话事务失败: %w",
			err,
		)
	}

	return session, nil
}

// ListCoursewareStyleMessages 按序号返回会话消息。
func ListCoursewareStyleMessages(
	ctx context.Context,
	coursewareID string,
	sessionID string,
	userID string,
) ([]*models.CoursewareStyleMessage, error) {
	if _, err := GetCoursewareStyleSessionByID(
		ctx,
		coursewareID,
		sessionID,
		userID,
	); err != nil {
		return nil, err
	}

	rows, err := database.DB.Query(
		ctx,
		`SELECT `+
			coursewareStyleMessageSelectColumns+
			` FROM courseware_style_messages
WHERE session_id = $1
  AND courseware_id = $2
ORDER BY sequence_no ASC`,
		sessionID,
		coursewareID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询风格消息失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CoursewareStyleMessage,
		0,
	)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareStyleMessage(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描风格消息失败: %w",
				scanErr,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历风格消息失败: %w",
			err,
		)
	}

	return items, nil
}

func validateCoursewareStyleMessage(
	item *models.CoursewareStyleMessage,
) error {
	if item == nil {
		return ErrCoursewareStyleMessageInvalid
	}

	item.SessionID =
		strings.TrimSpace(item.SessionID)
	item.CoursewareID =
		strings.TrimSpace(item.CoursewareID)
	item.Role =
		strings.TrimSpace(item.Role)
	item.Content =
		strings.TrimSpace(item.Content)
	item.StyleAOCIText =
		strings.TrimSpace(item.StyleAOCIText)

	if item.SessionID == "" ||
		item.CoursewareID == "" {
		return fmt.Errorf(
			"%w：会话ID或课件ID为空",
			ErrCoursewareStyleMessageInvalid,
		)
	}

	if !models.IsValidCWStyleMessageRole(
		item.Role,
	) {
		return fmt.Errorf(
			"%w：消息角色不合法",
			ErrCoursewareStyleMessageInvalid,
		)
	}

	if item.Content == "" &&
		(item.ReferenceAssetID == nil ||
			strings.TrimSpace(
				*item.ReferenceAssetID,
			) == "") {
		return fmt.Errorf(
			"%w：消息文字和参考图片不能同时为空",
			ErrCoursewareStyleMessageInvalid,
		)
	}

	return nil
}

func validateCoursewareStyleTurn(
	userMessage *models.CoursewareStyleMessage,
	assistantMessage *models.CoursewareStyleMessage,
	referenceMode string,
	styleAOCIText string,
) error {
	if err := validateCoursewareStyleMessage(
		userMessage,
	); err != nil {
		return err
	}

	if err := validateCoursewareStyleMessage(
		assistantMessage,
	); err != nil {
		return err
	}

	if userMessage.Role !=
		models.CWStyleMessageRoleUser {
		return fmt.Errorf(
			"一轮对话的第一条必须是user消息",
		)
	}

	if assistantMessage.Role !=
		models.CWStyleMessageRoleAssistant {
		return fmt.Errorf(
			"一轮对话的第二条必须是assistant消息",
		)
	}

	if userMessage.SessionID !=
		assistantMessage.SessionID ||
		userMessage.CoursewareID !=
			assistantMessage.CoursewareID {
		return fmt.Errorf(
			"同一轮对话的会话和课件必须一致",
		)
	}

	if !models.IsValidCWStyleReferenceMode(
		referenceMode,
	) {
		return fmt.Errorf(
			"参考图模式不合法: %s",
			referenceMode,
		)
	}

	if strings.TrimSpace(styleAOCIText) == "" {
		return fmt.Errorf(
			"AI回复必须形成完整IAOCI草稿",
		)
	}

	return nil
}

func nextCoursewareStyleMessageSequenceTx(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
) (int, error) {
	var sequenceNo int

	err := tx.QueryRow(
		ctx,
		`SELECT COALESCE(MAX(sequence_no), 0) + 1
FROM courseware_style_messages
WHERE session_id = $1`,
		sessionID,
	).Scan(&sequenceNo)
	if err != nil {
		return 0, fmt.Errorf(
			"分配风格消息序号失败: %w",
			err,
		)
	}

	return sequenceNo, nil
}

func insertCoursewareStyleMessageTx(
	ctx context.Context,
	tx pgx.Tx,
	item *models.CoursewareStyleMessage,
) error {
	err := tx.QueryRow(
		ctx,
		`INSERT INTO courseware_style_messages (
session_id,
courseware_id,
role,
content,
reference_asset_id,
style_aoci_text,
sequence_no
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at`,
		item.SessionID,
		item.CoursewareID,
		item.Role,
		strings.TrimSpace(item.Content),
		nullableCoursewareStyleString(
			item.ReferenceAssetID,
		),
		strings.TrimSpace(item.StyleAOCIText),
		item.SequenceNo,
	).Scan(
		&item.ID,
		&item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"保存风格消息失败: %w",
			err,
		)
	}

	return nil
}
