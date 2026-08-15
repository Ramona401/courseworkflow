package integration

// courseware_review_self_post_apply_page_test.go
//
// R-01.1 自审 applied 后页面新鲜度专项集成测试：
//
//   - applied后页面再变化：继续调整必须进入stale；
//   - dismissed期间原页删除：恢复必须进入orphaned；
//   - applied期间原页删除：确认解决必须进入orphaned；
//   - 首次修改前原页已经删除：不能开始应用，必须进入orphaned；
//   - ON DELETE SET NULL不得把历史页级问题误解释成整课问题。

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/models"
	"tedna/internal/repository"
)

func TestCWReviewSelfReapplyChangedPageBecomesStale(
	t *testing.T,
) {
	fixture :=
		setupCWReviewSelfPostApplyFixture(
			t,
		)

	firstAppliedHTML :=
		`<div class="cw-page"><p>第一次修改完成。</p></div>`

	applied :=
		applyCWReviewSelfPostApplyFirstChange(
			t,
			fixture,
			firstAppliedHTML,
		)

	updateCWReviewSelfPostApplyPageHTML(
		t,
		fixture,
		`<div class="cw-page"><p>之后又发生了额外页面变化。</p></div>`,
	)

	_, err :=
		repository.BeginCoursewareReviewItemApplicationWithVersion(
			context.Background(),
			cwReviewSelfPostApplyBeginInput(
				fixture,
			),
		)

	if !errors.Is(
		err,
		repository.ErrCoursewareReviewItemApplicationPageStale,
	) {
		t.Fatalf(
			"页面变化后继续调整应返回PageStale，实际: %v",
			err,
		)
	}

	stale :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if stale.Status !=
		models.CWReviewItemStatusStale {
		t.Fatalf(
			"页面变化后状态错误: got=%s want=stale",
			stale.Status,
		)
	}

	if stale.AppliedAt == nil ||
		stale.AppliedPageHash !=
			applied.AppliedPageHash ||
		stale.AppliedInstructionVersionID !=
			applied.AppliedInstructionVersionID {
		t.Fatal(
			"进入stale时必须保留上一轮完整applied证据",
		)
	}

	if versionStatus :=
		loadCWReviewSelfPostApplyVersionStatus(
			t,
			fixture.VersionID,
		); versionStatus !=
		models.CWReviewInstructionVersionStatusInvalidForPage {
		t.Fatalf(
			"页面变化后当前版本应失效: got=%s want=invalid_for_page",
			versionStatus,
		)
	}

	if count :=
		countCWReviewSelfPostApplyEvent(
			t,
			fixture.ItemID,
			"stale",
		); count != 1 {
		t.Fatalf(
			"stale状态事件数量错误: got=%d want=1",
			count,
		)
	}
}

func TestCWReviewSelfPostApplyRestoreDeletedPageBecomesOrphaned(
	t *testing.T,
) {
	fixture :=
		setupCWReviewSelfPostApplyFixture(
			t,
		)

	applied :=
		applyCWReviewSelfPostApplyFirstChange(
			t,
			fixture,
			`<div class="cw-page"><p>已经完成一次修改。</p></div>`,
		)

	if err :=
		repository.DismissAppliedSelfCoursewareReviewItem(
			context.Background(),
			fixture.ItemID,
			SeedOperatorID,
			"稍后再检查。",
		); err != nil {
		t.Fatalf(
			"删除页面测试前暂时不处理失败: %v",
			err,
		)
	}

	deleteCWReviewSelfPostApplyPage(
		t,
		fixture,
	)

	afterDelete :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if afterDelete.PageID != "" {
		t.Fatalf(
			"页面删除后ON DELETE SET NULL未生效: %s",
			afterDelete.PageID,
		)
	}

	if afterDelete.PageNumberSnapshot != 1 {
		t.Fatalf(
			"页面删除后历史页码快照丢失: %d",
			afterDelete.PageNumberSnapshot,
		)
	}

	err :=
		repository.RestoreDismissedAppliedSelfCoursewareReviewItem(
			context.Background(),
			fixture.ItemID,
			SeedOperatorID,
		)

	if !errors.Is(
		err,
		repository.ErrCoursewareReviewItemAppliedPageMissing,
	) {
		t.Fatalf(
			"删除原页面后恢复应返回PageMissing，实际: %v",
			err,
		)
	}

	orphaned :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if orphaned.Status !=
		models.CWReviewItemStatusOrphaned {
		t.Fatalf(
			"删除原页面后恢复状态错误: got=%s want=orphaned",
			orphaned.Status,
		)
	}

	if orphaned.PageID != "" ||
		orphaned.PageNumberSnapshot != 1 {
		t.Fatal(
			"orphaned必须保留历史页码且不得重新伪造page_id",
		)
	}

	if orphaned.AppliedAt == nil ||
		orphaned.AppliedPageHash !=
			applied.AppliedPageHash ||
		orphaned.AppliedInstructionVersionID !=
			applied.AppliedInstructionVersionID {
		t.Fatal(
			"删除页面进入orphaned时必须保留历史applied证据",
		)
	}

	if versionStatus :=
		loadCWReviewSelfPostApplyVersionStatus(
			t,
			fixture.VersionID,
		); versionStatus !=
		models.CWReviewInstructionVersionStatusInvalidForPage {
		t.Fatalf(
			"删除页面后当前版本应失效: got=%s want=invalid_for_page",
			versionStatus,
		)
	}

	if count :=
		countCWReviewSelfPostApplyEvent(
			t,
			fixture.ItemID,
			"orphaned",
		); count != 1 {
		t.Fatalf(
			"orphaned状态事件数量错误: got=%d want=1",
			count,
		)
	}
}

func TestCWReviewSelfResolveDeletedPageBecomesOrphaned(
	t *testing.T,
) {
	fixture :=
		setupCWReviewSelfPostApplyFixture(
			t,
		)

	applied :=
		applyCWReviewSelfPostApplyFirstChange(
			t,
			fixture,
			`<div class="cw-page"><p>等待作者确认修改效果。</p></div>`,
		)

	deleteCWReviewSelfPostApplyPage(
		t,
		fixture,
	)

	err :=
		repository.ResolveSelfCoursewareReviewItem(
			context.Background(),
			fixture.ItemID,
			SeedOperatorID,
			"确认已经解决",
		)

	if !errors.Is(
		err,
		repository.ErrCoursewareReviewItemAppliedPageMissing,
	) {
		t.Fatalf(
			"删除原页面后确认解决应返回PageMissing，实际: %v",
			err,
		)
	}

	orphaned :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if orphaned.Status !=
		models.CWReviewItemStatusOrphaned {
		t.Fatalf(
			"删除原页面后确认解决状态错误: got=%s want=orphaned",
			orphaned.Status,
		)
	}

	if orphaned.ResolvedBy != "" {
		t.Fatalf(
			"原页面已删除时绝不能写入解决确认人: %s",
			orphaned.ResolvedBy,
		)
	}

	if orphaned.AppliedAt == nil ||
		orphaned.AppliedPageHash !=
			applied.AppliedPageHash {
		t.Fatal(
			"确认解决失败转orphaned后应保留历史applied事实",
		)
	}

	if count :=
		countCWReviewSelfPostApplyEvent(
			t,
			fixture.ItemID,
			"orphaned",
		); count != 1 {
		t.Fatalf(
			"确认解决删除页面的orphaned事件数量错误: got=%d want=1",
			count,
		)
	}
}

func TestCWReviewSelfInitialApplicationDeletedPageBecomesOrphaned(
	t *testing.T,
) {
	fixture :=
		setupCWReviewSelfPostApplyFixture(
			t,
		)

	deleteCWReviewSelfPostApplyPage(
		t,
		fixture,
	)

	beforeBegin :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if beforeBegin.Status !=
		models.CWReviewItemStatusConfirmed {
		t.Fatalf(
			"首次应用删除页面测试前状态错误: %s",
			beforeBegin.Status,
		)
	}

	if beforeBegin.PageID != "" ||
		beforeBegin.PageNumberSnapshot != 1 {
		t.Fatal(
			"删除页面后必须表现为page_id为空但历史页码仍为1",
		)
	}

	_, err :=
		repository.BeginCoursewareReviewItemApplicationWithVersion(
			context.Background(),
			cwReviewSelfPostApplyBeginInput(
				fixture,
			),
		)

	if !errors.Is(
		err,
		repository.ErrCoursewareReviewItemApplicationPageOrphaned,
	) {
		t.Fatalf(
			"首次修改前页面已删除应返回PageOrphaned，实际: %v",
			err,
		)
	}

	orphaned :=
		loadCWReviewSelfPostApplyItemState(
			t,
			fixture.ItemID,
		)

	if orphaned.Status !=
		models.CWReviewItemStatusOrphaned {
		t.Fatalf(
			"首次修改前页面已删除状态错误: got=%s want=orphaned",
			orphaned.Status,
		)
	}

	if orphaned.AppliedInstructionVersionID != "" ||
		orphaned.AppliedAt != nil ||
		orphaned.AppliedPageHash != "" {
		t.Fatal(
			"首次修改从未完成时进入orphaned不得伪造applied事实",
		)
	}

	if orphaned.CurrentInstructionVersionID !=
		fixture.VersionID {
		t.Fatal(
			"页面删除不应删除历史确认版本引用",
		)
	}

	if versionStatus :=
		loadCWReviewSelfPostApplyVersionStatus(
			t,
			fixture.VersionID,
		); versionStatus !=
		models.CWReviewInstructionVersionStatusInvalidForPage {
		t.Fatalf(
			"首次修改前页面删除后确认版本应失效: got=%s want=invalid_for_page",
			versionStatus,
		)
	}
}
