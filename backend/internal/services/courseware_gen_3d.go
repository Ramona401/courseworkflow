package services

// courseware_gen_3d.go — 3D互动单页课件AI生成服务
//
// v0.42.11 新增：
//   - Generate3DSinglePage：一次性生成完整的3D互动HTML单页（60-70KB级别）
//   - 使用 Three.js + OrbitControls 技术栈
//   - 跳过标准课件的索引→风格→导航栏六步流程，直接生成
//   - 生成完成后状态从 generating → preview
//
// 调用链：
//   handler.Generate3DPage → genService.Generate3DSinglePage → ai.CallAI → repository.UpdateCWPageHTML

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 3D单页课件生成 ====================

// Generate3DSinglePage 一次性生成完整的3D互动HTML单页
// 前置条件：课件 source_type='3d_single'，status='generating'
// 流程：加载提示词 → 构建用户提示词 → 调用AI → 提取HTML → 写入page_1 → 状态改preview
func (s *CoursewareGenService) Generate3DSinglePage(ctx context.Context, coursewareID string, userID string) error {
	startTime := time.Now()

	// ---- 1. 获取课件信息并校验 ----
	cw, err := repository.GetCoursewareByID(ctx, coursewareID)
	if err != nil {
		s.broadcastError(coursewareID, "课件不存在: "+err.Error())
		return fmt.Errorf("课件不存在: %w", err)
	}
	if cw.UserID != userID {
		s.broadcastError(coursewareID, "无权操作此课件")
		return fmt.Errorf("无权操作此课件")
	}
	if cw.SourceType != models.CWSource3DSingle {
		s.broadcastError(coursewareID, "此课件不是3D单页类型")
		return fmt.Errorf("课件来源类型不是3d_single: %s", cw.SourceType)
	}
	if cw.Status != models.CoursewareStatusGenerating {
		s.broadcastError(coursewareID, "当前状态不允许生成: "+cw.Status)
		return fmt.Errorf("当前状态不允许生成: %s", cw.Status)
	}

	// ---- 2. 获取页面信息（应该只有1页） ----
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(pages) == 0 {
		s.broadcastError(coursewareID, "课件没有页面记录")
		return fmt.Errorf("课件页面为空")
	}
	page := pages[0] // 3D单页只有第1页

	// ---- 3. 加载3D单页生成提示词 ----
	genPrompt, err := repository.GetCurrentPromptByKey("prompt_courseware_3d_single")
	if err != nil {
		s.broadcastError(coursewareID, "加载3D生成提示词失败: "+err.Error())
		return fmt.Errorf("加载3D生成提示词失败: %w", err)
	}

	// ---- 4. 获取AI配置（使用 courseware_3d_single 场景） ----
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "courseware_3d_single",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		s.broadcastError(coursewareID, "获取AI配置失败: "+err.Error())
		return fmt.Errorf("获取AI配置失败: %w", err)
	}

	// ---- 5. 广播开始事件 ----
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenStart,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"total_pages":  1,
			"template":     "3D互动单页",
			"message":      "正在生成3D互动单页，这可能需要1-3分钟，请耐心等待...",
			"is_preview":   false,
			"is_3d_single": true,
		},
	})

	// ---- 6. 构建用户提示词 ----
	userPrompt := s.build3DSingleUserPrompt(cw, page)

	cwGenLog.Info("开始生成3D单页",
		"courseware_id", coursewareID,
		"title", cw.Title,
		"subject", cw.Subject,
		"grade", cw.Grade,
	)

	// 广播进度
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenProgress,
		Data: map[string]interface{}{
			"current_page": 1,
			"total_pages":  1,
			"page_title":   cw.Title,
			"message":      "AI正在构建3D场景、粒子系统和交互逻辑...",
		},
	})

	// ---- 7. 调用AI生成（单次调用，输出完整3D HTML） ----
	traceCtx := &ai.TraceContext{SceneCode: "courseware_3d_single", UserID: &userID}
	result, aiErr := ai.CallAI(aiCfg, genPrompt.Content, userPrompt, traceCtx)
	if aiErr != nil {
		errMsg := fmt.Sprintf("3D单页AI生成失败: %v", aiErr)
		cwGenLog.Error("3D单页AI生成失败", "error", aiErr, "courseware_id", coursewareID)
		s.broadcastError(coursewareID, errMsg)
		return fmt.Errorf(errMsg)
	}

	// ---- 8. 提取完整HTML ----
	htmlContent := s.extract3DHTMLFromAIOutput(result.Content)
	if htmlContent == "" {
		errMsg := "3D单页AI输出未包含有效HTML"
		cwGenLog.Warn(errMsg, "courseware_id", coursewareID, "output_len", len(result.Content))
		s.broadcastError(coursewareID, errMsg)
		return fmt.Errorf(errMsg)
	}

	cwGenLog.Info("3D单页HTML提取成功",
		"courseware_id", coursewareID,
		"html_len", len(htmlContent),
		"model", result.ModelUsed,
		"tokens", result.TokensUsed,
	)

	// ---- 9. 写入数据库（第1页） ----
	if dbErr := repository.UpdateCWPageHTML(ctx, page.ID, htmlContent, "", "", models.CWPageStatusGenerated); dbErr != nil {
		errMsg := fmt.Sprintf("3D单页保存HTML失败: %v", dbErr)
		cwGenLog.Error(errMsg, "courseware_id", coursewareID)
		s.broadcastError(coursewareID, errMsg)
		return fmt.Errorf(errMsg)
	}

	// ---- 10. 更新课件状态 generating → preview ----
	if err := repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusPreview); err != nil {
		cwGenLog.Warn("更新课件状态失败", "error", err, "courseware_id", coursewareID)
	}

	// ---- 11. 广播完成 ----
	elapsed := time.Since(startTime)
	cwGenLog.Info("3D单页生成完成",
		"courseware_id", coursewareID,
		"html_len", len(htmlContent),
		"model", result.ModelUsed,
		"tokens", result.TokensUsed,
		"elapsed_ms", elapsed.Milliseconds(),
	)

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenPage,
		Data: map[string]interface{}{
			"page_number":  1,
			"page_id":      page.ID,
			"title":        cw.Title,
			"html_content": htmlContent,
			"model_used":   result.ModelUsed,
			"tokens_used":  result.TokensUsed,
		},
	})

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenDone,
		Data: map[string]interface{}{
			"courseware_id":  coursewareID,
			"success_count": 1,
			"fail_count":    0,
			"total_pages":   1,
			"elapsed_ms":    elapsed.Milliseconds(),
			"is_preview":    false,
			"is_3d_single":  true,
			"message":       fmt.Sprintf("3D互动单页生成完成！耗时 %.1f 秒", elapsed.Seconds()),
		},
	})

	return nil
}

// ==================== 3D单页用户提示词构建 ====================

// build3DSingleUserPrompt 构建3D单页的AI用户提示词
// 包含课件主题信息、学科年级、描述等
func (s *CoursewareGenService) build3DSingleUserPrompt(cw *models.Courseware, page *models.CoursewarePage) string {
	var sb strings.Builder

	sb.WriteString("## 3D互动单页课件需求\n\n")
	sb.WriteString(fmt.Sprintf("- 课件标题：%s\n", cw.Title))
	sb.WriteString(fmt.Sprintf("- 学科：%s\n", cw.Subject))
	sb.WriteString(fmt.Sprintf("- 年级：%s\n", cw.Grade))
	sb.WriteString("\n")

	// 页面方案信息（如果有）
	if page.Purpose != "" {
		sb.WriteString(fmt.Sprintf("- 教学目的：%s\n", page.Purpose))
	}
	if page.ContentSummary != "" {
		sb.WriteString(fmt.Sprintf("- 内容概要：%s\n", page.ContentSummary))
	}
	if page.MediaRequirements != "" {
		sb.WriteString(fmt.Sprintf("- 额外说明：%s\n", page.MediaRequirements))
	}
	sb.WriteString("\n")

	sb.WriteString("请根据以上主题信息，生成一个完整的3D互动单页课件HTML。\n")
	sb.WriteString("严格遵守系统提示词中的所有技术规范和质量要求。\n")
	sb.WriteString("输出完整的HTML文件（从<!doctype html>到</html>），不要有任何省略。\n")

	return sb.String()
}

// ==================== 3D HTML提取 ====================

// extract3DHTMLFromAIOutput 从AI输出中提取完整的3D HTML文件
// 3D单页输出是完整的HTML文件（<!doctype html>...到...</html>）
// 与标准课件不同，不是<div>片段而是完整文档
func (s *CoursewareGenService) extract3DHTMLFromAIOutput(aiOutput string) string {
	text := strings.TrimSpace(aiOutput)
	if text == "" {
		return ""
	}

	// 去除markdown代码块标记
	text = cwGenStripCodeFences(text)

	// 查找 <!doctype html> 或 <!DOCTYPE html> 开始
	lowerText := strings.ToLower(text)
	docStart := strings.Index(lowerText, "<!doctype html>")
	if docStart < 0 {
		docStart = strings.Index(lowerText, "<!doctype html")
		if docStart < 0 {
			// 尝试找 <html
			docStart = strings.Index(lowerText, "<html")
		}
	}
	if docStart < 0 {
		cwGenLog.Warn("3D输出中未找到<!doctype html>或<html>标记")
		return ""
	}

	// 查找 </html> 结束
	htmlEnd := strings.LastIndex(lowerText, "</html>")
	if htmlEnd < 0 {
		cwGenLog.Warn("3D输出中未找到</html>闭合标记")
		// 如果没有闭合标签，取到末尾
		return strings.TrimSpace(text[docStart:])
	}

	// 提取从 doctype 到 </html> 的完整内容
	result := text[docStart : htmlEnd+len("</html>")]
	return strings.TrimSpace(result)
}
