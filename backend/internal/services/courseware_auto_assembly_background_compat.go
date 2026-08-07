package services

// courseware_auto_assembly_background_compat.go — 自动装配模板背景兼容提取
//
// 旧提取器只识别 .cw-page.cover 和 .cw-page.inner。
// 实际模板还会使用 .tpl-xxx.cover、.tpl-xxx.inner，
// 或把背景写在根画布的内联style中。
//
// 本文件只提取含url()的根级background声明，
// 不处理普通卡片、轮播、视频帧和内容组件背景。

import (
	"regexp"
	"strings"
)

var (
	cwAutoAssemblyCompatStyleBlockRe =
		regexp.MustCompile(
			`(?is)<style\b[^>]*>(.*?)</style>`,
		)

	cwAutoAssemblyCompatRuleRe =
		regexp.MustCompile(
			`(?s)([^{}]+)\{([^{}]*)\}`,
		)

	cwAutoAssemblyCompatRootOpenRe =
		regexp.MustCompile(
			`(?is)<(?:div|section|main|article)\b[^>]*\bclass\s*=\s*(?:"[^"]*\bcw-page\b[^"]*"|'[^']*\bcw-page\b[^']*')[^>]*>`,
		)

	cwAutoAssemblyCompatStyleAttrRe =
		regexp.MustCompile(
			`(?is)\bstyle\s*=\s*(?:"([^"]*)"|'([^']*)')`,
		)
)

// extractAutoAssemblyTemplateBackgroundDecls 提取兼容模板背景声明。
func extractAutoAssemblyTemplateBackgroundDecls(
	samplePages []string,
	pageNumber int,
) string {
	if len(samplePages) == 0 {
		return ""
	}

	indexes :=
		cwAutoAssemblyBackgroundSampleIndexes(
			len(samplePages),
			pageNumber,
		)

	bestScore := -100000
	bestDeclarations := ""

	for priority, sampleIndex :=
		range indexes {
		if sampleIndex < 0 ||
			sampleIndex >=
				len(samplePages) {
			continue
		}

		sample :=
			strings.TrimSpace(
				samplePages[sampleIndex],
			)

		if sample == "" {
			continue
		}

		styleBlocks :=
			cwAutoAssemblyCompatStyleBlockRe.
				FindAllStringSubmatch(
					sample,
					-1,
				)

		for _, styleBlock :=
			range styleBlocks {
			if len(styleBlock) < 2 {
				continue
			}

			rules :=
				cwAutoAssemblyCompatRuleRe.
					FindAllStringSubmatch(
						styleBlock[1],
						-1,
					)

			for _, rule :=
				range rules {
				if len(rule) < 3 {
					continue
				}

				selector :=
					strings.ToLower(
						strings.TrimSpace(
							rule[1],
						),
					)

				if !cwAutoAssemblyRootBackgroundSelector(
					selector,
				) {
					continue
				}

				declarations :=
					cwExtractURLBackgroundDeclarations(
						rule[2],
					)

				if declarations == "" {
					continue
				}

				score :=
					cwAutoAssemblyBackgroundSelectorScore(
						selector,
						pageNumber,
					) +
						20 -
						priority

				if score > bestScore {
					bestScore =
						score

					bestDeclarations =
						declarations
				}
			}
		}

		rootDeclarations :=
			cwExtractRootInlineURLBackground(
				sample,
			)

		if rootDeclarations != "" {
			score :=
				80 +
					20 -
					priority

			if pageNumber == 1 &&
				sampleIndex == 0 {
				score += 40
			}

			if pageNumber > 1 &&
				sampleIndex != 0 {
				score += 40
			}

			if score > bestScore {
				bestScore =
					score

				bestDeclarations =
					rootDeclarations
			}
		}
	}

	return strings.TrimSpace(
		bestDeclarations,
	)
}

// cwAutoAssemblyBackgroundSampleIndexes 构造稳定样例检查顺序。
func cwAutoAssemblyBackgroundSampleIndexes(
	count int,
	pageNumber int,
) []int {
	result :=
		make(
			[]int,
			0,
			count,
		)

	used :=
		make(
			map[int]bool,
		)

	add :=
		func(index int) {
			if index < 0 ||
				index >= count ||
				used[index] {
				return
			}

			used[index] =
				true

			result =
				append(
					result,
					index,
				)
		}

	if pageNumber == 1 {
		add(0)
	} else {
		add(2)
		add(1)
		add(count - 1)
	}

	for index := 0;
		index < count;
		index++ {
		add(index)
	}

	return result
}

// cwAutoAssemblyRootBackgroundSelector 判断是否为根级模板选择器。
func cwAutoAssemblyRootBackgroundSelector(
	selector string,
) bool {
	if selector == "" {
		return false
	}

	return strings.Contains(
		selector,
		"cw-page",
	) ||
		strings.Contains(
			selector,
			"tpl-",
		) ||
		strings.Contains(
			selector,
			"cover",
		) ||
		strings.Contains(
			selector,
			"inner",
		)
}

// cwAutoAssemblyBackgroundSelectorScore 对背景选择器评分。
func cwAutoAssemblyBackgroundSelectorScore(
	selector string,
	pageNumber int,
) int {
	score := 0

	if strings.Contains(
		selector,
		"cw-page",
	) {
		score += 30
	}

	if strings.Contains(
		selector,
		"tpl-",
	) {
		score += 30
	}

	if pageNumber == 1 {
		if strings.Contains(
			selector,
			"cover",
		) {
			score += 120
		}

		if strings.Contains(
			selector,
			".p1",
		) {
			score += 40
		}

		if strings.Contains(
			selector,
			"inner",
		) {
			score -= 80
		}
	} else {
		if strings.Contains(
			selector,
			"inner",
		) ||
			strings.Contains(
				selector,
				"content",
			) {
			score += 120
		}

		if strings.Contains(
			selector,
			"cover",
		) {
			score -= 80
		}

		if strings.Contains(
			selector,
			".p1",
		) {
			score -= 40
		}
	}

	return score
}

// cwExtractRootInlineURLBackground 读取样例根画布内联背景。
func cwExtractRootInlineURLBackground(
	sample string,
) string {
	openTag :=
		cwAutoAssemblyCompatRootOpenRe.
			FindString(
				sample,
			)

	if openTag == "" {
		return ""
	}

	match :=
		cwAutoAssemblyCompatStyleAttrRe.
			FindStringSubmatch(
				openTag,
			)

	if len(match) < 3 {
		return ""
	}

	styleValue :=
		match[1]

	if styleValue == "" {
		styleValue =
			match[2]
	}

	return cwExtractURLBackgroundDeclarations(
		styleValue,
	)
}

// cwExtractURLBackgroundDeclarations 只保留含url()的background系列声明。
func cwExtractURLBackgroundDeclarations(
	declarations string,
) string {
	parts :=
		make(
			[]string,
			0,
		)

	hasURL := false

	for _, declaration :=
		range strings.Split(
			declarations,
			";",
		) {
		declaration =
			strings.TrimSpace(
				declaration,
			)

		if declaration == "" {
			continue
		}

		colon :=
			strings.Index(
				declaration,
				":",
			)

		if colon <= 0 {
			continue
		}

		property :=
			strings.ToLower(
				strings.TrimSpace(
					declaration[:colon],
				),
			)

		if !strings.HasPrefix(
			property,
			"background",
		) {
			continue
		}

		if strings.Contains(
			strings.ToLower(
				declaration,
			),
			"url(",
		) {
			hasURL =
				true
		}

		parts =
			append(
				parts,
				declaration,
			)
	}

	if !hasURL {
		return ""
	}

	return strings.Join(
		parts,
		";",
	)
}
