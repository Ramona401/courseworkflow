package services

// stage_coach_rules.go — 阶段教练：规则引擎完成度检测
//
// v81新增（原stage_coach.go的规则引擎部分）
// v89-3拆分：从stage_coach.go中拆出规则引擎检测逻辑
//
// 职责：
//   1. CheckStageCompleteness — 规则引擎完成度检测（各阶段关键词+字数检测）
//   2. 各阶段专属检测规则（analyze/design/write/review/revise/generic）
//   3. countUserMessagesInStage — 统计当前阶段用户消息数
//   4. 通用检查辅助函数（关键词检测/字数检测/用户参与度/产出物读取）
//
// 被引用：
//   stage_coach.go — fallbackToRuleResult降级调用
//   handlers/workshop_stage_handler.go — GetStageCompleteness接口

import (
	"context"
	"encoding/json"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 规则引擎完成度检测 ====================

// CheckStageCompleteness 检测指定阶段的产出物完成度（规则引擎版）
// 这是原v81的核心逻辑，保留作为基础检测和LLM降级兜底
func CheckStageCompleteness(ctx context.Context, lessonPlanID string, stageCode string) (*StageCompletenessResponse, error) {
	// 获取教案信息
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}

	// 获取阶段产出物
	output, _ := repository.GetStageOutput(ctx, lessonPlanID, stageCode)

	// 统计当前阶段的用户消息数（关键：排除AI开场白的干扰）
	userMsgCount := countUserMessagesInStage(ctx, lessonPlanID, stageCode, lp)

	// 获取阶段名称
	stageName := stageCodeToName(stageCode)

	// 如果用户完全没发言，直接返回0%
	if userMsgCount == 0 {
		hints := []string{"你还没有在本阶段与AI对话，先聊聊你的想法吧"}
		return &StageCompletenessResponse{
			StageCode:    stageCode,
			StageName:    stageName,
			Percentage:   0,
			IsComplete:   false,
			UserMessages: 0,
			CheckedItems: []CompletenessItem{
				{Label: "用户参与", Passed: false, Detail: "你还没有在本阶段与AI对话，先聊聊你的想法吧"},
			},
			MissingHints: hints,
		}, nil
	}

	// 根据阶段代码分派检测规则
	var items []CompletenessItem
	switch stageCode {
	case "analyze":
		items = checkAnalyzeStage(output, userMsgCount)
	case "design":
		items = checkDesignStage(output, userMsgCount)
	case "write":
		items = checkWriteStage(output, lp, userMsgCount)
	case "review":
		items = checkReviewStage(output)
	case "revise":
		items = checkReviseStage(output, lp, userMsgCount)
	default:
		items = checkGenericStage(output, userMsgCount)
	}

	// 计算完成度百分比
	passedCount := 0
	var missingHints []string
	for _, item := range items {
		if item.Passed {
			passedCount++
		} else {
			missingHints = append(missingHints, item.Detail)
		}
	}

	percentage := 0
	if len(items) > 0 {
		percentage = passedCount * 100 / len(items)
	}

	resp := &StageCompletenessResponse{
		StageCode:    stageCode,
		StageName:    stageName,
		Percentage:   percentage,
		IsComplete:   percentage >= 80,
		UserMessages: userMsgCount,
		CheckedItems: items,
		MissingHints: missingHints,
	}

	coachLog.Info("阶段完成度检测（规则引擎）",
		"plan_id", lessonPlanID, "stage", stageCode,
		"user_msgs", userMsgCount, "percentage", percentage,
		"passed", passedCount, "total", len(items),
	)

	return resp, nil
}

// ==================== 各阶段规则检测 ====================

// checkAnalyzeStage 教学分析阶段检测
func checkAnalyzeStage(output *models.WorkshopStageOutput, userMsgs int) []CompletenessItem {
	narrative := getOutputNarrative(output)
	structured := getOutputStructured(output)
	combined := narrative + " " + structured

	return []CompletenessItem{
		checkUserEngagement(userMsgs, 2, "建议至少与AI交流2轮，讨论课程定位和学情"),
		checkContainsKeywords("教学目标", combined, []string{"教学目标", "学习目标", "目标"}, userMsgs),
		checkContainsKeywords("学情分析", combined, []string{"学情", "学生特点", "学生基础", "认知水平", "学习基础"}, userMsgs),
		checkContainsKeywords("重难点", combined, []string{"重点", "难点", "重难点", "教学重点", "教学难点"}, userMsgs),
	}
}

// checkDesignStage 教学设计阶段检测
func checkDesignStage(output *models.WorkshopStageOutput, userMsgs int) []CompletenessItem {
	narrative := getOutputNarrative(output)
	structured := getOutputStructured(output)
	combined := narrative + " " + structured

	return []CompletenessItem{
		checkUserEngagement(userMsgs, 2, "建议至少与AI交流2轮，讨论教学活动和方法"),
		checkContainsKeywords("教学活动", combined, []string{"活动", "环节", "导入", "探究", "练习", "拓展", "小组"}, userMsgs),
		checkContainsKeywords("时间分配", combined, []string{"分钟", "时间", "时长", "课时"}, userMsgs),
		checkContainsKeywords("教学方法", combined, []string{"方法", "策略", "讲授", "合作", "探究", "演示", "讨论"}, userMsgs),
	}
}

// checkWriteStage 教案撰写阶段检测
func checkWriteStage(output *models.WorkshopStageOutput, lp *models.LessonPlan, userMsgs int) []CompletenessItem {
	content := ""
	if lp != nil && lp.ContentMarkdown != "" {
		content = lp.ContentMarkdown
	}
	if content == "" {
		content = getOutputNarrative(output)
	}

	return []CompletenessItem{
		checkUserEngagement(userMsgs, 1, "建议至少确认一次教案框架"),
		checkMinLength("教案篇幅", content, 2000, "教案内容偏短，建议继续完善各教学环节"),
		checkContainsKeywords("教学过程", content, []string{"教学过程", "教学环节", "教学步骤"}, userMsgs),
		checkContainsKeywords("教学结尾", content, []string{"作业", "板书", "小结", "总结", "课堂小结", "布置作业"}, userMsgs),
	}
}

// checkReviewStage AI评审阶段检测（评审是AI自动做的，不需要用户参与度检查）
func checkReviewStage(output *models.WorkshopStageOutput) []CompletenessItem {
	structured := getOutputStructured(output)
	narrative := getOutputNarrative(output)
	combined := structured + " " + narrative

	hasScore := strings.Contains(combined, "total_score") ||
		strings.Contains(combined, "总分") ||
		strings.Contains(combined, "评分")

	hasImprovements := strings.Contains(combined, "improvements") ||
		strings.Contains(combined, "改进") ||
		strings.Contains(combined, "建议")

	hasDimensions := strings.Contains(combined, "dimensions") ||
		strings.Contains(combined, "维度")

	return []CompletenessItem{
		{Label: "评审评分", Passed: hasScore, Detail: boolHint(hasScore, "已获得评审评分", "尚未获得评审评分，请等待AI完成评审")},
		{Label: "维度分析", Passed: hasDimensions, Detail: boolHint(hasDimensions, "已完成多维度分析", "缺少多维度评分分析")},
		{Label: "改进建议", Passed: hasImprovements, Detail: boolHint(hasImprovements, "已获得改进建议", "尚未获得改进建议")},
	}
}

// checkReviseStage 修订定稿阶段检测
func checkReviseStage(output *models.WorkshopStageOutput, lp *models.LessonPlan, userMsgs int) []CompletenessItem {
	hasContent := lp != nil && len(lp.ContentMarkdown) > 2000
	hasNarrative := len(getOutputNarrative(output)) > 100

	return []CompletenessItem{
		checkUserEngagement(userMsgs, 1, "建议至少与AI讨论一轮修订方案"),
		{Label: "教案内容", Passed: hasContent, Detail: boolHint(hasContent, "教案内容已更新", "教案内容尚未更新，请与AI讨论修订方案")},
		{Label: "修订对话", Passed: hasNarrative, Detail: boolHint(hasNarrative, "已进行修订讨论", "建议与AI讨论评审中的改进建议")},
	}
}

// checkGenericStage 通用阶段检测（自定义阶段）
func checkGenericStage(output *models.WorkshopStageOutput, userMsgs int) []CompletenessItem {
	hasOutput := output != nil && output.Status != "pending"

	return []CompletenessItem{
		checkUserEngagement(userMsgs, 1, "建议至少与AI交流1轮"),
		{Label: "阶段产出", Passed: hasOutput, Detail: boolHint(hasOutput, "已有阶段产出", "尚未开始此阶段")},
	}
}

// ==================== 统计当前阶段用户消息数 ====================

// countUserMessagesInStage 统计当前阶段中用户发送的消息数
// 通过conversation_log中的阶段分隔符定位当前阶段的消息范围
// 然后计算role=user的消息数（排除系统自动触发的指令消息）
func countUserMessagesInStage(ctx context.Context, lessonPlanID string, stageCode string, lp *models.LessonPlan) int {
	// 读取对话记录（返回 []*models.ConversationMessage）
	messages, err := repository.GetConversationLog(ctx, lessonPlanID)
	if err != nil || len(messages) == 0 {
		return 0
	}

	// 获取阶段配置，用于定位阶段分隔符
	var snapshots []models.StageConfigSnapshot
	if lp.StageConfig != "" && lp.StageConfig != "[]" {
		_ = json.Unmarshal([]byte(lp.StageConfig), &snapshots)
	}

	// 构建阶段名→阶段代码映射
	stageNameToCode := make(map[string]string)
	for _, snap := range snapshots {
		stageNameToCode[snap.StageName] = snap.StageCode
	}

	// 扫描消息，找到当前阶段的范围
	startIdx := -1
	endIdx := len(messages)
	firstStageCode := ""
	if len(snapshots) > 0 {
		firstStageCode = snapshots[0].StageCode
	}

	for i, msg := range messages {
		if string(msg.Role) == "system" && strings.HasPrefix(msg.Content, "__STAGE_SEP__") {
			// 解析分隔符：__STAGE_SEP__阶段名__AI角色
			rest := msg.Content[len("__STAGE_SEP__"):]
			parts := strings.SplitN(rest, "__", 2)
			sepStageName := ""
			if len(parts) > 0 {
				sepStageName = parts[0]
			}

			// 查找匹配的阶段代码
			sepStageCode := ""
			if code, ok := stageNameToCode[sepStageName]; ok {
				sepStageCode = code
			}

			if sepStageCode == stageCode {
				startIdx = i // 当前阶段分隔符位置
			} else if startIdx >= 0 && endIdx == len(messages) {
				endIdx = i // 下一个分隔符位置
			}
		}
	}

	// 如果是第一个阶段且没有找到分隔符，从头开始
	if startIdx < 0 && stageCode == firstStageCode {
		startIdx = 0
		// 找第一个分隔符作为结束
		for i, msg := range messages {
			if string(msg.Role) == "system" && strings.HasPrefix(msg.Content, "__STAGE_SEP__") {
				endIdx = i
				break
			}
		}
	}

	if startIdx < 0 {
		return 0
	}

	// 自动触发消息的前缀（不算用户真正的发言）
	autoTriggerPrefixes := []string{
		"我们进入",
		"请对上一阶段完成的教案进行全面专业评审",
	}

	// 统计范围内的用户消息数
	count := 0
	for i := startIdx; i < endIdx && i < len(messages); i++ {
		if string(messages[i].Role) != "user" {
			continue
		}
		// 排除系统自动触发的指令消息
		isAutoTrigger := false
		for _, prefix := range autoTriggerPrefixes {
			if strings.HasPrefix(messages[i].Content, prefix) {
				isAutoTrigger = true
				break
			}
		}
		if !isAutoTrigger {
			count++
		}
	}

	return count
}

// ==================== 通用辅助函数 ====================

// getOutputNarrative 安全获取产出物的narrative
func getOutputNarrative(output *models.WorkshopStageOutput) string {
	if output == nil {
		return ""
	}
	return output.NarrativeOutput
}

// getOutputStructured 安全获取产出物的structured输出（JSON字符串展平）
func getOutputStructured(output *models.WorkshopStageOutput) string {
	if output == nil || output.StructuredOutput == "" || output.StructuredOutput == "{}" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(output.StructuredOutput), &m); err == nil {
		b, _ := json.Marshal(m)
		return string(b)
	}
	return output.StructuredOutput
}

// checkUserEngagement 检查用户参与度（对话轮次门槛）
func checkUserEngagement(userMsgs int, minRounds int, hint string) CompletenessItem {
	if userMsgs >= minRounds {
		return CompletenessItem{
			Label:  "对话参与",
			Passed: true,
			Detail: "已与AI充分交流",
		}
	}
	return CompletenessItem{
		Label:  "对话参与",
		Passed: false,
		Detail: hint,
	}
}

// checkContainsKeywords 检查文本是否包含指定关键词组中的至少一个
// 额外条件：用户消息数≥1才有意义（避免AI开场白中的关键词误判）
func checkContainsKeywords(label string, text string, keywords []string, userMsgs int) CompletenessItem {
	if userMsgs < 1 {
		return CompletenessItem{Label: label, Passed: false, Detail: "缺少" + label + "相关内容，请先与AI对话"}
	}
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return CompletenessItem{Label: label, Passed: true, Detail: "已包含" + label + "相关内容"}
		}
	}
	return CompletenessItem{Label: label, Passed: false, Detail: "缺少" + label + "相关内容，建议补充"}
}

// checkMinLength 检查文本是否达到最低字数
func checkMinLength(label string, text string, minLen int, hint string) CompletenessItem {
	textLen := len([]rune(strings.TrimSpace(text)))
	if textLen >= minLen {
		return CompletenessItem{Label: label, Passed: true, Detail: label + "已达标"}
	}
	return CompletenessItem{Label: label, Passed: false, Detail: hint}
}

// boolHint 根据布尔值返回不同提示
func boolHint(passed bool, passMsg string, failMsg string) string {
	if passed {
		return passMsg
	}
	return failMsg
}

// GetLessonPlanForCheck 获取教案用于权限检查（供handler层调用）
func GetLessonPlanForCheck(ctx context.Context, planID string) (*models.LessonPlan, error) {
	return repository.GetLessonPlanByID(ctx, planID)
}
