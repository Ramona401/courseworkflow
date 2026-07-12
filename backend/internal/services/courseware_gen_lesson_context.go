package services

// courseware_gen_lesson_context.go — 课件HTML生成的"教案正文校准"取数与按页定向匹配
//
// 【为什么有这个文件】
//   课件生成是"教案 → 层2方案(8字段) → 逐页HTML"两段式。方案是唯一中转，
//   而方案是对教案的有损压缩：方案没写到的细节，AI 生成 HTML 时只能脑补，于是与教案偏离。
//   （老实践平台生成页面时教案在上文常驻，本套没有，故偏离。）
//
//   本文件的职责：在生成/重生/微调每一页 HTML 时，把"教案原文中与本页最相关的那一段"
//   作为【事实校准锚】注入提示词，让 AI 有据可依，而不是靠方案8字段凭空发挥。
//
// 【为什么按页定向而非整篇常驻】
//   - 整篇教案动辄数千字，每页都拼进 prompt → token 随页数线性放大（30页课件=教案×30）；
//   - 教案是"一节课的设计"，单页只对应其中一小段，整篇注入会稀释当前页焦点，AI 可能抓错重点。
//   故：入口一次性取教案全文（判 source_type=lesson_plan），每页再做轻量关键词匹配截取最相关段落。
//   纯 Go 字符串处理，零 AI 调用，零额外生成成本。
//
// 【被谁调用】
//   - courseware_gen_service.go：GenerateRemainingPages / GeneratePreviewPages 入口取全文，
//     经 buildBatchUserPrompt / buildPreviewUserPrompt 逐页定向拼接（用 appendLessonPlanCalibration）。
//   - courseware_gen_refine.go：
//       · RegenerateSinglePage（从零重画最易脑补）→ 复用批量 build，走 appendLessonPlanCalibration；
//       · RefinePage（基于现有HTML按指令增量改）→ 走 appendLessonPlanCalibrationForRefine（措辞克制版，
//         教案仅作"执行本次修改时的事实参照"，绝不借机改动老师没要求的地方，服从最小改动主任务）。
//
// 【安全边界】
//   - 只对 source_type=lesson_plan 且 lesson_plan_id 非空的课件注入；其余来源(主题/PPT/Doc/3D)返空串，
//     行为与改造前完全一致，零回归。
//   - 任一步取数失败(教案已删/查库异常/正文为空)一律返空串，绝不阻断生成——最坏等于改造前现状。
//
// 【增强：解决"教案有、方案没写到的细节，课件仍不体现"】
//   同事反馈：像"第3题第一问处给一个停顿动画""小掘对三个点子分别红灯/问号/打转"这类
//   深埋在教案某个环节长段落里的单点细节，虽然机制已在注入教案，却仍漏掉。根因三条：
//     1. 打分只吃 Title+ContentSummary，没吃 MediaRequirements（动画/交互类细节常在多媒体需求里），
//        导致这类细节所在段落匹配不上、定位不到；
//     2. 教案按空行切成碎段后逐段独立打分，某个细节句所在的正文段整体命中率不如标题段，直接落选，
//        命中的是干巴巴的标题段、真正带细节的正文段反而被丢；
//     3. 单页上限 2500 rune 偏紧，装不下"教学环节+对应课件设计"两块相关内容。
//   针对性增强（均在本文件内，调用契约不变）：
//     A. 关键词来源扩展为 Title + ContentSummary + MediaRequirements；
//     B. 命中段落"连带相邻上下文一起带出"（命中段前后各带一段），把被孤立的细节句拉回视野——最关键一招；
//     C. 单页上限 2500 → 3800 rune。
//   所有安全边界（非教案来源返空、全0分保底、rune安全截断）逐字保留。
//
// 【阶段一增强：跨页共享案例一致性（解决 P5/P6 共享案例对不上）】
//   同事反馈：教案里"6个点子""三个关卡题目"这类案例清单，本应被多页(如 P5/P6)共用且完全一致，
//   但并行生成时每页各自按自身关键词定向截取教案片段，两页拿到的教案依据可能不同段、或被 3800 上限
//   截成不同部分，加上方案层压缩后各页关键词本就有差异，于是 AI 各页自行现编案例、结果互不一致。
//   （见对话截图案例：P5/P6 的 6 个点子应完全一致却完全不同。）
//
//   治本思路——绕开"按页定向截取"的自由度，额外叠加一层【课件级共享案例段】：
//     · 从教案全文里识别"枚举型案例清单"整段（"N个点子/例子/关卡/案例/应用/情境"+ 列举结构），
//       把这段完整原文作为全课件统一、逐字相同的公共上下文；
//     · 对每一页都注入这段完全相同的案例原文（措辞为"若本页涉及这套案例则必须与之完全一致，
//       若不涉及则忽略此段"），从根上消除"各页各自现编案例"的自由度。
//   为什么全页注入而非"只注入涉及案例的页"：判断"哪些页涉及案例"需要 AI 或复杂规则、易判错，
//   判错=没治；全页注入 + "涉及才用"的措辞，代价只是每页多几百 rune，却 100% 保证任何用到案例的页
//   都拿到同一份，不引入新的判断环节（与"按页定向片段"并列，各司其职、互不干扰）。
//   安全边界：非枚举型教案(识别不到案例清单)返空串、不注入，行为与改造前完全一致，零回归。
//   纯 Go 规则识别，零 AI 调用，零 DB 改动。

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 常量：截断上限 ====================

const (
	// cwLessonFullContextMaxRunes 教案全文注入生成上下文的整体上限（rune计数，防中文截半）。
	//   教案正文普遍 2000~5000 字，8000 上限足够覆盖绝大多数完整教案；
	//   极长教案截断到 8000 也够做逐页定向匹配的"素材池"。
	cwLessonFullContextMaxRunes = 8000

	// cwLessonPageSectionMaxRunes 单页定向截取的教案相关片段上限（rune计数）。
	//   增强：2500 → 3800。教案单个环节的"详细教学过程 + 对应课件设计"两块相关内容
	//   经常合计 3000+ 字，2500 会把带细节的课件设计段截掉。3800 在"给足细节"与
	//   "不稀释本页焦点、不让 prompt 过大"之间取平衡（仍远小于全文 8000）。
	cwLessonPageSectionMaxRunes = 3800

	// cwLessonSectionSplitMinRunes 段落切分后过短段落的丢弃阈值：
	//   短于此长度的"段落"多为标题行/空行/单句，不足以作校准素材，打分时跳过。
	cwLessonSectionSplitMinRunes = 20

	// cwLessonNeighborRadius 命中段落"连带相邻上下文"的半径：
	//   一个命中段（得分>0）在拼回时，连同其前后各 N 段一起纳入。
	//   目的：教案里"环节标题段命中关键词、但真正的细节句在紧邻的正文/课件设计段"这种常见结构，
	//   靠单段独立打分会漏掉细节段；带上相邻段即可把被孤立的细节句拉回。
	//   取 1（前后各一段）：既能兜住"标题段命中→正文段被带出"，又不至于把无关段大量灌入。
	cwLessonNeighborRadius = 1

	// cwSharedExampleMaxRunes 课件级共享案例段的注入上限（rune计数，防中文截半）。
	//   共享案例清单（"6个点子"及其展开）通常几百到一千余字；2200 足以容纳一份完整案例清单，
	//   又不至于让每页 prompt 过大。超长则 rune 安全截断（宁可截尾也保住前面的核心案例条目）。
	cwSharedExampleMaxRunes = 2200
)

// ==================== 入口：取教案全文（判来源 + 截断） ====================

// loadLessonPlanContextForGen 取课件对应教案的正文，供本次批量/重生/微调生成全程复用。
//
// 判定与取数：
//  1. 仅当 cw.SourceType == lesson_plan 且 cw.LessonPlanID 非空非空串时才取；否则返空串。
//  2. 查教案 → ExtractLessonPlanContentForCW 按优先级链提取正文
//     （content_markdown → conversation_log最长assistant消息 → ai_review_result → ai_review_history）。
//  3. 正文按 cwLessonFullContextMaxRunes 截断（超长截取前段，够做逐页定向匹配素材池）。
//
// 返回空串的所有情形（均不阻断生成，最坏退回改造前现状）：
//   - 非教案来源课件；LessonPlanID 为空；教案查不到(已删/异常)；教案正文为空或过短。
//
// 由生成入口调用一次，返回值随 build 函数传给每一页，避免每页重复查库。
func loadLessonPlanContextForGen(ctx context.Context, cw *models.Courseware) string {
	if cw == nil {
		return ""
	}

	// 规整缓存优先：若本课件已有可用的规整结果（去噪保核、预置清单一字不差的干净教案），
	//   直接用它作为生成上下文——比原始教案排版更干净、按环节切段更可靠、跨页案例更一致。
	//   这正是本次"教案预处理规整层"的落点：把又长又乱的教案在生成前先规整好。
	// best-effort：查不到/未完成(非done)/正文空 → 静默退回下方原有的原文提取逻辑，零回归。
	//   规整同样按 cwLessonFullContextMaxRunes 截断，与原文路径的上限口径完全一致。
	if norm, e := repository.GetNormalizedByCoursewareID(ctx, cw.ID); e == nil && norm != nil && norm.HasUsableContent() {
		normContent := strings.TrimSpace(norm.NormalizedContent)
		if normContent != "" {
			runes := []rune(normContent)
			if len(runes) > cwLessonFullContextMaxRunes {
				normContent = string(runes[:cwLessonFullContextMaxRunes])
			}
			cwGenLog.Info("课件生成采用规整后教案作校准上下文",
				"courseware_id", cw.ID, "norm_runes", len([]rune(normContent)))
			return normContent
		}
	}

	// 按课件来源分流取"原文全文"（仿 courseware_index_refine.go 的 buildRefineSourceContext）：
	//   - lesson_plan：从教案表取正文（原有逻辑）；
	//   - doc_upload ：重新读上传的 docx 全文（本次新增——doc 上传的往往就是完整教案，
	//                  同样需要案例一致性与教案校准，此前被漏在门外导致 P6/P7 各自现编案例）；
	//   - 其余来源（ppt/topic/3d/html）：无可靠原文，返空串，行为与改造前完全一致、零回归。
	var content string
	switch cw.SourceType {
	case models.CWSourceLessonPlan:
		if cw.LessonPlanID == nil || strings.TrimSpace(*cw.LessonPlanID) == "" {
			return ""
		}
		lp, err := repository.GetLessonPlanByID(ctx, *cw.LessonPlanID)
		if err != nil || lp == nil {
			// 教案已删或查询异常：不阻断生成，退回"无教案校准"的原有行为。
			cwGenLog.Warn("课件生成取教案正文失败，本次不注入教案校准",
				"courseware_id", cw.ID, "lesson_plan_id", *cw.LessonPlanID, "error", err)
			return ""
		}
		content = strings.TrimSpace(ExtractLessonPlanContentForCW(lp))

	case models.CWSourceDocUpload:
		// doc_upload：原文以 .docx 文件形式存于 DocUploadDir/SourceFilePath。
		//   实时读取解析（readDocxFullText 为同包纯标准库实现，毫秒级，相对单页AI耗时可忽略）。
		//   路径拼法与 RefineIndex 的 buildRefineSourceContext 完全一致，避免拼错读不到。
		if strings.TrimSpace(cw.SourceFilePath) == "" {
			return ""
		}
		docFullPath := filepath.Join(DocUploadDir, cw.SourceFilePath)
		text, err := readDocxFullText(docFullPath)
		if err != nil || strings.TrimSpace(text) == "" {
			// 文件缺失/解析失败/空内容：不阻断生成，退回"无原文校准"行为。
			cwGenLog.Warn("课件生成读取docx原文失败，本次不注入原文校准",
				"courseware_id", cw.ID, "path", docFullPath, "error", err)
			return ""
		}
		content = strings.TrimSpace(text)

	default:
		// ppt_upload / topic_direct / 3d_single / html_import：无可靠原文，跳过。
		return ""
	}

	if content == "" {
		return ""
	}

	// rune 安全截断，防中文截半（两种来源统一在此收口）
	runes := []rune(content)
	if len(runes) > cwLessonFullContextMaxRunes {
		content = string(runes[:cwLessonFullContextMaxRunes])
	}
	return content
}

// ==================== 按页定向匹配：从教案全文截取与本页最相关的片段 ====================

// extractPageRelevantLessonSection 从教案全文中截取"与本页方案最相关"的片段。
//
// 匹配思路（纯 Go，零 AI）：
//  1. 把教案全文按空行/段落分隔符切成若干段；
//  2. 用本页 Title + ContentSummary + MediaRequirements 抽出的关键词，对每段做命中计数打分；
//  3. 命中段落（得分>0）连同其前后各 cwLessonNeighborRadius 段一起纳入（把被孤立的细节段拉回），
//     去重后按原文顺序拼回，累计不超过 cwLessonPageSectionMaxRunes；
//  4. 若全部段落都 0 分（本页与教案措辞对不上），退回教案正文开头一段作保底校准锚，
//     保证 AI 至少有一段真实教案上下文可依，不至于完全脑补。
//
// lessonContent 为空(非教案来源/取数失败)时直接返空串——调用方据此不注入。
func extractPageRelevantLessonSection(lessonContent string, page *models.CoursewarePage) string {
	lessonContent = strings.TrimSpace(lessonContent)
	if lessonContent == "" || page == nil {
		return ""
	}

	// ---- 1. 切段：按连续空行/换行切成候选段落，丢弃过短噪音段 ----
	sections := cwSplitLessonSections(lessonContent)
	if len(sections) == 0 {
		return cwTruncateRunes(lessonContent, cwLessonPageSectionMaxRunes)
	}

	// ---- 2. 抽关键词：本页标题 + 内容概要 + 多媒体需求 分词 ----
	//   增强：纳入 MediaRequirements。教案里"停顿动画/红灯/问号/打转/剥洋葱"这类
	//   过程细节，方案侧往往落在多媒体需求字段里；只吃标题+概要会定位不到细节所在段。
	keywordSource := page.Title + " " + page.ContentSummary + " " + page.MediaRequirements
	keywords := cwExtractPageKeywords(keywordSource)

	// ---- 3. 无关键词可用时，退回教案开头一段作保底 ----
	if len(keywords) == 0 {
		return cwTruncateRunes(lessonContent, cwLessonPageSectionMaxRunes)
	}

	// ---- 4. 逐段打分 ----
	scores := make([]int, len(sections))
	anyHit := false
	for i, sec := range sections {
		scores[i] = cwScoreSectionByKeywords(sec, keywords)
		if scores[i] > 0 {
			anyHit = true
		}
	}

	// ---- 5. 全 0 分：本页与教案措辞对不上，退回开头一段保底 ----
	if !anyHit {
		return cwTruncateRunes(lessonContent, cwLessonPageSectionMaxRunes)
	}

	// ---- 6. 命中段"连带相邻上下文"：把命中段及其前后 radius 段一并标记为选中 ----
	//   关键增强：教案常见"环节标题段命中关键词、真正的教学细节在紧邻的实施段/课件设计段"，
	//   靠单段独立打分会把带细节的段丢掉；带上相邻段即可兜住。
	selected := make([]bool, len(sections))
	for i := range sections {
		if scores[i] > 0 {
			lo := i - cwLessonNeighborRadius
			if lo < 0 {
				lo = 0
			}
			hi := i + cwLessonNeighborRadius
			if hi > len(sections)-1 {
				hi = len(sections) - 1
			}
			for j := lo; j <= hi; j++ {
				selected[j] = true
			}
		}
	}

	// ---- 7. 预算控制：若选中段落总长超上限，按"段落有效得分"降序裁剪 ----
	//   有效得分：命中段用自身分，被相邻带入的段(自身0分)给一个基础分1，
	//   保证"直接命中的段"优先于"仅因相邻被带入的段"被保留。
	type pickItem struct {
		idx    int
		effScr int
		text   string
		length int
	}
	items := make([]pickItem, 0, len(sections))
	total := 0
	for i, sel := range selected {
		if !sel {
			continue
		}
		eff := scores[i]
		if eff == 0 {
			eff = 1 // 仅因相邻被带入的上下文段，基础分1（低于任何直接命中段）
		}
		l := len([]rune(sections[i]))
		items = append(items, pickItem{idx: i, effScr: eff, text: sections[i], length: l})
		total += l
	}
	if len(items) == 0 {
		return cwTruncateRunes(lessonContent, cwLessonPageSectionMaxRunes)
	}

	// 若总长在上限内，直接按原文顺序拼回（items 收集时已按 idx 升序）
	if total <= cwLessonPageSectionMaxRunes {
		parts := make([]string, 0, len(items))
		for _, it := range items {
			parts = append(parts, it.text)
		}
		return cwTruncateRunes(strings.Join(parts, "\n\n"), cwLessonPageSectionMaxRunes)
	}

	// 超上限：按有效得分降序挑选，累计不超上限；得分相同者保持原顺序(稳定)。
	byScore := make([]pickItem, len(items))
	copy(byScore, items)
	sort.SliceStable(byScore, func(a, b int) bool {
		return byScore[a].effScr > byScore[b].effScr
	})
	keep := make(map[int]bool, len(byScore))
	used := 0
	for _, it := range byScore {
		if used > 0 && used+it.length > cwLessonPageSectionMaxRunes {
			continue // 超预算则跳过该段，尽量多塞几段高分内容
		}
		keep[it.idx] = true
		used += it.length
		if used >= cwLessonPageSectionMaxRunes {
			break
		}
	}
	if len(keep) == 0 {
		// 极端兜底：最高分段单段就超上限，直接截断该段
		return cwTruncateRunes(byScore[0].text, cwLessonPageSectionMaxRunes)
	}

	// 按原文顺序(idx升序)还原被保留的段，保证读起来连贯
	parts := make([]string, 0, len(keep))
	for _, it := range items { // items 本就按 idx 升序
		if keep[it.idx] {
			parts = append(parts, it.text)
		}
	}
	return cwTruncateRunes(strings.Join(parts, "\n\n"), cwLessonPageSectionMaxRunes)
}

// ==================== 阶段一：课件级共享案例段提取（跨页一致性治理） ====================

// extractSharedExampleFromLesson 从教案全文中识别"枚举型案例清单"整段，作为全课件共享案例返回。
//
// 【解决什么】P5/P6 等多页共用的一套案例（"6个点子""三个关卡"），并行生成时各页自行现编致不一致。
//
//	本函数把这套案例的完整原文抽出一份，供所有页注入完全相同的文本，消除"各页各编"的自由度。
//
// 识别规则（纯 Go，零 AI；宁可漏识别也不误识别，识别不到就返空串→不注入→行为不变）：
//  1. 切段（复用 cwSplitLessonSections）；
//  2. 找"锚段"——同时满足：
//     (a) 含"计数短语"：数字/中文数字 + 案例类量词（个/种/条/项/则）+ 案例类名词
//     （点子/例子/案例/应用/情境/场景/关卡/题/问题/环节/步骤/角色/形象 等），
//     例如"6个点子""三个关卡""三道关卡题目""几个生活中的例子"；
//     (b) 该段或其紧邻后段带"列举结构"：①②③ / 1. 2. / 1、2、 / 多个"、"或多行短句
//     （避免把只是一句"我们来看6个例子"、后面却没真正列出的段落误判为案例清单）。
//  3. 命中锚段后，把锚段 + 其后若干"承接列举段"（含列举结构或明显是案例条目的短段）一起纳入，
//     直到遇到明显换主题的段落或达到 cwSharedExampleMaxRunes 上限；
//  4. 全程未命中任何锚段 → 返空串（非枚举型教案，不注入，零回归）。
//
// 返回：识别到的共享案例整段原文（已 rune 安全截断）；未识别到返空串。
func extractSharedExampleFromLesson(lessonContent string) string {
	lessonContent = strings.TrimSpace(lessonContent)
	if lessonContent == "" {
		return ""
	}

	sections := cwSplitLessonSections(lessonContent)
	if len(sections) == 0 {
		return ""
	}

	// ---- 1. 找锚段下标：计数短语 + 本段或紧邻后段带列举结构 ----
	anchorIdx := -1
	for i, sec := range sections {
		if !cwHasCountedExamplePhrase(sec) {
			continue
		}
		// 列举结构可能在锚段本身，也可能在紧邻的后一段（"下面是6个点子：\n\n① ...② ..."）
		hasList := cwHasEnumerationStructure(sec)
		if !hasList && i+1 < len(sections) {
			hasList = cwHasEnumerationStructure(sections[i+1])
		}
		if hasList {
			anchorIdx = i
			break
		}
	}
	if anchorIdx < 0 {
		return "" // 非枚举型教案：不注入，行为与改造前完全一致
	}

	// ---- 2. 从锚段起纳入"锚段 + 后续承接列举段"，直到换主题或达上限 ----
	var parts []string
	used := 0
	appendSec := func(idx int) bool {
		text := strings.TrimSpace(sections[idx])
		if text == "" {
			return true
		}
		l := len([]rune(text))
		if used > 0 && used+l > cwSharedExampleMaxRunes {
			return false // 达上限，停止纳入
		}
		parts = append(parts, text)
		used += l
		return used < cwSharedExampleMaxRunes
	}

	// 锚段必纳入
	if !appendSec(anchorIdx) {
		return cwTruncateRunes(strings.Join(parts, "\n\n"), cwSharedExampleMaxRunes)
	}

	// 向后延伸：只要后续段仍是"列举结构/明显案例条目"，就继续纳入；
	//   一旦连续遇到非列举段（换主题信号）即停止，避免把后面无关环节也卷进来。
	nonListStreak := 0
	for j := anchorIdx + 1; j < len(sections); j++ {
		sec := sections[j]
		if cwHasEnumerationStructure(sec) || cwLooksLikeExampleItem(sec) {
			nonListStreak = 0
			if !appendSec(j) {
				break
			}
			continue
		}
		// 非列举段：容忍紧邻锚段的一段承接说明（如"具体如下："已在锚段），
		//   但连续出现即视为换主题，停止延伸。
		nonListStreak++
		if nonListStreak >= 1 {
			break
		}
	}

	return cwTruncateRunes(strings.Join(parts, "\n\n"), cwSharedExampleMaxRunes)
}

// cwHasCountedExamplePhrase 判断段落是否含"计数短语"：数字/中文数字 + 案例类量词 + 案例类名词。
//
//	例："6个点子""三个关卡""三道题目""几个生活中的例子""3种应用场景"。
//	纯规则匹配：先找"数量词"(阿拉伯数字 或 一二三…十/几/数)，再在其后不远处找案例类名词。
func cwHasCountedExamplePhrase(section string) bool {
	if section == "" {
		return false
	}
	runes := []rune(section)
	// 案例类名词：命中其一即认为是"案例/枚举内容"的名词
	exampleNouns := []string{
		"点子", "例子", "案例", "应用", "情境", "场景", "关卡",
		"题目", "问题", "环节", "步骤", "角色", "形象", "示例", "实例", "例",
	}
	// 中文数字（含"几/数"这类模糊计数）
	cnNums := "一二三四五六七八九十两几数"
	// 案例类量词
	quantifiers := "个种条项则道张幅次"

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		isDigit := r >= '0' && r <= '9'
		isCnNum := strings.ContainsRune(cnNums, r)
		if !isDigit && !isCnNum {
			continue
		}
		// 数量词之后 6 个字符窗口内，找"量词/名词"组合：允许"6个点子"也允许"6点子""6例"
		hi := i + 7
		if hi > len(runes) {
			hi = len(runes)
		}
		window := string(runes[i:hi])
		// 窗口内需同时(或就近)出现 案例类名词；量词可有可无（"6例""3题"量词即名词）
		for _, noun := range exampleNouns {
			if strings.Contains(window, noun) {
				return true
			}
		}
		// 量词 + 名词稍远的情况：窗口含量词也放宽再看整段是否有名词（降低漏判）
		if strings.ContainsAny(window, quantifiers) {
			for _, noun := range exampleNouns {
				if strings.Contains(section, noun) {
					return true
				}
			}
		}
	}
	return false
}

// cwHasEnumerationStructure 判断段落是否带"列举结构"（多条并列，说明真的把清单列出来了）。
//
//	信号：圈码①②③ / 阿拉伯序号 1. 2. 或 1、2、 / 中文序号 一、二、 / 多个换行短句 / 多个顿号。
//	要求"多条"（≥2 处序号或 ≥2 个顿号或 ≥2 行），避免把只提一句的段落误判为清单。
func cwHasEnumerationStructure(section string) bool {
	if section == "" {
		return false
	}
	// 圈码序号
	circled := []string{"①", "②", "③", "④", "⑤", "⑥", "⑦", "⑧", "⑨", "⑩"}
	circledHit := 0
	for _, c := range circled {
		if strings.Contains(section, c) {
			circledHit++
		}
	}
	if circledHit >= 2 {
		return true
	}
	// 阿拉伯/中文序号 "1." "2." / "1、" "2、" / "一、" "二、"
	seqPairs := [][2]string{
		{"1.", "2."}, {"1、", "2、"}, {"1）", "2）"}, {"1)", "2)"},
		{"一、", "二、"}, {"第一", "第二"},
	}
	for _, p := range seqPairs {
		if strings.Contains(section, p[0]) && strings.Contains(section, p[1]) {
			return true
		}
	}
	// 多行短句（清单常见逐行列）：≥3 个换行
	if strings.Count(section, "\n") >= 3 {
		return true
	}
	// 多个顿号并列（"红灯、问号、打转"这类）：≥3 个顿号
	if strings.Count(section, "、") >= 3 {
		return true
	}
	return false
}

// cwLooksLikeExampleItem 判断段落"看起来像案例清单里的一个条目"（用于向后延伸纳入承接段）。
//
//	比 cwHasEnumerationStructure 更宽松：段落以序号/圈码/破折号开头，或较短且含案例类名词，
//	即视为案例条目；用于把"清单被拆成多段、每段一条"的情况整体带入。
func cwLooksLikeExampleItem(section string) bool {
	s := strings.TrimSpace(section)
	if s == "" {
		return false
	}
	// 以列举前缀开头
	prefixes := []string{"①", "②", "③", "④", "⑤", "⑥", "⑦", "⑧", "⑨", "⑩",
		"1.", "2.", "3.", "4.", "5.", "6.", "7.", "8.", "9.",
		"1、", "2、", "3、", "4、", "5、", "6、", "7、", "8、", "9、",
		"- ", "• ", "* ", "第"}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// ==================== 拼接：把定向片段拼成"教案原文校准段"追加进提示词 ====================

// appendLessonPlanCalibration 把"与本页相关的教案原文片段"拼成硬约束段，追加进 AI 生成提示词。
// 【用于：批量生成 / 单页重生（从零重画场景）】
//
// 措辞铁律（对齐"忠实教案、不脑补"目标）：
//   - 方案未写到的细节以教案原文为准；教案也没写的不得自行脑补编造；
//   - 本页只呈现与【本页方案】对应的教学环节，不得把教案其它环节的内容搬到本页(不越界不超纲)；
//   - 教案是事实来源，方案是本页范围，两者结合——用教案的"实"填方案的"框"。
//
// section 为空(非教案来源/无相关片段)时不追加任何内容，行为与改造前一致。
// 纯追加文本，不改任何既有逻辑；仿 appendRichnessGuidance 风格。
func (s *CoursewareGenService) appendLessonPlanCalibration(sb *strings.Builder, section string) {
	section = strings.TrimSpace(section)
	if section == "" {
		return
	}

	sb.WriteString("## 教案原文校准（事实来源，务必忠实）\n")
	sb.WriteString("下面是本课件所源自的教案原文中，与【本页方案】最相关的片段。请把它作为本页内容的事实来源：\n")
	sb.WriteString("- 【本页方案】只给出了本页的框架(标题/目的/概要)，具体的知识点讲法、例子、数据、步骤、结论等细节，凡教案原文写到的，一律以教案原文为准，不得改写或简化；\n")
	sb.WriteString("- 教案原文里写到的、与本页相关的具体交互与呈现细节（例如某道题在某处要有停顿/提示、某个角色对不同情况要有不同反应、某段动画要逐层展开等），只要属于本页范围，都要在页面上如实体现，不得因【本页方案】没提到就省略；\n")
	sb.WriteString("- 方案和教案都没有写到的细节，绝不允许自行脑补、编造或想当然填充；宁可少写，也不要写教案里没有的内容；\n")
	sb.WriteString("- 但本页只呈现与【本页方案】对应的那部分教学内容，不要把教案里属于其它页/其它环节的内容搬到本页(不越界、不超纲)；\n")
	sb.WriteString("- 简言之：用教案原文的\"实\"来填【本页方案】的\"框\"，做到既忠实教案、又聚焦本页。\n")
	sb.WriteString("\n【教案原文片段】\n")
	sb.WriteString(section)
	sb.WriteString("\n\n")
}

// appendLessonPlanCalibrationForRefine 把教案原文片段拼成"事实参照段"追加进【单页微调】提示词。
// 【用于：RefinePage（基于现有HTML按老师指令增量修改的场景）】
//
// 为什么微调要单独一版、且措辞必须"克制"：
//
//	微调的核心是「只改老师明确要求的那一处、其余原样完整保留」（见 RefinePage 系统提示词第1/3/8条）。
//	若把重生那版「方案没写的以教案为准、用教案的实填方案的框」直接塞进来，会与「最小改动」打架——
//	AI 可能借「教案校准」之名去改老师根本没提到的地方，破坏微调语义。
//	故本版把教案严格定位为「执行本次修改时的事实参照」：只在落实老师这次要求时用来核对事实/补正确细节，
//	老师没要求改的地方即便与教案有出入也绝不借机改动；定位服从于「最小改动」这一主任务。
//
// section 为空(非教案来源/无相关片段)时不追加任何内容，行为与改造前一致。
func (s *CoursewareGenService) appendLessonPlanCalibrationForRefine(sb *strings.Builder, section string) {
	section = strings.TrimSpace(section)
	if section == "" {
		return
	}

	sb.WriteString("\n\n## 教案原文参考（仅供核对事实，服从「最小改动」）\n")
	sb.WriteString("下面是本课件所源自的教案原文中，与本页最相关的片段，仅作为你执行老师这次修改时的事实参照：\n")
	sb.WriteString("- 若老师这次的修改要求涉及具体知识点、例子、数据、步骤、台词、交互细节，且教案原文对此有明确写法，请以教案原文为准来落实，避免改错或凭空编造；\n")
	sb.WriteString("- 【最重要】本参考不改变「只修改老师明确要求的部分、其余原样完整保留」这一铁律：老师没有要求改动的地方，即便与教案原文有出入，也绝不允许借此参考去改动；\n")
	sb.WriteString("- 不得因为看到教案里有更多内容，就往本页新增老师没有要求的内容；本次任务始终是「按老师意见做最小改动」，而非「照教案重写本页」。\n")
	sb.WriteString("\n【教案原文片段（仅供参考）】\n")
	sb.WriteString(section)
	sb.WriteString("\n")
}

// ==================== 阶段一：拼接课件级共享案例段（跨页一致性） ====================

// appendSharedExampleCalibration 把"课件级共享案例清单"拼成硬约束段，追加进【批量生成/单页重生】提示词。
//
// 【解决 P5/P6 案例对不上】把教案里的一套案例(如6个点子)完整原文，对每一页注入完全相同的这一段，
//
//	并硬性约束"凡涉及这套案例的页面，必须使用完全相同的这几个案例，不得各页自行替换/新增/改写"。
//	由此不同页拿到的是逐字相同的案例原文，AI 没有各自现编的空间，跨页一致性从根上得到保证。
//
// 措辞关键（全页注入 + "涉及才用"）：
//   - 若本页涉及这套案例 → 必须与清单完全一致；
//   - 若本页不涉及 → 忽略此段，不强行把案例塞进无关页面（避免污染不相关页）。
//
// lessonContent 为空 / 识别不到枚举型案例清单时，extractSharedExampleFromLesson 返空串，本函数不追加，
// 行为与改造前完全一致，零回归。参数为教案全文(已由入口取好)，共享案例段在此现算(纯字符串，微秒级)。
func (s *CoursewareGenService) appendSharedExampleCalibration(sb *strings.Builder, lessonContent string) {
	shared := extractSharedExampleFromLesson(lessonContent)
	if shared == "" {
		return
	}

	sb.WriteString("## 全课件共享案例清单（跨页必须一致，硬性要求）\n")
	sb.WriteString("下面这份案例清单来自教案原文，是本课件多个页面共用的同一套案例（例如同一组\"点子/例子/关卡/情境\"）。请严格遵守：\n")
	sb.WriteString("- 如果【本页方案】涉及这套案例（需要展示、举例、出题或延续其中的内容），本页必须使用下面清单里\"完全相同\"的这几个案例，一字不差地沿用其名称、内容与顺序，绝不允许自行替换、新增、删减或改写成别的案例；\n")
	sb.WriteString("- 多页共用这套案例时（如某页介绍这些案例、后续页对其中每个案例做练习或延伸），各页涉及的必须是同一批案例、同样的表述，保证前后页衔接一致、学生看到的是同一套内容；\n")
	sb.WriteString("- 如果【本页方案】并不涉及这套案例（本页讲的是别的内容），则忽略本段清单，不要强行把这些案例塞进本页。\n")
	sb.WriteString("\n【教案原文中的共享案例清单】\n")
	sb.WriteString(shared)
	sb.WriteString("\n\n")
}

// appendSharedExampleCalibrationForRefine 把课件级共享案例清单拼成"参照段"追加进【单页微调】提示词（克制版）。
//
// 与批量/重生版的区别：微调服从「最小改动」铁律，共享案例清单仅作为"落实老师本次修改时对齐案例的依据"，
// 绝不因看到清单就主动改动老师没要求的地方（例如老师只让改配色，不得借机把案例也改成清单里的）。
// 典型有用场景：老师说"这页的例子要和前面几页对上/统一"，此时清单正好提供权威依据。
//
// shared 为空(非枚举型教案)时不追加，行为与改造前一致。
func (s *CoursewareGenService) appendSharedExampleCalibrationForRefine(sb *strings.Builder, lessonContent string) {
	shared := extractSharedExampleFromLesson(lessonContent)
	if shared == "" {
		return
	}

	sb.WriteString("\n\n## 全课件共享案例清单（仅供对齐案例时参照，服从「最小改动」）\n")
	sb.WriteString("下面是本课件多个页面共用的同一套案例（来自教案原文）。仅在与老师这次修改要求相关时作为依据：\n")
	sb.WriteString("- 若老师这次要求本页的案例/例子与其它页对齐、统一或延续，请以下面这份清单为准，使用其中\"完全相同\"的案例、表述与顺序；\n")
	sb.WriteString("- 【最重要】不改变「只修改老师明确要求的部分、其余原样保留」的铁律：老师没有要求改动案例时，不得因看到此清单就把本页案例改掉。\n")
	sb.WriteString("\n【教案原文中的共享案例清单（仅供参照）】\n")
	sb.WriteString(shared)
	sb.WriteString("\n")
}

// ==================== 内部辅助（纯字符串处理） ====================

// cwSplitLessonSections 把教案正文切成候选段落。
//
//	优先按"连续空行(段落分隔)"切；切出的段落再按单换行细分并丢弃过短噪音段。
func cwSplitLessonSections(content string) []string {
	// 统一换行，按"空行"切成大段
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	// 先按 markdown 分隔线/连续空行切
	normalized = strings.ReplaceAll(normalized, "\n---\n", "\n\n")
	rawBlocks := strings.Split(normalized, "\n\n")

	out := make([]string, 0, len(rawBlocks))
	for _, blk := range rawBlocks {
		blk = strings.TrimSpace(blk)
		if blk == "" {
			continue
		}
		// 段落过短(纯标题/单句)也保留但打分时权重自然低；仅丢弃极短噪音
		if len([]rune(blk)) < cwLessonSectionSplitMinRunes {
			// 极短段落(如单个 # 标题)：仍保留，因为标题常含关键词有助定位，
			// 但若纯符号无中文/字母则丢弃
			if !cwHasMeaningfulChar(blk) {
				continue
			}
		}
		out = append(out, blk)
	}
	return out
}

// cwExtractPageKeywords 从"本页标题+概要+多媒体需求"抽取关键词(去重、去停用词、去过短词)。
//
//	中文按 2-4 字滑窗切出候选词 + 直接保留连续英文/数字串；
//	过滤常见教学停用词，避免"学生/掌握/理解"这类高频词干扰打分。
func cwExtractPageKeywords(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// 停用词：教学场景高频但对定位无区分度的词，命中它们不加分
	stop := map[string]bool{
		"学生": true, "掌握": true, "理解": true, "了解": true, "知道": true,
		"学习": true, "教学": true, "老师": true, "教师": true, "本页": true,
		"通过": true, "能够": true, "培养": true, "提高": true, "内容": true,
		"介绍": true, "讲解": true, "认识": true, "感受": true, "体会": true,
	}

	seen := map[string]bool{}
	var kws []string

	// 1) 抽连续的英文/数字串(如 AI、CPU、3D、Python)——这些是强区分度关键词
	var cur strings.Builder
	flushASCII := func() {
		w := strings.TrimSpace(cur.String())
		cur.Reset()
		if len([]rune(w)) >= 2 && !seen[strings.ToLower(w)] {
			seen[strings.ToLower(w)] = true
			kws = append(kws, w)
		}
	}
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			cur.WriteRune(r)
		} else {
			flushASCII()
		}
	}
	flushASCII()

	// 2) 中文按 2 字、3 字滑窗切候选词(覆盖大部分学科名词)
	runes := []rune(text)
	for _, n := range []int{2, 3} {
		for i := 0; i+n <= len(runes); i++ {
			seg := string(runes[i : i+n])
			if !cwIsAllChineseWord(seg) {
				continue
			}
			if stop[seg] || seen[seg] {
				continue
			}
			seen[seg] = true
			kws = append(kws, seg)
		}
	}
	return kws
}

// cwScoreSectionByKeywords 用关键词对单个教案段落打分：每命中一个关键词计其出现次数。
//
//	命中越多、频次越高，说明该段与本页越相关。
func cwScoreSectionByKeywords(section string, keywords []string) int {
	if section == "" || len(keywords) == 0 {
		return 0
	}
	lowerSec := strings.ToLower(section)
	score := 0
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		// 中文直接计数，英文/数字用小写匹配
		cnt := strings.Count(lowerSec, strings.ToLower(kw))
		score += cnt
	}
	return score
}

// cwTruncateRunes 按 rune 安全截断字符串，防中文截半。
func cwTruncateRunes(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}

// cwHasMeaningfulChar 判断字符串是否含有中文/字母/数字(而非纯符号)。
func cwHasMeaningfulChar(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return true
		}
		if r >= 0x4E00 && r <= 0x9FFF { // CJK 基本区
			return true
		}
	}
	return false
}

// cwIsAllChineseWord 判断一个候选词是否全为中文字符(CJK基本区)。
func cwIsAllChineseWord(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 0x4E00 || r > 0x9FFF {
			return false
		}
	}
	return true
}
