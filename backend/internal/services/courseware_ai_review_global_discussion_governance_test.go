package services

// courseware_ai_review_global_discussion_governance_test.go
//
// 验证全局讨论治理层中的纯输入规范化、可信关系与提案匹配、
// 以及整课人工整改项构造。
//
// 本文件不连接数据库、不调用AI、不改变课件或审核状态。
//
// 关系测试使用当前v2协议：
//   - SourceItemID保存可信源端；
//   - TargetItemID保存可信目标端；
//   - ItemIDs严格等于[source_item_id, target_item_id]；
//   - conflict虽然业务无方向，但进入可信协议前已经按文本升序规范化。

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"tedna/internal/models"
)

func TestCWAIReviewGlobalGovernanceNormalizeManualPageIDs(
	t *testing.T,
) {
	actual, err :=
		normalizeCWAIReviewGlobalManualPageIDs(
			[]string{
				" page-2 ",
				"",
				"page-1",
				"page-2",
				"   ",
			},
		)
	if err != nil {
		t.Fatalf(
			"页面ID规范化失败：%v",
			err,
		)
	}

	expected := []string{
		"page-2",
		"page-1",
	}

	if len(actual) != len(expected) {
		t.Fatalf(
			"页面ID数量错误：expected=%v actual=%v",
			expected,
			actual,
		)
	}

	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf(
				"页面ID顺序或内容错误：expected=%v actual=%v",
				expected,
				actual,
			)
		}
	}
}

func TestCWAIReviewGlobalGovernanceRejectsTooManyManualPageIDs(
	t *testing.T,
) {
	input := make(
		[]string,
		0,
		cwAIReviewGlobalManualMaxPages+1,
	)

	for index := 0; index <= cwAIReviewGlobalManualMaxPages; index++ {
		input = append(
			input,
			fmt.Sprintf(
				"page-%03d",
				index,
			),
		)
	}

	actual, err :=
		normalizeCWAIReviewGlobalManualPageIDs(
			input,
		)

	if actual != nil {
		t.Fatalf(
			"超量页面ID不应返回结果：%v",
			actual,
		)
	}

	if !errors.Is(
		err,
		ErrCWAIReviewGlobalManualItemInvalid,
	) {
		t.Fatalf(
			"超量页面ID错误类型不正确：%v",
			err,
		)
	}
}

func TestCWAIReviewGlobalGovernanceFindsTrustedRelation(
	t *testing.T,
) {
	relations := []CWAIReviewGlobalRelation{
		{
			Type: models.
				CWReviewItemRelationConflict,

			// conflict已经按文本升序规范化。
			SourceItemID: "item-1",
			TargetItemID: "item-2",

			ItemIDs: []string{
				"item-1",
				"item-2",
			},

			Explanation: "两条整改建议互相冲突。",
		},
		{
			Type: models.
				CWReviewItemRelationDependency,

			SourceItemID: "item-1",
			TargetItemID: "item-3",

			ItemIDs: []string{
				"item-1",
				"item-3",
			},

			Explanation: "",
		},
	}

	actual :=
		findCWAIReviewGlobalSuggestedRelation(
			relations,
			models.CWReviewItemRelationConflict,
			"item-1",
			"item-2",
		)

	if actual == nil {
		t.Fatal(
			"没有找到符合v2协议的可信冲突关系",
		)
	}

	if actual.Explanation !=
		"两条整改建议互相冲突。" {
		t.Fatalf(
			"可信关系说明错误：%q",
			actual.Explanation,
		)
	}

	emptyExplanation :=
		findCWAIReviewGlobalSuggestedRelation(
			relations,
			models.CWReviewItemRelationDependency,
			"item-1",
			"item-3",
		)

	if emptyExplanation != nil {
		t.Fatal(
			"关系说明为空时不应作为可信建议返回",
		)
	}
}

func TestCWAIReviewGlobalGovernanceFindsProposal(
	t *testing.T,
) {
	proposals := []CWAIReviewGlobalProposal{
		{
			ItemID: " item-1 ",

			Recommendation: "consider_dismiss",

			Reason: "问题已被其他整改项覆盖。",
		},
	}

	actual :=
		findCWAIReviewGlobalProposal(
			proposals,
			"item-1",
		)

	if actual == nil {
		t.Fatal(
			"没有找到去空后的可信提案",
		)
	}

	if actual.Recommendation !=
		"consider_dismiss" {
		t.Fatalf(
			"可信提案类型错误：%q",
			actual.Recommendation,
		)
	}

	if findCWAIReviewGlobalProposal(
		proposals,
		"item-2",
	) != nil {
		t.Fatal(
			"错误匹配不存在的提案",
		)
	}
}

func TestCWAIReviewGlobalGovernanceBuildsCoursewareManualItem(
	t *testing.T,
) {
	session :=
		&models.CoursewareAIReviewSession{
			ID: "session-1",

			ReviewerID: "reviewer-1",

			ReviewLevel: models.CWAIReviewLevelSelf,
		}

	courseware :=
		&models.Courseware{
			ID:     "courseware-1",
			UserID: "owner-1",
		}

	sourceMessageID :=
		"message-1"

	item, err :=
		buildCWAIReviewGlobalManualItem(
			session,
			courseware,
			"manual-source-1",
			models.CWReviewItemSourceSelf,
			&sourceMessageID,
			"人工发现的问题",
			"问题描述",
			"候选修改指令",
			models.CWReviewSeverityHigh,
			"content_accuracy",
			nil,
		)
	if err != nil {
		t.Fatalf(
			"构造整课人工整改项失败：%v",
			err,
		)
	}

	if item == nil {
		t.Fatal(
			"整课人工整改项为空",
		)
	}

	if item.CoursewareID != courseware.ID ||
		item.SourceSessionID != session.ID ||
		item.SourceFindingID !=
			"manual-source-1" ||
		item.OriginType !=
			models.
				CWReviewItemOriginGlobalDiscussionManual ||
		item.SourceGlobalMessageID == nil ||
		*item.SourceGlobalMessageID !=
			sourceMessageID ||
		item.SourceType !=
			models.CWReviewItemSourceSelf ||
		item.CreatedBy !=
			session.ReviewerID ||
		item.OwnerID != courseware.UserID ||
		item.Status !=
			models.CWReviewItemStatusDetected ||
		!item.IsGlobalIssue() {
		t.Fatalf(
			"整课人工整改项核心字段错误：%+v",
			item,
		)
	}

	var evidence map[string]interface{}

	if err := json.Unmarshal(
		[]byte(item.EvidenceJSON),
		&evidence,
	); err != nil {
		t.Fatalf(
			"人工整改项证据JSON无效：%v",
			err,
		)
	}

	if evidence["origin_type"] !=
		models.
			CWReviewItemOriginGlobalDiscussionManual ||
		evidence["source_global_message_id"] !=
			sourceMessageID ||
		evidence["scope"] != "courseware" {
		t.Fatalf(
			"人工整改项证据内容错误：%v",
			evidence,
		)
	}
}
