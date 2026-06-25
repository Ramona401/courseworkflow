package services

// course_outline_match.go — 课程大纲与教案的「学段范围相交」匹配（备课注入用）
//
// ============================== 背景与本次改版原因 ==============================
//
// 教案 lesson_plans 只有 subject + grade。历史上 grade 存的是"小学低段/小学中段/七年级"
// 这类学段或单年级写法；大纲 course_outlines 的 grade 则可能是"一年级/六年级/小学一至六年级"
// 等任意写法。
//
// 旧逻辑（v1）用"大纲集合 ⊇ 教案集合"（spanCovers）判定命中，并且只取最贴合的一份。
// 这在真实数据上大面积失败：
//   - 语文大纲是"按单个年级分册"录入的（一年级…六年级，每年级一份），grade 是单点集合，
//     例如"六年级" → {6}；
//   - 而语文教案的 grade 是学段写法，例如"小学中段" → {3,4}；
//   - "大纲 ⊇ 教案" 要求 {6} ⊇ {3,4}，永远不成立 → 大纲从不注入。
//   （人工智能学科恰好用"小学一至六年级"这种宽集合大纲，才偶然能覆盖教案单点而注入成功，
//    掩盖了这个 bug。）
//
// 本次改版（Yuhan 决策）：
//   1) 判定从"超集覆盖"改为「年级集合相交即命中」——只要大纲覆盖的年级与教案覆盖的年级
//      有任一交集，就算命中。这样：
//        · 新教案"一年级"{1} 与大纲"一年级"{1} 相交 → 命中；
//        · 老教案"小学低段"{1,2} 与大纲"一年级"{1}、"二年级"{2} 都相交 → 都命中；
//        · 教案"七年级"{7} 与任何小学语文大纲都不相交 → 不注入（正确）。
//   2) 存量学段教案（"小学低段/中段/高段"）会同时相交到多个年级的大纲，按 Yuhan 决策
//      「多份全注入」——返回所有相交命中的大纲并全部拼进上下文（含同年级上下册）。
//      代价是 token 偏多，但保证学段写法的老教案也能吃到对应年级的整册大纲。
//
// 文案升级（2026-06-22，硬指令）：线上实测发现大纲已成功注入 system prompt，但 AI 仍
//   按 training data 里的旧版教材记忆作答，还反复声称"读不到您上传的资料/附件"。根因是
//   旧注入文案太软（"供你把握…不要整段复述"），没让 AI 认领并优先采信这份大纲。因此把
//   BuildCourseOutlinesContext 的文案从"软背景"升级为"硬指令"，明确告知 AI：这就是它
//   已拥有的、权威的、最新版大纲，也正是老师口中的"备课资料"，必须优先据此回答篇目/单元/
//   课时等事实，绝不能说读不到、也不能用旧记忆硬猜。匹配逻辑未动，仅改注入文案。
//
// ============================== 兼容性说明 ==============================
//
//   - 旧函数 MatchBestOutline / BuildCourseOutlineContext 的签名完全保留，内部转调新逻辑，
//     供 unit_plan_service.go 等旧调用方继续使用（取相交命中里"最贴合"的一份，行为安全）。
//   - 新增 MatchOutlines（复数，返回全部相交命中）+ BuildCourseOutlinesContext（复数，
//     把多份大纲拼成一个上下文块），供备课工坊 analyze/design 注入「多份全注入」。
//
// 原则（Yuhan 决策）：宁缺不错——一份都没相交就不注入，这是正常状态不是缺陷。

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

	// 全学段
	if strings.Contains(s, "全册") || strings.Contains(s, "不限") || strings.Contains(s, "通用") || strings.Contains(s, "全学段") {
		add(1, 12)
		return span
	}

	hasPrimary := strings.Contains(s, "小学")
	hasJunior := strings.Contains(s, "初中")
	hasSenior := strings.Contains(s, "高中")
	low := strings.Contains(s, "低段") || strings.Contains(s, "低年级")
	mid := strings.Contains(s, "中段") || strings.Contains(s, "中年级")
	high := strings.Contains(s, "高段") || strings.Contains(s, "高年级")

	if hasPrimary {
		switch {
		case low:
			add(1, 2)
		case mid:
			add(3, 4)
		case high:
			add(5, 6)
		default:
			// 仅"小学"两字，或"小学一至六年级"这类：先铺满小学段，
			// 后面的"一至六/1至6"分支会再次确认，不冲突
			add(1, 6)
		}
	}
	if hasJunior {
		add(7, 9)
	}
	if hasSenior {
		add(10, 12)
	}
	// "一至六年级 / 1至6年级"（不含"小学"前缀时也兜住）
	if !hasPrimary && (strings.Contains(s, "一至六") || strings.Contains(s, "1至6")) {
		add(1, 6)
	}
	// "七至九 / 7至9"（初中段）
	if strings.Contains(s, "七至九") || strings.Contains(s, "7至9") {
		add(7, 9)
	}
	// "七年级到十二年级 / 7到12"（中学全段，覆盖"中学七年级到十二年级"这类大纲写法）
	if strings.Contains(s, "七年级到十二") || strings.Contains(s, "七到十二") || strings.Contains(s, "7到12") {
		add(7, 12)
	}

	// 单年级关键词（与上面的段级写法可叠加，互不冲突）
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
	for _, gw := range gradeWords {
		for _, k := range gw.keys {
			if strings.Contains(s, k) {
				span[gw.num] = struct{}{}
				break
			}
		}
	}
	return span
}

// spansIntersect 两个年级集合是否有交集（任一相同年级即相交）
//
// 这是本次改版的核心判据，取代旧的 spanCovers（超集覆盖）。
// 任一集合为空都视为不相交（教案没年级或大纲没年级，无从匹配）。
func spansIntersect(a, b gradeSpan) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	// 遍历较小的那个集合，降低比较次数
	small, large := a, b
	if len(b) < len(a) {
		small, large = b, a
	}
	for g := range small {
		if _, ok := large[g]; ok {
			return true
		}
	}
	return false
}

// intersectionSize 两个年级集合的交集大小（用于"最贴合"打分，越大越贴合）
func intersectionSize(a, b gradeSpan) int {
	small, large := a, b
	if len(b) < len(a) {
		small, large = b, a
	}
	cnt := 0
	for g := range small {
		if _, ok := large[g]; ok {
			cnt++
		}
	}
	return cnt
}

// ==================== 新逻辑：多份相交命中（备课工坊 analyze/design 用） ====================

// MatchOutlines 从同学科候选里挑出「年级集合与教案相交」的全部大纲
//
// 用于备课工坊 analyze/design 阶段的「多份全注入」（Yuhan 决策）：
//   - 教案"一年级"{1}：通常只相交到"一年级"上下册（如有两册则两份都命中）；
//   - 教案"小学低段"{1,2}：相交到一年级、二年级的所有册次大纲，全部返回；
//   - 一份都没相交 → 返回空切片（注入层据此静默跳过，正常单课备课）。
//
// candidates 由 repository.ListActiveOutlinesBySubject 提供，已按 updated_at 倒序，
// 本函数保持该相对顺序返回（注入时较新的大纲排在前面）。
func MatchOutlines(planGradeRaw string, candidates []*models.CourseOutline) []*models.CourseOutline {
	planSpan := normalizeGradeToSpan(planGradeRaw)
	if len(planSpan) == 0 {
		return nil
	}
	var hits []*models.CourseOutline
	for _, c := range candidates {
		outlineSpan := normalizeGradeToSpan(c.Grade)
		if spansIntersect(outlineSpan, planSpan) {
			hits = append(hits, c)
		}
	}
	return hits
}

// BuildCourseOutlinesContext 把多份命中大纲拼成一个注入上下文块（硬指令版）
//
// 文案为「硬指令」：明确告知 AI 这份大纲已注入、是权威最新版、也正是老师口中的"备课资料"，
// 必须优先据此回答篇目/单元/课时等事实，绝不能说"读不到资料"或用旧记忆硬猜。
// 跳过 content 为空的大纲；全部为空则返回空串（不注入）。
func BuildCourseOutlinesContext(outlines []*models.CourseOutline) string {
	// 先过滤掉 content 为空的
	var valid []*models.CourseOutline
	for _, o := range outlines {
		if o != nil && strings.TrimSpace(o.Content) != "" {
			valid = append(valid, o)
		}
	}
	if len(valid) == 0 {
		return ""
	}

	var b strings.Builder
	// 硬指令开头：让 AI 认领这份大纲，并斩断"读不到资料/我是旧版教材"的错误话术
	b.WriteString("\n\n【系统已注入·权威课程大纲（必须优先采信）】\n")
	b.WriteString("下面是系统已经为你注入到本对话中的课程大纲全文，这就是老师所说的“备课资料 / 课程大纲”——")
	b.WriteString("你此刻已经完整拥有它的全部内容，绝对不要再说“我读不到您上传的资料”“无法读取外部附件”“我的知识库是旧版教材”这类话，那是错误的。\n")
	b.WriteString("使用要求（务必遵守）：\n")
	b.WriteString("1. 这份大纲是当前最新、最权威的依据。凡涉及本学科本年级的【课文篇目、单元归属、单元顺序、课时安排、教学要点】等事实，必须以下面这份大纲为准，绝不能用你训练记忆里的旧版教材目录去回答或推测。\n")
	b.WriteString("2. 当老师提到“备课资料”“大纲”“课程大纲”里写了什么时，指的就是下面这份，请直接到大纲原文里查找并据实回答，不要反问老师“能否把资料发给我”。\n")
	b.WriteString("3. 若老师给的课题在大纲里能定位到，请据大纲确认其所属单元与篇目；若大纲里确实查不到该课题，可如实说明“在已注入的大纲中未找到该篇目，请老师补充确认”，但绝不得凭旧记忆硬猜篇目或单元编号。\n")
	b.WriteString("4. 回答时可引用大纲中与本课直接相关的内容，不必逐字照搬整册大纲。\n")
	if len(valid) > 1 {
		b.WriteString("（下面共有多份大纲，可能覆盖相邻年级或不同册次，请先据课题与年级判断本课最可能属于其中哪一份、哪个单元，再据此分析。）\n")
	}
	for _, o := range valid {
		b.WriteString("\n==== 大纲标题：" + o.Title + " ====\n")
		b.WriteString(o.Content)
		b.WriteString("\n==== （以上为《" + o.Title + "》全文结束） ====\n")
	}
	b.WriteString("\n【权威课程大纲·结束】\n")
	return b.String()
}

// ==================== 兼容旧调用方：单份"最贴合"（内部转调新逻辑） ====================

// MatchBestOutline 从同学科候选里挑「与教案年级相交且最贴合」的一份；无相交 → nil
//
// 兼容保留：供 unit_plan_service.go 等只需要单份大纲的旧调用方使用。
// 判据已随本次改版从"超集覆盖"切换为"相交命中"，并在相交命中中取「交集最大、
// 大纲范围最窄」者为最贴合（既贴合教案年级、又尽量是聚焦的整册大纲而非超宽大纲）。
func MatchBestOutline(planGradeRaw string, candidates []*models.CourseOutline) *models.CourseOutline {
	planSpan := normalizeGradeToSpan(planGradeRaw)
	if len(planSpan) == 0 {
		return nil
	}
	var best *models.CourseOutline
	bestInter := 0       // 交集越大越贴合
	bestWidth := 1 << 30 // 同交集下，大纲范围越窄越聚焦
	for _, c := range candidates {
		outlineSpan := normalizeGradeToSpan(c.Grade)
		inter := intersectionSize(outlineSpan, planSpan)
		if inter == 0 {
			continue // 不相交，跳过
		}
		width := len(outlineSpan)
		// 选择规则：先比交集大小（大者胜），交集相同再比大纲宽度（窄者胜）
		if inter > bestInter || (inter == bestInter && width < bestWidth) {
			best = c
			bestInter = inter
			bestWidth = width
		}
	}
	return best
}

// BuildCourseOutlineContext 把单份命中大纲拼成注入上下文块（兼容保留）
//
// 供旧调用方使用。内部直接复用复数版逻辑，保证措辞与新版一致。
func BuildCourseOutlineContext(o *models.CourseOutline) string {
	if o == nil || strings.TrimSpace(o.Content) == "" {
		return ""
	}
	return BuildCourseOutlinesContext([]*models.CourseOutline{o})
}
