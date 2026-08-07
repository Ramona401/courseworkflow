package services

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestCWAIReviewBatchResultMapsOldDimension(
	t *testing.T,
) {
	session := newTestCWAIReviewSession(
		t,
		[]string{
			models.CWAIReviewDimensionPageReadability,
		},
		models.CWAIReviewLessonReferenceNoLesson,
	)

	raw := `{
		"batch_no": 99,
		"page_numbers": [99],
		"batch_summary": "测试",
		"findings": [
			{
				"id": "",
				"severity": "unexpected",
				"dimension": "visual_load",
				"page_numbers": [2, 2, -1],
				"title": "字号过小",
				"description": "页面阅读困难",
				"lesson_or_outline_basis": "模型伪造的教案依据",
				"page_evidence": "正文过密",
				"code_evidence": "",
				"continuity_evidence": "",
				"suggestion": "调整字号",
				"confidence": 120,
				"manual_review_required": false
			}
		],
		"continuity_ledger": {},
		"risk_pages": [],
		"manual_review_required": false
	}`

	result, _, _, _, err :=
		parseCWAIReviewBatchResult(
			raw,
			session,
			1,
			[]int{2},
			`{"version":1}`,
		)
	if err != nil {
		t.Fatalf("解析批次结果失败: %v", err)
	}

	if result.BatchNo != 1 {
		t.Fatalf(
			"批次号未由后端覆盖: %d",
			result.BatchNo,
		)
	}

	if len(result.Findings) != 1 {
		t.Fatalf(
			"finding数量错误: %d",
			len(result.Findings),
		)
	}

	finding := result.Findings[0]

	if finding.Dimension !=
		models.CWAIReviewDimensionPageReadability {
		t.Fatalf(
			"旧维度未正确映射: %s",
			finding.Dimension,
		)
	}

	if !finding.ManualReviewRequired {
		t.Fatalf(
			"后端修正维度后必须要求人工复核",
		)
	}

	if finding.LessonOrOutlineBasis != "" {
		t.Fatalf(
			"no_lesson未清空教案依据: %q",
			finding.LessonOrOutlineBasis,
		)
	}

	if finding.Confidence != 100 {
		t.Fatalf(
			"置信度未限制到100: %d",
			finding.Confidence,
		)
	}

	if len(finding.PageNumbers) != 1 ||
		finding.PageNumbers[0] != 2 {
		t.Fatalf(
			"页码未去重去非法值: %v",
			finding.PageNumbers,
		)
	}
}

func TestCWAIReviewUnknownDimensionFallsBackToCustom(
	t *testing.T,
) {
	session := newTestCWAIReviewSession(
		t,
		[]string{
			models.CWAIReviewDimensionCustom,
		},
		models.CWAIReviewLessonReferenceCurrentCompatible,
	)

	finding := &models.CWAIReviewFinding{
		Dimension:   "unclassified",
		Title:       "特殊校本要求",
		Description: "没有标准维度对应",
	}

	if err := normalizeCWAIReviewFindingForSession(
		session,
		finding,
	); err != nil {
		t.Fatalf("收敛finding失败: %v", err)
	}

	if finding.Dimension !=
		models.CWAIReviewDimensionCustom {
		t.Fatalf(
			"未知维度未回退到custom: %s",
			finding.Dimension,
		)
	}

	if !finding.ManualReviewRequired {
		t.Fatalf(
			"未知维度回退后必须要求人工复核",
		)
	}
}

func TestCWAIReviewFinalReportOverridesModelConfig(
	t *testing.T,
) {
	session := newTestCWAIReviewSession(
		t,
		[]string{
			models.CWAIReviewDimensionTechnicalImplementation,
		},
		models.CWAIReviewLessonReferenceNoLesson,
	)

	raw := `{
		"review_config": {
			"schema_version": 999,
			"review_dimensions": ["custom"],
			"custom_dimension_description": "模型伪造",
			"lesson_reference_mode": "strict_alignment",
			"review_config_hash": "fake"
		},
		"overall_risk": "high",
		"summary": "综合报告",
		"strengths": [],
		"findings": [
			{
				"id": "FINAL-F1",
				"severity": "high",
				"dimension": "interaction",
				"page_numbers": [1],
				"title": "按钮无反馈",
				"description": "点击后状态不明确",
				"lesson_or_outline_basis": "伪造教案要求",
				"page_evidence": "按钮存在",
				"code_evidence": "事件函数异常",
				"continuity_evidence": "",
				"suggestion": "修复事件",
				"confidence": 80,
				"manual_review_required": false
			}
		],
		"priority_actions": [],
		"manual_review_pages": [],
		"review_comment_draft": "草稿",
		"human_decision_reminder": ""
	}`

	report, reportJSON, err :=
		parseCWAIReviewFinalReport(
			raw,
			session,
		)
	if err != nil {
		t.Fatalf("解析最终报告失败: %v", err)
	}

	if report.ReviewConfig.SchemaVersion !=
		models.CWAIReviewConfigSchemaVersion {
		t.Fatalf(
			"最终报告采用了模型伪造协议版本: %d",
			report.ReviewConfig.SchemaVersion,
		)
	}

	if report.ReviewConfig.ReviewConfigHash !=
		session.ReviewConfigHash {
		t.Fatalf(
			"最终报告配置哈希未由后端覆盖",
		)
	}

	if report.ReviewConfig.LessonReferenceMode !=
		models.CWAIReviewLessonReferenceNoLesson {
		t.Fatalf(
			"最终报告教案模式未由后端覆盖: %s",
			report.ReviewConfig.LessonReferenceMode,
		)
	}

	if len(report.Findings) != 1 ||
		report.Findings[0].Dimension !=
			models.CWAIReviewDimensionTechnicalImplementation {
		t.Fatalf(
			"最终finding维度未收敛: %+v",
			report.Findings,
		)
	}

	if report.Findings[0].LessonOrOutlineBasis != "" {
		t.Fatalf(
			"最终报告no_lesson仍保留教案依据",
		)
	}

	if !report.Findings[0].ManualReviewRequired {
		t.Fatalf(
			"最终维度被修正后未要求人工复核",
		)
	}

	if strings.Contains(
		reportJSON,
		`"review_config_hash":"fake"`,
	) {
		t.Fatalf(
			"序列化报告仍包含模型伪造配置",
		)
	}
}
