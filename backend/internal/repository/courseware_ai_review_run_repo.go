package repository

// courseware_ai_review_run_repo.go
//
// 课件 AI 审核顺序批次执行的仓储状态机。
//
// 状态流：
//
//   pending
//      ↓ ClaimNextCoursewareAIReviewBatch
//   running
//      ├─ CompleteCoursewareAIReviewBatch → done
//      └─ FailCoursewareAIReviewBatch     → failed
//
// 顺序保证：
//   - 只有前面所有批次都为 done，下一批才允许被领取；
//   - SELECT ... FOR UPDATE SKIP LOCKED 防止同一批被重复执行；
//   - 批次完成时，同一事务更新 Session 的当前批次、连续性账本和Token；
//   - 最后一批完成后，Session 进入 aggregating / risk_recheck，等待最终综合报告；
//   - AI执行失败不会修改人工审核决定。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ClaimNextCoursewareAIReviewBatch 原子领取下一条可执行批次。
//
// 没有可领取批次时返回 (nil, nil)，可能原因：
//   - 所有批次已经完成；
//   - 当前已有批次正在执行；
//   - 前一批失败或尚未完成。
func ClaimNextCoursewareAIReviewBatch(
	ctx context.Context,
	sessionID string,
) (*models.CoursewareAIReviewBatch, error) {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开启课件AI审核批次领取事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	row := tx.QueryRow(
		ctx,
		`SELECT `+cwAIReviewBatchSelectColumns+`
		 FROM courseware_ai_review_batches b
		 WHERE b.session_id = $1
			AND b.status = 'pending'
			AND NOT EXISTS (
				SELECT 1
				FROM courseware_ai_review_batches previous
				WHERE previous.session_id = b.session_id
					AND previous.batch_no < b.batch_no
					AND previous.status <> 'done'
			)
		 ORDER BY b.batch_no ASC
		 FOR UPDATE SKIP LOCKED
		 LIMIT 1`,
		strings.TrimSpace(sessionID),
	)

	batch, err := scanCoursewareAIReviewBatch(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return nil, fmt.Errorf(
					"提交空批次领取事务失败: %w",
					commitErr,
				)
			}
			return nil, nil
		}

		return nil, fmt.Errorf(
			"读取下一课件AI审核批次失败: %w",
			err,
		)
	}

	err = tx.QueryRow(
		ctx,
		`
		UPDATE courseware_ai_review_batches
		SET
			status = 'running',
			error_message = '',
			started_at = NOW(),
			completed_at = NULL,
			updated_at = NOW()
		WHERE id = $1
		RETURNING started_at, updated_at`,
		batch.ID,
	).Scan(
		&batch.StartedAt,
		&batch.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"标记课件AI审核批次执行中失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf(
			"提交课件AI审核批次领取事务失败: %w",
			err,
		)
	}

	batch.Status = models.CWAIReviewBatchRunning
	return batch, nil
}

// GetPreviousCompletedCoursewareAIReviewBatch 获取当前批次前一条已完成结果。
func GetPreviousCompletedCoursewareAIReviewBatch(
	ctx context.Context,
	sessionID string,
	currentBatchNo int,
) (*models.CoursewareAIReviewBatch, error) {
	if currentBatchNo <= 1 {
		return nil, nil
	}

	batch, err := scanCoursewareAIReviewBatch(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwAIReviewBatchSelectColumns+`
			 FROM courseware_ai_review_batches
			 WHERE session_id = $1
				AND batch_no < $2
				AND status = 'done'
			 ORDER BY batch_no DESC
			 LIMIT 1`,
			strings.TrimSpace(sessionID),
			currentBatchNo,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"查询前序课件AI审核批次失败: %w",
			err,
		)
	}

	return batch, nil
}

// CompleteCoursewareAIReviewBatch 完成一个批次，并推进会话状态。
func CompleteCoursewareAIReviewBatch(
	ctx context.Context,
	batchID string,
	resultJSON string,
	continuityAfterJSON string,
	riskPagesJSON string,
	modelUsed string,
	tokensUsed int,
) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启课件AI审核批次完成事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var sessionID string
	var batchNo int

	err = tx.QueryRow(
		ctx,
		`
		SELECT session_id, batch_no
		FROM courseware_ai_review_batches
		WHERE id = $1
		FOR UPDATE`,
		strings.TrimSpace(batchID),
	).Scan(
		&sessionID,
		&batchNo,
	)
	if err != nil {
		return fmt.Errorf(
			"锁定课件AI审核批次失败: %w",
			err,
		)
	}

	result, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_ai_review_batches
		SET
			status = 'done',
			result_json = $2,
			continuity_after_json = $3,
			risk_pages_json = $4,
			model_used = $5,
			tokens_used = $6,
			error_message = '',
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
			AND status = 'running'`,
		batchID,
		cwAIReviewJSONOrDefault(resultJSON, "{}"),
		cwAIReviewJSONOrDefault(
			continuityAfterJSON,
			"{}",
		),
		cwAIReviewJSONOrDefault(riskPagesJSON, "[]"),
		strings.TrimSpace(modelUsed),
		tokensUsed,
	)
	if err != nil {
		return fmt.Errorf(
			"写入课件AI审核批次结果失败: %w",
			err,
		)
	}
	if result.RowsAffected() == 0 {
		return errors.New(
			"课件AI审核批次不是可完成的执行中状态",
		)
	}

	var remaining int
	if err := tx.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM courseware_ai_review_batches
		WHERE session_id = $1
			AND status <> 'done'`,
		sessionID,
	).Scan(&remaining); err != nil {
		return fmt.Errorf(
			"统计课件AI审核剩余批次失败: %w",
			err,
		)
	}

	sessionStatus := models.CWAIReviewStatusReviewing
	sessionStage := models.CWAIReviewStageBatch

	if remaining == 0 {
		sessionStatus = models.CWAIReviewStatusAggregating
		sessionStage = models.CWAIReviewStageRiskRecheck
	}

	sessionResult, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_ai_review_sessions
		SET
			status = $2,
			current_stage = $3,
			current_batch_no = $4,
			continuity_ledger_json = $5,
			tokens_used = tokens_used + $6,
			error_message = '',
			updated_at = NOW()
		WHERE id = $1
			AND status IN ('reviewing', 'aggregating')`,
		sessionID,
		sessionStatus,
		sessionStage,
		batchNo,
		cwAIReviewJSONOrDefault(
			continuityAfterJSON,
			"{}",
		),
		tokensUsed,
	)
	if err != nil {
		return fmt.Errorf(
			"推进课件AI审核会话状态失败: %w",
			err,
		)
	}
	if sessionResult.RowsAffected() == 0 {
		return errors.New(
			"课件AI审核会话不是可推进状态",
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交课件AI审核批次结果事务失败: %w",
			err,
		)
	}

	return nil
}

// FailCoursewareAIReviewBatch 标记批次和会话失败。
//
// 失败原因保存内部可读说明；人工审核状态不受影响。
func FailCoursewareAIReviewBatch(
	ctx context.Context,
	batchID string,
	errorMessage string,
) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf(
			"开启课件AI审核失败事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var sessionID string

	err = tx.QueryRow(
		ctx,
		`
		UPDATE courseware_ai_review_batches
		SET
			status = 'failed',
			error_message = $2,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
			AND status IN ('pending', 'running')
		RETURNING session_id`,
		strings.TrimSpace(batchID),
		strings.TrimSpace(errorMessage),
	).Scan(&sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New(
				"课件AI审核批次不是可失败状态",
			)
		}
		return fmt.Errorf(
			"标记课件AI审核批次失败: %w",
			err,
		)
	}

	if _, err := tx.Exec(
		ctx,
		`
		UPDATE courseware_ai_review_sessions
		SET
			status = 'failed',
			error_message = $2,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1`,
		sessionID,
		strings.TrimSpace(errorMessage),
	); err != nil {
		return fmt.Errorf(
			"标记课件AI审核会话失败: %w",
			err,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf(
			"提交课件AI审核失败事务失败: %w",
			err,
		)
	}

	return nil
}

// HasRemainingCoursewareAIReviewBatches 判断是否仍有未完成批次。
func HasRemainingCoursewareAIReviewBatches(
	ctx context.Context,
	sessionID string,
) (bool, error) {
	var count int

	err := database.DB.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM courseware_ai_review_batches
		WHERE session_id = $1
			AND status <> 'done'`,
		strings.TrimSpace(sessionID),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf(
			"查询课件AI审核剩余批次失败: %w",
			err,
		)
	}

	return count > 0, nil
}
