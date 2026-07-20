package services

// lesson_plan_import_helpers.go
//
// 本文件承载已有教案导入的纯校验、对象构造与阶段快照辅助函数。
// 主编排保留在lesson_plan_import_service.go，避免单文件超过600行。
//
// 这些函数不启动后台任务，也不直接修改阶段状态；
// 教育域解析和显式创建仍通过统一Repository能力完成。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// normalizeLessonPlanImportSourceType 严格规范化导入来源。
func normalizeLessonPlanImportSourceType(
	sourceType string,
) (string, error) {
	normalized := strings.ToLower(
		strings.TrimSpace(sourceType),
	)

	switch normalized {
	case "paste", "docx", "pdf":
		return normalized, nil
	default:
		return "", fmt.Errorf(
			"%w: %q",
			ErrLPGenImportSourceInvalid,
			sourceType,
		)
	}
}

// resolveImportedLessonPlanCreationDomain
// 实时读取用户角色并解析导入教案唯一具体教学域。
func resolveImportedLessonPlanCreationDomain(
	ctx context.Context,
	authorID string,
	deps lessonPlanImportCreationDeps,
) (string, error) {
	user, err := deps.findUser(
		ctx,
		authorID,
	)
	if err != nil {
		lpGenLog.Error(
			"导入教案读取用户实时角色失败",
			"author", authorID,
			"error", err,
		)
		return "", fmt.Errorf(
			"%w: 读取用户实时角色失败",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}
	if user == nil ||
		strings.TrimSpace(user.Role) == "" {
		return "", fmt.Errorf(
			"%w: 用户实时角色为空",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}

	domain, err := deps.resolveEducationDomain(
		ctx,
		authorID,
		user.Role,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.
				ErrLessonPlanCreationEducationDomainConflict,
		):
			return "", fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainConflict,
				err,
			)

		case errors.Is(
			err,
			repository.
				ErrLessonPlanCreationEducationDomainUnavailable,
		),
			errors.Is(
				err,
				repository.
					ErrRegionAdminEducationDomainNotReady,
			):
			return "", fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainRequired,
				err,
			)

		default:
			lpGenLog.Error(
				"导入教案解析教育域失败",
				"author", authorID,
				"role", user.Role,
				"error", err,
			)
			return "", fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainResolveFailed,
				err,
			)
		}
	}

	domain = strings.ToLower(
		strings.TrimSpace(domain),
	)
	if !models.IsTeachingEducationDomain(domain) {
		return "", fmt.Errorf(
			"%w: 解析结果不是具体教学域",
			ErrLPCreationEducationDomainResolveFailed,
		)
	}

	return domain, nil
}

// buildImportedLessonPlan 构造待显式写域的导入教案。
func buildImportedLessonPlan(
	req *models.ImportExistingPlanRequest,
	authorID string,
	duration int,
) (*models.LessonPlan, error) {
	if req == nil {
		return nil, errors.New(
			"导入教案请求不能为空",
		)
	}

	lessonPlan := &models.LessonPlan{
		Title: fmt.Sprintf(
			"%s %s — %s",
			req.Grade,
			req.Subject,
			req.Topic,
		),
		Subject:           req.Subject,
		Grade:             req.Grade,
		Topic:             req.Topic,
		DurationMinutes:   duration,
		ContentMarkdown:   req.ContentMarkdown,
		ContentStructured: "{}",
		Status:            models.LPStatusDraft,
		Visibility:        models.LPVisibilityPersonal,
		AuthorID:          authorID,
		ConversationLog:   "[]",
	}

	if req.GroupID != "" {
		groupID := req.GroupID
		lessonPlan.GroupID = &groupID
	}

	if req.RecipeID != "" {
		recipeID := req.RecipeID
		lessonPlan.RecipeID = &recipeID
	}

	if len(req.TextbookPageIDs) > 0 {
		textbookJSON, err := json.Marshal(
			req.TextbookPageIDs,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"序列化导入教案课本图片ID失败: %w",
				err,
			)
		}
		lessonPlan.TextbookPageIDs =
			string(textbookJSON)
	}

	return lessonPlan, nil
}

// createImportedLessonPlanWithEducationDomain
// 显式创建并再次核对数据库最终教育域快照。
func createImportedLessonPlanWithEducationDomain(
	ctx context.Context,
	lessonPlan *models.LessonPlan,
	creationDomain string,
	deps lessonPlanImportCreationDeps,
) error {
	if lessonPlan == nil {
		return errors.New("导入教案对象为空")
	}

	if err := deps.createWithEducationDomain(
		ctx,
		lessonPlan,
		creationDomain,
	); err != nil {
		switch {
		case errors.Is(
			err,
			repository.
				ErrLessonPlanExplicitEducationDomainRequired,
		),
			errors.Is(
				err,
				repository.
					ErrLessonPlanExplicitEducationDomainSnapshotMismatch,
			):
			return fmt.Errorf(
				"%w: %v",
				ErrLPCreationEducationDomainResolveFailed,
				err,
			)

		default:
			return fmt.Errorf(
				"创建导入教案失败: %w",
				err,
			)
		}
	}

	storedDomain := strings.ToLower(
		strings.TrimSpace(
			lessonPlan.EducationDomain,
		),
	)
	if storedDomain != creationDomain ||
		!models.IsTeachingEducationDomain(
			storedDomain,
		) {
		return fmt.Errorf(
			"%w: service=%s database=%s",
			ErrLPCreationEducationDomainResolveFailed,
			creationDomain,
			storedDomain,
		)
	}

	return nil
}

// buildImportedLessonPlanStageOutputs
// 把review之前的阶段标记为skipped，并把review标记为in_progress。
//
// 不调用普通SkipStage，避免产生自动开场白、阶段分隔符、
// 质量评估goroutine和多次非事务写入。
func buildImportedLessonPlanStageOutputs(
	snapshots []models.StageConfigSnapshot,
) (
	[]models.WorkshopStageOutput,
	[]string,
	error,
) {
	reviewIndex := -1
	for index, snapshot := range snapshots {
		if strings.TrimSpace(
			snapshot.StageCode,
		) == "review" {
			reviewIndex = index
			break
		}
	}

	if reviewIndex < 0 {
		return nil, nil,
			ErrLPGenImportReviewStageRequired
	}

	outputs := make(
		[]models.WorkshopStageOutput,
		0,
		reviewIndex+1,
	)
	skippedStages := make(
		[]string,
		0,
		reviewIndex,
	)

	for index := 0; index <= reviewIndex; index++ {
		snapshot := snapshots[index]
		stageCode := strings.TrimSpace(
			snapshot.StageCode,
		)
		if stageCode == "" {
			return nil, nil, errors.New(
				"导入流程存在空阶段代码",
			)
		}

		status := models.StageOutputSkipped
		if index == reviewIndex {
			status = models.StageOutputInProgress
		} else {
			skippedStages = append(
				skippedStages,
				stageCode,
			)
		}

		outputs = append(
			outputs,
			models.WorkshopStageOutput{
				StageCode:            stageCode,
				StageOrder:           snapshot.StageOrder,
				StructuredOutput:     "{}",
				NarrativeOutput:      "",
				ConversationSnapshot: "[]",
				Status:               status,
			},
		)
	}

	return outputs, skippedStages, nil
}

// buildImportOpeningMessage 构建导入成功开场消息。
func buildImportOpeningMessage(
	req *models.ImportExistingPlanRequest,
	skippedStages []string,
) *models.ConversationMessage {
	sourceLabel := map[string]string{
		"paste": "粘贴文本",
		"docx":  "Word文档",
		"pdf":   "PDF文件",
	}[req.SourceType]
	if sourceLabel == "" {
		sourceLabel = "已有文档"
	}

	contentLength := len(
		[]rune(
			strings.TrimSpace(
				req.ContentMarkdown,
			),
		),
	)

	skippedDescription := strings.Join(
		skippedStages,
		" → ",
	)
	if skippedDescription == "" {
		skippedDescription =
			"无需跳过前置阶段"
	} else {
		skippedDescription =
			"已跳过：" + skippedDescription
	}

	content := fmt.Sprintf(
		`您好！我已成功导入您的 **%s %s「%s」** 教案。

**导入信息**
- 来源：%s
- 字数：约 %d 字
- %s

**当前状态**
教案正文、阶段快照和评审入口均已写入系统，正在对您的教案进行 **AI质量评审**，请稍候……评审完成后会在右侧面板显示详细评分和改进建议。

**接下来您可以：**
1. 等待右侧出现AI评审报告
2. 现在就告诉我您希望优化哪些部分，我来帮您修改
3. 评审满意后进入修订定稿`,
		req.Grade,
		req.Subject,
		req.Topic,
		sourceLabel,
		contentLength,
		skippedDescription,
	)

	return &models.ConversationMessage{
		ID:        generateMsgID(),
		Role:      models.ConvRoleAssistant,
		Type:      models.ConvMsgTypeText,
		Content:   content,
		CreatedAt: time.Now(),
	}
}
