package services

// lesson_plan_publish_intent.go — 对话式备课明确发布意图守卫
//
// 本文件只负责识别明确、短促、可以直接执行的定稿或发布确认语。
// 普通讨论中即使出现“发布”“定稿”等词，也不能升级为写操作。
//
// Chat入口在登记后台任务和写入用户消息前调用本守卫，
// 使旧缓存页面、旧客户端和手工API调用都无法误启动正式产物Harness。

import (
	"errors"
	"strings"
)

var ErrLPGenPublishIntent = errors.New(
	"检测到明确的定稿或发布意图，请使用“发布教案”操作确认当前正式版本",
)

var lessonPlanPublishIntentTexts = map[string]struct{}{
	"教案我满意了就这样定稿": {},
	"我满意了就这样定稿":   {},
	"就这样定稿":       {},
	"完成并发布":       {},
	"不用改了发布":      {},
	"确认发布":        {},
	"确认发布教案":      {},
	"发布教案":        {},
	"就按这个版本发布":    {},
	"就按这个版本定稿":    {},
	"确定这个版本":      {},
	"确认这个版本":      {},
	"就这个版本":       {},
}

// isLessonPlanPublishIntent 只识别明确终态确认语。
func isLessonPlanPublishIntent(
	value string,
) bool {
	normalized :=
		normalizeLessonPlanPublishIntent(
			value,
		)
	if normalized == "" {
		return false
	}

	_, exists :=
		lessonPlanPublishIntentTexts[normalized]

	return exists
}

func normalizeLessonPlanPublishIntent(
	value string,
) string {
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

	return strings.NewReplacer(
		" ", "",
		"\n", "",
		"\r", "",
		"\t", "",
		"，", "",
		",", "",
		"。", "",
		".", "",
		"！", "",
		"!", "",
		"？", "",
		"?", "",
		"；", "",
		";", "",
		"：", "",
		":", "",
	).Replace(value)
}
