package services

// workshop_stage_prompts.go — 阶段化备课工坊提示词构建
//
// Phase 7B 新增：四层系统提示词拼接+对话提示词+产出物解析
// 迭代1 改造：五层拼接 + BuildLessonStructurePrompt + selectPromptVariant + buildTechSegment
// 迭代2 改造：降级提示词注入 + BuildPriorOutputsContext跳过阶段标注
// v73 BugFix：extractStructuredFallback 增加第三层容错
// v74 改造：buildTechSegment → buildDialogueGuidelines（去掉标签要求）
//          AutoMatchStageComponents + normalizeGradeToNumber
// v75 重大重构：
//   1. buildTechSegment 完全替换为 buildDialogueGuidelines —— AI不再输出XML标签
//   2. ParseStageOutput / DetectStageComplete / CleanStageMarkers 标记废弃，保留签名
//   3. extractStructuredFallback / extractFieldFromText 保留，供降级提取使用
//   4. 新增 ExtractStructuredFromNaturalReply —— 从自然语言回复中提取结构化数据
//   5. 新增 DetectLessonPlanContent —— 检测AI回复是否包含完整教案内容

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 阶段完整系统提示词构建 ====================

// BuildStageSystemPrompt 构建某阶段完整的系统提示词（六层拼接）
//
// v75变更：第5层从 buildTechSegment 改为 buildDialogueGuidelines
func BuildStageSystemPrompt(
	ctx context.Context,
	stage *models.WorkshopStage,
	recipe *models.TeachingRecipe,
	priorOutputs []*models.WorkshopStageOutput,
	subject string,
	grade string,
	promptMode string,
	lessonStructure string,
) string {
	var sb strings.Builder

	// 第1层：配方全局上下文
	if recipe != nil {
		globalCtx := BuildStageGlobalContext(recipe)
		if globalCtx != "" {
			sb.WriteString(globalCtx)
			sb.WriteString("\n")
		}
	}

	// 第2层：前序阶段产出物
	if len(priorOutputs) > 0 {
		priorCtx := BuildPriorOutputsContext(priorOutputs)
		if priorCtx != "" {
			sb.WriteString(priorCtx)
			sb.WriteString("\n")
		}
	}

	// 第2.5层：降级提示词（跳过的前置阶段补偿指令）
	if len(priorOutputs) > 0 {
		degradation := buildDegradationHint(priorOutputs, recipe)
		if degradation != "" {
			sb.WriteString(degradation)
			sb.WriteString("\n")
		}
	}

	// 第3层：本阶段专属组件内容
	// v74改进：优先使用配方中选择的组件，配方无组件时自动按学科+年级匹配
	if stage.ComponentTypes != "" && stage.ComponentTypes != "[]" {
		componentCtx := ""
		if recipe != nil {
			// 优先从配方组件中筛选
			componentCtx = BuildStageComponentContext(ctx, recipe, stage.ComponentTypes)
		}
		if componentCtx == "" {
			// 配方无组件或未匹配到：自动从组件库按学科+年级+阶段类型匹配
			componentCtx = AutoMatchStageComponents(ctx, stage.ComponentTypes, subject, grade)
		}
		if componentCtx != "" {
			sb.WriteString(componentCtx)
			sb.WriteString("\n")
		}
	}

	// 第3.5层：教案结构偏好
	if lessonStructure != "" && lessonStructure != "[]" {
		structureCtx := BuildLessonStructurePrompt(stage.StageCode, lessonStructure)
		if structureCtx != "" {
			sb.WriteString(structureCtx)
			sb.WriteString("\n")
		}
	}

	// 第4层：阶段角色提示词（来自DB的system_prompt，v74已改为纯自然语言）
	if stage.SystemPrompt != "" {
		sb.WriteString(stage.SystemPrompt)
		sb.WriteString("\n")
	}
	variantText := selectPromptVariant(stage.PromptVariants, promptMode)
	if variantText != "" {
		sb.WriteString("\n")
		sb.WriteString(variantText)
		sb.WriteString("\n")
	}

	// 第5层：对话规范指引（v75替换原buildTechSegment）
	// 不再要求AI输出XML标签，只追加自然语言输出规范
	dialogueGuide := buildDialogueGuidelines(stage.StageCode)
	if dialogueGuide != "" {
		sb.WriteString("\n")
		sb.WriteString(dialogueGuide)
	}

	return sb.String()
}

// ==================== 降级提示词注入 ====================

func buildDegradationHint(priorOutputs []*models.WorkshopStageOutput, recipe *models.TeachingRecipe) string {
	var skippedStages []string
	for _, out := range priorOutputs {
		if out.Status == models.StageOutputSkipped {
			name := stageCodeToName(out.StageCode)
			skippedStages = append(skippedStages, name)
		}
	}
	if len(skippedStages) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("=== 降级补偿提示 ===\n")
	sb.WriteString(fmt.Sprintf("注意：老师跳过了以下阶段：%s。\n", strings.Join(skippedStages, "、")))
	sb.WriteString("这些阶段没有产生分析报告或设计方案。请你根据以下信息自行快速补充：\n")
	if recipe != nil {
		if strings.TrimSpace(recipe.StudentProfile) != "" {
			sb.WriteString("- 学情信息已在配方中提供，请参考配方全局信息中的学情档案\n")
		} else {
			sb.WriteString("- 学情信息缺失，请根据学科和年级特点做合理假设\n")
		}
		if strings.TrimSpace(recipe.TeachingStyle) != "" {
			sb.WriteString("- 教学风格偏好已在配方中提供，请参考\n")
		}
	} else {
		sb.WriteString("- 没有配方信息，请根据学科、年级和课题特点做合理的教学分析和设计假设\n")
	}
	sb.WriteString("请在本阶段工作中自然融入这些补充分析，不需要单独列出。\n")
	return sb.String()
}

// ==================== 教案结构注入 ====================

func BuildLessonStructurePrompt(stageCode string, lessonStructureJSON string) string {
	if stageCode == "analyze" {
		return ""
	}
	var blocks []models.LessonStructureBlock
	if err := json.Unmarshal([]byte(lessonStructureJSON), &blocks); err != nil || len(blocks) == 0 {
		return ""
	}
	var sb strings.Builder
	if stageCode == "design" {
		sb.WriteString("=== 老师期望的教案结构（概要）===\n")
		sb.WriteString("老师希望最终教案包含以下板块，请在设计时考虑：\n")
		for _, b := range blocks {
			required := ""
			if b.Required {
				required = "（必含）"
			}
			sb.WriteString(fmt.Sprintf("- %s%s\n", b.Name, required))
		}
		return sb.String()
	}
	sb.WriteString("=== 老师定义的教案结构 ===\n")
	sb.WriteString("请严格按照以下结构输出教案，每个板块的要求务必遵循：\n\n")
	totalDuration := 0
	for _, b := range blocks {
		required := "选填"
		if b.Required {
			required = "必填"
		}
		sb.WriteString(fmt.Sprintf("【%s】（%s）\n", b.Name, required))
		if b.Requirement != "" {
			sb.WriteString(fmt.Sprintf("  要求：%s\n", b.Requirement))
		}
		if len(b.SubSections) > 0 {
			sb.WriteString("  教学过程环节安排：\n")
			for _, sub := range b.SubSections {
				sb.WriteString(fmt.Sprintf("    ▸ %s（%d分钟）", sub.Name, sub.Duration))
				if sub.Goal != "" {
					sb.WriteString(fmt.Sprintf(" — 目标：%s", sub.Goal))
				}
				if sub.OutputRequirement != "" {
					sb.WriteString(fmt.Sprintf(" — 输出要求：%s", sub.OutputRequirement))
				}
				sb.WriteString("\n")
				totalDuration += sub.Duration
			}
		}
		sb.WriteString("\n")
	}
	if totalDuration > 0 {
		sb.WriteString(fmt.Sprintf("⏱ 教学过程各环节合计 %d 分钟，请确保总时长与课时一致。\n", totalDuration))
	}
	return sb.String()
}

// ==================== 变体段选择 ====================

func selectPromptVariant(promptVariantsJSON string, promptMode string) string {
	if promptVariantsJSON == "" || promptVariantsJSON == "{}" {
		return ""
	}
	var variants map[string]string
	if err := json.Unmarshal([]byte(promptVariantsJSON), &variants); err != nil {
		return ""
	}
	mode := promptMode
	if mode == "" || mode == models.PromptModePerStage {
		mode = models.PromptModeGuided
	}
	if text, ok := variants[mode]; ok && strings.TrimSpace(text) != "" {
		return text
	}
	if text, ok := variants[models.PromptModeGuided]; ok {
		return text
	}
	return ""
}

// ==================== 对话规范指引（v75替换原buildTechSegment）====================

// buildDialogueGuidelines 构建对话规范指引（替代旧的buildTechSegment）
//
// v75新增：不再要求AI输出任何XML标签或JSON格式。
// 只追加自然语言对话规范和各阶段的输出格式提示。
// AI所有输出直接展示给用户，无需后端解析标签。
func buildDialogueGuidelines(stageCode string) string {
	base := `== 对话规范 ==
1. 请用自然语言与老师对话，所有内容直接输出，不要使用任何XML标签（如<stage_output>等）。
2. 不要输出JSON格式的结构化数据，老师看不懂这些格式。
3. 你的所有回复内容都会直接展示给老师，请确保内容对老师友好且有价值。
4. 阶段是否完成由老师手动点击"完成本阶段"按钮决定，你不需要判断阶段是否结束。
`

	switch stageCode {
	case "analyze":
		return base + `
本阶段特殊规范：
- 通过对话了解学情和教学需求，直接输出你的分析和建议
- 可以引用课标、学生特征等进行分析讨论
- 当分析充分时告诉老师可以进入下一阶段，但不要强制
`

	case "design":
		return base + `
本阶段特殊规范：
- 基于前序分析成果，与老师讨论教学设计方案
- 可以提供多个方案选项供老师选择，用自然语言描述每个方案的优劣
- 讨论教学目标、重难点、教学策略、活动设计等
- 当设计方案确定后告诉老师可以进入下一阶段
`

	case "write":
		return base + `
本阶段特殊规范：
- 当你准备好输出教案时，请直接输出完整的Markdown格式教案
- 教案用Markdown格式书写，使用 # 一级标题、## 二级标题、### 三级标题等层次结构
- 必须包含的结构：教学目标、教学重难点、教学准备、教学过程（含各环节时间分配）、作业布置、板书设计
- 教案输出后，继续与老师讨论是否需要局部修改
- 如果老师要求修改某个部分，可以只输出修改后的该部分（无需重复输出整篇教案）
`

	case "review":
		return base + `
本阶段特殊规范：
- 直接输出评审报告，包含：总评分(满分10分)、各维度评分和点评、优点、改进建议
- 评审维度包括：教学目标(T1)、教学内容(T2)、教学方法(T3)、教学评价(T4)
- 每个维度给出具体分数和简短评语
- 改进建议要具体可操作，指出具体位置和修改方向
- 评审完成后等待老师确认，不要主动修改教案
`

	case "revise":
		return base + `
本阶段特殊规范：
- 先列出修改清单（哪些地方需要改、为什么改、怎么改）
- 与老师确认修改方案后，输出修订后的完整Markdown教案
- 修订后的教案同样使用 # ## ### 等Markdown层次结构
- 修订说明可以在教案前面简要列出，然后紧接完整教案
`

	default:
		return base
	}
}

// ==================== 配方全局上下文 ====================

func BuildStageGlobalContext(recipe *models.TeachingRecipe) string {
	var sb strings.Builder
	hasContent := false
	sb.WriteString("=== 配方全局信息 ===\n")
	if strings.TrimSpace(recipe.StudentProfile) != "" {
		sb.WriteString(fmt.Sprintf("\n【学情档案】\n%s\n", recipe.StudentProfile))
		hasContent = true
	}
	if strings.TrimSpace(recipe.TeachingStyle) != "" {
		sb.WriteString(fmt.Sprintf("\n【教学风格偏好】\n%s\n", recipe.TeachingStyle))
		hasContent = true
	}
	if strings.TrimSpace(recipe.SchoolRequirements) != "" {
		sb.WriteString(fmt.Sprintf("\n【学校要求】\n%s\n", recipe.SchoolRequirements))
		hasContent = true
	}
	if strings.TrimSpace(recipe.CustomNotes) != "" {
		sb.WriteString(fmt.Sprintf("\n【备课心得】\n%s\n", recipe.CustomNotes))
		hasContent = true
	}
	if strings.TrimSpace(recipe.CustomPrompt) != "" {
		sb.WriteString(fmt.Sprintf("\n【自定义指令】\n%s\n", recipe.CustomPrompt))
		hasContent = true
	}
	if !hasContent {
		return ""
	}
	return sb.String()
}

// ==================== 前序阶段产出物上下文 ====================

func BuildPriorOutputsContext(outputs []*models.WorkshopStageOutput) string {
	if len(outputs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("=== 前序阶段产出 ===\n")
	for _, out := range outputs {
		stageName := stageCodeToName(out.StageCode)
		if out.Status == models.StageOutputSkipped {
			sb.WriteString(fmt.Sprintf("\n【阶段%d — %s】（已跳过）\n", out.StageOrder, stageName))
			continue
		}
		if out.Status != models.StageOutputCompleted {
			continue
		}
		sb.WriteString(fmt.Sprintf("\n【阶段%d — %s】\n", out.StageOrder, stageName))
		if out.StructuredOutput != "" && out.StructuredOutput != "{}" {
			sb.WriteString(out.StructuredOutput)
			sb.WriteString("\n")
		}
		if strings.TrimSpace(out.NarrativeOutput) != "" {
			narrative := safeUTF8Truncate(out.NarrativeOutput, 500)
			sb.WriteString(fmt.Sprintf("总结：%s\n", narrative))
		}
	}
	return sb.String()
}

// ==================== 阶段专属组件上下文 ====================

func BuildStageComponentContext(ctx context.Context, recipe *models.TeachingRecipe, componentTypesJSON string) string {
	var stageTypes []string
	if err := json.Unmarshal([]byte(componentTypesJSON), &stageTypes); err != nil || len(stageTypes) == 0 {
		return ""
	}
	var allComponentIDs []string
	if err := json.Unmarshal([]byte(recipe.ComponentIDs), &allComponentIDs); err != nil || len(allComponentIDs) == 0 {
		return ""
	}
	groups, err := repository.GetRecipeComponentContents(ctx, allComponentIDs)
	if err != nil || len(groups) == 0 {
		return ""
	}
	typeSet := make(map[string]bool)
	for _, t := range stageTypes {
		typeSet[t] = true
	}
	var sb strings.Builder
	hasContent := false
	for _, g := range groups {
		if !typeSet[g.LibraryType] {
			continue
		}
		if !hasContent {
			sb.WriteString("=== 本阶段参考资料 ===\n")
			hasContent = true
		}
		sb.WriteString(fmt.Sprintf("\n【%s】\n", g.LibraryName))
		for _, c := range g.Components {
			sb.WriteString(fmt.Sprintf("▸ %s\n", c.DisplayLabel))
			if c.DesignLogic != "" {
				sb.WriteString(fmt.Sprintf("  设计逻辑：%s\n", c.DesignLogic))
			}
			if c.FullGuide != "" {
				guide := c.FullGuide
				if len(guide) > 1000 {
					guide = guide[:1000] + "...(已截断)"
				}
				sb.WriteString(fmt.Sprintf("  完整指引：%s\n", guide))
			}
		}
	}
	return sb.String()
}

// ==================== 阶段内对话提示词 ====================

func BuildStageChatPrompt(
	lp *models.LessonPlan,
	stageHistory []*models.ConversationMessage,
	userMsg *models.ConversationMessage,
) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【当前备课信息】\n学科：%s\n年级：%s\n课题：%s\n课时：%d分钟\n\n",
		lp.Subject, lp.Grade, lp.Topic, lp.DurationMinutes))
	if lp.ContentMarkdown != "" {
		sb.WriteString("【已有教案内容】\n")
		content := lp.ContentMarkdown
		if len(content) > 3000 {
			content = content[:3000] + "\n...(教案内容已截断)"
		}
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}
	recentHistory := stageHistory
	if len(recentHistory) > 15 {
		recentHistory = recentHistory[len(recentHistory)-15:]
	}
	if len(recentHistory) > 0 {
		sb.WriteString("【本阶段对话记录】\n")
		for _, h := range recentHistory {
			role := "教师"
			if h.Role == models.ConvRoleAssistant {
				role = "AI助手"
			}
			sb.WriteString(fmt.Sprintf("%s：%s\n", role, h.Content))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("教师：%s\n\nAI助手：", userMsg.Content))
	return sb.String()
}

// ==================== 阶段开场白提示词 ====================

func BuildStageOpeningPrompt(
	lp *models.LessonPlan,
	stage *models.WorkshopStage,
	stageOrder int,
	totalStages int,
) string {
	return fmt.Sprintf(`教师正在进行阶段化备课，现在进入第%d/%d个阶段。

备课信息：
学科：%s
年级：%s
课题：%s
课时：%d分钟

当前阶段：%s（%s）
你的角色：%s

请用友好的对话方式开场，简要说明本阶段的目标和你能帮助老师做什么。
不要超过150字，用自然的口吻。如果有前序阶段的成果，简要提及将如何在本阶段利用。`,
		stageOrder, totalStages,
		lp.Subject, lp.Grade, lp.Topic, lp.DurationMinutes,
		stage.StageName, stage.StageCode,
		stage.AIRole,
	)
}

// ==================== v75新增：从自然语言回复中提取结构化数据 ====================

// ExtractStructuredFromNaturalReply 从AI自然语言回复中提取结构化信息
//
// v75新增：AI不再输出XML标签，所有内容都是自然语言。
// 后端从自然语言中检测并提取关键结构化数据。
//
// 策略：
//   write/revise阶段：检测是否包含完整教案（Markdown标题结构），提取content_markdown
//   review阶段：检测评审报告特征，提取总分和维度
//   其他阶段：将AI回复的前500字符作为narrative保存
//
// 返回值：
//   structuredJSON: 结构化数据JSON字符串（可能为"{}"表示未提取到）
//   narrative: 叙事性摘要文本
//   hasContent: 是否成功提取到有意义的内容
func ExtractStructuredFromNaturalReply(stageCode string, content string) (structuredJSON string, narrative string, hasContent bool) {
	switch stageCode {
	case "write", "revise":
		return extractWriteStageFromNatural(content)
	case "review":
		return extractReviewStageFromNatural(content)
	case "analyze", "design":
		return extractGenericStageFromNatural(stageCode, content)
	default:
		return extractGenericStageFromNatural(stageCode, content)
	}
}

// extractWriteStageFromNatural 从write/revise阶段的自然语言回复中提取教案内容
//
// 检测逻辑：AI回复中包含Markdown教案的标志性标题（如"# 教案"、"## 教学目标"等）
// 提取逻辑：从第一个 # 标题开始，到回复末尾（或下一个明显的对话分隔）作为教案内容
func extractWriteStageFromNatural(content string) (string, string, bool) {
	lessonContent := DetectLessonPlanContent(content)
	if lessonContent == "" {
		// 没有检测到教案内容，保存narrative
		narrative := safeUTF8Truncate(content, 500)
		return "{}", narrative, false
	}

	// 成功提取到教案内容
	structured := map[string]interface{}{
		"content_markdown": lessonContent,
	}
	b, _ := json.Marshal(structured)

	// narrative = 教案内容之前的对话文本
	narrativeIdx := strings.Index(content, lessonContent)
	narrative := ""
	if narrativeIdx > 0 {
		narrative = strings.TrimSpace(content[:narrativeIdx])
	}
	if narrative == "" {
		narrative = fmt.Sprintf("已生成教案（%d字符）", len(lessonContent))
	}

	wsLog.Info("从自然语言回复中提取到教案内容",
		"content_len", len(lessonContent),
		"narrative_len", len(narrative),
	)

	return string(b), narrative, true
}

// DetectLessonPlanContent 检测并提取AI回复中的完整教案Markdown内容
//
// v75新增：AI现在直接用自然语言输出教案，不再包裹在标签中。
// 本函数检测回复中是否包含完整的教案结构，如果是则提取出来。
//
// 检测标志（满足任意2个即认为包含教案）：
//   - # 或 ## 开头的教案相关标题（教案/教学目标/教学重点/教学过程等）
//   - 包含"教学目标"和"教学过程"关键词
//
// 提取规则：
//   从第一个教案标题行开始，一直到内容结尾
func DetectLessonPlanContent(content string) string {
	if content == "" {
		return ""
	}

	// 教案标志性关键词（用于确认这是教案而非普通讨论）
	lessonMarkers := []string{
		"教学目标", "教学重点", "教学难点", "教学重难点",
		"教学过程", "教学准备", "作业布置", "板书设计",
		"教学方法", "教学评价", "课时安排",
	}

	// 统计出现的标志词数量
	markerCount := 0
	for _, marker := range lessonMarkers {
		if strings.Contains(content, marker) {
			markerCount++
		}
	}

	// 至少命中3个标志词才认为包含完整教案
	if markerCount < 3 {
		return ""
	}

	// 查找第一个教案相关的 Markdown 标题行
	lines := strings.Split(content, "\n")
	startIdx := -1

	// 教案标题的识别模式：# 开头的行，包含教案相关关键词
	titleMarkers := []string{
		"教案", "教学设计", "教学目标", "课题", "课时",
		"教学重点", "教学难点", "教学重难点", "教学准备",
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 检查是否是 Markdown 标题行（# 开头）
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		// 检查标题是否包含教案相关关键词
		for _, marker := range titleMarkers {
			if strings.Contains(trimmed, marker) {
				startIdx = i
				break
			}
		}
		if startIdx >= 0 {
			break
		}
	}

	if startIdx < 0 {
		// 没找到标题行，但标志词足够多，尝试从内容开头提取
		// 可能AI没有用#标题而是用了其他格式
		return ""
	}

	// 从标题行开始到末尾作为教案内容
	lessonLines := lines[startIdx:]
	result := strings.TrimSpace(strings.Join(lessonLines, "\n"))

	// 清理末尾可能的对话尾巴（如"如果您有任何修改意见..."等常见AI客套话）
	result = trimTrailingChatter(result)

	if len(result) < 100 {
		// 太短不算完整教案
		return ""
	}

	return result
}

// trimTrailingChatter 去掉教案末尾的AI客套话
//
// AI经常在教案输出后追加类似"如有需要请告诉我"之类的话，
// 这些不属于教案正文，需要清除。
func trimTrailingChatter(content string) string {
	// 常见的AI尾巴模式
	chatterPrefixes := []string{
		"如果您有任何", "如果你有任何", "如有任何",
		"如果您觉得", "如果你觉得",
		"如果需要修改", "如需修改", "如需调整",
		"希望这份教案", "以上是", "以上就是",
		"如果有其他", "如有其他",
		"您可以点击", "你可以点击",
		"请问还有", "还有什么",
		"---\n\n如果", "---\n\n以上", "---\n\n希望",
	}

	lines := strings.Split(content, "\n")

	// 从末尾往前扫描，找到最后一行非客套话的内容
	trimEnd := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || trimmed == "---" {
			trimEnd = i
			continue
		}
		isChatter := false
		for _, prefix := range chatterPrefixes {
			if strings.HasPrefix(trimmed, prefix) {
				isChatter = true
				break
			}
		}
		if isChatter {
			trimEnd = i
			continue
		}
		// 遇到正常内容行，停止向前扫描
		break
	}

	return strings.TrimSpace(strings.Join(lines[:trimEnd], "\n"))
}

// extractReviewStageFromNatural 从review阶段的自然语言回复中提取评审信息
//
// 尝试提取：总分、各维度分数、改进建议
// 格式灵活匹配：支持"总分：8.5"、"总评分：8.5/10"等多种自然语言表述
func extractReviewStageFromNatural(content string) (string, string, bool) {
	// 尝试提取总分
	totalScore := extractScoreFromText(content, []string{
		"总评分", "总分", "综合评分", "综合得分", "总体评分",
	})

	if totalScore <= 0 {
		// 未找到评分，作为普通narrative保存
		narrative := safeUTF8Truncate(content, 500)
		return "{}", narrative, false
	}

	// 尝试提取各维度分数
	dimensions := []map[string]interface{}{}
	dimDefs := []struct {
		code    string
		name    string
		aliases []string
	}{
		{"T1", "教学目标", []string{"教学目标", "目标设计", "目标"}},
		{"T2", "教学内容", []string{"教学内容", "内容设计", "内容"}},
		{"T3", "教学方法", []string{"教学方法", "方法设计", "教学策略", "方法"}},
		{"T4", "教学评价", []string{"教学评价", "评价设计", "评价"}},
	}
	for _, dim := range dimDefs {
		score := extractScoreFromText(content, dim.aliases)
		if score > 0 {
			dimensions = append(dimensions, map[string]interface{}{
				"code":  dim.code,
				"name":  dim.name,
				"score": score,
			})
		}
	}

	structured := map[string]interface{}{
		"total_score": totalScore,
	}
	if len(dimensions) > 0 {
		structured["dimensions"] = dimensions
	}

	b, _ := json.Marshal(structured)

	// narrative = 完整评审文本的摘要
	narrative := safeUTF8Truncate(content, 500)

	wsLog.Info("从自然语言回复中提取到评审信息",
		"total_score", totalScore,
		"dimensions_count", len(dimensions),
	)

	return string(b), narrative, true
}

// extractScoreFromText 从文本中提取特定关键词后的分数
//
// 支持格式：
//   "总评分：8.5" / "总评分：8.5/10" / "总评分：8.5分" / "总分 8.5"
//   "教学目标（T1）：9.0" / "T1 教学目标：9.0分"
func extractScoreFromText(text string, keywords []string) float64 {
	for _, kw := range keywords {
		idx := strings.Index(text, kw)
		if idx == -1 {
			continue
		}
		// 从关键词后面查找数字
		after := text[idx+len(kw):]
		// 跳过分隔符和空白（使用rune遍历以正确处理中文字符）
		runes := []rune(after)
		ri := 0
		for ri < len(runes) {
			r := runes[ri]
			if r == ':' || r == '：' || r == ' ' || r == '\t' ||
				r == '(' || r == ')' || r == '（' || r == '）' {
				ri++
				continue
			}
			break
		}
		if ri >= len(runes) {
			continue
		}

		// 尝试读取数字（包括小数点）
		numStr := ""
		for j := ri; j < len(runes); j++ {
			r := runes[j]
			if (r >= '0' && r <= '9') || r == '.' {
				numStr += string(r)
			} else {
				break
			}
		}
		if numStr == "" {
			continue
		}

		var score float64
		if _, err := fmt.Sscanf(numStr, "%f", &score); err == nil && score > 0 && score <= 10 {
			return score
		}
	}
	return 0
}

// extractGenericStageFromNatural 通用阶段（analyze/design等）从自然语言中提取
//
// 这些阶段不需要特殊的结构化提取，只保存narrative摘要
func extractGenericStageFromNatural(stageCode string, content string) (string, string, bool) {
	if strings.TrimSpace(content) == "" {
		return "{}", "", false
	}

	narrative := safeUTF8Truncate(content, 500)

	// 构造简单的structured，包含阶段标识
	structured := map[string]interface{}{
		"stage":   stageCode,
		"summary": narrative,
	}
	b, _ := json.Marshal(structured)

	return string(b), narrative, true
}

// ==================== 废弃函数（保留签名，兼容性需要）====================

// ParseStageOutput 从AI回复中解析<stage_output>标签内的JSON
//
// v75废弃：AI不再输出<stage_output>标签，此函数保留签名但功能降级。
// 如果回复中恰好包含标签（兼容旧行为），仍然可以解析。
// 正常情况下应使用 ExtractStructuredFromNaturalReply 替代。
func ParseStageOutput(content string) (structuredJSON string, narrativeText string, found bool) {
	startTag := "<stage_output>"
	endTag := "</stage_output>"
	startIdx := strings.Index(content, startTag)
	if startIdx == -1 {
		// v75：没有标签是正常情况，直接返回未找到
		return "", "", false
	}
	endIdx := strings.Index(content, endTag)
	if endIdx == -1 || endIdx <= startIdx {
		return "", "", false
	}
	rawJSON := strings.TrimSpace(content[startIdx+len(startTag) : endIdx])
	if rawJSON == "" {
		return "", "", false
	}

	// narrative = 标签前的自然语言内容
	narrativeText = strings.TrimSpace(content[:startIdx])

	// 策略1：标准JSON解析
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err == nil {
		structuredData := "{}"
		if s, ok := parsed["structured"]; ok {
			if b, err2 := json.Marshal(s); err2 == nil {
				structuredData = string(b)
			}
		}
		if narrativeText == "" {
			if n, ok := parsed["narrative"]; ok {
				if ns, ok := n.(string); ok {
					narrativeText = ns
				}
			}
		}
		return structuredData, narrativeText, true
	}

	// 策略2：括号计数法提取structured
	wsLog.Warn("stage_output JSON解析失败，启用容错提取", "raw_len", len(rawJSON))
	structuredData := extractStructuredFallback(rawJSON)

	// 策略3：逐字段提取
	if structuredData == "{}" {
		wsLog.Warn("括号计数法也失败，启用逐字段提取", "raw_len", len(rawJSON))
		structuredData = extractStructuredByFields(rawJSON)
	}

	return structuredData, narrativeText, true
}

// extractStructuredFallback 策略2：括号计数法提取structured字段
// v75：保留，供降级提取路径使用
func extractStructuredFallback(rawJSON string) string {
	key := `"structured":`
	idx := strings.Index(rawJSON, key)
	if idx == -1 {
		return "{}"
	}
	rest := rawJSON[idx+len(key):]
	start := strings.Index(rest, "{")
	if start == -1 {
		return "{}"
	}
	absStart := idx + len(key) + start

	depth := 0
	inString := false
	escaped := false
	for i := absStart; i < len(rawJSON); i++ {
		c := rawJSON[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				extracted := rawJSON[absStart : i+1]
				var test map[string]interface{}
				if err := json.Unmarshal([]byte(extracted), &test); err == nil {
					result, _ := json.Marshal(test)
					return string(result)
				}
				return extracted
			}
		}
	}
	return "{}"
}

// extractStructuredByFields 策略3：逐字段提取重组structured
// v75：保留，供降级提取路径使用
func extractStructuredByFields(rawJSON string) string {
	if strings.Contains(rawJSON, `"content_markdown"`) {
		cm := extractFieldFromText(rawJSON, "content_markdown")
		if cm != "" {
			result := map[string]interface{}{
				"content_markdown": cm,
			}
			if strings.Contains(rawJSON, `"content_structured"`) {
				csKey := `"content_structured":`
				csIdx := strings.Index(rawJSON, csKey)
				if csIdx >= 0 {
					csRest := rawJSON[csIdx+len(csKey):]
					csStart := strings.Index(csRest, "{")
					if csStart >= 0 {
						depth := 0
						for i := csStart; i < len(csRest); i++ {
							if csRest[i] == '{' {
								depth++
							} else if csRest[i] == '}' {
								depth--
								if depth == 0 {
									var cs map[string]interface{}
									if err := json.Unmarshal([]byte(csRest[csStart:i+1]), &cs); err == nil {
										result["content_structured"] = cs
									}
									break
								}
							}
						}
					}
				}
			}
			b, _ := json.Marshal(result)
			wsLog.Info("逐字段提取write阶段structured成功", "content_len", len(cm))
			return string(b)
		}
	}

	if strings.Contains(rawJSON, `"textbook_analysis"`) {
		ta := extractFieldFromText(rawJSON, "textbook_analysis")
		if ta != "" {
			result := map[string]interface{}{"textbook_analysis": ta}
			if strings.Contains(rawJSON, `"key_concepts"`) {
				kcIdx := strings.Index(rawJSON, `"key_concepts"`)
				if kcIdx >= 0 {
					kcRest := rawJSON[kcIdx+len(`"key_concepts"`):]
					arrStart := strings.Index(kcRest, "[")
					if arrStart >= 0 {
						depth := 0
						for i := arrStart; i < len(kcRest); i++ {
							if kcRest[i] == '[' {
								depth++
							} else if kcRest[i] == ']' {
								depth--
								if depth == 0 {
									var kc []interface{}
									if err := json.Unmarshal([]byte(kcRest[arrStart:i+1]), &kc); err == nil {
										result["key_concepts"] = kc
									}
									break
								}
							}
						}
					}
				}
			}
			b, _ := json.Marshal(result)
			return string(b)
		}
	}

	if strings.Contains(rawJSON, `"strategy"`) {
		st := extractFieldFromText(rawJSON, "strategy")
		if st != "" {
			result := map[string]interface{}{"strategy": st}
			b, _ := json.Marshal(result)
			return string(b)
		}
	}

	wsLog.Warn("逐字段提取也失败，返回{}", "raw_len", len(rawJSON))
	return "{}"
}

// DetectStageComplete 检测AI回复中是否包含<stage_complete/>标签
//
// v75废弃：AI不再输出<stage_complete/>标签。
// 保留签名以兼容可能残留的调用，但正常流程不再依赖此函数。
func DetectStageComplete(content string) bool {
	return strings.Contains(content, "<stage_complete/>") ||
		strings.Contains(content, "<stage_complete />")
}

// CleanStageMarkers 清除AI回复中的阶段标记标签
//
// v75废弃：AI不再输出标签，此函数通常不会清除任何内容。
// 保留签名以兼容可能残留的调用。
func CleanStageMarkers(content string) string {
	startTag := "<stage_output>"
	endTag := "</stage_output>"
	for {
		startIdx := strings.Index(content, startTag)
		if startIdx == -1 {
			break
		}
		endIdx := strings.Index(content, endTag)
		if endIdx == -1 {
			break
		}
		content = content[:startIdx] + content[endIdx+len(endTag):]
	}
	content = strings.ReplaceAll(content, "<stage_complete/>", "")
	content = strings.ReplaceAll(content, "<stage_complete />", "")
	return strings.TrimSpace(content)
}

// ==================== 辅助函数 ====================

// AutoMatchStageComponents 自动匹配阶段组件（配方无组件时的兜底方案）
//
// v74新增：当配方没有选择组件，或配方组件中没有匹配当前阶段类型的组件时，
// 自动从组件库按学科+年级+阶段需要的组件类型匹配，每种类型取质量分最高的2个。
func AutoMatchStageComponents(ctx context.Context, componentTypesJSON string, subject string, grade string) string {
	var stageTypes []string
	if err := json.Unmarshal([]byte(componentTypesJSON), &stageTypes); err != nil || len(stageTypes) == 0 {
		return ""
	}

	// v74：将中文年级转换为数字格式，确保年级范围匹配生效
	normalizedGrade := normalizeGradeToNumber(grade)

	groups, err := repository.MatchComponents(ctx, &models.MatchComponentsRequest{
		Subject:      subject,
		GradeRange:   normalizedGrade,
		LibraryTypes: stageTypes,
		Limit:        2,
	})
	if err != nil || len(groups) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("=== 本阶段参考资料（系统自动匹配）===\n")
	sb.WriteString("以下是根据学科和年级自动匹配的教学参考组件，请在本阶段工作中适当参考：\n")

	for _, g := range groups {
		sb.WriteString(fmt.Sprintf("\n【%s】\n", g.LibraryName))
		for _, c := range g.Components {
			sb.WriteString(fmt.Sprintf("▸ %s\n", c.DisplayLabel))
			if c.DesignLogic != "" {
				sb.WriteString(fmt.Sprintf("  设计逻辑：%s\n", c.DesignLogic))
			}
			if c.FullGuide != "" {
				guide := c.FullGuide
				if len(guide) > 1000 {
					guide = guide[:1000] + "...(已截断)"
				}
				sb.WriteString(fmt.Sprintf("  完整指引：%s\n", guide))
			}
		}
	}

	wsPromptLog := logger.WithModule("workshop_stage_prompts")
	wsPromptLog.Info("自动匹配阶段组件",
		"subject", subject,
		"grade", grade,
		"stage_types", stageTypes,
		"matched_groups", len(groups),
	)

	return sb.String()
}

// normalizeGradeToNumber 将中文年级名转换为数字格式
//
// v74新增：组件库的 grade_range 使用纯数字格式（如"7-9""3-6"），
// 但教案的 grade 字段可能是中文（如"七年级""三年级"）。
// 此函数将中文年级转为阿拉伯数字字符串，确保年级匹配生效。
func normalizeGradeToNumber(grade string) string {
	// 先尝试直接提取数字
	var digits []byte
	for _, b := range []byte(grade) {
		if b >= '0' && b <= '9' {
			digits = append(digits, b)
		}
	}
	if len(digits) > 0 {
		return string(digits)
	}

	// 中文数字映射
	cnMap := map[string]string{
		"一": "1", "二": "2", "三": "3", "四": "4", "五": "5",
		"六": "6", "七": "7", "八": "8", "九": "9", "十": "10",
		"十一": "11", "十二": "12",
	}

	// 初中/高中别名映射
	aliasMap := map[string]string{
		"初一": "7", "初二": "8", "初三": "9",
		"高一": "10", "高二": "11", "高三": "12",
	}

	// 优先匹配别名
	for alias, num := range aliasMap {
		if strings.Contains(grade, alias) {
			return num
		}
	}

	// 再匹配中文数字（先匹配两字数字，再匹配单字）
	for cn, num := range cnMap {
		if len(cn) > 3 && strings.Contains(grade, cn) {
			return num
		}
	}
	for cn, num := range cnMap {
		if strings.Contains(grade, cn) {
			return num
		}
	}

	return grade
}

// stageCodeToName 阶段代码转中文名
func stageCodeToName(code string) string {
	nameMap := map[string]string{
		"analyze": "教学分析",
		"design":  "教学设计",
		"write":   "教案撰写",
		"review":  "AI评审",
		"revise":  "修订定稿",
	}
	if name, ok := nameMap[code]; ok {
		return name
	}
	return code
}

// generateStageOpeningMsgID 生成阶段开场消息ID
func generateStageOpeningMsgID(stageCode string) string {
	return fmt.Sprintf("msg_stage_%s_%d", stageCode, time.Now().UnixNano())
}

// extractFieldFromText 从任意文本中提取指定JSON字段的字符串值
//
// 算法：找到 "fieldName": 位置，跳过空白到值的开始引号，
// 逐字节读取处理转义序列，验证结束引号后的后续字符确认真正结束
// v75：保留，供降级提取路径使用
func extractFieldFromText(text, fieldName string) string {
	key := `"` + fieldName + `"`
	keyIdx := strings.Index(text, key)
	if keyIdx == -1 {
		return ""
	}

	afterKey := text[keyIdx+len(key):]
	colonIdx := strings.Index(afterKey, ":")
	if colonIdx == -1 {
		return ""
	}

	afterColon := afterKey[colonIdx+1:]
	valueStart := -1
	for i, ch := range afterColon {
		if ch == '"' {
			valueStart = i
			break
		}
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			return ""
		}
	}
	if valueStart == -1 {
		return ""
	}

	content := afterColon[valueStart+1:]
	var sb strings.Builder
	i := 0
	for i < len(content) {
		b := content[i]

		if b == '\\' && i+1 < len(content) {
			next := content[i+1]
			switch next {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			case '/':
				sb.WriteByte('/')
			case 'u':
				if i+5 < len(content) {
					hex := content[i+2 : i+6]
					var codePoint rune
					fmt.Sscanf(hex, "%04x", &codePoint)
					sb.WriteRune(codePoint)
					i += 6
					continue
				}
				sb.WriteByte(next)
			default:
				sb.WriteByte(next)
			}
			i += 2
			continue
		}

		if b == '"' {
			rest := content[i+1:]
			rest = strings.TrimLeft(rest, " \t\r\n")
			if len(rest) == 0 {
				break
			}
			if rest[0] == ',' || rest[0] == '}' || rest[0] == ']' {
				break
			}
			if rest[0] == '"' {
				nextQuote := strings.Index(rest[1:], `"`)
				if nextQuote >= 0 {
					afterNextKey := strings.TrimLeft(rest[1+nextQuote+1:], " \t")
					if len(afterNextKey) > 0 && afterNextKey[0] == ':' {
						break
					}
				}
			}
			if strings.HasPrefix(rest, "</") || strings.HasPrefix(rest, "<stage") {
				break
			}
			sb.WriteByte(b)
			i++
			continue
		}

		sb.WriteByte(b)
		i++
	}

	return strings.TrimSpace(sb.String())
}
