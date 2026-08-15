package repository

// courseware_review_page_snapshot_repo.go
//
// R-03正式审核页面历史快照仓储。
//
// 写入边界：
//   - 只能由正式审核决定事务调用CreateCoursewareReviewPageSnapshotsTx；
//   - 使用调用方已经开启的pgx.Tx，不自行Begin/Commit；
//   - 先锁定并完整读取当前全部页面，再逐页写入不可变快照；
//   - HTML SHA-256由数据库计算，不能信任应用层传入哈希；
//   - 任一页面读取或写入失败，由上层回滚整笔正式审核决定。
//
// 读取边界：
//   - 历史详情只按courseware_review_id读取本表；
//   - 不复用courseware_page_versions编辑恢复链；
//   - page_id不依赖当前courseware_pages，因此原页面删除后仍可读取历史HTML。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
)

var (
	// ErrCoursewareReviewPageSnapshotEmpty 表示正式审核提交时没有任何真实页面可冻结。
	ErrCoursewareReviewPageSnapshotEmpty = errors.New(
		"正式审核提交时课件没有可冻结页面",
	)

	// ErrCoursewareReviewPageSnapshotInvalid 表示页面集合自身存在非法稳定身份。
	ErrCoursewareReviewPageSnapshotInvalid = errors.New(
		"正式审核页面集合不完整或存在重复身份",
	)
)

// CoursewareReviewHistoricalPageSnapshot 是数据库中的R-03不可变页面事实。
type CoursewareReviewHistoricalPageSnapshot struct {
	ID string

	CoursewareReviewID string
	CoursewareID       string

	PageID             string
	PageNumberSnapshot int
	PageTitleSnapshot  string

	HTMLContent string
	HTMLHash    string

	PageUpdatedAtSnapshot *time.Time
	CreatedAt             *time.Time
}

type coursewareReviewSnapshotSourcePage struct {
	ID         string
	PageNumber int
	Title      string
	HTML       string
	UpdatedAt  *time.Time
}

// CreateCoursewareReviewPageSnapshotsTx 在正式审核决定事务中冻结全部当前页面。
//
// 返回成功写入的页面数量。调用方应把返回数量纳入正式审核提交的不变量校验。
func CreateCoursewareReviewPageSnapshotsTx(
	ctx context.Context,
	tx pgx.Tx,
	reviewID string,
	coursewareID string,
) (int64, error) {
	reviewID = strings.TrimSpace(reviewID)
	coursewareID = strings.TrimSpace(coursewareID)

	if tx == nil || reviewID == "" || coursewareID == "" {
		return 0, errors.New("缺少正式审核页面快照事务参数")
	}

	rows, err := tx.Query(
		ctx,
		`
        SELECT
            id,
            page_number,
            COALESCE(title, ''),
            COALESCE(html_content, ''),
            updated_at
        FROM courseware_pages
        WHERE courseware_id = $1
        ORDER BY page_number ASC, id ASC
        FOR SHARE`,
		coursewareID,
	)
	if err != nil {
		return 0, fmt.Errorf(
			"锁定正式审核页面集合失败: %w",
			err,
		)
	}

	pages := make([]coursewareReviewSnapshotSourcePage, 0)

	for rows.Next() {
		var page coursewareReviewSnapshotSourcePage

		if err := rows.Scan(
			&page.ID,
			&page.PageNumber,
			&page.Title,
			&page.HTML,
			&page.UpdatedAt,
		); err != nil {
			rows.Close()
			return 0, fmt.Errorf(
				"读取正式审核页面快照源数据失败: %w",
				err,
			)
		}

		pages = append(pages, page)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf(
			"遍历正式审核页面快照源数据失败: %w",
			err,
		)
	}

	rows.Close()

	if len(pages) == 0 {
		return 0, ErrCoursewareReviewPageSnapshotEmpty
	}

	seenPageIDs := make(map[string]struct{}, len(pages))
	seenPageNumbers := make(map[int]struct{}, len(pages))

	for _, page := range pages {
		pageID := strings.TrimSpace(page.ID)

		if pageID == "" || page.PageNumber <= 0 {
			return 0, ErrCoursewareReviewPageSnapshotInvalid
		}

		if _, exists := seenPageIDs[pageID]; exists {
			return 0, ErrCoursewareReviewPageSnapshotInvalid
		}
		seenPageIDs[pageID] = struct{}{}

		if _, exists := seenPageNumbers[page.PageNumber]; exists {
			return 0, ErrCoursewareReviewPageSnapshotInvalid
		}
		seenPageNumbers[page.PageNumber] = struct{}{}
	}

	var insertedCount int64

	for _, page := range pages {
		result, err := tx.Exec(
			ctx,
			`
            INSERT INTO courseware_review_page_snapshots (
                courseware_review_id,
                courseware_id,
                page_id,
                page_number_snapshot,
                page_title_snapshot,
                html_content,
                html_hash,
                page_updated_at_snapshot,
                created_at
            )
            VALUES (
                $1,
                $2,
                $3,
                $4,
                $5,
                $6,
                encode(
                    digest(
                        convert_to($6, 'UTF8'),
                        'sha256'
                    ),
                    'hex'
                ),
                $7,
                NOW()
            )`,
			reviewID,
			coursewareID,
			strings.TrimSpace(page.ID),
			page.PageNumber,
			page.Title,
			page.HTML,
			page.UpdatedAt,
		)
		if err != nil {
			return 0, fmt.Errorf(
				"写入正式审核第%d页历史快照失败: %w",
				page.PageNumber,
				err,
			)
		}

		if result.RowsAffected() != 1 {
			return 0, fmt.Errorf(
				"%w: 第%d页写入数量=%d",
				ErrCoursewareReviewPageSnapshotInvalid,
				page.PageNumber,
				result.RowsAffected(),
			)
		}

		insertedCount++
	}

	if insertedCount != int64(len(pages)) {
		return 0, fmt.Errorf(
			"%w: 源页面=%d，已写入=%d",
			ErrCoursewareReviewPageSnapshotInvalid,
			len(pages),
			insertedCount,
		)
	}

	return insertedCount, nil
}

// ListCoursewareReviewPageSnapshotsByReviewID 按审核记录读取不可变历史页面。
func ListCoursewareReviewPageSnapshotsByReviewID(
	ctx context.Context,
	reviewID string,
) ([]*CoursewareReviewHistoricalPageSnapshot, error) {
	reviewID = strings.TrimSpace(reviewID)

	if reviewID == "" {
		return nil, errors.New("缺少课件审核记录ID")
	}

	rows, err := database.DB.Query(
		ctx,
		`
        SELECT
            id,
            courseware_review_id,
            courseware_id,
            page_id,
            page_number_snapshot,
            COALESCE(page_title_snapshot, ''),
            COALESCE(html_content, ''),
            html_hash,
            page_updated_at_snapshot,
            created_at
        FROM courseware_review_page_snapshots
        WHERE courseware_review_id = $1
        ORDER BY page_number_snapshot ASC, page_id ASC`,
		reviewID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"查询课件审核历史页面快照失败: %w",
			err,
		)
	}
	defer rows.Close()

	snapshots := make(
		[]*CoursewareReviewHistoricalPageSnapshot,
		0,
	)

	for rows.Next() {
		snapshot := &CoursewareReviewHistoricalPageSnapshot{}

		if err := rows.Scan(
			&snapshot.ID,
			&snapshot.CoursewareReviewID,
			&snapshot.CoursewareID,
			&snapshot.PageID,
			&snapshot.PageNumberSnapshot,
			&snapshot.PageTitleSnapshot,
			&snapshot.HTMLContent,
			&snapshot.HTMLHash,
			&snapshot.PageUpdatedAtSnapshot,
			&snapshot.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描课件审核历史页面快照失败: %w",
				err,
			)
		}

		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历课件审核历史页面快照失败: %w",
			err,
		)
	}

	return snapshots, nil
}
