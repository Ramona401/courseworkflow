package repository

import (
	"context"
	"errors"
	"testing"

	"tedna/internal/database"
	"tedna/internal/models"
)

// TestR07ImpactAtomicSmoke 只允许对一次性tedna_r07_smoke_*数据库运行。
//
// 覆盖：
//  1. group + relation + candidate + create_item + dismiss混合成功提交；
//  2. 非法selected operation ID整笔拒绝；
//  3. 可信source message变化整笔拒绝；
//  4. mixed plan中目标stale时group/item/relation全部零写入。
func TestR07ImpactAtomicSmoke(t *testing.T) {
	ctx := context.Background()
	fixture := newR07SmokeFixture(t, ctx)

	t.Run("mixed_success_commits_all_operations", func(t *testing.T) {
		itemA := fixture.cloneDetectedItem(t, ctx, "success-a")
		itemB := fixture.cloneDetectedItem(t, ctx, "success-b")
		messageID := fixture.createTrustedMessage(t, ctx, "mixed-success")

		groupName := fixture.marker + "-success-group"
		candidateInstruction := "R07 smoke候选建议：保持候选态，不得自动确认。"
		createItemTitle := fixture.marker + "-created-item"

		createGroup := newR07SmokeCreateGroupOperation(
			t,
			itemA,
			groupName,
			"smoke建立问题组",
		)

		createRelation := newR07SmokeCreateRelationOperation(
			t,
			itemA,
			itemB,
			models.CWReviewItemRelationDependency,
			"smoke验证依赖关系",
		)

		updateCandidate := newR07SmokeUpdateCandidateOperation(
			t,
			itemB,
			candidateInstruction,
		)

		createItem := newR07SmokeCreateItemOperation(
			t,
			createItemTitle,
		)

		dismissItem := newR07SmokeDismissOperation(
			t,
			itemB,
			"smoke教师明确暂不处理",
		)

		operations := []models.CoursewareReviewImpactOperation{
			createGroup,
			createRelation,
			updateCandidate,
			createItem,
			dismissItem,
		}

		plan := fixture.createPlan(t, ctx, messageID, operations)
		selectedIDs := r07SmokeOperationIDs(operations)

		applied, err := ApplyCoursewareReviewImpactPlan(
			ctx,
			plan.ID,
			fixture.sessionID,
			1,
			fixture.actorID,
			selectedIDs,
		)
		if err != nil {
			t.Fatalf("mixed success Apply失败: %v", err)
		}

		assertR07SmokePlanApplied(
			t,
			ctx,
			fixture,
			applied,
			selectedIDs,
		)

		assertR07SmokeMixedBusinessResult(
			t,
			ctx,
			fixture,
			plan.ID,
			itemA,
			itemB,
			groupName,
			createRelation,
			updateCandidate,
			createItem,
		)
	})

	t.Run("invalid_selection_writes_nothing", func(t *testing.T) {
		item := fixture.cloneDetectedItem(t, ctx, "invalid-selection")
		messageID := fixture.createTrustedMessage(t, ctx, "invalid-selection")
		groupName := fixture.marker + "-invalid-selection-group"

		operation := newR07SmokeCreateGroupOperation(
			t,
			item,
			groupName,
			"smoke非法selection",
		)

		plan := fixture.createPlan(
			t,
			ctx,
			messageID,
			[]models.CoursewareReviewImpactOperation{operation},
		)

		_, err := ApplyCoursewareReviewImpactPlan(
			ctx,
			plan.ID,
			fixture.sessionID,
			1,
			fixture.actorID,
			[]string{fixture.newUUID(t, ctx)},
		)
		if !errors.Is(err, ErrCoursewareReviewImpactSelectionInvalid) {
			t.Fatalf("期望invalid selection错误，实际=%v", err)
		}

		assertR07SmokePlanStillDraft(t, ctx, fixture, plan.ID)
		assertR07SmokeGroupAbsent(t, ctx, fixture, groupName)
	})

	t.Run("source_message_hash_change_writes_nothing", func(t *testing.T) {
		item := fixture.cloneDetectedItem(t, ctx, "source-hash")
		messageID := fixture.createTrustedMessage(t, ctx, "source-hash")
		groupName := fixture.marker + "-source-hash-group"

		operation := newR07SmokeCreateGroupOperation(
			t,
			item,
			groupName,
			"smoke来源hash变化",
		)

		plan := fixture.createPlan(
			t,
			ctx,
			messageID,
			[]models.CoursewareReviewImpactOperation{operation},
		)

		result, err := database.DB.Exec(
			ctx,
			`UPDATE courseware_ai_review_messages
			 SET content = content || ' [SMOKE MUTATED]'
			 WHERE id = $1
			   AND session_id = $2`,
			messageID,
			fixture.sessionID,
		)
		if err != nil {
			t.Fatalf("修改隔离库可信消息失败: %v", err)
		}
		if result.RowsAffected() != 1 {
			t.Fatalf("修改可信消息影响行数=%d，期望1", result.RowsAffected())
		}

		_, err = ApplyCoursewareReviewImpactPlan(
			ctx,
			plan.ID,
			fixture.sessionID,
			1,
			fixture.actorID,
			[]string{operation.OperationID},
		)
		if !errors.Is(err, ErrCoursewareReviewImpactPlanConflict) {
			t.Fatalf("期望source hash conflict，实际=%v", err)
		}

		assertR07SmokePlanStillDraft(t, ctx, fixture, plan.ID)
		assertR07SmokeGroupAbsent(t, ctx, fixture, groupName)
	})

	t.Run("mixed_stale_target_rolls_back_everything", func(t *testing.T) {
		itemA := fixture.cloneDetectedItem(t, ctx, "stale-a")
		itemB := fixture.cloneDetectedItem(t, ctx, "stale-b")
		messageID := fixture.createTrustedMessage(t, ctx, "mixed-stale")

		groupName := fixture.marker + "-stale-group"
		createItemTitle := fixture.marker + "-stale-created-item"
		candidateInstruction := "R07 stale smoke候选建议"

		createGroup := newR07SmokeCreateGroupOperation(
			t,
			itemA,
			groupName,
			"stale smoke建立组",
		)

		createRelation := newR07SmokeCreateRelationOperation(
			t,
			itemA,
			itemB,
			models.CWReviewItemRelationDependency,
			"stale smoke依赖关系",
		)

		updateCandidate := newR07SmokeUpdateCandidateOperation(
			t,
			itemB,
			candidateInstruction,
		)

		createItem := newR07SmokeCreateItemOperation(
			t,
			createItemTitle,
		)

		operations := []models.CoursewareReviewImpactOperation{
			createGroup,
			createRelation,
			updateCandidate,
			createItem,
		}

		plan := fixture.createPlan(t, ctx, messageID, operations)

		result, err := database.DB.Exec(
			ctx,
			`UPDATE courseware_review_items
			 SET status = 'discussing',
			     updated_at = clock_timestamp()
			 WHERE id = $1
			   AND status = 'detected'`,
			itemB.ID,
		)
		if err != nil {
			t.Fatalf("制造stale目标失败: %v", err)
		}
		if result.RowsAffected() != 1 {
			t.Fatalf("制造stale目标影响行数=%d，期望1", result.RowsAffected())
		}

		_, err = ApplyCoursewareReviewImpactPlan(
			ctx,
			plan.ID,
			fixture.sessionID,
			1,
			fixture.actorID,
			r07SmokeOperationIDs(operations),
		)
		if !errors.Is(err, ErrCoursewareReviewImpactPlanConflict) {
			t.Fatalf("期望mixed stale conflict，实际=%v", err)
		}

		assertR07SmokePlanStillDraft(t, ctx, fixture, plan.ID)

		assertR07SmokeMixedRollback(
			t,
			ctx,
			fixture,
			plan.ID,
			itemA,
			itemB,
			groupName,
			createRelation,
			updateCandidate,
			createItem,
		)
	})
}
