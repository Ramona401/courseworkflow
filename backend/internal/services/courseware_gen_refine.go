package services

// courseware_gen_refine.go — 课件导航栏微调 + 单页微调 + 单页重新生成
//
// 本文件包含：
//   - RefineNav：导航栏AI微调（同步，返回修改后HTML）
//   - RefinePage：单页AI微调（同步；支持可选截图多模态；提取后走 normalizeRootCanvas 压住画布契约）
//   - RegenerateSinglePage：单页重新生成（依据页面方案从零重画，复用批量零件 + normalizeRootCanvas）
//
// 页面级版本与回退：RefinePage 与 RegenerateSinglePage 在覆盖 html_content 前，
//   各调一次 s.SavePageVersionBeforeOverwrite 把旧 HTML 存为版本快照（refine/regenerate），
//   供老师查看历次版本并一键回退。统一快照入口实现见 courseware_page_version.go（内部判空跳过首次生成）。
//
// 教案原文校准改造（本次）：RegenerateSinglePage（从零重画，最易脑补跑偏）入口调
//   loadLessonPlanContextForGen(ctx, cw) 取教案正文，传给 buildBatchUserPrompt，
//   函数内按页定向匹配注入教案相关片段，令重生页面忠实教案、不脑补。
//   非教案来源返空串，行为与改造前一致。RefineNav/RefinePage 是基于现有HTML增量改，本轮不接教案校准。
//
// 【输出完整性校验闸门（截断防护）】
//   根因：RefinePage/RegenerateSinglePage 采用"全量重写整页"策略，AI 输出超 max_tokens 时服务端
//   静默截断，残缺 HTML（卡片/交互整段丢失、根容器未闭合）若被写库即表现为"微调后交互变少、内容删减"。
//   防护：在"存版之后、UpdateCWPageHTML 写库之前"插入 validateRefinedPageHTML 校验闸门
//   （实现见 courseware_gen_validate.go）——结构闭合 + 关键资产比对 + 体量骤降三重判定，
//   判为疑似截断即【保留原版、明确报错返回】，绝不静默写残缺品。
//   轻微漏闭合（AI 手滑只缺 1~2 个 </div> 且脚本/样式配平、尾部完整）由闸门自动补全后放行，
//   通过 vr.FixedHTML 返回补全后的 HTML；本文件写库时若 vr.FixedHTML 非空则用它替换待写 HTML，
//   使"差一个闭合标签"的正常微调不再被整页毙掉。
//   RefinePage 走全量校验（isRegenerate=false），RegenerateSinglePage 走结构闭合校验（isRegenerate=true）。

import (
	"context"
	"fmt"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== P0-2: 导航栏AI微调 ====================

// RefineNav 导航栏AI微调：根据老师修改意见调整导航栏样式
// 同步调用，返回修改后的导航栏HTML
// 每次修改都基于当前最新的导航栏HTML（支持多轮微调）
func (s *CoursewareGenService) RefineNav(ctx context.Context, coursewareID string, userID string, instruction string) (string, error) {
	// 1. 获取课件信息并校验
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return "", fmt.Errorf("课件不存在: %w", err)
	}
	// 集体备课（阶段4）：微调权从"仅作者"放宽到"作者 or 集体备课参与者"，且非锁定态。
	// 复用 CoursewareService.canRefineCourseware 统一判定（admin 不在此特判，方案C）。
	if canEdit, ceErr := (&CoursewareService{}).canRefineCourseware(ctx, cw, userID); ceErr != nil {
		return "", ceErr
	} else if !canEdit {
		return "", fmt.Errorf("无权操作此课件")
	}

	// 2. 获取当前导航栏HTML（可能是从封面页提取的，也可能是之前微调过的）
	currentNav := cw.NavTemplateHTML
	if strings.TrimSpace(currentNav) == "" {
		// 如果还没有保存导航栏模板，尝试从封面页提取
		pages, pErr := repository.ListCoursewarePages(ctx, coursewareID)
		if pErr != nil || len(pages) == 0 {
			return "", fmt.Errorf("没有可用的导航栏HTML")
		}
		for _, p := range pages {
			if p.PageNumber == 1 && p.HTMLContent != "" {
				currentNav = ExtractNavByMarkers(p.HTMLContent)
				break
			}
		}
		if currentNav == "" {
			return "", fmt.Errorf("无法从封面页提取导航栏")
		}
	}

	// 3. 构建微调系统提示词（严格约束AI只修改指定部分）
	systemPrompt := `你是课件导航栏样式微调助手。你会收到一段导航栏HTML代码和老师的修改意见。

【绝对约束】
1. 只修改老师明确要求修改的部分
2. 不得修改老师未提到的任何样式、颜色、字号、布局、文字
3. 不得添加新元素或删除现有元素
4. 不得修改整体结构（div嵌套关系不变）
5. 修改后必须保持导航栏总高度80px不变
6. 输出完整的修改后导航栏HTML，用<!-- NAV_START -->和<!-- NAV_END -->包裹

如果老师的要求模糊，选择最小改动方案。
直接输出修改后的HTML代码，不要输出任何解释文字。`

	// 4. 构建用户提示词
	userPrompt := "## 当前导航栏HTML\n```html\n" + currentNav + "\n```\n\n## 老师的修改意见\n" + instruction + "\n\n请根据修改意见调整导航栏HTML，保持80px高度不变。用<!-- NAV_START -->和<!-- NAV_END -->包裹输出。"

	// 5. 调用AI（使用低成本模型）
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), models.SceneCWNavRefine,
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", fmt.Errorf("获取AI配置失败: %w", err)
	}
	// v198：解析操作者所属学校ID，供模型境内/境外分流判定（导航栏微调，操作者=userID）
	navSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceCtx := &ai.TraceContext{SceneCode: models.SceneCWNavRefine, UserID: &userID, SchoolID: schoolIDPtr(navSchoolID)}
	result, aiErr := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if aiErr != nil {
		return "", fmt.Errorf("AI微调失败: %w", aiErr)
	}

	// 6. 从AI输出中提取导航栏HTML
	refined := ExtractNavByMarkers(result.Content)
	if refined == "" {
		// 兜底：直接提取HTML
		refined = s.extractHTMLFromAIOutput(result.Content)
	}
	if refined == "" {
		return "", fmt.Errorf("AI输出未包含有效的导航栏HTML")
	}

	// 7. 剥除页码元素并保存（模板不存页码，拼接时后端追加）
	refined = StripNavPageNumbers(refined)
	if dbErr := repository.UpdateCoursewareNavTemplate(ctx, coursewareID, refined); dbErr != nil {
		return "", fmt.Errorf("保存微调后的导航栏失败: %w", dbErr)
	}

	cwGenLog.Info("导航栏微调完成", "courseware_id", coursewareID, "instruction", instruction)

	// 8. 返回替换了页码的预览版本（用第1页页码展示）
	totalPages := 0
	pages, _ := repository.ListCoursewarePages(ctx, coursewareID)
	if len(pages) > 0 {
		totalPages = len(pages)
	}
	preview := injectPageNumIntoNav(refined, 1, totalPages)

	return preview, nil
}

// ==================== P0-4: 单页AI微调（支持可选截图多模态） ====================

// RefinePage 单页AI微调：根据老师修改意见调整指定页面
// 同步调用，返回修改后的完整页面HTML
// imageDataURI 可选：非空时走多模态Vision调用（让AI看到该页实际渲染截图来定位版面问题），
//
//	格式 data:image/png;base64,xxx；多模态失败时自动降级为纯文本微调。
func (s *CoursewareGenService) RefinePage(ctx context.Context, coursewareID string, userID string, pageNum int, instruction string, imageDataURI string) (string, error) {
	// 1. 获取课件信息并校验
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return "", fmt.Errorf("课件不存在: %w", err)
	}
	// 集体备课（阶段4）：微调权从"仅作者"放宽到"作者 or 集体备课参与者"，且非锁定态。
	// 复用 CoursewareService.canRefineCourseware 统一判定（admin 不在此特判，方案C）。
	if canEdit, ceErr := (&CoursewareService{}).canRefineCourseware(ctx, cw, userID); ceErr != nil {
		return "", ceErr
	} else if !canEdit {
		return "", fmt.Errorf("无权操作此课件")
	}

	// 2. 获取目标页面
	page, err := repository.GetCoursewarePageByNumber(ctx, coursewareID, pageNum)
	if err != nil {
		return "", fmt.Errorf("页面不存在: %w", err)
	}
	if page.HTMLContent == "" {
		return "", fmt.Errorf("该页面尚未生成HTML，无法微调")
	}

	// 教案原文校准（本次）：取教案正文，供下方按页定向匹配后作为「事实参照」注入微调提示词。
	//   非教案来源/取数失败返空串，appendLessonPlanCalibrationForRefine 内部判空跳过，行为与改造前一致。
	lessonContext := loadLessonPlanContextForGen(ctx, cw)

	hasImage := strings.TrimSpace(imageDataURI) != ""

	// 3. 构建微调系统提示词
	//   第8条「完整性义务」+ 结尾「截断主动预警」为截断防护——
	//   要求 AI 无论改动多少都输出结构完整可运行的整页，原有功能除非老师明确要求否则必须保留，
	//   有截断风险须主动告知而非静默输出残缺代码（与后端 validateRefinedPageHTML 校验闸门呼应）。
	systemPrompt := `你是课件页面微调助手。你会收到一页完整的课件HTML和老师的修改意见。

【绝对约束】
1. 只修改老师明确要求修改的部分
2. 不得修改导航栏（页面顶部80px区域，即<!-- NAV_START -->到<!-- NAV_END -->之间的内容）的任何内容
3. 不得修改老师未提到的布局、配色、字号、内容
4. 不得添加或删除页面主要结构块
5. 保持画布尺寸1920×1080不变，不得出现滚动条
6. 不得给最外层div添加任何 transform / scale，不得改最外层div的 width / height（缩放由播放器外层统一处理）
7. 输出完整的修改后页面HTML（从<div style="width:1920px开始到</div>结束）
8. 【完整性义务】无论你修改了多少内容，都必须输出结构完整、可直接运行的整页HTML：所有<div>/<script>/<style>标签必须成对闭合；原页面已有的卡片、交互脚本(onclick/函数/事件监听)、内容区块，除非老师明确要求删除，否则必须原样完整保留，绝不能因为"只改一处"就省略或丢弃其余部分。

如果老师的要求模糊，选择最小改动方案。
直接输出修改后的完整HTML代码，不要输出任何解释文字。
【截断预警】如果你预计完整输出会因为内容过长而无法在一次回复中写完，请不要静默地输出半截代码——而是先在第一行用一句话告知"内容过长可能无法完整输出，建议拆分修改"，再尽量输出，让系统能够识别并提示老师。`

	if hasImage {
		systemPrompt += "\n\n【关于截图】你会额外收到一张该页面在播放器中实际渲染的截图（注意：截图是1920×1080画布被等比缩放后的结果）。请结合截图定位老师描述的版面问题（如内容出界、文字与图片重叠、被裁切、错位等），但你修改的始终是源HTML里的固定px坐标与样式；不要因为截图是缩放后的就改动画布尺寸或给根容器加transform。"
	}

	// 4. 构建用户提示词
	userPrompt := fmt.Sprintf("## 当前页面HTML（第%d页：%s）\n```html\n%s\n```\n\n## 老师的修改意见\n%s\n\n请根据修改意见调整页面HTML，保持1920x1080画布不变，不修改导航栏。", pageNum, page.Title, page.HTMLContent, instruction)
	// 教案原文校准（本次）：按页定向匹配教案相关片段，用「克制版」追加为事实参照。
	//   与批量/重生的 appendLessonPlanCalibration 不同：这里严格服从微调的「最小改动」铁律，
	//   教案仅供落实老师本次要求时核对事实，绝不借机改动老师没提到的地方。section 为空则不追加。
	{
		var lpsb strings.Builder
		lpsb.WriteString(userPrompt)
		s.appendLessonPlanCalibrationForRefine(&lpsb, extractPageRelevantLessonSection(lessonContext, page))
		// 阶段一（跨页共享案例一致性·克制版）：若老师本次要求本页案例与其它页对齐/统一，
		//   共享案例清单提供权威依据；服从「最小改动」，老师没要求改案例时不借此改动。
		//   非枚举型教案识别不到则不注入，行为不变、零回归。
		s.appendSharedExampleCalibrationForRefine(&lpsb, lessonContext)
		userPrompt = lpsb.String()
	}

	// 5. 获取AI配置
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), models.SceneCWPageRefine,
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", fmt.Errorf("获取AI配置失败: %w", err)
	}
	// v198：解析操作者所属学校ID，供模型境内/境外分流判定（单页微调，操作者=userID）
	pageSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceCtx := &ai.TraceContext{SceneCode: models.SceneCWPageRefine, UserID: &userID, SchoolID: schoolIDPtr(pageSchoolID)}

	// 5.1 调用AI：有截图走多模态，失败则降级纯文本；无截图直接纯文本
	var result *ai.CallResult
	var aiErr error
	if hasImage {
		result, aiErr = ai.CallAIMultimodal(aiCfg, systemPrompt, userPrompt, imageDataURI, traceCtx)
		if aiErr != nil {
			cwGenLog.Warn("多模态微调失败，降级为纯文本微调",
				"courseware_id", coursewareID, "page_num", pageNum, "error", aiErr)
			result, aiErr = ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
		}
	} else {
		result, aiErr = ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	}
	if aiErr != nil {
		return "", fmt.Errorf("AI微调失败: %w", aiErr)
	}

	// 6. 提取HTML（截断到最后一个</div>，剥掉AI追加的解释文字/围栏残留）
	refined := s.extractHTMLFromAIOutput(result.Content)
	if refined == "" {
		return "", fmt.Errorf("AI输出未包含有效HTML")
	}

	// 6.5 归一化根容器：压住1920×1080画布契约、剥除AI误加的transform、补cw-page类
	refined = normalizeRootCanvas(refined)

	// 批次1（背景保持）：若AI微调中弄丢了注入的背景<style>块，按三级优先级幂等补注
	rStyleCfg := s.parseStyleConfig(cw.StyleConfig)
	if rTplInfo, rtErr := s.loadTemplateInfo(ctx, rStyleCfg.TemplateID); rtErr == nil {
		s.attachUserBackground(ctx, cw, rTplInfo)
		refined = s.applyTemplateBackground(refined, rTplInfo, pageNum)
	}

	// 【输出完整性校验闸门·截断防护】在存版之后、写库之前校验 AI 产出是否完整：
	//   结构闭合 + 关键资产比对(与原页 page.HTMLContent 对照) + 体量骤降三重判定。
	//   判为疑似截断 → 保留原版(不写库)、记日志、把人话原因返回前端，由前端提示老师重试/拆分。
	//   轻微漏闭合(只缺1~2个</div>且脚本样式配平、尾部完整)由闸门自动补全后放行，经 vr.FixedHTML 返回。
	//   微调走全量校验（isRegenerate=false）。
	vr := validateRefinedPageHTML(page.HTMLContent, refined, instruction, false)
	if !vr.OK {
		cwGenLog.Warn("单页微调输出未通过完整性校验，已保留原版未写库",
			"courseware_id", coursewareID, "page_num", pageNum,
			"reason", vr.Reason, "detail", vr.Detail, "instruction", instruction)
		return "", fmt.Errorf("%s", vr.Reason)
	}
	// 闸门做了轻微漏闭合自动补全：写库使用补全后的 HTML（FixedHTML 非空才替换）。
	if vr.FixedHTML != "" {
		cwGenLog.Info("单页微调输出经轻微漏闭合自动补全后写库",
			"courseware_id", coursewareID, "page_num", pageNum, "detail", vr.Detail)
		refined = vr.FixedHTML
	}

	// 【页面级版本】保存微调结果前，先把旧 HTML(page.HTMLContent) 存为一个 refine 版本快照，
	//   供老师改坏后回退到微调前。统一入口内部判空（首次生成旧值为空则跳过），存版失败不阻断微调。
	// 集体备课（阶段4）：若课件处于集体备课态，给版本备注加"【集体备课】"前缀，
	//   使版本历史能一眼看出这一版是集体备课期间改的（source 枚举仍保持 refine 语义纯净）。
	refineNote := instruction
	if cw.CollabState == models.CWCollabInSession {
		refineNote = "【集体备课】" + instruction
	}
	s.SavePageVersionBeforeOverwrite(ctx, page.ID, coursewareID, page.HTMLContent, models.CWPageVersionSourceRefine, refineNote)

	// 7. 保存微调结果
	if dbErr := repository.UpdateCWPageHTML(ctx, page.ID, refined, "", page.MatchedComponentIDs, models.CWPageStatusGenerated); dbErr != nil {
		return "", fmt.Errorf("保存微调结果失败: %w", dbErr)
	}

	cwGenLog.Info("单页微调完成", "courseware_id", coursewareID, "page_num", pageNum, "has_image", hasImage, "instruction", instruction)
	return refined, nil
}

// ==================== 单页重新生成（从方案从零重画，不基于现有HTML） ====================

// RegenerateSinglePage 单页重新生成：丢弃该页当前HTML，依据页面方案(scheme)从零重画内容区，
// 再拼接已确认的导航栏。与 RefinePage（基于现有HTML增量微调）不同——它是整页重做。
// 同步调用，返回重生后的完整页面HTML。复用批量生成的全部零件 + normalizeRootCanvas 画布闸门。
//
// 教案原文校准（本次）：从零重画最容易脱离教案脑补，故取教案正文按页定向匹配后注入 build。
func (s *CoursewareGenService) RegenerateSinglePage(ctx context.Context, coursewareID string, userID string, pageNum int) (string, error) {
	// 1. 课件校验
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return "", fmt.Errorf("课件不存在: %w", err)
	}
	// 集体备课（阶段4）：微调权从"仅作者"放宽到"作者 or 集体备课参与者"，且非锁定态。
	// 复用 CoursewareService.canRefineCourseware 统一判定（admin 不在此特判，方案C）。
	if canEdit, ceErr := (&CoursewareService{}).canRefineCourseware(ctx, cw, userID); ceErr != nil {
		return "", ceErr
	} else if !canEdit {
		return "", fmt.Errorf("无权操作此课件")
	}
	// 必须已确认导航栏（重生走批量模式：AI只产内容区，导航栏由后端拼接）
	if strings.TrimSpace(cw.NavTemplateHTML) == "" {
		return "", fmt.Errorf("请先确认导航栏样式后再重新生成单页")
	}

	// 2. 取全部页面（拿到目标页 + 总页数）
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(pages) == 0 {
		return "", fmt.Errorf("课件没有页面方案")
	}
	totalPages := len(pages)
	var page *models.CoursewarePage
	for _, p := range pages {
		if p.PageNumber == pageNum {
			page = p
			break
		}
	}
	if page == nil {
		return "", fmt.Errorf("第%d页不存在", pageNum)
	}

	// 3. 风格 + Logo + 提示词 + AI配置（与批量生成一致）
	styleCfg := s.parseStyleConfig(cw.StyleConfig)
	tplInfo, tErr := s.loadTemplateInfo(ctx, styleCfg.TemplateID)
	if tErr != nil {
		tplInfo = s.defaultTemplateInfo()
	}
	logoURL, orgName := s.resolveLogoAndOrg(ctx, cw, styleCfg)

	// 批次1（背景图库）：单页重生同样挂载老师选择的背景（三级优先级第一级）
	s.attachUserBackground(ctx, cw, tplInfo)

	// 教案原文校准（本次）：取一次教案正文，供本页定向匹配注入（非教案来源返空串，行为不变）
	lessonContext := loadLessonPlanContextForGen(ctx, cw)

	genPrompt, err := repository.GetCurrentPromptByKey("prompt_courseware_generate")
	if err != nil {
		return "", fmt.Errorf("加载生成提示词失败: %w", err)
	}
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "courseware_generate",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", fmt.Errorf("获取AI配置失败: %w", err)
	}

	// 4. 匹配组件 + 构建批量模式提示词（AI只产内容区）
	//   教案原文校准（本次）：末参传 lessonContext，函数内按页定向匹配注入教案相关片段。
	matchedComps := s.matchComponentsForPage(ctx, page, cw.Subject, cw.Grade)
	userPrompt := s.buildBatchUserPrompt(page, pageNum, totalPages, tplInfo, logoURL, orgName, matchedComps, cw, lessonContext)
	// 封面页(第1页)补封面提示（批量提示词默认不含第1页封面提示，仅重生第1页时需要）
	if pageNum == 1 {
		userPrompt = "⚠️ 这是封面页（第1页），请生成大标题居中的封面设计，突出课件标题、学科年级和机构品牌。\n\n" + userPrompt
	}

	// 5. 调用AI生成内容区
	// v198：解析操作者所属学校ID，供模型境内/境外分流判定（单页重生，操作者=userID）
	regenSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceCtx := &ai.TraceContext{SceneCode: "courseware_generate", UserID: &userID, SchoolID: schoolIDPtr(regenSchoolID)}
	result, aiErr := ai.CallAI(aiCfg, genPrompt.Content, userPrompt, traceCtx)
	if aiErr != nil {
		return "", fmt.Errorf("AI重新生成失败: %w", aiErr)
	}
	contentHTML := s.extractHTMLFromAIOutput(result.Content)
	if contentHTML == "" {
		return "", fmt.Errorf("AI输出未包含有效HTML")
	}

	// 6. 后端拼接导航栏（assembleFullPage 内含 normalizeRootCanvas 画布闸门）
	fullPageHTML := s.assembleFullPage(contentHTML, cw.NavTemplateHTML, pageNum, totalPages, tplInfo)

	// 【输出完整性校验闸门·截断防护】重生是"从零重画"，与原页资产无继承关系，
	//   故走结构闭合校验（isRegenerate=true）：只判 <div>/<script>/<style> 闭合与尾部是否断裂，
	//   不做资产/体量比对（重画后内容本就可能与原页大不同）。判为疑似截断即保留原版、报错返回。
	//   轻微漏闭合同样自动补全后放行，经 vr.FixedHTML 返回。
	vr := validateRefinedPageHTML(page.HTMLContent, fullPageHTML, "", true)
	if !vr.OK {
		cwGenLog.Warn("单页重生输出未通过完整性校验，已保留原版未写库",
			"courseware_id", coursewareID, "page_num", pageNum,
			"reason", vr.Reason, "detail", vr.Detail)
		return "", fmt.Errorf("%s", vr.Reason)
	}
	// 闸门做了轻微漏闭合自动补全：写库使用补全后的 HTML。
	if vr.FixedHTML != "" {
		cwGenLog.Info("单页重生输出经轻微漏闭合自动补全后写库",
			"courseware_id", coursewareID, "page_num", pageNum, "detail", vr.Detail)
		fullPageHTML = vr.FixedHTML
	}

	// 【页面级版本】保存重生结果前，先把旧 HTML(page.HTMLContent) 存为一个 regenerate 版本快照，
	//   供老师重生后觉得旧版更好时回退。统一入口内部判空（旧值为空则跳过），存版失败不阻断重生。
	s.SavePageVersionBeforeOverwrite(ctx, page.ID, coursewareID, page.HTMLContent, models.CWPageVersionSourceRegenerate, "")

	// 7. 保存
	matchedIDs := s.buildMatchedComponentIDs(matchedComps)
	if dbErr := repository.UpdateCWPageHTML(ctx, page.ID, fullPageHTML, "", matchedIDs, models.CWPageStatusGenerated); dbErr != nil {
		return "", fmt.Errorf("保存重生结果失败: %w", dbErr)
	}

	cwGenLog.Info("单页重新生成完成", "courseware_id", coursewareID, "page_num", pageNum, "model", result.ModelUsed, "tokens", result.TokensUsed)
	return fullPageHTML, nil
}
