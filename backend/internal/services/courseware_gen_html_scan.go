package services

// courseware_gen_html_scan.go — 课件HTML词法扫描与AI输出安全提取
//
// 课件页面既可以是完整HTML文档，也可以是以.cw-page为根的HTML片段。
// 片段根节点后允许存在同级<script>/<style>，因此不能用原始字符串中的
// “最后一个</div>”判断页面终点：JavaScript字符串和模板字符串也可能包含
// <div>...</div>，旧做法会把真实</script>及后续样式截掉。
//
// 本文件提供统一的轻量词法扫描：
//   - 识别真实HTML标签、注释和属性引号；
//   - script/style按HTML原始文本元素处理，内部伪标签不参与结构计数；
//   - 提取完整文档或课件片段，并保留根节点后的同级脚本、样式和双画布兄弟；
//   - 输出结构摘要供生成、微调和完整性校验共用。
//
// 该扫描器不重排DOM、不修复业务结构，只负责边界识别和计数。

import (
	"fmt"
	"strings"
)

type cwHTMLTagToken struct {
	Start       int
	End         int
	Name        string
	Closing     bool
	SelfClosing bool
}

type cwHTMLStructure struct {
	Tokens         []cwHTMLTagToken
	DoctypeStart   int
	DivOpen        int
	DivClose       int
	ScriptOpen     int
	ScriptClose    int
	StyleOpen      int
	StyleClose     int
	EndsMidTag     bool
	UnclosedRawTag string
}

type cwHTMLExtractionResult struct {
	Cleaned  string
	HTML     string
	RootName string
	Complete bool
	Start    int
	End      int
}

func cwDescribeHTMLStructure(source string) string {
	structure := cwScanHTMLStructure(source)
	return fmt.Sprintf(
		"bytes=%d div(%d/%d) script(%d/%d) style(%d/%d) mid_tag=%t unclosed_raw=%q",
		len(source),
		structure.DivOpen,
		structure.DivClose,
		structure.ScriptOpen,
		structure.ScriptClose,
		structure.StyleOpen,
		structure.StyleClose,
		structure.EndsMidTag,
		structure.UnclosedRawTag,
	)
}

func cwScanHTMLStructure(source string) cwHTMLStructure {
	result := cwHTMLStructure{
		Tokens:       make([]cwHTMLTagToken, 0, 64),
		DoctypeStart: -1,
	}

	lower := strings.ToLower(source)

	for cursor := 0; cursor < len(source); {
		relative := strings.IndexByte(source[cursor:], '<')
		if relative < 0 {
			break
		}

		start := cursor + relative

		if strings.HasPrefix(lower[start:], "<!--") {
			commentEnd := strings.Index(lower[start+4:], "-->")
			if commentEnd < 0 {
				result.EndsMidTag = true
				break
			}
			cursor = start + 4 + commentEnd + 3
			continue
		}

		if strings.HasPrefix(lower[start:], "<![cdata[") {
			cdataEnd := strings.Index(lower[start+9:], "]]>")
			if cdataEnd < 0 {
				result.EndsMidTag = true
				break
			}
			cursor = start + 9 + cdataEnd + 3
			continue
		}

		end := cwFindHTMLTagEnd(source, start)
		if end < 0 {
			result.EndsMidTag = true
			break
		}

		rawToken := strings.TrimSpace(source[start+1 : end])
		if strings.HasPrefix(
			strings.ToLower(rawToken),
			"!doctype",
		) {
			if result.DoctypeStart < 0 {
				result.DoctypeStart = start
			}
			cursor = end + 1
			continue
		}

		name, closing, selfClosing := cwParseHTMLTagToken(rawToken)
		if name == "" {
			cursor = end + 1
			continue
		}

		token := cwHTMLTagToken{
			Start:       start,
			End:         end + 1,
			Name:        name,
			Closing:     closing,
			SelfClosing: selfClosing,
		}
		result.Tokens = append(result.Tokens, token)
		cwCountHTMLStructureToken(&result, token)

		if !closing && !selfClosing && (name == "script" || name == "style") {
			closeToken, ok := cwFindRawTextClose(source, lower, end+1, name)
			if !ok {
				result.UnclosedRawTag = name
				break
			}

			result.Tokens = append(result.Tokens, closeToken)
			cwCountHTMLStructureToken(&result, closeToken)
			cursor = closeToken.End
			continue
		}

		cursor = end + 1
	}

	return result
}

func cwCountHTMLStructureToken(
	result *cwHTMLStructure,
	token cwHTMLTagToken,
) {
	if result == nil {
		return
	}

	switch token.Name {
	case "div":
		if token.Closing {
			result.DivClose++
		} else if !token.SelfClosing {
			result.DivOpen++
		}
	case "script":
		if token.Closing {
			result.ScriptClose++
		} else if !token.SelfClosing {
			result.ScriptOpen++
		}
	case "style":
		if token.Closing {
			result.StyleClose++
		} else if !token.SelfClosing {
			result.StyleOpen++
		}
	}
}

func cwFindHTMLTagEnd(source string, start int) int {
	if start < 0 || start >= len(source) || source[start] != '<' {
		return -1
	}

	var quote byte

	for index := start + 1; index < len(source); index++ {
		current := source[index]

		if quote != 0 {
			if current == quote {
				quote = 0
			}
			continue
		}

		if current == '"' || current == '\'' {
			quote = current
			continue
		}

		if current == '>' {
			return index
		}
	}

	return -1
}

func cwParseHTMLTagToken(token string) (
	name string,
	closing bool,
	selfClosing bool,
) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" ||
		strings.HasPrefix(trimmed, "!") ||
		strings.HasPrefix(trimmed, "?") {
		return "", false, true
	}

	if strings.HasPrefix(trimmed, "/") {
		closing = true
		trimmed = strings.TrimSpace(trimmed[1:])
	}

	selfClosing = strings.HasSuffix(trimmed, "/")

	end := 0
	for end < len(trimmed) && cwIsHTMLTagNameByte(trimmed[end]) {
		end++
	}

	if end == 0 {
		return "", closing, selfClosing
	}

	return strings.ToLower(trimmed[:end]),
		closing,
		selfClosing
}

func cwIsHTMLTagNameByte(value byte) bool {
	return value == '-' ||
		value == ':' ||
		value == '_' ||
		(value >= 'a' && value <= 'z') ||
		(value >= 'A' && value <= 'Z') ||
		(value >= '0' && value <= '9')
}

func cwFindRawTextClose(
	source string,
	lower string,
	from int,
	name string,
) (
	cwHTMLTagToken,
	bool,
) {
	needle := "</" + name
	search := from

	for search < len(source) {
		relative := strings.Index(lower[search:], needle)
		if relative < 0 {
			return cwHTMLTagToken{}, false
		}

		start := search + relative
		afterName := start + len(needle)

		if afterName < len(source) &&
			cwIsHTMLTagNameByte(source[afterName]) {
			search = afterName
			continue
		}

		end := cwFindHTMLTagEnd(source, start)
		if end < 0 {
			return cwHTMLTagToken{}, false
		}

		tokenName, closing, selfClosing :=
			cwParseHTMLTagToken(source[start+1 : end])

		if closing && tokenName == name {
			return cwHTMLTagToken{
				Start:       start,
				End:         end + 1,
				Name:        tokenName,
				Closing:     true,
				SelfClosing: selfClosing,
			}, true
		}

		search = end + 1
	}

	return cwHTMLTagToken{}, false
}

func cwExtractCoursewareHTMLFromAIOutput(
	aiOutput string,
) cwHTMLExtractionResult {
	cleaned := strings.TrimSpace(
		cwGenStripCodeFences(
			strings.TrimSpace(aiOutput),
		),
	)

	result := cwHTMLExtractionResult{
		Cleaned: cleaned,
		Start:   -1,
		End:     -1,
	}

	if cleaned == "" {
		return result
	}

	structure := cwScanHTMLStructure(cleaned)

	docStart := structure.DoctypeStart

	htmlTokenIndex := cwFindFirstOpeningToken(
		structure.Tokens,
		map[string]bool{"html": true},
	)

	if htmlTokenIndex >= 0 {
		htmlStart := structure.Tokens[htmlTokenIndex].Start
		if docStart < 0 || htmlStart < docStart {
			docStart = htmlStart
		}

		if end, ok := cwFindMatchingHTMLElementEnd(
			structure.Tokens,
			htmlTokenIndex,
		); ok {
			result.HTML = strings.TrimSpace(cleaned[docStart:end])
			result.RootName = "html"
			result.Complete = true
			result.Start = docStart
			result.End = end
			return result
		}
	}

	if docStart >= 0 {
		result.HTML = strings.TrimSpace(cleaned[docStart:])
		result.RootName = "html"
		result.Start = docStart
		result.End = len(cleaned)
		return result
	}

	rootTokenIndex := cwFindFirstOpeningToken(
		structure.Tokens,
		map[string]bool{
			"div":     true,
			"section": true,
			"main":    true,
			"article": true,
		},
	)

	if rootTokenIndex < 0 {
		if strings.Contains(cleaned, "<") &&
			strings.Contains(cleaned, ">") {
			result.HTML = cleaned
			result.Start = 0
			result.End = len(cleaned)
		}
		return result
	}

	root := structure.Tokens[rootTokenIndex]
	result.RootName = root.Name
	result.Start = root.Start

	rootEnd, complete :=
		cwFindMatchingHTMLElementEnd(
			structure.Tokens,
			rootTokenIndex,
		)

	if !complete {
		result.HTML = strings.TrimSpace(cleaned[root.Start:])
		result.End = len(cleaned)
		return result
	}

	end := rootEnd
	cursor := rootEnd
	fragmentComplete := true

	for {
		next, gapEnd := cwSkipHTMLFragmentGap(cleaned, cursor)
		if gapEnd > end {
			end = gapEnd
		}
		if next >= len(cleaned) {
			break
		}

		siblingIndex := cwFindTokenStartingAt(
			structure.Tokens,
			next,
		)
		if siblingIndex < 0 {
			break
		}

		sibling := structure.Tokens[siblingIndex]
		if sibling.Closing ||
			!cwIsAllowedCoursewareTopLevelSibling(
				sibling.Name,
			) {
			break
		}

		siblingEnd, siblingComplete :=
			cwFindMatchingHTMLElementEnd(
				structure.Tokens,
				siblingIndex,
			)

		if !siblingComplete {
			end = len(cleaned)
			fragmentComplete = false
			break
		}

		end = siblingEnd
		cursor = siblingEnd
	}

	result.HTML = strings.TrimSpace(cleaned[root.Start:end])
	result.Complete = fragmentComplete
	result.End = end

	return result
}

func cwFindFirstOpeningToken(
	tokens []cwHTMLTagToken,
	allowed map[string]bool,
) int {
	best := -1

	for index, token := range tokens {
		if token.Closing ||
			!allowed[token.Name] {
			continue
		}

		if best < 0 ||
			token.Start < tokens[best].Start {
			best = index
		}
	}

	return best
}

func cwFindMatchingHTMLElementEnd(
	tokens []cwHTMLTagToken,
	startIndex int,
) (
	int,
	bool,
) {
	if startIndex < 0 ||
		startIndex >= len(tokens) {
		return 0, false
	}

	start := tokens[startIndex]
	if start.Closing {
		return 0, false
	}
	if start.SelfClosing {
		return start.End, true
	}

	depth := 0

	for index := startIndex; index < len(tokens); index++ {
		token := tokens[index]
		if token.Name != start.Name {
			continue
		}

		if token.Closing {
			depth--
			if depth == 0 {
				return token.End, true
			}
			continue
		}

		if !token.SelfClosing {
			depth++
		}
	}

	return 0, false
}

func cwSkipHTMLFragmentGap(
	source string,
	cursor int,
) (
	next int,
	safeEnd int,
) {
	next = cursor
	safeEnd = cursor
	lower := strings.ToLower(source)

	for next < len(source) {
		for next < len(source) {
			switch source[next] {
			case ' ', '\t', '\r', '\n':
				next++
				safeEnd = next
			default:
				goto gapToken
			}
		}

	gapToken:
		if next >= len(source) ||
			!strings.HasPrefix(lower[next:], "<!--") {
			break
		}

		commentEnd := strings.Index(lower[next+4:], "-->")
		if commentEnd < 0 {
			break
		}

		next = next + 4 + commentEnd + 3
		safeEnd = next
	}

	return next, safeEnd
}

func cwFindTokenStartingAt(
	tokens []cwHTMLTagToken,
	start int,
) int {
	for index, token := range tokens {
		if token.Start == start {
			return index
		}
		if token.Start > start {
			break
		}
	}

	return -1
}

func cwIsAllowedCoursewareTopLevelSibling(
	name string,
) bool {
	switch name {
	case "div",
		"section",
		"main",
		"article",
		"header",
		"nav",
		"script",
		"style",
		"link",
		"meta",
		"template":
		return true
	default:
		return false
	}
}
