package utils

// grade_normalize.go — 中文年级名统一转换工具
//
// v82新增，v97重构：拆分为独立转换函数降低认知复杂度
//
// 转换规则（按优先级）：
//   1. 含横杠+数字（如"3-6"）→ 已是范围格式，直接返回
//   2. 含阿拉伯数字 → 提取数字部分
//   3. 小学段别名 → 范围格式（"小学低段"→"1-2"）
//   4. 初高中别名 → 转换（"初一"→"7"）
//   5. 中文数字 → 转换（"三年级"→"3"）
//   6. 无法识别 → 返回原值

import "strings"

// NormalizeGradeToNumber 将中文年级名转换为数字格式
func NormalizeGradeToNumber(grade string) string {
	if strings.TrimSpace(grade) == "" {
		return grade
	}

	// 步骤1：含横杠+数字，已是范围格式，直接返回
	if result, ok := tryParseRangeFormat(grade); ok {
		return result
	}

	// 步骤2：提取纯阿拉伯数字
	if result, ok := tryExtractDigits(grade); ok {
		return result
	}

	// 步骤3：小学段别名
	if result, ok := tryMatchSegmentAlias(grade); ok {
		return result
	}

	// 步骤4：初高中别名
	if result, ok := tryMatchSchoolAlias(grade); ok {
		return result
	}

	// 步骤5：中文数字
	if result, ok := tryMatchChineseNumber(grade); ok {
		return result
	}

	// 步骤6：无法识别，返回原值
	return grade
}

// tryParseRangeFormat 检查是否为含横杠的范围格式（如"3-6"）
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

// tryMatchSegmentAlias 匹配小学段别名（返回范围格式）
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

// tryMatchChineseNumber 匹配中文数字（先长匹配后短匹配）
func tryMatchChineseNumber(grade string) (string, bool) {
	// 先匹配长的"十一""十二"，防止"十"提前命中
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
