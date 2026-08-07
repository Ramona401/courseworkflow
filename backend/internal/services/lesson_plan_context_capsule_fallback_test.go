package services

// lesson_plan_context_capsule_fallback_test.go
//
// 验证：
//   - 胶囊模型JSON失败时安全复制当前active稳定内容；
//   - 降级后仍能从当前AI回复识别已生成待确认环节；
//   - “确认环节三和四”可直接恢复确认范围；
//   - “确认，请开始写环节三和四”不会误确认三、四；
//   - 否定确认表达不会被误判；
//   - 进入review后仍保留已确认和待确认进度。

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestBuildLessonPlanContextCapsuleDeterministicFallback(
	t *testing.T,
) {
	document :=
		lessonPlanCapsuleFallbackTestDocument()

	current :=
		&models.LessonPlanContextCapsule{
			CapsuleJSON: lessonPlanCapsuleFallbackTestMarshal(
				t,
				document,
			),
		}

	result, err :=
		buildLessonPlanContextCapsuleDeterministicFallback(
			current,
		)

	if err != nil {
		t.Fatalf(
			"确定性降级不应失败: %v",
			err,
		)
	}

	if result == nil {
		t.Fatal(
			"确定性降级结果不能为空",
		)
	}

	if result.Capsule.Summary !=
		document.Summary {
		t.Fatalf(
			"确定性降级应保留当前摘要: got=%q want=%q",
			result.Capsule.Summary,
			document.Summary,
		)
	}

	if len(
		result.Capsule.TeachingConsensus,
	) != len(
		document.TeachingConsensus,
	) {
		t.Fatalf(
			"确定性降级应保留稳定教学共识: got=%d want=%d",
			len(result.Capsule.TeachingConsensus),
			len(document.TeachingConsensus),
		)
	}
}

func TestLessonPlanCapsuleFallbackStillTracksGeneratedSections(
	t *testing.T,
) {
	currentDocument :=
		lessonPlanCapsuleFallbackTestDocument()

	current :=
		&models.LessonPlanContextCapsule{
			CapsuleJSON: lessonPlanCapsuleFallbackTestMarshal(
				t,
				currentDocument,
			),
		}

	result, err :=
		buildLessonPlanContextCapsuleDeterministicFallback(
			current,
		)

	if err != nil {
		t.Fatalf(
			"构造降级胶囊失败: %v",
			err,
		)
	}

	changed :=
		reconcileLessonPlanContextCapsuleProgress(
			&result.Capsule,
			current.CapsuleJSON,
			"请继续完成完整教案。",
			lessonPlanCapsuleFallbackTestDetailedThreeAndFour(),
			"write",
			"turn_fallback_15",
		)

	if !changed {
		t.Fatal(
			"降级后识别到环节三、四正文应更新进度",
		)
	}

	if !strings.Contains(
		result.Capsule.Summary,
		"环节一、环节二已确认",
	) {
		t.Fatalf(
			"摘要应保留原确认范围: %q",
			result.Capsule.Summary,
		)
	}

	if !strings.Contains(
		result.Capsule.Summary,
		"环节三、环节四已生成待确认",
	) {
		t.Fatalf(
			"降级后应记录环节三、四待确认: %q",
			result.Capsule.Summary,
		)
	}
}

func TestLessonPlanCapsuleExplicitNamedConfirmation(
	t *testing.T,
) {
	testCases := []struct {
		Name    string
		Message string
		Want    []int
	}{
		{
			Name:    "完整重复环节前缀",
			Message: "确认环节三和环节四，请继续完善完整教案。",
			Want:    []int{3, 4},
		},
		{
			Name:    "省略第二个环节前缀",
			Message: "确认环节三和四，请继续。",
			Want:    []int{3, 4},
		},
		{
			Name:    "顿号连接",
			Message: "确认环节三、四，请继续。",
			Want:    []int{3, 4},
		},
		{
			Name:    "确认放在末尾",
			Message: "环节三和环节四可以确认",
			Want:    []int{3, 4},
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.Name,
			func(t *testing.T) {
				got :=
					lessonPlanCapsuleExplicitlyConfirmedSections(
						testCase.Message,
					)

				if !reflect.DeepEqual(
					got,
					testCase.Want,
				) {
					t.Fatalf(
						"明确确认环节识别异常: got=%v want=%v",
						got,
						testCase.Want,
					)
				}
			},
		)
	}
}

func TestLessonPlanCapsuleExplicitNamedConfirmationRejectsFalsePositives(
	t *testing.T,
) {
	messages := []string{
		"确认，请开始写环节三和环节四。",
		"暂不确认环节三和环节四。",
		"请确认环节三和环节四是否合理。",
		"确认后再继续写环节三和环节四。",
	}

	for _, message := range messages {
		got :=
			lessonPlanCapsuleExplicitlyConfirmedSections(
				message,
			)

		if len(got) != 0 {
			t.Fatalf(
				"不应把生成请求或否定表达识别为点名确认: message=%q got=%v",
				message,
				got,
			)
		}
	}

	for _, message := range []string{
		"暂不确认。",
		"还未确认。",
		"请确认一下。",
		"确认后再继续。",
	} {
		if teacherExplicitlyConfirmsLessonPlanProgress(
			message,
		) {
			t.Fatalf(
				"不应把否定、疑问或未来条件识别为确认: %q",
				message,
			)
		}
	}
}

func TestReconcileLessonPlanCapsuleDirectConfirmationInReview(
	t *testing.T,
) {
	current :=
		lessonPlanCapsuleFallbackTestDocument()

	document :=
		lessonPlanCapsuleFallbackTestClone(
			t,
			current,
		)

	changed :=
		reconcileLessonPlanContextCapsuleProgress(
			document,
			lessonPlanCapsuleFallbackTestMarshal(
				t,
				current,
			),
			"确认环节三和环节四，请继续检查完整教案。",
			"收到，继续检查完整教案。",
			"review",
			"turn_review_18",
		)

	if !changed {
		t.Fatal(
			"review阶段明确确认环节三、四后应更新胶囊",
		)
	}

	if lessonPlanCapsuleProgressIsPending(
		document.Summary,
	) {
		t.Fatalf(
			"明确确认后不应继续显示待确认: %q",
			document.Summary,
		)
	}

	for _, expected := range []string{
		"环节一",
		"环节二",
		"环节三",
		"环节四",
	} {
		if !strings.Contains(
			document.Summary,
			expected,
		) {
			t.Fatalf(
				"review摘要缺少%s: %q",
				expected,
				document.Summary,
			)
		}
	}

	if !strings.Contains(
		document.Summary,
		"教案核对进度",
	) {
		t.Fatalf(
			"review摘要应使用核对阶段表述: %q",
			document.Summary,
		)
	}

	confirmedItem, exists :=
		findLessonPlanCapsuleItemByKey(
			document,
			lessonPlanCapsuleConfirmedSectionsKey,
		)

	if !exists {
		t.Fatal(
			"明确确认后缺少累积确认条目",
		)
	}

	for _, expected := range []string{
		"环节一",
		"环节二",
		"环节三",
		"环节四",
	} {
		if !strings.Contains(
			confirmedItem.Content,
			expected,
		) {
			t.Fatalf(
				"累积确认条目缺少%s: %q",
				expected,
				confirmedItem.Content,
			)
		}
	}
}

func TestReconcileLessonPlanCapsuleReviewPreservesPending(
	t *testing.T,
) {
	current :=
		lessonPlanCapsuleFallbackTestDocument()

	current.Summary =
		"本课已形成9项有效教学共识。教案撰写进度：环节一、环节二已确认；环节三、环节四已生成待确认。"

	current.StageFocus.CurrentTask =
		"本轮已生成环节三、环节四的详细教案内容，等待教师确认或修改。"

	document :=
		lessonPlanCapsuleFallbackTestClone(
			t,
			current,
		)

	changed :=
		reconcileLessonPlanContextCapsuleProgress(
			document,
			lessonPlanCapsuleFallbackTestMarshal(
				t,
				current,
			),
			"继续检查完整教案。",
			"收到，开始检查。",
			"review",
			"turn_review_19",
		)

	if !changed {
		t.Fatal(
			"从write进入review应更新阶段进度",
		)
	}

	if !lessonPlanCapsuleProgressIsPending(
		document.StageFocus.CurrentTask,
	) {
		t.Fatalf(
			"review阶段必须保留待确认状态: %q",
			document.StageFocus.CurrentTask,
		)
	}

	if !strings.Contains(
		document.StageFocus.CurrentTask,
		"环节三、环节四",
	) {
		t.Fatalf(
			"review阶段缺少待确认环节: %q",
			document.StageFocus.CurrentTask,
		)
	}

	if !strings.Contains(
		document.Summary,
		"教案核对进度",
	) {
		t.Fatalf(
			"review摘要表述异常: %q",
			document.Summary,
		)
	}
}

func lessonPlanCapsuleFallbackTestDocument() *models.LessonPlanContextCapsuleDocument {
	return &models.LessonPlanContextCapsuleDocument{
		SchemaVersion: 1,
		Summary:       "本课已形成9项有效教学共识。教案撰写进度：环节一、环节二已确认。",
		CourseCore: []models.LessonPlanContextCapsuleItem{
			{
				Key:        "course.hainan",
				Title:      "课程核心",
				Content:    "三年级语文《海南岛》聚焦文学阅读与创意表达。",
				State:      models.LessonPlanContextCapsuleItemStateActive,
				Authority:  models.LessonPlanContextCapsuleAuthoritySourceVerified,
				Importance: 5,
			},
		},
		TeachingConsensus: []models.LessonPlanContextCapsuleItem{
			{
				Key:        "consensus.stable.design",
				Title:      "教学设计主线",
				Content:    "采用旅行推荐官情境和跨主题分层写作。",
				State:      models.LessonPlanContextCapsuleItemStateActive,
				Authority:  models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
				Importance: 5,
			},
			{
				Key:             "consensus.lesson_plan_part1_and_part2",
				Title:           "环节一与环节二教案确认",
				Content:         "教师确认环节一与环节二的详细教案内容。",
				State:           models.LessonPlanContextCapsuleItemStateActive,
				Authority:       models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
				Importance:      5,
				UpdatedByTurnID: "turn_14",
			},
		},
		StageFocus: models.LessonPlanContextCapsuleStageFocus{
			StageCode:   "write",
			CurrentTask: "正在依据已确认的环节一、环节二继续撰写和完善完整教案。",
		},
	}
}

func lessonPlanCapsuleFallbackTestDetailedThreeAndFour() string {
	return `环节三：创作“我的海南手账”——分层实践，读写联动（15分钟）

教师话术：掌握了金牌文案秘籍，现在就是各位推荐官大显身手的时候。
学生活动：学生选择两星或三星任务，完成跨主题分层写作。
任务支架：两星任务提供海滩贝壳填空式支架；三星任务提供椰林或五指山总起句。
教师巡视并关注结构完整性与语言生动性。

环节四：手账发布与“星级推荐官”选拔——汇报评价（7分钟）

教师话术：欢迎来到海南旅行社推荐官选拔现场。
学生活动：四人小组轮流朗读并推选代表，上台展示手账。
评价标准：骨架完整、句式规整、语言生动。
教师为学生授予新星或超星推荐官称号。`
}

func lessonPlanCapsuleFallbackTestMarshal(
	t *testing.T,
	document *models.LessonPlanContextCapsuleDocument,
) string {
	t.Helper()

	encoded, err :=
		json.Marshal(
			document,
		)

	if err != nil {
		t.Fatalf(
			"序列化测试胶囊失败: %v",
			err,
		)
	}

	return string(encoded)
}

func lessonPlanCapsuleFallbackTestClone(
	t *testing.T,
	document *models.LessonPlanContextCapsuleDocument,
) *models.LessonPlanContextCapsuleDocument {
	t.Helper()

	encoded :=
		lessonPlanCapsuleFallbackTestMarshal(
			t,
			document,
		)

	cloned :=
		&models.LessonPlanContextCapsuleDocument{}

	if err := json.Unmarshal(
		[]byte(encoded),
		cloned,
	); err != nil {
		t.Fatalf(
			"克隆测试胶囊失败: %v",
			err,
		)
	}

	return cloned
}
