package services

// recipe_stage_access_service.go — 配方自定义阶段访问控制
//
// 本文件不修改原有workshop_stage_components.go，只在HTTP入口和
// 既有自定义阶段CRUD之间增加统一配方权限校验。
//
// 列表读取：
//   - 配方必须active；
//   - 当前Actor必须满足配方教育域和组织可见性。
//
// 创建、更新、删除：
//   - 配方必须active；
//   - 当前Actor必须是配方作者；
//   - 配方教育域必须与Actor当前教育域兼容。
//
// admin不会因为管理身份自动取得他人配方的阶段修改权，
// 与配方正文更新、删除和共享保持相同的作者专属规则。

import (
	"context"

	"tedna/internal/models"
)

// ListCustomStagesForActor 列出当前Actor有权查看的配方自定义阶段。
func (s *WorkshopStageService) ListCustomStagesForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
) ([]*models.CustomStageResponse, error) {
	if _, err := loadVisibleRecipeForActor(
		ctx,
		actor,
		recipeID,
	); err != nil {
		return nil, err
	}

	return s.ListCustomStages(
		ctx,
		recipeID,
	)
}

// CreateCustomStageForActor 为作者本人且教育域兼容的配方创建阶段。
func (s *WorkshopStageService) CreateCustomStageForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
	req *models.CreateCustomStageRequest,
) (*models.CustomStageResponse, error) {
	if _, err := loadManagedRecipeForActor(
		ctx,
		actor,
		recipeID,
	); err != nil {
		return nil, err
	}

	return s.CreateCustomStage(
		ctx,
		recipeID,
		req,
		actor.UserID,
	)
}

// UpdateCustomStageForActor 更新作者本人且教育域兼容的配方阶段。
func (s *WorkshopStageService) UpdateCustomStageForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
	stageCode string,
	req *models.UpdateCustomStageRequest,
) error {
	if _, err := loadManagedRecipeForActor(
		ctx,
		actor,
		recipeID,
	); err != nil {
		return err
	}

	return s.UpdateCustomStage(
		ctx,
		recipeID,
		stageCode,
		req,
		actor.UserID,
	)
}

// DeleteCustomStageForActor 删除作者本人且教育域兼容的配方阶段。
func (s *WorkshopStageService) DeleteCustomStageForActor(
	ctx context.Context,
	actor *AssistantActorContext,
	recipeID string,
	stageCode string,
) error {
	if _, err := loadManagedRecipeForActor(
		ctx,
		actor,
		recipeID,
	); err != nil {
		return err
	}

	return s.DeleteCustomStage(
		ctx,
		recipeID,
		stageCode,
		actor.UserID,
	)
}
