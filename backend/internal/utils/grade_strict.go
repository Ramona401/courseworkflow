package utils

// grade_strict.go — 助手与备课配方的具体年级严格匹配工具
//
// 背景：
// 历史上的 AI 助手匹配会把高一、高二、高三统一归一化为“高中”，
// 并允许空年级或学段级资源兜底。这会导致高三课程错误使用高一、
// 高二或仅标记为“高中”的助手和配方。
//
// 本文件提供 fail-closed 的具体年级规则：
//   - 请求年级和资源年级都必须能归一化为单一的 1—12 年级；
//   - 两者的具体数字必须完全一致；
//   - 学段、小学分段、跨年级范围、空值和无法识别值均不参与匹配。
//
// 示例：
//   高三、十二年级、12年级、12 → 都归一化为 12，可以互相匹配；
//   高中、高一、高二、10-12、空串 → 均不能匹配高三。

import (
	"strconv"
	"strings"
)

// NormalizeGradeToSpecific 将年级归一化为单一的 1—12 数字字符串。
//
// 返回值：
//
//	normalized：如“高三”返回“12”；
//	ok：只有输入明确表示一个具体年级时才为 true。
//
// 明确拒绝：
//   - 学段：小学、初中、高中；
//   - 范围：1-6、7-9、10-12、小学低段等；
//   - 空值及无法识别的文字。
func NormalizeGradeToSpecific(grade string) (normalized string, ok bool) {
	trimmed := strings.TrimSpace(grade)
	if trimmed == "" {
		return "", false
	}

	value := strings.TrimSpace(NormalizeGradeToNumber(trimmed))
	if value == "" || strings.Contains(value, "-") {
		return "", false
	}

	number, err := strconv.Atoi(value)
	if err != nil || number < 1 || number > 12 {
		return "", false
	}

	return strconv.Itoa(number), true
}

// NormalizeGradeToStandardLabel 将具体年级统一转换为平台标准名称。
//
// 合法示例：
//
//	1 / 1年级 / 一年级 → 一年级
//	初一 / 7 / 七年级   → 七年级
//	高三 / 12 / 十二年级 → 高三
//
// 学段、范围、空值和无法识别值返回("", false)。
func NormalizeGradeToStandardLabel(
	grade string,
) (string, bool) {
	normalized, ok := NormalizeGradeToSpecific(grade)
	if !ok {
		return "", false
	}

	labels := map[string]string{
		"1":  "一年级",
		"2":  "二年级",
		"3":  "三年级",
		"4":  "四年级",
		"5":  "五年级",
		"6":  "六年级",
		"7":  "七年级",
		"8":  "八年级",
		"9":  "九年级",
		"10": "高一",
		"11": "高二",
		"12": "高三",
	}

	label, exists := labels[normalized]
	return label, exists
}

// IsStrictGradeMatch 判断资源年级是否与当前课程的具体年级完全一致。
//
// 任一侧不是具体单一年级时都返回 false，禁止学段或空值兜底。
func IsStrictGradeMatch(resourceGrade string, requestedGrade string) bool {
	resource, resourceOK := NormalizeGradeToSpecific(resourceGrade)
	requested, requestedOK := NormalizeGradeToSpecific(requestedGrade)

	return resourceOK && requestedOK && resource == requested
}

// IsStrictSubjectGradeMatch 同时判断学科和具体年级。
//
// 学科必须为非空且去除首尾空格后完全一致；
// 年级必须通过 IsStrictGradeMatch。
func IsStrictSubjectGradeMatch(
	resourceSubject string,
	resourceGrade string,
	requestedSubject string,
	requestedGrade string,
) bool {
	resourceSubject = strings.TrimSpace(resourceSubject)
	requestedSubject = strings.TrimSpace(requestedSubject)

	if resourceSubject == "" ||
		requestedSubject == "" ||
		resourceSubject != requestedSubject {
		return false
	}

	return IsStrictGradeMatch(resourceGrade, requestedGrade)
}
