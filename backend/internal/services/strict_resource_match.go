package services

// strict_resource_match.go — 助手与配方的统一学科、具体年级、场景闸门
//
// 本文件只做确定性校验，不调用AI。
// 自动匹配、老师旧偏好、显式ID、导入教案和旧教案运行时注入
// 应复用同一套规则，防止不同入口出现不同口径。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

var (
	// ErrAssistantLessonMismatch 表示助手存在且可能可见，
	// 但不适用于当前教案的学科、具体年级或当前场景。
	ErrAssistantLessonMismatch = errors.New(
		"AI助手与当前教案的学科、具体年级或场景不匹配",
	)

	// ErrRecipeLessonMismatch 表示配方存在，
	// 但不适用于当前教案的学科和具体年级。
	ErrRecipeLessonMismatch = errors.New(
		"备课配方与当前教案的学科或具体年级不匹配",
	)
)

// strictAssistantMatchesListItem 校验助手列表项。
// scene为空时只校验学科和年级；非空时还要求明确包含该场景。
func strictAssistantMatchesListItem(
	item *models.AIAssistantListItem,
	subject string,
	grade string,
	scene string,
) bool {
	if item == nil ||
		!utils.IsStrictSubjectGradeMatch(
			item.Subject,
			item.GradeRange,
			subject,
			grade,
		) {
		return false
	}

	scene = strings.TrimSpace(scene)
	if scene == "" {
		return true
	}

	for _, candidate := range item.Scenes {
		if strings.TrimSpace(candidate) == scene {
			return true
		}
	}

	return false
}

// strictAssistantMatchesEntity 校验助手完整实体。
func strictAssistantMatchesEntity(
	assistant *models.AIAssistant,
	subject string,
	grade string,
	scene string,
) bool {
	if assistant == nil ||
		!utils.IsStrictSubjectGradeMatch(
			assistant.Subject,
			assistant.GradeRange,
			subject,
			grade,
		) {
		return false
	}

	scene = strings.TrimSpace(scene)
	if scene == "" {
		return true
	}

	var scenes []string
	if err := json.Unmarshal(
		[]byte(assistant.Scenes),
		&scenes,
	); err != nil {
		return false
	}

	for _, candidate := range scenes {
		if strings.TrimSpace(candidate) == scene {
			return true
		}
	}

	return false
}

// normalizeStrictResourceScope 规范化资源写入时的学科和具体年级。
//
// 学科只做非空和首尾空格清洗，保留平台未来增加新学科的能力。
// 年级必须明确对应1—12中的一个具体年级，并统一为标准中文名称。
func normalizeStrictResourceScope(
	subject string,
	grade string,
) (
	normalizedSubject string,
	normalizedGrade string,
	ok bool,
) {
	normalizedSubject = strings.TrimSpace(subject)
	if normalizedSubject == "" {
		return "", "", false
	}

	normalizedGrade, ok =
		utils.NormalizeGradeToStandardLabel(grade)
	if !ok {
		return "", "", false
	}

	return normalizedSubject, normalizedGrade, true
}

// strictRecipeMatchesEntity 校验配方学科与具体年级。
func strictRecipeMatchesEntity(
	recipe *models.TeachingRecipe,
	subject string,
	grade string,
) bool {
	return recipe != nil &&
		utils.IsStrictSubjectGradeMatch(
			recipe.Subject,
			recipe.GradeRange,
			subject,
			grade,
		)
}

// ValidateAssistantForLesson 对助手执行运行时严格校验，但不增加使用次数。
//
// 用途：
//   - 读取和写入老师助手偏好；
//   - 列表或入口的后端二次校验；
//   - 仅判断可用性而尚未真正调用助手的场景。
func (s *AIAssistantService) ValidateAssistantForLesson(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
	subject string,
	grade string,
	scene string,
) (*models.AIAssistant, error) {
	assistant, err := repository.GetAIAssistantByID(
		ctx,
		strings.TrimSpace(id),
	)
	if err != nil {
		return nil, err
	}

	if !assistant.IsActive {
		return nil, repository.ErrAIAssistantInactive
	}

	if !s.canView(actor, assistant) {
		return nil, ErrAssistantPermDenied
	}

	if !strictAssistantMatchesEntity(
		assistant,
		subject,
		grade,
		scene,
	) {
		return nil, ErrAssistantLessonMismatch
	}

	return assistant, nil
}

// LoadActiveAssistantForLessonUse 是真正调用助手时的最终防线。
//
// 全部严格校验通过后才递增使用次数，错误助手不会污染使用统计。
func (s *AIAssistantService) LoadActiveAssistantForLessonUse(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
	subject string,
	grade string,
	scene string,
) (*models.AIAssistant, error) {
	assistant, err := s.ValidateAssistantForLesson(
		ctx,
		actor,
		id,
		subject,
		grade,
		scene,
	)
	if err != nil {
		return nil, err
	}

	go func(assistantID string) {
		_ = repository.IncrementAIAssistantUseCount(
			context.Background(),
			assistantID,
		)
	}(assistant.ID)

	return assistant, nil
}

// loadRecipeForLesson 加载并严格校验一个配方。
//
// 不改变数据库，也不自动换用其它配方。
func loadRecipeForLesson(
	ctx context.Context,
	recipeID string,
	subject string,
	grade string,
) (*models.TeachingRecipe, error) {
	recipe, err := repository.GetRecipeByID(
		ctx,
		strings.TrimSpace(recipeID),
	)
	if err != nil {
		return nil, err
	}

	if recipe.Status != recipeAutoMountActiveStatus ||
		!strictRecipeMatchesEntity(
			recipe,
			subject,
			grade,
		) {
		return nil, ErrRecipeLessonMismatch
	}

	return recipe, nil
}
