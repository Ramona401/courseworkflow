package services

// workshop_stage_extract.go — 阶段产出物自然语言提取
//
// v76拆分自 workshop_stage_prompts.go
// v77修复：extractReviewStageFromNatural 全面重写
// v82清理：删除废弃函数
// v84拆分：GenerateStageSummary及相关摘要生成函数 移至 workshop_stage_summary.go
//
// v169改动（评审"有分无内容"治本）：
//   - extractReviewStageFromNatural 改为「JSON优先 + 正则降级」双链路：
//       1) 先尝试从 AI 回复尾部的 ```json 代码块解析结构化评审数据（与新版 review
//          阶段提示词约定的格式对齐，最可靠）
//       2) 解析失败再降级到原有的 Markdown 正则提取（兼容旧格式/AI偶尔漏 JSON）
//   - 修正原正则两处脏数据：
//       a) extractDimensionsFromTable 跳过表头行（"评审维度/维度/评分/简短评语"等）
//       b) 维度 name 剥离 Markdown 粗体星号（"**T1-教学目标**" → "教学目标"）
//   - parseReviewJSONBlock 复用 ai.ExtractJSON，但优先定位「最后一个 ```json 块」，
//     避免报告正文里的花括号干扰花括号配平
//
// v189改动（改动A：放宽 DetectLessonPlanContent 的 # 标题硬判定 — 评审bug治本核心）：
//   - 原逻辑：必须存在以 "#" 开头且含教案标题词的行作为正文起点，否则整体返回 ""。
//     副作用：AI 偶尔输出"通篇无 # 标题"（纯文本小标题）的完整教案时，即便五重叠加
//     闸门（≥3标记词 + 教学过程 + 结尾标记 + ≥800字）全部满足，也会被这一道格式井号
//     判死，导致正文不落库、评审拿不到正文报"请回撰写阶段"。
//   - 新逻辑：保留"优先找 # 标题行"为主路径；找不到 # 标题时，降级用"第一行含教案
//     标题词的纯文本行"作为正文起点。后续 hasProcess + hasEnding + ≥800字 三道闸 +
//     前置 markerCount≥3 全部不变（五重叠加仍足够严，唯一放宽的只是格式井号）。
//   - 实现：把原"只扫 # 行"的单次循环拆为两段——先按原口径找带 # 的标题行；
//     未命中再做一次降级扫描，找第一行 TrimSpace 后含 titleMarkers 任一词的非空行。
//
// v189改动（改动B：剥离评审 narrative 里泄漏的 ```json 块 — 评审报告夹JSON给老师看）：
//   - 问题：extractReviewStageFromNatural 两条链路构造 narrative 时直接 safeUTF8Truncate
//     完整原文，而原文末尾带着供解析用的 ```json 块（提示词约定 AI 输出），JSON 被
//     parseReviewJSONBlock 解析走了，但原文里那段 JSON 未被剥离，原样下发到前端，
//     老师就在报告里看到一坨 json{...}。
//   - 修法：新增 stripReviewJSONBlock，复用 extractLastJSONCodeBlock 同款围栏正则把
//     ```json / ``` 块整段移除并清理尾部空白与 ---，两条链路构造 narrative 时先剥块
//     再截断。只剥展示文本，parseReviewJSONBlock 仍吃原始 content，解析逻辑零影响；
//     无 JSON 块的旧格式报告原样返回不受影响。
//
// v193改动（write/revise 阶段「教师建议块」硬切 — 创新剥离·轻路落地）：
//   - 背景：write/revise 阶段 AI 出完整教案时，常自作主张新增设计阶段从未与老师讨论过、
//     却会实质影响教学组织的新活动（如"录配音发班级群评选最美好声音"这类作业形式），
//     把这些"加戏"直接写进教案正文当作既定方案，甚至评审阶段还自夸这些虚构内容，
//     老师难以分辨哪些是自己拍板的、哪些是 AI 私自添加的。
//   - 方案（轻路）：约定 AI 把这类"设计阶段未讨论、需老师定夺的创新建议"写进一个专用围栏
//     ```teacher_suggestion ... ``` 块（提示词侧约束，见 workshop_stage_prompts.go 与
//     lesson_plan_gen_fullgen_prompts.go 的 write/revise 四处）。后端在提取教案前，用
//     splitSuggestionBlock 把该块从 content 中整段切走：
//       * pureContent（已无建议块）喂 DetectLessonPlanContent → 教案正文【硬保证】不含建议；
//       * 切出的建议文本拼进 narrative（老师在 AI 对话气泡看得到、但不落库教案正文）。
//   - 为什么可靠：剥离是【代码强制】，无论 AI 把建议块放正文前还是正文后，后端都先切干净
//     再提取，不依赖 AI 自觉把建议放对位置；即便 AI 没用建议块、把创新混进正文，也只是
//     回退到 v193 前的老行为（不会更糟），用提示词持续引导 AI 用建议块。
//   - 安全：本次仅新增 splitSuggestionBlock + 改 extractWriteStageFromNatural 一个函数；
//     DetectLessonPlanContent / extractReviewStageFromNatural / 所有评审辅助函数一字未动。
//     专用围栏 ```teacher_suggestion 与 ```json（评审）、```suggested_actions（芯片）三套
//     围栏互不匹配（info-string 各不相同），互不干扰。
//
// v200改动（P0-04：评审 narrative 残留 ```suggested_actions 芯片块 — 治本）：
//   - 问题：review 阶段 AI 在评审报告末尾同时输出两套围栏——评审用 ```json 块、给老师的
//     推进芯片用 ```suggested_actions 块。extractReviewStageFromNatural 构造 narrative 时
//     只调 stripReviewJSONBlock 剥 json 块，而该正则按设计不匹配 ```suggested_actions
//     围栏（两套围栏 info-string 不同、刻意互斥，见 lesson_plan_gen_actions.go），导致
//     芯片块原样残留在 narrative 中，下发并显示在右侧教案画布，老师看到一坨
//     suggested_actions{...} 代码。
//   - 修法：在三处 narrative 构造时，于 stripReviewJSONBlock 外再套一层同包现成的
//     StripSuggestedActionsBlock（专剥 ```suggested_actions 围栏，绝不动 ```json）。
//
// v200改动（P0-04 补漏：write/revise 阶段教案正文落库残留 ```suggested_actions 芯片块）：
//   - 问题：write/revise 阶段 AI 出完整教案时，也会在教案正文末尾（板书设计之后）追加
//     ```suggested_actions 芯片块。extractWriteStageFromNatural 先 splitSuggestionBlock
//     只剥 ```teacher_suggestion 块，再 DetectLessonPlanContent 从标题行 lines[startIdx:]
//     一直取到末尾（trimTrailingChatter 只去客套话、不剥芯片围栏），导致芯片块被当成正文
//     尾巴一起截进 content_markdown 落库，显示到右侧画布。这是与 review 阶段同源但不同
//     出口的残留（首例《春》因芯片块恰好被截在正文范围外而未暴露，本例《分数》暴露）。
//   - 修法：extractWriteStageFromNatural 中，对 DetectLessonPlanContent 提取出的
//     lessonContent 再套一层 StripSuggestedActionsBlock 后才落库（治本：不论 AI 把芯片块
//     放正文何处，落库正文都不含芯片块）；未识别到教案的降级 narrative 同样补剥一层。
//     不改 DetectLessonPlanContent 本体（它被前端 isFullLessonPlanMessage 口径对齐、被多
//     处复用，影响面大），只在 write/revise 专用提取入口剥，影响面最小。
//
// 包含：
//   - ExtractStructuredFromNaturalReply：从自然语言回复中提取结构化数据（v75）
//   - DetectLessonPlanContent：检测教案Markdown内容（v75，v189放宽#判定）
//   - splitSuggestionBlock：切出 write/revise 的「教师建议块」（v193）
//   - extractScoreFromText：提取评审分数（v75）
//   - 评审信息提取：extractReviewStageFromNatural等（v77重写，v169增强，v189剥JSON块，v200剥芯片块）

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	aiClient "tedna/internal/ai"
)

// ==================== 从自然语言回复中提取结构化数据（v75）====================

// ExtractStructuredFromNaturalReply 从AI自然语言回复中提取结构化信息
func ExtractStructuredFromNaturalReply(stageCode string, content string) (structuredJSON string, narrative string, hasContent bool) {
	switch stageCode {
	case "write", "revise":
		return extractWriteStageFromNatural(content)
	case "review":
		return extractReviewStageFromNatural(content)
	default:
		return extractGenericStageFromNatural(stageCode, content)
	}
}

// ==================== v193：教师建议块（teacher_suggestion）切割 ====================

// suggestionFenceRegex 匹配「教师建议块」专用围栏 ```teacher_suggestion\n ... \n```
// 非贪婪、跨行；info-string 固定为 teacher_suggestion，与 ```json（评审）、
// ```suggested_actions（芯片）两套围栏的 info-string 均不同，正则互不匹配、互不干扰。
var suggestionFenceRegex = regexp.MustCompile("(?s)```teacher_suggestion\\s*\\n(.*?)```")

// splitSuggestionBlock 从 AI 回复中切出所有「教师建议块」。
//
// 返回：
//   - pureContent：移除所有 ```teacher_suggestion 块后的纯净回复（用于教案提取，
//     保证落库教案正文绝不含建议——这是 v193 的硬保证核心）；
//   - suggestionText：所有建议块内文本合并后的结果（用于拼进 narrative 让老师看到）。
//
// 设计要点：
//   - 无建议块时：pureContent 原样返回、suggestionText 为空，行为与 v193 前完全一致；
//   - 多个建议块时：内容按出现顺序用空行合并；
//   - 只切专用围栏，绝不动 ```json / ```suggested_actions 等其它围栏。
func splitSuggestionBlock(content string) (pureContent string, suggestionText string) {
	if content == "" {
		return "", ""
	}

	// 提取所有建议块内文本
	matches := suggestionFenceRegex.FindAllStringSubmatch(content, -1)
	var parts []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		inner := strings.TrimSpace(m[1])
		if inner != "" {
			parts = append(parts, inner)
		}
	}
	suggestionText = strings.TrimSpace(strings.Join(parts, "\n\n"))

	// 从原文整段移除建议块，得到纯净内容
	pureContent = suggestionFenceRegex.ReplaceAllString(content, "")
	// 清理因移除产生的连续空行（最多保留一个空行）
	pureContent = regexp.MustCompile(`\n{3,}`).ReplaceAllString(pureContent, "\n\n")
	pureContent = strings.TrimSpace(pureContent)

	return pureContent, suggestionText
}

// extractWriteStageFromNatural 从write/revise阶段的自然语言回复中提取教案内容
//
// v193：提取教案前先用 splitSuggestionBlock 切走「教师建议块」——
//   * pureContent（无建议块）喂 DetectLessonPlanContent，落库教案正文硬保证不含建议；
//   * 切出的建议文本拼进 narrative，老师在对话气泡看得到、但不进教案。
//
// v200（P0-04补漏）：DetectLessonPlanContent 提取出的正文会从标题行一直取到末尾，AI 追加在
//   教案末尾的 ```suggested_actions 芯片块会被一并截进正文。故对 lessonContent 再套
//   StripSuggestedActionsBlock 后才落库，保证落库教案正文绝不含芯片块；未识别到教案时的
//   降级 narrative 同样补剥一层（与展示路径口径一致）。
func extractWriteStageFromNatural(content string) (string, string, bool) {
	// v193：先切走教师建议块，后续教案提取只认纯净内容
	pureContent, suggestionText := splitSuggestionBlock(content)

	lessonContent := DetectLessonPlanContent(pureContent)
	if lessonContent == "" {
		// 未识别到完整教案：narrative 用纯净内容截断（v200补剥芯片块）；若有建议块，仍把建议拼上供老师看
		narrative := safeUTF8Truncate(StripSuggestedActionsBlock(pureContent), 500)
		narrative = appendSuggestionToNarrative(narrative, suggestionText)
		return "{}", narrative, false
	}

	// v200（P0-04补漏）：剥掉被 DetectLessonPlanContent 一并截进正文的 ```suggested_actions 芯片块，
	// 保证落库教案正文绝不含芯片代码；剥后重新 TrimSpace 去尾部残留空白。
	lessonContent = strings.TrimSpace(StripSuggestedActionsBlock(lessonContent))

	structured := map[string]interface{}{"content_markdown": lessonContent}
	b, _ := json.Marshal(structured)

	// narrative：取纯净内容里「教案正文之前」的文字（原逻辑），再拼上建议块文本
	narrativeIdx := strings.Index(pureContent, lessonContent)
	narrative := ""
	if narrativeIdx > 0 {
		narrative = strings.TrimSpace(pureContent[:narrativeIdx])
	}
	if narrative == "" {
		narrative = fmt.Sprintf("已生成教案（%d字符）", len(lessonContent))
	}
	narrative = appendSuggestionToNarrative(narrative, suggestionText)

	wsLog.Info("从自然语言回复中提取到教案内容",
		"content_len", len(lessonContent), "narrative_len", len(narrative),
		"has_suggestion", suggestionText != "")
	return string(b), narrative, true
}

// appendSuggestionToNarrative 把「教师建议块」文本以醒目前缀拼到 narrative 末尾（v193）。
//
// narrative 是给老师在对话气泡展示的文本（不落库教案正文），把 AI 的创新建议放这里，
// 既让老师看得到、可决策，又与教案正文物理分离。建议为空时原样返回 narrative。
func appendSuggestionToNarrative(narrative string, suggestionText string) string {
	if strings.TrimSpace(suggestionText) == "" {
		return narrative
	}
	block := "💡 我的补充建议（供您参考，未写入教案正文）：\n" + suggestionText
	if strings.TrimSpace(narrative) == "" {
		return block
	}
	return narrative + "\n\n" + block
}

// DetectLessonPlanContent 检测并提取AI回复中的完整教案Markdown内容
//
// 判定为有效教案需同时满足（五重叠加，缺一不可）：
//  1. markerCount ≥ 3：命中至少3个教案标记词（教学目标/教学过程/作业布置…）
//  2. 能定位正文起点 startIdx：
//     - 主路径：存在以 "#" 开头且含教案标题词的行（最规范）
//     - 降级路径（v189新增）：无 # 标题时，取第一行"含教案标题词的纯文本非空行"
//  3. hasProcess：含"教学过程/教学环节/教学活动"任一
//  4. hasEnding：含"作业布置/板书设计/课后作业/课堂小结…"任一结尾标记
//  5. 截取后正文长度 ≥ 800 字符
//
// v189：放宽第2条对格式井号的硬要求——格式井号缺失不再判死，但其余四道闸不变。
func DetectLessonPlanContent(content string) string {
	if content == "" {
		return ""
	}
	lessonMarkers := []string{
		"教学目标", "教学重点", "教学难点", "教学重难点",
		"教学过程", "教学准备", "作业布置", "板书设计",
		"教学方法", "教学评价", "课时安排",
		// 扩展变体（AI实际输出中常见）
		"课后作业", "课后练习", "课堂小结", "课堂总结",
		"教学内容", "学习目标", "学习重点", "教学环节",
		"导入", "新课", "巩固练习", "小结",
	}
	markerCount := 0
	for _, marker := range lessonMarkers {
		if strings.Contains(content, marker) {
			markerCount++
		}
	}
	// 放宽阈值：只需3个核心标记词即可（AI输出格式多样，不同教案结构差异大）
	if markerCount < 3 {
		return ""
	}
	lines := strings.Split(content, "\n")

	// titleMarkers：用于定位"正文起点"的教案标题词；# 标题与纯文本降级两路径共用
	titleMarkers := []string{
		"教案", "教学设计", "教学目标", "课题", "课时",
		"教学重点", "教学难点", "教学重难点", "教学准备",
	}

	// ---------- 起点定位：主路径——优先找以 # 开头且含标题词的行 ----------
	startIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, marker := range titleMarkers {
			if strings.Contains(trimmed, marker) {
				startIdx = i
				break
			}
		}
		if startIdx >= 0 {
			break
		}
	}

	// ---------- 起点定位：降级路径（v189）——无 # 标题时，取第一行含标题词的纯文本非空行 ----------
	// 仅当主路径未命中（startIdx<0）才执行；后续 hasProcess/hasEnding/≥800字 三道闸不受影响。
	if startIdx < 0 {
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			for _, marker := range titleMarkers {
				if strings.Contains(trimmed, marker) {
					startIdx = i
					break
				}
			}
			if startIdx >= 0 {
				break
			}
		}
	}

	// 两条路径都找不到正文起点：仍判否（连一行含标题词的内容都没有，不像教案）
	if startIdx < 0 {
		return ""
	}

	// 教学过程：支持多种命名方式
	hasProcess := strings.Contains(content, "教学过程") ||
		strings.Contains(content, "教学环节") ||
		strings.Contains(content, "教学活动") ||
		(strings.Contains(content, "学生活动") &&
			(strings.Contains(content, "教师话术") || strings.Contains(content, "教师活动")))
	// 结尾标记：支持多种命名方式（AI输出变体较多）
	hasEnding := strings.Contains(content, "作业布置") ||
		strings.Contains(content, "板书设计") ||
		strings.Contains(content, "课后作业") ||
		strings.Contains(content, "课后练习") ||
		strings.Contains(content, "课堂小结") ||
		strings.Contains(content, "课堂总结") ||
		strings.Contains(content, "教学反思") ||
		strings.Contains(content, "小结与作业")
	if !hasProcess || !hasEnding {
		return ""
	}

	lessonLines := lines[startIdx:]
	result := strings.TrimSpace(strings.Join(lessonLines, "\n"))
	result = trimTrailingChatter(result)
	// 放宽最小长度：800字符以上即视为有效教案内容
	if len(result) < 800 {
		return ""
	}
	return result
}

// trimTrailingChatter 去掉教案末尾的AI客套话
func trimTrailingChatter(content string) string {
	chatterPrefixes := []string{
		"如果您有任何", "如果你有任何", "如有任何",
		"如果您觉得", "如果你觉得",
		"如果需要修改", "如需修改", "如需调整",
		"希望这份教案", "以上是", "以上就是",
		"如果有其他", "如有其他",
		"您可以点击", "你可以点击",
		"请问还有", "还有什么",
		"---\n\n如果", "---\n\n以上", "---\n\n希望",
	}
	lines := strings.Split(content, "\n")
	trimEnd := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || trimmed == "---" {
			trimEnd = i
			continue
		}
		isChatter := false
		for _, prefix := range chatterPrefixes {
			if strings.HasPrefix(trimmed, prefix) {
				isChatter = true
				break
			}
		}
		if isChatter {
			trimEnd = i
			continue
		}
		break
	}
	return strings.TrimSpace(strings.Join(lines[:trimEnd], "\n"))
}

// ==================== 评审信息提取（v77重写，v169增强：JSON优先+正则降级；v189剥JSON块；v200剥芯片块）====================

// extractReviewStageFromNatural 从review阶段的自然语言回复中提取评审信息
//
// v169双链路：
//
//      链路1（优先）：解析 AI 回复尾部的 ```json 结构化块（新版提示词约定输出此块）
//      链路2（降级）：解析失败时，沿用原有 Markdown 正则提取（兼容旧格式）
//
// 两条链路任一成功（total_score>0）即返回 hasContent=true；
// 都失败时返回 hasContent=false，narrative 兜底为对话原文截断（供上层 fallback）。
//
// v189（改动B）：两条链路构造 narrative 前先调 stripReviewJSONBlock 剥掉尾部 ```json 块，
// 避免供解析用的 JSON 原样泄漏给老师看；解析仍吃原始 content 不受影响。
//
// v200（P0-04）：在 stripReviewJSONBlock 外再套 StripSuggestedActionsBlock，剥掉评审报告
// 末尾的 ```suggested_actions 芯片块（stripReviewJSONBlock 的正则按设计不匹配该围栏，
// 故芯片块原本会残留在 narrative 里显示到右侧画布）。两函数职责正交、叠加无副作用；
// 芯片解析/广播走 ParseSuggestedActions 吃原始 content，不受本改动影响。
func extractReviewStageFromNatural(content string) (string, string, bool) {
	// ---------- 链路1：JSON 块优先 ----------
	if structuredJSON, ok := parseReviewJSONBlock(content); ok {
		// narrative 用原文（含 Markdown 报告）但先剥掉尾部 JSON 块与芯片块，再截断到 2000 字符供前端展示/记忆
		narrative := safeUTF8Truncate(StripSuggestedActionsBlock(stripReviewJSONBlock(content)), 2000)
		wsLog.Info("评审提取走JSON块链路（v169）", "structured_len", len(structuredJSON))
		return structuredJSON, narrative, true
	}

	// ---------- 链路2：Markdown 正则降级 ----------
	totalScore := extractTotalScoreFromReview(content)
	if totalScore <= 0 {
		narrative := safeUTF8Truncate(StripSuggestedActionsBlock(stripReviewJSONBlock(content)), 2000)
		wsLog.Warn("评审提取两条链路均失败，返回无结构化（上层将走兜底）", "content_len", len(content))
		return "{}", narrative, false
	}

	dimensions := extractDimensionsFromTable(content)
	goodPoints := extractGoodPoints(content)
	improvements := extractImprovements(content)
	summary := extractSummary(content)

	structured := map[string]interface{}{
		"total_score":  totalScore,
		"dimensions":   dimensions,
		"good_points":  goodPoints,
		"improvements": improvements,
		"summary":      summary,
	}
	b, _ := json.Marshal(structured)
	narrative := safeUTF8Truncate(StripSuggestedActionsBlock(stripReviewJSONBlock(content)), 2000)

	wsLog.Info("评审提取走Markdown正则降级链路（v169）",
		"total_score", totalScore,
		"dimensions_count", len(dimensions),
		"good_points_count", len(goodPoints),
		"improvements_count", len(improvements),
		"summary_len", len(summary),
	)

	return string(b), narrative, true
}

// stripReviewJSONBlock 从评审原文中剥离所有 ```json / ``` 围栏代码块（v189改动B新增）
//
// 用途：构造给老师看的 narrative 前，把供 parseReviewJSONBlock 解析用、但不该展示的
// 尾部 JSON 块整段移除，避免报告里夹一坨 json{...}。
//
// 实现：复用与解析端 extractLastJSONCodeBlock 同款围栏正则（```json 或 ``` 无语言标注），
// 全局移除所有匹配块（不止最后一个，AI 偶尔多输出几个也一并清掉），再清理因移除产生的
// 尾部多余空白与单独成行的 ---。只剥围栏块，不动报告正文。
//
// 安全：只用于展示文本构造，不改变解析输入；无围栏块时原样返回。
func stripReviewJSONBlock(content string) string {
	if content == "" {
		return ""
	}
	// 与 extractLastJSONCodeBlock 同款围栏匹配：```json\n...\n``` 或 ```\n...\n```，跨行非贪婪
	re := regexp.MustCompile("(?s)```(?:json)?\\s*\\n.*?```")
	stripped := re.ReplaceAllString(content, "")

	// 清理尾部：移除因剥块残留的空行与单独成行的 ---
	lines := strings.Split(stripped, "\n")
	trimEnd := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || trimmed == "---" {
			trimEnd = i
			continue
		}
		break
	}
	return strings.TrimSpace(strings.Join(lines[:trimEnd], "\n"))
}

// parseReviewJSONBlock 从 AI 回复中提取并校验尾部的评审 JSON 块（v169新增）
//
// 设计：
//   - 优先定位「最后一个 ```json ... ``` 代码块」，避免报告正文里的花括号干扰
//   - 找不到代码块再回退用 ai.ExtractJSON 做花括号配平兜底
//   - 解析后做最小校验：total_score 必须 >0，否则视为无效
//   - 解析成功后回填两件事：
//     a) improvements 缺 id 的补 imp_N
//     b) dimensions 清洗（剥星号、过滤表头行）——双保险，即便 AI 没完全守约也干净
//   - 返回标准化后的 JSON 字符串（字段对齐 models.AIReviewResult + 前端 ReviewPanel）
func parseReviewJSONBlock(content string) (string, bool) {
	jsonStr := extractLastJSONCodeBlock(content)
	if jsonStr == "" {
		// 回退：用全局 ExtractJSON 做花括号配平（可能误匹配，故仅作兜底）
		if s, ok := aiClient.ExtractJSON(content); ok {
			jsonStr = s
		}
	}
	if jsonStr == "" {
		return "", false
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return "", false
	}

	// total_score 校验
	score := toFloat(parsed["total_score"])
	if score <= 0 || score > 10 {
		return "", false
	}

	// dimensions 清洗：剥星号 + 过滤表头行
	if rawDims, ok := parsed["dimensions"].([]interface{}); ok {
		cleanDims := make([]interface{}, 0, len(rawDims))
		for _, d := range rawDims {
			dm, ok := d.(map[string]interface{})
			if !ok {
				continue
			}
			name := strings.TrimSpace(stripBold(toStr(dm["name"])))
			if isDimensionHeaderRow(name) {
				continue // 跳过"评审维度/维度/评分"等表头脏行
			}
			dm["name"] = name
			dm["code"] = strings.TrimSpace(stripBold(toStr(dm["code"])))
			cleanDims = append(cleanDims, dm)
		}
		parsed["dimensions"] = cleanDims
	}

	// improvements 补 id
	if rawImps, ok := parsed["improvements"].([]interface{}); ok {
		for i, imp := range rawImps {
			im, ok := imp.(map[string]interface{})
			if !ok {
				continue
			}
			if strings.TrimSpace(toStr(im["id"])) == "" {
				im["id"] = fmt.Sprintf("imp_%d", i+1)
			}
		}
	}

	out, err := json.Marshal(parsed)
	if err != nil {
		return "", false
	}
	return string(out), true
}

// extractLastJSONCodeBlock 提取文本中「最后一个」```json ... ``` 代码块的内容（v169新增）
// 兼容 ```json 与 ``` （无语言标注）两种围栏；取最后一个，因为评审 JSON 约定在报告末尾
func extractLastJSONCodeBlock(text string) string {
	// 匹配 ```json\n...\n``` 或 ```\n...\n```，非贪婪，跨行
	re := regexp.MustCompile("(?s)```(?:json)?\\s*\\n(.*?)```")
	matches := re.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return ""
	}
	// 取最后一个代码块（评审 JSON 约定在最后）
	last := matches[len(matches)-1]
	if len(last) < 2 {
		return ""
	}
	candidate := strings.TrimSpace(last[1])
	// 必须像个 JSON 对象
	if !strings.HasPrefix(candidate, "{") {
		return ""
	}
	return candidate
}

// isDimensionHeaderRow 判断维度 name 是否是表格表头脏行（v169新增）
func isDimensionHeaderRow(name string) bool {
	if name == "" {
		return true
	}
	headerKeywords := []string{"评审维度", "维度", "评分", "简短评语", "评语", "得分", "分数"}
	for _, kw := range headerKeywords {
		if name == kw {
			return true
		}
	}
	return false
}

// stripBold 剥离 Markdown 粗体星号（v169新增）："**T1-教学目标**" → "T1-教学目标"
func stripBold(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "*", ""))
}

// toFloat 宽容地把 interface{} 转 float64（支持 float64/json.Number/字符串）（v169新增）
func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(n), "%f", &f); err == nil {
			return f
		}
	}
	return 0
}

// toStr 宽容地把 interface{} 转 string（v169新增）
func toStr(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// extractTotalScoreFromReview 从评审报告中提取总分
// v104修复：扩展关键词列表，并跳过括号内的说明文字（如"满分10分"），支持更多AI输出格式
func extractTotalScoreFromReview(content string) float64 {
	// 优先尝试完整关键词匹配（更精确）
	totalPatterns := []string{
		"总评分", "总分", "综合评分", "综合得分", "总体评分",
		"TOTAL", "总体得分", "评审总分", "最终评分",
	}
	score := extractScoreFromTextSkipParens(content, totalPatterns)
	if score > 0 {
		return score
	}
	// 降级：尝试表格格式的最后一行总分（如Markdown表格末行）
	return extractTotalScoreFromTable(content)
}

// extractScoreFromTextSkipParens 提取分数时跳过括号内容（如"总评分(满分10分)：8.2"）
func extractScoreFromTextSkipParens(text string, keywords []string) float64 {
	for _, kw := range keywords {
		idx := strings.Index(text, kw)
		if idx == -1 {
			continue
		}
		after := text[idx+len(kw):]
		runes := []rune(after)
		ri := 0
		// 跳过空白和冒号
		for ri < len(runes) {
			r := runes[ri]
			if r == ':' || r == '：' || r == ' ' || r == '\t' {
				ri++
				continue
			}
			break
		}
		// 跳过括号内容（如"(满分10分)"）
		if ri < len(runes) && (runes[ri] == '(' || runes[ri] == '（') {
			closeChar := rune(')')
			if runes[ri] == '（' {
				closeChar = '）'
			}
			ri++
			for ri < len(runes) && runes[ri] != closeChar {
				ri++
			}
			if ri < len(runes) {
				ri++ // 跳过闭括号
			}
			// 再次跳过空白和冒号
			for ri < len(runes) {
				r := runes[ri]
				if r == ':' || r == '：' || r == ' ' || r == '\t' {
					ri++
					continue
				}
				break
			}
		}
		// 跳过星号（粗体标记 **）
		for ri < len(runes) && runes[ri] == '*' {
			ri++
		}
		if ri >= len(runes) {
			continue
		}
		// 提取数字
		numStr := ""
		for j := ri; j < len(runes); j++ {
			r := runes[j]
			if (r >= '0' && r <= '9') || r == '.' {
				numStr += string(r)
			} else {
				break
			}
		}
		if numStr == "" {
			continue
		}
		var score float64
		if _, err := fmt.Sscanf(numStr, "%f", &score); err == nil && score > 0 && score <= 10 {
			return score
		}
	}
	return 0
}

// extractTotalScoreFromTable 从Markdown表格中提取总分行
// 匹配类似 "| 总分 | 8.5 |" 的表格行
func extractTotalScoreFromTable(content string) float64 {
	lines := strings.Split(content, "\n")
	scoreRegex := regexp.MustCompile(`(\d+\.?\d*)`)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		if strings.Contains(trimmed, "---") {
			continue
		}
		// 检查是否包含总分关键词
		isTotal := false
		for _, kw := range []string{"总分", "总评分", "综合评分", "TOTAL", "总体"} {
			if strings.Contains(trimmed, kw) {
				isTotal = true
				break
			}
		}
		if !isTotal {
			continue
		}
		matches := scoreRegex.FindAllString(trimmed, -1)
		for _, m := range matches {
			var score float64
			if _, err := fmt.Sscanf(m, "%f", &score); err == nil && score > 0 && score <= 10 {
				return score
			}
		}
	}
	return 0
}

// extractDimensionsFromTable 从Markdown表格中提取维度评分
// v169修正：跳过表头行（"评审维度/维度/评分"等）+ 剥离 name 的粗体星号
func extractDimensionsFromTable(content string) []map[string]interface{} {
	var dimensions []map[string]interface{}

	lines := strings.Split(content, "\n")
	scoreRegex := regexp.MustCompile(`(\d+\.?\d*)`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "---") {
			continue
		}
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}

		cells := strings.Split(trimmed, "|")
		var cleanCells []string
		for _, c := range cells {
			c = strings.TrimSpace(c)
			if c != "" {
				cleanCells = append(cleanCells, c)
			}
		}

		if len(cleanCells) < 3 {
			continue
		}

		dimName := strings.TrimSpace(stripBold(cleanCells[0]))
		scoreStr := strings.TrimSpace(cleanCells[1])
		comment := strings.TrimSpace(cleanCells[2])

		// v169：跳过表头脏行（原代码只挡了"维度"和含"评分"，漏了"评审维度"作为首格的整行）
		if isDimensionHeaderRow(dimName) {
			continue
		}

		matches := scoreRegex.FindStringSubmatch(scoreStr)
		if len(matches) < 2 {
			continue
		}
		var score float64
		if _, err := fmt.Sscanf(matches[1], "%f", &score); err != nil || score <= 0 || score > 10 {
			continue
		}

		code := ""
		name := dimName
		// 支持 "T1 教学目标" 或 "T1-教学目标" 两种写法
		codeRegex := regexp.MustCompile(`^(T\d+)[\s\-]+(.+)$`)
		codeMatches := codeRegex.FindStringSubmatch(dimName)
		if len(codeMatches) == 3 {
			code = codeMatches[1]
			name = strings.TrimSpace(codeMatches[2])
		}

		dim := map[string]interface{}{
			"name":    name,
			"score":   score,
			"comment": comment,
		}
		if code != "" {
			dim["code"] = code
		}
		dimensions = append(dimensions, dim)
	}

	return dimensions
}

// extractGoodPoints 提取"做得好的点"章节内容
func extractGoodPoints(content string) []string {
	var points []string

	sectionStart := -1
	sectionHeaders := []string{"做得好的点", "做得好", "亮点", "优点", "优秀之处"}
	for _, header := range sectionHeaders {
		idx := strings.Index(content, header)
		if idx >= 0 {
			sectionStart = idx
			break
		}
	}
	if sectionStart < 0 {
		return points
	}

	sectionContent := content[sectionStart:]
	lines := strings.Split(sectionContent, "\n")
	var sectionLines []string
	started := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !started {
			started = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "---") {
			break
		}
		sectionLines = append(sectionLines, line)
	}

	currentPoint := ""
	for _, line := range sectionLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if currentPoint != "" {
				points = append(points, strings.TrimSpace(currentPoint))
				currentPoint = ""
			}
			continue
		}
		isBoldTitle := strings.HasPrefix(trimmed, "**") && strings.Contains(trimmed, ".")
		if isBoldTitle {
			if currentPoint != "" {
				points = append(points, strings.TrimSpace(currentPoint))
			}
			title := strings.ReplaceAll(trimmed, "**", "")
			currentPoint = title
		} else if currentPoint != "" {
			currentPoint += " " + trimmed
		}
	}
	if currentPoint != "" {
		points = append(points, strings.TrimSpace(currentPoint))
	}

	return points
}

// extractImprovements 提取改进建议章节
func extractImprovements(content string) []map[string]interface{} {
	var improvements []map[string]interface{}

	sectionStart := -1
	sectionHeaders := []string{
		"可以更好", "改进建议", "需要改进", "提升空间",
		"建议改进", "不足之处", "待改进",
	}
	for _, header := range sectionHeaders {
		idx := strings.Index(content, header)
		if idx >= 0 {
			sectionStart = idx
			break
		}
	}
	if sectionStart < 0 {
		return improvements
	}

	sectionContent := content[sectionStart:]
	lines := strings.Split(sectionContent, "\n")
	var sectionLines []string
	started := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !started {
			started = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "---") {
			break
		}
		sectionLines = append(sectionLines, line)
	}

	currentIssue := ""
	currentDetail := ""
	issueCount := 0
	for _, line := range sectionLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if currentIssue != "" {
				issueCount++
				improvements = append(improvements, map[string]interface{}{
					"id":         fmt.Sprintf("imp_%d", issueCount),
					"issue":      currentIssue,
					"suggestion": strings.TrimSpace(currentDetail),
				})
				currentIssue = ""
				currentDetail = ""
			}
			continue
		}
		isBoldTitle := strings.HasPrefix(trimmed, "**") && strings.Contains(trimmed, ".")
		if isBoldTitle {
			if currentIssue != "" {
				issueCount++
				improvements = append(improvements, map[string]interface{}{
					"id":         fmt.Sprintf("imp_%d", issueCount),
					"issue":      currentIssue,
					"suggestion": strings.TrimSpace(currentDetail),
				})
			}
			currentIssue = strings.ReplaceAll(trimmed, "**", "")
			currentDetail = ""
		} else if currentIssue != "" {
			currentDetail += " " + trimmed
		}
	}
	if currentIssue != "" {
		issueCount++
		improvements = append(improvements, map[string]interface{}{
			"id":         fmt.Sprintf("imp_%d", issueCount),
			"issue":      currentIssue,
			"suggestion": strings.TrimSpace(currentDetail),
		})
	}

	return improvements
}

// extractSummary 提取总评/综述内容
func extractSummary(content string) string {
	sectionStart := -1
	sectionHeaders := []string{"总评", "综述", "整体评价", "综合评价"}
	for _, header := range sectionHeaders {
		idx := strings.Index(content, header)
		if idx >= 0 {
			sectionStart = idx
			break
		}
	}
	if sectionStart < 0 {
		return ""
	}

	sectionContent := content[sectionStart:]
	lines := strings.Split(sectionContent, "\n")
	var summaryLines []string
	started := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !started {
			started = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "---") {
			break
		}
		if strings.HasPrefix(trimmed, "**总分") {
			break
		}
		// v169：遇到 JSON 围栏停止（避免把尾部 JSON 块抠进 summary）
		if strings.HasPrefix(trimmed, "```") {
			break
		}
		if trimmed != "" {
			summaryLines = append(summaryLines, trimmed)
		}
	}

	return strings.TrimSpace(strings.Join(summaryLines, " "))
}

// extractScoreFromText 从文本中提取特定关键词后的分数
func extractScoreFromText(text string, keywords []string) float64 {
	for _, kw := range keywords {
		idx := strings.Index(text, kw)
		if idx == -1 {
			continue
		}
		after := text[idx+len(kw):]
		runes := []rune(after)
		ri := 0
		for ri < len(runes) {
			r := runes[ri]
			if r == ':' || r == '：' || r == ' ' || r == '\t' ||
				r == '(' || r == ')' || r == '（' || r == '）' {
				ri++
				continue
			}
			break
		}
		if ri >= len(runes) {
			continue
		}
		numStr := ""
		for j := ri; j < len(runes); j++ {
			r := runes[j]
			if (r >= '0' && r <= '9') || r == '.' {
				numStr += string(r)
			} else {
				break
			}
		}
		if numStr == "" {
			continue
		}
		var score float64
		if _, err := fmt.Sscanf(numStr, "%f", &score); err == nil && score > 0 && score <= 10 {
			return score
		}
	}
	return 0
}

// extractGenericStageFromNatural 通用阶段从自然语言中提取
func extractGenericStageFromNatural(stageCode string, content string) (string, string, bool) {
	if strings.TrimSpace(content) == "" {
		return "{}", "", false
	}
	narrative := safeUTF8Truncate(content, 500)
	structured := map[string]interface{}{"stage": stageCode, "summary": narrative}
	b, _ := json.Marshal(structured)
	return string(b), narrative, true
}
