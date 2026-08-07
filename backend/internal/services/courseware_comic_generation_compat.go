package services

// courseware_comic_generation_compat.go — 漫画图片生产局部公共辅助
//
// 本文件提供图片生产模块需要的两个包级基础能力：
//   1. coursewareComicGenerationLog：统一的漫画图片生产日志对象；
//   2. coursewareComicPlanTruncateRunes：Unicode安全提示词截断。
//
// 这些能力独立于漫画规划文件中的私有函数名称，避免图片生产模块
// 因其它文件重命名或职责拆分而发生未定义符号编译错误。

import (
	"strings"

	"tedna/internal/logger"
)

// coursewareComicGenerationLog 是漫画图片生产统一日志对象。
var coursewareComicGenerationLog =
	logger.WithModule(
		"courseware_comic_generation",
	)

// coursewareComicPlanTruncateRunes 按Unicode字符安全截断漫画提示词片段。
func coursewareComicPlanTruncateRunes(
	value string,
	limit int,
) string {
	value = strings.TrimSpace(value)

	if limit <= 0 {
		return ""
	}

	runes := []rune(value)

	if len(runes) <= limit {
		return value
	}

	return string(
		runes[:limit],
	)
}
