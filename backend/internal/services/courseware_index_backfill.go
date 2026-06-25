package services

// courseware_index_backfill.go — doc/ppt 直翻路径补脉络（前台）与补索引（后台异步）
//
// 从 courseware_index_service.go 拆出（超 600 行红线拆分）。
// 与主文件同 package 同 *CoursewareIndexService 接收器，复用主文件的
// splitOverviewAndPages/parseAOCIIndexOutput 及包级 cwClamp/cwStripCodeFences，
// 调用方（courseware_ppt_service / courseware_doc_service）零改动。
//
// v0.44 新增（直翻路径补脉络与索引，体验Y+方案）：
//   - GenerateOverviewFromPages：方案出页后用 haiku 快速生成"哪几页干什么"脉络（前台，几秒）
//   - BackfillPageIndexAsync：后台异步对照"原文+当前方案"为每页生成 AOCI 索引并回填
//
// v0.44.1 防贴错双保险：
//   A. 页数守卫——记录喂AI页数，回填前重查当前页，页数不一致则整体放弃本次回填
//   B. 标题锚点匹配——用 AI 回显的页标题(TT) 与当前页标题匹配定位 page_number，匹配不上跳过

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// GenerateOverviewFromPages 根据已生成的页面方案，用 haiku 快速生成"哪几页干什么"的脉络概述
//
// 用途：doc/ppt 直翻路径出页后，前台立即调用此方法补一段真脉络（替代套话overview）。
// 输入小、模型便宜（scanner/haiku）、几秒返回。
// 返回脉络字符串；失败或为空时返回空串，调用方应退回套话overview，不阻塞主流程。
func (s *CoursewareIndexService) GenerateOverviewFromPages(
	ctx context.Context, userID string,
	title string, subject string, grade string,
	pages []*models.CoursewarePage,
) string {
	if len(pages) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("你是课件结构分析专家。下面是一份课件的逐页方案，请你用一段连贯的中文概括这份课件的整体脉络。\n\n")
	sb.WriteString(fmt.Sprintf("课件标题：%s\n学科：%s\n年级：%s\n总页数：%d\n\n", title, subject, grade, len(pages)))
	sb.WriteString("## 逐页方案\n")
	for _, p := range pages {
		sb.WriteString(fmt.Sprintf("第%d页：%s —— %s\n", p.PageNumber, p.Title, p.Purpose))
	}
	sb.WriteString("\n## 输出要求\n")
	sb.WriteString("1. 用80-150字概括，说明这份课件分几个部分、哪几页讲什么，例如\"第1-3页为情境导入，第4-8页讲解核心概念，第9-12页为分组练习，第13页课堂小结\"。\n")
	sb.WriteString("2. 按页码顺序归并相邻的同类页面，体现教学的递进逻辑。\n")
	sb.WriteString("3. 只输出这段脉络文字，不要任何标题、前缀、markdown 或额外说明。\n")

	systemPrompt := "你是课件结构分析专家，擅长用简洁连贯的语言概括课件的教学脉络。"

	// 用 scanner 场景（haiku，便宜快）
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "scanner",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		log.Printf("[courseware_index] 生成脉络-获取AI配置失败，退回套话: %v", err)
		return ""
	}

	// v198：解析操作者所属学校ID，供模型境内/境外分流判定
	ovSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceCtx := &ai.TraceContext{SceneCode: "scanner", UserID: &userID, SchoolID: schoolIDPtr(ovSchoolID)}
	callResult, err := ai.CallAI(aiCfg, systemPrompt, sb.String(), traceCtx)
	if err != nil {
		log.Printf("[courseware_index] 生成脉络-AI调用失败，退回套话: %v", err)
		return ""
	}

	overview := strings.TrimSpace(cwStripCodeFences(callResult.Content))
	// 防御：若AI输出异常长（跑题），截断到合理长度
	if len([]rune(overview)) > 400 {
		overview = string([]rune(overview)[:400])
	}
	return overview
}

// BackfillPageIndexAsync 后台异步：对照"教案原文 + 当前页面方案"，为每页生成 AOCI 索引并回填
//
// 用途：doc/ppt 直翻路径下，页面创建时 page_index 及 CG/IL/VF 索引列为空。
// 本方法在方案保存后由调用方以 go func 异步触发，用 haiku 整批对照原文+方案逐页编码并回填。
//
// 设计要点：
//   - 独立 context.Background()，不受原请求 ctx 取消影响
//   - 回填前重新 ListCoursewarePages 拿"当前"页，对齐老师可能的方案改动
//   - 复用层1解析 parseAOCIIndexOutput + 编码映射
//   - 全程失败不 panic、不影响课件可用，仅记日志
//   - 调用 repository.UpdateCWPageIndexFields 只更新索引列，不碰方案/HTML
//
// v0.44.1 防贴错双保险（A页数守卫 + B标题锚点匹配），详见文件头。
//
// 参数 rawText：教案/文档原文（doc 传 docx 全文，ppt 传各页文本拼接）
func (s *CoursewareIndexService) BackfillPageIndexAsync(
	coursewareID string, userID string,
	title string, subject string, grade string, rawText string,
) {
	// 独立后台上下文（不随请求取消而中断）
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[courseware_index] 后台补索引 panic 已恢复: cw=%s r=%v", coursewareID, r)
		}
	}()

	// ---- 1. 取当前页面（构建提示词所依据的快照）----
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(pages) == 0 {
		log.Printf("[courseware_index] 后台补索引-取当前页面失败或为空: cw=%s err=%v", coursewareID, err)
		return
	}
	promptPageCount := len(pages) // 保险A：记录喂给AI的页数

	// ---- 2. 加载层1提示词（AOCI索引字典）----
	dictPrompt, err := repository.GetCurrentPromptByKey("prompt_courseware_index")
	if err != nil {
		log.Printf("[courseware_index] 后台补索引-加载索引字典失败: cw=%s err=%v", coursewareID, err)
		return
	}

	// ---- 3. 构建"对照原文+方案"的用户提示词 ----
	var sb strings.Builder
	sb.WriteString("请为下面这份课件的每一页生成AOCI压缩索引（仅输出PAGE索引，不需要OVERVIEW概述）。\n")
	sb.WriteString("索引必须对照【教案/文档原文】与【逐页方案】两者：方案告诉你每页讲什么，原文帮你准确判断知识点、认知层次、能力目标。\n\n")
	sb.WriteString(fmt.Sprintf("## 课件基本信息\n- 标题：%s\n- 学科：%s\n- 年级：%s\n- 总页数：%d\n\n", title, subject, grade, len(pages)))

	// 原文（截断避免超上下文）
	srcText := rawText
	if len([]rune(srcText)) > 20000 {
		srcText = string([]rune(srcText)[:20000]) + "\n\n[原文过长，已截取前20000字]"
	}
	sb.WriteString("## 教案/文档原文\n\n")
	sb.WriteString(srcText)
	sb.WriteString("\n\n## 逐页方案（共" + strconv.Itoa(len(pages)) + "页，请严格按此页数与顺序逐页输出索引）\n")
	for _, p := range pages {
		sb.WriteString(fmt.Sprintf("第%d页｜标题：%s｜目的：%s｜概要：%s｜交互：%s｜视觉：%s\n",
			p.PageNumber, p.Title, p.Purpose, p.ContentSummary, p.InteractionType, p.VisualFormat))
	}

	sb.WriteString("\n## 输出要求\n")
	sb.WriteString("严格为上面每一页输出一个AOCI索引块，顺序、页数与上面的方案完全一致。\n")
	sb.WriteString("特别注意：每个索引块的 PAGE 行必须原样回显该页的标题（TT字段），以便系统按标题对齐回填。\n")
	sb.WriteString("每块格式如下（PAGE行 + 编码行 + 语义行）：\n")
	sb.WriteString("PAGE:页码|TT:页面标题（与上面方案该页标题保持一致）\n")
	sb.WriteString("KT:知识类型|CG:认知层次(1-6)|IL:交互复杂度(1-5)|VF:视觉形式编码|TG:类型标记\n")
	sb.WriteString("[K]知识目标（这一页要让学生掌握的核心知识点，依据原文准确归纳）\n")
	sb.WriteString("[A]能力目标（这一页训练的能力）\n")
	sb.WriteString("[I]交互说明（这一页的交互/活动形式）\n")
	sb.WriteString("[C]内容要点（这一页承载的具体内容）\n\n")
	sb.WriteString("编码取值说明：\n")
	sb.WriteString("- CG认知层次：1记忆 2理解 3应用 4分析 5评价 6创造\n")
	sb.WriteString("- IL交互复杂度：1静态展示 2点击 3输入 4拖拽 5游戏\n")
	sb.WriteString("- VF视觉形式编码：TH纯文字 IT图文 DG图示 CT图表 TL时间线 CP对比 GL画廊 FM全屏媒体\n\n")
	sb.WriteString("只输出这些PAGE索引块，不要OVERVIEW、不要JSON、不要任何额外说明文字。")

	// ---- 4. 调用 AI（scanner场景，haiku，整批）----
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "scanner",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		log.Printf("[courseware_index] 后台补索引-获取AI配置失败: cw=%s err=%v", coursewareID, err)
		return
	}

	// v198：解析操作者所属学校ID，供模型境内/境外分流判定
	bfSchoolID, _ := repository.GetSchoolIDByUserID(ctx, userID)
	traceCtx := &ai.TraceContext{SceneCode: "scanner", UserID: &userID, SchoolID: schoolIDPtr(bfSchoolID)}
	callResult, err := ai.CallAI(aiCfg, dictPrompt.Content, sb.String(), traceCtx)
	if err != nil {
		log.Printf("[courseware_index] 后台补索引-AI调用失败: cw=%s err=%v", coursewareID, err)
		return
	}

	// ---- 5. 解析 AOCI 索引输出（splitOverviewAndPages 兼容"无OVERVIEW、直接PAGE"）----
	_, pageText := s.splitOverviewAndPages(callResult.Content)
	rawPages, err := s.parseAOCIIndexOutput(pageText)
	if err != nil || len(rawPages) == 0 {
		log.Printf("[courseware_index] 后台补索引-解析索引失败: cw=%s err=%v", coursewareID, err)
		return
	}

	// ---- 6. 保险A：页数守卫——回填前重查当前页，页数变了就整体放弃 ----
	curPages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(curPages) == 0 {
		log.Printf("[courseware_index] 后台补索引-回填前重查页面失败或为空: cw=%s err=%v", coursewareID, err)
		return
	}
	if len(curPages) != promptPageCount {
		log.Printf("[courseware_index] 后台补索引-页数已变化(喂AI时=%d 当前=%d)，疑似方案被修改，放弃本次回填，留待夜间轮询: cw=%s",
			promptPageCount, len(curPages), coursewareID)
		return
	}

	// ---- 7. 保险B：按标题锚点匹配回填 ----
	titleMap := make(map[string]*models.CoursewarePage)
	for _, p := range curPages {
		key := cwNormalizeTitle(p.Title)
		if key != "" {
			if _, exists := titleMap[key]; !exists {
				titleMap[key] = p
			}
		}
	}

	filled := 0
	matchedPageNums := make(map[int]bool) // 防止两个解析块匹配到同一页
	for _, rp := range rawPages {
		rpTitleKey := cwNormalizeTitle(rp.Title)
		if rpTitleKey == "" {
			continue // AI未回显标题，无法锚定，跳过
		}

		// 先精确匹配
		target, ok := titleMap[rpTitleKey]
		if !ok {
			// 再模糊匹配：互相包含（AI可能轻微改写标题）
			for k, p := range titleMap {
				if matchedPageNums[p.PageNumber] {
					continue
				}
				if strings.Contains(k, rpTitleKey) || strings.Contains(rpTitleKey, k) {
					target = p
					ok = true
					break
				}
			}
		}
		if !ok || target == nil {
			continue // 匹配不上，宁可留空也不贴错
		}
		if matchedPageNums[target.PageNumber] {
			continue // 该页已被其它块匹配过，跳过
		}

		cg := cwClamp(rp.CG, 1, 6)
		il := cwClamp(rp.IL, 1, 5)
		vf := cwNormalizeVF(rp.VF)
		if err := repository.UpdateCWPageIndexFields(ctx, coursewareID, target.PageNumber, rp.RawIndex, cg, il, vf); err != nil {
			log.Printf("[courseware_index] 后台补索引-回填第%d页失败: cw=%s err=%v", target.PageNumber, coursewareID, err)
			continue
		}
		matchedPageNums[target.PageNumber] = true
		filled++
	}

	log.Printf("[courseware_index] 后台补索引完成: cw=%s 当前页=%d 解析索引=%d 标题匹配回填=%d model=%s tokens=%d",
		coursewareID, len(curPages), len(rawPages), filled, callResult.ModelUsed, callResult.TokensUsed)
}

// cwNormalizeTitle 规整页标题用于锚点匹配：去首尾空白 + 去全部内部空白
func cwNormalizeTitle(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range t {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\u3000' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// cwNormalizeVF 规整视觉形式编码：非法/空值归一为 TH（纯文字），仅接受 8 个合法编码
func cwNormalizeVF(vf string) string {
	v := strings.ToUpper(strings.TrimSpace(vf))
	switch v {
	case "TH", "IT", "DG", "CT", "TL", "CP", "GL", "FM":
		return v
	default:
		return "TH"
	}
}
