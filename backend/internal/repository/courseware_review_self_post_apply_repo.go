package repository

// courseware_review_self_post_apply_repo.go
//
// 作者自审问题在“修改完成，等待检查”之后的人工分支仓储。
//
// 本文件只处理source=self且仍未进入正式审核历史的applied事实：
//
//   1. “暂时不处理”：applied -> dismissed，保留applied版本、时间和页面指纹；
//   2. “恢复调整”：dismissed -> applied，继续等待检查或再次调整；
//   3. 页面变化或删除时转为stale/orphaned，并保留原有修改证据和历史。
//
// 浏览器不参与目标状态、页面HTML、页面指纹或回滚时间的决定。
// page_id因ON DELETE SET NULL变空时，page_number_snapshot用于区分：
//   - 0：真正整课问题；
//   - >0：原页级问题页面已经删除。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

type cwSelfReviewPostApplyRecord struct {
	ID                 string
	SessionID          string
	CoursewareID       string
	SourceType         string
	OwnerID            string
	Status             string
	PageID             string
	PageNumberSnapshot int

	AppliedAt                   *time.Time
	AppliedPageHash             string
	AppliedInstructionVersionID string
	AlreadyDelivered            bool
}

func lockSelfCoursewareReviewPostApplyRecord(
	ctx context.Context,
	tx pgx.Tx,
	itemID string,
	ownerID string,
) (*cwSelfReviewPostApplyRecord, error) {
	record := &cwSelfReviewPostApplyRecord{}

	err := tx.QueryRow(
		ctx,
		`
		SELECT
			id::text,
			source_session_id,
			courseware_id,
			source_type,
			owner_id,
			status,
			COALESCE(page_id::text, ''),
			page_number_snapshot,
			applied_at,
			COALESCE(applied_page_hash, ''),
			COALESCE(applied_instruction_version_id::text, ''),
			(
				courseware_review_id IS NOT NULL
				OR feedback_id IS NOT NULL
			)
		FROM courseware_review_items
		WHERE id = $1
		  AND owner_id = $2
		FOR UPDATE`,
		strings.TrimSpace(itemID),
		strings.TrimSpace(ownerID),
	).Scan(
		&record.ID,
		&record.SessionID,
		&record.CoursewareID,
		&record.SourceType,
		&record.OwnerID,
		&record.Status,
		&record.PageID,
		&record.PageNumberSnapshot,
		&record.AppliedAt,
		&record.AppliedPageHash,
		&record.AppliedInstructionVersionID,
		&record.AlreadyDelivered,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewItemNotFound
		}
		return nil, fmt.Errorf("锁定作者自审修改完成记录失败: %w", err)
	}

	return record, nil
}

func validateSelfCoursewareReviewPostApplyRecord(
	record *cwSelfReviewPostApplyRecord,
	ownerID string,
	expectedStatus string,
) error {
	if record == nil {
		return ErrCoursewareReviewItemNotFound
	}

	if record.SourceType != models.CWReviewItemSourceSelf ||
		record.OwnerID != strings.TrimSpace(ownerID) ||
		record.AlreadyDelivered ||
		record.Status != expectedStatus ||
		record.AppliedAt == nil ||
		strings.TrimSpace(record.AppliedPageHash) == "" ||
		strings.TrimSpace(record.AppliedInstructionVersionID) == "" {
		return ErrCoursewareReviewItemConflict
	}

	return nil
}

func lockSelfCoursewareReviewPostApplyPageHTML(
	ctx context.Context,
	tx pgx.Tx,
	record *cwSelfReviewPostApplyRecord,
) (string, error) {
	if record == nil {
		return "", ErrCoursewareReviewItemNotFound
	}

	pageID := strings.TrimSpace(record.PageID)
	if pageID == "" {
		if record.PageNumberSnapshot > 0 {
			return "", ErrCoursewareReviewItemAppliedPageMissing
		}
		return "", nil
	}

	var currentHTML string
	err := tx.QueryRow(
		ctx,
		`
		SELECT COALESCE(html_content, '')
		FROM courseware_pages
		WHERE id = $1
		  AND courseware_id = $2
		FOR SHARE`,
		pageID,
		strings.TrimSpace(record.CoursewareID),
	).Scan(&currentHTML)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrCoursewareReviewItemAppliedPageMissing
		}
		return "", fmt.Errorf("读取作者自审修改完成页面失败: %w", err)
	}

	return currentHTML, nil
}

func invalidateSelfCoursewareReviewPostApplyTx(
	ctx context.Context,
	tx pgx.Tx,
	record *cwSelfReviewPostApplyRecord,
	ownerID string,
	currentStatus string,
	nextStatus string,
	message string,
) error {
	if record == nil {
		return ErrCoursewareReviewItemNotFound
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = $4,
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
		  AND status = $3
		  AND applied_instruction_version_id IS NOT NULL
		  AND applied_at IS NOT NULL
		  AND BTRIM(COALESCE(applied_page_hash, '')) <> ''
		  AND courseware_review_id IS NULL
		  AND feedback_id IS NULL`,
		record.ID,
		strings.TrimSpace(ownerID),
		currentStatus,
		nextStatus,
	)
	if err != nil {
		return fmt.Errorf("更新作者自审修改完成状态失败: %w", err)
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
			PreviousStatus: currentStatus,
			NextStatus:     nextStatus,
			Reason:         message,
		},
	)
}

func verifySelfCoursewareReviewPostApplyPageTx(
	ctx context.Context,
	tx pgx.Tx,
	record *cwSelfReviewPostApplyRecord,
	ownerID string,
	currentStatus string,
) error {
	currentHTML, err := lockSelfCoursewareReviewPostApplyPageHTML(
		ctx,
		tx,
		record,
	)
	if err != nil {
		if errors.Is(err, ErrCoursewareReviewItemAppliedPageMissing) {
			if invalidateErr := invalidateSelfCoursewareReviewPostApplyTx(
				ctx,
				tx,
				record,
				ownerID,
				currentStatus,
				models.CWReviewItemStatusOrphaned,
				"原页面已不存在，需要人工重新检查相关页面后再继续处理。",
			); invalidateErr != nil {
				return invalidateErr
			}
		}
		return err
	}

	if strings.TrimSpace(record.PageID) == "" {
		return nil
	}

	if cwReviewItemContentHash(currentHTML) !=
		strings.TrimSpace(record.AppliedPageHash) {
		if err := invalidateSelfCoursewareReviewPostApplyTx(
			ctx,
			tx,
			record,
			ownerID,
			currentStatus,
			models.CWReviewItemStatusStale,
			"页面内容已变化，需要人工重新检查当前页面后再继续处理。",
		); err != nil {
			return err
		}
		return ErrCoursewareReviewItemAppliedPageChanged
	}

	return nil
}

// DismissAppliedSelfCoursewareReviewItem 将“修改完成，等待检查”的自审问题暂时搁置。
//
// applied事实完整保留，恢复时仍回到applied，而不是伪造回第一次修改前。
func DismissAppliedSelfCoursewareReviewItem(
	ctx context.Context,
	itemID string,
	ownerID string,
	reason string,
) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("暂时不处理原因不能为空")
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始作者自审暂时不处理事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	record, err := lockSelfCoursewareReviewPostApplyRecord(
		ctx,
		tx,
		itemID,
		ownerID,
	)
	if err != nil {
		return err
	}

	if err := validateSelfCoursewareReviewPostApplyRecord(
		record,
		ownerID,
		models.CWReviewItemStatusApplied,
	); err != nil {
		return err
	}

	if err := verifySelfCoursewareReviewPostApplyPageTx(
		ctx,
		tx,
		record,
		ownerID,
		models.CWReviewItemStatusApplied,
	); err != nil {
		if errors.Is(err, ErrCoursewareReviewItemAppliedPageChanged) ||
			errors.Is(err, ErrCoursewareReviewItemAppliedPageMissing) {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return fmt.Errorf(
					"提交作者自审页面重新检查状态失败: %w",
					commitErr,
				)
			}
		}
		return err
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = 'dismissed',
			updated_at = NOW()
		WHERE id = $1
		  AND owner_id = $2
		  AND source_type = 'self'
		  AND status = 'applied'
		  AND applied_instruction_version_id IS NOT NULL
		  AND applied_at = $3
		  AND BTRIM(COALESCE(applied_page_hash, '')) = $4
		  AND courseware_review_id IS NULL
		  AND feedback_id IS NULL`,
		record.ID,
		strings.TrimSpace(ownerID),
		record.AppliedAt,
		strings.TrimSpace(record.AppliedPageHash),
	)
	if err != nil {
		return fmt.Errorf("暂时搁置作者自审问题失败: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	if err := appendCoursewareReviewItemStateEventTx(
		ctx,
		tx,
		record.SessionID,
		record.ID,
		ownerID,
		"已暂时不处理这条自审问题。说明："+reason,
		cwReviewItemStateEventMeta{
			Event:          "dismissed",
			PreviousStatus: models.CWReviewItemStatusApplied,
			NextStatus:     models.CWReviewItemStatusDismissed,
			Reason:         reason,
		},
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交作者自审暂时不处理事务失败: %w", err)
	}

	return nil
}

// RestoreDismissedAppliedSelfCoursewareReviewItem 恢复曾经已经完成修改的自审问题。
//
// 恢复后仍是applied，教师可以继续选择“确认已经解决”或“继续调整”。
func RestoreDismissedAppliedSelfCoursewareReviewItem(
	ctx context.Context,
	itemID string,
	ownerID string,
) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始恢复作者自审修改完成状态事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	record, err := lockSelfCoursewareReviewPostApplyRecord(
		ctx,
		tx,
		itemID,
		ownerID,
	)
	if err != nil {
		return err
	}

	if err := validateSelfCoursewareReviewPostApplyRecord(
		record,
		ownerID,
		models.CWReviewItemStatusDismissed,
	); err != nil {
		return err
	}

	if err := verifySelfCoursewareReviewPostApplyPageTx(
		ctx,
		tx,
		record,
		ownerID,
		models.CWReviewItemStatusDismissed,
	); err != nil {
		if errors.Is(err, ErrCoursewareReviewItemAppliedPageChanged) ||
			errors.Is(err, ErrCoursewareReviewItemAppliedPageMissing) {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return fmt.Errorf(
					"提交作者自审恢复前页面状态失败: %w",
					commitErr,
				)
			}
		}
		return err
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = 'applied',
			updated_at = NOW()
		WHERE id = $1
		  AND owner_id = $2
		  AND source_type = 'self'
		  AND status = 'dismissed'
		  AND applied_instruction_version_id IS NOT NULL
		  AND applied_at = $3
		  AND BTRIM(COALESCE(applied_page_hash, '')) = $4
		  AND courseware_review_id IS NULL
		  AND feedback_id IS NULL`,
		record.ID,
		strings.TrimSpace(ownerID),
		record.AppliedAt,
		strings.TrimSpace(record.AppliedPageHash),
	)
	if err != nil {
		return fmt.Errorf("恢复作者自审修改完成状态失败: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	if err := appendCoursewareReviewItemStateEventTx(
		ctx,
		tx,
		record.SessionID,
		record.ID,
		ownerID,
		"已恢复这条自审问题，继续等待检查或调整。",
		cwReviewItemStateEventMeta{
			Event:          "restored",
			PreviousStatus: models.CWReviewItemStatusDismissed,
			NextStatus:     models.CWReviewItemStatusApplied,
		},
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交恢复作者自审修改完成状态事务失败: %w", err)
	}

	return nil
}
