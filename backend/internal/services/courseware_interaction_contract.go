package services

// courseware_interaction_contract.go —— 课件方案互动方式的执行契约与生成后验收
//
// 背景：
//   课件方案中的 interaction_type 过去只作为一行普通提示文字传给模型。
//   组件匹配也只读取 idx_interaction_level，索引缺失时甚至把
//   estimated_complexity（内容丰富度）误当成交互等级。
//   因此老师选择 input，最终页面仍可能生成 click；选择 click，页面也可能只有
//   “点击卡片”提示文字而没有任何真实事件。
//
// 本文件建立四层闭环：
//   1. 将八种 interaction_type 翻译成明确的人话、DOM、事件和反馈契约；
//   2. 将方案互动类型确定性映射为组件匹配所需的交互等级；
//   3. 过滤与方案互动类型明显冲突的参考组件，避免错误组件压过方案；
//   4. 对生成后的 HTML 做纯静态验收，不合格时把明确原因交给模型自动重试。
//
// 设计边界：
//   - 本模块只做纯字符串和正则判断，不写数据库、不调用AI；
//   - 验收检查“是否具有完成该互动所需的最低代码结构”，不试图证明教学效果；
//   - static 页面不强制交互；
//   - video 当前平台完整自动链仍只生成首帧和分镜，因此本模块只要求页面至少具有
//     可识别的 <video> 播放器结构或结构化视频占位，真实视频生成另行治理。

import (
	"regexp"
	"strings"

	"tedna/internal/models"
)

// cwInteractionValidationResult 是生成后互动契约验收结果。
//
// OK=true：具备该互动类型最低可运行结构。
// OK=false：页面虽然可能是合法HTML，但没有落实方案要求的互动方式。
type cwInteractionValidationResult struct {
	OK     bool
	Reason string
	Detail string
}

// 八种方案互动类型的人话名称。
var cwInteractionTypeNames = map[string]string{
	"static":    "静态展示",
	"click":     "点击交互",
	"drag":      "拖拽操作",
	"input":     "输入填写",
	"animation": "动画演示",
	"video":     "视频播放",
	"game":      "游戏互动",
	"quiz":      "答题测验",
}

// 生成后验收与组件识别使用的正则。
var (
	cwContractAnyInlineEventRe = regexp.MustCompile(
		`(?i)\bon(?:click|input|change|submit|dragstart|dragover|drop|pointerdown|pointermove|pointerup|mousedown|mousemove|mouseup|touchstart|touchmove|touchend|blur|keydown|keyup)\s*=`,
	)
	cwContractAnyListenerRe = regexp.MustCompile(
		`(?i)addEventListener\s*\(\s*["'](?:click|input|change|submit|dragstart|dragover|drop|pointerdown|pointermove|pointerup|mousedown|mousemove|mouseup|touchstart|touchmove|touchend|blur|keydown|keyup)["']`,
	)
	cwContractClickEventRe = regexp.MustCompile(
		`(?i)\bonclick\s*=|addEventListener\s*\(\s*["']click["']`,
	)
	cwContractInputLikeRe = regexp.MustCompile(
		`(?is)<(?:input|textarea|select)\b|contenteditable\s*=\s*["']?(?:true|plaintext-only)`,
	)
	cwContractInputEventRe = regexp.MustCompile(
		`(?i)\bon(?:input|change|submit|blur|keyup|keydown|click)\s*=|addEventListener\s*\(\s*["'](?:input|change|submit|blur|keyup|keydown|click)["']|<form\b`,
	)
	cwContractDragEventRe = regexp.MustCompile(
		`(?i)\bdraggable\s*=\s*["']?true|\bon(?:dragstart|dragover|drop|pointerdown|pointermove|pointerup|mousedown|mousemove|mouseup|touchstart|touchmove|touchend)\s*=|addEventListener\s*\(\s*["'](?:dragstart|dragover|drop|pointerdown|pointermove|pointerup|mousedown|mousemove|mouseup|touchstart|touchmove|touchend)["']`,
	)
	cwContractAnimationRe = regexp.MustCompile(
		`(?i)@keyframes\b|\banimation\s*:|requestAnimationFrame\s*\(|setInterval\s*\(|setTimeout\s*\(`,
	)
	cwContractVideoTagRe         = regexp.MustCompile(`(?i)<video\b`)
	cwContractVideoPlaceholderRe = regexp.MustCompile(
		`(?i)video-placeholder|data-video-placeholder\s*=|data-media-type\s*=\s*["']video["']`,
	)
	cwContractChoiceRe = regexp.MustCompile(
		`(?is)<input\b[^>]*\btype\s*=\s*["']?(?:radio|checkbox)|\bdata-option\s*=|class\s*=\s*["'][^"']*(?:option|choice|answer)[^"']*["']`,
	)
)

// normalizeCWInteractionType 统一互动类型字符串。
func normalizeCWInteractionType(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

// cwInteractionTypeLabel 返回人话名称，未知值原样返回。
func cwInteractionTypeLabel(kind string) string {
	normalized := normalizeCWInteractionType(kind)
	if label, ok := cwInteractionTypeNames[normalized]; ok {
		return label
	}
	if normalized == "" {
		return "静态展示"
	}
	return normalized
}

// cwInteractionClamp 把互动等级限制到1-5。
func cwInteractionClamp(value int) int {
	if value < 1 {
		return 1
	}
	if value > 5 {
		return 5
	}
	return value
}

// cwInteractionLevelForPlan 将老师选择的互动类型映射为组件库的交互等级。
//
// 方案字段是老师最后确认的事实源，因此已知 interaction_type 优先于旧索引字段。
// 只有互动类型为空或未知时，才回退 idx_interaction_level，最后再用复杂度兜底。
func cwInteractionLevelForPlan(kind string, indexLevel int, complexity int) int {
	switch normalizeCWInteractionType(kind) {
	case "static":
		return 1
	case "click", "video":
		return 2
	case "input", "animation", "quiz":
		return 3
	case "drag":
		return 4
	case "game":
		return 5
	}

	if indexLevel > 0 {
		return cwInteractionClamp(indexLevel)
	}
	if complexity > 0 {
		return cwInteractionClamp(complexity)
	}
	return 1
}

// cwVisualFormatForMatch 返回组件匹配应使用的视觉形式。
//
// 老师确认或修改后的 page.VisualFormat 优先；只有它为空时才回退层1索引字段。
func cwVisualFormatForMatch(page *models.CoursewarePage) string {
	if page == nil {
		return ""
	}
	if value := strings.TrimSpace(page.VisualFormat); value != "" {
		return value
	}
	return strings.TrimSpace(page.IdxVisualFormat)
}

// cwInteractionContractText 把方案互动方式翻译为明确的HTML执行契约。
func cwInteractionContractText(page *models.CoursewarePage) string {
	if page == nil {
		return ""
	}

	kind := normalizeCWInteractionType(page.InteractionType)
	if kind == "" {
		kind = "static"
	}

	var sb strings.Builder
	sb.WriteString("## 互动方式执行契约（最高优先级，必须真实实现）\n")
	sb.WriteString("老师已经在方案中选择「")
	sb.WriteString(cwInteractionTypeLabel(kind))
	sb.WriteString("（")
	sb.WriteString(kind)
	sb.WriteString("）」。这不是风格建议，而是本页必须执行的功能契约。\n")

	switch kind {
	case "static":
		sb.WriteString("- 本页允许静态展示，不强制加入交互。\n")
		sb.WriteString("- 不得写“点击、拖拽、输入、播放”等操作提示却不提供真实可操作功能。\n")

	case "click":
		sb.WriteString("- 至少提供一个清晰可见、可点击的按钮、卡片、热点或步骤控件。\n")
		sb.WriteString("- 必须通过 onclick 或 addEventListener('click', ...) 绑定真实点击事件。\n")
		sb.WriteString("- 点击后必须发生可见状态变化，例如展开详情、切换内容、推进步骤、高亮或反馈。\n")
		sb.WriteString("- 禁止只显示“点击卡片”“点击查看”等文字而没有任何事件代码。\n")

	case "drag":
		sb.WriteString("- 必须存在可拖动物体和明确的目标区域。\n")
		sb.WriteString("- 必须实现 draggable/dragstart/dragover/drop，或完整的 pointer/mouse/touch 拖拽事件链。\n")
		sb.WriteString("- 拖放成功或失败后必须给出可见反馈，并提供重置或再次操作能力。\n")
		sb.WriteString("- 禁止退化为单纯点击按钮或静态位置示意。\n")

	case "input":
		sb.WriteString("- 必须存在 input、textarea、select 或 contenteditable 等真实输入区域。\n")
		sb.WriteString("- 必须提供提交、检查、确认或即时校验机制，并绑定 input/change/submit/click 等事件。\n")
		sb.WriteString("- 输入后必须出现可见反馈、结果、提示或状态变化。\n")
		sb.WriteString("- 禁止把“输入填写”替换成“上一步/下一步”或仅点击卡片的步骤演示。\n")

	case "animation":
		sb.WriteString("- 必须存在与教学内容直接相关的真实动画过程，不得只有装饰性渐变或静态示意图。\n")
		sb.WriteString("- 使用 @keyframes、animation、requestAnimationFrame 或定时器实现动画。\n")
		sb.WriteString("- 优先提供开始、暂停、重播或逐步演示控制，让课堂中可以反复观察。\n")

	case "video":
		sb.WriteString("- 页面必须使用 <video controls> 播放器结构，或使用带 data-video-placeholder 的结构化视频占位容器。\n")
		sb.WriteString("- 禁止只写“点击播放视频”文字或用普通图片框冒充视频播放器。\n")
		sb.WriteString("- 暂无真实视频地址时，占位容器也必须明确标记为视频，并保留poster、播放控制和后续替换入口。\n")

	case "game":
		sb.WriteString("- 必须存在学生可操作的游戏规则、动作控件和状态变化。\n")
		sb.WriteString("- 必须绑定真实事件，并至少维护得分、进度、关卡、次数、生命或计时中的一种状态。\n")
		sb.WriteString("- 必须提供成功/失败反馈以及重新开始或重置能力。\n")

	case "quiz":
		sb.WriteString("- 必须存在可选择或可填写的题目作答区，不能只展示题目与答案。\n")
		sb.WriteString("- 必须提供提交或检查按钮，并绑定真实事件。\n")
		sb.WriteString("- 作答后必须显示正确/错误、解析、提示或得分反馈，并允许重试或进入下一题。\n")

	default:
		sb.WriteString("- 请严格按照 interaction_type 的真实含义实现可操作功能，不得只做文字说明。\n")
	}

	sb.WriteString("- 系统会在生成后检查DOM、控件和事件结构；不符合契约的结果不会直接作为成功页面保存。\n\n")
	return sb.String()
}

// appendInteractionContract 把互动执行契约追加到生成提示词。
func (s *CoursewareGenService) appendInteractionContract(
	sb *strings.Builder,
	page *models.CoursewarePage,
) {
	if sb == nil || page == nil {
		return
	}
	sb.WriteString(cwInteractionContractText(page))
}

// cwContainsAny 判断文本是否包含任一标记。
func cwContainsAny(text string, markers ...string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// cwComponentSupportsInteraction 判断参考组件是否与方案互动类型基本相符。
//
// 这是组件参考过滤器，不要求组件自身是完整课件页，因此比生成后验收略宽松。
func cwComponentSupportsInteraction(
	component *models.MatchedCWComponent,
	kind string,
) bool {
	if component == nil {
		return false
	}

	normalized := normalizeCWInteractionType(kind)
	if normalized == "" || normalized == "static" {
		return true
	}

	code := strings.TrimSpace(component.CodeContent)
	if code == "" {
		return false
	}
	lower := strings.ToLower(code)

	switch normalized {
	case "click":
		return cwContractClickEventRe.MatchString(code)
	case "drag":
		return cwContractDragEventRe.MatchString(code)
	case "input":
		return cwContractInputLikeRe.MatchString(code)
	case "animation":
		return cwContractAnimationRe.MatchString(code)
	case "video":
		return cwContractVideoTagRe.MatchString(code) ||
			cwContractVideoPlaceholderRe.MatchString(code)
	case "game":
		hasEvent := cwContractAnyInlineEventRe.MatchString(code) ||
			cwContractAnyListenerRe.MatchString(code)
		hasState := cwContainsAny(
			lower,
			"score", "得分", "计分", "level", "关卡",
			"progress", "进度", "lives", "生命", "timer", "倒计时",
		)
		return hasEvent && hasState
	case "quiz":
		hasEvent := cwContractAnyInlineEventRe.MatchString(code) ||
			cwContractAnyListenerRe.MatchString(code)
		return cwContractChoiceRe.MatchString(code) &&
			hasEvent
	default:
		return true
	}
}

// filterCWComponentsForInteraction 去掉与老师选择的互动类型明显冲突的参考组件。
//
// 宁可不注入组件，也不把“逐步点击组件”注入 input 页面后诱导模型改成交互类型。
func filterCWComponentsForInteraction(
	components []*models.MatchedCWComponent,
	kind string,
) []*models.MatchedCWComponent {
	if len(components) == 0 {
		return nil
	}

	normalized := normalizeCWInteractionType(kind)
	if normalized == "" || normalized == "static" {
		return components
	}

	filtered := make([]*models.MatchedCWComponent, 0, len(components))
	for _, component := range components {
		if cwComponentSupportsInteraction(component, normalized) {
			filtered = append(filtered, component)
		}
	}
	return filtered
}

// validateGeneratedPageInteraction 检查生成后的HTML是否落实方案互动方式。
func validateGeneratedPageInteraction(
	kind string,
	html string,
) cwInteractionValidationResult {
	normalized := normalizeCWInteractionType(kind)
	if normalized == "" {
		normalized = "static"
	}

	content := strings.TrimSpace(html)
	if content == "" {
		return cwInteractionValidationResult{
			OK:     false,
			Reason: "页面HTML为空",
			Detail: "empty html",
		}
	}

	lower := strings.ToLower(content)
	hasAnyEvent := cwContractAnyInlineEventRe.MatchString(content) ||
		cwContractAnyListenerRe.MatchString(content)

	switch normalized {
	case "static":
		return cwInteractionValidationResult{OK: true}

	case "click":
		if !cwContractClickEventRe.MatchString(content) {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "方案要求点击交互，但页面没有 onclick 或 click 事件监听",
				Detail: "missing click event",
			}
		}
		return cwInteractionValidationResult{OK: true}

	case "drag":
		if !cwContractDragEventRe.MatchString(content) {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "方案要求拖拽操作，但页面没有 draggable、drag/drop 或 pointer/mouse/touch 拖拽事件",
				Detail: "missing drag event chain",
			}
		}
		return cwInteractionValidationResult{OK: true}

	case "input":
		if !cwContractInputLikeRe.MatchString(content) {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "方案要求输入填写，但页面没有 input、textarea、select 或 contenteditable 输入区域",
				Detail: "missing input control",
			}
		}
		if !cwContractInputEventRe.MatchString(content) {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "页面有输入框，但没有提交、检查、输入变化或反馈事件",
				Detail: "missing input validation event",
			}
		}
		if !cwContainsAny(
			lower,
			"feedback", "result", "message", "status",
			"正确", "错误", "反馈", "结果", "提示", "校验",
		) {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "页面有输入控件和事件，但没有可识别的结果或反馈区域",
				Detail: "missing input feedback",
			}
		}
		return cwInteractionValidationResult{OK: true}

	case "animation":
		if !cwContractAnimationRe.MatchString(content) {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "方案要求动画演示，但页面没有 @keyframes、animation、requestAnimationFrame 或定时动画代码",
				Detail: "missing animation implementation",
			}
		}
		return cwInteractionValidationResult{OK: true}

	case "video":
		if cwContractVideoTagRe.MatchString(content) {
			if !strings.Contains(lower, "controls") {
				return cwInteractionValidationResult{
					OK:     false,
					Reason: "页面有 video 标签，但没有 controls 播放控制",
					Detail: "video without controls",
				}
			}
			return cwInteractionValidationResult{OK: true}
		}
		if cwContractVideoPlaceholderRe.MatchString(content) &&
			cwContractClickEventRe.MatchString(content) {
			return cwInteractionValidationResult{OK: true}
		}
		return cwInteractionValidationResult{
			OK:     false,
			Reason: "方案要求视频播放，但页面既没有 <video controls>，也没有可操作的结构化视频占位",
			Detail: "missing video player",
		}

	case "game":
		if !hasAnyEvent {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "方案要求游戏互动，但页面没有真实操作事件",
				Detail: "game missing event",
			}
		}
		if !cwContainsAny(
			lower,
			"score", "得分", "计分", "level", "关卡",
			"progress", "进度", "lives", "生命", "timer", "倒计时",
		) {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "页面有操作事件，但没有得分、进度、关卡、次数或计时等游戏状态",
				Detail: "game missing state",
			}
		}
		return cwInteractionValidationResult{OK: true}

	case "quiz":
		if !cwContractChoiceRe.MatchString(content) &&
			!cwContractInputLikeRe.MatchString(content) {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "方案要求答题测验，但页面没有选项或输入作答区",
				Detail: "quiz missing answer controls",
			}
		}
		if !hasAnyEvent {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "答题页面没有提交、检查或选项事件",
				Detail: "quiz missing event",
			}
		}
		if !cwContainsAny(
			lower,
			"feedback", "result", "score", "answer",
			"正确", "错误", "解析", "得分", "答案", "反馈",
		) {
			return cwInteractionValidationResult{
				OK:     false,
				Reason: "答题页面没有可识别的正确错误、解析或得分反馈",
				Detail: "quiz missing feedback",
			}
		}
		return cwInteractionValidationResult{OK: true}

	default:
		return cwInteractionValidationResult{OK: true}
	}
}

// buildCWInteractionRepairPrompt 在上一轮互动验收失败后构造纠偏重试提示词。
//
// 始终以原始完整 userPrompt 为基础，不把上轮HTML重新注入，避免提示词越来越长。
// 只追加确定性的失败原因和完整互动契约，让模型从零重新生成本页。
func buildCWInteractionRepairPrompt(
	basePrompt string,
	page *models.CoursewarePage,
	result cwInteractionValidationResult,
) string {
	var sb strings.Builder
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n")
	sb.WriteString("## 自动验收未通过，必须完整重新生成本页\n")
	sb.WriteString("上一轮HTML虽然可能能够显示，但没有落实老师确认的互动方式。\n")
	sb.WriteString("未通过原因：")
	sb.WriteString(strings.TrimSpace(result.Reason))
	sb.WriteString("\n")
	if strings.TrimSpace(result.Detail) != "" {
		sb.WriteString("技术检查：")
		sb.WriteString(strings.TrimSpace(result.Detail))
		sb.WriteString("\n")
	}
	sb.WriteString("请不要解释、不要修补上一轮代码，直接重新输出完整HTML，并严格满足以下契约：\n\n")
	sb.WriteString(cwInteractionContractText(page))
	sb.WriteString("重新生成时仍须遵守原提示词中的教学内容、风格、背景、导航栏和1920×1080画布要求。\n")
	return sb.String()
}
