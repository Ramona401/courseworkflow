package services

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestCWAIReviewBatchSystemPromptUsesSelectedDimensions(
	t *testing.T,
) {
	session := newTestCWAIReviewSession(
		t,
		[]string{
			models.CWAIReviewDimensionTechnicalImplementation,
			models.CWAIReviewDimensionCustom,
		},
		models.CWAIReviewLessonReferenceLessonIntent,
	)

	prompt, err :=
		buildCWAIReviewBatchSystemPrompt(
			session,
		)
	if err != nil {
		t.Fatalf("构建系统提示词失败: %v", err)
	}

	expectedAllowed :=
		`允许值严格为：["technical_implementation","custom"]`

	if !strings.Contains(
		prompt,
		expectedAllowed,
	) {
		t.Fatalf(
			"系统提示词没有使用会话已选维度: %s",
			prompt,
		)
	}

	oldEnum :=
		`"dimension": "knowledge_accuracy|lesson_alignment|grade_fit`

	if strings.Contains(prompt, oldEnum) {
		t.Fatalf(
			"系统提示词仍暴露旧维度枚举",
		)
	}
}

func TestCWAIReviewNoLessonSanitizesActualBatchInput(
	t *testing.T,
) {
	session := newTestCWAIReviewSession(
		t,
		[]string{
			models.CWAIReviewDimensionTeachingLogic,
		},
		models.CWAIReviewLessonReferenceNoLesson,
	)

	session.BaselineJSON = `{
		"courseware": {
			"title": "测试课件"
		},
		"lesson_plan": {
			"available": true,
			"used": true,
			"content": "绝密教案正文-SENTINEL"
		},
		"course_outline": {
			"available": true,
			"used": true,
			"context": "绝密课程大纲-SENTINEL"
		},
		"alignment_report": {
			"available": true,
			"used": true,
			"report": {
				"secret": "绝密对齐报告-SENTINEL"
			}
		}
	}`

	batch := &models.CoursewareAIReviewBatch{
		BatchNo: 1,
		PageScopeJSON: `{
			"batch_no": 1,
			"start_page": 1,
			"end_page": 1,
			"page_numbers": [1],
			"page_ids": ["page-1"],
			"overlap_from_previous": false,
			"boundary_reason": "测试"
		}`,
	}

	pages := []models.CWAIReviewPageDigest{
		{
			PageID:         "page-1",
			PageNumber:     1,
			Title:          "第一页",
			VisibleText:    "页面真实内容",
			ContentSummary: "页面摘要",
		},
	}

	userPrompt, pageNumbers, err :=
		buildCWAIReviewBatchUserPrompt(
			session,
			batch,
			pages,
			`{"version":1}`,
		)
	if err != nil {
		t.Fatalf("构建批次输入失败: %v", err)
	}

	if len(pageNumbers) != 1 ||
		pageNumbers[0] != 1 {
		t.Fatalf(
			"批次页码错误: %v",
			pageNumbers,
		)
	}

	sentinels := []string{
		"绝密教案正文-SENTINEL",
		"绝密课程大纲-SENTINEL",
		"绝密对齐报告-SENTINEL",
	}

	for _, sentinel := range sentinels {
		if strings.Contains(
			userPrompt,
			sentinel,
		) {
			t.Fatalf(
				"no_lesson真实AI输入泄露材料: %s",
				sentinel,
			)
		}
	}

	requiredMarkers := []string{
		`"lesson_plan":{"available":false,"used":false}`,
		`"course_outline":{"available":false,"titles":[],"used":false}`,
		`"alignment_report":{"available":false,"status":"","summary":"","used":false}`,
	}

	for _, marker := range requiredMarkers {
		if !strings.Contains(
			userPrompt,
			marker,
		) {
			t.Fatalf(
				"no_lesson输入缺少隔离标记: %s",
				marker,
			)
		}
	}
}
