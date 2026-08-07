package utils

// lesson_plan_section.go — 教案Markdown目录和段落范围的确定性解析工具。
//
// 本文件不访问数据库、不调用AI、不执行权限判断，只完成纯文本结构解析。
// Service层和Repository层共同使用本解析器，保证AI预览和最终应用使用完全一致的定位规则。
//
// 标题识别优先保证少误判：
//   - Markdown的“1. 教师展示图片”属于正文有序列表，不能进入目录。
//   - “1. 教学目标”等已知教案栏目可以进入目录。
//   - “1. 活动一：观察图片”等明显的活动、任务或环节标题可以进入目录。
//   - Markdown显式标题以及常见中文教案标题继续正常识别。

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"tedna/internal/models"
)

const (
	// LessonPlanFullDocumentHeading 是无可识别标题时使用的虚拟标题。
	// 旧导入教案和纯文本教案仍可通过该节点修改完整正文。
	LessonPlanFullDocumentHeading = "__FULL_DOCUMENT__"
)

var (
	markdownHeadingPattern = regexp.MustCompile(`^(#{1,6})[ \t]+(.+?)\s*$`)

	chinesePrimaryHeadingPattern = regexp.MustCompile(
		`^([一二三四五六七八九十百]+)[、.．][ \t]*(.+?)\s*$`,
	)
	chineseSecondaryHeadingPattern = regexp.MustCompile(
		`^[（(]([一二三四五六七八九十百]+)[）)][ \t]*(.+?)\s*$`,
	)

	// 数字编号与Markdown有序列表语法重叠，必须再执行语义判断。
	numberedHeadingPattern = regexp.MustCompile(
		`^(?:([0-9]+)[、.．]|[（(]([0-9]+)[）)])[ \t]*(.+?)\s*$`,
	)

	chapterHeadingPattern = regexp.MustCompile(
		`^第[一二三四五六七八九十百0-9]+[章节部分单元][ \t、:：.\-]*(.*?)\s*$`,
	)

	// 仅允许明显表示教案结构的编号标题进入目录。
	numberedHeadingSemanticPrefixPattern = regexp.MustCompile(
		`^(活动|任务|环节|步骤|问题|情境|案例|实验|练习|小结|作业|板书|评价|导入|新授|探究|合作|展示|交流|总结)[一二三四五六七八九十百0-9]*[：:\-—]?[ \t]*`,
	)
)

// 常见教案栏目允许在没有Markdown符号时进入目录。
// 仅做精确匹配，避免普通教学叙述被误判为标题。
var lessonPlanPlainHeadingTitles = map[string]struct{}{
	"教材分析":   {},
	"课标分析":   {},
	"课程标准":   {},
	"学情分析":   {},
	"教学内容":   {},
	"教学目标":   {},
	"学习目标":   {},
	"核心素养目标": {},
	"教学重点":   {},
	"教学难点":   {},
	"教学重难点":  {},
	"教学准备":   {},
	"教学方法":   {},
	"教学策略":   {},
	"教学过程":   {},
	"教学活动":   {},
	"教学环节":   {},
	"教师活动":   {},
	"学生活动":   {},
	"设计意图":   {},
	"时间分配":   {},
	"导入新课":   {},
	"新课导入":   {},
	"新授":     {},
	"新课讲授":   {},
	"探究活动":   {},
	"合作学习":   {},
	"课堂练习":   {},
	"巩固练习":   {},
	"课堂小结":   {},
	"总结提升":   {},
	"评价设计":   {},
	"作业设计":   {},
	"板书设计":   {},
	"教学反思":   {},
}

// lessonPlanTextLine 保存正文一行的UTF-8字节范围。
type lessonPlanTextLine struct {
	Start int
	End   int
	Next  int
	Text  string
	Trim  string
}

// lessonPlanHeading 保存构建目录前识别出的标题行。
type lessonPlanHeading struct {
	Line  lessonPlanTextLine
	Title string
	Level int
}

// ParseLessonPlanDocumentSections 将教案正文解析为目录节点和可替换正文范围。
//
// 范围规则：
//   - 标题行本身不属于可替换正文。
//   - 可替换正文从标题行后的第一个字节开始，到下一个任意标题之前结束。
//   - 父标题不会吞并子标题，避免修改父栏目时误删全部子栏目。
//   - 没有标题时返回覆盖全文的虚拟节点。
func ParseLessonPlanDocumentSections(content string) []models.LessonPlanDocumentSection {
	lines := splitLessonPlanTextLines(content)
	headings := make([]lessonPlanHeading, 0)

	for _, line := range lines {
		level, title, ok := parseLessonPlanHeadingLine(line.Trim)
		if !ok {
			continue
		}

		headings = append(headings, lessonPlanHeading{
			Line:  line,
			Title: title,
			Level: level,
		})
	}

	if len(headings) == 0 {
		return []models.LessonPlanDocumentSection{
			{
				ID:                 buildLessonPlanSectionID(LessonPlanFullDocumentHeading, 1),
				Title:              "教案正文",
				HeadingText:        LessonPlanFullDocumentHeading,
				Level:              1,
				HeadingPath:        []string{"教案正文"},
				Occurrence:         1,
				StartOffset:        0,
				ContentStartOffset: 0,
				EndOffset:          len(content),
				BodyMarkdown:       content,
				SectionHash:        HashLessonPlanSectionBody(content),
				Locator: models.LessonPlanSectionLocator{
					HeadingText: LessonPlanFullDocumentHeading,
					Occurrence:  1,
				},
			},
		}
	}

	type pathEntry struct {
		Level int
		Title string
	}

	pathStack := make([]pathEntry, 0)
	occurrenceByHeading := make(map[string]int)
	sections := make([]models.LessonPlanDocumentSection, 0, len(headings))

	for index, heading := range headings {
		for len(pathStack) > 0 && pathStack[len(pathStack)-1].Level >= heading.Level {
			pathStack = pathStack[:len(pathStack)-1]
		}

		headingPath := make([]string, 0, len(pathStack)+1)
		for _, entry := range pathStack {
			headingPath = append(headingPath, entry.Title)
		}
		headingPath = append(headingPath, heading.Title)

		pathStack = append(pathStack, pathEntry{
			Level: heading.Level,
			Title: heading.Title,
		})

		headingText := heading.Line.Trim
		occurrenceByHeading[headingText]++
		occurrence := occurrenceByHeading[headingText]

		endOffset := len(content)
		if index+1 < len(headings) {
			endOffset = headings[index+1].Line.Start
		}

		contentStartOffset := heading.Line.Next
		if contentStartOffset > endOffset {
			contentStartOffset = endOffset
		}

		body := content[contentStartOffset:endOffset]

		sections = append(sections, models.LessonPlanDocumentSection{
			ID:                 buildLessonPlanSectionID(headingText, occurrence),
			Title:              heading.Title,
			HeadingText:        headingText,
			Level:              heading.Level,
			HeadingPath:        headingPath,
			Occurrence:         occurrence,
			StartOffset:        heading.Line.Start,
			ContentStartOffset: contentStartOffset,
			EndOffset:          endOffset,
			BodyMarkdown:       body,
			SectionHash:        HashLessonPlanSectionBody(body),
			Locator: models.LessonPlanSectionLocator{
				HeadingText: headingText,
				Occurrence:  occurrence,
			},
		})
	}

	return sections
}

// FindLessonPlanDocumentSection 在数据库正式正文中重新解析并定位一个段落。
// 浏览器只能提交标题文本和出现序号，正文范围与哈希始终由后端重新计算。
func FindLessonPlanDocumentSection(
	content string,
	locator models.LessonPlanSectionLocator,
) (models.LessonPlanDocumentSection, bool) {
	headingText := strings.TrimSpace(locator.HeadingText)
	occurrence := locator.Occurrence
	if occurrence <= 0 {
		occurrence = 1
	}

	sections := ParseLessonPlanDocumentSections(content)
	for _, section := range sections {
		if section.HeadingText == headingText && section.Occurrence == occurrence {
			return section, true
		}
	}

	return models.LessonPlanDocumentSection{}, false
}

// HashLessonPlanSectionBody 计算段落直属正文的SHA-256哈希。
// 使用数据库原始UTF-8字节，不执行TrimSpace或换行归一化。
func HashLessonPlanSectionBody(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

// ReplaceLessonPlanDocumentSectionBody 只替换目标标题下方的直属正文。
// 标题行和其它章节保持逐字不变。
func ReplaceLessonPlanDocumentSectionBody(
	content string,
	section models.LessonPlanDocumentSection,
	replacement string,
) string {
	if section.ContentStartOffset < 0 ||
		section.EndOffset < section.ContentStartOffset ||
		section.EndOffset > len(content) {
		return content
	}

	prefix := content[:section.ContentStartOffset]
	suffix := content[section.EndOffset:]
	cleanReplacement := strings.TrimSpace(replacement)

	var inserted string
	switch {
	case cleanReplacement == "":
		inserted = ""
	case suffix != "":
		inserted = cleanReplacement + "\n\n"
	default:
		inserted = cleanReplacement + "\n"
	}

	return prefix + inserted + suffix
}

// splitLessonPlanTextLines 将正文拆成带精确字节范围的行。
// 不使用bufio.Scanner，避免超长表格、图片地址或导入文本触发长度上限。
func splitLessonPlanTextLines(content string) []lessonPlanTextLine {
	if content == "" {
		return []lessonPlanTextLine{
			{
				Start: 0,
				End:   0,
				Next:  0,
				Text:  "",
				Trim:  "",
			},
		}
	}

	lines := make([]lessonPlanTextLine, 0)
	start := 0

	for start < len(content) {
		relativeEnd := strings.IndexByte(content[start:], '\n')
		end := len(content)
		next := len(content)

		if relativeEnd >= 0 {
			end = start + relativeEnd
			next = end + 1
		}

		text := strings.TrimSuffix(content[start:end], "\r")
		lines = append(lines, lessonPlanTextLine{
			Start: start,
			End:   end,
			Next:  next,
			Text:  text,
			Trim:  strings.TrimSpace(text),
		})

		start = next
	}

	return lines
}

// parseLessonPlanHeadingLine 判断一行是否属于可进入目录的标题。
func parseLessonPlanHeadingLine(raw string) (level int, title string, ok bool) {
	line := strings.TrimSpace(raw)
	if line == "" {
		return 0, "", false
	}

	if match := markdownHeadingPattern.FindStringSubmatch(line); len(match) == 3 {
		title = cleanLessonPlanHeadingTitle(match[2])
		if title == "" {
			return 0, "", false
		}
		return len(match[1]), title, true
	}

	visibleLine := unwrapLessonPlanHeadingEmphasis(line)

	if match := chinesePrimaryHeadingPattern.FindStringSubmatch(visibleLine); len(match) == 3 {
		title = cleanLessonPlanHeadingTitle(match[2])
		return 2, title, title != ""
	}

	if match := chineseSecondaryHeadingPattern.FindStringSubmatch(visibleLine); len(match) == 3 {
		title = cleanLessonPlanHeadingTitle(match[2])
		return 3, title, title != ""
	}

	if match := numberedHeadingPattern.FindStringSubmatch(visibleLine); len(match) == 4 {
		title = cleanLessonPlanHeadingTitle(match[3])
		if !isLikelyLessonPlanNumberedHeadingTitle(title) {
			return 0, "", false
		}

		if match[1] != "" {
			return 2, title, true
		}
		return 3, title, true
	}

	if match := chapterHeadingPattern.FindStringSubmatch(visibleLine); len(match) == 2 {
		title = cleanLessonPlanHeadingTitle(match[1])
		if title == "" {
			title = cleanLessonPlanHeadingTitle(visibleLine)
		}
		return 1, title, title != ""
	}

	plainTitle := strings.TrimSpace(visibleLine)
	plainTitle = strings.TrimSuffix(plainTitle, "：")
	plainTitle = strings.TrimSuffix(plainTitle, ":")

	if _, exists := lessonPlanPlainHeadingTitles[plainTitle]; exists {
		return 2, plainTitle, true
	}

	return 0, "", false
}

// isLikelyLessonPlanNumberedHeadingTitle 判断数字编号后的文字是否像教案结构标题。
//
// 规则故意保守：
//   - 已知教案栏目直接接受。
//   - 活动、任务、环节等明显结构名接受。
//   - 带完整句结束标点或超过40字的内容拒绝。
//   - “教师展示图片”等普通编号步骤不会进入目录。
func isLikelyLessonPlanNumberedHeadingTitle(title string) bool {
	cleaned := cleanLessonPlanHeadingTitle(title)
	if cleaned == "" {
		return false
	}

	if _, exists := lessonPlanPlainHeadingTitles[cleaned]; exists {
		return true
	}

	if utf8.RuneCountInString(cleaned) > 40 {
		return false
	}

	if strings.ContainsAny(cleaned, "。！？；") {
		return false
	}

	return numberedHeadingSemanticPrefixPattern.MatchString(cleaned)
}

// unwrapLessonPlanHeadingEmphasis 去除整行粗体或下划线强调外壳。
func unwrapLessonPlanHeadingEmphasis(line string) string {
	trimmed := strings.TrimSpace(line)

	if len(trimmed) >= 4 &&
		strings.HasPrefix(trimmed, "**") &&
		strings.HasSuffix(trimmed, "**") {
		return strings.TrimSpace(trimmed[2 : len(trimmed)-2])
	}

	if len(trimmed) >= 4 &&
		strings.HasPrefix(trimmed, "__") &&
		strings.HasSuffix(trimmed, "__") {
		return strings.TrimSpace(trimmed[2 : len(trimmed)-2])
	}

	return trimmed
}

// cleanLessonPlanHeadingTitle 清理目录显示标题，不改变原始HeadingText定位值。
func cleanLessonPlanHeadingTitle(title string) string {
	cleaned := unwrapLessonPlanHeadingEmphasis(title)
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.TrimSuffix(cleaned, "：")
	cleaned = strings.TrimSuffix(cleaned, ":")

	if utf8.RuneCountInString(cleaned) > 120 {
		runes := []rune(cleaned)
		cleaned = string(runes[:120])
	}

	return strings.TrimSpace(cleaned)
}

// buildLessonPlanSectionID 生成确定性的前端节点ID。
// ID仅用于React键、DOM锚点和展示状态，不作为权限或写入依据。
func buildLessonPlanSectionID(headingText string, occurrence int) string {
	raw := headingText + "\x1f" + strconv.Itoa(occurrence)
	sum := sha256.Sum256([]byte(raw))
	return "lp-section-" + hex.EncodeToString(sum[:8])
}
