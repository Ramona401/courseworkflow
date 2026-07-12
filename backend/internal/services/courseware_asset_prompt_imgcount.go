package services

// courseware_asset_prompt_imgcount.go — 课件「AI 写配图提示词」的占位计数与数量硬约束模块
//
// 【为什么单独成文件】
// 主文件 courseware_asset_prompt.go 已达 671 行超 600 行红线，本模块从其中拆出，
// 专职解决一个长期存在的顽疾：一页 HTML 里有 N 个真实图片占位，但 AI 只写了少于 N 条提示词
// （例如 7 张占位只出 2 条），导致部分占位始终没有对应的生图提示词。
//
// 【根因（已由生产数据坐实）】
//  1. HTML 截断丢占位：页面主体 HTML 常超过旧上限 8000 字符（生产可见 13806 / 15707），
//     截断后靠后的占位 AI 根本看不到，自然不会为其写提示词。
//     —— 修复：主文件上限提到 24000，并在截断前先用 cwStripSVGBlocks 剥掉体量最大的 <svg> 块。
//  2. 数量无人校验：占位数量一直由 AI 自己"看着数"，AI 少数就少写，后端无从知晓、无从纠偏。
//     —— 修复：后端用 cwCountImgPlaceholders 在【截断前的原始 HTML】上精确数出"内容配图占位数 N"，
//        经 cwBuildImgCountConstraint 作为硬约束注入到给 AI 的用户输入；解析后再比对条数是否 = N。
//
// 【计数口径——极其重要】
//   - 数的是 <div class="img-placeholder"> 占位 div（提示词 prompt_courseware_generate 规定的标准占位格式），
//     而不是 <img> 标签。AI 生成 HTML 时用 <div class="img-placeholder" data-desc="...">🖼️ ...</div>
//     作为"此处需要真实图片"的占位符，后续配图环节据此识别并填入真实图片。
//   - 不数 <img> 标签（那是配图完成后才插入的真实图片，或者是机构 Logo）。
//   - 不数 CSS background-image、不数 SVG <image> 元素。
//   - N = 本页"需要 AI 配图的占位数"，与老师"这一页要配几张内容图"的直觉一致。
//
// 【v231 修复：占位识别从 <img> 改为 <div class="img-placeholder">】
//   此前 cwCountImgPlaceholders 只数 <img> 标签，但 prompt_courseware_generate 要求 AI 用
//   <div class="img-placeholder"> 作占位 → 计数永远返回 0 → 数量硬约束不注入 → AI 随意出 1-3 条。
//   现改为数 class 含 "img-placeholder" 的 <div> 标签，与提示词规定的占位格式和装配流程
//   (courseware_auto_assembly_media.go 的 cwEmptyPlaceholderRe) 完全对齐。

import (
	"fmt"
	"strings"
)

// cwPlaceholderClassMarker 图片占位 div 的 class 特征串。
// prompt_courseware_generate 规定占位格式为 <div class="img-placeholder" ...>，
// courseware_auto_assembly_media.go 的 cwEmptyPlaceholderRe 也按此 class 识别。
// 据此作为占位计数的唯一判据，三处（生成提示词/计数/装配识别）口径统一。
const cwPlaceholderClassMarker = "img-placeholder"

// cwStripSVGBlocks 去掉 HTML 中所有 <svg>...</svg> 块（大小写不敏感，含跨行）。
//
// 目的：SVG 自绘图形（迷宫/图标/图表/流程图）不是"需要真实图片的占位"，且其内部 <path d="..."> 的
// 路径数据往往极长（单个 SVG 可达数千字符），会把真正的占位挤到 HTML 截断区之外而丢失。
// 在"数占位"和"喂 HTML 给 AI"之前先剥掉 SVG，既不影响占位识别，又大幅压缩体量、避免截断丢图。
//
// 实现复用主文件同款 cwStripTagBlock（小写扫描定位成对标签，不依赖正则回溯），对成对 <svg> 足够可靠；
// 未闭合的残缺 <svg 会被 cwStripTagBlock 保守保留剩余内容（不误删），安全。
func cwStripSVGBlocks(html string) string {
	return cwStripTagBlock(html, "svg")
}

// cwCountImgPlaceholders 数出本页 HTML 中【需要 AI 配图的真实图片占位】数量。
//
// 规则：
//   - 逐个匹配 "<div"（大小写不敏感），要求其后紧跟空白 / '>' 之一（标签边界校验，
//     避免 "<divx" 这类非 div 标签误计）。
//   - 对每个 <div>，向后截取到该标签的 '>' 处（取整个开标签），检查其中是否含
//     cwPlaceholderClassMarker（即 "img-placeholder"）。命中即计入。
//   - 不匹配 <img> 标签（那是配图完成后插入的真实图片或机构 Logo，非占位）。
//   - 不匹配 SVG <image> 元素（SVG 自绘图形非占位，且入参已经过 cwStripSVGBlocks 剥除）。
//
// 入参应传【剥掉 SVG 后、尚未截断的原始页面 HTML】，保证计数不受截断影响、也不被 SVG 干扰。
// 返回值即"本页应产出的内容配图提示词条数"。
func cwCountImgPlaceholders(html string) int {
	if strings.TrimSpace(html) == "" {
		return 0
	}
	lower := strings.ToLower(html)
	marker := strings.ToLower(cwPlaceholderClassMarker)
	count := 0
	i := 0
	for {
		// 定位下一个 <div 标签
		idx := strings.Index(lower[i:], "<div")
		if idx < 0 {
			break
		}
		start := i + idx
		after := start + len("<div")
		// 标签边界校验：<div 之后必须是 空白 / '>'，否则是 <divx 之类非目标标签
		if after < len(lower) {
			c := lower[after]
			if c != ' ' && c != '>' && c != '\t' && c != '\n' && c != '\r' {
				i = after
				continue
			}
		}
		// 截取整个开标签（<div ... >），用于判断 class 是否含 img-placeholder
		tagEndRel := strings.Index(lower[start:], ">")
		var tag string
		if tagEndRel < 0 {
			tag = lower[start:] // 无闭合 '>'，保守取到末尾
			i = len(lower)
		} else {
			tag = lower[start : start+tagEndRel+1]
			i = start + tagEndRel + 1
		}
		// 检查该 <div> 开标签中是否含 img-placeholder class 标记
		if strings.Contains(tag, marker) {
			count++
		}
	}
	return count
}

// cwBuildImgCountConstraint 生成"数量硬约束"文案，注入到给 AI 的用户输入末尾。
//
// n 为 cwCountImgPlaceholders 数出的内容配图占位数：
//   - n <= 0：返回空串（本页无 img-placeholder 占位，交由系统提示词的"无占位返回空数组"分支处理，不加约束）。
//   - n >= 1：明确告知 AI 本页确切占位数，要求最终 JSON 数组长度恰好等于 n，逐个对应、不漏不多。
//
// 这是修复"多占位漏写"的核心：把"应有几张"从 AI 主观判断变成后端下发的确定值，AI 有确切数字后漏写率骤降。
func cwBuildImgCountConstraint(n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n## 【数量硬约束——必须严格满足】\n")
	b.WriteString(fmt.Sprintf("经系统精确统计，本页 HTML 中共有 **%d** 个需要真实图片的 `img-placeholder` 占位（即 `<div class=\"img-placeholder\" ...>` 占位区域）。\n", n))
	b.WriteString(fmt.Sprintf("你【必须】为这 %d 个占位【逐一】各写一条提示词，最终输出的 JSON 数组长度【必须恰好等于 %d】——一个都不能漏，也不要多写。\n", n, n))
	b.WriteString("请按占位在 HTML 中从上到下的出现顺序依次编写，每条的 caption 注明它对应页面中的哪个位置/用途（可参考占位的 data-desc 属性），便于逐一核对。\n")
	b.WriteString(fmt.Sprintf("若某个占位你判断其实是图标或装饰性小图，也【仍需保留一条】并在 caption 中注明，绝不能因此少于 %d 条。\n", n))
	return b.String()
}
