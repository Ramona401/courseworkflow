package services

// courseware_asset_prompt.go — 课件「AI 写详细提示词」服务（图片/视频）
//
// 给前端"✨AI 写详细提示词"按钮用：读本页方案(标题/目的/内容摘要/配图需求/视觉形式/交互类型)
// + 课件信息(标题/学科/年级/风格)，喂给系统提示词(prompt_courseware_image_prompt /
// prompt_courseware_video_prompt)，让模型产出"紧扣本页教学需求、可控、默认无文字"的详细提示词。
//
//   - SuggestImagePrompt: 返回【一条或多条】详细生图提示词(每条含 caption 用途说明 + prompt 正文)
//       AI 读本页配图需求自主判断该页要几张图(1-6 条)，每条独立可编辑/生成。
//       JSON 数组解析失败时兜底为单条(整段当作 prompt)，向后保证至少返回一条。
//   - SuggestVideoPrompt: 返回三件物料(首帧分镜图提示词 / 图生视频提示词 / 台词)，做 JSON 容错解析
//
// 复用 CoursewareGenService 的同款 AI 调用范式(GetEffectiveConfig + CallAI)。
// 场景码 courseware_media_prompt 未在 ai_scene_configs 显式配置时，自动回退全局默认模型。
//
// 优化记录：
//   - StyleConfig 提纯：只把配色/风格相关字段喂给模型，剔除 logo_url/template_id/org_name 等噪音(cwExtractStyleHint)
//   - 视频 JSON 解析加固：先剥 ```json 围栏再做花括号提取，减少静默退化(cwParseVideoStoryboardsJSON)
//   - 图片多提示词：新增 ImagePromptItem + cwParseImagePromptsJSON，解析 AI 输出的 JSON 数组；
//     非数组/解析失败时兜底为单条；上限 cwMaxImagePrompts 条防发散。
//   - 【多占位漏写修复(本轮)】三管齐下解决"一页 N 个 <img> 占位却只出少于 N 条提示词"：
//       ① 截断上限 cwImagePromptHTMLLimit 8000 → 24000，覆盖生产可见的 13806/15707 等长页；
//       ② cwExtractPageBodyHTML 截断前先 cwStripSVGBlocks 剥掉体量最大的 <svg> 块，
//          避免 SVG 路径数据把靠后的 <img> 占位挤出截断区而丢失；
//       ③ buildImagePromptUserInputFromHTML 用 cwCountImgPlaceholders 在【剥SVG后、截断前】的原始 HTML 上
//          精确数出"内容配图占位数 N"（自动排除 Logo），经 cwBuildImgCountConstraint 作为硬约束注入用户输入，
//          把"应有几张"从 AI 主观判断变成后端下发的确定值；SuggestImagePrompt 解析后比对条数是否 = N，不符记 Warn。
//     以上占位计数/剥SVG/约束文案在同 package 的 courseware_asset_prompt_imgcount.go。
//   - 风格锚点联动(图片轮)：写图片提示词时，若课件已设风格锚点，向 AI 输入注入三段上下文——
//       ①【已锚定视觉风格（最高优先级）】：从锚点 VAOCI 切出 A 属性段，要求 AI 严格采用该风格、
//          禁止自创"扁平插画"等冲突风格词（修复"皮克斯锚点却写出插画风"的根因：此前漏注入 A 段）；
//       ②【已锚定人物形象】：从锚点 VAOCI 切出 C 角色段，告知 AI"若本页出现人物则沿用统一外貌"——
//          但明确"是否出现人物由本页内容决定"，事物/示意图页(叶片、地图、光合作用等)就画事物本身、不强行加人；
//       ③【锚点图参考提示词】：锚点图当初的生成提示词(generation_prompt 非空时)，供 AI 参考画风措辞。
//     未设锚点则三段都不注入，AI 按 prompt_courseware_image_prompt 的"未锚定"分支自行决定画风与人物外貌。
//   - 风格锚点联动(视频轮)：buildMediaPromptUserInput 同样在课件已设锚点时注入上述三段(措辞按视频物料调整)，
//     供 prompt_courseware_video_prompt v2 据此约束：首帧图(storyboard_prompt)严守锚定风格、人物外貌留白；
//     图生视频(video_prompt)只描述运镜动作、不重述风格人物(首帧已锁，重述反干扰)；台词(narration)不受影响。
//     视频两步法的风格人物一致性在"首帧图"环节用图生图锁定，故 video 段不必再背风格约束。
//     判据确定性：AI 生成图 generation_prompt 存真实提示词，手动上传图存空串，故"非空即可参考"无需启发式。
//     注：本服务负责"写提示词文字层"——保证提示词措辞与锚点一致；出图保真度由生图环节(ref_image 图生图)负责。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/repository"
)

// sceneCWMediaPrompt AI 调用场景码（未配置则回退全局默认）
const sceneCWMediaPrompt = "courseware_media_prompt"

// cwMaxImagePrompts 单页最多产出的配图提示词条数(工程护栏,防 AI 在模糊页面上无限发散 + 保护 token)
const cwMaxImagePrompts = 6

// cwMaxVideoStoryboards 单页最多产出的视频分镜数(工程护栏, 防 AI 无限拆镜 + 控制时长/成本)
const cwMaxVideoStoryboards = 4

// ImagePromptItem 单条配图提示词(一张图)
type ImagePromptItem struct {
	Caption string `json:"caption"` // 该图用途的简短说明(8-20字,供前端区分多条分别对应什么)
	Prompt  string `json:"prompt"`
	// Size AI建议的图片尺寸（豆包合法档位：横 2560x1440 / 竖 1440x2560 / 方 1920x1920）；
	// 为空时由装配链回退默认 2560x1440。用于按占位形状做比例适配。json解析自动接住。
	Size string `json:"size,omitempty"` // 该图的详细中文生图提示词正文(150-300字)
}

// VideoStoryboardItem 单个分镜的三件物料(本轮: 视频提示词由单组改为按分镜数组返回)
type VideoStoryboardItem struct {
	Scene       string `json:"scene"`        // 本镜环节简述(8-20字, 供老师区分各镜)
	ImagePrompt string `json:"image_prompt"` // 本镜首帧分镜图提示词
	VideoPrompt string `json:"video_prompt"` // 本镜图生视频提示词
	Narration   string `json:"narration"`    // 本镜口播台词
}

// cwImagePromptHTMLLimit 喂给 AI 的页面主体 HTML 字符上限(防 SVG 巨型路径等爆 token)
//
// 从 8000 提到 24000：生产多图页普遍超 8000(可见 13806/15707)，旧上限会截掉靠后的 <img> 占位致漏写。
// 已在 cwExtractPageBodyHTML 里"截断前先剥 SVG"进一步压缩体量，24000 足以覆盖复杂长页。
const cwImagePromptHTMLLimit = 24000

// cwExtractPageBodyHTML 提取页面主体 HTML 供 AI 分析图片占位:
//   - 去掉所有 <style>...</style> 与 <script>...</script> 块(动画CSS/JS 对"找图片占位"无意义且占大量 token)
//   - 去掉所有 <svg>...</svg> 块(SVG 自绘图形不是真实图片占位, 且路径数据极长会把靠后的 <img> 挤出截断区)
//   - 超过 cwImagePromptHTMLLimit 字符则截断(仍超长的页找占位无需读完整内容)
//   - 不剥导航栏(几百字符无妨, AI 不会把导航 Logo 当图片占位), 保持实现简单可靠
func cwExtractPageBodyHTML(html string) string {
	html = strings.TrimSpace(html)
	if html == "" {
		return ""
	}
	// 去 <style>...</style>(大小写不敏感, 跨行)
	html = cwStripTagBlock(html, "style")
	// 去 <script>...</script>
	html = cwStripTagBlock(html, "script")
	// 去 <svg>...</svg>（截断前先剥，防 SVG 路径数据把 <img> 占位挤出截断区）
	html = cwStripSVGBlocks(html)
	html = strings.TrimSpace(html)
	// 截断超长
	r := []rune(html)
	if len(r) > cwImagePromptHTMLLimit {
		html = string(r[:cwImagePromptHTMLLimit]) + "\n<!-- ...(HTML 已截断, 仅用于识别图片占位) -->"
	}
	return html
}

// cwStripTagBlock 去掉 HTML 中所有 <tag>...</tag> 块(大小写不敏感, 含跨行)。
// 用简单的小写扫描定位, 不依赖正则的贪婪/回溯, 对成对标签足够可靠。
func cwStripTagBlock(html, tag string) string {
	lower := strings.ToLower(html)
	open := "<" + tag
	closeTag := "</" + tag + ">"
	var b strings.Builder
	i := 0
	for {
		idx := strings.Index(lower[i:], open)
		if idx < 0 {
			b.WriteString(html[i:])
			break
		}
		start := i + idx
		// 确认是 <style 或 <style> 或 <style ...>(下一个字符是空白/>/换行), 避免误伤 <styled> 这类
		after := start + len(open)
		if after < len(lower) {
			c := lower[after]
			if c != ' ' && c != '>' && c != '\t' && c != '\n' && c != '\r' {
				// 不是目标标签, 原样保留这个 '<' 继续找
				b.WriteString(html[i : start+1])
				i = start + 1
				continue
			}
		}
		end := strings.Index(lower[start:], closeTag)
		if end < 0 {
			// 没有闭合标签, 保留剩余全部(不误删)
			b.WriteString(html[i:])
			break
		}
		// 写入块之前的内容, 跳过整个 <tag>...</tag>
		b.WriteString(html[i:start])
		i = start + end + len(closeTag)
	}
	return b.String()
}

// cwExtractVAOCIField 从一行 VAOCI 索引文本中切出指定字段段(如 A 属性段 / C 角色段)的正文。
//
// VAOCI 落库形态为单行，字段以 " | " 分隔，例如：
//
//	风格锚点[d-a-c-a-b-b]: F:焦点 | L:前-..;中-..;后-.. | A:皮克斯3D渲染.. | C:角色1：约7-8岁.. | S:情境 | E:16:9
//
// 本函数定位 "{字母}:"（容忍全/半角冒号）起点，截到下一个字段分隔符(" | " 或 "|")之前（或行尾）。
//
// 稳健性考量：
//   - 各段内部可能含中文冒号(如"角色1：")，但字段分隔符是竖线，按竖线切分不会误伤内部冒号。
//   - 找字段标记时优先匹配带前导分隔符的形式(" | A:")，避免命中其它字段内部偶然出现的同名子串；
//     找不到再回退到裸标记("A:")。
//   - 任何解析不确定的情况一律返回空串，宁可不注入也不注入垃圾。
//
// 参数 letter 为字段字母(大写，如 "A" / "C")。
func cwExtractVAOCIField(vaoci string, letter string) string {
	s := strings.TrimSpace(vaoci)
	if s == "" || letter == "" {
		return ""
	}

	// 1) 定位字段起点：优先带分隔符的形式，避免命中其它字段内部子串
	startMarkers := []string{
		" | " + letter + ":", " | " + letter + "：",
		"| " + letter + ":", "| " + letter + "：",
		" |" + letter + ":", " |" + letter + "：",
	}
	idx := -1
	markerLen := 0
	for _, m := range startMarkers {
		if p := strings.Index(s, m); p >= 0 {
			idx = p
			markerLen = len(m)
			break
		}
	}
	// 回退：裸标记（取首个）
	if idx < 0 {
		for _, m := range []string{letter + ":", letter + "："} {
			if p := strings.Index(s, m); p >= 0 {
				idx = p
				markerLen = len(m)
				break
			}
		}
	}
	if idx < 0 {
		return ""
	}

	contentStart := idx + markerLen
	rest := s[contentStart:]

	// 2) 截到下一个字段分隔符之前（" | " 或 "|"）
	cut := len(rest)
	for _, sep := range []string{" | ", " |", "| ", "|"} {
		if p := strings.Index(rest, sep); p >= 0 && p < cut {
			cut = p
		}
	}
	return strings.TrimSpace(rest[:cut])
}

// cwExtractVAOCIStyleSection 切出 VAOCI 的【A 属性段】(视觉风格)。
// A 段是风格控制力最强的字段(皮克斯3D渲染、光影、色彩、质感等)，用于强压制本页配图风格。
// A 段一般不会是空语义(总会有风格描述)，故只做空白判定。
func cwExtractVAOCIStyleSection(vaoci string) string {
	return cwExtractVAOCIField(vaoci, "A")
}

// cwExtractVAOCICharSection 切出 VAOCI 的【C 角色段】(人物固定外貌)。
// 若 C 段为"无具体角色"等空语义，视为无可用人物形象，返回空串(调用方据此不注入人物段)。
func cwExtractVAOCICharSection(vaoci string) string {
	charSec := cwExtractVAOCIField(vaoci, "C")
	if charSec == "" {
		return ""
	}
	for _, empty := range []string{"无具体角色", "无角色", "无人物", "无", "none", "无具体人物"} {
		if charSec == empty {
			return ""
		}
	}
	return charSec
}

// buildImagePromptUserInputFromHTML 图片专用: 校验权限, 喂"页面主体 HTML"为主 + 少量方案语义辅助。
//
// 返回值：(用户输入文本, 检测到的内容配图占位数, error)。占位数供 SuggestImagePrompt 解析后校验条数。
//
//	配图提示词依据【当前页 HTML 实际的图片占位】(<img> 标签/明确的图片占位容器)产出:
//	有占位才给提示词, SVG/CSS 自绘图形不算占位, 无占位则 AI 返回空数组。
//	方案字段(标题/目的/摘要)仅作语义辅助, 帮 AI 理解每个占位该配什么图。
//
//	数量硬约束(本轮)：对【剥SVG后、截断前】的原始 HTML 用 cwCountImgPlaceholders 数出内容配图占位数 N，
//	  经 cwBuildImgCountConstraint 注入用户输入末尾，明确要求 AI 输出数组长度恰好 = N，杜绝多占位漏写。
//	  计数排除机构 Logo(<img src 含 courseware-logos/)，与老师"这一页要配几张内容图"的直觉一致。
//
//	风格锚点联动(图片轮)：课件已设锚点时，按优先级注入三段——
//	  ①【已锚定视觉风格（最高优先级）】(A 属性段) — 强压制画风，放在最显眼处；
//	  ②【已锚定人物形象】(C 角色段) — 人物外貌留白交给图生图；
//	  ③【锚点图参考提示词】(锚点图 generation_prompt 非空时) — 参考画风措辞。
//	系统提示词 v5 据这三段决定"严格采用锚定风格 / 不写详细人物外貌 / 参考锚点画风措辞"。
func (s *CoursewareAssetService) buildImagePromptUserInputFromHTML(
	ctx context.Context,
	coursewareID string,
	pageNum int,
	actor *CoursewareActorContext,
) (
	string,
	int,
	*CoursewareActorContext,
	error,
) {
	cw, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return "", 0, nil, err
	}

	page, err :=
		repository.GetCoursewarePageByNumber(
			ctx,
			coursewareID,
			pageNum,
		)
	if err != nil {
		return "", 0, nil, fmt.Errorf(
			"页面不存在: 课件=%s 页码=%d",
			coursewareID,
			pageNum,
		)
	}

	// 先在【剥SVG后、尚未截断】的原始 HTML 上精确数占位，保证计数不受截断影响、不被 SVG 干扰。
	// 顺序必须是先剥 SVG 再数：SVG 内不含 <img>，剥不剥不影响计数，但与喂给 AI 的正文同源更一致。
	rawForCount := cwStripSVGBlocks(page.HTMLContent)
	imgCount := cwCountImgPlaceholders(rawForCount)

	var b strings.Builder
	b.WriteString("## 课件整体信息\n")
	b.WriteString(fmt.Sprintf("- 课件标题：%s\n- 学科：%s\n- 年级：%s\n", cw.Title, cw.Subject, cw.Grade))
	if styleHint := cwExtractStyleHint(cw.StyleConfig); styleHint != "" {
		b.WriteString(fmt.Sprintf("- 课件配色/风格参考(JSON)：%s\n", styleHint))
	}

	// ---- 风格锚点联动：已设锚点时注入「视觉风格 + 人物形象 + 参考提示词」三段 ----
	if cw.StyleAnchorAssetID != nil && strings.TrimSpace(*cw.StyleAnchorAssetID) != "" {
		// ① 已锚定视觉风格（最高优先级）：从锚点 VAOCI 切出 A 属性段，强压制画风
		//    放在最显眼位置（紧接课件信息之后），并明确要求"严格采用、忽略冲突风格词"。
		if styleSec := cwExtractVAOCIStyleSection(cw.StyleAnchorVAOCI); styleSec != "" {
			b.WriteString("\n## 已锚定视觉风格（最高优先级，必须严格遵守）\n")
			b.WriteString("本套课件已锚定统一画风，以下为锚点图提取的风格 DNA。你写的每一条提示词的风格描述都【必须严格采用】此风格关键词，\n")
			b.WriteString("【严禁】改写为或混入\"扁平插画/扁平卡通/现代插画/写实插画/矢量插画\"等任何与之冲突的风格词；当与课件配色或页面 HTML 有出入时，一律以本风格为准。\n")
			b.WriteString(fmt.Sprintf("已锚定风格：%s\n", styleSec))
		}
		// ② 已锚定人物形象：从锚点 VAOCI 切出 C 角色段
		if charSec := cwExtractVAOCICharSection(cw.StyleAnchorVAOCI); charSec != "" {
			b.WriteString("\n## 已锚定人物形象（来自风格锚点）\n")
			b.WriteString("本套课件设定了以下统一角色形象，供【本页画面确实需要出现人物时】保持外貌一致（会用锚点图做图生图锁定）。\n")
			b.WriteString("【重要·是否出现人物由本页内容决定，不得强行加人】：先判断本页配图主体到底是什么——\n")
			b.WriteString("  · 若本页确实以人物/角色为主体（如情境对话、人物示范动作），则沿用下方锚定角色，提示词里不要再堆砌详细外貌，只点名角色并着重描述其在本页的动作/表情/场景/构图；\n")
			b.WriteString("  · 若本页主体是事物、自然对象、器材、图表或示意图（如叶片、细胞、地图、实验装置、光合作用流程等），则【就画该事物本身，不要塞入任何人物】，忠实呈现学科内容即可；\n")
			b.WriteString("  人物一致性是【若出现则须一致】的可选约束，绝不是【每页都必须有人物】的硬要求。\n")
			b.WriteString(fmt.Sprintf("已锚定角色形象：%s\n", charSec))
		}
		// ③ 锚点图参考提示词：取锚点资产的 generation_prompt(非空才注入；手动上传图为空串自动跳过)
		if anchorAsset, aerr := repository.GetCWAssetByID(ctx, *cw.StyleAnchorAssetID); aerr == nil && anchorAsset != nil {
			if refPrompt := strings.TrimSpace(anchorAsset.GenerationPrompt); refPrompt != "" {
				b.WriteString("\n## 锚点图参考提示词\n")
				b.WriteString("以下是本套课件锚点图当初的生成提示词，请参考其句式与画风措辞，使本页配图与锚点图画风一致（画面主体仍以本页占位需求为准）：\n")
				b.WriteString(refPrompt + "\n")
			}
		}
		// 注：上述各段任何一步失败或为空都只是"少注入一段"，不影响主流程；
		//     未设锚点则整体跳过，AI 走"未锚定"分支自行决定画风与人物外貌。
	}

	// 方案语义辅助(帮 AI 理解占位该配什么图, 非主依据)
	b.WriteString(fmt.Sprintf("\n## 本页方案语义参考（第 %d 页）\n", pageNum))
	b.WriteString(fmt.Sprintf("- 页面标题：%s\n", page.Title))
	if strings.TrimSpace(page.Purpose) != "" {
		b.WriteString(fmt.Sprintf("- 教学目的：%s\n", page.Purpose))
	}
	if strings.TrimSpace(page.ContentSummary) != "" {
		b.WriteString(fmt.Sprintf("- 内容摘要：%s\n", page.ContentSummary))
	}
	// 主依据: 当前页主体 HTML
	body := cwExtractPageBodyHTML(page.HTMLContent)
	b.WriteString("\n## 本页当前 HTML（主依据，请据此识别图片占位）\n")
	if body == "" {
		b.WriteString("（本页暂无 HTML 内容）\n")
	} else {
		b.WriteString("```html\n")
		b.WriteString(body)
		b.WriteString("\n```\n")
	}
	// 数量硬约束：把后端数出的确切占位数下发给 AI，要求逐一对应、数组长度恰好等于 N。
	// n<=0（无真实占位）时返回空串，交由系统提示词的"无占位返回空数组"分支处理。
	b.WriteString(cwBuildImgCountConstraint(imgCount))
	b.WriteString("\n请严格按系统提示词的规则, 仅为 HTML 中真实存在的图片占位产出提示词; 无图片占位则返回空数组。")
	return b.String(), imgCount, scopedActor, nil
}

// buildMediaPromptUserInput 校验权限并把本页方案+课件信息拼成喂给 AI 的用户输入（视频物料专用）
//
// 视频锚点联动(本轮)：课件已设锚点时，在课件信息之后、本页方案之前注入三段——
//
//	①【已锚定视觉风格】(A 段) — 供 storyboard_prompt(首帧图)严守锚定风格；
//	②【已锚定人物形象】(C 段) — 供首帧图人物外貌留白(图生视频保一致)；
//	③【锚点图参考提示词】(锚点图 generation_prompt 非空时) — 供首帧图参考画风措辞。
//
// 三段措辞针对"视频两步法"调整：明确风格人物锁定在首帧图、video_prompt 不必重述。
// prompt_courseware_video_prompt v2 据这三段是否存在决定各物料的分工。
// 未设锚点则三段全跳过，AI 走"未锚定"分支自行决定首帧图画风与人物外貌。
func (s *CoursewareAssetService) buildMediaPromptUserInput(
	ctx context.Context,
	coursewareID string,
	pageNum int,
	actor *CoursewareActorContext,
) (
	string,
	*CoursewareActorContext,
	error,
) {
	cw, scopedActor, err :=
		(&CoursewareService{}).
			LoadCoursewareForOwnerRuntime(
				ctx,
				coursewareID,
				actor,
			)
	if err != nil {
		return "", nil, err
	}

	page, err :=
		repository.GetCoursewarePageByNumber(
			ctx,
			coursewareID,
			pageNum,
		)
	if err != nil {
		return "", nil, fmt.Errorf(
			"页面不存在: 课件=%s 页码=%d",
			coursewareID,
			pageNum,
		)
	}

	var b strings.Builder
	b.WriteString("## 课件整体信息\n")
	b.WriteString(fmt.Sprintf("- 课件标题：%s\n- 学科：%s\n- 年级：%s\n", cw.Title, cw.Subject, cw.Grade))
	// 提纯：只把配色/风格相关字段喂给模型，剔除 Logo/模板ID/机构名等与画面风格无关的噪音
	if styleHint := cwExtractStyleHint(cw.StyleConfig); styleHint != "" {
		b.WriteString(fmt.Sprintf("- 课件配色/风格参考(JSON)：%s\n", styleHint))
	}

	// ---- 视频风格锚点联动：已设锚点时注入「视觉风格 + 人物形象 + 参考提示词」三段 ----
	// 措辞针对视频两步法（先出首帧图、再图生视频）：风格人物一致性锁定在首帧图环节。
	if cw.StyleAnchorAssetID != nil && strings.TrimSpace(*cw.StyleAnchorAssetID) != "" {
		// ① 已锚定视觉风格（最高优先级）：从锚点 VAOCI 切出 A 属性段，供首帧图严守画风
		if styleSec := cwExtractVAOCIStyleSection(cw.StyleAnchorVAOCI); styleSec != "" {
			b.WriteString("\n## 已锚定视觉风格（最高优先级，必须严格遵守）\n")
			b.WriteString("本套课件已锚定统一画风，以下为锚点图提取的风格 DNA。你写的 storyboard_prompt（首帧分镜图）的风格描述【必须严格采用】此风格关键词，\n")
			b.WriteString("【严禁】改写为或混入\"扁平插画/扁平卡通/现代插画/写实插画/矢量插画\"等任何与之冲突的风格词；当与课件配色或本页方案有出入时，一律以本风格为准。\n")
			b.WriteString(fmt.Sprintf("已锚定风格：%s\n", styleSec))
		}
		// ② 已锚定人物形象：从锚点 VAOCI 切出 C 角色段，供首帧图人物外貌留白
		if charSec := cwExtractVAOCICharSection(cw.StyleAnchorVAOCI); charSec != "" {
			b.WriteString("\n## 已锚定人物形象（来自风格锚点）\n")
			b.WriteString("本套课件设定了以下统一角色形象，供【本镜画面确实需要出现人物时】保持外貌一致（首帧图做图生视频锁定）。\n")
			b.WriteString("【重要·是否出现人物由本页内容决定，不得强行加人】：先判断本镜画面主体到底是什么——\n")
			b.WriteString("  · 若本镜确实以人物/角色为主体，则沿用下方锚定角色，storyboard_prompt 不要堆砌详细外貌，只点名角色并着重描述其在首帧中的动作/表情/场景/构图；\n")
			b.WriteString("  · 若本镜主体是事物、自然对象、器材、图表或示意图（如叶片、细胞、地图、实验装置、光合作用流程等），则【就画该事物本身，不要塞入任何人物】，忠实呈现学科内容即可；\n")
			b.WriteString("  人物一致性是【若出现则须一致】的可选约束，绝不是【每镜都必须有人物】的硬要求。\n")
			b.WriteString(fmt.Sprintf("已锚定角色形象：%s\n", charSec))
		}
		// ③ 锚点图参考提示词：取锚点资产的 generation_prompt(非空才注入；手动上传图为空串自动跳过)
		if anchorAsset, aerr := repository.GetCWAssetByID(ctx, *cw.StyleAnchorAssetID); aerr == nil && anchorAsset != nil {
			if refPrompt := strings.TrimSpace(anchorAsset.GenerationPrompt); refPrompt != "" {
				b.WriteString("\n## 锚点图参考提示词\n")
				b.WriteString("以下是本套课件锚点图当初的生成提示词，请参考其句式与画风措辞，使首帧图与锚点图画风一致（画面主体仍以本页需求为准）：\n")
				b.WriteString(refPrompt + "\n")
			}
		}
		// 注：上述各段任何一步失败或为空都只是"少注入一段"，不影响主流程；
		//     未设锚点则整体跳过，AI 走"未锚定"分支自行决定首帧图画风与人物外貌。
	}

	b.WriteString(fmt.Sprintf("\n## 本页方案（第 %d 页）\n", pageNum))
	b.WriteString(fmt.Sprintf("- 页面标题：%s\n", page.Title))
	if strings.TrimSpace(page.Purpose) != "" {
		b.WriteString(fmt.Sprintf("- 教学目的：%s\n", page.Purpose))
	}
	if strings.TrimSpace(page.ContentSummary) != "" {
		b.WriteString(fmt.Sprintf("- 内容摘要：%s\n", page.ContentSummary))
	}
	if strings.TrimSpace(page.MediaRequirements) != "" {
		b.WriteString(fmt.Sprintf("- 配图/媒体需求：%s\n", page.MediaRequirements))
	}
	if strings.TrimSpace(page.VisualFormat) != "" {
		b.WriteString(fmt.Sprintf("- 视觉形式：%s\n", page.VisualFormat))
	}
	if strings.TrimSpace(page.InteractionType) != "" {
		b.WriteString(fmt.Sprintf("- 交互类型：%s\n", page.InteractionType))
	}
	b.WriteString("\n请严格基于以上本页教学需求产出物料，不要编造与本页无关的内容。")
	return b.String(), scopedActor, nil
}

// persistCoursewareImagePromptSuggestions 在已授权Service内部保存配图建议。
//
// 落库属于缓存和复用增强，失败只记录日志，不影响老师取得本次AI结果。
func persistCoursewareImagePromptSuggestions(
	ctx context.Context,
	coursewareID string,
	pageNum int,
	items []ImagePromptItem,
) {
	if len(items) == 0 {
		return
	}

	data, err := json.Marshal(items)
	if err != nil {
		cwAssetLog.Warn(
			"配图建议JSON序列化失败",
			"courseware_id",
			coursewareID,
			"page_number",
			pageNum,
			"error",
			err,
		)
		return
	}

	if err := repository.UpdatePageImageSuggestions(
		ctx,
		coursewareID,
		pageNum,
		string(data),
	); err != nil {
		cwAssetLog.Warn(
			"配图建议落库失败(不影响响应)",
			"courseware_id",
			coursewareID,
			"page_number",
			pageNum,
			"error",
			err,
		)
	}
}

// persistCoursewareVideoPromptSuggestions 在已授权Service内部保存视频分镜。
func persistCoursewareVideoPromptSuggestions(
	ctx context.Context,
	coursewareID string,
	pageNum int,
	items []VideoStoryboardItem,
) {
	if len(items) == 0 {
		return
	}

	data, err := json.Marshal(items)
	if err != nil {
		cwAssetLog.Warn(
			"视频分镜JSON序列化失败",
			"courseware_id",
			coursewareID,
			"page_number",
			pageNum,
			"error",
			err,
		)
		return
	}

	if err := repository.UpdatePageVideoStoryboards(
		ctx,
		coursewareID,
		pageNum,
		string(data),
	); err != nil {
		cwAssetLog.Warn(
			"视频分镜落库失败(不影响响应)",
			"courseware_id",
			coursewareID,
			"page_number",
			pageNum,
			"error",
			err,
		)
	}
}

// SuggestImagePrompt 生成【一条或多条】详细、可控的生图提示词
// AI 读本页配图需求自主判断该页要几张图(1-cwMaxImagePrompts 条)，每条含 caption + prompt。
// 解析失败兜底为单条(整段当 prompt)；始终保证返回至少一条非空提示词，否则报错。
//
// 数量校验(本轮)：拿到后端数出的占位数 imgCount 后，与 AI 实际返回条数比对，不一致记 Warn 便于追踪
// "多占位漏写"是否复发；数量吻合记 Info。校验只记日志不阻断（AI 少写仍返回已有的，好过整体失败）。
func (s *CoursewareAssetService) SuggestImagePrompt(
	ctx context.Context,
	coursewareID string,
	pageNum int,
	actor *CoursewareActorContext,
) ([]ImagePromptItem, error) {
	userInput, imgCount, scopedActor, err :=
		s.buildImagePromptUserInputFromHTML(
			ctx,
			coursewareID,
			pageNum,
			actor,
		)
	if err != nil {
		return nil, err
	}

	systemPrompt, err :=
		repository.GetCurrentPromptByKey(
			"prompt_courseware_image_prompt",
		)
	if err != nil {
		return nil, fmt.Errorf(
			"加载配图提示词模板失败: %w",
			err,
		)
	}

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		sceneCWMediaPrompt,
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"获取AI配置失败: %w",
			err,
		)
	}

	userID := scopedActor.UserID
	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			userID,
		)

	traceContext := &ai.TraceContext{
		SceneCode: sceneCWMediaPrompt,
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	result, aiErr := ai.CallAI(
		aiConfig,
		systemPrompt.Content,
		userInput,
		traceContext,
	)
	if aiErr != nil {
		return nil, fmt.Errorf(
			"AI生成提示词失败: %w",
			aiErr,
		)
	}

	items := cwParseImagePromptsJSON(
		result.Content,
	)

	if len(items) == 0 {
		fallback := cwStripPromptWrappers(
			result.Content,
		)
		if fallback != "" {
			items = []ImagePromptItem{
				{
					Caption: "",
					Prompt:  fallback,
				},
			}
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf(
			"AI未返回有效提示词",
		)
	}

	if imgCount >= 1 &&
		len(items) != imgCount {
		cwAssetLog.Warn(
			"AI配图提示词条数与占位数不一致",
			"courseware_id",
			coursewareID,
			"page_number",
			pageNum,
			"expected_placeholders",
			imgCount,
			"got_prompts",
			len(items),
		)
	} else {
		cwAssetLog.Info(
			"AI配图提示词生成成功",
			"courseware_id",
			coursewareID,
			"page_number",
			pageNum,
			"placeholders",
			imgCount,
			"count",
			len(items),
		)
	}

	persistCoursewareImagePromptSuggestions(
		ctx,
		coursewareID,
		pageNum,
		items,
	)

	return items, nil
}

// SuggestVideoPrompt 生成视频分镜数组(每镜含 scene/image_prompt/video_prompt/narration)
// AI 按本页内容自主拆 1-cwMaxVideoStoryboards 个分镜; 解析失败兜底为单分镜(整段当 video_prompt);
// 始终保证返回至少一个分镜, 否则报错。
func (s *CoursewareAssetService) SuggestVideoPrompt(
	ctx context.Context,
	coursewareID string,
	pageNum int,
	actor *CoursewareActorContext,
) ([]VideoStoryboardItem, error) {
	userInput, scopedActor, err :=
		s.buildMediaPromptUserInput(
			ctx,
			coursewareID,
			pageNum,
			actor,
		)
	if err != nil {
		return nil, err
	}

	systemPrompt, err :=
		repository.GetCurrentPromptByKey(
			"prompt_courseware_video_prompt",
		)
	if err != nil {
		return nil, fmt.Errorf(
			"加载配视频提示词模板失败: %w",
			err,
		)
	}

	aiConfig, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(),
		sceneCWMediaPrompt,
		s.cfg.AIAPIBaseURL,
		s.cfg.AIAPIKey,
		s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"获取AI配置失败: %w",
			err,
		)
	}

	userID := scopedActor.UserID
	schoolID, _ :=
		repository.GetSchoolIDByUserID(
			ctx,
			userID,
		)

	traceContext := &ai.TraceContext{
		SceneCode: sceneCWMediaPrompt,
		UserID:    &userID,
		SchoolID:  schoolIDPtr(schoolID),
	}

	result, aiErr := ai.CallAI(
		aiConfig,
		systemPrompt.Content,
		userInput,
		traceContext,
	)
	if aiErr != nil {
		return nil, fmt.Errorf(
			"AI生成视频物料失败: %w",
			aiErr,
		)
	}

	items := cwParseVideoStoryboardsJSON(
		result.Content,
	)

	if len(items) == 0 {
		fallback := cwStripPromptWrappers(
			result.Content,
		)
		if fallback != "" {
			items = []VideoStoryboardItem{
				{
					VideoPrompt: fallback,
				},
			}
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf(
			"AI未返回有效视频物料",
		)
	}

	cwAssetLog.Info(
		"AI视频分镜物料生成成功",
		"courseware_id",
		coursewareID,
		"page_number",
		pageNum,
		"count",
		len(items),
	)

	persistCoursewareVideoPromptSuggestions(
		ctx,
		coursewareID,
		pageNum,
		items,
	)

	return items, nil
}

// cwExtractStyleHint 从课件 StyleConfig(JSON) 提纯出与画面配色/风格相关的字段，
// 剔除 logo_url / template_id / org_name 等与生图无关的噪音，降噪 + 省 token。
// 解析失败或提纯后为空时返回 ""(宁可不传，也不塞噪音)。
func cwExtractStyleHint(styleConfig string) string {
	styleConfig = strings.TrimSpace(styleConfig)
	if styleConfig == "" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(styleConfig), &m); err != nil {
		return ""
	}
	for _, k := range []string{
		"logo_url", "logoUrl", "org_name", "orgName",
		"template_id", "templateId", "templateID",
	} {
		delete(m, k)
	}
	if len(m) == 0 {
		return ""
	}
	out, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(out)
}

// cwStripPromptWrappers 去掉模型可能误加的 markdown 代码围栏与首尾引号
func cwStripPromptWrappers(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
	}
	s = strings.Trim(s, "\"'“”")
	return strings.TrimSpace(s)
}

// cwRepairInnerQuotes 兜底修复模型 JSON 的常见破坏：字符串值内部出现【未转义的半角双引号】
// （模型常给 "+1"/"+10" 等加强调，导致字符串提前闭合、json.Unmarshal 失败）。
// 仅在严格解析失败后调用：线性扫描维护是否在字符串内；串内遇到 " 时看其后首个非空白字符，
// 为 : , } ] （键/值结束结构符）才算真闭引号，否则判为杂散内部引号并补转义。
// 中文全角标点不在结构符集合内，故值内中文标点不误判；合法 JSON 走到这步也原样还原，无副作用。
func cwRepairInnerQuotes(s string) string {
	r := []rune(s)
	n := len(r)
	var b strings.Builder
	inStr := false
	for i := 0; i < n; i++ {
		c := r[i]
		if inStr {
			if c == '\\' {
				b.WriteRune(c)
				if i+1 < n {
					b.WriteRune(r[i+1])
					i++
				}
				continue
			}
			if c == '"' {
				j := i + 1
				for j < n && (r[j] == ' ' || r[j] == '\t' || r[j] == '\n' || r[j] == '\r') {
					j++
				}
				if j >= n || r[j] == ':' || r[j] == ',' || r[j] == '}' || r[j] == ']' {
					inStr = false
					b.WriteRune(c)
				} else {
					b.WriteString("\\\"")
				}
				continue
			}
			b.WriteRune(c)
			continue
		}
		if c == '"' {
			inStr = true
		}
		b.WriteRune(c)
	}
	return b.String()
}

// cwParseImagePromptsJSON 从(可能带围栏/前后缀的)模型输出里提取 JSON 数组并解析为多条配图提示词。
// 加固：先用 cwStripPromptWrappers 剥掉 ```json 围栏，再做方括号提取(首个 '[' 到最后一个 ']')。
// 仅保留 prompt 非空的条目；caption 去空白；超过 cwMaxImagePrompts 条时截断。
// 解析失败或无有效条目时返回 nil(由调用方走单条兜底)。
func cwParseImagePromptsJSON(raw string) []ImagePromptItem {
	s := cwStripPromptWrappers(raw)
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		return nil
	}
	frag := s[start : end+1]
	var arr []ImagePromptItem
	if err := json.Unmarshal([]byte(frag), &arr); err != nil {
		arr = nil
		_ = json.Unmarshal([]byte(cwRepairInnerQuotes(frag)), &arr)
	}
	if len(arr) == 0 {
		return nil
	}
	out := make([]ImagePromptItem, 0, len(arr))
	for _, it := range arr {
		p := strings.TrimSpace(it.Prompt)
		if p == "" {
			continue // prompt 为空的条目无意义，跳过
		}
		out = append(out, ImagePromptItem{
			Caption: strings.TrimSpace(it.Caption),
			Prompt:  p,
			Size:    strings.TrimSpace(it.Size),
		})
		if len(out) >= cwMaxImagePrompts {
			break // 工程护栏：最多 cwMaxImagePrompts 条
		}
	}
	return out
}

// cwParseVideoStoryboardsJSON 从(可能带围栏/前后缀的)模型输出里解析视频分镜数组。
// 容错三级: 1) 按 JSON 数组解析(v3 输出); 2) 退化为单 JSON 对象并兼容旧字段 storyboard_prompt, 兜成 1 镜;
// 3) 仍失败返回 nil, 由调用方把整段当作 1 镜的 video_prompt 兜底。
// 仅保留 image_prompt 或 video_prompt 至少一项非空的分镜; 上限 cwMaxVideoStoryboards 个。
func cwParseVideoStoryboardsJSON(raw string) []VideoStoryboardItem {
	s := cwStripPromptWrappers(raw)

	// 1) 数组解析(v3)
	if start := strings.Index(s, "["); start >= 0 {
		if end := strings.LastIndex(s, "]"); end > start {
			frag := s[start : end+1]
			var arr []VideoStoryboardItem
			if err := json.Unmarshal([]byte(frag), &arr); err != nil {
				arr = nil
				_ = json.Unmarshal([]byte(cwRepairInnerQuotes(frag)), &arr)
			}
			if len(arr) > 0 {
				out := make([]VideoStoryboardItem, 0, len(arr))
				for _, it := range arr {
					item := VideoStoryboardItem{
						Scene:       strings.TrimSpace(it.Scene),
						ImagePrompt: strings.TrimSpace(it.ImagePrompt),
						VideoPrompt: strings.TrimSpace(it.VideoPrompt),
						Narration:   strings.TrimSpace(it.Narration),
					}
					if item.ImagePrompt == "" && item.VideoPrompt == "" {
						continue
					}
					out = append(out, item)
					if len(out) >= cwMaxVideoStoryboards {
						break
					}
				}
				if len(out) > 0 {
					return out
				}
			}
		}
	}

	// 2) 单对象兜底(兼容旧 v2 字段 storyboard_prompt)
	if start := strings.Index(s, "{"); start >= 0 {
		if end := strings.LastIndex(s, "}"); end > start {
			var obj struct {
				Scene            string `json:"scene"`
				ImagePrompt      string `json:"image_prompt"`
				StoryboardPrompt string `json:"storyboard_prompt"`
				VideoPrompt      string `json:"video_prompt"`
				Narration        string `json:"narration"`
			}
			frag := s[start : end+1]
			if json.Unmarshal([]byte(frag), &obj) != nil {
				_ = json.Unmarshal([]byte(cwRepairInnerQuotes(frag)), &obj)
			}
			{
				img := strings.TrimSpace(obj.ImagePrompt)
				if img == "" {
					img = strings.TrimSpace(obj.StoryboardPrompt)
				}
				item := VideoStoryboardItem{
					Scene:       strings.TrimSpace(obj.Scene),
					ImagePrompt: img,
					VideoPrompt: strings.TrimSpace(obj.VideoPrompt),
					Narration:   strings.TrimSpace(obj.Narration),
				}
				if item.ImagePrompt != "" || item.VideoPrompt != "" {
					return []VideoStoryboardItem{item}
				}
			}
		}
	}

	return nil
}
