package services

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

type r03IntegrationPageByIDState struct {
	ID           string
	CoursewareID string
	PageNumber   int
	Title        string
	HTMLContent  string
}

func readR03IntegrationPageByID(
	t *testing.T,
	ctx context.Context,
	pageID string,
) (
	r03IntegrationPageByIDState,
	bool,
) {
	t.Helper()

	var result r03IntegrationPageByIDState

	err := database.DB.QueryRow(
		ctx,
		`
		SELECT
			id::text,
			courseware_id::text,
			page_number,
			COALESCE(title, ''),
			COALESCE(html_content, '')
		FROM courseware_pages
		WHERE id = $1
		  AND courseware_id = $2`,
		pageID,
		r03IntegrationCoursewareID,
	).Scan(
		&result.ID,
		&result.CoursewareID,
		&result.PageNumber,
		&result.Title,
		&result.HTMLContent,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r03IntegrationPageByIDState{}, false
		}

		t.Fatalf(
			"读取R-03页面失败(page_id=%s): %v",
			pageID,
			err,
		)
	}

	return result, true
}

func assertR03HistoryAfterOriginalPageDeleted(
	t *testing.T,
	detail *models.CoursewareReviewHistoryDetail,
) {
	t.Helper()

	if detail == nil {
		t.Fatal("时间点D历史详情为空")
	}

	if detail.ReviewID != r03IntegrationReviewID ||
		detail.Review.ReviewLevel != models.ReviewLevelL1 ||
		detail.Review.ReviewRound != 2 ||
		detail.Review.Decision != models.ReviewDecisionRevision ||
		detail.Review.Comment != r03IntegrationReviewComment ||
		detail.Review.Score == nil ||
		*detail.Review.Score != 82.5 {
		t.Fatalf(
			"删除当前原页面污染了正式审核决定: %+v",
			detail.Review,
		)
	}

	if !detail.ReviewConfig.Available ||
		detail.ReviewConfig.SchemaVersion !=
			models.CWAIReviewConfigSchemaVersion ||
		detail.ReviewConfig.CustomFocus !=
			r03IntegrationCustomFocus ||
		detail.ReviewConfig.LessonReferenceMode !=
			models.CWAIReviewLessonReferenceNoLesson ||
		detail.ReviewConfig.LessonMaterialsUsed == nil ||
		*detail.ReviewConfig.LessonMaterialsUsed {
		t.Fatalf(
			"删除当前原页面污染了审核配置: %+v",
			detail.ReviewConfig,
		)
	}

	if !detail.HistoricalPagesAvailable {
		t.Fatalf(
			"删除当前原页面后历史页面不可用: %q",
			detail.HistoricalPagesUnavailableReason,
		)
	}

	if len(detail.HistoricalPages) != 2 {
		t.Fatalf(
			"删除当前P1后历史页面数量异常: %d",
			len(detail.HistoricalPages),
		)
	}

	if len(detail.CurrentPages) != 1 {
		t.Fatalf(
			"删除当前P1后当前页面数量异常: %d",
			len(detail.CurrentPages),
		)
	}

	historicalP1, ok :=
		findR03HistoricalPageByID(
			detail,
			r03IntegrationPage1ID,
		)
	if !ok {
		t.Fatal("删除当前P1后历史集合丢失审核时P1")
	}

	if historicalP1.PageNumber != 1 ||
		historicalP1.PageTitle != "一次函数导入" ||
		historicalP1.HTMLContent !=
			r03IntegrationPage1HTML ||
		historicalP1.CurrentExists {
		t.Fatalf(
			"删除当前P1后历史P1事实异常: %+v",
			historicalP1,
		)
	}

	historicalP2, ok :=
		findR03HistoricalPageByID(
			detail,
			r03IntegrationPage2ID,
		)
	if !ok {
		t.Fatal("删除当前P1后历史集合丢失审核时P2")
	}

	if historicalP2.PageNumber != 2 ||
		historicalP2.PageTitle != "函数图像观察" ||
		historicalP2.HTMLContent !=
			r03IntegrationPage2HTML ||
		!historicalP2.CurrentExists {
		t.Fatalf(
			"删除P1后历史P2事实异常: %+v",
			historicalP2,
		)
	}

	currentP2, ok :=
		findR03CurrentPageByID(
			detail,
			r03IntegrationPage2ID,
		)
	if !ok {
		t.Fatal("删除P1后当前页面集合缺少P2")
	}

	if currentP2.PageNumber != 1 ||
		currentP2.PageTitle != "函数图像观察" ||
		currentP2.HTMLContent !=
			r03IntegrationPage2HTML {
		t.Fatalf(
			"删除P1后的当前P2校准结果异常: %+v",
			currentP2,
		)
	}

	if !detail.IssuesAvailable {
		t.Fatalf(
			"删除原页面后正式交付问题不可用: %q",
			detail.IssuesUnavailableReason,
		)
	}

	if len(detail.Issues) != 1 {
		t.Fatalf(
			"删除原页面后正式交付问题数量异常: %d",
			len(detail.Issues),
		)
	}

	issue := detail.Issues[0]

	if issue.ID != r03IntegrationItemID ||
		issue.PageNumber != 1 ||
		issue.PageTitle != "一次函数导入" {
		t.Fatalf(
			"删除原页面污染了历史问题页面快照: %+v",
			issue,
		)
	}

	if !issue.DeliveredInstructionAvailable ||
		issue.DeliveredInstruction == nil ||
		issue.DeliveredInstruction.VersionID !=
			r03IntegrationV1ID ||
		issue.DeliveredInstruction.VersionNo != 1 ||
		issue.DeliveredInstruction.Content !=
			r03IntegrationV1Content {
		t.Fatalf(
			"删除原页面污染了历史交付V1: %+v",
			issue.DeliveredInstruction,
		)
	}

	if issue.TeacherView.TeacherTitle !=
		r03IntegrationTeacherTitle ||
		issue.TeacherView.WhatHappened !=
			r03IntegrationWhatHappened ||
		issue.TeacherView.TeachingImpact !=
			r03IntegrationTeachingImpact ||
		issue.TeacherView.ImprovementGoal !=
			r03IntegrationImprovementGoal ||
		!issue.TeacherView.ManualCheckRequired ||
		!slices.Equal(
			issue.TeacherView.AcceptanceChecks,
			r03IntegrationAcceptanceChecks,
		) {
		t.Fatalf(
			"删除原页面污染了冻结教师视图: %+v",
			issue.TeacherView,
		)
	}

	// R-03关键验收：
	//
	// 当前courseware_review_items.page_id会因FK ON DELETE SET NULL变空，
	// 但历史详情必须继续使用审核时不可变页面身份，不能把页级问题变成整课问题。
	//
	// 当前实现若仍直接读取item.PageID，这里应先真实失败。
	if issue.PageID == nil ||
		*issue.PageID != r03IntegrationPage1ID {
		t.Fatalf(
			"R03-D-STABLE-PAGE-ID-LOST: "+
				"原页面删除后历史issue必须仍绑定审核时P1，got=%v",
			issue.PageID,
		)
	}
}

// TestR03ReviewHistoryIntegrationDeletedOriginalPage 验证时间点D：
//
//   - 从时间点C状态出发，真实执行既有DeletePage业务链删除P1；
//   - 删除后P2校准成当前第1页，coursewares.page_count同步为1；
//   - courseware_review_items.page_id按既有FK变为NULL；
//   - 审核时P1快照仍完整，并标记CurrentExists=false；
//   - 正式review/config/teacher_view/delivered V1全部保持历史事实；
//   - History issue必须从不可变历史事实恢复稳定P1身份。
//
// 测试可重复运行：若P1已经删除，只执行后置不变量复核，绝不再次按页码删除P2。
func TestR03ReviewHistoryIntegrationDeletedOriginalPage(
	t *testing.T,
) {
	ctx := openR03IntegrationDatabase(t)

	beforeLifecycle :=
		readR03IntegrationItemLifecycleState(
			t,
			ctx,
		)

	assertR03IntegrationAppliedWithV1(
		t,
		beforeLifecycle,
	)

	p1Before, p1Exists :=
		readR03IntegrationPageByID(
			t,
			ctx,
			r03IntegrationPage1ID,
		)

	if p1Exists {
		if p1Before.PageNumber != 1 ||
			p1Before.Title != "一次函数导入" ||
			p1Before.HTMLContent !=
				r03IntegrationPage1CurrentHTML {
			t.Fatalf(
				"时间点D删除前P1不是时间点C状态: %+v",
				p1Before,
			)
		}

		err :=
			NewCoursewareService().
				DeletePage(
					ctx,
					r03IntegrationCoursewareID,
					1,
					r03IntegrationAuthorID,
				)
		if err != nil {
			t.Fatalf(
				"时间点D真实DeletePage失败: %v",
				err,
			)
		}
	}

	if _, stillExists :=
		readR03IntegrationPageByID(
			t,
			ctx,
			r03IntegrationPage1ID,
		); stillExists {
		t.Fatal("时间点D后P1仍存在")
	}

	pages, err :=
		repository.ListCoursewarePages(
			ctx,
			r03IntegrationCoursewareID,
		)
	if err != nil {
		t.Fatalf(
			"读取删除后当前页面失败: %v",
			err,
		)
	}

	if len(pages) != 1 ||
		pages[0] == nil ||
		pages[0].ID != r03IntegrationPage2ID ||
		pages[0].PageNumber != 1 ||
		pages[0].Title != "函数图像观察" ||
		pages[0].HTMLContent !=
			r03IntegrationPage2HTML {
		t.Fatalf(
			"DeletePage后的P2校准结果异常: %+v",
			pages,
		)
	}

	var pageCount int

	if err := database.DB.QueryRow(
		ctx,
		`
		SELECT page_count
		FROM coursewares
		WHERE id = $1
		  AND deleted_at IS NULL`,
		r03IntegrationCoursewareID,
	).Scan(&pageCount); err != nil {
		t.Fatalf(
			"读取删除后课件总页数失败: %v",
			err,
		)
	}

	if pageCount != 1 {
		t.Fatalf(
			"删除P1后page_count异常: %d",
			pageCount,
		)
	}

	var (
		itemStatus         string
		itemPageID         string
		pageNumberSnapshot int
		pageTitleSnapshot  string
		currentVersionID   string
		deliveredVersionID string
		appliedVersionID   string
	)

	if err := database.DB.QueryRow(
		ctx,
		`
		SELECT
			status,
			COALESCE(page_id::text, ''),
			page_number_snapshot,
			COALESCE(page_title_snapshot, ''),
			COALESCE(current_instruction_version_id::text, ''),
			COALESCE(delivered_instruction_version_id::text, ''),
			COALESCE(applied_instruction_version_id::text, '')
		FROM courseware_review_items
		WHERE id = $1`,
		r03IntegrationItemID,
	).Scan(
		&itemStatus,
		&itemPageID,
		&pageNumberSnapshot,
		&pageTitleSnapshot,
		&currentVersionID,
		&deliveredVersionID,
		&appliedVersionID,
	); err != nil {
		t.Fatalf(
			"读取删除后整改项状态失败: %v",
			err,
		)
	}

	if itemStatus != models.CWReviewItemStatusApplied ||
		itemPageID != "" ||
		pageNumberSnapshot != 1 ||
		pageTitleSnapshot != "一次函数导入" ||
		currentVersionID != r03IntegrationV1ID ||
		deliveredVersionID != r03IntegrationV1ID ||
		appliedVersionID != r03IntegrationV1ID {
		t.Fatalf(
			"删除P1后的整改项数据库事实异常: "+
				"status=%s page_id=%q page=%d title=%q "+
				"current=%s delivered=%s applied=%s",
			itemStatus,
			itemPageID,
			pageNumberSnapshot,
			pageTitleSnapshot,
			currentVersionID,
			deliveredVersionID,
			appliedVersionID,
		)
	}

	afterLifecycle :=
		readR03IntegrationItemLifecycleState(
			t,
			ctx,
		)

	assertR03IntegrationAppliedWithV1(
		t,
		afterLifecycle,
	)

	var snapshotCount int

	if err := database.DB.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM courseware_review_page_snapshots
		WHERE courseware_review_id = $1`,
		r03IntegrationReviewID,
	).Scan(&snapshotCount); err != nil {
		t.Fatalf(
			"读取删除后历史页面快照数量失败: %v",
			err,
		)
	}

	if snapshotCount != 2 {
		t.Fatalf(
			"删除P1破坏了历史页面快照: %d",
			snapshotCount,
		)
	}

	author :=
		buildR03IntegrationActor(
			t,
			ctx,
			r03IntegrationAuthorID,
			models.RoleOperator,
		)

	detail, err :=
		NewCoursewareReviewService().
			GetReviewHistoryDetail(
				ctx,
				r03IntegrationReviewID,
				author,
			)
	if err != nil {
		t.Fatalf(
			"删除原页面后读取历史详情失败: %v",
			err,
		)
	}

	assertR03HistoryAfterOriginalPageDeleted(
		t,
		detail,
	)
}
