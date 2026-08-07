package services

// courseware_ai_review_global_discussion_relation_legacy_test.go
//
// 验证旧版只保存item_ids、没有可信source/target的关系，
// 不能通过当前v2可信关系匹配。
//
// 旧消息仍可用于历史展示，但不能由浏览器补充或猜测方向后确认。

import (
	"testing"

	"tedna/internal/models"
)

func TestCWAIReviewGlobalRelationRejectsLegacyItemIDsOnly(
	t *testing.T,
) {
	legacyRelations :=
		[]CWAIReviewGlobalRelation{
			{
				Type: models.
					CWReviewItemRelationConflict,

				ItemIDs: []string{
					"item-1",
					"item-2",
				},

				Explanation: "旧版消息中的关系说明。",
			},
		}

	actual :=
		findCWAIReviewGlobalSuggestedRelation(
			legacyRelations,
			models.CWReviewItemRelationConflict,
			"item-1",
			"item-2",
		)

	if actual != nil {
		t.Fatal(
			"没有source/target的旧关系不应作为可信确认依据",
		)
	}
}

func TestCWAIReviewGlobalRelationRejectsMissingTarget(
	t *testing.T,
) {
	relations :=
		[]CWAIReviewGlobalRelation{
			{
				Type: models.
					CWReviewItemRelationDuplicate,

				SourceItemID: "item-source",

				TargetItemID: "",

				ItemIDs: []string{
					"item-source",
					"item-target",
				},

				Explanation: "缺少可信目标端。",
			},
		}

	actual :=
		findCWAIReviewGlobalSuggestedRelation(
			relations,
			models.CWReviewItemRelationDuplicate,
			"item-source",
			"item-target",
		)

	if actual != nil {
		t.Fatal(
			"缺少可信目标端的关系不应被匹配",
		)
	}
}

func TestCWAIReviewGlobalRelationRejectsBlankExplanation(
	t *testing.T,
) {
	relations :=
		[]CWAIReviewGlobalRelation{
			{
				Type: models.
					CWReviewItemRelationPossiblyResolved,

				SourceItemID: "item-source",

				TargetItemID: "item-target",

				ItemIDs: []string{
					"item-source",
					"item-target",
				},

				Explanation: "   ",
			},
		}

	actual :=
		findCWAIReviewGlobalSuggestedRelation(
			relations,
			models.
				CWReviewItemRelationPossiblyResolved,
			"item-source",
			"item-target",
		)

	if actual != nil {
		t.Fatal(
			"说明为空的关系不应被视为可信建议",
		)
	}
}
