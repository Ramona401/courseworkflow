package services

// workshop_stage_prompts_test.go — 阶段提示词构建纯函数单元测试
//
// 测试范围（全部为纯函数，无数据库依赖）：
//   - stageCodeToName：阶段代码→中文名转换
//   - safeUTF8Truncate：安全UTF-8截断
//   - selectPromptVariant：变体段选择
//   - buildDialogueGuidelines：对话规范指引
//   - BuildStageGlobalContext：配方全局上下文
//   - BuildPriorOutputsContext：前序阶段产出物上下文
//   - BuildLessonStructurePrompt：教案结构注入
//   - BuildStageOpeningPrompt：阶段开场白提示词
//   - BuildStageChatPromptV2：分层记忆对话提示词
//   - buildDegradationHint：降级提示词
//   - stageTimingMap：阶段时机映射表

import (
	"strings"
	"testing"

	"tedna/internal/models"
)

// ==================== stageCodeToName 测试 ====================

func TestStageCodeToName_KnownCodes(t *testing.T) {
	tests := map[string]string{
		"analyze": "教学分析",
		"design":  "教学设计",
		"write":   "教案撰写",
		"review":  "AI评审",
		"revise":  "修订定稿",
	}
	for code, expected := range tests {
		result := stageCodeToName(code)
		if result != expected {
			t.Errorf("stageCodeToName(%q)=%q, 期望%q", code, result, expected)
		}
	}
}

func TestStageCodeToName_UnknownCode(t *testing.T) {
	result := stageCodeToName("custom_stage")
	if result != "custom_stage" {
		t.Errorf("未知代码应原样返回，实际%q", result)
	}
}

// ==================== safeUTF8Truncate 测试 ====================

func TestSafeUTF8Truncate_Short(t *testing.T) {
	result := safeUTF8Truncate("你好世界", 10)
	if result != "你好世界" {
		t.Errorf("短文本不应截断，实际%q", result)
	}
}

func TestSafeUTF8Truncate_ExactLength(t *testing.T) {
	result := safeUTF8Truncate("你好世界", 4)
	if result != "你好世界" {
		t.Errorf("恰好等于maxChars不应截断，实际%q", result)
	}
}

func TestSafeUTF8Truncate_Truncated(t *testing.T) {
	result := safeUTF8Truncate("你好世界欢迎你", 4)
	if !strings.HasSuffix(result, "...") {
		t.Errorf("截断后应以...结尾，实际%q", result)
	}
	// 应保留前4个字符
	if !strings.HasPrefix(result, "你好世界") {
		t.Errorf("应保留前4个字符，实际%q", result)
	}
}

func TestSafeUTF8Truncate_Empty(t *testing.T) {
	result := safeUTF8Truncate("", 10)
	if result != "" {
		t.Errorf("空字符串应返回空，实际%q", result)
	}
}

// ==================== selectPromptVariant 测试 ====================

func TestSelectPromptVariant_GuidedMode(t *testing.T) {
	variants := `{"guided":"引导版提示词","efficient":"高效版提示词"}`
	result := selectPromptVariant(variants, "guided")
	if result != "引导版提示词" {
		t.Errorf("guided模式应返回引导版，实际%q", result)
	}
}

func TestSelectPromptVariant_EfficientMode(t *testing.T) {
	variants := `{"guided":"引导版提示词","efficient":"高效版提示词"}`
	result := selectPromptVariant(variants, "efficient")
	if result != "高效版提示词" {
		t.Errorf("efficient模式应返回高效版，实际%q", result)
	}
}

func TestSelectPromptVariant_PerStageDefault(t *testing.T) {
	variants := `{"guided":"引导版","efficient":"高效版"}`
	// per_stage模式默认回退到guided
	result := selectPromptVariant(variants, "per_stage")
	if result != "引导版" {
		t.Errorf("per_stage应回退到guided，实际%q", result)
	}
}

func TestSelectPromptVariant_EmptyJSON(t *testing.T) {
	result := selectPromptVariant("{}", "guided")
	if result != "" {
		t.Errorf("空JSON应返回空，实际%q", result)
	}
}

func TestSelectPromptVariant_EmptyString(t *testing.T) {
	result := selectPromptVariant("", "guided")
	if result != "" {
		t.Errorf("空字符串应返回空，实际%q", result)
	}
}

func TestSelectPromptVariant_InvalidJSON(t *testing.T) {
	result := selectPromptVariant("not json", "guided")
	if result != "" {
		t.Errorf("无效JSON应返回空，实际%q", result)
	}
}

func TestSelectPromptVariant_EmptyMode(t *testing.T) {
	variants := `{"guided":"引导版"}`
	result := selectPromptVariant(variants, "")
	if result != "引导版" {
		t.Errorf("空mode应回退到guided，实际%q", result)
	}
}

// ==================== buildDialogueGuidelines 测试 ====================

func TestBuildDialogueGuidelines_AllStages(t *testing.T) {
	stages := []string{"analyze", "design", "write", "review", "revise", "custom_stage"}
	for _, code := range stages {
		result := buildDialogueGuidelines(code)
		if result == "" {
			t.Errorf("阶段%q应返回非空对话规范", code)
		}
		if !strings.Contains(result, "对话规范") {
			t.Errorf("阶段%q应包含'对话规范'", code)
		}
		if !strings.Contains(result, "不要使用任何XML标签") {
			t.Errorf("阶段%q应包含XML标签禁用提醒", code)
		}
	}
}

func TestBuildDialogueGuidelines_WriteSpecial(t *testing.T) {
	result := buildDialogueGuidelines("write")
	if !strings.Contains(result, "分段确认") {
		t.Error("write阶段应包含分段确认机制")
	}
}

func TestBuildDialogueGuidelines_ReviewSpecial(t *testing.T) {
	result := buildDialogueGuidelines("review")
	if !strings.Contains(result, "评审") {
		t.Error("review阶段应包含评审相关规范")
	}
}

// ==================== BuildStageGlobalContext 测试 ====================

func TestBuildStageGlobalContext_FullRecipe(t *testing.T) {
	recipe := &models.TeachingRecipe{
		StudentProfile:     "学生基础较好",
		TeachingStyle:      "互动式教学",
		SchoolRequirements: "每周一测",
		CustomNotes:        "注意分层教学",
		CustomPrompt:       "重点关注后进生",
	}
	result := BuildStageGlobalContext(recipe)
	if !strings.Contains(result, "学情档案") {
		t.Error("应包含学情档案段")
	}
	if !strings.Contains(result, "学生基础较好") {
		t.Error("应包含学情内容")
	}
	if !strings.Contains(result, "教学风格偏好") {
		t.Error("应包含教学风格段")
	}
	if !strings.Contains(result, "学校要求") {
		t.Error("应包含学校要求段")
	}
	if !strings.Contains(result, "备课心得") {
		t.Error("应包含备课心得段")
	}
	if !strings.Contains(result, "自定义指令") {
		t.Error("应包含自定义指令段")
	}
}

func TestBuildStageGlobalContext_EmptyRecipe(t *testing.T) {
	recipe := &models.TeachingRecipe{}
	result := BuildStageGlobalContext(recipe)
	if result != "" {
		t.Errorf("所有字段为空时应返回空，实际%q", result)
	}
}

func TestBuildStageGlobalContext_PartialRecipe(t *testing.T) {
	recipe := &models.TeachingRecipe{StudentProfile: "学情信息"}
	result := BuildStageGlobalContext(recipe)
	if !strings.Contains(result, "学情档案") {
		t.Error("应包含学情档案段")
	}
	if strings.Contains(result, "教学风格偏好") {
		t.Error("空字段不应生成段")
	}
}

// ==================== BuildPriorOutputsContext 测试 ====================

func TestBuildPriorOutputsContext_Empty(t *testing.T) {
	result := BuildPriorOutputsContext(nil)
	if result != "" {
		t.Error("空列表应返回空")
	}
}

func TestBuildPriorOutputsContext_CompletedStages(t *testing.T) {
	outputs := []*models.WorkshopStageOutput{
		{StageCode: "analyze", StageOrder: 1, Status: "completed", StructuredOutput: `{"key":"value"}`, NarrativeOutput: "分析总结"},
		{StageCode: "design", StageOrder: 2, Status: "completed", StructuredOutput: "{}", NarrativeOutput: "设计总结"},
	}
	result := BuildPriorOutputsContext(outputs)
	if !strings.Contains(result, "前序阶段产出") {
		t.Error("应包含前序阶段产出标题")
	}
	if !strings.Contains(result, "教学分析") {
		t.Error("应包含analyze的中文名")
	}
	if !strings.Contains(result, "分析总结") {
		t.Error("应包含narrative内容")
	}
}

func TestBuildPriorOutputsContext_SkippedStage(t *testing.T) {
	outputs := []*models.WorkshopStageOutput{
		{StageCode: "analyze", StageOrder: 1, Status: "skipped"},
	}
	result := BuildPriorOutputsContext(outputs)
	if !strings.Contains(result, "已跳过") {
		t.Error("跳过的阶段应标记'已跳过'")
	}
}

func TestBuildPriorOutputsContext_InProgressIgnored(t *testing.T) {
	outputs := []*models.WorkshopStageOutput{
		{StageCode: "analyze", StageOrder: 1, Status: "in_progress", NarrativeOutput: "进行中内容"},
	}
	result := BuildPriorOutputsContext(outputs)
	if strings.Contains(result, "进行中内容") {
		t.Error("进行中的阶段不应包含其内容")
	}
}

// ==================== BuildLessonStructurePrompt 测试 ====================

func TestBuildLessonStructurePrompt_AnalyzeStage(t *testing.T) {
	result := BuildLessonStructurePrompt("analyze", `[{"name":"教学目标","required":true}]`)
	if result != "" {
		t.Error("analyze阶段不应注入结构")
	}
}

func TestBuildLessonStructurePrompt_DesignStage(t *testing.T) {
	structure := `[{"name":"教学目标","required":true},{"name":"教学过程","required":false}]`
	result := BuildLessonStructurePrompt("design", structure)
	if !strings.Contains(result, "概要") {
		t.Error("design阶段应使用概要模式")
	}
	if !strings.Contains(result, "教学目标") {
		t.Error("应包含板块名")
	}
	if !strings.Contains(result, "必含") {
		t.Error("必填板块应标记'必含'")
	}
}

func TestBuildLessonStructurePrompt_WriteStage(t *testing.T) {
	structure := `[{"name":"教学目标","required":true,"requirement":"三维目标"},{"name":"教学过程","required":true,"sub_sections":[{"name":"导入","duration":5,"goal":"激发兴趣"}]}]`
	result := BuildLessonStructurePrompt("write", structure)
	if !strings.Contains(result, "严格按照") {
		t.Error("write阶段应使用严格模式")
	}
	if !strings.Contains(result, "三维目标") {
		t.Error("应包含具体要求")
	}
	if !strings.Contains(result, "导入") {
		t.Error("应包含子环节")
	}
	if !strings.Contains(result, "5分钟") {
		t.Error("应包含时长")
	}
}

func TestBuildLessonStructurePrompt_EmptyJSON(t *testing.T) {
	result := BuildLessonStructurePrompt("write", "[]")
	if result != "" {
		t.Error("空数组应返回空")
	}
}

func TestBuildLessonStructurePrompt_InvalidJSON(t *testing.T) {
	result := BuildLessonStructurePrompt("write", "not json")
	if result != "" {
		t.Error("无效JSON应返回空")
	}
}

// ==================== BuildStageOpeningPrompt 测试 ====================

func TestBuildStageOpeningPrompt(t *testing.T) {
	lp := &models.LessonPlan{Subject: "数学", Grade: "七年级", Topic: "一元二次方程", DurationMinutes: 45}
	stage := &models.WorkshopStage{StageCode: "analyze", StageName: "教学分析", AIRole: "教学分析师"}
	result := BuildStageOpeningPrompt(lp, stage, 1, 5)
	if !strings.Contains(result, "数学") {
		t.Error("应包含学科")
	}
	if !strings.Contains(result, "七年级") {
		t.Error("应包含年级")
	}
	if !strings.Contains(result, "1/5") {
		t.Error("应包含阶段序号")
	}
	if !strings.Contains(result, "教学分析") {
		t.Error("应包含阶段名")
	}
	if !strings.Contains(result, "教学分析师") {
		t.Error("应包含AI角色")
	}
}

// ==================== BuildStageChatPromptV2 测试 ====================

func TestBuildStageChatPromptV2_BasicInfo(t *testing.T) {
	lp := &models.LessonPlan{Subject: "英语", Grade: "八年级", Topic: "过去式", DurationMinutes: 40}
	userMsg := &models.ConversationMessage{Content: "学生基础怎么样？"}
	result := BuildStageChatPromptV2(lp, nil, "", userMsg)
	if !strings.Contains(result, "英语") {
		t.Error("应包含学科")
	}
	if !strings.Contains(result, "八年级") {
		t.Error("应包含年级")
	}
	if !strings.Contains(result, "学生基础怎么样") {
		t.Error("应包含用户消息")
	}
}

func TestBuildStageChatPromptV2_WithEpisodicMemory(t *testing.T) {
	lp := &models.LessonPlan{Subject: "数学", Grade: "三年级", Topic: "分数", DurationMinutes: 45}
	userMsg := &models.ConversationMessage{Content: "继续"}
	episodic := "=== 历史摘要 ===\n分析阶段：已确定教学目标"
	result := BuildStageChatPromptV2(lp, nil, episodic, userMsg)
	if !strings.Contains(result, "历史摘要") {
		t.Error("应包含Episodic Memory")
	}
}

func TestBuildStageChatPromptV2_WithLongContent(t *testing.T) {
	longContent := strings.Repeat("教案内容", 1000)
	lp := &models.LessonPlan{Subject: "语文", Grade: "一年级", Topic: "识字", DurationMinutes: 40, ContentMarkdown: longContent}
	userMsg := &models.ConversationMessage{Content: "请优化"}
	result := BuildStageChatPromptV2(lp, nil, "", userMsg)
	if !strings.Contains(result, "已截断") {
		t.Error("超长内容应被截断并标记")
	}
}

func TestBuildStageChatPromptV2_HistoryTruncation(t *testing.T) {
	lp := &models.LessonPlan{Subject: "数学", Grade: "三年级", Topic: "加法", DurationMinutes: 45}
	// 生成25条消息，应只保留最后20条
	var history []*models.ConversationMessage
	for i := 0; i < 25; i++ {
		history = append(history, &models.ConversationMessage{
			Role: models.ConvRoleUser, Content: "消息内容",
		})
	}
	userMsg := &models.ConversationMessage{Content: "新消息"}
	result := BuildStageChatPromptV2(lp, history, "", userMsg)
	// 简单验证包含对话记录
	if !strings.Contains(result, "本阶段对话记录") {
		t.Error("应包含对话记录段")
	}
}

// ==================== buildDegradationHint 测试 ====================

func TestBuildDegradationHint_NoSkipped(t *testing.T) {
	outputs := []*models.WorkshopStageOutput{
		{StageCode: "analyze", Status: "completed"},
	}
	result := buildDegradationHint(outputs, nil)
	if result != "" {
		t.Error("无跳过阶段应返回空")
	}
}

func TestBuildDegradationHint_WithSkipped(t *testing.T) {
	outputs := []*models.WorkshopStageOutput{
		{StageCode: "analyze", Status: "skipped"},
	}
	result := buildDegradationHint(outputs, nil)
	if !strings.Contains(result, "降级补偿") {
		t.Error("有跳过阶段应包含降级提示")
	}
	if !strings.Contains(result, "教学分析") {
		t.Error("应包含跳过阶段的中文名")
	}
}

func TestBuildDegradationHint_WithRecipe(t *testing.T) {
	outputs := []*models.WorkshopStageOutput{
		{StageCode: "analyze", Status: "skipped"},
	}
	recipe := &models.TeachingRecipe{StudentProfile: "学情信息", TeachingStyle: "互动式"}
	result := buildDegradationHint(outputs, recipe)
	if !strings.Contains(result, "学情信息已在配方中提供") {
		t.Error("有配方学情时应提示参考")
	}
}

// ==================== stageTimingMap 测试 ====================

func TestStageTimingMap_Defined(t *testing.T) {
	expectedStages := []string{"analyze", "design", "write", "review"}
	for _, stage := range expectedStages {
		timings, ok := stageTimingMap[stage]
		if !ok {
			t.Errorf("stageTimingMap应包含%q", stage)
			continue
		}
		if len(timings) == 0 {
			t.Errorf("阶段%q的timings不应为空", stage)
		}
	}
}

func TestStageTimingMap_AnalyzeValues(t *testing.T) {
	timings := stageTimingMap["analyze"]
	// analyze应包含1(开场)和4(贯穿)
	has1, has4 := false, false
	for _, v := range timings {
		if v == 1 { has1 = true }
		if v == 4 { has4 = true }
	}
	if !has1 || !has4 {
		t.Errorf("analyze应包含[1,4]，实际%v", timings)
	}
}
