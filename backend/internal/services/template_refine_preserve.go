package services

// template_refine_preserve.go — 模板微调结构保真与确定性样式应用工具。
//
// 核心原则：
//   1. AI不得重新输出或重写原始sample_pages。
//   2. AI只生成风格参数和纯CSS覆盖规则。
//   3. 后端对已存在的颜色、字体和阴影令牌做确定性替换。
//   4. 后端向每页插入带固定标记的CSS覆盖层。
//   5. 再次微调时替换旧覆盖层，不重复累加。
//   6. 本工具不解析或重建DOM，不删除HTML节点，不改ID，不改JavaScript逻辑。

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// templateStyleRefineResult 是AI模板微调的新输出协议。
//
// SamplePages不再属于AI输出协议，原始页面始终由后端持有并确定性修改。
type templateStyleRefineResult struct {
	ColorScheme       map[string]string `json:"color_scheme"`
	CSSVariables      map[string]string `json:"css_variables"`
	CSSOverrides      string            `json:"css_overrides"`
	ChangeSummary     string            `json:"change_summary"`
	SuggestedCategory string            `json:"suggested_category"`
	Error             string            `json:"error,omitempty"`
}

// 单次AI微调允许返回的CSS覆盖规则最大字符数。
const templateRefineMaxCSSOverrideRunes = 24000

// 模板微调CSS覆盖层的固定标记。
// 再次微调时按此标记整块替换，避免历史样式无限叠加。
const (
	templateRefineStyleStartMarker = "/* TEDNA-TPL-REFINE-START */"
	templateRefineStyleEndMarker   = "/* TEDNA-TPL-REFINE-END */"
)

var templateRefineStyleBlockRe = regexp.MustCompile(
	`(?is)<style\b[^>]*>\s*/\*\s*TEDNA-TPL-REFINE-START\s*\*/.*?/\*\s*TEDNA-TPL-REFINE-END\s*\*/\s*</style\s*>`,
)

// templateStyleTokenReplacement 描述一项确定性样式令牌替换。
type templateStyleTokenReplacement struct {
	Old             string
	New             string
	CaseInsensitive bool
}

// decodeTemplateSamplePages 解析数据库中的sample_pages完整页面数组。
func decodeTemplateSamplePages(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "[]" {
		return nil, fmt.Errorf("模板没有可用的样例页面")
	}

	var pages []string
	if err := json.Unmarshal([]byte(raw), &pages); err != nil {
		return nil, fmt.Errorf("解析模板样例页面失败: %w", err)
	}

	validPages := make([]string, 0, len(pages))
	for _, page := range pages {
		if strings.TrimSpace(page) != "" {
			validPages = append(validPages, page)
		}
	}
	if len(validPages) == 0 {
		return nil, fmt.Errorf("模板样例页面全部为空")
	}

	return validPages, nil
}

// parseTemplateStringMap 解析配色或CSS变量JSON对象。
func parseTemplateStringMap(raw string) (map[string]string, error) {
	result := map[string]string{}

	if strings.TrimSpace(raw) == "" {
		return result, nil
	}

	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// decodeExtractedJSONString 解码字段级宽容解析得到的JSON转义字符串。
func decodeExtractedJSONString(raw string) string {
	if raw == "" {
		return ""
	}

	decoded, err := strconv.Unquote(`"` + raw + `"`)
	if err == nil {
		return decoded
	}

	return raw
}

// validateTemplateStyleRefineResult 校验AI返回的风格参数。
func validateTemplateStyleRefineResult(result *templateStyleRefineResult) error {
	if result == nil {
		return fmt.Errorf("AI未返回模板微调结果")
	}

	requiredColorKeys := []string{
		"primary",
		"secondary",
		"background",
		"accent",
		"text",
	}
	for _, key := range requiredColorKeys {
		if strings.TrimSpace(result.ColorScheme[key]) == "" {
			return fmt.Errorf("color_scheme.%s缺失", key)
		}
	}

	requiredCSSKeys := []string{
		"--cw-primary",
		"--cw-secondary",
		"--cw-bg",
		"--cw-accent",
		"--cw-text",
		"--cw-font-heading",
		"--cw-font-body",
		"--cw-radius",
		"--cw-shadow",
	}
	for _, key := range requiredCSSKeys {
		value := strings.TrimSpace(result.CSSVariables[key])
		if value == "" {
			return fmt.Errorf("css_variables.%s缺失", key)
		}
		if err := validateTemplateCSSVariableValue(key, value); err != nil {
			return err
		}
	}

	for key, value := range result.CSSVariables {
		if !strings.HasPrefix(key, "--cw-") {
			continue
		}
		if err := validateTemplateCSSVariableValue(key, value); err != nil {
			return err
		}
	}

	return nil
}

// validateTemplateCSSVariableValue 防止CSS变量值逃逸出声明块。
func validateTemplateCSSVariableValue(key string, value string) error {
	lower := strings.ToLower(value)

	forbidden := []string{
		";",
		"{",
		"}",
		"</style",
		"<script",
		"javascript:",
		"expression(",
		"@import",
	}
	for _, token := range forbidden {
		if strings.Contains(lower, token) {
			return fmt.Errorf("css_variables.%s包含不安全内容", key)
		}
	}

	return nil
}

// sanitizeTemplateCSSOverrides 清理并校验AI返回的纯CSS覆盖规则。
func sanitizeTemplateCSSOverrides(raw string) (string, error) {
	css := strings.TrimSpace(cwStripCodeFences(raw))

	// 兼容AI误包裹<style>标签的情况，只剥最外层包装。
	lower := strings.ToLower(css)
	if strings.HasPrefix(lower, "<style") {
		openEnd := strings.Index(css, ">")
		closeStart := strings.LastIndex(strings.ToLower(css), "</style")
		if openEnd >= 0 && closeStart > openEnd {
			css = strings.TrimSpace(css[openEnd+1 : closeStart])
		}
	}

	if css == "" {
		return "", nil
	}

	if len([]rune(css)) > templateRefineMaxCSSOverrideRunes {
		return "", fmt.Errorf(
			"AI返回的CSS覆盖规则超过%d字符上限",
			templateRefineMaxCSSOverrideRunes,
		)
	}

	lower = strings.ToLower(css)
	forbidden := []string{
		"</style",
		"<script",
		"</script",
		"@import",
		"url(",
		"expression(",
		"javascript:",
		"behavior:",
		"-moz-binding",
	}
	for _, token := range forbidden {
		if strings.Contains(lower, token) {
			return "", fmt.Errorf("AI返回的CSS覆盖规则包含禁止内容：%s", token)
		}
	}

	if !templateCSSBracesBalanced(css) {
		return "", fmt.Errorf("AI返回的CSS覆盖规则花括号不完整")
	}

	return css, nil
}

// templateCSSBracesBalanced 检查CSS花括号是否闭合，忽略字符串内部的括号。
func templateCSSBracesBalanced(css string) bool {
	depth := 0
	var quote byte
	escaped := false

	for i := 0; i < len(css); i++ {
		char := css[i]

		if escaped {
			escaped = false
			continue
		}
		if char == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}

		switch char {
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}

	return depth == 0 && quote == 0
}

// buildTemplateRefineStyleBlock 构造写入每个原始页面的受控CSS覆盖层。
func buildTemplateRefineStyleBlock(
	cssVariables map[string]string,
	cssOverrides string,
) string {
	keys := make([]string, 0, len(cssVariables))
	for key := range cssVariables {
		if strings.HasPrefix(key, "--cw-") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(`<style data-tedna-template-refine="1">`)
	sb.WriteString("\n")
	sb.WriteString(templateRefineStyleStartMarker)
	sb.WriteString("\n")

	if len(keys) > 0 {
		sb.WriteString(":root,.cw-page{\n")
		for _, key := range keys {
			value := strings.TrimSpace(cssVariables[key])
			if value == "" {
				continue
			}
			sb.WriteString("  ")
			sb.WriteString(key)
			sb.WriteString(":")
			sb.WriteString(value)
			sb.WriteString(";\n")
		}
		sb.WriteString("}\n")

		// 基础字体和正文颜色统一从变量层落实。
		sb.WriteString(
			".cw-page{color:var(--cw-text)!important;" +
				"font-family:var(--cw-font-body)!important;}\n",
		)
		sb.WriteString(
			".cw-page h1,.cw-page h2,.cw-page h3,.cw-page h4," +
				".cw-page h5,.cw-page h6{" +
				"font-family:var(--cw-font-heading)!important;}\n",
		)
	}

	if strings.TrimSpace(cssOverrides) != "" {
		sb.WriteString("\n")
		sb.WriteString(strings.TrimSpace(cssOverrides))
		sb.WriteString("\n")
	}

	sb.WriteString(templateRefineStyleEndMarker)
	sb.WriteString("\n</style>")

	return sb.String()
}

// insertOrReplaceTemplateRefineStyleBlock 插入或替换模板微调CSS块。
func insertOrReplaceTemplateRefineStyleBlock(
	source string,
	styleBlock string,
) string {
	if templateRefineStyleBlockRe.MatchString(source) {
		return templateRefineStyleBlockRe.ReplaceAllString(
			source,
			styleBlock,
		)
	}

	lower := strings.ToLower(source)

	// 完整HTML文档优先写入head末尾。
	if idx := strings.Index(lower, "</head>"); idx >= 0 {
		return source[:idx] + styleBlock + "\n" + source[idx:]
	}

	// 无head但有body时，写入body之前。
	if idx := strings.Index(lower, "<body"); idx >= 0 {
		return source[:idx] + styleBlock + "\n" + source[idx:]
	}

	// HTML片段允许style作为首个节点。
	return styleBlock + "\n" + source
}

// buildTemplateStyleTokenReplacements 构造旧风格令牌到新令牌的映射。
//
// 只替换明确的颜色、字体和完整阴影字符串，不直接全局替换12px等通用尺寸，
// 避免把圆角修改误作用到间距、字号或定位值。
func buildTemplateStyleTokenReplacements(
	oldColors map[string]string,
	newColors map[string]string,
	oldVariables map[string]string,
	newVariables map[string]string,
) []templateStyleTokenReplacement {
	result := make([]templateStyleTokenReplacement, 0, 12)
	seen := map[string]struct{}{}

	add := func(oldValue string, newValue string, caseInsensitive bool) {
		oldValue = strings.TrimSpace(oldValue)
		newValue = strings.TrimSpace(newValue)

		if oldValue == "" || newValue == "" || oldValue == newValue {
			return
		}
		if len([]rune(oldValue)) < 4 {
			return
		}

		seenKey := fmt.Sprintf("%t:%s", caseInsensitive, strings.ToLower(oldValue))
		if _, exists := seen[seenKey]; exists {
			return
		}
		seen[seenKey] = struct{}{}

		result = append(result, templateStyleTokenReplacement{
			Old:             oldValue,
			New:             newValue,
			CaseInsensitive: caseInsensitive,
		})
	}

	colorKeys := []string{
		"primary",
		"secondary",
		"background",
		"accent",
		"text",
	}
	for _, key := range colorKeys {
		add(oldColors[key], newColors[key], true)
	}

	colorVariableKeys := []string{
		"--cw-primary",
		"--cw-secondary",
		"--cw-bg",
		"--cw-accent",
		"--cw-text",
	}
	for _, key := range colorVariableKeys {
		oldValue := oldVariables[key]
		if templateLooksLikeColorToken(oldValue) {
			add(oldValue, newVariables[key], true)
		}
	}

	// 字体和阴影通常是完整长字符串，可安全做精确替换。
	exactVariableKeys := []string{
		"--cw-font-heading",
		"--cw-font-body",
		"--cw-shadow",
	}
	for _, key := range exactVariableKeys {
		add(oldVariables[key], newVariables[key], false)
	}

	// 优先替换更长的令牌，防止短令牌先吃掉长令牌的一部分。
	sort.SliceStable(result, func(i int, j int) bool {
		return len(result[i].Old) > len(result[j].Old)
	})

	return result
}

// templateLooksLikeColorToken 判断值是否为足够明确的颜色表达。
func templateLooksLikeColorToken(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "#") ||
		strings.HasPrefix(lower, "rgb(") ||
		strings.HasPrefix(lower, "rgba(") ||
		strings.HasPrefix(lower, "hsl(") ||
		strings.HasPrefix(lower, "hsla(") ||
		strings.HasPrefix(lower, "oklch(")
}

// applyTemplateStyleTokenReplacements 使用占位符实现近似“同时替换”，避免映射串联污染。
func applyTemplateStyleTokenReplacements(
	source string,
	replacements []templateStyleTokenReplacement,
) string {
	if len(replacements) == 0 {
		return source
	}

	result := source
	placeholders := make([]string, len(replacements))

	for idx, replacement := range replacements {
		placeholder := fmt.Sprintf(
			"__TEDNA_TEMPLATE_STYLE_TOKEN_%03d__",
			idx,
		)
		placeholders[idx] = placeholder

		if replacement.CaseInsensitive {
			pattern, err := regexp.Compile(
				"(?i)" + regexp.QuoteMeta(replacement.Old),
			)
			if err != nil {
				continue
			}
			result = pattern.ReplaceAllString(result, placeholder)
		} else {
			result = strings.ReplaceAll(
				result,
				replacement.Old,
				placeholder,
			)
		}
	}

	for idx, replacement := range replacements {
		result = strings.ReplaceAll(
			result,
			placeholders[idx],
			replacement.New,
		)
	}

	return result
}

// applyTemplateStyleRefinement 把风格微调结果确定性应用到所有原始母版页面。
func applyTemplateStyleRefinement(
	pages []string,
	oldColors map[string]string,
	newColors map[string]string,
	oldVariables map[string]string,
	newVariables map[string]string,
	cssOverrides string,
) ([]string, error) {
	if len(pages) == 0 {
		return nil, fmt.Errorf("模板没有可供微调的原始页面")
	}

	safeOverrides, err := sanitizeTemplateCSSOverrides(cssOverrides)
	if err != nil {
		return nil, err
	}

	styleBlock := buildTemplateRefineStyleBlock(
		newVariables,
		safeOverrides,
	)
	tokenReplacements := buildTemplateStyleTokenReplacements(
		oldColors,
		newColors,
		oldVariables,
		newVariables,
	)

	result := make([]string, 0, len(pages))
	for idx, page := range pages {
		if strings.TrimSpace(page) == "" {
			return nil, fmt.Errorf("模板第%d页为空，无法执行微调", idx+1)
		}

		updated := applyTemplateStyleTokenReplacements(
			page,
			tokenReplacements,
		)
		updated = insertOrReplaceTemplateRefineStyleBlock(
			updated,
			styleBlock,
		)

		result = append(result, updated)
	}

	return result, nil
}
