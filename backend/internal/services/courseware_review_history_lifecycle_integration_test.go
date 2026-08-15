package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
	"tedna/internal/repository"
)

const r03IntegrationRejectedV2Content = "V2尝试：将导入改为分组讨论后再展示结论，并增加一次即时投票反馈。"

type r03IntegrationItemLifecycleState struct {
	Status string

	CurrentVersionID   string
	DeliveredVersionID string
	AppliedVersionID   string

	AppliedAtExists bool
	AppliedPageHash string

	VersionCount int
	V1Status     string
}

func readR03IntegrationItemLifecycleState(
	t *testing.T,
	ctx context.Context,
) r03IntegrationItemLifecycleState {
	t.Helper()

	var state r03IntegrationItemLifecycleState

	err := database.DB.QueryRow(
		ctx,
		`
		SELECT
			item.status,
			COALESCE(item.current_instruction_version_id::text, ''),
			COALESCE(item.delivered_instruction_version_id::text, ''),
			COALESCE(item.applied_instruction_version_id::text, ''),
			item.applied_at IS NOT NULL,
			COALESCE(item.applied_page_hash, ''),
			(
				SELECT COUNT(*)
				FROM courseware_review_instruction_versions AS version
				WHERE version.item_id = item.id
			),
			COALESCE(
				(
					SELECT version.status
					FROM courseware_review_instruction_versions AS version
					WHERE version.id = $2
					  AND version.item_id = item.id
				),
				''
			)
		FROM courseware_review_items AS item
		WHERE item.id = $1`,
		r03IntegrationItemID,
		r03IntegrationV1ID,
	).Scan(
		&state.Status,
		&state.CurrentVersionID,
		&state.DeliveredVersionID,
		&state.AppliedVersionID,
		&state.AppliedAtExists,
		&state.AppliedPageHash,
		&state.VersionCount,
		&state.V1Status,
	)
	if err != nil {
		t.Fatalf(
			"读取R-03整改项生命周期状态失败: %v",
			err,
		)
	}

	return state
}

func assertR03IntegrationNoV2(
	t *testing.T,
	state r03IntegrationItemLifecycleState,
) {
	t.Helper()

	if state.CurrentVersionID != r03IntegrationV1ID {
		t.Fatalf(
			"交付后current版本发生变化: %s",
			state.CurrentVersionID,
		)
	}

	if state.DeliveredVersionID != r03IntegrationV1ID {
		t.Fatalf(
			"交付版本发生变化: %s",
			state.DeliveredVersionID,
		)
	}

	if state.VersionCount != 1 {
		t.Fatalf(
			"交付后产生了额外指令版本: count=%d",
			state.VersionCount,
		)
	}

	if state.V1Status !=
		models.CWReviewInstructionVersionStatusConfirmed {
		t.Fatalf(
			"交付V1不再保持confirmed: %s",
			state.V1Status,
		)
	}
}

func assertR03IntegrationAppliedWithV1(
	t *testing.T,
	state r03IntegrationItemLifecycleState,
) {
	t.Helper()

	assertR03IntegrationNoV2(t, state)

	if state.Status != models.CWReviewItemStatusApplied {
		t.Fatalf(
			"整改项没有进入applied: %s",
			state.Status,
		)
	}

	if state.AppliedVersionID != r03IntegrationV1ID {
		t.Fatalf(
			"实际应用版本不是交付V1: %s",
			state.AppliedVersionID,
		)
	}

	if !state.AppliedAtExists {
		t.Fatal("applied状态缺少applied_at")
	}

	if len(strings.TrimSpace(state.AppliedPageHash)) != 64 {
		t.Fatalf(
			"applied_page_hash无效: %q",
			state.AppliedPageHash,
		)
	}
}

// TestR03ReviewHistoryIntegrationPostDeliveryLifecycle 验证真实时间点B：
//
//  1. 正式交付后审核员无法再创建V2；
//  2. 拒绝V2后V1仍同时是current和delivered；
//  3. 作者使用真实Begin/Complete链将整改项推进为applied；
//  4. 实际应用版本仍严格绑定交付V1；
//  5. applied当前状态不得重写旧审核意见、教师视图、交付V1或历史页面。
//
// 本测试会把fixture从confirmed推进到applied；重复运行时若已经处于合法
// applied+V1状态，则直接复核最终历史不变量，不重复制造状态。
func TestR03ReviewHistoryIntegrationPostDeliveryLifecycle(
	t *testing.T,
) {
	ctx := openR03IntegrationDatabase(t)

	reviewService := NewCoursewareReviewService()

	author := buildR03IntegrationActor(
		t,
		ctx,
		r03IntegrationAuthorID,
		models.RoleOperator,
	)
	assertR03TeachingActorDomain(t, author)

	// 时间点B开始前，历史详情必须仍满足时间点A全部事实。
	beforeDetail, err := reviewService.GetReviewHistoryDetail(
		ctx,
		r03IntegrationReviewID,
		author,
	)
	if err != nil {
		t.Fatalf(
			"时间点B开始前读取历史失败: %v",
			err,
		)
	}

	assertR03BaselineHistoryDetail(
		t,
		beforeDetail,
	)

	// B1：使用真实不可变版本Repository尝试创建V2。
	//
	// formal项由审核员创建，因此使用真实created_by senior账号发起。
	// 由于该项已经正式交付，本调用必须在写任何版本之前稳定拒绝。
	version, confirmErr :=
		repository.ConfirmCoursewareReviewInstructionVersion(
			ctx,
			&repository.ConfirmCoursewareReviewInstructionVersionInput{
				ItemID: r03IntegrationItemID,

				ActorID: r03IntegrationSeniorID,

				Instruction: r03IntegrationRejectedV2Content,

				ExpectedCurrentVersionID: r03IntegrationV1ID,

				SourceType: models.CWReviewInstructionVersionSourceManual,
			},
		)

	if version != nil {
		t.Fatalf(
			"交付后V2请求不应返回新版本: %+v",
			version,
		)
	}

	if !errors.Is(
		confirmErr,
		repository.ErrCoursewareReviewInstructionVersionNotConfirmable,
	) {
		t.Fatalf(
			"交付后V2必须稳定返回NotConfirmable，got=%v",
			confirmErr,
		)
	}

	stateAfterRejectedV2 :=
		readR03IntegrationItemLifecycleState(
			t,
			ctx,
		)

	assertR03IntegrationNoV2(
		t,
		stateAfterRejectedV2,
	)

	// B2：只在第一次运行时执行真实confirmed -> applying -> applied。
	switch stateAfterRejectedV2.Status {
	case models.CWReviewItemStatusConfirmed:
		guard, beginErr :=
			BeginCWReviewItemApplication(
				ctx,
				r03IntegrationItemID,
				r03IntegrationCoursewareID,
				1,
				r03IntegrationV1ID,
				r03IntegrationV1Content,
				author,
			)
		if beginErr != nil {
			t.Fatalf(
				"开始正式整改项应用失败: %v",
				beginErr,
			)
		}

		if guard == nil ||
			guard.PageID != r03IntegrationPage1ID ||
			guard.PageNumber != 1 ||
			len(strings.TrimSpace(guard.HTMLHash)) != 64 {
			t.Fatalf(
				"页面应用守卫异常: %+v",
				guard,
			)
		}

		applyingState :=
			readR03IntegrationItemLifecycleState(
				t,
				ctx,
			)

		if applyingState.Status !=
			models.CWReviewItemStatusApplying {
			t.Fatalf(
				"Begin后未进入applying: %+v",
				applyingState,
			)
		}

		if applyingState.AppliedVersionID !=
			r03IntegrationV1ID {
			t.Fatalf(
				"Begin绑定的应用版本不是V1: %s",
				applyingState.AppliedVersionID,
			)
		}

		// 本轮只验证状态生命周期，不改变课件页面。
		//
		// Complete仍走真实页面读取和哈希复核，因此传入当前已知HTML；
		// 后续时间点C再单独制造真实页面修改。
		completeResult, completeErr :=
			CompleteCWReviewItemApplication(
				ctx,
				r03IntegrationItemID,
				r03IntegrationCoursewareID,
				1,
				r03IntegrationPage1HTML,
				author,
			)
		if completeErr != nil {
			// 首次应用若完成失败，尽最大努力恢复confirmed，
			// 避免测试进程留下半完成applying状态。
			_ = AbortCWReviewItemApplication(
				ctx,
				r03IntegrationItemID,
				author,
			)

			t.Fatalf(
				"完成正式整改项应用失败: %v",
				completeErr,
			)
		}

		if completeResult == nil ||
			completeResult.Status !=
				models.CWReviewItemStatusApplied ||
			len(
				strings.TrimSpace(
					completeResult.AppliedPageHash,
				),
			) != 64 {
			t.Fatalf(
				"Complete结果异常: %+v",
				completeResult,
			)
		}

	case models.CWReviewItemStatusApplied:
		// 允许成功后的重复验证。

	case models.CWReviewItemStatusApplying:
		_ = AbortCWReviewItemApplication(
			ctx,
			r03IntegrationItemID,
			author,
		)

		t.Fatal(
			"发现上次测试遗留applying，已尝试恢复；请重新运行本测试",
		)

	default:
		t.Fatalf(
			"时间点B前整改项状态异常: %s",
			stateAfterRejectedV2.Status,
		)
	}

	finalState :=
		readR03IntegrationItemLifecycleState(
			t,
			ctx,
		)

	assertR03IntegrationAppliedWithV1(
		t,
		finalState,
	)

	// applied是今天的执行状态；历史详情必须仍完整等于时间点A。
	afterDetail, err := reviewService.GetReviewHistoryDetail(
		ctx,
		r03IntegrationReviewID,
		author,
	)
	if err != nil {
		t.Fatalf(
			"applied后读取历史失败: %v",
			err,
		)
	}

	assertR03BaselineHistoryDetail(
		t,
		afterDetail,
	)
}
