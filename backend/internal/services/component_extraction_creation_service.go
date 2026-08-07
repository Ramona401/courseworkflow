package services

// component_extraction_creation_service.go — 组件萃取安全创建。
//
// 创建时始终重新读取来源教案，并使用lesson_plans.education_domain快照。
// 不使用审核人当前域、作者当前域、组织当前域或数据库K12兜底。
// 组件与component_extractions记录通过Repository事务同时创建。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	aiClient "tedna/internal/ai"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// validateExtractionLessonDomain 校验来源教案的资源快照域。
func validateExtractionLessonDomain(
	domain string,
) (string, error) {
	domain = strings.ToLower(
		strings.TrimSpace(domain),
	)

	if !models.IsTeachingEducationDomain(
		domain,
	) {
		return "",
			ErrComponentEducationDomainInvalid
	}

	return domain, nil
}

// SaveExtractionFromChat 保存对话中产生的组件萃取。
//
// 保留原方法签名，内部改为重新读取教案。
// 只有教案作者本人可以发起对话萃取。
func (s *ComponentService) SaveExtractionFromChat(
	ctx context.Context,
	planID string,
	sourceContent string,
	extractionType string,
	displayLabel string,
	designLogic string,
	createdBy string,
) error {
	lessonPlan, err :=
		repository.GetLessonPlanByID(
			ctx,
			strings.TrimSpace(planID),
		)
	if err != nil {
		return err
	}

	createdBy = strings.TrimSpace(
		createdBy,
	)

	if lessonPlan.AuthorID != createdBy {
		return ErrComponentEducationDomainForbidden
	}

	educationDomain, err :=
		validateExtractionLessonDomain(
			lessonPlan.EducationDomain,
		)
	if err != nil {
		return err
	}

	extractionType = strings.TrimSpace(
		extractionType,
	)

	if !models.IsValidLibraryType(
		extractionType,
	) {
		return ErrComponentLibTypeInvalid
	}

	displayLabel = strings.TrimSpace(
		displayLabel,
	)

	if displayLabel == "" {
		return ErrComponentLabelRequired
	}

	sourcePlanID := lessonPlan.ID
	creatorID := createdBy

	component := &models.LessonPlanComponent{
		EducationDomain: educationDomain,
		LibraryType:     extractionType,
		Subject:         lessonPlan.Subject,
		GradeRange:      lessonPlan.Grade,
		DisplayLabel:    displayLabel,
		DesignLogic:     designLogic,
		Source:          "ai_extracted",
		SourceRef:       lessonPlan.ID,
		Scope:           models.ScopeGroup,
		CreatedBy:       &creatorID,
		ReviewStatus:    models.ComponentReviewCaptured,
		Status:          "active",
	}

	extraction := &models.ComponentExtraction{
		SourceType:         "conversation",
		SourceLessonPlanID: &sourcePlanID,
		SourceContent:      sourceContent,
		ExtractionType:     extractionType,
		Status:             "pending",
		CreatedBy:          &creatorID,
	}

	return repository.
		CreateExtractedComponentWithRecord(
			ctx,
			component,
			extraction,
		)
}

// AutoExtractFromLessonPlan 从高质量教案自动萃取组件。
//
// 为兼容两条现有审核调用链保留原签名。
// 传入的正文、学科和年级可能是异步任务旧快照，因此不再信任，
// 方法内部重新读取数据库中的最新教案。
func (s *ComponentService) AutoExtractFromLessonPlan(
	ctx context.Context,
	planID string,
	planContent string,
	subject string,
	grade string,
	reviewerID string,
) error {
	_ = planContent
	_ = subject
	_ = grade

	lessonPlan, err :=
		repository.GetLessonPlanByID(
			ctx,
			strings.TrimSpace(planID),
		)
	if err != nil {
		return err
	}

	educationDomain, err :=
		validateExtractionLessonDomain(
			lessonPlan.EducationDomain,
		)
	if err != nil {
		return err
	}

	latestContent := strings.TrimSpace(
		lessonPlan.ContentMarkdown,
	)

	if latestContent == "" {
		compLog.Warn(
			"教案正文为空，跳过自动萃取",
			"plan_id", lessonPlan.ID,
		)
		return nil
	}

	reviewerID = strings.TrimSpace(
		reviewerID,
	)

	if reviewerID == "" {
		return ErrComponentEducationDomainForbidden
	}

	aiConfig, err :=
		aiClient.GetEffectiveConfig(
			s.cfg.GetAESKey(),
			"scanner",
			"",
			"",
			"",
		)
	if err != nil {
		return fmt.Errorf(
			"获取AI萃取配置失败: %w",
			err,
		)
	}

	systemPrompt := `你是一位教育专家，负责从优秀教案中提取可复用的教学设计逻辑片段。
请识别2-4个高价值教学设计片段，要求具有明确教学意图、可复用性和可操作性。

严格输出JSON数组，不要输出Markdown代码块或其它文字：
[
  {
    "extraction_type": "activity_design",
    "display_label": "🎯 简短标签",
    "design_logic": "核心设计逻辑",
    "source_snippet": "来源教案片段"
  }
]

extraction_type只能使用：
activity_design、questioning_strategy、pedagogy、
assessment_strategy、cross_subject、scenario_material。`

	userPrompt := fmt.Sprintf(
		"请从以下%s课程（学习层级：%s）的教案中提取可复用设计：\n\n%s",
		lessonPlan.Subject,
		lessonPlan.Grade,
		latestContent,
	)

	planIDCopy := lessonPlan.ID
	reviewerIDCopy := reviewerID

	traceContext := &aiClient.TraceContext{
		SceneCode:    "scanner",
		LessonPlanID: &planIDCopy,
		UserID:       &reviewerIDCopy,
	}

	result, err := aiClient.CallAI(
		aiConfig,
		systemPrompt,
		userPrompt,
		traceContext,
	)
	if err != nil {
		return fmt.Errorf(
			"AI萃取调用失败: %w",
			err,
		)
	}

	items, err := parseAutoExtractionResult(
		result.Content,
	)
	if err != nil {
		return err
	}

	successCount := 0

	for _, item := range items {
		extractionType :=
			strings.TrimSpace(
				item.ExtractionType,
			)

		if !isAutoExtractionLibraryType(
			extractionType,
		) {
			continue
		}

		displayLabel :=
			strings.TrimSpace(
				item.DisplayLabel,
			)

		designLogic :=
			strings.TrimSpace(
				item.DesignLogic,
			)

		if displayLabel == "" ||
			designLogic == "" {
			continue
		}

		sourcePlanID := lessonPlan.ID
		creatorID := reviewerIDCopy

		component := &models.LessonPlanComponent{
			EducationDomain: educationDomain,
			LibraryType:     extractionType,
			Subject:         lessonPlan.Subject,
			GradeRange:      lessonPlan.Grade,
			DisplayLabel:    displayLabel,
			DesignLogic:     designLogic,
			Source:          "ai_extracted",
			SourceRef:       lessonPlan.ID,
			Scope:           models.ScopeGroup,
			CreatedBy:       &creatorID,
			ReviewStatus:    models.ComponentReviewPending,
			Status:          "active",
		}

		extraction := &models.ComponentExtraction{
			SourceType:         "lesson_plan",
			SourceLessonPlanID: &sourcePlanID,
			SourceContent: strings.TrimSpace(
				item.SourceSnippet,
			),
			ExtractionType: extractionType,
			Status:         "pending",
			CreatedBy:      &creatorID,
		}

		err := repository.
			CreateExtractedComponentWithRecord(
				ctx,
				component,
				extraction,
			)

		if err != nil {
			compLog.Warn(
				"事务创建自动萃取失败",
				"plan_id", lessonPlan.ID,
				"education_domain", educationDomain,
				"error", err,
			)
			continue
		}

		successCount++
	}

	compLog.Info(
		"自动萃取完成",
		"plan_id", lessonPlan.ID,
		"education_domain", educationDomain,
		"found", len(items),
		"saved", successCount,
	)

	return nil
}

// autoExtractionItem 表示AI返回的单个萃取项。
type autoExtractionItem struct {
	ExtractionType string `json:"extraction_type"`
	DisplayLabel   string `json:"display_label"`
	DesignLogic    string `json:"design_logic"`
	SourceSnippet  string `json:"source_snippet"`
}

// parseAutoExtractionResult 解析AI返回的JSON数组。
func parseAutoExtractionResult(
	content string,
) ([]autoExtractionItem, error) {
	jsonText := strings.TrimSpace(
		content,
	)

	arrayStart := strings.Index(
		jsonText,
		"[",
	)

	arrayEnd := strings.LastIndex(
		jsonText,
		"]",
	)

	if arrayStart >= 0 &&
		arrayEnd > arrayStart {
		jsonText =
			jsonText[arrayStart : arrayEnd+1]
	}

	if !strings.HasPrefix(
		jsonText,
		"[",
	) {
		jsonText = "[" +
			jsonText +
			"]"
	}

	var items []autoExtractionItem

	if err := json.Unmarshal(
		[]byte(jsonText),
		&items,
	); err != nil {
		return nil,
			fmt.Errorf(
				"解析AI萃取JSON失败: %w",
				err,
			)
	}

	return items, nil
}

// isAutoExtractionLibraryType 判断类型是否属于本萃取通道白名单。
func isAutoExtractionLibraryType(
	libraryType string,
) bool {
	switch libraryType {
	case models.LibActivityDesign:
		return true

	case models.LibQuestioningStrategy:
		return true

	case models.LibPedagogy:
		return true

	case models.LibAssessmentStrategy:
		return true

	case models.LibCrossSubject:
		return true

	case models.LibScenarioMaterial:
		return true

	default:
		return false
	}
}
