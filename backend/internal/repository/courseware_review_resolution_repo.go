package repository

// courseware_review_resolution_repo.go
//
// V1.3课件整改问题解决确认专用仓储。
//
// 本文件集中处理两类最终确认：
//
//   一、正式审核员在正式审核决定中确认旧问题已经解决；
//   二、作者检查自己的自审修改后，明确确认问题已经解决。
//
// 两类确认共同遵守：
//
//   - 只有applied可以进入resolved；
//   - 必须具有applied_at和applied_page_hash；
//   - 页级问题必须重新读取当前HTML；
//   - 当前HTML的SHA-256必须仍等于applied_page_hash；
//   - 页面变化后不能确认解决；
//   - 页面删除后不能确认解决；
//   - 页面写入成功本身不自动等于问题解决。
//
// page_id使用ON DELETE SET NULL，因此page_number_snapshot > 0而page_id为空
// 明确表示“历史页级问题的原页面已经删除”，不能当作整课问题处理。
//
// 正式问题必须绑定真实复审记录。
// 自审问题由作者本人确认，不绑定正式审核记录。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	ErrCoursewareReviewItemAppliedPageChanged = errors.New(
		"页面在修改完成后又发生变化，需要重新检查",
	)
	ErrCoursewareReviewItemAppliedPageMissing = errors.New(
		"问题对应的原页面已经删除",
	)
)

// cwReviewCarryoverRecord 是事务内锁定的正式复审问题事实。
type cwReviewCarryoverRecord struct {
	ID                 string
	Status             string
	PageID             string
	PageNumberSnapshot int
	CurrentPageID      string
	CurrentHTML        string
	AppliedAt          *time.Time
	AppliedPageHash    string
}

// cwReviewItemContentHash 与服务层cwAIReviewHash保持完全一致。
//
// 算法：SHA-256（字符串UTF-8原始字节）。
func cwReviewItemContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// lockCWReviewCarryoverItems 锁定本级、本轮尚未解决的正式旧问题。
//
// 同时读取当前页面HTML，供事务内判断问题是否仍可确认解决。
func lockCWReviewCarryoverItems(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	reviewLevel int,
	reviewRound int,
) ([]*cwReviewCarryoverRecord, error) {
	rows, err := tx.Query(
		ctx,
		`
		SELECT
			item.id::text,
			item.status,
			COALESCE(item.page_id::text, ''),
			item.page_number_snapshot,
			COALESCE(page.id::text, ''),
			COALESCE(page.html_content, ''),
			item.applied_at,
			COALESCE(item.applied_page_hash, '')
		FROM courseware_review_items AS item
		LEFT JOIN courseware_pages AS page
		       ON page.id = item.page_id
		      AND page.courseware_id = item.courseware_id
		WHERE item.courseware_id = $1
		  AND item.source_type = 'formal'
		  AND item.feedback_id IS NOT NULL
		  AND item.resubmitted_review_level = $2
		  AND item.resubmitted_review_round = $3
		  AND item.status <> 'resolved'
		ORDER BY item.id
		FOR UPDATE OF item`,
		strings.TrimSpace(coursewareID),
		reviewLevel,
		reviewRound,
	)
	if err != nil {
		return nil, fmt.Errorf("锁定本轮复审问题失败: %w", err)
	}
	defer rows.Close()

	result := make([]*cwReviewCarryoverRecord, 0)

	for rows.Next() {
		record := &cwReviewCarryoverRecord{}
		if err := rows.Scan(
			&record.ID,
			&record.Status,
			&record.PageID,
			&record.PageNumberSnapshot,
			&record.CurrentPageID,
			&record.CurrentHTML,
			&record.AppliedAt,
			&record.AppliedPageHash,
		); err != nil {
			return nil, fmt.Errorf("扫描本轮复审问题失败: %w", err)
		}
		result = append(result, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历本轮复审问题失败: %w", err)
	}

	return result, nil
}

// normalizeCWReviewDecisionItemIDs 去空、去重并保留原顺序。
func normalizeCWReviewDecisionItemIDs(input []string) []string {
	result := make([]string, 0, len(input))
	seen := make(map[string]bool)

	for _, raw := range input {
		value := strings.TrimSpace(raw)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}

	return result
}

// validateCWReviewCarryoverResolution 校验本轮旧问题ID选择。
//
// 保留该纯规则函数供既有单元测试使用。
func validateCWReviewCarryoverResolution(
	decision string,
	carryoverIDs []string,
	resolvedIDs []string,
) error {
	carryoverSet := make(map[string]bool, len(carryoverIDs))

	for _, raw := range carryoverIDs {
		value := strings.TrimSpace(raw)
		if value != "" {
			carryoverSet[value] = true
		}
	}

	normalizedResolvedIDs := normalizeCWReviewDecisionItemIDs(resolvedIDs)
	for _, itemID := range normalizedResolvedIDs {
		if !carryoverSet[itemID] {
			return ErrCWReviewDecisionCarryoverInvalid
		}
	}

	switch decision {
	case models.ReviewDecisionApproved:
		if len(normalizedResolvedIDs) != len(carryoverSet) {
			return ErrCWReviewDecisionCarryoverInvalid
		}
	case models.ReviewDecisionRevision:
		// 继续退回允许只确认一部分旧问题已经解决。
	default:
		return ErrCWReviewDecisionCarryoverInvalid
	}

	return nil
}

// validateCWReviewCarryoverResolutionRecords 校验问题是否真正具备关闭条件。
func validateCWReviewCarryoverResolutionRecords(
	decision string,
	records []*cwReviewCarryoverRecord,
	resolvedIDs []string,
) error {
	carryoverIDs := make([]string, 0, len(records))
	recordByID := make(map[string]*cwReviewCarryoverRecord, len(records))

	for _, record := range records {
		if record == nil {
			continue
		}

		itemID := strings.TrimSpace(record.ID)
		if itemID == "" {
			continue
		}

		carryoverIDs = append(carryoverIDs, itemID)
		recordByID[itemID] = record
	}

	if err := validateCWReviewCarryoverResolution(
		decision,
		carryoverIDs,
		resolvedIDs,
	); err != nil {
		return err
	}

	for _, itemID := range normalizeCWReviewDecisionItemIDs(resolvedIDs) {
		if !isCWReviewCarryoverRecordResolvable(recordByID[itemID]) {
			return ErrCWReviewDecisionCarryoverInvalid
		}
	}

	return nil
}

// isCWReviewCarryoverRecordResolvable 判断正式问题是否真的可以关闭。
func isCWReviewCarryoverRecordResolvable(
	record *cwReviewCarryoverRecord,
) bool {
	if record == nil ||
		record.Status != models.CWReviewItemStatusApplied ||
		record.AppliedAt == nil ||
		strings.TrimSpace(record.AppliedPageHash) == "" {
		return false
	}

	if strings.TrimSpace(record.PageID) == "" {
		// 只有从创建开始就是整课问题（page_number_snapshot=0）
		// 才能在没有page_id时依据整课人工检查继续确认。
		return record.PageNumberSnapshot == 0
	}

	if strings.TrimSpace(record.CurrentPageID) == "" {
		return false
	}

	return cwReviewItemContentHash(record.CurrentHTML) ==
		strings.TrimSpace(record.AppliedPageHash)
}

// resolveCWReviewCarryoverItems 将审核员明确确认的旧问题标记为解决。
//
// UPDATE中再次校验状态、修改证据和当前页面内容，阻止锁定后的并发变化。
func resolveCWReviewCarryoverItems(
	ctx context.Context,
	tx pgx.Tx,
	itemIDs []string,
	review *models.CoursewareReview,
) (int64, error) {
	if review == nil {
		return 0, ErrCWReviewDecisionCarryoverInvalid
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_review_items AS item
		SET
			status = 'resolved',
			resolved_at = NOW(),
			resolved_by = $4,
			resolved_review_id = $5,
			resolved_review_level = $2,
			resolved_review_round = $3,
			resolution_note = $6,
			updated_at = NOW()
		WHERE item.id::text = ANY($1::text[])
		  AND item.courseware_id = $7
		  AND item.source_type = 'formal'
		  AND item.feedback_id IS NOT NULL
		  AND item.resubmitted_review_level = $2
		  AND item.resubmitted_review_round = $3
		  AND item.status = 'applied'
		  AND item.applied_at IS NOT NULL
		  AND BTRIM(COALESCE(item.applied_page_hash, '')) <> ''
		  AND (
				(
					item.page_id IS NULL
					AND item.page_number_snapshot = 0
				)
				OR EXISTS (
					SELECT 1
					FROM courseware_pages AS page
					WHERE page.id = item.page_id
					  AND page.courseware_id = item.courseware_id
					  AND encode(
							digest(
								convert_to(
									COALESCE(page.html_content, ''),
									'UTF8'
								),
								'sha256'
							),
							'hex'
					  ) = BTRIM(
							COALESCE(item.applied_page_hash, '')
					  )
				)
		  )`,
		itemIDs,
		review.ReviewLevel,
		review.ReviewRound,
		review.ReviewerID,
		review.ID,
		review.Comment,
		review.CoursewareID,
	)
	if err != nil {
		return 0, fmt.Errorf("确认课件复审问题已解决失败: %w", err)
	}

	return result.RowsAffected(), nil
}

// ResolveSelfCoursewareReviewItem 原子确认作者自审问题已经解决。
//
// 页面变化或删除时，会先把问题改为stale或orphaned并提交，
// 再向上层返回对应错误，避免问题继续错误显示为“修改完成待确认”。
func ResolveSelfCoursewareReviewItem(
	ctx context.Context,
	itemID string,
	ownerID string,
	resolutionNote string,
) error {
	itemID = strings.TrimSpace(itemID)
	ownerID = strings.TrimSpace(ownerID)
	resolutionNote = strings.TrimSpace(resolutionNote)

	if itemID == "" || ownerID == "" {
		return ErrCoursewareReviewItemNotFound
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始作者自审确认事务失败: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		sessionID          string
		coursewareID       string
		sourceType         string
		currentOwnerID     string
		currentStatus      string
		pageID             string
		pageNumberSnapshot int
		appliedAt          *time.Time
		appliedPageHash    string
		resolvedBy         string
		alreadyDelivered   bool
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			source_session_id,
			courseware_id,
			source_type,
			owner_id,
			status,
			COALESCE(page_id::text, ''),
			page_number_snapshot,
			applied_at,
			COALESCE(applied_page_hash, ''),
			COALESCE(resolved_by::text, ''),
			(
				courseware_review_id IS NOT NULL
				OR feedback_id IS NOT NULL
			)
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
		&currentOwnerID,
		&currentStatus,
		&pageID,
		&pageNumberSnapshot,
		&appliedAt,
		&appliedPageHash,
		&resolvedBy,
		&alreadyDelivered,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCoursewareReviewItemNotFound
		}
		return fmt.Errorf("锁定作者自审问题失败: %w", err)
	}

	if sourceType != models.CWReviewItemSourceSelf ||
		currentOwnerID != ownerID ||
		alreadyDelivered {
		return ErrCoursewareReviewItemConflict
	}

	// 重复点击时保持幂等。
	if currentStatus == models.CWReviewItemStatusResolved &&
		resolvedBy == ownerID {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("提交作者自审重复确认事务失败: %w", err)
		}
		return nil
	}

	if currentStatus != models.CWReviewItemStatusApplied ||
		appliedAt == nil ||
		strings.TrimSpace(appliedPageHash) == "" {
		return ErrCoursewareReviewItemConflict
	}

	if strings.TrimSpace(pageID) == "" && pageNumberSnapshot > 0 {
		if invalidateErr := invalidateSelfCoursewareReviewItemTx(
			ctx,
			tx,
			sessionID,
			itemID,
			ownerID,
			models.CWReviewItemStatusOrphaned,
			"原页面已不存在，需要人工重新检查相关页面后再继续处理。",
		); invalidateErr != nil {
			return invalidateErr
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("提交作者自审页面删除状态失败: %w", err)
		}

		return ErrCoursewareReviewItemAppliedPageMissing
	}

	if pageID != "" {
		var currentHTML string

		err = tx.QueryRow(
			ctx,
			`
			SELECT COALESCE(html_content, '')
			FROM courseware_pages
			WHERE id = $1
			  AND courseware_id = $2`,
			pageID,
			coursewareID,
		).Scan(&currentHTML)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				if invalidateErr := invalidateSelfCoursewareReviewItemTx(
					ctx,
					tx,
					sessionID,
					itemID,
					ownerID,
					models.CWReviewItemStatusOrphaned,
					"原页面已不存在，需要人工重新检查相关页面后再继续处理。",
				); invalidateErr != nil {
					return invalidateErr
				}

				if err := tx.Commit(ctx); err != nil {
					return fmt.Errorf(
						"提交作者自审页面删除状态失败: %w",
						err,
					)
				}

				return ErrCoursewareReviewItemAppliedPageMissing
			}

			return fmt.Errorf("读取作者自审问题当前页面失败: %w", err)
		}

		if cwReviewItemContentHash(currentHTML) !=
			strings.TrimSpace(appliedPageHash) {
			if invalidateErr := invalidateSelfCoursewareReviewItemTx(
				ctx,
				tx,
				sessionID,
				itemID,
				ownerID,
				models.CWReviewItemStatusStale,
				"页面内容已变化，需要人工重新检查当前页面后再继续处理。",
			); invalidateErr != nil {
				return invalidateErr
			}

			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf(
					"提交作者自审页面变化状态失败: %w",
					err,
				)
			}

			return ErrCoursewareReviewItemAppliedPageChanged
		}
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = 'resolved',
			resolved_at = NOW(),
			resolved_by = $3,
			resolved_review_id = NULL,
			resolved_review_level = 0,
			resolved_review_round = 0,
			resolution_note = $4,
			updated_at = NOW()
		WHERE id = $1
		  AND owner_id = $2
		  AND source_type = 'self'
		  AND status = 'applied'
		  AND applied_at IS NOT NULL
		  AND BTRIM(COALESCE(applied_page_hash, '')) <> ''
		  AND courseware_review_id IS NULL
		  AND feedback_id IS NULL`,
		itemID,
		ownerID,
		ownerID,
		resolutionNote,
	)
	if err != nil {
		return fmt.Errorf("确认作者自审问题已经解决失败: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	if err := appendCoursewareReviewItemStateEventTx(
		ctx,
		tx,
		sessionID,
		itemID,
		ownerID,
		"作者检查当前课件后，明确确认这条自审问题已经解决。",
		cwReviewItemStateEventMeta{
			Event:          "resolved",
			PreviousStatus: models.CWReviewItemStatusApplied,
			NextStatus:     models.CWReviewItemStatusResolved,
			Reason:         resolutionNote,
		},
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交作者自审确认事务失败: %w", err)
	}

	return nil
}

// invalidateSelfCoursewareReviewItemTx 将已完成修改的问题改为失效状态。
func invalidateSelfCoursewareReviewItemTx(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	itemID string,
	ownerID string,
	nextStatus string,
	message string,
) error {
	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_review_items
		SET
			status = $3,
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
		  AND status = 'applied'`,
		itemID,
		ownerID,
		nextStatus,
	)
	if err != nil {
		return fmt.Errorf("更新作者自审问题失效状态失败: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	return appendCoursewareReviewItemStateEventTx(
		ctx,
		tx,
		sessionID,
		itemID,
		ownerID,
		message,
		cwReviewItemStateEventMeta{
			Event:          nextStatus,
			PreviousStatus: models.CWReviewItemStatusApplied,
			NextStatus:     nextStatus,
			Reason:         message,
		},
	)
}

// appendCoursewareReviewItemStateEventTx 写入可回看的状态系统记录。
func appendCoursewareReviewItemStateEventTx(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	itemID string,
	userID string,
	content string,
	meta cwReviewItemStateEventMeta,
) error {
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("序列化课件整改项状态事件失败: %w", err)
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
		strings.TrimSpace(userID),
		strings.TrimSpace(content),
		string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf("记录课件整改项状态事件失败: %w", err)
	}

	return nil
}
