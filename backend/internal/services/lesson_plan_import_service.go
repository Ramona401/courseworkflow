package services

// lesson_plan_import_service.go — 已有教案导入服务
//
// v108新增：支持老师将现有教案（Word/PDF/粘贴文本）导入系统
//
// 流程：
//   1. 创建教案记录，写入已有正文
//   2. 初始化阶段配置（和正常备课相同）
//   3. 批量跳过 analyze / design / write 三个阶段
//   4. 切换到 review 阶段，发送导入成功开场白
//   5. 异步触发AI评审（executeAIReviewAsync）
//   6. 返回教案+开场白给前端，前端建立SSE等待评审结果

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ImportExistingPlan 导入已有教案主流程
func (s *LessonPlanGenService) ImportExistingPlan(
	ctx context.Context,
	req *models.ImportExistingPlanRequest,
	authorID string,
) (*models.ImportExistingPlanResponse, error) {

	// ---- 参数校验 ----
	if strings.TrimSpace(req.Subject) == "" {
		return nil, ErrLPGenSubjectRequired
	}
	if strings.TrimSpace(req.Grade) == "" {
		return nil, ErrLPGenGradeRequired
	}
	if strings.TrimSpace(req.Topic) == "" {
		return nil, ErrLPGenTopicRequired
	}
	if strings.TrimSpace(req.ContentMarkdown) == "" {
		return nil, fmt.Errorf("教案内容不能为空")
	}

	dur := req.DurationMinutes
	if dur <= 0 {
		dur = 45
	}

	// ---- 1. 创建教案记录，写入已有正文 ----
	title := fmt.Sprintf("%s %s — %s", req.Grade, req.Subject, req.Topic)
	lp := &models.LessonPlan{
		Title:           title,
		Subject:         req.Subject,
		Grade:           req.Grade,
		Topic:           req.Topic,
		DurationMinutes: dur,
		ContentMarkdown: strings.TrimSpace(req.ContentMarkdown),
		Status:          models.LPStatusDraft,
		Visibility:      models.LPVisibilityPersonal,
		AuthorID:        authorID,
		ConversationLog: "[]",
	}
	if req.GroupID != "" {
		lp.GroupID = &req.GroupID
	}
	if req.RecipeID != "" {
		lp.RecipeID = &req.RecipeID
	}
	if len(req.TextbookPageIDs) > 0 {
		tbJSON, _ := json.Marshal(req.TextbookPageIDs)
		lp.TextbookPageIDs = string(tbJSON)
	}

	if err := repository.CreateLessonPlan(ctx, lp); err != nil {
		return nil, fmt.Errorf("创建教案失败: %w", err)
	}
	lpGenLog.Info("导入已有教案-教案记录创建",
		"plan_id", lp.ID, "topic", req.Topic,
		"author", authorID, "source_type", req.SourceType,
		"content_len", len(req.ContentMarkdown))

	// ---- 2. 初始化阶段配置 ----
	recipeStagesConfig := ""
	if req.RecipeID != "" {
		if recipe, err := repository.GetRecipeByID(ctx, req.RecipeID); err == nil {
			recipeStagesConfig = recipe.StagesConfig
		}
	}

	snapshots, err := s.stageService.InitStagesForPlan(ctx, lp.ID, recipeStagesConfig, req.RecipeID)
	if err != nil {
		lpGenLog.Error("导入已有教案-阶段初始化失败", "plan_id", lp.ID, "error", err)
		return nil, fmt.Errorf("阶段初始化失败: %w", err)
	}

	lp.CurrentStage = snapshots[0].StageCode
	configJSON, _ := json.Marshal(snapshots)
	lp.StageConfig = string(configJSON)

	// ---- 3. 找到 review 阶段位置，批量跳过之前所有阶段 ----
	reviewIdx := -1
	for i, snap := range snapshots {
		if snap.StageCode == "review" {
			reviewIdx = i
			break
		}
	}

	skippedStages := []string{}

	if reviewIdx > 0 {
		// 依次跳过 review 之前的所有阶段（analyze / design / write）
		// SkipStage 每次跳过当前阶段并进入下一个，所以循环调用 reviewIdx 次
		for i := 0; i < reviewIdx; i++ {
			// 每次跳过后当前阶段已变化，直接调用 SkipStage 跳过当前阶段
			currentSnap := snapshots[i]
			if _, skipErr := s.stageService.SkipStage(ctx, lp.ID, currentSnap.StageCode, authorID); skipErr != nil {
				lpGenLog.Warn("导入已有教案-跳过阶段失败",
					"plan_id", lp.ID, "stage", currentSnap.StageCode, "error", skipErr)
				// 跳过失败不阻断主流程
			} else {
				skippedStages = append(skippedStages, currentSnap.StageCode)
				lpGenLog.Info("导入已有教案-跳过阶段成功",
					"plan_id", lp.ID, "stage", currentSnap.StageCode)
			}
		}
	} else {
		lpGenLog.Warn("导入已有教案-未找到review阶段，停留在当前阶段", "plan_id", lp.ID)
	}

	// ---- 4. 确保教案正文写入（防止阶段跳转副作用覆盖内容）----
	if err := repository.UpdateLessonPlanContent(
		ctx, lp.ID, lp.Title,
		strings.TrimSpace(req.ContentMarkdown), "{}", dur,
	); err != nil {
		lpGenLog.Warn("导入已有教案-确认写入正文失败", "plan_id", lp.ID, "error", err)
	}

	// ---- 5. 构建导入成功的AI开场白 ----
	openingMsg := buildImportOpeningMessage(req, skippedStages)
	if err2 := s.appendMessage(ctx, lp.ID, openingMsg); err2 != nil {
		lpGenLog.Warn("导入已有教案-写入开场消息失败", "plan_id", lp.ID, "error", err2)
	}

	// ---- 6. 异步触发AI评审 ----
	go func() {
		bgCtx := context.Background()
		// 稍等确保阶段写库完成
		time.Sleep(800 * time.Millisecond)
		freshLP, freshErr := repository.GetLessonPlanByID(bgCtx, lp.ID)
		if freshErr != nil {
			lpGenLog.Warn("导入已有教案-加载最新教案失败", "plan_id", lp.ID, "error", freshErr)
			return
		}
		lpGenLog.Info("导入已有教案-开始异步AI评审", "plan_id", lp.ID)
		s.executeAIReviewAsync(bgCtx, freshLP)
	}()

	// ---- 7. 刷新阶段状态，让前端拿到最新 current_stage ----
	if freshLP, err3 := repository.GetLessonPlanByID(ctx, lp.ID); err3 == nil {
		lp.CurrentStage = freshLP.CurrentStage
		lp.StageConfig = freshLP.StageConfig
	}

	lpGenLog.Info("导入已有教案-主流程完成",
		"plan_id", lp.ID,
		"skipped_stages", skippedStages,
		"current_stage", lp.CurrentStage)

	return &models.ImportExistingPlanResponse{
		Plan:           lp,
		OpeningMessage: openingMsg,
		SkippedStages:  skippedStages,
	}, nil
}

// buildImportOpeningMessage 构建导入成功的AI开场白消息
func buildImportOpeningMessage(req *models.ImportExistingPlanRequest, skippedStages []string) *models.ConversationMessage {
	sourceLabel := map[string]string{
		"paste": "粘贴文本",
		"docx":  "Word文档",
		"pdf":   "PDF文件",
	}[req.SourceType]
	if sourceLabel == "" {
		sourceLabel = "已有文档"
	}

	contentLen := len([]rune(strings.TrimSpace(req.ContentMarkdown)))
	skippedDesc := strings.Join(skippedStages, " → ")
	if skippedDesc != "" {
		skippedDesc = "已跳过：" + skippedDesc
	}

	content := fmt.Sprintf(`您好！我已成功导入您的 **%s %s「%s」** 教案。

**导入信息**
- 来源：%s
- 字数：约 %d 字
- %s

**当前状态**
教案正文已写入系统，正在对您的教案进行 **AI质量评审**，请稍候……评审完成后会在右侧面板显示详细评分和改进建议。

**接下来您可以：**
1. 等待右侧出现AI评审报告
2. 现在就告诉我您希望优化哪些部分，我来帮您修改
3. 评审满意后点击「完成本阶段」进入修订定稿`,
		req.Grade, req.Subject, req.Topic,
		sourceLabel, contentLen, skippedDesc)

	return &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		Type:      models.ConvMsgTypeText,
		Content:   content,
		CreatedAt: time.Now(),
	}
}
