package services

// recipe_market_service.go — 配方市场Actor接入
//
// 本文件把统一AssistantActorContext转换为市场可信查询参数，
// 不修改大型recipe_service.go。

import (
	"context"

	"tedna/internal/repository"
)

// ListMarketRecipesForActor 查询当前Actor可见的市场排行榜。
func (s *RecipeService) ListMarketRecipesForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	subject string,
	gradeRange string,
	sortBy string,
	limit int,
	offset int,
) (*MarketRecipesResponse, error) {
	if actor == nil ||
		actor.UserID == "" {
		return nil, ErrRecipeUnauthorized
	}

	items, total, err :=
		repository.
			ListMarketRecipesForActorDomain(
				ctx,
				&repository.RecipeMarketDomainParams{
					CurrentRole: actor.Role,

					CurrentSchoolID: actor.SchoolID,
					CurrentEducationDomain: actor.
						EducationDomain,
					CurrentGroupIDs: actor.MyGroupIDs,

					Subject:    subject,
					GradeRange: gradeRange,
					SortBy:     sortBy,

					Limit:  limit,
					Offset: offset,
				},
			)
	if err != nil {
		return nil, err
	}

	return &MarketRecipesResponse{
		Recipes: items,
		Total:   total,
	}, nil
}
