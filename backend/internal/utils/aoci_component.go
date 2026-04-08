package utils

// aoci_component.go — 组件AOCI索引编码映射+解析+格式化工具
//
// v83新增：P1-1 组件AOCI索引体系
//
// 包含：
//   - 编码映射表（LT/SJ/CG/TM/PQ的编码↔含义映射）
//   - ParseIndexField：从索引文本中提取指定编码字段的值
//   - ParseCognitiveLevel / ParseStageTiming / ParsePedagogyIntensity：提取冗余列值
//   - FormatIndexForPrompt：将索引文本格式化为AI注入格式（L2压缩摘要）
//   - BuildComponentFullText：拼接组件全文供AI压缩读取

import (
	"fmt"
	"strconv"
	"strings"
)

// ==================== 编码映射表 ====================

// LibraryTypeCodeMap 库类型 → 2字母编码
var LibraryTypeCodeMap = map[string]string{
	"curriculum_standard":  "CS",
	"knowledge_graph":      "KG",
	"student_profile":      "SP",
	"pedagogy":             "PD",
	"assessment_strategy":  "AS",
	"activity_design":      "AD",
	"questioning_strategy": "QS",
	"cross_subject":        "XS",
	"teaching_tool":        "TT",
	"scenario_material":    "SM",
	"quality_rubric":       "QR",
	"design_defect":        "DD",
	"review_rubric":        "RR",
}

// SubjectCodeMap 学科 → 2字母编码
var SubjectCodeMap = map[string]string{
	"general":    "GN",
	"数学":       "MA",
	"语文":       "CN",
	"英语":       "EN",
	"科学":       "SC",
	"AI":         "AI",
	"历史":       "HI",
	"道德与法治": "PO",
	"物理":       "PH",
	"地理":       "GE",
	"美术":       "AR",
	"化学":       "CH",
	"体育":       "PE",
	"音乐":       "MU",
}

// CognitiveLevelNameMap 认知层级数字 → 中文名
var CognitiveLevelNameMap = map[int]string{
	1: "记忆", 2: "理解", 3: "应用",
	4: "分析", 5: "评价", 6: "创造",
}

// StageTimingNameMap 课堂时机数字 → 中文名
var StageTimingNameMap = map[int]string{
	1: "开场", 2: "主环节", 3: "收尾", 4: "贯穿",
}

// PedagogyIntensityNameMap 教法强度数字 → 中文名
var PedagogyIntensityNameMap = map[int]string{
	1: "通用", 2: "特定", 3: "专精",
}

// ==================== 索引解析函数 ====================

// ParseIndexField 从索引文本第一行中提取指定编码字段的值
// 索引第一行格式: "LT:PD|SJ:MA|GR:7-9|CG:4|TM:2|PQ:3"
// 调用示例: ParseIndexField(indexText, "CG") → "4"
func ParseIndexField(indexText string, fieldCode string) string {
	if indexText == "" {
		return ""
	}
	// 取第一行（编码行）
	firstLine := indexText
	if idx := strings.Index(indexText, "\n"); idx >= 0 {
		firstLine = indexText[:idx]
	}
	firstLine = strings.TrimSpace(firstLine)

	// 按|分割，查找目标字段
	prefix := fieldCode + ":"
	for _, seg := range strings.Split(firstLine, "|") {
		seg = strings.TrimSpace(seg)
		if strings.HasPrefix(seg, prefix) {
			return strings.TrimSpace(seg[len(prefix):])
		}
	}
	return ""
}

// ParseCognitiveLevel 从索引文本提取认知层级数值（0=未索引，1-6有效值）
func ParseCognitiveLevel(indexText string) int {
	val := ParseIndexField(indexText, "CG")
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 || n > 6 {
		return 0
	}
	return n
}

// ParseStageTiming 从索引文本提取课堂时机数值（0=未索引，1-4有效值）
func ParseStageTiming(indexText string) int {
	val := ParseIndexField(indexText, "TM")
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 || n > 4 {
		return 0
	}
	return n
}

// ParsePedagogyIntensity 从索引文本提取教法强度数值（0=未索引，1-3有效值）
func ParsePedagogyIntensity(indexText string) int {
	val := ParseIndexField(indexText, "PQ")
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 || n > 3 {
		return 0
	}
	return n
}

// ==================== AI注入格式化 ====================

// FormatIndexForPrompt 将组件索引文本格式化为AI注入格式（L2压缩摘要）
// 输入: 完整索引文本 + 组件展示名称
// 输出: 用于注入AI system prompt的压缩文本（约150-200字）
//
// 示例输出:
//   ▸ PD-MA-79 🎯 探究式教学——问题链驱动
//     [F]用于新概念深化,通过问题链驱动探究
//     [T]方程求解,几何证明 [P]探究式,问题链
func FormatIndexForPrompt(indexText string, displayLabel string) string {
	if indexText == "" {
		// 未索引的组件，返回简化格式
		return fmt.Sprintf("▸ %s（未索引）", displayLabel)
	}

	// 提取编码行的关键部分构造简短标识
	lt := ParseIndexField(indexText, "LT")
	sj := ParseIndexField(indexText, "SJ")
	gr := ParseIndexField(indexText, "GR")
	shortCode := fmt.Sprintf("%s-%s-%s", lt, sj, gr)

	// 提取语义标签行
	lines := strings.Split(indexText, "\n")
	var tagLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// 跳过编码行（第一行）
		if strings.Contains(trimmed, "LT:") && strings.Contains(trimmed, "|") {
			continue
		}
		// 收集语义标签行 [F] [T] [P] [D] [C]
		if len(trimmed) >= 3 && trimmed[0] == '[' && trimmed[2] == ']' {
			tagLines = append(tagLines, trimmed)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("▸ %s %s\n", shortCode, displayLabel))
	for _, tl := range tagLines {
		sb.WriteString(fmt.Sprintf("  %s\n", tl))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ==================== 组件全文构建（供AI压缩读取）====================

// BuildComponentFullText 拼接组件的全部内容字段，供AI压缩索引时读取
// 按优先级排列：display_label > design_logic > full_guide > example_snippet
func BuildComponentFullText(libraryType, subject, gradeRange, displayLabel, designLogic, fullGuide, exampleSnippet string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("库类型=%s\n", libraryType))
	sb.WriteString(fmt.Sprintf("学科=%s\n", subject))
	sb.WriteString(fmt.Sprintf("学段=%s\n", gradeRange))
	sb.WriteString(fmt.Sprintf("名称=%s\n", displayLabel))

	if strings.TrimSpace(designLogic) != "" {
		sb.WriteString(fmt.Sprintf("设计逻辑=%s\n", designLogic))
	}
	if strings.TrimSpace(fullGuide) != "" {
		guide := fullGuide
		// 截断过长的全指引，保留前1500字供AI阅读
		if len([]rune(guide)) > 1500 {
			guide = string([]rune(guide)[:1500]) + "...(已截断)"
		}
		sb.WriteString(fmt.Sprintf("完整指引=%s\n", guide))
	}
	if strings.TrimSpace(exampleSnippet) != "" {
		sb.WriteString(fmt.Sprintf("参考案例=%s\n", exampleSnippet))
	}
	return sb.String()
}
