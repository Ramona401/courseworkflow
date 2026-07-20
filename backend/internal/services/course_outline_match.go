package services

// course_outline_match.go — 课程大纲与教案的「学段范围相交 + 教材版本」匹配（备课注入用）
//
// ============================== 背景与演进 ==============================
//
// 教案 lesson_plans 只有 subject + grade。历史上 grade 存的是"小学低段/小学中段/七年级"
// 这类学段或单年级写法；大纲 course_outlines 的 grade 则可能是"一年级/六年级/小学一至六年级"
// 等任意写法。
//
// v1（已废弃）：用"大纲集合 ⊇ 教案集合"（spanCovers）判定命中，且只取最贴合一份。
//   在真实数据上大面积失败：语文大纲按单年级分册录入（grade={6}），而教案是学段写法
//   （grade={3,4}），"大纲 ⊇ 教案"永不成立 → 大纲从不注入。
//
// v2（学段相交·多份全注入，Yuhan 决策）：
//   判定改为「年级集合相交即命中」，并把所有相交命中的大纲全部注入。
//
// v3（教材版本，本次 Yuhan 决策）：一标多本，同学科同年级同册次可能有人教版/北师大版/
//   统编版等多套大纲。改为「老师在备课首屏显式选定教材版本」，注入时按版本精确过滤：
//     · 严格只注入 publisher == 选定版本 的大纲；
//     · 绝不做任何跨版本兜底（不拿人教版兜底、也不拿通用版兜底）——不同版本教材单元结构、
//       篇目、课时完全不同，跨版本注入是「错的资料」，比不注入更糟。对不上就不注入。
//     · "通用/不限版本"本身是一个独立的版本值（publisher 空串）；老师选"通用"时，
//       也只注入 publisher 为空串的大纲，不与任何具名版本互相兜底。
//   版本选择落点：备课首屏的教材版本选择器（见 ListAvailablePublishers）。没选版本=不注入。
//
// 文案（硬指令）：BuildCourseOutlinesContext 明确告知 AI 这份大纲已注入、是权威最新版、
//   也正是老师口中的"备课资料"，必须优先据此回答篇目/单元/课时等事实，绝不能说"读不到资料"
//   或用旧记忆硬猜。
//
// ============================== 兼容性说明 ==============================
//
//   - MatchBestOutline / BuildCourseOutlineContext（单份，旧签名）保留，供 unit_plan_service.go
//     等旧调用方使用（取相交命中里"最贴合"的一份，不涉及版本，行为安全）。
//   - MatchOutlines（复数，仅学段相交、不过滤版本）保留：供 ListAvailablePublishers 汇总
//     "该学科年级有哪些版本"，以及任何只需学段相交的场景。
//   - 新增 MatchOutlinesByPublisher（学段相交 + 版本精确过滤）：供备课工坊注入按选定版本取大纲。
//
// 原则（Yuhan 决策）：宁缺不错——一份都没相交、或选定版本下无大纲，就不注入，这是正常状态。

import (
	"strings"

	"tedna/internal/models"
)

// gradeSpan 年级覆盖范围：1-12 年级编号集合（小学1-6 初中7-9 高中10-12）
type gradeSpan map[int]struct{}

// normalizeGradeToSpan 把年级/学段写法归一化成"覆盖的年级编号集合"
//
// 支持的写法示例：
//
//	"一年级"/"6年级"/"七年级"/"初一"/"高一"  → 单点 {1}/{6}/{7}/{7}/{10}
//	"小学低段"/"小学中段"/"小学高段"          → {1,2}/{3,4}/{5,6}
//	"小学"/"初中"/"高中"（仅段名）            → {1..6}/{7,8,9}/{10,11,12}
//	"小学一至六年级"/"中学七年级到十二年级"    → {1..6}/{7..12}（含"到"字写法）
//	"全册"/"通用"/"不限"/"全学段"             → {1..12}
func normalizeGradeToSpan(raw string) gradeSpan {
	s := strings.TrimSpace(raw)
	span := gradeSpan{}
	if s == "" {
		return span
	}

	add := func(from, to int) {
		for i := from; i <= to; i++ {
			span[i] = struct{}{}
		}
	}

	if strings.Contains(s, "全册") ||
		strings.Contains(s, "不限") ||
		strings.Contains(s, "通用") ||
		strings.Contains(s, "全学段") {
		add(1, 12)
		return span
	}

	hasPrimary := strings.Contains(s, "小学")
	hasJunior := strings.Contains(s, "初中")
	hasSenior := strings.Contains(s, "高中")
	low := strings.Contains(s, "低段") ||
		strings.Contains(s, "低年级")
	mid := strings.Contains(s, "中段") ||
		strings.Contains(s, "中年级")
	high := strings.Contains(s, "高段") ||
		strings.Contains(s, "高年级")

	if hasPrimary {
		switch {
		case low:
			add(1, 2)
		case mid:
			add(3, 4)
		case high:
			add(5, 6)
		default:
			add(1, 6)
		}
	}

	if hasJunior {
		add(7, 9)
	}

	if hasSenior {
		add(10, 12)
	}

	if !hasPrimary &&
		(strings.Contains(s, "一至六") ||
			strings.Contains(s, "1至6")) {
		add(1, 6)
	}

	if strings.Contains(s, "七至九") ||
		strings.Contains(s, "7至9") {
		add(7, 9)
	}

	if strings.Contains(s, "七年级到十二") ||
		strings.Contains(s, "七到十二") ||
		strings.Contains(s, "7到12") {
		add(7, 12)
	}

	gradeWords := []struct {
		keys []string
		num  int
	}{
		{[]string{"一年级", "1年级"}, 1},
		{[]string{"二年级", "2年级"}, 2},
		{[]string{"三年级", "3年级"}, 3},
		{[]string{"四年级", "4年级"}, 4},
		{[]string{"五年级", "5年级"}, 5},
		{[]string{"六年级", "6年级"}, 6},
		{[]string{"七年级", "7年级", "初一"}, 7},
		{[]string{"八年级", "8年级", "初二"}, 8},
		{[]string{"九年级", "9年级", "初三"}, 9},
		{[]string{"高一"}, 10},
		{[]string{"高二"}, 11},
		{[]string{"高三"}, 12},
	}

	for _, gradeWord := range gradeWords {
		for _, key := range gradeWord.keys {
			if strings.Contains(s, key) {
				span[gradeWord.num] = struct{}{}
				break
			}
		}
	}

	return span
}

// spansIntersect 两个年级集合是否有交集（任一相同年级即相交）。
func spansIntersect(a, b gradeSpan) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}

	small, large := a, b
	if len(b) < len(a) {
		small, large = b, a
	}

	for grade := range small {
		if _, exists := large[grade]; exists {
			return true
		}
	}

	return false
}

// intersectionSize 两个年级集合的交集大小。
func intersectionSize(a, b gradeSpan) int {
	small, large := a, b
	if len(b) < len(a) {
		small, large = b, a
	}

	count := 0
	for grade := range small {
		if _, exists := large[grade]; exists {
			count++
		}
	}

	return count
}

// normalizeCourseOutlineLevelLabel
// 规范化职教、成教等非K12学习层级文本。
func normalizeCourseOutlineLevelLabel(
	raw string,
) string {
	return strings.ToLower(
		strings.Join(
			strings.Fields(
				strings.TrimSpace(raw),
			),
			"",
		),
	)
}

// courseOutlineGradesMatch
// K12采用年级集合相交；无法解析为K12年级时采用完整文本精确匹配。
func courseOutlineGradesMatch(
	outlineGradeRaw string,
	planGradeRaw string,
) bool {
	outlineSpan :=
		normalizeGradeToSpan(
			outlineGradeRaw,
		)
	planSpan :=
		normalizeGradeToSpan(
			planGradeRaw,
		)

	if len(outlineSpan) > 0 &&
		len(planSpan) > 0 {
		return spansIntersect(
			outlineSpan,
			planSpan,
		)
	}

	outlineLabel :=
		normalizeCourseOutlineLevelLabel(
			outlineGradeRaw,
		)
	planLabel :=
		normalizeCourseOutlineLevelLabel(
			planGradeRaw,
		)

	return outlineLabel != "" &&
		outlineLabel == planLabel
}

// MatchOutlines 返回所有学习层级匹配的大纲，不过滤出版社。
func MatchOutlines(
	planGradeRaw string,
	candidates []*models.CourseOutline,
) []*models.CourseOutline {
	hits := make(
		[]*models.CourseOutline,
		0,
	)

	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}

		if courseOutlineGradesMatch(
			candidate.Grade,
			planGradeRaw,
		) {
			hits = append(
				hits,
				candidate,
			)
		}
	}

	return hits
}

// MatchOutlinesByPublisher
// 返回学习层级匹配且出版社与选择值严格相等的大纲。
func MatchOutlinesByPublisher(
	planGradeRaw string,
	selectedPublisher string,
	candidates []*models.CourseOutline,
) []*models.CourseOutline {
	want := strings.TrimSpace(
		selectedPublisher,
	)
	hits := make(
		[]*models.CourseOutline,
		0,
	)

	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}

		if strings.TrimSpace(
			candidate.Publisher,
		) != want {
			continue
		}

		if courseOutlineGradesMatch(
			candidate.Grade,
			planGradeRaw,
		) {
			hits = append(
				hits,
				candidate,
			)
		}
	}

	return hits
}

// BuildCourseOutlinesContext 把多份命中大纲拼成一个注入上下文块。
func BuildCourseOutlinesContext(
	outlines []*models.CourseOutline,
) string {
	valid := make(
		[]*models.CourseOutline,
		0,
		len(outlines),
	)

	for _, outline := range outlines {
		if outline != nil &&
			strings.TrimSpace(
				outline.Content,
			) != "" {
			valid = append(
				valid,
				outline,
			)
		}
	}

	if len(valid) == 0 {
		return ""
	}

	var builder strings.Builder

	builder.WriteString(
		"\n\n【系统已注入·权威课程大纲（必须优先采信）】\n",
	)
	builder.WriteString(
		"下面是系统已经为你注入到本对话中的课程大纲全文，这就是老师所说的“备课资料 / 课程大纲”——",
	)
	builder.WriteString(
		"你此刻已经完整拥有它的全部内容，绝对不要再说“我读不到您上传的资料”“无法读取外部附件”“我的知识库是旧版教材”这类话，那是错误的。\n",
	)
	builder.WriteString(
		"使用要求（务必遵守）：\n",
	)
	builder.WriteString(
		"1. 这份大纲是当前最新、最权威的依据。凡涉及本学科本年级的【课文篇目、单元归属、单元顺序、课时安排、教学要点】等事实，必须以下面这份大纲为准，绝不能用你训练记忆里的旧版教材目录去回答或推测。\n",
	)
	builder.WriteString(
		"2. 当老师提到“备课资料”“大纲”“课程大纲”里写了什么时，指的就是下面这份，请直接到大纲原文里查找并据实回答，不要反问老师“能否把资料发给我”。\n",
	)
	builder.WriteString(
		"3. 若老师给的课题在大纲里能定位到，请据大纲确认其所属单元与篇目；若大纲里确实查不到该课题，可如实说明“在已注入的大纲中未找到该篇目，请老师补充确认”，但绝不得凭旧记忆硬猜篇目或单元编号。\n",
	)
	builder.WriteString(
		"4. 回答时可引用大纲中与本课直接相关的内容，不必逐字照搬整册大纲。\n",
	)

	if len(valid) > 1 {
		builder.WriteString(
			"（下面共有多份大纲，可能覆盖相邻年级或不同册次，请先据课题与年级判断本课最可能属于其中哪一份、哪个单元，再据此分析。）\n",
		)
	}

	for _, outline := range valid {
		builder.WriteString(
			"\n==== 大纲标题：" +
				outline.Title +
				" ====\n",
		)
		builder.WriteString(
			outline.Content,
		)
		builder.WriteString(
			"\n==== （以上为《" +
				outline.Title +
				"》全文结束） ====\n",
		)
	}

	builder.WriteString(
		"\n【权威课程大纲·结束】\n",
	)

	return builder.String()
}

// MatchBestOutline
// 从同学科候选中选择学习层级最贴合的一份大纲。
func MatchBestOutline(
	planGradeRaw string,
	candidates []*models.CourseOutline,
) *models.CourseOutline {
	planSpan :=
		normalizeGradeToSpan(
			planGradeRaw,
		)

	var best *models.CourseOutline
	bestIntersection := 0
	bestWidth := 1 << 30

	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}

		outlineSpan :=
			normalizeGradeToSpan(
				candidate.Grade,
			)

		intersection := 0
		width := 1

		if len(planSpan) > 0 &&
			len(outlineSpan) > 0 {
			intersection =
				intersectionSize(
					outlineSpan,
					planSpan,
				)
			if intersection == 0 {
				continue
			}
			width = len(outlineSpan)
		} else {
			if !courseOutlineGradesMatch(
				candidate.Grade,
				planGradeRaw,
			) {
				continue
			}
			intersection = 1
		}

		if intersection > bestIntersection ||
			(intersection == bestIntersection &&
				width < bestWidth) {
			best = candidate
			bestIntersection = intersection
			bestWidth = width
		}
	}

	return best
}

// BuildCourseOutlineContext 把单份命中大纲拼成注入上下文块。
func BuildCourseOutlineContext(
	outline *models.CourseOutline,
) string {
	if outline == nil ||
		strings.TrimSpace(
			outline.Content,
		) == "" {
		return ""
	}

	return BuildCourseOutlinesContext(
		[]*models.CourseOutline{
			outline,
		},
	)
}
