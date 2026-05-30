package services

// courseware_gen_service.go — 课件HTML逐页AI生成服务（核心生成逻辑）
//
// 本文件包含：
//   - CoursewareGenService 结构体定义和构造函数
//   - 风格配置解析结构体
//   - GeneratePreviewPages：生成预览页（仅封面P1）
//   - GenerateRemainingPages：生成剩余页面（后端硬拼接导航栏）
//   - CancelGenerate：中途中断生成
//   - broadcastError：SSE错误广播
//
// 拆分自原 courseware_gen_service.go（v142 结构化日志迁移+模块化拆分）
// 关联文件：
//   - courseware_gen_refine.go：导航栏微调+单页微调
//   - courseware_gen_helpers.go：提示词构建+HTML提取+风格解析+组件匹配+导航栏提取

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"tedna/internal/ai"
	"tedna/internal/config"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// cwGenLog 课件HTML生成模块级结构化日志器
var cwGenLog = logger.WithModule("cw_gen")

// ==================== 课件HTML生成服务 ====================

// CoursewareGenService 课件HTML逐页AI生成服务
type CoursewareGenService struct {
	cfg *config.Config
}

// NewCoursewareGenService 创建课件HTML生成服务
func NewCoursewareGenService(cfg *config.Config) *CoursewareGenService {
	return &CoursewareGenService{cfg: cfg}
}

// ==================== 风格配置解析结构 ====================

// cwStyleConfig 从课件style_config JSON中解析的风格配置
type cwStyleConfig struct {
	TemplateID         string `json:"template_id"`
	LogoURL            string `json:"logo_url"`
	OrgName            string `json:"org_name"`
	CustomPrimaryColor string `json:"custom_primary_color"`
}

// cwTemplateInfo 模板的关键信息（用于注入AI提示词）
type cwTemplateInfo struct {
	Name          string            // 模板名称
	StyleCategory string            // 风格类别
	CSSVariables  map[string]string // CSS变量键值对
	ColorScheme   map[string]string // 配色方案键值对
}

// ==================== Step 1: 生成预览页（仅封面P1） ====================

// GeneratePreviewPages P0-1改造：仅生成封面页(P1)，让老师确认导航栏样式
// AI生成时用 <!-- NAV_START --> / <!-- NAV_END --> 标记包裹导航栏
// 生成完成后不改变课件状态（仍为generating），等老师确认导航栏
func (s *CoursewareGenService) GeneratePreviewPages(ctx context.Context, coursewareID string, userID string) error {
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
	if cw.Status != models.CoursewareStatusGenerating {
		s.broadcastError(coursewareID, "当前状态不允许生成预览: "+cw.Status)
		return fmt.Errorf("当前状态不允许生成预览: %s", cw.Status)
	}

	// ---- 2. 获取全部页面方案 ----
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(pages) == 0 {
		s.broadcastError(coursewareID, "课件没有页面方案，请先生成索引")
		return fmt.Errorf("课件页面为空")
	}

	// P0-1：只取第1页（封面页）
	previewCount := 1
	previewPages := pages[:previewCount]

	// ---- 3. 解析风格配置 + 加载模板 ----
	styleCfg := s.parseStyleConfig(cw.StyleConfig)
	tplInfo, err := s.loadTemplateInfo(ctx, styleCfg.TemplateID)
	if err != nil {
		cwGenLog.Warn("加载模板失败，使用默认风格", "error", err, "courseware_id", coursewareID)
		tplInfo = s.defaultTemplateInfo()
	}

	logoURL, orgName := s.resolveLogoAndOrg(ctx, cw, styleCfg)

	// ---- 4. 加载生成提示词 ----
	genPrompt, err := repository.GetCurrentPromptByKey("prompt_courseware_generate")
	if err != nil {
		s.broadcastError(coursewareID, "加载生成提示词失败: "+err.Error())
		return fmt.Errorf("加载生成提示词失败: %w", err)
	}

	// ---- 5. 获取AI配置 ----
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "courseware_generate",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		s.broadcastError(coursewareID, "获取AI配置失败: "+err.Error())
		return fmt.Errorf("获取AI配置失败: %w", err)
	}

	totalPages := len(pages)

	// ---- 6. 广播开始事件 ----
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenStart,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"total_pages":  previewCount,
			"template":     tplInfo.Name,
			"message":      "正在生成封面预览页，请稍候...",
			"is_preview":   true,
		},
	})

	// ---- 7. 生成封面页 ----
	successCount := 0
	failCount := 0
	var errors []string

	for i, page := range previewPages {
		pageNum := i + 1
		cwGenLog.Info("生成预览页", "page_num", pageNum, "title", page.Title, "courseware_id", coursewareID)

		// 广播进度
		GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
			EventType: CWSSEGenProgress,
			Data: map[string]interface{}{
				"current_page": pageNum,
				"total_pages":  previewCount,
				"page_title":   page.Title,
				"message":      fmt.Sprintf("正在生成封面预览页：%s", page.Title),
			},
		})

		// 匹配组件
		matchedComps := s.matchComponentsForPage(ctx, page, cw.Subject, cw.Grade)

		// 构建用户提示词（预览模式：AI自由生成导航栏，用标记包裹）
		userPrompt := s.buildPreviewUserPrompt(page, pageNum, totalPages, tplInfo, logoURL, orgName, matchedComps, cw)

		// 调用AI生成
		traceCtx := &ai.TraceContext{SceneCode: "courseware_generate", UserID: &userID}
		result, aiErr := ai.CallAI(aiCfg, genPrompt.Content, userPrompt, traceCtx)
		if aiErr != nil {
			errMsg := fmt.Sprintf("封面预览AI生成失败: %v", aiErr)
			cwGenLog.Error("封面预览AI生成失败", "error", aiErr, "courseware_id", coursewareID, "page_num", pageNum)
			errors = append(errors, errMsg)
			failCount++
			GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
				EventType: CWSSEGenProgress,
				Data: map[string]interface{}{
					"current_page": pageNum,
					"total_pages":  previewCount,
					"page_title":   page.Title,
					"error":        errMsg,
					"message":      "⚠️ 封面预览生成失败",
				},
			})
			continue
		}

		// 提取HTML
		htmlContent := s.extractHTMLFromAIOutput(result.Content)
		if htmlContent == "" {
			errMsg := "封面预览AI输出未包含有效HTML"
			cwGenLog.Warn("封面预览AI输出未包含有效HTML", "courseware_id", coursewareID, "page_num", pageNum)
			errors = append(errors, errMsg)
			failCount++
			continue
		}

		// 构建匹配组件ID列表
		matchedIDs := s.buildMatchedComponentIDs(matchedComps)

		// 写入数据库
		if dbErr := repository.UpdateCWPageHTML(ctx, page.ID, htmlContent, "", matchedIDs, models.CWPageStatusGenerated); dbErr != nil {
			errMsg := fmt.Sprintf("封面预览保存HTML失败: %v", dbErr)
			cwGenLog.Error("封面预览保存HTML失败", "error", dbErr, "courseware_id", coursewareID, "page_num", pageNum)
			errors = append(errors, errMsg)
			failCount++
			continue
		}

		successCount++
		cwGenLog.Info("封面预览生成成功", "model", result.ModelUsed, "tokens", result.TokensUsed, "courseware_id", coursewareID)

		// 广播单页完成
		GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
			EventType: CWSSEGenPage,
			Data: map[string]interface{}{
				"page_number":  pageNum,
				"page_id":      page.ID,
				"title":        page.Title,
				"html_content": htmlContent,
				"model_used":   result.ModelUsed,
				"tokens_used":  result.TokensUsed,
			},
		})
	}

	// ---- 8. 预览生成完成（不改变课件状态） ----
	elapsed := time.Since(startTime)
	cwGenLog.Info("封面预览生成完成",
		"courseware_id", coursewareID,
		"success", successCount,
		"fail", failCount,
		"elapsed_ms", elapsed.Milliseconds(),
	)

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenDone,
		Data: map[string]interface{}{
			"courseware_id":  coursewareID,
			"success_count": successCount,
			"fail_count":    failCount,
			"total_pages":   previewCount,
			"elapsed_ms":    elapsed.Milliseconds(),
			"errors":        errors,
			"is_preview":    true,
			"message":       fmt.Sprintf("封面预览生成完成！请确认导航栏样式后继续。"),
		},
	})

	return nil
}

// ==================== Step 2: 生成剩余页面（后端硬拼接导航栏） ====================

// GenerateRemainingPages P0-1改造：AI只生成内容区HTML，后端硬拼接导航栏
// 前提：nav_template_html已保存（含 {{PAGE_NUM}} / {{TOTAL_PAGES}} 占位符）
// 完成后状态generating→preview
func (s *CoursewareGenService) GenerateRemainingPages(ctx context.Context, coursewareID string, userID string) error {
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
	if cw.Status != models.CoursewareStatusGenerating {
		s.broadcastError(coursewareID, "当前状态不允许生成课件: "+cw.Status)
		return fmt.Errorf("当前状态不允许生成课件: %s", cw.Status)
	}
	// 必须已保存导航栏模板
	if strings.TrimSpace(cw.NavTemplateHTML) == "" {
		s.broadcastError(coursewareID, "请先确认导航栏样式")
		return fmt.Errorf("导航栏模板未保存，请先确认导航栏样式")
	}

	// ---- 2. 获取全部页面方案 ----
	pages, err := repository.ListCoursewarePages(ctx, coursewareID)
	if err != nil || len(pages) == 0 {
		s.broadcastError(coursewareID, "课件没有页面方案")
		return fmt.Errorf("课件页面为空")
	}

	// 找出尚未生成HTML的页面（跳过已生成的预览页）
	var remainingPages []*models.CoursewarePage
	for _, p := range pages {
		if p.HTMLContent == "" {
			remainingPages = append(remainingPages, p)
		}
	}

	if len(remainingPages) == 0 {
		// 所有页面都已生成，直接完成
		_ = repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusPreview)
		GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
			EventType: CWSSEGenDone,
			Data: map[string]interface{}{
				"courseware_id":  coursewareID,
				"success_count": len(pages),
				"fail_count":    0,
				"total_pages":   len(pages),
				"message":       "所有页面已生成完毕！",
			},
		})
		return nil
	}

	// ---- 3. 解析风格配置 + 加载模板 ----
	styleCfg := s.parseStyleConfig(cw.StyleConfig)
	tplInfo, err := s.loadTemplateInfo(ctx, styleCfg.TemplateID)
	if err != nil {
		cwGenLog.Warn("加载模板失败，使用默认风格", "error", err, "courseware_id", coursewareID)
		tplInfo = s.defaultTemplateInfo()
	}

	logoURL, orgName := s.resolveLogoAndOrg(ctx, cw, styleCfg)

	// ---- 4. 加载生成提示词 ----
	genPrompt, err := repository.GetCurrentPromptByKey("prompt_courseware_generate")
	if err != nil {
		s.broadcastError(coursewareID, "加载生成提示词失败: "+err.Error())
		return fmt.Errorf("加载生成提示词失败: %w", err)
	}

	// ---- 5. 获取AI配置 ----
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.GetAESKey(), "courseware_generate",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		s.broadcastError(coursewareID, "获取AI配置失败: "+err.Error())
		return fmt.Errorf("获取AI配置失败: %w", err)
	}

	totalPages := len(pages)
	navTemplate := cw.NavTemplateHTML

	// ---- 6. 广播开始事件 ----
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenStart,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"total_pages":  len(remainingPages),
			"template":     tplInfo.Name,
			"message":      fmt.Sprintf("开始生成剩余 %d 页课件（导航栏已固定）...", len(remainingPages)),
			"is_preview":   false,
		},
	})

	// ---- 7. 逐页生成剩余页面（AI只生成内容区，后端硬拼接导航栏） ----
	successCount := 0
	failCount := 0
	var genErrors []string

	// P0-5: 注册取消信号
	cancelCh := make(chan struct{})
	cwGenCancelMap.Store(coursewareID, cancelCh)
	defer cwGenCancelMap.Delete(coursewareID)

	for i, page := range remainingPages {
		// P0-5: 检查取消信号
		select {
		case <-cancelCh:
			cwGenLog.Info("收到取消信号，停止生成",
				"courseware_id", coursewareID,
				"generated", successCount,
			)
			GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
				EventType: CWSSEGenDone,
				Data: map[string]interface{}{
					"courseware_id":  coursewareID,
					"success_count": successCount,
					"fail_count":    failCount,
					"total_pages":   len(remainingPages),
					"elapsed_ms":    time.Since(startTime).Milliseconds(),
					"errors":        genErrors,
					"is_preview":    false,
					"cancelled":     true,
					"message":       fmt.Sprintf("已停止生成，已完成 %d 页", successCount),
				},
			})
			return nil
		default:
		}

		progressNum := i + 1
		pageNum := page.PageNumber
		cwGenLog.Info("生成批量页",
			"progress", fmt.Sprintf("%d/%d", progressNum, len(remainingPages)),
			"page_num", pageNum,
			"title", page.Title,
			"courseware_id", coursewareID,
		)

		// 广播进度
		GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
			EventType: CWSSEGenProgress,
			Data: map[string]interface{}{
				"current_page": progressNum,
				"total_pages":  len(remainingPages),
				"page_title":   page.Title,
				"message":      fmt.Sprintf("正在生成第 %d/%d 页(P%d)：%s", progressNum, len(remainingPages), pageNum, page.Title),
			},
		})

		// 匹配组件
		matchedComps := s.matchComponentsForPage(ctx, page, cw.Subject, cw.Grade)

		// P0-1: 构建用户提示词（批量模式：AI只生成内容区，不含导航栏）
		userPrompt := s.buildBatchUserPrompt(page, pageNum, totalPages, tplInfo, logoURL, orgName, matchedComps, cw)

		// 调用AI生成（AI只输出内容区HTML）
		traceCtx := &ai.TraceContext{SceneCode: "courseware_generate", UserID: &userID}
		result, aiErr := ai.CallAI(aiCfg, genPrompt.Content, userPrompt, traceCtx)
		if aiErr != nil {
			errMsg := fmt.Sprintf("第%d页AI生成失败: %v", pageNum, aiErr)
			cwGenLog.Error("批量页AI生成失败", "error", aiErr, "courseware_id", coursewareID, "page_num", pageNum)
			genErrors = append(genErrors, errMsg)
			failCount++
			GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
				EventType: CWSSEGenProgress,
				Data: map[string]interface{}{
					"current_page": progressNum,
					"total_pages":  len(remainingPages),
					"page_title":   page.Title,
					"error":        errMsg,
					"message":      fmt.Sprintf("⚠️ 第 %d 页生成失败，继续下一页", pageNum),
				},
			})
			continue
		}

		// 提取AI输出的内容区HTML
		contentHTML := s.extractHTMLFromAIOutput(result.Content)
		if contentHTML == "" {
			errMsg := fmt.Sprintf("第%d页AI输出未包含有效HTML", pageNum)
			cwGenLog.Warn("批量页AI输出未包含有效HTML", "courseware_id", coursewareID, "page_num", pageNum)
			genErrors = append(genErrors, errMsg)
			failCount++
			continue
		}

		// P0-1核心：后端硬拼接导航栏 + 内容区 → 完整页面
		fullPageHTML := s.assembleFullPage(contentHTML, navTemplate, pageNum, totalPages, tplInfo)

		// 构建匹配组件ID列表
		matchedIDs := s.buildMatchedComponentIDs(matchedComps)

		// 写入数据库（写入的是拼接后的完整页面HTML）
		if dbErr := repository.UpdateCWPageHTML(ctx, page.ID, fullPageHTML, "", matchedIDs, models.CWPageStatusGenerated); dbErr != nil {
			errMsg := fmt.Sprintf("第%d页保存HTML失败: %v", pageNum, dbErr)
			cwGenLog.Error("批量页保存HTML失败", "error", dbErr, "courseware_id", coursewareID, "page_num", pageNum)
			genErrors = append(genErrors, errMsg)
			failCount++
			continue
		}

		successCount++
		cwGenLog.Info("批量页生成成功",
			"progress", fmt.Sprintf("%d/%d", progressNum, len(remainingPages)),
			"page_num", pageNum,
			"model", result.ModelUsed,
			"tokens", result.TokensUsed,
			"courseware_id", coursewareID,
		)

		// 广播单页完成（返回拼接后的完整HTML给前端显示）
		GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
			EventType: CWSSEGenPage,
			Data: map[string]interface{}{
				"page_number":  pageNum,
				"page_id":      page.ID,
				"title":        page.Title,
				"html_content": fullPageHTML,
				"model_used":   result.ModelUsed,
				"tokens_used":  result.TokensUsed,
			},
		})
	}

	// ---- 8. 全部完成，更新课件状态 ----
	elapsed := time.Since(startTime)
	if successCount > 0 {
		_ = repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusPreview)
	}

	cwGenLog.Info("课件剩余页面生成完成",
		"courseware_id", coursewareID,
		"success", successCount,
		"fail", failCount,
		"elapsed_ms", elapsed.Milliseconds(),
	)

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenDone,
		Data: map[string]interface{}{
			"courseware_id":  coursewareID,
			"success_count": successCount,
			"fail_count":    failCount,
			"total_pages":   len(remainingPages),
			"elapsed_ms":    elapsed.Milliseconds(),
			"errors":        genErrors,
			"is_preview":    false,
			"message":       fmt.Sprintf("课件生成完成！成功 %d 页，失败 %d 页", successCount, failCount),
		},
	})

	return nil
}

// ==================== P0-5: 中途中断生成 ====================

// cwGenCancelMap 存储每个coursewareID的取消信号channel
var cwGenCancelMap sync.Map

// CancelGenerate 发送取消信号，中断正在进行的批量生成
func (s *CoursewareGenService) CancelGenerate(coursewareID string) {
	if ch, ok := cwGenCancelMap.Load(coursewareID); ok {
		select {
		case <-ch.(chan struct{}):
			// 已经关闭了
		default:
			close(ch.(chan struct{}))
			cwGenLog.Info("发送取消信号", "courseware_id", coursewareID)
		}
	} else {
		cwGenLog.Warn("没有正在进行的生成任务", "courseware_id", coursewareID)
	}
}

// ==================== SSE错误广播 ====================

// broadcastError 广播错误事件
func (s *CoursewareGenService) broadcastError(coursewareID string, message string) {
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEError,
		Data:      map[string]interface{}{"message": message},
	})
}
