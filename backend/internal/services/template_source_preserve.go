package services

// template_source_preserve.go — 自建模板源页面保真与多页匹配公共工具。
//
// 核心职责：
//   1. 校验并完整保留老师提交的原始HTML母版页。
//   2. 页面较多时确定性抽取少量代表页供AI分析，避免把全部长HTML塞入模型。
//   3. 分析副本剥除base64图片和脚本正文，但数据库中的原始HTML不做任何改写。
//   4. 按页面代码与语义推断封面、目标、内容、互动、总结、作业等页型。
//   5. 为任意页数模板选择更贴近当前课件页面的参考样例，消除仅支持五页模板的假设。
//
// 本文件只做确定性字符串处理，不调用AI、不访问数据库、不修改原始页面。

import (
	"fmt"
	"regexp"
	"strings"

	"tedna/internal/models"
)

const (
	// 自建模板允许保留的最大母版页数量。
	templateSourceMaxPages = 20

	// 单页HTML最大字符数。防止单个异常页面或内嵌超大资源占满请求和数据库。
	templateSourceMaxSingleHTMLLen = 120000

	// 全部母版页HTML总字符数上限。
	templateSourceMaxTotalHTMLLen = 600000

	// 模板提取时送给AI分析的最大代表页数量。
	templateAnalysisMaxPages = 8

	// 每个代表页送给AI的最大rune数量，按Unicode字符截断避免截断中文。
	templateAnalysisMaxRunesPerPage = 18000
)

// templateAnalysisPage 是送给AI的代表页副本。
// HTML仅用于AI分析，原始页面仍由调用方完整保存。
type templateAnalysisPage struct {
	SourcePageNumber int
	Role             string
	RoleLabel        string
	HTML             string
}

// templatePageRoleMeta 写入extract_source_meta，供后续页面选择与问题排查使用。
type templatePageRoleMeta struct {
	PageNumber int    `json:"page_number"`
	Role       string `json:"role"`
	Label      string `json:"label"`
}

// 分析副本中的大体积资源清理规则。
// 这些规则绝不作用于最终入库的原始母版，只作用于送给AI的临时字符串。
var (
	templateAnalysisDataURIRe = regexp.MustCompile(
		`(?is)data:image/[^;,"')\s]+;base64,[a-z0-9+/=\r\n]+`,
	)
	templateAnalysisScriptRe = regexp.MustCompile(
		`(?is)<script\b[^>]*>.*?</script\s*>`,
	)
)

// prepareTemplateSourcePages 清理空页面并执行数量和体量校验。
// 返回值pages中的每一项仍是老师提交的完整原始HTML，仅去除首尾空白。
func prepareTemplateSourcePages(pages []string) ([]string, int, error) {
	if len(pages) == 0 {
		return nil, 0, fmt.Errorf("请至少提供一页 HTML 代码")
	}

	cleaned := make([]string, 0, len(pages))
	totalLen := 0

	for sourceIndex, page := range pages {
		trimmed := strings.TrimSpace(page)
		if trimmed == "" {
			continue
		}

		if len(trimmed) > templateSourceMaxSingleHTMLLen {
			return nil, 0, fmt.Errorf(
				"第%d页HTML长度%d字符，超过单页上限%d字符",
				sourceIndex+1,
				len(trimmed),
				templateSourceMaxSingleHTMLLen,
			)
		}

		cleaned = append(cleaned, trimmed)
		totalLen += len(trimmed)
	}

	if len(cleaned) == 0 {
		return nil, 0, fmt.Errorf("提供的 HTML 内容为空，请粘贴有效的 HTML 代码")
	}

	if len(cleaned) > templateSourceMaxPages {
		return nil, 0, fmt.Errorf(
			"有效页面共%d页，超过模板上限%d页",
			len(cleaned),
			templateSourceMaxPages,
		)
	}

	if totalLen > templateSourceMaxTotalHTMLLen {
		return nil, 0, fmt.Errorf(
			"HTML总长度%d字符，超过上限%d字符",
			totalLen,
			templateSourceMaxTotalHTMLLen,
		)
	}

	return cleaned, totalLen, nil
}

// selectTemplateAnalysisPages 从全部原始母版中确定性抽取代表页。
// 页面数不超过limit时全部分析；超过时按首页到末页均匀取样，确保首尾页必定入选。
func selectTemplateAnalysisPages(
	pages []string,
	limit int,
	maxRunesPerPage int,
) []templateAnalysisPage {
	if len(pages) == 0 || limit <= 0 {
		return []templateAnalysisPage{}
	}

	if limit > len(pages) {
		limit = len(pages)
	}

	indices := make([]int, 0, limit)

	if limit == 1 {
		indices = append(indices, 0)
	} else {
		// 用整数四舍五入在[0,n-1]上均匀取点。
		// 例如20页取8页时，首页、中间页和末页都会被覆盖。
		denominator := limit - 1
		lastIndex := len(pages) - 1

		for i := 0; i < limit; i++ {
			idx := (i*lastIndex + denominator/2) / denominator
			if len(indices) == 0 || indices[len(indices)-1] != idx {
				indices = append(indices, idx)
			}
		}
	}

	result := make([]templateAnalysisPage, 0, len(indices))
	for _, idx := range indices {
		role := inferTemplatePageRole(pages[idx], idx+1, len(pages))
		result = append(result, templateAnalysisPage{
			SourcePageNumber: idx + 1,
			Role:             role,
			RoleLabel:        templatePageRoleLabel(role),
			HTML: truncateTemplateAnalysisHTML(
				sanitizeTemplateHTMLForAnalysis(pages[idx]),
				maxRunesPerPage,
			),
		})
	}

	return result
}

// sanitizeTemplateHTMLForAnalysis 构造AI分析副本。
// base64图片和脚本正文体积大且不影响风格识别，替换为明确占位；原始HTML不受影响。
func sanitizeTemplateHTMLForAnalysis(source string) string {
	result := templateAnalysisDataURIRe.ReplaceAllString(
		source,
		"data:image/omitted;base64,TEDNA_TEMPLATE_IMAGE_OMITTED",
	)
	result = templateAnalysisScriptRe.ReplaceAllString(
		result,
		"<script>/* TEDNA: 脚本正文已从风格分析副本中省略，原始母版仍完整保存 */</script>",
	)
	return result
}

// truncateTemplateAnalysisHTML 按rune安全截断分析副本。
func truncateTemplateAnalysisHTML(source string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}

	runes := []rune(source)
	if len(runes) <= maxRunes {
		return source
	}

	return string(runes[:maxRunes]) +
		"\n<!-- TEDNA：本页分析副本过长，后续代码已截断；原始母版页仍完整保存 -->"
}

// buildTemplatePageRoleMeta 为全部原始母版生成轻量页型元信息。
func buildTemplatePageRoleMeta(pages []string) []templatePageRoleMeta {
	result := make([]templatePageRoleMeta, 0, len(pages))

	for idx, page := range pages {
		role := inferTemplatePageRole(page, idx+1, len(pages))
		result = append(result, templatePageRoleMeta{
			PageNumber: idx + 1,
			Role:       role,
			Label:      templatePageRoleLabel(role),
		})
	}

	return result
}

// inferTemplatePageRole 根据页面位置、可见中文关键词和交互代码特征推断页型。
// 该判断只用于推荐与展示标签，不参与权限和数据写入决策。
func inferTemplatePageRole(source string, pageNumber int, totalPages int) string {
	lower := strings.ToLower(source)

	containsAny := func(words ...string) bool {
		for _, word := range words {
			if strings.Contains(lower, strings.ToLower(word)) {
				return true
			}
		}
		return false
	}

	switch {
	case containsAny("课后作业", "课堂作业", "作业设计", "拓展任务", "课后任务"):
		return "homework"
	case containsAny("学习目标", "教学目标", "课程目标", "本课目标"):
		return "goal"
	case containsAny("课堂小结", "本课小结", "总结回顾", "知识回顾", "思维导图"):
		return "summary"
	case containsAny("目录", "contents", "课程结构", "学习路径"):
		return "agenda"
	case containsAny(
		"onclick=",
		"addeventlistener",
		"<button",
		"<canvas",
		"dragstart",
		"draggable=",
		"quiz",
		"互动练习",
		"点击选择",
		"拖拽",
	):
		return "interaction"
	case containsAny("数据图表", "统计图", "折线图", "柱状图", "饼图", "chart"):
		return "data"
	case pageNumber == 1:
		return "cover"
	case totalPages > 1 && pageNumber == totalPages:
		return "closing"
	default:
		return "content"
	}
}

// templatePageRoleLabel 把内部页型转换为老师可理解的展示名。
func templatePageRoleLabel(role string) string {
	switch role {
	case "cover":
		return "封面"
	case "agenda":
		return "目录/结构"
	case "goal":
		return "学习目标"
	case "interaction":
		return "互动练习"
	case "data":
		return "数据图表"
	case "summary":
		return "总结回顾"
	case "homework":
		return "课后作业"
	case "closing":
		return "结束页"
	default:
		return "内容讲解"
	}
}

// inferTargetTemplatePageRole 根据当前课件页面方案推断需要匹配的模板样例页类型。
func inferTargetTemplatePageRole(
	page *models.CoursewarePage,
	pageNumber int,
	totalPages int,
) string {
	if pageNumber == 1 {
		return "cover"
	}

	text := ""
	if page != nil {
		text = strings.ToLower(strings.Join([]string{
			page.Title,
			page.Purpose,
			page.ContentSummary,
			page.InteractionType,
			page.VisualFormat,
			page.MediaRequirements,
		}, "\n"))
	}

	containsAny := func(words ...string) bool {
		for _, word := range words {
			if strings.Contains(text, strings.ToLower(word)) {
				return true
			}
		}
		return false
	}

	switch {
	case containsAny("学习目标", "教学目标", "课程目标"):
		return "goal"
	case containsAny("目录", "课程结构", "学习路径"):
		return "agenda"
	case containsAny("作业", "课后任务", "拓展任务"):
		return "homework"
	case containsAny("总结", "小结", "回顾", "思维导图"):
		return "summary"
	case page != nil &&
		(page.InteractionType == "click" ||
			page.InteractionType == "drag" ||
			page.InteractionType == "game" ||
			page.InteractionType == "quiz" ||
			page.IdxInteractionLevel >= 4):
		return "interaction"
	case containsAny("图表", "数据", "统计", "chart"):
		return "data"
	case totalPages > 1 && pageNumber == totalPages:
		return "homework"
	case totalPages > 2 && pageNumber == totalPages-1:
		return "summary"
	case pageNumber == 2:
		return "goal"
	default:
		return "content"
	}
}

// pickTemplateSamplePageIndex 为任意页数模板选择最合适的样例页。
// 优先寻找页型完全匹配的样例；找不到时才按页面位置做比例映射。
func pickTemplateSamplePageIndex(
	samples []string,
	page *models.CoursewarePage,
	pageNumber int,
	totalPages int,
) (int, string) {
	if len(samples) == 0 {
		return -1, ""
	}

	targetRole := inferTargetTemplatePageRole(page, pageNumber, totalPages)

	// 第一优先级：页型完全匹配。
	for idx, sample := range samples {
		role := inferTemplatePageRole(sample, idx+1, len(samples))
		if role == targetRole {
			return idx, templatePageRoleLabel(role)
		}
	}

	// 第二优先级：稳定的位置兜底。
	switch targetRole {
	case "cover":
		return 0, "封面"
	case "goal":
		idx := 1
		if idx >= len(samples) {
			idx = 0
		}
		return idx, "学习目标"
	case "homework", "closing":
		return len(samples) - 1, templatePageRoleLabel(targetRole)
	}

	// 第三优先级：按课件页位置映射到模板样例位置，使十几页模板不会永远只使用末页。
	if len(samples) == 1 || totalPages <= 1 {
		return 0, templatePageRoleLabel(targetRole)
	}

	numerator := (pageNumber - 1) * (len(samples) - 1)
	denominator := totalPages - 1
	idx := (numerator + denominator/2) / denominator

	if idx < 0 {
		idx = 0
	}
	if idx >= len(samples) {
		idx = len(samples) - 1
	}

	role := inferTemplatePageRole(samples[idx], idx+1, len(samples))
	return idx, templatePageRoleLabel(role)
}
