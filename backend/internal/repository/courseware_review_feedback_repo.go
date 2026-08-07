package repository

// courseware_review_feedback_repo.go
//
// 正式课件审核整体反馈快照的数据访问层。
//
// 写入边界：
//   - 正式审核记录、反馈快照、整改项绑定和课件发布态更新必须由
//     上层服务放在同一个数据库事务中；
//   - 本文件提供Tx写入函数，不自行提交事务；
//   - 反馈快照写入后不提供普通更新接口，确保审核历史不可变。
//
// 读取边界：
//   - 仓储只按主键或课件ID读取；
//   - 课件可见性、教育域和审核身份必须由Service在调用前校验。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrCoursewareReviewFeedbackNotFound 表示正式审核反馈不存在。
var ErrCoursewareReviewFeedbackNotFound = errors.New(
	"课件审核反馈不存在",
)

const cwReviewFeedbackSelectColumns = `
	id,
	courseware_review_id,
	courseware_id,
	COALESCE(ai_review_session_id::text, ''),
	review_level,
	review_round,
	decision,
	overall_risk,
	overall_summary,
	COALESCE(strengths_json::text, '[]'),
	COALESCE(obvious_problems_json::text, '[]'),
	review_comment_snapshot,
	created_by,
	created_at`

func scanCoursewareReviewFeedback(row interface {
	Scan(dest ...interface{}) error
}) (*models.CoursewareReviewFeedback, error) {
	feedback := &models.CoursewareReviewFeedback{}
	var sessionID string

	err := row.Scan(
		&feedback.ID,
		&feedback.CoursewareReviewID,
		&feedback.CoursewareID,
		&sessionID,
		&feedback.ReviewLevel,
		&feedback.ReviewRound,
		&feedback.Decision,
		&feedback.OverallRisk,
		&feedback.OverallSummary,
		&feedback.StrengthsJSON,
		&feedback.ObviousProblemsJSON,
		&feedback.ReviewCommentSnapshot,
		&feedback.CreatedBy,
		&feedback.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if sessionID != "" {
		feedback.AIReviewSessionID = &sessionID
	}

	return feedback, nil
}

// CreateCoursewareReviewFeedbackTx 在调用方事务中写入不可变反馈快照。
func CreateCoursewareReviewFeedbackTx(
	ctx context.Context,
	tx pgx.Tx,
	feedback *models.CoursewareReviewFeedback,
) error {
	if tx == nil {
		return errors.New("课件审核反馈事务不能为空")
	}
	if feedback == nil {
		return errors.New("课件审核反馈不能为空")
	}

	err := tx.QueryRow(
		ctx,
		`
		INSERT INTO courseware_review_feedback (
			courseware_review_id,
			courseware_id,
			ai_review_session_id,
			review_level,
			review_round,
			decision,
			overall_risk,
			overall_summary,
			strengths_json,
			obvious_problems_json,
			review_comment_snapshot,
			created_by,
			created_at
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
			$9::jsonb,
			$10::jsonb,
			$11,
			$12,
			NOW()
		)
		RETURNING id, created_at`,
		strings.TrimSpace(feedback.CoursewareReviewID),
		strings.TrimSpace(feedback.CoursewareID),
		feedback.AIReviewSessionID,
		feedback.ReviewLevel,
		feedback.ReviewRound,
		strings.TrimSpace(feedback.Decision),
		strings.TrimSpace(feedback.OverallRisk),
		strings.TrimSpace(feedback.OverallSummary),
		cwAIReviewJSONOrDefault(
			feedback.StrengthsJSON,
			"[]",
		),
		cwAIReviewJSONOrDefault(
			feedback.ObviousProblemsJSON,
			"[]",
		),
		strings.TrimSpace(
			feedback.ReviewCommentSnapshot,
		),
		strings.TrimSpace(feedback.CreatedBy),
	).Scan(
		&feedback.ID,
		&feedback.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"创建课件审核反馈快照失败: %w",
			err,
		)
	}

	return nil
}

// GetCoursewareReviewFeedbackByID 按反馈ID读取。
func GetCoursewareReviewFeedbackByID(
	ctx context.Context,
	feedbackID string,
) (*models.CoursewareReviewFeedback, error) {
	feedback, err := scanCoursewareReviewFeedback(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwReviewFeedbackSelectColumns+`
			 FROM courseware_review_feedback
			 WHERE id = $1`,
			strings.TrimSpace(feedbackID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewFeedbackNotFound
		}
		return nil, fmt.Errorf(
			"查询课件审核反馈失败: %w",
			err,
		)
	}

	return feedback, nil
}

// GetCoursewareReviewFeedbackByReviewID 按正式审核记录读取反馈。
func GetCoursewareReviewFeedbackByReviewID(
	ctx context.Context,
	reviewID string,
) (*models.CoursewareReviewFeedback, error) {
	feedback, err := scanCoursewareReviewFeedback(
		database.DB.QueryRow(
			ctx,
			`SELECT `+cwReviewFeedbackSelectColumns+`
			 FROM courseware_review_feedback
			 WHERE courseware_review_id = $1`,
			strings.TrimSpace(reviewID),
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewFeedbackNotFound
		}
		return nil, fmt.Errorf(
			"按审核记录查询课件反馈失败: %w",
			err,
		)
	}

	return feedback, nil
}

// ListCoursewareReviewFeedbackByCourseware 返回课件全部正式反馈历史。
func ListCoursewareReviewFeedbackByCourseware(
	ctx context.Context,
	coursewareID string,
) ([]*models.CoursewareReviewFeedback, error) {
	rows, err := database.DB.Query(
		ctx,
		`SELECT `+cwReviewFeedbackSelectColumns+`
		 FROM courseware_review_feedback
		 WHERE courseware_id = $1
		 ORDER BY created_at DESC`,
		strings.TrimSpace(coursewareID),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件审核反馈历史失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := make(
		[]*models.CoursewareReviewFeedback,
		0,
	)

	for rows.Next() {
		item, scanErr :=
			scanCoursewareReviewFeedback(rows)
		if scanErr != nil {
			return nil, fmt.Errorf(
				"扫描课件审核反馈失败: %w",
				scanErr,
			)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件审核反馈失败: %w",
			err,
		)
	}

	return items, nil
}
