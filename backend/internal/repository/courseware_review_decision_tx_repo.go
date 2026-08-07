package repository

// courseware_review_decision_tx_repo.go
//
// 课件正式审核决定的原子提交仓储。
//
// 同一事务内依次完成：
//
//   1. 锁定课件主记录；
//   2. 再次确认课件仍处于预期待审状态；
//   3. 校验AI审核会话属于当前课件、审核员和审核级别；
//   4. 计算本次审核轮次；
//   5. 锁定本级、本轮需要复查的旧问题；
//   6. 校验审核员提交的“已解决旧问题”选择；
//   7. 重新检查问题是否已经完成修改及页面内容指纹；
//   8. 写courseware_reviews审核记录；
//   9. 将审核员确认解决的旧问题写为resolved；
//  10. 写不可变的courseware_review_feedback快照；
//  11. 绑定本轮新确认并交付作者的问题；
//  12. 更新课件发布状态和审核层级；
//  13. 提交事务。
//
// 复审规则：
//
//   - approved要求本级、本轮所有旧问题全部确认解决；
//   - revision允许只确认一部分旧问题已经解决；
//   - 只有状态为applied且修改证据完整的问题可以确认解决；
//   - 页级问题的当前HTML必须仍与applied_page_hash一致；
//   - confirmed、applying、stale、orphaned等状态不能确认解决；
//   - 提交的问题ID必须全部属于本级、本轮；
//   - 任意一步失败都会回滚。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrCWReviewDecisionStateConflict = errors.New(
		"课件审核状态已变化，请刷新后重试",
	)

	ErrCWReviewDecisionSessionInvalid = errors.New(
		"课件AI审核会话无效或不属于当前审核",
	)

	ErrCWReviewDecisionItemsInvalid = errors.New(
		"选中的课件整改项无效或已被其他审核使用",
	)

	ErrCWReviewDecisionCarryoverInvalid = errors.New(
		"复审问题选择无效、尚未完成修改、页面已变化或存在遗漏",
	)
)

// CoursewareReviewDecisionCommitInput 是原子审核提交输入。
type CoursewareReviewDecisionCommitInput struct {
	CoursewareID string
	ReviewerID   string
	ReviewLevel  int

	Decision   string
	Score      *float64
	Comment    string
	Dimensions string

	ExpectedPublishState string
	ExpectedReviewLevel  int

	NextPublishState   string
	NextReviewLevel    int
	NextReviewSchoolID *string

	AIReviewSessionID *string
	ReviewItemIDs     []string

	// ResolvedReviewItemIDs是审核员本轮明确确认解决的旧问题。
	ResolvedReviewItemIDs []string

	OverallRisk         string
	OverallSummary      string
	StrengthsJSON       string
	ObviousProblemsJSON string
}

// CommitCoursewareReviewDecision 原子提交正式审核决定。
func CommitCoursewareReviewDecision(
	ctx context.Context,
	input *CoursewareReviewDecisionCommitInput,
) (
	*models.CoursewareReview,
	*models.CoursewareReviewFeedback,
	error,
) {
	if input == nil {
		return nil, nil,
			errors.New(
				"课件审核事务输入不能为空",
			)
	}

	tx, err :=
		database.DB.Begin(
			ctx,
		)
	if err != nil {
		return nil, nil,
			fmt.Errorf(
				"开始课件审核事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		currentPublishState string
		currentReviewLevel  int
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			COALESCE(publish_state, 'private'),
			COALESCE(review_level, 0)
		FROM coursewares
		WHERE id = $1
		  AND deleted_at IS NULL
		FOR UPDATE`,
		strings.TrimSpace(
			input.CoursewareID,
		),
	).Scan(
		&currentPublishState,
		&currentReviewLevel,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil, nil,
				ErrCWReviewDecisionStateConflict
		}

		return nil, nil,
			fmt.Errorf(
				"锁定待审核课件失败: %w",
				err,
			)
	}

	if currentPublishState !=
		input.ExpectedPublishState ||
		currentReviewLevel !=
			input.ExpectedReviewLevel {
		return nil, nil,
			ErrCWReviewDecisionStateConflict
	}

	if input.AIReviewSessionID != nil {
		sessionID :=
			strings.TrimSpace(
				*input.AIReviewSessionID,
			)
		if sessionID == "" {
			return nil, nil,
				ErrCWReviewDecisionSessionInvalid
		}

		var sessionValid bool

		err = tx.QueryRow(
			ctx,
			`
			SELECT EXISTS (
				SELECT 1
				FROM courseware_ai_review_sessions
				WHERE id = $1
				  AND courseware_id = $2
				  AND reviewer_id = $3
				  AND review_level = $4
				  AND status = 'done'
			)`,
			sessionID,
			strings.TrimSpace(
				input.CoursewareID,
			),
			strings.TrimSpace(
				input.ReviewerID,
			),
			input.ReviewLevel,
		).Scan(
			&sessionValid,
		)
		if err != nil {
			return nil, nil,
				fmt.Errorf(
					"校验课件AI审核会话失败: %w",
					err,
				)
		}

		if !sessionValid {
			return nil, nil,
				ErrCWReviewDecisionSessionInvalid
		}
	} else if len(
		input.ReviewItemIDs,
	) > 0 {
		return nil, nil,
			ErrCWReviewDecisionItemsInvalid
	}

	review :=
		&models.CoursewareReview{
			CoursewareID: strings.TrimSpace(
				input.CoursewareID,
			),
			ReviewLevel: input.ReviewLevel,
			ReviewerID: strings.TrimSpace(
				input.ReviewerID,
			),
			Decision: strings.TrimSpace(
				input.Decision,
			),
			Score: input.Score,
			Comment: strings.TrimSpace(
				input.Comment,
			),
			Dimensions: strings.TrimSpace(
				input.Dimensions,
			),
		}

	if review.Dimensions == "" {
		review.Dimensions = "{}"
	}

	err = tx.QueryRow(
		ctx,
		`
		SELECT COUNT(*) + 1
		FROM courseware_reviews
		WHERE courseware_id = $1
		  AND review_level = $2`,
		review.CoursewareID,
		review.ReviewLevel,
	).Scan(
		&review.ReviewRound,
	)
	if err != nil {
		return nil, nil,
			fmt.Errorf(
				"计算课件审核轮次失败: %w",
				err,
			)
	}

	carryoverRecords, err :=
		lockCWReviewCarryoverItems(
			ctx,
			tx,
			review.CoursewareID,
			review.ReviewLevel,
			review.ReviewRound,
		)
	if err != nil {
		return nil, nil, err
	}

	resolvedReviewItemIDs :=
		normalizeCWReviewDecisionItemIDs(
			input.ResolvedReviewItemIDs,
		)

	if err :=
		validateCWReviewCarryoverResolutionRecords(
			review.Decision,
			carryoverRecords,
			resolvedReviewItemIDs,
		); err != nil {
		return nil, nil, err
	}

	err = tx.QueryRow(
		ctx,
		`
		INSERT INTO courseware_reviews (
			courseware_id,
			review_level,
			reviewer_id,
			decision,
			score,
			comment,
			dimensions,
			review_round
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7::jsonb,
			$8
		)
		RETURNING
			id,
			created_at`,
		review.CoursewareID,
		review.ReviewLevel,
		review.ReviewerID,
		review.Decision,
		review.Score,
		review.Comment,
		review.Dimensions,
		review.ReviewRound,
	).Scan(
		&review.ID,
		&review.CreatedAt,
	)
	if err != nil {
		return nil, nil,
			fmt.Errorf(
				"创建课件审核记录失败: %w",
				err,
			)
	}

	if len(
		resolvedReviewItemIDs,
	) > 0 {
		affected, resolveErr :=
			resolveCWReviewCarryoverItems(
				ctx,
				tx,
				resolvedReviewItemIDs,
				review,
			)
		if resolveErr != nil {
			return nil, nil,
				resolveErr
		}

		if affected !=
			int64(
				len(
					resolvedReviewItemIDs,
				),
			) {
			return nil, nil,
				ErrCWReviewDecisionCarryoverInvalid
		}
	}

	feedback :=
		&models.CoursewareReviewFeedback{
			CoursewareReviewID: review.ID,
			CoursewareID:       review.CoursewareID,
			AIReviewSessionID:  input.AIReviewSessionID,

			ReviewLevel: review.ReviewLevel,
			ReviewRound: review.ReviewRound,
			Decision:    review.Decision,

			OverallRisk: strings.TrimSpace(
				input.OverallRisk,
			),
			OverallSummary: strings.TrimSpace(
				input.OverallSummary,
			),
			StrengthsJSON: cwAIReviewJSONOrDefault(
				input.StrengthsJSON,
				"[]",
			),
			ObviousProblemsJSON: cwAIReviewJSONOrDefault(
				input.ObviousProblemsJSON,
				"[]",
			),
			ReviewCommentSnapshot: review.Comment,

			CreatedBy: review.ReviewerID,
		}

	if feedback.OverallRisk == "" {
		feedback.OverallRisk =
			models.CWReviewSeverityInfo
	}

	if err :=
		CreateCoursewareReviewFeedbackTx(
			ctx,
			tx,
			feedback,
		); err != nil {
		return nil, nil, err
	}

	if len(
		input.ReviewItemIDs,
	) > 0 {
		affected, attachErr :=
			AttachCoursewareReviewItemsToFeedbackTx(
				ctx,
				tx,
				input.ReviewItemIDs,
				review.CoursewareID,
				*input.AIReviewSessionID,
				review.ReviewerID,
				review.ID,
				feedback.ID,
				review.ReviewLevel,
				review.ReviewRound,
			)
		if attachErr != nil {
			return nil, nil,
				attachErr
		}

		if affected !=
			int64(
				len(
					input.ReviewItemIDs,
				),
			) {
			return nil, nil,
				ErrCWReviewDecisionItemsInvalid
		}
	}

	var schoolArg interface{}

	if input.NextReviewSchoolID != nil &&
		strings.TrimSpace(
			*input.NextReviewSchoolID,
		) != "" {
		schoolArg =
			strings.TrimSpace(
				*input.NextReviewSchoolID,
			)
	}

	commandTag, err :=
		tx.Exec(
			ctx,
			`
			UPDATE coursewares
			SET
				publish_state = $2,
				review_level = $3,
				review_school_id = $4,
				updated_at = NOW()
			WHERE id = $1
			  AND publish_state = $5
			  AND review_level = $6
			  AND deleted_at IS NULL`,
			review.CoursewareID,
			strings.TrimSpace(
				input.NextPublishState,
			),
			input.NextReviewLevel,
			schoolArg,
			input.ExpectedPublishState,
			input.ExpectedReviewLevel,
		)
	if err != nil {
		return nil, nil,
			fmt.Errorf(
				"更新课件审核状态失败: %w",
				err,
			)
	}

	if commandTag.RowsAffected() == 0 {
		return nil, nil,
			ErrCWReviewDecisionStateConflict
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return nil, nil,
			fmt.Errorf(
				"提交课件审核事务失败: %w",
				err,
			)
	}

	return review, feedback, nil
}
