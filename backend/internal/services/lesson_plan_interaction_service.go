package services

// lesson_plan_interaction_service.go — 共享教案互动业务逻辑
//
// 点赞、收藏、互动统计和我的收藏均属于共享教案市场能力。
// 所有入口必须先通过统一共享可见性底座，不能通过直接提交教案ID绕过。
// 历史非共享互动记录不主动删除，但不会继续出现在安全收藏列表中。

import (
	"context"
	"errors"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

var (
	ErrInvalidInteractionType = errors.New(
		"互动类型无效，可选值：like / favorite",
	)
)

// LessonPlanInteractionService 复用LessonPlanService的统一可见性底座。
type LessonPlanInteractionService struct {
	lpService *LessonPlanService
}

var lpInterLog = logger.WithModule(
	"lesson_plan_interaction",
)

// NewLessonPlanInteractionService 创建互动服务。
func NewLessonPlanInteractionService() *LessonPlanInteractionService {
	return &LessonPlanInteractionService{
		lpService: &LessonPlanService{},
	}
}

// loadVisibleSharedPlan 在任何统计或写入前校验目标共享资源。
func (
	s *LessonPlanInteractionService,
) loadVisibleSharedPlan(
	ctx context.Context,
	userID string,
	planID string,
) error {
	if s == nil || s.lpService == nil {
		return errors.New("教案互动服务未初始化")
	}

	_, err := s.lpService.loadSharedLessonPlanForRead(
		ctx,
		planID,
		userID,
		nil,
	)
	return err
}

// ToggleInteraction 切换点赞或收藏。
func (
	s *LessonPlanInteractionService,
) ToggleInteraction(
	ctx context.Context,
	userID string,
	planID string,
	interactionType string,
) (*models.ToggleInteractionResponse, error) {
	if interactionType != models.InteractionTypeLike &&
		interactionType != models.InteractionTypeFavorite {
		return nil, ErrInvalidInteractionType
	}

	if err := s.loadVisibleSharedPlan(
		ctx,
		userID,
		planID,
	); err != nil {
		return nil, err
	}

	active, err := repository.ToggleInteraction(
		ctx,
		userID,
		planID,
		interactionType,
	)
	if err != nil {
		lpInterLog.Error(
			"切换互动状态失败",
			"user_id", userID,
			"plan_id", planID,
			"type", interactionType,
			"error", err,
		)
		return nil, err
	}

	newCount, err := repository.GetInteractionCount(
		ctx,
		planID,
		interactionType,
	)
	if err != nil {
		// 互动写入已完成，计数查询属于响应增强，不回滚主操作。
		lpInterLog.Error(
			"查询互动计数失败",
			"plan_id", planID,
			"type", interactionType,
			"error", err,
		)
		newCount = 0
	}

	action := "取消"
	if active {
		action = "添加"
	}
	lpInterLog.Info(
		"互动操作完成",
		"user_id", userID,
		"plan_id", planID,
		"type", interactionType,
		"action", action,
		"new_count", newCount,
	)

	return &models.ToggleInteractionResponse{
		Active:   active,
		NewCount: newCount,
	}, nil
}

// GetInteractionCounts 在聚合前校验目标共享教案。
func (
	s *LessonPlanInteractionService,
) GetInteractionCounts(
	ctx context.Context,
	planID string,
	currentUserID string,
) (*models.InteractionCounts, error) {
	if err := s.loadVisibleSharedPlan(
		ctx,
		currentUserID,
		planID,
	); err != nil {
		return nil, err
	}

	return repository.GetInteractionCounts(
		ctx,
		planID,
		currentUserID,
	)
}

// ListMyFavorites 只返回当前用户仍有权访问的共享收藏。
func (
	s *LessonPlanInteractionService,
) ListMyFavorites(
	ctx context.Context,
	userID string,
	limit int,
	offset int,
) (*models.FavoriteListResponse, error) {
	access, err := resolveLessonPlanSharedAccessContext(
		ctx,
		userID,
		nil,
	)
	if err != nil {
		if errors.Is(
			err,
			errLPSharedAccessUnavailable,
		) {
			return &models.FavoriteListResponse{
				Items: []*models.FavoriteListItem{},
				Total: 0,
			}, nil
		}
		return nil, err
	}

	items, total, err :=
		repository.ListUserFavoritesWithSharedAccess(
			ctx,
			userID,
			limit,
			offset,
			access.VisibleAuthorIDs,
			access.CurrentEducationDomain,
		)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.FavoriteListItem{}
	}

	return &models.FavoriteListResponse{
		Items: items,
		Total: total,
	}, nil
}
