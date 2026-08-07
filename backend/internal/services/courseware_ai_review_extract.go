package services

// courseware_ai_review_extract.go
//
// 课件 AI 审核的页面静态证据提取器。
//
// 审核目标不是只读页面文字，还要分析：
//   - 页面初始状态是否直接暴露答案；
//   - 学生是否需要点击、输入、拖拽或提交后才得到反馈；
//   - 点击事件最终调用了哪些函数；
//   - 函数修改了哪些 DOM 节点和状态变量；
//   - CSS 是否通过 display、visibility、opacity 等控制答案显隐；
//   - 是否存在动态执行、外部脚本或复杂运行时逻辑，需要人工操作复核。
//
// 设计边界：
//   - 纯静态分析，不执行课件 JavaScript；
//   - 不写数据库、不调用 AI；
//   - 结果只作为 AI 审核证据，不能冒充浏览器真实运行测试；
//   - 无法可靠解析时必须标记 ManualReviewRequired，绝不默认判定正常。

import (
	"crypto/sha256"
	"encoding/hex"
	htmlstd "html"
	"regexp"
	"sort"
	"strings"

	"tedna/internal/models"
)

const (
	cwAIReviewVisibleTextMaxRunes = 5000
	cwAIReviewEvidenceMaxRunes    = 900
	cwAIReviewMaxFunctions        = 24
	cwAIReviewMaxCSSRules         = 20
	cwAIReviewCallDepth           = 3
)

var (
	cwAIReviewScriptRe = regexp.MustCompile(
		`(?is)<script\b[^>]*>(.*?)</script\s*>`,
	)
	cwAIReviewStyleRe = regexp.MustCompile(
		`(?is)<style\b[^>]*>(.*?)</style\s*>`,
	)
	cwAIReviewCommentRe = regexp.MustCompile(
		`(?is)<!--.*?-->|/\*.*?\*/|(?m)//[^\r\n]*`,
	)
	cwAIReviewTagRe = regexp.MustCompile(
		`(?is)<[^>]+>`,
	)
	cwAIReviewSpaceRe = regexp.MustCompile(
		`[\t\r\n ]+`,
	)

	cwAIReviewInlineEventRe = regexp.MustCompile(
		`(?is)\bon(click|input|change|submit|dragstart|dragover|drop|pointerdown|pointermove|pointerup|mousedown|mousemove|mouseup|touchstart|touchmove|touchend|blur|keydown|keyup)\s*=\s*["']([^"']*)["']`,
	)
	cwAIReviewListenerRe = regexp.MustCompile(
		`(?is)([A-Za-z_$][\w$.\[\]'"]*)\s*\.addEventListener\s*\(\s*["']([^"']+)["']\s*,\s*([A-Za-z_$][\w$]*|function\s*\([^)]*\)|(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>)`,
	)
	cwAIReviewFunctionStartRe = regexp.MustCompile(
		`(?m)(?:function\s+([A-Za-z_$][\w$]*)\s*\([^)]*\)\s*\{|(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>\s*\{)`,
	)
	cwAIReviewCallRe = regexp.MustCompile(
		`\b([A-Za-z_$][\w$]*)\s*\(`,
	)
	cwAIReviewVariableRe = regexp.MustCompile(
		`(?m)\b(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*([^\r\n;]{1,180})`,
	)
	cwAIReviewAssignRe = regexp.MustCompile(
		`(?m)\b([A-Za-z_$][\w$]*)\s*(?:=|\+=|-=|\+\+|--)\s*([^\r\n;]{0,160})`,
	)
	cwAIReviewDOMIDRe = regexp.MustCompile(
		`(?is)getElementById\s*\(\s*["']([^"']+)["']\s*\)`,
	)
	cwAIReviewSelectorRe = regexp.MustCompile(
		`(?is)querySelector(?:All)?\s*\(\s*["']([^"']+)["']\s*\)`,
	)

	cwAIReviewAnswerElementRe = regexp.MustCompile(
		`(?is)<[^>]+(?:id|class|data-[\w-]+)\s*=\s*["'][^"']*(?:answer|solution|result|feedback|解析|答案|正确答案)[^"']*["'][^>]*>`,
	)
	cwAIReviewHiddenAnswerRe = regexp.MustCompile(
		`(?is)(?:answer|solution|result|feedback|解析|答案|正确答案)[^{}<>]{0,260}(?:display\s*:\s*none|visibility\s*:\s*hidden|opacity\s*:\s*0|\bhidden\b)`,
	)
	cwAIReviewVisibleAnswerTextRe = regexp.MustCompile(
		`(?is)(?:正确答案|参考答案|答案\s*[:：]|解析\s*[:：])`,
	)
	cwAIReviewEmbeddedAnswerRe = regexp.MustCompile(
		`(?is)\b(?:data-answer|data-correct|correctAnswer|rightAnswer|expectedAnswer|solutionKey)\b`,
	)
	cwAIReviewRevealRe = regexp.MustCompile(
		`(?is)\b(?:reveal|showAnswer|displayAnswer|toggleAnswer|解锁答案|揭晓答案|显示答案)\b`,
	)
	cwAIReviewRetryRe = regexp.MustCompile(
		`(?is)\b(?:reset|retry|restart|tryAgain|重新开始|再试一次|重置|重试)\b`,
	)
	cwAIReviewDynamicRe = regexp.MustCompile(
		`(?is)\beval\s*\(|new\s+Function\s*\(|document\.write\s*\(|setAttribute\s*\(\s*["']on`,
	)
	cwAIReviewExternalRuntimeRe = regexp.MustCompile(
		`(?is)<script\b[^>]*\bsrc\s*=|\bfetch\s*\(|XMLHttpRequest|WebSocket\s*\(`,
	)
	cwAIReviewDOMContentLoadedRe = regexp.MustCompile(
		`(?is)DOMContentLoaded|window\.onload|\bonload\s*=`,
	)
)

// cwAIReviewJSFunction 保存静态提取出的函数代码。
type cwAIReviewJSFunction struct {
	Name string
	Code string
}

// BuildCWAIReviewPageDigest 为单页构建内容与互动双重摘要。
func BuildCWAIReviewPageDigest(
	page *models.CoursewarePage,
) models.CWAIReviewPageDigest {
	if page == nil {
		return models.CWAIReviewPageDigest{}
	}

	htmlContent := strings.TrimSpace(page.HTMLContent)

	return models.CWAIReviewPageDigest{
		PageID:            page.ID,
		PageNumber:        page.PageNumber,
		Title:             strings.TrimSpace(page.Title),
		Purpose:           strings.TrimSpace(page.Purpose),
		ContentSummary:    strings.TrimSpace(page.ContentSummary),
		InteractionType:   strings.TrimSpace(page.InteractionType),
		VisualFormat:      strings.TrimSpace(page.VisualFormat),
		MediaRequirements: strings.TrimSpace(page.MediaRequirements),
		PageIndex:         strings.TrimSpace(page.PageIndex),
		VisibleText:       cwAIReviewExtractVisibleText(htmlContent),
		HTMLHash:          cwAIReviewSHA256(htmlContent),
		Interaction:       cwAIReviewExtractInteraction(page),
	}
}

// cwAIReviewExtractInteraction 提取事件、可达函数、状态和答案暴露证据。
func cwAIReviewExtractInteraction(
	page *models.CoursewarePage,
) models.CWAIReviewInteractionEvidence {
	htmlContent := strings.TrimSpace(page.HTMLContent)
	scripts := cwAIReviewExtractBlocks(
		htmlContent,
		cwAIReviewScriptRe,
	)
	styles := cwAIReviewExtractBlocks(
		htmlContent,
		cwAIReviewStyleRe,
	)
	scriptText := strings.Join(scripts, "\n\n")

	contract := validateGeneratedPageInteraction(
		page.InteractionType,
		htmlContent,
	)

	events := cwAIReviewExtractEvents(
		htmlContent,
		scriptText,
	)
	functions := cwAIReviewExtractFunctions(
		scriptText,
	)
	reachable := cwAIReviewResolveReachableFunctions(
		events,
		functions,
	)

	combinedReachableCode := cwAIReviewReachableCode(
		reachable,
	)

	domTargets := cwAIReviewExtractDOMTargets(
		combinedReachableCode,
	)
	stateVariables := cwAIReviewExtractStateVariables(
		combinedReachableCode,
	)
	cssRules := cwAIReviewExtractRelevantCSSRules(
		styles,
		domTargets,
	)

	initialSignals, riskFlags, manualReview :=
		cwAIReviewDetectInteractionRisks(
			page,
			htmlContent,
			scriptText,
			events,
			reachable,
			contract,
		)

	return models.CWAIReviewInteractionEvidence{
		DeclaredType: strings.TrimSpace(
			page.InteractionType,
		),
		ContractOK:     contract.OK,
		ContractReason: strings.TrimSpace(contract.Reason),
		ContractDetail: strings.TrimSpace(contract.Detail),

		Events:             events,
		ReachableFunctions: reachable,
		StateVariables:     stateVariables,
		DOMTargets:         domTargets,
		CSSStateRules:      cssRules,

		InitialExposureSignals: initialSignals,
		RiskFlags:              riskFlags,

		ScriptRuneCount:      len([]rune(scriptText)),
		ManualReviewRequired: manualReview,
	}
}

// cwAIReviewExtractBlocks 提取 script/style 块正文。
func cwAIReviewExtractBlocks(
	content string,
	re *regexp.Regexp,
) []string {
	matches := re.FindAllStringSubmatch(content, -1)
	out := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		value := strings.TrimSpace(match[1])
		if value == "" {
			continue
		}
		out = append(out, value)
	}

	return out
}

// cwAIReviewExtractVisibleText 去掉 script/style/tag 后提取可见文本近似值。
//
// 注意：这是静态近似，不等同于浏览器布局后的真实可见文本。
// 答案显隐判断由 InteractionEvidence 单独负责。
func cwAIReviewExtractVisibleText(
	content string,
) string {
	withoutScripts := cwAIReviewScriptRe.ReplaceAllString(
		content,
		" ",
	)
	withoutStyles := cwAIReviewStyleRe.ReplaceAllString(
		withoutScripts,
		" ",
	)
	withoutComments := cwAIReviewCommentRe.ReplaceAllString(
		withoutStyles,
		" ",
	)
	withoutTags := cwAIReviewTagRe.ReplaceAllString(
		withoutComments,
		" ",
	)

	text := htmlstd.UnescapeString(withoutTags)
	text = cwAIReviewSpaceRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return cwAIReviewTruncate(
		text,
		cwAIReviewVisibleTextMaxRunes,
	)
}

// cwAIReviewExtractEvents 提取内联事件与 addEventListener。
func cwAIReviewExtractEvents(
	htmlContent string,
	scriptText string,
) []models.CWAIReviewInteractionEvent {
	events := make(
		[]models.CWAIReviewInteractionEvent,
		0,
	)

	for _, match := range cwAIReviewInlineEventRe.FindAllStringSubmatch(
		htmlContent,
		-1,
	) {
		if len(match) < 3 {
			continue
		}

		events = append(
			events,
			models.CWAIReviewInteractionEvent{
				EventType: strings.ToLower(
					strings.TrimSpace(match[1]),
				),
				Trigger: "inline",
				Handler: strings.TrimSpace(match[2]),
				Evidence: cwAIReviewTruncate(
					strings.TrimSpace(match[0]),
					cwAIReviewEvidenceMaxRunes,
				),
			},
		)
	}

	for _, match := range cwAIReviewListenerRe.FindAllStringSubmatch(
		scriptText,
		-1,
	) {
		if len(match) < 4 {
			continue
		}

		events = append(
			events,
			models.CWAIReviewInteractionEvent{
				EventType: strings.ToLower(
					strings.TrimSpace(match[2]),
				),
				Trigger: strings.TrimSpace(match[1]),
				Handler: strings.TrimSpace(match[3]),
				Evidence: cwAIReviewTruncate(
					strings.TrimSpace(match[0]),
					cwAIReviewEvidenceMaxRunes,
				),
			},
		)
	}

	return cwAIReviewDedupeEvents(events)
}

// cwAIReviewExtractFunctions 提取常规函数和块级箭头函数。
func cwAIReviewExtractFunctions(
	scriptText string,
) map[string]cwAIReviewJSFunction {
	out := make(
		map[string]cwAIReviewJSFunction,
	)

	indexes := cwAIReviewFunctionStartRe.FindAllStringSubmatchIndex(
		scriptText,
		-1,
	)

	for _, idx := range indexes {
		if len(idx) < 6 {
			continue
		}

		name := ""
		if idx[2] >= 0 && idx[3] >= 0 {
			name = scriptText[idx[2]:idx[3]]
		} else if idx[4] >= 0 && idx[5] >= 0 {
			name = scriptText[idx[4]:idx[5]]
		}

		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		matchEnd := idx[1]
		matchStart := idx[0]
		openRel := strings.LastIndex(
			scriptText[matchStart:matchEnd],
			"{",
		)
		if openRel < 0 {
			continue
		}

		openPos := matchStart + openRel
		closePos := cwAIReviewFindBalancedBrace(
			scriptText,
			openPos,
		)
		if closePos <= openPos {
			continue
		}

		code := strings.TrimSpace(
			scriptText[matchStart : closePos+1],
		)
		out[name] = cwAIReviewJSFunction{
			Name: name,
			Code: cwAIReviewTruncate(
				code,
				2400,
			),
		}

		if len(out) >= cwAIReviewMaxFunctions*3 {
			break
		}
	}

	return out
}

// cwAIReviewFindBalancedBrace 查找函数块配对的右花括号。
//
// 该实现处理单双引号、模板字符串和转义字符；
// 不执行 JavaScript，因此仍可能遇到极端语法，届时上层会要求人工复核。
func cwAIReviewFindBalancedBrace(
	content string,
	openPos int,
) int {
	if openPos < 0 ||
		openPos >= len(content) ||
		content[openPos] != '{' {
		return -1
	}

	depth := 0
	var quote byte
	escaped := false

	for i := openPos; i < len(content); i++ {
		ch := content[i]

		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		if ch == '\'' ||
			ch == '"' ||
			ch == '`' {
			quote = ch
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

// cwAIReviewResolveReachableFunctions 从事件入口递归展开最多三层函数调用。
func cwAIReviewResolveReachableFunctions(
	events []models.CWAIReviewInteractionEvent,
	functions map[string]cwAIReviewJSFunction,
) []models.CWAIReviewReachableFunction {
	type queueItem struct {
		Name  string
		Depth int
	}

	queue := make([]queueItem, 0)
	queued := make(map[string]bool)

	for _, event := range events {
		for _, name := range cwAIReviewFindCalls(
			event.Handler + "\n" + event.Evidence,
		) {
			if _, exists := functions[name]; !exists {
				continue
			}
			if queued[name] {
				continue
			}
			queued[name] = true
			queue = append(queue, queueItem{
				Name:  name,
				Depth: 0,
			})
		}
	}

	result := make(
		[]models.CWAIReviewReachableFunction,
		0,
	)

	for len(queue) > 0 &&
		len(result) < cwAIReviewMaxFunctions {
		item := queue[0]
		queue = queue[1:]

		fn, exists := functions[item.Name]
		if !exists {
			continue
		}

		result = append(
			result,
			models.CWAIReviewReachableFunction{
				Name:  item.Name,
				Depth: item.Depth,
				Evidence: cwAIReviewTruncate(
					fn.Code,
					cwAIReviewEvidenceMaxRunes,
				),
			},
		)

		if item.Depth >= cwAIReviewCallDepth {
			continue
		}

		for _, child := range cwAIReviewFindCalls(fn.Code) {
			if _, exists := functions[child]; !exists {
				continue
			}
			if queued[child] {
				continue
			}

			queued[child] = true
			queue = append(queue, queueItem{
				Name:  child,
				Depth: item.Depth + 1,
			})
		}
	}

	return result
}

// cwAIReviewFindCalls 提取函数调用名并排除 JavaScript 关键字。
func cwAIReviewFindCalls(
	content string,
) []string {
	ignored := map[string]bool{
		"if": true, "for": true, "while": true,
		"switch": true, "catch": true, "function": true,
		"return": true, "typeof": true, "new": true,
		"setTimeout": true, "setInterval": true,
		"requestAnimationFrame": true,
		"querySelector":         true, "querySelectorAll": true,
		"getElementById": true, "addEventListener": true,
		"preventDefault": true, "stopPropagation": true,
		"parseInt": true, "parseFloat": true,
		"Number": true, "String": true, "Boolean": true,
		"Array": true, "Object": true, "Math": true,
		"Date": true, "JSON": true, "Promise": true,
		"alert": true, "confirm": true, "prompt": true,
	}

	seen := make(map[string]bool)
	out := make([]string, 0)

	for _, match := range cwAIReviewCallRe.FindAllStringSubmatch(
		content,
		-1,
	) {
		if len(match) < 2 {
			continue
		}

		name := strings.TrimSpace(match[1])
		if name == "" ||
			ignored[name] ||
			seen[name] {
			continue
		}

		seen[name] = true
		out = append(out, name)
	}

	return out
}

// cwAIReviewReachableCode 把可达函数证据拼成后续状态扫描输入。
func cwAIReviewReachableCode(
	functions []models.CWAIReviewReachableFunction,
) string {
	var builder strings.Builder

	for _, fn := range functions {
		builder.WriteString(fn.Evidence)
		builder.WriteString("\n")
	}

	return builder.String()
}

// cwAIReviewExtractDOMTargets 提取事件链实际读写的 DOM 目标。
func cwAIReviewExtractDOMTargets(
	content string,
) []string {
	seen := make(map[string]bool)
	out := make([]string, 0)

	for _, match := range cwAIReviewDOMIDRe.FindAllStringSubmatch(
		content,
		-1,
	) {
		if len(match) < 2 {
			continue
		}
		value := "#" + strings.TrimSpace(match[1])
		if value == "#" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}

	for _, match := range cwAIReviewSelectorRe.FindAllStringSubmatch(
		content,
		-1,
	) {
		if len(match) < 2 {
			continue
		}
		value := strings.TrimSpace(match[1])
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}

	sort.Strings(out)
	return out
}

// cwAIReviewExtractStateVariables 提取可达事件链中的状态变量和赋值。
func cwAIReviewExtractStateVariables(
	content string,
) []string {
	seen := make(map[string]bool)
	out := make([]string, 0)

	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		out = append(out, cwAIReviewTruncate(value, 220))
	}

	for _, match := range cwAIReviewVariableRe.FindAllStringSubmatch(
		content,
		-1,
	) {
		if len(match) < 3 {
			continue
		}
		appendValue(
			strings.TrimSpace(match[1]) +
				" = " +
				strings.TrimSpace(match[2]),
		)
	}

	for _, match := range cwAIReviewAssignRe.FindAllStringSubmatch(
		content,
		-1,
	) {
		if len(match) < 3 {
			continue
		}
		appendValue(
			strings.TrimSpace(match[1]) +
				" ← " +
				strings.TrimSpace(match[2]),
		)
	}

	if len(out) > 30 {
		out = out[:30]
	}

	return out
}

// cwAIReviewExtractRelevantCSSRules 保留与显隐和事件目标有关的 CSS。
func cwAIReviewExtractRelevantCSSRules(
	styles []string,
	domTargets []string,
) []string {
	seen := make(map[string]bool)
	out := make([]string, 0)

	for _, style := range styles {
		for _, rawRule := range strings.Split(style, "}") {
			rule := strings.TrimSpace(rawRule)
			if rule == "" {
				continue
			}
			rule += "}"

			lower := strings.ToLower(rule)
			relevant := strings.Contains(lower, "display:") ||
				strings.Contains(lower, "display :") ||
				strings.Contains(lower, "visibility:") ||
				strings.Contains(lower, "visibility :") ||
				strings.Contains(lower, "opacity:") ||
				strings.Contains(lower, "opacity :") ||
				strings.Contains(lower, ".hidden") ||
				strings.Contains(lower, ".visible") ||
				strings.Contains(lower, ".active") ||
				strings.Contains(lower, "answer") ||
				strings.Contains(lower, "solution") ||
				strings.Contains(lower, "feedback")

			if !relevant {
				for _, target := range domTargets {
					if target != "" &&
						strings.Contains(rule, target) {
						relevant = true
						break
					}
				}
			}

			if !relevant {
				continue
			}

			rule = cwAIReviewTruncate(rule, 520)
			if seen[rule] {
				continue
			}

			seen[rule] = true
			out = append(out, rule)

			if len(out) >= cwAIReviewMaxCSSRules {
				return out
			}
		}
	}

	return out
}

// cwAIReviewDetectInteractionRisks 识别答案暴露、契约缺失和静态解析盲区。
func cwAIReviewDetectInteractionRisks(
	page *models.CoursewarePage,
	htmlContent string,
	scriptText string,
	events []models.CWAIReviewInteractionEvent,
	reachable []models.CWAIReviewReachableFunction,
	contract cwInteractionValidationResult,
) (
	[]string,
	[]string,
	bool,
) {
	initialSignals := make([]string, 0)
	riskFlags := make([]string, 0)
	manualReview := false

	hasAnswerElement :=
		cwAIReviewAnswerElementRe.MatchString(htmlContent)

	hasHiddenAnswer :=
		cwAIReviewHiddenAnswerRe.MatchString(
			htmlContent,
		)

	hasVisibleAnswerText :=
		cwAIReviewVisibleAnswerTextRe.MatchString(
			cwAIReviewExtractVisibleText(htmlContent),
		)

	hasEmbeddedAnswer :=
		cwAIReviewEmbeddedAnswerRe.MatchString(
			htmlContent,
		)

	hasRevealFlow :=
		cwAIReviewRevealRe.MatchString(
			scriptText,
		)

	if hasAnswerElement {
		initialSignals = append(
			initialSignals,
			"页面包含答案、解析、结果或反馈节点",
		)
	}

	if hasHiddenAnswer {
		initialSignals = append(
			initialSignals,
			"检测到答案或反馈节点的初始隐藏规则",
		)
	}

	if hasVisibleAnswerText {
		initialSignals = append(
			initialSignals,
			"静态可见文本中出现“答案/正确答案/解析”字样",
		)
	}

	if hasEmbeddedAnswer {
		initialSignals = append(
			initialSignals,
			"HTML或脚本中存在可识别的正确答案字段",
		)
	}

	if hasRevealFlow {
		initialSignals = append(
			initialSignals,
			"脚本中存在显示或揭晓答案的操作链",
		)
	}

	if !contract.OK {
		riskFlags = append(
			riskFlags,
			"页面未满足方案声明的最低互动契约："+strings.TrimSpace(contract.Reason),
		)
	}

	if hasAnswerElement &&
		!hasHiddenAnswer &&
		hasVisibleAnswerText {
		riskFlags = append(
			riskFlags,
			"答案或解析可能在页面初始状态直接暴露",
		)
	}

	if hasEmbeddedAnswer &&
		!hasRevealFlow &&
		len(events) == 0 {
		riskFlags = append(
			riskFlags,
			"页面保存了正确答案，但未识别到提交、检查或延迟揭晓操作",
		)
	}

	kind := normalizeCWInteractionType(
		page.InteractionType,
	)

	if (kind == "quiz" ||
		kind == "game" ||
		kind == "input") &&
		len(events) > 0 &&
		!cwAIReviewRetryRe.MatchString(scriptText) {
		riskFlags = append(
			riskFlags,
			"未识别到重试、重置或重新开始机制",
		)
	}

	if cwAIReviewDOMContentLoadedRe.MatchString(scriptText) &&
		hasAnswerElement {
		riskFlags = append(
			riskFlags,
			"页面加载阶段会执行脚本，需确认初始化过程是否提前改变答案状态",
		)
		manualReview = true
	}

	if cwAIReviewDynamicRe.MatchString(scriptText) {
		riskFlags = append(
			riskFlags,
			"存在动态执行代码，静态调用链无法完整证明实际行为",
		)
		manualReview = true
	}

	if cwAIReviewExternalRuntimeRe.MatchString(htmlContent) {
		riskFlags = append(
			riskFlags,
			"存在外部脚本、网络请求或实时连接，需在浏览器中操作复核",
		)
		manualReview = true
	}

	if len([]rune(scriptText)) > 18000 &&
		len(reachable) == 0 {
		riskFlags = append(
			riskFlags,
			"脚本体量较大且未解析出稳定事件调用链",
		)
		manualReview = true
	}

	if kind != "" &&
		kind != "static" &&
		len(events) == 0 {
		riskFlags = append(
			riskFlags,
			"页面声明为互动页，但静态分析未找到明确操作入口",
		)
		manualReview = true
	}

	initialSignals = cwAIReviewDedupeStrings(
		initialSignals,
	)
	riskFlags = cwAIReviewDedupeStrings(
		riskFlags,
	)

	return initialSignals, riskFlags, manualReview
}

// cwAIReviewDedupeEvents 去重事件证据。
func cwAIReviewDedupeEvents(
	input []models.CWAIReviewInteractionEvent,
) []models.CWAIReviewInteractionEvent {
	seen := make(map[string]bool)
	out := make(
		[]models.CWAIReviewInteractionEvent,
		0,
		len(input),
	)

	for _, item := range input {
		key := item.EventType +
			"\x00" +
			item.Trigger +
			"\x00" +
			item.Handler

		if seen[key] {
			continue
		}

		seen[key] = true
		out = append(out, item)
	}

	return out
}

// cwAIReviewDedupeStrings 去重并保留原顺序。
func cwAIReviewDedupeStrings(
	input []string,
) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(input))

	for _, value := range input {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}

		seen[value] = true
		out = append(out, value)
	}

	return out
}

// cwAIReviewSHA256 生成页面快照哈希。
func cwAIReviewSHA256(
	content string,
) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// cwAIReviewTruncate 按 rune 截断，避免中文字符被切坏。
func cwAIReviewTruncate(
	content string,
	maxRunes int,
) string {
	content = strings.TrimSpace(content)
	if content == "" || maxRunes <= 0 {
		return ""
	}

	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}

	return string(runes[:maxRunes])
}
