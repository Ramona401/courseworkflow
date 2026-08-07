package services

// courseware_comic_generation_prompt_test.go
//
// 本文件仅测试纯函数：
//   - 图片生成侧Unicode中文截断；
//   - 人物设定图无文字约束；
//   - 单格图片无文字与气泡分层约束。
//
// 不连接数据库、不启动后台任务、不调用图片模型。

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

func TestCoursewareComicGenerationUnicodeTruncate(
	t *testing.T,
) {
	result :=
		truncateCoursewareComicGenerationRunes(
			"  电解质与非电解质  ",
			4,
		)

	if result != "电解质与" {
		t.Fatalf(
			"Unicode截断结果错误: %q",
			result,
		)
	}

	unchanged :=
		truncateCoursewareComicGenerationRunes(
			"  完整内容  ",
			20,
		)

	if unchanged != "完整内容" {
		t.Fatalf(
			"短文本规范化结果错误: %q",
			unchanged,
		)
	}

	empty :=
		truncateCoursewareComicGenerationRunes(
			"内容",
			0,
		)

	if empty != "" {
		t.Fatalf(
			"非正数限制应返回空字符串: %q",
			empty,
		)
	}
}

func TestCoursewareComicGenerationPromptsForbidText(
	t *testing.T,
) {
	project :=
		&models.CoursewareComicProject{
			StyleAOCIText:
				"@ANCHOR [A]清晰教学漫画",
			CharacterBibleJSON:
				`{"version":1,"characters":[{"id":"CHAR-01","name":"小科"}]}`,
		}

	panel :=
		&models.CoursewareComicPanel{
			VisualPrompt:
				"学生观察烧杯中的物质变化",
			AOCIText:
				"@I-000000000001 [A]清晰教学漫画",
			NegativePrompt:
				"禁止错误实验器材",
		}

	characterPrompt :=
		buildCoursewareComicCharacterSheetPrompt(
			project,
		)

	panelPrompt :=
		buildCoursewareComicPanelGenerationPrompt(
			project,
			panel,
		)

	requiredCharacterRules := []string{
		"人物设定参考图",
		"不得生成角色姓名",
		"Logo",
		"水印",
		"任何可读文字",
	}

	for _, required :=
		range requiredCharacterRules {
		if !strings.Contains(
			characterPrompt,
			required,
		) {
			t.Fatalf(
				"人物设定图提示词缺少硬约束: %s",
				required,
			)
		}
	}

	requiredPanelRules := []string{
		"跨格人物固定设定",
		"本格完整IAOCI",
		"不得生成任何文字",
		"公式",
		"字幕",
		"Logo",
		"水印",
		"气泡",
	}

	for _, required :=
		range requiredPanelRules {
		if !strings.Contains(
			panelPrompt,
			required,
		) {
			t.Fatalf(
				"漫画格提示词缺少硬约束: %s",
				required,
			)
		}
	}
}
