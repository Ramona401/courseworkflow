package services

// courseware_gen_refine.go — 课件导航栏微调+单页微调服务
//
// 本文件包含：
//   - RefineNav：导航栏AI微调（同步调用，返回修改后HTML）
//   - RefinePage：单页AI微调（同步调用，返回修改后完整页面HTML）
//
// 拆分自原 courseware_gen_service.go（v142 结构化日志迁移+模块化拆分）

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
	traceCtx := &ai.TraceContext{SceneCode: models.SceneCWNavRefine, UserID: &userID}
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

// ==================== P0-4: 单页AI微调 ====================

// RefinePage 单页AI微调：根据老师修改意见调整指定页面
// 同步调用，返回修改后的完整页面HTML
func (s *CoursewareGenService) RefinePage(ctx context.Context, coursewareID string, userID string, pageNum int, instruction string) (string, error) {
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

	// 3. 构建微调系统提示词
	systemPrompt := `你是课件页面微调助手。你会收到一页完整的课件HTML和老师的修改意见。

【绝对约束】
1. 只修改老师明确要求修改的部分
2. 不得修改导航栏（页面顶部80px区域，即<!-- NAV_START -->到<!-- NAV_END -->之间的内容）的任何内容
3. 不得修改老师未提到的布局、配色、字号、内容
4. 不得添加或删除页面主要结构块
5. 保持画布尺寸1920×1080不变，不得出现滚动条
6. 输出完整的修改后页面HTML（从<div style="width:1920px开始到</div>结束）

如果老师的要求模糊，选择最小改动方案。
直接输出修改后的完整HTML代码，不要输出任何解释文字。`

	// 4. 构建用户提示词
	userPrompt := fmt.Sprintf("## 当前页面HTML（第%d页：%s）\n```html\n%s\n```\n\n## 老师的修改意见\n%s\n\n请根据修改意见调整页面HTML，保持1920x1080画布不变，不修改导航栏。", pageNum, page.Title, page.HTMLContent, instruction)

	// 5. 调用AI
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), models.SceneCWPageRefine,
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		return "", fmt.Errorf("获取AI配置失败: %w", err)
	}
	traceCtx := &ai.TraceContext{SceneCode: models.SceneCWPageRefine, UserID: &userID}
	result, aiErr := ai.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if aiErr != nil {
		return "", fmt.Errorf("AI微调失败: %w", aiErr)
	}

	// 6. 提取HTML
	refined := s.extractHTMLFromAIOutput(result.Content)
	if refined == "" {
		return "", fmt.Errorf("AI输出未包含有效HTML")
	}

	// 7. 保存微调结果
	if dbErr := repository.UpdateCWPageHTML(ctx, page.ID, refined, "", page.MatchedComponentIDs, models.CWPageStatusGenerated); dbErr != nil {
		return "", fmt.Errorf("保存微调结果失败: %w", dbErr)
	}

	cwGenLog.Info("单页微调完成", "courseware_id", coursewareID, "page_num", pageNum, "instruction", instruction)
	return refined, nil
}
