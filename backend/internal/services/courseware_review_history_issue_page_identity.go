package services

// courseware_review_history_issue_page_identity.go
//
// R-03已审核历史中的稳定页面身份恢复。
//
// 当前courseware_review_items.page_id属于当前页面引用。
// 原页面删除后，该字段会因ON DELETE SET NULL变为空。
//
// 历史详情不能因此丢失正式审核时的page_id。
// 新R-03记录必须从本次review不可变页面快照恢复历史身份；
// legacy审核没有页面快照时，不伪造已经丢失的page_id。

import (
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func resolveCWReviewHistoryIssuePageID(
	item *models.CoursewareReviewItem,
	snapshots []*repository.CoursewareReviewHistoricalPageSnapshot,
) (*string, error) {
	if item == nil {
		return nil, errors.New("课件审核历史整改项为空")
	}

	currentPageID := ""
	if item.PageID != nil {
		currentPageID = strings.TrimSpace(*item.PageID)
	}

	// R-03上线前的legacy审核没有不可变页面快照。
	if len(snapshots) == 0 {
		if currentPageID == "" {
			return nil, nil
		}

		result := currentPageID
		return &result, nil
	}

	// 当前page_id还存在时，只把它作为与本次review快照的交叉验证。
	if currentPageID != "" {
		for _, snapshot := range snapshots {
			if snapshot == nil {
				return nil, errors.New(
					"课件审核历史页面快照存在空记录",
				)
			}

			if strings.TrimSpace(snapshot.PageID) != currentPageID {
				continue
			}

			if err := validateCWReviewHistoryIssuePageHash(
				item,
				snapshot,
			); err != nil {
				return nil, err
			}

			result := strings.TrimSpace(snapshot.PageID)
			return &result, nil
		}

		return nil, errors.New(
			"课件审核历史整改项page_id不属于本次审核页面快照",
		)
	}

	// 非页级问题本身没有稳定页面身份。
	if item.PageNumberSnapshot <= 0 {
		return nil, nil
	}

	candidates :=
		make(
			[]*repository.CoursewareReviewHistoricalPageSnapshot,
			0,
		)

	itemHash := strings.TrimSpace(item.PageHTMLHash)

	for _, snapshot := range snapshots {
		if snapshot == nil {
			return nil, errors.New(
				"课件审核历史页面快照存在空记录",
			)
		}

		// 正常R-03页级问题优先按审核时HTML hash恢复。
		if itemHash != "" {
			if strings.EqualFold(
				strings.TrimSpace(snapshot.HTMLHash),
				itemHash,
			) {
				candidates = append(candidates, snapshot)
			}

			continue
		}

		// 极早期异常记录如果缺少hash，只能退化使用审核时页码。
		if snapshot.PageNumberSnapshot == item.PageNumberSnapshot {
			candidates = append(candidates, snapshot)
		}
	}

	// 相同HTML可能出现在多页，使用审核时页码消歧。
	if len(candidates) > 1 {
		candidates =
			filterCWReviewHistorySnapshotsByPageNumber(
				candidates,
				item.PageNumberSnapshot,
			)
	}

	// 极端情况下继续使用审核时标题消歧。
	if len(candidates) > 1 &&
		strings.TrimSpace(item.PageTitleSnapshot) != "" {
		candidates =
			filterCWReviewHistorySnapshotsByTitle(
				candidates,
				item.PageTitleSnapshot,
			)
	}

	if len(candidates) == 0 {
		return nil, errors.New(
			"无法从本次审核不可变页面快照恢复整改项稳定页面身份",
		)
	}

	if len(candidates) != 1 {
		return nil, errors.New(
			"本次审核不可变页面快照无法唯一确定整改项稳定页面身份",
		)
	}

	result := strings.TrimSpace(candidates[0].PageID)

	if result == "" {
		return nil, errors.New(
			"恢复出的课件审核历史稳定page_id为空",
		)
	}

	return &result, nil
}

func validateCWReviewHistoryIssuePageHash(
	item *models.CoursewareReviewItem,
	snapshot *repository.CoursewareReviewHistoricalPageSnapshot,
) error {
	itemHash := strings.TrimSpace(item.PageHTMLHash)

	if itemHash == "" {
		return nil
	}

	if !strings.EqualFold(
		strings.TrimSpace(snapshot.HTMLHash),
		itemHash,
	) {
		return errors.New(
			"课件审核历史整改项页面哈希与审核快照不一致",
		)
	}

	return nil
}

func filterCWReviewHistorySnapshotsByPageNumber(
	snapshots []*repository.CoursewareReviewHistoricalPageSnapshot,
	pageNumber int,
) []*repository.CoursewareReviewHistoricalPageSnapshot {
	filtered :=
		make(
			[]*repository.CoursewareReviewHistoricalPageSnapshot,
			0,
		)

	for _, snapshot := range snapshots {
		if snapshot != nil &&
			snapshot.PageNumberSnapshot == pageNumber {
			filtered = append(filtered, snapshot)
		}
	}

	if len(filtered) == 0 {
		return snapshots
	}

	return filtered
}

func filterCWReviewHistorySnapshotsByTitle(
	snapshots []*repository.CoursewareReviewHistoricalPageSnapshot,
	pageTitle string,
) []*repository.CoursewareReviewHistoricalPageSnapshot {
	pageTitle = strings.TrimSpace(pageTitle)

	filtered :=
		make(
			[]*repository.CoursewareReviewHistoricalPageSnapshot,
			0,
		)

	for _, snapshot := range snapshots {
		if snapshot != nil &&
			strings.TrimSpace(snapshot.PageTitleSnapshot) == pageTitle {
			filtered = append(filtered, snapshot)
		}
	}

	if len(filtered) == 0 {
		return snapshots
	}

	return filtered
}
