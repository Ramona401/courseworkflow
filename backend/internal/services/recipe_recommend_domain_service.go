package services

// recipe_recommend_domain_service.go — 配方组件推荐教育域业务层。
//
// 本文件只处理创建或编辑配方时的组件推荐：
//   - 普通Actor始终使用服务端可信教学域，忽略客户端伪造域；
//   - mixed管理员必须在请求中显式指定k12、vocational或adult；
//   - common只能作为被推荐资源域，不能作为本次推荐的当前教学域；
//   - 普通推荐和画像推荐使用完全相同的教育域候选集；
//   - 画像标签只影响候选排序，绝不能扩大教育域范围。

import (
	"context"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
	"tedna/internal/utils"
)

// buildRecipeRecommendationMatchRequest 构建可信域推荐请求。
//
// limit由调用方决定：普通推荐每类3条，画像推荐每类5条。
func buildRecipeRecommendationMatchRequest(
	actor *AssistantActorContext,
	request *models.RecipeRecommendRequest,
	limit int,
) (
	*models.MatchComponentsRequest,
	string,
	error,
) {
	if request == nil {
		return nil, "",
			ErrComponentMatchRequestRequired
	}

	subject := strings.TrimSpace(
		request.Subject,
	)
	if subject == "" {
		return nil, "",
			ErrRecipeSubjectRequired
	}

	currentDomain, err :=
		resolveComponentMatchDomain(
			actor,
			request.EducationDomain,
		)
	if err != nil {
		return nil, "", err
	}

	if limit <= 0 {
		limit = 3
	}

	matchRequest := &models.MatchComponentsRequest{
		EducationDomain: currentDomain,
		Subject:         subject,
		GradeRange: utils.NormalizeGradeToNumber(
			strings.TrimSpace(
				request.GradeRange,
			),
		),
		Limit: limit,
	}

	return matchRequest,
		currentDomain,
		nil
}

// RecommendComponentsForActor 按可信Actor域普通推荐配方组件。
func (s *RecipeService) RecommendComponentsForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	request *models.RecipeRecommendRequest,
) ([]*models.MatchedComponentGroup, error) {
	matchRequest,
		currentDomain,
		err :=
		buildRecipeRecommendationMatchRequest(
			actor,
			request,
			3,
		)
	if err != nil {
		return nil, err
	}

	groups, err :=
		repository.MatchComponentsForEducationDomain(
			ctx,
			matchRequest,
			currentDomain,
		)
	if err != nil {
		return nil, err
	}

	if groups == nil {
		groups =
			[]*models.MatchedComponentGroup{}
	}

	return groups, nil
}

// SmartRecommendComponentsForActor 按可信Actor域画像加权推荐配方组件。
func (s *RecipeService) SmartRecommendComponentsForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	request *models.RecipeRecommendRequest,
	profile *models.TeachingProfile,
) ([]*models.MatchedComponentGroup, error) {
	matchRequest,
		currentDomain,
		err :=
		buildRecipeRecommendationMatchRequest(
			actor,
			request,
			5,
		)
	if err != nil {
		return nil, err
	}

	groups, err :=
		repository.
			SmartMatchComponentsForEducationDomain(
				ctx,
				matchRequest,
				currentDomain,
				buildProfileTags(profile),
			)
	if err != nil {
		return nil, err
	}

	if groups == nil {
		groups =
			[]*models.MatchedComponentGroup{}
	}

	return groups, nil
}
