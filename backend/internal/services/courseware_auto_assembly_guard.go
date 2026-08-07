package services

// courseware_auto_assembly_guard.go — 全自动装配页面最终呈现保护
//
// 本文件只服务于全自动装配流程：
//   - 给AI提示词追加1920×1080画布与图片槽位结构约束；
//   - 对新生成页和断点续装页统一重新执行背景裁决；
//   - 更新导航安全壳CSS，使存量页面同步获得导航归位修复；
//   - 注入幂等背景与画布保护样式；
//   - 保留页面元数据和状态。
//
// 背景优先级：
// 页级覆盖 > 课件级背景 > 旧模板提取 > 兼容模板根背景提取 > 无背景。

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

const cwAutoAssemblyLayoutRules = `

## 全自动装配画布边界（最高优先级硬约束）
- 最终页面固定为1920×1080，顶部0—80px由系统导航栏占用，正文只能位于y=80—1080范围内。
- 所有可见文字、卡片、图片、SVG、Canvas、按钮和互动控件必须完整落在画布内。
- 禁止使用position:fixed；禁止使用页面内部滚动条。
- 图片、视频、SVG和Canvas必须设置max-width:100%、max-height:100%和合适的object-fit。
- 内容较多时优先精炼措辞、减少装饰、改用两栏或三栏布局。
- 输出前核对：最右元素不得超过x=1920，最下元素不得超过y=1080。

## 全自动装配图片槽位（最高优先级结构约束）
- 每一个独立静态图片需求都必须对应一个独立叶子节点：
  <div class="img-placeholder" data-desc="该图唯一语义"></div>。
- 严禁一个img-placeholder内部再嵌套其它img-placeholder。
- 外层布局容器不得使用img-placeholder类。
- 多个不同图片需求不得合并成一个大占位。
- 视频、动画、音频需求不得冒充静态图片槽位。
- data-desc只能描述当前单个槽位需要生成的那一张图。
`

var (
	cwAutoAssemblyLegacyBackgroundStyleRe = regexp.MustCompile(
		`(?is)<style\b[^>]*>\s*/\*\s*TEDNA-TPL-BG\s+模板官方背景兜底注入\s*\*/.*?</style>`,
	)

	cwAutoAssemblyPresentationStyleRe = regexp.MustCompile(
		`(?is)<style\b[^>]*>\s*/\*\s*TEDNA-AUTO-PRESENTATION\s*\*/.*?</style>`,
	)
)

// cloneCoursewarePromptWithAutoAssemblyLayoutRules 克隆提示词并追加规则。
func cloneCoursewarePromptWithAutoAssemblyLayoutRules(
	prompt *models.Prompt,
) *models.Prompt {
	if prompt == nil {
		return nil
	}

	cloned := *prompt

	cloned.Content =
		strings.TrimSpace(
			cloned.Content,
		) +
			cwAutoAssemblyLayoutRules

	return &cloned
}

// ensureAutoAssemblyPagePresentation 对数据库当前页执行最终呈现保护。
func (s *CoursewareAutoAssemblyService) ensureAutoAssemblyPagePresentation(
	ctx context.Context,
	pageContext *cwAssemblyPageContext,
	page *models.CoursewarePage,
) (
	string,
	error,
) {
	if pageContext == nil ||
		pageContext.tplInfo == nil {
		return "",
			fmt.Errorf(
				"自动装配模板上下文不可用",
			)
	}

	if page == nil ||
		strings.TrimSpace(
			page.CoursewareID,
		) == "" ||
		page.PageNumber <= 0 {
		return "",
			fmt.Errorf(
				"自动装配页面上下文不可用",
			)
	}

	fresh, err :=
		repository.GetCoursewarePageByNumber(
			ctx,
			page.CoursewareID,
			page.PageNumber,
		)

	if err != nil {
		return "",
			fmt.Errorf(
				"重新读取自动装配页面失败: %w",
				err,
			)
	}

	if fresh == nil ||
		strings.TrimSpace(
			fresh.HTMLContent,
		) == "" {
		return "",
			fmt.Errorf(
				"自动装配页面HTML为空",
			)
	}

	original :=
		fresh.HTMLContent

	// 自动展示保护只接收结构完整的页面。过去这里未校验AI结果，
	// 即使<script>已经被截断，也会继续按原始字符串寻找最后一个</div>，
	// 最终把保护样式插进JavaScript字符串。现在先复用页面完整性闸门：
	// 可确定修复的轻微div误差先修复，其余残缺页面明确返回错误。
	validation :=
		validateRefinedPageHTML(
			"",
			original,
			"",
			true,
		)

	if !validation.OK {
		return "",
			fmt.Errorf(
				"自动装配页面HTML结构不完整: %s",
				validation.Reason,
			)
	}

	sourceHTML := original

	if validation.FixedHTML != "" {
		sourceHTML =
			validation.FixedHTML
	}

	repaired :=
		buildAutoAssemblyPresentationHTML(
			sourceHTML,
			pageContext.tplInfo,
			page.PageNumber,
		)

	if repaired == original {
		return repaired,
			nil
	}

	if err :=
		repository.UpdateCWPageHTML(
			ctx,
			fresh.ID,
			repaired,
			fresh.PlaceholderMap,
			fresh.MatchedComponentIDs,
			fresh.Status,
		); err != nil {
		return "",
			fmt.Errorf(
				"写回自动装配背景与画布保护失败: %w",
				err,
			)
	}

	cwAssemblyLog.Info(
		"自动装配页面背景、导航与画布保护已统一写回",
		"courseware_id", fresh.CoursewareID,
		"page_id", fresh.ID,
		"page_number", fresh.PageNumber,
	)

	return repaired,
		nil
}

// buildAutoAssemblyPresentationHTML 构造最终HTML。
func buildAutoAssemblyPresentationHTML(
	html string,
	templateInfo *cwTemplateInfo,
	pageNumber int,
) string {
	if templateInfo == nil ||
		strings.TrimSpace(
			html,
		) == "" {
		return html
	}

	result :=
		normalizeRootCanvas(
			html,
		)

	// 用最新导航守卫替换旧守卫，只更新平台CSS，不改导航正文。
	result =
		ensureCWNavGuardStyle(
			result,
		)

	// 只有能在真实DOM边界上找到安全插入点时才继续处理。
	// 旧实现直接对原始字符串做LastIndex("</div>")，当AI输出在<script>
	// 字符串中途截断时，会把保护<style>插进JavaScript字符串，导致页面脚本彻底失效。
	// 这里先验证当前HTML存在可配对的<body>或顶层根容器闭合标签；
	// 找不到时保持原HTML不动，交由上游完整性校验处理。
	if findAutoAssemblySafeInsertionIndex(
		result,
	) < 0 {
		return html
	}

	result =
		cwAutoAssemblyLegacyBackgroundStyleRe.
			ReplaceAllString(
				result,
				"",
			)

	result =
		cwAutoAssemblyPresentationStyleRe.
			ReplaceAllString(
				result,
				"",
			)

	backgroundDeclarations :=
		resolveAutoAssemblyBackgroundDeclarations(
			templateInfo,
			pageNumber,
		)

	styleTag :=
		buildAutoAssemblyPresentationStyleTag(
			backgroundDeclarations,
		)

	return insertAutoAssemblyStyleAtDocumentEnd(
		result,
		styleTag,
	)
}

// resolveAutoAssemblyBackgroundDeclarations 按优先级解析本页背景。
func resolveAutoAssemblyBackgroundDeclarations(
	templateInfo *cwTemplateInfo,
	pageNumber int,
) string {
	if templateInfo == nil {
		return ""
	}

	courseLevelDeclarations :=
		resolveUserBgDecls(
			templateInfo,
			pageNumber,
		)

	backgroundDeclarations := ""

	if templateInfo.PageBgSettings != nil {
		if pageSetting, exists :=
			templateInfo.PageBgSettings[pageNumber]; exists {
			backgroundDeclarations =
				resolvePageBgDecls(
					pageSetting,
					pageNumber,
					courseLevelDeclarations,
				)
		}
	}

	if backgroundDeclarations == "" {
		backgroundDeclarations =
			courseLevelDeclarations
	}

	if backgroundDeclarations == "" {
		backgroundDeclarations =
			extractSampleBackgroundDecls(
				templateInfo.SamplePages,
				pageNumber,
			)
	}

	if backgroundDeclarations == "" {
		backgroundDeclarations =
			extractAutoAssemblyTemplateBackgroundDecls(
				templateInfo.SamplePages,
				pageNumber,
			)
	}

	return strings.TrimSpace(
		backgroundDeclarations,
	)
}

// buildAutoAssemblyPresentationStyleTag 构造最终保护样式。
func buildAutoAssemblyPresentationStyleTag(
	backgroundDeclarations string,
) string {
	var backgroundParts []string

	for _, declaration := range strings.Split(
		backgroundDeclarations,
		";",
	) {
		declaration =
			strings.TrimSpace(
				declaration,
			)

		if declaration == "" {
			continue
		}

		declaration =
			strings.ReplaceAll(
				declaration,
				"!important",
				"",
			)

		backgroundParts =
			append(
				backgroundParts,
				strings.TrimSpace(
					declaration,
				)+
					" !important",
			)
	}

	var builder strings.Builder

	builder.WriteString(
		`<style>/* TEDNA-AUTO-PRESENTATION */`,
	)

	builder.WriteString(`
.cw-page:last-of-type{
  width:1920px !important;
  height:1080px !important;
  max-width:1920px !important;
  max-height:1080px !important;
  overflow:hidden !important;
  position:relative;
  box-sizing:border-box !important;
  isolation:isolate;
}`)

	if len(backgroundParts) > 0 {
		builder.WriteString(
			`.cw-page:last-of-type{`,
		)

		builder.WriteString(
			strings.Join(
				backgroundParts,
				";",
			),
		)

		builder.WriteString(
			`}`,
		)
	}

	builder.WriteString(`
.cw-page:last-of-type,
.cw-page:last-of-type *{
  box-sizing:border-box;
}
.cw-page:last-of-type > div:last-of-type{
  min-width:0;
  min-height:0;
  max-width:100%;
  max-height:100%;
  overflow:hidden;
}
.cw-page:last-of-type img,
.cw-page:last-of-type video,
.cw-page:last-of-type svg,
.cw-page:last-of-type canvas,
.cw-page:last-of-type iframe{
  max-width:100% !important;
  max-height:100% !important;
}
.cw-page:last-of-type img,
.cw-page:last-of-type video{
  object-fit:contain;
}
.cw-page:last-of-type p,
.cw-page:last-of-type li,
.cw-page:last-of-type td,
.cw-page:last-of-type th,
.cw-page:last-of-type h1,
.cw-page:last-of-type h2,
.cw-page:last-of-type h3,
.cw-page:last-of-type h4,
.cw-page:last-of-type h5,
.cw-page:last-of-type h6,
.cw-page:last-of-type span{
  overflow-wrap:anywhere;
  word-break:break-word;
}
.cw-page:last-of-type [style*="position:fixed"],
.cw-page:last-of-type [style*="position: fixed"]{
  position:absolute !important;
}
.cw-page:last-of-type [style*="overflow:auto"],
.cw-page:last-of-type [style*="overflow: auto"],
.cw-page:last-of-type [style*="overflow-y:auto"],
.cw-page:last-of-type [style*="overflow-y: auto"],
.cw-page:last-of-type [style*="overflow:scroll"],
.cw-page:last-of-type [style*="overflow: scroll"]{
  overflow:hidden !important;
  overflow-y:hidden !important;
}
</style>`)

	return builder.String()
}

// findAutoAssemblyTagEnd 返回从“<”开始的标签结束“>”位置。
// 属性引号中的“>”不视为标签结束，避免误切带内联脚本或复杂URL的开标签。
func findAutoAssemblyTagEnd(
	html string,
	start int,
) int {
	quote := byte(0)

	for index := start + 1; index < len(html); index++ {
		current := html[index]

		if quote != 0 {
			if current == quote {
				quote = 0
			}
			continue
		}

		if current == '\'' ||
			current == '"' {
			quote = current
			continue
		}

		if current == '>' {
			return index
		}
	}

	return -1
}

// parseAutoAssemblyTagToken 解析一个不含尖括号的标签片段。
// 返回标签名、是否闭合标签、是否自闭合；注释/声明返回空标签名。
func parseAutoAssemblyTagToken(
	token string,
) (
	string,
	bool,
	bool,
) {
	token =
		strings.TrimSpace(
			token,
		)

	if token == "" ||
		strings.HasPrefix(
			token,
			"!",
		) ||
		strings.HasPrefix(
			token,
			"?",
		) {
		return "",
			false,
			false
	}

	closing :=
		strings.HasPrefix(
			token,
			"/",
		)

	if closing {
		token =
			strings.TrimSpace(
				token[1:],
			)
	}

	selfClosing :=
		strings.HasSuffix(
			strings.TrimSpace(
				token,
			),
			"/",
		)

	end := 0

	for end < len(token) {
		current := token[end]

		if current == ' ' ||
			current == '\t' ||
			current == '\r' ||
			current == '\n' ||
			current == '/' {
			break
		}

		end++
	}

	if end == 0 {
		return "",
			closing,
			selfClosing
	}

	return strings.ToLower(
			token[:end],
		),
		closing,
		selfClosing
}

// findAutoAssemblyElementClose 按同名标签深度寻找指定元素的真实闭合标签。
// script/style正文中的“</div>”字符串不会参与根容器配对。
func findAutoAssemblyElementClose(
	html string,
	openStart int,
	wanted string,
) int {
	lower :=
		strings.ToLower(
			html,
		)

	depth := 0
	rawTag := ""

	for cursor := openStart; cursor < len(html); {
		relative :=
			strings.Index(
				html[cursor:],
				"<",
			)

		if relative < 0 {
			return -1
		}

		start :=
			cursor +
				relative

		if strings.HasPrefix(
			lower[start:],
			"<!--",
		) {
			commentEnd :=
				strings.Index(
					lower[start+4:],
					"-->",
				)

			if commentEnd < 0 {
				return -1
			}

			cursor =
				start +
					4 +
					commentEnd +
					3
			continue
		}

		end :=
			findAutoAssemblyTagEnd(
				html,
				start,
			)

		if end < 0 {
			return -1
		}

		name, closing, selfClosing :=
			parseAutoAssemblyTagToken(
				html[start+1 : end],
			)

		if rawTag != "" {
			if closing &&
				name == rawTag {
				rawTag = ""
			}

			cursor =
				end +
					1
			continue
		}

		if name == "script" ||
			name == "style" {
			if !closing &&
				!selfClosing {
				rawTag = name
			}

			cursor =
				end +
					1
			continue
		}

		if name == wanted {
			if closing {
				depth--

				if depth == 0 {
					return start
				}

				if depth < 0 {
					return -1
				}
			} else if !selfClosing {
				depth++
			}
		}

		cursor =
			end +
				1
	}

	return -1
}

// skipAutoAssemblyWhitespaceAndComments 跳过顶层空白和HTML注释。
func skipAutoAssemblyWhitespaceAndComments(
	html string,
	start int,
) int {
	lower :=
		strings.ToLower(
			html,
		)

	cursor := start

	for cursor < len(html) {
		for cursor < len(html) {
			current := html[cursor]

			if current != ' ' &&
				current != '\t' &&
				current != '\r' &&
				current != '\n' {
				break
			}

			cursor++
		}

		if cursor >= len(html) ||
			!strings.HasPrefix(
				lower[cursor:],
				"<!--",
			) {
			break
		}

		commentEnd :=
			strings.Index(
				lower[cursor+4:],
				"-->",
			)

		if commentEnd < 0 {
			return -1
		}

		cursor =
			cursor +
				4 +
				commentEnd +
				3
	}

	return cursor
}

// findAutoAssemblySafeInsertionIndex 返回保护样式的安全插入点。
// 完整文档放在真实</body>之前；片段放在最后一个顶层页面根容器闭合标签之前。
// 找不到配对闭合标签时返回-1，禁止退化到原始字符串LastIndex。
func findAutoAssemblySafeInsertionIndex(
	html string,
) int {
	if strings.TrimSpace(
		html,
	) == "" {
		return -1
	}

	lower :=
		strings.ToLower(
			html,
		)

	if bodyOpen :=
		strings.Index(
			lower,
			"<body",
		); bodyOpen >= 0 {
		return findAutoAssemblyElementClose(
			html,
			bodyOpen,
			"body",
		)
	}

	allowedRoots :=
		map[string]bool{
			"div":     true,
			"section": true,
			"main":    true,
			"article": true,
		}

	cursor :=
		skipAutoAssemblyWhitespaceAndComments(
			html,
			0,
		)

	if cursor < 0 {
		return -1
	}

	lastClose := -1

	for cursor < len(html) {
		if html[cursor] != '<' {
			break
		}

		end :=
			findAutoAssemblyTagEnd(
				html,
				cursor,
			)

		if end < 0 {
			return -1
		}

		name, closing, selfClosing :=
			parseAutoAssemblyTagToken(
				html[cursor+1 : end],
			)

		if closing ||
			selfClosing ||
			!allowedRoots[name] {
			break
		}

		closeIndex :=
			findAutoAssemblyElementClose(
				html,
				cursor,
				name,
			)

		if closeIndex < 0 {
			return -1
		}

		lastClose =
			closeIndex

		closeEnd :=
			findAutoAssemblyTagEnd(
				html,
				closeIndex,
			)

		if closeEnd < 0 {
			return -1
		}

		cursor =
			skipAutoAssemblyWhitespaceAndComments(
				html,
				closeEnd+1,
			)

		if cursor < 0 {
			return -1
		}
	}

	return lastClose
}

// insertAutoAssemblyStyleAtDocumentEnd 将保护样式放在真实页面DOM末尾。
// 绝不在script/style正文或JavaScript字符串中的“</div>”前插入。
func insertAutoAssemblyStyleAtDocumentEnd(
	html string,
	styleTag string,
) string {
	if strings.TrimSpace(
		styleTag,
	) == "" {
		return html
	}

	insertionIndex :=
		findAutoAssemblySafeInsertionIndex(
			html,
		)

	if insertionIndex < 0 {
		return html
	}

	return html[:insertionIndex] +
		styleTag +
		html[insertionIndex:]
}
