package repository

// courseware_ai_review_finalize_repo.go
//
// 课件 AI 审核真实执行所需的补充仓储：
//   - 保存某批实际执行时使用的前序账本和提示词哈希；
//   - 按ID读取执行后的批次；
//   - 全部批次完成后写入最终综合报告并关闭会话。
//
// 最终完成操作使用数据库条件确认所有批次均为 done，
// 防止在存在 pending、running 或 failed 批次时提前生成“完整报告”。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// UpdateCoursewareAIReviewBatchInput 保存本批真实输入清单。
//
// continuityBeforeJSON 是本次模型调用真正继承的账本；
// inputHash 用于审计本次提示词和页面输入是否发生变化；
// inputManifestJSON 不保存API密钥，只保存输入结构、哈希和页码。
func UpdateCoursewareAIReviewBatchInput(
	ctx context.Context,
	batchID string,
	continuityBeforeJSON string,
	inputHash string,
	inputManifestJSON string,
) error {
	result, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_ai_review_batches
		SET
			continuity_before_json = $2,
			input_hash = $3,
			input_manifest_json = $4,
			updated_at = NOW()
		WHERE id = $1
			AND status = 'running'`,
		strings.TrimSpace(batchID),
		cwAIReviewJSONOrDefault(
			continuityBeforeJSON,
			"{}",
		),
		strings.TrimSpace(inputHash),
		cwAIReviewJSONOrDefault(
			inputManifestJSON,
			"{}",
		),
	)
	if err != nil {
		return fmt.Errorf(
			"保存课件AI审核批次输入失败: %w",
			err,
		)
	}
	if result.RowsAffected() == 0 {
		return errors.New(
			"课件AI审核批次不是执行中状态",
		)
	}

	return nil
}

// GetCoursewareAIReviewBatchByID 按批次ID查询。
func GetCoursewareAIReviewBatchByID(
	ctx context.Context,
	batchID string,
) (*models.CoursewareAIReviewBatch, error) {
	batch, err := scanCoursewareAIReviewBatch(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwAIReviewBatchSelectColumns+`
			 FROM courseware_ai_review_batches
			 WHERE id = $1`,
			strings.TrimSpace(batchID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf(
			"查询课件AI审核批次失败: %w",
			err,
		)
	}

	return batch, nil
}

// CompleteCoursewareAIReviewSession 写入最终报告并结束会话。
//
// SQL中的 NOT EXISTS 是最终一致性闸门：
// 只要还有任何批次不是 done，就不会把会话标记为完成。
func CompleteCoursewareAIReviewSession(
	ctx context.Context,
	sessionID string,
	finalReportJSON string,
	modelUsed string,
	tokensUsed int,
) error {
	result, err := database.DB.Exec(
		ctx,
		`
		UPDATE courseware_ai_review_sessions session
		SET
			status = 'done',
			current_stage = 'done',
			current_batch_no = total_batches,
			final_report_json = $2,
			model_used = $3,
			tokens_used = tokens_used + $4,
			error_message = '',
			completed_at = NOW(),
			updated_at = NOW()
		WHERE session.id = $1
			AND session.status = 'aggregating'
			AND NOT EXISTS (
				SELECT 1
				FROM courseware_ai_review_batches batch
				WHERE batch.session_id = session.id
					AND batch.status <> 'done'
			)`,
		strings.TrimSpace(sessionID),
		cwAIReviewJSONOrDefault(
			finalReportJSON,
			"{}",
		),
		strings.TrimSpace(modelUsed),
		tokensUsed,
	)
	if err != nil {
		return fmt.Errorf(
			"完成课件AI审核会话失败: %w",
			err,
		)
	}
	if result.RowsAffected() == 0 {
		return errors.New(
			"课件AI审核尚有未完成批次，或会话不在综合阶段",
		)
	}

	return nil
}
