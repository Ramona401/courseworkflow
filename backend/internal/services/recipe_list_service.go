package services

// recipe_list_service.go — 配方普通列表Actor接入
//
// 本文件负责把统一AssistantActorContext转为列表仓储参数。
// 不修改大型recipe_service.go，也不复用前端scope_ref_id决定权限。

import (
	"context"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ListRecipesForActor 查询当前Actor可信范围内的配方列表。
//
// scope、scopeRefID、subject和gradeRange均只作为附加筛选，
// 实际可见范围由Actor的用户、学校、教研组和教育域决定。
func (s *RecipeService) ListRecipesForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	scope string,
	scopeRefID string,
	subject string,
	gradeRange string,
	limit int,
	offset int,
) (*models.RecipeListResponse, error) {
	if actor == nil ||
		actor.UserID == "" {
		return nil, ErrRecipeUnauthorized
	}

	items, total, err :=
		repository.ListRecipesForActorDomain(
			ctx,
			&repository.RecipeDomainListParams{
				CurrentUserID: actor.UserID,
				CurrentRole:   actor.Role,

				CurrentSchoolID: actor.SchoolID,
				CurrentEducationDomain: actor.
					EducationDomain,
				CurrentGroupIDs: actor.MyGroupIDs,

				RequestedScope:      scope,
				RequestedScopeRefID: scopeRefID,
				Subject:             subject,
				GradeRange:          gradeRange,

				Limit:  limit,
				Offset: offset,
			},
		)
	if err != nil {
		return nil, err
	}

	return &models.RecipeListResponse{
		Recipes: items,
		Total:   total,
	}, nil
}
