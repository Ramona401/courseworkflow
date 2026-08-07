package repository

// courseware_review_resubmission_repo.go
//
// 作者重新提交课件与旧问题进入复审的原子仓储。
//
// 重新提交时在同一事务中完成：
//
//   1. 锁定课件主记录；
//   2. 再次确认操作者是课件作者；
//   3. 再次确认当前发布状态允许提交；
//   4. 锁定全部已正式交付且尚未解决的旧问题；
//   5. 确认每条旧问题都已经达到applied；
//   6. 页级问题重新读取当前页面并校验修改完成指纹；
//   7. 页面后来变化的问题改为stale；
//   8. 原页面删除的问题改为orphaned；
//   9. 存在任何未完成、变化或删除的问题时阻止重新提交；
//  10. 全部问题符合条件后，登记下一次同级复审轮次；
//  11. 将课件更新为L1待审核状态；
//  12. 提交事务。
//
// 问题不会复制，也不会覆盖原审核轮次。
//
// 提交门槛：
//
//   - confirmed、applying、detected、discussing均表示整改尚未完成；
//   - stale表示页面在整改完成后又发生变化，需要作者重新检查；
//   - orphaned表示原页面已经删除，不能直接进入复审；
//   - applied必须具有修改时间和页面内容指纹；
//   - 页级applied问题的当前HTML必须仍与applied_page_hash一致；
//   - 整课applied问题没有唯一页面，只依赖完整修改完成证据；
//   - resolved不再参与下一轮复审。

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
	ErrCWReviewSubmissionCoursewareNotFound = errors.New(
		"课件不存在",
	)

	ErrCWReviewSubmissionOwnerMismatch = errors.New(
		"只有课件作者本人可以提交审核",
	)

	ErrCWReviewSubmissionStateConflict = errors.New(
		"当前课件状态不可提交审核",
	)

	ErrCWReviewSubmissionRemediationIncomplete = errors.New(
		"正式审核整改尚未全部完成",
	)
)

const (
	cwReviewSubmissionAssessmentReady = "ready"

	cwReviewSubmissionAssessmentUnfinished = "unfinished"

	cwReviewSubmissionAssessmentStale = "stale"

	cwReviewSubmissionAssessmentOrphaned = "orphaned"
)

// CoursewareReviewSubmissionCommitResult 是重新提交事务结果。
type CoursewareReviewSubmissionCommitResult struct {
	CarryoverItemCount int64
}

// CWReviewSubmissionRemediationError 描述阻止重新提交的问题数量。
type CWReviewSubmissionRemediationError struct {
	UnfinishedCount  int
	ChangedPageCount int
	MissingPageCount int
}

// Error 返回可以直接向作者展示的自然语言说明。
func (e *CWReviewSubmissionRemediationError) Error() string {
	if e == nil {
		return ErrCWReviewSubmissionRemediationIncomplete.Error()
	}

	details :=
		make(
			[]string,
			0,
			3,
		)

	if e.UnfinishedCount > 0 {
		details =
			append(
				details,
				fmt.Sprintf(
					"%d条尚未完成页面修改",
					e.UnfinishedCount,
				),
			)
	}

	if e.ChangedPageCount > 0 {
		details =
			append(
				details,
				fmt.Sprintf(
					"%d条页面内容已变化，需要重新检查",
					e.ChangedPageCount,
				),
			)
	}

	if e.MissingPageCount > 0 {
		details =
			append(
				details,
				fmt.Sprintf(
					"%d条原问题页面已经删除",
					e.MissingPageCount,
				),
			)
	}

	if len(details) == 0 {
		return "还有正式审核整改尚未完成，请先处理后再重新提交审核"
	}

	return "暂时不能重新提交审核：" +
		strings.Join(
			details,
			"；",
		) +
		"。请回到课件整改区处理并检查全部问题"
}

// Unwrap 支持errors.Is识别统一业务错误。
func (e *CWReviewSubmissionRemediationError) Unwrap() error {
	return ErrCWReviewSubmissionRemediationIncomplete
}

type cwReviewSubmissionItemFact struct {
	ID        string
	SessionID string
	Status    string
	PageID    string

	AppliedAt       *time.Time
	AppliedPageHash string
}

type cwReviewSubmissionReadiness struct {
	ReadyCount       int
	UnfinishedCount  int
	ChangedPageCount int
	MissingPageCount int
	InvalidatedCount int
}

func (result *cwReviewSubmissionReadiness) blocked() bool {
	if result == nil {
		return false
	}

	return result.UnfinishedCount > 0 ||
		result.ChangedPageCount > 0 ||
		result.MissingPageCount > 0
}

func (result *cwReviewSubmissionReadiness) businessError() error {
	if !result.blocked() {
		return nil
	}

	return &CWReviewSubmissionRemediationError{
		UnfinishedCount:  result.UnfinishedCount,
		ChangedPageCount: result.ChangedPageCount,
		MissingPageCount: result.MissingPageCount,
	}
}

// classifyCWReviewSubmissionItem 判断一条旧问题是否达到重新提交条件。
//
// currentPageExists和currentHTML只在页级applied问题中使用。
func classifyCWReviewSubmissionItem(
	item *cwReviewSubmissionItemFact,
	currentPageExists bool,
	currentHTML string,
) string {
	if item == nil {
		return cwReviewSubmissionAssessmentUnfinished
	}

	switch item.Status {
	case models.CWReviewItemStatusStale:
		return cwReviewSubmissionAssessmentStale

	case models.CWReviewItemStatusOrphaned:
		return cwReviewSubmissionAssessmentOrphaned

	case models.CWReviewItemStatusApplied:
		// 继续执行修改证据和当前页面检查。

	default:
		return cwReviewSubmissionAssessmentUnfinished
	}

	if item.AppliedAt == nil ||
		strings.TrimSpace(
			item.AppliedPageHash,
		) == "" {
		return cwReviewSubmissionAssessmentUnfinished
	}

	// 整课问题没有唯一页面。只接受完整的修改完成证据，
	// 后续仍由审核员进行整课复审确认。
	if strings.TrimSpace(
		item.PageID,
	) == "" {
		return cwReviewSubmissionAssessmentReady
	}

	if !currentPageExists {
		return cwReviewSubmissionAssessmentOrphaned
	}

	if cwReviewItemContentHash(
		currentHTML,
	) != strings.TrimSpace(
		item.AppliedPageHash,
	) {
		return cwReviewSubmissionAssessmentStale
	}

	return cwReviewSubmissionAssessmentReady
}

// CommitCoursewareReviewSubmission 原子提交课件审核并登记旧问题复审轮次。
func CommitCoursewareReviewSubmission(
	ctx context.Context,
	coursewareID string,
	ownerID string,
	reviewSchoolID string,
) (
	*CoursewareReviewSubmissionCommitResult,
	error,
) {
	coursewareID =
		strings.TrimSpace(
			coursewareID,
		)

	ownerID =
		strings.TrimSpace(
			ownerID,
		)

	reviewSchoolID =
		strings.TrimSpace(
			reviewSchoolID,
		)

	tx, err :=
		database.DB.Begin(
			ctx,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"开始课件提交审核事务失败: %w",
				err,
			)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		currentOwnerID      string
		currentPublishState string
	)

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			user_id,
			COALESCE(publish_state, 'private')
		FROM coursewares
		WHERE id = $1
		  AND deleted_at IS NULL
		FOR UPDATE`,
		coursewareID,
	).Scan(
		&currentOwnerID,
		&currentPublishState,
	)
	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return nil,
				ErrCWReviewSubmissionCoursewareNotFound
		}

		return nil,
			fmt.Errorf(
				"锁定待提交课件失败: %w",
				err,
			)
	}

	if currentOwnerID != ownerID {
		return nil,
			ErrCWReviewSubmissionOwnerMismatch
	}

	switch currentPublishState {
	case models.CWPublishPrivate,
		models.CWPublishPublishedPersonal,
		models.CWPublishRevision:
	default:
		return nil,
			ErrCWReviewSubmissionStateConflict
	}

	readiness, err :=
		validateCWReviewSubmissionRemediationTx(
			ctx,
			tx,
			coursewareID,
			ownerID,
		)
	if err != nil {
		return nil, err
	}

	if readiness.blocked() {
		businessErr :=
			readiness.businessError()

		// 页面变化或删除属于本次检查发现的新事实。
		// 即使课件提交被阻止，也要提交这些状态更新，
		// 让作者刷新后看到正确的重新检查或保留记录入口。
		if readiness.InvalidatedCount > 0 {
			if err := tx.Commit(
				ctx,
			); err != nil {
				return nil,
					fmt.Errorf(
						"提交整改问题新鲜度状态失败: %w",
						err,
					)
			}
		}

		return nil, businessErr
	}

	// 每条正式问题回到它原本所属的审核级别复查。
	//
	// 预计轮次按该课件、该审核级别已经存在的审核记录数加一计算。
	// 全部旧问题已经在上方锁定并校验为applied。
	carryoverTag, err :=
		tx.Exec(
			ctx,
			`
			UPDATE courseware_review_items AS item
			SET
				resubmitted_at = NOW(),
				resubmitted_review_level =
					item.review_level,
				resubmitted_review_round = (
					SELECT COUNT(*) + 1
					FROM courseware_reviews AS review
					WHERE review.courseware_id =
					      item.courseware_id
					  AND review.review_level =
					      item.review_level
				),
				updated_at = NOW()
			WHERE item.courseware_id = $1
			  AND item.owner_id = $2
			  AND item.source_type = 'formal'
			  AND item.feedback_id IS NOT NULL
			  AND item.courseware_review_id IS NOT NULL
			  AND item.review_level IN (1, 2)
			  AND item.status = 'applied'
			  AND item.applied_at IS NOT NULL
			  AND BTRIM(
			      COALESCE(
			          item.applied_page_hash,
			          ''
			      )
			  ) <> ''`,
			coursewareID,
			ownerID,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"登记课件整改问题复审轮次失败: %w",
				err,
			)
	}

	commandTag, err :=
		tx.Exec(
			ctx,
			`
			UPDATE coursewares
			SET
				publish_state = 'submitted',
				review_level = 0,
				review_school_id = $3,
				updated_at = NOW()
			WHERE id = $1
			  AND user_id = $2
			  AND publish_state = $4
			  AND deleted_at IS NULL`,
			coursewareID,
			ownerID,
			reviewSchoolID,
			currentPublishState,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"更新课件提交审核状态失败: %w",
				err,
			)
	}

	if commandTag.RowsAffected() != 1 {
		return nil,
			ErrCWReviewSubmissionStateConflict
	}

	if err := tx.Commit(
		ctx,
	); err != nil {
		return nil,
			fmt.Errorf(
				"提交课件审核事务失败: %w",
				err,
			)
	}

	return &CoursewareReviewSubmissionCommitResult{
		CarryoverItemCount: carryoverTag.RowsAffected(),
	}, nil
}

// validateCWReviewSubmissionRemediationTx 锁定并检查全部正式旧问题。
func validateCWReviewSubmissionRemediationTx(
	ctx context.Context,
	tx pgx.Tx,
	coursewareID string,
	ownerID string,
) (
	*cwReviewSubmissionReadiness,
	error,
) {
	rows, err :=
		tx.Query(
			ctx,
			`
			SELECT
				id::text,
				source_session_id,
				status,
				COALESCE(page_id::text, ''),
				applied_at,
				COALESCE(applied_page_hash, '')
			FROM courseware_review_items
			WHERE courseware_id = $1
			  AND owner_id = $2
			  AND source_type = 'formal'
			  AND feedback_id IS NOT NULL
			  AND courseware_review_id IS NOT NULL
			  AND status <> 'resolved'
			ORDER BY id
			FOR UPDATE`,
			strings.TrimSpace(
				coursewareID,
			),
			strings.TrimSpace(
				ownerID,
			),
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"锁定课件正式整改问题失败: %w",
				err,
			)
	}

	items :=
		make(
			[]*cwReviewSubmissionItemFact,
			0,
		)

	for rows.Next() {
		item :=
			&cwReviewSubmissionItemFact{}

		if err :=
			rows.Scan(
				&item.ID,
				&item.SessionID,
				&item.Status,
				&item.PageID,
				&item.AppliedAt,
				&item.AppliedPageHash,
			); err != nil {
			rows.Close()

			return nil,
				fmt.Errorf(
					"扫描课件正式整改问题失败: %w",
					err,
				)
		}

		items =
			append(
				items,
				item,
			)
	}

	if err := rows.Err(); err != nil {
		rows.Close()

		return nil,
			fmt.Errorf(
				"遍历课件正式整改问题失败: %w",
				err,
			)
	}

	rows.Close()

	result :=
		&cwReviewSubmissionReadiness{}

	for _, item := range items {
		currentPageExists :=
			false

		currentHTML :=
			""

		needsCurrentPage :=
			item.Status ==
				models.CWReviewItemStatusApplied &&
				item.AppliedAt != nil &&
				strings.TrimSpace(
					item.AppliedPageHash,
				) != "" &&
				strings.TrimSpace(
					item.PageID,
				) != ""

		if needsCurrentPage {
			err = tx.QueryRow(
				ctx,
				`
				SELECT
					COALESCE(html_content, '')
				FROM courseware_pages
				WHERE id = $1
				  AND courseware_id = $2
				FOR SHARE`,
				item.PageID,
				coursewareID,
			).Scan(
				&currentHTML,
			)

			switch {
			case err == nil:
				currentPageExists = true

			case errors.Is(
				err,
				pgx.ErrNoRows,
			):
				currentPageExists = false

			default:
				return nil,
					fmt.Errorf(
						"读取正式整改问题当前页面失败: %w",
						err,
					)
			}
		}

		assessment :=
			classifyCWReviewSubmissionItem(
				item,
				currentPageExists,
				currentHTML,
			)

		switch assessment {
		case cwReviewSubmissionAssessmentReady:
			result.ReadyCount++

		case cwReviewSubmissionAssessmentUnfinished:
			result.UnfinishedCount++

		case cwReviewSubmissionAssessmentStale:
			result.ChangedPageCount++

			if item.Status ==
				models.CWReviewItemStatusApplied {
				if err :=
					invalidateCWReviewSubmissionItemTx(
						ctx,
						tx,
						item,
						ownerID,
						models.CWReviewItemStatusStale,
						"重新提交审核前检查发现：页面在完成本条整改后又发生变化，需要作者重新检查。",
					); err != nil {
					return nil, err
				}

				result.InvalidatedCount++
			}

		case cwReviewSubmissionAssessmentOrphaned:
			result.MissingPageCount++

			if item.Status ==
				models.CWReviewItemStatusApplied {
				if err :=
					invalidateCWReviewSubmissionItemTx(
						ctx,
						tx,
						item,
						ownerID,
						models.CWReviewItemStatusOrphaned,
						"重新提交审核前检查发现：原问题页面已经删除，不能直接进入复审。",
					); err != nil {
					return nil, err
				}

				result.InvalidatedCount++
			}
		}
	}

	return result, nil
}

// invalidateCWReviewSubmissionItemTx 保存重新提交检查发现的页面变化。
//
// applied_at和applied_page_hash继续保留，用于回看最后一次修改完成事实。
func invalidateCWReviewSubmissionItemTx(
	ctx context.Context,
	tx pgx.Tx,
	item *cwReviewSubmissionItemFact,
	ownerID string,
	nextStatus string,
	message string,
) error {
	if item == nil {
		return ErrCoursewareReviewItemConflict
	}

	updateTag, err :=
		tx.Exec(
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
			  AND source_type = 'formal'
			  AND status = 'applied'`,
			item.ID,
			strings.TrimSpace(
				ownerID,
			),
			nextStatus,
		)
	if err != nil {
		return fmt.Errorf(
			"更新重新提交整改问题状态失败: %w",
			err,
		)
	}

	if updateTag.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	return appendCoursewareReviewItemStateEventTx(
		ctx,
		tx,
		item.SessionID,
		item.ID,
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

// ListCoursewareReviewItemsForPendingRound 查询当前级别、当前轮次需要复审的问题。
//
// 只返回已经随历史正式反馈交付、由作者重新提交且尚未确认解决的问题。
func ListCoursewareReviewItemsForPendingRound(
	ctx context.Context,
	coursewareID string,
	reviewLevel int,
	reviewRound int,
) (
	[]*models.CoursewareReviewItem,
	error,
) {
	if reviewLevel <= 0 ||
		reviewRound <= 0 {
		return []*models.CoursewareReviewItem{},
			nil
	}

	return listCoursewareReviewItems(
		ctx,
		`WHERE courseware_id = $1
		   AND source_type = 'formal'
		   AND feedback_id IS NOT NULL
		   AND courseware_review_id IS NOT NULL
		   AND resubmitted_review_level = $2
		   AND resubmitted_review_round = $3
		   AND status <> 'resolved'`,
		strings.TrimSpace(
			coursewareID,
		),
		reviewLevel,
		reviewRound,
	)
}
