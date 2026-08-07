package services

// courseware_ai_review_global_discussion_relation_match_test.go
//
// 验证v2可信关系必须严格匹配：
//   - 关系类型；
//   - SourceItemID；
//   - TargetItemID；
//   - 有序ItemIDs；
//   - 非空Explanation。
//
// 该严格性阻止浏览器反转dependency、duplicate、merge和
// possibly_resolved等有方向关系。

import (
	"testing"

	"tedna/internal/models"
)

func TestCWAIReviewGlobalRelationRequiresExactDirection(
	t *testing.T,
) {
	relations := []CWAIReviewGlobalRelation{
		{
			Type: models.
				CWReviewItemRelationDependency,

			SourceItemID: "item-dependent",
			TargetItemID: "item-prerequisite",

			ItemIDs: []string{
				"item-dependent",
				"item-prerequisite",
			},

			Explanation: "必须先完成前置问题。",
		},
	}

	direct :=
		findCWAIReviewGlobalSuggestedRelation(
			relations,
			models.CWReviewItemRelationDependency,
			"item-dependent",
			"item-prerequisite",
		)

	if direct == nil {
		t.Fatal(
			"正确方向的dependency关系未被匹配",
		)
	}

	reversed :=
		findCWAIReviewGlobalSuggestedRelation(
			relations,
			models.CWReviewItemRelationDependency,
			"item-prerequisite",
			"item-dependent",
		)

	if reversed != nil {
		t.Fatal(
			"反转后的dependency关系不应被匹配",
		)
	}
}

func TestCWAIReviewGlobalRelationRequiresOrderedItemIDs(
	t *testing.T,
) {
	relations := []CWAIReviewGlobalRelation{
		{
			Type: models.
				CWReviewItemRelationMerge,

			SourceItemID: "item-source",
			TargetItemID: "item-target",

			// 故意与source/target顺序相反。
			ItemIDs: []string{
				"item-target",
				"item-source",
			},

			Explanation: "源问题应合并到目标问题。",
		},
	}

	actual :=
		findCWAIReviewGlobalSuggestedRelation(
			relations,
			models.CWReviewItemRelationMerge,
			"item-source",
			"item-target",
		)

	if actual != nil {
		t.Fatal(
			"item_ids顺序错误的关系不应被视为可信关系",
		)
	}
}

func TestCWAIReviewGlobalConflictMatchesNormalizedOrder(
	t *testing.T,
) {
	relations := []CWAIReviewGlobalRelation{
		{
			Type: models.
				CWReviewItemRelationConflict,

			SourceItemID: "item-a",
			TargetItemID: "item-b",

			ItemIDs: []string{
				"item-a",
				"item-b",
			},

			Explanation: "两条修改指令冲突。",
		},
	}

	actual :=
		findCWAIReviewGlobalSuggestedRelation(
			relations,
			models.CWReviewItemRelationConflict,
			"item-a",
			"item-b",
		)

	if actual == nil {
		t.Fatal(
			"规范化后的conflict关系应被匹配",
		)
	}
}
