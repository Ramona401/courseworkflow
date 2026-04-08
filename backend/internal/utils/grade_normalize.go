package utils

// grade_normalize.go — 中文年级名统一转换工具
//
// v82新增：从 workshop_stage_prompts.go(normalizeGradeToNumber) 和
// component_service.go(normalizeGradeForMatch) 两处重复实现中抽取合并。
//
// 转换规则：
//   - 已含阿拉伯数字 → 直接提取数字部分（如"7年级"→"7"、"3-6"→"36"不变因为含数字直接返回）
//   - 小学段别名 → 返回范围格式（如"小学低段"→"1-2"、"小学中段"→"3-4"、"小学高段"→"5-6"）
//   - 初高中别名 → 转换（如"初一"→"7"、"高一"→"10"）
//   - 中文数字 → 转换（如"三年级"→"3"、"十二年级"→"12"）
//   - 无法识别 → 返回原值

import "strings"

// NormalizeGradeToNumber 将中文年级名转换为数字格式
// 用于组件匹配SQL的年级范围比较
//
// 示例：
//   "三年级" → "3"
//   "七年级" → "7"
//   "初一"   → "7"
//   "高二"   → "11"
//   "十二年级" → "12"
//   "小学低段" → "1-2"
//   "7"      → "7"
//   "3-6"    → "3-6"（已含数字+横杠，保留原值更合理——见特殊处理）
func NormalizeGradeToNumber(grade string) string {
	if strings.TrimSpace(grade) == "" {
		return grade
	}

	// 1. 先检查是否已包含阿拉伯数字
	// 特殊情况：如果包含横杠（如"3-6"），说明已经是范围格式，直接返回原值
	if strings.Contains(grade, "-") {
		hasDigit := false
		for _, b := range []byte(grade) {
			if b >= '0' && b <= '9' {
				hasDigit = true
				break
			}
		}
		if hasDigit {
			return grade
		}
	}

	// 提取纯数字部分
	var digits []byte
	for _, b := range []byte(grade) {
		if b >= '0' && b <= '9' {
			digits = append(digits, b)
		}
	}
	if len(digits) > 0 {
		return string(digits)
	}

	// 2. 小学段别名（特殊处理，返回范围格式）
	segmentMap := map[string]string{
		"小学低段": "1-2",
		"小学中段": "3-4",
		"小学高段": "5-6",
	}
	for seg, num := range segmentMap {
		if strings.Contains(grade, seg) {
			return num
		}
	}

	// 3. 初高中别名
	aliasMap := map[string]string{
		"初一": "7", "初二": "8", "初三": "9",
		"高一": "10", "高二": "11", "高三": "12",
	}
	for alias, num := range aliasMap {
		if strings.Contains(grade, alias) {
			return num
		}
	}

	// 4. 中文数字转换（先匹配长的"十一""十二"，再匹配短的，防止"十"提前命中）
	cnMapLong := map[string]string{
		"十一": "11", "十二": "12",
	}
	for cn, num := range cnMapLong {
		if strings.Contains(grade, cn) {
			return num
		}
	}
	cnMap := map[string]string{
		"一": "1", "二": "2", "三": "3", "四": "4", "五": "5",
		"六": "6", "七": "7", "八": "8", "九": "9", "十": "10",
	}
	for cn, num := range cnMap {
		if strings.Contains(grade, cn) {
			return num
		}
	}

	// 5. 无法识别，返回原值
	return grade
}
