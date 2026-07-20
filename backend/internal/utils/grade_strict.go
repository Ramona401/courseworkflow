package utils

// grade_strict.go — 跨教育域具体教学层级严格匹配工具
//
// 自动匹配必须遵循以下规则：
//   - K12：一年级至高三中的单一具体年级；
//   - 职业教育：中职Ⅰ、Ⅱ、Ⅲ年级；
//   - 成人教育：成人入门、进阶、高级、管理者；
//   - 学段、范围、不限值和空值均不参与自动匹配；
//   - 不同教育域绝不能因为序号相同而互相匹配。
//
// 例如：
//   - K12一年级的内部标识是“1”；
//   - 中职Ⅰ年级的内部标识是“vocational:1”；
//   - 两者不会误判为同一层级。

import (
	"strconv"
	"strings"
)

// 职业教育具体年级的规范值。
const (
	VocationalGradeOne   = "中职Ⅰ年级"
	VocationalGradeTwo   = "中职Ⅱ年级"
	VocationalGradeThree = "中职Ⅲ年级"
	VocationalGradeAll   = "中职不限年级"
)

// 成人教育具体层级的规范值。
const (
	AdultLevelEntry    = "成人入门"
	AdultLevelAdvanced = "成人进阶"
	AdultLevelSenior   = "成人高级"
	AdultLevelManager  = "成人管理者"
	AdultLevelAll      = "成人不限层级"
)

// vocationalSpecificAliases 接受常见职业教育输入别名，统一落为规范值。
var vocationalSpecificAliases = map[string]string{
	"中职Ⅰ年级": VocationalGradeOne,
	"中职I年级": VocationalGradeOne,
	"中职1年级": VocationalGradeOne,
	"中职一年级": VocationalGradeOne,
	"中职Ⅰ":   VocationalGradeOne,
	"中职I":   VocationalGradeOne,
	"中职1":   VocationalGradeOne,
	"中职一":   VocationalGradeOne,
	"职一":    VocationalGradeOne,

	"中职Ⅱ年级":  VocationalGradeTwo,
	"中职II年级": VocationalGradeTwo,
	"中职2年级":  VocationalGradeTwo,
	"中职二年级":  VocationalGradeTwo,
	"中职Ⅱ":    VocationalGradeTwo,
	"中职II":   VocationalGradeTwo,
	"中职2":    VocationalGradeTwo,
	"中职二":    VocationalGradeTwo,
	"职二":     VocationalGradeTwo,

	"中职Ⅲ年级":   VocationalGradeThree,
	"中职III年级": VocationalGradeThree,
	"中职3年级":   VocationalGradeThree,
	"中职三年级":   VocationalGradeThree,
	"中职Ⅲ":     VocationalGradeThree,
	"中职III":   VocationalGradeThree,
	"中职3":     VocationalGradeThree,
	"中职三":     VocationalGradeThree,
	"职三":      VocationalGradeThree,
}

// adultSpecificAliases 接受成人教育常见简称，统一落为规范值。
var adultSpecificAliases = map[string]string{
	"成人入门": AdultLevelEntry,
	"入门":   AdultLevelEntry,

	"成人进阶": AdultLevelAdvanced,
	"进阶":   AdultLevelAdvanced,

	"成人高级": AdultLevelSenior,
	"高级":   AdultLevelSenior,

	"成人管理者": AdultLevelManager,
	"管理者":   AdultLevelManager,
}

// NormalizeGradeToSpecific 将任意具体教学层级归一化为严格匹配标识。
//
// K12继续返回历史数字格式，例如高三返回12，保持存量调用兼容。
// 职业教育和成人教育使用带教育域前缀的标识，防止跨域串用。
//
// 明确拒绝：
//   - K12学段与范围；
//   - 中职不限年级；
//   - 成人不限层级；
//   - 空值和无法识别值。
func NormalizeGradeToSpecific(grade string) (normalized string, ok bool) {
	trimmed := strings.TrimSpace(grade)
	if trimmed == "" {
		return "", false
	}

	if standard, found := normalizeVocationalSpecificLevel(trimmed); found {
		switch standard {
		case VocationalGradeOne:
			return "vocational:1", true
		case VocationalGradeTwo:
			return "vocational:2", true
		case VocationalGradeThree:
			return "vocational:3", true
		}
	}

	if standard, found := normalizeAdultSpecificLevel(trimmed); found {
		switch standard {
		case AdultLevelEntry:
			return "adult:entry", true
		case AdultLevelAdvanced:
			return "adult:advanced", true
		case AdultLevelSenior:
			return "adult:senior", true
		case AdultLevelManager:
			return "adult:manager", true
		}
	}

	if _, broadOK := NormalizeBroadLearningLevel(trimmed); broadOK {
		return "", false
	}

	// 任何明显属于非K12的文本都不能继续进入K12数字解析。
	if isVocationalLearningLevelText(trimmed) ||
		isAdultLearningLevelText(trimmed) {
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

// NormalizeGradeToStandardLabel 将具体教学层级转换为平台规范显示值。
//
// 具体层级示例：
//   - 十二年级 → 高三
//   - 职一 → 中职Ⅰ年级
//   - 进阶 → 成人进阶
//
// 不限层级与学段不会通过本函数，确保配方只能绑定具体层级。
// AI助手的手动通用层级由NormalizeBroadLearningLevel单独处理。
func NormalizeGradeToStandardLabel(grade string) (string, bool) {
	normalized, ok := NormalizeGradeToSpecific(grade)
	if !ok {
		return "", false
	}

	switch normalized {
	case "vocational:1":
		return VocationalGradeOne, true
	case "vocational:2":
		return VocationalGradeTwo, true
	case "vocational:3":
		return VocationalGradeThree, true
	case "adult:entry":
		return AdultLevelEntry, true
	case "adult:advanced":
		return AdultLevelAdvanced, true
	case "adult:senior":
		return AdultLevelSenior, true
	case "adult:manager":
		return AdultLevelManager, true
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

// NormalizeBroadLearningLevel 规范化只供手动选择的通用层级。
//
// 返回值仅包括：
//   - K12小学、初中、高中；
//   - 中职不限年级；
//   - 成人不限层级。
//
// 空字符串由上层助手逻辑作为历史“不限年级”值单独处理。
func NormalizeBroadLearningLevel(level string) (string, bool) {
	trimmed := strings.TrimSpace(level)

	switch trimmed {
	case "小学", "初中", "高中":
		return trimmed, true
	case "中职不限年级", "中职不限", "不限中职年级":
		return VocationalGradeAll, true
	case "成人不限层级", "成人不限", "不限成人层级":
		return AdultLevelAll, true
	default:
		return "", false
	}
}

// IsStrictGradeMatch 判断资源层级是否与当前课程具体层级完全一致。
//
// 内部严格标识包含教育域前缀，因此：
//   - 中职Ⅰ年级不会匹配K12一年级；
//   - 成人入门不会匹配任何K12或中职层级。
func IsStrictGradeMatch(resourceGrade string, requestedGrade string) bool {
	resource, resourceOK := NormalizeGradeToSpecific(resourceGrade)
	requested, requestedOK := NormalizeGradeToSpecific(requestedGrade)

	return resourceOK && requestedOK && resource == requested
}

// IsStrictSubjectGradeMatch 同时判断课程与具体教学层级。
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

// normalizeVocationalSpecificLevel 返回职业教育具体年级规范值。
func normalizeVocationalSpecificLevel(value string) (string, bool) {
	standard, ok := vocationalSpecificAliases[strings.TrimSpace(value)]
	return standard, ok
}

// normalizeAdultSpecificLevel 返回成人教育具体层级规范值。
func normalizeAdultSpecificLevel(value string) (string, bool) {
	standard, ok := adultSpecificAliases[strings.TrimSpace(value)]
	return standard, ok
}
