package services

// assessment_service.go — 教学风格前测服务
//
// 迭代3新增：
//   - StartAssessment：初始化前测对话，返回AI开场白
//   - ChatAssessment：前测对话轮次处理（6个问题渐进采集）
//   - SubmitAssessment：提交AI判定结果+规则兜底校验+写入teaching_profile+自动生成配方
//   - SkipAssessment：跳过前测，写入默认画像
//   - GetAssessmentResult：获取当前用户的前测结果
//   - AutoGenerateRecipeFromProfile：根据teaching_profile自动生成个性化配方
//
// AI调用模式：
//   import aiClient "tedna/internal/ai"
//   aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), "", "", "", "")
//   result, err := aiClient.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
//   result.Content — AI输出文本
//
// v89-2变更：callAI方法增加userID参数，传入真实TraceContext替代nil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aiClient "tedna/internal/ai"
	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 错误常量 ====================

var (
	ErrAssessmentAlreadyDone = errors.New("您已完成风格前测，如需重新测评请使用重测入口")
	ErrAssessmentNotStarted  = errors.New("前测会话未开始")
	ErrAssessmentInvalidStep = errors.New("前测步骤异常")
)

// AssessmentService 前测服务
type AssessmentService struct {
	recipeService *RecipeService
	cfg           interface{ GetAESKey() string } // 用于获取AES Key调用AI
}

var assessLog = logger.WithModule("assessment")

// NewAssessmentService 创建前测服务实例
// cfg 需实现 GetAESKey() string 接口（与 LessonPlanGenService 一致）
func NewAssessmentService(recipeSvc *RecipeService, cfg interface{ GetAESKey() string }) *AssessmentService {
	return &AssessmentService{
		recipeService: recipeSvc,
		cfg:           cfg,
	}
}

// ==================== 前测系统提示词 ====================

const assessmentSystemPrompt = `你是一位经验丰富的教研专家，正在和一位老师进行轻松的对话，了解他/她的教学风格和AI协作偏好。

## 你的任务
通过6个问题的自然对话，了解这位老师的教学经验、备课习惯和AI使用偏好。注意：这不是问卷调查，是一次友好的专业对话。

## 对话流程（6个问题，逐步推进）

**Q1 教龄和学科年级**（直接问）
- 问老师教了几年书、教什么学科和年级
- 这是基础信息，直接友好地问即可

**Q2 备课起点习惯**（给选项引导）
- "您平时备课一般从哪里开始？"
- 参考选项：A.先看教材和课标 B.先想学生情况 C.先找好的活动设计 D.凭经验直接写
- 不必限定选项，老师怎么说都行

**Q3 AI协作偏好**（3选1引导）
- "如果AI帮您备课，您更希望哪种方式？"
- A.「你帮我做完，我来改」— 工具型
- B.「我们一起想」— 协作型
- C.「你带着我一步步走」— 引导型

**Q4 教学设计思路**（给具体课题看第一反应）
- 给老师一个简单的课题（根据Q1的学科），问"如果让您教这个内容，您第一个想到的是什么？"
- 观察老师是先想目标、先想活动、还是先想学生反应

**Q5 教案质量关注点**（可多选）
- "您觉得一份好教案最重要的是什么？（可以选多个）"
- 活动设计细节 / 学生预期反应 / 时间分配合理性 / 知识点准确性 / 分层教学设计 / 评估方式设计 / 资源与目标对齐 / 教学创新性

**Q6 习惯的教案结构**（自由描述）
- "您平时写教案一般包含哪些部分？大概什么结构？"
- 让老师自由描述，不限定格式

## 对话规则
1. 每次只问一个问题，等老师回答后再问下一个
2. 语气温暖专业，像同事间的交流
3. 对老师的回答给予简短的认可和回应，然后自然过渡到下一个问题
4. 如果老师的回答已经包含了后续问题的信息，可以跳过或简化
5. 全程用中文交流

## 完成判定
当6个问题都聊完后（或老师提供了足够信息），输出判定结果。格式如下：

<assessment_result>
{
  "experience_years": 5,
  "subject_primary": "信息技术",
  "grade_primary": "七年级",
  "teaching_style": "growing",
  "ai_collaboration": "collaborative",
  "priorities": ["activity_detail", "student_response"],
  "lesson_structure_desc": "教学目标、重难点、课前准备、教学过程（导入+新授+练习+小结）、作业、板书"
}
</assessment_result>

其中：
- teaching_style: mature(8年以上且目标明确) / growing(3-8年) / beginner(3年以下)
- ai_collaboration: tool / collaborative / guided
- priorities: 从以下选取：activity_detail/student_response/time_allocation/knowledge_accuracy/differentiation/assessment_design/resource_alignment/innovation

注意：输出 <assessment_result> 标签后，再用一两句话做个友好的总结，告诉老师"我已经了解您的风格了，接下来会为您准备一套个性化的备课配方"。`

// ==================== 内部：获取AI配置并调用 ====================

// callAI 封装AI调用：获取配置→调用→返回文本
// v89-2变更：增加userID参数，构建真实TraceContext
// 前测对话属于lesson_plan场景范畴（但无plan_id，只有user_id）
func (s *AssessmentService) callAI(systemPrompt string, userPrompt string, userID string) (string, error) {
	aiCfg, err := aiClient.GetEffectiveConfig(s.cfg.GetAESKey(), "", "", "", "")
	if err != nil {
		return "", fmt.Errorf("获取AI配置失败: %w", err)
	}

	// v89-2：构建真实TraceContext，记录场景和用户ID
	var traceCtx *aiClient.TraceContext
	if userID != "" {
		uid := userID
		traceCtx = &aiClient.TraceContext{
			SceneCode: "lesson_plan", // 前测对话归类为教案场景
			UserID:    &uid,
		}
	}

	result, err := aiClient.CallAI(aiCfg, systemPrompt, userPrompt, traceCtx)
	if err != nil {
		return "", fmt.Errorf("AI调用失败: %w", err)
	}
	return result.Content, nil
}

// ==================== 开始前测 ====================

// StartAssessment 初始化前测对话
func (s *AssessmentService) StartAssessment(ctx context.Context, userID string) (*models.AssessmentStartResponse, error) {
	// 检查是否已完成前测（允许重测，不阻断，只记日志）
	existing, _ := repository.GetTeachingProfile(ctx, userID)
	if existing != nil {
		assessLog.Info("用户重新进行前测", "user_id", userID, "previous_version", existing.AssessmentVersion)
	}

	// 调用AI获取开场白（v89-2：传入userID用于追踪）
	openingPrompt := "请开始和这位老师进行风格前测对话。先用一句温暖的开场白打招呼，然后问第一个问题（Q1：教龄和学科年级）。"
	aiContent, err := s.callAI(assessmentSystemPrompt, openingPrompt, userID)
	if err != nil {
		assessLog.Error("前测AI调用失败，使用降级开场白", "user_id", userID, "error", err)
		// 降级：使用固定开场白
		aiContent = "您好！我是您的教研助手 😊 很高兴认识您！\n\n在开始备课之前，我想先了解一下您的教学经验和习惯，这样我就能更好地配合您的节奏。\n\n请问您教书几年了？主要教什么学科和年级呢？"
	}

	now := time.Now()
	opening := models.AssessmentMessage{
		Role:      "assistant",
		Content:   aiContent,
		Timestamp: now.Format(time.RFC3339),
		StepCode:  "q1",
	}

	return &models.AssessmentStartResponse{
		SessionID:      fmt.Sprintf("assess_%s_%d", userID[:8], now.Unix()),
		OpeningMessage: opening,
		TotalSteps:     6,
		CurrentStep:    "q1",
	}, nil
}

// ==================== 前测对话 ====================

// ChatAssessment 前测对话轮次处理
func (s *AssessmentService) ChatAssessment(ctx context.Context, userID string, userMessage string, conversationHistory []models.AssessmentMessage) (*models.AssessmentChatResponse, error) {
	// 构建多轮对话prompt（将历史对话拼入user消息）
	fullPrompt := buildMultiTurnPrompt(conversationHistory, userMessage)

	// v89-2：传入userID用于追踪
	aiContent, err := s.callAI(assessmentSystemPrompt, fullPrompt, userID)
	if err != nil {
		assessLog.Error("前测对话AI调用失败", "user_id", userID, "error", err)
		return nil, fmt.Errorf("AI服务暂时不可用，请稍后重试")
	}

	// 检测是否包含 <assessment_result> 标签（对话结束判定）
	isComplete := strings.Contains(aiContent, "<assessment_result>")

	// 估算当前步骤（根据对话轮次）
	userMsgCount := 0
	for _, msg := range conversationHistory {
		if msg.Role == "user" {
			userMsgCount++
		}
	}
	userMsgCount++ // 加上当前这条
	currentStep := estimateStep(userMsgCount)
	progress := userMsgCount * 100 / 6
	if progress > 100 {
		progress = 100
	}

	now := time.Now()
	return &models.AssessmentChatResponse{
		AIMessage: models.AssessmentMessage{
			Role:      "assistant",
			Content:   aiContent,
			Timestamp: now.Format(time.RFC3339),
			StepCode:  currentStep,
		},
		CurrentStep: currentStep,
		IsComplete:  isComplete,
		Progress:    progress,
	}, nil
}

// ==================== 提交前测结果 ====================

// SubmitAssessment 提交前测判定结果
// 1. 接收AI判定的原始结果
// 2. 规则兜底校验（新手不能被推荐efficient）
// 3. 计算推荐模式和阶段
// 4. 写入 users.teaching_profile
// 5. 自动生成个性化配方
func (s *AssessmentService) SubmitAssessment(ctx context.Context, userID string, req *models.AssessmentSubmitRequest, conversationLog []models.AssessmentMessage) (*models.AssessmentSubmitResponse, error) {
	now := time.Now()

	// === 规则兜底校验 ===

	// 校验教学风格有效性
	style := req.TeachingStyle
	if style != models.StyleMature && style != models.StyleGrowing && style != models.StyleBeginner {
		style = models.StyleGrowing // 无效值默认为成长型
	}

	// 校验AI协作偏好有效性
	collab := req.AICollaboration
	if collab != models.CollabTool && collab != models.CollabCollaborative && collab != models.CollabGuided {
		collab = models.CollabCollaborative // 无效值默认为协作型
	}

	// 规则兜底：新手(beginner) 不能被推荐为 efficient 模式
	recommendedMode := models.StyleModeMap[style]
	if override, ok := models.CollabModeOverride[collab]; ok {
		recommendedMode = override
	}
	if style == models.StyleBeginner && recommendedMode == "efficient" {
		recommendedMode = "guided"
		assessLog.Info("规则兜底：新手强制使用guided模式", "user_id", userID, "original_collab", collab)
	}

	// 推荐阶段
	recommendedStages := models.StyleStagesMap[style]
	if recommendedStages == nil {
		recommendedStages = []string{"analyze", "design", "write", "review", "revise"}
	}

	// 校验priorities
	validPriorities := filterValidPriorities(req.Priorities)

	// 校验教龄合理性
	experienceYears := req.ExperienceYears
	if experienceYears < 0 {
		experienceYears = 0
	}
	if experienceYears > 50 {
		experienceYears = 50
	}
	// 教龄与风格交叉校验
	if experienceYears < 3 && style == models.StyleMature {
		style = models.StyleGrowing
		assessLog.Info("规则兜底：教龄<3年不能为mature", "user_id", userID, "years", experienceYears)
	}

	// === 构建 TeachingProfile ===
	profile := &models.TeachingProfile{
		AssessmentVersion:   1,
		AssessedAt:          &now,
		ExperienceYears:     experienceYears,
		SubjectPrimary:      strings.TrimSpace(req.SubjectPrimary),
		GradePrimary:        strings.TrimSpace(req.GradePrimary),
		TeachingStyle:       style,
		AICollaboration:     collab,
		Priorities:          validPriorities,
		RecommendedMode:     recommendedMode,
		RecommendedStages:   recommendedStages,
		LessonStructureDesc: strings.TrimSpace(req.LessonStructureDesc),
		ConversationLog:     conversationLog,
	}

	// === 写入数据库 ===
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		assessLog.Error("序列化前测结果失败", "user_id", userID, "error", err)
		return nil, err
	}

	if err := repository.UpdateTeachingProfile(ctx, userID, string(profileJSON)); err != nil {
		assessLog.Error("保存前测结果失败", "user_id", userID, "error", err)
		return nil, err
	}
	assessLog.Info("前测结果保存成功", "user_id", userID, "style", style, "collab", collab, "mode", recommendedMode)

	// === 自动生成配方 ===
	recipeID := ""
	recipe, err := s.autoGenerateRecipe(ctx, userID, profile)
	if err != nil {
		assessLog.Error("自动生成配方失败（不影响前测结果）", "user_id", userID, "error", err)
	} else if recipe != nil {
		recipeID = recipe.ID
		assessLog.Info("自动生成配方成功", "user_id", userID, "recipe_id", recipeID)
	}

	return &models.AssessmentSubmitResponse{
		Profile:  profile,
		RecipeID: recipeID,
	}, nil
}

// ==================== 跳过前测 ====================

// SkipAssessment 跳过前测，写入默认画像（协作型+5步引导版）
func (s *AssessmentService) SkipAssessment(ctx context.Context, userID string) (*models.AssessmentSubmitResponse, error) {
	now := time.Now()

	profile := &models.TeachingProfile{
		AssessmentVersion: 1,
		AssessedAt:        &now,
		TeachingStyle:     models.StyleGrowing,
		AICollaboration:   models.CollabCollaborative,
		RecommendedMode:   "guided",
		RecommendedStages: []string{"analyze", "design", "write", "review", "revise"},
		Priorities:        []string{},
		ConversationLog:   []models.AssessmentMessage{},
	}

	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}

	if err := repository.UpdateTeachingProfile(ctx, userID, string(profileJSON)); err != nil {
		assessLog.Error("保存跳过前测默认画像失败", "user_id", userID, "error", err)
		return nil, err
	}
	assessLog.Info("用户跳过前测，使用默认画像", "user_id", userID)

	// 自动生成默认配方
	recipeID := ""
	recipe, err := s.autoGenerateRecipe(ctx, userID, profile)
	if err != nil {
		assessLog.Error("跳过前测后自动生成配方失败", "user_id", userID, "error", err)
	} else if recipe != nil {
		recipeID = recipe.ID
	}

	return &models.AssessmentSubmitResponse{
		Profile:  profile,
		RecipeID: recipeID,
	}, nil
}

// ==================== 获取前测结果 ====================

// GetAssessmentResult 获取当前用户的前测结果
func (s *AssessmentService) GetAssessmentResult(ctx context.Context, userID string) (*models.AssessmentResultResponse, error) {
	profile, err := repository.GetTeachingProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &models.AssessmentResultResponse{
		HasProfile: profile != nil,
		Profile:    profile,
	}, nil
}

// ==================== 自动生成配方 ====================

// AutoGenerateRecipeFromProfile 外部调用入口
func (s *AssessmentService) AutoGenerateRecipeFromProfile(ctx context.Context, userID string) (*models.AutoRecipeResponse, error) {
	profile, err := repository.GetTeachingProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, errors.New("请先完成风格前测")
	}

	recipe, err := s.autoGenerateRecipe(ctx, userID, profile)
	if err != nil {
		return nil, err
	}

	return &models.AutoRecipeResponse{
		RecipeID:   recipe.ID,
		RecipeName: recipe.Name,
	}, nil
}

// autoGenerateRecipe 内部方法：根据画像自动生成配方
func (s *AssessmentService) autoGenerateRecipe(ctx context.Context, userID string, profile *models.TeachingProfile) (*models.TeachingRecipe, error) {
	// 配方名称
	subject := profile.SubjectPrimary
	if subject == "" {
		subject = "通用"
	}
	grade := profile.GradePrimary
	if grade == "" {
		grade = "通用"
	}
	recipeName := fmt.Sprintf("%s·%s·我的默认配方", subject, grade)

	// 构建 stages_config（StageFlowItem格式）
	stagesConfig := buildStagesConfigFromProfile(profile)
	stagesJSON, _ := json.Marshal(stagesConfig)

	// 构建 lesson_structure（如果用户描述了偏好）
	lessonStructure := "[]"
	if profile.LessonStructureDesc != "" {
		blocks := parseLessonStructureFromDesc(profile.LessonStructureDesc)
		if len(blocks) > 0 {
			lsJSON, _ := json.Marshal(blocks)
			lessonStructure = string(lsJSON)
		}
	}

	// 教学风格描述
	styleDesc := ""
	switch profile.TeachingStyle {
	case models.StyleMature:
		styleDesc = "经验丰富，目标明确，偏好高效直接的备课方式"
	case models.StyleGrowing:
		styleDesc = "有一定教学经验，希望在AI协助下不断提升教学设计水平"
	case models.StyleBeginner:
		styleDesc = "教学新手，希望AI提供更多引导和建议"
	}

	// 协作偏好描述
	collabDesc := ""
	switch profile.AICollaboration {
	case models.CollabTool:
		collabDesc = "偏好工具型协作：AI完成初稿，教师审核修改"
	case models.CollabCollaborative:
		collabDesc = "偏好协作型：教师和AI共同思考、共同设计"
	case models.CollabGuided:
		collabDesc = "偏好引导型：AI逐步引导，教师跟随节奏完成"
	}

	teachingStyle := styleDesc
	if collabDesc != "" {
		teachingStyle += "；" + collabDesc
	}

	// 创建配方
	req := &models.CreateRecipeRequest{
		Name:            recipeName,
		Description:     fmt.Sprintf("系统根据教学风格前测自动生成的个性化配方（%s，%s模式）", profile.TeachingStyle, profile.RecommendedMode),
		Subject:         subject,
		GradeRange:      grade,
		TeachingStyle:   teachingStyle,
		PromptMode:      profile.RecommendedMode,
		StagesConfig:    string(stagesJSON),
		LessonStructure: lessonStructure,
	}

	recipe, err := s.recipeService.CreateRecipe(ctx, req, userID)
	if err != nil {
		return nil, fmt.Errorf("创建配方失败: %w", err)
	}

	return recipe, nil
}

// ==================== 内部工具函数 ====================

// buildMultiTurnPrompt 将多轮对话消息+当前用户消息构建为单个prompt文本
// 因为CallAI目前只支持单轮(system+user)，历史对话拼入user消息
func buildMultiTurnPrompt(history []models.AssessmentMessage, currentUserMsg string) string {
	var sb strings.Builder
	sb.WriteString("以下是之前的对话记录：\n\n")
	for _, msg := range history {
		if msg.Role == "assistant" {
			sb.WriteString(fmt.Sprintf("【你的回复】：%s\n\n", msg.Content))
		} else if msg.Role == "user" {
			sb.WriteString(fmt.Sprintf("【老师说】：%s\n\n", msg.Content))
		}
	}
	sb.WriteString(fmt.Sprintf("【老师说】：%s\n\n", currentUserMsg))
	sb.WriteString("请根据以上对话继续。如果信息已经足够，请输出 <assessment_result> 判定结果。")
	return sb.String()
}

// estimateStep 根据用户消息数估算当前步骤
func estimateStep(userMsgCount int) string {
	switch {
	case userMsgCount <= 1:
		return "q1"
	case userMsgCount == 2:
		return "q2"
	case userMsgCount == 3:
		return "q3"
	case userMsgCount == 4:
		return "q4"
	case userMsgCount == 5:
		return "q5"
	default:
		return "q6"
	}
}

// filterValidPriorities 过滤有效的质量关注点
func filterValidPriorities(input []string) []string {
	validSet := make(map[string]bool)
	for _, p := range models.AssessmentPriorityOptions {
		validSet[p] = true
	}
	var result []string
	for _, p := range input {
		if validSet[p] {
			result = append(result, p)
		}
	}
	if result == nil {
		result = []string{}
	}
	return result
}

// buildStagesConfigFromProfile 根据画像构建 stages_config
func buildStagesConfigFromProfile(profile *models.TeachingProfile) []models.StageFlowItem {
	allStages := []string{"analyze", "design", "write", "review", "revise"}
	recommendedSet := make(map[string]bool)
	for _, s := range profile.RecommendedStages {
		recommendedSet[s] = true
	}

	var items []models.StageFlowItem
	order := 1
	for _, code := range allStages {
		items = append(items, models.StageFlowItem{
			StageCode: code,
			Enabled:   recommendedSet[code],
			Order:     order,
		})
		order++
	}
	return items
}

// parseLessonStructureFromDesc 从自然语言描述解析教案结构块
func parseLessonStructureFromDesc(desc string) []models.LessonStructureBlock {
	normalized := desc
	for _, sep := range []string{"、", "，", ",", "；", ";", "\n", "/"} {
		normalized = strings.ReplaceAll(normalized, sep, "|")
	}

	parts := strings.Split(normalized, "|")
	var blocks []models.LessonStructureBlock
	order := 1
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		blocks = append(blocks, models.LessonStructureBlock{
			Name:     name,
			Required: true,
			Order:    order,
		})
		order++
	}
	return blocks
}
