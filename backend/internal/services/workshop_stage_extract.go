package services

// workshop_stage_extract.go — 阶段产出物自然语言提取 + 废弃兼容函数
//
// v76拆分自 workshop_stage_prompts.go
// v77修复：extractReviewStageFromNatural 全面重写，支持：
//   - 从Markdown表格中提取维度评分
//   - 提取"做得好的点"(good_points)
//   - 提取"改进建议"(improvements)
//   - 提取总评(summary)
//
// 包含：
//   - ExtractStructuredFromNaturalReply：从自然语言回复中提取结构化数据（v75）
//   - DetectLessonPlanContent：检测教案Markdown内容（v75）
//   - extractScoreFromText：提取评审分数（v75）
//   - ParseStageOutput / DetectStageComplete / CleanStageMarkers：废弃函数（v75保留签名）

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
	}
	markerCount := 0
	for _, marker := range lessonMarkers {
		if strings.Contains(content, marker) {
			markerCount++
		}
	}
	// v76修复：命中数从3提升到5，防止AI分段输出教案时部分内容被误判为完整教案
	if markerCount < 5 {
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

	// v76修复：完整性检查
	hasProcess := strings.Contains(content, "教学过程")
	hasEnding := strings.Contains(content, "作业布置") || strings.Contains(content, "板书设计")
	if !hasProcess || !hasEnding {
		return ""
	}

	lessonLines := lines[startIdx:]
	result := strings.TrimSpace(strings.Join(lessonLines, "\n"))
	result = trimTrailingChatter(result)
	if len(result) < 2000 {
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
//
// v77重写：全面解析AI评审报告的Markdown格式，提取：
//   - total_score: 总分（从"总分：X.X / 10"格式提取）
//   - dimensions: 从Markdown表格中提取各维度评分和简评
//   - good_points: 从"做得好的点"章节提取
//   - improvements: 从改进建议章节提取（issue + suggestion格式）
//   - summary: 从总评段落提取
func extractReviewStageFromNatural(content string) (string, string, bool) {
	// 1. 提取总分
	totalScore := extractTotalScoreFromReview(content)
	if totalScore <= 0 {
		narrative := safeUTF8Truncate(content, 500)
		return "{}", narrative, false
	}

	// 2. 从Markdown表格提取维度评分
	dimensions := extractDimensionsFromTable(content)

	// 3. 提取"做得好的点"
	goodPoints := extractGoodPoints(content)

	// 4. 提取改进建议
	improvements := extractImprovements(content)

	// 5. 提取总评/综述
	summary := extractSummary(content)

	// 构建结构化JSON
	structured := map[string]interface{}{
		"total_score":  totalScore,
		"dimensions":   dimensions,
		"good_points":  goodPoints,
		"improvements": improvements,
		"summary":      summary,
	}
	b, _ := json.Marshal(structured)

	// narrative保留完整内容（截断到合理长度）
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
// 支持格式："总分：8.6 / 10"、"总分：8.6/10"、"总评分: 8.6"等
func extractTotalScoreFromReview(content string) float64 {
	// 优先匹配"总分：X.X / 10"或"总分：X.X/10"格式
	totalPatterns := []string{
		"总分", "总评分", "综合评分", "综合得分", "总体评分",
	}
	return extractScoreFromText(content, totalPatterns)
}

// extractDimensionsFromTable 从Markdown表格中提取维度评分
// 支持格式：| T1 教学目标 | 8.5 | 三维目标完整 |
func extractDimensionsFromTable(content string) []map[string]interface{} {
	var dimensions []map[string]interface{}

	lines := strings.Split(content, "\n")
	// 正则匹配表格行：| 维度名 | 分数 | 简评 |
	// 分数可能是 8.5 或 8.5/10 等格式
	scoreRegex := regexp.MustCompile(`(\d+\.?\d*)`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过表头分隔行（|---|---|---|）
		if strings.Contains(trimmed, "---") {
			continue
		}
		// 必须是表格行（以|开头）
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}

		cells := strings.Split(trimmed, "|")
		// 去掉首尾空cell（因为 |xxx|yyy| 分割后首尾是空字符串）
		var cleanCells []string
		for _, c := range cells {
			c = strings.TrimSpace(c)
			if c != "" {
				cleanCells = append(cleanCells, c)
			}
		}

		// 至少需要3列：维度名、分数、简评
		if len(cleanCells) < 3 {
			continue
		}

		dimName := strings.TrimSpace(cleanCells[0])
		scoreStr := strings.TrimSpace(cleanCells[1])
		comment := strings.TrimSpace(cleanCells[2])

		// 跳过表头行（"维度"、"评分"等关键词）
		if dimName == "维度" || strings.Contains(dimName, "评分") {
			continue
		}

		// 提取分数
		matches := scoreRegex.FindStringSubmatch(scoreStr)
		if len(matches) < 2 {
			continue
		}
		var score float64
		if _, err := fmt.Sscanf(matches[1], "%f", &score); err != nil || score <= 0 || score > 10 {
			continue
		}

		// 提取维度代码（如 "T1 教学目标" -> code="T1", name="教学目标"）
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
// 查找"做得好"相关标题后的编号条目
func extractGoodPoints(content string) []string {
	var points []string

	// 找到"做得好的点"章节的起始位置
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

	// 从章节开始位置向后扫描，提取编号条目（**1. xxx**、**2. xxx**等）
	sectionContent := content[sectionStart:]

	// 找到下一个 ## 标题作为章节结束
	lines := strings.Split(sectionContent, "\n")
	var sectionLines []string
	started := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !started {
			started = true
			continue // 跳过标题行本身
		}
		// 遇到下一个二级标题，结束
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "---") {
			break
		}
		sectionLines = append(sectionLines, line)
	}

	// 解析编号条目：**1. xxx** 或 **xxx** 作为标题，后面的行作为详情
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
		// 检测是否是新的编号条目标题（**数字. xxx** 格式）
		isBoldTitle := strings.HasPrefix(trimmed, "**") && strings.Contains(trimmed, ".")
		if isBoldTitle {
			if currentPoint != "" {
				points = append(points, strings.TrimSpace(currentPoint))
			}
			// 去掉**标记，提取标题文字
			title := strings.ReplaceAll(trimmed, "**", "")
			currentPoint = title
		} else if currentPoint != "" {
			// 追加详情到当前条目
			currentPoint += " " + trimmed
		}
	}
	if currentPoint != "" {
		points = append(points, strings.TrimSpace(currentPoint))
	}

	return points
}

// extractImprovements 提取改进建议章节
// 返回 [{id, issue, suggestion}] 格式
func extractImprovements(content string) []map[string]interface{} {
	var improvements []map[string]interface{}

	// 找到改进建议章节
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

	// 解析编号条目
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
	// 找到"总评"或"综述"章节
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
		// 遇到分隔线、下一个二级标题、或"总分"行结束
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

// ==================== 废弃函数（保留签名，兼容性需要）====================

// ParseStageOutput v75废弃：AI不再输出<stage_output>标签
func ParseStageOutput(content string) (structuredJSON string, narrativeText string, found bool) {
	startTag := "<stage_output>"
	endTag := "</stage_output>"
	startIdx := strings.Index(content, startTag)
	if startIdx == -1 {
		return "", "", false
	}
	endIdx := strings.Index(content, endTag)
	if endIdx == -1 || endIdx <= startIdx {
		return "", "", false
	}
	rawJSON := strings.TrimSpace(content[startIdx+len(startTag) : endIdx])
	if rawJSON == "" {
		return "", "", false
	}
	narrativeText = strings.TrimSpace(content[:startIdx])
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err == nil {
		structuredData := "{}"
		if s, ok := parsed["structured"]; ok {
			if b, err2 := json.Marshal(s); err2 == nil {
				structuredData = string(b)
			}
		}
		if narrativeText == "" {
			if n, ok := parsed["narrative"]; ok {
				if ns, ok := n.(string); ok {
					narrativeText = ns
				}
			}
		}
		return structuredData, narrativeText, true
	}
	wsLog.Warn("stage_output JSON解析失败，启用容错提取", "raw_len", len(rawJSON))
	structuredData := extractStructuredFallback(rawJSON)
	if structuredData == "{}" {
		wsLog.Warn("括号计数法也失败，启用逐字段提取", "raw_len", len(rawJSON))
		structuredData = extractStructuredByFields(rawJSON)
	}
	return structuredData, narrativeText, true
}

// extractStructuredFallback 策略2：括号计数法提取structured字段
func extractStructuredFallback(rawJSON string) string {
	key := `"structured":`
	idx := strings.Index(rawJSON, key)
	if idx == -1 {
		return "{}"
	}
	rest := rawJSON[idx+len(key):]
	start := strings.Index(rest, "{")
	if start == -1 {
		return "{}"
	}
	absStart := idx + len(key) + start
	depth := 0
	inString := false
	escaped := false
	for i := absStart; i < len(rawJSON); i++ {
		c := rawJSON[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				extracted := rawJSON[absStart : i+1]
				var test map[string]interface{}
				if err := json.Unmarshal([]byte(extracted), &test); err == nil {
					result, _ := json.Marshal(test)
					return string(result)
				}
				return extracted
			}
		}
	}
	return "{}"
}

// extractStructuredByFields 策略3：逐字段提取重组structured
func extractStructuredByFields(rawJSON string) string {
	if strings.Contains(rawJSON, `"content_markdown"`) {
		cm := extractFieldFromText(rawJSON, "content_markdown")
		if cm != "" {
			result := map[string]interface{}{"content_markdown": cm}
			if strings.Contains(rawJSON, `"content_structured"`) {
				csKey := `"content_structured":`
				csIdx := strings.Index(rawJSON, csKey)
				if csIdx >= 0 {
					csRest := rawJSON[csIdx+len(csKey):]
					csStart := strings.Index(csRest, "{")
					if csStart >= 0 {
						depth := 0
						for i := csStart; i < len(csRest); i++ {
							if csRest[i] == '{' {
								depth++
							} else if csRest[i] == '}' {
								depth--
								if depth == 0 {
									var cs map[string]interface{}
									if err := json.Unmarshal([]byte(csRest[csStart:i+1]), &cs); err == nil {
										result["content_structured"] = cs
									}
									break
								}
							}
						}
					}
				}
			}
			b, _ := json.Marshal(result)
			wsLog.Info("逐字段提取write阶段structured成功", "content_len", len(cm))
			return string(b)
		}
	}
	if strings.Contains(rawJSON, `"textbook_analysis"`) {
		ta := extractFieldFromText(rawJSON, "textbook_analysis")
		if ta != "" {
			result := map[string]interface{}{"textbook_analysis": ta}
			if strings.Contains(rawJSON, `"key_concepts"`) {
				kcIdx := strings.Index(rawJSON, `"key_concepts"`)
				if kcIdx >= 0 {
					kcRest := rawJSON[kcIdx+len(`"key_concepts"`):]
					arrStart := strings.Index(kcRest, "[")
					if arrStart >= 0 {
						depth := 0
						for i := arrStart; i < len(kcRest); i++ {
							if kcRest[i] == '[' {
								depth++
							} else if kcRest[i] == ']' {
								depth--
								if depth == 0 {
									var kc []interface{}
									if err := json.Unmarshal([]byte(kcRest[arrStart:i+1]), &kc); err == nil {
										result["key_concepts"] = kc
									}
									break
								}
							}
						}
					}
				}
			}
			b, _ := json.Marshal(result)
			return string(b)
		}
	}
	if strings.Contains(rawJSON, `"strategy"`) {
		st := extractFieldFromText(rawJSON, "strategy")
		if st != "" {
			result := map[string]interface{}{"strategy": st}
			b, _ := json.Marshal(result)
			return string(b)
		}
	}
	wsLog.Warn("逐字段提取也失败，返回{}", "raw_len", len(rawJSON))
	return "{}"
}

// DetectStageComplete v75废弃
func DetectStageComplete(content string) bool {
	return strings.Contains(content, "<stage_complete/>") || strings.Contains(content, "<stage_complete />")
}

// CleanStageMarkers v75废弃
func CleanStageMarkers(content string) string {
	startTag := "<stage_output>"
	endTag := "</stage_output>"
	for {
		startIdx := strings.Index(content, startTag)
		if startIdx == -1 {
			break
		}
		endIdx := strings.Index(content, endTag)
		if endIdx == -1 {
			break
		}
		content = content[:startIdx] + content[endIdx+len(endTag):]
	}
	content = strings.ReplaceAll(content, "<stage_complete/>", "")
	content = strings.ReplaceAll(content, "<stage_complete />", "")
	return strings.TrimSpace(content)
}

// ==================== extractFieldFromText 辅助函数 ====================

func extractFieldFromText(text, fieldName string) string {
	key := `"` + fieldName + `"`
	keyIdx := strings.Index(text, key)
	if keyIdx == -1 {
		return ""
	}
	afterKey := text[keyIdx+len(key):]
	colonIdx := strings.Index(afterKey, ":")
	if colonIdx == -1 {
		return ""
	}
	afterColon := afterKey[colonIdx+1:]
	valueStart := -1
	for i, ch := range afterColon {
		if ch == '"' {
			valueStart = i
			break
		}
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			return ""
		}
	}
	if valueStart == -1 {
		return ""
	}
	content2 := afterColon[valueStart+1:]
	var sb strings.Builder
	i := 0
	for i < len(content2) {
		b := content2[i]
		if b == '\\' && i+1 < len(content2) {
			next := content2[i+1]
			switch next {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			case '/':
				sb.WriteByte('/')
			case 'u':
				if i+5 < len(content2) {
					hex := content2[i+2 : i+6]
					var codePoint rune
					fmt.Sscanf(hex, "%04x", &codePoint)
					sb.WriteRune(codePoint)
					i += 6
					continue
				}
				sb.WriteByte(next)
			default:
				sb.WriteByte(next)
			}
			i += 2
			continue
		}
		if b == '"' {
			rest := content2[i+1:]
			rest = strings.TrimLeft(rest, " \t\r\n")
			if len(rest) == 0 {
				break
			}
			if rest[0] == ',' || rest[0] == '}' || rest[0] == ']' {
				break
			}
			if rest[0] == '"' {
				nextQuote := strings.Index(rest[1:], `"`)
				if nextQuote >= 0 {
					afterNextKey := strings.TrimLeft(rest[1+nextQuote+1:], " \t")
					if len(afterNextKey) > 0 && afterNextKey[0] == ':' {
						break
					}
				}
			}
			if strings.HasPrefix(rest, "</") || strings.HasPrefix(rest, "<stage") {
				break
			}
			sb.WriteByte(b)
			i++
			continue
		}
		sb.WriteByte(b)
		i++
	}
	return strings.TrimSpace(sb.String())
}
