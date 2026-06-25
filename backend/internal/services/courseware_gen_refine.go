package services

// courseware_gen_refine.go — 课件导航栏微调 + 单页微调 + 单页重新生成
//
// 本文件包含：
//   - RefineNav：导航栏AI微调（同步，返回修改后HTML）
//   - RefinePage：单页AI微调（同步；支持可选截图多模态；提取后走 normalizeRootCanvas 压住画布契约）
//   - RegenerateSinglePage：单页重新生成（依据页面方案从零重画，复用批量零件 + normalizeRootCanvas）
//
// 拆分自原 courseware_gen_service.go（v142 结构化日志迁移+模块化拆分）
//
// 页面级版本与回退（新增）：RefinePage 与 RegenerateSinglePage 在覆盖 html_content 前，
//   各调一次 s.SavePageVersionBeforeOverwrite 把旧 HTML 存为版本快照（refine/regenerate），
//   供老师查看历次版本并一键回退。统一快照入口实现见 courseware_page_version.go（内部判空跳过首次生成）。

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
	if cw.UserID != userID {
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

	// 7. 替换页码为占位符并保存
	refined = ReplaceNavPageNumbers(refined)
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
	preview := strings.ReplaceAll(refined, "{{PAGE_NUM}}", "1")
	preview = strings.ReplaceAll(preview, "{{TOTAL_PAGES}}", fmt.Sprintf("%d", totalPages))

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
	if cw.UserID != userID {
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

	hasImage := strings.TrimSpace(imageDataURI) != ""

	// 3. 构建微调系统提示词
	systemPrompt := `你是课件页面微调助手。你会收到一页完整的课件HTML和老师的修改意见。

【绝对约束】
1. 只修改老师明确要求修改的部分
2. 不得修改导航栏（页面顶部80px区域，即<!-- NAV_START -->到<!-- NAV_END -->之间的内容）的任何内容
3. 不得修改老师未提到的布局、配色、字号、内容
4. 不得添加或删除页面主要结构块
5. 保持画布尺寸1920×1080不变，不得出现滚动条
6. 不得给最外层div添加任何 transform / scale，不得改最外层div的 width / height（缩放由播放器外层统一处理）
7. 输出完整的修改后页面HTML（从<div style="width:1920px开始到</div>结束）

如果老师的要求模糊，选择最小改动方案。
直接输出修改后的完整HTML代码，不要输出任何解释文字。`

	if hasImage {
		systemPrompt += "\n\n【关于截图】你会额外收到一张该页面在播放器中实际渲染的截图（注意：截图是1920×1080画布被等比缩放后的结果）。请结合截图定位老师描述的版面问题（如内容出界、文字与图片重叠、被裁切、错位等），但你修改的始终是源HTML里的固定px坐标与样式；不要因为截图是缩放后的就改动画布尺寸或给根容器加transform。"
	}

	// 4. 构建用户提示词
	userPrompt := fmt.Sprintf("## 当前页面HTML（第%d页：%s）\n```html\n%s\n```\n\n## 老师的修改意见\n%s\n\n请根据修改意见调整页面HTML，保持1920x1080画布不变，不修改导航栏。", pageNum, page.Title, page.HTMLContent, instruction)

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

	// 6. 提取HTML（已修复：截断到最后一个</div>，剥掉AI追加的解释文字/围栏残留）
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

	// 【页面级版本】保存微调结果前，先把旧 HTML(page.HTMLContent) 存为一个 refine 版本快照，
	//   供老师改坏后回退到微调前。统一入口内部判空（首次生成旧值为空则跳过），存版失败不阻断微调。
	s.SavePageVersionBeforeOverwrite(ctx, page.ID, coursewareID, page.HTMLContent, models.CWPageVersionSourceRefine, instruction)

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
func (s *CoursewareGenService) RegenerateSinglePage(ctx context.Context, coursewareID string, userID string, pageNum int) (string, error) {
	// 1. 课件校验
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		return "", fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
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
	matchedComps := s.matchComponentsForPage(ctx, page, cw.Subject, cw.Grade)
	userPrompt := s.buildBatchUserPrompt(page, pageNum, totalPages, tplInfo, logoURL, orgName, matchedComps, cw)
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
