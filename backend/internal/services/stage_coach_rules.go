package services

// stage_coach_rules.go — 阶段教练：规则引擎完成度检测
//
// v81新增，v89-3拆分，v97重构：拆分countUserMessagesInStage降低认知复杂度

import (
	"context"
	"encoding/json"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 规则引擎完成度检测 ====================

// CheckStageCompleteness 检测指定阶段的产出物完成度（规则引擎版）
func CheckStageCompleteness(ctx context.Context, lessonPlanID string, stageCode string) (*StageCompletenessResponse, error) {
	lp, err := repository.GetLessonPlanByID(ctx, lessonPlanID)
	if err != nil {
		return nil, err
	}

	output, _ := repository.GetStageOutput(ctx, lessonPlanID, stageCode)
	userMsgCount := countUserMessagesInStage(ctx, lessonPlanID, stageCode, lp)
	stageName := stageCodeToName(stageCode)

	// 用户完全没发言时，检查是否已有阶段产出物
	// 如果已有产出物（如AI已生成完整教案），不应返回0%，而是走正常检测流程
	if userMsgCount == 0 {
		hasOutput := output != nil && (output.NarrativeOutput != "" || output.StructuredOutput != "")
		if !hasOutput {
			return buildZeroCompletenessResponse(stageCode, stageName), nil
		}
		// 有产出物但无用户消息：将userMsgCount视为1，走正常检测流程
		userMsgCount = 1
	}

	// 根据阶段代码分派检测规则
	items := dispatchStageCheck(stageCode, output, lp, userMsgCount)

	// 计算完成度
	return buildCompletenessResponse(stageCode, stageName, items, userMsgCount, lessonPlanID), nil
}

// buildZeroCompletenessResponse 构建0%完成度响应（用户未发言时）
func buildZeroCompletenessResponse(stageCode, stageName string) *StageCompletenessResponse {
	return &StageCompletenessResponse{
		StageCode:    stageCode,
		StageName:    stageName,
		Percentage:   0,
		IsComplete:   false,
		UserMessages: 0,
		CheckedItems: []CompletenessItem{
			{Label: "用户参与", Passed: false, Detail: "你还没有在本阶段与AI对话，先聊聊你的想法吧"},
		},
		MissingHints: []string{"你还没有在本阶段与AI对话，先聊聊你的想法吧"},
	}
}

// dispatchStageCheck 根据阶段代码分派检测规则
func dispatchStageCheck(stageCode string, output *models.WorkshopStageOutput, lp *models.LessonPlan, userMsgCount int) []CompletenessItem {
	switch stageCode {
	case "analyze":
		return checkAnalyzeStage(output, userMsgCount)
	case "design":
		return checkDesignStage(output, userMsgCount)
	case "write":
		return checkWriteStage(output, lp, userMsgCount)
	case "review":
		return checkReviewStage(output)
	case "revise":
		return checkReviseStage(output, lp, userMsgCount)
	default:
		return checkGenericStage(output, userMsgCount)
	}
}

// buildCompletenessResponse 根据检查项计算完成度并构建响应
func buildCompletenessResponse(stageCode, stageName string, items []CompletenessItem, userMsgCount int, planID string) *StageCompletenessResponse {
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

	coachLog.Info("阶段完成度检测（规则引擎）",
		"plan_id", planID, "stage", stageCode,
		"user_msgs", userMsgCount, "percentage", percentage,
		"passed", passedCount, "total", len(items),
	)

	return &StageCompletenessResponse{
		StageCode:    stageCode,
		StageName:    stageName,
		Percentage:   percentage,
		IsComplete:   percentage >= 80,
		UserMessages: userMsgCount,
		CheckedItems: items,
		MissingHints: missingHints,
	}
}

// ==================== 各阶段规则检测 ====================

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

func checkReviewStage(output *models.WorkshopStageOutput) []CompletenessItem {
	structured := getOutputStructured(output)
	narrative := getOutputNarrative(output)
	combined := structured + " " + narrative
	hasScore := strings.Contains(combined, "total_score") || strings.Contains(combined, "总分") || strings.Contains(combined, "评分")
	hasImprovements := strings.Contains(combined, "improvements") || strings.Contains(combined, "改进") || strings.Contains(combined, "建议")
	hasDimensions := strings.Contains(combined, "dimensions") || strings.Contains(combined, "维度")
	return []CompletenessItem{
		{Label: "评审评分", Passed: hasScore, Detail: boolHint(hasScore, "已获得评审评分", "尚未获得评审评分，请等待AI完成评审")},
		{Label: "维度分析", Passed: hasDimensions, Detail: boolHint(hasDimensions, "已完成多维度分析", "缺少多维度评分分析")},
		{Label: "改进建议", Passed: hasImprovements, Detail: boolHint(hasImprovements, "已获得改进建议", "尚未获得改进建议")},
	}
}

func checkReviseStage(output *models.WorkshopStageOutput, lp *models.LessonPlan, userMsgs int) []CompletenessItem {
	hasContent := lp != nil && len(lp.ContentMarkdown) > 2000
	hasNarrative := len(getOutputNarrative(output)) > 100
	return []CompletenessItem{
		checkUserEngagement(userMsgs, 1, "建议至少与AI讨论一轮修订方案"),
		{Label: "教案内容", Passed: hasContent, Detail: boolHint(hasContent, "教案内容已更新", "教案内容尚未更新，请与AI讨论修订方案")},
		{Label: "修订对话", Passed: hasNarrative, Detail: boolHint(hasNarrative, "已进行修订讨论", "建议与AI讨论评审中的改进建议")},
	}
}

func checkGenericStage(output *models.WorkshopStageOutput, userMsgs int) []CompletenessItem {
	hasOutput := output != nil && output.Status != "pending"
	return []CompletenessItem{
		checkUserEngagement(userMsgs, 1, "建议至少与AI交流1轮"),
		{Label: "阶段产出", Passed: hasOutput, Detail: boolHint(hasOutput, "已有阶段产出", "尚未开始此阶段")},
	}
}

// ==================== 统计当前阶段用户消息数 ====================

// countUserMessagesInStage 统计当前阶段中用户发送的消息数
func countUserMessagesInStage(ctx context.Context, lessonPlanID string, stageCode string, lp *models.LessonPlan) int {
	messages, err := repository.GetConversationLog(ctx, lessonPlanID)
	if err != nil || len(messages) == 0 {
		return 0
	}

	snapshots := parseStageSnapshots(lp)
	startIdx, endIdx := findStageMessageRange(messages, snapshots, stageCode)

	if startIdx < 0 {
		return 0
	}

	return countNonAutoUserMessages(messages, startIdx, endIdx)
}

// parseStageSnapshots 从教案的stage_config解析阶段快照列表
func parseStageSnapshots(lp *models.LessonPlan) []models.StageConfigSnapshot {
	var snapshots []models.StageConfigSnapshot
	if lp.StageConfig != "" && lp.StageConfig != "[]" {
		_ = json.Unmarshal([]byte(lp.StageConfig), &snapshots)
	}
	return snapshots
}

// findStageMessageRange 在消息列表中定位指定阶段的起止索引
// 返回 (startIdx, endIdx)，startIdx<0 表示未找到
func findStageMessageRange(messages []*models.ConversationMessage, snapshots []models.StageConfigSnapshot, stageCode string) (int, int) {
	// 构建阶段名→阶段代码映射
	stageNameToCode := make(map[string]string)
	for _, snap := range snapshots {
		stageNameToCode[snap.StageName] = snap.StageCode
	}

	firstStageCode := ""
	if len(snapshots) > 0 {
		firstStageCode = snapshots[0].StageCode
	}

	startIdx := -1
	endIdx := len(messages)

	for i, msg := range messages {
		if string(msg.Role) != "system" || !strings.HasPrefix(msg.Content, "__STAGE_SEP__") {
			continue
		}
		sepStageCode := parseStageSepCode(msg.Content, stageNameToCode)
		if sepStageCode == stageCode {
			startIdx = i
		} else if startIdx >= 0 && endIdx == len(messages) {
			endIdx = i
		}
	}

	// 如果是第一个阶段且没有找到分隔符，从头开始
	if startIdx < 0 && stageCode == firstStageCode {
		startIdx = 0
		for i, msg := range messages {
			if string(msg.Role) == "system" && strings.HasPrefix(msg.Content, "__STAGE_SEP__") {
				endIdx = i
				break
			}
		}
	}

	return startIdx, endIdx
}

// parseStageSepCode 从阶段分隔符消息中解析阶段代码
// 分隔符格式：__STAGE_SEP__阶段名__AI角色
func parseStageSepCode(content string, stageNameToCode map[string]string) string {
	rest := content[len("__STAGE_SEP__"):]
	parts := strings.SplitN(rest, "__", 2)
	if len(parts) == 0 {
		return ""
	}
	if code, ok := stageNameToCode[parts[0]]; ok {
		return code
	}
	return ""
}

// countNonAutoUserMessages 在指定消息范围内统计非自动触发的用户消息数
func countNonAutoUserMessages(messages []*models.ConversationMessage, startIdx, endIdx int) int {
	autoTriggerPrefixes := []string{
		"我们进入",
		"请对上一阶段完成的教案进行全面专业评审",
	}

	count := 0
	for i := startIdx; i < endIdx && i < len(messages); i++ {
		if string(messages[i].Role) != "user" {
			continue
		}
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

func getOutputNarrative(output *models.WorkshopStageOutput) string {
	if output == nil {
		return ""
	}
	return output.NarrativeOutput
}

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

func checkUserEngagement(userMsgs int, minRounds int, hint string) CompletenessItem {
	if userMsgs >= minRounds {
		return CompletenessItem{Label: "对话参与", Passed: true, Detail: "已与AI充分交流"}
	}
	return CompletenessItem{Label: "对话参与", Passed: false, Detail: hint}
}

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

func checkMinLength(label string, text string, minLen int, hint string) CompletenessItem {
	textLen := len([]rune(strings.TrimSpace(text)))
	if textLen >= minLen {
		return CompletenessItem{Label: label, Passed: true, Detail: label + "已达标"}
	}
	return CompletenessItem{Label: label, Passed: false, Detail: hint}
}

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
