package services

// courseware_ai_review_global_discussion_test.go
//
// 验证全局讨论主服务中的纯选择规范化、可信助手结果恢复和
// 整改项ID匹配逻辑。
//
// 本文件不连接数据库、不调用AI、不修改审核状态。

import (
	"encoding/json"
	"errors"
	"testing"

	"tedna/internal/models"
)

func TestCWAIReviewGlobalDiscussionNormalizesItemIDs(
	t *testing.T,
) {
	actual, err :=
		normalizeCWAIReviewGlobalItemIDs(
			[]string{
				" item-2 ",
				"item-1",
				"item-2",
				"",
			},
		)
	if err != nil {
		t.Fatalf(
			"整改项ID规范化失败：%v",
			err,
		)
	}

	expected := []string{
		"item-2",
		"item-1",
	}

	if len(actual) != len(expected) {
		t.Fatalf(
			"整改项ID数量错误：expected=%v actual=%v",
			expected,
			actual,
		)
	}

	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf(
				"整改项ID顺序或内容错误：expected=%v actual=%v",
				expected,
				actual,
			)
		}
	}
}

func TestCWAIReviewGlobalDiscussionRejectsInvalidItemCount(
	t *testing.T,
) {
	cases := []struct {
		name  string
		input []string
	}{
		{
			name:  "少于最小数量",
			input: []string{"item-1"},
		},
		{
			name: "去重后少于最小数量",
			input: []string{
				"item-1",
				"item-1",
			},
		},
		{
			name: "超过最大数量",
			input: []string{
				"item-01",
				"item-02",
				"item-03",
				"item-04",
				"item-05",
				"item-06",
				"item-07",
				"item-08",
				"item-09",
				"item-10",
				"item-11",
				"item-12",
				"item-13",
			},
		},
	}

	for _, item := range cases {
		t.Run(
			item.name,
			func(t *testing.T) {
				actual, err :=
					normalizeCWAIReviewGlobalItemIDs(
						item.input,
					)

				if actual != nil {
					t.Fatalf(
						"非法数量不应返回结果：%v",
						actual,
					)
				}

				if !errors.Is(
					err,
					ErrCWAIReviewGlobalSelectionInvalid,
				) {
					t.Fatalf(
						"非法数量错误类型不正确：%v",
						err,
					)
				}
			},
		)
	}
}

func TestCWAIReviewGlobalDiscussionUsesLatestTrustedAssistant(
	t *testing.T,
) {
	olderMeta, err := json.Marshal(
		cwAIReviewGlobalDiscussionMeta{
			Kind:    cwAIReviewGlobalMetadataKind,
			Summary: "较早结论",
			SelectedItemIDs: []string{
				"item-1",
				"item-2",
			},
			Relations: []CWAIReviewGlobalRelation{},
			Proposals: []CWAIReviewGlobalProposal{},
		},
	)
	if err != nil {
		t.Fatalf(
			"序列化较早元数据失败：%v",
			err,
		)
	}

	latestMeta, err := json.Marshal(
		cwAIReviewGlobalDiscussionMeta{
			Kind:    cwAIReviewGlobalMetadataKind,
			Summary: "最新可信结论",
			SelectedItemIDs: []string{
				"item-2",
				"item-3",
			},
			Relations: []CWAIReviewGlobalRelation{
				{
					Type: models.CWReviewItemRelationDependency,
					ItemIDs: []string{
						"item-2",
						"item-3",
					},
					Explanation: "item-2依赖item-3。",
				},
			},
			Proposals: []CWAIReviewGlobalProposal{
				{
					ItemID:               "item-2",
					Recommendation:       "revise",
					Reason:               "存在依赖关系",
					SuggestedInstruction: "先完成item-3，再处理item-2。",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"序列化最新元数据失败：%v",
			err,
		)
	}

	messages :=
		[]*models.CoursewareAIReviewMessage{
			{
				ID:            "assistant-old",
				Role:          "assistant",
				CitationsJSON: string(olderMeta),
			},
			{
				ID:            "assistant-invalid",
				Role:          "assistant",
				CitationsJSON: `{"kind":"untrusted"}`,
			},
			{
				ID:            "assistant-latest",
				Role:          "assistant",
				CitationsJSON: string(latestMeta),
			},
		}

	result :=
		buildCWAIReviewGlobalDiscussionResult(
			messages,
		)

	if result == nil {
		t.Fatal("全局讨论结果为空")
	}

	if result.LatestMessageID !=
		"assistant-latest" ||
		result.Summary !=
			"最新可信结论" {
		t.Fatalf(
			"没有恢复最新可信助手消息：%+v",
			result,
		)
	}

	if len(result.SelectedItemIDs) != 2 ||
		result.SelectedItemIDs[0] != "item-2" ||
		result.SelectedItemIDs[1] != "item-3" ||
		len(result.Relations) != 1 ||
		len(result.Proposals) != 1 {
		t.Fatalf(
			"最新可信结构化结果恢复错误：%+v",
			result,
		)
	}
}

func TestCWAIReviewGlobalDiscussionDefaultsToEmptyCollections(
	t *testing.T,
) {
	result :=
		buildCWAIReviewGlobalDiscussionResult(
			nil,
		)

	if result == nil {
		t.Fatal("空消息列表结果为空")
	}

	if result.Messages != nil ||
		result.Relations == nil ||
		result.Proposals == nil ||
		result.SelectedItemIDs == nil {
		t.Fatalf(
			"空消息列表默认集合错误：%+v",
			result,
		)
	}
}

func TestCWAIReviewGlobalDiscussionContainsItemID(
	t *testing.T,
) {
	itemIDs := []string{
		" item-1 ",
		"item-2",
	}

	if !containsCWAIReviewGlobalItemID(
		itemIDs,
		"item-1",
	) {
		t.Fatal("未匹配去空后的整改项ID")
	}

	if containsCWAIReviewGlobalItemID(
		itemIDs,
		"item-3",
	) {
		t.Fatal("错误匹配不存在的整改项ID")
	}
}
