package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const r03IntegrationPage1CurrentHTML = `<div data-r03="current-page-1-after-review">` +
	`<h1>一次函数导入</h1>` +
	`<p>作者在审核后修改：先让学生独立思考，再根据回答逐步呈现结论。</p>` +
	`</div>`

type r03IntegrationCurrentPageState struct {
	PageID              string
	CoursewareID        string
	PageNumber          int
	HTMLContent         string
	PlaceholderMap      string
	MatchedComponentIDs string
	Status              string
}

func r03IntegrationHTMLHash(
	html string,
) string {
	sum := sha256.Sum256([]byte(html))

	return fmt.Sprintf(
		"%x",
		sum[:],
	)
}

func readR03IntegrationCurrentPageState(
	t *testing.T,
	ctx context.Context,
) r03IntegrationCurrentPageState {
	t.Helper()

	var result r03IntegrationCurrentPageState

	err := database.DB.QueryRow(
		ctx,
		`
		SELECT
			id::text,
			courseware_id::text,
			page_number,
			COALESCE(html_content, ''),
			COALESCE(placeholder_map::text, ''),
			COALESCE(matched_component_ids::text, ''),
			status
		FROM courseware_pages
		WHERE id = $1
		  AND courseware_id = $2`,
		r03IntegrationPage1ID,
		r03IntegrationCoursewareID,
	).Scan(
		&result.PageID,
		&result.CoursewareID,
		&result.PageNumber,
		&result.HTMLContent,
		&result.PlaceholderMap,
		&result.MatchedComponentIDs,
		&result.Status,
	)
	if err != nil {
		t.Fatalf(
			"读取时间点C当前P1失败: %v",
			err,
		)
	}

	return result
}

func findR03HistoricalPageByID(
	detail *models.CoursewareReviewHistoryDetail,
	pageID string,
) (
	models.CoursewareReviewHistoryPage,
	bool,
) {
	if detail == nil {
		return models.CoursewareReviewHistoryPage{}, false
	}

	for _, page := range detail.HistoricalPages {
		if page.PageID == pageID {
			return page, true
		}
	}

	return models.CoursewareReviewHistoryPage{}, false
}

func findR03CurrentPageByID(
	detail *models.CoursewareReviewHistoryDetail,
	pageID string,
) (
	models.CoursewareReviewHistoryCurrentPage,
	bool,
) {
	if detail == nil {
		return models.CoursewareReviewHistoryCurrentPage{}, false
	}

	for _, page := range detail.CurrentPages {
		if page.PageID == pageID {
			return page, true
		}
	}

	return models.CoursewareReviewHistoryCurrentPage{}, false
}

func assertR03HistoryAfterCurrentPageMutation(
	t *testing.T,
	detail *models.CoursewareReviewHistoryDetail,
) {
	t.Helper()

	if detail == nil {
		t.Fatal("时间点C历史详情为空")
	}

	// 正式审核决定仍必须是原记录事实。
	if detail.ReviewID != r03IntegrationReviewID ||
		detail.Review.ReviewLevel != models.ReviewLevelL1 ||
		detail.Review.ReviewRound != 2 ||
		detail.Review.Decision != models.ReviewDecisionRevision ||
		detail.Review.Comment != r03IntegrationReviewComment ||
		detail.Review.Score == nil ||
		*detail.Review.Score != 82.5 {
		t.Fatalf(
			"页面变化污染了原审核决定: %+v",
			detail.Review,
		)
	}

	// R-02配置仍来自审核时不可变Session。
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
			"页面变化污染了审核配置: %+v",
			detail.ReviewConfig,
		)
	}

	if len(detail.Issues) != 1 {
		t.Fatalf(
			"页面变化后正式交付问题数量异常: %d",
			len(detail.Issues),
		)
	}

	issue := detail.Issues[0]

	if issue.ID != r03IntegrationItemID ||
		issue.PageID == nil ||
		*issue.PageID != r03IntegrationPage1ID {
		t.Fatalf(
			"历史问题稳定页面身份变化: %+v",
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
			"当前页面变化污染了历史交付V1: %+v",
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
			"当前页面变化污染了冻结教师视图: %+v",
			issue.TeacherView,
		)
	}

	if !detail.HistoricalPagesAvailable {
		t.Fatalf(
			"页面变化后历史页面不可用: %q",
			detail.HistoricalPagesUnavailableReason,
		)
	}

	if len(detail.HistoricalPages) != 2 {
		t.Fatalf(
			"历史页面数量被当前编辑影响: %d",
			len(detail.HistoricalPages),
		)
	}

	if len(detail.CurrentPages) != 2 {
		t.Fatalf(
			"当前页面数量异常: %d",
			len(detail.CurrentPages),
		)
	}

	historicalP1, ok :=
		findR03HistoricalPageByID(
			detail,
			r03IntegrationPage1ID,
		)
	if !ok {
		t.Fatal("历史页面集合缺少P1")
	}

	if historicalP1.PageNumber != 1 ||
		historicalP1.PageTitle != "一次函数导入" ||
		historicalP1.HTMLContent !=
			r03IntegrationPage1HTML ||
		!historicalP1.CurrentExists {
		t.Fatalf(
			"审核时P1被当前编辑污染: %+v",
			historicalP1,
		)
	}

	currentP1, ok :=
		findR03CurrentPageByID(
			detail,
			r03IntegrationPage1ID,
		)
	if !ok {
		t.Fatal("当前页面集合缺少P1")
	}

	if currentP1.PageNumber != 1 ||
		currentP1.PageTitle != "一次函数导入" ||
		currentP1.HTMLContent !=
			r03IntegrationPage1CurrentHTML {
		t.Fatalf(
			"当前P1没有显示后续修改内容: %+v",
			currentP1,
		)
	}

	if currentP1.HTMLContent ==
		historicalP1.HTMLContent {
		t.Fatal(
			"审核时页面与当前页面被错误合并为同一HTML",
		)
	}

	// P2未修改，继续保持当前存在。
	historicalP2, ok :=
		findR03HistoricalPageByID(
			detail,
			r03IntegrationPage2ID,
		)
	if !ok ||
		historicalP2.HTMLContent !=
			r03IntegrationPage2HTML ||
		!historicalP2.CurrentExists {
		t.Fatalf(
			"未修改P2的历史事实异常: %+v",
			historicalP2,
		)
	}

	currentP2, ok :=
		findR03CurrentPageByID(
			detail,
			r03IntegrationPage2ID,
		)
	if !ok ||
		currentP2.HTMLContent !=
			r03IntegrationPage2HTML {
		t.Fatalf(
			"未修改P2的当前事实异常: %+v",
			currentP2,
		)
	}
}

// TestR03ReviewHistoryIntegrationCurrentPageMutation 验证时间点C：
//
//   - 使用生产页面CAS Repository修改当前P1；
//   - CAS事务必须保存修改前的页面完整版本；
//   - R-03审核时页面仍返回审核提交时冻结HTML；
//   - 当前页面Tab返回修改后的HTML；
//   - 两个时间点共享稳定page_id，但绝不能共享HTML事实；
//   - review、R-02配置、教师视图、delivered V1全部保持不变。
//
// 测试可重复运行：当前页已经是时间点C HTML时只复核最终不变量。
func TestR03ReviewHistoryIntegrationCurrentPageMutation(
	t *testing.T,
) {
	ctx := openR03IntegrationDatabase(t)

	itemState :=
		readR03IntegrationItemLifecycleState(
			t,
			ctx,
		)

	assertR03IntegrationAppliedWithV1(
		t,
		itemState,
	)

	pageBefore :=
		readR03IntegrationCurrentPageState(
			t,
			ctx,
		)

	if pageBefore.PageID !=
		r03IntegrationPage1ID ||
		pageBefore.CoursewareID !=
			r03IntegrationCoursewareID ||
		pageBefore.PageNumber != 1 {
		t.Fatalf(
			"时间点C P1身份异常: %+v",
			pageBefore,
		)
	}

	switch pageBefore.HTMLContent {
	case r03IntegrationPage1HTML:
		result, err :=
			repository.UpdateCWPageHTMLWithVersionCAS(
				ctx,
				&repository.CoursewarePageCASWriteInput{
					PageID: pageBefore.PageID,

					CoursewareID: pageBefore.CoursewareID,

					PageNumber: pageBefore.PageNumber,

					ExpectedHTMLHash: r03IntegrationHTMLHash(
						pageBefore.HTMLContent,
					),

					ExpectedPlaceholderMap: pageBefore.PlaceholderMap,

					ExpectedMatchedComponentIDs: pageBefore.MatchedComponentIDs,

					ExpectedPageStatus: pageBefore.Status,

					NewHTMLContent: r03IntegrationPage1CurrentHTML,

					NewPlaceholderMap: pageBefore.PlaceholderMap,

					NewMatchedComponentIDs: pageBefore.MatchedComponentIDs,

					NewPageStatus: pageBefore.Status,

					VersionSource: models.CWPageVersionSourceManual,

					VersionNote: "R-03时间点C当前页面修改前",
				},
			)
		if err != nil {
			t.Fatalf(
				"时间点C页面CAS修改失败: %v",
				err,
			)
		}

		if result == nil ||
			strings.TrimSpace(result.VersionID) == "" ||
			result.VersionNo <= 0 {
			t.Fatalf(
				"页面CAS没有保存修改前版本: %+v",
				result,
			)
		}

		savedVersion, err :=
			repository.GetPageVersion(
				ctx,
				result.VersionID,
				r03IntegrationPage1ID,
				r03IntegrationCoursewareID,
			)
		if err != nil {
			t.Fatalf(
				"读取CAS保存的旧页面版本失败: %v",
				err,
			)
		}

		if savedVersion.HTMLContent !=
			r03IntegrationPage1HTML ||
			!savedVersion.MetadataSnapshotComplete ||
			savedVersion.Source !=
				models.CWPageVersionSourceManual {
			t.Fatalf(
				"CAS保存的修改前页面版本异常: %+v",
				savedVersion,
			)
		}

	case r03IntegrationPage1CurrentHTML:
		// 已成功执行过时间点C，允许只复核。

	default:
		t.Fatalf(
			"当前P1不是时间点B基线，也不是时间点C目标HTML: %q",
			pageBefore.HTMLContent,
		)
	}

	pageAfter :=
		readR03IntegrationCurrentPageState(
			t,
			ctx,
		)

	if pageAfter.HTMLContent !=
		r03IntegrationPage1CurrentHTML {
		t.Fatalf(
			"CAS后当前P1未更新: %q",
			pageAfter.HTMLContent,
		)
	}

	if pageAfter.PageID !=
		r03IntegrationPage1ID ||
		pageAfter.PageNumber != 1 {
		t.Fatalf(
			"CAS修改改变了稳定页面身份: %+v",
			pageAfter,
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
			"时间点C读取历史详情失败: %v",
			err,
		)
	}

	assertR03HistoryAfterCurrentPageMutation(
		t,
		detail,
	)
}
