package services

// lesson_plan_context_capsule_stage_decision_test.go
//
// 不连接数据库，只验证write → review的纯确定性胶囊变换：
//   - 正式正文中的环节一至四全部识别；
//   - 旧确认范围环节一、二得到继承；
//   - 本次新增确认范围为环节三、四；
//   - 阶段变为review；
//   - 摘要不再显示待确认；
//   - 简短框架不会被错误确认。

import (
	"reflect"
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestReconcileLessonPlanContextCapsuleWriteReviewDecision(
	t *testing.T,
) {
	current :=
		lessonPlanCapsuleStageDecisionTestDocument()

	document :=
		lessonPlanCapsuleStageDecisionTestDocument()

	generated,
		newlyConfirmed,
		confirmed,
		changed :=
		reconcileLessonPlanContextCapsuleWriteReviewDecision(
			document,
			current,
			lessonPlanCapsuleStageDecisionDetailedContent(),
			"stage_write_confirm_1785603000000",
		)

	if !changed {
		t.Fatal(
			"完整教案推进评审时应更新胶囊",
		)
	}

	if !reflect.DeepEqual(
		generated,
		[]int{1, 2, 3, 4},
	) {
		t.Fatalf(
			"正式正文环节识别异常: %v",
			generated,
		)
	}

	if !reflect.DeepEqual(
		newlyConfirmed,
		[]int{3, 4},
	) {
		t.Fatalf(
			"本次新增确认范围异常: %v",
			newlyConfirmed,
		)
	}

	if !reflect.DeepEqual(
		confirmed,
		[]int{1, 2, 3, 4},
	) {
		t.Fatalf(
			"全部确认范围异常: %v",
			confirmed,
		)
	}

	if document.StageFocus.StageCode !=
		"review" {
		t.Fatalf(
			"阶段焦点应进入review: %q",
			document.StageFocus.StageCode,
		)
	}

	if lessonPlanCapsuleProgressIsPending(
		document.Summary,
	) {
		t.Fatalf(
			"提交评审后不应继续显示待确认: %q",
			document.Summary,
		)
	}

	if !strings.Contains(
		document.Summary,
		"教案核对进度",
	) {
		t.Fatalf(
			"review摘要应显示核对进度: %q",
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
				"摘要缺少%s: %q",
				expected,
				document.Summary,
			)
		}
	}

	item, exists :=
		findLessonPlanCapsuleItemByKey(
			document,
			lessonPlanCapsuleConfirmedSectionsKey,
		)

	if !exists {
		t.Fatal(
			"缺少累积确认条目",
		)
	}

	if item.UpdatedByTurnID !=
		"stage_write_confirm_1785603000000" {
		t.Fatalf(
			"累积确认条目轮次异常: %q",
			item.UpdatedByTurnID,
		)
	}

	for _, expected := range []string{
		"环节一",
		"环节二",
		"环节三",
		"环节四",
	} {
		if !strings.Contains(
			item.Content,
			expected,
		) {
			t.Fatalf(
				"累积确认条目缺少%s: %q",
				expected,
				item.Content,
			)
		}
	}
}

func TestReconcileLessonPlanContextCapsuleWriteReviewDecisionRejectsFramework(
	t *testing.T,
) {
	current :=
		lessonPlanCapsuleStageDecisionTestDocument()

	document :=
		lessonPlanCapsuleStageDecisionTestDocument()

	framework := `教学框架：
环节一：导入（8分钟）
环节二：品读（15分钟）
环节三：写作（15分钟）
环节四：评价（7分钟）`

	generated,
		newlyConfirmed,
		confirmed,
		changed :=
		reconcileLessonPlanContextCapsuleWriteReviewDecision(
			document,
			current,
			framework,
			"stage_write_confirm_1785603000001",
		)

	if changed ||
		len(generated) != 0 ||
		len(newlyConfirmed) != 0 ||
		len(confirmed) != 0 {
		t.Fatalf(
			"简短框架不应被确认为详细教案: generated=%v newly=%v confirmed=%v changed=%v",
			generated,
			newlyConfirmed,
			confirmed,
			changed,
		)
	}
}

func lessonPlanCapsuleStageDecisionTestDocument() *models.LessonPlanContextCapsuleDocument {
	return &models.LessonPlanContextCapsuleDocument{
		SchemaVersion: 1,
		Summary:       "本课已形成9项有效教学共识。教案撰写进度：环节一、环节二已确认；环节三、环节四已生成待确认。",
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
				Key:        "consensus.lesson_plan_confirmed_sections",
				Title:      "已确认教案环节",
				Content:    "教师已确认环节一、环节二的教案内容。",
				State:      models.LessonPlanContextCapsuleItemStateActive,
				Authority:  models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
				Importance: 5,
				ApplicableStages: []string{
					"write",
					"revise",
					"review",
				},
				DoNotReconfirm:  true,
				UpdatedByTurnID: "t1_1785597859178",
			},
		},
		StageFocus: models.LessonPlanContextCapsuleStageFocus{
			StageCode:   "write",
			CurrentTask: "本轮已生成环节三、环节四的详细教案内容，等待教师确认或修改。",
		},
	}
}

func lessonPlanCapsuleStageDecisionDetailedContent() string {
	return `环节一：绘制手账路线图——情境导入与整体感知（8分钟）

教师话术：同学们，今天我们化身海南岛旅行推荐官。
学生活动：学生朗读课文并圈画海滩、椰林、海底和五指山。
教师引导学生完成旅行路线图并交流景点顺序。

环节二：探寻金牌文案秘籍——双线并行（15分钟）

教师话术：先寻找总起句和分述句，搭好文案骨架。
学生活动：学生品读海底鱼多的段落，并从椰林和海滩段落发现比喻、拟人等表达。
教师引导学生总结寻骨架和填血肉两条秘籍。

环节三：创作我的海南手账——分层实践（15分钟）

教师话术：请选择两星或三星任务完成推荐文案。
学生活动：两星学生完成海滩贝壳填空式支架，三星学生围绕椰林或五指山自主仿写。
任务支架强调总起句、分述句和生动表达。

环节四：手账发布与星级推荐官选拔（7分钟）

教师话术：欢迎来到海南旅行社推荐官选拔现场。
学生活动：四人小组轮流朗读、推选代表并上台展示。
评价标准包括骨架完整、句式规整和语言生动。`
}
