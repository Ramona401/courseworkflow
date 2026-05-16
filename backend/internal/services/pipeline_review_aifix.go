package services

// pipeline_review_aifix.go — Pipeline AI快修服务
//
// 从 pipeline_review.go 拆分,包含:
//   - AIFixPage: 非流式AI快修
//   - AIFixPageStream: SSE流式AI快修
//   - extractFixSummary/extractFixedHTML: 输出解析

import (
	"fmt"
	"regexp"
	"strings"
	"tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== AI快修 ====================

// aiFixSystemPrompt AI快修专用系统提示词（v68新增）
const aiFixSystemPrompt = `你是一个精确的HTML课件修复专家。你的唯一任务是根据审核员的修复指令，在现有HTML代码基础上做最小化修改。

## 铁律（违反任何一条即为失败）

1. **只改指令要求的部分**：审核员没提到的地方，一个字符都不能动
2. **保持原有格式和布局**：CSS样式、class名称、id属性、HTML结构层级必须原封不动保留
3. **禁止重写页面**：你不是在"重新生成"页面，你是在"修补"页面
4. **导航栏/视频/图片/音频等媒体元素**：绝对禁止任何改动
5. **保留所有注释和空行**：原代码中的注释和格式化空行必须保留
6. **输出完整HTML**：输出修改后的完整HTML，不要输出代码片段

## 输出格式要求

你必须先输出修改说明，再输出完整HTML。格式如下：

<<<FIX_SUMMARY>>>
- 修改点1：简要说明改了什么（例如：将第3题的选项A从"光合作用"改为"呼吸作用"）
- 修改点2：简要说明改了什么
<<<END_FIX_SUMMARY>>>

<<<FIXED_HTML>>>
这里是修改后的完整HTML代码
<<<END_FIXED_HTML>>>

## 修改说明的规范
- 每个修改点用"- "开头，一行一个
- 说明要具体：改了哪里、从什么改成什么
- 如果修改涉及多处，逐一列出
- 总共不超过10个修改点

## 参考页面的使用
- 如果审核员提供了参考页面，仅作为风格/格式/内容的参考依据
- 参考页面的代码不能直接复制到当前页面
- 你只能参考其设计思路、交互方式、排版风格来改进当前页面`

// reFixSummary 提取修改说明的正则（v68新增）
var reFixSummary = regexp.MustCompile(`(?s)<<<FIX_SUMMARY>>>(.*?)<<<END_FIX_SUMMARY>>>`)

// reFixedHTML 提取修复后HTML的正则（v68新增）
var reFixedHTML = regexp.MustCompile(`(?s)<<<FIXED_HTML>>>(.*?)<<<END_FIXED_HTML>>>`)

// AIFixResult AI快修返回结果（v68增强）
type AIFixResult struct {
	NewHTML    string `json:"new_html"`
	FixSummary string `json:"fix_summary"`
	HTMLLength int    `json:"html_length"`
}

// AIFixPage AI快修单个页面
// v68增强：使用ai_fix独立场景(Sonnet) + 专用系统提示词 + 修改说明提取 + 参考页面支持
// 参数 referencePageNums：审核员选择的参考页码列表（可为空）
func (s *PipelineService) AIFixPage(pipelineID string, pageNumber int, fixInstruction string, referencePageNums []int) (*AIFixResult, error) {
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		return nil, ErrPipelineNotFound
	}
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue:     true,
		models.PipelineStatusNeedsHuman:      true,
		models.PipelineStatusFinalized:       true,
		models.PipelineStatusPendingFinalize: true,
	}
	if !allowedStatuses[pipeline.Status] {
		return nil, ErrPipelineNotReviewable
	}

	// 获取所有页面数据（供当前页面+参考页面使用）
	pages, err := repository.GetGeneratedPagesWithHTML(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("获取页面数据失败: %w", err)
	}

	// 查找当前页面
	var currentPage *repository.GeneratedPageFullRow
	// 构建页码到页面的映射（用于参考页面查找）
	pageMap := make(map[int]*repository.GeneratedPageFullRow)
	for _, p := range pages {
		pageMap[p.PageNumber] = p
		if p.PageNumber == pageNumber {
			currentPage = p
		}
	}
	if currentPage == nil {
		return nil, ErrPageNotFound
	}

	// 获取当前最新HTML（优先级：final_html > generated_html > original_html）
	currentHTML := currentPage.FinalHTML
	if currentHTML == "" {
		currentHTML = currentPage.GeneratedHTML
	}
	if currentHTML == "" {
		currentHTML = currentPage.OriginalHTML
	}
	if currentHTML == "" {
		return nil, fmt.Errorf("页面P%d无可用HTML内容", pageNumber)
	}

	// v68改动：使用ai_fix独立场景（Sonnet + 64000 tokens）替代generator场景（Opus + 200000）
	// AI快修只修改单页局部内容，不需要最强模型和最大token
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey, "ai_fix",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		return nil, fmt.Errorf("获取AI配置失败: %w", err)
	}

	// 使用专用系统提示词
	systemPrompt := aiFixSystemPrompt

	// 构建用户提示词
	var userPromptParts []string

	userPromptParts = append(userPromptParts,
		fmt.Sprintf("【当前页面】P%02d — %s", pageNumber, currentPage.PageTitle),
		"",
		"【当前页面完整HTML — 你必须在此基础上修复，禁止重写】",
		currentHTML,
		"",
	)

	// v68新增：拼入参考页面HTML（审核员选择的其他页面作为参考）
	if len(referencePageNums) > 0 {
		userPromptParts = append(userPromptParts,
			"══════════════════════════════════════════════",
			"【参考页面（仅供参考风格/格式/内容，禁止直接复制代码）】",
			"",
		)
		refCount := 0
		for _, refPN := range referencePageNums {
			if refPN == pageNumber {
				continue // 跳过当前页面自身
			}
			refPage, exists := pageMap[refPN]
			if !exists {
				continue
			}
			// 获取参考页面的最新HTML
			refHTML := refPage.FinalHTML
			if refHTML == "" {
				refHTML = refPage.GeneratedHTML
			}
			if refHTML == "" {
				refHTML = refPage.OriginalHTML
			}
			if refHTML == "" {
				continue
			}
			refCount++
			userPromptParts = append(userPromptParts,
				fmt.Sprintf("--- 参考页面 P%02d: %s ---", refPN, refPage.PageTitle),
				refHTML,
				"",
			)
			// 最多5个参考页面，避免token爆炸
			if refCount >= 5 {
				userPromptParts = append(userPromptParts, "（已达参考页面上限，后续省略）")
				break
			}
		}
		userPromptParts = append(userPromptParts, "")
	}

	userPromptParts = append(userPromptParts,
		"══════════════════════════════════════════════",
		"【审核员修复指令（必须严格执行，只改这里要求的部分）】",
		fixInstruction,
	)

	userPrompt := strings.Join(userPromptParts, "\n")

	// 调用AI
	callResult, callErr := s.callAIWithSemaphore(aiCfg, systemPrompt, userPrompt, pipeline.ID)
	if callErr != nil {
		return nil, fmt.Errorf("%w: %s", ErrAIFixFailed, callErr.Error())
	}

	aiOutput := callResult.Content

	// 提取修改说明
	fixSummary := extractFixSummary(aiOutput)

	// 优先从<<<FIXED_HTML>>>标签提取HTML
	newHTML := extractFixedHTML(aiOutput)
	if newHTML == "" {
		// 降级：使用原有的extractGeneratedHTML（定义在generator_service.go）
		newHTML = extractGeneratedHTML(aiOutput)
	}

	if len(newHTML) < 100 {
		return nil, fmt.Errorf("%w: AI输出HTML过短(%d字符)", ErrAIFixFailed, len(newHTML))
	}

	// 保存修复后的HTML
	if err := repository.UpdateGeneratedPageHTML(pipelineID, pageNumber, newHTML, newHTML); err != nil {
		return nil, fmt.Errorf("保存修复后HTML失败: %w", err)
	}

	return &AIFixResult{
		NewHTML:    newHTML,
		FixSummary: fixSummary,
		HTMLLength: len(newHTML),
	}, nil
}

// extractFixSummary 从AI输出中提取修改说明
func extractFixSummary(output string) string {
	match := reFixSummary.FindStringSubmatch(output)
	if len(match) >= 2 {
		summary := strings.TrimSpace(match[1])
		if summary != "" {
			return summary
		}
	}

	// 降级：查找第一个HTML标签之前的文本
	htmlStartIdx := strings.Index(output, "<!DOCTYPE")
	if htmlStartIdx < 0 {
		htmlStartIdx = strings.Index(output, "<html")
	}
	if htmlStartIdx < 0 {
		htmlStartIdx = strings.Index(output, "<div")
	}
	if htmlStartIdx > 20 {
		preText := strings.TrimSpace(output[:htmlStartIdx])
		if len(preText) > 10 && len(preText) < 2000 {
			return preText
		}
	}

	return ""
}

// extractFixedHTML 从AI输出中提取<<<FIXED_HTML>>>块中的HTML
func extractFixedHTML(output string) string {
	match := reFixedHTML.FindStringSubmatch(output)
	if len(match) >= 2 {
		html := strings.TrimSpace(match[1])
		if len(html) >= 100 {
			return html
		}
	}
	return ""
}


// ==================== AI快修流式调用（v69新增，编号5）====================

// AIFixPageStream AI快修流式版本 — 通过onChunk回调逐token推送
// 与AIFixPage逻辑一致，但使用CallAIStream流式调用
// 完成后自动保存HTML、提取修改说明
// 参数：
//   - onChunk: 每收到一个AI输出token时的回调（前端SSE推送用）
//   - onDone: AI输出完成后回调，传入最终结果（含new_html/fix_summary）
//   - onError: 出错时回调
func (s *PipelineService) AIFixPageStream(
	pipelineID string, pageNumber int, fixInstruction string, referencePageNums []int,
	onChunk func(chunk string),
	onDone func(result *AIFixResult),
	onError func(errMsg string),
) {
	// ===== 前置校验（与AIFixPage一致）=====
	pipeline, err := repository.GetPipelineByID(pipelineID)
	if err != nil {
		onError("Pipeline不存在")
		return
	}
	allowedStatuses := map[string]bool{
		models.PipelineStatusReviewQueue:     true,
		models.PipelineStatusNeedsHuman:      true,
		models.PipelineStatusFinalized:       true,
		models.PipelineStatusPendingFinalize: true,
	}
	if !allowedStatuses[pipeline.Status] {
		onError("Pipeline不在审核状态，无法进行AI快修")
		return
	}

	// 获取所有页面数据
	pages, err := repository.GetGeneratedPagesWithHTML(pipelineID)
	if err != nil {
		onError("获取页面数据失败: " + err.Error())
		return
	}

	var currentPage *repository.GeneratedPageFullRow
	pageMap := make(map[int]*repository.GeneratedPageFullRow)
	for _, p := range pages {
		pageMap[p.PageNumber] = p
		if p.PageNumber == pageNumber {
			currentPage = p
		}
	}
	if currentPage == nil {
		onError(fmt.Sprintf("页面P%d不存在", pageNumber))
		return
	}

	// 获取当前最新HTML
	currentHTML := currentPage.FinalHTML
	if currentHTML == "" {
		currentHTML = currentPage.GeneratedHTML
	}
	if currentHTML == "" {
		currentHTML = currentPage.OriginalHTML
	}
	if currentHTML == "" {
		onError(fmt.Sprintf("页面P%d无可用HTML内容", pageNumber))
		return
	}

	// 获取AI配置
	aiCfg, err := ai.GetEffectiveConfig(
		s.cfg.AESKey, "ai_fix",
		s.cfg.AIAPIBaseURL, s.cfg.AIAPIKey, s.cfg.AIDefaultModel,
	)
	if err != nil {
		onError("获取AI配置失败: " + err.Error())
		return
	}

	// 构建提示词（与AIFixPage完全一致）
	systemPrompt := aiFixSystemPrompt
	var userPromptParts []string
	userPromptParts = append(userPromptParts,
		fmt.Sprintf("【当前页面】P%02d — %s", pageNumber, currentPage.PageTitle),
		"",
		"【当前页面完整HTML — 你必须在此基础上修复，禁止重写】",
		currentHTML,
		"",
	)
	if len(referencePageNums) > 0 {
		userPromptParts = append(userPromptParts,
			"══════════════════════════════════════════════",
			"【参考页面（仅供参考风格/格式/内容，禁止直接复制代码）】", "",
		)
		refCount := 0
		for _, refPN := range referencePageNums {
			if refPN == pageNumber {
				continue
			}
			refPage, exists := pageMap[refPN]
			if !exists {
				continue
			}
			refHTML := refPage.FinalHTML
			if refHTML == "" {
				refHTML = refPage.GeneratedHTML
			}
			if refHTML == "" {
				refHTML = refPage.OriginalHTML
			}
			if refHTML == "" {
				continue
			}
			refCount++
			userPromptParts = append(userPromptParts,
				fmt.Sprintf("--- 参考页面 P%02d: %s ---", refPN, refPage.PageTitle),
				refHTML, "",
			)
			if refCount >= 5 {
				break
			}
		}
		userPromptParts = append(userPromptParts, "")
	}
	userPromptParts = append(userPromptParts,
		"══════════════════════════════════════════════",
		"【审核员修复指令（必须严格执行，只改这里要求的部分）】",
		fixInstruction,
	)
	userPrompt := strings.Join(userPromptParts, "\n")

	// ===== 流式AI调用（带信号量控制）=====
	if s.engine != nil {
		s.engine.AcquireAI()
		defer s.engine.ReleaseAI()
	}

	callResult, callErr := ai.CallAIStream(aiCfg, systemPrompt, userPrompt, func(chunk string) error {
		onChunk(chunk)
		return nil
	}, nil)

	if callErr != nil {
		onError("AI快修调用失败: " + callErr.Error())
		return
	}

	aiOutput := callResult.Content

	// 提取修改说明和HTML（与AIFixPage一致）
	fixSummary := extractFixSummary(aiOutput)
	newHTML := extractFixedHTML(aiOutput)
	if newHTML == "" {
		newHTML = extractGeneratedHTML(aiOutput)
	}
	if len(newHTML) < 100 {
		onError(fmt.Sprintf("AI输出HTML过短(%d字符)，修复可能失败", len(newHTML)))
		return
	}

	// 保存修复后的HTML
	if err := repository.UpdateGeneratedPageHTML(pipelineID, pageNumber, newHTML, newHTML); err != nil {
		onError("保存修复后HTML失败: " + err.Error())
		return
	}

	onDone(&AIFixResult{
		NewHTML:    newHTML,
		FixSummary: fixSummary,
		HTMLLength: len(newHTML),
	})
}
