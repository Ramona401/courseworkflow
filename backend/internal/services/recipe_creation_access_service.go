package services

// recipe_creation_access_service.go — 配方创建教育域访问控制
//
// 数据库教育域触发器为了兼容旧二进制、管理员测试和运维修复，
// 会把mixed管理角色或无法解析具体教学组织的用户回退到K12。
//
// 当前应用层采用更严格规则：
//   - 教师创建个人配方前必须建立可信Actor；
//   - Actor教育域必须是k12、vocational或adult；
//   - mixed、common、空值和非法域均禁止创建具体教学资源；
//   - Fork不经过本文件，继续继承来源配方教育域快照。
//
// 本文件同时保护：
//   - 普通配方创建接口；
//   - 前测提交后的自动配方生成；
//   - 跳过前测后的默认配方生成；
//   - 根据已有画像手动重新生成配方。

import (
	"context"
	"strings"

	"tedna/internal/models"
)

// validateTeachingRecipeCreationActor 校验配方创建Actor。
func validateTeachingRecipeCreationActor(
	actor *AssistantActorContext,
) error {
	if actor == nil ||
		strings.TrimSpace(actor.UserID) == "" {
		return ErrRecipeUnauthorized
	}

	domain := strings.ToLower(
		strings.TrimSpace(
			actor.EducationDomain,
		),
	)

	if !models.IsTeachingEducationDomain(
		domain,
	) {
		return ErrRecipeUnauthorized
	}

	return nil
}

// CreateRecipeForActor 为具有具体教学域的Actor创建个人配方。
func (s *RecipeService) CreateRecipeForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	req *models.CreateRecipeRequest,
) (*models.TeachingRecipe, error) {
	if req == nil {
		return nil,
			ErrRecipeComponentConfigInvalid
	}

	if err := validateTeachingRecipeCreationActor(
		actor,
	); err != nil {
		return nil, err
	}

	validatedComponentIDs, err :=
		ValidateRecipeComponentIDsForWrite(
			ctx,
			req.ComponentIDs,
			actor.EducationDomain,
		)
	if err != nil {
		return nil, err
	}

	validatedRequest := *req
	validatedRequest.ComponentIDs =
		validatedComponentIDs

	return s.CreateRecipe(
		ctx,
		&validatedRequest,
		actor.UserID,
	)
}

// SubmitAssessmentForActor 提交前测并生成个人配方。
func (s *AssessmentService) SubmitAssessmentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	req *models.AssessmentSubmitRequest,
	conversationLog []models.AssessmentMessage,
) (*models.AssessmentSubmitResponse, error) {
	if err := validateTeachingRecipeCreationActor(
		actor,
	); err != nil {
		return nil, err
	}

	return s.SubmitAssessment(
		ctx,
		actor.UserID,
		req,
		conversationLog,
	)
}

// SkipAssessmentForActor 跳过前测并尝试生成默认配方。
func (s *AssessmentService) SkipAssessmentForActor(
	ctx context.Context,
	actor *AssistantActorContext,
) (*models.AssessmentSubmitResponse, error) {
	if err := validateTeachingRecipeCreationActor(
		actor,
	); err != nil {
		return nil, err
	}

	return s.SkipAssessment(
		ctx,
		actor.UserID,
	)
}

// AutoGenerateRecipeFromProfileForActor 根据画像重新生成个人配方。
func (s *AssessmentService) AutoGenerateRecipeFromProfileForActor(
	ctx context.Context,
	actor *AssistantActorContext,
) (*models.AutoRecipeResponse, error) {
	if err := validateTeachingRecipeCreationActor(
		actor,
	); err != nil {
		return nil, err
	}

	return s.AutoGenerateRecipeFromProfile(
		ctx,
		actor.UserID,
	)
}
