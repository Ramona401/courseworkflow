package services

// courseware_index_json.go — 课件「层2方案翻译」AI输出的JSON解析与多重兜底
//
// 从 courseware_index_service.go 拆出，集中管理 courseware_scheme 场景专属的
// JSON 解析逻辑，保持主文件在 600 行以内。
//
// 仅迁移 courseware_index 专属函数：
//   - (*CoursewareIndexService).parseSchemeJSON  多重兜底解析入口（择优）
//   - cwExtractJSONArray   提取JSON数组（括号配对扫描，加固版）
//   - cwExtractJSONObjects 按 page_number 逐对象提取（括号配对找对象结尾，加固版）
//   - cwExtractSchemeByFields 正则逐字段宽容提取（绕开引号转义）
//   - cwTruncate           按 rune 安全截断
//
// 不迁移、不改动以下跨服务共享的公共清洗函数（仍在 courseware_index_service.go）：
//   cwStripCodeFences / cwCleanChinesePunctuation / cwFixJSONQuotes
//   —— 它们被 template_extract_service.go / template_refine_service.go 依赖，
//      改动会波及模板提取/微调功能，故本次保持原状。
//
// v0.44 修复（长文档只解析出极少页：25913字符输出却只得2页）：
//   根因——旧第④级 cwExtractJSONObjects 用 strings.Index(chunk,"}") 截到
//   「第一个 }」作为对象结尾，当某对象字段值内部含有 '}'（如 content_summary
//   写了选项/示例花括号）或内嵌引号导致该小段 Unmarshal 失败时，该对象被
//   continue 丢弃；20+对象里多数命中后只剩2个。且旧逻辑「第④级非空即 return」
//   提前出口，挡住了更稳健的第⑤级正则提取。
//   修复——
//     (a) 第④级改用括号配对扫描找对象真正的闭合 '}'（跳过字符串字面量内的括号）；
//     (b) parseSchemeJSON 在前三级失败后，第④、⑤级都执行，取条数更多者返回，
//         避免「少数成功项」挡住「多数成功项」；
//     (c) 其余逻辑、函数签名、跨服务清洗函数一律不动。

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// ==================== 层2 JSON解析（多重兜底 + 择优） ====================

// parseSchemeJSON 解析层2 AI返回的JSON数组
//
// 兜底策略（前三级任一成功即返回；第四、五级择优）：
//  1. 括号配对提取数组后直接解析
//  2. 清理中文标点后解析
//  3. 修复未转义引号后解析（cwFixJSONQuotes）
//  4. 按 page_number 逐对象提取（括号配对找对象结尾）
//  5. 正则逐字段宽容提取（绕开引号转义，处理字段值内含英文双引号的脏JSON）
//
// 关键：第四、五级都执行，取「解析出条数更多」的结果。
//
//	因为脏 JSON 下，第四级可能只救出少数干净对象，而第五级正则能救出更多；
//	旧逻辑「第四级非空即返回」会让少数项挡住多数项，导致长文档只出极少页。
func (s *CoursewareIndexService) parseSchemeJSON(aiOutput string) ([]cwSchemeItem, error) {
	text := strings.TrimSpace(aiOutput)
	text = cwStripCodeFences(text)

	jsonStr := cwExtractJSONArray(text)
	if jsonStr == "" {
		// 数组括号都找不到，仍尝试用第五重正则提取整段文本兜底
		if items := cwExtractSchemeByFields(text); len(items) > 0 {
			log.Printf("[courseware_index] 层2无JSON数组括号，正则逐字段提取成功: %d个", len(items))
			return items, nil
		}
		return nil, fmt.Errorf("层2输出中未找到JSON数组")
	}

	// 第一重：括号配对提取后直接解析
	var schemes []cwSchemeItem
	if err := json.Unmarshal([]byte(jsonStr), &schemes); err == nil && len(schemes) > 0 {
		return schemes, nil
	}

	// 第二重：清理中文标点后解析
	cleaned := cwCleanChinesePunctuation(jsonStr)
	if err := json.Unmarshal([]byte(cleaned), &schemes); err == nil && len(schemes) > 0 {
		return schemes, nil
	}

	// 第三重：修复JSON值内部的未转义引号
	fixed := cwFixJSONQuotes(cleaned)
	if err := json.Unmarshal([]byte(fixed), &schemes); err == nil && len(schemes) > 0 {
		log.Printf("[courseware_index] 层2 JSON修复后解析成功: %d个", len(schemes))
		return schemes, nil
	}

	// 第四重：按 page_number 逐对象提取（括号配对找对象结尾）
	byObjects := cwExtractJSONObjects(cleaned)

	// 第五重：正则逐字段宽容提取，绕开所有引号转义问题
	// 注意用原始 jsonStr（未删中文引号），保留字段值原貌
	byFields := cwExtractSchemeByFields(jsonStr)

	// 择优：取条数更多的结果（避免少数成功项挡住多数成功项）
	if len(byObjects) == 0 && len(byFields) == 0 {
		return nil, fmt.Errorf("层2 JSON解析失败(多重兜底均失败), 前200字: %s", cwTruncate(jsonStr, 200))
	}
	if len(byFields) > len(byObjects) {
		log.Printf("[courseware_index] 层2择优-正则逐字段提取胜出: 逐对象=%d 逐字段=%d 采用逐字段", len(byObjects), len(byFields))
		return byFields, nil
	}
	log.Printf("[courseware_index] 层2择优-逐对象提取胜出: 逐对象=%d 逐字段=%d 采用逐对象", len(byObjects), len(byFields))
	return byObjects, nil
}

// ==================== 提取JSON数组（括号配对扫描，加固版） ====================

// cwExtractJSONArray 从文本中提取JSON数组
//
// 加固说明：旧实现用 Index("[") + LastIndex("]")，当字段值内部含有
//
//	'[' 或 ']' 字符（如 "[图片]"、"标签[A]"）时，LastIndex 会截到
//	字段值内部的 ']'，导致数组边界错误。
//
// 新实现做括号配对扫描：从第一个顶层 '[' 开始，用计数器跟踪嵌套深度，
//
//	并跳过字符串字面量内部的括号（识别 \" 转义），找到与之配对的 ']'
//	作为数组真正结尾。配对失败时回退到 LastIndex 兜底。
func cwExtractJSONArray(text string) string {
	start := strings.IndexByte(text, '[')
	if start < 0 {
		return ""
	}

	depth := 0        // 方括号嵌套深度
	inString := false // 是否处于字符串字面量内部
	escaped := false  // 上一个字符是否为反斜杠转义
	for i := start; i < len(text); i++ {
		c := text[i]

		if inString {
			// 字符串内部：只关心引号结束与转义
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}

		switch c {
		case '"':
			inString = true
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				// 找到与起始 '[' 配对的 ']'
				return text[start : i+1]
			}
		}
	}

	// 配对扫描未闭合（AI输出被截断等），回退到 LastIndex 兜底
	end := strings.LastIndexByte(text, ']')
	if end > start {
		return text[start : end+1]
	}
	return ""
}

// ==================== 按 page_number 逐对象提取（第四重兜底，括号配对加固版） ====================

// cwExtractJSONObjects 逐个提取JSON对象
//
// 加固说明（v0.44 核心修复）：
//
//	旧实现用 strings.Index(chunk,"}") 取「第一个 }」作为对象结尾，
//	当对象字段值内部含有 '}'（如选项/示例花括号）时会过早截断，
//	导致 json.Unmarshal 失败、对象被丢弃。长文档下大量对象被吃掉。
//
//	新实现：按 "page_number" 关键字段定位每个对象的起点，再用括号配对
//	扫描（跳过字符串字面量内部的 { }）找到该对象真正的闭合 '}'，
//	从而正确截取完整对象。单个对象解析失败仍 continue 跳过，不影响其余。
func cwExtractJSONObjects(text string) []cwSchemeItem {
	var items []cwSchemeItem

	// 定位所有 "page_number" 出现位置作为对象起点
	const key = "\"page_number\""
	var starts []int
	searchFrom := 0
	for {
		idx := strings.Index(text[searchFrom:], key)
		if idx < 0 {
			break
		}
		starts = append(starts, searchFrom+idx)
		searchFrom = searchFrom + idx + len(key)
	}
	if len(starts) == 0 {
		return nil
	}

	for _, sp := range starts {
		// 从该 page_number 起点向前找到所属对象的 '{'
		// （通常 page_number 是对象首字段，向前回溯到最近的 '{'）
		objStart := strings.LastIndexByte(text[:sp], '{')
		if objStart < 0 {
			// 找不到前置 '{'，则以 page_number 前补一个 '{' 兜底
			objStart = sp
			objStr := cwSliceObjectByBraces("{" + text[objStart:])
			if objStr == "" {
				continue
			}
			var item cwSchemeItem
			if err := json.Unmarshal([]byte(objStr), &item); err == nil && item.Title != "" {
				items = append(items, item)
			}
			continue
		}

		// 从 objStart 的 '{' 开始括号配对，截取完整对象
		objStr := cwSliceObjectByBraces(text[objStart:])
		if objStr == "" {
			continue
		}
		var item cwSchemeItem
		if err := json.Unmarshal([]byte(objStr), &item); err == nil && item.Title != "" {
			items = append(items, item)
		}
	}
	return items
}

// cwSliceObjectByBraces 从以 '{' 开头的文本中，用括号配对扫描截取一个完整JSON对象
// 跳过字符串字面量内部的 { } 与转义引号；配对失败返回空串。
func cwSliceObjectByBraces(text string) string {
	if len(text) == 0 || text[0] != '{' {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[:i+1]
			}
		}
	}
	return ""
}

// ==================== 正则逐字段宽容提取（第五重兜底） ====================

// 字段提取正则：匹配 "字段名" : "值" 或 "字段名" : 数字
// 值采用「贪婪匹配到下一个字段名出现之前」的策略，因此字段值内部即便含有
// 未转义的英文双引号、中文引号、换行等也能完整捕获，彻底绕开 JSON 结构解析。
var (
	// 以 "page_number": 作为每个对象的起点切分
	cwReSchemeBlockSplit = regexp.MustCompile(`(?s)"page_number"\s*:`)
	// 字符串型字段：值在一对引号内，但允许值内出现引号，故匹配到
	// 「下一字段名 或 末尾 }」之前的最后一个引号
	cwReField = func(field string) *regexp.Regexp {
		return regexp.MustCompile(`(?s)"` + field + `"\s*:\s*"(.*?)"\s*(?:,\s*"(?:page_number|title|purpose|content_summary|interaction_type|visual_format|media_requirements|estimated_complexity)"|}\s*)`)
	}
	cwReIntField = func(field string) *regexp.Regexp {
		return regexp.MustCompile(`"` + field + `"\s*:\s*(-?\d+)`)
	}
)

// cwExtractSchemeByFields 用正则按固定字段名逐个对象、逐个字段提取方案
// 输入应为 JSON 数组文本（或包含若干 {…} 对象的文本）
func cwExtractSchemeByFields(text string) []cwSchemeItem {
	// 以 "page_number": 作为每个对象的起点切分
	locs := cwReSchemeBlockSplit.FindAllStringIndex(text, -1)
	if len(locs) == 0 {
		return nil
	}

	var items []cwSchemeItem
	for i := 0; i < len(locs); i++ {
		startPos := locs[i][0]
		endPos := len(text)
		if i+1 < len(locs) {
			endPos = locs[i+1][0]
		}
		block := text[startPos:endPos]

		item := cwParseSchemeBlock(block)
		if item.Title != "" || item.ContentSummary != "" || item.Purpose != "" {
			items = append(items, item)
		}
	}
	return items
}

// cwParseSchemeBlock 从单个对象文本块中逐字段提取
func cwParseSchemeBlock(block string) cwSchemeItem {
	var item cwSchemeItem

	// page_number / estimated_complexity 为整型
	if m := cwReIntField("page_number").FindStringSubmatch(block); len(m) == 2 {
		item.PageNumber, _ = strconv.Atoi(m[1])
	}
	if m := cwReIntField("estimated_complexity").FindStringSubmatch(block); len(m) == 2 {
		item.EstimatedComplexity, _ = strconv.Atoi(m[1])
	}

	// 字符串字段：取捕获组1并清洗内部多余引号
	item.Title = cwFieldValue(block, "title")
	item.Purpose = cwFieldValue(block, "purpose")
	item.ContentSummary = cwFieldValue(block, "content_summary")
	item.InteractionType = cwFieldValue(block, "interaction_type")
	item.VisualFormat = cwFieldValue(block, "visual_format")
	item.MediaRequirements = cwFieldValue(block, "media_requirements")

	return item
}

// cwFieldValue 提取单个字符串字段的值并做轻量清洗
// 清洗：去掉值内部残留的英文/中文双引号（这些是AI错误嵌入的，非结构引号），
//
//	折叠多余空白；保留其余文字原貌
func cwFieldValue(block, field string) string {
	m := cwReField(field).FindStringSubmatch(block)
	if len(m) < 2 {
		return ""
	}
	v := m[1]
	// 去掉值内部的双引号（结构引号已被正则边界吃掉，这里剩下的都是错误内嵌引号）
	v = strings.NewReplacer(`"`, "", "\u201c", "", "\u201d", "").Replace(v)
	v = strings.TrimSpace(v)
	return v
}

// ==================== 通用：按 rune 安全截断 ====================

// cwTruncate 按 rune 截断字符串，避免中文被截半
func cwTruncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
