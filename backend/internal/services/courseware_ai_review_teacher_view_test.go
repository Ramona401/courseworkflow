package services

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestCWAIReviewBatchSystemPromptRequiresTeacherViewSnapshot(
	t *testing.T,
) {
	session := newTestCWAIReviewSession(
		t,
		[]string{
			models.CWAIReviewDimensionTechnicalImplementation,
			models.CWAIReviewDimensionInteractionExperience,
		},
		models.CWAIReviewLessonReferenceLessonIntent,
	)

	prompt, err :=
		buildCWAIReviewBatchSystemPrompt(session)
	if err != nil {
		t.Fatalf(
			"构建课件AI审核提示词失败: %v",
			err,
		)
	}

	requiredMarkers := []string{
		`"teacher_view_snapshot"`,
		`"teacher_title"`,
		`"what_happened"`,
		`"teaching_impact"`,
		`"improvement_goal"`,
		`"acceptance_checks"`,
		`"teacher_context": ""`,
		`"manual_check_required"`,
		`"internal_execution_plan"`,
		"必须有2至5条",
		"禁止出现函数名",
	}

	for _, marker := range requiredMarkers {
		if !strings.Contains(prompt, marker) {
			t.Fatalf(
				"提示词缺少教师视图约束 %q",
				marker,
			)
		}
	}
}

func TestNormalizeCWAIReviewTeacherViewRemovesPlatformTerms(
	t *testing.T,
) {
	finding := &models.CWAIReviewFinding{
		Dimension:   models.CWAIReviewDimensionTechnicalImplementation,
		Title:       "JavaScript函数handleNext无法找到DOM元素ID",
		Description: "控制台出现脚本解析错误，CSS选择器没有命中。",
		Suggestion:  "修改HTML和JavaScript并检查localStorage状态。",
		TeacherViewSnapshot: models.CWAIReviewTeacherViewSnapshot{
			TeacherTitle:    "DOM元素ID错误",
			WhatHappened:    "JavaScript函数没有执行",
			TeachingImpact:  "脚本运行时可能报错",
			ImprovementGoal: "修复HTML和CSS选择器",
			AcceptanceChecks: []string{
				"打开控制台查看错误堆栈",
			},
			TeacherContext: "确认localStorage内容",
		},
		ManualReviewRequired: true,
	}

	normalizeCWAIReviewTeacherViewForFinding(
		finding,
	)

	snapshot := finding.TeacherViewSnapshot

	teacherText := strings.Join(
		append(
			[]string{
				snapshot.TeacherTitle,
				snapshot.WhatHappened,
				snapshot.TeachingImpact,
				snapshot.ImprovementGoal,
				snapshot.TeacherContext,
				finding.Suggestion,
			},
			snapshot.AcceptanceChecks...,
		),
		"\n",
	)

	if cwAIReviewTeacherViewContainsPlatformTerm(
		teacherText,
	) {
		t.Fatalf(
			"教师视图仍含平台实现术语: %s",
			teacherText,
		)
	}

	if len(snapshot.AcceptanceChecks) <
		cwAIReviewTeacherCheckMin ||
		len(snapshot.AcceptanceChecks) >
			cwAIReviewTeacherCheckMax {
		t.Fatalf(
			"检查清单数量不符合2至5条要求: %v",
			snapshot.AcceptanceChecks,
		)
	}

	if !snapshot.ManualCheckRequired {
		t.Fatalf(
			"技术证据无法直接教师化时必须要求人工检查",
		)
	}

	if !finding.ManualReviewRequired {
		t.Fatalf(
			"教师快照要求人工检查时finding也必须保留人工复核标记",
		)
	}

	if strings.TrimSpace(
		finding.InternalExecutionPlan,
	) == "" {
		t.Fatalf(
			"原始内部执行计划不应丢失",
		)
	}

	if finding.Suggestion !=
		snapshot.ImprovementGoal {
		t.Fatalf(
			"旧suggestion兼容字段没有收敛为教师调整目标",
		)
	}
}

func TestNormalizeCWAIReviewTeacherViewKeepsValidTeacherLanguage(
	t *testing.T,
) {
	finding := &models.CWAIReviewFinding{
		Dimension: models.CWAIReviewDimensionPageReadability,
		TeacherViewSnapshot: models.CWAIReviewTeacherViewSnapshot{
			TeacherTitle:    "这一页的重点文字不够醒目",
			WhatHappened:    "标题、正文和补充说明的层级比较接近。",
			TeachingImpact:  "学生可能不能快速找到当前讲解重点。",
			ImprovementGoal: "拉开标题、重点和补充内容的视觉层级。",
			AcceptanceChecks: []string{
				"投影查看时能够快速找到页面主标题。",
				"重点内容和补充说明可以明显区分。",
			},
		},
	}

	normalizeCWAIReviewTeacherViewForFinding(
		finding,
	)

	snapshot := finding.TeacherViewSnapshot

	if snapshot.TeacherTitle !=
		"这一页的重点文字不够醒目" {
		t.Fatalf(
			"有效教师标题不应被替换: %s",
			snapshot.TeacherTitle,
		)
	}

	if len(snapshot.AcceptanceChecks) != 2 {
		t.Fatalf(
			"无需人工复核时，有效检查清单不应被无故扩展: %v",
			snapshot.AcceptanceChecks,
		)
	}

	if snapshot.ManualCheckRequired {
		t.Fatalf(
			"完整且安全的教师字段不应被错误标记为人工降级",
		)
	}
}

func TestNormalizeCWAIReviewTeacherViewAppendsManualBrowserCheckWithinBounds(
	t *testing.T,
) {
	finding := &models.CWAIReviewFinding{
		Dimension: models.CWAIReviewDimensionTechnicalImplementation,
		TeacherViewSnapshot: models.CWAIReviewTeacherViewSnapshot{
			TeacherTitle:    "第4页的时间线操作可能不够稳定",
			WhatHappened:    "点击上一步或下一步时，页面可能没有及时变化。",
			TeachingImpact:  "讲解可能中断，学生也可能无法跟随操作。",
			ImprovementGoal: "让时间线的前后切换稳定，并显示清楚的当前位置。",
			AcceptanceChecks: []string{
				"连续点击上一步和下一步，确认页面按顺序变化。",
				"点击重置后，确认页面回到清楚的起始状态。",
			},
		},
		ManualReviewRequired: true,
	}

	normalizeCWAIReviewTeacherViewForFinding(
		finding,
	)

	snapshot := finding.TeacherViewSnapshot

	if len(snapshot.AcceptanceChecks) != 3 {
		t.Fatalf(
			"人工复核项应在两条专项检查后补充为第三条: %v",
			snapshot.AcceptanceChecks,
		)
	}

	foundManualBrowserCheck := false
	for _, check := range snapshot.AcceptanceChecks {
		if check == cwAIReviewManualBrowserCheck {
			foundManualBrowserCheck = true
			break
		}
	}

	if !foundManualBrowserCheck {
		t.Fatalf(
			"人工复核清单缺少浏览器实际操作检查: %v",
			snapshot.AcceptanceChecks,
		)
	}

	if !snapshot.ManualCheckRequired ||
		!finding.ManualReviewRequired {
		t.Fatalf(
			"人工检查标记没有在教师快照与finding之间保持一致",
		)
	}

	if len(snapshot.AcceptanceChecks) <
		cwAIReviewTeacherCheckMin ||
		len(snapshot.AcceptanceChecks) >
			cwAIReviewTeacherCheckMax {
		t.Fatalf(
			"人工复核后的检查清单超出2至5条边界: %v",
			snapshot.AcceptanceChecks,
		)
	}
}
