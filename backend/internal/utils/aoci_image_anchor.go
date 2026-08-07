package utils

// aoci_image_anchor.go — 课程锚点IAOCI输出清理与提纯
//
// 课程锚点只允许锁定：
//   - [A]艺术风格；
//   - [C]固定人物、动物或标志性主体身份。
//
// 课程锚点不得锁定：
//   - [L]构图、镜头、景别、主体位置；
//   - [S]教室、课桌、家具和具体环境；
//   - [R]跨图关系。

import (
	"fmt"
	"strings"

	"tedna/internal/models"
)

// CleanImageAOCIOutput 从AI响应中提取纯IAOCI。
func CleanImageAOCIOutput(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	startIndex := -1
	endIndex := -1

	for index, line := range lines {
		trimmed := strings.TrimSpace(line)

		if startIndex < 0 &&
			strings.HasPrefix(trimmed, "IK:") &&
			strings.Contains(trimmed, "|IV:") &&
			strings.Contains(trimmed, "|IT:") {
			startIndex = index
		}

		if startIndex >= 0 &&
			isImageAOCISemanticLine(trimmed) {
			endIndex = index
		}
	}

	if startIndex >= 0 && endIndex >= startIndex {
		selected := make(
			[]string,
			0,
			endIndex-startIndex+1,
		)

		for index := startIndex; index <= endIndex; index++ {
			line := strings.TrimSpace(lines[index])
			if line != "" {
				selected = append(selected, line)
			}
		}

		return strings.Join(selected, "\n")
	}

	cleaned := text

	for strings.HasPrefix(cleaned, "```") {
		newlineIndex := strings.Index(cleaned, "\n")
		if newlineIndex < 0 {
			break
		}

		cleaned = strings.TrimSpace(
			cleaned[newlineIndex+1:],
		)
	}

	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimSuffix(cleaned, "```")

	return strings.TrimSpace(cleaned)
}

// PurifyCoursewareAnchorAOCI 将模型输出收敛为课程锚点。
func PurifyCoursewareAnchorAOCI(
	source *models.ImageAOCI,
) (*models.ImageAOCI, error) {
	if source == nil {
		return nil, fmt.Errorf(
			"课程锚点IAOCI对象为空",
		)
	}

	purified := *source
	purified.Relations = nil
	purified.RawText = ""

	purified.ImageKey = imageAOCIAnchorKey
	purified.IndexVersion = 1
	purified.IndexType =
		models.CWImageIndexTypeAnchor
	purified.UsageRole =
		models.CWImageUsageBackground
	purified.ContinuityLevel = 0
	purified.AspectRatio =
		models.CWImageAspectFlexible
	purified.RelationCount = "0"

	purified.FocusText =
		"定义本课件统一艺术语言和可复用的主要角色或标志性主体"

	purified.LayoutText =
		"Ø；课程锚点不锁定具体构图、镜头、景别、主体位置和留白"

	purified.SceneText =
		"Ø；锚点原图中的教室、课桌、黑板、家具、背景、镜头和道具位置均不继承"

	purified.ExportText =
		"统一渲染质量；具体尺寸和画幅由每张页面图片自己的槽位决定"

	purified.ArtText =
		purifyAnchorArtText(
			purified.ArtText,
		)

	if isEmptyImageAOCISemantic(
		purified.ArtText,
	) {
		return nil, fmt.Errorf(
			"锚点未提取到可用[A]艺术风格",
		)
	}

	if isEmptyImageAOCISemantic(
		purified.CharacterText,
	) {
		purified.CharacterText = "Ø"
		purified.SubjectType =
			models.CWImageSubjectNone
	} else {
		purified.SubjectType =
			inferAnchorSubjectType(
				purified.CharacterText,
				purified.SubjectType,
			)
	}

	requiredNegative :=
		"禁止继承锚点环境、家具、构图、镜头、景别和道具位置；" +
			"禁止把固定角色强行加入不需要人物的页面"

	purified.NegativeText =
		mergeImageAOCISemantics(
			purified.NegativeText,
			requiredNegative,
		)

	formatted, err :=
		FormatImageAOCI(&purified)
	if err != nil {
		return nil, fmt.Errorf(
			"锚点提纯后校验失败: %w",
			err,
		)
	}

	parsed, err := ParseImageAOCI(formatted)
	if err != nil {
		return nil, err
	}

	return parsed, nil
}

func inferAnchorSubjectType(
	characterText string,
	current string,
) string {
	hasPerson := strings.Contains(
		characterText,
		"C",
	)
	hasAnimal := strings.Contains(
		characterText,
		"A",
	)
	hasObject := strings.Contains(
		characterText,
		"O",
	)

	total := 0
	if hasPerson {
		total++
	}
	if hasAnimal {
		total++
	}
	if hasObject {
		total++
	}

	switch {
	case total >= 2:
		return models.CWImageSubjectMixed
	case hasPerson:
		return models.CWImageSubjectPerson
	case hasAnimal:
		return models.CWImageSubjectAnimal
	case hasObject:
		return models.CWImageSubjectObject
	case models.IsValidCWImageSubjectType(current):
		return current
	default:
		return models.CWImageSubjectNone
	}
}

func purifyAnchorArtText(value string) string {
	value = normalizedImageAOCISemantic(value)

	if isEmptyImageAOCISemantic(value) {
		return ""
	}

	forbiddenTokens := []string{
		"教室",
		"课桌",
		"黑板",
		"讲台",
		"书桌",
		"家具",
		"固定机位",
		"镜头",
		"构图",
		"景别",
		"前景",
		"中景",
		"后景",
		"主体位于",
		"位于左侧",
		"位于右侧",
		"具体场景",
		"背景环境",
	}

	clauses := strings.FieldsFunc(
		value,
		func(code rune) bool {
			switch code {
			case ';', '；', '。', '\n':
				return true
			default:
				return false
			}
		},
	)

	kept := make([]string, 0, len(clauses))

	for _, clause := range clauses {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}

		forbidden := false

		for _, token := range forbiddenTokens {
			if strings.Contains(clause, token) {
				forbidden = true
				break
			}
		}

		if !forbidden {
			kept = append(kept, clause)
		}
	}

	return strings.Join(kept, "；")
}

func mergeImageAOCISemantics(
	current string,
	required string,
) string {
	current = strings.TrimSpace(current)
	required = strings.TrimSpace(required)

	if isEmptyImageAOCISemantic(current) {
		return required
	}
	if required == "" ||
		strings.Contains(current, required) {
		return current
	}

	return current + "；" + required
}

func isEmptyImageAOCISemantic(
	value string,
) bool {
	value = strings.TrimSpace(value)

	switch strings.ToLower(value) {
	case "",
		"ø",
		"无",
		"none",
		"无具体角色",
		"无角色",
		"无人物",
		"无具体人物":
		return true
	default:
		return false
	}
}
