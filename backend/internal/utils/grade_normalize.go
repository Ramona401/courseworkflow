package utils

// grade_normalize.go — 教学层级的基础归一化工具
//
// 本文件最初只处理K12一年级至高三。
// 当前平台已经支持三个确定性教学教育域：
//   - k12：一年级至高三；
//   - vocational：中职Ⅰ、Ⅱ、Ⅲ年级；
//   - adult：成人入门、进阶、高级、管理者。
//
// 重要边界：
//   1. NormalizeGradeToNumber继续只负责K12数字归一化；
//   2. 职业教育和成人教育层级不得被转换为K12数字；
//   3. 跨教育域严格匹配由grade_strict.go统一完成；
//   4. NormalizeGradeToSegment兼容返回中职和成人教育分组，
//      供AI助手手动候选相关性排序使用。

import (
	"strconv"
	"strings"
)

// NormalizeGradeToNumber 将K12中文年级名转换为数字格式。
//
// 职业教育和成人教育层级原样返回，禁止被K12中文数字规则误识别。
func NormalizeGradeToNumber(grade string) string {
	if strings.TrimSpace(grade) == "" {
		return grade
	}

	trimmed := strings.TrimSpace(grade)

	// 职业教育与成人教育层级不能进入K12数字提取链路。
	//
	// 例如“中职一年级”若继续向下执行，会被中文数字规则错误识别为
	// K12一年级。这里先原样返回，真正的规范化交给grade_strict.go。
	if isVocationalLearningLevelText(trimmed) ||
		isAdultLearningLevelText(trimmed) {
		return trimmed
	}

	// 步骤1：含横杠和数字，已是K12范围格式，直接返回。
	if result, ok := tryParseRangeFormat(trimmed); ok {
		return result
	}

	// 步骤2：提取阿拉伯数字。
	if result, ok := tryExtractDigits(trimmed); ok {
		return result
	}

	// 步骤3：小学段别名。
	if result, ok := tryMatchSegmentAlias(trimmed); ok {
		return result
	}

	// 步骤4：初高中别名。
	if result, ok := tryMatchSchoolAlias(trimmed); ok {
		return result
	}

	// 步骤5：中文数字。
	if result, ok := tryMatchChineseNumber(trimmed); ok {
		return result
	}

	// 步骤6：无法识别时保留原值。
	return grade
}

// tryParseRangeFormat 检查是否为含横杠的数字范围格式，例如3-6。
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

// tryExtractDigits 提取字符串中的阿拉伯数字。
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

// tryMatchSegmentAlias 匹配K12小学段别名。
func tryMatchSegmentAlias(grade string) (string, bool) {
	segmentMap := map[string]string{
		"小学低段": "1-2",
		"小学中段": "3-4",
		"小学高段": "5-6",
	}

	for segment, numberRange := range segmentMap {
		if strings.Contains(grade, segment) {
			return numberRange, true
		}
	}

	return "", false
}

// tryMatchSchoolAlias 匹配K12初高中别名。
func tryMatchSchoolAlias(grade string) (string, bool) {
	aliasMap := map[string]string{
		"初一": "7",
		"初二": "8",
		"初三": "9",
		"高一": "10",
		"高二": "11",
		"高三": "12",
	}

	for alias, number := range aliasMap {
		if strings.Contains(grade, alias) {
			return number, true
		}
	}

	return "", false
}

// tryMatchChineseNumber 匹配K12中文数字。
func tryMatchChineseNumber(grade string) (string, bool) {
	// 先匹配十一、十二，避免短字符串提前命中。
	longMap := map[string]string{
		"十一": "11",
		"十二": "12",
	}

	for chinese, number := range longMap {
		if strings.Contains(grade, chinese) {
			return number, true
		}
	}

	shortMap := map[string]string{
		"一": "1",
		"二": "2",
		"三": "3",
		"四": "4",
		"五": "5",
		"六": "6",
		"七": "7",
		"八": "8",
		"九": "9",
		"十": "10",
	}

	for chinese, number := range shortMap {
		if strings.Contains(grade, chinese) {
			return number, true
		}
	}

	return "", false
}

// 教学层级分组常量。
//
// K12继续使用小学、初中、高中；职业教育和成人教育使用独立分组，
// 防止职业教育的“职一”与K12“一年级”落入同一匹配桶。
const (
	SegmentPrimary    = "小学"
	SegmentJunior     = "初中"
	SegmentSenior     = "高中"
	SegmentVocational = "中职"
	SegmentAdult      = "成人教育"
	SegmentAll        = ""
)

// NormalizeGradeToSegment 将教学层级归一化为候选排序分组。
//
// K12示例：
//   七年级 → 初中
//   高三   → 高中
//
// 职业教育示例：
//   中职Ⅰ年级、职一、中职不限年级 → 中职
//
// 成人教育示例：
//   成人入门、成人高级、成人不限层级 → 成人教育
//
// 空值返回空字符串，表示不限层级。
func NormalizeGradeToSegment(grade string) string {
	trimmed := strings.TrimSpace(grade)
	if trimmed == "" {
		return SegmentAll
	}

	if isVocationalLearningLevelText(trimmed) {
		return SegmentVocational
	}

	if isAdultLearningLevelText(trimmed) {
		return SegmentAdult
	}

	// K12学段直通。
	if strings.Contains(trimmed, SegmentPrimary) {
		return SegmentPrimary
	}

	if strings.Contains(trimmed, SegmentJunior) {
		return SegmentJunior
	}

	if strings.Contains(trimmed, SegmentSenior) {
		return SegmentSenior
	}

	normalized := NormalizeGradeToNumber(trimmed)
	return rangeOrNumberToSegment(normalized)
}

// rangeOrNumberToSegment 将K12数字或范围格式归类到学段。
func rangeOrNumberToSegment(value string) string {
	if value == "" {
		return SegmentAll
	}

	if strings.Contains(value, "-") {
		parts := strings.SplitN(value, "-", 2)
		if len(parts) > 0 {
			value = parts[0]
		}
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return SegmentAll
	}

	switch {
	case number >= 1 && number <= 6:
		return SegmentPrimary
	case number >= 7 && number <= 9:
		return SegmentJunior
	case number >= 10 && number <= 12:
		return SegmentSenior
	default:
		return SegmentAll
	}
}

// isVocationalLearningLevelText 判断文本是否明显属于职业教育层级。
func isVocationalLearningLevelText(value string) bool {
	trimmed := strings.TrimSpace(value)

	return strings.Contains(trimmed, "中职") ||
		strings.HasPrefix(trimmed, "职一") ||
		strings.HasPrefix(trimmed, "职二") ||
		strings.HasPrefix(trimmed, "职三")
}

// isAdultLearningLevelText 判断文本是否明显属于成人教育层级。
func isAdultLearningLevelText(value string) bool {
	trimmed := strings.TrimSpace(value)

	if strings.Contains(trimmed, "成人") {
		return true
	}

	switch trimmed {
	case "入门", "进阶", "高级", "管理者", "不限层级":
		return true
	default:
		return false
	}
}
