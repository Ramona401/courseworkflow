package services

// workshop_stage_prompts.go — 阶段化备课工坊提示词构建
//
// 包含：六层系统提示词拼接 + 各上下文构建函数 + 对话规范 + 组件注入 + 辅助函数
//
// v76拆分：自然语言提取+废弃函数 移至 workshop_stage_extract.go
// v82变更：normalizeGradeToNumber 抽取为 utils.NormalizeGradeToNumber 统一工具函数
// v84变更：BuildStageChatPrompt 改造为分层记忆版本
//          新增 BuildStageChatPromptV2 支持 Working+Episodic 分层上下文
//          旧版 BuildStageChatPrompt 保留向下兼容（内部调用V2）

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// ==================== 阶段完整系统提示词构建（六层拼接）====================

// BuildStageSystemPrompt 构建某阶段完整的系统提示词
func BuildStageSystemPrompt(
	ctx context.Context,
	stage *models.WorkshopStage,
	recipe *models.TeachingRecipe,
	priorOutputs []*models.WorkshopStageOutput,
	subject string,
	grade string,
	promptMode string,
	lessonStructure string,
	selectedCompIDs []string,
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
	// 优先级：用户手动选中 > 配方组件 > 自动匹配
	if stage.ComponentTypes != "" && stage.ComponentTypes != "[]" {
		componentCtx := ""
		if len(selectedCompIDs) > 0 {
			componentCtx = BuildSelectedComponentContext(ctx, selectedCompIDs)
		}
		if componentCtx == "" && recipe != nil {
			componentCtx = BuildStageComponentContext(ctx, recipe, stage.ComponentTypes)
		}
		if componentCtx == "" {
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

	// 第4层：阶段角色提示词
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

	// 第5层：对话规范指引
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
			skippedStages = append(skippedStages, stageCodeToName(out.StageCode))
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

// ==================== 对话规范指引 ====================

func buildDialogueGuidelines(stageCode string) string {
	base := `== 对话规范 ==
1. 请用自然语言与老师对话，所有内容直接输出，不要使用任何XML标签（如<stage_output>等）。
2. 不要输出JSON格式的结构化数据，老师看不懂这些格式。
3. 你的所有回复内容都会直接展示给老师，请确保内容对老师友好且有价值。
4. 阶段是否完成由老师手动点击"完成本阶段"按钮决定，你不需要判断阶段是否结束。
`
	switch stageCode {
	case "analyze":
		return base + "\n本阶段特殊规范：\n- 通过对话了解学情和教学需求，直接输出你的分析和建议\n- 可以引用课标、学生特征等进行分析讨论\n- 当分析充分时告诉老师可以进入下一阶段，但不要强制\n"
	case "design":
		return base + "\n本阶段特殊规范：\n- 基于前序分析成果，与老师讨论教学设计方案\n- 可以提供多个方案选项供老师选择，用自然语言描述每个方案的优劣\n- 讨论教学目标、重难点、教学策略、活动设计等\n- 当设计方案确定后告诉老师可以进入下一阶段\n"
	case "write":
		return base + "\n本阶段特殊规范（分段确认机制）：\n- 先展示教案框架（各环节标题+时间），等老师确认\n- 每次只输出1-2个环节的详细内容，然后停下来等老师确认\n- 老师说确认/继续/可以后，再输出下一批环节\n- 所有环节输出完毕后，最后一次性输出包含全部内容的完整Markdown教案（用于系统提取保存）\n- 完整教案必须包含：教学目标、教学重难点、教学准备、教学过程（含各环节时间分配）、作业布置、板书设计\n- 如果老师中途提修改意见，先调整该环节再继续\n"
	case "review":
		return base + "\n本阶段特殊规范：\n- 直接输出评审报告，包含：总评分(满分10分)、各维度评分和点评、优点、改进建议\n- 评审维度包括：教学目标(T1)、教学内容(T2)、教学方法(T3)、教学评价(T4)\n- 每个维度给出具体分数和简短评语\n- 改进建议要具体可操作，指出具体位置和修改方向\n- 评审完成后等待老师确认，不要主动修改教案\n"
	case "revise":
		return base + "\n本阶段特殊规范：\n- 先列出修改清单（哪些地方需要改、为什么改、怎么改）\n- 与老师确认修改方案后，输出修订后的完整Markdown教案\n- 修订后的教案同样使用 # ## ### 等Markdown层次结构\n- 修订说明可以在教案前面简要列出，然后紧接完整教案\n"
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

// ==================== 用户手动选择组件上下文（迭代12）====================

func BuildSelectedComponentContext(ctx context.Context, componentIDs []string) string {
	if len(componentIDs) == 0 {
		return ""
	}
	groups, err := repository.GetRecipeComponentContents(ctx, componentIDs)
	if err != nil || len(groups) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("=== 本阶段参考资料（老师手动选择）===\n")
	sb.WriteString("以下是老师为本阶段特别选择的教学参考组件，请重点参考：\n")
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
	wsPromptLog.Info("用户手动选择的组件已注入提示词", "component_count", len(componentIDs), "matched_groups", len(groups))
	return sb.String()
}

// ==================== 自动匹配阶段组件 ====================

// AutoMatchStageComponents 根据阶段组件类型+学科+年级自动匹配组件
// v82变更：年级转换改用统一工具函数 utils.NormalizeGradeToNumber
func AutoMatchStageComponents(ctx context.Context, componentTypesJSON string, subject string, grade string) string {
	var stageTypes []string
	if err := json.Unmarshal([]byte(componentTypesJSON), &stageTypes); err != nil || len(stageTypes) == 0 {
		return ""
	}
	normalizedGrade := utils.NormalizeGradeToNumber(grade)
	groups, err := repository.MatchComponents(ctx, &models.MatchComponentsRequest{
		Subject: subject, GradeRange: normalizedGrade, LibraryTypes: stageTypes, Limit: 2,
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
			// v83: 优先用AOCI压缩索引（L2格式），无索引时降级为旧全文格式
			if c.ComponentIndex != "" {
				sb.WriteString(utils.FormatIndexForPrompt(c.ComponentIndex, c.DisplayLabel))
				sb.WriteString("\n")
			} else {
				sb.WriteString(fmt.Sprintf("▸ %s\n", c.DisplayLabel))
				if c.DesignLogic != "" {
					sb.WriteString(fmt.Sprintf("  设计逻辑：%s\n", c.DesignLogic))
				}
			}
		}
	}
	wsPromptLog := logger.WithModule("workshop_stage_prompts")
	wsPromptLog.Info("自动匹配阶段组件", "subject", subject, "grade", grade, "stage_types", stageTypes, "matched_groups", len(groups))
	return sb.String()
}

// ==================== 阶段内对话提示词（v84分层记忆改造）====================

// BuildStageChatPrompt 构建阶段内对话的用户提示词（向下兼容版本）
//
// v84变更：内部改为调用 BuildStageChatPromptV2，传入空的episodicSummary
// 保留此函数签名是为了兼容可能的外部调用
func BuildStageChatPrompt(lp *models.LessonPlan, stageHistory []*models.ConversationMessage, userMsg *models.ConversationMessage) string {
	return BuildStageChatPromptV2(lp, stageHistory, "", userMsg)
}

// BuildStageChatPromptV2 构建阶段内对话的用户提示词（v84分层记忆版）
//
// v84新增：支持三层记忆架构
//   - Working Memory: currentStageMessages — 当前阶段的完整对话（由调用方从分隔符提取）
//   - Episodic Memory: episodicSummary — 历史阶段的结构化摘要（由BuildEpisodicSummaryFromOutputs生成）
//   - Semantic Memory: 教案正文+组件+配方 — 已在systemPrompt中注入，此处不重复
//
// 与旧版 BuildStageChatPrompt 的差异：
//   1. stageHistory 现在只包含当前阶段的消息（不再是全量历史）
//   2. 新增 episodicSummary 参数，注入历史阶段摘要
//   3. 当前阶段对话截断阈值从15条提升到20条（因为不再跨阶段）
//
// 参数：
//   - lp: 教案信息
//   - currentStageMessages: 当前阶段的对话消息（Working Memory）
//   - episodicSummary: 历史阶段摘要文本（Episodic Memory，可为空）
//   - userMsg: 本轮用户输入
func BuildStageChatPromptV2(
	lp *models.LessonPlan,
	currentStageMessages []*models.ConversationMessage,
	episodicSummary string,
	userMsg *models.ConversationMessage,
) string {
	var sb strings.Builder

	// 第1部分：当前备课基本信息
	sb.WriteString(fmt.Sprintf("【当前备课信息】\n学科：%s\n年级：%s\n课题：%s\n课时：%d分钟\n\n",
		lp.Subject, lp.Grade, lp.Topic, lp.DurationMinutes))

	// 第2部分：已有教案内容（Semantic Memory的一部分）
	if lp.ContentMarkdown != "" {
		content := lp.ContentMarkdown
		if len(content) > 3000 {
			content = content[:3000] + "\n...(教案内容已截断)"
		}
		sb.WriteString("【已有教案内容】\n")
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	// 第3部分：历史阶段摘要（Episodic Memory，v84新增）
	// 只在有摘要内容时注入，避免空段干扰AI
	if strings.TrimSpace(episodicSummary) != "" {
		sb.WriteString(episodicSummary)
		sb.WriteString("\n")
	}

	// 第4部分：当前阶段对话记录（Working Memory）
	// v84改进：currentStageMessages已经只包含当前阶段的消息
	// 截断阈值从15提升到20（因为不再跨阶段，20条纯当前阶段对话更合理）
	recentHistory := currentStageMessages
	if len(recentHistory) > 20 {
		recentHistory = recentHistory[len(recentHistory)-20:]
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

	// 第5部分：本轮用户输入
	sb.WriteString(fmt.Sprintf("教师：%s\n\nAI助手：", userMsg.Content))
	return sb.String()
}

// ==================== 阶段开场白提示词 ====================

func BuildStageOpeningPrompt(lp *models.LessonPlan, stage *models.WorkshopStage, stageOrder int, totalStages int) string {
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
		stageOrder, totalStages, lp.Subject, lp.Grade, lp.Topic, lp.DurationMinutes,
		stage.StageName, stage.StageCode, stage.AIRole)
}

// ==================== 辅助函数 ====================

// stageCodeToName 阶段代码转中文名
func stageCodeToName(code string) string {
	nameMap := map[string]string{
		"analyze": "教学分析", "design": "教学设计", "write": "教案撰写",
		"review": "AI评审", "revise": "修订定稿",
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

// safeUTF8Truncate 安全截断UTF-8字符串
func safeUTF8Truncate(s string, maxChars int) string {
	runes := []rune(s)
	if len(runes) <= maxChars {
		return s
	}
	return string(runes[:maxChars]) + "..."
}
