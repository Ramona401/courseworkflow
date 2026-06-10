package services

// courseware_gen_service.go — 课件HTML逐页AI生成服务（核心生成逻辑）
//
// 本文件包含：
//   - CoursewareGenService 结构体定义和构造函数
//   - 风格配置解析结构体
//   - GeneratePreviewPages：生成预览页（仅封面P1）
//   - GenerateRemainingPages：生成剩余页面（后端硬拼接导航栏，迭代二P1改为受控并发）
//   - CancelGenerate：中途中断生成
//   - broadcastError：SSE错误广播
//
// 拆分自原 courseware_gen_service.go（v142 结构化日志迁移+模块化拆分）
// 迭代二Phase1-P1：GenerateRemainingPages 由逐页串行改为信号量受控并发，
//   单页 AI 出图耗时砍不掉，但多页"排队等"被砍掉，20页总时长由 ~19分钟降到 5-6分钟（并发4）。
//
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

// P2：单页 AI 生成失败时的自动重试参数
//   cwGenMaxAttempts —— 单页最多尝试次数（1 次原始 + 重试，共 3 次尝试 = 重试2次）
//   cwGenRetryBaseDelay —— 重试基础间隔，第 n 次重试前等待 n*base（1s、2s），扛中转商偶发 HTTP 500
//   重试只在该页失败时发生，且在该页自身 goroutine 内串行进行，不阻塞其他并发页 → 成功页速度不受影响
const (
	cwGenMaxAttempts    = 3
	cwGenRetryBaseDelay = 1 * time.Second
)

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
	SamplePages   []string          // 任务2新增：模板样例页HTML数组（分页参考注入生成提示词）
	CoverBgURL    string            // 批次1新增：课件级老师选择的封面背景URL（图库快照，空=未选）
	ContentBgURL  string            // 批次1新增：课件级老师选择的内页背景URL（空=未选）
}

// ==================== Step 1: 生成预览页（仅封面P1） ====================

// GeneratePreviewPages P0-1改造：仅生成封面页(P1)，让老师确认导航栏样式
// AI生成时用 <!-- NAV_START --> / <!-- NAV_END --> 标记包裹导航栏
// 生成完成后不改变课件状态（仍为generating），等老师确认导航栏
//
// 说明：本方法只生成1页封面，无并发改造价值，迭代二P1保持原串行不动。
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

	// 批次1（背景图库）：把课件级老师选择的背景URL挂进生成上下文（三级优先级第一级）
	s.attachUserBackground(ctx, cw, tplInfo)

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
			"total_pages":   previewCount,
			"template":      tplInfo.Name,
			"message":       "正在生成封面预览页，请稍候...",
			"is_preview":    true,
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
		// 背景兜底注入（修复）：封面预览路径不走 assembleFullPage，单独接入官方背景强制注入，
		// 确保确认导航栏页看到的封面必然带模板背景图，不再依赖AI是否采纳。
		htmlContent = s.applyTemplateBackground(htmlContent, tplInfo, pageNum)
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

	// 小修：message 按成败分情况说真话。失败(failCount>0)时把真实原因（errors 首条，
	//   如"积分余额不足/超时/HTTP 500"）拼进 message，让前端直接显示真实原因，
	//   不再无论成败都报"生成完成"误导老师（曾导致老师以为成功、对着空白预览区一脸懵）。
	previewMsg := "封面预览生成完成！请确认导航栏样式后继续。"
	if failCount > 0 {
		reason := "AI生成失败"
		if len(errors) > 0 && errors[0] != "" {
			reason = errors[0]
		}
		previewMsg = fmt.Sprintf("封面预览生成失败：%s。请稍后重试，若持续失败请联系管理员。", reason)
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenDone,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"success_count": successCount,
			"fail_count":    failCount,
			"total_pages":   previewCount,
			"elapsed_ms":    elapsed.Milliseconds(),
			"errors":        errors,
			"is_preview":    true,
			"message":       previewMsg,
		},
	})

	return nil
}

// ==================== Step 2: 生成剩余页面（后端硬拼接导航栏，受控并发） ====================

// cwGenPageResult 单页并发生成的结果（goroutine 内填充，汇总时回放 SSE 与计数）
//
// 迭代二P1：每个并发 goroutine 把"是否成功 / 完整HTML / 错误信息 / AI元数据"装进本结构，
// 而不是直接在 goroutine 里改共享计数器。计数与"单页完成"SSE 广播统一在持锁段做，
// 既保证 successCount/failCount/genErrors 无数据竞争，又让前端进度按页落地不丢事件。
type cwGenPageResult struct {
	progressNum int    // 本次批量内的序号（1..N，用于进度文案）
	pageNum     int    // 课件内真实页码
	pageID      string // 页面ID
	title       string // 页标题
	ok          bool   // 是否成功
	fullHTML    string // 成功时：拼接导航栏后的完整页面HTML
	modelUsed   string // 成功时：使用的模型
	tokensUsed  int    // 成功时：消耗tokens
	errMsg      string // 失败时：错误信息
}

// GenerateRemainingPages 生成剩余页面（AI只生成内容区HTML，后端硬拼接导航栏）
// 前提：nav_template_html已保存（含 {{PAGE_NUM}} / {{TOTAL_PAGES}} 占位符）
// 完成后状态generating→preview
//
// 迭代二Phase1-P1 改造要点：
//   - 逐页串行循环 → 信号量受控并发（容量 s.cfg.CoursewareGenConcurrency，默认4）
//   - 共享计数器 successCount/failCount/genErrors 用 sync.Mutex 保护
//   - 每个 goroutine 进入前 select 检查取消信号，已取消则跳过该页
//   - SSE「进度提示」在 goroutine 启动时广播（并发安全，Hub内有锁）
//     「单页完成/失败」在汇总持锁段广播，保证计数与事件一致、顺序稳定
//   - 单页失败不影响其他页（隔离性），页面按 page.ID 独立落库（无写库竞争）
//   - 并发数配置为 1 即完全退化为原串行行为（零风险回滚）
func (s *CoursewareGenService) GenerateRemainingPages(ctx context.Context, coursewareID string, userID string) error {
	startTime := time.Now()

	// 防并发：同一课件同时只允许一个批量生成任务，避免连点/多标签页重复生成、重复扣 token、写库竞争
	if _, busy := cwGenRunning.LoadOrStore(coursewareID, struct{}{}); busy {
		s.broadcastError(coursewareID, "该课件正在生成中，请勿重复触发")
		return fmt.Errorf("课件正在生成中: %s", coursewareID)
	}
	defer cwGenRunning.Delete(coursewareID)

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
	if cw.Status != models.CoursewareStatusGenerating && cw.Status != models.CoursewareStatusPreview {
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
				"courseware_id": coursewareID,
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

	// 批次1（背景图库）：把课件级老师选择的背景URL挂进生成上下文（三级优先级第一级）
	s.attachUserBackground(ctx, cw, tplInfo)

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
	remainingCount := len(remainingPages)

	// ---- 6. 解析并发数（兜底再保险：config 已保证≥1，此处再防御一次） ----
	concurrency := s.cfg.CoursewareGenConcurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > remainingCount {
		concurrency = remainingCount // 并发数不超过待生成页数，避免开多余goroutine
	}

	// ---- 7. 广播开始事件 ----
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenStart,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"total_pages":   remainingCount,
			"template":      tplInfo.Name,
			"concurrency":   concurrency,
			"message":       fmt.Sprintf("开始生成剩余 %d 页课件（导航栏已固定，并发 %d）...", remainingCount, concurrency),
			"is_preview":    false,
		},
	})

	// ---- 8. 注册取消信号 ----
	cancelCh := make(chan struct{})
	cwGenCancelMap.Store(coursewareID, cancelCh)
	defer cwGenCancelMap.Delete(coursewareID)

	// ---- 9. 受控并发生成 ----
	// 共享计数器 + 错误收集，全部用 mu 保护
	var (
		mu           sync.Mutex
		successCount int
		failCount    int
		genErrors    []string
		cancelled    bool // 是否因取消信号提前结束
	)

	// 信号量：容量=concurrency，限制同时在跑的 goroutine 数
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, page := range remainingPages {
		// 派发前先看是否已取消：已取消则不再派发后续页
		select {
		case <-cancelCh:
			mu.Lock()
			cancelled = true
			mu.Unlock()
		default:
		}
		mu.Lock()
		stop := cancelled
		mu.Unlock()
		if stop {
			break
		}

		wg.Add(1)
		sem <- struct{}{} // 占用一个并发名额（满则阻塞，天然限流）

		go func(idx int, p *models.CoursewarePage) {
			defer wg.Done()
			defer func() { <-sem }() // 释放并发名额

			progressNum := idx + 1
			pageNum := p.PageNumber

			// goroutine 内再次检查取消信号：取消后已占名额的页直接跳过，不再烧 token
			select {
			case <-cancelCh:
				mu.Lock()
				cancelled = true
				mu.Unlock()
				return
			default:
			}

			// 广播进度（Hub内有锁，并发安全）——本页开始生成
			GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
				EventType: CWSSEGenProgress,
				Data: map[string]interface{}{
					"current_page": progressNum,
					"total_pages":  remainingCount,
					"page_title":   p.Title,
					"message":      fmt.Sprintf("正在生成 P%d：%s（本次进度 %d/%d）", pageNum, p.Title, progressNum, remainingCount),
				},
			})

			res := cwGenPageResult{progressNum: progressNum, pageNum: pageNum, pageID: p.ID, title: p.Title}

			cwGenLog.Info("生成批量页",
				"progress", fmt.Sprintf("%d/%d", progressNum, remainingCount),
				"page_num", pageNum,
				"title", p.Title,
				"courseware_id", coursewareID,
			)

			// 匹配组件（纯查询，并发安全）
			matchedComps := s.matchComponentsForPage(ctx, p, cw.Subject, cw.Grade)

			// 构建用户提示词（批量模式：AI只生成内容区，不含导航栏）
			userPrompt := s.buildBatchUserPrompt(p, pageNum, totalPages, tplInfo, logoURL, orgName, matchedComps, cw)

			// 调用AI生成（每页独立 traceCtx，无共享）；P2：失败自动重试最多 cwGenMaxAttempts 次
			//   重试仅发生在该页失败时，串行在本 goroutine 内重试，不阻塞其他并发页 → 成功页速度不变。
			traceCtx := &ai.TraceContext{SceneCode: "courseware_generate", UserID: &userID}
			var result *ai.CallResult
			var aiErr error
			for attempt := 1; attempt <= cwGenMaxAttempts; attempt++ {
				// 每次尝试前再检查取消信号：取消后不再浪费 AI 调用
				select {
				case <-cancelCh:
					mu.Lock()
					cancelled = true
					mu.Unlock()
					return
				default:
				}
				result, aiErr = ai.CallAI(aiCfg, genPrompt.Content, userPrompt, traceCtx)
				if aiErr == nil {
					if attempt > 1 {
						cwGenLog.Info("批量页AI重试成功",
							"courseware_id", coursewareID, "page_num", pageNum, "attempt", attempt)
					}
					break // 成功，跳出重试循环
				}
				// 本次失败：记录并在还有重试机会时退避后重试
				cwGenLog.Warn("批量页AI生成失败，准备重试",
					"error", aiErr, "courseware_id", coursewareID, "page_num", pageNum,
					"attempt", attempt, "max_attempts", cwGenMaxAttempts)
				if attempt < cwGenMaxAttempts {
					// 退避间隔随重试次数递增（1s、2s），扛中转商偶发 HTTP 500
					time.Sleep(time.Duration(attempt) * cwGenRetryBaseDelay)
				}
			}
			if aiErr != nil {
				// 重试 cwGenMaxAttempts 次仍失败，才真正计入失败
				res.errMsg = fmt.Sprintf("第%d页AI生成失败(已重试%d次): %v", pageNum, cwGenMaxAttempts-1, aiErr)
				cwGenLog.Error("批量页AI生成最终失败", "error", aiErr, "courseware_id", coursewareID, "page_num", pageNum, "attempts", cwGenMaxAttempts)
				s.collectPageResult(coursewareID, &mu, &successCount, &failCount, &genErrors, res, remainingCount)
				return
			}

			// 提取AI输出的内容区HTML
			contentHTML := s.extractHTMLFromAIOutput(result.Content)
			if contentHTML == "" {
				res.errMsg = fmt.Sprintf("第%d页AI输出未包含有效HTML", pageNum)
				cwGenLog.Warn("批量页AI输出未包含有效HTML", "courseware_id", coursewareID, "page_num", pageNum)
				s.collectPageResult(coursewareID, &mu, &successCount, &failCount, &genErrors, res, remainingCount)
				return
			}

			// P0-1核心：后端硬拼接导航栏 + 内容区 → 完整页面（纯函数，并发安全）
			fullPageHTML := s.assembleFullPage(contentHTML, navTemplate, pageNum, totalPages, tplInfo)

			// 构建匹配组件ID列表
			matchedIDs := s.buildMatchedComponentIDs(matchedComps)

			// 写入数据库（按 page.ID 独立行 UPDATE，pgxpool 并发安全，无行竞争）
			if dbErr := repository.UpdateCWPageHTML(ctx, p.ID, fullPageHTML, "", matchedIDs, models.CWPageStatusGenerated); dbErr != nil {
				res.errMsg = fmt.Sprintf("第%d页保存HTML失败: %v", pageNum, dbErr)
				cwGenLog.Error("批量页保存HTML失败", "error", dbErr, "courseware_id", coursewareID, "page_num", pageNum)
				s.collectPageResult(coursewareID, &mu, &successCount, &failCount, &genErrors, res, remainingCount)
				return
			}

			// 成功
			res.ok = true
			res.fullHTML = fullPageHTML
			res.modelUsed = result.ModelUsed
			res.tokensUsed = result.TokensUsed
			cwGenLog.Info("批量页生成成功",
				"progress", fmt.Sprintf("%d/%d", progressNum, remainingCount),
				"page_num", pageNum,
				"model", result.ModelUsed,
				"tokens", result.TokensUsed,
				"courseware_id", coursewareID,
			)
			s.collectPageResult(coursewareID, &mu, &successCount, &failCount, &genErrors, res, remainingCount)
		}(i, page)
	}

	// 等待所有已派发的 goroutine 完成（已派发的页会跑完，未派发的因取消而不再派发）
	wg.Wait()

	// ---- 10. 完成（含取消分支） ----
	elapsed := time.Since(startTime)

	mu.Lock()
	finalSuccess := successCount
	finalFail := failCount
	finalErrors := append([]string(nil), genErrors...)
	wasCancelled := cancelled
	mu.Unlock()

	// 只要有任意页成功落库，就把状态推进到 preview（与原串行行为一致）
	if finalSuccess > 0 {
		_ = repository.UpdateCoursewareStatus(ctx, coursewareID, models.CoursewareStatusPreview)
	}

	if wasCancelled {
		cwGenLog.Info("课件生成被取消",
			"courseware_id", coursewareID,
			"success", finalSuccess,
			"fail", finalFail,
			"elapsed_ms", elapsed.Milliseconds(),
		)
		GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
			EventType: CWSSEGenDone,
			Data: map[string]interface{}{
				"courseware_id": coursewareID,
				"success_count": finalSuccess,
				"fail_count":    finalFail,
				"total_pages":   remainingCount,
				"elapsed_ms":    elapsed.Milliseconds(),
				"errors":        finalErrors,
				"is_preview":    false,
				"cancelled":     true,
				"message":       fmt.Sprintf("已停止生成，已完成 %d 页", finalSuccess),
			},
		})
		return nil
	}

	cwGenLog.Info("课件剩余页面生成完成",
		"courseware_id", coursewareID,
		"success", finalSuccess,
		"fail", finalFail,
		"concurrency", concurrency,
		"elapsed_ms", elapsed.Milliseconds(),
	)

	// 小修：批量生成 message 也按成败给真实信息。有失败页时把首条真实原因带上，
	//   与封面预览保持一致的"说真话"口径，便于前端/老师定位（如积分余额不足/超时）。
	doneMsg := fmt.Sprintf("课件生成完成！成功 %d 页，失败 %d 页", finalSuccess, finalFail)
	if finalFail > 0 && len(finalErrors) > 0 && finalErrors[0] != "" {
		doneMsg = fmt.Sprintf("课件生成完成：成功 %d 页，失败 %d 页（失败原因：%s）", finalSuccess, finalFail, finalErrors[0])
	}

	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenDone,
		Data: map[string]interface{}{
			"courseware_id": coursewareID,
			"success_count": finalSuccess,
			"fail_count":    finalFail,
			"total_pages":   remainingCount,
			"elapsed_ms":    elapsed.Milliseconds(),
			"errors":        finalErrors,
			"is_preview":    false,
			"message":       doneMsg,
		},
	})

	return nil
}

// collectPageResult 汇总单页生成结果（持锁段统一更新计数器 + 广播单页完成/失败事件）
//
// 迭代二P1：所有 goroutine 通过本方法把结果写回共享计数器，并发下用 mu 串行化，
// 保证 successCount/failCount/genErrors 无数据竞争；同时把「单页完成」SSE 放在锁内，
// 既与计数一致，又避免多 goroutine 同时构造大 HTML 事件时的顺序错乱。
func (s *CoursewareGenService) collectPageResult(
	coursewareID string,
	mu *sync.Mutex,
	successCount *int,
	failCount *int,
	genErrors *[]string,
	res cwGenPageResult,
	remainingCount int,
) {
	mu.Lock()
	defer mu.Unlock()

	if res.ok {
		*successCount++
		// 广播单页完成（返回拼接后的完整HTML给前端显示）
		GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
			EventType: CWSSEGenPage,
			Data: map[string]interface{}{
				"page_number":  res.pageNum,
				"page_id":      res.pageID,
				"title":        res.title,
				"html_content": res.fullHTML,
				"model_used":   res.modelUsed,
				"tokens_used":  res.tokensUsed,
			},
		})
		return
	}

	// 失败：计数 + 收集错误 + 广播失败进度
	*failCount++
	if res.errMsg != "" {
		*genErrors = append(*genErrors, res.errMsg)
	}
	GlobalCWSSEHub.Broadcast(coursewareID, CWSSEEvent{
		EventType: CWSSEGenProgress,
		Data: map[string]interface{}{
			"current_page": res.progressNum,
			"total_pages":  remainingCount,
			"page_title":   res.title,
			"error":        res.errMsg,
			"message":      fmt.Sprintf("⚠️ 第 %d 页生成失败，继续其他页", res.pageNum),
		},
	})
}

// ==================== P0-5: 中途中断生成 ====================

// cwGenRunning 标记 coursewareID 是否有批量生成进行中（防并发重复生成）
var cwGenRunning sync.Map

// cwGenCancelMap 存储每个coursewareID的取消信号channel
var cwGenCancelMap sync.Map

// CancelGenerate 发送取消信号，中断正在进行的批量生成
//
// 迭代二P1：并发改造后取消语义不变——关闭 cancelCh 后，
// 尚未派发的页不再派发，已占名额但未开跑 AI 的 goroutine 也会 select 命中后跳过；
// 已经在跑 AI 的页会自然跑完（不强杀正在进行的 HTTP 调用），符合预期。
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
