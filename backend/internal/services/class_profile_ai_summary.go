package services

// class_profile_ai_summary.go — 班级学情·AI 总结学情（批次2c，合规核心）
//
// 一句话职责：把 class_students 的学生个体明细，在后端【就地脱敏聚合】成"匿名统计量"纯文本，
// 喂给 AI，让 AI 产出班级卡的四大段（整体画像/分层结构/薄弱点/教学建议），
// 【只生成、不落库】返回给前端，由老师在预览弹窗里点"采用"后才走现成的 UpdateProfile 通道写回。
//
// ============================ 合规红线（贯穿全文，不可妥协）============================
//   1. 个体明细（尤其 student_code 学号代号）【永不进 AI 链路】。
//      喂给 AI 的只有 buildAnonymizedClassStats 产出的匿名统计量——只有数字、占比、
//      聚合后的薄弱点频次，没有任何单个学生可识别信息。
//   2. 薄弱点频次是【全班聚合计数】（"分数加减 出现 8 次"），不是"某学号的薄弱点"。
//   3. 即便 AI 产出的四大段，也只是"群体结论"，老师确认后才写回班级卡（卡本身就是匿名群体描述）。
//   依据：《未成年人保护法》《个人信息保护法》对未成年人评价性记录的要求。
//
// ============================ 三个不能错的技术点（照 unit_plan_service.go 抄）============================
//   1. 复用 lesson_plan 场景码（已配 gemini-3.1-pro-preview，绝不新建场景码——新码回落豆包 503）。
//   2. 填 SchoolID 进 TraceContext（GetSchoolIDByUserID + schoolIDPtr），供境内外双网关分流。
//   3. 单轮 CallAI（system + 一条 user），不走流式、不带历史数组。
//
// ============================ 产品决策（已与 Yuhan 敲定）============================
//   决策1-A：薄弱点频次用"精确匹配"（去空格去标点后精确字符串计数），v1 不做近义合并。
//   决策2-B：只生成不落库；前端弹预览，老师点"采用"才覆盖四大段（复用现成 UpdateProfile）。
//   决策3-B：数据少时不硬拦（前端软提示 5 人阈值）；后端仅在【0 人】时拒绝（无数据无法总结），
//            并把真实人数写进统计量让 AI 自己注意样本量。
//
// 与 class_profile_service.go 分文件：本文件是唯一会调 AI 的班级学情逻辑，独立成文便于审计合规链路；
// 方法仍挂在 ClassProfileService 上（共用 ensureProfileOwned 鉴权闸门）。
//
// ⚠ ClassProfileService 当前构造函数 NewClassProfileService() 不持 cfg（批次1/2a 不调 AI）。
//   本文件需要 cfg 取 AES 密钥与兜底模型，故新增一个"带 cfg 的总结方法接收 cfg 作参数"的形态——
//   见 SummarizeClassProfile 的 cfg 参数。这样不改 service 结构体、不改已有构造函数，改动面最小。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ---------- 业务错误（sentinel，供 handler 用 errors.Is 映射 HTTP 码）----------

var (
	// ErrClassSummaryNoData 班里一个学生都没有，无数据可总结（决策3-B：仅 0 人时硬拒）
	ErrClassSummaryNoData = errors.New("该班还没有任何学生档案，请先录入或导入成绩单后再让 AI 总结")
	// ErrClassSummaryAIFailed AI 返回内容无法解析为四大段（多重兜底后仍失败）
	ErrClassSummaryAIFailed = errors.New("AI 总结结果解析失败，请稍后重试")
)

// ---------- 复用场景码常量（与 unit_plan_service.go 同口径）----------

// classSummarySceneCode 复用备课文本场景（已配模型 + 走境内外分流），绝不新建场景码
const classSummarySceneCode = "lesson_plan"

// 薄弱点频次展示上限（决策1-A：精确计数后频次降序取前 N 个进统计量）
const weakTopicTopN = 10

// ---------- 响应 DTO（本文件内定义，纯响应结构不污染 model 文件）----------

// ClassSummaryResult AI 总结的返回结构（只生成不落库）
//
// 四大段交前端填进预览弹窗，老师点"采用"后用这四段调现成的 UpdateProfile 写回班级卡。
// StatsText 是喂给 AI 的那段匿名统计量原文，回传给前端展示"AI 是基于这些数据总结的"，
// 增强可信度与透明度（这段本身就是脱敏后的，可安全展示给老师）。
type ClassSummaryResult struct {
	OverallProfile string `json:"overall_profile"` // 整体画像
	TierStructure  string `json:"tier_structure"`  // 分层结构（A/B/C 群体描述）
	WeakPoints     string `json:"weak_points"`     // 学科薄弱点
	TeachingAdvice string `json:"teaching_advice"` // 分层教学建议

	StudentCount int    `json:"student_count"` // 参与总结的学生数（供前端展示样本量）
	StatsText    string `json:"stats_text"`    // 喂给 AI 的匿名统计量原文（脱敏，可展示）
}

// classSummaryAIOutput AI 严格按此 JSON 结构返回四大段（解析目标）
type classSummaryAIOutput struct {
	OverallProfile string `json:"overall_profile"`
	TierStructure  string `json:"tier_structure"`
	WeakPoints     string `json:"weak_points"`
	TeachingAdvice string `json:"teaching_advice"`
}

// ========================================================================
// 对外主方法：SummarizeClassProfile —— 脱敏聚合 + AI 总结（只生成不落库）
// ========================================================================

// SummarizeClassProfile 让 AI 基于该班学生明细的匿名统计量，生成班级卡四大段。
//
// 参数 cfg 由 handler 透传（取 AES 密钥 + 兜底模型/网关）。
// 流程：
//   1. ensureProfileOwned 校验班级卡归属当前老师（复用同包闸门）。
//   2. 拉本班全部学生；0 人则返 ErrClassSummaryNoData（决策3-B 仅此一档硬拒）。
//   3. buildAnonymizedClassStats 就地脱敏聚合成匿名统计量纯文本（合规灵魂）。
//   4. 复用 lesson_plan 场景 + 填 SchoolID + 单轮 CallAI。
//   5. 解析 AI 返回的四大段 JSON（多重兜底）。
//   6. 返回四大段 + 统计量原文，【不落库】。
func (s *ClassProfileService) SummarizeClassProfile(
	ctx context.Context, cfg *config.Config, userID, classProfileID string,
) (*ClassSummaryResult, error) {

	// 1) 归属校验（拿到班级卡，后续把学科/年级/班名带进提示词增强针对性）
	profile, err := s.ensureProfileOwned(ctx, userID, classProfileID)
	if err != nil {
		return nil, err
	}

	// 2) 拉学生明细
	students, err := repository.ListClassStudents(ctx, classProfileID)
	if err != nil {
		return nil, err
	}
	if len(students) == 0 {
		return nil, ErrClassSummaryNoData
	}

	// 3) 脱敏聚合（合规灵魂）——此后进入 AI 的只有 statsText，绝无个体明细
	statsText := buildAnonymizedClassStats(students)

	// 4) 单轮 CallAI（复用 lesson_plan 场景 + 填 SchoolID）
	systemPrompt := buildClassSummarySystemPrompt()
	userPrompt := buildClassSummaryUserPrompt(profile, len(students), statsText)

	aiCfg, err := ai.GetEffectiveConfig(
		cfg.GetAESKey(),
		classSummarySceneCode,
		cfg.AIAPIBaseURL,
		cfg.AIAPIKey,
		cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf("AI配置获取失败: %w", err)
	}

	// 填 SchoolID 进 TraceContext（境内外分流，照 unit_plan_service.go 抄）
	schoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	uid := userID
	traceCtx := &ai.TraceContext{
		SceneCode: classSummarySceneCode,
		UserID:    &uid,
		SchoolID:  schoolIDPtr(schoolID), // schoolIDPtr 为 services 包内现成辅助（lesson_plan_gen_service.go）
	}

	result, err := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if err != nil {
		return nil, err
	}

	// 5) 解析四大段（多重兜底）
	parsed, perr := parseClassSummaryOutput(result.Content)
	if perr != nil {
		classProfileLog.Warn("AI 总结解析失败",
			"profile", classProfileID, "owner", userID, "err", perr.Error())
		return nil, ErrClassSummaryAIFailed
	}

	classProfileLog.Info("AI 总结学情完成（仅生成未落库）",
		"profile", classProfileID, "owner", userID, "students", len(students))

	// 6) 返回（不落库；落库交前端确认后走 UpdateProfile）
	return &ClassSummaryResult{
		OverallProfile: strings.TrimSpace(parsed.OverallProfile),
		TierStructure:  strings.TrimSpace(parsed.TierStructure),
		WeakPoints:     strings.TrimSpace(parsed.WeakPoints),
		TeachingAdvice: strings.TrimSpace(parsed.TeachingAdvice),
		StudentCount:   len(students),
		StatsText:      statsText,
	}, nil
}

// ========================================================================
// 脱敏聚合：buildAnonymizedClassStats —— 合规灵魂
// ========================================================================

// buildAnonymizedClassStats 把学生明细就地聚合成【只含匿名统计量】的纯文本。
//
// ⚠ 合规红线逐项保证：
//   - 输出里【绝不含】student_code、绝不含任何单个学生的明细。
//   - 薄弱点是【全班聚合计数】（频次表），不是"某学号的薄弱点"。
//   - 成绩是均分/分层均分/趋势，都是聚合数字。
//
// 产出结构（喂给 AI 的统计量）：
//   一、总人数
//   二、分层结构（A/B/C/未分层 各人数 + 占比）
//   三、成绩分布（全班均分 + 各层均分 + 最近两次考试均分趋势）
//   四、薄弱点频次（精确匹配计数，频次降序 Top10）
func buildAnonymizedClassStats(students []*models.ClassStudent) string {
	var b strings.Builder
	total := len(students)

	// ---------- 一、总人数 ----------
	b.WriteString(fmt.Sprintf("【班级规模】共 %d 名学生。\n\n", total))

	// ---------- 二、分层结构 ----------
	tierCount := map[string]int{
		models.StudentTierA:    0,
		models.StudentTierB:    0,
		models.StudentTierC:    0,
		models.StudentTierNone: 0, // 未分层
	}
	for _, st := range students {
		t := strings.TrimSpace(st.Tier)
		if t != models.StudentTierA && t != models.StudentTierB && t != models.StudentTierC {
			t = models.StudentTierNone
		}
		tierCount[t]++
	}
	b.WriteString("【分层结构】\n")
	b.WriteString(fmt.Sprintf("  A 层（拔尖）：%d 人（%s）\n", tierCount[models.StudentTierA], pct(tierCount[models.StudentTierA], total)))
	b.WriteString(fmt.Sprintf("  B 层（中等）：%d 人（%s）\n", tierCount[models.StudentTierB], pct(tierCount[models.StudentTierB], total)))
	b.WriteString(fmt.Sprintf("  C 层（学困）：%d 人（%s）\n", tierCount[models.StudentTierC], pct(tierCount[models.StudentTierC], total)))
	if tierCount[models.StudentTierNone] > 0 {
		b.WriteString(fmt.Sprintf("  未分层：%d 人（%s）\n", tierCount[models.StudentTierNone], pct(tierCount[models.StudentTierNone], total)))
	}
	b.WriteString("\n")

	// ---------- 三、成绩分布 ----------
	b.WriteString(buildScoreStats(students))

	// ---------- 四、薄弱点频次（精确匹配计数）----------
	b.WriteString(buildWeakTopicStats(students))

	return b.String()
}

// buildScoreStats 成绩分布段：全班均分 + 各层均分 + 最近两次考试均分趋势。
//
// 取数口径（全聚合，无个体）：
//   - 每个学生取其 latest_score（最近一次成绩）算"全班/各层 最新均分"。
//   - 趋势：把所有学生 scores 里的考试条目按"考试名+日期"聚合，找出全班覆盖最广的最近两次考试，
//     比较这两次的全班均分（仅作粗略趋势提示，无个体数据）。
func buildScoreStats(students []*models.ClassStudent) string {
	var b strings.Builder
	b.WriteString("【成绩分布】\n")

	// 收集每个学生的最新成绩 + 其分层，算全班/各层最新均分
	var allLatest []float64
	tierLatest := map[string][]float64{
		models.StudentTierA: {}, models.StudentTierB: {}, models.StudentTierC: {},
	}
	scoredCount := 0
	for _, st := range students {
		if st.LatestScore == nil {
			continue
		}
		scoredCount++
		v := *st.LatestScore
		allLatest = append(allLatest, v)
		t := strings.TrimSpace(st.Tier)
		if _, ok := tierLatest[t]; ok {
			tierLatest[t] = append(tierLatest[t], v)
		}
	}

	if scoredCount == 0 {
		b.WriteString("  暂无成绩数据（尚未导入任何成绩单）。\n\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("  有成绩记录的学生：%d 名。\n", scoredCount))
	b.WriteString(fmt.Sprintf("  全班最新均分：%s。\n", avgStr(allLatest)))
	if len(tierLatest[models.StudentTierA]) > 0 {
		b.WriteString(fmt.Sprintf("  A 层最新均分：%s。\n", avgStr(tierLatest[models.StudentTierA])))
	}
	if len(tierLatest[models.StudentTierB]) > 0 {
		b.WriteString(fmt.Sprintf("  B 层最新均分：%s。\n", avgStr(tierLatest[models.StudentTierB])))
	}
	if len(tierLatest[models.StudentTierC]) > 0 {
		b.WriteString(fmt.Sprintf("  C 层最新均分：%s。\n", avgStr(tierLatest[models.StudentTierC])))
	}

	// 趋势：按"考试名+日期"聚合全班均分，取覆盖人数最多的最近两次比较
	trend := buildScoreTrend(students)
	if trend != "" {
		b.WriteString(trend)
	}
	b.WriteString("\n")
	return b.String()
}

// examAgg 一次考试的全班聚合（用于趋势）
type examAgg struct {
	key   string  // 考试名+日期
	name  string  // 考试名
	at    string  // 日期（YYYY-MM-DD，字典序即时间序）
	sum   float64 // 该次考试全班总分
	count int     // 该次考试有成绩的人数
}

// buildScoreTrend 取覆盖人数足够的、日期最新的两次考试，比较全班均分给出趋势提示。
//
// 仅当能找到 >=2 次考试时才输出，否则返回空串（不强行造趋势）。
func buildScoreTrend(students []*models.ClassStudent) string {
	aggMap := map[string]*examAgg{}
	for _, st := range students {
		scores := models.ParseClassStudentScores(st.Scores)
		for _, sc := range scores {
			name := strings.TrimSpace(sc.Name)
			at := strings.TrimSpace(sc.At)
			if name == "" && at == "" {
				continue
			}
			key := name + "|" + at
			a, ok := aggMap[key]
			if !ok {
				a = &examAgg{key: key, name: name, at: at}
				aggMap[key] = a
			}
			a.sum += sc.Score
			a.count++
		}
	}
	if len(aggMap) < 2 {
		return ""
	}

	// 转切片，按日期降序（最新在前）；日期为 YYYY-MM-DD 字典序即时间序
	aggs := make([]*examAgg, 0, len(aggMap))
	for _, a := range aggMap {
		aggs = append(aggs, a)
	}
	sort.Slice(aggs, func(i, j int) bool {
		if aggs[i].at != aggs[j].at {
			return aggs[i].at > aggs[j].at // 日期新的在前
		}
		return aggs[i].name > aggs[j].name // 同日期按名字稳定排序
	})

	// 取最新两次
	latest := aggs[0]
	prev := aggs[1]
	if latest.count == 0 || prev.count == 0 {
		return ""
	}
	latestAvg := latest.sum / float64(latest.count)
	prevAvg := prev.sum / float64(prev.count)
	diff := latestAvg - prevAvg

	dir := "基本持平"
	if diff > 0.5 {
		dir = "较上次有所上升"
	} else if diff < -0.5 {
		dir = "较上次有所下降"
	}
	return fmt.Sprintf(
		"  近两次考试趋势：上一次「%s」全班均分 %.1f（%d 人），最近一次「%s」全班均分 %.1f（%d 人），%s。\n",
		examLabel(prev), prevAvg, prev.count,
		examLabel(latest), latestAvg, latest.count, dir,
	)
}

// examLabel 拼考试展示名（名/日期可能其一为空）
func examLabel(a *examAgg) string {
	switch {
	case a.name != "" && a.at != "":
		return a.name + " " + a.at
	case a.name != "":
		return a.name
	default:
		return a.at
	}
}

// buildWeakTopicStats 薄弱点频次段（决策1-A：精确匹配计数）。
//
// 把每个学生的 weak_topics 文本拆成"薄弱点条目"，对每条做规范化（去首尾空格、去标点），
// 再按规范化后的【精确字符串】计数。频次降序取 Top N 写入统计量。
// v1 不做近义合并（"分数加减"/"分数的加减"算两个，等真实数据积累后再决定是否做 B 方案）。
//
// 拆分规则：weak_topics 里老师可能用中文逗号/顿号/分号/斜杠/换行分隔多个薄弱点，统一按这些分隔符拆。
func buildWeakTopicStats(students []*models.ClassStudent) string {
	var b strings.Builder
	b.WriteString("【薄弱点频次（全班聚合，按精确匹配计数）】\n")

	freq := map[string]int{}       // 规范化文本 → 出现人次
	display := map[string]string{} // 规范化文本 → 首次出现的原始展示文本（保留可读性）
	order := []string{}            // 规范化文本首次出现顺序（频次相同时稳定排序用）

	for _, st := range students {
		raw := strings.TrimSpace(st.WeakTopics)
		if raw == "" {
			continue
		}
		for _, item := range splitWeakTopics(raw) {
			norm := normalizeWeakTopic(item)
			if norm == "" {
				continue
			}
			if _, ok := freq[norm]; !ok {
				display[norm] = strings.TrimSpace(item)
				order = append(order, norm)
			}
			freq[norm]++
		}
	}

	if len(freq) == 0 {
		b.WriteString("  暂无薄弱点记录（学生档案的薄弱点字段均为空）。\n\n")
		return b.String()
	}

	// 频次降序排序；频次相同按首次出现顺序（稳定、可复现）
	type wt struct {
		norm  string
		count int
		idx   int
	}
	items := make([]wt, 0, len(freq))
	for i, norm := range order {
		items = append(items, wt{norm: norm, count: freq[norm], idx: i})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count != items[j].count {
			return items[i].count > items[j].count
		}
		return items[i].idx < items[j].idx
	})

	limit := weakTopicTopN
	if limit > len(items) {
		limit = len(items)
	}
	for i := 0; i < limit; i++ {
		b.WriteString(fmt.Sprintf("  · %s：%d 人次\n", display[items[i].norm], items[i].count))
	}
	if len(items) > limit {
		b.WriteString(fmt.Sprintf("  （另有 %d 个低频薄弱点未列出）\n", len(items)-limit))
	}
	b.WriteString("\n")
	return b.String()
}

// ---------- 纯函数辅助 ----------

// pct 计算占比字符串（n/total），total 为 0 返 "0%"
func pct(n, total int) string {
	if total <= 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", float64(n)/float64(total)*100)
}

// avgStr 计算均值并格式化为一位小数字符串；空切片返 "—"
func avgStr(vals []float64) string {
	if len(vals) == 0 {
		return "—"
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return fmt.Sprintf("%.1f", sum/float64(len(vals)))
}

// weakTopicSeparators 薄弱点条目分隔符（中英文逗号/顿号/分号/斜杠/竖线/换行）
var weakTopicSeparators = []string{"，", ",", "、", "；", ";", "/", "／", "|", "\n", "\r"}

// splitWeakTopics 把一段薄弱点文本按分隔符拆成多个条目
func splitWeakTopics(raw string) []string {
	parts := []string{raw}
	for _, sep := range weakTopicSeparators {
		next := make([]string, 0, len(parts))
		for _, p := range parts {
			next = append(next, strings.Split(p, sep)...)
		}
		parts = next
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// weakTopicPunct 规范化时要剥掉的标点（中英文常见标点 + 空白）
var weakTopicPunct = []string{
	" ", "\t", "　", // 半角空格/制表/全角空格
	"。", ".", "！", "!", "？", "?",
	"：", ":", "；", ";", "，", ",", "、",
	"（", "）", "(", ")", "【", "】", "[", "]", "「", "」",
	"\"", "'", "“", "”", "‘", "’", "·", "—", "-", "～", "~",
}

// normalizeWeakTopic 规范化薄弱点文本（去空格去标点）用于精确匹配计数。
//
// 决策1-A：仅做"去空格去标点"这种最轻的规范化，不做近义/同义合并。
// 例："分数 加减。" 与 "分数加减" 归一为同一 key "分数加减"；
//     但 "分数加减" 与 "分数的加减" 仍是两个不同 key（v1 不合并）。
func normalizeWeakTopic(s string) string {
	out := strings.TrimSpace(s)
	for _, p := range weakTopicPunct {
		out = strings.ReplaceAll(out, p, "")
	}
	return out
}

// ========================================================================
// 提示词构建（硬编码，不入库；固定聚合总结指令，无需老师改/版本管理）
// ========================================================================

// buildClassSummarySystemPrompt 系统提示词：界定 AI 角色、合规边界、输出格式。
//
// 关键约束：
//   - 明确告知 AI 它拿到的是"匿名群体统计量"，禁止虚构、禁止编造个体姓名/学号。
//   - 输出严格 JSON 四大段，便于后端精确解析。
//   - 用中文「」包裹示例，避免英文双引号破坏 JSON（与 designerMetaPrompt 同思路）。
func buildClassSummarySystemPrompt() string {
	return `你是一位经验丰富的中小学班级学情分析师，擅长把班级的匿名统计数据，转写成对一线老师备课真正有用的"班级学情画像"。

【你拿到的数据是什么】
你只会拿到一个班级的【匿名群体统计量】：总人数、ABC 各层人数与占比、成绩均分与分层均分、近两次考试趋势、以及全班薄弱点的聚合频次。
你【看不到、也不会拿到】任何单个学生的姓名、学号或个人明细——数据已在后端脱敏聚合。

【严格禁止】
- 禁止虚构任何不在统计量里的数据（不许编造具体分数段人数、不许编造统计量没给的薄弱点）。
- 禁止出现任何学生姓名、学号、代号或指向某个具体学生的描述。你的全部结论都必须是"群体层面"的。
- 如果某项数据缺失（如暂无成绩），就如实说明该维度信息不足，不要硬编。

【样本量提醒】
统计量会告诉你真实人数。若人数很少（如不足 5 人），请在"整体画像"里温和提示老师"当前样本偏少，结论仅供参考，建议补充更多学生数据后再分析"，但仍照常给出基于现有数据的分析。

【你要产出什么】
产出班级学情卡的四大段，全部面向"老师备课时怎么用"，具体、可操作、避免空话套话：
1. overall_profile（整体画像）：班级整体基础水平、可能的学习风格倾向、由数据可推断的班风/参与度线索。
2. tier_structure（分层结构）：A/B/C 三层的人数占比与各层群体特征，以及各层在备课时应被如何区别对待（匿名群体描述）。
3. weak_points（学科薄弱点）：基于薄弱点频次，归纳出最需要在教学中重点突破的若干知识点/能力点，按重要程度排序。
4. teaching_advice（分层教学建议）：针对上述画像，给出可落地的分层教学策略（导入如何照顾 C 层、A 层如何拓展、关键环节如何设分层练习等）。

【输出格式（务必严格遵守）】
只输出一个 JSON 对象，不要任何额外文字、不要 Markdown 代码块包裹。结构如下（值为中文正文，可含换行）：
{"overall_profile":"……","tier_structure":"……","weak_points":"……","teaching_advice":"……"}

每段控制在 120～300 字，语言朴实、像一位资深教研员对同事说话，避免浮夸赞美与空泛口号。`
}

// buildClassSummaryUserPrompt 用户提示词：把班级定位信息 + 匿名统计量喂给 AI。
//
// profile 提供学科/年级/班名增强针对性（这些是班级层面的非敏感信息，可入 AI）。
// statsText 是 buildAnonymizedClassStats 产出的脱敏统计量。
func buildClassSummaryUserPrompt(profile *models.ClassProfile, studentCount int, statsText string) string {
	var b strings.Builder
	b.WriteString("请为下面这个班级生成学情画像四大段。\n\n")
	b.WriteString("【班级基本信息】\n")
	b.WriteString(fmt.Sprintf("  学科：%s\n", safeField(profile.Subject)))
	b.WriteString(fmt.Sprintf("  年级：%s\n", safeField(profile.Grade)))
	b.WriteString(fmt.Sprintf("  班级：%s\n", safeField(profile.ClassName)))
	if strings.TrimSpace(profile.Term) != "" {
		b.WriteString(fmt.Sprintf("  学期：%s\n", profile.Term))
	}
	b.WriteString("\n")
	b.WriteString("【该班匿名学情统计量（已脱敏，不含任何学生个人信息）】\n")
	b.WriteString(statsText)
	b.WriteString("\n请严格按系统提示词要求，只输出 JSON 四大段。")
	return b.String()
}

// safeField 字段为空时给占位，避免提示词里出现"学科：（空）"这种突兀表达
func safeField(s string) string {
	if strings.TrimSpace(s) == "" {
		return "（未填写）"
	}
	return s
}

// ========================================================================
// AI 输出解析（多重兜底）
// ========================================================================

// parseClassSummaryOutput 解析 AI 返回的四大段 JSON（多重兜底）。
//
// 兜底链：
//   ① 直接 Unmarshal 原文。
//   ② 剥 Markdown 代码围栏（```json ... ```）后再 Unmarshal。
//   ③ 从首个 { 到末个 } 截取对象再 Unmarshal。
// 三步任一成功且至少有一段非空即返回；全失败返错误。
func parseClassSummaryOutput(raw string) (*classSummaryAIOutput, error) {
	candidates := []string{
		strings.TrimSpace(raw),
		stripCodeFenceForSummary(raw),
		sliceFirstJSONObjectForSummary(raw),
	}
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		var out classSummaryAIOutput
		if err := json.Unmarshal([]byte(c), &out); err == nil {
			// 至少一段非空才算解析成功（防 AI 返回全空 JSON）
			if strings.TrimSpace(out.OverallProfile) != "" ||
				strings.TrimSpace(out.TierStructure) != "" ||
				strings.TrimSpace(out.WeakPoints) != "" ||
				strings.TrimSpace(out.TeachingAdvice) != "" {
				return &out, nil
			}
		}
	}
	return nil, errors.New("无法从 AI 输出解析出四大段")
}

// stripCodeFenceForSummary 剥掉 ```json ... ``` 或 ``` ... ``` 围栏，返回内部内容
func stripCodeFenceForSummary(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return ""
	}
	// 去掉首行 ```xxx
	if idx := strings.IndexByte(t, '\n'); idx >= 0 {
		t = t[idx+1:]
	}
	// 去掉结尾 ```
	if idx := strings.LastIndex(t, "```"); idx >= 0 {
		t = t[:idx]
	}
	return strings.TrimSpace(t)
}

// sliceFirstJSONObjectForSummary 从首个 { 到最后一个 } 截取（最朴素的对象抽取兜底）
func sliceFirstJSONObjectForSummary(s string) string {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end < 0 || end <= start {
		return ""
	}
	return s[start : end+1]
}
