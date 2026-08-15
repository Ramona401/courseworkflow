package integration

// courseware_review_self_post_apply_state_test.go
//
// R-01.1 自审 applied 后人工三项决策的稳定状态链测试：
//
//   - 首次修改失败：applying -> confirmed；
//   - 暂时不处理：applied -> dismissed，完整保留applied事实；
//   - 恢复：dismissed -> applied，不伪造新的修改完成事实；
//   - 继续调整：直接以applied_page_hash作为新起点；
//   - 再次调整失败：恢复上一轮applied_at与applied_page_hash；
//   - applied从不自动等于resolved。

import (
	"context"
	"testing"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestCWReviewSelfInitialApplicationAbortReturnsConfirmed(
	t *testing.T,
) {
	fixture :=
		setupCWReviewSelfPostApplyFixture(
			t,
		)

	beginCWReviewSelfPostApply(
		t,
		fixture,
	)

	applying :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if applying.Status !=
		models.CWReviewItemStatusApplying {
		t.Fatalf(
			"首次应用开始后状态错误: got=%s want=applying",
			applying.Status,
		)
	}

	if applying.AppliedInstructionVersionID !=
		fixture.VersionID {
		t.Fatalf(
			"首次应用没有绑定确认版本: got=%s want=%s",
			applying.AppliedInstructionVersionID,
			fixture.VersionID,
		)
	}

	if applying.AppliedAt != nil {
		t.Fatal(
			"首次应用尚未完成时不应存在applied_at",
		)
	}

	if applying.AppliedPageHash != "" {
		t.Fatalf(
			"首次应用尚未完成时不应存在applied_page_hash: %s",
			applying.AppliedPageHash,
		)
	}

	if err :=
		repository.AbortCoursewareReviewItemInitialApplication(
			context.Background(),
			fixture.ItemID,
			SeedOperatorID,
		); err != nil {
		t.Fatalf(
			"首次页面微调失败回退失败: %v",
			err,
		)
	}

	confirmed :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if confirmed.Status !=
		models.CWReviewItemStatusConfirmed {
		t.Fatalf(
			"首次应用失败后状态错误: got=%s want=confirmed",
			confirmed.Status,
		)
	}

	if confirmed.AppliedInstructionVersionID != "" {
		t.Fatalf(
			"首次应用失败后临时应用版本未清理: %s",
			confirmed.AppliedInstructionVersionID,
		)
	}

	if confirmed.AppliedAt != nil ||
		confirmed.AppliedPageHash != "" {
		t.Fatal(
			"首次应用失败后不应伪造任何applied事实",
		)
	}

	if confirmed.CurrentInstructionVersionID !=
		fixture.VersionID {
		t.Fatalf(
			"首次应用失败不应破坏当前确认版本: got=%s want=%s",
			confirmed.CurrentInstructionVersionID,
			fixture.VersionID,
		)
	}
}

func TestCWReviewSelfPostApplyDismissRestorePreservesEvidence(
	t *testing.T,
) {
	fixture :=
		setupCWReviewSelfPostApplyFixture(
			t,
		)

	appliedHTML :=
		`<div class="cw-page"><p>第一次修改完成：操作后显示分层反馈。</p></div>`

	beforeDismiss :=
		applyCWReviewSelfPostApplyFirstChange(
			t,
			fixture,
			appliedHTML,
		)

	if beforeDismiss.Status ==
		models.CWReviewItemStatusResolved {
		t.Fatal(
			"页面修改完成不能自动等于问题已解决",
		)
	}

	if err :=
		repository.DismissAppliedSelfCoursewareReviewItem(
			context.Background(),
			fixture.ItemID,
			SeedOperatorID,
			"这节课先不继续处理，下一轮再检查。",
		); err != nil {
		t.Fatalf(
			"暂时不处理applied自审问题失败: %v",
			err,
		)
	}

	dismissed :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if dismissed.Status !=
		models.CWReviewItemStatusDismissed {
		t.Fatalf(
			"暂时不处理后状态错误: got=%s want=dismissed",
			dismissed.Status,
		)
	}

	assertCWReviewSelfPostApplyEvidenceEqual(
		t,
		beforeDismiss,
		dismissed,
		"暂时不处理",
	)

	if count :=
		countCWReviewSelfPostApplyEvent(
			t,
			fixture.ItemID,
			"dismissed",
		); count != 1 {
		t.Fatalf(
			"暂时不处理事件数量错误: got=%d want=1",
			count,
		)
	}

	if err :=
		repository.RestoreDismissedAppliedSelfCoursewareReviewItem(
			context.Background(),
			fixture.ItemID,
			SeedOperatorID,
		); err != nil {
		t.Fatalf(
			"恢复暂时不处理的自审问题失败: %v",
			err,
		)
	}

	restored :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if restored.Status !=
		models.CWReviewItemStatusApplied {
		t.Fatalf(
			"恢复后状态错误: got=%s want=applied",
			restored.Status,
		)
	}

	assertCWReviewSelfPostApplyEvidenceEqual(
		t,
		beforeDismiss,
		restored,
		"恢复",
	)

	if restored.ResolvedBy != "" {
		t.Fatalf(
			"恢复applied不能伪造解决人: %s",
			restored.ResolvedBy,
		)
	}

	if count :=
		countCWReviewSelfPostApplyEvent(
			t,
			fixture.ItemID,
			"restored",
		); count != 1 {
		t.Fatalf(
			"恢复事件数量错误: got=%d want=1",
			count,
		)
	}
}

func TestCWReviewSelfReapplyUsesAppliedHashAndAbortRestoresEvidence(
	t *testing.T,
) {
	fixture :=
		setupCWReviewSelfPostApplyFixture(
			t,
		)

	// 第一次真实修改后的HTML故意与原始审核快照不同。
	//
	// 如果再次调整仍错误比较原始item/version快照，
	// 这一用例会在Begin阶段失败。
	firstAppliedHTML :=
		`<div class="cw-page"><p>第一次修改后的新页面内容。</p></div>`

	previousApplied :=
		applyCWReviewSelfPostApplyFirstChange(
			t,
			fixture,
			firstAppliedHTML,
		)

	if previousApplied.AppliedPageHash ==
		fixture.OriginalPageHash {
		t.Fatal(
			"测试前置失败：第一次修改后的哈希必须不同于原始快照",
		)
	}

	guard, err :=
		repository.BeginCoursewareReviewItemApplicationWithVersion(
			context.Background(),
			cwReviewSelfPostApplyBeginInput(
				fixture,
			),
		)
	if err != nil {
		t.Fatalf(
			"applied后继续调整应以applied_page_hash为起点，实际失败: %v",
			err,
		)
	}

	if guard == nil {
		t.Fatal(
			"继续调整成功但页面守卫为空",
		)
	}

	if guard.PageHTMLHash !=
		previousApplied.AppliedPageHash {
		t.Fatalf(
			"继续调整守卫没有使用当前applied页面: got=%s want=%s",
			guard.PageHTMLHash,
			previousApplied.AppliedPageHash,
		)
	}

	applying :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if applying.Status !=
		models.CWReviewItemStatusApplying {
		t.Fatalf(
			"继续调整开始后状态错误: got=%s want=applying",
			applying.Status,
		)
	}

	if applying.AppliedAt != nil {
		t.Fatal(
			"继续调整执行中应暂时清空applied_at",
		)
	}

	if applying.AppliedPageHash !=
		previousApplied.AppliedPageHash {
		t.Fatal(
			"继续调整执行中必须保留上一轮applied_page_hash作为恢复标记",
		)
	}

	if applying.AppliedInstructionVersionID !=
		previousApplied.AppliedInstructionVersionID {
		t.Fatal(
			"继续调整不得替换已确认的应用版本",
		)
	}

	if count :=
		countCWReviewSelfPostApplyEvent(
			t,
			fixture.ItemID,
			"self_reapply_started",
		); count != 1 {
		t.Fatalf(
			"继续调整开始事件数量错误: got=%d want=1",
			count,
		)
	}

	if err :=
		repository.RestoreSelfCoursewareReviewItemAfterReapplyAbort(
			context.Background(),
			fixture.ItemID,
			SeedOperatorID,
		); err != nil {
		t.Fatalf(
			"继续调整失败后恢复上一轮applied事实失败: %v",
			err,
		)
	}

	restored :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if restored.Status !=
		models.CWReviewItemStatusApplied {
		t.Fatalf(
			"继续调整失败恢复后状态错误: got=%s want=applied",
			restored.Status,
		)
	}

	assertCWReviewSelfPostApplyEvidenceEqual(
		t,
		previousApplied,
		restored,
		"继续调整失败恢复",
	)

	if count :=
		countCWReviewSelfPostApplyEvent(
			t,
			fixture.ItemID,
			"reapply_aborted",
		); count != 1 {
		t.Fatalf(
			"继续调整失败恢复事件数量错误: got=%d want=1",
			count,
		)
	}
}

func assertCWReviewSelfPostApplyEvidenceEqual(
	t *testing.T,
	expected *cwReviewSelfPostApplyItemState,
	actual *cwReviewSelfPostApplyItemState,
	action string,
) {
	t.Helper()

	if expected == nil ||
		actual == nil {
		t.Fatalf(
			"%s证据比较收到空状态",
			action,
		)
	}

	if actual.AppliedInstructionVersionID !=
		expected.AppliedInstructionVersionID {
		t.Fatalf(
			"%s改写了应用版本: got=%s want=%s",
			action,
			actual.AppliedInstructionVersionID,
			expected.AppliedInstructionVersionID,
		)
	}

	if actual.AppliedPageHash !=
		expected.AppliedPageHash {
		t.Fatalf(
			"%s改写了applied_page_hash: got=%s want=%s",
			action,
			actual.AppliedPageHash,
			expected.AppliedPageHash,
		)
	}

	if expected.AppliedAt == nil ||
		actual.AppliedAt == nil {
		t.Fatalf(
			"%s丢失applied_at",
			action,
		)
	}

	if !actual.AppliedAt.Equal(
		*expected.AppliedAt,
	) {
		t.Fatalf(
			"%s改写了applied_at: got=%s want=%s",
			action,
			actual.AppliedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			expected.AppliedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		)
	}
}
