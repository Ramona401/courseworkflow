package services

import (
	"context"
	"testing"

	"tedna/internal/models"
)

func TestRuntimeComponentReceiptUsesActualLoadedCount(
	t *testing.T,
) {
	groups := []*models.MatchedComponentGroup{
		{
			LibraryType: models.LibPedagogy,
			LibraryName: "教学方法",
			Components: []*models.MatchedComponent{
				{
					ID:           "component-a",
					DisplayLabel: "组件A",
				},
				{
					ID:           "component-b",
					DisplayLabel: "组件B",
				},
			},
		},
	}

	receipt := runtimeComponentReceiptFromGroups(
		groups,
		"manual",
		false,
	)

	if receipt.Status != models.ContextReceiptLoaded {
		t.Fatalf(
			"状态应为loaded，got=%s",
			receipt.Status,
		)
	}

	if receipt.CandidateCount != 2 {
		t.Fatalf(
			"CandidateCount必须等于实际加载数2，got=%d",
			receipt.CandidateCount,
		)
	}

	if len(receipt.Items) != 2 {
		t.Fatalf(
			"Items应为2，got=%d",
			len(receipt.Items),
		)
	}
}

func TestBuildAutoComponentsReceiptFailsClosedWithoutDomain(
	t *testing.T,
) {
	receipt := buildAutoComponentsReceipt(
		context.Background(),
		`["pedagogy"]`,
		"数学",
		"三年级",
		"design",
		"想增加小组讨论",
	)

	if receipt.Status !=
		models.ContextReceiptUnavailable {
		t.Fatalf(
			"无教案域Context必须unavailable，got=%s",
			receipt.Status,
		)
	}

	if len(receipt.Items) != 0 {
		t.Fatalf(
			"无教案域时不能返回组件，got=%d",
			len(receipt.Items),
		)
	}
}

func TestBuildRuntimeRerankedReceiptUsesActualCandidates(
	t *testing.T,
) {
	groups := []*models.MatchedComponentGroup{
		{
			LibraryType: models.LibPedagogy,
			LibraryName: "教学方法",
			Components: []*models.MatchedComponent{
				{
					ID:             "component-a",
					DisplayLabel:   "小组讨论",
					ComponentIndex: "[F]小组讨论",
					QualityScore:   8.5,
				},
				{
					ID:             "component-b",
					DisplayLabel:   "讲授",
					ComponentIndex: "[F]教师讲授",
					QualityScore:   9.0,
				},
			},
		},
	}

	receipt := buildRuntimeRerankedReceipt(
		groups,
		"我想安排小组讨论",
	)

	if receipt.Status != models.ContextReceiptLoaded {
		t.Fatalf(
			"状态应为loaded，got=%s",
			receipt.Status,
		)
	}

	if receipt.CandidateCount != 2 {
		t.Fatalf(
			"候选数必须是实际同域候选数2，got=%d",
			receipt.CandidateCount,
		)
	}

	if len(receipt.Items) != 2 {
		t.Fatalf(
			"TopN范围内应返回2项，got=%d",
			len(receipt.Items),
		)
	}

	if receipt.Items[0].ID != "component-a" {
		t.Fatalf(
			"关键词命中的组件应排第一，got=%s",
			receipt.Items[0].ID,
		)
	}
}

func TestRuntimeComponentReceiptEmptyGroups(
	t *testing.T,
) {
	receipt := runtimeComponentReceiptFromGroups(
		nil,
		"recipe",
		false,
	)

	if receipt.Status != models.ContextReceiptNotFound {
		t.Fatalf(
			"空实际结果应为not_found，got=%s",
			receipt.Status,
		)
	}

	if receipt.CandidateCount != 0 {
		t.Fatalf(
			"空实际结果候选数应为0，got=%d",
			receipt.CandidateCount,
		)
	}
}
