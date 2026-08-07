package services

// courseware_image_iaoci_planner_helpers.go — 图片IAOCI规划通用纯函数
//
// 本文件负责：
//   1. 校验AI生成的图片IAOCI是否与真实槽位一一对应；
//   2. 构造图片IAOCI规划输入；
//   3. 解析多条IAOCI文本；
//   4. 把IAOCI编译成最终生图提示词；
//   5. 安全读取新旧课程锚点；
//   6. 判断锚点实体和图片关系；
//   7. 根据画幅编码选择图片尺寸。
//
// HTML占位识别和填图函数位于：
// courseware_image_iaoci_placeholder.go。

import (
	"fmt"
	"regexp"
	"strings"

	"tedna/internal/models"
	"tedna/internal/utils"
)

// C1表示固定人物，A1表示固定动物，O1表示固定物体。
var cwImageEntityCodeRe = regexp.MustCompile(`\b[CAO][0-9]+\b`)

// cwValidatePlannedImageAOCIs 校验IAOCI计划与真实图片槽位严格对应。
func cwValidatePlannedImageAOCIs(
	parsedAOCIs []*models.ImageAOCI,
	slots []cwImagePlaceholderSlot,
	expectedOrder map[string]int,
	expectedSlot map[string]cwImagePlaceholderSlot,
	historyKeySet map[string]bool,
) error {
	if len(parsedAOCIs) != len(slots) {
		return fmt.Errorf(
			"图片IAOCI数量与真实占位不一致：占位=%d，IAOCI=%d",
			len(slots),
			len(parsedAOCIs),
		)
	}

	seenKeys := make(
		map[string]bool,
		len(parsedAOCIs),
	)

	for _, imageAOCI := range parsedAOCIs {
		if imageAOCI == nil {
			return fmt.Errorf(
				"图片IAOCI中存在空对象",
			)
		}

		if imageAOCI.IndexType !=
			models.CWImageIndexTypeImage {
			return fmt.Errorf(
				"页面图片IAOCI的IT必须为I：%s",
				imageAOCI.ImageKey,
			)
		}

		slot, exists :=
			expectedSlot[imageAOCI.ImageKey]
		if !exists {
			return fmt.Errorf(
				"AI返回了未分配的image_key：%s",
				imageAOCI.ImageKey,
			)
		}

		if seenKeys[imageAOCI.ImageKey] {
			return fmt.Errorf(
				"AI重复返回image_key：%s",
				imageAOCI.ImageKey,
			)
		}

		seenKeys[imageAOCI.ImageKey] = true

		for _, relation := range imageAOCI.Relations {
			targetOrder, samePage :=
				expectedOrder[relation.TargetImageKey]

			// 同页图片只能引用更早的槽位，
			// 保证按顺序生成时参考图已经存在。
			if samePage &&
				targetOrder >= slot.Order {
				return fmt.Errorf(
					"同页R关系只能指向更早槽位：%s -> %s",
					imageAOCI.ImageKey,
					relation.TargetImageKey,
				)
			}

			// 跨页目标必须是已经生成成功的历史图片。
			if !samePage &&
				!historyKeySet[relation.TargetImageKey] {
				return fmt.Errorf(
					"R关系目标不在允许引用范围：%s",
					relation.TargetImageKey,
				)
			}
		}
	}

	for _, slot := range slots {
		if !seenKeys[slot.ImageKey] {
			return fmt.Errorf(
				"AI漏掉图片槽位：%s",
				slot.ImageKey,
			)
		}
	}

	return nil
}

// cwBuildImageIAOCIPlanInput 构造图片IAOCI规划用户输入。
func cwBuildImageIAOCIPlanInput(
	courseware *models.Courseware,
	page *models.CoursewarePage,
	slots []cwImagePlaceholderSlot,
	anchor *models.ImageAOCI,
	history []*models.CoursewareImageIndex,
) string {
	var builder strings.Builder

	builder.WriteString("## 课件信息\n")
	builder.WriteString(
		fmt.Sprintf(
			"- 标题：%s\n- 学科：%s\n- 年级：%s\n",
			courseware.Title,
			courseware.Subject,
			courseware.Grade,
		),
	)

	builder.WriteString("\n## 本页教学事实\n")
	builder.WriteString(
		fmt.Sprintf(
			"- 页码：%d\n- 页面标题：%s\n- 教学目的：%s\n- 内容摘要：%s\n- 媒体需求：%s\n- 视觉形式：%s\n",
			page.PageNumber,
			page.Title,
			page.Purpose,
			page.ContentSummary,
			page.MediaRequirements,
			page.VisualFormat,
		),
	)

	builder.WriteString("\n## 本页真实图片槽位\n")

	for _, slot := range slots {
		builder.WriteString(
			fmt.Sprintf(
				"%d. placeholder_id=%s | image_key=%s | data_desc=%s\n",
				slot.Order,
				slot.PlaceholderID,
				slot.ImageKey,
				slot.Description,
			),
		)
	}

	if anchor != nil {
		builder.WriteString(
			"\n## 课程锚点允许继承的内容\n",
		)
		builder.WriteString(
			"锚点只控制艺术风格，以及本图确实出现相同实体时的固定主体身份。\n",
		)
		builder.WriteString(
			fmt.Sprintf(
				"- 锚点[A]：%s\n- 锚点[C]：%s\n",
				anchor.ArtText,
				anchor.CharacterText,
			),
		)
		builder.WriteString(
			"- 严禁继承锚点[L]、[S]、背景、教室、课桌、家具、镜头和构图。\n",
		)
	}

	historyLines := make([]string, 0, 12)

	for index := len(history) - 1; index >= 0 && len(historyLines) < 12; index-- {
		item := history[index]

		if item == nil ||
			item.PageID == nil ||
			*item.PageID == page.ID ||
			item.IndexType !=
				models.CWImageIndexTypeImage ||
			item.Status !=
				models.CWImageIndexStatusGenerated {
			continue
		}

		historyLines = append(
			historyLines,
			fmt.Sprintf(
				"- %s | F:%s | C:%s | S:%s",
				item.ImageKey,
				item.FocusText,
				item.CharacterText,
				item.SceneText,
			),
		)
	}

	if len(historyLines) > 0 {
		builder.WriteString(
			"\n## 可被R引用的已生成历史图片\n",
		)

		for index := len(historyLines) - 1; index >= 0; index-- {
			builder.WriteString(
				historyLines[index] + "\n",
			)
		}
	}

	builder.WriteString(
		"\n## 页面HTML上下文\n```html\n",
	)
	builder.WriteString(
		cwExtractPageBodyHTML(
			page.HTMLContent,
		),
	)
	builder.WriteString("\n```\n")

	builder.WriteString(
		"\n必须为每个槽位输出一条IAOCI，数量和image_key必须精确匹配上面的槽位列表。每张图独立遵守自己的data_desc和本页教学事实。",
	)

	return builder.String()
}

// cwParseImageAOCIBlocks 解析AI返回的多条IAOCI。
func cwParseImageAOCIBlocks(
	raw string,
) ([]*models.ImageAOCI, error) {
	raw = strings.ReplaceAll(
		raw,
		"\r\n",
		"\n",
	)
	raw = strings.ReplaceAll(
		raw,
		"\r",
		"\n",
	)

	lines := strings.Split(raw, "\n")
	blocks := make([]string, 0)
	current := make([]string, 0)

	flush := func() {
		if len(current) == 0 {
			return
		}

		blocks = append(
			blocks,
			strings.Join(current, "\n"),
		)

		current = make([]string, 0)
	}

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		if line == "" ||
			strings.HasPrefix(line, "```") {
			continue
		}

		if strings.HasPrefix(line, "IK:") {
			flush()
			current = append(current, line)
			continue
		}

		if len(current) > 0 {
			current = append(current, line)
		}
	}

	flush()

	if len(blocks) == 0 {
		return nil, fmt.Errorf(
			"AI输出中没有找到IAOCI块",
		)
	}

	result := make(
		[]*models.ImageAOCI,
		0,
		len(blocks),
	)

	for index, block := range blocks {
		parsed, err :=
			utils.ParseImageAOCI(block)
		if err != nil {
			return nil, fmt.Errorf(
				"第%d条图片IAOCI无效: %w",
				index+1,
				err,
			)
		}

		result = append(result, parsed)
	}

	return result, nil
}

// cwCompileImageGenerationPrompt 将IAOCI编译为最终生图提示词。
func cwCompileImageGenerationPrompt(
	imageAOCI *models.ImageAOCI,
	anchorAOCI *models.ImageAOCI,
) string {
	if imageAOCI == nil {
		return ""
	}

	var builder strings.Builder

	builder.WriteString(
		"【本图最高优先级教学内容】",
	)
	builder.WriteString(imageAOCI.FocusText)

	builder.WriteString("\n【本图独立构图与镜头】")
	builder.WriteString(imageAOCI.LayoutText)

	builder.WriteString("\n【本图独立场景】")
	builder.WriteString(imageAOCI.SceneText)

	styleText := imageAOCI.ArtText

	if anchorAOCI != nil &&
		!cwIsEmptyIAOCISemantic(
			anchorAOCI.ArtText,
		) {
		// 锚点只能替换艺术风格，不能替换本图内容、场景和构图。
		styleText = anchorAOCI.ArtText
	}

	builder.WriteString(
		"\n【艺术风格，只控制如何绘制，不控制画什么或在哪里】",
	)
	builder.WriteString(styleText)

	builder.WriteString("\n【本图主体】")
	builder.WriteString(imageAOCI.CharacterText)

	if anchorAOCI != nil &&
		cwImageAOCIUsesAnchorSubject(
			imageAOCI,
			anchorAOCI,
		) {
		builder.WriteString(
			"\n【需要保持一致的锚点固定主体身份】",
		)
		builder.WriteString(
			anchorAOCI.CharacterText,
		)
	}

	if len(imageAOCI.Relations) > 0 {
		builder.WriteString("\n【图片关系R】")

		for _, relation := range imageAOCI.Relations {
			builder.WriteString(
				fmt.Sprintf(
					"\n%s %s，仅继承[%s]：%s",
					relation.RelationCode,
					relation.TargetImageKey,
					relation.InheritMask,
					relation.SemanticNote,
				),
			)
		}
	}

	builder.WriteString("\n【输出规格】")
	builder.WriteString(imageAOCI.ExportText)

	builder.WriteString("\n【禁止项】")
	builder.WriteString(imageAOCI.NegativeText)
	builder.WriteString(
		"；禁止照搬课程锚点原图的环境、家具、构图、镜头和主体位置；页面内容和本图构图始终优先于锚点。",
	)

	return builder.String()
}

// cwParseCoursewareAnchorAOCI 安全读取课程锚点。
//
// 新IAOCI格式严格解析。
// 旧单行VAOCI仅兼容A艺术风格，不继承角色、场景或构图。
func cwParseCoursewareAnchorAOCI(
	courseware *models.Courseware,
) *models.ImageAOCI {
	if courseware == nil ||
		strings.TrimSpace(
			courseware.StyleAnchorVAOCI,
		) == "" {
		return nil
	}

	parsed, err := utils.ParseImageAOCI(
		courseware.StyleAnchorVAOCI,
	)

	if err == nil &&
		parsed.IndexType ==
			models.CWImageIndexTypeAnchor {
		return parsed
	}

	styleText :=
		cwExtractVAOCIStyleSection(
			courseware.StyleAnchorVAOCI,
		)

	if strings.TrimSpace(styleText) == "" {
		return nil
	}

	return &models.ImageAOCI{
		ImageKey:        "@ANCHOR",
		IndexVersion:    1,
		IndexType:       models.CWImageIndexTypeAnchor,
		UsageRole:       models.CWImageUsageBackground,
		ContinuityLevel: 0,
		SubjectType:     models.CWImageSubjectNone,
		AspectRatio:     models.CWImageAspectFlexible,
		RelationCount:   "0",
		FocusText:       "存量锚点艺术风格兼容",
		LayoutText:      "Ø",
		ArtText:         styleText,
		CharacterText:   "Ø",
		SceneText:       "Ø",
		ExportText:      "统一渲染质量",
		NegativeText:    "禁止继承存量锚点环境、构图、镜头和角色",
		Relations:       nil,
	}
}

// cwImageAOCIUsesAnchorSubject 判断本图是否明确引用锚点实体。
func cwImageAOCIUsesAnchorSubject(
	imageAOCI *models.ImageAOCI,
	anchorAOCI *models.ImageAOCI,
) bool {
	if imageAOCI == nil ||
		anchorAOCI == nil {
		return false
	}

	imageCodes :=
		cwImageEntityCodeSet(
			imageAOCI.CharacterText,
		)

	if len(imageCodes) == 0 {
		return false
	}

	anchorCodes :=
		cwImageEntityCodeSet(
			anchorAOCI.CharacterText,
		)

	for code := range imageCodes {
		if anchorCodes[code] {
			return true
		}
	}

	return false
}

func cwImageEntityCodeSet(
	value string,
) map[string]bool {
	result := make(map[string]bool)

	codes := cwImageEntityCodeRe.FindAllString(
		strings.ToUpper(value),
		-1,
	)

	for _, code := range codes {
		result[code] = true
	}

	return result
}

// cwGeneratedImageHistoryKeySet 提取可被跨页R引用的历史图片键。
func cwGeneratedImageHistoryKeySet(
	indexes []*models.CoursewareImageIndex,
	currentPageID string,
) map[string]bool {
	result := make(map[string]bool)

	for _, item := range indexes {
		if item == nil ||
			item.PageID == nil ||
			*item.PageID == currentPageID ||
			item.Status !=
				models.CWImageIndexStatusGenerated ||
			item.IndexType !=
				models.CWImageIndexTypeImage {
			continue
		}

		result[item.ImageKey] = true
	}

	return result
}

// cwImageAOCISize 将画幅编码转换为图片API尺寸。
func cwImageAOCISize(
	imageAOCI *models.ImageAOCI,
) string {
	if imageAOCI == nil {
		return "2560x1440"
	}

	switch imageAOCI.AspectRatio {
	case models.CWImageAspectVertical:
		return "1440x2560"

	case models.CWImageAspectSquare:
		return "1920x1920"

	case models.CWImageAspectHorizontal:
		return "2560x1440"

	default:
		return "2560x1440"
	}
}

// cwIsEmptyIAOCISemantic 判断IAOCI语义是否为空。
func cwIsEmptyIAOCISemantic(
	value string,
) bool {
	value = strings.TrimSpace(value)

	switch strings.ToLower(value) {
	case "", "ø", "无", "none":
		return true
	default:
		return false
	}
}
