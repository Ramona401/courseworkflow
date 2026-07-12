package services

// courseware_auto_assembly_media.go —— 全自动装配「单页处理」实现
//
// 本文件是 courseware_auto_assembly_service.go（主编排）的搭档，专注单页级别的三条干活链：
//   ①  generateOnePageHTML     —— 单页 HTML 生成（复用 gen_service 私有零件，与批量生成完全一致）
//   ② assembleOnePageMedia    —— 单页配图链 + 视频首帧占位链的统一入口
//        · 配图链：SuggestImagePrompt(取主图) → GenerateImage(带锚点) → 上云回写 →
//              · 封面页(第1页)：占位直填（字符串替换，不重排、不调 RefinePage，保住标题版式）
//              · 非封面页：RefinePage 融图（AI 重写把图融进占位，版面协调）→ postProcessPageImages 收尾
//        · 视频占位链：关键词命中页 SuggestVideoPrompt(取首镜) → 首帧图 → 上云 → 落到分镜
//          （交付模式"HTML+配图不做视频"即 pc.skipVideo=true 时，所有页一律跳过视频占位链）
//
// 【封面为何单独走"占位直填"，而非 RefinePage 融图】—— v0.44 关键修复
//   RefinePage 是"整页重写融图"：对普通内容页版面协调效果好，但对封面这种强排版页（大标题居中、
//   精心的品牌/学科年级布局）会挪动、覆盖、冲乱标题版式（历史反馈"封面被改坏"的根因）。
//   因此封面（page.PageNumber==1）改走 fillCoverImageIntoPlaceholder：纯字符串处理——
//   定位封面 HTML 里第一个 img-placeholder 占位 div，【保留其 class/style 外壳】，
//   只把 div 内部替换成一张撑满占位的 <img>。占位 div 之外的一切（标题、品牌、布局）
//   一个字符都不动，从根上杜绝封面被改坏。若封面无占位/定位失败，降级回 RefinePage（不失配图能力）。
//
// 【非封面为何用 RefinePage 融图，而非前端插标签】
//   本平台真实配图路径是"统一微调融合能力"：图片上云拿公网URL → 连同含图片占位的整页HTML交给AI →
//   AI 建议重写把图融进它预留的占位div，融合自然、版面协调。因此装配配图链复用 gen_service.RefinePage
//   （instruction 带图片公网URL），与老师手动配图同一条已验证路径、同等质量。
//   RefinePage 内部自带「不改导航栏/保持1920×1080/完整性校验闸门/权限校验」，装配直接复用即安全。
//
// 【融图收尾后处理 postProcessPageImages —— 本轮关键修复(A+B)】
//   RefinePage 让 AI「重写整页」把图融进占位，但 AI 转写长 URL 时会偶发写坏（实测：
//   把 "https://" 写成 "https"、或在 URL 中间插入空格），导致该图 src 失效、图裂开只显示 alt 文字；
//   且一页多占位时 AI 往往只填最匹配的一处，其余 img-placeholder 占位空留 "🖼️..." 裂框。
//   因此非封面融图成功后，统一追加 postProcessPageImages 收尾（全程 best-effort、失败不影响主流程）：
//     A. URL 校正闸门(sanitizeImgSrcInHTML)：纯语法修复所有 <img src>——
//        为 http/https 补回丢失的 "://"、去掉 src 属性值内部的空格；URL 由代码修，不信任 AI 转写。
//     B. 空占位按需兜底(fillOrRemoveEmptyPlaceholders)：扫描残留的 img-placeholder 空占位 div——
//        · 本页确有配图需求(MediaRequirements 非空) → 用本页主图 publicURL 补填(复用主图，胜过裂框)；
//        · 本页无配图需求 → 收起该空占位 div，不留 "🖼️..." 裂框。
//   （封面走占位直填、URL 由代码拼接不经过 AI，天然不会写坏，故封面分支不经此后处理。）
//
// 全部步骤均为 best-effort：任一步失败只影响当前页对应结果标记，不 panic、不中断整体装配。
//
// SSE 约定：GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{EventType: "assembly_xxx", Data: {...}})

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// cwAssemblyVideoKeywords 视频首帧占位触发关键词
// （方案 MediaRequirements/VisualFormat/InteractionType 命中任一即为该页生成首帧占位）
var cwAssemblyVideoKeywords = []string{"视频", "动画", "演示", "短片", "影片", "动效", "video", "animation"}

// ==================== 链① 单页 HTML 生成 ====================

// generateOnePageHTML 生成单页完整 HTML 并落库，返回组装后的完整 HTML。
//
// 完全复刻 GenerateRemainingPages 的单页生成链路，保证与批量生成产出一致：
//   匹配组件 → 构建 batch user prompt → 调 AI → 抽取内容区HTML → 后端拼接导航模板组装完整页 → 落库。
// 出错返回 error，由主编排记为该页 HTML 失败（该页不再进入配图流水线）。
func (s *CoursewareAutoAssemblyService) generateOnePageHTML(
	ctx context.Context, pc *cwAssemblyPageContext, page *models.CoursewarePage,
) (string, error) {
	// 1. 匹配组件（真实签名：matchComponentsForPage(ctx, page, subject, grade)）
	matched := s.genService.matchComponentsForPage(ctx, page, pc.cw.Subject, pc.cw.Grade)

	// 2. 构建批量模式 user prompt（真实签名 9 参：
	//    page, pageNum, totalPages, tplInfo, logoURL, orgName, matchedComps, cw, lessonContext）
	userPrompt := s.genService.buildBatchUserPrompt(
		page, page.PageNumber, pc.totalPages,
		pc.tplInfo, pc.logoURL, pc.orgName,
		matched, pc.cw, pc.lessonContext,
	)
	// 封面页(第1页)补封面提示（与 RegenerateSinglePage 对齐；批量提示词默认不含封面提示）
	if page.PageNumber == 1 {
		userPrompt = "⚠️ 这是封面页（第1页），请生成大标题居中的封面设计，突出课件标题、学科年级与机构品牌。\n\n" + userPrompt
	}

	// 3. 调 AI 生成内容区（真实签名 4 参：cfg, systemPrompt, userPrompt, traceCtx；schoolID 走 traceCtx）
	traceCtx := &ai.TraceContext{
		SceneCode: "courseware_generate",
		UserID:    &pc.userID,
		SchoolID:  schoolIDPtr(pc.schoolID),
	}
	result, err := ai.CallAI(pc.aiCfg, pc.genPrompt.Content, userPrompt, traceCtx)
	if err != nil {
		return "", fmt.Errorf("AI生成失败: %w", err)
	}
	if result == nil || strings.TrimSpace(result.Content) == "" {
		return "", fmt.Errorf("AI返回空内容")
	}

	// 4. 抽取内容区 HTML
	contentHTML := s.genService.extractHTMLFromAIOutput(result.Content)
	if strings.TrimSpace(contentHTML) == "" {
		return "", fmt.Errorf("抽取HTML为空")
	}

	// 5. 后端拼接导航模板组装完整页（真实签名 5 参：
	//    assembleFullPage(contentHTML, navTemplate, pageNum, totalPages, tplInfo)）
	fullHTML := s.genService.assembleFullPage(contentHTML, pc.navHTML, page.PageNumber, pc.totalPages, pc.tplInfo)

	// 6. 落库（真实签名 6 参：UpdateCWPageHTML(ctx, pageID, html, placeholderMap, matchedIDs, status)）
	matchedIDs := s.genService.buildMatchedComponentIDs(matched)
	if err := repository.UpdateCWPageHTML(ctx, page.ID, fullHTML, "", matchedIDs, models.CWPageStatusGenerated); err != nil {
		return "", fmt.Errorf("HTML落库失败: %w", err)
	}

	return fullHTML, nil
}

// ==================== 链② 单页配图 + 视频首帧占位 ====================

// assembleOnePageMedia 单页配图链 + 视频首帧占位链，返回结果快照供主编排汇总。全程 best-effort。
func (s *CoursewareAutoAssemblyService) assembleOnePageMedia(
	ctx context.Context, pc *cwAssemblyPageContext, page *models.CoursewarePage,
) cwAssemblyPageResult {
	res := cwAssemblyPageResult{
		pageNum: page.PageNumber,
		pageID:  page.ID,
		title:   page.Title,
		htmlOK:  true, // 能进本函数说明该页 HTML 已就绪
	}

	// 配图链
	s.assembleImage(ctx, pc, page, &res)

	// 视频首帧占位链：中间档 skipVideo=true 时所有页一律跳过；否则按关键词命中决定。
	if !pc.skipVideo && s.pageNeedsVideo(page) {
		s.assembleVideoPlaceholder(ctx, pc, page, &res)
	} else {
		res.videoSkipped = true
	}

	return res
}

// assembleImage 单页配图链（best-effort，结果写入 res）。
//
// 流程（对齐平台真实"统一微调融图"路径 + 封面直填分支 + 融图收尾后处理）：
//   1. SuggestImagePrompt 读本页 HTML 里的真实图片占位，产出配图提示词（无占位则空 → 跳过配图）
//   2. 取主图（items[0]），GenerateImage 生图（不传 RefImageURL → 内部自动带风格锚点风格档位）
//   3. 上云：UploadAssetToOSS(本地URL) 拿公网URL，UpdateCWAssetPublicURL 回写 public_oss_url
//   4. 融图，按页型分流：
//        · 封面页(page.PageNumber==1)：fillCoverImageIntoPlaceholder 占位直填，不调 RefinePage、不重排；
//          若封面无占位/定位失败，降级回 RefinePage（不失配图能力）。
//        · 非封面页：RefinePage 融图，AI 把图融进本页预留占位（内部已落库）。
//   5. 【非封面融图成功后】postProcessPageImages 收尾（A:URL校正 + B:空占位按需兜底）。
//
// 【配图价值守卫 v0.43.2】是否配图完全由本页 HTML 是否有真实图片占位决定（含封面）：
//   · 封面若在生成阶段预留了主视觉图占位（img-placeholder），会正常配图增吸引力；纯 SVG 装饰无占位则自动跳过。
//   · 纯文字/练习页在生成阶段就不留占位，SuggestImagePrompt 返回空数组、自动跳过（不再按页码一刀切）。
//   保留"空 MediaRequirements 跳过"作为省钱双保险：此类页方案本无配图需求，连生图都不必启动。
//   注意本函数无返回值，跳过用 return（非 return nil）。
func (s *CoursewareAutoAssemblyService) assembleImage(
	ctx context.Context, pc *cwAssemblyPageContext, page *models.CoursewarePage, res *cwAssemblyPageResult,
) {
	// 【配图价值守卫】无配图需求页直接跳过（省钱双保险；封面等有需求页照常走占位判断）
	// 注意：不再按 page.PageNumber==1 跳过封面——封面是否配图由其 HTML 是否有真实图片占位决定。
	if strings.TrimSpace(page.MediaRequirements) == "" {
		res.imageSkipped = true
		cwAssemblyLog.Info("跳过无配图需求页（media_requirements为空，避免多页雷同）", "page", page.PageNumber, "title", page.Title)
		return
	}

	GlobalCWSSEHub.Broadcast(pc.coursewareID, CWSSEEvent{
		EventType: "assembly_page_image",
		Data: map[string]interface{}{
			"page_number": page.PageNumber,
			"stage":       "image_prompt",
			"message":     fmt.Sprintf("第 %d 页：正在分析配图需求...", page.PageNumber),
		},
	})

	// 1. AI 写配图提示词（读该页 HTML 里真实图片占位驱动；无占位返回空数组）
	suggestions, err := s.assetService.SuggestImagePrompt(ctx, pc.coursewareID, page.PageNumber, pc.userID)
	if err != nil {
		res.imageOK = false
		res.errMsg = fmt.Sprintf("第%d页写配图提示词失败: %v", page.PageNumber, err)
		cwAssemblyLog.Warn("配图提示词失败", "page", page.PageNumber, "error", err)
		return
	}
	// 无占位 → 跳过配图（非失败）
	if len(suggestions) == 0 {
		res.imageSkipped = true
		return
	}

	// 2. 多图策略：只取主图（第1条）
	mainPrompt := strings.TrimSpace(suggestions[0].Prompt)
	if mainPrompt == "" {
		res.imageSkipped = true
		return
	}

	GlobalCWSSEHub.Broadcast(pc.coursewareID, CWSSEEvent{
		EventType: "assembly_page_image",
		Data: map[string]interface{}{
			"page_number": page.PageNumber,
			"stage":       "image_gen",
			"message":     fmt.Sprintf("第 %d 页：正在生成配图（自动带风格档位）...", page.PageNumber),
		},
	})

	// 【比例适配 v0.43.1】用 AI 为该图建议的尺寸；空则回退默认16:9横图。
	// 尺寸兜底闸门(client_image.go)保证即便偏小也会自动放大达标、不会 400。
	imgSize := strings.TrimSpace(suggestions[0].Size)
	if imgSize == "" {
		imgSize = "2560x1440"
	}

	// 3. 生图（不传 RefImageURL → GenerateImage 内部自动带风格档位；PlaceholderID 传空）
	imgResp, err := s.assetService.GenerateImage(ctx, &GenerateImageServiceRequest{
		CoursewareID:  pc.coursewareID,
		PageNumber:    page.PageNumber,
		PlaceholderID: "",
		Prompt:        mainPrompt,
		Size:          imgSize,
		UserID:        pc.userID,
	})
	if err != nil || imgResp == nil {
		res.imageOK = false
		res.errMsg = fmt.Sprintf("第%d页生图失败: %v", page.PageNumber, err)
		cwAssemblyLog.Warn("生图失败", "page", page.PageNumber, "error", err)
		return
	}

	// 4. 上云拿公网URL（UploadAssetToOSS 收本地URL字符串）；回写 public_oss_url。
	//    上云失败降级用本地URL（RefinePage/直填 仍能用，但分享稳定性降低）。
	publicURL, upErr := s.ossService.UploadAssetToOSS(imgResp.URL)
	if upErr != nil || strings.TrimSpace(publicURL) == "" {
		cwAssemblyLog.Warn("配图上云失败，降级用本地URL", "page", page.PageNumber, "asset", imgResp.AssetID, "error", upErr)
		publicURL = imgResp.URL
	} else {
		if wErr := repository.UpdateCWAssetPublicURL(ctx, imgResp.AssetID, publicURL); wErr != nil {
			cwAssemblyLog.Warn("配图公网URL回写失败(不阻断融图)", "page", page.PageNumber, "asset", imgResp.AssetID, "error", wErr)
		}
	}

	// 5. 融图，按页型分流：
	//    · 封面页：占位直填（不重排、不调 RefinePage，保住标题版式）；无占位则降级 RefinePage。
	//    · 非封面页：RefinePage 融图。
	if page.PageNumber == 1 {
		GlobalCWSSEHub.Broadcast(pc.coursewareID, CWSSEEvent{
			EventType: "assembly_page_image",
			Data: map[string]interface{}{
				"page_number": page.PageNumber,
				"stage":       "image_fuse",
				"message":     "封面：正在把主视觉图填入占位（保持标题版式不变）...",
			},
		})
		filled, ferr := s.fillCoverImageIntoPlaceholder(ctx, page, publicURL)
		if ferr == nil && filled {
			// 封面直填成功：标题版式纹丝不动，仅占位被填图。URL 由代码拼接不经 AI，无需后处理。
			res.imageOK = true
			cwAssemblyLog.Info("封面配图完成（占位直填，未重排）", "page", page.PageNumber, "asset", imgResp.AssetID, "size", imgSize, "url", publicURL)
			return
		}
		// 封面无占位可填 / 定位失败 → 降级走 RefinePage（保证不失配图能力）。
		cwAssemblyLog.Warn("封面占位直填未命中，降级走 RefinePage 融图", "page", page.PageNumber, "filled", filled, "error", ferr)
	}

	GlobalCWSSEHub.Broadcast(pc.coursewareID, CWSSEEvent{
		EventType: "assembly_page_image",
		Data: map[string]interface{}{
			"page_number": page.PageNumber,
			"stage":       "image_fuse",
			"message":     fmt.Sprintf("第 %d 页：正在把配图融入版面...", page.PageNumber),
		},
	})

	// 6. RefinePage 融图：把公网URL写进指令，让 AI 把图融进本页预留占位。
	//    imageDataURI 传空（该参数是"页面渲染截图多模态"，非配图用途）。
	fuseInstruction := s.buildImageFuseInstruction(publicURL)
	if _, rfErr := s.genService.RefinePage(ctx, pc.coursewareID, pc.userID, page.PageNumber, fuseInstruction, ""); rfErr != nil {
		res.imageOK = false
		res.errMsg = fmt.Sprintf("第%d页融图失败: %v", page.PageNumber, rfErr)
		cwAssemblyLog.Warn("融图失败", "page", page.PageNumber, "error", rfErr)
		return
	}

	// 6.5【导航栏复位】RefinePage 让 AI 重写整页融图时，可能改动/挪位/丢失顶部导航栏
	//    （position:absolute;top:0 的定位样式或 NAV_START/NAV_END 标记被 AI 改坏，
	//     表现为"导航栏没压住"）。此处用后端权威导航栏模板(pc.navHTML,与 assembleFullPage 同口径)
	//    把该页导航栏整体复位，确保与手动批量生成一致。仅装配路径生效，best-effort，失败不影响主流程。
	s.restoreAuthoritativeNav(ctx, pc, page)

	// 7. 【融图收尾后处理 A+B】RefinePage 已把图融入并落库，但 AI 可能写坏 URL / 漏填多余占位。
	//    postProcessPageImages 全程 best-effort：修 URL、按需填/删空占位，失败不影响 imageOK。
	s.postProcessPageImages(ctx, page, publicURL)

	res.imageOK = true
	cwAssemblyLog.Info("单页配图完成", "page", page.PageNumber, "asset", imgResp.AssetID, "size", imgSize, "url", publicURL)
}

// ==================== 导航栏复位（仅装配路径：融图后用权威导航栏压回）====================

// restoreAuthoritativeNav 用后端权威导航栏模板(pc.navHTML)把融图后可能被 AI 改坏的
// 顶部导航栏整体复位，落库。仅在全自动装配的非封面融图后调用，best-effort。
//
// 【为什么需要】RefinePage 是"AI 重写整页"融图。虽指令要求"不动导航栏"，但 AI 整页重写时
//   常改动导航栏的 position:absolute;top:0;height:80px;z-index 定位样式，或挪位、丢失
//   NAV_START/NAV_END 标记——表现为老师反馈的"自动生成课件，导航栏格式没压住"。
//   手动批量生成后不融图，导航栏一直是 assembleFullPage 硬拼的权威版本，故无此问题。
//
// 【权威导航栏口径】与 assembleFullPage 完全一致：pc.navHTML(即 cw.NavTemplateHTML) 内
//   {{PAGE_NUM}}→页号、{{TOTAL_PAGES}}→总页数。
//
// 【双策略定位页内导航栏区块】
//   策略A：页面含 <!-- NAV_START -->/<!-- NAV_END --> 标记 → 把标记区间整体替换为权威导航栏；
//   策略B：无标记(assembleFullPage 插入的 nav 本就不带标记，或 AI 弄丢标记) →
//         用 cwFindMatchingDivClose 定位"根 div 内第一个子 div"(= 导航栏在页面里的固定位置，
//         与 assembleFullPage 的插入位置一致)，整体替换为权威导航栏。
//   两策略均定位失败 → 原样返回，不写库(best-effort，绝不影响装配主流程)。
func (s *CoursewareAutoAssemblyService) restoreAuthoritativeNav(
	ctx context.Context, pc *cwAssemblyPageContext, page *models.CoursewarePage,
) {
	// 无导航栏模板则无从复位(理论上装配必有；防御性跳过)
	if strings.TrimSpace(pc.navHTML) == "" {
		return
	}

	// 重取 RefinePage 落库后的最新 HTML(内存 page.HTMLContent 是融图前的旧值)
	fresh, err := repository.GetCoursewarePageByNumber(ctx, page.CoursewareID, page.PageNumber)
	if err != nil || fresh == nil || strings.TrimSpace(fresh.HTMLContent) == "" {
		cwAssemblyLog.Warn("导航栏复位：重取页面HTML失败，跳过", "page", page.PageNumber, "error", err)
		return
	}
	orig := fresh.HTMLContent

	// 权威导航栏 = 模板 + 后端确定性追加页码（与 assembleFullPage 同口径，模板不含页码）
	nav := injectPageNumIntoNav(pc.navHTML, page.PageNumber, pc.totalPages)
	nav = strings.TrimSpace(nav)
	if nav == "" {
		return
	}

	newHTML, changed := replaceNavInPageHTML(orig, nav)
	if !changed || newHTML == orig {
		return // 定位失败或无变化，不写库
	}
	if e := repository.UpdateCWPageHTML(ctx, fresh.ID, newHTML, fresh.PlaceholderMap, fresh.MatchedComponentIDs, fresh.Status); e != nil {
		cwAssemblyLog.Warn("导航栏复位：写回HTML失败", "page", page.PageNumber, "error", e)
		return
	}
	cwAssemblyLog.Info("导航栏复位完成(权威导航栏压回)", "page", page.PageNumber)
}

// replaceNavInPageHTML 把页面 HTML 里的导航栏区块整体替换为权威导航栏 authNav。
// 返回新 HTML 与是否发生替换。双策略：NAV 标记优先，无标记退化到"根 div 第一个子 div"。
//
// 复用同包现成常量/函数：cwNavStartMarker/cwNavEndMarker(resync)、cwFindMatchingDivClose(本文件)。
func replaceNavInPageHTML(html string, authNav string) (string, bool) {
	// ---- 策略A：NAV_START/NAV_END 标记区间整体替换 ----
	start := strings.Index(html, cwNavStartMarker)
	if start >= 0 {
		end := strings.Index(html, cwNavEndMarker)
		if end > start {
			segStart := start + len(cwNavStartMarker)
			// 标记之间(含两侧换行归一)替换为权威导航栏
			return html[:segStart] + "\n" + authNav + "\n" + html[end:], true
		}
	}

	// ---- 策略B：无标记 → 根 div 内第一个子 div 即导航栏(与 assembleFullPage 插入位置一致) ----
	lower := strings.ToLower(html)
	// 1) 定位根 div 开标签结束位置
	rootOpen := strings.Index(lower, "<div")
	if rootOpen < 0 {
		return html, false
	}
	rootGT := strings.Index(html[rootOpen:], ">")
	if rootGT < 0 {
		return html, false
	}
	afterRootOpen := rootOpen + rootGT + 1
	// 2) 在根 div 内定位第一个子 div 的开标签起点
	childRel := strings.Index(lower[afterRootOpen:], "<div")
	if childRel < 0 {
		return html, false
	}
	childOpen := afterRootOpen + childRel
	childGT := strings.Index(html[childOpen:], ">")
	if childGT < 0 {
		return html, false
	}
	childInner := childOpen + childGT + 1
	// 3) 用本文件现成的 div 配对，找到第一个子 div(导航栏)的配对 </div>
	closeStart, ok := cwFindMatchingDivClose(html, lower, childInner)
	if !ok {
		return html, false
	}
	navEnd := closeStart + len("</div>")
	// 4) 用权威导航栏整体替换 [childOpen, navEnd)
	newHTML := html[:childOpen] + authNav + html[navEnd:]
	return newHTML, true
}

// ==================== 融图收尾后处理（A: URL 校正 + B: 空占位按需兜底）====================

// cwImgTagRe 匹配 HTML 里的 <img ... src="..." ...> 标签，用于逐个校正 src。
//   捕获组1 = <img 到 src=" 之前的部分；组2 = src 引号内的 URL；组3 = src 之后到 > 的部分。
//   只处理双引号 src（本平台生成的 <img> 一律双引号），单引号极罕见不处理，避免误伤。
var cwImgTagRe = regexp.MustCompile(`(?is)(<img\b[^>]*?\bsrc=")([^"]*)("[^>]*>)`)

// cwEmptyPlaceholderRe 匹配一个 img-placeholder 占位 div（class 含 img-placeholder）及其内容到配对 </div>。
//   (?is) 忽略大小写 + 让 . 跨行；[^>]* 容忍占位 div 上的 data-desc/style 等属性；
//   内部用 .*? 非贪婪吃到第一个 </div>（占位内部不嵌套 div，故第一个 </div> 即配对；
//   Go 的 RE2 引擎不支持负向先行断言 (?!，故用 .*? 而非 (?:(?!</div>).)*?）。
//   命中后由回调 fillOrRemoveEmptyPlaceholders 判定该块内部是否已含 <img>：含则跳过、不含才处理。
var cwEmptyPlaceholderRe = regexp.MustCompile(`(?is)<div\b[^>]*\bimg-placeholder\b[^>]*>.*?</div>`)

// postProcessPageImages 融图收尾后处理（best-effort，失败仅记日志不影响主流程）。
//
// 读该页 RefinePage 落库后的最新 HTML，依次做两件事后写回：
//   A. sanitizeImgSrcInHTML：修所有 <img src> 的坏 URL（补 "://"、去 src 内空格）；
//   B. fillOrRemoveEmptyPlaceholders：按本页是否有配图需求，补填或收起残留的空 img-placeholder。
// 若两步都未改动 HTML，则不落库（省一次写库）。
func (s *CoursewareAutoAssemblyService) postProcessPageImages(
	ctx context.Context, page *models.CoursewarePage, mainPublicURL string,
) {
	// RefinePage 已更新 page 的 HTML 并落库，但内存里的 page.HTMLContent 是旧的，需重新取最新。
	fresh, err := repository.GetCoursewarePageByNumber(ctx, page.CoursewareID, page.PageNumber)
	if err != nil || fresh == nil || strings.TrimSpace(fresh.HTMLContent) == "" {
		cwAssemblyLog.Warn("融图后处理：重取页面HTML失败，跳过后处理", "page", page.PageNumber, "error", err)
		return
	}
	orig := fresh.HTMLContent
	html := orig

	// A. URL 校正
	html = sanitizeImgSrcInHTML(html)

	// B. 空占位按需兜底：本页确有配图需求(MediaRequirements 非空)才补填主图；否则收起空占位。
	needImage := strings.TrimSpace(page.MediaRequirements) != ""
	html = fillOrRemoveEmptyPlaceholders(html, mainPublicURL, needImage)

	// 无变化则不写库
	if html == orig {
		return
	}
	if err := repository.UpdateCWPageHTML(ctx, fresh.ID, html, fresh.PlaceholderMap, fresh.MatchedComponentIDs, fresh.Status); err != nil {
		cwAssemblyLog.Warn("融图后处理：写回HTML失败", "page", page.PageNumber, "error", err)
		return
	}
	cwAssemblyLog.Info("融图后处理完成（URL校正+空占位兜底）", "page", page.PageNumber, "need_image", needImage)
}

// sanitizeImgSrcInHTML 纯语法修复 HTML 中所有 <img> 的 src（A 闸门）。
//
// 只做两件确定安全、不涉及业务语义的修复：
//   1. 为协议头补回丢失的 "://"：AI 转写偶发把 "https://x" 写成 "httpsx" / "https:/x" / "https//x"，
//      统一修正为 "https://x"（http 同理）。仅当 src 以 http/https 开头且协议分隔符残缺时才修，
//      已正确的 "https://" 不受影响。
//   2. 去掉 src 值内部的空格：AI 偶发在 URL 中间插空格（如 ".../de3 / 22010..."），一律删除。
//      URL 本身不允许裸空格，删除是安全的（真正需要空格的会被编码为 %20，不含裸空格）。
//   本地相对路径（/uploads/...）不带协议头，第1条不动它；第2条同样为其去空格（修 logo 被插空格的问题）。
func sanitizeImgSrcInHTML(html string) string {
	if strings.TrimSpace(html) == "" {
		return html
	}
	return cwImgTagRe.ReplaceAllStringFunc(html, func(tag string) string {
		m := cwImgTagRe.FindStringSubmatch(tag)
		if len(m) != 4 {
			return tag
		}
		pre, url, post := m[1], m[2], m[3]
		fixed := fixOneURL(url)
		if fixed == url {
			return tag
		}
		return pre + fixed + post
	})
}

// fixOneURL 修复单个 URL 字符串：补协议分隔符 "://" + 去内部空格。
func fixOneURL(url string) string {
	u := url
	// 1) 去掉 URL 内部所有空格（含被 AI 插入的 " / "）
	if strings.Contains(u, " ") {
		u = strings.ReplaceAll(u, " ", "")
	}
	// 2) 补协议分隔符：处理 https / http 开头但 "://" 残缺的情况。
	//    先把已正确的放过，再依次纠正 "https:/x" / "https//x" / "httpsx" 三种残缺形态。
	for _, scheme := range []string{"https", "http"} {
		if strings.HasPrefix(u, scheme+"://") {
			break // 已正确
		}
		if strings.HasPrefix(u, scheme) {
			rest := strings.TrimPrefix(u, scheme)
			// 剥掉紧跟的残缺分隔符：可能是 ":/" / "//" / ":" / 空
			rest = strings.TrimPrefix(rest, ":")
			rest = strings.TrimPrefix(rest, "/")
			rest = strings.TrimPrefix(rest, "/")
			u = scheme + "://" + rest
			break
		}
	}
	return u
}

// fillOrRemoveEmptyPlaceholders 处理残留的空 img-placeholder 占位（B 兜底）。
//
//   needImage=true （本页确有配图需求）：把每个空占位 div 的【内部】替换为撑满的 <img>（复用主图 URL），
//     保留占位 div 的 class/style 外壳（定位与尺寸），只填内容——胜过留一个 "🖼️..." 裂框。
//   needImage=false（本页无配图需求，占位多半是误判/装饰）：整个空占位 div 移除，不留裂框。
//
// 「空占位」判定：命中的占位块内部不含 <img>（含 <img> 说明已被融图填充，原样保留不动）。
// mainPublicURL 为空时（理论不会）退化为仅移除，避免填入空 src。
func fillOrRemoveEmptyPlaceholders(html string, mainPublicURL string, needImage bool) string {
	if strings.TrimSpace(html) == "" {
		return html
	}
	url := strings.TrimSpace(mainPublicURL)
	return cwEmptyPlaceholderRe.ReplaceAllStringFunc(html, func(divBlock string) string {
		// 内部已有 <img> → 已被融图填充，原样保留不动。
		if strings.Contains(strings.ToLower(divBlock), "<img") {
			return divBlock
		}
		// 无配图需求，或没有可用主图 → 移除整个空占位 div。
		if !needImage || url == "" {
			return ""
		}
		// 有配图需求 → 保留占位外壳，把内部替换为撑满的 <img>（复用主图）。
		// 取占位 div 的起始标签（从块首到第一个 '>'），保留其 class/style。
		gt := strings.Index(divBlock, ">")
		if gt < 0 {
			return divBlock
		}
		openTag := divBlock[:gt+1]
		img := fmt.Sprintf(
			`<img src="%s" alt="配图" style="width:100%%;height:100%%;object-fit:cover;display:block;border-radius:inherit;" />`,
			url,
		)
		return openTag + img + "</div>"
	})
}

// fillCoverImageIntoPlaceholder 封面「占位直填」：不重排、不调 AI，把生成好的图直接填进封面占位。
//
// 目的：封面是强排版页（大标题居中、品牌/学科年级布局精心），RefinePage 整页重写会冲乱标题版式。
// 本函数纯字符串处理，只改封面 HTML 里的图片占位【内部】，占位 div 的 class/style 外壳与其它一切原样保留，
// 从根上保证封面标题、布局纹丝不动。
//
// 返回：
//   - (true, nil)  ：找到占位并成功填图落库；
//   - (false, nil) ：封面 HTML 里没有 img-placeholder 占位（调用方据此降级走 RefinePage）；
//   - (false, err) ：定位/落库出错（调用方据此降级走 RefinePage）。
func (s *CoursewareAutoAssemblyService) fillCoverImageIntoPlaceholder(
	ctx context.Context, page *models.CoursewarePage, publicURL string,
) (bool, error) {
	html := page.HTMLContent
	if strings.TrimSpace(html) == "" {
		return false, fmt.Errorf("封面页无 HTML 内容")
	}
	publicURL = strings.TrimSpace(publicURL)
	if publicURL == "" {
		return false, fmt.Errorf("封面配图公网地址为空")
	}

	// 1) 定位第一个含 "img-placeholder" 的 <div ...> 起始标签。
	lower := strings.ToLower(html)
	clsIdx := strings.Index(lower, "img-placeholder")
	if clsIdx < 0 {
		return false, nil // 无占位，调用方降级 RefinePage
	}
	// 从 clsIdx 向前回溯最近的 "<div"
	divOpenStart := strings.LastIndex(lower[:clsIdx], "<div")
	if divOpenStart < 0 {
		return false, fmt.Errorf("占位 class 之前未找到 <div 起始标签")
	}
	// 起始标签的结束 '>'（从 divOpenStart 起找第一个 '>'）
	gt := strings.Index(html[divOpenStart:], ">")
	if gt < 0 {
		return false, fmt.Errorf("占位 <div 起始标签未闭合")
	}
	openTagEnd := divOpenStart + gt + 1 // 起始标签之后第一个字符位置（innerHTML 起点）

	// 2) 从 openTagEnd 起按 div 层级计数，找到与占位 <div> 配对的 </div>。
	closeStart, ok := cwFindMatchingDivClose(html, lower, openTagEnd)
	if !ok {
		return false, fmt.Errorf("占位 <div> 未找到配对的 </div>")
	}

	// 3) 组装：起始标签原样 + 撑满占位的 <img> + 配对 </div> 原样。占位 div 之外的一切都不动。
	openTag := html[divOpenStart:openTagEnd]      // 含 class/style 的占位起始标签，原样保留
	afterClose := html[closeStart+len("</div>"):] // 占位 </div> 之后的所有内容，原样保留
	before := html[:divOpenStart]                 // 占位 <div> 之前的所有内容，原样保留

	img := fmt.Sprintf(
		`<img src="%s" alt="封面配图" style="width:100%%;height:100%%;object-fit:cover;display:block;border-radius:inherit;" />`,
		publicURL,
	)

	newHTML := before + openTag + img + "</div>" + afterClose

	// 4) 落库（状态 generated，不动 placeholder_map / matched_component_ids）
	if err := repository.UpdateCWPageHTML(ctx, page.ID, newHTML, "", "", models.CWPageStatusGenerated); err != nil {
		return false, fmt.Errorf("封面直填后落库失败: %w", err)
	}
	return true, nil
}

// cwFindMatchingDivClose 从 innerStart（占位 <div> 起始标签之后的位置）起，
// 按 div 层级配对，返回占位 div 自身对应的 </div> 的起始下标。
func cwFindMatchingDivClose(html, lower string, innerStart int) (int, bool) {
	n := len(lower)
	i := innerStart
	depth := 0
	for i < n {
		openIdx := strings.Index(lower[i:], "<div")
		closeIdx := strings.Index(lower[i:], "</div>")

		// 没有更多 </div> —— 无法配对
		if closeIdx < 0 {
			return 0, false
		}
		closeAbs := i + closeIdx

		if openIdx >= 0 {
			openAbs := i + openIdx
			// 校验 <div 后紧跟空白或 '>'（排除 <divx 之类），排除误判
			isRealDivOpen := false
			after := openAbs + len("<div")
			if after < n {
				c := lower[after]
				if c == ' ' || c == '>' || c == '\t' || c == '\n' || c == '\r' || c == '/' {
					isRealDivOpen = true
				}
			}
			// 若下一个内部 <div 在下一个 </div> 之前，则先处理内部开标签，depth++
			if isRealDivOpen && openAbs < closeAbs {
				depth++
				i = openAbs + len("<div")
				continue
			}
		}

		// 处理这个 </div>
		if depth == 0 {
			return closeAbs, true // 占位 div 自身的闭合标签
		}
		depth--
		i = closeAbs + len("</div>")
	}
	return 0, false
}

// buildImageFuseInstruction 构造"把配图融入本页占位"的融图指令（喂给 RefinePage 的 instruction）。
// 贴合 RefinePage 的「最小改动」铁律：只把图放进已预留的图片占位，不动导航栏与其它内容。
// （仅非封面页使用；封面走占位直填不经此。）
//
// 【URL 完整性强调】RefinePage 让 AI 重写整页，AI 转写长 URL 易手滑（丢 "://"、插空格）。
// 虽有 postProcessPageImages 事后校正兜底，这里仍在指令中显式强调「原样一字不差复制 URL」，双保险降低出错率。
func (s *CoursewareAutoAssemblyService) buildImageFuseInstruction(publicURL string) string {
	var b strings.Builder
	b.WriteString("本页已生成一张与内容匹配的配图，图片公网地址为：\n")
	b.WriteString(publicURL)
	b.WriteString("\n\n请将这张图片融入本页内容区中已预留的图片占位位置：")
	b.WriteString("用 <img src=\"上述地址\"> 放入对应的图片占位容器/区域，并设置合适的 max-width/height 与 object-fit，")
	b.WriteString("使图片与版面自然协调、不溢出、不遮挡文字。")
	b.WriteString("【重要】src 里的图片地址必须与上面给出的地址【一字不差、原样复制】——包含 https:// 协议头，")
	b.WriteString("不得丢失冒号斜杠、不得在地址中间加空格或换行、不得改写任何字符。")
	b.WriteString("除把图片放入占位外，不要改动本页的导航栏、文字内容、布局与配色。")
	b.WriteString("若本页存在多处图片占位，仅将此图放入语义最匹配的一处即可。")
	return b.String()
}

// ==================== 链③ 视频首帧占位 ====================

// assembleVideoPlaceholder 单页视频首帧占位链（best-effort，结果写入 res）。
func (s *CoursewareAutoAssemblyService) assembleVideoPlaceholder(
	ctx context.Context, pc *cwAssemblyPageContext, page *models.CoursewarePage, res *cwAssemblyPageResult,
) {
	GlobalCWSSEHub.Broadcast(pc.coursewareID, CWSSEEvent{
		EventType: "assembly_page_video",
		Data: map[string]interface{}{
			"page_number": page.PageNumber,
			"stage":       "video_storyboard",
			"message":     fmt.Sprintf("第 %d 页：正在生成视频分镜与首帧占位...", page.PageNumber),
		},
	})

	// 1. AI 写视频分镜（多镜，每镜含首帧图提示词 ImagePrompt）
	shots, err := s.assetService.SuggestVideoPrompt(ctx, pc.coursewareID, page.PageNumber, pc.userID)
	if err != nil || len(shots) == 0 {
		res.videoOK = false
		cwAssemblyLog.Warn("视频分镜生成失败或为空，跳过视频占位", "page", page.PageNumber, "error", err)
		return
	}

	// 2. 取首镜的首帧图提示词
	framePrompt := strings.TrimSpace(shots[0].ImagePrompt)
	if framePrompt == "" {
		res.videoOK = false
		return
	}

	// 3. 生成首帧图（带风格档位；16:9 贴合视频画幅；不融入页面，作为视频占位资产）
	frameResp, err := s.assetService.GenerateImage(ctx, &GenerateImageServiceRequest{
		CoursewareID:  pc.coursewareID,
		PageNumber:    page.PageNumber,
		PlaceholderID: "",
		Prompt:        framePrompt,
		Size:          "2560x1440",
		UserID:        pc.userID,
	})
	if err != nil || frameResp == nil {
		res.videoOK = false
		cwAssemblyLog.Warn("首帧图生成失败", "page", page.PageNumber, "error", err)
		return
	}

	// 4. 首帧图上云（供后续图生视频 source_frame + 占位展示）。失败降级本地URL。
	framePublicURL, upErr := s.ossService.UploadAssetToOSS(frameResp.URL)
	if upErr != nil || strings.TrimSpace(framePublicURL) == "" {
		framePublicURL = frameResp.URL
	} else {
		if wErr := repository.UpdateCWAssetPublicURL(ctx, frameResp.AssetID, framePublicURL); wErr != nil {
			cwAssemblyLog.Warn("首帧图公网URL回写失败(不阻断)", "page", page.PageNumber, "asset", frameResp.AssetID, "error", wErr)
		}
	}

	// 5. 落库分镜数组（与手动 SuggestVideoPrompt 落库口径一致）
	storyboardsJSON := s.marshalStoryboards(shots)
	if storyboardsJSON != "" {
		if dbErr := repository.UpdatePageVideoStoryboards(ctx, pc.coursewareID, page.PageNumber, storyboardsJSON); dbErr != nil {
			cwAssemblyLog.Warn("分镜落库失败（首帧图仍可用）", "page", page.PageNumber, "error", dbErr)
		}
	}

	res.videoOK = true
	cwAssemblyLog.Info("视频首帧占位完成", "page", page.PageNumber, "frame_asset", frameResp.AssetID, "frame_url", framePublicURL)
}

// ==================== 辅助 ====================

// pageNeedsVideo 判定该页是否需要视频首帧占位（方案关键词命中即需要）。
func (s *CoursewareAutoAssemblyService) pageNeedsVideo(page *models.CoursewarePage) bool {
	haystack := strings.ToLower(strings.Join([]string{
		page.MediaRequirements,
		page.VisualFormat,
		page.InteractionType,
	}, " "))
	if strings.TrimSpace(haystack) == "" {
		return false
	}
	for _, kw := range cwAssemblyVideoKeywords {
		if strings.Contains(haystack, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// marshalStoryboards 把分镜数组序列化为 JSON 字符串（与手动 SuggestVideoPrompt 落库口径一致）。
func (s *CoursewareAutoAssemblyService) marshalStoryboards(shots []VideoStoryboardItem) string {
	if len(shots) == 0 {
		return ""
	}
	b, err := json.Marshal(shots)
	if err != nil {
		cwAssemblyLog.Warn("分镜JSON序列化失败", "error", err)
		return ""
	}
	return string(b)
}
