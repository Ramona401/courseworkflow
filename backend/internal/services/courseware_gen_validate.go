package services

// courseware_gen_validate.go — 课件页面 HTML 输出完整性校验闸门
//
// 设计背景（根因）：
//   单页微调(RefinePage)与单页重生(RegenerateSinglePage)采用"全量重写整页"策略——
//   把整页 HTML 喂给 AI、要求 AI 返回完整的一整页。当输出超过模型 max_tokens 上限时，
//   服务端会在中途【静默截断】返回，AI 输出停在某个标签中间。残缺 HTML（卡片/交互整段丢失、
//   根容器未闭合）若被静默写库并渲染，即表现为"微调后交互变少、内容删减"。
//
// 本文件职责（输出端兜底）：
//   在"已存旧版快照之后、即将 UpdateCWPageHTML 覆盖之前"插入校验闸门，对 AI 产出的 HTML 做：
//     1) 结构闭合校验——<div>/<script>/<style> 开闭配对，且末尾不在标签中途断开；
//     2) 关键资产比对——改前扫描原页 id=/function/onclick/addEventListener 清单，改后比对，
//        老师没要求删却整类消失的资产判为高危（典型即被截断丢失的交互脚本/卡片）；
//     3) 体量骤降预警——输出长度不到原页一定比例且老师意图非"精简/删除"时预警。
//
// 【轻微漏闭合自动补全】（关键设计取舍）：
//   实践中区分两类"div 开闭不相等"：
//     A. 真截断——输出被服务端腰斩，会同时丢失大量 </div>（差值大，常伴 script/style 也不配平、
//        或尾部断在标签中途、或体量腰斩、或交互资产整类归零）。这类必须拦截、保留原版。
//     B. AI 手滑漏闭合——AI 把整页几乎写完，只是末尾漏写 1~2 个 </div>（差值极小，script/style
//        配平，尾部完整收在 '>'）。这类不是截断，整页内容都在，本可自动补上缺的 </div> 后放行，
//        不该整页毙掉让老师反复重试（重试同样易漏，会让老师陷入"每次都失败"）。
//   故 div 不配对时先用 cwTryAutoCloseDivs 判断是否属 B 类：满足"仅缺 1~2 个 </div> 且 script/style
//   配平且尾部不在标签中途"则在末尾补足 </div> 并放行（结果经 FixedHTML 返回供调用方写库），
//   否则判 A 类截断拦截。补全后仍继续走资产/体量校验（补的是闭合标签，不改变资产清单）。
//
// 设计原则：
//   - 纯计算无副作用：不写库、不调 AI；唯一"修改"是 cwTryAutoCloseDivs 在末尾追加 </div> 字符串。
//   - 宁可放过不可错杀：仅在明确截断信号（大量丢失/尾部断裂/资产归零/体量腰斩）才拦截。

import (
	"regexp"
	"strings"
)

// ==================== 校验结果类型 ====================

// cwRefineValidateResult 输出完整性校验结果
//
//	OK=true     → 通过，可安全写库
//	OK=false    → 疑似截断/残缺，调用方应保留原版并报错
//	Reason      → 给用户看的人话原因（OK=false 时非空）
//	Detail      → 给日志看的技术细节（开闭差值/丢失资产/自动补全等）
//	FixedHTML   → 非空表示校验过程对 newHTML 做了轻微漏闭合自动补全，
//	              调用方写库时应使用 FixedHTML 而非原 newHTML；为空表示未修改沿用原值。
type cwRefineValidateResult struct {
	OK        bool
	Reason    string
	Detail    string
	FixedHTML string
}

// cwAutoCloseMaxFill 轻微漏闭合自动补全允许补的最大 </div> 数。
// 超过此数视为真截断（丢失太多不是手滑），不补、判拦截。
const cwAutoCloseMaxFill = 2

// ==================== 关键资产清单 ====================

// cwAssetInventory 一页 HTML 的关键资产清单（用于改前改后比对）
type cwAssetInventory struct {
	IDs            map[string]struct{}
	Functions      map[string]struct{}
	OnClicks       int
	EventListeners int
	ScriptBlocks   int
}

// 资产扫描用正则（包级编译一次，复用）
var (
	cwAssetIDRe       = regexp.MustCompile(`(?i)\bid\s*=\s*["']([a-zA-Z][\w\-]*)["']`)
	cwAssetFuncRe     = regexp.MustCompile(`(?i)\bfunction\s+([a-zA-Z_$][\w$]*)\s*\(`)
	cwAssetOnClickRe  = regexp.MustCompile(`(?i)\bon[a-z]+\s*=\s*["']`)   // onclick/onmouseenter 等内联事件
	cwAssetListenerRe = regexp.MustCompile(`(?i)\.addEventListener\s*\(`) // 事件监听
	cwScriptOpenRe    = regexp.MustCompile(`(?i)<script\b`)
	cwScriptCloseRe   = regexp.MustCompile(`(?i)</script\s*>`)
	cwStyleOpenRe     = regexp.MustCompile(`(?i)<style\b`)
	cwStyleCloseRe    = regexp.MustCompile(`(?i)</style\s*>`)
)

// buildCWAssetInventory 扫描一页 HTML，提取关键资产清单
func buildCWAssetInventory(html string) cwAssetInventory {
	inv := cwAssetInventory{
		IDs:       make(map[string]struct{}),
		Functions: make(map[string]struct{}),
	}
	for _, m := range cwAssetIDRe.FindAllStringSubmatch(html, -1) {
		if len(m) >= 2 {
			inv.IDs[m[1]] = struct{}{}
		}
	}
	for _, m := range cwAssetFuncRe.FindAllStringSubmatch(html, -1) {
		if len(m) >= 2 {
			inv.Functions[m[1]] = struct{}{}
		}
	}
	inv.OnClicks = len(cwAssetOnClickRe.FindAllString(html, -1))
	inv.EventListeners = len(cwAssetListenerRe.FindAllString(html, -1))
	inv.ScriptBlocks = len(cwScriptOpenRe.FindAllString(html, -1))
	return inv
}

// ==================== 标签闭合校验 ====================

// cwCountTag 统计某标签的开/闭数量（大小写不敏感，开标签用 openRe，闭标签用 closeRe）
func cwCountTag(html string, openRe, closeRe *regexp.Regexp) (int, int) {
	return len(openRe.FindAllString(html, -1)), len(closeRe.FindAllString(html, -1))
}

var (
	cwDivOpenRe  = regexp.MustCompile(`(?i)<div\b`)
	cwDivCloseRe = regexp.MustCompile(`(?i)</div\s*>`)
)

// cwCountDivTags 统计真实HTML结构中的div开闭标签。
//
// script/style正文按HTML原始文本元素处理，其中的innerHTML字符串、模板字符串
// 和CSS content伪标签均不参与计数。
func cwCountDivTags(html string) (int, int) {
	structure := cwScanHTMLStructure(html)
	return structure.DivOpen, structure.DivClose
}

// cwEndsMidTag 判断HTML是否在真实标签中途结束。
// JavaScript比较符、字符串里的“<”以及script/style正文不会造成误报。
func cwEndsMidTag(html string) bool {
	if strings.TrimSpace(html) == "" {
		return true
	}
	return cwScanHTMLStructure(html).EndsMidTag
}

// cwTryAutoCloseDivs 尝试对"仅缺少量 </div>"的 HTML 自动补全。
//
// 仅在【明确属于 AI 手滑漏闭合】时才补，判定条件全部满足：
//   - divOpen > divClose（确实缺闭合）且缺口 missing := divOpen-divClose 在 [1, cwAutoCloseMaxFill]；
//   - script/style 均配平（截断常导致脚本/样式也断，配平说明主体写完了）；
//   - 尾部不在标签中途（!cwEndsMidTag，说明最后一个标签是完整收尾的）。
//
// 满足则在 HTML 末尾追加 missing 个 </div> 返回 (fixed, true)；
// 任一不满足返回 ("", false)，交由调用方判截断拦截。
//
// 注：追加在末尾是安全的——根容器是最外层 div，漏的闭合一定在尾部；补 </div> 不影响已有资产清单。
func cwTryAutoCloseDivs(html string, divOpen, divClose, scriptOpen, scriptClose, styleOpen, styleClose int) (string, bool) {
	if divOpen <= divClose {
		return "", false // 不缺闭合（相等或闭比开还多，后者属异常不在本函数处理）
	}
	missing := divOpen - divClose
	if missing > cwAutoCloseMaxFill {
		return "", false // 缺太多，视为真截断
	}
	if scriptOpen != scriptClose || styleOpen != styleClose {
		return "", false // 脚本/样式也不配平 → 截断特征，不补
	}
	if cwEndsMidTag(html) {
		return "", false // 尾部断在标签中途 → 截断，不补
	}
	// 通过：在末尾补足缺的 </div>
	var sb strings.Builder
	sb.WriteString(strings.TrimRight(html, " \t\r\n"))
	for i := 0; i < missing; i++ {
		sb.WriteString("\n</div>")
	}
	return sb.String(), true
}

var cwTrailingDocumentCloseRe = regexp.MustCompile(
	`(?is)^\s*(?:<!--.*?-->\s*)*(?:</body\s*>\s*)?(?:</html\s*>\s*)?$`,
)

// cwTryRemoveSingleTrailingExtraDiv 安全修复AI结果末尾多出的一个</div>。
//
// 只在以下条件全部满足时处理：
//   - 正则计数恰好多一个</div>；
//   - script/style均配平且HTML未断在标签中途；
//   - 按真实标签扫描后仅有一个未匹配的</div>；
//   - 该未匹配标签之后只剩空白、注释或</body></html>。
//
// script/style正文里的“<div>”字符串会被忽略，避免修改JavaScript模板字符串。
func cwTryRemoveSingleTrailingExtraDiv(
	html string,
	divOpen int,
	divClose int,
	scriptOpen int,
	scriptClose int,
	styleOpen int,
	styleClose int,
) (string, bool) {
	if divClose-divOpen != 1 ||
		scriptOpen != scriptClose ||
		styleOpen != styleClose ||
		cwEndsMidTag(html) {
		return "", false
	}

	lower := strings.ToLower(html)
	depth := 0
	rawTag := ""
	unmatchedStart := -1
	unmatchedEnd := -1

	for cursor := 0; cursor < len(html); {
		relative := strings.Index(html[cursor:], "<")
		if relative < 0 {
			break
		}

		start := cursor + relative

		if strings.HasPrefix(lower[start:], "<!--") {
			commentEnd := strings.Index(lower[start+4:], "-->")
			if commentEnd < 0 {
				return "", false
			}

			cursor = start + 4 + commentEnd + 3
			continue
		}

		end := findAutoAssemblyTagEnd(html, start)
		if end < 0 {
			return "", false
		}

		name, closing, selfClosing :=
			parseAutoAssemblyTagToken(html[start+1 : end])

		if rawTag != "" {
			if closing && name == rawTag {
				rawTag = ""
			}

			cursor = end + 1
			continue
		}

		if name == "script" || name == "style" {
			if !closing && !selfClosing {
				rawTag = name
			}

			cursor = end + 1
			continue
		}

		if name == "div" {
			if closing {
				if depth == 0 {
					if unmatchedStart >= 0 {
						return "", false
					}

					unmatchedStart = start
					unmatchedEnd = end + 1
				} else {
					depth--
				}
			} else if !selfClosing {
				depth++
			}
		}

		cursor = end + 1
	}

	if rawTag != "" ||
		depth != 0 ||
		unmatchedStart < 0 ||
		unmatchedEnd <= unmatchedStart ||
		!cwTrailingDocumentCloseRe.MatchString(
			html[unmatchedEnd:],
		) {
		return "", false
	}

	fixed :=
		strings.TrimSpace(
			html[:unmatchedStart] +
				html[unmatchedEnd:],
		)

	return fixed,
		true
}

// ==================== 主校验入口 ====================

// validateRefinedPageHTML 校验"微调/重生后的 HTML"相对"原 HTML"是否完整可用。
//
// 参数：
//
//	oldHTML       —— 覆盖前的原页 HTML（page.HTMLContent 旧值，可能为空=首次生成）
//	newHTML       —— AI 产出并经 extractHTMLFromAIOutput/normalizeRootCanvas 处理后的待写库 HTML
//	instruction   —— 老师的修改意见（用于识别"精简/删除"类合法缩量意图）
//	isRegenerate  —— true=单页重生（从零重画，跳过资产/体量比对，只做结构闭合）；false=微调（三类全做）
//
// 返回 cwRefineValidateResult：
//   - OK=false 时调用方保留原版、把 Reason 返回前端；
//   - OK=true 且 FixedHTML 非空时调用方写库须用 FixedHTML（已对轻微漏闭合自动补全）。
func validateRefinedPageHTML(oldHTML, newHTML, instruction string, isRegenerate bool) cwRefineValidateResult {
	nt := strings.TrimSpace(newHTML)

	// ---- 防御：空输出直接判失败 ----
	if nt == "" {
		return cwRefineValidateResult{OK: false, Reason: "AI 未返回有效页面内容，已保留原版，请重试。", Detail: "newHTML empty"}
	}

	// ---- 校验1：结构闭合 ----
	//
	// 使用与AI输出提取器相同的词法扫描结果。script/style正文中的伪标签
	// 不再参与div计数，避免“看似配平、实际被脚本字符串补平”的误判。
	structure := cwScanHTMLStructure(nt)
	divOpen, divClose := structure.DivOpen, structure.DivClose
	scriptOpen, scriptClose := structure.ScriptOpen, structure.ScriptClose
	styleOpen, styleClose := structure.StyleOpen, structure.StyleClose

	// 脚本/样式块未闭合 → 截断硬信号，先判（这两类不做自动补全，缺一块脚本/样式往往真断在中间）。
	if scriptOpen != scriptClose {
		return cwRefineValidateResult{
			OK:     false,
			Reason: "页面处理结果中的脚本块未闭合（<script> 缺少配对的 </script>）。系统已保留原版；日志已记录AI原始输出与提取阶段结构，请重试。",
			Detail: cwFmtTagDetail("script", divOpen, divClose, scriptOpen, scriptClose, styleOpen, styleClose),
		}
	}
	if styleOpen != styleClose {
		return cwRefineValidateResult{
			OK:     false,
			Reason: "页面处理结果中的样式块未闭合（<style> 缺少配对的 </style>）。系统已保留原版；日志已记录AI原始输出与提取阶段结构，请重试。",
			Detail: cwFmtTagDetail("style", divOpen, divClose, scriptOpen, scriptClose, styleOpen, styleClose),
		}
	}

	// <div> 开闭不配对时分两类确定性小修：
	//   1. 缺少1~2个闭合标签 → 在末尾补齐；
	//   2. 仅多出一个且位于真实文档尾部 → 删除该多余闭合标签。
	// 其它情况继续拦截，绝不对页面中部结构做猜测性修复。
	if divOpen != divClose {
		if fixed, ok := cwTryAutoCloseDivs(
			nt,
			divOpen,
			divClose,
			scriptOpen,
			scriptClose,
			styleOpen,
			styleClose,
		); ok {
			missing := divOpen - divClose
			nt = strings.TrimSpace(fixed)
			autoFixDetail :=
				"auto_close_div fill=" +
					cwItoa(missing) +
					" " +
					cwFmtTagDetail(
						"div",
						divOpen,
						divClose,
						scriptOpen,
						scriptClose,
						styleOpen,
						styleClose,
					)

			if isRegenerate {
				return cwRefineValidateResult{
					OK:        true,
					FixedHTML: nt,
					Detail:    autoFixDetail,
				}
			}

			return cwValidateAssetsAndSize(
				oldHTML,
				nt,
				instruction,
				nt,
				autoFixDetail,
			)
		}

		if fixed, ok :=
			cwTryRemoveSingleTrailingExtraDiv(
				nt,
				divOpen,
				divClose,
				scriptOpen,
				scriptClose,
				styleOpen,
				styleClose,
			); ok {
			nt = strings.TrimSpace(fixed)
			autoFixDetail :=
				"auto_remove_extra_div count=1 " +
					cwFmtTagDetail(
						"div",
						divOpen,
						divClose,
						scriptOpen,
						scriptClose,
						styleOpen,
						styleClose,
					)

			if isRegenerate {
				return cwRefineValidateResult{
					OK:        true,
					FixedHTML: nt,
					Detail:    autoFixDetail,
				}
			}

			return cwValidateAssetsAndSize(
				oldHTML,
				nt,
				instruction,
				nt,
				autoFixDetail,
			)
		}

		reason :=
			"AI 输出的页面结构不完整（标签未闭合），疑似生成中途被截断。已保留原版，请重试。"

		if divClose > divOpen {
			reason =
				"AI 输出包含无法安全定位的多余闭合标签，页面结构存在冲突。已保留原版，请重试。"
		}

		return cwRefineValidateResult{
			OK:     false,
			Reason: reason,
			Detail: cwFmtTagDetail(
				"div",
				divOpen,
				divClose,
				scriptOpen,
				scriptClose,
				styleOpen,
				styleClose,
			),
		}
	}

	// 尾部"卡在标签中途"——div 配平但尾部断裂仍能抓到。
	if cwEndsMidTag(nt) {
		return cwRefineValidateResult{
			OK:     false,
			Reason: "AI 输出在标签中途结束，疑似被截断。已保留原版，请重试。",
			Detail: "html ends mid-tag",
		}
	}

	// 结构完整（未触发自动补全）：重生到此即可。
	if isRegenerate {
		return cwRefineValidateResult{OK: true}
	}

	// 微调：继续资产/体量校验（FixedHTML 留空，写库沿用原 newHTML）。
	return cwValidateAssetsAndSize(oldHTML, nt, instruction, "", "")
}

// cwValidateAssetsAndSize 微调路径专属：在结构闭合通过后，做关键资产比对 + 体量骤降校验。
//
// fixedHTML/fixedDetail：若上游因自动补全替换了 HTML，则把补全后的 HTML 与日志细节透传进来，
//
//	校验通过时随结果带回（FixedHTML），让调用方写库用补全版；未补全则两者为空。
//
// htmlForCheck：实际用于资产/体量比对的 HTML（已是补全后或原始的待写库 HTML）。
func cwValidateAssetsAndSize(oldHTML, htmlForCheck, instruction, fixedHTML, fixedDetail string) cwRefineValidateResult {
	// 原 HTML 为空（理论上微调不会，防御）则跳过比对，直接通过。
	if strings.TrimSpace(oldHTML) == "" {
		return cwRefineValidateResult{OK: true, FixedHTML: fixedHTML, Detail: fixedDetail}
	}

	oldInv := buildCWAssetInventory(oldHTML)
	newInv := buildCWAssetInventory(htmlForCheck)

	// ---- 校验2：关键资产比对 ----
	// 老师意图若明确是"删除/移除/精简"，资产减少属合法，放宽资产校验（结构闭合已在上游保证）。
	if !cwInstructionAllowsRemoval(instruction) {
		// 2a. 交互整类归零：原页有 <script>/函数/事件，改后全没了 → 几乎一定是截断丢失。
		oldHadInteractivity := oldInv.ScriptBlocks > 0 || len(oldInv.Functions) > 0 ||
			oldInv.OnClicks > 0 || oldInv.EventListeners > 0
		newHasInteractivity := newInv.ScriptBlocks > 0 || len(newInv.Functions) > 0 ||
			newInv.OnClicks > 0 || newInv.EventListeners > 0
		if oldHadInteractivity && !newHasInteractivity {
			return cwRefineValidateResult{
				OK:     false,
				Reason: "微调后页面的交互功能整体消失了（原页有点击/脚本交互，改后全部不见），疑似被截断或误删。已保留原版，请重试或换一种说法描述你的修改。",
				Detail: cwFmtAssetDetail(oldInv, newInv),
			}
		}

		// 2b. 函数定义大量丢失：原页函数改后丢了过半且至少丢2个 → 截断信号。
		lostFuncs := cwCountLostKeys(oldInv.Functions, newInv.Functions)
		if len(oldInv.Functions) >= 2 && lostFuncs >= 2 && lostFuncs*2 > len(oldInv.Functions) {
			return cwRefineValidateResult{
				OK:     false,
				Reason: "微调后页面丢失了多个交互函数（原页的部分功能在结果里不见了），疑似被截断。已保留原版，请重试或把修改拆小。",
				Detail: cwFmtAssetDetail(oldInv, newInv),
			}
		}

		// 2c. id 锚点大量丢失：原页 id 改后丢了过半且至少丢3个 → 卡片/区块成段消失。
		lostIDs := cwCountLostKeys(oldInv.IDs, newInv.IDs)
		if len(oldInv.IDs) >= 4 && lostIDs >= 3 && lostIDs*2 > len(oldInv.IDs) {
			return cwRefineValidateResult{
				OK:     false,
				Reason: "微调后页面丢失了大量内容区块（原页的多个卡片/容器在结果里不见了），疑似被截断。已保留原版，请重试或把修改拆小。",
				Detail: cwFmtAssetDetail(oldInv, newInv),
			}
		}
	}

	// ---- 校验3：体量骤降（意图非精简时）----
	if !cwInstructionAllowsRemoval(instruction) {
		oldLen := len([]rune(strings.TrimSpace(oldHTML)))
		newLen := len([]rune(strings.TrimSpace(htmlForCheck)))
		if oldLen >= 1500 && newLen*100 < oldLen*55 {
			return cwRefineValidateResult{
				OK:     false,
				Reason: "微调后页面体量大幅缩水（不到原来的一半），且你的修改并非删减内容，疑似被截断。已保留原版，请重试或把修改拆小。",
				Detail: cwFmtLenDetail(oldLen, newLen),
			}
		}
	}

	return cwRefineValidateResult{OK: true, FixedHTML: fixedHTML, Detail: fixedDetail}
}

// ==================== 辅助函数 ====================

// cwInstructionAllowsRemoval 判断老师意见是否属"删除/精简/移除"类——是则放宽资产/体量校验（保留结构闭合）。
func cwInstructionAllowsRemoval(instruction string) bool {
	s := strings.ToLower(strings.TrimSpace(instruction))
	if s == "" {
		return false
	}
	keywords := []string{
		"删除", "删掉", "去掉", "去除", "移除", "精简", "简化", "压缩",
		"减少", "拿掉", "砍掉", "清空", "只保留", "仅保留", "留下",
		"remove", "delete", "simplify", "reduce", "clean up", "strip",
	}
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// cwCountLostKeys 统计 oldSet 有、newSet 无的键数（资产丢失数）
func cwCountLostKeys(oldSet, newSet map[string]struct{}) int {
	lost := 0
	for k := range oldSet {
		if _, ok := newSet[k]; !ok {
			lost++
		}
	}
	return lost
}

// cwFmtTagDetail 格式化标签闭合差值（日志用）
func cwFmtTagDetail(which string, dO, dC, sO, sC, yO, yC int) string {
	var sb strings.Builder
	sb.WriteString("tag_mismatch=")
	sb.WriteString(which)
	sb.WriteString(" div(")
	sb.WriteString(cwItoa(dO))
	sb.WriteString("/")
	sb.WriteString(cwItoa(dC))
	sb.WriteString(") script(")
	sb.WriteString(cwItoa(sO))
	sb.WriteString("/")
	sb.WriteString(cwItoa(sC))
	sb.WriteString(") style(")
	sb.WriteString(cwItoa(yO))
	sb.WriteString("/")
	sb.WriteString(cwItoa(yC))
	sb.WriteString(")")
	return sb.String()
}

// cwFmtAssetDetail 格式化资产比对差值（日志用）
func cwFmtAssetDetail(oldInv, newInv cwAssetInventory) string {
	var sb strings.Builder
	sb.WriteString("assets old{ids:")
	sb.WriteString(cwItoa(len(oldInv.IDs)))
	sb.WriteString(",func:")
	sb.WriteString(cwItoa(len(oldInv.Functions)))
	sb.WriteString(",onclick:")
	sb.WriteString(cwItoa(oldInv.OnClicks))
	sb.WriteString(",listener:")
	sb.WriteString(cwItoa(oldInv.EventListeners))
	sb.WriteString(",script:")
	sb.WriteString(cwItoa(oldInv.ScriptBlocks))
	sb.WriteString("} new{ids:")
	sb.WriteString(cwItoa(len(newInv.IDs)))
	sb.WriteString(",func:")
	sb.WriteString(cwItoa(len(newInv.Functions)))
	sb.WriteString(",onclick:")
	sb.WriteString(cwItoa(newInv.OnClicks))
	sb.WriteString(",listener:")
	sb.WriteString(cwItoa(newInv.EventListeners))
	sb.WriteString(",script:")
	sb.WriteString(cwItoa(newInv.ScriptBlocks))
	sb.WriteString("}")
	return sb.String()
}

// cwFmtLenDetail 格式化体量差值（日志用）
func cwFmtLenDetail(oldLen, newLen int) string {
	return "len old=" + cwItoa(oldLen) + " new=" + cwItoa(newLen)
}

// cwItoa 轻量 int→string（避免仅为日志拼接引入 strconv）
func cwItoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
