package services

// workshop_stage_extract.go — 阶段产出物自然语言提取
//
// v76拆分自 workshop_stage_prompts.go
// v77修复：extractReviewStageFromNatural 全面重写
// v82清理：删除废弃函数
// v84拆分：GenerateStageSummary及相关摘要生成函数 移至 workshop_stage_summary.go
//
// 包含：
//   - ExtractStructuredFromNaturalReply：从自然语言回复中提取结构化数据（v75）
//   - DetectLessonPlanContent：检测教案Markdown内容（v75）
//   - extractScoreFromText：提取评审分数（v75）
//   - 评审信息提取：extractReviewStageFromNatural等（v77）

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
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

// ==================== 评审信息提取（v77重写）====================

// extractReviewStageFromNatural 从review阶段的自然语言回复中提取评审信息
func extractReviewStageFromNatural(content string) (string, string, bool) {
	totalScore := extractTotalScoreFromReview(content)
	if totalScore <= 0 {
		narrative := safeUTF8Truncate(content, 500)
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

	wsLog.Info("从评审报告中提取结构化数据",
		"total_score", totalScore,
		"dimensions_count", len(dimensions),
		"good_points_count", len(goodPoints),
		"improvements_count", len(improvements),
		"summary_len", len(summary),
	)

	return string(b), narrative, true
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

		dimName := strings.TrimSpace(cleanCells[0])
		scoreStr := strings.TrimSpace(cleanCells[1])
		comment := strings.TrimSpace(cleanCells[2])

		if dimName == "维度" || strings.Contains(dimName, "评分") {
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
		codeRegex := regexp.MustCompile(`^(T\d+)\s+(.+)$`)
		codeMatches := codeRegex.FindStringSubmatch(dimName)
		if len(codeMatches) == 3 {
			code = codeMatches[1]
			name = codeMatches[2]
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
