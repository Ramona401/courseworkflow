package handlers

import (
	"encoding/json"
	"testing"

	"tedna/internal/models"
)

// TestBuildCoursewareAIReviewSafeBatchResultJSONKeepsEmptyLedgerShape
// 固化旧前端批次解析器仍依赖的结构：
//
// continuity_ledger必须存在且为对象，但其中不得包含任何内部事实。
func TestBuildCoursewareAIReviewSafeBatchResultJSONKeepsEmptyLedgerShape(
	t *testing.T,
) {
	raw, err := json.Marshal(
		&models.CWAIReviewBatchAIResult{
			BatchNo:      2,
			PageNumbers:  []int{4, 5},
			BatchSummary: "本批页面已经完成检查。",
			Findings:     []models.CWAIReviewFinding{},
			ContinuityLedger: map[string]interface{}{
				"teaching_thread": map[string]interface{}{
					"secret": "不得返回浏览器",
				},
			},
			RiskPages: []models.CWAIReviewRiskPage{},
		},
	)
	if err != nil {
		t.Fatalf(
			"序列化内部批次结果失败: %v",
			err,
		)
	}

	safeJSON :=
		buildCoursewareAIReviewSafeBatchResultJSON(
			string(raw),
		)

	if safeJSON == "" {
		t.Fatal(
			"安全批次JSON不应为空",
		)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(
		[]byte(safeJSON),
		&parsed,
	); err != nil {
		t.Fatalf(
			"解析安全批次JSON失败: %v",
			err,
		)
	}

	ledger, exists :=
		parsed["continuity_ledger"]
	if !exists {
		t.Fatalf(
			"安全批次JSON缺少continuity_ledger兼容字段: %s",
			safeJSON,
		)
	}

	ledgerObject, ok :=
		ledger.(map[string]interface{})
	if !ok {
		t.Fatalf(
			"continuity_ledger必须为对象: %#v",
			ledger,
		)
	}

	if len(ledgerObject) != 0 {
		t.Fatalf(
			"安全批次JSON泄露连续性事实: %#v",
			ledgerObject,
		)
	}
}
