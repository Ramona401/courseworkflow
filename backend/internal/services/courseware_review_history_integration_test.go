package services

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tedna/internal/database"
	"tedna/internal/models"
)

// R-03 集成测试固定事实。
//
// 这些 UUID 只属于 tedna_test 中的 R03-INTEGRATION fixture，
// 不允许复用生产数据，也不允许测试自行创建或修改业务数据。
const (
	r03IntegrationSchoolID = "33000000-0000-4000-8000-000000000001"

	r03IntegrationCoursewareID = "33000000-0000-4000-8000-000000000002"
	r03IntegrationPage1ID      = "33000000-0000-4000-8000-000000000003"
	r03IntegrationPage2ID      = "33000000-0000-4000-8000-000000000004"

	r03IntegrationLegacyReviewID = "33000000-0000-4000-8000-000000000006"
	r03IntegrationReviewID       = "33000000-0000-4000-8000-000000000007"

	r03IntegrationItemID = "33000000-0000-4000-8000-000000000009"
	r03IntegrationV1ID   = "33000000-0000-4000-8000-00000000000a"

	r03IntegrationAdminID  = "00000000-0000-0000-0000-000000000001"
	r03IntegrationAuthorID = "00000000-0000-0000-0000-000000000002"
	r03IntegrationSeniorID = "00000000-0000-0000-0000-000000000003"
	r03IntegrationViewerID = "00000000-0000-0000-0000-000000000004"
)

const (
	r03IntegrationReviewComment = "R03正式审核：问题链需要增加学生真实思考与反馈环节。"

	r03IntegrationCustomFocus = "重点检查课堂问题是否给学生留下真实思考时间"

	r03IntegrationV1Content = "V1正式交付：保留导入问题，但先隐藏结论；" +
		"给学生独立思考时间，并由教师根据回答再逐步呈现结论。"

	r03IntegrationTeacherTitle = "让学生先思考，再出现结论"

	r03IntegrationWhatHappened = "导入问题提出后，页面立即展示了结论，" +
		"学生几乎没有独立判断时间。"

	r03IntegrationTeachingImpact = "学生容易直接跟随页面答案，" +
		"课堂提问难以形成真实的思考和反馈。"

	r03IntegrationImprovementGoal = "让学生先形成自己的判断，" +
		"再通过教师追问或页面反馈逐步呈现结论。"

	r03IntegrationPage1HTML = `<div data-r03="history-page-1">` +
		`<h1>审核时页面P1</h1><p>原始课堂导入内容</p></div>`

	r03IntegrationPage2HTML = `<div data-r03="history-page-2">` +
		`<h1>审核时页面P2</h1><p>原始图像观察内容</p></div>`
)

var r03IntegrationAcceptanceChecks = []string{
	"问题提出后不会立即显示最终结论",
	"课堂上能留出明确的学生思考时间",
	"教师可以根据学生回答再推进到结论",
}

// openR03IntegrationDatabase 只允许显式启用后连接 tedna_test。
//
// TEST DSN 由命令行环境传入，不读取、不打印数据库密码。
// 建立连接后再次从 PostgreSQL 查询 current_database()，
// 防止错误 DSN 将集成测试指向生产库。
func openR03IntegrationDatabase(t *testing.T) context.Context {
	t.Helper()

	if strings.TrimSpace(os.Getenv("TEDNA_R03_INTEGRATION")) != "1" {
		t.Skip("R-03数据库集成测试未显式启用")
	}

	dsn := strings.TrimSpace(os.Getenv("TEDNA_R03_TEST_DSN"))
	if dsn == "" {
		t.Fatal("缺少TEDNA_R03_TEST_DSN")
	}

	if database.DB != nil {
		t.Fatal("测试开始前database.DB已经初始化，拒绝覆盖全局连接池")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("创建R-03测试连接池失败: %v", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		t.Fatalf("连接R-03测试数据库失败: %v", err)
	}

	var databaseName string
	if err := pool.QueryRow(
		pingCtx,
		`SELECT current_database()`,
	).Scan(&databaseName); err != nil {
		pool.Close()
		t.Fatalf("确认R-03测试数据库名称失败: %v", err)
	}

	if databaseName != "tedna_test" {
		pool.Close()
		t.Fatalf(
			"R-03集成测试只允许tedna_test，当前数据库=%q",
			databaseName,
		)
	}

	database.DB = pool

	t.Cleanup(func() {
		pool.Close()
		database.DB = nil
	})

	return ctx
}

func buildR03IntegrationActor(
	t *testing.T,
	ctx context.Context,
	userID string,
	role string,
) *CoursewareActorContext {
	t.Helper()

	actor := BuildCoursewareActorFromClaims(
		ctx,
		userID,
		role,
	)
	if actor == nil {
		t.Fatalf("构造Actor失败: user=%s role=%s", userID, role)
	}

	if actor.UserID != userID {
		t.Fatalf(
			"Actor用户身份异常: want=%s got=%s",
			userID,
			actor.UserID,
		)
	}

	if actor.Role != role {
		t.Fatalf(
			"Actor角色异常: want=%s got=%s",
			role,
			actor.Role,
		)
	}

	return actor
}

func assertR03TeachingActorDomain(
	t *testing.T,
	actor *CoursewareActorContext,
) {
	t.Helper()

	if actor.EducationDomain != models.EducationDomainK12 {
		t.Fatalf(
			"测试教学Actor必须由真实组织关系解析为k12，got=%q",
			actor.EducationDomain,
		)
	}
}

func assertR03BaselineHistoryDetail(
	t *testing.T,
	detail *models.CoursewareReviewHistoryDetail,
) {
	t.Helper()

	if detail == nil {
		t.Fatal("R-03新审核历史详情为空")
	}

	if detail.ReviewID != r03IntegrationReviewID {
		t.Fatalf(
			"review_id异常: want=%s got=%s",
			r03IntegrationReviewID,
			detail.ReviewID,
		)
	}

	if detail.Courseware.ID != r03IntegrationCoursewareID ||
		detail.Courseware.Title != "R03-INTEGRATION 已审核历史测试课件" ||
		detail.Courseware.Subject != "数学" ||
		detail.Courseware.Grade != "八年级" {
		t.Fatalf(
			"课件基本信息异常: %+v",
			detail.Courseware,
		)
	}

	if detail.Reviewer.ID != r03IntegrationSeniorID ||
		detail.Reviewer.DisplayName != "高级操作员1" {
		t.Fatalf(
			"审核教师身份异常: %+v",
			detail.Reviewer,
		)
	}

	if detail.Review.ReviewLevel != models.ReviewLevelL1 ||
		detail.Review.ReviewRound != 2 ||
		detail.Review.Decision != models.ReviewDecisionRevision ||
		detail.Review.Comment != r03IntegrationReviewComment {
		t.Fatalf(
			"正式审核决定异常: %+v",
			detail.Review,
		)
	}

	if detail.Review.Score == nil ||
		*detail.Review.Score != 82.5 {
		t.Fatalf(
			"正式审核评分异常: %+v",
			detail.Review.Score,
		)
	}

	if detail.Review.ReviewedAt == nil {
		t.Fatal("正式审核时间缺失")
	}

	config := detail.ReviewConfig
	if !config.Available {
		t.Fatalf(
			"R-02历史配置应可用，reason=%q",
			config.UnavailableReason,
		)
	}

	expectedDimensions := []string{
		models.CWAIReviewDimensionTeachingLogic,
		models.CWAIReviewDimensionPageReadability,
		models.CWAIReviewDimensionCustom,
	}

	if config.SchemaVersion != models.CWAIReviewConfigSchemaVersion ||
		!slices.Equal(config.Dimensions, expectedDimensions) ||
		config.CustomFocus != r03IntegrationCustomFocus ||
		config.LessonReferenceMode != models.CWAIReviewLessonReferenceNoLesson {
		t.Fatalf(
			"R-02历史配置异常: %+v",
			config,
		)
	}

	if config.LessonMaterialsUsed == nil {
		t.Fatal("no_lesson应能证明本次没有实际使用教案类材料")
	}
	if *config.LessonMaterialsUsed {
		t.Fatal("no_lesson历史记录不能报告实际使用了教案类材料")
	}

	if !detail.IssuesAvailable {
		t.Fatalf(
			"新审核正式交付问题应可用，reason=%q",
			detail.IssuesUnavailableReason,
		)
	}

	if len(detail.Issues) != 1 {
		t.Fatalf(
			"正式交付问题数量异常: want=1 got=%d",
			len(detail.Issues),
		)
	}

	issue := detail.Issues[0]

	if issue.ID != r03IntegrationItemID ||
		issue.PageID == nil ||
		*issue.PageID != r03IntegrationPage1ID ||
		issue.PageNumber != 1 ||
		issue.PageTitle != "一次函数导入" ||
		issue.Severity != models.CWReviewSeverityHigh ||
		issue.Dimension != models.CWAIReviewDimensionTeachingLogic {
		t.Fatalf(
			"正式交付问题稳定身份异常: %+v",
			issue,
		)
	}

	teacherView := issue.TeacherView

	if teacherView.TeacherTitle != r03IntegrationTeacherTitle ||
		teacherView.WhatHappened != r03IntegrationWhatHappened ||
		teacherView.TeachingImpact != r03IntegrationTeachingImpact ||
		teacherView.ImprovementGoal != r03IntegrationImprovementGoal ||
		!teacherView.ManualCheckRequired ||
		teacherView.TeacherContext != "" ||
		!slices.Equal(
			teacherView.AcceptanceChecks,
			r03IntegrationAcceptanceChecks,
		) {
		t.Fatalf(
			"教师视图快照异常: %+v",
			teacherView,
		)
	}

	if !issue.DeliveredInstructionAvailable ||
		issue.DeliveredInstruction == nil {
		t.Fatalf(
			"V1正式交付版本应可用，reason=%q",
			issue.DeliveredInstructionUnavailableReason,
		)
	}

	delivered := issue.DeliveredInstruction

	if delivered.VersionID != r03IntegrationV1ID ||
		delivered.VersionNo != 1 ||
		delivered.Content != r03IntegrationV1Content ||
		delivered.SourceType !=
			models.CWReviewInstructionVersionSourceManual ||
		delivered.ConfirmedAt == nil {
		t.Fatalf(
			"正式交付版本不是审核时V1: %+v",
			delivered,
		)
	}

	if len(issue.PreviousModificationRecords) != 0 {
		t.Fatalf(
			"时间点A不应存在作者后续修改记录: %+v",
			issue.PreviousModificationRecords,
		)
	}

	if !detail.HistoricalPagesAvailable {
		t.Fatalf(
			"新审核历史页面应可用，reason=%q",
			detail.HistoricalPagesUnavailableReason,
		)
	}

	if len(detail.HistoricalPages) != 2 {
		t.Fatalf(
			"历史页面数量异常: want=2 got=%d",
			len(detail.HistoricalPages),
		)
	}

	if len(detail.CurrentPages) != 2 {
		t.Fatalf(
			"当前页面数量异常: want=2 got=%d",
			len(detail.CurrentPages),
		)
	}

	historicalByID := make(
		map[string]models.CoursewareReviewHistoryPage,
		len(detail.HistoricalPages),
	)
	for _, page := range detail.HistoricalPages {
		historicalByID[page.PageID] = page
	}

	currentByID := make(
		map[string]models.CoursewareReviewHistoryCurrentPage,
		len(detail.CurrentPages),
	)
	for _, page := range detail.CurrentPages {
		currentByID[page.PageID] = page
	}

	page1History, ok := historicalByID[r03IntegrationPage1ID]
	if !ok {
		t.Fatal("审核时页面缺少P1稳定page_id")
	}

	if page1History.PageNumber != 1 ||
		page1History.PageTitle != "一次函数导入" ||
		page1History.HTMLContent != r03IntegrationPage1HTML ||
		!page1History.CurrentExists {
		t.Fatalf(
			"审核时P1事实异常: %+v",
			page1History,
		)
	}

	page2History, ok := historicalByID[r03IntegrationPage2ID]
	if !ok {
		t.Fatal("审核时页面缺少P2稳定page_id")
	}

	if page2History.PageNumber != 2 ||
		page2History.PageTitle != "函数图像观察" ||
		page2History.HTMLContent != r03IntegrationPage2HTML ||
		!page2History.CurrentExists {
		t.Fatalf(
			"审核时P2事实异常: %+v",
			page2History,
		)
	}

	page1Current, ok := currentByID[r03IntegrationPage1ID]
	if !ok || page1Current.HTMLContent != r03IntegrationPage1HTML {
		t.Fatalf(
			"时间点A当前P1与审核时P1不一致: %+v",
			page1Current,
		)
	}

	page2Current, ok := currentByID[r03IntegrationPage2ID]
	if !ok || page2Current.HTMLContent != r03IntegrationPage2HTML {
		t.Fatalf(
			"时间点A当前P2与审核时P2不一致: %+v",
			page2Current,
		)
	}
}

func assertR03LegacyHistoryDetail(
	t *testing.T,
	detail *models.CoursewareReviewHistoryDetail,
) {
	t.Helper()

	if detail == nil {
		t.Fatal("legacy审核历史详情为空")
	}

	if detail.ReviewID != r03IntegrationLegacyReviewID {
		t.Fatalf(
			"legacy review_id异常: %s",
			detail.ReviewID,
		)
	}

	if detail.Review.Comment !=
		"LEGACY-R03：请进一步明确课堂问题链。" {
		t.Fatalf(
			"legacy审核意见被改写: %q",
			detail.Review.Comment,
		)
	}

	if detail.ReviewConfig.Available {
		t.Fatal("无feedback的legacy审核不能伪造R-02不可变配置")
	}

	if detail.ReviewConfig.UnavailableReason !=
		models.CWReviewHistoryConfigUnavailableLegacy {
		t.Fatalf(
			"legacy配置不可用原因异常: %q",
			detail.ReviewConfig.UnavailableReason,
		)
	}

	if detail.IssuesAvailable {
		t.Fatal("无feedback的legacy审核不能用空列表冒充当时没有交付问题")
	}

	if detail.IssuesUnavailableReason !=
		models.CWReviewHistoryIssuesUnavailableLegacy {
		t.Fatalf(
			"legacy问题不可用原因异常: %q",
			detail.IssuesUnavailableReason,
		)
	}

	if len(detail.Issues) != 0 {
		t.Fatalf(
			"legacy审核不应产生伪造交付问题: %+v",
			detail.Issues,
		)
	}

	if detail.HistoricalPagesAvailable {
		t.Fatal("legacy审核不能把当前页面冒充审核时页面")
	}

	if detail.HistoricalPagesUnavailableReason !=
		models.CWReviewHistoryPagesUnavailableLegacy {
		t.Fatalf(
			"legacy历史页面不可用原因异常: %q",
			detail.HistoricalPagesUnavailableReason,
		)
	}

	if len(detail.HistoricalPages) != 0 {
		t.Fatalf(
			"legacy历史页面必须为空: %+v",
			detail.HistoricalPages,
		)
	}

	if len(detail.CurrentPages) != 2 {
		t.Fatalf(
			"legacy记录仍应显式提供当前页面Tab，want=2 got=%d",
			len(detail.CurrentPages),
		)
	}
}

// TestR03ReviewHistoryIntegrationBaseline 验证“时间点A”：
//
//  1. 新审核记录真实返回审核决定、R-02不可变配置、正式交付问题、V1和历史HTML；
//  2. legacy记录明确报告不可恢复，不以当前数据冒充历史事实；
//  3. 作者、本校senior、admin有权查看；
//  4. 同校同域但无审核角色的viewer仍统一按NotFound拒绝。
//
// 本测试全程只读数据库。
func TestR03ReviewHistoryIntegrationBaseline(t *testing.T) {
	ctx := openR03IntegrationDatabase(t)

	service := NewCoursewareReviewService()

	allowedActors := []struct {
		name   string
		userID string
		role   string
		domain string
	}{
		{
			name:   "author",
			userID: r03IntegrationAuthorID,
			role:   models.RoleOperator,
			domain: models.EducationDomainK12,
		},
		{
			name:   "review_school_senior",
			userID: r03IntegrationSeniorID,
			role:   models.RoleSeniorOperator,
			domain: models.EducationDomainK12,
		},
		{
			name:   "admin",
			userID: r03IntegrationAdminID,
			role:   models.RoleAdmin,
			domain: models.EducationDomainMixed,
		},
	}

	for _, tc := range allowedActors {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			actor := buildR03IntegrationActor(
				t,
				ctx,
				tc.userID,
				tc.role,
			)

			if actor.EducationDomain != tc.domain {
				t.Fatalf(
					"Actor教育域异常: want=%q got=%q",
					tc.domain,
					actor.EducationDomain,
				)
			}

			newDetail, err := service.GetReviewHistoryDetail(
				ctx,
				r03IntegrationReviewID,
				actor,
			)
			if err != nil {
				t.Fatalf(
					"读取R-03新审核历史失败: %v",
					err,
				)
			}

			assertR03BaselineHistoryDetail(
				t,
				newDetail,
			)

			legacyDetail, err := service.GetReviewHistoryDetail(
				ctx,
				r03IntegrationLegacyReviewID,
				actor,
			)
			if err != nil {
				t.Fatalf(
					"读取legacy审核历史失败: %v",
					err,
				)
			}

			assertR03LegacyHistoryDetail(
				t,
				legacyDetail,
			)
		})
	}

	t.Run("same_school_viewer_denied_without_probing", func(t *testing.T) {
		actor := buildR03IntegrationActor(
			t,
			ctx,
			r03IntegrationViewerID,
			models.RoleViewer,
		)

		assertR03TeachingActorDomain(t, actor)

		for _, reviewID := range []string{
			r03IntegrationLegacyReviewID,
			r03IntegrationReviewID,
		} {
			detail, err := service.GetReviewHistoryDetail(
				ctx,
				reviewID,
				actor,
			)

			if detail != nil {
				t.Fatalf(
					"越权viewer不应取得审核历史详情: review=%s detail=%+v",
					reviewID,
					detail,
				)
			}

			if !errors.Is(
				err,
				ErrCWReviewHistoryDetailNotFound,
			) {
				t.Fatalf(
					"越权viewer必须统一NotFound防探测: review=%s err=%v",
					reviewID,
					err,
				)
			}
		}
	})
}
