package repository

// courseware_review_recheck_repo.go
//
// 页面发生后续变化后，由课件作者重新检查当前页面的原子仓储。
//
// 适用范围：
//
//   - 作者自审问题；
//   - 已经随正式审核结果交付作者的正式整改问题。
//
// 操作含义：
//
//   1. 作者已经打开并实际检查当前页面；
//   2. 作者确认当前页面仍然满足既有修改方案或正式整改要求；
//   3. 系统把当前页面HTML重新登记为本条问题的修改完成结果；
//   4. 问题回到applied，等待后续人工确认；
//   5. 自审问题仍需作者另行确认解决；
//   6. 正式问题仍需审核员复审确认解决。
//
// 本入口不会：
//
//   - 修改课件页面；
//   - 改写原审核问题和整改要求；
//   - 自动把问题标记为resolved；
//   - 绕过正式复审。
//
// 安全边界：
//
//   - 只有课件作者本人能够操作；
//   - 问题必须处于stale；
//   - 必须存在已确认的修改方案或整改要求；
//   - 必须绑定稳定页面；
//   - 课件必须仍处于作者可修改阶段；
//   - 页面读取、问题状态更新和系统记录位于同一事务。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var ErrCoursewareReviewItemRecheckInvalid = errors.New(
	"当前问题不能重新登记为修改完成",
)

// RecheckCoursewareReviewItem 原子记录作者对当前页面的重新检查结果。
func RecheckCoursewareReviewItem(
	ctx context.Context,
	itemID string,
	ownerID string,
) error {
	itemID =
		strings.TrimSpace(
			itemID,
		)

	ownerID =
		strings.TrimSpace(
			ownerID,
		)

	if itemID == "" ||
		ownerID == "" {
		return ErrCoursewareReviewItemNotFound
	}

	tx, err :=
		database.DB.Begin(
			ctx,
		)
	if err != nil {
		return fmt.Errorf(
			"开始课件整改重新检查事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		sessionID            string
		coursewareID         string
		sourceType           string
		currentStatus        string
		pageID               string
		confirmedInstruction string
		coursewareReviewID   string
		feedbackID           string
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			source_session_id,
			courseware_id,
			source_type,
			status,
			COALESCE(page_id::text, ''),
			COALESCE(confirmed_instruction, ''),
			COALESCE(courseware_review_id::text, ''),
			COALESCE(feedback_id::text, '')
		FROM courseware_review_items
		WHERE id = $1
		  AND owner_id = $2
		FOR UPDATE`,
		itemID,
		ownerID,
	).Scan(
		&sessionID,
		&coursewareID,
		&sourceType,
		&currentStatus,
		&pageID,
		&confirmedInstruction,
		&coursewareReviewID,
		&feedbackID,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrCoursewareReviewItemNotFound
		}

		return fmt.Errorf(
			"锁定待重新检查的课件整改问题失败: %w",
			err,
		)
	}

	if currentStatus !=
		models.CWReviewItemStatusStale ||
		strings.TrimSpace(
			pageID,
		) == "" ||
		strings.TrimSpace(
			confirmedInstruction,
		) == "" {
		return ErrCoursewareReviewItemRecheckInvalid
	}

	switch sourceType {
	case models.CWReviewItemSourceSelf:
		if coursewareReviewID != "" ||
			feedbackID != "" {
			return ErrCoursewareReviewItemRecheckInvalid
		}

	case models.CWReviewItemSourceFormal:
		if coursewareReviewID == "" ||
			feedbackID == "" {
			return ErrCoursewareReviewItemRecheckInvalid
		}

	default:
		return ErrCoursewareReviewItemRecheckInvalid
	}

	var publishState string

	err = tx.QueryRow(
		ctx,
		`
		SELECT COALESCE(publish_state, 'private')
		FROM coursewares
		WHERE id = $1
		  AND user_id = $2
		  AND deleted_at IS NULL
		FOR SHARE`,
		coursewareID,
		ownerID,
	).Scan(
		&publishState,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrCoursewareReviewItemNotFound
		}

		return fmt.Errorf(
			"读取课件重新检查状态失败: %w",
			err,
		)
	}

	switch publishState {
	case models.CWPublishPrivate,
		models.CWPublishPublishedPersonal,
		models.CWPublishRevision:
	default:
		return ErrCoursewareReviewItemRecheckInvalid
	}

	var currentHTML string

	err = tx.QueryRow(
		ctx,
		`
		SELECT COALESCE(html_content, '')
		FROM courseware_pages
		WHERE id = $1
		  AND courseware_id = $2
		FOR SHARE`,
		pageID,
		coursewareID,
	).Scan(
		&currentHTML,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			result, updateErr :=
				tx.Exec(
					ctx,
					`
					UPDATE courseware_review_items
					SET
						status = 'orphaned',
						resolved_at = NULL,
						resolved_by = NULL,
						resolved_review_id = NULL,
						resolved_review_level = 0,
						resolved_review_round = 0,
						resolution_note = '',
						updated_at = NOW()
					WHERE id = $1
					  AND owner_id = $2
					  AND status = 'stale'`,
					itemID,
					ownerID,
				)
			if updateErr != nil {
				return fmt.Errorf(
					"记录整改问题原页面删除失败: %w",
					updateErr,
				)
			}
			if result.RowsAffected() != 1 {
				return ErrCoursewareReviewItemConflict
			}

			if eventErr :=
				appendCoursewareReviewItemStateEventTx(
					ctx,
					tx,
					sessionID,
					itemID,
					ownerID,
					"重新检查时发现原问题页面已经删除，无法登记为修改完成。",
					cwReviewItemStateEventMeta{
						Event:          models.CWReviewItemStatusOrphaned,
						PreviousStatus: models.CWReviewItemStatusStale,
						NextStatus:     models.CWReviewItemStatusOrphaned,
						Reason:         "重新检查时发现原问题页面已经删除",
					},
				); eventErr != nil {
				return eventErr
			}

			if commitErr :=
				tx.Commit(
					ctx,
				); commitErr != nil {
				return fmt.Errorf(
					"提交整改问题页面删除状态失败: %w",
					commitErr,
				)
			}

			return ErrCoursewareReviewItemAppliedPageMissing
		}

		return fmt.Errorf(
			"读取重新检查页面失败: %w",
			err,
		)
	}

	currentHash :=
		cwReviewItemContentHash(
			currentHTML,
		)

	result, err :=
		tx.Exec(
			ctx,
			`
			UPDATE courseware_review_items
			SET
				status = 'applied',
				applied_at = NOW(),
				applied_page_hash = $3,
				resolved_at = NULL,
				resolved_by = NULL,
				resolved_review_id = NULL,
				resolved_review_level = 0,
				resolved_review_round = 0,
				resolution_note = '',
				updated_at = NOW()
			WHERE id = $1
			  AND owner_id = $2
			  AND status = 'stale'
			  AND BTRIM(
			      COALESCE(
			          confirmed_instruction,
			          ''
			      )
			  ) <> ''`,
			itemID,
			ownerID,
			currentHash,
		)
	if err != nil {
		return fmt.Errorf(
			"登记课件整改重新检查结果失败: %w",
			err,
		)
	}

	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	message :=
		"作者重新检查当前页面，确认页面仍符合既有修改方案，现等待作者最终确认。"

	if sourceType ==
		models.CWReviewItemSourceFormal {
		message =
			"作者重新检查当前页面，确认页面仍符合正式整改要求，现等待审核员复审。"
	}

	if err :=
		appendCoursewareReviewItemStateEventTx(
			ctx,
			tx,
			sessionID,
			itemID,
			ownerID,
			message,
			cwReviewItemStateEventMeta{
				Event:          "rechecked",
				PreviousStatus: models.CWReviewItemStatusStale,
				NextStatus:     models.CWReviewItemStatusApplied,
				Reason:         "作者重新检查当前页面并确认仍符合既有要求",
			},
		); err != nil {
		return err
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return fmt.Errorf(
			"提交课件整改重新检查事务失败: %w",
			err,
		)
	}

	return nil
}
