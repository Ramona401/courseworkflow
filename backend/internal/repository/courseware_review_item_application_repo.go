package repository

// courseware_review_item_application_repo.go
//
// 课件审核整改项页面应用的版本绑定事务仓储。
//
// 本文件负责 confirmed -> applying 的原子开始动作：
//   1. 锁定整改项；
//   2. 复核课件作者、课件归属和整改项状态；
//   3. 正式问题必须使用正式交付版本，自审问题必须使用当前确认版本；
//   4. 复核请求version_id对应的版本归属、状态和可信正文；
//   5. 复核稳定page_id、当前页码、问题快照哈希和版本确认时页面哈希；
//   6. 页面变化或删除时提交stale或orphaned状态；
//   7. 校验通过后同时写入applying和applied_instruction_version_id；
//   8. 返回本事务确认的稳定page_id、页码和页面HTML哈希。
//
// 浏览器提交的指令正文只用于确认其完整包含可信版本正文。
// 版本归属、页面快照、版本内容和页面守卫均以数据库记录为准。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tedna/internal/database"
	"tedna/internal/models"
)

var (
	ErrCoursewareReviewItemApplicationVersionMismatch = errors.New(
		"页面应用指令版本已变化，请刷新后重试",
	)
	ErrCoursewareReviewItemApplicationInstructionMismatch = errors.New(
		"页面微调指令未完整包含指定版本的整改指令",
	)
	ErrCoursewareReviewItemApplicationPageMismatch = errors.New(
		"整改项与当前课件页面不匹配",
	)
	ErrCoursewareReviewItemApplicationPageStale = errors.New(
		"整改项对应页面已经变化，当前指令版本不能继续执行",
	)
	ErrCoursewareReviewItemApplicationPageOrphaned = errors.New(
		"整改项对应页面已经删除，当前指令版本不能继续执行",
	)
	ErrCoursewareReviewItemApplicationNotDelivered = errors.New(
		"正式整改项尚未随审核反馈交付作者",
	)
)

// BeginCoursewareReviewItemApplicationInput 是页面应用事务的可信输入。
//
// ActorID来自JWT构建的Actor。
// 浏览器只负责提交整改项ID、版本ID和实际页面微调指令。
type BeginCoursewareReviewItemApplicationInput struct {
	ItemID               string
	ActorID              string
	CoursewareID         string
	PageNumber           int
	InstructionVersionID string
	SubmittedInstruction string
}

// BeginCoursewareReviewItemApplicationResult 是事务确认后的页面执行守卫。
//
// PageHTMLHash是事务持有页面FOR SHARE锁期间计算出的正式页面哈希。
// 事务提交后，服务层必须在进入AI调用前再次读取并匹配该守卫；
// AI返回后的最终写入仍由页面CAS事务再次验证。
type BeginCoursewareReviewItemApplicationResult struct {
	PageID               string
	PageNumber           int
	PageHTMLHash         string
	InstructionVersionID string
}

// BeginCoursewareReviewItemApplicationWithVersion 原子开始一次页面应用。
func BeginCoursewareReviewItemApplicationWithVersion(
	ctx context.Context,
	input *BeginCoursewareReviewItemApplicationInput,
) (*BeginCoursewareReviewItemApplicationResult, error) {
	if input == nil {
		return nil, errors.New("课件整改项页面应用输入不能为空")
	}

	itemID := strings.TrimSpace(input.ItemID)
	actorID := strings.TrimSpace(input.ActorID)
	coursewareID := strings.TrimSpace(input.CoursewareID)
	instructionVersionID := strings.TrimSpace(input.InstructionVersionID)
	submittedInstruction := strings.TrimSpace(input.SubmittedInstruction)

	if itemID == "" || actorID == "" {
		return nil, ErrCoursewareReviewItemNotFound
	}
	if coursewareID == "" || input.PageNumber <= 0 {
		return nil, ErrCoursewareReviewItemApplicationPageMismatch
	}
	if instructionVersionID == "" {
		return nil, ErrCoursewareReviewItemApplicationVersionMismatch
	}
	if submittedInstruction == "" {
		return nil, ErrCoursewareReviewItemApplicationInstructionMismatch
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"开始课件整改项页面应用事务失败: %w",
			err,
		)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var (
		itemCoursewareID   string
		sourceType         string
		itemStatus         string
		pageID             string
		itemPageHash       string
		currentVersionID   string
		deliveredVersionID string
		appliedVersionID   string
		alreadyApplied     bool
	)

	err = tx.QueryRow(
		ctx,
		`SELECT
			courseware_id,
			source_type,
			status,
			COALESCE(page_id::text, ''),
			COALESCE(page_html_hash, ''),
			COALESCE(current_instruction_version_id::text, ''),
			COALESCE(delivered_instruction_version_id::text, ''),
			COALESCE(applied_instruction_version_id::text, ''),
			(applied_at IS NOT NULL)
		 FROM courseware_review_items
		 WHERE id = $1
		   AND owner_id = $2
		 FOR UPDATE`,
		itemID,
		actorID,
	).Scan(
		&itemCoursewareID,
		&sourceType,
		&itemStatus,
		&pageID,
		&itemPageHash,
		&currentVersionID,
		&deliveredVersionID,
		&appliedVersionID,
		&alreadyApplied,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewItemNotFound
		}
		return nil, fmt.Errorf(
			"锁定待应用课件整改项失败: %w",
			err,
		)
	}

	if itemCoursewareID != coursewareID {
		return nil, ErrCoursewareReviewItemNotFound
	}
	if itemStatus != models.CWReviewItemStatusConfirmed ||
		appliedVersionID != "" ||
		alreadyApplied {
		return nil, ErrCoursewareReviewItemConflict
	}
	if pageID == "" {
		return nil, ErrCoursewareReviewItemApplicationPageMismatch
	}

	if sourceType == models.CWReviewItemSourceFormal &&
		deliveredVersionID == "" {
		return nil, ErrCoursewareReviewItemApplicationNotDelivered
	}

	expectedVersionID := deliveredVersionID
	if expectedVersionID == "" {
		expectedVersionID = currentVersionID
	}

	if expectedVersionID == "" ||
		instructionVersionID != expectedVersionID {
		return nil, ErrCoursewareReviewItemApplicationVersionMismatch
	}

	var (
		versionContent  string
		versionStatus   string
		versionPageHash string
	)

	err = tx.QueryRow(
		ctx,
		`SELECT
			content,
			status,
			COALESCE(page_snapshot_hash, '')
		 FROM courseware_review_instruction_versions
		 WHERE id = $1
		   AND item_id = $2
		 FOR SHARE`,
		instructionVersionID,
		itemID,
	).Scan(
		&versionContent,
		&versionStatus,
		&versionPageHash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCoursewareReviewItemApplicationVersionMismatch
		}
		return nil, fmt.Errorf(
			"读取页面应用指令版本失败: %w",
			err,
		)
	}

	if versionStatus != models.CWReviewInstructionVersionStatusConfirmed {
		return nil, ErrCoursewareReviewItemApplicationVersionMismatch
	}

	trustedInstruction := strings.TrimSpace(versionContent)
	if trustedInstruction == "" ||
		!strings.Contains(submittedInstruction, trustedInstruction) {
		return nil, ErrCoursewareReviewItemApplicationInstructionMismatch
	}

	var (
		currentPageNumber int
		currentPageHash   string
	)

	err = tx.QueryRow(
		ctx,
		`SELECT
			page_number,
			encode(
				digest(
					convert_to(
						COALESCE(html_content, ''),
						'UTF8'
					),
					'sha256'
				),
				'hex'
			)
		 FROM courseware_pages
		 WHERE id = $1
		   AND courseware_id = $2
		 FOR SHARE`,
		pageID,
		coursewareID,
	).Scan(
		&currentPageNumber,
		&currentPageHash,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if markErr := markCoursewareReviewItemApplicationInvalidTx(
				ctx,
				tx,
				itemID,
				models.CWReviewItemStatusOrphaned,
			); markErr != nil {
				return nil, markErr
			}

			if commitErr := tx.Commit(ctx); commitErr != nil {
				return nil, fmt.Errorf(
					"提交整改项页面删除状态失败: %w",
					commitErr,
				)
			}

			return nil, ErrCoursewareReviewItemApplicationPageOrphaned
		}

		return nil, fmt.Errorf(
			"读取整改项当前页面失败: %w",
			err,
		)
	}

	if currentPageNumber != input.PageNumber {
		return nil, ErrCoursewareReviewItemApplicationPageMismatch
	}

	itemPageHash = strings.TrimSpace(itemPageHash)
	versionPageHash = strings.TrimSpace(versionPageHash)
	currentPageHash = strings.ToLower(strings.TrimSpace(currentPageHash))

	if itemPageHash == "" ||
		versionPageHash == "" ||
		currentPageHash != itemPageHash ||
		currentPageHash != versionPageHash {
		if err := markCoursewareReviewItemApplicationInvalidTx(
			ctx,
			tx,
			itemID,
			models.CWReviewItemStatusStale,
		); err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf(
				"提交整改项页面变化状态失败: %w",
				err,
			)
		}

		return nil, ErrCoursewareReviewItemApplicationPageStale
	}

	updateResult, err := tx.Exec(
		ctx,
		`UPDATE courseware_review_items
		 SET
			status = 'applying',
			applied_instruction_version_id = $3,
			updated_at = NOW()
		 WHERE id = $1
		   AND owner_id = $2
		   AND status = 'confirmed'
		   AND applied_instruction_version_id IS NULL
		   AND applied_at IS NULL
		   AND COALESCE(
				delivered_instruction_version_id,
				current_instruction_version_id
		   ) = $3`,
		itemID,
		actorID,
		instructionVersionID,
	)
	if err != nil {
		return nil, mapCoursewareReviewItemApplicationWriteError(
			"绑定页面应用指令版本失败",
			err,
		)
	}
	if updateResult.RowsAffected() != 1 {
		return nil, ErrCoursewareReviewItemApplicationVersionMismatch
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, mapCoursewareReviewItemApplicationWriteError(
			"提交课件整改项页面应用事务失败",
			err,
		)
	}

	return &BeginCoursewareReviewItemApplicationResult{
		PageID:               pageID,
		PageNumber:           currentPageNumber,
		PageHTMLHash:         currentPageHash,
		InstructionVersionID: instructionVersionID,
	}, nil
}

func markCoursewareReviewItemApplicationInvalidTx(
	ctx context.Context,
	tx pgx.Tx,
	itemID string,
	nextStatus string,
) error {
	result, err := tx.Exec(
		ctx,
		`UPDATE courseware_review_items
		 SET
			status = $2,
			updated_at = NOW()
		 WHERE id = $1
		   AND status = 'confirmed'
		   AND applied_instruction_version_id IS NULL
		   AND applied_at IS NULL`,
		itemID,
		nextStatus,
	)
	if err != nil {
		return mapCoursewareReviewItemApplicationWriteError(
			"标记整改项页面失效状态失败",
			err,
		)
	}
	if result.RowsAffected() != 1 {
		return ErrCoursewareReviewItemConflict
	}

	return nil
}

func mapCoursewareReviewItemApplicationWriteError(
	message string,
	err error,
) error {
	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503", "23505", "23514", "40001", "P0001":
			return ErrCoursewareReviewItemConflict
		}
	}

	return fmt.Errorf("%s: %w", message, err)
}
