package services

// recipe_access_service.go — 配方按ID访问控制
//
// 本文件不重写既有RecipeService业务逻辑，只在HTTP入口与原方法之间
// 增加统一的Actor、组织范围、教育域和共享目标权限校验。
//
// 教育域规则：
//   - k12只能访问k12或common配方；
//   - vocational只能访问vocational或common配方；
//   - adult只能访问adult或common配方；
//   - mixed仅供admin跨域管理合法资源；
//   - 空值、非法值和current=common严格拒绝。
//
// 组织可见性规则：
//   - personal：作者本人；
//   - group：目标教研组成员；
//   - school：当前学校成员；
//   - admin：可跨组织查看合法教育域资源。
//
// 写权限规则：
//   - 更新、学情更新、删除和共享保持作者本人专属；
//   - Fork要求来源配方可见、active且教育域兼容；
//   - 效果统计仅作者本人或admin可读。

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// loadActiveRecipeForActor 读取active配方并统一翻译NotFound错误。
func loadActiveRecipeForActor(
	ctx context.Context,
	recipeID string,
) (*models.TeachingRecipe, error) {
	recipeID = strings.TrimSpace(recipeID)
	if recipeID == "" {
		return nil, ErrRecipeNotFound
	}

	recipe, err := repository.GetRecipeByID(
		ctx,
		recipeID,
	)
	if err != nil {
		if errors.Is(
			err,
			repository.ErrRecipeNotFound,
		) {
			return nil, ErrRecipeNotFound
		}
		return nil, err
	}

	if recipe == nil ||
		recipe.Status != recipeAutoMountActiveStatus {
		return nil, ErrRecipeNotFound
	}

	return recipe, nil
}

// recipeActorDomainMatches 校验配方资源域与Actor当前域。
func recipeActorDomainMatches(
	actor *AssistantActorContext,
	recipe *models.TeachingRecipe,
) bool {
	if actor == nil || recipe == nil {
		return false
	}

	return models.ResourceEducationDomainMatches(
		recipe.EducationDomain,
		actor.EducationDomain,
	)
}

// recipeTargetID 安全读取配方scope_ref_id。
func recipeTargetID(
	recipe *models.TeachingRecipe,
) string {
	if recipe == nil ||
		recipe.ScopeRefID == nil {
		return ""
	}

	return strings.TrimSpace(
		*recipe.ScopeRefID,
	)
}

// recipeActorCanView 判断Actor是否同时满足教育域与组织可见性。
func recipeActorCanView(
	actor *AssistantActorContext,
	recipe *models.TeachingRecipe,
) bool {
	if !recipeActorDomainMatches(
		actor,
		recipe,
	) {
		return false
	}

	if actor.Role == models.RoleAdmin {
		return true
	}

	switch recipe.Scope {
	case models.RecipeScopePersonal:
		return recipe.AuthorID == actor.UserID

	case models.RecipeScopeGroup:
		targetID := recipeTargetID(recipe)
		return targetID != "" &&
			containsStr(
				actor.MyGroupIDs,
				targetID,
			)

	case models.RecipeScopeSchool:
		targetID := recipeTargetID(recipe)
		return targetID != "" &&
			strings.TrimSpace(actor.SchoolID) != "" &&
			targetID == strings.TrimSpace(
				actor.SchoolID,
			)
	}

	return false
}

// recipeActorCanManage 判断Actor是否为配方作者且教育域兼容。
//
// admin不会因为管理身份自动获得他人配方的修改、删除或共享权。
func recipeActorCanManage(
	actor *AssistantActorContext,
	recipe *models.TeachingRecipe,
) bool {
	if !recipeActorDomainMatches(
		actor,
		recipe,
	) {
		return false
	}

	return actor != nil &&
		recipe != nil &&
		recipe.AuthorID == actor.UserID
}

// recipeActorCanViewStats 校验包含用户和教案ID的统计明细权限。
func recipeActorCanViewStats(
	actor *AssistantActorContext,
	recipe *models.TeachingRecipe,
) bool {
	if !recipeActorDomainMatches(
		actor,
		recipe,
	) {
		return false
	}

	return actor.Role == models.RoleAdmin ||
		recipe.AuthorID == actor.UserID
}

// loadVisibleRecipeForActor 读取当前Actor可见的active配方。
func loadVisibleRecipeForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
) (*models.TeachingRecipe, error) {
	recipe, err := loadActiveRecipeForActor(
		ctx,
		recipeID,
	)
	if err != nil {
		return nil, err
	}

	if !recipeActorCanView(
		actor,
		recipe,
	) {
		return nil, ErrRecipeUnauthorized
	}

	return recipe, nil
}

// loadManagedRecipeForActor 读取作者本人可管理的active配方。
func loadManagedRecipeForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
) (*models.TeachingRecipe, error) {
	recipe, err := loadActiveRecipeForActor(
		ctx,
		recipeID,
	)
	if err != nil {
		return nil, err
	}

	if !recipeActorCanManage(
		actor,
		recipe,
	) {
		return nil, ErrRecipeUnauthorized
	}

	return recipe, nil
}

// validateRecipeTargetSchool 校验共享目标必须是active教学学校，
// 且目标学校教育域能够使用当前配方资源域。
func validateRecipeTargetSchool(
	recipe *models.TeachingRecipe,
	target *models.OrganizationEducationDomainItem,
) error {
	if recipe == nil ||
		target == nil ||
		strings.TrimSpace(target.Type) != "school" ||
		strings.TrimSpace(target.Status) != "active" {
		return fmt.Errorf(
			"%w: 共享目标必须是启用状态的学校",
			ErrRecipeShareInvalid,
		)
	}

	targetDomain := strings.ToLower(
		strings.TrimSpace(
			target.EducationDomain,
		),
	)
	if !models.IsTeachingEducationDomain(
		targetDomain,
	) {
		return fmt.Errorf(
			"%w: 共享目标学校没有确定的教学教育域",
			ErrRecipeShareInvalid,
		)
	}

	if !models.ResourceEducationDomainMatches(
		recipe.EducationDomain,
		targetDomain,
	) {
		return fmt.Errorf(
			"%w: 配方教育域与目标学校不匹配",
			ErrRecipeShareInvalid,
		)
	}

	return nil
}

// validateRecipeShareTargetForActor 校验共享目标和发布权限。
func validateRecipeShareTargetForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipe *models.TeachingRecipe,
	scope string,
	targetID string,
) error {
	if actor == nil || recipe == nil {
		return ErrRecipeUnauthorized
	}

	scope = strings.TrimSpace(scope)
	targetID = strings.TrimSpace(targetID)

	if targetID == "" {
		return fmt.Errorf(
			"%w: 共享目标ID不能为空",
			ErrRecipeShareInvalid,
		)
	}

	switch scope {
	case models.RecipeScopeGroup:
		// 非admin必须是目标组lead或backbone。
		if actor.Role != models.RoleAdmin &&
			!containsStr(
				actor.MyLeadOrBackboneGroupIDs,
				targetID,
			) {
			return ErrRecipeUnauthorized
		}

		group, err :=
			repository.GetTeachingGroupByID(
				ctx,
				targetID,
			)
		if err != nil || group == nil {
			return fmt.Errorf(
				"%w: 共享目标教研组不存在",
				ErrRecipeShareInvalid,
			)
		}

		schoolID := strings.TrimSpace(
			group.SchoolID,
		)
		if schoolID == "" {
			return fmt.Errorf(
				"%w: 目标教研组没有绑定学校",
				ErrRecipeShareInvalid,
			)
		}

		targetSchool, err :=
			repository.
				GetOrganizationEducationDomainByID(
					ctx,
					schoolID,
				)
		if err != nil {
			return fmt.Errorf(
				"%w: 无法确认目标教研组所属学校",
				ErrRecipeShareInvalid,
			)
		}

		return validateRecipeTargetSchool(
			recipe,
			targetSchool,
		)

	case models.RecipeScopeSchool:
		// 非admin必须是目标学校本校senior_operator。
		if actor.Role != models.RoleAdmin {
			if actor.Role !=
				models.RoleSeniorOperator ||
				strings.TrimSpace(
					actor.SchoolID,
				) == "" ||
				strings.TrimSpace(
					actor.SchoolID,
				) != targetID {
				return ErrRecipeUnauthorized
			}
		}

		targetSchool, err :=
			repository.
				GetOrganizationEducationDomainByID(
					ctx,
					targetID,
				)
		if err != nil {
			return fmt.Errorf(
				"%w: 共享目标学校不存在",
				ErrRecipeShareInvalid,
			)
		}

		return validateRecipeTargetSchool(
			recipe,
			targetSchool,
		)

	default:
		return fmt.Errorf(
			"%w: 共享范围只能是group或school",
			ErrRecipeShareInvalid,
		)
	}
}

// GetRecipeForActor 获取当前Actor有权查看的配方详情。
func (s *RecipeService) GetRecipeForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
) (*models.RecipeDetailResponse, error) {
	recipe, err := loadVisibleRecipeForActor(
		ctx,
		actor,
		recipeID,
	)
	if err != nil {
		return nil, err
	}

	return buildRecipeDetailForResourceDomain(
		ctx,
		recipe,
	)
}

// UpdateRecipeForActor 更新作者本人且教育域兼容的配方。
func (s *RecipeService) UpdateRecipeForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
	req *models.UpdateRecipeRequest,
) error {
	if req == nil {
		return ErrRecipeComponentConfigInvalid
	}

	recipe, err := loadManagedRecipeForActor(
		ctx,
		actor,
		recipeID,
	)
	if err != nil {
		return err
	}

	validatedComponentIDs, err :=
		ValidateRecipeComponentIDsForWrite(
			ctx,
			req.ComponentIDs,
			recipe.EducationDomain,
		)
	if err != nil {
		return err
	}

	validatedRequest := *req
	validatedRequest.ComponentIDs =
		validatedComponentIDs

	return s.UpdateRecipe(
		ctx,
		recipeID,
		&validatedRequest,
		actor.UserID,
	)
}

// UpdateStudentProfileForActor 更新作者本人配方的学情记录。
func (s *RecipeService) UpdateStudentProfileForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
	profile string,
) error {
	if _, err := loadManagedRecipeForActor(
		ctx,
		actor,
		recipeID,
	); err != nil {
		return err
	}

	return s.UpdateStudentProfile(
		ctx,
		recipeID,
		profile,
		actor.UserID,
	)
}

// DeleteRecipeForActor 删除作者本人且教育域兼容的配方。
func (s *RecipeService) DeleteRecipeForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
) error {
	if _, err := loadManagedRecipeForActor(
		ctx,
		actor,
		recipeID,
	); err != nil {
		return err
	}

	return s.DeleteRecipe(
		ctx,
		recipeID,
		actor.UserID,
	)
}

// ForkRecipeForActor Fork当前Actor可见的配方。
func (s *RecipeService) ForkRecipeForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
) (*models.TeachingRecipe, error) {
	if _, err := loadVisibleRecipeForActor(
		ctx,
		actor,
		recipeID,
	); err != nil {
		return nil, err
	}

	return s.ForkRecipe(
		ctx,
		recipeID,
		actor.UserID,
	)
}

// ShareRecipeForActor 将作者本人的配方共享到合法目标。
func (s *RecipeService) ShareRecipeForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
	req *models.ShareRecipeRequest,
) error {
	if req == nil {
		return ErrRecipeShareInvalid
	}

	recipe, err := loadManagedRecipeForActor(
		ctx,
		actor,
		recipeID,
	)
	if err != nil {
		return err
	}

	if err := validateRecipeShareTargetForActor(
		ctx,
		actor,
		recipe,
		req.Scope,
		req.ScopeRefID,
	); err != nil {
		return err
	}

	return s.ShareRecipe(
		ctx,
		recipeID,
		req,
		actor.UserID,
	)
}

// PreviewContextForActor 预览当前Actor有权查看的配方上下文。
func (s *RecipeService) PreviewContextForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
) (*models.RecipeContextPreview, error) {
	recipe, err := loadVisibleRecipeForActor(
		ctx,
		actor,
		recipeID,
	)
	if err != nil {
		return nil, err
	}

	return buildRecipePreviewForResourceDomain(
		ctx,
		recipe,
	)
}

// GetRecipeStatsForActor 获取作者本人或admin有权查看的统计。
func (s *RecipeService) GetRecipeStatsForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
) (*RecipeStatsResponse, error) {
	recipe, err := loadActiveRecipeForActor(
		ctx,
		recipeID,
	)
	if err != nil {
		return nil, err
	}

	if !recipeActorCanViewStats(
		actor,
		recipe,
	) {
		return nil, ErrRecipeUnauthorized
	}

	return s.GetRecipeStats(
		ctx,
		recipeID,
	)
}
