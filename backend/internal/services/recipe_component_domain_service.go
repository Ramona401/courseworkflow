package services

// recipe_component_domain_service.go — 配方绑定组件教育域业务层。
//
// 本文件统一处理四类行为：
//   - 创建配方前严格校验前端直接提交的组件ID；
//   - 更新配方前严格校验新的组件ID集合；
//   - 配方详情只展示配方资源域允许看到的组件摘要；
//   - 配方上下文预览只注入配方资源域允许使用的有效组件正文。
//
// 写入与历史读取的策略不同：
//   - 新提交ID任一不存在、失效、未审核或异域，整组拒绝；
//   - 历史component_ids中的失效或异域ID静默过滤，不泄漏其元数据。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	// ErrRecipeComponentConfigInvalid 表示数据库中的配方组件JSON损坏，
	// 或调用方传入了非法配方资源域。
	ErrRecipeComponentConfigInvalid = errors.New("配方组件配置无效")
)

// normalizeRecipeComponentResourceDomain 校验配方资源域。
func normalizeRecipeComponentResourceDomain(
	rawDomain string,
) (string, error) {
	domain := strings.ToLower(
		strings.TrimSpace(rawDomain),
	)

	if !models.IsResourceEducationDomain(
		domain,
	) {
		return "",
			ErrRecipeComponentConfigInvalid
	}

	return domain, nil
}

// recipeComponentDomainMatches 判断组件是否可以绑定到该资源域配方。
//
// common配方必须保持完全公共，只能绑定common组件；
// 具体教学域配方可以绑定同域组件或common组件。
func recipeComponentDomainMatches(
	componentDomain string,
	recipeDomain string,
) bool {
	componentDomain = strings.ToLower(
		strings.TrimSpace(componentDomain),
	)
	recipeDomain = strings.ToLower(
		strings.TrimSpace(recipeDomain),
	)

	if recipeDomain ==
		models.EducationDomainCommon {
		return componentDomain ==
			models.EducationDomainCommon
	}

	if !models.IsTeachingEducationDomain(
		recipeDomain,
	) {
		return false
	}

	return models.ResourceEducationDomainMatches(
		componentDomain,
		recipeDomain,
	)
}

// validateRecipeComponentAccessRecords 对直接ID数据库快照执行整组校验。
func validateRecipeComponentAccessRecords(
	componentIDs []string,
	records []*repository.ComponentAccessRecord,
	recipeDomain string,
) error {
	if _, err :=
		normalizeRecipeComponentResourceDomain(
			recipeDomain,
		); err != nil {
		return err
	}

	normalizedIDs :=
		NormalizeUniqueComponentIDs(
			componentIDs,
		)

	if len(normalizedIDs) == 0 {
		return nil
	}

	recordMap := make(
		map[string]*repository.ComponentAccessRecord,
		len(records),
	)

	for _, record := range records {
		if record == nil {
			continue
		}

		recordMap[record.ID] = record
	}

	for _, componentID := range normalizedIDs {
		record, exists := recordMap[componentID]

		if !exists {
			return ErrComponentSelectionInvalid
		}

		if record.Status != "active" ||
			record.ReviewStatus !=
				models.ComponentReviewApproved {
			return ErrComponentSelectionInvalid
		}

		if !recipeComponentDomainMatches(
			record.EducationDomain,
			recipeDomain,
		) {
			return ErrComponentSelectionInvalid
		}
	}

	return nil
}

// ValidateRecipeComponentIDsForWrite 严格验证新提交的配方组件ID。
func ValidateRecipeComponentIDsForWrite(
	ctx context.Context,
	componentIDs []string,
	recipeDomain string,
) ([]string, error) {
	recipeDomain, err :=
		normalizeRecipeComponentResourceDomain(
			recipeDomain,
		)
	if err != nil {
		return nil, err
	}

	normalizedIDs :=
		NormalizeUniqueComponentIDs(
			componentIDs,
		)

	if len(normalizedIDs) == 0 {
		return []string{}, nil
	}

	records, err :=
		repository.GetComponentAccessRecordsByIDs(
			ctx,
			normalizedIDs,
		)
	if err != nil {
		return nil, err
	}

	if err := validateRecipeComponentAccessRecords(
		normalizedIDs,
		records,
		recipeDomain,
	); err != nil {
		return nil, err
	}

	return normalizedIDs, nil
}

// parseRecipeComponentIDs 解析历史配方组件ID。
func parseRecipeComponentIDs(
	recipe *models.TeachingRecipe,
) ([]string, error) {
	if recipe == nil {
		return nil,
			ErrRecipeComponentConfigInvalid
	}

	raw := strings.TrimSpace(
		recipe.ComponentIDs,
	)

	if raw == "" || raw == "[]" {
		return []string{}, nil
	}

	var componentIDs []string

	if err := json.Unmarshal(
		[]byte(raw),
		&componentIDs,
	); err != nil {
		return nil,
			fmt.Errorf(
				"%w: %v",
				ErrRecipeComponentConfigInvalid,
				err,
			)
	}

	return NormalizeUniqueComponentIDs(
		componentIDs,
	), nil
}

// buildRecipeDetailForResourceDomain 构建域过滤后的配方详情。
func buildRecipeDetailForResourceDomain(
	ctx context.Context,
	recipe *models.TeachingRecipe,
) (*models.RecipeDetailResponse, error) {
	if recipe == nil {
		return nil, ErrRecipeNotFound
	}

	resourceDomain, err :=
		normalizeRecipeComponentResourceDomain(
			recipe.EducationDomain,
		)
	if err != nil {
		return nil, err
	}

	componentIDs, err :=
		parseRecipeComponentIDs(recipe)
	if err != nil {
		return nil, err
	}

	components, err :=
		repository.
			GetRecipeComponentBriefsForResourceDomain(
				ctx,
				componentIDs,
				resourceDomain,
			)
	if err != nil {
		return nil, err
	}

	if components == nil {
		components =
			[]*models.RecipeComponentBrief{}
	}

	authorName := ""

	if user, userErr :=
		repository.FindUserByID(
			ctx,
			recipe.AuthorID,
		); userErr == nil &&
		user != nil {
		authorName = user.DisplayName
	}

	return &models.RecipeDetailResponse{
		TeachingRecipe: *recipe,
		ComponentCount: len(components),
		Components:     components,
		AuthorName:     authorName,
		ScopeName:      models.RecipeScopeNameMap[recipe.Scope],
	}, nil
}

// buildRecipeContextForResourceDomain 构建域过滤后的配方AI上下文。
func buildRecipeContextForResourceDomain(
	ctx context.Context,
	recipe *models.TeachingRecipe,
) (string, error) {
	if recipe == nil {
		return "", ErrRecipeNotFound
	}

	resourceDomain, err :=
		normalizeRecipeComponentResourceDomain(
			recipe.EducationDomain,
		)
	if err != nil {
		return "", err
	}

	componentIDs, err :=
		parseRecipeComponentIDs(recipe)
	if err != nil {
		return "", err
	}

	groups, err :=
		repository.
			GetRecipeComponentContentsForResourceDomain(
				ctx,
				componentIDs,
				resourceDomain,
			)
	if err != nil {
		return "", err
	}

	var builder strings.Builder

	builder.WriteString(
		fmt.Sprintf(
			"【备课配方：%s v%d】\n",
			recipe.Name,
			recipe.Version,
		),
	)

	for _, group := range groups {
		if group == nil {
			continue
		}

		builder.WriteString(
			fmt.Sprintf(
				"\n== %s ==\n",
				group.LibraryName,
			),
		)

		for _, component := range group.Components {
			if component == nil {
				continue
			}

			builder.WriteString(
				fmt.Sprintf(
					"▸ %s\n",
					component.DisplayLabel,
				),
			)

			if component.DesignLogic != "" {
				builder.WriteString(
					fmt.Sprintf(
						"  设计逻辑：%s\n",
						component.DesignLogic,
					),
				)
			}

			if component.FullGuide != "" {
				builder.WriteString(
					fmt.Sprintf(
						"  完整指引：%s\n",
						component.FullGuide,
					),
				)
			}
		}
	}

	if strings.TrimSpace(
		recipe.StudentProfile,
	) != "" {
		builder.WriteString(
			fmt.Sprintf(
				"\n== 学情档案 ==\n%s\n",
				recipe.StudentProfile,
			),
		)
	}

	if strings.TrimSpace(
		recipe.TeachingStyle,
	) != "" {
		builder.WriteString(
			fmt.Sprintf(
				"\n== 教师偏好 ==\n%s\n",
				recipe.TeachingStyle,
			),
		)
	}

	if strings.TrimSpace(
		recipe.SchoolRequirements,
	) != "" {
		builder.WriteString(
			fmt.Sprintf(
				"\n== 学校要求 ==\n%s\n",
				recipe.SchoolRequirements,
			),
		)
	}

	if strings.TrimSpace(
		recipe.CustomNotes,
	) != "" {
		builder.WriteString(
			fmt.Sprintf(
				"\n== 备课心得 ==\n%s\n",
				recipe.CustomNotes,
			),
		)
	}

	if strings.TrimSpace(
		recipe.CustomPrompt,
	) != "" {
		builder.WriteString(
			fmt.Sprintf(
				"\n== 自定义指令 ==\n%s\n",
				recipe.CustomPrompt,
			),
		)
	}

	return builder.String(), nil
}

// buildRecipePreviewForResourceDomain 构建域过滤后的上下文预览。
func buildRecipePreviewForResourceDomain(
	ctx context.Context,
	recipe *models.TeachingRecipe,
) (*models.RecipeContextPreview, error) {
	contextText, err :=
		buildRecipeContextForResourceDomain(
			ctx,
			recipe,
		)
	if err != nil {
		return nil, err
	}

	return &models.RecipeContextPreview{
		RecipeID:      recipe.ID,
		RecipeName:    recipe.Name,
		ContextText:   contextText,
		TokenEstimate: len(contextText) / 2,
	}, nil
}
