package services

// courseware_review_history_pages.go
//
// R-03“审核时页面 / 当前页面”双时间点读取。
//
// 历史页永远来自courseware_review_page_snapshots。
// 当前页永远来自当前courseware_pages。
// 两组数据不得互相回退或冒充。
//
// 原页面删除后，历史HTML仍保留，CurrentExists=false。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func buildCWReviewHistoryPages(
	ctx context.Context,
	review *models.CoursewareReview,
) (
	[]models.CoursewareReviewHistoryPage,
	[]models.CoursewareReviewHistoryCurrentPage,
	bool,
	string,
	error,
) {
	if review == nil {
		return nil, nil, false, "",
			errors.New(
				"缺少课件审核历史记录",
			)
	}

	snapshots, err :=
		repository.
			ListCoursewareReviewPageSnapshotsByReviewID(
				ctx,
				review.ID,
			)
	if err != nil {
		return nil, nil, false, "", err
	}

	currentPages, err :=
		repository.ListCoursewarePages(
			ctx,
			review.CoursewareID,
		)
	if err != nil {
		return nil, nil, false, "", err
	}

	currentResult :=
		make(
			[]models.CoursewareReviewHistoryCurrentPage,
			0,
			len(currentPages),
		)

	currentPageIDs :=
		make(
			map[string]bool,
			len(currentPages),
		)

	for _, page := range currentPages {
		if page == nil {
			continue
		}

		if strings.TrimSpace(
			page.CoursewareID,
		) != strings.TrimSpace(
			review.CoursewareID,
		) {
			return nil, nil, false, "",
				errors.New(
					"课件当前页面归属关系异常",
				)
		}

		currentPageIDs[page.ID] = true

		currentResult = append(
			currentResult,
			models.CoursewareReviewHistoryCurrentPage{
				PageID:      page.ID,
				PageNumber:  page.PageNumber,
				PageTitle:   page.Title,
				HTMLContent: page.HTMLContent,
				UpdatedAt:   page.UpdatedAt,
			},
		)
	}

	historicalResult :=
		make(
			[]models.CoursewareReviewHistoryPage,
			0,
			len(snapshots),
		)

	for _, snapshot := range snapshots {
		if snapshot == nil {
			continue
		}

		if strings.TrimSpace(
			snapshot.CoursewareReviewID,
		) != strings.TrimSpace(review.ID) ||
			strings.TrimSpace(
				snapshot.CoursewareID,
			) != strings.TrimSpace(
				review.CoursewareID,
			) {
			return nil, nil, false, "",
				errors.New(
					"课件审核历史页面快照归属异常",
				)
		}

		historicalResult = append(
			historicalResult,
			models.CoursewareReviewHistoryPage{
				PageID: snapshot.PageID,

				PageNumber: snapshot.PageNumberSnapshot,

				PageTitle: snapshot.PageTitleSnapshot,

				HTMLContent: snapshot.HTMLContent,

				PageUpdatedAt: snapshot.PageUpdatedAtSnapshot,

				CurrentExists: currentPageIDs[snapshot.PageID],
			},
		)
	}

	if len(historicalResult) == 0 {
		return historicalResult,
			currentResult,
			false,
			models.
				CWReviewHistoryPagesUnavailableLegacy,
			nil
	}

	return historicalResult,
		currentResult,
		true,
		"",
		nil
}
