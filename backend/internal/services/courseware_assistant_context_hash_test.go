package services

// courseware_assistant_context_hash_test.go
//
// 本测试只验证纯内存的稳定序列化与哈希规则，不连接数据库、不调用AI。

import (
	"strings"
	"testing"
	"time"

	"tedna/internal/models"
)

// TestCoursewareAssistantContextHashIgnoresGeneratedAt 验证装配时间不改变内容哈希。
func TestCoursewareAssistantContextHashIgnoresGeneratedAt(
	t *testing.T,
) {
	firstTime :=
		time.Date(
			2026,
			time.July,
			26,
			9,
			0,
			0,
			0,
			time.UTC,
		)
	secondTime :=
		firstTime.Add(
			10 * time.Minute,
		)

	snapshot :=
		models.AssistantDeploymentContextSnapshot{
			Version:     models.AssistantDeploymentSnapshotVersion,
			GeneratedAt: &firstTime,
			CurrentPage: models.AssistantDeploymentPageContextSnapshot{
				PageID:         "11111111-1111-1111-1111-111111111111",
				PageNumber:     3,
				Title:          "三角形面积探究",
				Purpose:        "通过转化发现面积公式",
				ContentSummary: "把两个相同三角形拼成平行四边形",
				VisibleText:    "拖动三角形，观察拼接后的图形变化。",
				InteractionEvidence: models.CWAIReviewInteractionEvidence{
					DeclaredType: "drag",
					Events: []models.CWAIReviewInteractionEvent{
						{
							EventType: "pointerdown",
							Trigger:   "#triangle",
							Handler:   "startDrag",
						},
					},
					ReachableFunctions: []models.CWAIReviewReachableFunction{},
					StateVariables: []string{
						"dragging = false",
					},
					DOMTargets: []string{
						"#triangle",
						"#target",
					},
					CSSStateRules:          []string{},
					InitialExposureSignals: []string{},
					RiskFlags:              []string{},
				},
			},
		}

	firstHash, err :=
		hashCoursewareAssistantContextSnapshot(
			snapshot,
		)
	if err != nil {
		t.Fatalf(
			"第一次计算上下文哈希失败: %v",
			err,
		)
	}

	snapshot.GeneratedAt =
		&secondTime

	secondHash, err :=
		hashCoursewareAssistantContextSnapshot(
			snapshot,
		)
	if err != nil {
		t.Fatalf(
			"第二次计算上下文哈希失败: %v",
			err,
		)
	}

	if firstHash != secondHash {
		t.Fatalf(
			"仅装配时间变化时哈希不应改变: first=%s second=%s",
			firstHash,
			secondHash,
		)
	}

	snapshot.CurrentPage.VisibleText =
		"内容已经发生变化"

	changedHash, err :=
		hashCoursewareAssistantContextSnapshot(
			snapshot,
		)
	if err != nil {
		t.Fatalf(
			"计算变化后上下文哈希失败: %v",
			err,
		)
	}

	if changedHash == firstHash {
		t.Fatalf(
			"教学内容变化后哈希必须改变: hash=%s",
			changedHash,
		)
	}
}

// TestCoursewareAssistantContextJSONIncludesGeneratedAt 验证持久化JSON保留装配时间。
func TestCoursewareAssistantContextJSONIncludesGeneratedAt(
	t *testing.T,
) {
	generatedAt :=
		time.Date(
			2026,
			time.July,
			26,
			9,
			30,
			0,
			0,
			time.UTC,
		)

	snapshot :=
		models.AssistantDeploymentContextSnapshot{
			Version:     models.AssistantDeploymentSnapshotVersion,
			GeneratedAt: &generatedAt,
			CurrentPage: models.AssistantDeploymentPageContextSnapshot{
				PageID:     "22222222-2222-2222-2222-222222222222",
				PageNumber: 1,
				Title:      "导入",
				InteractionEvidence: emptyCoursewareAssistantInteractionEvidence(
					"static",
				),
			},
		}

	encoded, err :=
		marshalCoursewareAssistantContextSnapshot(
			snapshot,
		)
	if err != nil {
		t.Fatalf(
			"序列化上下文失败: %v",
			err,
		)
	}

	if !strings.Contains(
		encoded,
		`"generated_at":"2026-07-26T09:30:00Z"`,
	) {
		t.Fatalf(
			"上下文JSON缺少生成时间: %s",
			encoded,
		)
	}
}
