package repository

// courseware_review_item_page_repo.go
//
// 课件审核整改项使用的稳定页面快照读取仓储。
//
// 与按页码读取不同，本文件始终使用page_id和courseware_id联合查询：
//   - 页面重新排序后仍能定位原页面；
//   - 页面归属发生异常时直接查询失败；
//   - 页面被删除后返回明确的不存在错误；
//   - Service负责比较审核快照HTML哈希与当前HTML哈希。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
)

// ErrCoursewareReviewPageSnapshotNotFound 表示审核快照对应的正式页面已不存在。
var ErrCoursewareReviewPageSnapshotNotFound = errors.New(
	"课件审核意见对应的页面已不存在",
)

// CoursewareReviewPageSnapshot 是整改服务需要的当前正式页面数据。
type CoursewareReviewPageSnapshot struct {
	ID           string
	CoursewareID string
	PageNumber   int
	Title        string
	HTMLContent  string
	UpdatedAt    *time.Time
}

// GetCoursewareReviewPageSnapshotByID 按稳定页面ID读取当前正式页面。
func GetCoursewareReviewPageSnapshotByID(
	ctx context.Context,
	pageID string,
	coursewareID string,
) (*CoursewareReviewPageSnapshot, error) {
	item := &CoursewareReviewPageSnapshot{}

	err := database.DB.QueryRow(
		ctx,
		`
		SELECT
			id,
			courseware_id,
			page_number,
			COALESCE(title, ''),
			COALESCE(html_content, ''),
			updated_at
		FROM courseware_pages
		WHERE id = $1
		  AND courseware_id = $2`,
		strings.TrimSpace(pageID),
		strings.TrimSpace(coursewareID),
	).Scan(
		&item.ID,
		&item.CoursewareID,
		&item.PageNumber,
		&item.Title,
		&item.HTMLContent,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil,
				ErrCoursewareReviewPageSnapshotNotFound
		}

		return nil, fmt.Errorf(
			"读取课件审核意见对应页面失败: %w",
			err,
		)
	}

	return item, nil
}
