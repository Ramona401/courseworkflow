package services

// courseware_ai_review_continuity.go
//
// 课件AI审核连续性账本合并与通用JSON辅助。
//
// 合并规则：
//   - 对象递归合并；
//   - 数组保留旧值并追加新值，按规范JSON去重；
//   - 空字符串、空数组、空对象或nil不能覆盖已有事实；
//   - 其它非空标量允许由新批次更新；
//   - 缺少version时自动补为1。

import (
	"encoding/json"
	"strings"
)

// mergeCWAIReviewContinuityLedger 字段级递归合并连续性账本。
func mergeCWAIReviewContinuityLedger(
	previous map[string]interface{},
	incoming map[string]interface{},
) map[string]interface{} {
	base := make(map[string]interface{})

	for key, value := range previous {
		base[key] =
			cloneCWAIReviewJSONValue(value)
	}

	for key, newValue := range incoming {
		oldValue, exists := base[key]
		if !exists {
			if !isCWAIReviewEmptyValue(newValue) {
				base[key] =
					cloneCWAIReviewJSONValue(
						newValue,
					)
			}
			continue
		}

		base[key] = mergeCWAIReviewJSONValue(
			oldValue,
			newValue,
		)
	}

	if _, exists := base["version"]; !exists {
		base["version"] = 1
	}

	return base
}

func mergeCWAIReviewJSONValue(
	oldValue interface{},
	newValue interface{},
) interface{} {
	if isCWAIReviewEmptyValue(newValue) {
		return cloneCWAIReviewJSONValue(
			oldValue,
		)
	}

	oldMap, oldMapOK :=
		oldValue.(map[string]interface{})
	newMap, newMapOK :=
		newValue.(map[string]interface{})

	if oldMapOK && newMapOK {
		return mergeCWAIReviewContinuityLedger(
			oldMap,
			newMap,
		)
	}

	oldArray, oldArrayOK :=
		oldValue.([]interface{})
	newArray, newArrayOK :=
		newValue.([]interface{})

	if oldArrayOK && newArrayOK {
		return mergeCWAIReviewJSONArray(
			oldArray,
			newArray,
		)
	}

	return cloneCWAIReviewJSONValue(
		newValue,
	)
}

func mergeCWAIReviewJSONArray(
	oldArray []interface{},
	newArray []interface{},
) []interface{} {
	result := make(
		[]interface{},
		0,
		len(oldArray)+len(newArray),
	)
	seen := make(map[string]bool)

	appendUnique := func(value interface{}) {
		encoded, err := json.Marshal(value)
		if err != nil {
			return
		}

		key := string(encoded)
		if seen[key] {
			return
		}

		seen[key] = true
		result = append(
			result,
			cloneCWAIReviewJSONValue(value),
		)
	}

	for _, value := range oldArray {
		appendUnique(value)
	}
	for _, value := range newArray {
		appendUnique(value)
	}

	return result
}

func cloneCWAIReviewJSONValue(
	value interface{},
) interface{} {
	encoded, err := json.Marshal(value)
	if err != nil {
		return value
	}

	var cloned interface{}
	if err := json.Unmarshal(
		encoded,
		&cloned,
	); err != nil {
		return value
	}

	return cloned
}

func isCWAIReviewEmptyValue(
	value interface{},
) bool {
	if value == nil {
		return true
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []interface{}:
		return len(typed) == 0
	case map[string]interface{}:
		return len(typed) == 0
	default:
		return false
	}
}

// normalizeCWAIReviewSeverity 规范化风险严重程度。
func normalizeCWAIReviewSeverity(
	raw string,
) string {
	switch strings.ToLower(
		strings.TrimSpace(raw),
	) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	case "info":
		return "info"
	default:
		return "medium"
	}
}

// cwAIReviewDecodeJSON 解析内部JSON快照，失败时返回指定默认值。
func cwAIReviewDecodeJSON(
	raw string,
	fallback interface{},
) interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}

	var value interface{}
	if err := json.Unmarshal(
		[]byte(raw),
		&value,
	); err != nil {
		return fallback
	}

	return value
}
