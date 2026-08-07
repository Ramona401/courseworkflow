package services

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

// TestCWImageAOCIBlocksAndRelations 验证多条IAOCI解析和合法关系。
func TestCWImageAOCIBlocksAndRelations(
	t *testing.T,
) {
	raw := `
IK:@I-111111111111|IV:1|IT:I|UR:EX|CT:0|SB:O|AR:H|RC:0
[F]实验装置初始状态
[L]横向中景，装置位于画面中央
[A]清晰教学插画，柔和光影
[C]O1实验装置
[S]明亮实验操作台
[E]横图，高清，无文字
[R]0
[N]禁止装置结构错误

IK:@I-222222222222|IV:1|IT:I|UR:EX|CT:3|SB:O|AR:H|RC:1
[F]实验装置反应后的状态
[L]保持观察角度，突出状态变化
[A]清晰教学插画，柔和光影
[C]O1实验装置
[S]同一实验操作台
[E]横图，高清，无文字
[R]>@I-111111111111[ACSO]{继续展示同一装置的反应结果}
[N]禁止改变装置身份
`

	items, err :=
		cwParseImageAOCIBlocks(raw)
	if err != nil {
		t.Fatalf(
			"解析多条IAOCI失败：%v",
			err,
		)
	}

	if len(items) != 2 {
		t.Fatalf(
			"IAOCI数量错误：got=%d want=2",
			len(items),
		)
	}

	slots := []cwImagePlaceholderSlot{
		{
			PlaceholderID: "IMG_SLOT_01",
			ImageKey:      "@I-111111111111",
			Order:         1,
		},
		{
			PlaceholderID: "IMG_SLOT_02",
			ImageKey:      "@I-222222222222",
			Order:         2,
		},
	}

	expectedOrder := map[string]int{
		"@I-111111111111": 1,
		"@I-222222222222": 2,
	}

	expectedSlot :=
		map[string]cwImagePlaceholderSlot{
			"@I-111111111111": slots[0],
			"@I-222222222222": slots[1],
		}

	if err := cwValidatePlannedImageAOCIs(
		items,
		slots,
		expectedOrder,
		expectedSlot,
		map[string]bool{},
	); err != nil {
		t.Fatalf(
			"合法图片关系未通过校验：%v",
			err,
		)
	}
}

// TestCWImageAOCIRejectForwardRelation 验证同页不能引用后续槽位。
func TestCWImageAOCIRejectForwardRelation(
	t *testing.T,
) {
	first := &models.ImageAOCI{
		ImageKey:      "@I-111111111111",
		IndexType:     models.CWImageIndexTypeImage,
		RelationCount: "1",
		Relations: []models.CoursewareImageRelationSpec{
			{
				TargetImageKey: "@I-222222222222",
				RelationCode:   models.CWImageRelationContinue,
				InheritMask:    "AC",
			},
		},
	}

	second := &models.ImageAOCI{
		ImageKey:      "@I-222222222222",
		IndexType:     models.CWImageIndexTypeImage,
		RelationCount: "0",
	}

	slots := []cwImagePlaceholderSlot{
		{
			PlaceholderID: "IMG_SLOT_01",
			ImageKey:      "@I-111111111111",
			Order:         1,
		},
		{
			PlaceholderID: "IMG_SLOT_02",
			ImageKey:      "@I-222222222222",
			Order:         2,
		},
	}

	err := cwValidatePlannedImageAOCIs(
		[]*models.ImageAOCI{
			first,
			second,
		},
		slots,
		map[string]int{
			"@I-111111111111": 1,
			"@I-222222222222": 2,
		},
		map[string]cwImagePlaceholderSlot{
			"@I-111111111111": slots[0],
			"@I-222222222222": slots[1],
		},
		map[string]bool{},
	)

	if err == nil {
		t.Fatal(
			"同页前序槽位引用后续槽位应当被拒绝",
		)
	}
}

// TestCWLegacyAnchorOnlyKeepsArt 验证旧锚点只继承艺术风格。
func TestCWLegacyAnchorOnlyKeepsArt(
	t *testing.T,
) {
	courseware := &models.Courseware{
		StyleAnchorVAOCI: "风格锚点 | L:教室全景 | A:柔和水彩插画、纸张纹理、暖色光影 | C:C1短发学生 | S:教室",
	}

	anchor :=
		cwParseCoursewareAnchorAOCI(
			courseware,
		)

	if anchor == nil {
		t.Fatal(
			"旧锚点没有提取到艺术风格",
		)
	}

	if !strings.Contains(
		anchor.ArtText,
		"柔和水彩插画",
	) {
		t.Fatalf(
			"旧锚点艺术风格提取错误：%s",
			anchor.ArtText,
		)
	}

	if anchor.CharacterText != "Ø" {
		t.Fatalf(
			"旧锚点不应继承人物：%s",
			anchor.CharacterText,
		)
	}

	if anchor.SceneText != "Ø" ||
		anchor.LayoutText != "Ø" {
		t.Fatal(
			"旧锚点不应继承环境或构图",
		)
	}

	if anchor.SubjectType !=
		models.CWImageSubjectNone {
		t.Fatal(
			"旧锚点安全兼容应当使用无固定主体",
		)
	}
}
