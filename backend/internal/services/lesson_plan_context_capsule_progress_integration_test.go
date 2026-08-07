package services

// lesson_plan_context_capsule_progress_integration_test.go
//
// 验证：
//   - 教学共识展开后不再截断为6条；
//   - 内部教案确认进度条目不重复展示或注入运行时；
//   - 简短框架不会被识别为详细正文；
//   - 当前AI生成环节三、四后进入“已生成待确认”；
//   - 教师下一轮确认后转为“已确认”；
//   - 模型仅改写summary时不会制造新的确定性进度；
//   - 进度变化会改变版本指纹。

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestLessonPlanCapsuleDisplayShowsAllStableConsensus(
	t *testing.T,
) {
	document :=
		&models.LessonPlanContextCapsuleDocument{
			SchemaVersion: 1,
			Summary:       "本课已形成9项有效教学共识。教案撰写进度：环节一、环节二已确认；环节三、环节四已生成待确认。",
			StageFocus: models.LessonPlanContextCapsuleStageFocus{
				StageCode:   "write",
				CurrentTask: "本轮已生成环节三、环节四的详细教案内容，等待教师确认或修改。",
			},
		}

	for index := 1; index <= 9; index++ {
		document.TeachingConsensus = append(
			document.TeachingConsensus,
			models.LessonPlanContextCapsuleItem{
				Key: fmt.Sprintf(
					"consensus.stable.%d",
					index,
				),
				Title: fmt.Sprintf(
					"稳定共识%d",
					index,
				),
				Content: fmt.Sprintf(
					"教学共识内容%d",
					index,
				),
				State:      models.LessonPlanContextCapsuleItemStateActive,
				Authority:  models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
				Importance: 1,
				UpdatedByTurnID: fmt.Sprintf(
					"t%d_%d",
					index,
					1000+index,
				),
			},
		)
	}

	document.TeachingConsensus = append(
		document.TeachingConsensus,
		models.LessonPlanContextCapsuleItem{
			Key:             lessonPlanCapsuleConfirmedSectionsKey,
			Title:           "已确认教案环节",
			Content:         "教师已确认环节一、环节二的教案内容。",
			State:           models.LessonPlanContextCapsuleItemStateActive,
			Authority:       models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
			Importance:      5,
			UpdatedByTurnID: "t10_1010",
		},
	)

	view :=
		buildLessonPlanContextCapsuleDisplayView(
			document,
			"撰写进度已更新",
		)

	consensusSection :=
		lessonPlanCapsuleTestDisplaySection(
			view.Sections,
			"teaching_consensus",
		)

	if consensusSection == nil {
		t.Fatal(
			"缺少教学共识展示区域",
		)
	}

	if len(consensusSection.Items) != 9 {
		t.Fatalf(
			"教学共识不应截断或包含内部进度条目: got=%d want=9",
			len(consensusSection.Items),
		)
	}

	if consensusSection.Items[0] !=
		"教学共识内容9" {
		t.Fatalf(
			"最近确认内容应优先展示: %q",
			consensusSection.Items[0],
		)
	}

	for _, value := range consensusSection.Items {
		if strings.Contains(
			value,
			"教师已确认环节一",
		) {
			t.Fatalf(
				"内部进度条目不应出现在已经确定的区域: %q",
				value,
			)
		}
	}

	progressSection :=
		lessonPlanCapsuleTestDisplaySection(
			view.Sections,
			"stage_focus",
		)

	if progressSection == nil ||
		len(progressSection.Items) != 1 {
		t.Fatal(
			"当前撰写进度应单独展示",
		)
	}

	if view.StateLabel !=
		"教案撰写中，部分内容待确认" {
		t.Fatalf(
			"待确认状态标签异常: %q",
			view.StateLabel,
		)
	}

	runtimeText :=
		buildLessonPlanContextCapsuleContextText(
			document,
		)

	if strings.Contains(
		runtimeText,
		"教师已确认环节一、环节二的教案内容",
	) {
		t.Fatal(
			"内部撰写进度不应作为稳定教学方向注入AI上下文",
		)
	}
}

func TestLessonPlanCapsuleDetailedSectionsExcludeFramework(
	t *testing.T,
) {
	framework := `教学框架与时间分配：
环节一：绘制手账路线图（8分钟）
环节二：探寻金牌文案秘籍（15分钟）
环节三：创作我的海南手账（15分钟）
环节四：手账发布与评价（7分钟）`

	frameworkSections :=
		lessonPlanCapsuleDetailedGeneratedSections(
			framework,
		)

	if len(frameworkSections) != 0 {
		t.Fatalf(
			"简短框架不应被识别为详细教案正文: %v",
			frameworkSections,
		)
	}

	detailedSections :=
		lessonPlanCapsuleDetailedGeneratedSections(
			lessonPlanCapsuleProgressTestDetailedThreeAndFour(),
		)

	if !reflect.DeepEqual(
		detailedSections,
		[]int{3, 4},
	) {
		t.Fatalf(
			"详细教案环节识别异常: %v",
			detailedSections,
		)
	}
}

func TestLessonPlanCapsuleConfirmedParsingDoesNotIncludeFutureSections(
	t *testing.T,
) {
	text :=
		"教师已确认环节一（路线图）与环节二（文案秘籍）的具体教案内容，后续将继续撰写环节三与环节四。"

	sections :=
		lessonPlanCapsuleConfirmedSectionsFromText(
			text,
		)

	if !reflect.DeepEqual(
		sections,
		[]int{1, 2},
	) {
		t.Fatalf(
			"未来环节不应被误判为已确认: %v",
			sections,
		)
	}
}

func TestLessonPlanCapsuleProgressMarksCurrentGenerationPending(
	t *testing.T,
) {
	current :=
		lessonPlanCapsuleProgressTestCurrentOneAndTwo()

	document :=
		lessonPlanCapsuleProgressTestClone(
			t,
			current,
		)

	// 模拟模型错误地把本轮刚生成的环节三、四写成教师确认。
	document.TeachingConsensus = append(
		document.TeachingConsensus,
		models.LessonPlanContextCapsuleItem{
			Key:             "consensus.lesson_plan_part3_and_part4",
			Title:           "环节三与环节四教案确认",
			Content:         "教师确认环节三与环节四的详细教案内容。",
			State:           models.LessonPlanContextCapsuleItemStateActive,
			Authority:       models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
			Importance:      5,
			UpdatedByTurnID: "turn_15",
		},
	)

	changed :=
		reconcileLessonPlanContextCapsuleProgress(
			document,
			lessonPlanCapsuleProgressTestMarshal(
				t,
				current,
			),
			"确认，请继续写下一部分",
			lessonPlanCapsuleProgressTestDetailedThreeAndFour(),
			"write",
			"turn_15",
		)

	if !changed {
		t.Fatal(
			"生成环节三、四后应产生确定性进度变化",
		)
	}

	if !strings.Contains(
		document.Summary,
		"环节一、环节二已确认",
	) {
		t.Fatalf(
			"摘要缺少已确认范围: %q",
			document.Summary,
		)
	}

	if !strings.Contains(
		document.Summary,
		"环节三、环节四已生成待确认",
	) {
		t.Fatalf(
			"摘要缺少待确认范围: %q",
			document.Summary,
		)
	}

	if !lessonPlanCapsuleProgressIsPending(
		document.StageFocus.CurrentTask,
	) {
		t.Fatalf(
			"当前进度应处于待确认状态: %q",
			document.StageFocus.CurrentTask,
		)
	}

	if _, exists :=
		findLessonPlanCapsuleItemByKey(
			document,
			"consensus.lesson_plan_part3_and_part4",
		); exists {
		t.Fatal(
			"当前AI刚生成的环节三、四不能成为教师确认条目",
		)
	}

	confirmedItem, exists :=
		findLessonPlanCapsuleItemByKey(
			document,
			lessonPlanCapsuleConfirmedSectionsKey,
		)

	if !exists {
		t.Fatal(
			"应建立累积确认范围条目",
		)
	}

	if !strings.Contains(
		confirmedItem.Content,
		"环节一、环节二",
	) ||
		strings.Contains(
			confirmedItem.Content,
			"环节三",
		) {
		t.Fatalf(
			"累积确认范围异常: %q",
			confirmedItem.Content,
		)
	}
}

func TestLessonPlanCapsuleProgressConfirmsPreviousPending(
	t *testing.T,
) {
	current :=
		lessonPlanCapsuleProgressTestCurrentOneAndTwo()

	reconcileLessonPlanContextCapsuleProgress(
		current,
		lessonPlanCapsuleProgressTestMarshal(
			t,
			current,
		),
		"请继续写下一部分",
		lessonPlanCapsuleProgressTestDetailedThreeAndFour(),
		"write",
		"turn_15",
	)

	document :=
		lessonPlanCapsuleProgressTestClone(
			t,
			current,
		)

	document.Summary =
		"模型本轮生成的自由摘要"

	document.StageFocus.CurrentTask =
		"模型本轮生成的自由进度"

	changed :=
		reconcileLessonPlanContextCapsuleProgress(
			document,
			lessonPlanCapsuleProgressTestMarshal(
				t,
				current,
			),
			"确认，生成完整教案",
			"收到，正在继续整理完整教案。",
			"write",
			"turn_16",
		)

	if !changed {
		t.Fatal(
			"教师确认上一版待确认内容后应更新进度",
		)
	}

	if lessonPlanCapsuleProgressIsPending(
		document.Summary,
	) {
		t.Fatalf(
			"确认后不应继续显示待确认: %q",
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
				"摘要缺少已确认范围%s: %q",
				expected,
				document.Summary,
			)
		}
	}

	confirmedItem, exists :=
		findLessonPlanCapsuleItemByKey(
			document,
			lessonPlanCapsuleConfirmedSectionsKey,
		)

	if !exists {
		t.Fatal(
			"缺少累积确认范围条目",
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

func TestLessonPlanCapsuleProgressIgnoresModelWordingOnlyChange(
	t *testing.T,
) {
	current :=
		lessonPlanCapsuleProgressTestCurrentOneAndTwo()

	reconcileLessonPlanContextCapsuleProgress(
		current,
		lessonPlanCapsuleProgressTestMarshal(
			t,
			current,
		),
		"请继续写下一部分",
		lessonPlanCapsuleProgressTestDetailedThreeAndFour(),
		"write",
		"turn_15",
	)

	document :=
		lessonPlanCapsuleProgressTestClone(
			t,
			current,
		)

	document.Summary =
		"模型换了一种摘要措辞"

	document.StageFocus.CurrentTask =
		"模型换了一种进度措辞"

	changed :=
		reconcileLessonPlanContextCapsuleProgress(
			document,
			lessonPlanCapsuleProgressTestMarshal(
				t,
				current,
			),
			"继续看看",
			"好的，我们继续。",
			"write",
			"turn_17",
		)

	if changed {
		t.Fatal(
			"数据库确定性进度未变化时，不应因模型改写措辞制造变化",
		)
	}

	if document.Summary !=
		current.Summary {
		t.Fatalf(
			"应恢复数据库确定性摘要: got=%q want=%q",
			document.Summary,
			current.Summary,
		)
	}

	if document.StageFocus.CurrentTask !=
		current.StageFocus.CurrentTask {
		t.Fatalf(
			"应恢复数据库确定性进度: got=%q want=%q",
			document.StageFocus.CurrentTask,
			current.StageFocus.CurrentTask,
		)
	}
}

func TestLessonPlanCapsuleProgressHashChanges(
	t *testing.T,
) {
	document :=
		lessonPlanCapsuleProgressTestCurrentOneAndTwo()

	document.Summary =
		"摘要一"

	document.StageFocus.CurrentTask =
		"正在撰写环节一与环节二。"

	first :=
		hashLessonPlanContextCapsuleVersionWithProgress(
			strings.Repeat(
				"a",
				64,
			),
			document,
		)

	document.Summary =
		"摘要二"

	document.StageFocus.CurrentTask =
		"环节三、环节四已生成待确认。"

	second :=
		hashLessonPlanContextCapsuleVersionWithProgress(
			strings.Repeat(
				"a",
				64,
			),
			document,
		)

	if first == second {
		t.Fatal(
			"真实进度变化必须产生不同版本指纹",
		)
	}

	if len(first) != 64 ||
		len(second) != 64 {
		t.Fatalf(
			"指纹长度异常: first=%d second=%d",
			len(first),
			len(second),
		)
	}
}

func lessonPlanCapsuleProgressTestCurrentOneAndTwo() *models.LessonPlanContextCapsuleDocument {
	return &models.LessonPlanContextCapsuleDocument{
		SchemaVersion: 1,
		Summary:       "教师已确认环节一与环节二的具体教案内容，后续将继续撰写环节三与环节四。",
		TeachingConsensus: []models.LessonPlanContextCapsuleItem{
			{
				Key:             "consensus.stable.design",
				Title:           "跨主题分层设计",
				Content:         "环节三采用跨主题分层写作，环节四采用旅行社选拔评价。",
				State:           models.LessonPlanContextCapsuleItemStateActive,
				Authority:       models.LessonPlanContextCapsuleAuthorityTeacherExplicit,
				Importance:      5,
				UpdatedByTurnID: "turn_10",
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
			CurrentTask: "环节一与环节二教案已撰写并确认，下一步将撰写环节三与环节四。",
		},
	}
}

func lessonPlanCapsuleProgressTestDetailedThreeAndFour() string {
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

func lessonPlanCapsuleProgressTestMarshal(
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
			"构造测试胶囊失败: %v",
			err,
		)
	}

	return string(encoded)
}

func lessonPlanCapsuleProgressTestClone(
	t *testing.T,
	document *models.LessonPlanContextCapsuleDocument,
) *models.LessonPlanContextCapsuleDocument {
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

	cloned :=
		&models.LessonPlanContextCapsuleDocument{}

	if err := json.Unmarshal(
		encoded,
		cloned,
	); err != nil {
		t.Fatalf(
			"克隆测试胶囊失败: %v",
			err,
		)
	}

	return cloned
}

func lessonPlanCapsuleTestDisplaySection(
	sections []models.LessonPlanContextCapsuleDisplaySection,
	key string,
) *models.LessonPlanContextCapsuleDisplaySection {
	for index := range sections {
		if sections[index].Key == key {
			return &sections[index]
		}
	}

	return nil
}
