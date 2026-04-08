package utils

// aoci_lesson.go — 教案AOCI索引编码映射+解析+格式化工具
//
// v86新增：P2-1 教案AOCI索引扩展
//
// 包含：
//   - 结构类型/质量等级编码映射
//   - ParseLessonIndexField：从教案索引文本中提取指定编码字段的值
//   - ParseStructureType / ParseQualityLevel：提取冗余列值
//   - CalculateQualityLevel：根据AI评分+状态计算质量等级
//   - FormatLessonIndexForPrompt：将教案索引格式化为检索摘要
//   - BuildLessonFullText：拼接教案全文供AI压缩读取

import (
	"fmt"
	"strconv"
	"strings"
)

// ==================== 教案索引编码映射 ====================

// StructureTypeNameMap 结构类型数字 → 中文名
var StructureTypeNameMap = map[int]string{
	1: "讲授型",
	2: "探究型",
	3: "项目型",
	4: "翻转型",
	5: "混合型",
}

// QualityLevelNameMap 质量等级数字 → 中文名
var QualityLevelNameMap = map[int]string{
	1: "草稿级",
	2: "可用级",
	3: "良好级",
	4: "优秀级",
	5: "精品级",
}

// StructureTypeCodeMap 结构类型中文 → 数字（用于从AI输出解析）
var StructureTypeCodeMap = map[string]int{
	"讲授型": 1, "探究型": 2, "项目型": 3, "翻转型": 4, "混合型": 5,
}

// ==================== 索引解析函数 ====================

// ParseLessonIndexField 从教案索引文本第一行中提取指定编码字段的值
// 索引第一行格式: "SJ:MA|GR:7|CG:4|PQ:2|ST:2|QL:4"
// 调用示例: ParseLessonIndexField(indexText, "CG") → "4"
func ParseLessonIndexField(indexText string, fieldCode string) string {
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

// ParseLessonCognitiveLevel 从教案索引提取认知层级（0=未索引，1-6有效值）
func ParseLessonCognitiveLevel(indexText string) int {
	val := ParseLessonIndexField(indexText, "CG")
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 || n > 6 {
		return 0
	}
	return n
}

// ParseLessonPedagogyIntensity 从教案索引提取教法强度（0=未索引，1-3有效值）
func ParseLessonPedagogyIntensity(indexText string) int {
	val := ParseLessonIndexField(indexText, "PQ")
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 || n > 3 {
		return 0
	}
	return n
}

// ParseStructureType 从教案索引提取结构类型（0=未索引，1-5有效值）
func ParseStructureType(indexText string) int {
	val := ParseLessonIndexField(indexText, "ST")
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 || n > 5 {
		return 0
	}
	return n
}

// ParseQualityLevel 从教案索引提取质量等级（0=未索引，1-5有效值）
func ParseQualityLevel(indexText string) int {
	val := ParseLessonIndexField(indexText, "QL")
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 || n > 5 {
		return 0
	}
	return n
}

// ==================== 质量等级计算 ====================

// CalculateQualityLevel 根据AI评审分数和教案状态计算质量等级
// 规则：
//   QL=5: AI评分≥9.0 且 状态为approved/published_shared
//   QL=4: AI评分≥8.0 或 (AI评分≥7.5 且 状态为approved)
//   QL=3: AI评分≥7.0 或 状态为approved
//   QL=2: AI评分≥5.0 或 状态为published_personal
//   QL=1: 其他（草稿/无评分）
func CalculateQualityLevel(aiScore *float64, status string) int {
	score := 0.0
	if aiScore != nil {
		score = *aiScore
	}

	isApproved := status == "approved" || status == "published_shared"
	isPersonalPublished := status == "published_personal"

	// QL=5: 精品级
	if score >= 9.0 && isApproved {
		return 5
	}
	// QL=4: 优秀级
	if score >= 8.0 || (score >= 7.5 && isApproved) {
		return 4
	}
	// QL=3: 良好级
	if score >= 7.0 || isApproved {
		return 3
	}
	// QL=2: 可用级
	if score >= 5.0 || isPersonalPublished {
		return 2
	}
	// QL=1: 草稿级
	return 1
}

// ==================== AI注入格式化 ====================

// FormatLessonIndexForPrompt 将教案索引文本格式化为检索摘要（用于推荐/展示）
// 输出格式：简短编码 + 标题 + 语义标签（约200字）
func FormatLessonIndexForPrompt(indexText string, title string) string {
	if indexText == "" {
		return fmt.Sprintf("▸ %s（未索引）", title)
	}

	// 提取编码行的关键部分
	sj := ParseLessonIndexField(indexText, "SJ")
	gr := ParseLessonIndexField(indexText, "GR")
	st := ParseLessonIndexField(indexText, "ST")
	ql := ParseLessonIndexField(indexText, "QL")
	shortCode := fmt.Sprintf("%s-%s-ST%s-QL%s", sj, gr, st, ql)

	// 提取语义标签行
	lines := strings.Split(indexText, "\n")
	var tagLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// 跳过编码行
		if strings.Contains(trimmed, "SJ:") && strings.Contains(trimmed, "|") {
			continue
		}
		// 收集语义标签行 [O] [S] [M] [H] [R]
		if len(trimmed) >= 3 && trimmed[0] == '[' && trimmed[2] == ']' {
			tagLines = append(tagLines, trimmed)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("▸ %s %s\n", shortCode, title))
	for _, tl := range tagLines {
		sb.WriteString(fmt.Sprintf("  %s\n", tl))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ==================== 教案全文构建（供AI压缩读取）====================

// BuildLessonFullText 拼接教案全部内容字段，供AI索引压缩时读取
// 包含：基本信息 + 教案正文 + 结构化内容 + AI评审结果
// contentMarkdown 截断上限3000字符，防止超长教案消耗过多token
func BuildLessonFullText(
	subject, grade, topic, title string,
	durationMinutes int,
	contentMarkdown string,
	contentStructured string,
	aiReviewResult string,
	matchedComponents string,
) string {
	var sb strings.Builder

	// 基本信息
	sb.WriteString(fmt.Sprintf("学科=%s\n", subject))
	sb.WriteString(fmt.Sprintf("年级=%s\n", grade))
	sb.WriteString(fmt.Sprintf("课题=%s\n", topic))
	sb.WriteString(fmt.Sprintf("标题=%s\n", title))
	sb.WriteString(fmt.Sprintf("时长=%d分钟\n", durationMinutes))

	// 教案正文（截断保护）
	if strings.TrimSpace(contentMarkdown) != "" {
		md := contentMarkdown
		if len([]rune(md)) > 3000 {
			md = string([]rune(md)[:3000]) + "...(已截断)"
		}
		sb.WriteString(fmt.Sprintf("教案正文=\n%s\n", md))
	}

	// 结构化内容（截断保护）
	if strings.TrimSpace(contentStructured) != "" && contentStructured != "{}" {
		cs := contentStructured
		if len([]rune(cs)) > 1500 {
			cs = string([]rune(cs)[:1500]) + "...(已截断)"
		}
		sb.WriteString(fmt.Sprintf("结构化内容=%s\n", cs))
	}

	// AI评审结果（如有，提供给AI参考亮点和不足）
	if strings.TrimSpace(aiReviewResult) != "" && aiReviewResult != "{}" {
		ar := aiReviewResult
		if len([]rune(ar)) > 800 {
			ar = string([]rune(ar)[:800]) + "...(已截断)"
		}
		sb.WriteString(fmt.Sprintf("AI评审结果=%s\n", ar))
	}

	// 匹配组件（简要参考）
	if strings.TrimSpace(matchedComponents) != "" && matchedComponents != "[]" {
		mc := matchedComponents
		if len([]rune(mc)) > 500 {
			mc = string([]rune(mc)[:500]) + "...(已截断)"
		}
		sb.WriteString(fmt.Sprintf("使用的教学组件=%s\n", mc))
	}

	return sb.String()
}

// ==================== 索引验证 ====================

// ValidateLessonIndex 验证教案索引格式是否合法
// 必须包含SJ:和至少一个语义标签[O]
func ValidateLessonIndex(indexText string) bool {
	if indexText == "" {
		return false
	}
	hasSJ := strings.Contains(indexText, "SJ:")
	hasObjective := strings.Contains(indexText, "[O]")
	return hasSJ && hasObjective
}
