package repository

// courseware_review_item_application_state_repo.go
//
// 页面应用过程中的状态回退与作者自审“继续调整”恢复凭据。
//
// 首次微调失败时清除临时版本绑定并回到confirmed；
// 自审再次微调真正开始时记录上一轮applied时间，失败时恢复上一轮applied事实。
// 页面已经变化或删除时不会错误恢复applied，而是转为stale/orphaned。

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

const cwSelfReviewReapplyStartedEvent = "self_reapply_started"

type cwSelfReviewReapplyStartMeta struct {
	Event             string `json:"event"`
	PreviousStatus    string `json:"previous_status"`
	NextStatus        string `json:"next_status"`
	PreviousAppliedAt string `json:"previous_applied_at"`
}

// AbortCoursewareReviewItemInitialApplication 取消首次页面微调的临时版本绑定。
//
// confirmed首次进入applying时尚未形成applied事实，失败后必须同时清除
// applied_instruction_version_id，才能满足数据库约束并安全回到confirmed。
func AbortCoursewareReviewItemInitialApplication(
	ctx context.Context,
	itemID string,
	ownerID string,
) error {
	result, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = 'confirmed',
			applied_instruction_version_id = NULL,
			updated_at = NOW()
		WHERE id = $1
		  AND owner_id = $2
		  AND status = 'applying'
		  AND applied_instruction_version_id IS NOT NULL
		  AND applied_at IS NULL
		  AND BTRIM(COALESCE(applied_page_hash, '')) = ''`,
		strings.TrimSpace(itemID),
		strings.TrimSpace(ownerID),
	)
	if err != nil {
		return mapCoursewareReviewItemApplicationWriteError(
			"取消课件整改项首次页面应用失败",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	return nil
}

// InvalidateCoursewareReviewItemInitialApplication 将首次应用中的问题标记为页面已变化。
//
// 尚未形成applied事实，因此必须清除临时应用版本绑定后再进入stale/orphaned。
func InvalidateCoursewareReviewItemInitialApplication(
	ctx context.Context,
	itemID string,
	ownerID string,
	nextStatus string,
) error {
	if nextStatus != models.CWReviewItemStatusStale &&
		nextStatus != models.CWReviewItemStatusOrphaned {
		return ErrCoursewareReviewItemConflict
	}

	result, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = $3,
			applied_instruction_version_id = NULL,
			updated_at = NOW()
		WHERE id = $1
		  AND owner_id = $2
		  AND status = 'applying'
		  AND applied_instruction_version_id IS NOT NULL
		  AND applied_at IS NULL
		  AND BTRIM(COALESCE(applied_page_hash, '')) = ''`,
		strings.TrimSpace(itemID),
		strings.TrimSpace(ownerID),
		nextStatus,
	)
	if err != nil {
		return mapCoursewareReviewItemApplicationWriteError(
			"标记首次页面应用失效状态失败",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	return nil
}

func appendSelfCoursewareReviewItemReapplyStartedTx(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	itemID string,
	ownerID string,
	previousAppliedAt time.Time,
) error {
	metaJSON, err := json.Marshal(
		cwSelfReviewReapplyStartMeta{
			Event:             cwSelfReviewReapplyStartedEvent,
			PreviousStatus:    models.CWReviewItemStatusApplied,
			NextStatus:        models.CWReviewItemStatusApplying,
			PreviousAppliedAt: previousAppliedAt.UTC().Format(time.RFC3339Nano),
		},
	)
	if err != nil {
		return fmt.Errorf(
			"序列化作者自审继续调整事件失败: %w",
			err,
		)
	}

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO courseware_ai_review_messages (
			session_id,
			review_item_id,
			user_id,
			role,
			content,
			citations_json,
			tokens_used,
			model_used,
			created_at
		)
		VALUES (
			$1,
			$2,
			$3,
			'system',
			$4,
			$5::jsonb,
			0,
			'',
			NOW()
		)`,
		strings.TrimSpace(sessionID),
		strings.TrimSpace(itemID),
		strings.TrimSpace(ownerID),
		"已继续调整这条自审问题，完成本次页面修改后仍需再次检查。",
		string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf(
			"记录作者自审继续调整事件失败: %w",
			err,
		)
	}

	return nil
}

func readSelfCoursewareReviewPreviousAppliedAtTx(
	ctx context.Context,
	tx pgx.Tx,
	itemID string,
) (*time.Time, error) {
	var previousAppliedAt time.Time

	err := tx.QueryRow(
		ctx,
		`
		SELECT
			(citations_json ->> 'previous_applied_at')::timestamptz
		FROM courseware_ai_review_messages
		WHERE review_item_id = $1
		  AND role = 'system'
		  AND citations_json ->> 'event' = $2
		  AND BTRIM(
			COALESCE(
				citations_json ->> 'previous_applied_at',
				''
			)
		  ) <> ''
		ORDER BY created_at DESC, id DESC
		LIMIT 1`,
		strings.TrimSpace(itemID),
		cwSelfReviewReapplyStartedEvent,
	).Scan(&previousAppliedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewItemConflict
		}

		return nil, fmt.Errorf(
			"读取作者自审上次修改完成时间失败: %w",
			err,
		)
	}

	return &previousAppliedAt, nil
}

func invalidateSelfCoursewareReviewReapplyTx(
	ctx context.Context,
	tx pgx.Tx,
	record *cwSelfReviewPostApplyRecord,
	ownerID string,
	previousAppliedAt time.Time,
	nextStatus string,
	message string,
) error {
	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = $3,
			applied_at = $4,
			resolved_at = NULL,
			resolved_by = NULL,
			resolved_review_id = NULL,
			resolved_review_level = 0,
			resolved_review_round = 0,
			resolution_note = '',
			updated_at = NOW()
		WHERE id = $1
		  AND owner_id = $2
		  AND source_type = 'self'
		  AND status = 'applying'
		  AND applied_instruction_version_id IS NOT NULL
		  AND applied_at IS NULL
		  AND BTRIM(COALESCE(applied_page_hash, '')) <> ''
		  AND courseware_review_id IS NULL
		  AND feedback_id IS NULL`,
		record.ID,
		strings.TrimSpace(ownerID),
		nextStatus,
		previousAppliedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"恢复作者自审继续调整前状态失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	return appendCoursewareReviewItemStateEventTx(
		ctx,
		tx,
		record.SessionID,
		record.ID,
		ownerID,
		message,
		cwReviewItemStateEventMeta{
			Event:          nextStatus,
			PreviousStatus: models.CWReviewItemStatusApplying,
			NextStatus:     nextStatus,
			Reason:         message,
		},
	)
}

// RestoreSelfCoursewareReviewItemAfterReapplyAbort 在再次微调失败后恢复上一轮applied事实。
//
// 上一次applied时间由服务端开始再次微调时写入的审计事件恢复；
// 页面若已发生额外变化，则改记为stale/orphaned而不是错误恢复applied。
func RestoreSelfCoursewareReviewItemAfterReapplyAbort(
	ctx context.Context,
	itemID string,
	ownerID string,
) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开始恢复作者自审继续调整状态事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	record, err :=
		lockSelfCoursewareReviewPostApplyRecord(
			ctx,
			tx,
			itemID,
			ownerID,
		)
	if err != nil {
		return err
	}

	if record.SourceType != models.CWReviewItemSourceSelf ||
		record.OwnerID != strings.TrimSpace(ownerID) ||
		record.AlreadyDelivered ||
		record.Status != models.CWReviewItemStatusApplying ||
		record.AppliedAt != nil ||
		strings.TrimSpace(record.AppliedPageHash) == "" ||
		strings.TrimSpace(record.AppliedInstructionVersionID) == "" {
		return ErrCoursewareReviewItemConflict
	}

	previousAppliedAt, err :=
		readSelfCoursewareReviewPreviousAppliedAtTx(
			ctx,
			tx,
			record.ID,
		)
	if err != nil {
		return err
	}

	currentHTML, pageErr :=
		lockSelfCoursewareReviewPostApplyPageHTML(
			ctx,
			tx,
			record,
		)
	if pageErr != nil {
		if errors.Is(
			pageErr,
			ErrCoursewareReviewItemAppliedPageMissing,
		) {
			if invalidateErr :=
				invalidateSelfCoursewareReviewReapplyTx(
					ctx,
					tx,
					record,
					ownerID,
					*previousAppliedAt,
					models.CWReviewItemStatusOrphaned,
					"原页面已不存在，需要人工重新检查相关页面后再继续处理。",
				); invalidateErr != nil {
				return invalidateErr
			}

			if commitErr := tx.Commit(ctx); commitErr != nil {
				return fmt.Errorf(
					"提交作者自审继续调整页面删除状态失败: %w",
					commitErr,
				)
			}
		}

		return pageErr
	}

	if strings.TrimSpace(record.PageID) != "" &&
		cwReviewItemContentHash(currentHTML) !=
			strings.TrimSpace(record.AppliedPageHash) {
		if err :=
			invalidateSelfCoursewareReviewReapplyTx(
				ctx,
				tx,
				record,
				ownerID,
				*previousAppliedAt,
				models.CWReviewItemStatusStale,
				"页面内容已变化，需要人工重新检查当前页面后再继续处理。",
			); err != nil {
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf(
				"提交作者自审继续调整页面变化状态失败: %w",
				err,
			)
		}

		return ErrCoursewareReviewItemAppliedPageChanged
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = 'applied',
			applied_at = $3,
			updated_at = NOW()
		WHERE id = $1
		  AND owner_id = $2
		  AND source_type = 'self'
		  AND status = 'applying'
		  AND applied_instruction_version_id IS NOT NULL
		  AND applied_at IS NULL
		  AND BTRIM(COALESCE(applied_page_hash, '')) <> ''
		  AND courseware_review_id IS NULL
		  AND feedback_id IS NULL`,
		record.ID,
		strings.TrimSpace(ownerID),
		*previousAppliedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"恢复作者自审上次修改完成状态失败: %w",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	if err :=
		appendCoursewareReviewItemStateEventTx(
			ctx,
			tx,
			record.SessionID,
			record.ID,
			ownerID,
			"本次继续调整未完成，已保留上一次修改完成状态。",
			cwReviewItemStateEventMeta{
				Event:          "reapply_aborted",
				PreviousStatus: models.CWReviewItemStatusApplying,
				NextStatus:     models.CWReviewItemStatusApplied,
			},
		); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交恢复作者自审上次修改完成状态事务失败: %w",
			err,
		)
	}

	return nil
}
