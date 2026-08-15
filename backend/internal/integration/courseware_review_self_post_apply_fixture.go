package integration

// courseware_review_self_post_apply_fixture.go
//
// R-01.1 作者自审 applied 后三项人工决策的专用集成测试夹具。
//
// 本文件只负责建立真实 tedna_test 前置事实：
//
//   1. 创建课件与稳定页面；
//   2. 创建 review_level=0 的自审 AI 会话外键事实；
//   3. 通过正式 repository 创建 self 整改项；
//   4. 通过正式不可变版本仓储确认修改方案；
//   5. 提供页面修改、删除、状态和审计事件读取辅助。
//
// 不直接伪造 current/applied 指令版本引用。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const (
	cwReviewSelfPostApplyCoursewareID = "21000000-0000-4000-8000-000000000001"

	cwReviewSelfPostApplyPageID = "21000000-0000-4000-8000-000000000002"

	cwReviewSelfPostApplySessionID = "21000000-0000-4000-8000-000000000003"

	cwReviewSelfPostApplyInstruction = "把本页练习反馈改得更明确，并让学生完成操作后再看到结论。"

	cwReviewSelfPostApplyOriginalHTML = `<div class="cw-page"><button id="try">开始练习</button></div>`
)

type cwReviewSelfPostApplyFixture struct {
	CoursewareID string
	PageID       string
	SessionID    string

	ItemID    string
	VersionID string

	Instruction      string
	OriginalPageHTML string
	OriginalPageHash string
}

type cwReviewSelfPostApplyItemState struct {
	Status string

	PageID             string
	PageNumberSnapshot int

	CurrentInstructionVersionID string
	AppliedInstructionVersionID string

	AppliedPageHash string
	AppliedAt       *time.Time

	ResolvedBy string
}

func cwReviewSelfPostApplyHash(
	content string,
) string {
	sum := sha256.Sum256(
		[]byte(content),
	)

	return hex.EncodeToString(
		sum[:],
	)
}

func setupCWReviewSelfPostApplyFixture(
	t *testing.T,
) *cwReviewSelfPostApplyFixture {
	t.Helper()

	cfg := testConfig()
	initTestDB(t, cfg)
	CleanAndSeed(t)

	ctx := context.Background()

	_, err := database.DB.Exec(
		ctx,
		`
		INSERT INTO coursewares (
			id,
			lesson_plan_id,
			user_id,
			title,
			subject,
			grade,
			status,
			page_count,
			source_type,
			publish_state,
			review_level,
			code_share_scope,
			collab_state,
			education_domain,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			NULL,
			$2,
			'R-01.1 自审状态链测试课件',
			'数学',
			'七年级',
			'preview',
			1,
			'topic_direct',
			'private',
			0,
			'none',
			'idle',
			'k12',
			NOW(),
			NOW()
		)`,
		cwReviewSelfPostApplyCoursewareID,
		SeedOperatorID,
	)
	if err != nil {
		t.Fatalf(
			"创建R-01.1测试课件失败: %v",
			err,
		)
	}

	_, err = database.DB.Exec(
		ctx,
		`
		INSERT INTO courseware_pages (
			id,
			courseware_id,
			page_number,
			title,
			purpose,
			content_summary,
			interaction_type,
			visual_format,
			media_requirements,
			estimated_complexity,
			html_content,
			status,
			page_index,
			idx_cognitive_level,
			idx_interaction_level,
			idx_visual_format,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			$2,
			1,
			'课堂练习',
			'检查学生是否理解当前知识点',
			'学生完成操作后查看反馈',
			'button',
			'interactive_diagram',
			'',
			2,
			$3,
			'generated',
			'',
			2,
			2,
			'interactive_diagram',
			NOW(),
			NOW()
		)`,
		cwReviewSelfPostApplyPageID,
		cwReviewSelfPostApplyCoursewareID,
		cwReviewSelfPostApplyOriginalHTML,
	)
	if err != nil {
		t.Fatalf(
			"创建R-01.1测试课件页面失败: %v",
			err,
		)
	}

	// 当前专项状态链只依赖会话作为真实外键和审计归属。
	//
	// 会话本身不经过AI执行，因此直接写最小可信会话事实，
	// 避免把模型调用或R-02配置准备流程引入当前状态测试。
	_, err = database.DB.Exec(
		ctx,
		`
		INSERT INTO courseware_ai_review_sessions (
			id,
			courseware_id,
			reviewer_id,
			assistant_id,
			lesson_plan_id,
			review_level,
			education_domain,
			subject,
			grade,
			status,
			current_stage,
			created_at,
			updated_at,
			completed_at
		)
		VALUES (
			$1,
			$2,
			$3,
			NULL,
			NULL,
			0,
			'k12',
			'数学',
			'七年级',
			'done',
			'done',
			NOW(),
			NOW(),
			NOW()
		)`,
		cwReviewSelfPostApplySessionID,
		cwReviewSelfPostApplyCoursewareID,
		SeedOperatorID,
	)
	if err != nil {
		t.Fatalf(
			"创建R-01.1自审会话事实失败: %v",
			err,
		)
	}

	pageID := cwReviewSelfPostApplyPageID

	item := &models.CoursewareReviewItem{
		CoursewareID: cwReviewSelfPostApplyCoursewareID,

		SourceSessionID: cwReviewSelfPostApplySessionID,

		SourceFindingID: "r011_self_post_apply_fixture",

		OriginType: models.CWReviewItemOriginAIFinding,

		SourceType: models.CWReviewItemSourceSelf,

		ReviewLevel: 0,
		ReviewRound: 0,

		CreatedBy: SeedOperatorID,

		OwnerID: SeedOperatorID,

		PageID: &pageID,

		PageNumberSnapshot: 1,
		PageTitleSnapshot:  "课堂练习",

		PageHTMLHash: cwReviewSelfPostApplyHash(
			cwReviewSelfPostApplyOriginalHTML,
		),

		Severity: models.CWReviewSeverityMedium,

		Dimension: "teaching_logic",

		Title: "练习反馈需要更明确",

		Description: "当前练习完成后的反馈不足以帮助学生判断自己的理解情况。",

		EvidenceJSON: `{"teacher_snapshot":{"teacher_title":"练习反馈需要更明确"}}`,

		OriginalSuggestion: cwReviewSelfPostApplyInstruction,

		Status: models.CWReviewItemStatusDetected,
	}

	if err := repository.CreateCoursewareReviewItem(
		ctx,
		item,
	); err != nil {
		t.Fatalf(
			"通过正式仓储创建自审整改项失败: %v",
			err,
		)
	}

	if item.ID == "" {
		t.Fatal(
			"创建自审整改项后ID为空",
		)
	}

	version, err :=
		repository.ConfirmCoursewareReviewInstructionVersion(
			ctx,
			&repository.ConfirmCoursewareReviewInstructionVersionInput{
				ItemID: item.ID,

				ActorID: SeedOperatorID,

				Instruction: cwReviewSelfPostApplyInstruction,

				ExpectedCurrentVersionID: "",

				SourceType: "manual",
			},
		)
	if err != nil {
		t.Fatalf(
			"通过正式版本仓储确认自审修改方案失败: %v",
			err,
		)
	}

	if version == nil ||
		version.ID == "" {
		t.Fatal(
			"确认自审修改方案后版本为空",
		)
	}

	return &cwReviewSelfPostApplyFixture{
		CoursewareID: cwReviewSelfPostApplyCoursewareID,

		PageID: cwReviewSelfPostApplyPageID,

		SessionID: cwReviewSelfPostApplySessionID,

		ItemID: item.ID,

		VersionID: version.ID,

		Instruction: cwReviewSelfPostApplyInstruction,

		OriginalPageHTML: cwReviewSelfPostApplyOriginalHTML,

		OriginalPageHash: cwReviewSelfPostApplyHash(
			cwReviewSelfPostApplyOriginalHTML,
		),
	}
}

func cwReviewSelfPostApplyBeginInput(
	fixture *cwReviewSelfPostApplyFixture,
) *repository.BeginCoursewareReviewItemApplicationInput {
	if fixture == nil {
		return nil
	}

	return &repository.BeginCoursewareReviewItemApplicationInput{
		ItemID: fixture.ItemID,

		ActorID: SeedOperatorID,

		CoursewareID: fixture.CoursewareID,

		PageNumber: 1,

		InstructionVersionID: fixture.VersionID,

		SubmittedInstruction: fixture.Instruction,
	}
}

func beginCWReviewSelfPostApply(
	t *testing.T,
	fixture *cwReviewSelfPostApplyFixture,
) *repository.BeginCoursewareReviewItemApplicationResult {
	t.Helper()

	result, err :=
		repository.BeginCoursewareReviewItemApplicationWithVersion(
			context.Background(),
			cwReviewSelfPostApplyBeginInput(
				fixture,
			),
		)
	if err != nil {
		t.Fatalf(
			"开始R-01.1测试页面应用失败: %v",
			err,
		)
	}

	if result == nil {
		t.Fatal(
			"开始页面应用成功但守卫为空",
		)
	}

	return result
}

func updateCWReviewSelfPostApplyPageHTML(
	t *testing.T,
	fixture *cwReviewSelfPostApplyFixture,
	html string,
) {
	t.Helper()

	result, err := database.DB.Exec(
		context.Background(),
		`
		UPDATE courseware_pages
		SET
			html_content = $3,
			updated_at = NOW()
		WHERE id = $1
		  AND courseware_id = $2`,
		fixture.PageID,
		fixture.CoursewareID,
		html,
	)
	if err != nil {
		t.Fatalf(
			"更新R-01.1测试页面失败: %v",
			err,
		)
	}

	if result.RowsAffected() != 1 {
		t.Fatalf(
			"更新R-01.1测试页面影响行数错误: %d",
			result.RowsAffected(),
		)
	}
}

func deleteCWReviewSelfPostApplyPage(
	t *testing.T,
	fixture *cwReviewSelfPostApplyFixture,
) {
	t.Helper()

	result, err := database.DB.Exec(
		context.Background(),
		`
		DELETE FROM courseware_pages
		WHERE id = $1
		  AND courseware_id = $2`,
		fixture.PageID,
		fixture.CoursewareID,
	)
	if err != nil {
		t.Fatalf(
			"删除R-01.1测试页面失败: %v",
			err,
		)
	}

	if result.RowsAffected() != 1 {
		t.Fatalf(
			"删除R-01.1测试页面影响行数错误: %d",
			result.RowsAffected(),
		)
	}
}

func applyCWReviewSelfPostApplyFirstChange(
	t *testing.T,
	fixture *cwReviewSelfPostApplyFixture,
	appliedHTML string,
) *cwReviewSelfPostApplyItemState {
	t.Helper()

	guard := beginCWReviewSelfPostApply(
		t,
		fixture,
	)

	if guard.PageID != fixture.PageID {
		t.Fatalf(
			"首次页面应用守卫page_id错误: got=%s want=%s",
			guard.PageID,
			fixture.PageID,
		)
	}

	if appliedHTML != fixture.OriginalPageHTML {
		updateCWReviewSelfPostApplyPageHTML(
			t,
			fixture,
			appliedHTML,
		)
	}

	appliedHash :=
		cwReviewSelfPostApplyHash(
			appliedHTML,
		)

	if err := repository.MarkCoursewareReviewItemApplied(
		context.Background(),
		fixture.ItemID,
		SeedOperatorID,
		appliedHash,
	); err != nil {
		t.Fatalf(
			"记录R-01.1测试页面修改完成失败: %v",
			err,
		)
	}

	state := loadCWReviewSelfPostApplyItemState(
		t,
		fixture.ItemID,
	)

	if state.Status !=
		models.CWReviewItemStatusApplied {
		t.Fatalf(
			"首次页面修改完成后状态错误: got=%s want=applied",
			state.Status,
		)
	}

	if state.AppliedAt == nil {
		t.Fatal(
			"首次页面修改完成后缺少applied_at",
		)
	}

	if state.AppliedPageHash != appliedHash {
		t.Fatalf(
			"首次页面修改完成哈希错误: got=%s want=%s",
			state.AppliedPageHash,
			appliedHash,
		)
	}

	return state
}

func loadCWReviewSelfPostApplyItemState(
	t *testing.T,
	itemID string,
) *cwReviewSelfPostApplyItemState {
	t.Helper()

	state :=
		&cwReviewSelfPostApplyItemState{}

	err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT
			status,
			COALESCE(page_id::text, ''),
			page_number_snapshot,
			COALESCE(current_instruction_version_id::text, ''),
			COALESCE(applied_instruction_version_id::text, ''),
			COALESCE(applied_page_hash, ''),
			applied_at,
			COALESCE(resolved_by::text, '')
		FROM courseware_review_items
		WHERE id = $1`,
		itemID,
	).Scan(
		&state.Status,
		&state.PageID,
		&state.PageNumberSnapshot,
		&state.CurrentInstructionVersionID,
		&state.AppliedInstructionVersionID,
		&state.AppliedPageHash,
		&state.AppliedAt,
		&state.ResolvedBy,
	)
	if err != nil {
		t.Fatalf(
			"读取R-01.1整改项状态失败: %v",
			err,
		)
	}

	return state
}

func loadCWReviewSelfPostApplyVersionStatus(
	t *testing.T,
	versionID string,
) string {
	t.Helper()

	var status string

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT status
		FROM courseware_review_instruction_versions
		WHERE id = $1`,
		versionID,
	).Scan(
		&status,
	); err != nil {
		t.Fatalf(
			"读取R-01.1指令版本状态失败: %v",
			err,
		)
	}

	return status
}

func countCWReviewSelfPostApplyEvent(
	t *testing.T,
	itemID string,
	event string,
) int {
	t.Helper()

	var count int

	if err := database.DB.QueryRow(
		context.Background(),
		`
		SELECT COUNT(*)::integer
		FROM courseware_ai_review_messages
		WHERE review_item_id = $1
		  AND role = 'system'
		  AND citations_json ->> 'event' = $2`,
		itemID,
		event,
	).Scan(
		&count,
	); err != nil {
		t.Fatalf(
			"统计R-01.1状态事件失败(event=%s): %v",
			event,
			err,
		)
	}

	return count
}
