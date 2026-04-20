package utils

// grade_normalize.go — 中文年级名统一转换工具
//
// v82新增，v97重构：拆分为独立转换函数降低认知复杂度
// v115新增(2026-04-20 AI助手学段匹配):
//   新增 NormalizeGradeToSegment —— 将任意年级输入归一化为学段("小学"/"初中"/"高中"/"")
//   用于 AI 助手 grade_range 的学段级匹配(避免前端传"七年级"匹配不上"初中"的问题)
//
// 转换规则(NormalizeGradeToNumber,按优先级):
//   1. 含横杠+数字(如"3-6")→ 已是范围格式，直接返回
//   2. 含阿拉伯数字 → 提取数字部分
//   3. 小学段别名 → 范围格式("小学低段"→"1-2")
//   4. 初高中别名 → 转换("初一"→"7")
//   5. 中文数字 → 转换("三年级"→"3")
//   6. 无法识别 → 返回原值
//
// NormalizeGradeToSegment 规则:先调 NormalizeGradeToNumber 取得数字/范围,
//   再映射到学段(1-6=小学 / 7-9=初中 / 10-12=高中),
//   对于"小学"/"初中"/"高中"文本输入直接识别,空串返回空串

import (
	"strconv"
	"strings"
)

// NormalizeGradeToNumber 将中文年级名转换为数字格式
func NormalizeGradeToNumber(grade string) string {
	if strings.TrimSpace(grade) == "" {
		return grade
	}

	// 步骤1:含横杠+数字,已是范围格式,直接返回
	if result, ok := tryParseRangeFormat(grade); ok {
		return result
	}

	// 步骤2:提取纯阿拉伯数字
	if result, ok := tryExtractDigits(grade); ok {
		return result
	}

	// 步骤3:小学段别名
	if result, ok := tryMatchSegmentAlias(grade); ok {
		return result
	}

	// 步骤4:初高中别名
	if result, ok := tryMatchSchoolAlias(grade); ok {
		return result
	}

	// 步骤5:中文数字
	if result, ok := tryMatchChineseNumber(grade); ok {
		return result
	}

	// 步骤6:无法识别,返回原值
	return grade
}

// tryParseRangeFormat 检查是否为含横杠的范围格式(如"3-6")
func tryParseRangeFormat(grade string) (string, bool) {
	if !strings.Contains(grade, "-") {
		return "", false
	}
	for _, b := range []byte(grade) {
		if b >= '0' && b <= '9' {
			return grade, true
		}
	}
	return "", false
}

// tryExtractDigits 提取字符串中的纯阿拉伯数字部分
func tryExtractDigits(grade string) (string, bool) {
	var digits []byte
	for _, b := range []byte(grade) {
		if b >= '0' && b <= '9' {
			digits = append(digits, b)
		}
	}
	if len(digits) > 0 {
		return string(digits), true
	}
	return "", false
}

// tryMatchSegmentAlias 匹配小学段别名(返回范围格式)
func tryMatchSegmentAlias(grade string) (string, bool) {
	segmentMap := map[string]string{
		"小学低段": "1-2",
		"小学中段": "3-4",
		"小学高段": "5-6",
	}
	for seg, num := range segmentMap {
		if strings.Contains(grade, seg) {
			return num, true
		}
	}
	return "", false
}

// tryMatchSchoolAlias 匹配初高中别名
func tryMatchSchoolAlias(grade string) (string, bool) {
	aliasMap := map[string]string{
		"初一": "7", "初二": "8", "初三": "9",
		"高一": "10", "高二": "11", "高三": "12",
	}
	for alias, num := range aliasMap {
		if strings.Contains(grade, alias) {
			return num, true
		}
	}
	return "", false
}

// tryMatchChineseNumber 匹配中文数字(先长匹配后短匹配)
func tryMatchChineseNumber(grade string) (string, bool) {
	// 先匹配长的"十一""十二",防止"十"提前命中
	longMap := map[string]string{"十一": "11", "十二": "12"}
	for cn, num := range longMap {
		if strings.Contains(grade, cn) {
			return num, true
		}
	}
	shortMap := map[string]string{
		"一": "1", "二": "2", "三": "3", "四": "4", "五": "5",
		"六": "6", "七": "7", "八": "8", "九": "9", "十": "10",
	}
	for cn, num := range shortMap {
		if strings.Contains(grade, cn) {
			return num, true
		}
	}
	return "", false
}

// ==================== v115 新增:学段级归一化 ====================

// 学段常量
const (
	SegmentPrimary  = "小学" // 1-6 年级
	SegmentJunior   = "初中" // 7-9 年级
	SegmentSenior   = "高中" // 10-12 年级
	SegmentAll      = ""    // 通用(空字符串表示不限学段)
)

// NormalizeGradeToSegment 将任意年级输入归一化为学段标签
//
// 输入示例 → 输出学段:
//   "七年级"   → "初中"
//   "三年级"   → "小学"
//   "高一"     → "高中"
//   "初一"     → "初中"
//   "2"        → "小学"
//   "7-9"      → "初中"
//   "3-6"      → "小学"
//   "1-6"      → "小学"(历史数据兼容)
//   "小学"     → "小学"(直通)
//   "初中"     → "初中"(直通)
//   "高中"     → "高中"(直通)
//   ""         → ""(保持通用)
//   "小学低段" → "小学"
//   "小学高段" → "小学"
//
// 用于 AI 助手 grade_range 的学段匹配(当 grade_range 字段存"小学"/"初中"/"高中"时),
// 让前端传来的各种年级输入都能正确映射到学段。
//
// 注意:返回空字符串 "" 时表示"通用/不限学段",调用方需配合"grade_range=''或 grade_range='学段'"
// 的 SQL 逻辑做兜底匹配。
func NormalizeGradeToSegment(grade string) string {
	trimmed := strings.TrimSpace(grade)
	if trimmed == "" {
		return SegmentAll
	}

	// 优先识别直通输入(数据库里已经存的就是学段标签)
	if strings.Contains(trimmed, "小学") {
		return SegmentPrimary
	}
	if strings.Contains(trimmed, "初中") {
		return SegmentJunior
	}
	if strings.Contains(trimmed, "高中") {
		return SegmentSenior
	}

	// 先归一化为数字(含范围),再按数字映射到学段
	normalized := NormalizeGradeToNumber(trimmed)
	return rangeOrNumberToSegment(normalized)
}

// rangeOrNumberToSegment 将"数字"或"数字-数字"格式归类到学段
//
// 输入:
//   "3"   → "小学"   (1-6)
//   "7"   → "初中"   (7-9)
//   "10"  → "高中"   (10-12)
//   "7-9" → "初中"   (范围的起点决定学段)
//   "1-6" → "小学"
//   其他  → ""(无法识别)
//
// 范围跨学段时(如罕见的"5-8"),按范围起点归类。这是工程折衷,
// 现实中专业助手不应跨学段定位,跨段的助手建议直接用空 grade_range(通用)
func rangeOrNumberToSegment(s string) string {
	if s == "" {
		return SegmentAll
	}
	// 如果是范围(含横杠),取起点
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		if len(parts) > 0 {
			s = parts[0]
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return SegmentAll
	}
	switch {
	case n >= 1 && n <= 6:
		return SegmentPrimary
	case n >= 7 && n <= 9:
		return SegmentJunior
	case n >= 10 && n <= 12:
		return SegmentSenior
	default:
		return SegmentAll
	}
}
