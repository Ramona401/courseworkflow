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
// 包含：
//   - ExtractStructuredFromNaturalReply：从自然语言回复中提取结构化数据（v75）
//   - DetectLessonPlanContent：检测教案Markdown内容（v75）
//   - extractScoreFromText：提取评审分数（v75）
//   - 评审信息提取：extractReviewStageFromNatural等（v77重写，v169增强）

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

// extractWriteStageFromNatural 从write/revise阶段的自然语言回复中提取教案内容
func extractWriteStageFromNatural(content string) (string, string, bool) {
	lessonContent := DetectLessonPlanContent(content)
	if lessonContent == "" {
		narrative := safeUTF8Truncate(content, 500)
		return "{}", narrative, false
	}
	structured := map[string]interface{}{"content_markdown": lessonContent}
	b, _ := json.Marshal(structured)
	narrativeIdx := strings.Index(content, lessonContent)
	narrative := ""
	if narrativeIdx > 0 {
		narrative = strings.TrimSpace(content[:narrativeIdx])
	}
	if narrative == "" {
		narrative = fmt.Sprintf("已生成教案（%d字符）", len(lessonContent))
	}
	wsLog.Info("从自然语言回复中提取到教案内容", "content_len", len(lessonContent), "narrative_len", len(narrative))
	return string(b), narrative, true
}

// DetectLessonPlanContent 检测并提取AI回复中的完整教案Markdown内容
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
	startIdx := -1
	titleMarkers := []string{
		"教案", "教学设计", "教学目标", "课题", "课时",
		"教学重点", "教学难点", "教学重难点", "教学准备",
	}
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
	if startIdx < 0 {
		return ""
	}

	// 教学过程：支持多种命名方式
	hasProcess := strings.Contains(content, "教学过程") ||
		strings.Contains(content, "教学环节") ||
		strings.Contains(content, "教学活动")
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

// ==================== 评审信息提取（v77重写，v169增强：JSON优先+正则降级）====================

// extractReviewStageFromNatural 从review阶段的自然语言回复中提取评审信息
//
// v169双链路：
//
//	链路1（优先）：解析 AI 回复尾部的 ```json 结构化块（新版提示词约定输出此块）
//	链路2（降级）：解析失败时，沿用原有 Markdown 正则提取（兼容旧格式）
//
// 两条链路任一成功（total_score>0）即返回 hasContent=true；
// 都失败时返回 hasContent=false，narrative 兜底为对话原文截断（供上层 fallback）。
func extractReviewStageFromNatural(content string) (string, string, bool) {
	// ---------- 链路1：JSON 块优先 ----------
	if structuredJSON, ok := parseReviewJSONBlock(content); ok {
		// narrative 用完整原文（含 Markdown 报告），截断到 2000 字符供前端展示/记忆
		narrative := safeUTF8Truncate(content, 2000)
		wsLog.Info("评审提取走JSON块链路（v169）", "structured_len", len(structuredJSON))
		return structuredJSON, narrative, true
	}

	// ---------- 链路2：Markdown 正则降级 ----------
	totalScore := extractTotalScoreFromReview(content)
	if totalScore <= 0 {
		narrative := safeUTF8Truncate(content, 2000)
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
	narrative := safeUTF8Truncate(content, 2000)

	wsLog.Info("评审提取走Markdown正则降级链路（v169）",
		"total_score", totalScore,
		"dimensions_count", len(dimensions),
		"good_points_count", len(goodPoints),
		"improvements_count", len(improvements),
		"summary_len", len(summary),
	)

	return string(b), narrative, true
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
