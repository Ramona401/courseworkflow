package repository

// courseware_review_history_repo.go
//
// R-03“已审核记录只读详情”的独立历史事实仓储。
//
// 本文件故意与大型courseware_review_repo.go分离：
//   - 按courseware_review_id读取一次真实人工审核；
//   - 精确读取本次正式交付的指令版本；
//   - 不提供任何UPDATE/DELETE；
//   - 不从当前课件状态、整改项当前状态或current_instruction_version_id
//     推断旧审核事实。
//
// 权限、教育域、学校/教研组/区域范围和review↔courseware关系复核
// 统一由上层CoursewareReviewService完成。

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

var (
	ErrCoursewareReviewHistoryNotFound = errors.New(
		"课件审核历史记录不存在",
	)

	ErrCoursewareReviewHistoryInstructionVersionNotFound = errors.New(
		"课件审核历史交付指令版本不存在",
	)
)

// GetCoursewareReviewHistoryRecordByID 按真实审核记录ID读取人工审核事实。
func GetCoursewareReviewHistoryRecordByID(
	ctx context.Context,
	reviewID string,
) (*models.CoursewareReview, error) {
	reviewID = strings.TrimSpace(reviewID)

	if reviewID == "" {
		return nil, ErrCoursewareReviewHistoryNotFound
	}

	review := &models.CoursewareReview{}

	err := database.DB.QueryRow(
		ctx,
		`
		SELECT
			id,
			courseware_id,
			review_level,
			reviewer_id,
			decision,
			score,
			COALESCE(comment, ''),
			COALESCE(dimensions, '{}'::jsonb)::text,
			review_round,
			created_at
		FROM courseware_reviews
		WHERE id = $1`,
		reviewID,
	).Scan(
		&review.ID,
		&review.CoursewareID,
		&review.ReviewLevel,
		&review.ReviewerID,
		&review.Decision,
		&review.Score,
		&review.Comment,
		&review.Dimensions,
		&review.ReviewRound,
		&review.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewHistoryNotFound
		}

		return nil, fmt.Errorf(
			"查询课件审核历史记录失败: %w",
			err,
		)
	}

	return review, nil
}

// GetCoursewareReviewDeliveredInstructionVersion 精确读取本次审核实际交付版本。
//
// 调用者必须同时提供：
//   - courseware_review_id；
//   - 整改项ID；
//   - 整改项冻结的delivered_instruction_version_id。
//
// SQL再次校验三者真实绑定关系，绝不读取或回退current_instruction_version_id。
func GetCoursewareReviewDeliveredInstructionVersion(
	ctx context.Context,
	reviewID string,
	itemID string,
	deliveredVersionID string,
) (*models.CoursewareReviewInstructionVersion, error) {
	reviewID = strings.TrimSpace(reviewID)
	itemID = strings.TrimSpace(itemID)
	deliveredVersionID = strings.TrimSpace(deliveredVersionID)

	if reviewID == "" || itemID == "" || deliveredVersionID == "" {
		return nil,
			ErrCoursewareReviewHistoryInstructionVersionNotFound
	}

	version := &models.CoursewareReviewInstructionVersion{}

	var (
		createdAt   time.Time
		confirmedBy string
		confirmedAt *time.Time
	)

	err := database.DB.QueryRow(
		ctx,
		`
		SELECT
			version.id,
			version.item_id,
			version.version_no,
			version.content,
			version.content_hash,
			version.source_type,
			version.created_by,
			version.created_at,
			COALESCE(version.confirmed_by::text, ''),
			version.confirmed_at,
			COALESCE(version.page_snapshot_hash, ''),
			version.status
		FROM courseware_review_instruction_versions AS version
		JOIN courseware_review_items AS item
		  ON item.id = version.item_id
		WHERE item.courseware_review_id = $1
		  AND item.id = $2
		  AND item.delivered_instruction_version_id = version.id
		  AND version.id = $3`,
		reviewID,
		itemID,
		deliveredVersionID,
	).Scan(
		&version.ID,
		&version.ItemID,
		&version.VersionNo,
		&version.Content,
		&version.ContentHash,
		&version.SourceType,
		&version.CreatedBy,
		&createdAt,
		&confirmedBy,
		&confirmedAt,
		&version.PageSnapshotHash,
		&version.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewHistoryInstructionVersionNotFound
		}

		return nil, fmt.Errorf(
			"查询课件审核历史交付指令版本失败: %w",
			err,
		)
	}

	version.CreatedAt = &createdAt
	version.ConfirmedAt = confirmedAt

	confirmedBy = strings.TrimSpace(confirmedBy)
	if confirmedBy != "" {
		version.ConfirmedBy = &confirmedBy
	}

	return version, nil
}
