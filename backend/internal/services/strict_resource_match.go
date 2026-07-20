package services

// strict_resource_match.go — 助手与配方的自动严格、手动授权双通道
//
// 自动匹配通道：
//   - 助手：课程、具体教学层级和场景全部严格一致；
//   - 配方：课程和具体教学层级全部严格一致。
//
// 具体教学层级包括：
//   - K12一年级至高三；
//   - 职业教育中职Ⅰ、Ⅱ、Ⅲ年级；
//   - 成人教育入门、进阶、高级、管理者。
//
// 手动选择通道：
//   - 助手：必须active、当前用户可见、同课程并支持当前场景；
//     具体层级不作为阻断条件；
//   - 配方：必须active且当前用户有权使用；
//     课程和层级不作为阻断条件，老师明确选择拥有最终决定权。
//
// 自动与手动使用独立函数，避免为了放宽手动入口而误伤自动匹配边界。

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
	// ErrAssistantLessonMismatch 表示助手不适用于自动匹配要求。
	ErrAssistantLessonMismatch = errors.New(
		"AI助手与当前教案的课程、具体教学层级或场景不匹配",
	)

	// ErrAssistantManualMismatch 表示手动选择助手的课程或场景不匹配。
	ErrAssistantManualMismatch = errors.New(
		"AI助手与当前教案的课程或场景不匹配",
	)

	// ErrAssistantEducationDomainMismatch 表示助手资源域不匹配。
	ErrAssistantEducationDomainMismatch = errors.New(
		"AI助手与当前教学教育域不匹配",
	)

	// ErrRecipeLessonMismatch 表示配方不适用于自动匹配要求。
	ErrRecipeLessonMismatch = errors.New(
		"备课配方与当前教案的课程或具体教学层级不匹配",
	)

	// ErrRecipeManualSelectionDenied 表示手动指定配方不可用。
	ErrRecipeManualSelectionDenied = errors.New(
		"备课配方不存在、未启用或当前用户无权使用",
	)
)

// strictAssistantMatchesListItem 校验自动助手列表项。
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

	return assistantScenesContain(
		item.Scenes,
		scene,
	)
}

// strictAssistantMatchesEntity 校验自动助手完整实体。
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

	scenes, ok :=
		parseAssistantScenes(
			assistant.Scenes,
		)

	if !ok {
		return false
	}

	return assistantScenesContain(
		scenes,
		scene,
	)
}

// assistantResourceEducationDomainMatches 校验助手资源教育域。
func assistantResourceEducationDomainMatches(
	actor *AssistantActorContext,
	assistant *models.AIAssistant,
) bool {
	if actor == nil ||
		assistant == nil {
		return false
	}

	return models.ResourceEducationDomainMatches(
		assistant.EducationDomain,
		actor.EducationDomain,
	)
}

// manualAssistantMatchesEntity 校验老师手动选择的助手。
//
// 手动选择不限制具体年级或学习层级，但仍保留课程和场景硬门槛。
func manualAssistantMatchesEntity(
	assistant *models.AIAssistant,
	subject string,
	scene string,
) bool {
	if assistant == nil ||
		strings.TrimSpace(
			assistant.Subject,
		) != strings.TrimSpace(subject) {
		return false
	}

	scenes, ok :=
		parseAssistantScenes(
			assistant.Scenes,
		)

	if !ok {
		return false
	}

	return assistantScenesContain(
		scenes,
		scene,
	)
}

// parseAssistantScenes 解析助手场景JSON。
func parseAssistantScenes(
	raw string,
) (
	[]string,
	bool,
) {
	var scenes []string

	if err := json.Unmarshal(
		[]byte(raw),
		&scenes,
	); err != nil {
		return nil, false
	}

	return scenes, true
}

// assistantScenesContain 判断场景列表是否包含当前场景。
func assistantScenesContain(
	scenes []string,
	scene string,
) bool {
	scene = strings.TrimSpace(scene)

	if scene == "" {
		return true
	}

	for _, candidate := range scenes {
		if strings.TrimSpace(candidate) ==
			scene {
			return true
		}
	}

	return false
}

// normalizeStrictResourceScope 规范化配方等自动匹配资源的课程与具体层级。
//
// 只允许具体层级，不允许学段、范围或不限值。
// 具体层级由utils.NormalizeGradeToStandardLabel跨教育域统一处理。
func normalizeStrictResourceScope(
	subject string,
	grade string,
) (
	normalizedSubject string,
	normalizedGrade string,
	ok bool,
) {
	normalizedSubject =
		strings.TrimSpace(subject)

	if normalizedSubject == "" {
		return "", "", false
	}

	normalizedGrade, ok =
		utils.NormalizeGradeToStandardLabel(
			grade,
		)

	if !ok {
		return "", "", false
	}

	return normalizedSubject,
		normalizedGrade,
		true
}

// normalizeAssistantResourceScope 规范化助手的课程和适用层级。
//
// 助手允许：
//   1. 各教育域具体层级，可参与平台自动匹配；
//   2. K12小学/初中/高中；
//   3. 中职不限年级；
//   4. 成人不限层级；
//   5. 空字符串，兼容历史“不限年级”数据。
//
// 通用层级只供老师手动选择，不参与自动匹配。
func normalizeAssistantResourceScope(
	subject string,
	grade string,
) (
	normalizedSubject string,
	normalizedGrade string,
	ok bool,
) {
	normalizedSubject =
		strings.TrimSpace(subject)

	if normalizedSubject == "" {
		return "", "", false
	}

	rawGrade :=
		strings.TrimSpace(grade)

	if rawGrade == "" {
		return normalizedSubject,
			"",
			true
	}

	if concreteGrade, concreteOK :=
		utils.NormalizeGradeToStandardLabel(
			rawGrade,
		); concreteOK {
		return normalizedSubject,
			concreteGrade,
			true
	}

	if broadGrade, broadOK :=
		utils.NormalizeBroadLearningLevel(
			rawGrade,
		); broadOK {
		return normalizedSubject,
			broadGrade,
			true
	}

	return "", "", false
}

// strictRecipeMatchesEntity 校验自动配方课程与具体层级。
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

// ValidateAssistantForLesson 对自动匹配助手执行运行时严格校验。
func (s *AIAssistantService) ValidateAssistantForLesson(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
	subject string,
	grade string,
	scene string,
) (*models.AIAssistant, error) {
	assistant, err :=
		repository.GetAIAssistantByID(
			ctx,
			strings.TrimSpace(id),
		)

	if err != nil {
		return nil, err
	}

	if !assistant.IsActive {
		return nil,
			repository.ErrAIAssistantInactive
	}

	if !assistantResourceEducationDomainMatches(
		actor,
		assistant,
	) {
		return nil,
			ErrAssistantEducationDomainMismatch
	}

	if !s.canView(actor, assistant) {
		return nil,
			ErrAssistantPermDenied
	}

	if !strictAssistantMatchesEntity(
		assistant,
		subject,
		grade,
		scene,
	) {
		return nil,
			ErrAssistantLessonMismatch
	}

	return assistant, nil
}

// ValidateAssistantForManualLesson 对老师手动选择助手执行校验。
func (s *AIAssistantService) ValidateAssistantForManualLesson(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
	subject string,
	scene string,
) (*models.AIAssistant, error) {
	assistant, err :=
		repository.GetAIAssistantByID(
			ctx,
			strings.TrimSpace(id),
		)

	if err != nil {
		return nil, err
	}

	if !assistant.IsActive {
		return nil,
			repository.ErrAIAssistantInactive
	}

	if !assistantResourceEducationDomainMatches(
		actor,
		assistant,
	) {
		return nil,
			ErrAssistantEducationDomainMismatch
	}

	if !s.canView(actor, assistant) {
		return nil,
			ErrAssistantPermDenied
	}

	if !manualAssistantMatchesEntity(
		assistant,
		subject,
		scene,
	) {
		return nil,
			ErrAssistantManualMismatch
	}

	return assistant, nil
}

// LoadActiveAssistantForLessonUse 加载自动匹配助手并递增使用次数。
func (s *AIAssistantService) LoadActiveAssistantForLessonUse(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
	subject string,
	grade string,
	scene string,
) (*models.AIAssistant, error) {
	assistant, err :=
		s.ValidateAssistantForLesson(
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

	incrementAssistantUseCount(
		assistant.ID,
	)

	return assistant, nil
}

// LoadActiveAssistantForManualLessonUse 加载手动选择助手。
func (s *AIAssistantService) LoadActiveAssistantForManualLessonUse(
	ctx context.Context,
	actor *AssistantActorContext,
	id string,
	subject string,
	scene string,
) (*models.AIAssistant, error) {
	assistant, err :=
		s.ValidateAssistantForManualLesson(
			ctx,
			actor,
			id,
			subject,
			scene,
		)

	if err != nil {
		return nil, err
	}

	incrementAssistantUseCount(
		assistant.ID,
	)

	return assistant, nil
}

// incrementAssistantUseCount 异步记录助手真实使用次数。
func incrementAssistantUseCount(
	assistantID string,
) {
	go func(id string) {
		_ = repository.IncrementAIAssistantUseCount(
			context.Background(),
			id,
		)
	}(assistantID)
}

// loadRecipeForLesson 加载自动匹配配方并执行严格校验。
func loadRecipeForLesson(
	ctx context.Context,
	recipeID string,
	subject string,
	grade string,
	currentEducationDomain string,
) (*models.TeachingRecipe, error) {
	recipe, err :=
		repository.GetRecipeByID(
			ctx,
			strings.TrimSpace(recipeID),
		)

	if err != nil {
		return nil, err
	}

	if recipe.Status !=
		recipeAutoMountActiveStatus ||
		!strictRecipeMatchesEntity(
			recipe,
			subject,
			grade,
		) {
		return nil,
			ErrRecipeLessonMismatch
	}

	if !models.ResourceEducationDomainMatches(
		recipe.EducationDomain,
		currentEducationDomain,
	) {
		return nil,
			ErrRecipeLessonMismatch
	}

	return recipe, nil
}

// loadRecipeForPlanUse 根据教案记录的配方选择方式加载本轮配方。
func loadRecipeForPlanUse(
	ctx context.Context,
	lp *models.LessonPlan,
) (
	*models.TeachingRecipe,
	models.RecipeSelectionMode,
	error,
) {
	if lp == nil {
		return nil,
			models.RecipeSelectionModeAuto,
			ErrRecipeLessonMismatch
	}

	selectionMode :=
		resolveRecipeSelectionModeForReceipt(
			ctx,
			lp,
		)

	if selectionMode ==
		models.RecipeSelectionModeNone {
		return nil, selectionMode, nil
	}

	if lp.RecipeID == nil ||
		strings.TrimSpace(
			*lp.RecipeID,
		) == "" {
		return nil, selectionMode, nil
	}

	recipeID :=
		strings.TrimSpace(
			*lp.RecipeID,
		)

	planEducationDomain :=
		strings.ToLower(
			strings.TrimSpace(
				lp.EducationDomain,
			),
		)

	if !models.IsTeachingEducationDomain(
		planEducationDomain,
	) {
		if selectionMode ==
			models.RecipeSelectionModeSelected {
			return nil,
				selectionMode,
				ErrRecipeManualSelectionDenied
		}

		return nil,
			selectionMode,
			ErrRecipeLessonMismatch
	}

	if selectionMode ==
		models.RecipeSelectionModeSelected {
		recipe, err :=
			loadRecipeForManualSelectionForDomain(
				ctx,
				recipeID,
				lp.AuthorID,
				planEducationDomain,
			)

		if err != nil {
			return nil,
				selectionMode,
				err
		}

		if recipe == nil ||
			!models.ResourceEducationDomainMatches(
				recipe.EducationDomain,
				planEducationDomain,
			) {
			return nil,
				selectionMode,
				ErrRecipeManualSelectionDenied
		}

		return recipe,
			selectionMode,
			nil
	}

	recipe, err :=
		loadRecipeForLesson(
			ctx,
			recipeID,
			lp.Subject,
			lp.Grade,
			planEducationDomain,
		)

	if err != nil {
		return nil,
			selectionMode,
			err
	}

	if recipe == nil ||
		!models.ResourceEducationDomainMatches(
			recipe.EducationDomain,
			planEducationDomain,
		) {
		return nil,
			selectionMode,
			ErrRecipeLessonMismatch
	}

	return recipe,
		selectionMode,
		nil
}

// loadRecipeForManualSelection 加载新建或导入教案时老师明确选择的配方。
func loadRecipeForManualSelection(
	ctx context.Context,
	recipeID string,
	authorID string,
) (*models.TeachingRecipe, error) {
	recipeID =
		strings.TrimSpace(recipeID)

	authorID =
		strings.TrimSpace(authorID)

	if recipeID == "" ||
		authorID == "" {
		return nil,
			ErrRecipeManualSelectionDenied
	}

	author, err :=
		repository.FindUserByID(
			ctx,
			authorID,
		)

	if err != nil {
		return nil, err
	}

	if author == nil {
		return nil,
			ErrRecipeManualSelectionDenied
	}

	actor :=
		BuildActorFromClaims(
			ctx,
			authorID,
			author.Role,
		)

	if actor == nil ||
		!models.IsTeachingEducationDomain(
			actor.EducationDomain,
		) {
		return nil,
			ErrRecipeManualSelectionDenied
	}

	return loadRecipeForManualSelectionWithActor(
		ctx,
		recipeID,
		actor,
	)
}

// loadRecipeForManualSelectionForDomain 加载已有教案selected模式的配方。
func loadRecipeForManualSelectionForDomain(
	ctx context.Context,
	recipeID string,
	authorID string,
	currentEducationDomain string,
) (*models.TeachingRecipe, error) {
	recipeID =
		strings.TrimSpace(recipeID)

	authorID =
		strings.TrimSpace(authorID)

	if recipeID == "" ||
		authorID == "" {
		return nil,
			ErrRecipeManualSelectionDenied
	}

	normalizedDomain :=
		strings.ToLower(
			strings.TrimSpace(
				currentEducationDomain,
			),
		)

	if !models.IsTeachingEducationDomain(
		normalizedDomain,
	) {
		return nil,
			ErrRecipeManualSelectionDenied
	}

	author, err :=
		repository.FindUserByID(
			ctx,
			authorID,
		)

	if err != nil {
		return nil, err
	}

	if author == nil {
		return nil,
			ErrRecipeManualSelectionDenied
	}

	actor :=
		BuildActorFromClaims(
			ctx,
			authorID,
			author.Role,
		)

	if actor == nil {
		return nil,
			ErrRecipeManualSelectionDenied
	}

	actor.EducationDomain =
		normalizedDomain

	return loadRecipeForManualSelectionWithActor(
		ctx,
		recipeID,
		actor,
	)
}

// loadRecipeForManualSelectionWithActor 使用统一Actor执行手动配方终校验。
func loadRecipeForManualSelectionWithActor(
	ctx context.Context,
	recipeID string,
	actor *AssistantActorContext,
) (*models.TeachingRecipe, error) {
	recipeID =
		strings.TrimSpace(recipeID)

	if recipeID == "" ||
		actor == nil ||
		strings.TrimSpace(
			actor.UserID,
		) == "" ||
		!models.IsTeachingEducationDomain(
			actor.EducationDomain,
		) {
		return nil,
			ErrRecipeManualSelectionDenied
	}

	recipe, allowed, err :=
		repository.GetVisibleActiveRecipeByIDForDomain(
			ctx,
			actor.UserID,
			actor.MyGroupIDs,
			actor.SchoolID,
			actor.EducationDomain,
			recipeID,
		)

	if err != nil {
		return nil, err
	}

	if !allowed ||
		recipe == nil {
		return nil,
			ErrRecipeManualSelectionDenied
	}

	if !models.ResourceEducationDomainMatches(
		recipe.EducationDomain,
		actor.EducationDomain,
	) {
		return nil,
			ErrRecipeManualSelectionDenied
	}

	return recipe, nil
}
