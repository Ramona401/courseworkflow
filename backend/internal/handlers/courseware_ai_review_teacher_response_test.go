package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestBuildCoursewareAIReviewRunNextViewHidesInternalFindingData(
	t *testing.T,
) {
	result := &models.CWAIReviewBatchAIResult{
		BatchNo:      1,
		PageNumbers:  []int{2},
		BatchSummary: "JavaScript DOM检查完成",
		Findings: []models.CWAIReviewFinding{
			{
				ID:          "F-1",
				Severity:    "high",
				Dimension:   models.CWAIReviewDimensionTechnicalImplementation,
				PageNumbers: []int{2},

				Title:                 "JavaScript函数无法找到DOM元素ID",
				Description:           "控制台出现脚本错误",
				LessonOrOutlineBasis:  "内部教案依据",
				PageEvidence:          "内部页面证据",
				CodeEvidence:          "document.querySelector调用失败",
				ContinuityEvidence:    "内部连续性账本",
				Suggestion:            "修复HTML和CSS",
				InternalExecutionPlan: "修改函数和选择器",

				TeacherViewSnapshot: models.CWAIReviewTeacherViewSnapshot{
					TeacherTitle:    "第2页的课堂操作可能不够稳定",
					WhatHappened:    "点击主要按钮后，页面可能没有清楚反馈。",
					TeachingImpact:  "讲解可能中断，影响课堂节奏。",
					ImprovementGoal: "让主要操作稳定完成，并显示清楚结果。",
					AcceptanceChecks: []string{
						"逐一点击主要按钮，确认每次都有清楚结果。",
						"重复操作一次，确认页面不会中断。",
					},
				},
			},
		},
		ContinuityLedger: map[string]interface{}{
			"secret": "内部连续性账本",
		},
		RiskPages: []models.CWAIReviewRiskPage{
			{
				PageNumber:           2,
				Severity:             "high",
				Reason:               "DOM运行时需要浏览器检查",
				EvidenceType:         "runtime",
				ManualReviewRequired: true,
			},
		},
	}

	view :=
		buildCoursewareAIReviewRunNextView(
			&models.CWAIReviewRunNextResponse{
				Result: result,
			},
		)

	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatalf(
			"序列化浏览器批次视图失败: %v",
			err,
		)
	}

	text := strings.ToLower(string(encoded))

	for _, forbidden := range []string{
		"internal_execution_plan",
		"code_evidence",
		"page_evidence",
		"continuity_evidence",
		"lesson_or_outline_basis",
		"document.queryselector",
		"javascript",
		"dom运行时",
		`"secret":"内部连续性账本"`,
	} {
		if strings.Contains(text, strings.ToLower(forbidden)) {
			t.Fatalf(
				"浏览器批次视图泄露内部内容 %q: %s",
				forbidden,
				string(encoded),
			)
		}
	}

	if !strings.Contains(
		string(encoded),
		`"teacher_title":"第2页的课堂操作可能不够稳定"`,
	) {
		t.Fatalf(
			"浏览器批次视图缺少教师标题: %s",
			string(encoded),
		)
	}

	if !strings.Contains(
		string(encoded),
		`"continuity_ledger":{}`,
	) {
		t.Fatalf(
			"安全批次响应缺少旧前端所需的空账本占位: %s",
			string(encoded),
		)
	}
}

func TestBuildCoursewareAIReviewSessionViewSanitizesFinalReportJSON(
	t *testing.T,
) {
	report := models.CWAIReviewFinalReport{
		OverallRisk: "high",
		Summary:     "JavaScript脚本和DOM存在问题",
		Strengths: []string{
			"教学结构清楚",
		},
		Findings: []models.CWAIReviewFinding{
			{
				ID:                    "F-2",
				Severity:              "high",
				Dimension:             models.CWAIReviewDimensionInteractionExperience,
				PageNumbers:           []int{3},
				Title:                 "内部脚本错误",
				CodeEvidence:          "window.addEventListener",
				InternalExecutionPlan: "重写事件函数",
				TeacherViewSnapshot: models.CWAIReviewTeacherViewSnapshot{
					TeacherTitle:    "第3页的互动反馈不够清楚",
					WhatHappened:    "完成操作后，页面可能没有显示明确结果。",
					TeachingImpact:  "学生可能不知道操作是否成功。",
					ImprovementGoal: "让每次操作后都有清楚的文字反馈。",
					AcceptanceChecks: []string{
						"完成一次操作，确认页面显示文字结果。",
						"重复操作，确认反馈始终清楚一致。",
					},
				},
			},
		},
		PriorityActions:       []models.CWAIReviewPriorityAction{},
		ManualReviewPages:     []int{3},
		HumanDecisionReminder: "最终决定由教师确认。",
	}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf(
			"序列化内部最终报告失败: %v",
			err,
		)
	}

	view :=
		buildCoursewareAIReviewSessionView(
			&models.CoursewareAIReviewSession{
				Status:          models.CWAIReviewStatusDone,
				FinalReportJSON: string(raw),
				ModelUsed:       "secret-model",
				TokensUsed:      999,
				ErrorMessage:    "SQL internal error",
			},
		)

	if view == nil {
		t.Fatal("会话浏览器视图不应为空")
	}

	text := strings.ToLower(
		view.FinalReportJSON,
	)

	for _, forbidden := range []string{
		"internal_execution_plan",
		"code_evidence",
		"window.addeventlistener",
		"javascript",
		"secret-model",
		"sql internal error",
	} {
		if strings.Contains(text, strings.ToLower(forbidden)) {
			t.Fatalf(
				"最终报告浏览器JSON泄露内部内容 %q: %s",
				forbidden,
				view.FinalReportJSON,
			)
		}
	}

	if view.ModelUsed != "" ||
		view.TokensUsed != 0 {
		t.Fatalf(
			"会话浏览器视图不应返回模型和Token: %#v",
			view,
		)
	}

	if view.ErrorMessage !=
		"本次检查未能完成，请稍后重试。" {
		t.Fatalf(
			"内部错误没有收敛为教师提示: %s",
			view.ErrorMessage,
		)
	}
}

func TestBuildCoursewareAIReviewItemViewUsesTeacherSnapshotOnly(
	t *testing.T,
) {
	teacherView :=
		models.CWAIReviewTeacherViewSnapshot{
			TeacherTitle:    "第4页的时间线操作可能不够稳定",
			WhatHappened:    "点击上一步或下一步时，页面可能没有及时变化。",
			TeachingImpact:  "讲解可能中断，学生也可能无法跟随操作。",
			ImprovementGoal: "让时间线的前后切换稳定，并显示清楚的当前位置。",
			AcceptanceChecks: []string{
				"连续点击上一步和下一步，确认页面按顺序变化。",
				"点击重置后，确认页面回到清楚的起始状态。",
			},
			ManualCheckRequired: true,
		}

	evidence, err := json.Marshal(
		map[string]interface{}{
			"teacher_view_snapshot":   teacherView,
			"code_evidence":           "document.getElementById失败",
			"internal_execution_plan": "修改JavaScript函数",
			"review_config_hash":      "secret-config-hash",
		},
	)
	if err != nil {
		t.Fatalf(
			"序列化整改项内部证据失败: %v",
			err,
		)
	}

	view :=
		buildCoursewareAIReviewItemView(
			&models.CoursewareReviewItem{
				ID:                 "item-1",
				Dimension:          models.CWAIReviewDimensionTechnicalImplementation,
				Title:              "JavaScript函数错误",
				Description:        "DOM元素不存在",
				EvidenceJSON:       string(evidence),
				OriginalSuggestion: "修改HTML和脚本",
				PageHTMLHash:       "page-secret-hash",
				AppliedPageHash:    "applied-secret-hash",
			},
		)

	if view == nil {
		t.Fatal("整改项浏览器视图不应为空")
	}

	if view.Title !=
		teacherView.TeacherTitle ||
		view.Description !=
			teacherView.WhatHappened ||
		view.ImprovementGoal !=
			teacherView.ImprovementGoal {
		t.Fatalf(
			"整改项没有使用固化教师快照: %#v",
			view,
		)
	}

	if view.PageHTMLHash != "" ||
		view.AppliedPageHash != "" {
		t.Fatalf(
			"整改项浏览器视图泄露页面哈希: %#v",
			view,
		)
	}

	for _, forbidden := range []string{
		"document.getelementbyid",
		"internal_execution_plan",
		"secret-config-hash",
		"javascript",
		"page-secret-hash",
		"applied-secret-hash",
	} {
		if strings.Contains(
			strings.ToLower(view.EvidenceJSON),
			strings.ToLower(forbidden),
		) {
			t.Fatalf(
				"整改项教师证据JSON泄露内部内容 %q: %s",
				forbidden,
				view.EvidenceJSON,
			)
		}
	}

	if len(view.AcceptanceChecks) < 2 ||
		len(view.AcceptanceChecks) > 5 {
		t.Fatalf(
			"整改项检查清单数量不符合2至5条要求: %v",
			view.AcceptanceChecks,
		)
	}

	if !view.ManualCheckRequired {
		t.Fatalf(
			"整改项人工检查标记没有保留",
		)
	}
}
