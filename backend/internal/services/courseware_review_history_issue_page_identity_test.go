package services

import (
	"testing"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestResolveCWReviewHistoryIssuePageIDAfterDelete(
	t *testing.T,
) {
	item :=
		&models.CoursewareReviewItem{
			PageNumberSnapshot: 1,
			PageTitleSnapshot:  "一次函数导入",
			PageHTMLHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
				"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}

	snapshots :=
		[]*repository.CoursewareReviewHistoricalPageSnapshot{
			{
				PageID:             "history-page-1",
				PageNumberSnapshot: 1,
				PageTitleSnapshot:  "一次函数导入",
				HTMLHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		}

	got, err :=
		resolveCWReviewHistoryIssuePageID(
			item,
			snapshots,
		)
	if err != nil {
		t.Fatalf(
			"删除后恢复page_id失败: %v",
			err,
		)
	}

	if got == nil ||
		*got != "history-page-1" {
		t.Fatalf(
			"删除后稳定page_id异常: %v",
			got,
		)
	}
}

func TestResolveCWReviewHistoryIssuePageIDRejectsForeignCurrentID(
	t *testing.T,
) {
	pageID := "foreign-page"

	item :=
		&models.CoursewareReviewItem{
			PageID:             &pageID,
			PageNumberSnapshot: 1,
		}

	snapshots :=
		[]*repository.CoursewareReviewHistoricalPageSnapshot{
			{
				PageID:             "history-page-1",
				PageNumberSnapshot: 1,
			},
		}

	if _, err :=
		resolveCWReviewHistoryIssuePageID(
			item,
			snapshots,
		); err == nil {
		t.Fatal(
			"本次审核快照外的当前page_id必须fail-closed",
		)
	}
}

func TestResolveCWReviewHistoryIssuePageIDAmbiguousHashUsesPageNumber(
	t *testing.T,
) {
	hash :=
		"cccccccccccccccccccccccccccccccc" +
			"cccccccccccccccccccccccccccccccc"

	item :=
		&models.CoursewareReviewItem{
			PageNumberSnapshot: 2,
			PageTitleSnapshot:  "目标页面",
			PageHTMLHash:       hash,
		}

	snapshots :=
		[]*repository.CoursewareReviewHistoricalPageSnapshot{
			{
				PageID:             "history-page-1",
				PageNumberSnapshot: 1,
				PageTitleSnapshot:  "其它页面",
				HTMLHash:           hash,
			},
			{
				PageID:             "history-page-2",
				PageNumberSnapshot: 2,
				PageTitleSnapshot:  "目标页面",
				HTMLHash:           hash,
			},
		}

	got, err :=
		resolveCWReviewHistoryIssuePageID(
			item,
			snapshots,
		)
	if err != nil {
		t.Fatalf(
			"重复hash消歧失败: %v",
			err,
		)
	}

	if got == nil ||
		*got != "history-page-2" {
		t.Fatalf(
			"重复hash应恢复审核时第二页: %v",
			got,
		)
	}
}

func TestResolveCWReviewHistoryIssuePageIDPreservesLegacyCurrentID(
	t *testing.T,
) {
	pageID := "legacy-page"

	item :=
		&models.CoursewareReviewItem{
			PageID:             &pageID,
			PageNumberSnapshot: 1,
		}

	got, err :=
		resolveCWReviewHistoryIssuePageID(
			item,
			nil,
		)
	if err != nil {
		t.Fatalf(
			"legacy当前page_id处理失败: %v",
			err,
		)
	}

	if got == nil ||
		*got != pageID {
		t.Fatalf(
			"legacy仍存在page_id应保持: %v",
			got,
		)
	}
}

func TestResolveCWReviewHistoryIssuePageIDDoesNotInventLegacyDeletedID(
	t *testing.T,
) {
	item :=
		&models.CoursewareReviewItem{
			PageNumberSnapshot: 1,
			PageTitleSnapshot:  "旧审核页面",
		}

	got, err :=
		resolveCWReviewHistoryIssuePageID(
			item,
			nil,
		)
	if err != nil {
		t.Fatalf(
			"legacy删除页面处理失败: %v",
			err,
		)
	}

	if got != nil {
		t.Fatalf(
			"legacy没有不可变快照时不得伪造page_id: %v",
			got,
		)
	}
}
