package services

// courseware_comic_free_knowledge_test.go
//
// 只测试纯函数和JSON快照，不连接数据库、不调用AI。

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"tedna/internal/models"
)

func TestCoursewareComicFreeKnowledgeBuildsCompatibleSnapshot(
	t *testing.T,
) {
	courseware :=
		&models.Courseware{
			Title:
				"九年级化学复习",
			Subject:
				"化学",
			Grade:
				"九年级",
		}

	knowledgeText :=
		"电解质与非电解质的区别、判断依据和常见误区。\n" +
			"重点说明物质在水溶液中能否产生自由移动离子。"

	source, err :=
		buildCoursewareComicFreeKnowledgeSource(
			courseware,
			knowledgeText,
		)
	if err != nil {
		t.Fatalf(
			"自由知识点快照构建失败: %v",
			err,
		)
	}

	if source.Publisher !=
		coursewareComicCustomPublisher {
		t.Fatalf(
			"自由知识点出版社标记错误: %q",
			source.Publisher,
		)
	}

	if _, err :=
		uuid.Parse(
			source.UnitID,
		); err != nil {
		t.Fatalf(
			"虚拟单元ID不是合法UUID: %q",
			source.UnitID,
		)
	}

	var unit models.CoursewareComicTextbookUnitSnapshot

	if err :=
		json.Unmarshal(
			[]byte(
				source.UnitSnapshotJSON,
			),
			&unit,
		); err != nil {
		t.Fatalf(
			"教材单元快照JSON无效: %v",
			err,
		)
	}

	if unit.ID != source.UnitID ||
		unit.GradeNum != 9 ||
		len(unit.KPCodes) != 1 {
		t.Fatalf(
			"自由知识点单元快照内容错误: %+v",
			unit,
		)
	}

	var points []models.CoursewareComicKnowledgePointSnapshot

	if err :=
		json.Unmarshal(
			[]byte(
				source.KnowledgePointsJSON,
			),
			&points,
		); err != nil {
		t.Fatalf(
			"知识点快照JSON无效: %v",
			err,
		)
	}

	if len(points) != 1 ||
		points[0].KPCode !=
			coursewareComicCustomKPCode ||
		points[0].ContentRequirement !=
			knowledgeText {
		t.Fatalf(
			"教师知识点快照内容错误: %+v",
			points,
		)
	}

	if !strings.Contains(
		source.KnowledgeContent,
		"自由移动离子",
	) {
		t.Fatalf(
			"知识内容没有保存教师原文: %q",
			source.KnowledgeContent,
		)
	}
}

func TestCoursewareComicFreeKnowledgeAutomaticDefaults(
	t *testing.T,
) {
	request :=
		&models.CreateCoursewareComicProjectRequest{
			KnowledgeText:
				"通过烧杯实验探究电解质与非电解质的判断依据。",
		}

	courseware :=
		&models.Courseware{
			Title:
				"化学课堂",
			Subject:
				"化学",
			Grade:
				"九年级",
		}

	applyCoursewareComicAutomaticDefaults(
		request,
		courseware,
	)

	if request.Title == "" {
		t.Fatal(
			"系统没有自动生成漫画标题",
		)
	}

	if request.NarrativeMode !=
		"inquiry_mystery" {
		t.Fatalf(
			"实验主题应自动使用探究叙事，实际为: %q",
			request.NarrativeMode,
		)
	}

	if request.VisualStyle == "" {
		t.Fatal(
			"系统没有自动选择视觉风格",
		)
	}

	if request.PanelCount < 4 ||
		request.PanelCount > 8 {
		t.Fatalf(
			"自动格数超出4至8格: %d",
			request.PanelCount,
		)
	}

	if !models.IsValidCWComicLayoutMode(
		request.LayoutMode,
	) {
		t.Fatalf(
			"自动布局无效: %q",
			request.LayoutMode,
		)
	}
}

func TestCoursewareComicFreeKnowledgeRejectsInvalidText(
	t *testing.T,
) {
	courseware :=
		&models.Courseware{
			Subject:
				"化学",
			Grade:
				"九年级",
		}

	if _, err :=
		buildCoursewareComicFreeKnowledgeSource(
			courseware,
			"   ",
		); err == nil {
		t.Fatal(
			"空白知识点应被拒绝",
		)
	}

	overLimit :=
		strings.Repeat(
			"知",
			coursewareComicKnowledgeTextMaxRunes+1,
		)

	if _, err :=
		buildCoursewareComicFreeKnowledgeSource(
			courseware,
			overLimit,
		); err == nil {
		t.Fatal(
			"超长知识点应被拒绝",
		)
	}
}
