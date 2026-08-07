package services

// course_outline_match.go — 课程大纲年级、学段与教材版本匹配
//
// 本文件提供课程大纲运行时共用的确定性匹配规则：
//   - K12具体年级或学段先转换为1至12年级集合，再判断是否相交；
//   - 明确具体年级的优先级高于“小学、初中、高中”等宽泛学段词；
//   - 多位年级优先于内部子串，十一年级不能被识别为一年级；
//   - “小学三年级”只能解析为{3}，不能错误扩大为{1..6}；
//   - “初中一年级”和“高中一年级”分别解析为{7}和{10}；
//   - 职教、成教和培训层级不转成K12数字集合，继续使用文本精确匹配；
//   - 出版社匹配保持严格相等，不进行跨出版社兜底。
//
// 自动候选和手动候选的区别不在本文件：
//   - 自动候选由仓储要求grade文本完全相等；
//   - 手动候选由服务层调用courseOutlineGradesMatch允许年级或学段相交。

import (
	"strings"

	"tedna/internal/models"
)

// gradeSpan 年级覆盖集合，编号1至12分别表示小学、初中和高中年级。
type gradeSpan map[int]struct{}

// containsAny 判断文本是否包含任意一个指定片段。
func containsAny(value string, parts ...string) bool {
	for _, part := range parts {
		if part != "" && strings.Contains(value, part) {
			return true
		}
	}

	return false
}

// normalizeCourseOutlineLevelLabel 对学习层级文本执行小写、去首尾空白和删除内部空白。
func normalizeCourseOutlineLevelLabel(raw string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(raw)), ""))
}

// isExplicitNonK12LevelLabel 识别不能使用K12年级数字推断的职教、成教和培训层级。
func isExplicitNonK12LevelLabel(normalized string) bool {
	return containsAny(
		normalized,
		"中职",
		"高职",
		"职教",
		"职业教育",
		"职业高中",
		"成人教育",
		"培训",
	)
}

// normalizeGradeToSpan 把K12具体年级或学段写法归一化成年级编号集合。
//
// 解析顺序非常重要：
//   1. 非K12标记；
//   2. 全学段与明确范围；
//   3. 小学低中高段；
//   4. 初中和高中内部年级；
//   5. 十二至一年级，按数字从大到小判断；
//   6. 纯小学、初中、高中或中学段名。
//
// 多位年级必须早于内部子串判断，防止“十一年级”匹配“一年级”。
func normalizeGradeToSpan(raw string) gradeSpan {
	normalized := normalizeCourseOutlineLevelLabel(raw)
	span := gradeSpan{}

	if normalized == "" || isExplicitNonK12LevelLabel(normalized) {
		return span
	}

	addRange := func(from, to int) {
		for grade := from; grade <= to; grade++ {
			span[grade] = struct{}{}
		}
	}

	setSingle := func(grade int) gradeSpan {
		span[grade] = struct{}{}
		return span
	}

	if containsAny(normalized, "全册", "不限", "通用", "全学段") {
		addRange(1, 12)
		return span
	}

	// 明确范围必须在具体年级之前判断。
	if containsAny(
		normalized,
		"七年级到十二年级",
		"七年级至十二年级",
		"七到十二",
		"七至十二",
		"7年级到12年级",
		"7年级至12年级",
		"7到12",
		"7至12",
		"7-12",
		"7—12",
	) {
		addRange(7, 12)
		return span
	}

	if containsAny(
		normalized,
		"一年级到六年级",
		"一年级至六年级",
		"一到六",
		"一至六",
		"1年级到6年级",
		"1年级至6年级",
		"1到6",
		"1至6",
		"1-6",
		"1—6",
	) {
		addRange(1, 6)
		return span
	}

	if containsAny(
		normalized,
		"七年级到九年级",
		"七年级至九年级",
		"七到九",
		"七至九",
		"7年级到9年级",
		"7年级至9年级",
		"7到9",
		"7至9",
		"7-9",
		"7—9",
	) {
		addRange(7, 9)
		return span
	}

	if containsAny(
		normalized,
		"十年级到十二年级",
		"十年级至十二年级",
		"十到十二",
		"十至十二",
		"10年级到12年级",
		"10年级至12年级",
		"10到12",
		"10至12",
		"10-12",
		"10—12",
	) {
		addRange(10, 12)
		return span
	}

	// 小学明确学段。
	if containsAny(normalized, "小学低段", "小学低年级") {
		addRange(1, 2)
		return span
	}

	if containsAny(normalized, "小学中段", "小学中年级") {
		addRange(3, 4)
		return span
	}

	if containsAny(normalized, "小学高段", "小学高年级") {
		addRange(5, 6)
		return span
	}

	// 初中和高中内部年级必须先于普通“一年级”等判断。
	switch {
	case containsAny(normalized, "初中一年级", "初中1年级", "初一"):
		return setSingle(7)
	case containsAny(normalized, "初中二年级", "初中2年级", "初二"):
		return setSingle(8)
	case containsAny(normalized, "初中三年级", "初中3年级", "初三"):
		return setSingle(9)
	case containsAny(normalized, "高中一年级", "高中1年级", "高一"):
		return setSingle(10)
	case containsAny(normalized, "高中二年级", "高中2年级", "高二"):
		return setSingle(11)
	case containsAny(normalized, "高中三年级", "高中3年级", "高三"):
		return setSingle(12)
	}

	// 从十二年级向一年级倒序判断，避免多位数字或中文数字被内部子串抢先命中。
	gradeWords := []struct {
		keys []string
		num  int
	}{
		{keys: []string{"十二年级", "12年级"}, num: 12},
		{keys: []string{"十一年级", "11年级"}, num: 11},
		{keys: []string{"十年级", "10年级"}, num: 10},
		{keys: []string{"九年级", "9年级"}, num: 9},
		{keys: []string{"八年级", "8年级"}, num: 8},
		{keys: []string{"七年级", "7年级"}, num: 7},
		{keys: []string{"六年级", "6年级"}, num: 6},
		{keys: []string{"五年级", "5年级"}, num: 5},
		{keys: []string{"四年级", "4年级"}, num: 4},
		{keys: []string{"三年级", "3年级"}, num: 3},
		{keys: []string{"二年级", "2年级"}, num: 2},
		{keys: []string{"一年级", "1年级"}, num: 1},
	}

	for _, gradeWord := range gradeWords {
		if containsAny(normalized, gradeWord.keys...) {
			return setSingle(gradeWord.num)
		}
	}

	// 只有没有具体年级或明确范围时，纯学段名称才扩展。
	switch {
	case strings.Contains(normalized, "小学"):
		addRange(1, 6)
	case strings.Contains(normalized, "初中"):
		addRange(7, 9)
	case strings.Contains(normalized, "高中"):
		addRange(10, 12)
	case strings.Contains(normalized, "中学"):
		addRange(7, 12)
	}

	return span
}

// spansIntersect 判断两个非空年级集合是否存在交集。
func spansIntersect(left gradeSpan, right gradeSpan) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}

	small, large := left, right
	if len(right) < len(left) {
		small, large = right, left
	}

	for grade := range small {
		if _, exists := large[grade]; exists {
			return true
		}
	}

	return false
}

// intersectionSize 返回两个年级集合的交集数量。
func intersectionSize(left gradeSpan, right gradeSpan) int {
	small, large := left, right
	if len(right) < len(left) {
		small, large = right, left
	}

	count := 0
	for grade := range small {
		if _, exists := large[grade]; exists {
			count++
		}
	}

	return count
}

// courseOutlineGradesMatch K12采用年级集合相交；无法安全解析为K12时采用规范化文本精确匹配。
func courseOutlineGradesMatch(outlineGradeRaw string, planGradeRaw string) bool {
	outlineSpan := normalizeGradeToSpan(outlineGradeRaw)
	planSpan := normalizeGradeToSpan(planGradeRaw)

	if len(outlineSpan) > 0 && len(planSpan) > 0 {
		return spansIntersect(outlineSpan, planSpan)
	}

	outlineLabel := normalizeCourseOutlineLevelLabel(outlineGradeRaw)
	planLabel := normalizeCourseOutlineLevelLabel(planGradeRaw)

	return outlineLabel != "" && outlineLabel == planLabel
}

// MatchOutlines 返回所有年级或学段匹配的大纲，不过滤出版社。
func MatchOutlines(planGradeRaw string, candidates []*models.CourseOutline) []*models.CourseOutline {
	hits := make([]*models.CourseOutline, 0)

	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}

		if courseOutlineGradesMatch(candidate.Grade, planGradeRaw) {
			hits = append(hits, candidate)
		}
	}

	return hits
}

// MatchOutlinesByPublisher 返回年级或学段匹配且出版社与选择值严格相等的大纲。
func MatchOutlinesByPublisher(
	planGradeRaw string,
	selectedPublisher string,
	candidates []*models.CourseOutline,
) []*models.CourseOutline {
	want := strings.TrimSpace(selectedPublisher)
	hits := make([]*models.CourseOutline, 0)

	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}

		if strings.TrimSpace(candidate.Publisher) != want {
			continue
		}

		if courseOutlineGradesMatch(candidate.Grade, planGradeRaw) {
			hits = append(hits, candidate)
		}
	}

	return hits
}

// BuildCourseOutlinesContext 把多份有效大纲拼成AI运行上下文。
func BuildCourseOutlinesContext(outlines []*models.CourseOutline) string {
	valid := make([]*models.CourseOutline, 0, len(outlines))

	for _, outline := range outlines {
		if outline != nil && strings.TrimSpace(outline.Content) != "" {
			valid = append(valid, outline)
		}
	}

	if len(valid) == 0 {
		return ""
	}

	var builder strings.Builder

	builder.WriteString("\n\n【系统已注入·权威课程大纲（必须优先采信）】\n")
	builder.WriteString("下面是系统已经注入到本对话中的课程大纲全文。这就是老师所说的“备课资料 / 课程大纲”，你已经拥有其内容。\n")
	builder.WriteString("使用要求：\n")
	builder.WriteString("1. 涉及课文篇目、单元归属、单元顺序、课时安排和教学要点时，必须优先依据下面的大纲，不得使用旧记忆猜测。\n")
	builder.WriteString("2. 老师询问大纲内容时，应直接依据已注入原文回答，不得声称无法读取资料。\n")
	builder.WriteString("3. 若在大纲中确实找不到课题，应明确说明未找到并请老师确认，不得虚构篇目或单元编号。\n")
	builder.WriteString("4. 可引用与当前课程直接相关的内容，不必复述整份大纲。\n")

	if len(valid) > 1 {
		builder.WriteString("下面存在多份相交大纲，请结合当前课题、年级和册次确定最相关内容。\n")
	}

	for _, outline := range valid {
		builder.WriteString("\n==== 大纲标题：" + outline.Title + " ====\n")
		builder.WriteString(outline.Content)
		builder.WriteString("\n==== 《" + outline.Title + "》全文结束 ====\n")
	}

	builder.WriteString("\n【权威课程大纲·结束】\n")
	return builder.String()
}

// MatchBestOutline 从同学科候选中选择年级或学段最贴合的一份大纲。
func MatchBestOutline(planGradeRaw string, candidates []*models.CourseOutline) *models.CourseOutline {
	planSpan := normalizeGradeToSpan(planGradeRaw)

	var best *models.CourseOutline
	bestIntersection := 0
	bestWidth := 1 << 30

	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}

		outlineSpan := normalizeGradeToSpan(candidate.Grade)
		intersection := 0
		width := 1

		if len(planSpan) > 0 && len(outlineSpan) > 0 {
			intersection = intersectionSize(outlineSpan, planSpan)
			if intersection == 0 {
				continue
			}
			width = len(outlineSpan)
		} else {
			if !courseOutlineGradesMatch(candidate.Grade, planGradeRaw) {
				continue
			}
			intersection = 1
		}

		if intersection > bestIntersection ||
			(intersection == bestIntersection && width < bestWidth) {
			best = candidate
			bestIntersection = intersection
			bestWidth = width
		}
	}

	return best
}

// BuildCourseOutlineContext 把唯一一份有效大纲拼成AI运行上下文。
func BuildCourseOutlineContext(outline *models.CourseOutline) string {
	if outline == nil || strings.TrimSpace(outline.Content) == "" {
		return ""
	}

	return BuildCourseOutlinesContext([]*models.CourseOutline{outline})
}
